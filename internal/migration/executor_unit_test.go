package migration

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// TestCalculateAdaptivePollInterval tests the adaptive polling interval calculation
func TestCalculateAdaptivePollInterval(t *testing.T) {
	tests := []struct {
		name              string
		elapsed           time.Duration
		initial           time.Duration
		max               time.Duration
		fastPhaseDuration time.Duration
		expectLessThan    time.Duration
		expectGreaterThan time.Duration
	}{
		{
			name:              "during fast phase - returns initial",
			elapsed:           5 * time.Minute,
			initial:           30 * time.Second,
			max:               5 * time.Minute,
			fastPhaseDuration: 10 * time.Minute,
			expectLessThan:    31 * time.Second,
			expectGreaterThan: 29 * time.Second,
		},
		{
			name:              "at start of fast phase - returns initial",
			elapsed:           0,
			initial:           30 * time.Second,
			max:               5 * time.Minute,
			fastPhaseDuration: 10 * time.Minute,
			expectLessThan:    31 * time.Second,
			expectGreaterThan: 29 * time.Second,
		},
		{
			name:              "after fast phase - starts backoff",
			elapsed:           15 * time.Minute,
			initial:           30 * time.Second,
			max:               5 * time.Minute,
			fastPhaseDuration: 10 * time.Minute,
			expectLessThan:    5*time.Minute + 1*time.Second, // Allow hitting max exactly
			expectGreaterThan: 30 * time.Second,
		},
		{
			name:              "long elapsed time - capped at max",
			elapsed:           2 * time.Hour,
			initial:           30 * time.Second,
			max:               5 * time.Minute,
			fastPhaseDuration: 10 * time.Minute,
			expectLessThan:    5*time.Minute + 1*time.Second,
			expectGreaterThan: 4*time.Minute + 59*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAdaptivePollInterval(tt.elapsed, tt.initial, tt.max, tt.fastPhaseDuration)

			if result >= tt.expectLessThan {
				t.Errorf("Expected result < %v, got %v", tt.expectLessThan, result)
			}
			if result <= tt.expectGreaterThan {
				t.Errorf("Expected result > %v, got %v", tt.expectGreaterThan, result)
			}
		})
	}
}

// TestSanitizeRepoName tests repository name sanitization
func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no spaces",
			input:    "my-repo",
			expected: "my-repo",
		},
		{
			name:     "single space",
			input:    "my repo",
			expected: "my-repo",
		},
		{
			name:     "multiple spaces",
			input:    "my awesome repo name",
			expected: "my-awesome-repo-name",
		},
		{
			name:     "leading and trailing spaces",
			input:    " my repo ",
			expected: "-my-repo-",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: "---",
		},
		{
			name:     "already has hyphens",
			input:    "my-existing-repo",
			expected: "my-existing-repo",
		},
		{
			name:     "mixed spaces and hyphens",
			input:    "my repo-name here",
			expected: "my-repo-name-here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeRepoName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeRepoName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDetermineTargetVisibility tests visibility transformation logic
func TestDetermineTargetVisibility(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name             string
		sourceVisibility string
		publicRepos      string
		internalRepos    string
		expected         string
	}{
		// Public repo transformations
		{
			name:             "public to private (default)",
			sourceVisibility: "public",
			publicRepos:      "private",
			internalRepos:    "private",
			expected:         "private",
		},
		{
			name:             "public stays public",
			sourceVisibility: "public",
			publicRepos:      "public",
			internalRepos:    "private",
			expected:         "public",
		},
		{
			name:             "public to internal",
			sourceVisibility: "public",
			publicRepos:      "internal",
			internalRepos:    "private",
			expected:         "internal",
		},
		// Internal repo transformations
		{
			name:             "internal to private (default)",
			sourceVisibility: "internal",
			publicRepos:      "private",
			internalRepos:    "private",
			expected:         "private",
		},
		{
			name:             "internal stays internal",
			sourceVisibility: "internal",
			publicRepos:      "private",
			internalRepos:    "internal",
			expected:         "internal",
		},
		// Private repos always stay private
		{
			name:             "private stays private",
			sourceVisibility: "private",
			publicRepos:      "public",
			internalRepos:    "internal",
			expected:         "private",
		},
		// Edge cases
		{
			name:             "unknown visibility defaults to private",
			sourceVisibility: "unknown",
			publicRepos:      "public",
			internalRepos:    "internal",
			expected:         "private",
		},
		{
			name:             "empty visibility defaults to private",
			sourceVisibility: "",
			publicRepos:      "public",
			internalRepos:    "internal",
			expected:         "private",
		},
		{
			name:             "case insensitive - PUBLIC",
			sourceVisibility: "PUBLIC",
			publicRepos:      "internal",
			internalRepos:    "private",
			expected:         "internal",
		},
		{
			name:             "case insensitive - INTERNAL",
			sourceVisibility: "INTERNAL",
			publicRepos:      "private",
			internalRepos:    "internal",
			expected:         "internal",
		},
		{
			name:             "invalid public config defaults to private",
			sourceVisibility: "public",
			publicRepos:      "invalid",
			internalRepos:    "private",
			expected:         "private",
		},
		{
			name:             "invalid internal config defaults to private",
			sourceVisibility: "internal",
			publicRepos:      "private",
			internalRepos:    "public", // Invalid - internal can't become public
			expected:         "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(ExecutorConfig{
				SourceClient: &github.Client{},
				DestClient:   &github.Client{},
				Storage:      &storage.Database{},
				Logger:       logger,
				VisibilityHandling: VisibilityHandling{
					PublicRepos:   tt.publicRepos,
					InternalRepos: tt.internalRepos,
				},
			})
			if err != nil {
				t.Fatalf("Failed to create executor: %v", err)
			}

			result := executor.determineTargetVisibility(tt.sourceVisibility)
			if result != tt.expected {
				t.Errorf("determineTargetVisibility(%q) = %q, want %q", tt.sourceVisibility, result, tt.expected)
			}
		})
	}
}

