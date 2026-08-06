package migration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/elm"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// ELMClient is the subset of the elm CLI wrapper this service uses. *elm.Client
// satisfies it; tests drive a real *elm.Client over a fake elm.Transport so the
// exact command strings issued (and NOT issued) can be asserted.
type ELMClient interface {
	CreateMigration(ctx context.Context, req elm.CreateMigrationRequest) (*elm.MigrationRef, error)
	StartMigration(ctx context.Context, migrationID string) error
	GetMigrationStatus(ctx context.Context, migrationID string) (*elm.MigrationStatus, error)
	ListMigrations(ctx context.Context, cursor string) (*elm.ListPage, error)
	CutoverToDestination(ctx context.Context, migrationID string) error
	CancelMigration(ctx context.Context, migrationID string) error
}

// ELM service sentinel errors. Callers distinguish them with errors.Is.
var (
	// ErrELMAdmissionCeiling reports that admitting another live migration would
	// exceed a concurrency ceiling. The repository is deliberately left in its
	// existing (queued) state so it is retried later rather than failed.
	ErrELMAdmissionCeiling = errors.New("elm concurrency ceiling reached")

	// ErrELMNotReadyForCutover reports that a cutover was refused because the
	// migration is not reported as ready. Cutover is the one irreversible step in a
	// live migration, so this refusal commits no state and issues no command.
	ErrELMNotReadyForCutover = errors.New("elm migration is not ready for cutover")

	// ErrELMNotConfigured reports that a repository is routed to ELM but the
	// deployment has no ELM service configured.
	ErrELMNotConfigured = errors.New("elm is not configured for this deployment")

	// ErrELMNoRecord reports that no ELM migration record exists for a repository.
	ErrELMNoRecord = errors.New("no elm migration record for repository")
)

// Default ceilings published for the ELM preview.
const (
	defaultELMMaxConcurrentSource      = 10
	defaultELMMaxConcurrentDestination = 20
	defaultELMPollInterval             = 60 * time.Second
)

// ELMTarget is the destination coordinate for a live migration. It is resolved by
// the caller (the strategy, which knows about batches and visibility handling) so
// the service itself owns only ELM orchestration.
type ELMTarget struct {
	Org        string
	Repo       string
	Visibility string
}

// ELMServiceConfig configures the ELM service.
type ELMServiceConfig struct {
	Storage                  *storage.Database
	Client                   ELMClient
	Logger                   *slog.Logger
	PollInterval             time.Duration
	MaxConcurrentSource      int
	MaxConcurrentDestination int
}

// ELMService owns everything long-lived about Enterprise Live Migrations:
// admission control, the create/start lifecycle, the poll loop that advances a
// migration towards cutover readiness, restart reconciliation, and the gated
// cutover itself.
//
// The poll loop runs OUTSIDE the migration worker, so a repository sitting in
// awaiting_cutover for days consumes no worker slot.
type ELMService struct {
	storage      *storage.Database
	client       ELMClient
	logger       *slog.Logger
	pollInterval time.Duration

	// maxConcurrentSource is the per-source-instance ceiling. It is enforced against
	// the in-flight count grouped by the existing repositories.source_id FK.
	maxConcurrentSource int
	// maxConcurrentGlobal is the destination ceiling. There is exactly one
	// configured destination per deployment (config.Destination is a single
	// DestinationConfig with one base_url), so this is a GLOBAL in-flight count and
	// not a grouped one. If multi-destination support ever lands it must become
	// grouped.
	maxConcurrentGlobal int
}

