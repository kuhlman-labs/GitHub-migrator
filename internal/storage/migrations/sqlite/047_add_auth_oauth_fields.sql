-- Add OAuth client ID and secret columns to settings table
ALTER TABLE settings ADD COLUMN auth_github_oauth_client_id TEXT;
ALTER TABLE settings ADD COLUMN auth_github_oauth_client_secret TEXT;

