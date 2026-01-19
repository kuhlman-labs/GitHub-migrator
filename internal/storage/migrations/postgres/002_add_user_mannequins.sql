-- +goose Up
-- Add mannequin_org column to user_mappings table
ALTER TABLE user_mappings ADD COLUMN IF NOT EXISTS mannequin_org TEXT;

-- Create index for mannequin_org
CREATE INDEX IF NOT EXISTS idx_user_mappings_mannequin_org ON user_mappings(mannequin_org);

-- User Mannequins table: tracks mannequin info per org (a user can have mannequins in multiple orgs)
CREATE TABLE IF NOT EXISTS user_mannequins (
    id BIGSERIAL PRIMARY KEY,
    source_login TEXT NOT NULL,
    mannequin_org TEXT NOT NULL,
    mannequin_id TEXT NOT NULL,
    mannequin_login TEXT,
    reclaim_status TEXT,
    reclaim_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_login, mannequin_org)
);

CREATE INDEX IF NOT EXISTS idx_user_mannequins_source ON user_mannequins(source_login);
CREATE INDEX IF NOT EXISTS idx_user_mannequins_org ON user_mannequins(mannequin_org);
CREATE INDEX IF NOT EXISTS idx_user_mannequins_status ON user_mannequins(reclaim_status);

-- +goose Down
DROP INDEX IF EXISTS idx_user_mannequins_status;
DROP INDEX IF EXISTS idx_user_mannequins_org;
DROP INDEX IF EXISTS idx_user_mannequins_source;
DROP TABLE IF EXISTS user_mannequins;
DROP INDEX IF EXISTS idx_user_mappings_mannequin_org;
ALTER TABLE user_mappings DROP COLUMN IF EXISTS mannequin_org;
