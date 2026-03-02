package main

import "testing"

func TestPKCEChallengeDeterministic(t *testing.T) {
	verifier := "test-verifier-1234567890"
	challenge := sha256Base64URL(verifier)
	if challenge == "" {
		t.Fatalf("expected non-empty challenge")
	}
	if got := sha256Base64URL(verifier); got != challenge {
		t.Fatalf("challenge not deterministic: %s vs %s", challenge, got)
	}
}

func TestParseCallbackURL(t *testing.T) {
	cases := []struct {
		in          string
		code        string
		state       string
		loginOption string
		path        string
		err         bool
	}{
		{"http://127.0.0.1:19876/oauth/callback?code=abc&state=xyz", "abc", "xyz", "", "/oauth/callback", false},
		{"\"http://127.0.0.1:19876/oauth/callback?code=abc&amp;state=xyz\"", "abc", "xyz", "", "/oauth/callback", false},
		{"https%3A%2F%2F127.0.0.1%3A19876%2Foauth%2Fcallback%3Fcode%3Dc1%26state%3Ds1", "c1", "s1", "", "/oauth/callback", false},
		{"http://localhost:3128/signin/callback?code=abc&state=xyz&login_option=google", "abc", "xyz", "google", "/signin/callback", false},
		{"http://127.0.0.1:19876/oauth/callback?error=access_denied", "", "", "", "", true},
	}
	for _, tc := range cases {
		_, code, state, loginOption, path, errMsg := parseCallbackURL(tc.in)
		if tc.err {
			if errMsg == "" {
				t.Fatalf("expected error for %q", tc.in)
			}
			continue
		}
		if errMsg != "" {
			t.Fatalf("unexpected error for %q: %s", tc.in, errMsg)
		}
		if code != tc.code || state != tc.state {
			t.Fatalf("parse mismatch for %q: got code=%q state=%q", tc.in, code, state)
		}
		if loginOption != tc.loginOption || path != tc.path {
			t.Fatalf("parse mismatch for %q: got login_option=%q path=%q", tc.in, loginOption, path)
		}
	}
}
