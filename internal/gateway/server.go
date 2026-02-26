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
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/", s.handleV1Proxy)

	mux.Handle("/admin/static/", http.StripPrefix("/admin/static/", http.FileServer(s.staticFS)))
	mux.HandleFunc("/admin", s.wrapAdminPage(s.handleDashboardPage))
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
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	client, ok := s.authenticateClientKey(w, r)
	if !ok {
		return
	}
	_ = client
	list, err := s.resolver.BuildModelList(r.Context())
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, err.Error(), "server_error", "model_list_error")
		return
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   list,
	})
}

func (s *Server) handleV1Proxy(w http.ResponseWriter, r *http.Request) {
	client, ok := s.authenticateClientKey(w, r)
	if !ok {
		return
	}
	start := time.Now()
	requestID := requestIDFromContext(r.Context())

	bodyBytes, modelID, readErr := readBodyAndModel(r)
	if readErr != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Body must be valid JSON", "invalid_request_error", "invalid_json")
		_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
			Timestamp:   time.Now().UTC(),
			RequestID:   requestID,
			ClientKeyID: client.ID,
			Path:        r.URL.Path,
			Status:      http.StatusBadRequest,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   "invalid_json",
		})
		return
	}
	r.Body = ioNopCloser(bodyBytes)

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
				Path:        r.URL.Path,
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
			Path:        r.URL.Path,
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
			Path:        r.URL.Path,
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
		Path:        r.URL.Path,
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
	case "providers", "models", "marketplace", "logs", "docs":
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
	case "/models":
		s.wrapAdminAPI(s.handleModelsAPI)(w, r)
	case "/dashboard":
		s.wrapAdminAPI(s.handleDashboardAPI)(w, r)
	case "/logs":
		s.wrapAdminAPI(s.handleLogsAPI)(w, r)
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
	default:
		http.NotFound(w, r)
	}
}
