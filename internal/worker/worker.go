package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/migration"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// MigrationWorker polls for queued repositories and executes migrations
type MigrationWorker struct {
	executor     *migration.Executor
	storage      *storage.Database
	logger       *slog.Logger
	pollInterval time.Duration
	workers      int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
	active map[int64]bool // Track active migrations
}

// WorkerConfig configures the migration worker
type WorkerConfig struct {
	Executor     *migration.Executor
	Storage      *storage.Database
	Logger       *slog.Logger
	PollInterval time.Duration
	Workers      int // Number of parallel migration workers
}

// NewMigrationWorker creates a new migration worker
func NewMigrationWorker(cfg WorkerConfig) (*MigrationWorker, error) {
	if cfg.Executor == nil {
		return nil, fmt.Errorf("executor is required")
	}
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 30 * time.Second
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 5
	}

	return &MigrationWorker{
		executor:     cfg.Executor,
		storage:      cfg.Storage,
		logger:       cfg.Logger,
		pollInterval: cfg.PollInterval,
		workers:      cfg.Workers,
		active:       make(map[int64]bool),
	}, nil
}

// Start starts the migration worker
func (w *MigrationWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.ctx != nil {
		w.mu.Unlock()
		return fmt.Errorf("worker already started")
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.mu.Unlock()

	w.logger.Info("Starting migration worker",
		"poll_interval", w.pollInterval,
		"workers", w.workers)

	// Start the polling loop
	w.wg.Add(1)
	go w.pollLoop()

	return nil
}

// Stop stops the migration worker and waits for all migrations to complete
func (w *MigrationWorker) Stop() error {
	w.mu.Lock()
	if w.cancel == nil {
		w.mu.Unlock()
		return fmt.Errorf("worker not started")
	}
	w.cancel()
	w.mu.Unlock()

	w.logger.Info("Stopping migration worker, waiting for active migrations to complete...")

	// Wait for all workers to finish
	w.wg.Wait()

	w.logger.Info("Migration worker stopped")
	return nil
}

// pollLoop continuously polls for queued repositories
func (w *MigrationWorker) pollLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Process immediately on start
	w.processQueuedRepositories()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Poll loop stopped")
			return
		case <-ticker.C:
			w.processQueuedRepositories()
		}
	}
}

// processQueuedRepositories fetches queued repositories and dispatches them to workers
func (w *MigrationWorker) processQueuedRepositories() {
	ctx := context.Background()

	// Get count of currently active migrations
	w.mu.RLock()
	activeCount := len(w.active)
	w.mu.RUnlock()

	// Calculate available worker slots
	availableSlots := w.workers - activeCount
	if availableSlots <= 0 {
		w.logger.Debug("All worker slots busy",
			"active", activeCount,
			"max_workers", w.workers)
		return
	}

	// Fetch queued repositories (limit to available slots)
	filters := map[string]interface{}{
		"status": []string{
			string(models.StatusQueuedForMigration),
			string(models.StatusDryRunQueued),
		},
		"limit": availableSlots,
		"order": "priority DESC, created_at ASC", // High priority first, then FIFO
	}

	repos, err := w.storage.ListRepositories(ctx, filters)
	if err != nil {
		w.logger.Error("Failed to fetch queued repositories", "error", err)
		return
	}

	if len(repos) == 0 {
		w.logger.Debug("No queued repositories found")
		return
	}

	w.logger.Info("Found queued repositories",
		"count", len(repos),
		"available_slots", availableSlots)

	// Dispatch each repository to a worker
	for _, repo := range repos {
		// Check if already processing (shouldn't happen, but defensive)
		w.mu.RLock()
		if w.active[repo.ID] {
			w.mu.RUnlock()
			continue
		}
		w.mu.RUnlock()

		// Mark as active
		w.mu.Lock()
		w.active[repo.ID] = true
		w.mu.Unlock()

		// Start migration in background
		w.wg.Add(1)
		go w.executeMigration(repo)
	}
}

