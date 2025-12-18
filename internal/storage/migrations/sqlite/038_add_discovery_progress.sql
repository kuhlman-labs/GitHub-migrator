-- Add discovery_progress table for tracking discovery operations
-- This table stores progress information for enterprise/org/repo discovery operations
CREATE TABLE IF NOT EXISTS discovery_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    discovery_type TEXT NOT NULL,           -- 'enterprise', 'organization', or 'repository'
    target TEXT NOT NULL,                   -- enterprise slug, org name, or 'org/repo'
    status TEXT NOT NULL DEFAULT 'in_progress', -- 'in_progress', 'completed', 'failed'
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    total_orgs INTEGER NOT NULL DEFAULT 0,
    processed_orgs INTEGER NOT NULL DEFAULT 0,
    current_org TEXT NOT NULL DEFAULT '',
    total_repos INTEGER NOT NULL DEFAULT 0,
    processed_repos INTEGER NOT NULL DEFAULT 0,
    phase TEXT NOT NULL DEFAULT 'listing_repos', -- 'listing_repos', 'profiling_repos', 'discovering_teams', 'discovering_members'
    error_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

-- Index on status for quickly finding active discoveries
CREATE INDEX IF NOT EXISTS idx_discovery_progress_status ON discovery_progress(status);


