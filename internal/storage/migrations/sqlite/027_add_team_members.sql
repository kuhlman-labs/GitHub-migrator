-- Add team members table to track which users belong to which teams
-- This enables user identity mapping and permission planning

-- +goose Up
CREATE TABLE IF NOT EXISTS github_team_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id INTEGER NOT NULL,
    login TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (team_id) REFERENCES github_teams(id) ON DELETE CASCADE,
    UNIQUE(team_id, login)
);

-- Index for finding all members of a team
CREATE INDEX IF NOT EXISTS idx_github_team_members_team_id 
    ON github_team_members(team_id);

-- Index for finding all teams a user belongs to
CREATE INDEX IF NOT EXISTS idx_github_team_members_login 
    ON github_team_members(login);

-- +goose Down
DROP INDEX IF EXISTS idx_github_team_members_login;
DROP INDEX IF EXISTS idx_github_team_members_team_id;
DROP TABLE IF EXISTS github_team_members;
