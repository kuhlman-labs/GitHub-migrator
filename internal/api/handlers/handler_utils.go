package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
)

// HandlerUtils provides shared utility functions for handlers.
// This is used to avoid code duplication across domain-specific handlers.
type HandlerUtils struct {
	authConfig       *config.AuthConfig
	sourceDualClient *github.DualClient
	sourceBaseConfig *github.ClientConfig
	sourceBaseURL    string
	logger           *slog.Logger
}

// NewHandlerUtils creates a new HandlerUtils instance.
func NewHandlerUtils(
	authConfig *config.AuthConfig,
	sourceDualClient *github.DualClient,
	sourceBaseConfig *github.ClientConfig,
	sourceBaseURL string,
	logger *slog.Logger,
) *HandlerUtils {
	return &HandlerUtils{
		authConfig:       authConfig,
		sourceDualClient: sourceDualClient,
		sourceBaseConfig: sourceBaseConfig,
		sourceBaseURL:    sourceBaseURL,
		logger:           logger,
	}
}

// CheckRepositoryAccess validates that the user has access to a specific repository.
// Returns an error if auth is enabled and user doesn't have access.
func (u *HandlerUtils) CheckRepositoryAccess(ctx context.Context, repoFullName string) error {
	// If auth is not enabled, allow access
	if u.authConfig == nil || !u.authConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if u.sourceDualClient == nil {
		u.logger.Warn("Cannot check repository access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := u.sourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, u.authConfig, u.logger, u.sourceBaseURL)

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
func (u *HandlerUtils) CheckRepositoriesAccess(ctx context.Context, repoFullNames []string) error {
	// If auth is not enabled, allow access
	if u.authConfig == nil || !u.authConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if u.sourceDualClient == nil {
		u.logger.Warn("Cannot check repositories access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := u.sourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, u.authConfig, u.logger, u.sourceBaseURL)

	// Validate access to all repositories
	return checker.ValidateRepositoryAccess(ctx, user, token, repoFullNames)
}

// GetClientForOrg returns the appropriate GitHub client for an organization.
func (u *HandlerUtils) GetClientForOrg(ctx context.Context, org string) (*github.Client, error) {
	// Check if we're in JWT-only mode (App auth without installation ID)
	isJWTOnlyMode := u.sourceBaseConfig != nil &&
		u.sourceBaseConfig.AppID > 0 &&
		u.sourceBaseConfig.AppInstallationID == 0

	if isJWTOnlyMode {
		u.logger.Debug("Creating org-specific client for single-repo operation",
			"org", org,
			"app_id", u.sourceBaseConfig.AppID)

		// Use the JWT client to get the installation ID for this org
		jwtClient := u.sourceDualClient.APIClient()
		installationID, err := jwtClient.GetOrganizationInstallationID(ctx, org)
		if err != nil {
			return nil, fmt.Errorf("failed to get installation ID for org %s: %w", org, err)
		}

		// Create org-specific client
		orgConfig := *u.sourceBaseConfig
		orgConfig.AppInstallationID = installationID

		orgClient, err := github.NewClient(orgConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create org-specific client for %s: %w", org, err)
		}

		u.logger.Debug("Created org-specific client",
			"org", org,
			"installation_id", installationID)

		return orgClient, nil
	}

	// Use the existing API client (PAT or App with installation ID)
	return u.sourceDualClient.APIClient(), nil
}
