-- +goose Up
-- Add timestamp tracking for operations
-- last_discovery_at: When repository metadata was last refreshed during discovery
-- last_dry_run_at: When a dry run was last executed on this repository

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'last_discovery_at')
BEGIN
    ALTER TABLE repositories ADD last_discovery_at DATETIME2;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'last_dry_run_at')
BEGIN
    ALTER TABLE repositories ADD last_dry_run_at DATETIME2;
END
GO

-- Add batch-level operation timestamps
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'last_dry_run_at')
BEGIN
    ALTER TABLE batches ADD last_dry_run_at DATETIME2;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'last_migration_attempt_at')
BEGIN
    ALTER TABLE batches ADD last_migration_attempt_at DATETIME2;
END
GO

-- Initialize last_discovery_at with discovered_at for existing records
UPDATE repositories SET last_discovery_at = discovered_at WHERE discovered_at IS NOT NULL;



GO

-- +goose Down
-- Add rollback logic here
GO
