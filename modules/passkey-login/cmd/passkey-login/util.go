package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

var b64 = base64.RawURLEncoding

func randomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, errors.New("invalid random length")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func base64urlEncode(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return b64.EncodeToString(b)
}

func base64urlDecode(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty base64url")
	}
	return b64.DecodeString(s)
}

func userHandleForUsername(username string) []byte {
	sum := sha256.Sum256([]byte("lightbridge-admin:" + strings.TrimSpace(username)))
	return sum[:]
}

