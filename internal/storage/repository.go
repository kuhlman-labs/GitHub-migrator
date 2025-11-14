package storage

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

const (
	// Batch status constants
	batchStatusPending    = "pending"
	batchStatusReady      = "ready"
	batchStatusInProgress = "in_progress"
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
	// Preserve the ID
	repo.ID = existing.ID

	// GORM's Updates() with structs skips zero values (false, 0, "", nil) by default
	// We need to convert to a map to include these values in the update
	// This is especially important for ADOIsGit which can be false (TFVC repos)
	updateMap := map[string]interface{}{
		"source":            repo.Source,
		"source_url":        repo.SourceURL,
		"total_size":        repo.TotalSize,
		"default_branch":    repo.DefaultBranch,
		"branch_count":      repo.BranchCount,
		"commit_count":      repo.CommitCount,
		"visibility":        repo.Visibility,
		"status":            repo.Status,
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
		"release_count":                repo.ReleaseCount,
		"oversized_commit_details":     repo.OversizedCommitDetails,
		"long_ref_details":             repo.LongRefDetails,
		"blocking_file_details":        repo.BlockingFileDetails,
		"large_file_warning_details":   repo.LargeFileWarningDetails,
		"oversized_repository_details": repo.OversizedRepositoryDetails,
		"estimated_metadata_size":      repo.EstimatedMetadataSize,
		"metadata_size_details":        repo.MetadataSizeDetails,
		"source_migration_id":          repo.SourceMigrationID,
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

	// Calculate complexity score
	_ = d.populateComplexityScores(ctx, []*models.Repository{&repo})

	return &repo, nil
}

// buildADOComplexityScoreSQL generates the SQL expression for calculating Azure DevOps-specific complexity scores
// Based on what GitHub Enterprise Importer supports for ADO migrations
//
//nolint:unused // Reserved for future ADO-specific complexity queries
func (d *Database) buildADOComplexityScoreSQL() string {
	// ADO-specific complexity scoring based on GEI migration support
	// Categories: simple (≤5), medium (6-10), complex (11-17), very_complex (≥18)
	//
	// Scoring factors:
	// BLOCKING (50 points):
	//   - TFVC repository: +50 (requires git conversion before migration)
	//
	// HIGH IMPACT (3 points):
	//   - Azure Boards: +3 (work items don't migrate, only PR links)
	//   - Azure Pipelines: +3 (YAML files migrate, but history doesn't)
	//
	// MODERATE IMPACT (2 points):
	//   - Many PRs: +2 (PRs migrate with GEI, but many means more complexity)
	//   - LFS: +2 (special handling)
	//   - Submodules: +2 (dependency management)
	//
	// LOW IMPACT (1 point):
	//   - Branch policies: +1 (migrate with GEI)
	//   - Work item links: +1 (PR links migrate)
	//   - GHAS: +1 (GitHub Advanced Security for ADO)
	//
	// STANDARD GIT FACTORS:
	//   - Size-based: 0-9 points
	//   - Large files: +4 points
	//   - Activity-based: 0-4 points

	return `(
		-- TFVC repos are blocking (50 points)
		(CASE WHEN ado_is_git = FALSE THEN 50 ELSE 0 END) +
		
		-- Azure Boards (work items don't migrate)
		(CASE WHEN ado_has_boards = TRUE THEN 3 ELSE 0 END) +
		
		-- Azure Pipelines (history doesn't migrate)
		(CASE WHEN ado_has_pipelines = TRUE THEN 3 ELSE 0 END) +
		
		-- Pull requests (these migrate, but many PRs adds complexity)
		(CASE WHEN ado_pull_request_count > 50 THEN 2 
		      WHEN ado_pull_request_count > 10 THEN 1 
		      ELSE 0 END) +
		
		-- Branch policies (these migrate with GEI)
		(CASE WHEN ado_branch_policy_count > 0 THEN 1 ELSE 0 END) +
		
		-- Work items (PR links migrate, not the items themselves)
		(CASE WHEN ado_work_item_count > 0 THEN 1 ELSE 0 END) +
		
		-- GHAS for ADO
		(CASE WHEN ado_has_ghas = TRUE THEN 1 ELSE 0 END) +
		
		-- Standard git factors (size, large files, LFS, submodules, activity)
		-- These are shared with GitHub scoring
		` + d.buildStandardGitComplexityFactors() + `
	)`
}

// buildStandardGitComplexityFactors returns SQL for complexity factors common to both GitHub and ADO
//
//nolint:unused // Reserved for future shared complexity calculations
func (d *Database) buildStandardGitComplexityFactors() string {
	const (
		MB100 = 104857600  // 100MB
		GB1   = 1073741824 // 1GB
		GB5   = 5368709120 // 5GB
	)

	return fmt.Sprintf(`(
		-- Size tier scoring (0-9 points)
		(CASE 
			WHEN total_size IS NULL THEN 0
			WHEN total_size < %d THEN 0
			WHEN total_size < %d THEN 1
			WHEN total_size < %d THEN 2
			ELSE 3
		END) * 3 +
		
		-- Large files (blocking for GitHub migrations)
		(CASE WHEN has_large_files = TRUE THEN 4 ELSE 0 END) +
		
		-- LFS
		(CASE WHEN has_lfs = TRUE THEN 2 ELSE 0 END) +
		
		-- Submodules
		(CASE WHEN has_submodules = TRUE THEN 2 ELSE 0 END)
	)`, MB100, GB1, GB5)
}

// buildGitHubComplexityScoreSQL generates the SQL expression for calculating GitHub-specific complexity scores
// This includes activity-based scoring using quantiles from the repository dataset
func (d *Database) buildGitHubComplexityScoreSQL() string {
	// GitHub-specific complexity scoring (refined based on GEI migration documentation)
	// Categories: simple (≤5), medium (6-10), complex (11-17), very_complex (≥18)
	//
	// Scoring factors based on remediation difficulty:
	// HIGH IMPACT (3-4 points):
	//   - Large files: +4 (requires remediation before migration)
	//   - Environments: +3 (manual recreation of configs, protection rules)
	//   - Secrets: +3 (manual recreation, high security sensitivity)
	//   - Packages: +3 (don't migrate with GEI)
	//   - Self-hosted runners: +3 (infrastructure reconfiguration)
	//
	// MODERATE IMPACT (2 points):
	//   - Variables: +2 (manual recreation, less sensitive than secrets)
	//   - Discussions: +2 (don't migrate, community impact)
	//   - Releases: +2 (GHES 3.5.0+ only, may need manual migration)
	//   - LFS: +2 (special handling required)
	//   - Submodules: +2 (dependency management complexity)
	//   - GitHub Apps: +2 (reconfiguration/reinstallation)
	//   - ProjectsV2: +2 (don't migrate, must be manually recreated)
	//
	// LOW IMPACT (1 point):
	//   - GHAS features: +1 (code scanning, dependabot, secret scanning - simple toggles)
	//   - Webhooks: +1 (must re-enable, straightforward)
	//   - Branch protections: +1 (migrates but may need adjustment)
	//   - Rulesets: +1 (manual recreation, replaces deprecated tag protections)
	//   - Public visibility: +1 (transformation considerations)
	//   - Internal visibility: +1 (transformation considerations)
	//   - CODEOWNERS: +1 (verification required)
	//
	// ACTIVITY TIER (0-4 points):
	//   Uses quantiles to determine activity level relative to customer's repos
	//   High-activity repos require significantly more planning and coordination
	//   - High (top 25%): +4 (many users, extensive coordination, high impact)
	//   - Moderate (25-75%): +2 (some coordination needed)
	//   - Low (bottom 25%): +0 (few users, low coordination needs)
	//   Activity combines: branch_count, commit_count, issue_count, pull_request_count

	const (
		MB100 = 104857600  // 100MB
		GB1   = 1073741824 // 1GB
		GB5   = 5368709120 // 5GB
	)

	// Use TRUE/FALSE for boolean comparisons (works across all databases: SQLite, PostgreSQL, SQL Server)
	const trueVal = "TRUE"

	return fmt.Sprintf(`(
		-- Size tier scoring (0-9 points)
		(CASE 
			WHEN total_size IS NULL THEN 0
			WHEN total_size < %d THEN 0
			WHEN total_size < %d THEN 1
			WHEN total_size < %d THEN 2
			ELSE 3
		END) * 3 +
		
		-- High impact features (3-4 points each)
		CASE WHEN has_large_files = %s THEN 4 ELSE 0 END +
		CASE WHEN environment_count > 0 THEN 3 ELSE 0 END +
		CASE WHEN secret_count > 0 THEN 3 ELSE 0 END +
		CASE WHEN has_packages = %s THEN 3 ELSE 0 END +
		CASE WHEN has_self_hosted_runners = %s THEN 3 ELSE 0 END +
		
		-- Moderate impact features (2 points each)
		CASE WHEN variable_count > 0 THEN 2 ELSE 0 END +
		CASE WHEN has_discussions = %s THEN 2 ELSE 0 END +
		CASE WHEN release_count > 0 THEN 2 ELSE 0 END +
		CASE WHEN has_lfs = %s THEN 2 ELSE 0 END +
		CASE WHEN has_submodules = %s THEN 2 ELSE 0 END +
		CASE WHEN installed_apps_count > 0 THEN 2 ELSE 0 END +
		CASE WHEN has_projects = %s THEN 2 ELSE 0 END +
		
		-- Low impact features (1 point each)
		CASE WHEN has_code_scanning = %s OR has_dependabot = %s OR has_secret_scanning = %s THEN 1 ELSE 0 END +
		CASE WHEN webhook_count > 0 THEN 1 ELSE 0 END +
		CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END +
		CASE WHEN has_rulesets = %s THEN 1 ELSE 0 END +
		CASE WHEN visibility = 'public' THEN 1 ELSE 0 END +
		CASE WHEN visibility = 'internal' THEN 1 ELSE 0 END +
		CASE WHEN has_codeowners = %s THEN 1 ELSE 0 END +
		
		-- Activity-based scoring (0-4 points) using quantiles
		-- High-activity repos need significantly more coordination and planning
		-- Calculate percentile rank for each activity metric and average them
		(CASE 
			WHEN (
				-- Average percentile across activity metrics
				(CAST(branch_count AS REAL) / NULLIF((SELECT MAX(branch_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(commit_count AS REAL) / NULLIF((SELECT MAX(commit_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(issue_count AS REAL) / NULLIF((SELECT MAX(issue_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(pull_request_count AS REAL) / NULLIF((SELECT MAX(pull_request_count) FROM repositories WHERE source = 'ghes'), 0)) / 4.0
			) >= 0.75 THEN 4
			WHEN (
				(CAST(branch_count AS REAL) / NULLIF((SELECT MAX(branch_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(commit_count AS REAL) / NULLIF((SELECT MAX(commit_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(issue_count AS REAL) / NULLIF((SELECT MAX(issue_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(pull_request_count AS REAL) / NULLIF((SELECT MAX(pull_request_count) FROM repositories WHERE source = 'ghes'), 0)) / 4.0
			) >= 0.25 THEN 2
			ELSE 0
		END)
	)`, MB100, GB1, GB5,
		trueVal, trueVal, trueVal, // has_large_files, has_packages, has_self_hosted_runners
		trueVal, trueVal, trueVal, // has_discussions, has_lfs, has_submodules
		trueVal,                   // has_projects
		trueVal, trueVal, trueVal, // has_code_scanning, has_dependabot, has_secret_scanning
		trueVal, trueVal) // has_rulesets, has_codeowners
}

// Individual complexity component SQL builders
// These match the logic in buildGitHubComplexityScoreSQL() but return individual scores

func buildSizePointsSQL() string {
	const (
		MB100 = 104857600  // 100MB
		GB1   = 1073741824 // 1GB
		GB5   = 5368709120 // 5GB
	)
	return fmt.Sprintf(`(CASE 
		WHEN total_size IS NULL THEN 0
		WHEN total_size < %d THEN 0
		WHEN total_size < %d THEN 1
		WHEN total_size < %d THEN 2
		ELSE 3
	END) * 3`, MB100, GB1, GB5)
}

func buildLargeFilesPointsSQL() string {
	return "CASE WHEN has_large_files = TRUE THEN 4 ELSE 0 END"
}

func buildEnvironmentsPointsSQL() string {
	return "CASE WHEN environment_count > 0 THEN 3 ELSE 0 END"
}

func buildSecretsPointsSQL() string {
	return "CASE WHEN secret_count > 0 THEN 3 ELSE 0 END"
}

func buildPackagesPointsSQL() string {
	return "CASE WHEN has_packages = TRUE THEN 3 ELSE 0 END"
}

func buildRunnersPointsSQL() string {
	return "CASE WHEN has_self_hosted_runners = TRUE THEN 3 ELSE 0 END"
}

func buildVariablesPointsSQL() string {
	return "CASE WHEN variable_count > 0 THEN 2 ELSE 0 END"
}

func buildDiscussionsPointsSQL() string {
	return "CASE WHEN has_discussions = TRUE THEN 2 ELSE 0 END"
}

func buildReleasesPointsSQL() string {
	return "CASE WHEN release_count > 0 THEN 2 ELSE 0 END"
}

func buildLFSPointsSQL() string {
	return "CASE WHEN has_lfs = TRUE THEN 2 ELSE 0 END"
}

func buildSubmodulesPointsSQL() string {
	return "CASE WHEN has_submodules = TRUE THEN 2 ELSE 0 END"
}

func buildAppsPointsSQL() string {
	return "CASE WHEN installed_apps_count > 0 THEN 2 ELSE 0 END"
}

func buildProjectsPointsSQL() string {
	return "CASE WHEN has_projects = TRUE THEN 2 ELSE 0 END"
}

func buildSecurityPointsSQL() string {
	return "CASE WHEN has_code_scanning = TRUE OR has_dependabot = TRUE OR has_secret_scanning = TRUE THEN 1 ELSE 0 END"
}

func buildWebhooksPointsSQL() string {
	return "CASE WHEN webhook_count > 0 THEN 1 ELSE 0 END"
}

func buildBranchProtectionsPointsSQL() string {
	return "CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END"
}

func buildRulesetsPointsSQL() string {
	return "CASE WHEN has_rulesets = TRUE THEN 1 ELSE 0 END"
}

func buildPublicVisibilityPointsSQL() string {
	return "CASE WHEN visibility = 'public' THEN 1 ELSE 0 END"
}

func buildInternalVisibilityPointsSQL() string {
	return "CASE WHEN visibility = 'internal' THEN 1 ELSE 0 END"
}

func buildCodeownersPointsSQL() string {
	return "CASE WHEN has_codeowners = TRUE THEN 1 ELSE 0 END"
}

func buildActivityPointsSQL() string {
	return `(CASE 
		WHEN (
			-- Average percentile across activity metrics
			(CAST(branch_count AS REAL) / NULLIF((SELECT MAX(branch_count) FROM repositories WHERE source = 'ghes'), 0) +
			 CAST(commit_count AS REAL) / NULLIF((SELECT MAX(commit_count) FROM repositories WHERE source = 'ghes'), 0) +
			 CAST(issue_count AS REAL) / NULLIF((SELECT MAX(issue_count) FROM repositories WHERE source = 'ghes'), 0) +
			 CAST(pull_request_count AS REAL) / NULLIF((SELECT MAX(pull_request_count) FROM repositories WHERE source = 'ghes'), 0)) / 4.0
		) >= 0.75 THEN 4
		WHEN (
			(CAST(branch_count AS REAL) / NULLIF((SELECT MAX(branch_count) FROM repositories WHERE source = 'ghes'), 0) +
			 CAST(commit_count AS REAL) / NULLIF((SELECT MAX(commit_count) FROM repositories WHERE source = 'ghes'), 0) +
			 CAST(issue_count AS REAL) / NULLIF((SELECT MAX(issue_count) FROM repositories WHERE source = 'ghes'), 0) +
			 CAST(pull_request_count AS REAL) / NULLIF((SELECT MAX(pull_request_count) FROM repositories WHERE source = 'ghes'), 0)) / 4.0
		) >= 0.25 THEN 2
		ELSE 0
	END)`
}

// ListRepositories retrieves repositories with optional filters
// ListRepositories retrieves repositories with GORM scopes for filtering
func (d *Database) ListRepositories(ctx context.Context, filters map[string]interface{}) ([]*models.Repository, error) {
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

// applyListScopes applies GORM scopes based on the provided filters
//
//nolint:gocyclo // Complexity is justified for handling multiple filter types
func (d *Database) applyListScopes(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	// Apply status filter
	if status, ok := filters["status"]; ok {
		query = query.Scopes(WithStatus(status))
	}

	// Apply batch_id filter
	if batchID, ok := filters["batch_id"].(int64); ok {
		query = query.Scopes(WithBatchID(batchID))
	}

	// Apply source filter
	if source, ok := filters["source"].(string); ok {
		query = query.Scopes(WithSource(source))
	}

	// Apply size range filters
	minSize, hasMin := filters["min_size"].(int64)
	maxSize, hasMax := filters["max_size"].(int64)
	if hasMin || hasMax {
		query = query.Scopes(WithSizeRange(minSize, maxSize))
	}

	// Apply search filter
	if search, ok := filters["search"].(string); ok {
		query = query.Scopes(WithSearch(search))
	}

	// Apply organization filter
	if org, ok := filters["organization"]; ok {
		query = query.Scopes(WithOrganization(org))
	}

	// Apply visibility filter
	if visibility, ok := filters["visibility"].(string); ok {
		query = query.Scopes(WithVisibility(visibility))
	}

	// Apply feature flags
	featureFlags := make(map[string]bool)
	featureKeys := []string{
		"has_lfs", "has_submodules", "has_large_files", "has_actions", "has_wiki",
		"has_pages", "has_discussions", "has_projects", "has_packages", "has_rulesets",
		"is_archived", "is_fork", "has_code_scanning", "has_dependabot", "has_secret_scanning",
		"has_codeowners", "has_self_hosted_runners", "has_release_assets", "has_branch_protections",
		"has_webhooks",
	}
	for _, key := range featureKeys {
		if value, ok := filters[key].(bool); ok {
			featureFlags[key] = value
		}
	}
	if len(featureFlags) > 0 {
		query = query.Scopes(WithFeatureFlags(featureFlags))
	}

	// Apply size category filter
	if sizeCategory, ok := filters["size_category"]; ok {
		query = query.Scopes(WithSizeCategory(sizeCategory))
	}

	// Apply complexity filter
	if complexity, ok := filters["complexity"]; ok {
		query = query.Scopes(WithComplexity(complexity))
	}

	// Apply available for batch filter
	if availableForBatch, ok := filters["available_for_batch"].(bool); ok && availableForBatch {
		query = query.Scopes(WithAvailableForBatch())
	}

	// Apply ordering
	sortBy := "name" // default
	if sort, ok := filters["sort_by"].(string); ok {
		sortBy = sort
	}
	query = query.Scopes(WithOrdering(sortBy))

	// Apply pagination
	limit, _ := filters["limit"].(int)
	offset, _ := filters["offset"].(int)
	if limit > 0 || offset > 0 {
		query = query.Scopes(WithPagination(limit, offset))
	}

	return query
}

// populateComplexityScores calculates and sets the complexity_score and complexity_breakdown fields for repositories
func (d *Database) populateComplexityScores(ctx context.Context, repos []*models.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	// Build a map of repo IDs for quick lookup
	repoIDs := make([]string, len(repos))
	repoMap := make(map[int64]*models.Repository)
	for i, repo := range repos {
		repoIDs[i] = fmt.Sprintf("%d", repo.ID)
		repoMap[repo.ID] = repo
	}

	// Calculate complexity scores AND individual components in one query
	query := fmt.Sprintf(`
		SELECT 
			id,
			%s as complexity_score,
			%s as size_points,
			%s as large_files_points,
			%s as environments_points,
			%s as secrets_points,
			%s as packages_points,
			%s as runners_points,
			%s as variables_points,
			%s as discussions_points,
			%s as releases_points,
			%s as lfs_points,
			%s as submodules_points,
			%s as apps_points,
			%s as projects_points,
			%s as security_points,
			%s as webhooks_points,
			%s as branch_protections_points,
			%s as rulesets_points,
			%s as public_visibility_points,
			%s as internal_visibility_points,
			%s as codeowners_points,
			%s as activity_points
		FROM repositories
		WHERE id IN (%s)
	`,
		d.buildGitHubComplexityScoreSQL(),
		buildSizePointsSQL(),
		buildLargeFilesPointsSQL(),
		buildEnvironmentsPointsSQL(),
		buildSecretsPointsSQL(),
		buildPackagesPointsSQL(),
		buildRunnersPointsSQL(),
		buildVariablesPointsSQL(),
		buildDiscussionsPointsSQL(),
		buildReleasesPointsSQL(),
		buildLFSPointsSQL(),
		buildSubmodulesPointsSQL(),
		buildAppsPointsSQL(),
		buildProjectsPointsSQL(),
		buildSecurityPointsSQL(),
		buildWebhooksPointsSQL(),
		buildBranchProtectionsPointsSQL(),
		buildRulesetsPointsSQL(),
		buildPublicVisibilityPointsSQL(),
		buildInternalVisibilityPointsSQL(),
		buildCodeownersPointsSQL(),
		buildActivityPointsSQL(),
		strings.Join(repoIDs, ","))

	// Use GORM Raw() for complex analytics query
	type ComplexityResult struct {
		ID                       int64
		ComplexityScore          int
		SizePoints               int
		LargeFilesPoints         int
		EnvironmentsPoints       int
		SecretsPoints            int
		PackagesPoints           int
		RunnersPoints            int
		VariablesPoints          int
		DiscussionsPoints        int
		ReleasesPoints           int
		LFSPoints                int
		SubmodulesPoints         int
		AppsPoints               int
		ProjectsPoints           int
		SecurityPoints           int
		WebhooksPoints           int
		BranchProtectionsPoints  int
		RulesetsPoints           int
		PublicVisibilityPoints   int
		InternalVisibilityPoints int
		CodeownersPoints         int
		ActivityPoints           int
	}

	var results []ComplexityResult
	err := d.db.WithContext(ctx).Raw(query).Scan(&results).Error
	if err != nil {
		return err
	}

	for _, result := range results {
		if repo, ok := repoMap[result.ID]; ok {
			score := result.ComplexityScore
			repo.ComplexityScore = &score
			repo.ComplexityBreakdown = &models.ComplexityBreakdown{
				SizePoints:               result.SizePoints,
				LargeFilesPoints:         result.LargeFilesPoints,
				EnvironmentsPoints:       result.EnvironmentsPoints,
				SecretsPoints:            result.SecretsPoints,
				PackagesPoints:           result.PackagesPoints,
				RunnersPoints:            result.RunnersPoints,
				VariablesPoints:          result.VariablesPoints,
				DiscussionsPoints:        result.DiscussionsPoints,
				ReleasesPoints:           result.ReleasesPoints,
				LFSPoints:                result.LFSPoints,
				SubmodulesPoints:         result.SubmodulesPoints,
				AppsPoints:               result.AppsPoints,
				ProjectsPoints:           result.ProjectsPoints,
				SecurityPoints:           result.SecurityPoints,
				WebhooksPoints:           result.WebhooksPoints,
				BranchProtectionsPoints:  result.BranchProtectionsPoints,
				RulesetsPoints:           result.RulesetsPoints,
				PublicVisibilityPoints:   result.PublicVisibilityPoints,
				InternalVisibilityPoints: result.InternalVisibilityPoints,
				CodeownersPoints:         result.CodeownersPoints,
				ActivityPoints:           result.ActivityPoints,
			}
		}
	}

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
		Updates(map[string]interface{}{
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
		Updates(map[string]interface{}{
			"last_dry_run_at": now,
			"updated_at":      now,
		})

	return result.Error
}

// UpdateBatchDryRunTimestamp updates the last_dry_run_at timestamp for a batch using GORM
func (d *Database) UpdateBatchDryRunTimestamp(ctx context.Context, batchID int64) error {
	result := d.db.WithContext(ctx).Model(&models.Batch{}).
		Where("id = ?", batchID).
		Update("last_dry_run_at", time.Now().UTC())

	return result.Error
}

// UpdateBatchMigrationAttemptTimestamp updates the last_migration_attempt_at timestamp for a batch using GORM
func (d *Database) UpdateBatchMigrationAttemptTimestamp(ctx context.Context, batchID int64) error {
	result := d.db.WithContext(ctx).Model(&models.Batch{}).
		Where("id = ?", batchID).
		Update("last_migration_attempt_at", time.Now().UTC())

	return result.Error
}

// UpdateBatchProgress updates batch status and operational timestamps without affecting user-configured fields using GORM
// This preserves scheduled_at and other user-set fields while updating execution state
func (d *Database) UpdateBatchProgress(ctx context.Context, batchID int64, status string, startedAt, lastDryRunAt, lastMigrationAttemptAt *time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Only update timestamps if provided (COALESCE behavior)
	if startedAt != nil {
		updates["started_at"] = startedAt
	}
	if lastDryRunAt != nil {
		updates["last_dry_run_at"] = lastDryRunAt
	}
	if lastMigrationAttemptAt != nil {
		updates["last_migration_attempt_at"] = lastMigrationAttemptAt
	}

	result := d.db.WithContext(ctx).Model(&models.Batch{}).
		Where("id = ?", batchID).
		Updates(updates)

	return result.Error
}

// DeleteRepository deletes a repository by full name using GORM
func (d *Database) DeleteRepository(ctx context.Context, fullName string) error {
	result := d.db.WithContext(ctx).Where("full_name = ?", fullName).Delete(&models.Repository{})
	return result.Error
}

// CountRepositories returns the total count of repositories with optional filters using GORM
func (d *Database) CountRepositories(ctx context.Context, filters map[string]interface{}) (int, error) {
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

	// Calculate complexity score
	_ = d.populateComplexityScores(ctx, []*models.Repository{&repo})

	return &repo, nil
}

// GetMigrationHistory retrieves migration history for a repository using GORM
func (d *Database) GetMigrationHistory(ctx context.Context, repoID int64) ([]*models.MigrationHistory, error) {
	var history []*models.MigrationHistory
	err := d.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("started_at DESC").
		Find(&history).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}

	return history, nil
}

// GetMigrationLogs retrieves detailed logs for a repository's migration operations
func (d *Database) GetMigrationLogs(ctx context.Context, repoID int64, level, phase string, limit, offset int) ([]*models.MigrationLog, error) {
	var logs []*models.MigrationLog
	query := d.db.WithContext(ctx).Where("repository_id = ?", repoID)

	// Add optional filters
	if level != "" {
		query = query.Where("level = ?", level)
	}
	if phase != "" {
		query = query.Where("phase = ?", phase)
	}

	err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration logs: %w", err)
	}

	return logs, nil
}

// GetBatch retrieves a batch by ID using GORM
func (d *Database) GetBatch(ctx context.Context, id int64) (*models.Batch, error) {
	var batch models.Batch
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&batch).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	return &batch, nil
}

// UpdateBatch updates a batch using GORM
func (d *Database) UpdateBatch(ctx context.Context, batch *models.Batch) error {
	result := d.db.WithContext(ctx).Save(batch)
	return result.Error
}

// ListBatches retrieves all batches using GORM
func (d *Database) ListBatches(ctx context.Context) ([]*models.Batch, error) {
	var batches []*models.Batch
	err := d.db.WithContext(ctx).Order("created_at DESC").Find(&batches).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	return batches, nil
}

// CreateBatch creates a new batch using GORM
func (d *Database) CreateBatch(ctx context.Context, batch *models.Batch) error {
	// Set default migration API if not specified
	if batch.MigrationAPI == "" {
		batch.MigrationAPI = models.MigrationAPIGEI
	}

	result := d.db.WithContext(ctx).Create(batch)
	if result.Error != nil {
		return fmt.Errorf("failed to create batch: %w", result.Error)
	}

	return nil
}

// DeleteBatch deletes a batch and clears batch_id from all associated repositories using GORM
func (d *Database) DeleteBatch(ctx context.Context, batchID int64) error {
	// Use GORM transaction
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear batch_id from all repositories in this batch
		now := time.Now().UTC()
		result := tx.Model(&models.Repository{}).
			Where("batch_id = ?", batchID).
			Updates(map[string]interface{}{
				"batch_id":   nil,
				"updated_at": now,
			})
		if result.Error != nil {
			return fmt.Errorf("failed to clear batch from repositories: %w", result.Error)
		}

		// Delete the batch
		result = tx.Delete(&models.Batch{}, batchID)
		if result.Error != nil {
			return fmt.Errorf("failed to delete batch: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("batch not found")
		}

		return nil
	})
}

