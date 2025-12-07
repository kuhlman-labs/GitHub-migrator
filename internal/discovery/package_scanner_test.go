package discovery

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

func TestPackageScanner_ScanGoModWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create go.mod with GitHub dependencies
	goMod := `module github.com/example/test

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

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find GitHub dependencies (gin-gonic/gin, stretchr/testify, single/dep)
	// golang.org/x/sync is not a github.com dependency so it should be excluded
	expectedDeps := map[string]bool{
		"gin-gonic/gin":    false,
		"stretchr/testify": false,
		"single/dep":       false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
			if dep.DependencyType != "package" {
				t.Errorf("Expected dependency type 'package', got '%s'", dep.DependencyType)
			}
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find GitHub dependency '%s'", name)
		}
	}
}

func TestPackageScanner_ScanNpmWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create package.json with GitHub dependencies
	packageJSON := `{
		"name": "test-package",
		"dependencies": {
			"express": "^4.18.0",
			"my-lib": "github:myorg/mylib",
			"another-lib": "git+https://github.com/owner/repo.git"
		},
		"devDependencies": {
			"jest": "^29.0.0",
			"custom-tool": "someuser/sometool"
		}
	}`
	writeTestFile(t, dir, "package.json", packageJSON)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find GitHub dependencies
	expectedDeps := map[string]bool{
		"myorg/mylib":       false,
		"owner/repo":        false,
		"someuser/sometool": false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find GitHub dependency '%s'", name)
		}
	}
}

func TestPackageScanner_ScanPythonWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create requirements.txt with GitHub dependencies
	requirements := `flask==2.3.0
git+https://github.com/pallets/click.git@8.0.0
-e git+https://github.com/pytest-dev/pytest.git#egg=pytest
requests>=2.28.0
git+ssh://git@github.com/private/repo.git
`
	writeTestFile(t, dir, "requirements.txt", requirements)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find GitHub dependencies
	expectedDeps := map[string]bool{
		"pallets/click":     false,
		"pytest-dev/pytest": false,
		"private/repo":      false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find GitHub dependency '%s'", name)
		}
	}
}

func TestPackageScanner_RepoNamesWithDots(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Test that repo names with dots are correctly extracted (not truncated at first dot)
	requirements := `flask==2.3.0
git+https://github.com/owner/my-lib.backup.git@v1.0
git+https://github.com/owner/package.core@main
git+ssh://git@github.com/owner/lib.utils.js.git#egg=lib
`
	writeTestFile(t, dir, "requirements.txt", requirements)

	packageJSON := `{
		"dependencies": {
			"dotted-lib": "github:owner/my.dotted.lib",
			"backup-lib": "git+https://github.com/owner/backup.lib.git"
		}
	}`
	writeTestFile(t, dir, "package.json", packageJSON)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// These repo names should NOT be truncated at dots
	expectedDeps := map[string]bool{
		"owner/my-lib.backup": false,
		"owner/package.core":  false,
		"owner/lib.utils.js":  false,
		"owner/my.dotted.lib": false,
		"owner/backup.lib":    false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find repo with dots in name '%s' (repo name was likely truncated)", name)
			// Log what was actually found to help debug
			for _, dep := range deps {
				t.Logf("Found: %s", dep.DependencyFullName)
			}
		}
	}
}

func TestPackageScanner_NoGitHubDepsReturnsEmpty(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create package.json with only npm registry dependencies (no GitHub refs)
	packageJSON := `{
		"dependencies": {
			"express": "^4.18.0",
			"lodash": "^4.17.21"
		}
	}`
	writeTestFile(t, dir, "package.json", packageJSON)

	// Create requirements.txt with only PyPI packages (no GitHub refs)
	requirements := `flask==2.3.0
requests>=2.28.0
`
	writeTestFile(t, dir, "requirements.txt", requirements)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// No GitHub dependencies, so should return empty
	if len(deps) != 0 {
		t.Errorf("Expected 0 GitHub dependencies, found %d", len(deps))
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

func TestPackageScanner_MonorepoWithMultipleGoMods(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create multiple go.mod files with GitHub dependencies
	writeTestFile(t, dir, "go.mod", `module main
