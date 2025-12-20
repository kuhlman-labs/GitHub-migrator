package models

import "time"

// Repository Component Types
//
// This file defines component types that logically group related Repository fields.
// These types are used for:
//   - Type-safe access to related field groups
//   - Documentation of field relationships
//   - Setter methods for bulk field updates
//   - Future refactoring to embedded structs
//
// The component types include GORM tags to enable future embedding with
// `gorm:"embedded"`. Currently, Repository uses flat fields for backward
// compatibility, but provides Get/Set methods for working with components.
//
// # Usage
//
//	// Get a component view (read-only copy)
//	git := repo.GetGitProperties()
//	if git.HasLFS { ... }
//
//	// Set from a component (bulk update)
//	repo.SetGitProperties(GitProperties{
//	    HasLFS: true,
//	    BranchCount: 5,
//	})
//
//	// Check migration state
//	if repo.IsMigrationComplete() { ... }
//	if repo.CanBeMigrated() { ... }

// GitProperties contains core Git repository properties.
// These fields describe the repository's Git-level characteristics.
type GitProperties struct {
	TotalSize          *int64     `json:"total_size,omitempty" gorm:"column:total_size"`
	LargestFile        *string    `json:"largest_file,omitempty" gorm:"column:largest_file"`
	LargestFileSize    *int64     `json:"largest_file_size,omitempty" gorm:"column:largest_file_size"`
	LargestCommit      *string    `json:"largest_commit,omitempty" gorm:"column:largest_commit"`
	LargestCommitSize  *int64     `json:"largest_commit_size,omitempty" gorm:"column:largest_commit_size"`
	HasLFS             bool       `json:"has_lfs" gorm:"column:has_lfs;default:false"`
	HasSubmodules      bool       `json:"has_submodules" gorm:"column:has_submodules;default:false"`
	HasLargeFiles      bool       `json:"has_large_files" gorm:"column:has_large_files;default:false"`
	LargeFileCount     int        `json:"large_file_count" gorm:"column:large_file_count;default:0"`
	DefaultBranch      *string    `json:"default_branch,omitempty" gorm:"column:default_branch"`
	BranchCount        int        `json:"branch_count" gorm:"column:branch_count;default:0"`
	CommitCount        int        `json:"commit_count" gorm:"column:commit_count;default:0"`
	CommitsLast12Weeks int        `json:"commits_last_12_weeks" gorm:"column:commits_last_12_weeks;default:0"`
	LastCommitSHA      *string    `json:"last_commit_sha,omitempty" gorm:"column:last_commit_sha"`
	LastCommitDate     *time.Time `json:"last_commit_date,omitempty" gorm:"column:last_commit_date"`
}

// GitHubFeatures contains GitHub-specific feature flags.
// These indicate which GitHub features are enabled on the repository.
type GitHubFeatures struct {
	IsArchived        bool `json:"is_archived" gorm:"column:is_archived;default:false"`
	IsFork            bool `json:"is_fork" gorm:"column:is_fork;default:false"`
	HasWiki           bool `json:"has_wiki" gorm:"column:has_wiki;default:false"`
	HasPages          bool `json:"has_pages" gorm:"column:has_pages;default:false"`
	HasDiscussions    bool `json:"has_discussions" gorm:"column:has_discussions;default:false"`
	HasActions        bool `json:"has_actions" gorm:"column:has_actions;default:false"`
	HasProjects       bool `json:"has_projects" gorm:"column:has_projects;default:false"`
	HasPackages       bool `json:"has_packages" gorm:"column:has_packages;default:false"`
	BranchProtections int  `json:"branch_protections" gorm:"column:branch_protections;default:0"`
	HasRulesets       bool `json:"has_rulesets" gorm:"column:has_rulesets;default:false"`
}

// SecurityFeatures contains security and compliance-related flags.
// These indicate GitHub Advanced Security features enabled on the repository.
type SecurityFeatures struct {
	HasCodeScanning   bool `json:"has_code_scanning" gorm:"column:has_code_scanning;default:false"`
	HasDependabot     bool `json:"has_dependabot" gorm:"column:has_dependabot;default:false"`
	HasSecretScanning bool `json:"has_secret_scanning" gorm:"column:has_secret_scanning;default:false"`
	HasCodeowners     bool `json:"has_codeowners" gorm:"column:has_codeowners;default:false"`
}