// scanRepositories is a legacy helper that is no longer used with GORM
// All repository queries now use GORM's automatic scanning
// This function is kept for reference but should not be called

// CreateMigrationHistory creates a new migration history record using GORM
func (d *Database) CreateMigrationHistory(ctx context.Context, history *models.MigrationHistory) (int64, error) {
	result := d.db.WithContext(ctx).Create(history)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to create migration history: %w", result.Error)
	}

	return history.ID, nil
}

// UpdateMigrationHistory updates a migration history record using GORM
func (d *Database) UpdateMigrationHistory(ctx context.Context, id int64, status string, errorMsg *string) error {
	completedAt := time.Now()

	// Get the started_at time to calculate duration
	var history models.MigrationHistory
	err := d.db.WithContext(ctx).Select("started_at").Where("id = ?", id).First(&history).Error
	if err != nil {
		return fmt.Errorf("failed to get started_at time: %w", err)
	}

	durationSeconds := int(completedAt.Sub(history.StartedAt).Seconds())

	// Update the record
	result := d.db.WithContext(ctx).Model(&models.MigrationHistory{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":           status,
		"error_message":    errorMsg,
		"completed_at":     completedAt,
		"duration_seconds": durationSeconds,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update migration history: %w", result.Error)
	}

	return nil
}

// CreateMigrationLog creates a new migration log entry using GORM
func (d *Database) CreateMigrationLog(ctx context.Context, log *models.MigrationLog) error {
	result := d.db.WithContext(ctx).Create(log)
	if result.Error != nil {
		return fmt.Errorf("failed to create migration log: %w", result.Error)
	}

	return nil
}

// AddRepositoriesToBatch assigns multiple repositories to a batch
//
//nolint:dupl // Similar to RemoveRepositoriesFromBatch but performs different operations
func (d *Database) AddRepositoriesToBatch(ctx context.Context, batchID int64, repoIDs []int64) error {
	if len(repoIDs) == 0 {
		return nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(repoIDs))
	args := make([]interface{}, len(repoIDs)+1)
	args[0] = batchID
	for i, id := range repoIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}

	// Use GORM to update batch_id for specified repositories
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("id IN ?", repoIDs).
		Updates(map[string]interface{}{
			"batch_id":   batchID,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to add repositories to batch: %w", result.Error)
	}

	// Update batch repository count and status
	if result.RowsAffected > 0 {
		if err := d.updateBatchRepositoryCount(ctx, batchID); err != nil {
			return err
		}
		// Recalculate batch status based on repository dry run readiness
		if err := d.UpdateBatchStatus(ctx, batchID); err != nil {
			return err
		}
	}

	return nil
}

// RemoveRepositoriesFromBatch removes repositories from a batch using GORM
func (d *Database) RemoveRepositoriesFromBatch(ctx context.Context, batchID int64, repoIDs []int64) error {
	if len(repoIDs) == 0 {
		return nil
	}

	// Use GORM to clear batch_id for specified repositories
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("batch_id = ? AND id IN ?", batchID, repoIDs).
		Updates(map[string]interface{}{
			"batch_id":   nil,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to remove repositories from batch: %w", result.Error)
	}

	// Update batch repository count and status
	rowsAffected := result.RowsAffected
	if rowsAffected > 0 {
		if err := d.updateBatchRepositoryCount(ctx, batchID); err != nil {
			return err
		}
		// Recalculate batch status after removal
		if err := d.UpdateBatchStatus(ctx, batchID); err != nil {
			return err
		}
	}

	return nil
}

// updateBatchRepositoryCount updates the repository count for a batch
func (d *Database) updateBatchRepositoryCount(ctx context.Context, batchID int64) error {
	query := `
		UPDATE batches 
		SET repository_count = (
			SELECT COUNT(*) FROM repositories WHERE batch_id = ?
		)
		WHERE id = ?
	`

	// Use GORM Raw() for complex query with subquery
	err := d.db.WithContext(ctx).Exec(query, batchID, batchID).Error
	if err != nil {
		return fmt.Errorf("failed to update batch repository count: %w", err)
	}

	return nil
}

// UpdateBatchStatus recalculates and updates the batch status based on repository statuses
// Batch is 'ready' only if ALL repositories have completed dry runs
// Batch is 'pending' if ANY repository hasn't completed a dry run
func (d *Database) UpdateBatchStatus(ctx context.Context, batchID int64) error {
	// Get all repositories in the batch
	repos, err := d.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		return fmt.Errorf("failed to list batch repositories: %w", err)
	}

	// Get current batch to check if it's in a terminal or active state
	batch, err := d.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch not found")
	}

	// Don't update status if batch is in progress or completed
	terminalStates := []string{"in_progress", "completed", "completed_with_errors", "failed", "cancelled"}
	for _, state := range terminalStates {
		if batch.Status == state {
			// Batch is actively running or finished, don't change status
			return nil
		}
	}

	// Calculate new status based on repository dry run status
	newStatus := calculateBatchReadiness(repos)

	// Only update if status changed
	if newStatus != batch.Status {
		result := d.db.WithContext(ctx).Model(&models.Batch{}).Where("id = ?", batchID).Update("status", newStatus)
		if result.Error != nil {
			return fmt.Errorf("failed to update batch status: %w", result.Error)
		}
	}

	return nil
}

