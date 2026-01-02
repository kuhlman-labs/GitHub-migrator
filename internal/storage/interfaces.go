package storage

import (
	"context"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
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
	ListRepositories(ctx context.Context, filters map[string]any) ([]*models.Repository, error)
	// CountRepositories counts repositories matching the given filters.
	CountRepositories(ctx context.Context, filters map[string]any) (int, error)
	// CountRepositoriesWithFilters counts repositories with filters applied.
	CountRepositoriesWithFilters(ctx context.Context, filters map[string]any) (int, error)
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
	// RollbackRepository rolls back a repository migration.
	RollbackRepository(ctx context.Context, fullName string, reason string) error
	// UpdateLocalDependencyFlags updates dependency flags for all repositories.
	UpdateLocalDependencyFlags(ctx context.Context) error
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
	// UpdateBatchProgress updates batch progress tracking.
	UpdateBatchProgress(ctx context.Context, batchID int64, status string, startedAt, lastDryRunAt, lastMigrationAttemptAt *time.Time) error
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
	// GetCompletedMigrations retrieves all completed migrations, optionally filtered by source.
	GetCompletedMigrations(ctx context.Context, sourceID *int64) ([]*CompletedMigration, error)
}

// DependencyStore defines operations for repository dependencies.
type DependencyStore interface {
	// GetRepositoryDependencies retrieves dependencies for a repository by ID.
	GetRepositoryDependencies(ctx context.Context, repoID int64) ([]*models.RepositoryDependency, error)
	// GetRepositoryDependenciesByFullName retrieves dependencies by repository full name.
	GetRepositoryDependenciesByFullName(ctx context.Context, fullName string) ([]*models.RepositoryDependency, error)
	// GetDependentRepositories retrieves repositories that depend on a given repository.
	GetDependentRepositories(ctx context.Context, dependencyFullName string) ([]*models.Repository, error)
	// GetAllLocalDependencyPairs retrieves all local dependency relationships.
	GetAllLocalDependencyPairs(ctx context.Context, dependencyTypes []string, sourceID *int64) ([]DependencyPair, error)
}

// AnalyticsStore defines operations for analytics and statistics.
type AnalyticsStore interface {
	// GetRepositoryStatsByStatus returns repository counts grouped by status.
	GetRepositoryStatsByStatus(ctx context.Context) (map[string]int, error)
	// GetRepositoryStatsByStatusFiltered returns filtered repository counts by status.
	GetRepositoryStatsByStatusFiltered(ctx context.Context, org, project, batchFilter string, sourceID *int64) (map[string]int, error)
	// GetComplexityDistribution returns complexity score distribution.
	GetComplexityDistribution(ctx context.Context, org, project, batchFilter string, sourceID *int64) ([]*ComplexityDistribution, error)
	// GetSizeDistributionFiltered returns repository size distribution.
	GetSizeDistributionFiltered(ctx context.Context, org, project, batchFilter string, sourceID *int64) ([]*SizeDistribution, error)
	// GetFeatureStatsFiltered returns feature usage statistics.
	GetFeatureStatsFiltered(ctx context.Context, org, project, batchFilter string, sourceID *int64) (*FeatureStats, error)
	// GetMigrationVelocity returns migration velocity metrics.
	GetMigrationVelocity(ctx context.Context, org, project, batchFilter string, sourceID *int64, days int) (*MigrationVelocity, error)
	// GetMigrationTimeSeries returns migration time series data.
	GetMigrationTimeSeries(ctx context.Context, org, project, batchFilter string, sourceID *int64) ([]*MigrationTimeSeriesPoint, error)
	// GetAverageMigrationTime returns average migration duration.
	GetAverageMigrationTime(ctx context.Context, org, project, batchFilter string, sourceID *int64) (float64, error)
	// GetMedianMigrationTime returns median migration duration.
	GetMedianMigrationTime(ctx context.Context, org, project, batchFilter string, sourceID *int64) (float64, error)
	// GetOrganizationStats returns statistics grouped by organization.
	GetOrganizationStats(ctx context.Context) ([]*OrganizationStats, error)
	// GetOrganizationStatsFiltered returns filtered organization statistics.
	GetOrganizationStatsFiltered(ctx context.Context, org, project, batchFilter string, sourceID *int64) ([]*OrganizationStats, error)
	// GetProjectStatsFiltered returns filtered project statistics.
	GetProjectStatsFiltered(ctx context.Context, org, project, batchFilter string, sourceID *int64) ([]*OrganizationStats, error)
	// GetMigrationCompletionStatsByOrgFiltered returns migration completion by org.
	GetMigrationCompletionStatsByOrgFiltered(ctx context.Context, org, project, batchFilter string, sourceID *int64) ([]*MigrationCompletionStats, error)
	// GetMigrationCompletionStatsByProjectFiltered returns migration completion by project.
	GetMigrationCompletionStatsByProjectFiltered(ctx context.Context, org, project, batchFilter string, sourceID *int64) ([]*MigrationCompletionStats, error)
	// GetDistinctOrganizations returns all unique organizations.
	GetDistinctOrganizations(ctx context.Context) ([]string, error)
	// GetDashboardActionItems returns action items for the dashboard.
	GetDashboardActionItems(ctx context.Context) (*DashboardActionItems, error)
}

