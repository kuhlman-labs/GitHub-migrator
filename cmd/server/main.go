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

	// Initialize GitHub clients
	sourceClient := initializeSourceClient(cfg, logger)
	destClient := initializeDestClient(cfg, logger)

	// Create API server
	server := api.NewServer(cfg, db, logger, sourceClient, destClient)

	// Initialize migration executor and worker (if both clients are available)
	migrationWorker := initializeMigrationWorker(cfg, sourceClient, destClient, db, logger)

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

// initializeSourceClient initializes the GitHub source client if configured
func initializeSourceClient(cfg *config.Config, logger *slog.Logger) *github.Client {
	return initializeGitHubClient(
		cfg.Source.Token,
		cfg.Source.BaseURL,
		cfg.Source.Type,
		"source",
		logger,
	)
}

// initializeDestClient initializes the GitHub destination client if configured
func initializeDestClient(cfg *config.Config, logger *slog.Logger) *github.Client {
	return initializeGitHubClient(
		cfg.Destination.Token,
		cfg.Destination.BaseURL,
		cfg.Destination.Type,
		"destination",
		logger,
	)
}

// initializeGitHubClient initializes a GitHub client with the given configuration
func initializeGitHubClient(token, baseURL, clientType, name string, logger *slog.Logger) *github.Client {
	if token == "" || baseURL == "" || clientType != "github" {
		return nil
	}

	client, err := github.NewClient(github.ClientConfig{
		BaseURL:     baseURL,
		Token:       token,
		Timeout:     30 * time.Second,
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      logger,
	})
	if err != nil {
		slog.Warn("Failed to initialize "+name+" GitHub client", "error", err)
		return nil
	}

	slog.Info(name+" GitHub client initialized",
		"base_url", baseURL,
		"type", clientType)
	return client
}

// initializeMigrationWorker creates and starts the migration worker if configured
func initializeMigrationWorker(cfg *config.Config, sourceClient, destClient *github.Client, db *storage.Database, logger *slog.Logger) *worker.MigrationWorker {
	if sourceClient == nil || destClient == nil {
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

	// Create migration executor
	executor, err := migration.NewExecutor(migration.ExecutorConfig{
		SourceClient:         sourceClient,
		DestClient:           destClient,
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
