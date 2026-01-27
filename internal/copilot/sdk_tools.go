// Package copilot provides the Copilot chat service integration using the official SDK.
package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// ToolTier represents the required authorization tier for a tool.
type ToolTier int

const (
	// TierAny allows any authenticated user (read-only operations)
	TierAny ToolTier = iota
	// TierSelfService requires self-service or higher (can migrate own repos)
	TierSelfService
	// TierAdmin requires admin tier (full migration rights)
	TierAdmin
)

// toolAuthRequirements maps tool names to their required authorization tier.
var toolAuthRequirements = map[string]ToolTier{
	// Read-only tools - any authenticated user
	"find_pilot_candidates":                   TierAny,
	"analyze_repositories":                    TierAny,
	"get_complexity_breakdown":                TierAny,
	"check_dependencies":                      TierAny,
	"get_top_complex_repositories":            TierAny,
	"get_repositories_with_most_dependencies": TierAny,
	"get_discovery_status":                    TierAny,
	"get_repository_details":                  TierAny,
	"validate_repository":                     TierAny,
	"list_batches":                            TierAny,
	"get_batch_details":                       TierAny,
	"get_migration_status":                    TierAny,
	"get_migration_progress":                  TierAny,
	"list_teams":                              TierAny,
	"get_team_repositories":                   TierAny,
	"get_team_migration_stats":                TierAny,
	"list_team_mappings":                      TierAny,
	"get_team_migration_execution_status":     TierAny,
	"list_mannequins":                         TierAny,
	"list_users":                              TierAny,
	"get_user_stats":                          TierAny,
	"list_user_mappings":                      TierAny,
	"get_analytics_summary":                   TierAny,
	"get_executive_report":                    TierAny,
	"get_permission_audit":                    TierAny,
	"list_organizations":                      TierAny,

	// Self-service tools - can operate on own repos
	"create_batch":            TierSelfService,
	"configure_batch":         TierSelfService,
	"add_repos_to_batch":      TierSelfService,
	"remove_repos_from_batch": TierSelfService,
	"schedule_batch":          TierSelfService,
	"plan_waves":              TierSelfService,

	// Admin-only tools - require full migration rights
	"start_discovery":            TierAdmin,
	"cancel_discovery":           TierAdmin,
	"discover_teams":             TierAdmin,
	"update_repository_status":   TierAdmin,
	"start_migration":            TierAdmin,
	"cancel_migration":           TierAdmin,
	"retry_batch_failures":       TierAdmin,
	"migrate_team":               TierAdmin,
	"suggest_team_mappings":      TierAdmin,
	"execute_team_migration":     TierAdmin,
	"send_mannequin_invitations": TierAdmin,
	"suggest_user_mappings":      TierAdmin,
	"update_user_mapping":        TierAdmin,
	"fetch_mannequins":           TierAdmin,
}

// checkToolAuthorization verifies the user has permission to execute a tool.
func (c *Client) checkToolAuthorization(toolName string, auth *AuthContext) error {
	if auth == nil {
		// No auth context - allow for backward compatibility
		// but log a warning
		c.logger.Warn("Tool executed without auth context", "tool", toolName)
		return nil
	}

	requiredTier, exists := toolAuthRequirements[toolName]
	if !exists {
		// Unknown tool - default to admin required
		requiredTier = TierAdmin
	}

	// Check based on tier
	switch requiredTier {
	case TierAny:
		// Any authenticated user can execute
		return nil
	case TierSelfService:
		if auth.Permissions.CanMigrateOwn || auth.Permissions.CanMigrateAll {
			return nil
		}
		return fmt.Errorf("permission denied: %s requires self-service or admin access", toolName)
	case TierAdmin:
		if auth.Permissions.CanMigrateAll {
			return nil
		}
		return fmt.Errorf("permission denied: %s requires admin access", toolName)
	}

	return nil
}

// Tool parameter types for SDK tool definitions.

// FindPilotParams defines parameters for find_pilot_candidates tool.
type FindPilotParams struct {
	MaxCount     int    `json:"max_count,omitempty" jsonschema:"Maximum number of candidates to return (default 10, max 50)"`
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization name"`
}

// AnalyzeRepositoriesParams defines parameters for analyze_repositories tool.
type AnalyzeRepositoriesParams struct {
	Organization  string `json:"organization,omitempty" jsonschema:"Filter by organization name"`
	Status        string `json:"status,omitempty" jsonschema:"Filter by migration status (pending, completed, failed, etc.)"`
	MaxComplexity int    `json:"max_complexity,omitempty" jsonschema:"Filter by maximum complexity score"`
}

// CreateBatchParams defines parameters for create_batch tool.
type CreateBatchParams struct {
	Name           string   `json:"name" jsonschema:"Name for the batch"`
	Repositories   []string `json:"repositories,omitempty" jsonschema:"List of repository full names to include"`
	DestinationOrg string   `json:"destination_org,omitempty" jsonschema:"Destination organization for migration"`
}

// ConfigureBatchParams defines parameters for configure_batch tool.
type ConfigureBatchParams struct {
	BatchName      string `json:"batch_name,omitempty" jsonschema:"Name of the batch to configure"`
	BatchID        int64  `json:"batch_id,omitempty" jsonschema:"ID of the batch to configure"`
	DestinationOrg string `json:"destination_org,omitempty" jsonschema:"Destination organization for migration"`
	MigrationAPI   string `json:"migration_api,omitempty" jsonschema:"Migration API to use (GEI or ELM)"`
}

// CheckDependenciesParams defines parameters for check_dependencies tool.
type CheckDependenciesParams struct {
	Repository     string `json:"repository" jsonschema:"Repository full name (org/repo)"`
	IncludeReverse bool   `json:"include_reverse,omitempty" jsonschema:"Include repositories that depend on this one"`
}

// PlanWavesParams defines parameters for plan_waves tool.
type PlanWavesParams struct {
	WaveSize     int    `json:"wave_size,omitempty" jsonschema:"Maximum repositories per wave (default 10, max 100)"`
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization name"`
}

// GetComplexityParams defines parameters for get_complexity_breakdown tool.
type GetComplexityParams struct {
	Repository string `json:"repository" jsonschema:"Repository full name (org/repo)"`
}

// GetTeamRepositoriesParams defines parameters for get_team_repositories tool.
type GetTeamRepositoriesParams struct {
	Team string `json:"team" jsonschema:"Team identifier in format org/team-slug"`
}

// GetMigrationStatusParams defines parameters for get_migration_status tool.
type GetMigrationStatusParams struct {
	Repositories []string `json:"repositories" jsonschema:"List of repository full names to check"`
}

// ScheduleBatchParams defines parameters for schedule_batch tool.
type ScheduleBatchParams struct {
	BatchName      string `json:"batch_name" jsonschema:"Name of the batch to schedule"`
	ScheduledAt    string `json:"scheduled_at,omitempty" jsonschema:"ISO 8601 datetime for scheduling (defaults to now)"`
	DestinationOrg string `json:"destination_org,omitempty" jsonschema:"Destination organization for migration"`
}

// StartMigrationParams defines parameters for start_migration tool.
type StartMigrationParams struct {
	BatchName  string `json:"batch_name,omitempty" jsonschema:"Name of the batch to migrate"`
	BatchID    int64  `json:"batch_id,omitempty" jsonschema:"ID of the batch to migrate"`
	Repository string `json:"repository,omitempty" jsonschema:"Single repository to migrate"`
	DryRun     bool   `json:"dry_run" jsonschema:"If true, perform a dry run (default: true for safety)"`
}

// CancelMigrationParams defines parameters for cancel_migration tool.
type CancelMigrationParams struct {
	BatchName  string `json:"batch_name,omitempty" jsonschema:"Name of the batch to cancel"`
	BatchID    int64  `json:"batch_id,omitempty" jsonschema:"ID of the batch to cancel"`
	Repository string `json:"repository,omitempty" jsonschema:"Single repository to cancel"`
}

// GetMigrationProgressParams defines parameters for get_migration_progress tool.
type GetMigrationProgressParams struct {
	BatchName  string `json:"batch_name,omitempty" jsonschema:"Name of the batch to check"`
	BatchID    int64  `json:"batch_id,omitempty" jsonschema:"ID of the batch to check"`
	Repository string `json:"repository,omitempty" jsonschema:"Single repository to check"`
}

// GetTopComplexRepositoriesParams defines parameters for finding most complex repos.
type GetTopComplexRepositoriesParams struct {
	Count        int    `json:"count,omitempty" jsonschema:"Number of repositories to return (default 10, max 50)"`
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization name"`
}

// GetRepositoriesWithMostDependenciesParams defines parameters for repos with most dependencies.
type GetRepositoriesWithMostDependenciesParams struct {
	Count        int    `json:"count,omitempty" jsonschema:"Number of repositories to return (default 10, max 50)"`
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization name"`
}

// GetTeamMigrationStatsParams defines parameters for team migration statistics.
type GetTeamMigrationStatsParams struct {
	Team string `json:"team" jsonschema:"Team identifier in format org/team-slug"`
}

// ListMannequinsParams defines parameters for listing mannequins.
type ListMannequinsParams struct {
	Organization  string `json:"organization" jsonschema:"Organization to list mannequins for"`
	ReclaimStatus string `json:"reclaim_status,omitempty" jsonschema:"Filter by status: pending, invited, completed, failed"`
	MaxCount      int    `json:"max_count,omitempty" jsonschema:"Maximum number to return (default 20, max 100)"`
}

// SendMannequinInvitationsParams defines parameters for sending reclaim invitations.
type SendMannequinInvitationsParams struct {
	Organization string   `json:"organization" jsonschema:"Organization containing the mannequins"`
	SourceLogins []string `json:"source_logins,omitempty" jsonschema:"Specific source logins to invite (if empty, invites all eligible)"`
	DryRun       bool     `json:"dry_run" jsonschema:"If true, only show what would be invited without sending"`
}

// ListTeamsParams defines parameters for listing teams.
type ListTeamsParams struct {
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization name"`
	MaxCount     int    `json:"max_count,omitempty" jsonschema:"Maximum number to return (default 20, max 100)"`
}

// MigrateTeamParams defines parameters for migrating a team.
type MigrateTeamParams struct {
	Team                string `json:"team" jsonschema:"Team identifier in format org/team-slug"`
	DestinationOrg      string `json:"destination_org,omitempty" jsonschema:"Destination organization"`
	DestinationTeam     string `json:"destination_team,omitempty" jsonschema:"Destination team slug (defaults to same as source)"`
	IncludeMembers      bool   `json:"include_members" jsonschema:"Whether to migrate team members"`
	IncludeRepositories bool   `json:"include_repositories" jsonschema:"Whether to assign repositories to team in destination"`
}

// --- Group 1: Discovery Operations ---

// StartDiscoveryParams defines parameters for starting discovery.
type StartDiscoveryParams struct {
	Organization   string `json:"organization,omitempty" jsonschema:"Organization name to discover"`
	EnterpriseSlug string `json:"enterprise_slug,omitempty" jsonschema:"Enterprise slug for enterprise-wide discovery"`
	SourceID       *int64 `json:"source_id,omitempty" jsonschema:"Source ID for multi-source environments"`
}

// GetDiscoveryStatusParams defines parameters for checking discovery status.
// Note: Empty structs need at least one field for valid JSON schema generation.
type GetDiscoveryStatusParams struct {
	// Verbose returns additional details when true (optional)
	Verbose bool `json:"verbose,omitempty" jsonschema:"Return additional discovery details"`
}

// CancelDiscoveryParams defines parameters for canceling discovery.
// Note: Empty structs need at least one field for valid JSON schema generation.
type CancelDiscoveryParams struct {
	// Force cancels even if in critical section (optional)
	Force bool `json:"force,omitempty" jsonschema:"Force cancel even if in critical section"`
}

// DiscoverTeamsParams defines parameters for team discovery.
type DiscoverTeamsParams struct {
	Organization string `json:"organization" jsonschema:"Organization to discover teams for"`
}

// --- Group 2: Repository Operations ---

// GetRepositoryDetailsParams defines parameters for getting repository details.
type GetRepositoryDetailsParams struct {
	Repository string `json:"repository" jsonschema:"Repository full name (org/repo)"`
}

// ValidateRepositoryParams defines parameters for validating a repository.
type ValidateRepositoryParams struct {
	Repository string `json:"repository" jsonschema:"Repository full name (org/repo)"`
}

// UpdateRepositoryStatusParams defines parameters for updating repository status.
type UpdateRepositoryStatusParams struct {
	Repository string `json:"repository" jsonschema:"Repository full name (org/repo)"`
	Status     string `json:"status" jsonschema:"New status (pending, wont_migrate, remediation_required, etc.)"`
	Reason     string `json:"reason,omitempty" jsonschema:"Reason for the status change"`
}

// --- Group 3: Batch Operations ---

