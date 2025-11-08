-- +goose Up
-- Fix INT columns that can exceed 2^31-1 (2,147,483,647 bytes ~= 2GB)
-- Large repositories can have sizes in the billions of bytes
-- Change to BIGINT to support values up to 2^63-1 (~9 exabytes)

-- Repository size columns
ALTER TABLE repositories ALTER COLUMN total_size BIGINT;
ALTER TABLE repositories ALTER COLUMN largest_file_size BIGINT;
ALTER TABLE repositories ALTER COLUMN largest_commit_size BIGINT;
ALTER TABLE repositories ALTER COLUMN estimated_metadata_size BIGINT;
GO

-- +goose Down
-- Rollback to INT (data loss may occur if values exceed INT range)
ALTER TABLE repositories ALTER COLUMN total_size INT;
ALTER TABLE repositories ALTER COLUMN largest_file_size INT;
ALTER TABLE repositories ALTER COLUMN largest_commit_size INT;
ALTER TABLE repositories ALTER COLUMN estimated_metadata_size INT;
GO

