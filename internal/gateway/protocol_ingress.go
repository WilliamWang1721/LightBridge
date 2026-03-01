package gateway

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"lightbridge/internal/types"
)

const (
	endpointKindUnknown               = "unknown"
	endpointKindModels                = "models"
	endpointKindChatCompletions       = "chat_completions"
	endpointKindResponses             = "responses"
	endpointKindMessages              = "messages"
	endpointKindGenerateContent       = "generate_content"
	endpointKindStreamGenerateContent = "stream_generate_content"
	endpointKindCountTokens           = "count_tokens"
)

type ingressRoute struct {
	Protocol            string
	AppID               string
	ProxyPath           string
	EndpointKind        string
	ForceProviderByType bool
}

func (s *Server) handleProtocolIngress(w http.ResponseWriter, r *http.Request) {
	meta, ok := parseIngressRoute(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	ctx := context.WithValue(r.Context(), ctxKeyOriginalPath, r.URL.Path)
	ctx = context.WithValue(ctx, ctxKeyAppID, meta.AppID)
	ctx = context.WithValue(ctx, ctxKeyIngressProtocol, meta.Protocol)
	ctx = context.WithValue(ctx, ctxKeyEndpointKind, meta.EndpointKind)
	ctx = context.WithValue(ctx, ctxKeyForceProviderByType, meta.ForceProviderByType)

	r2 := r.Clone(ctx)
	r2.URL.Path = meta.ProxyPath

	if meta.ProxyPath == "/v1/models" {
		p := types.NormalizeProtocol(meta.Protocol)
		if p == types.ProtocolOpenAI || p == types.ProtocolOpenAIResponses {
			s.handleModelsForApp(w, r2, meta.AppID)
			return
		}
	}
	s.handleV1Proxy(w, r2)
}

func parseIngressRoute(path string) (ingressRoute, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return ingressRoute{}, false
	}

	if path == "/v1/models" || strings.HasPrefix(path, "/v1/") {
		return ingressRoute{
			Protocol:     types.ProtocolOpenAI,
			AppID:        "",
			ProxyPath:    path,
			EndpointKind: endpointKindFromPath(path),
		}, true
	}

	if meta, ok := parseV1Prefixed(path, "/openai", types.ProtocolOpenAI); ok {
		return meta, true
	}
	if meta, ok := parseV1Prefixed(path, "/openai-responses", types.ProtocolOpenAIResponses); ok {
		return meta, true
	}
	if meta, ok := parseV1Prefixed(path, "/anthropic", types.ProtocolAnthropic); ok {
		if meta.ProxyPath == "/v1/messages" {
			meta.ForceProviderByType = true
		}
		return meta, true
	}
	if meta, ok := parseV1Prefixed(path, "/claude", types.ProtocolAnthropic); ok {
		if meta.ProxyPath == "/v1/messages" {
			meta.ForceProviderByType = true
		}
		return meta, true
	}
	if meta, ok := parseGeminiPrefixed(path); ok {
		return meta, true
	}
	if meta, ok := parseAzurePrefixed(path); ok {
		return meta, true
	}
	return ingressRoute{}, false
}

func parseV1Prefixed(path, prefix, protocol string) (ingressRoute, bool) {
	rest, ok := trimPrefixPath(path, prefix)
	if !ok || rest == "" {
		return ingressRoute{}, false
	}
	parts := splitPathParts(rest)
	appID := ""
	proxy := ""

	if len(parts) >= 1 && parts[0] == "v1" {
		proxy = "/v1"
		if len(parts) > 1 {
			proxy += "/" + strings.Join(parts[1:], "/")
		}
	} else if len(parts) >= 2 && parts[1] == "v1" {
		appID = parts[0]
		proxy = "/v1"
		if len(parts) > 2 {
			proxy += "/" + strings.Join(parts[2:], "/")
		}
	} else {
		return ingressRoute{}, false
	}

	return ingressRoute{
		Protocol:     protocol,
		AppID:        appID,
		ProxyPath:    proxy,
		EndpointKind: endpointKindFromPath(proxy),
	}, true
}

