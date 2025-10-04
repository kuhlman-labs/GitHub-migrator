# GitHub Migration Server

A comprehensive solution for migrating repositories from multiple sources (GitHub Enterprise Server, Azure DevOps, GitLab, BitBucket) to GitHub Enterprise Cloud.

## Overview

This project provides an automated migration server with discovery, profiling, batch management, and migration execution capabilities. It features a modern web dashboard for tracking migration progress, analytics, and detailed repository information.

## Features

- **Repository Discovery**: Automatically discover and profile repositories from source systems
- **Comprehensive Profiling**: Gather git properties (size, LFS, submodules) and GitHub features (actions, wikis, pages, protections, etc.)
- **Batch Management**: Organize repositories into pilot groups and migration waves
- **Migration Execution**: Execute migrations with detailed status tracking through all phases
- **Migration Control**: Trigger migrations via UI or API (single, batch, bulk, or self-service)
- **Self-Service**: Developers can migrate their own repositories with a simple interface
- **Dashboard**: Real-time monitoring of migration progress with professional UI
- **Analytics**: Track migration metrics, completion rates, and average durations
- **Detailed Reporting**: Per-repository views with complete history and statistics
- **Programmatic Access**: Full REST API for automation and integration

## Tech Stack

### Backend
- **Go**: Core backend implementation
- **GitHub APIs**: `google/go-github` (REST) and `shurcooL/githubv4` (GraphQL)
- **Configuration**: Viper + Cobra for config and CLI
- **Logging**: Structured logging with rotation and colorized output
- **Database**: SQLite (MVP) with PostgreSQL support

### Frontend
- **React 18+ with TypeScript**
- **Vite**: Fast build tooling
- **Tailwind CSS**: Minimal, Apple-like design
- **React Router**: Navigation
- **Recharts**: Analytics visualizations

### DevOps
- **Docker**: Containerized deployment
- **Makefile**: Build, test, lint automation
- **golangci-lint**: Comprehensive Go linting
- **gosec**: Security scanning

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 20+
- Docker (optional)

### Build

```bash
# Install tools and dependencies
make install-tools

# Build backend and frontend
make build

# Run tests
make test

# Run linters
make lint
```

### Configuration

Create `configs/config.yaml`:

```yaml
server:
  port: 8080

database:
  type: sqlite
  dsn: ./data/migrator.db

github:
  source:
    base_url: "https://github.company.com/api/v3"
    token: "${GITHUB_SOURCE_TOKEN}"
  destination:
    base_url: "https://api.github.com"
    token: "${GITHUB_DEST_TOKEN}"

logging:
  level: info
  format: json
  output_file: ./logs/migrator.log
```

### Run Locally

```bash
# Run backend server
make run-server

# Or run both backend and frontend in dev mode
make run-dev
```

### Docker Deployment

```bash
# Build and run with Docker Compose
make docker-build
make docker-run

# Access at http://localhost:8080
```

## Project Structure

```
github-migrator/
├── cmd/                    # Application entry points
│   ├── server/            # HTTP server
│   └── cli/               # CLI commands
├── internal/              # Private application code
│   ├── api/               # HTTP API handlers
│   ├── discovery/         # Repository discovery engine
│   ├── migration/         # Migration execution
│   ├── batch/             # Batch management
│   ├── analytics/         # Analytics and reporting
│   ├── github/            # GitHub API clients
│   ├── models/            # Data models
│   ├── storage/           # Database layer
│   ├── config/            # Configuration
│   └── logging/           # Logging setup
├── web/                   # Frontend React application
│   ├── src/
│   │   ├── components/    # React components
│   │   ├── services/      # API services
│   │   └── types/         # TypeScript types
│   └── tests/
├── configs/               # Configuration files
├── migrations/            # Database migrations
├── tests/                 # Integration tests
├── Makefile
├── Dockerfile
└── docker-compose.yml
```

## Migration Workflows

The system supports multiple migration approaches:

### 1. Single Repository Migration
- Navigate to a repository detail page
- Click "Start Migration" or "Dry Run" button
- Monitor real-time status updates

