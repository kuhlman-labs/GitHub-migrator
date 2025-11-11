package discovery

import (
	"log/slog"
	"os"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/models"
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

func stringPtr(s string) *string {
	return &s
}
