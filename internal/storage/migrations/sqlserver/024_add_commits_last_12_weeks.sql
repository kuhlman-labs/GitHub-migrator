-- Migration 024: Add commits_last_12_weeks column to repositories table
-- This tracks commit activity over the past 12 weeks to provide insights into repository activity

-- Add commits_last_12_weeks column (integer, default 0)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'repositories') AND name = 'commits_last_12_weeks')
BEGIN
    ALTER TABLE repositories ADD commits_last_12_weeks INT NOT NULL DEFAULT 0;
END
GO

-- Add index on commits_last_12_weeks for efficient sorting/filtering
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE object_id = OBJECT_ID(N'repositories') AND name = 'idx_repositories_commits_last_12_weeks')
BEGIN
    CREATE INDEX idx_repositories_commits_last_12_weeks ON repositories(commits_last_12_weeks);
END
GO

-- Add extended property comment explaining the field
EXEC sys.sp_addextendedproperty 
    @name = N'MS_Description', 
    @value = N'Number of commits made in the past 12 weeks across all branches',
    @level0type = N'SCHEMA', @level0name = N'dbo',
    @level1type = N'TABLE',  @level1name = N'repositories',
    @level2type = N'COLUMN', @level2name = N'commits_last_12_weeks';
GO

