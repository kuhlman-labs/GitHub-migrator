-- Add exclude_attachments column to batches table
-- This allows batch-level control over whether to exclude attachments during migration
ALTER TABLE batches ADD COLUMN exclude_attachments BOOLEAN DEFAULT FALSE;

