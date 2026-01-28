-- +goose Up
-- Add Copilot GH token column to settings table for CLI authentication
ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_gh_token TEXT;

-- Add model column to copilot_sessions table for per-session model tracking
ALTER TABLE copilot_sessions ADD COLUMN IF NOT EXISTS model TEXT;

-- +goose Down
ALTER TABLE copilot_sessions DROP COLUMN IF EXISTS model;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_gh_token;
