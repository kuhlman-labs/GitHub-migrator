package services

import (
	"context"
	"errors"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// MockHistoryStore is a mock implementation of storage.MigrationHistoryStore.
type MockHistoryStore struct {
	history    map[int64][]*models.MigrationHistory
	getHistErr error
}

func NewMockHistoryStore() *MockHistoryStore {
	return &MockHistoryStore{
		history: make(map[int64][]*models.MigrationHistory),
	}
}

func (m *MockHistoryStore) GetMigrationHistory(_ context.Context, repoID int64) ([]*models.MigrationHistory, error) {
	if m.getHistErr != nil {
		return nil, m.getHistErr
	}
	return m.history[repoID], nil
}

func (m *MockHistoryStore) GetMigrationLogs(_ context.Context, _ int64, _, _ string, _, _ int) ([]*models.MigrationLog, error) {
	return nil, nil
}

func (m *MockHistoryStore) CreateMigrationHistory(_ context.Context, _ *models.MigrationHistory) (int64, error) {
	return 1, nil
}

func (m *MockHistoryStore) UpdateMigrationHistory(_ context.Context, _ int64, _ string, _ *string) error {
	return nil
}

func (m *MockHistoryStore) CreateMigrationLog(_ context.Context, _ *models.MigrationLog) error {
	return nil
}

func (m *MockHistoryStore) GetCompletedMigrations(_ context.Context, _ *int64) ([]*storage.CompletedMigration, error) {
	return nil, nil
}

// MockDepStore is a mock implementation of storage.DependencyStore.
type MockDepStore struct {
	deps             map[int64][]*models.RepositoryDependency
	depsByFullName   map[string][]*models.RepositoryDependency
	dependents       map[string][]*models.Repository
	getDepsErr       error
	getDepsByNameErr error
	getDependentsErr error
}

func NewMockDepStore() *MockDepStore {
	return &MockDepStore{
		deps:           make(map[int64][]*models.RepositoryDependency),
		depsByFullName: make(map[string][]*models.RepositoryDependency),
		dependents:     make(map[string][]*models.Repository),
	}
}

func (m *MockDepStore) GetRepositoryDependencies(_ context.Context, repoID int64) ([]*models.RepositoryDependency, error) {
	if m.getDepsErr != nil {
		return nil, m.getDepsErr
	}
	return m.deps[repoID], nil
}

func (m *MockDepStore) GetRepositoryDependenciesByFullName(_ context.Context, fullName string) ([]*models.RepositoryDependency, error) {
	if m.getDepsByNameErr != nil {
		return nil, m.getDepsByNameErr
	}
	return m.depsByFullName[fullName], nil
}

func (m *MockDepStore) GetDependentRepositories(_ context.Context, dependencyFullName string) ([]*models.Repository, error) {
	if m.getDependentsErr != nil {
		return nil, m.getDependentsErr
	}
	return m.dependents[dependencyFullName], nil
}

func (m *MockDepStore) GetAllLocalDependencyPairs(_ context.Context, _ []string, _ *int64) ([]storage.DependencyPair, error) {
	return nil, nil
}

func TestNewRepositoryService(t *testing.T) {
	repoStore := NewMockRepoStore()
	histStore := NewMockHistoryStore()
	depStore := NewMockDepStore()
	logger := newTestLogger()

	svc := NewRepositoryService(repoStore, histStore, depStore, logger)
	if svc == nil {
		t.Fatal("expected non-nil RepositoryService")
		return
	}
	if svc.repoStore == nil {
		t.Error("repoStore not set")
	}
	if svc.historyStore == nil {
		t.Error("historyStore not set")
	}
	if svc.depStore == nil {
		t.Error("depStore not set")
	}
}

func TestGetRepositoryWithDetails(t *testing.T) {
	tests := []struct {
		name        string
		fullName    string
		repo        *models.Repository
		history     []*models.MigrationHistory
		deps        []*models.RepositoryDependency
		getRepoErr  error
		getHistErr  error
		getDepsErr  error
		wantNil     bool
		wantErr     bool
		wantHistory int
		wantDeps    int
	}{
		{
			name:     "repo not found",
			fullName: "org/unknown",
			wantNil:  true,
		},
		{
			name:       "get repo error",
			fullName:   "org/repo1",
			getRepoErr: errors.New("db error"),
			wantErr:    true,
		},
		{
			name:     "repo with no history or deps",
			fullName: "org/repo1",
			repo:     &models.Repository{ID: 1, FullName: "org/repo1"},
		},
		{
			name:     "repo with history and deps",
			fullName: "org/repo1",
			repo:     &models.Repository{ID: 1, FullName: "org/repo1"},
			history: []*models.MigrationHistory{
				{ID: 1, RepositoryID: 1, Status: "completed"},
			},
			deps: []*models.RepositoryDependency{
				{ID: 1, RepositoryID: 1, DependencyFullName: "org/dep1"},
			},
			wantHistory: 1,
			wantDeps:    1,
		},
		{
			name:        "history error continues",
			fullName:    "org/repo1",
			repo:        &models.Repository{ID: 1, FullName: "org/repo1"},
			getHistErr:  errors.New("history error"),
			wantHistory: 0,
		},
		{
			name:       "deps error continues",
			fullName:   "org/repo1",
			repo:       &models.Repository{ID: 1, FullName: "org/repo1"},
			getDepsErr: errors.New("deps error"),
			wantDeps:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoStore := NewMockRepoStore()
			histStore := NewMockHistoryStore()
			depStore := NewMockDepStore()

			if tt.repo != nil {
				repoStore.AddRepo(tt.repo)
				histStore.history[tt.repo.ID] = tt.history
				depStore.deps[tt.repo.ID] = tt.deps
			}
			repoStore.getRepoErr = tt.getRepoErr
			histStore.getHistErr = tt.getHistErr
			depStore.getDepsErr = tt.getDepsErr

			svc := NewRepositoryService(repoStore, histStore, depStore, newTestLogger())
			result, err := svc.GetRepositoryWithDetails(context.Background(), tt.fullName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.wantNil {
				if result != nil {
					t.Error("expected nil result")
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
				return
			}

			if len(result.History) != tt.wantHistory {
				t.Errorf("history count = %d, want %d", len(result.History), tt.wantHistory)
			}
			if len(result.Dependencies) != tt.wantDeps {
				t.Errorf("deps count = %d, want %d", len(result.Dependencies), tt.wantDeps)
			}
		})
	}
}

func TestMarkAsWontMigrate(t *testing.T) {
	tests := []struct {
		name          string
		fullName      string
		repo          *models.Repository
		getRepoErr    error
		updateRepoErr error
		wantErr       bool
		wantStatus    string
	}{
		{
			name:     "repo not found",
			fullName: "org/unknown",
			wantErr:  true,
		},
		{
			name:       "get repo error",
			fullName:   "org/repo1",
			getRepoErr: errors.New("db error"),
			wantErr:    true,
		},
		{
			name:     "cannot mark completed repo",
			fullName: "org/repo1",
			repo:     &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusMigrationComplete)},
			wantErr:  true,
		},
		{
			name:     "cannot mark complete repo",
			fullName: "org/repo1",
			repo:     &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusComplete)},
			wantErr:  true,
		},
		{
			name:       "mark as wont migrate successfully",
			fullName:   "org/repo1",
			repo:       &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
			wantStatus: string(models.StatusWontMigrate),
		},
		{
			name:          "update error",
			fullName:      "org/repo1",
			repo:          &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
			updateRepoErr: errors.New("update error"),
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoStore := NewMockRepoStore()
			if tt.repo != nil {
				repoStore.AddRepo(tt.repo)
			}
			repoStore.getRepoErr = tt.getRepoErr
			repoStore.updateRepoErr = tt.updateRepoErr

			svc := NewRepositoryService(repoStore, NewMockHistoryStore(), NewMockDepStore(), newTestLogger())
			result, err := svc.MarkAsWontMigrate(context.Background(), tt.fullName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestResetToDiscovered(t *testing.T) {
	tests := []struct {
		name          string
		fullName      string
		repo          *models.Repository
		getRepoErr    error
		updateRepoErr error
		wantErr       bool
		wantStatus    string
		wantBatchNil  bool
	}{
		{
			name:     "repo not found",
			fullName: "org/unknown",
			wantErr:  true,
		},
		{
			name:       "get repo error",
			fullName:   "org/repo1",
			getRepoErr: errors.New("db error"),
			wantErr:    true,
		},
		{
			name:     "cannot reset pending repo",
			fullName: "org/repo1",
			repo:     &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
			wantErr:  true,
		},
		{
			name:         "reset wont_migrate repo",
			fullName:     "org/repo1",
			repo:         &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusWontMigrate), BatchID: ptrInt64(1)},
			wantStatus:   string(models.StatusPending),
			wantBatchNil: true,
		},
		{
			name:         "reset failed repo",
			fullName:     "org/repo1",
			repo:         &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusMigrationFailed)},
			wantStatus:   string(models.StatusPending),
			wantBatchNil: true,
		},
		{
			name:         "reset rolled back repo",
			fullName:     "org/repo1",
			repo:         &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusRolledBack)},
			wantStatus:   string(models.StatusPending),
			wantBatchNil: true,
		},
		{
			name:          "update error",
			fullName:      "org/repo1",
			repo:          &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusWontMigrate)},
			updateRepoErr: errors.New("update error"),
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoStore := NewMockRepoStore()
			if tt.repo != nil {
				repoStore.AddRepo(tt.repo)
			}
			repoStore.getRepoErr = tt.getRepoErr
			repoStore.updateRepoErr = tt.updateRepoErr

			svc := NewRepositoryService(repoStore, NewMockHistoryStore(), NewMockDepStore(), newTestLogger())
			result, err := svc.ResetToDiscovered(context.Background(), tt.fullName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", result.Status, tt.wantStatus)
			}
			if tt.wantBatchNil && result.BatchID != nil {
				t.Error("expected BatchID to be nil")
			}
		})
	}
}

