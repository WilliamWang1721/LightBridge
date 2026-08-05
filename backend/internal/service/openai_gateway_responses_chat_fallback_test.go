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

	"github.com/WilliamWang1721/LightBridge/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardResponses_ForceChatCompletionsRoutesNonStreamingToChatCompletions(t *testing.T) {

	body := []byte(`{"model":"gpt-5.4","input":"hello","stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid_resp_chat_json"}},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"chatcmpl_json","object":"chat.completion","model":"gpt-5.4","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5,"prompt_tokens_details":{"cached_tokens":1}}}`,
		)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}

	result, err := svc.Forward(context.Background(), c, forceChatResponsesFallbackAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://upstream.example/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileFromContext(upstream.lastReq.Context()))
	require.Equal(t, "hello", gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "input").Exists())
	require.Equal(t, "response", gjson.Get(rec.Body.String(), "object").String())
	require.Equal(t, "ok", gjson.Get(rec.Body.String(), "output.0.content.0.text").String())
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
	require.Equal(t, 1, result.Usage.CacheReadInputTokens)
	require.False(t, result.Stream)
}

func TestForwardResponses_ForceChatCompletionsRoutesStreamingToChatCompletions(t *testing.T) {

	body := []byte(`{"model":"gpt-5.4","input":"hello","stream":true}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := strings.Join([]string{
		`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","model":"gpt-5.4","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","model":"gpt-5.4","choices":[{"index":0,"delta":{"content":"he"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","model":"gpt-5.4","choices":[{"index":0,"delta":{"content":"llo"},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","model":"gpt-5.4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		"",
		`data: {"id":"chatcmpl_stream","object":"chat.completion.chunk","model":"gpt-5.4","choices":[],"usage":{"prompt_tokens":4,"completion_tokens":3,"total_tokens":7}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid_resp_chat_stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}

	result, err := svc.Forward(context.Background(), c, forceChatResponsesFallbackAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://upstream.example/v1/chat/completions", upstream.lastReq.URL.String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "stream_options.include_usage").Bool())
	require.Contains(t, rec.Body.String(), "event: response.output_text.delta")
	require.Contains(t, rec.Body.String(), `"delta":"he"`)
	require.Contains(t, rec.Body.String(), "event: response.completed")
	require.Contains(t, rec.Body.String(), `"input_tokens":4`)
	require.Contains(t, rec.Body.String(), "data: [DONE]")
	require.Equal(t, 4, result.Usage.InputTokens)
	require.Equal(t, 3, result.Usage.OutputTokens)
	require.True(t, result.Stream)
	require.NotNil(t, result.FirstTokenMs)
}

func TestForwardResponses_AutoSupportedAccountStillUsesResponsesEndpoint(t *testing.T) {

	body := []byte(`{"model":"gpt-5.4","input":"hello","stream":false}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid_resp_native"}},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"resp_native","object":"response","model":"gpt-5.4","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}],"status":"completed"}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}`,
		)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}
	account := rawChatCompletionsTestAccount()
	account.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesMode:      string(openai_compat.ResponsesSupportModeAuto),
		openai_compat.ExtraKeyResponsesSupported: true,
	}

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://upstream.example/v1/responses", upstream.lastReq.URL.String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "input").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, "messages").Exists())
	require.Equal(t, "ok", gjson.Get(rec.Body.String(), "output.0.content.0.text").String())
}

func TestForwardResponses_GrokBuildUsesStrictChatBridgeForLiteLLMToolCalls(t *testing.T) {

	body := []byte(`{
		"model":"grok-4-fast",
		"input":"list the current directory",
		"stream":true,
		"parallel_tool_calls":true,
		"tools":[{"type":"function","name":"list_dir","description":"List a directory","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]},"strict":true}]
	}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("User-Agent", "grok-shell/0.2.101 (linux; x86_64)")

	upstreamBody := strings.Join([]string{
		`data: {"id":"chatcmpl_tool","object":"chat.completion.chunk","model":"grok-4-fast","choices":[{"index":0,"delta":{"tool_calls":[{"id":"call_list","type":"function","function":{"name":"list","arguments":"{\"path\""}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl_tool","object":"chat.completion.chunk","model":"grok-4-fast","choices":[{"index":0,"delta":{"tool_calls":[{"id":"call_list","type":"function","function":{"name":"_dir","arguments":"{\"path\":\".\"}"}}]},"finish_reason":""}]}`,
		"",
		`data: {"id":"chatcmpl_tool","object":"chat.completion.chunk","model":"grok-4-fast","choices":[],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid_grok_tool"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          rawChatCompletionsTestConfig(),
		httpUpstream: upstream,
	}
	account := rawChatCompletionsTestAccount()
	account.Credentials["base_url"] = "http://litellm.example/v1"
	account.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesMode:      string(openai_compat.ResponsesSupportModeAuto),
		openai_compat.ExtraKeyResponsesSupported: true,
	}
	ctx := WithRouterClientProfile(context.Background(), DetectRouterClientProfile(c.Request))

	result, err := svc.Forward(ctx, c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "http://litellm.example/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "list_dir", gjson.GetBytes(upstream.lastBody, "tools.0.function.name").String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "tools.0.function.strict").Bool())
	require.True(t, gjson.GetBytes(upstream.lastBody, "parallel_tool_calls").Bool())

	streamBody := rec.Body.String()
	require.Contains(t, streamBody, `"type":"function_call"`)
	require.Contains(t, streamBody, `"call_id":"call_list"`)
	require.Contains(t, streamBody, `"name":"list_dir"`)
	require.Contains(t, streamBody, `"arguments":"{\"path\":\".\"}"`)
	require.NotContains(t, streamBody, `"name":"list"`)
	require.Contains(t, streamBody, "event: response.completed")
	require.Contains(t, streamBody, "data: [DONE]")
	for _, line := range strings.Split(streamBody, "\n") {
		payload, ok := strings.CutPrefix(line, "data: ")
		if !ok || payload == "[DONE]" || !gjson.Valid(payload) {
			continue
		}
		require.True(t, gjson.Get(payload, "sequence_number").Exists(), "event is missing sequence_number: %s", payload)
	}
	require.Equal(t, 20, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
}

func TestForwardResponses_GrokBuildRestoresNamespaceAndCustomToolCalls(t *testing.T) {

	body := []byte(`{
		"model":"grok-4-fast",
		"input":"listen and run",
		"stream":true,
		"tools":[
			{"type":"namespace","name":"dictionary","tools":[{"type":"function","name":"listen_dictionary","parameters":{"type":"object","properties":{"word":{"type":"string"}}}}]},
			{"type":"custom","name":"shell","description":"Run a shell command"}
		]
	}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("User-Agent", "grok-shell/0.2.101 (linux; x86_64)")

	upstreamBody := strings.Join([]string{
		`data: {"id":"chatcmpl_tools","object":"chat.completion.chunk","model":"grok-4-fast","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_dict","type":"function","function":{"name":"dictionary__listen_","arguments":"{\"word\":"}},{"index":1,"id":"call_shell","type":"function","function":{"name":"shell","arguments":"{\"input\":\"pw"}}]},"finish_reason":null}]}`,
		"",
		`data: {"id":"chatcmpl_tools","object":"chat.completion.chunk","model":"grok-4-fast","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"name":"dictionary","arguments":"\"test\"}"}},{"index":1,"function":{"arguments":"d\"}"}}]},"finish_reason":"tool_calls"}]}`,
		"",
		`data: {"id":"chatcmpl_tools","object":"chat.completion.chunk","model":"grok-4-fast","choices":[],"usage":{"prompt_tokens":30,"completion_tokens":8,"total_tokens":38}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid_grok_strict_tools"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
	account := rawChatCompletionsTestAccount()
	account.Credentials["base_url"] = "http://litellm.example/v1"
	account.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesMode:      string(openai_compat.ResponsesSupportModeAuto),
		openai_compat.ExtraKeyResponsesSupported: true,
	}
	ctx := WithRouterClientProfile(context.Background(), DetectRouterClientProfile(c.Request))

	result, err := svc.Forward(ctx, c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "dictionary__listen_dictionary", gjson.GetBytes(upstream.lastBody, "tools.0.function.name").String())
	require.Equal(t, "shell", gjson.GetBytes(upstream.lastBody, "tools.1.function.name").String())
	require.Equal(t, "string", gjson.GetBytes(upstream.lastBody, "tools.1.function.parameters.properties.input.type").String())

	var payloads []gjson.Result
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		payload, ok := strings.CutPrefix(line, "data: ")
		if ok && payload != "[DONE]" && gjson.Valid(payload) {
			payloads = append(payloads, gjson.Parse(payload))
		}
	}
	var dictionaryAdded, customAdded, customDone, completed gjson.Result
	for _, payload := range payloads {
		switch payload.Get("type").String() {
		case "response.output_item.added":
			switch payload.Get("item.call_id").String() {
			case "call_dict":
				dictionaryAdded = payload
			case "call_shell":
				customAdded = payload
			}
		case "response.custom_tool_call_input.done":
			customDone = payload
		case "response.completed":
			completed = payload
		}
	}
	require.Equal(t, "function_call", dictionaryAdded.Get("item.type").String())
	require.Equal(t, "listen_dictionary", dictionaryAdded.Get("item.name").String())
	require.Equal(t, "dictionary", dictionaryAdded.Get("item.namespace").String())
	require.Equal(t, "custom_tool_call", customAdded.Get("item.type").String())
	require.Equal(t, "shell", customAdded.Get("item.name").String())
	require.Equal(t, "pwd", customDone.Get("input").String())
	require.Equal(t, "listen_dictionary", completed.Get("response.output.0.name").String())
	require.Equal(t, "dictionary", completed.Get("response.output.0.namespace").String())
	require.Equal(t, "custom_tool_call", completed.Get("response.output.1.type").String())
	require.Equal(t, "pwd", completed.Get("response.output.1.input").String())
	require.Equal(t, 30, result.Usage.InputTokens)
	require.Equal(t, 8, result.Usage.OutputTokens)
}

func TestShouldBridgeResponsesThroughChatCompletions_HonorsRouteAndManualOverride(t *testing.T) {
	autoSupported := rawChatCompletionsTestAccount()
	autoSupported.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesMode:      string(openai_compat.ResponsesSupportModeAuto),
		openai_compat.ExtraKeyResponsesSupported: true,
	}
	routeCtx := WithProtocolRouteDecision(context.Background(), ProtocolRouteDecision{
		TargetProtocol: CustomProtocolOpenAIChatCompletions,
	})
	require.True(t, shouldBridgeResponsesThroughChatCompletions(routeCtx, nil, autoSupported))

	grokCtx := WithRouterClientProfile(context.Background(), RouterClientProfile{
		Kind:                    RouterClientGrokBuild,
		StrictResponsesTerminal: true,
	})
	require.True(t, shouldBridgeResponsesThroughChatCompletions(grokCtx, nil, autoSupported))

	forceResponses := rawChatCompletionsTestAccount()
	forceResponses.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesMode: string(openai_compat.ResponsesSupportModeForceResponses),
	}
	require.False(t, shouldBridgeResponsesThroughChatCompletions(grokCtx, nil, forceResponses))
}

func forceChatResponsesFallbackAccount() *Account {
	account := rawChatCompletionsTestAccount()
	account.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesMode: string(openai_compat.ResponsesSupportModeForceChatCompletions),
	}
	return account
}
