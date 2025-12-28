package storage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

const (
	// Batch status constants
	batchStatusPending    = "pending"
	batchStatusReady      = "ready"
	batchStatusInProgress = "in_progress"
)

// GetBatch retrieves a batch by ID using GORM
func (d *Database) GetBatch(ctx context.Context, id int64) (*models.Batch, error) {
	var batch models.Batch
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&batch).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	// Ensure repository_count is accurate by querying the actual count
	// This prevents data inconsistency issues where the count may be stale
	var count int64
	if err := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("batch_id = ?", batch.ID).
		Count(&count).Error; err == nil {
		batch.RepositoryCount = int(count)
	}

	return &batch, nil
}

// UpdateBatch updates a batch using GORM
func (d *Database) UpdateBatch(ctx context.Context, batch *models.Batch) error {
	result := d.db.WithContext(ctx).Save(batch)
	return result.Error
}

// ListBatches retrieves all batches using GORM
func (d *Database) ListBatches(ctx context.Context) ([]*models.Batch, error) {
	var batches []*models.Batch
	err := d.db.WithContext(ctx).Order("created_at DESC").Find(&batches).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	// Ensure repository_count is accurate by querying the actual count
	// This prevents data inconsistency issues where the count may be stale
	for _, batch := range batches {
		var count int64
		if err := d.db.WithContext(ctx).Model(&models.Repository{}).
			Where("batch_id = ?", batch.ID).
			Count(&count).Error; err == nil {
			batch.RepositoryCount = int(count)
		}
	}

	return batches, nil
}

// CreateBatch creates a new batch using GORM
func (d *Database) CreateBatch(ctx context.Context, batch *models.Batch) error {
	// Set default migration API if not specified
	if batch.MigrationAPI == "" {
		batch.MigrationAPI = models.MigrationAPIGEI
	}

	result := d.db.WithContext(ctx).Create(batch)
	if result.Error != nil {
		return fmt.Errorf("failed to create batch: %w", result.Error)
	}

	return nil
}

// DeleteBatch deletes a batch and clears batch_id from all associated repositories using GORM
func (d *Database) DeleteBatch(ctx context.Context, batchID int64) error {
	// Use GORM transaction
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear batch_id from all repositories in this batch
		now := time.Now().UTC()
		result := tx.Model(&models.Repository{}).
			Where("batch_id = ?", batchID).
			Updates(map[string]any{
				"batch_id":   nil,
				"updated_at": now,
			})
		if result.Error != nil {
			return fmt.Errorf("failed to clear batch from repositories: %w", result.Error)
		}

		// Delete the batch
		result = tx.Delete(&models.Batch{}, batchID)
		if result.Error != nil {
			return fmt.Errorf("failed to delete batch: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("batch not found")
		}

		return nil
	})
}

// UpdateBatchDryRunTimestamp updates the last_dry_run_at timestamp for a batch using GORM
func (d *Database) UpdateBatchDryRunTimestamp(ctx context.Context, batchID int64) error {
	result := d.db.WithContext(ctx).Model(&models.Batch{}).
		Where("id = ?", batchID).
		Update("last_dry_run_at", time.Now().UTC())

	return result.Error
}

// UpdateBatchMigrationAttemptTimestamp updates the last_migration_attempt_at timestamp for a batch using GORM
func (d *Database) UpdateBatchMigrationAttemptTimestamp(ctx context.Context, batchID int64) error {
	result := d.db.WithContext(ctx).Model(&models.Batch{}).
		Where("id = ?", batchID).
		Update("last_migration_attempt_at", time.Now().UTC())

	return result.Error
}

// UpdateBatchProgress updates batch status and operational timestamps without affecting user-configured fields using GORM
// This preserves scheduled_at and other user-set fields while updating execution state
func (d *Database) UpdateBatchProgress(ctx context.Context, batchID int64, status string, startedAt, lastDryRunAt, lastMigrationAttemptAt *time.Time) error {
	updates := map[string]any{
		"status": status,
	}

	// Only update timestamps if provided (COALESCE behavior)
	if startedAt != nil {
		updates["started_at"] = startedAt
	}
	if lastDryRunAt != nil {
		updates["last_dry_run_at"] = lastDryRunAt
	}
	if lastMigrationAttemptAt != nil {
		updates["last_migration_attempt_at"] = lastMigrationAttemptAt
	}

	result := d.db.WithContext(ctx).Model(&models.Batch{}).
		Where("id = ?", batchID).
		Updates(updates)

	return result.Error
}

