-- +goose Up
-- Add batch-level migration settings
-- These settings apply to all repositories in a batch unless overridden at the repository level

-- Destination organization for batch migrations
-- If set, repositories without a destination_full_name will use this org as their destination
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'destination_org')
BEGIN
    ALTER TABLE batches ADD destination_org NVARCHAR(MAX);
END
GO

-- Migration API to use (GEI or ELM)
-- Default to GEI (GitHub Enterprise Importer)
-- ELM (Enterprise Live Migrator) is for future implementation
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'migration_api')
BEGIN
    ALTER TABLE batches ADD migration_api NVARCHAR(MAX) DEFAULT 'GEI';
END
GO

-- Exclude releases during migration
-- If true, releases will be skipped for all repos in this batch (unless repo overrides)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'exclude_releases')
BEGIN
    ALTER TABLE batches ADD exclude_releases BIT DEFAULT 0;
END
GO

-- Create index for filtering batches by migration API type
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_batches_migration_api' AND object_id = OBJECT_ID('batches'))
BEGIN
    CREATE INDEX idx_batches_migration_api ON batches(migration_api);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
