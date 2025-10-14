-- Migration: Add rollback tracking
-- This migration documents the rollback status support.
-- No schema changes are required as the existing repositories table
-- already supports storing the 'rolled_back' status in the status column.

-- Valid status values now include:
-- - pending
-- - dry_run_queued
-- - dry_run_in_progress
-- - dry_run_complete
-- - dry_run_failed
-- - pre_migration
-- - archive_generating
-- - queued_for_migration
-- - migrating_content
-- - migration_complete
-- - migration_failed
-- - post_migration
-- - complete
-- - rolled_back (NEW)

-- The rolled_back status indicates that a successfully migrated repository
-- has been rolled back and is eligible for re-migration.

-- Rollback events are tracked in the migration_history table with phase='rollback'
-- and the rollback reason stored in the message field.

