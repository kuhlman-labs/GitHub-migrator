package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

func TestPermissionChecker_HasFullAccess_EnterpriseAdmin(t *testing.T) {
	// Mock GitHub GraphQL API for enterprise admin check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/graphql" {
			// Return enterprise admin response
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"enterprise": map[string]interface{}{
						"slug":          "test-enterprise",
						"viewerIsAdmin": true,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			RequireEnterpriseAdmin: true,
			RequireEnterpriseSlug:  "test-enterprise",
		},
	}

	client := &github.Client{}
	checker := NewPermissionChecker(client, cfg, logger, server.URL)

	user := &GitHubUser{Login: "testuser", ID: 123}
	hasAccess, err := checker.HasFullAccess(context.Background(), user, "test-token")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasAccess {
		t.Error("expected enterprise admin to have full access")
	}
}

func TestPermissionChecker_HasFullAccess_PrivilegedTeam(t *testing.T) {
	// Mock GitHub API for team membership check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Correct path: /orgs/{org}/teams/{team-slug}/memberships/{username}
		if r.URL.Path == "/orgs/test-org/teams/admin-team/memberships/testuser" {
			resp := map[string]interface{}{
				"state": "active",
				"role":  "member",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			PrivilegedTeams: []string{"test-org/admin-team"},
		},
	}

	client := &github.Client{}
	checker := NewPermissionChecker(client, cfg, logger, server.URL)

	user := &GitHubUser{Login: "testuser", ID: 123}
	hasAccess, err := checker.HasFullAccess(context.Background(), user, "test-token")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasAccess {
		t.Error("expected privileged team member to have full access")
	}
}

