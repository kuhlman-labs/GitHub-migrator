package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
)

// setupTestADOHandler creates an ADOHandler for testing.
// Uses the shared MockSourceProvider from test_helpers_test.go.
func setupTestADOHandler(t *testing.T) (*ADOHandler, *Handler) {
	t.Helper()
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	baseHandler := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://dev.azure.com", models.SourceTypeAzureDevOps)

	// Use the shared MockSourceProvider configured for Azure DevOps
	adoProvider := NewMockSourceProvider(source.ProviderAzureDevOps)

	adoHandler := &ADOHandler{
		Handler:      baseHandler,
		adoClient:    nil, // Nil to test error handling when not configured
		adoProvider:  adoProvider,
		adoCollector: nil, // Nil to test error handling when not configured
	}

	return adoHandler, baseHandler
}

func TestStartADODiscovery(t *testing.T) {
	t.Run("ClientNotConfigured_ReturnsServiceUnavailable", testADOClientNotConfigured)
	t.Run("ValidationErrors", testADODiscoveryValidation)
}

// testADOClientNotConfigured tests that the handler returns a 503 Service Unavailable
// error when the ADO client/collector is not configured. This prevents creating stuck
// progress records that block future discovery attempts.
func testADOClientNotConfigured(t *testing.T) {
	adoHandler, _ := setupTestADOHandler(t)

	tests := []struct {
		name    string
		reqBody map[string]any
	}{
		{
			name: "organization discovery without client",
			reqBody: map[string]any{
				"organization": "test-ado-org",
				"workers":      5,
			},
		},
		{
			name: "project discovery without client",
			reqBody: map[string]any{
				"organization": "test-ado-org",
				"projects":     []string{"Project1", "Project2"},
				"workers":      3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/ado/discover", bytes.NewReader(body))
			w := httptest.NewRecorder()

			adoHandler.StartADODiscovery(w, req)

			// Should return 503 Service Unavailable when ADO client is not configured
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status %d (Service Unavailable), got %d", http.StatusServiceUnavailable, w.Code)
			}

			var response map[string]any
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Should include error details
			if response["error"] == nil {
				t.Error("Expected error field in response")
			}
		})
	}
}

func testADODiscoveryValidation(t *testing.T) {
	adoHandler, _ := setupTestADOHandler(t)

	tests := []struct {
		name     string
		reqBody  map[string]any
		rawBody  string
		wantCode int
	}{
		{
			name:     "missing organization",
			reqBody:  map[string]any{},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid json",
			rawBody:  "invalid json",
			wantCode: http.StatusBadRequest,
		},
		{
			name: "empty organization",
			reqBody: map[string]any{
				"organization": "",
			},
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

			req := httptest.NewRequest(http.MethodPost, "/api/v1/ado/discover", bytes.NewReader(body))
			w := httptest.NewRecorder()

			adoHandler.StartADODiscovery(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

func TestADODiscoveryStatus(t *testing.T) {
	adoHandler, baseHandler := setupTestADOHandler(t)
	ctx := context.Background()

	// Add some ADO repositories
	projectName := "TestProject"
	repo1 := &models.Repository{
		FullName:   "test-org/TestProject/repo1",
		Status:     string(models.StatusPending),
		Source:     models.SourceTypeAzureDevOps,
		ADOProject: &projectName,
		ADOIsGit:   true,
	}
	repo2 := &models.Repository{
		FullName:   "test-org/TestProject/repo2",
		Status:     string(models.StatusPending),
		Source:     models.SourceTypeAzureDevOps,
		ADOProject: &projectName,
		ADOIsGit:   false, // TFVC
	}
	if err := baseHandler.db.SaveRepository(ctx, repo1); err != nil {
		t.Fatalf("Failed to save repo1: %v", err)
	}
	if err := baseHandler.db.SaveRepository(ctx, repo2); err != nil {
		t.Fatalf("Failed to save repo2: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ado/discovery/status?organization=test-org", nil)
	w := httptest.NewRecorder()

	adoHandler.ADODiscoveryStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify the response contains expected fields
	totalRepos, ok := response["total_repositories"].(float64)
	if !ok {
		t.Error("Expected total_repositories to be a number")
	}
	if response["organization"] != "test-org" {
		t.Errorf("Expected organization 'test-org', got %v", response["organization"])
	}

	// Check TFVC vs Git counts
	tfvcCount, ok := response["tfvc_repositories"].(float64)
	if !ok {
		t.Error("Expected tfvc_repositories to be a number")
	}

	gitCount, ok := response["git_repositories"].(float64)
	if !ok {
		t.Error("Expected git_repositories to be a number")
	}

	// Verify the counts add up correctly
	if int(tfvcCount)+int(gitCount) != int(totalRepos) {
		t.Errorf("TFVC (%v) + Git (%v) should equal Total (%v)", tfvcCount, gitCount, totalRepos)
	}

	// Verify we have at least one TFVC (the one we created with ADOIsGit=false)
	if int(tfvcCount) == 0 {
		t.Error("Expected at least 1 TFVC repository")
	}
}

func TestListADOProjects(t *testing.T) {
	adoHandler, baseHandler := setupTestADOHandler(t)
	ctx := context.Background()

	// Create some ADO projects
	project1 := &models.ADOProject{
		Organization: "test-org",
		Name:         "Project1",
		State:        "wellFormed",
		Visibility:   "private",
	}
	project2 := &models.ADOProject{
		Organization: "test-org",
		Name:         "Project2",
		State:        "wellFormed",
		Visibility:   "public",
	}

	baseHandler.db.SaveADOProject(ctx, project1)
	baseHandler.db.SaveADOProject(ctx, project2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ado/projects?organization=test-org", nil)
	w := httptest.NewRecorder()

	adoHandler.ListADOProjects(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	projects, ok := response["projects"].([]any)
	if !ok {
		t.Fatal("Expected projects to be an array")
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	total, ok := response["total"].(float64)
	if !ok || int(total) != 2 {
		t.Errorf("Expected total to be 2, got %v", response["total"])
	}
}

func TestSplitADOFullName(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     []string
	}{
		{
			name:     "standard format",
			fullName: "org/project/repo",
			want:     []string{"org", "project", "repo"},
		},
		{
			name:     "single part",
			fullName: "org",
			want:     []string{"org"},
		},
		{
			name:     "two parts",
			fullName: "org/project",
			want:     []string{"org", "project"},
		},
		{
			name:     "repo with slashes",
			fullName: "org/project/repo/with/slashes",
			want:     []string{"org", "project", "repo/with/slashes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitADOFullName(tt.fullName)
			if len(got) != len(tt.want) {
				t.Errorf("splitADOFullName(%q) returned %d parts, want %d", tt.fullName, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitADOFullName(%q)[%d] = %q, want %q", tt.fullName, i, got[i], tt.want[i])
				}
			}
		})
	}
}
