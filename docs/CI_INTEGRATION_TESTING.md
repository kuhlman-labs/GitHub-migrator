# CI/CD Integration Testing Strategy

## Overview

This document outlines the strategy for running database integration tests in the CI/CD pipeline.

## Test Matrix

### Always Run: SQLite ✅
- **When**: Every PR, push to main/develop
- **Duration**: ~30 seconds
- **Cost**: Free (no external services)
- **Purpose**: Fast feedback, development verification

### Run on Main/PRs: PostgreSQL 🐘
- **When**: Every PR, push to main/develop, scheduled daily
- **Duration**: ~2 minutes
- **Cost**: Free (GitHub Actions service container)
- **Purpose**: Production database validation

### Optional: SQL Server 🏢
- **When**: 
  - Daily scheduled runs (2 AM UTC)
  - Manual workflow dispatch
  - Push to main branch (optional)
- **Duration**: ~5 minutes
- **Cost**: Free in CI (uses Developer edition)
- **Purpose**: Enterprise database compatibility

## CI Workflow Structure

```yaml
.github/workflows/integration-tests.yml
├── test-sqlite (always)
├── test-postgresql (always)
├── test-sqlserver (conditional)
└── integration-test-summary
```

## Triggering Tests

### 1. **Automatic Triggers**

#### On Pull Requests
```bash
# Tests run automatically on PR open/update
# Tests: SQLite + PostgreSQL
```

#### On Push to Main/Develop
```bash
# Tests run automatically on merge
# Tests: SQLite + PostgreSQL + SQL Server (optional)
```

#### Scheduled (Daily)
```bash
# Runs at 2 AM UTC daily
# Tests: SQLite + PostgreSQL + SQL Server (all)
```

#### Path Filters
Tests only run when relevant files change:
- `internal/storage/**`
- `internal/models/**`
- `go.mod`, `go.sum`
- Workflow file itself

### 2. **Manual Triggers**

#### Run All Tests Including SQL Server
```bash
# Via GitHub UI: Actions → Integration Tests → Run workflow
# Check the "Run SQL Server tests" option
```

#### Run via GitHub CLI
```bash
gh workflow run integration-tests.yml \
  -f test-sqlserver=true \
  -r main
```

## Test Environment Configuration

### SQLite
```yaml
# No external services required
# Uses in-memory or temp file database
CGO_ENABLED: 1
```

### PostgreSQL
```yaml
# GitHub Actions service container
services:
  postgres:
    image: postgres:16-alpine
    env:
      POSTGRES_DB: migrator_test
      POSTGRES_USER: migrator
      POSTGRES_PASSWORD: test_password_123
    ports:
      - 5432:5432

# Connection string
POSTGRES_TEST_DSN: "postgres://migrator:test_password_123@localhost:5432/migrator_test?sslmode=disable"
```

### SQL Server
```yaml
# Docker container (Docker-in-Docker)
# Started on-demand in workflow
docker run mcr.microsoft.com/mssql/server:2022-latest

# Connection string
SQLSERVER_TEST_DSN: "sqlserver://sa:YourStrong@Passw0rd@localhost:1433?database=migrator_test"
```

## Integration with Existing CI

### Updated CI Pipeline Flow

```
┌─────────────────────────────────────────┐
│  Pull Request / Push to Main/Develop   │
└────────────────┬────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
        ▼                 ▼
┌──────────────┐  ┌──────────────┐
│  Existing CI │  │ Integration  │
│   (ci.yml)   │  │    Tests     │
└──────┬───────┘  └──────┬───────┘
       │                 │
       ├─ Backend CI     ├─ SQLite Tests
       ├─ Frontend CI    ├─ PostgreSQL Tests
       ├─ Security Scan  └─ SQL Server Tests*
       ├─ Dependencies
       └─ Docker Build
                │
                ▼
        ┌───────────────┐
        │  All Passed?  │
        └───────┬───────┘
                │
                ▼
         ┌──────────────┐
         │ Deploy/Merge │
         └──────────────┘

* SQL Server: Only on schedule or manual trigger
```

### Modify Existing ci.yml

Add integration tests as a dependency for the summary job:

```yaml
# In .github/workflows/ci.yml
ci-summary:
  name: CI Summary
  runs-on: ubuntu-latest
  needs: 
    - backend-ci
    - frontend-ci
    - security-scan
    - dependency-check
    - docker-build-test
    # Add this:
    - integration-tests  # Wait for integration tests (if running)
  if: always()
```

