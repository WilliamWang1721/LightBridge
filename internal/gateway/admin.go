package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"lightbridge/internal/types"
	"lightbridge/internal/util"
)

type adminPayload struct {
	Username string         `json:"username"`
	Password string         `json:"password"`
	Remember bool           `json:"remember"`
	Device   map[string]any `json:"device"`
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
		var payload types.Provider
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(payload.ID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provider id is required"})
			return
		}
		if payload.Type == "" {
			payload.Type = types.ProviderTypeBuiltin
		}
		if payload.Protocol == "" {
			payload.Protocol = types.ProtocolForward
		}
		if payload.ConfigJSON == "" {
			payload.ConfigJSON = "{}"
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
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": providers,
		"models":    models,
		"modules":   modules,
		"logs":      logs,
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
	rt, err := s.moduleMgr.StartInstalledModule(r.Context(), installed.ID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "installed": installed})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "installed": installed, "runtime": rt})
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
	if err := s.moduleMgr.StopModule(r.Context(), req.ModuleID); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
