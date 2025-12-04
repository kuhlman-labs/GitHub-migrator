# Documentation

Welcome to the GitHub Migrator documentation. This guide will help you deploy, operate, and extend the migration platform.

## Quick Navigation

| Document | Description |
|----------|-------------|
| [Deployment Guide](./deployment/) | Docker, Azure, and Kubernetes deployment |
| [API Reference](./API.md) | REST API documentation and examples |
| [Operations Guide](./OPERATIONS.md) | Authentication, workflows, monitoring, troubleshooting |
| [Contributing Guide](./CONTRIBUTING.md) | Development setup, testing, and code standards |
| [OpenAPI Specification](./openapi.json) | Machine-readable API specification |

## Getting Started

### For Operators

1. **Deploy the Application** - Start with the [Deployment Guide](./deployment/)
   - [Docker Quick Start](./deployment/README.md) - Fastest way to get running
   - [Azure Deployment](./deployment/AZURE.md) - Azure App Service with Terraform
   - [Kubernetes Deployment](./deployment/KUBERNETES.md) - Production K8s setup

2. **Configure Authentication** - See [OPERATIONS.md](./OPERATIONS.md#authentication-setup)
   - GitHub App setup for enhanced rate limits
   - OAuth configuration for user access control
   - Azure DevOps setup for ADO migrations

3. **Run Your First Migration** - Follow the workflow in [OPERATIONS.md](./OPERATIONS.md#migration-workflows)

### For Developers

1. **Set Up Development Environment** - See [CONTRIBUTING.md](./CONTRIBUTING.md#development-environment)
2. **Understand the API** - Review [API.md](./API.md)
3. **Submit Changes** - Follow [CONTRIBUTING.md](./CONTRIBUTING.md#making-changes)

## Documentation Structure

```
docs/
├── README.md              # This file - documentation index
├── deployment/            # Deployment guides
│   ├── README.md          # Docker and common setup
│   ├── AZURE.md           # Azure App Service deployment
│   └── KUBERNETES.md      # Kubernetes deployment
├── API.md                 # REST API reference
├── openapi.json           # OpenAPI 3.0 specification
├── OPERATIONS.md          # Operations runbook
└── CONTRIBUTING.md        # Development guide
```

## Key Topics by Role

### Migration Administrators

- [Authentication Setup](./OPERATIONS.md#authentication-setup) - Configure access control
- [Migration Workflows](./OPERATIONS.md#migration-workflows) - Step-by-step migration process
- [Monitoring & Alerts](./OPERATIONS.md#monitoring--alerts) - Set up observability
- [Troubleshooting Guide](./OPERATIONS.md#troubleshooting-guide) - Common issues and solutions

### Platform Engineers

- [Azure Deployment](./deployment/AZURE.md) - Terraform and CI/CD setup
- [Kubernetes Deployment](./deployment/KUBERNETES.md) - Production configuration
- [Database Setup](./OPERATIONS.md#database-setup) - SQLite vs PostgreSQL
- [Maintenance Tasks](./OPERATIONS.md#maintenance-tasks) - Ongoing operations

### Developers

- [API Reference](./API.md) - Complete endpoint documentation
- [OpenAPI Spec](./openapi.json) - Generate clients in any language
- [Development Setup](./CONTRIBUTING.md#development-environment) - Local development
- [Code Standards](./CONTRIBUTING.md#code-standards) - Coding guidelines

### Application Integrators

- [API Reference](./API.md) - REST endpoints and examples
- [Self-Service Migration](./API.md#selfservice) - Programmatic migrations
- [Webhooks](./OPERATIONS.md#monitoring--alerts) - Event notifications

## External Resources

- [GitHub Migrations API](https://docs.github.com/en/rest/migrations)
- [GitHub Enterprise Importer](https://docs.github.com/en/migrations)
- [GitHub App Authentication](https://docs.github.com/en/apps)
- [GitHub Actions Importer](https://docs.github.com/en/actions/migrating-to-github-actions)

## Need Help?

1. Check the [Troubleshooting Guide](./OPERATIONS.md#troubleshooting-guide)
2. Review [API Error Handling](./API.md#error-handling)
3. Open an issue on GitHub