func TestCheckBatchEligibility(t *testing.T) {
	tests := []struct {
		name         string
		fullName     string
		repo         *models.Repository
		getRepoErr   error
		wantErr      bool
		wantEligible bool
		wantReason   string
	}{
		{
			name:     "repo not found",
			fullName: "org/unknown",
			wantErr:  true,
		},
		{
			name:       "get repo error",
			fullName:   "org/repo1",
			getRepoErr: errors.New("db error"),
			wantErr:    true,
		},
		{
			name:         "eligible pending repo",
			fullName:     "org/repo1",
			repo:         &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
			wantEligible: true,
		},
		{
			name:         "not eligible - already in batch",
			fullName:     "org/repo1",
			repo:         &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending), BatchID: ptrInt64(1)},
			wantEligible: false,
			wantReason:   "repository is already assigned to a batch",
		},
		{
			name:         "not eligible - oversized",
			fullName:     "org/repo1",
			repo:         &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending), HasOversizedRepository: true},
			wantEligible: false,
			wantReason:   "repository exceeds GitHub's 40 GiB size limit and requires remediation",
		},
		{
			name:         "not eligible - wrong status",
			fullName:     "org/repo1",
			repo:         &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusComplete)},
			wantEligible: false,
			wantReason:   "repository status 'complete' is not eligible for batch assignment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoStore := NewMockRepoStore()
			if tt.repo != nil {
				repoStore.AddRepo(tt.repo)
			}
			repoStore.getRepoErr = tt.getRepoErr

			svc := NewRepositoryService(repoStore, NewMockHistoryStore(), NewMockDepStore(), newTestLogger())
			result, err := svc.CheckBatchEligibility(context.Background(), tt.fullName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Eligible != tt.wantEligible {
				t.Errorf("eligible = %v, want %v", result.Eligible, tt.wantEligible)
			}
			if result.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestGetDependencyChain(t *testing.T) {
	tests := []struct {
		name             string
		fullName         string
		dependents       []*models.Repository
		deps             []*models.RepositoryDependency
		getDependentsErr error
		getDepsByNameErr error
		wantErr          bool
		wantDependents   int
		wantDeps         int
	}{
		{
			name:             "get dependents error",
			fullName:         "org/repo1",
			getDependentsErr: errors.New("db error"),
			wantErr:          true,
		},
		{
			name:             "get deps by name error",
			fullName:         "org/repo1",
			getDepsByNameErr: errors.New("db error"),
			wantErr:          true,
		},
		{
			name:     "no dependencies",
			fullName: "org/repo1",
		},
		{
			name:     "with dependents and dependencies",
			fullName: "org/repo1",
			dependents: []*models.Repository{
				{FullName: "org/dependent1"},
				{FullName: "org/dependent2"},
			},
			deps: []*models.RepositoryDependency{
				{DependencyFullName: "org/dep1", IsLocal: true},
				{DependencyFullName: "org/dep2", IsLocal: true},
				{DependencyFullName: "external/dep", IsLocal: false}, // Should be excluded
			},
			wantDependents: 2,
			wantDeps:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoStore := NewMockRepoStore()
			depStore := NewMockDepStore()

			depStore.dependents[tt.fullName] = tt.dependents
			depStore.depsByFullName[tt.fullName] = tt.deps
			depStore.getDependentsErr = tt.getDependentsErr
			depStore.getDepsByNameErr = tt.getDepsByNameErr

			svc := NewRepositoryService(repoStore, NewMockHistoryStore(), depStore, newTestLogger())
			dependents, deps, err := svc.GetDependencyChain(context.Background(), tt.fullName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(dependents) != tt.wantDependents {
				t.Errorf("dependents count = %d, want %d", len(dependents), tt.wantDependents)
			}
			if len(deps) != tt.wantDeps {
				t.Errorf("deps count = %d, want %d", len(deps), tt.wantDeps)
			}
		})
	}
}
