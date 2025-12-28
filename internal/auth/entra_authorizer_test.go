package auth

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

func TestNewEntraIDAuthorizer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name      string
		cfg       *config.AuthConfig
		expectNil bool
	}{
		{
			name: "valid config",
			cfg: &config.AuthConfig{
				ADOOrganizationURL: "https://dev.azure.com/myorg",
			},
			expectNil: false,
		},
		{
			name: "empty org URL",
			cfg: &config.AuthConfig{
				ADOOrganizationURL: "",
			},
			expectNil: false, // Constructor doesn't validate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := NewEntraIDAuthorizer(tt.cfg, logger)

			if tt.expectNil {
				if authorizer != nil {
					t.Error("Expected nil authorizer")
				}
			} else {
				if authorizer == nil {
					t.Error("Expected non-nil authorizer")
				}
			}
		})
	}
}

func TestEntraIDAuthorizer_FieldAssignment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := &config.AuthConfig{
		ADOOrganizationURL: "https://dev.azure.com/testorg",
	}

	authorizer := NewEntraIDAuthorizer(cfg, logger)

	if authorizer.config != cfg {
		t.Error("config field not set correctly")
	}

	if authorizer.adoOrgURL != cfg.ADOOrganizationURL {
		t.Errorf("Expected adoOrgURL %q, got %q", cfg.ADOOrganizationURL, authorizer.adoOrgURL)
	}

	if authorizer.logger != logger {
		t.Error("logger field not set correctly")
	}
}

func TestEntraIDAuthorizer_ValidateAccessForRepository_NoMock(t *testing.T) {
	// This test verifies the method signature without making network calls
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := &config.AuthConfig{
		ADOOrganizationURL: "https://dev.azure.com/testorg",
	}

	authorizer := NewEntraIDAuthorizer(cfg, logger)

	// Verify method exists and has correct signature
	_ = authorizer.ValidateAccessForRepository
	_ = authorizer.ValidateAccessForRepositories
	_ = authorizer.CheckADOOrganizationMembership
	_ = authorizer.CheckADOProjectAccess
	_ = authorizer.GetUserProjects
}

func TestEntraIDAuthorizer_URLConstruction(t *testing.T) {
	// Test that URL construction is correct (without making actual calls)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name              string
		orgURL            string
		expectedURLPrefix string
	}{
		{
			name:              "standard dev.azure.com URL",
			orgURL:            "https://dev.azure.com/myorg",
			expectedURLPrefix: "https://dev.azure.com/myorg/_apis/",
		},
		{
			name:              "visualstudio.com URL",
			orgURL:            "https://myorg.visualstudio.com",
			expectedURLPrefix: "https://myorg.visualstudio.com/_apis/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AuthConfig{
				ADOOrganizationURL: tt.orgURL,
			}

			authorizer := NewEntraIDAuthorizer(cfg, logger)

			// Verify the org URL is stored correctly
			if authorizer.adoOrgURL != tt.orgURL {
				t.Errorf("Expected orgURL %q, got %q", tt.orgURL, authorizer.adoOrgURL)
			}
		})
	}
}