// UserStore defines operations for users and user mappings.
type UserStore interface {
	// ListUsers lists users with pagination.
	ListUsers(ctx context.Context, sourceInstance string, limit, offset int) ([]*models.GitHubUser, int64, error)
	// ListUsersWithMappings lists users with their mappings.
	ListUsersWithMappings(ctx context.Context, filters UserWithMappingFilters) ([]UserWithMapping, int64, error)
	// GetUserByLogin retrieves a user by login.
	GetUserByLogin(ctx context.Context, login string) (*models.GitHubUser, error)
	// GetUserStats returns user statistics.
	GetUserStats(ctx context.Context) (map[string]any, error)
	// GetUsersWithMappingsStats returns user mapping statistics.
	GetUsersWithMappingsStats(ctx context.Context, org string, sourceID *int) (map[string]any, error)
	// GetUserOrgMemberships retrieves organization memberships for a user.
	GetUserOrgMemberships(ctx context.Context, login string) ([]*models.UserOrgMembership, error)
	// GetUserMappingSourceOrgs returns source organizations for user mappings.
	GetUserMappingSourceOrgs(ctx context.Context) ([]string, error)
	// SyncUserMappingsFromUsers syncs user mappings from discovered users.
	SyncUserMappingsFromUsers(ctx context.Context) (int64, error)
	// UpdateUserMappingSourceOrgsFromMemberships updates source orgs from memberships.
	UpdateUserMappingSourceOrgsFromMemberships(ctx context.Context) (int64, error)
}

// UserMappingStore defines operations for user mappings.
type UserMappingStore interface {
	// ListUserMappings lists user mappings with filters.
	ListUserMappings(ctx context.Context, filters UserMappingFilters) ([]*models.UserMapping, int64, error)
	// GetUserMappingBySourceLogin retrieves a mapping by source login.
	GetUserMappingBySourceLogin(ctx context.Context, sourceLogin string) (*models.UserMapping, error)
	// SaveUserMapping creates or updates a user mapping.
	SaveUserMapping(ctx context.Context, mapping *models.UserMapping) error
	// DeleteUserMapping removes a user mapping.
	DeleteUserMapping(ctx context.Context, sourceLogin string) error
	// UpdateReclaimStatus updates the reclaim status of a user mapping.
	UpdateReclaimStatus(ctx context.Context, sourceLogin string, status string, errorMsg *string) error
	// GetUserMappingStats returns user mapping statistics.
	GetUserMappingStats(ctx context.Context, org string) (map[string]any, error)
}

// TeamStore defines operations for teams.
type TeamStore interface {
	// ListTeams lists teams for an organization.
	ListTeams(ctx context.Context, org string) ([]*models.GitHubTeam, error)
	// ListTeamsWithMappings lists teams with their mappings.
	ListTeamsWithMappings(ctx context.Context, filters TeamWithMappingFilters) ([]TeamWithMapping, int64, error)
	// GetTeamSourceOrgs returns source organizations for teams.
	GetTeamSourceOrgs(ctx context.Context) ([]string, error)
	// GetTeamsWithMappingsStats returns team mapping statistics.
	GetTeamsWithMappingsStats(ctx context.Context, org string, sourceID *int) (map[string]any, error)
	// GetTeamMembersByOrgAndSlug retrieves team members.
	GetTeamMembersByOrgAndSlug(ctx context.Context, org, slug string) ([]*models.GitHubTeamMember, error)
	// GetTeamDetail retrieves detailed team information.
	GetTeamDetail(ctx context.Context, org, slug string) (*TeamDetail, error)
}

// TeamMappingStore defines operations for team mappings.
type TeamMappingStore interface {
	// ListTeamMappings lists team mappings with filters.
	ListTeamMappings(ctx context.Context, filters TeamMappingFilters) ([]*models.TeamMapping, int64, error)
	// GetTeamMapping retrieves a team mapping.
	GetTeamMapping(ctx context.Context, sourceOrg, sourceTeamSlug string) (*models.TeamMapping, error)
	// SaveTeamMapping creates or updates a team mapping.
	SaveTeamMapping(ctx context.Context, mapping *models.TeamMapping) error
	// DeleteTeamMapping removes a team mapping.
	DeleteTeamMapping(ctx context.Context, sourceOrg, sourceSlug string) error
	// SyncTeamMappingsFromTeams syncs team mappings from discovered teams.
	SyncTeamMappingsFromTeams(ctx context.Context) (int64, error)
	// SuggestTeamMappings suggests mappings based on destination teams.
	SuggestTeamMappings(ctx context.Context, destOrg string, destSlugs []string) (map[string]string, error)
	// GetTeamMappingStats returns team mapping statistics.
	GetTeamMappingStats(ctx context.Context, org string) (map[string]any, error)
	// GetTeamMigrationExecutionStats returns team migration execution statistics.
	GetTeamMigrationExecutionStats(ctx context.Context) (map[string]any, error)
	// ResetTeamMigrationStatus resets migration status for teams in an org.
	ResetTeamMigrationStatus(ctx context.Context, sourceOrg string) error
}

