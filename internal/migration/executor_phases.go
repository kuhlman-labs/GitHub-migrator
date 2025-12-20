package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// MigrationContext holds all state needed during a migration execution.
// It provides a clean way to pass state between phases without cluttering method signatures.
type MigrationContext struct {
	// Input parameters
	Repo      *models.Repository
	Batch     *models.Batch
	DryRun    bool
	HistoryID *int64

	// Computed values
	ExcludeReleases    bool
	ExcludeAttachments bool
	LockRepositories   bool

	// Migration state
	ArchiveIDs  *ArchiveIDs
	ArchiveURLs *ArchiveURLs
	MigrationID string
}

// NewMigrationContext creates a new MigrationContext with computed values.
func (e *Executor) NewMigrationContext(repo *models.Repository, batch *models.Batch, dryRun bool) *MigrationContext {
	return &MigrationContext{
		Repo:               repo,
		Batch:              batch,
		DryRun:             dryRun,
		ExcludeReleases:    e.shouldExcludeReleases(repo, batch),
		ExcludeAttachments: e.shouldExcludeAttachments(repo, batch),
		LockRepositories:   !dryRun,
	}
}

// phasePreMigration runs pre-migration validation and discovery.
// Phase 1: Validates repository and runs optional pre-migration discovery.
func (e *Executor) phasePreMigration(ctx context.Context, mc *MigrationContext) error {
	e.logger.Info("Running pre-migration validation", "repo", mc.Repo.FullName)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "pre_migration", "validate", "Running pre-migration validation", nil)

	// Run discovery on source repository for production migrations to get latest stats
	if !mc.DryRun {
		e.logger.Info("Running pre-migration discovery to refresh repository data", "repo", mc.Repo.FullName)
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "pre_migration", "discovery", "Refreshing repository characteristics", nil)

		if err := e.runPreMigrationDiscovery(ctx, mc.Repo); err != nil {
			// Log warning but don't fail migration
			errMsg := err.Error()
			e.logger.Warn("Pre-migration discovery failed, continuing with existing data",
				"repo", mc.Repo.FullName,
				"error", err)
			e.logOperation(ctx, mc.Repo, mc.HistoryID, "WARN", "pre_migration", "discovery", "Pre-migration discovery failed", &errMsg)
		} else {
			e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "pre_migration", "discovery", "Repository data refreshed successfully", nil)
		}
	}

	if err := e.validatePreMigration(ctx, mc.Repo, mc.Batch); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "pre_migration", "validate", "Pre-migration validation failed", &errMsg)
		return fmt.Errorf("pre-migration validation failed: %w", err)
	}

	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "pre_migration", "validate", "Pre-migration validation passed", nil)

	// Update status
	status := models.StatusPreMigration
	if mc.DryRun {
		status = models.StatusDryRunInProgress
	}
	mc.Repo.Status = string(status)
	if err := e.storage.UpdateRepository(ctx, mc.Repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	return nil
}

// phaseArchiveGeneration generates migration archives on source (GHES).
// Phase 2: Creates git and metadata archives for migration.
func (e *Executor) phaseArchiveGeneration(ctx context.Context, mc *MigrationContext) error {
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

	mc.Repo.Status = string(models.StatusArchiveGenerating)
	// Track migration ID and lock status for production migrations (use git archive ID as primary)
	migID := archiveIDs.GitArchiveID
	mc.Repo.SourceMigrationID = &migID
	mc.Repo.IsSourceLocked = mc.LockRepositories
	if err := e.storage.UpdateRepository(ctx, mc.Repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	return nil
}

// phaseArchivePolling polls for archive generation completion.
// Phase 3: Waits for both git and metadata archives to be ready.
func (e *Executor) phaseArchivePolling(ctx context.Context, mc *MigrationContext) error {
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

// phaseMigrationStart starts the migration on destination (GHEC).
// Phase 4: Initiates the repository migration using GraphQL API.
func (e *Executor) phaseMigrationStart(ctx context.Context, mc *MigrationContext) error {
	e.logger.Info("Starting migration on GHEC", "repo", mc.Repo.FullName)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration_start", "initiate", "Starting migration on destination", nil)

	migrationID, err := e.startRepositoryMigration(ctx, mc.Repo, mc.Batch, mc.ArchiveURLs)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "migration_start", "initiate", "Failed to start migration", &errMsg)
		return fmt.Errorf("failed to start migration: %w", err)
	}

	mc.MigrationID = migrationID
	details := fmt.Sprintf("Migration ID: %s", migrationID)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration_start", "initiate", "Migration started successfully", &details)

	mc.Repo.Status = string(models.StatusMigratingContent)
	if err := e.storage.UpdateRepository(ctx, mc.Repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	return nil
}

// phaseMigrationPolling polls for migration completion on destination.
// Phase 5: Waits for the migration to complete on GHEC.
func (e *Executor) phaseMigrationPolling(ctx context.Context, mc *MigrationContext) error {
	e.logger.Info("Polling migration status", "repo", mc.Repo.FullName, "migration_id", mc.MigrationID)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration_progress", "poll", "Polling for migration completion", nil)

	if err := e.pollMigrationStatus(ctx, mc.Repo, mc.Batch, mc.HistoryID, mc.MigrationID); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "migration_progress", "poll", "Migration failed", &errMsg)
		return fmt.Errorf("migration failed: %w", err)
	}

	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration_progress", "complete", "Migration completed successfully", nil)

	return nil
}

