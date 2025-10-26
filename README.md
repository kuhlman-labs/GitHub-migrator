# GitHub Migration Server

A comprehensive, production-ready solution for migrating repositories from various source control systems (GitHub Enterprise Server, Azure DevOps, GitLab, Bitbucket) to GitHub Enterprise Cloud.

## Overview

The GitHub Migration Server provides an automated, scalable platform for managing large-scale repository migrations. It features intelligent discovery, batch organization, detailed profiling, and comprehensive migration execution with a modern web dashboard for monitoring and control.

### Key Capabilities

- **Automated Discovery**: Scan entire organizations or enterprises to identify and profile repositories
- **Intelligent Profiling**: Analyze repository characteristics including size, Git LFS usage, submodules, large files, GitHub features (Actions, Pages, Wikis), branch protections, and more
- **Flexible Migration Workflows**: Support for single repository, batch, bulk, and self-service migrations
- **Batch Management**: Organize repositories into pilot groups and migration waves for controlled rollouts
- **Real-time Monitoring**: Track migration progress with detailed status updates through all phases
- **Modern Web Dashboard**: Professional UI for visualization, analytics, and migration control
- **Full REST API**: Complete programmatic access for automation and CI/CD integration
- **Dual Authentication**: Support for both Personal Access Tokens (PAT) and GitHub Apps
- **Comprehensive Analytics**: Track metrics, completion rates, duration statistics, and migration trends

### Why Use This Tool?

Migrating repositories at scale is complex. This tool solves common challenges:

- **Visibility**: Know exactly what you're migrating before you start
- **Control**: Organize migrations into manageable batches and waves
- **Safety**: Dry-run testing and detailed validation before actual migration
- **Efficiency**: Parallel processing and intelligent rate limiting
- **Tracking**: Complete audit trail and migration history
- **Self-Service**: Enable developers to migrate their own repositories
- **Recovery**: Built-in rollback and retry mechanisms

## Features

### Repository Discovery & Profiling

- Discover repositories from organizations or entire enterprises
- Profile Git properties: size, LFS usage, submodules, large files (>100MB), commit count, branch count
- Identify GitHub features: Actions workflows, Wikis, Pages, Discussions, Projects
- Detect advanced settings: branch protections, environments, secrets, webhooks, rulesets
- Calculate complexity scores for migration planning
- Track contributors, issues, pull requests, and tags

### Migration Execution

- **Single Repository**: Migrate individual repositories on-demand
- **Batch Migration**: Group repositories and migrate entire batches together
- **Bulk Migration**: Select multiple repositories and migrate simultaneously
- **Self-Service**: Enable developers to migrate their repositories via UI or API
- **Dry Run**: Test migrations without actual execution
- **Phase Tracking**: Monitor progress through all migration phases (pending → pre-migration → migration → post-migration → complete)
- **Lock Management**: Automatically lock source repositories during migration
- **Rollback Support**: Revert completed migrations if needed

### Dashboard & Analytics

- Real-time migration status visualization
- Repository grid with advanced filtering and search
- Detailed repository views with complete migration history
- Analytics charts showing progress trends and completion stats
- Complexity and size distribution analysis
- Organization-level statistics
- Migration velocity tracking and ETA calculations
- Export capabilities for reporting

### Batch Management

- Create and manage migration batches
- Assign repositories to batches
- Set priorities for migration ordering
- Schedule batch migrations
- Retry failed migrations
- Track batch-level progress and statistics

### API & Automation

- Complete REST API for all operations
- Programmatic migration triggering
- Status polling and monitoring
- Integration with CI/CD pipelines
- Webhook support (planned)
- Export data in CSV or JSON formats

## Quick Start

### Prerequisites

- **Go** 1.21 or higher
- **Node.js** 20 or higher
- **Git** 2.30+ with Git LFS support
- **Docker** (optional, for containerized deployment)

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/github-migrator.git
cd github-migrator
```

### 2. Install Dependencies

```bash
# Install all dependencies (Go + Node.js) and development tools
make install
```

This command will:
- Download Go module dependencies
- Install `golangci-lint`, `gosec`, and `git-sizer`
- Install frontend npm packages

### 3. Configure the Application

Create a configuration file at `configs/config.yaml`:

```yaml
server:
  port: 8080

database:
  type: sqlite
  dsn: ./data/migrator.db

source:
  type: github
  base_url: "https://github.company.com/api/v3"  # Your GitHub Enterprise Server
  token: "${GITHUB_SOURCE_TOKEN}"

destination:
  type: github
  base_url: "https://api.github.com"  # GitHub Enterprise Cloud or GitHub.com
  token: "${GITHUB_DEST_TOKEN}"

