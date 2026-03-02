package routing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"lightbridge/internal/store"
	"lightbridge/internal/types"
)

var ErrNoHealthyProvider = errors.New("no healthy provider available")

type Resolver struct {
	store *store.Store
	rand  *rand.Rand
}

func NewResolver(st *store.Store, rng *rand.Rand) *Resolver {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return &Resolver{store: st, rand: rng}
}

func (r *Resolver) Resolve(ctx context.Context, model string) (*types.ResolvedRoute, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, errors.New("model is required")
	}

	if base, alias, ok := splitVariant(model); ok {
		provider, err := r.store.GetProvider(ctx, alias)
		if err != nil {
			return nil, err
		}
		if provider == nil || !provider.Enabled || !isHealthy(provider.Health) {
			return nil, ErrNoHealthyProvider
		}
		routes, err := r.store.ListModelRoutes(ctx, base, false)
		if err != nil {
			return nil, err
		}
		upstream := base
		for _, route := range routes {
			if route.ProviderID == alias {
				if route.UpstreamModel != "" {
					upstream = route.UpstreamModel
				}
				break
			}
		}
		return &types.ResolvedRoute{
			RequestedModel: model,
			ProviderID:     alias,
			UpstreamModel:  upstream,
			Variant:        true,
		}, nil
	}

	routes, err := r.store.ListModelRoutes(ctx, model, false)
	if err != nil {
		return nil, err
	}
	if len(routes) > 0 {
		chosen, err := r.selectRoute(ctx, routes)
		if err != nil {
			return nil, err
		}
		upstream := chosen.UpstreamModel
		if upstream == "" {
			upstream = model
		}
		return &types.ResolvedRoute{
			RequestedModel: model,
			ProviderID:     chosen.ProviderID,
			UpstreamModel:  upstream,
			Variant:        false,
		}, nil
	}

	fallbacks := inferFallbackProviders(model)
	for _, fallback := range fallbacks {
		provider, err := r.store.GetProvider(ctx, fallback)
		if err != nil {
			return nil, err
		}
		if provider != nil && provider.Enabled && isHealthy(provider.Health) {
			return &types.ResolvedRoute{
				RequestedModel: model,
				ProviderID:     fallback,
				UpstreamModel:  model,
				Variant:        false,
			}, nil
		}
	}

	// Fallback provider unavailable — try any healthy enabled provider as last resort.
	anyProvider, err := r.findAnyHealthyProvider(ctx, nil)
	if err != nil {
		return nil, err
	}
	if anyProvider == nil {
		return nil, fmt.Errorf("fallback providers %v unavailable: %w", fallbacks, ErrNoHealthyProvider)
	}
	return &types.ResolvedRoute{
		RequestedModel: model,
		ProviderID:     anyProvider.ID,
		UpstreamModel:  model,
		Variant:        false,
	}, nil
}

func (r *Resolver) BuildModelList(ctx context.Context) ([]types.VirtualModelListing, error) {
	models, err := r.store.ListModels(ctx, false)
	if err != nil {
		return nil, err
	}
	allRoutes, err := r.store.ListAllModelRoutes(ctx, false)
	if err != nil {
		return nil, err
	}
	providers, err := r.store.ListProviders(ctx, false)
	if err != nil {
		return nil, err
	}
	providerEnabled := map[string]bool{}
	for _, p := range providers {
		providerEnabled[p.ID] = p.Enabled && isHealthy(p.Health)
	}

	now := time.Now().Unix()
	out := make([]types.VirtualModelListing, 0, len(models)+len(allRoutes))
	seen := map[string]struct{}{}

	for _, m := range models {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, types.VirtualModelListing{
			ModelID: id,
			Object:  "model",
			Created: now,
			OwnedBy: "lightbridge",
		})
		seen[id] = struct{}{}
	}

	for _, route := range allRoutes {
		if !providerEnabled[route.ProviderID] {
			continue
		}
		base := route.UpstreamModel
		if base == "" {
			base = route.ModelID
		}
		id := fmt.Sprintf("%s@%s", base, route.ProviderID)
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, types.VirtualModelListing{
			ModelID:      id,
			Object:       "model",
			Created:      now,
			OwnedBy:      "lightbridge",
			ProviderHint: route.ProviderID,
		})
		seen[id] = struct{}{}
	}

	// Also expose built-in alias variants for default no-route models.
	for _, m := range models {
		for _, p := range providers {
			if !providerEnabled[p.ID] {
				continue
			}
			id := fmt.Sprintf("%s@%s", m.ID, p.ID)
			if _, ok := seen[id]; ok {
				continue
			}
			out = append(out, types.VirtualModelListing{
				ModelID:      id,
				Object:       "model",
				Created:      now,
				OwnedBy:      "lightbridge",
				ProviderHint: p.ID,
			})
			seen[id] = struct{}{}
		}
	}
	return out, nil
}