// TestShouldExcludeReleases tests the release exclusion logic
func TestShouldExcludeReleases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name            string
		repoExclude     bool
		batchExclude    bool
		batchNil        bool
		expectedExclude bool
	}{
		{
			name:            "repo excludes, batch nil",
			repoExclude:     true,
			batchNil:        true,
			expectedExclude: true,
		},
		{
			name:            "repo includes, batch nil",
			repoExclude:     false,
			batchNil:        true,
			expectedExclude: false,
		},
		{
			name:            "repo excludes, batch includes",
			repoExclude:     true,
			batchExclude:    false,
			batchNil:        false,
			expectedExclude: true, // Repo takes precedence
		},
		{
			name:            "repo includes, batch excludes",
			repoExclude:     false,
			batchExclude:    true,
			batchNil:        false,
			expectedExclude: true, // Either can enable exclusion
		},
		{
			name:            "both exclude",
			repoExclude:     true,
			batchExclude:    true,
			batchNil:        false,
			expectedExclude: true,
		},
		{
			name:            "both include",
			repoExclude:     false,
			batchExclude:    false,
			batchNil:        false,
			expectedExclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &models.Repository{
				ExcludeReleases: tt.repoExclude,
			}

			var batch *models.Batch
			if !tt.batchNil {
				batch = &models.Batch{
					ExcludeReleases: tt.batchExclude,
				}
			}

			result := executor.shouldExcludeReleases(repo, batch)
			if result != tt.expectedExclude {
				t.Errorf("shouldExcludeReleases() = %v, want %v", result, tt.expectedExclude)
			}
		})
	}
}

