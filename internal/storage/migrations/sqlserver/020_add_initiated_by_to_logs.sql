-- Migration: Add initiated_by column to migration_logs table
-- This tracks which GitHub user initiated a migration action when auth is enabled

ALTER TABLE migration_logs ADD initiated_by NVARCHAR(255);

CREATE NONCLUSTERED INDEX idx_migration_logs_initiated_by ON migration_logs(initiated_by);

