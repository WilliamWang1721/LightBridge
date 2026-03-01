package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		_ = db.Close()
		return nil, err
	}
	wrapper := &DB{DB: db}
	if err := wrapper.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return wrapper, nil
}

func (d *DB) Migrate(ctx context.Context) error {
	if _, err := d.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`); err != nil {
		return err
	}

	migrations := []struct {
		version int
		sql     string
	}{
		{1, migrationV1},
		{2, migrationV2},
		{3, migrationV3},
		{4, migrationV4},
		{5, migrationV5},
	}

	for _, m := range migrations {
		var exists int
		err := d.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", m.version).Scan(&exists)
		if err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", m.version, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES (?)", m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d mark failed: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

const migrationV1 = `
CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS admin_users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS client_api_keys (
	id TEXT PRIMARY KEY,
	key TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	last_used_at TEXT
);

CREATE TABLE IF NOT EXISTS providers (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	protocol TEXT NOT NULL,
	endpoint TEXT NOT NULL,
	config_json TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	health_status TEXT NOT NULL DEFAULT 'unknown',
	last_check_at TEXT
);

CREATE TABLE IF NOT EXISTS models (
	id TEXT PRIMARY KEY,
	display_name TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS model_routes (
	model_id TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	upstream_model TEXT NOT NULL,
	priority INTEGER NOT NULL DEFAULT 100,
	weight INTEGER NOT NULL DEFAULT 1,
	enabled INTEGER NOT NULL DEFAULT 1,
	PRIMARY KEY (model_id, provider_id, upstream_model),
	FOREIGN KEY(model_id) REFERENCES models(id) ON DELETE CASCADE,
	FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS modules_installed (
	id TEXT PRIMARY KEY,
	version TEXT NOT NULL,
	install_path TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	protocols TEXT NOT NULL,
	sha256 TEXT NOT NULL,
	installed_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS module_runtime (
	module_id TEXT PRIMARY KEY,
	pid INTEGER NOT NULL,
	http_port INTEGER NOT NULL,
	grpc_port INTEGER NOT NULL,
	status TEXT NOT NULL,
	last_start_at TEXT NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY(module_id) REFERENCES modules_installed(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS request_logs_meta (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts TEXT NOT NULL DEFAULT (datetime('now')),
	request_id TEXT NOT NULL,
	client_key_id TEXT,
	provider_id TEXT,
	model_id TEXT,
	path TEXT NOT NULL,
	status INTEGER NOT NULL,
	latency_ms INTEGER NOT NULL,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	error_code TEXT
);

CREATE INDEX IF NOT EXISTS idx_logs_ts ON request_logs_meta(ts DESC);
CREATE INDEX IF NOT EXISTS idx_logs_provider ON request_logs_meta(provider_id);
CREATE INDEX IF NOT EXISTS idx_logs_model ON request_logs_meta(model_id);
CREATE INDEX IF NOT EXISTS idx_logs_status ON request_logs_meta(status);
`

const migrationV2 = `
ALTER TABLE providers ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
`

const migrationV3 = `
ALTER TABLE providers ADD COLUMN group_name TEXT NOT NULL DEFAULT '';
`

const migrationV4 = `
ALTER TABLE request_logs_meta ADD COLUMN reasoning_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE request_logs_meta ADD COLUMN cached_tokens INTEGER NOT NULL DEFAULT 0;
`

const migrationV5 = `
CREATE TABLE IF NOT EXISTS chat_conversations (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	model_id TEXT NOT NULL,
	system_prompt TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS chat_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id TEXT NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	reasoning_text TEXT NOT NULL DEFAULT '',
	provider_id TEXT NOT NULL DEFAULT '',
	route_model TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY(conversation_id) REFERENCES chat_conversations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chat_conversations_updated_at
	ON chat_conversations(updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation
	ON chat_messages(conversation_id, id ASC);
`
