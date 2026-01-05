package migration

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

func setupTestFactory(t *testing.T) (*ExecutorFactory, *storage.Database) {
	t.Helper()

	// Create test database
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create mock destination client
	destClient := &github.Client{}

	// Create factory
	factory, err := NewExecutorFactory(ExecutorFactoryConfig{
		Storage:              db,
		DestClient:           destClient,
		Logger:               logger,
		PostMigrationMode:    PostMigrationProductionOnly,
		DestRepoExistsAction: DestinationRepoExistsFail,
	})
	if err != nil {
		t.Fatalf("Failed to create executor factory: %v", err)
	}

	return factory, db
}

func TestNewExecutorFactory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		cfg     ExecutorFactoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			cfg: ExecutorFactoryConfig{
				Storage:    &storage.Database{},
				DestClient: &github.Client{},
				Logger:     logger,
			},
			wantErr: false,
		},
		{
			name: "missing storage",
			cfg: ExecutorFactoryConfig{
				DestClient: &github.Client{},
				Logger:     logger,
			},
			wantErr: true,
			errMsg:  "storage is required",
		},
		{
			name: "missing dest client",
			cfg: ExecutorFactoryConfig{
				Storage: &storage.Database{},
				Logger:  logger,
			},
			wantErr: true,
			errMsg:  "destination client is required",
		},
		{
			name: "missing logger",
			cfg: ExecutorFactoryConfig{
				Storage:    &storage.Database{},
				DestClient: &github.Client{},
			},
			wantErr: true,
			errMsg:  "logger is required",
		},
		{
			name: "with post migration mode",
			cfg: ExecutorFactoryConfig{
				Storage:           &storage.Database{},
				DestClient:        &github.Client{},
				Logger:            logger,
				PostMigrationMode: PostMigrationAlways,
			},
			wantErr: false,
		},
		{
			name: "with visibility handling",
			cfg: ExecutorFactoryConfig{
				Storage:    &storage.Database{},
				DestClient: &github.Client{},
				Logger:     logger,
				VisibilityHandling: VisibilityHandling{
					PublicRepos:   models.VisibilityInternal,
					InternalRepos: models.VisibilityPrivate,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewExecutorFactory(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message %q but got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if factory == nil {
				t.Error("Expected non-nil factory")
			}
		})
	}
}

