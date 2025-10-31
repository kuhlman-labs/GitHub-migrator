package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/config"
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
		if r.URL.Path == "/user/memberships/orgs/allowed-org" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member",
			})
		} else if r.URL.Path == "/user/memberships/orgs/forbidden-org" {
			w.WriteHeader(http.StatusNotFound) // User is not a member
		} else {
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
		if r.URL.Path == testTeamMembershipPath {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
			})
		} else if r.URL.Path == "/orgs/test-org/teams/other-team/memberships/testuser" {
			w.WriteHeader(http.StatusNotFound)
		} else {
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
			json.NewEncoder(w).Encode(map[string]string{
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
		if r.URL.Path == "/user/memberships/orgs/test-org" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "member",
			})
		} else if r.URL.Path == testTeamMembershipPath {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
			})
		} else {
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
		if r.URL.Path == "/user/memberships/orgs/test-org" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
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
			json.NewEncoder(w).Encode(map[string]string{
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