// calculateBatchReadiness determines if a batch should be 'ready' or 'pending'
// based on the dry run status of its repositories
func calculateBatchReadiness(repos []*models.Repository) string {
	if len(repos) == 0 {
		return batchStatusPending
	}

	allDryRunComplete := true
	for _, repo := range repos {
		// Repository needs dry run if it's in any of these states
		needsDryRun := repo.Status == string(models.StatusPending) ||
			repo.Status == string(models.StatusDryRunFailed) ||
			repo.Status == string(models.StatusMigrationFailed) ||
			repo.Status == string(models.StatusRolledBack)

		if needsDryRun {
			allDryRunComplete = false
			break
		}

		// If repo is not dry_run_complete, it also needs dry run
		if repo.Status != string(models.StatusDryRunComplete) {
			allDryRunComplete = false
			break
		}
	}

	if allDryRunComplete {
		return batchStatusReady
	}
	return batchStatusPending
}

// OrganizationStats represents statistics for a single organization
type OrganizationStats struct {
	Organization string         `json:"organization"`
	TotalRepos   int            `json:"total_repos"`
	StatusCounts map[string]int `json:"status_counts"`
}

// GetOrganizationStats returns repository counts grouped by organization
func (d *Database) GetOrganizationStats(ctx context.Context) ([]*OrganizationStats, error) {
	// Use dialect-specific string functions
	var query string
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, POSITION('/' IN full_name) - 1) as org,
				COUNT(*) as total,
				status,
				COUNT(*) as status_count
			FROM repositories
			WHERE POSITION('/' IN full_name) > 0
			AND status != 'wont_migrate'
			GROUP BY org, status
			ORDER BY total DESC, org ASC
		`
	case DBTypeSQLServer, DBTypeMSSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, CHARINDEX('/', full_name) - 1) as org,
				COUNT(*) as total,
				status,
				COUNT(*) as status_count
			FROM repositories
			WHERE CHARINDEX('/', full_name) > 0
			AND status != 'wont_migrate'
			GROUP BY org, status
			ORDER BY total DESC, org ASC
		`
	default: // SQLite
		query = `
			SELECT 
				SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as org,
				COUNT(*) as total,
				status,
				COUNT(*) as status_count
			FROM repositories
			WHERE INSTR(full_name, '/') > 0
			AND status != 'wont_migrate'
			GROUP BY org, status
			ORDER BY total DESC, org ASC
		`
	}

	// Use GORM Raw() for analytics query
	type OrgStatusResult struct {
		Org         string
		Total       int
		Status      string
		StatusCount int
	}

	var results []OrgStatusResult
	err := d.db.WithContext(ctx).Raw(query).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get organization stats: %w", err)
	}

	// Build organization stats map
	orgMap := make(map[string]*OrganizationStats)
	for _, result := range results {
		if _, exists := orgMap[result.Org]; !exists {
			orgMap[result.Org] = &OrganizationStats{
				Organization: result.Org,
				TotalRepos:   0,
				StatusCounts: make(map[string]int),
			}
		}

		orgMap[result.Org].StatusCounts[result.Status] = result.StatusCount
		orgMap[result.Org].TotalRepos += result.StatusCount
	}

	// Convert map to slice
	stats := make([]*OrganizationStats, 0, len(orgMap))
	for _, stat := range orgMap {
		stats = append(stats, stat)
	}

	// Sort by total repos (descending), then by organization name (ascending) for consistent ordering
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalRepos == stats[j].TotalRepos {
			return stats[i].Organization < stats[j].Organization
		}
		return stats[i].TotalRepos > stats[j].TotalRepos
	})

	return stats, nil
}