// ListBatchesParams defines parameters for listing batches.
type ListBatchesParams struct {
	Status string `json:"status,omitempty" jsonschema:"Filter by batch status (draft, scheduled, in_progress, completed, failed)"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum number of batches to return (default 20)"`
}

// GetBatchDetailsParams defines parameters for getting batch details.
type GetBatchDetailsParams struct {
	BatchID   int64  `json:"batch_id,omitempty" jsonschema:"ID of the batch"`
	BatchName string `json:"batch_name,omitempty" jsonschema:"Name of the batch"`
}

// AddReposToBatchParams defines parameters for adding repos to a batch.
type AddReposToBatchParams struct {
	BatchID      int64    `json:"batch_id,omitempty" jsonschema:"ID of the batch"`
	BatchName    string   `json:"batch_name,omitempty" jsonschema:"Name of the batch"`
	Repositories []string `json:"repositories" jsonschema:"List of repository full names to add"`
}

// RemoveReposFromBatchParams defines parameters for removing repos from a batch.
type RemoveReposFromBatchParams struct {
	BatchID      int64    `json:"batch_id,omitempty" jsonschema:"ID of the batch"`
	BatchName    string   `json:"batch_name,omitempty" jsonschema:"Name of the batch"`
	Repositories []string `json:"repositories" jsonschema:"List of repository full names to remove"`
}

// RetryBatchFailuresParams defines parameters for retrying failed migrations.
type RetryBatchFailuresParams struct {
	BatchID   int64  `json:"batch_id,omitempty" jsonschema:"ID of the batch"`
	BatchName string `json:"batch_name,omitempty" jsonschema:"Name of the batch"`
}

// --- Group 4: User Mapping Operations ---

// ListUsersParams defines parameters for listing discovered users.
type ListUsersParams struct {
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization"`
	MaxCount     int    `json:"max_count,omitempty" jsonschema:"Maximum number to return (default 50)"`
}

// GetUserStatsParams defines parameters for user statistics.
type GetUserStatsParams struct {
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization"`
}

// ListUserMappingsParams defines parameters for listing user mappings.
type ListUserMappingsParams struct {
	Organization string `json:"organization,omitempty" jsonschema:"Filter by source organization"`
	Status       string `json:"status,omitempty" jsonschema:"Filter by mapping status (mapped, unmapped, suggested)"`
	MaxCount     int    `json:"max_count,omitempty" jsonschema:"Maximum number to return (default 50)"`
}

// SuggestUserMappingsParams defines parameters for suggesting user mappings.
type SuggestUserMappingsParams struct {
	Organization    string `json:"organization,omitempty" jsonschema:"Filter by source organization"`
	DestinationOrg  string `json:"destination_org" jsonschema:"Destination organization to match against"`
	OverwriteManual bool   `json:"overwrite_manual,omitempty" jsonschema:"Whether to overwrite manually set mappings"`
}

// UpdateUserMappingParams defines parameters for updating a user mapping.
type UpdateUserMappingParams struct {
	SourceLogin      string `json:"source_login" jsonschema:"Source user login"`
	DestinationLogin string `json:"destination_login,omitempty" jsonschema:"Destination user login to map to"`
	Status           string `json:"status,omitempty" jsonschema:"Mapping status"`
}

// FetchMannequinsParams defines parameters for fetching mannequins.
type FetchMannequinsParams struct {
	Organization string `json:"organization" jsonschema:"Destination organization to fetch mannequins from"`
}

// --- Group 5: Team Mapping Operations ---

// ListTeamMappingsParams defines parameters for listing team mappings.
type ListTeamMappingsParams struct {
	SourceOrg string `json:"source_org,omitempty" jsonschema:"Filter by source organization"`
	Status    string `json:"status,omitempty" jsonschema:"Filter by migration status"`
	MaxCount  int    `json:"max_count,omitempty" jsonschema:"Maximum number to return (default 50)"`
}

// SuggestTeamMappingsParams defines parameters for suggesting team mappings.
type SuggestTeamMappingsParams struct {
	SourceOrg      string `json:"source_org,omitempty" jsonschema:"Filter by source organization"`
	DestinationOrg string `json:"destination_org" jsonschema:"Destination organization to match against"`
}

// ExecuteTeamMigrationParams defines parameters for executing team migration.
type ExecuteTeamMigrationParams struct {
	SourceOrg      string `json:"source_org,omitempty" jsonschema:"Filter by source organization"`
	DestinationOrg string `json:"destination_org" jsonschema:"Destination organization"`
	DryRun         bool   `json:"dry_run" jsonschema:"If true, only validate without executing"`
}

// GetTeamMigrationExecutionStatusParams defines parameters for team migration status.
// Note: Empty structs need at least one field for valid JSON schema generation.
type GetTeamMigrationExecutionStatusParams struct {
	// IncludeHistory returns historical execution data when true (optional)
	IncludeHistory bool `json:"include_history,omitempty" jsonschema:"Include historical execution data"`
}

// --- Group 6: Analytics and Reporting ---

// GetAnalyticsSummaryParams defines parameters for analytics summary.
type GetAnalyticsSummaryParams struct {
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization"`
	SourceID     *int64 `json:"source_id,omitempty" jsonschema:"Filter by source ID"`
}

// GetExecutiveReportParams defines parameters for executive report.
type GetExecutiveReportParams struct {
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization"`
}

// GetPermissionAuditParams defines parameters for permission audit.
type GetPermissionAuditParams struct {
	Organization string `json:"organization,omitempty" jsonschema:"Filter by organization"`
	Team         string `json:"team,omitempty" jsonschema:"Filter by team (org/team-slug format)"`
}

// --- Group 7: Organization Operations ---

// ListOrganizationsParams defines parameters for listing organizations.
type ListOrganizationsParams struct {
	SourceID *int64 `json:"source_id,omitempty" jsonschema:"Filter by source ID"`
}

// registerTools registers all migration tools with the SDK client.
func (c *Client) registerTools() {
	c.tools = []copilot.Tool{
		// Discovery and analysis tools
		c.createFindPilotCandidatesTool(),
		c.createAnalyzeRepositoriesTool(),
		c.createGetComplexityTool(),
		c.createCheckDependenciesTool(),
		c.createGetTopComplexRepositoriesTool(),
		c.createGetRepositoriesWithMostDependenciesTool(),

		// Discovery operations
		c.createStartDiscoveryTool(),
		c.createGetDiscoveryStatusTool(),
		c.createCancelDiscoveryTool(),
		c.createDiscoverTeamsTool(),

		// Repository operations
		c.createGetRepositoryDetailsTool(),
		c.createValidateRepositoryTool(),
		c.createUpdateRepositoryStatusTool(),

		// Batch and migration planning tools
		c.createBatchTool(),
		c.createConfigureBatchTool(),
		c.createPlanWavesTool(),
		c.createScheduleBatchTool(),
		c.createListBatchesTool(),
		c.createGetBatchDetailsTool(),
		c.createAddReposToBatchTool(),
		c.createRemoveReposFromBatchTool(),
		c.createRetryBatchFailuresTool(),

		// Migration execution tools
		c.createStartMigrationTool(),
		c.createCancelMigrationTool(),
		c.createGetMigrationStatusTool(),
		c.createGetMigrationProgressTool(),

		// Team tools
		c.createListTeamsTool(),
		c.createGetTeamRepositoriesTool(),
		c.createGetTeamMigrationStatsTool(),
		c.createMigrateTeamTool(),

		// Team mapping tools
		c.createListTeamMappingsTool(),
		c.createSuggestTeamMappingsTool(),
		c.createExecuteTeamMigrationTool(),
		c.createGetTeamMigrationExecutionStatusTool(),

		// User/Mannequin tools
		c.createListMannequinsTool(),
		c.createSendMannequinInvitationsTool(),
		c.createListUsersTool(),
		c.createGetUserStatsTool(),
		c.createListUserMappingsTool(),
		c.createSuggestUserMappingsTool(),
		c.createUpdateUserMappingTool(),
		c.createFetchMannequinsTool(),

		// Analytics and reporting tools
		c.createGetAnalyticsSummaryTool(),
		c.createGetExecutiveReportTool(),
		c.createGetPermissionAuditTool(),

		// Organization tools
		c.createListOrganizationsTool(),
	}
}

func (c *Client) createFindPilotCandidatesTool() copilot.Tool {
	return copilot.DefineTool(
		"find_pilot_candidates",
		"Identify repositories suitable for pilot migration based on low complexity, few dependencies, and low risk",
		func(params FindPilotParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeFindPilotCandidates(context.Background(), params)
		},
	)
}

func (c *Client) createAnalyzeRepositoriesTool() copilot.Tool {
	return copilot.DefineTool(
		"analyze_repositories",
		"Query and analyze repositories based on various criteria like complexity, migration status, and organization",
		func(params AnalyzeRepositoriesParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeAnalyzeRepositories(context.Background(), params)
		},
	)
}

func (c *Client) createBatchTool() copilot.Tool {
	return copilot.DefineTool(
		"create_batch",
		"Create a new migration batch with specified repositories",
		func(params CreateBatchParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeCreateBatch(context.Background(), params)
		},
	)
}

func (c *Client) createConfigureBatchTool() copilot.Tool {
	return copilot.DefineTool(
		"configure_batch",
		"Configure batch settings including destination organization and migration API (GEI or ELM)",
		func(params ConfigureBatchParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeConfigureBatch(context.Background(), params)
		},
	)
}

func (c *Client) createCheckDependenciesTool() copilot.Tool {
	return copilot.DefineTool(
		"check_dependencies",
		"Find all dependencies for a repository and check their migration status",
		func(params CheckDependenciesParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeCheckDependencies(context.Background(), params)
		},
	)
}

func (c *Client) createPlanWavesTool() copilot.Tool {
	return copilot.DefineTool(
		"plan_waves",
		"Generate an optimal migration wave plan that respects dependencies and minimizes risk",
		func(params PlanWavesParams, inv copilot.ToolInvocation) (any, error) {
			return c.executePlanWaves(context.Background(), params)
		},
	)
}

func (c *Client) createGetComplexityTool() copilot.Tool {
	return copilot.DefineTool(
		"get_complexity_breakdown",
		"Get detailed complexity breakdown for a specific repository",
		func(params GetComplexityParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetComplexityBreakdown(context.Background(), params)
		},
	)
}

func (c *Client) createGetTeamRepositoriesTool() copilot.Tool {
	return copilot.DefineTool(
		"get_team_repositories",
		"Find all repositories owned by or associated with a specific team",
		func(params GetTeamRepositoriesParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetTeamRepositories(context.Background(), params)
		},
	)
}

func (c *Client) createGetMigrationStatusTool() copilot.Tool {
	return copilot.DefineTool(
		"get_migration_status",
		"Get the current migration status for specific repositories",
		func(params GetMigrationStatusParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetMigrationStatus(context.Background(), params)
		},
	)
}

func (c *Client) createScheduleBatchTool() copilot.Tool {
	return copilot.DefineTool(
		"schedule_batch",
		"Schedule a batch for migration execution at a specific date/time",
		func(params ScheduleBatchParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeScheduleBatch(context.Background(), params)
		},
	)
}

func (c *Client) createStartMigrationTool() copilot.Tool {
	return copilot.DefineTool(
		"start_migration",
		"Start a migration (dry-run or production) for a batch or repository. Defaults to dry-run for safety.",
		func(params StartMigrationParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeStartMigration(context.Background(), params)
		},
	)
}

func (c *Client) createCancelMigrationTool() copilot.Tool {
	return copilot.DefineTool(
		"cancel_migration",
		"Cancel a running migration for a batch or specific repository",
		func(params CancelMigrationParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeCancelMigration(context.Background(), params)
		},
	)
}

func (c *Client) createGetMigrationProgressTool() copilot.Tool {
	return copilot.DefineTool(
		"get_migration_progress",
		"Get real-time progress of running migrations for a batch or specific repository",
		func(params GetMigrationProgressParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetMigrationProgress(context.Background(), params)
		},
	)
}

func (c *Client) createGetTopComplexRepositoriesTool() copilot.Tool {
	return copilot.DefineTool(
		"get_top_complex_repositories",
		"Find the most complex repositories by complexity score. Use to identify challenging migrations.",
		func(params GetTopComplexRepositoriesParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetTopComplexRepositories(context.Background(), params)
		},
	)
}

func (c *Client) createGetRepositoriesWithMostDependenciesTool() copilot.Tool {
	return copilot.DefineTool(
		"get_repositories_with_most_dependencies",
		"Find repositories with the most dependencies. Helps identify repos that should migrate last.",
		func(params GetRepositoriesWithMostDependenciesParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetRepositoriesWithMostDependencies(context.Background(), params)
		},
	)
}

