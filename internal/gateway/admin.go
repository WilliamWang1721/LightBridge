package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"lightbridge/internal/advstats"
	"lightbridge/internal/routing"
	"lightbridge/internal/types"
	"lightbridge/internal/util"
)

const codexOAuthModuleID = "openai-codex-oauth"
const passkeyLoginModuleID = "passkey-login"
const advancedStatisticsModuleID = "advanced-statistics"
const anthropicVersionHeaderValue = "2023-06-01"

type adminPayload struct {
	Username string         `json:"username"`
	Password string         `json:"password"`
	Remember bool           `json:"remember"`
	Device   map[string]any `json:"device"`
}

type providerUpdatePayload struct {
	ID          string     `json:"id"`
	DisplayName *string    `json:"displayName"`
	GroupName   *string    `json:"groupName"`
	Type        string     `json:"type"`
	Protocol    string     `json:"protocol"`
	Endpoint    string     `json:"endpoint"`
	APIKey      *string    `json:"apiKey"`
	Token       *string    `json:"token"`
	ConfigJSON  string     `json:"configJSON"`
	Enabled     *bool      `json:"enabled"`
	Health      *string    `json:"health"`
	LastCheckAt *time.Time `json:"lastCheckAt"`
}

func (s *Server) wrapAdminPage(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hasAdmin, err := s.store.HasAdmin(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !hasAdmin {
			http.Redirect(w, r, "/admin/setup", http.StatusFound)
			return
		}
		if _, ok := s.sessions.username(r); !ok {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *Server) wrapAdminAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hasAdmin, err := s.store.HasAdmin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		if !hasAdmin {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "setup required"})
			return
		}
		if _, ok := s.sessions.username(r); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "login required"})
			return
		}
		next(w, r)
	}
}

func (s *Server) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	hasAdmin, err := s.store.HasAdmin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if hasAdmin {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}
	s.renderPage(w, "setup", map[string]any{"Page": "Setup Wizard"})
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	hasAdmin, err := s.store.HasAdmin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !hasAdmin {
		http.Redirect(w, r, "/admin/setup", http.StatusFound)
		return
	}
	passkeyInstalled := false
	if mod, err := s.store.GetInstalledModule(r.Context(), passkeyLoginModuleID); err == nil && mod != nil && mod.Enabled {
		passkeyInstalled = true
	}
	twoFAInstalled := false
	if mod, err := s.store.GetInstalledModule(r.Context(), totp2FAModuleID); err == nil && mod != nil && mod.Enabled {
		twoFAInstalled = true
	}
	s.renderPage(w, "login", map[string]any{
		"Page":             "Admin Login",
		"PasskeyInstalled": passkeyInstalled,
		"TwoFAInstalled":   twoFAInstalled,
	})
}

func (s *Server) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	providers, _ := s.store.ListProviders(r.Context(), true)
	models, _ := s.store.ListModels(r.Context(), true)
	modules, _ := s.store.ListInstalledModules(r.Context())
	username, _ := s.sessions.username(r)
	if strings.TrimSpace(username) == "" {
		username = "Admin"
	}
	s.renderPage(w, "dashboard", map[string]any{
		"Page":          "Dashboard",
		"ProviderCount": len(providers),
		"ModelCount":    len(models),
		"ModuleCount":   len(modules),
		"Username":      username,
	})
}

func (s *Server) renderPage(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Page"]; !ok {
		data["Page"] = strings.Title(name)
	}
	if err := s.templates.ExecuteTemplate(w, name+".html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminSetupAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	hasAdmin, err := s.store.HasAdmin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if hasAdmin {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "admin already initialized"})
		return
	}
	var payload adminPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Password) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "username and password are required"})
		return
	}
	hash, err := util.HashPassword(payload.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.CreateAdmin(r.Context(), payload.Username, hash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defaultKeyValue, err := util.NewClientAPIKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	keyID, _ := util.RandomToken(8)
	if err := s.store.CreateClientKey(r.Context(), types.ClientAPIKey{
		ID:        "default_" + keyID,
		Key:       defaultKeyValue,
		Name:      "Default Client Key",
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if err := s.sessions.newSession(w, payload.Username, false); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                 true,
		"default_client_key": defaultKeyValue,
		"next":               "/admin/dashboard",
		"message":            "setup complete",
	})
}

func (s *Server) handleAdminLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var payload adminPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	hash, err := s.store.GetAdminPasswordHash(r.Context(), payload.Username)
	if err != nil || !util.CheckPassword(hash, payload.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid username or password"})
		return
	}
	s.finalizePrimaryLogin(w, r, payload.Username, payload.Remember, "password")
}

