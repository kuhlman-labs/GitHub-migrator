# Backend Architecture

This document provides an overview of the GitHub Migrator backend architecture.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                    CLIENTS                                          │
│                         (Web UI, CLI, External Systems)                             │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              API LAYER (internal/api)                               │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                           HTTP Server & Router                                 │  │
│  │                              (server.go)                                       │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────┐ │
│  │  Middleware │  │    Auth     │  │   Logging   │  │        Rate Limiting        │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────────────┘ │
│                                        │                                            │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                        HANDLERS (internal/api/handlers)                        │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │  │
│  │  │ Repository   │ │    Batch     │ │  Analytics   │ │      Migration       │   │  │
│  │  │   Handler    │ │   Handler    │ │   Handler    │ │      Handlers        │   │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │  │
│  │  │   Discovery  │ │     Team     │ │     User     │ │     ADO Handlers     │   │  │
│  │  │   Handlers   │ │   Handlers   │ │   Handlers   │ │                      │   │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │  │
│  │  │     Auth     │ │    Setup     │ │   Source     │ │     Settings         │   │  │
│  │  │   Handlers   │ │   Handlers   │ │   Handlers   │ │     Handlers         │   │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                            │  │
│  │  │ Organization │ │  Dependency  │ │   Copilot    │                            │  │
│  │  │   Handlers   │ │   Handlers   │ │   Handlers   │                            │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                            │  │
│  │                              │                                                  │  │
│  │  ┌───────────────────────────────────────────────────────────────────────────┐ │  │
│  │  │                    HandlerUtils (Shared Utilities)                        │ │  │
│  │  │            CheckRepositoryAccess(), GetClientForOrg()                     │ │  │
│  │  └───────────────────────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          SERVICE LAYER (internal/services)                          │
│  ┌─────────────────────────────────┐  ┌─────────────────────────────────────────┐   │
│  │       RepositoryService         │  │            BatchService                 │   │
│  │  - GetRepositoryWithDetails()   │  │  - GetBatchWithStats()                  │   │
│  │  - MarkAsWontMigrate()          │  │  - AddRepositoriesToBatch()             │   │
│  │  - CheckBatchEligibility()      │  │  - DryRunBatch()                        │   │
│  │  - RediscoverRepository()       │  │  - StartBatch()                         │   │
│  └─────────────────────────────────┘  └─────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
            ┌───────────────────────────┼───────────────────────────┐
            ▼                           ▼                           ▼
┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────────────┐
│   DISCOVERY LAYER     │  │   MIGRATION LAYER     │  │       BATCH LAYER             │
│ (internal/discovery)  │  │  (internal/migration) │  │     (internal/batch)          │
│                       │  │                       │  │                               │
│ ┌───────────────────┐ │  │ ┌───────────────────┐ │  │ ┌─────────────────────────┐   │
│ │    Collector      │ │  │ │     Executor      │ │  │ │     Orchestrator        │   │
│ │  - Orchestrates   │ │  │ │  - Multi-phase    │ │  │ │  - Batch coordination   │   │
│ │    discovery      │ │  │ │    migration      │ │  │ └─────────────────────────┘   │
│ └───────────────────┘ │  │ │  - Polling        │ │  │ ┌─────────────────────────┐   │
│ ┌───────────────────┐ │  │ │  - Validation     │ │  │ │      Scheduler          │   │
│ │   ADOCollector    │ │  │ └───────────────────┘ │  │ │  - Priority queuing     │   │
│ │  - ADO discovery  │ │  │ ┌───────────────────┐ │  │ └─────────────────────────┘   │
│ └───────────────────┘ │  │ │  ExecutorFactory  │ │  │ ┌─────────────────────────┐   │
│ ┌───────────────────┐ │  │ │  - Create executor│ │  │ │     Organizer           │   │
│ │  RepoDiscoverer   │ │  │ │    per strategy   │ │  │ │  - Batch grouping       │   │
│ │  - List repos     │ │  │ └───────────────────┘ │  │ └─────────────────────────┘   │
│ │  - Filter/Stats   │ │  │ ┌───────────────────┐ │  │ ┌─────────────────────────┐   │
│ └───────────────────┘ │  │ │    Strategies     │ │  │ │    StatusUpdater        │   │
│ ┌───────────────────┐ │  │ │  - GitHub         │ │  │ │  - Batch status sync    │   │
│ │  TeamDiscoverer   │ │  │ │  - ADO            │ │  │ └─────────────────────────┘   │
│ │  - Teams & roles  │ │  │ └───────────────────┘ │  └───────────────────────────────┘
│ └───────────────────┘ │  │ ┌───────────────────┐ │
│ ┌───────────────────┐ │  │ │  TeamExecutor     │ │
│ │ MemberDiscoverer  │ │  │ │  - Team migration │ │
│ │  - Org members    │ │  │ └───────────────────┘ │
│ └───────────────────┘ │  └───────────────────────┘
│ ┌───────────────────┐ │
│ │     Profiler      │ │
│ │  - Repo analysis  │ │
│ │  - Complexity     │ │
│ └───────────────────┘ │
│ ┌───────────────────┐ │
│ │   ADOProfiler     │ │
│ │  - ADO repo       │ │
│ │    profiling      │ │
│ └───────────────────┘ │
│ ┌───────────────────┐ │
│ │     Analyzer      │ │
│ │  - git-sizer      │ │
│ │  - git count-obj  │ │
│ └───────────────────┘ │
│ ┌───────────────────┐ │
│ │ DependencyAnalyzer│ │
│ │  - Package deps   │ │
│ │  - Submodules     │ │
│ │  - Workflows      │ │
│ └───────────────────┘ │
│ ┌───────────────────┐ │
│ │  PackageScanner   │ │
│ │  - Multi-lang     │ │
│ │    manifest parse │ │
│ └───────────────────┘ │
│ ┌───────────────────┐ │
│ │ ProgressTracker   │ │
│ │  - Discovery      │ │
│ │    progress       │ │
│ └───────────────────┘ │
└───────────────────────┘
            │                           │                           │
            └───────────────────────────┼───────────────────────────┘
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          STORAGE LAYER (internal/storage)                           │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                              Database                                          │  │
│  │                         (GORM-based ORM)                                       │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                            │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                         Focused Interfaces                                     │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │  │
│  │  │ Repository   │ │   Batch      │ │  Analytics   │ │  MigrationHistory    │   │  │
│  │  │   Store      │ │   Store      │ │    Store     │ │      Store           │   │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │  │
│  │  │  Dependency  │ │    User      │ │    Team      │ │       ADO            │   │  │
│  │  │    Store     │ │   Store      │ │   Store      │ │      Store           │   │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │  │
│  │  │ UserMapping  │ │ TeamMapping  │ │   Source     │ │     Discovery        │   │  │
│  │  │    Store     │ │    Store     │ │    Store     │ │      Store           │   │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐   │  │
│  │  │    Setup     │ │  Settings    │ │   Copilot    │ │  UserMannequin       │   │  │
│  │  │    Store     │ │    Store     │ │    Store     │ │      Store           │   │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                            │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                           DialectDialer                                        │  │
│  │           (Portable SQL for SQLite, PostgreSQL, SQL Server)                    │  │
│  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────────────────────┐   │  │
│  │  │  SQLiteDialect  │ │ PostgresDialect │ │       SQLServerDialect          │   │  │
│  │  └─────────────────┘ └─────────────────┘ └─────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
                              ┌─────────────────┐
                              │    Database     │
                              │ SQLite/Postgres │
                              │   SQL Server    │
                              └─────────────────┘
```

## External Integrations

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                        EXTERNAL CLIENTS (internal/github, internal/azuredevops)     │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐    │
│  │                          GitHub Client                                       │    │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────────────────┐  │    │
│  │  │   REST API     │  │  GraphQL API   │  │        Rate Limiter            │  │    │
│  │  └────────────────┘  └────────────────┘  └────────────────────────────────┘  │    │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────────────────┐  │    │
│  │  │  Migrations    │  │     Teams      │  │       Organizations            │  │    │
│  │  └────────────────┘  └────────────────┘  └────────────────────────────────┘  │    │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────────────────┐  │    │
│  │  │  Dependencies  │  │ Retry/Circuit  │  │        Error Handling          │  │    │
│  │  │   (GraphQL)    │  │   Breaker      │  │                                │  │    │
│  │  └────────────────┘  └────────────────┘  └────────────────────────────────┘  │    │
│  │  ┌────────────────────────────────────────────────────────────────────────┐  │    │
│  │  │                       DualClient                                        │  │    │
│  │  │            (Manages PAT + GitHub App token switching)                   │  │    │
│  │  └────────────────────────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐    │
│  │                      Azure DevOps Client                                     │    │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────────────────┐  │    │
│  │  │  Repositories  │  │   Pipelines    │  │         Work Items             │  │    │
│  │  └────────────────┘  └────────────────┘  └────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
            ┌───────────────────────────┴───────────────────────────┐
            │                                                       │
┌───────────────────────────┐                       ┌───────────────────────────┐
│  GitHub Enterprise Server │                       │      Azure DevOps         │
│  GitHub Enterprise Cloud  │                       │   (Projects, Repos,       │
│        github.com         │                       │   Pipelines, Boards)      │
└───────────────────────────┘                       └───────────────────────────┘
```

