package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// TestIntegration_RepositoryLifecycle tests the full lifecycle of a repository
func TestIntegration_RepositoryLifecycle(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil)
	router := server.Router()

	// Test: Create repository
	repo := &models.Repository{
		FullName:     "test-org/integration-repo",
		Source:       "ghes",
		SourceURL:    "https://github.test.com/test-org/integration-repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(context.Background(), repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Test: List repositories
	t.Run("list repositories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var repos []*models.Repository
		if err := json.NewDecoder(w.Body).Decode(&repos); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(repos) != 1 {
			t.Errorf("Expected 1 repository, got %d", len(repos))
		}
	})

	// Test: Get single repository
	t.Run("get repository", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories/test-org%2Fintegration-repo", nil)
		req.SetPathValue("fullName", "test-org/integration-repo")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

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
	})

	// Test: Update repository
	t.Run("update repository", func(t *testing.T) {
		updates := map[string]interface{}{
			"priority": 1,
		}
		body, _ := json.Marshal(updates)

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/repositories/test-org%2Fintegration-repo", bytes.NewReader(body))
		req.SetPathValue("fullName", "test-org/integration-repo")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// TestIntegration_BatchWorkflow tests the batch creation and execution workflow
func TestIntegration_BatchWorkflow(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil)
	router := server.Router()

	ctx := context.Background()

	// Create test repositories
	repos := []*models.Repository{
		{
			FullName:     "test-org/repo1",
			Source:       "ghes",
			SourceURL:    "https://github.test.com/test-org/repo1",
			Status:       string(models.StatusPending),
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			FullName:     "test-org/repo2",
			Source:       "ghes",
			SourceURL:    "https://github.test.com/test-org/repo2",
			Status:       string(models.StatusPending),
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
	for _, r := range repos {
		if err := db.SaveRepository(ctx, r); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	// Test: Create batch
	var batchID int64
	t.Run("create batch", func(t *testing.T) {
		desc := "Integration test batch"
		batch := models.Batch{
			Name:        "Test Batch",
			Description: &desc,
			Type:        "pilot",
		}
		body, _ := json.Marshal(batch)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}

		var created models.Batch
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		batchID = created.ID
		if batchID == 0 {
			t.Error("Expected non-zero batch ID")
		}
	})

	// Test: List batches
	t.Run("list batches", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var batches []*models.Batch
		if err := json.NewDecoder(w.Body).Decode(&batches); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(batches) != 1 {
			t.Errorf("Expected 1 batch, got %d", len(batches))
		}
	})
}

// TestIntegration_MigrationStartWorkflow tests starting a migration
func TestIntegration_MigrationStartWorkflow(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil)
	router := server.Router()

	ctx := context.Background()

	// Create test repository
	repo := &models.Repository{
		FullName:     "test-org/migration-repo",
		Source:       "ghes",
		SourceURL:    "https://github.test.com/test-org/migration-repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Test: Start migration by full name
	t.Run("start migration", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"full_names": []string{"test-org/migration-repo"},
			"dry_run":    true,
			"priority":   1,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/start", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["count"] != float64(1) {
			t.Errorf("Expected count 1, got %v", response["count"])
		}
	})
}

// TestIntegration_Analytics tests the analytics endpoints
func TestIntegration_Analytics(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil)
	router := server.Router()

	ctx := context.Background()

	// Create test repositories with different statuses
	repos := []*models.Repository{
		{
			FullName:     "test-org/pending-repo",
			Source:       "ghes",
			SourceURL:    "https://github.test.com/test-org/pending-repo",
			Status:       string(models.StatusPending),
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			FullName:     "test-org/completed-repo",
			Source:       "ghes",
			SourceURL:    "https://github.test.com/test-org/completed-repo",
			Status:       string(models.StatusComplete),
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
	for _, r := range repos {
		if err := db.SaveRepository(ctx, r); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	// Test: Get analytics summary
	t.Run("analytics summary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["total_repositories"] != float64(2) {
			t.Errorf("Expected 2 total repositories, got %v", response["total_repositories"])
		}
		if response["migrated_count"] != float64(1) {
			t.Errorf("Expected 1 migrated, got %v", response["migrated_count"])
		}
		if response["pending_count"] != float64(1) {
			t.Errorf("Expected 1 pending, got %v", response["pending_count"])
		}
	})

	// Test: Get migration progress
	t.Run("migration progress", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/progress", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["total"] != float64(2) {
			t.Errorf("Expected total 2, got %v", response["total"])
		}
	})
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *storage.Database {
	t.Helper()
	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, err := storage.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	return db
}