// NewELMService creates an ELM service.
func NewELMService(cfg ELMServiceConfig) (*ELMService, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Client == nil {
		return nil, fmt.Errorf("elm client is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultELMPollInterval
	}
	maxSource := cfg.MaxConcurrentSource
	if maxSource <= 0 {
		maxSource = defaultELMMaxConcurrentSource
	}
	maxGlobal := cfg.MaxConcurrentDestination
	if maxGlobal <= 0 {
		maxGlobal = defaultELMMaxConcurrentDestination
	}

	return &ELMService{
		storage:             cfg.Storage,
		client:              cfg.Client,
		logger:              logger,
		pollInterval:        pollInterval,
		maxConcurrentSource: maxSource,
		maxConcurrentGlobal: maxGlobal,
	}, nil
}

// PollInterval returns the configured poll interval.
func (s *ELMService) PollInterval() time.Duration { return s.pollInterval }

// Preflight proves the appliance is reachable and the credentials work WITHOUT
// creating anything. It issues `elm migration list` and runs the repository
// eligibility checks; this is the whole of an ELM dry run.
func (s *ELMService) Preflight(ctx context.Context, repo *models.Repository) error {
	if err := ELMEligibility(repo); err != nil {
		return err
	}
	if _, err := s.client.ListMigrations(ctx, ""); err != nil {
		return fmt.Errorf("elm preflight failed: %w", err)
	}
	return nil
}

// ELMEligibility reports why a repository cannot take the ELM corridor, or nil.
// ELM carries GHES -> GHE.com repositories only.
func ELMEligibility(repo *models.Repository) error {
	if repo == nil {
		return fmt.Errorf("repository is required")
	}
	if !repo.IsELMRouted() {
		return fmt.Errorf("repository %s is not routed to ELM (route=%s)", repo.FullName, repo.GetMigrationRoute())
	}
	if ado := repo.GetADOProject(); ado != nil && *ado != "" {
		return fmt.Errorf("repository %s is an Azure DevOps repository; ELM covers GHES sources only", repo.FullName)
	}
	if repo.Source != models.SourceGHES {
		return fmt.Errorf("repository %s has source %q; ELM covers GHES sources only", repo.FullName, repo.Source)
	}
	return nil
}

// checkAdmission refuses a new live migration that would breach either ceiling.
//
// Both counts come from one query so they are always coherent: the per-source
// count is grouped by repositories.source_id (a repository with a NULL source_id
// is counted under storage.ELMUnknownSourceBucket rather than dropped), and the
// destination count is global because the deployment targets one destination.
func (s *ELMService) checkAdmission(ctx context.Context, repo *models.Repository) error {
	counts, err := s.storage.CountELMMigrationsInFlight(ctx)
	if err != nil {
		return fmt.Errorf("failed to read in-flight ELM counts: %w", err)
	}

	if counts.Global >= s.maxConcurrentGlobal {
		return fmt.Errorf("%w: %d live migrations already in flight for the destination (limit %d)",
			ErrELMAdmissionCeiling, counts.Global, s.maxConcurrentGlobal)
	}
	if inSource := counts.CountForSource(repo.SourceID); inSource >= s.maxConcurrentSource {
		return fmt.Errorf("%w: %d live migrations already in flight for this source (limit %d)",
			ErrELMAdmissionCeiling, inSource, s.maxConcurrentSource)
	}
	return nil
}

// StartBackfill admits, creates and starts a live migration for a repository and
// moves it to `syncing`. The poll loop owns every transition from here on.
//
// When a ceiling refuses admission the repository status is left EXACTLY as it
// was, so a queued repository stays queued and is retried rather than failed.
func (s *ELMService) StartBackfill(ctx context.Context, repo *models.Repository, target ELMTarget) (string, error) {
	if err := ELMEligibility(repo); err != nil {
		return "", err
	}
	if err := s.checkAdmission(ctx, repo); err != nil {
		return "", err
	}

	ref, err := s.client.CreateMigration(ctx, elm.CreateMigrationRequest{
		SourceOrg:  repo.Organization(),
		SourceRepo: repo.Name(),
		TargetOrg:  target.Org,
		TargetRepo: target.Repo,
		Visibility: target.Visibility,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create ELM migration: %w", err)
	}

	if err := s.client.StartMigration(ctx, ref.ID); err != nil {
		// The migration exists on the appliance; persist the handle (best effort --
		// the repository stays queued, never syncing, so a lost record here cannot
		// strand it) so a later reconcile can adopt or cancel it rather than orphaning it.
		_ = s.persistRecord(ctx, repo.ID, ref.ID, elm.StateQueued, "", nil, false, errString(err))
		return "", fmt.Errorf("failed to start ELM migration %s: %w", ref.ID, err)
	}

	// The durable elm_migrations row is the ONLY handle the poll loop and restart
	// reconciliation have on this live migration. If it cannot be written we must NOT
	// advance the repository to syncing: an in-flight status with no record strands
	// the repository where nothing can adopt or advance it. Leave it in its existing
	// (queued) state so it is retried, and surface the failure.
	if err := s.persistRecord(ctx, repo.ID, ref.ID, elm.StateQueued, "", nil, false, nil); err != nil {
		return "", fmt.Errorf("failed to persist ELM migration handle for %s (elm id %s): %w", repo.FullName, ref.ID, err)
	}

	repo.Status = string(models.StatusSyncing)
	if err := s.storage.UpdateRepository(ctx, repo); err != nil {
		return ref.ID, fmt.Errorf("failed to record syncing status: %w", err)
	}

	s.logger.Info("ELM backfill started",
		"repo", repo.FullName,
		"elm_migration_id", ref.ID,
		"target", target.Org+"/"+target.Repo)
	return ref.ID, nil
}

// persistRecord upserts the ELM record for a repository and returns the storage
// error so callers can decide whether it is fatal. It is the durable handle for
// the whole live-migration lifecycle, so StartBackfill treats a failure here as
// fatal and refuses to advance the repository into an in-flight status.
func (s *ELMService) persistRecord(ctx context.Context, repoID int64, elmID, state, phase string, progress *int, ready bool, lastErr *string) error {
	now := time.Now()
	rec := &models.ELMMigration{
		RepositoryID:    repoID,
		ELMMigrationID:  elmID,
		ELMStatus:       state,
		ELMPhase:        phase,
		CutoverReady:    ready,
		ProgressPercent: progress,
		LastPolledAt:    &now,
		LastError:       lastErr,
	}
	if err := s.storage.UpsertELMMigration(ctx, rec); err != nil {
		s.logger.Error("Failed to persist ELM migration record", "error", err, "elm_migration_id", elmID)
		return err
	}
	return nil
}

func errString(err error) *string {
	if err == nil {
		return nil
	}
	msg := err.Error()
	return &msg
}

// Run reconciles once and then polls on the configured interval until ctx ends.
// It takes no MigrationWorker slot.
func (s *ELMService) Run(ctx context.Context) {
	if err := s.Reconcile(ctx); err != nil {
		s.logger.Error("ELM reconcile failed at startup", "error", err)
	}

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	s.logger.Info("ELM poll loop started", "interval", s.pollInterval)
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("ELM poll loop stopped")
			return
		case <-ticker.C:
			if err := s.PollOnce(ctx); err != nil {
				s.logger.Error("ELM poll failed", "error", err)
			}
		}
	}
}

