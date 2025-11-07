-- +goose Up
-- Add GitHub migration limit validation fields
-- These fields track violations of GitHub's migration limitations:
-- - 2 GiB single commit limit
-- - 255 byte git reference name limit
-- - 400 MiB file size limit during migration (100 MiB post-migration)

-- Oversized commits (>2 GiB)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_oversized_commits')
BEGIN
    ALTER TABLE repositories ADD has_oversized_commits BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'oversized_commit_details')
BEGIN
    ALTER TABLE repositories ADD oversized_commit_details NVARCHAR(MAX);
END
GO

-- Long git references (>255 bytes)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_long_refs')
BEGIN
    ALTER TABLE repositories ADD has_long_refs BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'long_ref_details')
BEGIN
    ALTER TABLE repositories ADD long_ref_details NVARCHAR(MAX);
END
GO

-- Blocking files (>400 MiB during migration)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_blocking_files')
BEGIN
    ALTER TABLE repositories ADD has_blocking_files BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'blocking_file_details')
BEGIN
    ALTER TABLE repositories ADD blocking_file_details NVARCHAR(MAX);
END
GO

-- Large file warnings (100-400 MiB - allowed during migration, need post-migration remediation)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_large_file_warnings')
BEGIN
    ALTER TABLE repositories ADD has_large_file_warnings BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'large_file_warning_details')
BEGIN
    ALTER TABLE repositories ADD large_file_warning_details NVARCHAR(MAX);
END
GO

-- Create indexes for filtering repositories with validation issues
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_oversized_commits' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_oversized_commits ON repositories(has_oversized_commits);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_long_refs' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_long_refs ON repositories(has_long_refs);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_blocking_files' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_blocking_files ON repositories(has_blocking_files);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_large_file_warnings' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_large_file_warnings ON repositories(has_large_file_warnings);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
