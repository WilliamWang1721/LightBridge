package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lightbridge/internal/types"
)

const anthropicVersionHeader = "2023-06-01"

type AnthropicAdapter struct {
	client *http.Client
}

type AnthropicConfig struct {
	BaseURL      string   `json:"base_url"`
	APIKey       string   `json:"api_key"`
	DefaultModel []string `json:"default_models"`
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

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature *float64        `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicMessageRequest struct {
	Model       string                  `json:"model"`
	MaxTokens   int                     `json:"max_tokens"`
	Messages    []anthropicMessageInput `json:"messages"`
	System      string                  `json:"system,omitempty"`
	Stream      bool                    `json:"stream"`
	Temperature *float64                `json:"temperature,omitempty"`
}

type anthropicMessageInput struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicMessageResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *AnthropicAdapter) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	if req.Method != http.MethodPost || req.URL.Path != "/v1/chat/completions" {
		writeOpenAIError(w, http.StatusNotImplemented, "Endpoint not supported by anthropic provider", "not_supported", "501_not_supported")
		return http.StatusNotImplemented, "501_not_supported", nil
	}

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
	if cfg.APIKey == "" {
		writeOpenAIError(w, http.StatusBadGateway, "Anthropic API key missing", "provider_misconfigured", "provider_misconfigured")
		return http.StatusBadGateway, "provider_misconfigured", nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid request body", "invalid_request_error", "invalid_body")
		return http.StatusBadRequest, "invalid_body", nil
	}
	_ = req.Body.Close()

	var openReq openAIChatRequest
	if err := json.Unmarshal(body, &openReq); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Body must be valid JSON", "invalid_request_error", "invalid_json")
		return http.StatusBadRequest, "invalid_json", nil
	}
	anthReq := convertOpenAIToAnthropic(openReq, route)
	payload, _ := json.Marshal(anthReq)

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	upstreamReq.Header.Set("content-type", "application/json")
	upstreamReq.Header.Set("x-api-key", cfg.APIKey)
	upstreamReq.Header.Set("anthropic-version", anthropicVersionHeader)
	if openReq.Stream {
		upstreamReq.Header.Set("accept", "text/event-stream")
	}

	resp, err := a.client.Do(upstreamReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := parseUpstreamError(resp.Body)
		writeOpenAIError(w, resp.StatusCode, msg, "upstream_error", "anthropic_upstream_error")
		return resp.StatusCode, "anthropic_upstream_error", nil
	}

	if openReq.Stream {
		status, code, err := streamAnthropicToOpenAI(w, resp.Body, route.RequestedModel)
		return status, code, err
	}

	var anthResp anthropicMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return http.StatusBadGateway, "upstream_decode_failed", err
	}
	openResp := mapResponseToOpenAI(anthResp, route.RequestedModel)
	encoded, _ := json.Marshal(openResp)
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
	return http.StatusOK, "", nil
}

func convertOpenAIToAnthropic(in openAIChatRequest, route *types.ResolvedRoute) anthropicMessageRequest {
	msgs := make([]anthropicMessageInput, 0, len(in.Messages))
	var systemParts []string
	for _, m := range in.Messages {
		text := stringifyContent(m.Content)
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if text == "" {
			continue
		}
		if role == "system" {
			systemParts = append(systemParts, text)
			continue
		}
		if role == "assistant" || role == "user" {
			msgs = append(msgs, anthropicMessageInput{
				Role: role,
				Content: []anthropicContentBlock{{
					Type: "text",
					Text: text,
				}},
			})
		}
	}
	if len(msgs) == 0 {
		msgs = []anthropicMessageInput{{
			Role: "user",
			Content: []anthropicContentBlock{{
				Type: "text",
				Text: "",
			}},
		}}
	}
	maxTokens := in.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	model := in.Model
	if route != nil && route.UpstreamModel != "" {
		model = route.UpstreamModel
	}
	return anthropicMessageRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Messages:    msgs,
		System:      strings.Join(systemParts, "\n"),
		Stream:      in.Stream,
		Temperature: in.Temperature,
	}
}

func stringifyContent(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if txt, ok := m["text"].(string); ok {
				parts = append(parts, txt)
			}
		}
		return strings.Join(parts, "")
	case map[string]any:
		if txt, ok := t["text"].(string); ok {
			return txt
		}
	}
	return ""
}

func mapResponseToOpenAI(in anthropicMessageResponse, requestedModel string) map[string]any {
	contentParts := make([]string, 0, len(in.Content))
	for _, block := range in.Content {
		if block.Type == "text" {
			contentParts = append(contentParts, block.Text)
		}
	}
	if requestedModel == "" {
		requestedModel = in.Model
	}
	if requestedModel == "" {
		requestedModel = "claude"
	}
	finish := in.StopReason
	if finish == "" {
		finish = "stop"
	}
	return map[string]any{
		"id":      in.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   requestedModel,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": strings.Join(contentParts, ""),
				},
				"finish_reason": finish,
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     in.Usage.InputTokens,
			"completion_tokens": in.Usage.OutputTokens,
			"total_tokens":      in.Usage.InputTokens + in.Usage.OutputTokens,
		},
	}
}

func streamAnthropicToOpenAI(w http.ResponseWriter, body io.Reader, requestedModel string) (int, string, error) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return http.StatusInternalServerError, "stream_not_supported", errors.New("response writer does not support flushing")
	}

	scanner := bufio.NewScanner(body)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	currentEvent := ""
	var dataLines []string
	chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	if requestedModel == "" {
		requestedModel = "claude"
	}
	sentRole := false
	sentStop := false

	emit := func(payload map[string]any) error {
		bytesPayload, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", string(bytesPayload)); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	processEvent := func(event string, data string) error {
		if strings.TrimSpace(data) == "" {
			return nil
		}
		if strings.TrimSpace(data) == "[DONE]" {
			return nil
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(data), &obj); err != nil {
			return nil
		}
		switch event {
		case "message_start", "content_block_start":
			if !sentRole {
				err := emit(openAIChunk(chunkID, requestedModel, map[string]any{"role": "assistant"}, nil))
				if err != nil {
					return err
				}
				sentRole = true
			}
		case "content_block_delta":
			deltaText := ""
			if delta, ok := obj["delta"].(map[string]any); ok {
				if text, ok := delta["text"].(string); ok {
					deltaText = text
				}
			}
			if deltaText != "" {
				if !sentRole {
					err := emit(openAIChunk(chunkID, requestedModel, map[string]any{"role": "assistant"}, nil))
					if err != nil {
						return err
					}
					sentRole = true
				}
				err := emit(openAIChunk(chunkID, requestedModel, map[string]any{"content": deltaText}, nil))
				if err != nil {
					return err
				}
			}
		case "message_delta":
			if delta, ok := obj["delta"].(map[string]any); ok {
				if stop, ok := delta["stop_reason"].(string); ok && stop != "" {
					stopReason := stop
					err := emit(openAIChunk(chunkID, requestedModel, map[string]any{}, &stopReason))
					if err != nil {
						return err
					}
					sentStop = true
				}
			}
		case "message_stop":
			if !sentStop {
				stopReason := "stop"
				err := emit(openAIChunk(chunkID, requestedModel, map[string]any{}, &stopReason))
				if err != nil {
					return err
				}
				sentStop = true
			}
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			continue
		}
		if line == "" {
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				if err := processEvent(currentEvent, data); err != nil {
					return http.StatusBadGateway, "stream_convert_failed", err
				}
			}
			currentEvent = ""
			dataLines = dataLines[:0]
		}
	}
	if err := scanner.Err(); err != nil {
		return http.StatusBadGateway, "stream_read_failed", err
	}
	if !sentStop {
		stopReason := "stop"
		if err := emit(openAIChunk(chunkID, requestedModel, map[string]any{}, &stopReason)); err != nil {
			return http.StatusBadGateway, "stream_flush_failed", err
		}
	}
	if _, err := fmt.Fprint(w, "data: [DONE]\n\n"); err != nil {
		return http.StatusBadGateway, "stream_done_failed", err
	}
	flusher.Flush()
	return http.StatusOK, "", nil
}

func openAIChunk(id, model string, delta map[string]any, finishReason *string) map[string]any {
	choice := map[string]any{
		"index": 0,
		"delta": delta,
	}
	if finishReason == nil {
		choice["finish_reason"] = nil
	} else {
		choice["finish_reason"] = *finishReason
	}
	return map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{choice},
	}
}

func parseUpstreamError(body io.Reader) string {
	buf, _ := io.ReadAll(io.LimitReader(body, 1<<20))
	if len(buf) == 0 {
		return "Upstream provider returned an error"
	}
	var obj map[string]any
	if err := json.Unmarshal(buf, &obj); err == nil {
		if errObj, ok := obj["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok && msg != "" {
				return msg
			}
		}
		if msg, ok := obj["message"].(string); ok && msg != "" {
			return msg
		}
	}
	msg := strings.TrimSpace(string(buf))
	if msg == "" {
		return "Upstream provider returned an error"
	}
	return msg
}

func writeOpenAIError(w http.ResponseWriter, status int, message, errType, code string) {
	if errType == "" {
		errType = "invalid_request_error"
	}
	if code == "" {
		code = "error"
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
			"code":    code,
		},
	})
}
