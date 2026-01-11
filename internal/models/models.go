package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Migration API types
const (
	MigrationAPIGEI = "GEI" // GitHub Enterprise Importer
	MigrationAPIELM = "ELM" // Enterprise Live Migrator
)

// Repository represents a Git repository to be migrated.
//
// # Field Organization
//
// This model uses related tables for detailed properties:
//   - GitProperties: size, LFS, submodules, branches, commits (1:1 relationship)
//   - Features: wiki, pages, actions, packages, protections (1:1 relationship)
//   - ADOProperties: project, pipelines, boards (1:1, only for Azure DevOps repos)
//   - Validation: complexity scores, limit violations (1:1 relationship)
//
// The main Repository table is kept narrow for fast list queries.
// Related data is loaded on demand via GORM's Preload.
//
// # Helper Methods
//
//   - IsADORepository(): returns true if this is an Azure DevOps source
//   - HasMigrationBlockers(): returns true if migration blockers exist
//   - NeedsRemediation(): returns true if status is remediation_required
//   - GetTotalSize(), HasLFS(), etc.: convenience accessors for related table fields
type Repository struct {
	ID        int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	FullName  string `json:"full_name" gorm:"uniqueIndex;not null"` // org/repo
	Source    string `json:"source" gorm:"not null"`                // "ghes", "gitlab", etc.
	SourceURL string `json:"source_url" gorm:"not null"`
	SourceID  *int64 `json:"source_id,omitempty" gorm:"index"` // Foreign key to sources table

	// Core status fields (kept in main table for fast filtering)
	Status     string `json:"status" gorm:"not null;index"`
	BatchID    *int64 `json:"batch_id,omitempty" gorm:"index"`
	Priority   int    `json:"priority" gorm:"default:0"`
	Visibility string `json:"visibility"`
	IsArchived bool   `json:"is_archived" gorm:"default:false"`
	IsFork     bool   `json:"is_fork" gorm:"default:false"`

	// Migration destination
	DestinationURL      *string `json:"destination_url,omitempty"`
	DestinationFullName *string `json:"destination_full_name,omitempty"`
	SourceMigrationID   *int64  `json:"source_migration_id,omitempty"`
	IsSourceLocked      bool    `json:"is_source_locked" gorm:"default:false"`

	// Migration exclusions (batch-level overrides)
	ExcludeReleases      bool `json:"exclude_releases" gorm:"default:false"`
	ExcludeAttachments   bool `json:"exclude_attachments" gorm:"default:false"`
	ExcludeMetadata      bool `json:"exclude_metadata" gorm:"default:false"`
	ExcludeGitData       bool `json:"exclude_git_data" gorm:"default:false"`
	ExcludeOwnerProjects bool `json:"exclude_owner_projects" gorm:"default:false"`

	// Timestamps
	DiscoveredAt    time.Time  `json:"discovered_at" gorm:"not null"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime"`
	MigratedAt      *time.Time `json:"migrated_at,omitempty"`
	LastDiscoveryAt *time.Time `json:"last_discovery_at,omitempty"`
	LastDryRunAt    *time.Time `json:"last_dry_run_at,omitempty"`

	// Related tables (loaded on demand via Preload)
	GitProperties *RepositoryGitProperties `json:"git_properties,omitempty" gorm:"foreignKey:RepositoryID"`
	Features      *RepositoryFeatures      `json:"features,omitempty" gorm:"foreignKey:RepositoryID"`
	ADOProperties *RepositoryADOProperties `json:"ado_properties,omitempty" gorm:"foreignKey:RepositoryID"`
	Validation    *RepositoryValidation    `json:"validation,omitempty" gorm:"foreignKey:RepositoryID"`
}

// TableName specifies the table name for Repository model
func (Repository) TableName() string {
	return "repositories"
}

// Convenience accessors for commonly-used fields from related tables

// GetTotalSize returns the total size from git properties, or nil if not loaded
func (r *Repository) GetTotalSize() *int64 {
	if r.GitProperties != nil {
		return r.GitProperties.TotalSize
	}
	return nil
}

// GetDefaultBranch returns the default branch from git properties, or nil if not loaded
func (r *Repository) GetDefaultBranch() *string {
	if r.GitProperties != nil {
		return r.GitProperties.DefaultBranch
	}
	return nil
}

// HasLFS returns true if the repository uses Git LFS
func (r *Repository) HasLFS() bool {
	return r.GitProperties != nil && r.GitProperties.HasLFS
}

// HasSubmodules returns true if the repository has submodules
func (r *Repository) HasSubmodules() bool {
	return r.GitProperties != nil && r.GitProperties.HasSubmodules
}

// HasLargeFiles returns true if the repository has large files (>100MB)
func (r *Repository) HasLargeFiles() bool {
	return r.GitProperties != nil && r.GitProperties.HasLargeFiles
}

// GetBranchCount returns the branch count from git properties
func (r *Repository) GetBranchCount() int {
	if r.GitProperties != nil {
		return r.GitProperties.BranchCount
	}
	return 0
}

// GetCommitCount returns the commit count from git properties
func (r *Repository) GetCommitCount() int {
	if r.GitProperties != nil {
		return r.GitProperties.CommitCount
	}
	return 0
}

// HasWiki returns true if the repository has wiki enabled
func (r *Repository) HasWiki() bool {
	return r.Features != nil && r.Features.HasWiki
}

// HasActions returns true if the repository has GitHub Actions enabled
func (r *Repository) HasActions() bool {
	return r.Features != nil && r.Features.HasActions
}

// HasPackages returns true if the repository has packages
func (r *Repository) HasPackages() bool {
	return r.Features != nil && r.Features.HasPackages
}

// HasPages returns true if the repository has GitHub Pages enabled
func (r *Repository) HasPages() bool {
	return r.Features != nil && r.Features.HasPages
}

// HasDiscussions returns true if the repository has discussions enabled
func (r *Repository) HasDiscussions() bool {
	return r.Features != nil && r.Features.HasDiscussions
}

// HasProjects returns true if the repository has projects enabled
func (r *Repository) HasProjects() bool {
	return r.Features != nil && r.Features.HasProjects
}

// HasRulesets returns true if the repository has rulesets
func (r *Repository) HasRulesets() bool {
	return r.Features != nil && r.Features.HasRulesets
}

// GetBranchProtections returns the branch protection count
func (r *Repository) GetBranchProtections() int {
	if r.Features != nil {
		return r.Features.BranchProtections
	}
	return 0
}

// HasCodeScanning returns true if the repository has code scanning enabled
func (r *Repository) HasCodeScanning() bool {
	return r.Features != nil && r.Features.HasCodeScanning
}

// HasDependabot returns true if the repository has Dependabot enabled
func (r *Repository) HasDependabot() bool {
	return r.Features != nil && r.Features.HasDependabot
}