go 1.21
require github.com/gin-gonic/gin v1.9.0`)

	writeTestFile(t, dir, "services/api/go.mod", `module api
go 1.21
require github.com/gorilla/mux v1.8.0`)

	writeTestFile(t, dir, "services/worker/go.mod", `module worker
go 1.21
require github.com/gin-gonic/gin v1.9.0`)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should deduplicate: gin-gonic/gin appears twice but should only be in result once
	foundGin := false
	foundMux := false
	for _, dep := range deps {
		if dep.DependencyFullName == "gin-gonic/gin" {
			if foundGin {
				t.Error("gin-gonic/gin should not appear twice (deduplication failed)")
			}
			foundGin = true
		}
		if dep.DependencyFullName == "gorilla/mux" {
			foundMux = true
		}
	}

	if !foundGin {
		t.Error("Expected to find gin-gonic/gin")
	}
	if !foundMux {
		t.Error("Expected to find gorilla/mux")
	}
}

func TestPackageScanner_SkipsNodeModules(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create go.mod in root with a GitHub dependency
	writeTestFile(t, dir, "go.mod", `module test
go 1.21
require github.com/stretchr/testify v1.8.0`)

	// Create go.mod inside node_modules (should be skipped)
	writeTestFile(t, dir, "node_modules/some-package/go.mod", `module nodemod
go 1.21
require github.com/should/skip v1.0.0`)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should only find testify, not the one from node_modules
	foundTestify := false
	foundSkip := false
	for _, dep := range deps {
		if dep.DependencyFullName == "stretchr/testify" {
			foundTestify = true
		}
		if dep.DependencyFullName == "should/skip" {
			foundSkip = true
		}
	}

	if !foundTestify {
		t.Error("Expected to find stretchr/testify")
	}
	if foundSkip {
		t.Error("Should NOT find should/skip (node_modules should be skipped)")
	}
}

func TestPackageScanner_ExtractGitHubFromNpmVersion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	tests := []struct {
		version       string
		expectedOwner string
		expectedRepo  string
		expectedHost  string
	}{
		{"github:owner/repo", "owner", "repo", "github.com"},
		{"github:owner/repo#v1.0.0", "owner", "repo", "github.com"},
		{"git+https://github.com/owner/repo.git", "owner", "repo", "github.com"},
		{"git+https://github.com/owner/repo.git#branch", "owner", "repo", "github.com"},
		{"https://github.com/owner/repo", "owner", "repo", "github.com"},
		{"owner/repo", "owner", "repo", "github.com"},
		{"owner/repo#branch", "owner", "repo", "github.com"},
		// Repo names with dots should be fully extracted
		{"github:owner/my-lib.backup", "owner", "my-lib.backup", "github.com"},
		{"git+https://github.com/owner/lib.js.git", "owner", "lib.js", "github.com"},
		{"https://github.com/owner/package.core", "owner", "package.core", "github.com"},
		{"owner/my.dotted.repo#v1.0", "owner", "my.dotted.repo", "github.com"},
		// These should NOT match
		{"^1.0.0", "", "", ""},
		{"~2.0.0", "", "", ""},
		{"1.0.0", "", "", ""},
		{"@scope/package", "", "", ""},
	}

	for _, tc := range tests {
		owner, repo, host, _ := scanner.extractGitHubFromNpmVersion(tc.version)
		if owner != tc.expectedOwner || repo != tc.expectedRepo || host != tc.expectedHost {
			t.Errorf("extractGitHubFromNpmVersion(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.version, owner, repo, host, tc.expectedOwner, tc.expectedRepo, tc.expectedHost)
		}
	}
}

func TestPackageScanner_ParseGoMod(t *testing.T) {
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

	deps := scanner.parseGoMod(filepath.Join(dir, "go.mod"), "go.mod")

	// Should find 3 GitHub deps (golang.org is excluded)
	if len(deps) != 3 {
		t.Errorf("Expected 3 GitHub dependencies, got %d", len(deps))
	}

	expectedDeps := map[string]bool{
		"github.com/gin-gonic/gin":    false,
		"github.com/stretchr/testify": false,
		"github.com/single/dep":       false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.Name]; ok {
			expectedDeps[dep.Name] = true
			if !dep.IsGitHubRepo {
				t.Errorf("Dependency %s should be marked as GitHub repo", dep.Name)
			}
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find dependency '%s'", name)
		}
	}
}

func TestPackageScanner_DependencyMetadata(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	goMod := `module test
go 1.21
require github.com/gin-gonic/gin v1.9.0`
	writeTestFile(t, dir, "go.mod", goMod)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	if dep.DependencyFullName != "gin-gonic/gin" {
		t.Errorf("Expected DependencyFullName 'gin-gonic/gin', got '%s'", dep.DependencyFullName)
	}
	if dep.DependencyType != "package" {
		t.Errorf("Expected DependencyType 'package', got '%s'", dep.DependencyType)
	}
	if dep.DependencyURL != "https://github.com/gin-gonic/gin" {
		t.Errorf("Expected DependencyURL 'https://github.com/gin-gonic/gin', got '%s'", dep.DependencyURL)
	}
	if dep.Metadata == nil {
		t.Error("Expected Metadata to be set")
	}
}

func TestPackageScanner_ScanRubyGemfileWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Gemfile with GitHub dependencies
	gemfile := `source 'https://rubygems.org'

