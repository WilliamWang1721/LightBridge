package main

import (
	"strings"
	"testing"
)

func TestParseCallbackURLInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		wantCode  string
		wantState string
		errSubstr string
	}{
		{
			name:      "direct callback url",
			input:     "http://localhost:1455/auth/callback?code=abc&state=xyz",
			wantCode:  "abc",
			wantState: "xyz",
		},
		{
			name:      "html escaped amp",
			input:     "\"http://localhost:1455/auth/callback?code=abc&amp;state=xyz\"",
			wantCode:  "abc",
			wantState: "xyz",
		},
		{
			name:      "url in free text",
			input:     "登录完成后链接： http://localhost:1455/auth/callback?code=c2&state=s2",
			wantCode:  "c2",
			wantState: "s2",
		},
		{
			name:      "missing scheme localhost",
			input:     "localhost:1455/auth/callback?code=c3&state=s3",
			wantCode:  "c3",
			wantState: "s3",
		},
		{
			name:      "percent encoded callback url",
			input:     "https%3A%2F%2Flocalhost%3A1455%2Fauth%2Fcallback%3Fcode%3Dc4%26state%3Ds4",
			wantCode:  "c4",
			wantState: "s4",
		},
		{
			name:      "nested callback_url",
			input:     "https://example.com/callback?callback_url=http%3A%2F%2Flocalhost%3A1455%2Fauth%2Fcallback%3Fcode%3Dc5%26state%3Ds5",
			wantCode:  "c5",
			wantState: "s5",
		},
		{
			name:      "oauth error in callback url",
			input:     "http://localhost:1455/auth/callback?error=access_denied&error_description=user%20cancelled",
			errSubstr: "access_denied: user cancelled",
		},
		{
			name:      "invalid callback input",
			input:     "not-a-url",
			errSubstr: "invalid callback_url",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := parseCallbackURLInput(tc.input)
			if tc.errSubstr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errSubstr)
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("expected error containing %q, got %q", tc.errSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsed == nil {
				t.Fatalf("expected parsed result, got nil")
			}
			if parsed.Code != tc.wantCode {
				t.Fatalf("expected code %q, got %q", tc.wantCode, parsed.Code)
			}
			if parsed.State != tc.wantState {
				t.Fatalf("expected state %q, got %q", tc.wantState, parsed.State)
			}
			if strings.TrimSpace(parsed.NormalizedURL) == "" {
				t.Fatalf("expected normalized url")
			}
		})
	}
}
