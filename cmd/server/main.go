package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/api"
	"github.com/kuhlman-labs/github-migrator/internal/batch"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/configsvc"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/logging"
	"github.com/kuhlman-labs/github-migrator/internal/migration"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/kuhlman-labs/github-migrator/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logger := logging.NewLogger(cfg.Logging)
	slog.SetDefault(logger)

	// Initialize database
	db, err := storage.NewDatabase(cfg.Database)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	// Run migrations
	if err := db.Migrate(); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize ConfigService for dynamic configuration
	cfgSvc, err := configsvc.New(db, cfg, logger)
	if err != nil {
		slog.Error("Failed to initialize config service", "error", err)
		os.Exit(1)
	}

	// Migrate existing .env config to database if settings are empty
	migrateEnvConfigToDatabase(db, cfg, logger)

	// Migrate legacy source config from .env to sources table
	migrateEnvSourceToDatabase(db, cfg, logger)

	// Reload ConfigService to pick up any migrated settings
	if err := cfgSvc.Reload(); err != nil {
		slog.Warn("Failed to reload config after migration", "error", err)
	}

	// Initialize GitHub dual clients (PAT + optional App auth)
	// Try legacy config first, then fall back to database settings
	// Note: sourceDualClient is only used for API handlers (discovery), not for migrations
	sourceDualClient := initializeSourceClient(cfg, logger)
	destDualClient := initializeDestClientWithFallback(cfg, cfgSvc, logger)

	// Create API server
	server := api.NewServer(cfg, db, logger, sourceDualClient, destDualClient)

	// Set ConfigService for dynamic settings management
	server.SetConfigService(cfgSvc)

	// Create cancellable context for all background workers (must be created before callback registration)
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	// Mutex to protect worker references and shutdown flag from concurrent access
	// (callback runs asynchronously, shutdown code runs in main goroutine)
	var workerMu sync.Mutex
	shuttingDown := false

	// Initialize migration executor and worker (destination client required)
	// Source clients are created dynamically per-source by ExecutorFactory
	migrationWorker := initializeMigrationWorker(workerCtx, cfg, cfgSvc, destDualClient, db, logger)

	// Initialize and start scheduler worker for scheduled batches
	// Uses ExecutorFactory for dynamic multi-source support
	schedulerWorker := initializeSchedulerWorker(cfg, cfgSvc, destDualClient, db, logger)

	// Register callback to start workers when destination is configured dynamically
	// This handles the case where destination is configured via UI after server start
	cfgSvc.OnReload(func() {
		workerMu.Lock()
		defer workerMu.Unlock()

		// Don't start workers if we're shutting down (context is cancelled)
		if shuttingDown {
			slog.Debug("Ignoring OnReload callback during shutdown")
			return
		}

		// Check if destination is now configured and workers haven't started yet
		if migrationWorker == nil && cfgSvc.IsDestinationConfigured() {
			slog.Info("Destination configured via settings, attempting to start migration worker...")
			// Create destination client from database settings
			destCfg := cfgSvc.GetDestinationConfig()
			newDestClient := initializeGitHubDualClient(
				destCfg.Token,
				destCfg.BaseURL,
				"github", // Always github for destination
				destCfg.AppID,
				destCfg.AppPrivateKey,
				destCfg.AppInstallationID,
				"destination",
				logger,
			)
			if newDestClient != nil {
				migrationWorker = initializeMigrationWorker(workerCtx, cfg, cfgSvc, newDestClient, db, logger)
				if migrationWorker != nil {
					slog.Info("Migration worker started after destination configuration")
				}
			}
		}
		if schedulerWorker == nil && cfgSvc.IsDestinationConfigured() {
			slog.Info("Destination configured via settings, attempting to start scheduler worker...")
			destCfg := cfgSvc.GetDestinationConfig()
			newDestClient := initializeGitHubDualClient(
				destCfg.Token,
				destCfg.BaseURL,
				"github",
				destCfg.AppID,
				destCfg.AppPrivateKey,
				destCfg.AppInstallationID,
				"destination",
				logger,
			)
			if newDestClient != nil {
				schedulerWorker = initializeSchedulerWorker(cfg, cfgSvc, newDestClient, db, logger)
				if schedulerWorker != nil {
					go schedulerWorker.Start(workerCtx)
					slog.Info("Scheduler worker started after destination configuration")
				}
			}
		}
	})

	// Initialize and start batch status updater
	statusUpdater := initializeBatchStatusUpdater(db, logger)

	if statusUpdater != nil {
		go statusUpdater.Start(workerCtx)
	}

	// Start scheduler worker with mutex protection (callback may modify schedulerWorker concurrently)
	workerMu.Lock()
	if schedulerWorker != nil {
		go schedulerWorker.Start(workerCtx)
	}
	workerMu.Unlock()

	// Start HTTP server
	// Timeouts increased for large responses (e.g., 4k+ mannequins)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      server.Router(),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("Starting server", "port", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal or shutdown request from setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("Received interrupt signal")
	case <-server.ShutdownChan():
		slog.Info("Received shutdown request from setup configuration")
	}

	slog.Info("Shutting down server...")

	// Set shutdown flag and cancel worker context atomically
	workerMu.Lock()
	shuttingDown = true
	cancelWorkers()

	// Stop migration worker (already holding mutex)
	if migrationWorker != nil {
		slog.Info("Stopping migration worker...")
		if err := migrationWorker.Stop(); err != nil {
			slog.Error("Failed to stop migration worker", "error", err)
		}
	}
	workerMu.Unlock()

	// Stop batch status updater explicitly (in addition to context cancellation)
	if statusUpdater != nil {
		slog.Info("Stopping batch status updater...")
		statusUpdater.Stop()
	}

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}

