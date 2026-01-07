package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// MockBatchStore is a mock implementation of storage.BatchStore for testing.
type MockBatchStore struct {
	batches        map[int64]*models.Batch
	getBatchErr    error
	updateBatchErr error
	deleteBatchErr error
}

func NewMockBatchStore() *MockBatchStore {
	return &MockBatchStore{
		batches: make(map[int64]*models.Batch),
	}
}

func (m *MockBatchStore) GetBatch(_ context.Context, id int64) (*models.Batch, error) {
	if m.getBatchErr != nil {
		return nil, m.getBatchErr
	}
	return m.batches[id], nil
}

func (m *MockBatchStore) ListBatches(_ context.Context) ([]*models.Batch, error) {
	result := make([]*models.Batch, 0, len(m.batches))
	for _, b := range m.batches {
		result = append(result, b)
	}
	return result, nil
}

func (m *MockBatchStore) CreateBatch(_ context.Context, batch *models.Batch) error {
	if batch.ID == 0 {
		batch.ID = int64(len(m.batches) + 1)
	}
	m.batches[batch.ID] = batch
	return nil
}

func (m *MockBatchStore) UpdateBatch(_ context.Context, batch *models.Batch) error {
	if m.updateBatchErr != nil {
		return m.updateBatchErr
	}
	m.batches[batch.ID] = batch
	return nil
}

func (m *MockBatchStore) DeleteBatch(_ context.Context, batchID int64) error {
	if m.deleteBatchErr != nil {
		return m.deleteBatchErr
	}
	delete(m.batches, batchID)
	return nil
}

func (m *MockBatchStore) AddRepositoriesToBatch(_ context.Context, _ int64, _ []int64) error {
	return nil
}

func (m *MockBatchStore) RemoveRepositoriesFromBatch(_ context.Context, _ int64, _ []int64) error {
	return nil
}

func (m *MockBatchStore) UpdateBatchProgress(_ context.Context, _ int64, _ string, _, _, _, _ *time.Time) error {
	return nil
}

// MockRepoStore is a mock implementation of storage.RepositoryStore for testing.
type MockRepoStore struct {
	repos           map[string]*models.Repository
	reposByID       map[int64]*models.Repository
	getRepoErr      error
	getRepoByIDErr  error
	listReposErr    error
	updateRepoErr   error
	listReposResult []*models.Repository
}

func NewMockRepoStore() *MockRepoStore {
	return &MockRepoStore{
		repos:     make(map[string]*models.Repository),
		reposByID: make(map[int64]*models.Repository),
	}
}

func (m *MockRepoStore) GetRepository(_ context.Context, fullName string) (*models.Repository, error) {
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	return m.repos[fullName], nil
}

func (m *MockRepoStore) GetRepositoryByID(_ context.Context, id int64) (*models.Repository, error) {
	if m.getRepoByIDErr != nil {
		return nil, m.getRepoByIDErr
	}
	return m.reposByID[id], nil
}

func (m *MockRepoStore) GetRepositoriesByIDs(_ context.Context, ids []int64) ([]*models.Repository, error) {
	var result []*models.Repository
	for _, id := range ids {
		if repo := m.reposByID[id]; repo != nil {
			result = append(result, repo)
		}
	}
	return result, nil
}

func (m *MockRepoStore) GetRepositoriesByNames(_ context.Context, names []string) ([]*models.Repository, error) {
	var result []*models.Repository
	for _, name := range names {
		if repo := m.repos[name]; repo != nil {
			result = append(result, repo)
		}
	}
	return result, nil
}

func (m *MockRepoStore) ListRepositories(_ context.Context, _ map[string]any) ([]*models.Repository, error) {
	if m.listReposErr != nil {
		return nil, m.listReposErr
	}
	if m.listReposResult != nil {
		return m.listReposResult, nil
	}
	result := make([]*models.Repository, 0, len(m.repos))
	for _, r := range m.repos {
		result = append(result, r)
	}
	return result, nil
}

