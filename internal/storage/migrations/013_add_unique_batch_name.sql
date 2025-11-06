-- Add unique constraint to batch names
-- This ensures batch names are distinct and prevents confusion

-- For PostgreSQL: Add unique constraint directly (will be handled by transform function)
-- For SQLite: This gets transformed to table recreation (handled in application code if needed)

-- Create a unique index on name (works for both SQLite and PostgreSQL)
-- Using CREATE UNIQUE INDEX is compatible with both databases and doesn't require dropping tables
CREATE UNIQUE INDEX IF NOT EXISTS idx_batches_name_unique ON batches(name);

