package migration

import (
	"context"
	"fmt"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// GitHubMigrationStrategy implements MigrationStrategy for GitHub-to-GitHub migrations.
// This handles migrations from GitHub Enterprise Server (GHES) to GitHub Enterprise Cloud (GHEC).
type GitHubMigrationStrategy struct {
	executor *Executor
}

// NewGitHubMigrationStrategy creates a new GitHub migration strategy.
func NewGitHubMigrationStrategy(executor *Executor) *GitHubMigrationStrategy {
	return &GitHubMigrationStrategy{
		executor: executor,
	}
}

// Name returns the strategy name.
func (s *GitHubMigrationStrategy) Name() string {
	return "GitHub"
}

// SupportsRepository returns true if this is a GitHub source repository.
// GitHub repositories are identified by NOT having an ADO project set.
func (s *GitHubMigrationStrategy) SupportsRepository(repo *models.Repository) bool {
	// A repository is considered a GitHub source if it doesn't have an ADO project
	adoProject := repo.GetADOProject()
	return adoProject == nil || *adoProject == ""
}

// ValidateSource validates that the source client is configured for GitHub migrations.
func (s *GitHubMigrationStrategy) ValidateSource(ctx context.Context, repo *models.Repository) error {
	if s.executor.sourceClient == nil {
		return fmt.Errorf("source client is required for GitHub-to-GitHub migrations")
	}
	return nil
}

// PrepareArchives generates migration archives on the source (GHES).
// This delegates to the existing phase methods to avoid code duplication:
// 1. phaseArchiveGeneration - initiates archive creation
// 2. phaseArchivePolling - polls for completion and retrieves URLs
func (s *GitHubMigrationStrategy) PrepareArchives(ctx context.Context, mc *MigrationContext) error {
	// Delegate to existing phase methods which contain the archive generation logic
	if err := s.executor.phaseArchiveGeneration(ctx, mc); err != nil {
		return err
	}
	return s.executor.phaseArchivePolling(ctx, mc)
}

// StartMigration starts the migration using the archive URLs.
// This delegates to phaseMigrationStart to avoid code duplication.
func (s *GitHubMigrationStrategy) StartMigration(ctx context.Context, mc *MigrationContext) (string, error) {
	// Delegate to existing phase method which contains the migration start logic
	if err := s.executor.phaseMigrationStart(ctx, mc); err != nil {
		return "", err
	}
	return mc.MigrationID, nil
}

// ShouldUnlockSource returns true since GitHub source repositories should be unlocked.
func (s *GitHubMigrationStrategy) ShouldUnlockSource() bool {
	return true
}
