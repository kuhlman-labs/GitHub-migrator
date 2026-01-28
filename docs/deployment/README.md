# Deployment Guide

This guide covers deploying GitHub Migrator to various environments.

## Deployment Options

| Platform | Best For | Guide |
|----------|----------|-------|
| [Docker](#docker-quick-start) | Local testing, small deployments | This page |
| [Azure App Service](./AZURE.md) | Azure-native deployments with managed infrastructure | Azure Guide |
| [Kubernetes](./KUBERNETES.md) | Large-scale, highly available deployments | Kubernetes Guide |

## Prerequisites

### Required Software

- **Docker** 20.10+ and Docker Compose 2.0+
- **Git** 2.30+ with Git LFS support

### GitHub Tokens

You need two GitHub Personal Access Tokens (PATs):

| Token | Scopes | Purpose |
|-------|--------|---------|
| **Source** | `repo`, `read:org`, `read:user`, `admin:org` | Discover and read source repositories |
| **Destination** | `repo`, `admin:org`, `workflow` | Create migrations and import repositories |

### Network Requirements

- Outbound: GitHub Enterprise Server (source) and GitHub.com (destination)
- Inbound: Port 8080 for web UI

---

## Docker Quick Start

### 1. Configure Environment

```bash
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"
```

### 2. Build and Run

```bash
# Build the Docker image
make docker-build

# Run with Docker Compose
make docker-run

# Access UI at http://localhost:8080
```

### 3. Verify

```bash
curl http://localhost:8080/health
# Expected: {"status":"healthy","time":"..."}
```

---

## Docker Compose Configuration

The included `docker-compose.yml` provides a complete setup:

```yaml
version: '3.8'

services:
  migrator:
    build: .
    image: github-migrator:latest
    container_name: github-migrator
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
      - ./configs:/app/configs
    environment:
      - GHMIG_SERVER_PORT=8080
      - GHMIG_DATABASE_TYPE=sqlite
      - GHMIG_DATABASE_DSN=/app/data/migrator.db
      - GHMIG_LOGGING_LEVEL=info
      - GHMIG_LOGGING_FORMAT=json
      - GITHUB_SOURCE_TOKEN=${GITHUB_SOURCE_TOKEN}
      - GITHUB_DEST_TOKEN=${GITHUB_DEST_TOKEN}
    restart: unless-stopped
```

### Commands

```bash
docker-compose up -d        # Start
docker-compose logs -f      # View logs
docker-compose down         # Stop
docker-compose restart      # Restart
```

---

## Configuration

### Environment Variables

All configuration can be set via environment variables with the `GHMIG_` prefix:

```bash
# Server
GHMIG_SERVER_PORT=8080

# Database
GHMIG_DATABASE_TYPE=sqlite          # or postgresql
GHMIG_DATABASE_DSN=/app/data/migrator.db

# Logging
GHMIG_LOGGING_LEVEL=info            # debug, info, warn, error
GHMIG_LOGGING_FORMAT=json           # json, text

# Source GitHub
GITHUB_SOURCE_TOKEN=ghp_xxx
GHMIG_SOURCE_BASE_URL=https://github.company.com/api/v3

# Destination GitHub
GITHUB_DEST_TOKEN=ghp_yyy
GHMIG_DESTINATION_BASE_URL=https://api.github.com
```

### Configuration File

Alternatively, use `configs/config.yaml`. See [config_template.yml](../../configs/config_template.yml) for all options.

---

## Database Options

### SQLite (Default)

Good for development and small deployments (<10,000 repositories):

```yaml
database:
  type: sqlite
  dsn: ./data/migrator.db
```

### PostgreSQL (Production)

Recommended for production deployments:

```yaml
database:
  type: postgresql
  dsn: "host=localhost port=5432 user=migrator password=secret dbname=migrator sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
```

See [OPERATIONS.md](../OPERATIONS.md#database-setup) for PostgreSQL setup instructions.

---

## Copilot Authentication

The Copilot Assistant feature requires authentication to GitHub's Copilot API. There are several ways to configure this:

### Local Development

**Option A: Interactive Login (Easiest)**

```bash
# Install the Copilot CLI
brew install copilot-cli  # macOS
# or: npm install -g @github/copilot

# Authenticate via browser
copilot auth login
```

This stores credentials in `~/.copilot/` and the SDK will automatically use them.

**Option B: Token-Based**

Create a fine-grained PAT at https://github.com/settings/personal-access-tokens/new with the **"Copilot Requests"** permission enabled, then:

```bash
export GH_TOKEN=github_pat_xxx
make run-server
```

### Server Deployment

For production/server deployments, set the token via environment variable:

```bash
# In Docker/Kubernetes environment
GH_TOKEN=github_pat_xxx  # Service account PAT with "Copilot Requests" permission
```

Or use the app-specific environment variable:

```bash
GHMIG_COPILOT_GH_TOKEN=github_pat_xxx
```

### Token Priority

The system checks for Copilot authentication in this order:

1. Token configured in Settings UI (stored in database)
2. `GHMIG_COPILOT_GH_TOKEN` environment variable
3. `GH_TOKEN` or `GITHUB_TOKEN` environment variable
4. Cached credentials from `copilot auth login` in `~/.copilot/`

### Requirements

- A GitHub account with an active Copilot subscription (Individual, Business, or Enterprise)
- For token-based auth: A fine-grained PAT with "Copilot Requests" permission

---

## Production Considerations

### Security

- Use HTTPS in production (reverse proxy with TLS)
- Store tokens in secret management systems
- Enable OAuth authentication for user access control
- Configure CORS for your domain

### Monitoring

- Health endpoint: `GET /health`
- Application logs in `./logs/migrator.log`
- Set up alerts for failed migrations

### Backup

- SQLite: Regular file backups of `./data/migrator.db`
- PostgreSQL: Standard pg_dump procedures

---

## Next Steps

- [Azure Deployment](./AZURE.md) - Deploy to Azure App Service
- [Kubernetes Deployment](./KUBERNETES.md) - Deploy to Kubernetes
- [Operations Guide](../OPERATIONS.md) - Authentication, monitoring, troubleshooting

