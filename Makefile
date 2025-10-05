.PHONY: help build test lint clean docker-build docker-run install-tools install-dependencies install setup

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

setup: install-dependencies install-tools ## Install all dependencies and tools
	@echo "Setup complete!"

install: setup ## Alias for setup

build: ## Build the application
	@echo "Building backend..."
	CGO_ENABLED=1 go build -o bin/$(APP_NAME)-server cmd/server/main.go
	@echo "Build complete!"

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
	$(GOBIN)/gosec ./...

fmt: ## Format code
	@echo "Formatting Go code..."
	go fmt ./...
	gofmt -s -w $(GO_FILES)

run-server: ## Run the server locally
	go run cmd/server/main.go

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

all: lint test build ## Run all checks and build

.DEFAULT_GOAL := help
