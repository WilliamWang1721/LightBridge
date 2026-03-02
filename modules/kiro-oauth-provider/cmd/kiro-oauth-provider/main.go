package main

import (
	"context"
	"encoding/json"
	"errors"
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
		port = "39112"
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
	storePath := filepath.Join(dataDir, "accounts.json")
	acctStore, err := newAccountStore(storePath)
	if err != nil {
		log.Fatalf("init account store: %v", err)
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	s := &server{
		cfg:       cfg,
		cfgPath:   cfgPath,
		httpc:     &http.Client{Timeout: timeout},
		store:     acctStore,
		storePath: storePath,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

	mux.HandleFunc("/auth/status", s.handleAuthStatus)
	mux.HandleFunc("/auth/oauth/start", s.handleAuthOAuthStart)
	mux.HandleFunc("/auth/oauth/exchange", s.handleAuthOAuthExchange)
	mux.HandleFunc("/auth/device/start", s.handleAuthDeviceStart)
	mux.HandleFunc("/auth/import", s.handleAuthImport)
	mux.HandleFunc("/auth/refresh", s.handleAuthRefresh)
	mux.HandleFunc("/auth/accounts/enable", s.handleAuthAccountEnable)
	mux.HandleFunc("/auth/accounts/disable", s.handleAuthAccountDisable)
	mux.HandleFunc("/auth/accounts/delete", s.handleAuthAccountDelete)
	mux.HandleFunc("/auth/accounts/activate", s.handleAuthAccountActivate)

	mux.HandleFunc("/usage/limits", s.handleUsageLimits)

	addr := "127.0.0.1:" + port
	log.Printf("kiro-oauth-provider module listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

type server struct {
	cfg     config
	cfgPath string

	httpc *http.Client

	store     *accountStore
	storePath string

	oauthMu sync.Mutex
	oauth   *oauthFlow

	deviceMu sync.Mutex
	device   *deviceFlow
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"module": "kiro-oauth-provider",
	})
}

func (s *server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	models := mergeKiroModels(s.cfg.Models)
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
			"owned_by": "kiro",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": out})
}

func (s *server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var oauthCopy *oauthFlow
	s.oauthMu.Lock()
	if s.oauth != nil {
		cp := *s.oauth
		oauthCopy = &cp
	}
	s.oauthMu.Unlock()

	var deviceCopy *deviceFlow
	s.deviceMu.Lock()
	if s.device != nil {
		cp := *s.device
		deviceCopy = &cp
	}
	s.deviceMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                 true,
		"accounts":           s.store.sanitizedAccounts(),
		"active_account_id":  s.store.activeAccountID(),
		"selection_strategy": normalizeSelectionStrategy(s.cfg.SelectionStrategy),
		"oauth":              oauthCopy,
		"device":             deviceCopy,
	})
}

func (s *server) handleAuthAccountEnable(w http.ResponseWriter, r *http.Request) {
	s.handleAccountToggle(w, r, true)
}

func (s *server) handleAuthAccountDisable(w http.ResponseWriter, r *http.Request) {
	s.handleAccountToggle(w, r, false)
}

func (s *server) handleAccountToggle(w http.ResponseWriter, r *http.Request, enabled bool) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		AccountID string `json:"account_id"`
		ID        string `json:"id"`
	}
	if err := decodeJSONBody(r, 1<<20, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	id := nonEmpty(req.AccountID, req.ID)
	if strings.TrimSpace(id) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing account_id"})
		return
	}
	if err := s.store.setAccountEnabled(id, enabled); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) handleAuthAccountDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		AccountID string `json:"account_id"`
		ID        string `json:"id"`
	}
	if err := decodeJSONBody(r, 1<<20, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	id := nonEmpty(req.AccountID, req.ID)
	if strings.TrimSpace(id) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing account_id"})
		return
	}
	if err := s.store.deleteAccount(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) handleAuthAccountActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		AccountID string `json:"account_id"`
		ID        string `json:"id"`
	}
	if err := decodeJSONBody(r, 1<<20, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	id := nonEmpty(req.AccountID, req.ID)
	if strings.TrimSpace(id) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing account_id"})
		return
	}
	if err := s.store.setActiveAccount(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *server) getAccountForRequest(ctx context.Context) (*account, error) {
	acc, reason := s.store.selectAccount(s.cfg.SelectionStrategy)
	if acc == nil {
		if reason == "no_healthy_account" {
			return nil, errors.New("no available Kiro account")
		}
		return nil, errors.New("no account configured")
	}
	if _, err := s.ensureAccountAccessToken(ctx, acc.ID, false); err != nil {
		return nil, err
	}
	latest, ok := s.store.getAccount(acc.ID)
	if !ok {
		return nil, errors.New("selected account disappeared")
	}
	return latest, nil
}

func (s *server) ensureAccountAccessToken(ctx context.Context, accountID string, force bool) (*account, error) {
	acc, ok := s.store.getAccount(accountID)
	if !ok {
		return nil, errors.New("account not found")
	}
	if !force {
		if strings.TrimSpace(acc.AccessToken) != "" {
			exp := parseRFC3339OrZero(acc.ExpiresAt)
			if exp.IsZero() {
				return acc, nil
			}
			near := time.Duration(s.cfg.NearExpiryMinutes) * time.Minute
			if near < 0 {
				near = 0
			}
			if time.Until(exp) > near {
				return acc, nil
			}
		}
	}
	return s.refreshAccountTokens(ctx, accountID)
}

func (s *server) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		AccountID string `json:"account_id"`
		Force     bool   `json:"force"`
	}
	if err := decodeJSONBody(r, 1<<20, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	id := strings.TrimSpace(req.AccountID)
	if id == "" {
		id = s.store.activeAccountID()
	}
	if strings.TrimSpace(id) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing account_id"})
		return
	}
	acc, err := s.ensureAccountAccessToken(r.Context(), id, req.Force)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "account": acc})
}

func (s *server) nextMonthFirstUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

func summarizeHTTPError(status int, body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "upstream error"
	}
	var parsed map[string]any
	if json.Unmarshal(body, &parsed) == nil {
		if msg, ok := parsed["message"].(string); ok && strings.TrimSpace(msg) != "" {
			return strings.TrimSpace(msg)
		}
		if e, ok := parsed["error"].(string); ok && strings.TrimSpace(e) != "" {
			return strings.TrimSpace(e)
		}
	}
	if len(trimmed) > 600 {
		trimmed = trimmed[:600]
	}
	return "upstream " + strconv.Itoa(status) + ": " + trimmed
}
