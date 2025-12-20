package models

// Repository Component Types
//
// This file defines component types that logically group related Repository fields.
// These types are used for:
//   - Type-safe access to related field groups
//   - Documentation of field relationships
//   - Future refactoring to embedded structs
//
// Note: These are NOT embedded in Repository yet to maintain backward compatibility.
// They serve as documentation and can be used in new code for type safety.

// GitProperties contains core Git repository properties.
type GitProperties struct {
	TotalSize          *int64  `json:"total_size,omitempty"`
	LargestFile        *string `json:"largest_file,omitempty"`
	LargestFileSize    *int64  `json:"largest_file_size,omitempty"`
	LargestCommit      *string `json:"largest_commit,omitempty"`
	LargestCommitSize  *int64  `json:"largest_commit_size,omitempty"`
	HasLFS             bool    `json:"has_lfs"`
	HasSubmodules      bool    `json:"has_submodules"`
	HasLargeFiles      bool    `json:"has_large_files"`
	LargeFileCount     int     `json:"large_file_count"`
	DefaultBranch      *string `json:"default_branch,omitempty"`
	BranchCount        int     `json:"branch_count"`
	CommitCount        int     `json:"commit_count"`
	CommitsLast12Weeks int     `json:"commits_last_12_weeks"`
	LastCommitSHA      *string `json:"last_commit_sha,omitempty"`
}

// GitHubFeatures contains GitHub-specific feature flags.
type GitHubFeatures struct {
	IsArchived        bool `json:"is_archived"`
	IsFork            bool `json:"is_fork"`
	HasWiki           bool `json:"has_wiki"`
	HasPages          bool `json:"has_pages"`
	HasDiscussions    bool `json:"has_discussions"`
	HasActions        bool `json:"has_actions"`
	HasProjects       bool `json:"has_projects"`
	HasPackages       bool `json:"has_packages"`
	BranchProtections int  `json:"branch_protections"`
	HasRulesets       bool `json:"has_rulesets"`
}

// SecurityFeatures contains security and compliance-related flags.
type SecurityFeatures struct {
	HasCodeScanning   bool `json:"has_code_scanning"`
	HasDependabot     bool `json:"has_dependabot"`
	HasSecretScanning bool `json:"has_secret_scanning"`
	HasCodeowners     bool `json:"has_codeowners"`
}

// CodeownersInfo contains CODEOWNERS file details.
type CodeownersInfo struct {
	Content *string `json:"content,omitempty"`
	Teams   *string `json:"teams,omitempty"`
	Users   *string `json:"users,omitempty"`
}

// MigrationState contains migration status and tracking fields.
type MigrationState struct {
	Status              string  `json:"status"`
	BatchID             *int64  `json:"batch_id,omitempty"`
	Priority            int     `json:"priority"`
	DestinationURL      *string `json:"destination_url,omitempty"`
	DestinationFullName *string `json:"destination_full_name,omitempty"`
	SourceMigrationID   *int64  `json:"source_migration_id,omitempty"`
	IsSourceLocked      bool    `json:"is_source_locked"`
}

// MigrationExclusions contains flags for excluding content during migration.
type MigrationExclusions struct {
	ExcludeReleases      bool `json:"exclude_releases"`
	ExcludeAttachments   bool `json:"exclude_attachments"`
	ExcludeMetadata      bool `json:"exclude_metadata"`
	ExcludeGitData       bool `json:"exclude_git_data"`
	ExcludeOwnerProjects bool `json:"exclude_owner_projects"`
}

// ValidationState contains post-migration validation tracking.
type ValidationState struct {
	ValidationStatus  *string `json:"validation_status,omitempty"`
	ValidationDetails *string `json:"validation_details,omitempty"`
	DestinationData   *string `json:"destination_data,omitempty"`
}

// GHESLimitViolations contains GitHub Enterprise Server migration limit violations.
type GHESLimitViolations struct {
	HasOversizedCommits        bool    `json:"has_oversized_commits"`
	OversizedCommitDetails     *string `json:"oversized_commit_details,omitempty"`
	HasLongRefs                bool    `json:"has_long_refs"`
	LongRefDetails             *string `json:"long_ref_details,omitempty"`
	HasBlockingFiles           bool    `json:"has_blocking_files"`
	BlockingFileDetails        *string `json:"blocking_file_details,omitempty"`
	HasLargeFileWarnings       bool    `json:"has_large_file_warnings"`
	LargeFileWarningDetails    *string `json:"large_file_warning_details,omitempty"`
	HasOversizedRepository     bool    `json:"has_oversized_repository"`
	OversizedRepositoryDetails *string `json:"oversized_repository_details,omitempty"`
}

// ADOProperties contains Azure DevOps specific fields.
type ADOProperties struct {
	Project           *string `json:"project,omitempty"`
	IsGit             bool    `json:"is_git"`
	HasBoards         bool    `json:"has_boards"`
	HasPipelines      bool    `json:"has_pipelines"`
	HasGHAS           bool    `json:"has_ghas"`
	PullRequestCount  int     `json:"pull_request_count"`
	WorkItemCount     int     `json:"work_item_count"`
	BranchPolicyCount int     `json:"branch_policy_count"`
}

// ADOPipelineDetails contains detailed Azure DevOps pipeline information.
type ADOPipelineDetails struct {
	PipelineCount         int  `json:"pipeline_count"`
	YAMLPipelineCount     int  `json:"yaml_pipeline_count"`
	ClassicPipelineCount  int  `json:"classic_pipeline_count"`
	PipelineRunCount      int  `json:"pipeline_run_count"`
	HasServiceConnections bool `json:"has_service_connections"`
	HasVariableGroups     bool `json:"has_variable_groups"`
	HasSelfHostedAgents   bool `json:"has_self_hosted_agents"`
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
