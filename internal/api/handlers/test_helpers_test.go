package handlers

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

const (
	testMainBranch = "main"
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

func setupTestHandler(t *testing.T) (*Handler, *storage.Database) {
	t.Helper()
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}
	handler := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")
	return handler, db
}

// createTestDualClient creates a DualClient with minimal configuration for testing
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
