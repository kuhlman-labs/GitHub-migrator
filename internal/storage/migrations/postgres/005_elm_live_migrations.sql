-- +goose Up
-- Enterprise Live Migrations (ELM) support.
-- migration_route records which corridor a repository takes: NULL or empty reads
-- as the GEI default, so no backfill is required and no existing row changes behavior.
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS migration_route TEXT;

CREATE INDEX IF NOT EXISTS idx_repositories_migration_route ON repositories(migration_route);

-- Per-repository state of an in-flight or completed live migration.
CREATE TABLE IF NOT EXISTS elm_migrations (
    id BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL UNIQUE REFERENCES repositories(id) ON DELETE CASCADE,
    elm_migration_id TEXT NOT NULL UNIQUE,
    elm_status TEXT NOT NULL,
    elm_phase TEXT,
    cutover_ready BOOLEAN NOT NULL DEFAULT FALSE,
    progress_percent INTEGER,
    last_polled_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_elm_migrations_repo ON elm_migrations(repository_id);
CREATE INDEX IF NOT EXISTS idx_elm_migrations_cutover_ready ON elm_migrations(cutover_ready);

-- +goose Down
DROP TABLE IF EXISTS elm_migrations;
DROP INDEX IF EXISTS idx_repositories_migration_route;
ALTER TABLE repositories DROP COLUMN IF EXISTS migration_route;
