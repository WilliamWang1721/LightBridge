package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"lightbridge/internal/db"
	"lightbridge/internal/types"
)

type Store struct {
	db *db.DB
}

func New(database *db.DB) *Store {
	return &Store{db: database}
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

func (s *Store) GetSetting(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *Store) HasAdmin(ctx context.Context) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM admin_users").Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) CreateAdmin(ctx context.Context, username, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO admin_users(username, password_hash) VALUES (?, ?)`, username, passwordHash)
	return err
}

func (s *Store) GetAdminPasswordHash(ctx context.Context, username string) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM admin_users WHERE username = ?`, username).Scan(&hash)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (s *Store) CreateClientKey(ctx context.Context, key types.ClientAPIKey) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO client_api_keys(id, key, name, enabled, created_at) VALUES (?, ?, ?, ?, ?)
	`, key.ID, key.Key, key.Name, boolInt(key.Enabled), key.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) FindClientKeyByValue(ctx context.Context, value string) (*types.ClientAPIKey, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, key, name, enabled, created_at, last_used_at
		FROM client_api_keys
		WHERE key = ?
	`, value)
	var item types.ClientAPIKey
	var enabled int
	var createdAt string
	var lastUsed sql.NullString
	if err := row.Scan(&item.ID, &item.Key, &item.Name, &enabled, &createdAt, &lastUsed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item.Enabled = enabled == 1
	if ts, err := time.Parse(time.RFC3339, createdAt); err == nil {
		item.CreatedAt = ts
	}
	if lastUsed.Valid {
		if ts, err := time.Parse(time.RFC3339, lastUsed.String); err == nil {
			item.LastUsedAt = &ts
		}
	}
	return &item, nil
}

func (s *Store) TouchClientKey(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE client_api_keys SET last_used_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *Store) ListClientKeys(ctx context.Context) ([]types.ClientAPIKey, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key, name, enabled, created_at, last_used_at
		FROM client_api_keys
		ORDER BY datetime(created_at) DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.ClientAPIKey, 0)
	for rows.Next() {
		var item types.ClientAPIKey
		var enabled int
		var createdAt string
		var lastUsed sql.NullString
		if err := rows.Scan(&item.ID, &item.Key, &item.Name, &enabled, &createdAt, &lastUsed); err != nil {
			return nil, err
		}
		item.Enabled = enabled == 1
		if ts, err := time.Parse(time.RFC3339, createdAt); err == nil {
			item.CreatedAt = ts
		}
		if lastUsed.Valid {
			if ts, err := time.Parse(time.RFC3339, lastUsed.String); err == nil {
				item.LastUsedAt = &ts
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) SetClientKeyEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE client_api_keys SET enabled = ? WHERE id = ?`, boolInt(enabled), id)
	return err
}

func (s *Store) DeleteClientKey(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM client_api_keys WHERE id = ?`, id)
	return err
}

func (s *Store) UpsertProvider(ctx context.Context, p types.Provider) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO providers(id, type, protocol, endpoint, config_json, enabled, health_status, last_check_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			protocol = excluded.protocol,
			endpoint = excluded.endpoint,
			config_json = excluded.config_json,
			enabled = excluded.enabled,
			health_status = excluded.health_status,
			last_check_at = excluded.last_check_at
	`, p.ID, p.Type, p.Protocol, p.Endpoint, emptyJSON(p.ConfigJSON), boolInt(p.Enabled), defaultStatus(p.Health), formatNullTime(p.LastCheckAt))
	return err
}

