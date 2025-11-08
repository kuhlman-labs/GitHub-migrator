-- +goose Up
-- Fix INTEGER columns that can exceed 2^31-1 (2,147,483,647 bytes ~= 2GB)
-- Large repositories can have sizes in the billions of bytes
-- Change to BIGINT to support values up to 2^63-1 (~9 exabytes)

-- Repository size columns
ALTER TABLE repositories ALTER COLUMN total_size TYPE BIGINT;
ALTER TABLE repositories ALTER COLUMN largest_file_size TYPE BIGINT;
ALTER TABLE repositories ALTER COLUMN largest_commit_size TYPE BIGINT;
ALTER TABLE repositories ALTER COLUMN estimated_metadata_size TYPE BIGINT;

-- +goose Down
-- Rollback to INTEGER (data loss may occur if values exceed INTEGER range)
ALTER TABLE repositories ALTER COLUMN total_size TYPE INTEGER;
ALTER TABLE repositories ALTER COLUMN largest_file_size TYPE INTEGER;
ALTER TABLE repositories ALTER COLUMN largest_commit_size TYPE INTEGER;
ALTER TABLE repositories ALTER COLUMN estimated_metadata_size TYPE INTEGER;

