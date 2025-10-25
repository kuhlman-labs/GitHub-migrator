-- Backfill last_dry_run_at from migration_history for existing dry runs
-- This updates repositories that completed dry runs before timestamp tracking was added

UPDATE repositories
SET last_dry_run_at = (
    SELECT MAX(mh.completed_at)
    FROM migration_history mh
    WHERE mh.repository_id = repositories.id
    AND mh.phase = 'dry_run'
    AND mh.status = 'completed'
    AND mh.completed_at IS NOT NULL
)
WHERE EXISTS (
    SELECT 1
    FROM migration_history mh
    WHERE mh.repository_id = repositories.id
    AND mh.phase = 'dry_run'
    AND mh.status = 'completed'
);

-- Also backfill last_discovery_at for repositories that don't have it set
-- Use discovered_at as fallback
UPDATE repositories
SET last_discovery_at = discovered_at
WHERE last_discovery_at IS NULL;