gem 'rails', '~> 7.0'
gem 'pg', '~> 1.4'
gem 'devise', github: 'heartcombo/devise'
gem 'custom-gem', github: 'myorg/custom-gem', branch: 'main'
gem 'private-lib', git: 'https://github.com/private/lib.git'
gem 'ssh-lib', git: 'git@github.com:another/repo.git'
`
	writeTestFile(t, dir, "Gemfile", gemfile)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find GitHub dependencies
	expectedDeps := map[string]bool{
		"heartcombo/devise": false,
		"myorg/custom-gem":  false,
		"private/lib":       false,
		"another/repo":      false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
			if dep.DependencyType != "package" {
				t.Errorf("Expected dependency type 'package', got '%s'", dep.DependencyType)
			}
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find GitHub dependency '%s'", name)
		}
	}
}

func TestPackageScanner_LocalDependenciesWithSourceURL(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create go.mod with both public GitHub and enterprise dependencies
	goMod := `module github.example.com/myorg/myapp

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.example.com/internal/shared-lib v1.0.0
	github.example.com/internal/utils v2.0.0
)
`
	writeTestFile(t, dir, "go.mod", goMod)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://github.example.com")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 3 dependencies
	if len(deps) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(deps))
	}

	// Check that local dependencies are marked correctly
	localCount := 0
	externalCount := 0
	for _, dep := range deps {
		if dep.IsLocal {
			localCount++
			// Local deps should have the enterprise URL
			if !strings.Contains(dep.DependencyURL, "github.example.com") {
				t.Errorf("Local dependency %s should have enterprise URL, got %s", dep.DependencyFullName, dep.DependencyURL)
			}
		} else {
			externalCount++
			// External deps should have github.com URL
			if !strings.Contains(dep.DependencyURL, "github.com") {
				t.Errorf("External dependency %s should have github.com URL, got %s", dep.DependencyFullName, dep.DependencyURL)
			}
		}
	}

	if localCount != 2 {
		t.Errorf("Expected 2 local dependencies, got %d", localCount)
	}
	if externalCount != 1 {
		t.Errorf("Expected 1 external dependency, got %d", externalCount)
	}
}

func TestPackageScanner_RubyGemfileLocalDependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Gemfile with local enterprise dependencies
	gemfile := `source 'https://rubygems.org'

