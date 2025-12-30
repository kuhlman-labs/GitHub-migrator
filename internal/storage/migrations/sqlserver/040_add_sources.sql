-- Migration: Add sources table for multi-source support
-- This enables configuring multiple migration sources (GitHub, Azure DevOps)
-- that all migrate to a shared destination.

-- Create sources table
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'sources')
BEGIN
    CREATE TABLE sources (
        id INT IDENTITY(1,1) PRIMARY KEY,
        name NVARCHAR(255) NOT NULL UNIQUE,
        type NVARCHAR(50) NOT NULL,                    -- 'github' or 'azuredevops'
        base_url NVARCHAR(500) NOT NULL,
        token NVARCHAR(MAX) NOT NULL,
        organization NVARCHAR(255),                     -- Required for Azure DevOps
        app_id BIGINT,                                  -- GitHub App ID (optional)
        app_private_key NVARCHAR(MAX),                  -- GitHub App private key (optional)
        app_installation_id BIGINT,                     -- GitHub App installation ID (optional)
        is_active BIT NOT NULL DEFAULT 1,
        repository_count INT NOT NULL DEFAULT 0,
        last_sync_at DATETIME2,
        created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
    );
END;
GO

-- Create indexes for sources table
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_sources_type' AND object_id = OBJECT_ID('sources'))
BEGIN
    CREATE INDEX idx_sources_type ON sources(type);
END;
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_sources_is_active' AND object_id = OBJECT_ID('sources'))
BEGIN
    CREATE INDEX idx_sources_is_active ON sources(is_active);
END;
GO

-- Add source_id column to repositories table
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'source_id')
BEGIN
    ALTER TABLE repositories ADD source_id INT;
END;
GO

-- Add foreign key constraint
IF NOT EXISTS (SELECT * FROM sys.foreign_keys WHERE name = 'FK_repositories_source_id')
BEGIN
    ALTER TABLE repositories ADD CONSTRAINT FK_repositories_source_id
    FOREIGN KEY (source_id) REFERENCES sources(id);
END;
GO

-- Create index on source_id for efficient filtering
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_source_id' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_source_id ON repositories(source_id);
END;
GO