// TestShouldExcludeAttachments tests the attachment exclusion logic
func TestShouldExcludeAttachments(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name            string
		repoExclude     bool
		batchExclude    bool
		batchNil        bool
		expectedExclude bool
	}{
		{
			name:            "repo excludes, batch nil",
			repoExclude:     true,
			batchNil:        true,
			expectedExclude: true,
		},
		{
			name:            "repo includes, batch nil",
			repoExclude:     false,
			batchNil:        true,
			expectedExclude: false,
		},
		{
			name:            "repo excludes, batch includes",
			repoExclude:     true,
			batchExclude:    false,
			batchNil:        false,
			expectedExclude: true,
		},
		{
			name:            "repo includes, batch excludes",
			repoExclude:     false,
			batchExclude:    true,
			batchNil:        false,
			expectedExclude: true,
		},
		{
			name:            "both exclude",
			repoExclude:     true,
			batchExclude:    true,
			batchNil:        false,
			expectedExclude: true,
		},
		{
			name:            "both include",
			repoExclude:     false,
			batchExclude:    false,
			batchNil:        false,
			expectedExclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &models.Repository{
				ExcludeAttachments: tt.repoExclude,
			}

			var batch *models.Batch
			if !tt.batchNil {
				batch = &models.Batch{
					ExcludeAttachments: tt.batchExclude,
				}
			}

			result := executor.shouldExcludeAttachments(repo, batch)
			if result != tt.expectedExclude {
				t.Errorf("shouldExcludeAttachments() = %v, want %v", result, tt.expectedExclude)
			}
		})
	}
}

// TestShouldRunPostMigration tests the post-migration mode logic
func TestShouldRunPostMigration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name     string
		mode     PostMigrationMode
		dryRun   bool
		expected bool
	}{
		// Never mode
		{
			name:     "never mode - production",
			mode:     PostMigrationNever,
			dryRun:   false,
			expected: false,
		},
		{
			name:     "never mode - dry run",
			mode:     PostMigrationNever,
			dryRun:   true,
			expected: false,
		},
		// Production only mode
		{
			name:     "production only - production",
			mode:     PostMigrationProductionOnly,
			dryRun:   false,
			expected: true,
		},
		{
			name:     "production only - dry run",
			mode:     PostMigrationProductionOnly,
			dryRun:   true,
			expected: false,
		},
		// Dry run only mode
		{
			name:     "dry run only - production",
			mode:     PostMigrationDryRunOnly,
			dryRun:   false,
			expected: false,
		},
		{
			name:     "dry run only - dry run",
			mode:     PostMigrationDryRunOnly,
			dryRun:   true,
			expected: true,
		},
		// Always mode
		{
			name:     "always mode - production",
			mode:     PostMigrationAlways,
			dryRun:   false,
			expected: true,
		},
		{
			name:     "always mode - dry run",
			mode:     PostMigrationAlways,
			dryRun:   true,
			expected: true,
		},
		// Default (empty) mode
		{
			name:     "default mode - production",
			mode:     "",
			dryRun:   false,
			expected: true, // Defaults to production only
		},
		{
			name:     "default mode - dry run",
			mode:     "",
			dryRun:   true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(ExecutorConfig{
				SourceClient:      &github.Client{},
				DestClient:        &github.Client{},
				Storage:           &storage.Database{},
				Logger:            logger,
				PostMigrationMode: tt.mode,
			})
			if err != nil {
				t.Fatalf("Failed to create executor: %v", err)
			}

			result := executor.shouldRunPostMigration(tt.dryRun)
			if result != tt.expected {
				t.Errorf("shouldRunPostMigration(%v) = %v, want %v", tt.dryRun, result, tt.expected)
			}
		})
	}
}

