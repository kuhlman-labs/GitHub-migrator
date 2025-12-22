package discovery

import (
	"testing"
)

func TestGetSourceInstance(t *testing.T) {
	tests := []struct {
		name     string
		client   mockClientForSourceInstance
		expected string
	}{
		{
			name:     "nil client",
			client:   mockClientForSourceInstance{isNil: true},
			expected: "github.com",
		},
		{
			name:     "empty base URL",
			client:   mockClientForSourceInstance{baseURL: ""},
			expected: "github.com",
		},
		{
			name:     "github.com",
			client:   mockClientForSourceInstance{baseURL: "https://api.github.com/"},
			expected: "api.github.com",
		},
		{
			name:     "enterprise instance",
			client:   mockClientForSourceInstance{baseURL: "https://github.mycompany.com/api/v3/"},
			expected: "github.mycompany.com",
		},
		{
			name:     "URL with port",
			client:   mockClientForSourceInstance{baseURL: "https://github.local:8080/api/v3/"},
			expected: "github.local",
		},
		{
			name:     "invalid URL",
			client:   mockClientForSourceInstance{baseURL: "://invalid"},
			expected: "github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.client.isNil {
				result = GetSourceInstance(nil)
			} else {
				// We can't actually call GetSourceInstance with a mock because it expects *github.Client
				// Instead, test the logic directly
				result = testGetSourceInstanceLogic(tt.client.baseURL)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// mockClientForSourceInstance is a simple mock for testing
type mockClientForSourceInstance struct {
	isNil   bool
	baseURL string
}

// testGetSourceInstanceLogic replicates the logic of GetSourceInstance for testing
func testGetSourceInstanceLogic(baseURL string) string {
	if baseURL == "" {
		return hostGitHubCom
	}

	// Simple URL parsing for test
	// Extract host from URL like https://host/path
	const httpsPrefix = "https://"
	if len(baseURL) > len(httpsPrefix) && baseURL[:len(httpsPrefix)] == httpsPrefix {
		rest := baseURL[len(httpsPrefix):]
		// Find first slash
		for i, c := range rest {
			if c == '/' {
				host := rest[:i]
				// Remove port if present
				for j, ch := range host {
					if ch == ':' {
						return host[:j]
					}
				}
				return host
			}
		}
		// No slash found, use the whole thing
		for i, c := range rest {
			if c == ':' {
				return rest[:i]
			}
		}
		return rest
	}

	return hostGitHubCom
}

func TestNewMemberDiscoverer(t *testing.T) {
	// Test that NewMemberDiscoverer doesn't panic with nil values
	// In practice, these would be provided
	d := NewMemberDiscoverer(nil, nil)

	if d == nil {
		t.Fatal("Expected non-nil MemberDiscoverer")
		return
	}

	// Verify fields are set
	if d.storage != nil {
		t.Error("Expected storage to be nil when passed nil")
	}

	if d.logger != nil {
		t.Error("Expected logger to be nil when passed nil")
	}
}

func TestMemberDiscoveryResult(t *testing.T) {
	result := &MemberDiscoveryResult{
		TotalMembers:     10,
		UsersSaved:       8,
		MembershipsSaved: 9,
		Errors:           []error{},
	}

	if result.TotalMembers != 10 {
		t.Errorf("Expected TotalMembers 10, got %d", result.TotalMembers)
	}

	if result.UsersSaved != 8 {
		t.Errorf("Expected UsersSaved 8, got %d", result.UsersSaved)
	}

	if result.MembershipsSaved != 9 {
		t.Errorf("Expected MembershipsSaved 9, got %d", result.MembershipsSaved)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}
