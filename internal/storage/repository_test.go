package storage

import (
	"context"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

const testDefaultBranch = "main"

func setupTestDB(t *testing.T) *Database {
	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func TestSaveRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch
	topContrib := "user1,user2"

	repo := &models.Repository{
		FullName:         "test/repo",
		Source:           "ghes",
		SourceURL:        "https://github.com/test/repo",
		TotalSize:        &totalSize,
		DefaultBranch:    &defaultBranch,
		HasWiki:          true,
		HasPages:         false,
		HasLFS:           true,
		HasSubmodules:    false,
		BranchCount:      5,
		CommitCount:      100,
		ContributorCount: 2,
		TopContributors:  &topContrib,
		Status:           string(models.StatusPending),
		DiscoveredAt:     time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Test insert
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Test update (upsert)
	newSize := int64(2048 * 1024)
	repo.TotalSize = &newSize
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}
}

func TestGetRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	original := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, original); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Test get existing repository
	retrieved, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved repository is nil")
	}

	if retrieved.FullName != original.FullName {
		t.Errorf("Expected FullName '%s', got '%s'", original.FullName, retrieved.FullName)
	}

	// Test get non-existent repository
	notFound, err := db.GetRepository(ctx, "nonexistent/repo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if notFound != nil {
		t.Error("Expected nil for non-existent repository")
	}
}

func TestListRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	// Create test repositories
	repos := []*models.Repository{
		{
			FullName:      "org/repo1",
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo1",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusPending),
			HasLFS:        true,
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			FullName:      "org/repo2",
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo2",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(models.StatusComplete),
			HasLFS:        false,
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	for _, repo := range repos {
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	tests := []struct {
		name     string
		filters  map[string]interface{}
		expected int
	}{
		{
			name:     "No filters",
			filters:  nil,
			expected: 2,
		},
		{
			name:     "Filter by status",
			filters:  map[string]interface{}{"status": string(models.StatusPending)},
			expected: 1,
		},
		{
			name:     "Filter by LFS",
			filters:  map[string]interface{}{"has_lfs": true},
			expected: 1,
		},
		{
			name:     "Filter by source",
			filters:  map[string]interface{}{"source": "ghes"},
			expected: 2,
		},
		{
			name:     "Filter with limit",
			filters:  map[string]interface{}{"limit": 1},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.ListRepositories(ctx, tt.filters)
			if err != nil {
				t.Fatalf("ListRepositories failed: %v", err)
			}

			if len(results) != tt.expected {
				t.Errorf("Expected %d repositories, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestUpdateRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	repo := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Update the repository
	newSize := int64(2048 * 1024)
	repo.TotalSize = &newSize
	repo.Status = string(models.StatusComplete)
	now := time.Now()
	repo.MigratedAt = &now

	if err := db.UpdateRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}

	// Verify update
	updated, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if updated.TotalSize == nil || *updated.TotalSize != newSize {
		t.Errorf("Expected TotalSize %d, got %v", newSize, updated.TotalSize)
	}
	if updated.Status != string(models.StatusComplete) {
		t.Errorf("Expected Status 'complete', got '%s'", updated.Status)
	}
	if updated.MigratedAt == nil {
		t.Error("Expected MigratedAt to be set")
	}
}

func TestUpdateRepositoryStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	repo := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Update status
	if err := db.UpdateRepositoryStatus(ctx, "test/repo", models.StatusComplete); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify
	updated, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if updated.Status != string(models.StatusComplete) {
		t.Errorf("Expected Status 'complete', got '%s'", updated.Status)
	}
}

func TestDeleteRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	repo := &models.Repository{
		FullName:      "test/repo",
		Source:        "ghes",
		SourceURL:     "https://github.com/test/repo",
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	// Delete
	if err := db.DeleteRepository(ctx, "test/repo"); err != nil {
		t.Fatalf("Failed to delete repository: %v", err)
	}

	// Verify deletion
	deleted, err := db.GetRepository(ctx, "test/repo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if deleted != nil {
		t.Error("Expected repository to be deleted")
	}
}

func TestCountRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	// Create test repositories
	for i := 1; i <= 5; i++ {
		status := models.StatusPending
		if i > 3 {
			status = models.StatusComplete
		}

		repo := &models.Repository{
			FullName:      "org/repo" + string(rune('0'+i)),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(status),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	tests := []struct {
		name     string
		filters  map[string]interface{}
		expected int
	}{
		{
			name:     "Count all",
			filters:  nil,
			expected: 5,
		},
		{
			name:     "Count pending",
			filters:  map[string]interface{}{"status": string(models.StatusPending)},
			expected: 3,
		},
		{
			name:     "Count complete",
			filters:  map[string]interface{}{"status": string(models.StatusComplete)},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := db.CountRepositories(ctx, tt.filters)
			if err != nil {
				t.Fatalf("CountRepositories failed: %v", err)
			}

			if count != tt.expected {
				t.Errorf("Expected count %d, got %d", tt.expected, count)
			}
		})
	}
}

func TestGetRepositoryStatsByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	totalSize := int64(1024 * 1024)
	defaultBranch := testDefaultBranch

	// Create test repositories with different statuses
	statuses := []models.MigrationStatus{
		models.StatusPending,
		models.StatusPending,
		models.StatusPending,
		models.StatusComplete,
		models.StatusComplete,
		models.StatusMigrationFailed,
	}

	for i, status := range statuses {
		repo := &models.Repository{
			FullName:      "org/repo" + string(rune('0'+i)),
			Source:        "ghes",
			SourceURL:     "https://github.com/org/repo",
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
			Status:        string(status),
			DiscoveredAt:  time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	stats, err := db.GetRepositoryStatsByStatus(ctx)
	if err != nil {
		t.Fatalf("GetRepositoryStatsByStatus failed: %v", err)
	}

	if stats[string(models.StatusPending)] != 3 {
		t.Errorf("Expected 3 pending, got %d", stats[string(models.StatusPending)])
	}
	if stats[string(models.StatusComplete)] != 2 {
		t.Errorf("Expected 2 complete, got %d", stats[string(models.StatusComplete)])
	}
	if stats[string(models.StatusMigrationFailed)] != 1 {
		t.Errorf("Expected 1 failed, got %d", stats[string(models.StatusMigrationFailed)])
	}
}
