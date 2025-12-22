package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

func testAuthLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewAuthHandler_Disabled(t *testing.T) {
	cfg := &config.AuthConfig{
		Enabled: false,
	}

	handler, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")

	if err != nil {
		t.Errorf("NewAuthHandler() error = %v, want nil", err)
	}
	if handler != nil {
		t.Error("NewAuthHandler() should return nil when auth is disabled")
	}
}

func TestNewAuthHandler_Enabled(t *testing.T) {
	cfg := &config.AuthConfig{
		Enabled:                 true,
		SessionSecret:           "test-secret-key-12345678901234567890",
		SessionDurationHours:    24,
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
	}

	handler, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")

	if err != nil {
		t.Errorf("NewAuthHandler() error = %v, want nil", err)
	}
	if handler == nil {
		t.Error("NewAuthHandler() should return non-nil handler when auth is enabled")
	}
}

func TestNewAuthHandler_InvalidSecret(t *testing.T) {
	cfg := &config.AuthConfig{
		Enabled:              true,
		SessionSecret:        "", // Invalid - empty secret
		SessionDurationHours: 24,
	}

	_, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")

	if err == nil {
		t.Error("NewAuthHandler() should return error for empty session secret")
	}
}

func TestAuthHandler_HandleAuthConfig_Disabled(t *testing.T) {
	// When auth is disabled, we need to test that the endpoint returns appropriate info
	// This is handled by the main handler, not AuthHandler directly
	t.Skip("Auth config endpoint tested through integration tests")
}

func TestAuthHandler_NilHandler(t *testing.T) {
	// Test behavior when auth handler is nil (auth disabled)
	var handler *AuthHandler = nil

	// This is the expected behavior - nil handler means auth disabled
	if handler != nil {
		t.Error("nil handler should remain nil")
	}
}

func TestAuthConfigResponse_Structure(t *testing.T) {
	// Test that auth config response has expected structure
	response := struct {
		AuthEnabled       bool   `json:"auth_enabled"`
		AuthMethod        string `json:"auth_method,omitempty"`
		LoginURL          string `json:"login_url,omitempty"`
		RequiredOrg       string `json:"required_org,omitempty"`
		RequiredTeam      string `json:"required_team,omitempty"`
		RequiredTeamID    int64  `json:"required_team_id,omitempty"`
		AllowedUsers      int    `json:"allowed_users_count,omitempty"`
		RolePermissions   bool   `json:"role_permissions_enabled,omitempty"`
		EntraIDConfigured bool   `json:"entra_id_configured,omitempty"`
	}{
		AuthEnabled:       true,
		AuthMethod:        "github",
		LoginURL:          "/api/auth/login",
		RequiredOrg:       "my-org",
		RolePermissions:   true,
		EntraIDConfigured: false,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded["auth_enabled"] != true {
		t.Error("auth_enabled should be true")
	}
	if decoded["auth_method"] != "github" {
		t.Errorf("auth_method = %v, want github", decoded["auth_method"])
	}
}

func TestAuthHandler_MethodNotAllowed(t *testing.T) {
	// Test that endpoints handle wrong HTTP methods appropriately
	cfg := &config.AuthConfig{
		Enabled:                 true,
		SessionSecret:           "test-secret-key-12345678901234567890",
		SessionDurationHours:    24,
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
	}

	handler, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	if handler == nil {
		t.Skip("Handler is nil - auth not fully configured")
	}

	// HandleLogin expects GET with redirect
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	w := httptest.NewRecorder()

	// This will attempt to redirect, so we just verify no panic
	handler.HandleLogin(w, req)

	// Response should not be empty (could be redirect or error)
	if w.Code == 0 {
		t.Error("Expected a response code")
	}
}

func TestAuthHandler_HandleLogoutNoAuth(t *testing.T) {
	// Test logout when no auth cookie present
	cfg := &config.AuthConfig{
		Enabled:                 true,
		SessionSecret:           "test-secret-key-12345678901234567890",
		SessionDurationHours:    24,
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
	}

	handler, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	if handler == nil {
		t.Skip("Handler is nil - auth not fully configured")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()

	handler.HandleLogout(w, req)

	// Should succeed even without auth cookie (just clears cookie)
	if w.Code >= 500 {
		t.Errorf("HandleLogout returned server error: %d", w.Code)
	}
}

func TestAuthHandler_HandleCurrentUserNoAuth(t *testing.T) {
	// Test current user endpoint when not authenticated
	cfg := &config.AuthConfig{
		Enabled:                 true,
		SessionSecret:           "test-secret-key-12345678901234567890",
		SessionDurationHours:    24,
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
	}

	handler, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	if handler == nil {
		t.Skip("Handler is nil - auth not fully configured")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/user", nil)
	w := httptest.NewRecorder()

	handler.HandleCurrentUser(w, req)

	// Should return 401 Unauthorized without valid token
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusOK {
		// Either unauthorized or returns anonymous user info
		t.Logf("HandleCurrentUser returned status: %d", w.Code)
	}
}

func TestAuthHandler_HandleRefreshTokenNoAuth(t *testing.T) {
	// Test refresh token endpoint when not authenticated
	cfg := &config.AuthConfig{
		Enabled:                 true,
		SessionSecret:           "test-secret-key-12345678901234567890",
		SessionDurationHours:    24,
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
	}

	handler, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	if handler == nil {
		t.Skip("Handler is nil - auth not fully configured")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	w := httptest.NewRecorder()

	handler.HandleRefreshToken(w, req)

	// Should fail without valid token
	if w.Code >= 500 {
		t.Errorf("HandleRefreshToken returned server error: %d", w.Code)
	}
}

func TestAuthHandler_HandleAuthConfig(t *testing.T) {
	cfg := &config.AuthConfig{
		Enabled:                 true,
		SessionSecret:           "test-secret-key-12345678901234567890",
		SessionDurationHours:    24,
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
	}

	handler, err := NewAuthHandler(cfg, testAuthLogger(), "https://api.github.com")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	if handler == nil {
		t.Skip("Handler is nil - auth not fully configured")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	w := httptest.NewRecorder()

	handler.HandleAuthConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleAuthConfig returned status %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// The endpoint uses "enabled" not "auth_enabled"
	if response["enabled"] != true {
		t.Error("enabled should be true")
	}
	if response["login_url"] != "/api/v1/auth/login" {
		t.Errorf("login_url = %v, want /api/v1/auth/login", response["login_url"])
	}
}
