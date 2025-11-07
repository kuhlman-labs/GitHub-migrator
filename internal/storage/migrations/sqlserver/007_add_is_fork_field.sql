-- +goose Up
-- Add is_fork field to repositories table
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'is_fork')
BEGIN
    ALTER TABLE repositories ADD is_fork BIT DEFAULT 0;
END
GO

-- Create index for efficient filtering by fork status
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_is_fork' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_is_fork ON repositories(is_fork);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
