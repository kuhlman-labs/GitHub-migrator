// Package handlers provides HTTP request handlers for the API.
//
// # Service Interfaces
//
// This file defines service interfaces that can be used for dependency injection
// and testing. While the current implementation directly uses storage.Database,
// these interfaces establish contracts that enable:
//
//   - Unit testing with mock implementations
//   - Future migration to different storage backends
//   - Clear separation of concerns between HTTP handling and data access
//
// Usage in tests:
//
//	type mockRepositoryService struct {
//	    repos map[string]*models.Repository
//	}
//
//	func (m *mockRepositoryService) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
//	    if repo, ok := m.repos[fullName]; ok {
//	        return repo, nil
//	    }
//	    return nil, nil
//	}
package handlers

import (
	"context"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// RepositoryReader defines read operations for repositories.
// Use this interface when you only need to read repository data.
type RepositoryReader interface {
	GetRepository(ctx context.Context, fullName string) (*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error)
}

// RepositoryWriter defines write operations for repositories.
// Use this interface when you need to modify repository data.
type RepositoryWriter interface {
	SaveRepository(ctx context.Context, repo *models.Repository) error
	UpdateRepository(ctx context.Context, repo *models.Repository) error
	DeleteRepository(ctx context.Context, fullName string) error
}

// BatchReader defines read operations for batches.
type BatchReader interface {
	GetBatch(ctx context.Context, id int64) (*models.Batch, error)
	ListBatches(ctx context.Context) ([]*models.Batch, error)
}

// BatchWriter defines write operations for batches.
type BatchWriter interface {
	CreateBatch(ctx context.Context, batch *models.Batch) error
	UpdateBatch(ctx context.Context, batch *models.Batch) error
	DeleteBatch(ctx context.Context, id int64) error
}

// MigrationHistoryReader defines read operations for migration history.
type MigrationHistoryReader interface {
	GetMigrationHistory(ctx context.Context, repoID int64) ([]*models.MigrationHistory, error)
	GetMigrationLogs(ctx context.Context, historyID int64) ([]*models.MigrationLog, error)
}

// MigrationHistoryWriter defines write operations for migration history.
type MigrationHistoryWriter interface {
	CreateMigrationHistory(ctx context.Context, history *models.MigrationHistory) (int64, error)
	UpdateMigrationHistory(ctx context.Context, id int64, status string, errorMsg *string) error
	CreateMigrationLog(ctx context.Context, log *models.MigrationLog) error
}

// DependencyReader defines read operations for dependencies.
type DependencyReader interface {
	GetDependencies(ctx context.Context, repoID int64) ([]*models.RepositoryDependency, error)
	GetDependents(ctx context.Context, repoFullName string) ([]*models.RepositoryDependency, error)
}

// DiscoveryProgressTracker defines operations for tracking discovery progress.
type DiscoveryProgressTracker interface {
	CreateDiscoveryProgress(ctx context.Context, progress *models.DiscoveryProgress) (int64, error)
	UpdateDiscoveryProgress(ctx context.Context, progress *models.DiscoveryProgress) error
	GetDiscoveryProgress(ctx context.Context, id int64) (*models.DiscoveryProgress, error)
	GetActiveDiscoveryProgress(ctx context.Context) (*models.DiscoveryProgress, error)
}

// HealthChecker defines the interface for health check operations.
type HealthChecker interface {
	Ping() error
}

// TimeProvider is an interface for getting current time.
// Useful for testing time-dependent logic.
type TimeProvider interface {
	Now() time.Time
}

// RealTimeProvider implements TimeProvider with real system time.
type RealTimeProvider struct{}

// Now returns the current time.
func (RealTimeProvider) Now() time.Time {
	return time.Now()
}

// MockTimeProvider implements TimeProvider with a fixed time for testing.
type MockTimeProvider struct {
	FixedTime time.Time
}

// Now returns the fixed time.
func (m MockTimeProvider) Now() time.Time {
	return m.FixedTime
}
