package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// HandlerUtils provides shared utility functions for handlers.
// This is used to avoid code duplication across domain-specific handlers.
type HandlerUtils struct {
	authConfig       *config.AuthConfig
	sourceDualClient *github.DualClient
	sourceBaseConfig *github.ClientConfig
	sourceBaseURL    string
	logger           *slog.Logger
	db               *storage.Database // For source lookups
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

// SetDatabase sets the database for source lookups
func (u *HandlerUtils) SetDatabase(db *storage.Database) {
	u.db = db
}

// CheckRepositoryAccess validates that the user has access to a specific repository.
// Returns an error if auth is enabled and user doesn't have access.
// Supports source-scoped authentication: checks permissions via the authenticated source's API.
// If a repository's source has no OAuth configured, access is allowed (trusting admin-configured PAT).
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

	// Get user's authenticated source (if any)
	claims, hasClaims := auth.GetClaimsFromContext(ctx)
	userSourceID := int64(0)
	if hasClaims && claims.SourceID != nil {
		userSourceID = *claims.SourceID
	}

	// Try to get the repository to find its source
	repoSource, err := u.getRepositorySource(ctx, repoFullName)
	if err != nil {
		u.logger.Debug("Could not determine repository source, falling back to legacy check",
			"repo", repoFullName, "error", err)
		return u.checkDestinationBasedAccess(ctx, user, token, repoFullName)
	}

	// If repository has no source assigned, use legacy permission checking
	if repoSource == nil {
		u.logger.Debug("Repository has no source assigned, using legacy permission check",
			"repo", repoFullName)
		return u.checkDestinationBasedAccess(ctx, user, token, repoFullName)
	}

	// If the source has no OAuth configured, allow access
	// (trusting admin-configured PAT for this source)
	if !repoSource.HasOAuth() {
		u.logger.Debug("Source has no OAuth configured, allowing access",
			"repo", repoFullName,
			"source_id", repoSource.ID,
			"source_name", repoSource.Name)
		return nil
	}

	// Source has OAuth configured - user must be authenticated for this source
	if userSourceID != repoSource.ID {
		u.logger.Warn("User authenticated for different source than repository",
			"user", user.Login,
			"repo", repoFullName,
			"user_source_id", userSourceID,
			"repo_source_id", repoSource.ID,
			"repo_source_name", repoSource.Name)
		return fmt.Errorf("you are not authenticated for this repository's source (%s); please re-authenticate", repoSource.Name)
	}

	// User is authenticated for the correct source - check permissions
	return u.checkSourceScopedAccess(ctx, user, token, repoFullName, repoSource.ID, repoSource.Type)
}

// getRepositorySource looks up the source for a repository
func (u *HandlerUtils) getRepositorySource(ctx context.Context, repoFullName string) (*models.Source, error) {
	if u.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Get the repository
	repo, err := u.db.GetRepository(ctx, repoFullName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found")
	}

	// If repository has no source assigned, return nil
	if repo.SourceID == nil {
		return nil, nil
	}

	// Get the source
	source, err := u.db.GetSource(ctx, *repo.SourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return source, nil
}

// checkSourceScopedAccess checks repository access using the authenticated source's API
func (u *HandlerUtils) checkSourceScopedAccess(ctx context.Context, user *auth.GitHubUser, token string, repoFullName string, sourceID int64, sourceType string) error {
	u.logger.Debug("Checking source-scoped repository access",
		"user", user.Login,
		"repo", repoFullName,
		"source_id", sourceID,
		"source_type", sourceType)

	// Get the source to determine the API URL
	if u.db == nil {
		u.logger.Warn("Cannot check source-scoped access: database not available")
		return nil // Allow access if we can't check
	}

	source, err := u.db.GetSource(ctx, sourceID)
	if err != nil {
		u.logger.Warn("Failed to get source for permission check", "source_id", sourceID, "error", err)
		return nil // Allow access if we can't verify
	}
	if source == nil {
		u.logger.Warn("Source not found for permission check", "source_id", sourceID)
		return nil // Allow access if source not found
	}

	// For source-scoped auth, we use the user's OAuth token to check their permissions
	// The token was obtained from the source's OAuth flow, so it's valid for the source's API
	if source.IsGitHub() {
		return u.checkGitHubSourceAccess(ctx, user, token, repoFullName, source)
	}

	if source.IsAzureDevOps() {
		// For Azure DevOps, we currently allow access if user is authenticated
		// ADO doesn't expose fine-grained repo permissions via API the same way GitHub does
		u.logger.Debug("Azure DevOps source: allowing authenticated user access",
			"user", user.Login,
			"repo", repoFullName)
		return nil
	}

	return fmt.Errorf("unsupported source type for permission checking: %s", source.Type)
}

// checkGitHubSourceAccess checks repository access via GitHub source API
func (u *HandlerUtils) checkGitHubSourceAccess(ctx context.Context, user *auth.GitHubUser, token string, repoFullName string, source *models.Source) error {
	// Create a permission checker pointing to the source's API
	// The user's token is their OAuth token from authenticating with this source
	checker := auth.NewPermissionChecker(nil, u.authConfig, u.logger, source.BaseURL)

	hasAccess, err := checker.HasRepoAccess(ctx, user, token, repoFullName)
	if err != nil {
		return fmt.Errorf("failed to check repository access: %w", err)
	}

	if !hasAccess {
		return fmt.Errorf("you don't have admin access to repository: %s", repoFullName)
	}

	return nil
}

// checkDestinationBasedAccess checks repository access using the destination OAuth (legacy)
func (u *HandlerUtils) checkDestinationBasedAccess(ctx context.Context, user *auth.GitHubUser, token string, repoFullName string) error {
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
// Uses the same source-aware logic as CheckRepositoryAccess for each repository.
func (u *HandlerUtils) CheckRepositoriesAccess(ctx context.Context, repoFullNames []string) error {
	// If auth is not enabled, allow access
	if u.authConfig == nil || !u.authConfig.Enabled {
		return nil
	}

	// Check each repository individually using the source-aware logic
	for _, repoFullName := range repoFullNames {
		if err := u.CheckRepositoryAccess(ctx, repoFullName); err != nil {
			return err
		}
	}

	return nil
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