// CodeownersInfo contains CODEOWNERS file details.
type CodeownersInfo struct {
	Content *string `json:"content,omitempty" gorm:"column:codeowners_content;type:text"`
	Teams   *string `json:"teams,omitempty" gorm:"column:codeowners_teams;type:text"`
	Users   *string `json:"users,omitempty" gorm:"column:codeowners_users;type:text"`
}

// MigrationState contains migration status and tracking fields.
// These track the current state of the migration process.
type MigrationState struct {
	Status              string  `json:"status" gorm:"column:status;not null;index"`
	BatchID             *int64  `json:"batch_id,omitempty" gorm:"column:batch_id;index"`
	Priority            int     `json:"priority" gorm:"column:priority;default:0"`
	DestinationURL      *string `json:"destination_url,omitempty" gorm:"column:destination_url"`
	DestinationFullName *string `json:"destination_full_name,omitempty" gorm:"column:destination_full_name"`
	SourceMigrationID   *int64  `json:"source_migration_id,omitempty" gorm:"column:source_migration_id"`
	IsSourceLocked      bool    `json:"is_source_locked" gorm:"column:is_source_locked;default:false"`
}

// MigrationExclusions contains flags for excluding content during migration.
// These control what content is excluded when using GitHub Enterprise Importer.
type MigrationExclusions struct {
	ExcludeReleases      bool `json:"exclude_releases" gorm:"column:exclude_releases;default:false"`
	ExcludeAttachments   bool `json:"exclude_attachments" gorm:"column:exclude_attachments;default:false"`
	ExcludeMetadata      bool `json:"exclude_metadata" gorm:"column:exclude_metadata;default:false"`
	ExcludeGitData       bool `json:"exclude_git_data" gorm:"column:exclude_git_data;default:false"`
	ExcludeOwnerProjects bool `json:"exclude_owner_projects" gorm:"column:exclude_owner_projects;default:false"`
}

// ValidationState contains post-migration validation tracking.
type ValidationState struct {
	ValidationStatus  *string `json:"validation_status,omitempty" gorm:"column:validation_status"`
	ValidationDetails *string `json:"validation_details,omitempty" gorm:"column:validation_details;type:text"`
	DestinationData   *string `json:"destination_data,omitempty" gorm:"column:destination_data;type:text"`
}

// GHESLimitViolations contains GitHub Enterprise Server migration limit violations.
// These track issues that would prevent or complicate migration.
type GHESLimitViolations struct {
	HasOversizedCommits        bool    `json:"has_oversized_commits" gorm:"column:has_oversized_commits;default:false"`
	OversizedCommitDetails     *string `json:"oversized_commit_details,omitempty" gorm:"column:oversized_commit_details;type:text"`
	HasLongRefs                bool    `json:"has_long_refs" gorm:"column:has_long_refs;default:false"`
	LongRefDetails             *string `json:"long_ref_details,omitempty" gorm:"column:long_ref_details;type:text"`
	HasBlockingFiles           bool    `json:"has_blocking_files" gorm:"column:has_blocking_files;default:false"`
	BlockingFileDetails        *string `json:"blocking_file_details,omitempty" gorm:"column:blocking_file_details;type:text"`
	HasLargeFileWarnings       bool    `json:"has_large_file_warnings" gorm:"column:has_large_file_warnings;default:false"`
	LargeFileWarningDetails    *string `json:"large_file_warning_details,omitempty" gorm:"column:large_file_warning_details;type:text"`
	HasOversizedRepository     bool    `json:"has_oversized_repository" gorm:"column:has_oversized_repository;default:false"`
	OversizedRepositoryDetails *string `json:"oversized_repository_details,omitempty" gorm:"column:oversized_repository_details;type:text"`
}