// initializeSourceClient initializes the GitHub source dual client if configured
func initializeSourceClient(cfg *config.Config, logger *slog.Logger) *github.DualClient {
	return initializeGitHubDualClient(
		cfg.Source.Token,
		cfg.Source.BaseURL,
		cfg.Source.Type,
		cfg.Source.AppID,
		cfg.Source.AppPrivateKey,
		cfg.Source.AppInstallationID,
		"source",
		logger,
	)
}

// initializeDestClient initializes the GitHub destination dual client if configured
func initializeDestClient(cfg *config.Config, logger *slog.Logger) *github.DualClient {
	return initializeGitHubDualClient(
		cfg.Destination.Token,
		cfg.Destination.BaseURL,
		cfg.Destination.Type,
		cfg.Destination.AppID,
		cfg.Destination.AppPrivateKey,
		cfg.Destination.AppInstallationID,
		"destination",
		logger,
	)
}

// initializeDestClientWithFallback initializes the destination client from legacy config,
// falling back to database settings if legacy config is not present.
func initializeDestClientWithFallback(cfg *config.Config, cfgSvc *configsvc.Service, logger *slog.Logger) *github.DualClient {
	// Try legacy config first
	if cfg.Destination.Token != "" && cfg.Destination.BaseURL != "" {
		return initializeDestClient(cfg, logger)
	}

	// Fall back to database settings
	destConfig := cfgSvc.GetDestinationConfig()
	if !destConfig.Configured {
		logger.Info("No destination configured in legacy config or database")
		return nil
	}

	logger.Info("Initializing destination client from database settings",
		"base_url", destConfig.BaseURL,
		"has_app_auth", destConfig.AppID > 0)

	return initializeGitHubDualClient(
		destConfig.Token,
		destConfig.BaseURL,
		"github", // Destination is always GitHub
		destConfig.AppID,
		destConfig.AppPrivateKey,
		destConfig.AppInstallationID,
		"destination",
		logger,
	)
}

// initializeGitHubDualClient initializes a GitHub dual client with PAT and optional App auth
func initializeGitHubDualClient(token, baseURL, clientType string, appID int64, appPrivateKey string, appInstallationID int64, name string, logger *slog.Logger) *github.DualClient {
	if token == "" || baseURL == "" || clientType != "github" {
		return nil
	}

	// Configure PAT client
	// Timeout increased to 120s for large org operations like mannequin listing
	patConfig := github.ClientConfig{
		BaseURL:     baseURL,
		Token:       token,
		Timeout:     120 * time.Second,
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      logger,
	}

	// Configure App client if credentials provided
	// Note: AppInstallationID is optional - if omitted, JWT-only client is created for enterprise discovery
	var appConfig *github.ClientConfig
	if appID > 0 && appPrivateKey != "" {
		appConfig = &github.ClientConfig{
			BaseURL:           baseURL,
			Timeout:           120 * time.Second,
			RetryConfig:       github.DefaultRetryConfig(),
			Logger:            logger,
			AppID:             appID,
			AppPrivateKey:     appPrivateKey,
			AppInstallationID: appInstallationID, // May be 0 for JWT-only auth
		}
	}

	dualClient, err := github.NewDualClient(github.DualClientConfig{
		PATConfig: patConfig,
		AppConfig: appConfig,
		Logger:    logger,
	})
	if err != nil {
		slog.Warn("Failed to initialize "+name+" GitHub dual client", "error", err)
		return nil
	}

	slog.Info(name+" GitHub dual client initialized",
		"base_url", baseURL,
		"type", clientType,
		"has_app_auth", dualClient.HasAppClient())
	return dualClient
}

