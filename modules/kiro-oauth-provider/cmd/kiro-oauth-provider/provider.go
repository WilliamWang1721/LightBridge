package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *server) doKiroRequest(ctx context.Context, acc *account, model string, payload map[string]any, stream bool) (*http.Response, []byte, error) {
	if acc == nil {
		return nil, nil, errors.New("missing account")
	}
	region := nonEmpty(acc.Region, s.cfg.Region)
	base := s.cfg.BaseURL
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "amazonq") {
		base = nonEmpty(s.cfg.AmazonQBaseURL, s.cfg.BaseURL)
	}
	endpoint := renderRegionTemplate(base, region)

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+strings.TrimSpace(acc.AccessToken))
	req.Header.Set("amz-sdk-invocation-id", newUUID())
	req.Header.Set("user-agent", defaultUserAgent)
	if stream {
		req.Header.Set("accept", "application/vnd.amazon.eventstream")
	} else {
		req.Header.Set("accept", "application/json")
	}

	resp, err := s.httpc.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if stream {
		return resp, nil, nil
	}
	bodyResp, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	_ = resp.Body.Close()
	return resp, bodyResp, nil
}

func (s *server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 20<<20))
	_ = r.Body.Close()
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "failed to read body", "invalid_request_error", "invalid_body")
		return
	}
	var req chatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "body must be valid JSON", "invalid_request_error", "invalid_json")
		return
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return
	}
	if len(req.Messages) == 0 {
		writeOpenAIError(w, http.StatusBadRequest, "messages is required", "invalid_request_error", "missing_messages")
		return
	}

	acc, err := s.getAccountForRequest(r.Context())
	if err != nil {
		injectQuotaError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	kiroReq, err := buildKiroRequest(req, acc)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_request")
		return
	}

	promptTokens := estimatePromptTokens(req)
	if req.Stream {
		s.proxyKiroStream(w, r.Context(), req, acc, kiroReq, promptTokens)
		return
	}
	s.proxyKiroNonStream(w, r.Context(), req, acc, kiroReq, promptTokens)
}

func (s *server) handleUpstreamStatus(w http.ResponseWriter, status int, body []byte, acc *account) bool {
	if status == http.StatusUnauthorized {
		if acc != nil {
			if _, err := s.refreshAccountTokens(context.Background(), acc.ID); err != nil {
				writeOpenAIError(w, http.StatusUnauthorized, err.Error(), "authentication_error", "invalid_token")
				return true
			}
		}
		return false
	}
	if status == http.StatusPaymentRequired {
		if acc != nil {
			_ = s.store.markCooldown(acc.ID, s.nextMonthFirstUTC(), "quota exhausted")
		}
		injectQuotaError(w, http.StatusServiceUnavailable, "Kiro quota exhausted")
		return true
	}
	if status == http.StatusForbidden {
		if acc != nil {
			_ = s.store.markCooldown(acc.ID, time.Now().UTC().Add(1*time.Hour), "forbidden")
		}
		writeOpenAIError(w, http.StatusForbidden, summarizeHTTPError(status, body), "permission_error", "forbidden")
		return true
	}
	if status == http.StatusTooManyRequests {
		if acc != nil {
			_ = s.store.markCooldown(acc.ID, time.Now().UTC().Add(2*time.Minute), "rate_limited")
		}
		writeOpenAIError(w, http.StatusTooManyRequests, summarizeHTTPError(status, body), "rate_limit_error", "rate_limited")
		return true
	}
	if status >= 500 {
		writeOpenAIError(w, http.StatusBadGateway, summarizeHTTPError(status, body), "api_error", "upstream_error")
		return true
	}
	return false
}

func (s *server) proxyKiroNonStream(w http.ResponseWriter, ctx context.Context, req chatCompletionRequest, acc *account, kiroReq map[string]any, promptTokens int) {
	resp, body, err := s.doKiroRequest(ctx, acc, req.Model, kiroReq, false)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "api_error", "upstream_unreachable")
		return
	}
	status := resp.StatusCode
	if status == http.StatusUnauthorized {
		if _, err := s.refreshAccountTokens(ctx, acc.ID); err == nil {
			newAcc, ok := s.store.getAccount(acc.ID)
			if ok {
				resp2, body2, err2 := s.doKiroRequest(ctx, newAcc, req.Model, kiroReq, false)
				if err2 == nil {
					resp = resp2
					body = body2
					status = resp2.StatusCode
				}
			}
		}
	}
	if status < 200 || status >= 300 {
		if s.handleUpstreamStatus(w, status, body, acc) {
			return
		}
		writeOpenAIError(w, http.StatusBadGateway, summarizeHTTPError(status, body), "api_error", "upstream_error")
		return
	}

	text, toolCalls, _ := parseKiroMixedResponse(string(body))
	reasoning, normal := splitThinking(text)
	completionTokens := estimateCompletionTokens(text, toolCalls)
	usage := withUsageObject(promptTokens, completionTokens)

	choice := map[string]any{
		"index": 0,
		"message": map[string]any{
			"role":    "assistant",
			"content": normal,
		},
		"finish_reason": "stop",
	}
	if strings.TrimSpace(reasoning) != "" {
		choice["message"].(map[string]any)["reasoning_content"] = reasoning
	}
	if len(toolCalls) > 0 {
		toolArr := make([]map[string]any, 0, len(toolCalls))
		for _, tc := range toolCalls {
			toolArr = append(toolArr, map[string]any{
				"id":   nonEmpty(strings.TrimSpace(tc.ID), newUUID()),
				"type": "function",
				"function": map[string]any{
					"name":      strings.TrimSpace(tc.Name),
					"arguments": strings.TrimSpace(tc.Args),
				},
			})
		}
		choice["message"].(map[string]any)["tool_calls"] = toolArr
		choice["finish_reason"] = "tool_calls"
	}

	respPayload := map[string]any{
		"id":      "chatcmpl-" + newUUID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]any{choice},
		"usage":   usage,
	}
	writeJSON(w, http.StatusOK, respPayload)
}

