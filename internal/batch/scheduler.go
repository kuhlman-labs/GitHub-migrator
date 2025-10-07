package batch

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// MigrationExecutor is the interface for executing repository migrations
type MigrationExecutor interface {
	ExecuteMigration(ctx context.Context, repo *models.Repository, dryRun bool) error
}

// Scheduler handles batch scheduling and execution
type Scheduler struct {
	storage  *storage.Database
	executor MigrationExecutor
	logger   *slog.Logger
	mu       sync.RWMutex
	running  map[int64]context.CancelFunc // batchID -> cancel function
}

// SchedulerConfig holds configuration for the batch scheduler
type SchedulerConfig struct {
	Storage  *storage.Database
	Executor MigrationExecutor
	Logger   *slog.Logger
}

// NewScheduler creates a new batch scheduler
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Executor == nil {
		return nil, fmt.Errorf("executor is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &Scheduler{
		storage:  cfg.Storage,
		executor: cfg.Executor,
		logger:   cfg.Logger,
		running:  make(map[int64]context.CancelFunc),
	}, nil
}

// ScheduleBatch schedules a batch to start at a specific time
func (s *Scheduler) ScheduleBatch(ctx context.Context, batchID int64, scheduledAt time.Time) error {
	s.logger.Info("Scheduling batch", "batch_id", batchID, "scheduled_at", scheduledAt)

	batch, err := s.storage.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch not found")
	}

	// Update batch scheduled time
	batch.ScheduledAt = &scheduledAt
	if err := s.storage.UpdateBatch(ctx, batch); err != nil {
		return fmt.Errorf("failed to update batch schedule: %w", err)
	}

	s.logger.Info("Batch scheduled successfully", "batch_id", batchID, "scheduled_at", scheduledAt)

	return nil
}

// ExecuteBatch executes all migrations in a batch
func (s *Scheduler) ExecuteBatch(ctx context.Context, batchID int64, dryRun bool) error {
	s.logger.Info("Starting batch execution", "batch_id", batchID, "dry_run", dryRun)

	// Check if batch is already running
	s.mu.RLock()
	if _, running := s.running[batchID]; running {
		s.mu.RUnlock()
		return fmt.Errorf("batch %d is already running", batchID)
	}
	s.mu.RUnlock()

	// Get batch
	batch, err := s.storage.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch not found")
	}

	// Get all repositories in batch
	repos, err := s.storage.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		return fmt.Errorf("failed to list batch repositories: %w", err)
	}

	if len(repos) == 0 {
		return fmt.Errorf("batch has no repositories")
	}

	// Filter repositories that can be migrated
	var migratable []*models.Repository
	for _, repo := range repos {
		if canMigrate(repo.Status) {
			migratable = append(migratable, repo)
		}
	}

	if len(migratable) == 0 {
		return fmt.Errorf("no repositories in batch can be migrated")
	}

	s.logger.Info("Found migratable repositories", "count", len(migratable), "total", len(repos))

	// Update batch status
	batch.Status = "in_progress"
	now := time.Now()
	batch.StartedAt = &now
	if err := s.storage.UpdateBatch(ctx, batch); err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	// Create cancellable context for this batch
	batchCtx, cancel := context.WithCancel(ctx)

	// Register running batch
	s.mu.Lock()
	s.running[batchID] = cancel
	s.mu.Unlock()

	// Start batch execution in background
	go s.executeBatchAsync(batchCtx, batch, migratable, dryRun)

	return nil
}

// executeBatchAsync executes batch migrations asynchronously
func (s *Scheduler) executeBatchAsync(ctx context.Context, batch *models.Batch, repos []*models.Repository, dryRun bool) {
	defer func() {
		// Clean up running batch
		s.mu.Lock()
		delete(s.running, batch.ID)
		s.mu.Unlock()
	}()

	s.logger.Info("Starting async batch execution",
		"batch_id", batch.ID,
		"batch_name", batch.Name,
		"repo_count", len(repos),
		"dry_run", dryRun)

	// Execute migrations sequentially for each repository
	successCount := 0
	failCount := 0

	for _, repo := range repos {
		select {
		case <-ctx.Done():
			s.logger.Warn("Batch execution cancelled",
				"batch_id", batch.ID,
				"completed", successCount,
				"failed", failCount)
			return
		default:
		}

		s.logger.Info("Executing migration",
			"batch_id", batch.ID,
			"repo", repo.FullName,
			"dry_run", dryRun)

		// Execute migration
		if err := s.executor.ExecuteMigration(ctx, repo, dryRun); err != nil {
			s.logger.Error("Migration failed",
				"batch_id", batch.ID,
				"repo", repo.FullName,
				"error", err)
			failCount++
		} else {
			s.logger.Info("Migration completed",
				"batch_id", batch.ID,
				"repo", repo.FullName)
			successCount++
		}
	}

	// Update batch completion
	s.logger.Info("Batch execution completed",
		"batch_id", batch.ID,
		"success", successCount,
		"failed", failCount)

	s.completeBatch(context.Background(), batch.ID, successCount, failCount)
}