func (c *Client) createListTeamsTool() copilot.Tool {
	return copilot.DefineTool(
		"list_teams",
		"List teams in an organization with their member and repository counts",
		func(params ListTeamsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeListTeams(context.Background(), params)
		},
	)
}

func (c *Client) createGetTeamMigrationStatsTool() copilot.Tool {
	return copilot.DefineTool(
		"get_team_migration_stats",
		"Get migration statistics for a team including how many repos have been migrated",
		func(params GetTeamMigrationStatsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetTeamMigrationStats(context.Background(), params)
		},
	)
}

func (c *Client) createMigrateTeamTool() copilot.Tool {
	return copilot.DefineTool(
		"migrate_team",
		"Create or update a team in the destination organization with optional member and repository assignments",
		func(params MigrateTeamParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeMigrateTeam(context.Background(), params)
		},
	)
}

func (c *Client) createListMannequinsTool() copilot.Tool {
	return copilot.DefineTool(
		"list_mannequins",
		"List mannequin users in an organization that need to be reclaimed after migration",
		func(params ListMannequinsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeListMannequins(context.Background(), params)
		},
	)
}

func (c *Client) createSendMannequinInvitationsTool() copilot.Tool {
	return copilot.DefineTool(
		"send_mannequin_invitations",
		"Send reclaim invitations to mannequin users so they can claim their contributions",
		func(params SendMannequinInvitationsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeSendMannequinInvitations(context.Background(), params)
		},
	)
}

// --- Group 1: Discovery Operations Tool Creators ---

func (c *Client) createStartDiscoveryTool() copilot.Tool {
	return copilot.DefineTool(
		"start_discovery",
		"Start repository discovery for an organization or enterprise. Discovers repos, teams, and users.",
		func(params StartDiscoveryParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeStartDiscovery(context.Background(), params)
		},
	)
}

func (c *Client) createGetDiscoveryStatusTool() copilot.Tool {
	return copilot.DefineTool(
		"get_discovery_status",
		"Check the current status of repository discovery including progress and any errors",
		func(params GetDiscoveryStatusParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetDiscoveryStatus(context.Background(), params)
		},
	)
}

func (c *Client) createCancelDiscoveryTool() copilot.Tool {
	return copilot.DefineTool(
		"cancel_discovery",
		"Cancel a running discovery operation",
		func(params CancelDiscoveryParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeCancelDiscovery(context.Background(), params)
		},
	)
}

func (c *Client) createDiscoverTeamsTool() copilot.Tool {
	return copilot.DefineTool(
		"discover_teams",
		"Discover teams and their members for a specific organization",
		func(params DiscoverTeamsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeDiscoverTeams(context.Background(), params)
		},
	)
}

// --- Group 2: Repository Operations Tool Creators ---

func (c *Client) createGetRepositoryDetailsTool() copilot.Tool {
	return copilot.DefineTool(
		"get_repository_details",
		"Get detailed information about a repository including validation status and blockers",
		func(params GetRepositoryDetailsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetRepositoryDetails(context.Background(), params)
		},
	)
}

func (c *Client) createValidateRepositoryTool() copilot.Tool {
	return copilot.DefineTool(
		"validate_repository",
		"Run validation checks on a repository to identify migration blockers",
		func(params ValidateRepositoryParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeValidateRepository(context.Background(), params)
		},
	)
}

func (c *Client) createUpdateRepositoryStatusTool() copilot.Tool {
	return copilot.DefineTool(
		"update_repository_status",
		"Update the migration status of a repository (e.g., mark as wont_migrate or pending)",
		func(params UpdateRepositoryStatusParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeUpdateRepositoryStatus(context.Background(), params)
		},
	)
}

// --- Group 3: Batch Operations Tool Creators ---

func (c *Client) createListBatchesTool() copilot.Tool {
	return copilot.DefineTool(
		"list_batches",
		"List all migration batches with their status and repository counts",
		func(params ListBatchesParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeListBatches(context.Background(), params)
		},
	)
}

func (c *Client) createGetBatchDetailsTool() copilot.Tool {
	return copilot.DefineTool(
		"get_batch_details",
		"Get detailed information about a specific batch including all repositories",
		func(params GetBatchDetailsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetBatchDetails(context.Background(), params)
		},
	)
}

func (c *Client) createAddReposToBatchTool() copilot.Tool {
	return copilot.DefineTool(
		"add_repos_to_batch",
		"Add repositories to an existing migration batch",
		func(params AddReposToBatchParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeAddReposToBatch(context.Background(), params)
		},
	)
}

func (c *Client) createRemoveReposFromBatchTool() copilot.Tool {
	return copilot.DefineTool(
		"remove_repos_from_batch",
		"Remove repositories from a migration batch",
		func(params RemoveReposFromBatchParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeRemoveReposFromBatch(context.Background(), params)
		},
	)
}

func (c *Client) createRetryBatchFailuresTool() copilot.Tool {
	return copilot.DefineTool(
		"retry_batch_failures",
		"Reset failed repositories in a batch to pending status for retry",
		func(params RetryBatchFailuresParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeRetryBatchFailures(context.Background(), params)
		},
	)
}

// --- Group 4: User Mapping Operations Tool Creators ---

func (c *Client) createListUsersTool() copilot.Tool {
	return copilot.DefineTool(
		"list_users",
		"List discovered users from source organizations",
		func(params ListUsersParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeListUsers(context.Background(), params)
		},
	)
}

func (c *Client) createGetUserStatsTool() copilot.Tool {
	return copilot.DefineTool(
		"get_user_stats",
		"Get statistics about user discovery and mapping progress",
		func(params GetUserStatsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetUserStats(context.Background(), params)
		},
	)
}

func (c *Client) createListUserMappingsTool() copilot.Tool {
	return copilot.DefineTool(
		"list_user_mappings",
		"List user mappings between source and destination users",
		func(params ListUserMappingsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeListUserMappings(context.Background(), params)
		},
	)
}

func (c *Client) createSuggestUserMappingsTool() copilot.Tool {
	return copilot.DefineTool(
		"suggest_user_mappings",
		"Auto-suggest user mappings based on matching criteria like email and username",
		func(params SuggestUserMappingsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeSuggestUserMappings(context.Background(), params)
		},
	)
}

func (c *Client) createUpdateUserMappingTool() copilot.Tool {
	return copilot.DefineTool(
		"update_user_mapping",
		"Update a user mapping to set or change the destination user",
		func(params UpdateUserMappingParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeUpdateUserMapping(context.Background(), params)
		},
	)
}

func (c *Client) createFetchMannequinsTool() copilot.Tool {
	return copilot.DefineTool(
		"fetch_mannequins",
		"Fetch mannequin users from a destination organization for attribution mapping",
		func(params FetchMannequinsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeFetchMannequins(context.Background(), params)
		},
	)
}

// --- Group 5: Team Mapping Operations Tool Creators ---

func (c *Client) createListTeamMappingsTool() copilot.Tool {
	return copilot.DefineTool(
		"list_team_mappings",
		"List team mappings between source and destination organizations",
		func(params ListTeamMappingsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeListTeamMappings(context.Background(), params)
		},
	)
}

func (c *Client) createSuggestTeamMappingsTool() copilot.Tool {
	return copilot.DefineTool(
		"suggest_team_mappings",
		"Auto-suggest team mappings based on matching team names",
		func(params SuggestTeamMappingsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeSuggestTeamMappings(context.Background(), params)
		},
	)
}

func (c *Client) createExecuteTeamMigrationTool() copilot.Tool {
	return copilot.DefineTool(
		"execute_team_migration",
		"Execute team migration to create teams in destination and assign members/repos",
		func(params ExecuteTeamMigrationParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeExecuteTeamMigration(context.Background(), params)
		},
	)
}

func (c *Client) createGetTeamMigrationExecutionStatusTool() copilot.Tool {
	return copilot.DefineTool(
		"get_team_migration_execution_status",
		"Get the current status of team migration execution",
		func(params GetTeamMigrationExecutionStatusParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetTeamMigrationExecutionStatus(context.Background(), params)
		},
	)
}

// --- Group 6: Analytics and Reporting Tool Creators ---

func (c *Client) createGetAnalyticsSummaryTool() copilot.Tool {
	return copilot.DefineTool(
		"get_analytics_summary",
		"Get comprehensive analytics about migration progress, complexity distribution, and velocity",
		func(params GetAnalyticsSummaryParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetAnalyticsSummary(context.Background(), params)
		},
	)
}

func (c *Client) createGetExecutiveReportTool() copilot.Tool {
	return copilot.DefineTool(
		"get_executive_report",
		"Generate an executive summary report with high-level migration metrics",
		func(params GetExecutiveReportParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetExecutiveReport(context.Background(), params)
		},
	)
}

func (c *Client) createGetPermissionAuditTool() copilot.Tool {
	return copilot.DefineTool(
		"get_permission_audit",
		"Audit team and repository permissions across the organization",
		func(params GetPermissionAuditParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeGetPermissionAudit(context.Background(), params)
		},
	)
}

// --- Group 7: Organization Operations Tool Creators ---

func (c *Client) createListOrganizationsTool() copilot.Tool {
	return copilot.DefineTool(
		"list_organizations",
		"List all discovered source organizations",
		func(params ListOrganizationsParams, inv copilot.ToolInvocation) (any, error) {
			return c.executeListOrganizations(context.Background(), params)
		},
	)
}

// Tool implementation methods.

func (c *Client) executeFindPilotCandidates(ctx context.Context, params FindPilotParams) (any, error) {
	maxCount := params.MaxCount
	if maxCount == 0 {
		maxCount = 10
	}
	if maxCount > 50 {
		maxCount = 50
	}

	filters := map[string]any{
		"status":          StatusPending,
		"max_complexity":  5,
		"limit":           maxCount * 2,
		"include_details": true,
	}
	if params.Organization != "" {
		filters["organization"] = params.Organization
	}

	repos, err := c.db.ListRepositories(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories: %w", err)
	}

	// Score and sort candidates
	type scoredRepo struct {
		repo  *models.Repository
		score int
	}

	scored := make([]scoredRepo, 0, len(repos))
	for _, repo := range repos {
		score := 0

		deps, _ := c.db.GetRepositoryDependenciesByFullName(ctx, repo.FullName)
		localDeps := 0
		for _, dep := range deps {
			if dep.IsLocal {
				localDeps++
			}
		}
		score += localDeps * 10

		if repo.IsArchived {
			score += 5
		}
		if repo.IsFork {
			score += 5
		}

		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			score += *repo.Validation.ComplexityScore
		}

		scored = append(scored, scoredRepo{repo: repo, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score < scored[j].score
	})

	candidates := make([]map[string]any, 0, maxCount)
	for i := 0; i < len(scored) && len(candidates) < maxCount; i++ {
		repo := scored[i].repo
		complexity := 0
		rating := RatingUnknown
		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			complexity = *repo.Validation.ComplexityScore
			rating = getComplexityRating(complexity)
		}

		size := int64(0)
		if repo.GitProperties != nil && repo.GitProperties.TotalSize != nil {
			size = *repo.GitProperties.TotalSize / 1024
		}

		candidates = append(candidates, map[string]any{
			"full_name":         repo.FullName,
			"complexity_score":  complexity,
			"complexity_rating": rating,
			"size_kb":           size,
			"is_archived":       repo.IsArchived,
			"is_fork":           repo.IsFork,
		})
	}

	return map[string]any{
		"candidates": candidates,
		"count":      len(candidates),
		"summary":    fmt.Sprintf("Found %d repositories suitable for pilot migration", len(candidates)),
	}, nil
}

func (c *Client) executeAnalyzeRepositories(ctx context.Context, params AnalyzeRepositoriesParams) (any, error) {
	filters := map[string]any{
		"limit":           20,
		"include_details": true,
	}

	if params.Organization != "" {
		filters["organization"] = params.Organization
	}
	if params.Status != "" {
		filters["status"] = params.Status
	}
	if params.MaxComplexity > 0 {
		filters["max_complexity"] = params.MaxComplexity
	}

	repos, err := c.db.ListRepositories(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories: %w", err)
	}

	results := make([]map[string]any, 0, len(repos))
	for _, repo := range repos {
		complexity := 0
		rating := RatingUnknown
		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			complexity = *repo.Validation.ComplexityScore
			rating = getComplexityRating(complexity)
		}

		results = append(results, map[string]any{
			"full_name":         repo.FullName,
			"status":            repo.Status,
			"complexity_score":  complexity,
			"complexity_rating": rating,
			"is_archived":       repo.IsArchived,
			"is_fork":           repo.IsFork,
		})
	}

	status := "all"
	if params.Status != "" {
		status = params.Status
	}

	return map[string]any{
		"repositories": results,
		"count":        len(results),
		"summary":      fmt.Sprintf("Found %d repositories (filter: %s)", len(results), status),
	}, nil
}

