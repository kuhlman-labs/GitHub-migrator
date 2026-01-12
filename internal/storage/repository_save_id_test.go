package storage

import (
	"context"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// TestSaveRepository_SetsIDOnInsert verifies that SaveRepository sets the ID after INSERT
func TestSaveRepository_SetsIDOnInsert(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create a new repository (INSERT case)
	repo := &models.Repository{
		FullName:  "test-org/new-repo",
		Source:    "github",
		SourceURL: "https://github.com/test-org/new-repo",
		Status:    string(models.StatusPending),
	}

	// Before save, ID should be 0
	if repo.ID != 0 {
		t.Fatalf("Expected repo.ID to be 0 before save, got %d", repo.ID)
	}

	// Save the repository (INSERT)
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// After save, ID should be set
	if repo.ID == 0 {
		t.Fatalf("Expected repo.ID to be set after INSERT, but it's still 0")
	}

	t.Logf("✅ INSERT: repo.ID correctly set to %d", repo.ID)
}

// TestSaveRepository_SetsIDOnUpdate verifies that SaveRepository sets the ID after UPDATE
func TestSaveRepository_SetsIDOnUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// First, insert a repository
	repo1 := &models.Repository{
		FullName:  "test-org/existing-repo",
		Source:    "github",
		SourceURL: "https://github.com/test-org/existing-repo",
		Status:    string(models.StatusPending),
	}

	if err := db.SaveRepository(ctx, repo1); err != nil {
		t.Fatalf("Failed to save initial repository: %v", err)
	}

	originalID := repo1.ID
	if originalID == 0 {
		t.Fatalf("Expected repo.ID to be set after initial save, but it's 0")
	}

	// Now update the same repository (UPDATE case)
	repo2 := &models.Repository{
		FullName:  "test-org/existing-repo", // Same full_name triggers UPDATE
		Source:    "github",
		SourceURL: "https://github.com/test-org/existing-repo-updated",
		Status:    string(models.StatusComplete),
	}

	// Before update, ID is 0 (new struct)
	if repo2.ID != 0 {
		t.Fatalf("Expected repo2.ID to be 0 before update, got %d", repo2.ID)
	}

	// Save the repository (UPDATE due to conflict on full_name)
	if err := db.SaveRepository(ctx, repo2); err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}

	// After update, ID should be set to the same value
	if repo2.ID == 0 {
		t.Fatalf("Expected repo.ID to be set after UPDATE, but it's still 0")
	}

	if repo2.ID != originalID {
		t.Fatalf("Expected repo.ID to remain %d after UPDATE, but got %d", originalID, repo2.ID)
	}

	t.Logf("✅ UPDATE: repo.ID correctly set to %d (matches original)", repo2.ID)
}

// TestSaveRepository_IDUsableForDependencies verifies the ID can be used immediately after save
func TestSaveRepository_IDUsableForDependencies(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Save a new repository
	repo := &models.Repository{
		FullName:  "test-org/repo-with-deps",
		Source:    "github",
		SourceURL: "https://github.com/test-org/repo-with-deps",
		Status:    string(models.StatusPending),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Verify we can immediately use the ID for dependencies
	if repo.ID == 0 {
		t.Fatalf("Cannot use repo.ID for dependencies - it's 0")
	}

	// Create a dependency using the repo.ID
	deps := []*models.RepositoryDependency{
		{
			RepositoryID:       repo.ID,
			DependencyFullName: "external-org/external-repo",
			DependencyType:     models.DependencyTypeSubmodule,
			DependencyURL:      "https://github.com/external-org/external-repo",
			IsLocal:            false,
		},
	}

	// Save dependencies
	if err := db.SaveRepositoryDependencies(ctx, repo.ID, deps); err != nil {
		t.Fatalf("Failed to save dependencies: %v", err)
	}

	// Retrieve and verify
	savedDeps, err := db.GetRepositoryDependencies(ctx, repo.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve dependencies: %v", err)
	}

	if len(savedDeps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(savedDeps))
	}

	if savedDeps[0].RepositoryID != repo.ID {
		t.Fatalf("Expected dependency.RepositoryID = %d, got %d", repo.ID, savedDeps[0].RepositoryID)
	}

	t.Logf("✅ Dependencies: Successfully saved and retrieved with repo.ID = %d", repo.ID)
}