func (m *MockRepoStore) CountRepositories(_ context.Context, _ map[string]any) (int, error) {
	return len(m.repos), nil
}

func (m *MockRepoStore) CountRepositoriesWithFilters(_ context.Context, _ map[string]any) (int, error) {
	return len(m.repos), nil
}

func (m *MockRepoStore) SaveRepository(_ context.Context, repo *models.Repository) error {
	if repo.ID == 0 {
		repo.ID = int64(len(m.repos) + 1)
	}
	m.repos[repo.FullName] = repo
	m.reposByID[repo.ID] = repo
	return nil
}

func (m *MockRepoStore) UpdateRepository(_ context.Context, repo *models.Repository) error {
	if m.updateRepoErr != nil {
		return m.updateRepoErr
	}
	m.repos[repo.FullName] = repo
	m.reposByID[repo.ID] = repo
	return nil
}

func (m *MockRepoStore) UpdateRepositoryStatus(_ context.Context, fullName string, status models.MigrationStatus) error {
	if repo := m.repos[fullName]; repo != nil {
		repo.Status = string(status)
	}
	return nil
}

func (m *MockRepoStore) DeleteRepository(_ context.Context, fullName string) error {
	if repo := m.repos[fullName]; repo != nil {
		delete(m.reposByID, repo.ID)
	}
	delete(m.repos, fullName)
	return nil
}

func (m *MockRepoStore) RollbackRepository(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *MockRepoStore) UpdateLocalDependencyFlags(_ context.Context) error {
	return nil
}

func (m *MockRepoStore) AddRepo(repo *models.Repository) {
	if repo.ID == 0 {
		repo.ID = int64(len(m.repos) + 1)
	}
	m.repos[repo.FullName] = repo
	m.reposByID[repo.ID] = repo
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewBatchService(t *testing.T) {
	batchStore := NewMockBatchStore()
	repoStore := NewMockRepoStore()
	logger := newTestLogger()

	svc := NewBatchService(batchStore, repoStore, logger)
	if svc == nil {
		t.Fatal("expected non-nil BatchService")
		return
	}
	if svc.batchStore == nil {
		t.Error("batchStore not set")
	}
	if svc.repoStore == nil {
		t.Error("repoStore not set")
	}
	if svc.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestGetBatchWithStats(t *testing.T) {
	tests := []struct {
		name           string
		batchID        int64
		batch          *models.Batch
		repos          []*models.Repository
		getBatchErr    error
		listReposErr   error
		wantNil        bool
		wantErr        bool
		wantCompleted  int
		wantInProgress int
		wantPending    int
		wantFailed     int
	}{
		{
			name:    "batch not found",
			batchID: 999,
			wantNil: true,
			wantErr: false,
		},
		{
			name:        "get batch error",
			batchID:     1,
			getBatchErr: errors.New("database error"),
			wantErr:     true,
		},
		{
			name:    "empty batch",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Name: "Test Batch", Status: models.BatchStatusPending},
			repos:   []*models.Repository{},
		},
		{
			name:    "batch with mixed status repos",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Name: "Test Batch", Status: models.BatchStatusInProgress},
			repos: []*models.Repository{
				{ID: 1, FullName: "org/repo1", Status: string(models.StatusComplete)},
				{ID: 2, FullName: "org/repo2", Status: string(models.StatusMigrationComplete)},
				{ID: 3, FullName: "org/repo3", Status: string(models.StatusPreMigration)},
				{ID: 4, FullName: "org/repo4", Status: string(models.StatusArchiveGenerating)},
				{ID: 5, FullName: "org/repo5", Status: string(models.StatusPending)},
				{ID: 6, FullName: "org/repo6", Status: string(models.StatusDryRunComplete)},
				{ID: 7, FullName: "org/repo7", Status: string(models.StatusMigrationFailed)},
				{ID: 8, FullName: "org/repo8", Status: string(models.StatusRolledBack)},
			},
			wantCompleted:  2,
			wantInProgress: 2,
			wantPending:    2,
			wantFailed:     2,
		},
		{
			name:         "list repos error",
			batchID:      1,
			batch:        &models.Batch{ID: 1, Name: "Test Batch"},
			listReposErr: errors.New("list error"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batchStore := NewMockBatchStore()
			repoStore := NewMockRepoStore()

			if tt.batch != nil {
				batchStore.batches[tt.batch.ID] = tt.batch
			}
			batchStore.getBatchErr = tt.getBatchErr

			repoStore.listReposResult = tt.repos
			repoStore.listReposErr = tt.listReposErr

			svc := NewBatchService(batchStore, repoStore, newTestLogger())
			result, err := svc.GetBatchWithStats(context.Background(), tt.batchID)

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

			if result.CompletedCount != tt.wantCompleted {
				t.Errorf("CompletedCount = %d, want %d", result.CompletedCount, tt.wantCompleted)
			}
			if result.InProgressCount != tt.wantInProgress {
				t.Errorf("InProgressCount = %d, want %d", result.InProgressCount, tt.wantInProgress)
			}
			if result.PendingCount != tt.wantPending {
				t.Errorf("PendingCount = %d, want %d", result.PendingCount, tt.wantPending)
			}
			if result.FailedCount != tt.wantFailed {
				t.Errorf("FailedCount = %d, want %d", result.FailedCount, tt.wantFailed)
			}
		})
	}
}

