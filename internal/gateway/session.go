package gateway

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"lightbridge/internal/util"
)

const adminSessionCookie = "lightbridge_admin"

type sessionManager struct {
	secret []byte
	mu     sync.RWMutex
}

func newSessionManager(secret string) *sessionManager {
	return &sessionManager{
		secret: []byte(secret),
	}
}

func (s *sessionManager) newSession(w http.ResponseWriter, username string, remember bool) error {
	nonce, err := util.RandomToken(16)
	if err != nil {
		return err
	}
	now := time.Now()
	ttl := 24 * time.Hour
	if remember {
		ttl = 30 * 24 * time.Hour
	}
	expires := now.Add(ttl)

	// Cookie format: base64url(payload) + "." + base64url(hmac(payload))
	// payload = "v2|<username>|<expiresUnix>|<nonce>"
	payload := fmt.Sprintf("v2|%s|%d|%s", username, expires.Unix(), nonce)
	sig := s.sign(payload)
	payloadEnc := base64.RawURLEncoding.EncodeToString([]byte(payload))
	cookie := &http.Cookie{
		Name:     adminSessionCookie,
		Value:    payloadEnc + "." + sig,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	// Only persist across browser restarts if user selected "remember device".
	if remember {
		cookie.Expires = expires
		cookie.MaxAge = int(ttl.Seconds())
	}
	http.SetCookie(w, cookie)
	return nil
}

func (s *sessionManager) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (s *sessionManager) username(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(adminSessionCookie)
	if err != nil {
		return "", false
	}
	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	payloadEnc, sig := parts[0], parts[1]
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadEnc)
	if err != nil {
		return "", false
	}
	payload := string(payloadBytes)
	if !hmac.Equal([]byte(s.sign(payload)), []byte(sig)) {
		return "", false
	}
	payloadParts := strings.Split(payload, "|")
	if len(payloadParts) != 4 || payloadParts[0] != "v2" {
		return "", false
	}
	username := payloadParts[1]
	expUnix, err := strconv.ParseInt(payloadParts[2], 10, 64)
	if err != nil {
		return "", false
	}
	if username == "" || time.Now().Unix() > expUnix {
		return "", false
	}
	return username, true
}

func (s *sessionManager) sign(token string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
