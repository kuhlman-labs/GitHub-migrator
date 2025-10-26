# GitHub Migrator

A comprehensive, production-ready solution for migrating repositories from GitHub Enterprise Server or GitHub.com to GitHub.com, GitHub with data residency, or GitHub Enterprise Cloud with EMU.

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![Node Version](https://img.shields.io/badge/node-20+-green.svg)](https://nodejs.org/)

---

## Table of Contents

- [Overview](#overview)
  - [Supported Migrations](#supported-migrations)
  - [Key Capabilities](#key-capabilities)
  - [Why Use This Tool?](#why-use-this-tool)
- [Quick Start](#quick-start)
  - [Prerequisites](#prerequisites)
  - [Installation & Setup](#installation--setup)
  - [First Migration](#first-migration)
- [Features](#features)
- [Documentation](#documentation)
- [Development](#development)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [Tech Stack](#tech-stack)
- [License](#license)

---

## Overview

The GitHub Migration Server provides an automated, scalable platform for managing large-scale GitHub-to-GitHub repository migrations. It features intelligent discovery, batch organization, detailed profiling, and comprehensive migration execution with a modern web dashboard for monitoring and control.

### Supported Migrations

**Source:**
- GitHub.com
- GitHub Enterprise Server

**Destination:**
- GitHub.com
- GitHub with data residency
- GitHub Enterprise Cloud with EMU (Managed Users)

> **Note**: Only GitHub to GitHub migrations are currently supported with plans to add support for more sources.

### Key Capabilities

- 🔍 **Automated Discovery** - Scan entire organizations or enterprises to identify and profile repositories
- 📊 **Intelligent Profiling** - Analyze size, LFS usage, submodules, large files, Actions, branch protections, and more
- 🎯 **Flexible Workflows** - Single repository, batch, bulk, and self-service migration options
- 📦 **Batch Management** - Organize into pilot groups and migration waves for controlled rollouts
- 📈 **Real-time Monitoring** - Track progress with detailed status updates through all migration phases
- 🎨 **Modern Dashboard** - Professional web UI for visualization, analytics, and control
- 🔌 **Complete REST API** - Full programmatic access for automation and CI/CD integration
- 🔐 **Dual Authentication** - PAT required for migrations; optional GitHub App for discovery with better rate limits
- 📉 **Comprehensive Analytics** - Track metrics, completion rates, duration stats, and trends

### Why Use This Tool?

Migrating repositories at scale is complex. This tool provides:

| Challenge | Solution |
|-----------|----------|
| **Visibility** | Know exactly what you're migrating before you start |
| **Control** | Organize migrations into manageable batches and waves |
| **Safety** | Dry-run testing and detailed validation before migration |
| **Efficiency** | Parallel processing with intelligent rate limiting |
| **Tracking** | Complete audit trail and migration history |
| **Self-Service** | Enable developers to migrate their own repositories |
| **Recovery** | Built-in rollback and retry mechanisms |

---

## Quick Start

### Prerequisites

- **Go** 1.21 or higher
- **Node.js** 20 or higher  
- **Git** 2.30+ with Git LFS support
- **Docker** (optional, for containerized deployment)

### Installation & Setup

**1. Clone and install dependencies:**

```bash
git clone https://github.com/kuhlman-labs/GitHub-migrator.git
cd GitHub-migrator
make install  # Installs Go dependencies, golangci-lint, gosec, git-sizer, and npm packages
```

**2. Configure authentication:**

```bash
# Set environment variables
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"  # Source system token
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"    # Destination system token
```

**Token Requirements:**
- **PAT (Personal Access Token) - REQUIRED for migrations**
  - **Source**: Org admin access with `repo`, `read:user`, `admin:org` scopes
  - **Destination**: Org admin access with `repo`, `admin:org`, `workflow` scopes
- **GitHub App - OPTIONAL for discovery** (provides better rate limits)

See [config_example.yml](./configs/config_example.yml) for complete configuration options.

**3. Run the application:**

```bash
# Option A: Development mode (separate terminals)
make run-server  # Terminal 1 - Backend at http://localhost:8080
make web-dev     # Terminal 2 - Frontend at http://localhost:3000

# Option B: Docker
make docker-build && make docker-run  # Access at http://localhost:8080
```

### First Migration

1. **Open the dashboard** at http://localhost:3000 (or :8080 with Docker)
2. **Discover repositories** - Enter your organization name and start discovery
3. **Create a pilot batch** - Select 3-5 simple repositories for testing
4. **Run dry run** - Test without actual migration
5. **Execute migration** - Start the actual migration
6. **Monitor progress** - Watch real-time status and detailed logs

For detailed workflows, see [OPERATIONS.md](./docs/OPERATIONS.md#migration-workflows).

---

## Features

<details>
<summary><strong>🔍 Repository Discovery & Profiling</strong></summary>

- Discover repositories from organizations or entire enterprises
- Profile Git properties: size, LFS usage, submodules, large files (>100MB), commits, branches
- Identify GitHub features: Actions, Wikis, Pages, Discussions, Projects, Environments
- Detect advanced settings: branch protections, secrets, webhooks, rulesets, packages
- Calculate complexity scores for migration planning
- Track contributors, issues, pull requests, and tags

</details>

<details>
<summary><strong>🚀 Migration Execution</strong></summary>

- **Single Repository** - On-demand individual repository migrations
- **Batch Migration** - Group repositories and migrate entire batches together  
- **Bulk Migration** - Select multiple repositories and migrate simultaneously
- **Self-Service** - Enable developers to migrate their own repositories via UI or API
- **Dry Run** - Test migrations without actual execution
- **Phase Tracking** - Monitor through all phases: pending → pre-migration → migration → post-migration → complete
- **Lock Management** - Automatically lock source repositories during migration
- **Rollback Support** - Revert completed migrations if validation fails

</details>

<details>
<summary><strong>📊 Dashboard & Analytics</strong></summary>

- Real-time migration status visualization
- Repository grid with advanced filtering and search
- Detailed repository views with complete migration history
- Analytics charts showing progress trends and completion statistics
- Complexity and size distribution analysis
- Organization-level statistics and reporting
- Migration velocity tracking and ETA calculations
- Export capabilities for reporting (CSV, JSON)

</details>

<details>
<summary><strong>📦 Batch Management</strong></summary>

- Create and manage migration batches
- Assign repositories to batches with priority ordering
- Schedule batch migrations for planned rollouts
- Retry failed migrations automatically or manually
- Track batch-level progress and statistics
- Support for pilot, wave, and self-service batch types

</details>

<details>
<summary><strong>🔌 API & Automation</strong></summary>

- Complete REST API for all operations
- Programmatic migration triggering and status monitoring
- Integration with CI/CD pipelines
- Export data in CSV or JSON formats
- Comprehensive OpenAPI 3.0 specification
- See [API.md](./docs/API.md) for full documentation

</details>

---

## Documentation

### 📚 Getting Started
- **[Quick Start](#quick-start)** - Get up and running in minutes (above)
- **[CONTRIBUTING.md](./docs/CONTRIBUTING.md)** - Development setup, coding standards, testing, and debugging

### 🚀 Operations & Deployment
- **[DEPLOYMENT.md](./docs/DEPLOYMENT.md)** - Production deployment for Docker, Kubernetes, and manual installation
- **[OPERATIONS.md](./docs/OPERATIONS.md)** - Daily operations, monitoring, incident response, and troubleshooting
- **[API.md](./docs/API.md)** - Complete REST API reference with examples

### 🔧 Technical Reference
- **[IMPLEMENTATION_GUIDE.md](./docs/IMPLEMENTATION_GUIDE.md)** - Architecture, components, and implementation details
- **[OpenAPI Specification](./docs/openapi.json)** - Machine-readable API specification

### ⚙️ Configuration
- **[config_example.yml](./configs/config_example.yml)** - Complete configuration reference (start here)
- [env.example](./configs/env.example) - Environment variables template
- [config.yaml](./configs/config.yaml) - Default configuration
- [production.yaml](./configs/production.yaml) - Production settings
- [development.yaml](./configs/development.yaml) - Development settings
- [docker.yaml](./configs/docker.yaml) - Docker-specific configuration

---

## Development

### Quick Reference

```bash
make help          # Show all available commands
make install       # Install dependencies and tools
make build-all     # Build backend and frontend
make test          # Run backend tests
make lint-all      # Lint backend and frontend
make run-server    # Run backend (http://localhost:8080)
make web-dev       # Run frontend dev server (http://localhost:3000)
```

### Project Structure

```
github-migrator/
├── cmd/server/            # HTTP server entry point
├── internal/              # Backend Go packages
│   ├── api/              # HTTP handlers and middleware
│   ├── discovery/        # Repository discovery and profiling
│   ├── migration/        # Migration execution engine
│   ├── batch/            # Batch orchestration
│   ├── storage/          # Database layer
│   └── github/           # GitHub API clients
├── web/src/              # React frontend
│   ├── components/       # UI components
│   ├── hooks/            # Custom React hooks
│   └── services/         # API client
├── configs/              # Configuration files
└── docs/                 # Documentation
```

For comprehensive development information, see **[CONTRIBUTING.md](./docs/CONTRIBUTING.md)**.

---

## Deployment

### Docker Quick Deploy

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

For production deployments, see **[DEPLOYMENT.md](./docs/DEPLOYMENT.md)** which covers:

- 🐳 Docker and Docker Compose configuration
- ☸️ Kubernetes deployment with manifests  
- 🗄️ PostgreSQL database setup and optimization
- 🔒 Security hardening and best practices
- 📊 Monitoring and alerting setup
- 💾 Backup and recovery procedures
- 🔧 Troubleshooting common issues

---

## Contributing

We welcome contributions! Please see **[CONTRIBUTING.md](./docs/CONTRIBUTING.md)** for:

- Development environment setup
- Coding standards and conventions
- Testing requirements and coverage
- Pull request process
- Architecture overview
- Debugging tips and techniques

### Quick Contribution Steps

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and add tests
4. Run tests and linters (`make all`)
5. Commit your changes with a descriptive message
6. Push to your branch
7. Open a Pull Request

---

## Tech Stack

### Backend
- **Go 1.21+** - Core backend implementation
- **SQLite/PostgreSQL** - Data storage (SQLite for dev, PostgreSQL for production)
- **GitHub APIs** - `google/go-github/v75` (REST) and `shurcooL/githubv4` (GraphQL)
- **Viper** - Configuration management
- **Lumberjack** - Log rotation

### Frontend
- **React 18** - UI framework with TypeScript
- **Vite** - Fast build tooling
- **Tailwind CSS** - Utility-first styling
- **TanStack Query** - Data fetching and caching
- **Recharts** - Analytics visualizations

### DevOps
- **Docker** - Containerized deployment
- **golangci-lint** - Go code linting
- **gosec** - Security scanning

---

## Support & Resources

### Internal Documentation
- [API Documentation](./docs/API.md) - Complete API reference
- [Operations Guide](./docs/OPERATIONS.md) - Day-to-day operations
- [Implementation Guide](./docs/IMPLEMENTATION_GUIDE.md) - Technical deep dive
- [Deployment Guide](./docs/DEPLOYMENT.md) - Production deployment

### External Resources
- [GitHub Migrations API](https://docs.github.com/en/rest/migrations) - Official API documentation
- [GitHub App Authentication](https://docs.github.com/en/apps) - App authentication guide
- [GitHub Rate Limiting](https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting) - Rate limit information

---

## Acknowledgments

- Built with [go-github](https://github.com/google/go-github) for GitHub API integration
- Uses [git-sizer](https://github.com/github/git-sizer) for repository analysis
- Powered by Go, React, and modern open-source tools

---

<div align="center">

**Version**: 1.0.0  
**Last Updated**: October 2025  
**Status**: Production Ready  
**Maintained by**: [@kuhlman-labs](https://github.com/kuhlman-labs)

</div>
