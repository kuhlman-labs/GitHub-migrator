// Package copilot provides the Copilot chat service integration.
//
// DEPRECATION NOTICE: The tool implementations in this file are stubs.
// Real tool execution is now handled by the MCP server in internal/mcp/.
// The ToolRegistry is kept for backward compatibility to provide tool
// descriptions for the system prompt. When the Copilot CLI connects to
// the MCP server, tools are executed via the MCP protocol with full
// database access.
package copilot

import (
	"context"
	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Tool represents a custom tool that Copilot can use.
// NOTE: Tool execution is now handled by the MCP server.
// This struct is kept for building system prompts with tool descriptions.
type Tool struct {
	Name        string                                                      `json:"name"`
	Description string                                                      `json:"description"`
	Parameters  map[string]ToolParameter                                    `json:"parameters"`
	Execute     func(ctx context.Context, args map[string]any) (any, error) `json:"-"`
}

// ToolParameter describes a parameter for a tool
type ToolParameter struct {
	Type        string   `json:"type"` // "string", "number", "boolean", "array", "object"
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"` // For string types with allowed values
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools  map[string]Tool
	db     *storage.Database
	logger *slog.Logger
}

// NewToolRegistry creates a new tool registry with default tools
func NewToolRegistry(db *storage.Database, logger *slog.Logger) *ToolRegistry {
	registry := &ToolRegistry{
		tools:  make(map[string]Tool),
		db:     db,
		logger: logger,
	}

	// Register default tools
	registry.registerDefaultTools()

	return registry
}

// RegisterTool adds a tool to the registry
func (r *ToolRegistry) RegisterTool(tool Tool) {
	r.tools[tool.Name] = tool
	r.logger.Debug("Registered Copilot tool", "name", tool.Name)
}

// GetTool returns a tool by name
func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetTools returns all registered tools
func (r *ToolRegistry) GetTools() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ExecuteTool executes a tool by name with the given arguments
func (r *ToolRegistry) ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, &ToolNotFoundError{Name: name}
	}
	return tool.Execute(ctx, args)
}

// ToolNotFoundError is returned when a tool is not found
type ToolNotFoundError struct {
	Name string
}

func (e *ToolNotFoundError) Error() string {
	return "tool not found: " + e.Name
}

