# GitHub Migrator - Implementation Guide

This guide provides deep technical insights into the architecture, implementation details, and internal workings of the GitHub Migrator.

## Table of Contents

- [System Architecture](#system-architecture)
- [Core Components](#core-components)
- [Database Schema](#database-schema)
- [API Implementation](#api-implementation)
- [Migration Workflow Internals](#migration-workflow-internals)
- [Extension Points](#extension-points)
- [Performance Considerations](#performance-considerations)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

---

## System Architecture

### Overview

The GitHub Migrator is built as a monolithic Go application with an embedded React frontend. It uses a polling-based architecture for migration status updates and supports both SQLite (development) and PostgreSQL (production).

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                         │
│                    Static files served by backend                │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTP/REST API
┌──────────────────────────▼──────────────────────────────────────┐
│                     Backend (Go HTTP Server)                     │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    API Layer (Handlers)                     │ │
│  │  - REST endpoints - Request validation - Response format   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Discovery   │  │  Migration   │  │      Batch           │  │
│  │  Engine      │  │  Executor    │  │   Orchestrator       │  │
│  │              │  │              │  │                      │  │
│  │ - Collector  │  │ - State      │  │ - Batch Manager      │  │
│  │ - Analyzer   │  │   Machine    │  │ - Scheduler          │  │
│  │ - Profiler   │  │ - Validator  │  │ - Priority Queue     │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Analytics   │  │  Worker      │  │     Storage          │  │
│  │              │  │  Pool        │  │   (Repository        │  │
│  │ - Metrics    │  │              │  │    Pattern)          │  │
│  │ - Reporting  │  │ - Scheduler  │  │                      │  │
│  │ - Trends     │  │ - Poller     │  │ - SQLite/PostgreSQL  │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                GitHub Dual Client                           │ │
│  │                                                             │ │
│  │  ┌──────────────────┐       ┌──────────────────────┐      │ │
│  │  │  PAT Client      │       │  GitHub App Client   │      │ │
│  │  │ (Required for    │       │  (Optional for rate  │      │ │
│  │  │  migrations)     │       │   limit benefits)    │      │ │
│  │  └──────────────────┘       └──────────────────────┘      │ │
│  │                                                             │ │
│  │  - REST API (go-github)                                    │ │
│  │  - GraphQL API (githubv4)                                  │ │
│  │  - Rate Limiting                                           │ │
│  │  - Retry Logic                                             │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTPS
            ┌──────────────┴──────────────┐
            │                             │
┌───────────▼──────────┐      ┌───────────▼──────────┐
│  GitHub Enterprise   │      │  GitHub Enterprise   │
│      Server          │      │       Cloud          │
│     (Source)         │      │    (Destination)     │
└──────────────────────┘      └──────────────────────┘
```

### Technology Choices

#### Backend: Go 1.21+

**Rationale:**
- Excellent concurrency support for parallel operations
- Strong standard library for HTTP servers and APIs
- Fast compilation and execution
- Easy cross-compilation for different platforms
- Great GitHub API client libraries

#### Database: SQLite (dev) / PostgreSQL (prod)

**Rationale:**
- **SQLite**: Zero configuration, perfect for development and small deployments
- **PostgreSQL**: Proven production-ready database with excellent performance
- Abstraction layer allows switching between them without code changes

#### Frontend: React 18 + TypeScript

**Rationale:**
- Modern, component-based UI development
- Type safety with TypeScript
- Rich ecosystem of UI libraries
- Fast development with Vite

#### Authentication Strategy: Dual Client (PAT + GitHub App)

**Rationale:**
- GitHub Migrations API requires PAT (Personal Access Token)
- GitHub Apps offer higher rate limits for other operations
- Dual client automatically uses best authentication for each operation

---

## Core Components

### API Layer (`internal/api`)

#### Handler Structure

```go
type Handler struct {
    db               *storage.Database
    logger           *slog.Logger
    sourceDualClient *github.DualClient
    destDualClient   *github.DualClient
    collector        *discovery.Collector
}
```

**Key Design Decisions:**

1. **Single Handler struct**: All HTTP handlers are methods on a single struct, providing centralized access to dependencies
2. **Dependency Injection**: All dependencies injected via constructor
3. **Structured Logging**: Uses `slog` for structured, leveled logging
4. **Dual Client Pattern**: Separate clients for source and destination

#### Request/Response Flow

```
HTTP Request
    ↓
Router (internal/api/server.go)
    ↓
Middleware Chain
    - CORS
    - Logging
    - (Future: Authentication, Rate Limiting)
    ↓
Handler Method
    - Parse Request
    - Validate Input
    - Call Business Logic
    - Format Response
    ↓
HTTP Response (JSON)
```

#### Error Handling Pattern

```go
func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}
```

All handlers use these helper methods for consistent error and success responses.

#### Middleware Implementation

```go
func (s *Server) setupRoutes() {
    // Apply CORS middleware to all routes
    s.router.Use(corsMiddleware)
    
    // Health check (no auth required)
    s.router.HandleFunc("/health", s.handler.Health).Methods("GET")
    
    // API routes (could add auth middleware here in future)
    api := s.router.PathPrefix("/api/v1").Subrouter()
    // ... route definitions
}
```

**Current Middleware:**
- CORS: Allows cross-origin requests from frontend dev server

**Planned Middleware:**
- Authentication: API key or OAuth validation
- Rate Limiting: Request throttling
- Request ID: Unique ID for tracing

---

### Discovery System (`internal/discovery`)

#### Component Interaction

```
Collector
    ├── Fetches repositories from source
    ├── Calls Analyzer for basic metadata
    └── Calls Profiler for detailed analysis
        ↓
Analyzer
    ├── Extracts GitHub features
    ├── Identifies security settings
    └── Counts collaborators, issues, PRs
        ↓
Profiler
    ├── Runs git-sizer for repository metrics
    ├── Analyzes LFS usage
    ├── Detects large files
    └── Calculates complexity scores
```

#### Collector Implementation

```go
type Collector struct {
    client   github.ClientInterface
    db       *storage.Database
    logger   *slog.Logger
    provider source.Provider
    workers  int  // Parallel worker count
}

func (c *Collector) DiscoverOrganization(ctx context.Context, org string) error {
    // 1. Fetch all repositories from organization
    repos := c.fetchRepositoriesFromOrg(ctx, org)
    
    // 2. Process repositories in parallel with worker pool
    jobs := make(chan *github.Repository, len(repos))
    results := make(chan error, len(repos))
    
    // Start workers
    for i := 0; i < c.workers; i++ {
        go c.worker(ctx, jobs, results)
    }
    
    // Send jobs
    for _, repo := range repos {
        jobs <- repo
    }
    close(jobs)
    
    // Collect results
    for i := 0; i < len(repos); i++ {
        <-results
    }
    
    return nil
}
```

**Worker Pattern:**
- Configurable number of parallel workers (default: 10)
- Each worker processes repositories independently
- Rate limiting handled by GitHub client layer

#### Analyzer Workflow

```go
func (a *Analyzer) Analyze(ctx context.Context, repo *models.Repository) error {
    // Fetch detailed repository data
    repoDetails := a.fetchRepositoryDetails(ctx, repo)
    
    // Analyze GitHub features
    repo.HasActions = a.detectActions(ctx, repo)
    repo.HasWiki = repoDetails.HasWiki
    repo.HasPages = repoDetails.HasPages
    
    // Analyze security features
    repo.BranchProtections = a.countBranchProtections(ctx, repo)
    repo.HasRulesets = a.detectRulesets(ctx, repo)
    
    // Count collaborators, issues, PRs
    repo.CollaboratorCount = a.countCollaborators(ctx, repo)
    repo.IssueCount = a.countIssues(ctx, repo)
    
    return nil
}
```

#### Profiler Details

```go
func (p *Profiler) Profile(ctx context.Context, repo *models.Repository) error {
    // Clone repository (bare clone for analysis)
    tempDir := p.cloneRepository(ctx, repo)
    defer os.RemoveAll(tempDir)
    
    // Run git-sizer
    sizerOutput := p.runGitSizer(ctx, tempDir)
    
    // Parse git-sizer output
    repo.TotalSize = sizerOutput.TotalSize
    repo.LargestFile = sizerOutput.LargestFile
    repo.LargestFileSize = sizerOutput.LargestFileSize
    repo.HasLFS = sizerOutput.HasLFS
    repo.HasSubmodules = sizerOutput.HasSubmodules
    
    // Detect large files (>100MB)
    repo.HasLargeFiles = p.detectLargeFiles(tempDir)
    repo.LargeFileCount = p.countLargeFiles(tempDir)
    
    // Calculate complexity score
    repo.ComplexityScore = p.calculateComplexity(repo)
    
    return nil
}
```

**git-sizer Integration:**
- Embedded binaries for portability (Darwin/Linux/Windows, AMD64/ARM64)
- Executed as subprocess with JSON output parsing
- Timeout protection (5 minutes default)
- Error handling for very large repositories

---

### Migration Engine (`internal/migration`)

#### Executor State Machine

The migration executor implements a sequential state machine with the following phases:

```
┌─────────────────────────────────────────────────────────────────┐
│                      Migration State Machine                     │
└─────────────────────────────────────────────────────────────────┘

1. PRE_MIGRATION
   ├── Run pre-migration discovery (production only)
   ├── Validate repository can be migrated
   ├── Check destination organization exists
   └── Verify no existing repository at destination
   
2. ARCHIVE_GENERATING
   ├── Call GitHub Migrations API (start_for_org)
   ├── Create migration archive on source system
   ├── Lock repository if production migration
   └── Poll until archive ready
   
3. QUEUED_FOR_MIGRATION
   ├── Upload archive URL to destination
   ├── Call start_import API
   └── Receive migration ID
   
4. MIGRATING_CONTENT
   ├── Poll migration status (GraphQL)
   ├── States: QUEUED → IN_PROGRESS → PENDING_VALIDATION
   └── Wait for SUCCEEDED or FAILED
   
5. MIGRATION_COMPLETE
   ├── Archive import completed
   └── Repository available at destination
   
6. POST_MIGRATION (configurable)
   ├── Run post-migration validation
   ├── Compare source vs destination
   ├── Verify commit count, branches, tags
   └── Check LFS objects, releases, etc.
   
7. COMPLETE
   ├── Unlock source repository
   ├── Update status and timestamps
   └── Mark migration complete

┌─────────────────────────────────────────────────────────────────┐
│                     Dry Run Differences                          │
└─────────────────────────────────────────────────────────────────┘

Dry Run Mode (lock_repositories: false):
- Skips pre-migration discovery
- Archives created without locking
- Full migration executed
- Repository NOT locked during process
- Allows testing without affecting source
- Sets dry_run_complete status
```

#### Migration Phase Implementation

```go
func (e *Executor) ExecuteMigration(ctx context.Context, repo *models.Repository, dryRun bool) error {
    // Create migration history record
    historyID := e.createMigrationHistory(ctx, repo, dryRun)
    
    // Phase 1: Pre-migration validation
    if err := e.validatePreMigration(ctx, repo); err != nil {
        return err
    }
    
    // Phase 2: Generate archives on source (GHES)
    lockRepos := !dryRun
    archiveID, err := e.generateArchivesOnGHES(ctx, repo, lockRepos)
    if err != nil {
        return err
    }
    
    // Phase 3: Wait for archive generation
    archiveURL, err := e.pollArchiveStatus(ctx, repo, archiveID)
    if err != nil {
        return err
    }
    
    // Phase 4: Import to destination (GitHub.com or GHEC)
    migrationID, err := e.importToDestination(ctx, repo, archiveURL)
    if err != nil {
        return err
    }
    
    // Phase 5: Poll for migration completion
    if err := e.pollMigrationStatus(ctx, repo, historyID, migrationID); err != nil {
        return err
    }
    
    // Phase 6: Post-migration validation (if configured)
    if e.shouldRunPostMigration(dryRun) {
        e.validatePostMigration(ctx, repo)
    }
    
    // Phase 7: Mark complete and unlock
    e.finalizeM igration(ctx, repo, dryRun)
    
    return nil
}
```

#### GitHub API Interactions

**Source System (GHES) - REST API:**

```go
// Start migration (create archive)
POST /orgs/{org}/migrations
{
  "repositories": ["repo-name"],
  "lock_repositories": false,  // false for dry run
  "exclude_attachments": false
}

Response: { "id": 12345, "state": "pending" }

// Poll archive status
GET /orgs/{org}/migrations/{migration_id}

Response: {
  "id": 12345,
  "state": "exported",  // pending → exporting → exported
  "archive_url": "https://..."
}
```

**Destination System (GHEC) - REST + GraphQL:**

```go
// Start import (REST)
PUT /repos/{owner}/{repo}/import
{
  "vcs": "git",
  "vcs_url": "archive_url_from_source"
}

Response: { "status": "importing" }

// Poll import status (GraphQL for better details)
query {
  node(id: "migration_id") {
    ... on Migration {
      state  // QUEUED, IN_PROGRESS, SUCCEEDED, FAILED
      failureReason
      repositoryName
    }
  }
}
```

#### Error Recovery

```go
func (e *Executor) handleMigrationFailure(ctx context.Context, repo *models.Repository, err error) {
    // Log detailed error
    e.logger.Error("Migration failed", "repo", repo.FullName, "error", err)
    
    // Update repository status
    repo.Status = string(models.StatusMigrationFailed)
    
    // Unlock repository if it was locked
    if repo.IsSourceLocked && repo.SourceMigrationID != nil {
        e.unlockSourceRepository(ctx, repo)
        repo.IsSourceLocked = false
    }
    
    // Save to database
    e.storage.UpdateRepository(ctx, repo)
}
```

**Retry Strategy:**
- Manual retry via API or UI
- Automatic retry for transient failures (planned)
- Rollback support for completed migrations

---

### Batch Management (`internal/batch`)

#### Orchestrator Design

```go
type Orchestrator struct {
    db               *storage.Database
    executor         *migration.Executor
    logger           *slog.Logger
    maxParallelJobs  int
}

func (o *Orchestrator) StartBatch(ctx context.Context, batchID int64, dryRun bool) error {
    // 1. Get batch and repositories
    batch := o.db.GetBatch(ctx, batchID)
    repos := o.db.GetRepositoriesByBatch(ctx, batchID)
    
    // 2. Sort by priority (higher first)
    sort.Slice(repos, func(i, j int) bool {
        return repos[i].Priority > repos[j].Priority
    })
    
    // 3. Execute migrations with concurrency control
    semaphore := make(chan struct{}, o.maxParallelJobs)
    var wg sync.WaitGroup
    
    for _, repo := range repos {
        wg.Add(1)
        go func(r *models.Repository) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            o.executor.ExecuteMigration(ctx, r, dryRun)
        }(repo)
    }
    
    wg.Wait()
    
    // 4. Update batch status
    o.updateBatchStatus(ctx, batch)
    
    return nil
}
```

#### Scheduler Implementation

```go
type Scheduler struct {
    db     *storage.Database
    logger *slog.Logger
}

func (s *Scheduler) GetNextMigrations(ctx context.Context, limit int) []*models.Repository {
    // Priority: pilot > high priority > normal
    // Status: queued_for_migration, ordered by priority DESC, created_at ASC
    return s.db.GetRepositoriesForMigration(ctx, limit)
}
```

#### Status Tracking

```
Batch Statuses:
- ready: Created, no migrations started
- in_progress: At least one migration running
- complete: All migrations completed successfully
- failed: One or more migrations failed
- partial: Mix of completed and failed
```

---

### Storage Layer (`internal/storage`)

#### Database Abstraction

```go
type Database struct {
    db     *sql.DB
    dbType string  // "sqlite" or "postgresql"
    logger *slog.Logger
}

func NewDatabase(dbType, dsn string, logger *slog.Logger) (*Database, error) {
    var db *sql.DB
    var err error
    
    switch dbType {
    case "sqlite":
        db, err = sql.Open("sqlite3", dsn)
    case "postgresql":
        db, err = sql.Open("postgres", dsn)
    default:
        return nil, fmt.Errorf("unsupported database type: %s", dbType)
    }
    
    // Run migrations
    if err := runMigrations(db, dbType); err != nil {
        return nil, err
    }
    
    return &Database{db: db, dbType: dbType, logger: logger}, nil
}
```

#### Repository Pattern

All database operations use the repository pattern for consistency:

```go
// Create
func (d *Database) CreateRepository(ctx context.Context, repo *models.Repository) error

// Read
func (d *Database) GetRepository(ctx context.Context, id int64) (*models.Repository, error)
func (d *Database) GetRepositoryByFullName(ctx context.Context, fullName string) (*models.Repository, error)
func (d *Database) GetRepositories(ctx context.Context, filters RepositoryFilters) ([]*models.Repository, error)

// Update
func (d *Database) UpdateRepository(ctx context.Context, repo *models.Repository) error

// Delete
func (d *Database) DeleteRepository(ctx context.Context, id int64) error
```

#### Migration Management

**Schema Migrations:**

```
internal/storage/migrations/
├── 001_initial_schema.sql
├── 002_add_discovery_fields.sql
├── 003_add_archived_field.sql
├── 004_add_lock_tracking.sql
├── 005_add_rollback_tracking.sql
├── 006_add_validation_fields.sql
├── 007_add_is_fork_field.sql
├── 008_add_has_packages_field.sql
├── 009_add_advanced_features.sql
├── 010_add_rulesets.sql
├── 011_add_operation_timestamps.sql
├── 012_backfill_operation_timestamps.sql
└── 013_add_unique_batch_name.sql
```

**Migration Runner:**

```go
func runMigrations(db *sql.DB, dbType string) error {
    // Create schema_migrations table
    createSchemaTable(db)
    
    // Get current version
    currentVersion := getCurrentVersion(db)
    
    // Read migration files
    migrations := loadMigrationFiles()
    
    // Apply pending migrations
    for _, migration := range migrations {
        if migration.Version > currentVersion {
            executeMigration(db, migration)
            updateSchemaVersion(db, migration.Version)
        }
    }
    
    return nil
}
```

#### Query Optimization

**Indexes:**

```sql
-- Frequently queried fields
CREATE INDEX idx_repositories_status ON repositories(status);
CREATE INDEX idx_repositories_batch_id ON repositories(batch_id);
CREATE INDEX idx_repositories_full_name ON repositories(full_name);

-- Migration history queries
CREATE INDEX idx_migration_history_repo ON migration_history(repository_id);
CREATE INDEX idx_migration_history_status ON migration_history(status);

-- Log queries
CREATE INDEX idx_migration_logs_repo ON migration_logs(repository_id);
CREATE INDEX idx_migration_logs_timestamp ON migration_logs(timestamp);
```

**Connection Pooling:**

```go
func configureConnectionPool(db *sql.DB, dbType string) {
    if dbType == "postgresql" {
        db.SetMaxOpenConns(50)
        db.SetMaxIdleConns(10)
        db.SetConnMaxLifetime(10 * time.Minute)
    } else {
        // SQLite doesn't support multiple connections
        db.SetMaxOpenConns(1)
    }
}
```

---

### GitHub Integration (`internal/github`)

#### Dual Client Design

```go
type DualClient struct {
    patClient *Client    // Personal Access Token client
    appClient *Client    // GitHub App client (optional)
    logger    *slog.Logger
}

// APIClient returns the best client for non-migration operations
func (dc *DualClient) APIClient() *Client {
    if dc.appClient != nil {
        return dc.appClient  // Prefer App for higher rate limits
    }
    return dc.patClient
}

// MigrationClient returns the PAT client (required for migrations)
func (dc *DualClient) MigrationClient() *Client {
    return dc.patClient  // Migrations API requires PAT
}
```

**Why Dual Client?**
- GitHub Migrations API requires Personal Access Token (PAT)
- GitHub Apps have higher rate limits (5,000 vs 5,000 per installation)
- Using App for discovery/profiling preserves PAT rate limit for migrations

#### Rate Limiting Strategy

```go
type RateLimiter struct {
    client      *github.Client
    logger      *slog.Logger
    waitOnLimit bool  // Auto-wait when exhausted
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    // Get current rate limit
    limits, _, err := rl.client.RateLimits(ctx)
    if err != nil {
        return err
    }
    
    core := limits.Core
    
    // Check if exhausted
    if core.Remaining == 0 {
        resetTime := core.Reset.Time
        waitDuration := time.Until(resetTime)
        
        if rl.waitOnLimit {
            rl.logger.Info("Rate limit exhausted, waiting",
                "reset_at", resetTime,
                "wait_duration", waitDuration)
            time.Sleep(waitDuration)
            return nil
        }
        
        return fmt.Errorf("rate limit exceeded, resets at %s", resetTime)
    }
    
    return nil
}
```

**Rate Limit Headers:**
```
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4850
X-RateLimit-Reset: 1705324800
```

#### Retry Logic

```go
type RetryConfig struct {
    MaxAttempts    int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
}

func (c *Client) retryRequest(ctx context.Context, fn func() error) error {
    var lastErr error
    backoff := c.retryConfig.InitialBackoff
    
    for attempt := 0; attempt < c.retryConfig.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        lastErr = err
        
        // Don't retry on non-transient errors
        if !isRetryable(err) {
            return err
        }
        
        // Wait with exponential backoff
        if attempt < c.retryConfig.MaxAttempts-1 {
            time.Sleep(backoff)
            backoff = time.Duration(float64(backoff) * c.retryConfig.Multiplier)
            if backoff > c.retryConfig.MaxBackoff {
                backoff = c.retryConfig.MaxBackoff
            }
        }
    }
    
    return fmt.Errorf("max retry attempts exceeded: %w", lastErr)
}

func isRetryable(err error) bool {
    // Retry on: 5xx errors, rate limits, network errors
    // Don't retry: 4xx errors (except 429), validation errors
    if strings.Contains(err.Error(), "rate limit") {
        return true
    }
    if strings.Contains(err.Error(), "50") {  // 5xx errors
        return true
    }
    return false
}
```

---

## Database Schema

### Tables Overview

```sql
repositories          -- Main repository data
migration_history     -- Migration event log
migration_logs        -- Detailed operation logs
batches              -- Migration batches
```

### repositories Table

Stores all discovered repositories and their characteristics.

```sql
CREATE TABLE repositories (
    -- Identity
    id INTEGER PRIMARY KEY,
    full_name TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    source_url TEXT NOT NULL,
    
    -- Git Properties
    total_size INTEGER,
    largest_file TEXT,
    largest_file_size INTEGER,
    largest_commit TEXT,
    largest_commit_size INTEGER,
    has_lfs BOOLEAN DEFAULT FALSE,
    has_submodules BOOLEAN DEFAULT FALSE,
    has_large_files BOOLEAN DEFAULT FALSE,
    large_file_count INTEGER DEFAULT 0,
    default_branch TEXT,
    branch_count INTEGER DEFAULT 0,
    commit_count INTEGER DEFAULT 0,
    last_commit_sha TEXT,
    last_commit_date DATETIME,
    
    -- GitHub Features
    is_archived BOOLEAN DEFAULT FALSE,
    is_fork BOOLEAN DEFAULT FALSE,
    has_wiki BOOLEAN DEFAULT FALSE,
    has_pages BOOLEAN DEFAULT FALSE,
    has_discussions BOOLEAN DEFAULT FALSE,
    has_actions BOOLEAN DEFAULT FALSE,
    has_projects BOOLEAN DEFAULT FALSE,
    has_packages BOOLEAN DEFAULT FALSE,
    branch_protections INTEGER DEFAULT 0,
    has_rulesets BOOLEAN DEFAULT FALSE,
    environment_count INTEGER DEFAULT 0,
    secret_count INTEGER DEFAULT 0,
    variable_count INTEGER DEFAULT 0,
    webhook_count INTEGER DEFAULT 0,
    
    -- Security & Compliance
    has_code_scanning BOOLEAN DEFAULT FALSE,
    has_dependabot BOOLEAN DEFAULT FALSE,
    has_secret_scanning BOOLEAN DEFAULT FALSE,
    has_codeowners BOOLEAN DEFAULT FALSE,
    
    -- Repository Settings
    visibility TEXT,
    workflow_count INTEGER DEFAULT 0,
    
    -- Infrastructure & Access
    has_self_hosted_runners BOOLEAN DEFAULT FALSE,
    collaborator_count INTEGER DEFAULT 0,
    installed_apps_count INTEGER DEFAULT 0,
    
    -- Releases
    release_count INTEGER DEFAULT 0,
    has_release_assets BOOLEAN DEFAULT FALSE,
    
    -- Contributors
    contributor_count INTEGER DEFAULT 0,
    top_contributors TEXT,  -- JSON array
    
    -- Verification Data
    issue_count INTEGER DEFAULT 0,
    pull_request_count INTEGER DEFAULT 0,
    tag_count INTEGER DEFAULT 0,
    open_issue_count INTEGER DEFAULT 0,
    open_pr_count INTEGER DEFAULT 0,
    
    -- Status Tracking
    status TEXT NOT NULL,
    batch_id INTEGER,
    priority INTEGER DEFAULT 0,
    
    -- Migration Details
    destination_url TEXT,
    destination_full_name TEXT,
    
    -- Lock Tracking
    source_migration_id INTEGER,
    is_source_locked BOOLEAN DEFAULT FALSE,
    
    -- Validation Tracking
    validation_status TEXT,
    validation_details TEXT,  -- JSON
    destination_data TEXT,    -- JSON
    
    -- Timestamps
    discovered_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    migrated_at DATETIME,
    last_discovery_at DATETIME,
    last_dry_run_at DATETIME,
    
    FOREIGN KEY (batch_id) REFERENCES batches(id)
);
```

### migration_history Table

Tracks each migration attempt and its phases.

```sql
CREATE TABLE migration_history (
    id INTEGER PRIMARY KEY,
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
```

**Usage:**
- One record per migration attempt
- Status updated as migration progresses
- Duration calculated on completion

### migration_logs Table

Detailed operation logs for troubleshooting.

```sql
CREATE TABLE migration_logs (
    id INTEGER PRIMARY KEY,
    repository_id INTEGER NOT NULL,
    history_id INTEGER,
    level TEXT NOT NULL,        -- DEBUG, INFO, WARN, ERROR
    phase TEXT NOT NULL,
    operation TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT,               -- JSON or text
    timestamp DATETIME NOT NULL,
    
    FOREIGN KEY (repository_id) REFERENCES repositories(id),
    FOREIGN KEY (history_id) REFERENCES migration_history(id)
);
```

**Usage:**
- Granular logging of migration operations
- Queryable for debugging
- Filtered by level, phase, or time range

### batches Table

Organizes repositories into migration groups.

```sql
CREATE TABLE batches (
    id INTEGER PRIMARY KEY,
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
```

**Relationships:**
- One-to-many: batch → repositories (via batch_id foreign key)
- Status calculated from repository statuses

---

## API Implementation Details

### Endpoint Patterns

#### List Endpoints

Pattern: `GET /api/v1/{resource}`

```go
func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
    // 1. Parse query parameters
    filters := parseFilters(r.URL.Query())
    
    // 2. Validate parameters
    if err := validateFilters(filters); err != nil {
        h.sendError(w, http.StatusBadRequest, err.Error())
        return
    }
    
    // 3. Query database
    repos, err := h.db.GetRepositories(r.Context(), filters)
    if err != nil {
        h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
        return
    }
    
    // 4. Return response
    h.sendJSON(w, http.StatusOK, repos)
}
```

#### Detail Endpoints

Pattern: `GET /api/v1/{resource}/{id}`

```go
func (h *Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
    // 1. Extract path parameter
    fullName := mux.Vars(r)["fullName"]
    
    // 2. Decode if URL-encoded
    fullName, _ = url.QueryUnescape(fullName)
    
    // 3. Query database
    repo, err := h.db.GetRepositoryByFullName(r.Context(), fullName)
    if err != nil {
        h.sendError(w, http.StatusNotFound, "Repository not found")
        return
    }
    
    // 4. Fetch related data (history, logs)
    history, _ := h.db.GetMigrationHistory(r.Context(), repo.ID)
    
    // 5. Return combined response
    h.sendJSON(w, http.StatusOK, map[string]interface{}{
        "repository": repo,
        "history":    history,
    })
}
```

#### Action Endpoints

Pattern: `POST /api/v1/{resource}/{action}`

```go
func (h *Handler) StartMigration(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request body
    var req struct {
        RepositoryIDs []int64  `json:"repository_ids"`
        FullNames     []string `json:"full_names"`
        DryRun        bool     `json:"dry_run"`
        Priority      int      `json:"priority"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    // 2. Validate request
    if len(req.RepositoryIDs) == 0 && len(req.FullNames) == 0 {
        h.sendError(w, http.StatusBadRequest, "Must provide repository_ids or full_names")
        return
    }
    
    // 3. Execute async operation
    migrationIDs := h.startMigrationsAsync(req)
    
    // 4. Return 202 Accepted
    h.sendJSON(w, http.StatusAccepted, map[string]interface{}{
        "migration_ids": migrationIDs,
        "count":         len(migrationIDs),
        "message":       "Migrations started",
    })
}
```

### Filtering Implementation

```go
type RepositoryFilters struct {
    Status          string
    BatchID         *int64
    Organization    []string
    Search          string
    HasLFS          *bool
    HasSubmodules   *bool
    MinSize         *int64
    MaxSize         *int64
    SortBy          string
    Limit           *int
    Offset          *int
}

func buildWhereClause(filters RepositoryFilters) (string, []interface{}) {
    conditions := []string{}
    args := []interface{}{}
    
    if filters.Status != "" {
        conditions = append(conditions, "status = ?")
        args = append(args, filters.Status)
    }
    
    if filters.BatchID != nil {
        conditions = append(conditions, "batch_id = ?")
        args = append(args, *filters.BatchID)
    }
    
    if len(filters.Organization) > 0 {
        placeholders := strings.Repeat(",?", len(filters.Organization))[1:]
        conditions = append(conditions, fmt.Sprintf("organization IN (%s)", placeholders))
        for _, org := range filters.Organization {
            args = append(args, org)
        }
    }
    
    if filters.Search != "" {
        conditions = append(conditions, "full_name LIKE ?")
        args = append(args, "%"+filters.Search+"%")
    }
    
    where := strings.Join(conditions, " AND ")
    if where != "" {
        where = "WHERE " + where
    }
    
    return where, args
}
```

### Pagination Patterns

```go
func (h *Handler) ListWithPagination(w http.ResponseWriter, r *http.Request) {
    limit := parseIntParam(r, "limit", 50)
    offset := parseIntParam(r, "offset", 0)
    
    repos, total, err := h.db.GetRepositoriesWithTotal(r.Context(), filters, limit, offset)
    if err != nil {
        h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
        return
    }
    
    h.sendJSON(w, http.StatusOK, map[string]interface{}{
        "repositories": repos,
        "total":        total,
        "limit":        limit,
        "offset":       offset,
        "has_more":     offset+len(repos) < total,
    })
}
```

---

## Migration Workflow Internals

### Detailed State Transitions

```
┌───────────────────────────────────────────────────────────────┐
│                    State Transition Diagram                    │
└───────────────────────────────────────────────────────────────┘

pending
    │
    ├──[START DRY RUN]───────────────────────────────────────────┐
    │                                                             │
    └──[START MIGRATION]────> pre_migration                      │
                                    │                            │
                                    ▼                            │
                              archive_generating                 │
                                    │                            │
                                    ▼                            │
                            queued_for_migration                 │
                                    │                            │
                                    ▼                            │
                            migrating_content                    │
                                    │                            │
                          ┌─────────┴─────────┐                 │
                          │                   │                 │
                          ▼                   ▼                 │
                  migration_complete    migration_failed        │
                          │                   │                 │
                          ▼                   │                 │
                    post_migration            │                 │
                          │                   │                 │
                          ▼                   │                 │
                      complete                │                 │
                                              │                 │
                                              │                 │
       ┌──────────────────────────────────────┴─────────────────┘
       │
       ▼
dry_run_in_progress
       │
       ├───> dry_run_complete
       │
       └───> dry_run_failed
```

### Lock Management

**When Locking Occurs:**
- Production migrations: `lock_repositories: true`
- Dry runs: `lock_repositories: false`

**Lock Process:**

```go
func (e *Executor) generateArchivesOnGHES(ctx context.Context, repo *models.Repository, lockRepos bool) (int64, error) {
    // Create migration on source with lock parameter
    migration, _, err := e.sourceClient.Rest().Migrations.StartMigration(ctx, org, &github.MigrationOptions{
        Repositories:       []string{repo.Name()},
        LockRepositories:   &lockRepos,
        ExcludeAttachments: github.Bool(false),
    })
    
    if err != nil {
        return 0, err
    }
    
    // Track lock status
    if lockRepos {
        repo.SourceMigrationID = migration.ID
        repo.IsSourceLocked = true
        e.storage.UpdateRepository(ctx, repo)
    }
    
    return *migration.ID, nil
}
```

**Unlock Process:**

```go
func (e *Executor) unlockSourceRepository(ctx context.Context, repo *models.Repository) error {
    if repo.SourceMigrationID == nil {
        return nil
    }
    
    // Unlock via API
    org := repo.Organization()
    _, err := e.sourceClient.Rest().Migrations.UnlockRepository(ctx, org, *repo.SourceMigrationID, repo.Name())
    
    if err != nil {
        e.logger.Error("Failed to unlock repository", "repo", repo.FullName, "error", err)
        return err
    }
    
    e.logger.Info("Repository unlocked", "repo", repo.FullName)
    return nil
}
```

**Unlock Triggers:**
- Migration success: Automatic unlock
- Migration failure: Automatic unlock
- Manual unlock: Via API endpoint `/api/v1/repositories/{fullName}/unlock`

### Rollback Mechanism

```go
func (e *Executor) RollbackMigration(ctx context.Context, repo *models.Repository, reason string) error {
    // 1. Verify repository can be rolled back
    if repo.Status != string(models.StatusComplete) {
        return fmt.Errorf("repository not in completed state")
    }
    
    if repo.DestinationFullName == nil {
        return fmt.Errorf("no destination repository to delete")
    }
    
    // 2. Delete destination repository
    destOrg := e.getDestinationOrg(repo)
    destName := e.getDestinationRepoName(repo)
    
    _, err := e.destClient.Rest().Repositories.Delete(ctx, destOrg, destName)
    if err != nil {
        return fmt.Errorf("failed to delete destination repository: %w", err)
    }
    
    // 3. Update repository status
    repo.Status = string(models.StatusRolledBack)
    repo.DestinationURL = nil
    repo.DestinationFullName = nil
    repo.MigratedAt = nil
    
    // 4. Create rollback history entry
    history := &models.MigrationHistory{
        RepositoryID: repo.ID,
        Status:       "rolled_back",
        Phase:        "rollback",
        Message:      &reason,
        StartedAt:    time.Now(),
    }
    e.storage.CreateMigrationHistory(ctx, history)
    
    // 5. Save repository
    return e.storage.UpdateRepository(ctx, repo)
}
```

---

## Extension Points

### Adding New Source Providers

The system uses a provider abstraction for source systems:

```go
type Provider interface {
    // GetType returns the provider type (currently only "github" is implemented)
    GetType() string
    
    // ListRepositories fetches repositories from the source
    ListRepositories(ctx context.Context, org string) ([]*Repository, error)
    
    // GetRepository fetches detailed repository information
    GetRepository(ctx context.Context, owner, name string) (*Repository, error)
    
    // CloneURL returns the clone URL for a repository
    CloneURL(repo *Repository) string
}
```

**Current Implementation Status:**

The provider interface is designed to support multiple source systems, but currently only GitHub to GitHub migrations are supported.

**Supported:**
- ✅ GitHub to GitHub migrations
  - Source: GitHub.com or GitHub Enterprise Server
  - Destination: GitHub.com, GitHub with data residency, or GitHub Enterprise Server
  - Both PAT and GitHub App authentication fully implemented

**Future Considerations:**

The provider abstraction allows for potential future support of other source systems (GitLab, Azure DevOps, etc.), but these are not currently implemented or on the roadmap. The focus is on providing the best possible GitHub to GitHub migration experience.

### Custom Migration Workflows

**Hook System (Planned):**

```go
type MigrationHook interface {
    PreMigration(ctx context.Context, repo *models.Repository) error
    PostMigration(ctx context.Context, repo *models.Repository) error
    OnFailure(ctx context.Context, repo *models.Repository, err error)
}

// Example: Notification Hook
type NotificationHook struct {
    webhookURL string
}

func (h *NotificationHook) PostMigration(ctx context.Context, repo *models.Repository) error {
    payload := map[string]interface{}{
        "repository": repo.FullName,
        "status":     "completed",
        "url":        repo.DestinationURL,
    }
    
    // Send webhook notification
    return sendWebhook(h.webhookURL, payload)
}
```

### Plugin Architecture (Future)

**Plugin Interface:**

```go
type Plugin interface {
    Name() string
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, data interface{}) error
}

type PluginManager struct {
    plugins map[string]Plugin
}

func (pm *PluginManager) Register(plugin Plugin) {
    pm.plugins[plugin.Name()] = plugin
}

func (pm *PluginManager) Execute(ctx context.Context, pluginName string, data interface{}) error {
    plugin, exists := pm.plugins[pluginName]
    if !exists {
        return fmt.Errorf("plugin not found: %s", pluginName)
    }
    
    return plugin.Execute(ctx, data)
}
```

### Webhook Integration (Future)

**Event System:**

```go
type Event struct {
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
}

const (
    EventDiscoveryStarted   = "discovery.started"
    EventDiscoveryCompleted = "discovery.completed"
    EventMigrationStarted   = "migration.started"
    EventMigrationCompleted = "migration.completed"
    EventMigrationFailed    = "migration.failed"
)

type WebhookManager struct {
    subscribers map[string][]string  // event type -> webhook URLs
}

func (wm *WebhookManager) Publish(event Event) {
    urls := wm.subscribers[event.Type]
    for _, url := range urls {
        go wm.sendWebhook(url, event)
    }
}
```

---

## Performance Considerations

### Parallel Processing

**Discovery Workers:**
```go
const (
    DefaultDiscoveryWorkers = 10
    MaxDiscoveryWorkers     = 50
)

func (c *Collector) DiscoverWithWorkers(ctx context.Context, org string, workers int) error {
    if workers > MaxDiscoveryWorkers {
        workers = MaxDiscoveryWorkers
    }
    
    // Use semaphore pattern for controlled concurrency
    semaphore := make(chan struct{}, workers)
    // ... worker implementation
}
```

**Migration Workers:**
```go
const (
    DefaultMigrationWorkers = 5
    MaxMigrationWorkers     = 20
)

// Configuration
migration:
  workers: 10  # Adjust based on rate limits and resources
```

### Database Optimization

**Connection Pooling:**

```go
// PostgreSQL
db.SetMaxOpenConns(50)        // Maximum connections
db.SetMaxIdleConns(10)        // Idle connections
db.SetConnMaxLifetime(10 * time.Minute)

// SQLite (single connection)
db.SetMaxOpenConns(1)
```

**Query Optimization:**

```go
// Use prepared statements
stmt, err := db.PrepareContext(ctx, "SELECT * FROM repositories WHERE status = ?")
defer stmt.Close()

// Batch inserts
tx, _ := db.BeginTx(ctx, nil)
for _, repo := range repos {
    tx.ExecContext(ctx, "INSERT INTO repositories (...) VALUES (...)", ...)
}
tx.Commit()

// Use indexes for frequent queries
CREATE INDEX idx_repositories_status_batch ON repositories(status, batch_id);
```

### Rate Limit Management

**Strategies:**

1. **Auto-Wait**: Wait for rate limit reset (default)
2. **Fail Fast**: Return error immediately
3. **Token Rotation**: Use multiple tokens (future)

```go
rateLimit:
  strategy: "auto_wait"  # auto_wait, fail_fast, rotate
  wait_on_exhaustion: true
  check_before_request: true
```

### Memory Usage Patterns

**Large Repository Handling:**

```go
// Don't load entire repository into memory
func (p *Profiler) Profile(ctx context.Context, repo *models.Repository) error {
    // Use bare clone (no working directory)
    git clone --bare <url> /tmp/repo
    
    // Stream git-sizer output
    cmd := exec.Command("git-sizer", "--json")
    output, _ := cmd.Output()
    
    // Parse incrementally
    parseGitSizerOutput(output)
    
    // Clean up immediately
    defer os.RemoveAll(tempDir)
}
```

**Pagination:**

```go
// Don't fetch all repositories at once
func (db *Database) GetRepositories(ctx context.Context, filters Filters) ([]*Repository, error) {
    limit := 1000  // Reasonable default
    if filters.Limit != nil {
        limit = *filters.Limit
    }
    
    query := "SELECT * FROM repositories WHERE ... LIMIT ? OFFSET ?"
    // ...
}
```

---

## Security

### Token Management

**Storage:**
- Tokens stored in environment variables or configuration files
- Never logged or displayed in UI/logs
- Masked in error messages

```go
func maskToken(token string) string {
    if len(token) < 10 {
        return "***"
    }
    return token[:4] + "..." + token[len(token)-4:]
}

// Log with masked token
logger.Info("Using token", "token", maskToken(token))
```

### Input Validation

**SQL Injection Prevention:**

```go
// Always use parameterized queries
query := "SELECT * FROM repositories WHERE full_name = ?"
row := db.QueryRowContext(ctx, query, fullName)

// Never concatenate user input
// BAD: query := "SELECT * FROM repositories WHERE full_name = '" + fullName + "'"
```

**Request Validation:**

```go
func validateRepositoryName(name string) error {
    if name == "" {
        return fmt.Errorf("repository name cannot be empty")
    }
    
    // Check format: org/repo
    parts := strings.Split(name, "/")
    if len(parts) != 2 {
        return fmt.Errorf("invalid repository name format: %s", name)
    }
    
    // Check for valid characters (alphanumeric, hyphens, underscores)
    validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
    if !validName.MatchString(parts[0]) || !validName.MatchString(parts[1]) {
        return fmt.Errorf("repository name contains invalid characters")
    }
    
    return nil
}
```

### XSS Protection

**Frontend:**
- React automatically escapes user input
- Use `DOMPurify` for rendering HTML content
- CSP headers (planned)

```typescript
// Safe by default in React
<div>{repository.name}</div>

// For rendering HTML, sanitize first
import DOMPurify from 'dompurify';
const clean = DOMPurify.sanitize(userContent);
<div dangerouslySetInnerHTML={{ __html: clean }} />
```

**Backend:**

```go
// Escape HTML in responses
import "html"

message := html.EscapeString(userInput)
```

---

## Troubleshooting

### Common Issues and Solutions

#### 1. Database Lock Errors (SQLite)

**Issue:**
```
database is locked
```

**Cause:**
- Multiple processes accessing SQLite
- Long-running transactions

**Solution:**
```go
// Ensure only one process uses SQLite
// For production, use PostgreSQL

// Or configure SQLite for better concurrency
db, _ := sql.Open("sqlite3", "file:migrator.db?_busy_timeout=5000&_journal_mode=WAL")
```

#### 2. Rate Limit Exhaustion

**Issue:**
```
API rate limit exceeded
```

**Cause:**
- Too many concurrent operations
- Insufficient rate limit budget

**Solution:**
```yaml
# Reduce workers
migration:
  workers: 3

discovery:
  workers: 5

# Enable auto-wait
github:
  rate_limit:
    wait_on_exhaustion: true
```

#### 3. Migration Stuck

**Issue:**
Migration remains in "migrating_content" status indefinitely

**Cause:**
- GitHub migration failed silently
- Network connectivity issues

**Debug:**
```go
// Check migration status on GitHub directly
query {
  node(id: "migration_id") {
    ... on Migration {
      state
      failureReason
    }
  }
}

// Check repository lock status
SELECT is_source_locked, source_migration_id 
FROM repositories 
WHERE full_name = 'org/repo';
```

**Solution:**
```bash
# Manually unlock if stuck
curl -X POST http://localhost:8080/api/v1/repositories/org%2Frepo/unlock
```

#### 4. Out of Memory

**Issue:**
```
cannot allocate memory
```

**Cause:**
- Too many parallel operations
- Large repository profiling

**Solution:**
```yaml
# Reduce parallelism
migration:
  workers: 2

discovery:
  workers: 3

# Skip profiling for very large repos
# Or increase memory limit for Docker
docker update --memory 2g github-migrator
```

### Debugging Techniques

**Enable Debug Logging:**

```yaml
logging:
  level: debug
  format: text  # More readable than JSON
```

**Inspect Database:**

```bash
sqlite3 data/migrator.db

-- Check repository status
SELECT full_name, status, updated_at 
FROM repositories 
WHERE status LIKE '%migrat%' 
ORDER BY updated_at DESC;

-- Check migration history
SELECT r.full_name, mh.phase, mh.status, mh.error_message, mh.started_at
FROM migration_history mh
JOIN repositories r ON r.id = mh.repository_id
WHERE r.full_name = 'org/repo'
ORDER BY mh.started_at DESC;

-- Check migration logs
SELECT level, phase, operation, message, timestamp
FROM migration_logs
WHERE repository_id = 123
ORDER BY timestamp DESC
LIMIT 50;
```

**Log Analysis:**

```bash
# Filter by error level
tail -1000 logs/migrator.log | jq 'select(.level=="ERROR")'

# Filter by repository
tail -1000 logs/migrator.log | jq 'select(.repo=="org/repo")'

# Filter by time range
tail -1000 logs/migrator.log | jq 'select(.time > "2024-01-15T10:00:00Z")'
```

**Performance Profiling:**

```go
import _ "net/http/pprof"

// Add to main.go
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Access profiler
// http://localhost:6060/debug/pprof/
```

---

## Summary

This implementation guide covers the internal architecture and technical details of the GitHub Migrator. For operational procedures, see [OPERATIONS.md](./OPERATIONS.md). For API usage, see [API.md](./API.md). For contributing, see [CONTRIBUTING.md](./CONTRIBUTING.md).

**Key Takeaways:**

1. **Modular Architecture**: Clear separation of concerns with well-defined interfaces
2. **Dual Authentication**: Leverages both PAT and GitHub App for optimal rate limits
3. **State Machine**: Sequential migration phases with clear transitions
4. **Extensibility**: Provider abstraction allows adding new source systems
5. **Observability**: Comprehensive logging and history tracking
6. **Performance**: Parallel processing with rate limit management
7. **Reliability**: Retry logic, error recovery, and rollback support

---

**Implementation Guide Version:** 1.0.0  
**Last Updated:** October 2025  
**Status:** Production Ready

