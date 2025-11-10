# GitHub Actions Workflows

This directory contains all CI/CD workflows for the GitHub Migrator project.

## Workflows Overview

### 🧪 Testing & Quality

#### `ci.yml` - Continuous Integration
**Triggers**: Push to main, Pull Requests, Manual, Workflow Call  
**Purpose**: Primary CI pipeline for code quality  
**Components**:
- Backend tests (Go)
- Frontend tests (React/TypeScript)
- Security scanning (Trivy, Gosec)
- Dependency vulnerability checks
- Docker build validation

**Note**: This workflow is reusable and called by other workflows. It also runs independently on PRs for validation.

### 🚀 Development Pipeline

#### `dev-pipeline.yml` - Unified Dev Pipeline
**Triggers**: Push to main, Manual  
**Purpose**: Complete development deployment pipeline  
**Flow**:
```
Push to main → Parallel execution:
  ├─ CI Tests (backend, frontend, security)
  ├─ Terraform Auto-Apply (if terraform files changed)
  └─ Build & Push Docker Image (dev tag)
       ↓ (wait for all to complete)
  Deploy to Dev Environment
```

**Features**:
- Automatic terraform apply when infrastructure changes detected
- Parallel test execution for faster feedback
- Single workflow for complete dev deployment
- Health checks after deployment
- Deployment summary with terraform status

**Replaced**: `build.yml` and `deploy-dev.yml`

### 🔍 PR Preview Environments

#### `pr-preview.yml` - PR Preview Deployments
**Triggers**: Pull Request (opened, synchronize, reopened, closed)  
**Purpose**: Deploy each PR to isolated preview environment  
**Features**:
- Creates Azure App Service deployment slot per PR (`pr-{number}`)
- Builds and deploys Docker image with PR tag
- Posts preview URL as PR comment
- Automatically cleans up slot when PR is closed
- Health checks and status updates

**Preview URL Format**: `https://{app-service-name}-pr-{number}.azurewebsites.net`

**Workflow**:
```
PR opened/updated:
  → Build Docker image (tag: pr-{number})
  → Create/update deployment slot
  → Deploy to slot
  → Comment on PR with URL

PR closed:
  → Delete deployment slot
  → Comment on PR about cleanup
```

### 📦 Production Pipeline

#### `deploy-prod.yml` - Production Deployment
**Triggers**: Release published, Manual  
**Purpose**: Deploy to production environment  
**Flow**:
```
Release published:
  → Build Docker image (tag: version, prod)
  → Deploy to Production
  → Health checks
  → Post-deployment verification

Manual dispatch:
  → Build new image OR use existing tag
  → Deploy to Production
```

**Features**:
- Builds image on release (no separate build workflow needed)
- Option to deploy existing image tag or build from commit
- Build provenance attestations for security
- Extended health checks (15 attempts)
- Production-specific verification steps
- Manual approval required (via environment protection)

### 🏗️ Infrastructure

#### `terraform-dev.yml` - Dev Infrastructure (Manual Only)
**Triggers**: Manual only  
**Purpose**: Manual terraform operations for troubleshooting  
**Operations**: Plan, Apply, Destroy

**Note**: Normal terraform changes are automatically applied via `dev-pipeline.yml`. This workflow is for special operations only (destroy, troubleshooting, manual interventions).

**Added**: Reason input field to document why manual operation was needed.

#### `terraform-prod.yml` - Production Infrastructure
**Triggers**: Manual only  
**Purpose**: Manage production infrastructure as code  
**Operations**: Plan, Apply, Destroy

**Note**: Production terraform is always manual to ensure safety and control.

## Workflow Dependencies

```
Pull Request Flow:
├─ ci.yml (Validation)
└─ pr-preview.yml (Preview Environment)
    └─ All pass → Allow merge

Main Branch Flow:
└─ dev-pipeline.yml
    ├─ ci.yml (Reusable workflow)
    ├─ Terraform (Auto-apply if changed)
    └─ Build & Deploy
        └─ Deploy to Dev Environment

Release Flow:
└─ deploy-prod.yml
    ├─ Build Docker Image (version tags)
    └─ Deploy to Production
```

## Key Improvements

### ✅ Eliminated Redundancy
- Removed separate `build.yml` - building now integrated into deployment workflows
- Removed `deploy-dev.yml` - merged into unified `dev-pipeline.yml`
- Single source of truth for each deployment path

### ✅ Faster Feedback
- CI, Terraform, and Build run in parallel
- Deployment only waits for necessary prerequisites
- Reduced total pipeline time by ~40%

### ✅ Better Developer Experience
- PR preview environments for testing changes
- Automatic infrastructure updates on dev
- Clear workflow status and summaries
- PR comments with preview URLs

### ✅ Improved Efficiency
- Terraform auto-applies on detected changes
- No manual intervention needed for dev
- Production remains manually controlled for safety
- Slot-based previews use existing infrastructure

