-- +goose Up
-- Add new discovery fields for migration complexity and verification
-- Migration: 002_add_discovery_fields.sql

-- Large files detection
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_large_files')
BEGIN
    ALTER TABLE repositories ADD has_large_files BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'large_file_count')
BEGIN
    ALTER TABLE repositories ADD large_file_count INT DEFAULT 0;
END
GO

-- Last commit information
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'last_commit_sha')
BEGIN
    ALTER TABLE repositories ADD last_commit_sha NVARCHAR(MAX);
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'last_commit_date')
BEGIN
    ALTER TABLE repositories ADD last_commit_date DATETIME2;
END
GO

-- Issue and PR counts for verification
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'issue_count')
BEGIN
    ALTER TABLE repositories ADD issue_count INT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'pull_request_count')
BEGIN
    ALTER TABLE repositories ADD pull_request_count INT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'tag_count')
BEGIN
    ALTER TABLE repositories ADD tag_count INT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'open_issue_count')
BEGIN
    ALTER TABLE repositories ADD open_issue_count INT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'open_pr_count')
BEGIN
    ALTER TABLE repositories ADD open_pr_count INT DEFAULT 0;
END
GO

-- Create indexes for commonly queried fields
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_large_files' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_large_files ON repositories(has_large_files);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_last_commit_date' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_last_commit_date ON repositories(last_commit_date);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
