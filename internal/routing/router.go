package routing

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"lightbridge/internal/store"
	"lightbridge/internal/types"
)

var ErrProviderUnavailable = errors.New("provider unavailable")

type BridgeMode string

const (
	BridgeModeNone              BridgeMode = "none"
	BridgeModeAnthropicMessages BridgeMode = "anthropic_messages"
	BridgeModeGeminiNative      BridgeMode = "gemini_native"
	BridgeModeAzureLegacy       BridgeMode = "azure_legacy"
)

const (
	endpointKindChatCompletions       = "chat_completions"
	endpointKindMessages              = "messages"
	endpointKindGenerateContent       = "generate_content"
	endpointKindStreamGenerateContent = "stream_generate_content"
	endpointKindCountTokens           = "count_tokens"
)

type DispatchRequest struct {
	ModelID             string
	IngressProtocol     string
	EndpointKind        string
	ForceProviderByType bool
}

type DispatchDecision struct {
	Route           *types.ResolvedRoute
	Provider        *types.Provider
	IngressProtocol string
	EndpointKind    string
	BridgeMode      BridgeMode
}

type ProtocolRouteNotSupportedError struct {
	SourceProtocol string
	TargetProtocol string
	EndpointKind   string
}

func (e *ProtocolRouteNotSupportedError) Error() string {
	if e == nil {
		return "route is not supported for this protocol combination"
	}
	return fmt.Sprintf(
		"route is not supported for this protocol combination (source=%s target=%s kind=%s)",
		strings.TrimSpace(e.SourceProtocol),
		strings.TrimSpace(e.TargetProtocol),
		strings.TrimSpace(e.EndpointKind),
	)
}

type Router struct {
	store    *store.Store
	resolver *Resolver
}

func NewRouter(st *store.Store, resolver *Resolver) *Router {
	return &Router{
		store:    st,
		resolver: resolver,
	}
}

func (r *Router) Resolve(ctx context.Context, req DispatchRequest) (*DispatchDecision, error) {
	return r.resolve(ctx, req, nil, false)
}

func (r *Router) ResolveExcluding(ctx context.Context, req DispatchRequest, excludeProviders map[string]struct{}) (*DispatchDecision, error) {
	return r.resolve(ctx, req, excludeProviders, true)
}

func (r *Router) resolve(ctx context.Context, req DispatchRequest, excludeProviders map[string]struct{}, withExclude bool) (*DispatchDecision, error) {
	if r == nil || r.store == nil || r.resolver == nil {
		return nil, errors.New("router is not initialized")
	}

	modelID := strings.TrimSpace(req.ModelID)
	ingressProtocol := types.NormalizeProtocol(req.IngressProtocol)
	if ingressProtocol == "" {
		ingressProtocol = types.ProtocolOpenAI
	}
	endpointKind := strings.TrimSpace(req.EndpointKind)

	var (
		route    *types.ResolvedRoute
		provider *types.Provider
		err      error
	)

	if req.ForceProviderByType {
		// Native protocol ingress prefers protocol-matched providers.
		// We still resolve by model first so protocol-incompatible routes/models can return
		// a clear structured not_supported error instead of ambiguous upstream failures.
		if modelID != "" {
			shouldResolveByModel := false
			if isExplicitVariantModel(modelID) {
				shouldResolveByModel = true
			} else {
				routes, listErr := r.store.ListModelRoutes(ctx, modelID, false)
				if listErr != nil {
					return nil, listErr
				}
				shouldResolveByModel = len(routes) > 0
			}

			if shouldResolveByModel {
				if withExclude {
					route, err = r.resolver.ResolveExcluding(ctx, modelID, excludeProviders)
				} else {
					route, err = r.resolver.Resolve(ctx, modelID)
				}
			}

			if err == nil && route != nil {
				provider, err = r.store.GetProvider(ctx, route.ProviderID)
				if err != nil {
					return nil, err
				}
				if provider != nil && !SupportsProtocolRoute(ingressProtocol, provider.Protocol, endpointKind) {
					return nil, &ProtocolRouteNotSupportedError{
						SourceProtocol: ingressProtocol,
						TargetProtocol: types.NormalizeProtocol(provider.Protocol),
						EndpointKind:   endpointKind,
					}
				}
			}
			if err != nil {
				route = nil
				provider = nil
			}
		}
		if route == nil {
			provider, err = r.findHealthyProviderByProtocol(ctx, ingressProtocol, excludeProviders)
			if err != nil {
				return nil, err
			}
			if provider == nil {
				return nil, ErrProviderUnavailable
			}
			route = &types.ResolvedRoute{
				RequestedModel: modelID,
				ProviderID:     provider.ID,
				UpstreamModel:  modelID,
				Variant:        true,
			}
		}
	} else if modelID == "" {
		defaultProviderID, err := r.defaultProviderIDForIngress(ctx, ingressProtocol, excludeProviders)
		if err != nil {
			return nil, err
		}
		route = &types.ResolvedRoute{
			RequestedModel: "",
			ProviderID:     defaultProviderID,
			UpstreamModel:  "",
			Variant:        false,
		}
	} else {
		if ingressProtocol != types.ProtocolOpenAI {
			routes, err := r.store.ListModelRoutes(ctx, modelID, false)
			if err != nil {
				return nil, err
			}
			if len(routes) == 0 {
				provider, err = r.findHealthyProviderByProtocol(ctx, ingressProtocol, excludeProviders)
				if err != nil {
					return nil, err
				}
				if provider != nil {
					route = &types.ResolvedRoute{
						RequestedModel: modelID,
						ProviderID:     provider.ID,
						UpstreamModel:  modelID,
						Variant:        false,
					}
				}
			}
		}
		if route == nil {
			if withExclude {
				route, err = r.resolver.ResolveExcluding(ctx, modelID, excludeProviders)
			} else {
				route, err = r.resolver.Resolve(ctx, modelID)
			}
			if err != nil {
				return nil, err
			}
		}
	}

	if route == nil {
		return nil, ErrProviderUnavailable
	}
	if provider == nil {
		provider, err = r.store.GetProvider(ctx, route.ProviderID)
		if err != nil {
			return nil, err
		}
	}
	if provider == nil {
		return nil, ErrProviderUnavailable
	}
	if !SupportsProtocolRoute(ingressProtocol, provider.Protocol, endpointKind) {
		return nil, &ProtocolRouteNotSupportedError{
			SourceProtocol: ingressProtocol,
			TargetProtocol: types.NormalizeProtocol(provider.Protocol),
			EndpointKind:   endpointKind,
		}
	}

	return &DispatchDecision{
		Route:           route,
		Provider:        provider,
		IngressProtocol: ingressProtocol,
		EndpointKind:    endpointKind,
		BridgeMode:      SelectBridgeMode(ingressProtocol, endpointKind, provider.Protocol),
	}, nil
}

