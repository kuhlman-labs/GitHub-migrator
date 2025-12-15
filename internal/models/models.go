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

// Repository represents a Git repository to be migrated
type Repository struct {
	ID        int64  `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	FullName  string `json:"full_name" db:"full_name" gorm:"column:full_name;uniqueIndex;not null"` // org/repo
	Source    string `json:"source" db:"source" gorm:"column:source;not null"`                      // "ghes", "gitlab", etc.
	SourceURL string `json:"source_url" db:"source_url" gorm:"column:source_url;not null"`

	// Git properties
	TotalSize          *int64     `json:"total_size,omitempty" db:"total_size" gorm:"column:total_size"`
	LargestFile        *string    `json:"largest_file,omitempty" db:"largest_file" gorm:"column:largest_file"`
	LargestFileSize    *int64     `json:"largest_file_size,omitempty" db:"largest_file_size" gorm:"column:largest_file_size"`
	LargestCommit      *string    `json:"largest_commit,omitempty" db:"largest_commit" gorm:"column:largest_commit"`
	LargestCommitSize  *int64     `json:"largest_commit_size,omitempty" db:"largest_commit_size" gorm:"column:largest_commit_size"`
	HasLFS             bool       `json:"has_lfs" db:"has_lfs" gorm:"column:has_lfs;default:false"`
	HasSubmodules      bool       `json:"has_submodules" db:"has_submodules" gorm:"column:has_submodules;default:false"`
	HasLargeFiles      bool       `json:"has_large_files" db:"has_large_files" gorm:"column:has_large_files;default:false"` // Files > 100MB in history
	LargeFileCount     int        `json:"large_file_count" db:"large_file_count" gorm:"column:large_file_count;default:0"`
	DefaultBranch      *string    `json:"default_branch,omitempty" db:"default_branch" gorm:"column:default_branch"`
	BranchCount        int        `json:"branch_count" db:"branch_count" gorm:"column:branch_count;default:0"`
	CommitCount        int        `json:"commit_count" db:"commit_count" gorm:"column:commit_count;default:0"`
	CommitsLast12Weeks int        `json:"commits_last_12_weeks" db:"commits_last_12_weeks" gorm:"column:commits_last_12_weeks;default:0"`
	LastCommitSHA      *string    `json:"last_commit_sha,omitempty" db:"last_commit_sha" gorm:"column:last_commit_sha"`
	LastCommitDate     *time.Time `json:"last_commit_date,omitempty" db:"last_commit_date" gorm:"column:last_commit_date"`

	// GitHub features
	IsArchived         bool `json:"is_archived" db:"is_archived" gorm:"column:is_archived;default:false"`
	IsFork             bool `json:"is_fork" db:"is_fork" gorm:"column:is_fork;default:false"`
	HasWiki            bool `json:"has_wiki" db:"has_wiki" gorm:"column:has_wiki;default:false"`
	HasPages           bool `json:"has_pages" db:"has_pages" gorm:"column:has_pages;default:false"`
	HasDiscussions     bool `json:"has_discussions" db:"has_discussions" gorm:"column:has_discussions;default:false"`
	HasActions         bool `json:"has_actions" db:"has_actions" gorm:"column:has_actions;default:false"`
	HasProjects        bool `json:"has_projects" db:"has_projects" gorm:"column:has_projects;default:false"`
	HasPackages        bool `json:"has_packages" db:"has_packages" gorm:"column:has_packages;default:false"`
	BranchProtections  int  `json:"branch_protections" db:"branch_protections" gorm:"column:branch_protections;default:0"`
	HasRulesets        bool `json:"has_rulesets" db:"has_rulesets" gorm:"column:has_rulesets;default:false"`
	TagProtectionCount int  `json:"tag_protection_count" db:"tag_protection_count" gorm:"column:tag_protection_count;default:0"`
	EnvironmentCount   int  `json:"environment_count" db:"environment_count" gorm:"column:environment_count;default:0"`
	SecretCount        int  `json:"secret_count" db:"secret_count" gorm:"column:secret_count;default:0"`
	VariableCount      int  `json:"variable_count" db:"variable_count" gorm:"column:variable_count;default:0"`
	WebhookCount       int  `json:"webhook_count" db:"webhook_count" gorm:"column:webhook_count;default:0"`

	// Security & Compliance
	HasCodeScanning   bool `json:"has_code_scanning" db:"has_code_scanning" gorm:"column:has_code_scanning;default:false"`
	HasDependabot     bool `json:"has_dependabot" db:"has_dependabot" gorm:"column:has_dependabot;default:false"`
	HasSecretScanning bool `json:"has_secret_scanning" db:"has_secret_scanning" gorm:"column:has_secret_scanning;default:false"`
	HasCodeowners     bool `json:"has_codeowners" db:"has_codeowners" gorm:"column:has_codeowners;default:false"`

	// CODEOWNERS details (populated when HasCodeowners is true)
	CodeownersContent *string `json:"codeowners_content,omitempty" db:"codeowners_content" gorm:"column:codeowners_content;type:text"` // Raw CODEOWNERS file content
	CodeownersTeams   *string `json:"codeowners_teams,omitempty" db:"codeowners_teams" gorm:"column:codeowners_teams;type:text"`       // JSON array of team references (e.g., ["@org/team1", "@org/team2"])
	CodeownersUsers   *string `json:"codeowners_users,omitempty" db:"codeowners_users" gorm:"column:codeowners_users;type:text"`       // JSON array of user references (e.g., ["@user1", "@user2"])

	// Repository Settings
	Visibility    string `json:"visibility" db:"visibility" gorm:"column:visibility"` // "public", "private", "internal"
	WorkflowCount int    `json:"workflow_count" db:"workflow_count" gorm:"column:workflow_count;default:0"`

	// Infrastructure & Access
	HasSelfHostedRunners bool `json:"has_self_hosted_runners" db:"has_self_hosted_runners" gorm:"column:has_self_hosted_runners;default:false"`
	CollaboratorCount    int  `json:"collaborator_count" db:"collaborator_count" gorm:"column:collaborator_count;default:0"`
	InstalledAppsCount   int  `json:"installed_apps_count" db:"installed_apps_count" gorm:"column:installed_apps_count;default:0"`

	// Releases
	ReleaseCount     int  `json:"release_count" db:"release_count" gorm:"column:release_count;default:0"`
	HasReleaseAssets bool `json:"has_release_assets" db:"has_release_assets" gorm:"column:has_release_assets;default:false"`

	// Contributors
	ContributorCount int     `json:"contributor_count" db:"contributor_count" gorm:"column:contributor_count;default:0"`
	TopContributors  *string `json:"top_contributors,omitempty" db:"top_contributors" gorm:"column:top_contributors;type:text"` // JSON array

	// Verification data (for post-migration verification)
	IssueCount       int `json:"issue_count" db:"issue_count" gorm:"column:issue_count;default:0"`
	PullRequestCount int `json:"pull_request_count" db:"pull_request_count" gorm:"column:pull_request_count;default:0"`
	TagCount         int `json:"tag_count" db:"tag_count" gorm:"column:tag_count;default:0"`
	OpenIssueCount   int `json:"open_issue_count" db:"open_issue_count" gorm:"column:open_issue_count;default:0"`
	OpenPRCount      int `json:"open_pr_count" db:"open_pr_count" gorm:"column:open_pr_count;default:0"`

	// Status Tracking
	Status   string `json:"status" db:"status" gorm:"column:status;not null;index"`
	BatchID  *int64 `json:"batch_id,omitempty" db:"batch_id" gorm:"column:batch_id;index"`
	Priority int    `json:"priority" db:"priority" gorm:"column:priority;default:0"` // 0=normal, 1=pilot

	// Migration Details
	DestinationURL      *string `json:"destination_url,omitempty" db:"destination_url" gorm:"column:destination_url"`
	DestinationFullName *string `json:"destination_full_name,omitempty" db:"destination_full_name" gorm:"column:destination_full_name"`

	// Lock Tracking (for failed migrations)
	SourceMigrationID *int64 `json:"source_migration_id,omitempty" db:"source_migration_id" gorm:"column:source_migration_id"` // GHES migration ID
	IsSourceLocked    bool   `json:"is_source_locked" db:"is_source_locked" gorm:"column:is_source_locked;default:false"`      // Whether source repo is locked

	// Validation Tracking (for post-migration validation)
	ValidationStatus  *string `json:"validation_status,omitempty" db:"validation_status" gorm:"column:validation_status"`              // "passed", "failed", "skipped"
	ValidationDetails *string `json:"validation_details,omitempty" db:"validation_details" gorm:"column:validation_details;type:text"` // JSON with comparison results
	DestinationData   *string `json:"destination_data,omitempty" db:"destination_data" gorm:"column:destination_data;type:text"`       // JSON with destination repo data (only on validation failure)

	// GitHub Migration Limit Validations
	HasOversizedCommits     bool    `json:"has_oversized_commits" db:"has_oversized_commits" gorm:"column:has_oversized_commits;default:false"`                      // Commits >2 GiB
	OversizedCommitDetails  *string `json:"oversized_commit_details,omitempty" db:"oversized_commit_details" gorm:"column:oversized_commit_details;type:text"`       // JSON: [{sha, size}]
	HasLongRefs             bool    `json:"has_long_refs" db:"has_long_refs" gorm:"column:has_long_refs;default:false"`                                              // Git refs >255 bytes
	LongRefDetails          *string `json:"long_ref_details,omitempty" db:"long_ref_details" gorm:"column:long_ref_details;type:text"`                               // JSON: [ref names]
	HasBlockingFiles        bool    `json:"has_blocking_files" db:"has_blocking_files" gorm:"column:has_blocking_files;default:false"`                               // Files >400 MiB
	BlockingFileDetails     *string `json:"blocking_file_details,omitempty" db:"blocking_file_details" gorm:"column:blocking_file_details;type:text"`                // JSON: [{path, size}]
	HasLargeFileWarnings    bool    `json:"has_large_file_warnings" db:"has_large_file_warnings" gorm:"column:has_large_file_warnings;default:false"`                // Files 100-400 MiB
	LargeFileWarningDetails *string `json:"large_file_warning_details,omitempty" db:"large_file_warning_details" gorm:"column:large_file_warning_details;type:text"` // JSON: [{path, size}]

	// Repository Size Validation (40 GiB limit)
	HasOversizedRepository     bool    `json:"has_oversized_repository" db:"has_oversized_repository" gorm:"column:has_oversized_repository;default:false"`                   // Repository >40 GiB
	OversizedRepositoryDetails *string `json:"oversized_repository_details,omitempty" db:"oversized_repository_details" gorm:"column:oversized_repository_details;type:text"` // JSON: {size, limit}

	// Metadata Size Estimation (40 GiB metadata limit)
	EstimatedMetadataSize *int64  `json:"estimated_metadata_size,omitempty" db:"estimated_metadata_size" gorm:"column:estimated_metadata_size"`     // Estimated metadata size in bytes
	MetadataSizeDetails   *string `json:"metadata_size_details,omitempty" db:"metadata_size_details" gorm:"column:metadata_size_details;type:text"` // JSON: breakdown of metadata components

	// Migration Exclusion Flags (per-repository settings for GitHub Enterprise Importer API)
	ExcludeReleases      bool `json:"exclude_releases" db:"exclude_releases" gorm:"column:exclude_releases;default:false"`                   // Skip releases during migration
	ExcludeAttachments   bool `json:"exclude_attachments" db:"exclude_attachments" gorm:"column:exclude_attachments;default:false"`          // Skip attachments during migration
	ExcludeMetadata      bool `json:"exclude_metadata" db:"exclude_metadata" gorm:"column:exclude_metadata;default:false"`                   // Exclude all metadata (issues, PRs, etc.)
	ExcludeGitData       bool `json:"exclude_git_data" db:"exclude_git_data" gorm:"column:exclude_git_data;default:false"`                   // Exclude git data (commits, refs)
	ExcludeOwnerProjects bool `json:"exclude_owner_projects" db:"exclude_owner_projects" gorm:"column:exclude_owner_projects;default:false"` // Exclude organization/user projects

	// Azure DevOps specific fields
	ADOProject           *string `json:"ado_project,omitempty" db:"ado_project" gorm:"column:ado_project"`                                     // ADO project name
	ADOIsGit             bool    `json:"ado_is_git" db:"ado_is_git" gorm:"column:ado_is_git;default:true"`                                     // false = TFVC
	ADOHasBoards         bool    `json:"ado_has_boards" db:"ado_has_boards" gorm:"column:ado_has_boards;default:false"`                        // Azure Boards integration
	ADOHasPipelines      bool    `json:"ado_has_pipelines" db:"ado_has_pipelines" gorm:"column:ado_has_pipelines;default:false"`               // Azure Pipelines configured
	ADOHasGHAS           bool    `json:"ado_has_ghas" db:"ado_has_ghas" gorm:"column:ado_has_ghas;default:false"`                              // GitHub Advanced Security
	ADOPullRequestCount  int     `json:"ado_pull_request_count" db:"ado_pull_request_count" gorm:"column:ado_pull_request_count;default:0"`    // Total PR count
	ADOWorkItemCount     int     `json:"ado_work_item_count" db:"ado_work_item_count" gorm:"column:ado_work_item_count;default:0"`             // Linked work items
	ADOBranchPolicyCount int     `json:"ado_branch_policy_count" db:"ado_branch_policy_count" gorm:"column:ado_branch_policy_count;default:0"` // Branch policies

	// Enhanced Pipeline Data
	ADOPipelineCount         int  `json:"ado_pipeline_count" db:"ado_pipeline_count" gorm:"column:ado_pipeline_count;default:0"`                                // Total number of pipelines
	ADOYAMLPipelineCount     int  `json:"ado_yaml_pipeline_count" db:"ado_yaml_pipeline_count" gorm:"column:ado_yaml_pipeline_count;default:0"`                 // YAML pipelines (easier to migrate)
	ADOClassicPipelineCount  int  `json:"ado_classic_pipeline_count" db:"ado_classic_pipeline_count" gorm:"column:ado_classic_pipeline_count;default:0"`        // Classic pipelines (require manual recreation)
	ADOPipelineRunCount      int  `json:"ado_pipeline_run_count" db:"ado_pipeline_run_count" gorm:"column:ado_pipeline_run_count;default:0"`                    // Recent pipeline runs (indicates active CI/CD)
	ADOHasServiceConnections bool `json:"ado_has_service_connections" db:"ado_has_service_connections" gorm:"column:ado_has_service_connections;default:false"` // External service integrations
	ADOHasVariableGroups     bool `json:"ado_has_variable_groups" db:"ado_has_variable_groups" gorm:"column:ado_has_variable_groups;default:false"`             // Variable groups used
	ADOHasSelfHostedAgents   bool `json:"ado_has_self_hosted_agents" db:"ado_has_self_hosted_agents" gorm:"column:ado_has_self_hosted_agents;default:false"`    // Uses self-hosted agents

	// Enhanced Work Item Data
	ADOWorkItemLinkedCount int     `json:"ado_work_item_linked_count" db:"ado_work_item_linked_count" gorm:"column:ado_work_item_linked_count;default:0"` // Work items with commit/PR links
	ADOActiveWorkItemCount int     `json:"ado_active_work_item_count" db:"ado_active_work_item_count" gorm:"column:ado_active_work_item_count;default:0"` // Active (non-closed) work items
	ADOWorkItemTypes       *string `json:"ado_work_item_types,omitempty" db:"ado_work_item_types" gorm:"column:ado_work_item_types;type:text"`            // JSON array of work item types used

	// Pull Request Details
	ADOOpenPRCount           int `json:"ado_open_pr_count" db:"ado_open_pr_count" gorm:"column:ado_open_pr_count;default:0"`                                     // Open pull requests
	ADOPRWithLinkedWorkItems int `json:"ado_pr_with_linked_work_items" db:"ado_pr_with_linked_work_items" gorm:"column:ado_pr_with_linked_work_items;default:0"` // PRs with work item links (these migrate)
	ADOPRWithAttachments     int `json:"ado_pr_with_attachments" db:"ado_pr_with_attachments" gorm:"column:ado_pr_with_attachments;default:0"`                   // PRs with attachments

	// Enhanced Branch Policy Data
	ADOBranchPolicyTypes       *string `json:"ado_branch_policy_types,omitempty" db:"ado_branch_policy_types" gorm:"column:ado_branch_policy_types;type:text"`         // JSON array of policy types
	ADORequiredReviewerCount   int     `json:"ado_required_reviewer_count" db:"ado_required_reviewer_count" gorm:"column:ado_required_reviewer_count;default:0"`       // Required reviewer policies
	ADOBuildValidationPolicies int     `json:"ado_build_validation_policies" db:"ado_build_validation_policies" gorm:"column:ado_build_validation_policies;default:0"` // Build validation requirements

	// Wiki & Documentation
	ADOHasWiki       bool `json:"ado_has_wiki" db:"ado_has_wiki" gorm:"column:ado_has_wiki;default:false"`                  // Repository has wiki (doesn't migrate)
	ADOWikiPageCount int  `json:"ado_wiki_page_count" db:"ado_wiki_page_count" gorm:"column:ado_wiki_page_count;default:0"` // Number of wiki pages

	// Test Plans
	ADOTestPlanCount int `json:"ado_test_plan_count" db:"ado_test_plan_count" gorm:"column:ado_test_plan_count;default:0"` // Test plans linked to repo
	ADOTestCaseCount int `json:"ado_test_case_count" db:"ado_test_case_count" gorm:"column:ado_test_case_count;default:0"` // Test cases in plans

	// Artifacts & Packages
	ADOPackageFeedCount int  `json:"ado_package_feed_count" db:"ado_package_feed_count" gorm:"column:ado_package_feed_count;default:0"` // Package feeds associated
	ADOHasArtifacts     bool `json:"ado_has_artifacts" db:"ado_has_artifacts" gorm:"column:ado_has_artifacts;default:false"`            // Build artifacts configured

	// Service Hooks & Extensions
	ADOServiceHookCount    int     `json:"ado_service_hook_count" db:"ado_service_hook_count" gorm:"column:ado_service_hook_count;default:0"`                 // Service hooks/webhooks configured
	ADOInstalledExtensions *string `json:"ado_installed_extensions,omitempty" db:"ado_installed_extensions" gorm:"column:ado_installed_extensions;type:text"` // JSON array of repo-specific extensions

	// Timestamps
	DiscoveredAt    time.Time  `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
	MigratedAt      *time.Time `json:"migrated_at,omitempty" db:"migrated_at" gorm:"column:migrated_at"`
	LastDiscoveryAt *time.Time `json:"last_discovery_at,omitempty" db:"last_discovery_at" gorm:"column:last_discovery_at"` // Latest discovery refresh
	LastDryRunAt    *time.Time `json:"last_dry_run_at,omitempty" db:"last_dry_run_at" gorm:"column:last_dry_run_at"`       // Latest dry run execution

	// Complexity scoring (calculated during profiling and stored for performance)
	ComplexityScore     *int    `json:"complexity_score,omitempty" db:"complexity_score" gorm:"column:complexity_score"` // Calculated during profiling
	ComplexityBreakdown *string `json:"-" db:"complexity_breakdown" gorm:"column:complexity_breakdown;type:text"`        // JSON breakdown stored as string, marshaled as object via MarshalJSON
}

