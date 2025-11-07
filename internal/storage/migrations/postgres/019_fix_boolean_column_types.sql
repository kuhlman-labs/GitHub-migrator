-- +goose Up
-- Fix column types from INTEGER to BOOLEAN for fields that were incorrectly created
-- This migration corrects columns added in migrations 015, 016, and 017
-- that should have been BOOLEAN but were created as INTEGER
-- NOTE: This is PostgreSQL-specific syntax. For SQLite, these statements are skipped
-- since SQLite treats booleans as 0/1 integers anyway.

-- For each column, we need to:
-- 1. Drop the default constraint
-- 2. Change the type with USING clause
-- 3. Set the new default

-- Migration 015 fields - repository validation flags
ALTER TABLE repositories ALTER COLUMN has_oversized_commits DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN has_oversized_commits TYPE BOOLEAN USING has_oversized_commits::boolean;
ALTER TABLE repositories ALTER COLUMN has_oversized_commits SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN has_long_refs DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN has_long_refs TYPE BOOLEAN USING has_long_refs::boolean;
ALTER TABLE repositories ALTER COLUMN has_long_refs SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN has_blocking_files DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN has_blocking_files TYPE BOOLEAN USING has_blocking_files::boolean;
ALTER TABLE repositories ALTER COLUMN has_blocking_files SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN has_large_file_warnings DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN has_large_file_warnings TYPE BOOLEAN USING has_large_file_warnings::boolean;
ALTER TABLE repositories ALTER COLUMN has_large_file_warnings SET DEFAULT FALSE;

-- Migration 016 fields - repository size and exclusion flags
ALTER TABLE repositories ALTER COLUMN has_oversized_repository DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN has_oversized_repository TYPE BOOLEAN USING has_oversized_repository::boolean;
ALTER TABLE repositories ALTER COLUMN has_oversized_repository SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN exclude_releases DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN exclude_releases TYPE BOOLEAN USING exclude_releases::boolean;
ALTER TABLE repositories ALTER COLUMN exclude_releases SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN exclude_attachments DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN exclude_attachments TYPE BOOLEAN USING exclude_attachments::boolean;
ALTER TABLE repositories ALTER COLUMN exclude_attachments SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN exclude_metadata DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN exclude_metadata TYPE BOOLEAN USING exclude_metadata::boolean;
ALTER TABLE repositories ALTER COLUMN exclude_metadata SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN exclude_git_data DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN exclude_git_data TYPE BOOLEAN USING exclude_git_data::boolean;
ALTER TABLE repositories ALTER COLUMN exclude_git_data SET DEFAULT FALSE;

ALTER TABLE repositories ALTER COLUMN exclude_owner_projects DROP DEFAULT;
ALTER TABLE repositories ALTER COLUMN exclude_owner_projects TYPE BOOLEAN USING exclude_owner_projects::boolean;
ALTER TABLE repositories ALTER COLUMN exclude_owner_projects SET DEFAULT FALSE;

-- Migration 017 fields - batch exclusion flags
ALTER TABLE batches ALTER COLUMN exclude_releases DROP DEFAULT;
ALTER TABLE batches ALTER COLUMN exclude_releases TYPE BOOLEAN USING exclude_releases::boolean;
ALTER TABLE batches ALTER COLUMN exclude_releases SET DEFAULT FALSE;



-- +goose Down
-- Add rollback logic here