// initializeMigrationWorker creates and starts the migration worker if configured.
// Uses ExecutorFactory for dynamic multi-source support.
// The provided context is used to control worker lifecycle for graceful shutdown.
func initializeMigrationWorker(ctx context.Context, cfg *config.Config, cfgSvc *configsvc.Service, destDualClient *github.DualClient, db *storage.Database, logger *slog.Logger) *worker.MigrationWorker {
	// Destination client is always required for migrations
	if destDualClient == nil {
		logger.Info("Migration worker not started - destination GitHub client not configured")
		return nil
	}

	// Create executor factory for dynamic multi-source support
	executorFactory, err := createExecutorFactory(cfg, cfgSvc, destDualClient, db, logger)
	if err != nil {
		slog.Error("Failed to create executor factory", "error", err)
		return nil
	}

	slog.Info("Migration executor factory created")

	// Create and start migration worker with factory
	pollInterval := time.Duration(cfg.Migration.PollIntervalSeconds) * time.Second
	migrationWorker, err := worker.NewMigrationWorker(worker.WorkerConfig{
		ExecutorFactory: executorFactory,
		Storage:         db,
		Logger:          logger,
		PollInterval:    pollInterval,
		Workers:         cfg.Migration.Workers,
	})
	if err != nil {
		slog.Error("Failed to create migration worker", "error", err)
		return nil
	}

	// Start worker in background with provided context for graceful shutdown
	if err := migrationWorker.Start(ctx); err != nil {
		slog.Error("Failed to start migration worker", "error", err)
		return nil
	}

	slog.Info("Migration worker started successfully",
		"workers", cfg.Migration.Workers,
		"poll_interval", pollInterval)

	return migrationWorker
}

// createExecutorFactory creates an executor factory with the shared configuration.
// Uses ConfigService for dynamic settings from database, falling back to static config.
func createExecutorFactory(cfg *config.Config, cfgSvc *configsvc.Service, destDualClient *github.DualClient, db *storage.Database, logger *slog.Logger) (*migration.ExecutorFactory, error) {
	// Get migration settings from database (via ConfigService) if available
	migCfg := cfgSvc.GetMigrationConfig()

	// Parse post-migration mode (still from static config as it's not in DB settings yet)
	var postMigMode migration.PostMigrationMode
	switch cfg.Migration.PostMigrationMode {
	case "never":
		postMigMode = migration.PostMigrationNever
	case "production_only":
		postMigMode = migration.PostMigrationProductionOnly
	case "dry_run_only":
		postMigMode = migration.PostMigrationDryRunOnly
	case "always":
		postMigMode = migration.PostMigrationAlways
	default:
		postMigMode = migration.PostMigrationProductionOnly
	}

	// Parse destination repo exists action from database settings
	var destRepoAction migration.DestinationRepoExistsAction
	switch migCfg.DestRepoExistsAction {
	case "fail":
		destRepoAction = migration.DestinationRepoExistsFail
	case "skip":
		destRepoAction = migration.DestinationRepoExistsSkip
	case "delete":
		destRepoAction = migration.DestinationRepoExistsDelete
	default:
		destRepoAction = migration.DestinationRepoExistsFail
	}

	// Parse visibility handling from database settings
	visibilityHandling := migration.VisibilityHandling{
		PublicRepos:   migCfg.VisibilityPublic,
		InternalRepos: migCfg.VisibilityInternal,
	}

	logger.Info("Creating migration executor factory",
		"visibility_public_to", visibilityHandling.PublicRepos,
		"visibility_internal_to", visibilityHandling.InternalRepos,
		"post_migration_mode", postMigMode,
		"dest_repo_exists_action", destRepoAction)

	return migration.NewExecutorFactory(migration.ExecutorFactoryConfig{
		Storage:              db,
		DestClient:           destDualClient.MigrationClient(),
		Logger:               logger,
		PostMigrationMode:    postMigMode,
		DestRepoExistsAction: destRepoAction,
		VisibilityHandling:   visibilityHandling,
		ConfigProvider:       cfgSvc, // Dynamic config provider for live setting updates
	})
}

// initializeBatchStatusUpdater creates the batch status updater service
func initializeBatchStatusUpdater(db *storage.Database, logger *slog.Logger) *batch.StatusUpdater {
	statusUpdater, err := batch.NewStatusUpdater(batch.StatusUpdaterConfig{
		Storage:  db,
		Logger:   logger,
		Interval: 30 * time.Second, // Update every 30 seconds
	})
	if err != nil {
		slog.Error("Failed to create batch status updater", "error", err)
		return nil
	}

	slog.Info("Batch status updater initialized", "interval", "30s")
	return statusUpdater
}

