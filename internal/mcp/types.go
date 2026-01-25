// Package mcp provides a Model Context Protocol server for the GitHub Migrator.
// It exposes migration-related tools to AI agents via the MCP protocol.
package mcp

import (
	"time"
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
	TotalScore    int                    `json:"total_score"`
	Rating        string                 `json:"rating"`
	Components    map[string]int         `json:"components"`
	Blockers      []string               `json:"blockers,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
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
	WaveNumber   int                 `json:"wave_number"`
	Repositories []RepositorySummary `json:"repositories"`
	TotalSize    int64               `json:"total_size_kb"`
	AvgComplexity float64            `json:"avg_complexity"`
	Dependencies  int                `json:"dependency_count"`
}

// BatchInfo represents batch information
type BatchInfo struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	Status          string    `json:"status"`
	RepositoryCount int       `json:"repository_count"`
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

// ------- Tool Output Types -------

// AnalyzeRepositoriesOutput is the output for analyze_repositories tool
type AnalyzeRepositoriesOutput struct {
	Repositories []RepositorySummary `json:"repositories"`
	TotalCount   int                 `json:"total_count"`
	Message      string              `json:"message"`
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
	Waves            []WavePlan `json:"waves"`
	TotalWaves       int        `json:"total_waves"`
	TotalRepositories int       `json:"total_repositories"`
	Message          string     `json:"message"`
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
