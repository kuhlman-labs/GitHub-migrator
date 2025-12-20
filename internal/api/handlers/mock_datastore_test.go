package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"gorm.io/gorm"
)

// MockDataStore is a test implementation of DataStore that stores data in memory.
// It supports error injection for testing error paths.
type MockDataStore struct {
	mu sync.RWMutex

	// Data stores
	Repos            map[string]*models.Repository
	ReposByID        map[int64]*models.Repository
	Batches          map[int64]*models.Batch
	MigrationHistory map[int64][]*models.MigrationHistory
	MigrationLogs    map[int64][]*models.MigrationLog
	Dependencies     map[int64][]*models.RepositoryDependency
	Users            map[string]*models.GitHubUser
	UserMappings     map[string]*models.UserMapping
	Teams            map[string]*models.GitHubTeam // key: "org/slug"
	TeamMappings     map[string]*models.TeamMapping
	ADOProjects      map[string]*models.ADOProject // key: "org/project"

	// Auto-increment counters
	nextRepoID    int64
	nextBatchID   int64
	nextHistoryID int64

	// Error injection fields - set these to simulate errors
	GetRepoErr                error
	GetRepoByIDErr            error
	ListReposErr              error
	SaveRepoErr               error
	UpdateRepoErr             error
	GetBatchErr               error
	CreateBatchErr            error
	UpdateBatchErr            error
	DeleteBatchErr            error
	GetMigrationHistoryErr    error
	CreateMigrationHistoryErr error
	GetDependenciesErr        error
	GetUserErr                error
	SaveUserMappingErr        error
	GetTeamErr                error
	SaveTeamMappingErr        error
	GetADOProjectsErr         error
	SaveADOProjectErr         error
}

// NewMockDataStore creates a new MockDataStore with initialized maps.
func NewMockDataStore() *MockDataStore {
	return &MockDataStore{
		Repos:            make(map[string]*models.Repository),
		ReposByID:        make(map[int64]*models.Repository),
		Batches:          make(map[int64]*models.Batch),
		MigrationHistory: make(map[int64][]*models.MigrationHistory),
		MigrationLogs:    make(map[int64][]*models.MigrationLog),
		Dependencies:     make(map[int64][]*models.RepositoryDependency),
		Users:            make(map[string]*models.GitHubUser),
		UserMappings:     make(map[string]*models.UserMapping),
		Teams:            make(map[string]*models.GitHubTeam),
		TeamMappings:     make(map[string]*models.TeamMapping),
		ADOProjects:      make(map[string]*models.ADOProject),
		nextRepoID:       1,
		nextBatchID:      1,
		nextHistoryID:    1,
	}
}

// Error injection helpers - fluent API for chaining
func (m *MockDataStore) WithGetRepoError(err error) *MockDataStore {
	m.GetRepoErr = err
	return m
}

func (m *MockDataStore) WithSaveRepoError(err error) *MockDataStore {
	m.SaveRepoErr = err
	return m
}

func (m *MockDataStore) WithGetBatchError(err error) *MockDataStore {
	m.GetBatchErr = err
	return m
}

func (m *MockDataStore) WithCreateBatchError(err error) *MockDataStore {
	m.CreateBatchErr = err
	return m
}

// ============================================================================
// Repository Operations
// ============================================================================

