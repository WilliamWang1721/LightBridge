package gateway

import (
	"bufio"
	"bytes"
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

	"github.com/tidwall/gjson"
)

//go:embed web/templates/*.html web/static/*
var webFS embed.FS

type ctxKey string

const (
	ctxKeyOriginalPath        ctxKey = "original_path"
	ctxKeyAppID               ctxKey = "app_id"
	ctxKeyIngressProtocol     ctxKey = "ingress_protocol"
	ctxKeyEndpointKind        ctxKey = "endpoint_kind"
	ctxKeyForceProviderByType ctxKey = "force_provider_by_type"
)

type Config struct {
	ListenAddr      string
	ModuleIndexURL  string
	ModelTagAliases map[string]string
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
	rl        *rateLimiter

	mu sync.Mutex

	voucherMu      sync.RWMutex
	voucherCfg     voucherConfig
	voucherCfgAt   time.Time
	voucherCfgOnce bool

	authTicketMu sync.Mutex
	authTickets  map[string]authTicket

	codexOAuthCallbackMu      sync.Mutex
	codexOAuthCallbackStarted bool
	codexOAuthCallbackErr     error
}

type usageCaptureResponseWriter struct {
	http.ResponseWriter
	statusCode int
	captured   []byte
	maxCapture int
}

func newUsageCaptureResponseWriter(w http.ResponseWriter, maxCapture int) *usageCaptureResponseWriter {
	if maxCapture <= 0 {
		maxCapture = 8 << 20
	}
	return &usageCaptureResponseWriter{
		ResponseWriter: w,
		maxCapture:     maxCapture,
	}
}

func (w *usageCaptureResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *usageCaptureResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	if remain := w.maxCapture - len(w.captured); remain > 0 {
		if remain > len(p) {
			remain = len(p)
		}
		w.captured = append(w.captured, p[:remain]...)
	}
	return w.ResponseWriter.Write(p)
}

func (w *usageCaptureResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *usageCaptureResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *usageCaptureResponseWriter) CapturedBody() []byte {
	return w.captured
}

func (w *usageCaptureResponseWriter) CapturedContentType() string {
	return w.Header().Get("content-type")
}

type usageStats struct {
	InputTokens     int
	OutputTokens    int
	ReasoningTokens int
	CachedTokens    int
}

func usageFromResponse(contentType string, body []byte) usageStats {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return usageStats{}
	}

	if stats := usageFromJSON(body); stats.InputTokens > 0 || stats.OutputTokens > 0 || stats.ReasoningTokens > 0 || stats.CachedTokens > 0 {
		return stats
	}

	lowerType := strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(lowerType, "text/event-stream") || bytes.Contains(body, []byte("data:")) {
		sc := bufio.NewScanner(bytes.NewReader(body))
		sc.Buffer(make([]byte, 0, 64*1024), 8<<20)
		maxStats := usageStats{}
		for sc.Scan() {
			line := bytes.TrimSpace(sc.Bytes())
			if len(line) == 0 || !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}
			payload := bytes.TrimSpace(line[len("data:"):])
			if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) || !gjson.ValidBytes(payload) {
				continue
			}
			item := usageFromJSON(payload)
			if item.InputTokens > maxStats.InputTokens {
				maxStats.InputTokens = item.InputTokens
			}
			if item.OutputTokens > maxStats.OutputTokens {
				maxStats.OutputTokens = item.OutputTokens
			}
			if item.ReasoningTokens > maxStats.ReasoningTokens {
				maxStats.ReasoningTokens = item.ReasoningTokens
			}
			if item.CachedTokens > maxStats.CachedTokens {
				maxStats.CachedTokens = item.CachedTokens
			}
		}
		if maxStats.InputTokens > 0 || maxStats.OutputTokens > 0 || maxStats.ReasoningTokens > 0 || maxStats.CachedTokens > 0 {
			return maxStats
		}
	}

	return usageStats{}
}

