-- +goose Up
-- Add timestamp tracking for operations
-- last_discovery_at: When repository metadata was last refreshed during discovery
-- last_dry_run_at: When a dry run was last executed on this repository

ALTER TABLE repositories ADD COLUMN last_discovery_at TIMESTAMP;
ALTER TABLE repositories ADD COLUMN last_dry_run_at TIMESTAMP;

-- Add batch-level operation timestamps
ALTER TABLE batches ADD COLUMN last_dry_run_at TIMESTAMP;
ALTER TABLE batches ADD COLUMN last_migration_attempt_at TIMESTAMP;

-- Initialize last_discovery_at with discovered_at for existing records
UPDATE repositories SET last_discovery_at = discovered_at WHERE discovered_at IS NOT NULL;



-- +goose Down
-- Add rollback logic here