func (m *MockDataStore) GetRepository(_ context.Context, fullName string) (*models.Repository, error) {
	if m.GetRepoErr != nil {
		return nil, m.GetRepoErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Repos[fullName], nil
}

func (m *MockDataStore) GetRepositoryByID(_ context.Context, id int64) (*models.Repository, error) {
	if m.GetRepoByIDErr != nil {
		return nil, m.GetRepoByIDErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ReposByID[id], nil
}

func (m *MockDataStore) GetRepositoriesByIDs(_ context.Context, ids []int64) ([]*models.Repository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Repository
	for _, id := range ids {
		if repo := m.ReposByID[id]; repo != nil {
			result = append(result, repo)
		}
	}
	return result, nil
}

func (m *MockDataStore) GetRepositoriesByNames(_ context.Context, names []string) ([]*models.Repository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Repository
	for _, name := range names {
		if repo := m.Repos[name]; repo != nil {
			result = append(result, repo)
		}
	}
	return result, nil
}

func (m *MockDataStore) ListRepositories(_ context.Context, _ map[string]interface{}) ([]*models.Repository, error) {
	if m.ListReposErr != nil {
		return nil, m.ListReposErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Repository, 0, len(m.Repos))
	for _, repo := range m.Repos {
		result = append(result, repo)
	}
	return result, nil
}

func (m *MockDataStore) CountRepositories(_ context.Context, _ map[string]interface{}) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Repos), nil
}

func (m *MockDataStore) CountRepositoriesWithFilters(_ context.Context, _ map[string]interface{}) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Repos), nil
}

