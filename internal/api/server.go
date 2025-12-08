package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/api/handlers"
	"github.com/kuhlman-labs/github-migrator/internal/api/middleware"
	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

type Server struct {
	config              *config.Config
	db                  *storage.Database
	logger              *slog.Logger
	handler             *handlers.Handler
	authHandler         *handlers.AuthHandler
	adoHandler          *handlers.ADOHandler
	entraIDOAuthHandler *auth.EntraIDOAuthHandler
	shutdownChan        chan struct{}
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

	// Determine source base URL for permission checks
	sourceBaseURL := cfg.Auth.GetOAuthBaseURL(cfg)

	// Create main handler
	mainHandler := handlers.NewHandler(db, logger, sourceDualClient, destDualClient, sourceProvider, sourceBaseConfig, &cfg.Auth, sourceBaseURL, cfg.Source.Type)

	// Create ADO handler if source is Azure DevOps
	var adoHandler *handlers.ADOHandler
	var entraIDOAuthHandler *auth.EntraIDOAuthHandler
	if cfg.Source.Type == "azuredevops" && cfg.Source.Token != "" {
		// Validate ADO configuration before attempting connection
		if cfg.Source.BaseURL == "" {
			logger.Warn("Azure DevOps integration disabled: SOURCE_BASE_URL not configured")
		} else {
			// Create ADO client
			adoClient, err := azuredevops.NewClient(azuredevops.ClientConfig{
				OrganizationURL:     cfg.Source.BaseURL,
				PersonalAccessToken: cfg.Source.Token,
				Logger:              logger,
			})
			if err != nil {
				logger.Warn("Failed to create ADO client - check your ADO organization URL and PAT permissions",
					"error", err,
					"org_url", cfg.Source.BaseURL,
					"hint", "Ensure SOURCE_BASE_URL is a valid Azure DevOps organization URL (e.g., https://dev.azure.com/your-org)")
			} else {
				adoHandler = handlers.NewADOHandler(mainHandler, adoClient, sourceProvider)
				// Link the ADO handler to the main handler so it can delegate ADO repo operations
				mainHandler.SetADOHandler(adoHandler)
				logger.Info("Azure DevOps integration enabled", "org_url", cfg.Source.BaseURL)
			}
		}

		// Create Entra ID OAuth handler if enabled
		if cfg.Auth.Enabled && cfg.Auth.EntraIDEnabled {
			entraIDOAuthHandler = auth.NewEntraIDOAuthHandler(&cfg.Auth)
			logger.Info("Entra ID OAuth enabled for Azure DevOps")
		}
	}

	return &Server{
		config:              cfg,
		db:                  db,
		logger:              logger,
		handler:             mainHandler,
		authHandler:         authHandler,
		adoHandler:          adoHandler,
		entraIDOAuthHandler: entraIDOAuthHandler,
		shutdownChan:        make(chan struct{}),
	}
}

