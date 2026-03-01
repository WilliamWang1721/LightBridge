package main

import (
	"encoding/base32"
	"testing"
	"time"
)

func testTOTPCode(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	key, err := enc.DecodeString(secret)
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}
	step := (at.Unix() + totpSkewSec) / totpPeriod
	code, err := hotp(key, step, totpDigits)
	if err != nil {
		t.Fatalf("hotp: %v", err)
	}
	return code
}

func TestCleanCode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "123456", want: "123456"},
		{in: "123 456", want: "123456"},
		{in: "12-34-56", want: "123456"},
		{in: "12345", want: ""},
		{in: "1234567", want: ""},
		{in: "abc123", want: ""},
	}
	for _, tt := range tests {
		if got := cleanCode(tt.in); got != tt.want {
			t.Fatalf("cleanCode(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestVerifyTOTPAcceptsCurrentWindow(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Unix(1700000000, 0).UTC()
	code := testTOTPCode(t, secret, now)

	ok, _ := verifyTOTP(secret, code, now)
	if !ok {
		t.Fatal("expected current step code to be accepted")
	}

	prevCode := testTOTPCode(t, secret, now.Add(-30*time.Second))
	ok, _ = verifyTOTP(secret, prevCode, now)
	if !ok {
		t.Fatal("expected previous window code to be accepted")
	}

	nextCode := testTOTPCode(t, secret, now.Add(30*time.Second))
	ok, _ = verifyTOTP(secret, nextCode, now)
	if !ok {
		t.Fatal("expected next window code to be accepted")
	}

	farCode := testTOTPCode(t, secret, now.Add(2*time.Minute))
	ok, _ = verifyTOTP(secret, farCode, now)
	if ok {
		t.Fatal("expected far-window code to be rejected")
	}
}
