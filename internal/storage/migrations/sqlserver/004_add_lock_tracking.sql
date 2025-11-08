-- +goose Up
-- Add fields to track repository lock status and migration ID
-- This allows unlocking repositories that get stuck after failed migrations

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'source_migration_id')
BEGIN
    ALTER TABLE repositories ADD source_migration_id INT;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'is_source_locked')
BEGIN
    ALTER TABLE repositories ADD is_source_locked BIT DEFAULT 0;
END
GO

-- Index for finding locked repositories
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_source_locked' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_source_locked ON repositories(is_source_locked) WHERE is_source_locked = TRUE;
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
