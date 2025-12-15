-- Add team migration execution status fields to team_mappings table
-- This enables tracking the progress of team creation and permission sync

-- +goose Up
ALTER TABLE team_mappings ADD COLUMN migration_status TEXT DEFAULT 'pending';
ALTER TABLE team_mappings ADD COLUMN migrated_at TIMESTAMP;
ALTER TABLE team_mappings ADD COLUMN error_message TEXT;
ALTER TABLE team_mappings ADD COLUMN repos_synced INTEGER DEFAULT 0;

-- Index for migration status filtering
CREATE INDEX IF NOT EXISTS idx_team_mappings_migration_status 
    ON team_mappings(migration_status);

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we leave the columns in place during downgrade
DROP INDEX IF EXISTS idx_team_mappings_migration_status;

