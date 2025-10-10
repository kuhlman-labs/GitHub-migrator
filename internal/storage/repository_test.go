package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

const (
	testDefaultBranch = "main"
	testMessage       = "Test message"
)

func setupTestDB(t *testing.T) *Database {
	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func TestSaveRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch
	topContrib := "user1,user2"

	repo := &models.Repository{
		FullName:         "test/repo",
		Source:           "ghes",
		SourceURL:        "https://github.com/test/repo",
		TotalSize:        &totalSize,
		DefaultBranch:    &defaultBranch,
		HasWiki:          true,
		HasPages:         false,
		HasLFS:           true,
		HasSubmodules:    false,
		BranchCount:      5,
		CommitCount:      100,
		ContributorCount: 2,
		TopContributors:  &topContrib,
		Status:           string(models.StatusPending),
		DiscoveredAt:     time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Test insert
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Test update (upsert)
	newSize := int64(2048 * 1024)
	repo.TotalSize = &newSize
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}
}

func TestGetRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	original := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, original); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Test get existing repository
	retrieved, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved repository is nil")
	}

	if retrieved.FullName != original.FullName {
		t.Errorf("Expected FullName '%s', got '%s'", original.FullName, retrieved.FullName)
	}

	// Test get non-existent repository
	notFound, err := db.GetRepository(ctx, "nonexistent/repo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent repository")
	}
}

func TestListRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	// Create test repositories
	repos := []*models.Repository{
		{
			FullName:      "org/repo1",
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo1",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			HasLFS:        true,
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			FullName:      "org/repo2",
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo2",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusComplete),
			HasLFS:        false,
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			FullName:      "org/repo3",
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo3",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusQueuedForMigration),
			HasLFS:        false,
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	for _, repo := range repos {
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	tests := []struct {
		name     string
		filters  map[string]interface{}
		expected int
	}{
		{
			name:     "No filters",
			filters:  nil,
			expected: 3,
		},
		{
			name:     "Filter by status",
			filters:  map[string]interface{}{"status": string(models.StatusPending)},
			expected: 1,
		},
		{
			name:     "Filter by multiple statuses",
			filters:  map[string]interface{}{"status": []string{string(models.StatusPending), string(models.StatusComplete)}},
			expected: 2,
		},
		{
			name:     "Filter by LFS",
			filters:  map[string]interface{}{"has_lfs": true},
			expected: 1,
		},
		{
			name:     "Filter by source",
			filters:  map[string]interface{}{"source": "ghes"},
			expected: 3,
		},
		{
			name:     "Filter with limit",
			filters:  map[string]interface{}{"limit": 1},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.ListRepositories(ctx, tt.filters)
			if err != nil {
				t.Fatalf("ListRepositories failed: %v", err)
			}

			if len(results) != tt.expected {
				t.Errorf("Expected %d repositories, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestUpdateRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	repo := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Update the repository
	newSize := int64(2048 * 1024)
	repo.TotalSize = &newSize
	repo.Status = string(models.StatusComplete)
	now := time.Now()
	repo.MigratedAt = &now

	if err := db.UpdateRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}

	// Verify update
	updated, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if updated.TotalSize == nil || *updated.TotalSize != newSize {
		t.Errorf("Expected TotalSize %d, got %v", newSize, updated.TotalSize)
	}
	if updated.Status != string(models.StatusComplete) {
		t.Errorf("Expected Status 'complete', got '%s'", updated.Status)
	}
	if updated.MigratedAt == nil {
		t.Error("Expected MigratedAt to be set")
	}
}

func TestUpdateRepositoryStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	repo := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Update status
	if err := db.UpdateRepositoryStatus(ctx, "test/repo", models.StatusComplete); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify
	updated, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if updated.Status != string(models.StatusComplete) {
		t.Errorf("Expected Status 'complete', got '%s'", updated.Status)
	}
}

func TestDeleteRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	repo := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Delete
	if err := db.DeleteRepository(ctx, "test/repo"); err != nil {
		t.Fatalf("Failed to delete repository: %v", err)
	}

	// Verify deletion
	deleted, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if deleted != nil {
		t.Error("Expected repository to be deleted")
	}
}

func TestCountRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	// Create test repositories
	for i := 1; i <= 5; i++ {
		status := models.StatusPending
		if i > 3 {
			status = models.StatusComplete
		}

		repo := &models.Repository{
			FullName:      "org/repo" + string(rune('0'+i)),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(status),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	tests := []struct {
		name     string
		filters  map[string]interface{}
		expected int
	}{
		{
			name:     "Count all",
			filters:  nil,
			expected: 5,
		},
		{
			name:     "Count pending",
			filters:  map[string]interface{}{"status": string(models.StatusPending)},
			expected: 3,
		},
		{
			name:     "Count complete",
			filters:  map[string]interface{}{"status": string(models.StatusComplete)},
			expected: 2,
		},
		{
			name:     "Count multiple statuses",
			filters:  map[string]interface{}{"status": []string{string(models.StatusPending), string(models.StatusComplete)}},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := db.CountRepositories(ctx, tt.filters)
			if err != nil {
				t.Fatalf("CountRepositories failed: %v", err)
			}

			if count != tt.expected {
				t.Errorf("Expected count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestGetRepositoryStatsByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	// Create test repositories with different statuses
	statuses := []models.MigrationStatus{
		models.StatusPending,
		models.StatusPending,
		models.StatusPending,
		models.StatusComplete,
		models.StatusComplete,
		models.StatusMigrationFailed,
	}

	for i, status := range statuses {
		repo := &models.Repository{
			FullName:      "org/repo" + string(rune('0'+i)),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(status),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	stats, err := db.GetRepositoryStatsByStatus(ctx)
	if err != nil {
		t.Fatalf("GetRepositoryStatsByStatus failed: %v", err)
	}

	if stats[string(models.StatusPending)] != 3 {
		t.Errorf("Expected 3 pending, got %d", stats[string(models.StatusPending)])
	}
	if stats[string(models.StatusComplete)] != 2 {
		t.Errorf("Expected 2 complete, got %d", stats[string(models.StatusComplete)])
	}
	if stats[string(models.StatusMigrationFailed)] != 1 {
		t.Errorf("Expected 1 failed, got %d", stats[string(models.StatusMigrationFailed)])
	}
}

func TestGetRepositoriesByIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch

	// Save test repositories
	var ids []int64
	for i := 0; i < 3; i++ {
		repo := &models.Repository{
			FullName:      fmt.Sprintf("org/repo%d", i),
			Source:        "ghes",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}

		saved, _ := db.GetRepository(ctx, repo.FullName)
		ids = append(ids, saved.ID)
	}

	// Test getting repositories by IDs
	found, err := db.GetRepositoriesByIDs(ctx, ids[:2])
	if err != nil {
		t.Fatalf("GetRepositoriesByIDs() error = %v", err)
	}

	if len(found) != 2 {
		t.Errorf("GetRepositoriesByIDs() returned %d repos, want 2", len(found))
	}

	// Test empty IDs
	empty, err := db.GetRepositoriesByIDs(ctx, []int64{})
	if err != nil {
		t.Fatalf("GetRepositoriesByIDs() with empty IDs error = %v", err)
	}

	if len(empty) != 0 {
		t.Errorf("GetRepositoriesByIDs() with empty IDs returned %d repos, want 0", len(empty))
	}
}

func TestGetRepositoriesByNames(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch

	// Save test repositories
	names := []string{"org/repoa", "org/repob", "org/repoc"}
	for _, name := range names {
		repo := &models.Repository{
			FullName:      name,
			Source:        "ghes",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
	}

	// Test getting repositories by names
	found, err := db.GetRepositoriesByNames(ctx, []string{"org/repoa", "org/repoc"})
	if err != nil {
		t.Fatalf("GetRepositoriesByNames() error = %v", err)
	}

	if len(found) != 2 {
		t.Errorf("GetRepositoriesByNames() returned %d repos, want 2", len(found))
	}

	// Test empty names
	empty, err := db.GetRepositoriesByNames(ctx, []string{})
	if err != nil {
		t.Fatalf("GetRepositoriesByNames() with empty names error = %v", err)
	}

	if len(empty) != 0 {
		t.Errorf("GetRepositoriesByNames() with empty names returned %d repos, want 0", len(empty))
	}
}

func TestGetRepositoryByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch

	repo := &models.Repository{
		FullName:      "org/repo1",
		Source:        "ghes",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	saved, _ := db.GetRepository(ctx, repo.FullName)

	// Get by ID
	found, err := db.GetRepositoryByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetRepositoryByID() error = %v", err)
	}

	if found == nil {
		t.Fatal("GetRepositoryByID() returned nil")
	}

	if found.FullName != repo.FullName {
		t.Errorf("GetRepositoryByID() FullName = %s, want %s", found.FullName, repo.FullName)
	}

	// Test non-existent ID
	notFound, err := db.GetRepositoryByID(ctx, 999999)
	if err != nil {
		t.Fatalf("GetRepositoryByID() with invalid ID error = %v", err)
	}

	if notFound != nil {
		t.Error("GetRepositoryByID() with invalid ID should return nil")
	}
}

func TestBatchOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create batch
	desc := "Test batch description"
	batch := &models.Batch{
		Name:            "Test Batch",
		Description:     &desc,
		Type:            "pilot",
		RepositoryCount: 5,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}

	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	if batch.ID == 0 {
		t.Error("CreateBatch() should set batch ID")
	}

	// Get batch
	found, err := db.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("GetBatch() error = %v", err)
	}

	if found == nil {
		t.Fatal("GetBatch() returned nil")
	}

	if found.Name != batch.Name {
		t.Errorf("GetBatch() Name = %s, want %s", found.Name, batch.Name)
	}

	// Update batch
	found.Status = "in_progress"
	now := time.Now()
	found.StartedAt = &now

	if err := db.UpdateBatch(ctx, found); err != nil {
		t.Fatalf("UpdateBatch() error = %v", err)
	}

	// Verify update
	updated, err := db.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("GetBatch() after update error = %v", err)
	}

	if updated.Status != "in_progress" {
		t.Errorf("UpdateBatch() Status = %s, want in_progress", updated.Status)
	}

	// List batches
	batches, err := db.ListBatches(ctx)
	if err != nil {
		t.Fatalf("ListBatches() error = %v", err)
	}

	if len(batches) == 0 {
		t.Error("ListBatches() returned empty list")
	}

	// Test non-existent batch
	notFound, err := db.GetBatch(ctx, 999999)
	if err != nil {
		t.Fatalf("GetBatch() with invalid ID error = %v", err)
	}

	if notFound != nil {
		t.Error("GetBatch() with invalid ID should return nil")
	}
}

func TestGetMigrationHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test repository
	repo := &models.Repository{
		FullName:     "test/repo",
		Source:       "ghes",
		SourceURL:    "https://github.com/test/repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}
	savedRepo, _ := db.GetRepository(ctx, repo.FullName)

	// Initially should have no history
	history, err := db.GetMigrationHistory(ctx, savedRepo.ID)
	if err != nil {
		t.Fatalf("GetMigrationHistory() error = %v", err)
	}
	if len(history) != 0 {
		t.Errorf("Expected 0 history entries, got %d", len(history))
	}

	// Create migration history
	msg := testMessage
	hist := &models.MigrationHistory{
		RepositoryID: savedRepo.ID,
		Status:       "in_progress",
		Phase:        "migration",
		Message:      &msg,
		StartedAt:    time.Now(),
	}
	historyID, err := db.CreateMigrationHistory(ctx, hist)
	if err != nil {
		t.Fatalf("CreateMigrationHistory() error = %v", err)
	}
	if historyID == 0 {
		t.Error("Expected non-zero history ID")
	}

	// Retrieve history
	history, err = db.GetMigrationHistory(ctx, savedRepo.ID)
	if err != nil {
		t.Fatalf("GetMigrationHistory() error = %v", err)
	}
	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	}
}

func TestCreateMigrationHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test repository
	repo := &models.Repository{
		FullName:     "test/repo",
		Source:       "ghes",
		SourceURL:    "https://github.com/test/repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}
	savedRepo, _ := db.GetRepository(ctx, repo.FullName)

	// Create migration history
	msg := testMessage
	hist := &models.MigrationHistory{
		RepositoryID: savedRepo.ID,
		Status:       "in_progress",
		Phase:        "migration",
		Message:      &msg,
		StartedAt:    time.Now(),
	}
	historyID, err := db.CreateMigrationHistory(ctx, hist)
	if err != nil {
		t.Fatalf("CreateMigrationHistory() error = %v", err)
	}
	if historyID == 0 {
		t.Error("Expected non-zero history ID")
	}
}

func TestUpdateMigrationHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test repository
	repo := &models.Repository{
		FullName:     "test/repo",
		Source:       "ghes",
		SourceURL:    "https://github.com/test/repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}
	savedRepo, _ := db.GetRepository(ctx, repo.FullName)

	// Create migration history
	msg := testMessage
	hist := &models.MigrationHistory{
		RepositoryID: savedRepo.ID,
		Status:       "in_progress",
		Phase:        "migration",
		Message:      &msg,
		StartedAt:    time.Now(),
	}
	historyID, err := db.CreateMigrationHistory(ctx, hist)
	if err != nil {
		t.Fatalf("CreateMigrationHistory() error = %v", err)
	}

	// Update history status
	errMsg := "Test error"
	err = db.UpdateMigrationHistory(ctx, historyID, "failed", &errMsg)
	if err != nil {
		t.Fatalf("UpdateMigrationHistory() error = %v", err)
	}
}

func TestGetMigrationLogs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test repository
	repo := &models.Repository{
		FullName:     "test/repo",
		Source:       "ghes",
		SourceURL:    "https://github.com/test/repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}
	savedRepo, _ := db.GetRepository(ctx, repo.FullName)

	// Initially should have no logs
	logs, err := db.GetMigrationLogs(ctx, savedRepo.ID, "", "", 100, 0)
	if err != nil {
		t.Fatalf("GetMigrationLogs() error = %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("Expected 0 logs, got %d", len(logs))
	}

	// Create migration logs
	log1 := &models.MigrationLog{
		RepositoryID: savedRepo.ID,
		Level:        "INFO",
		Phase:        "pre_migration",
		Operation:    "validate",
		Message:      "Validation started",
		Timestamp:    time.Now(),
	}
	log2 := &models.MigrationLog{
		RepositoryID: savedRepo.ID,
		Level:        "ERROR",
		Phase:        "migration",
		Operation:    "migrate",
		Message:      "Migration failed",
		Timestamp:    time.Now(),
	}
	if err := db.CreateMigrationLog(ctx, log1); err != nil {
		t.Fatalf("CreateMigrationLog() error = %v", err)
	}
	if err := db.CreateMigrationLog(ctx, log2); err != nil {
		t.Fatalf("CreateMigrationLog() error = %v", err)
	}

	// Get all logs
	logs, err = db.GetMigrationLogs(ctx, savedRepo.ID, "", "", 100, 0)
	if err != nil {
		t.Fatalf("GetMigrationLogs() error = %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}

	// Filter by level
	logs, err = db.GetMigrationLogs(ctx, savedRepo.ID, "ERROR", "", 100, 0)
	if err != nil {
		t.Fatalf("GetMigrationLogs() with level filter error = %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 ERROR log, got %d", len(logs))
	}

	// Filter by phase
	logs, err = db.GetMigrationLogs(ctx, savedRepo.ID, "", "pre_migration", 100, 0)
	if err != nil {
		t.Fatalf("GetMigrationLogs() with phase filter error = %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 pre_migration log, got %d", len(logs))
	}

	// Test pagination
	logs, err = db.GetMigrationLogs(ctx, savedRepo.ID, "", "", 1, 0)
	if err != nil {
		t.Fatalf("GetMigrationLogs() with limit error = %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log with limit=1, got %d", len(logs))
	}
}

func TestCreateMigrationLog(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test repository
	repo := &models.Repository{
		FullName:     "test/repo",
		Source:       "ghes",
		SourceURL:    "https://github.com/test/repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}
	savedRepo, _ := db.GetRepository(ctx, repo.FullName)

	// Create migration log
	log := &models.MigrationLog{
		RepositoryID: savedRepo.ID,
		Level:        "INFO",
		Phase:        "pre_migration",
		Operation:    "validate",
		Message:      "Validation started",
		Timestamp:    time.Now(),
	}
	err := db.CreateMigrationLog(ctx, log)
	if err != nil {
		t.Fatalf("CreateMigrationLog() error = %v", err)
	}
}

func TestAddRepositoriesToBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a batch
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "pilot",
		RepositoryCount: 0,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create test repositories
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch
	var repoIDs []int64
	for i := 0; i < 3; i++ {
		repo := &models.Repository{
			FullName:      fmt.Sprintf("org/repo%d", i),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
		saved, _ := db.GetRepository(ctx, repo.FullName)
		repoIDs = append(repoIDs, saved.ID)
	}

	// Add repositories to batch
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
		t.Fatalf("AddRepositoriesToBatch() error = %v", err)
	}

	// Verify repositories are assigned
	repos, err := db.ListRepositories(ctx, map[string]interface{}{"batch_id": batch.ID})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("Expected 3 repositories in batch, got %d", len(repos))
	}

	// Verify batch count was updated
	updatedBatch, err := db.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("GetBatch() error = %v", err)
	}
	if updatedBatch.RepositoryCount != 3 {
		t.Errorf("Expected batch repository count 3, got %d", updatedBatch.RepositoryCount)
	}

	// Test with empty IDs
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{}); err != nil {
		t.Errorf("AddRepositoriesToBatch() with empty IDs should not error, got %v", err)
	}
}

func TestRemoveRepositoriesFromBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a batch
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "pilot",
		RepositoryCount: 0,
		Status:          "ready",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create and assign repositories to batch
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch
	var repoIDs []int64
	for i := 0; i < 3; i++ {
		repo := &models.Repository{
			FullName:      fmt.Sprintf("org/repo%d", i),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			BatchID:       &batch.ID,
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
		saved, _ := db.GetRepository(ctx, repo.FullName)
		repoIDs = append(repoIDs, saved.ID)
	}

	// Update batch count manually for test
	batch.RepositoryCount = 3
	if err := db.UpdateBatch(ctx, batch); err != nil {
		t.Fatalf("UpdateBatch() error = %v", err)
	}

	// Remove some repositories from batch
	if err := db.RemoveRepositoriesFromBatch(ctx, batch.ID, repoIDs[:2]); err != nil {
		t.Fatalf("RemoveRepositoriesFromBatch() error = %v", err)
	}

	// Verify repositories were removed
	repos, err := db.ListRepositories(ctx, map[string]interface{}{"batch_id": batch.ID})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("Expected 1 repository remaining in batch, got %d", len(repos))
	}

	// Verify batch count was updated
	updatedBatch, err := db.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("GetBatch() error = %v", err)
	}
	if updatedBatch.RepositoryCount != 1 {
		t.Errorf("Expected batch repository count 1, got %d", updatedBatch.RepositoryCount)
	}

	// Test with empty IDs
	if err := db.RemoveRepositoriesFromBatch(ctx, batch.ID, []int64{}); err != nil {
		t.Errorf("RemoveRepositoriesFromBatch() with empty IDs should not error, got %v", err)
	}
}

func TestListRepositoriesWithSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch

	// Create test repositories with different names
	repos := []string{
		"acme/frontend-app",
		"acme/backend-api",
		"company/mobile-app",
		"company/web-service",
	}

	for _, name := range repos {
		repo := &models.Repository{
			FullName:      name,
			Source:        "ghes",
			SourceURL:     "https://github.com/" + name,
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
	}

	tests := []struct {
		name     string
		search   string
		expected int
	}{
		{
			name:     "Search for 'app'",
			search:   "app",
			expected: 2,
		},
		{
			name:     "Search for 'acme'",
			search:   "acme",
			expected: 2,
		},
		{
			name:     "Search for 'frontend'",
			search:   "frontend",
			expected: 1,
		},
		{
			name:     "Search for 'service'",
			search:   "service",
			expected: 1,
		},
		{
			name:     "Case insensitive search",
			search:   "ACME",
			expected: 2,
		},
		{
			name:     "No results",
			search:   "nonexistent",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.ListRepositories(ctx, map[string]interface{}{"search": tt.search})
			if err != nil {
				t.Fatalf("ListRepositories() error = %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d repositories, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestListRepositoriesAvailableForBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch

	// Create repositories with different statuses
	testCases := []struct {
		name   string
		status models.MigrationStatus
	}{
		{"org/pending", models.StatusPending},
		{"org/complete", models.StatusComplete},
		{"org/queued", models.StatusQueuedForMigration},
		{"org/dry-run-complete", models.StatusDryRunComplete},
		{"org/dry-run-failed", models.StatusDryRunFailed},
		{"org/migrating", models.StatusMigratingContent},
		{"org/failed", models.StatusMigrationFailed},
	}

	for _, tc := range testCases {
		repo := &models.Repository{
			FullName:      tc.name,
			Source:        "ghes",
			SourceURL:     "https://github.com/" + tc.name,
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(tc.status),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
	}

	// Get repositories available for batch
	results, err := db.ListRepositories(ctx, map[string]interface{}{"available_for_batch": true})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	// Should include: pending, dry_run_complete, dry_run_failed, migration_failed
	// Should exclude: complete, queued_for_migration, migrating_content
	expectedCount := 4
	if len(results) != expectedCount {
		t.Errorf("Expected %d repositories available for batch, got %d", expectedCount, len(results))
	}

	// Verify excluded statuses are not present
	excludedStatuses := map[string]bool{
		string(models.StatusComplete):           true,
		string(models.StatusQueuedForMigration): true,
		string(models.StatusMigratingContent):   true,
	}

	for _, repo := range results {
		if excludedStatuses[repo.Status] {
			t.Errorf("Repository with status %s should not be available for batch", repo.Status)
		}
	}
}

func TestListRepositoriesWithPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := testDefaultBranch

	// Create 10 test repositories
	for i := 0; i < 10; i++ {
		repo := &models.Repository{
			FullName:      fmt.Sprintf("org/repo%02d", i),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
	}

	tests := []struct {
		name     string
		limit    int
		offset   int
		expected int
	}{
		{
			name:     "First page",
			limit:    5,
			offset:   0,
			expected: 5,
		},
		{
			name:     "Second page",
			limit:    5,
			offset:   5,
			expected: 5,
		},
		{
			name:     "Third page (partial)",
			limit:    5,
			offset:   8,
			expected: 2,
		},
		{
			name:     "Beyond available",
			limit:    5,
			offset:   15,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.ListRepositories(ctx, map[string]interface{}{
				"limit":  tt.limit,
				"offset": tt.offset,
			})
			if err != nil {
				t.Fatalf("ListRepositories() error = %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d repositories, got %d", tt.expected, len(results))
			}
		})
	}
}
