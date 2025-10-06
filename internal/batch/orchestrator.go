package batch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// Orchestrator coordinates batch organization, scheduling, and execution
type Orchestrator struct {
	organizer *Organizer
	scheduler *Scheduler
	storage   *storage.Database
	logger    *slog.Logger
}

// OrchestratorConfig holds configuration for the orchestrator
type OrchestratorConfig struct {
	Storage  *storage.Database
	Executor MigrationExecutor
	Logger   *slog.Logger
}

// NewOrchestrator creates a new batch orchestrator
func NewOrchestrator(cfg OrchestratorConfig) (*Orchestrator, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Executor == nil {
		return nil, fmt.Errorf("executor is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create organizer
	organizer, err := NewOrganizer(OrganizerConfig{
		Storage: cfg.Storage,
		Logger:  cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create organizer: %w", err)
	}

	// Create scheduler
	scheduler, err := NewScheduler(SchedulerConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	return &Orchestrator{
		organizer: organizer,
		scheduler: scheduler,
		storage:   cfg.Storage,
		logger:    cfg.Logger,
	}, nil
}

// CreateAndExecutePilot creates a pilot batch and executes it
func (o *Orchestrator) CreateAndExecutePilot(ctx context.Context, name string, criteria PilotCriteria, dryRun bool) (*models.Batch, error) {
	o.logger.Info("Creating and executing pilot batch",
		"name", name,
		"dry_run", dryRun,
		"criteria", criteria)

	// Create pilot batch
	batch, repos, err := o.organizer.CreatePilotBatch(ctx, name, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to create pilot batch: %w", err)
	}

	o.logger.Info("Pilot batch created",
		"batch_id", batch.ID,
		"repo_count", len(repos))

	// Execute batch
	if err := o.scheduler.ExecuteBatch(ctx, batch.ID, dryRun); err != nil {
		return nil, fmt.Errorf("failed to execute pilot batch: %w", err)
	}

	o.logger.Info("Pilot batch execution started", "batch_id", batch.ID)

	return batch, nil
}

// CreateAndScheduleWaves creates migration waves and schedules them
func (o *Orchestrator) CreateAndScheduleWaves(ctx context.Context, criteria WaveCriteria, startTime time.Time, intervalHours int) ([]*models.Batch, error) {
	o.logger.Info("Creating and scheduling waves",
		"wave_size", criteria.WaveSize,
		"start_time", startTime,
		"interval_hours", intervalHours)

	// Create waves
	waves, err := o.organizer.OrganizeIntoWaves(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to organize waves: %w", err)
	}

	if len(waves) == 0 {
		return []*models.Batch{}, nil
	}

	o.logger.Info("Waves created", "wave_count", len(waves))

	// Schedule each wave with interval
	scheduleTime := startTime
	for i, wave := range waves {
		if err := o.scheduler.ScheduleBatch(ctx, wave.ID, scheduleTime); err != nil {
			o.logger.Error("Failed to schedule wave",
				"wave_number", i+1,
				"batch_id", wave.ID,
				"error", err)
		} else {
			o.logger.Info("Wave scheduled",
				"wave_number", i+1,
				"batch_id", wave.ID,
				"scheduled_at", scheduleTime)
		}

		scheduleTime = scheduleTime.Add(time.Duration(intervalHours) * time.Hour)
	}

	return waves, nil
}

// ExecuteScheduledBatches executes all batches that are scheduled to start
func (o *Orchestrator) ExecuteScheduledBatches(ctx context.Context, dryRun bool) error {
	o.logger.Info("Checking for scheduled batches", "dry_run", dryRun)

	// Get all batches
	batches, err := o.storage.ListBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed to list batches: %w", err)
	}

	now := time.Now()
	executed := 0

	for _, batch := range batches {
		// Check if batch is scheduled and ready to execute
		if batch.ScheduledAt != nil &&
			batch.Status == "ready" &&
			batch.ScheduledAt.Before(now) {

			o.logger.Info("Executing scheduled batch",
				"batch_id", batch.ID,
				"batch_name", batch.Name,
				"scheduled_at", batch.ScheduledAt)

			if err := o.scheduler.ExecuteBatch(ctx, batch.ID, dryRun); err != nil {
				o.logger.Error("Failed to execute scheduled batch",
					"batch_id", batch.ID,
					"error", err)
			} else {
				executed++
			}
		}
	}

	o.logger.Info("Scheduled batch execution complete", "executed_count", executed)

	return nil
}

// GetAllBatchProgress returns progress for all batches
func (o *Orchestrator) GetAllBatchProgress(ctx context.Context) ([]*BatchProgress, error) {
	batches, err := o.storage.ListBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	progress := make([]*BatchProgress, 0, len(batches))
	for _, batch := range batches {
		batchProgress, err := o.organizer.GetBatchProgress(ctx, batch.ID)
		if err != nil {
			o.logger.Error("Failed to get batch progress",
				"batch_id", batch.ID,
				"error", err)
			continue
		}
		progress = append(progress, batchProgress)
	}

	return progress, nil
}

// ExecuteBatch executes a specific batch
func (o *Orchestrator) ExecuteBatch(ctx context.Context, batchID int64, dryRun bool) error {
	return o.scheduler.ExecuteBatch(ctx, batchID, dryRun)
}

// CancelBatch cancels a running batch
func (o *Orchestrator) CancelBatch(ctx context.Context, batchID int64) error {
	return o.scheduler.CancelBatch(ctx, batchID)
}

// ScheduleBatch schedules a batch to execute at a specific time
func (o *Orchestrator) ScheduleBatch(ctx context.Context, batchID int64, scheduledAt time.Time) error {
	return o.scheduler.ScheduleBatch(ctx, batchID, scheduledAt)
}

// GetBatchProgress returns progress for a specific batch
func (o *Orchestrator) GetBatchProgress(ctx context.Context, batchID int64) (*BatchProgress, error) {
	return o.organizer.GetBatchProgress(ctx, batchID)
}

// SelectPilotRepositories selects repositories for pilot migration
func (o *Orchestrator) SelectPilotRepositories(ctx context.Context, criteria PilotCriteria) ([]*models.Repository, error) {
	return o.organizer.SelectPilotRepositories(ctx, criteria)
}

// GetRunningBatches returns all currently running batch IDs
func (o *Orchestrator) GetRunningBatches() []int64 {
	return o.scheduler.GetRunningBatches()
}

// IsBatchRunning checks if a specific batch is running
func (o *Orchestrator) IsBatchRunning(batchID int64) bool {
	return o.scheduler.IsBatchRunning(batchID)
}

// ExecuteSequentialWaves executes all waves in sequence
// Useful for controlled rollout where each wave must complete before the next
func (o *Orchestrator) ExecuteSequentialWaves(ctx context.Context, dryRun bool) error {
	o.logger.Info("Starting sequential wave execution", "dry_run", dryRun)

	// Get all wave batches
	batches, err := o.storage.ListBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed to list batches: %w", err)
	}

	// Filter and sort wave batches
	var waves []*models.Batch
	for _, batch := range batches {
		if batch.Type != "pilot" && batch.Type != "ready" && batch.Type != "" {
			waves = append(waves, batch)
		}
	}

	if len(waves) == 0 {
		o.logger.Info("No waves found to execute")
		return nil
	}

	// Extract batch IDs
	batchIDs := make([]int64, len(waves))
	for i, wave := range waves {
		batchIDs[i] = wave.ID
	}

	o.logger.Info("Executing waves sequentially", "wave_count", len(batchIDs))

	return o.scheduler.ExecuteSequentialBatches(ctx, batchIDs, dryRun)
}
