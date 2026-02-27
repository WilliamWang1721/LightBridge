package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	port := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_HTTP_PORT"))
	if port == "" {
		port = "39111"
	}
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("invalid LIGHTBRIDGE_HTTP_PORT: %q", port)
	}

	cfgPath := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_CONFIG_PATH"))
	cfg, cfgErr := loadConfig(cfgPath)
	if cfgErr != nil {
		log.Printf("config: %v", cfgErr)
	}

	dataDir := strings.TrimSpace(os.Getenv("LIGHTBRIDGE_DATA_DIR"))
	if dataDir == "" {
		dataDir = "."
	}
	credsPath := filepath.Join(dataDir, "credentials.json")

	s := &server{
		cfg:       cfg,
		cfgPath:   cfgPath,
		credsPath: credsPath,
		httpc: &http.Client{
			Timeout: 120 * time.Second,
		},
		conversations: map[string]conversationEntry{},
	}

	if err := s.loadCredentials(); err != nil {
		log.Printf("auth: %v", err)
	}
	if err := s.maybeRefreshCredentials(context.Background()); err != nil {
		log.Printf("auth refresh: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/auth/device/start", s.handleAuthDeviceStart)
	mux.HandleFunc("/auth/oauth/start", s.handleAuthOAuthStart)
	mux.HandleFunc("/auth/oauth/exchange", s.handleAuthOAuthExchange)
	mux.HandleFunc("/auth/import", s.handleAuthImport)
	mux.HandleFunc("/auth/status", s.handleAuthStatus)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

	addr := "127.0.0.1:" + port
	log.Printf("openai-codex-oauth module listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

type server struct {
	cfg     config
	cfgPath string

	httpc *http.Client

	credsMu   sync.Mutex
	creds     *credentials
	credsPath string

	flowMu sync.Mutex
	flow   *deviceFlow

	oauthMu sync.Mutex
	oauth   *oauthFlow

	convMu        sync.Mutex
	conversations map[string]conversationEntry
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}

	models := s.cfg.Models
	if len(models) == 0 {
		models = []string{"gpt-5-codex", "gpt-5-codex-mini", "gpt-5.1-codex", "gpt-5.1-codex-mini", "gpt-5.2-codex"}
	}
	now := time.Now().Unix()
	out := make([]map[string]any, 0, len(models))
	for _, id := range models {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out = append(out, map[string]any{
			"id":       id,
			"object":   "model",
			"created":  now,
			"owned_by": "openai",
		})
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   out,
	})
}

func (s *server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
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

	var req chatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "body must be valid JSON", "invalid_request_error", "invalid_json")
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
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

	stream := req.Stream
	codexBody, revToolName, promptCacheKey, err := buildCodexRequest(body, stream)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_request")
		return
	}

	if promptCacheKey == "" {
		if sessionKey := s.sessionKeyFromRequest(&req); sessionKey != "" {
			promptCacheKey = s.promptCacheKeyFromSession(req.Model, sessionKey)
		}
	}
	if promptCacheKey != "" {
		codexBody = injectPromptCacheKey(codexBody, promptCacheKey)
	}

	upstream := strings.TrimRight(strings.TrimSpace(s.cfg.BaseURL), "/") + "/responses"
	doReq := func(token string) (*http.Response, error) {
		return s.callCodex(ctx, upstream, token, accountID, codexBody, promptCacheKey, true)
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		writeOpenAIError(w, http.StatusBadGateway, summarizeUpstreamError(resp.StatusCode, b), "api_error", "upstream_error")
		return
	}

	if stream {
		s.proxyChatCompletionStream(w, resp, req.Model, body, codexBody, revToolName)
		return
	}
	s.proxyChatCompletionNonStream(w, resp, req.Model, body, codexBody, revToolName)
}

func (s *server) sessionKeyFromRequest(req *chatCompletionRequest) string {
	if req == nil || req.Metadata == nil {
		return ""
	}
	for _, k := range []string{"session_id", "conversation_id", "user_id"} {
		if v, ok := req.Metadata[k]; ok {
			if s, ok := v.(string); ok {
				if strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return ""
}

func (s *server) proxyChatCompletionStream(w http.ResponseWriter, upstream *http.Response, requestedModel string, originalOpenAIReq, codexReq []byte, revToolName map[string]string) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")
	w.Header().Set("x-accel-buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeOpenAIError(w, http.StatusInternalServerError, "streaming not supported by server", "server_error", "stream_unsupported")
		return
	}

	state := &streamState{
		model:          requestedModel,
		createdAt:      time.Now().Unix(),
		toolCallIndex:  -1,
		revToolNameMap: revToolName,
	}

	sc := newSSEScanner(upstream.Body)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		if !hasDataPrefix(line) {
			continue
		}
		data := trimDataPrefix(line)
		if len(data) == 0 || string(data) == "[DONE]" {
			continue
		}
		chunks, done, err := convertCodexEventToOpenAIChunks(data, state)
		if err != nil {
			continue
		}
		for _, chunk := range chunks {
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(chunk)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
		if done {
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
			return
		}
	}
}

func (s *server) proxyChatCompletionNonStream(w http.ResponseWriter, upstream *http.Response, requestedModel string, originalOpenAIReq, codexReq []byte, revToolName map[string]string) {
	var completed *codexEvent
	sc := newSSEScanner(upstream.Body)
	for sc.Scan() {
		line := sc.Bytes()
		if !hasDataPrefix(line) {
			continue
		}
		data := trimDataPrefix(line)
		if len(data) == 0 || string(data) == "[DONE]" {
			continue
		}
		var ev codexEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			continue
		}
		if ev.Type == "response.completed" {
			completed = &ev
			break
		}
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "api_error", "upstream_stream_failed")
		return
	}
	if completed == nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream stream closed before response.completed", "api_error", "upstream_incomplete")
		return
	}

	out, err := convertCodexCompletedToOpenAICompletion(completed, requestedModel, revToolName)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, err.Error(), "api_error", "translation_failed")
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func summarizeUpstreamError(status int, body []byte) string {
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = "empty upstream response"
	}
	if len(msg) > 500 {
		msg = msg[:500] + "..."
	}
	return "upstream error (" + strconv.Itoa(status) + "): " + msg
}

var errNoCredentials = errors.New("no credentials")
