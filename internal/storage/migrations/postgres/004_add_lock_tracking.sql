-- +goose Up
-- Add fields to track repository lock status and migration ID
-- This allows unlocking repositories that get stuck after failed migrations

ALTER TABLE repositories ADD COLUMN source_migration_id INTEGER;
ALTER TABLE repositories ADD COLUMN is_source_locked BOOLEAN DEFAULT FALSE;

-- Index for finding locked repositories
CREATE INDEX IF NOT EXISTS idx_repositories_source_locked ON repositories(is_source_locked) WHERE is_source_locked = TRUE;



-- +goose Down
-- Add rollback logic here
