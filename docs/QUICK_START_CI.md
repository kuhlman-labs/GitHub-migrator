# Quick Start: CI Integration Testing

## TL;DR

Your CI pipeline now automatically tests your database layer against **SQLite** and **PostgreSQL** on every PR. **SQL Server** tests run daily or manually.

## What Just Happened?

✅ **New GitHub Workflow**: `.github/workflows/integration-tests.yml`  
✅ **Comprehensive Docs**: `docs/CI_INTEGRATION_TESTING.md`  
✅ **Ready to Use**: Works immediately on next PR

## Running Tests

### Locally (Before Pushing)
```bash
# Quick check - SQLite only (~30s)
make test-integration-sqlite

# Production validation - PostgreSQL (~2min, requires Docker)
make test-integration-postgres

# Complete check - All databases (~8min, requires Docker)
make test-integration
```

### In CI (Automatic)
Tests run automatically on:
- ✅ Every Pull Request → SQLite + PostgreSQL
- ✅ Push to main/develop → SQLite + PostgreSQL  
- ✅ Daily at 2 AM UTC → All 3 databases
- ✅ Manual trigger → Your choice

### Manual Trigger (Optional SQL Server)
```bash
# Via GitHub UI
Actions → Integration Tests → Run workflow → ☑️ Run SQL Server tests

# Via CLI
gh workflow run integration-tests.yml -f test-sqlserver=true
```

## What Gets Tested?

Each database runs **7 comprehensive tests**:
1. ✅ Migrations (all 19 applied correctly)
2. ✅ Repository CRUD (create, read, update, delete)
3. ✅ Batch Operations (create, update, delete)
4. ✅ Migration History (tracking)
5. ✅ Repository Dependencies (relationships)
6. ✅ List with Filters (queries with scopes)
7. ✅ Analytics Queries (complex SQL)

## Viewing Results

### In Pull Requests
Check status shows: `Integration Tests - SQLite` ✅ and `Integration Tests - PostgreSQL` ✅

### In GitHub Actions
1. Go to **Actions** tab
2. Click **Integration Tests** workflow
3. See detailed results for each database

### Summary View
Every run creates a summary showing:
```
🧪 Database Integration Test Results

| Database   | Status    | Notes                    |
|------------|-----------|--------------------------|
| SQLite     | ✅ Passed | Fast, always tested      |
| PostgreSQL | ✅ Passed | Production database      |
| SQL Server | ⏭️ Skipped | Only on schedule/manual  |
```

## Estimated Times

| Database   | Time  | When              |
|------------|-------|-------------------|
| SQLite     | ~30s  | Every PR          |
| PostgreSQL | ~2min | Every PR          |
| SQL Server | ~5min | Daily/Manual only |

**Total PR Check Time**: ~3 minutes (runs in parallel with other CI)

## What If Tests Fail?

### 1. Check the Logs
```bash
gh run view --log-failed
```

### 2. Reproduce Locally
```bash
# Use same test
make test-integration-sqlite

# Or specific database
POSTGRES_TEST_DSN="postgres://..." go test -tags=integration -v ./internal/storage -run TestIntegrationPostgreSQL
```

### 3. Common Issues

#### SQLite: "FOREIGN KEY constraint failed"
- **Cause**: Delete order wrong
- **Fix**: Delete children before parents

#### PostgreSQL: "Connection timeout"
- **Cause**: Service not ready
- **Fix**: Already handled in workflow (automatic retry)

#### SQL Server: "Startup timeout"
- **Cause**: Slow container startup
- **Fix**: Already handled (60s wait time)

## Cost & Resources

### GitHub Actions Minutes
- **Free Tier**: 2,000 min/month (private repos)
- **Per PR**: ~3 minutes
- **Monthly Usage**: ~390 minutes (50 PRs + daily runs)
- **Remaining**: ~1,600 minutes buffer 🎉

### Zero Cost Strategy ✅
- SQLite: No external services
- PostgreSQL: GitHub service container (free)
- SQL Server: Only on schedule (minimized usage)

## Branch Protection

### Recommended Setup
Add these as **required status checks** for `main` branch:

In GitHub → Settings → Branches → Add rule:
```
☑️ Require status checks to pass before merging
  ☑️ Backend CI (Go)
  ☑️ Frontend CI (React/TypeScript)  
  ☑️ Integration Tests - SQLite
  ☑️ Integration Tests - PostgreSQL
  ☐ Integration Tests - SQL Server (optional)
```

## Developer Workflow

### Before Creating PR
```bash
# 1. Make your changes to storage layer
vim internal/storage/repository.go

# 2. Run quick local test
make test-integration-sqlite

# 3. If testing PostgreSQL changes, test locally
make test-integration-postgres

# 4. Push and create PR
git push origin feature-branch
```

### After Creating PR
1. CI automatically runs tests
2. Check results in PR status
3. Fix any failures
4. Push fixes → Tests rerun automatically

### Before Merging
1. Ensure all checks pass ✅
2. Review test summary in Actions
3. Merge with confidence 🚀

## Monitoring

### Key Metrics Dashboard
View in Actions → Integration Tests:
- ✅ Pass rate by database
- ⏱️ Execution time trends  
- 📊 Test coverage
- 🔍 Flaky test detection

### Weekly Review
Check for:
- Increasing test duration (optimization needed)
- Failing tests (fix or skip temporarily)
- Resource usage (stay within free tier)

## Advanced Usage

### Test Specific Database Only
```bash
# In CI - edit workflow file temporarily
# Or use manual trigger with options
```

### Add New Database Test
Edit `internal/storage/integration_test.go`:
```go
func TestIntegrationMySQL(t *testing.T) {
    // Your test here
    runIntegrationTests(t, cfg, "MySQL")
}
```

### Skip Tests on PR
Add to PR description:
```
[skip integration]
```
*(Note: You'll need to add this check to workflow)*

## Troubleshooting

### "Workflow not found"
- **Issue**: New workflow not visible
- **Fix**: Push to `main` or `develop` branch first

### "Tests skipped"
- **Check**: Path filters - only runs on storage changes
- **Fix**: Make change to `internal/storage/**` or trigger manually

### "Can't connect to database"
- **Local**: Ensure Docker is running
- **CI**: Check workflow logs for service container status

## Next Steps

1. ✅ **Done**: Integration tests set up
2. ⏳ **Review**: Test results on next PR
3. ⏳ **Configure**: Branch protection rules
4. ⏳ **Monitor**: Test execution for first week
5. ⏳ **Optimize**: Adjust if needed

## Documentation

- 📖 [Detailed CI Strategy](./CI_INTEGRATION_TESTING.md)
- 📊 [Pipeline Diagram](./CI_PIPELINE_DIAGRAM.md)
- 🔧 [GORM Refactoring Summary](./GORM_REFACTORING_SUMMARY.md)
- 📝 [Workflow README](../.github/workflows/README.md)

## Quick Commands

```bash
# Local testing
make test-integration-sqlite          # Fast check
make test-integration-postgres        # Production check
make test-integration                 # Full check

# CI management
gh workflow list                       # See all workflows
gh run list --workflow=integration    # Recent runs
gh run watch                          # Watch current run
gh workflow run integration-tests.yml  # Manual trigger

# Debugging
gh run view --log-failed              # View failed logs
make test-integration-sqlite -v       # Verbose local test
```

## Support

- 🐛 **Bug?** Open issue with workflow run ID
- 📚 **Question?** Check docs above
- 💡 **Suggestion?** Open discussion or PR

---

**Ready to go!** 🚀 Your next PR will automatically run integration tests.

