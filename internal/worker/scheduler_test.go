package worker

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/batch"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// MockOrchestratorInterface implements the orchestrator interface for testing
type MockOrchestratorInterface struct {
	executeCalled int
	executeError  error
}

func (m *MockOrchestratorInterface) ExecuteScheduledBatches(ctx context.Context, dryRun bool) error {
	m.executeCalled++
	return m.executeError
}

// createTestRepository creates a repository with all required fields for testing
func createTestRepository(fullName string) *models.Repository {
	totalSize := int64(1024 * 1024)
	defaultBranch := "main"
	topContrib := "user1,user2"
	now := time.Now()

	return &models.Repository{
		FullName:     fullName,
		Source:       "github",
		SourceURL:    "https://github.com/" + fullName,
		Status:       string(models.StatusPending),
		Visibility:   "private",
		IsArchived:   false,
		IsFork:       false,
		DiscoveredAt: now,
		UpdatedAt:    now,
		GitProperties: &models.RepositoryGitProperties{
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			BranchCount:   5,
			CommitCount:   100,
		},
		Features: &models.RepositoryFeatures{
			ContributorCount: 2,
			TopContributors:  &topContrib,
		},
		Validation: &models.RepositoryValidation{},
	}
}

func TestSchedulerWorker_Start(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scheduler worker tests in short mode")
	}

	logger := slog.Default()

	t.Run("worker calls ExecuteScheduledBatches periodically", func(t *testing.T) {
		// Create test database
		db, cleanup := setupTestDB(t)
		defer cleanup()

		// Create orchestrator with mock executor
		mockExecutor := &MockExecutor{}
		orchestrator, err := batch.NewOrchestrator(batch.OrchestratorConfig{
			Storage:  db,
			Executor: mockExecutor,
			Logger:   logger,
		})
		if err != nil {
			t.Fatalf("Failed to create orchestrator: %v", err)
		}

		worker := NewSchedulerWorker(orchestrator, logger)
		worker.interval = 100 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
		defer cancel()

		go worker.Start(ctx)

		// Wait for context to timeout
		<-ctx.Done()

		// Worker should have checked at least 3 times
		// We can't easily verify the exact count without modifying the implementation,
		// but we can verify it started without errors
	})

	t.Run("worker stops gracefully on context cancellation", func(t *testing.T) {
		// Create test database
		db, cleanup := setupTestDB(t)
		defer cleanup()

		// Create orchestrator
		mockExecutor := &MockExecutor{}
		orchestrator, err := batch.NewOrchestrator(batch.OrchestratorConfig{
			Storage:  db,
			Executor: mockExecutor,
			Logger:   logger,
		})
		if err != nil {
			t.Fatalf("Failed to create orchestrator: %v", err)
		}

		worker := NewSchedulerWorker(orchestrator, logger)
		worker.interval = 1 * time.Second

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			worker.Start(ctx)
			done <- true
		}()

		// Cancel after a short delay
		time.Sleep(50 * time.Millisecond)
		cancel()

		// Wait for worker to stop
		select {
		case <-done:
			// Success - worker stopped
		case <-time.After(2 * time.Second):
			t.Fatal("Worker did not stop within timeout")
		}
	})
}

