package modules

import (
	"context"
	"testing"

	"lightbridge/internal/testutil"
	"lightbridge/internal/types"
)

func TestRegisterProviderAliasesHelperUpdatesExistingProvider(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	// Simulate a previously-saved provider with a stale endpoint/health.
	if err := st.UpsertProvider(ctx, types.Provider{
		ID:         "kiro",
		Type:       types.ProviderTypeModule,
		Protocol:   types.ProtocolHTTPOpenAI,
		Endpoint:   "http://127.0.0.1:1111",
		ConfigJSON: `{"active_account_id":"a"}`,
		Enabled:    true,
		Health:     "down",
	}); err != nil {
		t.Fatalf("upsert kiro provider: %v", err)
	}

	m := NewManager(st, dir)
	services := []types.ManifestService{{
		Kind:     "provider",
		Protocol: types.ProtocolHTTPOpenAI,
		Health:   types.ManifestHealth{Type: "http", Path: "/health"},
	}}

	if err := m.registerProviderAliases(ctx, services, "kiro-oauth-provider", 43210, 0); err != nil {
		t.Fatalf("registerProviderAliases: %v", err)
	}

	got, err := st.GetProvider(ctx, "kiro")
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if got == nil {
		t.Fatalf("expected kiro provider to exist")
	}
	if got.Endpoint != "http://127.0.0.1:43210" {
		t.Fatalf("expected endpoint refreshed, got %q", got.Endpoint)
	}
	if got.Protocol != types.ProtocolHTTPOpenAI {
		t.Fatalf("expected protocol %q, got %q", types.ProtocolHTTPOpenAI, got.Protocol)
	}
	if got.Health != "healthy" {
		t.Fatalf("expected health healthy, got %q", got.Health)
	}
	// Config is user-owned and must be preserved.
	if got.ConfigJSON != `{"active_account_id":"a"}` {
		t.Fatalf("expected config preserved, got %q", got.ConfigJSON)
	}
}

func TestRegisterProviderAliasesHelperDoesNotCreateProvider(t *testing.T) {
	st, dir := testutil.NewStore(t)
	ctx := context.Background()

	m := NewManager(st, dir)
	services := []types.ManifestService{{
		Kind:     "provider",
		Protocol: types.ProtocolHTTPOpenAI,
		Health:   types.ManifestHealth{Type: "http", Path: "/health"},
	}}

	if err := m.registerProviderAliases(ctx, services, "kiro-oauth-provider", 43210, 0); err != nil {
		t.Fatalf("registerProviderAliases: %v", err)
	}

	got, err := st.GetProvider(ctx, "kiro")
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if got != nil {
		t.Fatalf("expected helper module not to auto-create provider, got %+v", *got)
	}
}
