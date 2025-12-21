package handlers

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

const (
	testMainBranch = "main"
)

// ============================================================================
// Test Configuration
// ============================================================================

// TestConfig allows customizing test setup for handlers.
// Use DefaultTestConfig() as a starting point and modify as needed.
type TestConfig struct {
	// UseMockDB uses MockDataStore instead of real SQLite database.
	// This is faster and allows error injection for testing error paths.
	UseMockDB bool

	// AuthEnabled enables authentication checks in the handler.
	AuthEnabled bool

	// SourceType specifies the source type ("github" or "azuredevops").
	SourceType string

	// SourceBaseURL is the base URL for the source.
	SourceBaseURL string

	// PreloadRepos is a list of repositories to preload into the mock data store.
	PreloadRepos []*models.Repository

	// PreloadBatches is a list of batches to preload into the mock data store.
	PreloadBatches []*models.Batch

	// MockErrors allows injecting errors into the mock data store.
	MockErrors *MockDataStoreErrors
}

// MockDataStoreErrors contains error values to inject into MockDataStore.
type MockDataStoreErrors struct {
	GetRepoErr     error
	SaveRepoErr    error
	GetBatchErr    error
	CreateBatchErr error
}

// DefaultTestConfig returns a TestConfig with sensible defaults for most tests.
// By default, it uses MockDataStore for speed.
func DefaultTestConfig() TestConfig {
	return TestConfig{
		UseMockDB:     true, // Default to mock for speed
		AuthEnabled:   false,
		SourceType:    models.SourceTypeGitHub,
		SourceBaseURL: "https://api.github.com",
	}
}

// WithRealDB returns a TestConfig that uses a real SQLite in-memory database.
// Use this when you need to test complex queries or database constraints.
func (c TestConfig) WithRealDB() TestConfig {
	c.UseMockDB = false
	return c
}

// WithAuth returns a TestConfig with authentication enabled.
func (c TestConfig) WithAuth() TestConfig {
	c.AuthEnabled = true
	return c
}

// WithSourceType returns a TestConfig with the specified source type.
func (c TestConfig) WithSourceType(sourceType string) TestConfig {
	c.SourceType = sourceType
	return c
}

// WithRepos returns a TestConfig with preloaded repositories.
func (c TestConfig) WithRepos(repos ...*models.Repository) TestConfig {
	c.PreloadRepos = repos
	return c
}

// WithBatches returns a TestConfig with preloaded batches.
func (c TestConfig) WithBatches(batches ...*models.Batch) TestConfig {
	c.PreloadBatches = batches
	return c
}

// WithErrors returns a TestConfig with error injection configured.
func (c TestConfig) WithErrors(errors *MockDataStoreErrors) TestConfig {
	c.MockErrors = errors
	return c
}

// ============================================================================
// Mock Source Provider
// ============================================================================

// MockSourceProvider is a configurable mock implementation of source.Provider.
// It replaces both mockSourceProvider and mockADOSourceProvider.
type MockSourceProvider struct {
	providerType source.ProviderType
	providerName string
}

// NewMockSourceProvider creates a new MockSourceProvider with the specified type.
func NewMockSourceProvider(providerType source.ProviderType) *MockSourceProvider {
	name := "Mock Provider"
	switch providerType {
	case source.ProviderGitHub:
		name = "Mock GitHub Provider"
	case source.ProviderAzureDevOps:
		name = "Mock Azure DevOps Provider"
	case source.ProviderGitLab:
		name = "Mock GitLab Provider"
	}
	return &MockSourceProvider{
		providerType: providerType,
		providerName: name,
	}
}

func (m *MockSourceProvider) Type() source.ProviderType {
	return m.providerType
}

func (m *MockSourceProvider) Name() string {
	return m.providerName
}

func (m *MockSourceProvider) CloneRepository(_ context.Context, _ source.RepositoryInfo, _ string, _ source.CloneOptions) error {
	return nil
}

func (m *MockSourceProvider) GetAuthenticatedCloneURL(cloneURL string) (string, error) {
	return cloneURL, nil
}