// ADOProperties contains Azure DevOps specific fields.
// These are populated only for repositories sourced from Azure DevOps.
type ADOProperties struct {
	Project           *string `json:"project,omitempty" gorm:"column:ado_project"`
	IsGit             bool    `json:"is_git" gorm:"column:ado_is_git;default:true"`
	HasBoards         bool    `json:"has_boards" gorm:"column:ado_has_boards;default:false"`
	HasPipelines      bool    `json:"has_pipelines" gorm:"column:ado_has_pipelines;default:false"`
	HasGHAS           bool    `json:"has_ghas" gorm:"column:ado_has_ghas;default:false"`
	PullRequestCount  int     `json:"pull_request_count" gorm:"column:ado_pull_request_count;default:0"`
	WorkItemCount     int     `json:"work_item_count" gorm:"column:ado_work_item_count;default:0"`
	BranchPolicyCount int     `json:"branch_policy_count" gorm:"column:ado_branch_policy_count;default:0"`
}

// ADOPipelineDetails contains detailed Azure DevOps pipeline information.
type ADOPipelineDetails struct {
	PipelineCount         int  `json:"pipeline_count" gorm:"column:ado_pipeline_count;default:0"`
	YAMLPipelineCount     int  `json:"yaml_pipeline_count" gorm:"column:ado_yaml_pipeline_count;default:0"`
	ClassicPipelineCount  int  `json:"classic_pipeline_count" gorm:"column:ado_classic_pipeline_count;default:0"`
	PipelineRunCount      int  `json:"pipeline_run_count" gorm:"column:ado_pipeline_run_count;default:0"`
	HasServiceConnections bool `json:"has_service_connections" gorm:"column:ado_has_service_connections;default:false"`
	HasVariableGroups     bool `json:"has_variable_groups" gorm:"column:ado_has_variable_groups;default:false"`
	HasSelfHostedAgents   bool `json:"has_self_hosted_agents" gorm:"column:ado_has_self_hosted_agents;default:false"`
}

// Helper methods on Repository to access component views

// GetGitProperties returns a copy of the Git-related properties.
func (r *Repository) GetGitProperties() GitProperties {
	return GitProperties{
		TotalSize:          r.TotalSize,
		LargestFile:        r.LargestFile,
		LargestFileSize:    r.LargestFileSize,
		LargestCommit:      r.LargestCommit,
		LargestCommitSize:  r.LargestCommitSize,
		HasLFS:             r.HasLFS,
		HasSubmodules:      r.HasSubmodules,
		HasLargeFiles:      r.HasLargeFiles,
		LargeFileCount:     r.LargeFileCount,
		DefaultBranch:      r.DefaultBranch,
		BranchCount:        r.BranchCount,
		CommitCount:        r.CommitCount,
		CommitsLast12Weeks: r.CommitsLast12Weeks,
		LastCommitSHA:      r.LastCommitSHA,
		LastCommitDate:     r.LastCommitDate,
	}
}

// GetGitHubFeatures returns a copy of the GitHub feature flags.
func (r *Repository) GetGitHubFeatures() GitHubFeatures {
	return GitHubFeatures{
		IsArchived:        r.IsArchived,
		IsFork:            r.IsFork,
		HasWiki:           r.HasWiki,
		HasPages:          r.HasPages,
		HasDiscussions:    r.HasDiscussions,
		HasActions:        r.HasActions,
		HasProjects:       r.HasProjects,
		HasPackages:       r.HasPackages,
		BranchProtections: r.BranchProtections,
		HasRulesets:       r.HasRulesets,
	}
}

// GetSecurityFeatures returns a copy of the security feature flags.
func (r *Repository) GetSecurityFeatures() SecurityFeatures {
	return SecurityFeatures{
		HasCodeScanning:   r.HasCodeScanning,
		HasDependabot:     r.HasDependabot,
		HasSecretScanning: r.HasSecretScanning,
		HasCodeowners:     r.HasCodeowners,
	}
}

// GetMigrationState returns a copy of the migration state.
func (r *Repository) GetMigrationState() MigrationState {
	return MigrationState{
		Status:              r.Status,
		BatchID:             r.BatchID,
		Priority:            r.Priority,
		DestinationURL:      r.DestinationURL,
		DestinationFullName: r.DestinationFullName,
		SourceMigrationID:   r.SourceMigrationID,
		IsSourceLocked:      r.IsSourceLocked,
	}
}