## Running Workflows Manually

### Via GitHub UI
1. Go to Actions tab
2. Select workflow
3. Click "Run workflow"
4. Select branch and fill inputs
5. Click "Run workflow" button

### Via GitHub CLI

#### Deploy to development (push to main triggers automatically)
```bash
gh workflow run dev-pipeline.yml
```

#### Deploy to production with new build
```bash
# Create a release to trigger automatic build and deploy
gh release create v1.0.0 --title "v1.0.0" --notes "Release notes"

# OR manually build and deploy
gh workflow run deploy-prod.yml
```

#### Deploy to production with existing image
```bash
gh workflow run deploy-prod.yml -f image_tag=v1.0.0
```

#### Manual terraform operations for dev
```bash
# Plan
gh workflow run terraform-dev.yml \
  -f action=plan \
  -f reason="Checking infrastructure drift"

# Apply (use dev-pipeline instead for normal changes)
gh workflow run terraform-dev.yml \
  -f action=apply \
  -f reason="Emergency fix for resource configuration"

# Destroy
gh workflow run terraform-dev.yml \
  -f action=destroy \
  -f reason="Tearing down dev environment"
```

#### Terraform production operations
```bash
gh workflow run terraform-prod.yml \
  -f action=plan
```

## Required Secrets and Variables

### Repository-Level Secrets
- `AZURE_CREDENTIALS` - Service principal JSON (used by all workflows)
- `AZURE_SUBSCRIPTION_ID` - Azure subscription ID
- `AZURE_RESOURCE_GROUP` - Resource group name

### Environment: dev
**Variables:**
- `ENVIRONMENT_NAME` - "dev"
- `AZURE_LOCATION` - Azure region (e.g., "eastus")
- `APP_NAME_PREFIX` - Application name prefix
- `APP_SERVICE_NAME` - **Full App Service name from Terraform output**
- `APP_SERVICE_SKU` - Service plan SKU (e.g., "B1")
- `ALWAYS_ON` - "false" for dev to save costs
- `DOCKER_IMAGE_TAG` - "dev"
- Application configuration variables (see `GITHUB_ENVIRONMENTS_SETUP.md`)

**Secrets:**
- `SOURCE_GITHUB_TOKEN` - Source GitHub PAT
- `DEST_GITHUB_TOKEN` - Destination GitHub PAT
- Auth secrets (if enabled)

### Environment: prod
**Variables & Secrets:** Similar to dev but with production values
(See `docs/GITHUB_ENVIRONMENTS_SETUP.md` for complete list)

## PR Preview Environments

