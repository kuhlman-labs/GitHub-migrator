-- Add fields to track partial vs. full team migration
-- This enables distinguishing between teams created before repo migrations vs. fully synced teams

-- +goose Up
-- total_source_repos: Total repos this team has access to in source org
ALTER TABLE team_mappings ADD COLUMN total_source_repos INTEGER DEFAULT 0;

-- repos_eligible: How many of those repos have been migrated and are available for permission sync
ALTER TABLE team_mappings ADD COLUMN repos_eligible INTEGER DEFAULT 0;

-- team_created_in_dest: Whether the team structure has been created in the destination org
ALTER TABLE team_mappings ADD COLUMN team_created_in_dest INTEGER DEFAULT 0;

-- last_synced_at: When permissions were last synced (different from migrated_at which is team creation time)
ALTER TABLE team_mappings ADD COLUMN last_synced_at TIMESTAMP;

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we leave the columns in place during downgrade