// GetMigrationExclusions returns a copy of the migration exclusion flags.
func (r *Repository) GetMigrationExclusions() MigrationExclusions {
	return MigrationExclusions{
		ExcludeReleases:      r.ExcludeReleases,
		ExcludeAttachments:   r.ExcludeAttachments,
		ExcludeMetadata:      r.ExcludeMetadata,
		ExcludeGitData:       r.ExcludeGitData,
		ExcludeOwnerProjects: r.ExcludeOwnerProjects,
	}
}

// GetGHESLimitViolations returns a copy of the GHES limit violation flags.
func (r *Repository) GetGHESLimitViolations() GHESLimitViolations {
	return GHESLimitViolations{
		HasOversizedCommits:        r.HasOversizedCommits,
		OversizedCommitDetails:     r.OversizedCommitDetails,
		HasLongRefs:                r.HasLongRefs,
		LongRefDetails:             r.LongRefDetails,
		HasBlockingFiles:           r.HasBlockingFiles,
		BlockingFileDetails:        r.BlockingFileDetails,
		HasLargeFileWarnings:       r.HasLargeFileWarnings,
		LargeFileWarningDetails:    r.LargeFileWarningDetails,
		HasOversizedRepository:     r.HasOversizedRepository,
		OversizedRepositoryDetails: r.OversizedRepositoryDetails,
	}
}

// GetADOProperties returns a copy of the Azure DevOps properties.
func (r *Repository) GetADOProperties() ADOProperties {
	return ADOProperties{
		Project:           r.ADOProject,
		IsGit:             r.ADOIsGit,
		HasBoards:         r.ADOHasBoards,
		HasPipelines:      r.ADOHasPipelines,
		HasGHAS:           r.ADOHasGHAS,
		PullRequestCount:  r.ADOPullRequestCount,
		WorkItemCount:     r.ADOWorkItemCount,
		BranchPolicyCount: r.ADOBranchPolicyCount,
	}
}

// IsADORepository returns true if this repository is from Azure DevOps.
func (r *Repository) IsADORepository() bool {
	return r.ADOProject != nil && *r.ADOProject != ""
}

