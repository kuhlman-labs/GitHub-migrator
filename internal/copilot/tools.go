// Package copilot provides the Copilot chat service integration.
package copilot

import (
	"log/slog"
)

// ToolDescription represents a tool that can be used by the AI assistant.
// This is used to build the system prompt with available tool descriptions.
// Actual tool execution is handled by the MCP server in internal/mcp/.
type ToolDescription struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToolRegistry provides tool descriptions for the system prompt.
// Tool execution is handled by the MCP server.
type ToolRegistry struct {
	tools  []ToolDescription
	logger *slog.Logger
}

// NewToolRegistry creates a new tool registry with default tool descriptions
func NewToolRegistry(logger *slog.Logger) *ToolRegistry {
	registry := &ToolRegistry{
		tools:  make([]ToolDescription, 0),
		logger: logger,
	}

	// Register tool descriptions (execution is via MCP server)
	registry.registerToolDescriptions()

	return registry
}

// GetTools returns all tool descriptions for building the system prompt
func (r *ToolRegistry) GetTools() []ToolDescription {
	return r.tools
}

// registerToolDescriptions adds descriptions for all available migration tools
func (r *ToolRegistry) registerToolDescriptions() {
	r.tools = []ToolDescription{
		{
			Name:        "analyze_repositories",
			Description: "Query and analyze repositories based on various criteria like complexity, migration status, features, and organization. Returns a list of repositories matching the filters.",
		},
		{
			Name:        "get_complexity_breakdown",
			Description: "Get detailed complexity breakdown for a specific repository including size, features, dependencies, and activity scores.",
		},
		{
			Name:        "check_dependencies",
			Description: "Find all dependencies for a repository and check their migration status. Can also show reverse dependencies (repos that depend on this one).",
		},
		{
			Name:        "find_pilot_candidates",
			Description: "Identify repositories suitable for a pilot migration based on low complexity, few dependencies, and low risk.",
		},
		{
			Name:        "create_batch",
			Description: "Create a new migration batch with specified repositories. Batches group repositories for coordinated migration.",
		},
		{
			Name:        "plan_waves",
			Description: "Generate an optimal migration wave plan that respects dependencies and minimizes risk. Waves are ordered groups of batches.",
		},
		{
			Name:        "get_team_repositories",
			Description: "Find all repositories owned by or associated with a specific team.",
		},
		{
			Name:        "get_migration_status",
			Description: "Get the current migration status for specific repositories including progress and any errors.",
		},
		{
			Name:        "schedule_batch",
			Description: "Schedule a batch for migration execution at a specific date/time.",
		},
	}

	for _, tool := range r.tools {
		r.logger.Debug("Registered tool description", "name", tool.Name)
	}
}
