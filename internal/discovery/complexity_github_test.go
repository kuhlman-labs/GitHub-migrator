package discovery

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestCalculateSizePoints(t *testing.T) {
	tests := []struct {
		name     string
		size     *int64
		expected int
	}{
		{
			name:     "nil size",
			size:     nil,
			expected: 0,
		},
		{
			name:     "zero size",
			size:     int64Ptr(0),
			expected: 0,
		},
		{
			name:     "small size (< 100MB)",
			size:     int64Ptr(50 * 1024 * 1024), // 50MB
			expected: 0,
		},
		{
			name:     "medium size (100MB - 1GB)",
			size:     int64Ptr(500 * 1024 * 1024), // 500MB
			expected: 3,
		},
		{
			name:     "large size (1GB - 5GB)",
			size:     int64Ptr(2 * 1024 * 1024 * 1024), // 2GB
			expected: 6,
		},
		{
			name:     "very large size (> 5GB)",
			size:     int64Ptr(10 * 1024 * 1024 * 1024), // 10GB
			expected: 9,
		},
		{
			name:     "exactly 100MB",
			size:     int64Ptr(104857600),
			expected: 3,
		},
		{
			name:     "exactly 1GB",
			size:     int64Ptr(1073741824),
			expected: 6,
		},
		{
			name:     "exactly 5GB",
			size:     int64Ptr(5368709120),
			expected: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSizePoints(tt.size)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCalculateActivityPoints(t *testing.T) {
	tests := []struct {
		name           string
		commitCount    int
		openIssueCount int
		expected       int
	}{
		{
			name:           "low activity",
			commitCount:    10,
			openIssueCount: 5,
			expected:       0, // 10 + (5*2) = 20
		},
		{
			name:           "medium activity",
			commitCount:    100,
			openIssueCount: 10,
			expected:       2, // 100 + (10*2) = 120
		},
		{
			name:           "high activity",
			commitCount:    500,
			openIssueCount: 300,
			expected:       4, // 500 + (300*2) = 1100
		},
		{
			name:           "zero activity",
			commitCount:    0,
			openIssueCount: 0,
			expected:       0,
		},
		{
			name:           "boundary just below 100",
			commitCount:    99,
			openIssueCount: 0,
			expected:       0,
		},
		{
			name:           "boundary at 101",
			commitCount:    101,
			openIssueCount: 0,
			expected:       2,
		},
		{
			name:           "boundary at 1001",
			commitCount:    1001,
			openIssueCount: 0,
			expected:       4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &models.Repository{
				CommitCount:    tt.commitCount,
				OpenIssueCount: tt.openIssueCount,
			}
			result := calculateActivityPoints(repo)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

//nolint:gocyclo // Table-driven test with multiple assertions and setup
func TestCalculateComplexity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a client for the profiler (it won't be used for complexity calculations)
	client, err := github.NewClient(github.ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	profiler := NewProfiler(client, logger)

	t.Run("simple repository with no features", func(t *testing.T) {
		repo := &models.Repository{
			FullName: "test/repo",
		}

		complexity, breakdown := profiler.CalculateComplexity(repo)

		if complexity != 0 {
			t.Errorf("Expected complexity 0, got %d", complexity)
		}
		if breakdown == nil {
			t.Fatal("Expected non-nil breakdown")
			return // Explicitly unreachable, but satisfies static analysis
		}
		if breakdown.SizePoints != 0 {
			t.Errorf("Expected SizePoints 0, got %d", breakdown.SizePoints)
		}
	})

	t.Run("repository with large files", func(t *testing.T) {
		repo := &models.Repository{
			FullName:      "test/repo",
			HasLargeFiles: true,
		}

		complexity, breakdown := profiler.CalculateComplexity(repo)

		if breakdown.LargeFilesPoints != 4 {
			t.Errorf("Expected LargeFilesPoints 4, got %d", breakdown.LargeFilesPoints)
		}
		if complexity < 4 {
			t.Errorf("Expected complexity >= 4, got %d", complexity)
		}
	})

	t.Run("repository with environments and secrets", func(t *testing.T) {
		repo := &models.Repository{
			FullName:         "test/repo",
			EnvironmentCount: 3,
			SecretCount:      5,
		}

		complexity, breakdown := profiler.CalculateComplexity(repo)

		if breakdown.EnvironmentsPoints != 3 {
			t.Errorf("Expected EnvironmentsPoints 3, got %d", breakdown.EnvironmentsPoints)
		}
		if breakdown.SecretsPoints != 3 {
			t.Errorf("Expected SecretsPoints 3, got %d", breakdown.SecretsPoints)
		}
		if complexity < 6 {
			t.Errorf("Expected complexity >= 6, got %d", complexity)
		}
	})

	t.Run("repository with packages and self-hosted runners", func(t *testing.T) {
		repo := &models.Repository{
			FullName:             "test/repo",
			HasPackages:          true,
			HasSelfHostedRunners: true,
		}

		_, breakdown := profiler.CalculateComplexity(repo)

		if breakdown.PackagesPoints != 3 {
			t.Errorf("Expected PackagesPoints 3, got %d", breakdown.PackagesPoints)
		}
		if breakdown.RunnersPoints != 3 {
			t.Errorf("Expected RunnersPoints 3, got %d", breakdown.RunnersPoints)
		}
	})

	t.Run("repository with moderate impact features", func(t *testing.T) {
		repo := &models.Repository{
			FullName:           "test/repo",
			VariableCount:      2,
			HasDiscussions:     true,
			ReleaseCount:       10,
			HasLFS:             true,
			HasSubmodules:      true,
			InstalledAppsCount: 2,
			HasProjects:        true,
		}

		_, breakdown := profiler.CalculateComplexity(repo)

		if breakdown.VariablesPoints != 2 {
			t.Errorf("Expected VariablesPoints 2, got %d", breakdown.VariablesPoints)
		}
		if breakdown.DiscussionsPoints != 2 {
			t.Errorf("Expected DiscussionsPoints 2, got %d", breakdown.DiscussionsPoints)
		}
		if breakdown.ReleasesPoints != 2 {
			t.Errorf("Expected ReleasesPoints 2, got %d", breakdown.ReleasesPoints)
		}
		if breakdown.LFSPoints != 2 {
			t.Errorf("Expected LFSPoints 2, got %d", breakdown.LFSPoints)
		}
		if breakdown.SubmodulesPoints != 2 {
			t.Errorf("Expected SubmodulesPoints 2, got %d", breakdown.SubmodulesPoints)
		}
		if breakdown.AppsPoints != 2 {
			t.Errorf("Expected AppsPoints 2, got %d", breakdown.AppsPoints)
		}
		if breakdown.ProjectsPoints != 2 {
			t.Errorf("Expected ProjectsPoints 2, got %d", breakdown.ProjectsPoints)
		}
	})

	t.Run("repository with low impact features", func(t *testing.T) {
		repo := &models.Repository{
			FullName:          "test/repo",
			HasCodeScanning:   true,
			HasDependabot:     true,
			HasSecretScanning: true,
			WebhookCount:      5,
			BranchProtections: 2,
			HasRulesets:       true,
			Visibility:        "public",
			HasCodeowners:     true,
		}

		_, breakdown := profiler.CalculateComplexity(repo)

		if breakdown.SecurityPoints != 1 {
			t.Errorf("Expected SecurityPoints 1, got %d", breakdown.SecurityPoints)
		}
		if breakdown.WebhooksPoints != 1 {
			t.Errorf("Expected WebhooksPoints 1, got %d", breakdown.WebhooksPoints)
		}
		if breakdown.BranchProtectionsPoints != 1 {
			t.Errorf("Expected BranchProtectionsPoints 1, got %d", breakdown.BranchProtectionsPoints)
		}
		if breakdown.RulesetsPoints != 1 {
			t.Errorf("Expected RulesetsPoints 1, got %d", breakdown.RulesetsPoints)
		}
		if breakdown.PublicVisibilityPoints != 1 {
			t.Errorf("Expected PublicVisibilityPoints 1, got %d", breakdown.PublicVisibilityPoints)
		}
		if breakdown.CodeownersPoints != 1 {
			t.Errorf("Expected CodeownersPoints 1, got %d", breakdown.CodeownersPoints)
		}
	})

	t.Run("repository with internal visibility", func(t *testing.T) {
		repo := &models.Repository{
			FullName:   "test/repo",
			Visibility: "internal",
		}

		_, breakdown := profiler.CalculateComplexity(repo)

		if breakdown.InternalVisibilityPoints != 1 {
			t.Errorf("Expected InternalVisibilityPoints 1, got %d", breakdown.InternalVisibilityPoints)
		}
		if breakdown.PublicVisibilityPoints != 0 {
			t.Errorf("Expected PublicVisibilityPoints 0, got %d", breakdown.PublicVisibilityPoints)
		}
	})

	t.Run("complex repository with many features", func(t *testing.T) {
		size := int64(3 * 1024 * 1024 * 1024) // 3GB
		repo := &models.Repository{
			FullName:             "test/repo",
			TotalSize:            &size,
			HasLargeFiles:        true,
			EnvironmentCount:     3,
			SecretCount:          5,
			HasPackages:          true,
			HasSelfHostedRunners: true,
			VariableCount:        2,
			HasDiscussions:       true,
			ReleaseCount:         10,
			HasLFS:               true,
			HasSubmodules:        true,
			HasCodeScanning:      true,
			WebhookCount:         5,
			BranchProtections:    2,
			Visibility:           "public",
			CommitCount:          1500,
			OpenIssueCount:       50,
		}

		complexity, breakdown := profiler.CalculateComplexity(repo)

		// Size: 6 points (1-5GB tier)
		// Large files: 4 points
		// High impact: 12 points (envs, secrets, packages, runners)
		// Moderate: 10 points (vars, discussions, releases, LFS, submodules)
		// Low: 4 points (security, webhooks, branch protections, public)
		// Activity: 4 points (high activity)
		// Total: 6 + 4 + 12 + 10 + 4 + 4 = 40

		if breakdown.SizePoints != 6 {
			t.Errorf("Expected SizePoints 6, got %d", breakdown.SizePoints)
		}
		if breakdown.ActivityPoints != 4 {
			t.Errorf("Expected ActivityPoints 4, got %d", breakdown.ActivityPoints)
		}
		if complexity < 30 {
			t.Errorf("Expected high complexity >= 30, got %d", complexity)
		}
	})
}

// Helper function
func int64Ptr(v int64) *int64 {
	return &v
}
