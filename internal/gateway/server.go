package gateway

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"lightbridge/internal/modules"
	"lightbridge/internal/providers"
	"lightbridge/internal/routing"
	"lightbridge/internal/store"
	"lightbridge/internal/types"
	"lightbridge/internal/util"
)

//go:embed web/templates/*.html web/static/*
var webFS embed.FS

type ctxKey string

const (
	ctxKeyOriginalPath ctxKey = "original_path"
	ctxKeyAppID        ctxKey = "app_id"
)

type Config struct {
	ListenAddr     string
	ModuleIndexURL string
}

type Server struct {
	cfg         Config
	store       *store.Store
	resolver    *routing.Resolver
	providers   *providers.Registry
	marketplace *modules.Marketplace
	moduleMgr   *modules.Manager

	templates *template.Template
	staticFS  http.FileSystem
	sessions  *sessionManager
	startedAt time.Time

	mu sync.Mutex

	voucherMu      sync.RWMutex
	voucherCfg     voucherConfig
	voucherCfgAt   time.Time
	voucherCfgOnce bool

	codexOAuthCallbackMu      sync.Mutex
	codexOAuthCallbackStarted bool
	codexOAuthCallbackErr     error
}

func New(cfg Config, st *store.Store, resolver *routing.Resolver, providerRegistry *providers.Registry, marketplace *modules.Marketplace, moduleMgr *modules.Manager, cookieSecret string) (*Server, error) {
	tmpl, err := template.ParseFS(webFS, "web/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	staticSub, err := fs.Sub(webFS, "web/static")
	if err != nil {
		return nil, fmt.Errorf("static fs: %w", err)
	}
	if cookieSecret == "" {
		cookieSecret, _ = util.RandomToken(32)
	}
	return &Server{
		cfg:         cfg,
		store:       st,
		resolver:    resolver,
		providers:   providerRegistry,
		marketplace: marketplace,
		moduleMgr:   moduleMgr,
		templates:   tmpl,
		staticFS:    http.FS(staticSub),
		sessions:    newSessionManager(cookieSecret),
		startedAt:   time.Now().UTC(),
		voucherCfg:  defaultVoucherConfig(),
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/", s.handleV1Proxy)
	mux.HandleFunc("/openai/", s.handleOpenAIAlias)

	mux.Handle("/admin/static/", http.StripPrefix("/admin/static/", http.FileServer(s.staticFS)))
	mux.HandleFunc("/admin", s.wrapAdminPage(s.handleDashboardPage))
	mux.HandleFunc("/admin/codex/oauth/callback", s.wrapAdminPage(s.handleCodexOAuthCallbackPage))
	mux.HandleFunc("/admin/", s.routeAdminPages)
	mux.HandleFunc("/admin/api/", s.routeAdminAPI)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	return requestIDMiddleware(loggingMiddleware(mux))
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	srv := &http.Server{
		Addr:         s.cfg.ListenAddr,
		Handler:      s.Handler(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"name":   "lightbridge",
	})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	s.handleModelsForApp(w, r, "")
}

func (s *Server) handleModelsForApp(w http.ResponseWriter, r *http.Request, appID string) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	client, ok := s.authenticateClientKey(w, r)
	if !ok {
		return
	}
	appID = strings.ToLower(strings.TrimSpace(appID))
	if appID != "" {
		cfg := s.getVoucherConfig(r.Context())
		want := strings.TrimSpace(cfg.Apps[appID].KeyID)
		if want != "" && want != client.ID {
			writeOpenAIError(w, http.StatusUnauthorized, "invalid api key", "authentication_error", "invalid_api_key")
			return
		}
	}
	list, err := s.resolver.BuildModelList(r.Context())
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, err.Error(), "server_error", "model_list_error")
		return
	}

	if appID != "" {
		cfg := s.getVoucherConfig(r.Context())
		app := cfg.Apps[appID]
		if len(app.ModelMappings) > 0 {
			seen := map[string]struct{}{}
			for _, m := range list {
				seen[m.ModelID] = struct{}{}
			}
			now := time.Now().Unix()
			for _, mm := range app.ModelMappings {
				from := strings.TrimSpace(mm.From)
				to := strings.TrimSpace(mm.To)
				if from == "" || to == "" {
					continue
				}
				if _, ok := seen[from]; ok {
					continue
				}
				seen[from] = struct{}{}
				list = append(list, types.VirtualModelListing{
					ModelID:      from,
					Object:       "model",
					Created:      now,
					OwnedBy:      "lightbridge",
					ProviderHint: "mapped->" + to,
				})
			}
		}
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   list,
	})
}

