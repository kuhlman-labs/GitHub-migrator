package source

import (
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

func TestNewProviderFromConfig(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.SourceConfig
		expectError  bool
		expectedType ProviderType
	}{
		{
			name: "GitHub provider",
			cfg: config.SourceConfig{
				Type:    "github",
				BaseURL: "https://api.github.com",
				Token:   "test_token",
			},
			expectError:  false,
			expectedType: ProviderGitHub,
		},
		{
			name: "GitHub provider (case insensitive)",
			cfg: config.SourceConfig{
				Type:    "GitHub",
				BaseURL: "https://api.github.com",
				Token:   "test_token",
			},
			expectError:  false,
			expectedType: ProviderGitHub,
		},
		{
			name: "GitLab provider",
			cfg: config.SourceConfig{
				Type:    "gitlab",
				BaseURL: "https://gitlab.com",
				Token:   "test_token",
			},
			expectError:  false,
			expectedType: ProviderGitLab,
		},
		{
			name: "Azure DevOps provider",
			cfg: config.SourceConfig{
				Type:         "azuredevops",
				Organization: "myorg",
				Token:        "test_token",
			},
			expectError:  false,
			expectedType: ProviderAzureDevOps,
		},
		{
			name: "Azure DevOps provider (short name)",
			cfg: config.SourceConfig{
				Type:         "ado",
				Organization: "myorg",
				Token:        "test_token",
			},
			expectError:  false,
			expectedType: ProviderAzureDevOps,
		},
		{
			name: "Azure DevOps without organization",
			cfg: config.SourceConfig{
				Type:  "azuredevops",
				Token: "test_token",
			},
			expectError: true,
		},
		{
			name: "Unsupported provider type",
			cfg: config.SourceConfig{
				Type:  "bitbucket",
				Token: "test_token",
			},
			expectError: true,
		},
		{
			name: "Empty token",
			cfg: config.SourceConfig{
				Type:    "github",
				BaseURL: "https://api.github.com",
				Token:   "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProviderFromConfig(tt.cfg)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if provider == nil {
				t.Error("Expected provider but got nil")
				return
			}
			if provider.Type() != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, provider.Type())
			}
		})
	}
}

func TestNewDestinationProviderFromConfig(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.DestinationConfig
		expectError  bool
		expectedType ProviderType
	}{
		{
			name: "GitHub destination",
			cfg: config.DestinationConfig{
				Type:    "github",
				BaseURL: "https://api.github.com",
				Token:   "test_token",
			},
			expectError:  false,
			expectedType: ProviderGitHub,
		},
		{
			name: "GitLab destination",
			cfg: config.DestinationConfig{
				Type:    "gitlab",
				BaseURL: "https://gitlab.com",
				Token:   "test_token",
			},
			expectError:  false,
			expectedType: ProviderGitLab,
		},
		{
			name: "Azure DevOps destination (not implemented)",
			cfg: config.DestinationConfig{
				Type:  "azuredevops",
				Token: "test_token",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewDestinationProviderFromConfig(tt.cfg)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if provider == nil {
				t.Error("Expected provider but got nil")
				return
			}
			if provider.Type() != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, provider.Type())
			}
		})
	}
}