migration:
  workers: 10
  poll_interval_seconds: 30

logging:
  level: info
  format: json
  output_file: ./logs/migrator.log
```

Set your GitHub tokens as environment variables:

```bash
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"  # Source system token
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"    # Destination system token
```

**Token Requirements:**
- **Source Token**: Organization admin access with `repo`, `read:org`, `read:user`, `admin:org` scopes
- **Destination Token**: Organization admin access with `repo`, `admin:org`, `workflow` scopes

### 4. Build the Application

```bash
# Build backend and frontend
make build-all
```

### 5. Run Locally

**Option A: Run in separate terminals**

Terminal 1 - Backend:
```bash
make run-server
```

Terminal 2 - Frontend:
```bash
make web-dev
```

- Backend API: http://localhost:8080
- Frontend UI: http://localhost:3000

**Option B: Run with Docker**

```bash
# Build and run with Docker Compose
make docker-build
make docker-run

# Access at http://localhost:8080
```

### 6. Start Your First Migration

1. **Open the dashboard** at http://localhost:3000 (or http://localhost:8080 with Docker)

2. **Discover repositories**:
   - Navigate to the Dashboard
   - Click "Start Discovery"
   - Enter your organization name
   - Wait for discovery to complete

3. **Create a pilot batch**:
   - Go to "Batch Management"
   - Create a new batch (e.g., "Pilot - Wave 1")
   - Select 3-5 simple repositories
   - Click "Start Batch Migration"

4. **Monitor progress**:
   - Watch real-time status updates
   - View detailed logs for each repository
   - Check analytics for overall progress

## Documentation

### Getting Started
- **[Quick Start](#quick-start)** - Get up and running quickly (above)
- **[CONTRIBUTING.md](./docs/CONTRIBUTING.md)** - Contributor guide with development setup, coding standards, and debugging tips

### Operations
- **[DEPLOYMENT.md](./docs/DEPLOYMENT.md)** - Production deployment guide for Docker, Kubernetes, and manual deployment
- **[OPERATIONS.md](./docs/OPERATIONS.md)** - Operations runbook with daily checklists, monitoring, and incident response
- **[API.md](./docs/API.md)** - Complete REST API documentation with examples

### Technical Details
- **[IMPLEMENTATION_GUIDE.md](./docs/IMPLEMENTATION_GUIDE.md)** - Deep dive into architecture, components, and implementation details
- **[OpenAPI Specification](./docs/openapi.json)** - Machine-readable API specification

### Configuration
- [config.yaml](./configs/config.yaml) - Default configuration
- [production.yaml](./configs/production.yaml) - Production settings
- [development.yaml](./configs/development.yaml) - Development settings
- [docker.yaml](./configs/docker.yaml) - Docker-specific configuration
- [env.example](./configs/env.example) - Environment variables template

## Tech Stack

### Backend
- **Go 1.21+**: Core backend implementation
- **SQLite/PostgreSQL**: Data storage (SQLite for development, PostgreSQL for production)
- **GitHub APIs**: `google/go-github/v75` (REST) and `shurcooL/githubv4` (GraphQL)
- **Viper**: Configuration management
- **Lumberjack**: Log rotation

### Frontend
- **React 18**: UI framework
- **TypeScript**: Type-safe JavaScript
- **Vite**: Fast build tooling
- **Tailwind CSS**: Utility-first styling
- **React Router**: Navigation
- **Recharts**: Analytics visualizations
- **TanStack Query**: Data fetching and caching

### DevOps
- **Docker**: Containerized deployment
- **Docker Compose**: Multi-container orchestration
- **golangci-lint**: Go code linting
- **gosec**: Security scanning
- **Makefile**: Build automation

## Project Structure

```
github-migrator/
├── cmd/
│   ├── server/             # HTTP server entry point
│   └── cli/                # CLI commands (future)
├── internal/
│   ├── api/                # HTTP API handlers and middleware
│   ├── analytics/          # Analytics and reporting
│   ├── batch/              # Batch management and orchestration
│   ├── config/             # Configuration loading
│   ├── discovery/          # Repository discovery and profiling
│   ├── embedded/           # Embedded binaries (git-sizer)
│   ├── github/             # GitHub API clients (REST + GraphQL)
│   ├── logging/            # Logging setup
│   ├── migration/          # Migration execution engine
│   ├── models/             # Data models
│   ├── source/             # Source provider abstraction
│   ├── storage/            # Database layer
│   └── worker/             # Background worker scheduler
├── web/
│   ├── src/
│   │   ├── components/     # React components
│   │   ├── hooks/          # Custom React hooks
│   │   ├── services/       # API services
│   │   └── types/          # TypeScript types
│   └── dist/               # Built frontend (served by backend)
├── configs/                # Configuration files
├── docs/                   # Documentation
├── scripts/                # Utility scripts
├── Makefile                # Build automation
├── Dockerfile              # Container definition
└── docker-compose.yml      # Container orchestration
```

## Development

### Available Commands

```bash
make help                   # Show all available commands
make install                # Install all dependencies and tools
make build                  # Build backend only
make web-build              # Build frontend only
make build-all              # Build both backend and frontend
make test                   # Run backend tests
make test-coverage          # Generate coverage report
make lint                   # Lint backend code
make web-lint               # Lint frontend code
make lint-all               # Lint everything
make fmt                    # Format Go code
make run-server             # Run backend server
make web-dev                # Run frontend dev server
make docker-build           # Build Docker image
make docker-run             # Run with Docker Compose
make docker-down            # Stop Docker containers
make clean                  # Clean build artifacts
make all                    # Run all checks and build
```

### Running Tests

```bash
# Backend tests
make test

