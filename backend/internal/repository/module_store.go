package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/modules"
)

type ModuleStore struct{ db *sql.DB }

func NewModuleStore(db *sql.DB) *ModuleStore { return &ModuleStore{db: db} }
func (s *ModuleStore) ListInstalled(ctx context.Context) ([]modules.InstalledModule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,type,version,status,install_path,manifest_json,installed_at,enabled_at,COALESCE(last_error,'') FROM installed_modules WHERE status <> 'purged' ORDER BY installed_at DESC,id ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []modules.InstalledModule
	for rows.Next() {
		m, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}
func (s *ModuleStore) GetInstalled(ctx context.Context, id string) (*modules.InstalledModule, error) {
	return scanModule(s.db.QueryRowContext(ctx, `SELECT id,name,type,version,status,install_path,manifest_json,installed_at,enabled_at,COALESCE(last_error,'') FROM installed_modules WHERE id=$1`, id))
}
func (s *ModuleStore) SaveInstalled(ctx context.Context, m modules.InstalledModule) error {
	b, err := json.Marshal(m.Manifest)
	if err != nil {
		return err
	}
	if m.InstalledAt.IsZero() {
		m.InstalledAt = time.Now().UTC()
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO installed_modules (id,name,type,version,status,install_path,manifest_json,installed_at,enabled_at,last_error,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),NOW()) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,type=EXCLUDED.type,version=EXCLUDED.version,status=EXCLUDED.status,install_path=EXCLUDED.install_path,manifest_json=EXCLUDED.manifest_json,enabled_at=EXCLUDED.enabled_at,last_error=EXCLUDED.last_error,updated_at=NOW()`, m.ID, m.Name, string(m.Type), m.Version, string(m.Status), m.InstallPath, string(b), m.InstalledAt, m.EnabledAt, m.LastError)
	return err
}
func (s *ModuleStore) SavePermissions(ctx context.Context, moduleID string, perms []modules.PermissionRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM module_permissions WHERE module_id=$1`, moduleID); err != nil {
		return err
	}
	for _, p := range perms {
		if _, err := tx.ExecContext(ctx, `INSERT INTO module_permissions (module_id,permission_type,permission_value,approved,created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (module_id,permission_type,permission_value) DO NOTHING`, moduleID, p.PermissionType, p.PermissionValue, p.Approved, p.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *ModuleStore) ListPermissions(ctx context.Context, moduleID string) ([]modules.PermissionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT module_id,permission_type,permission_value,approved,approved_at,created_at FROM module_permissions WHERE module_id=$1 ORDER BY permission_type,permission_value`, moduleID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []modules.PermissionRecord
	for rows.Next() {
		var p modules.PermissionRecord
		if err := rows.Scan(&p.ModuleID, &p.PermissionType, &p.PermissionValue, &p.Approved, &p.ApprovedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *ModuleStore) ApprovePermissions(ctx context.Context, moduleID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE module_permissions SET approved=true, approved_at=NOW() WHERE module_id=$1`, moduleID)
	return err
}
func (s *ModuleStore) SetStatus(ctx context.Context, id string, status modules.ModuleStatus, lastErr string) error {
	var enabled any
	if status == modules.ModuleStatusEnabled {
		enabled = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `UPDATE installed_modules SET status=$2, enabled_at=COALESCE($3,enabled_at), last_error=NULLIF($4,''), updated_at=NOW() WHERE id=$1`, id, string(status), enabled, lastErr)
	return err
}
func (s *ModuleStore) PurgeModuleData(ctx context.Context, m modules.InstalledModule) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM module_kv WHERE module_id=$1`, m.ID)
	return err
}

type moduleScanner interface{ Scan(...any) error }

func scanModule(row moduleScanner) (*modules.InstalledModule, error) {
	var m modules.InstalledModule
	var typ, status, manifest string
	err := row.Scan(&m.ID, &m.Name, &typ, &m.Version, &status, &m.InstallPath, &manifest, &m.InstalledAt, &m.EnabledAt, &m.LastError)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	m.Type = modules.ModuleType(typ)
	m.Status = modules.ModuleStatus(status)
	_ = json.Unmarshal([]byte(manifest), &m.Manifest)
	return &m, nil
}
