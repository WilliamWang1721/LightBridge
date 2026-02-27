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

const codexOAuthLocalRedirectURI = "http://localhost:1455/auth/callback"

func (s *Server) ensureCodexOAuthCallbackServer() {
	s.codexOAuthCallbackMu.Lock()
	defer s.codexOAuthCallbackMu.Unlock()

	if s.codexOAuthCallbackStarted {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", s.handleCodexOAuthLocalCallback)

	handler := requestIDMiddleware(loggingMiddleware(mux))

	start := func(addr string) error {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		srv := &http.Server{
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("codex oauth callback server (%s) error: %v", addr, err)
			}
		}()
		return nil
	}

	var problems []string
	started := false
	for _, addr := range []string{"127.0.0.1:1455", "[::1]:1455"} {
		if err := start(addr); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", addr, err))
			continue
		}
		started = true
	}

	if !started {
		s.codexOAuthCallbackErr = fmt.Errorf("failed to start local callback server on :1455 (%s)", strings.Join(problems, "; "))
		return
	}

	s.codexOAuthCallbackStarted = true
	s.codexOAuthCallbackErr = nil
}

func (s *Server) localAdminProvidersURL() string {
	addr := strings.TrimSpace(s.cfg.ListenAddr)
	if addr == "" {
		return "http://127.0.0.1:3210/admin/providers"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		// Handle forms like ":3210".
		if strings.HasPrefix(addr, ":") && strings.TrimSpace(addr[1:]) != "" {
			port = strings.TrimSpace(addr[1:])
		}
	}
	if strings.TrimSpace(port) == "" {
		port = "3210"
	}
	return "http://127.0.0.1:" + port + "/admin/providers"
}

func (s *Server) handleCodexOAuthLocalCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errStr := strings.TrimSpace(q.Get("error")); errStr != "" {
		desc := strings.TrimSpace(q.Get("error_description"))
		msg := errStr
		if desc != "" {
			msg += ": " + desc
		}
		s.renderCodexOAuthCallbackResultTo(w, false, msg, s.localAdminProvidersURL())
		return
	}

	code := strings.TrimSpace(q.Get("code"))
	state := strings.TrimSpace(q.Get("state"))
	if code == "" || state == "" {
		s.renderCodexOAuthCallbackResultTo(w, false, "missing code/state in callback url", s.localAdminProvidersURL())
		return
	}

	payload, _ := json.Marshal(map[string]string{"code": code, "state": state})
	status, body, _, err := s.proxyModuleHTTP(r.Context(), codexOAuthModuleID, http.MethodPost, "/auth/oauth/exchange", payload)
	if err != nil {
		s.renderCodexOAuthCallbackResultTo(w, false, err.Error(), s.localAdminProvidersURL())
		return
	}
	if status < 200 || status >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("token exchange failed (%d)", status)
		}
		s.renderCodexOAuthCallbackResultTo(w, false, msg, s.localAdminProvidersURL())
		return
	}

	s.renderCodexOAuthCallbackResultTo(w, true, "OAuth success. You can close this page and return to LightBridge.", s.localAdminProvidersURL())
}

