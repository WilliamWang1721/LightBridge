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
)

type HTTPForwardAdapter struct {
	protocol string
	client   *http.Client
}

type ForwardConfig struct {
	BaseURL      string            `json:"base_url"`
	APIKey       string            `json:"api_key"`
	ExtraHeaders map[string]string `json:"extra_headers"`
	ModelRemap   map[string]string `json:"model_remap"`
}

func NewHTTPForwardAdapter(protocol string, client *http.Client) *HTTPForwardAdapter {
	if client == nil {
		client = &http.Client{}
	}
	return &HTTPForwardAdapter{protocol: protocol, client: client}
}

func (a *HTTPForwardAdapter) Protocol() string {
	return a.protocol
}

func (a *HTTPForwardAdapter) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request, provider types.Provider, route *types.ResolvedRoute) (int, string, error) {
	cfg := ForwardConfig{}
	if provider.ConfigJSON != "" {
		_ = json.Unmarshal([]byte(provider.ConfigJSON), &cfg)
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(provider.Endpoint)
	}
	if baseURL == "" {
		return http.StatusBadGateway, "provider_misconfigured", fmt.Errorf("provider %s missing endpoint", provider.ID)
	}

	targetURL, err := url.Parse(baseURL)
	if err != nil {
		return http.StatusBadGateway, "provider_misconfigured", err
	}

	relPath := req.URL.Path
	if !strings.HasPrefix(relPath, "/") {
		relPath = "/" + relPath
	}
	targetURL.Path = strings.TrimRight(targetURL.Path, "/") + relPath
	targetURL.RawQuery = req.URL.RawQuery

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
	}
	if req.Body != nil {
		_ = req.Body.Close()
	}

	bodyBytes = rewriteModel(bodyBytes, route, cfg.ModelRemap)
	upstreamReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return http.StatusBadGateway, "upstream_request_failed", err
	}

	copyHeaders(upstreamReq.Header, req.Header)
	if cfg.APIKey != "" {
		upstreamReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	for k, v := range cfg.ExtraHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		upstreamReq.Header.Set(k, v)
	}
	upstreamReq.Header.Del("Accept-Encoding")

	resp, err := a.client.Do(upstreamReq)
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

func rewriteModel(body []byte, route *types.ResolvedRoute, remap map[string]string) []byte {
	if len(body) == 0 || route == nil {
		return body
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	_, hasModel := payload["model"]
	if !hasModel {
		return body
	}
	nextModel := route.UpstreamModel
	if remap != nil {
		if mapped, ok := remap[route.RequestedModel]; ok && mapped != "" {
			nextModel = mapped
		}
	}
	if nextModel == "" {
		return body
	}
	payload["model"] = nextModel
	patched, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return patched
}

func copyHeaders(dst, src http.Header) {
	for k := range dst {
		dst.Del(k)
	}
	for k, values := range src {
		switch strings.ToLower(k) {
		case "connection", "proxy-connection", "keep-alive", "te", "trailer", "transfer-encoding", "upgrade":
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}
