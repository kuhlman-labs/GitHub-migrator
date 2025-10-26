# Contributing to GitHub Migrator

Thank you for your interest in contributing to the GitHub Migrator. This guide will help you get started with development, understand our standards, and make effective contributions.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Making Changes](#making-changes)
- [Architecture Overview](#architecture-overview)
- [Debugging Tips](#debugging-tips)
- [Available Make Targets](#available-make-targets)
- [Project Conventions](#project-conventions)

---

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go** 1.21 or higher ([installation guide](https://go.dev/doc/install))
- **Node.js** 20 or higher ([installation guide](https://nodejs.org/))
- **Git** 2.30+ with Git LFS support
- **Make** (usually pre-installed on macOS/Linux)
- **Docker** (optional, for containerized development)
- **A code editor** (VS Code, GoLand, or your preferred IDE)

### Initial Setup

1. **Fork and clone the repository**:
   ```bash
   git fork https://github.com/your-org/github-migrator.git
   cd github-migrator
   ```

2. **Install all dependencies and tools**:
   ```bash
   make install
   ```
   
   This command will:
   - Download all Go module dependencies
   - Install development tools (`golangci-lint`, `gosec`, `git-sizer`)
   - Install frontend npm packages
   - Download embedded git-sizer binaries

3. **Set up your configuration**:
   ```bash
   cp configs/config.yaml configs/local.yaml
   ```
   
   Edit `configs/local.yaml` with your development settings:
   ```yaml
   server:
     port: 8080

   database:
     type: sqlite
     dsn: ./data/migrator-dev.db

   source:
     type: github
     base_url: "https://github.company.com/api/v3"
     token: "${GITHUB_SOURCE_TOKEN}"

   destination:
     type: github
     base_url: "https://api.github.com"
     token: "${GITHUB_DEST_TOKEN}"

   logging:
     level: debug
     format: text  # More readable for development
   ```

4. **Set environment variables**:
   ```bash
   export GITHUB_SOURCE_TOKEN="ghp_your_source_token"
   export GITHUB_DEST_TOKEN="ghp_your_dest_token"
   ```

5. **Verify your setup**:
   ```bash
   make test
   make lint
   ```

---

## Development Environment

### Running the Application

#### Backend Development

```bash
# Run the backend server
make run-server

# Backend will be available at http://localhost:8080
# API endpoints at http://localhost:8080/api/v1/
```

The server will automatically:
- Create the SQLite database if it doesn't exist
- Run database migrations
- Load configuration from `configs/config.yaml`
- Start the HTTP server

#### Frontend Development

```bash
# Run the frontend dev server (in a separate terminal)
make web-dev

# Frontend will be available at http://localhost:3000
# Hot reload enabled for rapid development
```

#### Full Stack Development

For convenience, run both in separate terminals:

Terminal 1:
```bash
make run-server
```

Terminal 2:
```bash
make web-dev
```

#### Docker Development

```bash
# Build and run with Docker Compose
make docker-build
make docker-run

# View logs
docker-compose logs -f

# Stop containers
make docker-down
```

### Database Setup

#### SQLite (Default for Development)

SQLite is used by default and requires no setup:

```yaml
database:
  type: sqlite
  dsn: ./data/migrator-dev.db
```

The database file is created automatically in the `data/` directory.

**Inspecting the database:**
```bash
# Open SQLite CLI
sqlite3 data/migrator-dev.db

# Useful commands:
.tables                          # List all tables
.schema repositories             # Show table schema
SELECT * FROM repositories LIMIT 5;
.exit
```

#### PostgreSQL (Optional)

For testing PostgreSQL locally:

```bash
# Start PostgreSQL with Docker
docker run -d \
  --name migrator-postgres \
  -e POSTGRES_DB=migrator \
  -e POSTGRES_USER=migrator \
  -e POSTGRES_PASSWORD=dev \
  -p 5432:5432 \
  postgres:15

# Update your config
database:
  type: postgresql
  dsn: "host=localhost port=5432 user=migrator password=dev dbname=migrator sslmode=disable"
```

### Hot Reloading

For faster backend development with hot reloading:

```bash
# Install air (Go hot reload)
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

---

## Code Standards

### Go Code Style

#### Formatting

- **Always run `gofmt`** before committing:
  ```bash
  make fmt
  ```

- Use **tabs for indentation** (Go standard)
- Line length: **aim for 100 characters**, but not a hard limit
- **Group imports** into three categories:
  ```go
  import (
      // Standard library
      "context"
      "fmt"
      "log"

      // Third-party packages
      "github.com/google/go-github/v75/github"
      "github.com/spf13/viper"

      // Internal packages
      "github.com/brettkuhlman/github-migrator/internal/models"
      "github.com/brettkuhlman/github-migrator/internal/storage"
  )
  ```

#### Naming Conventions

- **Packages**: lowercase, single word (e.g., `storage`, `github`, `models`)
- **Files**: lowercase with underscores (e.g., `dual_client.go`, `rate_limiter.go`)
- **Types**: PascalCase (e.g., `DualClient`, `Repository`, `MigrationHistory`)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
  ```go
  // Exported
  func StartMigration(ctx context.Context) error

  // Unexported
  func validateRepository(repo *models.Repository) error
  ```
- **Variables**: camelCase (e.g., `repoCount`, `migrationID`)
- **Constants**: PascalCase or SCREAMING_SNAKE_CASE for package-level
  ```go
  const (
      MaxRetries = 3
      DefaultTimeout = 30 * time.Second
  )
  ```

#### Comments

- **Package comments**: Every package should have a doc comment
  ```go
  // Package discovery provides repository discovery and profiling capabilities.
  package discovery
  ```

- **Exported functions**: Must have comments starting with the function name
  ```go
  // StartDiscovery initiates repository discovery for the given organization.
  // It returns an error if discovery is already in progress or if the
  // organization is not accessible.
  func StartDiscovery(ctx context.Context, org string) error {
  ```

- **Struct fields**: Document non-obvious fields
  ```go
  type Repository struct {
      ID       int64  `json:"id"`
      FullName string `json:"full_name"`
      
      // TotalSize is the repository size in bytes, including LFS objects
      TotalSize *int64 `json:"total_size,omitempty"`
  }
  ```

#### Error Handling

- **Always handle errors explicitly** (no naked returns with errors):
  ```go
  // Good
  repos, err := storage.GetRepositories(ctx)
  if err != nil {
      return fmt.Errorf("failed to get repositories: %w", err)
  }

  // Bad
  repos, _ := storage.GetRepositories(ctx)  // Never ignore errors
  ```

- **Wrap errors with context** using `%w`:
  ```go
  return fmt.Errorf("failed to migrate repository %s: %w", repo.FullName, err)
  ```

- **Check errors first** (happy path last):
  ```go
  func DoSomething() error {
      if err := validate(); err != nil {
          return err
      }
      if err := execute(); err != nil {
          return err
      }
      return nil
  }
  ```

#### Logging

Use structured logging with `slog`:

```go
import "log/slog"

// Log with context
logger.Info("starting migration",
    slog.String("repository", repo.FullName),
    slog.Int64("id", repo.ID),
)

// Log errors
logger.Error("migration failed",
    slog.String("repository", repo.FullName),
    slog.Any("error", err),
)

// Different levels
logger.Debug("detailed debugging info")
logger.Info("general information")
logger.Warn("warning message")
logger.Error("error message")
```

### TypeScript/React Code Style

#### Formatting

- **Use Prettier** (configured in project):
  ```bash
  cd web && npm run format
  ```

- **Use ESLint**:
  ```bash
  make web-lint
  ```

- Use **2 spaces** for indentation
- Use **semicolons**
- Use **single quotes** for strings

#### Component Structure

```typescript
// Use functional components with TypeScript
interface ComponentProps {
  title: string;
  count: number;
  onAction?: () => void;
}

export const MyComponent: React.FC<ComponentProps> = ({ title, count, onAction }) => {
  const [state, setState] = useState<string>('');

  useEffect(() => {
    // Side effects
  }, []);

  const handleClick = () => {
    onAction?.();
  };

  return (
    <div className="container">
      <h1>{title}</h1>
      <p>Count: {count}</p>
    </div>
  );
};
```

#### File Organization

- **Components**: One component per file, named with PascalCase
  ```
  components/
    Dashboard/
      index.tsx           # Main component
      DashboardCard.tsx   # Sub-components
      types.ts            # Component-specific types
  ```

- **Hooks**: Custom hooks in `hooks/`, prefix with `use`
  ```typescript
  // useRepositories.ts
  export const useRepositories = () => {
    // Hook logic
  };
  ```

- **Types**: Shared types in `types/`, named with PascalCase
  ```typescript
  export interface Repository {
    id: number;
    fullName: string;
    status: string;
  }
  ```

#### Naming Conventions

- **Components**: PascalCase (e.g., `RepositoryList`, `MigrationStatus`)
- **Files**: PascalCase for components, camelCase for utilities
- **Props Interfaces**: Suffix with `Props` (e.g., `RepositoryListProps`)
- **Event Handlers**: Prefix with `handle` (e.g., `handleClick`, `handleSubmit`)
- **Boolean Props**: Prefix with `is`, `has`, `should` (e.g., `isLoading`, `hasError`)

---

## Testing

### Backend Testing

#### Writing Tests

- **Test files**: Named `*_test.go` in the same package
- **Test functions**: Prefix with `Test`
- **Table-driven tests** for multiple cases:

```go
func TestValidateRepository(t *testing.T) {
    tests := []struct {
        name    string
        repo    *models.Repository
        wantErr bool
    }{
        {
            name: "valid repository",
            repo: &models.Repository{
                ID:       1,
                FullName: "org/repo",
                Status:   "pending",
            },
            wantErr: false,
        },
        {
            name: "empty full name",
            repo: &models.Repository{
                ID:     1,
                Status: "pending",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateRepository(tt.repo)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateRepository() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test -v ./internal/storage/...

# Run specific test
go test -v -run TestValidateRepository ./internal/storage/

# Run with race detector (always use for concurrent code)
go test -v -race ./...
```

#### Test Coverage Requirements

- **Minimum**: 60% overall coverage
- **Target**: 70%+ for new code
- **Critical paths**: 80%+ coverage required (API handlers, storage, migration logic)

View coverage:
```bash
make test-coverage
open coverage.html
```

#### Mocking

Use interfaces for mocking dependencies:

```go
type GitHubClient interface {
    GetRepository(ctx context.Context, owner, name string) (*Repository, error)
}

// In tests
type mockGitHubClient struct {
    getRepoFunc func(ctx context.Context, owner, name string) (*Repository, error)
}

func (m *mockGitHubClient) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
    if m.getRepoFunc != nil {
        return m.getRepoFunc(ctx, owner, name)
    }
    return nil, nil
}
```

### Frontend Testing

```bash
# Run frontend tests
cd web && npm test

# With coverage
cd web && npm test -- --coverage

# Watch mode for development
cd web && npm test -- --watch
```

### Integration Tests

Integration tests are in `internal/api/integration_test.go`:

```bash
# Run integration tests
go test -v -tags=integration ./internal/api/
```

---

## Making Changes

### Branch Naming

Use descriptive branch names with prefixes:

- `feature/` - New features (e.g., `feature/add-retry-logic`)
- `fix/` - Bug fixes (e.g., `fix/migration-status-update`)
- `docs/` - Documentation (e.g., `docs/update-contributing-guide`)
- `refactor/` - Code refactoring (e.g., `refactor/simplify-client-logic`)
- `test/` - Test additions/updates (e.g., `test/add-storage-tests`)

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic changes)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(discovery): add support for enterprise-wide discovery

Implement discovery across all organizations in an enterprise.
Adds new API endpoint and collector logic.

Closes #123
```

```
fix(migration): prevent race condition in status updates

Use mutex to protect concurrent status updates during migration.
Fixes intermittent status update failures.

Fixes #456
```

### Pull Request Process

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Make your changes** and commit incrementally:
   ```bash
   git add .
   git commit -m "feat(scope): description"
   ```

3. **Run all checks** before pushing:
   ```bash
   make all  # Runs lint, test, and build
   ```

4. **Push your branch**:
   ```bash
   git push origin feature/my-new-feature
   ```

5. **Create a Pull Request** with:
   - **Clear title** summarizing the change
   - **Description** explaining what and why
   - **Testing** notes (how you tested)
   - **Screenshots** for UI changes
   - **Breaking changes** clearly marked

**PR Template:**

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
How this was tested

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] All tests passing
- [ ] Linters passing
- [ ] No breaking changes (or documented)
```

### Code Review

- **Be responsive** to feedback
- **Be respectful** and constructive
- **Explain your reasoning** for decisions
- **Keep PRs focused** - one feature or fix per PR
- **Request reviews** from appropriate team members

---

## Architecture Overview

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                         Web Browser                          │
│                    (React Frontend - port 3000)              │
└──────────────────────────┬───────────────────────────────────┘
                           │ HTTP/REST
┌──────────────────────────▼───────────────────────────────────┐
│                      API Server (Go)                          │
│                                                               │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   API       │  │  Discovery   │  │  Migration   │       │
│  │  Handlers   │  │   Engine     │  │   Engine     │       │
│  └─────────────┘  └──────────────┘  └──────────────┘       │
│                                                               │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Batch     │  │   Analytics  │  │   Storage    │       │
│  │ Orchestrator│  │              │  │   (SQLite/   │       │
│  │             │  │              │  │  PostgreSQL) │       │
│  └─────────────┘  └──────────────┘  └──────────────┘       │
│                                                               │
│  ┌──────────────────────────────────────────────────┐       │
│  │           GitHub Dual Client                     │       │
│  │    (PAT for migrations, App for discovery)       │       │
│  └──────────────────────────────────────────────────┘       │
└──────────────────────────┬───────────────────────────────────┘
                           │ HTTPS
            ┌──────────────┴──────────────┐
            │                             │
┌───────────▼──────────┐      ┌───────────▼──────────┐
│  GitHub Enterprise   │      │  GitHub Enterprise   │
│      Server          │      │       Cloud          │
│     (Source)         │      │    (Destination)     │
└──────────────────────┘      └──────────────────────┘
```

### Key Components

#### 1. API Layer (`internal/api`)
- HTTP handlers for all endpoints
- Request validation and response formatting
- Middleware (CORS, logging, etc.)
- Error handling

#### 2. Discovery System (`internal/discovery`)
- **Collector**: Discovers repositories from source systems
- **Analyzer**: Analyzes repository metadata
- **Profiler**: Deep profiling of Git properties
- Parallel processing with worker pools

#### 3. Migration Engine (`internal/migration`)
- **Executor**: Orchestrates migration workflow
- State machine for phase transitions
- GitHub Migrations API integration
- Lock management and rollback support

#### 4. Batch Management (`internal/batch`)
- **Orchestrator**: Manages batch operations
- **Scheduler**: Schedules and prioritizes migrations
- Status tracking and reporting

#### 5. Storage Layer (`internal/storage`)
- Database abstraction
- Repository pattern
- Migration management
- Support for SQLite and PostgreSQL

#### 6. GitHub Integration (`internal/github`)
- **Dual Client**: PAT + GitHub App support
- **Rate Limiter**: Intelligent rate limiting
- **Retry Logic**: Exponential backoff
- REST and GraphQL API clients

### Data Flow

**Discovery Flow:**
```
User → API Handler → Collector → GitHub API → Analyzer → Profiler → Storage
```

**Migration Flow:**
```
User → API Handler → Executor → GitHub Migrations API → Status Poller → Storage → User
```

### Important Design Decisions

1. **Dual Authentication**: Supports both PAT and GitHub App tokens. PAT required for migrations (GitHub limitation), App optional for non-migration operations.

2. **SQLite for Development**: Simple, no external dependencies. PostgreSQL recommended for production.

3. **Embedded Binaries**: git-sizer binaries embedded in the application for portability.

4. **Background Workers**: Polling-based workers for migration status updates (no webhooks yet).

5. **Frontend Served by Backend**: Single deployment artifact, frontend served from `/` after build.

---

## Debugging Tips

### Backend Debugging

#### Enable Debug Logging

```yaml
# configs/config.yaml
logging:
  level: debug
  format: text  # More readable than JSON for development
```

#### Use Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug cmd/server/main.go

# Or attach to running process
dlv attach <pid>
```

In VS Code, use `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server/main.go",
      "env": {
        "GITHUB_SOURCE_TOKEN": "your_token",
        "GITHUB_DEST_TOKEN": "your_token"
      }
    }
  ]
}
```

#### Common Issues

**Issue: "database is locked"**
- SQLite doesn't support concurrent writes
- Ensure only one server instance is running
- For concurrent testing, use PostgreSQL

**Issue: "rate limit exceeded"**
- Check token validity and scopes
- Server automatically waits for rate limit reset
- Monitor rate limits: `curl http://localhost:8080/api/v1/analytics/summary`

**Issue: "migrations stuck"**
- Check logs: `tail -f logs/migrator.log`
- Inspect database: `sqlite3 data/migrator.db "SELECT * FROM repositories WHERE status='migrating'"`
- Check GitHub migration status directly via API

### Frontend Debugging

#### Browser DevTools

- Open React DevTools
- Use Network tab to inspect API calls
- Check Console for errors

#### Debug API Calls

```typescript
// In api.ts, add logging
export const getRepositories = async () => {
  console.log('Fetching repositories...');
  const response = await fetch('/api/v1/repositories');
  console.log('Response:', response);
  return response.json();
};
```

#### React Query DevTools

Already integrated - open React Query DevTools in browser to inspect:
- Query states
- Cache contents
- Refetch behavior

### Database Inspection

```bash
# SQLite
sqlite3 data/migrator.db

# Useful queries
SELECT COUNT(*) FROM repositories;
SELECT status, COUNT(*) FROM repositories GROUP BY status;
SELECT * FROM migration_history WHERE repository_id = 123 ORDER BY started_at DESC;
SELECT * FROM repositories WHERE status = 'migration_failed';

# Check migration logs
SELECT * FROM migration_logs WHERE repository_id = 123 ORDER BY timestamp DESC LIMIT 50;
```

### Network Debugging

```bash
# Test API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/repositories

# With verbose output
curl -v http://localhost:8080/api/v1/repositories

# Test with authentication (if added)
curl -H "Authorization: Bearer token" http://localhost:8080/api/v1/repositories
```

---

## Available Make Targets

Detailed explanation of all make commands:

### Setup Commands

```bash
make install                # Install all dependencies (Go + Node.js) and tools
                           # Runs: install-dependencies + install-tools + web-install

make install-dependencies  # Install Go module dependencies only

make install-tools         # Install development tools (golangci-lint, gosec, git-sizer)

make web-install          # Install frontend npm packages

make download-binaries    # Download git-sizer binaries for embedding
```

### Build Commands

```bash
make build                # Build backend only (includes download-binaries)
                         # Output: bin/github-migrator-server

make web-build           # Build frontend only
                         # Output: web/dist/

make build-all           # Build both backend and frontend
```

### Test Commands

```bash
make test                # Run all backend tests with race detector

make test-coverage       # Run tests and generate HTML coverage report
                         # Output: coverage.html
```

### Lint Commands

```bash
make lint                # Run golangci-lint and gosec on backend

make web-lint            # Run ESLint on frontend

make lint-all            # Run all linters (backend + frontend)

make fmt                 # Format Go code with gofmt
```

### Run Commands

```bash
make run-server          # Run backend server locally
                         # Starts HTTP server on port 8080

make web-dev             # Run frontend dev server
                         # Starts Vite dev server on port 3000
```

### Docker Commands

```bash
make docker-build        # Build Docker image (github-migrator:latest)

make docker-run          # Start containers with docker-compose
                         # Runs: docker-compose up

make docker-down         # Stop and remove containers
                         # Runs: docker-compose down
```

### Utility Commands

```bash
make clean               # Remove build artifacts
                         # Deletes: bin/, coverage files, web/dist/, web/node_modules

make all                 # Run all checks and build
                         # Runs: lint + test + build + web-build

make help                # Show all available commands with descriptions
```

### Testing Commands

```bash
make create-test-repos ORG=your-org      # Create test repositories in GitHub org

make cleanup-test-repos ORG=your-org     # Delete test repositories from GitHub org
```

### CI/CD Integration

These commands are designed for CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Setup
  run: make install

- name: Lint
  run: make lint-all

- name: Test
  run: make test-coverage

- name: Build
  run: make build-all

- name: Docker Build
  run: make docker-build
```

---

## Project Conventions

### Error Messages

- Be specific and actionable
- Include context (repository name, IDs, etc.)
- Use proper error wrapping (`%w`)

```go
// Good
return fmt.Errorf("failed to migrate repository %s (ID: %d): %w", repo.FullName, repo.ID, err)

// Bad
return fmt.Errorf("migration failed: %w", err)
```

### Constants

- Define constants for magic values
- Group related constants

```go
const (
    StatusPending    = "pending"
    StatusMigrating  = "migrating"
    StatusCompleted  = "completed"
    StatusFailed     = "failed"
)

const (
    DefaultWorkers = 5
    DefaultTimeout = 30 * time.Minute
    MaxRetries     = 3
)
```

### Context Usage

- Always pass `context.Context` as first parameter
- Use context for cancellation and timeouts

```go
func DoWork(ctx context.Context, repo *Repository) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    // Use ctx in operations
}
```

### Dependencies

- Keep dependencies minimal
- Prefer standard library when possible
- Document why third-party dependencies are needed

---

## Questions?

- **Architecture questions**: See [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)
- **Deployment questions**: See [DEPLOYMENT.md](./DEPLOYMENT.md)
- **API questions**: See [API.md](./API.md)
- **Operations questions**: See [OPERATIONS.md](./OPERATIONS.md)

For other questions, open an issue or reach out to the maintainers.

---

**Contributing Guide Version**: 1.0.0  
**Last Updated**: October 2025

---

**Thank you for contributing to the GitHub Migration Server!** 🎉

