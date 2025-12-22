.PHONY: help build test lint clean docker-build docker-run install-tools install-dependencies install setup web-install web-build web-dev web-lint web-test web-test-coverage web-test-watch test-all lint-all download-binaries all

# Variables
APP_NAME=github-migrator
DOCKER_IMAGE=$(APP_NAME):latest
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")
GOBIN=$(shell go env GOPATH)/bin

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

install-dependencies: ## Install Go module dependencies
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed successfully!"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/github/git-sizer@latest
	@echo "Development tools installed successfully!"

web-install: ## Install frontend dependencies
	@echo "Installing frontend dependencies..."
	cd web && npm install
	@echo "Frontend dependencies installed successfully!"

setup: install-dependencies install-tools web-install ## Install all dependencies and tools
	@echo "Setup complete!"

install: setup ## Alias for setup

download-binaries: ## Download git-sizer binaries for embedding
	@echo "Downloading git-sizer binaries..."
	@./scripts/download-git-sizer.sh
	@echo "Binaries downloaded!"

build: download-binaries ## Build the application (backend only)
	@echo "Building backend..."
	CGO_ENABLED=1 go build -o bin/$(APP_NAME)-server cmd/server/main.go
	@echo "Build complete!"

web-build: ## Build the frontend
	@echo "Building frontend..."
	cd web && npm run build
	@echo "Frontend build complete!"

build-all: build web-build ## Build both backend and frontend
	@echo "Full build complete!"

test: ## Run tests
	@echo "Running backend tests..."
	go test -v -race -coverprofile=coverage.out ./cmd/... ./internal/...

test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out ./cmd/... ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run all integration tests (SQLite, PostgreSQL, SQL Server)
	@./scripts/run-integration-tests.sh

test-integration-sqlite: ## Run SQLite integration tests only
	@echo "Running SQLite integration tests..."
	@go test -tags=integration -v ./internal/storage -run TestIntegrationSQLite -timeout 30s

test-integration-postgres: ## Run PostgreSQL integration tests (requires Docker)
	@echo "Starting PostgreSQL..."
	@docker compose -f docker-compose.postgres.yml up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@timeout 30 bash -c 'until docker compose -f docker-compose.postgres.yml exec -T postgres pg_isready -U migrator > /dev/null 2>&1; do sleep 1; done' || true
	@docker compose -f docker-compose.postgres.yml exec -T postgres psql -U migrator -d migrator -c "CREATE DATABASE migrator_test;" 2>/dev/null || true
	@echo "Running PostgreSQL integration tests..."
	@POSTGRES_TEST_DSN="postgres://migrator:migrator_dev_password@localhost:5432/migrator_test?sslmode=disable" \
		go test -tags=integration -v ./internal/storage -run TestIntegrationPostgreSQL -timeout 30s
	@echo "Cleaning up PostgreSQL..."
	@docker compose -f docker-compose.postgres.yml down

test-integration-sqlserver: ## Run SQL Server integration tests (requires Docker)
	@echo "Starting SQL Server..."
	@docker compose -f docker-compose.sqlserver.yml up -d sqlserver
	@echo "Waiting for SQL Server to be ready (this may take a minute)..."
	@timeout 60 bash -c 'until docker compose -f docker-compose.sqlserver.yml exec -T sqlserver /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "YourStrong@Passw0rd" -Q "SELECT 1" > /dev/null 2>&1; do sleep 1; done' || true
	@docker compose -f docker-compose.sqlserver.yml exec -T sqlserver /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "YourStrong@Passw0rd" -Q "CREATE DATABASE migrator_test;" 2>/dev/null || true
	@echo "Running SQL Server integration tests..."
	@SQLSERVER_TEST_DSN="sqlserver://sa:YourStrong@Passw0rd@localhost:1433?database=migrator_test" \
		go test -tags=integration -v ./internal/storage -run TestIntegrationSQLServer -timeout 30s
	@echo "Cleaning up SQL Server..."
	@docker compose -f docker-compose.sqlserver.yml down

lint: ## Run linters
	@echo "Linting backend..."
	$(GOBIN)/golangci-lint run --config .golangci.yml ./cmd/... ./internal/...
	@echo "Running security scan..."
	$(GOBIN)/gosec -exclude=G201,G202 -exclude-dir=scripts ./...

web-lint: ## Run frontend linter
	@echo "Linting frontend..."
	cd web && npm run lint

web-test: ## Run frontend tests
	@echo "Running frontend tests..."
	cd web && npm run test:run

web-test-coverage: ## Run frontend tests with coverage
	@echo "Running frontend tests with coverage..."
	cd web && npm run test:coverage

web-test-watch: ## Run frontend tests in watch mode
	@echo "Running frontend tests in watch mode..."
	cd web && npm run test

lint-all: lint web-lint ## Run all linters

test-all: test web-test ## Run all tests (backend and frontend)

fmt: ## Format code
	@echo "Formatting Go code..."
	go fmt ./...
	gofmt -s -w $(GO_FILES)

run-server: ## Run the server locally
	go run cmd/server/main.go

web-dev: ## Run frontend dev server
	cd web && npm run dev

run-dev: ## Run both backend and frontend in dev mode (requires tmux or run in separate terminals)
	@echo "Run 'make run-server' in one terminal and 'make web-dev' in another"

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container with SQLite
	docker-compose up

docker-run-postgres: ## Run Docker containers with PostgreSQL (production-like setup)
	docker compose -f docker-compose.yml -f docker-compose.postgres.yml up

docker-run-postgres-detached: ## Run Docker containers with PostgreSQL in background
	docker compose -f docker-compose.yml -f docker-compose.postgres.yml up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-down-postgres: ## Stop Docker containers with PostgreSQL (removes volumes)
	docker compose -f docker-compose.yml -f docker-compose.postgres.yml down -v

docker-logs: ## View Docker container logs
	docker-compose logs -f

docker-logs-postgres: ## View Docker container logs (PostgreSQL setup)
	docker compose -f docker-compose.yml -f docker-compose.postgres.yml logs -f

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f $(APP_NAME)-server $(APP_NAME)-cli
	rm -rf web/dist web/node_modules

create-test-repos: ## Create test repositories in GitHub organization (requires GITHUB_TOKEN and ORG env vars)
	@if [ -z "$(ORG)" ]; then \
		echo "Error: ORG environment variable is required"; \
		echo "Usage: make create-test-repos ORG=your-org-name"; \
		exit 1; \
	fi
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required"; \
		exit 1; \
	fi
	@echo "Creating test repositories in organization: $(ORG)"
	go run scripts/create-test-repos.go -org $(ORG)

cleanup-test-repos: ## Delete all test repositories from GitHub organization (requires GITHUB_TOKEN and ORG env vars)
	@if [ -z "$(ORG)" ]; then \
		echo "Error: ORG environment variable is required"; \
		echo "Usage: make cleanup-test-repos ORG=your-org-name"; \
		exit 1; \
	fi
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required"; \
		exit 1; \
	fi
	@echo "Cleaning up test repositories in organization: $(ORG)"
	go run scripts/create-test-repos.go -org $(ORG) -cleanup

all: lint-all test-all build web-build ## Run all checks, tests, and build

.DEFAULT_GOAL := help
