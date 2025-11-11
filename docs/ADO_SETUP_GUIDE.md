# Azure DevOps Migration Setup Guide

This guide explains how to set up the GitHub Migrator for migrating repositories from Azure DevOps (ADO) to GitHub Enterprise Cloud using the GitHub Enterprise Importer (GEI).

## 📋 Overview

The GitHub Migrator supports Azure DevOps as a source provider, enabling migrations from ADO organizations and projects to GitHub Enterprise Cloud. The migrator leverages GEI's native ADO support to migrate:

- ✅ Git repositories (full history)
- ✅ Pull requests and PR history
- ✅ User history for PRs
- ✅ Work item links on PRs
- ✅ Attachments on PRs
- ✅ Branch policies

**Note**: TFVC (Team Foundation Version Control) repositories require conversion to Git before migration.

## 🔑 Prerequisites

### 1. Azure DevOps Requirements

- Access to an Azure DevOps organization
- Azure DevOps Personal Access Token (PAT) with the following scopes:
  - **Code (Read)**: Access repository data
  - **Build (Read)**: Access pipeline information
  - **Work Items (Read)**: Access work item data
  - **Project and Team (Read)**: Access project information
  - **Graph (Read)**: Access organizational structure

### 2. Microsoft Entra ID (Azure AD) OAuth App

For user authentication and authorization, you'll need to set up an OAuth application in Microsoft Entra ID:

1. Sign in to the [Azure Portal](https://portal.azure.com)
2. Navigate to **Microsoft Entra ID** → **App registrations** → **New registration**
3. Configure the app:
   - **Name**: GitHub Migrator (ADO)
   - **Supported account types**: Accounts in this organizational directory only
   - **Redirect URI**: `https://your-migrator-domain.com/api/v1/auth/entraid/callback`
4. After creation, note the **Application (client) ID** and **Directory (tenant) ID**
5. Create a client secret:
   - Navigate to **Certificates & secrets** → **New client secret**
   - Add a description and set expiration
   - **Copy the secret value immediately** (it won't be shown again)
6. Configure API permissions:
   - Navigate to **API permissions** → **Add a permission**
   - Select **Azure DevOps** → **Delegated permissions**
   - Add: `user_impersonation`
   - Click **Grant admin consent** for your organization

### 3. GitHub Enterprise Cloud Requirements

- GitHub Enterprise Cloud organization
- GitHub Personal Access Token with:
  - `repo` scope
  - `workflow` scope
  - `admin:org` scope (for migrations)
- Organization admin access

## 🔧 Configuration

### Server Configuration

Update your `config.yaml` or set environment variables:

```yaml
# Source Configuration (Azure DevOps)
source:
  type: azuredevops
  organization: your-ado-org-name  # ADO organization name
  base_url: https://dev.azure.com/your-ado-org-name
  token: ${SOURCE_ADO_TOKEN}  # Server-level ADO PAT

# Destination Configuration (GitHub)
destination:
  type: github
  base_url: https://api.github.com
  token: ${DEST_GITHUB_TOKEN}

# Authentication Configuration
auth:
  enabled: true
  
  # Entra ID OAuth for ADO user authentication
  entraid_enabled: true
  entraid_tenant_id: ${ENTRAID_TENANT_ID}
  entraid_client_id: ${ENTRAID_CLIENT_ID}
  entraid_client_secret: ${ENTRAID_CLIENT_SECRET}
  entraid_callback_url: https://your-migrator-domain.com/api/v1/auth/entraid/callback
  
  # ADO Organization URL for validation
  ado_organization_url: https://dev.azure.com/your-ado-org-name
  
  # Existing GitHub OAuth for GitHub-to-GitHub migrations (optional)
  github_oauth_client_id: ${GITHUB_OAUTH_CLIENT_ID}
  github_oauth_client_secret: ${GITHUB_OAUTH_CLIENT_SECRET}
  callback_url: https://your-migrator-domain.com/api/v1/auth/callback
  
  frontend_url: https://your-migrator-domain.com
  session_secret: ${AUTH_SESSION_SECRET}
  session_duration_hours: 24
```

### Environment Variables

Set these environment variables in your deployment:

```bash
# Azure DevOps Source
export SOURCE_ADO_TOKEN="your-ado-pat"

# Microsoft Entra ID OAuth
export ENTRAID_TENANT_ID="your-tenant-id"
export ENTRAID_CLIENT_ID="your-client-id"
export ENTRAID_CLIENT_SECRET="your-client-secret"

# GitHub Destination
export DEST_GITHUB_TOKEN="your-github-pat"

# Authentication
export AUTH_SESSION_SECRET="$(openssl rand -base64 32)"
```

## 🔍 Discovery

### Discover an Entire ADO Organization

```bash
curl -X POST http://localhost:8080/api/v1/ado/discover \
  -H "Content-Type: application/json" \
  -d '{
    "organization": "your-org",
    "workers": 5,
    "full_profile": true
  }'
```

This will:
1. List all projects in the ADO organization
2. For each project, discover all repositories
3. Profile each repository for:
   - Git vs. TFVC detection
   - Repository size and structure
   - Azure Boards usage
   - Azure Pipelines configuration
   - GitHub Advanced Security features
   - Branch policies
   - Pull request and work item counts

### Discover Specific Projects

```bash
curl -X POST http://localhost:8080/api/v1/ado/discover \
  -H "Content-Type: application/json" \
  -d '{
    "organization": "your-org",
    "projects": ["Project1", "Project2"],
    "workers": 3
  }'
```

### Check Discovery Status

```bash
curl http://localhost:8080/api/v1/ado/discovery/status?organization=your-org
```

Response:
```json
{
  "total_repositories": 150,
  "total_projects": 5,
  "tfvc_repositories": 10,
  "git_repositories": 140,
  "status_breakdown": {
    "pending": 120,
    "profiled": 20,
    "ready": 10
  }
}
```

### List ADO Projects

```bash
curl http://localhost:8080/api/v1/ado/projects?organization=your-org
```

## 🚀 Migration Process

### 1. Review Discovered Repositories

```bash
curl http://localhost:8080/api/v1/repositories?source=azuredevops
```

### 2. Handle TFVC Repositories

TFVC repositories are marked with `ado_is_git: false` and require remediation:

- **Option A**: Convert to Git in ADO first, then re-discover
- **Option B**: Exclude from migration and handle separately
- **Option C**: Use the `git-tfs` tool for manual conversion

Mark TFVC repos as "wont_migrate" if not converting:

```bash
curl -X POST http://localhost:8080/api/v1/repositories/{id}/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "wont_migrate",
    "reason": "TFVC repository requires conversion"
  }'
```

### 3. Create a Migration Batch

```bash
curl -X POST http://localhost:8080/api/v1/batches \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ADO Project1 Migration",
    "description": "Migrating Project1 repositories from ADO",
    "destination_org": "github-org-name",
    "repository_ids": [1, 2, 3, 4, 5]
  }'
```

### 4. Run a Dry Run

```bash
curl -X POST http://localhost:8080/api/v1/batches/{batch_id}/dry-run
```

### 5. Execute Migration

```bash
curl -X POST http://localhost:8080/api/v1/batches/{batch_id}/migrate
```

## 📊 Complexity Scoring

ADO repositories are scored based on migration complexity:

| Factor | Points | Description |
|--------|--------|-------------|
| **TFVC** | +50 | Blocking - requires conversion |
| **Azure Boards** | +3 | Work items don't migrate |
| **Azure Pipelines** | +3 | Pipeline history doesn't migrate |
| **Large PRs** | +2 | Many PRs increase complexity |
| **Branch Policies** | +1 | Need manual recreation |
| **Standard Git Factors** | Variable | Size, LFS, submodules, etc. |

**Remediation Required**: Repositories with TFVC (50+ points) need conversion before migration.

## 🔐 Authorization Model

### Application Access

Users authenticate via Entra ID OAuth. Configure organization membership requirements:

```yaml
auth:
  # Require users to be members of specific ADO teams or groups
  ado_require_organization_membership: true
  ado_organization: your-org
```

### Repository Access

After authentication, users can migrate repositories based on their ADO permissions:

- **Organization Admins**: Can migrate any repository
- **Project Admins**: Can migrate repositories in their projects
- **Repository Contributors**: Can migrate repositories they have write access to

## 📈 Monitoring

### Check Migration Progress

```bash
curl http://localhost:8080/api/v1/batches/{batch_id}/status
```

### View Migration Logs

```bash
curl http://localhost:8080/api/v1/repositories/{id}/logs
```

### Analytics Dashboard

Access the web UI at `https://your-migrator-domain.com` to view:

- Migration progress by project
- TFVC vs. Git repository breakdown
- Complexity distribution
- Feature usage statistics (Boards, Pipelines, GHAS)

## 🐛 Troubleshooting

### TFVC Repository Detected

**Problem**: Repository is marked as "remediation_required" with TFVC flag.

**Solution**: Convert to Git in ADO:
```bash
# In ADO, import to a new Git repository
# Or use git-tfs tool for local conversion
```

### Authentication Failed

**Problem**: "failed to validate credentials" error during discovery.

**Check**:
1. ADO PAT has required scopes
2. PAT hasn't expired
3. Organization name is correct in base URL

### Azure Boards Data Missing

**Note**: Azure Boards work items are not migrated by GEI. Only work item links on PRs are preserved.

**Recommendation**: Export work items separately or use Azure DevOps migration tools for work item migration.

### Pipeline History Not Migrated

**Note**: GEI doesn't migrate Azure Pipelines execution history.

**Recommendation**: 
1. Keep ADO organization active for historical reference
2. Recreate pipelines as GitHub Actions in destination
3. Document pipeline configurations before migration

## 📚 Additional Resources

- [GitHub Enterprise Importer Documentation](https://docs.github.com/en/migrations/using-github-enterprise-importer)
- [Migrating from Azure DevOps to GitHub](https://docs.github.com/en/migrations/using-github-enterprise-importer/migrating-from-azure-devops-to-github-enterprise-cloud)
- [Azure DevOps REST API Reference](https://learn.microsoft.com/en-us/rest/api/azure/devops/)
- [Microsoft Entra ID OAuth](https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/entra)

## ✅ Migration Checklist

- [ ] Create ADO Personal Access Token with required scopes
- [ ] Set up Entra ID OAuth App for user authentication
- [ ] Configure server-level ADO PAT
- [ ] Discover ADO organization/projects
- [ ] Review TFVC repositories and plan conversion
- [ ] Review complexity scores
- [ ] Create migration batches by project
- [ ] Run dry runs to test migrations
- [ ] Execute production migrations
- [ ] Validate migrated repositories
- [ ] Update team documentation with new GitHub URLs
- [ ] Recreate branch policies in GitHub
- [ ] Set up GitHub Actions to replace Azure Pipelines

