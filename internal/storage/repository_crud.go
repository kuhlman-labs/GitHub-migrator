package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RepositoryFilter contains optional filters for repository queries
type RepositoryFilter struct {
	Status          *string // Filter by status
	BatchID         *int64  // Filter by batch ID
	Owner           *string // Filter by owner (organization)
	HasADOProject   *bool   // Filter repositories that have ADO project set
	ADOOrganization *string // Filter by ADO organization
	ADOProject      *string // Filter by ADO project name
	IsADOGit        *bool   // Filter by ADO Git vs TFVC
}

// SaveRepository inserts or updates a repository in the database using GORM
// This handles both the main repository table and all related sub-tables
func (d *Database) SaveRepository(ctx context.Context, repo *models.Repository) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if repository already exists
		var existing models.Repository
		err := tx.Where("full_name = ?", repo.FullName).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// Insert new repository
			return d.createRepository(tx, repo)
		} else if err != nil {
			return fmt.Errorf("failed to check existing repository: %w", err)
		}

		// Repository exists - update it
		return d.updateExistingRepository(tx, &existing, repo)
	})
}

// createRepository inserts a new repository with all related tables
func (d *Database) createRepository(tx *gorm.DB, repo *models.Repository) error {
	// Set timestamps if not already set
	if repo.DiscoveredAt.IsZero() {
		repo.DiscoveredAt = time.Now()
	}
	if repo.UpdatedAt.IsZero() {
		repo.UpdatedAt = time.Now()
	}

	// Store related tables temporarily and clear from repo to prevent GORM from
	// auto-creating them (we'll create them explicitly with proper repository_id)
	gitProps := repo.GitProperties
	features := repo.Features
	adoProps := repo.ADOProperties
	validation := repo.Validation

	repo.GitProperties = nil
	repo.Features = nil
	repo.ADOProperties = nil
	repo.Validation = nil

	// Create the main repository record (without associations)
	if err := tx.Create(repo).Error; err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	// Restore and create related tables with correct repository_id
	// Use Select("*") to ensure all fields (including zero values) are saved
	if gitProps != nil {
		gitProps.RepositoryID = repo.ID
		if err := tx.Select("*").Create(gitProps).Error; err != nil {
			return fmt.Errorf("failed to create git properties: %w", err)
		}
		repo.GitProperties = gitProps
	}

	if features != nil {
		features.RepositoryID = repo.ID
		if err := tx.Select("*").Create(features).Error; err != nil {
			return fmt.Errorf("failed to create features: %w", err)
		}
		repo.Features = features
	}

	if adoProps != nil {
		adoProps.RepositoryID = repo.ID
		// Use Omit with an empty struct field to force all fields to be saved
		// This ensures boolean false values like IsGit=false are saved correctly
		if err := tx.Model(adoProps).Create(map[string]interface{}{
			"repository_id":             adoProps.RepositoryID,
			"project":                   adoProps.Project,
			"is_git":                    adoProps.IsGit,
			"has_boards":                adoProps.HasBoards,
			"has_pipelines":             adoProps.HasPipelines,
			"has_ghas":                  adoProps.HasGHAS,
			"pipeline_count":            adoProps.PipelineCount,
			"yaml_pipeline_count":       adoProps.YAMLPipelineCount,
			"classic_pipeline_count":    adoProps.ClassicPipelineCount,
			"pipeline_run_count":        adoProps.PipelineRunCount,
			"has_service_connections":   adoProps.HasServiceConnections,
			"has_variable_groups":       adoProps.HasVariableGroups,
			"has_self_hosted_agents":    adoProps.HasSelfHostedAgents,
			"pull_request_count":        adoProps.PullRequestCount,
			"open_pr_count":             adoProps.OpenPRCount,
			"pr_with_linked_work_items": adoProps.PRWithLinkedWorkItems,
			"pr_with_attachments":       adoProps.PRWithAttachments,
			"work_item_count":           adoProps.WorkItemCount,
			"work_item_linked_count":    adoProps.WorkItemLinkedCount,
			"active_work_item_count":    adoProps.ActiveWorkItemCount,
			"work_item_types":           adoProps.WorkItemTypes,
			"branch_policy_count":       adoProps.BranchPolicyCount,
			"branch_policy_types":       adoProps.BranchPolicyTypes,
			"required_reviewer_count":   adoProps.RequiredReviewerCount,
			"build_validation_policies": adoProps.BuildValidationPolicies,
			"has_wiki":                  adoProps.HasWiki,
			"wiki_page_count":           adoProps.WikiPageCount,
			"test_plan_count":           adoProps.TestPlanCount,
			"test_case_count":           adoProps.TestCaseCount,
			"package_feed_count":        adoProps.PackageFeedCount,
			"has_artifacts":             adoProps.HasArtifacts,
			"service_hook_count":        adoProps.ServiceHookCount,
			"installed_extensions":      adoProps.InstalledExtensions,
		}).Error; err != nil {
			return fmt.Errorf("failed to create ADO properties: %w", err)
		}
		repo.ADOProperties = adoProps
	}

	if validation != nil {
		validation.RepositoryID = repo.ID
		if err := tx.Select("*").Create(validation).Error; err != nil {
			return fmt.Errorf("failed to create validation: %w", err)
		}
		repo.Validation = validation
	}

	return nil
}

