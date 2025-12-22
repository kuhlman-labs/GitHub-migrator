package source

import (
	"testing"
)

func TestNewAzureDevOpsProvider(t *testing.T) {
	tests := []struct {
		name         string
		organization string
		token        string
		username     string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid provider with all fields",
			organization: "myorg",
			token:        "my-pat-token",
			username:     "myuser",
			wantErr:      false,
		},
		{
			name:         "valid provider without username",
			organization: "myorg",
			token:        "my-pat-token",
			username:     "",
			wantErr:      false,
		},
		{
			name:         "missing organization",
			organization: "",
			token:        "my-pat-token",
			username:     "myuser",
			wantErr:      true,
			errContains:  "organization is required",
		},
		{
			name:         "missing token",
			organization: "myorg",
			token:        "",
			username:     "myuser",
			wantErr:      true,
			errContains:  "token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAzureDevOpsProvider(tt.organization, tt.token, tt.username)

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
			if provider.Type() != ProviderAzureDevOps {
				t.Errorf("Expected type %v, got %v", ProviderAzureDevOps, provider.Type())
			}

			// Verify name format
			expectedNameContains := tt.organization
			if !contains(provider.Name(), expectedNameContains) {
				t.Errorf("Expected name to contain %q, got %q", expectedNameContains, provider.Name())
			}
		})
	}
}

func TestAzureDevOpsProvider_Type(t *testing.T) {
	provider, err := NewAzureDevOpsProvider("myorg", "token", "")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.Type() != ProviderAzureDevOps {
		t.Errorf("Expected type %v, got %v", ProviderAzureDevOps, provider.Type())
	}
}

func TestAzureDevOpsProvider_Name(t *testing.T) {
	provider, err := NewAzureDevOpsProvider("testorg", "token", "")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	name := provider.Name()
	if name != "Azure DevOps (testorg)" {
		t.Errorf("Expected name 'Azure DevOps (testorg)', got %q", name)
	}
}

func TestAzureDevOpsProvider_GetAuthenticatedCloneURL(t *testing.T) {
	provider, err := NewAzureDevOpsProvider("myorg", "mysecretpat", "")
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
			cloneURL:     "https://dev.azure.com/myorg/myproject/_git/myrepo",
			wantErr:      false,
			expectedPart: "mysecretpat@dev.azure.com",
		},
		{
			name:         "valid HTTPS URL with trailing path",
			cloneURL:     "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			wantErr:      false,
			expectedPart: "mysecretpat@dev.azure.com",
		},
		{
			name:     "empty URL",
			cloneURL: "",
			wantErr:  true,
		},
		{
			name:     "invalid URL",
			cloneURL: "not-a-url",
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

func TestAzureDevOpsProvider_SupportsFeature(t *testing.T) {
	provider, err := NewAzureDevOpsProvider("myorg", "token", "")
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
		{FeaturePages, false},       // GitHub-specific
		{FeatureActions, false},     // GitHub-specific
		{FeatureDiscussions, false}, // GitHub-specific
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

func TestAzureDevOpsProvider_NormalizeRepoURL(t *testing.T) {
	provider, err := NewAzureDevOpsProvider("myorg", "token", "")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name      string
		rawURL    string
		wantErr   bool
		checkFunc func(string) bool
	}{
		{
			name:    "dev.azure.com URL unchanged",
			rawURL:  "https://dev.azure.com/myorg/myproject/_git/myrepo",
			wantErr: false,
			checkFunc: func(url string) bool {
				return contains(url, "dev.azure.com")
			},
		},
		{
			name:    "visualstudio.com URL converted",
			rawURL:  "https://myorg.visualstudio.com/myproject/_git/myrepo",
			wantErr: false,
			checkFunc: func(url string) bool {
				return contains(url, "dev.azure.com")
			},
		},
		{
			name:    "invalid URL",
			rawURL:  "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.NormalizeRepoURL(tt.rawURL)

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

			if tt.checkFunc != nil && !tt.checkFunc(result) {
				t.Errorf("Result %q did not pass check", result)
			}
		})
	}
}

func TestAzureDevOpsProvider_DefaultUsername(t *testing.T) {
	// When username is empty, it should default to the token
	provider, err := NewAzureDevOpsProvider("myorg", "mytoken", "")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// The username field should be set to token when empty
	if provider.username != "mytoken" {
		t.Errorf("Expected username to default to token, got %q", provider.username)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
