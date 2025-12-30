# GitHub Migrator - API Documentation

## Overview

The GitHub Migrator provides a comprehensive REST API for managing repository discovery, profiling, batch organization, and migration execution. All endpoints return JSON responses and follow RESTful conventions.

**Base URL:** `http://localhost:8080`  
**API Version:** v1  
**Content-Type:** `application/json`  
**OpenAPI Spec:** [openapi.json](./openapi.json)

## Table of Contents

- [Health Check](#health-check)
- [Configuration](#configuration)
- [Authentication](#authentication)
- [Setup](#setup)
- [Sources](#sources)
- [Discovery](#discovery)
- [Repositories](#repositories)
- [Organizations & Projects](#organizations--projects)
- [Dashboard](#dashboard)
- [Batches](#batches)
- [Migrations](#migrations)
- [Analytics](#analytics)
- [Azure DevOps](#azure-devops)
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

---

## Configuration

### GET /api/v1/config

Get application configuration for the frontend.

**Response 200 OK:**
```json
{
  "source_type": "github",
  "auth_enabled": true,
  "entraid_enabled": false
}
```

---

## Authentication

GitHub Migrator supports OAuth authentication via GitHub and Microsoft Entra ID.

### GET /api/v1/auth/login

Initiate GitHub OAuth login. Redirects to GitHub authorization page.

**Response 302:** Redirect to GitHub OAuth

### GET /api/v1/auth/callback

OAuth callback endpoint. Handles authorization code exchange and issues JWT.

**Query Parameters:**
- `code` (string) - Authorization code from GitHub
- `state` (string) - CSRF protection state

**Response 302:** Redirect to frontend with token

### GET /api/v1/auth/config

Get authentication configuration.

**Response 200 OK:**
```json
{
  "enabled": true,
  "login_url": "/api/v1/auth/login"
}
```

### POST /api/v1/auth/logout

Logout and invalidate session. Requires authentication.

**Response 200 OK:**
```json
{
  "message": "Logged out successfully"
}
```

### GET /api/v1/auth/user

Get current authenticated user information.

**Response 200 OK:**
```json
{
  "login": "username",
  "name": "User Name",
  "avatar_url": "https://avatars.githubusercontent.com/...",
  "is_enterprise_admin": false,
  "is_privileged": true
}
```

### POST /api/v1/auth/refresh

Refresh authentication token.

**Response 200 OK:**
```json
{
  "token": "eyJ...",
  "expires_at": "2024-01-16T10:30:00Z"
}
```

---

## Setup

Initial configuration endpoints for setting up the application.

### GET /api/v1/setup/status

Get current setup status.

**Response 200 OK:**
```json
{
  "configured": true,
  "source_configured": true,
  "destination_configured": true,
  "database_configured": true
}
```

### POST /api/v1/setup/validate-source

Validate source GitHub/ADO connection.

**Request Body:**
```json
{
  "type": "github",
  "base_url": "https://github.company.com/api/v3",
  "token": "ghp_xxxxxxxxxxxx"
}
```

**Response 200 OK:**
```json
{
  "valid": true,
  "message": "Connection successful",
  "details": {
    "user": "admin",
    "scopes": ["repo", "admin:org"]
  }
}
```

### POST /api/v1/setup/validate-destination

Validate destination GitHub connection.

### POST /api/v1/setup/validate-database

Validate database connection.

**Request Body:**
```json
{
  "type": "postgresql",
  "dsn": "host=localhost port=5432 user=migrator password=secret dbname=migrator"
}
```

### POST /api/v1/setup/apply

Apply configuration and write env file.

---

## Sources

Multi-source configuration endpoints for managing GitHub and Azure DevOps migration sources.
Sources allow configuring multiple source systems that all migrate to a shared destination.

### GET /api/v1/sources

List all configured sources.

**Query Parameters:**
- `active` (boolean, optional) - If `true`, only return active sources

**Response 200 OK:**
```json
[
  {
    "id": 1,
    "name": "GHES Production",
    "type": "github",
    "base_url": "https://github.company.com/api/v3",
    "has_app_auth": false,
    "is_active": true,
    "repository_count": 150,
    "last_sync_at": "2024-12-28T10:30:00Z",
    "created_at": "2024-12-01T09:00:00Z",
    "updated_at": "2024-12-28T10:30:00Z",
    "masked_token": "ghp_...xxxx"
  }
]
```

### POST /api/v1/sources

Create a new source.

**Request Body:**
```json
{
  "name": "GHES Production",
  "type": "github",
  "base_url": "https://github.company.com/api/v3",
  "token": "ghp_xxxxxxxxxxxxxxxxxxxx",
  "organization": "my-org"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Unique, user-friendly name for this source |
| type | string | Yes | Source type: `github` or `azuredevops` |
| base_url | string | Yes | API base URL for the source |
| token | string | Yes | Personal Access Token |
| organization | string | ADO only | Required for Azure DevOps sources |
| app_id | number | No | GitHub App ID for App authentication |
| app_private_key | string | No | GitHub App private key |
| app_installation_id | number | No | GitHub App installation ID |

**Response 201 Created:**
```json
{
  "id": 1,
  "name": "GHES Production",
  "type": "github",
  "base_url": "https://github.company.com/api/v3",
  "is_active": true,
  "repository_count": 0,
  "created_at": "2024-12-29T09:00:00Z",
  "updated_at": "2024-12-29T09:00:00Z",
  "masked_token": "ghp_...xxxx"
}
```

### GET /api/v1/sources/{id}

Get a single source by ID.

**Response 200 OK:** Source object (same format as list response)

### PUT /api/v1/sources/{id}

Update an existing source.

**Request Body:** Same as create, all fields optional

**Response 200 OK:** Updated source object

### DELETE /api/v1/sources/{id}

Delete a source. Fails if repositories are associated with the source.

**Response 204 No Content:** Success
**Response 409 Conflict:** Source has associated repositories

### POST /api/v1/sources/validate

Validate a source connection with inline credentials.

**Request Body:**
```json
{
  "type": "github",
  "base_url": "https://api.github.com",
  "token": "ghp_xxxxxxxxxxxx",
  "organization": "my-org"
}
```

**Response 200 OK:**
```json
{
  "valid": true,
  "details": {
    "authenticated_user": "admin",
    "connection_status": "connected"
  }
}
```

### POST /api/v1/sources/{id}/validate

Validate connection using stored source credentials.

**Response 200 OK:** Same as inline validation

### POST /api/v1/sources/{id}/set-active

Set a source's active/inactive status.

**Request Body:**
```json
{
  "is_active": true
}
```

**Response 200 OK:**
```json
{
  "success": true,
  "source_id": 1,
  "is_active": true
}
```

### GET /api/v1/sources/{id}/repositories

Get all repositories associated with a source.

**Response 200 OK:** Array of Repository objects

---

## Discovery

### POST /api/v1/discovery/start

Start repository discovery from the source GitHub system.

**Request Body:**
```json
{
  "organization": "acme-corp",
  "enterprise_slug": "acme-enterprise",
  "workers": 5
}
```

**Response 202 Accepted:**
```json
{
  "status": "started",
  "organization": "acme-corp",
  "type": "organization",
  "message": "Discovery started successfully"
}
```

### GET /api/v1/discovery/status

Get the status of the current or last discovery operation.

**Response 200 OK:**
```json
{
  "status": "running",
  "repositories_found": 127,
  "completed_at": null
}
```

**Status Values:** `idle`, `running`, `completed`, `failed`

---

## Repositories

### GET /api/v1/repositories

List all repositories with optional filtering.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by migration status (comma-separated) |
| `batch_id` | int | Filter by batch ID |
| `organization` | string | Filter by organization (comma-separated) |
| `project` | string | Filter by ADO project (comma-separated) |
| `search` | string | Search by repository name |
| `sort_by` | string | Sort field: name, size, org, updated |
| `available_for_batch` | bool | Filter repositories available for batch |
| `has_lfs` | bool | Filter by LFS usage |
| `has_submodules` | bool | Filter by submodules |
| `has_large_files` | bool | Filter by large files (>100MB) |
| `has_actions` | bool | Filter by GitHub Actions |
| `has_wiki` | bool | Filter by wiki |
| `has_pages` | bool | Filter by GitHub Pages |
| `has_discussions` | bool | Filter by discussions |
| `has_projects` | bool | Filter by projects |
| `has_packages` | bool | Filter by packages |
| `has_environments` | bool | Filter by environments |
| `has_secrets` | bool | Filter by secrets |
| `has_variables` | bool | Filter by variables |
| `has_webhooks` | bool | Filter by webhooks |
| `has_branch_protections` | bool | Filter by branch protections |
| `has_rulesets` | bool | Filter by rulesets |
| `has_code_scanning` | bool | Filter by code scanning |
| `has_dependabot` | bool | Filter by Dependabot |
| `has_secret_scanning` | bool | Filter by secret scanning |
| `has_codeowners` | bool | Filter by CODEOWNERS |
| `is_archived` | bool | Filter by archived status |
| `is_fork` | bool | Filter by fork status |
| `visibility` | string | Filter by visibility: public, private, internal |
| `min_size` | int | Minimum size in bytes |
| `max_size` | int | Maximum size in bytes |
| `size_category` | string | Size category (comma-separated) |
| `complexity` | string | Complexity level (comma-separated) |
| `limit` | int | Pagination limit |
| `offset` | int | Pagination offset |

**Response 200 OK:**
```json
{
  "repositories": [
    {
      "id": 1,
      "full_name": "acme-corp/api-gateway",
      "status": "ready",
      "total_size": 15234000,
      "has_lfs": true,
      "has_actions": true,
      "batch_id": 1,
      "complexity_score": 5
    }
  ],
  "total": 150
}
```

### POST /api/v1/repositories/batch-update

Batch update status for multiple repositories.

**Request Body:**
```json
{
  "repository_ids": [1, 2, 3],
  "action": "mark_migrated",
  "reason": "Migrated manually"
}
```

**Actions:** `mark_migrated`, `mark_wont_migrate`, `unmark_wont_migrate`, `rollback`

**Response 200 OK:**
```json
{
  "updated_count": 3,
  "failed_count": 0,
  "errors": []
}
```

### GET /api/v1/repositories/{fullName}

Get detailed repository information including migration history.

**Response 200 OK:**
```json
{
  "repository": {
    "id": 1,
    "full_name": "acme-corp/api-gateway",
    "status": "migrating"
  },
  "history": [
    {
      "id": 1,
      "status": "migrating",
      "phase": "migration",
      "started_at": "2024-01-15T11:00:00Z"
    }
  ]
}
```

### PATCH /api/v1/repositories/{fullName}

Update repository metadata.

**Request Body:**
```json
{
  "batch_id": 2,
  "priority": 10,
  "destination_full_name": "new-org/api-gateway"
}
```

### POST /api/v1/repositories/{fullName}/rediscover

Re-run discovery and profiling for a specific repository.

### POST /api/v1/repositories/{fullName}/unlock

Unlock a repository that was locked during migration.

### POST /api/v1/repositories/{fullName}/rollback

Rollback a completed migration.

### POST /api/v1/repositories/{fullName}/mark-wont-migrate

Mark or unmark a repository as "won't migrate".

### POST /api/v1/repositories/{fullName}/mark-remediated

Mark a repository as remediated and trigger re-validation.

---

## Organizations & Projects

### GET /api/v1/organizations

Get organization statistics.

**Response 200 OK:**
```json
[
  {
    "organization": "acme-corp",
    "repository_count": 150,
    "total_size": 50000000000
  }
]
```

### GET /api/v1/organizations/list

Get simple list of organization names for filters.

### GET /api/v1/projects

Get ADO projects with statistics.

---

## Dashboard

### GET /api/v1/dashboard/action-items

Get action items and alerts for the dashboard.

**Response 200 OK:**
```json
{
  "action_items": [
    {
      "type": "failed_migration",
      "count": 5,
      "severity": "high"
    }
  ]
}
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
    "status": "completed",
    "repository_count": 5
  }
]
```

**Status Values:** `ready`, `in_progress`, `complete`, `failed`

### POST /api/v1/batches

Create a new migration batch.

**Request Body:**
```json
{
  "name": "Wave 2 - Backend Services",
  "description": "Second wave containing backend microservices"
}
```

### GET /api/v1/batches/{id}

Get batch details including repositories.

### PATCH /api/v1/batches/{id}

Update batch metadata.

### DELETE /api/v1/batches/{id}

Delete a batch (not allowed for in-progress batches).

### POST /api/v1/batches/{id}/start

Start migration for all repositories in a batch.

### POST /api/v1/batches/{id}/dry-run

Start dry run for all repositories in a batch.

**Request Body:**
```json
{
  "only_pending": true
}
```

### POST /api/v1/batches/{id}/repositories

Add repositories to a batch.

**Request Body:**
```json
{
  "repository_ids": [10, 11, 12]
}
```

### DELETE /api/v1/batches/{id}/repositories

Remove repositories from a batch.

### POST /api/v1/batches/{id}/retry

Retry failed migrations in a batch.

---

## Migrations

### POST /api/v1/migrations/start

Start migration for repositories.

**Request Body:**
```json
{
  "repository_ids": [1, 2, 3],
  "dry_run": false,
  "priority": 5
}
```

Or by names (self-service):
```json
{
  "full_names": ["acme-corp/api-gateway"],
  "dry_run": false
}
```

### GET /api/v1/migrations/{id}

Get migration status.

**Response 200 OK:**
```json
{
  "repository_id": 1,
  "full_name": "acme-corp/api-gateway",
  "status": "migrating",
  "destination_url": null,
  "can_retry": false
}
```

### GET /api/v1/migrations/{id}/history

Get complete migration history.

### GET /api/v1/migrations/{id}/logs

Get migration logs with optional filtering.

**Query Parameters:**
- `level` - Filter by log level: DEBUG, INFO, WARN, ERROR
- `phase` - Filter by migration phase
- `limit` - Max logs to return (default: 500)
- `offset` - Pagination offset

### GET /api/v1/migrations/history

List completed migrations.

### GET /api/v1/migrations/history/export

Export migration history.

**Query Parameters:**
- `format` (required) - Export format: `csv` or `json`

### POST /api/v1/self-service/migrate

Self-service migration endpoint.

**Request Body:**
```json
{
  "repositories": ["acme-corp/api-gateway", "acme-corp/web-app"],
  "mappings": {
    "acme-corp/api-gateway": "new-org/api-gateway"
  },
  "dry_run": true
}
```

---

## Analytics

### GET /api/v1/analytics/summary

Get analytics summary.

**Query Parameters:**
- `organization` - Filter by organization
- `batch_id` - Filter by batch ID

### GET /api/v1/analytics/progress

Get migration progress statistics.

### GET /api/v1/analytics/executive-report

Get comprehensive executive-level report.

**Response 200 OK:**
```json
{
  "executive_summary": {
    "total_repositories": 500,
    "completion_percentage": 68.5,
    "success_rate": 95.8,
    "estimated_completion_date": "2025-02-15"
  },
  "velocity_metrics": {
    "repos_per_day": 16.3,
    "repos_per_week": 114
  },
  "organization_progress": [...],
  "risk_analysis": {...}
}
```

### GET /api/v1/analytics/executive-report/export

Export executive report in CSV or JSON format.

### GET /api/v1/analytics/detailed-discovery-report/export

Export detailed discovery report with all repository data.

**Query Parameters:**
- `format` (required) - Export format: `csv` or `json`
- `organization` - Filter by organization
- `project` - Filter by ADO project
- `batch_id` - Filter by batch ID

---

## Azure DevOps

Endpoints specific to Azure DevOps source migrations.

### POST /api/v1/ado/discover

Start ADO repository discovery.

**Request Body:**
```json
{
  "organization": "my-ado-org",
  "project": "MyProject",
  "workers": 5
}
```

### GET /api/v1/ado/discovery/status

Get ADO discovery status.

### GET /api/v1/ado/projects

List all ADO projects.

### GET /api/v1/ado/projects/{organization}/{project}

Get details for a specific ADO project.

---

## Error Handling

### Error Response Format

```json
{
  "error": "Detailed error message"
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | OK - Successful GET, PATCH requests |
| 201 | Created - Successful POST creating new resource |
| 202 | Accepted - Async operation started |
| 400 | Bad Request - Invalid request |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource not found |
| 500 | Internal Server Error |

---

## Rate Limiting

The server implements intelligent rate limiting for GitHub API calls:

- **Primary Rate Limit:** 5,000 requests/hour (authenticated)
- **Auto-Wait:** Server automatically waits when limits are exhausted
- **Retry Logic:** Exponential backoff for failed requests

**Rate Limit Headers:**
```
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4850
X-RateLimit-Reset: 1705324800
```

---

## Code Examples

### JavaScript/TypeScript

```typescript
const API_BASE = 'http://localhost:8080/api/v1';

async function startDiscovery(org: string) {
  const response = await fetch(`${API_BASE}/discovery/start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ organization: org })
  });
  return response.json();
}

async function getRepositories(filters?: Record<string, any>) {
  const params = new URLSearchParams(filters);
  const response = await fetch(`${API_BASE}/repositories?${params}`);
  return response.json();
}
```

### Python

```python
import requests

API_BASE = 'http://localhost:8080/api/v1'

def start_discovery(org):
    response = requests.post(
        f'{API_BASE}/discovery/start',
        json={'organization': org}
    )
    return response.json()

def get_repositories(filters=None):
    response = requests.get(
        f'{API_BASE}/repositories',
        params=filters or {}
    )
    return response.json()
```

### Go

```go
const apiBase = "http://localhost:8080/api/v1"

func startDiscovery(org string) error {
    body, _ := json.Marshal(map[string]string{"organization": org})
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
```

---

## Support

For detailed information, see:
- [README.md](../README.md) - Project overview and quickstart
- [Deployment Guide](./deployment/) - Docker, Azure, Kubernetes deployment
- [Operations Guide](./OPERATIONS.md) - Authentication, workflows, troubleshooting
- [Contributing Guide](./CONTRIBUTING.md) - Development and contributing

---

**API Version:** 1.1.0  
**Last Updated:** December 2025  
**Status:** Production Ready
