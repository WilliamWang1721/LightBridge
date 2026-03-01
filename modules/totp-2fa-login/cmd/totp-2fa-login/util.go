package main

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
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

func newTOTPSecret() (string, error) {
	raw, err := randomBytes(20)
	if err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return strings.ToUpper(enc.EncodeToString(raw)), nil
}

func buildOTPAuthURI(issuer, accountName, secret string) string {
	issuer = strings.TrimSpace(issuer)
	accountName = strings.TrimSpace(accountName)
	secret = strings.TrimSpace(secret)
	if accountName == "" {
		accountName = "admin"
	}
	label := accountName
	if issuer != "" {
		label = issuer + ":" + accountName
	}
	q := url.Values{}
	q.Set("secret", secret)
	if issuer != "" {
		q.Set("issuer", issuer)
	}
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	return fmt.Sprintf("otpauth://totp/%s?%s", url.PathEscape(label), q.Encode())
}

func cleanCode(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	buf := make([]byte, 0, 6)
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c >= '0' && c <= '9' {
			buf = append(buf, c)
		}
	}
	if len(buf) != 6 {
		return ""
	}
	return string(buf)
}
