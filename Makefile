.PHONY: help build test lint clean docker-build docker-run install-tools install-dependencies install setup web-install web-build web-dev web-lint

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

build: ## Build the application (backend only)
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
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linters
	@echo "Linting backend..."
	$(GOBIN)/golangci-lint run --config .golangci.yml
	@echo "Running security scan..."
	$(GOBIN)/gosec -exclude=G201,G202 ./...

web-lint: ## Run frontend linter
	@echo "Linting frontend..."
	cd web && npm run lint

lint-all: lint web-lint ## Run all linters

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

docker-run: ## Run Docker container
	docker-compose up

docker-down: ## Stop Docker containers
	docker-compose down

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

all: lint test build web-build ## Run all checks and build

.DEFAULT_GOAL := help
