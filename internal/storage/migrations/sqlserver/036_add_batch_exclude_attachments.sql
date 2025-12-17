-- Add exclude_attachments column to batches table
-- This allows batch-level control over whether to exclude attachments during migration
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'exclude_attachments')
BEGIN
    ALTER TABLE batches ADD exclude_attachments BIT DEFAULT 0;
END

