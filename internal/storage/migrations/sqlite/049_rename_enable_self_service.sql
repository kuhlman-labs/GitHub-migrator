-- Rename auth_require_identity_mapping_for_self_service to auth_enable_self_service
-- The new name is clearer: EnableSelfService = true means self-service is enabled

-- SQLite doesn't support RENAME COLUMN in older versions, so we use ALTER TABLE ADD
-- and copy data. For newer SQLite (3.25.0+), we can use RENAME COLUMN.

-- First add the new column with the default value
ALTER TABLE settings ADD COLUMN auth_enable_self_service BOOLEAN NOT NULL DEFAULT 0;

-- Copy data from old column to new column (same semantics, no inversion needed)
UPDATE settings SET auth_enable_self_service = auth_require_identity_mapping_for_self_service;

-- Note: SQLite doesn't support DROP COLUMN in older versions
-- The old column will remain but be unused. It can be dropped in a future migration
-- if needed by recreating the table.

