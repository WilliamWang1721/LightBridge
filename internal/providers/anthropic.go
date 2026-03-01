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
	"time"

	claudechat "lightbridge/internal/translator/claude/openai/chat_completions"
	clauderesp "lightbridge/internal/translator/claude/openai/responses"
	"lightbridge/internal/types"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const anthropicVersionHeader = "2023-06-01"

type AnthropicAdapter struct {
	client *http.Client
}

type AnthropicConfig struct {
	BaseURL      string            `json:"base_url"`
	APIKey       string            `json:"api_key"`
	ExtraHeaders map[string]string `json:"extra_headers"`
}

func NewAnthropicAdapter(client *http.Client) *AnthropicAdapter {
	if client == nil {
		client = &http.Client{}
	}
	return &AnthropicAdapter{client: client}
}

func (a *AnthropicAdapter) Protocol() string {
	return types.ProtocolAnthropic
}

func (a *AnthropicAdapter) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	switch req.URL.Path {
	case "/v1/messages":
		return a.handleNativeMessages(ctx, w, req, provider, route)
	case "/v1/chat/completions":
		return a.handleChatCompletions(ctx, w, req, provider, route)
	case "/v1/responses":
		return a.handleResponses(ctx, w, req, provider, route)
	default:
		writeOpenAIError(w, http.StatusNotImplemented, "Endpoint not supported by anthropic provider", "not_supported", "501_not_supported")
		return http.StatusNotImplemented, "501_not_supported", nil
	}
}

func (a *AnthropicAdapter) handleNativeMessages(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}
	cfg := a.parseConfig(provider)
	if cfg.APIKey == "" {
		writeOpenAIError(w, http.StatusBadGateway, "Anthropic API key missing", "provider_misconfigured", "provider_misconfigured")
		return http.StatusBadGateway, "provider_misconfigured", nil
	}

	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(io.LimitReader(req.Body, 20<<20))
		_ = req.Body.Close()
	}
	body = rewriteModel(body, route, nil)
	upstreamModel := strings.TrimSpace(route.UpstreamModel)
	if upstreamModel == "" && gjson.ValidBytes(body) {
		upstreamModel = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	}
	if upstreamModel != "" && isOfficialAnthropicHost(cfg.BaseURL) && isOpenAIStyleModel(upstreamModel) {
		writeOpenAIError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf("model %q is not supported by Anthropic Messages endpoint; use OpenAI-compatible ingress for GPT/o-series models", upstreamModel),
			"invalid_request_error",
			"incompatible_model",
		)
		return http.StatusBadRequest, "incompatible_model", nil
	}

	targetURL := strings.TrimRight(cfg.BaseURL, "/") + "/v1/messages"
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	copyHeaders(upReq.Header, req.Header)
	upReq.Header.Set("x-api-key", cfg.APIKey)
	upReq.Header.Set("anthropic-version", anthropicVersionHeader)
	upReq.Header.Del("authorization")
	upReq.Header.Del("accept-encoding")
	if strings.TrimSpace(upReq.Header.Get("content-type")) == "" {
		upReq.Header.Set("content-type", "application/json")
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return http.StatusBadGateway, "upstream_read_failed", err
	}
	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(bytes.NewReader(respBody))
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "anthropic_upstream_error")
		return resp.StatusCode, "anthropic_upstream_error", nil
	}

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(respBody); err != nil {
		return resp.StatusCode, "upstream_stream_failed", err
	}
	return resp.StatusCode, "", nil
}

