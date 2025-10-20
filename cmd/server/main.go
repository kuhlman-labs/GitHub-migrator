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

	"github.com/brettkuhlman/github-migrator/internal/api"
	"github.com/brettkuhlman/github-migrator/internal/batch"
	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/logging"
	"github.com/brettkuhlman/github-migrator/internal/migration"
	"github.com/brettkuhlman/github-migrator/internal/storage"
	"github.com/brettkuhlman/github-migrator/internal/worker"
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

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("Starting server", "port", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

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
	patConfig := github.ClientConfig{
		BaseURL:     baseURL,
		Token:       token,
		Timeout:     30 * time.Second,
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      logger,
	}

	// Configure App client if credentials provided
	var appConfig *github.ClientConfig
	if appID > 0 && appPrivateKey != "" && appInstallationID > 0 {
		appConfig = &github.ClientConfig{
			BaseURL:           baseURL,
			Timeout:           30 * time.Second,
			RetryConfig:       github.DefaultRetryConfig(),
			Logger:            logger,
			AppID:             appID,
			AppPrivateKey:     appPrivateKey,
			AppInstallationID: appInstallationID,
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
func initializeMigrationWorker(cfg *config.Config, sourceDualClient, destDualClient *github.DualClient, db *storage.Database, logger *slog.Logger) *worker.MigrationWorker {
	if sourceDualClient == nil || destDualClient == nil {
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

	// Create migration executor with PAT clients (required for migrations per GitHub API)
	logger.Info("Creating migration executor with PAT clients (per GitHub migration API requirements)")
	executor, err := migration.NewExecutor(migration.ExecutorConfig{
		SourceClient:         sourceDualClient.MigrationClient(),
		DestClient:           destDualClient.MigrationClient(),
		Storage:              db,
		Logger:               logger,
		PostMigrationMode:    postMigMode,
		DestRepoExistsAction: destRepoAction,
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
