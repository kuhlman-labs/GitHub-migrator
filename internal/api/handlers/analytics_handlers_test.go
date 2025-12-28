package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestGetAnalyticsSummary(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create repositories with different statuses
	repos := []*models.Repository{
		{FullName: "org/repo1", Status: string(models.StatusPending)},
		{FullName: "org/repo2", Status: string(models.StatusComplete)},
		{FullName: "org/repo3", Status: string(models.StatusComplete)},
		{FullName: "org/repo4", Status: string(models.StatusMigrationFailed)},
	}
	for _, repo := range repos {
		db.SaveRepository(ctx, repo)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/summary", nil)
	w := httptest.NewRecorder()

	h.GetAnalyticsSummary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["total_repositories"] != float64(4) {
		t.Errorf("Expected 4 total repositories, got %v", response["total_repositories"])
	}
	if response["migrated_count"] != float64(2) {
		t.Errorf("Expected 2 migrated, got %v", response["migrated_count"])
	}
	if response["failed_count"] != float64(1) {
		t.Errorf("Expected 1 failed, got %v", response["failed_count"])
	}
	if response["pending_count"] != float64(1) {
		t.Errorf("Expected 1 pending, got %v", response["pending_count"])
	}
}

func TestGetMigrationProgressHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create repositories
	repos := []*models.Repository{
		{FullName: "org/repo1", Status: string(models.StatusPending)},
		{FullName: "org/repo2", Status: string(models.StatusComplete)},
	}
	for _, repo := range repos {
		db.SaveRepository(ctx, repo)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/progress", nil)
	w := httptest.NewRecorder()

	h.GetMigrationProgress(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["total"] != float64(2) {
		t.Errorf("Expected total 2, got %v", response["total"])
	}
	if response["status_breakdown"] == nil {
		t.Error("Expected status_breakdown in response")
	}
}

func TestGetAnalyticsSummaryWithFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create repositories with different organizations
	repos := []*models.Repository{
		{FullName: "org1/repo1", Status: string(models.StatusComplete)},
		{FullName: "org1/repo2", Status: string(models.StatusPending)},
		{FullName: "org2/repo1", Status: string(models.StatusComplete)},
	}
	for _, repo := range repos {
		db.SaveRepository(ctx, repo)
	}

	t.Run("filter by organization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/summary?organization=org1", nil)
		w := httptest.NewRecorder()

		h.GetAnalyticsSummary(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["total_repositories"] != float64(2) {
			t.Errorf("Expected 2 total repositories for org1, got %v", response["total_repositories"])
		}
	})
}

func TestGetExecutiveReport(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create repositories with different statuses
	repos := []*models.Repository{
		{FullName: "org/repo1", Status: string(models.StatusComplete)},
		{FullName: "org/repo2", Status: string(models.StatusComplete)},
		{FullName: "org/repo3", Status: string(models.StatusPending)},
		{FullName: "org/repo4", Status: string(models.StatusMigrationFailed)},
		{FullName: "org/repo5", Status: string(models.StatusDryRunComplete)},
	}
	for _, repo := range repos {
		db.SaveRepository(ctx, repo)
	}

	t.Run("basic executive report", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/executive", nil)
		w := httptest.NewRecorder()

		h.GetExecutiveReport(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Just verify we get a valid JSON response
		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Log the actual response fields for debugging
		t.Logf("Executive report response keys: %v", getMapKeys(response))
	})

	t.Run("executive report with organization filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/executive?organization=org", nil)
		w := httptest.NewRecorder()

		h.GetExecutiveReport(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestGetMigrationProgressWithBatchFilter(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a batch
	batch := &models.Batch{
		Name:   "Test Batch",
		Type:   "pilot",
		Status: "ready",
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	// Create repositories in the batch
	for i := range 3 {
		status := string(models.StatusPending)
		if i == 0 {
			status = string(models.StatusComplete)
		}
		repo := &models.Repository{
			FullName: "org/batch-repo" + string(rune('0'+i)),
			Status:   status,
			BatchID:  &batch.ID,
		}
		db.SaveRepository(ctx, repo)
	}

	t.Run("filter by batch_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/progress?batch_id=1", nil)
		w := httptest.NewRecorder()

		h.GetMigrationProgress(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})
}
