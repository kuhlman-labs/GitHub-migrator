-- +goose Up
-- Add advanced repository features for enhanced migration planning
-- This migration adds 11 new fields across 4 categories:
-- 1. Security & Compliance (GHAS features)
-- 2. Repository Settings (visibility, workflows)
-- 3. Infrastructure & Access (runners, collaborators, apps)
-- 4. Releases

-- Security & Compliance Features
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_code_scanning')
BEGIN
    ALTER TABLE repositories ADD has_code_scanning BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_dependabot')
BEGIN
    ALTER TABLE repositories ADD has_dependabot BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_secret_scanning')
BEGIN
    ALTER TABLE repositories ADD has_secret_scanning BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_codeowners')
BEGIN
    ALTER TABLE repositories ADD has_codeowners BIT DEFAULT 0;
END
GO

-- Repository Settings
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'visibility')
BEGIN
    ALTER TABLE repositories ADD visibility NVARCHAR(MAX) DEFAULT 'public';
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'workflow_count')
BEGIN
    ALTER TABLE repositories ADD workflow_count INT DEFAULT 0;
END
GO

-- Infrastructure & Access
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_self_hosted_runners')
BEGIN
    ALTER TABLE repositories ADD has_self_hosted_runners BIT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'collaborator_count')
BEGIN
    ALTER TABLE repositories ADD collaborator_count INT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'installed_apps_count')
BEGIN
    ALTER TABLE repositories ADD installed_apps_count INT DEFAULT 0;
END
GO

-- Releases
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'release_count')
BEGIN
    ALTER TABLE repositories ADD release_count INT DEFAULT 0;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'has_release_assets')
BEGIN
    ALTER TABLE repositories ADD has_release_assets BIT DEFAULT 0;
END
GO

-- Create indexes for commonly filtered fields
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_visibility' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_visibility ON repositories(visibility);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_code_scanning' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_code_scanning ON repositories(has_code_scanning);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_self_hosted_runners' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_self_hosted_runners ON repositories(has_self_hosted_runners);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_has_codeowners' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_has_codeowners ON repositories(has_codeowners);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
