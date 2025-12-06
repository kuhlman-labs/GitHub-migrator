-- Create github_teams table to store GitHub team information
-- Teams are org-scoped, so the same team name can exist in different organizations

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'github_teams')
BEGIN
    CREATE TABLE github_teams (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        organization NVARCHAR(255) NOT NULL,
        slug NVARCHAR(255) NOT NULL,
        name NVARCHAR(255) NOT NULL,
        description NVARCHAR(MAX),
        privacy NVARCHAR(50) NOT NULL DEFAULT 'closed',
        discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        
        CONSTRAINT UQ_github_teams_org_slug UNIQUE(organization, slug)
    );
END;

-- Create junction table for team-repository relationships
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'github_team_repositories')
BEGIN
    CREATE TABLE github_team_repositories (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        team_id BIGINT NOT NULL,
        repository_id BIGINT NOT NULL,
        permission NVARCHAR(50) NOT NULL DEFAULT 'pull',
        discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        
        CONSTRAINT FK_github_team_repositories_team FOREIGN KEY (team_id) REFERENCES github_teams(id) ON DELETE CASCADE,
        CONSTRAINT FK_github_team_repositories_repo FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
        CONSTRAINT UQ_github_team_repositories UNIQUE(team_id, repository_id)
    );
END;

-- Index for finding teams by organization
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_teams_organization')
BEGIN
    CREATE INDEX idx_github_teams_organization ON github_teams(organization);
END;

-- Index for finding all team memberships for a repository
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_team_repositories_repo_id')
BEGIN
    CREATE INDEX idx_github_team_repositories_repo_id ON github_team_repositories(repository_id);
END;

-- Index for finding all repositories in a team
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_team_repositories_team_id')
BEGIN
    CREATE INDEX idx_github_team_repositories_team_id ON github_team_repositories(team_id);
END;

-- +goose Down
IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_team_repositories_team_id')
    DROP INDEX idx_github_team_repositories_team_id ON github_team_repositories;

IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_team_repositories_repo_id')
    DROP INDEX idx_github_team_repositories_repo_id ON github_team_repositories;

IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_github_teams_organization')
    DROP INDEX idx_github_teams_organization ON github_teams;

IF EXISTS (SELECT * FROM sys.tables WHERE name = 'github_team_repositories')
    DROP TABLE github_team_repositories;

IF EXISTS (SELECT * FROM sys.tables WHERE name = 'github_teams')
    DROP TABLE github_teams;

