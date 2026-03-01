package gateway

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"lightbridge/internal/providers"
	"lightbridge/internal/types"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func shouldBridgeGeminiNative(ingressProtocol, endpointKind, providerProtocol string) bool {
	if types.NormalizeProtocol(ingressProtocol) != types.ProtocolGemini {
		return false
	}
	switch endpointKind {
	case endpointKindGenerateContent, endpointKindStreamGenerateContent, endpointKindCountTokens:
		return types.NormalizeProtocol(providerProtocol) != types.ProtocolGemini
	default:
		return false
	}
}

func (s *Server) handleGeminiNativeBridge(
	ctx context.Context,
	w http.ResponseWriter,
	req *http.Request,
	adapter providers.Adapter,
	provider types.Provider,
	route *types.ResolvedRoute,
	rawBody []byte,
	endpointKind string,
) (int, string, error) {
	modelFromPath := strings.TrimSpace(requestModelFromPath(req.URL.Path))
	if endpointKind == endpointKindCountTokens {
		total := estimateGeminiCountTokens(rawBody)
		out := `{"totalTokens":0,"cachedContentTokenCount":0}`
		out, _ = sjson.Set(out, "totalTokens", total)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(out))
		return http.StatusOK, "", nil
	}

	wantsStream := endpointKind == endpointKindStreamGenerateContent
	openAIBody, err := convertGeminiNativeRequestToOpenAIChat(rawBody, route, modelFromPath, wantsStream)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_gemini_request")
		return http.StatusBadRequest, "invalid_gemini_request", nil
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
		out, convErr := convertOpenAIChatSSEToGeminiGenerateContentSSE(buffered.body.Bytes())
		if convErr != nil {
			return http.StatusBadGateway, "gemini_bridge_stream_translate_failed", convErr
		}
		w.Header().Set("content-type", "text/event-stream")
		w.Header().Set("cache-control", "no-cache")
		w.Header().Set("connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(out)
		return http.StatusOK, "", nil
	}

	out, convErr := convertOpenAIChatResponseToGeminiGenerateContent(buffered.body.Bytes())
	if convErr != nil {
		return http.StatusBadGateway, "gemini_bridge_translate_failed", convErr
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
	return http.StatusOK, "", nil
}

