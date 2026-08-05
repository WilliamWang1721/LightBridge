package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/xai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func grokWSBridgeCompletedSSE(responseID, encrypted, callID string) string {
	output := `[{"id":"rs_ws_1","type":"reasoning","encrypted_content":"` + encrypted + `","summary":[]},{"id":"fc_ws_1","type":"function_call","status":"completed","call_id":"` + callID + `","name":"read_file","arguments":"{\"path\":\"README.md\"}"}]`
	return strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"` + responseID + `","object":"response","status":"in_progress","model":"grok-4.5","output":[]}}`,
		"",
		`data: {"type":"response.output_item.done","output_index":0,"item":{"id":"rs_ws_1","type":"reasoning","encrypted_content":"` + encrypted + `","summary":[]}}`,
		"",
		`data: {"type":"response.output_item.done","output_index":1,"item":{"id":"fc_ws_1","type":"function_call","status":"completed","call_id":"` + callID + `","name":"read_file","arguments":"{\"path\":\"README.md\"}"}}`,
		"",
		`data: {"type":"response.completed","response":{"id":"` + responseID + `","object":"response","status":"completed","model":"grok-4.5","output":` + output + `,"usage":{"input_tokens":8,"output_tokens":3,"total_tokens":11}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
}

func grokWSBridgeTextCompletedSSE(responseID string) string {
	return strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"` + responseID + `","object":"response","status":"in_progress","model":"grok-4.5","output":[]}}`,
		"",
		`data: {"type":"response.completed","response":{"id":"` + responseID + `","object":"response","status":"completed","model":"grok-4.5","output":[{"id":"msg_ws_2","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"done"}]}],"usage":{"input_tokens":12,"output_tokens":2,"total_tokens":14}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
}

func TestOpenAIWSHTTPBridge_RetriesRejectedInputNamespace(t *testing.T) {

	payload := []byte(`{
		"type":"response.create",
		"model":"gpt-5.6-sol",
		"stream":true,
		"input":[
			{"type":"function_call","namespace":"mcp__docs","name":"lookup","call_id":"call_1","arguments":"{}"},
			{"type":"function_call_output","call_id":"call_1","output":"ok"}
		],
		"tools":[{"type":"namespace","name":"mcp__docs","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}]
	}`)
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[0].namespace'.","param":"input[0].namespace","type":"invalid_request_error"}}`,
			)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-http-bridge-namespace-ok"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`data: {"type":"response.completed","response":{"id":"resp_http_bridge_namespace_ok","object":"response","model":"gpt-5.6-sol","status":"completed","output":[],"usage":{"input_tokens":4,"output_tokens":1,"total_tokens":5}}}`,
				"",
				"data: [DONE]",
				"",
			}, "\n"))),
		},
	}}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream, toolCorrector: NewCodexToolCorrector()}
	account := &Account{
		ID:          95301,
		Name:        "openai-http-bridge-namespace",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://api.openai.com",
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/responses", nil)

	var events [][]byte
	result, err := svc.proxyOpenAIWSHTTPBridgeTurn(
		context.Background(), c, account, "sk-test", payload, len(payload), "gpt-5.6-sol", "", "", "", 1,
		func(message []byte) error {
			events = append(events, append([]byte(nil), message...))
			return nil
		},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "resp_http_bridge_namespace_ok", result.RequestID)
	require.NotEmpty(t, events)
	require.Len(t, upstream.bodies, 2)
	require.Equal(t, "mcp__docs", gjson.GetBytes(upstream.bodies[0], "input.0.namespace").String())
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.namespace").Exists())
	require.Equal(t, "mcp__docs", gjson.GetBytes(upstream.bodies[1], "tools.0.name").String())
}

