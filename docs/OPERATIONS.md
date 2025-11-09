# GitHub Migrator - Operations Runbook

## Table of Contents

- [Authentication Setup](#authentication-setup)
- [Visibility Handling Configuration](#visibility-handling-configuration)
- [Daily Operations](#daily-operations)
- [Migration Workflows](#migration-workflows)
- [Monitoring & Alerts](#monitoring--alerts)
- [Incident Response](#incident-response)
- [Maintenance Tasks](#maintenance-tasks)
- [Troubleshooting Guide](#troubleshooting-guide)
- [Runbooks](#runbooks)

---

## Authentication Setup

The GitHub Migrator supports two types of authentication:

1. **GitHub OAuth** - For controlling user access to the web UI (optional)
2. **GitHub App Authentication** - For API operations with better rate limits (optional, recommended for large migrations)

### GitHub App Authentication

GitHub App authentication provides significantly better rate limits for discovery and profiling operations while maintaining PAT-based authentication for migration operations (as required by GitHub's migration APIs).

#### Benefits of GitHub Apps

- **Higher Rate Limits**: 15,000 requests/hour per installation vs 5,000/hour shared with PAT
- **Better Isolation**: Per-organization tokens with proper scoping
- **Parallel Processing**: Multiple organizations can be processed simultaneously with their own tokens
- **Separation of Concerns**: Discovery uses App tokens, migrations use PAT

#### Two Operation Modes

##### Mode 1: With Installation ID (Simpler)

Best for:
- Single organization migrations
- GitHub Apps installed on one organization
- Testing and development
- Backwards compatibility

```yaml
source:
  token: "ghp_..." # PAT - required for migrations
  app_id: 123456
  app_private_key: "/path/to/private-key.pem"
  app_installation_id: 789012  # Provide for single-org mode
```

**How it works:**
- Uses the provided installation token for all API operations
- Single token used for discovery across enterprise (requires enterprise-level access)

##### Mode 2: Without Installation ID (Enterprise Multi-Org)

Best for:
- GitHub Enterprise Apps with multiple organizations
- Installations across many organizations
- Maximum rate limit efficiency
- Proper token isolation per organization

```yaml
source:
  token: "ghp_..." # PAT - required for migrations
  app_id: 123456
  app_private_key: "/path/to/private-key.pem"
  # app_installation_id omitted - system auto-discovers
```

**How it works:**
1. Uses JWT authentication to call GitHub App Installations API
2. Discovers all organizations where the app is installed
3. Creates per-org clients with org-specific installation tokens
4. Processes organizations in parallel (5 workers by default)
5. Each org uses its own token for complete isolation

#### Creating a GitHub App

**For GitHub.com:**
1. Navigate to your organization settings
2. Go to **Developer settings** > **GitHub Apps** > **New GitHub App**
3. Fill in the details:
   - **Name**: `GitHub Migrator Discovery`
   - **Homepage URL**: Your server URL
   - **Webhook**: Uncheck "Active" (not needed)
4. **Permissions** (Repository permissions):
   - **Contents**: Read-only (for cloning repositories)
   - **Metadata**: Read-only (for repository information)
   - **Administration**: Read-only (for settings)
5. **Where can this GitHub App be installed?**
   - Choose "Any account" or "Only on this account"
6. Click **Create GitHub App**
7. Generate and download private key (.pem file)
8. Note the **App ID**

**Installing the App:**
1. Go to app settings > **Install App**
2. Select your organization(s)
3. Choose **All repositories** or **Select repositories**
4. Note the **Installation ID** (found in URL: `/settings/installations/[ID]`)

**For GitHub Enterprise Server:**
- Similar process but navigate to `https://your-ghes.com/settings/apps/new`
- Requires enterprise admin access to create enterprise-level apps

#### Configuration Examples

**Environment Variables:**
```bash
# Source system with GitHub App
export GHMIG_SOURCE_APP_ID=123456
export GHMIG_SOURCE_APP_PRIVATE_KEY="/path/to/key.pem"
export GHMIG_SOURCE_APP_INSTALLATION_ID=789012  # Optional

# Or inline PEM (useful for containers):
export GHMIG_SOURCE_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"
```

**Config File:**
```yaml
source:
  type: github
  base_url: "https://api.github.com"
  token: "ghp_..." # REQUIRED for migrations
  
  # GitHub App for discovery (optional)
  app_id: 123456
  app_private_key: "./data/github-app-key.pem"
  # app_installation_id: 789012  # Omit for multi-org discovery
```

#### Operation Flow

**Enterprise Discovery with Multi-Org Mode:**
```
1. JWT Auth → List all app installations
2. Found: org1, org2, org3, org4, org5
3. Create 5 workers
4. Worker 1: org1 → get installation token → discover repos → profile
5. Worker 2: org2 → get installation token → discover repos → profile
6. Worker 3: org3 → get installation token → discover repos → profile
7. Worker 4: org4 → get installation token → discover repos → profile  
8. Worker 5: org5 → get installation token → discover repos → profile
```

**Single Repository Operations (Rediscovery, Pre-Migration):**
- System automatically creates org-specific client on-demand
- Uses JWT to get installation ID for repository's org
- Creates temporary client with that installation token
- Performs operation with proper authentication

#### Troubleshooting GitHub Apps

**"Bad credentials" errors:**
- Verify App ID is correct
- Ensure private key file path is accessible
- Check private key format (should start with `-----BEGIN RSA PRIVATE KEY-----`)
- Confirm app is installed on the target organization

**Slow discovery performance:**
- If using installation ID: Check that app has enterprise-level access
- If not using installation ID: Verify `app_installation_id` is omitted (set to 0 or removed)
- Check that multiple workers are being used (5 by default)

**"Installation not found" errors:**
- Verify app is installed on the organization
- Check installation ID is correct (found in GitHub UI)
- Ensure installation hasn't been suspended or removed

---

### GitHub OAuth (User Authentication)

The GitHub Migration Server also supports optional authentication using GitHub OAuth. When enabled, users must authenticate with GitHub and meet configurable authorization requirements to access the system.

**Important:** The OAuth App must be created on the **SOURCE** GitHub instance (where you're migrating FROM). Users authenticate against the source system, and authorization rules (organization membership, team membership, etc.) are validated against the source GitHub instance.

### Prerequisites

- GitHub organization or GitHub Enterprise account **on the SOURCE instance**
- Admin access to create OAuth Apps on the SOURCE instance
- SSL/TLS certificate (recommended for production)

### Creating a GitHub OAuth App

**Note:** Create this OAuth App on your SOURCE GitHub instance (GitHub.com, GHES, or GitHub with data residency).

#### For GitHub.com

1. Navigate to your organization's settings or your personal account settings
2. Go to **Developer settings** > **OAuth Apps** > **New OAuth App**
3. Fill in the application details:
   - **Application name**: `GitHub Migrator`
   - **Homepage URL**: Your server URL (e.g., `https://migrator.example.com`)
   - **Authorization callback URL**: `https://migrator.example.com/api/v1/auth/callback`
   - **Application description**: Optional description
4. Click **Register application**
5. Note the **Client ID** and generate a **Client Secret**

#### For GitHub Enterprise Server

1. Navigate to `https://your-ghes-instance.com/settings/applications/new`
2. Fill in the same details as above, adjusting URLs for your GHES instance
3. The authorization callback URL should be: `https://your-migrator.example.com/api/v1/auth/callback`

### Configuration

Add the following to your `configs/config.yaml`:

```yaml
# Source configuration - OAuth uses THIS base URL for authentication
source:
  type: github
  base_url: https://api.github.com  # OAuth app must be created on this GitHub instance
  token: ghp_source_token_here

# Destination configuration - NOT used for OAuth
destination:
  type: github
  base_url: https://api.github.com
  token: ghp_destination_token_here

# Authentication configuration
auth:
  enabled: true
  github_oauth_client_id: "Iv1.your_client_id_here"
  github_oauth_client_secret: "your_client_secret_here"
  callback_url: "https://migrator.example.com/api/v1/auth/callback"
  session_secret: "generate-a-random-secret-key-here"
  session_duration_hours: 24
  
  authorization_rules:
    # Require user to be member of these organizations (at least one)
    # These organizations are checked on the SOURCE GitHub instance
    require_org_membership:
      - "my-github-org"
    
    # Require user to be member of these teams (at least one)
    # Format: "org/team-slug"
    # These teams are checked on the SOURCE GitHub instance
    require_team_membership:
      - "my-github-org/migration-admins"
      - "my-github-org/platform-team"
    
    # Require user to be enterprise admin
    require_enterprise_admin: false
    
    # Enterprise slug (required if require_enterprise_admin is true)
    require_enterprise_slug: "my-enterprise"
```

**Key Points:**
- The OAuth App Client ID and Secret come from the OAuth App created on your **SOURCE** GitHub instance
- The `source.base_url` determines which GitHub instance users authenticate against
- Authorization rules (org/team membership) are validated against the SOURCE instance

### Generating Session Secret

Generate a secure random session secret:

```bash
# Using OpenSSL
openssl rand -base64 32

# Using Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"

# Using /dev/urandom
head -c 32 /dev/urandom | base64
```

### Authorization Rules

The system supports three types of authorization checks:

#### 1. Organization Membership

```yaml
authorization_rules:
  require_org_membership:
    - "my-org"
    - "another-org"
```

Users must be a member of at least one of the listed organizations.

#### 2. Team Membership

```yaml
authorization_rules:
  require_team_membership:
    - "my-org/platform-team"
    - "my-org/migration-admins"
```

Users must be a member of at least one of the listed teams. Format is `organization/team-slug`.

#### 3. Enterprise Admin

```yaml
authorization_rules:
  require_enterprise_admin: true
  require_enterprise_slug: "my-enterprise"
```

Users must be an enterprise administrator.

#### 4. Enterprise Membership

```yaml
authorization_rules:
  require_enterprise_membership: true
  require_enterprise_slug: "my-enterprise"
```

Users must be a member of the enterprise (any role, not just admin). This is more permissive than `require_enterprise_admin` and is useful for allowing all enterprise members to access the system. Repository-level permissions will still control what they can migrate.

**Note**: You can combine multiple rules. All configured rules must pass for a user to be authorized.

#### 5. Privileged Teams (Full Access)

```yaml
authorization_rules:
  privileged_teams:
    - "my-org/migration-admins"
    - "my-org/platform-team"
```

Members of privileged teams have **full access** to migrate any repository, bypassing per-repository permission checks. This is useful for dedicated migration admin teams.

### Repository-Level Permissions

Once a user passes the application-level authorization checks above, their repository access is determined by their GitHub permissions on the source instance:

#### Permission Levels

1. **Enterprise Admins** → Can migrate **all repositories**
   - Determined by `require_enterprise_admin` setting
   - Have unrestricted access to all migration operations

2. **Privileged Team Members** → Can migrate **all repositories**
   - Configured via `privileged_teams` setting
   - Bypass all per-repository permission checks
   - Useful for dedicated migration admin teams

3. **Organization Admins** → Can migrate **all repositories in their organizations**
   - Automatically detected via GitHub API
   - Admin role checked for each organization
   - Can migrate any repo in orgs they admin

4. **Repository Admins** → Can migrate **only repositories they have admin access to**
   - Default permission level for all other users
   - Each repository checked individually
   - Enables self-service migrations for developers

#### How It Works

1. **Repository List Filtering**: When users view the repository list, they only see repositories they have permission to migrate
2. **Migration Actions Protected**: All migration operations (batch creation, self-service migration, repository actions) validate permissions
3. **Real-Time Checks**: Permissions are checked at request time using the user's OAuth token
4. **Graceful Degradation**: If permission checks fail, users see appropriate error messages

#### Configuration Example

```yaml
auth:
  enabled: true
  # ... OAuth settings ...
  
  authorization_rules:
    # Application-level access (who can use the system)
    require_org_membership:
      - "my-company"
    
    # Privileged teams with full access
    privileged_teams:
      - "my-company/migration-admins"
      - "my-company/platform-team"
    
    # Optional: Require enterprise admin
    require_enterprise_admin: false
    require_enterprise_slug: "my-enterprise"
```

#### Use Cases

**Self-Service Developer Migrations:**
- Developers authenticate with their GitHub account
- They see only repos they admin
- Can migrate their own repositories
- No admin intervention needed

**Platform Team Full Access:**
- Add platform team to `privileged_teams`
- Team members see and can migrate all repositories
- Useful for handling complex migrations

**Org-Level Migration Coordinators:**
- Org admins automatically get access to all org repos
- Can coordinate migrations within their organization
- Don't need to be in privileged teams

### OAuth Base URL Configuration

By default, OAuth uses the **source** GitHub instance for authentication. This can be overridden:

```yaml
auth:
  github_oauth_base_url: "https://github.company.com/api/v3"
```

**Default behavior:**
- If source is GitHub → uses source base URL
- Otherwise → uses destination base URL

### Environment Variables

For sensitive configuration, use environment variables:

```bash
export GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID="Iv1.your_client_id"
export GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET="your_secret"
export GHMIG_AUTH_SESSION_SECRET="your_session_secret"
export GHMIG_AUTH_AUTHORIZATION_RULES_PRIVILEGED_TEAMS="my-org/migration-admins,my-org/platform-team"
```

### Testing Authentication

1. Start the server with authentication enabled
2. Navigate to `http://localhost:8080` (or your server URL)
3. You should be redirected to `/login`
4. Click "Sign in with GitHub"
5. Authorize the application
6. You should be redirected back to the dashboard

### Verifying Authorization

Test with different users to verify authorization rules:

```bash
# Check auth config endpoint
curl http://localhost:8080/api/v1/auth/config

# Try accessing protected endpoint without auth (should get 401)
curl -i http://localhost:8080/api/v1/repositories
```

### Disabling Authentication

To disable authentication and allow open access:

```yaml
auth:
  enabled: false
```

Or via environment variable:

```bash
export GHMIG_AUTH_ENABLED=false
```

### Troubleshooting Authentication

#### User sees "Access Denied"

Check the following:
1. User is a member of required organizations (verify in GitHub)
2. User is a member of required teams (check team membership)
3. Team slugs are correct (lowercase, hyphen-separated)
4. Organization names are correct

View server logs for authorization failures:

```bash
# Look for authorization check failures
docker logs github-migrator | grep "not authorized"

# Check what rules failed
docker logs github-migrator | grep "authorization"
```

#### OAuth callback fails

1. Verify callback URL in OAuth App matches `auth.callback_url` in config
2. Check that callback URL is accessible (not behind firewall)
3. Verify client ID and secret are correct
4. Check server logs for OAuth errors

#### Session expires immediately

1. Verify `session_secret` is set and consistent across restarts
2. Check `session_duration_hours` is set appropriately
3. For HTTPS deployments, ensure cookies are marked secure

#### Cannot access after enabling auth

1. Ensure you have a user account that meets the authorization requirements
2. Check logs for authorization failures
3. Temporarily disable auth to verify the issue:
   ```bash
   export GHMIG_AUTH_ENABLED=false
   ```

### Security Best Practices

1. **Use HTTPS**: Always use HTTPS in production for OAuth callbacks
2. **Secure Session Secret**: Use a strong, random session secret (32+ characters)
3. **Rotate Secrets**: Periodically rotate OAuth client secrets and session secrets
4. **Audit Access**: Monitor authentication logs for suspicious activity
5. **Principle of Least Privilege**: Only grant access to users who need it
6. **Team-Based Access**: Use team membership for fine-grained access control
7. **Session Duration**: Set appropriate session duration based on security requirements

---

## Daily Operations

### Morning Checklist

1. **Check System Health**
   ```bash
   curl http://localhost:8080/health
   ```
   Expected: `{"status": "healthy", "time": "..."}`

2. **Review Overnight Migrations**
   ```bash
   curl http://localhost:8080/api/v1/analytics/summary
   ```
   Check for:
   - Failed migrations (investigate if > 5%)
   - Completed migrations count
   - Repositories stuck in "migrating" status

3. **Check Logs for Errors**
   ```bash
   # Docker
   docker logs github-migrator --since 24h | grep ERROR
   
   # Systemd
   sudo journalctl -u github-migrator --since "24 hours ago" | grep ERROR
   
   # File
   tail -1000 logs/migrator.log | jq 'select(.level=="error")'
   ```

4. **Verify Disk Space**
   ```bash
   df -h /app/data
   df -h /app/logs
   ```
   Alert if > 80% used

5. **Check GitHub Rate Limits**
   ```bash
   # Source system
   curl -H "Authorization: token ${GITHUB_SOURCE_TOKEN}" \
     https://github.company.com/api/v3/rate_limit
   
   # Destination system
   curl -H "Authorization: token ${GITHUB_DEST_TOKEN}" \
     https://api.github.com/rate_limit
   ```

### End of Day Checklist

1. **Review Day's Migrations**
   ```bash
   curl "http://localhost:8080/api/v1/analytics/progress?days=1"
   ```

2. **Check for Stuck Migrations**
   ```bash
   curl "http://localhost:8080/api/v1/repositories?status=migrating"
   ```
   Investigate any migrations running > 2 hours

3. **Backup Database**
   ```bash
   # SQLite
   sqlite3 data/migrator.db ".backup data/backup-$(date +%Y%m%d).db"
   
   # PostgreSQL
   pg_dump migrator > backup-$(date +%Y%m%d).sql
   ```

4. **Review Logs Summary**
   ```bash
   # Count errors by type
   tail -10000 logs/migrator.log | jq -r '.msg' | sort | uniq -c | sort -rn
   ```

---

## Visibility Handling Configuration

### Overview

GitHub repository visibility controls **who can read the repository**:

- **Public**: Anyone with the URL can read/clone the repository
- **Internal**: Anyone in the GitHub Enterprise can read/clone the repository
- **Private**: Only users with explicit access can read/write to the repository

The GitHub Migrator provides configurable visibility transformation rules to handle migrations to different destination environments, particularly **GitHub Enterprise Cloud with EMU** and **GitHub with data residency**, which **do not support public repositories**.

### Configuration

Configure visibility handling in your `configs/config.yaml`:

```yaml
migration:
  visibility_handling:
    # How to handle public repositories: public, internal, or private
    # Default: private (safest, works in all environments)
    public_repos: "private"
    
    # How to handle internal repositories: internal or private
    # Default: private (safest for GitHub.com migrations)
    internal_repos: "private"
```

**Note**: Private repositories always migrate as private and cannot be changed.

### Visibility Mappings

#### Public Repositories

When migrating **public** repositories, you can configure one of three behaviors:

| Setting | Result | Use Case | Considerations |
|---------|--------|----------|----------------|
| `public` | Keep as public | Standard GitHub.com migrations | **Fails** in EMU and data residency environments |
| `internal` | Convert to internal | Maintain enterprise-wide read access | Requires Enterprise Cloud, preserves broad access |
| `private` | Convert to private | Maximum security, works everywhere | Most restrictive, requires explicit access grants |

#### Internal Repositories

When migrating **internal** repositories, you can configure one of two behaviors:

| Setting | Result | Use Case | Considerations |
|---------|--------|----------|----------------|
| `internal` | Keep as internal | Enterprise Cloud migrations | Requires destination to support internal visibility |
| `private` | Convert to private | GitHub.com, EMU, data residency | Works everywhere, most restrictive |

### Environment-Specific Recommendations

#### GitHub Enterprise Cloud (EMU)

EMU environments **do not support public repositories**. Recommended configuration:

```yaml
visibility_handling:
  public_repos: "internal"  # Maintains enterprise-wide read access
  internal_repos: "internal" # Maintain Enterprise visibility
```

#### GitHub with Data Residency

Data residency environments **do not support public repositories**. Recommended configuration:

```yaml
visibility_handling:
  public_repos: "internal"  # Maintains broad read access
  internal_repos: "internal" # If destination supports it
```

#### GitHub.com (Non Enterprise)

For standard GitHub.com migrations where public repos are supported:

```yaml
visibility_handling:
  public_repos: "public"   # Preserve public access
  internal_repos: "private" # GitHub.com (non Enterprise) doesn't have internal visibility
```

#### GitHub Enterprise Server → GitHub Enterprise

When migrating from GHES to GHEC:

```yaml
visibility_handling:
  public_repos: "public"   # Keep public if desired
  internal_repos: "internal" # Keep internal to maintain Enterprise visibility
```

### Environment Variables

For production deployments, use environment variables:

```bash
export GHMIG_MIGRATION_VISIBILITY_PUBLIC_REPOS=private
export GHMIG_MIGRATION_VISIBILITY_INTERNAL_REPOS=private
```

See `configs/env.example` for complete documentation.

### Complexity Scoring

The system uses **GitHub-specific complexity scoring** to estimate migration effort and identify potential challenges. The score is calculated based on remediation difficulty of features that don't migrate automatically.

#### Scoring Formula

The complexity score combines repository size, non-migrated features, and activity level:

**High Impact Features (3-4 points each):**
- Large files (>100MB): **4 points** - Must be remediated before migration
- Environments: **3 points** - Manual recreation of configs and protection rules required
- Secrets: **3 points** - Manual recreation required, high security sensitivity
- Packages: **3 points** - Don't migrate with GEI, manual migration required
- Self-hosted runners: **3 points** - Infrastructure reconfiguration needed

**Note:** Projects (classic) card-based boards DO migrate with GEI and are not scored. The new Projects experience (table-based at org level) doesn't migrate but isn't repository-level data.

**Moderate Impact Features (2 points each):**
- Variables: **2 points** - Manual recreation required
- Discussions: **2 points** - Don't migrate, community impact
- Releases: **2 points** - Only migrate on GHES 3.5.0+
- Git LFS: **2 points** - Special handling required
- Submodules: **2 points** - Dependency management complexity
- GitHub Apps: **2 points** - Reconfiguration/reinstallation needed

**Low Impact Features (1 point each):**
- GHAS (Code scanning/Dependabot/Secret scanning): **1 point** - Simple toggles to re-enable
- Webhooks: **1 point** - Must re-enable after migration
- Branch protections: **1 point** - Some rules don't migrate
- Rulesets: **1 point** - Manual recreation required (replaces deprecated tag protections)
- Public visibility: **1 point** - May need transformation
- Internal visibility: **1 point** - May need transformation
- CODEOWNERS: **1 point** - File detected in `.github/CODEOWNERS`, `docs/CODEOWNERS`, or `CODEOWNERS` on default branch. Verify file still exists and is valid.

**Repository Size (0-9 points):**
- <100MB: **0 points**
- 100MB-1GB: **3 points**
- 1GB-5GB: **6 points**
- >5GB: **9 points**

**Activity Level (0-4 points):**
Activity is calculated using **quantiles** relative to your repository dataset. High-activity repositories require significantly more planning, coordination, and stakeholder communication:
- High activity (top 25%): **4 points** - Many users, extensive coordination, high impact
- Moderate activity (25-75%): **2 points** - Some coordination needed
- Low activity (bottom 25%): **0 points** - Few users, minimal coordination

Activity combines: branch count, commit count, issue count, and pull request count.

#### Complexity Categories

Based on total score:
- **Simple** (≤5 points): Standard migration, minimal remediation
- **Medium** (6-10 points): Moderate effort, some planning needed
- **Complex** (11-17 points): Significant effort, careful planning required
- **Very Complex** (≥18 points): High effort, likely needs extensive remediation

#### Using Complexity Scores

Complexity scores help with:
- **Migration Planning**: Estimate effort and timeline
- **Resource Allocation**: Assign appropriate resources to complex migrations
- **Batch Organization**: Group repositories by complexity
- **Risk Assessment**: Identify high-risk migrations requiring extra attention
- **Remediation Planning**: Focus on repositories with high-weight features

The complexity score is visible in:
- Repository detail pages
- Batch management interface
- Analytics dashboards
- API responses (`GET /api/v1/analytics/complexity-distribution`)

#### Feature Detection Notes

The discovery process automatically detects repository features using the GitHub API. Here are important considerations:

**CODEOWNERS Detection:**
- Checks for files at: `.github/CODEOWNERS`, `docs/CODEOWNERS`, or `CODEOWNERS` (root)
- Only checks the repository's default branch
- Verifies the path is a file (not a directory)
- **Note**: Detection happens at discovery time. If files are added/removed after discovery, you should re-run discovery to update the detection

**Common False Positives:**
- Files that existed during discovery but were later deleted
- Empty or placeholder CODEOWNERS files
- Files on non-default branches (these are not detected)

**Troubleshooting Feature Detection:**
1. **Re-run Discovery**: Use the "Re-run Discovery" button in the UI to refresh feature detection
2. **Check Logs**: Enable debug logging (`LOG_LEVEL=debug`) to see detailed detection information including:
   - Which paths were checked
   - Whether files were found or returned 404
   - File types (file vs directory)
   - File sizes
3. **Manual Verification**: For critical features, manually verify in the source repository before migration

**Example Debug Log Output:**
```
level=debug msg="Checking for CODEOWNERS file" repo=org/repo
level=debug msg="CODEOWNERS check failed at location" repo=org/repo path=".github/CODEOWNERS" error="404 Not Found" is_404=true
level=debug msg="No CODEOWNERS file found" repo=org/repo
```

### Migration Logs

During migration, visibility transformations are logged:

```json
{
  "level": "info",
  "msg": "Applying visibility transformation",
  "repo": "acme-corp/api-service",
  "source_visibility": "public",
  "target_visibility": "internal"
}
```

This provides a clear audit trail of all visibility changes.

### Best Practices

1. **Test with Pilot Batch**: Always test visibility transformations with a pilot batch first
2. **Document Changes**: Maintain documentation of visibility changes for stakeholder communication
3. **Access Review**: Plan for post-migration access review, especially for public → private conversions
4. **Team Communication**: Inform teams about visibility changes before migration
5. **Validate Settings**: Ensure destination environment supports your configured visibility settings

### Troubleshooting

**Problem**: Migration fails with "public repositories not supported"

**Solution**: Your destination is EMU or data residency. Update configuration:
```yaml
visibility_handling:
  public_repos: "internal"  # or "private"
```

**Problem**: Internal repositories become private unexpectedly

**Solution**: Check your configuration. Default is `internal_repos: "private"`. To preserve internal visibility:
```yaml
visibility_handling:
  internal_repos: "internal"
```

---

## GitHub Enterprise Importer Limitations

The GitHub Migrator automatically detects and manages GitHub Enterprise Importer API limitations to prevent migration failures.

### Repository Size Limit (40 GiB)

GitHub enforces a **40 GiB total repository size limit** for migrations. Repositories exceeding this limit are automatically blocked and require remediation.

**Detection:**
- Automatically detected during repository discovery using git-sizer
- Repositories over 40 GiB are marked with `has_oversized_repository: true`
- Status automatically set to `remediation_required`

**Remediation Strategies:**

1. **Convert Large Files to Git LFS:**
   ```bash
   # Install git-lfs
   git lfs install
   
   # Track large file types
   git lfs track "*.psd"
   git lfs track "*.zip"
   
   # Migrate existing files to LFS
   git lfs migrate import --include="*.psd,*.zip" --everything
   
   # Push changes
   git push origin --all --force
   ```

2. **Remove Large Assets from History:**
   ```bash
   # Using BFG Repo-Cleaner
   java -jar bfg.jar --strip-blobs-bigger-than 100M repo.git
   
   # Or using git-filter-repo
   git filter-repo --strip-blobs-bigger-than 100M
   
   # Force push cleaned history
   git push origin --all --force
   ```

3. **Split Repository:**
   - Consider splitting monolithic repositories into smaller components
   - Move large assets to separate artifact storage

**After Remediation:**
1. In the web UI, click "Mark as Remediated" button
2. System will re-run discovery to verify size is under limit
3. Status will change from `remediation_required` to `pending` if successful

### Metadata Size Limit (40 GiB)

GitHub also enforces a **40 GiB metadata limit** (issues, PRs, releases, attachments).

**Detection:**
- Estimated during discovery based on:
  - Issue count × 5 KB average
  - PR count × 10 KB average
  - Actual release asset sizes from GitHub API
  - Estimated attachment sizes (10% of issue/PR data)

**Handling Large Metadata:**

If estimated metadata exceeds 35 GiB, the system shows warnings and recommends using exclusion flags.

**Exclusion Flags (Per-Repository Settings):**

Configure in the web UI under "Migration Options" or via API:

```bash
curl -X PATCH http://localhost:8080/api/v1/repositories/acme-corp%2Flarge-repo \
  -H "Content-Type: application/json" \
  -d '{
    "exclude_releases": true,
    "exclude_attachments": true
  }'
```

Available exclusion flags:

| Flag | Description | Use Case |
|------|-------------|----------|
| `exclude_releases` | Skip releases and assets | Large release assets (ISOs, binaries) |
| `exclude_attachments` | Skip issue/PR attachments | Many images/files attached to issues |
| `exclude_metadata` | Skip ALL metadata (issues, PRs, releases, wikis) | Code-only migration |
| `exclude_git_data` | Skip git data (rarely used) | Metadata-only migration (not recommended) |
| `exclude_owner_projects` | Skip org/user project boards | Org-level projects not needed |

**Example Workflow for Large Metadata:**

```bash
# 1. Check metadata size estimate
curl http://localhost:8080/api/v1/repositories/acme-corp%2Fmy-repo | \
  jq '{
    estimated_metadata_gb: (.estimated_metadata_size / 1073741824),
    metadata_details: .metadata_size_details | fromjson
  }'

# 2. If approaching limit, enable exclude_releases
curl -X PATCH http://localhost:8080/api/v1/repositories/acme-corp%2Fmy-repo \
  -H "Content-Type: application/json" \
  -d '{"exclude_releases": true}'

# 3. Migrate repository
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{
    "full_names": ["acme-corp/my-repo"],
    "dry_run": false
  }'

# 4. Manually migrate releases after repository migration completes
# Use gh CLI or GitHub API to recreate releases on destination
```

### Other Migration Limits

The system also validates:

- **2 GiB Commit Limit**: Single commits >2 GiB are blocked
- **255 Byte Reference Name Limit**: Git refs (branches/tags) >255 bytes blocked
- **400 MiB File Limit**: Individual files >400 MiB blocked during migration
- **100 MiB Post-Migration Limit**: Files 100-400 MiB allowed during migration but flagged for post-migration remediation

### Viewing Limitation Details

**Web UI:**
1. Navigate to repository detail page
2. View "Migration Limits" section
3. Shows blocking issues (red) and warnings (yellow)
4. Configure exclusion flags in "Migration Options" section

**API:**
```bash
# Get repository with limit details
curl http://localhost:8080/api/v1/repositories/acme-corp%2Fmy-repo | jq '{
  repository: .full_name,
  status: .status,
  size_gb: (.total_size / 1073741824),
  blocking_issues: {
    oversized_repository: .has_oversized_repository,
    oversized_commits: .has_oversized_commits,
    long_refs: .has_long_refs,
    blocking_files: .has_blocking_files
  },
  warnings: {
    large_files: .has_large_file_warnings,
    large_metadata: (.estimated_metadata_size > 37580963840)
  },
  exclusion_flags: {
    exclude_releases: .exclude_releases,
    exclude_attachments: .exclude_attachments,
    exclude_metadata: .exclude_metadata
  }
}'
```

---

## Migration Workflows

### Starting a New Migration Wave

#### 1. Pilot Migration (First 5-10 Repositories)

```bash
# Step 1: Discover repositories
curl -X POST http://localhost:8080/api/v1/discovery/start \
  -H "Content-Type: application/json" \
  -d '{"organization": "acme-corp"}'

# Step 2: Wait for discovery to complete
curl http://localhost:8080/api/v1/discovery/status

# Step 3: Create pilot batch
curl -X POST http://localhost:8080/api/v1/batches \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pilot - Wave 1",
    "description": "Initial pilot repositories for testing",
    "repository_ids": [1, 2, 3, 4, 5]
  }'

# Step 4: Start DRY RUN first
curl -X POST http://localhost:8080/api/v1/batches/1/start?dry_run=true

# Step 5: Monitor dry run results
curl http://localhost:8080/api/v1/batches/1

# Step 6: If successful, start actual migration
curl -X POST http://localhost:8080/api/v1/batches/1/start

# Step 7: Monitor progress
watch -n 30 'curl -s http://localhost:8080/api/v1/batches/1 | jq'
```

#### 2. Wave Migration (Larger Batches)

```bash
# Step 1: Identify repositories for wave
curl "http://localhost:8080/api/v1/repositories?status=ready&batch_id=null" | jq

# Step 2: Create wave batch
curl -X POST http://localhost:8080/api/v1/batches \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wave 2 - Backend Services",
    "description": "Core backend microservices",
    "repository_ids": [10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
  }'

# Step 3: Assign priorities (higher = migrates first)
for id in 10 11 12 13 14 15 16 17 18 19 20; do
  curl -X PATCH http://localhost:8080/api/v1/repositories/acme-corp/repo-${id} \
    -H "Content-Type: application/json" \
    -d "{\"priority\": $((20 - id))}"
done

# Step 4: Start batch migration
curl -X POST http://localhost:8080/api/v1/batches/2/start

# Step 5: Monitor with dashboard
# Open http://localhost:8080 in browser
```

#### 3. Self-Service Migration

For developers to migrate their own repositories:

```bash
# Provide developers with this command
curl -X POST http://localhost:8080/api/v1/migrations/start \
  -H "Content-Type: application/json" \
  -d '{
    "full_names": ["acme-corp/my-repo"],
    "dry_run": false,
    "priority": 5
  }'

# Or share self-service web UI
# http://localhost:8080/#/self-service
```

### GitHub Migration Limits Detection

The migrator automatically detects repositories that violate GitHub's migration limits and flags them for remediation before migration can proceed.

#### Detected Limitations

**Blocking Issues** (prevent migration):
- **2 GiB Commit Limit**: No single commit can exceed 2 GiB
- **255 Byte Reference Limit**: Git references (branches, tags) cannot exceed 255 bytes
- **400 MiB File Limit**: Files cannot exceed 400 MiB during migration

**Warnings** (non-blocking but require post-migration attention):
- **100 MiB File Warning**: Files 100-400 MiB are allowed during migration but exceed GitHub's 100 MiB post-migration limit

#### Status: Remediation Required

When a repository has blocking issues, it will be automatically marked with status `remediation_required`:

```bash
# List repositories requiring remediation
curl "http://localhost:8080/api/v1/repositories?status=remediation_required"
```

**Via Web UI:**
1. Navigate to repository detail page
2. View "Migration Limits" section showing all issues
3. See specific commits, refs, or files that need fixing

#### Remediation Workflow

**1. Fix Blocking Issues in Source Repository**

For oversized commits (>2 GiB):
```bash
# Split large commits using git filter-repo
git filter-repo --commit-callback '
  # Custom logic to split commits
'
```

For long git references (>255 bytes):
```bash
# Rename long branches
git branch -m "very-long-branch-name..." "shorter-name"

# Rename long tags
git tag new-name old-very-long-tag-name
git tag -d old-very-long-tag-name
```

For large files (>400 MiB):
```bash
# Option 1: Use Git LFS
git lfs track "*.zip"
git lfs migrate import --include="*.zip"

# Option 2: Remove from history
git filter-repo --path-glob '*.large-file' --invert-paths
```

**2. Mark Repository as Remediated**

After fixing issues, trigger re-validation:

```bash
# Via API
curl -X POST http://localhost:8080/api/v1/repositories/org/repo/mark-remediated

# Via Web UI
# 1. Go to repository detail page
# 2. Click "Mark as Remediated" button in Migration Limits section
# 3. System will re-analyze the repository
```

**3. Verify Resolution**

The system will:
- Re-clone the repository
- Re-run all git validation checks
- Update status to `pending` if all issues resolved
- Keep status as `remediation_required` if issues remain

#### Large File Warnings (100-400 MiB)

Files in this range don't block migration but should be addressed:

**Options:**
1. **Git LFS**: Convert files to LFS before migration
2. **Post-Migration**: Fix after migration completes
3. **External Storage**: Move to external storage system

**Recommendation**: Fix before migration when possible to avoid post-migration work.

#### Documentation References

- [GitHub Migration Limitations](https://docs.github.com/en/migrations/using-github-enterprise-importer/migrating-between-github-products/about-migrations-between-github-products#limitations-of-github)
- [Managing Large Files with Git LFS](https://docs.github.com/en/repositories/working-with-files/managing-large-files)
- [Removing Files from Git History](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- [git-filter-repo Tool](https://github.com/newren/git-filter-repo)

### Migration Best Practices

1. **Always Dry Run First**
   - Test with `dry_run: true` before actual migration
   - Validates repository can be migrated
   - Identifies potential issues

2. **Start Small**
   - Begin with 5-10 pilot repositories
   - Choose diverse repositories (different sizes, features)
   - Learn from pilot before scaling up

3. **Prioritize Critical Repos**
   - Set higher priority for important repositories
   - Low complexity repos should go first
   - Save problematic repos for last

4. **Monitor Closely**
   - Watch first few migrations closely
   - Check logs for errors
   - Validate migrated repositories in destination

5. **Batch Appropriately**
   - Group similar repositories together
   - Don't exceed 50 repos per batch
   - Consider off-peak hours for large batches

---

## Monitoring & Alerts

### Key Metrics to Monitor

#### 1. Migration Success Rate

```bash
# Check overall success rate
curl http://localhost:8080/api/v1/analytics/summary | \
  jq '{
    total: .total_repositories,
    completed: .by_status.completed,
    failed: .by_status.failed,
    success_rate: (.by_status.completed / .total_repositories * 100)
  }'
```

**Alert if:** Success rate < 95%

#### 2. Migration Duration

```bash
# Check average duration
curl http://localhost:8080/api/v1/analytics/progress | \
  jq '.average_duration_minutes'
```

**Alert if:** Average duration > 30 minutes

#### 3. Stuck Migrations

```bash
# Find migrations running > 2 hours
curl http://localhost:8080/api/v1/repositories?status=migrating | \
  jq '[.[] | select(.updated_at | fromdateiso8601 < (now - 7200))]'
```

**Alert if:** Any migration > 2 hours

#### 4. Error Rate

```bash
# Count errors in last hour
tail -1000 logs/migrator.log | \
  jq -r 'select(.level=="error") | .time' | \
  awk -v now=$(date +%s) '{
    if (systime($0) > now - 3600) count++
  } END {print count}'
```

**Alert if:** > 10 errors in 1 hour

#### 5. Disk Usage

```bash
# Check data directory
du -sh data/
# Check logs directory
du -sh logs/
```

**Alert if:** > 80% of allocated space

### Setting Up Alerts

#### Example: Prometheus AlertManager

```yaml
groups:
- name: github_migrator
  interval: 1m
  rules:
  
  # Health check failed
  - alert: MigratorDown
    expr: up{job="github-migrator"} == 0
    for: 2m
    annotations:
      summary: "GitHub Migrator is down"
      description: "Health check failing for 2+ minutes"
    
  # High error rate
  - alert: HighErrorRate
    expr: rate(migrator_errors_total[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High error rate detected"
      description: "Error rate > 10% for 5 minutes"
    
  # Stuck migrations
  - alert: StuckMigrations
    expr: migrator_migrations_stuck > 0
    for: 30m
    annotations:
      summary: "Migrations stuck"
      description: "{{ $value }} migrations stuck > 30 minutes"
```

#### Example: Simple Cron Monitoring

```bash
#!/bin/bash
# /usr/local/bin/monitor-migrator.sh

# Check health
if ! curl -sf http://localhost:8080/health > /dev/null; then
  echo "ALERT: Migrator health check failed" | mail -s "Migrator Alert" ops@company.com
  exit 1
fi

# Check for stuck migrations
STUCK=$(curl -s http://localhost:8080/api/v1/repositories?status=migrating | \
  jq '[.[] | select(.updated_at | fromdateiso8601 < (now - 7200))] | length')

if [ "$STUCK" -gt 0 ]; then
  echo "ALERT: $STUCK migrations stuck > 2 hours" | mail -s "Migrator Alert" ops@company.com
fi

# Check error rate
ERRORS=$(tail -1000 logs/migrator.log | grep -c '"level":"error"')
if [ "$ERRORS" -gt 10 ]; then
  echo "ALERT: High error rate ($ERRORS errors in recent logs)" | mail -s "Migrator Alert" ops@company.com
fi
```

Add to cron:
```bash
*/5 * * * * /usr/local/bin/monitor-migrator.sh
```

---

## Incident Response

### Severity Levels

- **P1 (Critical):** Service down, all migrations blocked
- **P2 (High):** Batch migration failing, high error rate
- **P3 (Medium):** Individual migration failures, performance issues
- **P4 (Low):** Minor issues, cosmetic bugs

### P1: Service Down

**Symptoms:**
- Health check failing
- API not responding
- Container/process crashed

**Immediate Actions:**

1. **Check service status**
   ```bash
   # Docker
   docker ps | grep migrator
   docker logs github-migrator --tail 100
   
   # Systemd
   sudo systemctl status github-migrator
   sudo journalctl -u github-migrator -n 100
   ```

2. **Restart service**
   ```bash
   # Docker
   docker-compose restart
   
   # Systemd
   sudo systemctl restart github-migrator
   ```

3. **Check health after restart**
   ```bash
   sleep 10
   curl http://localhost:8080/health
   ```

4. **If still failing, check:**
   - Database connectivity
   - Disk space
   - Memory usage
   - Port conflicts

5. **Escalate if not resolved in 15 minutes**

### P2: Batch Migration Failing

**Symptoms:**
- Multiple migrations failing in a batch
- High error rate
- Consistent failures on similar repositories

**Actions:**

1. **Identify failing batch**
   ```bash
   curl http://localhost:8080/api/v1/batches | \
     jq '.[] | select(.status=="failed")'
   ```

2. **Get batch details**
   ```bash
   curl http://localhost:8080/api/v1/batches/{id}
   ```

3. **Check migration logs**
   ```bash
   curl http://localhost:8080/api/v1/migrations/{id}/logs | \
     jq 'select(.level=="error")'
   ```

4. **Common causes:**
   - GitHub API rate limit (wait for reset)
   - Invalid token permissions
   - Network connectivity issues
   - Repository-specific issues (LFS, size, etc.)

5. **Resolution:**
   - Fix root cause
   - Retry failed migrations
   - Consider splitting batch into smaller groups

### P3: Individual Migration Failure

**Symptoms:**
- Single repository failing to migrate
- Migration stuck in specific phase

**Actions:**

1. **Get migration details**
   ```bash
   curl http://localhost:8080/api/v1/migrations/{id}
   ```

2. **Review migration history**
   ```bash
   curl http://localhost:8080/api/v1/migrations/{id}/history
   ```

3. **Check migration logs**
   ```bash
   curl http://localhost:8080/api/v1/migrations/{id}/logs
   ```

4. **Common issues:**
   - Repository too large (split using Git LFS or filter-branch)
   - LFS bandwidth limit (upgrade or wait)
   - Protected branches (update settings)
   - Permissions issues (check token scopes)

5. **Resolution:**
   - Address specific issue
   - Retry migration
   - If persistent, mark as manual migration

---

## Maintenance Tasks

### Weekly Maintenance

#### 1. Database Optimization

**SQLite:**
```bash
# Vacuum and optimize
sqlite3 data/migrator.db "VACUUM;"
sqlite3 data/migrator.db "ANALYZE;"
sqlite3 data/migrator.db "PRAGMA integrity_check;"
```

**PostgreSQL:**
```bash
psql -U migrator_user -d migrator -c "VACUUM ANALYZE;"
psql -U migrator_user -d migrator -c "REINDEX DATABASE migrator;"
```

#### 2. Log Rotation and Cleanup

```bash
# Clean old logs (older than 30 days)
find logs/ -name "*.log.*" -mtime +30 -delete

# Compress recent rotated logs
find logs/ -name "*.log.*" -mtime -7 ! -name "*.gz" -exec gzip {} \;
```

#### 3. Backup Verification

```bash
# Test latest backup
LATEST_BACKUP=$(ls -t data/backup-*.db | head -1)
sqlite3 $LATEST_BACKUP "SELECT COUNT(*) FROM repositories;"
```

#### 4. Dependency Updates

```bash
# Update Go dependencies
go get -u ./...
go mod tidy

# Update frontend dependencies
cd web && npm update && cd ..

# Rebuild and test
make build-all
make test
```

### Monthly Maintenance

#### 1. Security Audit

```bash
# Run security scan
make lint  # includes gosec

# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Update base images
docker pull golang:1.21-alpine
docker pull alpine:latest
docker build --no-cache -t github-migrator:latest .
```

#### 2. Token Rotation

```bash
# Generate new tokens in GitHub
# Update secrets/environment variables
# Test with new tokens
# Deploy new configuration
# Revoke old tokens
```

#### 3. Performance Review

```bash
# Check database size
du -sh data/migrator.db

# Review slow queries (if PostgreSQL)
# Check migration duration trends
curl http://localhost:8080/api/v1/analytics/progress?days=30
```

#### 4. Capacity Planning

```bash
# Calculate storage needs
TOTAL_REPOS=$(curl -s http://localhost:8080/api/v1/analytics/summary | jq '.total_repositories')
AVG_SIZE=$(curl -s http://localhost:8080/api/v1/analytics/summary | jq '.average_size_mb')
PROJECTED_REPOS=1000  # Your target

STORAGE_NEEDED=$(echo "$PROJECTED_REPOS * $AVG_SIZE / 1024" | bc)
echo "Estimated storage needed: ${STORAGE_NEEDED} GB"
```

---

## Troubleshooting Guide

### Issue: Migration Stuck in "Migrating" Status

**Diagnosis:**
```bash
# Get migration details
curl http://localhost:8080/api/v1/migrations/{id}

# Check logs
curl http://localhost:8080/api/v1/migrations/{id}/logs

# Check GitHub migration status
# (Use GitHub API to check actual migration)
```

**Possible Causes:**
1. GitHub migration actually in progress (wait)
2. Migration failed but status not updated
3. Network timeout
4. Server restart during migration

**Resolution:**
```bash
# Option 1: Wait longer (migrations can take hours)
# Option 2: Check GitHub directly for migration status
# Option 3: Update status manually in database (last resort)

# If migration completed on GitHub but status stuck:
sqlite3 data/migrator.db "UPDATE repositories SET status='completed' WHERE id={repo_id};"
```

### Issue: High Memory Usage

**Diagnosis:**
```bash
# Docker
docker stats github-migrator

# System
top -p $(pgrep -f github-migrator)
```

**Possible Causes:**
1. Too many parallel workers
2. Large repositories being profiled
3. Memory leak (check logs for patterns)

**Resolution:**
```bash
# Reduce parallel workers
# Edit config.yaml
migration:
  parallel_workers: 3
discovery:
  parallel_workers: 5

# Restart service
docker-compose restart
```

### Issue: Database Locked (SQLite)

**Diagnosis:**
```bash
# Check for multiple processes
ps aux | grep github-migrator

# Check file locks
lsof data/migrator.db
```

**Resolution:**
```bash
# Stop all instances
docker-compose down

# Ensure clean state
rm -f data/migrator.db-shm data/migrator.db-wal

# Start single instance
docker-compose up -d

# For production, migrate to PostgreSQL
```

### Issue: API Rate Limit Exceeded

**Diagnosis:**
```bash
# Check rate limit status
curl -H "Authorization: token $GITHUB_SOURCE_TOKEN" \
  https://github.company.com/api/v3/rate_limit
```

**Response:**
```json
{
  "resources": {
    "core": {
      "limit": 5000,
      "remaining": 0,
      "reset": 1705324800
    }
  }
}
```

**Resolution:**
```bash
# Wait for reset (automatic in server)
RESET_TIME=$(curl -s -H "Authorization: token $GITHUB_SOURCE_TOKEN" \
  https://github.company.com/api/v3/rate_limit | jq '.resources.core.reset')

echo "Rate limit resets at: $(date -d @$RESET_TIME)"

# Or use additional tokens (configure multiple sources)
```

---

## Runbooks

### Runbook: Complete Server Recovery

**Scenario:** Server crashed, need to restore from backup

```bash
# 1. Stop any running instances
docker-compose down
# or
sudo systemctl stop github-migrator

# 2. Restore database from backup
LATEST_BACKUP=$(ls -t backups/migrator-*.db | head -1)
cp $LATEST_BACKUP data/migrator.db

# 3. Verify database integrity
sqlite3 data/migrator.db "PRAGMA integrity_check;"

# 4. Start service
docker-compose up -d
# or
sudo systemctl start github-migrator

# 5. Verify health
sleep 10
curl http://localhost:8080/health

# 6. Check repository count
curl http://localhost:8080/api/v1/analytics/summary | jq '.total_repositories'

# 7. Resume any incomplete migrations
curl http://localhost:8080/api/v1/repositories?status=migrating
```

### Runbook: Emergency Stop All Migrations

**Scenario:** Need to immediately stop all migrations

```bash
# 1. Stop the service
docker-compose down
# or
sudo systemctl stop github-migrator

# 2. Update all in-progress migrations to 'failed' status
sqlite3 data/migrator.db <<EOF
UPDATE repositories 
SET status = 'failed' 
WHERE status IN ('migrating', 'queued');

UPDATE migration_history 
SET status = 'failed', 
    error_message = 'Manually stopped by operator',
    completed_at = datetime('now')
WHERE status IN ('migrating', 'queued', 'in_progress');
EOF

# 3. Start service
docker-compose up -d

# 4. Verify
curl http://localhost:8080/api/v1/repositories?status=migrating
# Should return empty array
```

### Runbook: Migrate from SQLite to PostgreSQL

**Scenario:** Need to scale up from SQLite to PostgreSQL

```bash
# 1. Stop service
docker-compose down

# 2. Backup SQLite database
cp data/migrator.db data/migrator-backup-$(date +%Y%m%d).db

# 3. Export data
sqlite3 data/migrator.db .dump > migrator-export.sql

# 4. Set up PostgreSQL
docker run -d \
  --name migrator-postgres \
  -e POSTGRES_DB=migrator \
  -e POSTGRES_USER=migrator_user \
  -e POSTGRES_PASSWORD=secure_password \
  -p 5432:5432 \
  postgres:15

# 5. Create schema (run migrations will do this automatically)

# 6. Update configuration
cat > configs/config.yaml <<EOF
database:
  type: postgresql
  dsn: "host=localhost port=5432 user=migrator_user password=secure_password dbname=migrator sslmode=disable"
EOF

# 7. Start service (migrations will run automatically)
docker-compose up -d

# 8. Verify
curl http://localhost:8080/api/v1/analytics/summary
```

### Runbook: Token Rotation

**Scenario:** Rotate GitHub tokens (security best practice)

```bash
# 1. Generate new tokens in GitHub UI
# Source: https://github.company.com/settings/tokens
# Destination: https://github.com/settings/tokens

# 2. Test new tokens
curl -H "Authorization: token NEW_SOURCE_TOKEN" \
  https://github.company.com/api/v3/user

curl -H "Authorization: token NEW_DEST_TOKEN" \
  https://api.github.com/user

# 3. Update environment variables
export GITHUB_SOURCE_TOKEN="new_source_token"
export GITHUB_DEST_TOKEN="new_dest_token"

# 4. Update config file or secrets
# Edit configs/config.yaml or update Docker secrets

# 5. Restart service
docker-compose down
docker-compose up -d

# 6. Verify service works with new tokens
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/discovery/start \
  -H "Content-Type: application/json" \
  -d '{"organization": "test-org"}'

# 7. Revoke old tokens in GitHub UI
```

---

## Performance Tuning

### Optimal Configuration for Different Scales

#### Small Scale (< 1,000 repositories)

```yaml
database:
  type: sqlite
  dsn: ./data/migrator.db

migration:
  parallel_workers: 3
  
discovery:
  parallel_workers: 5
```

#### Medium Scale (1,000 - 10,000 repositories)

```yaml
database:
  type: postgresql
  dsn: "postgres://..."
  max_open_conns: 25
  
migration:
  parallel_workers: 5
  
discovery:
  parallel_workers: 10
```

#### Large Scale (10,000+ repositories)

```yaml
database:
  type: postgresql
  dsn: "postgres://..."
  max_open_conns: 50
  
migration:
  parallel_workers: 10
  
discovery:
  parallel_workers: 20
```

---

## Contacts & Escalation

### Escalation Path

1. **L1 - Operations Team:** Initial triage and basic troubleshooting
2. **L2 - Platform Team:** Advanced troubleshooting, configuration changes
3. **L3 - Development Team:** Code issues, bugs, architecture changes

### Key Contacts

- **Operations Lead:** ops-lead@company.com
- **Platform Team:** platform@company.com
- **Development Team:** dev-team@company.com
- **On-Call:** oncall@company.com

### Internal Documentation

- **[README.md](../README.md)** - Project overview and quickstart
- **[CONTRIBUTING.md](./CONTRIBUTING.md)** - Development setup and contributing guidelines
- **[IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)** - Technical architecture and implementation details
- **[API.md](./API.md)** - Complete API reference
- **[DEPLOYMENT.md](./DEPLOYMENT.md)** - Production deployment instructions

### External Resources

- **GitHub Support:** https://support.github.com
- **GitHub Migration Docs:** https://docs.github.com/en/migrations
- **Project Repository:** https://github.com/your-org/github-migrator

---

**Operations Runbook Version:** 1.0.0  
**Last Updated:** October 2025  
**Next Review:** Quarterly