// AddRepositoriesToBatch assigns multiple repositories to a batch
//
//nolint:dupl // Similar to RemoveRepositoriesFromBatch but performs different operations
func (d *Database) AddRepositoriesToBatch(ctx context.Context, batchID int64, repoIDs []int64) error {
	if len(repoIDs) == 0 {
		return nil
	}

	// Use GORM to update batch_id for specified repositories
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("id IN ?", repoIDs).
		Updates(map[string]any{
			"batch_id":   batchID,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to add repositories to batch: %w", result.Error)
	}

	// Update batch repository count and status
	if result.RowsAffected > 0 {
		if err := d.updateBatchRepositoryCount(ctx, batchID); err != nil {
			return err
		}
		// Recalculate batch status based on repository dry run readiness
		if err := d.UpdateBatchStatus(ctx, batchID); err != nil {
			return err
		}
	}

	return nil
}

// RemoveRepositoriesFromBatch removes repositories from a batch using GORM
func (d *Database) RemoveRepositoriesFromBatch(ctx context.Context, batchID int64, repoIDs []int64) error {
	if len(repoIDs) == 0 {
		return nil
	}

	// Use GORM to clear batch_id for specified repositories
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("batch_id = ? AND id IN ?", batchID, repoIDs).
		Updates(map[string]any{
			"batch_id":   nil,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to remove repositories from batch: %w", result.Error)
	}

	// Update batch repository count and status
	rowsAffected := result.RowsAffected
	if rowsAffected > 0 {
		if err := d.updateBatchRepositoryCount(ctx, batchID); err != nil {
			return err
		}
		// Recalculate batch status after removal
		if err := d.UpdateBatchStatus(ctx, batchID); err != nil {
			return err
		}
	}

	return nil
}

// updateBatchRepositoryCount updates the repository count for a batch
func (d *Database) updateBatchRepositoryCount(ctx context.Context, batchID int64) error {
	query := `
		UPDATE batches 
		SET repository_count = (
			SELECT COUNT(*) FROM repositories WHERE batch_id = ?
		)
		WHERE id = ?
	`

	// Use GORM Raw() for complex query with subquery
	err := d.db.WithContext(ctx).Exec(query, batchID, batchID).Error
	if err != nil {
		return fmt.Errorf("failed to update batch repository count: %w", err)
	}

	return nil
}

// UpdateBatchStatus recalculates and updates the batch status based on repository statuses
// Batch is 'ready' only if ALL repositories have completed dry runs
// Batch is 'pending' if ANY repository hasn't completed a dry run
func (d *Database) UpdateBatchStatus(ctx context.Context, batchID int64) error {
	// Get all repositories in the batch
	repos, err := d.ListRepositories(ctx, map[string]any{
		"batch_id": batchID,
	})
	if err != nil {
		return fmt.Errorf("failed to list batch repositories: %w", err)
	}

	// Get current batch to check if it's in a terminal or active state
	batch, err := d.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch not found")
	}

	// Don't update status if batch is in progress or completed
	terminalStates := []string{"in_progress", "completed", "completed_with_errors", "failed", "cancelled"}
	if slices.Contains(terminalStates, batch.Status) {
		// Batch is actively running or finished, don't change status
		return nil
	}

	// Calculate new status based on repository dry run status
	newStatus := calculateBatchReadiness(repos)

	// Only update if status changed
	if newStatus != batch.Status {
		result := d.db.WithContext(ctx).Model(&models.Batch{}).Where("id = ?", batchID).Update("status", newStatus)
		if result.Error != nil {
			return fmt.Errorf("failed to update batch status: %w", result.Error)
		}
	}

	return nil
}

// calculateBatchReadiness determines if a batch should be 'ready' or 'pending'
// based on the dry run status of its repositories
func calculateBatchReadiness(repos []*models.Repository) string {
	if len(repos) == 0 {
		return batchStatusPending
	}

	allDryRunComplete := true
	for _, repo := range repos {
		// Repository needs dry run if it's in any of these states
		needsDryRun := repo.Status == string(models.StatusPending) ||
			repo.Status == string(models.StatusDryRunFailed) ||
			repo.Status == string(models.StatusMigrationFailed) ||
			repo.Status == string(models.StatusRolledBack)

		if needsDryRun {
			allDryRunComplete = false
			break
		}

		// If repo is not dry_run_complete, it also needs dry run
		if repo.Status != string(models.StatusDryRunComplete) {
			allDryRunComplete = false
			break
		}
	}

	if allDryRunComplete {
		return batchStatusReady
	}
	return batchStatusPending
}
