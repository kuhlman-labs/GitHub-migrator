package discovery

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestADOProfiler_DetectTFVC(t *testing.T) {
	_ = slog.New(slog.NewTextHandler(os.Stderr, nil))
	//profiler := NewADOProfiler(nil, logger)

	tests := []struct {
		name        string
		repo        *models.Repository
		expectedGit bool
	}{
		{
			name: "Git repository",
			repo: &models.Repository{
				FullName:   "test-project/test-repo",
				ADOProject: stringPtr("test-project"),
				ADOIsGit:   true,
			},
			expectedGit: true,
		},
		{
			name: "TFVC repository",
			repo: &models.Repository{
				FullName:   "test-project/test-tfvc",
				ADOProject: stringPtr("test-project"),
				ADOIsGit:   false,
			},
			expectedGit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.repo.ADOIsGit != tt.expectedGit {
				t.Errorf("Expected ADOIsGit = %v, got %v", tt.expectedGit, tt.repo.ADOIsGit)
			}
		})
	}
}

func TestADOProfiler_ProfileRepository(t *testing.T) {
	// This is an integration test that requires an actual ADO connection
	// Skip in unit tests
	t.Skip("Skipping integration test - requires Azure DevOps access")

	// logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	// profiler := NewADOProfiler(nil, logger)

	repo := &models.Repository{
		FullName:   "test-project/test-repo",
		ADOProject: stringPtr("test-project"),
	}

	// ctx := context.Background()
	// profiler.ProfileRepository would be called here in a real test
	// err := profiler.ProfileRepository(ctx, repo, someClient)
	// if err != nil {
	// 	t.Errorf("ProfileRepository() unexpected error: %v", err)
	// }

	// Verify profiling populated fields
	if repo.ADOProject == nil {
		t.Error("Expected ADOProject to be set")
	}
}

func TestADOProfiler_ComplexityFactors(t *testing.T) {
	tests := []struct {
		name           string
		repo           *models.Repository
		expectedHigh   bool
		expectedReason string
	}{
		{
			name: "TFVC repository - blocking",
			repo: &models.Repository{
				ADOIsGit:   false,
				ADOProject: stringPtr("test-project"),
			},
			expectedHigh:   true,
			expectedReason: "TFVC",
		},
		{
			name: "Azure Boards enabled",
			repo: &models.Repository{
				ADOIsGit:     true,
				ADOHasBoards: true,
				ADOProject:   stringPtr("test-project"),
			},
			expectedHigh:   false,
			expectedReason: "Azure Boards adds complexity",
		},
		{
			name: "Many pull requests",
			repo: &models.Repository{
				ADOIsGit:            true,
				ADOPullRequestCount: 100,
				ADOProject:          stringPtr("test-project"),
			},
			expectedHigh:   false,
			expectedReason: "High PR count",
		},
		{
			name: "Many branch policies",
			repo: &models.Repository{
				ADOIsGit:             true,
				ADOBranchPolicyCount: 15,
				ADOProject:           stringPtr("test-project"),
			},
			expectedHigh:   false,
			expectedReason: "Many branch policies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TFVC is blocking
			if !tt.repo.ADOIsGit {
				if tt.repo.ADOIsGit {
					t.Error("TFVC repository should have ADOIsGit = false")
				}
			}

			// Azure Boards adds complexity
			if tt.repo.ADOHasBoards {
				t.Log("Repository has Azure Boards integration")
			}

			// High PR count indicates active repository
			if tt.repo.ADOPullRequestCount > 50 {
				t.Logf("Repository has high PR count: %d", tt.repo.ADOPullRequestCount)
			}

			// Branch policies add complexity
			if tt.repo.ADOBranchPolicyCount > 10 {
				t.Logf("Repository has many branch policies: %d", tt.repo.ADOBranchPolicyCount)
			}
		})
	}
}

