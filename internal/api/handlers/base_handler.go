package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// BaseHandler provides common dependencies and utilities for domain-specific handlers.
// This struct can be embedded by domain handlers to provide shared functionality.
//
// Usage:
//
//	type RepositoryHandler struct {
//	    BaseHandler
//	    // domain-specific dependencies
//	}
type BaseHandler struct {
	DB               *storage.Database
	Logger           *slog.Logger
	SourceDualClient *github.DualClient
	DestDualClient   *github.DualClient
	AuthConfig       *config.AuthConfig
	SourceBaseURL    string
	SourceType       string
}

// NewBaseHandler creates a new BaseHandler with the provided dependencies.
func NewBaseHandler(
	db *storage.Database,
	logger *slog.Logger,
	sourceDualClient *github.DualClient,
	destDualClient *github.DualClient,
	authConfig *config.AuthConfig,
	sourceBaseURL string,
	sourceType string,
) *BaseHandler {
	return &BaseHandler{
		DB:               db,
		Logger:           logger,
		SourceDualClient: sourceDualClient,
		DestDualClient:   destDualClient,
		AuthConfig:       authConfig,
		SourceBaseURL:    sourceBaseURL,
		SourceType:       sourceType,
	}
}

// SendJSON sends a JSON response with the specified status code.
func (b *BaseHandler) SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		b.Logger.Error("Failed to encode JSON response", "error", err)
	}
}

// SendError sends an error response using the standardized APIError format.
func (b *BaseHandler) SendError(w http.ResponseWriter, err *APIError) {
	WriteError(w, *err)
}

// HandleContextError checks if an error is due to request cancellation and logs appropriately.
// Returns true if the error is a context cancellation (caller should return early).
func (b *BaseHandler) HandleContextError(ctx context.Context, err error, operation string, r *http.Request) bool {
	if ctx.Err() == context.Canceled {
		b.Logger.Debug("Request canceled by client",
			"operation", operation,
			"path", r.URL.Path,
			"method", r.Method)
		return true
	}
	if ctx.Err() == context.DeadlineExceeded {
		b.Logger.Warn("Request timeout",
			"operation", operation,
			"path", r.URL.Path,
			"method", r.Method,
			"error", err)
		return true
	}
	return false
}

// CheckRepositoryAccess validates that the user has access to a specific repository.
// Returns an error if auth is enabled and user doesn't have access.
func (b *BaseHandler) CheckRepositoryAccess(ctx context.Context, repoFullName string) error {
	// If auth is not enabled, allow access
	if b.AuthConfig == nil || !b.AuthConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if b.SourceDualClient == nil {
		b.Logger.Warn("Cannot check repository access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := b.SourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, b.AuthConfig, b.Logger, b.SourceBaseURL)

	// Check if user has access to this repository
	hasAccess, err := checker.HasRepoAccess(ctx, user, token, repoFullName)
	if err != nil {
		return fmt.Errorf("failed to check repository access: %w", err)
	}

	if !hasAccess {
		return fmt.Errorf("you don't have admin access to repository: %s", repoFullName)
	}

	return nil
}

// CheckRepositoriesAccess validates that the user has access to all specified repositories.
// Returns an error if auth is enabled and user doesn't have access to any repository.
func (b *BaseHandler) CheckRepositoriesAccess(ctx context.Context, repoFullNames []string) error {
	// If auth is not enabled, allow access
	if b.AuthConfig == nil || !b.AuthConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if b.SourceDualClient == nil {
		b.Logger.Warn("Cannot check repositories access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := b.SourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, b.AuthConfig, b.Logger, b.SourceBaseURL)

	// Validate access to all repositories
	return checker.ValidateRepositoryAccess(ctx, user, token, repoFullNames)
}

// IsAuthEnabled returns whether authentication is enabled.
func (b *BaseHandler) IsAuthEnabled() bool {
	return b.AuthConfig != nil && b.AuthConfig.Enabled
}

// GetInitiatingUser extracts the authenticated username from the context.
// Returns nil if auth is disabled or user not found.
func (b *BaseHandler) GetInitiatingUser(ctx context.Context) *string {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil
	}
	username := user.Login
	return &username
}

// DomainHandler is an interface that all domain-specific handlers should implement.
// This interface allows for registration and discovery of handlers.
type DomainHandler interface {
	// RegisterRoutes registers this handler's routes with the mux.
	// The protect function wraps handlers with authentication middleware when enabled.
	RegisterRoutes(mux *http.ServeMux, protect func(pattern string, handler http.HandlerFunc))
}
