-- Add source_org and auto-match fields to user_mappings table
-- This enables organizing users by source organization and tracking auto-match results

-- +goose Up
ALTER TABLE user_mappings ADD COLUMN source_org TEXT;
ALTER TABLE user_mappings ADD COLUMN match_confidence INTEGER;
ALTER TABLE user_mappings ADD COLUMN match_reason TEXT;

-- Index for source org filtering
CREATE INDEX IF NOT EXISTS idx_user_mappings_source_org 
    ON user_mappings(source_org);

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we leave the columns in place during downgrade
DROP INDEX IF EXISTS idx_user_mappings_source_org;