gem 'rails'
gem 'external-gem', github: 'external/gem'
gem 'internal-gem', git: 'https://github.example.com/internal/gem.git'
gem 'another-internal', git: 'git@github.example.com:internal/another.git'
`
	writeTestFile(t, dir, "Gemfile", gemfile)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://github.example.com")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 3 GitHub dependencies
	if len(deps) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(deps))
	}

	// Check local vs external
	localDeps := make(map[string]bool)
	externalDeps := make(map[string]bool)
	for _, dep := range deps {
		if dep.IsLocal {
			localDeps[dep.DependencyFullName] = true
		} else {
			externalDeps[dep.DependencyFullName] = true
		}
	}

	// external/gem should be external (from github.com)
	if !externalDeps["external/gem"] {
		t.Error("external/gem should be marked as external")
	}

	// internal gems should be local
	if !localDeps["internal/gem"] {
		t.Error("internal/gem should be marked as local")
	}
	if !localDeps["internal/another"] {
		t.Error("internal/another should be marked as local")
	}
}

func TestPackageScanner_ScanTerraformWithGitHubModules(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create main.tf with various module sources
	mainTF := `terraform {
  required_version = ">= 1.0"
}

# Registry module - should be ignored
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}

# GitHub.com direct reference
module "lambda" {
  source = "github.com/terraform-aws-modules/terraform-aws-lambda"
}

# GitHub.com with subdir
module "s3" {
  source = "github.com/terraform-aws-modules/terraform-aws-s3-bucket//modules/object"
}

# Git URL with HTTPS
module "eks" {
  source = "git::https://github.com/cloudposse/terraform-aws-eks-cluster.git"
}

# Git URL with ref
module "rds" {
  source = "git::https://github.com/cloudposse/terraform-aws-rds.git?ref=v1.0.0"
}

# Local module - should be ignored
module "local" {
  source = "./modules/my-module"
}
`
	writeTestFile(t, dir, "main.tf", mainTF)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find GitHub dependencies
	expectedDeps := map[string]bool{
		"terraform-aws-modules/terraform-aws-lambda":    false,
		"terraform-aws-modules/terraform-aws-s3-bucket": false,
		"cloudposse/terraform-aws-eks-cluster":          false,
		"cloudposse/terraform-aws-rds":                  false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
			if dep.DependencyType != "package" {
				t.Errorf("Expected dependency type 'package', got '%s'", dep.DependencyType)
			}
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find Terraform module '%s'", name)
		}
	}
}

func TestPackageScanner_TerraformLocalDependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create main.tf with both public and enterprise modules
	mainTF := `terraform {
  required_version = ">= 1.0"
}

# Public GitHub module
module "vpc" {
  source = "github.com/terraform-aws-modules/terraform-aws-vpc"
}

# Enterprise GitHub module (local)
module "internal-module" {
  source = "github.example.com/platform/terraform-network"
}

# Enterprise via git URL (local)
module "shared-infra" {
  source = "git::https://github.example.com/platform/shared-infrastructure.git"
}

# Enterprise via SSH (local)
module "secrets" {
  source = "git@github.example.com:security/terraform-secrets.git"
}
`
	writeTestFile(t, dir, "infra/main.tf", mainTF)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://github.example.com")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 4 dependencies
	if len(deps) != 4 {
		t.Errorf("Expected 4 dependencies, got %d", len(deps))
	}

	// Check local vs external
	localDeps := make(map[string]bool)
	externalDeps := make(map[string]bool)
	for _, dep := range deps {
		if dep.IsLocal {
			localDeps[dep.DependencyFullName] = true
		} else {
			externalDeps[dep.DependencyFullName] = true
		}
	}

	// Public module should be external
	if !externalDeps["terraform-aws-modules/terraform-aws-vpc"] {
		t.Error("terraform-aws-modules/terraform-aws-vpc should be marked as external")
	}

	// Enterprise modules should be local
	if !localDeps["platform/terraform-network"] {
		t.Error("platform/terraform-network should be marked as local")
	}
	if !localDeps["platform/shared-infrastructure"] {
		t.Error("platform/shared-infrastructure should be marked as local")
	}
	if !localDeps["security/terraform-secrets"] {
		t.Error("security/terraform-secrets should be marked as local")
	}
}

func TestPackageScanner_TerraformRegistryModulesIgnored(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create main.tf with only registry modules
	mainTF := `terraform {
  required_version = ">= 1.0"
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "19.0.0"
}

