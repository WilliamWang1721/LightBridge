package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	codexchat "lightbridge/internal/translator/codex/openai/chat_completions"
	codexresp "lightbridge/internal/translator/codex/openai/responses"
	"lightbridge/internal/types"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var dataTag = []byte("data:")

type CodexAdapter struct {
	client *http.Client
}

type CodexConfig struct {
	BaseURL      string            `json:"base_url"`
	APIKey       string            `json:"api_key"`
	ExtraHeaders map[string]string `json:"extra_headers"`
}

func NewCodexAdapter(client *http.Client) *CodexAdapter {
	if client == nil {
		client = &http.Client{}
	}
	return &CodexAdapter{client: client}
}

func (a *CodexAdapter) Protocol() string {
	return types.ProtocolCodex
}

func (a *CodexAdapter) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	switch req.URL.Path {
	case "/v1/chat/completions":
		return a.handleChatCompletions(ctx, w, req, provider, route)
	case "/v1/responses":
		return a.handleResponses(ctx, w, req, provider, route)
	default:
		writeOpenAIError(w, http.StatusNotImplemented, "Endpoint not supported by codex provider", "not_supported", "501_not_supported")
		return http.StatusNotImplemented, "501_not_supported", nil
	}
}

func (a *CodexAdapter) handleChatCompletions(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}

	cfg := CodexConfig{}
	if strings.TrimSpace(provider.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(provider.Endpoint)
	}
	if baseURL == "" {
		writeOpenAIError(w, http.StatusBadGateway, "Codex provider missing endpoint", "provider_misconfigured", "provider_misconfigured")
		return http.StatusBadGateway, "provider_misconfigured", nil
	}
	targetURL := strings.TrimRight(baseURL, "/") + "/responses"

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

	// Codex upstream is streamed SSE even when the client requests non-streaming.
	codexReqBody := codexchat.ConvertOpenAIRequestToCodex(upstreamModel, body, true)

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(codexReqBody))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	upstreamReq.Header.Set("content-type", "application/json")
	upstreamReq.Header.Set("accept", "text/event-stream")
	upstreamReq.Header.Set("accept-encoding", "identity")
	if strings.TrimSpace(cfg.APIKey) != "" {
		upstreamReq.Header.Set("authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		upstreamReq.Header.Set(k, v)
	}

	resp, err := a.client.Do(upstreamReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "codex_upstream_error")
		return resp.StatusCode, "codex_upstream_error", nil
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
		sc.Buffer(make([]byte, 0, 64*1024), 52_428_800) // ~50MB

		var param any
		for sc.Scan() {
			line := bytes.Clone(sc.Bytes())
			chunks := codexchat.ConvertCodexResponseToOpenAI(ctx, upstreamModel, body, codexReqBody, line, &param)
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
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800) // ~50MB
	var completed []byte
	for sc.Scan() {
		line := bytes.Clone(sc.Bytes())
		if payload := codexSSEDataPayload(line); payload != nil {
			if gjson.GetBytes(payload, "type").String() == "response.completed" {
				completed = payload
				break
			}
		}
	}
	if err := sc.Err(); err != nil {
		return http.StatusBadGateway, "upstream_stream_failed", err
	}
	if len(completed) == 0 {
		return http.StatusGatewayTimeout, "codex_stream_incomplete", fmt.Errorf("codex stream closed before response.completed")
	}

	out := codexchat.ConvertCodexResponseToOpenAINonStream(ctx, upstreamModel, body, codexReqBody, completed, nil)
	if strings.TrimSpace(out) == "" {
		return http.StatusBadGateway, "codex_translate_failed", fmt.Errorf("failed to translate codex response")
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func (a *CodexAdapter) handleResponses(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}

	cfg := CodexConfig{}
	if strings.TrimSpace(provider.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(provider.Endpoint)
	}
	if baseURL == "" {
		writeOpenAIError(w, http.StatusBadGateway, "Codex provider missing endpoint", "provider_misconfigured", "provider_misconfigured")
		return http.StatusBadGateway, "provider_misconfigured", nil
	}
	targetURL := strings.TrimRight(baseURL, "/") + "/responses"

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

	// Ensure upstream sees the resolved upstream model.
	bodyWithModel, _ := sjson.SetBytes(body, "model", upstreamModel)
	codexReqBody := codexresp.ConvertOpenAIResponsesRequestToCodex(upstreamModel, bodyWithModel, true)

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(codexReqBody))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	upstreamReq.Header.Set("content-type", "application/json")
	upstreamReq.Header.Set("accept", "text/event-stream")
	upstreamReq.Header.Set("accept-encoding", "identity")
	if strings.TrimSpace(cfg.APIKey) != "" {
		upstreamReq.Header.Set("authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		upstreamReq.Header.Set(k, v)
	}

	resp, err := a.client.Do(upstreamReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "codex_upstream_error")
		return resp.StatusCode, "codex_upstream_error", nil
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
		sc.Buffer(make([]byte, 0, 64*1024), 52_428_800) // ~50MB

		var param any
		for sc.Scan() {
			line := bytes.Clone(sc.Bytes())
			outLines := codexresp.ConvertCodexResponseToOpenAIResponses(ctx, upstreamModel, bodyWithModel, codexReqBody, line, &param)
			for _, outLine := range outLines {
				_, _ = io.WriteString(w, outLine+"\n")
			}
			flusher.Flush()
		}
		if err := sc.Err(); err != nil {
			return http.StatusBadGateway, "upstream_stream_failed", err
		}
		return http.StatusOK, "", nil
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800) // ~50MB
	var completed []byte
	for sc.Scan() {
		line := bytes.Clone(sc.Bytes())
		if payload := codexSSEDataPayload(line); payload != nil {
			if gjson.GetBytes(payload, "type").String() == "response.completed" {
				completed = payload
				break
			}
		}
	}
	if err := sc.Err(); err != nil {
		return http.StatusBadGateway, "upstream_stream_failed", err
	}
	if len(completed) == 0 {
		return http.StatusGatewayTimeout, "codex_stream_incomplete", fmt.Errorf("codex stream closed before response.completed")
	}
	out := codexresp.ConvertCodexResponseToOpenAIResponsesNonStream(ctx, upstreamModel, bodyWithModel, codexReqBody, completed, nil)
	if strings.TrimSpace(out) == "" {
		return http.StatusBadGateway, "codex_translate_failed", fmt.Errorf("failed to translate codex response")
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func codexSSEDataPayload(line []byte) []byte {
	trimmed := bytes.TrimSpace(line)
	if !bytes.HasPrefix(trimmed, dataTag) {
		return nil
	}
	payload := bytes.TrimSpace(trimmed[len(dataTag):])
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return nil
	}
	return payload
}

func isCodexResponseCompletedLine(line []byte) bool {
	payload := codexSSEDataPayload(line)
	if payload == nil {
		return false
	}
	return gjson.GetBytes(payload, "type").String() == "response.completed"
}
