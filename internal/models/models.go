package models

import (
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
	TotalSize         *int64     `json:"total_size,omitempty" db:"total_size" gorm:"column:total_size"`
	LargestFile       *string    `json:"largest_file,omitempty" db:"largest_file" gorm:"column:largest_file"`
	LargestFileSize   *int64     `json:"largest_file_size,omitempty" db:"largest_file_size" gorm:"column:largest_file_size"`
	LargestCommit     *string    `json:"largest_commit,omitempty" db:"largest_commit" gorm:"column:largest_commit"`
	LargestCommitSize *int64     `json:"largest_commit_size,omitempty" db:"largest_commit_size" gorm:"column:largest_commit_size"`
	HasLFS            bool       `json:"has_lfs" db:"has_lfs" gorm:"column:has_lfs;default:false"`
	HasSubmodules     bool       `json:"has_submodules" db:"has_submodules" gorm:"column:has_submodules;default:false"`
	HasLargeFiles     bool       `json:"has_large_files" db:"has_large_files" gorm:"column:has_large_files;default:false"` // Files > 100MB in history
	LargeFileCount    int        `json:"large_file_count" db:"large_file_count" gorm:"column:large_file_count;default:0"`
	DefaultBranch     *string    `json:"default_branch,omitempty" db:"default_branch" gorm:"column:default_branch"`
	BranchCount       int        `json:"branch_count" db:"branch_count" gorm:"column:branch_count;default:0"`
	CommitCount       int        `json:"commit_count" db:"commit_count" gorm:"column:commit_count;default:0"`
	LastCommitSHA     *string    `json:"last_commit_sha,omitempty" db:"last_commit_sha" gorm:"column:last_commit_sha"`
	LastCommitDate    *time.Time `json:"last_commit_date,omitempty" db:"last_commit_date" gorm:"column:last_commit_date"`

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

	// Timestamps
	DiscoveredAt    time.Time  `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
	MigratedAt      *time.Time `json:"migrated_at,omitempty" db:"migrated_at" gorm:"column:migrated_at"`
	LastDiscoveryAt *time.Time `json:"last_discovery_at,omitempty" db:"last_discovery_at" gorm:"column:last_discovery_at"` // Latest discovery refresh
	LastDryRunAt    *time.Time `json:"last_dry_run_at,omitempty" db:"last_dry_run_at" gorm:"column:last_dry_run_at"`       // Latest dry run execution

	// Computed fields (not stored in DB)
	ComplexityScore     *int                 `json:"complexity_score,omitempty" db:"-" gorm:"-"`     // Calculated server-side
	ComplexityBreakdown *ComplexityBreakdown `json:"complexity_breakdown,omitempty" db:"-" gorm:"-"` // Individual component scores
}

// TableName specifies the table name for Repository model
func (Repository) TableName() string {
	return "repositories"
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
	SecurityPoints           int `json:"security_points"`            // 1 point if has GHAS features
	WebhooksPoints           int `json:"webhooks_points"`            // 1 point if has webhooks
	TagProtectionsPoints     int `json:"tag_protections_points"`     // 1 point if has tag protections
	BranchProtectionsPoints  int `json:"branch_protections_points"`  // 1 point if has branch protections
	RulesetsPoints           int `json:"rulesets_points"`            // 1 point if has rulesets
	PublicVisibilityPoints   int `json:"public_visibility_points"`   // 1 point if public
	InternalVisibilityPoints int `json:"internal_visibility_points"` // 1 point if internal
	CodeownersPoints         int `json:"codeowners_points"`          // 1 point if has CODEOWNERS
	ActivityPoints           int `json:"activity_points"`            // 0, 2, or 4 points based on quantile
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
	Details      *string   `json:"details,omitempty" db:"details" gorm:"column:details;type:text"` // Additional context, JSON or text
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