module "s3" {
  source = "hashicorp/consul/aws"
}
`
	writeTestFile(t, dir, "main.tf", mainTF)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Registry modules should be ignored, so no dependencies
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies (registry modules should be ignored), got %d", len(deps))
		for _, dep := range deps {
			t.Logf("Found: %s", dep.DependencyFullName)
		}
	}
}

func TestPackageScanner_ScanRustCargoWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Cargo.toml with GitHub dependencies
	cargoToml := `[package]
name = "my-project"
version = "0.1.0"

[dependencies]
serde = "1.0"
my-lib = { git = "https://github.com/myorg/my-lib" }
another = { git = "https://github.com/owner/repo.git", branch = "main" }

[dev-dependencies]
test-utils = { git = "https://github.com/test/utils" }
`
	writeTestFile(t, dir, "Cargo.toml", cargoToml)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	expectedDeps := map[string]bool{
		"myorg/my-lib": false,
		"owner/repo":   false,
		"test/utils":   false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find Rust dependency '%s'", name)
		}
	}
}

func TestPackageScanner_RustCargoLocalDependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	cargoToml := `[package]
name = "my-project"

[dependencies]
external = { git = "https://github.com/external/lib" }
internal = { git = "https://github.example.com/internal/lib" }
`
	writeTestFile(t, dir, "Cargo.toml", cargoToml)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://github.example.com")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	localCount := 0
	for _, dep := range deps {
		if dep.IsLocal {
			localCount++
		}
	}
	if localCount != 1 {
		t.Errorf("Expected 1 local dependency, got %d", localCount)
	}
}

func TestPackageScanner_ScanHelmChartWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Chart.yaml with GitHub repository dependencies
	chartYaml := `apiVersion: v2
name: my-chart
version: 1.0.0

dependencies:
  - name: redis
    version: "17.0.0"
    repository: "https://charts.bitnami.com/bitnami"
  - name: custom-chart
    repository: "https://github.com/myorg/helm-charts"
  - name: internal-chart
    repository: "git+https://github.com/internal/charts"
`
	writeTestFile(t, dir, "Chart.yaml", chartYaml)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	expectedDeps := map[string]bool{
		"myorg/helm-charts": false,
		"internal/charts":   false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find Helm dependency '%s'", name)
		}
	}
}

func TestPackageScanner_ScanSwiftPackageWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create Package.swift with GitHub dependencies
	packageSwift := `// swift-tools-version:5.5
import PackageDescription

let package = Package(
    name: "MyApp",
    dependencies: [
        .package(url: "https://github.com/Alamofire/Alamofire", from: "5.0.0"),
        .package(url: "https://github.com/apple/swift-argument-parser.git", .upToNextMajor(from: "1.0.0")),
        .package(url: "https://github.com/vapor/vapor", .branch("main")),
    ],
    targets: [
        .target(name: "MyApp", dependencies: ["Alamofire", "Vapor"]),
    ]
)
`
	writeTestFile(t, dir, "Package.swift", packageSwift)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	expectedDeps := map[string]bool{
		"Alamofire/Alamofire":         false,
		"apple/swift-argument-parser": false,
		"vapor/vapor":                 false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find Swift dependency '%s'", name)
		}
	}
}

func TestPackageScanner_SwiftLocalDependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	packageSwift := `// swift-tools-version:5.5
import PackageDescription

let package = Package(
    name: "MyApp",
    dependencies: [
        .package(url: "https://github.com/public/lib", from: "1.0.0"),
        .package(url: "https://github.example.com/internal/sdk.git", from: "2.0.0"),
    ]
)
`
	writeTestFile(t, dir, "Package.swift", packageSwift)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://github.example.com")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	localCount := 0
	for _, dep := range deps {
		if dep.IsLocal {
			localCount++
			if dep.DependencyFullName != "internal/sdk" {
				t.Errorf("Expected internal/sdk to be local, got %s", dep.DependencyFullName)
			}
		}
	}
	if localCount != 1 {
		t.Errorf("Expected 1 local dependency, got %d", localCount)
	}
}

func TestPackageScanner_ScanElixirMixWithGitHubDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create mix.exs with GitHub dependencies
	mixExs := `defmodule MyApp.MixProject do
  use Mix.Project

  def project do
    [
      app: :my_app,
      version: "0.1.0",
      deps: deps()
    ]
  end

  defp deps do
    [
      {:phoenix, "~> 1.7"},
      {:ecto, "~> 3.10"},
      {:custom_lib, github: "myorg/custom-lib"},
      {:forked_dep, github: "myorg/forked-dep", branch: "fix"},
      {:private_lib, github: "internal/private-lib"},
    ]
  end
