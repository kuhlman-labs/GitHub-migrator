package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// CreateSource creates a new source in the database
func (d *Database) CreateSource(ctx context.Context, source *models.Source) error {
	if err := source.Validate(); err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}

	now := time.Now()
	source.CreatedAt = now
	source.UpdatedAt = now

	result := d.db.WithContext(ctx).Create(source)
	if result.Error != nil {
		return fmt.Errorf("failed to create source: %w", result.Error)
	}

	return nil
}

// GetSource retrieves a source by ID
func (d *Database) GetSource(ctx context.Context, id int64) (*models.Source, error) {
	var source models.Source
	err := d.db.WithContext(ctx).First(&source, id).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return &source, nil
}

// GetSourceByName retrieves a source by its unique name
func (d *Database) GetSourceByName(ctx context.Context, name string) (*models.Source, error) {
	var source models.Source
	err := d.db.WithContext(ctx).Where("name = ?", name).First(&source).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source by name: %w", err)
	}

	return &source, nil
}

// ListSources retrieves all sources, optionally filtered by active status
func (d *Database) ListSources(ctx context.Context) ([]*models.Source, error) {
	var sources []*models.Source
	err := d.db.WithContext(ctx).Order("name ASC").Find(&sources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}

	return sources, nil
}

// ListActiveSources retrieves only active sources
func (d *Database) ListActiveSources(ctx context.Context) ([]*models.Source, error) {
	var sources []*models.Source
	err := d.db.WithContext(ctx).Where("is_active = ?", true).Order("name ASC").Find(&sources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list active sources: %w", err)
	}

	return sources, nil
}

// UpdateSource updates an existing source
func (d *Database) UpdateSource(ctx context.Context, source *models.Source) error {
	if err := source.Validate(); err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}

	source.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Save(source)
	if result.Error != nil {
		return fmt.Errorf("failed to update source: %w", result.Error)
	}

	return nil
}

// DeleteSource deletes a source by ID
// Returns an error if there are repositories associated with this source
func (d *Database) DeleteSource(ctx context.Context, id int64) error {
	// Check if there are repositories using this source
	var count int64
	err := d.db.WithContext(ctx).Model(&models.Repository{}).Where("source_id = ?", id).Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to check repositories: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete source: %d repositories are associated with it", count)
	}

	result := d.db.WithContext(ctx).Delete(&models.Source{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete source: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("source not found")
	}

	return nil
}

// SetSourceActive sets a source's active status
func (d *Database) SetSourceActive(ctx context.Context, id int64, isActive bool) error {
	result := d.db.WithContext(ctx).Model(&models.Source{}).
		Where("id = ?", id).
		Update("is_active", isActive)

	if result.Error != nil {
		return fmt.Errorf("failed to update source active status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("source not found")
	}

	return nil
}

// UpdateSourceLastSync updates the last_sync_at timestamp for a source
func (d *Database) UpdateSourceLastSync(ctx context.Context, id int64) error {
	now := time.Now()
	result := d.db.WithContext(ctx).Model(&models.Source{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_sync_at": now,
			"updated_at":   now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update source last sync: %w", result.Error)
	}

	return nil
}

// UpdateSourceRepositoryCount updates the cached repository count for a source
func (d *Database) UpdateSourceRepositoryCount(ctx context.Context, id int64) error {
	// Count repositories with this source_id
	var count int64
	err := d.db.WithContext(ctx).Model(&models.Repository{}).Where("source_id = ?", id).Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to count repositories: %w", err)
	}

	result := d.db.WithContext(ctx).Model(&models.Source{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"repository_count": count,
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update source repository count: %w", result.Error)
	}

	return nil
}

// GetRepositoriesBySourceID retrieves all repositories for a specific source
func (d *Database) GetRepositoriesBySourceID(ctx context.Context, sourceID int64) ([]*models.Repository, error) {
	var repos []*models.Repository
	err := d.db.WithContext(ctx).Where("source_id = ?", sourceID).Find(&repos).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by source: %w", err)
	}

	return repos, nil
}

// CountRepositoriesBySourceID returns the count of repositories for a source
func (d *Database) CountRepositoriesBySourceID(ctx context.Context, sourceID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&models.Repository{}).Where("source_id = ?", sourceID).Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count repositories by source: %w", err)
	}

	return count, nil
}

// AssignRepositoryToSource assigns a repository to a source
func (d *Database) AssignRepositoryToSource(ctx context.Context, repoID, sourceID int64) error {
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("id = ?", repoID).
		Update("source_id", sourceID)

	if result.Error != nil {
		return fmt.Errorf("failed to assign repository to source: %w", result.Error)
	}

	return nil
}

// GetSourceByType retrieves sources of a specific type (github or azuredevops)
func (d *Database) GetSourcesByType(ctx context.Context, sourceType string) ([]*models.Source, error) {
	var sources []*models.Source
	err := d.db.WithContext(ctx).Where("type = ?", sourceType).Order("name ASC").Find(&sources).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get sources by type: %w", err)
	}

	return sources, nil
}