// SizeDistribution represents repository size distribution
type SizeDistribution struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetSizeDistribution categorizes repositories by size
func (d *Database) GetSizeDistribution(ctx context.Context) ([]*SizeDistribution, error) {
	// Size categories: small (<100MB), medium (100MB-1GB), large (1GB-5GB), very_large (>5GB)
	// Note: PostgreSQL doesn't allow GROUP BY on column aliases, so we use a subquery
	query := `
		SELECT 
			category,
			COUNT(*) as count
		FROM (
			SELECT 
				CASE 
					WHEN total_size IS NULL THEN 'unknown'
					WHEN total_size < 104857600 THEN 'small'
					WHEN total_size < 1073741824 THEN 'medium'
					WHEN total_size < 5368709120 THEN 'large'
					ELSE 'very_large'
				END as category
			FROM repositories
		) categorized
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'small' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'large' THEN 3
				WHEN 'very_large' THEN 4
				WHEN 'unknown' THEN 5
			END
	`

	// Use GORM Raw() for analytics query
	var distribution []*SizeDistribution
	err := d.db.WithContext(ctx).Raw(query).Scan(&distribution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get size distribution: %w", err)
	}

	return distribution, nil
}

// FeatureStats represents aggregated feature usage statistics
type FeatureStats struct {
	IsArchived           int `json:"is_archived" gorm:"column:archived_count"`
	IsFork               int `json:"is_fork" gorm:"column:fork_count"`
	HasLFS               int `json:"has_lfs" gorm:"column:lfs_count"`
	HasSubmodules        int `json:"has_submodules" gorm:"column:submodules_count"`
	HasLargeFiles        int `json:"has_large_files" gorm:"column:large_files_count"`
	HasWiki              int `json:"has_wiki" gorm:"column:wiki_count"`
	HasPages             int `json:"has_pages" gorm:"column:pages_count"`
	HasDiscussions       int `json:"has_discussions" gorm:"column:discussions_count"`
	HasActions           int `json:"has_actions" gorm:"column:actions_count"`
	HasProjects          int `json:"has_projects" gorm:"column:projects_count"`
	HasPackages          int `json:"has_packages" gorm:"column:packages_count"`
	HasBranchProtections int `json:"has_branch_protections" gorm:"column:branch_protections_count"`
	HasRulesets          int `json:"has_rulesets" gorm:"column:rulesets_count"`
	HasCodeScanning      int `json:"has_code_scanning" gorm:"column:code_scanning_count"`
	HasDependabot        int `json:"has_dependabot" gorm:"column:dependabot_count"`
	HasSecretScanning    int `json:"has_secret_scanning" gorm:"column:secret_scanning_count"`
	HasCodeowners        int `json:"has_codeowners" gorm:"column:codeowners_count"`
	HasSelfHostedRunners int `json:"has_self_hosted_runners" gorm:"column:self_hosted_runners_count"`
	HasReleaseAssets     int `json:"has_release_assets" gorm:"column:release_assets_count"`
	HasWebhooks          int `json:"has_webhooks" gorm:"column:webhooks_count"`
	TotalRepositories    int `json:"total_repositories" gorm:"column:total"`
}