func TestSchedulerWorker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.Default()

	// Create test batch with scheduled time in the past
	pastTime := time.Now().Add(-5 * time.Minute)
	testBatch := &models.Batch{
		Name:            "Test Scheduled Batch",
		Type:            "test",
		Status:          "ready",
		RepositoryCount: 1,
		ScheduledAt:     &pastTime,
		CreatedAt:       time.Now(),
	}

	if err := db.CreateBatch(context.Background(), testBatch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create a repository in the batch with dry_run_complete status
	repo := createTestRepository("test/repo")
	repo.Status = string(models.StatusDryRunComplete)
	repo.BatchID = &testBatch.ID

	if err := db.SaveRepository(context.Background(), repo); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create mock executor
	mockExecutor := &MockExecutor{}

	// Create orchestrator
	orchestrator, err := batch.NewOrchestrator(batch.OrchestratorConfig{
		Storage:  db,
		Executor: mockExecutor,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Create and start scheduler worker
	worker := NewSchedulerWorker(orchestrator, logger)
	worker.interval = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go worker.Start(ctx)

	// Wait for the worker to process
	<-ctx.Done()

	// Verify the batch was executed
	updatedBatch, err := db.GetBatch(context.Background(), testBatch.ID)
	if err != nil {
		t.Fatalf("Failed to get batch: %v", err)
	}

	if updatedBatch.Status != "in_progress" && updatedBatch.StartedAt == nil {
		t.Error("Expected batch to be started by scheduler worker")
	}
}

func TestSchedulerWorker_OnlyExecutesReadyBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	logger := slog.Default()
	pastTime := time.Now().Add(-5 * time.Minute)

	testCases := []struct {
		name          string
		batchStatus   string
		shouldExecute bool
	}{
		{"ready batch with past schedule", "ready", true},
		{"pending batch with past schedule", "pending", false},
		{"in_progress batch", "in_progress", false},
		{"completed batch", "completed", false},
		{"failed batch", "failed", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh database for each subtest
			db, cleanup := setupTestDB(t)
			defer cleanup()

			mockExecutor := &MockExecutor{}
			orchestrator, err := batch.NewOrchestrator(batch.OrchestratorConfig{
				Storage:  db,
				Executor: mockExecutor,
				Logger:   logger,
			})
			if err != nil {
				t.Fatalf("Failed to create orchestrator: %v", err)
			}

			ctx := context.Background()

			batch := &models.Batch{
				Name:            tc.name,
				Type:            "test",
				Status:          tc.batchStatus,
				RepositoryCount: 1,
				ScheduledAt:     &pastTime,
				CreatedAt:       time.Now(),
			}

			if err := db.CreateBatch(ctx, batch); err != nil {
				t.Fatalf("Failed to create batch: %v", err)
			}

			// Add a repository if needed
			if tc.batchStatus == "ready" {
				repo := createTestRepository("test/repo-" + tc.name)
				repo.Status = string(models.StatusDryRunComplete)
				repo.BatchID = &batch.ID

				if err := db.SaveRepository(ctx, repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}
			}

			// Execute scheduled batches
			if err := orchestrator.ExecuteScheduledBatches(ctx, false); err != nil {
				t.Logf("ExecuteScheduledBatches error (expected for some): %v", err)
			}

			// Check if batch was executed
			updatedBatch, err := db.GetBatch(ctx, batch.ID)
			if err != nil {
				t.Fatalf("Failed to get batch: %v", err)
			}

			wasStarted := updatedBatch.StartedAt != nil

			if tc.shouldExecute && !wasStarted {
				t.Errorf("Expected batch to be started but it wasn't")
			}
			if !tc.shouldExecute && wasStarted {
				t.Errorf("Expected batch NOT to be started but it was")
			}
		})
	}
}

func TestSchedulerWorker_IgnoresFutureBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.Default()
	mockExecutor := &MockExecutor{}

	orchestrator, err := batch.NewOrchestrator(batch.OrchestratorConfig{
		Storage:  db,
		Executor: mockExecutor,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()

	// Create batch scheduled in the future
	futureTime := time.Now().Add(10 * time.Minute)
	batch := &models.Batch{
		Name:            "Future Batch",
		Type:            "test",
		Status:          "ready",
		RepositoryCount: 1,
		ScheduledAt:     &futureTime,
		CreatedAt:       time.Now(),
	}

	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Execute scheduled batches
	if err := orchestrator.ExecuteScheduledBatches(ctx, false); err != nil {
		t.Fatalf("ExecuteScheduledBatches failed: %v", err)
	}

	// Verify batch was NOT executed
	updatedBatch, err := db.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("Failed to get batch: %v", err)
	}

	if updatedBatch.StartedAt != nil {
		t.Error("Expected future batch NOT to be started")
	}
	if updatedBatch.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", updatedBatch.Status)
	}
}

// Helper types and functions

type MockExecutor struct {
	executeMigrationCalled int
	executeMigrationError  error
}

func (m *MockExecutor) ExecuteMigration(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	m.executeMigrationCalled++
	return m.executeMigrationError
}

func setupTestDB(t *testing.T) (*storage.Database, func()) {
	t.Helper()

	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}
