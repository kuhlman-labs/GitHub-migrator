package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/api/handlers"
	"github.com/brettkuhlman/github-migrator/internal/api/middleware"
	"github.com/brettkuhlman/github-migrator/internal/auth"
	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/source"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

type Server struct {
	config      *config.Config
	db          *storage.Database
	logger      *slog.Logger
	handler     *handlers.Handler
	authHandler *handlers.AuthHandler
}

func NewServer(cfg *config.Config, db *storage.Database, logger *slog.Logger, sourceDualClient *github.DualClient, destDualClient *github.DualClient) *Server {
	// Create source provider from config
	var sourceProvider source.Provider
	if cfg.Source.Token != "" {
		var err error
		sourceProvider, err = source.NewProviderFromConfig(cfg.Source)
		if err != nil {
			logger.Warn("Failed to create source provider", "error", err)
		}
	}

	// Create base config for per-org client creation (if GitHub App auth is configured)
	var sourceBaseConfig *github.ClientConfig
	if cfg.Source.AppID > 0 && cfg.Source.AppPrivateKey != "" {
		sourceBaseConfig = &github.ClientConfig{
			BaseURL:           cfg.Source.BaseURL,
			Timeout:           30 * time.Second,
			RetryConfig:       github.DefaultRetryConfig(),
			Logger:            logger,
			AppID:             cfg.Source.AppID,
			AppPrivateKey:     cfg.Source.AppPrivateKey,
			AppInstallationID: cfg.Source.AppInstallationID, // May be 0 for JWT-only auth
		}
	}

	// Create auth handler if enabled
	var authHandler *handlers.AuthHandler
	if cfg.Auth.Enabled {
		var err error
		authHandler, err = handlers.NewAuthHandler(&cfg.Auth, logger, cfg.Source.BaseURL)
		if err != nil {
			logger.Error("Failed to create auth handler", "error", err)
		} else {
			logger.Info("Authentication enabled")
		}
	}

	return &Server{
		config:      cfg,
		db:          db,
		logger:      logger,
		handler:     handlers.NewHandler(db, logger, sourceDualClient, destDualClient, sourceProvider, sourceBaseConfig),
		authHandler: authHandler,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Create auth middleware if enabled
	var authMiddleware *auth.Middleware
	if s.config.Auth.Enabled && s.authHandler != nil {
		jwtManager, _ := auth.NewJWTManager(s.config.Auth.SessionSecret, s.config.Auth.SessionDurationHours)
		authorizer := auth.NewAuthorizer(&s.config.Auth, s.logger, s.config.Source.BaseURL)
		authMiddleware = auth.NewMiddleware(jwtManager, authorizer, s.logger, true)
	}

	// Public auth endpoints (no authentication required)
	if s.config.Auth.Enabled && s.authHandler != nil {
		mux.HandleFunc("GET /api/v1/auth/login", s.authHandler.HandleLogin)
		mux.HandleFunc("GET /api/v1/auth/callback", s.authHandler.HandleCallback)
		mux.HandleFunc("GET /api/v1/auth/config", s.authHandler.HandleAuthConfig)

		// Protected auth endpoints (require authentication)
		if authMiddleware != nil {
			mux.Handle("POST /api/v1/auth/logout", authMiddleware.RequireAuth(http.HandlerFunc(s.authHandler.HandleLogout)))
			mux.Handle("GET /api/v1/auth/user", authMiddleware.RequireAuth(http.HandlerFunc(s.authHandler.HandleCurrentUser)))
			mux.Handle("POST /api/v1/auth/refresh", authMiddleware.RequireAuth(http.HandlerFunc(s.authHandler.HandleRefreshToken)))
		}
	}

	// Health check (always public)
	mux.HandleFunc("/health", s.handler.Health)

	// Helper to conditionally wrap with auth
	protect := func(pattern string, handler http.HandlerFunc) {
		if authMiddleware != nil {
			mux.Handle(pattern, authMiddleware.RequireAuth(handler))
		} else {
			mux.HandleFunc(pattern, handler)
		}
	}

	// Discovery endpoints
	protect("POST /api/v1/discovery/start", s.handler.StartDiscovery)
	protect("GET /api/v1/discovery/status", s.handler.DiscoveryStatus)

	// Repository endpoints
	protect("GET /api/v1/repositories", s.handler.ListRepositories)
	protect("GET /api/v1/repositories/{fullName}", s.handler.GetRepository)
	protect("PATCH /api/v1/repositories/{fullName}", s.handler.UpdateRepository)
	protect("POST /api/v1/repositories/{fullName}/rediscover", s.handler.RediscoverRepository)
	protect("POST /api/v1/repositories/{fullName}/mark-remediated", s.handler.MarkRepositoryRemediated)
	protect("POST /api/v1/repositories/{fullName}/unlock", s.handler.UnlockRepository)
	protect("POST /api/v1/repositories/{fullName}/rollback", s.handler.RollbackRepository)
	protect("POST /api/v1/repositories/{fullName}/mark-wont-migrate", s.handler.MarkRepositoryWontMigrate)

	// Organization endpoints
	protect("GET /api/v1/organizations", s.handler.ListOrganizations)
	protect("GET /api/v1/organizations/list", s.handler.GetOrganizationList)

	// Batch endpoints
	protect("GET /api/v1/batches", s.handler.ListBatches)
	protect("POST /api/v1/batches", s.handler.CreateBatch)
	protect("GET /api/v1/batches/{id}", s.handler.GetBatch)
	protect("PATCH /api/v1/batches/{id}", s.handler.UpdateBatch)
	protect("DELETE /api/v1/batches/{id}", s.handler.DeleteBatch)
	protect("POST /api/v1/batches/{id}/dry-run", s.handler.DryRunBatch)
	protect("POST /api/v1/batches/{id}/start", s.handler.StartBatch)
	protect("POST /api/v1/batches/{id}/repositories", s.handler.AddRepositoriesToBatch)
	protect("DELETE /api/v1/batches/{id}/repositories", s.handler.RemoveRepositoriesFromBatch)
	protect("POST /api/v1/batches/{id}/retry", s.handler.RetryBatchFailures)

	// Migration endpoints
	protect("POST /api/v1/migrations/start", s.handler.StartMigration)
	protect("GET /api/v1/migrations/{id}", s.handler.GetMigrationStatus)
	protect("GET /api/v1/migrations/{id}/history", s.handler.GetMigrationHistory)
	protect("GET /api/v1/migrations/{id}/logs", s.handler.GetMigrationLogs)
	protect("GET /api/v1/migrations/history", s.handler.GetMigrationHistoryList)
	protect("GET /api/v1/migrations/history/export", s.handler.ExportMigrationHistory)

	// Analytics endpoints
	protect("GET /api/v1/analytics/summary", s.handler.GetAnalyticsSummary)
	protect("GET /api/v1/analytics/progress", s.handler.GetMigrationProgress)
	protect("GET /api/v1/analytics/executive-report", s.handler.GetExecutiveReport)
	protect("GET /api/v1/analytics/executive-report/export", s.handler.ExportExecutiveReport)

	// Self-service endpoints
	protect("POST /api/v1/self-service/migrate", s.handler.HandleSelfServiceMigration)

	// Apply middleware
	handler := middleware.CORS(
		middleware.Logging(s.logger)(
			middleware.Recovery(s.logger)(mux),
		),
	)

	return handler
}
