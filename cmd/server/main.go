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
	"github.com/brettkuhlman/github-migrator/internal/storage"
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

	// Initialize GitHub client (optional)
	var ghClient *github.Client
	if cfg.GitHub.Source.Token != "" && cfg.GitHub.Source.BaseURL != "" {
		ghClient, err = github.NewClient(github.ClientConfig{
			BaseURL:     cfg.GitHub.Source.BaseURL,
			Token:       cfg.GitHub.Source.Token,
			Timeout:     30 * time.Second,
			RetryConfig: github.DefaultRetryConfig(),
			Logger:      logger,
		})
		if err != nil {
			slog.Warn("Failed to initialize GitHub client", "error", err)
		} else {
			slog.Info("GitHub client initialized", "base_url", cfg.GitHub.Source.BaseURL)
		}
	}

	// Create API server
	server := api.NewServer(cfg, db, logger, ghClient)

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}
