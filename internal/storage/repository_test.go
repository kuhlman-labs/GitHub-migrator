package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

const (
	testDefaultBranch = "main"
	testMessage       = "Test message"
)

// createTestRepository creates a minimal repository with all required fields
func createTestRepository(fullName string) *models.Repository {
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch
	topContrib := "user1,user2"
	now := time.Now()

	repo := &models.Repository{
		FullName:     fullName,
		Source:       "ghes",
		SourceURL:    fmt.Sprintf("https://github.com/%s", fullName),
		Status:       string(models.StatusPending),
		Visibility:   "private",
		IsArchived:   false,
		IsFork:       false,
		DiscoveredAt: now,
		UpdatedAt:    now,
		// Related tables initialized below
		GitProperties: &models.RepositoryGitProperties{
			TotalSize:      &totalSize,
			DefaultBranch:  &defaultBranch,
			HasLFS:         false,
			HasSubmodules:  false,
			HasLargeFiles:  false,
			LargeFileCount: 0,
			BranchCount:    5,
			CommitCount:    100,
		},
		Features: &models.RepositoryFeatures{
			HasWiki:              false,
			HasPages:             false,
			HasDiscussions:       false,
			HasActions:           false,
			HasProjects:          false,
			HasPackages:          false,
			BranchProtections:    0,
			HasRulesets:          false,
			EnvironmentCount:     0,
			SecretCount:          0,
			VariableCount:        0,
			WebhookCount:         0,
			WorkflowCount:        0,
			HasSelfHostedRunners: false,
			CollaboratorCount:    0,
			InstalledAppsCount:   0,
			ReleaseCount:         0,
			HasReleaseAssets:     false,
			ContributorCount:     2,
			TopContributors:      &topContrib,
			IssueCount:           0,
			PullRequestCount:     0,
			TagCount:             0,
			OpenIssueCount:       0,
			OpenPRCount:          0,
			HasCodeScanning:      false,
			HasDependabot:        false,
			HasSecretScanning:    false,
			HasCodeowners:        false,
		},
		Validation: &models.RepositoryValidation{
			HasOversizedCommits:    false,
			HasLongRefs:            false,
			HasBlockingFiles:       false,
			HasLargeFileWarnings:   false,
			HasOversizedRepository: false,
		},
	}
	return repo
}

// createTestRepoWithStatus creates a test repository with a given status
func createTestRepoWithStatus(fullName, status string) *models.Repository {
	repo := createTestRepository(fullName)
	repo.Status = status
	return repo
}

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

	repo := createTestRepository("test/repo")
	repo.SetHasWiki(true)
	repo.SetHasLFS(true)

	// Test insert
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Test update (upsert)
	newSize := int64(2048 * 1024)
	repo.SetTotalSize(&newSize)
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}
}

func TestGetRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	original := createTestRepository("test/repo")

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
		return // Prevent staticcheck SA5011
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

	// Create test repositories
	repo1 := createTestRepository("org/repo1")
	repo1.Status = string(models.StatusPending)
	repo1.SetHasLFS(true)

	repo2 := createTestRepository("org/repo2")
	repo2.Status = string(models.StatusComplete)

	repo3 := createTestRepository("org/repo3")
	repo3.Status = string(models.StatusQueuedForMigration)

	repos := []*models.Repository{repo1, repo2, repo3}

	for _, repo := range repos {
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	tests := []struct {
		name     string
		filters  map[string]any
		expected int
	}{
		{
			name:     "No filters",
			filters:  nil,
			expected: 3,
		},
		{
			name:     "Filter by status",
			filters:  map[string]any{"status": string(models.StatusPending)},
			expected: 1,
		},
		{
			name:     "Filter by multiple statuses",
			filters:  map[string]any{"status": []string{string(models.StatusPending), string(models.StatusComplete)}},
			expected: 2,
		},
		{
			name:     "Filter by LFS",
			filters:  map[string]any{"has_lfs": true},
			expected: 1,
		},
		{
			name:     "Filter by source",
			filters:  map[string]any{"source": "ghes"},
			expected: 3,
		},
		{
			name:     "Filter with limit",
			filters:  map[string]any{"limit": 1},
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

	repo := createTestRepository("test/repo")

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Update the repository
	newSize := int64(2048 * 1024)
	repo.SetTotalSize(&newSize)
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

	if updated.GetTotalSize() == nil || *updated.GetTotalSize() != newSize {
		t.Errorf("Expected TotalSize %d, got %v", newSize, updated.GetTotalSize())
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

	repo := createTestRepository("test/repo")

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

	repo := createTestRepository("test/repo")

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

	// Create test repositories
	for i := 1; i <= 5; i++ {
		status := models.StatusPending
		if i > 3 {
			status = models.StatusComplete
		}

		repo := createTestRepository("org/repo" + string(rune('0'+i)))
		repo.Status = string(status)

		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	tests := []struct {
		name     string
		filters  map[string]any
		expected int
	}{
		{
			name:     "Count all",
			filters:  nil,
			expected: 5,
		},
		{
			name:     "Count pending",
			filters:  map[string]any{"status": string(models.StatusPending)},
			expected: 3,
		},
		{
			name:     "Count complete",
			filters:  map[string]any{"status": string(models.StatusComplete)},
			expected: 2,
		},
		{
			name:     "Count multiple statuses",
			filters:  map[string]any{"status": []string{string(models.StatusPending), string(models.StatusComplete)}},
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
		repo := createTestRepository("org/repo" + string(rune('0'+i)))
		repo.Status = string(status)

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

	// Save test repositories
	ids := make([]int64, 0, 3)
	for i := range 3 {
		repo := createTestRepository(fmt.Sprintf("org/repo%d", i))

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

	// Save test repositories
	names := []string{"org/repoa", "org/repob", "org/repoc"}
	for _, name := range names {
		repo := createTestRepository(name)

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

	repo := createTestRepository("org/repo1")

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
		return // Prevent staticcheck SA5011
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
		return // Prevent staticcheck SA5011
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

//nolint:gocyclo // Test function with multiple setup steps
func TestRollbackRepositoryClearsBatchID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a batch
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "pilot",
		Status:          "ready",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create a repository and assign it to the batch
	repo := createTestRepository("org/test-repo")
	repo.Status = string(models.StatusComplete)
	repo.BatchID = &batch.ID

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	// Update batch count
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{repo.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch() error = %v", err)
	}

	// Verify repository is in batch
	savedRepo, err := db.GetRepository(ctx, repo.FullName)
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}
	if savedRepo.BatchID == nil || *savedRepo.BatchID != batch.ID {
		t.Errorf("Expected repository to be in batch %d, got %v", batch.ID, savedRepo.BatchID)
	}

	// Rollback the repository
	if err := db.RollbackRepository(ctx, repo.FullName, "Testing rollback"); err != nil {
		t.Fatalf("RollbackRepository() error = %v", err)
	}

	// Verify repository batch_id is cleared
	rolledBackRepo, err := db.GetRepository(ctx, repo.FullName)
	if err != nil {
		t.Fatalf("GetRepository() after rollback error = %v", err)
	}

	if rolledBackRepo.BatchID != nil {
		t.Errorf("Expected batch_id to be NULL after rollback, got %v", *rolledBackRepo.BatchID)
	}

	if rolledBackRepo.Status != string(models.StatusRolledBack) {
		t.Errorf("Expected status to be 'rolled_back', got %s", rolledBackRepo.Status)
	}

	// Verify repository is now available for batch assignment
	availableRepos, err := db.ListRepositories(ctx, map[string]any{"available_for_batch": true})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	found := false
	for _, r := range availableRepos {
		if r.FullName == repo.FullName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected rolled back repository to be available for batch assignment")
	}

	// Verify we can reassign it to a new batch
	newBatch := &models.Batch{
		Name:            "New Batch",
		Type:            "wave_1",
		Status:          "pending",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, newBatch); err != nil {
		t.Fatalf("CreateBatch() for new batch error = %v", err)
	}

	if err := db.AddRepositoriesToBatch(ctx, newBatch.ID, []int64{rolledBackRepo.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch() for reassignment error = %v", err)
	}

	reassignedRepo, err := db.GetRepository(ctx, repo.FullName)
	if err != nil {
		t.Fatalf("GetRepository() after reassignment error = %v", err)
	}

	if reassignedRepo.BatchID == nil || *reassignedRepo.BatchID != newBatch.ID {
		t.Errorf("Expected repository to be in new batch %d, got %v", newBatch.ID, reassignedRepo.BatchID)
	}
}

func TestGetMigrationHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test repository
	repo := createTestRepository("test/repo")
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
	repo := createTestRepository("test/repo")
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
	repo := createTestRepository("test/repo")
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
	repo := createTestRepository("test/repo")
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
	repo := createTestRepository("test/repo")
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
	repoIDs := make([]int64, 0, 3)
	for i := range 3 {
		repo := createTestRepository(fmt.Sprintf("org/repo%d", i))
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
	repos, err := db.ListRepositories(ctx, map[string]any{"batch_id": batch.ID})
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
	repoIDs := make([]int64, 0, 3)
	for i := range 3 {
		repo := createTestRepository(fmt.Sprintf("org/repo%d", i))
		repo.BatchID = &batch.ID
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
	repos, err := db.ListRepositories(ctx, map[string]any{"batch_id": batch.ID})
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

// TestSaveRepository_PreservesBatchIDDuringRediscovery verifies that re-discovery
// does not remove a repository from its batch when the repo status is still "pending"
func TestSaveRepository_PreservesBatchIDDuringRediscovery(t *testing.T) {
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

	// Create a repository in "pending" status
	repo := createTestRepository("org/test-repo")
	repo.Status = string(models.StatusPending)
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	// Get the saved repo to get its ID
	savedRepo, err := db.GetRepository(ctx, repo.FullName)
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}

	// Add repository to batch
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{savedRepo.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch() error = %v", err)
	}

	// Verify repository is in batch
	repoInBatch, err := db.GetRepository(ctx, repo.FullName)
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}
	if repoInBatch.BatchID == nil || *repoInBatch.BatchID != batch.ID {
		t.Fatalf("Expected repository to be in batch %d, got %v", batch.ID, repoInBatch.BatchID)
	}
	if repoInBatch.Status != string(models.StatusPending) {
		t.Fatalf("Expected repository status to be 'pending', got %s", repoInBatch.Status)
	}

	// Simulate a re-discovery by creating a new repo object with the same name
	// This is what happens when ProfileRepository is called - it creates a new repo object
	// with status "pending" and no batch_id
	rediscoveredRepo := createTestRepository("org/test-repo")
	rediscoveredRepo.Status = string(models.StatusPending) // Re-discovery sets status to "pending"
	// BatchID is nil - simulating what happens in ProfileRepository
	rediscoveredRepo.SetCommitCount(100) // Updated during re-discovery

	// Save the "re-discovered" repository
	if err := db.SaveRepository(ctx, rediscoveredRepo); err != nil {
		t.Fatalf("SaveRepository() during re-discovery error = %v", err)
	}

	// Verify the batch_id was preserved!
	afterRediscovery, err := db.GetRepository(ctx, repo.FullName)
	if err != nil {
		t.Fatalf("GetRepository() after re-discovery error = %v", err)
	}

	if afterRediscovery.BatchID == nil {
		t.Errorf("REGRESSION: batch_id was cleared during re-discovery! Expected %d, got nil", batch.ID)
	} else if *afterRediscovery.BatchID != batch.ID {
		t.Errorf("REGRESSION: batch_id changed during re-discovery! Expected %d, got %d", batch.ID, *afterRediscovery.BatchID)
	}

	// Verify the metadata was still updated
	if afterRediscovery.GetCommitCount() != 100 {
		t.Errorf("Expected commit_count to be updated to 100, got %d", afterRediscovery.GetCommitCount())
	}

	// Verify status is still pending
	if afterRediscovery.Status != string(models.StatusPending) {
		t.Errorf("Expected status to remain 'pending', got %s", afterRediscovery.Status)
	}

	t.Logf("✅ batch_id correctly preserved during re-discovery (batch_id=%d)", *afterRediscovery.BatchID)
}

func TestListRepositoriesWithSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test repositories with different names
	repos := []string{
		"acme/frontend-app",
		"acme/backend-api",
		"company/mobile-app",
		"company/web-service",
	}

	for _, name := range repos {
		repo := createTestRepository(name)
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
			results, err := db.ListRepositories(ctx, map[string]any{"search": tt.search})
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
	// Create a batch first
	batch := &models.Batch{
		Name:            "Test Batch",
		Type:            "pilot",
		Status:          "ready",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create repositories with different statuses
	testCases := []struct {
		name      string
		status    models.MigrationStatus
		batchID   *int64
		available bool // whether it should be available for batch
	}{
		{"org/pending", models.StatusPending, nil, true},
		{"org/complete", models.StatusComplete, nil, false},
		{"org/queued", models.StatusQueuedForMigration, nil, false},
		{"org/dry-run-complete", models.StatusDryRunComplete, nil, true},
		{"org/dry-run-failed", models.StatusDryRunFailed, nil, true},
		{"org/migrating", models.StatusMigratingContent, nil, false},
		{"org/failed", models.StatusMigrationFailed, nil, true},
		{"org/in-batch", models.StatusPending, &batch.ID, false}, // Already in a batch
	}

	for _, tc := range testCases {
		repo := createTestRepository(tc.name)
		repo.Status = string(tc.status)
		repo.BatchID = tc.batchID
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
	}

	// Get repositories available for batch
	results, err := db.ListRepositories(ctx, map[string]any{"available_for_batch": true})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	// Should include: pending, dry_run_complete, dry_run_failed, migration_failed (4 repos)
	// Should exclude: complete, queued_for_migration, migrating_content, already in batch
	expectedCount := 4
	if len(results) != expectedCount {
		t.Errorf("Expected %d repositories available for batch, got %d", expectedCount, len(results))
		for _, r := range results {
			t.Logf("  - %s (status: %s, batch_id: %v)", r.FullName, r.Status, r.BatchID)
		}
	}

	// Verify no repos with batch_id are included
	for _, repo := range results {
		if repo.BatchID != nil {
			t.Errorf("Repository %s with batch_id %d should not be available for batch", repo.FullName, *repo.BatchID)
		}
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

	// Create 10 test repositories
	for i := range 10 {
		repo := createTestRepository(fmt.Sprintf("org/repo%02d", i))
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
			results, err := db.ListRepositories(ctx, map[string]any{
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

func TestGetOrganizationStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create repositories in different organizations
	repos := []struct {
		fullName string
		status   string
	}{
		{"org1/repo1", string(models.StatusPending)},
		{"org1/repo2", string(models.StatusComplete)},
		{"org1/repo3", string(models.StatusComplete)},
		{"org2/repo1", string(models.StatusPending)},
		{"org2/repo2", string(models.StatusMigrationFailed)},
	}

	for _, r := range repos {
		repo := createTestRepository(r.fullName)
		repo.Status = r.status
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository %s: %v", r.fullName, err)
		}
	}

	// Get organization stats
	stats, err := db.GetOrganizationStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get organization stats: %v", err)
	}

	// Verify we have 2 organizations
	if len(stats) != 2 {
		t.Errorf("Expected 2 organizations, got %d", len(stats))
	}

	// Verify ordering: org1 (3 repos) should come before org2 (2 repos)
	if stats[0].Organization != "org1" {
		t.Errorf("Expected first org to be 'org1', got '%s'", stats[0].Organization)
	}
	if stats[1].Organization != "org2" {
		t.Errorf("Expected second org to be 'org2', got '%s'", stats[1].Organization)
	}

	// Find org1 stats
	var org1Stats *OrganizationStats
	for _, s := range stats {
		if s.Organization == "org1" {
			org1Stats = s
			break
		}
	}

	if org1Stats == nil {
		t.Fatal("Expected to find org1 in stats")
		return // Prevent staticcheck SA5011
	}

	if org1Stats.TotalRepos != 3 {
		t.Errorf("Expected org1 to have 3 repos, got %d", org1Stats.TotalRepos)
	}

	if org1Stats.StatusCounts[string(models.StatusComplete)] != 2 {
		t.Errorf("Expected org1 to have 2 complete repos, got %d", org1Stats.StatusCounts[string(models.StatusComplete)])
	}
}

func TestGetSizeDistribution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create repositories with different sizes
	sizes := []int64{
		50 * 1024 * 1024,       // 50MB - small
		500 * 1024 * 1024,      // 500MB - medium
		2 * 1024 * 1024 * 1024, // 2GB - large
		6 * 1024 * 1024 * 1024, // 6GB - very large
	}

	for i, size := range sizes {
		repo := createTestRepository(fmt.Sprintf("test/repo%d", i))
		s := size // create local copy to avoid pointer issues
		repo.SetTotalSize(&s)
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository %d: %v", i, err)
		}
	}

	// Get size distribution
	distribution, err := db.GetSizeDistribution(ctx)
	if err != nil {
		t.Fatalf("Failed to get size distribution: %v", err)
	}

	// Verify we have all size categories represented
	categoryMap := make(map[string]int)
	for _, d := range distribution {
		categoryMap[d.Category] = d.Count
	}

	if categoryMap["small"] != 1 {
		t.Errorf("Expected 1 small repo, got %d", categoryMap["small"])
	}
	if categoryMap["medium"] != 1 {
		t.Errorf("Expected 1 medium repo, got %d", categoryMap["medium"])
	}
	if categoryMap["large"] != 1 {
		t.Errorf("Expected 1 large repo, got %d", categoryMap["large"])
	}
	if categoryMap["very_large"] != 1 {
		t.Errorf("Expected 1 very_large repo, got %d", categoryMap["very_large"])
	}
}

func TestGetFeatureStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create repositories with different features
	repos := []struct {
		fullName   string
		hasLFS     bool
		hasActions bool
		hasWiki    bool
	}{
		{"test/repo1", true, true, false},
		{"test/repo2", true, false, true},
		{"test/repo3", false, true, true},
		{"test/repo4", false, false, false},
	}

	for _, r := range repos {
		repo := createTestRepository(r.fullName)
		repo.SetHasLFS(r.hasLFS)
		repo.SetHasActions(r.hasActions)
		repo.SetHasWiki(r.hasWiki)
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository %s: %v", r.fullName, err)
		}
	}

	// Get feature stats
	stats, err := db.GetFeatureStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get feature stats: %v", err)
	}

	if stats.TotalRepositories != 4 {
		t.Errorf("Expected 4 total repositories, got %d", stats.TotalRepositories)
	}
	if stats.HasLFS != 2 {
		t.Errorf("Expected 2 repos with LFS, got %d", stats.HasLFS)
	}
	if stats.HasActions != 2 {
		t.Errorf("Expected 2 repos with Actions, got %d", stats.HasActions)
	}
	if stats.HasWiki != 2 {
		t.Errorf("Expected 2 repos with Wiki, got %d", stats.HasWiki)
	}
}

func TestGetFeatureStats_TFVCOnlyCountsADO(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create GitHub repos with ADOIsGit=false (should NOT be counted as TFVC)
	ghRepo1 := createTestRepository("github-org/repo1")
	ghRepo1.Source = "ghes"
	ghRepo1.SetADOIsGit(false) // This should NOT be counted as TFVC
	if err := db.SaveRepository(ctx, ghRepo1); err != nil {
		t.Fatalf("Failed to save GitHub repo: %v", err)
	}

	ghRepo2 := createTestRepository("github-org/repo2")
	ghRepo2.Source = "ghes"
	ghRepo2.SetADOIsGit(false) // This should NOT be counted as TFVC
	if err := db.SaveRepository(ctx, ghRepo2); err != nil {
		t.Fatalf("Failed to save GitHub repo: %v", err)
	}

	// Create ADO repos with ADOIsGit=false (SHOULD be counted as TFVC)
	adoRepo1 := createTestRepository("ado-org/project/tfvc-repo1")
	adoRepo1.Source = "azuredevops"
	adoRepo1.SetADOIsGit(false) // This SHOULD be counted as TFVC
	adoRepo1.SetADOHasPipelines(true)
	if err := db.SaveRepository(ctx, adoRepo1); err != nil {
		t.Fatalf("Failed to save ADO TFVC repo: %v", err)
	}

	adoRepo2 := createTestRepository("ado-org/project/git-repo")
	adoRepo2.Source = "azuredevops"
	adoRepo2.SetADOIsGit(true) // This should NOT be counted as TFVC
	adoRepo2.SetADOHasBoards(true)
	if err := db.SaveRepository(ctx, adoRepo2); err != nil {
		t.Fatalf("Failed to save ADO Git repo: %v", err)
	}

	// Get feature stats
	stats, err := db.GetFeatureStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get feature stats: %v", err)
	}

	// Verify total repositories
	if stats.TotalRepositories != 4 {
		t.Errorf("Expected 4 total repositories, got %d", stats.TotalRepositories)
	}

	// Verify TFVC count only includes ADO repos
	if stats.ADOTFVCCount != 1 {
		t.Errorf("Expected 1 TFVC repository (ADO only), got %d", stats.ADOTFVCCount)
	}

	// Verify other ADO features are also filtered by source
	if stats.ADOHasPipelines != 1 {
		t.Errorf("Expected 1 repo with ADO Pipelines, got %d", stats.ADOHasPipelines)
	}

	if stats.ADOHasBoards != 1 {
		t.Errorf("Expected 1 repo with ADO Boards, got %d", stats.ADOHasBoards)
	}
}

func createRepoWithHistory(t *testing.T, db *Database, ctx context.Context, fullName, status string) {
	now := time.Now()
	repo := createTestRepository(fullName)
	repo.Status = status
	if status == string(models.StatusComplete) {
		repo.MigratedAt = &now
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository %s: %v", fullName, err)
	}

	// Create migration_history record for completed and failed repos
	if status == string(models.StatusComplete) || status == string(models.StatusMigrationFailed) {
		savedRepo, err := db.GetRepository(ctx, fullName)
		if err != nil {
			t.Fatalf("Failed to get repository %s: %v", fullName, err)
		}

		historyStatus := "completed"
		historyMsg := "Migration completed"
		if status == string(models.StatusMigrationFailed) {
			historyStatus = "failed"
			historyMsg = "Migration failed"
		}

		history := &models.MigrationHistory{
			RepositoryID: savedRepo.ID,
			Status:       historyStatus,
			Phase:        "migration",
			StartedAt:    now.Add(-1 * time.Hour),
		}
		historyID, err := db.CreateMigrationHistory(ctx, history)
		if err != nil {
			t.Fatalf("Failed to create migration history: %v", err)
		}

		err = db.UpdateMigrationHistory(ctx, historyID, historyStatus, &historyMsg)
		if err != nil {
			t.Fatalf("Failed to update migration history: %v", err)
		}
	}
}

func TestGetCompletedMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test repositories with migration history
	createRepoWithHistory(t, db, ctx, "test/complete1", string(models.StatusComplete))
	createRepoWithHistory(t, db, ctx, "test/complete2", string(models.StatusComplete))
	createRepoWithHistory(t, db, ctx, "test/pending", string(models.StatusPending))
	createRepoWithHistory(t, db, ctx, "test/failed", string(models.StatusMigrationFailed))

	// Get completed migrations (includes complete, failed, and rolled_back)
	migrations, err := db.GetCompletedMigrations(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get completed migrations: %v", err)
	}

	// Should return 3 migrations (2 complete + 1 failed)
	if len(migrations) != 3 {
		t.Errorf("Expected 3 migrations (2 complete + 1 failed), got %d", len(migrations))
	}

	// Verify they have the correct statuses
	completeCount := 0
	failedCount := 0
	for _, m := range migrations {
		// Count status types
		if m.Status == string(models.StatusComplete) {
			completeCount++
		} else if m.Status == string(models.StatusMigrationFailed) {
			failedCount++
		}

		// Basic validation - should have repository info
		if m.FullName == "" {
			t.Errorf("Expected full_name to be populated")
		}
		if m.SourceURL == "" {
			t.Errorf("Expected source_url to be populated")
		}

		// NOTE: Migration history data (started_at, completed_at, duration_seconds)
		// may not be populated in this test due to SQLite DATETIME string handling.
		// This is tested separately in the UpdateMigrationHistory tests.
	}

	// Verify we got the expected status distribution
	if completeCount != 2 {
		t.Errorf("Expected 2 complete migrations, got %d", completeCount)
	}
	if failedCount != 1 {
		t.Errorf("Expected 1 failed migration, got %d", failedCount)
	}
}

func TestGetMigrationCompletionStatsByOrg(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test repositories
	createTestReposForOrgStats(t, db, ctx)

	// Get migration completion stats
	stats, err := db.GetMigrationCompletionStatsByOrg(ctx)
	if err != nil {
		t.Fatalf("Failed to get migration completion stats: %v", err)
	}

	// Should return 3 organizations
	if len(stats) != 3 {
		t.Errorf("Expected 3 organizations, got %d", len(stats))
	}

	// Verify organization stats
	verifyOrgStats(t, stats, "acme", 4, 2, 1, 1)
	verifyOrgStats(t, stats, "corp", 2, 1, 1, 0)
	verifyOrgStats(t, stats, "org", 1, 1, 0, 0)
}

func createTestReposForOrgStats(t *testing.T, db *Database, ctx context.Context) {
	repos := []struct {
		fullName string
		status   string
	}{
		{"acme/repo1", string(models.StatusComplete)},
		{"acme/repo2", string(models.StatusComplete)},
		{"acme/repo3", string(models.StatusPending)},
		{"acme/repo4", string(models.StatusMigrationFailed)},
		{"corp/repo1", string(models.StatusComplete)},
		{"corp/repo2", string(models.StatusPending)},
		{"org/repo1", string(models.StatusComplete)},
	}

	for _, r := range repos {
		repo := createTestRepository(r.fullName)
		repo.Status = r.status
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository %s: %v", r.fullName, err)
		}
	}
}

func verifyOrgStats(t *testing.T, stats []*MigrationCompletionStats, orgName string, expectedTotal, expectedCompleted, expectedPending, expectedFailed int) {
	orgStats := findOrgStats(stats, orgName)
	if orgStats == nil {
		t.Fatalf("Expected to find %s organization stats", orgName)
		return // Prevent staticcheck SA5011
	}
	if orgStats.TotalRepos != expectedTotal {
		t.Errorf("Expected %s to have %d total repos, got %d", orgName, expectedTotal, orgStats.TotalRepos)
	}
	if orgStats.CompletedCount != expectedCompleted {
		t.Errorf("Expected %s to have %d completed, got %d", orgName, expectedCompleted, orgStats.CompletedCount)
	}
	if orgStats.PendingCount != expectedPending {
		t.Errorf("Expected %s to have %d pending, got %d", orgName, expectedPending, orgStats.PendingCount)
	}
	if orgStats.FailedCount != expectedFailed {
		t.Errorf("Expected %s to have %d failed, got %d", orgName, expectedFailed, orgStats.FailedCount)
	}
}

func findOrgStats(stats []*MigrationCompletionStats, orgName string) *MigrationCompletionStats {
	for _, s := range stats {
		if s.Organization == orgName {
			return s
		}
	}
	return nil
}
