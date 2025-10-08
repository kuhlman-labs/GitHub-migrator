package worker

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/migration"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

func setupTestWorker(t *testing.T) (*MigrationWorker, *storage.Database, *migration.Executor) {
	t.Helper()

	// Create test database
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, err := storage.NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create mock GitHub clients
	sourceClient := &github.Client{}
	destClient := &github.Client{}

	// Create migration executor
	executor, err := migration.NewExecutor(migration.ExecutorConfig{
		SourceClient: sourceClient,
		DestClient:   destClient,
		Storage:      db,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Create worker
	worker, err := NewMigrationWorker(WorkerConfig{
		Executor:     executor,
		Storage:      db,
		Logger:       logger,
		PollInterval: 100 * time.Millisecond, // Short interval for tests
		Workers:      3,
	})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	return worker, db, executor
}

func TestNewMigrationWorker(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		cfg     WorkerConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			cfg: WorkerConfig{
				Executor:     &migration.Executor{},
				Storage:      &storage.Database{},
				Logger:       logger,
				PollInterval: 30 * time.Second,
				Workers:      5,
			},
			wantErr: false,
		},
		{
			name: "missing executor",
			cfg: WorkerConfig{
				Storage:      &storage.Database{},
				Logger:       logger,
				PollInterval: 30 * time.Second,
				Workers:      5,
			},
			wantErr: true,
		},
		{
			name: "missing storage",
			cfg: WorkerConfig{
				Executor:     &migration.Executor{},
				Logger:       logger,
				PollInterval: 30 * time.Second,
				Workers:      5,
			},
			wantErr: true,
		},
		{
			name: "missing logger",
			cfg: WorkerConfig{
				Executor:     &migration.Executor{},
				Storage:      &storage.Database{},
				PollInterval: 30 * time.Second,
				Workers:      5,
			},
			wantErr: true,
		},
		{
			name: "default poll interval",
			cfg: WorkerConfig{
				Executor:     &migration.Executor{},
				Storage:      &storage.Database{},
				Logger:       logger,
				PollInterval: 0, // Should default to 30s
				Workers:      5,
			},
			wantErr: false,
		},
		{
			name: "default workers",
			cfg: WorkerConfig{
				Executor:     &migration.Executor{},
				Storage:      &storage.Database{},
				Logger:       logger,
				PollInterval: 30 * time.Second,
				Workers:      0, // Should default to 5
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker, err := NewMigrationWorker(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMigrationWorker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && worker == nil {
				t.Error("Expected worker to be created")
			}
		})
	}
}

func TestMigrationWorker_StartStop(t *testing.T) {
	worker, db, _ := setupTestWorker(t)
	defer db.Close()

	ctx := context.Background()

	// Test starting worker
	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Verify worker is active
	if !worker.IsActive() {
		t.Error("Worker should be active after starting")
	}

	// Test double start
	err = worker.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already started worker")
	}

	// Test stopping worker
	err = worker.Stop()
	if err != nil {
		t.Fatalf("Failed to stop worker: %v", err)
	}

	// Give it a moment to fully stop
	time.Sleep(50 * time.Millisecond)

	// Verify worker is not active
	if worker.IsActive() {
		t.Error("Worker should not be active after stopping")
	}
}

func TestMigrationWorker_GetActiveCount(t *testing.T) {
	worker, db, _ := setupTestWorker(t)
	defer db.Close()

	// Initially should be 0
	if count := worker.GetActiveCount(); count != 0 {
		t.Errorf("Expected 0 active migrations, got %d", count)
	}

	// Manually add some active migrations for testing
	worker.mu.Lock()
	worker.active[1] = true
	worker.active[2] = true
	worker.mu.Unlock()

	if count := worker.GetActiveCount(); count != 2 {
		t.Errorf("Expected 2 active migrations, got %d", count)
	}
}