func TestAddRepositoriesToBatch(t *testing.T) {
	tests := []struct {
		name        string
		batchID     int64
		batch       *models.Batch
		repos       []*models.Repository
		repoIDs     []int64
		getBatchErr error
		wantErr     bool
		wantAdded   int
	}{
		{
			name:    "batch not found",
			batchID: 999,
			repoIDs: []int64{1},
			wantErr: true,
		},
		{
			name:        "get batch error",
			batchID:     1,
			getBatchErr: errors.New("db error"),
			wantErr:     true,
		},
		{
			name:    "batch not in pending/ready status",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusInProgress},
			repoIDs: []int64{1},
			wantErr: true,
		},
		{
			name:    "add repos successfully",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos: []*models.Repository{
				{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)},
				{ID: 2, FullName: "org/repo2", Status: string(models.StatusDryRunComplete)},
			},
			repoIDs:   []int64{1, 2},
			wantAdded: 2,
		},
		{
			name:      "repo not found",
			batchID:   1,
			batch:     &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos:     []*models.Repository{},
			repoIDs:   []int64{999},
			wantAdded: 0,
		},
		{
			name:    "repo already in another batch",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos: []*models.Repository{
				{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending), BatchID: ptrInt64(2)},
			},
			repoIDs:   []int64{1},
			wantAdded: 0,
		},
		{
			name:    "repo not eligible - oversized",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos: func() []*models.Repository {
				r := &models.Repository{ID: 1, FullName: "org/repo1", Status: string(models.StatusPending)}
				r.SetHasOversizedRepository(true)
				return []*models.Repository{r}
			}(),
			repoIDs:   []int64{1},
			wantAdded: 0,
		},
		{
			name:    "repo not eligible - wrong status",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos: []*models.Repository{
				{ID: 1, FullName: "org/repo1", Status: string(models.StatusComplete)},
			},
			repoIDs:   []int64{1},
			wantAdded: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batchStore := NewMockBatchStore()
			repoStore := NewMockRepoStore()

			if tt.batch != nil {
				batchStore.batches[tt.batch.ID] = tt.batch
			}
			batchStore.getBatchErr = tt.getBatchErr

			for _, repo := range tt.repos {
				repoStore.AddRepo(repo)
			}

			svc := NewBatchService(batchStore, repoStore, newTestLogger())
			results, err := svc.AddRepositoriesToBatch(context.Background(), tt.batchID, tt.repoIDs)

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

			addedCount := 0
			for _, r := range results {
				if r.Added {
					addedCount++
				}
			}
			if addedCount != tt.wantAdded {
				t.Errorf("added = %d, want %d", addedCount, tt.wantAdded)
			}
		})
	}
}

