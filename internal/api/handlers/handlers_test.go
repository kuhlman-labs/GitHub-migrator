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

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/source"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

const (
	testMainBranch = "main"
)

// mockSourceProvider is a mock implementation of source.Provider for testing
type mockSourceProvider struct{}

func (m *mockSourceProvider) Type() source.ProviderType {
	return source.ProviderGitHub
}

func (m *mockSourceProvider) Name() string {
	return "Mock Provider"
}

func (m *mockSourceProvider) CloneRepository(ctx context.Context, info source.RepositoryInfo, destPath string, opts source.CloneOptions) error {
	return nil
}

func (m *mockSourceProvider) GetAuthenticatedCloneURL(cloneURL string) (string, error) {
	return cloneURL, nil
}

func (m *mockSourceProvider) ValidateCredentials(ctx context.Context) error {
	return nil
}

func (m *mockSourceProvider) SupportsFeature(feature source.Feature) bool {
	return true
}

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

func setupTestHandler(t *testing.T) (*Handler, *storage.Database) {
	t.Helper()
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewHandler(db, logger, nil, nil, nil)
	return handler, db
}

func TestNewHandler(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("without GitHub clients", func(t *testing.T) {
		h := NewHandler(db, logger, nil, nil, nil)
		if h == nil {
			t.Fatal("Expected handler to be created")
		}
		if h.collector != nil {
			t.Error("Expected collector to be nil when GitHub clients are nil")
		}
	})

	t.Run("with GitHub clients but no source provider", func(t *testing.T) {
		sourceClient := &github.Client{}
		destClient := &github.Client{}
		h := NewHandler(db, logger, sourceClient, destClient, nil)
		if h == nil {
			t.Fatal("Expected handler to be created")
		}
		if h.collector != nil {
			t.Error("Expected collector to be nil when source provider is nil")
		}
	})
}

func TestHealth(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}

	if response["time"] == "" {
		t.Error("Expected time to be set")
	}
}

func TestStartDiscovery(t *testing.T) {
	testStartDiscoveryWithoutClient(t)
	testStartDiscoveryValidation(t)
	testStartDiscoveryOrganization(t)
	testStartDiscoveryEnterprise(t)
}

