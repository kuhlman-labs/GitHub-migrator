package discovery

import (
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestRepoDiscoverer_FilterByStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := &RepoDiscoverer{logger: logger}

	repos := []*models.Repository{
		{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
		{ID: 2, FullName: "org/repo2", Status: string(models.StatusComplete)},
		{ID: 3, FullName: "org/repo3", Status: string(models.StatusPending)},
		{ID: 4, FullName: "org/repo4", Status: string(models.StatusMigrationComplete)},
		{ID: 5, FullName: "org/repo5", Status: string(models.StatusMigrationFailed)},
	}

	tests := []struct {
		name     string
		statuses []models.MigrationStatus
		expected int
	}{
		{
			name:     "single status",
			statuses: []models.MigrationStatus{models.StatusPending},
			expected: 2,
		},
		{
			name:     "multiple statuses",
			statuses: []models.MigrationStatus{models.StatusPending, models.StatusComplete},
			expected: 3,
		},
		{
			name:     "no matching status",
			statuses: []models.MigrationStatus{"nonexistent"},
			expected: 0,
		},
		{
			name:     "all matching",
			statuses: []models.MigrationStatus{models.StatusPending, models.StatusComplete, models.StatusMigrationComplete, models.StatusMigrationFailed},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.FilterByStatus(repos, tt.statuses...)
			if len(result) != tt.expected {
				t.Errorf("Expected %d repos, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestRepoDiscoverer_FilterEligibleForMigration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := &RepoDiscoverer{logger: logger}

	// Create repos with different states
	// Note: CanBeMigrated() checks status and HasMigrationBlockers()
	repos := []*models.Repository{
		// Eligible: status that allows migration
		{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
		// Not eligible: already migrated
		{ID: 2, FullName: "org/repo2", Status: string(models.StatusComplete)},
		// Not eligible: wont migrate
		{ID: 3, FullName: "org/repo3", Status: string(models.StatusWontMigrate)},
	}

	result := d.FilterEligibleForMigration(repos)

	// Check that all returned repos pass CanBeMigrated
	for _, repo := range result {
		if !repo.CanBeMigrated() {
			t.Errorf("Returned repo %s should be eligible for migration", repo.FullName)
		}
	}
}

func TestRepoDiscoverer_FilterEligibleForBatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := &RepoDiscoverer{logger: logger}

	batchID := int64(1)
	repos := []*models.Repository{
		// Eligible: pending status, no batch
		{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending), BatchID: nil},
		// Not eligible: already in batch
		{ID: 2, FullName: "org/repo2", Status: string(models.StatusPending), BatchID: &batchID},
		// Not eligible: wont migrate status
		{ID: 3, FullName: "org/repo3", Status: string(models.StatusWontMigrate), BatchID: nil},
	}

	result := d.FilterEligibleForBatch(repos)

	// Check that all returned repos pass CanBeAssignedToBatch
	for _, repo := range result {
		ok, _ := repo.CanBeAssignedToBatch()
		if !ok {
			t.Errorf("Returned repo %s should be eligible for batch", repo.FullName)
		}
	}
}

func TestRepoDiscoverer_GroupByOrganization(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := &RepoDiscoverer{logger: logger}

	repos := []*models.Repository{
		{ID: 1, FullName: "org1/repo1"},
		{ID: 2, FullName: "org1/repo2"},
		{ID: 3, FullName: "org2/repo1"},
		{ID: 4, FullName: "org2/repo2"},
		{ID: 5, FullName: "org2/repo3"},
		{ID: 6, FullName: "org3/repo1"},
	}

	result := d.GroupByOrganization(repos)

	if len(result) != 3 {
		t.Errorf("Expected 3 organizations, got %d", len(result))
	}

	if len(result["org1"]) != 2 {
		t.Errorf("Expected 2 repos in org1, got %d", len(result["org1"]))
	}

	if len(result["org2"]) != 3 {
		t.Errorf("Expected 3 repos in org2, got %d", len(result["org2"]))
	}

	if len(result["org3"]) != 1 {
		t.Errorf("Expected 1 repo in org3, got %d", len(result["org3"]))
	}
}

func TestRepoDiscoverer_GroupByStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := &RepoDiscoverer{logger: logger}

	repos := []*models.Repository{
		{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
		{ID: 2, FullName: "org/repo2", Status: string(models.StatusPending)},
		{ID: 3, FullName: "org/repo3", Status: string(models.StatusComplete)},
		{ID: 4, FullName: "org/repo4", Status: string(models.StatusMigrationComplete)},
	}

	result := d.GroupByStatus(repos)

	if len(result) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(result))
	}

	if len(result[string(models.StatusPending)]) != 2 {
		t.Errorf("Expected 2 pending repos, got %d", len(result[string(models.StatusPending)]))
	}

	if len(result[string(models.StatusComplete)]) != 1 {
		t.Errorf("Expected 1 complete repo, got %d", len(result[string(models.StatusComplete)]))
	}

	if len(result[string(models.StatusMigrationComplete)]) != 1 {
		t.Errorf("Expected 1 migration_complete repo, got %d", len(result[string(models.StatusMigrationComplete)]))
	}
}

func TestRepoDiscoverer_GetRepositoryStats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := &RepoDiscoverer{logger: logger}

	size100MB := int64(100 * 1024 * 1024)
	size50MB := int64(50 * 1024 * 1024)

	repos := []*models.Repository{
		{
			ID:            1,
			FullName:      "org/repo1",
			Status:        string(models.StatusPending),
			TotalSize:     &size100MB,
			HasLFS:        true,
			HasSubmodules: false,
		},
		{
			ID:            2,
			FullName:      "org/repo2",
			Status:        string(models.StatusPending),
			TotalSize:     &size50MB,
			HasLFS:        false,
			HasSubmodules: true,
		},
		{
			ID:            3,
			FullName:      "org/repo3",
			Status:        string(models.StatusMigrationComplete),
			TotalSize:     nil, // No size
			HasLFS:        true,
			HasSubmodules: true,
		},
	}

	stats := d.GetRepositoryStats(repos)

	if stats.Total != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total)
	}

	if stats.TotalSizeBytes != 150*1024*1024 {
		t.Errorf("Expected total size 150MB, got %d", stats.TotalSizeBytes)
	}

	if stats.WithLFS != 2 {
		t.Errorf("Expected 2 repos with LFS, got %d", stats.WithLFS)
	}

	if stats.WithSubmodules != 2 {
		t.Errorf("Expected 2 repos with submodules, got %d", stats.WithSubmodules)
	}

	if stats.StatusCounts[string(models.StatusPending)] != 2 {
		t.Errorf("Expected 2 pending repos in status counts, got %d", stats.StatusCounts[string(models.StatusPending)])
	}
}