func (s *server) writeSSEChunk(w io.Writer, payload map[string]any) {
	b, _ := json.Marshal(payload)
	_, _ = io.WriteString(w, "data: ")
	_, _ = w.Write(b)
	_, _ = io.WriteString(w, "\n\n")
}

func (s *server) proxyKiroStream(w http.ResponseWriter, ctx context.Context, req chatCompletionRequest, acc *account, kiroReq map[string]any, promptTokens int) {
	resp, _, err := s.doKiroRequest(ctx, acc, req.Model, kiroReq, true)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "api_error", "upstream_unreachable")
		return
	}

	if resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()
		if _, err := s.refreshAccountTokens(ctx, acc.ID); err == nil {
			newAcc, ok := s.store.getAccount(acc.ID)
			if ok {
				resp, _, err = s.doKiroRequest(ctx, newAcc, req.Model, kiroReq, true)
				if err != nil {
					writeOpenAIError(w, http.StatusBadGateway, err.Error(), "api_error", "upstream_unreachable")
					return
				}
			}
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if s.handleUpstreamStatus(w, resp.StatusCode, body, acc) {
			return
		}
		writeOpenAIError(w, http.StatusBadGateway, summarizeHTTPError(resp.StatusCode, body), "api_error", "upstream_error")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.Header().Set("x-accel-buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeOpenAIError(w, http.StatusInternalServerError, "stream unsupported", "server_error", "stream_unsupported")
		return
	}

	chunkID := "chatcmpl-" + newUUID()
	created := time.Now().Unix()
	model := req.Model

	s.writeSSEChunk(w, map[string]any{
		"id":      chunkID,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{{
			"index": 0,
			"delta": map[string]any{"role": "assistant"},
		}},
	})
	flusher.Flush()

	reader := bufio.NewReader(resp.Body)
	var buffer strings.Builder
	var totalText strings.Builder
	toolCalls := make([]kiroToolCall, 0)
	streamTool := &streamToolState{}
	toolIndex := -1
	finishReason := "stop"

	for {
		part := make([]byte, 4096)
		n, readErr := reader.Read(part)
		if n > 0 {
			buffer.Write(part[:n])
			events, remaining := parseAwsEventStreamBuffer(buffer.String())
			buffer.Reset()
			buffer.WriteString(remaining)

			for _, ev := range events {
				switch ev.Type {
				case "content":
					if strings.TrimSpace(ev.Content) == "" {
						continue
					}
					totalText.WriteString(ev.Content)
					s.writeSSEChunk(w, map[string]any{
						"id":      chunkID,
						"object":  "chat.completion.chunk",
						"created": created,
						"model":   model,
						"choices": []map[string]any{{
							"index": 0,
							"delta": map[string]any{"content": ev.Content},
						}},
					})
					flusher.Flush()
				case "toolUse":
					finishReason = "tool_calls"
					if streamTool.ID != "" {
						toolCalls = append(toolCalls, kiroToolCall{ID: streamTool.ID, Name: streamTool.Name, Args: streamTool.Args.String()})
					}
					toolIndex++
					streamTool = &streamToolState{ID: nonEmpty(ev.ToolUseID, newUUID()), Name: ev.ToolName}
					if strings.TrimSpace(ev.ToolInput) != "" {
						streamTool.Args.WriteString(ev.ToolInput)
					}
					s.writeSSEChunk(w, map[string]any{
						"id":      chunkID,
						"object":  "chat.completion.chunk",
						"created": created,
						"model":   model,
						"choices": []map[string]any{{
							"index": 0,
							"delta": map[string]any{
								"tool_calls": []map[string]any{{
									"index": toolIndex,
									"id":    streamTool.ID,
									"type":  "function",
									"function": map[string]any{
										"name":      streamTool.Name,
										"arguments": "",
									},
								}},
							},
						}},
					})
					flusher.Flush()
				case "toolUseInput":
					if streamTool.ID == "" {
						continue
					}
					streamTool.Args.WriteString(ev.ToolInput)
					s.writeSSEChunk(w, map[string]any{
						"id":      chunkID,
						"object":  "chat.completion.chunk",
						"created": created,
						"model":   model,
						"choices": []map[string]any{{
							"index": 0,
							"delta": map[string]any{
								"tool_calls": []map[string]any{{
									"index": toolIndex,
									"function": map[string]any{
										"arguments": ev.ToolInput,
									},
								}},
							},
						}},
					})
					flusher.Flush()
				case "toolUseStop":
					if streamTool.ID != "" {
						toolCalls = append(toolCalls, kiroToolCall{ID: streamTool.ID, Name: streamTool.Name, Args: streamTool.Args.String()})
						streamTool = &streamToolState{}
					}
				}
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				writeOpenAIError(w, http.StatusBadGateway, readErr.Error(), "api_error", "stream_read_failed")
				return
			}
			break
		}
	}

	if streamTool.ID != "" {
		toolCalls = append(toolCalls, kiroToolCall{ID: streamTool.ID, Name: streamTool.Name, Args: streamTool.Args.String()})
	}
	toolCalls = dedupeToolCalls(toolCalls)

	completionTokens := estimateCompletionTokens(totalText.String(), toolCalls)
	s.writeSSEChunk(w, map[string]any{
		"id":      chunkID,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": finishReason,
		}},
		"usage": withUsageObject(promptTokens, completionTokens),
	})
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	flusher.Flush()
}
