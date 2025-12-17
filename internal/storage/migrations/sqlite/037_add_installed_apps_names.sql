-- Add installed_apps column to repositories table
-- This stores a JSON array of GitHub App names installed on the repository
ALTER TABLE repositories ADD COLUMN installed_apps TEXT;

