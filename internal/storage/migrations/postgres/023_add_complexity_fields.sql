-- Migration 023: Add complexity score fields to repositories table
-- This allows storing pre-calculated complexity scores for better performance
-- and proper support for both GitHub and Azure DevOps repositories

-- Add complexity_score column (integer, nullable)
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS complexity_score INTEGER;

-- Add complexity_breakdown column (JSON/TEXT, nullable)  
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS complexity_breakdown TEXT;

-- Add index on complexity_score for efficient sorting/filtering
CREATE INDEX IF NOT EXISTS idx_repositories_complexity_score ON repositories(complexity_score);

-- Add comment explaining the fields
COMMENT ON COLUMN repositories.complexity_score IS 'Pre-calculated complexity score (0-100+) based on migration difficulty factors';
COMMENT ON COLUMN repositories.complexity_breakdown IS 'JSON breakdown of individual complexity factors for detailed analysis';