// HasSecretScanning returns true if the repository has secret scanning enabled
func (r *Repository) HasSecretScanning() bool {
	return r.Features != nil && r.Features.HasSecretScanning
}

// HasCodeowners returns true if the repository has a CODEOWNERS file
func (r *Repository) HasCodeowners() bool {
	return r.Features != nil && r.Features.HasCodeowners
}

// HasSelfHostedRunners returns true if the repository has self-hosted runners
func (r *Repository) HasSelfHostedRunners() bool {
	return r.Features != nil && r.Features.HasSelfHostedRunners
}

// GetComplexityScore returns the complexity score from validation, or nil if not loaded
func (r *Repository) GetComplexityScore() *int {
	if r.Validation != nil {
		return r.Validation.ComplexityScore
	}
	return nil
}

// GetComplexityBreakdown deserializes the JSON complexity breakdown from validation
func (r *Repository) GetComplexityBreakdown() (*ComplexityBreakdown, error) {
	if r.Validation == nil || r.Validation.ComplexityBreakdown == nil || *r.Validation.ComplexityBreakdown == "" {
		return nil, nil
	}

	var breakdown ComplexityBreakdown
	if err := json.Unmarshal([]byte(*r.Validation.ComplexityBreakdown), &breakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal complexity breakdown: %w", err)
	}

	return &breakdown, nil
}

// SetComplexityBreakdown serializes a ComplexityBreakdown struct to JSON and stores it
func (r *Repository) SetComplexityBreakdown(breakdown *ComplexityBreakdown) error {
	if r.Validation == nil {
		r.Validation = &RepositoryValidation{}
	}

	if breakdown == nil {
		r.Validation.ComplexityBreakdown = nil
		return nil
	}

	data, err := json.Marshal(breakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal complexity breakdown: %w", err)
	}

	jsonStr := string(data)
	r.Validation.ComplexityBreakdown = &jsonStr
	return nil
}

// HasOversizedCommits returns true if the repository has oversized commits
func (r *Repository) HasOversizedCommits() bool {
	return r.Validation != nil && r.Validation.HasOversizedCommits
}

// HasLongRefs returns true if the repository has long refs
func (r *Repository) HasLongRefs() bool {
	return r.Validation != nil && r.Validation.HasLongRefs
}

// HasBlockingFiles returns true if the repository has blocking files
func (r *Repository) HasBlockingFiles() bool {
	return r.Validation != nil && r.Validation.HasBlockingFiles
}

// HasOversizedRepository returns true if the repository is oversized
func (r *Repository) HasOversizedRepository() bool {
	return r.Validation != nil && r.Validation.HasOversizedRepository
}

// HasLargeFileWarnings returns true if the repository has large file warnings
func (r *Repository) HasLargeFileWarnings() bool {
	return r.Validation != nil && r.Validation.HasLargeFileWarnings
}

// IsADORepository returns true if this repository is from Azure DevOps.
func (r *Repository) IsADORepository() bool {
	return r.ADOProperties != nil && r.ADOProperties.Project != nil && *r.ADOProperties.Project != ""
}

// GetADOProject returns the ADO project name if this is an ADO repository
func (r *Repository) GetADOProject() *string {
	if r.ADOProperties != nil {
		return r.ADOProperties.Project
	}
	return nil
}

// GetADOIsGit returns whether this is a Git repository (vs TFVC) for ADO repos
func (r *Repository) GetADOIsGit() bool {
	if r.ADOProperties != nil {
		return r.ADOProperties.IsGit
	}
	return true // Default to true for non-ADO repos
}

// HasMigrationBlockers returns true if the repository has any migration blockers.
func (r *Repository) HasMigrationBlockers() bool {
	return r.HasOversizedCommits() || r.HasLongRefs() || r.HasBlockingFiles() || r.HasOversizedRepository()
}

// NeedsRemediation returns true if the repository status indicates remediation is required.
func (r *Repository) NeedsRemediation() bool {
	return r.Status == string(StatusRemediationRequired)
}

// IsMigrationComplete returns true if the repository migration is complete.
func (r *Repository) IsMigrationComplete() bool {
	return r.Status == string(StatusComplete) || r.Status == string(StatusMigrationComplete)
}

// IsMigrationInProgress returns true if the repository is currently being migrated.
func (r *Repository) IsMigrationInProgress() bool {
	inProgressStatuses := map[string]bool{
		string(StatusPreMigration):       true,
		string(StatusArchiveGenerating):  true,
		string(StatusQueuedForMigration): true,
		string(StatusMigratingContent):   true,
		string(StatusPostMigration):      true,
	}
	return inProgressStatuses[r.Status]
}

// IsMigrationFailed returns true if the repository migration has failed.
func (r *Repository) IsMigrationFailed() bool {
	return r.Status == string(StatusMigrationFailed) || r.Status == string(StatusRolledBack)
}

// CanBeMigrated returns true if the repository is in a state where it can be queued for migration.
func (r *Repository) CanBeMigrated() bool {
	if r.Status == string(StatusWontMigrate) {
		return false
	}
	eligibleStatuses := map[string]bool{
		string(StatusPending):         true,
		string(StatusDryRunComplete):  true,
		string(StatusDryRunFailed):    true,
		string(StatusMigrationFailed): true,
		string(StatusRolledBack):      true,
		string(StatusDryRunQueued):    true,
	}
	return eligibleStatuses[r.Status]
}

// CanBeAssignedToBatch returns true if the repository can be assigned to a batch.
func (r *Repository) CanBeAssignedToBatch() (bool, string) {
	if r.BatchID != nil {
		return false, "repository is already assigned to a batch"
	}
	if r.HasOversizedRepository() {
		return false, "repository exceeds GitHub's 40 GiB size limit and requires remediation"
	}
	eligibleStatuses := map[string]bool{
		string(StatusPending):         true,
		string(StatusDryRunComplete):  true,
		string(StatusDryRunFailed):    true,
		string(StatusMigrationFailed): true,
		string(StatusRolledBack):      true,
	}
	if !eligibleStatuses[r.Status] {
		return false, "repository status is not eligible for batch assignment"
	}
	return true, ""
}

// GetComplexityCategoryFromFeatures returns the complexity category based on repository features.
func (r *Repository) GetComplexityCategoryFromFeatures() string {
	// Very complex: Has migration blockers
	if r.HasMigrationBlockers() {
		return ComplexityVeryComplex
	}

	// Count complex features
	complexFeatureCount := 0
	if r.HasLFS() {
		complexFeatureCount++
	}
	if r.HasSubmodules() {
		complexFeatureCount++
	}
	if r.HasLargeFiles() {
		complexFeatureCount++
	}
	if r.HasPackages() {
		complexFeatureCount++
	}
	if r.HasActions() {
		complexFeatureCount++
	}
	if r.GetBranchProtections() > 5 {
		complexFeatureCount++
	}
	if r.HasRulesets() {
		complexFeatureCount++
	}

	if complexFeatureCount >= 4 {
		return ComplexityVeryComplex
	}
	if complexFeatureCount >= 2 {
		return ComplexityComplex
	}
	if complexFeatureCount >= 1 {
		return ComplexityMedium
	}
	if r.GetTotalSize() != nil && *r.GetTotalSize() > 1<<30 { // > 1GB
		return ComplexityMedium
	}

	return ComplexitySimple
}

