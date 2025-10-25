-- Add unique constraint to batch names
-- This ensures batch names are distinct and prevents confusion

-- Step 1: Create new batches table with unique constraint
CREATE TABLE batches_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    type TEXT NOT NULL,
    repository_count INTEGER DEFAULT 0,
    status TEXT NOT NULL,
    scheduled_at DATETIME,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL,
    last_dry_run_at DATETIME,
    last_migration_attempt_at DATETIME
);

-- Step 2: Copy data from old table to new table
INSERT INTO batches_new (id, name, description, type, repository_count, status, scheduled_at, started_at, completed_at, created_at, last_dry_run_at, last_migration_attempt_at)
SELECT id, name, description, type, repository_count, status, scheduled_at, started_at, completed_at, created_at, last_dry_run_at, last_migration_attempt_at
FROM batches;

-- Step 3: Drop old table
DROP TABLE batches;

-- Step 4: Rename new table to original name
ALTER TABLE batches_new RENAME TO batches;

-- Step 5: Recreate indexes
CREATE INDEX idx_batches_status ON batches(status);
CREATE INDEX idx_batches_type ON batches(type);