func (c *Client) executeCreateBatch(ctx context.Context, params CreateBatchParams) (any, error) {
	// Check authorization - self-service or higher
	if err := c.checkToolAuthorization("create_batch", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	name := params.Name
	if name == "" {
		name = fmt.Sprintf("batch-%s", time.Now().Format("20060102-150405"))
	}

	if len(params.Repositories) == 0 {
		return nil, fmt.Errorf("no repositories specified for batch")
	}

	repos, err := c.db.GetRepositoriesByNames(ctx, params.Repositories)
	if err != nil {
		return nil, fmt.Errorf("failed to verify repositories: %w", err)
	}

	if len(repos) != len(params.Repositories) {
		return nil, fmt.Errorf("only %d of %d repositories found", len(repos), len(params.Repositories))
	}

	description := fmt.Sprintf("Created via Copilot with %d repositories", len(repos))
	batch := &models.Batch{
		Name:            name,
		Description:     &description,
		Type:            "custom",
		Status:          StatusPending,
		RepositoryCount: len(repos),
	}

	if params.DestinationOrg != "" {
		batch.DestinationOrg = &params.DestinationOrg
	}

	if err := c.db.CreateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	repoIDs := make([]int64, len(repos))
	for i, repo := range repos {
		repoIDs[i] = repo.ID
	}

	if err := c.db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
		return nil, fmt.Errorf("failed to add repositories to batch: %w", err)
	}

	result := map[string]any{
		"batch_id":         batch.ID,
		"batch_name":       batch.Name,
		"repository_count": batch.RepositoryCount,
		"status":           batch.Status,
		"summary":          fmt.Sprintf("Created batch '%s' with %d repositories", name, len(repos)),
	}

	if params.DestinationOrg != "" {
		result["destination_org"] = params.DestinationOrg
	}

	return result, nil
}

func (c *Client) executeConfigureBatch(ctx context.Context, params ConfigureBatchParams) (any, error) {
	// Check authorization - self-service or higher
	if err := c.checkToolAuthorization("configure_batch", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.BatchName == "" && params.BatchID == 0 {
		return nil, fmt.Errorf("batch_name or batch_id is required")
	}

	if params.DestinationOrg == "" && params.MigrationAPI == "" {
		return nil, fmt.Errorf("at least one setting must be specified (destination_org or migration_api)")
	}

	batches, err := c.db.ListBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	var batch *models.Batch
	for _, b := range batches {
		if (params.BatchName != "" && b.Name == params.BatchName) || (params.BatchID != 0 && b.ID == params.BatchID) {
			batch = b
			break
		}
	}

	if batch == nil {
		return nil, fmt.Errorf("batch not found")
	}

	changes := make([]string, 0)
	if params.DestinationOrg != "" {
		batch.DestinationOrg = &params.DestinationOrg
		changes = append(changes, fmt.Sprintf("destination organization set to '%s'", params.DestinationOrg))
	}
	if params.MigrationAPI != "" {
		migrationAPI := strings.ToUpper(params.MigrationAPI)
		if migrationAPI != models.MigrationAPIGEI && migrationAPI != models.MigrationAPIELM {
			return nil, fmt.Errorf("invalid migration_api '%s'. Must be 'GEI' or 'ELM'", params.MigrationAPI)
		}
		batch.MigrationAPI = migrationAPI
		changes = append(changes, fmt.Sprintf("migration API set to '%s'", migrationAPI))
	}

	if err := c.db.UpdateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to update batch: %w", err)
	}

	result := map[string]any{
		"batch_id":      batch.ID,
		"batch_name":    batch.Name,
		"status":        batch.Status,
		"migration_api": batch.MigrationAPI,
		"summary":       fmt.Sprintf("Batch '%s' updated: %s", batch.Name, strings.Join(changes, ", ")),
	}
	if batch.DestinationOrg != nil {
		result["destination_org"] = *batch.DestinationOrg
	}

	return result, nil
}

func (c *Client) executeCheckDependencies(ctx context.Context, params CheckDependenciesParams) (any, error) {
	if params.Repository == "" {
		return nil, fmt.Errorf("repository name is required")
	}

	deps, err := c.db.GetRepositoryDependenciesByFullName(ctx, params.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	dependencies := make([]map[string]any, 0, len(deps))
	for _, dep := range deps {
		info := map[string]any{
			"dependency":  dep.DependencyFullName,
			"type":        dep.DependencyType,
			"is_local":    dep.IsLocal,
			"is_migrated": false,
		}

		if dep.IsLocal {
			depRepo, err := c.db.GetRepository(ctx, dep.DependencyFullName)
			if err == nil && depRepo != nil {
				info["status"] = depRepo.Status
				info["is_migrated"] = depRepo.Status == StatusCompleted || depRepo.Status == StatusMigrationComplete
			}
		}

		dependencies = append(dependencies, info)
	}

	result := map[string]any{
		"repository":   params.Repository,
		"dependencies": dependencies,
		"count":        len(dependencies),
		"summary":      fmt.Sprintf("Found %d dependencies for %s", len(dependencies), params.Repository),
	}

	if params.IncludeReverse {
		reverseDeps, err := c.db.GetDependentRepositories(ctx, params.Repository)
		if err == nil {
			reverse := make([]map[string]any, 0, len(reverseDeps))
			for _, repo := range reverseDeps {
				reverse = append(reverse, map[string]any{
					"repository":  repo.FullName,
					"status":      repo.Status,
					"is_migrated": repo.Status == StatusCompleted || repo.Status == StatusMigrationComplete,
				})
			}
			result["reverse_dependencies"] = reverse
		}
	}

	return result, nil
}

func (c *Client) executePlanWaves(ctx context.Context, params PlanWavesParams) (any, error) {
	// Check authorization - self-service or higher
	if err := c.checkToolAuthorization("plan_waves", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	waveSize := params.WaveSize
	if waveSize == 0 {
		waveSize = 10
	}
	if waveSize > 100 {
		waveSize = 100
	}

	filters := map[string]any{
		"status":          StatusPending,
		"include_details": true,
	}
	if params.Organization != "" {
		filters["organization"] = params.Organization
	}

	repos, err := c.db.ListRepositories(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	if len(repos) == 0 {
		return map[string]any{
			"waves":   []any{},
			"summary": "No pending repositories found",
		}, nil
	}

	// Build dependency graph
	depGraph := make(map[string][]string)
	for _, repo := range repos {
		deps, _ := c.db.GetRepositoryDependenciesByFullName(ctx, repo.FullName)
		for _, dep := range deps {
			if dep.IsLocal {
				depGraph[repo.FullName] = append(depGraph[repo.FullName], dep.DependencyFullName)
			}
		}
	}

	// Create waves using topological sort
	waves := make([]map[string]any, 0)
	migrated := make(map[string]bool)
	repoMap := make(map[string]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.FullName] = repo
	}

	waveNum := 1
	remaining := len(repos)
	for remaining > 0 && waveNum <= 100 {
		waveRepos := make([]string, 0)

		for _, repo := range repos {
			if migrated[repo.FullName] {
				continue
			}

			allDepsMigrated := true
			for _, dep := range depGraph[repo.FullName] {
				if !migrated[dep] {
					if _, inPending := repoMap[dep]; inPending {
						allDepsMigrated = false
						break
					}
				}
			}

			if allDepsMigrated && len(waveRepos) < waveSize {
				waveRepos = append(waveRepos, repo.FullName)
				migrated[repo.FullName] = true
				remaining--
			}
		}

		// Handle circular dependencies
		if len(waveRepos) == 0 && remaining > 0 {
			for _, repo := range repos {
				if !migrated[repo.FullName] && len(waveRepos) < waveSize {
					waveRepos = append(waveRepos, repo.FullName)
					migrated[repo.FullName] = true
					remaining--
				}
			}
		}

		if len(waveRepos) > 0 {
			waves = append(waves, map[string]any{
				"wave_number":  waveNum,
				"repositories": waveRepos,
				"count":        len(waveRepos),
			})
			waveNum++
		}
	}

	return map[string]any{
		"waves":   waves,
		"summary": fmt.Sprintf("Planned %d waves for %d repositories", len(waves), len(repos)),
	}, nil
}

func (c *Client) executeGetComplexityBreakdown(ctx context.Context, params GetComplexityParams) (any, error) {
	if params.Repository == "" {
		return nil, fmt.Errorf("repository name is required")
	}

	repo, err := c.db.GetRepository(ctx, params.Repository)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %s", params.Repository)
	}

	breakdown := map[string]any{
		"repository":  params.Repository,
		"total_score": 0,
		"rating":      RatingUnknown,
		"components":  map[string]int{},
		"blockers":    []string{},
		"warnings":    []string{},
	}

	if repo.Validation != nil {
		if repo.Validation.ComplexityScore != nil {
			breakdown["total_score"] = *repo.Validation.ComplexityScore
			breakdown["rating"] = getComplexityRating(*repo.Validation.ComplexityScore)
		}

		if repo.Validation.ComplexityBreakdown != nil {
			var components map[string]int
			if err := json.Unmarshal([]byte(*repo.Validation.ComplexityBreakdown), &components); err == nil {
				breakdown["components"] = components
			}
		}

		blockers := []string{}
		warnings := []string{}

		if repo.Validation.HasBlockingFiles {
			blockers = append(blockers, "Has blocking files")
		}
		if repo.Validation.HasOversizedCommits {
			blockers = append(blockers, "Has oversized commits")
		}
		if repo.Validation.HasOversizedRepository {
			blockers = append(blockers, "Repository is oversized")
		}
		if repo.Validation.HasLongRefs {
			warnings = append(warnings, "Has long references")
		}
		if repo.Validation.HasLargeFileWarnings {
			warnings = append(warnings, "Has large file warnings")
		}

		breakdown["blockers"] = blockers
		breakdown["warnings"] = warnings
	}

	return breakdown, nil
}

func (c *Client) executeGetTeamRepositories(ctx context.Context, params GetTeamRepositoriesParams) (any, error) {
	if params.Team == "" {
		return nil, fmt.Errorf("team name is required (format: org/team-slug)")
	}

	parts := strings.SplitN(params.Team, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("team must be in format org/team-slug")
	}

	teamDetail, err := c.db.GetTeamDetail(ctx, parts[0], parts[1])
	if err != nil {
		return nil, fmt.Errorf("team not found: %s", params.Team)
	}

	repos := make([]map[string]any, 0)
	for _, tr := range teamDetail.Repositories {
		status := StatusPending
		if tr.MigrationStatus != nil {
			status = *tr.MigrationStatus
		}
		repos = append(repos, map[string]any{
			"full_name": tr.FullName,
			"status":    status,
		})
	}

	return map[string]any{
		"team":         params.Team,
		"repositories": repos,
		"count":        len(repos),
		"summary":      fmt.Sprintf("Found %d repositories for team %s", len(repos), params.Team),
	}, nil
}

func (c *Client) executeGetMigrationStatus(ctx context.Context, params GetMigrationStatusParams) (any, error) {
	if len(params.Repositories) == 0 {
		return nil, fmt.Errorf("at least one repository is required")
	}

	repos, err := c.db.GetRepositoriesByNames(ctx, params.Repositories)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	statuses := make([]map[string]any, 0, len(repos))
	for _, repo := range repos {
		statuses = append(statuses, map[string]any{
			"full_name": repo.FullName,
			"status":    repo.Status,
		})
	}

	return map[string]any{
		"statuses": statuses,
		"count":    len(statuses),
		"summary":  fmt.Sprintf("Found status for %d of %d repositories", len(statuses), len(params.Repositories)),
	}, nil
}

func (c *Client) executeScheduleBatch(ctx context.Context, params ScheduleBatchParams) (any, error) {
	// Check authorization - self-service or higher
	if err := c.checkToolAuthorization("schedule_batch", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.BatchName == "" {
		return nil, fmt.Errorf("batch_name is required")
	}

	var scheduledAt *time.Time
	if params.ScheduledAt != "" {
		parsed, err := time.Parse(time.RFC3339, params.ScheduledAt)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", params.ScheduledAt)
			if err != nil {
				return nil, fmt.Errorf("invalid datetime format. Use ISO 8601 (e.g., 2024-01-15T09:00:00Z)")
			}
		}
		scheduledAt = &parsed
	}

	batches, err := c.db.ListBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	var batch *models.Batch
	for _, b := range batches {
		if b.Name == params.BatchName {
			batch = b
			break
		}
	}

	if batch == nil {
		return nil, fmt.Errorf("batch not found: %s", params.BatchName)
	}

	if params.DestinationOrg != "" {
		batch.DestinationOrg = &params.DestinationOrg
	}

	if scheduledAt != nil {
		batch.ScheduledAt = scheduledAt
		batch.Status = StatusScheduled
	} else {
		now := time.Now()
		batch.ScheduledAt = &now
		batch.Status = StatusScheduled
		scheduledAt = &now
	}

	if err := c.db.UpdateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to schedule batch: %w", err)
	}

	result := map[string]any{
		"batch_id":     batch.ID,
		"batch_name":   batch.Name,
		"status":       batch.Status,
		"scheduled_at": scheduledAt.Format(time.RFC3339),
		"summary":      fmt.Sprintf("Batch '%s' scheduled for %s", params.BatchName, scheduledAt.Format("2006-01-02 15:04:05")),
	}

	if params.DestinationOrg != "" {
		result["destination_org"] = params.DestinationOrg
	}

	return result, nil
}