func parseGeminiPrefixed(path string) (ingressRoute, bool) {
	rest, ok := trimPrefixPath(path, "/gemini")
	if !ok || rest == "" {
		return ingressRoute{}, false
	}
	parts := splitPathParts(rest)
	appID := ""

	if len(parts) >= 1 && parts[0] == "v1beta" {
		proxy := "/v1beta"
		if len(parts) > 1 {
			proxy += "/" + strings.Join(parts[1:], "/")
		}
		return ingressRoute{
			Protocol:            types.ProtocolGemini,
			AppID:               appID,
			ProxyPath:           proxy,
			EndpointKind:        endpointKindFromPath(proxy),
			ForceProviderByType: true,
		}, true
	}
	if len(parts) >= 2 && parts[1] == "v1beta" {
		appID = parts[0]
		proxy := "/v1beta"
		if len(parts) > 2 {
			proxy += "/" + strings.Join(parts[2:], "/")
		}
		return ingressRoute{
			Protocol:            types.ProtocolGemini,
			AppID:               appID,
			ProxyPath:           proxy,
			EndpointKind:        endpointKindFromPath(proxy),
			ForceProviderByType: true,
		}, true
	}

	// Backward compatibility: /gemini/v1/* behaves like OpenAI-compatible path.
	if len(parts) >= 1 && parts[0] == "v1" {
		proxy := "/v1"
		if len(parts) > 1 {
			proxy += "/" + strings.Join(parts[1:], "/")
		}
		return ingressRoute{
			Protocol:     types.ProtocolOpenAI,
			AppID:        "",
			ProxyPath:    proxy,
			EndpointKind: endpointKindFromPath(proxy),
		}, true
	}
	if len(parts) >= 2 && parts[1] == "v1" {
		appID = parts[0]
		proxy := "/v1"
		if len(parts) > 2 {
			proxy += "/" + strings.Join(parts[2:], "/")
		}
		return ingressRoute{
			Protocol:     types.ProtocolOpenAI,
			AppID:        appID,
			ProxyPath:    proxy,
			EndpointKind: endpointKindFromPath(proxy),
		}, true
	}
	return ingressRoute{}, false
}

func parseAzurePrefixed(path string) (ingressRoute, bool) {
	rest, ok := trimPrefixPath(path, "/azure/openai")
	if !ok || rest == "" {
		return ingressRoute{}, false
	}
	parts := splitPathParts(rest)
	appID := ""

	if len(parts) >= 1 && parts[0] == "v1" {
		proxy := "/v1"
		if len(parts) > 1 {
			proxy += "/" + strings.Join(parts[1:], "/")
		}
		return ingressRoute{
			Protocol:     types.ProtocolAzureOpenAI,
			AppID:        appID,
			ProxyPath:    proxy,
			EndpointKind: endpointKindFromPath(proxy),
		}, true
	}
	if len(parts) >= 2 && parts[1] == "v1" {
		appID = parts[0]
		proxy := "/v1"
		if len(parts) > 2 {
			proxy += "/" + strings.Join(parts[2:], "/")
		}
		return ingressRoute{
			Protocol:     types.ProtocolAzureOpenAI,
			AppID:        appID,
			ProxyPath:    proxy,
			EndpointKind: endpointKindFromPath(proxy),
		}, true
	}

	if len(parts) >= 1 && parts[0] == "deployments" {
		proxy := "/azure/openai/" + strings.Join(parts, "/")
		return ingressRoute{
			Protocol:            types.ProtocolAzureOpenAI,
			AppID:               appID,
			ProxyPath:           proxy,
			EndpointKind:        "azure_legacy",
			ForceProviderByType: true,
		}, true
	}
	if len(parts) >= 2 && parts[1] == "deployments" {
		appID = parts[0]
		proxy := "/azure/openai/" + strings.Join(parts[1:], "/")
		return ingressRoute{
			Protocol:            types.ProtocolAzureOpenAI,
			AppID:               appID,
			ProxyPath:           proxy,
			EndpointKind:        "azure_legacy",
			ForceProviderByType: true,
		}, true
	}

	return ingressRoute{}, false
}

func trimPrefixPath(path, prefix string) (string, bool) {
	path = strings.TrimRight(strings.TrimSpace(path), "/")
	prefix = strings.TrimRight(strings.TrimSpace(prefix), "/")
	if !strings.HasPrefix(path, prefix+"/") {
		return "", false
	}
	return strings.TrimPrefix(path, prefix+"/"), true
}

