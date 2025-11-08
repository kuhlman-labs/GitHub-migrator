-- +goose Up
-- SQLite Note: SQLite's INTEGER type is dynamic and can store up to 8 bytes (BIGINT equivalent)
-- No changes needed for SQLite - this migration exists only for consistency with other dialects
-- SQLite automatically handles large integers without explicit BIGINT type

-- No-op for SQLite
SELECT 1;

-- +goose Down
-- No-op for SQLite
SELECT 1;

