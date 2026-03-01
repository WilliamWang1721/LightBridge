package gateway

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lightbridge/internal/providers"
	"lightbridge/internal/types"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type bufferedResponseWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header)}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(statusCode int) {
	if w.status == 0 {
		w.status = statusCode
	}
}

func (w *bufferedResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

func (w *bufferedResponseWriter) Flush() {}

func (w *bufferedResponseWriter) StatusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func shouldBridgeAnthropicMessages(ingressProtocol, endpointKind, providerProtocol string) bool {
	return types.NormalizeProtocol(ingressProtocol) == types.ProtocolAnthropic &&
		endpointKind == endpointKindMessages &&
		types.NormalizeProtocol(providerProtocol) != types.ProtocolAnthropic
}

func (s *Server) handleAnthropicMessagesBridge(
	ctx context.Context,
	w http.ResponseWriter,
	req *http.Request,
	adapter providers.Adapter,
	provider types.Provider,
	route *types.ResolvedRoute,
	rawBody []byte,
) (int, string, error) {
	openAIBody, wantsStream, err := convertAnthropicMessagesRequestToOpenAIChat(rawBody, route)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_anthropic_messages_request")
		return http.StatusBadRequest, "invalid_anthropic_messages_request", nil
	}

	req2 := req.Clone(ctx)
	req2.URL.Path = "/v1/chat/completions"
	req2.Body = ioNopCloser(openAIBody)
	req2.ContentLength = int64(len(openAIBody))

	buffered := newBufferedResponseWriter()
	status, code, callErr := adapter.Handle(ctx, buffered, req2, provider, route)
	if status == 0 {
		status = buffered.StatusCode()
	}

	if callErr != nil {
		if len(bytes.TrimSpace(buffered.body.Bytes())) > 0 {
			copyBridgeHeaders(w.Header(), buffered.header)
			w.WriteHeader(statusOrDefault(status, http.StatusBadGateway))
			_, _ = w.Write(buffered.body.Bytes())
			return statusOrDefault(status, http.StatusBadGateway), code, nil
		}
		return statusOrDefault(status, http.StatusBadGateway), code, callErr
	}

	if status >= 400 {
		copyBridgeHeaders(w.Header(), buffered.header)
		w.WriteHeader(status)
		_, _ = w.Write(buffered.body.Bytes())
		return status, code, nil
	}

	if wantsStream {
		out, convErr := convertOpenAIChatSSEToAnthropicMessagesSSE(buffered.body.Bytes(), route, rawBody)
		if convErr != nil {
			return http.StatusBadGateway, "anthropic_bridge_stream_translate_failed", convErr
		}
		w.Header().Set("content-type", "text/event-stream")
		w.Header().Set("cache-control", "no-cache")
		w.Header().Set("connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(out)
		return http.StatusOK, "", nil
	}

	out, convErr := convertOpenAIChatResponseToAnthropicMessageJSON(buffered.body.Bytes(), route, rawBody)
	if convErr != nil {
		return http.StatusBadGateway, "anthropic_bridge_translate_failed", convErr
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
	return http.StatusOK, "", nil
}

func convertAnthropicMessagesRequestToOpenAIChat(raw []byte, route *types.ResolvedRoute) ([]byte, bool, error) {
	if !gjson.ValidBytes(raw) {
		return nil, false, fmt.Errorf("body must be valid JSON")
	}
	root := gjson.ParseBytes(raw)

	model := strings.TrimSpace(route.UpstreamModel)
	if model == "" {
		model = strings.TrimSpace(root.Get("model").String())
	}
	if model == "" {
		return nil, false, fmt.Errorf("model is required")
	}
	stream := root.Get("stream").Type == gjson.True

	out := `{"model":"","messages":[],"stream":false}`
	out, _ = sjson.Set(out, "model", model)
	out, _ = sjson.Set(out, "stream", stream)

	if v := root.Get("max_tokens"); v.Exists() {
		out, _ = sjson.Set(out, "max_tokens", v.Int())
	}
	if v := root.Get("temperature"); v.Exists() {
		out, _ = sjson.Set(out, "temperature", v.Float())
	}
	if v := root.Get("top_p"); v.Exists() {
		out, _ = sjson.Set(out, "top_p", v.Float())
	}
	if v := root.Get("stop_sequences"); v.Exists() && v.IsArray() {
		stops := make([]string, 0, len(v.Array()))
		for _, s := range v.Array() {
			if t := strings.TrimSpace(s.String()); t != "" {
				stops = append(stops, t)
			}
		}
		if len(stops) > 0 {
			out, _ = sjson.Set(out, "stop", stops)
		}
	}

	var openAIMessages []string
	systemText := anthropicSystemToText(root.Get("system"))
	if systemText != "" {
		msg := `{"role":"system","content":""}`
		msg, _ = sjson.Set(msg, "content", systemText)
		openAIMessages = append(openAIMessages, msg)
	}

	if msgs := root.Get("messages"); msgs.Exists() && msgs.IsArray() {
		for _, m := range msgs.Array() {
			role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
			if role != "user" && role != "assistant" {
				continue
			}

			content := m.Get("content")
			if content.Type == gjson.String {
				msg := `{"role":"","content":""}`
				msg, _ = sjson.Set(msg, "role", role)
				msg, _ = sjson.Set(msg, "content", content.String())
				openAIMessages = append(openAIMessages, msg)
				continue
			}

			if !content.IsArray() {
				continue
			}

			textParts := make([]string, 0)
			toolCalls := make([]string, 0)
			toolResults := make([]string, 0)
			for _, block := range content.Array() {
				switch strings.ToLower(strings.TrimSpace(block.Get("type").String())) {
				case "text":
					if t := strings.TrimSpace(block.Get("text").String()); t != "" {
						textParts = append(textParts, t)
					}
				case "tool_use":
					if role != "assistant" {
						continue
					}
					name := strings.TrimSpace(block.Get("name").String())
					if name == "" {
						continue
					}
					id := strings.TrimSpace(block.Get("id").String())
					if id == "" {
						id = fmt.Sprintf("call_%d", time.Now().UnixNano())
					}
					argsRaw := strings.TrimSpace(block.Get("input").Raw)
					if argsRaw == "" || !gjson.Valid(argsRaw) {
						argsRaw = "{}"
					}
					tc := `{"id":"","type":"function","function":{"name":"","arguments":""}}`
					tc, _ = sjson.Set(tc, "id", id)
					tc, _ = sjson.Set(tc, "function.name", name)
					tc, _ = sjson.Set(tc, "function.arguments", argsRaw)
					toolCalls = append(toolCalls, tc)
				case "tool_result":
					if role != "user" {
						continue
					}
					toolID := strings.TrimSpace(block.Get("tool_use_id").String())
					if toolID == "" {
						continue
					}
					result := anthropicContentToText(block.Get("content"))
					toolMsg := `{"role":"tool","tool_call_id":"","content":""}`
					toolMsg, _ = sjson.Set(toolMsg, "tool_call_id", toolID)
					toolMsg, _ = sjson.Set(toolMsg, "content", result)
					toolResults = append(toolResults, toolMsg)
				}
			}

			if len(textParts) > 0 || len(toolCalls) > 0 {
				msg := `{"role":"","content":""}`
				msg, _ = sjson.Set(msg, "role", role)
				msg, _ = sjson.Set(msg, "content", strings.Join(textParts, "\n"))
				if len(toolCalls) > 0 {
					msg, _ = sjson.SetRaw(msg, "tool_calls", "["+strings.Join(toolCalls, ",")+"]")
				}
				openAIMessages = append(openAIMessages, msg)
			}
			openAIMessages = append(openAIMessages, toolResults...)
		}
	}

	if len(openAIMessages) == 0 {
		return nil, false, fmt.Errorf("messages is required")
	}
	for _, msg := range openAIMessages {
		out, _ = sjson.SetRaw(out, "messages.-1", msg)
	}
	return []byte(out), stream, nil
}

func anthropicSystemToText(v gjson.Result) string {
	if !v.Exists() {
		return ""
	}
	if v.Type == gjson.String {
		return strings.TrimSpace(v.String())
	}
	if !v.IsArray() {
		return ""
	}
	parts := make([]string, 0, len(v.Array()))
	for _, item := range v.Array() {
		if strings.ToLower(strings.TrimSpace(item.Get("type").String())) != "text" {
			continue
		}
		if t := strings.TrimSpace(item.Get("text").String()); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}

func anthropicContentToText(v gjson.Result) string {
	if !v.Exists() {
		return ""
	}
	if v.Type == gjson.String {
		return v.String()
	}
	if v.IsArray() {
		parts := make([]string, 0, len(v.Array()))
		for _, item := range v.Array() {
			if strings.ToLower(strings.TrimSpace(item.Get("type").String())) != "text" {
				continue
			}
			if t := strings.TrimSpace(item.Get("text").String()); t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n")
	}
	return strings.TrimSpace(v.String())
}

func convertOpenAIChatResponseToAnthropicMessageJSON(openAIResp []byte, route *types.ResolvedRoute, rawReq []byte) ([]byte, error) {
	if !gjson.ValidBytes(openAIResp) {
		return nil, fmt.Errorf("bridge response is not valid JSON")
	}
	root := gjson.ParseBytes(openAIResp)
	choice := root.Get("choices.0")
	if !choice.Exists() {
		return nil, fmt.Errorf("bridge response missing choices")
	}

	message := choice.Get("message")
	contentBlocks := make([]string, 0)
	text := strings.TrimSpace(message.Get("content").String())
	if text != "" {
		tb := `{"type":"text","text":""}`
		tb, _ = sjson.Set(tb, "text", text)
		contentBlocks = append(contentBlocks, tb)
	}

	if tc := message.Get("tool_calls"); tc.Exists() && tc.IsArray() {
		for _, item := range tc.Array() {
			id := strings.TrimSpace(item.Get("id").String())
			if id == "" {
				id = fmt.Sprintf("toolu_%d", time.Now().UnixNano())
			}
			name := strings.TrimSpace(item.Get("function.name").String())
			if name == "" {
				continue
			}
			argsRaw := strings.TrimSpace(item.Get("function.arguments").String())
			if argsRaw == "" || !gjson.Valid(argsRaw) {
				argsRaw = "{}"
			}
			tu := `{"type":"tool_use","id":"","name":"","input":{}}`
			tu, _ = sjson.Set(tu, "id", id)
			tu, _ = sjson.Set(tu, "name", name)
			tu, _ = sjson.SetRaw(tu, "input", argsRaw)
			contentBlocks = append(contentBlocks, tu)
		}
	}
	if len(contentBlocks) == 0 {
		tb := `{"type":"text","text":""}`
		contentBlocks = append(contentBlocks, tb)
	}

	model := strings.TrimSpace(root.Get("model").String())
	if model == "" && route != nil {
		model = strings.TrimSpace(route.UpstreamModel)
	}
	if model == "" {
		model = strings.TrimSpace(gjson.GetBytes(rawReq, "model").String())
	}
	if model == "" {
		model = "unknown"
	}

	stopReason := openAIFinishReasonToAnthropic(choice.Get("finish_reason").String())
	prompt := int(root.Get("usage.prompt_tokens").Int())
	completion := int(root.Get("usage.completion_tokens").Int())
	out := `{"id":"","type":"message","role":"assistant","model":"","content":[],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}}`
	id := strings.TrimSpace(root.Get("id").String())
	if id == "" {
		id = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	out, _ = sjson.Set(out, "id", id)
	out, _ = sjson.Set(out, "model", model)
	out, _ = sjson.Set(out, "stop_reason", stopReason)
	out, _ = sjson.Set(out, "usage.input_tokens", prompt)
	out, _ = sjson.Set(out, "usage.output_tokens", completion)
	for _, block := range contentBlocks {
		out, _ = sjson.SetRaw(out, "content.-1", block)
	}
	return []byte(out), nil
}

func convertOpenAIChatSSEToAnthropicMessagesSSE(openAISSE []byte, route *types.ResolvedRoute, rawReq []byte) ([]byte, error) {
	sc := bufio.NewScanner(bytes.NewReader(openAISSE))
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)

	var out bytes.Buffer
	msgID := ""
	model := ""
	opened := false
	contentOpened := false
	stopped := false

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			break
		}
		if !gjson.Valid(payload) {
			continue
		}

		if msgID == "" {
			msgID = strings.TrimSpace(gjson.Get(payload, "id").String())
			if msgID == "" {
				msgID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
			}
		}
		if model == "" {
			model = strings.TrimSpace(gjson.Get(payload, "model").String())
			if model == "" && route != nil {
				model = strings.TrimSpace(route.UpstreamModel)
			}
			if model == "" {
				model = strings.TrimSpace(gjson.GetBytes(rawReq, "model").String())
			}
			if model == "" {
				model = "unknown"
			}
		}

		if !opened {
			data := `{"type":"message_start","message":{"id":"","type":"message","role":"assistant","model":"","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":0,"output_tokens":0}}}`
			data, _ = sjson.Set(data, "message.id", msgID)
			data, _ = sjson.Set(data, "message.model", model)
			data, _ = sjson.Set(data, "message.usage.input_tokens", int(gjson.Get(payload, "usage.prompt_tokens").Int()))
			writeAnthropicSSEEvent(&out, "message_start", data)
			opened = true
		}

		deltaContent := gjson.Get(payload, "choices.0.delta.content").String()
		if deltaContent != "" {
			if !contentOpened {
				writeAnthropicSSEEvent(&out, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
				contentOpened = true
			}
			data := `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":""}}`
			data, _ = sjson.Set(data, "delta.text", deltaContent)
			writeAnthropicSSEEvent(&out, "content_block_delta", data)
		}

		if finish := strings.TrimSpace(gjson.Get(payload, "choices.0.finish_reason").String()); finish != "" {
			if contentOpened {
				writeAnthropicSSEEvent(&out, "content_block_stop", `{"type":"content_block_stop","index":0}`)
			}
			data := `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":0}}`
			data, _ = sjson.Set(data, "delta.stop_reason", openAIFinishReasonToAnthropic(finish))
			data, _ = sjson.Set(data, "usage.output_tokens", int(gjson.Get(payload, "usage.completion_tokens").Int()))
			writeAnthropicSSEEvent(&out, "message_delta", data)
			writeAnthropicSSEEvent(&out, "message_stop", `{"type":"message_stop"}`)
			stopped = true
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	if opened && !stopped {
		if contentOpened {
			writeAnthropicSSEEvent(&out, "content_block_stop", `{"type":"content_block_stop","index":0}`)
		}
		writeAnthropicSSEEvent(&out, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":0}}`)
		writeAnthropicSSEEvent(&out, "message_stop", `{"type":"message_stop"}`)
	}

	return out.Bytes(), nil
}

func writeAnthropicSSEEvent(out *bytes.Buffer, event, data string) {
	out.WriteString("event: ")
	out.WriteString(event)
	out.WriteString("\n")
	out.WriteString("data: ")
	out.WriteString(data)
	out.WriteString("\n\n")
}

func openAIFinishReasonToAnthropic(reason string) string {
	switch strings.TrimSpace(reason) {
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "stop_sequence"
	default:
		return "end_turn"
	}
}

func copyBridgeHeaders(dst, src http.Header) {
	for k := range dst {
		dst.Del(k)
	}
	for k, values := range src {
		switch strings.ToLower(k) {
		case "connection", "proxy-connection", "keep-alive", "te", "trailer", "transfer-encoding", "upgrade":
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}
