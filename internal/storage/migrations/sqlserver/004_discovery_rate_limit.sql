-- +goose Up
-- Add rate_limit_reset_at column to discovery_progress table for tracking rate limit waits
IF NOT EXISTS (SELECT 1 FROM sys.columns WHERE object_id = OBJECT_ID('discovery_progress') AND name = 'rate_limit_reset_at')
BEGIN
    ALTER TABLE discovery_progress ADD rate_limit_reset_at DATETIME2;
END

-- +goose Down
IF EXISTS (SELECT 1 FROM sys.columns WHERE object_id = OBJECT_ID('discovery_progress') AND name = 'rate_limit_reset_at')
BEGIN
    ALTER TABLE discovery_progress DROP COLUMN rate_limit_reset_at;
END
