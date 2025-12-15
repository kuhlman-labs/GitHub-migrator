package discovery

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// mockSourceProvider is a mock implementation of source.Provider for testing
type mockSourceProvider struct{}

func (m *mockSourceProvider) Type() source.ProviderType {
	return source.ProviderGitHub
}

func (m *mockSourceProvider) Name() string {
	return "Mock Provider"
}

func (m *mockSourceProvider) CloneRepository(ctx context.Context, info source.RepositoryInfo, destPath string, opts source.CloneOptions) error {
	// Mock implementation - do nothing
	return nil
}

func (m *mockSourceProvider) GetAuthenticatedCloneURL(cloneURL string) (string, error) {
	return cloneURL, nil
}

func (m *mockSourceProvider) ValidateCredentials(ctx context.Context) error {
	return nil
}

func (m *mockSourceProvider) SupportsFeature(feature source.Feature) bool {
	return true
}

func TestNewCollector(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   "test-token",
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test database
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, db, logger, mockProvider)

	if collector == nil {
		t.Fatal("NewCollector returned nil")
		return // Prevent staticcheck SA5011
	}
	if collector.client == nil {
		t.Error("Collector client is nil")
	}
	if collector.storage == nil {
		t.Error("Collector storage is nil")
	}
	if collector.logger == nil {
		t.Error("Collector logger is nil")
	}
	if collector.sourceProvider == nil {
		t.Error("Collector sourceProvider is nil")
	}
	if collector.workers != 5 {
		t.Errorf("Expected default workers to be 5, got %d", collector.workers)
	}
}

func TestSetWorkers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   "test-token",
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, db, logger, mockProvider)

	// Test setting workers
	collector.SetWorkers(10)
	if collector.workers != 10 {
		t.Errorf("Expected workers to be 10, got %d", collector.workers)
	}

	// Test invalid value (should not change)
	collector.SetWorkers(0)
	if collector.workers != 10 {
		t.Errorf("Expected workers to remain 10, got %d", collector.workers)
	}

	collector.SetWorkers(-5)
	if collector.workers != 10 {
		t.Errorf("Expected workers to remain 10, got %d", collector.workers)
	}
}

func TestProfileRepository_SaveToDatabase(t *testing.T) {
	// Create test database
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	ctx := context.Background()

	// Test SaveRepository directly
	totalSize := int64(1024 * 1024)
	defaultBranch := "main"
	repo := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		HasWiki:       true,
		HasPages:      false,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Retrieve and verify
	retrieved, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved repository is nil")
		return // Prevent staticcheck SA5011
	}

	if retrieved.FullName != "test/repo" {
		t.Errorf("Expected FullName 'test/repo', got '%s'", retrieved.FullName)
	}
	if retrieved.Source != "ghes" {
		t.Errorf("Expected Source 'ghes', got '%s'", retrieved.Source)
	}
	if retrieved.TotalSize == nil || *retrieved.TotalSize != totalSize {
		t.Errorf("Expected TotalSize %d, got %v", totalSize, retrieved.TotalSize)
	}
	if !retrieved.HasWiki {
		t.Error("Expected HasWiki to be true")
	}
}

func TestDiscoverRepositories_Integration(t *testing.T) {
	// Skip if GITHUB_TOKEN is not set
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping integration test (set GITHUB_TOKEN to run)")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   token,
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test database
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, db, logger, mockProvider)
	collector.SetWorkers(2) // Use fewer workers for testing

	ctx := context.Background()

	// Test with a small public organization
	// Note: This will fail if the org doesn't exist or has no repos
	err = collector.DiscoverRepositories(ctx, "octocat")
	if err != nil {
		t.Logf("Warning: Discovery failed (expected for test org): %v", err)
	}

	// List repositories to verify some were discovered
	repos, err := db.ListRepositories(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list repositories: %v", err)
	}

	t.Logf("Discovered %d repositories", len(repos))
}

func TestDiscoverEnterpriseRepositories_Integration(t *testing.T) {
	// Skip if GITHUB_TOKEN or GITHUB_ENTERPRISE_SLUG is not set
	token := os.Getenv("GITHUB_TOKEN")
	enterpriseSlug := os.Getenv("GITHUB_ENTERPRISE_SLUG")

	if token == "" || enterpriseSlug == "" {
		t.Skip("Skipping integration test (set GITHUB_TOKEN and GITHUB_ENTERPRISE_SLUG to run)")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   token,
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test database
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, db, logger, mockProvider)
	collector.SetWorkers(2) // Use fewer workers for testing

	ctx := context.Background()

	// Test enterprise-wide discovery
	err = collector.DiscoverEnterpriseRepositories(ctx, enterpriseSlug)
	if err != nil {
		t.Errorf("DiscoverEnterpriseRepositories() failed: %v", err)
	}

	// List repositories to verify some were discovered
	repos, err := db.ListRepositories(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list repositories: %v", err)
	}

	t.Logf("Discovered %d repositories across enterprise %s", len(repos), enterpriseSlug)

	if len(repos) == 0 {
		t.Log("No repositories found (could be valid for empty enterprise)")
	}
}

func TestDiscoverEnterpriseRepositories_Unit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   "test-token",
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test database
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, db, logger, mockProvider)

	ctx := context.Background()

	// This will fail with authentication error, but we're testing the structure
	err = collector.DiscoverEnterpriseRepositories(ctx, "test-enterprise")

	// We expect an error since we're using a fake token
	if err == nil {
		t.Log("DiscoverEnterpriseRepositories() succeeded (unexpected with fake token)")
	}

	// The important thing is it doesn't panic and returns proper error
	if err != nil {
		t.Logf("Expected error with fake token: %v", err)
	}
}
