package discovery

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// setupTestDir creates a temporary directory with test files
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "package_scanner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// writeTestFile creates a test file with the given content
func writeTestFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
}

func TestPackageScanner_ScanNodeJS(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create package.json with dependencies
	packageJSON := `{
		"name": "test-package",
		"dependencies": {
			"express": "^4.18.0",
			"lodash": "^4.17.21"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	writeTestFile(t, dir, "package.json", packageJSON)

	// Create package-lock.json
	writeTestFile(t, dir, "package-lock.json", `{"lockfileVersion": 3}`)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 1 npm manifest
	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "npm:package.json" {
			found = true
			if dep.DependencyType != "package" {
				t.Errorf("Expected dependency type 'package', got '%s'", dep.DependencyType)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find npm:package.json dependency")
	}
}

func TestPackageScanner_ScanGo(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create go.mod
	goMod := `module github.com/example/test

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/stretchr/testify v1.8.4
)
`
	writeTestFile(t, dir, "go.mod", goMod)
	writeTestFile(t, dir, "go.sum", "github.com/gin-gonic/gin v1.9.0 h1:...")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "go:go.mod" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find go:go.mod dependency")
	}
}

func TestPackageScanner_ScanPython(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create requirements.txt
	requirements := `flask==2.3.0
requests>=2.28.0
pytest  # dev dependency
`
	writeTestFile(t, dir, "requirements.txt", requirements)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "python:requirements.txt" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find python:requirements.txt dependency")
	}
}

func TestPackageScanner_ScanPyProject(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create pyproject.toml (Poetry style)
	pyproject := `[tool.poetry]
name = "test-project"

[tool.poetry.dependencies]
python = "^3.9"
django = "^4.2"

[build-system]
requires = ["poetry-core"]
`
	writeTestFile(t, dir, "pyproject.toml", pyproject)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "python:pyproject.toml" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find python:pyproject.toml dependency")
	}
}

func TestPackageScanner_ScanMaven(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create pom.xml
	pomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <dependencies>
        <dependency>
            <groupId>org.springframework</groupId>
            <artifactId>spring-core</artifactId>
            <version>5.3.0</version>
        </dependency>
        <dependency>
            <groupId>junit</groupId>
            <artifactId>junit</artifactId>
            <version>4.13</version>
            <scope>test</scope>
        </dependency>
    </dependencies>
</project>
`
	writeTestFile(t, dir, "pom.xml", pomXML)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "maven:pom.xml" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find maven:pom.xml dependency")
	}
}

func TestPackageScanner_ScanGradle(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create build.gradle
	buildGradle := `plugins {
    id 'java'
}

dependencies {
    implementation 'org.springframework:spring-core:5.3.0'
    testImplementation 'junit:junit:4.13'
}
`
	writeTestFile(t, dir, "build.gradle", buildGradle)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "gradle:build.gradle" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find gradle:build.gradle dependency")
	}
}

func TestPackageScanner_ScanDotNet(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create .csproj file
	csproj := `<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
    <PackageReference Include="Serilog" Version="2.12.0" />
  </ItemGroup>
</Project>
`
	writeTestFile(t, dir, "MyProject.csproj", csproj)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "nuget:MyProject.csproj" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find nuget:MyProject.csproj dependency")
	}
}

func TestPackageScanner_ScanRuby(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Gemfile
	gemfile := `source 'https://rubygems.org'

gem 'rails', '~> 7.0'
gem 'pg', '~> 1.4'
gem 'puma', '~> 6.0'
`
	writeTestFile(t, dir, "Gemfile", gemfile)
	writeTestFile(t, dir, "Gemfile.lock", "GEM\n  specs:\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "rubygems:Gemfile" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find rubygems:Gemfile dependency")
	}
}

func TestPackageScanner_ScanRust(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Cargo.toml
	cargoToml := `[package]
name = "my-project"
version = "0.1.0"

[dependencies]
serde = "1.0"
tokio = { version = "1.0", features = ["full"] }

[dev-dependencies]
criterion = "0.4"
`
	writeTestFile(t, dir, "Cargo.toml", cargoToml)
	writeTestFile(t, dir, "Cargo.lock", "[[package]]\nname = \"serde\"\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "cargo:Cargo.toml" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find cargo:Cargo.toml dependency")
	}
}

func TestPackageScanner_ScanPHP(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create composer.json
	composerJSON := `{
    "name": "example/app",
    "require": {
        "php": "^8.1",
        "laravel/framework": "^10.0"
    },
    "require-dev": {
        "phpunit/phpunit": "^10.0"
    }
}
`
	writeTestFile(t, dir, "composer.json", composerJSON)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "composer:composer.json" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find composer:composer.json dependency")
	}
}

func TestPackageScanner_ScanTerraform(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create main.tf
	mainTF := `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}
`
	writeTestFile(t, dir, "main.tf", mainTF)
	writeTestFile(t, dir, ".terraform.lock.hcl", "provider \"registry.terraform.io/hashicorp/aws\" {}")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "terraform:*.tf" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find terraform:*.tf dependency")
	}
}

