-- +goose Up
-- Add is_archived column to repositories table

ALTER TABLE repositories ADD COLUMN is_archived BOOLEAN NOT NULL DEFAULT FALSE;

-- Create index for filtering archived repositories
CREATE INDEX IF NOT EXISTS IF NOT EXISTS idx_repositories_is_archived ON repositories(is_archived);



-- +goose Down
-- Add rollback logic here
