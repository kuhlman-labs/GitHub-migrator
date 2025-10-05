-- Repositories table
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    full_name TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    source_url TEXT NOT NULL,
    
    -- Git properties
    total_size INTEGER,
    largest_file TEXT,
    largest_file_size INTEGER,
    largest_commit TEXT,
    largest_commit_size INTEGER,
    has_lfs BOOLEAN DEFAULT FALSE,
    has_submodules BOOLEAN DEFAULT FALSE,
    default_branch TEXT,
    branch_count INTEGER DEFAULT 0,
    commit_count INTEGER DEFAULT 0,
    
    -- GitHub features
    has_wiki BOOLEAN DEFAULT FALSE,
    has_pages BOOLEAN DEFAULT FALSE,
    has_discussions BOOLEAN DEFAULT FALSE,
    has_actions BOOLEAN DEFAULT FALSE,
    has_projects BOOLEAN DEFAULT FALSE,
    branch_protections INTEGER DEFAULT 0,
    environment_count INTEGER DEFAULT 0,
    secret_count INTEGER DEFAULT 0,
    variable_count INTEGER DEFAULT 0,
    webhook_count INTEGER DEFAULT 0,
    
    -- Contributors
    contributor_count INTEGER DEFAULT 0,
    top_contributors TEXT, -- JSON array
    
    -- Status
    status TEXT NOT NULL,
    batch_id INTEGER,
    priority INTEGER DEFAULT 0,
    
    -- Migration
    destination_url TEXT,
    destination_full_name TEXT,
    
    -- Timestamps
    discovered_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    migrated_at DATETIME,
    
    FOREIGN KEY (batch_id) REFERENCES batches(id)
);

CREATE INDEX idx_repositories_status ON repositories(status);
CREATE INDEX idx_repositories_batch_id ON repositories(batch_id);
CREATE INDEX idx_repositories_full_name ON repositories(full_name);

-- Migration history table
CREATE TABLE IF NOT EXISTS migration_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    phase TEXT NOT NULL,
    message TEXT,
    error_message TEXT,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    duration_seconds INTEGER,
    
    FOREIGN KEY (repository_id) REFERENCES repositories(id)
);

CREATE INDEX idx_migration_history_repo ON migration_history(repository_id);
CREATE INDEX idx_migration_history_status ON migration_history(status);

-- Migration logs table (for detailed troubleshooting)
CREATE TABLE IF NOT EXISTS migration_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    history_id INTEGER,
    level TEXT NOT NULL,  -- DEBUG, INFO, WARN, ERROR
    phase TEXT NOT NULL,
    operation TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT,  -- Additional context, JSON or text
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (repository_id) REFERENCES repositories(id),
    FOREIGN KEY (history_id) REFERENCES migration_history(id)
);

CREATE INDEX idx_migration_logs_repo ON migration_logs(repository_id);
CREATE INDEX idx_migration_logs_level ON migration_logs(level);
CREATE INDEX idx_migration_logs_timestamp ON migration_logs(timestamp);
CREATE INDEX idx_migration_logs_history ON migration_logs(history_id);

-- Batches table
CREATE TABLE IF NOT EXISTS batches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL, -- 'pilot', 'wave_1', etc.
    repository_count INTEGER DEFAULT 0,
    status TEXT NOT NULL,
    scheduled_at DATETIME,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL
);

CREATE INDEX idx_batches_status ON batches(status);
CREATE INDEX idx_batches_type ON batches(type);
