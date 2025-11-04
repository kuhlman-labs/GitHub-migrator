package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/config"
)

func TestRepositoryDependenciesTableExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Verify repository_dependencies table exists
	var tableName string
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='repository_dependencies'"
	err = db.db.QueryRow(query).Scan(&tableName)
	if err != nil {
		t.Fatalf("repository_dependencies table does not exist: %v", err)
	}

	// Verify all indexes exist
	expectedIndexes := []string{
		"idx_repository_dependencies_repo_id",
		"idx_repository_dependencies_dep_name",
		"idx_repository_dependencies_type",
		"idx_repository_dependencies_is_local",
		"idx_repository_dependencies_local_type",
	}

	for _, indexName := range expectedIndexes {
		var name string
		query := "SELECT name FROM sqlite_master WHERE type='index' AND name=?"
		err := db.db.QueryRow(query, indexName).Scan(&name)
		if err != nil {
			t.Errorf("Index %s does not exist: %v", indexName, err)
		}
	}

	// Verify we can insert and query data
	ctx := context.Background()

	// Create a test repository first
	_, err = db.db.ExecContext(ctx, `
		INSERT INTO repositories (full_name, source, source_url, status, discovered_at, updated_at)
		VALUES ('test-org/test-repo', 'github', 'https://github.com/test-org/test-repo', 'discovered', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Get the repo ID
	var repoID int64
	err = db.db.QueryRowContext(ctx, "SELECT id FROM repositories WHERE full_name = 'test-org/test-repo'").Scan(&repoID)
	if err != nil {
		t.Fatalf("Failed to get repository ID: %v", err)
	}

	// Insert a test dependency
	_, err = db.db.ExecContext(ctx, `
		INSERT INTO repository_dependencies 
		(repository_id, dependency_full_name, dependency_type, dependency_url, is_local)
		VALUES (?, ?, ?, ?, ?)
	`, repoID, "test-org/dependency-repo", "submodule", "https://github.com/test-org/dependency-repo", true)
	if err != nil {
		t.Fatalf("Failed to insert test dependency: %v", err)
	}

	// Query the dependency back
	var count int
	err = db.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM repository_dependencies WHERE repository_id = ?",
		repoID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query dependencies: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 dependency, got %d", count)
	}

	t.Log("✅ repository_dependencies table created successfully with all indexes")
}

func TestGetRepositoryDependencies_EmptyResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	ctx := context.Background()

	// Create a test repository
	_, err = db.db.ExecContext(ctx, `
		INSERT INTO repositories (full_name, source, source_url, status, discovered_at, updated_at)
		VALUES ('test-org/test-repo', 'github', 'https://github.com/test-org/test-repo', 'discovered', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	var repoID int64
	err = db.db.QueryRowContext(ctx, "SELECT id FROM repositories WHERE full_name = 'test-org/test-repo'").Scan(&repoID)
	if err != nil {
		t.Fatalf("Failed to get repository ID: %v", err)
	}

	// Get dependencies for a repo with no dependencies
	dependencies, err := db.GetRepositoryDependencies(ctx, repoID)
	if err != nil {
		t.Fatalf("GetRepositoryDependencies() error = %v", err)
	}

	// Should return empty slice, not nil
	if dependencies == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(dependencies) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(dependencies))
	}

	t.Log("✅ GetRepositoryDependencies returns empty slice (not nil) when no dependencies exist")
}