func convertGeminiNativeRequestToOpenAIChat(raw []byte, route *types.ResolvedRoute, modelFromPath string, stream bool) ([]byte, error) {
	if !gjson.ValidBytes(raw) {
		return nil, fmt.Errorf("body must be valid JSON")
	}
	root := gjson.ParseBytes(raw)

	model := strings.TrimSpace(route.UpstreamModel)
	if model == "" {
		model = strings.TrimSpace(modelFromPath)
	}
	if model == "" {
		model = strings.TrimSpace(root.Get("model").String())
	}
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}

	out := `{"model":"","messages":[],"stream":false}`
	out, _ = sjson.Set(out, "model", model)
	out, _ = sjson.Set(out, "stream", stream)

	genCfg := root.Get("generationConfig")
	if v := genCfg.Get("maxOutputTokens"); v.Exists() {
		out, _ = sjson.Set(out, "max_tokens", v.Int())
	}
	if v := genCfg.Get("temperature"); v.Exists() {
		out, _ = sjson.Set(out, "temperature", v.Float())
	}
	if v := genCfg.Get("topP"); v.Exists() {
		out, _ = sjson.Set(out, "top_p", v.Float())
	}
	if v := genCfg.Get("stopSequences"); v.Exists() && v.IsArray() {
		stops := make([]string, 0, len(v.Array()))
		for _, item := range v.Array() {
			if t := strings.TrimSpace(item.String()); t != "" {
				stops = append(stops, t)
			}
		}
		if len(stops) > 0 {
			out, _ = sjson.Set(out, "stop", stops)
		}
	}

	systemText := geminiPartsToText(root.Get("systemInstruction.parts"))
	if systemText == "" {
		systemText = strings.TrimSpace(root.Get("systemInstruction.text").String())
	}
	if systemText != "" {
		msg := `{"role":"system","content":""}`
		msg, _ = sjson.Set(msg, "content", systemText)
		out, _ = sjson.SetRaw(out, "messages.-1", msg)
	}

	var toolIDSeq int
	if contents := root.Get("contents"); contents.Exists() && contents.IsArray() {
		for _, item := range contents.Array() {
			role := strings.ToLower(strings.TrimSpace(item.Get("role").String()))
			switch role {
			case "", "user":
				role = "user"
			case "model", "assistant":
				role = "assistant"
			default:
				role = "user"
			}

			textParts := make([]string, 0)
			toolCalls := make([]string, 0)
			toolResults := make([]string, 0)
			parts := item.Get("parts")
			if parts.Exists() && parts.IsArray() {
				for _, part := range parts.Array() {
					if txt := strings.TrimSpace(part.Get("text").String()); txt != "" {
						textParts = append(textParts, txt)
						continue
					}
					fc := part.Get("functionCall")
					if fc.Exists() {
						name := strings.TrimSpace(fc.Get("name").String())
						if name == "" {
							continue
						}
						toolIDSeq++
						callID := fmt.Sprintf("call_%d", toolIDSeq)
						argsRaw := strings.TrimSpace(fc.Get("args").Raw)
						if argsRaw == "" || !gjson.Valid(argsRaw) {
							argsRaw = "{}"
						}
						tc := `{"id":"","type":"function","function":{"name":"","arguments":""}}`
						tc, _ = sjson.Set(tc, "id", callID)
						tc, _ = sjson.Set(tc, "function.name", name)
						tc, _ = sjson.Set(tc, "function.arguments", argsRaw)
						toolCalls = append(toolCalls, tc)
						continue
					}
					fr := part.Get("functionResponse")
					if fr.Exists() {
						name := strings.TrimSpace(fr.Get("name").String())
						if name == "" {
							name = "tool"
						}
						contentRaw := strings.TrimSpace(fr.Get("response").Raw)
						if contentRaw == "" {
							contentRaw = strings.TrimSpace(fr.Get("response").String())
						}
						if contentRaw == "" {
							contentRaw = "{}"
						}
						toolMsg := `{"role":"tool","tool_call_id":"","content":""}`
						toolMsg, _ = sjson.Set(toolMsg, "tool_call_id", name)
						toolMsg, _ = sjson.Set(toolMsg, "content", contentRaw)
						toolResults = append(toolResults, toolMsg)
					}
				}
			}

			if len(textParts) > 0 || len(toolCalls) > 0 {
				msg := `{"role":"","content":""}`
				msg, _ = sjson.Set(msg, "role", role)
				msg, _ = sjson.Set(msg, "content", strings.Join(textParts, "\n"))
				if len(toolCalls) > 0 {
					msg, _ = sjson.SetRaw(msg, "tool_calls", "["+strings.Join(toolCalls, ",")+"]")
				}
				out, _ = sjson.SetRaw(out, "messages.-1", msg)
			}
			for _, tm := range toolResults {
				out, _ = sjson.SetRaw(out, "messages.-1", tm)
			}
		}
	}

	if len(gjson.Get(out, "messages").Array()) == 0 {
		return nil, fmt.Errorf("contents is required")
	}

	if tools := root.Get("tools"); tools.Exists() && tools.IsArray() {
		for _, tool := range tools.Array() {
			decl := tool.Get("functionDeclarations")
			if !decl.Exists() || !decl.IsArray() {
				continue
			}
			for _, fn := range decl.Array() {
				name := strings.TrimSpace(fn.Get("name").String())
				if name == "" {
					continue
				}
				item := `{"type":"function","function":{"name":"","description":"","parameters":{}}}`
				item, _ = sjson.Set(item, "function.name", name)
				item, _ = sjson.Set(item, "function.description", strings.TrimSpace(fn.Get("description").String()))
				if params := fn.Get("parameters"); params.Exists() && strings.TrimSpace(params.Raw) != "" {
					item, _ = sjson.SetRaw(item, "function.parameters", params.Raw)
				}
				out, _ = sjson.SetRaw(out, "tools.-1", item)
			}
		}
	}

	mode := strings.ToUpper(strings.TrimSpace(root.Get("toolConfig.functionCallingConfig.mode").String()))
	switch mode {
	case "ANY":
		out, _ = sjson.Set(out, "tool_choice", "required")
	case "NONE":
		out, _ = sjson.Set(out, "tool_choice", "none")
	case "AUTO":
		out, _ = sjson.Set(out, "tool_choice", "auto")
	}

	return []byte(out), nil
}

