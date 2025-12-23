-- Add dry run timing columns to batches table
-- Tracks when batch dry run started, completed, and duration

ALTER TABLE batches ADD COLUMN dry_run_started_at TIMESTAMP;
ALTER TABLE batches ADD COLUMN dry_run_completed_at TIMESTAMP;
ALTER TABLE batches ADD COLUMN dry_run_duration_seconds INTEGER;