func (m *MockDataStore) SaveRepository(_ context.Context, repo *models.Repository) error {
	if m.SaveRepoErr != nil {
		return m.SaveRepoErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if repo.ID == 0 {
		repo.ID = m.nextRepoID
		m.nextRepoID++
	}
	m.Repos[repo.FullName] = repo
	m.ReposByID[repo.ID] = repo
	return nil
}

func (m *MockDataStore) UpdateRepository(_ context.Context, repo *models.Repository) error {
	if m.UpdateRepoErr != nil {
		return m.UpdateRepoErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Repos[repo.FullName] = repo
	m.ReposByID[repo.ID] = repo
	return nil
}

func (m *MockDataStore) RollbackRepository(_ context.Context, fullName string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if repo, ok := m.Repos[fullName]; ok {
		repo.Status = string(models.StatusRolledBack)
	}
	return nil
}

func (m *MockDataStore) UpdateLocalDependencyFlags(_ context.Context) error {
	return nil
}

// ============================================================================
// Batch Operations
// ============================================================================

func (m *MockDataStore) GetBatch(_ context.Context, id int64) (*models.Batch, error) {
	if m.GetBatchErr != nil {
		return nil, m.GetBatchErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Batches[id], nil
}

func (m *MockDataStore) ListBatches(_ context.Context) ([]*models.Batch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Batch, 0, len(m.Batches))
	for _, batch := range m.Batches {
		result = append(result, batch)
	}
	return result, nil
}

func (m *MockDataStore) CreateBatch(_ context.Context, batch *models.Batch) error {
	if m.CreateBatchErr != nil {
		return m.CreateBatchErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	batch.ID = m.nextBatchID
	m.nextBatchID++
	m.Batches[batch.ID] = batch
	return nil
}

func (m *MockDataStore) UpdateBatch(_ context.Context, batch *models.Batch) error {
	if m.UpdateBatchErr != nil {
		return m.UpdateBatchErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Batches[batch.ID] = batch
	return nil
}

func (m *MockDataStore) DeleteBatch(_ context.Context, batchID int64) error {
	if m.DeleteBatchErr != nil {
		return m.DeleteBatchErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Batches, batchID)
	return nil
}

func (m *MockDataStore) AddRepositoriesToBatch(_ context.Context, batchID int64, repoIDs []int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, repoID := range repoIDs {
		if repo := m.ReposByID[repoID]; repo != nil {
			repo.BatchID = &batchID
		}
	}
	return nil
}

func (m *MockDataStore) RemoveRepositoriesFromBatch(_ context.Context, _ int64, repoIDs []int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, repoID := range repoIDs {
		if repo := m.ReposByID[repoID]; repo != nil {
			repo.BatchID = nil
		}
	}
	return nil
}

func (m *MockDataStore) UpdateBatchProgress(_ context.Context, batchID int64, status string, startedAt, lastDryRunAt, lastMigrationAttemptAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if batch := m.Batches[batchID]; batch != nil {
		batch.Status = status
		if startedAt != nil {
			batch.StartedAt = startedAt
		}
		if lastDryRunAt != nil {
			batch.LastDryRunAt = lastDryRunAt
		}
		if lastMigrationAttemptAt != nil {
			batch.LastMigrationAttemptAt = lastMigrationAttemptAt
		}
	}
	return nil
}

// ============================================================================
// Migration History Operations
// ============================================================================

func (m *MockDataStore) GetMigrationHistory(_ context.Context, repoID int64) ([]*models.MigrationHistory, error) {
	if m.GetMigrationHistoryErr != nil {
		return nil, m.GetMigrationHistoryErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MigrationHistory[repoID], nil
}

func (m *MockDataStore) GetMigrationLogs(_ context.Context, repoID int64, _, _ string, _, _ int) ([]*models.MigrationLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MigrationLogs[repoID], nil
}

func (m *MockDataStore) CreateMigrationHistory(_ context.Context, history *models.MigrationHistory) (int64, error) {
	if m.CreateMigrationHistoryErr != nil {
		return 0, m.CreateMigrationHistoryErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	history.ID = m.nextHistoryID
	m.nextHistoryID++
	m.MigrationHistory[history.RepositoryID] = append(m.MigrationHistory[history.RepositoryID], history)
	return history.ID, nil
}

func (m *MockDataStore) UpdateMigrationHistory(_ context.Context, id int64, status string, errorMsg *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, histories := range m.MigrationHistory {
		for _, h := range histories {
			if h.ID == id {
				h.Status = status
				h.ErrorMessage = errorMsg
				return nil
			}
		}
	}
	return nil
}

func (m *MockDataStore) CreateMigrationLog(_ context.Context, log *models.MigrationLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Find the repo ID from history
	if log.HistoryID == nil {
		return nil
	}
	for repoID, histories := range m.MigrationHistory {
		for _, h := range histories {
			if h.ID == *log.HistoryID {
				m.MigrationLogs[repoID] = append(m.MigrationLogs[repoID], log)
				return nil
			}
		}
	}
	return nil
}

func (m *MockDataStore) GetCompletedMigrations(_ context.Context) ([]*storage.CompletedMigration, error) {
	return []*storage.CompletedMigration{}, nil
}

// ============================================================================
// Dependency Operations
// ============================================================================

func (m *MockDataStore) GetRepositoryDependencies(_ context.Context, repoID int64) ([]*models.RepositoryDependency, error) {
	if m.GetDependenciesErr != nil {
		return nil, m.GetDependenciesErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Dependencies[repoID], nil
}

func (m *MockDataStore) GetRepositoryDependenciesByFullName(_ context.Context, fullName string) ([]*models.RepositoryDependency, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if repo := m.Repos[fullName]; repo != nil {
		return m.Dependencies[repo.ID], nil
	}
	return nil, nil
}

func (m *MockDataStore) GetDependentRepositories(_ context.Context, _ string) ([]*models.Repository, error) {
	return []*models.Repository{}, nil
}

func (m *MockDataStore) GetAllLocalDependencyPairs(_ context.Context, _ []string) ([]storage.DependencyPair, error) {
	return []storage.DependencyPair{}, nil
}

// ============================================================================
// Analytics Operations
// ============================================================================

func (m *MockDataStore) GetRepositoryStatsByStatus(_ context.Context) (map[string]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := make(map[string]int)
	for _, repo := range m.Repos {
		stats[repo.Status]++
	}
	return stats, nil
}

func (m *MockDataStore) GetRepositoryStatsByStatusFiltered(_ context.Context, _, _, _ string) (map[string]int, error) {
	return m.GetRepositoryStatsByStatus(context.Background())
}

func (m *MockDataStore) GetComplexityDistribution(_ context.Context, _, _, _ string) ([]*storage.ComplexityDistribution, error) {
	return []*storage.ComplexityDistribution{}, nil
}

func (m *MockDataStore) GetSizeDistributionFiltered(_ context.Context, _, _, _ string) ([]*storage.SizeDistribution, error) {
	return []*storage.SizeDistribution{}, nil
}

func (m *MockDataStore) GetFeatureStatsFiltered(_ context.Context, _, _, _ string) (*storage.FeatureStats, error) {
	return &storage.FeatureStats{}, nil
}

func (m *MockDataStore) GetMigrationVelocity(_ context.Context, _, _, _ string, _ int) (*storage.MigrationVelocity, error) {
	return &storage.MigrationVelocity{}, nil
}

func (m *MockDataStore) GetMigrationTimeSeries(_ context.Context, _, _, _ string) ([]*storage.MigrationTimeSeriesPoint, error) {
	return []*storage.MigrationTimeSeriesPoint{}, nil
}

func (m *MockDataStore) GetAverageMigrationTime(_ context.Context, _, _, _ string) (float64, error) {
	return 0, nil
}

func (m *MockDataStore) GetMedianMigrationTime(_ context.Context, _, _, _ string) (float64, error) {
	return 0, nil
}

func (m *MockDataStore) GetOrganizationStats(_ context.Context) ([]*storage.OrganizationStats, error) {
	return []*storage.OrganizationStats{}, nil
}

func (m *MockDataStore) GetOrganizationStatsFiltered(_ context.Context, _, _, _ string) ([]*storage.OrganizationStats, error) {
	return []*storage.OrganizationStats{}, nil
}

func (m *MockDataStore) GetProjectStatsFiltered(_ context.Context, _, _, _ string) ([]*storage.OrganizationStats, error) {
	return []*storage.OrganizationStats{}, nil
}

func (m *MockDataStore) GetMigrationCompletionStatsByOrgFiltered(_ context.Context, _, _, _ string) ([]*storage.MigrationCompletionStats, error) {
	return []*storage.MigrationCompletionStats{}, nil
}

func (m *MockDataStore) GetMigrationCompletionStatsByProjectFiltered(_ context.Context, _, _, _ string) ([]*storage.MigrationCompletionStats, error) {
	return []*storage.MigrationCompletionStats{}, nil
}

func (m *MockDataStore) GetDistinctOrganizations(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	orgs := make(map[string]bool)
	for fullName := range m.Repos {
		// Extract org from "org/repo" format
		for i, c := range fullName {
			if c == '/' {
				orgs[fullName[:i]] = true
				break
			}
		}
	}
	result := make([]string, 0, len(orgs))
	for org := range orgs {
		result = append(result, org)
	}
	return result, nil
}

func (m *MockDataStore) GetDashboardActionItems(_ context.Context) (*storage.DashboardActionItems, error) {
	return &storage.DashboardActionItems{}, nil
}

// ============================================================================
// User Operations
// ============================================================================

func (m *MockDataStore) ListUsers(_ context.Context, _ string, _, _ int) ([]*models.GitHubUser, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.GitHubUser, 0, len(m.Users))
	for _, user := range m.Users {
		result = append(result, user)
	}
	return result, int64(len(result)), nil
}

func (m *MockDataStore) ListUsersWithMappings(_ context.Context, _ storage.UserWithMappingFilters) ([]storage.UserWithMapping, int64, error) {
	return []storage.UserWithMapping{}, 0, nil
}

func (m *MockDataStore) GetUserByLogin(_ context.Context, login string) (*models.GitHubUser, error) {
	if m.GetUserErr != nil {
		return nil, m.GetUserErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Users[login], nil
}

func (m *MockDataStore) GetUserStats(_ context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{"total": len(m.Users)}, nil
}

func (m *MockDataStore) GetUsersWithMappingsStats(_ context.Context, _ string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockDataStore) GetUserOrgMemberships(_ context.Context, _ string) ([]*models.UserOrgMembership, error) {
	return []*models.UserOrgMembership{}, nil
}

func (m *MockDataStore) GetUserMappingSourceOrgs(_ context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockDataStore) SyncUserMappingsFromUsers(_ context.Context) (int64, error) {
	return 0, nil
}

func (m *MockDataStore) UpdateUserMappingSourceOrgsFromMemberships(_ context.Context) (int64, error) {
	return 0, nil
}

// ============================================================================
// User Mapping Operations
// ============================================================================

func (m *MockDataStore) ListUserMappings(_ context.Context, _ storage.UserMappingFilters) ([]*models.UserMapping, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.UserMapping, 0, len(m.UserMappings))
	for _, mapping := range m.UserMappings {
		result = append(result, mapping)
	}
	return result, int64(len(result)), nil
}

func (m *MockDataStore) GetUserMappingBySourceLogin(_ context.Context, sourceLogin string) (*models.UserMapping, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.UserMappings[sourceLogin], nil
}

func (m *MockDataStore) SaveUserMapping(_ context.Context, mapping *models.UserMapping) error {
	if m.SaveUserMappingErr != nil {
		return m.SaveUserMappingErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UserMappings[mapping.SourceLogin] = mapping
	return nil
}

func (m *MockDataStore) DeleteUserMapping(_ context.Context, sourceLogin string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.UserMappings, sourceLogin)
	return nil
}

func (m *MockDataStore) UpdateReclaimStatus(_ context.Context, sourceLogin string, status string, _ *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mapping := m.UserMappings[sourceLogin]; mapping != nil {
		mapping.ReclaimStatus = &status
	}
	return nil
}

// ============================================================================
// Team Operations
// ============================================================================

func (m *MockDataStore) ListTeams(_ context.Context, _ string) ([]*models.GitHubTeam, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.GitHubTeam, 0, len(m.Teams))
	for _, team := range m.Teams {
		result = append(result, team)
	}
	return result, nil
}

func (m *MockDataStore) ListTeamsWithMappings(_ context.Context, _ storage.TeamWithMappingFilters) ([]storage.TeamWithMapping, int64, error) {
	return []storage.TeamWithMapping{}, 0, nil
}

func (m *MockDataStore) GetTeamSourceOrgs(_ context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockDataStore) GetTeamsWithMappingsStats(_ context.Context, _ string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockDataStore) GetTeamMembersByOrgAndSlug(_ context.Context, _, _ string) ([]*models.GitHubTeamMember, error) {
	return []*models.GitHubTeamMember{}, nil
}

func (m *MockDataStore) GetTeamDetail(_ context.Context, org, slug string) (*storage.TeamDetail, error) {
	if m.GetTeamErr != nil {
		return nil, m.GetTeamErr
	}
	key := fmt.Sprintf("%s/%s", org, slug)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if team := m.Teams[key]; team != nil {
		return &storage.TeamDetail{
			ID:           team.ID,
			Organization: team.Organization,
			Slug:         team.Slug,
			Name:         team.Name,
			Description:  team.Description,
			Privacy:      team.Privacy,
			DiscoveredAt: team.DiscoveredAt,
		}, nil
	}
	return nil, nil
}

func (m *MockDataStore) GetUserMappingStats(_ context.Context, _ string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockDataStore) GetTeamMappingStats(_ context.Context, _ string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockDataStore) GetTeamMigrationExecutionStats(_ context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockDataStore) ResetTeamMigrationStatus(_ context.Context, _ string) error {
	return nil
}

// ============================================================================
// Team Mapping Operations
// ============================================================================

func (m *MockDataStore) ListTeamMappings(_ context.Context, _ storage.TeamMappingFilters) ([]*models.TeamMapping, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.TeamMapping, 0, len(m.TeamMappings))
	for _, mapping := range m.TeamMappings {
		result = append(result, mapping)
	}
	return result, int64(len(result)), nil
}

func (m *MockDataStore) GetTeamMapping(_ context.Context, sourceOrg, sourceTeamSlug string) (*models.TeamMapping, error) {
	key := fmt.Sprintf("%s/%s", sourceOrg, sourceTeamSlug)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.TeamMappings[key], nil
}

func (m *MockDataStore) SaveTeamMapping(_ context.Context, mapping *models.TeamMapping) error {
	if m.SaveTeamMappingErr != nil {
		return m.SaveTeamMappingErr
	}
	key := fmt.Sprintf("%s/%s", mapping.SourceOrg, mapping.SourceTeamSlug)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TeamMappings[key] = mapping
	return nil
}

func (m *MockDataStore) DeleteTeamMapping(_ context.Context, sourceOrg, sourceSlug string) error {
	key := fmt.Sprintf("%s/%s", sourceOrg, sourceSlug)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.TeamMappings, key)
	return nil
}

func (m *MockDataStore) SyncTeamMappingsFromTeams(_ context.Context) (int64, error) {
	return 0, nil
}

func (m *MockDataStore) SuggestTeamMappings(_ context.Context, _ string, _ []string) (map[string]string, error) {
	return map[string]string{}, nil
}

// ============================================================================
// ADO Operations
// ============================================================================

func (m *MockDataStore) GetADOProjects(_ context.Context, org string) ([]models.ADOProject, error) {
	if m.GetADOProjectsErr != nil {
		return nil, m.GetADOProjectsErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []models.ADOProject
	for key, project := range m.ADOProjects {
		// Check if key starts with org
		if org == "" || (len(key) > len(org) && key[:len(org)] == org && key[len(org)] == '/') {
			result = append(result, *project)
		}
	}
	return result, nil
}

func (m *MockDataStore) GetADOProject(_ context.Context, org, projectName string) (*models.ADOProject, error) {
	key := fmt.Sprintf("%s/%s", org, projectName)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ADOProjects[key], nil
}

func (m *MockDataStore) SaveADOProject(_ context.Context, project *models.ADOProject) error {
	if m.SaveADOProjectErr != nil {
		return m.SaveADOProjectErr
	}
	key := fmt.Sprintf("%s/%s", project.Organization, project.Name)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ADOProjects[key] = project
	return nil
}

func (m *MockDataStore) CountRepositoriesByADOProject(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}

func (m *MockDataStore) CountRepositoriesByADOOrganization(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *MockDataStore) CountTFVCRepositories(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *MockDataStore) GetRepositoriesByADOProject(_ context.Context, _, _ string) ([]models.Repository, error) {
	return []models.Repository{}, nil
}

// ============================================================================
// Discovery Operations
// ============================================================================

func (m *MockDataStore) CreateDiscoveryProgress(_ *models.DiscoveryProgress) error {
	return nil
}

func (m *MockDataStore) UpdateDiscoveryProgress(_ *models.DiscoveryProgress) error {
	return nil
}

func (m *MockDataStore) GetActiveDiscoveryProgress() (*models.DiscoveryProgress, error) {
	return nil, nil
}

func (m *MockDataStore) GetLatestDiscoveryProgress() (*models.DiscoveryProgress, error) {
	return nil, nil
}

func (m *MockDataStore) MarkDiscoveryComplete(_ int64) error {
	return nil
}

func (m *MockDataStore) MarkDiscoveryFailed(_ int64, _ string) error {
	return nil
}

// ============================================================================
// Setup Operations
// ============================================================================

func (m *MockDataStore) GetSetupStatus() (*storage.SetupStatus, error) {
	return &storage.SetupStatus{SetupCompleted: true}, nil
}

func (m *MockDataStore) MarkSetupComplete() error {
	return nil
}

// ============================================================================
// Low-level DB Access
// ============================================================================

func (m *MockDataStore) DB() *gorm.DB {
	// Return nil - tests using MockDataStore shouldn't need raw DB access
	// If they do, they should use setupTestDB instead
	return nil
}

// Compile-time check that MockDataStore implements DataStore
var _ DataStore = (*MockDataStore)(nil)
