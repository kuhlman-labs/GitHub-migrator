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

func TestRequireAdmin_AuthDisabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	// Auth disabled - should allow all requests through
	m := NewMiddleware(jwtManager, nil, logger, false)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAdmin(testHandler)

	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !handlerCalled {
		t.Error("expected handler to be called when auth is disabled")
	}
}

func TestRequireAdmin_NoUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	m := NewMiddleware(jwtManager, nil, logger, true)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without user in context")
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAdmin(testHandler)

	// Request without user in context
	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAdmin_NoToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	m := NewMiddleware(jwtManager, nil, logger, true)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without token in context")
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAdmin(testHandler)

	// Request with user but no token
	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	user := &GitHubUser{ID: 123, Login: "testuser"}
	ctx := context.WithValue(req.Context(), ContextKeyUser, user)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAdmin_AdminUser(t *testing.T) {
	// Create mock GitHub API server that returns enterprise admin status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testGraphQLPath {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": true,
					},
				},
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			AllowEnterpriseAdminMigrations: true,
			RequireEnterpriseSlug:          "test-enterprise",
		},
	}
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	m := NewMiddleware(jwtManager, authorizer, logger, true)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAdmin(testHandler)

	// Request with admin user
	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	user := &GitHubUser{ID: 123, Login: "adminuser"}
	ctx := context.WithValue(req.Context(), ContextKeyUser, user)
	ctx = context.WithValue(ctx, ContextKeyGitHubToken, "test-token")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !handlerCalled {
		t.Error("expected handler to be called for admin user")
	}
}

func TestRequireAdmin_NonAdminUser(t *testing.T) {
	// Create mock GitHub API server - user has no admin privileges
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testGraphQLPath:
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": false,
					},
				},
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		case testUserMembershipsOrgPath:
			// Return empty org list - not an org admin
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]map[string]any{}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			AllowEnterpriseAdminMigrations: true,
			RequireEnterpriseSlug:          "test-enterprise",
			EnableSelfService:              true, // This makes non-admins Tier 2
		},
	}
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	m := NewMiddleware(jwtManager, authorizer, logger, true)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for non-admin user")
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAdmin(testHandler)

	// Request with non-admin user
	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	user := &GitHubUser{ID: 456, Login: "regularuser"}
	ctx := context.WithValue(req.Context(), ContextKeyUser, user)
	ctx = context.WithValue(ctx, ContextKeyGitHubToken, "test-token")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	// Check error message
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestRequireAdmin_MigrationTeamMember(t *testing.T) {
	// Create mock GitHub API server - user is member of migration admin team
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/myorg/teams/migration-admins/memberships/teamadmin":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		case testGraphQLPath:
			// Enterprise admin check - user is not enterprise admin
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": false,
					},
				},
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			MigrationAdminTeams:            []string{"myorg/migration-admins"},
			AllowEnterpriseAdminMigrations: true,
			RequireEnterpriseSlug:          "test-enterprise",
		},
	}
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	m := NewMiddleware(jwtManager, authorizer, logger, true)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAdmin(testHandler)

	// Request with migration team member
	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	user := &GitHubUser{ID: 789, Login: "teamadmin"}
	ctx := context.WithValue(req.Context(), ContextKeyUser, user)
	ctx = context.WithValue(ctx, ContextKeyGitHubToken, "test-token")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !handlerCalled {
		t.Error("expected handler to be called for migration team member")
	}
}

func TestRequireAdmin_OrgAdmin(t *testing.T) {
	// Create mock GitHub API server - user is org admin
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testUserMembershipsOrgPath:
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]map[string]any{
				{
					"organization": map[string]string{"login": "test-org"},
					"state":        "active",
				},
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		case testOrgMembershipPath:
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{
				"state": "active",
				"role":  "admin",
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		case testGraphQLPath:
			// Enterprise admin check - user is not enterprise admin
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": false,
					},
				},
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			AllowOrgAdminMigrations:        true,
			AllowEnterpriseAdminMigrations: true,
			RequireEnterpriseSlug:          "test-enterprise",
		},
	}
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	m := NewMiddleware(jwtManager, authorizer, logger, true)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAdmin(testHandler)

	// Request with org admin
	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	user := &GitHubUser{ID: 321, Login: "orgadmin"}
	ctx := context.WithValue(req.Context(), ContextKeyUser, user)
	ctx = context.WithValue(ctx, ContextKeyGitHubToken, "test-token")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !handlerCalled {
		t.Error("expected handler to be called for org admin")
	}
}

func TestRequireAdmin_ChainedWithRequireAuth(t *testing.T) {
	// Test the full chain: RequireAuth -> RequireAdmin
	// This simulates how the middleware is used in server.go

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testGraphQLPath {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"enterprise": map[string]any{
						"slug":          "test-enterprise",
						"viewerIsAdmin": true,
					},
				},
			}); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	cfg := &config.AuthConfig{
		AuthorizationRules: config.AuthorizationRules{
			AllowEnterpriseAdminMigrations: true,
			RequireEnterpriseSlug:          "test-enterprise",
		},
	}
	authorizer := NewAuthorizer(cfg, logger, server.URL)

	m := NewMiddleware(jwtManager, authorizer, logger, true)

	// Create test user and JWT
	user := &GitHubUser{
		ID:    12345,
		Login: "adminuser",
		Name:  "Admin User",
		Email: "admin@example.com",
	}
	jwtToken, err := jwtManager.GenerateToken(user, "gh_admin_token")
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Chain the middlewares as they would be in server.go
	handler := m.RequireAuth(m.RequireAdmin(testHandler))

	// Create request with JWT cookie
	req := httptest.NewRequest("PUT", "/api/v1/settings", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: jwtToken,
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !handlerCalled {
		t.Error("expected handler to be called for authenticated admin user")
	}
}
