# Configuration Files

This directory contains configuration templates for the GitHub Migrator application.

## 📁 Available Configurations

### Source-Specific Templates

Choose the appropriate template based on your **source** repository system:

| Source System | Environment File | YAML Config File |
|--------------|------------------|------------------|
| **GitHub** | `env.github.example` | `config.github.yml` |
| **Azure DevOps** | `env.azuredevops.example` | `config.azuredevops.yml` |
| **General** | `env.example` | `config_template.yml` |

> **Note**: The destination must always be GitHub (GEI requirement).

## 🚀 Quick Start

### Option 1: Using Environment Variables (Recommended)

1. **Copy the appropriate example file:**
   ```bash
   # For GitHub to GitHub migrations
   cp configs/env.github.example .env
   
   # For Azure DevOps to GitHub migrations
   cp configs/env.azuredevops.example .env
   ```

2. **Edit `.env` and fill in your values:**
   ```bash
   vim .env
   # or
   nano .env
   ```

3. **Run the application:**
   ```bash
   ./bin/github-migrator-server
   ```
   
   The `.env` file is automatically loaded and your secrets stay local (it's gitignored).

### Option 2: Using YAML Configuration

1. **Copy the appropriate template:**
   ```bash
   # For GitHub to GitHub migrations
   cp configs/config.github.yml configs/config.yml
   
   # For Azure DevOps to GitHub migrations
   cp configs/config.azuredevops.yml configs/config.yml
   ```

2. **Edit `configs/config.yml` and fill in your values:**
   ```bash
   vim configs/config.yml
   ```

3. **Run the application:**
   ```bash
   ./bin/github-migrator-server
   ```

### Option 3: Hybrid Approach

You can use **both** a YAML config file and environment variables. Environment variables always override YAML settings.

This is useful for:
- Keeping non-sensitive settings in YAML (checked into git)
- Keeping secrets in `.env` (gitignored)

```bash
# config.yml - checked into git
server:
  port: 8080
migration:
  workers: 5

# .env - gitignored, contains secrets
GHMIG_SOURCE_TOKEN=ghp_secret
GHMIG_DESTINATION_TOKEN=ghp_secret
```

## 📚 Configuration Formats

### GitHub Source Configuration

#### Environment Variables (`.env`)
```bash
GHMIG_SOURCE_TYPE=github
GHMIG_SOURCE_BASE_URL=https://api.github.com
GHMIG_SOURCE_TOKEN=ghp_YourGitHubPAT
GHMIG_DESTINATION_TYPE=github
GHMIG_DESTINATION_BASE_URL=https://api.github.com
GHMIG_DESTINATION_TOKEN=ghp_YourDestPAT
```

#### YAML (`config.yml`)
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

### Azure DevOps Source Configuration

#### Environment Variables (`.env`)
```bash
GHMIG_SOURCE_TYPE=azuredevops
GHMIG_SOURCE_BASE_URL=https://dev.azure.com/YOUR_ORG
GHMIG_SOURCE_TOKEN=your-ado-pat
GHMIG_SOURCE_ORGANIZATION=YOUR_ORG
GHMIG_DESTINATION_TYPE=github
GHMIG_DESTINATION_BASE_URL=https://api.github.com
GHMIG_DESTINATION_TOKEN=ghp_YourGitHubPAT
```

#### YAML (`config.yml`)
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

#### Multi-Organization Support

The system automatically supports migrating from **multiple Azure DevOps organizations**:

1. **Configure one primary organization** in `base_url`
2. **Ensure your PAT has access to ALL organizations** you want to migrate from
3. **During discovery**, discover repos from any ADO organization
4. **During migration**, the system automatically:
   - Extracts the ADO org/project/repo from each repository's SourceURL
   - Creates a single GitHub migration source (`https://dev.azure.com`) per GitHub's API requirements
   - Uses each repository's full URL to specify the exact source location

**Example**: Your PAT can access `org1`, `org2`, and `org3`. Configure `base_url` as `org1`, then discover and migrate repos from all three organizations seamlessly.

**Technical Note**: Per GitHub's Enterprise Importer API documentation, the migration source URL must be the base Azure DevOps URL (`https://dev.azure.com`). The specific organization, project, and repository are specified in the `sourceRepositoryUrl` parameter during migration.

## 🔑 Environment Variable Naming

All environment variables use the `GHMIG_` prefix and follow this pattern:

| YAML Path | Environment Variable |
|-----------|---------------------|
| `server.port` | `GHMIG_SERVER_PORT` |
| `source.type` | `GHMIG_SOURCE_TYPE` |
| `source.base_url` | `GHMIG_SOURCE_BASE_URL` |
| `source.token` | `GHMIG_SOURCE_TOKEN` |
| `destination.token` | `GHMIG_DESTINATION_TOKEN` |
| `auth.enabled` | `GHMIG_AUTH_ENABLED` |
| `auth.entraid_enabled` | `GHMIG_AUTH_ENTRAID_ENABLED` |

**Rule**: Dots (`.`) in YAML paths become underscores (`_`) in environment variables, and all variables are prefixed with `GHMIG_`.

## 🎯 Configuration Precedence

Settings are applied in this order (later overrides earlier):

1. **Built-in defaults** (in the code)
2. **YAML config file** (`configs/config.yml`)
3. **Environment variables** (`.env` or system environment)

Example:
```yaml
# config.yml
migration:
  workers: 5
```

```bash
# .env
GHMIG_MIGRATION_WORKERS=10
```

**Result**: Workers = 10 (environment variable wins)

## 📖 Additional Resources

- **GitHub Setup**: See `docs/GITHUB_SECRETS_SETUP.md` or `docs/GITHUB_ENVIRONMENTS_SETUP.md`
- **Azure DevOps Setup**: See `docs/ADO_SETUP_GUIDE.md`
- **Deployment**: See `docs/DEPLOYMENT.md` or `docs/TERRAFORM_DEPLOYMENT_QUICKSTART.md`
- **API Documentation**: See `docs/API.md`

## 🔒 Security Best Practices

1. **Never commit secrets**
   - `.env` is gitignored
   - Use environment variables or secret management for production

2. **Rotate tokens regularly**
   - GitHub PATs: Every 90 days
   - Azure DevOps PATs: Every 90 days

3. **Use minimal token scopes**
   - Only grant required permissions
   - Consider GitHub Apps for better security

4. **Protect your configuration**
   - Keep `.env` file permissions: `chmod 600 .env`
   - Use secret managers in production (Azure Key Vault, AWS Secrets Manager, etc.)

## 🆘 Troubleshooting

### Configuration not loading?

1. **Check file location**: `.env` must be in project root (not in `configs/`)
2. **Check naming**: Environment variables need `GHMIG_` prefix
3. **Check format**: YAML indentation must be exact (use spaces, not tabs)

### Common mistakes:

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

### Enable debug logging:

```bash
GHMIG_LOGGING_LEVEL=debug
```

Then check logs at `./logs/migrator.log` for configuration details.

## 📝 Examples

See the example files for complete, commented configurations:

- `env.github.example` - GitHub source with all options
- `env.azuredevops.example` - Azure DevOps source with all options
- `config.github.yml` - GitHub YAML configuration
- `config.azuredevops.yml` - Azure DevOps YAML configuration
- `config_template.yml` - General template with all sources

Each file includes:
- ✅ Complete examples
- ✅ Inline documentation
- ✅ Common use cases
- ✅ Security notes