func (s *Server) handleOpenAIAlias(w http.ResponseWriter, r *http.Request) {
	// Supports:
	// - Base URL:  {origin}/openai           -> {origin}/openai/v1/*
	// - App URL:   {origin}/openai/{app}     -> {origin}/openai/{app}/v1/*
	// This is only an HTTP path prefix router; auth is still handled by the same client API keys.
	origPath := r.URL.Path
	rest := strings.TrimPrefix(origPath, "/openai/")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(rest, "/")
	appID := ""
	v1Idx := -1
	if len(parts) >= 2 && parts[0] == "v1" {
		// /openai/v1/*
		v1Idx = 0
	} else if len(parts) >= 3 && parts[1] == "v1" {
		// /openai/{app}/v1/*
		appID = parts[0]
		v1Idx = 1
	} else {
		http.NotFound(w, r)
		return
	}

	proxyPath := "/v1"
	if len(parts) > v1Idx+1 {
		proxyPath += "/" + strings.Join(parts[v1Idx+1:], "/")
	}

	ctx := context.WithValue(r.Context(), ctxKeyOriginalPath, origPath)
	ctx = context.WithValue(ctx, ctxKeyAppID, appID)
	r2 := r.Clone(ctx)
	r2.URL.Path = proxyPath

	if proxyPath == "/v1/models" {
		s.handleModelsForApp(w, r2, appID)
		return
	}
	if strings.HasPrefix(proxyPath, "/v1/") {
		s.handleV1Proxy(w, r2)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleV1Proxy(w http.ResponseWriter, r *http.Request) {
	client, ok := s.authenticateClientKey(w, r)
	if !ok {
		return
	}
	start := time.Now()
	requestID := requestIDFromContext(r.Context())
	logPath := r.URL.Path
	if v := r.Context().Value(ctxKeyOriginalPath); v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			logPath = s
		}
	}

	appID := ""
	if v := r.Context().Value(ctxKeyAppID); v != nil {
		if s, ok := v.(string); ok {
			appID = strings.ToLower(strings.TrimSpace(s))
		}
	}
	if appID != "" {
		cfg := s.getVoucherConfig(r.Context())
		want := strings.TrimSpace(cfg.Apps[appID].KeyID)
		if want != "" && want != client.ID {
			writeOpenAIError(w, http.StatusUnauthorized, "invalid api key", "authentication_error", "invalid_api_key")
			_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
				Timestamp:   time.Now().UTC(),
				RequestID:   requestID,
				ClientKeyID: client.ID,
				Path:        logPath,
				Status:      http.StatusUnauthorized,
				LatencyMS:   time.Since(start).Milliseconds(),
				ErrorCode:   "invalid_api_key",
			})
			return
		}
	}

	bodyBytes, modelID, readErr := readBodyAndModel(r)
	if readErr != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Body must be valid JSON", "invalid_request_error", "invalid_json")
		_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
			Timestamp:   time.Now().UTC(),
			RequestID:   requestID,
			ClientKeyID: client.ID,
			Path:        logPath,
			Status:      http.StatusBadRequest,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   "invalid_json",
		})
		return
	}
	r.Body = ioNopCloser(bodyBytes)

	if modelID != "" && appID != "" {
		modelID = s.mapModelForApp(r.Context(), appID, modelID)
	}

	var (
		route *types.ResolvedRoute
		err   error
	)
	if modelID == "" {
		route = &types.ResolvedRoute{
			RequestedModel: "",
			ProviderID:     "forward",
			UpstreamModel:  "",
			Variant:        false,
		}
	} else {
		route, err = s.resolver.Resolve(r.Context(), modelID)
		if err != nil {
			writeOpenAIError(w, http.StatusBadGateway, err.Error(), "routing_error", "routing_failed")
			_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
				Timestamp:   time.Now().UTC(),
				RequestID:   requestID,
				ClientKeyID: client.ID,
				ModelID:     modelID,
				Path:        logPath,
				Status:      http.StatusBadGateway,
				LatencyMS:   time.Since(start).Milliseconds(),
				ErrorCode:   "routing_failed",
			})
			return
		}
	}

	provider, err := s.store.GetProvider(r.Context(), route.ProviderID)
	if err != nil || provider == nil {
		writeOpenAIError(w, http.StatusBadGateway, "provider unavailable", "provider_error", "provider_not_found")
		_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
			Timestamp:   time.Now().UTC(),
			RequestID:   requestID,
			ClientKeyID: client.ID,
			ProviderID:  route.ProviderID,
			ModelID:     modelID,
			Path:        logPath,
			Status:      http.StatusBadGateway,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   "provider_not_found",
		})
		return
	}

	adapter, ok := s.providers.Get(provider.Protocol)
	if !ok {
		writeOpenAIError(w, http.StatusNotImplemented, "provider protocol is not supported", "not_supported", "provider_protocol_not_supported")
		_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
			Timestamp:   time.Now().UTC(),
			RequestID:   requestID,
			ClientKeyID: client.ID,
			ProviderID:  route.ProviderID,
			ModelID:     modelID,
			Path:        logPath,
			Status:      http.StatusNotImplemented,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   "provider_protocol_not_supported",
		})
		return
	}

	status, code, err := adapter.Handle(r.Context(), w, r, *provider, route)
	if err != nil {
		if status == 0 {
			status = http.StatusBadGateway
		}
		if code == "" {
			code = "upstream_error"
		}
		writeOpenAIError(w, status, err.Error(), "upstream_error", code)
	}

	_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
		Timestamp:   time.Now().UTC(),
		RequestID:   requestID,
		ClientKeyID: client.ID,
		ProviderID:  route.ProviderID,
		ModelID:     modelID,
		Path:        logPath,
		Status:      statusOrDefault(status, http.StatusOK),
		LatencyMS:   time.Since(start).Milliseconds(),
		ErrorCode:   code,
	})
}

