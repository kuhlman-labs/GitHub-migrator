package azuredevops

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name         string
		config       ClientConfig
		wantErr      bool
		skipCreation bool
	}{
		{
			name: "valid configuration - validation only",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/testorg",
				PersonalAccessToken: "test-token",
			},
			wantErr:      false,
			skipCreation: true, // Skip actual client creation (requires network)
		},
		{
			name: "empty organization URL",
			config: ClientConfig{
				OrganizationURL:     "",
				PersonalAccessToken: "test-token",
			},
			wantErr:      true,
			skipCreation: false,
		},
		{
			name: "empty PAT",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/testorg",
				PersonalAccessToken: "",
			},
			wantErr:      true,
			skipCreation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First, validate configuration
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ClientConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Expected validation error, skip client creation
			}

			if tt.skipCreation {
				t.Skip("Skipping client creation - requires network access to Azure DevOps")
			}

			client, err := NewClient(tt.config)
			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

func TestClient_ListProjects(t *testing.T) {
	// Note: This test requires network access to Azure DevOps
	// In a real environment, you would mock the ADO API responses
	t.Skip("Skipping integration test - requires Azure DevOps access")

	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	projects, err := client.GetProjects(ctx)
	if err != nil {
		t.Fatalf("GetProjects() error: %v", err)
	}

	if projects == nil {
		t.Error("GetProjects() returned nil")
	}
}

func TestClient_ListRepositories(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure DevOps access")

	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	repos, err := client.GetRepositories(ctx, "TestProject")
	if err != nil {
		t.Fatalf("GetRepositories() error: %v", err)
	}

	if repos == nil {
		t.Error("GetRepositories() returned nil")
	}
}

func TestClient_GetRepository(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure DevOps access")

	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	repo, err := client.GetRepository(ctx, "TestProject", "test-repo-id")
	if err != nil {
		t.Fatalf("GetRepository() error: %v", err)
	}

	if repo == nil {
		t.Error("GetRepository() returned nil")
	}
}

func TestClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/testorg",
				PersonalAccessToken: "valid-token-123",
			},
			wantErr: false,
		},
		{
			name: "missing organization URL",
			config: ClientConfig{
				OrganizationURL:     "",
				PersonalAccessToken: "token",
			},
			wantErr: true,
		},
		{
			name: "missing PAT",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/testorg",
				PersonalAccessToken: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetPipelineDefinitions(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure DevOps access")

	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	defs, err := client.GetPipelineDefinitions(ctx, "TestProject", "test-repo-id")
	if err != nil {
		t.Fatalf("GetPipelineDefinitions() error: %v", err)
	}

	if defs == nil {
		t.Error("GetPipelineDefinitions() returned nil")
	}
}

func TestClient_GetPRDetails(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure DevOps access")

	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	openCount, withWorkItems, withAttachments, err := client.GetPRDetails(ctx, "TestProject", "test-repo-id")
	if err != nil {
		t.Fatalf("GetPRDetails() error: %v", err)
	}

	if openCount < 0 || withWorkItems < 0 || withAttachments < 0 {
		t.Error("GetPRDetails() returned negative counts")
	}
}

func TestClient_GetBranchPolicyDetails(t *testing.T) {
	t.Skip("Skipping integration test - requires Azure DevOps access")

	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	policyTypes, requiredReviewers, buildValidations, err := client.GetBranchPolicyDetails(ctx, "TestProject", "test-repo-id")
	if err != nil {
		t.Fatalf("GetBranchPolicyDetails() error: %v", err)
	}

	if policyTypes == nil {
		t.Error("GetBranchPolicyDetails() returned nil policy types")
	}

	if requiredReviewers < 0 || buildValidations < 0 {
		t.Error("GetBranchPolicyDetails() returned negative counts")
	}
}

func TestClientConfig_Structure(t *testing.T) {
	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/myorg",
		PersonalAccessToken: "my-pat-token-123",
		Logger:              nil,
	}

	if config.OrganizationURL != "https://dev.azure.com/myorg" {
		t.Errorf("Expected OrganizationURL 'https://dev.azure.com/myorg', got '%s'", config.OrganizationURL)
	}
	if config.PersonalAccessToken != "my-pat-token-123" {
		t.Errorf("Expected PersonalAccessToken 'my-pat-token-123', got '%s'", config.PersonalAccessToken)
	}
}

func TestTfsGitRepositoryType(t *testing.T) {
	if TfsGitRepositoryType != "TfsGit" {
		t.Errorf("Expected TfsGitRepositoryType 'TfsGit', got '%s'", TfsGitRepositoryType)
	}
}

