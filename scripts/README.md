# Test Repository Creation Scripts

This directory contains scripts for creating test repositories to verify the migrator's discovery features for both GitHub and Azure DevOps.

## Directory Structure

- **GitHub/** - Scripts for creating GitHub test repositories
- **ADO/** - Scripts for creating Azure DevOps test repositories

## GitHub/create-test-repos.go

A Go script that creates multiple test repositories in your GitHub organization, each configured to test different aspects of the discovery feature.

### Test Repositories Created

The script creates 20 different test repositories covering various use cases:

1. **test-minimal-empty** - Empty repository with no commits
2. **test-minimal-basic** - Basic repository with README only
3. **test-small-repo** - Small repository with multiple files and branches
4. **test-with-actions** - Repository with GitHub Actions workflows
5. **test-with-wiki** - Repository with Wiki enabled
6. **test-with-pages** - Repository configured for GitHub Pages
7. **test-with-lfs** - Repository using Git LFS
8. **test-with-submodules** - Repository with Git submodules
9. **test-many-branches** - Repository with multiple branches
10. **test-with-protection** - Repository with branch protection rules
11. **test-with-releases** - Repository with releases and tags
12. **test-with-issues-prs** - Repository with issues and pull requests
13. **test-with-tags** - Repository with multiple tags
14. **test-with-codeowners** - Repository with CODEOWNERS file
15. **test-with-environments** - Repository with deployment workflow (environments)
16. **test-private-repo** - Private repository for visibility testing
17. **test-archived-repo** - Archived repository
18. **test-complex-all-features** - Complex repository with multiple features
19. **test-large-file-history** - Repository simulating large files
20. **test-many-commits** - Repository with many commits

### Features Tested

The test repositories cover:

**Git Properties:**
- Repository sizes (empty, small, large)
- Multiple branches and tags
- Commit history
- Git LFS usage
- Git submodules

**GitHub Features:**
- Visibility (public/private)
- Archived status
- Wiki (with/without content)
- GitHub Pages
- GitHub Actions/Workflows
- Projects
- Branch Protection
- Releases and Assets

**Security & Compliance:**
- CODEOWNERS files
- Branch protection rules

**Activity & Collaboration:**
- Issues (open/closed)
- Pull Requests
- Contributors
- Multiple commits

### Prerequisites

1. **Go** - Installed and configured
2. **GitHub Personal Access Token (PAT)** with the following permissions:
   - `repo` (full control of private repositories)
   - `admin:org` (if creating in an organization)
   - `delete_repo` (for cleanup operations)

### Usage

#### Create Test Repositories

```bash
# Set your GitHub token as an environment variable
export GITHUB_TOKEN="your_github_token_here"

# Run the script to create test repositories
go run scripts/GitHub/create-test-repos.go -org "your-org-name"

# Or provide token directly
go run scripts/GitHub/create-test-repos.go -org "your-org-name" -token "your_token"
```

#### Cleanup Test Repositories

To remove all test repositories (repositories starting with "test-"):

```bash
go run scripts/GitHub/create-test-repos.go -org "your-org-name" -cleanup
```

### Command-Line Flags

- `-org` (required) - GitHub organization name where repositories will be created
- `-token` (optional) - GitHub personal access token (defaults to `GITHUB_TOKEN` environment variable)
- `-cleanup` (optional) - Delete all test repositories instead of creating them

### Example Workflow

```bash
# 1. Set your GitHub token
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# 2. Create test repositories in your organization
go run scripts/GitHub/create-test-repos.go -org "my-test-org"

# 3. Run your discovery feature against the organization
./bin/github-migrator-server discover -org "my-test-org"

# 4. Verify the discovery results in the database or UI

# 5. Clean up test repositories when done
go run scripts/GitHub/create-test-repos.go -org "my-test-org" -cleanup
```

### Notes

- **Rate Limiting**: The script includes delays between API calls to respect GitHub's rate limits
- **Partial Failures**: If some repository setups fail, the script continues with the next one
- **Wiki Content**: Wiki repositories are created with the feature enabled, but actual wiki content must be added via git clone
- **Branch Creation**: Branches are created by adding files to them
- **Tag Creation**: Tags are created via releases for better API compatibility
- **Archived Repos**: Archived repositories must be unarchived before deletion

### Troubleshooting

**Permission Errors**
- Ensure your GitHub token has the necessary permissions
- Verify you have admin access to the organization

**Repository Already Exists**
- The script skips repositories that already exist
- Use `-cleanup` flag to remove existing test repositories first

**Rate Limiting**
- The script includes delays, but you may need to wait if you hit rate limits
- GitHub's rate limit for authenticated requests is typically 5,000 requests per hour

**API Errors**
- Some features (like Packages, Code Scanning) may not be available on all GitHub plans
- The script logs warnings for partial failures but continues execution

### Testing Discovery Feature

After creating test repositories, you can test your discovery feature by:

1. Running discovery against the organization
2. Verifying that all repository attributes are correctly detected:
   - Git properties (size, branches, commits, LFS, submodules)
   - GitHub features (wiki, pages, actions, discussions, projects)
   - Security features (CODEOWNERS, branch protection)
   - Activity metrics (issues, PRs, contributors)
   - Repository settings (visibility, archived status)

### Cleanup

Always clean up test repositories after testing to avoid clutter:

```bash
go run scripts/GitHub/create-test-repos.go -org "my-test-org" -cleanup
```

This will delete all repositories starting with "test-" in the specified organization.

---

## ADO/create-ado-test-repos.go

A Go script that creates multiple test repositories in your Azure DevOps project, each configured to test different aspects of the ADO discovery feature.

### Test Repositories Created

The script creates 15 different test repositories covering various ADO use cases:

1. **test-ado-minimal-empty** - Empty repository with no commits
2. **test-ado-basic-repo** - Basic repository with initial commit and files
3. **test-ado-with-yaml-pipeline** - Repository with YAML pipeline configuration
4. **test-ado-with-classic-pipeline** - Repository ready for Classic build pipeline
5. **test-ado-with-work-items** - Repository with Azure Boards work items (Tasks)
6. **test-ado-with-pull-requests** - Repository with active pull requests
7. **test-ado-with-branch-policies** - Repository configured for branch policies
8. **test-ado-with-wiki** - Repository with project wiki
9. **test-ado-many-branches** - Repository with multiple feature/release branches
10. **test-ado-many-commits** - Repository with commit history
11. **test-ado-complex-all-features** - Complex repository with multiple ADO features
12. **test-ado-shared-library** - Shared library repository (npm, Maven, NuGet)
13. **test-ado-frontend-app** - Frontend application depending on shared-library
14. **test-ado-backend-api** - Backend API depending on shared-library
15. **test-ado-monorepo** - Monorepo with internal package dependencies

### ADO Features Tested

The test repositories cover:

**Git Properties:**
- Repository initialization
- Multiple branches (feature, release, hotfix)
- Commit history
- File structure

**Azure DevOps Features:**
- YAML Pipelines (azure-pipelines.yml)
- Classic Build Pipelines (requires manual creation)
- Azure Boards Work Items (Tasks)
- Pull Requests (active and with work item links)
- Branch Policies (requires manual configuration)
- Project Wikis
- Multiple repositories in single project

**Dependency Testing:**
- Cross-repository dependencies (shared-library → frontend/backend)
- Monorepo with internal dependencies (package-a → package-b → package-c)
- Multiple package managers (npm, Maven, NuGet, pip, Go modules)
- Yarn workspaces and Lerna configurations

**Migration Considerations:**
- YAML pipelines (migrate as code)
- Classic pipelines (require recreation)
- Work items (require manual migration or third-party tools)
- Branch policies (require recreation in GitHub)
- Wikis (require manual migration)

### Prerequisites

1. **Go** - Installed and configured
2. **Azure DevOps Personal Access Token (PAT)** with the following permissions:
   - `Code` (Read & Write) - For repository operations
   - `Build` (Read & Execute) - For pipeline operations
   - `Work Items` (Read, Write, & Manage) - For work item creation
   - `Project and Team` (Read) - For project access
   - `Wiki` (Read & Write) - For wiki operations

### Usage

#### Create Test Repositories

```bash
# Set your Azure DevOps PAT as an environment variable
export AZURE_DEVOPS_PAT="your_ado_pat_here"

# Run the script to create test repositories
go run scripts/ADO/create-ado-test-repos.go \
  -org "https://dev.azure.com/your-org" \
  -project "YourProject"

# Or provide PAT directly
go run scripts/ADO/create-ado-test-repos.go \
  -org "https://dev.azure.com/your-org" \
  -project "YourProject" \
  -pat "your_pat"
```

#### Cleanup Test Repositories

To remove all test repositories (repositories starting with "test-ado-"):

```bash
go run scripts/ADO/create-ado-test-repos.go \
  -org "https://dev.azure.com/your-org" \
  -project "YourProject" \
  -cleanup
```

### Command-Line Flags

- `-org` (required) - Azure DevOps organization URL (e.g., https://dev.azure.com/myorg)
- `-project` (required) - Azure DevOps project name
- `-pat` (optional) - Azure DevOps personal access token (defaults to `AZURE_DEVOPS_PAT` environment variable)
- `-cleanup` (optional) - Delete all test repositories instead of creating them

### Example Workflow

```bash
# 1. Set your Azure DevOps PAT
export AZURE_DEVOPS_PAT="your_pat_here"

# 2. Create test repositories in your project
go run scripts/ADO/create-ado-test-repos.go \
  -org "https://dev.azure.com/my-org" \
  -project "TestProject"

# 3. Manually configure features that can't be automated:
#    - Create Classic Build Pipeline in Azure DevOps UI
#    - Add branch policies (required reviewers, build validation)
#    - Configure service connections, variable groups
#    - Add package feeds, test plans

# 4. Run your discovery feature against the organization
curl -X POST http://localhost:8080/api/v1/ado/discover \
  -H "Content-Type: application/json" \
  -d '{
    "organization": "my-org",
    "projects": ["TestProject"],
    "workers": 5
  }'

# 5. Verify the discovery results in the database or UI

# 6. Clean up test repositories when done
go run scripts/ADO/create-ado-test-repos.go \
  -org "https://dev.azure.com/my-org" \
  -project "TestProject" \
  -cleanup
```

### Notes

- **Rate Limiting**: The script includes delays between API calls to respect Azure DevOps rate limits
- **Partial Failures**: If some repository setups fail, the script continues with the next one
- **Manual Configuration**: Some features (Classic Pipelines, Branch Policies, Service Connections, Variable Groups, Package Feeds, Test Plans) require manual configuration in the Azure DevOps UI
- **YAML Pipelines**: The YAML file is created, but you need to create the pipeline in the UI to trigger builds
- **Work Items**: Created at the project level and can be linked to commits/PRs manually
- **Wikis**: Project wikis are created but may need content added manually

### Troubleshooting

**Permission Errors**
- Ensure your Azure DevOps PAT has the necessary permissions
- Verify you have admin or contributor access to the project

**Repository Already Exists**
- The script skips repositories that already exist
- Use `-cleanup` flag to remove existing test repositories first

**Rate Limiting**
- The script includes delays, but you may need to wait if you hit rate limits
- Azure DevOps has rate limits based on your organization's tier

**API Errors**
- Some features (Test Plans, Artifacts) may not be available on all Azure DevOps tiers
- The script logs warnings for partial failures but continues execution

### Testing ADO Discovery Feature

After creating test repositories, you can test your discovery feature by:

1. Running discovery against the organization and project
2. Verifying that all repository attributes are correctly detected:
   - Git properties (size, branches, commits)
   - Pipeline types (YAML vs Classic)
   - Azure Boards integration (work item counts)
   - Pull request details
   - Branch policy configurations
   - Wiki presence and page counts
   - Test plan counts (if configured)
   - Package feed counts (if configured)

### Cleanup

Always clean up test repositories after testing to avoid clutter:

```bash
go run scripts/ADO/create-ado-test-repos.go \
  -org "https://dev.azure.com/my-org" \
  -project "TestProject" \
  -cleanup
```

This will delete all repositories starting with "test-ado-" in the specified project.