func TestExecutorFactory_GetExecutorForRepository_NoSourceID(t *testing.T) {
	factory, _ := setupTestFactory(t)

	repo := &models.Repository{
		ID:       1,
		FullName: "org/repo",
		SourceID: nil, // No source ID
	}

	ctx := context.Background()
	executor, err := factory.GetExecutorForRepository(ctx, repo)

	if err == nil {
		t.Error("Expected error for repository without source_id")
	}
	if executor != nil {
		t.Error("Expected nil executor for repository without source_id")
	}
	if err != nil && err.Error() != "repository org/repo has no source_id - cannot determine source credentials" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestExecutorFactory_GetExecutorForRepository_SourceNotFound(t *testing.T) {
	factory, _ := setupTestFactory(t)

	sourceID := int64(999) // Non-existent source
	repo := &models.Repository{
		ID:       1,
		FullName: "org/repo",
		SourceID: &sourceID,
	}

	ctx := context.Background()
	executor, err := factory.GetExecutorForRepository(ctx, repo)

	if err == nil {
		t.Error("Expected error for non-existent source")
	}
	if executor != nil {
		t.Error("Expected nil executor for non-existent source")
	}
}

func TestExecutorFactory_GetExecutorForRepository_InactiveSource(t *testing.T) {
	factory, db := setupTestFactory(t)

	ctx := context.Background()

	// Create an inactive source using raw SQL to bypass GORM defaults
	err := db.DB().Exec(`INSERT INTO sources (name, type, base_url, token, is_active) VALUES (?, ?, ?, ?, ?)`,
		"Inactive Source", "github", "https://github.example.com/api/v3", "test-token", false).Error
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Get the inserted source ID
	var sourceID int64
	err = db.DB().Raw(`SELECT id FROM sources WHERE name = ?`, "Inactive Source").Scan(&sourceID).Error
	if err != nil {
		t.Fatalf("Failed to get source ID: %v", err)
	}

	repo := &models.Repository{
		ID:       1,
		FullName: "org/repo",
		SourceID: &sourceID,
	}

	executor, err := factory.GetExecutorForRepository(ctx, repo)

	if err == nil {
		t.Error("Expected error for inactive source")
	}
	if executor != nil {
		t.Error("Expected nil executor for inactive source")
	}
}

func TestExecutorFactory_GetExecutorForRepository_UnsupportedSourceType(t *testing.T) {
	factory, db := setupTestFactory(t)

	ctx := context.Background()

	// Use raw SQL to insert unsupported type since model validation would reject it
	err := db.DB().Exec(`INSERT INTO sources (name, type, base_url, token, is_active) VALUES (?, ?, ?, ?, ?)`,
		"Unsupported Source", "gitlab", "https://gitlab.example.com", "test-token", true).Error
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Get the inserted source ID
	var sourceID int64
	err = db.DB().Raw(`SELECT id FROM sources WHERE name = ?`, "Unsupported Source").Scan(&sourceID).Error
	if err != nil {
		t.Fatalf("Failed to get source ID: %v", err)
	}

	repo := &models.Repository{
		ID:       1,
		FullName: "org/repo",
		SourceID: &sourceID,
	}

	executor, err := factory.GetExecutorForRepository(ctx, repo)

	if err == nil {
		t.Error("Expected error for unsupported source type")
	}
	if executor != nil {
		t.Error("Expected nil executor for unsupported source type")
	}
}

func TestExecutorFactory_CacheInvalidation(t *testing.T) {
	factory, _ := setupTestFactory(t)

	// Initially cache should be empty
	if len(factory.executorCache) != 0 {
		t.Error("Expected empty cache initially")
	}

	// InvalidateCache for non-existent source should not panic
	factory.InvalidateCache(999)

	// InvalidateAllCaches should not panic on empty cache
	factory.InvalidateAllCaches()

	if len(factory.executorCache) != 0 {
		t.Error("Expected empty cache after InvalidateAllCaches")
	}
}

func TestExecutorFactory_InvalidateCache(t *testing.T) {
	factory, _ := setupTestFactory(t)

	// Manually populate the cache to test invalidation
	factory.cacheMu.Lock()
	factory.executorCache[1] = &Executor{}
	factory.executorCache[2] = &Executor{}
	factory.cacheMu.Unlock()

	// Invalidate source 1
	factory.InvalidateCache(1)

	factory.cacheMu.RLock()
	_, exists1 := factory.executorCache[1]
	_, exists2 := factory.executorCache[2]
	factory.cacheMu.RUnlock()

	if exists1 {
		t.Error("Source 1 should have been invalidated")
	}
	if !exists2 {
		t.Error("Source 2 should still be in cache")
	}
}

func TestExecutorFactory_InvalidateAllCaches(t *testing.T) {
	factory, _ := setupTestFactory(t)

	// Manually populate the cache
	factory.cacheMu.Lock()
	factory.executorCache[1] = &Executor{}
	factory.executorCache[2] = &Executor{}
	factory.executorCache[3] = &Executor{}
	factory.cacheMu.Unlock()

	// Invalidate all
	factory.InvalidateAllCaches()

	factory.cacheMu.RLock()
	count := len(factory.executorCache)
	factory.cacheMu.RUnlock()

	if count != 0 {
		t.Errorf("Expected empty cache after InvalidateAllCaches, got %d entries", count)
	}
}

func TestExecutorFactory_DefaultConfiguration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create factory with minimal config to test defaults
	factory, err := NewExecutorFactory(ExecutorFactoryConfig{
		Storage:    &storage.Database{},
		DestClient: &github.Client{},
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Check defaults are applied
	if factory.postMigrationMode != PostMigrationProductionOnly {
		t.Errorf("Expected default PostMigrationMode to be ProductionOnly, got %s", factory.postMigrationMode)
	}

	// Use getter methods since fields are now named differently
	if factory.getDestRepoExistsAction() != DestinationRepoExistsFail {
		t.Errorf("Expected default DestRepoExistsAction to be Fail, got %s", factory.getDestRepoExistsAction())
	}

	visHandling := factory.getVisibilityHandling()
	if visHandling.PublicRepos != models.VisibilityPrivate {
		t.Errorf("Expected default PublicRepos visibility to be private, got %s", visHandling.PublicRepos)
	}

	if visHandling.InternalRepos != models.VisibilityPrivate {
		t.Errorf("Expected default InternalRepos visibility to be private, got %s", visHandling.InternalRepos)
	}
}

func TestExecutorFactory_ExecuteMigrationInterface(t *testing.T) {
	factory, _ := setupTestFactory(t)

	// Test that ExecutorFactory implements MigrationExecutor interface
	// by verifying ExecuteMigration method exists and has correct signature
	var _ interface {
		ExecuteMigration(context.Context, *models.Repository, *models.Batch, bool) error
	} = factory

	// The actual execution will fail because we don't have real sources,
	// but this test verifies the interface is implemented correctly
}

func TestExecutorFactory_ConcurrentAccess(t *testing.T) {
	factory, _ := setupTestFactory(t)

	// Test concurrent cache access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int64) {
			defer func() { done <- true }()

			// Simulate concurrent operations
			factory.InvalidateCache(id)
			factory.cacheMu.Lock()
			factory.executorCache[id] = &Executor{}
			factory.cacheMu.Unlock()
			factory.InvalidateCache(id)
		}(int64(i))
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}