func (s *Server) handleProvidersAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		providers, err := s.store.ListProviders(r.Context(), true)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": providers})
	case http.MethodPost:
		var req providerUpdatePayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(req.ID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provider id is required"})
			return
		}
		existing, err := s.store.GetProvider(r.Context(), req.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		payload := types.Provider{
			ID:       strings.TrimSpace(req.ID),
			Type:     strings.TrimSpace(req.Type),
			Protocol: strings.TrimSpace(req.Protocol),
			Endpoint: strings.TrimSpace(req.Endpoint),
		}
		if strings.TrimSpace(req.ConfigJSON) == "" {
			payload.ConfigJSON = "{}"
		} else {
			payload.ConfigJSON = req.ConfigJSON
		}
		apiKey := ""
		if req.APIKey != nil {
			apiKey = strings.TrimSpace(*req.APIKey)
		}
		if apiKey == "" && req.Token != nil {
			apiKey = strings.TrimSpace(*req.Token)
		}
		if apiKey != "" {
			cfg := map[string]any{}
			_ = json.Unmarshal([]byte(payload.ConfigJSON), &cfg)
			if cfg == nil {
				cfg = map[string]any{}
			}
			cfg["api_key"] = apiKey
			if b, err := json.Marshal(cfg); err == nil {
				payload.ConfigJSON = string(b)
			}
		}
		if req.Enabled != nil {
			payload.Enabled = *req.Enabled
		} else if existing != nil {
			payload.Enabled = existing.Enabled
		} else {
			payload.Enabled = true
		}

		if payload.Type == "" {
			payload.Type = types.ProviderTypeBuiltin
		}
		if payload.Protocol == "" {
			payload.Protocol = types.ProtocolForward
		}

		if req.DisplayName != nil {
			payload.DisplayName = strings.TrimSpace(*req.DisplayName)
		} else if existing != nil {
			payload.DisplayName = existing.DisplayName
		} else {
			payload.DisplayName = payload.ID
		}

		if req.GroupName != nil {
			payload.GroupName = strings.TrimSpace(*req.GroupName)
		} else if existing != nil {
			payload.GroupName = existing.GroupName
		}

		if req.Health != nil {
			payload.Health = strings.TrimSpace(*req.Health)
		} else if existing != nil {
			payload.Health = existing.Health
		}

		if req.LastCheckAt != nil {
			payload.LastCheckAt = req.LastCheckAt
		} else if existing != nil {
			payload.LastCheckAt = existing.LastCheckAt
		}

		if err := s.store.UpsertProvider(r.Context(), payload); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleProviderPullModelsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ID         string `json:"id"`
		ProviderID string `json:"provider_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	providerID := strings.TrimSpace(req.ProviderID)
	if providerID == "" {
		providerID = strings.TrimSpace(req.ID)
	}
	if providerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provider_id is required"})
		return
	}
	provider, err := s.store.GetProvider(r.Context(), providerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if provider == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "provider not found"})
		return
	}
	modelIDs, sourceURL, err := fetchProviderModelIDs(r.Context(), *provider)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	inserted, err := s.store.InsertModelsIfMissing(r.Context(), modelIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"provider_id": providerID,
		"source_url":  sourceURL,
		"total":       len(modelIDs),
		"inserted":    inserted,
	})
}

type providerModelFetchConfig struct {
	BaseURL    string `json:"base_url"`
	BaseOrigin string `json:"base_origin"`
	APIKey     string `json:"api_key"`
}

type openAIModelList struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type geminiModelList struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func fetchProviderModelIDs(ctx context.Context, provider types.Provider) ([]string, string, error) {
	proto := strings.TrimSpace(provider.Protocol)
	switch proto {
	case types.ProtocolOpenAI, types.ProtocolForward, types.ProtocolHTTPOpenAI, types.ProtocolHTTPRPC, types.ProtocolCodex, types.ProtocolOpenAIResponses, types.ProtocolGemini, types.ProtocolAnthropic, types.ProtocolAzureOpenAI:
		// ok
	default:
		return nil, "", fmt.Errorf("provider protocol %q does not support model listing", proto)
	}

	cfg := providerModelFetchConfig{}
	if strings.TrimSpace(provider.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}
	baseURL := strings.TrimSpace(cfg.BaseOrigin)
	if baseURL == "" {
		baseURL = strings.TrimSpace(cfg.BaseURL)
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(provider.Endpoint)
	}
	if baseURL == "" {
		return nil, "", fmt.Errorf("provider %s missing endpoint", provider.ID)
	}
	modelsPath := "/v1/models"
	switch proto {
	case types.ProtocolGemini:
		modelsPath = "/v1beta/models"
	case types.ProtocolAzureOpenAI:
		modelsPath = "/openai/v1/models"
	}
	modelsURL, err := joinUpstreamURL(baseURL, modelsPath)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("accept", "application/json")
	if apiKey := strings.TrimSpace(cfg.APIKey); apiKey != "" {
		switch proto {
		case types.ProtocolGemini:
			req.Header.Set("x-goog-api-key", apiKey)
		case types.ProtocolAnthropic:
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", anthropicVersionHeaderValue)
		case types.ProtocolAzureOpenAI:
			req.Header.Set("api-key", apiKey)
		default:
			req.Header.Set("authorization", "Bearer "+apiKey)
		}
	}

	httpc := &http.Client{Timeout: 12 * time.Second}
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, modelsURL, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, modelsURL, fmt.Errorf("upstream models failed (%d): %s", resp.StatusCode, msg)
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, 64)
	if proto == types.ProtocolGemini {
		var parsed geminiModelList
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, modelsURL, fmt.Errorf("decode models response: %w", err)
		}
		for _, item := range parsed.Models {
			id := strings.TrimSpace(item.Name)
			if i := strings.LastIndex(id, "/"); i >= 0 && i < len(id)-1 {
				id = id[i+1:]
			}
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	} else {
		var parsed openAIModelList
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, modelsURL, fmt.Errorf("decode models response: %w", err)
		}
		for _, item := range parsed.Data {
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, modelsURL, errors.New("upstream models returned empty list")
	}
	sort.Strings(out)
	return out, modelsURL, nil
}

func joinUpstreamURL(baseURL, reqPath string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", errors.New("base url is empty")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		return "", fmt.Errorf("base url missing scheme: %s", baseURL)
	}

	p := strings.TrimSpace(reqPath)
	if p == "" {
		return u.String(), nil
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	basePath := strings.TrimRight(u.Path, "/")
	if basePath == "" || basePath == "/" {
		u.Path = p
		return u.String(), nil
	}
	if p == basePath || strings.HasPrefix(p, basePath+"/") {
		u.Path = p
		return u.String(), nil
	}
	u.Path = basePath + p
	return u.String(), nil
}

type modelRoutePayload struct {
	Model  types.Model        `json:"model"`
	Routes []types.ModelRoute `json:"routes"`
}

type experimentChatPayload struct {
	ConversationID string           `json:"conversation_id"`
	Model          string           `json:"model"`
	Messages       []map[string]any `json:"messages"`
	Params         map[string]any   `json:"params"`
}

type chatboxConversationCreatePayload struct {
	Title        string `json:"title"`
	Model        string `json:"model"`
	SystemPrompt string `json:"system_prompt"`
}

type chatboxMessagePayload struct {
	Content string         `json:"content"`
	Model   string         `json:"model"`
	Params  map[string]any `json:"params"`
}

func (s *Server) handleProviderDeleteAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ID  string   `json:"id"`
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	ids := req.IDs
	if len(ids) == 0 && strings.TrimSpace(req.ID) != "" {
		ids = []string{strings.TrimSpace(req.ID)}
	}
	if len(ids) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id or ids is required"})
		return
	}
	var deleted, failed int
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		// If user deletes a builtin provider, persist a tombstone so it won't be recreated on restart.
		// (Builtin providers are normally ensured at startup.)
		if existing, err := s.store.GetProvider(r.Context(), id); err == nil && existing != nil && existing.Type == types.ProviderTypeBuiltin {
			_ = s.store.SetSetting(r.Context(), "builtin_provider_removed:"+id, time.Now().UTC().Format(time.RFC3339))
		}
		if err := s.store.DeleteProvider(r.Context(), id); err != nil {
			failed++
		} else {
			deleted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted, "failed": failed})
}

func (s *Server) handleModelsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		models, err := s.store.ListModels(r.Context(), true)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		routes, err := s.store.ListAllModelRoutes(r.Context(), true)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"models": models, "routes": routes})
	case http.MethodPost:
		var payload modelRoutePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(payload.Model.ID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model.id is required"})
			return
		}
		if payload.Model.DisplayName == "" {
			payload.Model.DisplayName = payload.Model.ID
		}
		if err := s.store.UpsertModel(r.Context(), payload.Model); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		for idx := range payload.Routes {
			payload.Routes[idx].ModelID = payload.Model.ID
			if payload.Routes[idx].Weight == 0 {
				payload.Routes[idx].Weight = 1
			}
		}
		if err := s.store.ReplaceModelRoutes(r.Context(), payload.Model.ID, payload.Routes); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleDashboardAPI(w http.ResponseWriter, r *http.Request) {
	providers, _ := s.store.ListProviders(r.Context(), true)
	models, _ := s.store.ListModels(r.Context(), true)
	modules, _ := s.store.ListInstalledModules(r.Context())
	logs, _ := s.store.ListRequestLogs(r.Context(), 20)
	now := time.Now().UTC()
	stats24h, _ := s.store.RequestStatsSince(r.Context(), now.Add(-24*time.Hour))
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -6)
	tokens7d, _ := s.store.TokenUsageLastNDays(r.Context(), startDay, 7)
	pathModel24h, _ := s.store.PathModelUsageSince(r.Context(), now.Add(-24*time.Hour), 300)
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": providers,
		"models":    models,
		"modules":   modules,
		"logs":      logs,
		"stats": map[string]any{
			"provider_total": len(providers),
			"model_total":    len(models),
			"module_total":   len(modules),
			"requests_24h":   stats24h.Requests,
			"tokens_24h":     stats24h.InputTokens + stats24h.OutputTokens,
			"uptime_sec":     int64(time.Since(s.startedAt).Seconds()),
		},
		"tokens_7d":      tokens7d,
		"path_model_24h": pathModel24h,
		"now":            now.Format(time.RFC3339),
	})
}

func (s *Server) handleChatboxConversationsAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListChatConversations(r.Context(), 300)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"conversations": items})
	case http.MethodPost:
		var req chatboxConversationCreatePayload
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}

		modelID := strings.TrimSpace(req.Model)
		if modelID == "" {
			models, err := s.store.ListModels(r.Context(), false)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			if len(models) == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no model available, please configure models first"})
				return
			}
			modelID = strings.TrimSpace(models[0].ID)
		}
		if modelID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model is required"})
			return
		}

		model, err := s.store.GetModel(r.Context(), modelID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		if model == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model does not exist"})
			return
		}

		conversationID := newChatboxConversationID()
		title := strings.TrimSpace(req.Title)
		if title == "" {
			title = "新对话"
		}
		if err := s.store.CreateChatConversation(r.Context(), types.ChatConversation{
			ID:           conversationID,
			Title:        title,
			ModelID:      modelID,
			SystemPrompt: strings.TrimSpace(req.SystemPrompt),
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		conv, err := s.store.GetChatConversation(r.Context(), conversationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"conversation": conv,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleChatboxConversationItemAPI(w http.ResponseWriter, r *http.Request, cleanedPath string) {
	conversationID, action, ok := parseChatboxConversationPath(cleanedPath)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch action {
	case "":
		switch r.Method {
		case http.MethodGet:
			conv, err := s.store.GetChatConversation(r.Context(), conversationID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			if conv == nil {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": "conversation not found"})
				return
			}
			msgs, err := s.store.ListChatMessages(r.Context(), conversationID, 3000)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"conversation": conv,
				"messages":     msgs,
			})
		case http.MethodDelete:
			if err := s.store.DeleteChatConversation(r.Context(), conversationID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		}
	case "messages":
		s.handleChatboxMessageAPI(w, r, conversationID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleChatboxMessageAPI(w http.ResponseWriter, r *http.Request, conversationID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req chatboxMessagePayload
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	userInput := strings.TrimSpace(req.Content)
	if userInput == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "content is required"})
		return
	}

	conv, err := s.store.GetChatConversation(r.Context(), conversationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if conv == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "conversation not found"})
		return
	}

	modelID := strings.TrimSpace(req.Model)
	if modelID == "" {
		modelID = strings.TrimSpace(conv.ModelID)
	}
	if modelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model is required"})
		return
	}
	model, err := s.store.GetModel(r.Context(), modelID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if model == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model does not exist"})
		return
	}

	history, err := s.store.ListChatMessages(r.Context(), conversationID, 4000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	reqMessages := make([]map[string]any, 0, len(history)+2)
	if systemPrompt := strings.TrimSpace(conv.SystemPrompt); systemPrompt != "" {
		reqMessages = append(reqMessages, map[string]any{
			"role":    "system",
			"content": systemPrompt,
		})
	}
	for _, item := range history {
		role := strings.TrimSpace(item.Role)
		if role != "user" && role != "assistant" && role != "system" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		reqMessages = append(reqMessages, map[string]any{
			"role":    role,
			"content": content,
		})
	}
	reqMessages = append(reqMessages, map[string]any{
		"role":    "user",
		"content": userInput,
	})

	requestBody := map[string]any{
		"model":    modelID,
		"messages": reqMessages,
		"stream":   false,
	}
	mergeExperimentParams(requestBody, req.Params)
	requestBody["model"] = modelID
	requestBody["messages"] = reqMessages
	requestBody["stream"] = false

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request payload"})
		return
	}

	baseModelID, tagEffort, hasModelTag, tagErr := parseModelTag(modelID, s.cfg.ModelTagAliases)
	if tagErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": tagErr.Error()})
		return
	}
	if hasModelTag {
		modelID = baseModelID
		requestBytes, err = patchReasoningEffort(requestBytes, endpointKindChatCompletions, baseModelID, tagEffort, true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request payload"})
			return
		}
	}

	route, err := s.resolver.Resolve(r.Context(), modelID)
	if err != nil && errors.Is(err, routing.ErrNoHealthyProvider) {
		s.startEnabledModulesBestEffort()
		route, err = s.resolver.Resolve(r.Context(), modelID)
	}
	if err != nil || route == nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "routing failed", "details": errString(err)})
		return
	}

	provider, err := s.store.GetProvider(r.Context(), route.ProviderID)
	if err != nil || provider == nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "provider unavailable"})
		return
	}
	adapter, ok := s.providers.Get(provider.Protocol)
	if !ok {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "provider protocol is not supported"})
		return
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "/v1/chat/completions", bytes.NewReader(requestBytes))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to build upstream request"})
		return
	}
	upstreamReq.Header.Set("content-type", "application/json")
	upstreamReq.Header.Set("accept", "application/json")

	buffered := newBufferedResponseWriter()
	status, errorCode, callErr := adapter.Handle(r.Context(), buffered, upstreamReq, *provider, route)
	if status == 0 {
		status = buffered.StatusCode()
	}
	if status == 0 {
		status = http.StatusBadGateway
	}

	rawResp := bytes.TrimSpace(buffered.body.Bytes())
	respObj := decodeExperimentResponse(rawResp)
	assistantText := strings.TrimSpace(extractExperimentAssistantText(respObj))
	reasoningText := strings.TrimSpace(extractExperimentReasoningText(respObj))

	if callErr != nil {
		writeJSON(w, status, map[string]any{
			"error":      "chat request failed",
			"details":    callErr.Error(),
			"error_code": nonEmpty(errorCode, "upstream_error"),
			"route":      experimentRouteInfo(route, provider),
			"response":   respObj,
		})
		return
	}
	if status >= 400 {
		writeJSON(w, status, map[string]any{
			"error":      nonEmpty(extractExperimentError(respObj), "upstream request failed"),
			"error_code": nonEmpty(errorCode, "upstream_error"),
			"route":      experimentRouteInfo(route, provider),
			"response":   respObj,
		})
		return
	}

	if assistantText == "" {
		assistantText = "[返回成功，但未提取到文本，请查看原始响应]"
	}
	title := strings.TrimSpace(conv.Title)
	if len(history) == 0 || title == "" || title == "新对话" {
		title = chatboxTitleFromInput(userInput)
	}

	routeModel := strings.TrimSpace(modelID)
	if route != nil && strings.TrimSpace(route.UpstreamModel) != "" {
		routeModel = strings.TrimSpace(route.UpstreamModel)
	}

	if err := s.store.AppendChatExchange(
		r.Context(),
		conversationID,
		modelID,
		title,
		userInput,
		assistantText,
		reasoningText,
		nonEmpty(provider.ID, ""),
		routeModel,
	); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	updatedConv, _ := s.store.GetChatConversation(r.Context(), conversationID)
	msgs, _ := s.store.ListChatMessages(r.Context(), conversationID, 4000)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"conversation_id": conversationID,
		"assistant_text":  assistantText,
		"reasoning_text":  reasoningText,
		"route":           experimentRouteInfo(route, provider),
		"response":        respObj,
		"conversation":    updatedConv,
		"messages":        msgs,
	})
}

func parseChatboxConversationPath(cleanedPath string) (conversationID, action string, ok bool) {
	const prefix = "/chatbox/conversations/"
	if !strings.HasPrefix(cleanedPath, prefix) {
		return "", "", false
	}
	rest := strings.Trim(strings.TrimPrefix(cleanedPath, prefix), "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) > 2 {
		return "", "", false
	}
	decodedID, err := url.PathUnescape(parts[0])
	if err != nil {
		decodedID = parts[0]
	}
	conversationID = strings.TrimSpace(decodedID)
	if conversationID == "" {
		return "", "", false
	}
	if len(parts) == 2 {
		action = strings.TrimSpace(parts[1])
	}
	return conversationID, action, true
}

func newChatboxConversationID() string {
	base := strconv.FormatInt(time.Now().UnixNano(), 36)
	suffix, err := util.RandomToken(4)
	if err != nil || strings.TrimSpace(suffix) == "" {
		return "chat_" + base
	}
	return "chat_" + base + "_" + strings.ToLower(suffix)
}

func chatboxTitleFromInput(content string) string {
	normalized := strings.Join(strings.Fields(strings.ReplaceAll(content, "\n", " ")), " ")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return "新对话"
	}
	runes := []rune(normalized)
	if len(runes) > 28 {
		return string(runes[:28]) + "..."
	}
	return normalized
}

func (s *Server) handleExperimentChatAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req experimentChatPayload
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	modelID := strings.TrimSpace(req.Model)
	if modelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model is required"})
		return
	}
	if len(req.Messages) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "messages are required"})
		return
	}

	requestBody := map[string]any{
		"model":    modelID,
		"messages": req.Messages,
		"stream":   false, // experiment endpoint returns non-stream JSON for easier debugging.
	}
	mergeExperimentParams(requestBody, req.Params)
	requestBody["model"] = modelID
	requestBody["messages"] = req.Messages
	requestBody["stream"] = false

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request payload"})
		return
	}

	baseModelID, tagEffort, hasModelTag, tagErr := parseModelTag(modelID, s.cfg.ModelTagAliases)
	if tagErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": tagErr.Error()})
		return
	}
	if hasModelTag {
		modelID = baseModelID
		requestBytes, err = patchReasoningEffort(requestBytes, endpointKindChatCompletions, baseModelID, tagEffort, true)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request payload"})
			return
		}
	}

	route, err := s.resolver.Resolve(r.Context(), modelID)
	if err != nil && errors.Is(err, routing.ErrNoHealthyProvider) {
		s.startEnabledModulesBestEffort()
		route, err = s.resolver.Resolve(r.Context(), modelID)
	}
	if err != nil || route == nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "routing failed", "details": errString(err)})
		return
	}

	provider, err := s.store.GetProvider(r.Context(), route.ProviderID)
	if err != nil || provider == nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "provider unavailable"})
		return
	}

	adapter, ok := s.providers.Get(provider.Protocol)
	if !ok {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "provider protocol is not supported"})
		return
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "/v1/chat/completions", bytes.NewReader(requestBytes))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to build upstream request"})
		return
	}
	upstreamReq.Header.Set("content-type", "application/json")
	upstreamReq.Header.Set("accept", "application/json")

	buffered := newBufferedResponseWriter()
	status, errorCode, callErr := adapter.Handle(r.Context(), buffered, upstreamReq, *provider, route)
	if status == 0 {
		status = buffered.StatusCode()
	}
	if status == 0 {
		status = http.StatusBadGateway
	}

	rawResp := bytes.TrimSpace(buffered.body.Bytes())
	respObj := decodeExperimentResponse(rawResp)
	assistantText := extractExperimentAssistantText(respObj)

	if callErr != nil {
		writeJSON(w, status, map[string]any{
			"error":           "experiment request failed",
			"details":         callErr.Error(),
			"error_code":      nonEmpty(errorCode, "upstream_error"),
			"conversation_id": strings.TrimSpace(req.ConversationID),
			"route":           experimentRouteInfo(route, provider),
			"response":        respObj,
		})
		return
	}

	if status >= 400 {
		writeJSON(w, status, map[string]any{
			"error":           nonEmpty(extractExperimentError(respObj), "upstream request failed"),
			"error_code":      nonEmpty(errorCode, "upstream_error"),
			"conversation_id": strings.TrimSpace(req.ConversationID),
			"route":           experimentRouteInfo(route, provider),
			"response":        respObj,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"conversation_id": strings.TrimSpace(req.ConversationID),
		"assistant_text":  assistantText,
		"route":           experimentRouteInfo(route, provider),
		"response":        respObj,
		"request": map[string]any{
			"model":         modelID,
			"message_count": len(req.Messages),
		},
	})
}

func mergeExperimentParams(dst map[string]any, params map[string]any) {
	if len(params) == 0 {
		return
	}
	for k, v := range params {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		dst[key] = v
	}
}

func decodeExperimentResponse(raw []byte) any {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal(trimmed, &out); err == nil {
		return out
	}
	return string(trimmed)
}

func extractExperimentAssistantText(resp any) string {
	root, ok := resp.(map[string]any)
	if !ok || root == nil {
		return ""
	}

	if choices, ok := root["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if msg, ok := choice["message"].(map[string]any); ok {
				if txt := extractMessageContentText(msg["content"]); txt != "" {
					return txt
				}
			}
			if delta, ok := choice["delta"].(map[string]any); ok {
				if txt := extractMessageContentText(delta["content"]); txt != "" {
					return txt
				}
			}
		}
	}

	if txt, ok := root["output_text"].(string); ok {
		return strings.TrimSpace(txt)
	}
	return ""
}

func extractExperimentReasoningText(resp any) string {
	root, ok := resp.(map[string]any)
	if !ok || root == nil {
		return ""
	}
	parts := make([]string, 0, 3)

	if choices, ok := root["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if msg, ok := choice["message"].(map[string]any); ok {
				appendNonEmpty(&parts, extractReasoningFromMessage(msg))
			}
			if delta, ok := choice["delta"].(map[string]any); ok {
				appendNonEmpty(&parts, extractReasoningFromMessage(delta))
			}
			appendNonEmpty(&parts, extractReasoningText(choice["reasoning"]))
			appendNonEmpty(&parts, extractReasoningText(choice["reasoning_content"]))
		}
	}

	appendNonEmpty(&parts, extractReasoningText(root["reasoning"]))
	appendNonEmpty(&parts, extractReasoningText(root["reasoning_content"]))

	if output, ok := root["output"].([]any); ok {
		for _, item := range output {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			itemType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", obj["type"])))
			if strings.Contains(itemType, "reasoning") {
				appendNonEmpty(&parts, extractReasoningText(obj["text"]))
				appendNonEmpty(&parts, extractReasoningText(obj["content"]))
				appendNonEmpty(&parts, extractReasoningText(obj["summary"]))
			}
			appendNonEmpty(&parts, extractReasoningText(obj["reasoning"]))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func extractReasoningFromMessage(msg map[string]any) string {
	parts := make([]string, 0, 3)
	appendNonEmpty(&parts, extractReasoningText(msg["reasoning"]))
	appendNonEmpty(&parts, extractReasoningText(msg["reasoning_content"]))
	if content, ok := msg["content"]; ok {
		appendNonEmpty(&parts, extractReasoningFromContentParts(content))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func extractReasoningFromContentParts(content any) string {
	items, ok := content.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		itemType := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", obj["type"])))
		if !strings.Contains(itemType, "reasoning") {
			continue
		}
		appendNonEmpty(&parts, extractReasoningText(obj["text"]))
		appendNonEmpty(&parts, extractReasoningText(obj["content"]))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func extractReasoningText(v any) string {
	switch vv := v.(type) {
	case string:
		return strings.TrimSpace(vv)
	case []any:
		parts := make([]string, 0, len(vv))
		for _, item := range vv {
			appendNonEmpty(&parts, extractReasoningText(item))
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	case map[string]any:
		parts := make([]string, 0, 4)
		appendNonEmpty(&parts, extractReasoningText(vv["text"]))
		appendNonEmpty(&parts, extractReasoningText(vv["content"]))
		appendNonEmpty(&parts, extractReasoningText(vv["summary"]))
		appendNonEmpty(&parts, extractReasoningText(vv["reasoning"]))
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func appendNonEmpty(parts *[]string, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	*parts = append(*parts, text)
}

func extractMessageContentText(content any) string {
	switch v := content.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if txt, ok := m["text"].(string); ok && strings.TrimSpace(txt) != "" {
				parts = append(parts, strings.TrimSpace(txt))
				continue
			}
			if txt, ok := m["content"].(string); ok && strings.TrimSpace(txt) != "" {
				parts = append(parts, strings.TrimSpace(txt))
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func extractExperimentError(resp any) string {
	root, ok := resp.(map[string]any)
	if !ok || root == nil {
		if text, ok := resp.(string); ok {
			return strings.TrimSpace(text)
		}
		return ""
	}

	if errObj, ok := root["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok && strings.TrimSpace(msg) != "" {
			return strings.TrimSpace(msg)
		}
	}
	if msg, ok := root["error"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	if msg, ok := root["message"].(string); ok && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	return ""
}

func experimentRouteInfo(route *types.ResolvedRoute, provider *types.Provider) map[string]any {
	out := map[string]any{
		"provider_id":     "",
		"provider_alias":  "",
		"provider_health": "",
		"requested_model": "",
		"upstream_model":  "",
		"variant":         false,
	}
	if provider != nil {
		out["provider_id"] = provider.ID
		out["provider_alias"] = provider.ID
		out["provider_health"] = provider.Health
	}
	if route != nil {
		out["requested_model"] = route.RequestedModel
		out["upstream_model"] = route.UpstreamModel
		out["variant"] = route.Variant
	}
	return out
}

func nonEmpty(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	return fallback
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Server) handleLogsAPI(w http.ResponseWriter, r *http.Request) {
	logs, err := s.store.ListRequestLogs(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": logs})
}

func (s *Server) handleAdvancedStatisticsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	now := time.Now().UTC()
	start, end, bucketSeconds, err := parseAdvancedStatisticsRange(r, now)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	windowLogsRaw, err := s.store.ListRequestLogsBetween(r.Context(), start, end, 50000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayLogsRaw, err := s.store.ListRequestLogsBetween(r.Context(), todayStart, now.Add(time.Second), 50000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	payload := advstats.AggregateRequest{
		Start:         start.Format(time.RFC3339),
		End:           end.Format(time.RFC3339),
		BucketSeconds: bucketSeconds,
		WindowLogs:    toAdvancedStatsLogs(windowLogsRaw),
		TodayLogs:     toAdvancedStatsLogs(todayLogsRaw),
	}

	moduleEnabled := s.isModuleInstalledAndEnabled(r.Context(), advancedStatisticsModuleID)
	if moduleEnabled {
		if result, err := s.callAdvancedStatisticsModule(r.Context(), payload); err == nil {
			result["source"] = "module"
			result["module"] = map[string]any{
				"id":      advancedStatisticsModuleID,
				"enabled": true,
			}
			writeJSON(w, http.StatusOK, result)
			return
		}
	}

	result := advstats.Aggregate(payload, now)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"source":         "core",
		"module":         map[string]any{"id": advancedStatisticsModuleID, "enabled": moduleEnabled},
		"start":          result.Start,
		"end":            result.End,
		"now":            result.Now,
		"bucket_seconds": result.BucketSeconds,
		"today":          result.Today,
		"window":         result.Window,
		"token_breakdown": map[string]any{
			"standard_tokens":  result.TokenBreakdown.StandardTokens,
			"reasoning_tokens": result.TokenBreakdown.ReasoningTokens,
			"cached_tokens":    result.TokenBreakdown.CachedTokens,
			"total_tokens":     result.TokenBreakdown.TotalTokens,
		},
		"model_usage":      result.ModelUsage,
		"provider_usage":   result.ProviderUsage,
		"api_usage":        result.APIUsage,
		"special_backends": result.SpecialBackends,
		"trend":            result.Trend,
	})
}

func parseAdvancedStatisticsRange(r *http.Request, now time.Time) (time.Time, time.Time, int, error) {
	q := r.URL.Query()
	days := 7
	if raw := strings.TrimSpace(q.Get("days")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid days")
		}
		if n > 90 {
			n = 90
		}
		days = n
	}

	end := now
	if raw := strings.TrimSpace(q.Get("end")); raw != "" {
		t, err := parseAdminTime(raw)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid end")
		}
		end = t
	}

	start := end.AddDate(0, 0, -days)
	if raw := strings.TrimSpace(q.Get("start")); raw != "" {
		t, err := parseAdminTime(raw)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid start")
		}
		start = t
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("start must be earlier than end")
	}

	seconds := end.Sub(start).Seconds()
	defaultBucket := 300
	if seconds > 0 {
		defaultBucket = int(math.Ceil(seconds / 96.0))
	}
	if defaultBucket < 60 {
		defaultBucket = 60
	}
	if defaultBucket > 3600 {
		defaultBucket = 3600
	}
	bucketSeconds := defaultBucket
	if raw := strings.TrimSpace(q.Get("bucket_seconds")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid bucket_seconds")
		}
		if n < 1 {
			n = 1
		}
		if n > 24*3600 {
			n = 24 * 3600
		}
		bucketSeconds = n
	}
	return start.UTC(), end.UTC(), bucketSeconds, nil
}

func parseAdminTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time format")
}

func toAdvancedStatsLogs(input []types.RequestLogMeta) []advstats.RequestLog {
	out := make([]advstats.RequestLog, 0, len(input))
	for _, row := range input {
		ts := row.Timestamp.UTC().Format(time.RFC3339)
		if row.Timestamp.IsZero() {
			ts = ""
		}
		out = append(out, advstats.RequestLog{
			Timestamp:       ts,
			ProviderID:      row.ProviderID,
			ModelID:         row.ModelID,
			Path:            row.Path,
			InputTokens:     row.InputTokens,
			OutputTokens:    row.OutputTokens,
			ReasoningTokens: row.ReasoningTokens,
			CachedTokens:    row.CachedTokens,
		})
	}
	return out
}

func (s *Server) callAdvancedStatisticsModule(ctx context.Context, payload advstats.AggregateRequest) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	status, respBody, _, err := s.proxyModuleHTTP(ctx, advancedStatisticsModuleID, http.MethodPost, "/stats/aggregate", body)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = fmt.Sprintf("module request failed (%d)", status)
		}
		return nil, errors.New(msg)
	}
	var out map[string]any
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func (s *Server) handleVoucherConfigAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.getVoucherConfig(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "config": cfg})
	case http.MethodPost:
		var cfg voucherConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if err := s.setVoucherConfig(r.Context(), cfg); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleServerAddrsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xf := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); xf != "" {
		// May contain "https,http" from some proxies.
		if p := strings.TrimSpace(strings.Split(xf, ",")[0]); p != "" {
			scheme = p
		}
	}

	host := strings.TrimSpace(r.Host)
	hostname := host
	port := ""
	if h, p, err := net.SplitHostPort(host); err == nil {
		hostname = h
		port = p
	}

	ips := make([]string, 0)
	if addrs, err := net.InterfaceAddrs(); err == nil {
		seen := map[string]struct{}{}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}
			ip := ipNet.IP
			if ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			s := ip4.String()
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			ips = append(ips, s)
		}
		sort.Strings(ips)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"scheme":   scheme,
		"host":     host,
		"hostname": hostname,
		"port":     port,
		"ips":      ips,
	})
}

func (s *Server) handleClientKeysAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys, err := s.store.ListClientKeys(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": keys})
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Key  string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = "Production"
		}
		keyValue := strings.TrimSpace(req.Key)
		if keyValue == "" {
			var err error
			keyValue, err = util.NewClientAPIKey()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
		}
		keyID, err := util.RandomToken(8)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		item := types.ClientAPIKey{
			ID:        "key_" + keyID,
			Key:       keyValue,
			Name:      name,
			Enabled:   true,
			CreatedAt: time.Now().UTC(),
		}
		if err := s.store.CreateClientKey(r.Context(), item); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "key": item})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleClientKeyEnableAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}
	if err := s.store.SetClientKeyEnabled(r.Context(), id, req.Enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleClientKeyDeleteAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}
	if err := s.store.DeleteClientKey(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMarketplaceIndexAPI(w http.ResponseWriter, r *http.Request) {
	indexURL := s.cfg.ModuleIndexURL
	if url := strings.TrimSpace(r.URL.Query().Get("url")); url != "" {
		indexURL = url
	}
	index, err := s.marketplace.FetchIndex(r.Context(), indexURL)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, index)
}

type installRequest struct {
	ModuleID string `json:"module_id"`
	IndexURL string `json:"index_url"`
}

func (s *Server) handleMarketplaceInstallAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req installRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.ModuleID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}
	indexURL := s.cfg.ModuleIndexURL
	if strings.TrimSpace(req.IndexURL) != "" {
		indexURL = req.IndexURL
	}
	index, err := s.marketplace.FetchIndex(r.Context(), indexURL)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var selected *types.ModuleEntry
	for i := range index.Modules {
		if index.Modules[i].ID == req.ModuleID {
			selected = &index.Modules[i]
			break
		}
	}
	if selected == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "module not found in index"})
		return
	}
	installed, _, err := s.marketplace.Install(r.Context(), *selected)
	if err != nil {
		code := http.StatusBadGateway
		if strings.Contains(err.Error(), "sha256") {
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]any{"error": err.Error()})
		return
	}
	var rt *types.ModuleRuntime
	if installed.Enabled {
		stopCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		_ = s.moduleMgr.StopModule(stopCtx, installed.ID)
		cancel()

		started, err := s.moduleMgr.StartInstalledModule(r.Context(), installed.ID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "installed": installed})
			return
		}
		rt = started
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "installed": installed, "runtime": rt})
}

type moduleStatus struct {
	Module    types.ModuleInstalled `json:"module"`
	Runtime   *types.ModuleRuntime  `json:"runtime,omitempty"`
	Providers []string              `json:"providers,omitempty"`
}

func (s *Server) handleModulesListAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	modules, err := s.store.ListInstalledModules(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	runtimes, err := s.store.ListModuleRuntimes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	rtByID := make(map[string]types.ModuleRuntime, len(runtimes))
	for _, rt := range runtimes {
		rtByID[rt.ModuleID] = rt
	}
	out := make([]moduleStatus, 0, len(modules))
	for _, mod := range modules {
		var rtPtr *types.ModuleRuntime
		if rt, ok := rtByID[mod.ID]; ok {
			copy := rt
			rtPtr = &copy
		}
		var providers []string
		if b, err := os.ReadFile(filepath.Join(mod.InstallPath, "manifest.json")); err == nil {
			var manifest types.ModuleManifest
			if err := json.Unmarshal(b, &manifest); err == nil {
				for alias := range exposedProviderProtocols(manifest.Services) {
					providers = append(providers, alias)
				}
				sort.Strings(providers)
			}
		}
		out = append(out, moduleStatus{Module: mod, Runtime: rtPtr, Providers: providers})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *Server) handleModuleStartAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ModuleID string `json:"module_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.ModuleID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}
	rt, err := s.moduleMgr.StartInstalledModule(r.Context(), req.ModuleID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "runtime": rt})
}

