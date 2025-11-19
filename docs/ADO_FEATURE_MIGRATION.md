# Azure DevOps Feature Migration Guide

This guide provides a comprehensive matrix of Azure DevOps features and their migration status to GitHub Enterprise Cloud using GitHub Enterprise Importer (GEI).

## 📊 Feature Migration Matrix

### Repository & Version Control

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Git Repositories** | ✅ Yes | GitHub Repositories | Full commit history migrates | 0 |
| **TFVC Repositories** | ❌ No | N/A | Must convert to Git first using git-tfs | +50 (BLOCKING) |
| **Branches** | ✅ Yes | Branches | All branches migrate | 0 |
| **Tags** | ✅ Yes | Tags | All tags migrate | 0 |
| **Commit History** | ✅ Yes | Commit History | Full history preserved | 0 |
| **Git LFS** | ✅ Yes | Git LFS | LFS objects migrate | +2 |
| **Submodules** | ✅ Yes | Submodules | Submodule references preserved | +2 |

### Pull Requests

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Pull Requests** | ✅ Yes | Pull Requests | Open and closed PRs migrate | 0 |
| **PR Comments** | ✅ Yes | PR Comments | All comments preserved | 0 |
| **PR Reviewers** | ✅ Yes | PR Reviewers | Reviewer history maintained | 0 |
| **PR Approvals** | ✅ Yes | PR Reviews | Approval status preserved | 0 |
| **PR Attachments** | ✅ Yes | PR Attachments | File attachments migrate | 0 |
| **Work Item Links on PRs** | ✅ Yes | Issue Links | Links preserved as references | 0 |
| **PR Status Checks** | ⚠️ Partial | Status Checks | Build status migrates, but not check definitions | +1 |

### Work Items & Boards

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Work Items** | ❌ No | GitHub Issues | Work items do NOT migrate | +3 (if active) |
| **Work Item Types** | ❌ No | Issue Labels | Must recreate as labels/templates | N/A |
| **Work Item States** | ❌ No | Issue States | Open/Closed only in GitHub | N/A |
| **Work Item Links** | ❌ No | Issue Links | Only PR links migrate | N/A |
| **Boards/Sprints** | ❌ No | Projects | Must recreate in GitHub Projects | N/A |
| **Queries** | ❌ No | Saved Searches | Must recreate using GitHub search syntax | N/A |