// initializeSchedulerWorker creates the scheduler worker for scheduled batches.
// Uses ExecutorFactory for dynamic multi-source support.
func initializeSchedulerWorker(cfg *config.Config, cfgSvc *configsvc.Service, destDualClient *github.DualClient, db *storage.Database, logger *slog.Logger) *worker.SchedulerWorker {
	// Destination client is always required
	if destDualClient == nil {
		logger.Info("Scheduler worker not started - destination GitHub client not configured")
		return nil
	}

	// Create executor factory for dynamic multi-source support
	executorFactory, err := createExecutorFactory(cfg, cfgSvc, destDualClient, db, logger)
	if err != nil {
		slog.Error("Failed to create executor factory for scheduler", "error", err)
		return nil
	}

	// Create orchestrator (which internally creates scheduler and organizer)
	// The factory implements MigrationExecutor interface
	orchestrator, err := batch.NewOrchestrator(batch.OrchestratorConfig{
		Storage:  db,
		Executor: executorFactory,
		Logger:   logger,
	})
	if err != nil {
		slog.Error("Failed to create batch orchestrator", "error", err)
		return nil
	}

	// Create scheduler worker
	schedulerWorker := worker.NewSchedulerWorker(orchestrator, logger)
	slog.Info("Scheduler worker initialized - will check for scheduled batches every minute")

	return schedulerWorker
}

// migrateEnvConfigToDatabase migrates existing .env configuration to the database settings table
// This is called on startup to handle upgrades from the old single-source configuration
func migrateEnvConfigToDatabase(db *storage.Database, cfg *config.Config, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get current settings from database
	settings, err := db.GetSettings(ctx)
	if err != nil {
		logger.Warn("Failed to get settings for migration check", "error", err)
		return
	}

	// Only migrate if destination is not yet configured in database
	if settings.HasDestination() {
		logger.Debug("Destination already configured in database, skipping .env migration")
		return
	}

	// Check if we have destination config in .env
	if cfg.Destination.Token == "" && cfg.Destination.AppID == 0 {
		logger.Debug("No destination configuration in .env to migrate")
		return
	}

	logger.Info("Migrating destination configuration from .env to database")

	// Build update request from .env config
	req := buildSettingsRequestFromEnv(cfg)

	// Apply the migration
	if _, err := db.UpdateSettings(ctx, req); err != nil {
		logger.Error("Failed to migrate .env config to database", "error", err)
		return
	}

	logger.Info("Successfully migrated .env configuration to database",
		"destination_base_url", cfg.Destination.BaseURL,
		"migration_workers", cfg.Migration.Workers,
		"auth_enabled", cfg.Auth.Enabled)
}

// migrateEnvSourceToDatabase migrates legacy source configuration from .env to the sources table.
// This enables users who configured the app via .env to see their source in the UI and avoid
// the "Add Migration Sources" step in the setup wizard.
func migrateEnvSourceToDatabase(db *storage.Database, cfg *config.Config, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if we already have sources configured in the database
	sources, err := db.ListSources(ctx)
	if err != nil {
		logger.Warn("Failed to list sources for migration check", "error", err)
		return
	}

	if len(sources) > 0 {
		logger.Debug("Sources already configured in database, skipping .env source migration")
		return
	}

	// Check if we have source config in .env
	// Token (PAT) is always required - GitHub Enterprise Importer API calls require a PAT
	// App auth (GitHub App) is optional and supplements but cannot replace the token
	if cfg.Source.Token == "" {
		logger.Debug("No source token in .env to migrate (app auth alone is insufficient)")
		return
	}

	// For Azure DevOps, organization is required
	if cfg.Source.Type == models.SourceConfigTypeAzureDevOps && cfg.Source.Organization == "" {
		logger.Debug("Azure DevOps source in .env is missing required organization, skipping migration")
		return
	}

	logger.Info("Migrating source configuration from .env to database")

	// Build source from .env config
	source := &models.Source{
		Name:     generateSourceName(cfg),
		Type:     cfg.Source.Type,
		BaseURL:  cfg.Source.BaseURL,
		Token:    cfg.Source.Token,
		IsActive: true,
	}

	// Set optional fields
	if cfg.Source.Organization != "" {
		source.Organization = &cfg.Source.Organization
	}
	if cfg.Source.AppID > 0 {
		source.AppID = &cfg.Source.AppID
	}
	if cfg.Source.AppPrivateKey != "" {
		source.AppPrivateKey = &cfg.Source.AppPrivateKey
	}
	if cfg.Source.AppInstallationID > 0 {
		source.AppInstallationID = &cfg.Source.AppInstallationID
	}

	// Create the source in the database
	if err := db.CreateSource(ctx, source); err != nil {
		logger.Error("Failed to migrate .env source to database", "error", err)
		return
	}

	logger.Info("Successfully migrated .env source configuration to database",
		"source_name", source.Name,
		"source_type", source.Type,
		"base_url", source.BaseURL,
		"has_app_auth", source.HasAppAuth())
}

