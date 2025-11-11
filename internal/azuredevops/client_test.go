package azuredevops

import (
	"context"
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
