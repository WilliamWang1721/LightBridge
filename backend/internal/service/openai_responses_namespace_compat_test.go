//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/openai_compat"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestShouldRetryOpenAIResponsesWithoutInputNamespaces(t *testing.T) {
	t.Parallel()

	reported := []byte(`{"error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[18].namespace'.","param":"input[18].namespace","type":"invalid_request_error"}}`)
	require.True(t, shouldRetryOpenAIResponsesWithoutInputNamespaces(http.StatusBadRequest, reported))
	require.True(t, shouldRetryOpenAIResponsesWithoutInputNamespaces(http.StatusBadRequest, []byte(`{"error":{"code":"unknown_parameter","message":"Unknown parameter: input.18.namespace.","param":"input.18.namespace","type":"invalid_request_error"}}`)))
	require.False(t, shouldRetryOpenAIResponsesWithoutInputNamespaces(http.StatusUnprocessableEntity, reported))
	require.False(t, shouldRetryOpenAIResponsesWithoutInputNamespaces(http.StatusBadRequest, []byte(`{"error":{"code":"unknown_parameter","param":"tools[0].namespace"}}`)))
	require.False(t, shouldRetryOpenAIResponsesWithoutInputNamespaces(http.StatusBadRequest, []byte(`{"error":{"code":"missing_parameter","param":"input[18].namespace"}}`)))
}

func TestShouldRetryOpenAIResponsesWSEventWithoutInputNamespaces(t *testing.T) {
	t.Parallel()

	require.True(t, shouldRetryOpenAIResponsesWSEventWithoutInputNamespaces([]byte(`{"type":"error","error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[4].namespace'.","param":"input[4].namespace","type":"invalid_request_error"}}`)))
	require.False(t, shouldRetryOpenAIResponsesWSEventWithoutInputNamespaces([]byte(`{"type":"error","error":{"code":"unknown_parameter","message":"Unknown parameter: 'tools[0].namespace'.","param":"tools[0].namespace","type":"invalid_request_error"}}`)))
}

func TestStripOpenAIResponsesInputNamespacesPreservesTools(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gpt-5.6-sol",
		"input":[
			{"type":"function_call","namespace":"mcp__docs","name":"lookup","call_id":"call_1","arguments":"{}"},
			{"type":"function_call_output","call_id":"call_1","output":"ok"}
		],
		"tools":[{"type":"namespace","name":"mcp__docs","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}]
	}`)

	normalized, changed, err := stripOpenAIResponsesInputNamespacesFromBody(body)
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(normalized, "input.0.namespace").Exists())
	require.Equal(t, "lookup", gjson.GetBytes(normalized, "input.0.name").String())
	require.Equal(t, "call_1", gjson.GetBytes(normalized, "input.0.call_id").String())
	require.Equal(t, "namespace", gjson.GetBytes(normalized, "tools.0.type").String())
	require.Equal(t, "mcp__docs", gjson.GetBytes(normalized, "tools.0.name").String())
}

func TestStripOpenAIResponsesInputNamespacesSupportsSingleInputObject(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gpt-5.6-sol",
		"input":{"type":"function_call","namespace":"mcp__docs","name":"lookup","call_id":"call_1","arguments":"{}"},
		"tools":[{"type":"namespace","name":"mcp__docs","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}]
	}`)

	normalized, changed, err := stripOpenAIResponsesInputNamespacesFromBody(body)
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(normalized, "input.namespace").Exists())
	require.Equal(t, "mcp__docs", gjson.GetBytes(normalized, "tools.0.name").String())
}