// generateSourceName creates a user-friendly name for a migrated source based on its configuration
func generateSourceName(cfg *config.Config) string {
	// Use organization name if available
	if cfg.Source.Organization != "" {
		return cfg.Source.Organization
	}

	// Extract hostname from base URL for a meaningful name
	baseURL := cfg.Source.BaseURL
	if baseURL == "https://api.github.com" {
		return "GitHub.com"
	}
	if baseURL == "https://dev.azure.com" {
		return "Azure DevOps"
	}

	// For GitHub Enterprise Server or other instances, try to extract hostname
	// Remove protocol prefix
	hostname := baseURL
	if idx := len("https://"); len(hostname) > idx && hostname[:idx] == "https://" {
		hostname = hostname[idx:]
	} else if idx := len("http://"); len(hostname) > idx && hostname[:idx] == "http://" {
		hostname = hostname[idx:]
	}

	// Remove path suffix (e.g., /api/v3 for GHES)
	for i, c := range hostname {
		if c == '/' {
			hostname = hostname[:i]
			break
		}
	}

	if hostname != "" {
		return hostname
	}

	// Fallback
	return "Primary Source"
}

// buildSettingsRequestFromEnv builds an UpdateSettingsRequest from the config
func buildSettingsRequestFromEnv(cfg *config.Config) *models.UpdateSettingsRequest {
	req := &models.UpdateSettingsRequest{}
	applyDestinationConfig(req, cfg)
	applyMigrationConfig(req, cfg)
	applyAuthConfig(req, cfg)
	return req
}

// applyDestinationConfig applies destination settings from config
func applyDestinationConfig(req *models.UpdateSettingsRequest, cfg *config.Config) {
	if cfg.Destination.BaseURL != "" {
		req.DestinationBaseURL = &cfg.Destination.BaseURL
	}
	if cfg.Destination.Token != "" {
		req.DestinationToken = &cfg.Destination.Token
	}
	if cfg.Destination.AppID > 0 {
		req.DestinationAppID = &cfg.Destination.AppID
	}
	if cfg.Destination.AppPrivateKey != "" {
		req.DestinationAppPrivateKey = &cfg.Destination.AppPrivateKey
	}
	if cfg.Destination.AppInstallationID > 0 {
		req.DestinationAppInstallationID = &cfg.Destination.AppInstallationID
	}
}

// applyMigrationConfig applies migration settings from config
func applyMigrationConfig(req *models.UpdateSettingsRequest, cfg *config.Config) {
	if cfg.Migration.Workers > 0 {
		req.MigrationWorkers = &cfg.Migration.Workers
	}
	if cfg.Migration.PollIntervalSeconds > 0 {
		req.MigrationPollIntervalSeconds = &cfg.Migration.PollIntervalSeconds
	}
	if cfg.Migration.DestRepoExistsAction != "" {
		req.MigrationDestRepoExistsAction = &cfg.Migration.DestRepoExistsAction
	}
	if cfg.Migration.VisibilityHandling.PublicRepos != "" {
		req.MigrationVisibilityPublic = &cfg.Migration.VisibilityHandling.PublicRepos
	}
	if cfg.Migration.VisibilityHandling.InternalRepos != "" {
		req.MigrationVisibilityInternal = &cfg.Migration.VisibilityHandling.InternalRepos
	}
}

// applyAuthConfig applies auth settings from config
func applyAuthConfig(req *models.UpdateSettingsRequest, cfg *config.Config) {
	req.AuthEnabled = &cfg.Auth.Enabled
	if cfg.Auth.SessionSecret != "" {
		req.AuthSessionSecret = &cfg.Auth.SessionSecret
	}
	if cfg.Auth.SessionDurationHours > 0 {
		req.AuthSessionDurationHours = &cfg.Auth.SessionDurationHours
	}
	if cfg.Auth.CallbackURL != "" {
		req.AuthCallbackURL = &cfg.Auth.CallbackURL
	}
	if cfg.Auth.FrontendURL != "" {
		req.AuthFrontendURL = &cfg.Auth.FrontendURL
	}
}
