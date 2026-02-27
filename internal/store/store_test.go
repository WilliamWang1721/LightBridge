package store

import (
	"context"
	"path/filepath"
	"testing"

	"lightbridge/internal/db"
)

func TestEnsureBuiltinProviders_SkipsRemovedBuiltinProviders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	st := New(database)

	if err := st.SetSetting(ctx, "builtin_provider_removed:forward", "1"); err != nil {
		t.Fatalf("set removed setting: %v", err)
	}
	if err := st.EnsureBuiltinProviders(ctx); err != nil {
		t.Fatalf("ensure builtin providers: %v", err)
	}

	forward, err := st.GetProvider(ctx, "forward")
	if err != nil {
		t.Fatalf("get forward provider: %v", err)
	}
	if forward != nil {
		t.Fatalf("expected forward provider to be skipped, got %+v", *forward)
	}

	anthropic, err := st.GetProvider(ctx, "anthropic")
	if err != nil {
		t.Fatalf("get anthropic provider: %v", err)
	}
	if anthropic == nil {
		t.Fatalf("expected anthropic provider to exist")
	}
}

