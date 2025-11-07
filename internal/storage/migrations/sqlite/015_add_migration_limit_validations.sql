-- Add GitHub migration limit validation fields
-- These fields track violations of GitHub's migration limitations:
-- - 2 GiB single commit limit
-- - 255 byte git reference name limit
-- - 400 MiB file size limit during migration (100 MiB post-migration)

-- Oversized commits (>2 GiB)
ALTER TABLE repositories ADD COLUMN has_oversized_commits BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN oversized_commit_details TEXT;

-- Long git references (>255 bytes)
ALTER TABLE repositories ADD COLUMN has_long_refs BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN long_ref_details TEXT;

-- Blocking files (>400 MiB during migration)
ALTER TABLE repositories ADD COLUMN has_blocking_files BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN blocking_file_details TEXT;

-- Large file warnings (100-400 MiB - allowed during migration, need post-migration remediation)
ALTER TABLE repositories ADD COLUMN has_large_file_warnings BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN large_file_warning_details TEXT;

-- Create indexes for filtering repositories with validation issues
CREATE INDEX idx_repositories_has_oversized_commits ON repositories(has_oversized_commits);
CREATE INDEX idx_repositories_has_long_refs ON repositories(has_long_refs);
CREATE INDEX idx_repositories_has_blocking_files ON repositories(has_blocking_files);
CREATE INDEX idx_repositories_has_large_file_warnings ON repositories(has_large_file_warnings);