func testStartDiscoveryWithoutClient(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	h := NewHandler(db, logger, nil, nil, nil)

	reqBody := map[string]interface{}{
		"organization": "test-org",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.StartDiscovery(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func testStartDiscoveryValidation(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewHandler(db, logger, &github.Client{}, nil, nil)

	tests := []struct {
		name     string
		reqBody  map[string]interface{}
		rawBody  string
		wantCode int
	}{
		{
			name:     "missing both",
			reqBody:  map[string]interface{}{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "both provided",
			reqBody: map[string]interface{}{
				"organization":    "test-org",
				"enterprise_slug": "test-enterprise",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid json",
			rawBody:  "invalid json",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.rawBody != "" {
				body = []byte(tt.rawBody)
			} else {
				body, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.StartDiscovery(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

func testStartDiscoveryOrganization(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   "test-token",
		Logger:  logger,
	}
	client, _ := github.NewClient(cfg)
	mockProvider := &mockSourceProvider{}
	h := NewHandler(db, logger, client, nil, mockProvider)

	reqBody := map[string]interface{}{
		"organization": "test-org",
		"workers":      10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.StartDiscovery(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["type"] != "organization" {
		t.Errorf("Expected type 'organization', got %v", response["type"])
	}
	if response["organization"] != "test-org" {
		t.Errorf("Expected organization 'test-org', got %v", response["organization"])
	}
}

func testStartDiscoveryEnterprise(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := github.ClientConfig{
		BaseURL: "https://api.github.com",
		Token:   "test-token",
		Logger:  logger,
	}
	client, _ := github.NewClient(cfg)
	mockProvider := &mockSourceProvider{}
	h := NewHandler(db, logger, client, nil, mockProvider)

	reqBody := map[string]interface{}{
		"enterprise_slug": "test-enterprise",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.StartDiscovery(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["type"] != "enterprise" {
		t.Errorf("Expected type 'enterprise', got %v", response["type"])
	}
	if response["enterprise"] != "test-enterprise" {
		t.Errorf("Expected enterprise 'test-enterprise', got %v", response["enterprise"])
	}
}

func TestDiscoveryStatus(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Add some repositories
	repo1 := &models.Repository{FullName: "org/repo1", Status: string(models.StatusPending)}
	repo2 := &models.Repository{FullName: "org/repo2", Status: string(models.StatusPending)}
	db.SaveRepository(ctx, repo1)
	db.SaveRepository(ctx, repo2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/status", nil)
	w := httptest.NewRecorder()

	h.DiscoveryStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["repositories_found"] != float64(2) {
		t.Errorf("Expected 2 repositories, got %v", response["repositories_found"])
	}
}

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

			var repos []*models.Repository
			if err := json.NewDecoder(w.Body).Decode(&repos); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(repos) != tt.expectedCount {
				t.Errorf("Expected %d repositories, got %d", tt.expectedCount, len(repos))
			}

			if tt.expectedStatus != "" && len(repos) > 0 {
				if repos[0].Status != tt.expectedStatus {
					t.Errorf("Expected status %s, got %s", tt.expectedStatus, repos[0].Status)
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

	repo := &models.Repository{
		FullName: "org/test-repo",
		Status:   string(models.StatusPending),
		Priority: 0,
	}
	db.SaveRepository(ctx, repo)

	t.Run("update batch_id and priority", func(t *testing.T) {
		updates := map[string]interface{}{
			"batch_id": float64(123),
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
		if updated.BatchID == nil || *updated.BatchID != 123 {
			t.Error("Expected batch_id to be 123")
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

func TestListBatches(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create test batches
	batch1 := &models.Batch{Name: "Batch 1", Type: "pilot", Status: "ready", CreatedAt: time.Now()}
	batch2 := &models.Batch{Name: "Batch 2", Type: "wave", Status: "ready", CreatedAt: time.Now()}
	db.CreateBatch(ctx, batch1)
	db.CreateBatch(ctx, batch2)

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
		if created.Status != "ready" {
			t.Errorf("Expected status 'ready', got '%s'", created.Status)
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
}

func TestGetBatch(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	batch := &models.Batch{Name: "Test Batch", Type: "pilot", Status: "ready", CreatedAt: time.Now()}
	db.CreateBatch(ctx, batch)

	t.Run("existing batch", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
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
	db.CreateBatch(ctx, batch)

	// Add repositories to batch
	repo1 := &models.Repository{FullName: "org/repo1", Status: string(models.StatusPending)}
	db.SaveRepository(ctx, repo1)
	batchID := batch.ID
	repo1.BatchID = &batchID
	db.UpdateRepository(ctx, repo1)

	t.Run("successful batch start", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/1/start", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		h.StartBatch(w, req)

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

func TestStartMigration(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create test repositories
	repo1 := &models.Repository{FullName: "org/repo1", Status: string(models.StatusPending)}
	repo2 := &models.Repository{FullName: "org/repo2", Status: string(models.StatusPending)}
	db.SaveRepository(ctx, repo1)
	db.SaveRepository(ctx, repo2)

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

func TestGetMigrationStatus(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	repo := &models.Repository{
		FullName: "org/test-repo",
		Status:   string(models.StatusComplete),
	}
	db.SaveRepository(ctx, repo)

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

		var response map[string]interface{}
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

func TestGetMigrationHistory(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	repo := &models.Repository{FullName: "org/test-repo", Status: string(models.StatusPending)}
	db.SaveRepository(ctx, repo)

	// Add migration history
	history := &models.MigrationHistory{
		RepositoryID: repo.ID,
		Phase:        "discovery",
		Status:       "complete",
		StartedAt:    time.Now(),
	}
	db.CreateMigrationHistory(ctx, history)

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

func TestGetMigrationLogs(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	repo := &models.Repository{FullName: "org/test-repo", Status: string(models.StatusPending)}
	db.SaveRepository(ctx, repo)

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
	db.CreateMigrationLog(ctx, log1)
	db.CreateMigrationLog(ctx, log2)

	idStr := fmt.Sprintf("%d", repo.ID)

	t.Run("all logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/"+idStr+"/logs", nil)
		req.SetPathValue("id", idStr)
		w := httptest.NewRecorder()

		h.GetMigrationLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check if logs exists in response
		logsInterface, ok := response["logs"]
		if !ok || logsInterface == nil {
			t.Error("Expected logs in response")
			return
		}

		logs, ok := logsInterface.([]interface{})
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

		var response map[string]interface{}
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

	var response map[string]interface{}
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

func TestGetMigrationProgress(t *testing.T) {
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

	var response map[string]interface{}
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

func TestCanMigrate(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{string(models.StatusPending), true},             // Can queue pending repos
		{string(models.StatusDryRunQueued), true},        // Can re-queue dry runs
		{string(models.StatusDryRunFailed), true},        // Can retry failed dry runs
		{string(models.StatusDryRunComplete), true},      // Can queue after dry run
		{string(models.StatusPreMigration), false},       // Already in migration process
		{string(models.StatusMigrationFailed), true},     // Can retry failed migrations
		{string(models.StatusComplete), false},           // Already complete
		{string(models.StatusMigratingContent), false},   // Already migrating
		{string(models.StatusQueuedForMigration), false}, // Already queued
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := canMigrate(tt.status)
			if result != tt.expected {
				t.Errorf("canMigrate(%s) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestStartBatchErrors(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create batch without repositories
	batch := &models.Batch{Name: "Empty Batch", Type: "pilot", Status: "ready", CreatedAt: time.Now()}
	db.CreateBatch(ctx, batch)

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

func TestDiscoveryStatusError(t *testing.T) {
	// This would require a database error to trigger the error path
	// Skip for now as it requires mocking
}

func TestListBatchesError(t *testing.T) {
	// This would require a database error to trigger the error path
	// Skip for now as it requires mocking
}

func TestListRepositoriesError(t *testing.T) {
	// This would require a database error to trigger the error path
	// Skip for now as it requires mocking
}

func TestSendJSON(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}

	h.sendJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["test"] != "value" {
		t.Errorf("Expected test=value, got test=%s", response["test"])
	}
}

func TestSendError(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	h.sendError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "test error" {
		t.Errorf("Expected error='test error', got error='%s'", response["error"])
	}
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
		updates := map[string]interface{}{
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

		updates := map[string]interface{}{
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

	t.Run("successful add", func(t *testing.T) {
		reqBody := map[string]interface{}{
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

		var response map[string]interface{}
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

		reqBody := map[string]interface{}{
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

	batch.RepositoryCount = 3
	db.UpdateBatch(ctx, batch)

	t.Run("successful remove", func(t *testing.T) {
		reqBody := map[string]interface{}{
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

		var response map[string]interface{}
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
	var failedRepoIDs []int64
	for i := 0; i < 2; i++ {
		repo := &models.Repository{
			FullName:      fmt.Sprintf("org/failed-repo%d", i),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/failed-repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusMigrationFailed),
			BatchID:       &batch.ID,
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("SaveRepository() error = %v", err)
		}
		saved, _ := db.GetRepository(ctx, repo.FullName)
		failedRepoIDs = append(failedRepoIDs, saved.ID)
	}

	t.Run("retry all failures", func(t *testing.T) {
		reqBody := map[string]interface{}{}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/batches/%d/retry", batch.ID), bytes.NewReader(body))
		req.SetPathValue("id", fmt.Sprintf("%d", batch.ID))
		w := httptest.NewRecorder()

		h.RetryBatchFailures(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}

		var response map[string]interface{}
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
			db.UpdateRepository(ctx, repo)
		}

		reqBody := map[string]interface{}{
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

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if int(response["retried_count"].(float64)) != 1 {
			t.Errorf("Expected 1 repository retried, got %v", response["retried_count"])
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

		var response []models.Repository
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Errorf("Expected 1 repository, got %d", len(response))
		}
	})

	t.Run("filter available_for_batch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/repositories?available_for_batch=true", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []models.Repository
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should include pending repos but exclude complete and queued
		if len(response) != 2 {
			t.Errorf("Expected 2 repositories available for batch, got %d", len(response))
		}
	})

	t.Run("pagination with limit and offset", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/repositories?limit=2&offset=0", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []models.Repository
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Errorf("Expected 2 repositories, got %d", len(response))
		}
	})
}

func TestListOrganizations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test repositories in different orgs
	repos := []struct {
		fullName string
		status   string
	}{
		{"org1/repo1", string(models.StatusPending)},
		{"org1/repo2", string(models.StatusComplete)},
		{"org2/repo1", string(models.StatusPending)},
	}

	for _, r := range repos {
		repo := &models.Repository{
			FullName:     r.fullName,
			Source:       "ghes",
			SourceURL:    fmt.Sprintf("https://github.com/%s", r.fullName),
			Status:       r.status,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.SaveRepository(context.Background(), repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{})

	req := httptest.NewRequest("GET", "/api/v1/organizations", nil)
	w := httptest.NewRecorder()

	h.ListOrganizations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []storage.OrganizationStats
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 organizations, got %d", len(response))
	}
}

func TestGetMigrationHistoryList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

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

	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{})

	req := httptest.NewRequest("GET", "/api/v1/migrations/history", nil)
	w := httptest.NewRecorder()

	h.GetMigrationHistoryList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	migrations, ok := response["migrations"].([]interface{})
	if !ok {
		t.Fatal("Expected migrations to be an array")
	}

	// Should only return completed migrations
	if len(migrations) != 2 {
		t.Errorf("Expected 2 completed migrations, got %d", len(migrations))
	}
}

func TestExportMigrationHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

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

	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{})

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
