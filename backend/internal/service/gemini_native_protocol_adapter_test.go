package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiGenerateContentToAnthropicMessages(t *testing.T) {
	body := []byte(`{
	  "systemInstruction": {"parts": [{"text": "be concise"}]},
	  "contents": [
	    {"role": "user", "parts": [
	      {"text": "hello"},
	      {"inlineData": {"mimeType": "image/png", "data": "abcd"}}
	    ]},
	    {"role": "model", "parts": [
	      {"functionCall": {"name": "lookup", "args": {"q": "x"}, "id": "call_1"}}
	    ]},
	    {"role": "user", "parts": [
	      {"functionResponse": {"name": "lookup", "response": {"ok": true}, "id": "call_1"}}
	    ]}
	  ],
	  "tools": [{"functionDeclarations": [{"name": "lookup", "description": "Lookup data", "parameters": {"type": "object"}}]}],
	  "generationConfig": {
	    "temperature": 0.2,
	    "topP": 0.9,
	    "maxOutputTokens": 123,
	    "stopSequences": ["END"],
	    "thinkingConfig": {"includeThoughts": true, "thinkingBudget": 64}
	  }
	}`)

	out, err := GeminiGenerateContentToAnthropicMessages(body, "gemini-2.5-pro", true)
	require.NoError(t, err)

	var req apicompat.AnthropicRequest
	require.NoError(t, json.Unmarshal(out, &req))
	require.Equal(t, "gemini-2.5-pro", req.Model)
	require.True(t, req.Stream)
	require.Equal(t, 123, req.MaxTokens)
	require.Equal(t, []string{"END"}, req.StopSeqs)
	require.NotNil(t, req.Thinking)
	require.Equal(t, "enabled", req.Thinking.Type)
	require.Equal(t, 64, req.Thinking.BudgetTokens)
	require.Len(t, req.Tools, 1)
	require.Equal(t, "lookup", req.Tools[0].Name)
	require.Len(t, req.Messages, 3)
	require.Equal(t, "user", req.Messages[0].Role)
	require.Equal(t, "assistant", req.Messages[1].Role)

	var firstBlocks []apicompat.AnthropicContentBlock
	require.NoError(t, json.Unmarshal(req.Messages[0].Content, &firstBlocks))
	require.Equal(t, "text", firstBlocks[0].Type)
	require.Equal(t, "hello", firstBlocks[0].Text)
	require.Equal(t, "image", firstBlocks[1].Type)
	require.Equal(t, "image/png", firstBlocks[1].Source.MediaType)

	var toolBlocks []apicompat.AnthropicContentBlock
	require.NoError(t, json.Unmarshal(req.Messages[1].Content, &toolBlocks))
	require.Equal(t, "tool_use", toolBlocks[0].Type)
	require.Equal(t, "call_1", toolBlocks[0].ID)
	require.JSONEq(t, `{"q":"x"}`, string(toolBlocks[0].Input))
}

func TestWriteCapturedAnthropicAsGeminiJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{
	  "id": "msg_1",
	  "type": "message",
	  "role": "assistant",
	  "model": "claude",
	  "content": [
	    {"type": "text", "text": "hi"},
	    {"type": "tool_use", "id": "toolu_1", "name": "lookup", "input": {"q": "x"}}
	  ],
	  "stop_reason": "end_turn",
	  "usage": {"input_tokens": 3, "output_tokens": 5}
	}`)

	headers := http.Header{}
	headers.Set("x-request-id", "req_1")
	err := WriteCapturedAnthropicAsGemini(c, http.StatusOK, headers, body, false, "gemini-2.5-pro")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "req_1", rec.Header().Get("x-request-id"))

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "gemini-2.5-pro", resp["modelVersion"])
	require.Equal(t, "msg_1", resp["responseId"])
	usageMetadata, ok := resp["usageMetadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(8), usageMetadata["totalTokenCount"])
	candidates := resp["candidates"].([]any)
	parts := candidates[0].(map[string]any)["content"].(map[string]any)["parts"].([]any)
	require.Equal(t, "hi", parts[0].(map[string]any)["text"])
	require.Equal(t, "lookup", parts[1].(map[string]any)["functionCall"].(map[string]any)["name"])
}

func TestWriteCapturedAnthropicAsGeminiStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	stream := strings.Join([]string{
		`event: content_block_start`,
		`data: {"type":"content_block_start","content_block":{"type":"text","text":"he"}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"llo"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":2,"output_tokens":3}}`,
		``,
	}, "\n")

	err := WriteCapturedAnthropicAsGemini(c, http.StatusOK, http.Header{}, []byte(stream), true, "gemini-2.5-flash")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), `"text":"he"`)
	require.Contains(t, rec.Body.String(), `"text":"llo"`)
	require.Contains(t, rec.Body.String(), `"finishReason":"STOP"`)
	require.Contains(t, rec.Body.String(), `"totalTokenCount":5`)
}
