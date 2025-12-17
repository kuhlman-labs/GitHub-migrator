-- Add fields to track partial vs. full team migration
-- This enables distinguishing between teams created before repo migrations vs. fully synced teams

-- +goose Up
-- total_source_repos: Total repos this team has access to in source org
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS total_source_repos INTEGER DEFAULT 0;

-- repos_eligible: How many of those repos have been migrated and are available for permission sync
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS repos_eligible INTEGER DEFAULT 0;

-- team_created_in_dest: Whether the team structure has been created in the destination org
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS team_created_in_dest BOOLEAN DEFAULT FALSE;

-- last_synced_at: When permissions were last synced (different from migrated_at which is team creation time)
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMP;

-- +goose Down
ALTER TABLE team_mappings DROP COLUMN IF EXISTS last_synced_at;
ALTER TABLE team_mappings DROP COLUMN IF EXISTS team_created_in_dest;
ALTER TABLE team_mappings DROP COLUMN IF EXISTS repos_eligible;
ALTER TABLE team_mappings DROP COLUMN IF EXISTS total_source_repos;
