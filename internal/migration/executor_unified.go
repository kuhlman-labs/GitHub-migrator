package migration

import (
	"context"
	"errors"
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
	return e.ExecuteWithStrategyAndELM(ctx, repo, batch, dryRun, nil)
}

// ExecuteWithStrategyAndELM is ExecuteWithStrategy with the deployment's ELM
// service supplied. The ELM service is owned by the ExecutorFactory (there is one
// per deployment, since ELM targets a single configured destination) rather than
// by the executor, so it is threaded through here instead of held on Executor.
// Passing nil registers an inert ELM strategy that refuses loudly if an
// ELM-routed repository reaches it.
func (e *Executor) ExecuteWithStrategyAndELM(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool, elmService *ELMService) error {
	registry := newMigrationStrategyRegistry(e, elmService)

	strategy := registry.GetStrategy(repo)
	if strategy == nil {
		return fmt.Errorf("no migration strategy found for repository %s", repo.FullName)
	}

	// ELM does not share the GEI phase pipeline: there are no archives to build and
	// the backfill is advanced by the long-lived ELM poll loop rather than by an
	// inline polling phase, so it gets its own short flow.
	if elmStrategy, ok := strategy.(*ELMStrategy); ok {
		return e.executeELM(ctx, elmStrategy, repo, batch, dryRun)
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
		fmt.Sprintf("Starting %s using %s strategy", runModeLabel(dryRun), strategy.Name()), nil)

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

// runModeLabel renders the human-readable label for a run mode.
func runModeLabel(dryRun bool) string {
	if dryRun {
		return "dry run"
	}
	return "migration"
}

// newMigrationStrategyRegistry builds the strategy registry in its runtime order.
//
// ELM MUST be registered FIRST: GetStrategy returns the first match and the
// GitHub strategy matches every non-ADO repository, so registering ELM later
// would make it permanently unreachable and the feature silently inert. Because
// ELMStrategy matches only on the recorded route, a repository with no route
// falls straight through to the GitHub/GEI strategy.
func newMigrationStrategyRegistry(e *Executor, elmService *ELMService) *StrategyRegistry {
	return NewStrategyRegistry(
		NewELMStrategy(e, elmService),
		NewGitHubMigrationStrategy(e),
		NewADOMigrationStrategy(e),
	)
}

// executeELM runs the Enterprise Live Migrations flow.
//
// A production run validates the source, then creates and starts the backfill and
// RETURNS -- the repository is left in `syncing` and the ELM poll loop advances it
// to awaiting_cutover, where it waits indefinitely on a deliberate operator
// cutover without holding a migration worker slot.
//
// A dry run is preflight-only and creates no migration at all.
func (e *Executor) executeELM(ctx context.Context, strategy *ELMStrategy, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	mc := e.NewMigrationContext(repo, batch, dryRun)
	// ELM keeps the source writable until cutover, so nothing is ever locked.
	mc.LockRepositories = false

	historyID, err := e.createMigrationHistory(ctx, repo, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
	}
	mc.HistoryID = historyID

	e.logOperation(ctx, repo, historyID, "INFO", "migration", "start",
		fmt.Sprintf("Starting %s using %s strategy", runModeLabel(dryRun), strategy.Name()), nil)

	if err := e.executeSourceValidation(ctx, mc, strategy); err != nil {
		e.handleStrategyPhaseError(ctx, mc, strategy, err)
		return err
	}

	if dryRun {
		return strategy.DryRun(ctx, mc)
	}

	migrationID, err := strategy.StartMigration(ctx, mc)
	if err != nil {
		e.handleStrategyPhaseError(ctx, mc, strategy, err)
		return err
	}
	mc.MigrationID = migrationID

	e.logger.Info("ELM live migration started; poll loop owns the remaining lifecycle",
		"repo", repo.FullName, "elm_migration_id", migrationID)
	return nil
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

	mc.Repo.Status = string(completionStatus)
	now := time.Now()

	if mc.DryRun {
		mc.Repo.LastDryRunAt = &now
	} else {
		mc.Repo.MigratedAt = &now
	}

	// Persist the repo update first so that the history status is only marked
	// "completed" when the repo status has actually been committed.
	if err := e.storage.UpdateRepository(ctx, mc.Repo); err != nil {
		return err
	}
	e.updateHistoryStatus(ctx, mc.HistoryID, "completed", nil)
	return nil
}

// handleStrategyPhaseError handles error recovery for a phase failure with strategy context.
func (e *Executor) handleStrategyPhaseError(ctx context.Context, mc *MigrationContext, strategy MigrationStrategy, err error) {
	errMsg := err.Error()
	e.updateHistoryStatus(ctx, mc.HistoryID, statusFailed, &errMsg)

	status := models.StatusMigrationFailed
	if mc.DryRun {
		status = models.StatusDryRunFailed
	}
	// An ELM concurrency ceiling is a capacity refusal, not a migration failure:
	// the repository goes back to queued so it is admitted on a later pass instead
	// of needing an operator to reset it.
	if errors.Is(err, ErrELMAdmissionCeiling) {
		status = models.StatusQueuedForMigration
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
