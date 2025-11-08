-- +goose Up
-- Add new discovery fields for migration complexity and verification
-- Migration: 002_add_discovery_fields.sql

-- Large files detection
ALTER TABLE repositories ADD COLUMN has_large_files BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN large_file_count INTEGER DEFAULT 0;

-- Last commit information
ALTER TABLE repositories ADD COLUMN last_commit_sha TEXT;
ALTER TABLE repositories ADD COLUMN last_commit_date TIMESTAMP;

-- Issue and PR counts for verification
ALTER TABLE repositories ADD COLUMN issue_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN pull_request_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN tag_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN open_issue_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN open_pr_count INTEGER DEFAULT 0;

-- Create indexes for commonly queried fields
CREATE INDEX IF NOT EXISTS idx_repositories_has_large_files ON repositories(has_large_files);
CREATE INDEX IF NOT EXISTS idx_repositories_last_commit_date ON repositories(last_commit_date);



-- +goose Down
-- Add rollback logic here
