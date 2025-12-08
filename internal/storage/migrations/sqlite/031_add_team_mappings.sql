-- Add team_mappings table to map source teams to destination teams
-- This enables team permission migration planning

-- +goose Up
CREATE TABLE IF NOT EXISTS team_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_org TEXT NOT NULL,
    source_team_slug TEXT NOT NULL,
    source_team_name TEXT,
    destination_org TEXT,
    destination_team_slug TEXT,
    destination_team_name TEXT,
    mapping_status TEXT NOT NULL DEFAULT 'unmapped',
    auto_created INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(source_org, source_team_slug)
);

-- Index for mapping status filtering
CREATE INDEX IF NOT EXISTS idx_team_mappings_status 
    ON team_mappings(mapping_status);

-- Index for destination org filtering
CREATE INDEX IF NOT EXISTS idx_team_mappings_destination_org 
    ON team_mappings(destination_org);

-- +goose Down
DROP INDEX IF EXISTS idx_team_mappings_destination_org;
DROP INDEX IF EXISTS idx_team_mappings_status;
DROP TABLE IF EXISTS team_mappings;