func TestPermissionChecker_HasRepoAccess_OrgAdmin(t *testing.T) {
	// Mock GitHub API for org admin check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/memberships/orgs/test-org" {
			resp := map[string]interface{}{
				"state": "active",
				"role":  "admin",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.AuthConfig{}

	client := &github.Client{}
	checker := NewPermissionChecker(client, cfg, logger, server.URL)

	user := &GitHubUser{Login: "testuser", ID: 123}
	hasAccess, err := checker.HasRepoAccess(context.Background(), user, "test-token", "test-org/test-repo")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasAccess {
		t.Error("expected org admin to have repo access")
	}
}

func TestPermissionChecker_HasRepoAccess_RepoAdmin(t *testing.T) {
	// Mock GitHub API for repo permission check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/memberships/orgs/test-org" {
			// Not an org admin
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/graphql" {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"viewerPermission": "ADMIN",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.AuthConfig{}

	client := &github.Client{}
	checker := NewPermissionChecker(client, cfg, logger, server.URL)

	user := &GitHubUser{Login: "testuser", ID: 123}
	hasAccess, err := checker.HasRepoAccess(context.Background(), user, "test-token", "test-org/test-repo")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !hasAccess {
		t.Error("expected repo admin to have repo access")
	}
}

func TestPermissionChecker_HasRepoAccess_NoPermission(t *testing.T) {
	// Mock GitHub API - user has no admin access
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/memberships/orgs/test-org" {
			// Not an org admin
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/graphql" {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"viewerPermission": "WRITE",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.AuthConfig{}

	client := &github.Client{}
	checker := NewPermissionChecker(client, cfg, logger, server.URL)

	user := &GitHubUser{Login: "testuser", ID: 123}
	hasAccess, err := checker.HasRepoAccess(context.Background(), user, "test-token", "test-org/test-repo")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if hasAccess {
		t.Error("expected user without admin permission to not have access")
	}
}

func TestPermissionChecker_HasRepoAccess_InvalidFormat(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.AuthConfig{}
	client := &github.Client{}
	checker := NewPermissionChecker(client, cfg, logger, "https://api.github.com")

	user := &GitHubUser{Login: "testuser", ID: 123}
	_, err := checker.HasRepoAccess(context.Background(), user, "test-token", "invalid-repo-name")

	if err == nil {
		t.Error("expected error for invalid repo name format")
	}
}

func TestPermissionChecker_ValidateRepositoryAccess(t *testing.T) {
	tests := []struct {
		name          string
		repoFullNames []string
		mockResponses func(w http.ResponseWriter, r *http.Request)
		expectError   bool
	}{
		{
			name:          "org admin can access all org repos",
			repoFullNames: []string{"test-org/repo1", "test-org/repo2"},
			mockResponses: func(w http.ResponseWriter, r *http.Request) {
				// Mock org membership endpoint to return active memberships
				if r.URL.Path == "/user/memberships/orgs" {
					resp := []map[string]interface{}{
						{
							"organization": map[string]string{"login": "test-org"},
							"state":        "active",
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				if r.URL.Path == "/user/memberships/orgs/test-org" {
					resp := map[string]interface{}{
						"state": "active",
						"role":  "admin",
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectError: false,
		},
		{
			name:          "user without access fails validation",
			repoFullNames: []string{"other-org/repo1"},
			mockResponses: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs" {
					resp := []map[string]interface{}{}
					json.NewEncoder(w).Encode(resp)
					return
				}
				if r.URL.Path == "/user/memberships/orgs/other-org" {
					http.NotFound(w, r)
					return
				}
				if r.URL.Path == "/graphql" {
					resp := map[string]interface{}{
						"data": map[string]interface{}{
							"repository": map[string]interface{}{
								"viewerPermission": "READ",
							},
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockResponses))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			cfg := &config.AuthConfig{}
			client := &github.Client{}
			checker := NewPermissionChecker(client, cfg, logger, server.URL)

			user := &GitHubUser{Login: "testuser", ID: 123}
			err := checker.ValidateRepositoryAccess(context.Background(), user, "test-token", tt.repoFullNames)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestPermissionChecker_FilterRepositoriesByAccess(t *testing.T) {
	repo1 := &models.Repository{FullName: "org1/repo1"}
	repo2 := &models.Repository{FullName: "org1/repo2"}
	repo3 := &models.Repository{FullName: "org2/repo3"}

	tests := []struct {
		name          string
		inputRepos    []*models.Repository
		mockResponses func(w http.ResponseWriter, r *http.Request)
		expectedCount int
	}{
		{
			name:       "org admin sees all org repos",
			inputRepos: []*models.Repository{repo1, repo2, repo3},
			mockResponses: func(w http.ResponseWriter, r *http.Request) {
				// Return org1 membership
				if r.URL.Path == "/user/memberships/orgs" {
					resp := []map[string]interface{}{
						{
							"organization": map[string]string{"login": "org1"},
							"state":        "active",
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				if r.URL.Path == "/user/memberships/orgs/org1" {
					resp := map[string]interface{}{
						"state": "active",
						"role":  "admin",
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				// Not admin of org2
				if r.URL.Path == "/user/memberships/orgs/org2" {
					http.NotFound(w, r)
					return
				}
				// No repo-level admin for org2/repo3
				if r.URL.Path == "/graphql" {
					resp := map[string]interface{}{
						"data": map[string]interface{}{
							"repository": map[string]interface{}{
								"viewerPermission": "READ",
							},
						},
					}
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
			server := httptest.NewServer(http.HandlerFunc(tt.mockResponses))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			cfg := &config.AuthConfig{}
			client := &github.Client{}
			checker := NewPermissionChecker(client, cfg, logger, server.URL)

			user := &GitHubUser{Login: "testuser", ID: 123}
			filtered, err := checker.FilterRepositoriesByAccess(context.Background(), user, "test-token", tt.inputRepos)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(filtered) != tt.expectedCount {
				t.Errorf("expected %d repos, got %d", tt.expectedCount, len(filtered))
			}
		})
	}
}

func TestPermissionChecker_GetUserOrganizationsWithAdminRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Request: %s %s\n", r.Method, r.URL.Path)
		if r.URL.Path == "/user/memberships/orgs" {
			resp := []map[string]interface{}{
				{
					"organization": map[string]string{"login": "org1"},
					"state":        "active",
				},
				{
					"organization": map[string]string{"login": "org2"},
					"state":        "active",
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/user/memberships/orgs/org1" {
			resp := map[string]interface{}{
				"state": "active",
				"role":  "admin",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/user/memberships/orgs/org2" {
			resp := map[string]interface{}{
				"state": "active",
				"role":  "member",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.AuthConfig{}
	client := &github.Client{}
	checker := NewPermissionChecker(client, cfg, logger, server.URL)

	user := &GitHubUser{Login: "testuser", ID: 123}
	orgs, err := checker.GetUserOrganizationsWithAdminRole(context.Background(), user, "test-token")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(orgs) != 1 {
		t.Errorf("expected 1 admin org, got %d: %v", len(orgs), orgs)
	}
	if !orgs["org1"] {
		t.Error("expected org1 to be an admin org")
	}
	if orgs["org2"] {
		t.Error("org2 should not be an admin org")
	}
}
