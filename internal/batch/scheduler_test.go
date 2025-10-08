package batch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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

func setupTestScheduler(t *testing.T) (*Scheduler, *storage.Database, *MockMigrationExecutor, func()) {
	t.Helper()

	// Create in-memory database
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
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
	scheduler, db, executor, cleanup := setupTestScheduler(t)
	defer func() {
		// Wait for any async batch operations to complete before cleanup
		time.Sleep(100 * time.Millisecond)
		cleanup()
	}()

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

	t.Run("execute batch successfully", func(t *testing.T) {
		err := scheduler.ExecuteBatch(ctx, batch.ID, false)
		if err != nil {
			t.Fatalf("ExecuteBatch() error = %v", err)
		}

		// Verify batch status was updated
		updated, _ := db.GetBatch(ctx, batch.ID)
		if updated.Status != "in_progress" {
			t.Errorf("Expected status 'in_progress', got %s", updated.Status)
		}

		if updated.StartedAt == nil {
			t.Error("Expected StartedAt to be set")
		}

		// Wait for async execution to complete
		time.Sleep(200 * time.Millisecond)

		// Verify repositories were executed
		executedRepos := executor.GetExecutedRepos()
		if len(executedRepos) != 2 {
			t.Errorf("Expected 2 repos executed, got %d", len(executedRepos))
		}

		// Verify batch is no longer running
		if scheduler.IsBatchRunning(batch.ID) {
			t.Error("Expected batch to be completed and not running")
		}
	})

	t.Run("cannot execute running batch", func(t *testing.T) {
		// Create a new batch for this test
		batch2 := &models.Batch{
			Name:            "Test Batch 2",
			Type:            "pilot",
			RepositoryCount: 1,
			Status:          "ready",
			CreatedAt:       time.Now(),
		}
		if err := db.CreateBatch(ctx, batch2); err != nil {
			t.Fatalf("Failed to create batch2: %v", err)
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
			BatchID:      &batch2.ID,
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}

		// Set delay to keep batch running
		executor.SetDelay(1 * time.Second)

		// Start batch
		if err := scheduler.ExecuteBatch(ctx, batch2.ID, false); err != nil {
			t.Fatalf("ExecuteBatch() error = %v", err)
		}

		// Try to execute same batch again while it's running
		err := scheduler.ExecuteBatch(ctx, batch2.ID, false)
		if err == nil {
			t.Error("Expected error when batch is already running")
		}

		// Reset delay for other tests
		executor.SetDelay(0)
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
	scheduler, db, executor, cleanup := setupTestScheduler(t)
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

	// Set delay to keep batches running
	executor.SetDelay(2 * time.Second)

	t.Run("track running batches", func(t *testing.T) {
		// Initially no batches running
		running := scheduler.GetRunningBatches()
		if len(running) != 0 {
			t.Errorf("Expected 0 running batches, got %d", len(running))
		}

		// Start first batch
		if err := scheduler.ExecuteBatch(ctx, batch1.ID, false); err != nil {
			t.Fatalf("ExecuteBatch(batch1) error = %v", err)
		}

		running = scheduler.GetRunningBatches()
		if len(running) != 1 {
			t.Errorf("Expected 1 running batch, got %d", len(running))
		}

		// Start second batch
		if err := scheduler.ExecuteBatch(ctx, batch2.ID, false); err != nil {
			t.Fatalf("ExecuteBatch(batch2) error = %v", err)
		}

		running = scheduler.GetRunningBatches()
		if len(running) != 2 {
			t.Errorf("Expected 2 running batches, got %d", len(running))
		}

		// Cancel both
		scheduler.CancelBatch(ctx, batch1.ID)
		scheduler.CancelBatch(ctx, batch2.ID)

		running = scheduler.GetRunningBatches()
		if len(running) != 0 {
			t.Errorf("Expected 0 running batches after cancel, got %d", len(running))
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
