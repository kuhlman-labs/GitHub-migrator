-- +goose Up
-- Add tag protection count field for GitHub repositories
-- Tag protection rules don't migrate with GEI and must be manually configured

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'tag_protection_count')
BEGIN
    ALTER TABLE repositories ADD tag_protection_count INT DEFAULT 0;
END
GO

-- Create index for filtering repositories with tag protections
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_tag_protection_count' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_tag_protection_count ON repositories(tag_protection_count);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
