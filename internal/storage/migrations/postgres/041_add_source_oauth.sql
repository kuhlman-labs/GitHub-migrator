-- +goose Up
-- Add OAuth configuration fields to sources table for user self-service authentication

-- GitHub OAuth fields (for GitHub/GHES sources)
ALTER TABLE sources ADD COLUMN IF NOT EXISTS oauth_client_id TEXT;
ALTER TABLE sources ADD COLUMN IF NOT EXISTS oauth_client_secret TEXT;

-- Entra ID OAuth fields (for Azure DevOps sources)
ALTER TABLE sources ADD COLUMN IF NOT EXISTS entra_tenant_id TEXT;
ALTER TABLE sources ADD COLUMN IF NOT EXISTS entra_client_id TEXT;
ALTER TABLE sources ADD COLUMN IF NOT EXISTS entra_client_secret TEXT;

-- +goose Down
ALTER TABLE sources DROP COLUMN IF EXISTS oauth_client_id;
ALTER TABLE sources DROP COLUMN IF EXISTS oauth_client_secret;
ALTER TABLE sources DROP COLUMN IF EXISTS entra_tenant_id;
ALTER TABLE sources DROP COLUMN IF EXISTS entra_client_id;
ALTER TABLE sources DROP COLUMN IF EXISTS entra_client_secret;