func (s *Server) handleModuleStopAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ModuleID string `json:"module_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.ModuleID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := s.moduleMgr.StopModule(stopCtx, req.ModuleID); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type moduleEnableRequest struct {
	ModuleID string `json:"module_id"`
	Enabled  bool   `json:"enabled"`
}

func exposedProviderProtocols(services []types.ManifestService) map[string]string {
	out := map[string]string{}
	for _, svc := range services {
		if svc.Kind != "provider" {
			continue
		}
		for _, alias := range svc.ExposeProviderAliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if _, ok := out[alias]; ok {
				continue
			}
			out[alias] = svc.Protocol
		}
	}
	return out
}

func (s *Server) setProviderEnabledAndHealth(ctx context.Context, id, protocol string, enabled bool, health string) error {
	existing, err := s.store.GetProvider(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if existing == nil {
		return s.store.UpsertProvider(ctx, types.Provider{
			ID:          id,
			Type:        types.ProviderTypeModule,
			Protocol:    protocol,
			Endpoint:    "",
			ConfigJSON:  "{}",
			Enabled:     enabled,
			Health:      health,
			LastCheckAt: &now,
		})
	}
	existing.Type = types.ProviderTypeModule
	if strings.TrimSpace(protocol) != "" {
		existing.Protocol = protocol
	}
	existing.Enabled = enabled
	existing.Health = health
	existing.LastCheckAt = &now
	return s.store.UpsertProvider(ctx, *existing)
}

func (s *Server) handleModuleEnableAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req moduleEnableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.ModuleID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}

	bg, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manifest, err := s.moduleMgr.LoadInstalledManifest(bg, req.ModuleID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	protos := exposedProviderProtocols(manifest.Services)

	if !req.Enabled {
		_ = s.moduleMgr.StopModule(bg, req.ModuleID)
		if err := s.store.SetModuleEnabled(bg, req.ModuleID, false); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		for alias, proto := range protos {
			_ = s.setProviderEnabledAndHealth(bg, alias, proto, false, "disabled")
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabled": false})
		return
	}

	if err := s.store.SetModuleEnabled(bg, req.ModuleID, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	for alias, proto := range protos {
		_ = s.setProviderEnabledAndHealth(bg, alias, proto, true, "down")
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabled": true})
}

func (s *Server) handleModuleManifestAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	moduleID := strings.TrimSpace(r.URL.Query().Get("module_id"))
	if moduleID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}
	manifest, err := s.moduleMgr.LoadInstalledManifest(r.Context(), moduleID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "manifest": manifest})
}

type moduleConfigUpdateRequest struct {
	ModuleID string `json:"module_id"`
	Config   any    `json:"config"`
	Restart  bool   `json:"restart"`
}

func (s *Server) handleModuleConfigAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		moduleID := strings.TrimSpace(r.URL.Query().Get("module_id"))
		if moduleID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
			return
		}
		manifest, err := s.moduleMgr.LoadInstalledManifest(r.Context(), moduleID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		cfg, err := s.moduleMgr.ReadModuleConfig(moduleID, manifest.ConfigDefaults)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":          true,
			"module_id":   moduleID,
			"config_path": s.moduleMgr.ModuleConfigPath(moduleID),
			"config":      cfg,
			"schema":      manifest.ConfigSchema,
			"defaults":    manifest.ConfigDefaults,
		})
	case http.MethodPost:
		var req moduleConfigUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(req.ModuleID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
			return
		}
		cfgObj, ok := req.Config.(map[string]any)
		if req.Config == nil {
			cfgObj = map[string]any{}
			ok = true
		}
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "config must be a JSON object"})
			return
		}

		bg, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		if err := s.moduleMgr.WriteModuleConfig(req.ModuleID, cfgObj); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		var rt *types.ModuleRuntime
		if req.Restart {
			_ = s.moduleMgr.StopModule(bg, req.ModuleID)
			started, err := s.moduleMgr.StartInstalledModule(bg, req.ModuleID)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
				return
			}
			rt = started
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "runtime": rt})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

type moduleUninstallRequest struct {
	ModuleID  string `json:"module_id"`
	PurgeData bool   `json:"purge_data"`
}

func (s *Server) handleModuleUninstallAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req moduleUninstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.ModuleID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}

	bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	manifest, err := s.moduleMgr.LoadInstalledManifest(bg, req.ModuleID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	protos := exposedProviderProtocols(manifest.Services)

	_ = s.moduleMgr.StopModule(bg, req.ModuleID)
	for alias := range protos {
		_ = s.store.DeleteProvider(bg, alias)
	}
	if err := s.store.DeleteInstalledModule(bg, req.ModuleID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	_ = os.RemoveAll(s.moduleMgr.ModuleInstallRoot(req.ModuleID))
	if req.PurgeData {
		_ = os.RemoveAll(s.moduleMgr.ModuleDataDir(req.ModuleID))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCodexOAuthCallbackPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errStr := strings.TrimSpace(q.Get("error")); errStr != "" {
		desc := strings.TrimSpace(q.Get("error_description"))
		msg := errStr
		if desc != "" {
			msg += ": " + desc
		}
		s.renderCodexOAuthCallbackResult(w, false, msg)
		return
	}

	code := strings.TrimSpace(q.Get("code"))
	state := strings.TrimSpace(q.Get("state"))
	if code == "" || state == "" {
		s.renderCodexOAuthCallbackResult(w, false, "missing code/state in callback url")
		return
	}

	payload, _ := json.Marshal(map[string]string{"code": code, "state": state})
	status, body, _, err := s.proxyModuleHTTP(r.Context(), codexOAuthModuleID, http.MethodPost, "/auth/oauth/exchange", payload)
	if err != nil {
		s.renderCodexOAuthCallbackResult(w, false, err.Error())
		return
	}
	if status < 200 || status >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("token exchange failed (%d)", status)
		}
		s.renderCodexOAuthCallbackResult(w, false, msg)
		return
	}

	s.renderCodexOAuthCallbackResult(w, true, "OAuth success. You can close this page and return to LightBridge.")
}

func (s *Server) renderCodexOAuthCallbackResult(w http.ResponseWriter, ok bool, message string) {
	s.renderCodexOAuthCallbackResultTo(w, ok, message, "/admin/providers")
}

func (s *Server) renderCodexOAuthCallbackResultTo(w http.ResponseWriter, ok bool, message string, returnURL string) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	returnURL = strings.TrimSpace(returnURL)
	if returnURL == "" {
		returnURL = "/admin/providers"
	}

	data := struct {
		Title        string
		OK           bool
		StatusCN     string
		StatusEN     string
		Headline     string
		Subtitle     string
		PrimaryLabel string
		Message      string
		ReturnURL    string
		AutoClose    bool
	}{
		Title:        "Codex OAuth",
		OK:           ok,
		StatusCN:     "认证成功",
		StatusEN:     "Successful",
		Headline:     "认证已完成",
		Subtitle:     "Token 已保存，可关闭本窗口并返回 LightBridge。",
		PrimaryLabel: "返回 Providers",
		Message:      strings.TrimSpace(message),
		ReturnURL:    returnURL,
		AutoClose:    ok,
	}
	if !ok {
		data.StatusCN = "认证失败"
		data.StatusEN = "Error"
		data.Headline = "认证未完成"
		data.Subtitle = "请返回 Providers 检查配置后重试。"
		data.PrimaryLabel = "返回 Providers 并重试"
	}

	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Title}}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=Space+Grotesk:wght@400;500;600;700&family=Noto+Sans+SC:wght@400;500;700&display=swap" rel="stylesheet" />
  <style>
    :root {
      --bg-surface: #fafafa;
      --bg-card: #ffffff;
      --border: #e8e8e8;
      --text-main: #0d0d0d;
      --text-secondary: #7a7a7a;
      --text-muted: #b0b0b0;
      --red-primary: #e42313;
      --success: #22c55e;
      --font-display: "Space Grotesk", "Noto Sans SC", sans-serif;
      --font-body: "Inter", "Noto Sans SC", sans-serif;
    }
    * { box-sizing: border-box; }
    html, body { height: 100%; margin: 0; }
    body {
      background: var(--bg-surface);
      color: var(--text-main);
      font-family: var(--font-body);
    }
    .page {
      min-height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px 16px;
    }
    .card {
      width: min(620px, calc(100vw - 32px));
      border: 1px solid var(--border);
      background: var(--bg-card);
      padding: 32px;
      display: flex;
      flex-direction: column;
      gap: 22px;
    }
    .brand {
      display: inline-flex;
      align-items: center;
      gap: 12px;
      font-family: var(--font-display);
      font-size: 20px;
      font-weight: 600;
    }
    .brand-mark {
      width: 28px;
      height: 28px;
      background: var(--red-primary);
      display: inline-block;
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      border-radius: 999px;
      width: fit-content;
      padding: 6px 12px;
      font-family: var(--font-display);
      font-size: 12px;
      font-weight: 600;
      letter-spacing: 0.02em;
    }
    .status-pill.ok {
      background: rgba(34, 197, 94, 0.1);
      color: var(--success);
    }
    .status-pill.error {
      background: rgba(228, 35, 19, 0.08);
      color: var(--red-primary);
    }
    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 999px;
      background: currentColor;
    }
    .title {
      margin: 0;
      font-family: var(--font-display);
      font-size: 30px;
      font-weight: 600;
      line-height: 1.15;
      letter-spacing: -0.5px;
    }
    .subtitle {
      margin: 8px 0 0;
      font-size: 14px;
      color: var(--text-secondary);
      line-height: 1.6;
    }
    .message-wrap {
      border: 1px solid var(--border);
      background: var(--bg-surface);
      padding: 12px 14px;
      display: grid;
      gap: 8px;
    }
    .message-label {
      margin: 0;
      font-family: var(--font-display);
      font-size: 12px;
      font-weight: 600;
      color: var(--text-secondary);
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }
    .message-body {
      margin: 0;
      font-size: 12px;
      line-height: 1.55;
      white-space: pre-wrap;
      word-break: break-word;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
      color: var(--text-main);
    }
    .tips {
      margin: 0;
      padding-left: 18px;
      display: grid;
      gap: 8px;
      color: var(--text-secondary);
      font-size: 13px;
      line-height: 1.5;
    }
    .actions {
      display: flex;
      gap: 10px;
      justify-content: flex-end;
      flex-wrap: wrap;
    }
    .btn {
      height: 40px;
      border: 1px solid var(--border);
      background: #ffffff;
      color: var(--text-main);
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 0 16px;
      font-size: 13px;
      font-family: var(--font-display);
      font-weight: 600;
      text-decoration: none;
      cursor: pointer;
    }
    .btn.primary {
      background: var(--text-main);
      border-color: var(--text-main);
      color: #ffffff;
    }
    @media (max-width: 560px) {
      .card { padding: 24px 18px; gap: 18px; }
      .title { font-size: 24px; }
      .actions { justify-content: stretch; }
      .btn { width: 100%; }
    }
  </style>
</head>
<body>
  <div class="page">
    <section class="card" aria-label="Codex OAuth Result">
      <div class="brand">
        <span class="brand-mark" aria-hidden="true"></span>
        <span>{{.Title}}</span>
      </div>
      <div class="status-pill {{if .OK}}ok{{else}}error{{end}}">
        <span class="status-dot" aria-hidden="true"></span>
        <span>{{.StatusCN}} · {{.StatusEN}}</span>
      </div>
      <div>
        <h1 class="title">{{.Headline}}</h1>
        <p class="subtitle">{{.Subtitle}}</p>
      </div>
      {{if .Message}}
      <div class="message-wrap">
        <p class="message-label">Details</p>
        <p class="message-body">{{.Message}}</p>
      </div>
      {{end}}
      <ul class="tips">
        <li>如果窗口未自动关闭，可点击下方按钮返回 Providers。</li>
        <li>返回后可在 Codex OAuth 弹窗中点击「刷新状态」确认认证是否生效。</li>
      </ul>
      <div class="actions">
        <button class="btn" type="button" onclick="window.close()">关闭窗口</button>
        <a class="btn primary" href="{{.ReturnURL}}">{{.PrimaryLabel}}</a>
      </div>
    </section>
  </div>
  <script>
    (function () {
      var shouldAutoClose = {{if .AutoClose}}true{{else}}false{{end}};
      if (!shouldAutoClose) return;
      try {
        if (window.opener) {
          setTimeout(function () { window.close(); }, 1200);
        }
      } catch (e) {}
    })();
  </script>
</body>
</html>`

	tpl, err := template.New("codex_oauth_callback").Parse(page)
	if err != nil {
		_, _ = io.WriteString(w, "<!doctype html><html><body>render failed</body></html>")
		return
	}
	_ = tpl.Execute(w, data)
}

func (s *Server) handleCodexOAuthStatusAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), codexOAuthModuleID, http.MethodGet, "/auth/status", nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handleCodexOAuthCredentialsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	credPath := filepath.Join(s.moduleMgr.ModuleDataDir(codexOAuthModuleID), "credentials.json")
	raw, err := os.ReadFile(credPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "credentials": nil})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	var creds map[string]any
	if err := json.Unmarshal(raw, &creds); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": fmt.Sprintf("decode credentials.json: %v", err)})
		return
	}
	if creds == nil {
		creds = map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "credentials": creds})
}

func (s *Server) handleCodexDeviceStartAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), codexOAuthModuleID, http.MethodPost, "/auth/device/start", nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handleCodexOAuthStartAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	s.ensureCodexOAuthCallbackServer()
	payload, _ := json.Marshal(map[string]string{"redirect_uri": codexOAuthLocalRedirectURI})
	status, body, hdr, err := s.proxyModuleHTTP(r.Context(), codexOAuthModuleID, http.MethodPost, "/auth/oauth/start", payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, body)
}

func (s *Server) handleCodexOAuthExchangeAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	_ = r.Body.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	status, respBody, hdr, err := s.proxyModuleHTTP(r.Context(), codexOAuthModuleID, http.MethodPost, "/auth/oauth/exchange", body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, respBody)
}

func (s *Server) handleCodexOAuthImportAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 6<<20))
	_ = r.Body.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	status, respBody, hdr, err := s.proxyModuleHTTP(r.Context(), codexOAuthModuleID, http.MethodPost, "/auth/import", body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeProxyResponse(w, status, hdr, respBody)
}

func baseURLFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := strings.TrimSpace(r.Host)
	if xfHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xfHost != "" {
		// X-Forwarded-Host can be a comma-separated list; first is original.
		parts := strings.Split(xfHost, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			host = strings.TrimSpace(parts[0])
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xfProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); xfProto != "" {
		parts := strings.Split(xfProto, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			scheme = strings.TrimSpace(parts[0])
		}
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return scheme + "://" + host
}

func (s *Server) ensureModuleHTTPRuntime(ctx context.Context, moduleID string) (*types.ModuleRuntime, error) {
	rt, err := s.store.GetModuleRuntime(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if rt != nil && rt.HTTPPort > 0 {
		return rt, nil
	}

	installed, err := s.store.GetInstalledModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if installed == nil {
		return nil, fmt.Errorf("module %s not installed", moduleID)
	}
	if !installed.Enabled {
		if err := s.store.SetModuleEnabled(ctx, moduleID, true); err != nil {
			return nil, err
		}
	}

	started, err := s.moduleMgr.StartInstalledModule(ctx, moduleID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already running") {
			// The module can be "already running" while the runtime record is not yet persisted.
			// Avoid returning (nil, nil) which could panic the caller; retry briefly.
			for i := 0; i < 12; i++ {
				rt, rtErr := s.store.GetModuleRuntime(ctx, moduleID)
				if rtErr != nil {
					return nil, rtErr
				}
				if rt != nil && rt.HTTPPort > 0 {
					return rt, nil
				}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(150 * time.Millisecond):
				}
			}
			return nil, fmt.Errorf("module %s runtime not ready", moduleID)
		}
		return nil, err
	}
	return started, nil
}

