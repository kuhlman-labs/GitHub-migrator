package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/configsvc"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// HandlerUtils provides shared utility functions for handlers.
// This is used to avoid code duplication across domain-specific handlers.
type HandlerUtils struct {
	authConfig       *config.AuthConfig
	destBaseURL      string // Destination GitHub API URL for authorization checks
	sourceDualClient *github.DualClient
	sourceBaseConfig *github.ClientConfig
	sourceBaseURL    string
	logger           *slog.Logger
	db               *storage.Database  // For source lookups and identity mapping
	configSvc        *configsvc.Service // For dynamic config (enterprise slug, auth rules)
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

// SetDestinationBaseURL sets the destination GitHub API URL for authorization checks
func (u *HandlerUtils) SetDestinationBaseURL(destBaseURL string) {
	u.destBaseURL = destBaseURL
}

// SetConfigService sets the config service for dynamic config access
func (u *HandlerUtils) SetConfigService(configSvc *configsvc.Service) {
	u.configSvc = configSvc
}

// getEffectiveAuthConfig returns the auth config with database settings merged in
func (u *HandlerUtils) getEffectiveAuthConfig() *config.AuthConfig {
	// If we have configSvc, use the effective config which includes database settings
	if u.configSvc != nil {
		effectiveCfg := u.configSvc.GetEffectiveAuthConfig()
		return &effectiveCfg
	}
	// Fall back to static config
	return u.authConfig
}

// CheckRepositoryAccess validates that the user has access to migrate a specific repository.
// Uses the destination-centric authorization model:
// - Tier 1 (Admin): Full access to all repos
// - Tier 2 (Self-Service): Access to repos where mapped source identity has admin rights
// - Tier 3 (Read-Only): No migration access
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

	// Check user's authorization tier
	destURL := u.destBaseURL
	if destURL == "" {
		destURL = "https://api.github.com"
	}
	authorizer := auth.NewAuthorizer(u.getEffectiveAuthConfig(), u.logger, destURL)

	// Check for Tier 1: Full migration rights
	hasFullAccess, reason, err := authorizer.CheckDestinationMigrationRights(ctx, user, token)
	if err != nil {
		u.logger.Warn("Failed to check destination migration rights", "user", user.Login, "error", err)
		// Continue to check other authorization methods
	} else if hasFullAccess {
		u.logger.Debug("User has full migration access",
			"user", user.Login,
			"repo", repoFullName,
			"reason", reason)
		return nil
	}

	// Check for Tier 2: Self-service via identity mapping
	return u.checkIdentityMappedAccess(ctx, user, repoFullName)
}

// checkIdentityMappedAccess checks if a user can access a repository via identity mapping
// The user's destination GitHub account must be mapped to a source identity,
// and that source identity must have admin access on the repository
func (u *HandlerUtils) checkIdentityMappedAccess(ctx context.Context, user *auth.GitHubUser, repoFullName string) error {
	rules := u.authConfig.AuthorizationRules

	// Check if identity mapping is required
	if !rules.RequireIdentityMappingForSelfService {
		// Identity mapping not required - allow self-service without verification
		u.logger.Debug("Identity mapping not required, allowing self-service access",
			"user", user.Login,
			"repo", repoFullName)
		return nil
	}

	// Identity mapping is required - look up user's source identity
	if u.db == nil {
		return fmt.Errorf("self-service migrations require identity mapping, but the system is not configured correctly")
	}

	// Look up the user's identity mapping by their GitHub (destination) login
	mapping, err := u.db.GetUserMappingByDestinationLogin(ctx, user.Login)
	if err != nil {
		u.logger.Warn("Failed to look up identity mapping", "user", user.Login, "error", err)
		return fmt.Errorf("failed to verify identity mapping: please try again later")
	}

	if mapping == nil {
		u.logger.Debug("User has no identity mapping", "user", user.Login)
		return fmt.Errorf("self-service migrations require identity mapping; please complete identity mapping in User Mappings to migrate repositories")
	}

	if mapping.MappingStatus != string(models.UserMappingStatusMapped) {
		u.logger.Debug("User identity mapping is not in mapped status",
			"user", user.Login,
			"status", mapping.MappingStatus)
		return fmt.Errorf("your identity mapping is incomplete (status: %s); please complete identity mapping to migrate repositories", mapping.MappingStatus)
	}

	// Get the repository's source to check permissions
	repoSource, err := u.getRepositorySource(ctx, repoFullName)
	if err != nil {
		u.logger.Debug("Could not determine repository source", "repo", repoFullName, "error", err)
		return fmt.Errorf("unable to verify repository access: %w", err)
	}

	if repoSource == nil {
		u.logger.Debug("Repository has no source assigned", "repo", repoFullName)
		return fmt.Errorf("repository %s has no source configured; contact an administrator", repoFullName)
	}

	// Verify the mapped source identity has admin access on the repository
	return u.checkSourceIdentityAccess(ctx, mapping.SourceLogin, repoFullName, repoSource)
}

