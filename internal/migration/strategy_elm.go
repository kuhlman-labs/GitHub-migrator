package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// ELMStrategyName is the registry name of the Enterprise Live Migrations strategy.
const ELMStrategyName = "ELM"

// ELMStrategy implements MigrationStrategy for GitHub Enterprise Live Migrations,
// the near-zero-downtime GHES -> GHE.com corridor.
//
// It is THE SINGLE READER of the ELM route contract: a repository takes this
// strategy if and only if its stored repositories.migration_route column says
// "elm". It defines no route concept of its own and consults nothing else.
type ELMStrategy struct {
	executor *Executor
	service  *ELMService
}

// NewELMStrategy creates the ELM migration strategy. The service may be nil when
// ELM is not configured; ValidateSource then refuses the migration loudly rather
// than letting an explicitly ELM-routed repository fall through to GEI.
func NewELMStrategy(executor *Executor, service *ELMService) *ELMStrategy {
	return &ELMStrategy{executor: executor, service: service}
}

// Name returns the strategy name.
func (s *ELMStrategy) Name() string { return ELMStrategyName }

// SupportsRepository returns true if and only if the repository is ELM-routed,
// has no ADO project, and is sourced from GHES.
//
// The route check is load-bearing. This strategy is registered FIRST (GetStrategy
// returns the first match and the GitHub strategy matches every non-ADO
// repository), so without it every unrouted GHES repository would be claimed by
// ELM instead of falling through to GEI.
func (s *ELMStrategy) SupportsRepository(repo *models.Repository) bool {
	if repo == nil {
		return false
	}
	if !repo.IsELMRouted() {
		return false
	}
	if ado := repo.GetADOProject(); ado != nil && *ado != "" {
		return false
	}
	return repo.Source == models.SourceGHES
}

// ValidateSource checks that ELM is configured and the appliance is reachable.
func (s *ELMStrategy) ValidateSource(ctx context.Context, repo *models.Repository) error {
	if s.service == nil {
		return fmt.Errorf("%w: repository %s is routed to ELM but no ELM service is configured",
			ErrELMNotConfigured, repo.FullName)
	}
	if err := ELMEligibility(repo); err != nil {
		return err
	}
	if _, err := s.service.client.ListMigrations(ctx, ""); err != nil {
		return fmt.Errorf("elm appliance is not reachable: %w", err)
	}
	return nil
}

// PrepareArchives is a no-op. ELM backfills live from the appliance; it builds no
// migration archives.
func (s *ELMStrategy) PrepareArchives(ctx context.Context, mc *MigrationContext) error {
	if s.executor != nil {
		s.executor.logger.Info("Skipping archive generation for ELM migration (live backfill, no archives)",
			"repo", mc.Repo.FullName)
		s.executor.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "archive_generation", "skip",
			"ELM migrations do not require archive generation - the appliance backfills live", nil)
	}
	return nil
}

// StartMigration creates and starts the live migration and returns its ELM id.
// The repository is left in `syncing`; the ELM poll loop owns every transition
// from here, so no migration worker slot is held while the backfill runs.
func (s *ELMStrategy) StartMigration(ctx context.Context, mc *MigrationContext) (string, error) {
	if s.service == nil {
		return "", fmt.Errorf("%w: cannot start ELM migration for %s", ErrELMNotConfigured, mc.Repo.FullName)
	}

	e := s.executor
	target := ELMTarget{
		Org:        e.getDestinationOrg(mc.Repo, mc.Batch),
		Repo:       e.getDestinationRepoName(mc.Repo),
		Visibility: e.determineTargetVisibility(mc.Repo.Visibility),
	}

	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration", "initiate",
		fmt.Sprintf("Starting ELM live migration to %s/%s", target.Org, target.Repo), nil)

	migrationID, err := s.service.StartBackfill(ctx, mc.Repo, target)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, mc.Repo, mc.HistoryID, "ERROR", "migration", "initiate",
			"Failed to start ELM live migration", &errMsg)
		return "", err
	}

	e.logOperation(ctx, mc.Repo, mc.HistoryID, "INFO", "migration", "initiated",
		fmt.Sprintf("ELM migration started with ID: %s", migrationID), nil)
	return migrationID, nil
}

// ShouldUnlockSource returns false: ELM keeps the source writable until cutover,
// so there is nothing to unlock.
func (s *ELMStrategy) ShouldUnlockSource() bool { return false }

// DryRun runs the ELM preflight. ELM's CLI has no preflight-only verb, so a dry
// run deliberately CREATES NO MIGRATION: it issues `elm migration list` to prove
// the transport and credentials, runs the eligibility checks, and records
// dry_run_complete. No create, start or cutover command is ever issued.
func (s *ELMStrategy) DryRun(ctx context.Context, mc *MigrationContext) error {
	e := s.executor
	repo := mc.Repo

	if s.service == nil {
		return fmt.Errorf("%w: cannot dry run ELM migration for %s", ErrELMNotConfigured, repo.FullName)
	}

	e.logOperation(ctx, repo, mc.HistoryID, "INFO", "dry_run", "preflight",
		"Running ELM preflight (no migration is created)", nil)

	if err := s.service.Preflight(ctx, repo); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, mc.HistoryID, "ERROR", "dry_run", "preflight", "ELM preflight failed", &errMsg)
		repo.Status = string(models.StatusDryRunFailed)
		if updErr := e.storage.UpdateRepository(ctx, repo); updErr != nil {
			e.logger.Error("Failed to update repository status", "error", updErr)
		}
		e.updateHistoryStatus(ctx, mc.HistoryID, statusFailed, &errMsg)
		return err
	}

	now := time.Now()
	repo.Status = string(models.StatusDryRunComplete)
	repo.LastDryRunAt = &now
	repo.IsSourceLocked = false
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		return err
	}

	e.logOperation(ctx, repo, mc.HistoryID, "INFO", "dry_run", "complete",
		"ELM preflight passed - no live migration was created", nil)
	e.updateHistoryStatus(ctx, mc.HistoryID, "completed", nil)
	return nil
}
