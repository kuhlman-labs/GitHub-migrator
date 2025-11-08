-- Add tag protection count field for GitHub repositories
-- Tag protection rules don't migrate with GEI and must be manually configured

ALTER TABLE repositories ADD COLUMN tag_protection_count INTEGER DEFAULT 0;

-- Create index for filtering repositories with tag protections
CREATE INDEX idx_repositories_tag_protection_count ON repositories(tag_protection_count);