func (s *Store) ListProviders(ctx context.Context, includeDisabled bool) ([]types.Provider, error) {
	query := `
		SELECT id, type, protocol, endpoint, config_json, enabled, health_status, last_check_at
		FROM providers
	`
	args := []any{}
	if !includeDisabled {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := make([]types.Provider, 0)
	for rows.Next() {
		var p types.Provider
		var enabled int
		var lastCheck sql.NullString
		if err := rows.Scan(&p.ID, &p.Type, &p.Protocol, &p.Endpoint, &p.ConfigJSON, &enabled, &p.Health, &lastCheck); err != nil {
			return nil, err
		}
		p.Enabled = enabled == 1
		if lastCheck.Valid {
			if ts, err := time.Parse(time.RFC3339, lastCheck.String); err == nil {
				p.LastCheckAt = &ts
			}
		}
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *Store) GetProvider(ctx context.Context, id string) (*types.Provider, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, type, protocol, endpoint, config_json, enabled, health_status, last_check_at
		FROM providers WHERE id = ?
	`, id)
	var p types.Provider
	var enabled int
	var lastCheck sql.NullString
	if err := row.Scan(&p.ID, &p.Type, &p.Protocol, &p.Endpoint, &p.ConfigJSON, &enabled, &p.Health, &lastCheck); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.Enabled = enabled == 1
	if lastCheck.Valid {
		if ts, err := time.Parse(time.RFC3339, lastCheck.String); err == nil {
			p.LastCheckAt = &ts
		}
	}
	return &p, nil
}

func (s *Store) UpdateProviderHealth(ctx context.Context, id, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE providers SET health_status = ?, last_check_at = ? WHERE id = ?`, status, now, id)
	return err
}

func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	return err
}

func (s *Store) UpsertModel(ctx context.Context, m types.Model) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO models(id, display_name, enabled) VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name = excluded.display_name,
			enabled = excluded.enabled
	`, m.ID, m.DisplayName, boolInt(m.Enabled))
	return err
}

func (s *Store) ListModels(ctx context.Context, includeDisabled bool) ([]types.Model, error) {
	query := `SELECT id, display_name, enabled FROM models`
	if !includeDisabled {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	models := make([]types.Model, 0)
	for rows.Next() {
		var m types.Model
		var enabled int
		if err := rows.Scan(&m.ID, &m.DisplayName, &enabled); err != nil {
			return nil, err
		}
		m.Enabled = enabled == 1
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return models, nil
}

func (s *Store) ReplaceModelRoutes(ctx context.Context, modelID string, routes []types.ModelRoute) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM model_routes WHERE model_id = ?`, modelID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, route := range routes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model_routes(model_id, provider_id, upstream_model, priority, weight, enabled)
			VALUES(?, ?, ?, ?, ?, ?)
		`, route.ModelID, route.ProviderID, route.UpstreamModel, route.Priority, route.Weight, boolInt(route.Enabled)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListModelRoutes(ctx context.Context, modelID string, includeDisabled bool) ([]types.ModelRoute, error) {
	query := `
		SELECT model_id, provider_id, upstream_model, priority, weight, enabled
		FROM model_routes
		WHERE model_id = ?
	`
	if !includeDisabled {
		query += " AND enabled = 1"
	}
	query += " ORDER BY priority ASC, provider_id ASC"
	rows, err := s.db.QueryContext(ctx, query, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]types.ModelRoute, 0)
	for rows.Next() {
		var r types.ModelRoute
		var enabled int
		if err := rows.Scan(&r.ModelID, &r.ProviderID, &r.UpstreamModel, &r.Priority, &r.Weight, &enabled); err != nil {
			return nil, err
		}
		r.Enabled = enabled == 1
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListAllModelRoutes(ctx context.Context, includeDisabled bool) ([]types.ModelRoute, error) {
	query := `
		SELECT model_id, provider_id, upstream_model, priority, weight, enabled
		FROM model_routes
	`
	if !includeDisabled {
		query += "WHERE enabled = 1 "
	}
	query += "ORDER BY model_id ASC, priority ASC, provider_id ASC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.ModelRoute, 0)
	for rows.Next() {
		var r types.ModelRoute
		var enabled int
		if err := rows.Scan(&r.ModelID, &r.ProviderID, &r.UpstreamModel, &r.Priority, &r.Weight, &enabled); err != nil {
			return nil, err
		}
		r.Enabled = enabled == 1
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) SaveInstalledModule(ctx context.Context, module types.ModuleInstalled) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO modules_installed(id, version, install_path, enabled, protocols, sha256, installed_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			version = excluded.version,
			install_path = excluded.install_path,
			enabled = excluded.enabled,
			protocols = excluded.protocols,
			sha256 = excluded.sha256,
			installed_at = excluded.installed_at
	`, module.ID, module.Version, module.InstallPath, boolInt(module.Enabled), module.Protocols, module.SHA256, module.InstalledAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) SetModuleEnabled(ctx context.Context, moduleID string, enabled bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE modules_installed SET enabled = ? WHERE id = ?`, boolInt(enabled), moduleID)
	return err
}

func (s *Store) ListInstalledModules(ctx context.Context) ([]types.ModuleInstalled, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, version, install_path, enabled, protocols, sha256, installed_at
		FROM modules_installed ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.ModuleInstalled, 0)
	for rows.Next() {
		var m types.ModuleInstalled
		var enabled int
		var installedAt string
		if err := rows.Scan(&m.ID, &m.Version, &m.InstallPath, &enabled, &m.Protocols, &m.SHA256, &installedAt); err != nil {
			return nil, err
		}
		m.Enabled = enabled == 1
		if ts, err := time.Parse(time.RFC3339, installedAt); err == nil {
			m.InstalledAt = ts
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetInstalledModule(ctx context.Context, id string) (*types.ModuleInstalled, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, version, install_path, enabled, protocols, sha256, installed_at
		FROM modules_installed WHERE id = ?
	`, id)
	var m types.ModuleInstalled
	var enabled int
	var installedAt string
	if err := row.Scan(&m.ID, &m.Version, &m.InstallPath, &enabled, &m.Protocols, &m.SHA256, &installedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m.Enabled = enabled == 1
	if ts, err := time.Parse(time.RFC3339, installedAt); err == nil {
		m.InstalledAt = ts
	}
	return &m, nil
}

