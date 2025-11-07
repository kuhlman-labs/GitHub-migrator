-- +goose Up
-- Add has_packages field to repositories table
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_packages')
BEGIN
    ALTER TABLE repositories ADD has_packages BIT DEFAULT 0;
END
GO

-- Create index for efficient filtering by package presence
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_packages' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_packages ON repositories(has_packages);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
