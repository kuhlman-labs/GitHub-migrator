# GitHub Environments Setup Guide

This guide shows how to set up GitHub Environments for organizing secrets and variables per environment (dev/production).

## 📋 Why Use GitHub Environments?

**Benefits:**
- ✅ **Better Organization** - Secrets scoped to specific environments
- ✅ **Protection Rules** - Require approvals for production deployments
- ✅ **Clear Separation** - Dev and prod configurations are isolated
- ✅ **Environment Variables** - Non-sensitive configs as variables (visible)
- ✅ **Deployment History** - Track deployments per environment

## 🏗️ Architecture

```
Repository
├── Secrets (shared across all environments)
│   └── AZURE_CREDENTIALS
│   └── AZURE_SUBSCRIPTION_ID
│   └── AZURE_RESOURCE_GROUP
│
├── Environment: dev
│   ├── Variables (non-sensitive, visible)
│   │   ├── APP_NAME_PREFIX = "github-migrator"
│   │   ├── APP_SERVICE_SKU = "B1"
│   │   └── ... (configuration)
│   └── Secrets (sensitive, hidden)
│       ├── SOURCE_GITHUB_TOKEN
│       ├── AUTH_GITHUB_OAUTH_CLIENT_SECRET
│       └── AUTH_SESSION_SECRET
│
└── Environment: production
    ├── Variables (different from dev)
    │   ├── APP_NAME_PREFIX = "github-migrator"
    │   ├── APP_SERVICE_SKU = "S1"
    │   └── ... (configuration)
    └── Secrets (separate from dev)
        ├── SOURCE_GITHUB_TOKEN
        ├── AUTH_GITHUB_OAUTH_CLIENT_SECRET
        └── AUTH_SESSION_SECRET
```

## 🚀 Step-by-Step Setup

### Step 1: Create Environments

1. Go to your repository on GitHub
2. Click **Settings**
3. In the left sidebar, click **Environments**
4. Click **New environment**
5. Name it: `dev`
6. Click **Configure environment**
7. (Leave protection rules empty for dev)
8. Click **Add environment**

Repeat for `production`:
1. Click **New environment**
2. Name it: `production`
3. Click **Configure environment**
4. **Configure Protection Rules** (recommended):
   - ✅ **Required reviewers** - Add yourself or team members
   - ✅ **Wait timer** - Optional: 5 minutes
   - ✅ **Deployment branches** - Select: `main` only
5. Click **Add environment**

### Step 2: Add Repository-Level Secrets

These are shared across all environments.

Navigate to: **Settings → Secrets and variables → Actions → Secrets**

Add these secrets at the **repository level**:

| Secret Name | Value | How to Get |
|------------|-------|------------|
| `AZURE_CREDENTIALS` | Service principal JSON | `az ad sp create-for-rbac --sdk-auth` |
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID | `az account show --query id -o tsv` |
| `AZURE_RESOURCE_GROUP` | Resource group name | Your existing resource group |

### Step 3: Configure Dev Environment

Navigate to: **Settings → Environments → dev**

#### Dev Environment Variables

Click **Add variable** for each:

| Variable Name | Value | Description |
|--------------|-------|-------------|
| `ENVIRONMENT_NAME` | `dev` | Environment identifier |
| `AZURE_LOCATION` | `eastus` | Azure region (see common regions below) |
| `APP_NAME_PREFIX` | `github-migrator` | App name prefix |
| `APP_SERVICE_NAME` | `github-migrator-dev` | **IMPORTANT**: Full App Service name (from Terraform output) |
| `APP_SERVICE_SKU` | `B1` | App Service tier (Basic) |
| `ALWAYS_ON` | `false` | Keep app always on (saves costs in dev) |
| `DOCKER_IMAGE_TAG` | `dev` | Container image tag |
| `SOURCE_TYPE` | `github` | Source provider type |
| `SOURCE_BASE_URL` | `https://api.github.com` | Source API URL |
| `DESTINATION_TYPE` | `github` | Destination provider type |
| `DESTINATION_BASE_URL` | `https://api.github.com` | Destination API URL |
| `MIGRATION_WORKERS` | `3` | Number of workers |
| `MIGRATION_POLL_INTERVAL_SECONDS` | `30` | Poll interval |
| `MIGRATION_POST_MIGRATION_MODE` | `production_only` | Post-migration mode |
| `MIGRATION_DEST_REPO_EXISTS_ACTION` | `fail` | Action if repo exists |
| `MIGRATION_VISIBILITY_PUBLIC_REPOS` | `private` | Public repo visibility |
| `MIGRATION_VISIBILITY_INTERNAL_REPOS` | `private` | Internal repo visibility |
| `LOGGING_LEVEL` | `info` | Log level |
| `LOGGING_FORMAT` | `json` | Log format |
| `AUTH_ENABLED` | `false` | Enable authentication (set to `true` if using) |
| `AUTH_CALLBACK_URL` | `https://github-migrator-dev.azurewebsites.net/api/v1/auth/callback` | OAuth callback URL |
| `AUTH_FRONTEND_URL` | `https://github-migrator-dev.azurewebsites.net` | Frontend URL |
| `AUTH_SESSION_DURATION_HOURS` | `24` | Session duration |
| `AUTH_REQUIRE_ORG_MEMBERSHIP` | `[]` | Required GitHub org membership (JSON array, e.g. `["my-org"]`) |
| `AUTH_REQUIRE_TEAM_MEMBERSHIP` | `[]` | Required team membership (JSON array, e.g. `["my-org/admins"]`) |
| `AUTH_REQUIRE_ENTERPRISE_ADMIN` | `false` | Require enterprise admin |
| `AUTH_REQUIRE_ENTERPRISE_SLUG` | `""` | Enterprise slug (if require_enterprise_admin is true) |
| `CORS_ALLOWED_ORIGINS` | `["*"]` | CORS origins (permissive for dev) |

**Common Azure Regions:**
- `eastus` - East US (Virginia)
- `westus2` - West US 2 (Washington)
- `centralus` - Central US (Iowa)
- `westeurope` - West Europe (Netherlands)
- `northeurope` - North Europe (Ireland)
- `southeastasia` - Southeast Asia (Singapore)
- `australiaeast` - Australia East (Sydney)