func TestOpenAIWSInputNamespaceCompatFrameConnRetriesOnSameConnection(t *testing.T) {
	t.Parallel()

	inner := &openAIWSCaptureConn{events: [][]byte{
		[]byte(`{"type":"error","error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[0].namespace'.","param":"input[0].namespace","type":"invalid_request_error"}}`),
		[]byte(`{"type":"response.completed","response":{"id":"resp_namespace_ws_ok","status":"completed","output":[],"usage":{"input_tokens":2,"output_tokens":1}}}`),
		[]byte(`{"type":"response.completed","response":{"id":"resp_namespace_ws_second","status":"completed","output":[],"usage":{"input_tokens":2,"output_tokens":1}}}`),
	}}
	conn := newOpenAIWSInputNamespaceCompatFrameConn(inner, 77, time.Second)
	request := []byte(`{
		"type":"response.create",
		"model":"gpt-5.6-sol",
		"input":[
			{"type":"function_call","namespace":"mcp__docs","name":"lookup","call_id":"call_1","arguments":"{}"},
			{"type":"function_call_output","call_id":"call_1","output":"ok"}
		],
		"tools":[{"type":"namespace","name":"mcp__docs","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}]
	}`)

	require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, request))
	msgType, response, err := conn.ReadFrame(context.Background())
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, msgType)
	require.Equal(t, "response.completed", gjson.GetBytes(response, "type").String())
	require.Len(t, inner.writes, 2)
	require.Equal(t, "mcp__docs", gjson.Get(requestToJSONString(inner.writes[0]), "input.0.namespace").String())
	require.False(t, gjson.Get(requestToJSONString(inner.writes[1]), "input.0.namespace").Exists())
	require.Equal(t, "mcp__docs", gjson.Get(requestToJSONString(inner.writes[1]), "tools.0.name").String())

	require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, request))
	_, secondResponse, err := conn.ReadFrame(context.Background())
	require.NoError(t, err)
	require.Equal(t, "resp_namespace_ws_second", gjson.GetBytes(secondResponse, "response.id").String())
	require.Len(t, inner.writes, 3, "learned connection capability must avoid another rejected write")
	require.False(t, gjson.Get(requestToJSONString(inner.writes[2]), "input.0.namespace").Exists())
	require.Equal(t, "mcp__docs", gjson.Get(requestToJSONString(inner.writes[2]), "tools.0.name").String())
}

func TestOpenAIWSInputNamespaceCompatFrameConnRetriesAtMostOnce(t *testing.T) {
	t.Parallel()

	rejection := []byte(`{"type":"error","error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[0].namespace'.","param":"input[0].namespace","type":"invalid_request_error"}}`)
	inner := &openAIWSCaptureConn{events: [][]byte{rejection, rejection}}
	conn := newOpenAIWSInputNamespaceCompatFrameConn(inner, 78, time.Second)
	request := []byte(`{"type":"response.create","model":"gpt-5.6-sol","input":[{"type":"function_call","namespace":"mcp__docs","name":"lookup","call_id":"call_1","arguments":"{}"}]}`)

	require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, request))
	_, response, err := conn.ReadFrame(context.Background())
	require.NoError(t, err)
	require.Equal(t, "error", gjson.GetBytes(response, "type").String(), "second rejection must be surfaced instead of looping")
	require.Len(t, inner.writes, 2)
}

func TestOpenAIGatewayForwardRetriesRejectedInputNamespaceOnce(t *testing.T) {
	body := namespaceCompatibilityRequestBody()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{responses: namespaceCompatibilityResponses()}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}
	account := namespaceCompatibilityAccount(false)

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2)
	require.Equal(t, "mcp__docs", gjson.GetBytes(upstream.bodies[0], "input.0.namespace").String())
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.namespace").Exists())
	require.Equal(t, "lookup", gjson.GetBytes(upstream.bodies[1], "input.0.name").String())
	require.Equal(t, "namespace", gjson.GetBytes(upstream.bodies[1], "tools.0.type").String())
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", gjson.Get(rec.Body.String(), "output.0.content.0.text").String())
}

func TestOpenAIGatewayPassthroughRetriesRejectedInputNamespaceOnce(t *testing.T) {
	body := namespaceCompatibilityRequestBody()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{responses: namespaceCompatibilityResponses()}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}
	account := namespaceCompatibilityAccount(true)

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.bodies, 2)
	require.Equal(t, "mcp__docs", gjson.GetBytes(upstream.bodies[0], "input.0.namespace").String())
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.namespace").Exists())
	require.Equal(t, "namespace", gjson.GetBytes(upstream.bodies[1], "tools.0.type").String())
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", gjson.Get(rec.Body.String(), "output.0.content.0.text").String())
}

