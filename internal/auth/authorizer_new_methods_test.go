package auth

//nolint:goconst // Test files can have repeated strings for clarity

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

func TestAuthorizer_IsOrgAdmin(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  func(w http.ResponseWriter, r *http.Request)
		expectedAdmin bool
		expectError   bool
	}{
		{
			name: "user is org admin",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
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
			expectedAdmin: true,
			expectError:   false,
		},
		{
			name: "user is org member but not admin",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs/test-org" {
					resp := map[string]interface{}{
						"state": "active",
						"role":  "member",
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedAdmin: false,
			expectError:   false,
		},
		{
			name: "user is not org member",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			expectedAdmin: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockResponse))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			cfg := &config.AuthConfig{}
			authorizer := NewAuthorizer(cfg, logger, server.URL)

			isAdmin, err := authorizer.IsOrgAdmin(context.Background(), "testuser", "test-org", "test-token")

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if isAdmin != tt.expectedAdmin {
				t.Errorf("expected admin=%v, got=%v", tt.expectedAdmin, isAdmin)
			}
		})
	}
}

func TestAuthorizer_HasRepoAdminPermission(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  func(w http.ResponseWriter, r *http.Request)
		expectedAdmin bool
		expectError   bool
	}{
		{
			name: "user has admin permission",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repos/test-org/test-repo/collaborators/testuser/permission" {
					resp := map[string]interface{}{
						"permission": "admin",
						"user": map[string]interface{}{
							"login": "testuser",
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedAdmin: true,
			expectError:   false,
		},
		{
			name: "user has write permission but not admin",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/repos/test-org/test-repo/collaborators/testuser/permission" {
					resp := map[string]interface{}{
						"permission": "write",
						"user": map[string]interface{}{
							"login": "testuser",
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedAdmin: false,
			expectError:   false,
		},
		{
			name: "user has no access to repo",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			expectedAdmin: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockResponse))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			cfg := &config.AuthConfig{}
			authorizer := NewAuthorizer(cfg, logger, server.URL)

			hasAdmin, err := authorizer.HasRepoAdminPermission(context.Background(), "testuser", "test-org", "test-repo", "test-token")

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if hasAdmin != tt.expectedAdmin {
				t.Errorf("expected admin=%v, got=%v", tt.expectedAdmin, hasAdmin)
			}
		})
	}
}

func TestAuthorizer_GetUserOrganizations(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse func(w http.ResponseWriter, r *http.Request)
		expectedOrgs []string
		expectError  bool
	}{
		{
			name: "user has multiple orgs",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
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
						{
							"organization": map[string]string{"login": "org3"},
							"state":        "pending", // Should be excluded
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedOrgs: []string{"org1", "org2"},
			expectError:  false,
		},
		{
			name: "user has no orgs",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/user/memberships/orgs" {
					resp := []map[string]interface{}{}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedOrgs: []string{},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockResponse))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			cfg := &config.AuthConfig{}
			authorizer := NewAuthorizer(cfg, logger, server.URL)

			orgs, err := authorizer.GetUserOrganizations(context.Background(), "test-token")

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(orgs) != len(tt.expectedOrgs) {
				t.Errorf("expected %d orgs, got %d", len(tt.expectedOrgs), len(orgs))
			}
			for i, expected := range tt.expectedOrgs {
				if i >= len(orgs) || orgs[i] != expected {
					t.Errorf("expected org[%d]=%s, got=%s", i, expected, orgs[i])
				}
			}
		})
	}
}

func TestAuthorizer_CheckEnterpriseMembership(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   func(w http.ResponseWriter, r *http.Request)
		expectedMember bool
		expectError    bool
	}{
		{
			name: "user is enterprise member",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/graphql" {
					resp := map[string]interface{}{
						"data": map[string]interface{}{
							"enterprise": map[string]interface{}{
								"slug":          "test-enterprise",
								"viewerIsAdmin": false,
							},
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedMember: true,
			expectError:    false,
		},
		{
			name: "user is not enterprise member",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/graphql" {
					resp := map[string]interface{}{
						"errors": []map[string]interface{}{
							{
								"message": "Resource not accessible by integration",
							},
						},
					}
					json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			},
			expectedMember: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockResponse))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			cfg := &config.AuthConfig{}
			authorizer := NewAuthorizer(cfg, logger, server.URL)

			isMember, err := authorizer.CheckEnterpriseMembership(context.Background(), "testuser", "test-enterprise", "test-token")

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if isMember != tt.expectedMember {
				t.Errorf("expected member=%v, got=%v", tt.expectedMember, isMember)
			}
		})
	}
}
