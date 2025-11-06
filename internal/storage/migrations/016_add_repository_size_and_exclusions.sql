-- Add GitHub Enterprise Importer API limitation fields
-- These fields track repository size limits and migration exclusion options

-- Repository size validation (40 GiB limit)
ALTER TABLE repositories ADD COLUMN has_oversized_repository BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN oversized_repository_details TEXT;

-- Metadata size estimation (40 GiB metadata limit)
ALTER TABLE repositories ADD COLUMN estimated_metadata_size INTEGER;
ALTER TABLE repositories ADD COLUMN metadata_size_details TEXT;

-- Migration exclusion flags (per-repository settings)
-- These flags control what gets migrated via GitHub Enterprise Importer API
ALTER TABLE repositories ADD COLUMN exclude_releases BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN exclude_attachments BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN exclude_metadata BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN exclude_git_data BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN exclude_owner_projects BOOLEAN DEFAULT FALSE;

-- Create index for filtering repositories with size issues
CREATE INDEX idx_repositories_has_oversized_repository ON repositories(has_oversized_repository);

