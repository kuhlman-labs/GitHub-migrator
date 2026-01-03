package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// Helper to create context with user and token for testing
func createAuthContext(user *auth.GitHubUser, token string) context.Context {
	ctx := context.WithValue(context.Background(), auth.ContextKeyUser, user)
	ctx = context.WithValue(ctx, auth.ContextKeyGitHubToken, token)
	return ctx
}

func TestCheckRepositoryAccess_AuthDisabled(t *testing.T) {
	utils := NewHandlerUtils(nil, nil, nil, "", testLogger())

	err := utils.CheckRepositoryAccess(context.Background(), "org/repo")
	if err != nil {
		t.Errorf("Expected no error when auth is disabled, got: %v", err)
	}
}

func TestCheckRepositoryAccess_NoUserInContext(t *testing.T) {
	cfg := &config.AuthConfig{Enabled: true}
	utils := NewHandlerUtils(cfg, nil, nil, "", testLogger())

	err := utils.CheckRepositoryAccess(context.Background(), "org/repo")
	if err == nil {
		t.Error("Expected error when user is not in context")
	}
	if err.Error() != "authentication required" {
		t.Errorf("Expected 'authentication required' error, got: %v", err)
	}
}

func TestCheckRepositoryAccess_AdminTier(t *testing.T) {
	// Create mock GitHub API server that returns migration team membership
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/orgs/myorg/teams/migration-admins/memberships/testuser" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"state": "active"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		Enabled: true,
		AuthorizationRules: config.AuthorizationRules{
			MigrationAdminTeams: []string{"myorg/migration-admins"},
		},
	}
	utils := NewHandlerUtils(cfg, nil, nil, "", testLogger())
	utils.SetDestinationBaseURL(server.URL)

	// Create context with user and token
	user := &auth.GitHubUser{ID: 123, Login: "testuser"}
	ctx := createAuthContext(user, "test-token")

	err := utils.CheckRepositoryAccess(ctx, "someorg/somerepo")
	if err != nil {
		t.Errorf("Expected no error for admin tier user, got: %v", err)
	}
}

func TestCheckRepositoryAccess_SelfServiceNoMappingRequired(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// User is not in any admin team or org admin
		if r.URL.Path == "/user/memberships/orgs" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		Enabled: true,
		AuthorizationRules: config.AuthorizationRules{
			RequireIdentityMappingForSelfService: false, // Self-service without identity mapping
		},
	}
	utils := NewHandlerUtils(cfg, nil, nil, "", testLogger())
	utils.SetDestinationBaseURL(server.URL)

	// Create context with user and token
	user := &auth.GitHubUser{ID: 123, Login: "testuser"}
	ctx := createAuthContext(user, "test-token")

	err := utils.CheckRepositoryAccess(ctx, "someorg/somerepo")
	if err != nil {
		t.Errorf("Expected no error when identity mapping is not required, got: %v", err)
	}
}

func TestCheckRepositoryAccess_SelfServiceNoMapping(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/memberships/orgs" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		Enabled: true,
		AuthorizationRules: config.AuthorizationRules{
			RequireIdentityMappingForSelfService: true, // Requires identity mapping
		},
	}
	utils := NewHandlerUtils(cfg, nil, nil, "", testLogger())
	utils.SetDestinationBaseURL(server.URL)
	// Note: No database set, so identity mapping lookup will fail

	// Create context with user and token
	user := &auth.GitHubUser{ID: 123, Login: "testuser"}
	ctx := createAuthContext(user, "test-token")

	err := utils.CheckRepositoryAccess(ctx, "someorg/somerepo")
	if err == nil {
		t.Error("Expected error when identity mapping is required but no database is available")
	}
}

