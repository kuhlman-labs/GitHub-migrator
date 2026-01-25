package copilot

import (
	"context"
	"testing"
)

func TestGetComplexityRating(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected string
	}{
		{
			name:     "zero is simple",
			score:    0,
			expected: "simple",
		},
		{
			name:     "low score is simple",
			score:    3,
			expected: "simple",
		},
		{
			name:     "score 5 is simple",
			score:    5,
			expected: "simple",
		},
		{
			name:     "score 6 is medium",
			score:    6,
			expected: "medium",
		},
		{
			name:     "score 10 is medium",
			score:    10,
			expected: "medium",
		},
		{
			name:     "score 11 is complex",
			score:    11,
			expected: "complex",
		},
		{
			name:     "score 17 is complex",
			score:    17,
			expected: "complex",
		},
		{
			name:     "score 18 is very_complex",
			score:    18,
			expected: "very_complex",
		},
		{
			name:     "high score is very_complex",
			score:    100,
			expected: "very_complex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getComplexityRating(tt.score)
			if result != tt.expected {
				t.Errorf("Expected %q for score %d, got %q", tt.expected, tt.score, result)
			}
		})
	}
}

func TestToolExecutionResult_Structure(t *testing.T) {
	// Test that ToolExecutionResult can be properly created and used
	result := &ToolExecutionResult{
		Tool:    "find_pilot_candidates",
		Success: true,
		Result: []map[string]any{
			{"full_name": "org/repo1", "complexity_score": 2},
			{"full_name": "org/repo2", "complexity_score": 3},
		},
		Summary: "Found 2 repositories suitable for pilot migration",
		Suggestions: []string{
			"These repositories have low complexity",
		},
		FollowUp: &FollowUpAction{
			Action:       "create_batch",
			Description:  "Create a batch with these 2 pilot repositories?",
			Repositories: []string{"org/repo1", "org/repo2"},
			DefaultName:  "pilot-wave-1",
		},
	}

	if result.Tool != ToolFindPilotCandidates {
		t.Errorf("Expected tool %q, got %q", ToolFindPilotCandidates, result.Tool)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.FollowUp == nil {
		t.Fatal("Expected FollowUp to be set")
	}

	if result.FollowUp.Action != ToolCreateBatch {
		t.Errorf("Expected FollowUp.Action %q, got %q", ToolCreateBatch, result.FollowUp.Action)
	}

	if len(result.FollowUp.Repositories) != 2 {
		t.Errorf("Expected 2 repositories in FollowUp, got %d", len(result.FollowUp.Repositories))
	}
}

func TestFollowUpAction_Structure(t *testing.T) {
	action := &FollowUpAction{
		Action:       "schedule_batch",
		Description:  "Would you like to schedule batch 'test-batch' for migration?",
		DefaultName:  "test-batch",
		Repositories: nil,
	}

	if action.Action != "schedule_batch" {
		t.Errorf("Expected action 'schedule_batch', got %q", action.Action)
	}

	if action.DefaultName != "test-batch" {
		t.Errorf("Expected DefaultName 'test-batch', got %q", action.DefaultName)
	}
}

func TestNewToolExecutor(t *testing.T) {
	// Test that NewToolExecutor doesn't panic with nil inputs
	// This is a basic smoke test - full integration tests would use a real DB
	executor := NewToolExecutor(nil, nil)
	if executor == nil {
		t.Error("Expected non-nil executor")
	}
}

func TestToolExecutor_ExecuteTool_NilIntent(t *testing.T) {
	executor := NewToolExecutor(nil, nil)
	ctx := context.Background()

	_, err := executor.ExecuteTool(ctx, nil, nil)
	if err == nil {
		t.Error("Expected error for nil intent")
	}

	expectedErr := "no intent provided"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestToolExecutor_ExecuteTool_UnknownTool(t *testing.T) {
	executor := NewToolExecutor(nil, nil)
	ctx := context.Background()

	intent := &DetectedIntent{
		Tool:       "unknown_tool",
		Args:       map[string]any{},
		Confidence: 0.9,
	}

	_, err := executor.ExecuteTool(ctx, intent, nil)
	if err == nil {
		t.Error("Expected error for unknown tool")
	}

	expectedErr := "unknown tool: unknown_tool"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}
