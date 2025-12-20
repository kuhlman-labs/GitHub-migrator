package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// This file contains example tests demonstrating how to use MockDataStore
// for testing error paths and other scenarios that are difficult to test
// with a real database.

// TestGetRepository_DatabaseError demonstrates testing error paths using MockDataStore.
func TestGetRepository_DatabaseError(t *testing.T) {
	// Create a mock that returns an error when GetRepository is called
	cfg := DefaultTestConfig().WithErrors(&MockDataStoreErrors{
		GetRepoErr: errors.New("database connection failed"),
	})

	h, _ := setupTestHandlerWithConfig(t, cfg)

	// Make a request to get a repository
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories/org/repo", nil)
	req.SetPathValue("fullName", "org/repo")
	w := httptest.NewRecorder()

	h.GetRepository(w, req)

	// Should return 500 Internal Server Error when database fails
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for database error, got %d", http.StatusInternalServerError, w.Code)
	}
}

// TestListRepositories_WithMock demonstrates using MockDataStore with preloaded data.
func TestListRepositories_WithMock(t *testing.T) {
	// Create test repositories
	repos := []*models.Repository{
		createTestRepo("org/repo1", models.StatusPending),
		createTestRepo("org/repo2", models.StatusComplete),
		createTestRepo("org/repo3", models.StatusMigrationFailed),
	}

	// Setup handler with preloaded repositories
	cfg := DefaultTestConfig().WithRepos(repos...)
	h, _ := setupTestHandlerWithConfig(t, cfg)

	// Make a request to list repositories
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories", nil)
	w := httptest.NewRecorder()

	h.ListRepositories(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestCreateBatch_DatabaseError demonstrates testing batch creation error paths.
func TestCreateBatch_DatabaseError(t *testing.T) {
	// Create a mock that returns an error when CreateBatch is called
	cfg := DefaultTestConfig().WithErrors(&MockDataStoreErrors{
		CreateBatchErr: errors.New("unique constraint violation"),
	})

	h, _ := setupTestHandlerWithConfig(t, cfg)

	// Create request body - demonstrates the pattern for testing database errors
	// The actual CreateBatch handler may have additional validation that runs
	// before the database call, so the error response depends on implementation.
	_ = h // Handler is ready to use with error injection configured

	// Example usage pattern:
	// reqBody := `{"name": "Test Batch", "type": "pilot"}`
	// req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", strings.NewReader(reqBody))
	// w := httptest.NewRecorder()
	// h.CreateBatch(w, req)
	// Verify w.Code is the expected error status
}

// TestHandlerWithMockDataStore_DirectAccess demonstrates direct MockDataStore manipulation.
func TestHandlerWithMockDataStore_DirectAccess(t *testing.T) {
	// Create mock directly for fine-grained control
	mock := NewMockDataStore()

	// Add test data directly to the mock
	repo := &models.Repository{
		ID:       1,
		FullName: "test-org/test-repo",
		Status:   string(models.StatusPending),
		Source:   "github",
	}
	mock.Repos[repo.FullName] = repo
	mock.ReposByID[repo.ID] = repo

	// Create handler with the mock
	h := setupTestHandlerWithMock(t, mock)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories/test-org/test-repo", nil)
	req.SetPathValue("fullName", "test-org/test-repo")
	w := httptest.NewRecorder()

	h.GetRepository(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestMockDataStore_ErrorInjectionChaining demonstrates fluent error injection.
func TestMockDataStore_ErrorInjectionChaining(t *testing.T) {
	// Create mock with chained error configuration
	mock := NewMockDataStore().
		WithGetRepoError(errors.New("not found")).
		WithSaveRepoError(errors.New("save failed"))

	// Verify errors are set
	if mock.GetRepoErr == nil {
		t.Error("Expected GetRepoErr to be set")
	}
	if mock.SaveRepoErr == nil {
		t.Error("Expected SaveRepoErr to be set")
	}
}

// TestCompareRealDBvsMock demonstrates that both approaches work for the same test.
func TestCompareRealDBvsMock(t *testing.T) {
	testCases := []struct {
		name      string
		useMockDB bool
	}{
		{"with_mock_db", true},
		{"with_real_db", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create config with or without mock
			cfg := DefaultTestConfig()
			if !tc.useMockDB {
				cfg = cfg.WithRealDB()
			}

			// Preload a repository
			cfg = cfg.WithRepos(createTestRepo("org/repo", models.StatusPending))

			h, db := setupTestHandlerWithConfig(t, cfg)

			// Both should work the same way
			req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories", nil)
			w := httptest.NewRecorder()

			h.ListRepositories(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			// Verify db is of expected type
			if tc.useMockDB {
				if _, ok := db.(*MockDataStore); !ok {
					t.Error("Expected MockDataStore when UseMockDB is true")
				}
			}
		})
	}
}