// registerDefaultTools adds the default migration-related tools
func (r *ToolRegistry) registerDefaultTools() {
	// Analyze repositories tool
	r.RegisterTool(Tool{
		Name:        "analyze_repositories",
		Description: "Query and analyze repositories based on various criteria like complexity, migration status, features, and organization",
		Parameters: map[string]ToolParameter{
			"organization": {
				Type:        "string",
				Description: "Filter by organization name",
				Required:    false,
			},
			"status": {
				Type:        "string",
				Description: "Filter by migration status",
				Required:    false,
				Enum:        []string{"pending", "in_progress", "completed", "failed", "not_started"},
			},
			"max_complexity": {
				Type:        "number",
				Description: "Maximum complexity score (1-100)",
				Required:    false,
			},
			"min_complexity": {
				Type:        "number",
				Description: "Minimum complexity score (1-100)",
				Required:    false,
			},
			"limit": {
				Type:        "number",
				Description: "Maximum number of repositories to return",
				Required:    false,
			},
		},
		Execute: r.analyzeRepositories,
	})

	// Get complexity breakdown tool
	r.RegisterTool(Tool{
		Name:        "get_complexity_breakdown",
		Description: "Get detailed complexity breakdown for a specific repository including size, features, and activity scores",
		Parameters: map[string]ToolParameter{
			"repository": {
				Type:        "string",
				Description: "Full repository name (org/repo)",
				Required:    true,
			},
		},
		Execute: r.getComplexityBreakdown,
	})

	// Check dependencies tool
	r.RegisterTool(Tool{
		Name:        "check_dependencies",
		Description: "Find all dependencies for a repository and check their migration status",
		Parameters: map[string]ToolParameter{
			"repository": {
				Type:        "string",
				Description: "Full repository name (org/repo)",
				Required:    true,
			},
			"include_reverse": {
				Type:        "boolean",
				Description: "Include repositories that depend on this one",
				Required:    false,
			},
		},
		Execute: r.checkDependencies,
	})

	// Find pilot candidates tool
	r.RegisterTool(Tool{
		Name:        "find_pilot_candidates",
		Description: "Identify repositories suitable for a pilot migration based on low complexity, few dependencies, and low risk",
		Parameters: map[string]ToolParameter{
			"max_count": {
				Type:        "number",
				Description: "Maximum number of candidates to return",
				Required:    false,
			},
			"organization": {
				Type:        "string",
				Description: "Filter by organization",
				Required:    false,
			},
		},
		Execute: r.findPilotCandidates,
	})

	// Create batch tool
	r.RegisterTool(Tool{
		Name:        "create_batch",
		Description: "Create a new migration batch with specified repositories",
		Parameters: map[string]ToolParameter{
			"name": {
				Type:        "string",
				Description: "Name for the batch",
				Required:    true,
			},
			"description": {
				Type:        "string",
				Description: "Description of the batch",
				Required:    false,
			},
			"repositories": {
				Type:        "array",
				Description: "List of repository full names to include",
				Required:    true,
			},
		},
		Execute: r.createBatch,
	})

	// Plan waves tool
	r.RegisterTool(Tool{
		Name:        "plan_waves",
		Description: "Generate an optimal migration wave plan that respects dependencies and minimizes risk",
		Parameters: map[string]ToolParameter{
			"wave_size": {
				Type:        "number",
				Description: "Target number of repositories per wave",
				Required:    false,
			},
			"organization": {
				Type:        "string",
				Description: "Filter by organization",
				Required:    false,
			},
		},
		Execute: r.planWaves,
	})

	// Get team repositories tool
	r.RegisterTool(Tool{
		Name:        "get_team_repositories",
		Description: "Find all repositories owned by or associated with a specific team",
		Parameters: map[string]ToolParameter{
			"team": {
				Type:        "string",
				Description: "Team name in format org/team-slug",
				Required:    true,
			},
			"include_migrated": {
				Type:        "boolean",
				Description: "Include already migrated repositories",
				Required:    false,
			},
		},
		Execute: r.getTeamRepositories,
	})

	// Get migration status tool
	r.RegisterTool(Tool{
		Name:        "get_migration_status",
		Description: "Get the current migration status for specific repositories",
		Parameters: map[string]ToolParameter{
			"repositories": {
				Type:        "array",
				Description: "List of repository full names to check",
				Required:    true,
			},
		},
		Execute: r.getMigrationStatus,
	})

	// Schedule batch tool
	r.RegisterTool(Tool{
		Name:        "schedule_batch",
		Description: "Schedule a batch for migration execution at a specific date/time",
		Parameters: map[string]ToolParameter{
			"batch_name": {
				Type:        "string",
				Description: "Name of the batch to schedule",
				Required:    true,
			},
			"scheduled_at": {
				Type:        "string",
				Description: "ISO 8601 datetime for when to execute the batch",
				Required:    true,
			},
		},
		Execute: r.scheduleBatch,
	})
}

// Tool execution stubs - real implementations are in internal/mcp/handlers.go
// These stubs are kept for backward compatibility but tools should be executed
// via the MCP server for full functionality with database access.

func (r *ToolRegistry) analyzeRepositories(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "analyze_repositories")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "analyze_repositories",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) getComplexityBreakdown(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "get_complexity_breakdown")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "get_complexity_breakdown",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) checkDependencies(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "check_dependencies")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "check_dependencies",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) findPilotCandidates(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "find_pilot_candidates")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "find_pilot_candidates",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) createBatch(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "create_batch")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "create_batch",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) planWaves(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "plan_waves")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "plan_waves",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) getTeamRepositories(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "get_team_repositories")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "get_team_repositories",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) getMigrationStatus(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "get_migration_status")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "get_migration_status",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) scheduleBatch(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Debug("Tool called via legacy path - use MCP server for full functionality", "tool", "schedule_batch")
	return map[string]any{
		"message": "This tool should be executed via the MCP server for full functionality. Enable MCP in Copilot settings.",
		"tool":    "schedule_batch",
		"args":    args,
	}, nil
}
