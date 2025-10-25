package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/batch"
)

// SchedulerWorker periodically checks for and executes scheduled batches
type SchedulerWorker struct {
	orchestrator *batch.Orchestrator
	logger       *slog.Logger
	interval     time.Duration
}

// NewSchedulerWorker creates a new scheduler worker
func NewSchedulerWorker(orchestrator *batch.Orchestrator, logger *slog.Logger) *SchedulerWorker {
	return &SchedulerWorker{
		orchestrator: orchestrator,
		logger:       logger,
		interval:     1 * time.Minute, // Check every minute
	}
}

// Start begins the scheduler worker loop
func (sw *SchedulerWorker) Start(ctx context.Context) {
	sw.logger.Info("Starting scheduler worker", "interval", sw.interval)

	ticker := time.NewTicker(sw.interval)
	defer ticker.Stop()

	// Run immediately on start
	sw.checkScheduledBatches(ctx)

	for {
		select {
		case <-ctx.Done():
			sw.logger.Info("Scheduler worker stopped")
			return
		case <-ticker.C:
			sw.checkScheduledBatches(ctx)
		}
	}
}

func (sw *SchedulerWorker) checkScheduledBatches(ctx context.Context) {
	sw.logger.Debug("Checking for scheduled batches")

	// Execute scheduled batches (dry_run=false for production migrations)
	if err := sw.orchestrator.ExecuteScheduledBatches(ctx, false); err != nil {
		sw.logger.Error("Failed to execute scheduled batches", "error", err)
	}
}