func (c *Client) executeStartMigration(ctx context.Context, params StartMigrationParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("start_migration", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.BatchName == "" && params.BatchID == 0 && params.Repository == "" {
		return nil, fmt.Errorf("at least one of batch_name, batch_id, or repository must be specified")
	}

	// Default to dry-run for safety
	dryRun := true
	// The params.DryRun field defaults to false in Go, so we need special handling
	// For now, we'll treat the explicit false as intentional production migration

	targetStatus := models.StatusQueuedForMigration
	if dryRun {
		targetStatus = models.StatusDryRunQueued
	}

	var queuedRepos []map[string]any
	var batch *models.Batch
	skippedCount := 0

	// Handle batch migration
	if params.BatchName != "" || params.BatchID != 0 {
		batches, err := c.db.ListBatches(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list batches: %w", err)
		}

		for _, b := range batches {
			if (params.BatchName != "" && b.Name == params.BatchName) || (params.BatchID != 0 && b.ID == params.BatchID) {
				batch = b
				break
			}
		}

		if batch == nil {
			return nil, fmt.Errorf("batch not found")
		}

		if batch.Status == models.BatchStatusInProgress {
			return nil, fmt.Errorf("batch '%s' is already running", batch.Name)
		}

		repos, err := c.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get batch repositories: %w", err)
		}

		if len(repos) == 0 {
			return nil, fmt.Errorf("batch '%s' has no repositories", batch.Name)
		}

		batch.Status = models.BatchStatusInProgress
		now := time.Now()
		if dryRun {
			batch.DryRunStartedAt = &now
			batch.LastDryRunAt = &now
		} else {
			batch.StartedAt = &now
			batch.LastMigrationAttemptAt = &now
		}
		if err := c.db.UpdateBatch(ctx, batch); err != nil {
			return nil, fmt.Errorf("failed to update batch status: %w", err)
		}

		priority := 0
		if batch.Type == models.BatchTypePilot {
			priority = 1
		}

		for _, repo := range repos {
			if canQueueForMigration(repo.Status, dryRun) {
				repo.Status = string(targetStatus)
				repo.Priority = priority
				if err := c.db.UpdateRepository(ctx, repo); err != nil {
					c.logger.Error("Failed to queue repository", "repo", repo.FullName, "error", err)
					continue
				}
				queuedRepos = append(queuedRepos, map[string]any{
					"full_name": repo.FullName,
					"status":    repo.Status,
				})
			} else {
				skippedCount++
			}
		}
	}

	// Handle single repository
	if params.Repository != "" {
		repo, err := c.db.GetRepository(ctx, params.Repository)
		if err != nil || repo == nil {
			return nil, fmt.Errorf("repository not found: %s", params.Repository)
		}

		if !canQueueForMigration(repo.Status, dryRun) {
			return nil, fmt.Errorf("repository '%s' cannot be queued for migration (status: %s)", params.Repository, repo.Status)
		}

		repo.Status = string(targetStatus)
		if err := c.db.UpdateRepository(ctx, repo); err != nil {
			return nil, fmt.Errorf("failed to queue repository: %w", err)
		}
		queuedRepos = append(queuedRepos, map[string]any{
			"full_name": repo.FullName,
			"status":    repo.Status,
		})
	}

	if len(queuedRepos) == 0 {
		return nil, fmt.Errorf("no repositories could be queued for migration")
	}

	migrationType := "production migration"
	if dryRun {
		migrationType = "dry-run"
	}

	result := map[string]any{
		"queued_count":  len(queuedRepos),
		"skipped_count": skippedCount,
		"dry_run":       dryRun,
		"repositories":  queuedRepos,
	}
	if batch != nil {
		result["batch_id"] = batch.ID
		result["batch_name"] = batch.Name
		result["summary"] = fmt.Sprintf("Started %s for batch '%s' (%d repositories)", migrationType, batch.Name, len(queuedRepos))
	} else {
		result["summary"] = fmt.Sprintf("Started %s for %d repositories", migrationType, len(queuedRepos))
	}

	return result, nil
}

func (c *Client) executeCancelMigration(ctx context.Context, params CancelMigrationParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("cancel_migration", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.BatchName == "" && params.BatchID == 0 && params.Repository == "" {
		return nil, fmt.Errorf("at least one of batch_name, batch_id, or repository must be specified")
	}

	cancelledCount := 0

	// Handle batch cancellation
	if params.BatchName != "" || params.BatchID != 0 {
		batches, err := c.db.ListBatches(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list batches: %w", err)
		}

		var batch *models.Batch
		for _, b := range batches {
			if (params.BatchName != "" && b.Name == params.BatchName) || (params.BatchID != 0 && b.ID == params.BatchID) {
				batch = b
				break
			}
		}

		if batch == nil {
			return nil, fmt.Errorf("batch not found")
		}

		repos, err := c.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get batch repositories: %w", err)
		}

		for _, repo := range repos {
			if isInQueuedOrInProgressState(repo.Status) {
				repo.Status = StatusPending
				if err := c.db.UpdateRepository(ctx, repo); err != nil {
					c.logger.Error("Failed to cancel repository", "repo", repo.FullName, "error", err)
					continue
				}
				cancelledCount++
			}
		}

		batch.Status = models.BatchStatusCancelled
		if err := c.db.UpdateBatch(ctx, batch); err != nil {
			c.logger.Error("Failed to update batch status", "batch", batch.Name, "error", err)
		}

		return map[string]any{
			"batch_id":        batch.ID,
			"batch_name":      batch.Name,
			"cancelled_count": cancelledCount,
			"summary":         fmt.Sprintf("Cancelled batch '%s' (%d repositories)", batch.Name, cancelledCount),
		}, nil
	}

	// Handle single repository cancellation
	if params.Repository != "" {
		repo, err := c.db.GetRepository(ctx, params.Repository)
		if err != nil || repo == nil {
			return nil, fmt.Errorf("repository not found: %s", params.Repository)
		}

		if !isInQueuedOrInProgressState(repo.Status) {
			return nil, fmt.Errorf("repository '%s' is not in a cancellable state (status: %s)", params.Repository, repo.Status)
		}

		repo.Status = StatusPending
		if err := c.db.UpdateRepository(ctx, repo); err != nil {
			return nil, fmt.Errorf("failed to cancel repository: %w", err)
		}

		return map[string]any{
			"repository":      params.Repository,
			"cancelled_count": 1,
			"summary":         fmt.Sprintf("Cancelled migration for repository '%s'", params.Repository),
		}, nil
	}

	return nil, fmt.Errorf("no target specified for cancellation")
}

func (c *Client) executeGetMigrationProgress(ctx context.Context, params GetMigrationProgressParams) (any, error) {
	// Handle single repository progress
	if params.Repository != "" {
		repo, err := c.db.GetRepository(ctx, params.Repository)
		if err != nil || repo == nil {
			return nil, fmt.Errorf("repository not found: %s", params.Repository)
		}

		progress := calculateProgress([]string{repo.Status})

		return map[string]any{
			"repository": params.Repository,
			"status":     repo.Status,
			"progress":   progress,
			"summary":    fmt.Sprintf("Repository '%s' status: %s", params.Repository, repo.Status),
		}, nil
	}

	// Handle batch progress
	if params.BatchName != "" || params.BatchID != 0 {
		batches, err := c.db.ListBatches(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list batches: %w", err)
		}

		var batch *models.Batch
		for _, b := range batches {
			if (params.BatchName != "" && b.Name == params.BatchName) || (params.BatchID != 0 && b.ID == params.BatchID) {
				batch = b
				break
			}
		}

		if batch == nil {
			searchTerm := params.BatchName
			if params.BatchID != 0 {
				searchTerm = strconv.FormatInt(params.BatchID, 10)
			}
			return nil, fmt.Errorf("batch not found: %s", searchTerm)
		}

		repos, err := c.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get batch repositories: %w", err)
		}

		statuses := make([]string, len(repos))
		repoDetails := make([]map[string]any, len(repos))
		for i, repo := range repos {
			statuses[i] = repo.Status
			repoDetails[i] = map[string]any{
				"full_name": repo.FullName,
				"status":    repo.Status,
			}
		}

		progress := calculateProgress(statuses)

		return map[string]any{
			"batch_id":     batch.ID,
			"batch_name":   batch.Name,
			"batch_status": batch.Status,
			"progress":     progress,
			"repositories": repoDetails,
			"summary": fmt.Sprintf("Batch '%s': %d/%d complete (%.1f%%)",
				batch.Name, progress["completed_count"], progress["total_count"], progress["percent_complete"]),
		}, nil
	}

	return nil, fmt.Errorf("at least one of batch_name, batch_id, or repository must be specified")
}

// Helper function for complexity rating.
func getComplexityRating(score int) string {
	switch {
	case score <= 5:
		return "simple"
	case score <= 10:
		return "medium"
	case score <= 17:
		return "complex"
	default:
		return "very_complex"
	}
}

// New tool implementations for discovery and user management.

func (c *Client) executeGetTopComplexRepositories(ctx context.Context, params GetTopComplexRepositoriesParams) (any, error) {
	count := params.Count
	if count == 0 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	filters := map[string]any{
		"include_details": true,
		"limit":           count * 2, // Get more to filter those without scores
	}
	if params.Organization != "" {
		filters["organization"] = params.Organization
	}

	repos, err := c.db.ListRepositories(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// Filter and sort by complexity score (descending)
	type scoredRepo struct {
		repo  *models.Repository
		score int
	}
	scored := make([]scoredRepo, 0)
	for _, repo := range repos {
		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			scored = append(scored, scoredRepo{repo: repo, score: *repo.Validation.ComplexityScore})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score // Descending
	})

	results := make([]map[string]any, 0, count)
	for i := 0; i < len(scored) && len(results) < count; i++ {
		repo := scored[i].repo
		blockers := []string{}
		if repo.Validation != nil {
			if repo.Validation.HasBlockingFiles {
				blockers = append(blockers, "blocking_files")
			}
			if repo.Validation.HasOversizedRepository {
				blockers = append(blockers, "oversized")
			}
		}
		results = append(results, map[string]any{
			"full_name":         repo.FullName,
			"complexity_score":  scored[i].score,
			"complexity_rating": getComplexityRating(scored[i].score),
			"status":            repo.Status,
			"blockers":          blockers,
		})
	}

	return map[string]any{
		"repositories": results,
		"count":        len(results),
		"summary":      fmt.Sprintf("Found %d most complex repositories", len(results)),
	}, nil
}

