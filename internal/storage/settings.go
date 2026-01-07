package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// GetSettings retrieves the application settings (singleton record with id=1)
func (d *Database) GetSettings(ctx context.Context) (*models.Settings, error) {
	var settings models.Settings
	err := d.db.WithContext(ctx).First(&settings, 1).Error

	if err == gorm.ErrRecordNotFound {
		// Initialize with defaults if not found (shouldn't happen after migration)
		settings = models.Settings{
			ID:                            1,
			DestinationBaseURL:            "https://api.github.com",
			MigrationWorkers:              5,
			MigrationPollIntervalSeconds:  30,
			MigrationDestRepoExistsAction: "fail",
			MigrationVisibilityPublic:     "private",
			MigrationVisibilityInternal:   "private",
			AuthEnabled:                   false,
			AuthSessionDurationHours:      24,
			AuthFrontendURL:               "http://localhost:3000",
			CreatedAt:                     time.Now(),
			UpdatedAt:                     time.Now(),
		}
		if err := d.db.WithContext(ctx).Create(&settings).Error; err != nil {
			return nil, fmt.Errorf("failed to create default settings: %w", err)
		}
		return &settings, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return &settings, nil
}

// UpdateSettings updates the application settings
func (d *Database) UpdateSettings(ctx context.Context, req *models.UpdateSettingsRequest) (*models.Settings, error) {
	// Get current settings
	settings, err := d.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	// Apply updates from request
	settings.ApplyUpdates(req)
	settings.UpdatedAt = time.Now()

	// Save updates
	result := d.db.WithContext(ctx).Save(settings)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update settings: %w", result.Error)
	}

	return settings, nil
}

// UpdateDestinationSettings updates only the destination-related settings
func (d *Database) UpdateDestinationSettings(ctx context.Context, baseURL, token string, appID, appInstallationID *int64, appPrivateKey *string) error {
	settings, err := d.GetSettings(ctx)
	if err != nil {
		return err
	}

	settings.DestinationBaseURL = baseURL
	settings.DestinationToken = &token
	settings.DestinationAppID = appID
	settings.DestinationAppPrivateKey = appPrivateKey
	settings.DestinationAppInstallationID = appInstallationID
	settings.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Save(settings)
	if result.Error != nil {
		return fmt.Errorf("failed to update destination settings: %w", result.Error)
	}

	return nil
}

// UpdateMigrationSettings updates only the migration-related settings
func (d *Database) UpdateMigrationSettings(ctx context.Context, workers, pollInterval int, destRepoAction, visPublic, visInternal string) error {
	settings, err := d.GetSettings(ctx)
	if err != nil {
		return err
	}

	settings.MigrationWorkers = workers
	settings.MigrationPollIntervalSeconds = pollInterval
	settings.MigrationDestRepoExistsAction = destRepoAction
	settings.MigrationVisibilityPublic = visPublic
	settings.MigrationVisibilityInternal = visInternal
	settings.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Save(settings)
	if result.Error != nil {
		return fmt.Errorf("failed to update migration settings: %w", result.Error)
	}

	return nil
}

// UpdateAuthSettings updates only the auth-related settings
func (d *Database) UpdateAuthSettings(ctx context.Context, enabled bool, sessionSecret string, sessionDuration int, callbackURL, frontendURL string) error {
	settings, err := d.GetSettings(ctx)
	if err != nil {
		return err
	}

	settings.AuthEnabled = enabled
	if sessionSecret != "" {
		settings.AuthSessionSecret = &sessionSecret
	}
	settings.AuthSessionDurationHours = sessionDuration
	if callbackURL != "" {
		settings.AuthCallbackURL = &callbackURL
	}
	settings.AuthFrontendURL = frontendURL
	settings.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Save(settings)
	if result.Error != nil {
		return fmt.Errorf("failed to update auth settings: %w", result.Error)
	}

	return nil
}

// IsDestinationConfigured returns true if destination credentials are set
func (d *Database) IsDestinationConfigured(ctx context.Context) (bool, error) {
	settings, err := d.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings.HasDestination(), nil
}
