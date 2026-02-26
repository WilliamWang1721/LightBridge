package routing

import (
	"context"
	"math/rand"
	"testing"

	"lightbridge/internal/testutil"
	"lightbridge/internal/types"
)

func TestResolveVariantAndFallback(t *testing.T) {
	st, _ := testutil.NewStore(t)
	ctx := context.Background()

	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "kiro",
		Type:       types.ProviderTypeModule,
		Protocol:   types.ProtocolHTTPOpenAI,
		Endpoint:   "http://127.0.0.1:9090",
		ConfigJSON: "{}",
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := st.UpsertModel(ctx, types.Model{ID: "claude-opus-4-5", DisplayName: "Claude Opus 4.5", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "claude-opus-4-5", []types.ModelRoute{{
		ModelID:       "claude-opus-4-5",
		ProviderID:    "kiro",
		UpstreamModel: "claude-opus-4.5-20250101",
		Priority:      10,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	resolver := NewResolver(st, rand.New(rand.NewSource(1)))

	route, err := resolver.Resolve(ctx, "claude-opus-4-5@kiro")
	if err != nil {
		t.Fatalf("resolve variant: %v", err)
	}
	if route.ProviderID != "kiro" {
		t.Fatalf("expected provider kiro, got %s", route.ProviderID)
	}
	if route.UpstreamModel != "claude-opus-4.5-20250101" {
		t.Fatalf("expected upstream mapped model, got %s", route.UpstreamModel)
	}

	route, err = resolver.Resolve(ctx, "claude-3-7-sonnet")
	if err != nil {
		t.Fatalf("resolve fallback anthropic: %v", err)
	}
	if route.ProviderID != "anthropic" {
		t.Fatalf("expected anthropic fallback, got %s", route.ProviderID)
	}

	route, err = resolver.Resolve(ctx, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("resolve fallback forward: %v", err)
	}
	if route.ProviderID != "forward" {
		t.Fatalf("expected forward fallback, got %s", route.ProviderID)
	}
}

func TestResolvePriorityAndWeight(t *testing.T) {
	st, _ := testutil.NewStore(t)
	ctx := context.Background()

	for _, id := range []string{"p1", "p2", "p3"} {
		if err := st.UpsertProvider(ctx, types.Provider{
			ID:         id,
			Type:       types.ProviderTypeModule,
			Protocol:   types.ProtocolHTTPOpenAI,
			Endpoint:   "http://127.0.0.1:1",
			ConfigJSON: "{}",
			Enabled:    true,
			Health:     "healthy",
		}); err != nil {
			t.Fatalf("upsert provider: %v", err)
		}
	}
	if err := st.UpsertModel(ctx, types.Model{ID: "m1", DisplayName: "m1", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "m1", []types.ModelRoute{
		{ModelID: "m1", ProviderID: "p1", UpstreamModel: "m1", Priority: 10, Weight: 1, Enabled: true},
		{ModelID: "m1", ProviderID: "p2", UpstreamModel: "m1", Priority: 10, Weight: 9, Enabled: true},
		{ModelID: "m1", ProviderID: "p3", UpstreamModel: "m1", Priority: 5, Weight: 1, Enabled: true},
	}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}
	resolver := NewResolver(st, rand.New(rand.NewSource(2)))

	for i := 0; i < 50; i++ {
		route, err := resolver.Resolve(ctx, "m1")
		if err != nil {
			t.Fatalf("resolve m1: %v", err)
		}
		if route.ProviderID != "p3" {
			t.Fatalf("expected highest priority provider p3, got %s", route.ProviderID)
		}
	}

	if err := st.ReplaceModelRoutes(ctx, "m1", []types.ModelRoute{
		{ModelID: "m1", ProviderID: "p1", UpstreamModel: "m1", Priority: 10, Weight: 1, Enabled: true},
		{ModelID: "m1", ProviderID: "p2", UpstreamModel: "m1", Priority: 10, Weight: 9, Enabled: true},
	}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	count := map[string]int{}
	for i := 0; i < 300; i++ {
		route, err := resolver.Resolve(ctx, "m1")
		if err != nil {
			t.Fatalf("resolve weighted m1: %v", err)
		}
		count[route.ProviderID]++
	}
	if count["p2"] <= count["p1"] {
		t.Fatalf("weighted random failed: p2=%d p1=%d", count["p2"], count["p1"])
	}
}

func TestBuildModelListIncludesVariants(t *testing.T) {
	st, _ := testutil.NewStore(t)
	ctx := context.Background()

	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "modx",
		Type:       types.ProviderTypeModule,
		Protocol:   types.ProtocolHTTPOpenAI,
		Endpoint:   "http://127.0.0.1:9999",
		ConfigJSON: "{}",
		Enabled:    true,
		Health:     "healthy",
	}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := st.UpsertModel(ctx, types.Model{ID: "nova", DisplayName: "nova", Enabled: true}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := st.ReplaceModelRoutes(ctx, "nova", []types.ModelRoute{{
		ModelID:       "nova",
		ProviderID:    "modx",
		UpstreamModel: "nova-v2",
		Priority:      1,
		Weight:        1,
		Enabled:       true,
	}}); err != nil {
		t.Fatalf("replace routes: %v", err)
	}

	resolver := NewResolver(st, rand.New(rand.NewSource(3)))
	list, err := resolver.BuildModelList(ctx)
	if err != nil {
		t.Fatalf("build model list: %v", err)
	}

	foundBase := false
	foundVariant := false
	for _, m := range list {
		if m.ModelID == "nova" {
			foundBase = true
		}
		if m.ModelID == "nova-v2@modx" {
			foundVariant = true
		}
	}
	if !foundBase || !foundVariant {
		t.Fatalf("expected base and variant, got base=%v variant=%v", foundBase, foundVariant)
	}
}