end
`
	writeTestFile(t, dir, "mix.exs", mixExs)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	expectedDeps := map[string]bool{
		"myorg/custom-lib":     false,
		"myorg/forked-dep":     false,
		"internal/private-lib": false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find Elixir dependency '%s'", name)
		}
	}
}

func TestPackageScanner_ScanGradleJitPackDeps(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create build.gradle with JitPack dependencies
	buildGradle := `plugins {
    id 'java'
}

repositories {
    mavenCentral()
    maven { url 'https://jitpack.io' }
}

dependencies {
    implementation 'org.springframework:spring-core:5.3.0'
    implementation 'com.github.User:Repo:v1.0'
    implementation 'com.github.AnotherUser:AnotherRepo:Tag'
    testImplementation 'com.github.TestOrg:TestLib:1.0.0'
}
`
	writeTestFile(t, dir, "build.gradle", buildGradle)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	expectedDeps := map[string]bool{
		"User/Repo":               false,
		"AnotherUser/AnotherRepo": false,
		"TestOrg/TestLib":         false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find Gradle JitPack dependency '%s'", name)
		}
	}
}

func TestPackageScanner_ScanGradleKotlinDSL(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create build.gradle.kts (Kotlin DSL)
	buildGradleKts := `plugins {
    kotlin("jvm") version "1.9.0"
}

repositories {
    mavenCentral()
    maven { url = uri("https://jitpack.io") }
}

dependencies {
    implementation("com.github.Owner:Project:v2.0")
    testImplementation("com.github.TestOwner:TestProject:1.0")
}
`
	writeTestFile(t, dir, "build.gradle.kts", buildGradleKts)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	expectedDeps := map[string]bool{
		"Owner/Project":         false,
		"TestOwner/TestProject": false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find Gradle Kotlin JitPack dependency '%s'", name)
		}
	}
}

func TestPackageScanner_MultipleEcosystemsInMonorepo(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	// Create a monorepo with multiple ecosystems
	writeTestFile(t, dir, "go.mod", `module test
go 1.21
require github.com/gin-gonic/gin v1.9.0`)

	writeTestFile(t, dir, "frontend/package.json", `{
		"dependencies": {
			"my-lib": "github:myorg/mylib"
		}
	}`)

	writeTestFile(t, dir, "infra/main.tf", `module "vpc" {
  source = "github.com/terraform-aws-modules/terraform-aws-vpc"
}`)

	writeTestFile(t, dir, "backend-rust/Cargo.toml", `[dependencies]
my-crate = { git = "https://github.com/owner/crate" }`)

	writeTestFile(t, dir, "ios/Package.swift", `import PackageDescription
