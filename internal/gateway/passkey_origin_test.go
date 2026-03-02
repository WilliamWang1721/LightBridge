package gateway

import (
	"net/http/httptest"
	"testing"
)

func TestWebAuthnOriginFromRequest_PrefersOriginHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "http://internal.local/admin/api/passkey/auth/begin", nil)
	req.Host = "127.0.0.1:3210"
	req.Header.Set("Origin", "HTTPS://admin.example.com:8443")

	got := webAuthnOriginFromRequest(req)
	if got != "https://admin.example.com:8443" {
		t.Fatalf("origin mismatch: got %q", got)
	}

	rpID := rpIDFromOrigin(got)
	if rpID != "admin.example.com" {
		t.Fatalf("rpId mismatch: got %q", rpID)
	}
}

func TestWebAuthnOriginFromRequest_FallbackToRequestHost(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.com/admin/api/passkey/auth/begin", nil)
	req.Header.Set("Origin", "null")

	got := webAuthnOriginFromRequest(req)
	if got != "http://example.com" {
		t.Fatalf("origin mismatch: got %q", got)
	}
}
