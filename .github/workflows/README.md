# GitHub Actions Workflows

This directory contains all CI/CD workflows for the GitHub Migrator project.

## Workflows Overview

### 🧪 Testing & Quality

#### `ci.yml` - Continuous Integration
**Triggers**: Push to main/develop, Pull Requests  
**Purpose**: Primary CI pipeline for code quality  
**Components**:
- Backend tests (Go)
- Frontend tests (React/TypeScript)
- Security scanning
- Dependency checks
- Docker build validation

#### `integration-tests.yml` - Database Integration Tests
**Triggers**: Push, PRs, Daily schedule, Manual  
**Purpose**: Validate database compatibility  
**Databases Tested**:
- ✅ SQLite (always) - ~30 seconds
- ✅ PostgreSQL (always) - ~2 minutes
- ⚠️ SQL Server (scheduled/manual) - ~5 minutes

**Strategy**:
- Fast feedback with SQLite
- Production validation with PostgreSQL
- Enterprise validation with SQL Server (conditional)

### 📦 Build & Deploy

#### `build.yml` - Container Image Build
**Triggers**: Push to main, Tags, PRs, Manual  
**Purpose**: Build and push Docker images  
**Outputs**:
- Docker images to GitHub Container Registry
- Build provenance attestations
- Multiple image tags (latest, dev, semver, sha)

#### `deploy-dev.yml` - Development Deployment
**Triggers**: Push to develop, Manual  
**Purpose**: Deploy to development environment  
**Target**: Azure development environment

#### `deploy-prod.yml` - Production Deployment
**Triggers**: Push to main, Tags (v*), Manual  
**Purpose**: Deploy to production environment  
**Target**: Azure production environment  
**Requires**: Manual approval

### 🏗️ Infrastructure

#### `terraform-dev.yml` - Development Infrastructure
**Triggers**: Push to terraform files (develop), PRs, Manual  
**Purpose**: Manage development infrastructure as code  
**Operations**: Plan, Apply, Destroy

#### `terraform-prod.yml` - Production Infrastructure
**Triggers**: Push to terraform files (main), Tags, Manual  
**Purpose**: Manage production infrastructure as code  
**Operations**: Plan, Apply (with approval)

## Workflow Dependencies

```
Pull Request Flow:
├─ ci.yml (Backend, Frontend, Security)
├─ integration-tests.yml (SQLite, PostgreSQL)
└─ build.yml (Docker build test)
    └─ All pass → Allow merge

Main Branch Flow:
├─ ci.yml
├─ integration-tests.yml (+ SQL Server optional)
├─ build.yml (Push images)
└─ deploy-prod.yml (If tag: v*)

Develop Branch Flow:
├─ ci.yml
├─ integration-tests.yml
├─ build.yml
└─ deploy-dev.yml
```

## Running Workflows Manually

### Via GitHub UI
1. Go to Actions tab
2. Select workflow
3. Click "Run workflow"
4. Select branch and options
5. Click "Run workflow" button

### Via GitHub CLI

#### Run integration tests with SQL Server
```bash
gh workflow run integration-tests.yml \
  -f test-sqlserver=true \
  -r main
```

#### Deploy to development
```bash
gh workflow run deploy-dev.yml -r develop
```

#### Deploy to production
```bash
gh workflow run deploy-prod.yml -r main
```

#### Terraform plan for production
```bash
gh workflow run terraform-prod.yml \
  -f action=plan \
  -r main
```

## Secrets Required

### GitHub Container Registry
- `GITHUB_TOKEN` (automatically provided)

### Azure Deployment
- `AZURE_CREDENTIALS` - Service principal credentials
- `AZURE_SUBSCRIPTION_ID` - Azure subscription ID
- `AZURE_TENANT_ID` - Azure tenant ID

### Application Secrets
- `GHMIG_SOURCE_TOKEN` - Source GitHub token
- `GHMIG_DESTINATION_TOKEN` - Destination GitHub token
- `GHMIG_SOURCE_BASE_URL` - Source GitHub URL
- `GHMIG_DESTINATION_BASE_URL` - Destination GitHub URL

### Database (Production)
- `POSTGRES_PASSWORD` - PostgreSQL password
- `DATABASE_DSN` - Database connection string

## Branch Protection Rules

### `main` branch
**Required Status Checks**:
- ✅ Backend CI (Go)
- ✅ Frontend CI (React/TypeScript)
- ✅ Integration Tests - SQLite
- ✅ Integration Tests - PostgreSQL
- ✅ Docker Build Test

**Additional Settings**:
- Require pull request reviews (1+)
- Require approval before merge
- Dismiss stale reviews on push
- Require status checks to pass
- Require branches to be up to date

### `develop` branch
**Required Status Checks**:
- ✅ Backend CI (Go)
- ✅ Integration Tests - SQLite

**Additional Settings**:
- Require pull request reviews (1)
- Require status checks to pass

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
 ├─ Backend CI (parallel)
 ├─ Frontend CI (parallel)
 ├─ Security Scan (parallel)
 └─ Integration Tests (parallel)
     ├─ SQLite (parallel)
     ├─ PostgreSQL (parallel)
     └─ SQL Server (conditional)
```

### Cost Optimization
- Use path filters to skip unnecessary runs
- Skip SQL Server on PRs (run on schedule)
- Use GitHub Actions cache for artifacts
- Parallelize independent jobs

## Monitoring & Debugging

### View Workflow Runs
```bash
# List recent runs
gh run list --workflow=integration-tests.yml

# View specific run
gh run view <run-id>

# View logs for failed run
gh run view <run-id> --log-failed

# Watch a running workflow
gh run watch <run-id>
```

### Common Issues

#### 1. PostgreSQL Connection Timeout
**Symptom**: Tests fail with connection timeout  
**Solution**: Increase health check retries in `integration-tests.yml`

#### 2. SQL Server Startup Timeout
**Symptom**: SQL Server not ready after 60 seconds  
**Solution**: Increase wait timeout to 120 seconds

#### 3. Cache Invalidation
**Symptom**: Stale dependencies  
**Solution**: Clear workflow cache or update cache key

#### 4. Docker Build Failures
**Symptom**: Build fails in CI but works locally  
**Solution**: Check Dockerfile PATH, ensure all files are committed

## Maintenance Schedule

### Daily
- ✅ Automatic: Integration tests (scheduled)
- ✅ Automatic: Security scans

### Weekly
- 🔍 Review: Workflow execution times
- 🔍 Review: Failure rates
- 🔍 Review: Resource usage

### Monthly
- 🔧 Update: Action versions
- 🔧 Update: Base images
- 🔧 Review: Caching strategy
- 🔧 Review: Cost optimization

## Documentation

- [CI Integration Testing Guide](../docs/CI_INTEGRATION_TESTING.md)
- [CI Pipeline Diagram](../docs/CI_PIPELINE_DIAGRAM.md)
- [GORM Refactoring Summary](../docs/GORM_REFACTORING_SUMMARY.md)
- [Deployment Guide](../docs/DEPLOYMENT.md)

## Contributing

When adding new workflows:
1. Follow existing naming conventions
2. Add comprehensive documentation
3. Use workflow caching where possible
4. Test with `workflow_dispatch` first
5. Update this README

## Support

For workflow issues:
1. Check workflow logs in GitHub Actions
2. Review documentation
3. Test locally using Make targets
4. Open an issue with workflow run ID
