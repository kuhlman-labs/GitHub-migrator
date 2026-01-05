package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/stretchr/testify/require"
)

// createTestSource creates a minimal source for testing
func createTestSource(name string, sourceType string) *models.Source {
	org := "test-org"
	source := &models.Source{
		Name:     name,
		Type:     sourceType,
		BaseURL:  "https://api.github.com",
		Token:    "ghp_test_token_12345678901234567890",
		IsActive: true,
	}
	if sourceType == models.SourceConfigTypeAzureDevOps {
		source.BaseURL = "https://dev.azure.com/test-org"
		source.Organization = &org
	}
	return source
}

func TestCreateSource(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		source    *models.Source
		expectErr bool
	}{
		{
			name:      "create github source",
			source:    createTestSource("GHES Production", models.SourceConfigTypeGitHub),
			expectErr: false,
		},
		{
			name:      "create azure devops source",
			source:    createTestSource("ADO Main", models.SourceConfigTypeAzureDevOps),
			expectErr: false,
		},
		{
			name: "fail on missing name",
			source: &models.Source{
				Type:    models.SourceConfigTypeGitHub,
				BaseURL: "https://api.github.com",
				Token:   "test-token",
			},
			expectErr: true,
		},
		{
			name: "fail on missing token",
			source: &models.Source{
				Name:    "Test Source",
				Type:    models.SourceConfigTypeGitHub,
				BaseURL: "https://api.github.com",
			},
			expectErr: true,
		},
		{
			name: "fail on invalid type",
			source: &models.Source{
				Name:    "Invalid Type Source",
				Type:    "invalid",
				BaseURL: "https://api.github.com",
				Token:   "test-token",
			},
			expectErr: true,
		},
		{
			name: "fail on ado without org",
			source: &models.Source{
				Name:    "ADO Without Org",
				Type:    models.SourceConfigTypeAzureDevOps,
				BaseURL: "https://dev.azure.com/test",
				Token:   "test-token",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.CreateSource(ctx, tt.source)
			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectErr && tt.source.ID == 0 {
				t.Error("expected source ID to be set after creation")
			}
		})
	}
}

func TestCreateSourceDuplicateName(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source1 := createTestSource("Duplicate Name", models.SourceConfigTypeGitHub)
	err := db.CreateSource(ctx, source1)
	if err != nil {
		t.Fatalf("failed to create first source: %v", err)
	}

	source2 := createTestSource("Duplicate Name", models.SourceConfigTypeGitHub)
	err = db.CreateSource(ctx, source2)
	if err == nil {
		t.Error("expected error for duplicate name but got nil")
	}
}

func TestGetSource(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source := createTestSource("Get Test Source", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Test get by ID
	retrieved, err := db.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatalf("failed to get source: %v", err)
	}
	require.NotNil(t, retrieved, "expected source but got nil")
	if retrieved.Name != source.Name {
		t.Errorf("name mismatch: got %q, want %q", retrieved.Name, source.Name)
	}
	if retrieved.Type != source.Type {
		t.Errorf("type mismatch: got %q, want %q", retrieved.Type, source.Type)
	}

	// Test get non-existent
	notFound, err := db.GetSource(ctx, 99999)
	if err != nil {
		t.Fatalf("unexpected error getting non-existent source: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent source")
	}
}

func TestGetSourceByName(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source := createTestSource("Named Source", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Test get by name
	retrieved, err := db.GetSourceByName(ctx, "Named Source")
	if err != nil {
		t.Fatalf("failed to get source by name: %v", err)
	}
	require.NotNil(t, retrieved, "expected source but got nil")
	if retrieved.ID != source.ID {
		t.Errorf("ID mismatch: got %d, want %d", retrieved.ID, source.ID)
	}

	// Test get non-existent name
	notFound, err := db.GetSourceByName(ctx, "Non-existent Source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent source name")
	}
}

func TestListSources(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	// Create multiple sources
	sources := []*models.Source{
		createTestSource("Alpha Source", models.SourceConfigTypeGitHub),
		createTestSource("Beta Source", models.SourceConfigTypeAzureDevOps),
		createTestSource("Gamma Source", models.SourceConfigTypeGitHub),
	}

	for _, s := range sources {
		if err := db.CreateSource(ctx, s); err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
	}

	// List all sources
	list, err := db.ListSources(ctx)
	if err != nil {
		t.Fatalf("failed to list sources: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 sources, got %d", len(list))
	}

	// Check alphabetical order
	if list[0].Name != "Alpha Source" {
		t.Errorf("expected first source to be 'Alpha Source', got %q", list[0].Name)
	}
}