func TestClientConfig_ValidateEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "valid HTTPS URL",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/myorg",
				PersonalAccessToken: "token",
			},
			wantErr: false,
		},
		{
			name: "URL with trailing slash",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/myorg/",
				PersonalAccessToken: "token",
			},
			wantErr: false,
		},
		{
			name: "whitespace only URL",
			config: ClientConfig{
				OrganizationURL:     "   ",
				PersonalAccessToken: "token",
			},
			wantErr: false, // Validation doesn't trim whitespace
		},
		{
			name: "whitespace only PAT",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/myorg",
				PersonalAccessToken: "   ",
			},
			wantErr: false, // Validation doesn't trim whitespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientConfig_ValidateAllScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		orgURL         string
		pat            string
		wantErr        bool
		wantErrContain string
	}{
		{
			name:    "valid config",
			orgURL:  "https://dev.azure.com/contoso",
			pat:     "valid-pat-token-12345",
			wantErr: false,
		},
		{
			name:           "empty org URL",
			orgURL:         "",
			pat:            "valid-token",
			wantErr:        true,
			wantErrContain: "organization URL is required",
		},
		{
			name:           "empty PAT",
			orgURL:         "https://dev.azure.com/contoso",
			pat:            "",
			wantErr:        true,
			wantErrContain: "personal access token is required",
		},
		{
			name:           "both empty",
			orgURL:         "",
			pat:            "",
			wantErr:        true,
			wantErrContain: "organization URL is required",
		},
		{
			name:    "on-premises URL",
			orgURL:  "https://tfs.company.com/tfs/DefaultCollection",
			pat:     "token",
			wantErr: false,
		},
		{
			name:    "visualstudio.com URL",
			orgURL:  "https://contoso.visualstudio.com",
			pat:     "token",
			wantErr: false,
		},
		{
			name:    "very long PAT",
			orgURL:  "https://dev.azure.com/org",
			pat:     "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := ClientConfig{
				OrganizationURL:     tt.orgURL,
				PersonalAccessToken: tt.pat,
			}

			err := config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.wantErrContain)
					return
				}
				if tt.wantErrContain != "" && !containsString(err.Error(), tt.wantErrContain) {
					t.Errorf("Validate() error = %q, should contain %q", err.Error(), tt.wantErrContain)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestClientConfig_WithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
		Logger:              logger,
	}

	if config.Logger != logger {
		t.Error("Logger not set correctly in config")
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestConstants(t *testing.T) {
	t.Run("TfsGitRepositoryType value", func(t *testing.T) {
		expected := "TfsGit"
		if TfsGitRepositoryType != expected {
			t.Errorf("TfsGitRepositoryType = %q, want %q", TfsGitRepositoryType, expected)
		}
	})
}

func TestNewClient_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "empty organization URL fails",
			config: ClientConfig{
				OrganizationURL:     "",
				PersonalAccessToken: "token",
			},
			wantErr: true,
		},
		{
			name: "empty PAT fails",
			config: ClientConfig{
				OrganizationURL:     "https://dev.azure.com/org",
				PersonalAccessToken: "",
			},
			wantErr: true,
		},
		{
			name: "both empty fails",
			config: ClientConfig{
				OrganizationURL:     "",
				PersonalAccessToken: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewClient_WithNilLogger(t *testing.T) {
	// Test that NewClient works with a nil logger (should use default)
	// This test validates the config but skips actual client creation
	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
		Logger:              nil,
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

func TestNewClient_WithCustomLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
		Logger:              logger,
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

// TestClient_Methods_Skipped provides documentation that these methods
// require network access and are tested via integration tests
func TestClient_Methods_Skipped(t *testing.T) {
	methodsRequiringNetwork := []string{
		"GetProjects",
		"GetProject",
		"GetRepositories",
		"GetRepository",
		"GetBranches",
		"GetPullRequests",
		"GetCommitCount",
		"IsGitRepo",
		"HasAzureBoards",
		"HasAzurePipelines",
		"GetBranchPolicies",
		"GetWorkItemsLinkedToRepo",
		"HasGHAS",
		"ValidateCredentials",
		"GetPipelineDefinitions",
		"GetPipelineRuns",
		"GetServiceConnections",
		"GetVariableGroups",
		"GetWikiDetails",
		"GetTestPlans",
		"GetServiceHooks",
		"GetPackageFeeds",
		"GetPRDetails",
		"GetWorkItemDetails",
		"GetBranchPolicyDetails",
	}

	t.Logf("The following %d methods require network access and are tested via integration tests:", len(methodsRequiringNetwork))
	for _, method := range methodsRequiringNetwork {
		t.Logf("  - %s", method)
	}
}

// Benchmark tests
func BenchmarkClientConfig_Validate(b *testing.B) {
	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token-12345",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

// Test that context is properly handled
func TestClient_ContextHandling(t *testing.T) {
	t.Skip("Skipping - requires network access")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := ClientConfig{
		OrganizationURL:     "https://dev.azure.com/testorg",
		PersonalAccessToken: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// These calls should fail due to cancelled context
	_, err = client.GetProjects(ctx)
	if err == nil {
		t.Error("Expected error from cancelled context")
	}
}
