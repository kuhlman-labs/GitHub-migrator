# GitHub Migrator

A comprehensive solution for migrating repositories between GitHub instances at scale.

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/dl/)
[![Node Version](https://img.shields.io/badge/node-24+-green.svg)](https://nodejs.org/)

## Overview

GitHub Migrator automates large-scale repository migrations with intelligent discovery, batch organization, and a modern web dashboard. It profiles repositories to identify migration complexity, detects GitHub Enterprise Importer limits, and provides real-time monitoring throughout the migration process.

### Supported Migration Paths

| Source | Destination |
|--------|-------------|
| GitHub Enterprise Server | GitHub Enterprise Cloud, EMU, or Data Residency |
| GitHub.com | GitHub Enterprise Cloud, EMU, or Data Residency |
| Azure DevOps Services | GitHub Enterprise Cloud, EMU, or Data Residency|

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 24+
- Docker

### Install & Run

```bash
# Clone and install
git clone https://github.com/kuhlman-labs/GitHub-migrator.git
cd GitHub-migrator
make install

# Configure authentication
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"   # Source org admin token
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"     # Destination org admin token

# Run (choose one)
make docker-build && make docker-run    # Docker: http://localhost:8080
# OR
make run-server   # Backend only: http://localhost:8080
make web-dev      # Frontend dev: http://localhost:3000
```

**Note:** The default config sets both source and destination to `github.com`. For other sources, see [Configuration](#configuration).

### First Migration

1. Open the dashboard at http://localhost:8080 (localhost:3000 if running in the terminal)
2. **Discover** - Enter your organization name to scan repositories
3. **Create Batch** - Select 3-5 simple repositories for a pilot batch
4. **Dry Run** - Test without executing the actual migration
5. **Migrate** - Execute and monitor progress in real-time

## Documentation

| Document | Description |
|----------|-------------|
| [Deployment Guide](./docs/deployment/) | Docker, Azure, and Kubernetes deployment |
| [API Reference](./docs/API.md) | REST API documentation and examples |
| [Operations Guide](./docs/OPERATIONS.md) | Authentication, workflows, monitoring, troubleshooting |
| [Contributing Guide](./docs/CONTRIBUTING.md) | Development setup, testing, and code standards |
| [OpenAPI Spec](./docs/openapi.json) | Machine-readable API specification |

### Configuration

| Source | Environment File | YAML Config |
|--------|------------------|-------------|
| GitHub | [env.github.example](./configs/env.github.example) | [config.github.yml](./configs/config.github.yml) |
| Azure DevOps | [env.azuredevops.example](./configs/env.azuredevops.example) | [config.azuredevops.yml](./configs/config.azuredevops.yml) |

See [configs/README.md](./configs/README.md) for detailed configuration guide.

The Server has a guided setup page if no configuration is detected.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for development setup and guidelines.

## Resources

- [GitHub Migration Guide](https://docs.github.com/en/migrations)

## Acknowledgments

Built with [go-github](https://github.com/google/go-github), [git-sizer](https://github.com/github/git-sizer), Go, and React.

---

**Maintained by**: [@kuhlman-labs](https://github.com/kuhlman-labs)
