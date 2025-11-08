-- +goose Up
-- Batches table (must be created first due to foreign key constraint)
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'batches')
BEGIN
    CREATE TABLE batches (
    id INT IDENTITY(1,1) PRIMARY KEY,
    name NVARCHAR(MAX) NOT NULL,
    description NVARCHAR(MAX),
    type NVARCHAR(MAX) NOT NULL, -- 'pilot', 'wave_1', etc.
    repository_count INT DEFAULT 0,
    status NVARCHAR(MAX) NOT NULL,
    scheduled_at DATETIME2,
    started_at DATETIME2,
    completed_at DATETIME2,
    created_at DATETIME2 NOT NULL
    );
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_batches_status' AND object_id = OBJECT_ID('batches'))
BEGIN
    CREATE INDEX idx_batches_status ON batches(status);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_batches_type' AND object_id = OBJECT_ID('batches'))
BEGIN
    CREATE INDEX idx_batches_type ON batches(type);
END
GO

-- Repositories table
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'repositories')
BEGIN
    CREATE TABLE repositories (
    id INT IDENTITY(1,1) PRIMARY KEY,
    full_name NVARCHAR(MAX) NOT NULL UNIQUE,
    source NVARCHAR(MAX) NOT NULL,
    source_url NVARCHAR(MAX) NOT NULL,
    
    -- Git properties
    total_size INT,
    largest_file NVARCHAR(MAX),
    largest_file_size INT,
    largest_commit NVARCHAR(MAX),
    largest_commit_size INT,
    has_lfs BIT DEFAULT 0,
    has_submodules BIT DEFAULT 0,
    default_branch NVARCHAR(MAX),
    branch_count INT DEFAULT 0,
    commit_count INT DEFAULT 0,
    
    -- GitHub features
    has_wiki BIT DEFAULT 0,
    has_pages BIT DEFAULT 0,
    has_discussions BIT DEFAULT 0,
    has_actions BIT DEFAULT 0,
    has_projects BIT DEFAULT 0,
    branch_protections INT DEFAULT 0,
    environment_count INT DEFAULT 0,
    secret_count INT DEFAULT 0,
    variable_count INT DEFAULT 0,
    webhook_count INT DEFAULT 0,
    
    -- Contributors
    contributor_count INT DEFAULT 0,
    top_contributors NVARCHAR(MAX), -- JSON array
    
    -- Status
    status NVARCHAR(MAX) NOT NULL,
    batch_id INT,
    priority INT DEFAULT 0,
    
    -- Migration
    destination_url NVARCHAR(MAX),
    destination_full_name NVARCHAR(MAX),
    
    -- Timestamps
    discovered_at DATETIME2 NOT NULL,
    updated_at DATETIME2 NOT NULL,
    migrated_at DATETIME2,
    
    FOREIGN KEY (batch_id) REFERENCES batches(id)
    );
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_status' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_status ON repositories(status);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_batch_id' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_batch_id ON repositories(batch_id);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_repositories_full_name' AND object_id = OBJECT_ID('repositories'))
BEGIN
    CREATE INDEX idx_repositories_full_name ON repositories(full_name);
END
GO

-- Migration history table
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'migration_history')
BEGIN
    CREATE TABLE migration_history (
    id INT IDENTITY(1,1) PRIMARY KEY,
    repository_id INT NOT NULL,
    status NVARCHAR(MAX) NOT NULL,
    phase NVARCHAR(MAX) NOT NULL,
    message NVARCHAR(MAX),
    error_message NVARCHAR(MAX),
    started_at DATETIME2 NOT NULL,
    completed_at DATETIME2,
    duration_seconds INT,
    
    FOREIGN KEY (repository_id) REFERENCES repositories(id)
    );
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_history_repo' AND object_id = OBJECT_ID('migration_history'))
BEGIN
    CREATE INDEX idx_migration_history_repo ON migration_history(repository_id);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_history_status' AND object_id = OBJECT_ID('migration_history'))
BEGIN
    CREATE INDEX idx_migration_history_status ON migration_history(status);
END
GO

-- Migration logs table (for detailed troubleshooting)
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'migration_logs')
BEGIN
    CREATE TABLE migration_logs (
    id INT IDENTITY(1,1) PRIMARY KEY,
    repository_id INT NOT NULL,
    history_id INT,
    level NVARCHAR(MAX) NOT NULL,  -- DEBUG, INFO, WARN, ERROR
    phase NVARCHAR(MAX) NOT NULL,
    operation NVARCHAR(MAX) NOT NULL,
    message NVARCHAR(MAX) NOT NULL,
    details NVARCHAR(MAX),  -- Additional context, JSON or text
    timestamp DATETIME2 NOT NULL DEFAULT CURRENT_DATETIME2,
    
    FOREIGN KEY (repository_id) REFERENCES repositories(id),
    FOREIGN KEY (history_id) REFERENCES migration_history(id)
    );
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_logs_repo' AND object_id = OBJECT_ID('migration_logs'))
BEGIN
    CREATE INDEX idx_migration_logs_repo ON migration_logs(repository_id);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_logs_level' AND object_id = OBJECT_ID('migration_logs'))
BEGIN
    CREATE INDEX idx_migration_logs_level ON migration_logs(level);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_logs_timestamp' AND object_id = OBJECT_ID('migration_logs'))
BEGIN
    CREATE INDEX idx_migration_logs_timestamp ON migration_logs(timestamp);
END
GO
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_migration_logs_history' AND object_id = OBJECT_ID('migration_logs'))
BEGIN
    CREATE INDEX idx_migration_logs_history ON migration_logs(history_id);
END
GO


GO

-- +goose Down
-- Add rollback logic here
GO
