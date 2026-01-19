-- +goose Up
-- This migration ensures mannequin_org column and user_mannequins table exist.
-- For new databases, these are already defined in 001_schema.sql.
-- For existing databases with old 001_schema.sql, these statements add the missing schema.
-- All statements use IF NOT EXISTS or are idempotent to work in both cases.

-- Note: We skip ALTER TABLE ADD COLUMN mannequin_org because:
-- 1. New databases already have it in 001_schema.sql
-- 2. SQLite doesn't support ADD COLUMN IF NOT EXISTS
-- 3. Existing databases that need the column should use manual migration or app-level handling

-- Create index for mannequin_org (IF NOT EXISTS is supported for indexes)
CREATE INDEX IF NOT EXISTS idx_user_mappings_mannequin_org ON user_mappings(mannequin_org);

-- User Mannequins table: tracks mannequin info per org (a user can have mannequins in multiple orgs)
-- Using IF NOT EXISTS makes this idempotent
CREATE TABLE IF NOT EXISTS user_mannequins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_login TEXT NOT NULL,
    mannequin_org TEXT NOT NULL,
    mannequin_id TEXT NOT NULL,
    mannequin_login TEXT,
    reclaim_status TEXT,
    reclaim_error TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
-- SQLite doesn't support DROP COLUMN easily, so we skip that in down migration
