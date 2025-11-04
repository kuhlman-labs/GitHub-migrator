package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	// Note: Using {fullName...} trailing wildcard to capture full repo name including slashes (e.g., "org/repo")
	protect("GET /api/v1/repositories", s.handler.ListRepositories)
	// Repository GET route handles both repo details and dependencies via suffix detection
	protect("GET /api/v1/repositories/{fullName...}", s.handler.GetRepositoryOrDependencies)
	protect("PATCH /api/v1/repositories/{fullName...}", s.handler.UpdateRepository)
	// For action routes, we need to parse the action from the fullName in the handler
	protect("POST /api/v1/repositories/{fullName...}", s.handler.HandleRepositoryAction)

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

	// Serve static frontend files for SPA
	mux.HandleFunc("/", s.serveFrontend)

	// Apply middleware
	handler := middleware.CORS(
		middleware.Logging(s.logger)(
			middleware.Recovery(s.logger)(mux),
		),
	)

	return handler
}

// serveFrontend serves the React frontend static files and handles SPA routing
func (s *Server) serveFrontend(w http.ResponseWriter, r *http.Request) {
	frontendDir := "./web/dist"

	// Validate and resolve the requested path
	absDir, absPath, ok := s.validateFrontendPath(r.URL.Path, frontendDir)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Try to serve the requested file if it exists
	if s.tryServeFile(w, r, absPath) {
		return
	}

	// Fall back to index.html for SPA routing
	s.serveSPAFallback(w, r, filepath.Clean(r.URL.Path), absDir)
}

// validateFrontendPath validates and resolves a path within the frontend directory
func (s *Server) validateFrontendPath(requestPath, frontendDir string) (absDir, absPath string, ok bool) {
	// Clean the path to prevent traversal attacks
	cleanPath := filepath.Clean(requestPath)
	fullPath := filepath.Join(frontendDir, cleanPath)

	// Get absolute paths for security validation
	absDir, err := filepath.Abs(frontendDir)
	if err != nil {
		s.logger.Error("Failed to get absolute path for frontend directory", "error", err)
		return "", "", false
	}

	absPath, err = filepath.Abs(fullPath)
	if err != nil {
		s.logger.Error("Failed to get absolute path for requested file", "error", err)
		return "", "", false
	}

	// Validate that the requested path is within the frontend directory
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) && absPath != absDir {
		s.logger.Warn("Path traversal attempt detected", "requested_path", requestPath, "resolved_path", absPath)
		return "", "", false
	}

	return absDir, absPath, true
}

// tryServeFile attempts to serve a file if it exists and is not a directory
func (s *Server) tryServeFile(w http.ResponseWriter, r *http.Request, absPath string) bool {
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return false
	}

	// Set appropriate content type
	s.setContentType(w, absPath)
	http.ServeFile(w, r, absPath)
	return true
}

// setContentType sets the HTTP content type header based on file extension
func (s *Server) setContentType(w http.ResponseWriter, filePath string) {
	ext := filepath.Ext(filePath)
	contentTypes := map[string]string{
		".js":   "application/javascript",
		".css":  "text/css",
		".json": "application/json",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
	}

	if contentType, ok := contentTypes[ext]; ok {
		w.Header().Set("Content-Type", contentType)
	}
}

// serveSPAFallback serves index.html for SPA routing
func (s *Server) serveSPAFallback(w http.ResponseWriter, r *http.Request, path, absDir string) {
	// Only serve SPA fallback for non-API routes
	if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/health") {
		http.NotFound(w, r)
		return
	}

	indexPath := filepath.Join(absDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}

	http.NotFound(w, r)
}
