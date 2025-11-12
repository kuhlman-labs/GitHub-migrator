package storage

import (
	"context"
	"fmt"

	"github.com/brettkuhlman/github-migrator/internal/models"
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

// CountRepositoriesByADOProject counts repositories for a specific ADO project
func (db *Database) CountRepositoriesByADOProject(ctx context.Context, organization, projectName string) (int, error) {
	var count int64
	err := db.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("ado_project = ?", projectName).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count repositories by ADO project: %w", err)
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
