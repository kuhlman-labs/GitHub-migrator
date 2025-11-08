-- +goose Up
-- Add GitHub Enterprise Importer API limitation fields
-- These fields track repository size limits and migration exclusion options

-- Repository size validation (40 GiB limit)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_oversized_repository')
BEGIN
    ALTER TABLE repositories ADD has_oversized_repository BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'oversized_repository_details')
BEGIN
    ALTER TABLE repositories ADD oversized_repository_details NVARCHAR(MAX);
END
GO

-- Metadata size estimation (40 GiB metadata limit)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'estimated_metadata_size')
BEGIN
    ALTER TABLE repositories ADD estimated_metadata_size INT;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'metadata_size_details')
BEGIN
    ALTER TABLE repositories ADD metadata_size_details NVARCHAR(MAX);
END
GO

-- Migration exclusion flags (per-repository settings)
-- These flags control what gets migrated via GitHub Enterprise Importer API
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'exclude_releases')
BEGIN
    ALTER TABLE repositories ADD exclude_releases BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'exclude_attachments')
BEGIN
    ALTER TABLE repositories ADD exclude_attachments BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'exclude_metadata')
BEGIN
    ALTER TABLE repositories ADD exclude_metadata BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'exclude_git_data')
BEGIN
    ALTER TABLE repositories ADD exclude_git_data BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'exclude_owner_projects')
BEGIN
    ALTER TABLE repositories ADD exclude_owner_projects BIT DEFAULT 0;
END
GO

-- Create index for filtering repositories with size issues
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_oversized_repository' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_oversized_repository ON repositories(has_oversized_repository);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
