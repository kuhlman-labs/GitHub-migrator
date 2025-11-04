# GitHub Migrator - Deployment Guide

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Deployment Methods](#deployment-methods)
  - [Docker Deployment](#docker-deployment)
  - [Kubernetes Deployment](#kubernetes-deployment)
  - [Manual Deployment](#manual-deployment)
- [Configuration](#configuration)
- [Database Setup](#database-setup)
- [Monitoring](#monitoring)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

- **Docker** 20.10+ and Docker Compose 2.0+ (for containerized deployment)
- **Go** 1.21+ (for manual deployment)
- **Node.js** 20+ and npm (for building frontend)
- **Git** 2.30+ with Git LFS support
- **git-sizer** (for repository analysis)

### GitHub Tokens

You need two GitHub Personal Access Tokens (PATs) with organization admin access:

1. **Source Token** (GitHub Enterprise Server)
   - **Requirements**: Organization admin access
   - Scopes: `repo`, `read:org`, `read:user`, `admin:org`
   - Used for discovering, reading, and migrating source repositories

2. **Destination Token** (GitHub Enterprise Cloud)
   - **Requirements**: Organization admin access
   - Scopes: `repo`, `admin:org`, `workflow`
   - Used for creating migrations and importing repositories

### Network Requirements

- Outbound access to GitHub Enterprise Server (source)
- Outbound access to GitHub Enterprise Cloud (destination)
- Inbound access for web UI (default port 8080)

---

## Environment Setup

### 1. Clone Repository

```bash
git clone https://github.com/your-org/github-migrator.git
cd github-migrator
```

### 2. Configure Environment Variables

Create a `.env` file:

```bash
# GitHub Source (Enterprise Server)
export GITHUB_SOURCE_TOKEN="ghp_xxxxxxxxxxxx"
export GITHUB_SOURCE_BASE_URL="https://github.company.com/api/v3"

# GitHub Destination (Enterprise Cloud)
export GITHUB_DEST_TOKEN="ghp_yyyyyyyyyyyy"
export GITHUB_DEST_BASE_URL="https://api.github.com"

# Server Configuration
export GHMIG_SERVER_PORT=8080
export GHMIG_DATABASE_TYPE=sqlite
export GHMIG_DATABASE_DSN=/app/data/migrator.db
export GHMIG_LOGGING_LEVEL=info
export GHMIG_LOGGING_FORMAT=json
```

Load environment variables:

```bash
source .env
```

---

## Deployment Methods

### Docker Deployment (Recommended)

#### Quick Start

```bash
# 1. Build the Docker image
make docker-build

# 2. Run with Docker Compose
make docker-run

# Access UI at http://localhost:8080
```

#### Manual Docker Steps

```bash
# Build image
docker build -t github-migrator:latest .

# Run container
docker run -d \
  --name github-migrator \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  -v $(pwd)/configs:/app/configs \
  -e GITHUB_SOURCE_TOKEN="${GITHUB_SOURCE_TOKEN}" \
  -e GITHUB_DEST_TOKEN="${GITHUB_DEST_TOKEN}" \
  -e GHMIG_SERVER_PORT=8080 \
  github-migrator:latest

# View logs
docker logs -f github-migrator

# Stop container
docker stop github-migrator
docker rm github-migrator
```

#### Docker Compose

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

**Commands:**

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Restart
docker-compose restart

# Update and restart
docker-compose pull
docker-compose up -d
```

---

### Kubernetes Deployment

#### 1. Create Namespace

```bash
kubectl create namespace github-migrator
```

#### 2. Create Secrets

```bash
# Create GitHub tokens secret
kubectl create secret generic github-tokens \
  --from-literal=source-token="${GITHUB_SOURCE_TOKEN}" \
  --from-literal=dest-token="${GITHUB_DEST_TOKEN}" \
  -n github-migrator
```

#### 3. Create ConfigMap

**config-map.yaml:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migrator-config
  namespace: github-migrator
data:
  config.yaml: |
    server:
      port: 8080
    database:
      type: sqlite
      dsn: /app/data/migrator.db
    github:
      source:
        base_url: "https://github.company.com/api/v3"
      destination:
        base_url: "https://api.github.com"
    logging:
      level: info
      format: json
      output_file: /app/logs/migrator.log
```

Apply:

```bash
kubectl apply -f config-map.yaml
```

#### 4. Create Deployment

**deployment.yaml:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: github-migrator
  namespace: github-migrator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: github-migrator
  template:
    metadata:
      labels:
        app: github-migrator
    spec:
      containers:
      - name: migrator
        image: github-migrator:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: GITHUB_SOURCE_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-tokens
              key: source-token
        - name: GITHUB_DEST_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-tokens
              key: dest-token
        - name: GHMIG_SERVER_PORT
          value: "8080"
        - name: GHMIG_DATABASE_TYPE
          value: "sqlite"
        - name: GHMIG_DATABASE_DSN
          value: "/app/data/migrator.db"
        - name: GHMIG_LOGGING_LEVEL
          value: "info"
        volumeMounts:
        - name: data
          mountPath: /app/data
        - name: logs
          mountPath: /app/logs
        - name: config
          mountPath: /app/configs
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: migrator-data
      - name: logs
        persistentVolumeClaim:
          claimName: migrator-logs
      - name: config
        configMap:
          name: migrator-config
```

#### 5. Create Persistent Volumes

**pvc.yaml:**

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: migrator-data
  namespace: github-migrator
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: migrator-logs
  namespace: github-migrator
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

Apply:

```bash
kubectl apply -f pvc.yaml
kubectl apply -f deployment.yaml
```

#### 6. Create Service

**service.yaml:**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: github-migrator
  namespace: github-migrator
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: github-migrator
```

Apply:

```bash
kubectl apply -f service.yaml
```

#### 7. Create Ingress (Optional)

**ingress.yaml:**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: github-migrator
  namespace: github-migrator
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - migrator.company.com
    secretName: migrator-tls
  rules:
  - host: migrator.company.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: github-migrator
            port:
              number: 80
```

Apply:

```bash
kubectl apply -f ingress.yaml
```

#### Verify Deployment

```bash
# Check pods
kubectl get pods -n github-migrator

# Check service
kubectl get svc -n github-migrator

# View logs
kubectl logs -f deployment/github-migrator -n github-migrator

# Get service URL
kubectl get svc github-migrator -n github-migrator
```

---

### Manual Deployment

For non-containerized deployment on Linux/macOS servers.

#### 1. Install Dependencies

```bash
# Install Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install git-sizer
go install github.com/github/git-sizer@latest
```

#### 2. Build Application

```bash
# Clone repository
git clone https://github.com/your-org/github-migrator.git
cd github-migrator

# Install dependencies
make install

# Build backend and frontend
make build-all
```

#### 3. Configure Service

Create systemd service file `/etc/systemd/system/github-migrator.service`:

```ini
[Unit]
Description=GitHub Migrator
After=network.target

[Service]
Type=simple
User=migrator
Group=migrator
WorkingDirectory=/opt/github-migrator
ExecStart=/opt/github-migrator/bin/github-migrator-server
Restart=on-failure
RestartSec=10

# Environment
Environment="GITHUB_SOURCE_TOKEN=ghp_xxxxxxxxxxxx"
Environment="GITHUB_DEST_TOKEN=ghp_yyyyyyyyyyyy"
Environment="GHMIG_SERVER_PORT=8080"
Environment="GHMIG_DATABASE_TYPE=sqlite"
Environment="GHMIG_DATABASE_DSN=/opt/github-migrator/data/migrator.db"
Environment="GHMIG_LOGGING_LEVEL=info"

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/github-migrator/data /opt/github-migrator/logs

[Install]
WantedBy=multi-user.target
```

#### 4. Deploy

```bash
# Create user
sudo useradd -r -s /bin/false migrator

# Copy files
sudo mkdir -p /opt/github-migrator
sudo cp -r . /opt/github-migrator/
sudo chown -R migrator:migrator /opt/github-migrator

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable github-migrator
sudo systemctl start github-migrator

# Check status
sudo systemctl status github-migrator

# View logs
sudo journalctl -u github-migrator -f
```

---

## Configuration

### Configuration File

The application uses `configs/config.yaml`:

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  type: sqlite          # or postgresql
  dsn: ./data/migrator.db
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

github:
  source:
    base_url: "https://github.company.com/api/v3"
    token: "${GITHUB_SOURCE_TOKEN}"
  destination:
    base_url: "https://api.github.com"
    token: "${GITHUB_DEST_TOKEN}"
  rate_limit:
    requests_per_hour: 5000
    wait_on_exhaustion: true
  retry:
    max_attempts: 3
    initial_backoff: 1s
    max_backoff: 30s

logging:
  level: info           # debug, info, warn, error
  format: json          # json, text
  output_file: ./logs/migrator.log
  max_size: 100         # MB
  max_backups: 3
  max_age: 28           # days
  compress: true

migration:
  parallel_workers: 5
  timeout: 30m
  dry_run_default: false

discovery:
  parallel_workers: 10
  profile_depth: full   # basic, standard, full
```

### Environment Variable Override

All configuration values can be overridden with environment variables using the `GHMIG_` prefix:

```bash
GHMIG_SERVER_PORT=9090
GHMIG_DATABASE_TYPE=postgresql
GHMIG_DATABASE_DSN="postgres://user:pass@localhost/migrator"
GHMIG_LOGGING_LEVEL=debug
```

### Production Configuration

For production deployments, copy `configs/config_template.yml` to `configs/config.yaml` and customize:

**Example production config:**

```yaml
server:
  port: 8080

database:
  type: postgres
  dsn: "${DATABASE_URL}"  # postgres://user:pass@host:5432/migrator?sslmode=require

source:
  type: github
  base_url: "${GITHUB_SOURCE_URL}"
  token: "${GITHUB_SOURCE_TOKEN}"

destination:
  type: github
  base_url: "${GITHUB_DEST_URL}"
  token: "${GITHUB_DEST_TOKEN}"

migration:
  workers: 10
  poll_interval_seconds: 30
  post_migration_mode: "production_only"
  dest_repo_exists_action: "fail"
  visibility_handling:
    public_repos: "private"
    internal_repos: "private"

logging:
  level: info
  format: json
  output_file: /var/log/github-migrator/migrator.log
  max_size: 500
  max_backups: 10
  max_age: 90

auth:
  enabled: true
  github_oauth_client_id: "${GITHUB_OAUTH_CLIENT_ID}"
  github_oauth_client_secret: "${GITHUB_OAUTH_CLIENT_SECRET}"
  callback_url: "${GITHUB_OAUTH_CALLBACK_URL}"
  session_secret: "${AUTH_SESSION_SECRET}"
  session_duration_hours: 24
```

**Note:** See `configs/config_template.yml` for complete documentation of all available options.

---

## Database Setup

### SQLite (Development/Small Scale)

SQLite is used by default for simplicity:

```yaml
database:
  type: sqlite
  dsn: ./data/migrator.db
```

**Pros:**
- No external database required
- Easy setup and backup
- Good for < 10,000 repositories

**Cons:**
- Limited concurrency
- Not suitable for high-scale operations

### PostgreSQL (Production/Large Scale)

For production deployments:

#### 1. Install PostgreSQL

```bash
# Ubuntu/Debian
sudo apt-get install postgresql postgresql-contrib

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### 2. Create Database

```sql
-- Connect as postgres user
sudo -u postgres psql

-- Create database and user
CREATE DATABASE migrator;
CREATE USER migrator_user WITH ENCRYPTED PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE migrator TO migrator_user;

-- Exit
\q
```

#### 3. Configure Application

```yaml
database:
  type: postgresql
  dsn: "host=localhost port=5432 user=migrator_user password=secure_password dbname=migrator sslmode=disable"
  max_open_conns: 50
  max_idle_conns: 10
```

Or use DATABASE_URL:

```bash
export DATABASE_URL="postgres://migrator_user:secure_password@localhost:5432/migrator?sslmode=disable"
```

#### 4. Run Migrations

Migrations run automatically on startup. To manually apply:

```bash
# Check current version
sqlite3 data/migrator.db "SELECT version FROM schema_migrations;"

# Migrations are in internal/storage/migrations/
```

---

## Monitoring

### Health Checks

```bash
# Health endpoint
curl http://localhost:8080/health

# Expected response
{
  "status": "healthy",
  "time": "2024-01-15T10:00:00Z"
}
```

### Metrics (Future)

Prometheus metrics will be available at `/metrics`:

```
# HELP migration_total Total number of migrations
# TYPE migration_total counter
migration_total{status="completed"} 150
migration_total{status="failed"} 5

# HELP migration_duration_seconds Migration duration
# TYPE migration_duration_seconds histogram
migration_duration_seconds_bucket{le="60"} 50
migration_duration_seconds_bucket{le="300"} 120
```

### Logs

Logs are written to:
- Console (stdout) - Human-readable format
- File - JSON format for parsing

**View logs:**

```bash
# Docker
docker logs -f github-migrator

# Systemd
sudo journalctl -u github-migrator -f

# File
tail -f logs/migrator.log
```

**Log levels:**
- `debug` - Detailed debugging information
- `info` - General informational messages
- `warn` - Warning messages
- `error` - Error messages

### Alerting

Set up alerts for:

1. **Health Check Failures**
   ```bash
   # Cron job to check health
   */5 * * * * curl -f http://localhost:8080/health || alert-script.sh
   ```

2. **High Error Rate**
   - Monitor logs for ERROR level messages
   - Alert if > 10 errors in 5 minutes

3. **Disk Space**
   - Monitor data/ and logs/ directories
   - Alert at 80% usage

4. **Migration Failures**
   - Track failed migrations from API
   - Alert if > 5% failure rate

---

## Security

### 1. Token Management

**DO:**
- Store tokens in environment variables or secret managers
- Use different tokens for source and destination
- Rotate tokens regularly (every 90 days)
- Use tokens with minimum required scopes

**DON'T:**
- Commit tokens to version control
- Share tokens between environments
- Use admin/owner tokens unnecessarily

### 2. Network Security

```bash
# Firewall rules (UFW)
sudo ufw allow 8080/tcp  # API access
sudo ufw enable
```

For production, use reverse proxy:

**Nginx config:**

```nginx
server {
    listen 80;
    server_name migrator.company.com;
    
    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name migrator.company.com;
    
    ssl_certificate /etc/ssl/certs/migrator.crt;
    ssl_certificate_key /etc/ssl/private/migrator.key;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 3. Database Security

- Use strong passwords
- Enable SSL/TLS for PostgreSQL connections
- Restrict database access to application only
- Regular backups

### 4. Application Security

```bash
# Run security scan
make lint  # includes gosec

# Update dependencies
go get -u ./...
go mod tidy
```

---

## Backup and Recovery

### Backup Strategy

#### SQLite Backup

```bash
# Manual backup
sqlite3 data/migrator.db ".backup data/migrator-backup-$(date +%Y%m%d).db"

# Automated backup script
#!/bin/bash
BACKUP_DIR="/backups/github-migrator"
DATE=$(date +%Y%m%d-%H%M%S)
sqlite3 /app/data/migrator.db ".backup ${BACKUP_DIR}/migrator-${DATE}.db"
# Keep last 30 days
find ${BACKUP_DIR} -name "migrator-*.db" -mtime +30 -delete
```

#### PostgreSQL Backup

```bash
# Manual backup
pg_dump -h localhost -U migrator_user migrator > migrator-backup-$(date +%Y%m%d).sql

# Automated backup
#!/bin/bash
BACKUP_DIR="/backups/github-migrator"
DATE=$(date +%Y%m%d-%H%M%S)
pg_dump -h localhost -U migrator_user migrator | gzip > ${BACKUP_DIR}/migrator-${DATE}.sql.gz
# Keep last 30 days
find ${BACKUP_DIR} -name "migrator-*.sql.gz" -mtime +30 -delete
```

### Recovery

#### SQLite Recovery

```bash
# Stop application
docker-compose down

# Restore from backup
cp data/migrator-backup-20240115.db data/migrator.db

# Start application
docker-compose up -d
```

#### PostgreSQL Recovery

```bash
# Stop application
docker-compose down

# Drop and recreate database
sudo -u postgres psql -c "DROP DATABASE migrator;"
sudo -u postgres psql -c "CREATE DATABASE migrator;"

# Restore from backup
psql -h localhost -U migrator_user migrator < migrator-backup-20240115.sql

# Start application
docker-compose up -d
```

---

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

**Error:** `bind: address already in use`

**Solution:**
```bash
# Find process using port 8080
lsof -i :8080
sudo kill -9 <PID>

# Or use different port
export GHMIG_SERVER_PORT=8081
```

#### 2. Database Locked (SQLite)

**Error:** `database is locked`

**Solution:**
- Only one process can write to SQLite
- Use PostgreSQL for concurrent access
- Check for hung processes:
  ```bash
  ps aux | grep github-migrator
  ```

#### 3. GitHub API Rate Limit

**Error:** `API rate limit exceeded`

**Solution:**
- Server automatically waits when rate limited
- Check token permissions
- Consider using multiple tokens
- Monitor rate limit:
  ```bash
  curl http://localhost:8080/api/v1/analytics/summary
  ```

#### 4. Migration Stuck

**Symptoms:** Migration stays in "migrating" status

**Solution:**
```bash
# Check migration logs
curl http://localhost:8080/api/v1/migrations/{id}/logs

# Check GitHub migration status directly
# https://docs.github.com/en/rest/migrations

# Cancel stuck migration (if needed)
# Update status in database
```

#### 5. Out of Memory

**Error:** `cannot allocate memory`

**Solution:**
```bash
# Increase Docker memory limit
docker update --memory 2g github-migrator

# Reduce parallel workers in config
migration:
  parallel_workers: 3
discovery:
  parallel_workers: 5
```

### Debug Mode

Enable debug logging:

```bash
export GHMIG_LOGGING_LEVEL=debug
docker-compose restart
```

View detailed logs:

```bash
docker-compose logs -f | grep DEBUG
```

---

## Scaling

### Horizontal Scaling

**Current Limitation:** SQLite doesn't support horizontal scaling

**PostgreSQL Setup:**

1. Deploy multiple instances
2. Use shared PostgreSQL database
3. Add load balancer
4. Ensure proper locking for migrations

**Example with multiple containers:**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: migrator
      POSTGRES_USER: migrator
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data

  migrator-1:
    image: github-migrator:latest
    environment:
      - GHMIG_DATABASE_TYPE=postgresql
      - GHMIG_DATABASE_DSN=postgres://migrator:secret@postgres:5432/migrator

  migrator-2:
    image: github-migrator:latest
    environment:
      - GHMIG_DATABASE_TYPE=postgresql
      - GHMIG_DATABASE_DSN=postgres://migrator:secret@postgres:5432/migrator

  nginx:
    image: nginx:alpine
    ports:
      - "8080:80"
    depends_on:
      - migrator-1
      - migrator-2

volumes:
  pgdata:
```

### Vertical Scaling

Increase resources for single instance:

```yaml
services:
  migrator:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

---

## Maintenance

### Updates

```bash
# Pull latest code
git pull origin main

# Rebuild and restart
make docker-build
docker-compose down
docker-compose up -d
```

### Log Rotation

Logs automatically rotate based on config:

```yaml
logging:
  max_size: 100      # MB per file
  max_backups: 3     # Keep 3 old files
  max_age: 28        # Days to keep
  compress: true     # Compress rotated files
```

Manual log cleanup:

```bash
# Clean old logs
find logs/ -name "*.log.*" -mtime +30 -delete
```

### Database Maintenance

#### SQLite

```bash
# Optimize database
sqlite3 data/migrator.db "VACUUM;"

# Check integrity
sqlite3 data/migrator.db "PRAGMA integrity_check;"
```

#### PostgreSQL

```bash
# Vacuum and analyze
psql -U migrator_user -d migrator -c "VACUUM ANALYZE;"

# Check database size
psql -U migrator_user -d migrator -c "SELECT pg_size_pretty(pg_database_size('migrator'));"
```

---

## Support

For additional help:

- **Project Overview:** [README.md](../README.md)
- **API Documentation:** [API.md](./API.md)
- **Operations Guide:** [OPERATIONS.md](./OPERATIONS.md)
- **Implementation Guide:** [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)
- **Contributing Guide:** [CONTRIBUTING.md](./CONTRIBUTING.md)

---

**Deployment Guide Version:** 1.0.0  
**Last Updated:** October 2025  
**Status:** Production Ready

