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

	geminichat "lightbridge/internal/translator/gemini/openai/chat_completions"
	geminiresp "lightbridge/internal/translator/gemini/openai/responses"
	"lightbridge/internal/types"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type GeminiAdapter struct {
	client *http.Client
}

type GeminiConfig struct {
	BaseURL      string            `json:"base_url"`
	APIKey       string            `json:"api_key"`
	ExtraHeaders map[string]string `json:"extra_headers"`
}

func NewGeminiAdapter(client *http.Client) *GeminiAdapter {
	if client == nil {
		client = &http.Client{}
	}
	return &GeminiAdapter{client: client}
}

func (a *GeminiAdapter) Protocol() string {
	return types.ProtocolGemini
}

func (a *GeminiAdapter) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	// Native Gemini passthrough.
	if strings.HasPrefix(req.URL.Path, "/v1beta/") {
		return a.handleNative(ctx, w, req, provider, route)
	}

	switch req.URL.Path {
	case "/v1/chat/completions":
		return a.handleChatCompletions(ctx, w, req, provider, route)
	case "/v1/responses":
		return a.handleResponses(ctx, w, req, provider, route)
	case "/v1/models":
		return a.handleModels(ctx, w, req, provider)
	default:
		writeOpenAIError(w, http.StatusNotImplemented, "Endpoint not supported by gemini provider", "not_supported", "501_not_supported")
		return http.StatusNotImplemented, "501_not_supported", nil
	}
}

func (a *GeminiAdapter) handleModels(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider) (int, string, error) {
	cfg := a.parseConfig(provider)
	targetURL, err := joinPathURL(cfg.BaseURL, "/v1beta/models")
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	a.applyGeminiAuth(targetURL, req.URL.Query(), cfg)
	upReq, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	upReq.Header.Set("accept", "application/json")
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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(bytes.NewReader(body))
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "gemini_upstream_error")
		return resp.StatusCode, "gemini_upstream_error", nil
	}

	out := `{"object":"list","data":[]}`
	if arr := gjson.GetBytes(body, "models"); arr.Exists() && arr.IsArray() {
		arr.ForEach(func(_, item gjson.Result) bool {
			name := strings.TrimSpace(item.Get("name").String()) // models/gemini-2.5-pro
			if name == "" {
				return true
			}
			id := name
			if i := strings.LastIndex(name, "/"); i >= 0 && i < len(name)-1 {
				id = name[i+1:]
			}
			entry := `{"id":"","object":"model","created":0,"owned_by":"gemini"}`
			entry, _ = sjson.Set(entry, "id", id)
			entry, _ = sjson.Set(entry, "created", item.Get("createTime").Int())
			out, _ = sjson.SetRaw(out, "data.-1", entry)
			return true
		})
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func (a *GeminiAdapter) handleNative(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	cfg := a.parseConfig(provider)
	path := req.URL.Path
	if route != nil {
		path = rewriteGeminiModelPath(path, route.UpstreamModel)
	}
	targetURL, err := joinPathURL(cfg.BaseURL, path)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	a.applyGeminiAuth(targetURL, req.URL.Query(), cfg)

	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	upReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL.String(), bytes.NewReader(body))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	copyHeaders(upReq.Header, req.Header)
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) != "" {
			upReq.Header.Set(k, v)
		}
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		upReq.Header.Set("x-goog-api-key", strings.TrimSpace(cfg.APIKey))
	}
	upReq.Header.Del("Authorization")
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

