-- +goose Up
-- +goose NO TRANSACTION
-- Enterprise Live Migrations (ELM) support.
-- migration_route records which corridor a repository takes: NULL or empty reads
-- as the GEI default, so no backfill is required and no existing row changes behavior.

-- +goose StatementBegin
ALTER TABLE repositories ADD COLUMN migration_route TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_repositories_migration_route ON repositories(migration_route);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS elm_migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL UNIQUE REFERENCES repositories(id) ON DELETE CASCADE,
    elm_migration_id TEXT NOT NULL UNIQUE,
    elm_status TEXT NOT NULL,
    elm_phase TEXT,
    cutover_ready INTEGER NOT NULL DEFAULT 0,
    progress_percent INTEGER,
    last_polled_at DATETIME,
    last_error TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_elm_migrations_repo ON elm_migrations(repository_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_elm_migrations_cutover_ready ON elm_migrations(cutover_ready);
-- +goose StatementEnd

-- +goose Down
-- +goose NO TRANSACTION

-- +goose StatementBegin
DROP TABLE IF EXISTS elm_migrations;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_repositories_migration_route;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE repositories DROP COLUMN migration_route;
-- +goose StatementEnd
