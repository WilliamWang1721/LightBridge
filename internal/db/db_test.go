package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrationIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "lightbridge.db")

	first, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open first: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}

	second, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open second: %v", err)
	}
	defer second.Close()

	ctx := context.Background()
	var migrationCount int
	if err := second.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = 1").Scan(&migrationCount); err != nil {
		t.Fatalf("query migrations: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("expected single migration row for v1, got %d", migrationCount)
	}

	for _, table := range []string{
		"settings", "admin_users", "client_api_keys", "providers", "models", "model_routes",
		"modules_installed", "module_runtime", "request_logs_meta",
	} {
		var count int
		query := `SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`
		if err := second.QueryRowContext(ctx, query, table).Scan(&count); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("expected table %s to exist", table)
		}
	}
}