func TestListActiveSources(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	// Create active and inactive sources
	active := createTestSource("Active Source", models.SourceConfigTypeGitHub)
	inactive := createTestSource("Inactive Source", models.SourceConfigTypeGitHub)

	if err := db.CreateSource(ctx, active); err != nil {
		t.Fatalf("failed to create active source: %v", err)
	}
	if err := db.CreateSource(ctx, inactive); err != nil {
		t.Fatalf("failed to create inactive source: %v", err)
	}

	// Set inactive source to inactive using SetSourceActive (to bypass GORM default)
	if err := db.SetSourceActive(ctx, inactive.ID, false); err != nil {
		t.Fatalf("failed to set source inactive: %v", err)
	}

	// List only active sources
	list, err := db.ListActiveSources(ctx)
	if err != nil {
		t.Fatalf("failed to list active sources: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 active source, got %d", len(list))
	}
	if list[0].Name != "Active Source" {
		t.Errorf("expected 'Active Source', got %q", list[0].Name)
	}
}

func TestUpdateSource(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source := createTestSource("Update Test", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Update the source
	source.Name = "Updated Name"
	source.BaseURL = "https://github.example.com/api/v3"
	source.Token = "new_token_value"

	if err := db.UpdateSource(ctx, source); err != nil {
		t.Fatalf("failed to update source: %v", err)
	}

	// Verify update
	retrieved, err := db.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatalf("failed to get updated source: %v", err)
	}
	if retrieved.Name != "Updated Name" {
		t.Errorf("name not updated: got %q", retrieved.Name)
	}
	if retrieved.BaseURL != "https://github.example.com/api/v3" {
		t.Errorf("base URL not updated: got %q", retrieved.BaseURL)
	}
}