func (r *Resolver) selectRoute(ctx context.Context, routes []types.ModelRoute) (*types.ModelRoute, error) {
	if len(routes) == 0 {
		return nil, ErrNoHealthyProvider
	}
	providers, err := r.store.ListProviders(ctx, false)
	if err != nil {
		return nil, err
	}
	providerMap := map[string]types.Provider{}
	for _, p := range providers {
		providerMap[p.ID] = p
	}

	filtered := make([]types.ModelRoute, 0, len(routes))
	minPriority := int(^uint(0) >> 1)
	for _, route := range routes {
		provider, ok := providerMap[route.ProviderID]
		if !ok || !provider.Enabled || !isHealthy(provider.Health) {
			continue
		}
		if !route.Enabled {
			continue
		}
		if route.Priority < minPriority {
			minPriority = route.Priority
		}
	}

	if minPriority == int(^uint(0)>>1) {
		return nil, ErrNoHealthyProvider
	}

	for _, route := range routes {
		provider, ok := providerMap[route.ProviderID]
		if !ok || !provider.Enabled || !isHealthy(provider.Health) {
			continue
		}
		if route.Enabled && route.Priority == minPriority {
			filtered = append(filtered, route)
		}
	}
	if len(filtered) == 0 {
		return nil, ErrNoHealthyProvider
	}

	total := 0
	for _, route := range filtered {
		w := route.Weight
		if w <= 0 {
			w = 1
		}
		total += w
	}
	if total <= 0 {
		return &filtered[0], nil
	}
	pick := r.rand.Intn(total)
	for _, route := range filtered {
		w := route.Weight
		if w <= 0 {
			w = 1
		}
		if pick < w {
			chosen := route
			return &chosen, nil
		}
		pick -= w
	}
	chosen := filtered[len(filtered)-1]
	return &chosen, nil
}

// ResolveExcluding resolves a route while excluding specific provider IDs.
// Used for retry/failover when an upstream provider returns a 5xx error.
func (r *Resolver) ResolveExcluding(ctx context.Context, model string, excludeProviders map[string]struct{}) (*types.ResolvedRoute, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, errors.New("model is required")
	}

	// Variant syntax bypasses failover (explicit provider choice)
	if _, _, ok := splitVariant(model); ok {
		return r.Resolve(ctx, model)
	}

	routes, err := r.store.ListModelRoutes(ctx, model, false)
	if err != nil {
		return nil, err
	}

	// Filter out excluded providers
	var filtered []types.ModelRoute
	for _, route := range routes {
		if _, excluded := excludeProviders[route.ProviderID]; excluded {
			continue
		}
		filtered = append(filtered, route)
	}

	if len(filtered) > 0 {
		chosen, err := r.selectRoute(ctx, filtered)
		if err != nil {
			return nil, err
		}
		upstream := chosen.UpstreamModel
		if upstream == "" {
			upstream = model
		}
		return &types.ResolvedRoute{
			RequestedModel: model,
			ProviderID:     chosen.ProviderID,
			UpstreamModel:  upstream,
			Variant:        false,
		}, nil
	}

	// Try fallback providers not in exclusion set
	for _, fallback := range inferFallbackProviders(model) {
		if _, excluded := excludeProviders[fallback]; excluded {
			continue
		}
		provider, err := r.store.GetProvider(ctx, fallback)
		if err != nil {
			return nil, err
		}
		if provider != nil && provider.Enabled && isHealthy(provider.Health) {
			return &types.ResolvedRoute{
				RequestedModel: model,
				ProviderID:     fallback,
				UpstreamModel:  model,
				Variant:        false,
			}, nil
		}
	}

	// Fallback unavailable or excluded — try any healthy provider not in exclusion set.
	anyProvider, err := r.findAnyHealthyProvider(ctx, excludeProviders)
	if err != nil {
		return nil, err
	}
	if anyProvider == nil {
		return nil, fmt.Errorf("all providers exhausted after failover: %w", ErrNoHealthyProvider)
	}
	return &types.ResolvedRoute{
		RequestedModel: model,
		ProviderID:     anyProvider.ID,
		UpstreamModel:  model,
		Variant:        false,
	}, nil
}

// inferFallbackProviders returns the best-guess provider IDs in descending preference order.
func inferFallbackProviders(model string) []string {
	lower := strings.ToLower(model)
	switch {
	case strings.HasPrefix(lower, "claude-"):
		// If Anthropic is not configured, Kiro is a common OAuth-backed Claude provider.
		return []string{"anthropic", "kiro"}
	case strings.HasPrefix(lower, "gemini-"):
		return []string{"gemini"}
	case strings.HasPrefix(lower, "gpt-"),
		strings.HasPrefix(lower, "o1-"),
		strings.HasPrefix(lower, "o3-"),
		strings.HasPrefix(lower, "o4-"),
		strings.HasPrefix(lower, "chatgpt-"):
		return []string{"codex", "forward"}
	default:
		return []string{"forward"}
	}
}

// findAnyHealthyProvider scans all enabled+healthy providers, skipping those in the exclude set.
func (r *Resolver) findAnyHealthyProvider(ctx context.Context, exclude map[string]struct{}) (*types.Provider, error) {
	providers, err := r.store.ListProviders(ctx, false)
	if err != nil {
		return nil, err
	}
	for i := range providers {
		p := &providers[i]
		if !p.Enabled || !isHealthy(p.Health) {
			continue
		}
		if exclude != nil {
			if _, skip := exclude[p.ID]; skip {
				continue
			}
		}
		return p, nil
	}
	return nil, nil
}

func splitVariant(model string) (base string, provider string, ok bool) {
	parts := strings.Split(model, "@")
	if len(parts) != 2 {
		return "", "", false
	}
	base = strings.TrimSpace(parts[0])
	provider = strings.TrimSpace(parts[1])
	if base == "" || provider == "" {
		return "", "", false
	}
	return base, provider, true
}

func isHealthy(status string) bool {
	if status == "" {
		return true
	}
	status = strings.ToLower(strings.TrimSpace(status))
	return status != "down" && status != "unhealthy" && status != "disabled"
}