func (a *GeminiAdapter) handleChatCompletions(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}
	cfg := a.parseConfig(provider)
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
	stream := gjson.GetBytes(body, "stream").Type == gjson.True
	upstreamModel := strings.TrimSpace(route.UpstreamModel)
	if upstreamModel == "" {
		upstreamModel = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	}
	if upstreamModel == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return http.StatusBadRequest, "missing_model", nil
	}

	gemBody := geminichat.ConvertOpenAIRequestToGemini(upstreamModel, body, stream)
	methodName := "generateContent"
	if stream {
		methodName = "streamGenerateContent"
	}
	targetPath := fmt.Sprintf("/v1beta/models/%s:%s", url.PathEscape(upstreamModel), methodName)
	targetURL, err := joinPathURL(cfg.BaseURL, targetPath)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	a.applyGeminiAuth(targetURL, req.URL.Query(), cfg)
	if stream {
		q := targetURL.Query()
		if strings.TrimSpace(q.Get("alt")) == "" {
			q.Set("alt", "sse")
		}
		targetURL.RawQuery = q.Encode()
	}
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL.String(), bytes.NewReader(gemBody))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	upReq.Header.Set("content-type", "application/json")
	upReq.Header.Set("accept", "application/json")
	if stream {
		upReq.Header.Set("accept", "text/event-stream")
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) != "" {
			upReq.Header.Set(k, v)
		}
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		upReq.Header.Set("x-goog-api-key", strings.TrimSpace(cfg.APIKey))
	}
	upReq.Header.Del("Authorization")
	upReq.Header.Del("Accept-Encoding")

	resp, err := a.client.Do(upReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "gemini_upstream_error")
		return resp.StatusCode, "gemini_upstream_error", nil
	}

	if stream {
		w.Header().Set("content-type", "text/event-stream")
		w.Header().Set("cache-control", "no-cache")
		w.Header().Set("connection", "keep-alive")
		w.Header().Set("x-accel-buffering", "no")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return http.StatusInternalServerError, "stream_not_supported", fmt.Errorf("flushing not supported")
		}
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)
		var param any
		for sc.Scan() {
			line := bytes.Clone(sc.Bytes())
			chunks := geminichat.ConvertGeminiResponseToOpenAI(ctx, upstreamModel, body, gemBody, line, &param)
			for _, chunk := range chunks {
				if strings.TrimSpace(chunk) == "" {
					continue
				}
				_, _ = io.WriteString(w, "data: "+chunk+"\n\n")
				flusher.Flush()
			}
		}
		if err := sc.Err(); err != nil {
			return http.StatusBadGateway, "upstream_stream_failed", err
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		flusher.Flush()
		return http.StatusOK, "", nil
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return http.StatusBadGateway, "upstream_read_failed", err
	}
	out := geminichat.ConvertGeminiResponseToOpenAINonStream(ctx, upstreamModel, body, gemBody, raw, nil)
	if strings.TrimSpace(out) == "" {
		return http.StatusBadGateway, "gemini_translate_failed", fmt.Errorf("failed to translate gemini response")
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func (a *GeminiAdapter) handleResponses(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}
	cfg := a.parseConfig(provider)
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
	stream := gjson.GetBytes(body, "stream").Type == gjson.True
	upstreamModel := strings.TrimSpace(route.UpstreamModel)
	if upstreamModel == "" {
		upstreamModel = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	}
	if upstreamModel == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return http.StatusBadRequest, "missing_model", nil
	}
	bodyWithModel, _ := sjson.SetBytes(body, "model", upstreamModel)
	gemBody := geminiresp.ConvertOpenAIResponsesRequestToGemini(upstreamModel, bodyWithModel, stream)

	methodName := "generateContent"
	if stream {
		methodName = "streamGenerateContent"
	}
	targetPath := fmt.Sprintf("/v1beta/models/%s:%s", url.PathEscape(upstreamModel), methodName)
	targetURL, err := joinPathURL(cfg.BaseURL, targetPath)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	a.applyGeminiAuth(targetURL, req.URL.Query(), cfg)
	if stream {
		q := targetURL.Query()
		if strings.TrimSpace(q.Get("alt")) == "" {
			q.Set("alt", "sse")
		}
		targetURL.RawQuery = q.Encode()
	}
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL.String(), bytes.NewReader(gemBody))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	upReq.Header.Set("content-type", "application/json")
	upReq.Header.Set("accept", "application/json")
	if stream {
		upReq.Header.Set("accept", "text/event-stream")
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) != "" {
			upReq.Header.Set(k, v)
		}
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		upReq.Header.Set("x-goog-api-key", strings.TrimSpace(cfg.APIKey))
	}
	upReq.Header.Del("Authorization")
	upReq.Header.Del("Accept-Encoding")

	resp, err := a.client.Do(upReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "gemini_upstream_error")
		return resp.StatusCode, "gemini_upstream_error", nil
	}

	if stream {
		w.Header().Set("content-type", "text/event-stream")
		w.Header().Set("cache-control", "no-cache")
		w.Header().Set("connection", "keep-alive")
		w.Header().Set("x-accel-buffering", "no")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return http.StatusInternalServerError, "stream_not_supported", fmt.Errorf("flushing not supported")
		}

		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)
		var param any
		for sc.Scan() {
			line := bytes.Clone(sc.Bytes())
			outLines := geminiresp.ConvertGeminiResponseToOpenAIResponses(ctx, upstreamModel, bodyWithModel, gemBody, line, &param)
			for _, outLine := range outLines {
				if strings.TrimSpace(outLine) == "" {
					continue
				}
				_, _ = io.WriteString(w, outLine+"\n\n")
			}
			flusher.Flush()
		}
		if err := sc.Err(); err != nil {
			return http.StatusBadGateway, "upstream_stream_failed", err
		}
		return http.StatusOK, "", nil
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return http.StatusBadGateway, "upstream_read_failed", err
	}
	out := geminiresp.ConvertGeminiResponseToOpenAIResponsesNonStream(ctx, upstreamModel, bodyWithModel, gemBody, raw, nil)
	if strings.TrimSpace(out) == "" {
		return http.StatusBadGateway, "gemini_translate_failed", fmt.Errorf("failed to translate gemini responses output")
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func (a *GeminiAdapter) parseConfig(provider types.Provider) GeminiConfig {
	cfg := GeminiConfig{}
	if strings.TrimSpace(provider.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = strings.TrimSpace(provider.Endpoint)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://generativelanguage.googleapis.com"
	}
	return cfg
}

func (a *GeminiAdapter) applyGeminiAuth(targetURL *url.URL, incomingQuery url.Values, cfg GeminiConfig) {
	q := targetURL.Query()
	if incomingQuery != nil {
		for k, vals := range incomingQuery {
			for _, v := range vals {
				q.Add(k, v)
			}
		}
	}
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		if strings.TrimSpace(q.Get("key")) == "" {
			q.Set("key", key)
		}
	}
	targetURL.RawQuery = q.Encode()
}

func rewriteGeminiModelPath(path, model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return path
	}
	lower := strings.ToLower(path)
	idx := strings.Index(lower, "/models/")
	if idx < 0 {
		return path
	}
	start := idx + len("/models/")
	rest := path[start:]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		slash := strings.Index(rest, "/")
		if slash < 0 {
			return path[:start] + model
		}
		return path[:start] + model + rest[slash:]
	}
	return path[:start] + model + rest[colon:]
}
