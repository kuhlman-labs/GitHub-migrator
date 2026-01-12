package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestListBatches(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create test batches
	batch1 := &models.Batch{Name: "Batch 1", Type: "pilot", Status: "ready", CreatedAt: time.Now()}
	batch2 := &models.Batch{Name: "Batch 2", Type: "wave", Status: "ready", CreatedAt: time.Now()}
	_ = db.CreateBatch(ctx, batch1)
	_ = db.CreateBatch(ctx, batch2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/batches", nil)
	w := httptest.NewRecorder()

	h.ListBatches(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var batches []*models.Batch
	if err := json.NewDecoder(w.Body).Decode(&batches); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(batches) != 2 {
		t.Errorf("Expected 2 batches, got %d", len(batches))
	}
}

//nolint:gocyclo // Test function with multiple test cases naturally has high complexity
func TestCreateBatch(t *testing.T) {
	h, _ := setupTestHandler(t)

	t.Run("valid batch", func(t *testing.T) {
		desc := "Test Description"
		batch := models.Batch{
			Name:        "Test Batch",
			Description: &desc,
			Type:        "pilot",
		}
		body, _ := json.Marshal(batch)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.CreateBatch(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}

		var created models.Batch
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.Name != "Test Batch" {
			t.Errorf("Expected name 'Test Batch', got '%s'", created.Name)
		}
		if created.Status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", created.Status)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader([]byte("invalid")))
		w := httptest.NewRecorder()

		h.CreateBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("duplicate batch name", func(t *testing.T) {
		// Create first batch
		desc := "Original Description"
		batch1 := models.Batch{
			Name:        "Duplicate Test Batch",
			Description: &desc,
			Type:        "pilot",
		}
		body1, _ := json.Marshal(batch1)

		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader(body1))
		w1 := httptest.NewRecorder()

		h.CreateBatch(w1, req1)

		if w1.Code != http.StatusCreated {
			t.Fatalf("Expected first batch to be created, got status %d", w1.Code)
		}

		// Attempt to create second batch with same name
		desc2 := "Duplicate Description"
		batch2 := models.Batch{
			Name:        "Duplicate Test Batch",
			Description: &desc2,
			Type:        "wave",
		}
		body2, _ := json.Marshal(batch2)

		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader(body2))
		w2 := httptest.NewRecorder()

		h.CreateBatch(w2, req2)

		if w2.Code != http.StatusConflict {
			t.Errorf("Expected status %d (Conflict), got %d", http.StatusConflict, w2.Code)
		}

		// Verify error message mentions the batch name
		var errResp map[string]string
		if err := json.NewDecoder(w2.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		// Check that the error message indicates a conflict (already exists)
		// The standardized APIError uses "Resource already exists" as the base message
		if !strings.Contains(errResp["error"], "already exists") && !strings.Contains(errResp["error"], "conflict") && !strings.Contains(errResp["error"], "Conflict") {
			t.Errorf("Error message should indicate resource already exists, got: %s", errResp["error"])
		}
	})

	t.Run("empty batch name", func(t *testing.T) {
		batch := models.Batch{
			Name: "   ", // Empty/whitespace-only name
			Type: "pilot",
		}
		body, _ := json.Marshal(batch)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.CreateBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d (Bad Request), got %d", http.StatusBadRequest, w.Code)
		}

		var errResp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		// Check that the error message indicates a missing/required field
		// The standardized APIError uses "Required field is missing" as the base message
		if !strings.Contains(errResp["error"], "required") && !strings.Contains(errResp["error"], "missing") && !strings.Contains(errResp["error"], "Missing") {
			t.Errorf("Error message should indicate name is required/missing, got: %s", errResp["error"])
		}
	})
}

func TestDeleteBatch(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	t.Run("delete existing batch", func(t *testing.T) {
		// Create a batch
		batch := &models.Batch{
			Name:      "Test Batch to Delete",
			Type:      "pilot",
			Status:    "pending",
			CreatedAt: time.Now(),
		}
		if err := db.CreateBatch(ctx, batch); err != nil {
			t.Fatalf("Failed to create batch: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/batches/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		h.DeleteBatch(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify batch is deleted
		deletedBatch, err := db.GetBatch(ctx, batch.ID)
		if err == nil && deletedBatch != nil {
			t.Errorf("Expected batch to be deleted")
		}
	})

	t.Run("delete non-existent batch", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/batches/99999", nil)
		req.SetPathValue("id", "99999")
		w := httptest.NewRecorder()

		h.DeleteBatch(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("delete batch in progress", func(t *testing.T) {
		// Create a batch in progress
		batch := &models.Batch{
			Name:      "In Progress Batch",
			Type:      "pilot",
			Status:    "in_progress",
			CreatedAt: time.Now(),
		}
		if err := db.CreateBatch(ctx, batch); err != nil {
			t.Fatalf("Failed to create batch: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/batches/"+fmt.Sprint(batch.ID), nil)
		req.SetPathValue("id", fmt.Sprint(batch.ID))
		w := httptest.NewRecorder()

		h.DeleteBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestGetBatch(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	batch := &models.Batch{Name: "Test Batch", Type: "pilot", Status: "ready", CreatedAt: time.Now()}
	_ = db.CreateBatch(ctx, batch)

	t.Run("existing batch", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["batch"] == nil {
			t.Error("Expected batch in response")
		}
		// Repositories will be an empty array if batch has no repos
		_, hasRepos := response["repositories"]
		if !hasRepos {
			t.Error("Expected repositories in response")
		}
	})

	t.Run("batch not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/999", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("invalid batch ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/invalid", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestStartBatch(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create batch
	batch := &models.Batch{Name: "Test Batch", Type: "pilot", Status: "ready", CreatedAt: time.Now()}
	_ = db.CreateBatch(ctx, batch)

	// Add repositories to batch
	repo1 := &models.Repository{FullName: "org/repo1", Status: string(models.StatusPending)}
	_ = db.SaveRepository(ctx, repo1)
	batchID := batch.ID
	repo1.BatchID = &batchID
	_ = db.UpdateRepository(ctx, repo1)

	t.Run("successful batch start", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/1/start", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		h.StartBatch(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["count"] != float64(1) {
			t.Errorf("Expected count 1, got %v", response["count"])
		}
	})

	t.Run("batch not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/999/start", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		h.StartBatch(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("invalid batch ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/invalid/start", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		h.StartBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestStartBatchErrors(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create batch without repositories
	batch := &models.Batch{Name: "Empty Batch", Type: "pilot", Status: "ready", CreatedAt: time.Now()}
	_ = db.CreateBatch(ctx, batch)

	t.Run("batch with no repositories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/1/start", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		h.StartBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestUpdateBatch(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a test batch
	desc := "Original description"
	batch := &models.Batch{
		Name:        "Test Batch",
		Description: &desc,
		Type:        "pilot",
		Status:      "ready",
		CreatedAt:   time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	t.Run("successful update", func(t *testing.T) {
		newDesc := "Updated description"
		updates := map[string]any{
			"name":        "Updated Batch",
			"description": newDesc,
			"type":        "wave_1",
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/batches/%d", batch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", batch.ID))
		w := httptest.NewRecorder()

		h.UpdateBatch(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.Batch
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != "Updated Batch" {
			t.Errorf("Expected name 'Updated Batch', got '%s'", response.Name)
		}
	})

	t.Run("cannot update non-ready batch", func(t *testing.T) {
		// Create a batch with in_progress status
		ipBatch := &models.Batch{
			Name:      "In Progress Batch",
			Type:      "pilot",
			Status:    "in_progress",
			CreatedAt: time.Now(),
		}
		if err := db.CreateBatch(ctx, ipBatch); err != nil {
			t.Fatalf("Failed to create batch: %v", err)
		}

		updates := map[string]any{
			"name": "Should Not Update",
		}

		body, _ := json.Marshal(updates)
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/batches/%d", ipBatch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", ipBatch.ID))
		w := httptest.NewRecorder()

		h.UpdateBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestUpdateBatchDestinationOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a batch with a destination org
	oldDestOrg := "old-org"
	destBatch := &models.Batch{
		Name:           "Dest Test Batch",
		Type:           "pilot",
		Status:         "ready",
		DestinationOrg: &oldDestOrg,
		CreatedAt:      time.Now(),
	}
	if err := db.CreateBatch(ctx, destBatch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create repositories with batch default destination
	totalSize := int64(1024)
	defaultBranch := testMainBranch
	repo1Dest := "old-org/repo1"
	repo2Dest := "old-org/repo2"
	customDest := "custom-org/repo3"

	repo1 := &models.Repository{
		FullName:            "source-org/repo1",
		Source:              "ghes",
		SourceURL:           "https://github.com/source-org/repo1",
		Status:              string(models.StatusPending),
		Visibility:          "private",
		BatchID:             &destBatch.ID,
		DestinationFullName: &repo1Dest,
		DiscoveredAt:        time.Now(),
		UpdatedAt:           time.Now(),
	}
	repo1.SetTotalSize(&totalSize)
	repo1.SetDefaultBranch(&defaultBranch)
	repo2 := &models.Repository{
		FullName:            "source-org/repo2",
		Source:              "ghes",
		SourceURL:           "https://github.com/source-org/repo2",
		Status:              string(models.StatusPending),
		Visibility:          "private",
		BatchID:             &destBatch.ID,
		DestinationFullName: &repo2Dest,
		DiscoveredAt:        time.Now(),
		UpdatedAt:           time.Now(),
	}
	repo2.SetTotalSize(&totalSize)
	repo2.SetDefaultBranch(&defaultBranch)
	repo3 := &models.Repository{
		FullName:            "source-org/repo3",
		Source:              "ghes",
		SourceURL:           "https://github.com/source-org/repo3",
		Status:              string(models.StatusPending),
		Visibility:          "private",
		BatchID:             &destBatch.ID,
		DestinationFullName: &customDest, // Custom destination, should not be updated
		DiscoveredAt:        time.Now(),
		UpdatedAt:           time.Now(),
	}
	repo3.SetTotalSize(&totalSize)
	repo3.SetDefaultBranch(&defaultBranch)

	if err := db.SaveRepository(ctx, repo1); err != nil {
		t.Fatalf("Failed to save repo1: %v", err)
	}
	if err := db.SaveRepository(ctx, repo2); err != nil {
		t.Fatalf("Failed to save repo2: %v", err)
	}
	if err := db.SaveRepository(ctx, repo3); err != nil {
		t.Fatalf("Failed to save repo3: %v", err)
	}

	// Update batch destination org
	newDestOrg := "new-org"
	updates := map[string]any{
		"destination_org": newDestOrg,
	}

	body, _ := json.Marshal(updates)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/batches/%d", destBatch.ID), bytes.NewReader(body))
	req.SetPathValue("id", fmt.Sprintf("%d", destBatch.ID))
	w := httptest.NewRecorder()

	h.UpdateBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify repositories with old batch default were updated
	updatedRepo1, err := db.GetRepository(ctx, "source-org/repo1")
	if err != nil {
		t.Fatalf("Failed to get repo1: %v", err)
	}
	if updatedRepo1.DestinationFullName == nil || *updatedRepo1.DestinationFullName != "new-org/repo1" {
		t.Errorf("Expected repo1 destination 'new-org/repo1', got '%v'", updatedRepo1.DestinationFullName)
	}

	updatedRepo2, err := db.GetRepository(ctx, "source-org/repo2")
	if err != nil {
		t.Fatalf("Failed to get repo2: %v", err)
	}
	if updatedRepo2.DestinationFullName == nil || *updatedRepo2.DestinationFullName != "new-org/repo2" {
		t.Errorf("Expected repo2 destination 'new-org/repo2', got '%v'", updatedRepo2.DestinationFullName)
	}

	// Verify custom destination was NOT updated
	updatedRepo3, err := db.GetRepository(ctx, "source-org/repo3")
	if err != nil {
		t.Fatalf("Failed to get repo3: %v", err)
	}
	if updatedRepo3.DestinationFullName == nil || *updatedRepo3.DestinationFullName != "custom-org/repo3" {
		t.Errorf("Expected repo3 destination 'custom-org/repo3' (unchanged), got '%v'", updatedRepo3.DestinationFullName)
	}
}

func TestAddRepositoriesToBatch(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a test batch
	batch := &models.Batch{
		Name:      "Test Batch",
		Type:      "pilot",
		Status:    "ready",
		CreatedAt: time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create test repositories
	totalSize := int64(1024)
	defaultBranch := testMainBranch
	repoIDs := make([]int64, 0, 3)
	for i := range 3 {
		repo := &models.Repository{
			FullName:     fmt.Sprintf("org/repo%d", i),
			Source:       "ghes",
			SourceURL:    "https://github.com/org/repo",
			Status:       string(models.StatusPending),
			Visibility:   "private",
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		repo.SetTotalSize(&totalSize)
		repo.SetDefaultBranch(&defaultBranch)
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
		saved, err := db.GetRepository(ctx, repo.FullName)
		if err != nil || saved == nil {
			t.Fatalf("GetRepository() error = %v, saved = %v", err, saved)
		}
		repoIDs = append(repoIDs, saved.ID)
	}

	t.Run("successful add", func(t *testing.T) {
		reqBody := map[string]any{
			"repository_ids": repoIDs,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/batches/%d/repositories", batch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", batch.ID))
		w := httptest.NewRecorder()

		h.AddRepositoriesToBatch(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if int(response["repositories_added"].(float64)) != len(repoIDs) {
			t.Errorf("Expected %d repositories added, got %v", len(repoIDs), response["repositories_added"])
		}
	})

	t.Run("cannot add to non-ready batch", func(t *testing.T) {
		// Create a batch with in_progress status
		ipBatch := &models.Batch{
			Name:      "In Progress Batch",
			Type:      "pilot",
			Status:    "in_progress",
			CreatedAt: time.Now(),
		}
		if err := db.CreateBatch(ctx, ipBatch); err != nil {
			t.Fatalf("Failed to create batch: %v", err)
		}

		reqBody := map[string]any{
			"repository_ids": repoIDs,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/batches/%d/repositories", ipBatch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", ipBatch.ID))
		w := httptest.NewRecorder()

		h.AddRepositoriesToBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestRemoveRepositoriesFromBatch(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a test batch
	batch := &models.Batch{
		Name:      "Test Batch",
		Type:      "pilot",
		Status:    "ready",
		CreatedAt: time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create and assign repositories to batch
	totalSize := int64(1024)
	defaultBranch := testMainBranch
	repoIDs := make([]int64, 0, 3)
	for i := range 3 {
		repo := &models.Repository{
			FullName:     fmt.Sprintf("org/repo%d", i),
			Source:       "ghes",
			SourceURL:    "https://github.com/org/repo",
			Status:       string(models.StatusPending),
			Visibility:   "private",
			BatchID:      &batch.ID,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		repo.SetTotalSize(&totalSize)
		repo.SetDefaultBranch(&defaultBranch)
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
		saved, _ := db.GetRepository(ctx, repo.FullName)
		repoIDs = append(repoIDs, saved.ID)
	}

	batch.RepositoryCount = 3
	_ = db.UpdateBatch(ctx, batch)

	t.Run("successful remove", func(t *testing.T) {
		reqBody := map[string]any{
			"repository_ids": repoIDs[:2],
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/batches/%d/repositories", batch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", batch.ID))
		w := httptest.NewRecorder()

		h.RemoveRepositoriesFromBatch(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if int(response["repositories_removed"].(float64)) != 2 {
			t.Errorf("Expected 2 repositories removed, got %v", response["repositories_removed"])
		}
	})
}

func TestRetryBatchFailures(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a test batch
	batch := &models.Batch{
		Name:      "Test Batch",
		Type:      "pilot",
		Status:    "in_progress",
		CreatedAt: time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create failed repositories in the batch
	totalSize := int64(1024)
	defaultBranch := testMainBranch
	failedRepoIDs := make([]int64, 0, 2)
	for i := range 2 {
		repo := &models.Repository{
			FullName:     fmt.Sprintf("org/failed-repo%d", i),
			Source:       "ghes",
			SourceURL:    "https://github.com/org/failed-repo",
			Status:       string(models.StatusMigrationFailed),
			Visibility:   "private",
			BatchID:      &batch.ID,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		repo.SetTotalSize(&totalSize)
		repo.SetDefaultBranch(&defaultBranch)
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
		saved, _ := db.GetRepository(ctx, repo.FullName)
		failedRepoIDs = append(failedRepoIDs, saved.ID)
	}

	t.Run("retry all failures", func(t *testing.T) {
		reqBody := map[string]any{}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/batches/%d/retry", batch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", batch.ID))
		w := httptest.NewRecorder()

		h.RetryBatchFailures(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if int(response["retried_count"].(float64)) != 2 {
			t.Errorf("Expected 2 repositories retried, got %v", response["retried_count"])
		}
	})

	t.Run("retry selected failures", func(t *testing.T) {
		// Reset failed repos
		for _, id := range failedRepoIDs {
			repo, _ := db.GetRepositoryByID(ctx, id)
			repo.Status = string(models.StatusMigrationFailed)
			_ = db.UpdateRepository(ctx, repo)
		}

		reqBody := map[string]any{
			"repository_ids": []int64{failedRepoIDs[0]},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/batches/%d/retry", batch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", batch.ID))
		w := httptest.NewRecorder()

		h.RetryBatchFailures(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if int(response["retried_count"].(float64)) != 1 {
			t.Errorf("Expected 1 repository retried, got %v", response["retried_count"])
		}
	})
}
