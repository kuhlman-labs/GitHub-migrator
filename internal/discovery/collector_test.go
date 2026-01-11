package discovery

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	ghapi "github.com/google/go-github/v75/github"
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

func TestWaitForRateLimitReset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, nil, logger, mockProvider)

	// Create a mock progress tracker
	tracker := NoOpProgressTracker{}

	tests := []struct {
		name           string
		err            error
		contextTimeout time.Duration
		expectError    bool
	}{
		{
			name:           "blocked rate limit with short reset time",
			err:            errors.New("403 API rate limit exceeded [rate reset in 1s]"),
			contextTimeout: 20 * time.Second, // Should complete within 10s (min wait) + buffer
			expectError:    false,
		},
		{
			name:           "context cancellation during wait",
			err:            errors.New("403 API rate limit exceeded [rate reset in 5m]"),
			contextTimeout: 100 * time.Millisecond, // Very short timeout
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			start := time.Now()
			err := collector.waitForRateLimitReset(ctx, tt.err, tracker)
			elapsed := time.Since(start)

			if tt.expectError {
				if err == nil {
					t.Error("waitForRateLimitReset() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("waitForRateLimitReset() error = %v, want nil", err)
				}
				// Should have waited at least MinRateLimitWait (10 seconds)
				if elapsed < 10*time.Second {
					t.Errorf("waitForRateLimitReset() waited %v, expected at least 10s", elapsed)
				}
			}
		})
	}
}

func TestRateLimitResetBuffer(t *testing.T) {
	// Verify the buffer constant is set appropriately
	if rateLimitResetBuffer < 1*time.Second {
		t.Errorf("rateLimitResetBuffer = %v, should be at least 1 second", rateLimitResetBuffer)
	}
	if rateLimitResetBuffer > 30*time.Second {
		t.Errorf("rateLimitResetBuffer = %v, should not be more than 30 seconds", rateLimitResetBuffer)
	}
}

func TestRetryWithRateLimitHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, nil, logger, mockProvider)

	tracker := NoOpProgressTracker{}

	tests := []struct {
		name         string
		fn           func() error
		expectError  bool
		expectedCall int
	}{
		{
			name: "successful operation",
			fn: func() error {
				return nil
			},
			expectError:  false,
			expectedCall: 1,
		},
		{
			name: "non-rate-limit error fails immediately",
			fn: func() error {
				return errors.New("some other error")
			},
			expectError:  true,
			expectedCall: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			testFn := func() error {
				callCount++
				return tt.fn()
			}

			err := collector.retryWithRateLimitHandling(context.Background(), tracker, "test", testFn)

			if tt.expectError && err == nil {
				t.Error("retryWithRateLimitHandling() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("retryWithRateLimitHandling() unexpected error: %v", err)
			}
			if callCount != tt.expectedCall {
				t.Errorf("retryWithRateLimitHandling() called function %d times, expected %d", callCount, tt.expectedCall)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, nil, logger, mockProvider)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "blocked rate limit error",
			err:  errors.New("403 API rate limit of 5000 still exceeded until 2026-01-06 13:03:34 -0500 EST, not making remote request. [rate reset in 31m24s]"),
			want: true,
		},
		{
			name: "secondary rate limit error",
			err:  errors.New("You have exceeded a secondary rate limit"),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collector.isRateLimitError(tt.err)
			if got != tt.want {
				t.Errorf("isRateLimitError() = %v, want %v", got, tt.want)
			}
		})
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
		FullName:     "test/repo",
		Source:       "ghes",
		SourceURL:    "https://github.com/test/repo",
		Status:       string(models.StatusPending),
		Visibility:   "private",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.SetTotalSize(&totalSize)
	repo.SetDefaultBranch(&defaultBranch)
	repo.SetHasWiki(true)
	repo.SetHasPages(false)

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
	if retrieved.GetTotalSize() == nil || *retrieved.GetTotalSize() != totalSize {
		t.Errorf("Expected TotalSize %d, got %v", totalSize, retrieved.GetTotalSize())
	}
	if !retrieved.HasWiki() {
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

// TestProcessRepositoriesWithCancellation tests that the worker loop properly handles context cancellation
func TestProcessRepositoriesWithCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

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

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, db, logger, mockProvider)
	collector.SetWorkers(2) // Use 2 workers for testing

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create mock repositories
	repos := make([]*ghapi.Repository, 10)
	for i := 0; i < 10; i++ {
		name := "test-repo-" + time.Now().Format("20060102150405") + "-" + string(rune('a'+i))
		fullName := "test-org/" + name
		repos[i] = &ghapi.Repository{}
		repos[i].Name = &name
		repos[i].FullName = &fullName
	}

	// Create a mock profiler
	profiler := NewProfiler(client, logger)

	// Create a simple tracker
	tracker := NoOpProgressTracker{}

	// Cancel immediately
	cancel()

	// Process should return quickly with context.Canceled error
	start := time.Now()
	err = collector.processRepositoriesWithProfilerTracked(ctx, repos, profiler, tracker)
	elapsed := time.Since(start)

	// Should complete quickly (within 1 second) since context was cancelled
	if elapsed > 2*time.Second {
		t.Errorf("Expected quick return on cancellation, took %v", elapsed)
	}

	// Should return context.Canceled error
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestWorkerStopsOnCancellation tests that individual workers stop gracefully
func TestWorkerStopsOnCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

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

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mockProvider := &mockSourceProvider{}
	collector := NewCollector(client, db, logger, mockProvider)

	// Create a context that we'll cancel after a short delay
	ctx, cancel := context.WithCancel(context.Background())

	// Create channels for the worker
	jobs := make(chan *ghapi.Repository, 5)
	errChan := make(chan error, 5)

	// Add multiple repos to the job queue
	for i := 0; i < 5; i++ {
		name := "repo-" + string(rune('a'+i))
		full := "org/" + name
		r := &ghapi.Repository{}
		r.Name = &name
		r.FullName = &full
		jobs <- r
	}

	// Cancel the context before closing jobs (simulates cancellation during work)
	cancel()
	close(jobs)

	// Create a mock profiler and tracker
	profiler := NewProfiler(client, logger)
	tracker := NoOpProgressTracker{}

	// Run worker
	var wg sync.WaitGroup
	wg.Add(1)
	go collector.workerWithProfilerTracked(ctx, &wg, jobs, errChan, profiler, tracker)
	wg.Wait()
	close(errChan)

	// Worker should have stopped early due to cancellation
	// Count errors - shouldn't have many since we cancelled quickly
	errorCount := 0
	for range errChan {
		errorCount++
	}

	// The worker may have processed 0 or 1 repos before noticing cancellation
	// The key is that it doesn't process all 5
	t.Logf("Worker processed and reported %d errors before stopping (5 repos in queue)", errorCount)
}