// PollOnce advances every in-flight live migration by one status read.
func (s *ELMService) PollOnce(ctx context.Context) error {
	records, err := s.storage.ListELMMigrationsInFlight(ctx)
	if err != nil {
		return fmt.Errorf("failed to list in-flight ELM migrations: %w", err)
	}
	for _, rec := range records {
		s.pollRecord(ctx, rec)
	}
	return nil
}

// pollRecord reads one migration's status and applies it.
//
// The three failure modes are handled distinctly and none of them ever writes an
// optimistic readiness:
//   - transport failure: retryable, record the error and change NOTHING else;
//   - non-zero exit: the appliance reports the migration failed, so the repository
//     moves to migration_failed;
//   - unparseable output: the preview CLI drifted, so the error is recorded and
//     cutover_ready is left UNCHANGED rather than written false.
func (s *ELMService) pollRecord(ctx context.Context, rec *models.ELMMigration) {
	status, err := s.client.GetMigrationStatus(ctx, rec.ELMMigrationID)
	if err != nil {
		s.handlePollError(ctx, rec, err)
		return
	}
	s.applyStatus(ctx, rec, status)
}

// handlePollError records a poll failure without ever inventing readiness.
func (s *ELMService) handlePollError(ctx context.Context, rec *models.ELMMigration, err error) {
	var transportErr *elm.TransportError
	var parseErr *elm.ParseError
	var cmdErr *elm.CommandError

	switch {
	case errors.As(err, &transportErr):
		s.logger.Warn("ELM status unreachable; leaving state unchanged",
			"elm_migration_id", rec.ELMMigrationID, "error", err)
		s.recordPollError(ctx, rec, err)
	case errors.As(err, &parseErr):
		// Unreadable output must not read as "not ready yet": cutover_ready keeps
		// whatever it was, and the operator sees last_error.
		s.logger.Error("ELM status output unparseable; readiness left unchanged",
			"elm_migration_id", rec.ELMMigrationID, "error", err)
		s.recordPollError(ctx, rec, err)
	case errors.As(err, &cmdErr):
		s.logger.Error("ELM status command failed on the appliance",
			"elm_migration_id", rec.ELMMigrationID, "exit_code", cmdErr.ExitCode, "error", err)
		s.recordPollError(ctx, rec, err)
		s.setRepositoryStatus(ctx, rec.RepositoryID, models.StatusMigrationFailed)
	default:
		s.logger.Error("ELM status failed", "elm_migration_id", rec.ELMMigrationID, "error", err)
		s.recordPollError(ctx, rec, err)
	}
}