func TestNewRepoDiscoverer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with nil values - should not panic
	d := NewRepoDiscoverer(nil, nil, logger)

	if d == nil {
		t.Fatal("Expected non-nil RepoDiscoverer")
		return
	}

	if d.logger != logger {
		t.Error("Expected logger to be set")
	}
}

func TestRepoDiscoverer_WithBaseConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := NewRepoDiscoverer(nil, nil, logger)

	// Create a config with an App ID
	d2 := d.WithBaseConfig(github.ClientConfig{
		AppID:             12345,
		AppInstallationID: 0,
	})

	// Should return same instance (fluent interface)
	if d2 != d {
		t.Error("WithBaseConfig should return same instance")
	}

	if d.baseConfig == nil {
		t.Error("baseConfig should be set")
	}

	if d.baseConfig.AppID != 12345 {
		t.Errorf("Expected AppID 12345, got %d", d.baseConfig.AppID)
	}
}

func TestRepositoryStats_ZeroValues(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	d := &RepoDiscoverer{logger: logger}

	// Empty slice
	stats := d.GetRepositoryStats([]*models.Repository{})

	if stats.Total != 0 {
		t.Errorf("Expected total 0, got %d", stats.Total)
	}

	if stats.TotalSizeBytes != 0 {
		t.Errorf("Expected total size 0, got %d", stats.TotalSizeBytes)
	}

	if stats.WithLFS != 0 {
		t.Errorf("Expected 0 repos with LFS, got %d", stats.WithLFS)
	}
}