// TableName specifies the table name for Repository model
func (Repository) TableName() string {
	return "repositories"
}

// SetComplexityBreakdown serializes a ComplexityBreakdown struct to JSON and stores it
func (r *Repository) SetComplexityBreakdown(breakdown *ComplexityBreakdown) error {
	if breakdown == nil {
		r.ComplexityBreakdown = nil
		return nil
	}

	data, err := json.Marshal(breakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal complexity breakdown: %w", err)
	}

	jsonStr := string(data)
	r.ComplexityBreakdown = &jsonStr
	return nil
}

// GetComplexityBreakdown deserializes the JSON complexity breakdown
func (r *Repository) GetComplexityBreakdown() (*ComplexityBreakdown, error) {
	if r.ComplexityBreakdown == nil || *r.ComplexityBreakdown == "" {
		return nil, nil
	}

	var breakdown ComplexityBreakdown
	if err := json.Unmarshal([]byte(*r.ComplexityBreakdown), &breakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal complexity breakdown: %w", err)
	}

	return &breakdown, nil
}

// MarshalJSON implements custom JSON marshaling to convert complexity_breakdown from string to object
func (r *Repository) MarshalJSON() ([]byte, error) {
	// Create an alias without MarshalJSON to avoid recursion
	type Alias Repository

	// Marshal the main struct (complexity_breakdown is excluded via json:"-")
	data, err := json.Marshal((*Alias)(r))
	if err != nil {
		return nil, err
	}

	// If no complexity breakdown, return as-is
	if r.ComplexityBreakdown == nil || *r.ComplexityBreakdown == "" {
		return data, nil
	}

	// Parse the marshaled JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Deserialize the complexity breakdown string and add it as an object
	var breakdown ComplexityBreakdown
	if err := json.Unmarshal([]byte(*r.ComplexityBreakdown), &breakdown); err == nil {
		result["complexity_breakdown"] = breakdown
	}

	return json.Marshal(result)
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
	ID              int64      `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	RepositoryID    int64      `json:"repository_id" db:"repository_id" gorm:"column:repository_id;not null;index"`
	Status          string     `json:"status" db:"status" gorm:"column:status;not null;index"`
	Phase           string     `json:"phase" db:"phase" gorm:"column:phase;not null"`
	Message         *string    `json:"message,omitempty" db:"message" gorm:"column:message;type:text"`
	ErrorMessage    *string    `json:"error_message,omitempty" db:"error_message" gorm:"column:error_message;type:text"`
	StartedAt       time.Time  `json:"started_at" db:"started_at" gorm:"column:started_at;not null"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at" gorm:"column:completed_at"`
	DurationSeconds *int       `json:"duration_seconds,omitempty" db:"duration_seconds" gorm:"column:duration_seconds"`
}

