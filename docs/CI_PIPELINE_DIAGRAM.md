# CI/CD Pipeline Architecture

## Overview Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                     Code Push / Pull Request                      │
└───────────────────────────┬──────────────────────────────────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
                ▼                       ▼
┌───────────────────────┐   ┌───────────────────────┐
│   Main CI Pipeline    │   │  Integration Tests    │
│     (ci.yml)          │   │ (integration-tests)   │
└───────┬───────────────┘   └───────┬───────────────┘
        │                           │
        ├─┬─┬─┬─┬─┐                ├─┬─┬─┐
        │ │ │ │ │ │                │ │ │ │
        ▼ ▼ ▼ ▼ ▼ ▼                ▼ ▼ ▼ ▼
        A B C D E F                G H I J

A: Backend CI (Go)
B: Frontend CI (React/TS)  
C: Security Scanning
D: Dependency Check
E: Docker Build Test
F: CI Summary

G: SQLite Tests (always)
H: PostgreSQL Tests (always)
I: SQL Server Tests (conditional)
J: Integration Summary

        │                           │
        └───────────┬───────────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │   All Tests Pass?     │
        └───────┬───────────────┘
                │
        ┌───────┴────────┐
        │                │
        ▼                ▼
     Success          Failure
        │                │
        ▼                ▼
  ┌─────────┐      ┌─────────┐
  │ Merge/  │      │ Block & │
  │ Deploy  │      │ Notify  │
  └─────────┘      └─────────┘
```

## Test Execution Flow

### 1. Pull Request Flow
```
PR Created/Updated
    │
    ├─> Backend Tests (2-3 min)
    │   └─> Go tests, linting, coverage
    │
    ├─> Frontend Tests (2-3 min)
    │   └─> TypeScript, ESLint, build
    │
    ├─> Security Scans (2-3 min)
    │   └─> Trivy, Gosec
    │
    ├─> Integration Tests (3-4 min)
    │   ├─> SQLite (30s)
    │   └─> PostgreSQL (2m)
    │
    └─> Docker Build (3-4 min)
        └─> Build validation

Total: ~10-15 minutes (parallel execution)
```

### 2. Main Branch Flow
```
Merge to Main
    │
    ├─> All PR checks (as above)
    │
    ├─> Additional Tests
    │   └─> SQL Server (5m) [optional]
    │
    └─> Deploy Workflows
        ├─> Build & Push Image
        └─> Deploy to Environments
```

### 3. Scheduled Flow (Daily)
```
Cron Trigger (2 AM UTC)
    │
    └─> Full Integration Suite
        ├─> SQLite (30s)
        ├─> PostgreSQL (2m)
        └─> SQL Server (5m)

Total: ~8 minutes
Purpose: Catch environment drift, dependency issues
```

## Database Test Strategy

### Quick Feedback Loop (Every PR)
```
┌─────────────┐
│ Developer   │
│ Pushes Code │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  SQLite     │  ← Fastest (30s)
│  Tests      │    No dependencies
└──────┬──────┘    High confidence
       │
       ▼
┌─────────────┐
│ PostgreSQL  │  ← Production DB (2m)
│  Tests      │    Real-world validation
└──────┬──────┘    
       │
       ▼
┌─────────────┐
│   Merge?    │  Decision point
└─────────────┘
```

### Comprehensive Validation (Scheduled/Manual)
```
┌─────────────┐
│  Scheduled  │
│   or        │
│  Manual     │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  All DBs    │  ← Complete validation
│  + Versions │    Multiple versions
└──────┬──────┘    Edge case testing
       │
       ▼
┌─────────────┐
│   Report    │  Metrics & trends
└─────────────┘
```

## Cost & Time Analysis

### Per Pull Request
| Stage              | Duration | Cost      | Blockable |
|--------------------|----------|-----------|-----------|
| Backend CI         | 2-3 min  | Free      | ✅ Yes    |
| Frontend CI        | 2-3 min  | Free      | ✅ Yes    |
| SQLite Tests       | 30s      | Free      | ✅ Yes    |
| PostgreSQL Tests   | 2 min    | Free      | ✅ Yes    |
| Security Scan      | 2-3 min  | Free      | ⚠️ Warn   |
| Docker Build       | 3-4 min  | Free      | ✅ Yes    |
| **Total**          | **~10-15 min** | **Free** | **-** |

### Per Main Branch Push
| Stage              | Duration | Cost      | Blockable |
|--------------------|----------|-----------|-----------|
| All PR checks      | 10-15 min| Free      | ✅ Yes    |
| SQL Server Tests*  | 5 min    | Free      | ⚠️ Optional |
| **Total**          | **~15-20 min** | **Free** | **-** |

*Only on main branch or scheduled

### Monthly Estimate
```
Assumptions:
- 50 PRs/month
- 30 scheduled runs/month
- 10 main pushes/month

