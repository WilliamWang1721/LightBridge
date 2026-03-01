package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"lightbridge/internal/types"

	"github.com/tidwall/gjson"
)

type AzureOpenAIAdapter struct {
	client *http.Client
}

type AzureOpenAIConfig struct {
	BaseURL          string            `json:"base_url"`
	BaseOrigin       string            `json:"base_origin"`
	APIKey           string            `json:"api_key"`
	APIVersion       string            `json:"api_version"`
	DefaultInterface string            `json:"default_interface"`
	ExtraHeaders     map[string]string `json:"extra_headers"`
}

func NewAzureOpenAIAdapter(client *http.Client) *AzureOpenAIAdapter {
	if client == nil {
		client = &http.Client{}
	}
	return &AzureOpenAIAdapter{client: client}
}

func (a *AzureOpenAIAdapter) Protocol() string {
	return types.ProtocolAzureOpenAI
}

func (a *AzureOpenAIAdapter) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	cfg := a.parseConfig(provider)
	if strings.HasPrefix(req.URL.Path, "/azure/openai/deployments/") || strings.HasPrefix(req.URL.Path, "/deployments/") || strings.HasPrefix(req.URL.Path, "/openai/deployments/") {
		return a.forwardLegacy(ctx, w, req, cfg, req.URL.Path, route)
	}
	if !strings.HasPrefix(req.URL.Path, "/v1/") {
		writeOpenAIError(w, http.StatusNotImplemented, "Endpoint not supported by azure_openai provider", "not_supported", "501_not_supported")
		return http.StatusNotImplemented, "501_not_supported", nil
	}

	if strings.EqualFold(cfg.DefaultInterface, "legacy") {
		return a.forwardLegacyFromV1(ctx, w, req, cfg, route)
	}
	return a.forwardV1(ctx, w, req, cfg, route)
}

func (a *AzureOpenAIAdapter) parseConfig(provider types.Provider) AzureOpenAIConfig {
	cfg := AzureOpenAIConfig{
		APIVersion:       "2024-10-21",
		DefaultInterface: "v1",
	}
	if strings.TrimSpace(provider.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}
	if strings.TrimSpace(cfg.BaseOrigin) == "" {
		cfg.BaseOrigin = strings.TrimSpace(cfg.BaseURL)
	}
	if strings.TrimSpace(cfg.BaseOrigin) == "" {
		cfg.BaseOrigin = strings.TrimSpace(provider.Endpoint)
	}
	if strings.TrimSpace(cfg.APIVersion) == "" {
		cfg.APIVersion = "2024-10-21"
	}
	if strings.TrimSpace(cfg.DefaultInterface) == "" {
		cfg.DefaultInterface = "v1"
	}
	return cfg
}

func (a *AzureOpenAIAdapter) forwardV1(ctx context.Context, w http.ResponseWriter, req *http.Request, cfg AzureOpenAIConfig, route *types.ResolvedRoute) (int, string, error) {
	targetPath := "/openai" + req.URL.Path
	targetURL, err := joinPathURL(cfg.BaseOrigin, targetPath)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	targetURL.RawQuery = req.URL.RawQuery
	return a.forward(ctx, w, req, cfg, route, targetURL)
}

func (a *AzureOpenAIAdapter) forwardLegacyFromV1(ctx context.Context, w http.ResponseWriter, req *http.Request, cfg AzureOpenAIConfig, route *types.ResolvedRoute) (int, string, error) {
	if req.URL.Path == "/v1/responses" {
		writeOpenAIError(w, http.StatusNotImplemented, "Azure legacy API does not support /responses", "not_supported", "not_supported")
		return http.StatusNotImplemented, "not_supported", nil
	}

	deployment := ""
	if route != nil {
		deployment = strings.TrimSpace(route.UpstreamModel)
	}
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	if deployment == "" {
		deployment = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	}
	if deployment == "" {
		writeOpenAIError(w, http.StatusBadRequest, "Azure legacy routing requires deployment (use model route upstream_model as deployment)", "invalid_request_error", "missing_deployment")
		return http.StatusBadRequest, "missing_deployment", nil
	}
	suffix := strings.TrimPrefix(req.URL.Path, "/v1/")
	targetPath := fmt.Sprintf("/openai/deployments/%s/%s", url.PathEscape(deployment), strings.TrimPrefix(suffix, "/"))
	targetURL, err := joinPathURL(cfg.BaseOrigin, targetPath)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	q := req.URL.Query()
	if strings.TrimSpace(q.Get("api-version")) == "" {
		q.Set("api-version", strings.TrimSpace(cfg.APIVersion))
	}
	targetURL.RawQuery = q.Encode()
	return a.forward(ctx, w, req, cfg, route, targetURL)
}

func (a *AzureOpenAIAdapter) forwardLegacy(ctx context.Context, w http.ResponseWriter, req *http.Request, cfg AzureOpenAIConfig, legacyPath string, route *types.ResolvedRoute) (int, string, error) {
	p := strings.TrimSpace(legacyPath)
	p = strings.TrimPrefix(p, "/azure")
	if !strings.HasPrefix(p, "/openai/deployments/") {
		if strings.HasPrefix(p, "/deployments/") {
			p = "/openai" + p
		}
	}
	if !strings.HasPrefix(p, "/openai/deployments/") {
		writeOpenAIError(w, http.StatusBadRequest, "invalid azure legacy path", "invalid_request_error", "invalid_legacy_path")
		return http.StatusBadRequest, "invalid_legacy_path", nil
	}
	if route != nil && strings.TrimSpace(route.UpstreamModel) != "" {
		p = rewriteAzureLegacyDeploymentPath(p, route.UpstreamModel)
	}
	targetURL, err := joinPathURL(cfg.BaseOrigin, p)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}
	q := req.URL.Query()
	if strings.TrimSpace(q.Get("api-version")) == "" {
		q.Set("api-version", strings.TrimSpace(cfg.APIVersion))
	}
	targetURL.RawQuery = q.Encode()
	return a.forward(ctx, w, req, cfg, nil, targetURL)
}

func rewriteAzureLegacyDeploymentPath(path, deployment string) string {
	deployment = strings.TrimSpace(deployment)
	if deployment == "" {
		return path
	}
	prefix := "/openai/deployments/"
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	rest := strings.TrimPrefix(path, prefix)
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return prefix + url.PathEscape(deployment)
	}
	return prefix + url.PathEscape(deployment) + rest[idx:]
}

func (a *AzureOpenAIAdapter) forward(ctx context.Context, w http.ResponseWriter, req *http.Request, cfg AzureOpenAIConfig, route *types.ResolvedRoute, targetURL *url.URL) (int, string, error) {
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	bodyBytes = rewriteModel(bodyBytes, route, nil)
	upReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}
	copyHeaders(upReq.Header, req.Header)
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		upReq.Header.Set("api-key", key)
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) != "" {
			upReq.Header.Set(k, v)
		}
	}
	upReq.Header.Del("Accept-Encoding")

	resp, err := a.client.Do(upReq)
	if err != nil {
		return http.StatusBadGateway, "upstream_unreachable", err
	}
	defer resp.Body.Close()
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return resp.StatusCode, "upstream_stream_failed", err
	}
	return resp.StatusCode, "", nil
}
