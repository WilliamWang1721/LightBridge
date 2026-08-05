//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProtocolResponseBridgeStreamsGeminiIncrementally(t *testing.T) {
	recorder := httptest.NewRecorder()
	target, _ := gin.CreateTestContext(recorder)
	bridge := NewProtocolResponseBridge(target, AnthropicBridgeTargetGemini, true, "gemini-2.5-flash")

	_, err := bridge.Write([]byte(strings.Join([]string{
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}`,
	}, "\n") + "\n\n"))
	require.NoError(t, err)
	require.Contains(t, recorder.Body.String(), `"text":"hello"`)
	require.NoError(t, bridge.Finalize())
}

func TestProtocolBridgeContextPreservesCancellation(t *testing.T) {
	recorder := httptest.NewRecorder()
	target, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(context.Background())
	bridge := NewProtocolResponseBridge(target, AnthropicBridgeTargetResponses, true, "gpt-test")
	capture, err := NewProtocolBridgeContext(target, ctx, "/v1/messages", []byte(`{}`), bridge)
	require.NoError(t, err)
	cancel()
	require.ErrorIs(t, capture.Request.Context().Err(), context.Canceled)
}

func TestProtocolBridgeContextCopiesRequestMetadataWithoutTestContext(t *testing.T) {
	recorder := httptest.NewRecorder()
	target, _ := gin.CreateTestContext(recorder)
	target.Set("tenant", "tenant-a")
	target.Params = gin.Params{{Key: "id", Value: "42"}}
	request := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{}`))
	request.RemoteAddr = "198.51.100.10:12345"
	target.Request = request

	bridge := NewProtocolResponseBridge(target, AnthropicBridgeTargetResponses, false, "gpt-test")
	capture, err := NewProtocolBridgeContext(target, request.Context(), "/v1/messages", []byte(`{}`), bridge)
	require.NoError(t, err)
	require.Same(t, bridge, capture.Writer)
	require.Equal(t, "tenant-a", capture.MustGet("tenant"))
	require.Equal(t, "42", capture.Param("id"))
	require.Equal(t, request.RemoteAddr, capture.Request.RemoteAddr)
}

func TestProtocolResponseBridgePassesThroughEmptyNoContentResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	target, _ := gin.CreateTestContext(recorder)
	bridge := NewProtocolResponseBridge(target, AnthropicBridgeTargetResponses, false, "gpt-test")

	bridge.WriteHeader(http.StatusNoContent)
	bridge.WriteHeaderNow()
	require.NoError(t, bridge.Finalize())
	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Body.String())
}
