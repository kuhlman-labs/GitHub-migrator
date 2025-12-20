package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// ExecuteWithStrategy executes a migration using the appropriate strategy based on the repository source.
// This is the unified entry point that automatically selects between GitHub and ADO migration strategies.
//
// The migration proceeds through these common phases:
//  1. Strategy selection and source validation
//  2. Pre-migration validation and discovery
//  3. Archive preparation (source-specific: GitHub generates archives, ADO skips)
//  4. Migration start (source-specific: different GraphQL mutations)
//  5. Migration status polling
//  6. Post-migration validation
//  7. Completion and cleanup
func (e *Executor) ExecuteWithStrategy(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	// Create strategy registry and get appropriate strategy
	registry := NewStrategyRegistry(
		NewGitHubMigrationStrategy(e),
		NewADOMigrationStrategy(e),
	)

	strategy := registry.GetStrategy(repo)
	if strategy == nil {
		return fmt.Errorf("no migration strategy found for repository %s", repo.FullName)
	}

	e.logger.Info("Selected migration strategy",
		"repo", repo.FullName,
		"strategy", strategy.Name(),
		"dry_run", dryRun)

	// Create migration context
	mc := e.NewMigrationContext(repo, batch, dryRun)

	e.logger.Info("Starting migration",
		"repo", repo.FullName,
		"strategy", strategy.Name(),
		"dry_run", dryRun,
		"has_batch", batch != nil)

	// Log all migration flags for observability and audit
	e.logger.Info("Migration flags",
		"repo", repo.FullName,
		"dry_run", dryRun,
		"lock_repositories", mc.LockRepositories,
		"exclude_releases", mc.ExcludeReleases,
		"exclude_attachments", mc.ExcludeAttachments,
		"strategy", strategy.Name())

	// Create migration history record
	historyID, err := e.createMigrationHistory(ctx, repo, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
	}
	mc.HistoryID = historyID

	// Log operation start
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "start",
		fmt.Sprintf("Starting %s using %s strategy",
			map[bool]string{true: "dry run", false: "migration"}[dryRun],
			strategy.Name()), nil)

	// Log migration flags to history for audit trail
	flagsDetails := fmt.Sprintf("strategy=%s, lock_repositories=%v, exclude_releases=%v, exclude_attachments=%v",
		strategy.Name(), mc.LockRepositories, mc.ExcludeReleases, mc.ExcludeAttachments)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "flags", "Migration flags configured", &flagsDetails)

	// Phase 1: Source validation (strategy-specific)
	if err := e.executeSourceValidation(ctx, mc, strategy); err != nil {
		e.handleStrategyPhaseError(ctx, mc, strategy, err)
		return err
	}

	// Phase 2: Pre-migration validation (common) - reuse existing phase method
	if err := e.phasePreMigration(ctx, mc); err != nil {
		e.handleStrategyPhaseError(ctx, mc, strategy, err)
		return err
	}

	// Phase 3: Archive preparation (strategy-specific)
	if err := e.executeArchivePreparation(ctx, mc, strategy); err != nil {
		e.handleStrategyPhaseError(ctx, mc, strategy, err)
		return err
	}

	// Phase 4: Migration start (strategy-specific)
	migrationID, err := strategy.StartMigration(ctx, mc)
	if err != nil {
		e.handleStrategyPhaseError(ctx, mc, strategy, err)
		return err
	}
	mc.MigrationID = migrationID

	// Phase 5: Migration polling (common) - reuse existing phase method
	if err := e.phaseMigrationPolling(ctx, mc); err != nil {
		e.handleStrategyPhaseError(ctx, mc, strategy, err)
		return err
	}

	// Phase 6: Post-migration validation (common, errors logged but don't fail) - reuse existing phase method
	if err := e.phasePostMigration(ctx, mc); err != nil {
		e.logger.Warn("Post-migration phase returned error", "error", err, "repo", repo.FullName)
	}

	// Phase 7: Completion (strategy-aware)
	return e.executeCompletion(ctx, mc, strategy)
}

// executeSourceValidation runs strategy-specific source validation.
func (e *Executor) executeSourceValidation(ctx context.Context, mc *MigrationContext, strategy MigrationStrategy) error {
	e.logger.Info("Validating source access", "repo", mc.Repo.FullName, "strategy", strategy.Name())
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "source_validation", "validate",
		fmt.Sprintf("Validating source access using %s strategy", strategy.Name()), nil)

	if err := strategy.ValidateSource(ctx, mc.Repo); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "source_validation", "validate",
			"Source validation failed", &errMsg)
		return fmt.Errorf("source validation failed: %w", err)
	}

	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "source_validation", "validate",
		"Source validation passed", nil)
	return nil
}

// executeArchivePreparation runs strategy-specific archive preparation.
func (e *Executor) executeArchivePreparation(ctx context.Context, mc *MigrationContext, strategy MigrationStrategy) error {
	return strategy.PrepareArchives(ctx, mc)
}

// executeCompletion marks the migration as complete with strategy-aware cleanup.
func (e *Executor) executeCompletion(ctx context.Context, mc *MigrationContext, strategy MigrationStrategy) error {
	completionStatus := models.StatusComplete
	completionMsg := msgMigrationComplete
	mc.Repo.IsSourceLocked = false

	// Unlock source repository if strategy supports it and this is a production migration
	if strategy.ShouldUnlockSource() && !mc.DryRun && mc.Repo.SourceMigrationID != nil {
		e.unlockSourceRepository(ctx, mc.Repo)
	}

	if mc.DryRun {
		completionStatus = models.StatusDryRunComplete
		completionMsg = msgDryRunComplete
	}

	e.logger.Info("Migration complete",
		"repo", mc.Repo.FullName,
		"strategy", strategy.Name(),
		"dry_run", mc.DryRun)
	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration", "complete", completionMsg, nil)
	e.updateHistoryStatus(ctx, mc.HistoryID, "completed", nil)

	mc.Repo.Status = string(completionStatus)
	now := time.Now()

	if mc.DryRun {
		mc.Repo.LastDryRunAt = &now
	} else {
		mc.Repo.MigratedAt = &now
	}

	return e.storage.UpdateRepository(ctx, mc.Repo)
}

// handleStrategyPhaseError handles error recovery for a phase failure with strategy context.
func (e *Executor) handleStrategyPhaseError(ctx context.Context, mc *MigrationContext, strategy MigrationStrategy, err error) {
	errMsg := err.Error()
	e.updateHistoryStatus(ctx, mc.HistoryID, statusFailed, &errMsg)

	status := models.StatusMigrationFailed
	if mc.DryRun {
		status = models.StatusDryRunFailed
	}
	mc.Repo.Status = string(status)

	// Unlock repository if strategy supports it and it was locked
	if strategy.ShouldUnlockSource() && mc.LockRepositories && mc.Repo.SourceMigrationID != nil {
		mc.Repo.IsSourceLocked = false
		e.unlockSourceRepository(ctx, mc.Repo)
	}

	if updateErr := e.storage.UpdateRepository(ctx, mc.Repo); updateErr != nil {
		e.logger.Error("Failed to update repository status", "error", updateErr)
	}
}
