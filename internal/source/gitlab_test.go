package source

import (
	"context"
	"testing"
)

func TestNewGitLabProvider(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		token       string
		wantErr     bool
		errContains string
		expectName  string
	}{
		{
			name:       "valid provider with gitlab.com",
			baseURL:    "https://gitlab.com",
			token:      "my-token",
			wantErr:    false,
			expectName: "GitLab.com",
		},
		{
			name:       "valid provider with empty baseURL defaults to gitlab.com",
			baseURL:    "",
			token:      "my-token",
			wantErr:    false,
			expectName: "GitLab.com",
		},
		{
			name:       "valid provider with self-hosted instance",
			baseURL:    "https://gitlab.example.com",
			token:      "my-token",
			wantErr:    false,
			expectName: "GitLab (gitlab.example.com)",
		},
		{
			name:        "missing token",
			baseURL:     "https://gitlab.com",
			token:       "",
			wantErr:     true,
			errContains: "token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitLabProvider(tt.baseURL, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("Expected non-nil provider")
				return
			}

			// Verify provider type
			if provider.Type() != ProviderGitLab {
				t.Errorf("Expected type %v, got %v", ProviderGitLab, provider.Type())
			}

			// Verify name
			if tt.expectName != "" && provider.Name() != tt.expectName {
				t.Errorf("Expected name %q, got %q", tt.expectName, provider.Name())
			}
		})
	}
}

func TestGitLabProvider_Type(t *testing.T) {
	provider, err := NewGitLabProvider("https://gitlab.com", "token")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.Type() != ProviderGitLab {
		t.Errorf("Expected type %v, got %v", ProviderGitLab, provider.Type())
	}
}

func TestGitLabProvider_Name(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		expectName string
	}{
		{
			name:       "gitlab.com",
			baseURL:    "https://gitlab.com",
			expectName: "GitLab.com",
		},
		{
			name:       "self-hosted",
			baseURL:    "https://gitlab.mycompany.com",
			expectName: "GitLab (gitlab.mycompany.com)",
		},
		{
			name:       "empty defaults to gitlab.com",
			baseURL:    "",
			expectName: "GitLab.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitLabProvider(tt.baseURL, "token")
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			if provider.Name() != tt.expectName {
				t.Errorf("Expected name %q, got %q", tt.expectName, provider.Name())
			}
		})
	}
}

func TestGitLabProvider_GetAuthenticatedCloneURL(t *testing.T) {
	provider, err := NewGitLabProvider("https://gitlab.com", "mysecrettoken")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name         string
		cloneURL     string
		wantErr      bool
		expectedPart string
	}{
		{
			name:         "valid HTTPS URL",
			cloneURL:     "https://gitlab.com/myorg/myrepo.git",
			wantErr:      false,
			expectedPart: "oauth2:mysecrettoken@gitlab.com",
		},
		{
			name:         "valid URL without .git suffix",
			cloneURL:     "https://gitlab.com/myorg/myrepo",
			wantErr:      false,
			expectedPart: "oauth2:mysecrettoken@gitlab.com",
		},
		{
			name:     "invalid URL",
			cloneURL: "://invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.GetAuthenticatedCloneURL(tt.cloneURL)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectedPart != "" && !contains(result, tt.expectedPart) {
				t.Errorf("Expected URL to contain %q, got %q", tt.expectedPart, result)
			}
		})
	}
}

func TestGitLabProvider_SupportsFeature(t *testing.T) {
	provider, err := NewGitLabProvider("https://gitlab.com", "token")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		feature  Feature
		expected bool
	}{
		{FeatureLFS, true},
		{FeatureSubmodules, true},
		{FeatureWiki, true},
		{FeaturePages, true},
		{FeatureActions, false},     // GitLab has CI/CD but not GitHub Actions
		{FeatureDiscussions, false}, // Not GitHub-style discussions
		{Feature("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			result := provider.SupportsFeature(tt.feature)
			if result != tt.expected {
				t.Errorf("SupportsFeature(%q) = %v, want %v", tt.feature, result, tt.expected)
			}
		})
	}
}

func TestGitLabProvider_CloneRepository_NotImplemented(t *testing.T) {
	provider, err := NewGitLabProvider("https://gitlab.com", "token")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// CloneRepository should return "not yet implemented" error
	err = provider.CloneRepository(context.TODO(), RepositoryInfo{}, "", CloneOptions{})
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !contains(err.Error(), "not yet implemented") {
		t.Errorf("Expected error to contain 'not yet implemented', got %q", err.Error())
	}
}

func TestGitLabProvider_ValidateCredentials_NotImplemented(t *testing.T) {
	provider, err := NewGitLabProvider("https://gitlab.com", "token")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// ValidateCredentials should return "not yet implemented" error
	err = provider.ValidateCredentials(context.TODO())
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !contains(err.Error(), "not yet implemented") {
		t.Errorf("Expected error to contain 'not yet implemented', got %q", err.Error())
	}
}
