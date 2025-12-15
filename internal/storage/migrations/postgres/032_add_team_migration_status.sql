-- Add team migration execution status fields to team_mappings table
-- This enables tracking the progress of team creation and permission sync

-- +goose Up
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS migration_status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS migrated_at TIMESTAMP;
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS error_message TEXT;
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS repos_synced INTEGER DEFAULT 0;

-- Index for migration status filtering
CREATE INDEX IF NOT EXISTS idx_team_mappings_migration_status 
    ON team_mappings(migration_status);

-- +goose Down
DROP INDEX IF EXISTS idx_team_mappings_migration_status;
ALTER TABLE team_mappings DROP COLUMN IF EXISTS repos_synced;
ALTER TABLE team_mappings DROP COLUMN IF EXISTS error_message;
ALTER TABLE team_mappings DROP COLUMN IF EXISTS migrated_at;
ALTER TABLE team_mappings DROP COLUMN IF EXISTS migration_status;

