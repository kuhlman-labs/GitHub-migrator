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

// UpdateSourceRepositoryCount updates the cached repository count and last sync time for a source
func (d *Database) UpdateSourceRepositoryCount(ctx context.Context, id int64) error {
	// Count repositories with this source_id
	var count int64
	err := d.db.WithContext(ctx).Model(&models.Repository{}).Where("source_id = ?", id).Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to count repositories: %w", err)
	}

	now := time.Now()
	result := d.db.WithContext(ctx).Model(&models.Source{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"repository_count": count,
			"last_sync_at":     now,
			"updated_at":       now,
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

// SourceDeletionPreview contains counts of all data that will be deleted when a source is deleted
type SourceDeletionPreview struct {
	SourceID              int64  `json:"source_id"`
	SourceName            string `json:"source_name"`
	RepositoryCount       int64  `json:"repository_count"`
	MigrationHistoryCount int64  `json:"migration_history_count"`
	MigrationLogCount     int64  `json:"migration_log_count"`
	DependencyCount       int64  `json:"dependency_count"`
	TeamRepositoryCount   int64  `json:"team_repository_count"`
	BatchRepositoryCount  int64  `json:"batch_repository_count"`
	TeamCount             int64  `json:"team_count"`
	UserCount             int64  `json:"user_count"`
	UserMappingCount      int64  `json:"user_mapping_count"`
	TeamMappingCount      int64  `json:"team_mapping_count"`
	TotalAffectedRecords  int64  `json:"total_affected_records"`
}

// GetSourceDeletionPreview returns counts of all data that would be deleted if the source is deleted
func (d *Database) GetSourceDeletionPreview(ctx context.Context, sourceID int64) (*SourceDeletionPreview, error) {
	preview := &SourceDeletionPreview{SourceID: sourceID}

	// Get the source name
	var source models.Source
	if err := d.db.WithContext(ctx).Select("name").First(&source, sourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("source not found")
		}
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	preview.SourceName = source.Name

	// Get repository IDs for this source (needed for cascading counts)
	var repoIDs []int64
	if err := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("source_id = ?", sourceID).
		Pluck("id", &repoIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get repository IDs: %w", err)
	}
	preview.RepositoryCount = int64(len(repoIDs))

	// Count data linked through repositories
	if len(repoIDs) > 0 {
		// Migration history
		if err := d.db.WithContext(ctx).Model(&models.MigrationHistory{}).
			Where("repository_id IN ?", repoIDs).
			Count(&preview.MigrationHistoryCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count migration history: %w", err)
		}

		// Migration logs
		if err := d.db.WithContext(ctx).Model(&models.MigrationLog{}).
			Where("repository_id IN ?", repoIDs).
			Count(&preview.MigrationLogCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count migration logs: %w", err)
		}

		// Repository dependencies
		if err := d.db.WithContext(ctx).Model(&models.RepositoryDependency{}).
			Where("repository_id IN ?", repoIDs).
			Count(&preview.DependencyCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count dependencies: %w", err)
		}

		// Team-repository associations
		if err := d.db.WithContext(ctx).Model(&models.GitHubTeamRepository{}).
			Where("repository_id IN ?", repoIDs).
			Count(&preview.TeamRepositoryCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count team repositories: %w", err)
		}

		// Count repositories in batches (batch_id is stored in repository, not a separate table)
		if err := d.db.WithContext(ctx).Model(&models.Repository{}).
			Where("id IN ? AND batch_id IS NOT NULL", repoIDs).
			Count(&preview.BatchRepositoryCount).Error; err != nil {
			return nil, fmt.Errorf("failed to count batch repositories: %w", err)
		}
	}

	// Count data directly linked to source
	// Teams
	if err := d.db.WithContext(ctx).Model(&models.GitHubTeam{}).
		Where("source_id = ?", sourceID).
		Count(&preview.TeamCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count teams: %w", err)
	}

	// Users
	if err := d.db.WithContext(ctx).Model(&models.GitHubUser{}).
		Where("source_id = ?", sourceID).
		Count(&preview.UserCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// User mappings
	if err := d.db.WithContext(ctx).Model(&models.UserMapping{}).
		Where("source_id = ?", sourceID).
		Count(&preview.UserMappingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count user mappings: %w", err)
	}

	// Team mappings
	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_id = ?", sourceID).
		Count(&preview.TeamMappingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count team mappings: %w", err)
	}

	// Calculate total
	preview.TotalAffectedRecords = preview.RepositoryCount +
		preview.MigrationHistoryCount +
		preview.MigrationLogCount +
		preview.DependencyCount +
		preview.TeamRepositoryCount +
		preview.BatchRepositoryCount +
		preview.TeamCount +
		preview.UserCount +
		preview.UserMappingCount +
		preview.TeamMappingCount

	return preview, nil
}

// DeleteSourceCascade deletes a source and all related data in a transaction
// This includes: repositories, migration history/logs, dependencies, team associations,
// batch associations, teams, users, user mappings, and team mappings
func (d *Database) DeleteSourceCascade(ctx context.Context, sourceID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, get all repository IDs for this source
		var repoIDs []int64
		if err := tx.Model(&models.Repository{}).
			Where("source_id = ?", sourceID).
			Pluck("id", &repoIDs).Error; err != nil {
			return fmt.Errorf("failed to get repository IDs: %w", err)
		}

		// Delete data linked through repositories
		if len(repoIDs) > 0 {
			// Delete migration logs first (references migration_history)
			if err := tx.Where("repository_id IN ?", repoIDs).
				Delete(&models.MigrationLog{}).Error; err != nil {
				return fmt.Errorf("failed to delete migration logs: %w", err)
			}

			// Delete migration history
			if err := tx.Where("repository_id IN ?", repoIDs).
				Delete(&models.MigrationHistory{}).Error; err != nil {
				return fmt.Errorf("failed to delete migration history: %w", err)
			}

			// Delete repository dependencies
			if err := tx.Where("repository_id IN ?", repoIDs).
				Delete(&models.RepositoryDependency{}).Error; err != nil {
				return fmt.Errorf("failed to delete dependencies: %w", err)
			}

			// Delete team-repository associations
			if err := tx.Where("repository_id IN ?", repoIDs).
				Delete(&models.GitHubTeamRepository{}).Error; err != nil {
				return fmt.Errorf("failed to delete team repositories: %w", err)
			}

			// Delete repositories (batch_id is a FK on the repository, so deleting the repo handles it)
			if err := tx.Where("source_id = ?", sourceID).
				Delete(&models.Repository{}).Error; err != nil {
				return fmt.Errorf("failed to delete repositories: %w", err)
			}
		}

		// Delete data directly linked to source
		// Delete team members first (references github_teams)
		var teamIDs []int64
		if err := tx.Model(&models.GitHubTeam{}).
			Where("source_id = ?", sourceID).
			Pluck("id", &teamIDs).Error; err != nil {
			return fmt.Errorf("failed to get team IDs: %w", err)
		}

		if len(teamIDs) > 0 {
			if err := tx.Where("team_id IN ?", teamIDs).
				Delete(&models.GitHubTeamMember{}).Error; err != nil {
				return fmt.Errorf("failed to delete team members: %w", err)
			}
		}

		// Delete teams
		if err := tx.Where("source_id = ?", sourceID).
			Delete(&models.GitHubTeam{}).Error; err != nil {
			return fmt.Errorf("failed to delete teams: %w", err)
		}

		// Delete users
		if err := tx.Where("source_id = ?", sourceID).
			Delete(&models.GitHubUser{}).Error; err != nil {
			return fmt.Errorf("failed to delete users: %w", err)
		}

		// Delete user mappings
		if err := tx.Where("source_id = ?", sourceID).
			Delete(&models.UserMapping{}).Error; err != nil {
			return fmt.Errorf("failed to delete user mappings: %w", err)
		}

		// Delete team mappings
		if err := tx.Where("source_id = ?", sourceID).
			Delete(&models.TeamMapping{}).Error; err != nil {
			return fmt.Errorf("failed to delete team mappings: %w", err)
		}

		// Finally, delete the source itself
		result := tx.Delete(&models.Source{}, sourceID)
		if result.Error != nil {
			return fmt.Errorf("failed to delete source: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("source not found")
		}

		return nil
	})
}