# With coverage report
make test-coverage

# Frontend tests
cd web && npm test
```

### Code Quality

```bash
# Lint backend
make lint

# Lint frontend
make web-lint

# Format code
make fmt

# Run everything
make all
```

## Migration Workflows

### 1. Single Repository Migration

- Navigate to repository detail page
- Click "Start Migration" or "Dry Run"
- Monitor real-time status updates
- View detailed logs

### 2. Batch Migration (Recommended for Pilots)

- Create a batch (e.g., "Pilot Repositories")
- Assign 5-10 repositories to the batch
- Start with dry run to validate
- Execute actual migration
- Perfect for organized, phased rollouts

### 3. Bulk Migration

- Select multiple repositories from dashboard using checkboxes
- Use "Bulk Migrate" or "Bulk Dry Run"
- Ideal for migrating groups of similar repositories

### 4. Self-Service Migration

- Developers access self-service page
- Enter repository names (`org/repo` format)
- Submit for migration
- Track progress in dashboard

### 5. Programmatic Migration

```bash
# Single repository migration
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{"repository_ids": [123], "dry_run": false}'

# Self-service by repository name
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{"full_names": ["org/repo"], "dry_run": false}'

# Batch migration
curl -X POST http://localhost:8080/api/v1/batches/5/start
```

## Security

- **Token Management**: Tokens stored in environment variables or secret managers
- **No Credentials in Code**: All sensitive data configured externally
- **Parameterized Queries**: SQL injection prevention
- **Input Validation**: Request validation and sanitization
- **Security Scanning**: Regular `gosec` scans
- **Dependency Updates**: Automated dependency checks
- **Container Security**: Non-root user in containers

## Deployment

### Quick Deploy with Docker

```bash
# 1. Set environment variables
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"

# 2. Build and run
make docker-build
make docker-run

# 3. Access at http://localhost:8080
```

### Production Deployment

See **[DEPLOYMENT.md](./docs/DEPLOYMENT.md)** for comprehensive deployment instructions including:
- Docker and Docker Compose setup
- Kubernetes deployment with manifests
- PostgreSQL configuration
- Security hardening
- Monitoring and alerting
- Backup and recovery procedures

## Contributing

We welcome contributions! Please see our **[Contributing Guide](./docs/CONTRIBUTING.md)** for:

- Development environment setup
- Coding standards and conventions
- Testing requirements
- Pull request process
- Architecture overview
- Debugging tips

### Quick Contribution Steps

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linters (`make all`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to your branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Support & Resources

### Documentation
- [API Documentation](./docs/API.md) - Complete API reference
- [Operations Guide](./docs/OPERATIONS.md) - Day-to-day operations
- [Implementation Guide](./docs/IMPLEMENTATION_GUIDE.md) - Technical deep dive
- [Deployment Guide](./docs/DEPLOYMENT.md) - Production deployment

### GitHub Resources
- [GitHub Migrations API](https://docs.github.com/en/rest/migrations)
- [GitHub App Authentication](https://docs.github.com/en/apps)
- [GitHub Rate Limiting](https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting)

## License

[Add your license here]

## Acknowledgments

- Built with [go-github](https://github.com/google/go-github) for GitHub API integration
- Uses [git-sizer](https://github.com/github/git-sizer) for repository analysis
- Powered by Go, React, and modern open-source tools

---

**Version**: 1.0.0  
**Last Updated**: October 2025  
**Status**: Production Ready  
**Maintained by**: @kuhlman-labs
