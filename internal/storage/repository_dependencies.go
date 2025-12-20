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
// For local dependencies, it enriches the dependency_url with the actual source_url from the repositories table
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

	// Enrich local dependencies with actual source URLs
	if err := d.enrichDependencyURLs(ctx, dependencies); err != nil {
		// Log error but don't fail the request
		// Dependencies will just have their stored URLs
		return dependencies, nil
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

// enrichDependencyURLs updates dependency URLs for local dependencies with actual source URLs
func (d *Database) enrichDependencyURLs(ctx context.Context, dependencies []*models.RepositoryDependency) error {
	if len(dependencies) == 0 {
		return nil
	}

	// Collect all local dependency names
	localDepNames := make([]string, 0)
	for _, dep := range dependencies {
		if dep.IsLocal {
			localDepNames = append(localDepNames, dep.DependencyFullName)
		}
	}

	if len(localDepNames) == 0 {
		return nil
	}

	// Fetch source URLs for all local dependencies in one query
	type RepoURL struct {
		FullName  string
		SourceURL string
	}
	var repoURLs []RepoURL
	err := d.db.WithContext(ctx).
		Model(&models.Repository{}).
		Select("full_name, source_url").
		Where("full_name IN ?", localDepNames).
		Find(&repoURLs).Error

	if err != nil {
		return fmt.Errorf("failed to fetch source URLs: %w", err)
	}

	// Create a map for quick lookup
	urlMap := make(map[string]string)
	for _, repoURL := range repoURLs {
		urlMap[repoURL.FullName] = repoURL.SourceURL
	}

	// Update dependency URLs
	for _, dep := range dependencies {
		if dep.IsLocal {
			if sourceURL, exists := urlMap[dep.DependencyFullName]; exists {
				dep.DependencyURL = sourceURL
			}
		}
	}

	return nil
}

// UpdateLocalDependencyFlags updates the is_local flag for all dependencies using GORM
// based on whether the dependency exists in our database
// This should be run after discovery to properly mark local dependencies
func (d *Database) UpdateLocalDependencyFlags(ctx context.Context) error {
	// Use dialect-specific boolean values via DialectDialer interface
	boolTrue := d.dialect.BooleanTrue()
	boolFalse := d.dialect.BooleanFalse()

	query := fmt.Sprintf(`
		UPDATE repository_dependencies
		SET is_local = CASE
			WHEN dependency_full_name IN (SELECT full_name FROM repositories)
			THEN %s
			ELSE %s
		END
	`, boolTrue, boolFalse)

	err := d.db.WithContext(ctx).Exec(query).Error
	if err != nil {
		return fmt.Errorf("failed to update local dependency flags: %w", err)
	}

	return nil
}

// DependencyPair represents a local dependency relationship between two repositories
type DependencyPair struct {
	SourceRepo     string
	TargetRepo     string
	DependencyType string
	DependencyURL  string // URL of the target repo (the dependency)
	SourceRepoURL  string // URL of the source repo (the repo that has the dependency)
}

// GetAllLocalDependencyPairs returns all local dependency relationships for the dependency graph
// It returns pairs where both the source and target repositories exist in the database
// Optionally filters by dependency types
func (d *Database) GetAllLocalDependencyPairs(ctx context.Context, dependencyTypes []string) ([]DependencyPair, error) {
	query := d.db.WithContext(ctx).
		Model(&models.RepositoryDependency{}).
		Select("r.full_name as source_repo, rd.dependency_full_name as target_repo, rd.dependency_type, rd.dependency_url, r.source_url as source_repo_url").
		Joins("JOIN repositories r ON rd.repository_id = r.id").
		Table("repository_dependencies rd").
		Where("rd.is_local = ?", true)

	// Add dependency type filter if specified
	if len(dependencyTypes) > 0 {
		query = query.Where("rd.dependency_type IN ?", dependencyTypes)
	}

	var results []DependencyPair
	err := query.Order("r.full_name, rd.dependency_full_name").Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get local dependency pairs: %w", err)
	}

	return results, nil
}
