package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fullPassthroughUpstreamStub struct {
	req  *http.Request
	body string
}

func (s *fullPassthroughUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return s.DoWithTLS(req, proxyURL, accountID, accountConcurrency, nil)
}

func (s *fullPassthroughUpstreamStub) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	s.req = req
	body, _ := io.ReadAll(req.Body)
	s.body = string(body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_raw"},
		},
		Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","usage":{"input_tokens":7,"output_tokens":11}}`)),
	}, nil
}

func TestForwardFullPassthroughPreservesWireRequestAndRewritesAuth(t *testing.T) {
	upstream := &fullPassthroughUpstreamStub{}
	svc := &GatewayService{
		httpUpstream: upstream,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true},
			},
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages?beta=true", strings.NewReader(`{"model":"claude","stream":false}`))
	c.Request.Header.Set("Authorization", "Bearer client")
	c.Request.Header.Set("X-API-Key", "client-key")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Custom", "kept")

	account := &Account{
		ID:       12,
		Name:     "raw",
		Platform: PlatformCustom,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "http://upstream.test/v1",
		},
		Extra: map[string]any{
			"protocol":   CustomProtocolOpenAIResponses,
			"relay_mode": RelayModeFullPassthrough,
		},
	}

	result, err := svc.ForwardFullPassthrough(context.Background(), c, account, FullPassthroughInput{
		Body:          []byte(`{"model":"claude","stream":false}`),
		Token:         "sk-upstream",
		TokenType:     "apikey",
		RequestModel:  "claude",
		RequestStream: false,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "req_raw", result.RequestID)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 11, result.Usage.OutputTokens)

	require.Equal(t, http.MethodPost, upstream.req.Method)
	require.Equal(t, "http://upstream.test/v1/messages?beta=true", upstream.req.URL.String())
	require.Equal(t, `{"model":"claude","stream":false}`, upstream.body)
	require.Equal(t, "Bearer sk-upstream", upstream.req.Header.Get("Authorization"))
	require.Empty(t, upstream.req.Header.Get("X-API-Key"))
	require.Equal(t, "kept", upstream.req.Header.Get("X-Custom"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "req_raw", rec.Header().Get("x-request-id"))
	require.JSONEq(t, `{"id":"resp_1","usage":{"input_tokens":7,"output_tokens":11}}`, rec.Body.String())
}

func TestAppendFullPassthroughPathAvoidsDuplicateVersionPrefix(t *testing.T) {
	got, err := appendFullPassthroughPath("https://api.example.com/v1", "/v1/chat/completions", "stream=true")
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/v1/chat/completions?stream=true", got)

	got, err = appendFullPassthroughPath("https://api.example.com/v1", "/messages", "")
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/v1/messages", got)
}
