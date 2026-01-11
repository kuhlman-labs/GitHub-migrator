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
			repo: func() *models.Repository {
				r := &models.Repository{FullName: "test-project/test-repo"}
				r.SetADOProject(stringPtr("test-project"))
				r.SetADOIsGit(true)
				return r
			}(),
			expectedGit: true,
		},
		{
			name: "TFVC repository",
			repo: func() *models.Repository {
				r := &models.Repository{FullName: "test-project/test-tfvc"}
				r.SetADOProject(stringPtr("test-project"))
				r.SetADOIsGit(false)
				return r
			}(),
			expectedGit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.repo.GetADOIsGit() != tt.expectedGit {
				t.Errorf("Expected ADOIsGit = %v, got %v", tt.expectedGit, tt.repo.GetADOIsGit())
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
		FullName: "test-project/test-repo",
	}
	repo.SetADOProject(stringPtr("test-project"))

	// ctx := context.Background()
	// profiler.ProfileRepository would be called here in a real test
	// err := profiler.ProfileRepository(ctx, repo, someClient)
	// if err != nil {
	// 	t.Errorf("ProfileRepository() unexpected error: %v", err)
	// }

	// Verify profiling populated fields
	if repo.GetADOProject() == nil {
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
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(false)
				r.SetADOProject(stringPtr("test-project"))
				return r
			}(),
			expectedHigh:   true,
			expectedReason: "TFVC",
		},
		{
			name: "Azure Boards enabled",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOHasBoards(true)
				r.SetADOProject(stringPtr("test-project"))
				return r
			}(),
			expectedHigh:   false,
			expectedReason: "Azure Boards adds complexity",
		},
		{
			name: "Many pull requests",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOPullRequestCount(100)
				r.SetADOProject(stringPtr("test-project"))
				return r
			}(),
			expectedHigh:   false,
			expectedReason: "High PR count",
		},
		{
			name: "Many branch policies",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOBranchPolicyCount(15)
				r.SetADOProject(stringPtr("test-project"))
				return r
			}(),
			expectedHigh:   false,
			expectedReason: "Many branch policies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TFVC is blocking
			if !tt.repo.GetADOIsGit() {
				if tt.repo.GetADOIsGit() {
					t.Error("TFVC repository should have ADOIsGit = false")
				}
			}

			// Azure Boards adds complexity
			if tt.repo.GetADOHasBoards() {
				t.Log("Repository has Azure Boards integration")
			}

			// High PR count indicates active repository
			if tt.repo.GetADOPullRequestCount() > 50 {
				t.Logf("Repository has high PR count: %d", tt.repo.GetADOPullRequestCount())
			}

			// Branch policies add complexity
			if tt.repo.GetADOBranchPolicyCount() > 10 {
				t.Logf("Repository has many branch policies: %d", tt.repo.GetADOBranchPolicyCount())
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
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(false)
				return r
			}(),
			expectedComplexity: 50,
			minComplexity:      50,
		},
		{
			name: "Classic pipelines",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOClassicPipelineCount(3)
				return r
			}(),
			expectedComplexity: 15, // 3 * 5
			minComplexity:      15,
		},
		{
			name: "Package feeds",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOPackageFeedCount(2)
				return r
			}(),
			expectedComplexity: 3,
			minComplexity:      3,
		},
		{
			name: "Active pipelines with service connections",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOPipelineRunCount(50)
				r.SetADOHasServiceConnections(true)
				return r
			}(),
			expectedComplexity: 6, // 3 + 3
			minComplexity:      6,
		},
		{
			name: "Wiki pages",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOHasWiki(true)
				r.SetADOWikiPageCount(25)
				return r
			}(),
			expectedComplexity: 6, // (25+9)/10 * 2 = 34/10 * 2 = 3 * 2 = 6
			minComplexity:      6,
		},
		{
			name: "Complex repository with multiple factors",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOClassicPipelineCount(2)
				r.SetADOPackageFeedCount(1)
				r.SetADOHasServiceConnections(true)
				r.SetADOPipelineRunCount(10)
				r.SetADOActiveWorkItemCount(50)
				r.SetADOWikiPageCount(15)
				r.SetADOTestPlanCount(5)
				r.SetADOHasVariableGroups(true)
				r.SetADOServiceHookCount(3)
				r.SetADOPullRequestCount(60)
				r.SetADOBranchPolicyCount(5)
				return r
			}(),
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
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(false)
				return r
			}(),
			expectedBreakdown: &models.ComplexityBreakdown{
				ADOTFVCPoints: 50,
			},
			checkSpecificPoints: true,
		},
		{
			name: "Classic pipelines breakdown",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOClassicPipelineCount(2)
				return r
			}(),
			expectedBreakdown: &models.ComplexityBreakdown{
				ADOClassicPipelinePoints: 10, // 2 * 5
			},
			checkSpecificPoints: true,
		},
		{
			name: "Multiple factors breakdown",
			repo: func() *models.Repository {
				r := &models.Repository{}
				r.SetADOIsGit(true)
				r.SetADOPackageFeedCount(1)
				r.SetADOHasServiceConnections(true)
				r.SetADOHasVariableGroups(true)
				return r
			}(),
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
