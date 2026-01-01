package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
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
func (d *Database) SaveRepository(ctx context.Context, repo *models.Repository) error {
	// Check if repository already exists
	var existing models.Repository
	err := d.db.WithContext(ctx).Where("full_name = ?", repo.FullName).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Insert new repository
		// Set timestamps if not already set
		if repo.DiscoveredAt.IsZero() {
			repo.DiscoveredAt = time.Now()
		}
		if repo.UpdatedAt.IsZero() {
			repo.UpdatedAt = time.Now()
		}

		// CRITICAL FIX: GORM by default omits zero values from INSERT statements
		// This means ADOIsGit=false won't be inserted, and DB uses DEFAULT TRUE
		// Solution: Explicitly set all fields that can have meaningful zero values
		// We use a map for the critical boolean field to force its inclusion

		// First, create a base insert to get the ID
		// We'll use a temporary approach: set ADOIsGit to the opposite, then update it
		tempIsGit := repo.ADOIsGit
		if !tempIsGit {
			// For TFVC repos (false), we need special handling
			repo.ADOIsGit = true // Temporarily set to true for insert
		}

		result := d.db.WithContext(ctx).Create(repo)
		if result.Error != nil {
			return fmt.Errorf("failed to create repository: %w", result.Error)
		}

		// Now immediately update the ADOIsGit field to the correct value
		if !tempIsGit {
			updateResult := d.db.WithContext(ctx).Model(&models.Repository{}).
				Where("id = ?", repo.ID).
				Update("ado_is_git", false)
			if updateResult.Error != nil {
				return fmt.Errorf("failed to set ado_is_git to false: %w", updateResult.Error)
			}
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing repository: %w", err)
	}

	// Repository exists - update it
	// Preserve the ID and migration-related fields
	repo.ID = existing.ID

	// IMPORTANT: Preserve existing migration state during re-discovery
	// Re-discovery should only update repository metadata (size, features, etc.),
	// not reset migration progress (status, batch_id, priority, etc.)
	// Preserve migration state if:
	// 1. The incoming status is "pending" (indicating it's a re-discovery, not an intentional status change), AND
	// 2. Either the existing status is not "pending" OR the existing repo is in a batch
	// This ensures that repos assigned to a batch (even if still in "pending" status) don't lose their batch assignment
	preserveMigrationState := repo.Status == string(models.StatusPending) &&
		(existing.Status != string(models.StatusPending) || existing.BatchID != nil)

	// If we're preserving migration state, restore the fields from existing repository
	if preserveMigrationState {
		repo.BatchID = existing.BatchID
		repo.Priority = existing.Priority
		repo.DestinationURL = existing.DestinationURL
		repo.DestinationFullName = existing.DestinationFullName
	}

	// GORM's Updates() with structs skips zero values (false, 0, "", nil) by default
	// We need to convert to a map to include these values in the update
	// This is especially important for ADOIsGit which can be false (TFVC repos)
	updateMap := map[string]any{
		"source":            repo.Source,
		"source_url":        repo.SourceURL,
		"total_size":        repo.TotalSize,
		"default_branch":    repo.DefaultBranch,
		"branch_count":      repo.BranchCount,
		"commit_count":      repo.CommitCount,
		"visibility":        repo.Visibility,
		"updated_at":        time.Now(),
		"last_discovery_at": repo.LastDiscoveryAt,
		// Boolean fields - include explicitly to handle false values
		"is_archived":              repo.IsArchived,
		"is_fork":                  repo.IsFork,
		"has_lfs":                  repo.HasLFS,
		"has_submodules":           repo.HasSubmodules,
		"has_large_files":          repo.HasLargeFiles,
		"has_wiki":                 repo.HasWiki,
		"has_pages":                repo.HasPages,
		"has_discussions":          repo.HasDiscussions,
		"has_actions":              repo.HasActions,
		"has_projects":             repo.HasProjects,
		"has_packages":             repo.HasPackages,
		"has_rulesets":             repo.HasRulesets,
		"has_code_scanning":        repo.HasCodeScanning,
		"has_dependabot":           repo.HasDependabot,
		"has_secret_scanning":      repo.HasSecretScanning,
		"has_codeowners":           repo.HasCodeowners,
		"has_self_hosted_runners":  repo.HasSelfHostedRunners,
		"has_release_assets":       repo.HasReleaseAssets,
		"has_oversized_commits":    repo.HasOversizedCommits,
		"has_long_refs":            repo.HasLongRefs,
		"has_blocking_files":       repo.HasBlockingFiles,
		"has_large_file_warnings":  repo.HasLargeFileWarnings,
		"has_oversized_repository": repo.HasOversizedRepository,
		"is_source_locked":         repo.IsSourceLocked,
		// ADO-specific fields - CRITICAL: Include ADOIsGit which can be false
		"ado_project":                   repo.ADOProject,
		"ado_is_git":                    repo.ADOIsGit, // This is the key field that needs false support
		"ado_has_boards":                repo.ADOHasBoards,
		"ado_has_pipelines":             repo.ADOHasPipelines,
		"ado_has_ghas":                  repo.ADOHasGHAS,
		"ado_pull_request_count":        repo.ADOPullRequestCount,
		"ado_work_item_count":           repo.ADOWorkItemCount,
		"ado_branch_policy_count":       repo.ADOBranchPolicyCount,
		"ado_pipeline_count":            repo.ADOPipelineCount,
		"ado_yaml_pipeline_count":       repo.ADOYAMLPipelineCount,
		"ado_classic_pipeline_count":    repo.ADOClassicPipelineCount,
		"ado_pipeline_run_count":        repo.ADOPipelineRunCount,
		"ado_has_service_connections":   repo.ADOHasServiceConnections,
		"ado_has_variable_groups":       repo.ADOHasVariableGroups,
		"ado_has_self_hosted_agents":    repo.ADOHasSelfHostedAgents,
		"ado_work_item_linked_count":    repo.ADOWorkItemLinkedCount,
		"ado_active_work_item_count":    repo.ADOActiveWorkItemCount,
		"ado_work_item_types":           repo.ADOWorkItemTypes,
		"ado_open_pr_count":             repo.ADOOpenPRCount,
		"ado_pr_with_linked_work_items": repo.ADOPRWithLinkedWorkItems,
		"ado_pr_with_attachments":       repo.ADOPRWithAttachments,
		"ado_branch_policy_types":       repo.ADOBranchPolicyTypes,
		"ado_required_reviewer_count":   repo.ADORequiredReviewerCount,
		"ado_build_validation_policies": repo.ADOBuildValidationPolicies,
		"ado_has_wiki":                  repo.ADOHasWiki,
		"ado_wiki_page_count":           repo.ADOWikiPageCount,
		"ado_test_plan_count":           repo.ADOTestPlanCount,
		"ado_test_case_count":           repo.ADOTestCaseCount,
		"ado_package_feed_count":        repo.ADOPackageFeedCount,
		"ado_has_artifacts":             repo.ADOHasArtifacts,
		"ado_service_hook_count":        repo.ADOServiceHookCount,
		"ado_installed_extensions":      repo.ADOInstalledExtensions,
		// Integer and string fields
		"large_file_count":             repo.LargeFileCount,
		"largest_file":                 repo.LargestFile,
		"largest_file_size":            repo.LargestFileSize,
		"largest_commit":               repo.LargestCommit,
		"largest_commit_size":          repo.LargestCommitSize,
		"last_commit_sha":              repo.LastCommitSHA,
		"last_commit_date":             repo.LastCommitDate,
		"branch_protections":           repo.BranchProtections,
		"tag_protection_count":         repo.TagProtectionCount,
		"environment_count":            repo.EnvironmentCount,
		"secret_count":                 repo.SecretCount,
		"variable_count":               repo.VariableCount,
		"webhook_count":                repo.WebhookCount,
		"contributor_count":            repo.ContributorCount,
		"top_contributors":             repo.TopContributors,
		"issue_count":                  repo.IssueCount,
		"pull_request_count":           repo.PullRequestCount,
		"tag_count":                    repo.TagCount,
		"open_issue_count":             repo.OpenIssueCount,
		"open_pr_count":                repo.OpenPRCount,
		"workflow_count":               repo.WorkflowCount,
		"collaborator_count":           repo.CollaboratorCount,
		"installed_apps_count":         repo.InstalledAppsCount,
		"installed_apps":               repo.InstalledApps,
		"release_count":                repo.ReleaseCount,
		"oversized_commit_details":     repo.OversizedCommitDetails,
		"long_ref_details":             repo.LongRefDetails,
		"blocking_file_details":        repo.BlockingFileDetails,
		"large_file_warning_details":   repo.LargeFileWarningDetails,
		"oversized_repository_details": repo.OversizedRepositoryDetails,
		"estimated_metadata_size":      repo.EstimatedMetadataSize,
		"metadata_size_details":        repo.MetadataSizeDetails,
		"source_migration_id":          repo.SourceMigrationID,
		// Complexity scoring fields
		"complexity_score":     repo.ComplexityScore,
		"complexity_breakdown": repo.ComplexityBreakdown,
	}

	// Only include migration-related updates if we're not preserving migration state
	if !preserveMigrationState {
		updateMap["status"] = repo.Status
		updateMap["batch_id"] = repo.BatchID
		updateMap["priority"] = repo.Priority
		updateMap["destination_url"] = repo.DestinationURL
		updateMap["destination_full_name"] = repo.DestinationFullName
	}

	result := d.db.WithContext(ctx).Model(&existing).Updates(updateMap)

	if result.Error != nil {
		return fmt.Errorf("failed to update repository: %w", result.Error)
	}

	return nil
}

