package migration

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// MigrationConfigProvider provides migration configuration dynamically.
// This interface allows the factory to read settings at execution time
// rather than caching them at creation time.
type MigrationConfigProvider interface {
	// GetDestRepoExistsAction returns the current action for existing destination repos
	GetDestRepoExistsAction() string
	// GetVisibilityPublic returns how to handle public repos
	GetVisibilityPublic() string
	// GetVisibilityInternal returns how to handle internal repos
	GetVisibilityInternal() string
}

// ExecutorFactory creates and caches source-specific migration executors.
// It enables multi-source migrations by dynamically creating executors
// based on each repository's source_id.
type ExecutorFactory struct {
	storage           *storage.Database
	destClient        *github.Client
	logger            *slog.Logger
	postMigrationMode PostMigrationMode
	configProvider    MigrationConfigProvider // Dynamic config provider (optional)

	// Static fallback values used when no configProvider is set
	staticDestRepoExistsAction DestinationRepoExistsAction
	staticVisibilityHandling   VisibilityHandling

	// Cache of executors per source ID
	executorCache map[int64]*Executor
	cacheMu       sync.RWMutex
}

// ExecutorFactoryConfig configures the executor factory
type ExecutorFactoryConfig struct {
	Storage              *storage.Database
	DestClient           *github.Client
	Logger               *slog.Logger
	PostMigrationMode    PostMigrationMode
	DestRepoExistsAction DestinationRepoExistsAction
	VisibilityHandling   VisibilityHandling
	ConfigProvider       MigrationConfigProvider // Optional: provides dynamic settings
}

// NewExecutorFactory creates a new executor factory
func NewExecutorFactory(cfg ExecutorFactoryConfig) (*ExecutorFactory, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.DestClient == nil {
		return nil, fmt.Errorf("destination client is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Apply defaults for static fallback values
	postMigMode := cfg.PostMigrationMode
	if postMigMode == "" {
		postMigMode = PostMigrationProductionOnly
	}

	destRepoAction := cfg.DestRepoExistsAction
	if destRepoAction == "" {
		destRepoAction = DestinationRepoExistsFail
	}

	visibilityHandling := cfg.VisibilityHandling
	if visibilityHandling.PublicRepos == "" {
		visibilityHandling.PublicRepos = models.VisibilityPrivate
	}
	if visibilityHandling.InternalRepos == "" {
		visibilityHandling.InternalRepos = models.VisibilityPrivate
	}

	return &ExecutorFactory{
		storage:                    cfg.Storage,
		destClient:                 cfg.DestClient,
		logger:                     cfg.Logger,
		postMigrationMode:          postMigMode,
		configProvider:             cfg.ConfigProvider,
		staticDestRepoExistsAction: destRepoAction,
		staticVisibilityHandling:   visibilityHandling,
		executorCache:              make(map[int64]*Executor),
	}, nil
}

// getDestRepoExistsAction returns the current destination repo exists action,
// reading from the dynamic config provider if available.
func (f *ExecutorFactory) getDestRepoExistsAction() DestinationRepoExistsAction {
	if f.configProvider != nil {
		switch f.configProvider.GetDestRepoExistsAction() {
		case "fail":
			return DestinationRepoExistsFail
		case "skip":
			return DestinationRepoExistsSkip
		case "delete":
			return DestinationRepoExistsDelete
		}
	}
	return f.staticDestRepoExistsAction
}

// getVisibilityHandling returns the current visibility handling settings,
// reading from the dynamic config provider if available.
func (f *ExecutorFactory) getVisibilityHandling() VisibilityHandling {
	if f.configProvider != nil {
		return VisibilityHandling{
			PublicRepos:   f.configProvider.GetVisibilityPublic(),
			InternalRepos: f.configProvider.GetVisibilityInternal(),
		}
	}
	return f.staticVisibilityHandling
}

// GetExecutorForRepository returns an executor configured for the repository's source.
// Executors are created fresh for each migration to ensure they use current settings.
// Source client connections are still efficiently reused via the GitHub client pool.
func (f *ExecutorFactory) GetExecutorForRepository(ctx context.Context, repo *models.Repository) (*Executor, error) {
	// Check if repository has a source_id
	if repo.SourceID == nil {
		return nil, fmt.Errorf("repository %s has no source_id - cannot determine source credentials", repo.FullName)
	}

	sourceID := *repo.SourceID

	// Fetch source from database
	source, err := f.storage.GetSource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch source %d: %w", sourceID, err)
	}
	if source == nil {
		return nil, fmt.Errorf("source %d not found", sourceID)
	}

	// Check if source is active
	if !source.IsActive {
		return nil, fmt.Errorf("source %s (ID: %d) is not active", source.Name, sourceID)
	}

	// Create executor with current settings (read dynamically)
	// This ensures settings changes (like dest_repo_exists_action) take effect immediately
	executor, err := f.createExecutorForSource(source)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor for source %s: %w", source.Name, err)
	}

	f.logger.Debug("Created executor for source with current settings",
		"source_id", sourceID,
		"source_name", source.Name,
		"source_type", source.Type,
		"repo", repo.FullName,
		"dest_repo_exists_action", f.getDestRepoExistsAction())

	return executor, nil
}

