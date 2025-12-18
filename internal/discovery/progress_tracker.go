package discovery

import (
	"log/slog"
	"sync"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// ProgressTracker tracks the progress of a discovery operation
type ProgressTracker interface {
	// SetTotalOrgs sets the total number of organizations to process
	SetTotalOrgs(total int)
	// StartOrg signals that processing of an organization has started
	StartOrg(org string, index int)
	// CompleteOrg signals that processing of an organization has completed
	CompleteOrg(org string, repoCount int)
	// SetTotalRepos sets the total number of repositories to process
	SetTotalRepos(total int)
	// AddRepos adds to the total repo count (for incremental discovery)
	AddRepos(count int)
	// IncrementProcessedRepos increments the processed repos counter
	IncrementProcessedRepos(count int)
	// SetPhase updates the current phase of discovery
	SetPhase(phase string)
	// RecordError records an error that occurred during discovery
	RecordError(err error)
	// GetProgressID returns the ID of the progress record
	GetProgressID() int64
}

// DBProgressTracker implements ProgressTracker using database storage
type DBProgressTracker struct {
	db       *storage.Database
	logger   *slog.Logger
	progress *models.DiscoveryProgress
	mu       sync.Mutex

	// Batching: accumulate repo progress and flush periodically
	pendingRepoIncrement int
	batchSize            int
}

// NewDBProgressTracker creates a new database-backed progress tracker
func NewDBProgressTracker(db *storage.Database, logger *slog.Logger, progress *models.DiscoveryProgress) *DBProgressTracker {
	return &DBProgressTracker{
		db:        db,
		logger:    logger,
		progress:  progress,
		batchSize: 5, // Flush every 5 repos to reduce DB writes
	}
}

// GetProgressID returns the ID of the progress record
func (t *DBProgressTracker) GetProgressID() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.progress.ID
}

// SetTotalOrgs sets the total number of organizations to process
func (t *DBProgressTracker) SetTotalOrgs(total int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.TotalOrgs = total
	if err := t.db.UpdateDiscoveryProgress(t.progress); err != nil {
		t.logger.Warn("Failed to update total orgs", "error", err)
	}
}

// StartOrg signals that processing of an organization has started
// Note: ProcessedOrgs is managed by CompleteOrg(), not here
func (t *DBProgressTracker) StartOrg(org string, index int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.CurrentOrg = org
	t.progress.Phase = models.PhaseListingRepos

	if err := t.db.UpdateDiscoveryProgress(t.progress); err != nil {
		t.logger.Warn("Failed to update org progress", "error", err, "org", org)
	}
}

// CompleteOrg signals that processing of an organization has completed
func (t *DBProgressTracker) CompleteOrg(org string, repoCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Flush any pending repo increments
	t.flushPendingReposLocked()

	t.progress.ProcessedOrgs++

	if err := t.db.UpdateDiscoveryProgress(t.progress); err != nil {
		t.logger.Warn("Failed to update org completion", "error", err, "org", org)
	}
}

// SetTotalRepos sets the total number of repositories to process
func (t *DBProgressTracker) SetTotalRepos(total int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.TotalRepos = total
	if err := t.db.UpdateDiscoveryRepoProgress(t.progress.ID, t.progress.ProcessedRepos, total); err != nil {
		t.logger.Warn("Failed to update total repos", "error", err)
	}
}

// AddRepos adds to the total repo count (for incremental discovery)
func (t *DBProgressTracker) AddRepos(count int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.TotalRepos += count
	if err := t.db.UpdateDiscoveryRepoProgress(t.progress.ID, t.progress.ProcessedRepos, t.progress.TotalRepos); err != nil {
		t.logger.Warn("Failed to add repos to total", "error", err)
	}
}

// IncrementProcessedRepos increments the processed repos counter
func (t *DBProgressTracker) IncrementProcessedRepos(count int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pendingRepoIncrement += count
	t.progress.ProcessedRepos += count

	// Batch updates to reduce DB writes
	if t.pendingRepoIncrement >= t.batchSize {
		t.flushPendingReposLocked()
	}
}

// flushPendingReposLocked flushes pending repo increments to the database
// Must be called with mu held
func (t *DBProgressTracker) flushPendingReposLocked() {
	if t.pendingRepoIncrement == 0 {
		return
	}

	if err := t.db.UpdateDiscoveryRepoProgress(t.progress.ID, t.progress.ProcessedRepos, t.progress.TotalRepos); err != nil {
		t.logger.Warn("Failed to flush processed repos", "error", err)
	}
	t.pendingRepoIncrement = 0
}

// SetPhase updates the current phase of discovery
func (t *DBProgressTracker) SetPhase(phase string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.Phase = phase
	if err := t.db.UpdateDiscoveryPhase(t.progress.ID, phase); err != nil {
		t.logger.Warn("Failed to update phase", "error", err, "phase", phase)
	}
}

// RecordError records an error that occurred during discovery
func (t *DBProgressTracker) RecordError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.ErrorCount++
	errMsg := err.Error()
	t.progress.LastError = &errMsg

	if dbErr := t.db.IncrementDiscoveryError(t.progress.ID, errMsg); dbErr != nil {
		t.logger.Warn("Failed to record error", "error", dbErr)
	}
}

// Flush ensures all pending updates are written to the database
func (t *DBProgressTracker) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushPendingReposLocked()
}

// NoOpProgressTracker is a no-op implementation for when progress tracking is disabled
type NoOpProgressTracker struct{}

func (NoOpProgressTracker) SetTotalOrgs(int)            {}
func (NoOpProgressTracker) StartOrg(string, int)        {}
func (NoOpProgressTracker) CompleteOrg(string, int)     {}
func (NoOpProgressTracker) SetTotalRepos(int)           {}
func (NoOpProgressTracker) AddRepos(int)                {}
func (NoOpProgressTracker) IncrementProcessedRepos(int) {}
func (NoOpProgressTracker) SetPhase(string)             {}
func (NoOpProgressTracker) RecordError(error)           {}
func (NoOpProgressTracker) GetProgressID() int64        { return 0 }
