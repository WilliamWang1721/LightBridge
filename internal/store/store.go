package store

import (
	"context"
	"database/sql"
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

func (s *Store) UpdateAdminPassword(ctx context.Context, username, passwordHash string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE admin_users SET password_hash = ? WHERE username = ?`, passwordHash, username)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return fmt.Errorf("admin user not found")
	}
	return err
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
	displayName := strings.TrimSpace(p.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(p.ID)
	}
	groupName := strings.TrimSpace(p.GroupName)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO providers(id, display_name, group_name, type, protocol, endpoint, config_json, enabled, health_status, last_check_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name = excluded.display_name,
			group_name = excluded.group_name,
			type = excluded.type,
			protocol = excluded.protocol,
			endpoint = excluded.endpoint,
			config_json = excluded.config_json,
			enabled = excluded.enabled,
			health_status = excluded.health_status,
			last_check_at = excluded.last_check_at
	`, p.ID, displayName, groupName, p.Type, p.Protocol, p.Endpoint, emptyJSON(p.ConfigJSON), boolInt(p.Enabled), defaultStatus(p.Health), formatNullTime(p.LastCheckAt))
	return err
}

func (s *Store) ListProviders(ctx context.Context, includeDisabled bool) ([]types.Provider, error) {
	query := `
		SELECT id, display_name, group_name, type, protocol, endpoint, config_json, enabled, health_status, last_check_at
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
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.GroupName, &p.Type, &p.Protocol, &p.Endpoint, &p.ConfigJSON, &enabled, &p.Health, &lastCheck); err != nil {
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
		SELECT id, display_name, group_name, type, protocol, endpoint, config_json, enabled, health_status, last_check_at
		FROM providers WHERE id = ?
	`, id)
	var p types.Provider
	var enabled int
	var lastCheck sql.NullString
	if err := row.Scan(&p.ID, &p.DisplayName, &p.GroupName, &p.Type, &p.Protocol, &p.Endpoint, &p.ConfigJSON, &enabled, &p.Health, &lastCheck); err != nil {
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

func (s *Store) InsertModelsIfMissing(ctx context.Context, ids []string) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	inserted := 0
	for _, idRaw := range ids {
		id := strings.TrimSpace(idRaw)
		if id == "" {
			continue
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO models(id, display_name, enabled) VALUES (?, ?, 1)
			ON CONFLICT(id) DO NOTHING
		`, id, id)
		if err != nil {
			_ = tx.Rollback()
			return inserted, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted += int(n)
		}
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
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

func (s *Store) GetModel(ctx context.Context, id string) (*types.Model, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, display_name, enabled FROM models WHERE id = ?`, id)
	var m types.Model
	var enabled int
	if err := row.Scan(&m.ID, &m.DisplayName, &enabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m.Enabled = enabled == 1
	return &m, nil
}

func (s *Store) DeleteModel(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM models WHERE id = ?`, id)
	return err
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
		query += " WHERE enabled = 1 "
	}
	query += " ORDER BY model_id ASC, priority ASC, provider_id ASC"

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

func (s *Store) CreateChatConversation(ctx context.Context, conv types.ChatConversation) error {
	id := strings.TrimSpace(conv.ID)
	if id == "" {
		return fmt.Errorf("conversation id is required")
	}
	title := strings.TrimSpace(conv.Title)
	if title == "" {
		title = "新对话"
	}
	modelID := strings.TrimSpace(conv.ModelID)
	if modelID == "" {
		return fmt.Errorf("model id is required")
	}
	now := time.Now().UTC()
	createdAt := conv.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := conv.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chat_conversations(id, title, model_id, system_prompt, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, id, title, modelID, strings.TrimSpace(conv.SystemPrompt), createdAt.Format(time.RFC3339), updatedAt.Format(time.RFC3339))
	return err
}

func (s *Store) GetChatConversation(ctx context.Context, id string) (*types.ChatConversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT
			c.id,
			c.title,
			c.model_id,
			c.system_prompt,
			c.created_at,
			c.updated_at,
			COALESCE((
				SELECT m.content
				FROM chat_messages m
				WHERE m.conversation_id = c.id
				ORDER BY m.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT COUNT(1)
				FROM chat_messages m
				WHERE m.conversation_id = c.id
			), 0)
		FROM chat_conversations c
		WHERE c.id = ?
	`, id)
	var out types.ChatConversation
	var createdAt, updatedAt string
	if err := row.Scan(
		&out.ID,
		&out.Title,
		&out.ModelID,
		&out.SystemPrompt,
		&createdAt,
		&updatedAt,
		&out.LastMessagePreview,
		&out.MessageCount,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if ts, err := time.Parse(time.RFC3339, createdAt); err == nil {
		out.CreatedAt = ts
	}
	if ts, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		out.UpdatedAt = ts
	}
	return &out, nil
}

func (s *Store) ListChatConversations(ctx context.Context, limit int) ([]types.ChatConversation, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.title,
			c.model_id,
			c.system_prompt,
			c.created_at,
			c.updated_at,
			COALESCE((
				SELECT m.content
				FROM chat_messages m
				WHERE m.conversation_id = c.id
				ORDER BY m.id DESC
				LIMIT 1
			), ''),
			COALESCE((
				SELECT COUNT(1)
				FROM chat_messages m
				WHERE m.conversation_id = c.id
			), 0)
		FROM chat_conversations c
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.ChatConversation, 0, limit)
	for rows.Next() {
		var item types.ChatConversation
		var createdAt, updatedAt string
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.ModelID,
			&item.SystemPrompt,
			&createdAt,
			&updatedAt,
			&item.LastMessagePreview,
			&item.MessageCount,
		); err != nil {
			return nil, err
		}
		if ts, err := time.Parse(time.RFC3339, createdAt); err == nil {
			item.CreatedAt = ts
		}
		if ts, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			item.UpdatedAt = ts
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListChatMessages(ctx context.Context, conversationID string, limit int) ([]types.ChatMessage, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return []types.ChatMessage{}, nil
	}
	if limit <= 0 {
		limit = 2000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			conversation_id,
			role,
			content,
			reasoning_text,
			provider_id,
			route_model,
			created_at
		FROM chat_messages
		WHERE conversation_id = ?
		ORDER BY id ASC
		LIMIT ?
	`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.ChatMessage, 0)
	for rows.Next() {
		var item types.ChatMessage
		var createdAt string
		if err := rows.Scan(
			&item.ID,
			&item.ConversationID,
			&item.Role,
			&item.Content,
			&item.ReasoningText,
			&item.ProviderID,
			&item.RouteModel,
			&createdAt,
		); err != nil {
			return nil, err
		}
		if ts, err := time.Parse(time.RFC3339, createdAt); err == nil {
			item.CreatedAt = ts
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) AppendChatExchange(ctx context.Context, conversationID, modelID, title, userContent, assistantContent, reasoningText, providerID, routeModel string) error {
	conversationID = strings.TrimSpace(conversationID)
	modelID = strings.TrimSpace(modelID)
	title = strings.TrimSpace(title)
	userContent = strings.TrimSpace(userContent)
	assistantContent = strings.TrimSpace(assistantContent)
	if conversationID == "" {
		return fmt.Errorf("conversation id is required")
	}
	if modelID == "" {
		return fmt.Errorf("model id is required")
	}
	if title == "" {
		title = "新对话"
	}
	if userContent == "" {
		return fmt.Errorf("user content is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_messages(conversation_id, role, content, reasoning_text, provider_id, route_model, created_at)
		VALUES(?, 'user', ?, '', '', '', ?)
	`, conversationID, userContent, now); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_messages(conversation_id, role, content, reasoning_text, provider_id, route_model, created_at)
		VALUES(?, 'assistant', ?, ?, ?, ?, ?)
	`, conversationID, assistantContent, strings.TrimSpace(reasoningText), strings.TrimSpace(providerID), strings.TrimSpace(routeModel), now); err != nil {
		_ = tx.Rollback()
		return err
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE chat_conversations
		SET title = ?, model_id = ?, updated_at = ?
		WHERE id = ?
	`, title, modelID, now, conversationID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_ = tx.Rollback()
		return fmt.Errorf("conversation not found")
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) DeleteChatConversation(ctx context.Context, conversationID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM chat_conversations WHERE id = ?`, conversationID)
	return err
}

func (s *Store) InsertRequestLog(ctx context.Context, meta types.RequestLogMeta) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO request_logs_meta(
			ts, request_id, client_key_id, provider_id, model_id, path, status, latency_ms,
			input_tokens, output_tokens, reasoning_tokens, cached_tokens, error_code
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, meta.Timestamp.UTC().Format(time.RFC3339), meta.RequestID, nullIfEmpty(meta.ClientKeyID), nullIfEmpty(meta.ProviderID), nullIfEmpty(meta.ModelID), meta.Path, meta.Status, meta.LatencyMS, meta.InputTokens, meta.OutputTokens, meta.ReasoningTokens, meta.CachedTokens, nullIfEmpty(meta.ErrorCode))
	return err
}

func (s *Store) ListRequestLogs(ctx context.Context, limit int) ([]types.RequestLogMeta, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			ts,
			request_id,
			COALESCE(client_key_id,''),
			COALESCE(provider_id,''),
			COALESCE(model_id,''),
			path,
			status,
			latency_ms,
			input_tokens,
			output_tokens,
			COALESCE(reasoning_tokens, 0),
			COALESCE(cached_tokens, 0),
			COALESCE(error_code,'')
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
		if err := rows.Scan(
			&item.ID,
			&ts,
			&item.RequestID,
			&item.ClientKeyID,
			&item.ProviderID,
			&item.ModelID,
			&item.Path,
			&item.Status,
			&item.LatencyMS,
			&item.InputTokens,
			&item.OutputTokens,
			&item.ReasoningTokens,
			&item.CachedTokens,
			&item.ErrorCode,
		); err != nil {
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

func (s *Store) ListRequestLogsBetween(ctx context.Context, start, end time.Time, limit int) ([]types.RequestLogMeta, error) {
	if limit <= 0 {
		limit = 20000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			ts,
			COALESCE(request_id,''),
			COALESCE(client_key_id,''),
			COALESCE(provider_id,''),
			COALESCE(model_id,''),
			COALESCE(path,''),
			COALESCE(status, 0),
			COALESCE(latency_ms, 0),
			COALESCE(input_tokens, 0),
			COALESCE(output_tokens, 0),
			COALESCE(reasoning_tokens, 0),
			COALESCE(cached_tokens, 0),
			COALESCE(error_code,'')
		FROM request_logs_meta
		WHERE datetime(ts) >= datetime(?) AND datetime(ts) < datetime(?)
		ORDER BY datetime(ts) ASC, id ASC
		LIMIT ?
	`, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.RequestLogMeta, 0)
	for rows.Next() {
		var item types.RequestLogMeta
		var ts string
		if err := rows.Scan(
			&item.ID,
			&ts,
			&item.RequestID,
			&item.ClientKeyID,
			&item.ProviderID,
			&item.ModelID,
			&item.Path,
			&item.Status,
			&item.LatencyMS,
			&item.InputTokens,
			&item.OutputTokens,
			&item.ReasoningTokens,
			&item.CachedTokens,
			&item.ErrorCode,
		); err != nil {
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

func (s *Store) PruneRequestLogs(ctx context.Context, olderThan time.Duration, keepMax int) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	var deleted int64

	if olderThan > 0 {
		threshold := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
		res, err := tx.ExecContext(ctx, `DELETE FROM request_logs_meta WHERE datetime(ts) < datetime(?)`, threshold)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if n, err := res.RowsAffected(); err == nil {
			deleted += n
		}
	}

	if keepMax > 0 {
		res, err := tx.ExecContext(ctx, `
			DELETE FROM request_logs_meta
			WHERE id IN (
				SELECT id
				FROM request_logs_meta
				ORDER BY id DESC
				LIMIT -1 OFFSET ?
			)
		`, keepMax)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if n, err := res.RowsAffected(); err == nil {
			deleted += n
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(deleted), nil
}

type RequestStats struct {
	Requests     int
	InputTokens  int
	OutputTokens int
}

type RequestUsageStats struct {
	Requests        int    `json:"requests"`
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	ReasoningTokens int    `json:"reasoning_tokens"`
	CachedTokens    int    `json:"cached_tokens"`
	Start           string `json:"start"`
	End             string `json:"end"`
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

func (s *Store) RequestUsageBetween(ctx context.Context, start, end time.Time) (RequestUsageStats, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(1),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(reasoning_tokens), 0),
			COALESCE(SUM(cached_tokens), 0)
		FROM request_logs_meta
		WHERE datetime(ts) >= datetime(?) AND datetime(ts) < datetime(?)
	`, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))

	stats := RequestUsageStats{
		Start: start.UTC().Format(time.RFC3339),
		End:   end.UTC().Format(time.RFC3339),
	}
	if err := row.Scan(&stats.Requests, &stats.InputTokens, &stats.OutputTokens, &stats.ReasoningTokens, &stats.CachedTokens); err != nil {
		return RequestUsageStats{}, err
	}
	return stats, nil
}

type DailyTokenUsage struct {
	Day          string `json:"day"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

type PathModelUsage struct {
	Path         string `json:"path"`
	ModelID      string `json:"model_id"`
	Requests     int    `json:"requests"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

type ModelTokenUsage struct {
	ModelID         string `json:"model_id"`
	Requests        int    `json:"requests"`
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	ReasoningTokens int    `json:"reasoning_tokens"`
	CachedTokens    int    `json:"cached_tokens"`
}

type TokenTrendPoint struct {
	BucketStart     string `json:"bucket_start"`
	Requests        int    `json:"requests"`
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	ReasoningTokens int    `json:"reasoning_tokens"`
	CachedTokens    int    `json:"cached_tokens"`
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

func (s *Store) PathModelUsageSince(ctx context.Context, since time.Time, limit int) ([]PathModelUsage, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			COALESCE(NULLIF(path, ''), '-'),
			COALESCE(NULLIF(model_id, ''), '-'),
			COUNT(1),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0)
		FROM request_logs_meta
		WHERE datetime(ts) >= datetime(?)
		GROUP BY 1, 2
		ORDER BY (COALESCE(SUM(input_tokens), 0) + COALESCE(SUM(output_tokens), 0)) DESC, COUNT(1) DESC, 1 ASC, 2 ASC
		LIMIT ?
	`, since.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PathModelUsage, 0, limit)
	for rows.Next() {
		var item PathModelUsage
		if err := rows.Scan(&item.Path, &item.ModelID, &item.Requests, &item.InputTokens, &item.OutputTokens); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ModelUsageBetween(ctx context.Context, start, end time.Time, limit int) ([]ModelTokenUsage, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			COALESCE(NULLIF(model_id, ''), '-'),
			COUNT(1),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(reasoning_tokens), 0),
			COALESCE(SUM(cached_tokens), 0)
		FROM request_logs_meta
		WHERE datetime(ts) >= datetime(?) AND datetime(ts) < datetime(?)
		GROUP BY 1
		ORDER BY (COALESCE(SUM(input_tokens), 0) + COALESCE(SUM(output_tokens), 0)) DESC, COUNT(1) DESC, 1 ASC
		LIMIT ?
	`, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ModelTokenUsage, 0, limit)
	for rows.Next() {
		var item ModelTokenUsage
		if err := rows.Scan(&item.ModelID, &item.Requests, &item.InputTokens, &item.OutputTokens, &item.ReasoningTokens, &item.CachedTokens); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) TokenTrendBetween(ctx context.Context, start, end time.Time, bucketSeconds int) ([]TokenTrendPoint, error) {
	if bucketSeconds <= 0 {
		bucketSeconds = 300
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			(CAST(strftime('%s', ts) AS INTEGER) / ?) * ? AS bucket_ts,
			COUNT(1),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(reasoning_tokens), 0),
			COALESCE(SUM(cached_tokens), 0)
		FROM request_logs_meta
		WHERE datetime(ts) >= datetime(?) AND datetime(ts) < datetime(?)
		GROUP BY 1
		ORDER BY 1 ASC
	`, bucketSeconds, bucketSeconds, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TokenTrendPoint, 0)
	for rows.Next() {
		var item TokenTrendPoint
		var bucket int64
		if err := rows.Scan(&bucket, &item.Requests, &item.InputTokens, &item.OutputTokens, &item.ReasoningTokens, &item.CachedTokens); err != nil {
			return nil, err
		}
		item.BucketStart = time.Unix(bucket, 0).UTC().Format(time.RFC3339)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) EnsureDefaultModels(ctx context.Context) error {
	defaults := []types.Model{
		{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Enabled: true},
		{ID: "claude-3-5-sonnet", DisplayName: "Claude 3.5 Sonnet", Enabled: true},
	}
	for _, m := range defaults {
		existing, err := s.GetModel(ctx, m.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if err := s.UpsertModel(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EnsureBuiltinProviders(ctx context.Context) error {
	defaults := []types.Provider{
		{
			ID:         "forward",
			Type:       types.ProviderTypeBuiltin,
			Protocol:   types.ProtocolForward,
			Endpoint:   "https://api.openai.com/v1",
			ConfigJSON: "{}",
			Enabled:    true,
			Health:     "healthy",
		},
		{
			ID:         "anthropic",
			Type:       types.ProviderTypeBuiltin,
			Protocol:   types.ProtocolAnthropic,
			Endpoint:   "https://api.anthropic.com",
			ConfigJSON: "{}",
			Enabled:    true,
			Health:     "healthy",
		},
	}
	const removedKeyPrefix = "builtin_provider_removed:"
	for _, p := range defaults {
		if _, ok, err := s.GetSetting(ctx, removedKeyPrefix+p.ID); err == nil && ok {
			continue
		}
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