// TableName specifies the table name for MigrationHistory model
func (MigrationHistory) TableName() string {
	return "migration_history"
}

// MigrationLog provides detailed logging for troubleshooting migrations
type MigrationLog struct {
	ID           int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	RepositoryID int64     `json:"repository_id" db:"repository_id" gorm:"column:repository_id;not null;index"`
	HistoryID    *int64    `json:"history_id,omitempty" db:"history_id" gorm:"column:history_id;index"`
	Level        string    `json:"level" db:"level" gorm:"column:level;not null;index"` // DEBUG, INFO, WARN, ERROR
	Phase        string    `json:"phase" db:"phase" gorm:"column:phase;not null"`
	Operation    string    `json:"operation" db:"operation" gorm:"column:operation;not null"`
	Message      string    `json:"message" db:"message" gorm:"column:message;not null"`
	Details      *string   `json:"details,omitempty" db:"details" gorm:"column:details;type:text"`      // Additional context, JSON or text
	InitiatedBy  *string   `json:"initiated_by,omitempty" db:"initiated_by" gorm:"column:initiated_by"` // GitHub username of user who initiated action (when auth enabled)
	Timestamp    time.Time `json:"timestamp" db:"timestamp" gorm:"column:timestamp;not null;index;autoCreateTime"`
}

