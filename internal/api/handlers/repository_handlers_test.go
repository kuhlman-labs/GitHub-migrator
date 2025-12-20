package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestListRepositories(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Add test repositories
	repo1 := &models.Repository{
		FullName:      "org/repo1",
		Status:        string(models.StatusPending),
		Source:        "github-enterprise",
		HasLFS:        true,
		HasSubmodules: false,
	}
	repo2 := &models.Repository{
		FullName:      "org/repo2",
		Status:        string(models.StatusComplete),
		Source:        "github-enterprise",
		HasLFS:        false,
		HasSubmodules: true,
	}
	db.SaveRepository(ctx, repo1)
	db.SaveRepository(ctx, repo2)

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		expectedStatus string
	}{
		{
			name:          "no filters",
			query:         "",
			expectedCount: 2,
		},
		{
			name:           "filter by status",
			query:          "?status=pending",
			expectedCount:  1,
			expectedStatus: "pending",
		},
		{
			name:          "filter by source",
			query:         "?source=github-enterprise",
			expectedCount: 2,
		},
		{
			name:          "filter by has_lfs",
			query:         "?has_lfs=true",
			expectedCount: 1,
		},
		{
			name:          "filter by has_submodules",
			query:         "?has_submodules=true",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories"+tt.query, nil)
			w := httptest.NewRecorder()

			h.ListRepositories(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			var response struct {
				Repositories []*models.Repository `json:"repositories"`
				Total        *int                 `json:"total,omitempty"`
			}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(response.Repositories) != tt.expectedCount {
				t.Errorf("Expected %d repositories, got %d", tt.expectedCount, len(response.Repositories))
			}

			if tt.expectedStatus != "" && len(response.Repositories) > 0 {
				if response.Repositories[0].Status != tt.expectedStatus {
					t.Errorf("Expected status %s, got %s", tt.expectedStatus, response.Repositories[0].Status)
				}
			}
		})
	}
}

func TestGetRepository(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	repo := &models.Repository{
		FullName: "org/test-repo",
		Status:   string(models.StatusPending),
	}
	db.SaveRepository(ctx, repo)

	t.Run("existing repository", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories/org/test-repo", nil)
		req.SetPathValue("fullName", "org/test-repo")
		w := httptest.NewRecorder()

		h.GetRepository(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["repository"] == nil {
			t.Error("Expected repository in response")
		}
		// History will be an empty array if no history exists
		_, hasHistory := response["history"]
		if !hasHistory {
			t.Error("Expected history in response")
		}
	})

	t.Run("repository not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories/org/nonexistent", nil)
		req.SetPathValue("fullName", "org/nonexistent")
		w := httptest.NewRecorder()

		h.GetRepository(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("missing fullName", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories/", nil)
		w := httptest.NewRecorder()

		h.GetRepository(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestUpdateRepository(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a batch first for the foreign key relationship
	desc := "Test batch for update repository"
	batch := &models.Batch{
		Name:        "test-batch",
		Description: &desc,
		Type:        "pilot",
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	repo := &models.Repository{
		FullName: "org/test-repo",
		Status:   string(models.StatusPending),
		Priority: 0,
	}
	db.SaveRepository(ctx, repo)

	t.Run("update batch_id and priority", func(t *testing.T) {
		updates := map[string]interface{}{
			"batch_id": float64(batch.ID),
			"priority": float64(1),
		}
		body, _ := json.Marshal(updates)

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/repositories/org/test-repo", bytes.NewReader(body))
		req.SetPathValue("fullName", "org/test-repo")
		w := httptest.NewRecorder()

		h.UpdateRepository(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var updated models.Repository
		if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if updated.Priority != 1 {
			t.Errorf("Expected priority 1, got %d", updated.Priority)
		}
		if updated.BatchID == nil || *updated.BatchID != batch.ID {
			t.Errorf("Expected batch_id to be %d, got %v", batch.ID, updated.BatchID)
		}
	})

	t.Run("repository not found", func(t *testing.T) {
		updates := map[string]interface{}{"priority": float64(1)}
		body, _ := json.Marshal(updates)

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/repositories/org/nonexistent", bytes.NewReader(body))
		req.SetPathValue("fullName", "org/nonexistent")
		w := httptest.NewRecorder()

		h.UpdateRepository(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/repositories/org/test-repo", bytes.NewReader([]byte("invalid")))
		req.SetPathValue("fullName", "org/test-repo")
		w := httptest.NewRecorder()

		h.UpdateRepository(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestListRepositoriesWithFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create test repositories
	totalSize := int64(1024)
	defaultBranch := testMainBranch

	repos := []struct {
		name   string
		status models.MigrationStatus
	}{
		{"org/pending-repo", models.StatusPending},
		{"org/complete-repo", models.StatusComplete},
		{"org/queued-repo", models.StatusQueuedForMigration},
		{"company/search-me", models.StatusPending},
	}

	for _, r := range repos {
		repo := &models.Repository{
			FullName:      r.name,
			Source:        "ghes",
			SourceURL:     "https://github.com/" + r.name,
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(r.status),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
	}

	t.Run("filter by search", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/repositories?search=search-me", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response struct {
			Repositories []models.Repository `json:"repositories"`
			Total        *int                `json:"total,omitempty"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Repositories) != 1 {
			t.Errorf("Expected 1 repository, got %d", len(response.Repositories))
		}
	})

	t.Run("filter available_for_batch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/repositories?available_for_batch=true", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response struct {
			Repositories []models.Repository `json:"repositories"`
			Total        *int                `json:"total,omitempty"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should include pending repos but exclude complete and queued
		if len(response.Repositories) != 2 {
			t.Errorf("Expected 2 repositories available for batch, got %d", len(response.Repositories))
		}
	})

	t.Run("pagination with limit and offset", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/repositories?limit=2&offset=0", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response struct {
			Repositories []models.Repository `json:"repositories"`
			Total        *int                `json:"total,omitempty"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Repositories) != 2 {
			t.Errorf("Expected 2 repositories, got %d", len(response.Repositories))
		}
	})
}
