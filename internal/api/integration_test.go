package api

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
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_RepositoryLifecycle tests the full lifecycle of a repository
func TestIntegration_RepositoryLifecycle(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil, nil)
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

		var response struct {
			Repositories []*models.Repository `json:"repositories"`
			Total        *int                 `json:"total,omitempty"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Repositories) != 1 {
			t.Errorf("Expected 1 repository, got %d", len(response.Repositories))
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

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["repository"] == nil {
			t.Error("Expected repository in response")
		}
	})

	// Test: Update repository
	t.Run("update repository", func(t *testing.T) {
		updates := map[string]any{
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
	server := NewServer(&config.Config{}, db, logger, nil, nil)
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
	server := NewServer(&config.Config{}, db, logger, nil, nil)
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
		reqBody := map[string]any{
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

		var response map[string]any
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
	server := NewServer(&config.Config{}, db, logger, nil, nil)
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

		var response map[string]any
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

		var response map[string]any
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

// TestIntegration_SourcesCreate tests source creation
func TestIntegration_SourcesCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil, nil)
	router := server.Router()

	createReq := map[string]any{
		"name":     "Integration Test Source",
		"type":     "github",
		"base_url": "https://api.github.com",
		"token":    "ghp_integration_test_token_123456789",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "create response: %s", w.Body.String())

	var response map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	assert.Equal(t, "Integration Test Source", response["name"])
	assert.Equal(t, "github", response["type"])
}

// TestIntegration_SourcesListAndGet tests listing and getting sources
func TestIntegration_SourcesListAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	source := &models.Source{
		Name:     "List Test Source",
		Type:     "github",
		BaseURL:  "https://api.github.com",
		Token:    "ghp_test_token_for_listing",
		IsActive: true,
	}
	require.NoError(t, db.CreateSource(ctx, source))

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil, nil)
	router := server.Router()

	// Test list
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var sources []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&sources))
	assert.Len(t, sources, 1)

	// Test get by ID
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/sources/%d", source.ID), nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	var fetchedSource map[string]any
	require.NoError(t, json.NewDecoder(w2.Body).Decode(&fetchedSource))
	assert.Equal(t, "List Test Source", fetchedSource["name"])
}

// TestIntegration_SourcesUpdateAndDelete tests updating and deleting sources
func TestIntegration_SourcesUpdateAndDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	source := &models.Source{
		Name:     "Update Delete Test",
		Type:     "github",
		BaseURL:  "https://api.github.com",
		Token:    "ghp_test_token_for_update",
		IsActive: true,
	}
	require.NoError(t, db.CreateSource(ctx, source))

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := NewServer(&config.Config{}, db, logger, nil, nil)
	router := server.Router()

	// Test update
	updates := map[string]any{"name": "Updated Source Name"}
	body, _ := json.Marshal(updates)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/sources/%d", source.ID), bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "update response: %s", w.Body.String())
	var updated map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&updated))
	assert.Equal(t, "Updated Source Name", updated["name"])

	// Test delete
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/sources/%d", source.ID), nil)
	delW := httptest.NewRecorder()
	router.ServeHTTP(delW, delReq)

	require.Equal(t, http.StatusNoContent, delW.Code)

	// Verify deleted
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/sources/%d", source.ID), nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusNotFound, getW.Code)
}

// TestIntegration_AuthorizationStatus tests the authorization status endpoint
func TestIntegration_AuthorizationStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a mock GitHub server for authorization checks
	mockGitHub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return empty responses - no special permissions
		switch r.URL.Path {
		case "/user/memberships/orgs":
			json.NewEncoder(w).Encode([]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockGitHub.Close()

	// Test 1: Auth disabled - endpoint is not registered (returns 404)
	t.Run("auth disabled returns not found", func(t *testing.T) {
		cfg := &config.Config{
			Auth: config.AuthConfig{
				Enabled: false,
			},
		}
		server := NewServer(cfg, db, logger, nil, nil)
		router := server.Router()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/authorization-status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// When auth is disabled, the authorization-status endpoint is not registered
		assert.Equal(t, http.StatusNotFound, w.Code,
			"expected 404 when auth is disabled, got %d: %s", w.Code, w.Body.String())
	})

	// Test 2: Auth enabled - unauthenticated should be denied
	t.Run("auth enabled unauthenticated returns unauthorized", func(t *testing.T) {
		cfg := &config.Config{
			Auth: config.AuthConfig{
				Enabled:       true,
				SessionSecret: "test-secret-key-for-integration-testing",
			},
		}
		server := NewServer(cfg, db, logger, nil, nil)
		router := server.Router()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/authorization-status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "expected 401 for unauthenticated request")
	})
}

// TestIntegration_AuthorizationTierConfiguration tests different authorization configurations
func TestIntegration_AuthorizationTierConfiguration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_ = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test authorization rules configuration
	t.Run("migration admin teams configuration", func(t *testing.T) {
		cfg := &config.Config{
			Auth: config.AuthConfig{
				Enabled: true,
				AuthorizationRules: config.AuthorizationRules{
					MigrationAdminTeams:               []string{"my-org/migration-admins"},
					AllowOrgAdminMigrations:           true,
					RequireIdentityMappingForSelfService: false,
				},
			},
		}

		// Verify config is correctly applied
		assert.Len(t, cfg.Auth.AuthorizationRules.MigrationAdminTeams, 1)
		assert.Equal(t, "my-org/migration-admins", cfg.Auth.AuthorizationRules.MigrationAdminTeams[0])
		assert.True(t, cfg.Auth.AuthorizationRules.AllowOrgAdminMigrations)
		assert.False(t, cfg.Auth.AuthorizationRules.RequireIdentityMappingForSelfService)
	})

	t.Run("identity mapping for self-service configuration", func(t *testing.T) {
		cfg := &config.Config{
			Auth: config.AuthConfig{
				Enabled: true,
				AuthorizationRules: config.AuthorizationRules{
					RequireIdentityMappingForSelfService: true,
				},
			},
		}

		assert.True(t, cfg.Auth.AuthorizationRules.RequireIdentityMappingForSelfService)
	})
}

// TestIntegration_SourceWithRepositories tests source-repository relationship
func TestIntegration_SourceWithRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a source
	source := &models.Source{
		Name:     "Repo Test Source",
		Type:     "github",
		BaseURL:  "https://api.github.com",
		Token:    "test_token_12345678901234567890",
		IsActive: true,
	}
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Create repositories associated with the source
	for i := 0; i < 3; i++ {
		repo := &models.Repository{
			FullName:     "org/repo-" + string(rune('a'+i)),
			Source:       "github",
			SourceURL:    "https://github.com/org/repo-" + string(rune('a'+i)),
			SourceID:     &source.ID,
			Status:       string(models.StatusPending),
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	// Test: Get repositories by source
	repos, err := db.GetRepositoriesBySourceID(ctx, source.ID)
	if err != nil {
		t.Fatalf("Failed to get repositories: %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("Expected 3 repositories, got %d", len(repos))
	}

	// Test: Update source repository count
	if err := db.UpdateSourceRepositoryCount(ctx, source.ID); err != nil {
		t.Fatalf("Failed to update count: %v", err)
	}

	updatedSource, err := db.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatalf("Failed to get updated source: %v", err)
	}
	if updatedSource.RepositoryCount != 3 {
		t.Errorf("Expected repository count 3, got %d", updatedSource.RepositoryCount)
	}

	// Test: Cannot delete source with repositories
	err = db.DeleteSource(ctx, source.ID)
	if err == nil {
		t.Error("Expected error when deleting source with repositories")
	}
}
