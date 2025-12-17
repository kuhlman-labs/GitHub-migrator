-- Add github_users table to track discovered users/contributors
-- This enables user identity mapping for mannequin reclaim

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[github_users]') AND type in (N'U'))
BEGIN
    CREATE TABLE github_users (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        login NVARCHAR(255) NOT NULL UNIQUE,
        name NVARCHAR(255),
        email NVARCHAR(255),
        avatar_url NVARCHAR(MAX),
        source_instance NVARCHAR(255) NOT NULL,
        discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        
        -- Contribution stats (aggregated)
        commit_count INT NOT NULL DEFAULT 0,
        issue_count INT NOT NULL DEFAULT 0,
        pr_count INT NOT NULL DEFAULT 0,
        comment_count INT NOT NULL DEFAULT 0,
        repository_count INT NOT NULL DEFAULT 0
    );

    CREATE INDEX idx_github_users_email ON github_users(email);
    CREATE INDEX idx_github_users_source_instance ON github_users(source_instance);
END;

-- +goose Down
DROP TABLE IF EXISTS github_users;
