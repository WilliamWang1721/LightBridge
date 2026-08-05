package service

import (
	"bytes"
	"context"
	"encoding/base64"
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

func TestOpenAIGatewayServiceForward_GrokOAuthUsesBuildProxy(t *testing.T) {

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"grok","stream":false,"input":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"x-request-id": []string{"rid-grok"},
		},
		Body: io.NopCloser(strings.NewReader(`{"id":"resp_grok","status":"completed","model":"grok-4.3","output":[],"usage":{"input_tokens":1,"output_tokens":1}}`)),
	}}

	svc := &OpenAIGatewayService{httpUpstream: upstream}
	account := &Account{
		ID:          42,
		Name:        "grok-oauth",
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "grok-access-token",
			"base_url":     "https://api.x.ai/v1",
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1/responses", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer grok-access-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, xai.GrokTokenAuthValue, upstream.lastReq.Header.Get(xai.GrokTokenAuthHeader))
	require.Equal(t, xai.GrokClientVersionValue, upstream.lastReq.Header.Get(xai.GrokClientVersionHeader))
	require.Empty(t, upstream.lastReq.Header.Get("chatgpt-account-id"))
	require.Equal(t, "grok", result.Model)
	require.Equal(t, "grok-4.3", result.UpstreamModel)
	require.Equal(t, "grok-4.3", gjson.GetBytes(upstream.lastBody, "model").String())
}

func TestBuildGrokResponsesRequestUsingAPIMode(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	account := &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"base_url":  "https://api.x.ai/v1",
			"using_api": true,
		},
	}

	req, err := buildGrokResponsesRequest(context.Background(), c, account, []byte(`{"model":"grok-4.3","stream":false}`), "token")
	require.NoError(t, err)
	require.Equal(t, "https://api.x.ai/v1/responses", req.URL.String())
	require.Empty(t, req.Header.Get(xai.GrokTokenAuthHeader))
	require.Empty(t, req.Header.Get(xai.GrokClientVersionHeader))
	require.Equal(t, HTTPUpstreamProfileGrok, HTTPUpstreamProfileFromContext(req.Context()))
}

func validGrokReplayEncryptedContentForTest() string {
	payload := make([]byte, 256)
	for i := range payload {
		payload[i] = byte(i)
	}
	return base64.RawStdEncoding.EncodeToString(payload)
}

func newGrokReplayTestContext(apiKeyID, groupID int64, body []byte) *gin.Context {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Set("api_key", &APIKey{ID: apiKeyID, GroupID: &groupID})
	return c
}

func TestGrokReasoningReplayInjectsContextBeforeToolOutput(t *testing.T) {
	model := "grok-4.5"
	firstBody := []byte(`{"model":"grok-4.5","input":[{"type":"message","role":"user","content":"inspect"}]}`)
	c := newGrokReplayTestContext(101, 202, firstBody)
	svc := &OpenAIGatewayService{}
	scope := resolveGrokReasoningReplayScope(c, firstBody, model)
	encrypted := validGrokReplayEncryptedContentForTest()
	completed := []byte(`{"id":"resp_grok_tool_1","output":[{"id":"rs_1","type":"reasoning","encrypted_content":"` + encrypted + `","summary":[]},{"id":"fc_1","type":"function_call","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"README.md\"}"}]}`)
	svc.cacheGrokReasoningReplay(context.Background(), scope, completed)

	secondBody := []byte(`{"model":"grok-4.5","previous_response_id":"resp_grok_tool_1","input":[{"type":"message","role":"user","content":"continue"},{"type":"function_call_output","call_id":"call_1","output":"contents"}]}`)
	patched, _, injected := svc.prepareGrokReasoningReplayRequest(context.Background(), c, secondBody, model)
	require.True(t, injected)
	require.False(t, gjson.GetBytes(patched, "previous_response_id").Exists())
	input := gjson.GetBytes(patched, "input").Array()
	require.Len(t, input, 4)
	require.Equal(t, "message", input[0].Get("type").String())
	require.Equal(t, "reasoning", input[1].Get("type").String())
	require.Equal(t, encrypted, input[1].Get("encrypted_content").String())
	require.Equal(t, "function_call", input[2].Get("type").String())
	require.Equal(t, "call_1", input[2].Get("call_id").String())
	require.Equal(t, "function_call_output", input[3].Get("type").String())
}