// createExecutorForSource creates a new executor for the given source
func (f *ExecutorFactory) createExecutorForSource(source *models.Source) (*Executor, error) {
	// Read settings dynamically to pick up any changes
	cfg := ExecutorConfig{
		DestClient:           f.destClient,
		Storage:              f.storage,
		Logger:               f.logger,
		PostMigrationMode:    f.postMigrationMode,
		DestRepoExistsAction: f.getDestRepoExistsAction(),
		VisibilityHandling:   f.getVisibilityHandling(),
	}

	if source.IsGitHub() {
		// Create GitHub client for this source
		clientConfig := github.ClientConfig{
			BaseURL:     source.BaseURL,
			Token:       source.Token,
			Timeout:     120 * time.Second,
			RetryConfig: github.DefaultRetryConfig(),
			Logger:      f.logger,
		}

		// Add App credentials if configured WITH an installation ID
		// JWT-only mode (no installation ID) cannot access repo-level APIs needed for migration
		// In that case, fall back to the PAT token
		if source.HasAppAuth() && source.AppInstallationID != nil && *source.AppInstallationID > 0 {
			clientConfig.AppID = *source.AppID
			clientConfig.AppPrivateKey = *source.AppPrivateKey
			clientConfig.AppInstallationID = *source.AppInstallationID
			f.logger.Debug("Creating GitHub client with App installation auth",
				"source_id", source.ID,
				"app_id", *source.AppID,
				"installation_id", *source.AppInstallationID)
		} else if source.HasAppAuth() {
			// App is configured but no installation ID - use PAT token for migrations
			f.logger.Debug("App configured but no installation ID - using PAT token for migration",
				"source_id", source.ID,
				"app_id", *source.AppID)
		}

		client, err := github.NewClient(clientConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client: %w", err)
		}

		cfg.SourceClient = client
		cfg.SourceURL = source.BaseURL
		cfg.SourceToken = source.Token

	} else if source.IsAzureDevOps() {
		// ADO sources don't use a GitHub source client
		// They use the ADO PAT directly via GEI
		cfg.SourceClient = nil
		cfg.SourceURL = source.BaseURL
		cfg.SourceToken = source.Token
	} else {
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}

	return NewExecutor(cfg)
}

// InvalidateCache removes a cached executor for the given source ID.
// Call this when source credentials are updated.
func (f *ExecutorFactory) InvalidateCache(sourceID int64) {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	if _, exists := f.executorCache[sourceID]; exists {
		delete(f.executorCache, sourceID)
		f.logger.Info("Invalidated cached executor", "source_id", sourceID)
	}
}

// InvalidateAllCaches clears all cached executors.
// Call this when settings that affect all executors are changed.
func (f *ExecutorFactory) InvalidateAllCaches() {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	count := len(f.executorCache)
	f.executorCache = make(map[int64]*Executor)
	f.logger.Info("Invalidated all cached executors", "count", count)
}

// ExecuteWithStrategy executes a migration for the repository using the appropriate source.
// This is the main entry point for multi-source migrations.
func (f *ExecutorFactory) ExecuteWithStrategy(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	executor, err := f.GetExecutorForRepository(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to get executor: %w", err)
	}

	return executor.ExecuteWithStrategy(ctx, repo, batch, dryRun)
}

// ExecuteMigration implements the MigrationExecutor interface for compatibility with batch scheduler.
// It routes to ExecuteWithStrategy internally.
func (f *ExecutorFactory) ExecuteMigration(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	return f.ExecuteWithStrategy(ctx, repo, batch, dryRun)
}
