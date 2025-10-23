# Test Repository Creation Script

This directory contains scripts for creating test repositories to verify the GitHub migrator's discovery feature.

## create-test-repos.go

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
go run scripts/create-test-repos.go -org "your-org-name"

# Or provide token directly
go run scripts/create-test-repos.go -org "your-org-name" -token "your_token"
```

#### Cleanup Test Repositories

To remove all test repositories (repositories starting with "test-"):

```bash
go run scripts/create-test-repos.go -org "your-org-name" -cleanup
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
go run scripts/create-test-repos.go -org "my-test-org"

# 3. Run your discovery feature against the organization
./bin/github-migrator-server discover -org "my-test-org"

# 4. Verify the discovery results in the database or UI

# 5. Clean up test repositories when done
go run scripts/create-test-repos.go -org "my-test-org" -cleanup
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
go run scripts/create-test-repos.go -org "my-test-org" -cleanup
```

This will delete all repositories starting with "test-" in the specified organization.