// TestCompareRepositoryCharacteristics tests the repository comparison logic
func TestCompareRepositoryCharacteristics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	mainBranch := "main"
	developBranch := "develop"
	sha1 := "abc123"
	sha2 := "def456"

	tests := []struct {
		name                  string
		source                *models.Repository
		dest                  *models.Repository
		expectedMismatchCount int
		expectedHasCritical   bool
	}{
		{
			name: "identical repositories",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				BranchCount:   5,
				CommitCount:   100,
				TagCount:      10,
				LastCommitSHA: &sha1,
				HasWiki:       true,
				HasPages:      false,
			},
			dest: &models.Repository{
				DefaultBranch: &mainBranch,
				BranchCount:   5,
				CommitCount:   100,
				TagCount:      10,
				LastCommitSHA: &sha1,
				HasWiki:       true,
				HasPages:      false,
			},
			expectedMismatchCount: 0,
			expectedHasCritical:   false,
		},
		{
			name: "different default branch - critical",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				BranchCount:   5,
				CommitCount:   100,
			},
			dest: &models.Repository{
				DefaultBranch: &developBranch,
				BranchCount:   5,
				CommitCount:   100,
			},
			expectedMismatchCount: 1,
			expectedHasCritical:   true,
		},
		{
			name: "different commit count - critical",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				CommitCount:   100,
			},
			dest: &models.Repository{
				DefaultBranch: &mainBranch,
				CommitCount:   95,
			},
			expectedMismatchCount: 1,
			expectedHasCritical:   true,
		},
		{
			name: "different branch count - critical",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				BranchCount:   5,
			},
			dest: &models.Repository{
				DefaultBranch: &mainBranch,
				BranchCount:   4,
			},
			expectedMismatchCount: 1,
			expectedHasCritical:   true,
		},
		{
			name: "different tag count - non-critical",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				TagCount:      10,
			},
			dest: &models.Repository{
				DefaultBranch: &mainBranch,
				TagCount:      9,
			},
			expectedMismatchCount: 1,
			expectedHasCritical:   false,
		},
		{
			name: "different last commit SHA - critical",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				LastCommitSHA: &sha1,
			},
			dest: &models.Repository{
				DefaultBranch: &mainBranch,
				LastCommitSHA: &sha2,
			},
			expectedMismatchCount: 1,
			expectedHasCritical:   true,
		},
		{
			name: "different wiki setting - non-critical",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				HasWiki:       true,
			},
			dest: &models.Repository{
				DefaultBranch: &mainBranch,
				HasWiki:       false,
			},
			expectedMismatchCount: 1,
			expectedHasCritical:   false,
		},
		{
			name: "multiple non-critical mismatches",
			source: &models.Repository{
				DefaultBranch:  &mainBranch,
				HasWiki:        true,
				HasPages:       true,
				HasDiscussions: true,
			},
			dest: &models.Repository{
				DefaultBranch:  &mainBranch,
				HasWiki:        false,
				HasPages:       false,
				HasDiscussions: false,
			},
			expectedMismatchCount: 3,
			expectedHasCritical:   false,
		},
		{
			name: "multiple mismatches including critical",
			source: &models.Repository{
				DefaultBranch: &mainBranch,
				CommitCount:   100,
				HasWiki:       true,
			},
			dest: &models.Repository{
				DefaultBranch: &developBranch,
				CommitCount:   95,
				HasWiki:       false,
			},
			expectedMismatchCount: 3,
			expectedHasCritical:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mismatches, hasCritical := executor.compareRepositoryCharacteristics(tt.source, tt.dest)

			if len(mismatches) != tt.expectedMismatchCount {
				t.Errorf("Expected %d mismatches, got %d", tt.expectedMismatchCount, len(mismatches))
				for _, m := range mismatches {
					t.Logf("  Mismatch: %s (critical: %v)", m.Field, m.Critical)
				}
			}

			if hasCritical != tt.expectedHasCritical {
				t.Errorf("Expected hasCritical = %v, got %v", tt.expectedHasCritical, hasCritical)
			}
		})
	}
}

// TestGetDestinationOrgWithBatch tests destination org resolution with batch settings
func TestGetDestinationOrgWithBatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	batchDestOrg := "batch-dest-org"

	tests := []struct {
		name        string
		repo        *models.Repository
		batch       *models.Batch
		expectedOrg string
	}{
		{
			name: "repo destination takes precedence over batch",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: ptrString("repo-dest-org/my-repo"),
			},
			batch: &models.Batch{
				DestinationOrg: &batchDestOrg,
			},
			expectedOrg: "repo-dest-org",
		},
		{
			name: "batch destination used when repo has none",
			repo: &models.Repository{
				FullName: "source-org/my-repo",
			},
			batch: &models.Batch{
				DestinationOrg: &batchDestOrg,
			},
			expectedOrg: "batch-dest-org",
		},
		{
			name: "source org used when no destination set",
			repo: &models.Repository{
				FullName: "source-org/my-repo",
			},
			batch:       &models.Batch{},
			expectedOrg: "source-org",
		},
		{
			name: "batch with nil destination org",
			repo: &models.Repository{
				FullName: "source-org/my-repo",
			},
			batch: &models.Batch{
				DestinationOrg: nil,
			},
			expectedOrg: "source-org",
		},
		{
			name: "batch with empty destination org",
			repo: &models.Repository{
				FullName: "source-org/my-repo",
			},
			batch: &models.Batch{
				DestinationOrg: ptrString(""),
			},
			expectedOrg: "source-org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.getDestinationOrg(tt.repo, tt.batch)
			if result != tt.expectedOrg {
				t.Errorf("getDestinationOrg() = %q, want %q", result, tt.expectedOrg)
			}
		})
	}
}

