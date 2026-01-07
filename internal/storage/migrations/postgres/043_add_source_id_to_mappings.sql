-- Add source_id to team_mappings and user_mappings for multi-source support
-- This allows us to know which source a team or user was discovered from

-- Add source_id to team_mappings
ALTER TABLE team_mappings ADD COLUMN IF NOT EXISTS source_id BIGINT REFERENCES sources(id);

-- Add source_id to user_mappings  
ALTER TABLE user_mappings ADD COLUMN IF NOT EXISTS source_id BIGINT REFERENCES sources(id);

-- Add indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_team_mappings_source_id ON team_mappings(source_id);
CREATE INDEX IF NOT EXISTS idx_user_mappings_source_id ON user_mappings(source_id);