// updateExistingRepository updates an existing repository and its related tables
func (d *Database) updateExistingRepository(tx *gorm.DB, existing, repo *models.Repository) error {
	// Preserve the ID
	repo.ID = existing.ID

	// Preserve migration state during re-discovery
	preserveMigrationState := repo.Status == string(models.StatusPending) &&
		(existing.Status != string(models.StatusPending) || existing.BatchID != nil)

	if preserveMigrationState {
		repo.Status = existing.Status
		repo.BatchID = existing.BatchID
		repo.Priority = existing.Priority
		repo.DestinationURL = existing.DestinationURL
		repo.DestinationFullName = existing.DestinationFullName
	}

	repo.UpdatedAt = time.Now()

	// Update main repository record
	if err := tx.Model(existing).Select("*").Omit("id", "full_name", "discovered_at").Updates(repo).Error; err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	// Upsert related tables if present
	if repo.GitProperties != nil {
		repo.GitProperties.RepositoryID = repo.ID
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repository_id"}},
			UpdateAll: true,
		}).Create(repo.GitProperties).Error; err != nil {
			return fmt.Errorf("failed to upsert git properties: %w", err)
		}
	}

	if repo.Features != nil {
		repo.Features.RepositoryID = repo.ID
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repository_id"}},
			UpdateAll: true,
		}).Create(repo.Features).Error; err != nil {
			return fmt.Errorf("failed to upsert features: %w", err)
		}
	}

	if repo.ADOProperties != nil {
		repo.ADOProperties.RepositoryID = repo.ID
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repository_id"}},
			UpdateAll: true,
		}).Create(repo.ADOProperties).Error; err != nil {
			return fmt.Errorf("failed to upsert ADO properties: %w", err)
		}
	}

	if repo.Validation != nil {
		repo.Validation.RepositoryID = repo.ID
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repository_id"}},
			UpdateAll: true,
		}).Create(repo.Validation).Error; err != nil {
			return fmt.Errorf("failed to upsert validation: %w", err)
		}
	}

	return nil
}

// GetRepository retrieves a repository by full name with all related data
func (d *Database) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	var repo models.Repository
	err := d.db.WithContext(ctx).
		Preload("GitProperties").
		Preload("Features").
		Preload("ADOProperties").
		Preload("Validation").
		Where("full_name = ?", fullName).
		First(&repo).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return &repo, nil
}

// ListRepositories retrieves repositories with GORM scopes for filtering
func (d *Database) ListRepositories(ctx context.Context, filters map[string]any) ([]*models.Repository, error) {
	var repos []*models.Repository

	// Start with base query
	query := d.db.WithContext(ctx).Model(&models.Repository{})

	// Only preload if detailed view requested
	if includeDetails, _ := filters["include_details"].(bool); includeDetails {
		query = query.
			Preload("GitProperties").
			Preload("Features").
			Preload("ADOProperties").
			Preload("Validation")
	}

	// Apply scopes based on filters
	query = d.applyListScopes(query, filters)

	// Execute query
	if err := query.Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	return repos, nil
}

// applyListScopes applies GORM scopes based on the provided filters.
// This method delegates to smaller helper functions for better organization and testability.
func (d *Database) applyListScopes(query *gorm.DB, filters map[string]any) *gorm.DB {
	// Apply core filters (status, batch, source, search)
	query = applyCoreFilters(query, filters)

	// Apply organization filters (GitHub org, ADO org/project, team)
	query = applyOrganizationFilters(query, filters)

	// Apply size and complexity filters
	query = applySizeFilters(query, filters)

	// Apply feature flag filters (boolean features)
	query = applyFeatureFlagFilters(query, filters)

	// Apply Azure DevOps count-based filters
	query = applyADOCountFilters(query, filters)

	// Apply batch availability filter
	if availableForBatch, ok := filters["available_for_batch"].(bool); ok && availableForBatch {
		query = query.Scopes(WithAvailableForBatch())
	}

	// Apply ordering and pagination
	query = applyOrderingAndPagination(query, filters)

	return query
}