func TestGetUserAuthorizationStatus_AuthDisabled(t *testing.T) {
	utils := NewHandlerUtils(nil, nil, nil, "", testLogger())

	status, err := utils.GetUserAuthorizationStatus(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.Tier != string(auth.TierAdmin) {
		t.Errorf("Expected admin tier when auth is disabled, got: %s", status.Tier)
	}
	if !status.Permissions.CanMigrateAllRepos {
		t.Error("Expected CanMigrateAllRepos to be true when auth is disabled")
	}
}

func TestGetUserAuthorizationStatus_NoUserInContext(t *testing.T) {
	cfg := &config.AuthConfig{Enabled: true}
	utils := NewHandlerUtils(cfg, nil, nil, "", testLogger())

	_, err := utils.GetUserAuthorizationStatus(context.Background())
	if err == nil {
		t.Error("Expected error when user is not in context")
	}
}

func TestGetUserAuthorizationStatus_AdminUser(t *testing.T) {
	// Create mock GitHub API server that returns migration team membership
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/orgs/myorg/teams/migration-admins/memberships/testuser" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"state": "active"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		Enabled: true,
		AuthorizationRules: config.AuthorizationRules{
			MigrationAdminTeams: []string{"myorg/migration-admins"},
		},
	}
	utils := NewHandlerUtils(cfg, nil, nil, "", testLogger())
	utils.SetDestinationBaseURL(server.URL)

	// Create context with user and token
	user := &auth.GitHubUser{ID: 123, Login: "testuser"}
	ctx := createAuthContext(user, "test-token")

	status, err := utils.GetUserAuthorizationStatus(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.Tier != string(auth.TierAdmin) {
		t.Errorf("Expected admin tier, got: %s", status.Tier)
	}
	if !status.Permissions.CanMigrateAllRepos {
		t.Error("Expected CanMigrateAllRepos to be true for admin")
	}
	if status.UpgradePath != nil {
		t.Error("Expected no upgrade path for admin user")
	}
}

func TestGetUserAuthorizationStatus_ReadOnlyUser(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/memberships/orgs" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.AuthConfig{
		Enabled: true,
		AuthorizationRules: config.AuthorizationRules{
			RequireIdentityMappingForSelfService: true,
		},
	}
	utils := NewHandlerUtils(cfg, nil, nil, "", testLogger())
	utils.SetDestinationBaseURL(server.URL)

	// Create context with user and token
	user := &auth.GitHubUser{ID: 123, Login: "testuser"}
	ctx := createAuthContext(user, "test-token")

	status, err := utils.GetUserAuthorizationStatus(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.Tier != string(auth.TierReadOnly) {
		t.Errorf("Expected read-only tier, got: %s", status.Tier)
	}
	if status.Permissions.CanMigrateOwnRepos {
		t.Error("Expected CanMigrateOwnRepos to be false for read-only user")
	}
	if status.UpgradePath == nil {
		t.Error("Expected upgrade path for read-only user")
	} else if status.UpgradePath.Action != "complete_identity_mapping" {
		t.Errorf("Expected upgrade action 'complete_identity_mapping', got: %s", status.UpgradePath.Action)
	}
}

// MockDataStoreForHandlerUtils is a minimal mock for testing handler utils
type MockDataStoreForHandlerUtils struct {
	userMappings   map[string]*models.UserMapping
	repositories   map[string]*models.Repository
	sources        map[int64]*models.Source
	returnError    error
}

func NewMockDataStoreForHandlerUtils() *MockDataStoreForHandlerUtils {
	return &MockDataStoreForHandlerUtils{
		userMappings: make(map[string]*models.UserMapping),
		repositories: make(map[string]*models.Repository),
		sources:      make(map[int64]*models.Source),
	}
}

func (m *MockDataStoreForHandlerUtils) GetUserMappingByDestinationLogin(ctx context.Context, login string) (*models.UserMapping, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	mapping, ok := m.userMappings[login]
	if !ok {
		return nil, nil
	}
	return mapping, nil
}

func (m *MockDataStoreForHandlerUtils) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	repo, ok := m.repositories[fullName]
	if !ok {
		return nil, nil
	}
	return repo, nil
}

func (m *MockDataStoreForHandlerUtils) GetSource(ctx context.Context, id int64) (*models.Source, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	source, ok := m.sources[id]
	if !ok {
		return nil, nil
	}
	return source, nil
}

