# Docker Environment Configuration Guide

This guide explains how to pass configuration to the GitHub Migrator Docker containers.

## Quick Start

Your existing `.env` file will work automatically! Just run:

```bash
make docker-run-postgres
```

Docker Compose automatically reads `.env` from the project root.

## Methods to Pass Configuration

### Method 1: Using .env File (Recommended)

Docker Compose automatically reads a `.env` file from the same directory as `docker-compose.yml`.

**Your current setup:**
```bash
# Your existing .env file in project root
GHMIG_SOURCE_TOKEN=ghp_xxxxx
GHMIG_DESTINATION_TOKEN=ghp_yyyyy
# ... other variables
```

**To use it:**
```bash
# Variables are automatically available to containers
make docker-run-postgres
```

**How it works:**
- Docker Compose reads `.env` automatically
- Variables are available for substitution in compose files (using `${VAR_NAME}`)
- Variables can be passed to containers using the `environment` section

### Method 2: Explicit env_file Directive

If you want to use a different file or be explicit:

**Update `docker-compose.postgres.yml`:**
```yaml
services:
  migrator:
    env_file:
      - .env           # Your main env file
      - .env.local     # Optional overrides
```

**Then run:**
```bash
make docker-run-postgres
```

### Method 3: Command Line Environment Variables

Pass variables directly when running:

```bash
# Single variable
GHMIG_SOURCE_TOKEN=ghp_xxxxx docker compose -f docker-compose.yml -f docker-compose.postgres.yml up

# Multiple variables
GHMIG_SOURCE_TOKEN=ghp_xxxxx \
GHMIG_DESTINATION_TOKEN=ghp_yyyyy \
make docker-run-postgres
```

### Method 4: Explicit Variable Passing in docker-compose.yml

Pass specific variables from host to container:

**Update `docker-compose.postgres.yml`:**
```yaml
services:
  migrator:
    environment:
      - GHMIG_SOURCE_TOKEN=${GHMIG_SOURCE_TOKEN}
      - GHMIG_DESTINATION_TOKEN=${GHMIG_DESTINATION_TOKEN}
      # ... other variables
```

This reads from `.env` or host environment and passes to container.

## Configuration Precedence

When using Docker Compose, configuration is applied in this order (later overrides earlier):

1. Default values in `config_template.yml`
2. Environment variables in `.env` file
3. Environment variables defined in `docker-compose.yml`
4. Environment variables defined in `docker-compose.postgres.yml` (override file)
5. Environment variables from command line
6. Environment variables passed with `docker-compose run -e`

## Example .env File Structure

```bash
# ==============================================================================
# Required: GitHub Tokens
# ==============================================================================
GHMIG_SOURCE_TOKEN=ghp_your_source_token
GHMIG_DESTINATION_TOKEN=ghp_your_destination_token

# ==============================================================================
# Source Configuration
# ==============================================================================
GHMIG_SOURCE_TYPE=github
GHMIG_SOURCE_BASE_URL=https://api.github.com

# Optional: GitHub App (for better rate limits)
# GHMIG_SOURCE_APP_ID=123456
# GHMIG_SOURCE_APP_PRIVATE_KEY=/path/to/key.pem
# GHMIG_SOURCE_APP_INSTALLATION_ID=789012

# ==============================================================================
# Destination Configuration
# ==============================================================================
GHMIG_DESTINATION_TYPE=github
GHMIG_DESTINATION_BASE_URL=https://api.github.com

# ==============================================================================
# Database (set by docker-compose.postgres.yml, but can override)
# ==============================================================================
# GHMIG_DATABASE_TYPE=postgres
# GHMIG_DATABASE_DSN=postgres://migrator:migrator_dev_password@postgres:5432/migrator?sslmode=disable

# ==============================================================================
# Migration Settings
# ==============================================================================
GHMIG_MIGRATION_WORKERS=5
GHMIG_MIGRATION_POLL_INTERVAL_SECONDS=30
GHMIG_MIGRATION_POST_MIGRATION_MODE=production_only
GHMIG_MIGRATION_DEST_REPO_EXISTS_ACTION=fail

# Visibility handling
GHMIG_MIGRATION_VISIBILITY_HANDLING_PUBLIC_REPOS=private
GHMIG_MIGRATION_VISIBILITY_HANDLING_INTERNAL_REPOS=private

# ==============================================================================
# Logging
# ==============================================================================
GHMIG_LOGGING_LEVEL=info
GHMIG_LOGGING_FORMAT=json

# ==============================================================================
# Authentication (Optional)
# ==============================================================================
# GHMIG_AUTH_ENABLED=true
# GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID=Iv1.xxxxx
# GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET=secret_xxxxx
# GHMIG_AUTH_CALLBACK_URL=http://localhost:8080/api/v1/auth/callback
# GHMIG_AUTH_SESSION_SECRET=your_random_32_char_secret
# GHMIG_AUTH_SESSION_DURATION_HOURS=24
# GHMIG_AUTH_FRONTEND_URL=/

# Authorization (comma-separated lists)
# GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ORG_MEMBERSHIP=org1,org2
# GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_TEAM_MEMBERSHIP=org/team1,org/team2
```

## Verifying Configuration

### Check what environment variables are set in the container:

```bash
# For running container
docker exec github-migrator env | grep GHMIG

# Or check during startup
docker compose -f docker-compose.yml -f docker-compose.postgres.yml config

# Or run a one-off container to inspect
docker compose -f docker-compose.yml -f docker-compose.postgres.yml run --rm migrator env | grep GHMIG
```

### Check application logs for configuration:

```bash
make docker-logs-postgres
```

Look for startup logs that show loaded configuration.

## Troubleshooting

### Problem: Variables not being passed to container

**Solution 1:** Explicitly pass them in docker-compose:
```yaml
environment:
  - GHMIG_SOURCE_TOKEN=${GHMIG_SOURCE_TOKEN}
```

**Solution 2:** Use env_file directive:
```yaml
env_file:
  - .env
```

### Problem: .env file not being read

**Checklist:**
- Is `.env` in the same directory as `docker-compose.yml`?
- Are you running docker-compose from the project root?
- Does `.env` have proper Unix line endings (LF not CRLF)?
- Are there any syntax errors in `.env`? (no spaces around `=`)

### Problem: Some variables work, others don't

- Check for typos in variable names
- Ensure variables use the `GHMIG_` prefix
- Check for special characters that need escaping
- Verify no conflicting values in docker-compose files

## Best Practices

1. **Use `.env` for local development** - Gitignored by default
2. **Use `env_file` for shared defaults** - Can be committed (without secrets)
3. **Use command-line for CI/CD** - Secrets from secret managers
4. **Use docker-compose `environment`** - For container-specific overrides
5. **Never commit secrets** - Keep `.env` in `.gitignore`

## Additional Resources

- [Docker Compose Environment Variables](https://docs.docker.com/compose/environment-variables/)
- [GitHub Migrator Configuration Guide](../configs/config_template.yml)
- [Environment Variables Example](../configs/env.example)

