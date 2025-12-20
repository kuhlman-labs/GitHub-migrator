// Package handlers provides HTTP request handlers for the API.
//
// # Service Interfaces
//
// This file defines service interfaces that can be used for dependency injection
// and testing. The DataStore interface is the primary interface used by handlers
// for data access operations.
//
// Usage in tests:
//
//	type mockDataStore struct {
//	    repos map[string]*models.Repository
//	}
//
//	func (m *mockDataStore) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
//	    if repo, ok := m.repos[fullName]; ok {
//	        return repo, nil
//	    }
//	    return nil, nil
//	}
//	// ... implement other methods as needed
package handlers

import (
	"context"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"gorm.io/gorm"
)

// DataStore defines the interface for all data access operations used by handlers.
// This interface enables dependency injection and easier unit testing through mocking.
//
// storage.Database implements this interface.
type DataStore interface {
	// Repository operations
	GetRepository(ctx context.Context, fullName string) (*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error)
	GetRepositoriesByIDs(ctx context.Context, ids []int64) ([]*models.Repository, error)
	GetRepositoriesByNames(ctx context.Context, names []string) ([]*models.Repository, error)
	ListRepositories(ctx context.Context, filters map[string]interface{}) ([]*models.Repository, error)
	CountRepositories(ctx context.Context, filters map[string]interface{}) (int, error)
	CountRepositoriesWithFilters(ctx context.Context, filters map[string]interface{}) (int, error)
	SaveRepository(ctx context.Context, repo *models.Repository) error
	UpdateRepository(ctx context.Context, repo *models.Repository) error
	RollbackRepository(ctx context.Context, fullName string, reason string) error
	UpdateLocalDependencyFlags(ctx context.Context) error

	// Batch operations
	GetBatch(ctx context.Context, id int64) (*models.Batch, error)
	ListBatches(ctx context.Context) ([]*models.Batch, error)
	CreateBatch(ctx context.Context, batch *models.Batch) error
	UpdateBatch(ctx context.Context, batch *models.Batch) error
	DeleteBatch(ctx context.Context, batchID int64) error
	AddRepositoriesToBatch(ctx context.Context, batchID int64, repoIDs []int64) error
	RemoveRepositoriesFromBatch(ctx context.Context, batchID int64, repoIDs []int64) error
	UpdateBatchProgress(ctx context.Context, batchID int64, status string, startedAt, lastDryRunAt, lastMigrationAttemptAt *time.Time) error

	// Migration history operations
	GetMigrationHistory(ctx context.Context, repoID int64) ([]*models.MigrationHistory, error)
	GetMigrationLogs(ctx context.Context, repoID int64, level, phase string, limit, offset int) ([]*models.MigrationLog, error)
	CreateMigrationHistory(ctx context.Context, history *models.MigrationHistory) (int64, error)
	UpdateMigrationHistory(ctx context.Context, id int64, status string, errorMsg *string) error
	CreateMigrationLog(ctx context.Context, log *models.MigrationLog) error
	GetCompletedMigrations(ctx context.Context) ([]*storage.CompletedMigration, error)

	// Dependency operations
	GetRepositoryDependencies(ctx context.Context, repoID int64) ([]*models.RepositoryDependency, error)
	GetRepositoryDependenciesByFullName(ctx context.Context, fullName string) ([]*models.RepositoryDependency, error)
	GetDependentRepositories(ctx context.Context, dependencyFullName string) ([]*models.Repository, error)
	GetAllLocalDependencyPairs(ctx context.Context, dependencyTypes []string) ([]storage.DependencyPair, error)

	// Analytics operations (return types match storage.Database implementation)
	GetRepositoryStatsByStatus(ctx context.Context) (map[string]int, error)
	GetRepositoryStatsByStatusFiltered(ctx context.Context, org, project, batchFilter string) (map[string]int, error)
	GetComplexityDistribution(ctx context.Context, org, project, batchFilter string) ([]*storage.ComplexityDistribution, error)
	GetSizeDistributionFiltered(ctx context.Context, org, project, batchFilter string) ([]*storage.SizeDistribution, error)
	GetFeatureStatsFiltered(ctx context.Context, org, project, batchFilter string) (*storage.FeatureStats, error)
	GetMigrationVelocity(ctx context.Context, org, project, batchFilter string, days int) (*storage.MigrationVelocity, error)
	GetMigrationTimeSeries(ctx context.Context, org, project, batchFilter string) ([]*storage.MigrationTimeSeriesPoint, error)
	GetAverageMigrationTime(ctx context.Context, org, project, batchFilter string) (float64, error)
	GetMedianMigrationTime(ctx context.Context, org, project, batchFilter string) (float64, error)
	GetOrganizationStats(ctx context.Context) ([]*storage.OrganizationStats, error)
	GetOrganizationStatsFiltered(ctx context.Context, org, project, batchFilter string) ([]*storage.OrganizationStats, error)
	GetProjectStatsFiltered(ctx context.Context, org, project, batchFilter string) ([]*storage.OrganizationStats, error)
	GetMigrationCompletionStatsByOrgFiltered(ctx context.Context, org, project, batchFilter string) ([]*storage.MigrationCompletionStats, error)
	GetMigrationCompletionStatsByProjectFiltered(ctx context.Context, org, project, batchFilter string) ([]*storage.MigrationCompletionStats, error)
	GetDistinctOrganizations(ctx context.Context) ([]string, error)
	GetDashboardActionItems(ctx context.Context) (*storage.DashboardActionItems, error)

	// User operations
	ListUsers(ctx context.Context, sourceInstance string, limit, offset int) ([]*models.GitHubUser, int64, error)
	ListUsersWithMappings(ctx context.Context, filters storage.UserWithMappingFilters) ([]storage.UserWithMapping, int64, error)
	GetUserByLogin(ctx context.Context, login string) (*models.GitHubUser, error)
	GetUserStats(ctx context.Context) (map[string]interface{}, error)
	GetUsersWithMappingsStats(ctx context.Context, org string) (map[string]interface{}, error)
	GetUserOrgMemberships(ctx context.Context, login string) ([]*models.UserOrgMembership, error)
	GetUserMappingSourceOrgs(ctx context.Context) ([]string, error)
	SyncUserMappingsFromUsers(ctx context.Context) (int64, error)
	UpdateUserMappingSourceOrgsFromMemberships(ctx context.Context) (int64, error)

	// User mapping operations
	ListUserMappings(ctx context.Context, filters storage.UserMappingFilters) ([]*models.UserMapping, int64, error)
	GetUserMappingBySourceLogin(ctx context.Context, sourceLogin string) (*models.UserMapping, error)
	SaveUserMapping(ctx context.Context, mapping *models.UserMapping) error
	DeleteUserMapping(ctx context.Context, sourceLogin string) error
	UpdateReclaimStatus(ctx context.Context, sourceLogin string, status string, errorMsg *string) error

	// Team operations
	ListTeams(ctx context.Context, org string) ([]*models.GitHubTeam, error)
	ListTeamsWithMappings(ctx context.Context, filters storage.TeamWithMappingFilters) ([]storage.TeamWithMapping, int64, error)
	GetTeamSourceOrgs(ctx context.Context) ([]string, error)
	GetTeamsWithMappingsStats(ctx context.Context, org string) (map[string]interface{}, error)
	GetTeamMembersByOrgAndSlug(ctx context.Context, org, slug string) ([]*models.GitHubTeamMember, error)
	GetTeamDetail(ctx context.Context, org, slug string) (*storage.TeamDetail, error)
	GetUserMappingStats(ctx context.Context, org string) (map[string]interface{}, error)
	GetTeamMappingStats(ctx context.Context, org string) (map[string]interface{}, error)
	GetTeamMigrationExecutionStats(ctx context.Context) (map[string]interface{}, error)
	ResetTeamMigrationStatus(ctx context.Context, sourceOrg string) error

	// Team mapping operations
	ListTeamMappings(ctx context.Context, filters storage.TeamMappingFilters) ([]*models.TeamMapping, int64, error)
	GetTeamMapping(ctx context.Context, sourceOrg, sourceTeamSlug string) (*models.TeamMapping, error)
	SaveTeamMapping(ctx context.Context, mapping *models.TeamMapping) error
	DeleteTeamMapping(ctx context.Context, sourceOrg, sourceSlug string) error
	SyncTeamMappingsFromTeams(ctx context.Context) (int64, error)
	SuggestTeamMappings(ctx context.Context, destOrg string, destSlugs []string) (map[string]string, error)

	// ADO operations
	GetADOProjects(ctx context.Context, org string) ([]models.ADOProject, error)
	GetADOProject(ctx context.Context, org, projectName string) (*models.ADOProject, error)
	SaveADOProject(ctx context.Context, project *models.ADOProject) error
	CountRepositoriesByADOProject(ctx context.Context, org, project string) (int, error)
	CountRepositoriesByADOOrganization(ctx context.Context, org string) (int, error)
	CountTFVCRepositories(ctx context.Context, org string) (int, error)
	GetRepositoriesByADOProject(ctx context.Context, org, project string) ([]models.Repository, error)

	// Discovery operations
	CreateDiscoveryProgress(progress *models.DiscoveryProgress) error
	UpdateDiscoveryProgress(progress *models.DiscoveryProgress) error
	GetActiveDiscoveryProgress() (*models.DiscoveryProgress, error)
	GetLatestDiscoveryProgress() (*models.DiscoveryProgress, error)
	MarkDiscoveryComplete(id int64) error
	MarkDiscoveryFailed(id int64, errorMsg string) error

	// Setup operations
	GetSetupStatus() (*storage.SetupStatus, error)
	MarkSetupComplete() error

	// Low-level DB access (for complex queries)
	DB() *gorm.DB
}

// Compile-time check that storage.Database implements DataStore
var _ DataStore = (*storage.Database)(nil)

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
