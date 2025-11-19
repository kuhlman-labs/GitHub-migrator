-- Migration 023: Add complexity score fields to repositories table
-- This allows storing pre-calculated complexity scores for better performance
-- and proper support for both GitHub and Azure DevOps repositories

-- Add complexity_score column (integer, nullable)
ALTER TABLE repositories ADD COLUMN complexity_score INTEGER;

-- Add complexity_breakdown column (TEXT, nullable)  
ALTER TABLE repositories ADD COLUMN complexity_breakdown TEXT;

-- Add index on complexity_score for efficient sorting/filtering
CREATE INDEX IF NOT EXISTS idx_repositories_complexity_score ON repositories(complexity_score);