// GetFeatureStats returns aggregated statistics on feature usage
func (d *Database) GetFeatureStats(ctx context.Context) (*FeatureStats, error) {
	query := `
		SELECT 
			SUM(CASE WHEN is_archived = TRUE THEN 1 ELSE 0 END) as archived_count,
			SUM(CASE WHEN is_fork = TRUE THEN 1 ELSE 0 END) as fork_count,
			SUM(CASE WHEN has_lfs = TRUE THEN 1 ELSE 0 END) as lfs_count,
			SUM(CASE WHEN has_submodules = TRUE THEN 1 ELSE 0 END) as submodules_count,
			SUM(CASE WHEN has_large_files = TRUE THEN 1 ELSE 0 END) as large_files_count,
			SUM(CASE WHEN has_wiki = TRUE THEN 1 ELSE 0 END) as wiki_count,
			SUM(CASE WHEN has_pages = TRUE THEN 1 ELSE 0 END) as pages_count,
			SUM(CASE WHEN has_discussions = TRUE THEN 1 ELSE 0 END) as discussions_count,
			SUM(CASE WHEN has_actions = TRUE THEN 1 ELSE 0 END) as actions_count,
			SUM(CASE WHEN has_projects = TRUE THEN 1 ELSE 0 END) as projects_count,
			SUM(CASE WHEN has_packages = TRUE THEN 1 ELSE 0 END) as packages_count,
			SUM(CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END) as branch_protections_count,
			SUM(CASE WHEN has_rulesets = TRUE THEN 1 ELSE 0 END) as rulesets_count,
			SUM(CASE WHEN has_code_scanning = TRUE THEN 1 ELSE 0 END) as code_scanning_count,
			SUM(CASE WHEN has_dependabot = TRUE THEN 1 ELSE 0 END) as dependabot_count,
			SUM(CASE WHEN has_secret_scanning = TRUE THEN 1 ELSE 0 END) as secret_scanning_count,
		SUM(CASE WHEN has_codeowners = TRUE THEN 1 ELSE 0 END) as codeowners_count,
		SUM(CASE WHEN has_self_hosted_runners = TRUE THEN 1 ELSE 0 END) as self_hosted_runners_count,
		SUM(CASE WHEN has_release_assets = TRUE THEN 1 ELSE 0 END) as release_assets_count,
		SUM(CASE WHEN webhook_count > 0 THEN 1 ELSE 0 END) as webhooks_count,
		COUNT(*) as total
	FROM repositories
`

	// Use GORM Raw() for analytics query
	var stats FeatureStats
	err := d.db.WithContext(ctx).Raw(query).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get feature stats: %w", err)
	}

	return &stats, nil
}