func (a *AnthropicAdapter) handleChatCompletions(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}

	cfg := a.parseConfig(provider)
	if cfg.APIKey == "" {
		writeOpenAIError(w, http.StatusBadGateway, "Anthropic API key missing", "provider_misconfigured", "provider_misconfigured")
		return http.StatusBadGateway, "provider_misconfigured", nil
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
		upstreamModel = gjson.GetBytes(body, "model").String()
	}

	// Translate OpenAI Chat Completions → Claude Messages
	claudeReqBody := claudechat.ConvertOpenAIRequestToClaude(upstreamModel, body, clientWantsStream)

	resp, err := a.doUpstream(ctx, cfg, claudeReqBody, clientWantsStream)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "anthropic_upstream_error")
		return resp.StatusCode, "anthropic_upstream_error", nil
	}

	if clientWantsStream {
		return a.streamClaudeToOpenAIChatCompletions(w, resp.Body, upstreamModel, body, claudeReqBody)
	}

	// Non-stream: Claude returns plain JSON (not SSE)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusBadGateway, "upstream_read_failed", err
	}

	// Check if response is SSE or plain JSON
	var out string
	if bytes.HasPrefix(bytes.TrimSpace(respBody), []byte("event:")) || bytes.HasPrefix(bytes.TrimSpace(respBody), []byte("data:")) {
		out = claudechat.ConvertClaudeResponseToOpenAINonStream(ctx, upstreamModel, body, claudeReqBody, respBody, nil)
	} else {
		// Plain JSON response from Claude — convert directly
		out = convertClaudeJSONToOpenAIChatCompletion(respBody, route.RequestedModel)
	}
	if strings.TrimSpace(out) == "" {
		return http.StatusBadGateway, "anthropic_translate_failed", fmt.Errorf("failed to translate claude response")
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func (a *AnthropicAdapter) handleResponses(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return http.StatusMethodNotAllowed, "method_not_allowed", nil
	}

	cfg := a.parseConfig(provider)
	if cfg.APIKey == "" {
		writeOpenAIError(w, http.StatusBadGateway, "Anthropic API key missing", "provider_misconfigured", "provider_misconfigured")
		return http.StatusBadGateway, "provider_misconfigured", nil
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
		upstreamModel = gjson.GetBytes(body, "model").String()
	}

	// Translate OpenAI Responses → Claude Messages (always stream upstream)
	claudeReqBody := clauderesp.ConvertOpenAIResponsesRequestToClaude(upstreamModel, body, true)

	resp, err := a.doUpstream(ctx, cfg, claudeReqBody, true)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "anthropic_upstream_error")
		return resp.StatusCode, "anthropic_upstream_error", nil
	}

	if clientWantsStream {
		return a.streamClaudeToOpenAIResponses(w, resp.Body, upstreamModel, body, claudeReqBody)
	}

	// Non-stream: collect all SSE, translate
	allLines := collectSSELines(resp.Body)
	out := clauderesp.ConvertClaudeResponseToOpenAIResponsesNonStream(ctx, upstreamModel, body, claudeReqBody, allLines, nil)
	if strings.TrimSpace(out) == "" {
		// Fallback: try reading as plain JSON
		return http.StatusBadGateway, "anthropic_translate_failed", fmt.Errorf("failed to translate claude response")
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, out)
	return http.StatusOK, "", nil
}

func (a *AnthropicAdapter) streamClaudeToOpenAIChatCompletions(w http.ResponseWriter, body io.Reader, model string, origReq, claudeReq []byte) (int, string, error) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return http.StatusInternalServerError, "stream_not_supported", fmt.Errorf("flushing not supported")
	}

	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)

	var param any
	for sc.Scan() {
		line := bytes.Clone(sc.Bytes())
		chunks := claudechat.ConvertClaudeResponseToOpenAI(context.Background(), model, origReq, claudeReq, line, &param)
		for _, chunk := range chunks {
			if strings.TrimSpace(chunk) == "" {
				continue
			}
			_, _ = io.WriteString(w, "data: "+chunk+"\n\n")
			flusher.Flush()
		}
	}
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
	if err := sc.Err(); err != nil {
		return http.StatusBadGateway, "upstream_stream_failed", err
	}
	return http.StatusOK, "", nil
}

func (a *AnthropicAdapter) streamClaudeToOpenAIResponses(w http.ResponseWriter, body io.Reader, model string, origReq, claudeReq []byte) (int, string, error) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.Header().Set("x-accel-buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return http.StatusInternalServerError, "stream_not_supported", fmt.Errorf("flushing not supported")
	}

	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)

	var param any
	for sc.Scan() {
		line := bytes.Clone(sc.Bytes())
		outLines := clauderesp.ConvertClaudeResponseToOpenAIResponses(context.Background(), model, origReq, claudeReq, line, &param)
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

func (a *AnthropicAdapter) parseConfig(provider types.Provider) AnthropicConfig {
	cfg := AnthropicConfig{}
	if provider.ConfigJSON != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = strings.TrimSpace(provider.Endpoint)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	}
	return cfg
}

