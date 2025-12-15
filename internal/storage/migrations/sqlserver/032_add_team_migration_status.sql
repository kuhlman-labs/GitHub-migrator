-- Add team migration execution status fields to team_mappings table
-- This enables tracking the progress of team creation and permission sync

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'migration_status')
BEGIN
    ALTER TABLE team_mappings ADD migration_status NVARCHAR(50) DEFAULT 'pending';
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'migrated_at')
BEGIN
    ALTER TABLE team_mappings ADD migrated_at DATETIME2;
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'error_message')
BEGIN
    ALTER TABLE team_mappings ADD error_message NVARCHAR(MAX);
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'repos_synced')
BEGIN
    ALTER TABLE team_mappings ADD repos_synced INT DEFAULT 0;
END;

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_mappings_migration_status')
BEGIN
    CREATE INDEX idx_team_mappings_migration_status ON team_mappings(migration_status);
END;

-- +goose Down
IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_team_mappings_migration_status')
BEGIN
    DROP INDEX idx_team_mappings_migration_status ON team_mappings;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'repos_synced')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN repos_synced;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'error_message')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN error_message;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'migrated_at')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN migrated_at;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND name = 'migration_status')
BEGIN
    ALTER TABLE team_mappings DROP COLUMN migration_status;
END;

