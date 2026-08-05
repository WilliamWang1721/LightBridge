package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/xai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func grokRouterTestAccount() *Account {
	return &Account{
		ID:          8451,
		Name:        "grok-router",
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "grok-router-token",
			"base_url":     "https://api.x.ai/v1",
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func grokRouterCompletedSSE(responseID string, output string) string {
	return strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"` + responseID + `","object":"response","status":"in_progress","model":"grok-4.5","output":[]}}`,
		"",
		`data: {"type":"response.completed","response":{"id":"` + responseID + `","object":"response","status":"completed","model":"grok-4.5","output":` + output + `,"usage":{"input_tokens":7,"output_tokens":3,"total_tokens":10}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
}

func TestForwardAsAnthropic_GrokBuildUsesXAIRequestAndPreservesTools(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{
		"model":"claude-sonnet-4-5",
		"max_tokens":1024,
		"stream":false,
		"messages":[{"role":"user","content":"Read README.md"}],
		"tools":[{"name":"read_file","description":"Read a file","input_schema":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	groupID := int64(8452)
	c.Set("api_key", &APIKey{ID: 8453, GroupID: &groupID})

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"x-request-id": []string{"rid-grok-anthropic"},
		},
		Body: io.NopCloser(strings.NewReader(grokRouterCompletedSSE("resp_grok_anthropic", `[{"id":"fc_1","type":"function_call","status":"completed","call_id":"toolu_1","name":"read_file","arguments":"{\"path\":\"README.md\"}"}]`))),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	result, err := svc.ForwardAsAnthropic(context.Background(), c, grokRouterTestAccount(), body, "grok-anthropic-session", "grok-4.5")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1/responses", upstream.lastReq.URL.String())
	require.Equal(t, xai.GrokTokenAuthValue, upstream.lastReq.Header.Get(xai.GrokTokenAuthHeader))
	require.Equal(t, xai.GrokClientVersionValue, upstream.lastReq.Header.Get(xai.GrokClientVersionHeader))
	require.Empty(t, upstream.lastReq.Header.Get("chatgpt-account-id"))
	require.Empty(t, upstream.lastReq.Header.Get("x-codex-turn-state"))
	require.True(t, gjson.GetBytes(upstream.lastBody, "stream").Bool())
	require.Equal(t, "grok-4.5", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "function", gjson.GetBytes(upstream.lastBody, "tools.0.type").String())
	require.Equal(t, "read_file", gjson.GetBytes(upstream.lastBody, "tools.0.name").String())
	require.Equal(t, "object", gjson.GetBytes(upstream.lastBody, "tools.0.parameters.type").String())
	require.Equal(t, "tool_use", gjson.GetBytes(rec.Body.Bytes(), "content.0.type").String())
	require.Equal(t, "read_file", gjson.GetBytes(rec.Body.Bytes(), "content.0.name").String())
}

func TestForwardAsChatCompletions_GrokBuildUsesXAIRequestAndReturnsToolCall(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{
		"model":"grok-4.5",
		"stream":false,
		"messages":[{"role":"user","content":"Read README.md"}],
		"tools":[{"type":"function","function":{"name":"read_file","description":"Read a file","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}}],
		"tool_choice":"auto"
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	groupID := int64(8462)
	c.Set("api_key", &APIKey{ID: 8463, GroupID: &groupID})

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"x-request-id": []string{"rid-grok-chat"},
		},
		Body: io.NopCloser(strings.NewReader(grokRouterCompletedSSE("resp_grok_chat", `[{"id":"fc_2","type":"function_call","status":"completed","call_id":"call_2","name":"read_file","arguments":"{\"path\":\"README.md\"}"}]`))),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, grokRouterTestAccount(), body, "grok-chat-session", "grok-4.5")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1/responses", upstream.lastReq.URL.String())
	require.Equal(t, xai.GrokTokenAuthValue, upstream.lastReq.Header.Get(xai.GrokTokenAuthHeader))
	require.Empty(t, upstream.lastReq.Header.Get("chatgpt-account-id"))
	require.Equal(t, "function", gjson.GetBytes(upstream.lastBody, "tools.0.type").String())
	require.Equal(t, "read_file", gjson.GetBytes(upstream.lastBody, "tools.0.name").String())
	require.Equal(t, "read_file", gjson.GetBytes(rec.Body.Bytes(), "choices.0.message.tool_calls.0.function.name").String())
	require.Equal(t, "call_2", gjson.GetBytes(rec.Body.Bytes(), "choices.0.message.tool_calls.0.id").String())
}

func TestForwardAsChatCompletions_GrokBuildReplaysReasoningForResponsesShapeToolOutput(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{
		"model":"grok-4.5",
		"stream":false,
		"prompt_cache_key":"grok-cli-session",
		"previous_response_id":"resp_grok_previous",
		"input":[{"type":"function_call_output","call_id":"call_replay","output":"file contents"}]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	groupID := int64(8472)
	c.Set("api_key", &APIKey{ID: 8473, GroupID: &groupID})

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
			"x-request-id": []string{"rid-grok-replay"},
		},
		Body: io.NopCloser(strings.NewReader(grokRouterCompletedSSE("resp_grok_after_tool", `[{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"done"}]}]`))),
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := grokRouterTestAccount()

	scope := resolveGrokReasoningReplayScope(c, body, "grok-4.5")
	encrypted := validGrokReplayEncryptedContentForTest()
	svc.cacheGrokReasoningReplay(context.Background(), scope, []byte(`{
		"id":"resp_grok_previous",
		"output":[
			{"id":"rs_replay","type":"reasoning","encrypted_content":"`+encrypted+`","summary":[]},
			{"id":"fc_replay","type":"function_call","call_id":"call_replay","name":"read_file","arguments":"{\"path\":\"README.md\"}"}
		]
	}`))

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "grok-cli-session", "grok-4.5")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, gjson.GetBytes(upstream.lastBody, "previous_response_id").Exists())
	input := gjson.GetBytes(upstream.lastBody, "input").Array()
	require.Len(t, input, 3)
	require.Equal(t, "reasoning", input[0].Get("type").String())
	require.Equal(t, encrypted, input[0].Get("encrypted_content").String())
	require.Equal(t, "function_call", input[1].Get("type").String())
	require.Equal(t, "call_replay", input[1].Get("call_id").String())
	require.Equal(t, "function_call_output", input[2].Get("type").String())
	require.Equal(t, "done", gjson.GetBytes(rec.Body.Bytes(), "choices.0.message.content").String())
}
