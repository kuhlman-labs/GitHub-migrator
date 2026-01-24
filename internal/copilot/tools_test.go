package copilot

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("NewToolRegistry creates registry with default tools", func(t *testing.T) {
		registry := NewToolRegistry(nil, logger)

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
			"plan_waves",
			"get_team_repositories",
			"get_migration_status",
			"schedule_batch",
		}

		for _, name := range expectedTools {
			if _, ok := registry.GetTool(name); !ok {
				t.Errorf("expected tool %q to be registered", name)
			}
		}
	})

	t.Run("RegisterTool adds custom tool", func(t *testing.T) {
		registry := NewToolRegistry(nil, logger)

		customTool := Tool{
			Name:        "custom_tool",
			Description: "A custom tool for testing",
			Parameters: map[string]ToolParameter{
				"param1": {
					Type:        "string",
					Description: "A test parameter",
					Required:    true,
				},
			},
			Execute: func(ctx context.Context, args map[string]any) (any, error) {
				return "custom result", nil
			},
		}

		registry.RegisterTool(customTool)

		tool, ok := registry.GetTool("custom_tool")
		if !ok {
			t.Error("expected custom tool to be registered")
		}
		if tool.Description != "A custom tool for testing" {
			t.Errorf("unexpected description: %s", tool.Description)
		}
	})

	t.Run("GetTool returns false for non-existent tool", func(t *testing.T) {
		registry := NewToolRegistry(nil, logger)

		_, ok := registry.GetTool("non_existent_tool")
		if ok {
			t.Error("expected GetTool to return false for non-existent tool")
		}
	})

	t.Run("ExecuteTool returns error for non-existent tool", func(t *testing.T) {
		registry := NewToolRegistry(nil, logger)

		_, err := registry.ExecuteTool(context.Background(), "non_existent_tool", nil)
		if err == nil {
			t.Error("expected error for non-existent tool")
		}

		// Check it's the right type of error
		if _, ok := err.(*ToolNotFoundError); !ok {
			t.Errorf("expected ToolNotFoundError, got %T", err)
		}
	})

	t.Run("ExecuteTool executes registered tool", func(t *testing.T) {
		registry := NewToolRegistry(nil, logger)

		customTool := Tool{
			Name:        "test_execute",
			Description: "Test execution",
			Parameters:  map[string]ToolParameter{},
			Execute: func(ctx context.Context, args map[string]any) (any, error) {
				return map[string]string{"result": "success"}, nil
			},
		}

		registry.RegisterTool(customTool)

		result, err := registry.ExecuteTool(context.Background(), "test_execute", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultMap, ok := result.(map[string]string)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if resultMap["result"] != "success" {
			t.Errorf("unexpected result: %v", resultMap)
		}
	})
}

func TestToolNotFoundError(t *testing.T) {
	err := &ToolNotFoundError{Name: "missing_tool"}
	expected := "tool not found: missing_tool"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestToolParameterTypes(t *testing.T) {
	// Test that tool parameters have correct types
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	registry := NewToolRegistry(nil, logger)

	tool, ok := registry.GetTool("analyze_repositories")
	if !ok {
		t.Fatal("expected analyze_repositories tool")
	}

	// Check parameter definitions
	if param, exists := tool.Parameters["organization"]; exists {
		if param.Type != "string" {
			t.Errorf("expected organization to be string, got %s", param.Type)
		}
		if param.Required {
			t.Error("expected organization to not be required")
		}
	} else {
		t.Error("expected organization parameter")
	}

	if param, exists := tool.Parameters["max_complexity"]; exists {
		if param.Type != "number" {
			t.Errorf("expected max_complexity to be number, got %s", param.Type)
		}
	} else {
		t.Error("expected max_complexity parameter")
	}
}
