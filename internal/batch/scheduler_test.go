package batch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// MockMigrationExecutor implements MigrationExecutor for testing
type MockMigrationExecutor struct {
	mu            sync.Mutex
	executedRepos []string
	shouldFail    bool
	delay         time.Duration
}

func (m *MockMigrationExecutor) ExecuteMigration(ctx context.Context, repo *models.Repository, dryRun bool) error {
	m.mu.Lock()
	m.executedRepos = append(m.executedRepos, repo.FullName)
	delay := m.delay
	m.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	if m.shouldFail {
		return fmt.Errorf("mock migration failed")
	}

	// Update repo status
	repo.Status = string(models.StatusComplete)
	return nil
}

func (m *MockMigrationExecutor) GetExecutedRepos() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.executedRepos))
	copy(result, m.executedRepos)
	return result
}

func (m *MockMigrationExecutor) SetDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = d
}

// waitForQueuedRepositories waits for repositories to be queued with retries
func waitForQueuedRepositories(t *testing.T, db *storage.Database, ctx context.Context, batchID int64, expectedCount int) int {
	t.Helper()
	maxRetries := 50 // 50 * 100ms = 5 seconds max wait
	for i := 0; i < maxRetries; i++ {
		time.Sleep(100 * time.Millisecond)

		repos, err := db.ListRepositories(ctx, map[string]interface{}{"batch_id": batchID})
		if err != nil {
			continue
		}

		queuedCount := 0
		for _, repo := range repos {
			if repo.Status == string(models.StatusQueuedForMigration) {
				queuedCount++
			}
		}

		if queuedCount == expectedCount {
			return queuedCount
		}
	}

	// Final check to get the actual count
	repos, _ := db.ListRepositories(ctx, map[string]interface{}{"batch_id": batchID})
	queuedCount := 0
	for _, repo := range repos {
		if repo.Status == string(models.StatusQueuedForMigration) {
			queuedCount++
		}
	}
	return queuedCount
}

func setupTestScheduler(t *testing.T) (*Scheduler, *storage.Database, *MockMigrationExecutor, func()) {
	t.Helper()

	// Create temporary database file for better concurrency support with race detector
	tmpFile := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  tmpFile + "?cache=shared&mode=rwc",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create mock executor
	executor := &MockMigrationExecutor{
		executedRepos: []string{},
		shouldFail:    false,
		delay:         0,
	}

	// Create scheduler
	scheduler, err := NewScheduler(SchedulerConfig{
		Storage:  db,
		Executor: executor,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return scheduler, db, executor, cleanup
}

// waitForBatchCompletion waits for a batch to complete or times out
func waitForBatchCompletion(t *testing.T, scheduler *Scheduler, batchID int64, timeout time.Duration) {
	t.Helper()
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			t.Fatalf("Timeout waiting for batch %d to complete", batchID)
		case <-ticker.C:
			if !scheduler.IsBatchRunning(batchID) {
				return
			}
		}
	}
}

