package copilot

import (
	"context"
	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Tool represents a custom tool that Copilot can use
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

// Tool implementation stubs - these will be fully implemented in the tools package
func (r *ToolRegistry) analyzeRepositories(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing analyze_repositories tool", "args", args)
	// TODO: Implement with actual database queries
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) getComplexityBreakdown(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing get_complexity_breakdown tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) checkDependencies(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing check_dependencies tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) findPilotCandidates(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing find_pilot_candidates tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) createBatch(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing create_batch tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) planWaves(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing plan_waves tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) getTeamRepositories(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing get_team_repositories tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) getMigrationStatus(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing get_migration_status tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}

func (r *ToolRegistry) scheduleBatch(ctx context.Context, args map[string]any) (any, error) {
	r.logger.Info("Executing schedule_batch tool", "args", args)
	return map[string]any{
		"message": "Tool execution pending full implementation",
		"args":    args,
	}, nil
}
