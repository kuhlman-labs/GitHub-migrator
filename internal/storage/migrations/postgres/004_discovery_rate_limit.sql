-- +goose Up
-- Add rate_limit_reset_at column to discovery_progress table for tracking rate limit waits
ALTER TABLE discovery_progress ADD COLUMN IF NOT EXISTS rate_limit_reset_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE discovery_progress DROP COLUMN IF EXISTS rate_limit_reset_at;