func (a *AnthropicAdapter) doUpstream(ctx context.Context, cfg AnthropicConfig, payload []byte, stream bool) (*http.Response, error) {
	url := strings.TrimRight(cfg.BaseURL, "/") + "/v1/messages"
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	upReq.Header.Set("content-type", "application/json")
	upReq.Header.Set("x-api-key", cfg.APIKey)
	upReq.Header.Set("anthropic-version", anthropicVersionHeader)
	if stream {
		upReq.Header.Set("accept", "text/event-stream")
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) != "" {
			upReq.Header.Set(k, v)
		}
	}
	return a.client.Do(upReq)
}

func isOfficialAnthropicHost(base string) bool {
	raw := strings.TrimSpace(base)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	return host == "api.anthropic.com"
}

func isOpenAIStyleModel(model string) bool {
	lower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(lower, "gpt-"),
		strings.HasPrefix(lower, "o1-"),
		strings.HasPrefix(lower, "o3-"),
		strings.HasPrefix(lower, "o4-"),
		strings.HasPrefix(lower, "chatgpt-"):
		return true
	default:
		return false
	}
}

func collectSSELines(body io.Reader) []byte {
	var buf bytes.Buffer
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)
	for sc.Scan() {
		buf.Write(sc.Bytes())
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// convertClaudeJSONToOpenAIChatCompletion converts a plain Claude Messages API
// JSON response to OpenAI Chat Completions format.
func convertClaudeJSONToOpenAIChatCompletion(body []byte, requestedModel string) string {
	root := gjson.ParseBytes(body)

	var contentParts []string
	var toolCalls []string
	if output := root.Get("content"); output.IsArray() {
		for _, block := range output.Array() {
			switch block.Get("type").String() {
			case "text":
				contentParts = append(contentParts, block.Get("text").String())
			case "tool_use":
				tc := `{"id":"","type":"function","function":{"name":"","arguments":""}}`
				tc, _ = sjson.Set(tc, "id", block.Get("id").String())
				tc, _ = sjson.Set(tc, "function.name", block.Get("name").String())
				if input := block.Get("input"); input.Exists() {
					tc, _ = sjson.Set(tc, "function.arguments", input.Raw)
				}
				toolCalls = append(toolCalls, tc)
			}
		}
	}

	model := requestedModel
	if model == "" {
		model = root.Get("model").String()
	}
	if model == "" {
		model = "claude"
	}

	finish := "stop"
	if sr := root.Get("stop_reason").String(); sr != "" {
		switch sr {
		case "end_turn":
			finish = "stop"
		case "tool_use":
			finish = "tool_calls"
		case "max_tokens":
			finish = "length"
		default:
			finish = "stop"
		}
	}

	out := `{"id":"","object":"chat.completion","created":0,"model":"","choices":[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"stop"}]}`
	out, _ = sjson.Set(out, "id", root.Get("id").String())
	out, _ = sjson.Set(out, "created", time.Now().Unix())
	out, _ = sjson.Set(out, "model", model)
	out, _ = sjson.Set(out, "choices.0.message.content", strings.Join(contentParts, ""))
	out, _ = sjson.Set(out, "choices.0.finish_reason", finish)

	if len(toolCalls) > 0 {
		out, _ = sjson.SetRaw(out, "choices.0.message.tool_calls", "["+strings.Join(toolCalls, ",")+"]")
		out, _ = sjson.Set(out, "choices.0.finish_reason", "tool_calls")
	}

	if usage := root.Get("usage"); usage.Exists() {
		input := usage.Get("input_tokens").Int()
		output := usage.Get("output_tokens").Int()
		out, _ = sjson.Set(out, "usage.prompt_tokens", input)
		out, _ = sjson.Set(out, "usage.completion_tokens", output)
		out, _ = sjson.Set(out, "usage.total_tokens", input+output)
	}

	return out
}
