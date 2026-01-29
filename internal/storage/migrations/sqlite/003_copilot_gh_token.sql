-- +goose Up
-- +goose NO TRANSACTION
-- Add Copilot GH token column to settings table for CLI authentication
-- Note: SQLite doesn't support IF NOT EXISTS for ADD COLUMN.
-- Using NO TRANSACTION mode allows the statement to fail gracefully
-- if the column already exists (making migration idempotent for re-runs).

-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN copilot_gh_token TEXT;
-- +goose StatementEnd

-- Add model column to copilot_sessions table for per-session model tracking
-- +goose StatementBegin
ALTER TABLE copilot_sessions ADD COLUMN model TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose NO TRANSACTION
-- Note: DROP COLUMN requires SQLite 3.35.0+ (March 2021)
-- Using NO TRANSACTION mode allows statements to fail gracefully on older versions

-- +goose StatementBegin
ALTER TABLE copilot_sessions DROP COLUMN model;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE settings DROP COLUMN copilot_gh_token;
-- +goose StatementEnd
