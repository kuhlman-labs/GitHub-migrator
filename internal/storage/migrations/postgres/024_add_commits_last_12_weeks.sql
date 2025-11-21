-- Migration 024: Add commits_last_12_weeks column to repositories table
-- This tracks commit activity over the past 12 weeks to provide insights into repository activity

-- Add commits_last_12_weeks column (integer, default 0)
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS commits_last_12_weeks INTEGER NOT NULL DEFAULT 0;

-- Add index on commits_last_12_weeks for efficient sorting/filtering
CREATE INDEX IF NOT EXISTS idx_repositories_commits_last_12_weeks ON repositories(commits_last_12_weeks);

-- Add comment explaining the field
COMMENT ON COLUMN repositories.commits_last_12_weeks IS 'Number of commits made in the past 12 weeks across all branches';

