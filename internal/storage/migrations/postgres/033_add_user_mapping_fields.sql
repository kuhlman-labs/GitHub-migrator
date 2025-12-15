-- Add source_org and auto-match fields to user_mappings table
-- This enables organizing users by source organization and tracking auto-match results

-- +goose Up
ALTER TABLE user_mappings ADD COLUMN IF NOT EXISTS source_org VARCHAR(255);
ALTER TABLE user_mappings ADD COLUMN IF NOT EXISTS match_confidence INTEGER;
ALTER TABLE user_mappings ADD COLUMN IF NOT EXISTS match_reason VARCHAR(50);

-- Index for source org filtering
CREATE INDEX IF NOT EXISTS idx_user_mappings_source_org 
    ON user_mappings(source_org);

-- +goose Down
DROP INDEX IF EXISTS idx_user_mappings_source_org;
ALTER TABLE user_mappings DROP COLUMN IF EXISTS match_reason;
ALTER TABLE user_mappings DROP COLUMN IF EXISTS match_confidence;
ALTER TABLE user_mappings DROP COLUMN IF EXISTS source_org;

