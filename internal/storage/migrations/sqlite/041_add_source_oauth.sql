-- +goose Up
-- Add OAuth configuration fields to sources table for user self-service authentication

-- GitHub OAuth fields (for GitHub/GHES sources)
ALTER TABLE sources ADD COLUMN oauth_client_id TEXT;
ALTER TABLE sources ADD COLUMN oauth_client_secret TEXT;

-- Entra ID OAuth fields (for Azure DevOps sources)
ALTER TABLE sources ADD COLUMN entra_tenant_id TEXT;
ALTER TABLE sources ADD COLUMN entra_client_id TEXT;
ALTER TABLE sources ADD COLUMN entra_client_secret TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
-- For simplicity, we'll leave the columns in place on rollback
-- They will be NULL and ignored by the application

