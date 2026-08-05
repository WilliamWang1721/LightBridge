package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/pagination"
	"github.com/WilliamWang1721/LightBridge/internal/service"
)

type distributionRepository struct {
	db *sql.DB
}

func NewDistributionRepository(db *sql.DB) service.DistributionRepository {
	return &distributionRepository{db: db}
}

func (r *distributionRepository) Create(ctx context.Context, distribution *service.Distribution, attachment *service.DistributionAttachment, recipients []service.DistributionRecipient) error {
	if distribution == nil || len(recipients) == 0 {
		return service.ErrDistributionInvalid
	}
	metadata, err := json.Marshal(distribution.Metadata)
	if err != nil {
		return err
	}
	audience, err := json.Marshal(distribution.Audience)
	if err != nil {
		return err
	}

	var fileName, contentType any
	var fileData []byte
	if attachment != nil {
		fileName = attachment.FileName
		contentType = attachment.ContentType
		fileData = attachment.Ciphertext
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
		INSERT INTO content_distributions
			(title, kind, content, file_name, content_type, file_size, file_data, metadata, audience, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`, distribution.Title, distribution.Kind, distribution.Content, fileName, contentType, distribution.FileSize, fileData, string(metadata), string(audience), distribution.CreatedBy)
	if err := row.Scan(&distribution.ID, &distribution.CreatedAt); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO content_distribution_recipients
			(distribution_id, user_id, title_override, content_override)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''))
		ON CONFLICT (distribution_id, user_id) DO UPDATE SET
			title_override = EXCLUDED.title_override,
			content_override = EXCLUDED.content_override
	`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, recipient := range recipients {
		if _, err := stmt.ExecContext(ctx, distribution.ID, recipient.UserID, recipient.TitleOverride, recipient.ContentOverride); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *distributionRepository) ListAdmin(ctx context.Context, params pagination.PaginationParams) ([]service.Distribution, *pagination.PaginationResult, error) {
	total, err := r.count(ctx, `SELECT COUNT(*) FROM content_distributions`)
	if err != nil {
		return nil, nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.title, d.kind, d.content,
		       COALESCE(d.file_name, ''), COALESCE(d.content_type, ''), d.file_size,
		       d.metadata, d.audience, d.created_by, d.created_at,
		       (SELECT COUNT(*) FROM content_distribution_recipients r WHERE r.distribution_id = d.id),
		       (SELECT COUNT(*) FROM content_distribution_recipients r WHERE r.distribution_id = d.id AND r.read_at IS NOT NULL),
		       (SELECT COUNT(*) FROM content_distribution_recipients r WHERE r.distribution_id = d.id AND r.downloaded_at IS NOT NULL),
		       NULL, NULL, '', ''
		FROM content_distributions d
		ORDER BY d.id DESC
		LIMIT $1 OFFSET $2
	`, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanDistributionRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, distributionPagination(total, params), nil
}

func (r *distributionRepository) ListForUser(ctx context.Context, userID int64, params pagination.PaginationParams, unreadOnly bool) ([]service.Distribution, *pagination.PaginationResult, error) {
	where := "WHERE r.user_id = $1"
	if unreadOnly {
		where += " AND r.read_at IS NULL"
	}
	total, err := r.count(ctx, `SELECT COUNT(*) FROM content_distribution_recipients r `+where, userID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.title, d.kind, d.content,
		       COALESCE(d.file_name, ''), COALESCE(d.content_type, ''), d.file_size,
		       '{}'::jsonb, '{}'::jsonb, NULL, d.created_at,
		       0, 0, 0,
		       r.read_at, r.downloaded_at,
		       COALESCE(r.title_override, ''), COALESCE(r.content_override, '')
		FROM content_distributions d
		JOIN content_distribution_recipients r ON r.distribution_id = d.id
		`+where+`
		ORDER BY d.id DESC
		LIMIT $2 OFFSET $3
	`, userID, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanDistributionRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return items, distributionPagination(total, params), nil
}

func (r *distributionRepository) GetByID(ctx context.Context, id int64) (*service.Distribution, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT d.id, d.title, d.kind, d.content,
		       COALESCE(d.file_name, ''), COALESCE(d.content_type, ''), d.file_size,
		       d.metadata, d.audience, d.created_by, d.created_at,
		       (SELECT COUNT(*) FROM content_distribution_recipients r WHERE r.distribution_id = d.id),
		       (SELECT COUNT(*) FROM content_distribution_recipients r WHERE r.distribution_id = d.id AND r.read_at IS NOT NULL),
		       (SELECT COUNT(*) FROM content_distribution_recipients r WHERE r.distribution_id = d.id AND r.downloaded_at IS NOT NULL),
		       NULL, NULL, '', ''
		FROM content_distributions d
		WHERE d.id = $1
	`, id)
	item, err := scanDistribution(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrDistributionNotFound
	}
	return item, err
}

func (r *distributionRepository) GetForUser(ctx context.Context, id, userID int64) (*service.Distribution, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT d.id, d.title, d.kind, d.content,
		       COALESCE(d.file_name, ''), COALESCE(d.content_type, ''), d.file_size,
		       '{}'::jsonb, '{}'::jsonb, NULL, d.created_at,
		       0, 0, 0,
		       r.read_at, r.downloaded_at,
		       COALESCE(r.title_override, ''), COALESCE(r.content_override, '')
		FROM content_distributions d
		JOIN content_distribution_recipients r ON r.distribution_id = d.id
		WHERE d.id = $1 AND r.user_id = $2
	`, id, userID)
	item, err := scanDistribution(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrDistributionNotFound
	}
	return item, err
}

func (r *distributionRepository) GetAttachment(ctx context.Context, id int64) (*service.DistributionAttachment, error) {
	return r.getAttachment(ctx, `
		SELECT COALESCE(file_name, ''), COALESCE(content_type, ''), file_size, file_data
		FROM content_distributions
		WHERE id = $1 AND file_data IS NOT NULL
	`, id)
}

func (r *distributionRepository) GetAttachmentForUser(ctx context.Context, id, userID int64) (*service.DistributionAttachment, error) {
	return r.getAttachment(ctx, `
		SELECT COALESCE(d.file_name, ''), COALESCE(d.content_type, ''), d.file_size, d.file_data
		FROM content_distributions d
		JOIN content_distribution_recipients r ON r.distribution_id = d.id
		WHERE d.id = $1 AND r.user_id = $2 AND d.file_data IS NOT NULL
	`, id, userID)
}

func (r *distributionRepository) getAttachment(ctx context.Context, query string, args ...any) (*service.DistributionAttachment, error) {
	attachment := &service.DistributionAttachment{}
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&attachment.FileName, &attachment.ContentType, &attachment.FileSize, &attachment.Ciphertext); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrDistributionNotFound
		}
		return nil, err
	}
	return attachment, nil
}

