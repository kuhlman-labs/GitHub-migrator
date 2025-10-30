-- Add GitHub Enterprise Importer API limitation fields
-- These fields track repository size limits and migration exclusion options

-- Repository size validation (40 GiB limit)
ALTER TABLE repositories ADD COLUMN has_oversized_repository INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN oversized_repository_details TEXT;

-- Metadata size estimation (40 GiB metadata limit)
ALTER TABLE repositories ADD COLUMN estimated_metadata_size INTEGER;
ALTER TABLE repositories ADD COLUMN metadata_size_details TEXT;

-- Migration exclusion flags (per-repository settings)
-- These flags control what gets migrated via GitHub Enterprise Importer API
ALTER TABLE repositories ADD COLUMN exclude_releases INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN exclude_attachments INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN exclude_metadata INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN exclude_git_data INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN exclude_owner_projects INTEGER DEFAULT 0;

-- Create index for filtering repositories with size issues
CREATE INDEX idx_repositories_has_oversized_repository ON repositories(has_oversized_repository);

