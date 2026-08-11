-- Paid distributions stay locked until the recipient accepts and pays.

ALTER TABLE content_distributions
    ADD COLUMN IF NOT EXISTS price NUMERIC(20, 8) NOT NULL DEFAULT 0;

ALTER TABLE content_distributions
    DROP CONSTRAINT IF EXISTS content_distributions_kind_check;

ALTER TABLE content_distributions
    ADD CONSTRAINT content_distributions_kind_check
    CHECK (kind IN ('text', 'message', 'file', 'account_export', 'paid'));

ALTER TABLE content_distributions
    DROP CONSTRAINT IF EXISTS content_distributions_price_check;

ALTER TABLE content_distributions
    ADD CONSTRAINT content_distributions_price_check CHECK (price >= 0);

ALTER TABLE content_distribution_recipients
    ADD COLUMN IF NOT EXISTS accepted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_content_distribution_recipients_pending
    ON content_distribution_recipients(user_id, distribution_id DESC)
    WHERE accepted_at IS NULL AND rejected_at IS NULL;
