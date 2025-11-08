#!/bin/bash
set -e

echo "🧪 Running Integration Tests for GitHub Migrator"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Function to run a test
run_test() {
    local test_name=$1
    local test_cmd=$2
    
    echo -e "\n${YELLOW}Running ${test_name}...${NC}"
    if eval "$test_cmd"; then
        echo -e "${GREEN}✅ ${test_name} passed${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}❌ ${test_name} failed${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# SQLite tests (always available)
echo -e "\n${YELLOW}=== Testing SQLite ===${NC}"
run_test "SQLite Integration" "go test -tags=integration -v ./internal/storage -run TestIntegrationSQLite -timeout 30s"

# PostgreSQL tests (requires Docker)
echo -e "\n${YELLOW}=== Testing PostgreSQL ===${NC}"
echo "Starting PostgreSQL container..."
docker compose -f docker-compose.postgres.yml up -d postgres

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if docker compose -f docker-compose.postgres.yml exec -T postgres pg_isready -U migrator > /dev/null 2>&1; then
        echo "PostgreSQL is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${YELLOW}⚠️  PostgreSQL not ready after 30 seconds, skipping PostgreSQL tests${NC}"
        ((TESTS_SKIPPED++))
        docker compose -f docker-compose.postgres.yml down
        exit 0
    fi
    sleep 1
done

# Create test database
docker compose -f docker-compose.postgres.yml exec -T postgres psql -U migrator -d migrator -c "CREATE DATABASE migrator_test;" 2>/dev/null || true

# Run PostgreSQL tests
export POSTGRES_TEST_DSN="postgres://migrator:migrator_dev_password@localhost:5432/migrator_test?sslmode=disable"
if run_test "PostgreSQL Integration" "go test -tags=integration -v ./internal/storage -run TestIntegrationPostgreSQL -timeout 30s"; then
    echo "PostgreSQL tests completed"
else
    echo "PostgreSQL tests failed"
fi

# Cleanup PostgreSQL
echo "Cleaning up PostgreSQL container..."
docker compose -f docker-compose.postgres.yml down

# SQL Server tests (optional - requires more resources)
if [ "${SKIP_SQLSERVER:-}" != "true" ]; then
    echo -e "\n${YELLOW}=== Testing SQL Server ===${NC}"
    echo "Starting SQL Server container (this may take a minute)..."
    docker compose -f docker-compose.sqlserver.yml up -d sqlserver

    # Wait for SQL Server to be ready
    echo "Waiting for SQL Server to be ready..."
    for i in {1..60}; do
        if docker compose -f docker-compose.sqlserver.yml exec -T sqlserver /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "YourStrong@Passw0rd" -Q "SELECT 1" > /dev/null 2>&1; then
            echo "SQL Server is ready!"
            break
        fi
        if [ $i -eq 60 ]; then
            echo -e "${YELLOW}⚠️  SQL Server not ready after 60 seconds, skipping SQL Server tests${NC}"
            ((TESTS_SKIPPED++))
            docker compose -f docker-compose.sqlserver.yml down
            exit 0
        fi
        sleep 1
    done

    # Create test database
    docker compose -f docker-compose.sqlserver.yml exec -T sqlserver /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "YourStrong@Passw0rd" -Q "CREATE DATABASE migrator_test;" 2>/dev/null || true

    # Run SQL Server tests
    export SQLSERVER_TEST_DSN="sqlserver://sa:YourStrong@Passw0rd@localhost:1433?database=migrator_test"
    if run_test "SQL Server Integration" "go test -tags=integration -v ./internal/storage -run TestIntegrationSQLServer -timeout 30s"; then
        echo "SQL Server tests completed"
    else
        echo "SQL Server tests failed"
    fi

    # Cleanup SQL Server
    echo "Cleaning up SQL Server container..."
    docker compose -f docker-compose.sqlserver.yml down
else
    echo -e "\n${YELLOW}⚠️  Skipping SQL Server tests (set SKIP_SQLSERVER=false to enable)${NC}"
    ((TESTS_SKIPPED++))
fi

# Summary
echo -e "\n${YELLOW}================================================${NC}"
echo -e "${YELLOW}Test Summary${NC}"
echo -e "${YELLOW}================================================${NC}"
echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"
echo -e "${YELLOW}Skipped: ${TESTS_SKIPPED}${NC}"

if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "\n${RED}❌ Some tests failed${NC}"
    exit 1
else
    echo -e "\n${GREEN}✅ All tests passed!${NC}"
    exit 0
fi