// TableName specifies the table name for MigrationLog model
func (MigrationLog) TableName() string {
	return "migration_logs"
}

// Batch represents a group of repositories to be migrated together
type Batch struct {
	ID                     int64      `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Name                   string     `json:"name" db:"name" gorm:"column:name;not null;uniqueIndex"`
	Description            *string    `json:"description,omitempty" db:"description" gorm:"column:description;type:text"`
	Type                   string     `json:"type" db:"type" gorm:"column:type;not null;index"` // "pilot", "wave_1", "wave_2", etc.
	RepositoryCount        int        `json:"repository_count" db:"repository_count" gorm:"column:repository_count;default:0"`
	Status                 string     `json:"status" db:"status" gorm:"column:status;not null;index"`
	ScheduledAt            *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at" gorm:"column:scheduled_at"`
	StartedAt              *time.Time `json:"started_at,omitempty" db:"started_at" gorm:"column:started_at"`
	CompletedAt            *time.Time `json:"completed_at,omitempty" db:"completed_at" gorm:"column:completed_at"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	LastDryRunAt           *time.Time `json:"last_dry_run_at,omitempty" db:"last_dry_run_at" gorm:"column:last_dry_run_at"`                               // When batch dry run was last executed
	LastMigrationAttemptAt *time.Time `json:"last_migration_attempt_at,omitempty" db:"last_migration_attempt_at" gorm:"column:last_migration_attempt_at"` // When migration was last attempted

	// Migration Settings (batch-level defaults, repository settings take precedence)
	DestinationOrg  *string `json:"destination_org,omitempty" db:"destination_org" gorm:"column:destination_org"`        // Default destination org for repositories in this batch
	MigrationAPI    string  `json:"migration_api" db:"migration_api" gorm:"column:migration_api;not null"`               // Migration API to use: "GEI" or "ELM" (default: "GEI")
	ExcludeReleases bool    `json:"exclude_releases" db:"exclude_releases" gorm:"column:exclude_releases;default:false"` // Skip releases during migration (applies if repo doesn't override)
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

// RepositoryDependency represents a dependency relationship between repositories
// Used for batch planning to understand which repositories should be migrated together
type RepositoryDependency struct {
	ID                 int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	RepositoryID       int64     `json:"repository_id" db:"repository_id" gorm:"column:repository_id;not null;index"`
	DependencyFullName string    `json:"dependency_full_name" db:"dependency_full_name" gorm:"column:dependency_full_name;not null"` // org/repo format
	DependencyType     string    `json:"dependency_type" db:"dependency_type" gorm:"column:dependency_type;not null"`                // submodule, workflow, dependency_graph, package
	DependencyURL      string    `json:"dependency_url" db:"dependency_url" gorm:"column:dependency_url;not null"`                   // Original URL/reference
	IsLocal            bool      `json:"is_local" db:"is_local" gorm:"column:is_local;default:false"`                                // Whether dependency is within same enterprise
	DiscoveredAt       time.Time `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
	Metadata           *string   `json:"metadata,omitempty" db:"metadata" gorm:"column:metadata;type:text"` // JSON with type-specific details (branch, version, path, etc.)
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
	ID           int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Organization string    `json:"organization" db:"organization" gorm:"column:organization;not null;uniqueIndex:idx_github_teams_org_slug"`
	Slug         string    `json:"slug" db:"slug" gorm:"column:slug;not null;uniqueIndex:idx_github_teams_org_slug"`
	Name         string    `json:"name" db:"name" gorm:"column:name;not null"`
	Description  *string   `json:"description,omitempty" db:"description" gorm:"column:description;type:text"`
	Privacy      string    `json:"privacy" db:"privacy" gorm:"column:privacy;not null;default:closed"`
	DiscoveredAt time.Time `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
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
	ID           int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	TeamID       int64     `json:"team_id" db:"team_id" gorm:"column:team_id;not null;uniqueIndex:idx_github_team_repo"`
	RepositoryID int64     `json:"repository_id" db:"repository_id" gorm:"column:repository_id;not null;uniqueIndex:idx_github_team_repo;index"`
	Permission   string    `json:"permission" db:"permission" gorm:"column:permission;not null;default:pull"` // pull, push, admin, maintain, triage
	DiscoveredAt time.Time `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
}