func (s *Server) proxyModuleHTTP(ctx context.Context, moduleID, method, endpointPath string, body []byte) (status int, respBody []byte, hdr http.Header, _ error) {
	rt, err := s.ensureModuleHTTPRuntime(ctx, moduleID)
	if err != nil {
		return 0, nil, nil, err
	}
	if rt == nil || rt.HTTPPort <= 0 {
		return 0, nil, nil, fmt.Errorf("module %s runtime not available", moduleID)
	}
	p := strings.TrimSpace(endpointPath)
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	u := fmt.Sprintf("http://127.0.0.1:%d%s", rt.HTTPPort, p)

	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("accept", "application/json")
	if method != http.MethodGet {
		req.Header.Set("content-type", "application/json")
	}

	httpc := &http.Client{Timeout: 45 * time.Second}
	resp, err := httpc.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return resp.StatusCode, b, resp.Header, nil
}

func writeProxyResponse(w http.ResponseWriter, status int, hdr http.Header, body []byte) {
	if ct := strings.TrimSpace(hdr.Get("content-type")); ct != "" {
		w.Header().Set("content-type", ct)
	} else {
		w.Header().Set("content-type", "application/json")
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (s *Server) authenticateClientKey(w http.ResponseWriter, r *http.Request) (*types.ClientAPIKey, bool) {
	token := clientTokenFromRequest(r)
	if token == "" {
		writeOpenAIError(w, http.StatusUnauthorized, "missing api key", "authentication_error", "missing_api_key")
		return nil, false
	}
	item, err := s.store.FindClientKeyByValue(r.Context(), token)
	if err != nil || item == nil || !item.Enabled {
		writeOpenAIError(w, http.StatusUnauthorized, "invalid api key", "authentication_error", "invalid_api_key")
		return nil, false
	}
	_ = s.store.TouchClientKey(r.Context(), item.ID)
	return item, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleModelDeleteAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ID  string   `json:"id"`
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	ids := req.IDs
	if len(ids) == 0 && strings.TrimSpace(req.ID) != "" {
		ids = []string{strings.TrimSpace(req.ID)}
	}
	if len(ids) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id or ids is required"})
		return
	}
	var deleted, failed int
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := s.store.DeleteModel(r.Context(), id); err != nil {
			failed++
		} else {
			deleted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted, "failed": failed})
}