// MigrationCompletionStats represents migration completion stats by organization
type MigrationCompletionStats struct {
	Organization    string `json:"organization"`
	TotalRepos      int    `json:"total_repos"`
	CompletedCount  int    `json:"completed_count"`
	InProgressCount int    `json:"in_progress_count"`
	PendingCount    int    `json:"pending_count"`
	FailedCount     int    `json:"failed_count"`
}

// CompletedMigration represents a completed migration for the history page
type CompletedMigration struct {
	ID              int64      `json:"id"`
	FullName        string     `json:"full_name"`
	SourceURL       string     `json:"source_url"`
	DestinationURL  *string    `json:"destination_url"`
	Status          string     `json:"status"`
	MigratedAt      *time.Time `json:"migrated_at"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	DurationSeconds *int       `json:"duration_seconds"`
}

// GetCompletedMigrations returns all completed, failed, and rolled back migrations
func (d *Database) GetCompletedMigrations(ctx context.Context) ([]*CompletedMigration, error) {
	query := `
		SELECT 
			r.id,
			r.full_name,
			r.source_url,
			r.destination_url,
			r.status,
			r.migrated_at,
			h.started_at as started_at_str,
			h.completed_at as completed_at_str,
			h.duration_seconds
		FROM repositories r
		LEFT JOIN (
			SELECT 
				repository_id,
				MIN(started_at) as started_at,
				MAX(completed_at) as completed_at,
				SUM(duration_seconds) as duration_seconds
			FROM migration_history
			WHERE phase IN ('migration', 'rollback') 
			GROUP BY repository_id
		) h ON r.id = h.repository_id
		WHERE r.status IN ('complete', 'migration_failed', 'rolled_back')
		ORDER BY r.migrated_at DESC, r.updated_at DESC
	`

	// Use a temporary struct to handle SQLite string datetime values
	type tempMigration struct {
		ID              int64      `gorm:"column:id"`
		FullName        string     `gorm:"column:full_name"`
		SourceURL       string     `gorm:"column:source_url"`
		DestinationURL  *string    `gorm:"column:destination_url"`
		Status          string     `gorm:"column:status"`
		MigratedAt      *time.Time `gorm:"column:migrated_at"`
		StartedAtStr    *string    `gorm:"column:started_at_str"`
		CompletedAtStr  *string    `gorm:"column:completed_at_str"`
		DurationSeconds *int       `gorm:"column:duration_seconds"`
	}

	var tempMigrations []tempMigration
	err := d.db.WithContext(ctx).Raw(query).Scan(&tempMigrations).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get completed migrations: %w", err)
	}

	// Convert to CompletedMigration with proper time parsing
	migrations := make([]*CompletedMigration, len(tempMigrations))
	for i, temp := range tempMigrations {
		migrations[i] = &CompletedMigration{
			ID:              temp.ID,
			FullName:        temp.FullName,
			SourceURL:       temp.SourceURL,
			DestinationURL:  temp.DestinationURL,
			Status:          temp.Status,
			MigratedAt:      temp.MigratedAt,
			DurationSeconds: temp.DurationSeconds,
		}

		// Parse started_at string to time.Time
		if temp.StartedAtStr != nil && *temp.StartedAtStr != "" {
			// Try multiple datetime formats for cross-database compatibility
			formats := []string{
				"2006-01-02 15:04:05.999999-07:00",        // SQLite with microseconds and full timezone offset
				"2006-01-02 15:04:05.999999-07",           // PostgreSQL with short timezone offset
				"2006-01-02 15:04:05.9999999",             // SQL Server with 7 fractional digits (no timezone)
				"2006-01-02 15:04:05.999999",              // Most databases with microseconds (no timezone)
				"2006-01-02 15:04:05.999999999-07:00",     // Nanoseconds with full timezone
				"2006-01-02 15:04:05.999999999-07",        // Nanoseconds with short timezone
				"2006-01-02 15:04:05.999999999 -0700 MST", // Go's default format
				"2006-01-02 15:04:05",                     // Basic format without fractional seconds or timezone
				time.RFC3339,                              // ISO 8601 (2006-01-02T15:04:05Z07:00)
				"2006-01-02T15:04:05.999999-07:00",        // ISO 8601 with microseconds
				"2006-01-02T15:04:05",                     // ISO 8601 without timezone
			}
			for _, format := range formats {
				if t, err := time.Parse(format, *temp.StartedAtStr); err == nil {
					migrations[i].StartedAt = &t
					break
				}
			}
		}

		// Parse completed_at string to time.Time
		if temp.CompletedAtStr != nil && *temp.CompletedAtStr != "" {
			// Try multiple datetime formats for cross-database compatibility
			formats := []string{
				"2006-01-02 15:04:05.999999-07:00",        // SQLite with microseconds and full timezone offset
				"2006-01-02 15:04:05.999999-07",           // PostgreSQL with short timezone offset
				"2006-01-02 15:04:05.9999999",             // SQL Server with 7 fractional digits (no timezone)
				"2006-01-02 15:04:05.999999",              // Most databases with microseconds (no timezone)
				"2006-01-02 15:04:05.999999999-07:00",     // Nanoseconds with full timezone
				"2006-01-02 15:04:05.999999999-07",        // Nanoseconds with short timezone
				"2006-01-02 15:04:05.999999999 -0700 MST", // Go's default format
				"2006-01-02 15:04:05",                     // Basic format without fractional seconds or timezone
				time.RFC3339,                              // ISO 8601 (2006-01-02T15:04:05Z07:00)
				"2006-01-02T15:04:05.999999-07:00",        // ISO 8601 with microseconds
				"2006-01-02T15:04:05",                     // ISO 8601 without timezone
			}
			for _, format := range formats {
				if t, err := time.Parse(format, *temp.CompletedAtStr); err == nil {
					migrations[i].CompletedAt = &t
					break
				}
			}
		}
	}

	return migrations, nil
}

// GetMigrationCompletionStatsByOrg returns migration completion stats grouped by organization
func (d *Database) GetMigrationCompletionStatsByOrg(ctx context.Context) ([]*MigrationCompletionStats, error) {
	// Use dialect-specific string functions
	var query string
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, POSITION('/' IN full_name) - 1) as organization,
				COUNT(*) as total_repos,
				SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
				SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
				SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
				SUM(CASE WHEN status LIKE '%failed%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
			FROM repositories
			WHERE full_name LIKE '%/%'
			AND status != 'wont_migrate'
			GROUP BY organization
			ORDER BY total_repos DESC
		`
	case DBTypeSQLServer, DBTypeMSSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, CHARINDEX('/', full_name) - 1) as organization,
				COUNT(*) as total_repos,
				SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
				SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
				SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
				SUM(CASE WHEN status LIKE '%failed%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
			FROM repositories
			WHERE full_name LIKE '%/%'
			AND status != 'wont_migrate'
			GROUP BY organization
			ORDER BY total_repos DESC
		`
	default: // SQLite
		query = `
			SELECT 
				SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as organization,
				COUNT(*) as total_repos,
				SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
				SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
				SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
				SUM(CASE WHEN status LIKE '%failed%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
			FROM repositories
			WHERE full_name LIKE '%/%'
			AND status != 'wont_migrate'
			GROUP BY organization
			ORDER BY total_repos DESC
		`
	}

	// Use GORM Raw() for analytics query
	var stats []*MigrationCompletionStats
	err := d.db.WithContext(ctx).Raw(query).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration completion stats: %w", err)
	}

	return stats, nil
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
		Updates(map[string]interface{}{
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

// ComplexityDistribution represents repository complexity distribution
type ComplexityDistribution struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetComplexityDistribution categorizes repositories by complexity score
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetComplexityDistribution(ctx context.Context, orgFilter, batchFilter string) ([]*ComplexityDistribution, error) {
	// Calculate complexity score using GitHub-specific formula with activity quantiles
	// This matches the calculation in applyComplexityFilter()

	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Note: PostgreSQL doesn't allow GROUP BY on column aliases, so we use nested subqueries
	query := `
		SELECT 
			category,
			COUNT(*) as count
		FROM (
			SELECT 
				CASE 
					WHEN complexity_score <= 5 THEN 'simple'
					WHEN complexity_score <= 10 THEN 'medium'
					WHEN complexity_score <= 17 THEN 'complex'
					ELSE 'very_complex'
				END as category
			FROM (
				SELECT ` + d.buildGitHubComplexityScoreSQL() + ` as complexity_score
				FROM repositories r
				WHERE 1=1
					AND status != 'wont_migrate'
					` + orgFilterSQL + `
					` + batchFilterSQL + `
			) as scored_repos
		) as categorized
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'simple' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'complex' THEN 3
				WHEN 'very_complex' THEN 4
			END
	`

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	var distribution []*ComplexityDistribution
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&distribution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get complexity distribution: %w", err)
	}

	return distribution, nil
}

// MigrationVelocity represents migration velocity metrics
type MigrationVelocity struct {
	ReposPerDay  float64 `json:"repos_per_day"`
	ReposPerWeek float64 `json:"repos_per_week"`
}

// GetMigrationVelocity calculates migration velocity over the specified period
func (d *Database) GetMigrationVelocity(ctx context.Context, orgFilter, batchFilter string, days int) (*MigrationVelocity, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Use dialect-specific date arithmetic
	var dateCondition string
	var args []interface{}
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		dateCondition = fmt.Sprintf("AND mh.completed_at >= NOW() - INTERVAL '%d days'", days)
		args = append(args, orgArgs...)
		args = append(args, batchArgs...)
	case DBTypeSQLServer, DBTypeMSSQL:
		dateCondition = fmt.Sprintf("AND mh.completed_at >= DATEADD(day, -%d, GETUTCDATE())", days)
		args = append(args, orgArgs...)
		args = append(args, batchArgs...)
	default: // SQLite
		dateCondition = "AND mh.completed_at >= datetime('now', '-' || ? || ' days')"
		args = []interface{}{days}
		args = append(args, orgArgs...)
		args = append(args, batchArgs...)
	}

	query := `
		SELECT COUNT(DISTINCT r.id) as total_completed
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed' 
			AND mh.phase = 'migration'
			` + dateCondition + `
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + batchFilterSQL + `
	`

	// Use GORM Raw() for analytics query
	var result struct {
		TotalCompleted int
	}
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration velocity: %w", err)
	}

	velocity := &MigrationVelocity{
		ReposPerDay:  float64(result.TotalCompleted) / float64(days),
		ReposPerWeek: (float64(result.TotalCompleted) / float64(days)) * 7,
	}

	return velocity, nil
}

