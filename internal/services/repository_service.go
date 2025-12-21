// Package services contains business logic services that orchestrate domain operations.
// Services encapsulate complex business rules and workflows separate from HTTP handling
// and data access, making the logic reusable and testable.
package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// RepositoryService encapsulates business logic for repository operations.
// It coordinates between different storage interfaces and applies business rules.
type RepositoryService struct {
	repoStore    storage.RepositoryStore
	historyStore storage.MigrationHistoryStore
	depStore     storage.DependencyStore
	logger       *slog.Logger
}

// NewRepositoryService creates a new RepositoryService with the required dependencies.
func NewRepositoryService(
	repoStore storage.RepositoryStore,
	historyStore storage.MigrationHistoryStore,
	depStore storage.DependencyStore,
	logger *slog.Logger,
) *RepositoryService {
	return &RepositoryService{
		repoStore:    repoStore,
		historyStore: historyStore,
		depStore:     depStore,
		logger:       logger,
	}
}

// RepositoryWithDetails contains a repository along with its related data.
type RepositoryWithDetails struct {
	Repository   *models.Repository             `json:"repository"`
	History      []*models.MigrationHistory     `json:"history,omitempty"`
	Dependencies []*models.RepositoryDependency `json:"dependencies,omitempty"`
}

// GetRepositoryWithDetails retrieves a repository along with its migration history and dependencies.
func (s *RepositoryService) GetRepositoryWithDetails(ctx context.Context, fullName string) (*RepositoryWithDetails, error) {
	repo, err := s.repoStore.GetRepository(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return nil, nil // Not found
	}

	result := &RepositoryWithDetails{Repository: repo}

	// Get migration history
	history, err := s.historyStore.GetMigrationHistory(ctx, repo.ID)
	if err != nil {
		s.logger.Warn("Failed to get migration history", "repo_id", repo.ID, "error", err)
		// Continue without history
	} else {
		result.History = history
	}

	// Get dependencies
	deps, err := s.depStore.GetRepositoryDependencies(ctx, repo.ID)
	if err != nil {
		s.logger.Warn("Failed to get dependencies", "repo_id", repo.ID, "error", err)
		// Continue without dependencies
	} else {
		result.Dependencies = deps
	}

	return result, nil
}

// MarkAsWontMigrate marks a repository as won't migrate and validates the operation.
func (s *RepositoryService) MarkAsWontMigrate(ctx context.Context, fullName string) (*models.Repository, error) {
	repo, err := s.repoStore.GetRepository(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found: %s", fullName)
	}

	// Validate status transition
	if repo.Status == string(models.StatusMigrationComplete) ||
		repo.Status == string(models.StatusComplete) {
		return nil, fmt.Errorf("cannot mark completed repository as won't migrate")
	}

	// Update status
	repo.Status = string(models.StatusWontMigrate)
	if err := s.repoStore.UpdateRepository(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	s.logger.Info("Repository marked as won't migrate", "full_name", fullName)
	return repo, nil
}

// ResetToDiscovered resets a repository's status back to discovered state.
// This is useful for repositories that need to be re-evaluated after remediation.
func (s *RepositoryService) ResetToDiscovered(ctx context.Context, fullName string) (*models.Repository, error) {
	repo, err := s.repoStore.GetRepository(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found: %s", fullName)
	}

	// Only allow reset from certain states
	allowedStates := map[string]bool{
		string(models.StatusWontMigrate):     true,
		string(models.StatusMigrationFailed): true,
		string(models.StatusRolledBack):      true,
	}

	if !allowedStates[repo.Status] {
		return nil, fmt.Errorf("cannot reset repository from status '%s'", repo.Status)
	}

	// Reset to pending
	repo.Status = string(models.StatusPending)
	repo.BatchID = nil // Remove from any batch

	if err := s.repoStore.UpdateRepository(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	s.logger.Info("Repository reset to pending", "full_name", fullName)
	return repo, nil
}

// BatchEligibilityResult contains the result of checking batch eligibility.
type BatchEligibilityResult struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

// CheckBatchEligibility checks if a repository is eligible for batch assignment.
func (s *RepositoryService) CheckBatchEligibility(ctx context.Context, fullName string) (*BatchEligibilityResult, error) {
	repo, err := s.repoStore.GetRepository(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found: %s", fullName)
	}

	// Check if already in a batch
	if repo.BatchID != nil {
		return &BatchEligibilityResult{
			Eligible: false,
			Reason:   "repository is already assigned to a batch",
		}, nil
	}

	// Check for oversized repository
	if repo.HasOversizedRepository {
		return &BatchEligibilityResult{
			Eligible: false,
			Reason:   "repository exceeds GitHub's 40 GiB size limit and requires remediation",
		}, nil
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
		return &BatchEligibilityResult{
			Eligible: false,
			Reason:   fmt.Sprintf("repository status '%s' is not eligible for batch assignment", repo.Status),
		}, nil
	}

	return &BatchEligibilityResult{Eligible: true}, nil
}

// GetDependencyChain returns repositories that depend on the given repository
// and that it depends on, useful for migration planning.
func (s *RepositoryService) GetDependencyChain(ctx context.Context, fullName string) ([]string, []string, error) {
	// Get repos that depend on this one
	dependents, err := s.depStore.GetDependentRepositories(ctx, fullName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get dependent repositories: %w", err)
	}

	dependentNames := make([]string, 0, len(dependents))
	for _, r := range dependents {
		dependentNames = append(dependentNames, r.FullName)
	}

	// Get what this repo depends on
	deps, err := s.depStore.GetRepositoryDependenciesByFullName(ctx, fullName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	var dependencyNames []string
	for _, d := range deps {
		if d.IsLocal {
			dependencyNames = append(dependencyNames, d.DependencyFullName)
		}
	}

	return dependentNames, dependencyNames, nil
}
