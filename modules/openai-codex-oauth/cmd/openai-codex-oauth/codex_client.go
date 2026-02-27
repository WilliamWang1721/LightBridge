package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *server) callCodex(ctx context.Context, upstreamURL, token, accountID string, body []byte, promptCacheKey string, stream bool) (*http.Response, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("missing access token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	parsed, _ := url.Parse(upstreamURL)
	if parsed != nil && parsed.Host != "" {
		req.Host = parsed.Host
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("connection", "Keep-Alive")
	req.Header.Set("accept-encoding", "identity")

	if strings.TrimSpace(s.cfg.ClientVersion) != "" {
		req.Header.Set("version", strings.TrimSpace(s.cfg.ClientVersion))
	}
	if strings.TrimSpace(s.cfg.UserAgent) != "" {
		req.Header.Set("user-agent", strings.TrimSpace(s.cfg.UserAgent))
	}

	if stream {
		req.Header.Set("accept", "text/event-stream")
	} else {
		req.Header.Set("accept", "application/json")
	}

	if strings.TrimSpace(s.cfg.BetaFeatures) != "" {
		req.Header.Set("x-codex-beta-features", strings.TrimSpace(s.cfg.BetaFeatures))
	}
	if s.cfg.WebSearchEligible {
		req.Header.Set("x-oai-web-search-eligible", "true")
	}

	// Codex backend expects these headers for OAuth sessions.
	req.Header.Set("originator", "codex_cli_rs")
	if strings.TrimSpace(accountID) != "" {
		req.Header.Set("chatgpt-account-id", strings.TrimSpace(accountID))
	}

	// Conversation/session identifiers (optional).
	if strings.TrimSpace(promptCacheKey) != "" {
		req.Header.Set("conversation_id", strings.TrimSpace(promptCacheKey))
		req.Header.Set("session_id", strings.TrimSpace(promptCacheKey))
	} else {
		req.Header.Set("session_id", newUUID())
	}

	// Ensure the upstream request doesn't hang forever on network stalls.
	if s.httpc != nil && s.httpc.Timeout == 0 {
		s.httpc.Timeout = 120 * time.Second
	}
	return s.httpc.Do(req)
}