// ADOStore defines operations for Azure DevOps data.
type ADOStore interface {
	// GetADOProjects retrieves ADO projects for an organization.
	GetADOProjects(ctx context.Context, org string) ([]models.ADOProject, error)
	// GetADOProjectsFiltered retrieves ADO projects with optional source_id filtering.
	GetADOProjectsFiltered(ctx context.Context, org string, sourceID *int64) ([]models.ADOProject, error)
	// GetADOProject retrieves a single ADO project.
	GetADOProject(ctx context.Context, org, projectName string) (*models.ADOProject, error)
	// SaveADOProject creates or updates an ADO project.
	SaveADOProject(ctx context.Context, project *models.ADOProject) error
	// CountRepositoriesByADOProject counts repositories in an ADO project.
	CountRepositoriesByADOProject(ctx context.Context, org, project string) (int, error)
	// CountRepositoriesByADOProjectFiltered counts repositories in an ADO project with source filtering.
	CountRepositoriesByADOProjectFiltered(ctx context.Context, org, project string, sourceID *int64) (int, error)
	// CountRepositoriesByADOOrganization counts repositories in an ADO organization.
	CountRepositoriesByADOOrganization(ctx context.Context, org string) (int, error)
	// CountTFVCRepositories counts TFVC repositories.
	CountTFVCRepositories(ctx context.Context, org string) (int, error)
	// GetRepositoriesByADOProject retrieves repositories for an ADO project.
	GetRepositoriesByADOProject(ctx context.Context, org, project string) ([]models.Repository, error)
}

// DiscoveryStore defines operations for discovery progress tracking.
type DiscoveryStore interface {
	// CreateDiscoveryProgress creates a new discovery progress record.
	CreateDiscoveryProgress(progress *models.DiscoveryProgress) error
	// UpdateDiscoveryProgress updates a discovery progress record.
	UpdateDiscoveryProgress(progress *models.DiscoveryProgress) error
	// UpdateDiscoveryRepoProgress updates repository-level progress counters.
	UpdateDiscoveryRepoProgress(id int64, processedRepos, totalRepos int) error
	// UpdateDiscoveryPhase updates the current phase of a discovery.
	UpdateDiscoveryPhase(id int64, phase string) error
	// IncrementDiscoveryError increments the error count and records the last error.
	IncrementDiscoveryError(id int64, errMsg string) error
	// GetActiveDiscoveryProgress retrieves the active discovery progress.
	GetActiveDiscoveryProgress() (*models.DiscoveryProgress, error)
	// GetLatestDiscoveryProgress retrieves the most recent discovery progress.
	GetLatestDiscoveryProgress() (*models.DiscoveryProgress, error)
	// MarkDiscoveryComplete marks a discovery as complete.
	MarkDiscoveryComplete(id int64) error
	// MarkDiscoveryFailed marks a discovery as failed.
	MarkDiscoveryFailed(id int64, errorMsg string) error
}

// SourceStore defines operations for migration sources.
type SourceStore interface {
	// GetSource retrieves a source by ID.
	GetSource(ctx context.Context, id int64) (*models.Source, error)
	// UpdateSourceRepositoryCount updates the cached repository count and last sync time for a source.
	UpdateSourceRepositoryCount(ctx context.Context, id int64) error
}

// SetupStore defines operations for setup status.
type SetupStore interface {
	// GetSetupStatus retrieves the current setup status.
	GetSetupStatus() (*SetupStatus, error)
	// MarkSetupComplete marks setup as complete.
	MarkSetupComplete() error
}

// DatabaseAccess provides low-level database access.
type DatabaseAccess interface {
	// DB returns the underlying GORM database connection.
	DB() *gorm.DB
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
	_ DependencyStore       = (*Database)(nil)
	_ AnalyticsStore        = (*Database)(nil)
	_ UserStore             = (*Database)(nil)
	_ UserMappingStore      = (*Database)(nil)
	_ TeamStore             = (*Database)(nil)
	_ TeamMappingStore      = (*Database)(nil)
	_ SourceStore           = (*Database)(nil)
	_ ADOStore              = (*Database)(nil)
	_ DiscoveryStore        = (*Database)(nil)
	_ SetupStore            = (*Database)(nil)
	_ DatabaseAccess        = (*Database)(nil)
)
