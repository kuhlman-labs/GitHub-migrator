package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// ErrDiscoveryInProgress is returned when attempting to start a discovery
// while another discovery is already in progress
var ErrDiscoveryInProgress = errors.New("discovery already in progress")

// CreateDiscoveryProgress creates a new discovery progress record
// It will delete any existing completed record first to maintain max 2 records
func (d *Database) CreateDiscoveryProgress(progress *models.DiscoveryProgress) error {
	// Check if there's already an in_progress discovery
	active, err := d.GetActiveDiscoveryProgress()
	if err != nil {
		return fmt.Errorf("failed to check for active discovery: %w", err)
	}
	if active != nil {
		return fmt.Errorf("%w (id: %d, target: %s)", ErrDiscoveryInProgress, active.ID, active.Target)
	}

	// Delete any existing completed records to maintain max 2 records
	if err := d.DeleteCompletedDiscoveryProgress(); err != nil {
		return fmt.Errorf("failed to clean up old discovery records: %w", err)
	}

	// Set defaults
	progress.Status = models.DiscoveryStatusInProgress
	progress.StartedAt = time.Now()
	progress.Phase = models.PhaseListingRepos

	if err := d.db.Create(progress).Error; err != nil {
		return fmt.Errorf("failed to create discovery progress: %w", err)
	}

	return nil
}

// UpdateDiscoveryProgress updates an existing discovery progress record
func (d *Database) UpdateDiscoveryProgress(progress *models.DiscoveryProgress) error {
	if progress.ID == 0 {
		return fmt.Errorf("discovery progress ID is required for update")
	}

	if err := d.db.Save(progress).Error; err != nil {
		return fmt.Errorf("failed to update discovery progress: %w", err)
	}

	return nil
}

// GetActiveDiscoveryProgress retrieves the currently active (in_progress) discovery
// Returns nil if no discovery is currently in progress
func (d *Database) GetActiveDiscoveryProgress() (*models.DiscoveryProgress, error) {
	var progress models.DiscoveryProgress
	err := d.db.Where("status = ?", models.DiscoveryStatusInProgress).First(&progress).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get active discovery progress: %w", err)
	}

	return &progress, nil
}

// GetLatestDiscoveryProgress retrieves the most recent discovery progress record
// This will return the active one if exists, otherwise the most recent completed one
func (d *Database) GetLatestDiscoveryProgress() (*models.DiscoveryProgress, error) {
	// First try to get an active discovery
	active, err := d.GetActiveDiscoveryProgress()
	if err != nil {
		return nil, err
	}
	if active != nil {
		return active, nil
	}

	// Otherwise get the most recent by ID (since IDs are auto-incrementing)
	var progress models.DiscoveryProgress
	err = d.db.Order("id DESC").First(&progress).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest discovery progress: %w", err)
	}

	return &progress, nil
}

// DeleteCompletedDiscoveryProgress deletes all completed discovery progress records
// This is called before creating a new discovery to maintain max 2 records
func (d *Database) DeleteCompletedDiscoveryProgress() error {
	result := d.db.Where("status IN ?", []string{
		models.DiscoveryStatusCompleted,
		models.DiscoveryStatusFailed,
		models.DiscoveryStatusCancelled,
	}).Delete(&models.DiscoveryProgress{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete completed discovery progress: %w", result.Error)
	}

	return nil
}

// MarkDiscoveryComplete marks a discovery as completed
func (d *Database) MarkDiscoveryComplete(id int64) error {
	now := time.Now()
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       models.DiscoveryStatusCompleted,
			"completed_at": now,
			"phase":        models.PhaseCompleted,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark discovery complete: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("discovery progress with id %d not found", id)
	}

	return nil
}

// MarkDiscoveryFailed marks a discovery as failed with an error message
func (d *Database) MarkDiscoveryFailed(id int64, errorMsg string) error {
	now := time.Now()
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       models.DiscoveryStatusFailed,
			"completed_at": now,
			"last_error":   errorMsg,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark discovery failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("discovery progress with id %d not found", id)
	}

	return nil
}

// MarkDiscoveryCancelled marks a discovery as cancelled
func (d *Database) MarkDiscoveryCancelled(id int64) error {
	now := time.Now()
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       models.DiscoveryStatusCancelled,
			"completed_at": now,
			"phase":        models.PhaseCancelling,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark discovery cancelled: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("discovery progress with id %d not found", id)
	}

	return nil
}

// IncrementDiscoveryError increments the error count and updates last error
func (d *Database) IncrementDiscoveryError(id int64, errorMsg string) error {
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"error_count": gorm.Expr("error_count + 1"),
			"last_error":  errorMsg,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to increment discovery error: %w", result.Error)
	}

	return nil
}

// UpdateDiscoveryPhase updates the current phase of the discovery
func (d *Database) UpdateDiscoveryPhase(id int64, phase string) error {
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Update("phase", phase)

	if result.Error != nil {
		return fmt.Errorf("failed to update discovery phase: %w", result.Error)
	}

	return nil
}

// UpdateDiscoveryOrgProgress updates the organization-level progress
func (d *Database) UpdateDiscoveryOrgProgress(id int64, currentOrg string, processedOrgs, totalOrgs int) error {
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"current_org":    currentOrg,
			"processed_orgs": processedOrgs,
			"total_orgs":     totalOrgs,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update discovery org progress: %w", result.Error)
	}

	return nil
}

// UpdateDiscoveryRepoProgress updates the repository-level progress
func (d *Database) UpdateDiscoveryRepoProgress(id int64, processedRepos, totalRepos int) error {
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"processed_repos": processedRepos,
			"total_repos":     totalRepos,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update discovery repo progress: %w", result.Error)
	}

	return nil
}

// IncrementProcessedRepos increments the processed repos counter by the given amount
func (d *Database) IncrementProcessedRepos(id int64, count int) error {
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("id = ?", id).
		Update("processed_repos", gorm.Expr("processed_repos + ?", count))

	if result.Error != nil {
		return fmt.Errorf("failed to increment processed repos: %w", result.Error)
	}

	return nil
}

// ForceResetDiscovery forcibly marks any stuck in_progress discovery as cancelled.
// This is used to recover from scenarios where the server crashed or restarted
// while a discovery was in progress, leaving the database in an inconsistent state.
// Returns the number of records affected and any error.
func (d *Database) ForceResetDiscovery() (int64, error) {
	now := time.Now()
	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("status = ?", models.DiscoveryStatusInProgress).
		Updates(map[string]any{
			"status":       models.DiscoveryStatusCancelled,
			"completed_at": now,
			"phase":        models.PhaseCancelling,
			"last_error":   "Force reset: discovery was stuck in progress state",
		})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to force reset discovery: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// RecoverStuckDiscoveries checks for any stuck discoveries on startup and resets them.
// A discovery is considered stuck if it's been in_progress for more than the timeout duration.
func (d *Database) RecoverStuckDiscoveries(timeout time.Duration) (int64, error) {
	now := time.Now()
	cutoff := now.Add(-timeout)

	result := d.db.Model(&models.DiscoveryProgress{}).
		Where("status = ? AND started_at < ?", models.DiscoveryStatusInProgress, cutoff).
		Updates(map[string]any{
			"status":       models.DiscoveryStatusCancelled,
			"completed_at": now,
			"phase":        models.PhaseCancelling,
			"last_error":   fmt.Sprintf("Auto-recovered: discovery was stuck for more than %s", timeout),
		})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to recover stuck discoveries: %w", result.Error)
	}

	return result.RowsAffected, nil
}