func usageFromJSON(raw []byte) usageStats {
	if !gjson.ValidBytes(raw) {
		return usageStats{}
	}
	pick := func(paths ...string) int {
		for _, p := range paths {
			v := gjson.GetBytes(raw, p)
			if !v.Exists() {
				continue
			}
			n := int(v.Int())
			if n > 0 {
				return n
			}
		}
		return 0
	}

	input := pick(
		"usage.prompt_tokens",
		"usage.input_tokens",
		"response.usage.input_tokens",
		"message.usage.input_tokens",
	)
	output := pick(
		"usage.completion_tokens",
		"usage.output_tokens",
		"response.usage.output_tokens",
		"message.usage.output_tokens",
	)
	if input == 0 && output == 0 {
		total := pick(
			"usage.total_tokens",
			"response.usage.total_tokens",
			"message.usage.total_tokens",
		)
		if total > 0 {
			output = total
		}
	}
	return usageStats{
		InputTokens:  input,
		OutputTokens: output,
		ReasoningTokens: pick(
			"usage.completion_tokens_details.reasoning_tokens",
			"usage.output_tokens_details.reasoning_tokens",
			"response.usage.completion_tokens_details.reasoning_tokens",
			"response.usage.output_tokens_details.reasoning_tokens",
			"message.usage.completion_tokens_details.reasoning_tokens",
			"message.usage.output_tokens_details.reasoning_tokens",
		),
		CachedTokens: pick(
			"usage.prompt_tokens_details.cached_tokens",
			"usage.input_tokens_details.cached_tokens",
			"response.usage.prompt_tokens_details.cached_tokens",
			"response.usage.input_tokens_details.cached_tokens",
			"message.usage.prompt_tokens_details.cached_tokens",
			"message.usage.input_tokens_details.cached_tokens",
		),
	}
}

func routeModelID(route *types.ResolvedRoute, fallback string) string {
	if route != nil {
		if m := strings.TrimSpace(route.UpstreamModel); m != "" {
			return m
		}
		if m := strings.TrimSpace(route.RequestedModel); m != "" {
			return m
		}
	}
	return strings.TrimSpace(fallback)
}

func New(cfg Config, st *store.Store, resolver *routing.Resolver, providerRegistry *providers.Registry, marketplace *modules.Marketplace, moduleMgr *modules.Manager, cookieSecret string) (*Server, error) {
	cfg.ModelTagAliases = normalizeModelTagAliases(cfg.ModelTagAliases)

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
	rl := newRateLimiter(120, 120) // 120 req/min per key
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.cleanup()
		}
	}()
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
		rl:          rl,
		authTickets: map[string]authTicket{},
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/", s.handleV1Proxy)
	mux.HandleFunc("/openai/", s.handleProtocolIngress)
	mux.HandleFunc("/openai-responses/", s.handleProtocolIngress)
	mux.HandleFunc("/gemini/", s.handleProtocolIngress)
	mux.HandleFunc("/anthropic/", s.handleProtocolIngress)
	mux.HandleFunc("/claude/", s.handleProtocolIngress)
	mux.HandleFunc("/azure/openai/", s.handleProtocolIngress)

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
	return requestIDMiddleware(loggingMiddleware(rateLimitMiddleware(s.rl, mux)))
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
	s.handleProtocolIngress(w, r)
}

func (s *Server) handleGeminiAlias(w http.ResponseWriter, r *http.Request) {
	s.handleProtocolIngress(w, r)
}

func (s *Server) handleClaudeAlias(w http.ResponseWriter, r *http.Request) {
	s.handleProtocolIngress(w, r)
}

