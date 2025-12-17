-- Add installed_apps column to repositories table
-- This stores a JSON array of GitHub App names installed on the repository
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'installed_apps')
BEGIN
    ALTER TABLE repositories ADD installed_apps NVARCHAR(MAX);
END

