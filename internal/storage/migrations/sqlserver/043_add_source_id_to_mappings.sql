-- Add source_id to team_mappings and user_mappings for multi-source support
-- This allows us to know which source a team or user was discovered from

-- Add source_id to team_mappings
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('team_mappings') AND name = 'source_id')
BEGIN
    ALTER TABLE team_mappings ADD source_id BIGINT NULL;
    ALTER TABLE team_mappings ADD CONSTRAINT FK_team_mappings_source FOREIGN KEY (source_id) REFERENCES sources(id);
END

-- Add source_id to user_mappings  
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('user_mappings') AND name = 'source_id')
BEGIN
    ALTER TABLE user_mappings ADD source_id BIGINT NULL;
    ALTER TABLE user_mappings ADD CONSTRAINT FK_user_mappings_source FOREIGN KEY (source_id) REFERENCES sources(id);
END

-- Add indexes for faster lookups
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_mappings_source_id')
    CREATE INDEX idx_team_mappings_source_id ON team_mappings(source_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_source_id')
    CREATE INDEX idx_user_mappings_source_id ON user_mappings(source_id);

