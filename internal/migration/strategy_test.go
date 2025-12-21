package migration

import (
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestStrategyRegistry_GetStrategy(t *testing.T) {
	// Create a mock executor (nil is fine for testing strategy selection)
	registry := NewStrategyRegistry(
		NewGitHubMigrationStrategy(nil),
		NewADOMigrationStrategy(nil),
	)

	tests := []struct {
		name         string
		repo         *models.Repository
		wantStrategy string
		wantNil      bool
	}{
		{
			name: "GitHub repository (no ADO project)",
			repo: &models.Repository{
				FullName:   "org/repo",
				ADOProject: nil,
			},
			wantStrategy: "GitHub",
			wantNil:      false,
		},
		{
			name: "GitHub repository (empty ADO project)",
			repo: &models.Repository{
				FullName:   "org/repo",
				ADOProject: strPtr(""),
			},
			wantStrategy: "GitHub",
			wantNil:      false,
		},
		{
			name: "ADO repository",
			repo: &models.Repository{
				FullName:   "project/repo",
				ADOProject: strPtr("MyProject"),
			},
			wantStrategy: "AzureDevOps",
			wantNil:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := registry.GetStrategy(tt.repo)

			if tt.wantNil {
				if strategy != nil {
					t.Errorf("GetStrategy() returned strategy %s, want nil", strategy.Name())
				}
				return
			}

			if strategy == nil {
				t.Error("GetStrategy() returned nil, want strategy")
				return
			}

			if strategy.Name() != tt.wantStrategy {
				t.Errorf("GetStrategy() = %s, want %s", strategy.Name(), tt.wantStrategy)
			}
		})
	}
}

func TestGitHubMigrationStrategy_SupportsRepository(t *testing.T) {
	strategy := NewGitHubMigrationStrategy(nil)

	tests := []struct {
		name       string
		repo       *models.Repository
		wantResult bool
	}{
		{
			name:       "nil ADO project",
			repo:       &models.Repository{ADOProject: nil},
			wantResult: true,
		},
		{
			name:       "empty ADO project",
			repo:       &models.Repository{ADOProject: strPtr("")},
			wantResult: true,
		},
		{
			name:       "has ADO project",
			repo:       &models.Repository{ADOProject: strPtr("MyProject")},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.SupportsRepository(tt.repo)
			if result != tt.wantResult {
				t.Errorf("SupportsRepository() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestADOMigrationStrategy_SupportsRepository(t *testing.T) {
	strategy := NewADOMigrationStrategy(nil)

	tests := []struct {
		name       string
		repo       *models.Repository
		wantResult bool
	}{
		{
			name:       "nil ADO project",
			repo:       &models.Repository{ADOProject: nil},
			wantResult: false,
		},
		{
			name:       "empty ADO project",
			repo:       &models.Repository{ADOProject: strPtr("")},
			wantResult: false,
		},
		{
			name:       "has ADO project",
			repo:       &models.Repository{ADOProject: strPtr("MyProject")},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.SupportsRepository(tt.repo)
			if result != tt.wantResult {
				t.Errorf("SupportsRepository() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestMigrationStrategy_ShouldUnlockSource(t *testing.T) {
	ghStrategy := NewGitHubMigrationStrategy(nil)
	adoStrategy := NewADOMigrationStrategy(nil)

	if !ghStrategy.ShouldUnlockSource() {
		t.Error("GitHubMigrationStrategy.ShouldUnlockSource() = false, want true")
	}

	if adoStrategy.ShouldUnlockSource() {
		t.Error("ADOMigrationStrategy.ShouldUnlockSource() = true, want false")
	}
}

func TestStrategyRegistry_RegisterStrategy(t *testing.T) {
	registry := NewStrategyRegistry()

	// Initially empty
	repo := &models.Repository{ADOProject: nil}
	if strategy := registry.GetStrategy(repo); strategy != nil {
		t.Error("Expected nil strategy for empty registry")
	}

	// Register GitHub strategy
	registry.RegisterStrategy(NewGitHubMigrationStrategy(nil))

	// Now should find it
	strategy := registry.GetStrategy(repo)
	if strategy == nil {
		t.Error("Expected strategy after registration")
	}
	if strategy.Name() != "GitHub" {
		t.Errorf("Expected GitHub strategy, got %s", strategy.Name())
	}
}

// strPtr is a helper to create string pointers
func strPtr(s string) *string {
	return &s
}
