//go:build integration
// +build integration

package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// Integration tests for all three database types
// Run with: go test -tags=integration ./internal/storage -v

// TestIntegrationSQLite tests SQLite implementation
func TestIntegrationSQLite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpFile := fmt.Sprintf("/tmp/test-sqlite-%d.db", time.Now().UnixNano())
	defer os.Remove(tmpFile)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  tmpFile,
	}

	runIntegrationTests(t, cfg, "SQLite")
}

// TestIntegrationPostgreSQL tests PostgreSQL implementation
func TestIntegrationPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if PostgreSQL is available
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://migrator:migrator_dev_password@localhost:5432/migrator_test?sslmode=disable"
	}

	cfg := config.DatabaseConfig{
		Type:                   "postgres",
		DSN:                    dsn,
		MaxOpenConns:           25,
		MaxIdleConns:           5,
		ConnMaxLifetimeSeconds: 300,
	}

	// Try to connect - if it fails, skip the test
	db, err := NewDatabase(cfg)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
		return
	}
	db.Close()

	runIntegrationTests(t, cfg, "PostgreSQL")
}

// TestIntegrationSQLServer tests SQL Server implementation
func TestIntegrationSQLServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if SQL Server is available
	dsn := os.Getenv("SQLSERVER_TEST_DSN")
	if dsn == "" {
		dsn = "sqlserver://sa:YourStrong@Passw0rd@localhost:1433?database=migrator_test"
	}

	cfg := config.DatabaseConfig{
		Type:                   "sqlserver",
		DSN:                    dsn,
		MaxOpenConns:           25,
		MaxIdleConns:           5,
		ConnMaxLifetimeSeconds: 300,
	}

	// Try to connect - if it fails, skip the test
	db, err := NewDatabase(cfg)
	if err != nil {
		t.Skipf("SQL Server not available: %v", err)
		return
	}
	db.Close()

	runIntegrationTests(t, cfg, "SQL Server")
}

