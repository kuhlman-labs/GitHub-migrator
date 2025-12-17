-- Add CODEOWNERS content and team reference fields to repositories
-- This enables parsing CODEOWNERS files and tracking team references for migration planning

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[repositories]') AND name = 'codeowners_content')
BEGIN
    ALTER TABLE repositories ADD codeowners_content NVARCHAR(MAX);
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[repositories]') AND name = 'codeowners_teams')
BEGIN
    ALTER TABLE repositories ADD codeowners_teams NVARCHAR(MAX);
END;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[repositories]') AND name = 'codeowners_users')
BEGIN
    ALTER TABLE repositories ADD codeowners_users NVARCHAR(MAX);
END;

-- +goose Down
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[repositories]') AND name = 'codeowners_users')
BEGIN
    ALTER TABLE repositories DROP COLUMN codeowners_users;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[repositories]') AND name = 'codeowners_teams')
BEGIN
    ALTER TABLE repositories DROP COLUMN codeowners_teams;
END;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'[dbo].[repositories]') AND name = 'codeowners_content')
BEGIN
    ALTER TABLE repositories DROP COLUMN codeowners_content;
END;
