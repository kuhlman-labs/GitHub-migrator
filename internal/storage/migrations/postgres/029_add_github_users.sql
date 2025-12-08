-- Add github_users table to track discovered users/contributors
-- This enables user identity mapping for mannequin reclaim

-- +goose Up
CREATE TABLE IF NOT EXISTS github_users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    email VARCHAR(255),
    avatar_url TEXT,
    source_instance VARCHAR(255) NOT NULL,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Contribution stats (aggregated)
    commit_count INTEGER NOT NULL DEFAULT 0,
    issue_count INTEGER NOT NULL DEFAULT 0,
    pr_count INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,
    repository_count INTEGER NOT NULL DEFAULT 0
);

-- Index for email lookups (for auto-matching)
CREATE INDEX IF NOT EXISTS idx_github_users_email 
    ON github_users(email);

-- Index for source instance filtering
CREATE INDEX IF NOT EXISTS idx_github_users_source_instance 
    ON github_users(source_instance);

-- +goose Down
DROP INDEX IF EXISTS idx_github_users_source_instance;
DROP INDEX IF EXISTS idx_github_users_email;
DROP TABLE IF EXISTS github_users;
