# GitHub Migrator - Operations Runbook

## Table of Contents

- [Daily Operations](#daily-operations)
- [Migration Workflows](#migration-workflows)
- [Monitoring & Alerts](#monitoring--alerts)
- [Incident Response](#incident-response)
- [Maintenance Tasks](#maintenance-tasks)
- [Troubleshooting Guide](#troubleshooting-guide)
- [Runbooks](#runbooks)

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

