-- Add source_org and auto-match fields to user_mappings table
-- This enables organizing users by source organization and tracking auto-match results

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[user_mappings]') AND name = 'source_org')
BEGIN
    ALTER TABLE user_mappings ADD source_org NVARCHAR(255);
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[user_mappings]') AND name = 'match_confidence')
BEGIN
    ALTER TABLE user_mappings ADD match_confidence INT;
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[user_mappings]') AND name = 'match_reason')
BEGIN
    ALTER TABLE user_mappings ADD match_reason NVARCHAR(50);
END;

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_source_org')
BEGIN
    CREATE INDEX idx_user_mappings_source_org ON user_mappings(source_org);
END;

-- +goose Down
IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_source_org')
BEGIN
    DROP INDEX idx_user_mappings_source_org ON user_mappings;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[user_mappings]') AND name = 'match_reason')
BEGIN
    ALTER TABLE user_mappings DROP COLUMN match_reason;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[user_mappings]') AND name = 'match_confidence')
BEGIN
    ALTER TABLE user_mappings DROP COLUMN match_confidence;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[user_mappings]') AND name = 'source_org')
BEGIN
    ALTER TABLE user_mappings DROP COLUMN source_org;
END;

