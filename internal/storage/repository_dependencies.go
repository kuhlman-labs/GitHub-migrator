package storage

import (
	"context"
	"fmt"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// SaveRepositoryDependencies saves or updates dependencies for a repository using GORM
// This replaces existing dependencies for the given repository
func (d *Database) SaveRepositoryDependencies(ctx context.Context, repoID int64, dependencies []*models.RepositoryDependency) error {
	// Use GORM transaction
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear existing dependencies for this repository
		if err := tx.Where("repository_id = ?", repoID).Delete(&models.RepositoryDependency{}).Error; err != nil {
			return fmt.Errorf("failed to clear existing dependencies: %w", err)
		}

		// Insert new dependencies
		if len(dependencies) > 0 {
			// Set repository_id for all dependencies
			for _, dep := range dependencies {
				dep.RepositoryID = repoID
			}

			// Batch create all dependencies
			if err := tx.Create(dependencies).Error; err != nil {
				return fmt.Errorf("failed to insert dependencies: %w", err)
			}
		}

		return nil
	})
}

// GetRepositoryDependencies retrieves all dependencies for a repository using GORM
func (d *Database) GetRepositoryDependencies(ctx context.Context, repoID int64) ([]*models.RepositoryDependency, error) {
	// Initialize as empty slice instead of nil so JSON serialization returns [] not null
	dependencies := make([]*models.RepositoryDependency, 0)

	err := d.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("dependency_type, dependency_full_name").
		Find(&dependencies).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies: %w", err)
	}

	return dependencies, nil
}

// GetRepositoryDependenciesByFullName retrieves all dependencies for a repository by its full name
func (d *Database) GetRepositoryDependenciesByFullName(ctx context.Context, fullName string) ([]*models.RepositoryDependency, error) {
	// First get the repository ID
	repo, err := d.GetRepository(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found: %s", fullName)
	}

	return d.GetRepositoryDependencies(ctx, repo.ID)
}

// GetDependentRepositories retrieves all repositories that depend on a specific dependency using GORM
// This is useful for batch planning - "what repos depend on X?"
func (d *Database) GetDependentRepositories(ctx context.Context, dependencyFullName string) ([]*models.Repository, error) {
	// Use GORM with join to get repositories with the specified dependency
	var repos []*models.Repository
	err := d.db.WithContext(ctx).
		Joins("INNER JOIN repository_dependencies rd ON repositories.id = rd.repository_id").
		Where("rd.dependency_full_name = ?", dependencyFullName).
		Order("repositories.full_name").
		Find(&repos).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query dependent repositories: %w", err)
	}

	return repos, nil
}

// ClearRepositoryDependencies removes all dependencies for a repository using GORM
// Useful before re-discovery
func (d *Database) ClearRepositoryDependencies(ctx context.Context, repoID int64) error {
	err := d.db.WithContext(ctx).Where("repository_id = ?", repoID).Delete(&models.RepositoryDependency{}).Error
	if err != nil {
		return fmt.Errorf("failed to clear dependencies: %w", err)
	}
	return nil
}

// UpdateLocalDependencyFlags updates the is_local flag for all dependencies using GORM
// based on whether the dependency exists in our database
// This should be run after discovery to properly mark local dependencies
func (d *Database) UpdateLocalDependencyFlags(ctx context.Context) error {
	// Use dialect-specific boolean values
	var query string
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL, DBTypeSQLServer, DBTypeMSSQL:
		// PostgreSQL and SQL Server use TRUE/FALSE for boolean columns
		query = `
			UPDATE repository_dependencies
			SET is_local = CASE
				WHEN dependency_full_name IN (SELECT full_name FROM repositories)
				THEN TRUE
				ELSE FALSE
			END
		`
	default: // SQLite
		// SQLite uses 1/0 for boolean columns
		query = `
			UPDATE repository_dependencies
			SET is_local = CASE
				WHEN dependency_full_name IN (SELECT full_name FROM repositories)
				THEN 1
				ELSE 0
			END
		`
	}

	err := d.db.WithContext(ctx).Exec(query).Error
	if err != nil {
		return fmt.Errorf("failed to update local dependency flags: %w", err)
	}

	return nil
}
