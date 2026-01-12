package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestStartMigrationHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create test repositories
	repo1 := &models.Repository{FullName: "org/repo1", Status: string(models.StatusPending)}
	repo2 := &models.Repository{FullName: "org/repo2", Status: string(models.StatusPending)}
	_ = db.SaveRepository(ctx, repo1)
	_ = db.SaveRepository(ctx, repo2)

	// Fetch the repos to get their IDs
	savedRepo1, _ := db.GetRepository(ctx, "org/repo1")
	savedRepo2, _ := db.GetRepository(ctx, "org/repo2")

	t.Run("start by repository IDs", func(t *testing.T) {
		reqBody := StartMigrationRequest{
			RepositoryIDs: []int64{savedRepo1.ID, savedRepo2.ID},
			DryRun:        false,
			Priority:      1,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/start", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.StartMigration(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}

		var response StartMigrationResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Count != 2 {
			t.Errorf("Expected count 2, got %d", response.Count)
		}
	})

	t.Run("start by full names", func(t *testing.T) {
		reqBody := StartMigrationRequest{
			FullNames: []string{"org/repo1"},
			DryRun:    true,
			Priority:  0,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/start", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.StartMigration(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}
	})

	t.Run("no repositories provided", func(t *testing.T) {
		reqBody := StartMigrationRequest{}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/start", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.StartMigration(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("repositories not found", func(t *testing.T) {
		reqBody := StartMigrationRequest{
			FullNames: []string{"org/nonexistent"},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/start", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.StartMigration(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestGetMigrationStatusHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	repo := &models.Repository{
		FullName: "org/test-repo",
		Status:   string(models.StatusComplete),
	}
	_ = db.SaveRepository(ctx, repo)

	// Fetch the repo to get its ID
	savedRepo, _ := db.GetRepository(ctx, "org/test-repo")

	t.Run("existing migration", func(t *testing.T) {
		// Use the actual repo ID that was assigned after save
		idStr := fmt.Sprintf("%d", savedRepo.ID)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/"+idStr, nil)
		req.SetPathValue("id", idStr)
		w := httptest.NewRecorder()

		h.GetMigrationStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["repository_id"] != float64(savedRepo.ID) {
			t.Errorf("Expected repository_id %d, got %v", savedRepo.ID, response["repository_id"])
		}
		if response["can_retry"] != false {
			t.Error("Expected can_retry to be false for completed migration")
		}
	})

	t.Run("migration not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/999", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		h.GetMigrationStatus(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("invalid migration ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/invalid", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		h.GetMigrationStatus(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestGetMigrationHistoryHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	repo := &models.Repository{FullName: "org/test-repo", Status: string(models.StatusPending)}
	_ = db.SaveRepository(ctx, repo)

	// Add migration history
	history := &models.MigrationHistory{
		RepositoryID: repo.ID,
		Phase:        "discovery",
		Status:       "complete",
		StartedAt:    time.Now(),
	}
	_, _ = db.CreateMigrationHistory(ctx, history)

	idStr := fmt.Sprintf("%d", repo.ID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/"+idStr+"/history", nil)
	req.SetPathValue("id", idStr)
	w := httptest.NewRecorder()

	h.GetMigrationHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var historyList []*models.MigrationHistory
	if err := json.NewDecoder(w.Body).Decode(&historyList); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(historyList) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(historyList))
	}
}

func TestGetMigrationLogsHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	repo := &models.Repository{FullName: "org/test-repo", Status: string(models.StatusPending)}
	_ = db.SaveRepository(ctx, repo)

	// Add migration logs
	log1 := &models.MigrationLog{
		RepositoryID: repo.ID,
		Level:        "INFO",
		Phase:        "discovery",
		Operation:    "test",
		Message:      "Test log 1",
		Timestamp:    time.Now(),
	}
	log2 := &models.MigrationLog{
		RepositoryID: repo.ID,
		Level:        "ERROR",
		Phase:        "migration",
		Operation:    "test",
		Message:      "Test log 2",
		Timestamp:    time.Now(),
	}
	_ = db.CreateMigrationLog(ctx, log1)
	_ = db.CreateMigrationLog(ctx, log2)

	idStr := fmt.Sprintf("%d", repo.ID)

	t.Run("all logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/"+idStr+"/logs", nil)
		req.SetPathValue("id", idStr)
		w := httptest.NewRecorder()

		h.GetMigrationLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check if logs exists in response
		logsInterface, ok := response["logs"]
		if !ok || logsInterface == nil {
			t.Error("Expected logs in response")
			return
		}

		logs, ok := logsInterface.([]any)
		if !ok {
			t.Errorf("Expected logs to be an array, got %T", logsInterface)
			return
		}

		if len(logs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(logs))
		}
	})

	t.Run("filtered by level", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/"+idStr+"/logs?level=ERROR", nil)
		req.SetPathValue("id", idStr)
		w := httptest.NewRecorder()

		h.GetMigrationLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("with limit and offset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/"+idStr+"/logs?limit=1&offset=1", nil)
		req.SetPathValue("id", idStr)
		w := httptest.NewRecorder()

		h.GetMigrationLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["limit"] != float64(1) {
			t.Errorf("Expected limit 1, got %v", response["limit"])
		}
		if response["offset"] != float64(1) {
			t.Errorf("Expected offset 1, got %v", response["offset"])
		}
	})
}

func TestGetMigrationHistoryListHandler(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create completed and incomplete repositories
	repos := []struct {
		fullName string
		status   string
	}{
		{"test/complete1", string(models.StatusComplete)},
		{"test/complete2", string(models.StatusComplete)},
		{"test/pending", string(models.StatusPending)},
	}

	for _, r := range repos {
		now := time.Now()
		repo := &models.Repository{
			FullName:     r.fullName,
			Source:       "ghes",
			SourceURL:    fmt.Sprintf("https://github.com/%s", r.fullName),
			Status:       r.status,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		if r.status == string(models.StatusComplete) {
			repo.MigratedAt = &now
		}
		if err := db.SaveRepository(context.Background(), repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{}, nil, authConfig, "https://api.github.com", "github")

	req := httptest.NewRequest("GET", "/api/v1/migrations/history", nil)
	w := httptest.NewRecorder()

	h.GetMigrationHistoryList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	migrations, ok := response["migrations"].([]any)
	if !ok {
		t.Fatal("Expected migrations to be an array")
	}

	// Should only return completed migrations
	if len(migrations) != 2 {
		t.Errorf("Expected 2 completed migrations, got %d", len(migrations))
	}
}

func TestExportMigrationHistoryHandler(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Create a completed repository
	now := time.Now()
	repo := &models.Repository{
		FullName:     "test/complete",
		Source:       "ghes",
		SourceURL:    "https://github.com/test/complete",
		Status:       string(models.StatusComplete),
		MigratedAt:   &now,
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(context.Background(), repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{}, nil, authConfig, "https://api.github.com", "github")

	t.Run("CSV export", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/migrations/history/export?format=csv", nil)
		w := httptest.NewRecorder()

		h.ExportMigrationHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/csv" {
			t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
		}

		contentDisposition := w.Header().Get("Content-Disposition")
		if contentDisposition != "attachment; filename=migration_history.csv" {
			t.Errorf("Expected Content-Disposition with filename, got '%s'", contentDisposition)
		}
	})

	t.Run("JSON export", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/migrations/history/export?format=json", nil)
		w := httptest.NewRecorder()

		h.ExportMigrationHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("Invalid format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/migrations/history/export?format=xml", nil)
		w := httptest.NewRecorder()

		h.ExportMigrationHistory(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestSelfServiceMigrationBatchAssignmentHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create a test repository
	totalSize := int64(1024)
	defaultBranch := testMainBranch
	repo := &models.Repository{
		FullName:     "test-org/test-repo",
		Source:       "ghes",
		SourceURL:    "https://github.com/test-org/test-repo",
		Status:       string(models.StatusPending),
		Visibility:   "private",
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.SetTotalSize(&totalSize)
	repo.SetDefaultBranch(&defaultBranch)

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Prepare self-service migration request (dry run = false for production migration)
	reqBody := map[string]any{
		"repositories": []string{"test-org/test-repo"},
		"dry_run":      false,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/self-service/migrate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleSelfServiceMigration(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusAccepted, w.Code, w.Body.String())
	}

	// Parse response to get batch ID
	var response struct {
		BatchID int64 `json:"batch_id"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify repository was assigned to the batch
	updatedRepo, err := db.GetRepository(ctx, "test-org/test-repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if updatedRepo.BatchID == nil {
		t.Errorf("Expected repository to have batch_id set, but it was nil")
	} else if *updatedRepo.BatchID != response.BatchID {
		t.Errorf("Expected repository batch_id to be %d, got %d", response.BatchID, *updatedRepo.BatchID)
	}

	// Verify repository appears when listing by batch_id
	repos, err := db.ListRepositories(ctx, map[string]any{"batch_id": response.BatchID})
	if err != nil {
		t.Fatalf("Failed to list repositories: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repository in batch, got %d", len(repos))
	}

	if len(repos) > 0 && repos[0].FullName != "test-org/test-repo" {
		t.Errorf("Expected repository 'test-org/test-repo', got '%s'", repos[0].FullName)
	}
}