// recordPollError stamps last_error and last_polled_at, leaving every other field
// (notably cutover_ready) exactly as persisted.
func (s *ELMService) recordPollError(ctx context.Context, rec *models.ELMMigration, err error) {
	now := time.Now()
	rec.LastPolledAt = &now
	rec.LastError = errString(err)
	if upErr := s.storage.UpsertELMMigration(ctx, rec); upErr != nil {
		s.logger.Error("Failed to record ELM poll error", "error", upErr)
	}
}

// applyStatus persists a fresh status and advances the repository lifecycle.
func (s *ELMService) applyStatus(ctx context.Context, rec *models.ELMMigration, status *elm.MigrationStatus) {
	now := time.Now()
	rec.ELMStatus = status.State
	rec.ELMPhase = status.Phase
	rec.ProgressPercent = status.ProgressPercent
	rec.CutoverReady = status.CutoverReady
	rec.LastPolledAt = &now
	rec.LastError = nil
	if err := s.storage.UpsertELMMigration(ctx, rec); err != nil {
		s.logger.Error("Failed to persist ELM status", "error", err, "elm_migration_id", rec.ELMMigrationID)
		return
	}

	if next := repositoryStatusForELM(status); next != "" {
		s.setRepositoryStatus(ctx, rec.RepositoryID, next)
	}
}

// repositoryStatusForELM maps an ELM state onto the repository lifecycle status.
// An empty result means "leave the repository where it is".
func repositoryStatusForELM(status *elm.MigrationStatus) models.MigrationStatus {
	switch status.State {
	case elm.StateQueued, elm.StateBackfilling:
		if status.CutoverReady {
			return models.StatusAwaitingCutover
		}
		return models.StatusSyncing
	case elm.StateReadyForCutover:
		return models.StatusAwaitingCutover
	case elm.StateCuttingOver:
		return models.StatusCuttingOver
	case elm.StateCompleted:
		return models.StatusMigrationComplete
	case elm.StateFailed, elm.StateCanceled:
		return models.StatusMigrationFailed
	default:
		return ""
	}
}

// setRepositoryStatus loads, mutates and persists a repository's status.
func (s *ELMService) setRepositoryStatus(ctx context.Context, repoID int64, status models.MigrationStatus) {
	repo, err := s.storage.GetRepositoryByID(ctx, repoID)
	if err != nil || repo == nil {
		s.logger.Error("Failed to load repository for ELM status update", "error", err, "repository_id", repoID)
		return
	}
	if repo.Status == string(status) {
		return
	}
	repo.Status = string(status)
	if status == models.StatusMigrationComplete {
		now := time.Now()
		repo.MigratedAt = &now
	}
	if err := s.storage.UpdateRepository(ctx, repo); err != nil {
		s.logger.Error("Failed to update repository status from ELM poll", "error", err, "repository_id", repoID)
	}
}

// Reconcile re-adopts live migrations that survived a restart.
//
// Every persisted in-flight record is cross-referenced against `elm migration
// list`: a record the appliance still knows is adopted (its status is refreshed
// from the listing), and a record the appliance no longer knows is marked failed
// with a reason rather than left polling an id that will never resolve.
func (s *ELMService) Reconcile(ctx context.Context) error {
	known, err := s.listAllMigrations(ctx)
	if err != nil {
		return err
	}

	records, err := s.storage.ListELMMigrationsInFlight(ctx)
	if err != nil {
		return fmt.Errorf("failed to list in-flight ELM migrations: %w", err)
	}

	for _, rec := range records {
		summary, ok := known[rec.ELMMigrationID]
		if !ok {
			reason := fmt.Sprintf("ELM migration %s is no longer known to the appliance; marked failed during restart reconciliation", rec.ELMMigrationID)
			s.logger.Error("Adopting orphaned ELM migration as failed", "elm_migration_id", rec.ELMMigrationID)
			now := time.Now()
			rec.LastPolledAt = &now
			rec.LastError = &reason
			rec.ELMStatus = elm.StateFailed
			if err := s.storage.UpsertELMMigration(ctx, rec); err != nil {
				s.logger.Error("Failed to record orphaned ELM migration", "error", err)
			}
			s.setRepositoryStatus(ctx, rec.RepositoryID, models.StatusMigrationFailed)
			continue
		}

		s.logger.Info("Re-adopted in-flight ELM migration", "elm_migration_id", rec.ELMMigrationID, "state", summary.State)
		s.applyStatus(ctx, rec, &elm.MigrationStatus{
			ID:           summary.ID,
			State:        summary.State,
			CutoverReady: summary.CutoverReady,
		})
	}
	return nil
}

