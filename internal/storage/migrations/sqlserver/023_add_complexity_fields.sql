-- Migration 023: Add complexity score fields to repositories table
-- This allows storing pre-calculated complexity scores for better performance
-- and proper support for both GitHub and Azure DevOps repositories

-- Add complexity_score column (integer, nullable)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'complexity_score')
BEGIN
    ALTER TABLE repositories ADD complexity_score INT NULL;
END;

-- Add complexity_breakdown column (NVARCHAR(MAX), nullable)  
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'complexity_breakdown')
BEGIN
    ALTER TABLE repositories ADD complexity_breakdown NVARCHAR(MAX) NULL;
END;

-- Add index on complexity_score for efficient sorting/filtering
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_complexity_score')
BEGIN
    CREATE INDEX idx_repositories_complexity_score ON repositories(complexity_score);
END;

