# GitHub Actions Workflows

This directory contains all CI/CD workflows for the GitHub Migrator project.

## Workflows Overview

### Testing & Quality

#### `ci.yml` - Continuous Integration
**Triggers**: Push to main/develop, Pull Requests  
**Purpose**: Primary CI pipeline for code quality  
**Components**:
- Backend tests (Go) with mandatory linting
- Frontend tests (React/TypeScript) with mandatory ESLint
- Security scanning
- Dependency checks
- Docker build validation

### Build & Deploy

#### `build.yml` - Container Image Build
**Triggers**: Push to main, Tags, PRs, Manual  
**Purpose**: Build and push Docker images  
**Outputs**:
- Docker images to GitHub Container Registry
- Build provenance attestations
- Multiple image tags (latest, dev, semver, sha)

#### `deploy-main.yml` - Development Slot Deployment
**Triggers**: Push to main, Manual  
**Purpose**: Deploy to dev slot on production App Service  
**Target**: Azure App Service `dev` deployment slot

#### `deploy-pr.yml` - PR Preview Deployment
**Triggers**: Pull Request (opened, synchronize, reopened, closed)  
**Purpose**: Create/update/cleanup PR preview environments  
**Target**: Dynamic `pr-{number}` deployment slots  
**Features**:
- Automatic slot creation on PR open
- Comments PR with preview URL
- Uses in-memory SQLite for ephemeral testing
- Automatic cleanup on PR close

#### `deploy-release.yml` - Production Deployment
**Triggers**: Release published, Manual  
**Purpose**: Deploy releases to production via staging slot  
**Flow**:
1. Build release image
2. Deploy to staging slot
3. Health check staging
4. Swap staging to production (zero-downtime)

### Infrastructure

#### `terraform.yml` - Infrastructure as Code
**Triggers**: Manual  
**Purpose**: Manage production infrastructure with deployment slots  
**Operations**: Plan, Apply, Destroy
**Resources Created**:
- App Service Plan (S1 or higher for slots)
- App Service with production slot
- Staging deployment slot
- Dev deployment slot
- PostgreSQL database

## Deployment Architecture

```
                    GitHub Actions
                         |
         +---------------+---------------+
         |               |               |
    PR Opened       Push to main    Release Published
         |               |               |
         v               v               v
    +---------+    +---------+    +-----------+
    | pr-123  |    |   dev   |    |  staging  |
    |  slot   |    |  slot   |    |   slot    |
    +---------+    +---------+    +-----------+
         |               |               |
         | (ephemeral)   | (persistent)  | swap
         |               |               v
    Delete on        Dev testing    +------------+
    PR close                        | production |
                                    |   slot     |
                                    +------------+
```

## Workflow Dependencies

```
Pull Request Flow:
+-- ci.yml (Backend, Frontend, Security)
+-- deploy-pr.yml
    +-- Build container image
    +-- Create/update pr-{number} slot
    +-- Comment PR with preview URL
    +-- All pass -> Allow merge

Main Branch Flow:
+-- ci.yml
+-- deploy-main.yml
    +-- Build container image (tag: dev)
    +-- Deploy to dev slot
    +-- Health check

Release Flow:
+-- deploy-release.yml
    +-- Build release image (tag: v1.0.0, prod)
    +-- Deploy to staging slot
    +-- Health check staging
    +-- Swap staging to production
    +-- Health check production
```

## Running Workflows Manually

### Via GitHub UI
1. Go to Actions tab
2. Select workflow
3. Click "Run workflow"
4. Select branch and options
5. Click "Run workflow" button

### Via GitHub CLI

#### Deploy to dev slot
```bash
gh workflow run deploy-main.yml -r main
```

#### Deploy to production (with tag)
```bash
gh workflow run deploy-release.yml \
  -f image_tag=v1.0.0 \
  -r main
```

#### Terraform plan
```bash
gh workflow run terraform.yml \
  -f action=plan \
  -r main
```

#### Terraform apply
```bash
gh workflow run terraform.yml \
  -f action=apply \
  -r main
```

## Repository Configuration

### Repository Secrets (Settings → Secrets → Actions)

These are shared across all environments:

| Secret | Description |
|--------|-------------|
| `AZURE_CREDENTIALS` | Azure service principal JSON (see format below) |
| `AZURE_RESOURCE_GROUP` | Azure resource group name |
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID (for Terraform) |

#### `AZURE_CREDENTIALS` Format
```json
{
  "clientId": "<service-principal-app-id>",
  "clientSecret": "<service-principal-secret>",
  "subscriptionId": "<azure-subscription-id>",
  "tenantId": "<azure-tenant-id>"
}
```

### Repository Variables (Settings → Variables → Actions)

| Variable | Description | Example |
|----------|-------------|---------|
| `APP_SERVICE_NAME` | Azure App Service name (from Terraform output) | `app-github-migrator-prod` |

### Terraform Variables (for infrastructure provisioning)

These are only needed if running the Terraform workflow:

