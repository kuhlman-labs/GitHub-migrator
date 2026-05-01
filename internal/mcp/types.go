// Package mcp provides a Model Context Protocol server for the GitHub Migrator.
// It exposes migration-related tools to AI agents via the MCP protocol.
package mcp

import (
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// Status constants for migration status checks
const (
	StatusPending           = "pending"
	StatusCompleted         = "completed"
	StatusComplete          = "complete"
	StatusMigrationComplete = "migration_complete"
)

// RepositorySummary represents a summarized view of a repository for tool responses
type RepositorySummary struct {
	FullName         string    `json:"full_name"`
	Organization     string    `json:"organization"`
	Status           string    `json:"status"`
	ComplexityScore  int       `json:"complexity_score"`
	ComplexityRating string    `json:"complexity_rating"`
	Size             int64     `json:"size_kb"`
	IsArchived       bool      `json:"is_archived"`
	IsFork           bool      `json:"is_fork"`
	BatchID          *int64    `json:"batch_id,omitempty"`
	BatchName        string    `json:"batch_name,omitempty"`
	MigratedAt       *string   `json:"migrated_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ComplexityBreakdown represents detailed complexity information
type ComplexityBreakdown struct {
	TotalScore      int            `json:"total_score"`
	Rating          string         `json:"rating"`
	Components      map[string]int `json:"components"`
	Blockers        []string       `json:"blockers,omitempty"`
	Warnings        []string       `json:"warnings,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

// DependencyInfo represents a repository dependency
type DependencyInfo struct {
	DependencyFullName string `json:"dependency_full_name"`
	DependencyType     string `json:"dependency_type"`
	IsLocal            bool   `json:"is_local"`
	IsMigrated         bool   `json:"is_migrated"`
	MigrationStatus    string `json:"migration_status,omitempty"`
}

// WavePlan represents a planned migration wave
type WavePlan struct {
	WaveNumber    int                 `json:"wave_number"`
	Repositories  []RepositorySummary `json:"repositories"`
	TotalSize     int64               `json:"total_size_kb"`
	AvgComplexity float64             `json:"avg_complexity"`
	Dependencies  int                 `json:"dependency_count"`
}

// BatchInfo represents batch information
type BatchInfo struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	Status          string    `json:"status"`
	RepositoryCount int       `json:"repository_count"`
	DestinationOrg  *string   `json:"destination_org,omitempty"`
	MigrationAPI    string    `json:"migration_api,omitempty"`
	ScheduledAt     *string   `json:"scheduled_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ------- Tool Input Types -------

// AnalyzeRepositoriesInput is the input for analyze_repositories tool
type AnalyzeRepositoriesInput struct {
	Organization  string `json:"organization,omitempty" jsonschema_description:"Filter by organization name"`
	Status        string `json:"status,omitempty" jsonschema:"enum=pending,enum=in_progress,enum=completed,enum=failed" jsonschema_description:"Filter by migration status"`
	MaxComplexity int    `json:"max_complexity,omitempty" jsonschema_description:"Maximum complexity score (1-100)"`
	MinComplexity int    `json:"min_complexity,omitempty" jsonschema_description:"Minimum complexity score (1-100)"`
	Limit         int    `json:"limit,omitempty" jsonschema:"default=20,minimum=1,maximum=100" jsonschema_description:"Maximum number of repositories to return"`
}

// GetComplexityBreakdownInput is the input for get_complexity_breakdown tool
type GetComplexityBreakdownInput struct {
	Repository string `json:"repository" jsonschema:"required" jsonschema_description:"Full repository name (org/repo)"`
}

// CheckDependenciesInput is the input for check_dependencies tool
type CheckDependenciesInput struct {
	Repository     string `json:"repository" jsonschema:"required" jsonschema_description:"Full repository name (org/repo)"`
	IncludeReverse bool   `json:"include_reverse,omitempty" jsonschema_description:"Include repositories that depend on this one"`
}

// FindPilotCandidatesInput is the input for find_pilot_candidates tool
type FindPilotCandidatesInput struct {
	MaxCount     int    `json:"max_count,omitempty" jsonschema:"default=10,minimum=1,maximum=50" jsonschema_description:"Maximum number of candidates to return"`
	Organization string `json:"organization,omitempty" jsonschema_description:"Filter by organization"`
}

// CreateBatchInput is the input for create_batch tool
type CreateBatchInput struct {
	Name         string   `json:"name" jsonschema:"required" jsonschema_description:"Name for the batch"`
	Description  string   `json:"description,omitempty" jsonschema_description:"Description of the batch"`
	Repositories []string `json:"repositories" jsonschema:"required" jsonschema_description:"List of repository full names to include"`
}

// PlanWavesInput is the input for plan_waves tool
type PlanWavesInput struct {
	WaveSize     int    `json:"wave_size,omitempty" jsonschema:"default=10,minimum=1,maximum=100" jsonschema_description:"Target number of repositories per wave"`
	Organization string `json:"organization,omitempty" jsonschema_description:"Filter by organization"`
}

// GetTeamRepositoriesInput is the input for get_team_repositories tool
type GetTeamRepositoriesInput struct {
	Team            string `json:"team" jsonschema:"required" jsonschema_description:"Team name in format org/team-slug"`
	IncludeMigrated bool   `json:"include_migrated,omitempty" jsonschema_description:"Include already migrated repositories"`
}

// GetMigrationStatusInput is the input for get_migration_status tool
type GetMigrationStatusInput struct {
	Repositories []string `json:"repositories" jsonschema:"required" jsonschema_description:"List of repository full names to check"`
}

// ScheduleBatchInput is the input for schedule_batch tool
type ScheduleBatchInput struct {
	BatchName   string `json:"batch_name" jsonschema:"required" jsonschema_description:"Name of the batch to schedule"`
	ScheduledAt string `json:"scheduled_at" jsonschema:"required" jsonschema_description:"ISO 8601 datetime for when to execute the batch"`
}

// ConfigureBatchInput is the input for configure_batch tool
type ConfigureBatchInput struct {
	BatchName      string `json:"batch_name,omitempty" jsonschema_description:"Name of the batch to configure"`
	BatchID        int64  `json:"batch_id,omitempty" jsonschema_description:"ID of the batch to configure (alternative to batch_name)"`
	DestinationOrg string `json:"destination_org,omitempty" jsonschema_description:"Destination organization within the configured enterprise"`
	MigrationAPI   string `json:"migration_api,omitempty" jsonschema:"enum=GEI,enum=ELM" jsonschema_description:"Migration API to use: GEI or ELM"`
}

// ------- Tool Output Types -------

// AnalyzeRepositoriesStats holds aggregate statistics for the analyze_repositories response
type AnalyzeRepositoriesStats struct {
	TotalSizeKB            int64          `json:"total_size_kb"`
	ComplexityDistribution map[string]int `json:"complexity_distribution"`
	FeatureCounts          map[string]int `json:"feature_counts"`
	BlockerCount           int            `json:"blocker_count"`
	ArchivedCount          int            `json:"archived_count"`
	ForkCount              int            `json:"fork_count"`
	StatusDistribution     map[string]int `json:"status_distribution"`
}

// AnalyzeRepositoriesOutput is the output for analyze_repositories tool
type AnalyzeRepositoriesOutput struct {
	Repositories []RepositorySummary       `json:"repositories"`
	TotalCount   int                       `json:"total_count"`
	Stats        *AnalyzeRepositoriesStats `json:"stats,omitempty"`
	Message      string                    `json:"message"`
}

// FeaturesSummary summarizes repository features for the get_repository_details tool
type FeaturesSummary struct {
	HasLFS               bool  `json:"has_lfs"`
	HasSubmodules        bool  `json:"has_submodules"`
	HasLargeFiles        bool  `json:"has_large_files"`
	LargeFileCount       int   `json:"large_file_count"`
	BranchCount          int   `json:"branch_count"`
	CommitCount          int   `json:"commit_count"`
	TotalSizeKB          int64 `json:"total_size_kb"`
	HasWiki              bool  `json:"has_wiki"`
	HasPages             bool  `json:"has_pages"`
	HasActions           bool  `json:"has_actions"`
	HasPackages          bool  `json:"has_packages"`
	HasDiscussions       bool  `json:"has_discussions"`
	HasProjects          bool  `json:"has_projects"`
	HasRulesets          bool  `json:"has_rulesets"`
	HasCodeScanning      bool  `json:"has_code_scanning"`
	HasDependabot        bool  `json:"has_dependabot"`
	HasSecretScanning    bool  `json:"has_secret_scanning"`
	HasCodeowners        bool  `json:"has_codeowners"`
	HasSelfHostedRunners bool  `json:"has_self_hosted_runners"`
	HasReleaseAssets     bool  `json:"has_release_assets"`
	BranchProtections    int   `json:"branch_protections"`
	WebhookCount         int   `json:"webhook_count"`
	EnvironmentCount     int   `json:"environment_count"`
	SecretCount          int   `json:"secret_count"`
	VariableCount        int   `json:"variable_count"`
	WorkflowCount        int   `json:"workflow_count"`
}

// GetRepositoryDetailsOutput is the output for get_repository_details tool
type GetRepositoryDetailsOutput struct {
	Repository          RepositorySummary        `json:"repository"`
	Complexity          ComplexityBreakdown      `json:"complexity"`
	Features            FeaturesSummary          `json:"features"`
	Dependencies        []DependencyInfo         `json:"dependencies"`
	DependencyCount     int                      `json:"dependency_count"`
	ReverseDependencies []DependencyInfo         `json:"reverse_dependencies,omitempty"`
	MigrationHistory    []MigrationHistoryRecord `json:"migration_history"`
	HistoryCount        int                      `json:"history_count"`
	Message             string                   `json:"message"`
}

// GetComplexityBreakdownOutput is the output for get_complexity_breakdown tool
type GetComplexityBreakdownOutput struct {
	Repository string              `json:"repository"`
	Breakdown  ComplexityBreakdown `json:"breakdown"`
	Message    string              `json:"message"`
}

// CheckDependenciesOutput is the output for check_dependencies tool
type CheckDependenciesOutput struct {
	Repository          string           `json:"repository"`
	Dependencies        []DependencyInfo `json:"dependencies"`
	DependencyCount     int              `json:"dependency_count"`
	ReverseDependencies []DependencyInfo `json:"reverse_dependencies,omitempty"`
	Message             string           `json:"message"`
}

// DependencyGraphNode represents a repository in the dependency graph
type DependencyGraphNode struct {
	FullName        string `json:"full_name"`
	Organization    string `json:"organization"`
	Status          string `json:"status"`
	DependsOnCount  int    `json:"depends_on_count"`
	DependedByCount int    `json:"depended_by_count"`
}

// DependencyGraphEdge represents a dependency relationship between two repos
type DependencyGraphEdge struct {
	Source         string `json:"source"`
	Target         string `json:"target"`
	DependencyType string `json:"dependency_type"`
}

// DependencyGraphStats holds summary statistics for the dependency graph
type DependencyGraphStats struct {
	TotalReposWithDeps   int `json:"total_repos_with_dependencies"`
	TotalLocalDeps       int `json:"total_local_dependencies"`
	CircularDependencies int `json:"circular_dependency_count"`
}

// GetDependencyGraphOutput is the output for get_dependency_graph tool
type GetDependencyGraphOutput struct {
	Nodes   []DependencyGraphNode `json:"nodes"`
	Edges   []DependencyGraphEdge `json:"edges"`
	Stats   DependencyGraphStats  `json:"stats"`
	Message string                `json:"message"`
}

// UpdateRepositoryStatusOutput is the output for update_repository_status tool
type UpdateRepositoryStatusOutput struct {
	UpdatedCount int      `json:"updated_count"`
	FailedCount  int      `json:"failed_count"`
	Errors       []string `json:"errors,omitempty"`
	Message      string   `json:"message"`
}

// FindPilotCandidatesOutput is the output for find_pilot_candidates tool
type FindPilotCandidatesOutput struct {
	Candidates []RepositorySummary `json:"candidates"`
	Count      int                 `json:"count"`
	Criteria   string              `json:"criteria"`
	Message    string              `json:"message"`
}

// CreateBatchOutput is the output for create_batch tool
type CreateBatchOutput struct {
	Batch   BatchInfo `json:"batch"`
	Success bool      `json:"success"`
	Message string    `json:"message"`
}

// PlanWavesOutput is the output for plan_waves tool
type PlanWavesOutput struct {
	Waves             []WavePlan `json:"waves"`
	TotalWaves        int        `json:"total_waves"`
	TotalRepositories int        `json:"total_repositories"`
	Message           string     `json:"message"`
}

// GetTeamRepositoriesOutput is the output for get_team_repositories tool
type GetTeamRepositoriesOutput struct {
	Team         string              `json:"team"`
	Repositories []RepositorySummary `json:"repositories"`
	Count        int                 `json:"count"`
	Message      string              `json:"message"`
}

// GetMigrationStatusOutput is the output for get_migration_status tool
type GetMigrationStatusOutput struct {
	Statuses []RepositorySummary `json:"statuses"`
	Count    int                 `json:"count"`
	Message  string              `json:"message"`
}

// ScheduleBatchOutput is the output for schedule_batch tool
type ScheduleBatchOutput struct {
	Batch   BatchInfo `json:"batch"`
	Success bool      `json:"success"`
	Message string    `json:"message"`
}

// ConfigureBatchOutput is the output for configure_batch tool
type ConfigureBatchOutput struct {
	Batch   BatchInfo `json:"batch"`
	Success bool      `json:"success"`
	Message string    `json:"message"`
}

// ------- Migration Execution Types -------

// StartMigrationInput is the input for start_migration tool
type StartMigrationInput struct {
	BatchName    string   `json:"batch_name,omitempty" jsonschema_description:"Name of batch to migrate"`
	BatchID      int64    `json:"batch_id,omitempty" jsonschema_description:"ID of batch to migrate"`
	Repository   string   `json:"repository,omitempty" jsonschema_description:"Single repository to migrate (format: org/repo)"`
	Repositories []string `json:"repositories,omitempty" jsonschema_description:"List of repositories to migrate"`
	DryRun       bool     `json:"dry_run" jsonschema:"default=true" jsonschema_description:"If true, perform dry-run only (default: true for safety)"`
}

// StartMigrationOutput is the output for start_migration tool
type StartMigrationOutput struct {
	BatchID      int64               `json:"batch_id,omitempty"`
	BatchName    string              `json:"batch_name,omitempty"`
	Repositories []RepositorySummary `json:"repositories"`
	QueuedCount  int                 `json:"queued_count"`
	SkippedCount int                 `json:"skipped_count"`
	DryRun       bool                `json:"dry_run"`
	Success      bool                `json:"success"`
	Message      string              `json:"message"`
	NextSteps    []string            `json:"next_steps,omitempty"`
}

// CancelMigrationInput is the input for cancel_migration tool
type CancelMigrationInput struct {
	BatchName  string `json:"batch_name,omitempty" jsonschema_description:"Name of batch to cancel"`
	BatchID    int64  `json:"batch_id,omitempty" jsonschema_description:"ID of batch to cancel"`
	Repository string `json:"repository,omitempty" jsonschema_description:"Single repository to cancel"`
}

// CancelMigrationOutput is the output for cancel_migration tool
type CancelMigrationOutput struct {
	BatchID        int64  `json:"batch_id,omitempty"`
	BatchName      string `json:"batch_name,omitempty"`
	Repository     string `json:"repository,omitempty"`
	CancelledCount int    `json:"cancelled_count"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
}

// GetMigrationProgressInput is the input for get_migration_progress tool
type GetMigrationProgressInput struct {
	BatchName  string `json:"batch_name,omitempty" jsonschema_description:"Name of batch to check progress"`
	BatchID    int64  `json:"batch_id,omitempty" jsonschema_description:"ID of batch to check progress"`
	Repository string `json:"repository,omitempty" jsonschema_description:"Single repository to check progress"`
}

// MigrationProgress represents progress information
type MigrationProgress struct {
	TotalCount      int     `json:"total_count"`
	PendingCount    int     `json:"pending_count"`
	QueuedCount     int     `json:"queued_count"`
	InProgressCount int     `json:"in_progress_count"`
	CompletedCount  int     `json:"completed_count"`
	FailedCount     int     `json:"failed_count"`
	SkippedCount    int     `json:"skipped_count"`
	PercentComplete float64 `json:"percent_complete"`
}

// GetMigrationProgressOutput is the output for get_migration_progress tool
type GetMigrationProgressOutput struct {
	BatchID      int64               `json:"batch_id,omitempty"`
	BatchName    string              `json:"batch_name,omitempty"`
	BatchStatus  string              `json:"batch_status,omitempty"`
	Repository   string              `json:"repository,omitempty"`
	Progress     MigrationProgress   `json:"progress"`
	Repositories []RepositorySummary `json:"repositories,omitempty"`
	Message      string              `json:"message"`
}

// ------- Destination Tool Types -------

// GetDestinationOutput is the output for get_destination tool
type GetDestinationOutput struct {
	BaseURL          string  `json:"base_url"`
	TokenConfigured  bool    `json:"token_configured"`
	EnterpriseSlug   *string `json:"enterprise_slug,omitempty"`
	AppID            *int64  `json:"app_id,omitempty"`
	AppKeyConfigured bool    `json:"app_key_configured"`
	InstallationID   *int64  `json:"installation_id,omitempty"`
	Configured       bool    `json:"configured"`
	Message          string  `json:"message"`
}

// ConfigureDestinationOutput is the output for configure_destination tool
type ConfigureDestinationOutput struct {
	BaseURL        string  `json:"base_url"`
	EnterpriseSlug *string `json:"enterprise_slug,omitempty"`
	Configured     bool    `json:"configured"`
	Message        string  `json:"message"`
}

// ------- Discovery Tool Types -------

// CreateSourceOutput is the output for create_source tool
type CreateSourceOutput struct {
	Source  *models.SourceResponse `json:"source"`
	Message string                 `json:"message"`
}

// ListSourcesOutput is the output for list_sources tool
type ListSourcesOutput struct {
	Sources []*models.SourceResponse `json:"sources"`
	Count   int                      `json:"count"`
	Message string                   `json:"message"`
}

// GetSourceOutput is the output for get_source tool
type GetSourceOutput struct {
	Source  *models.SourceResponse `json:"source"`
	Message string                 `json:"message"`
}

// StartDiscoveryOutput is the output for start_discovery tool
type StartDiscoveryOutput struct {
	ProgressID    int64  `json:"progress_id"`
	SourceID      int64  `json:"source_id"`
	DiscoveryType string `json:"discovery_type"`
	Target        string `json:"target"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	Hint          string `json:"hint"`
}

// GetDiscoveryProgressOutput is the output for get_discovery_progress tool
type GetDiscoveryProgressOutput struct {
	ProgressID        int64   `json:"progress_id,omitempty"`
	DiscoveryType     string  `json:"discovery_type,omitempty"`
	Target            string  `json:"target,omitempty"`
	Status            string  `json:"status"`
	Phase             string  `json:"phase,omitempty"`
	StartedAt         string  `json:"started_at,omitempty"`
	CompletedAt       *string `json:"completed_at,omitempty"`
	TotalOrgs         int     `json:"total_orgs,omitempty"`
	ProcessedOrgs     int     `json:"processed_orgs,omitempty"`
	CurrentOrg        string  `json:"current_org,omitempty"`
	TotalRepos        int     `json:"total_repos,omitempty"`
	ProcessedRepos    int     `json:"processed_repos,omitempty"`
	PercentComplete   float64 `json:"percent_complete,omitempty"`
	ErrorCount        int     `json:"error_count,omitempty"`
	LastError         *string `json:"last_error,omitempty"`
	RateLimitResetAt  *string `json:"rate_limit_reset_at,omitempty"`  // When rate limit will reset (only during waiting_for_rate_limit phase)
	RateLimitWaitSecs *int    `json:"rate_limit_wait_secs,omitempty"` // Seconds until rate limit resets
	Summary           *string `json:"summary,omitempty"`              // Post-discovery summary (only when completed)
	NextSteps         *string `json:"next_steps,omitempty"`           // Suggested next actions (only when completed)
	Message           string  `json:"message"`
}

// CancelDiscoveryOutput is the output for cancel_discovery tool
type CancelDiscoveryOutput struct {
	ProgressID int64  `json:"progress_id"`
	Target     string `json:"target"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

// ------- Readiness Types -------

// ReadinessCheckItem represents a single readiness check
type ReadinessCheckItem struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "fail", "warning"
	Detail string `json:"detail"`
}

// MigrationReadinessOutput is the output for get_migration_readiness tool
type MigrationReadinessOutput struct {
	ReadinessScore int                  `json:"readiness_score"` // 0-100
	Checklist      []ReadinessCheckItem `json:"checklist"`
	BlockingIssues []string             `json:"blocking_issues,omitempty"`
	NextSteps      []string             `json:"next_steps,omitempty"`
	Message        string               `json:"message"`
}

// UserMappingStatsOutput is the output for get_user_mapping_stats tool
type UserMappingStatsOutput struct {
	Stats   map[string]any `json:"stats"`
	Message string         `json:"message"`
}

// TeamMappingStatsOutput is the output for get_team_mapping_stats tool
type TeamMappingStatsOutput struct {
	Stats   map[string]any `json:"stats"`
	Message string         `json:"message"`
}

// RepositoryBlockerInfo represents a repository with migration blockers
type RepositoryBlockerInfo struct {
	FullName         string   `json:"full_name"`
	Organization     string   `json:"organization"`
	Status           string   `json:"status"`
	ComplexityScore  int      `json:"complexity_score"`
	Blockers         []string `json:"blockers"`
	RemediationSteps []string `json:"remediation_steps"`
}

// RepositoryBlockersOutput is the output for get_repository_blockers tool
type RepositoryBlockersOutput struct {
	Repositories []RepositoryBlockerInfo `json:"repositories"`
	TotalCount   int                     `json:"total_count"`
	Message      string                  `json:"message"`
}

// SuggestUserMappingsOutput is the output for suggest_user_mappings tool
type SuggestUserMappingsOutput struct {
	SyncedCount     int64          `json:"synced_count"`
	OrgsUpdated     int64          `json:"orgs_updated"`
	CurrentStats    map[string]any `json:"current_stats"`
	Recommendations []string       `json:"recommendations"`
	Message         string         `json:"message"`
}

// TeamMappingSuggestion represents a suggested team mapping
type TeamMappingSuggestion struct {
	SourceTeam      string `json:"source_team"`
	DestinationTeam string `json:"destination_team"`
	MatchReason     string `json:"match_reason"`
}

// SuggestTeamMappingsOutput is the output for suggest_team_mappings tool
type SuggestTeamMappingsOutput struct {
	SyncedCount    int64                   `json:"synced_count"`
	Suggestions    []TeamMappingSuggestion `json:"suggestions"`
	CurrentStats   map[string]any          `json:"current_stats"`
	DestinationOrg string                  `json:"destination_org"`
	Message        string                  `json:"message"`
}

// ------- Analytics Tool Types -------

// GetExecutiveSummaryInput is the input for get_executive_summary tool
type GetExecutiveSummaryInput struct {
	Organization string `json:"organization,omitempty" jsonschema_description:"Filter by organization name"`
	SourceID     int64  `json:"source_id,omitempty" jsonschema_description:"Filter by source ID"`
}

// ExecutiveSummaryVelocity represents migration velocity metrics
type ExecutiveSummaryVelocity struct {
	ReposPerDay  float64 `json:"repos_per_day"`
	ReposPerWeek float64 `json:"repos_per_week"`
}

// ExecutiveSummaryOrgStats represents per-organization stats in the summary
type ExecutiveSummaryOrgStats struct {
	Organization    string `json:"organization"`
	TotalRepos      int    `json:"total_repos"`
	MigratedCount   int    `json:"migrated_count"`
	FailedCount     int    `json:"failed_count"`
	PendingCount    int    `json:"pending_count"`
	InProgressCount int    `json:"in_progress_count"`
	ProgressPercent int    `json:"progress_percent"`
}

// GetExecutiveSummaryOutput is the output for get_executive_summary tool
type GetExecutiveSummaryOutput struct {
	TotalRepositories       int                        `json:"total_repositories"`
	MigratedCount           int                        `json:"migrated_count"`
	FailedCount             int                        `json:"failed_count"`
	PendingCount            int                        `json:"pending_count"`
	InProgressCount         int                        `json:"in_progress_count"`
	SuccessRate             float64                    `json:"success_rate_percent"`
	EstimatedCompletion     *string                    `json:"estimated_completion,omitempty"`
	Velocity                ExecutiveSummaryVelocity   `json:"velocity"`
	AvgMigrationTimeSeconds float64                    `json:"avg_migration_time_seconds"`
	StatusBreakdown         map[string]int             `json:"status_breakdown"`
	OrganizationBreakdown   []ExecutiveSummaryOrgStats `json:"organization_breakdown"`
	Message                 string                     `json:"message"`
}

// GetMigrationHistoryInput is the input for get_migration_history tool
type GetMigrationHistoryInput struct {
	Repository   string `json:"repository,omitempty" jsonschema_description:"Full repository name (org/repo)"`
	Limit        int    `json:"limit,omitempty" jsonschema:"default=20,minimum=1,maximum=100" jsonschema_description:"Maximum records to return"`
	StatusFilter string `json:"status_filter,omitempty" jsonschema:"enum=completed,enum=failed" jsonschema_description:"Filter by status category"`
}

// MigrationHistoryRecord represents a single migration history entry
type MigrationHistoryRecord struct {
	ID              int64   `json:"id"`
	RepositoryID    int64   `json:"repository_id,omitempty"`
	RepositoryName  string  `json:"repository_name"`
	Status          string  `json:"status"`
	Phase           string  `json:"phase,omitempty"`
	Message         string  `json:"message,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
	StartedAt       string  `json:"started_at,omitempty"`
	CompletedAt     *string `json:"completed_at,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
}

// GetMigrationHistoryOutput is the output for get_migration_history tool
type GetMigrationHistoryOutput struct {
	Repository string                   `json:"repository,omitempty"`
	Records    []MigrationHistoryRecord `json:"records"`
	Count      int                      `json:"count"`
	Message    string                   `json:"message"`
}

// GetMigrationLogsInput is the input for get_migration_logs tool
type GetMigrationLogsInput struct {
	Repository string `json:"repository" jsonschema:"required" jsonschema_description:"Full repository name (org/repo)"`
	Level      string `json:"level,omitempty" jsonschema:"enum=error,enum=warn,enum=info" jsonschema_description:"Filter by log level"`
	Phase      string `json:"phase,omitempty" jsonschema_description:"Filter by migration phase"`
	Limit      int    `json:"limit,omitempty" jsonschema:"default=50,minimum=1,maximum=500" jsonschema_description:"Maximum log entries to return"`
}

// MigrationLogEntry represents a single migration log entry
type MigrationLogEntry struct {
	ID           int64  `json:"id"`
	RepositoryID int64  `json:"repository_id"`
	Level        string `json:"level"`
	Phase        string `json:"phase"`
	Operation    string `json:"operation"`
	Message      string `json:"message"`
	Details      string `json:"details,omitempty"`
	Timestamp    string `json:"timestamp"`
}

// GetMigrationLogsOutput is the output for get_migration_logs tool
type GetMigrationLogsOutput struct {
	Repository string              `json:"repository"`
	Entries    []MigrationLogEntry `json:"entries"`
	Count      int                 `json:"count"`
	Message    string              `json:"message"`
}

// ActionItem represents a prioritized action item
type ActionItem struct {
	Severity    string   `json:"severity"` // critical, warning, info
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Count       int      `json:"count"`
	Details     []string `json:"details,omitempty"`
	Action      string   `json:"recommended_action"`
}

// GetActionItemsOutput is the output for get_action_items tool
type GetActionItemsOutput struct {
	Items   []ActionItem `json:"items"`
	Count   int          `json:"count"`
	Message string       `json:"message"`
}

// RollbackMigrationInput is the input for rollback_migration tool
type RollbackMigrationInput struct {
	Repository string `json:"repository" jsonschema:"required" jsonschema_description:"Full repository name (org/repo)"`
	Reason     string `json:"reason,omitempty" jsonschema_description:"Reason for rollback"`
}

// RollbackMigrationOutput is the output for rollback_migration tool
type RollbackMigrationOutput struct {
	Repository     string `json:"repository"`
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	Reason         string `json:"reason,omitempty"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
}
