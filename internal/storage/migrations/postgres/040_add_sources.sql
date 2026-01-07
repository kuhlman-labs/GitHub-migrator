-- +goose Up
-- Migration: Add sources table for multi-source support
-- This enables configuring multiple migration sources (GitHub, Azure DevOps)
-- that all migrate to a shared destination.

-- Create sources table
CREATE TABLE IF NOT EXISTS sources (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,                    -- 'github' or 'azuredevops'
    base_url TEXT NOT NULL,
    token TEXT NOT NULL,
    organization TEXT,                      -- Required for Azure DevOps
    app_id BIGINT,                          -- GitHub App ID (optional)
    app_private_key TEXT,                   -- GitHub App private key (optional)
    app_installation_id BIGINT,             -- GitHub App installation ID (optional)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    repository_count INTEGER NOT NULL DEFAULT 0,
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for sources table
CREATE INDEX IF NOT EXISTS idx_sources_type ON sources(type);
CREATE INDEX IF NOT EXISTS idx_sources_is_active ON sources(is_active);

-- Add source_id column to repositories table
-- This links each repository to its source configuration
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS source_id INTEGER REFERENCES sources(id);

-- Create index on source_id for efficient filtering
CREATE INDEX IF NOT EXISTS idx_repositories_source_id ON repositories(source_id);

-- +goose Down
DROP INDEX IF EXISTS idx_repositories_source_id;
ALTER TABLE repositories DROP COLUMN IF EXISTS source_id;
DROP INDEX IF EXISTS idx_sources_is_active;
DROP INDEX IF EXISTS idx_sources_type;
DROP TABLE IF EXISTS sources;

