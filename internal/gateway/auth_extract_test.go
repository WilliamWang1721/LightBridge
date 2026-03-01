package gateway

import (
	"net/http/httptest"
	"testing"
)

func TestClientTokenFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/v1/models", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	if got := clientTokenFromRequest(req); got != "bearer-token" {
		t.Fatalf("bearer token mismatch: %q", got)
	}

	req = httptest.NewRequest("GET", "http://example.com/v1/models", nil)
	req.Header.Set("x-goog-api-key", "goog-token")
	if got := clientTokenFromRequest(req); got != "goog-token" {
		t.Fatalf("x-goog-api-key mismatch: %q", got)
	}

	req = httptest.NewRequest("GET", "http://example.com/v1/models", nil)
	req.Header.Set("x-api-key", "x-token")
	req.Header.Set("Authorization", "Bearer ignored")
	if got := clientTokenFromRequest(req); got != "x-token" {
		t.Fatalf("x-api-key precedence mismatch: %q", got)
	}

	req = httptest.NewRequest("GET", "http://example.com/v1/models?key=query-token", nil)
	if got := clientTokenFromRequest(req); got != "query-token" {
		t.Fatalf("query key mismatch: %q", got)
	}
}
