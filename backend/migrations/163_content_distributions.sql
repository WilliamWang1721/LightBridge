-- Content distribution center.
-- A distribution owns the content/attachment once, while recipients are stored as a snapshot.

CREATE TABLE IF NOT EXISTS content_distributions (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    kind VARCHAR(32) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    file_name VARCHAR(255),
    content_type VARCHAR(255),
    file_size BIGINT NOT NULL DEFAULT 0,
    file_data BYTEA,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    audience JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT content_distributions_kind_check
        CHECK (kind IN ('text', 'message', 'file', 'account_export')),
    CONSTRAINT content_distributions_file_size_check
        CHECK (file_size >= 0)
);

CREATE TABLE IF NOT EXISTS content_distribution_recipients (
    distribution_id BIGINT NOT NULL REFERENCES content_distributions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_override VARCHAR(200),
    content_override TEXT,
    read_at TIMESTAMPTZ,
    downloaded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (distribution_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_content_distributions_created_at
    ON content_distributions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_content_distribution_recipients_user_created
    ON content_distribution_recipients(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_content_distribution_recipients_unread
    ON content_distribution_recipients(user_id, distribution_id DESC)
    WHERE read_at IS NULL;
