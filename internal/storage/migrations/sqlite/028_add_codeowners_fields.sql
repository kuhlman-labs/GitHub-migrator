-- Add CODEOWNERS content and team reference fields to repositories
-- This enables parsing CODEOWNERS files and tracking team references for migration planning

-- +goose Up
ALTER TABLE repositories ADD COLUMN codeowners_content TEXT;
ALTER TABLE repositories ADD COLUMN codeowners_teams TEXT;
ALTER TABLE repositories ADD COLUMN codeowners_users TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
-- For simplicity, we'll leave these columns in place during downgrade