func (s *Store) SaveModuleRuntime(ctx context.Context, rt types.ModuleRuntime) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO module_runtime(module_id, pid, http_port, grpc_port, status, last_start_at)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(module_id) DO UPDATE SET
			pid = excluded.pid,
			http_port = excluded.http_port,
			grpc_port = excluded.grpc_port,
			status = excluded.status,
			last_start_at = excluded.last_start_at
	`, rt.ModuleID, rt.PID, rt.HTTPPort, rt.GRPCPort, rt.Status, rt.LastStartAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) DeleteModuleRuntime(ctx context.Context, moduleID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM module_runtime WHERE module_id = ?`, moduleID)
	return err
}

func (s *Store) DeleteInstalledModule(ctx context.Context, moduleID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM modules_installed WHERE id = ?`, moduleID)
	return err
}

func (s *Store) GetModuleRuntime(ctx context.Context, moduleID string) (*types.ModuleRuntime, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT module_id, pid, http_port, grpc_port, status, last_start_at
		FROM module_runtime WHERE module_id = ?
	`, moduleID)
	var rt types.ModuleRuntime
	var lastStart string
	if err := row.Scan(&rt.ModuleID, &rt.PID, &rt.HTTPPort, &rt.GRPCPort, &rt.Status, &lastStart); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if ts, err := time.Parse(time.RFC3339, lastStart); err == nil {
		rt.LastStartAt = ts
	}
	return &rt, nil
}

func (s *Store) ListModuleRuntimes(ctx context.Context) ([]types.ModuleRuntime, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT module_id, pid, http_port, grpc_port, status, last_start_at
		FROM module_runtime
		ORDER BY module_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.ModuleRuntime, 0)
	for rows.Next() {
		var rt types.ModuleRuntime
		var lastStart string
		if err := rows.Scan(&rt.ModuleID, &rt.PID, &rt.HTTPPort, &rt.GRPCPort, &rt.Status, &lastStart); err != nil {
			return nil, err
		}
		if ts, err := time.Parse(time.RFC3339, lastStart); err == nil {
			rt.LastStartAt = ts
		}
		out = append(out, rt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) InsertRequestLog(ctx context.Context, meta types.RequestLogMeta) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO request_logs_meta(ts, request_id, client_key_id, provider_id, model_id, path, status, latency_ms, input_tokens, output_tokens, error_code)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, meta.Timestamp.UTC().Format(time.RFC3339), meta.RequestID, nullIfEmpty(meta.ClientKeyID), nullIfEmpty(meta.ProviderID), nullIfEmpty(meta.ModelID), meta.Path, meta.Status, meta.LatencyMS, meta.InputTokens, meta.OutputTokens, nullIfEmpty(meta.ErrorCode))
	return err
}

