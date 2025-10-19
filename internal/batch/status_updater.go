package batch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// StatusUpdater periodically updates batch statuses based on repository states
type StatusUpdater struct {
	storage  *storage.Database
	logger   *slog.Logger
	interval time.Duration
	stopCh   chan struct{}
}

// StatusUpdaterConfig holds configuration for the status updater
type StatusUpdaterConfig struct {
	Storage  *storage.Database
	Logger   *slog.Logger
	Interval time.Duration // How often to check and update statuses
}

// NewStatusUpdater creates a new batch status updater
func NewStatusUpdater(cfg StatusUpdaterConfig) (*StatusUpdater, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.Interval == 0 {
		cfg.Interval = 30 * time.Second // Default to 30 seconds
	}

	return &StatusUpdater{
		storage:  cfg.Storage,
		logger:   cfg.Logger,
		interval: cfg.Interval,
		stopCh:   make(chan struct{}),
	}, nil
}

// Start begins the status update loop
func (su *StatusUpdater) Start(ctx context.Context) {
	su.logger.Info("Starting batch status updater", "interval", su.interval)

	ticker := time.NewTicker(su.interval)
	defer ticker.Stop()

	// Run once immediately
	su.updateBatchStatuses(ctx)

	for {
		select {
		case <-ctx.Done():
			su.logger.Info("Batch status updater stopped")
			return
		case <-su.stopCh:
			su.logger.Info("Batch status updater stopped")
			return
		case <-ticker.C:
			su.updateBatchStatuses(ctx)
		}
	}
}

// Stop stops the status updater
func (su *StatusUpdater) Stop() {
	close(su.stopCh)
}

// updateBatchStatuses updates the status of all active batches
func (su *StatusUpdater) updateBatchStatuses(ctx context.Context) {
	batches, err := su.storage.ListBatches(ctx)
	if err != nil {
		su.logger.Error("Failed to list batches for status update", "error", err)
		return
	}

	updated := 0
	for _, batch := range batches {
		// Only update batches that are not in a terminal or stable state
		if batch.Status == StatusReady || batch.Status == StatusPending {
			continue // Ready and pending batches don't need status updates
		}

		newStatus, err := su.calculateBatchStatus(ctx, batch)
		if err != nil {
			su.logger.Error("Failed to calculate batch status",
				"batch_id", batch.ID,
				"batch_name", batch.Name,
				"error", err)
			continue
		}

		if newStatus != batch.Status {
			su.logger.Info("Updating batch status",
				"batch_id", batch.ID,
				"batch_name", batch.Name,
				"old_status", batch.Status,
				"new_status", newStatus)

			batch.Status = newStatus

			// Set completion time if batch is now complete
			if isTerminalStatus(newStatus) && batch.CompletedAt == nil {
				now := time.Now()
				batch.CompletedAt = &now
			}

			if err := su.storage.UpdateBatch(ctx, batch); err != nil {
				su.logger.Error("Failed to update batch status",
					"batch_id", batch.ID,
					"error", err)
				continue
			}

			updated++
		}
	}

	if updated > 0 {
		su.logger.Info("Batch status update complete", "updated_count", updated)
	}
}

// calculateBatchStatus determines the appropriate status for a batch based on its repositories
func (su *StatusUpdater) calculateBatchStatus(ctx context.Context, batch *models.Batch) (string, error) {
	repos, err := su.storage.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batch.ID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list batch repositories: %w", err)
	}

	if len(repos) == 0 {
		return StatusReady, nil
	}

	return CalculateBatchStatusFromRepos(repos), nil
}

// CalculateBatchStatusFromRepos calculates batch status from repository statuses
// This is exported so it can be used by other packages (e.g., scheduler)
func CalculateBatchStatusFromRepos(repos []*models.Repository) string {
	completedCount := 0
	failedCount := 0
	inProgressCount := 0
	dryRunCompleteCount := 0
	pendingCount := 0

	for _, repo := range repos {
		switch repo.Status {
		case string(models.StatusComplete):
			completedCount++
		case string(models.StatusMigrationFailed), string(models.StatusDryRunFailed):
			failedCount++
		case string(models.StatusDryRunQueued),
			string(models.StatusDryRunInProgress),
			string(models.StatusQueuedForMigration),
			string(models.StatusMigratingContent),
			string(models.StatusArchiveGenerating),
			string(models.StatusPreMigration),
			string(models.StatusPostMigration),
			string(models.StatusMigrationComplete):
			inProgressCount++
		case string(models.StatusDryRunComplete):
			dryRunCompleteCount++
		case string(models.StatusPending):
			pendingCount++
		}
	}

	totalRepos := len(repos)

	// Determine overall batch status
	if inProgressCount > 0 {
		return StatusInProgress
	}

	// If all migrations are complete
	if completedCount == totalRepos {
		return StatusCompleted
	}

	// If all migrations failed
	if failedCount == totalRepos {
		return StatusFailed
	}

	// If some completed and some failed
	if completedCount > 0 && failedCount > 0 {
		return StatusCompletedWithErrors
	}

	// If any failed during migration
	if failedCount > 0 {
		return StatusCompletedWithErrors
	}

	// If all dry runs are complete (batch is ready for migration)
	if dryRunCompleteCount == totalRepos {
		return StatusReady
	}

	// If some dry runs complete and some failed
	if dryRunCompleteCount > 0 && failedCount > 0 {
		return StatusReady
	}

	return StatusReady
}

// isTerminalStatus returns true if the status represents a completed state
func isTerminalStatus(status string) bool {
	return status == StatusCompleted || status == StatusFailed || status == StatusCompletedWithErrors || status == StatusCancelled
}
