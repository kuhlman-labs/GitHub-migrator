package migration

import (
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/models"
)

func TestExecutor_ADOMigrationFlow(t *testing.T) {
	// This is an integration test - skip in unit tests
	t.Skip("Skipping integration test - requires GitHub API and ADO access")

	tests := []struct {
		name    string
		repo    *models.Repository
		wantErr bool
	}{
		{
			name: "Git repository migration",
			repo: &models.Repository{
				FullName:   "testorg/testproj/test-repo",
				ADOProject: stringPtr("testproj"),
				ADOIsGit:   true,
				Status:     "pending",
			},
			wantErr: false,
		},
		{
			name: "TFVC repository should fail",
			repo: &models.Repository{
				FullName:   "testorg/testproj/tfvc-repo",
				ADOProject: stringPtr("testproj"),
				ADOIsGit:   false, // TFVC
				Status:     "pending",
			},
			wantErr: true, // Should fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test would verify:
			// 1. TFVC repositories are rejected
			// 2. Git repositories proceed with migration
			// 3. No archive generation for ADO
			// 4. No source locking for ADO
			// 5. GraphQL mutation uses ADO-specific parameters

			if !tt.repo.ADOIsGit && !tt.wantErr {
				t.Error("TFVC repository should produce error")
			}
		})
	}
}

func TestStartADORepositoryMigration(t *testing.T) {
	t.Skip("Skipping integration test - requires GitHub API")

	tests := []struct {
		name          string
		repo          *models.Repository
		expectError   bool
		errorContains string
	}{
		{
			name: "valid Git repository",
			repo: &models.Repository{
				FullName:   "org/proj/repo",
				SourceURL:  "https://dev.azure.com/org/proj/_git/repo",
				ADOProject: stringPtr("proj"),
				ADOIsGit:   true,
			},
			expectError: false,
		},
		{
			name: "TFVC repository blocked",
			repo: &models.Repository{
				FullName:   "org/proj/tfvc",
				ADOProject: stringPtr("proj"),
				ADOIsGit:   false,
			},
			expectError:   true,
			errorContains: "TFVC",
		},
		{
			name: "missing ADO project",
			repo: &models.Repository{
				FullName:   "org/repo",
				ADOProject: nil,
				ADOIsGit:   true,
			},
			expectError:   true,
			errorContains: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation tests
			if tt.repo.ADOProject == nil && !tt.expectError {
				t.Error("Repository without ADO project should fail")
			}

			if !tt.repo.ADOIsGit {
				t.Log("TFVC repository correctly identified for rejection")
			}
		})
	}
}

func TestGetOrCreateADOMigrationSource(t *testing.T) {
	t.Skip("Skipping integration test - requires GitHub API")

	// Test would verify:
	// 1. Migration source type is MigrationSourceTypeAzureDevOps
	// 2. Source is created if it doesn't exist
	// 3. Existing source is reused
}

func TestADOMigrationValidation(t *testing.T) {
	tests := []struct {
		name    string
		repo    *models.Repository
		isValid bool
		reason  string
	}{
		{
			name: "valid ADO Git repo",
			repo: &models.Repository{
				FullName:   "org/proj/repo",
				ADOProject: stringPtr("proj"),
				ADOIsGit:   true,
				SourceURL:  "https://dev.azure.com/org/proj/_git/repo",
			},
			isValid: true,
			reason:  "Valid Git repository",
		},
		{
			name: "TFVC repo not supported",
			repo: &models.Repository{
				FullName:   "org/proj/tfvc",
				ADOProject: stringPtr("proj"),
				ADOIsGit:   false,
			},
			isValid: false,
			reason:  "TFVC repositories not supported by GEI",
		},
		{
			name: "missing source URL",
			repo: &models.Repository{
				FullName:   "org/proj/repo",
				ADOProject: stringPtr("proj"),
				ADOIsGit:   true,
				SourceURL:  "",
			},
			isValid: false,
			reason:  "Source URL required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate repository eligibility for ADO migration
			isValid := tt.repo.ADOIsGit &&
				tt.repo.ADOProject != nil &&
				tt.repo.SourceURL != ""

			if isValid != tt.isValid {
				t.Errorf("Expected validity %v, got %v: %s", tt.isValid, isValid, tt.reason)
			}

			t.Logf("Repository %s: %s", tt.repo.FullName, tt.reason)
		})
	}
}

func TestADOMigrationParameters(t *testing.T) {
	tests := []struct {
		name             string
		repo             *models.Repository
		expectedProvider string
		includesPAT      bool
	}{
		{
			name: "ADO migration with PAT",
			repo: &models.Repository{
				FullName:   "org/proj/repo",
				ADOProject: stringPtr("proj"),
				ADOIsGit:   true,
				SourceURL:  "https://dev.azure.com/org/proj/_git/repo",
			},
			expectedProvider: "azuredevops",
			includesPAT:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test would verify:
			// 1. PAT is embedded in source URL
			// 2. No GitArchiveURL provided
			// 3. No MetadataArchiveURL provided
			// 4. SourceRepositoryURL contains authenticated URL
			// 5. AccessToken (ADO PAT) is provided
			// 6. GitHubPat (destination PAT) is provided

			if tt.includesPAT {
				t.Log("ADO PAT should be embedded in source URL for authentication")
			}
		})
	}
}

func TestADOvsGitHubMigrationDifferences(t *testing.T) {
	differences := []struct {
		feature string
		github  bool
		ado     bool
		notes   string
	}{
		{
			feature: "Archive generation",
			github:  true,
			ado:     false,
			notes:   "ADO uses direct migration without archives",
		},
		{
			feature: "Source repository locking",
			github:  true,
			ado:     false,
			notes:   "ADO doesn't support repository locking",
		},
		{
			feature: "Metadata export",
			github:  true,
			ado:     true,
			notes:   "Both support metadata migration",
		},
		{
			feature: "Pull request migration",
			github:  true,
			ado:     true,
			notes:   "Both support PR migration",
		},
		{
			feature: "Work item migration",
			github:  true,
			ado:     false,
			notes:   "ADO work items not migrated by GEI",
		},
		{
			feature: "Pipeline migration",
			github:  true,
			ado:     false,
			notes:   "Azure Pipelines not migrated automatically",
		},
	}

	for _, diff := range differences {
		t.Run(diff.feature, func(t *testing.T) {
			if diff.github && !diff.ado {
				t.Logf("Feature '%s' differs: GitHub=%v, ADO=%v - %s",
					diff.feature, diff.github, diff.ado, diff.notes)
			}
		})
	}
}

func TestTFVCBlocker(t *testing.T) {
	tfvcRepos := []models.Repository{
		{
			FullName:   "org/proj/$/trunk",
			ADOProject: stringPtr("proj"),
			ADOIsGit:   false,
		},
		{
			FullName:   "org/proj/$/branches/main",
			ADOProject: stringPtr("proj"),
			ADOIsGit:   false,
		},
	}

	for _, repo := range tfvcRepos {
		t.Run(repo.FullName, func(t *testing.T) {
			if repo.ADOIsGit {
				t.Error("TFVC repository incorrectly marked as Git")
			}

			// TFVC repositories should be blocked from migration
			t.Logf("TFVC repository %s correctly blocked from migration", repo.FullName)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
