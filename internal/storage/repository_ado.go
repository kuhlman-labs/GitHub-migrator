package storage

import (
	"context"
	"fmt"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm/clause"
)

// GetADOProjects retrieves all ADO projects, optionally filtered by organization
func (db *Database) GetADOProjects(ctx context.Context, organization string) ([]models.ADOProject, error) {
	var projects []models.ADOProject
	query := db.db.WithContext(ctx)

	if organization != "" {
		query = query.Where("organization = ?", organization)
	}

	err := query.Order("organization ASC, name ASC").Find(&projects).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get ADO projects: %w", err)
	}

	return projects, nil
}

// GetADOProjectsFiltered retrieves ADO projects, optionally filtered by organization and source_id
// This function queries the repositories table to find distinct ado_project values
// that belong to the specified source, supporting multi-source environments.
func (db *Database) GetADOProjectsFiltered(ctx context.Context, organization string, sourceID *int64) ([]models.ADOProject, error) {
	// Query repositories to get distinct ADO projects for the specified source
	query := db.db.WithContext(ctx).
		Model(&models.Repository{}).
		Select("DISTINCT ado_project as name, SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as organization").
		Where("ado_project IS NOT NULL AND ado_project != ''")

	if organization != "" {
		query = query.Where("full_name LIKE ?", organization+"/%")
	}

	if sourceID != nil {
		query = query.Where("source_id = ?", *sourceID)
	}

	var results []struct {
		Name         string
		Organization string
	}
	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get ADO projects filtered: %w", err)
	}

	// Convert to ADOProject models
	projects := make([]models.ADOProject, 0, len(results))
	for _, r := range results {
		projects = append(projects, models.ADOProject{
			Name:         r.Name,
			Organization: r.Organization,
		})
	}

	return projects, nil
}

// GetADOProject retrieves a specific ADO project by organization and name
func (db *Database) GetADOProject(ctx context.Context, organization, projectName string) (*models.ADOProject, error) {
	var project models.ADOProject
	err := db.db.WithContext(ctx).
		Where("organization = ? AND name = ?", organization, projectName).
		First(&project).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get ADO project: %w", err)
	}

	return &project, nil
}

// GetRepositoriesByADOProject retrieves all repositories for a specific ADO project
func (db *Database) GetRepositoriesByADOProject(ctx context.Context, organization, projectName string) ([]models.Repository, error) {
	var repos []models.Repository
	err := db.db.WithContext(ctx).
		Where("ado_project = ?", projectName).
		Order("full_name ASC").
		Find(&repos).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by ADO project: %w", err)
	}

	return repos, nil
}

// CountRepositoriesByADOProject counts repositories for a specific ADO project within an organization
// Filters by both ado_project AND organization (via full_name prefix) to handle
// duplicate project names across different ADO organizations
func (db *Database) CountRepositoriesByADOProject(ctx context.Context, organization, projectName string) (int, error) {
	var count int64
	query := db.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("ado_project = ?", projectName)

	// Also filter by organization to handle duplicate project names across different ADO orgs
	if organization != "" {
		query = query.Where("full_name LIKE ?", organization+"/%")
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count repositories by ADO project: %w", err)
	}

	return int(count), nil
}

// CountRepositoriesByADOProjectFiltered counts repositories for a specific ADO project,
// with optional source_id filtering for multi-source environments.
func (db *Database) CountRepositoriesByADOProjectFiltered(ctx context.Context, organization, projectName string, sourceID *int64) (int, error) {
	var count int64
	query := db.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("ado_project = ?", projectName)

	// Filter by organization to handle duplicate project names across different ADO orgs
	if organization != "" {
		query = query.Where("full_name LIKE ?", organization+"/%")
	}

	// Filter by source_id for multi-source support
	if sourceID != nil {
		query = query.Where("source_id = ?", *sourceID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count repositories by ADO project filtered: %w", err)
	}

	return int(count), nil
}

// CountRepositoriesByADOOrganization counts all repositories for an ADO organization
func (db *Database) CountRepositoriesByADOOrganization(ctx context.Context, organization string) (int, error) {
	var count int64
	// For ADO repos, the organization is the first part of full_name (e.g., "org/project/repo")
	// We filter by ado_project being NOT NULL to identify ADO repos
	err := db.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("full_name LIKE ? AND ado_project IS NOT NULL", organization+"/%").
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count repositories by ADO organization: %w", err)
	}

	return int(count), nil
}

// CountRepositoriesByADOProjects counts repositories across multiple ADO projects
func (db *Database) CountRepositoriesByADOProjects(ctx context.Context, organization string, projects []string) (int, error) {
	var count int64
	// For ADO repos, filter by full_name prefix and project names
	err := db.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("full_name LIKE ? AND ado_project IN ?", organization+"/%", projects).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count repositories by ADO projects: %w", err)
	}

	return int(count), nil
}

// CountTFVCRepositories counts TFVC (non-Git) repositories for an ADO organization
// If organization is empty, counts all TFVC repositories
func (db *Database) CountTFVCRepositories(ctx context.Context, organization string) (int, error) {
	var count int64
	query := db.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("ado_is_git = ? AND ado_project IS NOT NULL", false)

	if organization != "" {
		// Filter by full_name prefix for ADO organization
		query = query.Where("full_name LIKE ?", organization+"/%")
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count TFVC repositories: %w", err)
	}

	return int(count), nil
}

// SaveADOProject saves or updates an ADO project
func (db *Database) SaveADOProject(ctx context.Context, project *models.ADOProject) error {
	// Use Clauses(clause.OnConflict) to handle upsert based on unique constraint
	// This will update all fields if a conflict occurs on (organization, name)
	err := db.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization"}, {Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"description", "repository_count", "state", "visibility", "updated_at"}),
		}).
		Create(project).Error

	if err != nil {
		return fmt.Errorf("failed to save ADO project: %w", err)
	}

	return nil
}

// DeleteADOProject deletes an ADO project
func (db *Database) DeleteADOProject(ctx context.Context, organization, projectName string) error {
	err := db.db.WithContext(ctx).
		Where("organization = ? AND name = ?", organization, projectName).
		Delete(&models.ADOProject{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete ADO project: %w", err)
	}

	return nil
}