func TestDeleteSource(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source := createTestSource("Delete Test", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Delete the source
	if err := db.DeleteSource(ctx, source.ID); err != nil {
		t.Fatalf("failed to delete source: %v", err)
	}

	// Verify deletion
	retrieved, err := db.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved != nil {
		t.Error("expected source to be deleted")
	}

	// Test delete non-existent
	err = db.DeleteSource(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent source")
	}
}

func TestDeleteSourceWithRepositories(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	// Create source
	source := createTestSource("Source With Repos", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Create repository and assign to source
	repo := createTestRepository("org/repo-with-source")
	repo.SourceID = &source.ID
	repo.Status = string(models.StatusPending)
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Try to delete source - should fail
	err := db.DeleteSource(ctx, source.ID)
	if err == nil {
		t.Error("expected error when deleting source with repositories")
	}
}

func TestSetSourceActive(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source := createTestSource("Active Toggle Test", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Deactivate
	if err := db.SetSourceActive(ctx, source.ID, false); err != nil {
		t.Fatalf("failed to deactivate source: %v", err)
	}

	retrieved, _ := db.GetSource(ctx, source.ID)
	if retrieved.IsActive {
		t.Error("expected source to be inactive")
	}

	// Reactivate
	if err := db.SetSourceActive(ctx, source.ID, true); err != nil {
		t.Fatalf("failed to reactivate source: %v", err)
	}

	retrieved, _ = db.GetSource(ctx, source.ID)
	if !retrieved.IsActive {
		t.Error("expected source to be active")
	}
}

func TestUpdateSourceLastSync(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source := createTestSource("Sync Test", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Initially LastSyncAt should be nil
	retrieved, _ := db.GetSource(ctx, source.ID)
	if retrieved.LastSyncAt != nil {
		t.Error("expected LastSyncAt to be nil initially")
	}

	// Update last sync
	if err := db.UpdateSourceLastSync(ctx, source.ID); err != nil {
		t.Fatalf("failed to update last sync: %v", err)
	}

	retrieved, _ = db.GetSource(ctx, source.ID)
	if retrieved.LastSyncAt == nil {
		t.Error("expected LastSyncAt to be set")
	}
}

func TestUpdateSourceRepositoryCount(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	source := createTestSource("Repo Count Test", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Create repositories and assign to source
	for i := 0; i < 3; i++ {
		repo := createTestRepository(fmt.Sprintf("org/repo-%d", i))
		repo.SourceID = &source.ID
		repo.Status = string(models.StatusPending)
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}
	}

	// Update count
	if err := db.UpdateSourceRepositoryCount(ctx, source.ID); err != nil {
		t.Fatalf("failed to update repository count: %v", err)
	}

	retrieved, _ := db.GetSource(ctx, source.ID)
	if retrieved.RepositoryCount != 3 {
		t.Errorf("expected repository count 3, got %d", retrieved.RepositoryCount)
	}
}

func TestGetRepositoriesBySourceID(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	// Create two sources
	source1 := createTestSource("Source One", models.SourceConfigTypeGitHub)
	source2 := createTestSource("Source Two", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source1); err != nil {
		t.Fatalf("failed to create source1: %v", err)
	}
	if err := db.CreateSource(ctx, source2); err != nil {
		t.Fatalf("failed to create source2: %v", err)
	}

	// Create repos for source1
	for i := 0; i < 2; i++ {
		repo := createTestRepository(fmt.Sprintf("org1/repo-%d", i))
		repo.SourceID = &source1.ID
		repo.Status = string(models.StatusPending)
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("failed to create repository: %v", err)
		}
	}

	// Create repos for source2
	repo := createTestRepository("org2/repo-0")
	repo.SourceID = &source2.ID
	repo.Status = string(models.StatusPending)
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Get repos for source1
	repos, err := db.GetRepositoriesBySourceID(ctx, source1.ID)
	if err != nil {
		t.Fatalf("failed to get repositories: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repositories for source1, got %d", len(repos))
	}

	// Get repos for source2
	repos, err = db.GetRepositoriesBySourceID(ctx, source2.ID)
	if err != nil {
		t.Fatalf("failed to get repositories: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repository for source2, got %d", len(repos))
	}
}

func TestGetSourcesByType(t *testing.T) {
	db := setupSourcesTestDB(t)
	ctx := context.Background()

	// Create sources of different types
	github1 := createTestSource("GitHub One", models.SourceConfigTypeGitHub)
	github2 := createTestSource("GitHub Two", models.SourceConfigTypeGitHub)
	ado := createTestSource("ADO One", models.SourceConfigTypeAzureDevOps)

	for _, s := range []*models.Source{github1, github2, ado} {
		if err := db.CreateSource(ctx, s); err != nil {
			t.Fatalf("failed to create source: %v", err)
		}
	}

	// Get GitHub sources
	githubSources, err := db.GetSourcesByType(ctx, models.SourceConfigTypeGitHub)
	if err != nil {
		t.Fatalf("failed to get GitHub sources: %v", err)
	}
	if len(githubSources) != 2 {
		t.Errorf("expected 2 GitHub sources, got %d", len(githubSources))
	}

	// Get ADO sources
	adoSources, err := db.GetSourcesByType(ctx, models.SourceConfigTypeAzureDevOps)
	if err != nil {
		t.Fatalf("failed to get ADO sources: %v", err)
	}
	if len(adoSources) != 1 {
		t.Errorf("expected 1 ADO source, got %d", len(adoSources))
	}
}

func TestSourceModel(t *testing.T) {
	t.Run("IsGitHub", func(t *testing.T) {
		source := &models.Source{Type: models.SourceConfigTypeGitHub}
		if !source.IsGitHub() {
			t.Error("expected IsGitHub to return true")
		}
		if source.IsAzureDevOps() {
			t.Error("expected IsAzureDevOps to return false")
		}
	})

	t.Run("IsAzureDevOps", func(t *testing.T) {
		source := &models.Source{Type: models.SourceConfigTypeAzureDevOps}
		if !source.IsAzureDevOps() {
			t.Error("expected IsAzureDevOps to return true")
		}
		if source.IsGitHub() {
			t.Error("expected IsGitHub to return false")
		}
	})

	t.Run("HasAppAuth", func(t *testing.T) {
		appID := int64(12345)
		privateKey := "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----"

		source := &models.Source{}
		if source.HasAppAuth() {
			t.Error("expected HasAppAuth to return false without credentials")
		}

		source.AppID = &appID
		source.AppPrivateKey = &privateKey
		if !source.HasAppAuth() {
			t.Error("expected HasAppAuth to return true with credentials")
		}
	})

	t.Run("MaskedToken", func(t *testing.T) {
		source := &models.Source{Token: "ghp_abcd1234efgh5678ijkl"}
		masked := source.MaskedToken()
		if masked != "ghp_...ijkl" {
			t.Errorf("expected 'ghp_...ijkl', got %q", masked)
		}

		// Short token
		source.Token = "short"
		masked = source.MaskedToken()
		if masked != "****" {
			t.Errorf("expected '****' for short token, got %q", masked)
		}
	})

	t.Run("ToResponse", func(t *testing.T) {
		source := createTestSource("Response Test", models.SourceConfigTypeGitHub)
		source.ID = 42
		source.Token = "secret_token_12345678"

		resp := source.ToResponse()
		if resp.ID != 42 {
			t.Errorf("expected ID 42, got %d", resp.ID)
		}
		if resp.Name != "Response Test" {
			t.Errorf("expected name 'Response Test', got %q", resp.Name)
		}
		if resp.MaskedToken == source.Token {
			t.Error("token should be masked in response")
		}
	})
}

func setupSourcesTestDB(t *testing.T) *Database {
	t.Helper()
	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