func (m *MockSourceProvider) ValidateCredentials(_ context.Context) error {
	return nil
}

func (m *MockSourceProvider) SupportsFeature(_ source.Feature) bool {
	return true
}

// Backward compatibility: keep the old mockSourceProvider as an alias
type mockSourceProvider = MockSourceProvider

// ============================================================================
// Setup Functions
// ============================================================================

// setupTestDB creates a real SQLite in-memory database for testing.
// Use this when you need to test complex queries or database constraints.
func setupTestDB(t *testing.T) *storage.Database {
	t.Helper()
	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, err := storage.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	return db
}

// setupTestHandler creates a Handler with a real SQLite database.
// This is the original function, kept for backward compatibility with existing tests.
func setupTestHandler(t *testing.T) (*Handler, *storage.Database) {
	t.Helper()
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}
	handler := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")
	return handler, db
}

// setupTestHandlerWithConfig creates a Handler using the provided TestConfig.
// This is the recommended function for new tests.
func setupTestHandlerWithConfig(t *testing.T, cfg TestConfig) (*Handler, DataStore) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: cfg.AuthEnabled}

	var db DataStore
	if cfg.UseMockDB {
		mock := NewMockDataStore()

		// Preload repositories
		for _, repo := range cfg.PreloadRepos {
			if repo.ID == 0 {
				repo.ID = mock.nextRepoID
				mock.nextRepoID++
			}
			mock.Repos[repo.FullName] = repo
			mock.ReposByID[repo.ID] = repo
		}

		// Preload batches
		for _, batch := range cfg.PreloadBatches {
			if batch.ID == 0 {
				batch.ID = mock.nextBatchID
				mock.nextBatchID++
			}
			mock.Batches[batch.ID] = batch
		}

		// Configure error injection
		if cfg.MockErrors != nil {
			mock.GetRepoErr = cfg.MockErrors.GetRepoErr
			mock.SaveRepoErr = cfg.MockErrors.SaveRepoErr
			mock.GetBatchErr = cfg.MockErrors.GetBatchErr
			mock.CreateBatchErr = cfg.MockErrors.CreateBatchErr
		}

		db = mock
	} else {
		realDB := setupTestDB(t)

		// Preload repositories into real DB
		ctx := context.Background()
		for _, repo := range cfg.PreloadRepos {
			if err := realDB.SaveRepository(ctx, repo); err != nil {
				t.Fatalf("Failed to preload repository: %v", err)
			}
		}

		// Preload batches into real DB
		for _, batch := range cfg.PreloadBatches {
			if err := realDB.CreateBatch(ctx, batch); err != nil {
				t.Fatalf("Failed to preload batch: %v", err)
			}
		}

		db = realDB
	}

	handler := NewHandlerWithDataStore(db, logger, nil, nil, nil, nil, authConfig, cfg.SourceBaseURL, cfg.SourceType)
	return handler, db
}

// setupTestHandlerWithMock creates a Handler with a pre-configured MockDataStore.
// Use this when you need fine-grained control over the mock.
func setupTestHandlerWithMock(t *testing.T, mock *MockDataStore) *Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}
	return NewHandlerWithDataStore(mock, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")
}

// ============================================================================
// DualClient Helpers
// ============================================================================

// createTestDualClient creates a DualClient with minimal configuration for testing.
func createTestDualClient(t *testing.T, logger *slog.Logger) *github.DualClient {
	t.Helper()

	cfg := github.DualClientConfig{
		PATConfig: github.ClientConfig{
			BaseURL:     "https://api.github.com",
			Token:       "ghp_test_token",
			RetryConfig: github.DefaultRetryConfig(),
			Logger:      logger,
		},
		Logger: logger,
	}

	dc, err := github.NewDualClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create test DualClient: %v", err)
	}

	return dc
}

// ============================================================================
// Test Fixtures
// ============================================================================

// createTestRepo creates a test repository with sensible defaults.
func createTestRepo(fullName string, status models.MigrationStatus) *models.Repository {
	return &models.Repository{
		FullName: fullName,
		Status:   string(status),
		Source:   "github",
	}
}
