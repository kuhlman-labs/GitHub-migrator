package storage

import (
	"context"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestUpdateBatchStatus(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	tests := []struct {
		name               string
		repoStatuses       []string
		expectedStatus     string
		batchInitialStatus string
	}{
		{
			name:               "all pending should be pending",
			repoStatuses:       []string{string(models.StatusPending), string(models.StatusPending)},
			expectedStatus:     "pending",
			batchInitialStatus: "pending",
		},
		{
			name:               "all dry_run_complete should be ready",
			repoStatuses:       []string{string(models.StatusDryRunComplete), string(models.StatusDryRunComplete)},
			expectedStatus:     "ready",
			batchInitialStatus: "pending",
		},
		{
			name:               "mixed pending and dry_run_complete should be pending",
			repoStatuses:       []string{string(models.StatusPending), string(models.StatusDryRunComplete)},
			expectedStatus:     "pending",
			batchInitialStatus: "ready",
		},
		{
			name:               "dry_run_failed should make batch pending",
			repoStatuses:       []string{string(models.StatusDryRunFailed), string(models.StatusDryRunComplete)},
			expectedStatus:     "pending",
			batchInitialStatus: "ready",
		},
		{
			name:               "rolled_back should make batch pending",
			repoStatuses:       []string{string(models.StatusRolledBack), string(models.StatusDryRunComplete)},
			expectedStatus:     "pending",
			batchInitialStatus: "ready",
		},
		{
			name:               "migration_failed should make batch pending",
			repoStatuses:       []string{string(models.StatusMigrationFailed), string(models.StatusDryRunComplete)},
			expectedStatus:     "pending",
			batchInitialStatus: "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a batch
			batch := &models.Batch{
				Name:            "Test Batch - " + tt.name,
				Type:            "pilot",
				Status:          tt.batchInitialStatus,
				RepositoryCount: 0,
				CreatedAt:       time.Now(),
			}
			if err := db.CreateBatch(ctx, batch); err != nil {
				t.Fatalf("CreateBatch() error = %v", err)
			}

			// Create repositories with specified statuses
			repoIDs := make([]int64, 0, len(tt.repoStatuses))
			for i, status := range tt.repoStatuses {
				repo := createTestRepoWithStatus("org/repo-"+tt.name+"-"+string(rune(i)), status)
				if err := db.SaveRepository(ctx, repo); err != nil {
					t.Fatalf("SaveRepository() error = %v", err)
				}
				savedRepo, _ := db.GetRepository(ctx, repo.FullName)
				repoIDs = append(repoIDs, savedRepo.ID)
			}

			// Add repositories to batch (this should trigger status update)
			if err := db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
				t.Fatalf("AddRepositoriesToBatch() error = %v", err)
			}

			// Verify batch status
			updatedBatch, err := db.GetBatch(ctx, batch.ID)
			if err != nil {
				t.Fatalf("GetBatch() error = %v", err)
			}

			if updatedBatch.Status != tt.expectedStatus {
				t.Errorf("Expected batch status '%s', got '%s'", tt.expectedStatus, updatedBatch.Status)
			}
		})
	}
}

func TestUpdateBatchStatusDoesNotAffectInProgressBatches(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create a batch in 'in_progress' state
	batch := &models.Batch{
		Name:            "In Progress Batch",
		Type:            "pilot",
		Status:          batchStatusInProgress,
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create a repository with pending status
	repo := createTestRepoWithStatus("org/test-repo-in-progress", string(models.StatusPending))
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}
	savedRepo, _ := db.GetRepository(ctx, repo.FullName)

	// Add repository to in-progress batch
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{savedRepo.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch() error = %v", err)
	}

	// Verify batch status remains 'in_progress'
	updatedBatch, err := db.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("GetBatch() error = %v", err)
	}

	if updatedBatch.Status != batchStatusInProgress {
		t.Errorf("Expected batch status to remain 'in_progress', got '%s'", updatedBatch.Status)
	}
}

func TestUpdateBatchStatusAfterRepositoryRemoval(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create a batch
	batch := &models.Batch{
		Name:            "Test Removal Batch",
		Type:            "pilot",
		Status:          "pending",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create one pending and one dry_run_complete repo
	repo1 := createTestRepoWithStatus("org/repo-pending", string(models.StatusPending))
	repo2 := createTestRepoWithStatus("org/repo-complete", string(models.StatusDryRunComplete))

	if err := db.SaveRepository(ctx, repo1); err != nil {
		t.Fatalf("SaveRepository(repo1) error = %v", err)
	}
	if err := db.SaveRepository(ctx, repo2); err != nil {
		t.Fatalf("SaveRepository(repo2) error = %v", err)
	}

	savedRepo1, _ := db.GetRepository(ctx, repo1.FullName)
	savedRepo2, _ := db.GetRepository(ctx, repo2.FullName)

	// Add both repos to batch - batch should be pending because repo1 is pending
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{savedRepo1.ID, savedRepo2.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch() error = %v", err)
	}

	// Verify batch is pending
	updatedBatch, _ := db.GetBatch(ctx, batch.ID)
	if updatedBatch.Status != "pending" {
		t.Errorf("Expected batch status 'pending', got '%s'", updatedBatch.Status)
	}

	// Remove pending repo
	if err := db.RemoveRepositoriesFromBatch(ctx, batch.ID, []int64{savedRepo1.ID}); err != nil {
		t.Fatalf("RemoveRepositoriesFromBatch() error = %v", err)
	}

	// Verify batch is now ready (only dry_run_complete repo remains)
	updatedBatch, _ = db.GetBatch(ctx, batch.ID)
	if updatedBatch.Status != "ready" {
		t.Errorf("Expected batch status 'ready' after removing pending repo, got '%s'", updatedBatch.Status)
	}
}

func TestAddingPendingRepoToReadyBatchMakesPending(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create a batch
	batch := &models.Batch{
		Name:            "Ready Batch",
		Type:            "pilot",
		Status:          "ready",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create a dry_run_complete repo
	repo1 := createTestRepoWithStatus("org/repo-complete-1", string(models.StatusDryRunComplete))
	if err := db.SaveRepository(ctx, repo1); err != nil {
		t.Fatalf("SaveRepository(repo1) error = %v", err)
	}
	savedRepo1, _ := db.GetRepository(ctx, repo1.FullName)

	// Add dry_run_complete repo - batch should stay ready
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{savedRepo1.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch(complete) error = %v", err)
	}

	updatedBatch, _ := db.GetBatch(ctx, batch.ID)
	if updatedBatch.Status != "ready" {
		t.Errorf("Expected batch to be 'ready' with dry_run_complete repo, got '%s'", updatedBatch.Status)
	}

	// Now add a pending repo
	repo2 := createTestRepoWithStatus("org/repo-pending-2", string(models.StatusPending))
	if err := db.SaveRepository(ctx, repo2); err != nil {
		t.Fatalf("SaveRepository(repo2) error = %v", err)
	}
	savedRepo2, _ := db.GetRepository(ctx, repo2.FullName)

	// Add pending repo - batch should become pending
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{savedRepo2.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch(pending) error = %v", err)
	}

	updatedBatch, _ = db.GetBatch(ctx, batch.ID)
	if updatedBatch.Status != "pending" {
		t.Errorf("Expected batch to become 'pending' after adding pending repo, got '%s'", updatedBatch.Status)
	}
}