let package = Package(
    dependencies: [
        .package(url: "https://github.com/Alamofire/Alamofire", from: "5.0.0"),
    ]
)`)

	writeTestFile(t, dir, "android/app/build.gradle", `dependencies {
    implementation 'com.github.User:AndroidLib:v1.0'
}`)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger)

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find dependencies from all ecosystems
	expectedDeps := map[string]bool{
		"gin-gonic/gin": false,
		"myorg/mylib":   false,
		"terraform-aws-modules/terraform-aws-vpc": false,
		"owner/crate":         false,
		"Alamofire/Alamofire": false,
		"User/AndroidLib":     false,
	}

	for _, dep := range deps {
		if _, ok := expectedDeps[dep.DependencyFullName]; ok {
			expectedDeps[dep.DependencyFullName] = true
		}
	}

	for name, found := range expectedDeps {
		if !found {
			t.Errorf("Expected to find dependency '%s' in monorepo", name)
		}
	}
}

// Azure DevOps Dependency Detection Tests

func TestPackageScanner_ADOSourceURLParsing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name          string
		sourceURL     string
		expectedHost  string
		expectedOrg   string
		expectedIsADO bool
	}{
		{
			name:          "dev.azure.com URL",
			sourceURL:     "https://dev.azure.com/myorg/myproject/_git/myrepo",
			expectedHost:  "dev.azure.com",
			expectedOrg:   "myorg",
			expectedIsADO: true,
		},
		{
			name:          "visualstudio.com URL",
			sourceURL:     "https://myorg.visualstudio.com/myproject/_git/myrepo",
			expectedHost:  "myorg.visualstudio.com",
			expectedOrg:   "myorg",
			expectedIsADO: true,
		},
		{
			name:          "GitHub URL",
			sourceURL:     "https://github.com/owner/repo",
			expectedHost:  "github.com",
			expectedOrg:   "",
			expectedIsADO: false,
		},
		{
			name:          "GitHub Enterprise URL",
			sourceURL:     "https://github.example.com/owner/repo",
			expectedHost:  "github.example.com",
			expectedOrg:   "",
			expectedIsADO: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scanner := NewPackageScanner(logger).WithSourceURL(tc.sourceURL)
			if scanner.sourceHost != tc.expectedHost {
				t.Errorf("Expected sourceHost %q, got %q", tc.expectedHost, scanner.sourceHost)
			}
			if scanner.sourceOrg != tc.expectedOrg {
				t.Errorf("Expected sourceOrg %q, got %q", tc.expectedOrg, scanner.sourceOrg)
			}
			if scanner.isADOSource != tc.expectedIsADO {
				t.Errorf("Expected isADOSource %v, got %v", tc.expectedIsADO, scanner.isADOSource)
			}
		})
	}
}

func TestPackageScanner_ExtractADOReference(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://dev.azure.com/myorg/myproject/_git/myrepo")

	tests := []struct {
		name            string
		gitURL          string
		expectedOrg     string
		expectedProject string
		expectedRepo    string
		expectedHost    string
		expectedIsLocal bool
	}{
		{
			name:            "dev.azure.com HTTPS URL",
			gitURL:          "https://dev.azure.com/myorg/myproject/_git/myrepo",
			expectedOrg:     "myorg",
			expectedProject: "myproject",
			expectedRepo:    "myrepo",
			expectedHost:    "dev.azure.com",
			expectedIsLocal: true,
		},
		{
			name:            "dev.azure.com HTTPS URL - different org",
			gitURL:          "https://dev.azure.com/otherorg/proj/_git/repo",
			expectedOrg:     "otherorg",
			expectedProject: "proj",
			expectedRepo:    "repo",
			expectedHost:    "dev.azure.com",
			expectedIsLocal: false,
		},
		{
			name:            "visualstudio.com URL",
			gitURL:          "https://myorg.visualstudio.com/myproject/_git/myrepo",
			expectedOrg:     "myorg",
			expectedProject: "myproject",
			expectedRepo:    "myrepo",
			expectedHost:    "myorg.visualstudio.com",
			expectedIsLocal: true,
		},
		{
			name:            "SSH URL",
			gitURL:          "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo",
			expectedOrg:     "myorg",
			expectedProject: "myproject",
			expectedRepo:    "myrepo",
			expectedHost:    "ssh.dev.azure.com",
			expectedIsLocal: true,
		},
		{
			name:            "URL with .git suffix",
			gitURL:          "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			expectedOrg:     "myorg",
			expectedProject: "myproject",
			expectedRepo:    "myrepo",
			expectedHost:    "dev.azure.com",
			expectedIsLocal: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			org, project, repo, host, isLocal := scanner.extractADOReference(tc.gitURL)
			if org != tc.expectedOrg {
				t.Errorf("Expected org %q, got %q", tc.expectedOrg, org)
			}
			if project != tc.expectedProject {
				t.Errorf("Expected project %q, got %q", tc.expectedProject, project)
			}
			if repo != tc.expectedRepo {
				t.Errorf("Expected repo %q, got %q", tc.expectedRepo, repo)
			}
			if host != tc.expectedHost {
				t.Errorf("Expected host %q, got %q", tc.expectedHost, host)
			}
			if isLocal != tc.expectedIsLocal {
				t.Errorf("Expected isLocal %v, got %v", tc.expectedIsLocal, isLocal)
			}
		})
	}
}

func TestPackageScanner_RustCargoADODependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	cargoToml := `[package]
name = "my-project"

