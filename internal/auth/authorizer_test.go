package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

const testTeamMembershipPath = "/orgs/test-org/teams/admin-team/memberships/testuser"

func TestNewAuthorizer(t *testing.T) {
	cfg := &config.AuthConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name    string
		baseURL string
	}{
		{
			name:    "github.com",
			baseURL: "https://api.github.com",
		},
		{
			name:    "github enterprise",
			baseURL: "https://github.example.com/api/v3",
		},
		{
			name:    "empty base URL",
			baseURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := NewAuthorizer(cfg, logger, tt.baseURL)
			if authorizer == nil {
				t.Error("Expected non-nil authorizer")
				return
			}
			if authorizer.baseURL == "" {
				t.Error("Base URL should not be empty")
			}
		})
	}
}

func TestAuthorizeNoRules(t *testing.T) {
	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			// No rules configured
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, "https://api.github.com")

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	result, err := authorizer.Authorize(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Authorized {
		t.Error("User should be authorized when no rules are configured")
	}
}

func TestCheckOrganizationMembership(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Parse the URL to determine which org is being checked
		// New endpoint: /user/memberships/orgs/{org}
		switch r.URL.Path {
		case "/user/memberships/orgs/allowed-org":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member",
			})
		case "/user/memberships/orgs/forbidden-org":
			w.WriteHeader(http.StatusNotFound) // User is not a member
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	ctx := context.Background()

	tests := []struct {
		name     string
		orgs     []string
		expected bool
	}{
		{
			name:     "member of allowed org",
			orgs:     []string{"allowed-org"},
			expected: true,
		},
		{
			name:     "not member of forbidden org",
			orgs:     []string{"forbidden-org"},
			expected: false,
		},
		{
			name:     "member of one of multiple orgs",
			orgs:     []string{"forbidden-org", "allowed-org"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorized, err := authorizer.CheckOrganizationMembership(ctx, "testuser", tt.orgs, "test-token")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if authorized != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, authorized)
			}
		})
	}
}

func TestCheckTeamMembership(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Parse the URL to determine which team is being checked
		switch r.URL.Path {
		case testTeamMembershipPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
			})
		case "/orgs/test-org/teams/other-team/memberships/testuser":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	ctx := context.Background()

	tests := []struct {
		name     string
		teams    []string
		expected bool
	}{
		{
			name:     "member of admin team",
			teams:    []string{"test-org/admin-team"},
			expected: true,
		},
		{
			name:     "not member of other team",
			teams:    []string{"test-org/other-team"},
			expected: false,
		},
		{
			name:     "member of one of multiple teams",
			teams:    []string{"test-org/other-team", "test-org/admin-team"},
			expected: true,
		},
		{
			name:     "invalid team format",
			teams:    []string{"invalid-format"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorized, err := authorizer.CheckTeamMembership(ctx, "testuser", tt.teams, "test-token")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if authorized != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, authorized)
			}
		})
	}
}

func TestAuthorizeWithOrgMembership(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/memberships/orgs/allowed-org" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			RequireOrgMembership: []string{"allowed-org"},
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	result, err := authorizer.Authorize(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Authorized {
		t.Errorf("User should be authorized. Reason: %s", result.Reason)
	}
}

func TestAuthorizeWithOrgMembershipDenied(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound) // User is not a member
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			RequireOrgMembership: []string{"required-org"},
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	result, err := authorizer.Authorize(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Authorized {
		t.Error("User should not be authorized")
	}
	if result.Reason == "" {
		t.Error("Expected reason for denial")
	}
}

func TestAuthorizeWithMultipleRules(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testOrgMembershipPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member",
			})
		case testTeamMembershipPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			RequireOrgMembership:  []string{"test-org"},
			RequireTeamMembership: []string{"test-org/admin-team"},
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	result, err := authorizer.Authorize(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Authorized {
		t.Errorf("User should be authorized. Reason: %s", result.Reason)
	}
}