// TestGetDestinationRepoNameWithADO tests ADO-specific repo name extraction
func TestGetDestinationRepoNameWithADO(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	adoProject := "my-project"

	tests := []struct {
		name         string
		repo         *models.Repository
		expectedName string
	}{
		{
			name: "ADO repo - extracts last part",
			repo: &models.Repository{
				FullName:   "my-org/my-project/my-repo",
				ADOProject: &adoProject,
			},
			expectedName: "my-repo",
		},
		{
			name: "ADO repo with spaces in name",
			repo: &models.Repository{
				FullName:   "my-org/my-project/my awesome repo",
				ADOProject: &adoProject,
			},
			expectedName: "my-awesome-repo",
		},
		{
			name: "ADO repo with destination full name - overrides",
			repo: &models.Repository{
				FullName:            "my-org/my-project/old-name",
				ADOProject:          &adoProject,
				DestinationFullName: ptrString("dest-org/new-name"),
			},
			expectedName: "new-name",
		},
		{
			name: "Non-ADO repo - uses source name",
			repo: &models.Repository{
				FullName: "source-org/my-repo",
			},
			expectedName: "my-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.getDestinationRepoName(tt.repo)
			if result != tt.expectedName {
				t.Errorf("getDestinationRepoName() = %q, want %q", result, tt.expectedName)
			}
		})
	}
}

// TestExecutorConfigDefaults tests that executor config defaults are applied correctly
func TestExecutorConfigDefaults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("default post migration mode", func(t *testing.T) {
		executor, err := NewExecutor(ExecutorConfig{
			SourceClient: &github.Client{},
			DestClient:   &github.Client{},
			Storage:      &storage.Database{},
			Logger:       logger,
		})
		if err != nil {
			t.Fatalf("Failed to create executor: %v", err)
		}

		if executor.postMigrationMode != PostMigrationProductionOnly {
			t.Errorf("Expected default post migration mode %q, got %q",
				PostMigrationProductionOnly, executor.postMigrationMode)
		}
	})

	t.Run("default dest repo exists action", func(t *testing.T) {
		executor, err := NewExecutor(ExecutorConfig{
			SourceClient: &github.Client{},
			DestClient:   &github.Client{},
			Storage:      &storage.Database{},
			Logger:       logger,
		})
		if err != nil {
			t.Fatalf("Failed to create executor: %v", err)
		}

		if executor.destRepoExistsAction != DestinationRepoExistsFail {
			t.Errorf("Expected default dest repo action %q, got %q",
				DestinationRepoExistsFail, executor.destRepoExistsAction)
		}
	})

	t.Run("default visibility handling", func(t *testing.T) {
		executor, err := NewExecutor(ExecutorConfig{
			SourceClient: &github.Client{},
			DestClient:   &github.Client{},
			Storage:      &storage.Database{},
			Logger:       logger,
		})
		if err != nil {
			t.Fatalf("Failed to create executor: %v", err)
		}

		if executor.visibilityHandling.PublicRepos != models.VisibilityPrivate {
			t.Errorf("Expected default public repos visibility %q, got %q",
				models.VisibilityPrivate, executor.visibilityHandling.PublicRepos)
		}
		if executor.visibilityHandling.InternalRepos != models.VisibilityPrivate {
			t.Errorf("Expected default internal repos visibility %q, got %q",
				models.VisibilityPrivate, executor.visibilityHandling.InternalRepos)
		}
	})

	t.Run("caches are initialized", func(t *testing.T) {
		executor, err := NewExecutor(ExecutorConfig{
			SourceClient: &github.Client{},
			DestClient:   &github.Client{},
			Storage:      &storage.Database{},
			Logger:       logger,
		})
		if err != nil {
			t.Fatalf("Failed to create executor: %v", err)
		}

		if executor.orgIDCache == nil {
			t.Error("Expected orgIDCache to be initialized")
		}
		if executor.migSourceCache == nil {
			t.Error("Expected migSourceCache to be initialized")
		}
		if executor.adoMigSourceCache == nil {
			t.Error("Expected adoMigSourceCache to be initialized")
		}
	})
}

