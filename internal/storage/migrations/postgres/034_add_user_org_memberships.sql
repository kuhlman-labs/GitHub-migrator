-- Add user_org_memberships table to track which users belong to which organizations
-- This enables organizing users by source organization for mannequin reclamation

-- +goose Up
CREATE TABLE IF NOT EXISTS user_org_memberships (
    id SERIAL PRIMARY KEY,
    user_login VARCHAR(255) NOT NULL,
    organization VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_login, organization)
);

-- Index for finding all orgs a user belongs to
CREATE INDEX IF NOT EXISTS idx_user_org_memberships_login 
    ON user_org_memberships(user_login);

-- Index for finding all users in an org
CREATE INDEX IF NOT EXISTS idx_user_org_memberships_org 
    ON user_org_memberships(organization);

-- +goose Down
DROP INDEX IF EXISTS idx_user_org_memberships_org;
DROP INDEX IF EXISTS idx_user_org_memberships_login;
DROP TABLE IF EXISTS user_org_memberships;

