package storage

import (
	"context"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/models"
)

func TestDeleteBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := "main"

	// Create a batch
	batch := &models.Batch{
		Name:            "Test Batch to Delete",
		Type:            "pilot",
		Status:          "pending",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create repositories and add to batch
	var repoIDs []int64
	for i := 0; i < 3; i++ {
		repo := &models.Repository{
			FullName:      "org/delete-test-repo-" + string(rune('a'+i)),
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
		savedRepo, _ := db.GetRepository(ctx, repo.FullName)
		repoIDs = append(repoIDs, savedRepo.ID)
	}

	// Add repositories to batch
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
		t.Fatalf("AddRepositoriesToBatch() error = %v", err)
	}

	// Verify repositories are in the batch
	for _, repoID := range repoIDs {
		repo, err := db.GetRepositoryByID(ctx, repoID)
		if err != nil {
			t.Fatalf("GetRepositoryByID() error = %v", err)
		}
		if repo.BatchID == nil || *repo.BatchID != batch.ID {
			t.Errorf("Expected repository %d to be in batch %d, got %v", repoID, batch.ID, repo.BatchID)
		}
	}

	// Delete the batch
	if err := db.DeleteBatch(ctx, batch.ID); err != nil {
		t.Fatalf("DeleteBatch() error = %v", err)
	}

	// Verify batch is deleted
	deletedBatch, err := db.GetBatch(ctx, batch.ID)
	if err == nil && deletedBatch != nil {
		t.Errorf("Expected batch to be deleted, but still exists")
	}

	// Verify repositories' batch_id is cleared
	for _, repoID := range repoIDs {
		repo, err := db.GetRepositoryByID(ctx, repoID)
		if err != nil {
			t.Fatalf("GetRepositoryByID() after delete error = %v", err)
		}
		if repo.BatchID != nil {
			t.Errorf("Expected repository %d batch_id to be NULL after batch deletion, got %v", repoID, *repo.BatchID)
		}
	}
}

func TestDeleteBatchNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Try to delete a non-existent batch
	err := db.DeleteBatch(ctx, 99999)
	if err == nil {
		t.Errorf("Expected error when deleting non-existent batch, got nil")
	}
}

func TestDeleteEmptyBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create an empty batch
	batch := &models.Batch{
		Name:            "Empty Batch",
		Type:            "pilot",
		Status:          "pending",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Delete the empty batch
	if err := db.DeleteBatch(ctx, batch.ID); err != nil {
		t.Fatalf("DeleteBatch() error = %v", err)
	}

	// Verify batch is deleted
	deletedBatch, err := db.GetBatch(ctx, batch.ID)
	if err == nil && deletedBatch != nil {
		t.Errorf("Expected batch to be deleted, but still exists")
	}
}

func TestDeleteBatchMakesRepositoriesAvailable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024)
	defaultBranch := "main"

	// Create batch
	batch := &models.Batch{
		Name:            "Batch to Delete",
		Type:            "pilot",
		Status:          "pending",
		RepositoryCount: 0,
		CreatedAt:       time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}

	// Create repository
	repo := &models.Repository{
		FullName:      "org/test-available-repo",
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
	savedRepo, _ := db.GetRepository(ctx, repo.FullName)

	// Add to batch
	if err := db.AddRepositoriesToBatch(ctx, batch.ID, []int64{savedRepo.ID}); err != nil {
		t.Fatalf("AddRepositoriesToBatch() error = %v", err)
	}

	// Verify not available for batch
	availableRepos, err := db.ListRepositories(ctx, map[string]interface{}{
		"available_for_batch": true,
	})
	if err != nil {
		t.Fatalf("ListRepositories(available_for_batch) error = %v", err)
	}
	for _, r := range availableRepos {
		if r.ID == savedRepo.ID {
			t.Errorf("Repository should not be available for batch when already assigned")
		}
	}

	// Delete batch
	if err := db.DeleteBatch(ctx, batch.ID); err != nil {
		t.Fatalf("DeleteBatch() error = %v", err)
	}

	// Verify now available for batch
	availableRepos, err = db.ListRepositories(ctx, map[string]interface{}{
		"available_for_batch": true,
	})
	if err != nil {
		t.Fatalf("ListRepositories(available_for_batch) after delete error = %v", err)
	}

	found := false
	for _, r := range availableRepos {
		if r.ID == savedRepo.ID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Repository should be available for batch after batch deletion")
	}
}
