package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	codexchat "lightbridge/internal/translator/codex/openai/chat_completions"
	"lightbridge/internal/types"

	"github.com/tidwall/gjson"
)

type OpenAIResponsesAdapter struct {
	client *http.Client
}

type OpenAIResponsesConfig struct {
	BaseURL      string            `json:"base_url"`
	BaseOrigin   string            `json:"base_origin"`
	APIKey       string            `json:"api_key"`
	ExtraHeaders map[string]string `json:"extra_headers"`
}

func NewOpenAIResponsesAdapter(client *http.Client) *OpenAIResponsesAdapter {
	if client == nil {
		client = &http.Client{}
	}
	return &OpenAIResponsesAdapter{client: client}
}

func (a *OpenAIResponsesAdapter) Protocol() string {
	return types.ProtocolOpenAIResponses
}

func (a *OpenAIResponsesAdapter) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	switch req.URL.Path {
	case "/v1/chat/completions":
		return a.handleChatCompletions(ctx, w, req, provider, route)
	case "/v1/responses", "/v1/models":
		return a.handleForward(ctx, w, req, provider, route)
	default:
		writeOpenAIError(w, http.StatusNotImplemented, "Endpoint not supported by openai_responses provider", "not_supported", "501_not_supported")
		return http.StatusNotImplemented, "501_not_supported", nil
	}
}

func (a *OpenAIResponsesAdapter) handleForward(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	cfg := OpenAIResponsesConfig{}
	if strings.TrimSpace(provider.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}
	baseURL := strings.TrimSpace(cfg.BaseOrigin)
	if baseURL == "" {
		baseURL = strings.TrimSpace(cfg.BaseURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(provider.Endpoint)
	}
	if baseURL == "" {
		return http.StatusBadGateway, "provider_misconfigured", fmt.Errorf("provider %s missing endpoint", provider.ID)
	}

	targetURL, err := joinPathURL(baseURL, req.URL.Path)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	targetURL.RawQuery = req.URL.RawQuery

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	bodyBytes = rewriteModel(bodyBytes, route, nil)

	upReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	copyHeaders(upReq.Header, req.Header)
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		upReq.Header.Set("authorization", "Bearer "+key)
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		upReq.Header.Set(k, v)
	}
	upReq.Header.Del("Accept-Encoding")

	resp, err := a.client.Do(upReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return resp.StatusCode, "upstream_stream_failed", err
	}
	return resp.StatusCode, "", nil
}

func (a *OpenAIResponsesAdapter) handleChatCompletions(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}

	cfg := OpenAIResponsesConfig{}
	if strings.TrimSpace(provider.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}
	baseURL := strings.TrimSpace(cfg.BaseOrigin)
	if baseURL == "" {
		baseURL = strings.TrimSpace(cfg.BaseURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(provider.Endpoint)
	}
	if baseURL == "" {
		writeOpenAIError(w, http.StatusBadGateway, "OpenAI Responses provider missing endpoint", "provider_misconfigured", "provider_misconfigured")
		return http.StatusBadGateway, "provider_misconfigured", nil
	}
	targetURL, err := joinPathURL(baseURL, "/v1/responses")
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, 20<<20))
	_ = req.Body.Close()
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid request body", "invalid_request_error", "invalid_body")
		return http.StatusBadRequest, "invalid_body", nil
	}
	if !gjson.ValidBytes(body) {
		writeOpenAIError(w, http.StatusBadRequest, "Body must be valid JSON", "invalid_request_error", "invalid_json")
		return http.StatusBadRequest, "invalid_json", nil
	}

	clientWantsStream := gjson.GetBytes(body, "stream").Type == gjson.True
	upstreamModel := strings.TrimSpace(route.UpstreamModel)
	if upstreamModel == "" {
		upstreamModel = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	}
	if upstreamModel == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return http.StatusBadRequest, "missing_model", nil
	}

	// Reuse chat->responses translator; responses upstream is transformed back to chat format.
	upReqBody := codexchat.ConvertOpenAIRequestToCodex(upstreamModel, body, true)
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL.String(), bytes.NewReader(upReqBody))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	upReq.Header.Set("content-type", "application/json")
	upReq.Header.Set("accept", "text/event-stream")
	upReq.Header.Set("accept-encoding", "identity")
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		upReq.Header.Set("authorization", "Bearer "+key)
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) != "" {
			upReq.Header.Set(k, v)
		}
	}

	resp, err := a.client.Do(upReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "openai_responses_upstream_error")
		return resp.StatusCode, "openai_responses_upstream_error", nil
	}

	if clientWantsStream {
		w.Header().Set("content-type", "text/event-stream")
		w.Header().Set("cache-control", "no-cache")
		w.Header().Set("connection", "keep-alive")
		w.Header().Set("x-accel-buffering", "no")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeOpenAIError(w, http.StatusInternalServerError, "streaming not supported by server", "server_error", "stream_unsupported")
			return http.StatusInternalServerError, "stream_unsupported", nil
		}

		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)
		var param any
		for sc.Scan() {
			line := bytes.Clone(sc.Bytes())
			chunks := codexchat.ConvertCodexResponseToOpenAI(ctx, upstreamModel, body, upReqBody, line, &param)
			for _, chunk := range chunks {
				if strings.TrimSpace(chunk) == "" {
					continue
				}
				_, _ = io.WriteString(w, "data: "+chunk+"\n\n")
				flusher.Flush()
			}
			if isCodexResponseCompletedLine(line) {
				_, _ = io.WriteString(w, "data: [DONE]\n\n")
				flusher.Flush()
				return http.StatusOK, "", nil
			}
		}
		if err := sc.Err(); err != nil {
			return http.StatusBadGateway, "upstream_stream_failed", err
		}
		return http.StatusOK, "", nil
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)
	var completed []byte
	for sc.Scan() {
		line := bytes.Clone(sc.Bytes())
		if payload := codexSSEDataPayload(line); payload != nil && gjson.GetBytes(payload, "type").String() == "response.completed" {
			completed = payload
			break
		}
	}
	if err := sc.Err(); err != nil {
		return http.StatusBadGateway, "upstream_stream_failed", err
	}
	if len(completed) == 0 {
		return http.StatusGatewayTimeout, "openai_responses_stream_incomplete", fmt.Errorf("responses stream closed before response.completed")
	}
	out := codexchat.ConvertCodexResponseToOpenAINonStream(ctx, upstreamModel, body, upReqBody, completed, nil)
	if strings.TrimSpace(out) == "" {
		return http.StatusBadGateway, "openai_responses_translate_failed", fmt.Errorf("failed to translate responses output")
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func joinPathURL(baseURL, reqPath string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		return nil, fmt.Errorf("base url missing scheme: %s", baseURL)
	}
	p := strings.TrimSpace(reqPath)
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	basePath := strings.TrimRight(u.Path, "/")
	if basePath == "" || basePath == "/" {
		u.Path = p
	} else if p == basePath || strings.HasPrefix(p, basePath+"/") {
		u.Path = p
	} else {
		u.Path = basePath + p
	}
	return u, nil
}
