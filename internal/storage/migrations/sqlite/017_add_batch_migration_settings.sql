-- Add batch-level migration settings
-- These settings apply to all repositories in a batch unless overridden at the repository level

-- Destination organization for batch migrations
-- If set, repositories without a destination_full_name will use this org as their destination
ALTER TABLE batches ADD COLUMN destination_org TEXT;

-- Migration API to use (GEI or ELM)
-- Default to GEI (GitHub Enterprise Importer)
-- ELM (Enterprise Live Migrator) is for future implementation
ALTER TABLE batches ADD COLUMN migration_api TEXT DEFAULT 'GEI';

-- Exclude releases during migration
-- If true, releases will be skipped for all repos in this batch (unless repo overrides)
ALTER TABLE batches ADD COLUMN exclude_releases BOOLEAN DEFAULT FALSE;

-- Create index for filtering batches by migration API type
CREATE INDEX idx_batches_migration_api ON batches(migration_api);

