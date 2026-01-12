package discovery

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

func setupTestDatabase(t *testing.T) *storage.Database {
	t.Helper()

	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func TestNewDBProgressTracker(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:    1,
		Phase: models.PhaseListingRepos,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	if tracker == nil {
		t.Fatal("NewDBProgressTracker returned nil")
		return // Explicitly unreachable, but satisfies static analysis
	}
	if tracker.db == nil {
		t.Error("tracker.db is nil")
	}
	if tracker.logger == nil {
		t.Error("tracker.logger is nil")
	}
	if tracker.progress == nil {
		t.Error("tracker.progress is nil")
	}
	if tracker.batchSize != 1 {
		t.Errorf("Expected batchSize 1, got %d", tracker.batchSize)
	}
}

func TestDBProgressTracker_GetProgressID(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:    42,
		Phase: models.PhaseListingRepos,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	if id := tracker.GetProgressID(); id != 42 {
		t.Errorf("Expected progress ID 42, got %d", id)
	}
}

func TestDBProgressTracker_SetTotalOrgs(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:    1,
		Phase: models.PhaseListingRepos,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	tracker.SetTotalOrgs(10)

	if tracker.progress.TotalOrgs != 10 {
		t.Errorf("Expected TotalOrgs 10, got %d", tracker.progress.TotalOrgs)
	}
}

func TestDBProgressTracker_StartOrg(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:    1,
		Phase: models.PhaseListingRepos,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	tracker.StartOrg("test-org", 0)

	if tracker.progress.CurrentOrg != "test-org" {
		t.Errorf("Expected CurrentOrg 'test-org', got '%s'", tracker.progress.CurrentOrg)
	}
	if tracker.progress.Phase != models.PhaseListingRepos {
		t.Errorf("Expected Phase '%s', got '%s'", models.PhaseListingRepos, tracker.progress.Phase)
	}
}

func TestDBProgressTracker_CompleteOrg(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:            1,
		Phase:         models.PhaseListingRepos,
		ProcessedOrgs: 0,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	tracker.CompleteOrg("test-org", 5)

	if tracker.progress.ProcessedOrgs != 1 {
		t.Errorf("Expected ProcessedOrgs 1, got %d", tracker.progress.ProcessedOrgs)
	}

	// Complete another org
	tracker.CompleteOrg("another-org", 10)

	if tracker.progress.ProcessedOrgs != 2 {
		t.Errorf("Expected ProcessedOrgs 2, got %d", tracker.progress.ProcessedOrgs)
	}
}

func TestDBProgressTracker_SetTotalRepos(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:    1,
		Phase: models.PhaseListingRepos,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	tracker.SetTotalRepos(100)

	if tracker.progress.TotalRepos != 100 {
		t.Errorf("Expected TotalRepos 100, got %d", tracker.progress.TotalRepos)
	}
}

func TestDBProgressTracker_AddRepos(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:         1,
		Phase:      models.PhaseListingRepos,
		TotalRepos: 50,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	tracker.AddRepos(25)

	if tracker.progress.TotalRepos != 75 {
		t.Errorf("Expected TotalRepos 75, got %d", tracker.progress.TotalRepos)
	}

	tracker.AddRepos(10)

	if tracker.progress.TotalRepos != 85 {
		t.Errorf("Expected TotalRepos 85, got %d", tracker.progress.TotalRepos)
	}
}

func TestDBProgressTracker_IncrementProcessedRepos(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:             1,
		Phase:          models.PhaseProfilingRepos,
		ProcessedRepos: 0,
		TotalRepos:     100,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	tracker.IncrementProcessedRepos(1)

	if tracker.progress.ProcessedRepos != 1 {
		t.Errorf("Expected ProcessedRepos 1, got %d", tracker.progress.ProcessedRepos)
	}

	tracker.IncrementProcessedRepos(5)

	if tracker.progress.ProcessedRepos != 6 {
		t.Errorf("Expected ProcessedRepos 6, got %d", tracker.progress.ProcessedRepos)
	}
}

func TestDBProgressTracker_SetPhase(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:    1,
		Phase: models.PhaseListingRepos,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	tracker.SetPhase(models.PhaseProfilingRepos)

	if tracker.progress.Phase != models.PhaseProfilingRepos {
		t.Errorf("Expected Phase '%s', got '%s'", models.PhaseProfilingRepos, tracker.progress.Phase)
	}

	tracker.SetPhase(models.PhaseCompleted)

	if tracker.progress.Phase != models.PhaseCompleted {
		t.Errorf("Expected Phase '%s', got '%s'", models.PhaseCompleted, tracker.progress.Phase)
	}
}

func TestDBProgressTracker_RecordError(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:         1,
		Phase:      models.PhaseProfilingRepos,
		ErrorCount: 0,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	testErr := errors.New("test error")
	tracker.RecordError(testErr)

	if tracker.progress.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", tracker.progress.ErrorCount)
	}
	if tracker.progress.LastError == nil || *tracker.progress.LastError != "test error" {
		t.Errorf("Expected LastError 'test error', got %v", tracker.progress.LastError)
	}

	// Record another error
	anotherErr := errors.New("another error")
	tracker.RecordError(anotherErr)

	if tracker.progress.ErrorCount != 2 {
		t.Errorf("Expected ErrorCount 2, got %d", tracker.progress.ErrorCount)
	}
}

func TestDBProgressTracker_Flush(t *testing.T) {
	db := setupTestDatabase(t)
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	progress := &models.DiscoveryProgress{
		ID:             1,
		Phase:          models.PhaseProfilingRepos,
		ProcessedRepos: 0,
		TotalRepos:     100,
	}

	tracker := NewDBProgressTracker(db, logger, progress)

	// Increment without reaching batch size
	tracker.pendingRepoIncrement = 5

	// Flush should clear pending increment
	tracker.Flush()

	if tracker.pendingRepoIncrement != 0 {
		t.Errorf("Expected pendingRepoIncrement 0 after Flush, got %d", tracker.pendingRepoIncrement)
	}
}

func TestNoOpProgressTracker(t *testing.T) {
	tracker := NoOpProgressTracker{}

	// All methods should be callable without panic
	tracker.SetTotalOrgs(10)
	tracker.StartOrg("test-org", 0)
	tracker.CompleteOrg("test-org", 5)
	tracker.SetTotalRepos(100)
	tracker.AddRepos(50)
	tracker.IncrementProcessedRepos(10)
	tracker.SetPhase("testing")
	tracker.RecordError(errors.New("test error"))

	if id := tracker.GetProgressID(); id != 0 {
		t.Errorf("Expected GetProgressID() to return 0, got %d", id)
	}
}