[See all available regions](https://azure.microsoft.com/en-us/explore/global-infrastructure/geographies/)

**Optional: GitHub App Variables** (for enhanced discovery/profiling)

| Variable Name | Value | Description |
|--------------|-------|-------------|
| `SOURCE_APP_ID` | `0` or your App ID | GitHub App ID for source (set to `0` if not using) |
| `SOURCE_APP_INSTALLATION_ID` | `0` or installation ID | Installation ID (or `0` to auto-discover) |
| `DEST_APP_ID` | `0` or your App ID | GitHub App ID for destination (optional) |
| `DEST_APP_INSTALLATION_ID` | `0` or installation ID | Installation ID (optional) |

#### Dev Environment Secrets

Click **Add secret** for each:

| Secret Name | Value | How to Get |
|------------|-------|------------|
| `SOURCE_GITHUB_TOKEN` | GitHub PAT | GitHub Settings → Developer settings → PAT |
| `DEST_GITHUB_TOKEN` | GitHub PAT | (can be same as source) |
| `AUTH_GITHUB_OAUTH_CLIENT_ID` | OAuth Client ID | GitHub OAuth App (dev) |
| `AUTH_GITHUB_OAUTH_CLIENT_SECRET` | OAuth Client Secret | GitHub OAuth App (dev) |
| `AUTH_SESSION_SECRET` | Random 32-char string | `openssl rand -base64 32` |

**Optional: GitHub App Secrets** (for enhanced discovery/profiling)

| Secret Name | Value | How to Get |
|------------|-------|------------|
| `SOURCE_APP_PRIVATE_KEY` | GitHub App private key PEM | GitHub App settings → Generate private key |
| `DEST_APP_PRIVATE_KEY` | GitHub App private key PEM | (if using separate app for destination) |

### Step 4: Configure Production Environment

Navigate to: **Settings → Environments → production**

#### Production Environment Variables

Click **Add variable** for each:

| Variable Name | Value | Description |
|--------------|-------|-------------|
| `ENVIRONMENT_NAME` | `production` | Environment identifier |
| `AZURE_LOCATION` | `eastus` | Azure region (eastus, westus2, etc.) |
| `APP_NAME_PREFIX` | `github-migrator` | App name prefix |
| `APP_SERVICE_NAME` | `github-migrator-prod` | **IMPORTANT**: Full App Service name (from Terraform output) |
| `APP_SERVICE_SKU` | `S1` | App Service tier (Standard) |
| `ALWAYS_ON` | `true` | Keep app always on |
| `DOCKER_IMAGE_TAG` | `prod` | Container image tag |
| `SOURCE_TYPE` | `github` | Source provider type |
| `SOURCE_BASE_URL` | `https://api.github.com` | Source API URL |
| `DESTINATION_TYPE` | `github` | Destination provider type |
| `DESTINATION_BASE_URL` | `https://api.github.com` | Destination API URL |
| `MIGRATION_WORKERS` | `5` | Number of workers (more for prod) |
| `MIGRATION_POLL_INTERVAL_SECONDS` | `30` | Poll interval |
| `MIGRATION_POST_MIGRATION_MODE` | `production_only` | Post-migration mode |
| `MIGRATION_DEST_REPO_EXISTS_ACTION` | `fail` | Action if repo exists |
| `MIGRATION_VISIBILITY_PUBLIC_REPOS` | `private` | Public repo visibility |
| `MIGRATION_VISIBILITY_INTERNAL_REPOS` | `private` | Internal repo visibility |
| `LOGGING_LEVEL` | `info` | Log level |
| `LOGGING_FORMAT` | `json` | Log format |
| `AUTH_ENABLED` | `true` | Enable authentication |
| `AUTH_CALLBACK_URL` | `https://github-migrator-prod.azurewebsites.net/api/v1/auth/callback` | OAuth callback URL |
| `AUTH_FRONTEND_URL` | `https://github-migrator-prod.azurewebsites.net` | Frontend URL |
| `AUTH_SESSION_DURATION_HOURS` | `24` | Session duration |
| `AUTH_REQUIRE_ORG_MEMBERSHIP` | `["your-org"]` | Required GitHub org membership (JSON array) |
| `AUTH_REQUIRE_TEAM_MEMBERSHIP` | `["your-org/migration-admins"]` | Required team membership (JSON array) |
| `AUTH_REQUIRE_ENTERPRISE_ADMIN` | `false` | Require enterprise admin |
| `AUTH_REQUIRE_ENTERPRISE_SLUG` | `""` | Enterprise slug (if require_enterprise_admin is true) |
| `CORS_ALLOWED_ORIGINS` | `["https://github-migrator-prod.azurewebsites.net"]` | CORS origins (restrictive) |

#### Production-Specific Database Variables

| Variable Name | Value | Description |
|--------------|-------|-------------|
| `DATABASE_NAME` | `migrator` | Database name |
| `DATABASE_ADMIN_USERNAME` | `psqladmin` | Database admin user |
| `POSTGRES_VERSION` | `15` | PostgreSQL version |
| `DATABASE_SKU` | `GP_Standard_D2s_v3` | Database SKU |
| `DATABASE_STORAGE_MB` | `32768` | Storage in MB (32GB) |
| `DATABASE_BACKUP_RETENTION_DAYS` | `30` | Backup retention |
| `DATABASE_GEO_REDUNDANT_BACKUP` | `true` | Geo-redundant backups |
| `DATABASE_HIGH_AVAILABILITY_MODE` | `ZoneRedundant` | HA mode |

#### Production Environment Secrets

Click **Add secret** for each:

| Secret Name | Value | How to Get |
|------------|-------|------------|
| `SOURCE_GITHUB_TOKEN` | GitHub PAT | (separate from dev) |
| `DEST_GITHUB_TOKEN` | GitHub PAT | (can be same as source) |
| `AUTH_GITHUB_OAUTH_CLIENT_ID` | OAuth Client ID | GitHub OAuth App (production) |
| `AUTH_GITHUB_OAUTH_CLIENT_SECRET` | OAuth Client Secret | GitHub OAuth App (production) |
| `AUTH_SESSION_SECRET` | Random 32-char string | `openssl rand -base64 32` (different from dev!) |

**Optional: GitHub App Secrets** (for enhanced discovery/profiling)

| Secret Name | Value | How to Get |
|------------|-------|------------|
| `SOURCE_APP_PRIVATE_KEY` | GitHub App private key PEM | GitHub App settings → Generate private key |
| `DEST_APP_PRIVATE_KEY` | GitHub App private key PEM | (if using separate app for destination) |

## 📊 Variables vs Secrets

### Use **Variables** for:
- ✅ Non-sensitive configuration (SKU sizes, worker counts)
- ✅ URLs (callback URLs, base URLs)
- ✅ Feature flags (booleans like `AUTH_ENABLED`)
- ✅ Resource names (when not sensitive)

**Advantage**: Visible in workflow logs for debugging

### Use **Secrets** for:
- 🔐 Tokens and passwords
- 🔐 OAuth client secrets
- 🔐 Session secrets
- 🔐 Database credentials
- 🔐 API keys

**Advantage**: Encrypted and never visible in logs

## 🎯 Quick Setup Script

> ⚠️ **CRITICAL**: `APP_SERVICE_NAME` must be updated after Terraform runs. Start with placeholder value, then update with actual name from Terraform output. Deployments will fail without this!

Here's a checklist format for faster setup:

### Dev Environment

**Variables:**
```
☐ ENVIRONMENT_NAME = dev
☐ AZURE_LOCATION = eastus
☐ APP_NAME_PREFIX = github-migrator
☐ APP_SERVICE_NAME = github-migrator-dev  # ⚠️ UPDATE AFTER TERRAFORM - see note below
☐ APP_SERVICE_SKU = B1
☐ ALWAYS_ON = false
☐ DOCKER_IMAGE_TAG = dev
☐ SOURCE_TYPE = github
☐ SOURCE_BASE_URL = https://api.github.com
☐ DESTINATION_TYPE = github
☐ DESTINATION_BASE_URL = https://api.github.com
☐ MIGRATION_WORKERS = 3
☐ MIGRATION_POLL_INTERVAL_SECONDS = 30
☐ MIGRATION_POST_MIGRATION_MODE = production_only
☐ MIGRATION_DEST_REPO_EXISTS_ACTION = fail
☐ MIGRATION_VISIBILITY_PUBLIC_REPOS = private
☐ MIGRATION_VISIBILITY_INTERNAL_REPOS = private
☐ LOGGING_LEVEL = info
☐ LOGGING_FORMAT = json
☐ AUTH_ENABLED = false
☐ AUTH_CALLBACK_URL = https://github-migrator-dev.azurewebsites.net/api/v1/auth/callback
☐ AUTH_FRONTEND_URL = https://github-migrator-dev.azurewebsites.net
☐ AUTH_SESSION_DURATION_HOURS = 24
☐ AUTH_REQUIRE_ORG_MEMBERSHIP = []
☐ AUTH_REQUIRE_TEAM_MEMBERSHIP = []
☐ AUTH_REQUIRE_ENTERPRISE_ADMIN = false
☐ AUTH_REQUIRE_ENTERPRISE_SLUG = ""
☐ CORS_ALLOWED_ORIGINS = ["*"]
```

**Secrets:**
```
☐ SOURCE_GITHUB_TOKEN
☐ DEST_GITHUB_TOKEN
☐ AUTH_GITHUB_OAUTH_CLIENT_ID (if auth enabled)
☐ AUTH_GITHUB_OAUTH_CLIENT_SECRET (if auth enabled)
☐ AUTH_SESSION_SECRET (if auth enabled)
```

### Production Environment

**Additional Variables:**
```
☐ Same as dev but with these changes:
  - ENVIRONMENT_NAME = production
  - AZURE_LOCATION = eastus  # Or your preferred region
  - APP_SERVICE_NAME = github-migrator-prod  # ⚠️ UPDATE AFTER TERRAFORM - see note below
  - APP_SERVICE_SKU = S1
  - ALWAYS_ON = true
  - DOCKER_IMAGE_TAG = prod
  - MIGRATION_WORKERS = 5
  - AUTH_ENABLED = true
  - AUTH_CALLBACK_URL = https://github-migrator-prod.azurewebsites.net/api/v1/auth/callback
  - AUTH_FRONTEND_URL = https://github-migrator-prod.azurewebsites.net
  - AUTH_REQUIRE_ORG_MEMBERSHIP = ["your-org"]  # EXAMPLE: Restrict to your org
  - AUTH_REQUIRE_TEAM_MEMBERSHIP = ["your-org/migration-admins"]  # EXAMPLE: Restrict to team
  - AUTH_REQUIRE_ENTERPRISE_ADMIN = false
  - AUTH_REQUIRE_ENTERPRISE_SLUG = ""
  - CORS_ALLOWED_ORIGINS = ["https://github-migrator-prod.azurewebsites.net"]

☐ DATABASE_NAME = migrator
☐ DATABASE_ADMIN_USERNAME = psqladmin
☐ POSTGRES_VERSION = 15
☐ DATABASE_SKU = GP_Standard_D2s_v3
☐ DATABASE_STORAGE_MB = 32768
☐ DATABASE_BACKUP_RETENTION_DAYS = 30
☐ DATABASE_GEO_REDUNDANT_BACKUP = true
☐ DATABASE_HIGH_AVAILABILITY_MODE = ZoneRedundant
```

**Secrets:** (separate from dev!)
```
☐ SOURCE_GITHUB_TOKEN
☐ DEST_GITHUB_TOKEN
☐ AUTH_GITHUB_OAUTH_CLIENT_ID
☐ AUTH_GITHUB_OAUTH_CLIENT_SECRET
☐ AUTH_SESSION_SECRET
```

## 🎯 Getting App Service Names

After running Terraform, you'll need to add the actual app service names to your environments.

**Get the app service names:**
```bash
# For dev
cd terraform/environments/dev
terraform output app_service_name

# For prod  
cd terraform/environments/prod
terraform output app_service_name
```

**Add to environments:**
1. Copy the output value (e.g., `github-migrator-dev-abc123`)
2. Go to **Settings → Environments → dev** (or production)
3. Click **Variables** tab
4. Find `APP_SERVICE_NAME` and click **Edit** (or add if missing)
5. Update the value with the full name from Terraform output
6. Click **Update variable**

> 💡 **Tip**: The Terraform workflow shows this output automatically after running. You can also find it in Azure Portal under App Services.

## 🔄 How Workflows Use Environments

Workflows reference environment variables and secrets like this:

```yaml
jobs:
  terraform-dev:
    environment: dev  # ← Specifies which environment to use
    steps:
      - name: Use environment variable
        run: echo "${{ vars.APP_NAME_PREFIX }}"  # ← vars. for variables
      
      - name: Use environment secret
        run: echo "${{ secrets.SOURCE_GITHUB_TOKEN }}"  # ← secrets. for secrets
```

**Priority:**
1. Environment secrets/variables (highest priority)
2. Repository secrets/variables
3. Organization secrets/variables (if applicable)

## ✅ Verification

After setup, verify by running:

1. **Actions → Terraform Deploy - Dev → Run workflow**
2. Select action: `plan`
3. Check logs to see variables being used
4. Verify no errors about missing variables

## 🎓 Best Practices

1. **Keep Sensitive Data in Secrets**
   - Even if it seems "not that sensitive", use secrets for tokens/keys

2. **Use Different Credentials Per Environment**
   - Separate OAuth apps for dev/prod
   - Different session secrets
   - Ideally different GitHub PATs

3. **Document Custom Values**
   - If you use custom values, document why

4. **Test in Dev First**
   - Always test configuration changes in dev
   - Then replicate to production

5. **Regular Review**
   - Audit secrets quarterly
   - Rotate credentials regularly
   - Remove unused environments

## 🐛 Troubleshooting

### Variable Not Found

**Error:** `The template is not valid`

**Solution:**
- Verify variable name matches exactly (case-sensitive)
- Check variable is in correct environment (dev vs production)
- Ensure environment is specified in workflow: `environment: dev`

### Secret Not Found

**Error:** Empty or null values in terraform.tfvars

**Solution:**
- Verify secret exists in environment
- Check secret name matches workflow reference
- Secrets are environment-specific, not repository-wide

### Wrong Environment Used

**Problem:** Dev settings used in production

**Solution:**
- Check workflow specifies `environment: production`
- Verify you're running correct workflow
- Check environment selector in workflow dispatch UI

## 📚 Additional Resources

- [GitHub Environments Documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)
- [Environment Protection Rules](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment#environment-protection-rules)
- [Environment Variables and Secrets](https://docs.github.com/en/actions/learn-github-actions/variables)

## 🎉 You're Done!

Your environments are now configured with:
- ✅ Proper secret isolation
- ✅ Clear configuration separation
- ✅ Protection rules for production
- ✅ Easy-to-update variables
- ✅ Better security posture

Next: Run your Terraform workflows and watch them use environment-specific configurations! 🚀

