package batch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

func setupTestOrganizer(t *testing.T) (*Organizer, *storage.Database, func()) {
	t.Helper()

	// Create in-memory database
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	// Create organizer
	organizer, err := NewOrganizer(OrganizerConfig{
		Storage: db,
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("Failed to create organizer: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return organizer, db, cleanup
}

func createTestRepository(t *testing.T, db *storage.Database, fullName string, size int64, features map[string]bool) *models.Repository {
	t.Helper()

	sizeBytes := size * 1024 // Convert KB to bytes
	repo := &models.Repository{
		FullName:     fullName,
		Status:       string(models.StatusPending),
		Source:       "github",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
		GitProperties: &models.RepositoryGitProperties{
			TotalSize:     &sizeBytes,
			HasLFS:        features["lfs"],
			HasSubmodules: features["submodules"],
			CommitCount:   100,
			BranchCount:   2,
		},
		Features: &models.RepositoryFeatures{
			HasActions:  features["actions"],
			HasWiki:     features["wiki"],
			HasPages:    features["pages"],
			HasProjects: features["projects"],
		},
	}

	ctx := context.Background()
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	return repo
}

func TestNewOrganizer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name    string
		config  OrganizerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: OrganizerConfig{
				Storage: &storage.Database{},
				Logger:  logger,
			},
			wantErr: false,
		},
		{
			name: "missing storage",
			config: OrganizerConfig{
				Logger: logger,
			},
			wantErr: true,
		},
		{
			name: "missing logger",
			config: OrganizerConfig{
				Storage: &storage.Database{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOrganizer(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOrganizer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSelectPilotRepositories(t *testing.T) {
	organizer, db, cleanup := setupTestOrganizer(t)
	defer cleanup()

	ctx := context.Background()

	// Create diverse test repositories
	createTestRepository(t, db, "org1/small-repo", 500, map[string]bool{
		"lfs": false, "submodules": false, "actions": false,
	})

	createTestRepository(t, db, "org1/medium-repo-lfs", 100000, map[string]bool{
		"lfs": true, "submodules": false, "actions": true, "wiki": true,
	})

	createTestRepository(t, db, "org2/large-repo", 5000000, map[string]bool{
		"lfs": true, "submodules": true, "actions": true, "pages": true,
	})

	createTestRepository(t, db, "org2/actions-repo", 50000, map[string]bool{
		"actions": true, "wiki": true, "issues": true,
	})

	createTestRepository(t, db, "org3/too-large", 11000000, map[string]bool{
		"lfs": true,
	})

	t.Run("default criteria", func(t *testing.T) {
		criteria := DefaultPilotCriteria()
		repos, err := organizer.SelectPilotRepositories(ctx, criteria)
		if err != nil {
			t.Fatalf("SelectPilotRepositories() error = %v", err)
		}

		// Should exclude too-large repo (>10GB)
		if len(repos) > 4 {
			t.Errorf("Expected at most 4 repos, got %d", len(repos))
		}

		// Verify all selected repos are within size limits
		for _, repo := range repos {
			if repo.GetTotalSize() != nil {
				sizeKB := *repo.GetTotalSize() / 1024
				if sizeKB > criteria.MaxSize {
					t.Errorf("Repo %s exceeds max size: %d > %d", repo.FullName, sizeKB, criteria.MaxSize)
				}
			}
		}
	})

	t.Run("with LFS requirement", func(t *testing.T) {
		criteria := DefaultPilotCriteria()
		criteria.RequireLFS = true
		criteria.MaxCount = 5

		repos, err := organizer.SelectPilotRepositories(ctx, criteria)
		if err != nil {
			t.Fatalf("SelectPilotRepositories() error = %v", err)
		}

		// All selected repos should have LFS
		for _, repo := range repos {
			if !repo.HasLFS() {
				t.Errorf("Repo %s should have LFS", repo.FullName)
			}
		}
	})

	t.Run("with organization filter", func(t *testing.T) {
		criteria := DefaultPilotCriteria()
		criteria.Organizations = []string{"org1"}

		repos, err := organizer.SelectPilotRepositories(ctx, criteria)
		if err != nil {
			t.Fatalf("SelectPilotRepositories() error = %v", err)
		}

		// All selected repos should be from org1
		for _, repo := range repos {
			if repo.Organization() != "org1" {
				t.Errorf("Repo %s is not from org1", repo.FullName)
			}
		}
	})

	t.Run("max count limit", func(t *testing.T) {
		criteria := DefaultPilotCriteria()
		criteria.MaxCount = 2

		repos, err := organizer.SelectPilotRepositories(ctx, criteria)
		if err != nil {
			t.Fatalf("SelectPilotRepositories() error = %v", err)
		}

		if len(repos) > 2 {
			t.Errorf("Expected at most 2 repos, got %d", len(repos))
		}
	})
}

func TestCreatePilotBatch(t *testing.T) {
	organizer, db, cleanup := setupTestOrganizer(t)
	defer cleanup()

	ctx := context.Background()

	// Create test repositories
	createTestRepository(t, db, "org1/repo1", 100000, map[string]bool{
		"lfs": true, "actions": true,
	})
	createTestRepository(t, db, "org1/repo2", 200000, map[string]bool{
		"submodules": true,
	})

	t.Run("successful creation", func(t *testing.T) {
		criteria := DefaultPilotCriteria()
		criteria.MaxCount = 2

		batch, repos, err := organizer.CreatePilotBatch(ctx, "Test Pilot", criteria)
		if err != nil {
			t.Fatalf("CreatePilotBatch() error = %v", err)
		}

		if batch == nil {
			t.Fatal("Expected batch to be created")
			return // Prevent staticcheck SA5011
		}

		if batch.Type != "pilot" {
			t.Errorf("Expected batch type 'pilot', got %s", batch.Type)
		}

		if len(repos) == 0 {
			t.Error("Expected repositories to be selected")
		}

		// Verify repositories are assigned to batch
		for _, repo := range repos {
			if repo.BatchID == nil || *repo.BatchID != batch.ID {
				t.Errorf("Repo %s not assigned to batch", repo.FullName)
			}
			if repo.Priority != 1 {
				t.Errorf("Expected pilot repo priority 1, got %d", repo.Priority)
			}
		}
	})

	t.Run("no matching repositories", func(t *testing.T) {
		criteria := DefaultPilotCriteria()
		criteria.RequirePages = true // None have pages

		_, _, err := organizer.CreatePilotBatch(ctx, "Empty Pilot", criteria)
		if err == nil {
			t.Error("Expected error when no repos match criteria")
		}
	})
}

func TestOrganizeIntoWaves(t *testing.T) {
	organizer, db, cleanup := setupTestOrganizer(t)
	defer cleanup()

	ctx := context.Background()

	// Create 55 test repositories (will create 2 waves with default size 50)
	for i := range 55 {
		org := "org1"
		if i >= 30 {
			org = "org2"
		}
		createTestRepository(t, db, fmt.Sprintf("%s/repo%d", org, i), 100000, map[string]bool{})
	}

	t.Run("default wave organization", func(t *testing.T) {
		criteria := DefaultWaveCriteria()

		waves, err := organizer.OrganizeIntoWaves(ctx, criteria)
		if err != nil {
			t.Fatalf("OrganizeIntoWaves() error = %v", err)
		}

		if len(waves) != 2 {
			t.Errorf("Expected 2 waves, got %d", len(waves))
		}

		// Verify wave names
		if len(waves) > 0 && waves[0].Name != "Wave 1" {
			t.Errorf("Expected first wave named 'Wave 1', got %s", waves[0].Name)
		}

		// Verify repositories are assigned
		totalAssigned := 0
		for _, wave := range waves {
			repos, _ := db.ListRepositories(ctx, map[string]any{
				"batch_id": wave.ID,
			})
			totalAssigned += len(repos)
		}

		if totalAssigned != 55 {
			t.Errorf("Expected 55 repos assigned, got %d", totalAssigned)
		}
	})

	t.Run("custom wave size", func(t *testing.T) {
		// Create new organizer with fresh database
		testOrg, testDB, testCleanup := setupTestOrganizer(t)
		defer testCleanup()

		// Create 55 repos
		for i := range 55 {
			createTestRepository(t, testDB, fmt.Sprintf("org/repo%d", i), 100000, map[string]bool{})
		}

		criteria := WaveCriteria{
			WaveSize:            20,
			GroupByOrganization: false,
			SortBy:              "name",
		}

		waves, err := testOrg.OrganizeIntoWaves(context.Background(), criteria)
		if err != nil {
			t.Fatalf("OrganizeIntoWaves() error = %v", err)
		}

		if len(waves) != 3 {
			t.Errorf("Expected 3 waves (20+20+15), got %d", len(waves))
		}
	})
}

func TestGetBatchProgress(t *testing.T) {
	organizer, db, cleanup := setupTestOrganizer(t)
	defer cleanup()

	ctx := context.Background()

	// Create batch
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "pilot",
		RepositoryCount: 3,
		Status:          "in_progress",
		CreatedAt:       time.Now(),
	}
	now := time.Now()
	batch.StartedAt = &now

	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create repositories with different statuses
	repo1 := createTestRepository(t, db, "org/repo1", 100000, map[string]bool{})
	repo1.BatchID = &batch.ID
	repo1.Status = string(models.StatusComplete)
	db.UpdateRepository(ctx, repo1)

	repo2 := createTestRepository(t, db, "org/repo2", 100000, map[string]bool{})
	repo2.BatchID = &batch.ID
	repo2.Status = string(models.StatusMigratingContent)
	db.UpdateRepository(ctx, repo2)

	repo3 := createTestRepository(t, db, "org/repo3", 100000, map[string]bool{})
	repo3.BatchID = &batch.ID
	repo3.Status = string(models.StatusMigrationFailed)
	db.UpdateRepository(ctx, repo3)

	t.Run("calculate progress", func(t *testing.T) {
		progress, err := organizer.GetBatchProgress(ctx, batch.ID)
		if err != nil {
			t.Fatalf("GetBatchProgress() error = %v", err)
		}

		if progress.TotalRepos != 3 {
			t.Errorf("Expected 3 total repos, got %d", progress.TotalRepos)
		}

		if progress.CompletedRepos != 1 {
			t.Errorf("Expected 1 completed repo, got %d", progress.CompletedRepos)
		}

		if progress.FailedRepos != 1 {
			t.Errorf("Expected 1 failed repo, got %d", progress.FailedRepos)
		}

		if progress.InProgressRepos != 1 {
			t.Errorf("Expected 1 in-progress repo, got %d", progress.InProgressRepos)
		}

		expectedPercent := float64(1) / float64(3) * 100
		if progress.PercentComplete != expectedPercent {
			t.Errorf("Expected %.2f%% complete, got %.2f%%", expectedPercent, progress.PercentComplete)
		}
	})
}

func TestScoreRepositories(t *testing.T) {
	organizer, db, cleanup := setupTestOrganizer(t)
	defer cleanup()

	// Create repos with different features
	repo1 := createTestRepository(t, db, "org/basic", 100000, map[string]bool{})
	repo2 := createTestRepository(t, db, "org/with-lfs", 100000, map[string]bool{
		"lfs": true, "actions": true,
	})
	repo3 := createTestRepository(t, db, "org/full-featured", 100000, map[string]bool{
		"lfs": true, "submodules": true, "actions": true, "wiki": true, "pages": true,
	})

	repos := []*models.Repository{repo1, repo2, repo3}

	scored := organizer.scoreRepositories(repos)

	if len(scored) != 3 {
		t.Fatalf("Expected 3 scored repos, got %d", len(scored))
	}

	// Full-featured repo should have highest score
	maxScore := 0.0
	maxIdx := 0
	for i, s := range scored {
		if s.Score > maxScore {
			maxScore = s.Score
			maxIdx = i
		}
	}

	if scored[maxIdx].Repo.FullName != "org/full-featured" {
		t.Errorf("Expected full-featured repo to have highest score, got %s", scored[maxIdx].Repo.FullName)
	}
}
