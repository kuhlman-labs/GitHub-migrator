# GitHub Migrator

A comprehensive, production-ready solution for migrating repositories from GitHub Enterprise Server or GitHub.com to GitHub.com, GitHub with data residency, or GitHub Enterprise Cloud with EMU.

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![Node Version](https://img.shields.io/badge/node-20+-green.svg)](https://nodejs.org/)

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Features](#features)
- [Documentation](#documentation)
- [Development](#development)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [Tech Stack](#tech-stack)
- [Support & Resources](#support--resources)

## Overview

GitHub Migrator provides an automated, scalable platform for managing large-scale GitHub-to-GitHub repository migrations. It features intelligent discovery, batch organization, detailed profiling, and comprehensive migration execution with a modern web dashboard for monitoring and control.

### Supported Migrations

**Source:**
- GitHub.com
- GitHub Enterprise Server

**Destination:**
- GitHub.com
- GitHub with data residency
- GitHub Enterprise Cloud with EMU (Managed Users)

> **Note**: Only GitHub to GitHub migrations are currently supported. Support for additional source platforms is planned.

### Key Capabilities

**Automated Discovery & Profiling**
- Scan entire organizations or enterprises to identify and profile repositories
- Analyze Git properties: size, LFS usage, submodules, large files, commits, branches
- Identify GitHub features: Actions, Wikis, Pages, Discussions, Projects, Environments, Releases
- Detect advanced settings: branch protections, rulesets, secrets, variables, webhooks, packages
- Calculate source-aware complexity scores for migration planning
- Track contributors, issues, pull requests, tags, and repository activity levels

**Migration Limits Detection**
- Automatically detect and flag repositories violating GitHub Enterprise Importer limits
- 40 GiB repository size limit (automatic blocking)
- 40 GiB metadata size limit (warnings with size estimates)
- 2 GiB commit limit, 255-byte ref limit, 400 MiB file limit
- Configure per-repository exclusion flags (releases, attachments, metadata) to reduce migration size

**Flexible Migration Workflows**
- Single repository migrations for on-demand execution
- Batch migration for organized group migrations
- Bulk migration for simultaneous multi-repository migrations
- Self-service capabilities for developer-initiated migrations with proper access controls
- Repository-level permission enforcement for all migration operations
- Dry-run testing without actual execution
- Phase tracking: pending → pre-migration → migration → post-migration → complete

**Management & Control**
- Batch organization with pilot groups and migration waves
- Real-time monitoring with detailed status updates
- Modern web dashboard for visualization, analytics, and control
- Complete REST API for automation and CI/CD integration
- Lock management for source repositories during migration
- Rollback support for failed migrations
- Visibility transformation handling (public/internal/private)

**Authentication & Security**
- PAT (Personal Access Token) required for migrations
- Optional GitHub App for discovery operations (improved rate limits)
- Optional GitHub OAuth 2.0 authentication for self-hosted deployments
- Two-layer authorization model: application access + repository-level permissions
- Granular access control: Enterprise Admins, Org Admins, Privileged Teams, and Repo Admins
- Repository-level permissions enforced for all migration operations
- JWT session management with secure token storage
- User activity logging with audit trail for all migration actions
- Comprehensive security logging and monitoring

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Node.js 20 or higher
- Git 2.30+ with Git LFS support
- Docker (optional, for containerized deployment)

### Installation

Clone the repository and install dependencies:

```bash
git clone https://github.com/kuhlman-labs/GitHub-migrator.git
cd GitHub-migrator
make install
```

The `make install` command installs Go dependencies, golangci-lint, gosec, git-sizer, and npm packages.

### Authentication Configuration

Configure authentication using environment variables:

```bash
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"
```

**Personal Access Tokens (Required)**

- **Source**: Organization admin access with `repo`, `read:user`, `admin:org` scopes
- **Destination**: Organization admin access with `repo`, `admin:org`, `workflow` scopes

**GitHub App (Optional)**

Recommended for better rate limits during discovery and profiling operations. PAT is still required for migration operations.

- **With Installation ID**: Simplest setup using a single installation token
- **Without Installation ID**: Auto-discovers all installations and creates per-org tokens (best for multi-org apps)

See [GitHub App Setup Guide](./docs/OPERATIONS.md#github-app-authentication) for detailed configuration.

**OAuth Authentication (Optional)**

For production deployments with user authentication and access control:

```bash
# Enable authentication
export GHMIG_AUTH_ENABLED=true
export GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID="your_oauth_client_id"
export GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET="your_oauth_client_secret"
export GHMIG_AUTH_CALLBACK_URL="https://your-domain.com/api/v1/auth/callback"
export GHMIG_AUTH_FRONTEND_URL="https://your-domain.com"
export GHMIG_AUTH_SESSION_SECRET="generate_a_random_32_char_secret"

# Configure access control (examples)
# Option 1: Enterprise members with privileged migration teams
export GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_MEMBERSHIP=true
export GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_SLUG="my-enterprise"
export GHMIG_AUTH_AUTHORIZATION_RULES_PRIVILEGED_TEAMS="platform-org/migration-admins,platform-org/devops"

# Option 2: Specific organization members only
export GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ORG_MEMBERSHIP="my-org,migration-team-org"
export GHMIG_AUTH_AUTHORIZATION_RULES_PRIVILEGED_TEAMS="my-org/migration-coordinators"

# Option 3: Enterprise admins only (most restrictive)
export GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_ADMIN=true
export GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_SLUG="my-enterprise"
```

See [OPERATIONS.md](./docs/OPERATIONS.md#authentication-setup) for complete configuration options and examples.

### Running the Application

**Development Mode** (separate terminals):

```bash
make run-server  # Backend at http://localhost:8080
make web-dev     # Frontend at http://localhost:3000
```

**Docker**:

```bash
make docker-build && make docker-run  # Access at http://localhost:8080
```

### First Migration

1. Open the dashboard at http://localhost:3000 (or :8080 with Docker)
2. Discover repositories by entering your organization name
3. Create a pilot batch with 3-5 simple repositories for testing
4. Run a dry run to test without actual migration
5. Execute the migration
6. Monitor progress through real-time status and detailed logs

For detailed workflows, see [OPERATIONS.md](./docs/OPERATIONS.md#migration-workflows).

## Features

### Repository Discovery & Profiling

- Discover repositories from organizations or entire enterprises
- Profile Git properties: size, LFS usage, submodules, large files (>100MB), commits, branches
- Identify GitHub features: Actions, Wikis, Pages, Discussions, Projects, Environments, Releases
- Detect advanced settings: branch protections, rulesets, secrets, variables, webhooks, packages
- Calculate source-aware complexity scores for migration planning based on remediation difficulty
- Track contributors, issues, pull requests, tags, and repository activity levels
- Quantile-based activity scoring adaptive to your repository dataset

### Migration Execution

- **Single Repository**: On-demand individual repository migrations
- **Batch Migration**: Group repositories and migrate entire batches together
- **Bulk Migration**: Select multiple repositories and migrate simultaneously
- **Self-Service**: Enable developers to migrate their own repositories via UI or API with granular access controls
- **Permission Enforcement**: Validate user permissions before any migration action (add to batch, dry run, migrate, retry)
- **User Activity Logging**: Track who initiated each migration with complete audit trail
- **Dry Run**: Test migrations without actual execution
- **Phase Tracking**: Monitor through all phases (pending → pre-migration → migration → post-migration → complete)
- **Lock Management**: Automatically lock source repositories during migration
- **Rollback Support**: Revert completed migrations if validation fails
- **Visibility Handling**: Configurable visibility transformations (public/internal/private) with EMU and data residency support

### Dashboard & Analytics

- Real-time migration status visualization
- Repository grid with advanced filtering and search
- Detailed repository views with complete migration history
- Analytics charts showing progress trends and completion statistics
- Complexity and size distribution analysis
- Organization-level statistics and reporting
- Migration velocity tracking and ETA calculations
- Export capabilities for reporting (CSV, JSON)

### Batch Management

- Create and manage migration batches
- Assign repositories to batches with priority ordering
- Schedule batch migrations for planned rollouts
- Retry failed migrations automatically or manually
- Track batch-level progress and statistics
- Support for pilot, wave, and self-service batch types

### API & Automation

- Complete REST API for all operations
- Programmatic migration triggering and status monitoring
- Integration with CI/CD pipelines
- Export data in CSV or JSON formats
- Comprehensive OpenAPI 3.0 specification
- See [API.md](./docs/API.md) for full documentation

### Security & Authentication

Comprehensive security features for production deployments:

#### Authentication Options

- **GitHub OAuth 2.0**: Native GitHub authentication for self-hosted deployments
- **Multi-Factor Support**: Leverages GitHub's existing SAML/SSO if configured

#### Two-Layer Authorization Model

**Layer 1: Application Access**
- **Enterprise Membership**: Require users to be members of a specific enterprise
- **Enterprise Admin**: Restrict access to enterprise administrators only
- **Organization Membership**: Allow users from specific organizations
- **Team Membership**: Limit access to specific team members

**Layer 2: Repository-Level Permissions**
- **Enterprise Admins**: Full access to migrate any repository (when enterprise slug is configured)
- **Privileged Teams**: Designated teams with full migration access across all repositories
- **Organization Admins**: Can migrate all repositories within their organizations
- **Repository Admins**: Can only migrate repositories they have admin access to (default for all users)

#### Self-Service Developer Migrations

Enable developers to migrate their own repositories with proper access controls:
- Users see all repositories but can only migrate those they have admin access to
- No admin intervention needed for developers to migrate their own repos

#### Security Features

- **Permission Validation**: All migration operations validate user permissions before execution
- **Audit Logging**: Complete audit trail showing who initiated each migration action

See [OPERATIONS.md](./docs/OPERATIONS.md#authentication-setup) for detailed configuration guide.

## Documentation

### Getting Started

- [Quick Start](#quick-start) - Get up and running in minutes
- [CONTRIBUTING.md](./docs/CONTRIBUTING.md) - Development setup, coding standards, testing, and debugging

### Operations & Deployment

- [DEPLOYMENT.md](./docs/DEPLOYMENT.md) - Production deployment for Docker, Kubernetes, and manual installation
- [OPERATIONS.md](./docs/OPERATIONS.md) - Daily operations, monitoring, incident response, and troubleshooting
- [API.md](./docs/API.md) - Complete REST API reference with examples

### Technical Reference

- [IMPLEMENTATION_GUIDE.md](./docs/IMPLEMENTATION_GUIDE.md) - Architecture, components, and implementation details
- [OpenAPI Specification](./docs/openapi.json) - Machine-readable API specification

### Configuration

- [config_template.yml](./configs/config_template.yml) - Complete YAML configuration reference with examples
- [env.example](./configs/env.example) - Environment variables template (alternative to YAML config)

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

For comprehensive development information, see [CONTRIBUTING.md](./docs/CONTRIBUTING.md).

## Deployment

### Docker Quick Deploy

```bash
# Set environment variables
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"

# Build and run
make docker-build
make docker-run

# Access at http://localhost:8080
```

### Production Deployment

For production deployments, see [DEPLOYMENT.md](./docs/DEPLOYMENT.md) which covers:

- Docker and Docker Compose configuration
- Kubernetes deployment with manifests
- PostgreSQL database setup and optimization
- Security hardening and best practices
- Monitoring and alerting setup
- Backup and recovery procedures
- Troubleshooting common issues

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./docs/CONTRIBUTING.md)
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

## Acknowledgments

- Built with [go-github](https://github.com/google/go-github) for GitHub API integration
- Uses [git-sizer](https://github.com/github/git-sizer) for repository analysis
- Powered by Go, React, and modern open-source tools

---

**Last Updated**: November 2025  
**Maintained by**: [@kuhlman-labs](https://github.com/kuhlman-labs)