func TestNewScheduler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db := &storage.Database{}
	executor := &MockMigrationExecutor{}

	tests := []struct {
		name    string
		config  SchedulerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: SchedulerConfig{
				Storage:  db,
				Executor: executor,
				Logger:   logger,
			},
			wantErr: false,
		},
		{
			name: "missing storage",
			config: SchedulerConfig{
				Executor: executor,
				Logger:   logger,
			},
			wantErr: true,
		},
		{
			name: "missing executor",
			config: SchedulerConfig{
				Storage: db,
				Logger:  logger,
			},
			wantErr: true,
		},
		{
			name: "missing logger",
			config: SchedulerConfig{
				Storage:  db,
				Executor: executor,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewScheduler(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewScheduler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduleBatch(t *testing.T) {
	scheduler, db, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test batch
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "wave_1",
		RepositoryCount: 0,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	t.Run("schedule batch successfully", func(t *testing.T) {
		scheduledTime := time.Now().Add(1 * time.Hour)

		err := scheduler.ScheduleBatch(ctx, batch.ID, scheduledTime)
		if err != nil {
			t.Fatalf("ScheduleBatch() error = %v", err)
		}

		// Verify batch was updated
		updated, _ := db.GetBatch(ctx, batch.ID)
		if updated.ScheduledAt == nil {
			t.Error("Expected ScheduledAt to be set")
		}

		if !updated.ScheduledAt.Equal(scheduledTime) {
			t.Errorf("Expected ScheduledAt %v, got %v", scheduledTime, updated.ScheduledAt)
		}
	})

	t.Run("batch not found", func(t *testing.T) {
		err := scheduler.ScheduleBatch(ctx, 99999, time.Now())
		if err == nil {
			t.Error("Expected error when batch not found")
		}
	})
}

func TestExecuteBatch(t *testing.T) {
	t.Run("execute batch successfully", func(t *testing.T) {
		scheduler, db, _, cleanup := setupTestScheduler(t)
		defer cleanup()

		ctx := context.Background()

		// Create test batch
		batch := &models.Batch{
			Name:            "Test Batch",
			Type:            "pilot",
			RepositoryCount: 2,
			Status:          "ready",
			CreatedAt:       time.Now(),
		}
		if err := db.CreateBatch(ctx, batch); err != nil {
			t.Fatalf("Failed to create batch: %v", err)
		}

		// Create test repositories (queued for migration)
		size1 := int64(100000 * 1024) // Convert KB to bytes
		size2 := int64(200000 * 1024)
		repo1 := &models.Repository{
			FullName:     "org/repo1",
			TotalSize:    &size1,
			Status:       string(models.StatusQueuedForMigration),
			Source:       "github",
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
			BatchID:      &batch.ID,
		}
		repo2 := &models.Repository{
			FullName:     "org/repo2",
			TotalSize:    &size2,
			Status:       string(models.StatusQueuedForMigration),
			Source:       "github",
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
			BatchID:      &batch.ID,
		}

		if err := db.SaveRepository(ctx, repo1); err != nil {
			t.Fatalf("Failed to create repo1: %v", err)
		}
		if err := db.SaveRepository(ctx, repo2); err != nil {
			t.Fatalf("Failed to create repo2: %v", err)
		}

		err := scheduler.ExecuteBatch(ctx, batch.ID, false)
		if err != nil {
			t.Fatalf("ExecuteBatch() error = %v", err)
		}

		// Verify batch status was updated
		updated, _ := db.GetBatch(ctx, batch.ID)
		if updated.Status != StatusInProgress {
			t.Errorf("Expected status '%s', got %s", StatusInProgress, updated.Status)
		}

		if updated.StartedAt == nil {
			t.Error("Expected StartedAt to be set")
		}

		// Wait for the scheduler to queue all repositories
		queuedCount := waitForQueuedRepositories(t, db, ctx, batch.ID, 2)
		if queuedCount != 2 {
			t.Errorf("Expected 2 repos queued for migration, got %d", queuedCount)
		}

		// Verify batch is still marked as in progress (workers would normally pick up the queued repos)
		updated, _ = db.GetBatch(ctx, batch.ID)
		if updated.Status != StatusInProgress {
			t.Errorf("Expected batch status to remain '%s', got %s", StatusInProgress, updated.Status)
		}
	})

	t.Run("cannot execute running batch", func(t *testing.T) {
		scheduler, db, executor, cleanup := setupTestScheduler(t)
		defer cleanup()

		ctx := context.Background()

		// Create a batch for this test
		batch := &models.Batch{
			Name:            "Test Batch 2",
			Type:            "pilot",
			RepositoryCount: 1,
			Status:          "ready",
			CreatedAt:       time.Now(),
		}
		if err := db.CreateBatch(ctx, batch); err != nil {
			t.Fatalf("Failed to create batch: %v", err)
		}

		// Create a repo with delay to keep it running
		size := int64(100000 * 1024)
		repo := &models.Repository{
			FullName:     "org/repo3",
			TotalSize:    &size,
			Status:       string(models.StatusQueuedForMigration),
			Source:       "github",
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
			BatchID:      &batch.ID,
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}

		// Set delay to keep batch running
		executor.SetDelay(500 * time.Millisecond)
		defer executor.SetDelay(0) // Reset for other tests

		// Start batch
		if err := scheduler.ExecuteBatch(ctx, batch.ID, false); err != nil {
			t.Fatalf("ExecuteBatch() error = %v", err)
		}

		// Try to execute same batch again while it's running
		err := scheduler.ExecuteBatch(ctx, batch.ID, false)
		if err == nil {
			t.Error("Expected error when batch is already running")
		}

		// Wait for the batch to complete before test ends
		waitForBatchCompletion(t, scheduler, batch.ID, 2*time.Second)

		// Give extra time for cleanup
		time.Sleep(100 * time.Millisecond)
	})
}

func TestCancelBatch(t *testing.T) {
	scheduler, db, executor, cleanup := setupTestScheduler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test batch
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "pilot",
		RepositoryCount: 1,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create test repository
	size := int64(100000 * 1024) // Convert KB to bytes
	repo := &models.Repository{
		FullName:     "org/repo",
		TotalSize:    &size,
		Status:       string(models.StatusQueuedForMigration),
		Source:       "github",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
		BatchID:      &batch.ID,
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}

	// Set delay so we can cancel before completion
	executor.SetDelay(1 * time.Second)

	t.Run("cancel running batch", func(t *testing.T) {
		// Start batch execution
		if err := scheduler.ExecuteBatch(ctx, batch.ID, false); err != nil {
			t.Fatalf("ExecuteBatch() error = %v", err)
		}

		// Verify it's running
		if !scheduler.IsBatchRunning(batch.ID) {
			t.Error("Expected batch to be running")
		}

		// Cancel the batch
		if err := scheduler.CancelBatch(ctx, batch.ID); err != nil {
			t.Fatalf("CancelBatch() error = %v", err)
		}

		// Verify it's no longer running
		if scheduler.IsBatchRunning(batch.ID) {
			t.Error("Expected batch to be cancelled")
		}

		// Verify batch status
		updated, _ := db.GetBatch(ctx, batch.ID)
		if updated.Status != "cancelled" {
			t.Errorf("Expected status 'cancelled', got %s", updated.Status)
		}
	})

	t.Run("cannot cancel non-running batch", func(t *testing.T) {
		// Create another batch that's not running
		batch2 := &models.Batch{
			Name:            "Another Batch",
			Type:            "wave_1",
			RepositoryCount: 0,
			Status:          "ready",
			CreatedAt:       time.Now(),
		}
		if err := db.CreateBatch(ctx, batch2); err != nil {
			t.Fatalf("Failed to create batch2: %v", err)
		}

		err := scheduler.CancelBatch(ctx, batch2.ID)
		if err == nil {
			t.Error("Expected error when cancelling non-running batch")
		}
	})
}

func TestGetRunningBatches(t *testing.T) {
	scheduler, db, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test batches
	batch1 := &models.Batch{
		Name:            "Batch 1",
		Type:            "pilot",
		RepositoryCount: 1,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}
	batch2 := &models.Batch{
		Name:            "Batch 2",
		Type:            "wave_1",
		RepositoryCount: 1,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}

	if err := db.CreateBatch(ctx, batch1); err != nil {
		t.Fatalf("Failed to create batch1: %v", err)
	}
	if err := db.CreateBatch(ctx, batch2); err != nil {
		t.Fatalf("Failed to create batch2: %v", err)
	}

	// Create test repositories
	size := int64(100000 * 1024) // Convert KB to bytes
	repo1 := &models.Repository{
		FullName:     "org/repo1",
		TotalSize:    &size,
		Status:       string(models.StatusQueuedForMigration),
		Source:       "github",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
		BatchID:      &batch1.ID,
	}
	size2 := int64(100000 * 1024)
	repo2 := &models.Repository{
		FullName:     "org/repo2",
		TotalSize:    &size2,
		Status:       string(models.StatusQueuedForMigration),
		Source:       "github",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
		BatchID:      &batch2.ID,
	}

	if err := db.SaveRepository(ctx, repo1); err != nil {
		t.Fatalf("Failed to create repo1: %v", err)
	}
	if err := db.SaveRepository(ctx, repo2); err != nil {
		t.Fatalf("Failed to create repo2: %v", err)
	}

	t.Run("track running batches", func(t *testing.T) {
		// With the new implementation, batches queue repositories and exit immediately
		// So we check database batch status instead of the running map

		// Start first batch
		if err := scheduler.ExecuteBatch(ctx, batch1.ID, false); err != nil {
			t.Fatalf("ExecuteBatch(batch1) error = %v", err)
		}

		// Give it time to queue repositories
		time.Sleep(100 * time.Millisecond)

		// Check batch1 is in progress in the database
		batch1Updated, _ := db.GetBatch(ctx, batch1.ID)
		if batch1Updated.Status != StatusInProgress {
			t.Errorf("Expected batch1 status '%s', got %s", StatusInProgress, batch1Updated.Status)
		}

		// Start second batch
		if err := scheduler.ExecuteBatch(ctx, batch2.ID, false); err != nil {
			t.Fatalf("ExecuteBatch(batch2) error = %v", err)
		}

		// Give it time to queue repositories
		time.Sleep(100 * time.Millisecond)

		// Check batch2 is also in progress in the database
		batch2Updated, _ := db.GetBatch(ctx, batch2.ID)
		if batch2Updated.Status != StatusInProgress {
			t.Errorf("Expected batch2 status '%s', got %s", StatusInProgress, batch2Updated.Status)
		}

		// Verify both batches have repositories queued
		repos1, _ := db.ListRepositories(ctx, map[string]interface{}{"batch_id": batch1.ID})
		if repos1[0].Status != string(models.StatusQueuedForMigration) {
			t.Errorf("Expected batch1 repo queued, got status %s", repos1[0].Status)
		}

		repos2, _ := db.ListRepositories(ctx, map[string]interface{}{"batch_id": batch2.ID})
		if repos2[0].Status != string(models.StatusQueuedForMigration) {
			t.Errorf("Expected batch2 repo queued, got status %s", repos2[0].Status)
		}
	})
}

func TestIsBatchRunning(t *testing.T) {
	scheduler, db, executor, cleanup := setupTestScheduler(t)
	defer cleanup()

	ctx := context.Background()

	// Create test batch
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "pilot",
		RepositoryCount: 1,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create test repository
	size := int64(100000 * 1024) // Convert KB to bytes
	repo := &models.Repository{
		FullName:     "org/repo",
		TotalSize:    &size,
		Status:       string(models.StatusQueuedForMigration),
		Source:       "github",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
		BatchID:      &batch.ID,
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}

	executor.SetDelay(1 * time.Second)

	t.Run("check batch running status", func(t *testing.T) {
		// Initially not running
		if scheduler.IsBatchRunning(batch.ID) {
			t.Error("Expected batch to not be running initially")
		}

		// Start batch
		if err := scheduler.ExecuteBatch(ctx, batch.ID, false); err != nil {
			t.Fatalf("ExecuteBatch() error = %v", err)
		}

		// Should be running
		if !scheduler.IsBatchRunning(batch.ID) {
			t.Error("Expected batch to be running after execution")
		}

		// Cancel it
		scheduler.CancelBatch(ctx, batch.ID)

		// Should not be running
		if scheduler.IsBatchRunning(batch.ID) {
			t.Error("Expected batch to not be running after cancel")
		}
	})
}

func TestCanMigrate(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{string(models.StatusPending), false},           // Must be queued first
		{string(models.StatusQueuedForMigration), true}, // Explicitly queued
		{string(models.StatusDryRunQueued), true},       // Explicitly queued for dry run
		{string(models.StatusDryRunFailed), true},       // Can retry failed dry runs
		{string(models.StatusDryRunComplete), true},     // Can migrate after dry run
		{string(models.StatusPreMigration), false},      // Intermediate status during migration
		{string(models.StatusMigrationFailed), true},    // Can retry failed migrations
		{string(models.StatusComplete), false},          // Already complete
		{string(models.StatusMigratingContent), false},  // Already migrating
		{string(models.StatusArchiveGenerating), false}, // Already in progress
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := canMigrate(tt.status)
			if result != tt.expected {
				t.Errorf("canMigrate(%s) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}