| Variable | Description | Default |
|----------|-------------|---------|
| `AZURE_LOCATION` | Azure region | `eastus` |
| `APP_NAME_PREFIX` | Resource naming prefix | `github-migrator` |
| `APP_SERVICE_SKU` | App Service Plan SKU | `S1` |
| `ALWAYS_ON` | Keep app always running | `true` |
| `DOCKER_IMAGE_TAG` | Default image tag | `prod` |
| `DATABASE_NAME` | PostgreSQL database name | `migrator` |
| `DATABASE_ADMIN_USERNAME` | Database admin user | `psqladmin` |
| `POSTGRES_VERSION` | PostgreSQL version | `15` |
| `DATABASE_SKU` | Database SKU | `GP_Standard_D2s_v3` |
| `DATABASE_STORAGE_MB` | Storage size in MB | `32768` |
| `DATABASE_BACKUP_RETENTION_DAYS` | Backup retention | `30` |
| `DATABASE_GEO_REDUNDANT_BACKUP` | Geo-redundant backups | `true` |
| `DATABASE_HA_MODE` | High availability mode | `ZoneRedundant` |

> **Note**: Application configuration (source/destination GitHub, auth, logging, etc.) is done via the Settings UI in the app after deployment.

## GitHub Environments

Create these environments in GitHub (Settings → Environments):

| Environment | Purpose | Protection Rules |
|-------------|---------|------------------|
| `dev` | Dev slot deployments | None |
| `staging` | Staging slot (pre-prod) | Required reviewers (optional) |
| `prod` | Production & Terraform | Required reviewers |
| `pr-preview` | PR preview environments | None |

> **Note**: Secrets and variables are configured at the **repository level** (not per-environment) since all environments share the same Azure infrastructure. Environments are used for deployment protection rules only.

## Deployment Slots

### Slot Configuration

| Slot | Purpose | Database | Persistence |
|------|---------|----------|-------------|
| production | Live production | PostgreSQL | Yes |
| staging | Pre-prod testing | PostgreSQL (shared) | Yes |
| dev | Development testing | PostgreSQL (shared) | Yes |
| pr-{number} | PR previews | In-memory SQLite | No |

### SKU Requirements
Deployment slots require **Standard (S1)** tier or higher. The Terraform configuration defaults to S1.

## Branch Protection Rules

### `main` branch
**Required Status Checks**:
- Backend CI (Go)
- Frontend CI (React/TypeScript)
- Docker Build Test

**Additional Settings**:
- Require pull request reviews (1+)
- Require approval before merge
- Dismiss stale reviews on push
- Require status checks to pass
- Require branches to be up to date

## Workflow Optimization

### Caching Strategy
All workflows use caching to speed up execution:
- **Go modules**: `~/.cache/go-build`, `~/go/pkg/mod`
- **Node modules**: `~/.npm`, `node_modules`
- **Docker layers**: GitHub Actions cache
- **Terraform**: `.terraform`, provider plugins

### Parallel Execution
Jobs run in parallel where possible:
```
Start
 +-- Backend CI (parallel)
 +-- Frontend CI (parallel)
 +-- Security Scan (parallel)
 +-- Dependency Check (parallel)
```

### Zero-Downtime Deployments
Production deployments use slot swapping:
1. Deploy new version to staging slot
2. Warm up staging slot with health checks
3. Swap staging with production (instant)
4. Old version remains in staging for rollback

### Rollback Procedure
To rollback production:
```bash
az webapp deployment slot swap \
  --name <app-service-name> \
  --resource-group <resource-group> \
  --slot staging \
  --target-slot production
```

## Monitoring & Debugging

### View Workflow Runs
```bash
# List recent runs
gh run list --workflow=deploy-release.yml

# View specific run
gh run view <run-id>

# View logs for failed run
gh run view <run-id> --log-failed

# Watch a running workflow
gh run watch <run-id>
```

### Common Issues

#### 1. Deployment Slot Creation Fails
**Symptom**: PR slot creation fails  
**Solution**: Ensure App Service Plan is S1 or higher

#### 2. Health Check Timeout
**Symptom**: Deployment fails at health check  
**Solution**: Increase stabilization wait time or check app startup logs

#### 3. Swap Fails
**Symptom**: Staging to production swap fails  
**Solution**: Check staging health, ensure both slots are running

#### 4. PR Slot Cleanup Fails
**Symptom**: Orphaned PR slots  
**Solution**: Manually delete via Azure CLI:
```bash
az webapp deployment slot delete \
  --name <app-name> \
  --resource-group <rg> \
  --slot pr-123
```

## Documentation

- [Deployment Guide](../../docs/deployment/README.md)
- [Azure Deployment](../../docs/deployment/AZURE.md)
- [Kubernetes Deployment](../../docs/deployment/KUBERNETES.md)
- [Architecture Overview](../../docs/ARCHITECTURE.md)

## Contributing

When modifying workflows:
1. Follow existing naming conventions
2. Test with `workflow_dispatch` first
3. Use GitHub Actions cache where possible
4. Update this README
5. Consider impact on PR preview environments

## Support

For workflow issues:
1. Check workflow logs in GitHub Actions
2. Review this documentation
3. Test locally using Make targets
4. Open an issue with workflow run ID
