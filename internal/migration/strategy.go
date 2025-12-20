package migration

import (
	"context"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// MigrationStrategy defines the interface for source-specific migration implementations.
// The strategy pattern allows for clean separation of GitHub and ADO migration logic
// while sharing common infrastructure like polling, logging, and error handling.
type MigrationStrategy interface {
	// Name returns the strategy name for logging and identification.
	Name() string

	// SupportsRepository returns true if this strategy can handle the given repository.
	SupportsRepository(repo *models.Repository) bool

	// ValidateSource performs source-specific validation before migration.
	// For GitHub: verifies the source client is configured
	// For ADO: validates the ADO PAT can access the source repository
	ValidateSource(ctx context.Context, repo *models.Repository) error

	// PrepareArchives handles source-specific archive preparation.
	// For GitHub: generates git and metadata archives on GHES
	// For ADO: no-op since GEI pulls directly from ADO
	PrepareArchives(ctx context.Context, mc *MigrationContext) error

	// StartMigration initiates the migration on the destination.
	// For GitHub: starts migration using archive URLs
	// For ADO: starts migration with ADO-specific parameters
	StartMigration(ctx context.Context, mc *MigrationContext) (string, error)

	// ShouldUnlockSource returns true if the source repository should be unlocked after migration.
	ShouldUnlockSource() bool
}

// StrategyRegistry manages the available migration strategies.
type StrategyRegistry struct {
	strategies []MigrationStrategy
}

// NewStrategyRegistry creates a new strategy registry with the given strategies.
func NewStrategyRegistry(strategies ...MigrationStrategy) *StrategyRegistry {
	return &StrategyRegistry{
		strategies: strategies,
	}
}

// GetStrategy returns the appropriate strategy for the given repository.
// Returns nil if no suitable strategy is found.
func (r *StrategyRegistry) GetStrategy(repo *models.Repository) MigrationStrategy {
	for _, s := range r.strategies {
		if s.SupportsRepository(repo) {
			return s
		}
	}
	return nil
}

// RegisterStrategy adds a new strategy to the registry.
func (r *StrategyRegistry) RegisterStrategy(strategy MigrationStrategy) {
	r.strategies = append(r.strategies, strategy)
}