func TestRemoveRepositoriesFromBatch(t *testing.T) {
	tests := []struct {
		name        string
		batchID     int64
		batch       *models.Batch
		repos       []*models.Repository
		repoIDs     []int64
		getBatchErr error
		wantErr     bool
		wantRemoved int
	}{
		{
			name:    "batch not found",
			batchID: 999,
			repoIDs: []int64{1},
			wantErr: true,
		},
		{
			name:    "batch not in pending/ready status",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusInProgress},
			repoIDs: []int64{1},
			wantErr: true,
		},
		{
			name:    "remove repos successfully",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos: []*models.Repository{
				{ID: 1, FullName: "org/repo1", BatchID: ptrInt64(1)},
				{ID: 2, FullName: "org/repo2", BatchID: ptrInt64(1)},
			},
			repoIDs:     []int64{1, 2},
			wantRemoved: 2,
		},
		{
			name:    "repo not in this batch",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos: []*models.Repository{
				{ID: 1, FullName: "org/repo1", BatchID: ptrInt64(2)},
			},
			repoIDs:     []int64{1},
			wantRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batchStore := NewMockBatchStore()
			repoStore := NewMockRepoStore()

			if tt.batch != nil {
				batchStore.batches[tt.batch.ID] = tt.batch
			}
			batchStore.getBatchErr = tt.getBatchErr

			for _, repo := range tt.repos {
				repoStore.AddRepo(repo)
			}

			svc := NewBatchService(batchStore, repoStore, newTestLogger())
			removed, err := svc.RemoveRepositoriesFromBatch(context.Background(), tt.batchID, tt.repoIDs)

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

			if removed != tt.wantRemoved {
				t.Errorf("removed = %d, want %d", removed, tt.wantRemoved)
			}
		})
	}
}

