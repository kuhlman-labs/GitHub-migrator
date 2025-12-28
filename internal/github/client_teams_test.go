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

// mockRateLimitHandler returns a handler that responds to rate limit checks
func mockRateLimitHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"resources": map[string]any{
			"core": map[string]any{
				"limit":     5000,
				"remaining": 4999,
				"reset":     1234567890,
			},
		},
	})
}

func TestListOrganizationTeams(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandler)
	mux.HandleFunc("/api/v3/orgs/test-org/teams", func(w http.ResponseWriter, _ *http.Request) {
		teams := []map[string]any{
			{
				"id":          123,
				"slug":        "team-one",
				"name":        "Team One",
				"description": "First team",
				"privacy":     "closed",
			},
			{
				"id":          456,
				"slug":        "team-two",
				"name":        "Team Two",
				"description": "Second team",
				"privacy":     "secret",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(teams)
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

	teams, err := client.ListOrganizationTeams(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListOrganizationTeams() error = %v", err)
	}

	if len(teams) != 2 {
		t.Errorf("Expected 2 teams, got %d", len(teams))
	}

	if teams[0].Slug != "team-one" {
		t.Errorf("Expected first team slug 'team-one', got %s", teams[0].Slug)
	}
	if teams[0].Name != "Team One" {
		t.Errorf("Expected first team name 'Team One', got %s", teams[0].Name)
	}
}

func TestListTeamRepositories(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandler)
	mux.HandleFunc("/api/v3/orgs/test-org/teams/team-one/repos", func(w http.ResponseWriter, _ *http.Request) {
		repos := []map[string]any{
			{
				"id":        1,
				"full_name": "test-org/repo-one",
				"permissions": map[string]bool{
					"admin":    true,
					"push":     true,
					"pull":     true,
					"maintain": false,
					"triage":   false,
				},
			},
			{
				"id":        2,
				"full_name": "test-org/repo-two",
				"permissions": map[string]bool{
					"admin":    false,
					"push":     true,
					"pull":     true,
					"maintain": false,
					"triage":   false,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(repos)
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

	repos, err := client.ListTeamRepositories(context.Background(), "test-org", "team-one")
	if err != nil {
		t.Fatalf("ListTeamRepositories() error = %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(repos))
	}

	if repos[0].FullName != "test-org/repo-one" {
		t.Errorf("Expected first repo 'test-org/repo-one', got %s", repos[0].FullName)
	}
	if repos[0].Permission != "admin" {
		t.Errorf("Expected permission 'admin', got %s", repos[0].Permission)
	}
}

func TestGetTeamBySlug(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandler)
	mux.HandleFunc("/api/v3/orgs/test-org/teams/team-one", func(w http.ResponseWriter, _ *http.Request) {
		team := map[string]any{
			"id":          123,
			"slug":        "team-one",
			"name":        "Team One",
			"description": "First team",
			"privacy":     "closed",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(team)
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

	team, err := client.GetTeamBySlug(context.Background(), "test-org", "team-one")
	if err != nil {
		t.Fatalf("GetTeamBySlug() error = %v", err)
	}

	if team == nil {
		t.Fatal("GetTeamBySlug() returned nil team")
		return // Explicitly unreachable, but satisfies static analysis
	}

	if team.Slug != "team-one" {
		t.Errorf("Expected slug 'team-one', got %s", team.Slug)
	}
}

func TestGetTeamBySlug_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandler)
	mux.HandleFunc("/api/v3/orgs/test-org/teams/non-existent", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
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

	team, err := client.GetTeamBySlug(context.Background(), "test-org", "non-existent")
	if err != nil {
		t.Fatalf("GetTeamBySlug() error = %v, expected nil error for not found", err)
	}

	if team != nil {
		t.Errorf("Expected nil team for not found, got %v", team)
	}
}

func TestListTeamMembers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", mockRateLimitHandler)
	mux.HandleFunc("/api/v3/orgs/test-org/teams/team-one/members", func(w http.ResponseWriter, _ *http.Request) {
		members := []map[string]any{
			{
				"login": "user1",
				"id":    1001,
			},
			{
				"login": "user2",
				"id":    1002,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(members)
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

	members, err := client.ListTeamMembers(context.Background(), "test-org", "team-one")
	if err != nil {
		t.Fatalf("ListTeamMembers() error = %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	if members[0].Login != "user1" {
		t.Errorf("Expected first member 'user1', got %s", members[0].Login)
	}
}

func TestTeamInfo(t *testing.T) {
	t.Run("creates team info correctly", func(t *testing.T) {
		info := &TeamInfo{
			ID:          123,
			Slug:        "my-team",
			Name:        "My Team",
			Description: "A test team",
			Privacy:     "closed",
		}

		if info.ID != 123 {
			t.Errorf("Expected ID 123, got %d", info.ID)
		}
		if info.Slug != "my-team" {
			t.Errorf("Expected slug 'my-team', got %s", info.Slug)
		}
	})
}

func TestTeamRepository(t *testing.T) {
	t.Run("creates team repository correctly", func(t *testing.T) {
		repo := &TeamRepository{
			FullName:   "org/repo",
			Permission: "admin",
		}

		if repo.FullName != "org/repo" {
			t.Errorf("Expected full name 'org/repo', got %s", repo.FullName)
		}
		if repo.Permission != "admin" {
			t.Errorf("Expected permission 'admin', got %s", repo.Permission)
		}
	})
}

func TestTeamMember(t *testing.T) {
	t.Run("creates team member correctly", func(t *testing.T) {
		member := &TeamMember{
			Login: "user1",
			Role:  "maintainer",
		}

		if member.Login != "user1" {
			t.Errorf("Expected login 'user1', got %s", member.Login)
		}
		if member.Role != "maintainer" {
			t.Errorf("Expected role 'maintainer', got %s", member.Role)
		}
	})
}
