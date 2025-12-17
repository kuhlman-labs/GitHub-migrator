-- +goose Up
-- Add installed_apps column to repositories table
-- This stores a JSON array of GitHub App names installed on the repository
ALTER TABLE repositories ADD COLUMN installed_apps TEXT;

-- +goose Down
ALTER TABLE repositories DROP COLUMN IF EXISTS installed_apps;

