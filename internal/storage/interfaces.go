package storage

import (
	"context"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// RepositoryReader defines read operations for repositories.
// This interface enables dependency injection and easier testing.
type RepositoryReader interface {
	// GetRepository retrieves a single repository by full name.
	GetRepository(ctx context.Context, fullName string) (*models.Repository, error)
	// GetRepositoryByID retrieves a single repository by ID.
	GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error)
	// GetRepositoriesByIDs retrieves multiple repositories by their IDs.
	GetRepositoriesByIDs(ctx context.Context, ids []int64) ([]*models.Repository, error)
	// GetRepositoriesByNames retrieves multiple repositories by their full names.
	GetRepositoriesByNames(ctx context.Context, names []string) ([]*models.Repository, error)
	// ListRepositories returns repositories matching the given filters.
	ListRepositories(ctx context.Context, filters map[string]interface{}) ([]*models.Repository, error)
	// CountRepositories counts repositories matching the given filters.
	CountRepositories(ctx context.Context, filters map[string]interface{}) (int, error)
}

// RepositoryWriter defines write operations for repositories.
type RepositoryWriter interface {
	// SaveRepository creates or updates a repository.
	SaveRepository(ctx context.Context, repo *models.Repository) error
	// UpdateRepository updates an existing repository.
	UpdateRepository(ctx context.Context, repo *models.Repository) error
	// UpdateRepositoryStatus updates the status of a repository by full name.
	UpdateRepositoryStatus(ctx context.Context, fullName string, status models.MigrationStatus) error
	// DeleteRepository removes a repository by full name.
	DeleteRepository(ctx context.Context, fullName string) error
}

// RepositoryStore combines read and write operations for repositories.
type RepositoryStore interface {
	RepositoryReader
	RepositoryWriter
}

// BatchReader defines read operations for batches.
type BatchReader interface {
	// GetBatch retrieves a single batch by ID.
	GetBatch(ctx context.Context, id int64) (*models.Batch, error)
	// ListBatches returns all batches.
	ListBatches(ctx context.Context) ([]*models.Batch, error)
}

// BatchWriter defines write operations for batches.
type BatchWriter interface {
	// CreateBatch creates a new batch.
	CreateBatch(ctx context.Context, batch *models.Batch) error
	// UpdateBatch updates an existing batch.
	UpdateBatch(ctx context.Context, batch *models.Batch) error
	// DeleteBatch removes a batch by ID.
	DeleteBatch(ctx context.Context, batchID int64) error
	// AddRepositoriesToBatch adds repositories to a batch.
	AddRepositoriesToBatch(ctx context.Context, batchID int64, repoIDs []int64) error
	// RemoveRepositoriesFromBatch removes repositories from a batch.
	RemoveRepositoriesFromBatch(ctx context.Context, batchID int64, repoIDs []int64) error
}

// BatchStore combines read and write operations for batches.
type BatchStore interface {
	BatchReader
	BatchWriter
}

// MigrationHistoryStore defines operations for migration history and logs.
type MigrationHistoryStore interface {
	// GetMigrationHistory retrieves migration history for a repository.
	GetMigrationHistory(ctx context.Context, repoID int64) ([]*models.MigrationHistory, error)
	// GetMigrationLogs retrieves migration logs for a repository.
	GetMigrationLogs(ctx context.Context, repoID int64, level, phase string, limit, offset int) ([]*models.MigrationLog, error)
	// CreateMigrationHistory creates a new migration history entry.
	CreateMigrationHistory(ctx context.Context, history *models.MigrationHistory) (int64, error)
	// UpdateMigrationHistory updates a migration history entry.
	UpdateMigrationHistory(ctx context.Context, id int64, status string, errorMsg *string) error
	// CreateMigrationLog creates a new migration log entry.
	CreateMigrationLog(ctx context.Context, log *models.MigrationLog) error
}

// Compile-time interface checks.
// These ensure Database implements all defined interfaces.
var (
	_ RepositoryReader      = (*Database)(nil)
	_ RepositoryWriter      = (*Database)(nil)
	_ RepositoryStore       = (*Database)(nil)
	_ BatchReader           = (*Database)(nil)
	_ BatchWriter           = (*Database)(nil)
	_ BatchStore            = (*Database)(nil)
	_ MigrationHistoryStore = (*Database)(nil)
)