[dependencies]
external = { git = "https://github.com/external/lib" }
internal = { git = "https://dev.azure.com/myorg/myproject/_git/internal-lib" }
internal-ssh = { git = "git@ssh.dev.azure.com:v3/myorg/myproject/another-lib" }
`
	writeTestFile(t, dir, "Cargo.toml", cargoToml)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://dev.azure.com/myorg/myproject/_git/repo")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 3 dependencies
	if len(deps) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(deps))
		for _, dep := range deps {
			t.Logf("Found: %s (local: %v)", dep.DependencyFullName, dep.IsLocal)
		}
	}

	// Count local vs external
	localCount := 0
	for _, dep := range deps {
		if dep.IsLocal {
			localCount++
		}
	}
	if localCount != 2 {
		t.Errorf("Expected 2 local ADO dependencies, got %d", localCount)
	}
}

func TestPackageScanner_TerraformADOModules(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	mainTF := `terraform {
  required_version = ">= 1.0"
}

# Public GitHub module
module "vpc" {
  source = "github.com/terraform-aws-modules/terraform-aws-vpc"
}

# ADO internal module
module "internal" {
  source = "git::https://dev.azure.com/myorg/platform/_git/terraform-modules//network"
}

# ADO external module (different org)
module "external-ado" {
  source = "git::https://dev.azure.com/otherorg/shared/_git/common-modules"
}
`
	writeTestFile(t, dir, "main.tf", mainTF)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://dev.azure.com/myorg/project/_git/repo")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 3 dependencies (1 GitHub, 2 ADO)
	if len(deps) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(deps))
		for _, dep := range deps {
			t.Logf("Found: %s (local: %v, host: %s)", dep.DependencyFullName, dep.IsLocal, dep.DependencyURL)
		}
	}

	// Count local vs external
	localCount := 0
	for _, dep := range deps {
		if dep.IsLocal {
			localCount++
		}
	}
	// Only the myorg module should be local
	if localCount != 1 {
		t.Errorf("Expected 1 local ADO dependency, got %d", localCount)
	}
}

func TestPackageScanner_PythonADODependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	requirements := `flask==2.3.0
git+https://github.com/pallets/click.git@8.0.0
git+https://dev.azure.com/myorg/myproject/_git/internal-package@v1.0.0#egg=internal-package
requests>=2.28.0
`
	writeTestFile(t, dir, "requirements.txt", requirements)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://dev.azure.com/myorg/project/_git/repo")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 2 git dependencies (1 GitHub, 1 ADO)
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
		for _, dep := range deps {
			t.Logf("Found: %s (local: %v)", dep.DependencyFullName, dep.IsLocal)
		}
	}

	// Count local
	localCount := 0
	for _, dep := range deps {
		if dep.IsLocal {
			localCount++
		}
	}
	if localCount != 1 {
		t.Errorf("Expected 1 local ADO dependency, got %d", localCount)
	}
}

func TestPackageScanner_RubyGemfileADODependencies(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	gemfile := `source 'https://rubygems.org'

gem 'rails'
gem 'external-gem', github: 'external/gem'
gem 'internal-gem', git: 'https://dev.azure.com/myorg/myproject/_git/internal-gem.git'
`
	writeTestFile(t, dir, "Gemfile", gemfile)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	scanner := NewPackageScanner(logger).WithSourceURL("https://dev.azure.com/myorg/project/_git/repo")

	deps, err := scanner.ScanPackageManagers(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("ScanPackageManagers failed: %v", err)
	}

	// Should find 2 git dependencies (1 GitHub, 1 ADO)
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
		for _, dep := range deps {
			t.Logf("Found: %s (local: %v)", dep.DependencyFullName, dep.IsLocal)
		}
	}

	// Count local
	localCount := 0
	for _, dep := range deps {
		if dep.IsLocal {
			localCount++
		}
	}
	if localCount != 1 {
		t.Errorf("Expected 1 local ADO dependency, got %d", localCount)
	}
}
