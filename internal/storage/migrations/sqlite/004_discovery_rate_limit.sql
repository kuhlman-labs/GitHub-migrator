-- +goose Up
-- +goose NO TRANSACTION
-- Add rate_limit_reset_at column to discovery_progress table for tracking rate limit waits

-- +goose StatementBegin
ALTER TABLE discovery_progress ADD COLUMN rate_limit_reset_at DATETIME;
-- +goose StatementEnd

-- +goose Down
-- +goose NO TRANSACTION

-- +goose StatementBegin
ALTER TABLE discovery_progress DROP COLUMN rate_limit_reset_at;
-- +goose StatementEnd
