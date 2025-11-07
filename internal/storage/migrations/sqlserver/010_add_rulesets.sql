-- +goose Up
-- Add rulesets field to repositories table
-- Rulesets are a newer version of branch protections that don't migrate with GEI APIs

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_rulesets')
BEGIN
    ALTER TABLE repositories ADD has_rulesets BIT DEFAULT 0;
END
GO

-- Add index for filtering by rulesets
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_rulesets' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_rulesets ON repositories(has_rulesets);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