// RepositorySource constants for source types
const (
	SourceGHES        = "ghes"
	SourceGHEC        = "ghec"
	SourceGitLab      = "gitlab"
	SourceAzureDevOps = "azuredevops"
)

// RepositoryOptions provides options for creating a new repository
type RepositoryOptions struct {
	FullName      string
	Source        string
	SourceURL     string
	Visibility    string
	DefaultBranch *string
	TotalSize     *int64
	IsArchived    bool
	IsFork        bool
	HasWiki       bool
	HasPages      bool
	// ADO-specific options
	ADOProject *string
	ADOIsGit   bool
}

// NewRepository creates a new Repository with common fields initialized.
// This factory function ensures consistent initialization across all collectors.
func NewRepository(opts RepositoryOptions) *Repository {
	now := time.Now()
	repo := &Repository{
		FullName:        opts.FullName,
		Source:          opts.Source,
		SourceURL:       opts.SourceURL,
		Visibility:      opts.Visibility,
		IsArchived:      opts.IsArchived,
		IsFork:          opts.IsFork,
		Status:          string(StatusPending),
		DiscoveredAt:    now,
		UpdatedAt:       now,
		LastDiscoveryAt: &now,
	}

	// Initialize git properties if provided
	if opts.DefaultBranch != nil || opts.TotalSize != nil {
		repo.GitProperties = &RepositoryGitProperties{
			DefaultBranch: opts.DefaultBranch,
			TotalSize:     opts.TotalSize,
		}
	}

	// Initialize features if provided
	if opts.HasWiki || opts.HasPages {
		repo.Features = &RepositoryFeatures{
			HasWiki:  opts.HasWiki,
			HasPages: opts.HasPages,
		}
	}

	// Initialize ADO properties if provided
	if opts.ADOProject != nil {
		repo.ADOProperties = &RepositoryADOProperties{
			Project: opts.ADOProject,
			IsGit:   opts.ADOIsGit,
		}
	}

	return repo
}

// NewGitHubRepository creates a new Repository from GitHub API data with standard settings.
func NewGitHubRepository(fullName, sourceURL, visibility string, defaultBranch *string, totalSize *int64, isArchived, isFork, hasWiki, hasPages bool) *Repository {
	return NewRepository(RepositoryOptions{
		FullName:      fullName,
		Source:        SourceGHES,
		SourceURL:     sourceURL,
		Visibility:    visibility,
		DefaultBranch: defaultBranch,
		TotalSize:     totalSize,
		IsArchived:    isArchived,
		IsFork:        isFork,
		HasWiki:       hasWiki,
		HasPages:      hasPages,
	})
}

// NewADORepository creates a new Repository for Azure DevOps with standard settings.
func NewADORepository(fullName, sourceURL, visibility string, project *string, isGit bool) *Repository {
	repo := NewRepository(RepositoryOptions{
		FullName:   fullName,
		Source:     SourceAzureDevOps,
		SourceURL:  sourceURL,
		Visibility: visibility,
		ADOProject: project,
		ADOIsGit:   isGit,
	})

	// TFVC repos need special status
	if !isGit {
		repo.Status = string(StatusRemediationRequired)
	}

	return repo
}

// flattenOptionalFields adds optional core fields to the result map
func (r *Repository) flattenOptionalFields(result map[string]any) {
	if r.SourceID != nil {
		result["source_id"] = *r.SourceID
	}
	if r.BatchID != nil {
		result["batch_id"] = *r.BatchID
	}
	if r.DestinationURL != nil {
		result["destination_url"] = *r.DestinationURL
	}
	if r.DestinationFullName != nil {
		result["destination_full_name"] = *r.DestinationFullName
	}
	if r.SourceMigrationID != nil {
		result["source_migration_id"] = *r.SourceMigrationID
	}
	if r.MigratedAt != nil {
		result["migrated_at"] = *r.MigratedAt
	}
	if r.LastDiscoveryAt != nil {
		result["last_discovery_at"] = *r.LastDiscoveryAt
	}
	if r.LastDryRunAt != nil {
		result["last_dry_run_at"] = *r.LastDryRunAt
	}
}

// flattenGitProperties adds git properties to the result map
func (r *Repository) flattenGitProperties(result map[string]any) {
	if r.GitProperties == nil {
		return
	}
	gp := r.GitProperties
	if gp.TotalSize != nil {
		result["total_size"] = *gp.TotalSize
	}
	if gp.LargestFile != nil {
		result["largest_file"] = *gp.LargestFile
	}
	if gp.LargestFileSize != nil {
		result["largest_file_size"] = *gp.LargestFileSize
	}
	if gp.LargestCommit != nil {
		result["largest_commit"] = *gp.LargestCommit
	}
	if gp.LargestCommitSize != nil {
		result["largest_commit_size"] = *gp.LargestCommitSize
	}
	if gp.DefaultBranch != nil {
		result["default_branch"] = *gp.DefaultBranch
	}
	if gp.LastCommitSHA != nil {
		result["last_commit_sha"] = *gp.LastCommitSHA
	}
	if gp.LastCommitDate != nil {
		result["last_commit_date"] = *gp.LastCommitDate
	}
	result["has_lfs"] = gp.HasLFS
	result["has_submodules"] = gp.HasSubmodules
	result["has_large_files"] = gp.HasLargeFiles
	result["large_file_count"] = gp.LargeFileCount
	result["branch_count"] = gp.BranchCount
	result["commit_count"] = gp.CommitCount
	result["commits_last_12_weeks"] = gp.CommitsLast12Weeks
}

