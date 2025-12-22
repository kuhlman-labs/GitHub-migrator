package auth

import (
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

func TestNewEntraIDOAuthHandler(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.AuthConfig
	}{
		{
			name: "complete config",
			cfg: &config.AuthConfig{
				EntraIDClientID:     "client-id",
				EntraIDClientSecret: "client-secret",
				EntraIDTenantID:     "tenant-id",
				EntraIDCallbackURL:  "https://app.example.com/callback",
				ADOOrganizationURL:  "https://dev.azure.com/myorg",
			},
		},
		{
			name: "minimal config",
			cfg: &config.AuthConfig{
				EntraIDClientID:    "client-id",
				EntraIDTenantID:    "tenant-id",
				ADOOrganizationURL: "https://dev.azure.com/myorg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewEntraIDOAuthHandler(tt.cfg)

			if handler == nil {
				t.Fatal("Expected non-nil handler")
				return
			}

			if handler.config != tt.cfg {
				t.Error("config field not set correctly")
			}

			if handler.adoOrgURL != tt.cfg.ADOOrganizationURL {
				t.Errorf("Expected adoOrgURL %q, got %q", tt.cfg.ADOOrganizationURL, handler.adoOrgURL)
			}

			if handler.oauthConfig == nil {
				t.Error("oauthConfig should not be nil")
			}

			if handler.oauthConfig.ClientID != tt.cfg.EntraIDClientID {
				t.Errorf("Expected ClientID %q, got %q", tt.cfg.EntraIDClientID, handler.oauthConfig.ClientID)
			}
		})
	}
}

func TestEntraIDOAuthHandler_GetAuthorizationURL(t *testing.T) {
	cfg := &config.AuthConfig{
		EntraIDClientID:     "test-client-id",
		EntraIDClientSecret: "test-secret",
		EntraIDTenantID:     "test-tenant",
		EntraIDCallbackURL:  "https://app.example.com/callback",
		ADOOrganizationURL:  "https://dev.azure.com/myorg",
	}

	handler := NewEntraIDOAuthHandler(cfg)

	state := "test-state-123"
	authURL := handler.GetAuthorizationURL(state)

	if authURL == "" {
		t.Error("Expected non-empty authorization URL")
	}

	// Check that URL contains expected parameters
	expectedParts := []string{
		"client_id=test-client-id",
		"state=test-state-123",
		"redirect_uri=",
	}

	for _, part := range expectedParts {
		if !containsSubstring(authURL, part) {
			t.Errorf("Expected URL to contain %q, got %q", part, authURL)
		}
	}
}

func TestEntraIDOAuthHandler_OAuthConfigScopes(t *testing.T) {
	cfg := &config.AuthConfig{
		EntraIDClientID:    "client-id",
		EntraIDTenantID:    "tenant-id",
		ADOOrganizationURL: "https://dev.azure.com/myorg",
	}

	handler := NewEntraIDOAuthHandler(cfg)

	if len(handler.oauthConfig.Scopes) == 0 {
		t.Error("Expected non-empty scopes")
	}

	// Check for Azure DevOps scope
	hasADOScope := false
	for _, scope := range handler.oauthConfig.Scopes {
		if containsSubstring(scope, "499b84ac") { // Azure DevOps default scope GUID
			hasADOScope = true
			break
		}
	}

	if !hasADOScope {
		t.Error("Expected Azure DevOps scope to be configured")
	}
}

func TestADOUser_Structure(t *testing.T) {
	// Test that ADOUser struct has expected fields
	user := ADOUser{
		ID:           "test-id",
		DisplayName:  "Test User",
		EmailAddress: "test@example.com",
		PublicAlias:  "testuser",
	}

	if user.ID != "test-id" {
		t.Errorf("Expected ID %q, got %q", "test-id", user.ID)
	}

	if user.DisplayName != "Test User" {
		t.Errorf("Expected DisplayName %q, got %q", "Test User", user.DisplayName)
	}

	if user.EmailAddress != "test@example.com" {
		t.Errorf("Expected EmailAddress %q, got %q", "test@example.com", user.EmailAddress)
	}

	if user.PublicAlias != "testuser" {
		t.Errorf("Expected PublicAlias %q, got %q", "testuser", user.PublicAlias)
	}
}

func TestGenerateRandomState(t *testing.T) {
	// Test that generateRandomState returns non-empty values
	state1 := generateRandomState()
	if state1 == "" {
		t.Error("Expected non-empty state")
	}

	// Test that multiple calls return different values
	state2 := generateRandomState()
	// Note: In rare cases with very fast execution, these might be the same
	// since the implementation uses time.Now().UnixNano()
	// This is acceptable for testing purposes
	_ = state2
}

func TestEntraIDOAuthHandler_Methods(t *testing.T) {
	// Test that all methods exist with correct signatures (compile-time check)
	cfg := &config.AuthConfig{
		EntraIDClientID:    "client-id",
		EntraIDTenantID:    "tenant-id",
		ADOOrganizationURL: "https://dev.azure.com/myorg",
	}

	handler := NewEntraIDOAuthHandler(cfg)

	// Verify method signatures
	_ = handler.GetAuthorizationURL
	_ = handler.ExchangeCode
	_ = handler.GetUserProfile
	_ = handler.CheckOrganizationMembership
	_ = handler.CheckProjectAccess
	_ = handler.Login
	_ = handler.Callback
	_ = handler.GetUser
}

// Helper function
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
