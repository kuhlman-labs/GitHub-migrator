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

// mockRateLimitResponse returns a mock rate limit response
func mockRateLimitResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"resources": map[string]interface{}{
			"core": map[string]interface{}{
				"limit":     5000,
				"remaining": 4999,
				"reset":     1234567890,
			},
		},
	})
}

// TestGetOrganizationRepoCountIntegration requires a real GitHub token
// because it uses GraphQL which is complex to mock
func TestGetOrganizationRepoCountIntegration(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping integration test: GITHUB_TOKEN not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client, err := NewClient(ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       token,
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Use a well-known public org for testing
	count, err := client.GetOrganizationRepoCount(context.Background(), "golang")
	if err != nil {
		t.Fatalf("GetOrganizationRepoCount() error = %v", err)
	}

	// golang org should have at least some repos
	if count <= 0 {
		t.Errorf("Expected positive repo count for golang org, got %d", count)
	}
}

func TestListOrgMembers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rate_limit", func(w http.ResponseWriter, _ *http.Request) {
		mockRateLimitResponse(w)
	})
	mux.HandleFunc("/api/v3/orgs/test-org/members", func(w http.ResponseWriter, _ *http.Request) {
		members := []map[string]interface{}{
			{
				"login":      "user1",
				"id":         1001,
				"avatar_url": "https://github.com/avatars/user1.png",
				"html_url":   "https://github.com/user1",
			},
			{
				"login":      "user2",
				"id":         1002,
				"avatar_url": "https://github.com/avatars/user2.png",
				"html_url":   "https://github.com/user2",
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

	members, err := client.ListOrgMembers(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListOrgMembers() error = %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	if members[0].Login != "user1" {
		t.Errorf("Expected first member 'user1', got %s", members[0].Login)
	}
}

// TestGetUserByLoginIntegration requires a real GitHub token
// because it uses GraphQL which is complex to mock
func TestGetUserByLoginIntegration(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping integration test: GITHUB_TOKEN not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client, err := NewClient(ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       token,
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Use a well-known public user for testing
	user, err := client.GetUserByLogin(context.Background(), "octocat")
	if err != nil {
		t.Fatalf("GetUserByLogin() error = %v", err)
	}

	if user == nil {
		t.Fatal("GetUserByLogin() returned nil user")
		return // Explicitly unreachable, but satisfies static analysis
	}

	if user.Login != "octocat" {
		t.Errorf("Expected login 'octocat', got %s", user.Login)
	}
}

// TestUserInfo tests the UserInfo struct
func TestUserInfo(t *testing.T) {
	t.Run("creates user info correctly", func(t *testing.T) {
		user := &UserInfo{
			ID:        "MDQ6VXNlcjEyMzQ1",
			Login:     "testuser",
			Name:      "Test User",
			Email:     "test@example.com",
			AvatarURL: "https://github.com/avatars/testuser.png",
		}

		if user.Login != "testuser" {
			t.Errorf("Expected login 'testuser', got %s", user.Login)
		}
		if user.Name != "Test User" {
			t.Errorf("Expected name 'Test User', got %s", user.Name)
		}
	})
}

func TestEnterpriseOrgInfo(t *testing.T) {
	t.Run("creates enterprise org info correctly", func(t *testing.T) {
		info := &EnterpriseOrgInfo{
			Login:     "my-org",
			RepoCount: 100,
		}

		if info.Login != "my-org" {
			t.Errorf("Expected login 'my-org', got %s", info.Login)
		}
		if info.RepoCount != 100 {
			t.Errorf("Expected repo count 100, got %d", info.RepoCount)
		}
	})
}

func TestOrgMember(t *testing.T) {
	t.Run("creates org member correctly", func(t *testing.T) {
		member := &OrgMember{
			Login:     "user1",
			ID:        12345,
			AvatarURL: "https://github.com/avatars/user1.png",
			Role:      "admin",
		}

		if member.Login != "user1" {
			t.Errorf("Expected login 'user1', got %s", member.Login)
		}
		if member.ID != 12345 {
			t.Errorf("Expected ID 12345, got %d", member.ID)
		}
		if member.Role != "admin" {
			t.Errorf("Expected role 'admin', got %s", member.Role)
		}
	})
}

func TestMannequin(t *testing.T) {
	t.Run("creates mannequin correctly", func(t *testing.T) {
		mannequin := &Mannequin{
			ID:    "MDExOk1hbm5lcXVpbjEyMzQ1",
			Login: "ghost-user",
			Email: "ghost@example.com",
		}

		if mannequin.ID != "MDExOk1hbm5lcXVpbjEyMzQ1" {
			t.Errorf("Expected ID 'MDExOk1hbm5lcXVpbjEyMzQ1', got %s", mannequin.ID)
		}
		if mannequin.Login != "ghost-user" {
			t.Errorf("Expected login 'ghost-user', got %s", mannequin.Login)
		}
	})
}

func TestOrganizationProject(t *testing.T) {
	t.Run("creates organization project correctly", func(t *testing.T) {
		project := &OrganizationProject{
			Title:        "My Project",
			Repositories: []string{"repo1", "repo2"},
		}

		if project.Title != "My Project" {
			t.Errorf("Expected title 'My Project', got %s", project.Title)
		}
		if len(project.Repositories) != 2 {
			t.Errorf("Expected 2 repositories, got %d", len(project.Repositories))
		}
	})
}

func TestAttributionInvitationResult(t *testing.T) {
	t.Run("creates attribution invitation result correctly", func(t *testing.T) {
		result := &AttributionInvitationResult{
			Success:         true,
			MannequinID:     "mannequin123",
			MannequinLogin:  "ghost-user",
			TargetUserID:    "target456",
			TargetUserLogin: "real-user",
		}

		if !result.Success {
			t.Error("Expected success to be true")
		}
		if result.MannequinID != "mannequin123" {
			t.Errorf("Expected mannequin ID 'mannequin123', got %s", result.MannequinID)
		}
		if result.TargetUserLogin != "real-user" {
			t.Errorf("Expected target user login 'real-user', got %s", result.TargetUserLogin)
		}
	})
}