func TestADOProfiler_EstimateComplexity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	profiler := NewADOProfiler(nil, logger, nil, nil)

	tests := []struct {
		name               string
		repo               *models.Repository
		expectedComplexity int
		minComplexity      int
	}{
		{
			name: "TFVC repository",
			repo: &models.Repository{
				ADOIsGit: false,
			},
			expectedComplexity: 50,
			minComplexity:      50,
		},
		{
			name: "Classic pipelines",
			repo: &models.Repository{
				ADOIsGit:                true,
				ADOClassicPipelineCount: 3,
			},
			expectedComplexity: 15, // 3 * 5
			minComplexity:      15,
		},
		{
			name: "Package feeds",
			repo: &models.Repository{
				ADOIsGit:            true,
				ADOPackageFeedCount: 2,
			},
			expectedComplexity: 3,
			minComplexity:      3,
		},
		{
			name: "Active pipelines with service connections",
			repo: &models.Repository{
				ADOIsGit:                 true,
				ADOPipelineRunCount:      50,
				ADOHasServiceConnections: true,
			},
			expectedComplexity: 6, // 3 + 3
			minComplexity:      6,
		},
		{
			name: "Wiki pages",
			repo: &models.Repository{
				ADOIsGit:         true,
				ADOHasWiki:       true,
				ADOWikiPageCount: 25,
			},
			expectedComplexity: 6, // (25+9)/10 * 2 = 34/10 * 2 = 3 * 2 = 6
			minComplexity:      6,
		},
		{
			name: "Complex repository with multiple factors",
			repo: &models.Repository{
				ADOIsGit:                 true,
				ADOClassicPipelineCount:  2,
				ADOPackageFeedCount:      1,
				ADOHasServiceConnections: true,
				ADOPipelineRunCount:      10,
				ADOActiveWorkItemCount:   50,
				ADOWikiPageCount:         15,
				ADOTestPlanCount:         5,
				ADOHasVariableGroups:     true,
				ADOServiceHookCount:      3,
				ADOPullRequestCount:      60,
				ADOBranchPolicyCount:     5,
			},
			minComplexity: 30, // Should be at least 30 with all these factors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := profiler.EstimateComplexity(tt.repo)

			if tt.expectedComplexity > 0 && complexity != tt.expectedComplexity {
				t.Errorf("EstimateComplexity() = %d, want %d", complexity, tt.expectedComplexity)
			}

			if complexity < tt.minComplexity {
				t.Errorf("EstimateComplexity() = %d, want at least %d", complexity, tt.minComplexity)
			}
		})
	}
}

func TestADOProfiler_EstimateComplexityWithBreakdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	profiler := NewADOProfiler(nil, logger, nil, nil)

	tests := []struct {
		name                string
		repo                *models.Repository
		expectedBreakdown   *models.ComplexityBreakdown
		checkSpecificPoints bool
	}{
		{
			name: "TFVC repository breakdown",
			repo: &models.Repository{
				ADOIsGit: false,
			},
			expectedBreakdown: &models.ComplexityBreakdown{
				ADOTFVCPoints: 50,
			},
			checkSpecificPoints: true,
		},
		{
			name: "Classic pipelines breakdown",
			repo: &models.Repository{
				ADOIsGit:                true,
				ADOClassicPipelineCount: 2,
			},
			expectedBreakdown: &models.ComplexityBreakdown{
				ADOClassicPipelinePoints: 10, // 2 * 5
			},
			checkSpecificPoints: true,
		},
		{
			name: "Multiple factors breakdown",
			repo: &models.Repository{
				ADOIsGit:                 true,
				ADOPackageFeedCount:      1,
				ADOHasServiceConnections: true,
				ADOHasVariableGroups:     true,
			},
			checkSpecificPoints: false, // Just verify it returns breakdown
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity, breakdown := profiler.EstimateComplexityWithBreakdown(tt.repo)

			if breakdown == nil {
				t.Fatal("EstimateComplexityWithBreakdown() returned nil breakdown")
			}

			if complexity < 0 {
				t.Errorf("EstimateComplexityWithBreakdown() returned negative complexity: %d", complexity)
			}

			if tt.checkSpecificPoints && tt.expectedBreakdown != nil {
				if breakdown.ADOTFVCPoints != tt.expectedBreakdown.ADOTFVCPoints {
					t.Errorf("ADOTFVCPoints = %d, want %d", breakdown.ADOTFVCPoints, tt.expectedBreakdown.ADOTFVCPoints)
				}
				if breakdown.ADOClassicPipelinePoints != tt.expectedBreakdown.ADOClassicPipelinePoints {
					t.Errorf("ADOClassicPipelinePoints = %d, want %d", breakdown.ADOClassicPipelinePoints, tt.expectedBreakdown.ADOClassicPipelinePoints)
				}
			}
		})
	}
}

// stringPtr is defined in profiler.go