// applyCoreFilters applies basic filters: status, batch_id, source_id, source, visibility, search
func applyCoreFilters(query *gorm.DB, filters map[string]any) *gorm.DB {
	if status, ok := filters["status"]; ok {
		query = query.Scopes(WithStatus(status))
	}

	if batchID, ok := filters["batch_id"].(int64); ok {
		query = query.Scopes(WithBatchID(batchID))
	}

	// Multi-source filter
	if sourceID, ok := filters["source_id"].(int64); ok {
		query = query.Scopes(WithSourceID(sourceID))
	}

	if source, ok := filters["source"].(string); ok {
		query = query.Scopes(WithSource(source))
	}

	if visibility, ok := filters["visibility"].(string); ok {
		query = query.Scopes(WithVisibility(visibility))
	}

	if search, ok := filters["search"].(string); ok {
		query = query.Scopes(WithSearch(search))
	}

	return query
}

// applyOrganizationFilters applies GitHub organization, ADO organization/project, and team filters
func applyOrganizationFilters(query *gorm.DB, filters map[string]any) *gorm.DB {
	if org, ok := filters["organization"]; ok {
		query = query.Scopes(WithOrganization(org))
	}

	if adoOrg, ok := filters["ado_organization"]; ok {
		query = query.Scopes(WithADOOrganization(adoOrg))
	}

	if project, ok := filters["ado_project"]; ok {
		query = query.Scopes(WithADOProject(project))
	}

	if team, ok := filters["team"]; ok {
		query = query.Scopes(WithTeam(team))
	}

	return query
}

// applySizeFilters applies size-related filters: min/max size, size category, complexity
func applySizeFilters(query *gorm.DB, filters map[string]any) *gorm.DB {
	minSize, hasMin := filters["min_size"].(int64)
	maxSize, hasMax := filters["max_size"].(int64)
	if hasMin || hasMax {
		query = query.Scopes(WithSizeRange(minSize, maxSize))
	}

	if sizeCategory, ok := filters["size_category"]; ok {
		query = query.Scopes(WithSizeCategory(sizeCategory))
	}

	if complexity, ok := filters["complexity"]; ok {
		query = query.Scopes(WithComplexity(complexity))
	}

	return query
}

// featureFlagKeys contains all supported feature flag filter keys
var featureFlagKeys = []string{
	// GitHub features (now in repository_features table)
	"has_lfs", "has_submodules", "has_large_files", "has_actions", "has_wiki",
	"has_pages", "has_discussions", "has_projects", "has_packages", "has_rulesets",
	"has_code_scanning", "has_dependabot", "has_secret_scanning",
	"has_codeowners", "has_self_hosted_runners", "has_release_assets", "has_branch_protections",
	"has_webhooks", "has_environments", "has_secrets", "has_variables",
	// Core repo flags (in main table)
	"is_archived", "is_fork",
	// Azure DevOps features (now in repository_ado_properties table)
	"ado_is_git", "ado_has_boards", "ado_has_pipelines", "ado_has_ghas", "ado_has_wiki",
}

// applyFeatureFlagFilters applies boolean feature flag filters
func applyFeatureFlagFilters(query *gorm.DB, filters map[string]any) *gorm.DB {
	featureFlags := make(map[string]bool)
	for _, key := range featureFlagKeys {
		if value, ok := filters[key].(bool); ok {
			featureFlags[key] = value
		}
	}
	if len(featureFlags) > 0 {
		query = query.Scopes(WithFeatureFlags(featureFlags))
	}
	return query
}

// adoCountFilterKeys contains all supported ADO count filter keys
var adoCountFilterKeys = []string{
	"ado_pull_request_count", "ado_work_item_count", "ado_branch_policy_count",
	"ado_yaml_pipeline_count", "ado_classic_pipeline_count", "ado_test_plan_count",
	"ado_package_feed_count", "ado_service_hook_count",
}

