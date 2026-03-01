package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	totpPeriod  = int64(30)
	totpDigits  = 6
	totpWindow  = int64(1)
	totpSkewSec = int64(5)
)

func verifyTOTP(secret, code string, now time.Time) (bool, int64) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	code = cleanCode(code)
	if secret == "" || code == "" {
		return false, 0
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	key, err := enc.DecodeString(secret)
	if err != nil || len(key) == 0 {
		return false, 0
	}

	nowStep := (now.Unix() + totpSkewSec) / totpPeriod
	for offset := -totpWindow; offset <= totpWindow; offset++ {
		step := nowStep + offset
		want, err := hotp(key, step, totpDigits)
		if err != nil {
			continue
		}
		if want == code {
			return true, step
		}
	}
	return false, 0
}

func hotp(key []byte, counter int64, digits int) (string, error) {
	if len(key) == 0 {
		return "", fmt.Errorf("empty key")
	}
	if digits <= 0 || digits > 10 {
		return "", fmt.Errorf("invalid digits")
	}
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, uint64(counter))

	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binaryCode :=
		(uint32(sum[offset])&0x7f)<<24 |
			uint32(sum[offset+1])<<16 |
			uint32(sum[offset+2])<<8 |
			uint32(sum[offset+3])

	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	otp := binaryCode % mod
	return fmt.Sprintf("%0*d", digits, otp), nil
}
