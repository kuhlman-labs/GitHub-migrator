-- Migration: Add initiated_by column to migration_logs table
-- This tracks which GitHub user initiated a migration action when auth is enabled

ALTER TABLE migration_logs ADD COLUMN initiated_by TEXT;

CREATE INDEX IF NOT EXISTS idx_migration_logs_initiated_by ON migration_logs(initiated_by);