// completeBatch marks a batch as completed
func (s *Scheduler) completeBatch(ctx context.Context, batchID int64, successCount, failCount int) {
	batch, err := s.storage.GetBatch(ctx, batchID)
	if err != nil {
		s.logger.Error("Failed to get batch for completion", "batch_id", batchID, "error", err)
		return
	}

	if batch == nil {
		s.logger.Error("Batch not found for completion", "batch_id", batchID)
		return
	}

	now := time.Now()
	batch.CompletedAt = &now

	// Set status based on results
	if failCount == 0 {
		batch.Status = "completed"
	} else if successCount == 0 {
		batch.Status = "failed"
	} else {
		batch.Status = "completed_with_errors"
	}

	if err := s.storage.UpdateBatch(ctx, batch); err != nil {
		s.logger.Error("Failed to update batch completion", "batch_id", batchID, "error", err)
	}

	s.logger.Info("Batch marked as complete",
		"batch_id", batchID,
		"status", batch.Status,
		"success", successCount,
		"failed", failCount)
}

// CancelBatch cancels a running batch execution
func (s *Scheduler) CancelBatch(ctx context.Context, batchID int64) error {
	s.mu.Lock()
	cancel, exists := s.running[batchID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("batch %d is not running", batchID)
	}
	delete(s.running, batchID)
	s.mu.Unlock()

	s.logger.Info("Cancelling batch execution", "batch_id", batchID)

	// Cancel the batch context
	cancel()

	// Update batch status
	batch, err := s.storage.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	if batch != nil {
		batch.Status = "cancelled"
		if err := s.storage.UpdateBatch(ctx, batch); err != nil {
			s.logger.Error("Failed to update batch status", "batch_id", batchID, "error", err)
		}
	}

	return nil
}

// ExecuteSequentialBatches executes multiple batches sequentially
// This is useful for executing waves in order
func (s *Scheduler) ExecuteSequentialBatches(ctx context.Context, batchIDs []int64, dryRun bool) error {
	s.logger.Info("Starting sequential batch execution", "batch_count", len(batchIDs), "dry_run", dryRun)

	for i, batchID := range batchIDs {
		s.logger.Info("Executing batch in sequence",
			"batch_number", i+1,
			"total_batches", len(batchIDs),
			"batch_id", batchID)

		if err := s.ExecuteBatch(ctx, batchID, dryRun); err != nil {
			s.logger.Error("Failed to execute batch in sequence",
				"batch_id", batchID,
				"error", err)
			return fmt.Errorf("failed to execute batch %d: %w", batchID, err)
		}

		// Wait for batch to complete
		if err := s.waitForBatchCompletion(ctx, batchID); err != nil {
			s.logger.Error("Batch execution did not complete successfully",
				"batch_id", batchID,
				"error", err)
			// Continue to next batch even if this one failed
		}

		s.logger.Info("Batch completed in sequence",
			"batch_number", i+1,
			"batch_id", batchID)
	}

	s.logger.Info("Sequential batch execution completed", "batch_count", len(batchIDs))

	return nil
}

// waitForBatchCompletion waits for a batch to complete execution
func (s *Scheduler) waitForBatchCompletion(ctx context.Context, batchID int64) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(168 * time.Hour) // 7 days max

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("batch completion timeout exceeded")
		case <-ticker.C:
			// Check if batch is still running
			s.mu.RLock()
			_, running := s.running[batchID]
			s.mu.RUnlock()

			if !running {
				// Batch completed
				return nil
			}
		}
	}
}

// GetRunningBatches returns the IDs of all currently running batches
func (s *Scheduler) GetRunningBatches() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]int64, 0, len(s.running))
	for id := range s.running {
		ids = append(ids, id)
	}

	return ids
}

// IsBatchRunning returns true if a batch is currently executing
func (s *Scheduler) IsBatchRunning(batchID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, running := s.running[batchID]
	return running
}

// canMigrate checks if a repository can be migrated
func canMigrate(status string) bool {
	switch models.MigrationStatus(status) {
	case models.StatusPending,
		models.StatusDryRunQueued, // Allow re-queuing dry runs
		models.StatusDryRunFailed, // Allow retrying failed dry runs
		models.StatusDryRunComplete,
		models.StatusPreMigration,
		models.StatusMigrationFailed: // Allow retrying failed migrations
		return true
	default:
		return false
	}
}