func isExplicitVariantModel(model string) bool {
	parts := strings.Split(model, "@")
	if len(parts) != 2 {
		return false
	}
	return strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != ""
}

func SupportsProtocolRoute(sourceProtocol, targetProtocol, endpointKind string) bool {
	source := types.NormalizeProtocol(sourceProtocol)
	_ = types.NormalizeProtocol(targetProtocol)
	kind := strings.TrimSpace(endpointKind)

	if source == types.ProtocolGemini {
		switch kind {
		case endpointKindGenerateContent, endpointKindStreamGenerateContent, endpointKindCountTokens:
			return true
		}
	}
	if source == types.ProtocolAnthropic {
		if kind == endpointKindMessages {
			return true
		}
	}
	if source == types.ProtocolAzureOpenAI {
		if strings.HasPrefix(kind, "azure_legacy") {
			return true
		}
	}
	return true
}

func SelectBridgeMode(sourceProtocol, endpointKind, targetProtocol string) BridgeMode {
	source := types.NormalizeProtocol(sourceProtocol)
	target := types.NormalizeProtocol(targetProtocol)
	kind := strings.TrimSpace(endpointKind)

	if source == types.ProtocolAnthropic && kind == endpointKindMessages && target != types.ProtocolAnthropic {
		return BridgeModeAnthropicMessages
	}
	if source == types.ProtocolGemini {
		switch kind {
		case endpointKindGenerateContent, endpointKindStreamGenerateContent, endpointKindCountTokens:
			if target != types.ProtocolGemini {
				return BridgeModeGeminiNative
			}
		}
	}
	if source == types.ProtocolAzureOpenAI && strings.HasPrefix(kind, "azure_legacy") && target != types.ProtocolAzureOpenAI {
		return BridgeModeAzureLegacy
	}
	return BridgeModeNone
}

func (r *Router) defaultProviderIDForIngress(ctx context.Context, ingressProtocol string, excludeProviders map[string]struct{}) (string, error) {
	switch types.NormalizeProtocol(ingressProtocol) {
	case types.ProtocolGemini, types.ProtocolAnthropic, types.ProtocolOpenAIResponses, types.ProtocolAzureOpenAI:
		if p, err := r.findHealthyProviderByProtocol(ctx, ingressProtocol, excludeProviders); err != nil {
			return "", err
		} else if p != nil {
			return p.ID, nil
		}
	}
	if excludeProviders != nil {
		if _, excluded := excludeProviders["forward"]; excluded {
			if p, err := r.findAnyHealthyProvider(ctx, excludeProviders); err != nil {
				return "", err
			} else if p != nil {
				return p.ID, nil
			}
		}
	}
	return "forward", nil
}

func (r *Router) findHealthyProviderByProtocol(ctx context.Context, protocol string, excludeProviders map[string]struct{}) (*types.Provider, error) {
	protocol = types.NormalizeProtocol(protocol)
	list, err := r.store.ListProviders(ctx, false)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	for i := range list {
		p := &list[i]
		if !isHealthy(p.Health) {
			continue
		}
		if excludeProviders != nil {
			if _, excluded := excludeProviders[p.ID]; excluded {
				continue
			}
		}
		if types.NormalizeProtocol(p.Protocol) == protocol {
			return p, nil
		}
	}
	return nil, nil
}

func (r *Router) findAnyHealthyProvider(ctx context.Context, excludeProviders map[string]struct{}) (*types.Provider, error) {
	list, err := r.store.ListProviders(ctx, false)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	for i := range list {
		p := &list[i]
		if !isHealthy(p.Health) {
			continue
		}
		if excludeProviders != nil {
			if _, excluded := excludeProviders[p.ID]; excluded {
				continue
			}
		}
		return p, nil
	}
	return nil, nil
}
