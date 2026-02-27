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
	for _, v := range []int{1, 2} {
		var migrationCount int
		if err := second.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", v).Scan(&migrationCount); err != nil {
			t.Fatalf("query migrations v%d: %v", v, err)
		}
		if migrationCount != 1 {
			t.Fatalf("expected single migration row for v%d, got %d", v, migrationCount)
		}
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

	var hasDisplayName int
	if err := second.QueryRowContext(ctx, "SELECT COUNT(1) FROM pragma_table_info('providers') WHERE name = 'display_name'").Scan(&hasDisplayName); err != nil {
		t.Fatalf("check display_name column: %v", err)
	}
	if hasDisplayName != 1 {
		t.Fatalf("expected providers.display_name column to exist")
	}
}
