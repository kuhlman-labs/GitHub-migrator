package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server and provides migration-specific tools
type Server struct {
	mcpServer *server.MCPServer
	sseServer *server.SSEServer
	db        *storage.Database
	logger    *slog.Logger
	addr      string
	mu        sync.RWMutex
	running   bool
}

// Config holds configuration for the MCP server
type Config struct {
	// Address to listen on (e.g., ":8081")
	Address string
}

// NewServer creates a new MCP server with migration tools
func NewServer(db *storage.Database, logger *slog.Logger, cfg Config) *Server {
	// Create the MCP server with capabilities
	mcpServer := server.NewMCPServer(
		"GitHub Migrator",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithInstructions(`You are the GitHub Migrator assistant. You have access to tools that help analyze repositories, 
plan migrations, create batches, and check migration status. Use these tools to help users plan and execute 
their GitHub migrations effectively.

Key capabilities:
- Analyze repositories by organization, complexity, and status
- Get detailed complexity breakdowns for individual repositories
- Check dependencies between repositories
- Find good candidates for pilot migrations
- Create and schedule migration batches
- Plan migration waves that respect dependencies
- Get team repositories and migration status`),
	)

	s := &Server{
		mcpServer: mcpServer,
		db:        db,
		logger:    logger,
		addr:      cfg.Address,
	}

	// Register all migration tools
	s.registerTools()

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

	// Create SSE server for HTTP-based communication
	s.sseServer = server.NewSSEServer(s.mcpServer,
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
	)

	// Start the server (this blocks)
	if err := s.sseServer.Start(s.addr); err != nil && err != http.ErrServerClosed {
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

	if s.sseServer != nil {
		if err := s.sseServer.Shutdown(ctx); err != nil {
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
			mcp.WithDescription("Query and analyze repositories based on various criteria like complexity, migration status, features, and organization. Returns a list of repositories matching the filters."),
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
				mcp.Description("Maximum number of repositories to return (default 20, max 100)"),
			),
		),
		s.handleAnalyzeRepositories,
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

	// find_pilot_candidates - Find good pilot migration candidates
	s.mcpServer.AddTool(
		mcp.NewTool("find_pilot_candidates",
			mcp.WithDescription("Identify repositories suitable for a pilot migration based on low complexity, few dependencies, and low risk. Good for testing migration procedures."),
			mcp.WithNumber("max_count",
				mcp.Description("Maximum number of candidates to return (default 10)"),
			),
			mcp.WithString("organization",
				mcp.Description("Filter by organization"),
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

	s.logger.Info("Registered MCP tools", "count", 10)
}