func geminiPartsToText(v gjson.Result) string {
	if !v.Exists() || !v.IsArray() {
		return ""
	}
	parts := make([]string, 0, len(v.Array()))
	for _, item := range v.Array() {
		if t := strings.TrimSpace(item.Get("text").String()); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}

func convertOpenAIChatResponseToGeminiGenerateContent(openAIResp []byte) ([]byte, error) {
	if !gjson.ValidBytes(openAIResp) {
		return nil, fmt.Errorf("bridge response is not valid JSON")
	}
	root := gjson.ParseBytes(openAIResp)
	choice := root.Get("choices.0")
	if !choice.Exists() {
		return nil, fmt.Errorf("bridge response missing choices")
	}

	parts := make([]string, 0)
	if text := strings.TrimSpace(choice.Get("message.content").String()); text != "" {
		p := `{"text":""}`
		p, _ = sjson.Set(p, "text", text)
		parts = append(parts, p)
	}
	if tcs := choice.Get("message.tool_calls"); tcs.Exists() && tcs.IsArray() {
		for _, tc := range tcs.Array() {
			name := strings.TrimSpace(tc.Get("function.name").String())
			if name == "" {
				continue
			}
			argsRaw := strings.TrimSpace(tc.Get("function.arguments").String())
			if argsRaw == "" || !gjson.Valid(argsRaw) {
				argsRaw = "{}"
			}
			p := `{"functionCall":{"name":"","args":{}}}`
			p, _ = sjson.Set(p, "functionCall.name", name)
			p, _ = sjson.SetRaw(p, "functionCall.args", argsRaw)
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		p := `{"text":""}`
		parts = append(parts, p)
	}

	finish := openAIFinishReasonToGemini(choice.Get("finish_reason").String())
	out := `{"candidates":[{"index":0,"content":{"role":"model","parts":[]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":0,"candidatesTokenCount":0,"totalTokenCount":0},"modelVersion":""}`
	for _, p := range parts {
		out, _ = sjson.SetRaw(out, "candidates.0.content.parts.-1", p)
	}
	out, _ = sjson.Set(out, "candidates.0.finishReason", finish)
	out, _ = sjson.Set(out, "usageMetadata.promptTokenCount", int(root.Get("usage.prompt_tokens").Int()))
	out, _ = sjson.Set(out, "usageMetadata.candidatesTokenCount", int(root.Get("usage.completion_tokens").Int()))
	out, _ = sjson.Set(out, "usageMetadata.totalTokenCount", int(root.Get("usage.total_tokens").Int()))
	out, _ = sjson.Set(out, "modelVersion", strings.TrimSpace(root.Get("model").String()))
	return []byte(out), nil
}

func convertOpenAIChatSSEToGeminiGenerateContentSSE(openAISSE []byte) ([]byte, error) {
	sc := bufio.NewScanner(bytes.NewReader(openAISSE))
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800)
	var out bytes.Buffer

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		if !gjson.Valid(payload) {
			continue
		}

		idx := int(gjson.Get(payload, "choices.0.index").Int())
		if deltaText := strings.TrimSpace(gjson.Get(payload, "choices.0.delta.content").String()); deltaText != "" {
			chunk := `{"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":""}]}}]}`
			chunk, _ = sjson.Set(chunk, "candidates.0.index", idx)
			chunk, _ = sjson.Set(chunk, "candidates.0.content.parts.0.text", deltaText)
			writeGeminiSSEData(&out, chunk)
		}
		finish := strings.TrimSpace(gjson.Get(payload, "choices.0.finish_reason").String())
		if finish != "" {
			chunk := `{"candidates":[{"index":0,"finishReason":"STOP"}]}`
			chunk, _ = sjson.Set(chunk, "candidates.0.index", idx)
			chunk, _ = sjson.Set(chunk, "candidates.0.finishReason", openAIFinishReasonToGemini(finish))
			writeGeminiSSEData(&out, chunk)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeGeminiSSEData(out *bytes.Buffer, data string) {
	out.WriteString("data: ")
	out.WriteString(data)
	out.WriteString("\n\n")
}

func openAIFinishReasonToGemini(reason string) string {
	switch strings.TrimSpace(reason) {
	case "length":
		return "MAX_TOKENS"
	case "tool_calls":
		return "STOP"
	case "content_filter":
		return "SAFETY"
	default:
		return "STOP"
	}
}

func estimateGeminiCountTokens(raw []byte) int {
	if !gjson.ValidBytes(raw) {
		return 0
	}
	root := gjson.ParseBytes(raw)
	var buf strings.Builder
	if txt := geminiPartsToText(root.Get("systemInstruction.parts")); txt != "" {
		buf.WriteString(txt)
		buf.WriteByte('\n')
	}
	if txt := strings.TrimSpace(root.Get("systemInstruction.text").String()); txt != "" {
		buf.WriteString(txt)
		buf.WriteByte('\n')
	}
	if contents := root.Get("contents"); contents.Exists() && contents.IsArray() {
		for _, c := range contents.Array() {
			if parts := c.Get("parts"); parts.Exists() && parts.IsArray() {
				for _, p := range parts.Array() {
					if t := strings.TrimSpace(p.Get("text").String()); t != "" {
						buf.WriteString(t)
						buf.WriteByte('\n')
					}
					if fn := strings.TrimSpace(p.Get("functionCall.name").String()); fn != "" {
						buf.WriteString(fn)
						buf.WriteByte('\n')
					}
				}
			}
		}
	}
	runes := utf8.RuneCountInString(buf.String())
	if runes <= 0 {
		return 0
	}
	tokens := runes / 4
	if runes%4 != 0 {
		tokens++
	}
	if tokens <= 0 {
		tokens = 1
	}
	return tokens
}