func (s *Server) handleCompatAlias(prefix string, w http.ResponseWriter, r *http.Request) {
	_ = prefix
	s.handleProtocolIngress(w, r)
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

	endpointKind := endpointKindFromContext(r.Context())
	if endpointKind == endpointKindUnknown {
		endpointKind = endpointKindFromPath(r.URL.Path)
	}
	baseModelID, tagEffort, hasModelTag, tagErr := parseModelTag(modelID, s.cfg.ModelTagAliases)
	if tagErr != nil {
		errorCode := "invalid_model_tag"
		if errors.Is(tagErr, errMissingModelTag) {
			errorCode = "missing_model"
		}
		writeOpenAIError(w, http.StatusBadRequest, tagErr.Error(), "invalid_request_error", errorCode)
		_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
			Timestamp:   time.Now().UTC(),
			RequestID:   requestID,
			ClientKeyID: client.ID,
			Path:        logPath,
			Status:      http.StatusBadRequest,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   errorCode,
		})
		return
	}
	if hasModelTag {
		modelID = baseModelID
		patchedBody, patchErr := patchReasoningEffort(bodyBytes, endpointKind, baseModelID, tagEffort, true)
		if patchErr != nil {
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
		bodyBytes = patchedBody
		r.Body = ioNopCloser(bodyBytes)
	}

	if strings.TrimSpace(modelID) == "" {
		modelID = requestModelFromPath(r.URL.Path)
	}

	if modelID != "" && appID != "" {
		modelID = s.mapModelForApp(r.Context(), appID, modelID)
	}

	ingressProtocol := ingressProtocolFromContext(r.Context())
	forceByProtocol := forceProviderByProtocol(r.Context())

	var (
		route    *types.ResolvedRoute
		err      error
		provider *types.Provider
	)
	if forceByProtocol {
		// Native protocol ingress prefers protocol-matched providers.
		// We still resolve by model first so protocol-incompatible routes/models can return
		// a clear structured not_supported error instead of ambiguous upstream failures.
		if modelID != "" {
			route, err = s.resolver.Resolve(r.Context(), modelID)
			if err != nil && errors.Is(err, routing.ErrNoHealthyProvider) {
				s.startEnabledModulesBestEffort()
				route, err = s.resolver.Resolve(r.Context(), modelID)
			}
			if err == nil && route != nil {
				provider, _ = s.store.GetProvider(r.Context(), route.ProviderID)
				if provider != nil && !supportsProtocolRoute(ingressProtocol, provider.Protocol, endpointKind) {
					writeNotSupportedRouteError(
						w,
						types.NormalizeProtocol(ingressProtocol),
						types.NormalizeProtocol(provider.Protocol),
						endpointKind,
					)
					_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
						Timestamp:   time.Now().UTC(),
						RequestID:   requestID,
						ClientKeyID: client.ID,
						ProviderID:  provider.ID,
						ModelID:     routeModelID(route, modelID),
						Path:        logPath,
						Status:      http.StatusNotImplemented,
						LatencyMS:   time.Since(start).Milliseconds(),
						ErrorCode:   "not_supported",
					})
					return
				}
			}
			if err != nil {
				route = nil
			}
		}
		if route == nil {
			provider, err = s.findHealthyProviderByProtocol(r.Context(), ingressProtocol)
			if err != nil || provider == nil {
				writeOpenAIError(w, http.StatusBadGateway, "provider unavailable", "provider_error", "provider_not_found")
				_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
					Timestamp:   time.Now().UTC(),
					RequestID:   requestID,
					ClientKeyID: client.ID,
					ModelID:     strings.TrimSpace(modelID),
					Path:        logPath,
					Status:      http.StatusBadGateway,
					LatencyMS:   time.Since(start).Milliseconds(),
					ErrorCode:   "provider_not_found",
				})
				return
			}
			route = &types.ResolvedRoute{
				RequestedModel: strings.TrimSpace(modelID),
				ProviderID:     provider.ID,
				UpstreamModel:  strings.TrimSpace(modelID),
				Variant:        true,
			}
		}
	} else if modelID == "" {
		defaultProviderID := s.defaultProviderIDForIngress(r.Context(), ingressProtocol)
		route = &types.ResolvedRoute{
			RequestedModel: "",
			ProviderID:     defaultProviderID,
			UpstreamModel:  "",
			Variant:        false,
		}
	} else {
		if types.NormalizeProtocol(ingressProtocol) != types.ProtocolOpenAI {
			if routes, e := s.store.ListModelRoutes(r.Context(), modelID, false); e == nil && len(routes) == 0 {
				if p, _ := s.findHealthyProviderByProtocol(r.Context(), ingressProtocol); p != nil {
					provider = p
					route = &types.ResolvedRoute{
						RequestedModel: strings.TrimSpace(modelID),
						ProviderID:     p.ID,
						UpstreamModel:  strings.TrimSpace(modelID),
						Variant:        false,
					}
				}
			}
		}
		if route == nil {
			route, err = s.resolver.Resolve(r.Context(), modelID)
			if err != nil && errors.Is(err, routing.ErrNoHealthyProvider) {
				// Auto-heal: if no healthy provider is available, try starting enabled modules (best-effort)
				// and resolve once more. This helps when a module-backed provider (e.g. codex) is enabled
				// but not currently running (or was restarted/crashed).
				s.startEnabledModulesBestEffort()
				route, err = s.resolver.Resolve(r.Context(), modelID)
			}
		}
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

	if provider == nil {
		provider, err = s.store.GetProvider(r.Context(), route.ProviderID)
	}
	if err != nil || provider == nil {
		writeOpenAIError(w, http.StatusBadGateway, "provider unavailable", "provider_error", "provider_not_found")
		_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
			Timestamp:   time.Now().UTC(),
			RequestID:   requestID,
			ClientKeyID: client.ID,
			ProviderID:  route.ProviderID,
			ModelID:     routeModelID(route, modelID),
			Path:        logPath,
			Status:      http.StatusBadGateway,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   "provider_not_found",
		})
		return
	}

	if !supportsProtocolRoute(ingressProtocol, provider.Protocol, endpointKind) {
		writeNotSupportedRouteError(
			w,
			types.NormalizeProtocol(ingressProtocol),
			types.NormalizeProtocol(provider.Protocol),
			endpointKind,
		)
		_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
			Timestamp:   time.Now().UTC(),
			RequestID:   requestID,
			ClientKeyID: client.ID,
			ProviderID:  provider.ID,
			ModelID:     routeModelID(route, modelID),
			Path:        logPath,
			Status:      http.StatusNotImplemented,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   "not_supported",
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
			ModelID:     routeModelID(route, modelID),
			Path:        logPath,
			Status:      http.StatusNotImplemented,
			LatencyMS:   time.Since(start).Milliseconds(),
			ErrorCode:   "provider_protocol_not_supported",
		})
		return
	}

	// Retry/failover: if upstream returns 5xx and this is not a variant (explicit provider),
	// try resolving to a different provider up to 2 times.
	const maxRetries = 2
	excludedProviders := map[string]struct{}{}
	finalStatus, finalCode := 0, ""
	finalUsage := usageStats{}
	finalModelID := routeModelID(route, modelID)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Re-resolve excluding failed providers
			excludedProviders[route.ProviderID] = struct{}{}
			newRoute, resolveErr := s.resolver.ResolveExcluding(r.Context(), modelID, excludedProviders)
			if resolveErr != nil {
				break // No more providers to try
			}
			route = newRoute
			newProvider, provErr := s.store.GetProvider(r.Context(), route.ProviderID)
			if provErr != nil || newProvider == nil {
				break
			}
			provider = newProvider
			newAdapter, adapterOk := s.providers.Get(provider.Protocol)
			if !adapterOk {
				break
			}
			adapter = newAdapter
			// Reset request body for retry
			r.Body = ioNopCloser(bodyBytes)
		}
		finalModelID = routeModelID(route, modelID)

		captureW := newUsageCaptureResponseWriter(w, 8<<20)
		var status int
		var code string
		var err error
		if shouldBridgeAnthropicMessages(ingressProtocol, endpointKind, provider.Protocol) {
			status, code, err = s.handleAnthropicMessagesBridge(r.Context(), captureW, r, adapter, *provider, route, bodyBytes)
		} else if shouldBridgeGeminiNative(ingressProtocol, endpointKind, provider.Protocol) {
			status, code, err = s.handleGeminiNativeBridge(r.Context(), captureW, r, adapter, *provider, route, bodyBytes, endpointKind)
		} else if shouldBridgeAzureLegacy(ingressProtocol, endpointKind, provider.Protocol) {
			status, code, err = s.handleAzureLegacyBridge(r.Context(), captureW, r, adapter, *provider, route, bodyBytes)
		} else {
			status, code, err = adapter.Handle(r.Context(), captureW, r, *provider, route)
		}
		finalStatus = status
		if finalStatus == 0 {
			finalStatus = captureW.StatusCode()
		}
		finalCode = code
		finalUsage = usageFromResponse(captureW.CapturedContentType(), captureW.CapturedBody())

		if err != nil {
			if status == 0 {
				finalStatus = http.StatusBadGateway
			}
			if code == "" {
				finalCode = "upstream_error"
			}
			// Only retry on 5xx errors for non-variant routes
			if finalStatus >= 500 && !route.Variant && attempt < maxRetries {
				continue
			}
			writeOpenAIError(w, finalStatus, err.Error(), "upstream_error", finalCode)
		}
		break
	}

	_ = s.store.InsertRequestLog(r.Context(), types.RequestLogMeta{
		Timestamp:       time.Now().UTC(),
		RequestID:       requestID,
		ClientKeyID:     client.ID,
		ProviderID:      route.ProviderID,
		ModelID:         finalModelID,
		Path:            logPath,
		Status:          statusOrDefault(finalStatus, http.StatusOK),
		LatencyMS:       time.Since(start).Milliseconds(),
		InputTokens:     finalUsage.InputTokens,
		OutputTokens:    finalUsage.OutputTokens,
		ReasoningTokens: finalUsage.ReasoningTokens,
		CachedTokens:    finalUsage.CachedTokens,
		ErrorCode:       finalCode,
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
	case "login/passkey":
		s.handlePasskeyLoginPage(w, r)
	case "dashboard":
		s.wrapAdminPage(s.handleDashboardPage)(w, r)
	case "settings":
		s.wrapAdminPage(s.handleSettingsPage)(w, r)
	case "settings/auth":
		s.wrapAdminPage(s.handleSettingsAuthPage)(w, r)
	case "settings/auth/passkey":
		s.wrapAdminPage(s.handleSettingsAuthPasskeyPage)(w, r)
	case "settings/auth/2fa":
		s.wrapAdminPage(s.handleSettingsAuth2FAPage)(w, r)
	case "settings/auth/password":
		s.wrapAdminPage(s.handleSettingsAuthPasswordPage)(w, r)
	case "providers", "marketplace", "logs", "docs", "auth", "router", "consumption":
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
	case "/auth/methods":
		s.handleAuthMethodsAPI(w, r)
	case "/2fa/challenge/verify":
		s.handleTwoFAChallengeVerifyAPI(w, r)
	case "/2fa/totp-only/login":
		s.handleTwoFATOTPOnlyLoginAPI(w, r)
	case "/passkey/auth/begin":
		s.handlePasskeyAuthBeginAPI(w, r)
	case "/passkey/auth/finish":
		s.handlePasskeyAuthFinishAPI(w, r)
	case "/passkey/register/begin":
		s.wrapAdminAPI(s.handlePasskeyRegisterBeginAPI)(w, r)
	case "/passkey/register/finish":
		s.wrapAdminAPI(s.handlePasskeyRegisterFinishAPI)(w, r)
	case "/passkey/credentials":
		s.wrapAdminAPI(s.handlePasskeyCredentialsAPI)(w, r)
	case "/passkey/credentials/delete":
		s.wrapAdminAPI(s.handlePasskeyCredentialDeleteAPI)(w, r)
	case "/2fa/policy":
		s.wrapAdminAPI(s.handleTwoFAPolicyAPI)(w, r)
	case "/2fa/enroll/begin":
		s.wrapAdminAPI(s.handleTwoFAEnrollBeginAPI)(w, r)
	case "/2fa/enroll/confirm":
		s.wrapAdminAPI(s.handleTwoFAEnrollConfirmAPI)(w, r)
	case "/2fa/devices":
		s.wrapAdminAPI(s.handleTwoFADevicesAPI)(w, r)
	case "/2fa/devices/delete":
		s.wrapAdminAPI(s.handleTwoFADeviceDeleteAPI)(w, r)
	case "/providers":
		s.wrapAdminAPI(s.handleProvidersAPI)(w, r)
	case "/providers/pull_models":
		s.wrapAdminAPI(s.handleProviderPullModelsAPI)(w, r)
	case "/providers/delete":
		s.wrapAdminAPI(s.handleProviderDeleteAPI)(w, r)
	case "/models":
		s.wrapAdminAPI(s.handleModelsAPI)(w, r)
	case "/models/delete":
		s.wrapAdminAPI(s.handleModelDeleteAPI)(w, r)
	case "/dashboard":
		s.wrapAdminAPI(s.handleDashboardAPI)(w, r)
	case "/advanced_statistics":
		s.wrapAdminAPI(s.handleAdvancedStatisticsAPI)(w, r)
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
	case "/codex/oauth/credentials":
		s.wrapAdminAPI(s.handleCodexOAuthCredentialsAPI)(w, r)
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
