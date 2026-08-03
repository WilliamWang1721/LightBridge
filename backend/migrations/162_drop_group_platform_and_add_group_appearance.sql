-- Groups are provider-neutral containers. Provider and protocol capabilities
-- are derived from their bound upstream accounts.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS icon VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS color VARCHAR(7) NOT NULL DEFAULT '';

ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_icon_check;
ALTER TABLE groups ADD CONSTRAINT groups_icon_check CHECK (
    icon IN (
        '', 'folder', 'server', 'cloud', 'bolt', 'shield', 'cube',
        'terminal', 'sparkles', 'users'
    )
);

ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_color_check;
ALTER TABLE groups ADD CONSTRAINT groups_color_check
    CHECK (color = '' OR color ~ '^#[0-9A-Fa-f]{6}$');

DROP INDEX IF EXISTS idx_groups_platform;
ALTER TABLE groups DROP COLUMN IF EXISTS platform;
ALTER TABLE groups DROP COLUMN IF EXISTS supported_model_scopes;
