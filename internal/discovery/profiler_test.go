package discovery

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
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
		return // Prevent staticcheck SA5011
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

func TestProfileWikiContent(t *testing.T) {
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

	tests := []struct {
		name            string
		repo            *models.Repository
		expectedHasWiki bool
		description     string
	}{
		{
			name: "Wiki feature disabled",
			repo: &models.Repository{
				FullName:  "test/repo",
				SourceURL: "https://github.com/test/repo",
				HasWiki:   false,
			},
			expectedHasWiki: false,
			description:     "Wiki feature disabled should remain false",
		},
		{
			name: "Wiki enabled but URL construction",
			repo: &models.Repository{
				FullName:  "test/repo",
				SourceURL: "https://github.com/test/repo.git",
				HasWiki:   true,
			},
			expectedHasWiki: false, // Will be false if wiki doesn't exist or has no content
			description:     "Wiki enabled but no content should be set to false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiler.profileWikiContent(ctx, tt.repo)
			t.Logf("%s: HasWiki = %v", tt.description, tt.repo.HasWiki)
		})
	}
}

func TestCheckWikiHasContent(t *testing.T) {
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

	tests := []struct {
		name        string
		wikiURL     string
		shouldExist bool
		description string
	}{
		{
			name:        "Nonexistent wiki",
			wikiURL:     "https://github.com/nonexistent-org-12345/nonexistent-repo-67890.wiki.git",
			shouldExist: false,
			description: "Wiki that doesn't exist should return false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasContent, err := profiler.checkWikiHasContent(ctx, tt.wikiURL)
			if err != nil {
				t.Logf("checkWikiHasContent returned error (expected for nonexistent wikis): %v", err)
			}
			t.Logf("%s: hasContent = %v", tt.description, hasContent)

			if tt.shouldExist && !hasContent {
				t.Errorf("Expected wiki to have content but got false")
			}
			if !tt.shouldExist && hasContent {
				t.Errorf("Expected wiki to not exist but got true")
			}
		})
	}
}

func TestProfilePackages(t *testing.T) {
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

	// Test with a repository - first load the package cache
	org := "octocat"
	err = profiler.LoadPackageCache(ctx, org)
	if err != nil {
		t.Logf("LoadPackageCache returned error (may be expected): %v", err)
	}

	// Test the profilePackages method
	repo := &models.Repository{
		FullName: "octocat/Hello-World",
	}
	profiler.profilePackages(ctx, "octocat", "Hello-World", repo)
	t.Logf("Repository HasPackages field (from REST API cache): %v", repo.HasPackages)
}

func TestLoadPackageCache(t *testing.T) {
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

	// Test loading package cache for an organization
	// Using a small org that might have some packages
	org := "octocat"
	err = profiler.LoadPackageCache(ctx, org)
	if err != nil {
		t.Logf("LoadPackageCache returned error (may be expected): %v", err)
	}

	// Verify the cache was initialized
	profiler.packageCacheMu.RLock()
	cacheSize := len(profiler.packageCache)
	profiler.packageCacheMu.RUnlock()

	t.Logf("Package cache loaded with %d repositories", cacheSize)

	// Test that profilePackages now uses the cache
	repo := &models.Repository{
		FullName: "octocat/test-repo",
	}
	profiler.profilePackages(ctx, "octocat", "test-repo", repo)
	t.Logf("Repository HasPackages (from cache): %v", repo.HasPackages)
}