// flattenFeatures adds repository features to the result map
func (r *Repository) flattenFeatures(result map[string]any) {
	if r.Features == nil {
		return
	}
	f := r.Features
	result["has_wiki"] = f.HasWiki
	result["has_pages"] = f.HasPages
	result["has_discussions"] = f.HasDiscussions
	result["has_actions"] = f.HasActions
	result["has_projects"] = f.HasProjects
	result["has_packages"] = f.HasPackages
	result["branch_protections"] = f.BranchProtections
	result["has_rulesets"] = f.HasRulesets
	result["tag_protection_count"] = f.TagProtectionCount
	result["environment_count"] = f.EnvironmentCount
	result["secret_count"] = f.SecretCount
	result["variable_count"] = f.VariableCount
	result["webhook_count"] = f.WebhookCount
	result["has_code_scanning"] = f.HasCodeScanning
	result["has_dependabot"] = f.HasDependabot
	result["has_secret_scanning"] = f.HasSecretScanning
	result["has_codeowners"] = f.HasCodeowners
	result["workflow_count"] = f.WorkflowCount
	result["has_self_hosted_runners"] = f.HasSelfHostedRunners
	result["collaborator_count"] = f.CollaboratorCount
	result["installed_apps_count"] = f.InstalledAppsCount
	if f.InstalledApps != nil {
		result["installed_apps"] = *f.InstalledApps
	}
	result["release_count"] = f.ReleaseCount
	result["has_release_assets"] = f.HasReleaseAssets
	result["contributor_count"] = f.ContributorCount
	if f.TopContributors != nil {
		result["top_contributors"] = *f.TopContributors
	}
	result["issue_count"] = f.IssueCount
	result["pull_request_count"] = f.PullRequestCount
	result["tag_count"] = f.TagCount
	result["open_issue_count"] = f.OpenIssueCount
	result["open_pr_count"] = f.OpenPRCount
}

// flattenADOProperties adds Azure DevOps properties to the result map
func (r *Repository) flattenADOProperties(result map[string]any) {
	if r.ADOProperties == nil {
		return
	}
	ado := r.ADOProperties
	if ado.Project != nil {
		result["ado_project"] = *ado.Project
	}
	result["ado_is_git"] = ado.IsGit
	result["ado_has_boards"] = ado.HasBoards
	result["ado_has_pipelines"] = ado.HasPipelines
	result["ado_has_ghas"] = ado.HasGHAS
	result["ado_pull_request_count"] = ado.PullRequestCount
	result["ado_work_item_count"] = ado.WorkItemCount
	result["ado_branch_policy_count"] = ado.BranchPolicyCount
	result["ado_pipeline_count"] = ado.PipelineCount
	result["ado_yaml_pipeline_count"] = ado.YAMLPipelineCount
	result["ado_classic_pipeline_count"] = ado.ClassicPipelineCount
	result["ado_pipeline_run_count"] = ado.PipelineRunCount
	result["ado_has_service_connections"] = ado.HasServiceConnections
	result["ado_has_variable_groups"] = ado.HasVariableGroups
	result["ado_has_self_hosted_agents"] = ado.HasSelfHostedAgents
	result["ado_work_item_linked_count"] = ado.WorkItemLinkedCount
	result["ado_active_work_item_count"] = ado.ActiveWorkItemCount
	if ado.WorkItemTypes != nil {
		result["ado_work_item_types"] = *ado.WorkItemTypes
	}
	result["ado_open_pr_count"] = ado.OpenPRCount
	result["ado_pr_with_linked_work_items"] = ado.PRWithLinkedWorkItems
	result["ado_pr_with_attachments"] = ado.PRWithAttachments
	if ado.BranchPolicyTypes != nil {
		result["ado_branch_policy_types"] = *ado.BranchPolicyTypes
	}
	result["ado_required_reviewer_count"] = ado.RequiredReviewerCount
	result["ado_build_validation_policies"] = ado.BuildValidationPolicies
	result["ado_has_wiki"] = ado.HasWiki
	result["ado_wiki_page_count"] = ado.WikiPageCount
	result["ado_test_plan_count"] = ado.TestPlanCount
	result["ado_test_case_count"] = ado.TestCaseCount
	result["ado_package_feed_count"] = ado.PackageFeedCount
	result["ado_has_artifacts"] = ado.HasArtifacts
	result["ado_service_hook_count"] = ado.ServiceHookCount
	if ado.InstalledExtensions != nil {
		result["ado_installed_extensions"] = *ado.InstalledExtensions
	}
}

// flattenValidation adds validation data to the result map
func (r *Repository) flattenValidation(result map[string]any) {
	if r.Validation == nil {
		return
	}
	v := r.Validation
	result["has_oversized_commits"] = v.HasOversizedCommits
	if v.OversizedCommitDetails != nil {
		result["oversized_commit_details"] = *v.OversizedCommitDetails
	}
	result["has_long_refs"] = v.HasLongRefs
	if v.LongRefDetails != nil {
		result["long_ref_details"] = *v.LongRefDetails
	}
	result["has_blocking_files"] = v.HasBlockingFiles
	if v.BlockingFileDetails != nil {
		result["blocking_file_details"] = *v.BlockingFileDetails
	}
	result["has_large_file_warnings"] = v.HasLargeFileWarnings
	if v.LargeFileWarningDetails != nil {
		result["large_file_warning_details"] = *v.LargeFileWarningDetails
	}
	result["has_oversized_repository"] = v.HasOversizedRepository
	if v.OversizedRepositoryDetails != nil {
		result["oversized_repository_details"] = *v.OversizedRepositoryDetails
	}
	if v.EstimatedMetadataSize != nil {
		result["estimated_metadata_size"] = *v.EstimatedMetadataSize
	}
	if v.MetadataSizeDetails != nil {
		result["metadata_size_details"] = *v.MetadataSizeDetails
	}
	if v.ComplexityScore != nil {
		result["complexity_score"] = *v.ComplexityScore
	}
	// Parse complexity breakdown JSON string into object
	if v.ComplexityBreakdown != nil && *v.ComplexityBreakdown != "" {
		var breakdown ComplexityBreakdown
		if err := json.Unmarshal([]byte(*v.ComplexityBreakdown), &breakdown); err == nil {
			result["complexity_breakdown"] = breakdown
		}
	}
}

// MarshalJSON implements custom JSON marshaling to flatten related table data for API compatibility
func (r *Repository) MarshalJSON() ([]byte, error) {
	// Build flattened result map with core fields
	result := map[string]any{
		"id":                     r.ID,
		"full_name":              r.FullName,
		"source":                 r.Source,
		"source_url":             r.SourceURL,
		"status":                 r.Status,
		"priority":               r.Priority,
		"visibility":             r.Visibility,
		"is_archived":            r.IsArchived,
		"is_fork":                r.IsFork,
		"is_source_locked":       r.IsSourceLocked,
		"exclude_releases":       r.ExcludeReleases,
		"exclude_attachments":    r.ExcludeAttachments,
		"exclude_metadata":       r.ExcludeMetadata,
		"exclude_git_data":       r.ExcludeGitData,
		"exclude_owner_projects": r.ExcludeOwnerProjects,
		"discovered_at":          r.DiscoveredAt,
		"updated_at":             r.UpdatedAt,
	}

	// Flatten related data
	r.flattenOptionalFields(result)
	r.flattenGitProperties(result)
	r.flattenFeatures(result)
	r.flattenADOProperties(result)
	r.flattenValidation(result)

	// Also include the nested objects for clients that want them
	result["git_properties"] = r.GitProperties
	result["features"] = r.Features
	result["ado_properties"] = r.ADOProperties
	result["validation"] = r.Validation

	return json.Marshal(result)
}

