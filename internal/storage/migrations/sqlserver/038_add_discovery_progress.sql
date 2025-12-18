-- Add discovery_progress table for tracking discovery operations
-- This table stores progress information for enterprise/org/repo discovery operations
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'discovery_progress')
BEGIN
    CREATE TABLE discovery_progress (
        id INT IDENTITY(1,1) PRIMARY KEY,
        discovery_type NVARCHAR(50) NOT NULL,           -- 'enterprise', 'organization', or 'repository'
        target NVARCHAR(500) NOT NULL,                  -- enterprise slug, org name, or 'org/repo'
        status NVARCHAR(50) NOT NULL DEFAULT 'in_progress', -- 'in_progress', 'completed', 'failed'
        started_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        completed_at DATETIME2,
        total_orgs INT NOT NULL DEFAULT 0,
        processed_orgs INT NOT NULL DEFAULT 0,
        current_org NVARCHAR(500) NOT NULL DEFAULT '',
        total_repos INT NOT NULL DEFAULT 0,
        processed_repos INT NOT NULL DEFAULT 0,
        phase NVARCHAR(50) NOT NULL DEFAULT 'listing_repos', -- 'listing_repos', 'profiling_repos', 'discovering_teams', 'discovering_members'
        error_count INT NOT NULL DEFAULT 0,
        last_error NVARCHAR(MAX)
    );

    -- Index on status for quickly finding active discoveries
    CREATE INDEX idx_discovery_progress_status ON discovery_progress(status);
END


