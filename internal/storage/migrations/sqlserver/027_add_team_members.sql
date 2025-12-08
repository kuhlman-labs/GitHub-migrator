-- Add team members table to track which users belong to which teams
-- This enables user identity mapping and permission planning

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[github_team_members]') AND type in (N'U'))
BEGIN
    CREATE TABLE github_team_members (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        team_id BIGINT NOT NULL,
        login NVARCHAR(255) NOT NULL,
        role NVARCHAR(50) NOT NULL DEFAULT 'member',
        discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        
        CONSTRAINT FK_github_team_members_team FOREIGN KEY (team_id) REFERENCES github_teams(id) ON DELETE CASCADE,
        CONSTRAINT UQ_github_team_members_team_login UNIQUE (team_id, login)
    );

    CREATE INDEX idx_github_team_members_team_id ON github_team_members(team_id);
    CREATE INDEX idx_github_team_members_login ON github_team_members(login);
END;

-- +goose Down
DROP TABLE IF EXISTS github_team_members;
