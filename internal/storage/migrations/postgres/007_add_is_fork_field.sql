-- +goose Up
-- Add is_fork field to repositories table
ALTER TABLE repositories ADD COLUMN is_fork BOOLEAN DEFAULT FALSE;

-- Create index for efficient filtering by fork status
CREATE INDEX IF NOT EXISTS idx_repositories_is_fork ON repositories(is_fork);



-- +goose Down
-- Add rollback logic here