// applyADOCountFilters applies Azure DevOps count-based filters
func applyADOCountFilters(query *gorm.DB, filters map[string]any) *gorm.DB {
	adoCountFilters := make(map[string]string)
	for _, key := range adoCountFilterKeys {
		if value, ok := filters[key].(string); ok {
			adoCountFilters[key] = value
		}
	}
	if len(adoCountFilters) > 0 {
		query = query.Scopes(WithADOCountFilters(adoCountFilters))
	}
	return query
}

// applyOrderingAndPagination applies sort order and pagination filters
func applyOrderingAndPagination(query *gorm.DB, filters map[string]any) *gorm.DB {
	sortBy := "name" // default
	if sort, ok := filters["sort_by"].(string); ok {
		sortBy = sort
	}
	query = query.Scopes(WithOrdering(sortBy))

	limit, _ := filters["limit"].(int)
	offset, _ := filters["offset"].(int)
	if limit > 0 || offset > 0 {
		query = query.Scopes(WithPagination(limit, offset))
	}

	return query
}

// UpdateRepository updates a repository's fields
func (d *Database) UpdateRepository(ctx context.Context, repo *models.Repository) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update main repository record
		result := tx.Model(&models.Repository{}).
			Where("full_name = ?", repo.FullName).
			Select("*").
			Omit("id", "full_name", "discovered_at").
			Updates(repo)

		if result.Error != nil {
			return fmt.Errorf("failed to update repository: %w", result.Error)
		}

		// Get the repository ID for related table updates
		var existing models.Repository
		if err := tx.Where("full_name = ?", repo.FullName).First(&existing).Error; err != nil {
			return fmt.Errorf("failed to get repository for related updates: %w", err)
		}

		// Update related tables if present
		if repo.GitProperties != nil {
			repo.GitProperties.RepositoryID = existing.ID
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "repository_id"}},
				UpdateAll: true,
			}).Create(repo.GitProperties).Error; err != nil {
				return fmt.Errorf("failed to upsert git properties: %w", err)
			}
		}

		if repo.Features != nil {
			repo.Features.RepositoryID = existing.ID
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "repository_id"}},
				UpdateAll: true,
			}).Create(repo.Features).Error; err != nil {
				return fmt.Errorf("failed to upsert features: %w", err)
			}
		}

		if repo.ADOProperties != nil {
			repo.ADOProperties.RepositoryID = existing.ID
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "repository_id"}},
				UpdateAll: true,
			}).Create(repo.ADOProperties).Error; err != nil {
				return fmt.Errorf("failed to upsert ADO properties: %w", err)
			}
		}

		if repo.Validation != nil {
			repo.Validation.RepositoryID = existing.ID
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "repository_id"}},
				UpdateAll: true,
			}).Create(repo.Validation).Error; err != nil {
				return fmt.Errorf("failed to upsert validation: %w", err)
			}
		}

		return nil
	})
}

// UpdateRepositoryStatus updates only the status of a repository using GORM
func (d *Database) UpdateRepositoryStatus(ctx context.Context, fullName string, status models.MigrationStatus) error {
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("full_name = ?", fullName).
		Updates(map[string]any{
			"status":     string(status),
			"updated_at": time.Now().UTC(),
		})

	return result.Error
}

// UpdateRepositoryDryRunTimestamp updates the last_dry_run_at timestamp for a repository using GORM
func (d *Database) UpdateRepositoryDryRunTimestamp(ctx context.Context, fullName string) error {
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("full_name = ?", fullName).
		Updates(map[string]any{
			"last_dry_run_at": now,
			"updated_at":      now,
		})

	return result.Error
}

// DeleteRepository deletes a repository by full name using GORM
// Related tables are automatically deleted via ON DELETE CASCADE
func (d *Database) DeleteRepository(ctx context.Context, fullName string) error {
	result := d.db.WithContext(ctx).Where("full_name = ?", fullName).Delete(&models.Repository{})
	return result.Error
}

// CountRepositories returns the total count of repositories with optional filters using GORM
func (d *Database) CountRepositories(ctx context.Context, filters map[string]any) (int, error) {
	var count int64
	query := d.db.WithContext(ctx).Model(&models.Repository{})

	// Apply status filter
	if statusValue, ok := filters["status"]; ok {
		query = query.Scopes(WithStatus(statusValue))
	}

	// Apply batch_id filter
	if batchID, ok := filters["batch_id"].(int64); ok {
		query = query.Scopes(WithBatchID(batchID))
	}

	// Apply source filter
	if source, ok := filters["source"].(string); ok {
		query = query.Scopes(WithSource(source))
	}

	err := query.Count(&count).Error
	return int(count), err
}

