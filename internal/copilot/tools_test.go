package copilot

import (
	"log/slog"
	"os"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("NewToolRegistry creates registry with default tools", func(t *testing.T) {
		registry := NewToolRegistry(logger)

		tools := registry.GetTools()
		if len(tools) == 0 {
			t.Error("expected default tools to be registered")
		}

		// Check for expected tools
		expectedTools := []string{
			"analyze_repositories",
			"get_complexity_breakdown",
			"check_dependencies",
			"find_pilot_candidates",
			"create_batch",
			"configure_batch",
			"plan_waves",
			"get_team_repositories",
			"get_migration_status",
			"schedule_batch",
			"start_migration",
			"cancel_migration",
			"get_migration_progress",
		}

		// Create a map for quick lookup
		toolMap := make(map[string]bool)
		for _, tool := range tools {
			toolMap[tool.Name] = true
		}

		for _, name := range expectedTools {
			if !toolMap[name] {
				t.Errorf("expected tool %q to be registered", name)
			}
		}
	})

	t.Run("GetTools returns correct number of tools", func(t *testing.T) {
		registry := NewToolRegistry(logger)
		tools := registry.GetTools()

		// We expect 13 tools (including migration execution tools)
		if len(tools) != 13 {
			t.Errorf("expected 13 tools, got %d", len(tools))
		}
	})

	t.Run("All tools have descriptions", func(t *testing.T) {
		registry := NewToolRegistry(logger)
		tools := registry.GetTools()

		for _, tool := range tools {
			if tool.Name == "" {
				t.Error("tool has empty name")
			}
			if tool.Description == "" {
				t.Errorf("tool %q has empty description", tool.Name)
			}
		}
	})
}