func (c *Client) executeGetRepositoriesWithMostDependencies(ctx context.Context, params GetRepositoriesWithMostDependenciesParams) (any, error) {
	count := params.Count
	if count == 0 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	filters := map[string]any{
		"include_details": true,
	}
	if params.Organization != "" {
		filters["organization"] = params.Organization
	}

	repos, err := c.db.ListRepositories(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// Count dependencies for each repository
	type repoWithDeps struct {
		repo     *models.Repository
		depCount int
		deps     []string
	}
	reposWithDeps := make([]repoWithDeps, 0, len(repos))

	for _, repo := range repos {
		deps, _ := c.db.GetRepositoryDependenciesByFullName(ctx, repo.FullName)
		localDeps := []string{}
		for _, dep := range deps {
			if dep.IsLocal {
				localDeps = append(localDeps, dep.DependencyFullName)
			}
		}
		reposWithDeps = append(reposWithDeps, repoWithDeps{
			repo:     repo,
			depCount: len(localDeps),
			deps:     localDeps,
		})
	}

	// Sort by dependency count (descending)
	sort.Slice(reposWithDeps, func(i, j int) bool {
		return reposWithDeps[i].depCount > reposWithDeps[j].depCount
	})

	results := make([]map[string]any, 0, count)
	for i := 0; i < len(reposWithDeps) && len(results) < count; i++ {
		r := reposWithDeps[i]
		if r.depCount == 0 {
			continue // Skip repos with no dependencies
		}
		results = append(results, map[string]any{
			"full_name":        r.repo.FullName,
			"dependency_count": r.depCount,
			"dependencies":     r.deps,
			"status":           r.repo.Status,
		})
	}

	return map[string]any{
		"repositories": results,
		"count":        len(results),
		"summary":      fmt.Sprintf("Found %d repositories with the most dependencies", len(results)),
	}, nil
}

func (c *Client) executeListTeams(ctx context.Context, params ListTeamsParams) (any, error) {
	maxCount := params.MaxCount
	if maxCount == 0 {
		maxCount = 20
	}
	if maxCount > 100 {
		maxCount = 100
	}

	teams, err := c.db.ListTeams(ctx, params.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	results := make([]map[string]any, 0, maxCount)
	for i, team := range teams {
		if i >= maxCount {
			break
		}
		memberCount, _ := c.db.GetTeamMemberCount(ctx, team.ID)
		results = append(results, map[string]any{
			"team":         fmt.Sprintf("%s/%s", team.Organization, team.Slug),
			"name":         team.Name,
			"member_count": memberCount,
			"privacy":      team.Privacy,
		})
	}

	return map[string]any{
		"teams":   results,
		"count":   len(results),
		"summary": fmt.Sprintf("Found %d teams", len(results)),
	}, nil
}

func (c *Client) executeGetTeamMigrationStats(ctx context.Context, params GetTeamMigrationStatsParams) (any, error) {
	if params.Team == "" {
		return nil, fmt.Errorf("team is required (format: org/team-slug)")
	}

	parts := strings.SplitN(params.Team, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("team must be in format org/team-slug")
	}

	teamDetail, err := c.db.GetTeamDetail(ctx, parts[0], parts[1])
	if err != nil {
		return nil, fmt.Errorf("team not found: %s", params.Team)
	}

	// Calculate migration stats
	totalRepos := len(teamDetail.Repositories)
	migratedCount := 0
	pendingCount := 0
	failedCount := 0

	for _, tr := range teamDetail.Repositories {
		if tr.MigrationStatus != nil {
			switch models.MigrationStatus(*tr.MigrationStatus) {
			case models.StatusMigrationComplete, models.StatusComplete:
				migratedCount++
			case models.StatusMigrationFailed, models.StatusDryRunFailed:
				failedCount++
			default:
				pendingCount++
			}
		} else {
			pendingCount++
		}
	}

	percentMigrated := 0.0
	if totalRepos > 0 {
		percentMigrated = float64(migratedCount) / float64(totalRepos) * 100
	}

	return map[string]any{
		"team":             params.Team,
		"team_name":        teamDetail.Name,
		"total_repos":      totalRepos,
		"migrated_count":   migratedCount,
		"pending_count":    pendingCount,
		"failed_count":     failedCount,
		"percent_migrated": percentMigrated,
		"member_count":     len(teamDetail.Members),
		"summary": fmt.Sprintf("Team '%s': %d/%d repos migrated (%.1f%%)",
			teamDetail.Name, migratedCount, totalRepos, percentMigrated),
	}, nil
}

func (c *Client) executeMigrateTeam(ctx context.Context, params MigrateTeamParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("migrate_team", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.Team == "" {
		return nil, fmt.Errorf("team is required (format: org/team-slug)")
	}

	parts := strings.SplitN(params.Team, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("team must be in format org/team-slug")
	}

	// Get the team mapping
	mapping, err := c.db.GetTeamMapping(ctx, parts[0], parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to get team mapping: %w", err)
	}

	if mapping == nil {
		// Create a new mapping
		mapping = &models.TeamMapping{
			SourceOrg:      parts[0],
			SourceTeamSlug: parts[1],
		}
	}

	// Update destination if provided
	if params.DestinationOrg != "" {
		mapping.DestinationOrg = &params.DestinationOrg
	}
	if params.DestinationTeam != "" {
		mapping.DestinationTeamSlug = &params.DestinationTeam
	}

	// Set status to pending migration
	mapping.MigrationStatus = models.TeamMigrationPending

	if err := c.db.SaveTeamMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to save team mapping: %w", err)
	}

	return map[string]any{
		"team":             params.Team,
		"destination_org":  mapping.DestinationOrg,
		"destination_team": mapping.DestinationTeamSlug,
		"status":           mapping.MigrationStatus,
		"summary":          fmt.Sprintf("Team '%s' queued for migration", params.Team),
	}, nil
}

func (c *Client) executeListMannequins(ctx context.Context, params ListMannequinsParams) (any, error) {
	if params.Organization == "" {
		return nil, fmt.Errorf("organization is required")
	}

	maxCount := params.MaxCount
	if maxCount == 0 {
		maxCount = 20
	}
	if maxCount > 100 {
		maxCount = 100
	}

	filters := storage.UserMannequinFilters{
		MannequinOrg: params.Organization,
		Limit:        maxCount,
	}
	if params.ReclaimStatus != "" {
		filters.ReclaimStatus = params.ReclaimStatus
	}

	mannequins, total, err := c.db.ListUserMannequins(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list mannequins: %w", err)
	}

	// Get stats
	stats, _ := c.db.GetMannequinOrgStats(ctx, params.Organization)

	results := make([]map[string]any, 0, len(mannequins))
	for _, m := range mannequins {
		status := "unknown"
		if m.ReclaimStatus != nil && *m.ReclaimStatus != "" {
			status = *m.ReclaimStatus
		}
		results = append(results, map[string]any{
			"source_login":   m.SourceLogin,
			"mannequin_id":   m.MannequinID,
			"reclaim_status": status,
		})
	}

	response := map[string]any{
		"organization": params.Organization,
		"mannequins":   results,
		"count":        len(results),
		"total":        total,
	}

	if stats != nil {
		response["stats"] = map[string]any{
			"total":     stats.Total,
			"invitable": stats.Invitable,
			"pending":   stats.Pending,
			"completed": stats.Completed,
		}
		response["summary"] = fmt.Sprintf("Found %d mannequins in %s (%d invitable, %d pending, %d completed)",
			stats.Total, params.Organization, stats.Invitable, stats.Pending, stats.Completed)
	} else {
		response["summary"] = fmt.Sprintf("Found %d mannequins in %s", len(results), params.Organization)
	}

	return response, nil
}

func (c *Client) executeSendMannequinInvitations(ctx context.Context, params SendMannequinInvitationsParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("send_mannequin_invitations", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.Organization == "" {
		return nil, fmt.Errorf("organization is required")
	}

	// Get mannequins that are eligible for invitation
	filters := storage.UserMannequinFilters{
		MannequinOrg: params.Organization,
		Limit:        100,
	}

	mannequins, _, err := c.db.ListUserMannequins(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list mannequins: %w", err)
	}

	// Filter to specific logins if provided
	targetLogins := make(map[string]bool)
	if len(params.SourceLogins) > 0 {
		for _, login := range params.SourceLogins {
			targetLogins[login] = true
		}
	}

	// Find eligible mannequins
	eligible := make([]map[string]any, 0)
	for _, m := range mannequins {
		// Skip if specific logins provided and this isn't one of them
		if len(targetLogins) > 0 && !targetLogins[m.SourceLogin] {
			continue
		}

		// Skip already invited or completed
		if m.ReclaimStatus != nil && (*m.ReclaimStatus == string(models.ReclaimStatusInvited) ||
			*m.ReclaimStatus == string(models.ReclaimStatusCompleted)) {
			continue
		}

		currentStatus := "none"
		if m.ReclaimStatus != nil {
			currentStatus = *m.ReclaimStatus
		}
		eligible = append(eligible, map[string]any{
			"source_login":   m.SourceLogin,
			"mannequin_id":   m.MannequinID,
			"current_status": currentStatus,
		})
	}

	if params.DryRun {
		return map[string]any{
			"dry_run":        true,
			"organization":   params.Organization,
			"eligible_count": len(eligible),
			"mannequins":     eligible,
			"summary":        fmt.Sprintf("Dry run: %d mannequins eligible for invitation in %s", len(eligible), params.Organization),
		}, nil
	}

	// Update status to pending for eligible mannequins
	sentCount := 0
	for _, m := range eligible {
		login, ok := m["source_login"].(string)
		if !ok {
			continue
		}
		if err := c.db.UpdateMannequinReclaimStatus(ctx, login, params.Organization, string(models.ReclaimStatusPending), nil); err != nil {
			c.logger.Error("Failed to update mannequin status", "login", login, "error", err)
			continue
		}
		sentCount++
	}

	return map[string]any{
		"organization": params.Organization,
		"sent_count":   sentCount,
		"mannequins":   eligible,
		"summary":      fmt.Sprintf("Queued %d mannequin reclaim invitations for %s", sentCount, params.Organization),
	}, nil
}

// --- Group 1: Discovery Operations Implementations ---

func (c *Client) executeStartDiscovery(ctx context.Context, params StartDiscoveryParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("start_discovery", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	// Validate input
	if params.Organization == "" && params.EnterpriseSlug == "" {
		return nil, fmt.Errorf("either organization or enterprise_slug is required")
	}
	if params.Organization != "" && params.EnterpriseSlug != "" {
		return nil, fmt.Errorf("cannot specify both organization and enterprise_slug")
	}

	// Check if discovery is already in progress
	activeProgress, err := c.db.GetActiveDiscoveryProgress()
	if err != nil {
		return nil, fmt.Errorf("failed to check discovery status: %w", err)
	}
	if activeProgress != nil {
		return map[string]any{
			"error":   true,
			"message": fmt.Sprintf("Discovery already in progress for %s (started: %s)", activeProgress.Target, activeProgress.StartedAt.Format(time.RFC3339)),
			"status":  "in_progress",
		}, nil
	}

	// Determine discovery type and target
	var discoveryType, target string
	if params.EnterpriseSlug != "" {
		discoveryType = models.DiscoveryTypeEnterprise
		target = params.EnterpriseSlug
	} else {
		discoveryType = models.DiscoveryTypeOrganization
		target = params.Organization
	}

	// Create progress record
	progress := &models.DiscoveryProgress{
		DiscoveryType: discoveryType,
		Target:        target,
		TotalOrgs:     1,
	}

	if err := c.db.CreateDiscoveryProgress(progress); err != nil {
		return nil, fmt.Errorf("failed to start discovery: %w", err)
	}

	return map[string]any{
		"started":        true,
		"discovery_type": discoveryType,
		"target":         target,
		"summary":        fmt.Sprintf("Started %s discovery for %s", discoveryType, target),
	}, nil
}

func (c *Client) executeGetDiscoveryStatus(ctx context.Context, _ GetDiscoveryStatusParams) (any, error) {
	progress, err := c.db.GetActiveDiscoveryProgress()
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery status: %w", err)
	}

	if progress == nil {
		// Check for last completed discovery
		lastProgress, err := c.db.GetLatestDiscoveryProgress()
		if err != nil || lastProgress == nil {
			return map[string]any{
				"status":  "idle",
				"summary": "No discovery in progress or recent history",
			}, nil
		}

		return map[string]any{
			"status":          lastProgress.Status,
			"discovery_type":  lastProgress.DiscoveryType,
			"target":          lastProgress.Target,
			"started_at":      lastProgress.StartedAt.Format(time.RFC3339),
			"completed_at":    lastProgress.CompletedAt,
			"total_repos":     lastProgress.TotalRepos,
			"processed_repos": lastProgress.ProcessedRepos,
			"last_error":      lastProgress.LastError,
			"summary":         fmt.Sprintf("Last discovery: %s for %s (%s)", lastProgress.Status, lastProgress.Target, lastProgress.StartedAt.Format("2006-01-02")),
		}, nil
	}

	percentComplete := 0.0
	if progress.TotalRepos > 0 {
		percentComplete = float64(progress.ProcessedRepos) / float64(progress.TotalRepos) * 100
	}

	return map[string]any{
		"status":           progress.Status,
		"discovery_type":   progress.DiscoveryType,
		"target":           progress.Target,
		"started_at":       progress.StartedAt.Format(time.RFC3339),
		"total_orgs":       progress.TotalOrgs,
		"processed_orgs":   progress.ProcessedOrgs,
		"total_repos":      progress.TotalRepos,
		"processed_repos":  progress.ProcessedRepos,
		"percent_complete": percentComplete,
		"current_org":      progress.CurrentOrg,
		"summary": fmt.Sprintf("Discovery in progress: %s - %d/%d repos (%.1f%%)",
			progress.Target, progress.ProcessedRepos, progress.TotalRepos, percentComplete),
	}, nil
}

func (c *Client) executeCancelDiscovery(ctx context.Context, _ CancelDiscoveryParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("cancel_discovery", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	progress, err := c.db.GetActiveDiscoveryProgress()
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery status: %w", err)
	}

	if progress == nil {
		return map[string]any{
			"cancelled": false,
			"message":   "No discovery in progress",
		}, nil
	}

	// Update status to cancelled
	progress.Status = models.DiscoveryStatusCancelled
	now := time.Now()
	progress.CompletedAt = &now
	if err := c.db.UpdateDiscoveryProgress(progress); err != nil {
		return nil, fmt.Errorf("failed to cancel discovery: %w", err)
	}

	return map[string]any{
		"cancelled": true,
		"target":    progress.Target,
		"summary":   fmt.Sprintf("Cancelled discovery for %s", progress.Target),
	}, nil
}

func (c *Client) executeDiscoverTeams(ctx context.Context, params DiscoverTeamsParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("discover_teams", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.Organization == "" {
		return nil, fmt.Errorf("organization is required")
	}

	// Get existing teams for the organization
	teams, err := c.db.ListTeams(ctx, params.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	return map[string]any{
		"organization": params.Organization,
		"teams_found":  len(teams),
		"summary":      fmt.Sprintf("Found %d teams in %s. Use start_discovery for full team discovery.", len(teams), params.Organization),
	}, nil
}

// --- Group 2: Repository Operations Implementations ---

func (c *Client) executeGetRepositoryDetails(ctx context.Context, params GetRepositoryDetailsParams) (any, error) {
	if params.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	repo, err := c.db.GetRepository(ctx, params.Repository)
	if err != nil || repo == nil {
		return nil, fmt.Errorf("repository not found: %s", params.Repository)
	}

	// Extract organization from full name
	parts := strings.SplitN(repo.FullName, "/", 2)
	orgName := ""
	repoName := repo.FullName
	if len(parts) == 2 {
		orgName = parts[0]
		repoName = parts[1]
	}

	result := map[string]any{
		"full_name":     repo.FullName,
		"organization":  orgName,
		"name":          repoName,
		"status":        repo.Status,
		"discovered_at": repo.DiscoveredAt.Format(time.RFC3339),
		"is_archived":   repo.IsArchived,
		"is_fork":       repo.IsFork,
	}

	// Add git properties if available
	if repo.GitProperties != nil {
		if repo.GitProperties.TotalSize != nil {
			result["size_bytes"] = *repo.GitProperties.TotalSize
		}
		if repo.GitProperties.DefaultBranch != nil {
			result["default_branch"] = *repo.GitProperties.DefaultBranch
		}
	}

	// Add validation details
	if repo.Validation != nil {
		validation := map[string]any{}
		if repo.Validation.ValidationStatus != nil {
			validation["status"] = *repo.Validation.ValidationStatus
		}
		if repo.Validation.ComplexityScore != nil {
			validation["complexity_score"] = *repo.Validation.ComplexityScore
			validation["complexity_rating"] = getComplexityRating(*repo.Validation.ComplexityScore)
		}
		validation["has_blocking_files"] = repo.Validation.HasBlockingFiles
		validation["has_oversized_repository"] = repo.Validation.HasOversizedRepository
		result["validation"] = validation
	}

	// Get dependencies
	deps, _ := c.db.GetRepositoryDependenciesByFullName(ctx, params.Repository)
	if len(deps) > 0 {
		localDeps := []string{}
		for _, dep := range deps {
			if dep.IsLocal {
				localDeps = append(localDeps, dep.DependencyFullName)
			}
		}
		result["local_dependencies"] = localDeps
		result["dependency_count"] = len(localDeps)
	}

	result["summary"] = fmt.Sprintf("Repository %s: status=%s", repo.FullName, repo.Status)
	return result, nil
}

func (c *Client) executeValidateRepository(ctx context.Context, params ValidateRepositoryParams) (any, error) {
	if params.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	repo, err := c.db.GetRepository(ctx, params.Repository)
	if err != nil || repo == nil {
		return nil, fmt.Errorf("repository not found: %s", params.Repository)
	}

	blockers := []string{}
	warnings := []string{}

	if repo.Validation != nil {
		if repo.Validation.HasBlockingFiles {
			blockers = append(blockers, "Contains blocking files (e.g., large binaries)")
		}
		if repo.Validation.HasOversizedRepository {
			blockers = append(blockers, "Repository exceeds size limits")
		}
	}

	if repo.IsArchived {
		warnings = append(warnings, "Repository is archived")
	}
	if repo.IsFork {
		warnings = append(warnings, "Repository is a fork")
	}

	canMigrate := len(blockers) == 0

	return map[string]any{
		"repository":  params.Repository,
		"can_migrate": canMigrate,
		"blockers":    blockers,
		"warnings":    warnings,
		"summary": func() string {
			if canMigrate {
				return fmt.Sprintf("Repository %s is ready for migration", params.Repository)
			}
			return fmt.Sprintf("Repository %s has %d blockers preventing migration", params.Repository, len(blockers))
		}(),
	}, nil
}

func (c *Client) executeUpdateRepositoryStatus(ctx context.Context, params UpdateRepositoryStatusParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("update_repository_status", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}
	if params.Status == "" {
		return nil, fmt.Errorf("status is required")
	}

	repo, err := c.db.GetRepository(ctx, params.Repository)
	if err != nil || repo == nil {
		return nil, fmt.Errorf("repository not found: %s", params.Repository)
	}

	oldStatus := repo.Status
	repo.Status = params.Status

	if err := c.db.UpdateRepository(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	return map[string]any{
		"repository": params.Repository,
		"old_status": oldStatus,
		"new_status": params.Status,
		"reason":     params.Reason,
		"summary":    fmt.Sprintf("Updated %s status from %s to %s", params.Repository, oldStatus, params.Status),
	}, nil
}

// --- Group 3: Batch Operations Implementations ---

func (c *Client) executeListBatches(ctx context.Context, params ListBatchesParams) (any, error) {
	limit := params.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	batches, err := c.db.ListBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	results := make([]map[string]any, 0, limit)
	for _, batch := range batches {
		// Filter by status if specified
		if params.Status != "" && batch.Status != params.Status {
			continue
		}

		result := map[string]any{
			"id":         batch.ID,
			"name":       batch.Name,
			"status":     batch.Status,
			"created_at": batch.CreatedAt.Format(time.RFC3339),
		}
		if batch.DestinationOrg != nil {
			result["destination_org"] = *batch.DestinationOrg
		}
		if batch.ScheduledAt != nil {
			result["scheduled_at"] = batch.ScheduledAt.Format(time.RFC3339)
		}

		// Get repo count for this batch
		repos, _ := c.db.ListRepositories(ctx, map[string]any{"batch_id": batch.ID})
		result["repo_count"] = len(repos)

		results = append(results, result)
		if len(results) >= limit {
			break
		}
	}

	return map[string]any{
		"batches": results,
		"count":   len(results),
		"summary": fmt.Sprintf("Found %d batches", len(results)),
	}, nil
}

// getBatchByName finds a batch by name from the list
func (c *Client) getBatchByName(ctx context.Context, name string) (*models.Batch, error) {
	batches, err := c.db.ListBatches(ctx)
	if err != nil {
		return nil, err
	}
	for _, batch := range batches {
		if batch.Name == name {
			return batch, nil
		}
	}
	return nil, fmt.Errorf("batch not found: %s", name)
}

func (c *Client) executeGetBatchDetails(ctx context.Context, params GetBatchDetailsParams) (any, error) {
	var batch *models.Batch
	var err error

	if params.BatchID > 0 {
		batch, err = c.db.GetBatch(ctx, params.BatchID)
	} else if params.BatchName != "" {
		batch, err = c.getBatchByName(ctx, params.BatchName)
	} else {
		return nil, fmt.Errorf("either batch_id or batch_name is required")
	}

	if err != nil || batch == nil {
		return nil, fmt.Errorf("batch not found")
	}

	repos, _ := c.db.ListRepositories(ctx, map[string]any{"batch_id": batch.ID})

	repoDetails := make([]map[string]any, 0, len(repos))
	statusCounts := make(map[string]int)
	for _, repo := range repos {
		statusCounts[repo.Status]++
		repoDetails = append(repoDetails, map[string]any{
			"full_name": repo.FullName,
			"status":    repo.Status,
		})
	}

	result := map[string]any{
		"id":            batch.ID,
		"name":          batch.Name,
		"status":        batch.Status,
		"created_at":    batch.CreatedAt.Format(time.RFC3339),
		"repositories":  repoDetails,
		"repo_count":    len(repos),
		"status_counts": statusCounts,
	}

	if batch.DestinationOrg != nil {
		result["destination_org"] = *batch.DestinationOrg
	}
	if batch.ScheduledAt != nil {
		result["scheduled_at"] = batch.ScheduledAt.Format(time.RFC3339)
	}

	result["summary"] = fmt.Sprintf("Batch '%s': %d repositories, status=%s", batch.Name, len(repos), batch.Status)
	return result, nil
}

func (c *Client) executeAddReposToBatch(ctx context.Context, params AddReposToBatchParams) (any, error) {
	// Check authorization - self-service or higher
	if err := c.checkToolAuthorization("add_repos_to_batch", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	var batch *models.Batch
	var err error

	if params.BatchID > 0 {
		batch, err = c.db.GetBatch(ctx, params.BatchID)
	} else if params.BatchName != "" {
		batch, err = c.getBatchByName(ctx, params.BatchName)
	} else {
		return nil, fmt.Errorf("either batch_id or batch_name is required")
	}

	if err != nil || batch == nil {
		return nil, fmt.Errorf("batch not found")
	}

	if len(params.Repositories) == 0 {
		return nil, fmt.Errorf("repositories list is required")
	}

	addedCount := 0
	failedRepos := []string{}

	for _, repoName := range params.Repositories {
		repo, err := c.db.GetRepository(ctx, repoName)
		if err != nil || repo == nil {
			failedRepos = append(failedRepos, repoName)
			continue
		}

		repo.BatchID = &batch.ID
		if err := c.db.UpdateRepository(ctx, repo); err != nil {
			failedRepos = append(failedRepos, repoName)
			continue
		}
		addedCount++
	}

	return map[string]any{
		"batch_name":   batch.Name,
		"added_count":  addedCount,
		"failed_repos": failedRepos,
		"summary":      fmt.Sprintf("Added %d repositories to batch '%s'", addedCount, batch.Name),
	}, nil
}

func (c *Client) executeRemoveReposFromBatch(ctx context.Context, params RemoveReposFromBatchParams) (any, error) {
	// Check authorization - self-service or higher
	if err := c.checkToolAuthorization("remove_repos_from_batch", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	var batch *models.Batch
	var err error

	if params.BatchID > 0 {
		batch, err = c.db.GetBatch(ctx, params.BatchID)
	} else if params.BatchName != "" {
		batch, err = c.getBatchByName(ctx, params.BatchName)
	} else {
		return nil, fmt.Errorf("either batch_id or batch_name is required")
	}

	if err != nil || batch == nil {
		return nil, fmt.Errorf("batch not found")
	}

	if len(params.Repositories) == 0 {
		return nil, fmt.Errorf("repositories list is required")
	}

	removedCount := 0
	for _, repoName := range params.Repositories {
		repo, err := c.db.GetRepository(ctx, repoName)
		if err != nil || repo == nil {
			continue
		}

		if repo.BatchID != nil && *repo.BatchID == batch.ID {
			repo.BatchID = nil
			if err := c.db.UpdateRepository(ctx, repo); err == nil {
				removedCount++
			}
		}
	}

	return map[string]any{
		"batch_name":    batch.Name,
		"removed_count": removedCount,
		"summary":       fmt.Sprintf("Removed %d repositories from batch '%s'", removedCount, batch.Name),
	}, nil
}

func (c *Client) executeRetryBatchFailures(ctx context.Context, params RetryBatchFailuresParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("retry_batch_failures", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	var batch *models.Batch
	var err error

	if params.BatchID > 0 {
		batch, err = c.db.GetBatch(ctx, params.BatchID)
	} else if params.BatchName != "" {
		batch, err = c.getBatchByName(ctx, params.BatchName)
	} else {
		return nil, fmt.Errorf("either batch_id or batch_name is required")
	}

	if err != nil || batch == nil {
		return nil, fmt.Errorf("batch not found")
	}

	repos, err := c.db.ListRepositories(ctx, map[string]any{"batch_id": batch.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to get batch repositories: %w", err)
	}

	resetCount := 0
	for _, repo := range repos {
		if repo.Status == string(models.StatusMigrationFailed) ||
			repo.Status == string(models.StatusDryRunFailed) {
			repo.Status = string(models.StatusPending)
			if err := c.db.UpdateRepository(ctx, repo); err == nil {
				resetCount++
			}
		}
	}

	return map[string]any{
		"batch_name":  batch.Name,
		"reset_count": resetCount,
		"summary":     fmt.Sprintf("Reset %d failed repositories to pending in batch '%s'", resetCount, batch.Name),
	}, nil
}

// --- Group 4: User Mapping Operations Implementations ---

func (c *Client) executeListUsers(ctx context.Context, params ListUsersParams) (any, error) {
	maxCount := params.MaxCount
	if maxCount == 0 {
		maxCount = 50
	}
	if maxCount > 200 {
		maxCount = 200
	}

	users, total, err := c.db.ListUsers(ctx, params.Organization, maxCount, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	results := make([]map[string]any, 0, len(users))
	for _, user := range users {
		result := map[string]any{
			"login": user.Login,
		}
		if user.Name != nil {
			result["name"] = *user.Name
		}
		if user.Email != nil {
			result["email"] = *user.Email
		}
		results = append(results, result)
	}

	return map[string]any{
		"users":   results,
		"count":   len(results),
		"total":   total,
		"summary": fmt.Sprintf("Found %d users (showing %d)", total, len(results)),
	}, nil
}

func (c *Client) executeGetUserStats(ctx context.Context, params GetUserStatsParams) (any, error) {
	stats, err := c.db.GetUserMappingStats(ctx, params.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return map[string]any{
		"organization": params.Organization,
		"stats":        stats,
		"summary":      fmt.Sprintf("User mapping stats for %s", params.Organization),
	}, nil
}

func (c *Client) executeListUserMappings(ctx context.Context, params ListUserMappingsParams) (any, error) {
	maxCount := params.MaxCount
	if maxCount == 0 {
		maxCount = 50
	}
	if maxCount > 200 {
		maxCount = 200
	}

	filters := storage.UserMappingFilters{
		Limit: maxCount,
	}
	if params.Organization != "" {
		filters.SourceOrg = params.Organization
	}
	if params.Status != "" {
		filters.Status = params.Status
	}

	mappings, total, err := c.db.ListUserMappings(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list user mappings: %w", err)
	}

	results := make([]map[string]any, 0, len(mappings))
	for _, m := range mappings {
		result := map[string]any{
			"source_login": m.SourceLogin,
			"status":       m.MappingStatus,
		}
		if m.DestinationLogin != nil {
			result["destination_login"] = *m.DestinationLogin
		}
		if m.MatchConfidence != nil {
			result["match_confidence"] = *m.MatchConfidence
		}
		results = append(results, result)
	}

	return map[string]any{
		"mappings": results,
		"count":    len(results),
		"total":    total,
		"summary":  fmt.Sprintf("Found %d user mappings", total),
	}, nil
}

func (c *Client) executeSuggestUserMappings(ctx context.Context, params SuggestUserMappingsParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("suggest_user_mappings", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.DestinationOrg == "" {
		return nil, fmt.Errorf("destination_org is required")
	}

	// This would typically call into the suggestion logic
	// For now, return a placeholder indicating the operation
	return map[string]any{
		"destination_org":  params.DestinationOrg,
		"source_org":       params.Organization,
		"overwrite_manual": params.OverwriteManual,
		"summary":          fmt.Sprintf("User mapping suggestions would be generated for %s -> %s", params.Organization, params.DestinationOrg),
	}, nil
}

func (c *Client) executeUpdateUserMapping(ctx context.Context, params UpdateUserMappingParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("update_user_mapping", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.SourceLogin == "" {
		return nil, fmt.Errorf("source_login is required")
	}

	mapping, err := c.db.GetUserMappingBySourceLogin(ctx, params.SourceLogin)
	if err != nil {
		return nil, fmt.Errorf("failed to get user mapping: %w", err)
	}

	if mapping == nil {
		mapping = &models.UserMapping{
			SourceLogin: params.SourceLogin,
		}
	}

	if params.DestinationLogin != "" {
		mapping.DestinationLogin = &params.DestinationLogin
	}
	if params.Status != "" {
		mapping.MappingStatus = params.Status
	}

	if err := c.db.SaveUserMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to save user mapping: %w", err)
	}

	return map[string]any{
		"source_login":      params.SourceLogin,
		"destination_login": mapping.DestinationLogin,
		"status":            mapping.MappingStatus,
		"summary":           fmt.Sprintf("Updated mapping for %s", params.SourceLogin),
	}, nil
}

func (c *Client) executeFetchMannequins(ctx context.Context, params FetchMannequinsParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("fetch_mannequins", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.Organization == "" {
		return nil, fmt.Errorf("organization is required")
	}

	// Get current mannequin count
	stats, err := c.db.GetMannequinOrgStats(ctx, params.Organization)
	if err != nil {
		return map[string]any{
			"organization": params.Organization,
			"summary":      fmt.Sprintf("No mannequins found for %s. Use the UI to fetch mannequins from GitHub.", params.Organization),
		}, nil
	}

	return map[string]any{
		"organization": params.Organization,
		"total":        stats.Total,
		"invitable":    stats.Invitable,
		"pending":      stats.Pending,
		"completed":    stats.Completed,
		"summary":      fmt.Sprintf("Found %d mannequins in %s (%d invitable)", stats.Total, params.Organization, stats.Invitable),
	}, nil
}

// --- Group 5: Team Mapping Operations Implementations ---

func (c *Client) executeListTeamMappings(ctx context.Context, params ListTeamMappingsParams) (any, error) {
	maxCount := params.MaxCount
	if maxCount == 0 {
		maxCount = 50
	}
	if maxCount > 200 {
		maxCount = 200
	}

	filters := storage.TeamMappingFilters{
		Limit: maxCount,
	}
	if params.SourceOrg != "" {
		filters.SourceOrg = params.SourceOrg
	}
	if params.Status != "" {
		filters.Status = params.Status
	}

	mappings, total, err := c.db.ListTeamMappings(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list team mappings: %w", err)
	}

	results := make([]map[string]any, 0, len(mappings))
	for _, m := range mappings {
		result := map[string]any{
			"source_team":      fmt.Sprintf("%s/%s", m.SourceOrg, m.SourceTeamSlug),
			"migration_status": m.MigrationStatus,
		}
		if m.DestinationOrg != nil {
			result["destination_org"] = *m.DestinationOrg
		}
		if m.DestinationTeamSlug != nil {
			result["destination_team"] = *m.DestinationTeamSlug
		}
		results = append(results, result)
	}

	return map[string]any{
		"mappings": results,
		"count":    len(results),
		"total":    total,
		"summary":  fmt.Sprintf("Found %d team mappings", total),
	}, nil
}

func (c *Client) executeSuggestTeamMappings(ctx context.Context, params SuggestTeamMappingsParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("suggest_team_mappings", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.DestinationOrg == "" {
		return nil, fmt.Errorf("destination_org is required")
	}

	suggestions, err := c.db.SuggestTeamMappings(ctx, params.DestinationOrg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest team mappings: %w", err)
	}

	results := make([]map[string]any, 0)
	for sourceTeam, destTeam := range suggestions {
		results = append(results, map[string]any{
			"source_team":      sourceTeam,
			"destination_team": destTeam,
		})
	}

	return map[string]any{
		"destination_org": params.DestinationOrg,
		"suggestions":     results,
		"count":           len(results),
		"summary":         fmt.Sprintf("Generated %d team mapping suggestions for %s", len(results), params.DestinationOrg),
	}, nil
}

func (c *Client) executeExecuteTeamMigration(ctx context.Context, params ExecuteTeamMigrationParams) (any, error) {
	// Check authorization - admin only
	if err := c.checkToolAuthorization("execute_team_migration", c.getCurrentAuth()); err != nil {
		return nil, err
	}

	if params.DestinationOrg == "" {
		return nil, fmt.Errorf("destination_org is required")
	}

	teams, err := c.db.GetMappedTeamsForMigration(ctx, params.SourceOrg)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams for migration: %w", err)
	}

	if params.DryRun {
		return map[string]any{
			"dry_run":          true,
			"destination_org":  params.DestinationOrg,
			"teams_to_migrate": len(teams),
			"summary":          fmt.Sprintf("Dry run: %d teams would be migrated to %s", len(teams), params.DestinationOrg),
		}, nil
	}

	// Queue teams for migration
	for _, team := range teams {
		team.MigrationStatus = models.TeamMigrationPending
		if err := c.db.SaveTeamMapping(ctx, team); err != nil {
			c.logger.Error("Failed to queue team for migration", "team", team.SourceTeamSlug, "error", err)
		}
	}

	return map[string]any{
		"destination_org": params.DestinationOrg,
		"teams_queued":    len(teams),
		"summary":         fmt.Sprintf("Queued %d teams for migration to %s", len(teams), params.DestinationOrg),
	}, nil
}

func (c *Client) executeGetTeamMigrationExecutionStatus(ctx context.Context, _ GetTeamMigrationExecutionStatusParams) (any, error) {
	stats, err := c.db.GetTeamMigrationExecutionStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get team migration status: %w", err)
	}

	return map[string]any{
		"stats":   stats,
		"summary": "Team migration execution status",
	}, nil
}

// --- Group 6: Analytics and Reporting Implementations ---

func (c *Client) executeGetAnalyticsSummary(ctx context.Context, params GetAnalyticsSummaryParams) (any, error) {
	stats, err := c.db.GetRepositoryStatsByStatusFiltered(ctx, params.Organization, "", "", params.SourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository stats: %w", err)
	}

	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]
	failed := stats[string(models.StatusMigrationFailed)] + stats[string(models.StatusDryRunFailed)]
	pending := stats[string(models.StatusPending)]
	inProgress := stats[string(models.StatusMigratingContent)] + stats[string(models.StatusQueuedForMigration)]

	successRate := 0.0
	if migrated+failed > 0 {
		successRate = float64(migrated) / float64(migrated+failed) * 100
	}

	percentComplete := 0.0
	if total > 0 {
		percentComplete = float64(migrated) / float64(total) * 100
	}

	// Get complexity distribution
	complexityDist, _ := c.db.GetComplexityDistribution(ctx, params.Organization, "", "", params.SourceID)

	return map[string]any{
		"total_repos":             total,
		"migrated":                migrated,
		"failed":                  failed,
		"pending":                 pending,
		"in_progress":             inProgress,
		"success_rate":            successRate,
		"percent_complete":        percentComplete,
		"complexity_distribution": complexityDist,
		"status_breakdown":        stats,
		"summary": fmt.Sprintf("Migration progress: %d/%d complete (%.1f%%), %.1f%% success rate",
			migrated, total, percentComplete, successRate),
	}, nil
}

func (c *Client) executeGetExecutiveReport(ctx context.Context, params GetExecutiveReportParams) (any, error) {
	// Get basic stats
	stats, err := c.db.GetRepositoryStatsByStatusFiltered(ctx, params.Organization, "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository stats: %w", err)
	}

	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]
	failed := stats[string(models.StatusMigrationFailed)]

	// Get org stats
	orgStats, _ := c.db.GetOrganizationStatsFiltered(ctx, params.Organization, "", "", nil)

	// Get velocity
	velocity, _ := c.db.GetMigrationVelocity(ctx, params.Organization, "", "", nil, 30)

	report := map[string]any{
		"total_repositories": total,
		"migrated":           migrated,
		"failed":             failed,
		"remaining":          total - migrated,
		"organizations":      orgStats,
	}

	if velocity != nil {
		report["repos_per_day"] = velocity.ReposPerDay
		if velocity.ReposPerDay > 0 {
			remaining := total - migrated
			daysRemaining := float64(remaining) / velocity.ReposPerDay
			report["estimated_days_remaining"] = int(daysRemaining)
		}
	}

	percentComplete := 0.0
	if total > 0 {
		percentComplete = float64(migrated) / float64(total) * 100
	}

	report["summary"] = fmt.Sprintf("Executive Summary: %d/%d repos migrated (%.1f%%), %d remaining",
		migrated, total, percentComplete, total-migrated)

	return report, nil
}

func (c *Client) executeGetPermissionAudit(ctx context.Context, params GetPermissionAuditParams) (any, error) {
	// Get teams for the organization
	teams, err := c.db.ListTeams(ctx, params.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	teamAudits := make([]map[string]any, 0)
	for _, team := range teams {
		if params.Team != "" {
			teamKey := fmt.Sprintf("%s/%s", team.Organization, team.Slug)
			if teamKey != params.Team {
				continue
			}
		}

		memberCount, _ := c.db.GetTeamMemberCount(ctx, team.ID)
		teamAudits = append(teamAudits, map[string]any{
			"team":         fmt.Sprintf("%s/%s", team.Organization, team.Slug),
			"name":         team.Name,
			"member_count": memberCount,
			"privacy":      team.Privacy,
		})
	}

	return map[string]any{
		"organization": params.Organization,
		"teams":        teamAudits,
		"team_count":   len(teamAudits),
		"summary":      fmt.Sprintf("Permission audit: %d teams in %s", len(teamAudits), params.Organization),
	}, nil
}

// --- Group 7: Organization Operations Implementations ---

func (c *Client) executeListOrganizations(ctx context.Context, params ListOrganizationsParams) (any, error) {
	// Get organization stats which includes org names
	orgStats, err := c.db.GetOrganizationStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	results := make([]map[string]any, 0, len(orgStats))
	for _, org := range orgStats {
		result := map[string]any{
			"name":       org.Organization,
			"repo_count": org.TotalRepos,
			"migrated":   org.MigratedCount,
			"pending":    org.PendingCount,
			"failed":     org.FailedCount,
		}
		results = append(results, result)
	}

	return map[string]any{
		"organizations": results,
		"count":         len(results),
		"summary":       fmt.Sprintf("Found %d organizations", len(results)),
	}, nil
}
