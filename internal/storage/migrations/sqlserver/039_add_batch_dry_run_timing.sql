-- Add dry run timing columns to batches table
-- Tracks when batch dry run started, completed, and duration

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'dry_run_started_at')
BEGIN
    ALTER TABLE batches ADD dry_run_started_at DATETIME2;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'dry_run_completed_at')
BEGIN
    ALTER TABLE batches ADD dry_run_completed_at DATETIME2;
END
GO
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('batches') AND name = 'dry_run_duration_seconds')
BEGIN
    ALTER TABLE batches ADD dry_run_duration_seconds INT;
END
GO

