-- Create repository_dependencies table to track dependencies between repositories
-- This supports batch planning by identifying which repos depend on each other

-- +goose Up
CREATE TABLE IF NOT EXISTS repository_dependencies (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL,
    dependency_full_name TEXT NOT NULL,
    dependency_type TEXT NOT NULL CHECK(dependency_type IN ('submodule', 'workflow', 'dependency_graph', 'package')),
    dependency_url TEXT NOT NULL,
    is_local BOOLEAN NOT NULL DEFAULT FALSE,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT,
    
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- Index for finding all dependencies of a repository (most common query)
CREATE INDEX IF NOT EXISTS idx_repository_dependencies_repo_id 
    ON repository_dependencies(repository_id);

-- Index for finding all repositories that depend on a specific dependency
-- This is crucial for batch planning - "what repos depend on X?"
CREATE INDEX IF NOT EXISTS idx_repository_dependencies_dep_name 
    ON repository_dependencies(dependency_full_name);

-- Index for filtering by type
CREATE INDEX IF NOT EXISTS idx_repository_dependencies_type 
    ON repository_dependencies(dependency_type);

-- Index for local/external filtering
CREATE INDEX IF NOT EXISTS idx_repository_dependencies_is_local 
    ON repository_dependencies(is_local);

-- Composite index for common batch planning queries (local dependencies by type)
CREATE INDEX IF NOT EXISTS idx_repository_dependencies_local_type 
    ON repository_dependencies(is_local, dependency_type);

-- +goose Down
DROP INDEX IF EXISTS idx_repository_dependencies_local_type;
DROP INDEX IF EXISTS idx_repository_dependencies_is_local;
DROP INDEX IF EXISTS idx_repository_dependencies_type;
DROP INDEX IF EXISTS idx_repository_dependencies_dep_name;
DROP INDEX IF EXISTS idx_repository_dependencies_repo_id;
DROP TABLE IF EXISTS repository_dependencies;

