# Configuration Files

Configuration templates for the GitHub Migrator application.

## Multi-Source Support

GitHub Migrator supports **multiple migration sources** that all migrate to a shared destination.
You can configure sources in two ways:

1. **Database-based sources** (Recommended) - Configure via the UI at `/sources`
2. **Environment variables** (Legacy) - Single source via `.env` file

### Database-based Sources (Multi-Source)

After initial setup, navigate to the **Sources** page in the UI to add multiple GitHub or Azure DevOps sources. Each source stores its own connection credentials and can be managed independently.

Benefits:
- Configure multiple sources (e.g., GHES + ADO)
- Manage sources via the web UI
- Track repositories by source
- No server restart required to add sources

### Legacy Environment-Based Configuration

For single-source setups, you can still use environment variables:

## Quick Start

Choose templates based on your **source** system:

| Source System | Environment File | YAML Config |
|--------------|------------------|-------------|
| **GitHub** | [`env.github.example`](./env.github.example) | [`config.github.yml`](./config.github.yml) |
| **Azure DevOps** | [`env.azuredevops.example`](./env.azuredevops.example) | [`config.azuredevops.yml`](./config.azuredevops.yml) |

> **Note**: The destination is always GitHub (GEI requirement).

---

## Option 1: Environment Variables (Recommended)

```bash
# For GitHub to GitHub migrations
cp configs/env.github.example .env

# For Azure DevOps to GitHub migrations
cp configs/env.azuredevops.example .env

# Edit and fill in your values
vim .env

# Run the application
make docker-run
```

The `.env` file is automatically loaded and gitignored.

## Option 2: YAML Configuration

```bash
# For GitHub to GitHub migrations
cp configs/config.github.yml configs/config.yml

# For Azure DevOps to GitHub migrations
cp configs/config.azuredevops.yml configs/config.yml

# Edit and fill in your values
vim configs/config.yml

# Run the application
make docker-run
```

## Option 3: Hybrid Approach

Use both YAML for non-sensitive settings and `.env` for secrets:

```yaml
# configs/config.yml - checked into git
server:
  port: 8080
migration:
  workers: 5
```

```bash
# .env - gitignored, contains secrets
GHMIG_SOURCE_TOKEN=ghp_secret
GHMIG_DESTINATION_TOKEN=ghp_secret
```

Environment variables always override YAML settings.

---

## Configuration Examples

### GitHub Source

**Environment Variables:**
```bash
GHMIG_SOURCE_TYPE=github
GHMIG_SOURCE_BASE_URL=https://api.github.com
GHMIG_SOURCE_TOKEN=ghp_YourGitHubPAT
GHMIG_DESTINATION_TYPE=github
GHMIG_DESTINATION_BASE_URL=https://api.github.com
GHMIG_DESTINATION_TOKEN=ghp_YourDestPAT
```

**YAML:**
```yaml
source:
  type: github
  base_url: https://api.github.com
  token: ghp_YourGitHubPAT
destination:
  type: github
  base_url: https://api.github.com
  token: ghp_YourDestPAT
```

### Azure DevOps Source

**Environment Variables:**
```bash
GHMIG_SOURCE_TYPE=azuredevops
GHMIG_SOURCE_BASE_URL=https://dev.azure.com/YOUR_ORG
GHMIG_SOURCE_TOKEN=your-ado-pat
GHMIG_SOURCE_ORGANIZATION=YOUR_ORG
GHMIG_DESTINATION_TYPE=github
GHMIG_DESTINATION_BASE_URL=https://api.github.com
GHMIG_DESTINATION_TOKEN=ghp_YourGitHubPAT
```

**YAML:**
```yaml
source:
  type: azuredevops
  base_url: https://dev.azure.com/YOUR_ORG
  token: your-ado-pat
  organization: YOUR_ORG
destination:
  type: github
  base_url: https://api.github.com
  token: ghp_YourGitHubPAT
```

---

## Environment Variable Naming

All variables use the `GHMIG_` prefix. Dots in YAML paths become underscores:

| YAML Path | Environment Variable |
|-----------|---------------------|
| `server.port` | `GHMIG_SERVER_PORT` |
| `source.type` | `GHMIG_SOURCE_TYPE` |
| `source.base_url` | `GHMIG_SOURCE_BASE_URL` |
| `source.token` | `GHMIG_SOURCE_TOKEN` |
| `destination.token` | `GHMIG_DESTINATION_TOKEN` |
| `auth.enabled` | `GHMIG_AUTH_ENABLED` |

## Configuration Precedence

Settings are applied in order (later overrides earlier):

1. **Built-in defaults** (in code)
2. **YAML config file** (`configs/config.yml`)
3. **Environment variables** (`.env` or system)

---

## Authentication Configuration

GitHub Migrator uses **destination-centric authentication**. All users authenticate via GitHub OAuth against the destination GitHub instance. This simplifies multi-source setups by not requiring OAuth configuration for each source.

### Authorization Tiers

The system implements three authorization tiers:

| Tier | Name | Who | Capabilities |
|------|------|-----|-------------|
| 1 | Admin | Enterprise Admins, Migration Team Members | Full migration rights - any repository |
| 2 | Self-Service | Users with completed identity mapping | Can migrate repos they admin on source |
| 3 | Read-Only | All authenticated users | View status and history only |

### Example Configuration

```bash
# Enable authentication
GHMIG_AUTH_ENABLED=true
GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID=Iv1.YourClientID
GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET=YourClientSecret
GHMIG_AUTH_CALLBACK_URL=http://localhost:8080/api/v1/auth/callback
GHMIG_AUTH_FRONTEND_URL=http://localhost:3000
GHMIG_AUTH_SESSION_SECRET=your-secure-random-string

# Tier 1: Full migration rights for migration team
GHMIG_AUTH_AUTHORIZATION_RULES_MIGRATION_ADMIN_TEAMS=my-org/migration-admins

# Tier 2: Require identity mapping for self-service
GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_IDENTITY_MAPPING_FOR_SELF_SERVICE=true
```

For detailed authentication documentation, see [internal/auth/README.md](../internal/auth/README.md).

---

## Additional Resources

- [Deployment Guide](../docs/deployment/) - Docker, Azure, Kubernetes
- [Operations Guide](../docs/OPERATIONS.md) - Authentication, workflows, troubleshooting
- [API Documentation](../docs/API.md) - REST API reference
- [Authentication Guide](../internal/auth/README.md) - Destination-centric auth model

---

## Security Best Practices

1. **Never commit secrets** - `.env` is gitignored
2. **Rotate tokens regularly** - Every 90 days recommended
3. **Use minimal token scopes** - Only grant required permissions
4. **Protect your configuration** - `chmod 600 .env`
5. **Use secret managers in production** - Azure Key Vault, AWS Secrets Manager

## Troubleshooting

### Configuration not loading?

1. **Check file location**: `.env` must be in project root (not in `configs/`)
2. **Check naming**: Variables need `GHMIG_` prefix
3. **Check format**: YAML uses spaces (not tabs)

### Common mistakes

```bash
# ❌ WRONG - Missing GHMIG_ prefix
SOURCE_TYPE=github

# ✅ CORRECT
GHMIG_SOURCE_TYPE=github
```

```bash
# ❌ WRONG - Wrong location
configs/.env

# ✅ CORRECT
.env  # In project root
```

### Enable debug logging

```bash
GHMIG_LOGGING_LEVEL=debug
```

Then check logs at `./logs/migrator.log`.