func TestCanDeleteBatch(t *testing.T) {
	tests := []struct {
		name        string
		batchID     int64
		batch       *models.Batch
		getBatchErr error
		wantCan     bool
		wantReason  string
		wantErr     bool
	}{
		{
			name:        "get batch error",
			batchID:     1,
			getBatchErr: errors.New("db error"),
			wantErr:     true,
		},
		{
			name:       "batch not found",
			batchID:    999,
			wantCan:    false,
			wantReason: "batch not found",
		},
		{
			name:    "can delete pending batch",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			wantCan: true,
		},
		{
			name:       "cannot delete in_progress batch",
			batchID:    1,
			batch:      &models.Batch{ID: 1, Status: models.BatchStatusInProgress},
			wantCan:    false,
			wantReason: "cannot delete batch with status 'in_progress'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batchStore := NewMockBatchStore()
			if tt.batch != nil {
				batchStore.batches[tt.batch.ID] = tt.batch
			}
			batchStore.getBatchErr = tt.getBatchErr

			svc := NewBatchService(batchStore, NewMockRepoStore(), newTestLogger())
			canDelete, reason, err := svc.CanDeleteBatch(context.Background(), tt.batchID)

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

			if canDelete != tt.wantCan {
				t.Errorf("canDelete = %v, want %v", canDelete, tt.wantCan)
			}
			if reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestDeleteBatch(t *testing.T) {
	tests := []struct {
		name           string
		batchID        int64
		batch          *models.Batch
		repos          []*models.Repository
		deleteBatchErr error
		wantErr        bool
	}{
		{
			name:    "batch not found",
			batchID: 999,
			wantErr: true,
		},
		{
			name:    "cannot delete non-pending batch",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusInProgress},
			wantErr: true,
		},
		{
			name:    "delete batch successfully",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusPending},
			repos: []*models.Repository{
				{ID: 1, FullName: "org/repo1", BatchID: ptrInt64(1)},
			},
		},
		{
			name:           "delete batch store error",
			batchID:        1,
			batch:          &models.Batch{ID: 1, Status: models.BatchStatusPending},
			deleteBatchErr: errors.New("delete error"),
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batchStore := NewMockBatchStore()
			repoStore := NewMockRepoStore()

			if tt.batch != nil {
				batchStore.batches[tt.batch.ID] = tt.batch
			}
			batchStore.deleteBatchErr = tt.deleteBatchErr

			repoStore.listReposResult = tt.repos
			for _, repo := range tt.repos {
				repoStore.AddRepo(repo)
			}

			svc := NewBatchService(batchStore, repoStore, newTestLogger())
			err := svc.DeleteBatch(context.Background(), tt.batchID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStartBatch(t *testing.T) {
	tests := []struct {
		name           string
		batchID        int64
		batch          *models.Batch
		getBatchErr    error
		updateBatchErr error
		wantErr        bool
		wantStatus     string
	}{
		{
			name:    "batch not found",
			batchID: 999,
			wantErr: true,
		},
		{
			name:        "get batch error",
			batchID:     1,
			getBatchErr: errors.New("db error"),
			wantErr:     true,
		},
		{
			name:    "cannot start non-pending batch",
			batchID: 1,
			batch:   &models.Batch{ID: 1, Status: models.BatchStatusCompleted},
			wantErr: true,
		},
		{
			name:       "start pending batch",
			batchID:    1,
			batch:      &models.Batch{ID: 1, Status: models.BatchStatusPending},
			wantStatus: models.BatchStatusInProgress,
		},
		{
			name:       "start ready batch",
			batchID:    1,
			batch:      &models.Batch{ID: 1, Status: models.BatchStatusReady},
			wantStatus: models.BatchStatusInProgress,
		},
		{
			name:           "update batch error",
			batchID:        1,
			batch:          &models.Batch{ID: 1, Status: models.BatchStatusPending},
			updateBatchErr: errors.New("update error"),
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batchStore := NewMockBatchStore()
			if tt.batch != nil {
				batchStore.batches[tt.batch.ID] = tt.batch
			}
			batchStore.getBatchErr = tt.getBatchErr
			batchStore.updateBatchErr = tt.updateBatchErr

			svc := NewBatchService(batchStore, NewMockRepoStore(), newTestLogger())
			result, err := svc.StartBatch(context.Background(), tt.batchID)

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
			if result.StartedAt == nil {
				t.Error("StartedAt should be set")
			}
		})
	}
}

func TestCheckRepoEligibility(t *testing.T) {
	tests := []struct {
		name         string
		repo         *models.Repository
		wantEligible bool
		wantReason   string
	}{
		{
			name:         "eligible pending repo",
			repo:         &models.Repository{Status: string(models.StatusPending)},
			wantEligible: true,
		},
		{
			name:         "eligible dry run complete repo",
			repo:         &models.Repository{Status: string(models.StatusDryRunComplete)},
			wantEligible: true,
		},
		{
			name:         "eligible failed repo",
			repo:         &models.Repository{Status: string(models.StatusMigrationFailed)},
			wantEligible: true,
		},
		{
			name: "not eligible - oversized",
			repo: func() *models.Repository {
				r := &models.Repository{Status: string(models.StatusPending)}
				r.SetHasOversizedRepository(true)
				return r
			}(),
			wantEligible: false,
			wantReason:   "repository exceeds GitHub's 40 GiB size limit",
		},
		{
			name:         "not eligible - wrong status",
			repo:         &models.Repository{Status: string(models.StatusComplete)},
			wantEligible: false,
			wantReason:   fmt.Sprintf("status '%s' is not eligible", models.StatusComplete),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewBatchService(NewMockBatchStore(), NewMockRepoStore(), newTestLogger())
			eligible, reason := svc.checkRepoEligibility(tt.repo)

			if eligible != tt.wantEligible {
				t.Errorf("eligible = %v, want %v", eligible, tt.wantEligible)
			}
			if reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

// Helper function to create pointer to int64
func ptrInt64(i int64) *int64 {
	return &i
}
