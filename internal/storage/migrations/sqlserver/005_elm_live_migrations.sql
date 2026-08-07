-- +goose Up
-- Enterprise Live Migrations (ELM) support.
-- migration_route records which corridor a repository takes: NULL or empty reads
-- as the GEI default, so no backfill is required and no existing row changes behavior.
IF NOT EXISTS (SELECT 1 FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'migration_route')
ALTER TABLE repositories ADD migration_route NVARCHAR(32);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_migration_route')
CREATE INDEX idx_repositories_migration_route ON repositories(migration_route);

-- Per-repository state of an in-flight or completed live migration.
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'elm_migrations')
CREATE TABLE elm_migrations (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    repository_id BIGINT NOT NULL UNIQUE REFERENCES repositories(id) ON DELETE CASCADE,
    elm_migration_id NVARCHAR(255) NOT NULL UNIQUE,
    elm_status NVARCHAR(64) NOT NULL,
    elm_phase NVARCHAR(64),
    cutover_ready BIT NOT NULL DEFAULT 0,
    progress_percent INTEGER,
    last_polled_at DATETIME2,
    last_error NVARCHAR(MAX),
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_elm_migrations_repo')
CREATE INDEX idx_elm_migrations_repo ON elm_migrations(repository_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_elm_migrations_cutover_ready')
CREATE INDEX idx_elm_migrations_cutover_ready ON elm_migrations(cutover_ready);

-- +goose Down
IF EXISTS (SELECT * FROM sys.tables WHERE name = 'elm_migrations')
DROP TABLE elm_migrations;

IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_migration_route')
DROP INDEX idx_repositories_migration_route ON repositories;

IF EXISTS (SELECT 1 FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'migration_route')
ALTER TABLE repositories DROP COLUMN migration_route;
