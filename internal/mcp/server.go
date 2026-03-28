package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/kuhlman-labs/github-migrator/internal/configsvc"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server and provides migration-specific tools
type Server struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	db         *storage.Database
	logger     *slog.Logger
	configSvc  *configsvc.Service
	addr       string
	mu         sync.RWMutex
	running    bool

	// Discovery cancel tracking
	discoveryCancel map[int64]context.CancelFunc
	discoveryMu     sync.RWMutex
}

// Config holds configuration for the MCP server
type Config struct {
	// Address to listen on (e.g., ":8081")
	Address string
}

// SetConfigService sets the config service for hot-reloading after settings changes
func (s *Server) SetConfigService(configSvc *configsvc.Service) {
	s.configSvc = configSvc
}

// NewServer creates a new MCP server with migration tools
func NewServer(db *storage.Database, logger *slog.Logger, cfg Config) *Server {
	// Create the MCP server with capabilities
	mcpServer := server.NewMCPServer(
		"GitHub Migrator",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithRecovery(),
		server.WithInstructions(`You are the GitHub Migrator assistant. You have access to tools that help discover, analyze, plan, execute, and track GitHub migrations across the full lifecycle.

Key capabilities:
- Setup: Configure destination (get_destination, configure_destination), manage sources (create_source, list_sources)
- Discovery: List sources, start repository discovery, track discovery progress
- Analysis: Analyze repositories by complexity/status/org, check dependencies, find pilot candidates
- Readiness: Pre-flight readiness checks, user/team mapping stats, identify blockers, suggest mappings
- Batching: Create/configure/schedule migration batches, plan migration waves
- Execution: Start migrations (dry-run or production), cancel migrations, track progress
- Post-Migration: View migration history/logs, get executive summaries, rollback migrations
- Action Items: Get prioritized list of what needs attention across all domains

Workflow guidance:
1. Start by discovering repos from a source (list_sources → create_source if needed → start_discovery → get_discovery_progress)
2. Analyze discovered repos (analyze_repositories, get_complexity_breakdown, check_dependencies)
3. Check readiness (get_migration_readiness, get_user_mapping_stats, get_repository_blockers)
4. Plan migration waves (plan_waves, find_pilot_candidates)
5. Create and configure batches (create_batch, configure_batch)
6. Execute migrations (start_migration with dry_run=true first, then dry_run=false)
7. Monitor and report (get_migration_progress, get_executive_summary, get_action_items)`),
	)

	s := &Server{
		mcpServer:       mcpServer,
		db:              db,
		logger:          logger,
		addr:            cfg.Address,
		discoveryCancel: make(map[int64]context.CancelFunc),
	}

	// Register all migration tools and resources
	s.registerTools()
	s.registerResources()

	return s
}

// Start starts the MCP server on the configured address
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("MCP server already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting MCP server", "address", s.addr)

	// Create Streamable HTTP server for MCP communication
	s.httpServer = server.NewStreamableHTTPServer(s.mcpServer)

	// Start the server (this blocks)
	if err := s.httpServer.Start(s.addr); err != nil && err != http.ErrServerClosed {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the MCP server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping MCP server")
	s.running = false

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown MCP server: %w", err)
		}
	}

	return nil
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Address returns the server's listening address
func (s *Server) Address() string {
	return s.addr
}