func TestMigrationWorker_GetActiveMigrations(t *testing.T) {
	worker, db, _ := setupTestWorker(t)
	defer db.Close()

	// Initially should be empty
	if migrations := worker.GetActiveMigrations(); len(migrations) != 0 {
		t.Errorf("Expected 0 active migrations, got %d", len(migrations))
	}

	// Manually add some active migrations for testing
	worker.mu.Lock()
	worker.active[1] = true
	worker.active[2] = true
	worker.active[3] = true
	worker.mu.Unlock()

	migrations := worker.GetActiveMigrations()
	if len(migrations) != 3 {
		t.Errorf("Expected 3 active migrations, got %d", len(migrations))
	}

	// Verify all IDs are present
	idMap := make(map[int64]bool)
	for _, id := range migrations {
		idMap[id] = true
	}
	if !idMap[1] || !idMap[2] || !idMap[3] {
		t.Error("Not all expected migration IDs were returned")
	}
}

func TestMigrationWorker_ProcessQueuedRepositories(t *testing.T) {
	t.Skip("Skipping test that requires fully initialized GitHub clients - needs integration test")

	// This test would require real GitHub clients to work properly
	// Integration tests with actual credentials would be needed to test the full execution flow
	t.Log("Integration test needed for full migration execution")
}

func TestMigrationWorker_WorkerSlots(t *testing.T) {
	// Create worker with only 1 worker slot
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, _ := storage.NewDatabase(dbCfg)
	defer db.Close()
	db.Migrate()

	sourceClient := &github.Client{}
	destClient := &github.Client{}
	executor, _ := migration.NewExecutor(migration.ExecutorConfig{
		SourceClient: sourceClient,
		DestClient:   destClient,
		Storage:      db,
		Logger:       logger,
	})

	worker, _ := NewMigrationWorker(WorkerConfig{
		Executor:     executor,
		Storage:      db,
		Logger:       logger,
		PollInterval: 100 * time.Millisecond,
		Workers:      1, // Only 1 worker slot
	})

	// Manually occupy the worker slot
	worker.mu.Lock()
	worker.active[1] = true
	worker.mu.Unlock()

	// Try to process - should not pick up new repos
	ctx := context.Background()
	repo := &models.Repository{
		FullName:  "org/repo1",
		SourceURL: "https://github.com/org/repo1",
		Status:    string(models.StatusQueuedForMigration),
	}
	db.SaveRepository(ctx, repo)

	// Should not process because all slots are busy
	worker.processQueuedRepositories()

	time.Sleep(50 * time.Millisecond)

	// Should still only have 1 active (the one we manually added)
	if count := worker.GetActiveCount(); count != 1 {
		t.Errorf("Expected 1 active migration, got %d", count)
	}
}

func TestMigrationWorker_StopWithActiveMigrations(t *testing.T) {
	worker, db, _ := setupTestWorker(t)
	defer db.Close()

	ctx := context.Background()

	// Start worker
	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Manually add an "active" migration and a corresponding goroutine
	worker.mu.Lock()
	worker.active[1] = true
	worker.mu.Unlock()

	worker.wg.Add(1)
	go func() {
		defer worker.wg.Done()
		time.Sleep(100 * time.Millisecond)
		worker.mu.Lock()
		delete(worker.active, 1)
		worker.mu.Unlock()
	}()

	// This should block until the "migration" completes
	startTime := time.Now()
	err = worker.Stop()
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to stop worker: %v", err)
	}

	// Should have waited at least 100ms
	if duration < 100*time.Millisecond {
		t.Error("Stop() did not wait for active migrations")
	}
}

func TestMigrationWorker_IsActive(t *testing.T) {
	worker, db, _ := setupTestWorker(t)
	defer db.Close()

	// Initially not active
	if worker.IsActive() {
		t.Error("Worker should not be active before starting")
	}

	// Start worker
	ctx := context.Background()
	worker.Start(ctx)

	// Should be active
	if !worker.IsActive() {
		t.Error("Worker should be active after starting")
	}

	// Stop worker
	worker.Stop()

	// Give it time to stop
	time.Sleep(50 * time.Millisecond)

	// Should not be active
	if worker.IsActive() {
		t.Error("Worker should not be active after stopping")
	}
}
