package storage

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SetupStatus represents the server setup completion status
type SetupStatus struct {
	ID             int        `gorm:"primaryKey;check:id=1" json:"id"`
	SetupCompleted bool       `gorm:"not null;default:false" json:"setup_completed"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for SetupStatus
func (SetupStatus) TableName() string {
	return "setup_status"
}

// GetSetupStatus retrieves the current setup status from the database
func (d *Database) GetSetupStatus() (*SetupStatus, error) {
	var status SetupStatus
	err := d.db.Where("id = ?", 1).First(&status).Error

	if err == gorm.ErrRecordNotFound {
		// If no record exists, return default status (not completed)
		return &SetupStatus{
			ID:             1,
			SetupCompleted: false,
			UpdatedAt:      time.Now(),
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get setup status: %w", err)
	}

	return &status, nil
}

// MarkSetupComplete marks the setup as completed in the database
func (d *Database) MarkSetupComplete() error {
	now := time.Now()
	status := SetupStatus{
		ID:             1,
		SetupCompleted: true,
		CompletedAt:    &now,
		UpdatedAt:      now,
	}

	// Use Save which does an upsert - creates if not exists, updates if exists
	result := d.db.Save(&status)

	if result.Error != nil {
		return fmt.Errorf("failed to mark setup complete: %w", result.Error)
	}

	return nil
}

// ResetSetup resets the setup status to allow re-running setup
func (d *Database) ResetSetup() error {
	now := time.Now()
	status := SetupStatus{
		ID:             1,
		SetupCompleted: false,
		CompletedAt:    nil,
		UpdatedAt:      now,
	}

	// Use Save which does an upsert - creates if not exists, updates if exists
	result := d.db.Save(&status)

	if result.Error != nil {
		return fmt.Errorf("failed to reset setup: %w", result.Error)
	}

	return nil
}
