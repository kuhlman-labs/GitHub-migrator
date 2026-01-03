-- Add OAuth client ID and secret columns to settings table
ALTER TABLE settings ADD auth_github_oauth_client_id NVARCHAR(255);
ALTER TABLE settings ADD auth_github_oauth_client_secret NVARCHAR(255);

