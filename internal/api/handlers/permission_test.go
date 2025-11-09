package handlers

//nolint:goconst // Test files can have repeated strings for clarity

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/auth"
	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

func TestHandler_ListRepositories_Filtering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a test database
	dbConfig := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, err := storage.NewDatabase(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Add test repositories
	repos := []*models.Repository{
		{FullName: "org1/repo1", Source: "github", SourceURL: "https://github.com"},
		{FullName: "org1/repo2", Source: "github", SourceURL: "https://github.com"},
		{FullName: "org2/repo3", Source: "github", SourceURL: "https://github.com"},
	}
	for _, repo := range repos {
		if err := db.SaveRepository(context.Background(), repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	tests := []struct {
		name          string
		authEnabled   bool
		contextUser   *auth.GitHubUser
		contextToken  string
		mockGitHub    func(w http.ResponseWriter, r *http.Request)
		expectedCount int
	}{
		{
			name:          "auth disabled - returns all repos",
			authEnabled:   false,
			contextUser:   nil,
			contextToken:  "",
			mockGitHub:    func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
			expectedCount: 3,
		},
		{
			name:         "org admin sees only their org repos",
			authEnabled:  true,
			contextUser:  &auth.GitHubUser{Login: "testuser", ID: 123},
			contextToken: "test-token",
			mockGitHub: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs" {
					resp := []map[string]interface{}{
						{"organization": map[string]string{"login": "org1"}, "state": "active"},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				if r.URL.Path == "/user/memberships/orgs/org1" {
					resp := map[string]interface{}{"state": "active", "role": "admin"}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedCount: 2, // Only org1 repos
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock GitHub server
			server := httptest.NewServer(http.HandlerFunc(tt.mockGitHub))
			defer server.Close()

			// Create handler
			cfg := &config.AuthConfig{Enabled: tt.authEnabled}
			handler := &Handler{
				db:            db,
				logger:        logger,
				authConfig:    cfg,
				sourceBaseURL: server.URL,
			}

			// Create request with context
			req := httptest.NewRequest("GET", "/api/repositories", nil)
			ctx := req.Context()
			if tt.contextUser != nil {
				ctx = context.WithValue(ctx, auth.ContextKeyUser, tt.contextUser)
			}
			if tt.contextToken != "" {
				ctx = context.WithValue(ctx, auth.ContextKeyGitHubToken, tt.contextToken)
			}
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			// Execute request
			handler.ListRepositories(rec, req)

			// Verify response
			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
			}

			var response struct {
				Repositories []*models.Repository `json:"repositories"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(response.Repositories) != tt.expectedCount {
				t.Errorf("expected %d repos, got %d", tt.expectedCount, len(response.Repositories))
			}
		})
	}
}

func TestHandler_HandleRepositoryAction_PermissionCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a test database
	dbConfig := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, err := storage.NewDatabase(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Add test repository
	repo := &models.Repository{
		FullName:  "test-org/test-repo",
		Source:    "github",
		SourceURL: "https://github.com",
		Status:    "pending",
	}
	if err := db.SaveRepository(context.Background(), repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	tests := []struct {
		name           string
		authEnabled    bool
		contextUser    *auth.GitHubUser
		contextToken   string
		mockGitHub     func(w http.ResponseWriter, r *http.Request)
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "auth disabled - allows action",
			authEnabled:    false,
			contextUser:    nil,
			contextToken:   "",
			mockGitHub:     func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
			requestBody:    `{"action": "start", "repo_id": 1}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:         "auth enabled - repo admin can perform action",
			authEnabled:  true,
			contextUser:  &auth.GitHubUser{Login: "testuser", ID: 123},
			contextToken: "test-token",
			mockGitHub: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs/test-org" {
					resp := map[string]interface{}{"state": "active", "role": "admin"}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			requestBody:    `{"action": "start", "repo_id": 1}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:         "auth enabled - non-admin cannot perform action",
			authEnabled:  true,
			contextUser:  &auth.GitHubUser{Login: "testuser", ID: 123},
			contextToken: "test-token",
			mockGitHub: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs/test-org" {
					http.NotFound(w, r)
					return
				}
				if r.URL.Path == "/repos/test-org/test-repo/collaborators/testuser/permission" {
					resp := map[string]interface{}{"permission": "write"}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			requestBody:    `{"action": "start", "repo_id": 1}`,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock GitHub server
			server := httptest.NewServer(http.HandlerFunc(tt.mockGitHub))
			defer server.Close()

			// Create handler
			cfg := &config.AuthConfig{Enabled: tt.authEnabled}
			handler := &Handler{
				db:            db,
				logger:        logger,
				authConfig:    cfg,
				sourceBaseURL: server.URL,
			}

			// Create request with context
			req := httptest.NewRequest("POST", "/api/repositories/action", strings.NewReader(tt.requestBody))
			ctx := req.Context()
			if tt.contextUser != nil {
				ctx = context.WithValue(ctx, auth.ContextKeyUser, tt.contextUser)
			}
			if tt.contextToken != "" {
				ctx = context.WithValue(ctx, auth.ContextKeyGitHubToken, tt.contextToken)
			}
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			// Execute request
			handler.HandleRepositoryAction(rec, req)

			// Verify response
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandler_AddRepositoriesToBatch_PermissionCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a test database
	dbConfig := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}
	db, err := storage.NewDatabase(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Add test repositories
	repo1 := &models.Repository{FullName: "org1/repo1", Source: "github", SourceURL: "https://github.com"}
	repo2 := &models.Repository{FullName: "org2/repo2", Source: "github", SourceURL: "https://github.com"}
	db.SaveRepository(context.Background(), repo1)
	db.SaveRepository(context.Background(), repo2)

	// Create a test batch
	batch := &models.Batch{Name: "Test Batch", Status: "pending"}
	if err := db.CreateBatch(context.Background(), batch); err != nil {
		t.Fatalf("Failed to create batch: %v", err)
	}

	tests := []struct {
		name           string
		authEnabled    bool
		contextUser    *auth.GitHubUser
		contextToken   string
		mockGitHub     func(w http.ResponseWriter, r *http.Request)
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "auth disabled - allows adding repos",
			authEnabled:    false,
			contextUser:    nil,
			contextToken:   "",
			mockGitHub:     func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
			requestBody:    `{"batch_id": 1, "repository_names": ["org1/repo1"]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:         "auth enabled - org admin can add repos",
			authEnabled:  true,
			contextUser:  &auth.GitHubUser{Login: "testuser", ID: 123},
			contextToken: "test-token",
			mockGitHub: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs" {
					resp := []map[string]interface{}{
						{"organization": map[string]string{"login": "org1"}, "state": "active"},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				if r.URL.Path == "/user/memberships/orgs/org1" {
					resp := map[string]interface{}{"state": "active", "role": "admin"}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			requestBody:    `{"batch_id": 1, "repository_names": ["org1/repo1"]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:         "auth enabled - cannot add repos without permission",
			authEnabled:  true,
			contextUser:  &auth.GitHubUser{Login: "testuser", ID: 123},
			contextToken: "test-token",
			mockGitHub: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs" {
					resp := []map[string]interface{}{}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			requestBody:    `{"batch_id": 1, "repository_names": ["org2/repo2"]}`,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock GitHub server
			server := httptest.NewServer(http.HandlerFunc(tt.mockGitHub))
			defer server.Close()

			// Create handler
			cfg := &config.AuthConfig{Enabled: tt.authEnabled}
			handler := &Handler{
				db:            db,
				logger:        logger,
				authConfig:    cfg,
				sourceBaseURL: server.URL,
			}

			// Create request with context
			req := httptest.NewRequest("POST", "/api/batch/repositories", strings.NewReader(tt.requestBody))
			ctx := req.Context()
			if tt.contextUser != nil {
				ctx = context.WithValue(ctx, auth.ContextKeyUser, tt.contextUser)
			}
			if tt.contextToken != "" {
				ctx = context.WithValue(ctx, auth.ContextKeyGitHubToken, tt.contextToken)
			}
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			// Execute request
			handler.AddRepositoriesToBatch(rec, req)

			// Verify response
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