## Authentication & Authorization

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                            AUTH LAYER (internal/auth)                               │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐    │
│  │                         Authentication Flow                                  │    │
│  │                                                                              │    │
│  │    ┌─────────────┐      ┌─────────────┐      ┌─────────────────────────┐     │    │
│  │    │   GitHub    │ ──▶  │    OAuth    │ ──▶  │   JWT Token             │     │    │
│  │    │   OAuth     │      │  Callback   │      │   Generation            │     │    │
│  │    └─────────────┘      └─────────────┘      └─────────────────────────┘     │    │
│  │                                                                              │    │
│  │    Supports GitHub.com and GitHub Enterprise Server OAuth providers.          │    │
│  │    Encrypted JWT tokens with AES-256-GCM for session management.             │    │
│  └─────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐    │
│  │                         Authorization                                        │    │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────────┐   │    │
│  │  │    Authorizer    │  │ PermissionChecker│  │       Middleware         │   │    │
│  │  │  - Role checks   │  │  - Repo access   │  │  - Token validation      │   │    │
│  │  │  - Org admin     │  │  - Org membership│  │  - Context population    │   │    │
│  │  └──────────────────┘  └──────────────────┘  └──────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## Data Models

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                            MODELS (internal/models)                                 │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐    │
│  │                       Repository (core model)                                │    │
│  │    Linked via separate table structs for normalized storage:                  │    │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────────────────┐  │    │
│  │  │ Repository     │  │ Repository     │  │    Repository                  │  │    │
│  │  │ GitProperties  │  │   Features     │  │    ADOProperties               │  │    │
│  │  │ - Size, LFS    │  │ - Wiki, Pages  │  │  - Project, Pipelines          │  │    │
│  │  │ - Branches     │  │ - Actions      │  │  - Work Items                  │  │    │
│  │  │ - Commits      │  │ - Packages     │  │                                │  │    │
│  │  └────────────────┘  └────────────────┘  └────────────────────────────────┘  │    │
│  │  ┌────────────────┐                                                          │    │
│  │  │ Repository     │                                                          │    │
│  │  │  Validation    │                                                          │    │
│  │  │ - Complexity   │                                                          │    │
│  │  └────────────────┘                                                          │    │
│  └─────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐     │
│  │    Batch     │  │  Migration   │  │  GitHubTeam  │  │     GitHubUser       │     │
│  │              │  │   History    │  │              │  │                      │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────────────┘     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐     │
│  │ UserMapping  │  │ TeamMapping  │  │  ADOProject  │  │ DiscoveryProgress    │     │
│  │              │  │              │  │              │  │                      │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────────────┘     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐     │
│  │    Source    │  │  Settings    │  │   Copilot    │  │  UserMannequin       │     │
│  │              │  │              │  │   Session    │  │                      │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## Request Flow

```
┌────────────┐     ┌────────────┐     ┌────────────┐     ┌────────────┐
│   Client   │ ──▶ │   Router   │ ──▶ │ Middleware │ ──▶ │  Handler   │
│  Request   │     │            │     │  (Auth,    │     │            │
│            │     │            │     │  Logging)  │     │            │
└────────────┘     └────────────┘     └────────────┘     └────────────┘
                                                               │
                                                               ▼
┌────────────┐     ┌────────────┐     ┌────────────┐     ┌────────────┐
│  Response  │ ◀── │  Handler   │ ◀── │  Service   │ ◀── │  Storage   │
│            │     │            │     │   Layer    │     │   Layer    │
└────────────┘     └────────────┘     └────────────┘     └────────────┘
```