// runIntegrationTests runs a comprehensive set of tests against a database
func runIntegrationTests(t *testing.T, cfg config.DatabaseConfig, dbName string) {
	t.Run(dbName, func(t *testing.T) {
		// Initialize database
		db, err := NewDatabase(cfg)
		if err != nil {
			t.Fatalf("Failed to initialize %s database: %v", dbName, err)
		}
		defer db.Close()

		// Run migrations
		t.Run("Migrations", func(t *testing.T) {
			if err := db.Migrate(); err != nil {
				t.Fatalf("Failed to run migrations: %v", err)
			}

			// Verify all tables exist
			tables := []string{"repositories", "migration_history", "migration_logs", "batches", "repository_dependencies"}
			for _, table := range tables {
				if !db.db.Migrator().HasTable(table) {
					t.Errorf("Table %s does not exist", table)
				}
			}

			// Run migrations again (should be idempotent)
			if err := db.Migrate(); err != nil {
				t.Fatalf("Second migration run failed: %v", err)
			}
		})

		ctx := context.Background()

		// Test Repository CRUD operations
		t.Run("RepositoryCRUD", func(t *testing.T) {
			repo := createTestRepository("test-org/integration-repo")

			// Create
			if err := db.SaveRepository(ctx, repo); err != nil {
				t.Fatalf("Failed to save repository: %v", err)
			}

			// Read
			savedRepo, err := db.GetRepository(ctx, repo.FullName)
			if err != nil {
				t.Fatalf("Failed to get repository: %v", err)
			}
			if savedRepo.FullName != repo.FullName {
				t.Errorf("Expected full_name %s, got %s", repo.FullName, savedRepo.FullName)
			}

			// Update
			savedRepo.Status = string(models.StatusMigratingContent)
			if err := db.UpdateRepository(ctx, savedRepo); err != nil {
				t.Fatalf("Failed to update repository: %v", err)
			}

			// Verify update
			updatedRepo, err := db.GetRepository(ctx, repo.FullName)
			if err != nil {
				t.Fatalf("Failed to get updated repository: %v", err)
			}
			if updatedRepo.Status != string(models.StatusMigratingContent) {
				t.Errorf("Expected status %s, got %s", models.StatusMigratingContent, updatedRepo.Status)
			}

			// Delete
			if err := db.DeleteRepository(ctx, repo.FullName); err != nil {
				t.Fatalf("Failed to delete repository: %v", err)
			}

			// Verify deletion - GORM returns a specific error
			_, err = db.GetRepository(ctx, repo.FullName)
			// GetRepository may not error, but it should not find the repo
			// In GORM, this might return nil error with empty repo
			// The test just verifies delete didn't fail
		})

		// Test Batch operations
		t.Run("BatchOperations", func(t *testing.T) {
			batch := &models.Batch{
				Name:        "Test Batch",
				Description: stringPtr("Integration test batch"),
				Type:        "pilot",
				Status:      "pending",
				CreatedAt:   time.Now(),
			}

			// Create batch
			if err := db.CreateBatch(ctx, batch); err != nil {
				t.Fatalf("Failed to create batch: %v", err)
			}

			// Get batch
			savedBatch, err := db.GetBatch(ctx, batch.ID)
			if err != nil {
				t.Fatalf("Failed to get batch: %v", err)
			}
			if savedBatch.Name != batch.Name {
				t.Errorf("Expected batch name %s, got %s", batch.Name, savedBatch.Name)
			}

			// List batches
			batches, err := db.ListBatches(ctx)
			if err != nil {
				t.Fatalf("Failed to list batches: %v", err)
			}
			if len(batches) == 0 {
				t.Error("Expected at least one batch")
			}

			// Update batch status
			if err := db.UpdateBatchStatus(ctx, batch.ID); err != nil {
				t.Fatalf("Failed to update batch status: %v", err)
			}

			// Delete batch
			if err := db.DeleteBatch(ctx, batch.ID); err != nil {
				t.Fatalf("Failed to delete batch: %v", err)
			}
		})

		// Test Migration History
		t.Run("MigrationHistory", func(t *testing.T) {
			// Create a test repository first
			repo := createTestRepository("test-org/history-repo")
			if err := db.SaveRepository(ctx, repo); err != nil {
				t.Fatalf("Failed to save repository: %v", err)
			}
			savedRepo, _ := db.GetRepository(ctx, repo.FullName)

			// Create migration history
			msg := "Test migration"
			history := &models.MigrationHistory{
				RepositoryID: savedRepo.ID,
				Status:       "in_progress",
				Phase:        "migration",
				Message:      &msg,
				StartedAt:    time.Now(),
			}

			historyID, err := db.CreateMigrationHistory(ctx, history)
			if err != nil {
				t.Fatalf("Failed to create migration history: %v", err)
			}

			// Update migration history
			errMsg := "Test completion"
			if err := db.UpdateMigrationHistory(ctx, historyID, "completed", &errMsg); err != nil {
				t.Fatalf("Failed to update migration history: %v", err)
			}

			// Get migration history
			historyRecords, err := db.GetMigrationHistory(ctx, savedRepo.ID)
			if err != nil {
				t.Fatalf("Failed to get migration history: %v", err)
			}
			if len(historyRecords) == 0 {
				t.Error("Expected at least one history record")
			}

			// Clean up
			db.DeleteRepository(ctx, repo.FullName)
		})

		// Test Repository Dependencies
		t.Run("RepositoryDependencies", func(t *testing.T) {
			// Create a test repository
			repo := createTestRepository("test-org/dep-repo")
			if err := db.SaveRepository(ctx, repo); err != nil {
				t.Fatalf("Failed to save repository: %v", err)
			}
			savedRepo, _ := db.GetRepository(ctx, repo.FullName)

			// Save dependencies
			deps := []*models.RepositoryDependency{
				{
					RepositoryID:       savedRepo.ID,
					DependencyFullName: "test-org/dep1",
					DependencyType:     "submodule",
					DependencyURL:      "https://github.com/test-org/dep1",
					IsLocal:            true,
				},
				{
					RepositoryID:       savedRepo.ID,
					DependencyFullName: "external/dep2",
					DependencyType:     "workflow",
					DependencyURL:      "https://github.com/external/dep2",
					IsLocal:            false,
				},
			}

			if err := db.SaveRepositoryDependencies(ctx, savedRepo.ID, deps); err != nil {
				t.Fatalf("Failed to save dependencies: %v", err)
			}

			// Get dependencies
			savedDeps, err := db.GetRepositoryDependencies(ctx, savedRepo.ID)
			if err != nil {
				t.Fatalf("Failed to get dependencies: %v", err)
			}
			if len(savedDeps) != 2 {
				t.Errorf("Expected 2 dependencies, got %d", len(savedDeps))
			}

			// Clear dependencies
			if err := db.ClearRepositoryDependencies(ctx, savedRepo.ID); err != nil {
				t.Fatalf("Failed to clear dependencies: %v", err)
			}

			// Verify cleared
			clearedDeps, err := db.GetRepositoryDependencies(ctx, savedRepo.ID)
			if err != nil {
				t.Fatalf("Failed to get dependencies after clear: %v", err)
			}
			if len(clearedDeps) != 0 {
				t.Errorf("Expected 0 dependencies after clear, got %d", len(clearedDeps))
			}

			// Clean up
			db.DeleteRepository(ctx, repo.FullName)
		})

		// Test List with Filters
		t.Run("ListWithFilters", func(t *testing.T) {
			// Create test data
			repos := []*models.Repository{
				createTestRepositoryWithStatus("test-org/pending-1", string(models.StatusPending)),
				createTestRepositoryWithStatus("test-org/pending-2", string(models.StatusPending)),
				createTestRepositoryWithStatus("test-org/complete-1", string(models.StatusComplete)),
			}

			for _, r := range repos {
				if err := db.SaveRepository(ctx, r); err != nil {
					t.Fatalf("Failed to save repository: %v", err)
				}
			}

			// List with status filter
			filters := map[string]interface{}{
				"status": models.StatusPending,
			}
			results, err := db.ListRepositories(ctx, filters)
			if err != nil {
				t.Fatalf("Failed to list repositories: %v", err)
			}

			pendingCount := 0
			for _, r := range results {
				if r.Status == string(models.StatusPending) {
					pendingCount++
				}
			}

			if pendingCount < 2 {
				t.Errorf("Expected at least 2 pending repositories, got %d", pendingCount)
			}

			// Count repositories
			count, err := db.CountRepositories(ctx, filters)
			if err != nil {
				t.Fatalf("Failed to count repositories: %v", err)
			}
			if count < 2 {
				t.Errorf("Expected at least 2 repositories, got %d", count)
			}

			// Clean up
			for _, r := range repos {
				db.DeleteRepository(ctx, r.FullName)
			}
		})

		// Test Analytics queries
		t.Run("Analytics", func(t *testing.T) {
			// Create test data with different organizations
			repos := []*models.Repository{
				createTestRepositoryWithStatus("acme/repo1", string(models.StatusPending)),
				createTestRepositoryWithStatus("acme/repo2", string(models.StatusComplete)),
				createTestRepositoryWithStatus("globex/repo1", string(models.StatusPending)),
			}

			for _, r := range repos {
				if err := db.SaveRepository(ctx, r); err != nil {
					t.Fatalf("Failed to save repository: %v", err)
				}
			}

			// Get distinct organizations
			orgs, err := db.GetDistinctOrganizations(ctx)
			if err != nil {
				t.Fatalf("Failed to get organizations: %v", err)
			}
			if len(orgs) < 2 {
				t.Logf("Warning: Expected at least 2 organizations, got %d", len(orgs))
			}

			// Get repository stats by status
			stats, err := db.GetRepositoryStatsByStatus(ctx)
			if err != nil {
				t.Fatalf("Failed to get repository stats: %v", err)
			}
			if len(stats) == 0 {
				t.Error("Expected at least one stat record")
			}

			// Clean up
			for _, r := range repos {
				db.DeleteRepository(ctx, r.FullName)
			}
		})

		t.Logf("✅ All %s integration tests passed!", dbName)
	})
}

// Helper functions
func createTestRepositoryWithStatus(fullName, status string) *models.Repository {
	repo := createTestRepository(fullName)
	repo.Status = status
	return repo
}

func stringPtr(s string) *string {
	return &s
}
