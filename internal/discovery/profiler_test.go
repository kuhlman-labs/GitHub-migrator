package discovery

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

func TestNewProfiler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a minimal client config
	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   "test-token",
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	profiler := NewProfiler(client, logger)

	if profiler == nil {
		t.Fatal("NewProfiler returned nil")
	}
	if profiler.client == nil {
		t.Error("Profiler client is nil")
	}
	if profiler.logger == nil {
		t.Error("Profiler logger is nil")
	}
}

func TestProfileFeatures_InvalidFullName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   "test-token",
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	profiler := NewProfiler(client, logger)
	ctx := context.Background()

	// Test with invalid full_name format
	repo := &models.Repository{
		FullName: "invalid-format",
	}

	err = profiler.ProfileFeatures(ctx, repo)
	if err == nil {
		t.Error("Expected error for invalid full_name format, got nil")
	}
}

func TestProfileFeatures_Integration(t *testing.T) {
	// Skip if GITHUB_TOKEN is not set
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping integration test (set GITHUB_TOKEN to run)")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   token,
		Logger:  logger,
	}

	client, err := github.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	profiler := NewProfiler(client, logger)
	ctx := context.Background()

	// Test with a known public repository
	repo := &models.Repository{
		FullName: "octocat/Hello-World",
	}

	if err := profiler.ProfileFeatures(ctx, repo); err != nil {
		t.Fatalf("ProfileFeatures failed: %v", err)
	}

	// Verify some basic fields were populated
	// Note: These may vary based on the actual repository state
	t.Logf("Profiled repository: %s", repo.FullName)
	t.Logf("Has Wiki: %v", repo.HasWiki)
	t.Logf("Has Pages: %v", repo.HasPages)
	t.Logf("Has Actions: %v", repo.HasActions)
	t.Logf("Contributors: %d", repo.ContributorCount)
	t.Logf("Issues: %d (open: %d)", repo.IssueCount, repo.OpenIssueCount)
	t.Logf("Pull Requests: %d (open: %d)", repo.PullRequestCount, repo.OpenPRCount)
	t.Logf("Tags: %d", repo.TagCount)
}
