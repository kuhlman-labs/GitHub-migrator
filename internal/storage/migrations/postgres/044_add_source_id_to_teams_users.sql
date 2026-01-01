-- Add source_id to github_teams and github_users for multi-source support
-- This allows us to know which source a team or user was discovered from

-- Add source_id to github_teams
ALTER TABLE github_teams ADD COLUMN IF NOT EXISTS source_id BIGINT REFERENCES sources(id);

-- Add source_id to github_users  
ALTER TABLE github_users ADD COLUMN IF NOT EXISTS source_id BIGINT REFERENCES sources(id);

-- Add indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_github_teams_source_id ON github_teams(source_id);
CREATE INDEX IF NOT EXISTS idx_github_users_source_id ON github_users(source_id);