// TestValidationMismatch tests the ValidationMismatch struct
func TestValidationMismatch(t *testing.T) {
	mismatch := ValidationMismatch{
		Field:       "commit_count",
		SourceValue: 100,
		DestValue:   95,
		Critical:    true,
	}

	if mismatch.Field != "commit_count" {
		t.Errorf("Expected field %q, got %q", "commit_count", mismatch.Field)
	}
	if mismatch.SourceValue != 100 {
		t.Errorf("Expected source value 100, got %v", mismatch.SourceValue)
	}
	if mismatch.DestValue != 95 {
		t.Errorf("Expected dest value 95, got %v", mismatch.DestValue)
	}
	if !mismatch.Critical {
		t.Error("Expected mismatch to be critical")
	}
}

// TestArchiveIDs tests the ArchiveIDs struct
func TestArchiveIDs(t *testing.T) {
	ids := &ArchiveIDs{
		GitArchiveID:      12345,
		MetadataArchiveID: 67890,
	}

	if ids.GitArchiveID != 12345 {
		t.Errorf("Expected git archive ID 12345, got %d", ids.GitArchiveID)
	}
	if ids.MetadataArchiveID != 67890 {
		t.Errorf("Expected metadata archive ID 67890, got %d", ids.MetadataArchiveID)
	}
}

// TestPostMigrationModeConstants tests that all PostMigrationMode constants are valid
func TestPostMigrationModeConstants(t *testing.T) {
	modes := []PostMigrationMode{
		PostMigrationNever,
		PostMigrationProductionOnly,
		PostMigrationDryRunOnly,
		PostMigrationAlways,
	}

	for _, mode := range modes {
		if mode == "" {
			t.Errorf("PostMigrationMode constant should not be empty")
		}
	}

	// Verify expected values
	if PostMigrationNever != "never" {
		t.Errorf("Expected PostMigrationNever = 'never', got %q", PostMigrationNever)
	}
	if PostMigrationProductionOnly != "production_only" {
		t.Errorf("Expected PostMigrationProductionOnly = 'production_only', got %q", PostMigrationProductionOnly)
	}
	if PostMigrationDryRunOnly != "dry_run_only" {
		t.Errorf("Expected PostMigrationDryRunOnly = 'dry_run_only', got %q", PostMigrationDryRunOnly)
	}
	if PostMigrationAlways != "always" {
		t.Errorf("Expected PostMigrationAlways = 'always', got %q", PostMigrationAlways)
	}
}

// TestDestinationRepoExistsActionConstants tests DestinationRepoExistsAction constants
func TestDestinationRepoExistsActionConstants(t *testing.T) {
	actions := []DestinationRepoExistsAction{
		DestinationRepoExistsFail,
		DestinationRepoExistsSkip,
		DestinationRepoExistsDelete,
	}

	for _, action := range actions {
		if action == "" {
			t.Errorf("DestinationRepoExistsAction constant should not be empty")
		}
	}

	// Verify expected values
	if DestinationRepoExistsFail != "fail" {
		t.Errorf("Expected DestinationRepoExistsFail = 'fail', got %q", DestinationRepoExistsFail)
	}
	if DestinationRepoExistsSkip != "skip" {
		t.Errorf("Expected DestinationRepoExistsSkip = 'skip', got %q", DestinationRepoExistsSkip)
	}
	if DestinationRepoExistsDelete != "delete" {
		t.Errorf("Expected DestinationRepoExistsDelete = 'delete', got %q", DestinationRepoExistsDelete)
	}
}

// TestVisibilityHandlingStruct tests the VisibilityHandling struct
func TestVisibilityHandlingStruct(t *testing.T) {
	vh := VisibilityHandling{
		PublicRepos:   "internal",
		InternalRepos: "private",
	}

	if vh.PublicRepos != "internal" {
		t.Errorf("Expected PublicRepos = 'internal', got %q", vh.PublicRepos)
	}
	if vh.InternalRepos != "private" {
		t.Errorf("Expected InternalRepos = 'private', got %q", vh.InternalRepos)
	}
}