// GetRepository retrieves a repository by full name using GORM
func (d *Database) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	var repo models.Repository
	err := d.db.WithContext(ctx).Where("full_name = ?", fullName).First(&repo).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Complexity scores are now stored in the database, no need to calculate on retrieval

	return &repo, nil
}

// ListRepositories retrieves repositories with GORM scopes for filtering
func (d *Database) ListRepositories(ctx context.Context, filters map[string]any) ([]*models.Repository, error) {
	var repos []*models.Repository

	// Start with base query
	query := d.db.WithContext(ctx).Model(&models.Repository{})

	// Apply scopes based on filters
	query = d.applyListScopes(query, filters)

	// Execute query
	if err := query.Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Calculate and populate complexity scores
	// Complexity scores are nice-to-have, not critical, so we don't fail the request on error
	_ = d.populateComplexityScores(ctx, repos)

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
	// GitHub features
	"has_lfs", "has_submodules", "has_large_files", "has_actions", "has_wiki",
	"has_pages", "has_discussions", "has_projects", "has_packages", "has_rulesets",
	"is_archived", "is_fork", "has_code_scanning", "has_dependabot", "has_secret_scanning",
	"has_codeowners", "has_self_hosted_runners", "has_release_assets", "has_branch_protections",
	"has_webhooks", "has_environments", "has_secrets", "has_variables",
	// Azure DevOps features
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

// populateComplexityScores is DEPRECATED - complexity scores are now calculated during profiling and stored in the database
// This function is kept for backward compatibility but does nothing
func (d *Database) populateComplexityScores(ctx context.Context, repos []*models.Repository) error {
	// Complexity scores are now pre-calculated during repository profiling and stored in the database
	// No need to calculate them on every retrieval
	return nil
}

// UpdateRepository updates a repository's fields
// nolint:dupl // SaveRepository and UpdateRepository have different SQL operations
func (d *Database) UpdateRepository(ctx context.Context, repo *models.Repository) error {
	// Use GORM Updates with Select to update all fields, including zero values
	// Using Save() can cause foreign key issues when batch_id is updated but the batch doesn't exist yet
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("full_name = ?", repo.FullName).
		Select("*").
		Omit("id", "full_name", "created_at").
		Updates(repo)

	if result.Error != nil {
		return fmt.Errorf("failed to update repository: %w", result.Error)
	}

	return nil
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
	err := d.db.WithContext(ctx).Where("id IN ?", ids).Find(&repos).Error
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
	err := d.db.WithContext(ctx).Where("full_name IN ?", names).Find(&repos).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by names: %w", err)
	}

	return repos, nil
}

// GetRepositoryByID retrieves a repository by ID using GORM
func (d *Database) GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error) {
	var repo models.Repository
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&repo).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository by ID: %w", err)
	}

	// Complexity scores are now stored in the database, no need to calculate on retrieval

	return &repo, nil
}

// UpdateRepositoryValidation updates the validation fields for a repository using GORM
func (d *Database) UpdateRepositoryValidation(ctx context.Context, fullName string, validationStatus string, validationDetails, destinationData *string) error {
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("full_name = ?", fullName).
		Updates(map[string]any{
			"validation_status":  validationStatus,
			"validation_details": validationDetails,
			"destination_data":   destinationData,
			"updated_at":         now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update repository validation: %w", result.Error)
	}

	return nil
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
