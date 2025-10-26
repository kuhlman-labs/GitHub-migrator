# GitHub Migration Server - API Documentation

## Overview

The GitHub Migration Server provides a comprehensive REST API for managing repository discovery, profiling, batch organization, and migration execution. All endpoints return JSON responses and follow RESTful conventions.

**Base URL:** `http://localhost:8080`  
**API Version:** v1  
**Content-Type:** `application/json`

## Table of Contents

- [Health Check](#health-check)
- [Discovery](#discovery)
- [Repositories](#repositories)
- [Batches](#batches)
- [Migrations](#migrations)
- [Analytics](#analytics)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)

---

## Health Check

### GET /health

Health check endpoint to verify server status.

**Response 200 OK:**
```json
{
  "status": "healthy",
  "time": "2024-01-15T10:30:00Z"
}
```

**Example:**
```bash
curl http://localhost:8080/health
```

---

## Discovery

### POST /api/v1/discovery/start

Start repository discovery from the source GitHub system.

**Request Body:**
```json
{
  "organization": "acme-corp",
  "options": {
    "include_archived": false,
    "include_forks": true,
    "parallel_workers": 5
  }
}
```

**Response 202 Accepted:**
```json
{
  "status": "started",
  "organization": "acme-corp",
  "message": "Discovery started successfully"
}
```

**Response 400 Bad Request:**
```json
{
  "error": "Organization name is required"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/discovery/start \
  -H "Content-Type: application/json" \
  -d '{"organization": "acme-corp"}'
```

### GET /api/v1/discovery/status

Get the status of the current or last discovery operation.

**Response 200 OK:**
```json
{
  "status": "running",
  "organization": "acme-corp",
  "started_at": "2024-01-15T10:00:00Z",
  "repositories_found": 127,
  "repositories_profiled": 89,
  "current_repository": "acme-corp/api-gateway",
  "progress_percentage": 70
}
```

**Status Values:**
- `idle` - No discovery running
- `running` - Discovery in progress
- `completed` - Discovery completed successfully
- `failed` - Discovery failed

**Example:**
```bash
curl http://localhost:8080/api/v1/discovery/status
```

---

## Repositories

### GET /api/v1/repositories

List all repositories with optional filtering.

**Query Parameters:**
- `status` (string) - Filter by migration status (pending, profiled, ready, migrating, completed, failed)
- `batch_id` (int) - Filter by batch ID
- `source` (string) - Filter by source system
- `organization` (string) - Filter by organization name (supports comma-separated list)
- `search` (string) - Search repositories by name (case-insensitive)
- `sort_by` (string) - Sort order: name, size, org, updated (default: name)
- `available_for_batch` (bool) - Filter repositories available for batch assignment
- `min_size` (int) - Minimum size in bytes
- `max_size` (int) - Maximum size in bytes
- `size_category` (string) - Filter by size category: small (<100MB), medium (100MB-1GB), large (1GB-5GB), very_large (>5GB), unknown (supports comma-separated list)
- `complexity` (string) - Filter by complexity level: simple (score ≤3), medium (score 4-6), complex (score 7-9), very_complex (score ≥10). Scoring: size tier (0-3) × 3 + LFS (2) + submodules (2) + large files (4) + branch protections (1) (supports comma-separated list)
- `has_lfs` (bool) - Filter by LFS usage
- `has_submodules` (bool) - Filter by submodule presence
- `has_large_files` (bool) - Filter by large files (>100MB)
- `has_actions` (bool) - Filter by GitHub Actions presence
- `has_wiki` (bool) - Filter by wiki presence
- `has_pages` (bool) - Filter by GitHub Pages presence
- `has_discussions` (bool) - Filter by discussions presence
- `has_projects` (bool) - Filter by projects presence
- `has_branch_protections` (bool) - Filter by branch protection rules
- `is_archived` (bool) - Filter by archived status
- `limit` (int) - Limit number of results (enables pagination)
- `offset` (int) - Offset for pagination

**Response 200 OK:**
```json
[
  {
    "id": 1,
    "full_name": "acme-corp/api-gateway",
    "organization": "acme-corp",
    "name": "api-gateway",
    "description": "Main API Gateway service",
    "status": "ready",
    "size_kb": 15234,
    "stars": 45,
    "forks": 12,
    "open_issues": 8,
    "default_branch": "main",
    "is_private": true,
    "is_fork": false,
    "is_archived": false,
    "has_wiki": true,
    "has_pages": false,
    "has_issues": true,
    "has_projects": false,
    "has_actions": true,
    "has_lfs": true,
    "has_submodules": false,
    "protected_branches_count": 2,
    "webhooks_count": 3,
    "topics": ["api", "gateway", "microservices"],
    "language": "Go",
    "batch_id": 1,
    "priority": 5,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "discovered_at": "2024-01-15T10:00:00Z"
  }
]
```

**Examples:**
```bash
# Get all repositories
curl http://localhost:8080/api/v1/repositories

# Filter by status
curl "http://localhost:8080/api/v1/repositories?status=ready"

# Filter by batch
curl "http://localhost:8080/api/v1/repositories?batch_id=1"

# Filter by organization
curl "http://localhost:8080/api/v1/repositories?organization=acme-corp"

# Filter by multiple organizations
curl "http://localhost:8080/api/v1/repositories?organization=acme-corp,other-org"

# Search by name
curl "http://localhost:8080/api/v1/repositories?search=api"

# Filter by features
curl "http://localhost:8080/api/v1/repositories?has_lfs=true&has_actions=true"

# Filter by size range
curl "http://localhost:8080/api/v1/repositories?min_size=1000000&max_size=10000000"

# Filter archived repositories
curl "http://localhost:8080/api/v1/repositories?is_archived=true"

# Filter repositories with branch protections
curl "http://localhost:8080/api/v1/repositories?has_branch_protections=true"

# Filter by size category
curl "http://localhost:8080/api/v1/repositories?size_category=very_large"

# Filter by multiple size categories
curl "http://localhost:8080/api/v1/repositories?size_category=large,very_large"

# Filter by complexity
curl "http://localhost:8080/api/v1/repositories?complexity=high"

# Filter by multiple complexity levels
curl "http://localhost:8080/api/v1/repositories?complexity=high,very_high"

# Complex filter with sorting
curl "http://localhost:8080/api/v1/repositories?has_lfs=true&has_large_files=true&sort_by=size"

# Filter very large repos with high complexity
curl "http://localhost:8080/api/v1/repositories?size_category=very_large&complexity=high,very_high"

# Paginated results
curl "http://localhost:8080/api/v1/repositories?limit=50&offset=0"
```

### GET /api/v1/repositories/{fullName}

Get detailed information about a specific repository including migration history.

**Path Parameters:**
- `fullName` (string) - Repository full name (e.g., "acme-corp/api-gateway")

**Response 200 OK:**
```json
{
  "repository": {
    "id": 1,
    "full_name": "acme-corp/api-gateway",
    "organization": "acme-corp",
    "name": "api-gateway",
    "status": "migrating",
    "size_kb": 15234,
    "default_branch": "main",
    "has_lfs": true,
    "has_actions": true,
    "protected_branches_count": 2,
    "batch_id": 1,
    "priority": 5,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T11:00:00Z"
  },
  "history": [
    {
      "id": 1,
      "repository_id": 1,
      "migration_id": "mig_123456",
      "status": "migrating",
      "phase": "migration",
      "sub_phase": "migrating",
      "started_at": "2024-01-15T11:00:00Z",
      "completed_at": null,
      "error_message": null,
      "metadata": {
        "dry_run": false,
        "initiated_by": "admin",
        "batch_id": 1
      }
    }
  ]
}
```

**Response 404 Not Found:**
```json
{
  "error": "Repository not found"
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/repositories/acme-corp/api-gateway
```

### PATCH /api/v1/repositories/{fullName}

Update repository metadata (batch assignment, priority).

**Path Parameters:**
- `fullName` (string) - Repository full name

**Request Body:**
```json
{
  "batch_id": 2,
  "priority": 10
}
```

**Response 200 OK:**
```json
{
  "id": 1,
  "full_name": "acme-corp/api-gateway",
  "batch_id": 2,
  "priority": 10,
  "updated_at": "2024-01-15T12:00:00Z"
}
```

**Example:**
```bash
curl -X PATCH http://localhost:8080/api/v1/repositories/acme-corp/api-gateway \
  -H "Content-Type: application/json" \
  -d '{"batch_id": 2, "priority": 10}'
```

### POST /api/v1/repositories/{fullName}/rediscover

Re-run discovery and profiling for a specific repository to refresh its data.

**Path Parameters:**
- `fullName` (string) - Repository full name (URL-encoded)

**Response 202 Accepted:**
```json
{
  "message": "Re-discovery started",
  "full_name": "acme-corp/api-gateway",
  "status": "in_progress"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/repositories/acme-corp%2Fapi-gateway/rediscover
```

### POST /api/v1/repositories/{fullName}/unlock

Unlock a repository that was locked during migration.

**Path Parameters:**
- `fullName` (string) - Repository full name (URL-encoded)

**Response 200 OK:**
```json
{
  "message": "Repository unlocked successfully",
  "full_name": "acme-corp/api-gateway",
  "migration_id": 12345
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/repositories/acme-corp%2Fapi-gateway/unlock
```

### POST /api/v1/repositories/{fullName}/rollback

Rollback a completed migration by deleting the destination repository.

**Path Parameters:**
- `fullName` (string) - Repository full name (URL-encoded)

**Request Body (optional):**
```json
{
  "reason": "Migration validation failed"
}
```

**Response 200 OK:**
```json
{
  "message": "Repository rolled back successfully",
  "repository": {
    "id": 1,
    "full_name": "acme-corp/api-gateway",
    "status": "rolled_back",
    "destination_url": null
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/repositories/acme-corp%2Fapi-gateway/rollback \
  -H "Content-Type: application/json" \
  -d '{"reason": "Need to fix source repository first"}'
```

### POST /api/v1/repositories/{fullName}/mark-wont-migrate

Mark or unmark a repository as "won't migrate" to exclude it from migration planning.

**Path Parameters:**
- `fullName` (string) - Repository full name (URL-encoded)

**Request Body (optional):**
```json
{
  "unmark": false
}
```

**Response 200 OK:**
```json
{
  "message": "Repository marked as won't migrate",
  "repository": {
    "id": 1,
    "full_name": "acme-corp/legacy-app",
    "status": "wont_migrate",
    "batch_id": null
  }
}
```

**Example:**
```bash
# Mark repository as won't migrate
curl -X POST http://localhost:8080/api/v1/repositories/acme-corp%2Flegacy-app/mark-wont-migrate

# Unmark repository (change back to pending)
curl -X POST http://localhost:8080/api/v1/repositories/acme-corp%2Flegacy-app/mark-wont-migrate \
  -H "Content-Type: application/json" \
  -d '{"unmark": true}'
```

---

## Batches

### GET /api/v1/batches

List all migration batches.

**Response 200 OK:**
```json
[
  {
    "id": 1,
    "name": "Pilot Repositories",
    "description": "Initial pilot migration batch",
    "status": "completed",
    "repository_count": 5,
    "started_at": "2024-01-15T09:00:00Z",
    "completed_at": "2024-01-15T10:00:00Z",
    "created_at": "2024-01-14T15:00:00Z"
  },
  {
    "id": 2,
    "name": "Wave 1 - Critical Services",
    "description": "First wave of critical microservices",
    "status": "ready",
    "repository_count": 15,
    "created_at": "2024-01-15T10:00:00Z"
  }
]
```

**Batch Status Values:**
- `ready` - Batch created and ready to start
- `running` - Batch migration in progress
- `completed` - All repositories migrated successfully
- `failed` - Batch migration failed
- `partial` - Some repositories failed

**Example:**
```bash
curl http://localhost:8080/api/v1/batches
```

### POST /api/v1/batches

Create a new migration batch.

**Request Body:**
```json
{
  "name": "Wave 2 - Backend Services",
  "description": "Second wave containing backend microservices",
  "repository_ids": [10, 11, 12, 13, 14]
}
```

**Validation:**
- `name` is required and cannot be empty or whitespace-only
- `name` must be unique across all batches

**Response 201 Created:**
```json
{
  "id": 3,
  "name": "Wave 2 - Backend Services",
  "description": "Second wave containing backend microservices",
  "status": "ready",
  "repository_count": 5,
  "created_at": "2024-01-15T12:00:00Z"
}
```

**Response 400 Bad Request:**
```json
{
  "error": "Batch name is required"
}
```

**Response 409 Conflict:**
```json
{
  "error": "A batch with the name 'Wave 2 - Backend Services' already exists. Please choose a different name."
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/batches \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wave 2",
    "description": "Backend services",
    "repository_ids": [10, 11, 12]
  }'
```

### GET /api/v1/batches/{id}

Get detailed information about a specific batch including its repositories.

**Path Parameters:**
- `id` (int) - Batch ID

**Response 200 OK:**
```json
{
  "batch": {
    "id": 1,
    "name": "Pilot Repositories",
    "description": "Initial pilot migration batch",
    "status": "completed",
    "repository_count": 5,
    "started_at": "2024-01-15T09:00:00Z",
    "completed_at": "2024-01-15T10:00:00Z",
    "created_at": "2024-01-14T15:00:00Z"
  },
  "repositories": [
    {
      "id": 1,
      "full_name": "acme-corp/api-gateway",
      "status": "completed",
      "batch_id": 1,
      "priority": 10
    },
    {
      "id": 2,
      "full_name": "acme-corp/auth-service",
      "status": "completed",
      "batch_id": 1,
      "priority": 9
    }
  ]
}
```

**Response 404 Not Found:**
```json
{
  "error": "Batch not found"
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/batches/1
```

### POST /api/v1/batches/{id}/start

Start migration for all repositories in a batch.

**Path Parameters:**
- `id` (int) - Batch ID

**Query Parameters:**
- `dry_run` (bool) - Perform dry run validation (default: false)

**Response 202 Accepted:**
```json
{
  "batch_id": 1,
  "migration_ids": [101, 102, 103, 104, 105],
  "count": 5,
  "message": "Started migration for 5 repositories in batch 'Pilot Repositories'"
}
```

**Example:**
```bash
# Start actual migration
curl -X POST http://localhost:8080/api/v1/batches/1/start

# Start dry run
curl -X POST "http://localhost:8080/api/v1/batches/1/start?dry_run=true"
```

### POST /api/v1/batches/{id}/dry-run

Start dry run migration for all repositories in a batch.

**Path Parameters:**
- `id` (int) - Batch ID

**Request Body (optional):**
```json
{
  "only_pending": true
}
```

**Response 202 Accepted:**
```json
{
  "batch_id": 1,
  "batch_name": "Pilot Repositories",
  "dry_run_ids": [1, 2, 3, 4, 5],
  "count": 5,
  "skipped_count": 0,
  "message": "Started dry run for 5 repositories in batch 'Pilot Repositories'",
  "only_pending": true
}
```

**Example:**
```bash
# Start dry run for all eligible repositories
curl -X POST http://localhost:8080/api/v1/batches/1/dry-run

# Start dry run only for pending repositories
curl -X POST http://localhost:8080/api/v1/batches/1/dry-run \
  -H "Content-Type: application/json" \
  -d '{"only_pending": true}'
```

### POST /api/v1/batches/{id}/retry

Retry failed migrations in a batch.

**Path Parameters:**
- `id` (int) - Batch ID

**Request Body (optional):**
```json
{
  "repository_ids": [10, 11]
}
```

**Response 202 Accepted:**
```json
{
  "batch_id": 1,
  "batch_name": "Pilot Repositories",
  "retried_count": 2,
  "retried_ids": [10, 11],
  "message": "Queued 2 repositories for retry"
}
```

**Example:**
```bash
# Retry all failed repositories in batch
curl -X POST http://localhost:8080/api/v1/batches/1/retry

# Retry specific failed repositories
curl -X POST http://localhost:8080/api/v1/batches/1/retry \
  -H "Content-Type: application/json" \
  -d '{"repository_ids": [10, 11]}'
```

### DELETE /api/v1/batches/{id}

Delete a batch (only allowed for batches not in progress).

**Path Parameters:**
- `id` (int) - Batch ID

**Response 200 OK:**
```json
{
  "message": "Batch deleted successfully"
}
```

**Response 400 Bad Request:**
```json
{
  "error": "Cannot delete batch in 'in_progress' status"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/v1/batches/1
```

---

## Migrations

### POST /api/v1/migrations/start

Start migration for one or more repositories. Supports both repository IDs and full names (for self-service).

**Request Body (by IDs):**
```json
{
  "repository_ids": [1, 2, 3],
  "dry_run": false,
  "priority": 5
}
```

**Request Body (by Names - Self-Service):**
```json
{
  "full_names": ["acme-corp/api-gateway", "acme-corp/auth-service"],
  "dry_run": false,
  "priority": 5
}
```

**Response 202 Accepted:**
```json
{
  "migration_ids": [101, 102],
  "count": 2,
  "message": "Started migration for 2 repositories"
}
```

**Response 400 Bad Request:**
```json
{
  "error": "Must provide repository_ids or full_names"
}
```

**Response 404 Not Found:**
```json
{
  "error": "No repositories found"
}
```

**Examples:**
```bash
# Start migration by IDs
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{"repository_ids": [1, 2, 3], "dry_run": false}'

# Start migration by names (self-service)
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{
    "full_names": ["acme-corp/api-gateway", "acme-corp/auth-service"],
    "dry_run": false
  }'

# Start dry run
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{"repository_ids": [1], "dry_run": true}'
```

### GET /api/v1/migrations/{id}

Get the current status of a migration.

**Path Parameters:**
- `id` (int) - Migration ID

**Response 200 OK:**
```json
{
  "id": 101,
  "repository_id": 1,
  "repository_name": "acme-corp/api-gateway",
  "migration_id": "mig_123456",
  "status": "migrating",
  "phase": "migration",
  "sub_phase": "migrating",
  "started_at": "2024-01-15T11:00:00Z",
  "completed_at": null,
  "duration_seconds": 120,
  "error_message": null,
  "metadata": {
    "dry_run": false,
    "initiated_by": "admin",
    "batch_id": 1,
    "estimated_completion": "2024-01-15T11:10:00Z"
  }
}
```

**Migration Phases:**
1. `pending` - Migration queued
2. `dry_run` - Dry run validation
3. `pre_migration` - Pre-flight checks
4. `migration` - Active migration
   - Sub-phases: `archive` → `queue` → `migrating` → `complete`
5. `post_migration` - Post-migration tasks
6. `completed` - Fully migrated
7. `failed` - Migration failed

**Example:**
```bash
curl http://localhost:8080/api/v1/migrations/101
```

### GET /api/v1/migrations/{id}/history

Get complete migration history for a migration.

**Path Parameters:**
- `id` (int) - Migration ID

**Response 200 OK:**
```json
[
  {
    "id": 1,
    "repository_id": 1,
    "migration_id": "mig_123456",
    "status": "migrating",
    "phase": "migration",
    "sub_phase": "migrating",
    "started_at": "2024-01-15T11:00:00Z",
    "completed_at": null,
    "error_message": null,
    "metadata": {
      "dry_run": false,
      "batch_id": 1
    }
  }
]
```

**Example:**
```bash
curl http://localhost:8080/api/v1/migrations/101/history
```

### GET /api/v1/migrations/{id}/logs

Get migration logs with optional filtering.

**Path Parameters:**
- `id` (int) - Migration ID

**Query Parameters:**
- `level` (string) - Filter by log level (info, warn, error)
- `limit` (int) - Maximum number of logs to return (default: 100)

**Response 200 OK:**
```json
[
  {
    "id": 1,
    "migration_id": 101,
    "timestamp": "2024-01-15T11:00:00Z",
    "level": "info",
    "message": "Starting repository migration",
    "metadata": {
      "repository": "acme-corp/api-gateway",
      "phase": "pre_migration"
    }
  },
  {
    "id": 2,
    "migration_id": 101,
    "timestamp": "2024-01-15T11:00:05Z",
    "level": "info",
    "message": "Pre-flight checks passed",
    "metadata": {
      "checks": ["size", "lfs", "permissions"]
    }
  },
  {
    "id": 3,
    "migration_id": 101,
    "timestamp": "2024-01-15T11:00:10Z",
    "level": "info",
    "message": "Migration archive created",
    "metadata": {
      "archive_id": "arch_789012"
    }
  }
]
```

**Examples:**
```bash
# Get all logs
curl http://localhost:8080/api/v1/migrations/101/logs

# Filter by level
curl "http://localhost:8080/api/v1/migrations/101/logs?level=error"

# Limit results
curl "http://localhost:8080/api/v1/migrations/101/logs?limit=50"
```

---

## Self-Service

### POST /api/v1/self-service/migrate

Self-service migration endpoint that orchestrates repository discovery, batch creation, and migration execution.

**Request Body:**
```json
{
  "repositories": ["acme-corp/api-gateway", "acme-corp/web-app"],
  "mappings": {
    "acme-corp/api-gateway": "new-org/api-gateway",
    "acme-corp/web-app": "new-org/web-app"
  },
  "dry_run": true
}
```

**Request Parameters:**
- `repositories` (array, required) - List of repository full names to migrate
- `mappings` (object, optional) - Optional destination repository name mappings
- `dry_run` (boolean, required) - Whether to run in dry run mode

**Response 202 Accepted:**
```json
{
  "batch_id": 15,
  "batch_name": "Self-Service - 2025-01-15T10:30:00Z",
  "message": "Self-service dry run started for 2 repositories in batch 'Self-Service - 2025-01-15T10:30:00Z'",
  "total_repositories": 2,
  "newly_discovered": 1,
  "already_existed": 1,
  "discovery_errors": [],
  "execution_started": true
}
```

**Response 400 Bad Request:**
```json
{
  "error": "No repositories provided"
}
```

**Example:**
```bash
# Start dry run migration
curl -X POST http://localhost:8080/api/v1/self-service/migrate \
  -H "Content-Type: application/json" \
  -d '{
    "repositories": ["acme-corp/api-gateway", "acme-corp/web-app"],
    "mappings": {
      "acme-corp/api-gateway": "new-org/api-gateway"
    },
    "dry_run": true
  }'

# Start production migration
curl -X POST http://localhost:8080/api/v1/self-service/migrate \
  -H "Content-Type: application/json" \
  -d '{
    "repositories": ["acme-corp/api-gateway"],
    "dry_run": false
  }'
```

---

## Analytics

### GET /api/v1/analytics/summary

Get analytics summary with repository status breakdown.

**Response 200 OK:**
```json
{
  "total_repositories": 150,
  "by_status": {
    "pending": 20,
    "profiled": 15,
    "ready": 30,
    "migrating": 10,
    "completed": 70,
    "failed": 5
  },
  "total_size_gb": 234.5,
  "repositories_with_lfs": 45,
  "repositories_with_actions": 89,
  "repositories_with_submodules": 12,
  "average_size_mb": 1.6,
  "largest_repository": {
    "full_name": "acme-corp/monorepo",
    "size_gb": 15.2
  },
  "most_complex_repository": {
    "full_name": "acme-corp/legacy-platform",
    "complexity_score": 95,
    "has_lfs": true,
    "has_submodules": true,
    "protected_branches": 5
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/analytics/summary
```

### GET /api/v1/analytics/progress

Get migration progress metrics over time.

**Query Parameters:**
- `days` (int) - Number of days to include (default: 30)
- `group_by` (string) - Grouping interval: hour, day, week (default: day)

**Response 200 OK:**
```json
{
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z",
    "days": 30
  },
  "total_migrations": 150,
  "successful_migrations": 140,
  "failed_migrations": 10,
  "success_rate": 93.3,
  "average_duration_minutes": 8.5,
  "total_data_migrated_gb": 234.5,
  "daily_progress": [
    {
      "date": "2024-01-15",
      "migrations_completed": 12,
      "migrations_failed": 1,
      "data_migrated_gb": 18.3,
      "average_duration_minutes": 7.2
    },
    {
      "date": "2024-01-16",
      "migrations_completed": 15,
      "migrations_failed": 0,
      "data_migrated_gb": 22.1,
      "average_duration_minutes": 9.1
    }
  ],
  "by_batch": [
    {
      "batch_id": 1,
      "batch_name": "Pilot Repositories",
      "total": 5,
      "completed": 5,
      "failed": 0,
      "success_rate": 100
    }
  ]
}
```

**Example:**
```bash
# Get 30-day progress
curl http://localhost:8080/api/v1/analytics/progress

# Get 7-day progress grouped by hour
curl "http://localhost:8080/api/v1/analytics/progress?days=7&group_by=hour"
```

### GET /api/v1/analytics/executive-report

Get a comprehensive executive-level migration progress report with key metrics, velocity analysis, risk assessment, and completion projections.

**Query Parameters:**
- `organization` (string, optional) - Filter by organization name
- `batch_id` (string, optional) - Filter by batch ID

**Response 200 OK:**
```json
{
  "executive_summary": {
    "total_repositories": 500,
    "completion_percentage": 68.5,
    "migrated_count": 342,
    "in_progress_count": 23,
    "pending_count": 120,
    "failed_count": 15,
    "success_rate": 95.8,
    "estimated_completion_date": "2025-02-15",
    "days_remaining": 21,
    "first_migration_date": "2024-12-01T00:00:00Z",
    "report_generated_at": "2025-01-15T10:30:00Z"
  },
  "velocity_metrics": {
    "repos_per_day": 16.3,
    "repos_per_week": 114,
    "average_duration_sec": 450,
    "migration_trend": [
      {
        "date": "2025-01-01",
        "completed_migrations": 12,
        "cumulative_completed": 298
      }
    ]
  },
  "organization_progress": [
    {
      "organization": "acme-corp",
      "total": 150,
      "migrated": 120,
      "pending": 25,
      "in_progress": 3,
      "failed": 2,
      "completion_percentage": 80.0
    }
  ],
  "risk_analysis": {
    "high_complexity_pending": 8,
    "very_large_pending": 5,
    "failed_migrations": 15,
    "complexity_distribution": [
      {
        "complexity": "low",
        "count": 280
      },
      {
        "complexity": "medium",
        "count": 150
      },
      {
        "complexity": "high",
        "count": 70
      }
    ],
    "size_distribution": [
      {
        "size_category": "small (<100MB)",
        "count": 320
      },
      {
        "size_category": "medium (100MB-1GB)",
        "count": 150
      },
      {
        "size_category": "large (>1GB)",
        "count": 30
      }
    ]
  },
  "batch_performance": {
    "total_batches": 15,
    "completed_batches": 10,
    "in_progress_batches": 3,
    "pending_batches": 2
  },
  "feature_migration_status": {
    "total_with_lfs": 45,
    "total_with_actions": 120,
    "total_with_packages": 32,
    "total_with_environments": 18
  },
  "status_breakdown": {
    "pending": 120,
    "completed": 342,
    "migrating": 23,
    "failed": 15
  }
}
```

**Example:**
```bash
# Get full executive report
curl http://localhost:8080/api/v1/analytics/executive-report

# Get report for specific organization
curl "http://localhost:8080/api/v1/analytics/executive-report?organization=acme-corp"

# Get report for specific batch
curl "http://localhost:8080/api/v1/analytics/executive-report?batch_id=5"
```

### GET /api/v1/analytics/executive-report/export

Export executive report in CSV or JSON format for offline analysis and reporting.

**Query Parameters:**
- `format` (string, required) - Export format: `csv` or `json`
- `organization` (string, optional) - Filter by organization name
- `batch_id` (string, optional) - Filter by batch ID

**Response 200 OK (CSV):**
```
Content-Type: text/csv
Content-Disposition: attachment; filename=executive_migration_report.csv

[CSV content with executive summary, organization progress, and risk analysis]
```

**Response 200 OK (JSON):**
```json
{
  "report_metadata": {
    "generated_at": "2025-01-15T10:30:00Z",
    "report_type": "Executive Migration Progress Report",
    "version": "1.0"
  },
  "executive_summary": {
    "total_repositories": 500,
    "completion_percentage": 68.5,
    "migrated_count": 342,
    "in_progress_count": 23,
    "pending_count": 120,
    "failed_count": 15,
    "success_rate": 95.8,
    "estimated_completion_date": "2025-02-15",
    "days_remaining": 21
  },
  "velocity_metrics": {
    "repos_per_day": 16.3,
    "repos_per_week": 114,
    "average_duration_sec": 450
  },
  "organization_progress": [...],
  "complexity_distribution": [...],
  "size_distribution": [...],
  "feature_migration_status": {...},
  "batch_performance": {
    "completed_batches": 10,
    "in_progress_batches": 3,
    "pending_batches": 2
  },
  "status_breakdown": {...}
}
```

**Example:**
```bash
# Export as CSV
curl "http://localhost:8080/api/v1/analytics/executive-report/export?format=csv" \
  -o executive_report.csv

# Export as JSON for specific organization
curl "http://localhost:8080/api/v1/analytics/executive-report/export?format=json&organization=acme-corp" \
  -o acme_corp_report.json
```

---

## Error Handling

All API endpoints return consistent error responses.

### Error Response Format

```json
{
  "error": "Detailed error message",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-15T12:00:00Z"
}
```

### HTTP Status Codes

| Code | Description | Usage |
|------|-------------|-------|
| 200 | OK | Successful GET, PATCH requests |
| 201 | Created | Successful POST creating new resource |
| 202 | Accepted | Async operation started successfully |
| 400 | Bad Request | Invalid request body or parameters |
| 404 | Not Found | Resource not found |
| 500 | Internal Server Error | Server-side error |

### Common Error Scenarios

**Invalid Request Body:**
```json
{
  "error": "Invalid request body",
  "code": "INVALID_REQUEST"
}
```

**Resource Not Found:**
```json
{
  "error": "Repository not found",
  "code": "NOT_FOUND"
}
```

**Migration Not Allowed:**
```json
{
  "error": "Repository cannot be migrated: status is 'migrating'",
  "code": "MIGRATION_NOT_ALLOWED"
}
```

**Server Error:**
```json
{
  "error": "Failed to fetch repositories",
  "code": "INTERNAL_ERROR"
}
```

---

## Rate Limiting

The server implements intelligent rate limiting for GitHub API calls:

- **Primary Rate Limit:** 5,000 requests/hour (GitHub authenticated)
- **Secondary Rate Limit:** Dynamic based on response headers
- **Auto-Wait:** Server automatically waits when limits are exhausted
- **Retry Logic:** Exponential backoff for failed requests

### Rate Limit Headers

Response headers include rate limit information:

```
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4850
X-RateLimit-Reset: 1705324800
```

### Circuit Breaker

The server implements a circuit breaker pattern:

- **Closed:** Normal operation
- **Open:** Too many failures, requests blocked temporarily
- **Half-Open:** Testing if service recovered

---

## Pagination

Currently, the API returns all results. Future versions will implement pagination:

**Planned Headers:**
```
X-Total-Count: 150
X-Page: 1
X-Per-Page: 50
Link: <url?page=2>; rel="next", <url?page=3>; rel="last"
```

---

## Authentication

**Current Version:** No authentication (internal use only)

**Future Versions** will support:
- API Key authentication
- OAuth 2.0
- JWT tokens
- Role-based access control (RBAC)

---

## Webhooks

**Future Feature:** The server will support webhooks for event notifications:

**Planned Events:**
- `discovery.started`
- `discovery.completed`
- `migration.started`
- `migration.completed`
- `migration.failed`
- `batch.started`
- `batch.completed`

---

## Best Practices

### 1. Use Dry Run First

Always perform a dry run before actual migration:

```bash
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{"repository_ids": [1], "dry_run": true}'
```

### 2. Monitor Migration Status

Poll migration status to track progress:

```bash
# Get current status
curl http://localhost:8080/api/v1/migrations/101

# Get detailed logs
curl http://localhost:8080/api/v1/migrations/101/logs
```

### 3. Use Batches for Organization

Organize repositories into logical batches:

```bash
# Create batch
curl -X POST http://localhost:8080/api/v1/batches \
  -H "Content-Type: application/json" \
  -d '{"name": "Critical Services", "repository_ids": [1,2,3]}'

# Start batch migration
curl -X POST http://localhost:8080/api/v1/batches/1/start
```

### 4. Track Analytics

Monitor overall progress with analytics:

```bash
# Get summary
curl http://localhost:8080/api/v1/analytics/summary

# Get progress over time
curl http://localhost:8080/api/v1/analytics/progress?days=7
```

### 5. Self-Service by Name

Enable self-service migrations using repository names:

```bash
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{
    "full_names": ["acme-corp/my-app"],
    "dry_run": false
  }'
```

---

## Code Examples

### JavaScript/TypeScript

```typescript
const API_BASE = 'http://localhost:8080/api/v1';

// Start discovery
async function startDiscovery(org: string) {
  const response = await fetch(`${API_BASE}/discovery/start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ organization: org })
  });
  return response.json();
}

// Get repositories
async function getRepositories(filters?: Record<string, any>) {
  const params = new URLSearchParams(filters);
  const response = await fetch(`${API_BASE}/repositories?${params}`);
  return response.json();
}

// Start migration
async function startMigration(repoIds: number[], dryRun = false) {
  const response = await fetch(`${API_BASE}/migrations/start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      repository_ids: repoIds,
      dry_run: dryRun
    })
  });
  return response.json();
}
```

### Python

```python
import requests

API_BASE = 'http://localhost:8080/api/v1'

# Start discovery
def start_discovery(org):
    response = requests.post(
        f'{API_BASE}/discovery/start',
        json={'organization': org}
    )
    return response.json()

# Get repositories
def get_repositories(filters=None):
    response = requests.get(
        f'{API_BASE}/repositories',
        params=filters or {}
    )
    return response.json()

# Start migration
def start_migration(repo_ids, dry_run=False):
    response = requests.post(
        f'{API_BASE}/migrations/start',
        json={
            'repository_ids': repo_ids,
            'dry_run': dry_run
        }
    )
    return response.json()
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const apiBase = "http://localhost:8080/api/v1"

type DiscoveryRequest struct {
    Organization string `json:"organization"`
}

// Start discovery
func startDiscovery(org string) error {
    body, _ := json.Marshal(DiscoveryRequest{Organization: org})
    resp, err := http.Post(
        apiBase+"/discovery/start",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}

// Get repositories
func getRepositories() ([]Repository, error) {
    resp, err := http.Get(apiBase + "/repositories")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var repos []Repository
    json.NewDecoder(resp.Body).Decode(&repos)
    return repos, nil
}
```

---

## Support

For detailed information, see:
- [README.md](../README.md) - Project overview and quickstart
- [DEPLOYMENT.md](./DEPLOYMENT.md) - Deployment guide
- [OPERATIONS.md](./OPERATIONS.md) - Operations runbook
- [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) - Technical implementation details
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Development and contributing guide

---

**API Version:** 1.0.0  
**Last Updated:** October 2025  
**Status:** Production Ready