// listAllMigrations pages through `elm migration list`.
func (s *ELMService) listAllMigrations(ctx context.Context) (map[string]elm.MigrationSummary, error) {
	known := make(map[string]elm.MigrationSummary)
	cursor := ""
	for {
		page, err := s.client.ListMigrations(ctx, cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to list ELM migrations: %w", err)
		}
		for _, m := range page.Migrations {
			known[m.ID] = m
		}
		if page.NextPage == "" || page.NextPage == cursor {
			return known, nil
		}
		cursor = page.NextPage
	}
}

// GetRecord returns the persisted ELM record for a repository, or nil.
func (s *ELMService) GetRecord(ctx context.Context, repoID int64) (*models.ELMMigration, error) {
	return s.storage.GetELMMigration(ctx, repoID)
}

// Cutover performs the one irreversible step of a live migration.
//
// It is reachable only from the operator-triggered API path; nothing transitions
// into cutover automatically. Readiness is checked TWICE -- once against the
// persisted record and once against a fresh status call -- and a refusal issues no
// cutover command and commits no state change. --force is structurally
// unreachable: elm.Client.CutoverToDestination takes no force parameter.
func (s *ELMService) Cutover(ctx context.Context, repo *models.Repository) error {
	rec, err := s.storage.GetELMMigration(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to load ELM migration record: %w", err)
	}
	if rec == nil {
		return fmt.Errorf("%w: %s", ErrELMNoRecord, repo.FullName)
	}

	// Gate 1: the persisted record must report readiness.
	if !rec.CutoverReady {
		return fmt.Errorf("%w: %s reports elm status %q", ErrELMNotReadyForCutover, repo.FullName, rec.ELMStatus)
	}

	// Gate 2: re-confirm with a fresh status call so a stale record cannot cut over.
	status, err := s.client.GetMigrationStatus(ctx, rec.ELMMigrationID)
	if err != nil {
		return fmt.Errorf("failed to re-confirm ELM cutover readiness: %w", err)
	}
	if !status.CutoverReady {
		s.applyStatus(ctx, rec, status)
		return fmt.Errorf("%w: %s reports elm status %q on re-confirmation", ErrELMNotReadyForCutover, repo.FullName, status.State)
	}

	repo.Status = string(models.StatusCuttingOver)
	if err := s.storage.UpdateRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to record cutting_over status: %w", err)
	}

	if err := s.client.CutoverToDestination(ctx, rec.ELMMigrationID); err != nil {
		// The cutover never started; put the repository back where it was so the
		// operator can retry rather than leaving it stuck mid-flight.
		repo.Status = string(models.StatusAwaitingCutover)
		if updErr := s.storage.UpdateRepository(ctx, repo); updErr != nil {
			s.logger.Error("Failed to restore awaiting_cutover status", "error", updErr)
		}
		s.recordPollError(ctx, rec, err)
		return fmt.Errorf("elm cutover failed for %s: %w", repo.FullName, err)
	}

	rec.ELMStatus = elm.StateCuttingOver
	now := time.Now()
	rec.LastPolledAt = &now
	rec.LastError = nil
	if err := s.storage.UpsertELMMigration(ctx, rec); err != nil {
		s.logger.Error("Failed to persist cutting_over ELM record", "error", err)
	}

	s.logger.Info("ELM cutover issued", "repo", repo.FullName, "elm_migration_id", rec.ELMMigrationID)
	return nil
}

// Cancel aborts an in-flight live migration. Cancelling before cutover has no
// impact on the source repository, which stays writable throughout.
func (s *ELMService) Cancel(ctx context.Context, repo *models.Repository) error {
	rec, err := s.storage.GetELMMigration(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to load ELM migration record: %w", err)
	}
	if rec == nil {
		return fmt.Errorf("%w: %s", ErrELMNoRecord, repo.FullName)
	}
	if err := s.client.CancelMigration(ctx, rec.ELMMigrationID); err != nil {
		return fmt.Errorf("elm cancel failed for %s: %w", repo.FullName, err)
	}
	rec.ELMStatus = elm.StateCanceled
	rec.CutoverReady = false
	if err := s.storage.UpsertELMMigration(ctx, rec); err != nil {
		s.logger.Error("Failed to persist cancelled ELM record", "error", err)
	}
	s.setRepositoryStatus(ctx, repo.ID, models.StatusQueuedForMigration)
	return nil
}
