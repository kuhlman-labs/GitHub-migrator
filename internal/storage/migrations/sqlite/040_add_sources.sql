-- Migration: Add sources table for multi-source support
-- This enables configuring multiple migration sources (GitHub, Azure DevOps)
-- that all migrate to a shared destination.

-- Create sources table
CREATE TABLE IF NOT EXISTS sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,                    -- 'github' or 'azuredevops'
    base_url TEXT NOT NULL,
    token TEXT NOT NULL,
    organization TEXT,                      -- Required for Azure DevOps
    app_id INTEGER,                         -- GitHub App ID (optional)
    app_private_key TEXT,                   -- GitHub App private key (optional)
    app_installation_id INTEGER,            -- GitHub App installation ID (optional)
    is_active BOOLEAN NOT NULL DEFAULT 1,
    repository_count INTEGER NOT NULL DEFAULT 0,
    last_sync_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for sources table
CREATE INDEX idx_sources_type ON sources(type);
CREATE INDEX idx_sources_is_active ON sources(is_active);

-- Add source_id column to repositories table
-- This links each repository to its source configuration
ALTER TABLE repositories ADD COLUMN source_id INTEGER REFERENCES sources(id);

-- Create index on source_id for efficient filtering
CREATE INDEX idx_repositories_source_id ON repositories(source_id);

