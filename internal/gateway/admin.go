package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lightbridge/internal/types"
	"lightbridge/internal/util"
)

const codexOAuthModuleID = "openai-codex-oauth"

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
	s.renderPage(w, "login", map[string]any{"Page": "Admin Login"})
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
	if err := s.sessions.newSession(w, payload.Username, payload.Remember); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "next": "/admin/dashboard"})
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

type modelRoutePayload struct {
	Model  types.Model        `json:"model"`
	Routes []types.ModelRoute `json:"routes"`
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
		"tokens_7d": tokens7d,
		"now":       now.Format(time.RFC3339),
	})
}

func (s *Server) handleLogsAPI(w http.ResponseWriter, r *http.Request) {
	logs, err := s.store.ListRequestLogs(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": logs})
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
	title := "Codex OAuth"
	status := "Success"
	if !ok {
		status = "Error"
	}
	esc := template.HTMLEscapeString(strings.TrimSpace(message))
	returnURL = strings.TrimSpace(returnURL)
	if returnURL == "" {
		returnURL = "/admin/providers"
	}
	escReturnURL := template.HTMLEscapeString(returnURL)
	_, _ = io.WriteString(w, "<!doctype html><html><head><meta charset=\"utf-8\" /><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />")
	_, _ = io.WriteString(w, "<title>"+template.HTMLEscapeString(title)+"</title>")
	_, _ = io.WriteString(w, "<style>body{font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial;margin:0;padding:32px;background:#fff;color:#111} .card{max-width:720px;margin:0 auto;border:1px solid #e8e8e8;border-radius:12px;padding:18px} .muted{color:#6b7280;font-size:13px} pre{white-space:pre-wrap;word-break:break-word;background:#fafafa;border:1px solid #e5e7eb;border-radius:10px;padding:12px;font-size:12px} a{color:#111;text-decoration:none;border-bottom:1px solid #ddd} .row{display:flex;gap:12px;align-items:center;justify-content:space-between}</style></head><body>")
	_, _ = io.WriteString(w, "<div class=\"card\"><div class=\"row\"><h2 style=\"margin:0\">"+template.HTMLEscapeString(title)+"</h2><div class=\"muted\">"+template.HTMLEscapeString(status)+"</div></div>")
	_, _ = io.WriteString(w, "<p class=\"muted\" style=\"margin:10px 0 0\">You can close this window, or <a href=\""+escReturnURL+"\">return to Providers</a>.</p>")
	if esc != "" {
		_, _ = io.WriteString(w, "<pre style=\"margin:12px 0 0\">"+esc+"</pre>")
	}
	_, _ = io.WriteString(w, "</div><script>(function(){try{if(window.opener){setTimeout(function(){window.close();},800);}}catch(e){}})();</script></body></html>")
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
	token := util.ParseBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeOpenAIError(w, http.StatusUnauthorized, "missing bearer token", "authentication_error", "missing_api_key")
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