func TestIsOrgMember(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testOrgMembershipPath {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	ctx := context.Background()

	// Test member
	isMember, err := authorizer.isOrgMember(ctx, "testuser", "test-org", "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !isMember {
		t.Error("Expected user to be org member")
	}

	// Test non-member
	isMember, err = authorizer.isOrgMember(ctx, "testuser", "other-org", "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isMember {
		t.Error("Expected user to not be org member")
	}
}

func TestIsTeamMember(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testTeamMembershipPath {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	ctx := context.Background()

	// Test member
	isMember, err := authorizer.isTeamMember(ctx, "testuser", "test-org", "admin-team", "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !isMember {
		t.Error("Expected user to be team member")
	}

	// Test non-member
	isMember, err = authorizer.isTeamMember(ctx, "testuser", "test-org", "other-team", "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isMember {
		t.Error("Expected user to not be team member")
	}
}

// Tests for destination-centric authorization tiers

func TestGetUserAuthorizationTier_EnterpriseAdmin(t *testing.T) {
	// Create mock GitHub API server that returns enterprise admin status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testGraphQLPath {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": true,
					},
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			RequireEnterpriseSlug:          "test-enterprise",
			AllowEnterpriseAdminMigrations: true,
			EnableSelfService:              true,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	tierInfo, err := authorizer.GetUserAuthorizationTier(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tierInfo.Tier != TierAdmin {
		t.Errorf("Expected TierAdmin, got %s", tierInfo.Tier)
	}
	if tierInfo.TierName != "Full Migration Rights" {
		t.Errorf("Expected 'Full Migration Rights', got %s", tierInfo.TierName)
	}
	if !tierInfo.Permissions.CanMigrateAllRepos {
		t.Error("Expected CanMigrateAllRepos to be true")
	}
	if !tierInfo.Permissions.CanManageSources {
		t.Error("Expected CanManageSources to be true")
	}
}

func TestGetUserAuthorizationTier_MigrationTeamMember(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/myorg/teams/migration-admins/memberships/testuser":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
			})
		case testGraphQLPath:
			// Enterprise admin check - user is not admin
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": false,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			RequireEnterpriseSlug:          "test-enterprise",
			AllowEnterpriseAdminMigrations: true,
			MigrationAdminTeams:            []string{"myorg/migration-admins"},
			EnableSelfService:              true,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	tierInfo, err := authorizer.GetUserAuthorizationTier(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tierInfo.Tier != TierAdmin {
		t.Errorf("Expected TierAdmin, got %s", tierInfo.Tier)
	}
	if tierInfo.Reason != "Migration admin team member" {
		t.Errorf("Expected reason 'Migration admin team member', got %s", tierInfo.Reason)
	}
}

func TestGetUserAuthorizationTier_OrgAdmin(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testUserMembershipsOrgPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"organization": map[string]string{"login": "test-org"},
					"state":        "active",
				},
			})
		case testOrgMembershipPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "admin",
			})
		case testGraphQLPath:
			// Enterprise admin check - user is not admin
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": false,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			RequireEnterpriseSlug:          "test-enterprise",
			AllowEnterpriseAdminMigrations: true,
			AllowOrgAdminMigrations:        true,
			EnableSelfService:              true,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	tierInfo, err := authorizer.GetUserAuthorizationTier(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tierInfo.Tier != TierAdmin {
		t.Errorf("Expected TierAdmin, got %s", tierInfo.Tier)
	}
	if tierInfo.Reason != "Organization administrator (test-org)" {
		t.Errorf("Expected reason containing 'Organization administrator', got %s", tierInfo.Reason)
	}
}

