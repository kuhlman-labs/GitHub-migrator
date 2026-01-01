-- Add source_id to github_teams and github_users for multi-source support
-- This allows us to know which source a team or user was discovered from

-- Add source_id to github_teams
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('github_teams') AND name = 'source_id')
BEGIN
    ALTER TABLE github_teams ADD source_id BIGINT NULL;
    ALTER TABLE github_teams ADD CONSTRAINT FK_github_teams_source FOREIGN KEY (source_id) REFERENCES sources(id);
END

-- Add source_id to github_users  
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('github_users') AND name = 'source_id')
BEGIN
    ALTER TABLE github_users ADD source_id BIGINT NULL;
    ALTER TABLE github_users ADD CONSTRAINT FK_github_users_source FOREIGN KEY (source_id) REFERENCES sources(id);
END

-- Add indexes for faster lookups
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_teams_source_id')
    CREATE INDEX idx_github_teams_source_id ON github_teams(source_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_users_source_id')
    CREATE INDEX idx_github_users_source_id ON github_users(source_id);