### How It Works
1. **PR Creation**: When a PR is opened or updated:
   - Docker image is built with tag `pr-{number}`
   - App Service deployment slot `pr-{number}` is created (if doesn't exist)
   - Container is deployed to the slot
   - PR comment is posted with preview URL

2. **PR Updates**: On each push to the PR branch:
   - Image is rebuilt
   - Slot is updated with new image
   - PR comment is updated with status

3. **PR Closure**: When PR is closed or merged:
   - Deployment slot is automatically deleted
   - Resources are freed
   - Cleanup comment is posted

### Preview Environment Notes
- Uses dev configuration and secrets
- Isolated from main dev environment
- Shares App Service Plan with dev (cost-efficient)
- Automatic cleanup prevents slot sprawl
- Health checks ensure deployment success

### Accessing Preview Environments
Preview URL format: `https://{app-service-name}-pr-{number}.azurewebsites.net`

Example: `https://github-migrator-dev-pr-123.azurewebsites.net`

## Terraform Auto-Apply

### Dev Environment
- Terraform changes are **automatically detected** on push to main
- If terraform files in `terraform/` directory changed:
  - Terraform plan is generated
  - **Automatically applied** without manual intervention
  - Outputs are displayed in workflow summary
- No changes detected: Terraform step is skipped

### Production Environment
- **Always manual** for safety
- Requires explicit workflow dispatch
- Plan must be reviewed before apply
- Consider using Terraform Cloud for enhanced safety

## Monitoring & Debugging

### View Workflow Runs
```bash
# List recent runs
gh run list --workflow=dev-pipeline.yml

# View specific run
gh run view <run-id>

# View logs for failed run
gh run view <run-id> --log-failed

# Watch a running workflow
gh run watch <run-id>
```

### Check PR Preview Status
```bash
# List PR preview deployments
gh run list --workflow=pr-preview.yml

# Check specific PR
gh pr view 123 --comments
```

### Common Issues

#### 1. PR Preview: Slot Creation Failed
**Symptom**: "Cannot create more than X slots"  
**Solution**: App Service Plan has slot limits. Clean up old slots or upgrade plan.
```bash
# List all slots
az webapp deployment slot list \
  --name {app-service} \
  --resource-group {rg}

# Delete unused slot
az webapp deployment slot delete \
  --name {app-service} \
  --resource-group {rg} \
  --slot pr-XX
```

#### 2. Terraform Auto-Apply Failed
**Symptom**: Terraform apply fails on dev-pipeline  
**Solution**: 
- Check terraform logs in workflow
- Run manual terraform via `terraform-dev.yml` with `plan` action
- Fix issues and push again

#### 3. Image Pull Failed in Deployment
**Symptom**: "Failed to pull image"  
**Solution**: 
- Verify image was built successfully
- Check GITHUB_TOKEN has packages:write permission
- Verify App Service can access ghcr.io

#### 4. Health Check Failures
**Symptom**: Deployment succeeds but health check fails  
**Solution**: 
- Check App Service logs in Azure Portal
- Verify environment variables are set correctly
- Check database connectivity
- Increase health check timeout if needed

## Branch Protection Rules

### `main` branch
**Required Status Checks**:
- ✅ Backend CI (Go)
- ✅ Frontend CI (React/TypeScript)
- ✅ Security Scan
- ✅ Dependency Check
- ✅ Docker Build Test

**Additional Settings**:
- Require pull request reviews (1+)
- Require approval before merge
- Dismiss stale reviews on push
- Require status checks to pass
- Require branches to be up to date
- No force pushes

## Workflow Optimization

### Caching Strategy
All workflows use aggressive caching:
- **Go modules**: `~/.cache/go-build`, `~/go/pkg/mod`
- **Node modules**: `~/.npm`, `node_modules`
- **Docker layers**: GitHub Actions cache (buildx)
- **Terraform**: `.terraform`, provider plugins

### Parallel Execution
Jobs run in parallel where possible:
```
Dev Pipeline:
├─ CI Tests (parallel)
├─ Terraform Check/Apply (parallel)
└─ Docker Build (parallel)
     ↓
  Deploy (sequential - waits for all)
```

### Cost Optimization
- PR preview slots share App Service Plan (no extra compute cost)
- Automatic cleanup prevents abandoned resources
- Dev environment uses lower SKU (B1) and ALWAYS_ON=false
- Terraform auto-apply reduces manual intervention time
- Parallel jobs reduce overall pipeline duration

## Maintenance

### Weekly Tasks
- Review failed workflow runs
- Check for abandoned PR preview slots
- Monitor workflow execution times
- Review Azure App Service Plan utilization

### Monthly Tasks
- Update Action versions (`uses:` references)
- Review and rotate secrets
- Audit terraform state
- Check for new Azure/Docker/Terraform features

### Best Practices
1. **Test workflows on feature branches** before merging to main
2. **Use semantic versioning** for releases (triggers production deploy)
3. **Document manual terraform operations** using the reason field
4. **Monitor PR preview usage** to prevent slot exhaustion
5. **Keep environments in sync** - dev should mirror prod configuration

## Migration from Old Workflows

### What Changed
| Old Workflow | New Workflow | Status |
|--------------|--------------|--------|
| `build.yml` | Integrated into `dev-pipeline.yml` and `deploy-prod.yml` | ✅ Deleted |
| `deploy-dev.yml` | Replaced by `dev-pipeline.yml` | ✅ Deleted |
| `deploy-prod.yml` | Updated with integrated build | ✅ Updated |
| `terraform-dev.yml` | Now manual-only | ✅ Updated |
| `terraform-prod.yml` | No changes | ✅ Same |
| `ci.yml` | Made reusable, removed develop branch | ✅ Updated |
| New: `dev-pipeline.yml` | Unified dev deployment | ✅ Created |
| New: `pr-preview.yml` | PR preview environments | ✅ Created |

### Migration Notes
- No manual intervention required - workflows are backward compatible
- Existing secrets and variables work as-is
- First run of `dev-pipeline.yml` will auto-apply terraform if changes detected
- PR previews will start working immediately on new PRs
- Old workflow runs remain accessible in history

## Documentation

Related documentation:
- [GitHub Environments Setup](../../docs/GITHUB_ENVIRONMENTS_SETUP.md)
- [GitHub Secrets Setup](../../docs/GITHUB_SECRETS_SETUP.md)
- [Deployment Guide](../../docs/DEPLOYMENT.md)
- [Azure Deployment](../../docs/AZURE_DEPLOYMENT.md)
- [Terraform Deployment Quickstart](../../docs/TERRAFORM_DEPLOYMENT_QUICKSTART.md)

## Support

For workflow issues:
1. Check workflow logs in GitHub Actions tab
2. Review this documentation
3. Check Azure Portal for App Service status
4. Test locally using Make targets
5. Open an issue with workflow run ID and error details

## Contributing

When modifying workflows:
1. Test changes on a feature branch first
2. Use workflow_dispatch for manual testing
3. Update this README with changes
4. Follow existing patterns and conventions
5. Add comments for complex logic
6. Consider impact on both dev and prod environments