func TestOpenAIGatewayForwardRetriesRejectedInputNamespaceForStreaming(t *testing.T) {
	body := namespaceCompatibilityStreamingRequestBody()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{responses: namespaceCompatibilityStreamingResponses()}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}
	account := namespaceCompatibilityAccount(false)

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Len(t, upstream.bodies, 2)
	require.Equal(t, "mcp__docs", gjson.GetBytes(upstream.bodies[0], "input.0.namespace").String())
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.namespace").Exists())
	require.Contains(t, rec.Body.String(), `"type":"response.output_text.delta"`)
	require.Contains(t, rec.Body.String(), `"type":"response.completed"`)
	require.NotContains(t, rec.Body.String(), "Upstream request failed")
}

func TestOpenAIGatewayPassthroughRetriesRejectedInputNamespaceForStreaming(t *testing.T) {
	body := namespaceCompatibilityStreamingRequestBody()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{responses: namespaceCompatibilityStreamingResponses()}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}
	account := namespaceCompatibilityAccount(true)

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Len(t, upstream.bodies, 2)
	require.Equal(t, "mcp__docs", gjson.GetBytes(upstream.bodies[0], "input.0.namespace").String())
	require.False(t, gjson.GetBytes(upstream.bodies[1], "input.0.namespace").Exists())
	require.Contains(t, rec.Body.String(), `"type":"response.output_text.delta"`)
	require.Contains(t, rec.Body.String(), `"type":"response.completed"`)
	require.NotContains(t, rec.Body.String(), "Upstream request failed")
}

func namespaceCompatibilityRequestBody() []byte {
	return []byte(`{
		"model":"gpt-5.6-sol",
		"stream":false,
		"instructions":"continue the tool session",
		"input":[
			{"type":"function_call","namespace":"mcp__docs","name":"lookup","call_id":"call_1","arguments":"{}"},
			{"type":"function_call_output","call_id":"call_1","output":"ok"}
		],
		"tools":[{"type":"namespace","name":"mcp__docs","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}]
	}`)
}

func namespaceCompatibilityStreamingRequestBody() []byte {
	return bytes.Replace(namespaceCompatibilityRequestBody(), []byte(`"stream":false`), []byte(`"stream":true`), 1)
}

func namespaceCompatibilityResponses() []*http.Response {
	return []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[18].namespace'.","param":"input[18].namespace","type":"invalid_request_error"}}`,
			)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid_namespace_retry_ok"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"resp_namespace_retry_ok","object":"response","model":"gpt-5.6-sol","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}],"status":"completed"}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}`,
			)),
		},
	}
}

func namespaceCompatibilityStreamingResponses() []*http.Response {
	return []*http.Response{
		{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[18].namespace'.","param":"input[18].namespace","type":"invalid_request_error"}}`,
			)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid_namespace_stream_retry_ok"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`data: {"type":"response.created","response":{"id":"resp_namespace_stream_retry_ok","model":"gpt-5.6-sol","status":"in_progress","output":[]}}`,
				"",
				`data: {"type":"response.output_text.delta","delta":"ok"}`,
				"",
				`data: {"type":"response.completed","response":{"id":"resp_namespace_stream_retry_ok","object":"response","model":"gpt-5.6-sol","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}],"status":"completed"}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}}`,
				"",
			}, "\n"))),
		},
	}
}

func namespaceCompatibilityAccount(passthrough bool) *Account {
	account := rawChatCompletionsTestAccount()
	account.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesMode:      string(openai_compat.ResponsesSupportModeAuto),
		openai_compat.ExtraKeyResponsesSupported: true,
	}
	if passthrough {
		account.Extra["openai_passthrough"] = true
	}
	return account
}
