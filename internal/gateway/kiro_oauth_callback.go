package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	kiroOAuthLocalPortStart = 19876
	kiroOAuthLocalPortEnd   = 19880
)

func (s *Server) kiroOAuthLocalRedirectURI() string {
	s.kiroOAuthCallbackMu.Lock()
	port := s.kiroOAuthCallbackPort
	s.kiroOAuthCallbackMu.Unlock()
	if port <= 0 {
		port = kiroOAuthLocalPortStart
	}
	return fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", port)
}

func (s *Server) ensureKiroOAuthCallbackServer() {
	s.kiroOAuthCallbackMu.Lock()
	defer s.kiroOAuthCallbackMu.Unlock()

	if s.kiroOAuthCallbackStarted {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", s.handleKiroOAuthLocalCallback)
	handler := requestIDMiddleware(loggingMiddleware(mux))

	startOnPort := func(port int) (net.Listener, error) {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
		srv := &http.Server{
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("kiro oauth callback server (%s) error: %v", addr, err)
			}
		}()
		return ln, nil
	}

	var problems []string
	for port := kiroOAuthLocalPortStart; port <= kiroOAuthLocalPortEnd; port++ {
		ln, err := startOnPort(port)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%d: %v", port, err))
			continue
		}
		s.kiroOAuthCallbackStarted = true
		s.kiroOAuthCallbackErr = nil
		s.kiroOAuthCallbackPort = port
		_ = ln
		return
	}

	s.kiroOAuthCallbackErr = fmt.Errorf("failed to start local callback server on 127.0.0.1:%d-%d (%s)", kiroOAuthLocalPortStart, kiroOAuthLocalPortEnd, strings.Join(problems, "; "))
}

func (s *Server) handleKiroOAuthLocalCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errStr := strings.TrimSpace(q.Get("error")); errStr != "" {
		desc := strings.TrimSpace(q.Get("error_description"))
		msg := errStr
		if desc != "" {
			msg += ": " + desc
		}
		s.renderKiroOAuthCallbackResultTo(w, false, msg, s.localAdminProvidersURL())
		return
	}

	code := strings.TrimSpace(q.Get("code"))
	state := strings.TrimSpace(q.Get("state"))
	if code == "" || state == "" {
		s.renderKiroOAuthCallbackResultTo(w, false, "missing code/state in callback url", s.localAdminProvidersURL())
		return
	}

	payload, _ := json.Marshal(map[string]string{"code": code, "state": state})
	status, body, _, err := s.proxyModuleHTTP(r.Context(), kiroOAuthModuleID, http.MethodPost, "/auth/oauth/exchange", payload)
	if err != nil {
		s.renderKiroOAuthCallbackResultTo(w, false, err.Error(), s.localAdminProvidersURL())
		return
	}
	if status < 200 || status >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("token exchange failed (%d)", status)
		}
		s.renderKiroOAuthCallbackResultTo(w, false, msg, s.localAdminProvidersURL())
		return
	}

	s.renderKiroOAuthCallbackResultTo(w, true, "OAuth success. You can close this page and return to LightBridge.", s.localAdminProvidersURL())
}
