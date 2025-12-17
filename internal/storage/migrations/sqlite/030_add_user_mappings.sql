-- Add user_mappings table to map source users to destination users
-- This enables mannequin reclaim after migration

-- +goose Up
CREATE TABLE IF NOT EXISTS user_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_login TEXT NOT NULL UNIQUE,
    source_email TEXT,
    source_name TEXT,
    destination_login TEXT,
    destination_email TEXT,
    mapping_status TEXT NOT NULL DEFAULT 'unmapped',
    mannequin_id TEXT,
    mannequin_login TEXT,
    reclaim_status TEXT,
    reclaim_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for mapping status filtering
CREATE INDEX IF NOT EXISTS idx_user_mappings_status 
    ON user_mappings(mapping_status);

-- Index for source email lookups (for auto-matching)
CREATE INDEX IF NOT EXISTS idx_user_mappings_source_email 
    ON user_mappings(source_email);

-- Index for destination login lookups
CREATE INDEX IF NOT EXISTS idx_user_mappings_destination_login 
    ON user_mappings(destination_login);

-- +goose Down
DROP INDEX IF EXISTS idx_user_mappings_destination_login;
DROP INDEX IF EXISTS idx_user_mappings_source_email;
DROP INDEX IF EXISTS idx_user_mappings_status;
DROP TABLE IF EXISTS user_mappings;