## Migration Flow

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Migration Execution Flow                          │
│                                                                          │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌────────┐ │
│  │   Pre-   │──▶│ Archive  │──▶│  Queue   │──▶│ Migrate  │──▶│  Post- │ │
│  │Migration │   │ Generate │   │   for    │   │ Content  │   │Migrate │ │
│  └──────────┘   └──────────┘   │Migration │   └──────────┘   └────────┘ │
│       │              │         └──────────┘        │              │      │
│       ▼              ▼              │              ▼              ▼      │
│  ┌──────────┐   ┌──────────┐        │         ┌──────────┐   ┌────────┐ │
│  │ Lock     │   │ Create   │        │         │  Poll    │   │Validate│ │
│  │ Source   │   │ Archive  │        │         │ Progress │   │ Verify │ │
│  │ Repo     │   │  on      │        │         │          │   │        │ │
│  └──────────┘   │ Source   │        │         └──────────┘   └────────┘ │
│                 └──────────┘        │                                    │
│                                     ▼                                    │
│                              ┌──────────┐                               │
│                              │  Start   │                               │
│                              │Migration │                               │
│                              │  on GH   │                               │
│                              └──────────┘                               │
└──────────────────────────────────────────────────────────────────────────┘
```

## Package Dependencies

```
                                 ┌─────────┐
                                 │   cmd   │
                                 │ server  │
                                 └────┬────┘
                                      │
                                      ▼
                                 ┌─────────┐
                                 │   api   │
                                 │ server  │
                                 └────┬────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
             ┌──────────┐      ┌──────────┐      ┌──────────┐
             │ handlers │      │   auth   │      │middleware│
             └────┬─────┘      └──────────┘      └──────────┘
                  │
    ┌─────────────┼─────────────┬─────────────┐
    │             │             │             │
    ▼             ▼             ▼             ▼
┌────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│services│  │ discovery│  │ migration│  │ copilot  │
└───┬────┘  └────┬─────┘  └────┬─────┘  └──────────┘
    │            │             │
    │       ┌────┴────┐        │
    │       │         │        │
    ▼       ▼         ▼        ▼
┌────────────────┐  ┌────────────────┐
│    storage     │  │     github     │
│                │  │   azuredevops  │
└───────┬────────┘  └────────────────┘
        │                   │
        ▼                   ▼
┌────────────────┐  ┌────────────────┐
│     models     │  │     source     │
│    config      │  │   configsvc   │
│    logging     │  │      mcp      │
└────────────────┘  └────────────────┘
```

## Key Design Patterns

### 1. **Layered Architecture**
- Clear separation: API → Service → Storage
- Each layer has focused responsibilities
- Dependency injection for testability

### 2. **Interface Segregation**
- Focused interfaces (RepositoryStore, BatchStore, etc.)
- Composable DataStore interface
- Easy mocking for tests

### 3. **Strategy Pattern**
- Migration strategies for GitHub vs ADO
- Dialect strategies for SQL databases

### 4. **Component-Based Models**
- Repository fields grouped into logical components
- Getter/Setter methods for component access
- Prepared for future struct embedding

### 5. **Shared Utilities**
- HandlerUtils for common handler operations
- TeamMemberSaver for discovery operations
- Reduces code duplication

## Directory Structure

```
internal/
├── ado/              # Azure DevOps URL parsing utilities
├── api/              # HTTP API layer
│   ├── handlers/     # Domain-specific HTTP handlers
│   └── middleware/   # HTTP middleware (CORS, logging)
├── auth/             # Authentication & authorization (OAuth, JWT, RBAC)
├── azuredevops/      # Azure DevOps API client
├── batch/            # Batch orchestration, scheduling & status
├── config/           # Configuration loading (Viper)
├── configsvc/        # Dynamic configuration service (from DB)
├── copilot/          # GitHub Copilot integration (SDK, chat, licenses)
├── discovery/        # Repository/team/member discovery & profiling
├── embedded/         # Embedded binaries (git-sizer)
├── github/           # GitHub API client (REST + GraphQL + DualClient)
├── logging/          # Structured logging (slog)
├── mcp/              # Model Context Protocol server
├── migration/        # Migration execution, strategies & validation
├── models/           # Data models, constants & settings
├── services/         # Business logic layer
├── source/           # Source provider abstraction (GitHub, ADO, GitLab)
├── storage/          # Database access layer (GORM)
│   └── migrations/   # SQL migration files (SQLite, Postgres, SQL Server)
└── worker/           # Background worker pool & scheduler
```