func TestGetUserAuthorizationTier_SelfServiceDisabled(t *testing.T) {
	// When EnableSelfService is false, self-service is DISABLED
	// and users fall to read-only tier. Only admins can migrate.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testUserMembershipsOrgPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"organization": map[string]string{"login": "test-org"},
					"state":        "active",
				},
			})
		case testOrgMembershipPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member", // Not admin
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			AllowOrgAdminMigrations: true,
			EnableSelfService:       false, // Self-service DISABLED
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	tierInfo, err := authorizer.GetUserAuthorizationTier(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// When self-service is disabled, users should be read-only
	if tierInfo.Tier != TierReadOnly {
		t.Errorf("Expected TierReadOnly when self-service disabled, got %s", tierInfo.Tier)
	}
	if tierInfo.TierName != "Read-Only" {
		t.Errorf("Expected 'Read-Only', got %s", tierInfo.TierName)
	}
	if tierInfo.Permissions.CanMigrateOwnRepos {
		t.Error("Expected CanMigrateOwnRepos to be false when self-service disabled")
	}
	if tierInfo.Permissions.CanMigrateAllRepos {
		t.Error("Expected CanMigrateAllRepos to be false")
	}
}

func TestGetUserAuthorizationTier_ReadOnly(t *testing.T) {
	// Create mock GitHub API server - user has no admin privileges
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testUserMembershipsOrgPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"organization": map[string]string{"login": "test-org"},
					"state":        "active",
				},
			})
		case testOrgMembershipPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member", // Not admin
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			AllowOrgAdminMigrations: true,
			EnableSelfService:       true, // Requires identity mapping
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	tierInfo, err := authorizer.GetUserAuthorizationTier(context.Background(), user, "test-token")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tierInfo.Tier != TierReadOnly {
		t.Errorf("Expected TierReadOnly, got %s", tierInfo.Tier)
	}
	if tierInfo.TierName != "Read-Only" {
		t.Errorf("Expected 'Read-Only', got %s", tierInfo.TierName)
	}
	if tierInfo.Permissions.CanMigrateOwnRepos {
		t.Error("Expected CanMigrateOwnRepos to be false")
	}
	if tierInfo.Permissions.CanMigrateAllRepos {
		t.Error("Expected CanMigrateAllRepos to be false")
	}
	if !tierInfo.Permissions.CanViewRepos {
		t.Error("Expected CanViewRepos to be true")
	}
}

func TestCheckDestinationMigrationRights_AllTiers(t *testing.T) {
	tests := []struct {
		name           string
		serverHandler  func(w http.ResponseWriter, r *http.Request)
		cfg            *config.AuthConfig
		expectedAccess bool
	}{
		{
			name: "enterprise admin has access",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == testGraphQLPath {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{
						"data": map[string]any{
							"enterprise": map[string]any{
								"slug":          "test-enterprise",
								"viewerIsAdmin": true,
							},
						},
					})
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			cfg: &config.AuthConfig{
				AuthorizationRules: config.AuthorizationRules{
					RequireEnterpriseSlug:          "test-enterprise",
					AllowEnterpriseAdminMigrations: true,
				},
			},
			expectedAccess: true,
		},
		{
			name: "migration team member has access",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/orgs/myorg/teams/migrators/memberships/testuser" {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]string{"state": "active"})
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			cfg: &config.AuthConfig{
				AuthorizationRules: config.AuthorizationRules{
					MigrationAdminTeams: []string{"myorg/migrators"},
				},
			},
			expectedAccess: true,
		},
		{
			name: "regular user has no full access",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == testUserMembershipsOrgPath {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode([]map[string]any{})
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			cfg: &config.AuthConfig{
				AuthorizationRules: config.AuthorizationRules{
					AllowOrgAdminMigrations: true,
					EnableSelfService:       true,
				},
			},
			expectedAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			authorizer := NewAuthorizer(tt.cfg, logger, server.URL)

			user := &GitHubUser{
				ID:    12345,
				Login: "testuser",
			}

			hasAccess, _, err := authorizer.CheckDestinationMigrationRights(context.Background(), user, "test-token")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if hasAccess != tt.expectedAccess {
				t.Errorf("Expected access=%v, got %v", tt.expectedAccess, hasAccess)
			}
		})
	}
}