func (s *Store) ListRequestLogs(ctx context.Context, limit int) ([]types.RequestLogMeta, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, ts, request_id, COALESCE(client_key_id,''), COALESCE(provider_id,''), COALESCE(model_id,''), path, status, latency_ms, input_tokens, output_tokens, COALESCE(error_code,'')
		FROM request_logs_meta
		ORDER BY id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.RequestLogMeta, 0)
	for rows.Next() {
		var item types.RequestLogMeta
		var ts string
		if err := rows.Scan(&item.ID, &ts, &item.RequestID, &item.ClientKeyID, &item.ProviderID, &item.ModelID, &item.Path, &item.Status, &item.LatencyMS, &item.InputTokens, &item.OutputTokens, &item.ErrorCode); err != nil {
			return nil, err
		}
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			item.Timestamp = parsed
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type RequestStats struct {
	Requests     int
	InputTokens  int
	OutputTokens int
}

func (s *Store) RequestStatsSince(ctx context.Context, since time.Time) (RequestStats, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(1),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0)
		FROM request_logs_meta
		WHERE datetime(ts) >= datetime(?)
	`, since.UTC().Format(time.RFC3339))
	var stats RequestStats
	if err := row.Scan(&stats.Requests, &stats.InputTokens, &stats.OutputTokens); err != nil {
		return RequestStats{}, err
	}
	return stats, nil
}

type DailyTokenUsage struct {
	Day          string `json:"day"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

func (s *Store) TokenUsageLastNDays(ctx context.Context, startDay time.Time, days int) ([]DailyTokenUsage, error) {
	if days <= 0 {
		days = 7
	}
	start := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, days)

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			date(datetime(ts)),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0)
		FROM request_logs_meta
		WHERE datetime(ts) >= datetime(?) AND datetime(ts) < datetime(?)
		GROUP BY 1
		ORDER BY 1 ASC
	`, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type agg struct{ in, out int }
	byDay := map[string]agg{}
	for rows.Next() {
		var day string
		var in, out int
		if err := rows.Scan(&day, &in, &out); err != nil {
			return nil, err
		}
		byDay[day] = agg{in: in, out: out}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]DailyTokenUsage, 0, days)
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		sum := byDay[day]
		out = append(out, DailyTokenUsage{
			Day:          day,
			InputTokens:  sum.in,
			OutputTokens: sum.out,
		})
	}
	return out, nil
}

func (s *Store) EnsureDefaultModels(ctx context.Context) error {
	defaults := []types.Model{
		{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Enabled: true},
		{ID: "claude-3-5-sonnet", DisplayName: "Claude 3.5 Sonnet", Enabled: true},
	}
	for _, m := range defaults {
		if err := s.UpsertModel(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EnsureBuiltinProviders(ctx context.Context) error {
	defaults := []types.Provider{
		{
			ID:       "forward",
			Type:     types.ProviderTypeBuiltin,
			Protocol: types.ProtocolForward,
			Endpoint: "https://api.openai.com/v1",
			ConfigJSON: mustJSON(map[string]any{
				"base_url":      "https://api.openai.com/v1",
				"api_key":       "",
				"extra_headers": map[string]string{},
				"model_remap":   map[string]string{},
			}),
			Enabled: true,
			Health:  "unknown",
		},
		{
			ID:       "anthropic",
			Type:     types.ProviderTypeBuiltin,
			Protocol: types.ProtocolAnthropic,
			Endpoint: "https://api.anthropic.com",
			ConfigJSON: mustJSON(map[string]any{
				"base_url":       "https://api.anthropic.com",
				"api_key":        "",
				"default_models": []string{"claude-3-5-sonnet", "claude-3-5-haiku"},
			}),
			Enabled: true,
			Health:  "unknown",
		},
	}
	for _, p := range defaults {
		existing, err := s.GetProvider(ctx, p.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if err := s.UpsertProvider(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpsertModelRoutesBulk(ctx context.Context, modelRoutes map[string][]types.ModelRoute) error {
	for modelID, routes := range modelRoutes {
		if err := s.ReplaceModelRoutes(ctx, modelID, routes); err != nil {
			return fmt.Errorf("replace routes for %s: %w", modelID, err)
		}
	}
	return nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func emptyJSON(v string) string {
	if strings.TrimSpace(v) == "" {
		return "{}"
	}
	return v
}

func defaultStatus(v string) string {
	if strings.TrimSpace(v) == "" {
		return "unknown"
	}
	return v
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func formatNullTime(ts *time.Time) any {
	if ts == nil {
		return nil
	}
	return ts.UTC().Format(time.RFC3339)
}

func mustJSON(v any) string {
	encoded, _ := json.Marshal(v)
	return string(encoded)
}
