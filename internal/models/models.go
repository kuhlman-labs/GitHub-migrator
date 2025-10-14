package models

import (
	"strings"
	"time"
)

// Repository represents a Git repository to be migrated
type Repository struct {
	ID        int64  `json:"id" db:"id"`
	FullName  string `json:"full_name" db:"full_name"` // org/repo
	Source    string `json:"source" db:"source"`       // "ghes", "gitlab", etc.
	SourceURL string `json:"source_url" db:"source_url"`

	// Git properties
	TotalSize         *int64     `json:"total_size,omitempty" db:"total_size"`
	LargestFile       *string    `json:"largest_file,omitempty" db:"largest_file"`
	LargestFileSize   *int64     `json:"largest_file_size,omitempty" db:"largest_file_size"`
	LargestCommit     *string    `json:"largest_commit,omitempty" db:"largest_commit"`
	LargestCommitSize *int64     `json:"largest_commit_size,omitempty" db:"largest_commit_size"`
	HasLFS            bool       `json:"has_lfs" db:"has_lfs"`
	HasSubmodules     bool       `json:"has_submodules" db:"has_submodules"`
	HasLargeFiles     bool       `json:"has_large_files" db:"has_large_files"` // Files > 100MB in history
	LargeFileCount    int        `json:"large_file_count" db:"large_file_count"`
	DefaultBranch     *string    `json:"default_branch,omitempty" db:"default_branch"`
	BranchCount       int        `json:"branch_count" db:"branch_count"`
	CommitCount       int        `json:"commit_count" db:"commit_count"`
	LastCommitSHA     *string    `json:"last_commit_sha,omitempty" db:"last_commit_sha"`
	LastCommitDate    *time.Time `json:"last_commit_date,omitempty" db:"last_commit_date"`

	// GitHub features
	IsArchived        bool `json:"is_archived" db:"is_archived"`
	HasWiki           bool `json:"has_wiki" db:"has_wiki"`
	HasPages          bool `json:"has_pages" db:"has_pages"`
	HasDiscussions    bool `json:"has_discussions" db:"has_discussions"`
	HasActions        bool `json:"has_actions" db:"has_actions"`
	HasProjects       bool `json:"has_projects" db:"has_projects"`
	BranchProtections int  `json:"branch_protections" db:"branch_protections"`
	EnvironmentCount  int  `json:"environment_count" db:"environment_count"`
	SecretCount       int  `json:"secret_count" db:"secret_count"`
	VariableCount     int  `json:"variable_count" db:"variable_count"`
	WebhookCount      int  `json:"webhook_count" db:"webhook_count"`

	// Contributors
	ContributorCount int     `json:"contributor_count" db:"contributor_count"`
	TopContributors  *string `json:"top_contributors,omitempty" db:"top_contributors"` // JSON array

	// Verification data (for post-migration verification)
	IssueCount       int `json:"issue_count" db:"issue_count"`
	PullRequestCount int `json:"pull_request_count" db:"pull_request_count"`
	TagCount         int `json:"tag_count" db:"tag_count"`
	OpenIssueCount   int `json:"open_issue_count" db:"open_issue_count"`
	OpenPRCount      int `json:"open_pr_count" db:"open_pr_count"`

	// Status Tracking
	Status   string `json:"status" db:"status"`
	BatchID  *int64 `json:"batch_id,omitempty" db:"batch_id"`
	Priority int    `json:"priority" db:"priority"` // 0=normal, 1=pilot

	// Migration Details
	DestinationURL      *string `json:"destination_url,omitempty" db:"destination_url"`
	DestinationFullName *string `json:"destination_full_name,omitempty" db:"destination_full_name"`

	// Lock Tracking (for failed migrations)
	SourceMigrationID *int64 `json:"source_migration_id,omitempty" db:"source_migration_id"` // GHES migration ID
	IsSourceLocked    bool   `json:"is_source_locked" db:"is_source_locked"`                 // Whether source repo is locked

	// Timestamps
	DiscoveredAt time.Time  `json:"discovered_at" db:"discovered_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	MigratedAt   *time.Time `json:"migrated_at,omitempty" db:"migrated_at"`
}

// MigrationStatus represents the status of a repository migration
type MigrationStatus string

const (
	StatusPending            MigrationStatus = "pending"
	StatusDryRunQueued       MigrationStatus = "dry_run_queued"
	StatusDryRunInProgress   MigrationStatus = "dry_run_in_progress"
	StatusDryRunComplete     MigrationStatus = "dry_run_complete"
	StatusDryRunFailed       MigrationStatus = "dry_run_failed"
	StatusPreMigration       MigrationStatus = "pre_migration"
	StatusArchiveGenerating  MigrationStatus = "archive_generating"
	StatusQueuedForMigration MigrationStatus = "queued_for_migration"
	StatusMigratingContent   MigrationStatus = "migrating_content"
	StatusMigrationComplete  MigrationStatus = "migration_complete"
	StatusMigrationFailed    MigrationStatus = "migration_failed"
	StatusPostMigration      MigrationStatus = "post_migration"
	StatusComplete           MigrationStatus = "complete"
	StatusRolledBack         MigrationStatus = "rolled_back"
)

// MigrationHistory tracks the migration lifecycle of a repository
type MigrationHistory struct {
	ID              int64      `json:"id" db:"id"`
	RepositoryID    int64      `json:"repository_id" db:"repository_id"`
	Status          string     `json:"status" db:"status"`
	Phase           string     `json:"phase" db:"phase"`
	Message         *string    `json:"message,omitempty" db:"message"`
	ErrorMessage    *string    `json:"error_message,omitempty" db:"error_message"`
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	DurationSeconds *int       `json:"duration_seconds,omitempty" db:"duration_seconds"`
}

// MigrationLog provides detailed logging for troubleshooting migrations
type MigrationLog struct {
	ID           int64     `json:"id" db:"id"`
	RepositoryID int64     `json:"repository_id" db:"repository_id"`
	HistoryID    *int64    `json:"history_id,omitempty" db:"history_id"`
	Level        string    `json:"level" db:"level"` // DEBUG, INFO, WARN, ERROR
	Phase        string    `json:"phase" db:"phase"`
	Operation    string    `json:"operation" db:"operation"`
	Message      string    `json:"message" db:"message"`
	Details      *string   `json:"details,omitempty" db:"details"` // Additional context, JSON or text
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// Batch represents a group of repositories to be migrated together
type Batch struct {
	ID              int64      `json:"id" db:"id"`
	Name            string     `json:"name" db:"name"`
	Description     *string    `json:"description,omitempty" db:"description"`
	Type            string     `json:"type" db:"type"` // "pilot", "wave_1", "wave_2", etc.
	RepositoryCount int        `json:"repository_count" db:"repository_count"`
	Status          string     `json:"status" db:"status"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at"`
	StartedAt       *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
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
