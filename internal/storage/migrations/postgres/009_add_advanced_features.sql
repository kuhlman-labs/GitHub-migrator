-- +goose Up
-- Add advanced repository features for enhanced migration planning
-- This migration adds 11 new fields across 4 categories:
-- 1. Security & Compliance (GHAS features)
-- 2. Repository Settings (visibility, workflows)
-- 3. Infrastructure & Access (runners, collaborators, apps)
-- 4. Releases

-- Security & Compliance Features
ALTER TABLE repositories ADD COLUMN has_code_scanning BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN has_dependabot BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN has_secret_scanning BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN has_codeowners BOOLEAN DEFAULT FALSE;

-- Repository Settings
ALTER TABLE repositories ADD COLUMN visibility TEXT DEFAULT 'public'; -- 'public', 'private', 'internal'
ALTER TABLE repositories ADD COLUMN workflow_count INTEGER DEFAULT 0;

-- Infrastructure & Access
ALTER TABLE repositories ADD COLUMN has_self_hosted_runners BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN collaborator_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN installed_apps_count INTEGER DEFAULT 0;

-- Releases
ALTER TABLE repositories ADD COLUMN release_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN has_release_assets BOOLEAN DEFAULT FALSE;

-- Create indexes for commonly filtered fields
CREATE INDEX IF NOT EXISTS idx_repositories_visibility ON repositories(visibility);
CREATE INDEX IF NOT EXISTS idx_repositories_has_code_scanning ON repositories(has_code_scanning);
CREATE INDEX IF NOT EXISTS idx_repositories_has_self_hosted_runners ON repositories(has_self_hosted_runners);
CREATE INDEX IF NOT EXISTS idx_repositories_has_codeowners ON repositories(has_codeowners);



-- +goose Down
-- Add rollback logic here