// Organization extracts the organization from full_name (org/repo)
func (r *Repository) Organization() string {
	parts := strings.SplitN(r.FullName, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Name extracts the repository name from full_name (org/repo)
func (r *Repository) Name() string {
	parts := strings.SplitN(r.FullName, "/", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return r.FullName
}

// DestinationRepoName returns the appropriate destination repository name for GitHub.
// For ADO repos (org/project/repo format), returns "project-repo" pattern to preserve
// project context and avoid naming conflicts.
// For GitHub repos (org/repo format), returns just the repo name.
// Spaces and slashes are replaced with hyphens for GitHub compatibility.
func (r *Repository) DestinationRepoName() string {
	// For ADO repos, use project-repo pattern
	if r.IsADORepository() {
		parts := strings.Split(r.FullName, "/")
		if len(parts) >= 3 {
			project := sanitizeRepoName(parts[1])
			repoName := sanitizeRepoName(parts[len(parts)-1])
			return project + "-" + repoName
		}
	}

	// For GitHub repos (org/repo format), use just the repo name
	parts := strings.Split(r.FullName, "/")
	if len(parts) >= 2 {
		return sanitizeRepoName(parts[len(parts)-1])
	}

	// Fallback: sanitize the full name
	return sanitizeRepoName(r.FullName)
}

// sanitizeRepoName replaces slashes and spaces with hyphens for GitHub compatibility
func sanitizeRepoName(name string) string {
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// GetOrganization extracts the organization name from the full name.
func (r *Repository) GetOrganization() string {
	if r.FullName == "" {
		return ""
	}
	for i, c := range r.FullName {
		if c == '/' {
			return r.FullName[:i]
		}
	}
	return r.FullName
}

// GetRepoName extracts the repository name from the full name.
func (r *Repository) GetRepoName() string {
	if r.FullName == "" {
		return ""
	}
	for i := len(r.FullName) - 1; i >= 0; i-- {
		if r.FullName[i] == '/' {
			return r.FullName[i+1:]
		}
	}
	return r.FullName
}

// ComplexityBreakdown provides individual component scores for transparency
type ComplexityBreakdown struct {
	SizePoints               int `json:"size_points"`                // 0-9 points based on repository size
	LargeFilesPoints         int `json:"large_files_points"`         // 4 points if has large files
	EnvironmentsPoints       int `json:"environments_points"`        // 3 points if has environments
	SecretsPoints            int `json:"secrets_points"`             // 3 points if has secrets
	PackagesPoints           int `json:"packages_points"`            // 3 points if has packages
	RunnersPoints            int `json:"runners_points"`             // 3 points if has self-hosted runners
	VariablesPoints          int `json:"variables_points"`           // 2 points if has variables
	DiscussionsPoints        int `json:"discussions_points"`         // 2 points if has discussions
	ReleasesPoints           int `json:"releases_points"`            // 2 points if has releases
	LFSPoints                int `json:"lfs_points"`                 // 2 points if has LFS
	SubmodulesPoints         int `json:"submodules_points"`          // 2 points if has submodules
	AppsPoints               int `json:"apps_points"`                // 2 points if has installed apps
	ProjectsPoints           int `json:"projects_points"`            // 2 points if has ProjectsV2
	SecurityPoints           int `json:"security_points"`            // 1 point if has GHAS features
	WebhooksPoints           int `json:"webhooks_points"`            // 1 point if has webhooks
	BranchProtectionsPoints  int `json:"branch_protections_points"`  // 1 point if has branch protections
	RulesetsPoints           int `json:"rulesets_points"`            // 1 point if has rulesets
	PublicVisibilityPoints   int `json:"public_visibility_points"`   // 1 point if public
	InternalVisibilityPoints int `json:"internal_visibility_points"` // 1 point if internal
	CodeownersPoints         int `json:"codeowners_points"`          // 1 point if has CODEOWNERS
	ActivityPoints           int `json:"activity_points"`            // 0, 2, or 4 points based on quantile

	// Azure DevOps specific complexity factors
	ADOTFVCPoints              int `json:"ado_tfvc_points"`               // 50 points - blocking, requires Git conversion
	ADOClassicPipelinePoints   int `json:"ado_classic_pipeline_points"`   // 5 points per pipeline - manual recreation required
	ADOPackageFeedPoints       int `json:"ado_package_feed_points"`       // 3 points - separate migration process
	ADOServiceConnectionPoints int `json:"ado_service_connection_points"` // 3 points - must recreate in GitHub
	ADOActivePipelinePoints    int `json:"ado_active_pipeline_points"`    // 3 points - CI/CD reconfiguration needed
	ADOActiveBoardsPoints      int `json:"ado_active_boards_points"`      // 3 points - work items don't migrate
	ADOWikiPoints              int `json:"ado_wiki_points"`               // 2 points per 10 pages - manual migration needed
	ADOTestPlanPoints          int `json:"ado_test_plan_points"`          // 2 points - no GitHub equivalent
	ADOVariableGroupPoints     int `json:"ado_variable_group_points"`     // 1 point - convert to GitHub secrets
	ADOServiceHookPoints       int `json:"ado_service_hook_points"`       // 1 point - recreate webhooks
	ADOManyPRsPoints           int `json:"ado_many_prs_points"`           // 2 points - metadata migration time
	ADOBranchPolicyPoints      int `json:"ado_branch_policy_points"`      // 1 point - need validation/recreation
}

// MigrationStatus represents the status of a repository migration
type MigrationStatus string

const (
	StatusPending             MigrationStatus = "pending"
	StatusRemediationRequired MigrationStatus = "remediation_required"
	StatusDryRunQueued        MigrationStatus = "dry_run_queued"
	StatusDryRunInProgress    MigrationStatus = "dry_run_in_progress"
	StatusDryRunComplete      MigrationStatus = "dry_run_complete"
	StatusDryRunFailed        MigrationStatus = "dry_run_failed"
	StatusPreMigration        MigrationStatus = "pre_migration"
	StatusArchiveGenerating   MigrationStatus = "archive_generating"
	StatusQueuedForMigration  MigrationStatus = "queued_for_migration"
	StatusMigratingContent    MigrationStatus = "migrating_content"
	StatusMigrationComplete   MigrationStatus = "migration_complete"
	StatusMigrationFailed     MigrationStatus = "migration_failed"
	StatusPostMigration       MigrationStatus = "post_migration"
	StatusComplete            MigrationStatus = "complete"
	StatusRolledBack          MigrationStatus = "rolled_back"
	StatusWontMigrate         MigrationStatus = "wont_migrate"
)

// MigrationHistory tracks the migration lifecycle of a repository
type MigrationHistory struct {
	ID              int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	RepositoryID    int64      `json:"repository_id" gorm:"column:repository_id;not null;index"`
	Status          string     `json:"status" gorm:"column:status;not null;index"`
	Phase           string     `json:"phase" gorm:"column:phase;not null"`
	Message         *string    `json:"message,omitempty" gorm:"column:message;type:text"`
	ErrorMessage    *string    `json:"error_message,omitempty" gorm:"column:error_message;type:text"`
	StartedAt       time.Time  `json:"started_at" gorm:"column:started_at;not null"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" gorm:"column:completed_at"`
	DurationSeconds *int       `json:"duration_seconds,omitempty" gorm:"column:duration_seconds"`
}

// TableName specifies the table name for MigrationHistory model
func (MigrationHistory) TableName() string {
	return "migration_history"
}

// MigrationLog provides detailed logging for troubleshooting migrations
type MigrationLog struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	RepositoryID int64     `json:"repository_id" gorm:"column:repository_id;not null;index"`
	HistoryID    *int64    `json:"history_id,omitempty" gorm:"column:history_id;index"`
	Level        string    `json:"level" gorm:"column:level;not null;index"` // DEBUG, INFO, WARN, ERROR
	Phase        string    `json:"phase" gorm:"column:phase;not null"`
	Operation    string    `json:"operation" gorm:"column:operation;not null"`
	Message      string    `json:"message" gorm:"column:message;not null"`
	Details      *string   `json:"details,omitempty" gorm:"column:details;type:text"` // Additional context, JSON or text
	InitiatedBy  *string   `json:"initiated_by,omitempty" gorm:"column:initiated_by"` // GitHub username of user who initiated action (when auth enabled)
	Timestamp    time.Time `json:"timestamp" gorm:"column:timestamp;not null;index;autoCreateTime"`
}

// TableName specifies the table name for MigrationLog model
func (MigrationLog) TableName() string {
	return "migration_logs"
}

// Batch represents a group of repositories to be migrated together
type Batch struct {
	ID                     int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name                   string     `json:"name" gorm:"column:name;not null;uniqueIndex"`
	Description            *string    `json:"description,omitempty" gorm:"column:description;type:text"`
	Type                   string     `json:"type" gorm:"column:type;not null;index"` // "pilot", "wave_1", "wave_2", etc.
	RepositoryCount        int        `json:"repository_count" gorm:"column:repository_count;default:0"`
	Status                 string     `json:"status" gorm:"column:status;not null;index"`
	ScheduledAt            *time.Time `json:"scheduled_at,omitempty" gorm:"column:scheduled_at"`
	StartedAt              *time.Time `json:"started_at,omitempty" gorm:"column:started_at"`
	CompletedAt            *time.Time `json:"completed_at,omitempty" gorm:"column:completed_at"`
	CreatedAt              time.Time  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	LastDryRunAt           *time.Time `json:"last_dry_run_at,omitempty" gorm:"column:last_dry_run_at"`                     // When batch dry run was last executed
	LastMigrationAttemptAt *time.Time `json:"last_migration_attempt_at,omitempty" gorm:"column:last_migration_attempt_at"` // When migration was last attempted

	// Dry run timing tracking
	DryRunStartedAt       *time.Time `json:"dry_run_started_at,omitempty" gorm:"column:dry_run_started_at"`             // When batch dry run started
	DryRunCompletedAt     *time.Time `json:"dry_run_completed_at,omitempty" gorm:"column:dry_run_completed_at"`         // When batch dry run completed
	DryRunDurationSeconds *int       `json:"dry_run_duration_seconds,omitempty" gorm:"column:dry_run_duration_seconds"` // Dry run duration in seconds

	// Migration Settings (batch-level defaults, repository settings take precedence)
	DestinationOrg     *string `json:"destination_org,omitempty" gorm:"column:destination_org"`             // Default destination org for repositories in this batch
	MigrationAPI       string  `json:"migration_api" gorm:"column:migration_api;not null"`                  // Migration API to use: "GEI" or "ELM" (default: "GEI")
	ExcludeReleases    bool    `json:"exclude_releases" gorm:"column:exclude_releases;default:false"`       // Skip releases during migration (applies if repo doesn't override)
	ExcludeAttachments bool    `json:"exclude_attachments" gorm:"column:exclude_attachments;default:false"` // Skip attachments during migration (applies if repo doesn't override)
}

// TableName specifies the table name for Batch model
func (Batch) TableName() string {
	return "batches"
}

// Duration calculates the batch execution duration if both StartedAt and CompletedAt are set
func (b *Batch) Duration() *time.Duration {
	if b.StartedAt == nil || b.CompletedAt == nil {
		return nil
	}
	duration := b.CompletedAt.Sub(*b.StartedAt)
	return &duration
}

// DurationSeconds returns the batch duration in seconds, or 0 if not completed
func (b *Batch) DurationSeconds() float64 {
	duration := b.Duration()
	if duration == nil {
		return 0
	}
	return duration.Seconds()
}

// DryRunDuration calculates the dry run execution duration if both timestamps are set
func (b *Batch) DryRunDuration() *time.Duration {
	if b.DryRunStartedAt == nil || b.DryRunCompletedAt == nil {
		return nil
	}
	duration := b.DryRunCompletedAt.Sub(*b.DryRunStartedAt)
	return &duration
}

// RepositoryDependency represents a dependency relationship between repositories
// Used for batch planning to understand which repositories should be migrated together
type RepositoryDependency struct {
	ID                 int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	RepositoryID       int64     `json:"repository_id" gorm:"column:repository_id;not null;index"`
	DependencyFullName string    `json:"dependency_full_name" gorm:"column:dependency_full_name;not null"` // org/repo format
	DependencyType     string    `json:"dependency_type" gorm:"column:dependency_type;not null"`           // submodule, workflow, dependency_graph, package
	DependencyURL      string    `json:"dependency_url" gorm:"column:dependency_url;not null"`             // Original URL/reference
	IsLocal            bool      `json:"is_local" gorm:"column:is_local;default:false"`                    // Whether dependency is within same enterprise
	DiscoveredAt       time.Time `json:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
	Metadata           *string   `json:"metadata,omitempty" gorm:"column:metadata;type:text"` // JSON with type-specific details (branch, version, path, etc.)
}

// TableName specifies the table name for RepositoryDependency model
func (RepositoryDependency) TableName() string {
	return "repository_dependencies"
}

// DependencyType constants for type safety
const (
	DependencyTypeSubmodule       = "submodule"
	DependencyTypeWorkflow        = "workflow"
	DependencyTypeDependencyGraph = "dependency_graph"
	DependencyTypePackage         = "package"
)

// GitHubTeam represents a GitHub team for filtering repositories by team membership
// Teams are org-scoped, so the same team name can exist in different organizations
type GitHubTeam struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceID     *int64    `json:"source_id,omitempty" gorm:"column:source_id;index"` // Foreign key to sources table for multi-source support
	Organization string    `json:"organization" gorm:"column:organization;not null;uniqueIndex:idx_github_teams_org_slug"`
	Slug         string    `json:"slug" gorm:"column:slug;not null;uniqueIndex:idx_github_teams_org_slug"`
	Name         string    `json:"name" gorm:"column:name;not null"`
	Description  *string   `json:"description,omitempty" gorm:"column:description;type:text"`
	Privacy      string    `json:"privacy" gorm:"column:privacy;not null;default:closed"`
	DiscoveredAt time.Time `json:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName specifies the table name for GitHubTeam model
func (GitHubTeam) TableName() string {
	return "github_teams"
}

// FullSlug returns the unique identifier for the team in "org/team-slug" format
func (t *GitHubTeam) FullSlug() string {
	return t.Organization + "/" + t.Slug
}

// GitHubTeamRepository represents the many-to-many relationship between teams and repositories
type GitHubTeamRepository struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TeamID       int64     `json:"team_id" gorm:"column:team_id;not null;uniqueIndex:idx_github_team_repo"`
	RepositoryID int64     `json:"repository_id" gorm:"column:repository_id;not null;uniqueIndex:idx_github_team_repo;index"`
	Permission   string    `json:"permission" gorm:"column:permission;not null;default:pull"` // pull, push, admin, maintain, triage
	DiscoveredAt time.Time `json:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
}

// TableName specifies the table name for GitHubTeamRepository model
func (GitHubTeamRepository) TableName() string {
	return "github_team_repositories"
}

// GitHubTeamMember represents a member of a GitHub team
type GitHubTeamMember struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TeamID       int64     `json:"team_id" gorm:"column:team_id;not null;index"`
	Login        string    `json:"login" gorm:"column:login;not null"`
	Role         string    `json:"role" gorm:"column:role;not null"` // member, maintainer
	DiscoveredAt time.Time `json:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
}

// TableName specifies the table name for GitHubTeamMember model
func (GitHubTeamMember) TableName() string {
	return "github_team_members"
}

// GitHubUser represents a GitHub user discovered during profiling
// Used for user identity mapping and mannequin reclaim
type GitHubUser struct {
	ID             int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceID       *int64    `json:"source_id,omitempty" gorm:"column:source_id;index"` // Foreign key to sources table for multi-source support
	Login          string    `json:"login" gorm:"column:login;not null;uniqueIndex"`
	Name           *string   `json:"name,omitempty" gorm:"column:name"`
	Email          *string   `json:"email,omitempty" gorm:"column:email;index"`
	AvatarURL      *string   `json:"avatar_url,omitempty" gorm:"column:avatar_url"`
	SourceInstance string    `json:"source_instance" gorm:"column:source_instance;not null"` // Source GitHub URL (e.g., github.company.com)
	DiscoveredAt   time.Time `json:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`

	// Contribution stats (aggregated across all repositories)
	CommitCount     int `json:"commit_count" gorm:"column:commit_count;default:0"`
	IssueCount      int `json:"issue_count" gorm:"column:issue_count;default:0"`
	PRCount         int `json:"pr_count" gorm:"column:pr_count;default:0"`
	CommentCount    int `json:"comment_count" gorm:"column:comment_count;default:0"`
	RepositoryCount int `json:"repository_count" gorm:"column:repository_count;default:0"`
}

// TableName specifies the table name for GitHubUser model
func (GitHubUser) TableName() string {
	return "github_users"
}

// UserOrgMembership tracks which organizations a user belongs to
// This enables organizing users by source org for mannequin reclamation
type UserOrgMembership struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserLogin    string    `json:"user_login" gorm:"column:user_login;not null;uniqueIndex:idx_user_org"`
	Organization string    `json:"organization" gorm:"column:organization;not null;uniqueIndex:idx_user_org;index"`
	Role         string    `json:"role" gorm:"column:role;not null;default:member"` // member, admin
	DiscoveredAt time.Time `json:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
}

// TableName specifies the table name for UserOrgMembership model
func (UserOrgMembership) TableName() string {
	return "user_org_memberships"
}

// UserMappingStatus represents the status of a user mapping
type UserMappingStatus string

const (
	UserMappingStatusUnmapped  UserMappingStatus = "unmapped"
	UserMappingStatusMapped    UserMappingStatus = "mapped"
	UserMappingStatusReclaimed UserMappingStatus = "reclaimed"
	UserMappingStatusSkipped   UserMappingStatus = "skipped"
)

// ReclaimStatus represents the status of mannequin reclaim
type ReclaimStatus string

const (
	ReclaimStatusPending   ReclaimStatus = "pending"
	ReclaimStatusInvited   ReclaimStatus = "invited"
	ReclaimStatusCompleted ReclaimStatus = "completed"
	ReclaimStatusFailed    ReclaimStatus = "failed"
)

// UserMapping maps a source user to a destination user for mannequin reclaim
type UserMapping struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceID         *int64    `json:"source_id,omitempty" gorm:"column:source_id;index"` // Foreign key to sources table for multi-source support
	SourceLogin      string    `json:"source_login" gorm:"column:source_login;not null;uniqueIndex"`
	SourceEmail      *string   `json:"source_email,omitempty" gorm:"column:source_email;index"`
	SourceName       *string   `json:"source_name,omitempty" gorm:"column:source_name"`
	SourceOrg        *string   `json:"source_org,omitempty" gorm:"column:source_org;index"` // Organization where user was discovered
	DestinationLogin *string   `json:"destination_login,omitempty" gorm:"column:destination_login;index"`
	DestinationEmail *string   `json:"destination_email,omitempty" gorm:"column:destination_email"`
	MappingStatus    string    `json:"mapping_status" gorm:"column:mapping_status;not null;default:unmapped;index"` // unmapped, mapped, reclaimed, skipped
	MannequinID      *string   `json:"mannequin_id,omitempty" gorm:"column:mannequin_id"`                           // GEI mannequin ID after migration
	MannequinLogin   *string   `json:"mannequin_login,omitempty" gorm:"column:mannequin_login"`                     // Mannequin login (e.g., mona-user-12345)
	ReclaimStatus    *string   `json:"reclaim_status,omitempty" gorm:"column:reclaim_status"`                       // pending, invited, completed, failed
	ReclaimError     *string   `json:"reclaim_error,omitempty" gorm:"column:reclaim_error;type:text"`               // Error message if reclaim failed
	MatchConfidence  *int      `json:"match_confidence,omitempty" gorm:"column:match_confidence"`                   // Auto-match confidence score (0-100)
	MatchReason      *string   `json:"match_reason,omitempty" gorm:"column:match_reason"`                           // Why the match was made (email, login, name)
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName specifies the table name for UserMapping model
func (UserMapping) TableName() string {
	return "user_mappings"
}

// TeamMapping maps a source team to a destination team
type TeamMapping struct {
	ID                  int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	SourceID            *int64     `json:"source_id,omitempty" gorm:"column:source_id;index"` // Foreign key to sources table for multi-source support
	SourceOrg           string     `json:"source_org" gorm:"column:source_org;not null;uniqueIndex:idx_team_mapping_source"`
	SourceTeamSlug      string     `json:"source_team_slug" gorm:"column:source_team_slug;not null;uniqueIndex:idx_team_mapping_source"`
	SourceTeamName      *string    `json:"source_team_name,omitempty" gorm:"column:source_team_name"`
	DestinationOrg      *string    `json:"destination_org,omitempty" gorm:"column:destination_org;index"`
	DestinationTeamSlug *string    `json:"destination_team_slug,omitempty" gorm:"column:destination_team_slug"`
	DestinationTeamName *string    `json:"destination_team_name,omitempty" gorm:"column:destination_team_name"`
	MappingStatus       string     `json:"mapping_status" gorm:"column:mapping_status;not null;default:unmapped;index"` // unmapped, mapped, skipped
	AutoCreated         bool       `json:"auto_created" gorm:"column:auto_created;default:false"`                       // True if team was auto-created during migration
	MigrationStatus     string     `json:"migration_status" gorm:"column:migration_status;default:pending;index"`       // pending, in_progress, completed, failed
	MigratedAt          *time.Time `json:"migrated_at,omitempty" gorm:"column:migrated_at"`                             // When the team was created in destination
	ErrorMessage        *string    `json:"error_message,omitempty" gorm:"column:error_message"`                         // Error details if migration failed
	ReposSynced         int        `json:"repos_synced" gorm:"column:repos_synced;default:0"`                           // Count of repos with permissions applied
	// New fields for tracking partial vs. full migration
	TotalSourceRepos  int        `json:"total_source_repos" gorm:"column:total_source_repos;default:0"`         // Total repos this team has access to in source
	ReposEligible     int        `json:"repos_eligible" gorm:"column:repos_eligible;default:0"`                 // How many repos have been migrated and are available for sync
	TeamCreatedInDest bool       `json:"team_created_in_dest" gorm:"column:team_created_in_dest;default:false"` // Whether team exists in destination
	LastSyncedAt      *time.Time `json:"last_synced_at,omitempty" gorm:"column:last_synced_at"`                 // When permissions were last synced
	CreatedAt         time.Time  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time  `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
}

// NeedsReSync returns true if the team has been migrated but has new repos that need permission sync
func (t *TeamMapping) NeedsReSync() bool {
	return t.TeamCreatedInDest && t.ReposSynced < t.ReposEligible
}

// GetMigrationCompleteness returns the migration completeness state
// Returns: "pending", "team_only", "partial", "complete", "needs_sync"
func (t *TeamMapping) GetMigrationCompleteness() string {
	if !t.TeamCreatedInDest {
		return "pending"
	}
	if t.ReposEligible == 0 {
		return "team_only" // Team created but no repos migrated yet
	}
	if t.ReposSynced == 0 {
		return "needs_sync" // Repos available but none synced
	}
	if t.ReposSynced < t.ReposEligible {
		return "partial" // Some repos synced but not all
	}
	return "complete" // All eligible repos have been synced
}

// TableName specifies the table name for TeamMapping model
func (TeamMapping) TableName() string {
	return "team_mappings"
}

// SourceFullSlug returns the source team identifier in "org/team-slug" format
func (t *TeamMapping) SourceFullSlug() string {
	return t.SourceOrg + "/" + t.SourceTeamSlug
}

// DestinationFullSlug returns the destination team identifier in "org/team-slug" format
// Returns empty string if destination is not mapped
func (t *TeamMapping) DestinationFullSlug() string {
	if t.DestinationOrg == nil || t.DestinationTeamSlug == nil {
		return ""
	}
	return *t.DestinationOrg + "/" + *t.DestinationTeamSlug
}

// Discovery progress phase constants
const (
	PhaseListingRepos        = "listing_repos"
	PhaseProfilingRepos      = "profiling_repos"
	PhaseDiscoveringTeams    = "discovering_teams"
	PhaseDiscoveringMembers  = "discovering_members"
	PhaseWaitingForRateLimit = "waiting_for_rate_limit"
	PhaseCancelling          = "cancelling"
	PhaseCompleted           = "completed"
)

// Discovery progress status constants
const (
	DiscoveryStatusInProgress = "in_progress"
	DiscoveryStatusCompleted  = "completed"
	DiscoveryStatusFailed     = "failed"
	DiscoveryStatusCancelled  = "cancelled"
)

// Discovery type constants
const (
	DiscoveryTypeEnterprise   = "enterprise"
	DiscoveryTypeOrganization = "organization"
	DiscoveryTypeRepository   = "repository"
	// ADO discovery types
	DiscoveryTypeADOOrganization = "ado_organization"
	DiscoveryTypeADOProject      = "ado_project"
)

// DiscoveryProgress tracks the progress of a discovery operation
type DiscoveryProgress struct {
	ID             int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	DiscoveryType  string     `json:"discovery_type" gorm:"column:discovery_type;not null"`           // "enterprise", "organization", or "repository"
	Target         string     `json:"target" gorm:"column:target;not null"`                           // enterprise slug, org name, or "org/repo"
	Status         string     `json:"status" gorm:"column:status;not null;default:in_progress;index"` // "in_progress", "completed", "failed"
	StartedAt      time.Time  `json:"started_at" gorm:"column:started_at;not null;autoCreateTime"`
	CompletedAt    *time.Time `json:"completed_at,omitempty" gorm:"column:completed_at"`
	TotalOrgs      int        `json:"total_orgs" gorm:"column:total_orgs;default:0"`
	ProcessedOrgs  int        `json:"processed_orgs" gorm:"column:processed_orgs;default:0"`
	CurrentOrg     string     `json:"current_org" gorm:"column:current_org"`
	TotalRepos     int        `json:"total_repos" gorm:"column:total_repos;default:0"`
	ProcessedRepos int        `json:"processed_repos" gorm:"column:processed_repos;default:0"`
	Phase          string     `json:"phase" gorm:"column:phase;default:listing_repos"` // Current phase within org processing
	ErrorCount     int        `json:"error_count" gorm:"column:error_count;default:0"`
	LastError      *string    `json:"last_error,omitempty" gorm:"column:last_error"`
}

// TableName specifies the table name for DiscoveryProgress model
func (DiscoveryProgress) TableName() string {
	return "discovery_progress"
}

// IsActive returns true if the discovery is still in progress
func (d *DiscoveryProgress) IsActive() bool {
	return d.Status == DiscoveryStatusInProgress
}

// PercentComplete returns the completion percentage based on repos processed
func (d *DiscoveryProgress) PercentComplete() float64 {
	if d.TotalRepos == 0 {
		return 0
	}
	return float64(d.ProcessedRepos) / float64(d.TotalRepos) * 100
}