func TestGrokReasoningReplayIsIsolatedByAPIKey(t *testing.T) {
	model := "grok-4.5"
	firstBody := []byte(`{"model":"grok-4.5","input":"hello"}`)
	ownerContext := newGrokReplayTestContext(111, 333, firstBody)
	otherContext := newGrokReplayTestContext(222, 333, firstBody)
	svc := &OpenAIGatewayService{}
	scope := resolveGrokReasoningReplayScope(ownerContext, firstBody, model)
	encrypted := validGrokReplayEncryptedContentForTest()
	svc.cacheGrokReasoningReplay(context.Background(), scope, []byte(`{"id":"resp_isolated","output":[{"type":"reasoning","encrypted_content":"`+encrypted+`"},{"type":"function_call","call_id":"call_secret","name":"lookup","arguments":"{}"}]}`))

	secondBody := []byte(`{"model":"grok-4.5","previous_response_id":"resp_isolated","input":[{"type":"function_call_output","call_id":"call_secret","output":"ok"}]}`)
	patched, _, injected := svc.prepareGrokReasoningReplayRequest(context.Background(), otherContext, secondBody, model)
	require.False(t, injected)
	require.Len(t, gjson.GetBytes(patched, "input").Array(), 1)
	require.False(t, gjson.GetBytes(patched, "previous_response_id").Exists())
}

func TestGrokReplayCollectorIncludesReasoningAndToolCall(t *testing.T) {
	encrypted := validGrokReplayEncryptedContentForTest()
	collector := &openAIWSToolCallReplayCollector{}
	collector.AddEvent("response.output_item.done", []byte(`{"type":"response.output_item.done","item":{"id":"rs_1","type":"reasoning","encrypted_content":"`+encrypted+`"}}`))
	collector.AddEvent("response.output_item.done", []byte(`{"type":"response.output_item.done","item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"shell","arguments":"{}"}}`))

	items := collector.Items()
	require.Len(t, items, 2)
	require.Equal(t, "reasoning", gjson.GetBytes(items[0], "type").String())
	require.Equal(t, "function_call", gjson.GetBytes(items[1], "type").String())
}

func TestOpenAIGatewayServiceForward_GrokInvalidReplayRetriesWithoutCache(t *testing.T) {
	model := "grok-4.5"
	body := []byte(`{"model":"grok-4.5","stream":false,"previous_response_id":"resp_retry","input":[{"type":"function_call_output","call_id":"call_retry","output":"done"}]}`)
	c := newGrokReplayTestContext(401, 402, body)
	encrypted := validGrokReplayEncryptedContentForTest()
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"invalid encrypted_content signature for reasoning item"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-retry"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_retry_ok","status":"completed","model":"grok-4.5","output":[],"usage":{"input_tokens":1,"output_tokens":1}}`)),
		},
	}}
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	scope := resolveGrokReasoningReplayScope(c, body, model)
	svc.cacheGrokReasoningReplay(context.Background(), scope, []byte(`{"id":"resp_retry","output":[{"type":"reasoning","encrypted_content":"`+encrypted+`"},{"type":"function_call","call_id":"call_retry","name":"shell","arguments":"{}"}]}`))
	account := &Account{
		ID:          403,
		Name:        "grok-retry",
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "token", "base_url": "https://api.x.ai/v1"},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2)
	require.True(t, gjson.GetBytes(upstream.bodies[0], "input.0.type").String() == "reasoning")
	require.Equal(t, "function_call_output", gjson.GetBytes(upstream.bodies[1], "input.0.type").String())
	require.False(t, gjson.GetBytes(upstream.bodies[0], "previous_response_id").Exists())
	require.False(t, gjson.GetBytes(upstream.bodies[1], "previous_response_id").Exists())
}