// ShutdownChan returns the shutdown channel for graceful server shutdown
func (s *Server) ShutdownChan() chan struct{} {
	return s.shutdownChan
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

	// Entra ID OAuth endpoints for Azure DevOps (public - no auth required)
	if s.config.Auth.Enabled && s.entraIDOAuthHandler != nil {
		mux.HandleFunc("GET /api/v1/auth/entraid/login", s.entraIDOAuthHandler.Login)
		mux.HandleFunc("GET /api/v1/auth/entraid/callback", s.entraIDOAuthHandler.Callback)

		// Protected Entra ID endpoints
		if authMiddleware != nil {
			mux.Handle("GET /api/v1/auth/entraid/user", authMiddleware.RequireAuth(http.HandlerFunc(s.entraIDOAuthHandler.GetUser)))
		}
	}

	// Health check (always public)
	mux.HandleFunc("/health", s.handler.Health)

	// Application config endpoint (always public)
	mux.HandleFunc("GET /api/v1/config", s.handler.GetConfig)

	// Setup endpoints (public for initial configuration)
	setupHandler := handlers.NewSetupHandler(s.db, s.logger, s.config, s.shutdownChan)
	mux.HandleFunc("GET /api/v1/setup/status", setupHandler.GetSetupStatus)
	mux.HandleFunc("POST /api/v1/setup/validate-source", setupHandler.ValidateSource)
	mux.HandleFunc("POST /api/v1/setup/validate-destination", setupHandler.ValidateDestination)
	mux.HandleFunc("POST /api/v1/setup/validate-database", setupHandler.ValidateDatabase)
	mux.HandleFunc("POST /api/v1/setup/apply", setupHandler.ApplySetup)

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
	protect("POST /api/v1/repositories/batch-update", s.handler.BatchUpdateRepositoryStatus)
	// Repository GET route handles both repo details and dependencies via suffix detection
	protect("GET /api/v1/repositories/{fullName...}", s.handler.GetRepositoryOrDependencies)
	protect("PATCH /api/v1/repositories/{fullName...}", s.handler.UpdateRepository)
	// For action routes, we need to parse the action from the fullName in the handler
	protect("POST /api/v1/repositories/{fullName...}", s.handler.HandleRepositoryAction)

	// Dependency graph endpoints
	protect("GET /api/v1/dependencies/graph", s.handler.GetDependencyGraph)
	protect("GET /api/v1/dependencies/export", s.handler.ExportDependencies)

	// Organization endpoints
	protect("GET /api/v1/organizations", s.handler.ListOrganizations)
	protect("GET /api/v1/organizations/list", s.handler.GetOrganizationList)
	protect("GET /api/v1/projects", s.handler.ListProjects)

	// Team endpoints (GitHub only)
	protect("GET /api/v1/teams", s.handler.ListTeams)
	protect("GET /api/v1/teams/{org}/{teamSlug}/members", s.handler.GetTeamMembers)

	// Team mapping endpoints
	protect("GET /api/v1/team-mappings", s.handler.ListTeamMappings)
	protect("GET /api/v1/team-mappings/stats", s.handler.GetTeamMappingStats)
	protect("POST /api/v1/team-mappings", s.handler.CreateTeamMapping)
	protect("PATCH /api/v1/team-mappings/{sourceOrg}/{sourceTeamSlug}", s.handler.UpdateTeamMapping)
	protect("DELETE /api/v1/team-mappings/{sourceOrg}/{sourceTeamSlug}", s.handler.DeleteTeamMapping)
	protect("POST /api/v1/team-mappings/import", s.handler.ImportTeamMappings)
	protect("GET /api/v1/team-mappings/export", s.handler.ExportTeamMappings)
	protect("POST /api/v1/team-mappings/suggest", s.handler.SuggestTeamMappings)
	protect("POST /api/v1/team-mappings/sync", s.handler.SyncTeamMappingsFromDiscovery)

	// Permission audit endpoint
	protect("GET /api/v1/analytics/permission-audit", s.handler.GetPermissionAudit)

	// User discovery and mapping endpoints
	protect("GET /api/v1/users", s.handler.ListUsers)
	protect("GET /api/v1/users/stats", s.handler.GetUserStats)
	protect("GET /api/v1/user-mappings", s.handler.ListUserMappings)
	protect("GET /api/v1/user-mappings/stats", s.handler.GetUserMappingStats)
	protect("POST /api/v1/user-mappings", s.handler.CreateUserMapping)
	protect("PATCH /api/v1/user-mappings/{sourceLogin}", s.handler.UpdateUserMapping)
	protect("DELETE /api/v1/user-mappings/{sourceLogin}", s.handler.DeleteUserMapping)
	protect("POST /api/v1/user-mappings/import", s.handler.ImportUserMappings)
	protect("GET /api/v1/user-mappings/export", s.handler.ExportUserMappings)
	protect("GET /api/v1/user-mappings/generate-gei-csv", s.handler.GenerateGEICSV)
	protect("POST /api/v1/user-mappings/suggest", s.handler.SuggestUserMappings)
	protect("POST /api/v1/user-mappings/sync", s.handler.SyncUserMappingsFromDiscovery)
	protect("POST /api/v1/user-mappings/fetch-mannequins", s.handler.FetchMannequins)
	protect("POST /api/v1/user-mappings/reclaim-mannequins", s.handler.ReclaimMannequins)

	// Dashboard endpoints
	protect("GET /api/v1/dashboard/action-items", s.handler.GetDashboardActionItems)

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
	protect("GET /api/v1/analytics/detailed-discovery-report/export", s.handler.ExportDetailedDiscoveryReport)

	// Azure DevOps specific endpoints
	if s.adoHandler != nil {
		protect("POST /api/v1/ado/discover", s.adoHandler.StartADODiscovery)
		protect("GET /api/v1/ado/discovery/status", s.adoHandler.ADODiscoveryStatus)
		protect("GET /api/v1/ado/projects", s.adoHandler.ListADOProjects)
		protect("GET /api/v1/ado/projects/{organization}/{project}", s.adoHandler.GetADOProject)
	}

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
// This function prevents path traversal attacks by ensuring the resolved path
// stays within the allowed frontend directory boundaries.
func (s *Server) validateFrontendPath(requestPath, frontendDir string) (absDir, absPath string, ok bool) {
	// Clean the path to remove any path traversal sequences like ../
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

	// Security check: Use filepath.Rel to verify the resolved path is within frontendDir
	// This is more robust than string prefix checking and prevents directory traversal
	relPath, err := filepath.Rel(absDir, absPath)
	if err != nil {
		s.logger.Warn("Failed to compute relative path", "requested_path", requestPath, "error", err)
		return "", "", false
	}

	// Reject if the relative path tries to go up (..) or is absolute
	if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		s.logger.Warn("Path traversal attempt detected", "requested_path", requestPath, "resolved_path", absPath, "relative_path", relPath)
		return "", "", false
	}

	return absDir, absPath, true
}

// tryServeFile attempts to serve a file if it exists and is not a directory
func (s *Server) tryServeFile(w http.ResponseWriter, r *http.Request, absPath string) bool {
	// Open the file explicitly to avoid path injection concerns
	// #nosec G304 -- absPath is validated by validateFrontendPath to be within frontendDir bounds
	// codeql[go/path-injection] -- absPath is sanitized by filepath.Clean, validated by filepath.Abs, and checked to be within frontendDir
	file, err := os.Open(absPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Get file info to check if it's a directory
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	// Set appropriate content type
	s.setContentType(w, absPath)

	// Use ServeContent instead of ServeFile for explicit control
	// The absPath has been validated by validateFrontendPath to be within frontendDir
	http.ServeContent(w, r, filepath.Base(absPath), info.ModTime(), file)
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