Calculation:
  PRs:        50 × 15 min = 750 min
  Scheduled:  30 × 8 min  = 240 min
  Main:       10 × 20 min = 200 min
  ─────────────────────────────────
  Total:                   1,190 min (~20 hours)

GitHub Free Tier: 2,000 min/month
Remaining: ~800 min buffer
```

## Conditional Execution Rules

### SQLite Tests
```yaml
Runs: Always
Conditions: None
Reason: Fast, no dependencies, catches most issues
```

### PostgreSQL Tests
```yaml
Runs: Always (except scheduled SQL Server-only runs)
Conditions: 
  - Pull requests
  - Push to main/develop
  - File changes in storage layer
Reason: Production database, critical validation
```

### SQL Server Tests
```yaml
Runs: Conditionally
Conditions:
  - Scheduled daily at 2 AM UTC
  - Manual workflow dispatch
  - Push to main branch (optional)
Reason: Expensive startup time, enterprise validation
```

## Branch Protection Settings

### Recommended Configuration

#### For `main` branch:
```yaml
Required Status Checks:
  ✅ Backend CI (Go)
  ✅ Frontend CI (React/TypeScript)
  ✅ Integration Tests - SQLite
  ✅ Integration Tests - PostgreSQL
  ⚠️ Docker Build Test
  ❌ SQL Server (optional)
  ❌ Security Scan (warning only)
```

#### For `develop` branch:
```yaml
Required Status Checks:
  ✅ Backend CI (Go)
  ✅ Integration Tests - SQLite
  ⚠️ Integration Tests - PostgreSQL
  ❌ Everything else (optional)
```

## Failure Handling

### Automatic Retries
```
┌──────────────┐
│  Test Fails  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Transient?  │  (Network, timeout, etc.)
└──────┬───────┘
       │
   ┌───┴───┐
   │       │
   ▼       ▼
  Yes      No
   │       │
   ▼       ▼
Retry   Report
  │       │
  ▼       ▼
Pass?   Block
```

### Notification Flow
```
Test Failure
    │
    ├─> GitHub Check ❌
    │
    ├─> PR Comment (auto)
    │   └─> "Integration tests failed
    │        Database: PostgreSQL
    │        Error: Connection timeout"
    │
    ├─> Email (if configured)
    │
    └─> Slack (if configured)
```

## Monitoring & Metrics

### Key Metrics to Track
1. **Test Duration Trends**
   - SQLite: Should stay ~30s
   - PostgreSQL: Should stay ~2min
   - SQL Server: Should stay ~5min

2. **Failure Rates**
   - By database type
   - By test category
   - By time of day

3. **Flaky Tests**
   - Tests that fail intermittently
   - Need retry or fix

4. **Resource Usage**
   - GitHub Actions minutes consumed
   - Peak concurrent jobs
   - Database container resources

### Dashboard Example
```
┌─────────────────────────────────────┐
│  Integration Tests - Last 30 Days   │
├─────────────────────────────────────┤
│  SQLite:     ████████████ 98% pass  │
│  PostgreSQL: ███████████░ 95% pass  │
│  SQL Server: ██████████░░ 92% pass  │
├─────────────────────────────────────┤
│  Avg Duration: 12 min                │
│  Total Runs: 150                     │
│  Minutes Used: 1,800                 │
└─────────────────────────────────────┘
```

## Rollout Strategy

### Phase 1: Non-Blocking (Week 1)
- ✅ Deploy integration test workflow
- ✅ Run tests but don't block PRs
- ✅ Monitor for issues
- ✅ Fix any flaky tests

### Phase 2: SQLite Required (Week 2)
- ✅ Make SQLite tests required
- ✅ Continue monitoring
- ✅ Adjust timeouts if needed

### Phase 3: PostgreSQL Required (Week 3)
- ✅ Make PostgreSQL tests required
- ✅ Full integration with branch protection
- ✅ Document any known issues

### Phase 4: Full Automation (Week 4+)
- ✅ SQL Server on schedule
- ✅ Metrics dashboard
- ✅ Automated reporting

## Troubleshooting Guide

### "PostgreSQL connection timeout"
```bash
Solution: Increase health check retries
Location: .github/workflows/integration-tests.yml
Change: health-retries from 5 to 10
```

### "SQL Server not starting"
```bash
Solution: Increase wait time
Location: integration-tests.yml, Wait for SQL Server step
Change: Timeout from 60s to 120s
```

### "Tests passing locally but failing in CI"
```bash
Check:
1. Database versions match
2. Environment variables set correctly
3. Timezone differences
4. Parallel execution issues
```

## References

- [Integration Tests Workflow](.github/workflows/integration-tests.yml)
- [Integration Testing Docs](./CI_INTEGRATION_TESTING.md)
- [GORM Refactoring Summary](./GORM_REFACTORING_SUMMARY.md)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