// checkSourceIdentityAccess verifies that a source identity has admin access on a repository
// This uses the source's PAT (not the user's token) to check permissions
func (u *HandlerUtils) checkSourceIdentityAccess(ctx context.Context, sourceLogin string, repoFullName string, source *models.Source) error {
	u.logger.Debug("Checking source identity access",
		"source_login", sourceLogin,
		"repo", repoFullName,
		"source_id", source.ID,
		"source_name", source.Name)

	if !source.IsGitHub() {
		// For non-GitHub sources (like Azure DevOps), we trust the identity mapping
		// since we can't easily query permissions via API
		u.logger.Debug("Non-GitHub source: trusting identity mapping for access",
			"source_login", sourceLogin,
			"repo", repoFullName)
		return nil
	}

	// For GitHub sources, we need to check if the source identity has admin access
	// We use the source's PAT to query the repository collaborators
	// Note: This requires the PAT to have admin access to the repository

	// Parse org and repo name
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository name format: %s", repoFullName)
	}
	org, repo := parts[0], parts[1]

	// Create a GitHub client using the source's PAT
	clientConfig := github.ClientConfig{
		BaseURL: source.BaseURL,
		Token:   source.Token,
	}
	client, err := github.NewClient(clientConfig)
	if err != nil {
		u.logger.Warn("Failed to create GitHub client for source", "source_id", source.ID, "error", err)
		return fmt.Errorf("unable to verify repository access: please contact an administrator")
	}

	// Check if the source identity is a collaborator with admin permissions
	hasAccess, err := client.CheckCollaboratorPermission(ctx, org, repo, sourceLogin)
	if err != nil {
		u.logger.Warn("Failed to check collaborator permission",
			"source_login", sourceLogin,
			"repo", repoFullName,
			"error", err)
		// If we can't verify, we deny access for security
		return fmt.Errorf("unable to verify your access to %s: %w", repoFullName, err)
	}

	if !hasAccess {
		u.logger.Info("Source identity does not have admin access to repository",
			"source_login", sourceLogin,
			"repo", repoFullName)
		return fmt.Errorf("your source identity (%s) does not have admin access to %s", sourceLogin, repoFullName)
	}

	u.logger.Debug("Source identity has admin access to repository",
		"source_login", sourceLogin,
		"repo", repoFullName)
	return nil
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

// CheckRepositoriesAccess validates that the user has access to all specified repositories.
// Returns an error if auth is enabled and user doesn't have access to any repository.
func (u *HandlerUtils) CheckRepositoriesAccess(ctx context.Context, repoFullNames []string) error {
	// If auth is not enabled, allow access
	if u.authConfig == nil || !u.authConfig.Enabled {
		return nil
	}

	// Check each repository individually
	for _, repoFullName := range repoFullNames {
		if err := u.CheckRepositoryAccess(ctx, repoFullName); err != nil {
			return err
		}
	}

	return nil
}

