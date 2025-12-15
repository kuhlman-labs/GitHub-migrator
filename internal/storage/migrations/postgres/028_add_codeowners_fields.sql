-- Add CODEOWNERS content and team reference fields to repositories
-- This enables parsing CODEOWNERS files and tracking team references for migration planning

-- +goose Up
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS codeowners_content TEXT;
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS codeowners_teams TEXT;
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS codeowners_users TEXT;

-- +goose Down
ALTER TABLE repositories DROP COLUMN IF EXISTS codeowners_users;
ALTER TABLE repositories DROP COLUMN IF EXISTS codeowners_teams;
ALTER TABLE repositories DROP COLUMN IF EXISTS codeowners_content;