// GetRepositoryStatsByStatus returns counts grouped by status using GORM
func (d *Database) GetRepositoryStatsByStatus(ctx context.Context) (map[string]int, error) {
	type StatusCount struct {
		Status string
		Count  int
	}

	var results []StatusCount
	err := d.db.WithContext(ctx).
		Model(&models.Repository{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	for _, result := range results {
		stats[result.Status] = result.Count
	}

	return stats, nil
}

// GetRepositoriesByIDs retrieves multiple repositories by their IDs using GORM
func (d *Database) GetRepositoriesByIDs(ctx context.Context, ids []int64) ([]*models.Repository, error) {
	if len(ids) == 0 {
		return []*models.Repository{}, nil
	}

	var repos []*models.Repository
	err := d.db.WithContext(ctx).
		Preload("GitProperties").
		Preload("Features").
		Preload("ADOProperties").
		Preload("Validation").
		Where("id IN ?", ids).
		Find(&repos).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by IDs: %w", err)
	}

	return repos, nil
}

// GetRepositoriesByNames retrieves multiple repositories by their full names using GORM
func (d *Database) GetRepositoriesByNames(ctx context.Context, names []string) ([]*models.Repository, error) {
	if len(names) == 0 {
		return []*models.Repository{}, nil
	}

	var repos []*models.Repository
	err := d.db.WithContext(ctx).
		Preload("GitProperties").
		Preload("Features").
		Preload("ADOProperties").
		Preload("Validation").
		Where("full_name IN ?", names).
		Find(&repos).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by names: %w", err)
	}

	return repos, nil
}

// GetRepositoryByID retrieves a repository by ID using GORM
func (d *Database) GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error) {
	var repo models.Repository
	err := d.db.WithContext(ctx).
		Preload("GitProperties").
		Preload("Features").
		Preload("ADOProperties").
		Preload("Validation").
		Where("id = ?", id).
		First(&repo).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository by ID: %w", err)
	}

	return &repo, nil
}

// UpdateRepositoryValidation updates the validation fields for a repository using GORM
func (d *Database) UpdateRepositoryValidation(ctx context.Context, fullName string, validationStatus string, validationDetails, destinationData *string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get the repository ID
		var repo models.Repository
		if err := tx.Where("full_name = ?", fullName).First(&repo).Error; err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}

		// Update the validation table
		validation := &models.RepositoryValidation{
			RepositoryID:      repo.ID,
			ValidationStatus:  &validationStatus,
			ValidationDetails: validationDetails,
			DestinationData:   destinationData,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repository_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"validation_status", "validation_details", "destination_data"}),
		}).Create(validation).Error; err != nil {
			return fmt.Errorf("failed to update repository validation: %w", err)
		}

		// Update the main repository's updated_at timestamp
		if err := tx.Model(&models.Repository{}).
			Where("id = ?", repo.ID).
			Update("updated_at", time.Now().UTC()).Error; err != nil {
			return fmt.Errorf("failed to update repository timestamp: %w", err)
		}

		return nil
	})
}

// RollbackRepository marks a repository as rolled back and creates a migration history entry using GORM
func (d *Database) RollbackRepository(ctx context.Context, fullName string, reason string) error {
	// Get the repository
	repo, err := d.GetRepository(ctx, fullName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return fmt.Errorf("repository not found")
	}

	oldBatchID := repo.BatchID

	// Update repository status to rolled_back and clear batch assignment using GORM
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("full_name = ?", fullName).
		Updates(map[string]any{
			"status":     string(models.StatusRolledBack),
			"batch_id":   nil,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update repository status: %w", result.Error)
	}

	// Update the old batch's repository count if it was in a batch
	if oldBatchID != nil {
		if err := d.updateBatchRepositoryCount(ctx, *oldBatchID); err != nil {
			// Log but don't fail the rollback
			return fmt.Errorf("failed to update batch repository count: %w", err)
		}
	}

	// Create migration history entry for rollback using GORM
	message := "Repository rolled back"
	if reason != "" {
		message = reason
	}

	history := &models.MigrationHistory{
		RepositoryID: repo.ID,
		Status:       "rolled_back",
		Phase:        "rollback",
		Message:      &message,
		StartedAt:    now,
		CompletedAt:  &now,
	}

	result = d.db.WithContext(ctx).Create(history)
	if result.Error != nil {
		return fmt.Errorf("failed to create rollback history: %w", result.Error)
	}

	return nil
}
