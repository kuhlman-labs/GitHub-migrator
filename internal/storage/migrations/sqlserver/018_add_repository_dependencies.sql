-- Create repository_dependencies table to track dependencies between repositories
-- This supports batch planning by identifying which repos depend on each other

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repository_dependencies')
BEGIN
    CREATE TABLE repository_dependencies (
    id INT IDENTITY(1,1) PRIMARY KEY,
    repository_id INT NOT NULL,
    dependency_full_name NVARCHAR(MAX) NOT NULL,
    dependency_type NVARCHAR(MAX) NOT NULL CHECK(dependency_type IN ('submodule', 'workflow', 'dependency_graph', 'package')),
    dependency_url NVARCHAR(MAX) NOT NULL,
    is_local BIT NOT NULL DEFAULT 0,
    discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    metadata NVARCHAR(MAX),
    
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
    );
END
GO

-- Index for finding all dependencies of a repository (most common query)
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repository_dependencies_repo_id' AND object_id = OBJECT_ID('repository_dependencies'))
BEGIN
    CREATE INDEX idx_repository_dependencies_repo_id ON repository_dependencies(repository_id);
END
GO

-- Index for finding all repositories that depend on a specific dependency
-- This is crucial for batch planning - "what repos depend on X?"
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repository_dependencies_dep_name' AND object_id = OBJECT_ID('repository_dependencies'))
BEGIN
    CREATE INDEX idx_repository_dependencies_dep_name ON repository_dependencies(dependency_full_name);
END
GO

-- Index for filtering by type
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repository_dependencies_type' AND object_id = OBJECT_ID('repository_dependencies'))
BEGIN
    CREATE INDEX idx_repository_dependencies_type ON repository_dependencies(dependency_type);
END
GO

-- Index for local/external filtering
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repository_dependencies_is_local' AND object_id = OBJECT_ID('repository_dependencies'))
BEGIN
    CREATE INDEX idx_repository_dependencies_is_local ON repository_dependencies(is_local);
END
GO

-- Composite index for common batch planning queries (local dependencies by type)
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repository_dependencies_local_type' AND object_id = OBJECT_ID('repository_dependencies'))
BEGIN
    CREATE INDEX idx_repository_dependencies_local_type ON repository_dependencies(is_local, dependency_type);
END
GO

-- +goose Down
DROP INDEX IF EXISTS idx_repository_dependencies_local_type;
GO
DROP INDEX IF EXISTS idx_repository_dependencies_is_local;
GO
DROP INDEX IF EXISTS idx_repository_dependencies_type;
GO
DROP INDEX IF EXISTS idx_repository_dependencies_dep_name;
GO
DROP INDEX IF EXISTS idx_repository_dependencies_repo_id;
GO
DROP TABLE IF EXISTS repository_dependencies;
GO