// TableName specifies the table name for GitHubTeamRepository model
func (GitHubTeamRepository) TableName() string {
	return "github_team_repositories"
}

// GitHubTeamMember represents a member of a GitHub team
type GitHubTeamMember struct {
	ID           int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	TeamID       int64     `json:"team_id" db:"team_id" gorm:"column:team_id;not null;index"`
	Login        string    `json:"login" db:"login" gorm:"column:login;not null"`
	Role         string    `json:"role" db:"role" gorm:"column:role;not null"` // member, maintainer
	DiscoveredAt time.Time `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
}

// TableName specifies the table name for GitHubTeamMember model
func (GitHubTeamMember) TableName() string {
	return "github_team_members"
}

// GitHubUser represents a GitHub user discovered during profiling
// Used for user identity mapping and mannequin reclaim
type GitHubUser struct {
	ID             int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Login          string    `json:"login" db:"login" gorm:"column:login;not null;uniqueIndex"`
	Name           *string   `json:"name,omitempty" db:"name" gorm:"column:name"`
	Email          *string   `json:"email,omitempty" db:"email" gorm:"column:email;index"`
	AvatarURL      *string   `json:"avatar_url,omitempty" db:"avatar_url" gorm:"column:avatar_url"`
	SourceInstance string    `json:"source_instance" db:"source_instance" gorm:"column:source_instance;not null"` // Source GitHub URL (e.g., github.company.com)
	DiscoveredAt   time.Time `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`

	// Contribution stats (aggregated across all repositories)
	CommitCount     int `json:"commit_count" db:"commit_count" gorm:"column:commit_count;default:0"`
	IssueCount      int `json:"issue_count" db:"issue_count" gorm:"column:issue_count;default:0"`
	PRCount         int `json:"pr_count" db:"pr_count" gorm:"column:pr_count;default:0"`
	CommentCount    int `json:"comment_count" db:"comment_count" gorm:"column:comment_count;default:0"`
	RepositoryCount int `json:"repository_count" db:"repository_count" gorm:"column:repository_count;default:0"`
}

// TableName specifies the table name for GitHubUser model
func (GitHubUser) TableName() string {
	return "github_users"
}

// UserOrgMembership tracks which organizations a user belongs to
// This enables organizing users by source org for mannequin reclamation
type UserOrgMembership struct {
	ID           int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	UserLogin    string    `json:"user_login" db:"user_login" gorm:"column:user_login;not null;uniqueIndex:idx_user_org"`
	Organization string    `json:"organization" db:"organization" gorm:"column:organization;not null;uniqueIndex:idx_user_org;index"`
	Role         string    `json:"role" db:"role" gorm:"column:role;not null;default:member"` // member, admin
	DiscoveredAt time.Time `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null;autoCreateTime"`
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
	ID               int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	SourceLogin      string    `json:"source_login" db:"source_login" gorm:"column:source_login;not null;uniqueIndex"`
	SourceEmail      *string   `json:"source_email,omitempty" db:"source_email" gorm:"column:source_email;index"`
	SourceName       *string   `json:"source_name,omitempty" db:"source_name" gorm:"column:source_name"`
	SourceOrg        *string   `json:"source_org,omitempty" db:"source_org" gorm:"column:source_org;index"` // Organization where user was discovered
	DestinationLogin *string   `json:"destination_login,omitempty" db:"destination_login" gorm:"column:destination_login;index"`
	DestinationEmail *string   `json:"destination_email,omitempty" db:"destination_email" gorm:"column:destination_email"`
	MappingStatus    string    `json:"mapping_status" db:"mapping_status" gorm:"column:mapping_status;not null;default:unmapped;index"` // unmapped, mapped, reclaimed, skipped
	MannequinID      *string   `json:"mannequin_id,omitempty" db:"mannequin_id" gorm:"column:mannequin_id"`                             // GEI mannequin ID after migration
	MannequinLogin   *string   `json:"mannequin_login,omitempty" db:"mannequin_login" gorm:"column:mannequin_login"`                    // Mannequin login (e.g., mona-user-12345)
	ReclaimStatus    *string   `json:"reclaim_status,omitempty" db:"reclaim_status" gorm:"column:reclaim_status"`                       // pending, invited, completed, failed
	ReclaimError     *string   `json:"reclaim_error,omitempty" db:"reclaim_error" gorm:"column:reclaim_error;type:text"`                // Error message if reclaim failed
	MatchConfidence  *int      `json:"match_confidence,omitempty" db:"match_confidence" gorm:"column:match_confidence"`                 // Auto-match confidence score (0-100)
	MatchReason      *string   `json:"match_reason,omitempty" db:"match_reason" gorm:"column:match_reason"`                             // Why the match was made (email, login, name)
	CreatedAt        time.Time `json:"created_at" db:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName specifies the table name for UserMapping model
func (UserMapping) TableName() string {
	return "user_mappings"
}

// TeamMapping maps a source team to a destination team
type TeamMapping struct {
	ID                  int64      `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	SourceOrg           string     `json:"source_org" db:"source_org" gorm:"column:source_org;not null;uniqueIndex:idx_team_mapping_source"`
	SourceTeamSlug      string     `json:"source_team_slug" db:"source_team_slug" gorm:"column:source_team_slug;not null;uniqueIndex:idx_team_mapping_source"`
	SourceTeamName      *string    `json:"source_team_name,omitempty" db:"source_team_name" gorm:"column:source_team_name"`
	DestinationOrg      *string    `json:"destination_org,omitempty" db:"destination_org" gorm:"column:destination_org;index"`
	DestinationTeamSlug *string    `json:"destination_team_slug,omitempty" db:"destination_team_slug" gorm:"column:destination_team_slug"`
	DestinationTeamName *string    `json:"destination_team_name,omitempty" db:"destination_team_name" gorm:"column:destination_team_name"`
	MappingStatus       string     `json:"mapping_status" db:"mapping_status" gorm:"column:mapping_status;not null;default:unmapped;index"` // unmapped, mapped, skipped
	AutoCreated         bool       `json:"auto_created" db:"auto_created" gorm:"column:auto_created;default:false"`                         // True if team was auto-created during migration
	MigrationStatus     string     `json:"migration_status" db:"migration_status" gorm:"column:migration_status;default:pending;index"`     // pending, in_progress, completed, failed
	MigratedAt          *time.Time `json:"migrated_at,omitempty" db:"migrated_at" gorm:"column:migrated_at"`                                // When the team was created in destination
	ErrorMessage        *string    `json:"error_message,omitempty" db:"error_message" gorm:"column:error_message"`                          // Error details if migration failed
	ReposSynced         int        `json:"repos_synced" db:"repos_synced" gorm:"column:repos_synced;default:0"`                             // Count of repos with permissions applied
	// New fields for tracking partial vs. full migration
	TotalSourceRepos  int        `json:"total_source_repos" db:"total_source_repos" gorm:"column:total_source_repos;default:0"`           // Total repos this team has access to in source
	ReposEligible     int        `json:"repos_eligible" db:"repos_eligible" gorm:"column:repos_eligible;default:0"`                       // How many repos have been migrated and are available for sync
	TeamCreatedInDest bool       `json:"team_created_in_dest" db:"team_created_in_dest" gorm:"column:team_created_in_dest;default:false"` // Whether team exists in destination
	LastSyncedAt      *time.Time `json:"last_synced_at,omitempty" db:"last_synced_at" gorm:"column:last_synced_at"`                       // When permissions were last synced
	CreatedAt         time.Time  `json:"created_at" db:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
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