## Test Execution Times

| Database   | Startup Time | Test Time | Total Time |
|------------|-------------|-----------|------------|
| SQLite     | 0s          | ~30s      | ~30s       |
| PostgreSQL | ~10s        | ~90s      | ~2min      |
| SQL Server | ~60s        | ~120s     | ~5min      |

## Cost Analysis

### GitHub Actions Minutes
- **Free tier**: 2,000 minutes/month (private repos)
- **Estimated usage**:
  - Per PR: ~3 minutes (SQLite + PostgreSQL)
  - Daily scheduled: ~8 minutes (all databases)
  - Monthly: ~50 PRs × 3min + 30 days × 8min = ~390 minutes

### Recommendations
- ✅ Always run SQLite (fast feedback)
- ✅ Always run PostgreSQL (production DB)
- ⚠️ SQL Server on schedule/main only (saves ~90 min/month)

## Best Practices

### 1. **Local Development**
Developers should run integration tests locally before pushing:
```bash
# Quick check with SQLite
make test-integration-sqlite

# Full check including PostgreSQL
make test-integration-postgres

# Complete validation (optional)
make test-integration
```

### 2. **PR Requirements**
Set branch protection rules requiring:
- ✅ Backend CI pass
- ✅ SQLite integration tests pass
- ✅ PostgreSQL integration tests pass
- ⚠️ SQL Server tests optional (don't block PRs)

### 3. **Monitoring**
Track test metrics:
- Test duration trends
- Failure rates by database
- Most common failure points

### 4. **Failure Handling**
When tests fail:
1. Check the specific database test logs
2. Look for dialect-specific SQL issues
3. Verify migration compatibility
4. Test locally with same database version

## Environment Variables

### Required for CI
```bash
# None - all test databases are ephemeral
```

### Optional Overrides
```bash
# Override PostgreSQL connection
POSTGRES_TEST_DSN="postgres://custom:connection@host:5432/db"

# Override SQL Server connection
SQLSERVER_TEST_DSN="sqlserver://custom:connection@host:1433?database=db"
```

## Debugging Failed Tests

### View Logs
```bash
# Via GitHub UI
Actions → Integration Tests → Failed job → Expand logs

# Via GitHub CLI
gh run view <run-id> --log-failed
```

### Reproduce Locally
```bash
# Use same connection strings as CI
export POSTGRES_TEST_DSN="postgres://migrator:test_password_123@localhost:5432/migrator_test?sslmode=disable"

# Run the same test
go test -tags=integration -v ./internal/storage -run TestIntegrationPostgreSQL
```

### Common Issues

#### 1. PostgreSQL Connection Timeout
```
Solution: Increase health check retries in workflow
```

#### 2. SQL Server Startup Timeout
```
Solution: SQL Server can take 60-90 seconds to start
- Increase wait timeout
- Ensure adequate runner resources
```

#### 3. Migration Failures
```
Solution: Check dialect-specific SQL syntax
- Verify AUTO_INCREMENT vs SERIAL vs IDENTITY
- Check DATETIME vs TIMESTAMP vs DATETIME2
- Validate string functions (SUBSTR vs SUBSTRING vs CHARINDEX)
```

## Future Enhancements

### Potential Improvements
1. **Test Coverage Reporting**
   - Track integration test coverage separately
   - Set coverage thresholds per database

2. **Performance Benchmarking**
   - Track query performance across databases
   - Alert on performance regressions

3. **Matrix Testing**
   - Test multiple database versions
   - Example: PostgreSQL 14, 15, 16

4. **Parallel Execution**
   - Run all database tests in parallel
   - Reduce total CI time

5. **Test Result Caching**
   - Cache test results for unchanged code
   - Only rerun affected tests

## Support

For issues with CI integration tests:
1. Check workflow logs in GitHub Actions
2. Review this documentation
3. Test locally using Makefile targets
4. Check `internal/storage/integration_test.go` for test details

## References

- [GitHub Actions Service Containers](https://docs.github.com/en/actions/using-containerized-services)
- [Integration Test Documentation](./GORM_REFACTORING_SUMMARY.md)
- [Migration README](../internal/storage/migrations/README.md)

