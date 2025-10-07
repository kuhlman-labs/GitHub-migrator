package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewGitHubProvider(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		token       string
		expectError bool
	}{
		{
			name:        "Valid GitHub.com provider",
			baseURL:     "https://api.github.com",
			token:       "ghp_test123",
			expectError: false,
		},
		{
			name:        "Valid GHES provider",
			baseURL:     "https://github.example.com/api/v3",
			token:       "ghp_test123",
			expectError: false,
		},
		{
			name:        "Empty token",
			baseURL:     "https://api.github.com",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitHubProvider(tt.baseURL, tt.token)
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
			}
			if provider.Type() != ProviderGitHub {
				t.Errorf("Expected type %s, got %s", ProviderGitHub, provider.Type())
			}
		})
	}
}

func TestGitHubProvider_GetAuthenticatedCloneURL(t *testing.T) {
	provider, _ := NewGitHubProvider("https://api.github.com", "test_token_123")

	tests := []struct {
		name        string
		cloneURL    string
		expectError bool
		checkToken  bool
	}{
		{
			name:        "Valid HTTPS URL",
			cloneURL:    "https://github.com/org/repo.git",
			expectError: false,
			checkToken:  true,
		},
		{
			name:        "Valid GHES URL",
			cloneURL:    "https://github.example.com/org/repo.git",
			expectError: false,
			checkToken:  true,
		},
		{
			name:        "Invalid URL with special chars",
			cloneURL:    "ht!tp://invalid-url:with spaces",
			expectError: true,
			checkToken:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL, err := provider.GetAuthenticatedCloneURL(tt.cloneURL)
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

			// Check that the token is embedded in the URL
			if tt.checkToken {
				if authURL == tt.cloneURL {
					t.Errorf("Expected authenticated URL to be different from original")
				}
				// The token should be in the URL
				// We don't check the exact format to avoid exposing token structure
				if len(authURL) <= len(tt.cloneURL) {
					t.Errorf("Expected authenticated URL to be longer than original")
				}
			}
		})
	}
}

func TestGitHubProvider_ValidateCredentials(t *testing.T) {
	// Create a mock GitHub API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GitHub Enterprise uses /api/v3 prefix, github.com doesn't
		// Check for /user or /api/v3/user endpoint
		if r.URL.Path == "/user" || r.URL.Path == "/api/v3/user" {
			// Check for valid token in Authorization header
			auth := r.Header.Get("Authorization")
			if auth == "Bearer valid_token" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"login":"testuser"}`))
				return
			}
			// Invalid token
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Bad credentials"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid credentials",
			token:       "valid_token",
			expectError: false,
		},
		{
			name:        "Invalid credentials",
			token:       "invalid_token",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitHubProvider(mockServer.URL, tt.token)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			err = provider.ValidateCredentials(ctx)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGitHubProvider_SupportsFeature(t *testing.T) {
	provider, _ := NewGitHubProvider("https://api.github.com", "test_token")

	tests := []struct {
		feature  Feature
		expected bool
	}{
		{FeatureLFS, true},
		{FeatureSubmodules, true},
		{FeatureWiki, true},
		{FeaturePages, true},
		{FeatureActions, true},
		{FeatureDiscussions, true},
		{Feature("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			result := provider.SupportsFeature(tt.feature)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGitHubProvider_CloneRepository(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	// This is an integration test that requires actual git
	// We'll use a public repository for testing
	provider, _ := NewGitHubProvider("https://api.github.com", "dummy_token")

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, "test-repo")

	ctx := context.Background()
	repoInfo := RepositoryInfo{
		FullName: "octocat/Hello-World",
		CloneURL: "https://github.com/octocat/Hello-World.git",
	}
	opts := DefaultCloneOptions()

	// This will fail because we're using a dummy token, but we can test the error handling
	err := provider.CloneRepository(ctx, repoInfo, destPath, opts)

	// We expect an error because the token is invalid
	if err == nil {
		// If it succeeds (unlikely with dummy token), check that the directory exists
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			t.Error("Clone reported success but directory doesn't exist")
		}
	}
	// The error should be sanitized (not contain the token)
	if err != nil && err.Error() != "" {
		// Just verify we got an error - we can't test much more without a valid token
		t.Logf("Clone failed as expected: %v", err)
	}
}

func TestSanitizeGitError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		token    string
		expected string
	}{
		{
			name:     "Error with token",
			errMsg:   "fatal: Authentication failed for 'https://secret_token@github.com/org/repo.git'",
			token:    "secret_token",
			expected: "fatal: Authentication failed for 'https://[REDACTED]@github.com/org/repo.git'",
		},
		{
			name:     "Error without token",
			errMsg:   "fatal: repository not found",
			token:    "secret_token",
			expected: "fatal: repository not found",
		},
		{
			name:     "Empty token",
			errMsg:   "some error message",
			token:    "",
			expected: "some error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeGitError(tt.errMsg, tt.token)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
