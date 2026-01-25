package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// setupTestDB creates a test database with some sample data
func setupTestDB(t *testing.T) *storage.Database {
	t.Helper()

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// createTestRepositories creates sample repositories for testing
func createTestRepositories(t *testing.T, db *storage.Database, count int) []*models.Repository {
	t.Helper()

	repos := make([]*models.Repository, count)
	for i := 0; i < count; i++ {
		complexityScore := (i % 20) + 1 // Complexity 1-20
		size := int64((i + 1) * 1000)   // Size in bytes

		repo := &models.Repository{
			FullName:     "test-org/repo-" + string(rune('a'+i)),
			Source:       "github",
			SourceURL:    "https://github.example.com",
			Status:       "pending",
			Visibility:   "private",
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
			GitProperties: &models.RepositoryGitProperties{
				TotalSize: &size,
			},
			Validation: &models.RepositoryValidation{
				ComplexityScore: &complexityScore,
			},
		}

		if err := db.SaveRepository(context.Background(), repo); err != nil {
			t.Fatalf("Failed to create test repository: %v", err)
		}
		repos[i] = repo
	}

	return repos
}

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db := setupTestDB(t)

	server := NewServer(db, logger, Config{
		Address: ":8081",
	})

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.Address() != ":8081" {
		t.Errorf("Expected address :8081, got %s", server.Address())
	}

	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}
}

func TestRepoToSummary(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db := setupTestDB(t)
	server := NewServer(db, logger, Config{Address: ":8081"})

	size := int64(1024000) // 1 MB
	complexity := 5

	repo := &models.Repository{
		FullName:   "test-org/test-repo",
		Status:     "pending",
		IsArchived: false,
		IsFork:     false,
		UpdatedAt:  time.Now(),
		GitProperties: &models.RepositoryGitProperties{
			TotalSize: &size,
		},
		Validation: &models.RepositoryValidation{
			ComplexityScore: &complexity,
		},
	}

	summary := server.repoToSummary(repo)

	if summary.FullName != "test-org/test-repo" {
		t.Errorf("Expected full name 'test-org/test-repo', got '%s'", summary.FullName)
	}

	if summary.Organization != "test-org" {
		t.Errorf("Expected organization 'test-org', got '%s'", summary.Organization)
	}

	if summary.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", summary.Status)
	}

	if summary.Size != 1000 { // 1024000 / 1024 = 1000 KB
		t.Errorf("Expected size 1000 KB, got %d KB", summary.Size)
	}

	if summary.ComplexityScore != 5 {
		t.Errorf("Expected complexity 5, got %d", summary.ComplexityScore)
	}

	if summary.ComplexityRating != "simple" {
		t.Errorf("Expected rating 'simple', got '%s'", summary.ComplexityRating)
	}
}

func TestGetComplexityRating(t *testing.T) {
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

	for _, tc := range tests {
		result := getComplexityRating(tc.score)
		if result != tc.expected {
			t.Errorf("getComplexityRating(%d) = %s, expected %s", tc.score, result, tc.expected)
		}
	}
}

func TestHandlerTypes(t *testing.T) {
	// Test that output types serialize correctly to JSON
	t.Run("AnalyzeRepositoriesOutput", func(t *testing.T) {
		output := AnalyzeRepositoriesOutput{
			Repositories: []RepositorySummary{
				{FullName: "org/repo1", Status: "pending", ComplexityScore: 5},
			},
			TotalCount: 1,
			Message:    "Found 1 repository",
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded AnalyzeRepositoriesOutput
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.TotalCount != 1 {
			t.Errorf("Expected TotalCount 1, got %d", decoded.TotalCount)
		}
	})

	t.Run("ComplexityBreakdown", func(t *testing.T) {
		output := GetComplexityBreakdownOutput{
			Repository: "org/repo",
			Breakdown: ComplexityBreakdown{
				TotalScore: 15,
				Rating:     "complex",
				Components: map[string]int{
					"size":     3,
					"features": 5,
					"activity": 7,
				},
				Blockers:        []string{"Has blocking files"},
				Warnings:        []string{"Large files detected"},
				Recommendations: []string{"Run dry-run first"},
			},
			Message: "Breakdown complete",
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		if len(data) == 0 {
			t.Error("Expected non-empty JSON")
		}
	})

	t.Run("WavePlan", func(t *testing.T) {
		output := PlanWavesOutput{
			Waves: []WavePlan{
				{
					WaveNumber: 1,
					Repositories: []RepositorySummary{
						{FullName: "org/repo1"},
						{FullName: "org/repo2"},
					},
					TotalSize:     5000,
					AvgComplexity: 3.5,
					Dependencies:  2,
				},
			},
			TotalWaves:        1,
			TotalRepositories: 2,
			Message:           "Planned 1 wave",
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded PlanWavesOutput
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.TotalWaves != 1 {
			t.Errorf("Expected 1 wave, got %d", decoded.TotalWaves)
		}

		if len(decoded.Waves[0].Repositories) != 2 {
			t.Errorf("Expected 2 repos in wave, got %d", len(decoded.Waves[0].Repositories))
		}
	})
}

func TestInputTypes(t *testing.T) {
	// Test that input types deserialize correctly
	t.Run("AnalyzeRepositoriesInput", func(t *testing.T) {
		jsonData := `{
			"organization": "test-org",
			"status": "pending",
			"max_complexity": 10,
			"limit": 50
		}`

		var input AnalyzeRepositoriesInput
		if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if input.Organization != "test-org" {
			t.Errorf("Expected org 'test-org', got '%s'", input.Organization)
		}

		if input.MaxComplexity != 10 {
			t.Errorf("Expected max_complexity 10, got %d", input.MaxComplexity)
		}

		if input.Limit != 50 {
			t.Errorf("Expected limit 50, got %d", input.Limit)
		}
	})

	t.Run("CreateBatchInput", func(t *testing.T) {
		jsonData := `{
			"name": "pilot-batch",
			"description": "First migration batch",
			"repositories": ["org/repo1", "org/repo2", "org/repo3"]
		}`

		var input CreateBatchInput
		if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if input.Name != "pilot-batch" {
			t.Errorf("Expected name 'pilot-batch', got '%s'", input.Name)
		}

		if len(input.Repositories) != 3 {
			t.Errorf("Expected 3 repositories, got %d", len(input.Repositories))
		}
	})
}
