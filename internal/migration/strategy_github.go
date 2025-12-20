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
	return repo.ADOProject == nil || *repo.ADOProject == ""
}

// ValidateSource validates that the source client is configured for GitHub migrations.
func (s *GitHubMigrationStrategy) ValidateSource(ctx context.Context, repo *models.Repository) error {
	if s.executor.sourceClient == nil {
		return fmt.Errorf("source client is required for GitHub-to-GitHub migrations")
	}
	return nil
}

// PrepareArchives generates migration archives on the source (GHES).
// This is a multi-phase operation:
// 1. Generate git and metadata archives
// 2. Poll for archive completion
// 3. Return archive URLs for migration
func (s *GitHubMigrationStrategy) PrepareArchives(ctx context.Context, mc *MigrationContext) error {
	e := s.executor

	// Phase: Archive generation
	migrationMode := "production migration"
	if mc.DryRun {
		migrationMode = "dry run migration (lock_repositories: false)"
	}

	e.logger.Info("Generating archives on source repository", "repo", mc.Repo.FullName, "mode", migrationMode)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "archive_generation", "initiate",
		fmt.Sprintf("Initiating archive generation on %s with options: exclude_releases=%v, exclude_attachments=%v (%s)",
			e.sourceClient.BaseURL(), mc.ExcludeReleases, mc.ExcludeAttachments, migrationMode), nil)

	archiveIDs, err := e.generateArchivesOnGHES(ctx, mc.Repo, mc.Batch, mc.LockRepositories)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "archive_generation", "initiate", "Failed to generate archives", &errMsg)
		return fmt.Errorf("failed to generate archives: %w", err)
	}

	mc.ArchiveIDs = archiveIDs
	details := fmt.Sprintf("Git Archive ID: %d, Metadata Archive ID: %d", archiveIDs.GitArchiveID, archiveIDs.MetadataArchiveID)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "archive_generation", "initiate", "Archive generation initiated successfully", &details)

	// Update repository status
	mc.Repo.Status = string(models.StatusArchiveGenerating)
	migID := archiveIDs.GitArchiveID
	mc.Repo.SourceMigrationID = &migID
	mc.Repo.IsSourceLocked = mc.LockRepositories
	if err := e.storage.UpdateRepository(ctx, mc.Repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	// Phase: Archive polling
	e.logger.Info("Polling archive generation status", "repo", mc.Repo.FullName,
		"git_archive_id", mc.ArchiveIDs.GitArchiveID,
		"metadata_archive_id", mc.ArchiveIDs.MetadataArchiveID)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "archive_generation", "poll", "Polling for archive generation completion", nil)

	archiveURLs, err := e.pollArchiveGeneration(ctx, mc.Repo, mc.HistoryID, mc.ArchiveIDs)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "archive_generation", "poll", "Archive generation failed", &errMsg)
		return fmt.Errorf("archive generation failed: %w", err)
	}

	mc.ArchiveURLs = archiveURLs
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "archive_generation", "complete", "Archives generated successfully", nil)

	return nil
}

// StartMigration starts the migration using the archive URLs.
func (s *GitHubMigrationStrategy) StartMigration(ctx context.Context, mc *MigrationContext) (string, error) {
	e := s.executor

	e.logger.Info("Starting migration on GHEC", "repo", mc.Repo.FullName)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration_start", "initiate", "Starting migration on destination", nil)

	migrationID, err := e.startRepositoryMigration(ctx, mc.Repo, mc.Batch, mc.ArchiveURLs)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "migration_start", "initiate", "Failed to start migration", &errMsg)
		return "", fmt.Errorf("failed to start migration: %w", err)
	}

	details := fmt.Sprintf("Migration ID: %s", migrationID)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration_start", "initiate", "Migration started successfully", &details)

	mc.Repo.Status = string(models.StatusMigratingContent)
	if err := e.storage.UpdateRepository(ctx, mc.Repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	return migrationID, nil
}

// ShouldUnlockSource returns true since GitHub source repositories should be unlocked.
func (s *GitHubMigrationStrategy) ShouldUnlockSource() bool {
	return true
}
