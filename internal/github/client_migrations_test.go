package github

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// mockRateLimitHandlerMigrations returns a mock rate limit response
func mockRateLimitHandlerMigrations(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"resources": map[string]any{
			"core": map[string]any{
				"limit":     5000,
				"remaining": 4999,
				"reset":     1234567890,
			},
		},
	})
}

func TestStartMigrationWithOptions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandlerMigrations)
	mux.HandleFunc("/api/v3/orgs/test-org/migrations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Parse request body
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify required fields
		if req["lock_repositories"] == nil {
			t.Error("Expected lock_repositories to be set")
		}

		// Return mock migration
		migration := map[string]any{
			"id":                12345,
			"state":             "pending",
			"guid":              "test-migration-guid-123",
			"lock_repositories": true,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(migration)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client, err := NewClient(ClientConfig{
		BaseURL:     server.URL,
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	opts := StartMigrationOptions{
		Repositories:         []string{"test/repo"},
		LockRepositories:     true,
		ExcludeReleases:      false,
		ExcludeAttachments:   false,
		ExcludeMetadata:      false,
		ExcludeGitData:       false,
		ExcludeOwnerProjects: false,
	}

	migration, err := client.StartMigrationWithOptions(context.Background(), "test-org", opts)
	if err != nil {
		t.Fatalf("StartMigrationWithOptions() error = %v", err)
	}

	if migration == nil {
		t.Fatal("StartMigrationWithOptions() returned nil migration")
	}

	if migration.GetID() != 12345 {
		t.Errorf("Expected migration ID 12345, got %d", migration.GetID())
	}
}

func TestStartMigrationOptions(t *testing.T) {
	t.Run("creates options with defaults", func(t *testing.T) {
		opts := &StartMigrationOptions{}

		// All exclusion flags should default to false
		if opts.ExcludeReleases {
			t.Error("Expected ExcludeReleases to be false by default")
		}
		if opts.ExcludeAttachments {
			t.Error("Expected ExcludeAttachments to be false by default")
		}
		if opts.ExcludeMetadata {
			t.Error("Expected ExcludeMetadata to be false by default")
		}
		if opts.ExcludeGitData {
			t.Error("Expected ExcludeGitData to be false by default")
		}
		if opts.ExcludeOwnerProjects {
			t.Error("Expected ExcludeOwnerProjects to be false by default")
		}
	})

	t.Run("creates options with exclusions", func(t *testing.T) {
		opts := &StartMigrationOptions{
			LockRepositories:     true,
			ExcludeReleases:      true,
			ExcludeAttachments:   true,
			ExcludeMetadata:      false,
			ExcludeGitData:       false,
			ExcludeOwnerProjects: true,
		}

		if !opts.LockRepositories {
			t.Error("Expected LockRepositories to be true")
		}
		if !opts.ExcludeReleases {
			t.Error("Expected ExcludeReleases to be true")
		}
		if !opts.ExcludeAttachments {
			t.Error("Expected ExcludeAttachments to be true")
		}
		if opts.ExcludeMetadata {
			t.Error("Expected ExcludeMetadata to be false")
		}
		if !opts.ExcludeOwnerProjects {
			t.Error("Expected ExcludeOwnerProjects to be true")
		}
	})
}

func TestUnlockRepository(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandlerMigrations)
	mux.HandleFunc("/api/v3/orgs/test-org/migrations/12345/repos/test-repo/lock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client, err := NewClient(ClientConfig{
		BaseURL:     server.URL,
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.UnlockRepository(context.Background(), "test-org", "test-repo", 12345)
	if err != nil {
		t.Fatalf("UnlockRepository() error = %v", err)
	}
}

func TestListOrgInstallations(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandlerMigrations)
	mux.HandleFunc("/api/v3/orgs/test-org/installations", func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]any{
			"total_count": 2,
			"installations": []map[string]any{
				{
					"id":       1001,
					"app_id":   101,
					"app_slug": "my-app-one",
				},
				{
					"id":       1002,
					"app_id":   102,
					"app_slug": "my-app-two",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client, err := NewClient(ClientConfig{
		BaseURL:     server.URL,
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	installations, err := client.ListOrgInstallations(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListOrgInstallations() error = %v", err)
	}

	if len(installations) != 2 {
		t.Errorf("Expected 2 installations, got %d", len(installations))
	}

	if installations[0].ID != 1001 {
		t.Errorf("Expected first installation ID 1001, got %d", installations[0].ID)
	}
	if installations[0].AppSlug != "my-app-one" {
		t.Errorf("Expected first app slug 'my-app-one', got %s", installations[0].AppSlug)
	}
}

func TestOrgAppInstallation(t *testing.T) {
	t.Run("creates org app installation correctly", func(t *testing.T) {
		installation := &OrgAppInstallation{
			ID:                  1001,
			AppSlug:             "my-app",
			RepositorySelection: "all",
		}

		if installation.ID != 1001 {
			t.Errorf("Expected ID 1001, got %d", installation.ID)
		}
		if installation.AppSlug != "my-app" {
			t.Errorf("Expected app slug 'my-app', got %s", installation.AppSlug)
		}
		if installation.RepositorySelection != "all" {
			t.Errorf("Expected repository selection 'all', got %s", installation.RepositorySelection)
		}
	})
}

func TestStartMigrationRequest(t *testing.T) {
	t.Run("creates migration request with all fields", func(t *testing.T) {
		lockRepos := true
		excludeReleases := true
		excludeAttachments := false
		excludeMetadata := false
		excludeGitData := false
		excludeOwnerProjects := true

		req := startMigrationRequest{
			LockRepositories:     &lockRepos,
			ExcludeReleases:      &excludeReleases,
			ExcludeAttachments:   &excludeAttachments,
			ExcludeMetadata:      &excludeMetadata,
			ExcludeGitData:       &excludeGitData,
			ExcludeOwnerProjects: &excludeOwnerProjects,
			Repositories:         []string{"test/repo1", "test/repo2"},
		}

		if req.LockRepositories == nil || !*req.LockRepositories {
			t.Error("Expected LockRepositories to be true")
		}
		if len(req.Repositories) != 2 {
			t.Errorf("Expected 2 repositories, got %d", len(req.Repositories))
		}
	})
}
