package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
	"github.com/WilliamWang1721/LightBridge/internal/service"
)

type UIThemeRepository struct {
	db *sql.DB
}

func NewUIThemeRepository(db *sql.DB) service.UIThemeRepository {
	return &UIThemeRepository{db: db}
}

func (r *UIThemeRepository) List(ctx context.Context) ([]service.UITheme, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, version, source, entry_css, preview, manifest, config, active, created_at, updated_at
FROM ui_themes
ORDER BY active DESC, updated_at DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var themes []service.UITheme
	for rows.Next() {
		theme, err := scanUITheme(rows)
		if err != nil {
			return nil, err
		}
		themes = append(themes, *theme)
	}
	return themes, rows.Err()
}

func (r *UIThemeRepository) Get(ctx context.Context, id string) (*service.UITheme, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, version, source, entry_css, preview, manifest, config, active, created_at, updated_at
FROM ui_themes WHERE id = $1`, id)
	theme, err := scanUITheme(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("UI_THEME_NOT_FOUND", "theme not found")
		}
		return nil, err
	}
	return theme, nil
}

func (r *UIThemeRepository) GetActive(ctx context.Context) (*service.UITheme, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, version, source, entry_css, preview, manifest, config, active, created_at, updated_at
FROM ui_themes WHERE active = TRUE ORDER BY updated_at DESC LIMIT 1`)
	theme, err := scanUITheme(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return theme, nil
}

func (r *UIThemeRepository) Upsert(ctx context.Context, theme service.UITheme) error {
	now := time.Now()
	if len(theme.Manifest) == 0 {
		theme.Manifest = json.RawMessage(`{}`)
	}
	if len(theme.Config) == 0 {
		theme.Config = json.RawMessage(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO ui_themes (id, name, version, source, entry_css, preview, manifest, config, active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, FALSE, $9, $9)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  version = EXCLUDED.version,
  source = EXCLUDED.source,
  entry_css = EXCLUDED.entry_css,
  preview = EXCLUDED.preview,
  manifest = EXCLUDED.manifest,
  config = EXCLUDED.config,
  updated_at = EXCLUDED.updated_at`,
		theme.ID,
		theme.Name,
		theme.Version,
		theme.Source,
		theme.EntryCSS,
		theme.Preview,
		[]byte(theme.Manifest),
		[]byte(theme.Config),
		now,
	)
	return err
}

func (r *UIThemeRepository) UpdateConfig(ctx context.Context, id string, config json.RawMessage) (*service.UITheme, error) {
	row := r.db.QueryRowContext(ctx, `
UPDATE ui_themes SET config = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, name, version, source, entry_css, preview, manifest, config, active, created_at, updated_at`, id, []byte(config))
	theme, err := scanUITheme(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("UI_THEME_NOT_FOUND", "theme not found")
		}
		return nil, err
	}
	return theme, nil
}

func (r *UIThemeRepository) Activate(ctx context.Context, id string) (*service.UITheme, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE ui_themes SET active = FALSE WHERE active = TRUE`); err != nil {
		return nil, err
	}
	row := tx.QueryRowContext(ctx, `
UPDATE ui_themes SET active = TRUE, updated_at = NOW()
WHERE id = $1
RETURNING id, name, version, source, entry_css, preview, manifest, config, active, created_at, updated_at`, id)
	theme, err := scanUITheme(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("UI_THEME_NOT_FOUND", "theme not found")
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return theme, nil
}

func (r *UIThemeRepository) Deactivate(ctx context.Context, id string) (*service.UITheme, error) {
	row := r.db.QueryRowContext(ctx, `
UPDATE ui_themes SET active = FALSE, updated_at = NOW()
WHERE id = $1
RETURNING id, name, version, source, entry_css, preview, manifest, config, active, created_at, updated_at`, id)
	theme, err := scanUITheme(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.NotFound("UI_THEME_NOT_FOUND", "theme not found")
		}
		return nil, err
	}
	return theme, nil
}

func (r *UIThemeRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ui_themes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err == nil && count == 0 {
		return infraerrors.NotFound("UI_THEME_NOT_FOUND", "theme not found")
	}
	return nil
}

type uiThemeScanner interface {
	Scan(dest ...any) error
}

func scanUITheme(scanner uiThemeScanner) (*service.UITheme, error) {
	var theme service.UITheme
	var manifest, config []byte
	if err := scanner.Scan(
		&theme.ID,
		&theme.Name,
		&theme.Version,
		&theme.Source,
		&theme.EntryCSS,
		&theme.Preview,
		&manifest,
		&config,
		&theme.Active,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	); err != nil {
		return nil, err
	}
	theme.Manifest = json.RawMessage(manifest)
	theme.Config = json.RawMessage(config)
	return &theme, nil
}
