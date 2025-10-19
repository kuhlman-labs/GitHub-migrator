package api

import (
	"log/slog"
	"net/http"

	"github.com/brettkuhlman/github-migrator/internal/api/handlers"
	"github.com/brettkuhlman/github-migrator/internal/api/middleware"
	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/source"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

type Server struct {
	config  *config.Config
	db      *storage.Database
	logger  *slog.Logger
	handler *handlers.Handler
}

func NewServer(cfg *config.Config, db *storage.Database, logger *slog.Logger, sourceClient *github.Client, destClient *github.Client) *Server {
	// Create source provider from config
	var sourceProvider source.Provider
	if cfg.Source.Token != "" {
		var err error
		sourceProvider, err = source.NewProviderFromConfig(cfg.Source)
		if err != nil {
			logger.Warn("Failed to create source provider", "error", err)
		}
	}

	return &Server{
		config:  cfg,
		db:      db,
		logger:  logger,
		handler: handlers.NewHandler(db, logger, sourceClient, destClient, sourceProvider),
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Apply middleware
	handler := middleware.CORS(
		middleware.Logging(s.logger)(
			middleware.Recovery(s.logger)(mux),
		),
	)

	// Health check
	mux.HandleFunc("/health", s.handler.Health)

	// Discovery endpoints
	mux.HandleFunc("POST /api/v1/discovery/start", s.handler.StartDiscovery)
	mux.HandleFunc("GET /api/v1/discovery/status", s.handler.DiscoveryStatus)

	// Repository endpoints
	mux.HandleFunc("GET /api/v1/repositories", s.handler.ListRepositories)
	mux.HandleFunc("GET /api/v1/repositories/{fullName}", s.handler.GetRepository)
	mux.HandleFunc("PATCH /api/v1/repositories/{fullName}", s.handler.UpdateRepository)
	mux.HandleFunc("POST /api/v1/repositories/{fullName}/rediscover", s.handler.RediscoverRepository)
	mux.HandleFunc("POST /api/v1/repositories/{fullName}/unlock", s.handler.UnlockRepository)
	mux.HandleFunc("POST /api/v1/repositories/{fullName}/rollback", s.handler.RollbackRepository)

	// Organization endpoints
	mux.HandleFunc("GET /api/v1/organizations", s.handler.ListOrganizations)
	mux.HandleFunc("GET /api/v1/organizations/list", s.handler.GetOrganizationList)

	// Batch endpoints
	mux.HandleFunc("GET /api/v1/batches", s.handler.ListBatches)
	mux.HandleFunc("POST /api/v1/batches", s.handler.CreateBatch)
	mux.HandleFunc("GET /api/v1/batches/{id}", s.handler.GetBatch)
	mux.HandleFunc("PATCH /api/v1/batches/{id}", s.handler.UpdateBatch)
	mux.HandleFunc("POST /api/v1/batches/{id}/dry-run", s.handler.DryRunBatch)
	mux.HandleFunc("POST /api/v1/batches/{id}/start", s.handler.StartBatch)
	mux.HandleFunc("POST /api/v1/batches/{id}/repositories", s.handler.AddRepositoriesToBatch)
	mux.HandleFunc("DELETE /api/v1/batches/{id}/repositories", s.handler.RemoveRepositoriesFromBatch)
	mux.HandleFunc("POST /api/v1/batches/{id}/retry", s.handler.RetryBatchFailures)

	// Migration endpoints
	mux.HandleFunc("POST /api/v1/migrations/start", s.handler.StartMigration)
	mux.HandleFunc("GET /api/v1/migrations/{id}", s.handler.GetMigrationStatus)
	mux.HandleFunc("GET /api/v1/migrations/{id}/history", s.handler.GetMigrationHistory)
	mux.HandleFunc("GET /api/v1/migrations/{id}/logs", s.handler.GetMigrationLogs)
	mux.HandleFunc("GET /api/v1/migrations/history", s.handler.GetMigrationHistoryList)
	mux.HandleFunc("GET /api/v1/migrations/history/export", s.handler.ExportMigrationHistory)

	// Analytics endpoints
	mux.HandleFunc("GET /api/v1/analytics/summary", s.handler.GetAnalyticsSummary)
	mux.HandleFunc("GET /api/v1/analytics/progress", s.handler.GetMigrationProgress)

	return handler
}
