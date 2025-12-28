package batch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
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
//
//nolint:gocyclo // Complex status transition logic with multiple edge cases
func (su *StatusUpdater) updateBatchStatuses(ctx context.Context) {
	batches, err := su.storage.ListBatches(ctx)
	if err != nil {
		su.logger.Error("Failed to list batches for status update", "error", err)
		return
	}

	updated := 0
	for _, batch := range batches {
		// Only update batches that are not in a terminal or stable state
		if batch.Status == models.BatchStatusReady || batch.Status == models.BatchStatusPending {
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

			oldStatus := batch.Status
			batch.Status = newStatus

			// Track dry run completion: when batch transitions from in_progress to ready
			// and dry run was started (DryRunStartedAt is set), record completion time
			if oldStatus == models.BatchStatusInProgress && newStatus == models.BatchStatusReady {
				if batch.DryRunStartedAt != nil && batch.DryRunCompletedAt == nil {
					now := time.Now()
					batch.DryRunCompletedAt = &now

					// Calculate and store dry run duration
					duration := batch.DryRunDuration()
					if duration != nil {
						durationSeconds := int(duration.Seconds())
						batch.DryRunDurationSeconds = &durationSeconds

						su.logger.Info("Batch dry run completed",
							"batch_id", batch.ID,
							"batch_name", batch.Name,
							"started_at", batch.DryRunStartedAt.Format(time.RFC3339),
							"completed_at", batch.DryRunCompletedAt.Format(time.RFC3339),
							"duration_seconds", durationSeconds,
							"duration", duration.String())
					}
				}
			}

			// Set completion time if batch is now complete
			if isTerminalStatus(newStatus) && batch.CompletedAt == nil {
				now := time.Now()
				batch.CompletedAt = &now

				// Log completion with duration
				if batch.StartedAt != nil {
					duration := batch.Duration()
					if duration != nil {
						su.logger.Info("Batch completed",
							"batch_id", batch.ID,
							"batch_name", batch.Name,
							"status", newStatus,
							"started_at", batch.StartedAt.Format(time.RFC3339),
							"completed_at", batch.CompletedAt.Format(time.RFC3339),
							"duration_seconds", duration.Seconds(),
							"duration", duration.String())
					}
				}
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
	repos, err := su.storage.ListRepositories(ctx, map[string]any{
		"batch_id": batch.ID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list batch repositories: %w", err)
	}

	if len(repos) == 0 {
		return models.BatchStatusReady, nil
	}

	return CalculateBatchStatusFromRepos(repos), nil
}

// CalculateBatchStatusFromRepos calculates batch status from repository statuses
// This is exported so it can be used by other packages (e.g., scheduler)
//
//nolint:gocyclo // Complex status calculation logic
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
		return models.BatchStatusInProgress
	}

	// If all migrations are complete
	if completedCount == totalRepos {
		return models.BatchStatusCompleted
	}

	// If all migrations failed
	if failedCount == totalRepos {
		return models.BatchStatusFailed
	}

	// If some completed and some failed
	if completedCount > 0 && failedCount > 0 {
		return models.BatchStatusCompletedWithErrors
	}

	// If any failed during migration
	if failedCount > 0 {
		return models.BatchStatusCompletedWithErrors
	}

	// If all dry runs are complete (batch is ready for migration)
	if dryRunCompleteCount == totalRepos {
		return models.BatchStatusReady
	}

	// If some dry runs complete and some failed
	if dryRunCompleteCount > 0 && failedCount > 0 {
		return models.BatchStatusReady
	}

	return models.BatchStatusReady
}

// isTerminalStatus returns true if the status represents a completed state
func isTerminalStatus(status string) bool {
	return status == models.BatchStatusCompleted || status == models.BatchStatusFailed || status == models.BatchStatusCompletedWithErrors || status == models.BatchStatusCancelled
}
