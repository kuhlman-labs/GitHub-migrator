-- Create github_teams table to store GitHub team information
-- Teams are org-scoped, so the same team name can exist in different organizations

-- +goose Up
CREATE TABLE IF NOT EXISTS github_teams (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    organization TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    privacy TEXT NOT NULL DEFAULT 'closed',
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(organization, slug)
);

-- Create junction table for team-repository relationships
CREATE TABLE IF NOT EXISTS github_team_repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id INTEGER NOT NULL,
    repository_id INTEGER NOT NULL,
    permission TEXT NOT NULL DEFAULT 'pull',
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (team_id) REFERENCES github_teams(id) ON DELETE CASCADE,
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    UNIQUE(team_id, repository_id)
);

-- Index for finding teams by organization
CREATE INDEX IF NOT EXISTS idx_github_teams_organization 
    ON github_teams(organization);

-- Index for finding all team memberships for a repository
CREATE INDEX IF NOT EXISTS idx_github_team_repositories_repo_id 
    ON github_team_repositories(repository_id);

-- Index for finding all repositories in a team
CREATE INDEX IF NOT EXISTS idx_github_team_repositories_team_id 
    ON github_team_repositories(team_id);

-- +goose Down
DROP INDEX IF EXISTS idx_github_team_repositories_team_id;
DROP INDEX IF EXISTS idx_github_team_repositories_repo_id;
DROP INDEX IF EXISTS idx_github_teams_organization;
DROP TABLE IF EXISTS github_team_repositories;
DROP TABLE IF EXISTS github_teams;