func splitPathParts(rest string) []string {
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return nil
	}
	raw := strings.Split(rest, "/")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func endpointKindFromPath(path string) string {
	switch {
	case path == "/v1/models" || path == "/v1beta/models":
		return endpointKindModels
	case path == "/v1/chat/completions":
		return endpointKindChatCompletions
	case path == "/v1/responses":
		return endpointKindResponses
	case path == "/v1/messages":
		return endpointKindMessages
	case strings.Contains(path, ":streamGenerateContent"):
		return endpointKindStreamGenerateContent
	case strings.Contains(path, ":generateContent"):
		return endpointKindGenerateContent
	case strings.Contains(path, ":countTokens"):
		return endpointKindCountTokens
	default:
		return endpointKindUnknown
	}
}

func ingressProtocolFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyIngressProtocol).(string); ok {
		return strings.TrimSpace(v)
	}
	return types.ProtocolOpenAI
}

func endpointKindFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyEndpointKind).(string); ok {
		return strings.TrimSpace(v)
	}
	return endpointKindUnknown
}

func forceProviderByProtocol(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyForceProviderByType).(bool)
	return v
}

func supportsProtocolRoute(sourceProtocol, targetProtocol, endpointKind string) bool {
	source := types.NormalizeProtocol(sourceProtocol)
	target := types.NormalizeProtocol(targetProtocol)
	kind := strings.TrimSpace(endpointKind)

	if source == types.ProtocolGemini {
		switch kind {
		case endpointKindGenerateContent, endpointKindStreamGenerateContent, endpointKindCountTokens:
			return target == types.ProtocolGemini
		}
	}
	if source == types.ProtocolAnthropic {
		if kind == endpointKindMessages {
			return target == types.ProtocolAnthropic
		}
	}
	if source == types.ProtocolAzureOpenAI {
		if strings.HasPrefix(kind, "azure_legacy") {
			return target == types.ProtocolAzureOpenAI
		}
	}
	return true
}

func requestModelFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// Gemini native: /v1beta/models/{model}:generateContent
	if strings.HasPrefix(path, "/v1beta/models/") {
		rest := strings.TrimPrefix(path, "/v1beta/models/")
		if rest != "" {
			end := strings.Index(rest, ":")
			if end < 0 {
				end = strings.Index(rest, "/")
			}
			if end < 0 {
				end = len(rest)
			}
			model := strings.TrimSpace(rest[:end])
			if model != "" {
				if decoded, err := url.PathUnescape(model); err == nil {
					return decoded
				}
				return model
			}
		}
	}

	// Azure legacy: /azure/openai/deployments/{deployment}/...
	for _, prefix := range []string{"/azure/openai/deployments/", "/openai/deployments/", "/deployments/"} {
		if strings.HasPrefix(path, prefix) {
			rest := strings.TrimPrefix(path, prefix)
			end := strings.Index(rest, "/")
			if end < 0 {
				end = len(rest)
			}
			if end > 0 {
				deployment := strings.TrimSpace(rest[:end])
				if decoded, err := url.PathUnescape(deployment); err == nil {
					return decoded
				}
				return deployment
			}
		}
	}

	return ""
}

func (s *Server) defaultProviderIDForIngress(ctx context.Context, ingressProtocol string) string {
	switch types.NormalizeProtocol(ingressProtocol) {
	case types.ProtocolGemini, types.ProtocolAnthropic, types.ProtocolOpenAIResponses, types.ProtocolAzureOpenAI:
		if p, _ := s.findHealthyProviderByProtocol(ctx, ingressProtocol); p != nil {
			return p.ID
		}
	}
	return "forward"
}

func (s *Server) findHealthyProviderByProtocol(ctx context.Context, protocol string) (*types.Provider, error) {
	protocol = types.NormalizeProtocol(protocol)
	list, err := s.store.ListProviders(ctx, false)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	for i := range list {
		p := &list[i]
		if !providerHealthy(p.Health) {
			continue
		}
		if types.NormalizeProtocol(p.Protocol) == protocol {
			return p, nil
		}
	}
	return nil, nil
}

func providerHealthy(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return true
	}
	return status != "down" && status != "unhealthy" && status != "disabled"
}