// MigrationTimeSeriesPoint represents a point in the migration time series
type MigrationTimeSeriesPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetMigrationTimeSeries returns daily migration completions for the last 30 days
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetMigrationTimeSeries(ctx context.Context, orgFilter, batchFilter string) ([]*MigrationTimeSeriesPoint, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Use dialect-specific date arithmetic
	var dateCondition string
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		dateCondition = "AND mh.completed_at >= NOW() - INTERVAL '30 days'"
	case DBTypeSQLServer, DBTypeMSSQL:
		dateCondition = "AND mh.completed_at >= DATEADD(day, -30, GETUTCDATE())"
	default: // SQLite
		dateCondition = "AND mh.completed_at >= datetime('now', '-30 days')"
	}

	query := `
		SELECT 
			DATE(mh.completed_at) as date,
			COUNT(DISTINCT r.id) as count
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed'
			AND mh.phase = 'migration'
			` + dateCondition + `
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + batchFilterSQL + `
		GROUP BY DATE(mh.completed_at)
		ORDER BY date ASC
	`

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	var series []*MigrationTimeSeriesPoint
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&series).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration time series: %w", err)
	}

	return series, nil
}

// GetAverageMigrationTime calculates the average migration duration
func (d *Database) GetAverageMigrationTime(ctx context.Context, orgFilter, batchFilter string) (float64, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	query := `
		SELECT AVG(mh.duration_seconds) as avg_duration
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed'
			AND mh.phase = 'migration'
			AND mh.duration_seconds IS NOT NULL
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + batchFilterSQL + `
	`

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	var result struct {
		AvgDuration *float64
	}
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&result).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get average migration time: %w", err)
	}

	if result.AvgDuration == nil {
		return 0, nil
	}

	return *result.AvgDuration, nil
}

// buildOrgFilter builds the organization filter clause using parameterized queries
// Returns the SQL fragment and any additional arguments to append
func (d *Database) buildOrgFilter(orgFilter string) (string, []interface{}) {
	if orgFilter == "" {
		return "", nil
	}

	// Use dialect-specific string functions
	var filterSQL string
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		filterSQL = " AND SUBSTRING(r.full_name, 1, POSITION('/' IN r.full_name) - 1) = ?"
	case DBTypeSQLServer, DBTypeMSSQL:
		filterSQL = " AND SUBSTRING(r.full_name, 1, CHARINDEX('/', r.full_name) - 1) = ?"
	default: // SQLite
		filterSQL = " AND SUBSTR(r.full_name, 1, INSTR(r.full_name, '/') - 1) = ?"
	}

	return filterSQL, []interface{}{orgFilter}
}

// buildBatchFilter builds the batch filter clause using parameterized queries
// Returns the SQL fragment and any additional arguments to append
func (d *Database) buildBatchFilter(batchFilter string) (string, []interface{}) {
	if batchFilter == "" {
		return "", nil
	}
	// Validate that batchFilter contains only digits
	batchID, err := strconv.ParseInt(batchFilter, 10, 64)
	if err != nil {
		return "", nil
	}
	return " AND r.batch_id = ?", []interface{}{batchID}
}

// GetRepositoryStatsByStatusFiltered returns repository counts by status with filters
func (d *Database) GetRepositoryStatsByStatusFiltered(ctx context.Context, orgFilter, batchFilter string) (map[string]int, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	query := `
		SELECT status, COUNT(*) as count
		FROM repositories r
		WHERE 1=1
			` + orgFilterSQL + `
			` + batchFilterSQL + `
		GROUP BY status
	`

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	type StatusCount struct {
		Status string
		Count  int
	}

	var results []StatusCount
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get repository stats: %w", err)
	}

	stats := make(map[string]int)
	for _, result := range results {
		stats[result.Status] = result.Count
	}

	return stats, nil
}

