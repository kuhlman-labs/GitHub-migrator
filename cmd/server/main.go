package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/api"
	"github.com/kuhlman-labs/github-migrator/internal/batch"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/logging"
	"github.com/kuhlman-labs/github-migrator/internal/migration"
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
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize GitHub dual clients (PAT + optional App auth)
	sourceDualClient := initializeSourceClient(cfg, logger)
	destDualClient := initializeDestClient(cfg, logger)

	// Create API server
	server := api.NewServer(cfg, db, logger, sourceDualClient, destDualClient)

	// Initialize migration executor and worker (if both clients are available)
	migrationWorker := initializeMigrationWorker(cfg, sourceDualClient, destDualClient, db, logger)

	// Initialize and start batch status updater
	statusUpdater := initializeBatchStatusUpdater(db, logger)
	ctx, cancelStatusUpdater := context.WithCancel(context.Background())
	defer cancelStatusUpdater()

	if statusUpdater != nil {
		go statusUpdater.Start(ctx)
	}

	// Initialize and start scheduler worker for scheduled batches
	schedulerWorker := initializeSchedulerWorker(cfg, sourceDualClient, destDualClient, db, logger)
	if schedulerWorker != nil {
		go schedulerWorker.Start(ctx)
	}

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

	// Stop migration worker first
	if migrationWorker != nil {
		slog.Info("Stopping migration worker...")
		if err := migrationWorker.Stop(); err != nil {
			slog.Error("Failed to stop migration worker", "error", err)
		}
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

// initializeMigrationWorker creates and starts the migration worker if configured
//
//nolint:gocyclo // Complexity is inherent to worker initialization logic
func initializeMigrationWorker(cfg *config.Config, sourceDualClient, destDualClient *github.DualClient, db *storage.Database, logger *slog.Logger) *worker.MigrationWorker {
	// Destination client is always required for migrations
	// Source client is only required for GitHub-to-GitHub migrations (ADO migrations don't need it)
	if destDualClient == nil {
		logger.Info("Migration worker not started - destination GitHub client not configured")
		return nil
	}

	// For ADO sources, sourceDualClient will be nil - this is expected and supported
	if sourceDualClient == nil && cfg.Source.Type == "azuredevops" {
		logger.Info("Initializing migration worker for Azure DevOps source",
			"source_type", cfg.Source.Type)
	} else if sourceDualClient == nil {
		logger.Warn("Migration worker not started - source client not configured and source is not Azure DevOps")
		return nil
	}

	// Parse post-migration mode
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

	// Parse destination repo exists action
	var destRepoAction migration.DestinationRepoExistsAction
	switch cfg.Migration.DestRepoExistsAction {
	case "fail":
		destRepoAction = migration.DestinationRepoExistsFail
	case "skip":
		destRepoAction = migration.DestinationRepoExistsSkip
	case "delete":
		destRepoAction = migration.DestinationRepoExistsDelete
	default:
		destRepoAction = migration.DestinationRepoExistsFail
	}

	// Parse visibility handling configuration
	visibilityHandling := migration.VisibilityHandling{
		PublicRepos:   cfg.Migration.VisibilityHandling.PublicRepos,
		InternalRepos: cfg.Migration.VisibilityHandling.InternalRepos,
	}

	// Create migration executor with PAT clients (required for migrations per GitHub API)
	// For ADO sources, sourceDualClient will be nil - pass nil SourceClient to executor
	var sourceClient *github.Client
	if sourceDualClient != nil {
		sourceClient = sourceDualClient.MigrationClient()
	}

	logger.Info("Creating migration executor",
		"source_type", cfg.Source.Type,
		"has_source_client", sourceClient != nil,
		"has_source_token", cfg.Source.Token != "",
		"source_url", cfg.Source.BaseURL,
		"visibility_public_to", visibilityHandling.PublicRepos,
		"visibility_internal_to", visibilityHandling.InternalRepos)
	executor, err := migration.NewExecutor(migration.ExecutorConfig{
		SourceClient:         sourceClient,
		SourceToken:          cfg.Source.Token,   // ADO PAT for ADO sources, GitHub PAT for GitHub sources
		SourceURL:            cfg.Source.BaseURL, // GitHub base URL or ADO org URL (e.g., https://dev.azure.com/org)
		DestClient:           destDualClient.MigrationClient(),
		Storage:              db,
		Logger:               logger,
		PostMigrationMode:    postMigMode,
		DestRepoExistsAction: destRepoAction,
		VisibilityHandling:   visibilityHandling,
	})
	if err != nil {
		slog.Error("Failed to create migration executor", "error", err)
		return nil
	}

	slog.Info("Migration executor created")

	// Create and start migration worker
	pollInterval := time.Duration(cfg.Migration.PollIntervalSeconds) * time.Second
	migrationWorker, err := worker.NewMigrationWorker(worker.WorkerConfig{
		Executor:     executor,
		Storage:      db,
		Logger:       logger,
		PollInterval: pollInterval,
		Workers:      cfg.Migration.Workers,
	})
	if err != nil {
		slog.Error("Failed to create migration worker", "error", err)
		return nil
	}

	// Start worker in background
	ctx := context.Background()
	if err := migrationWorker.Start(ctx); err != nil {
		slog.Error("Failed to start migration worker", "error", err)
		return nil
	}

	slog.Info("Migration worker started successfully",
		"workers", cfg.Migration.Workers,
		"poll_interval", pollInterval)

	return migrationWorker
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

// initializeSchedulerWorker creates the scheduler worker for scheduled batches
func initializeSchedulerWorker(cfg *config.Config, sourceDualClient, destDualClient *github.DualClient, db *storage.Database, logger *slog.Logger) *worker.SchedulerWorker {
	// Destination client is always required
	// Source client is only required for GitHub-to-GitHub migrations (ADO migrations don't need it)
	if destDualClient == nil {
		logger.Info("Scheduler worker not started - destination GitHub client not configured")
		return nil
	}

	// For ADO sources, sourceDualClient will be nil - this is expected and supported
	if sourceDualClient == nil && cfg.Source.Type == "azuredevops" {
		logger.Info("Initializing scheduler worker for Azure DevOps source",
			"source_type", cfg.Source.Type)
	} else if sourceDualClient == nil {
		logger.Warn("Scheduler worker not started - source client not configured and source is not Azure DevOps")
		return nil
	}

	// Create migration executor (required for batch orchestrator)
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

	var destRepoAction migration.DestinationRepoExistsAction
	switch cfg.Migration.DestRepoExistsAction {
	case "fail":
		destRepoAction = migration.DestinationRepoExistsFail
	case "skip":
		destRepoAction = migration.DestinationRepoExistsSkip
	case "delete":
		destRepoAction = migration.DestinationRepoExistsDelete
	default:
		destRepoAction = migration.DestinationRepoExistsFail
	}

	visibilityHandlingScheduler := migration.VisibilityHandling{
		PublicRepos:   cfg.Migration.VisibilityHandling.PublicRepos,
		InternalRepos: cfg.Migration.VisibilityHandling.InternalRepos,
	}

	// Handle nil source client for ADO sources
	var sourceClientForScheduler *github.Client
	if sourceDualClient != nil {
		sourceClientForScheduler = sourceDualClient.MigrationClient()
	}

	executor, err := migration.NewExecutor(migration.ExecutorConfig{
		SourceClient:         sourceClientForScheduler,
		SourceToken:          cfg.Source.Token,   // ADO PAT for ADO sources
		SourceURL:            cfg.Source.BaseURL, // GitHub base URL or ADO org URL
		DestClient:           destDualClient.MigrationClient(),
		Storage:              db,
		Logger:               logger,
		PostMigrationMode:    postMigMode,
		DestRepoExistsAction: destRepoAction,
		VisibilityHandling:   visibilityHandlingScheduler,
	})
	if err != nil {
		slog.Error("Failed to create executor for scheduler", "error", err)
		return nil
	}

	// Create orchestrator (which internally creates scheduler and organizer)
	orchestrator, err := batch.NewOrchestrator(batch.OrchestratorConfig{
		Storage:  db,
		Executor: executor,
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