// registerTools registers all migration-related tools with the MCP server
func (s *Server) registerTools() {
	// analyze_repositories - Query and analyze repositories
	s.mcpServer.AddTool(
		mcp.NewTool("analyze_repositories",
			mcp.WithDescription("Query and analyze repositories based on various criteria. Returns matching repositories with aggregate statistics (complexity distribution, feature counts, size totals, blocker count). Accepts an optional list of specific repositories to analyze."),
			mcp.WithString("organization",
				mcp.Description("Filter by organization name"),
			),
			mcp.WithString("status",
				mcp.Description("Filter by migration status"),
				mcp.Enum("pending", "in_progress", "completed", "failed"),
			),
			mcp.WithNumber("max_complexity",
				mcp.Description("Maximum complexity score (1-100)"),
			),
			mcp.WithNumber("min_complexity",
				mcp.Description("Minimum complexity score (1-100)"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of repositories to return. Omit or set to 0 to return all repositories."),
			),
			mcp.WithArray("repositories",
				mcp.Description("Optional list of specific repository full names (org/repo) to analyze. When provided, other filters are still applied to this subset."),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		s.handleAnalyzeRepositories,
	)

	// get_repository_details - Get comprehensive details for a single repository
	s.mcpServer.AddTool(
		mcp.NewTool("get_repository_details",
			mcp.WithDescription("Get comprehensive details for a single repository including complexity breakdown, features, dependencies, and migration history. Consolidates multiple tool calls into one."),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("Full repository name (org/repo)"),
			),
			mcp.WithBoolean("include_reverse",
				mcp.Description("Include repositories that depend on this one"),
			),
			mcp.WithNumber("history_limit",
				mcp.Description("Maximum number of migration history records to include (default 10)"),
			),
		),
		s.handleGetRepositoryDetails,
	)

	// get_complexity_breakdown - Get detailed complexity for a repository
	s.mcpServer.AddTool(
		mcp.NewTool("get_complexity_breakdown",
			mcp.WithDescription("Get detailed complexity breakdown for a specific repository including size, features, activity scores, and migration blockers."),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("Full repository name (org/repo)"),
			),
		),
		s.handleGetComplexityBreakdown,
	)

	// check_dependencies - Find repository dependencies
	s.mcpServer.AddTool(
		mcp.NewTool("check_dependencies",
			mcp.WithDescription("Find all dependencies for a repository (submodules, workflow references, packages) and check their migration status. Optionally include reverse dependencies."),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("Full repository name (org/repo)"),
			),
			mcp.WithBoolean("include_reverse",
				mcp.Description("Include repositories that depend on this one"),
			),
		),
		s.handleCheckDependencies,
	)

	// get_dependency_graph - Enterprise-wide local dependency graph
	s.mcpServer.AddTool(
		mcp.NewTool("get_dependency_graph",
			mcp.WithDescription("Get the enterprise-wide internal dependency graph. Shows all repositories that have local (cross-repo) dependencies, their relationships, and dependency counts. Essential for migration wave planning."),
			mcp.WithString("dependency_type",
				mcp.Description("Filter by dependency type: submodule, workflow, dependency_graph, package (comma-separated for multiple)"),
			),
		),
		s.handleGetDependencyGraph,
	)

	// update_repository_status - Set migration status on repositories
	s.mcpServer.AddTool(
		mcp.NewTool("update_repository_status",
			mcp.WithDescription("Update the migration status of one or more repositories. Use to mark repos as won't migrate, migrated, or reset to pending."),
			mcp.WithString("action",
				mcp.Required(),
				mcp.Description("Action to perform: mark_wont_migrate, unmark_wont_migrate, mark_migrated, reset_to_pending"),
			),
			mcp.WithArray("repositories",
				mcp.Required(),
				mcp.Description("List of repository full names (org/repo) to update"),
			),
		),
		s.handleUpdateRepositoryStatus,
	)

	// find_pilot_candidates - Find good pilot migration candidates
	s.mcpServer.AddTool(
		mcp.NewTool("find_pilot_candidates",
			mcp.WithDescription("Identify repositories suitable for a pilot migration based on complexity, few dependencies, and low risk. Searches across all orgs/sources by default — use filters to narrow scope."),
			mcp.WithNumber("max_count",
				mcp.Description("Maximum number of candidates to return (default 10)"),
			),
			mcp.WithString("organization",
				mcp.Description("Filter by organization"),
			),
			mcp.WithNumber("source_id",
				mcp.Description("Filter by discovery source ID"),
			),
			mcp.WithNumber("max_complexity",
				mcp.Description("Maximum complexity score for candidates (default 5 = simple only, use 10 for medium, 17 for complex)"),
			),
		),
		s.handleFindPilotCandidates,
	)

	// create_batch - Create a migration batch
	s.mcpServer.AddTool(
		mcp.NewTool("create_batch",
			mcp.WithDescription("Create a new migration batch with specified repositories. The batch can then be scheduled for execution."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the batch"),
			),
			mcp.WithString("description",
				mcp.Description("Description of the batch"),
			),
			mcp.WithArray("repositories",
				mcp.Required(),
				mcp.Description("List of repository full names to include"),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		s.handleCreateBatch,
	)

	// plan_waves - Generate migration wave plan
	s.mcpServer.AddTool(
		mcp.NewTool("plan_waves",
			mcp.WithDescription("Generate an optimal migration wave plan that respects dependencies and minimizes risk. Organizes pending repositories into waves for systematic migration."),
			mcp.WithNumber("wave_size",
				mcp.Description("Target number of repositories per wave (default 10)"),
			),
			mcp.WithString("organization",
				mcp.Description("Filter by organization"),
			),
		),
		s.handlePlanWaves,
	)

	// get_team_repositories - Find repositories for a team
	s.mcpServer.AddTool(
		mcp.NewTool("get_team_repositories",
			mcp.WithDescription("Find all repositories owned by or associated with a specific team."),
			mcp.WithString("team",
				mcp.Required(),
				mcp.Description("Team name in format org/team-slug"),
			),
			mcp.WithBoolean("include_migrated",
				mcp.Description("Include already migrated repositories"),
			),
		),
		s.handleGetTeamRepositories,
	)

	// get_migration_status - Get status for specific repositories
	s.mcpServer.AddTool(
		mcp.NewTool("get_migration_status",
			mcp.WithDescription("Get the current migration status for specific repositories."),
			mcp.WithArray("repositories",
				mcp.Required(),
				mcp.Description("List of repository full names to check"),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		s.handleGetMigrationStatus,
	)

	// schedule_batch - Schedule a batch for execution
	s.mcpServer.AddTool(
		mcp.NewTool("schedule_batch",
			mcp.WithDescription("Schedule a batch for migration execution at a specific date/time."),
			mcp.WithString("batch_name",
				mcp.Required(),
				mcp.Description("Name of the batch to schedule"),
			),
			mcp.WithString("scheduled_at",
				mcp.Required(),
				mcp.Description("ISO 8601 datetime for when to execute the batch (e.g., 2024-01-15T09:00:00Z)"),
			),
		),
		s.handleScheduleBatch,
	)

	// configure_batch - Configure batch settings
	s.mcpServer.AddTool(
		mcp.NewTool("configure_batch",
			mcp.WithDescription("Configure batch settings including destination organization and migration API. Use this to set where repositories in a batch will be migrated to."),
			mcp.WithString("batch_name",
				mcp.Description("Name of the batch to configure"),
			),
			mcp.WithNumber("batch_id",
				mcp.Description("ID of the batch to configure (alternative to batch_name)"),
			),
			mcp.WithString("destination_org",
				mcp.Description("Destination organization within the configured enterprise where repositories will be migrated"),
			),
			mcp.WithString("migration_api",
				mcp.Description("Migration API to use: 'GEI' (GitHub Enterprise Importer) or 'ELM' (Enterprise Live Migrator)"),
			),
		),
		s.handleConfigureBatch,
	)

	// start_migration - Start a migration (dry-run or production)
	s.mcpServer.AddTool(
		mcp.NewTool("start_migration",
			mcp.WithDescription("Start a migration (dry-run or production) for a batch, single repository, or list of repositories. Defaults to dry-run for safety."),
			mcp.WithString("batch_name",
				mcp.Description("Name of batch to migrate"),
			),
			mcp.WithNumber("batch_id",
				mcp.Description("ID of batch to migrate"),
			),
			mcp.WithString("repository",
				mcp.Description("Single repository to migrate (format: org/repo)"),
			),
			mcp.WithArray("repositories",
				mcp.Description("List of repositories to migrate"),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithBoolean("dry_run",
				mcp.Description("If true, perform dry-run only. Defaults to true for safety. Set to false for production migration."),
			),
		),
		s.handleStartMigration,
	)

	// cancel_migration - Cancel a running migration
	s.mcpServer.AddTool(
		mcp.NewTool("cancel_migration",
			mcp.WithDescription("Cancel a running migration for a batch or specific repository."),
			mcp.WithString("batch_name",
				mcp.Description("Name of batch to cancel"),
			),
			mcp.WithNumber("batch_id",
				mcp.Description("ID of batch to cancel"),
			),
			mcp.WithString("repository",
				mcp.Description("Single repository to cancel (format: org/repo)"),
			),
		),
		s.handleCancelMigration,
	)

	// get_migration_progress - Get real-time progress of migrations
	s.mcpServer.AddTool(
		mcp.NewTool("get_migration_progress",
			mcp.WithDescription("Get real-time progress of running migrations for a batch or specific repository."),
			mcp.WithString("batch_name",
				mcp.Description("Name of batch to check progress"),
			),
			mcp.WithNumber("batch_id",
				mcp.Description("ID of batch to check progress"),
			),
			mcp.WithString("repository",
				mcp.Description("Single repository to check progress (format: org/repo)"),
			),
		),
		s.handleGetMigrationProgress,
	)

	// --- Destination Tools ---

	// get_destination - Get current destination configuration
	s.mcpServer.AddTool(
		mcp.NewTool("get_destination",
			mcp.WithDescription("Get the current migration destination configuration. Shows the target GitHub instance where repositories will be migrated to, with sensitive fields masked."),
		),
		s.handleGetDestination,
	)

	// configure_destination - Configure migration destination
	s.mcpServer.AddTool(
		mcp.NewTool("configure_destination",
			mcp.WithDescription("Configure the migration destination (target GitHub instance). This is the GitHub Enterprise Cloud or Server where repositories will be migrated to. A destination must be configured before running migrations."),
			mcp.WithString("base_url",
				mcp.Description("GitHub API base URL (default: https://api.github.com)"),
			),
			mcp.WithString("token",
				mcp.Description("Personal access token for the destination GitHub instance"),
			),
			mcp.WithString("enterprise_slug",
				mcp.Description("Enterprise slug for the destination (for GitHub Enterprise)"),
			),
		),
		s.handleConfigureDestination,
	)

	// --- Discovery Tools ---

	// create_source - Create a new migration source
	s.mcpServer.AddTool(
		mcp.NewTool("create_source",
			mcp.WithDescription("Create a new migration source for discovering repositories. Sources define where repositories are discovered from (e.g., GitHub Enterprise Cloud/Server, Azure DevOps)."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("User-friendly name for the source (e.g., 'GHES Production')"),
			),
			mcp.WithString("type",
				mcp.Description("Source type: 'github' or 'azuredevops' (default: github)"),
			),
			mcp.WithString("base_url",
				mcp.Description("API base URL (default: https://api.github.com for GitHub sources)"),
			),
			mcp.WithString("token",
				mcp.Required(),
				mcp.Description("Personal access token for authentication"),
			),
			mcp.WithString("organization",
				mcp.Description("Organization name (required for Azure DevOps)"),
			),
			mcp.WithString("enterprise_slug",
				mcp.Description("Enterprise slug for GitHub Enterprise discovery"),
			),
		),
		s.handleCreateSource,
	)

	// list_sources - List configured migration sources
	s.mcpServer.AddTool(
		mcp.NewTool("list_sources",
			mcp.WithDescription("List all configured migration sources. Sources define where repositories are discovered from (e.g., GitHub Enterprise Server instances)."),
			mcp.WithBoolean("active_only",
				mcp.Description("If true, only return active sources"),
			),
		),
		s.handleListSources,
	)

	// get_source - Get details for a specific source
	s.mcpServer.AddTool(
		mcp.NewTool("get_source",
			mcp.WithDescription("Get detailed information about a specific migration source by ID."),
			mcp.WithNumber("source_id",
				mcp.Required(),
				mcp.Description("ID of the source to retrieve"),
			),
		),
		s.handleGetSource,
	)

	// start_discovery - Start repository discovery for a source
	s.mcpServer.AddTool(
		mcp.NewTool("start_discovery",
			mcp.WithDescription("Start repository discovery for a configured source. Discovers repositories from either an organization or an entire enterprise. Runs asynchronously — use get_discovery_progress to track. If source_id is omitted, auto-resolves from enterprise_slug or uses the only active source. If neither organization nor enterprise_slug is specified, uses the source's configured values."),
			mcp.WithNumber("source_id",
				mcp.Description("ID of the source to discover from. Optional if enterprise_slug is provided or only one active source exists."),
			),
			mcp.WithString("organization",
				mcp.Description("Organization to discover repositories from (mutually exclusive with enterprise_slug). Falls back to source's configured organization if omitted."),
			),
			mcp.WithString("enterprise_slug",
				mcp.Description("Enterprise slug to discover all repositories from (mutually exclusive with organization). Falls back to source's configured enterprise_slug if omitted."),
			),
		),
		s.handleStartDiscovery,
	)

	// get_discovery_progress - Get latest discovery progress
	s.mcpServer.AddTool(
		mcp.NewTool("get_discovery_progress",
			mcp.WithDescription("Get the progress of the latest discovery operation, including status, phase, and repository counts."),
		),
		s.handleGetDiscoveryProgress,
	)

	// cancel_discovery - Cancel an active discovery
	s.mcpServer.AddTool(
		mcp.NewTool("cancel_discovery",
			mcp.WithDescription("Cancel the currently active discovery operation. Returns an error if no discovery is in progress."),
		),
		s.handleCancelDiscovery,
	)

	// --- Readiness Tools ---

	// get_migration_readiness - Pre-flight readiness check
	s.mcpServer.AddTool(
		mcp.NewTool("get_migration_readiness",
			mcp.WithDescription("Pre-flight readiness check for migration. Evaluates user mappings, team mappings, blockers, and dry-run status to produce a readiness score (0-100%) with a pass/fail checklist."),
			mcp.WithString("organization",
				mcp.Description("Filter readiness check by organization"),
			),
			mcp.WithNumber("batch_id",
				mcp.Description("Filter readiness check by batch ID"),
			),
			mcp.WithNumber("source_id",
				mcp.Description("Filter by source ID"),
			),
		),
		s.handleGetMigrationReadiness,
	)

	// get_user_mapping_stats - User mapping summary
	s.mcpServer.AddTool(
		mcp.NewTool("get_user_mapping_stats",
			mcp.WithDescription("Get summary statistics of user identity mapping progress, including total users, mapped count, unmapped count, and mapping percentage."),
			mcp.WithString("source_org",
				mcp.Description("Filter by source organization"),
			),
			mcp.WithNumber("source_id",
				mcp.Description("Filter by source ID"),
			),
		),
		s.handleGetUserMappingStats,
	)

	// get_team_mapping_stats - Team mapping summary
	s.mcpServer.AddTool(
		mcp.NewTool("get_team_mapping_stats",
			mcp.WithDescription("Get summary statistics of team mapping progress, including total teams, mapped count, unmapped count, and mapping percentage."),
			mcp.WithString("source_org",
				mcp.Description("Filter by source organization"),
			),
			mcp.WithNumber("source_id",
				mcp.Description("Filter by source ID"),
			),
		),
		s.handleGetTeamMappingStats,
	)

	// get_repository_blockers - List repos with migration blockers
	s.mcpServer.AddTool(
		mcp.NewTool("get_repository_blockers",
			mcp.WithDescription("List repositories with migration blockers that require remediation before migration. Includes blocker details and remediation guidance."),
			mcp.WithString("organization",
				mcp.Description("Filter by organization"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of repositories to return (default 20, max 100)"),
			),
		),
		s.handleGetRepositoryBlockers,
	)

	// suggest_user_mappings - Sync and suggest user identity mappings
	s.mcpServer.AddTool(
		mcp.NewTool("suggest_user_mappings",
			mcp.WithDescription("Sync user mappings from discovered users and provide recommendations for unmapped users. This syncs the mappings table with discovered users and reports current mapping coverage."),
		),
		s.handleSuggestUserMappings,
	)

	// suggest_team_mappings - Suggest team mappings based on name similarity
	s.mcpServer.AddTool(
		mcp.NewTool("suggest_team_mappings",
			mcp.WithDescription("Suggest team mappings by matching source teams to destination teams based on name similarity. Syncs team mappings from discovered teams first."),
			mcp.WithString("destination_org",
				mcp.Required(),
				mcp.Description("Destination organization to match teams against"),
			),
		),
		s.handleSuggestTeamMappings,
	)

	// --- Analytics & Post-Migration Tools ---

	// get_executive_summary - High-level migration overview
	s.mcpServer.AddTool(
		mcp.NewTool("get_executive_summary",
			mcp.WithDescription("Get a high-level executive summary of migration progress including total counts, success rates, velocity, and estimated completion date."),
			mcp.WithString("organization",
				mcp.Description("Filter by organization name"),
			),
			mcp.WithNumber("source_id",
				mcp.Description("Filter by source ID"),
			),
		),
		s.handleGetExecutiveSummary,
	)

	// get_migration_history - View migration history records
	s.mcpServer.AddTool(
		mcp.NewTool("get_migration_history",
			mcp.WithDescription("Get migration history records. Can show history for a specific repository or all completed/failed migrations."),
			mcp.WithString("repository",
				mcp.Description("Full repository name (org/repo) to get history for"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of records to return (default 20, max 100)"),
			),
			mcp.WithString("status_filter",
				mcp.Description("Filter by status category"),
				mcp.Enum("completed", "failed"),
			),
		),
		s.handleGetMigrationHistory,
	)

	// get_migration_logs - View migration log entries
	s.mcpServer.AddTool(
		mcp.NewTool("get_migration_logs",
			mcp.WithDescription("Get migration log entries for a specific repository. Useful for debugging migration failures."),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("Full repository name (org/repo)"),
			),
			mcp.WithString("level",
				mcp.Description("Filter by log level"),
				mcp.Enum("error", "warn", "info"),
			),
			mcp.WithString("phase",
				mcp.Description("Filter by migration phase"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of log entries to return (default 50, max 500)"),
			),
		),
		s.handleGetMigrationLogs,
	)

	// get_action_items - Get prioritized action items
	s.mcpServer.AddTool(
		mcp.NewTool("get_action_items",
			mcp.WithDescription("Get prioritized action items across all migration domains including failed repos, unmapped users/teams, and pending batches. Returns items sorted by severity."),
		),
		s.handleGetActionItems,
	)

	// rollback_migration - Rollback a completed migration
	s.mcpServer.AddTool(
		mcp.NewTool("rollback_migration",
			mcp.WithDescription("Rollback a completed migration for a specific repository. Only works on repositories with 'complete' or 'migration_complete' status."),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("Full repository name (org/repo) to rollback"),
			),
			mcp.WithString("reason",
				mcp.Description("Reason for the rollback"),
			),
		),
		s.handleRollbackMigration,
	)

	s.logger.Info("Registered MCP tools", "count", 33)
}
