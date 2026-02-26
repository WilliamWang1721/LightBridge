package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"lightbridge/internal/db"
	"lightbridge/internal/store"
)

func NewStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	st := store.New(database)
	ctx := context.Background()
	if err := st.EnsureBuiltinProviders(ctx); err != nil {
		t.Fatalf("ensure providers: %v", err)
	}
	if err := st.EnsureDefaultModels(ctx); err != nil {
		t.Fatalf("ensure models: %v", err)
	}
	return st, dir
}
