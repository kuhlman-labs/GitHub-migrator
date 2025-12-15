-- Add fields to track partial vs. full team migration
-- This enables distinguishing between teams created before repo migrations vs. fully synced teams

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'total_source_repos')
BEGIN
    ALTER TABLE team_mappings ADD total_source_repos INT DEFAULT 0;
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'repos_eligible')
BEGIN
    ALTER TABLE team_mappings ADD repos_eligible INT DEFAULT 0;
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'team_created_in_dest')
BEGIN
    ALTER TABLE team_mappings ADD team_created_in_dest BIT DEFAULT 0;
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'last_synced_at')
BEGIN
    ALTER TABLE team_mappings ADD last_synced_at DATETIME2;
END;

-- +goose Down
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'last_synced_at')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN last_synced_at;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'team_created_in_dest')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN team_created_in_dest;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'repos_eligible')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN repos_eligible;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'total_source_repos')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN total_source_repos;
END;