// HasMigrationBlockers returns true if the repository has any migration blockers.
func (r *Repository) HasMigrationBlockers() bool {
	return r.HasOversizedCommits || r.HasLongRefs || r.HasBlockingFiles || r.HasOversizedRepository
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
	if r.HasOversizedRepository {
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

// Setter methods for bulk updates from component types

// SetGitProperties updates all Git-related properties from a component.
func (r *Repository) SetGitProperties(p GitProperties) {
	r.TotalSize = p.TotalSize
	r.LargestFile = p.LargestFile
	r.LargestFileSize = p.LargestFileSize
	r.LargestCommit = p.LargestCommit
	r.LargestCommitSize = p.LargestCommitSize
	r.HasLFS = p.HasLFS
	r.HasSubmodules = p.HasSubmodules
	r.HasLargeFiles = p.HasLargeFiles
	r.LargeFileCount = p.LargeFileCount
	r.DefaultBranch = p.DefaultBranch
	r.BranchCount = p.BranchCount
	r.CommitCount = p.CommitCount
	r.CommitsLast12Weeks = p.CommitsLast12Weeks
	r.LastCommitSHA = p.LastCommitSHA
	r.LastCommitDate = p.LastCommitDate
}

// SetGitHubFeatures updates all GitHub feature flags from a component.
func (r *Repository) SetGitHubFeatures(f GitHubFeatures) {
	r.IsArchived = f.IsArchived
	r.IsFork = f.IsFork
	r.HasWiki = f.HasWiki
	r.HasPages = f.HasPages
	r.HasDiscussions = f.HasDiscussions
	r.HasActions = f.HasActions
	r.HasProjects = f.HasProjects
	r.HasPackages = f.HasPackages
	r.BranchProtections = f.BranchProtections
	r.HasRulesets = f.HasRulesets
}

// SetSecurityFeatures updates all security feature flags from a component.
func (r *Repository) SetSecurityFeatures(s SecurityFeatures) {
	r.HasCodeScanning = s.HasCodeScanning
	r.HasDependabot = s.HasDependabot
	r.HasSecretScanning = s.HasSecretScanning
	r.HasCodeowners = s.HasCodeowners
}

// SetMigrationState updates migration state fields from a component.
func (r *Repository) SetMigrationState(m MigrationState) {
	r.Status = m.Status
	r.BatchID = m.BatchID
	r.Priority = m.Priority
	r.DestinationURL = m.DestinationURL
	r.DestinationFullName = m.DestinationFullName
	r.SourceMigrationID = m.SourceMigrationID
	r.IsSourceLocked = m.IsSourceLocked
}

// SetMigrationExclusions updates migration exclusion flags from a component.
func (r *Repository) SetMigrationExclusions(e MigrationExclusions) {
	r.ExcludeReleases = e.ExcludeReleases
	r.ExcludeAttachments = e.ExcludeAttachments
	r.ExcludeMetadata = e.ExcludeMetadata
	r.ExcludeGitData = e.ExcludeGitData
	r.ExcludeOwnerProjects = e.ExcludeOwnerProjects
}

// SetGHESLimitViolations updates GHES limit violation flags from a component.
func (r *Repository) SetGHESLimitViolations(v GHESLimitViolations) {
	r.HasOversizedCommits = v.HasOversizedCommits
	r.OversizedCommitDetails = v.OversizedCommitDetails
	r.HasLongRefs = v.HasLongRefs
	r.LongRefDetails = v.LongRefDetails
	r.HasBlockingFiles = v.HasBlockingFiles
	r.BlockingFileDetails = v.BlockingFileDetails
	r.HasLargeFileWarnings = v.HasLargeFileWarnings
	r.LargeFileWarningDetails = v.LargeFileWarningDetails
	r.HasOversizedRepository = v.HasOversizedRepository
	r.OversizedRepositoryDetails = v.OversizedRepositoryDetails
}

// SetADOProperties updates Azure DevOps properties from a component.
func (r *Repository) SetADOProperties(a ADOProperties) {
	r.ADOProject = a.Project
	r.ADOIsGit = a.IsGit
	r.ADOHasBoards = a.HasBoards
	r.ADOHasPipelines = a.HasPipelines
	r.ADOHasGHAS = a.HasGHAS
	r.ADOPullRequestCount = a.PullRequestCount
	r.ADOWorkItemCount = a.WorkItemCount
	r.ADOBranchPolicyCount = a.BranchPolicyCount
}

// SetADOPipelineDetails updates Azure DevOps pipeline details from a component.
func (r *Repository) SetADOPipelineDetails(p ADOPipelineDetails) {
	r.ADOPipelineCount = p.PipelineCount
	r.ADOYAMLPipelineCount = p.YAMLPipelineCount
	r.ADOClassicPipelineCount = p.ClassicPipelineCount
	r.ADOPipelineRunCount = p.PipelineRunCount
	r.ADOHasServiceConnections = p.HasServiceConnections
	r.ADOHasVariableGroups = p.HasVariableGroups
	r.ADOHasSelfHostedAgents = p.HasSelfHostedAgents
}

// GetComplexityCategoryFromFeatures returns the complexity category based on repository features.
// This supplements the existing GetComplexityCategory function in constants.go
// by providing a feature-based assessment when complexity score is not available.
func (r *Repository) GetComplexityCategoryFromFeatures() string {
	// Very complex: Has migration blockers
	if r.HasMigrationBlockers() {
		return ComplexityVeryComplex
	}

	// Count complex features
	complexFeatureCount := 0
	if r.HasLFS {
		complexFeatureCount++
	}
	if r.HasSubmodules {
		complexFeatureCount++
	}
	if r.HasLargeFiles {
		complexFeatureCount++
	}
	if r.HasPackages {
		complexFeatureCount++
	}
	if r.HasActions {
		complexFeatureCount++
	}
	if r.BranchProtections > 5 {
		complexFeatureCount++
	}
	if r.HasRulesets {
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
	if r.TotalSize != nil && *r.TotalSize > 1<<30 { // > 1GB
		return ComplexityMedium
	}

	return ComplexitySimple
}