func TestPackageScanner_ScanHelm(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Chart.yaml
	chartYAML := `apiVersion: v2
name: my-chart
version: 1.0.0

dependencies:
  - name: redis
    version: 17.0.0
    repository: https://charts.bitnami.com/bitnami
  - name: postgresql
    version: 12.0.0
    repository: https://charts.bitnami.com/bitnami
`
	writeTestFile(t, dir, "Chart.yaml", chartYAML)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.DependencyFullName == "helm:Chart.yaml" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find helm:Chart.yaml dependency")
	}
}

func TestPackageScanner_ScanDocker(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Dockerfile
	dockerfile := `FROM node:18-alpine AS builder
WORKDIR /app
COPY . .

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
`
	writeTestFile(t, dir, "Dockerfile", dockerfile)

	// Create docker-compose.yml
	compose := `version: '3.8'
services:
  web:
    build: .
    ports:
      - "8080:80"
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
`
	writeTestFile(t, dir, "docker-compose.yml", compose)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	foundDockerfile := false
	foundCompose := false
	for _, dep := range deps {
		if dep.DependencyFullName == "docker:Dockerfile" {
			foundDockerfile = true
		}
		if dep.DependencyFullName == "docker:docker-compose.yml" {
			foundCompose = true
		}
	}

	if !foundDockerfile {
		t.Error("Expected to find docker:Dockerfile dependency")
	}
	if !foundCompose {
		t.Error("Expected to find docker:docker-compose.yml dependency")
	}
}

func TestPackageScanner_MonorepoMultipleEcosystems(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create a monorepo structure with multiple ecosystems
	writeTestFile(t, dir, "package.json", `{"dependencies": {"react": "^18.0"}}`)
	writeTestFile(t, dir, "services/api/go.mod", `module api
go 1.21
require github.com/gin-gonic/gin v1.9.0`)
	writeTestFile(t, dir, "services/worker/requirements.txt", `celery==5.3.0`)
	writeTestFile(t, dir, "infrastructure/main.tf", `provider "aws" {}`)
	writeTestFile(t, dir, "charts/app/Chart.yaml", `apiVersion: v2
name: app
version: 1.0.0`)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	ecosystems := make(map[string]bool)
	for _, dep := range deps {
		// Extract ecosystem from dependency full name (format: "ecosystem:path")
		parts := []byte(dep.DependencyFullName)
		for i, b := range parts {
			if b == ':' {
				ecosystems[string(parts[:i])] = true
				break
			}
		}
	}

	expectedEcosystems := []string{"npm", "go", "python", "terraform", "helm"}
	for _, eco := range expectedEcosystems {
		if !ecosystems[eco] {
			t.Errorf("Expected to find ecosystem '%s' in monorepo scan", eco)
		}
	}
}

func TestPackageScanner_SkipsNodeModules(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create package.json in root
	writeTestFile(t, dir, "package.json", `{"dependencies": {"express": "^4.18"}}`)

	// Create package.json inside node_modules (should be skipped)
	writeTestFile(t, dir, "node_modules/express/package.json", `{"name": "express"}`)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should only find 1 manifest (root package.json)
	npmCount := 0
	for _, dep := range deps {
		if len(dep.DependencyFullName) >= 3 && dep.DependencyFullName[:3] == "npm" {
			npmCount++
		}
	}

	if npmCount != 1 {
		t.Errorf("Expected 1 npm manifest, found %d (node_modules should be skipped)", npmCount)
	}
}

func TestPackageScanner_EmptyDirectory(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies in empty directory, found %d", len(deps))
	}
}

func TestPackageScanner_DependencyCount(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create package.json with known number of dependencies
	packageJSON := `{
		"dependencies": {
			"express": "^4.18.0",
			"lodash": "^4.17.21",
			"axios": "^1.0.0"
		},
		"devDependencies": {
			"jest": "^29.0.0",
			"typescript": "^5.0.0"
		}
	}`
	writeTestFile(t, dir, "package.json", packageJSON)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	manifests := scanner.scanNodeJS(context.Background(), dir)

	if len(manifests) != 1 {
		t.Fatalf("Expected 1 manifest, got %d", len(manifests))
	}

	// Should count 3 deps + 2 dev deps = 5 total
	if manifests[0].DependencyCount != 5 {
		t.Errorf("Expected 5 dependencies, got %d", manifests[0].DependencyCount)
	}
}

func TestPackageScanner_GoModDependencyCount(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	goMod := `module example.com/test

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/sync v0.3.0
)

require github.com/single/dep v1.0.0
`
	writeTestFile(t, dir, "go.mod", goMod)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	manifests := scanner.scanGo(context.Background(), dir)

	if len(manifests) != 1 {
		t.Fatalf("Expected 1 manifest, got %d", len(manifests))
	}

	// Should count 4 requires (3 in block + 1 single line)
	if manifests[0].DependencyCount != 4 {
		t.Errorf("Expected 4 dependencies, got %d", manifests[0].DependencyCount)
	}
}
