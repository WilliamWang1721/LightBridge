package main

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func (s *server) handleResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 20<<20))
	_ = r.Body.Close()
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "failed to read body", "invalid_request_error", "invalid_body")
		return
	}

	ctx := r.Context()
	if err := s.maybeRefreshCredentials(ctx); err != nil {
		// Non-fatal; continue and let upstream 401 surface if token is invalid.
		log.Printf("auth refresh: %v", err)
	}

	accessToken, accountID, ok := s.getAccessToken()
	if !ok {
		writeOpenAIError(w, http.StatusUnauthorized, "Codex OAuth not configured. Use /auth/oauth/start (recommended), /auth/device/start, or /auth/import.", "authentication_error", "not_authenticated")
		return
	}

	upstream := strings.TrimRight(strings.TrimSpace(s.cfg.BaseURL), "/") + "/responses"
	doReq := func(token string) (*http.Response, error) {
		return s.callCodex(ctx, upstream, token, accountID, body, "", true)
	}

	resp, err := doReq(accessToken)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "api_error", "upstream_unreachable")
		return
	}
	if resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()
		if refreshed := s.refreshOnce(ctx); refreshed {
			if token2, _, ok2 := s.getAccessToken(); ok2 {
				resp, err = doReq(token2)
			}
		}
	}
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "api_error", "upstream_unreachable")
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.Header().Set("x-accel-buffering", "no")
	if strings.TrimSpace(w.Header().Get("content-type")) == "" {
		w.Header().Set("content-type", "text/event-stream")
	}
	w.WriteHeader(resp.StatusCode)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(w, resp.Body)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		_, _ = io.Copy(w, resp.Body)
		return
	}

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			flusher.Flush()
		}
		if readErr != nil {
			return
		}
	}
}
