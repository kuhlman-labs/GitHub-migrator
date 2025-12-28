package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// BatchService encapsulates business logic for batch operations.
// It coordinates between repositories and batches, applying business rules.
type BatchService struct {
	batchStore storage.BatchStore
	repoStore  storage.RepositoryStore
	logger     *slog.Logger
}

// NewBatchService creates a new BatchService with the required dependencies.
func NewBatchService(
	batchStore storage.BatchStore,
	repoStore storage.RepositoryStore,
	logger *slog.Logger,
) *BatchService {
	return &BatchService{
		batchStore: batchStore,
		repoStore:  repoStore,
		logger:     logger,
	}
}

// BatchWithStats contains a batch with computed statistics.
type BatchWithStats struct {
	Batch           *models.Batch `json:"batch"`
	RepositoryCount int           `json:"repository_count"`
	CompletedCount  int           `json:"completed_count"`
	InProgressCount int           `json:"in_progress_count"`
	PendingCount    int           `json:"pending_count"`
	FailedCount     int           `json:"failed_count"`
	ProgressPercent float64       `json:"progress_percent"`
}

// GetBatchWithStats retrieves a batch with its computed statistics.
func (s *BatchService) GetBatchWithStats(ctx context.Context, batchID int64) (*BatchWithStats, error) {
	batch, err := s.batchStore.GetBatch(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return nil, nil
	}

	// Get repositories in this batch
	repos, err := s.repoStore.ListRepositories(ctx, map[string]any{
		"batch_id": fmt.Sprintf("%d", batchID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list batch repositories: %w", err)
	}

	result := &BatchWithStats{Batch: batch}
	result.RepositoryCount = len(repos)

	// Count by status
	for _, repo := range repos {
		switch repo.Status {
		case string(models.StatusComplete), string(models.StatusMigrationComplete):
			result.CompletedCount++
		case string(models.StatusPreMigration), string(models.StatusArchiveGenerating),
			string(models.StatusQueuedForMigration), string(models.StatusMigratingContent),
			string(models.StatusPostMigration):
			result.InProgressCount++
		case string(models.StatusPending), string(models.StatusDryRunQueued),
			string(models.StatusDryRunInProgress), string(models.StatusDryRunComplete):
			result.PendingCount++
		case string(models.StatusMigrationFailed), string(models.StatusRolledBack):
			result.FailedCount++
		}
	}

	// Calculate progress
	if result.RepositoryCount > 0 {
		result.ProgressPercent = float64(result.CompletedCount) / float64(result.RepositoryCount) * 100
	}

	return result, nil
}

// AddRepositoryResult represents the result of adding a repository to a batch.
type AddRepositoryResult struct {
	FullName string `json:"full_name"`
	Added    bool   `json:"added"`
	Reason   string `json:"reason,omitempty"`
}

// AddRepositoriesToBatch adds multiple repositories to a batch.
// Returns detailed results for each repository.
func (s *BatchService) AddRepositoriesToBatch(ctx context.Context, batchID int64, repoIDs []int64) ([]AddRepositoryResult, error) {
	batch, err := s.batchStore.GetBatch(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return nil, fmt.Errorf("batch not found: %d", batchID)
	}

	// Only allow adding to pending/ready batches
	if batch.Status != models.BatchStatusPending && batch.Status != models.BatchStatusReady {
		return nil, fmt.Errorf("cannot add repositories to batch with status '%s'", batch.Status)
	}

	results := make([]AddRepositoryResult, 0, len(repoIDs))

	for _, repoID := range repoIDs {
		result := AddRepositoryResult{}

		repo, err := s.repoStore.GetRepositoryByID(ctx, repoID)
		if err != nil || repo == nil {
			result.FullName = fmt.Sprintf("ID:%d", repoID)
			result.Reason = "repository not found"
			results = append(results, result)
			continue
		}

		result.FullName = repo.FullName

		// Check if already in a different batch
		if repo.BatchID != nil && *repo.BatchID != batchID {
			result.Reason = "repository is already in another batch"
			results = append(results, result)
			continue
		}

		// Check eligibility
		eligible, reason := s.checkRepoEligibility(repo)
		if !eligible {
			result.Reason = reason
			results = append(results, result)
			continue
		}

		// Add to batch
		repo.BatchID = &batchID
		if err := s.repoStore.UpdateRepository(ctx, repo); err != nil {
			result.Reason = "failed to update repository"
			s.logger.Error("Failed to add repo to batch", "repo_id", repoID, "batch_id", batchID, "error", err)
			results = append(results, result)
			continue
		}

		result.Added = true
		results = append(results, result)
	}

	return results, nil
}

// RemoveRepositoriesFromBatch removes repositories from a batch.
func (s *BatchService) RemoveRepositoriesFromBatch(ctx context.Context, batchID int64, repoIDs []int64) (int, error) {
	batch, err := s.batchStore.GetBatch(ctx, batchID)
	if err != nil {
		return 0, fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return 0, fmt.Errorf("batch not found: %d", batchID)
	}

	// Only allow removing from pending/ready batches
	if batch.Status != models.BatchStatusPending && batch.Status != models.BatchStatusReady {
		return 0, fmt.Errorf("cannot remove repositories from batch with status '%s'", batch.Status)
	}

	removed := 0
	for _, repoID := range repoIDs {
		repo, err := s.repoStore.GetRepositoryByID(ctx, repoID)
		if err != nil || repo == nil {
			continue
		}

		if repo.BatchID == nil || *repo.BatchID != batchID {
			continue
		}

		repo.BatchID = nil
		if err := s.repoStore.UpdateRepository(ctx, repo); err != nil {
			s.logger.Error("Failed to remove repo from batch", "repo_id", repoID, "batch_id", batchID, "error", err)
			continue
		}

		removed++
	}

	return removed, nil
}

// CanDeleteBatch checks if a batch can be deleted.
func (s *BatchService) CanDeleteBatch(ctx context.Context, batchID int64) (bool, string, error) {
	batch, err := s.batchStore.GetBatch(ctx, batchID)
	if err != nil {
		return false, "", fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return false, "batch not found", nil
	}

	// Only pending batches can be deleted
	if batch.Status != models.BatchStatusPending {
		return false, fmt.Sprintf("cannot delete batch with status '%s'", batch.Status), nil
	}

	return true, "", nil
}

// DeleteBatch deletes a batch and removes all repositories from it.
func (s *BatchService) DeleteBatch(ctx context.Context, batchID int64) error {
	canDelete, reason, err := s.CanDeleteBatch(ctx, batchID)
	if err != nil {
		return err
	}
	if !canDelete {
		return fmt.Errorf("cannot delete batch: %s", reason)
	}

	// Remove repositories from batch first
	repos, err := s.repoStore.ListRepositories(ctx, map[string]any{
		"batch_id": fmt.Sprintf("%d", batchID),
	})
	if err != nil {
		return fmt.Errorf("failed to list batch repositories: %w", err)
	}

	for _, repo := range repos {
		repo.BatchID = nil
		if err := s.repoStore.UpdateRepository(ctx, repo); err != nil {
			s.logger.Error("Failed to remove repo from batch during delete",
				"repo_id", repo.ID, "batch_id", batchID, "error", err)
		}
	}

	// Delete the batch
	if err := s.batchStore.DeleteBatch(ctx, batchID); err != nil {
		return fmt.Errorf("failed to delete batch: %w", err)
	}

	s.logger.Info("Batch deleted", "batch_id", batchID, "repos_removed", len(repos))
	return nil
}

// StartBatch transitions a batch to in_progress and records the start time.
func (s *BatchService) StartBatch(ctx context.Context, batchID int64) (*models.Batch, error) {
	batch, err := s.batchStore.GetBatch(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return nil, fmt.Errorf("batch not found: %d", batchID)
	}

	// Validate current status
	if batch.Status != models.BatchStatusPending && batch.Status != models.BatchStatusReady {
		return nil, fmt.Errorf("cannot start batch with status '%s'", batch.Status)
	}

	// Update status and start time
	now := time.Now().UTC()
	batch.Status = models.BatchStatusInProgress
	batch.StartedAt = &now

	if err := s.batchStore.UpdateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to update batch: %w", err)
	}

	s.logger.Info("Batch started", "batch_id", batchID)
	return batch, nil
}

// checkRepoEligibility checks if a repository is eligible for batch assignment.
func (s *BatchService) checkRepoEligibility(repo *models.Repository) (bool, string) {
	// Check for oversized repository
	if repo.HasOversizedRepository {
		return false, "repository exceeds GitHub's 40 GiB size limit"
	}

	// Check status eligibility
	eligibleStatuses := map[string]bool{
		string(models.StatusPending):         true,
		string(models.StatusDryRunComplete):  true,
		string(models.StatusDryRunFailed):    true,
		string(models.StatusMigrationFailed): true,
		string(models.StatusRolledBack):      true,
	}

	if !eligibleStatuses[repo.Status] {
		return false, fmt.Sprintf("status '%s' is not eligible", repo.Status)
	}

	return true, ""
}
