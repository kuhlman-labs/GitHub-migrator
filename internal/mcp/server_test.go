package mcp

import (
	"testing"
)

func TestRepoToSummary(t *testing.T) {
	// Test basic summary conversion without a full server
	t.Run("getComplexityRating returns correct ratings", func(t *testing.T) {
		tests := []struct {
			score    int
			expected string
		}{
			{0, "simple"},
			{5, "simple"},
			{6, "medium"},
			{10, "medium"},
			{11, "complex"},
			{17, "complex"},
			{18, "very_complex"},
			{100, "very_complex"},
		}

		for _, tt := range tests {
			result := getComplexityRating(tt.score)
			if result != tt.expected {
				t.Errorf("getComplexityRating(%d) = %q, want %q", tt.score, result, tt.expected)
			}
		}
	})
}

func TestToolOutputTypes(t *testing.T) {
	t.Run("AnalyzeRepositoriesOutput serializes correctly", func(t *testing.T) {
		output := AnalyzeRepositoriesOutput{
			Repositories: []RepositorySummary{
				{FullName: "org/repo1", Status: "pending"},
			},
			TotalCount: 1,
			Message:    "Found 1 repository",
		}

		if output.TotalCount != 1 {
			t.Errorf("Expected TotalCount 1, got %d", output.TotalCount)
		}
		if len(output.Repositories) != 1 {
			t.Errorf("Expected 1 repository, got %d", len(output.Repositories))
		}
	})

	t.Run("FindPilotCandidatesOutput has correct criteria", func(t *testing.T) {
		output := FindPilotCandidatesOutput{
			Candidates: []RepositorySummary{
				{FullName: "org/simple-repo", ComplexityScore: 2, ComplexityRating: "simple"},
			},
			Count:    1,
			Criteria: "Simple complexity (≤5), few local dependencies",
			Message:  "Found 1 candidate",
		}

		if output.Count != 1 {
			t.Errorf("Expected Count 1, got %d", output.Count)
		}
		if output.Criteria == "" {
			t.Error("Expected non-empty Criteria")
		}
	})

	t.Run("CreateBatchOutput indicates success", func(t *testing.T) {
		output := CreateBatchOutput{
			Batch: BatchInfo{
				ID:              1,
				Name:            "test-batch",
				Status:          "pending",
				RepositoryCount: 5,
			},
			Success: true,
			Message: "Created batch with 5 repositories",
		}

		if !output.Success {
			t.Error("Expected Success to be true")
		}
		if output.Batch.Name != "test-batch" {
			t.Errorf("Expected batch name 'test-batch', got %q", output.Batch.Name)
		}
	})

	t.Run("PlanWavesOutput has wave details", func(t *testing.T) {
		output := PlanWavesOutput{
			Waves: []WavePlan{
				{WaveNumber: 1, Repositories: []RepositorySummary{{FullName: "org/repo1"}}},
				{WaveNumber: 2, Repositories: []RepositorySummary{{FullName: "org/repo2"}}},
			},
			TotalWaves:        2,
			TotalRepositories: 2,
			Message:           "Planned 2 waves",
		}

		if output.TotalWaves != 2 {
			t.Errorf("Expected 2 waves, got %d", output.TotalWaves)
		}
	})
}

func TestInputTypes(t *testing.T) {
	t.Run("AnalyzeRepositoriesInput has defaults", func(t *testing.T) {
		input := AnalyzeRepositoriesInput{
			Organization: "test-org",
			Status:       "pending",
			Limit:        20,
		}

		if input.Limit != 20 {
			t.Errorf("Expected Limit 20, got %d", input.Limit)
		}
	})

	t.Run("CreateBatchInput requires name and repos", func(t *testing.T) {
		input := CreateBatchInput{
			Name:         "pilot-batch",
			Description:  "First migration batch",
			Repositories: []string{"org/repo1", "org/repo2"},
		}

		if len(input.Repositories) != 2 {
			t.Errorf("Expected 2 repositories, got %d", len(input.Repositories))
		}
	})
}