// phasePostMigration runs post-migration validation.
// Phase 6: Validates the migration was successful.
func (e *Executor) phasePostMigration(ctx context.Context, mc *MigrationContext) error {
	if !e.shouldRunPostMigration(mc.DryRun) {
		reason := fmt.Sprintf("Skipping post-migration validation (mode: %s, dry_run: %v)", e.postMigrationMode, mc.DryRun)
		e.logger.Info(reason, "repo", mc.Repo.FullName)
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "post_migration", "skip", reason, nil)
		return nil
	}

	e.logger.Info("Running post-migration validation", "repo", mc.Repo.FullName, "mode", e.postMigrationMode)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "post_migration", "validate", "Running post-migration validation", nil)

	if err := e.validatePostMigration(ctx, mc.Repo); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "WARN", "post_migration", "validate", "Post-migration validation failed", &errMsg)
		// Don't fail the migration on validation warnings
	} else {
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "post_migration", "validate", "Post-migration validation passed", nil)
	}

	return nil
}

// phaseCompletion marks the migration as complete.
// Phase 7: Updates status and unlocks source repository.
func (e *Executor) phaseCompletion(ctx context.Context, mc *MigrationContext) error {
	completionStatus := models.StatusComplete
	completionMsg := msgMigrationComplete
	// Clear lock status on successful completion
	mc.Repo.IsSourceLocked = false

	// Unlock source repository for production migrations
	if !mc.DryRun && mc.Repo.SourceMigrationID != nil {
		e.unlockSourceRepository(ctx, mc.Repo)
	}

	if mc.DryRun {
		completionStatus = models.StatusDryRunComplete
		completionMsg = msgDryRunComplete
	}

	e.logger.Info("Migration complete", "repo", mc.Repo.FullName, "dry_run", mc.DryRun)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration", "complete", completionMsg, nil)
	e.updateHistoryStatus(ctx, mc.HistoryID, "completed", nil)

	mc.Repo.Status = string(completionStatus)
	now := time.Now()

	// Set appropriate timestamps based on migration type
	if mc.DryRun {
		mc.Repo.LastDryRunAt = &now
	} else {
		mc.Repo.MigratedAt = &now
	}

	return e.storage.UpdateRepository(ctx, mc.Repo)
}

// handlePhaseError handles error recovery for a phase failure.
// This centralizes the error recovery logic used across all phases.
func (e *Executor) handlePhaseError(ctx context.Context, mc *MigrationContext, err error) {
	errMsg := err.Error()
	e.updateHistoryStatus(ctx, mc.HistoryID, statusFailed, &errMsg)

	status := models.StatusMigrationFailed
	if mc.DryRun {
		status = models.StatusDryRunFailed
	}
	mc.Repo.Status = string(status)

	// Unlock repository if it was locked
	if mc.LockRepositories && mc.Repo.SourceMigrationID != nil {
		mc.Repo.IsSourceLocked = false
		e.unlockSourceRepository(ctx, mc.Repo)
	}

	if updateErr := e.storage.UpdateRepository(ctx, mc.Repo); updateErr != nil {
		e.logger.Error("Failed to update repository status", "error", updateErr)
	}
}