func (r *distributionRepository) MarkRead(ctx context.Context, id, userID int64, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE content_distribution_recipients
		SET read_at = COALESCE(read_at, $3)
		WHERE distribution_id = $1 AND user_id = $2
	`, id, userID, at)
	return distributionAffected(result, err)
}

func (r *distributionRepository) MarkDownloaded(ctx context.Context, id, userID int64, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE content_distribution_recipients
		SET downloaded_at = COALESCE(downloaded_at, $3), read_at = COALESCE(read_at, $3)
		WHERE distribution_id = $1 AND user_id = $2
	`, id, userID, at)
	return distributionAffected(result, err)
}

func (r *distributionRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM content_distributions WHERE id = $1`, id)
	return distributionAffected(result, err)
}

func distributionAffected(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return service.ErrDistributionNotFound
	}
	return nil
}

func (r *distributionRepository) count(ctx context.Context, query string, args ...any) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

type distributionScanner interface {
	Scan(dest ...any) error
}

func scanDistribution(scanner distributionScanner) (*service.Distribution, error) {
	item := &service.Distribution{}
	var metadataRaw, audienceRaw []byte
	var createdBy sql.NullInt64
	var readAt, downloadedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.Title,
		&item.Kind,
		&item.Content,
		&item.FileName,
		&item.ContentType,
		&item.FileSize,
		&metadataRaw,
		&audienceRaw,
		&createdBy,
		&item.CreatedAt,
		&item.RecipientCount,
		&item.ReadCount,
		&item.DownloadCount,
		&readAt,
		&downloadedAt,
		&item.RecipientTitle,
		&item.RecipientContent,
	); err != nil {
		return nil, err
	}
	item.HasAttachment = item.FileSize > 0
	if createdBy.Valid {
		value := createdBy.Int64
		item.CreatedBy = &value
	}
	if readAt.Valid {
		value := readAt.Time
		item.ReadAt = &value
	}
	if downloadedAt.Valid {
		value := downloadedAt.Time
		item.DownloadedAt = &value
	}
	item.Metadata = decodeDistributionJSON(metadataRaw)
	item.Audience = decodeDistributionJSON(audienceRaw)
	if item.RecipientTitle != "" {
		item.Title = item.RecipientTitle
	}
	if item.RecipientContent != "" {
		item.Content = item.RecipientContent
	}
	return item, nil
}

func scanDistributionRows(rows *sql.Rows) ([]service.Distribution, error) {
	items := make([]service.Distribution, 0)
	for rows.Next() {
		item, err := scanDistribution(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func decodeDistributionJSON(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func distributionPagination(total int64, params pagination.PaginationParams) *pagination.PaginationResult {
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.Limit()
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if pages < 1 {
		pages = 1
	}
	return &pagination.PaginationResult{Total: total, Page: page, PageSize: pageSize, Pages: pages}
}

var _ service.DistributionRepository = (*distributionRepository)(nil)

func (r *distributionRepository) String() string {
	return fmt.Sprintf("distributionRepository(%p)", r.db)
}
