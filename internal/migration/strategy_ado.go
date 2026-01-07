package migration

import (
	"context"
	"fmt"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// ADOMigrationStrategy implements MigrationStrategy for Azure DevOps to GitHub migrations.
// This handles migrations from Azure DevOps (Git repositories) to GitHub Enterprise Cloud (GHEC).
// Note: ADO migrations differ from GitHub migrations in several ways:
// - No archive generation (GEI pulls directly from ADO)
// - No source repository locking (ADO doesn't support it)
// - Uses ADO-specific GraphQL mutation fields
type ADOMigrationStrategy struct {
	executor *Executor
}

// NewADOMigrationStrategy creates a new ADO migration strategy.
func NewADOMigrationStrategy(executor *Executor) *ADOMigrationStrategy {
	return &ADOMigrationStrategy{
		executor: executor,
	}
}

// Name returns the strategy name.
func (s *ADOMigrationStrategy) Name() string {
	return "AzureDevOps"
}

// SupportsRepository returns true if this is an Azure DevOps source repository.
// ADO repositories are identified by having an ADO project set.
func (s *ADOMigrationStrategy) SupportsRepository(repo *models.Repository) bool {
	adoProject := repo.GetADOProject()
	return adoProject != nil && *adoProject != ""
}

// ValidateSource validates that the ADO PAT can access the source repository.
// This catches permission issues before GitHub's preflight checks.
func (s *ADOMigrationStrategy) ValidateSource(ctx context.Context, repo *models.Repository) error {
	e := s.executor

	e.logger.Info("Validating ADO PAT access to source repository", "repo", repo.FullName)

	if err := e.validateADORepositoryAccess(ctx, repo); err != nil {
		return fmt.Errorf("ADO PAT validation failed: %w", err)
	}

	e.logger.Info("ADO PAT successfully validated", "repo", repo.FullName)
	return nil
}

// PrepareArchives is a no-op for ADO migrations.
// Unlike GitHub migrations, GEI pulls directly from ADO using the provided PAT,
// so no archive generation is needed.
func (s *ADOMigrationStrategy) PrepareArchives(ctx context.Context, mc *MigrationContext) error {
	e := s.executor

	e.logger.Info("Skipping archive generation for ADO migration (GEI pulls directly from ADO)",
		"repo", mc.Repo.FullName)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "archive_generation", "skip",
		"ADO migrations do not require archive generation - GEI pulls directly from source", nil)

	return nil
}

// StartMigration starts the ADO-to-GitHub migration using GEI.
func (s *ADOMigrationStrategy) StartMigration(ctx context.Context, mc *MigrationContext) (string, error) {
	e := s.executor

	e.logger.Info("Starting ADO repository migration on GitHub", "repo", mc.Repo.FullName)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration", "initiate",
		"Starting ADO-to-GitHub migration with GitHub Enterprise Importer", nil)

	migrationID, err := e.startADORepositoryMigration(ctx, mc.Repo, mc.Batch)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "migration", "initiate", "Failed to start migration", &errMsg)
		return "", fmt.Errorf("failed to start migration: %w", err)
	}

	e.logger.Info("Migration started successfully",
		"repo", mc.Repo.FullName,
		"migration_id", migrationID)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration", "initiated",
		fmt.Sprintf("Migration started with ID: %s", migrationID), nil)

	mc.Repo.Status = string(models.StatusMigratingContent)
	if err := e.storage.UpdateRepository(ctx, mc.Repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	return migrationID, nil
}

// ShouldUnlockSource returns false since ADO doesn't support repository locking.
func (s *ADOMigrationStrategy) ShouldUnlockSource() bool {
	return false
}