func (s *Server) routeAdminPages(w http.ResponseWriter, r *http.Request) {
	page := strings.TrimPrefix(r.URL.Path, "/admin/")
	page = strings.Trim(page, "/")
	if page == "" {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	switch page {
	case "setup":
		s.handleSetupPage(w, r)
	case "login":
		s.handleLoginPage(w, r)
	case "dashboard":
		s.wrapAdminPage(s.handleDashboardPage)(w, r)
	case "providers", "marketplace", "logs", "docs", "auth", "router":
		s.wrapAdminPage(func(w http.ResponseWriter, r *http.Request) {
			username, _ := s.sessions.username(r)
			if strings.TrimSpace(username) == "" {
				username = "Admin"
			}
			s.renderPage(w, page, map[string]any{
				"Page":     strings.Title(page),
				"Username": username,
			})
		})(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) routeAdminAPI(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/admin/api/")
	p = path.Clean("/" + p)
	switch p {
	case "/setup":
		s.handleAdminSetupAPI(w, r)
	case "/login":
		s.handleAdminLoginAPI(w, r)
	case "/providers":
		s.wrapAdminAPI(s.handleProvidersAPI)(w, r)
	case "/providers/delete":
		s.wrapAdminAPI(s.handleProviderDeleteAPI)(w, r)
	case "/models":
		s.wrapAdminAPI(s.handleModelsAPI)(w, r)
	case "/models/delete":
		s.wrapAdminAPI(s.handleModelDeleteAPI)(w, r)
	case "/dashboard":
		s.wrapAdminAPI(s.handleDashboardAPI)(w, r)
	case "/logs":
		s.wrapAdminAPI(s.handleLogsAPI)(w, r)
	case "/logs/prune":
		s.wrapAdminAPI(s.handleLogsPruneAPI)(w, r)
	case "/change_password":
		s.wrapAdminAPI(s.handleChangePasswordAPI)(w, r)
	case "/voucher/config":
		s.wrapAdminAPI(s.handleVoucherConfigAPI)(w, r)
	case "/server_addrs":
		s.wrapAdminAPI(s.handleServerAddrsAPI)(w, r)
	case "/client_keys":
		s.wrapAdminAPI(s.handleClientKeysAPI)(w, r)
	case "/client_keys/enable":
		s.wrapAdminAPI(s.handleClientKeyEnableAPI)(w, r)
	case "/client_keys/delete":
		s.wrapAdminAPI(s.handleClientKeyDeleteAPI)(w, r)
	case "/marketplace/index":
		s.wrapAdminAPI(s.handleMarketplaceIndexAPI)(w, r)
	case "/marketplace/install":
		s.wrapAdminAPI(s.handleMarketplaceInstallAPI)(w, r)
	case "/modules":
		s.wrapAdminAPI(s.handleModulesListAPI)(w, r)
	case "/modules/start":
		s.wrapAdminAPI(s.handleModuleStartAPI)(w, r)
	case "/modules/stop":
		s.wrapAdminAPI(s.handleModuleStopAPI)(w, r)
	case "/modules/enable":
		s.wrapAdminAPI(s.handleModuleEnableAPI)(w, r)
	case "/modules/manifest":
		s.wrapAdminAPI(s.handleModuleManifestAPI)(w, r)
	case "/modules/config":
		s.wrapAdminAPI(s.handleModuleConfigAPI)(w, r)
	case "/modules/uninstall":
		s.wrapAdminAPI(s.handleModuleUninstallAPI)(w, r)
	case "/modules/upgrade":
		s.wrapAdminAPI(s.handleModuleUpgradeAPI)(w, r)
	case "/codex/oauth/status":
		s.wrapAdminAPI(s.handleCodexOAuthStatusAPI)(w, r)
	case "/codex/oauth/start":
		s.wrapAdminAPI(s.handleCodexOAuthStartAPI)(w, r)
	case "/codex/oauth/exchange":
		s.wrapAdminAPI(s.handleCodexOAuthExchangeAPI)(w, r)
	case "/codex/oauth/import":
		s.wrapAdminAPI(s.handleCodexOAuthImportAPI)(w, r)
	case "/codex/device/start":
		s.wrapAdminAPI(s.handleCodexDeviceStartAPI)(w, r)
	default:
		http.NotFound(w, r)
	}
}
