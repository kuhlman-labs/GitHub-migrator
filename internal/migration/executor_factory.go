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

// ExecutorFactory creates and caches source-specific migration executors.
// It enables multi-source migrations by dynamically creating executors
// based on each repository's source_id.
type ExecutorFactory struct {
	storage              *storage.Database
	destClient           *github.Client
	logger               *slog.Logger
	postMigrationMode    PostMigrationMode
	destRepoExistsAction DestinationRepoExistsAction
	visibilityHandling   VisibilityHandling

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

	// Apply defaults
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
		storage:              cfg.Storage,
		destClient:           cfg.DestClient,
		logger:               cfg.Logger,
		postMigrationMode:    postMigMode,
		destRepoExistsAction: destRepoAction,
		visibilityHandling:   visibilityHandling,
		executorCache:        make(map[int64]*Executor),
	}, nil
}

// GetExecutorForRepository returns an executor configured for the repository's source.
// It caches executors per source to avoid recreating clients for each migration.
func (f *ExecutorFactory) GetExecutorForRepository(ctx context.Context, repo *models.Repository) (*Executor, error) {
	// Check if repository has a source_id
	if repo.SourceID == nil {
		return nil, fmt.Errorf("repository %s has no source_id - cannot determine source credentials", repo.FullName)
	}

	sourceID := *repo.SourceID

	// Check cache first
	f.cacheMu.RLock()
	if executor, exists := f.executorCache[sourceID]; exists {
		f.cacheMu.RUnlock()
		f.logger.Debug("Using cached executor for source",
			"source_id", sourceID,
			"repo", repo.FullName)
		return executor, nil
	}
	f.cacheMu.RUnlock()

	// Need to create a new executor - acquire write lock
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if executor, exists := f.executorCache[sourceID]; exists {
		return executor, nil
	}

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

	// Create executor based on source type
	executor, err := f.createExecutorForSource(source)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor for source %s: %w", source.Name, err)
	}

	// Cache the executor
	f.executorCache[sourceID] = executor

	f.logger.Info("Created new executor for source",
		"source_id", sourceID,
		"source_name", source.Name,
		"source_type", source.Type,
		"repo", repo.FullName)

	return executor, nil
}

// createExecutorForSource creates a new executor for the given source
func (f *ExecutorFactory) createExecutorForSource(source *models.Source) (*Executor, error) {
	cfg := ExecutorConfig{
		DestClient:           f.destClient,
		Storage:              f.storage,
		Logger:               f.logger,
		PostMigrationMode:    f.postMigrationMode,
		DestRepoExistsAction: f.destRepoExistsAction,
		VisibilityHandling:   f.visibilityHandling,
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