// GetUserAuthorizationStatus returns the current user's authorization tier and details
// This is used by the authorization-status API endpoint
func (u *HandlerUtils) GetUserAuthorizationStatus(ctx context.Context) (*UserAuthorizationStatus, error) {
	if u.authConfig == nil || !u.authConfig.Enabled {
		return &UserAuthorizationStatus{
			Tier:     string(auth.TierAdmin),
			TierName: "Full Migration Rights",
			Permissions: auth.TierPermissions{
				CanViewRepos:       true,
				CanMigrateOwnRepos: true,
				CanMigrateAllRepos: true,
				CanManageBatches:   true,
				CanManageSources:   true,
			},
			IdentityMapping: nil,
			UpgradePath:     nil,
		}, nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return nil, fmt.Errorf("authentication required")
	}

	// Get base tier from authorizer
	destURL := u.destBaseURL
	if destURL == "" {
		destURL = "https://api.github.com"
	}
	authorizer := auth.NewAuthorizer(u.getEffectiveAuthConfig(), u.logger, destURL)

	tierInfo, err := authorizer.GetUserAuthorizationTier(ctx, user, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization tier: %w", err)
	}

	status := &UserAuthorizationStatus{
		Tier:        string(tierInfo.Tier),
		TierName:    tierInfo.TierName,
		Permissions: tierInfo.Permissions,
	}

	// Check identity mapping status
	if u.db != nil {
		mapping, err := u.db.GetUserMappingByDestinationLogin(ctx, user.Login)
		if err != nil {
			u.logger.Warn("Failed to look up identity mapping for status", "user", user.Login, "error", err)
		} else if mapping != nil {
			var sourceID *int64
			var sourceName string
			if mapping.SourceID != nil {
				sourceID = mapping.SourceID
				if source, err := u.db.GetSource(ctx, *mapping.SourceID); err == nil && source != nil {
					sourceName = source.Name
				}
			}
			status.IdentityMapping = &IdentityMappingStatus{
				Completed:   mapping.MappingStatus == string(models.UserMappingStatusMapped),
				SourceLogin: mapping.SourceLogin,
				SourceID:    sourceID,
				SourceName:  sourceName,
			}

			// If user has identity mapping completed and is currently read-only, upgrade to self-service
			if tierInfo.Tier == auth.TierReadOnly && mapping.MappingStatus == string(models.UserMappingStatusMapped) {
				status.Tier = string(auth.TierSelfService)
				status.TierName = "Self-Service"
				status.Permissions = auth.TierPermissions{
					CanViewRepos:       true,
					CanMigrateOwnRepos: true,
					CanMigrateAllRepos: false,
					CanManageBatches:   true,
					CanManageSources:   false,
				}
			}
		}
	}

	// Add upgrade path for users who can improve their access
	if status.Tier == string(auth.TierReadOnly) {
		status.UpgradePath = &UpgradePath{
			Action:  "complete_identity_mapping",
			Message: "Complete identity mapping to enable self-service migrations",
			Link:    "/user-mappings",
		}
	}

	return status, nil
}

// UserAuthorizationStatus represents the full authorization status for the API response
type UserAuthorizationStatus struct {
	Tier            string                 `json:"tier"`
	TierName        string                 `json:"tier_name"`
	Permissions     auth.TierPermissions   `json:"permissions"`
	IdentityMapping *IdentityMappingStatus `json:"identity_mapping,omitempty"`
	UpgradePath     *UpgradePath           `json:"upgrade_path,omitempty"`
}

// IdentityMappingStatus represents the user's identity mapping status
type IdentityMappingStatus struct {
	Completed   bool   `json:"completed"`
	SourceLogin string `json:"source_login,omitempty"`
	SourceID    *int64 `json:"source_id,omitempty"`
	SourceName  string `json:"source_name,omitempty"`
}

// UpgradePath represents how a user can upgrade their access tier
type UpgradePath struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	Link    string `json:"link"`
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

// getMappingStatsWithFilters is a helper to reduce duplication in stats handlers.
// It parses the source_id filter from the request and returns it.
func getMappingStatsWithFilters(r *http.Request) *int {
	var sourceID *int
	if sourceIDStr := r.URL.Query().Get("source_id"); sourceIDStr != "" {
		if sid, err := strconv.Atoi(sourceIDStr); err == nil {
			sourceID = &sid
		}
	}
	return sourceID
}

// handleMappingStatsRequest is a generic helper for handling mapping stats requests
// to reduce code duplication between team and user mapping stats handlers.
func (h *Handler) handleMappingStatsRequest(
	w http.ResponseWriter,
	r *http.Request,
	orgQueryParam string,
	entityType string,
	getStatsFn func(ctx context.Context, orgFilter string, sourceID *int) (interface{}, error),
) {
	ctx := r.Context()
	orgFilter := r.URL.Query().Get(orgQueryParam)
	sourceID := getMappingStatsWithFilters(r)

	stats, err := getStatsFn(ctx, orgFilter, sourceID)
	if err != nil {
		if h.handleContextError(ctx, err, "get "+entityType+" mapping stats", r) {
			return
		}
		h.logger.Error("Failed to get "+entityType+" mapping stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails(entityType+" mapping stats"))
		return
	}

	h.sendJSON(w, http.StatusOK, stats)
}