// GetSizeDistributionFiltered returns size distribution with filters
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetSizeDistributionFiltered(ctx context.Context, orgFilter, batchFilter string) ([]*SizeDistribution, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Note: PostgreSQL doesn't allow GROUP BY on column aliases, so we use a subquery
	query := `
		SELECT 
			category,
			COUNT(*) as count
		FROM (
			SELECT 
				CASE 
					WHEN total_size IS NULL THEN 'unknown'
					WHEN total_size < 104857600 THEN 'small'
					WHEN total_size < 1073741824 THEN 'medium'
					WHEN total_size < 5368709120 THEN 'large'
					ELSE 'very_large'
				END as category
			FROM repositories r
			WHERE 1=1
				AND status != 'wont_migrate'
				` + orgFilterSQL + `
				` + batchFilterSQL + `
		) categorized
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'small' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'large' THEN 3
				WHEN 'very_large' THEN 4
				WHEN 'unknown' THEN 5
			END
	`

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	var distribution []*SizeDistribution
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&distribution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get size distribution: %w", err)
	}

	return distribution, nil
}

// GetFeatureStatsFiltered returns feature stats with filters
func (d *Database) GetFeatureStatsFiltered(ctx context.Context, orgFilter, batchFilter string) (*FeatureStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	query := `
		SELECT 
			SUM(CASE WHEN is_archived = TRUE THEN 1 ELSE 0 END) as archived_count,
			SUM(CASE WHEN is_fork = TRUE THEN 1 ELSE 0 END) as fork_count,
			SUM(CASE WHEN has_lfs = TRUE THEN 1 ELSE 0 END) as lfs_count,
			SUM(CASE WHEN has_submodules = TRUE THEN 1 ELSE 0 END) as submodules_count,
			SUM(CASE WHEN has_large_files = TRUE THEN 1 ELSE 0 END) as large_files_count,
			SUM(CASE WHEN has_wiki = TRUE THEN 1 ELSE 0 END) as wiki_count,
			SUM(CASE WHEN has_pages = TRUE THEN 1 ELSE 0 END) as pages_count,
			SUM(CASE WHEN has_discussions = TRUE THEN 1 ELSE 0 END) as discussions_count,
			SUM(CASE WHEN has_actions = TRUE THEN 1 ELSE 0 END) as actions_count,
			SUM(CASE WHEN has_projects = TRUE THEN 1 ELSE 0 END) as projects_count,
			SUM(CASE WHEN has_packages = TRUE THEN 1 ELSE 0 END) as packages_count,
			SUM(CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END) as branch_protections_count,
			SUM(CASE WHEN has_rulesets = TRUE THEN 1 ELSE 0 END) as rulesets_count,
			SUM(CASE WHEN has_code_scanning = TRUE THEN 1 ELSE 0 END) as code_scanning_count,
			SUM(CASE WHEN has_dependabot = TRUE THEN 1 ELSE 0 END) as dependabot_count,
			SUM(CASE WHEN has_secret_scanning = TRUE THEN 1 ELSE 0 END) as secret_scanning_count,
		SUM(CASE WHEN has_codeowners = TRUE THEN 1 ELSE 0 END) as codeowners_count,
		SUM(CASE WHEN has_self_hosted_runners = TRUE THEN 1 ELSE 0 END) as self_hosted_runners_count,
		SUM(CASE WHEN has_release_assets = TRUE THEN 1 ELSE 0 END) as release_assets_count,
		SUM(CASE WHEN webhook_count > 0 THEN 1 ELSE 0 END) as webhooks_count,
		COUNT(*) as total
	FROM repositories r
	WHERE 1=1
		AND status != 'wont_migrate'
		` + orgFilterSQL + `
		` + batchFilterSQL + `
`

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	var stats FeatureStats
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get feature stats: %w", err)
	}

	return &stats, nil
}

// GetOrganizationStatsFiltered returns organization stats with batch filter
func (d *Database) GetOrganizationStatsFiltered(ctx context.Context, orgFilter, batchFilter string) ([]*OrganizationStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Use dialect-specific string functions
	var query string
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, POSITION('/' IN full_name) - 1) as org,
				COUNT(*) as total,
				status,
				COUNT(*) as status_count
			FROM repositories r
			WHERE POSITION('/' IN full_name) > 0
				AND status != 'wont_migrate'
				` + orgFilterSQL + `
				` + batchFilterSQL + `
			GROUP BY org, status
			ORDER BY total DESC, org ASC
		`
	case DBTypeSQLServer, DBTypeMSSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, CHARINDEX('/', full_name) - 1) as org,
				COUNT(*) as total,
				status,
				COUNT(*) as status_count
			FROM repositories r
			WHERE CHARINDEX('/', full_name) > 0
				AND status != 'wont_migrate'
				` + orgFilterSQL + `
				` + batchFilterSQL + `
			GROUP BY org, status
			ORDER BY total DESC, org ASC
		`
	default: // SQLite
		query = `
			SELECT 
				SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as org,
				COUNT(*) as total,
				status,
				COUNT(*) as status_count
			FROM repositories r
			WHERE INSTR(full_name, '/') > 0
				AND status != 'wont_migrate'
				` + orgFilterSQL + `
				` + batchFilterSQL + `
			GROUP BY org, status
			ORDER BY total DESC, org ASC
		`
	}

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	type OrgStatusResult struct {
		Org         string
		Total       int
		Status      string
		StatusCount int
	}

	var results []OrgStatusResult
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get organization stats: %w", err)
	}

	orgMap := make(map[string]*OrganizationStats)
	for _, result := range results {
		if _, exists := orgMap[result.Org]; !exists {
			orgMap[result.Org] = &OrganizationStats{
				Organization: result.Org,
				TotalRepos:   0,
				StatusCounts: make(map[string]int),
			}
		}

		orgMap[result.Org].StatusCounts[result.Status] = result.StatusCount
		orgMap[result.Org].TotalRepos += result.StatusCount
	}

	stats := make([]*OrganizationStats, 0, len(orgMap))
	for _, stat := range orgMap {
		stats = append(stats, stat)
	}

	return stats, nil
}

// UpdateRepositoryValidation updates the validation fields for a repository using GORM
func (d *Database) UpdateRepositoryValidation(ctx context.Context, fullName string, validationStatus string, validationDetails, destinationData *string) error {
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&models.Repository{}).
		Where("full_name = ?", fullName).
		Updates(map[string]interface{}{
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

// GetMigrationCompletionStatsByOrgFiltered returns migration completion stats with org and batch filters
//
//nolint:dupl // Similar to GetMigrationCompletionStatsByOrg but with filters
func (d *Database) GetMigrationCompletionStatsByOrgFiltered(ctx context.Context, orgFilter, batchFilter string) ([]*MigrationCompletionStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Use dialect-specific string functions
	var query string
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, POSITION('/' IN full_name) - 1) as organization,
				COUNT(*) as total_repos,
				SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
				SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
				SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
				SUM(CASE WHEN status LIKE '%failed%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
			FROM repositories r
			WHERE full_name LIKE '%/%'
				AND status != 'wont_migrate'
				` + orgFilterSQL + `
				` + batchFilterSQL + `
			GROUP BY organization
			ORDER BY total_repos DESC
		`
	case DBTypeSQLServer, DBTypeMSSQL:
		query = `
			SELECT 
				SUBSTRING(full_name, 1, CHARINDEX('/', full_name) - 1) as organization,
				COUNT(*) as total_repos,
				SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
				SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
				SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
				SUM(CASE WHEN status LIKE '%failed%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
			FROM repositories r
			WHERE full_name LIKE '%/%'
				AND status != 'wont_migrate'
				` + orgFilterSQL + `
				` + batchFilterSQL + `
			GROUP BY organization
			ORDER BY total_repos DESC
		`
	default: // SQLite
		query = `
			SELECT 
				SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as organization,
				COUNT(*) as total_repos,
				SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
				SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
				SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
				SUM(CASE WHEN status LIKE '%failed%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
			FROM repositories r
			WHERE full_name LIKE '%/%'
				AND status != 'wont_migrate'
				` + orgFilterSQL + `
				` + batchFilterSQL + `
			GROUP BY organization
			ORDER BY total_repos DESC
		`
	}

	// Combine all arguments
	args := append(orgArgs, batchArgs...)

	// Use GORM Raw() for analytics query
	var stats []*MigrationCompletionStats
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration completion stats: %w", err)
	}

	return stats, nil
}
