package routing

import (
	"context"
	"math/rand"
	"testing"

	"lightbridge/internal/testutil"
	"lightbridge/internal/types"
)

func TestRouterResolveForceByProtocolPrefersProtocolProviderWhenNoRoutes(t *testing.T) {
	st, _ := testutil.NewStore(t)
	ctx := context.Background()

	router := NewRouter(st, NewResolver(st, rand.New(rand.NewSource(17))))
	decision, err := router.Resolve(ctx, DispatchRequest{
		ModelID:             "unknown-model",
		IngressProtocol:     types.ProtocolAnthropic,
		EndpointKind:        endpointKindMessages,
		ForceProviderByType: true,
	})
	if err != nil {
		t.Fatalf("resolve force-by-protocol model path: %v", err)
	}
	if decision.Provider == nil || decision.Provider.ID != "anthropic" {
		t.Fatalf("expected anthropic provider, got %#v", decision.Provider)
	}
	if decision.Route == nil || decision.Route.ProviderID != "anthropic" {
		t.Fatalf("expected anthropic route, got %#v", decision.Route)
	}
	if !decision.Route.Variant {
		t.Fatalf("expected variant route from protocol provider selection")
	}
	if decision.BridgeMode != BridgeModeNone {
		t.Fatalf("expected no bridge mode, got %s", decision.BridgeMode)
	}
}

func TestRouterResolveProtocolPreferredWhenModelHasNoRoutes(t *testing.T) {
	st, _ := testutil.NewStore(t)
	ctx := context.Background()

	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "gemini",
		Type:       types.ProviderTypeBuiltin,
		Protocol:   types.ProtocolGemini,
		Endpoint:   "https://generativelanguage.googleapis.com",
		ConfigJSON: "{}",
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert gemini provider: %v", err)
	}

	router := NewRouter(st, NewResolver(st, rand.New(rand.NewSource(23))))
	decision, err := router.Resolve(ctx, DispatchRequest{
		ModelID:         "gem-route-no-config",
		IngressProtocol: types.ProtocolGemini,
		EndpointKind:    endpointKindGenerateContent,
	})
	if err != nil {
		t.Fatalf("resolve protocol-preferred path: %v", err)
	}
	if decision.Provider == nil || decision.Provider.ID != "gemini" {
		t.Fatalf("expected gemini provider, got %#v", decision.Provider)
	}
	if decision.Route == nil || decision.Route.ProviderID != "gemini" {
		t.Fatalf("expected gemini route, got %#v", decision.Route)
	}
}

func TestRouterResolveBridgeModeForGeminiCrossProtocol(t *testing.T) {
	st, _ := testutil.NewStore(t)
	ctx := context.Background()

	if err := st.UpsertModel(ctx, types.Model{ID: "gem-route", DisplayName: "gem-route", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "gem-route", []types.ModelRoute{{
		ModelID:       "gem-route",
		ProviderID:    "forward",
		UpstreamModel: "gpt-4o-mini",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	router := NewRouter(st, NewResolver(st, rand.New(rand.NewSource(29))))
	decision, err := router.Resolve(ctx, DispatchRequest{
		ModelID:             "gem-route",
		IngressProtocol:     types.ProtocolGemini,
		EndpointKind:        endpointKindGenerateContent,
		ForceProviderByType: true,
	})
	if err != nil {
		t.Fatalf("resolve gemini cross-protocol: %v", err)
	}
	if decision.Provider == nil || decision.Provider.ID != "forward" {
		t.Fatalf("expected forward provider, got %#v", decision.Provider)
	}
	if decision.BridgeMode != BridgeModeGeminiNative {
		t.Fatalf("expected gemini bridge mode, got %s", decision.BridgeMode)
	}
}

func TestRouterResolveExcludingSkipsProvider(t *testing.T) {
	st, _ := testutil.NewStore(t)
	ctx := context.Background()

	for _, id := range []string{"p1", "p2"} {
		if err := st.UpsertProvider(ctx, types.Provider{
			ID:         id,
			Type:       types.ProviderTypeModule,
			Protocol:   types.ProtocolHTTPOpenAI,
			Endpoint:   "http://127.0.0.1:1",
			ConfigJSON: "{}",
			Enabled:    true,
			Health:     "healthy",
		}); err != nil {
			t.Fatalf("upsert provider %s: %v", id, err)
		}
	}
	if err := st.UpsertModel(ctx, types.Model{ID: "m-ex", DisplayName: "m-ex", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "m-ex", []types.ModelRoute{
		{ModelID: "m-ex", ProviderID: "p1", UpstreamModel: "m-ex", Priority: 1, Weight: 1, Enabled: true},
		{ModelID: "m-ex", ProviderID: "p2", UpstreamModel: "m-ex", Priority: 1, Weight: 1, Enabled: true},
	}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	router := NewRouter(st, NewResolver(st, rand.New(rand.NewSource(31))))
	decision, err := router.ResolveExcluding(ctx, DispatchRequest{
		ModelID:         "m-ex",
		IngressProtocol: types.ProtocolOpenAI,
		EndpointKind:    endpointKindChatCompletions,
	}, map[string]struct{}{"p1": {}})
	if err != nil {
		t.Fatalf("resolve excluding p1: %v", err)
	}
	if decision.Provider == nil || decision.Provider.ID != "p2" {
		t.Fatalf("expected provider p2, got %#v", decision.Provider)
	}
}
