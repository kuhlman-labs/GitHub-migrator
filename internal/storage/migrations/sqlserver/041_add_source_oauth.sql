-- +goose Up
-- Add OAuth configuration fields to sources table for user self-service authentication

-- GitHub OAuth fields (for GitHub/GHES sources)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'oauth_client_id')
BEGIN
    ALTER TABLE sources ADD oauth_client_id NVARCHAR(MAX);
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'oauth_client_secret')
BEGIN
    ALTER TABLE sources ADD oauth_client_secret NVARCHAR(MAX);
END;

-- Entra ID OAuth fields (for Azure DevOps sources)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'entra_tenant_id')
BEGIN
    ALTER TABLE sources ADD entra_tenant_id NVARCHAR(MAX);
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'entra_client_id')
BEGIN
    ALTER TABLE sources ADD entra_client_id NVARCHAR(MAX);
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'entra_client_secret')
BEGIN
    ALTER TABLE sources ADD entra_client_secret NVARCHAR(MAX);
END;

-- +goose Down
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'oauth_client_id')
BEGIN
    ALTER TABLE sources DROP COLUMN oauth_client_id;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'oauth_client_secret')
BEGIN
    ALTER TABLE sources DROP COLUMN oauth_client_secret;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'entra_tenant_id')
BEGIN
    ALTER TABLE sources DROP COLUMN entra_tenant_id;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'entra_client_id')
BEGIN
    ALTER TABLE sources DROP COLUMN entra_client_id;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'sources') AND name = 'entra_client_secret')
BEGIN
    ALTER TABLE sources DROP COLUMN entra_client_secret;
END;