**Recommendation**: Use third-party tools like [Azure DevOps Migration Tools](https://github.com/nkdAgility/azure-devops-migration-tools) for work item migration.

### Pipelines & CI/CD

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **YAML Pipelines** | ⚠️ Partial | GitHub Actions | Files migrate as source code, not as workflows | +3 (if active) |
| **Classic Pipelines** | ❌ No | GitHub Actions | Must manually recreate | +5 per pipeline |
| **Pipeline History** | ❌ No | N/A | Execution history does not migrate | N/A |
| **Pipeline Runs** | ❌ No | N/A | Build/run history lost | N/A |
| **Build Artifacts** | ❌ No | Actions Artifacts | Only source files migrate | +1 |
| **Service Connections** | ❌ No | Secrets/Variables | Must recreate in GitHub | +3 |
| **Variable Groups** | ❌ No | Secrets/Variables | Must recreate as repository or organization secrets | +1 |
| **Self-hosted Agents** | ❌ No | Self-hosted Runners | Must set up new runners in GitHub | +3 |
| **Deployment Environments** | ❌ No | Environments | Must recreate in GitHub | +2 |

**Recommendation**: Use [GitHub Actions Importer](https://docs.github.com/en/actions/migrating-to-github-actions/automated-migrations/migrating-from-azure-devops-with-github-actions-importer) to help convert Azure Pipelines to GitHub Actions.

### Branch Policies & Protection

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Required Reviewers** | ✅ Yes | Required Reviewers | Repository-level policies migrate | +1 |
| **Build Validation** | ⚠️ Partial | Status Checks | Policy migrates, but builds must be recreated | +1 |
| **Comment Requirements** | ⚠️ Partial | Must recreate | May need manual configuration | +1 |
| **Work Item Linking** | ❌ No | N/A | Not supported in GitHub | N/A |
| **User-scoped Policies** | ❌ No | N/A | Only repository-level policies migrate | N/A |
| **Cross-repo Policies** | ❌ No | N/A | Not supported | N/A |

### Documentation & Wiki

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Azure Repos Wikis** | ❌ No | GitHub Wiki | Different wiki systems - requires manual migration | +2 per 10 pages |
| **Wiki Pages** | ❌ No | Wiki Pages | Content must be manually migrated | N/A |
| **Wiki Attachments** | ❌ No | Wiki Attachments | Must be manually uploaded | N/A |
| **README.md** | ✅ Yes | README.md | Migrates as source code | 0 |

**Recommendation**: Export wiki as markdown and manually import to GitHub wiki or migrate to docs/ folder in repository.

### Testing

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Test Plans** | ❌ No | N/A | No GitHub equivalent | +2 (if exists) |
| **Test Suites** | ❌ No | N/A | Must use third-party tools | N/A |
| **Test Cases** | ❌ No | N/A | Manual recreation required | N/A |
| **Test Results** | ❌ No | N/A | Historical results lost | N/A |
| **Test Attachments** | ❌ No | N/A | Must be migrated separately | N/A |

**Recommendation**: Consider third-party test management solutions like TestRail, Zephyr, or qTest that integrate with GitHub.

### Packages & Artifacts

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Package Feeds** | ❌ No | GitHub Packages | Requires separate migration | +3 |
| **NuGet Packages** | ❌ No | GitHub Packages | Must republish to GitHub Packages | N/A |
| **npm Packages** | ❌ No | GitHub Packages | Must republish | N/A |
| **Maven Packages** | ❌ No | GitHub Packages | Must republish | N/A |
| **Universal Packages** | ❌ No | N/A | No direct equivalent | N/A |

**Recommendation**: Use package registry migration tools or republish packages to GitHub Packages.

### Integrations & Webhooks

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Service Hooks** | ❌ No | Webhooks | Must recreate as GitHub webhooks | +1 |
| **Extensions** | ❌ No | GitHub Apps | Must find equivalent GitHub Apps | N/A |
| **OAuth Apps** | ❌ No | OAuth Apps | Must create new GitHub OAuth apps | N/A |

### Security & Compliance

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **GitHub Advanced Security for ADO** | ⚠️ Partial | GitHub Advanced Security | Must enable GHAS in GitHub after migration | +1 |
| **Code Scanning Results** | ❌ No | Code Scanning | Historical scans lost, must rescan | N/A |
| **Secret Scanning** | ❌ No | Secret Scanning | Must enable in GitHub | N/A |
| **Permissions** | ❌ No | Teams/Collaborators | Must recreate permission structure | N/A |

### Releases & Deployment

| Azure DevOps Feature | Migrates? | GitHub Equivalent | Notes | Complexity Points |
|---------------------|-----------|-------------------|-------|-------------------|
| **Release Pipelines** | ❌ No | GitHub Actions | Must recreate as workflows | +3 |
| **Release History** | ❌ No | N/A | Historical releases lost | N/A |
| **Release Approvals** | ❌ No | Environment Protection | Must recreate approval rules | N/A |
| **Release Gates** | ❌ No | N/A | Must implement in workflows | N/A |

## 🎯 Migration Complexity Scoring

The GitHub Migrator automatically calculates a complexity score based on detected features:

| Feature | Points | Impact |
|---------|--------|--------|
| TFVC Repository | +50 | **BLOCKING** - Must convert to Git first |
| Classic Pipelines | +5 per pipeline | Manual recreation required |
| Package Feeds | +3 | Separate migration process |
| Service Connections | +3 | Must recreate in GitHub |
| Active Pipelines | +3 | CI/CD reconfiguration needed |
| Active Work Items | +3 | Work items don't migrate |
| Wiki Pages | +2 per 10 pages | Manual migration needed |
| Test Plans | +2 | No GitHub equivalent |
| Variable Groups | +1 | Convert to GitHub secrets |
| Service Hooks | +1 | Recreate webhooks |
| Many PRs (>50) | +2 | Metadata migration time |
| Branch Policies | +1 | Need validation/recreation |

### Complexity Thresholds

- **0-5 points**: Low complexity - straightforward migration
- **6-15 points**: Medium complexity - some manual work required
- **16-30 points**: High complexity - significant manual effort
- **31-49 points**: Very high complexity - extensive preparation needed
- **50+ points**: Blocking - requires remediation before migration (typically TFVC)

## 📋 Pre-Migration Checklist

### Required Actions

- [ ] **TFVC Conversion**: If using TFVC, convert repositories to Git
- [ ] **Pipeline Assessment**: Review classic pipelines and plan GitHub Actions migration
- [ ] **Work Items**: Plan separate work item migration strategy
- [ ] **Wiki Content**: Export and plan wiki migration
- [ ] **Test Plans**: Identify third-party test management solution
- [ ] **Package Feeds**: Plan package migration to GitHub Packages
- [ ] **Service Connections**: Document and plan to recreate in GitHub
- [ ] **Variable Groups**: Export variables for recreation as GitHub secrets

### Recommended Actions

- [ ] Review branch policies and plan GitHub branch protection rules
- [ ] Export Azure Boards work items for reference
- [ ] Document custom pipeline tasks for GitHub Actions recreation
- [ ] Inventory service hooks and plan webhook recreation
- [ ] Review team permissions and plan GitHub teams structure
- [ ] Identify installed extensions and find GitHub App equivalents

## 🔄 Migration Process

### 1. Discovery & Profiling

Run discovery to automatically profile your Azure DevOps repositories:

```bash
curl -X POST http://localhost:8080/api/v1/ado/discover \
  -H "Content-Type: application/json" \
  -d '{
    "organization": "your-org",
    "workers": 5
  }'
```

The migrator will detect:
- Repository type (Git vs TFVC)
- Pipeline types (YAML vs Classic)
- Active pipelines and recent runs
- Work items and boards usage
- Wiki pages
- Test plans
- Package feeds
- Service connections and variable groups
- Branch policies
- And more...

### 2. Review Complexity Scores

Check complexity scores and feature usage:

```bash
curl http://localhost:8080/api/v1/repositories?source=azuredevops
```

### 3. Handle Blocking Issues

Address any repositories with complexity score ≥50 (typically TFVC):

```bash
# Convert TFVC to Git using git-tfs
git tfs clone https://dev.azure.com/org/project $/Project/Repo

# Re-discover after conversion
```

### 4. Plan Manual Migrations

For features that don't migrate automatically:
- Classic pipelines → Convert using GitHub Actions Importer
- Work items → Export and import using migration tools
- Wikis → Export markdown and manually recreate
- Test plans → Set up third-party test management
- Package feeds → Republish to GitHub Packages

### 5. Execute Migration

Create a batch and migrate:

```bash
curl -X POST http://localhost:8080/api/v1/batches \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ADO Migration Wave 1",
    "destination_org": "github-org",
    "repository_ids": [1, 2, 3]
  }'

curl -X POST http://localhost:8080/api/v1/batches/{id}/migrate
```

### 6. Post-Migration Tasks

After migration completes:
- [ ] Verify branch policies migrated correctly
- [ ] Recreate GitHub Actions workflows from YAML pipelines
- [ ] Set up GitHub Actions secrets and variables
- [ ] Migrate wiki content manually
- [ ] Set up GitHub Packages feeds
- [ ] Recreate webhooks
- [ ] Enable GitHub Advanced Security
- [ ] Validate PR migration and work item links
- [ ] Update team permissions

## 📚 Additional Resources

- [GitHub Enterprise Importer Documentation](https://docs.github.com/en/migrations/using-github-enterprise-importer)
- [Migrating from Azure DevOps to GitHub](https://docs.github.com/en/migrations/using-github-enterprise-importer/migrating-from-azure-devops-to-github-enterprise-cloud)
- [GitHub Actions Importer for Azure Pipelines](https://docs.github.com/en/actions/migrating-to-github-actions/automated-migrations/migrating-from-azure-devops-with-github-actions-importer)
- [Azure DevOps Migration Tools (for work items)](https://github.com/nkdAgility/azure-devops-migration-tools)
- [Converting TFVC to Git](https://docs.microsoft.com/en-us/devops/develop/git/centralized-to-git)

## 🆘 Support

For questions or issues:
1. Check the [Troubleshooting Guide](./ADO_SETUP_GUIDE.md#troubleshooting)
2. Review [GitHub Discussions](https://github.com/github/gh-migrations/discussions)
3. Contact your GitHub Customer Success Manager