### 2. Batch Migration (Pilot/Waves)
- Create a batch (e.g., "Pilot Repositories")
- Assign repositories to the batch
- Click "Start Batch Migration" to migrate all at once
- Perfect for organized, phased rollouts

### 3. Bulk Migration
- Select multiple repositories from the dashboard using checkboxes
- Use "Bulk Migrate" or "Bulk Dry Run" from the action bar
- Ideal for migrating groups of similar repositories

### 4. Self-Service Migration
- Developers access the self-service page
- Enter repository names (org/repo format)
- Submit for migration
- Monitor progress in the dashboard

### 5. Programmatic Migration
- Use REST API endpoints
- Integrate with CI/CD pipelines
- Automate via scripts or bots
- Full control via API calls

## Migration Phases

Repositories progress through the following phases:

1. **Pending**: Discovered but not yet processed
2. **Dry Run**: Test migration validation
3. **Pre-migration**: Pre-flight checks
4. **Migration**: Active migration (archive → queue → migrating → complete)
5. **Post-migration**: Post-migration tasks
6. **Complete**: Fully migrated and validated

## Development

### Available Make Targets

```bash
make help              # Show all available commands
make build             # Build backend and frontend
make test              # Run all tests
make test-coverage     # Generate coverage report
make lint              # Run all linters
make fmt               # Format code
make run-server        # Run backend server
make run-dev           # Run backend + frontend in dev mode
make docker-build      # Build Docker image
make docker-run        # Run with Docker Compose
make clean             # Clean build artifacts
```

### Testing

```bash
# Backend tests
go test -v ./...

# Frontend tests
cd web && npm test

# Coverage report
make test-coverage
```

### Linting

```bash
# Lint backend
golangci-lint run

# Security scan
gosec ./...

# Lint frontend
cd web && npm run lint
```

## API Documentation

The server exposes a REST API at `/api/v1/`:

### Discovery
- `POST /api/v1/discovery/start` - Start repository discovery
- `GET /api/v1/discovery/status` - Get discovery status

### Repositories
- `GET /api/v1/repositories` - List repositories (supports filtering)
- `GET /api/v1/repositories/{fullName}` - Get repository details with migration history
- `PATCH /api/v1/repositories/{fullName}` - Update repository (batch assignment, priority)

### Batches
- `GET /api/v1/batches` - List all batches
- `POST /api/v1/batches` - Create migration batch
- `GET /api/v1/batches/{id}` - Get batch details
- `POST /api/v1/batches/{id}/start` - Start migration for entire batch

### Migrations
- `POST /api/v1/migrations/start` - Start migration (single or multiple repos)
- `GET /api/v1/migrations/{id}` - Get migration status
- `GET /api/v1/migrations/{id}/history` - Get complete migration history

### Analytics
- `GET /api/v1/analytics/summary` - Get analytics summary
- `GET /api/v1/analytics/progress` - Get migration progress metrics

### Migration Control Examples

**Start single repository migration:**
```bash
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{"repository_ids": [123], "dry_run": false}'
```

**Start migration by repository name (self-service):**
```bash
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{"full_names": ["org/repo1", "org/repo2"], "dry_run": false}'
```

**Start batch migration:**
```bash
curl -X POST http://localhost:8080/api/v1/batches/5/start
```

## Implementation Guide

For AI agents or detailed implementation instructions, see the comprehensive [**IMPLEMENTATION_GUIDE.md**](./IMPLEMENTATION_GUIDE.md) which includes:

- Complete architecture design
- Data models and schemas
- Detailed implementation phases
- Code examples for all components
- Testing and quality standards
- Deployment instructions
- External resource references

## Security

- Tokens managed via environment variables
- No sensitive data in version control
- Parameterized database queries
- Security scanning with gosec
- Regular dependency updates
- Container security best practices

## Contributing

1. Follow idiomatic Go and React best practices
2. Write tests for new features
3. Ensure linters pass
4. Update documentation

## License

[Add your license here]

## Support

For detailed implementation guidance, refer to [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)