func TestOpenAIWSHTTPBridge_GrokBuildReplaysReasoningAndToolCallAcrossTurns(t *testing.T) {

	firstPayload := []byte(`{
		"type":"response.create",
		"model":"grok-4.5",
		"stream":true,
		"prompt_cache_key":"grok-build-cli-ws-session",
		"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"Read README.md"}]}],
		"tools":[{"type":"function","name":"read_file","description":"Read a file","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}]
	}`)
	secondPayload := []byte(`{
		"type":"response.create",
		"model":"grok-4.5",
		"stream":true,
		"prompt_cache_key":"grok-build-cli-ws-session",
		"previous_response_id":"resp_grok_ws_tool_1",
		"input":[{"type":"function_call_output","call_id":"call_grok_ws_1","output":"README contents"}]
	}`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/responses", bytes.NewReader(nil))
	groupID := int64(95202)
	c.Set("api_key", &APIKey{ID: 95201, GroupID: &groupID})

	encrypted := validGrokReplayEncryptedContentForTest()
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-grok-ws-1"},
			},
			Body: io.NopCloser(strings.NewReader(grokWSBridgeCompletedSSE("resp_grok_ws_tool_1", encrypted, "call_grok_ws_1"))),
		},
		{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-grok-ws-2"},
			},
			Body: io.NopCloser(strings.NewReader(grokWSBridgeTextCompletedSSE("resp_grok_ws_tool_2"))),
		},
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := grokRouterTestAccount()

	var firstEvents [][]byte
	firstResult, err := svc.proxyOpenAIWSHTTPBridgeTurn(
		context.Background(),
		c,
		account,
		"grok-ws-token",
		firstPayload,
		len(firstPayload),
		"grok-4.5",
		"",
		"",
		"",
		1,
		func(message []byte) error {
			firstEvents = append(firstEvents, append([]byte(nil), message...))
			return nil
		},
	)
	require.NoError(t, err)
	require.NotNil(t, firstResult)
	require.Equal(t, "resp_grok_ws_tool_1", firstResult.RequestID)
	require.True(t, firstResult.wsReplayInputExists)
	require.Len(t, firstResult.wsReplayInput, 2)
	require.Equal(t, "reasoning", gjson.GetBytes(firstResult.wsReplayInput[0], "type").String())
	require.Equal(t, "function_call", gjson.GetBytes(firstResult.wsReplayInput[1], "type").String())
	require.NotEmpty(t, firstEvents)

	var secondEvents [][]byte
	secondResult, err := svc.proxyOpenAIWSHTTPBridgeTurn(
		context.Background(),
		c,
		account,
		"grok-ws-token",
		secondPayload,
		len(secondPayload),
		"grok-4.5",
		"",
		"",
		"",
		2,
		func(message []byte) error {
			secondEvents = append(secondEvents, append([]byte(nil), message...))
			return nil
		},
	)
	require.NoError(t, err)
	require.NotNil(t, secondResult)
	require.Equal(t, "resp_grok_ws_tool_2", secondResult.RequestID)
	require.NotEmpty(t, secondEvents)
	require.Len(t, upstream.requests, 2)
	require.Len(t, upstream.bodies, 2)

	for _, req := range upstream.requests {
		require.Equal(t, "https://cli-chat-proxy.grok.com/v1/responses", req.URL.String())
		require.Equal(t, "Bearer grok-ws-token", req.Header.Get("Authorization"))
		require.Equal(t, xai.GrokTokenAuthValue, req.Header.Get(xai.GrokTokenAuthHeader))
		require.Equal(t, xai.GrokClientVersionValue, req.Header.Get(xai.GrokClientVersionHeader))
		require.Empty(t, req.Header.Get("chatgpt-account-id"))
		require.Empty(t, req.Header.Get("x-codex-turn-state"))
	}

	secondUpstreamBody := upstream.bodies[1]
	require.False(t, gjson.GetBytes(secondUpstreamBody, "previous_response_id").Exists())
	input := gjson.GetBytes(secondUpstreamBody, "input").Array()
	require.Len(t, input, 3)
	require.Equal(t, "reasoning", input[0].Get("type").String())
	require.Equal(t, encrypted, input[0].Get("encrypted_content").String())
	require.Equal(t, "function_call", input[1].Get("type").String())
	require.Equal(t, "call_grok_ws_1", input[1].Get("call_id").String())
	require.Equal(t, "function_call_output", input[2].Get("type").String())
	require.Equal(t, "call_grok_ws_1", input[2].Get("call_id").String())
}
