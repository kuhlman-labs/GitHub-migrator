-- +goose Up
-- Add is_archived column to repositories table

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'is_archived')
BEGIN
    ALTER TABLE repositories ADD is_archived BIT NOT NULL DEFAULT FALSE;
END
GO

-- Create index for filtering archived repositories
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_is_archived' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_is_archived ON repositories(is_archived);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