// executeMigration executes a single migration
func (w *MigrationWorker) executeMigration(repo *models.Repository) {
	defer w.wg.Done()
	defer func() {
		// Remove from active list
		w.mu.Lock()
		delete(w.active, repo.ID)
		w.mu.Unlock()
	}()

	// Determine if this is a dry run
	dryRun := repo.Status == string(models.StatusDryRunQueued)

	w.logger.Info("Starting migration execution",
		"repo", repo.FullName,
		"repo_id", repo.ID,
		"dry_run", dryRun,
		"has_batch", repo.BatchID != nil)

	// Create context for this migration execution
	ctx := context.Background()

	// Fetch batch details if repository is part of a batch
	var batch *models.Batch
	if repo.BatchID != nil {
		fetchedBatch, err := w.storage.GetBatch(ctx, *repo.BatchID)
		if err != nil {
			w.logger.Warn("Failed to fetch batch for repository",
				"repo", repo.FullName,
				"batch_id", *repo.BatchID,
				"error", err)
			// Continue without batch - repo will use its own settings or defaults
		} else {
			batch = fetchedBatch
			w.logger.Debug("Fetched batch settings for repository",
				"repo", repo.FullName,
				"batch_name", batch.Name,
				"destination_org", batch.DestinationOrg,
				"exclude_releases", batch.ExcludeReleases)
		}
	}

	// Update status to in-progress
	statusUpdate := models.StatusMigratingContent
	if dryRun {
		statusUpdate = models.StatusDryRunInProgress
	}
	repo.Status = string(statusUpdate)
	if err := w.storage.UpdateRepository(ctx, repo); err != nil {
		w.logger.Error("Failed to update repository status",
			"repo", repo.FullName,
			"error", err)
		// Continue with migration anyway
	}

	// Execute the migration (pass batch for applying batch-level settings)
	// Route to ADO-specific migration handler if this is an ADO source repository
	var err error
	if repo.ADOProject != nil && *repo.ADOProject != "" {
		// This is an Azure DevOps source repository
		w.logger.Info("Executing ADO migration",
			"repo", repo.FullName,
			"ado_project", *repo.ADOProject,
			"dry_run", dryRun)
		err = w.executor.ExecuteADOMigration(ctx, repo, batch, dryRun)
	} else {
		// This is a GitHub source repository (default)
		err = w.executor.ExecuteMigration(ctx, repo, batch, dryRun)
	}

	if err != nil {
		w.logger.Error("Migration failed",
			"repo", repo.FullName,
			"repo_id", repo.ID,
			"dry_run", dryRun,
			"error", err)

		// Update status to failed
		failedStatus := models.StatusMigrationFailed
		if dryRun {
			failedStatus = models.StatusDryRunFailed
		}
		repo.Status = string(failedStatus)
		if updateErr := w.storage.UpdateRepository(ctx, repo); updateErr != nil {
			w.logger.Error("Failed to update repository status after failure",
				"repo", repo.FullName,
				"error", updateErr)
		}
	} else {
		w.logger.Info("Migration completed successfully",
			"repo", repo.FullName,
			"repo_id", repo.ID,
			"dry_run", dryRun)

		// Status should already be updated by executor, but verify
		updatedRepo, err := w.storage.GetRepository(ctx, repo.FullName)
		if err == nil && updatedRepo != nil {
			w.logger.Debug("Final repository status",
				"repo", repo.FullName,
				"status", updatedRepo.Status)
		}
	}
}

// GetActiveCount returns the number of currently active migrations
func (w *MigrationWorker) GetActiveCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.active)
}

// GetActiveMigrations returns the IDs of currently active migrations
func (w *MigrationWorker) GetActiveMigrations() []int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	ids := make([]int64, 0, len(w.active))
	for id := range w.active {
		ids = append(ids, id)
	}
	return ids
}

// IsActive returns true if the worker is currently running
func (w *MigrationWorker) IsActive() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.ctx != nil && w.ctx.Err() == nil
}