func (s *Server) handleChangePasswordAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	username, ok := s.sessions.username(r)
	if !ok || strings.TrimSpace(username) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authenticated"})
		return
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "new_password is required"})
		return
	}
	hash, err := s.store.GetAdminPasswordHash(r.Context(), username)
	if err != nil || !util.CheckPassword(hash, req.OldPassword) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "old password is incorrect"})
		return
	}
	newHash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if err := s.store.UpdateAdminPassword(r.Context(), username, newHash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleLogsPruneAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	// Default: delete logs older than 30 days, keep at most 50000 rows.
	deleted, err := s.store.PruneRequestLogs(r.Context(), 30*24*time.Hour, 50000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted})
}

func (s *Server) handleModuleUpgradeAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		ModuleID string `json:"module_id"`
		IndexURL string `json:"index_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.ModuleID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "module_id is required"})
		return
	}

	// Verify the module is actually installed.
	existing, err := s.store.GetInstalledModule(r.Context(), req.ModuleID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "module not installed"})
		return
	}

	// Fetch the marketplace index to get the latest entry.
	indexURL := s.cfg.ModuleIndexURL
	if strings.TrimSpace(req.IndexURL) != "" {
		indexURL = req.IndexURL
	}
	index, err := s.marketplace.FetchIndex(r.Context(), indexURL)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	var selected *types.ModuleEntry
	for i := range index.Modules {
		if index.Modules[i].ID == req.ModuleID {
			selected = &index.Modules[i]
			break
		}
	}
	if selected == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "module not found in index"})
		return
	}

	// Stop the running module before upgrading.
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_ = s.moduleMgr.StopModule(stopCtx, req.ModuleID)
	cancel()

	// Install the new version (marketplace.Install preserves enabled state).
	installed, _, err := s.marketplace.Install(r.Context(), *selected)
	if err != nil {
		code := http.StatusBadGateway
		if strings.Contains(err.Error(), "sha256") {
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]any{"error": err.Error()})
		return
	}

	// Restart if the module was enabled.
	var rt *types.ModuleRuntime
	if installed.Enabled {
		started, err := s.moduleMgr.StartInstalledModule(r.Context(), installed.ID)
		if err != nil {
			// Upgrade succeeded but start failed — still report success with a warning.
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":        true,
				"installed": installed,
				"runtime":   nil,
				"warning":   err.Error(),
			})
			return
		}
		rt = started
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"installed": installed,
		"runtime":   rt,
	})
}
