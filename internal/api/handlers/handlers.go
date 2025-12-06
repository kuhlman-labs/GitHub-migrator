package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/discovery"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

const (
	statusInProgress = "in_progress"
	statusReady      = "ready"
	statusPending    = "pending"
	boolTrue         = "true"

	formatCSV  = "csv"
	formatJSON = "json"

	sourceTypeGitHub      = "github"
	sourceTypeAzureDevOps = "azuredevops"

	// Complexity categories
	categoryComplex     = "complex"
	categoryVeryComplex = "very_complex"

	// Size categories
	categorySizeVeryLarge = "very_large"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	cleanFullNameKey contextKey = "cleanFullName"
)

// Handler contains all HTTP handlers
type Handler struct {
	db               *storage.Database
	logger           *slog.Logger
	sourceDualClient *github.DualClient
	destDualClient   *github.DualClient
	collector        *discovery.Collector
	sourceBaseConfig *github.ClientConfig // For creating org-specific clients (JWT-only mode)
	authConfig       *config.AuthConfig   // Auth configuration for permission checks
	sourceBaseURL    string               // Source GitHub base URL for permission checks
	sourceType       string               // Source type: sourceTypeGitHub or sourceTypeAzureDevOps
	adoHandler       *ADOHandler          // ADO-specific handler (set by server if ADO is configured)
}

// SetADOHandler sets the ADO handler reference for delegating ADO operations
func (h *Handler) SetADOHandler(adoHandler *ADOHandler) {
	h.adoHandler = adoHandler
}

// NewHandler creates a new Handler instance
// sourceProvider can be nil if discovery is not needed
// sourceBaseConfig is used for per-org client creation in enterprise discovery (can be nil for PAT-only mode)
// authConfig is used for permission checks (can be nil if auth is disabled)
// sourceBaseURL is the source GitHub base URL for permission checks
func NewHandler(db *storage.Database, logger *slog.Logger, sourceDualClient *github.DualClient, destDualClient *github.DualClient, sourceProvider source.Provider, sourceBaseConfig *github.ClientConfig, authConfig *config.AuthConfig, sourceBaseURL string, sourceType string) *Handler {
	var collector *discovery.Collector
	// Use API client for discovery operations (will use App client if available, otherwise PAT)
	if sourceDualClient != nil && sourceProvider != nil {
		apiClient := sourceDualClient.APIClient()
		collector = discovery.NewCollector(apiClient, db, logger, sourceProvider)

		// If we have a base config with GitHub App credentials, set it on the collector
		// This enables per-org client creation for enterprise-wide discovery
		if sourceBaseConfig != nil {
			collector.WithBaseConfig(*sourceBaseConfig)
		}
	}
	return &Handler{
		db:               db,
		logger:           logger,
		sourceDualClient: sourceDualClient,
		destDualClient:   destDualClient,
		collector:        collector,
		sourceBaseConfig: sourceBaseConfig,
		authConfig:       authConfig,
		sourceBaseURL:    sourceBaseURL,
		sourceType:       sourceType,
	}
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// GetConfig handles GET /api/v1/config
// Returns application-level configuration for the frontend
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	// Default to github if not set
	sourceType := h.sourceType
	if sourceType == "" {
		sourceType = sourceTypeGitHub
	}

	response := map[string]interface{}{
		"source_type":  sourceType,
		"auth_enabled": h.authConfig != nil && h.authConfig.Enabled,
	}

	// Add Entra ID enabled flag if auth is enabled
	if h.authConfig != nil && h.authConfig.Enabled {
		response["entraid_enabled"] = h.authConfig.EntraIDEnabled
	}

	h.sendJSON(w, http.StatusOK, response)
}

// getClientForOrg returns the appropriate GitHub client for an organization
// If using JWT-only mode, creates an org-specific client with installation token
// Otherwise, returns the existing API client
func (h *Handler) getClientForOrg(ctx context.Context, org string) (*github.Client, error) {
	// Check if we're in JWT-only mode (App auth without installation ID)
	isJWTOnlyMode := h.sourceBaseConfig != nil &&
		h.sourceBaseConfig.AppID > 0 &&
		h.sourceBaseConfig.AppInstallationID == 0

	if isJWTOnlyMode {
		h.logger.Debug("Creating org-specific client for single-repo operation",
			"org", org,
			"app_id", h.sourceBaseConfig.AppID)

		// Use the JWT client to get the installation ID for this org
		jwtClient := h.sourceDualClient.APIClient()
		installationID, err := jwtClient.GetOrganizationInstallationID(ctx, org)
		if err != nil {
			return nil, fmt.Errorf("failed to get installation ID for org %s: %w", org, err)
		}

		// Create org-specific client
		orgConfig := *h.sourceBaseConfig
		orgConfig.AppInstallationID = installationID

		orgClient, err := github.NewClient(orgConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create org-specific client for %s: %w", org, err)
		}

		h.logger.Debug("Created org-specific client",
			"org", org,
			"installation_id", installationID)

		return orgClient, nil
	}

	// Use the existing API client (PAT or App with installation ID)
	return h.sourceDualClient.APIClient(), nil
}

// StartDiscovery handles POST /api/v1/discovery/start
func (h *Handler) StartDiscovery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Organization   string `json:"organization,omitempty"`
		EnterpriseSlug string `json:"enterprise_slug,omitempty"`
		Workers        int    `json:"workers,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate that either organization or enterprise is provided, but not both
	if req.Organization == "" && req.EnterpriseSlug == "" {
		h.sendError(w, http.StatusBadRequest, "Either organization or enterprise_slug is required")
		return
	}

	if req.Organization != "" && req.EnterpriseSlug != "" {
		h.sendError(w, http.StatusBadRequest, "Cannot specify both organization and enterprise_slug")
		return
	}

	if h.collector == nil {
		h.sendError(w, http.StatusServiceUnavailable, "GitHub client not configured")
		return
	}

	// Set workers if specified
	if req.Workers > 0 {
		h.collector.SetWorkers(req.Workers)
	}

	// Start discovery asynchronously based on type
	if req.EnterpriseSlug != "" {
		// Enterprise-wide discovery
		go func() {
			ctx := context.Background()
			if err := h.collector.DiscoverEnterpriseRepositories(ctx, req.EnterpriseSlug); err != nil {
				h.logger.Error("Enterprise discovery failed", "error", err, "enterprise", req.EnterpriseSlug)
			}
		}()

		h.sendJSON(w, http.StatusAccepted, map[string]string{
			"message":    "Enterprise discovery started",
			"enterprise": req.EnterpriseSlug,
			"status":     statusInProgress,
			"type":       "enterprise",
		})
	} else {
		// Organization discovery
		go func() {
			ctx := context.Background()
			if err := h.collector.DiscoverRepositories(ctx, req.Organization); err != nil {
				h.logger.Error("Discovery failed", "error", err, "org", req.Organization)
			}
		}()

		h.sendJSON(w, http.StatusAccepted, map[string]string{
			"message":      "Discovery started",
			"organization": req.Organization,
			"status":       statusInProgress,
			"type":         "organization",
		})
	}
}

// DiscoveryStatus handles GET /api/v1/discovery/status
func (h *Handler) DiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Count total repositories discovered
	count, err := h.db.CountRepositories(ctx, nil)
	if err != nil {
		if h.handleContextError(ctx, err, "count repositories", r) {
			return
		}
		h.logger.Error("Failed to count repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get discovery status")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":             "complete",
		"repositories_found": count,
		"completed_at":       time.Now().Format(time.RFC3339),
	})
}

// ListRepositories handles GET /api/v1/repositories
//
//nolint:gocyclo // Complexity is due to multiple query parameter handlers
func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Build filters from query parameters
	filters := make(map[string]interface{})

	if status := r.URL.Query().Get("status"); status != "" {
		// Support comma-separated status values
		if strings.Contains(status, ",") {
			statusParts := strings.Split(status, ",")
			// Trim spaces from each status
			statusArray := make([]string, len(statusParts))
			for i, s := range statusParts {
				statusArray[i] = strings.TrimSpace(s)
			}
			h.logger.Info("Status filter (array)", "status", statusArray)
			filters["status"] = statusArray
		} else {
			h.logger.Info("Status filter (single)", "status", status)
			filters["status"] = status
		}
	}

	if batchIDStr := r.URL.Query().Get("batch_id"); batchIDStr != "" {
		if batchID, err := strconv.ParseInt(batchIDStr, 10, 64); err == nil {
			filters["batch_id"] = batchID
		}
	}

	if source := r.URL.Query().Get("source"); source != "" {
		filters["source"] = source
	}

	if hasLFS := r.URL.Query().Get("has_lfs"); hasLFS != "" {
		filters["has_lfs"] = hasLFS == boolTrue
	}

	if hasSubmodules := r.URL.Query().Get("has_submodules"); hasSubmodules != "" {
		filters["has_submodules"] = hasSubmodules == boolTrue
	}

	// Organization filter (can be comma-separated list)
	if org := r.URL.Query().Get("organization"); org != "" {
		if strings.Contains(org, ",") {
			filters["organization"] = strings.Split(org, ",")
		} else {
			filters["organization"] = org
		}
	}

	// ADO Organization filter (for Azure DevOps - can be comma-separated list)
	// This filters by ado_projects.organization rather than the full_name prefix
	if adoOrg := r.URL.Query().Get("ado_organization"); adoOrg != "" {
		if strings.Contains(adoOrg, ",") {
			filters["ado_organization"] = strings.Split(adoOrg, ",")
		} else {
			filters["ado_organization"] = adoOrg
		}
	}

	// Project filter (for Azure DevOps - can be comma-separated list)
	if project := r.URL.Query().Get("project"); project != "" {
		if strings.Contains(project, ",") {
			filters["ado_project"] = strings.Split(project, ",")
		} else {
			filters["ado_project"] = project
		}
	}

	// Team filter (for GitHub - can be comma-separated list)
	// Format: "org/team-slug" to uniquely identify teams across organizations
	if team := r.URL.Query().Get("team"); team != "" {
		if strings.Contains(team, ",") {
			filters["team"] = strings.Split(team, ",")
		} else {
			filters["team"] = team
		}
	}

	// Size range filters (in bytes)
	if minSizeStr := r.URL.Query().Get("min_size"); minSizeStr != "" {
		if minSize, err := strconv.ParseInt(minSizeStr, 10, 64); err == nil {
			filters["min_size"] = minSize
		}
	}
	if maxSizeStr := r.URL.Query().Get("max_size"); maxSizeStr != "" {
		if maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64); err == nil {
			filters["max_size"] = maxSize
		}
	}

	// Feature filters
	if hasActions := r.URL.Query().Get("has_actions"); hasActions != "" {
		filters["has_actions"] = hasActions == boolTrue
	}
	if hasWiki := r.URL.Query().Get("has_wiki"); hasWiki != "" {
		filters["has_wiki"] = hasWiki == boolTrue
	}
	if hasPages := r.URL.Query().Get("has_pages"); hasPages != "" {
		filters["has_pages"] = hasPages == boolTrue
	}
	if hasDiscussions := r.URL.Query().Get("has_discussions"); hasDiscussions != "" {
		filters["has_discussions"] = hasDiscussions == boolTrue
	}
	if hasProjects := r.URL.Query().Get("has_projects"); hasProjects != "" {
		filters["has_projects"] = hasProjects == boolTrue
	}
	if hasLargeFiles := r.URL.Query().Get("has_large_files"); hasLargeFiles != "" {
		filters["has_large_files"] = hasLargeFiles == boolTrue
	}
	if hasBranchProtections := r.URL.Query().Get("has_branch_protections"); hasBranchProtections != "" {
		filters["has_branch_protections"] = hasBranchProtections == boolTrue
	}
	if isArchived := r.URL.Query().Get("is_archived"); isArchived != "" {
		filters["is_archived"] = isArchived == boolTrue
	}
	if isFork := r.URL.Query().Get("is_fork"); isFork != "" {
		filters["is_fork"] = isFork == boolTrue
	}
	if hasPackages := r.URL.Query().Get("has_packages"); hasPackages != "" {
		filters["has_packages"] = hasPackages == boolTrue
	}
	if hasRulesets := r.URL.Query().Get("has_rulesets"); hasRulesets != "" {
		filters["has_rulesets"] = hasRulesets == boolTrue
	}
	if hasCodeScanning := r.URL.Query().Get("has_code_scanning"); hasCodeScanning != "" {
		filters["has_code_scanning"] = hasCodeScanning == boolTrue
	}
	if hasDependabot := r.URL.Query().Get("has_dependabot"); hasDependabot != "" {
		filters["has_dependabot"] = hasDependabot == boolTrue
	}
	if hasSecretScanning := r.URL.Query().Get("has_secret_scanning"); hasSecretScanning != "" {
		filters["has_secret_scanning"] = hasSecretScanning == boolTrue
	}
	if hasCodeowners := r.URL.Query().Get("has_codeowners"); hasCodeowners != "" {
		filters["has_codeowners"] = hasCodeowners == boolTrue
	}
	if hasSelfHostedRunners := r.URL.Query().Get("has_self_hosted_runners"); hasSelfHostedRunners != "" {
		filters["has_self_hosted_runners"] = hasSelfHostedRunners == boolTrue
	}
	if hasReleaseAssets := r.URL.Query().Get("has_release_assets"); hasReleaseAssets != "" {
		filters["has_release_assets"] = hasReleaseAssets == boolTrue
	}
	if hasWebhooks := r.URL.Query().Get("has_webhooks"); hasWebhooks != "" {
		filters["has_webhooks"] = hasWebhooks == boolTrue
	}
	if hasEnvironments := r.URL.Query().Get("has_environments"); hasEnvironments != "" {
		filters["has_environments"] = hasEnvironments == boolTrue
	}
	if hasSecrets := r.URL.Query().Get("has_secrets"); hasSecrets != "" {
		filters["has_secrets"] = hasSecrets == boolTrue
	}
	if hasVariables := r.URL.Query().Get("has_variables"); hasVariables != "" {
		filters["has_variables"] = hasVariables == boolTrue
	}

	// Azure DevOps feature filters
	if adoIsGit := r.URL.Query().Get("ado_is_git"); adoIsGit != "" {
		filters["ado_is_git"] = adoIsGit == boolTrue
	}
	if adoHasBoards := r.URL.Query().Get("ado_has_boards"); adoHasBoards != "" {
		filters["ado_has_boards"] = adoHasBoards == boolTrue
	}
	if adoHasPipelines := r.URL.Query().Get("ado_has_pipelines"); adoHasPipelines != "" {
		filters["ado_has_pipelines"] = adoHasPipelines == boolTrue
	}
	if adoHasGHAS := r.URL.Query().Get("ado_has_ghas"); adoHasGHAS != "" {
		filters["ado_has_ghas"] = adoHasGHAS == boolTrue
	}
	if adoPullRequestCount := r.URL.Query().Get("ado_pull_request_count"); adoPullRequestCount != "" {
		filters["ado_pull_request_count"] = adoPullRequestCount
	}
	if adoWorkItemCount := r.URL.Query().Get("ado_work_item_count"); adoWorkItemCount != "" {
		filters["ado_work_item_count"] = adoWorkItemCount
	}
	if adoBranchPolicyCount := r.URL.Query().Get("ado_branch_policy_count"); adoBranchPolicyCount != "" {
		filters["ado_branch_policy_count"] = adoBranchPolicyCount
	}
	if adoYAMLPipelineCount := r.URL.Query().Get("ado_yaml_pipeline_count"); adoYAMLPipelineCount != "" {
		filters["ado_yaml_pipeline_count"] = adoYAMLPipelineCount
	}
	if adoClassicPipelineCount := r.URL.Query().Get("ado_classic_pipeline_count"); adoClassicPipelineCount != "" {
		filters["ado_classic_pipeline_count"] = adoClassicPipelineCount
	}
	if adoHasWiki := r.URL.Query().Get("ado_has_wiki"); adoHasWiki != "" {
		filters["ado_has_wiki"] = adoHasWiki == boolTrue
	}
	if adoTestPlanCount := r.URL.Query().Get("ado_test_plan_count"); adoTestPlanCount != "" {
		filters["ado_test_plan_count"] = adoTestPlanCount
	}
	if adoPackageFeedCount := r.URL.Query().Get("ado_package_feed_count"); adoPackageFeedCount != "" {
		filters["ado_package_feed_count"] = adoPackageFeedCount
	}
	if adoServiceHookCount := r.URL.Query().Get("ado_service_hook_count"); adoServiceHookCount != "" {
		filters["ado_service_hook_count"] = adoServiceHookCount
	}

	// Visibility filter
	if visibility := r.URL.Query().Get("visibility"); visibility != "" {
		filters["visibility"] = visibility
	}

	// Size category filter (can be comma-separated list)
	if sizeCategory := r.URL.Query().Get("size_category"); sizeCategory != "" {
		if strings.Contains(sizeCategory, ",") {
			filters["size_category"] = strings.Split(sizeCategory, ",")
		} else {
			filters["size_category"] = sizeCategory
		}
	}

	// Complexity filter (can be comma-separated list)
	if complexity := r.URL.Query().Get("complexity"); complexity != "" {
		if strings.Contains(complexity, ",") {
			filters["complexity"] = strings.Split(complexity, ",")
		} else {
			filters["complexity"] = complexity
		}
	}

	// Search filter
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}

	// Sort filter
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy
	}

	// Available for batch filter
	if availableForBatch := r.URL.Query().Get("available_for_batch"); availableForBatch == "true" {
		filters["available_for_batch"] = true
	}

	// Pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters["limit"] = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters["offset"] = offset
		}
	}

	repos, err := h.db.ListRepositories(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "list repositories", r) {
			return
		}
		h.logger.Error("Failed to list repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

	// Note: We don't filter repositories by permissions here for performance reasons.
	// Instead, permission checks are enforced at the action level (when users try to
	// migrate, add to batch, etc.). This provides better UX - users see all repos
	// immediately and get clear error messages only when they attempt unauthorized actions.

	// Get total count if pagination is used
	response := map[string]interface{}{
		"repositories": repos,
	}

	if _, hasLimit := filters["limit"]; hasLimit {
		// Count total matching repositories
		totalCount, err := h.db.CountRepositoriesWithFilters(ctx, filters)
		if err != nil {
			h.logger.Error("Failed to count repositories", "error", err)
		} else {
			response["total"] = totalCount
		}
	}

	h.sendJSON(w, http.StatusOK, response)
}

// HandleRepositoryAction routes POST requests to repository actions
// Pattern: POST /api/v1/repositories/{fullName...}
// Where fullName can be "org/repo/action" - we parse out the action suffix
func (h *Handler) HandleRepositoryAction(w http.ResponseWriter, r *http.Request) {
	fullPath := r.PathValue("fullName")
	if fullPath == "" {
		h.sendError(w, http.StatusBadRequest, "Repository path is required")
		return
	}

	// Parse the action from the path
	// Possible actions: rediscover, mark-remediated, unlock, rollback, mark-wont-migrate, reset
	var action string
	var fullName string

	if strings.HasSuffix(fullPath, "/rediscover") {
		action = "rediscover"
		fullName = strings.TrimSuffix(fullPath, "/rediscover")
	} else if strings.HasSuffix(fullPath, "/mark-remediated") {
		action = "mark-remediated"
		fullName = strings.TrimSuffix(fullPath, "/mark-remediated")
	} else if strings.HasSuffix(fullPath, "/unlock") {
		action = "unlock"
		fullName = strings.TrimSuffix(fullPath, "/unlock")
	} else if strings.HasSuffix(fullPath, "/rollback") {
		action = "rollback" //nolint:goconst // action strings are contextual to routing
		fullName = strings.TrimSuffix(fullPath, "/rollback")
	} else if strings.HasSuffix(fullPath, "/mark-wont-migrate") {
		action = "mark-wont-migrate"
		fullName = strings.TrimSuffix(fullPath, "/mark-wont-migrate")
	} else if strings.HasSuffix(fullPath, "/reset") {
		action = "reset"
		fullName = strings.TrimSuffix(fullPath, "/reset")
	} else {
		h.sendError(w, http.StatusNotFound, "Unknown repository action")
		return
	}

	// Check if user has permission to access this repository
	if err := h.checkRepositoryAccess(r.Context(), fullName); err != nil {
		h.logger.Warn("Repository access denied", "repo", fullName, "action", action, "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	// Create a new request with the cleaned fullName in path value
	// We'll pass the fullName directly to the handlers
	ctx := context.WithValue(r.Context(), cleanFullNameKey, fullName)
	r = r.WithContext(ctx)

	// Route to the appropriate handler
	switch action {
	case "rediscover":
		h.RediscoverRepository(w, r)
	case "mark-remediated":
		h.MarkRepositoryRemediated(w, r)
	case "unlock":
		h.UnlockRepository(w, r)
	case "rollback":
		h.RollbackRepository(w, r)
	case "mark-wont-migrate":
		h.MarkRepositoryWontMigrate(w, r)
	case "reset":
		h.ResetRepositoryStatus(w, r)
	default:
		h.sendError(w, http.StatusNotFound, "Unknown repository action")
	}
}

// GetRepository handles GET /api/v1/repositories/{fullName}
func (h *Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}
	h.getRepository(w, r, fullName)
}

// getRepository is the internal implementation
func (h *Handler) getRepository(w http.ResponseWriter, r *http.Request, fullName string) {
	// URL decode the fullName (Go's PathValue should decode, but we ensure it here)
	// This handles cases like "org%2Frepo" -> "org/repo"
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName // Use original if decode fails
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil {
		if h.handleContextError(ctx, err, "get repository", r) {
			return
		}
		h.logger.Error("Failed to get repository", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repository")
		return
	}

	if repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Get migration history
	history, err := h.db.GetMigrationHistory(ctx, repo.ID)
	if err != nil {
		if h.handleContextError(ctx, err, "get migration history", r) {
			return
		}
		h.logger.Error("Failed to get migration history", "error", err)
		// Continue without history
		history = []*models.MigrationHistory{}
	}

	response := map[string]interface{}{
		"repository": repo,
		"history":    history,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetRepositoryOrDependencies routes GET requests to either repository details or dependencies
// Pattern: GET /api/v1/repositories/{fullName...}
// Routes based on path suffix:
// - /dependencies -> get dependencies
// - /dependents -> get repos that depend on this one
// - /dependencies/export -> export dependencies
// - otherwise -> repository details
func (h *Handler) GetRepositoryOrDependencies(w http.ResponseWriter, r *http.Request) {
	fullPath := r.PathValue("fullName")
	if fullPath == "" {
		h.sendError(w, http.StatusBadRequest, "Repository path is required")
		return
	}

	// Check for dependencies/export first (more specific)
	if strings.HasSuffix(fullPath, "/dependencies/export") {
		h.ExportRepositoryDependencies(w, r)
		return
	}

	// Check if this is a dependents request
	if strings.HasSuffix(fullPath, "/dependents") {
		h.GetRepositoryDependents(w, r)
		return
	}

	// Check if this is a dependencies request
	if strings.HasSuffix(fullPath, "/dependencies") {
		// Strip /dependencies and call the dependencies handler
		fullName := strings.TrimSuffix(fullPath, "/dependencies")
		h.getRepositoryDependencies(w, r, fullName)
		return
	}

	// Regular repository details request
	h.getRepository(w, r, fullPath)
}

// GetRepositoryDependencies returns all dependencies for a repository
func (h *Handler) GetRepositoryDependencies(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}
	h.getRepositoryDependencies(w, r, fullName)
}

// getRepositoryDependencies is the internal implementation
func (h *Handler) getRepositoryDependencies(w http.ResponseWriter, r *http.Request, fullName string) {

	// URL decode the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	// Get dependencies from database
	dependencies, err := h.db.GetRepositoryDependenciesByFullName(r.Context(), decodedFullName)
	if err != nil {
		h.logger.Error("Failed to get repository dependencies",
			"repo", decodedFullName,
			"error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to retrieve dependencies")
		return
	}

	// Calculate summary statistics
	summary := struct {
		Total    int            `json:"total"`
		Local    int            `json:"local"`
		External int            `json:"external"`
		ByType   map[string]int `json:"by_type"`
	}{
		Total:  len(dependencies),
		ByType: make(map[string]int),
	}

	for _, dep := range dependencies {
		if dep.IsLocal {
			summary.Local++
		} else {
			summary.External++
		}
		summary.ByType[dep.DependencyType]++
	}

	// Return response with dependencies and summary
	response := map[string]interface{}{
		"dependencies": dependencies,
		"summary":      summary,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// UpdateRepository handles PATCH /api/v1/repositories/{fullName}
func (h *Handler) UpdateRepository(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// URL decode the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Apply allowed updates
	if batchID, ok := updates["batch_id"].(float64); ok {
		id := int64(batchID)
		repo.BatchID = &id
	}

	if priority, ok := updates["priority"].(float64); ok {
		repo.Priority = int(priority)
	}

	if destFullName, ok := updates["destination_full_name"].(string); ok {
		repo.DestinationFullName = &destFullName
	}

	// Allow updating exclusion flags
	if excludeReleases, ok := updates["exclude_releases"].(bool); ok {
		repo.ExcludeReleases = excludeReleases
	}

	if excludeAttachments, ok := updates["exclude_attachments"].(bool); ok {
		repo.ExcludeAttachments = excludeAttachments
	}

	if excludeMetadata, ok := updates["exclude_metadata"].(bool); ok {
		repo.ExcludeMetadata = excludeMetadata
	}

	if excludeGitData, ok := updates["exclude_git_data"].(bool); ok {
		repo.ExcludeGitData = excludeGitData
	}

	if excludeOwnerProjects, ok := updates["exclude_owner_projects"].(bool); ok {
		repo.ExcludeOwnerProjects = excludeOwnerProjects
	}

	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to update repository", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update repository")
		return
	}

	h.sendJSON(w, http.StatusOK, repo)
}

// ResetRepositoryStatus handles POST /api/v1/repositories/{fullName}/reset
// Resets a stuck repository back to pending status
func (h *Handler) ResetRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	decodedFullName, err := h.getDecodedRepoName(r)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Only allow resetting repositories in "stuck" states
	allowedStates := []string{
		string(models.StatusDryRunInProgress),
		string(models.StatusMigratingContent),
		string(models.StatusPreMigration),
		string(models.StatusArchiveGenerating),
		string(models.StatusPostMigration),
	}

	allowed := false
	for _, state := range allowedStates {
		if repo.Status == state {
			allowed = true
			break
		}
	}

	if !allowed {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Cannot reset repository in status '%s'. Only in-progress states can be reset.", repo.Status))
		return
	}

	h.logger.Info("Resetting repository status",
		"repo", repo.FullName,
		"old_status", repo.Status)

	// Reset to pending
	repo.Status = string(models.StatusPending)
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to reset repository status", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to reset repository status")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Repository status reset to pending",
		"repository": repo,
	})
}

// RediscoverRepository handles POST /api/v1/repositories/{fullName}/rediscover
func (h *Handler) RediscoverRepository(w http.ResponseWriter, r *http.Request) {
	decodedFullName, err := h.getDecodedRepoName(r)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Delegate to appropriate handler based on repository type
	if h.isADORepository(repo) {
		h.rediscoverADORepository(w, ctx, repo, decodedFullName)
		return
	}

	h.rediscoverGitHubRepository(w, ctx, decodedFullName)
}

// getDecodedRepoName extracts and decodes the repository name from the request
func (h *Handler) getDecodedRepoName(r *http.Request) (string, error) {
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		return "", fmt.Errorf("repository name is required")
	}

	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		return fullName, nil
	}

	return decodedFullName, nil
}

// isADORepository checks if a repository is from Azure DevOps
func (h *Handler) isADORepository(repo *models.Repository) bool {
	return repo.ADOProject != nil && *repo.ADOProject != ""
}

// rediscoverADORepository handles rediscovery of Azure DevOps repositories
func (h *Handler) rediscoverADORepository(w http.ResponseWriter, ctx context.Context, repo *models.Repository, decodedFullName string) {
	if h.adoHandler == nil {
		h.sendError(w, http.StatusServiceUnavailable, "ADO discovery service not configured")
		return
	}

	h.logger.Info("Delegating rediscovery to ADO handler", "repo", decodedFullName)

	if err := h.adoHandler.RediscoverADORepository(ctx, repo); err != nil {
		h.logger.Error("Failed to rediscover ADO repository", "error", err, "repo", decodedFullName)
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to rediscover repository: %v", err))
		return
	}

	updatedRepo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil {
		h.logger.Error("Failed to fetch updated repository", "error", err, "repo", decodedFullName)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch updated repository")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Repository rediscovered successfully",
		"repository": updatedRepo,
	})
}

// rediscoverGitHubRepository handles rediscovery of GitHub repositories
func (h *Handler) rediscoverGitHubRepository(w http.ResponseWriter, ctx context.Context, decodedFullName string) {
	if h.collector == nil {
		h.sendError(w, http.StatusServiceUnavailable, "GitHub discovery service not configured")
		return
	}

	parts := strings.SplitN(decodedFullName, "/", 2)
	if len(parts) != 2 {
		h.sendError(w, http.StatusBadRequest, "Invalid repository name format")
		return
	}
	org, repoName := parts[0], parts[1]

	client, err := h.getClientForOrg(ctx, org)
	if err != nil {
		h.logger.Error("Failed to get client for organization", "error", err, "org", org)
		h.sendError(w, http.StatusInternalServerError, "Failed to initialize client for repository")
		return
	}

	ghRepo, _, err := client.REST().Repositories.Get(ctx, org, repoName)
	if err != nil {
		h.logger.Error("Failed to fetch repository from GitHub", "error", err, "repo", decodedFullName)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repository from GitHub")
		return
	}

	h.startAsyncRediscovery(ghRepo, decodedFullName)

	h.sendJSON(w, http.StatusAccepted, map[string]string{
		"message":   "Re-discovery started",
		"full_name": decodedFullName,
		"status":    "in_progress",
	})
}

// startAsyncRediscovery runs repository discovery asynchronously
func (h *Handler) startAsyncRediscovery(ghRepo *ghapi.Repository, decodedFullName string) {
	go func() {
		bgCtx := context.Background()
		if err := h.collector.ProfileRepository(bgCtx, ghRepo); err != nil {
			h.logger.Error("Re-discovery failed", "error", err, "repo", decodedFullName)
		} else {
			h.logger.Info("Re-discovery completed", "repo", decodedFullName)
			if err := h.db.UpdateLocalDependencyFlags(bgCtx); err != nil {
				h.logger.Warn("Failed to update local dependency flags after re-discovery", "error", err)
			}
		}
	}()
}

// MarkRepositoryRemediated handles POST /api/v1/repositories/{fullName}/mark-remediated
// This endpoint triggers a full re-discovery after the user has fixed blocking migration issues
func (h *Handler) MarkRepositoryRemediated(w http.ResponseWriter, r *http.Request) {
	// Get fullName from context (if routed via HandleRepositoryAction) or path value
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// URL decode the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	if h.collector == nil {
		h.sendError(w, http.StatusServiceUnavailable, "Discovery service not configured")
		return
	}

	ctx := r.Context()

	// Check if repository exists and has remediation_required status
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Verify the repository is in remediation_required status
	if repo.Status != string(models.StatusRemediationRequired) {
		h.sendError(w, http.StatusBadRequest,
			fmt.Sprintf("Repository status must be 'remediation_required', current status: '%s'", repo.Status))
		return
	}

	// Extract org and repo name from decodedFullName
	parts := strings.SplitN(decodedFullName, "/", 2)
	if len(parts) != 2 {
		h.sendError(w, http.StatusBadRequest, "Invalid repository name format")
		return
	}
	org, repoName := parts[0], parts[1]

	// Get the appropriate client for this organization
	client, err := h.getClientForOrg(ctx, org)
	if err != nil {
		h.logger.Error("Failed to get client for organization", "error", err, "org", org)
		h.sendError(w, http.StatusInternalServerError, "Failed to initialize client for repository")
		return
	}

	// Fetch repository from GitHub API
	ghRepo, _, err := client.REST().Repositories.Get(ctx, org, repoName)
	if err != nil {
		h.logger.Error("Failed to fetch repository from GitHub", "error", err, "repo", decodedFullName)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repository from GitHub")
		return
	}

	h.logger.Info("Starting re-validation after remediation",
		"repo", decodedFullName,
		"had_oversized_commits", repo.HasOversizedCommits,
		"had_long_refs", repo.HasLongRefs,
		"had_blocking_files", repo.HasBlockingFiles)

	// Run full discovery asynchronously (git analysis + API profiling + validation)
	go func() {
		bgCtx := context.Background()
		if err := h.collector.ProfileRepository(bgCtx, ghRepo); err != nil {
			h.logger.Error("Re-validation after remediation failed", "error", err, "repo", decodedFullName)
		} else {
			h.logger.Info("Re-validation completed", "repo", decodedFullName)

			// Update local dependency flags after re-validation
			// This ensures dependencies are correctly classified as local/external
			if err := h.db.UpdateLocalDependencyFlags(bgCtx); err != nil {
				h.logger.Warn("Failed to update local dependency flags after re-validation", "error", err)
			}
		}
	}()

	h.sendJSON(w, http.StatusAccepted, map[string]string{
		"message":   "Re-validation started - repository will be re-analyzed for migration limits",
		"full_name": decodedFullName,
		"status":    "validating",
	})
}

// UnlockRepository handles POST /api/v1/repositories/{fullName}/unlock
func (h *Handler) UnlockRepository(w http.ResponseWriter, r *http.Request) {
	// Get fullName from context (if routed via HandleRepositoryAction) or path value
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// URL decode the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	if h.sourceDualClient == nil {
		h.sendError(w, http.StatusServiceUnavailable, "Source client not configured")
		return
	}

	ctx := r.Context()

	// Get repository
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Check if repository has lock information
	if repo.SourceMigrationID == nil {
		h.sendError(w, http.StatusBadRequest, "No migration ID found for this repository")
		return
	}

	if !repo.IsSourceLocked {
		h.sendJSON(w, http.StatusOK, map[string]string{
			"message":   "Repository is not locked",
			"full_name": fullName,
		})
		return
	}

	// Extract org and repo name
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		h.sendError(w, http.StatusBadRequest, "Invalid repository name format")
		return
	}
	org, repoName := parts[0], parts[1]

	// Unlock the repository using migration client (PAT required)
	migrationClient := h.sourceDualClient.MigrationClient()
	err = migrationClient.UnlockRepository(ctx, org, repoName, *repo.SourceMigrationID)
	if err != nil {
		h.logger.Error("Failed to unlock repository", "error", err, "repo", fullName)
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to unlock repository: %v", err))
		return
	}

	// Update repository lock status
	repo.IsSourceLocked = false
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to update repository lock status", "error", err)
		// Continue anyway, the unlock was successful
	}

	h.logger.Info("Repository unlocked successfully", "repo", fullName, "migration_id", *repo.SourceMigrationID)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Repository unlocked successfully",
		"full_name":    fullName,
		"migration_id": *repo.SourceMigrationID,
	})
}

// RollbackRepository handles POST /api/v1/repositories/{fullName}/rollback
func (h *Handler) RollbackRepository(w http.ResponseWriter, r *http.Request) {
	// Get fullName from context (if routed via HandleRepositoryAction) or path value
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// URL decode the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	ctx := r.Context()

	// Get repository
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Validate repository status is complete
	if repo.Status != string(models.StatusComplete) {
		h.sendError(w, http.StatusBadRequest, "Only completed migrations can be rolled back")
		return
	}

	// Parse request body for optional reason
	var req struct {
		Reason string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is okay, reason is optional
		req.Reason = ""
	}

	// Perform rollback
	if err := h.db.RollbackRepository(ctx, decodedFullName, req.Reason); err != nil {
		h.logger.Error("Failed to rollback repository", "error", err, "repo", decodedFullName)
		h.sendError(w, http.StatusInternalServerError, "Failed to rollback repository")
		return
	}

	h.logger.Info("Repository rolled back successfully", "repo", fullName, "reason", req.Reason)

	// Get updated repository
	repo, _ = h.db.GetRepository(ctx, fullName)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Repository rolled back successfully",
		"repository": repo,
	})
}

// MarkRepositoryWontMigrate handles POST /api/v1/repositories/{fullName}/mark-wont-migrate
func (h *Handler) MarkRepositoryWontMigrate(w http.ResponseWriter, r *http.Request) {
	// Get fullName from context (if routed via HandleRepositoryAction) or path value
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// URL decode the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	ctx := r.Context()

	// Get repository
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Parse request body for action (mark or unmark)
	var req struct {
		Unmark bool `json:"unmark,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is okay, defaults to marking
		req.Unmark = false
	}

	var newStatus string
	var message string

	if req.Unmark {
		// Unmark: change from wont_migrate back to pending
		if repo.Status != string(models.StatusWontMigrate) {
			h.sendError(w, http.StatusBadRequest, "Repository is not marked as won't migrate")
			return
		}
		newStatus = string(models.StatusPending)
		message = "Repository unmarked - changed to pending status"
	} else {
		// Mark: only allow marking from certain statuses
		allowedStatuses := map[string]bool{
			string(models.StatusPending):         true,
			string(models.StatusDryRunComplete):  true,
			string(models.StatusDryRunFailed):    true,
			string(models.StatusMigrationFailed): true,
			string(models.StatusRolledBack):      true,
		}

		if !allowedStatuses[repo.Status] {
			h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Cannot mark repository with status '%s' as won't migrate", repo.Status))
			return
		}

		newStatus = string(models.StatusWontMigrate)
		message = "Repository marked as won't migrate"
	}

	// Remove from batch if assigned
	if repo.BatchID != nil && !req.Unmark {
		repo.BatchID = nil
	}

	// Update status
	repo.Status = newStatus
	repo.UpdatedAt = time.Now()
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to update repository status", "error", err, "repo", fullName)
		h.sendError(w, http.StatusInternalServerError, "Failed to update repository")
		return
	}

	h.logger.Info("Repository wont_migrate status changed", "repo", fullName, "status", newStatus)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":    message,
		"repository": repo,
	})
}

// ListBatches handles GET /api/v1/batches
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	batches, err := h.db.ListBatches(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "list batches", r) {
			return
		}
		h.logger.Error("Failed to list batches", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batches")
		return
	}
	h.sendJSON(w, http.StatusOK, batches)
}

// CreateBatch handles POST /api/v1/batches
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var batch models.Batch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate batch name is provided
	if strings.TrimSpace(batch.Name) == "" {
		h.sendError(w, http.StatusBadRequest, "Batch name is required")
		return
	}

	// Validate migration API if provided
	if batch.MigrationAPI != "" && batch.MigrationAPI != models.MigrationAPIGEI && batch.MigrationAPI != models.MigrationAPIELM {
		h.sendError(w, http.StatusBadRequest, "Invalid migration_api. Must be 'GEI' or 'ELM'")
		return
	}

	// Set default migration API if not specified
	if batch.MigrationAPI == "" {
		batch.MigrationAPI = models.MigrationAPIGEI
	}

	ctx := r.Context()
	batch.CreatedAt = time.Now()
	batch.Status = statusPending // Start batches in pending state

	if err := h.db.CreateBatch(ctx, &batch); err != nil {
		h.logger.Error("Failed to create batch", "error", err, "name", batch.Name)

		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "unique constraint") {
			h.sendError(w, http.StatusConflict, fmt.Sprintf("A batch with the name '%s' already exists. Please choose a different name.", batch.Name))
			return
		}

		h.sendError(w, http.StatusInternalServerError, "Failed to create batch")
		return
	}

	h.sendJSON(w, http.StatusCreated, batch)
}

// GetBatch handles GET /api/v1/batches/{id}
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Get repositories in batch
	repos, err := h.db.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		h.logger.Error("Failed to get batch repositories", "error", err)
		repos = []*models.Repository{}
	}

	response := map[string]interface{}{
		"batch":        batch,
		"repositories": repos,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// DryRunBatch handles POST /api/v1/batches/{id}/dry-run
//
//nolint:gocyclo // HTTP handler with multiple validation and processing steps
func (h *Handler) DryRunBatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()

	// Parse optional request body
	var req struct {
		OnlyPending bool `json:"only_pending,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Get batch
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Allow dry run for both "pending" and "ready" batches
	// (You can re-run dry runs even on ready batches)
	if batch.Status != statusPending && batch.Status != statusReady {
		h.sendError(w, http.StatusBadRequest, "Can only run dry run on batches with 'pending' or 'ready' status")
		return
	}

	// Get all repositories in batch
	repos, err := h.db.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		h.logger.Error("Failed to get batch repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

	if len(repos) == 0 {
		h.sendError(w, http.StatusBadRequest, "Batch has no repositories")
		return
	}

	// Queue repositories for dry run
	priority := 0
	if batch.Type == "pilot" {
		priority = 1
	}

	dryRunIDs := make([]int64, 0, len(repos))
	skippedCount := 0

	for _, repo := range repos {
		// If only_pending=true, only queue repos that need dry runs
		if req.OnlyPending {
			needsDryRun := repo.Status == string(models.StatusPending) ||
				repo.Status == string(models.StatusDryRunFailed) ||
				repo.Status == string(models.StatusMigrationFailed) ||
				repo.Status == string(models.StatusRolledBack)

			if !needsDryRun {
				skippedCount++
				continue
			}
		} else {
			// If running dry run on all, skip repos in terminal or active migration states
			if repo.Status == string(models.StatusComplete) ||
				repo.Status == string(models.StatusQueuedForMigration) ||
				repo.Status == string(models.StatusMigratingContent) ||
				repo.Status == string(models.StatusArchiveGenerating) ||
				repo.Status == string(models.StatusDryRunInProgress) ||
				repo.Status == string(models.StatusDryRunQueued) {
				skippedCount++
				continue
			}
		}

		repo.Status = string(models.StatusDryRunQueued)
		repo.Priority = priority

		if err := h.db.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to update repository", "error", err)
			continue
		}

		// Log the dry run initiation with user info
		initiatingUser := getInitiatingUser(ctx)
		logEntry := &models.MigrationLog{
			RepositoryID: repo.ID,
			Level:        "INFO",
			Phase:        "dry_run",
			Operation:    "queue",
			Message:      "Dry run queued",
			InitiatedBy:  initiatingUser,
		}
		if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
			h.logger.Warn("Failed to create migration log", "error", err)
		}

		dryRunIDs = append(dryRunIDs, repo.ID)
	}

	if len(dryRunIDs) == 0 {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("No repositories to run dry run. %d repositories were skipped.", skippedCount))
		return
	}

	// Update batch status to in_progress during dry run
	// Use UpdateBatchProgress to preserve user-configured fields like scheduled_at
	now := time.Now()
	if err := h.db.UpdateBatchProgress(ctx, batch.ID, statusInProgress, &now, &now, nil); err != nil {
		h.logger.Error("Failed to update batch progress", "error", err)
	}

	message := fmt.Sprintf("Started dry run for %d repositories in batch '%s'", len(dryRunIDs), batch.Name)
	if skippedCount > 0 {
		message += fmt.Sprintf(" (%d repositories skipped)", skippedCount)
	}

	response := map[string]interface{}{
		"batch_id":      batchID,
		"batch_name":    batch.Name,
		"dry_run_ids":   dryRunIDs,
		"count":         len(dryRunIDs),
		"skipped_count": skippedCount,
		"message":       message,
		"only_pending":  req.OnlyPending,
	}

	h.sendJSON(w, http.StatusAccepted, response)
}

// StartBatch handles POST /api/v1/batches/{id}/start
//
//nolint:gocyclo // Complexity justified for batch startup validation and orchestration
func (h *Handler) StartBatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()

	// Parse optional request body
	var req struct {
		SkipDryRun bool `json:"skip_dry_run,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Get batch
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Validate batch status
	if batch.Status == statusPending && !req.SkipDryRun {
		h.sendError(w, http.StatusBadRequest, "Batch is in 'pending' state. Run dry run first or set skip_dry_run=true")
		return
	}

	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only start batches with 'ready' or 'pending' status")
		return
	}

	// Get all repositories in batch
	repos, err := h.db.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		h.logger.Error("Failed to get batch repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

	if len(repos) == 0 {
		h.sendError(w, http.StatusBadRequest, "Batch has no repositories")
		return
	}

	// Check user has permission to access all repositories in the batch
	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Start batch access denied", "batch_id", batchID, "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	// Queue repositories for migration
	priority := 0
	if batch.Type == "pilot" {
		priority = 1
	}

	migrationIDs := make([]int64, 0, len(repos))
	for _, repo := range repos {
		if !canMigrate(repo.Status) {
			continue
		}

		repo.Status = string(models.StatusQueuedForMigration)
		repo.Priority = priority

		if err := h.db.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to update repository", "error", err)
			continue
		}

		// Log the migration initiation with user info
		initiatingUser := getInitiatingUser(ctx)
		logEntry := &models.MigrationLog{
			RepositoryID: repo.ID,
			Level:        "INFO",
			Phase:        "migration",
			Operation:    "queue",
			Message:      "Migration queued",
			InitiatedBy:  initiatingUser,
		}
		if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
			h.logger.Warn("Failed to create migration log", "error", err)
		}

		migrationIDs = append(migrationIDs, repo.ID)
	}

	// Update batch status
	// Use UpdateBatchProgress to preserve user-configured fields like scheduled_at
	now := time.Now()
	if err := h.db.UpdateBatchProgress(ctx, batch.ID, statusInProgress, &now, nil, &now); err != nil {
		h.logger.Error("Failed to update batch progress", "error", err)
	}

	response := map[string]interface{}{
		"batch_id":      batchID,
		"batch_name":    batch.Name,
		"migration_ids": migrationIDs,
		"count":         len(migrationIDs),
		"message":       fmt.Sprintf("Started migration for %d repositories in batch '%s'", len(migrationIDs), batch.Name),
	}

	h.sendJSON(w, http.StatusAccepted, response)
}

// UpdateBatch handles PATCH /api/v1/batches/{id}
//
//nolint:gocyclo // Update operations naturally involve multiple conditional checks
func (h *Handler) UpdateBatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()

	// Get existing batch
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Only allow updates for "pending" and "ready" batches
	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only edit batches with 'pending' or 'ready' status")
		return
	}

	// Parse update request
	var updates struct {
		Name            *string    `json:"name,omitempty"`
		Description     *string    `json:"description,omitempty"`
		Type            *string    `json:"type,omitempty"`
		ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
		DestinationOrg  *string    `json:"destination_org,omitempty"`
		MigrationAPI    *string    `json:"migration_api,omitempty"`
		ExcludeReleases *bool      `json:"exclude_releases,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate migration API if provided
	if updates.MigrationAPI != nil && *updates.MigrationAPI != models.MigrationAPIGEI && *updates.MigrationAPI != models.MigrationAPIELM {
		h.sendError(w, http.StatusBadRequest, "Invalid migration_api. Must be 'GEI' or 'ELM'")
		return
	}

	// Track if destination_org is being changed - we'll need to update repos
	oldDestinationOrg := ""
	newDestinationOrg := ""
	destinationOrgChanged := false

	if updates.DestinationOrg != nil {
		newDestinationOrg = *updates.DestinationOrg
		if batch.DestinationOrg != nil {
			oldDestinationOrg = *batch.DestinationOrg
		}
		// Destination changed if old and new are different (including nil -> value or value -> nil)
		destinationOrgChanged = oldDestinationOrg != newDestinationOrg
	}

	// Apply updates
	if updates.Name != nil {
		batch.Name = *updates.Name
	}
	if updates.Description != nil {
		batch.Description = updates.Description
	}
	if updates.Type != nil {
		batch.Type = *updates.Type
	}
	if updates.ScheduledAt != nil {
		batch.ScheduledAt = updates.ScheduledAt
	}
	if updates.DestinationOrg != nil {
		batch.DestinationOrg = updates.DestinationOrg
	}
	if updates.MigrationAPI != nil {
		batch.MigrationAPI = *updates.MigrationAPI
	}
	if updates.ExcludeReleases != nil {
		batch.ExcludeReleases = *updates.ExcludeReleases
	}

	if err := h.db.UpdateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to update batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update batch")
		return
	}

	// If destination_org changed, update repositories that were using the old batch default
	if destinationOrgChanged && oldDestinationOrg != "" {
		h.logger.Info("Batch destination_org changed, updating repository destinations",
			"batch_id", batchID,
			"old_destination_org", oldDestinationOrg,
			"new_destination_org", newDestinationOrg)

		// Get all repositories in this batch
		repos, err := h.db.ListRepositories(ctx, map[string]interface{}{"batch_id": batchID})
		if err != nil {
			h.logger.Error("Failed to list batch repositories for destination update", "error", err)
			// Don't fail the batch update, just log the error
		} else {
			updatedCount := 0
			for _, repo := range repos {
				if repo.DestinationFullName == nil || *repo.DestinationFullName == "" {
					continue
				}

				// Extract repo name from full_name (e.g., "org/repo" -> "repo")
				parts := strings.Split(repo.FullName, "/")
				if len(parts) != 2 {
					continue
				}
				repoName := parts[1]

				// Check if this repo was using the old batch default destination
				expectedOldDestination := fmt.Sprintf("%s/%s", oldDestinationOrg, repoName)
				if *repo.DestinationFullName == expectedOldDestination {
					// Update to new batch default destination
					if newDestinationOrg != "" {
						newDestination := fmt.Sprintf("%s/%s", newDestinationOrg, repoName)
						repo.DestinationFullName = &newDestination
					} else {
						// If new destination org is empty, clear the destination
						repo.DestinationFullName = nil
					}

					if err := h.db.UpdateRepository(ctx, repo); err != nil {
						h.logger.Error("Failed to update repository destination",
							"repo_id", repo.ID,
							"repo_name", repo.FullName,
							"error", err)
					} else {
						updatedCount++
						newDest := ""
						if repo.DestinationFullName != nil {
							newDest = *repo.DestinationFullName
						}
						h.logger.Debug("Updated repository destination",
							"repo_id", repo.ID,
							"repo_name", repo.FullName,
							"new_destination", newDest)
					}
				}
			}

			if updatedCount > 0 {
				h.logger.Info("Updated repository destinations for batch default change",
					"batch_id", batchID,
					"updated_count", updatedCount)
			}
		}
	}

	h.sendJSON(w, http.StatusOK, batch)
}

// DeleteBatch handles DELETE /api/v1/batches/{id}
func (h *Handler) DeleteBatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()

	// Get existing batch
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Only allow deletion of batches not in progress
	terminalStates := []string{"in_progress"}
	for _, state := range terminalStates {
		if batch.Status == state {
			h.sendError(w, http.StatusBadRequest, "Cannot delete batch in 'in_progress' status")
			return
		}
	}

	// Delete the batch (this will also clear batch_id from all repositories)
	if err := h.db.DeleteBatch(ctx, batchID); err != nil {
		h.logger.Error("Failed to delete batch", "error", err, "batch_id", batchID)
		h.sendError(w, http.StatusInternalServerError, "Failed to delete batch")
		return
	}

	h.logger.Info("Batch deleted successfully", "batch_id", batchID, "batch_name", batch.Name)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Batch deleted successfully",
	})
}

// AddRepositoriesToBatch handles POST /api/v1/batches/{id}/repositories
func (h *Handler) AddRepositoriesToBatch(w http.ResponseWriter, r *http.Request) { //nolint:gocyclo // TODO: refactor to reduce complexity
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()

	// Get batch
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Only allow adding repos to "pending" and "ready" batches
	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only add repositories to batches with 'pending' or 'ready' status")
		return
	}

	// Parse request
	var req struct {
		RepositoryIDs []int64 `json:"repository_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.RepositoryIDs) == 0 {
		h.sendError(w, http.StatusBadRequest, "No repository IDs provided")
		return
	}

	// Validate repositories are eligible for batch
	repos, err := h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
	if err != nil {
		h.logger.Error("Failed to get repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to validate repositories")
		return
	}

	// Check user has permission to access all repositories
	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Add repositories to batch access denied", "batch_id", batchID, "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	// Check each repo is eligible and separate into eligible and ineligible lists
	eligibleRepoIDs := []int64{}
	ineligibleRepos := []string{}
	ineligibleReasons := make(map[string]string)

	for _, repo := range repos {
		if eligible, reason := isRepositoryEligibleForBatch(repo); !eligible {
			ineligibleRepos = append(ineligibleRepos, repo.FullName)
			ineligibleReasons[repo.FullName] = reason
		} else {
			eligibleRepoIDs = append(eligibleRepoIDs, repo.ID)
		}
	}

	// If NO repos are eligible, return error
	if len(eligibleRepoIDs) == 0 {
		errorMsg := "No repositories are eligible for batch assignment:\n"
		for _, repoName := range ineligibleRepos {
			errorMsg += fmt.Sprintf("  - %s: %s\n", repoName, ineligibleReasons[repoName])
		}
		h.sendError(w, http.StatusBadRequest, strings.TrimSpace(errorMsg))
		return
	}

	// Add only eligible repositories to batch
	if err := h.db.AddRepositoriesToBatch(ctx, batchID, eligibleRepoIDs); err != nil {
		h.logger.Error("Failed to add repositories to batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to add repositories to batch")
		return
	}

	// Re-fetch repositories after adding them to batch to get updated batch_id
	// This prevents UpdateRepository from overwriting batch_id with the old nil value
	repos, err = h.db.GetRepositoriesByIDs(ctx, eligibleRepoIDs)
	if err != nil {
		h.logger.Error("Failed to re-fetch repositories after adding to batch", "error", err)
		// Continue anyway - defaults won't be applied but repos are in the batch
	}

	// Apply batch defaults to eligible repositories that don't have options set
	updatedCount := 0
	failedUpdates := []string{}

	for _, repo := range repos {

		needsUpdate := false

		// Apply destination org if batch has one and repo doesn't have a destination set
		if batch.DestinationOrg != nil && *batch.DestinationOrg != "" && repo.DestinationFullName == nil {
			destinationFullName := fmt.Sprintf("%s/%s", *batch.DestinationOrg, repo.Name())
			repo.DestinationFullName = &destinationFullName
			needsUpdate = true
		}

		// Apply exclude_releases setting from batch if repo doesn't have it set (assuming false is the default)
		// Only apply if batch has it enabled - we don't want to override if repo explicitly set to false
		if batch.ExcludeReleases && !repo.ExcludeReleases {
			repo.ExcludeReleases = batch.ExcludeReleases
			needsUpdate = true
		}

		if needsUpdate {
			repo.UpdatedAt = time.Now()
			if err := h.db.UpdateRepository(ctx, repo); err != nil {
				h.logger.Warn("Failed to apply batch defaults to repository", "repo", repo.FullName, "error", err)
				failedUpdates = append(failedUpdates, repo.FullName)
			} else {
				updatedCount++
			}
		}
	}

	// Get updated batch
	batch, _ = h.db.GetBatch(ctx, batchID)

	// Build response message
	message := fmt.Sprintf("Added %d of %d repositories to batch", len(eligibleRepoIDs), len(req.RepositoryIDs))
	if updatedCount > 0 {
		message += fmt.Sprintf(" (%d inherited batch defaults)", updatedCount)
	}
	if len(ineligibleRepos) > 0 {
		message += fmt.Sprintf(". %d repos skipped (ineligible)", len(ineligibleRepos))
	}
	if len(failedUpdates) > 0 {
		message += fmt.Sprintf(". %d repos failed to apply defaults", len(failedUpdates))
	}

	response := map[string]interface{}{
		"batch":                  batch,
		"repositories_added":     len(eligibleRepoIDs),
		"repositories_requested": len(req.RepositoryIDs),
		"defaults_applied_count": updatedCount,
		"message":                message,
	}

	// Include ineligible repos info for transparency
	if len(ineligibleRepos) > 0 {
		response["ineligible_count"] = len(ineligibleRepos)
		response["ineligible_repos"] = ineligibleReasons
	}

	// Include failed updates for transparency
	if len(failedUpdates) > 0 {
		response["failed_updates"] = failedUpdates
	}

	h.sendJSON(w, http.StatusOK, response)
}

// RemoveRepositoriesFromBatch handles DELETE /api/v1/batches/{id}/repositories
func (h *Handler) RemoveRepositoriesFromBatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()

	// Get batch
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Only allow removing repos from "pending" and "ready" batches
	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only remove repositories from batches with 'pending' or 'ready' status")
		return
	}

	// Parse request
	var req struct {
		RepositoryIDs []int64 `json:"repository_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.RepositoryIDs) == 0 {
		h.sendError(w, http.StatusBadRequest, "No repository IDs provided")
		return
	}

	// Remove repositories from batch
	if err := h.db.RemoveRepositoriesFromBatch(ctx, batchID, req.RepositoryIDs); err != nil {
		h.logger.Error("Failed to remove repositories from batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to remove repositories from batch")
		return
	}

	// Get updated batch
	batch, _ = h.db.GetBatch(ctx, batchID)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"batch":                batch,
		"repositories_removed": len(req.RepositoryIDs),
		"message":              fmt.Sprintf("Removed %d repositories from batch", len(req.RepositoryIDs)),
	})
}

// RetryBatchFailures handles POST /api/v1/batches/{id}/retry
//
//nolint:gocyclo // Complexity is due to multiple validation and error handling paths
func (h *Handler) RetryBatchFailures(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	ctx := r.Context()

	// Get batch
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Parse optional repository IDs
	var req struct {
		RepositoryIDs []int64 `json:"repository_ids,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is okay, means retry all
		req.RepositoryIDs = nil
	}

	var reposToRetry []*models.Repository

	if len(req.RepositoryIDs) > 0 {
		// Retry specific repositories
		reposToRetry, err = h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
		if err != nil {
			h.logger.Error("Failed to get repositories", "error", err)
			h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
			return
		}

		// Validate all repos are in this batch and failed
		for _, repo := range reposToRetry {
			if repo.BatchID == nil || *repo.BatchID != batchID {
				h.sendError(w, http.StatusBadRequest,
					fmt.Sprintf("Repository %s is not in this batch", repo.FullName))
				return
			}
			if repo.Status != string(models.StatusMigrationFailed) && repo.Status != string(models.StatusDryRunFailed) {
				h.sendError(w, http.StatusBadRequest,
					fmt.Sprintf("Repository %s is not in a failed state", repo.FullName))
				return
			}
		}
	} else {
		// Retry all failed repositories in batch
		filters := map[string]interface{}{
			"batch_id": batchID,
			"status": []string{
				string(models.StatusMigrationFailed),
				string(models.StatusDryRunFailed),
			},
		}
		reposToRetry, err = h.db.ListRepositories(ctx, filters)
		if err != nil {
			h.logger.Error("Failed to get failed repositories", "error", err)
			h.sendError(w, http.StatusInternalServerError, "Failed to fetch failed repositories")
			return
		}
	}

	if len(reposToRetry) == 0 {
		h.sendError(w, http.StatusBadRequest, "No failed repositories to retry")
		return
	}

	// Check user has permission to access all repositories
	repoFullNames := make([]string, len(reposToRetry))
	for i, repo := range reposToRetry {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Retry batch access denied", "batch_id", batchID, "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	// Queue repositories for retry
	retriedIDs := make([]int64, 0, len(reposToRetry))
	initiatingUser := getInitiatingUser(ctx)
	for _, repo := range reposToRetry {
		repo.Status = string(models.StatusQueuedForMigration)
		if err := h.db.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to update repository", "error", err, "repo", repo.FullName)
			continue
		}

		// Log the retry initiation with user info
		logEntry := &models.MigrationLog{
			RepositoryID: repo.ID,
			Level:        "INFO",
			Phase:        "migration",
			Operation:    "retry",
			Message:      "Migration retry queued",
			InitiatedBy:  initiatingUser,
		}
		if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
			h.logger.Warn("Failed to create migration log", "error", err)
		}

		retriedIDs = append(retriedIDs, repo.ID)
	}

	h.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"batch_id":      batchID,
		"batch_name":    batch.Name,
		"retried_count": len(retriedIDs),
		"retried_ids":   retriedIDs,
		"message":       fmt.Sprintf("Queued %d repositories for retry", len(retriedIDs)),
	})
}

// StartMigration handles POST /api/v1/migrations/start
type StartMigrationRequest struct {
	RepositoryIDs []int64  `json:"repository_ids,omitempty"`
	FullNames     []string `json:"full_names,omitempty"`
	DryRun        bool     `json:"dry_run"`
	Priority      int      `json:"priority"`
}

type StartMigrationResponse struct {
	MigrationIDs []int64 `json:"migration_ids"`
	Message      string  `json:"message"`
	Count        int     `json:"count"`
}

func (h *Handler) StartMigration(w http.ResponseWriter, r *http.Request) {
	var req StartMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	var repos []*models.Repository
	var err error

	// Support both repository IDs and full names
	if len(req.RepositoryIDs) > 0 {
		repos, err = h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
	} else if len(req.FullNames) > 0 {
		repos, err = h.db.GetRepositoriesByNames(ctx, req.FullNames)
	} else {
		h.sendError(w, http.StatusBadRequest, "Must provide repository_ids or full_names")
		return
	}

	if err != nil {
		h.logger.Error("Failed to fetch repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

	if len(repos) == 0 {
		h.sendError(w, http.StatusNotFound, "No repositories found")
		return
	}

	// Check user has permission to access all repositories
	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Start migration access denied", "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	// Start migrations asynchronously
	migrationIDs := make([]int64, 0, len(repos))
	for _, repo := range repos {
		// Validate repository can be migrated
		if !canMigrate(repo.Status) {
			h.logger.Warn("Repository cannot be migrated",
				"repo", repo.FullName,
				"status", repo.Status)
			continue
		}

		// Update status
		newStatus := models.StatusQueuedForMigration
		if req.DryRun {
			newStatus = models.StatusDryRunQueued
		}

		repo.Status = string(newStatus)
		repo.Priority = req.Priority

		if err := h.db.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to update repository",
				"repo", repo.FullName,
				"error", err)
			continue
		}

		// Log the migration initiation with user info
		initiatingUser := getInitiatingUser(ctx)
		phase := "migration"
		message := "Migration queued"
		if req.DryRun {
			phase = "dry_run"
			message = "Dry run queued"
		}
		logEntry := &models.MigrationLog{
			RepositoryID: repo.ID,
			Level:        "INFO",
			Phase:        phase,
			Operation:    "queue",
			Message:      message,
			InitiatedBy:  initiatingUser,
		}
		if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
			h.logger.Warn("Failed to create migration log", "error", err)
		}

		migrationIDs = append(migrationIDs, repo.ID)

		h.logger.Info("Migration queued",
			"repo", repo.FullName,
			"dry_run", req.DryRun)
	}

	response := StartMigrationResponse{
		MigrationIDs: migrationIDs,
		Count:        len(migrationIDs),
		Message:      fmt.Sprintf("Successfully queued %d repositories for migration", len(migrationIDs)),
	}

	h.sendJSON(w, http.StatusAccepted, response)
}

// BatchUpdateRepositoryStatusRequest defines the request for batch repository status updates
type BatchUpdateRepositoryStatusRequest struct {
	RepositoryIDs []int64  `json:"repository_ids,omitempty"`
	FullNames     []string `json:"full_names,omitempty"`
	Action        string   `json:"action"`           // "mark_migrated" | "mark_wont_migrate" | "unmark_wont_migrate" | "rollback"
	Reason        string   `json:"reason,omitempty"` // Optional reason for rollback
}

// BatchUpdateRepositoryStatusResponse defines the response for batch updates
type BatchUpdateRepositoryStatusResponse struct {
	UpdatedIDs   []int64  `json:"updated_ids"`
	FailedIDs    []int64  `json:"failed_ids,omitempty"`
	UpdatedCount int      `json:"updated_count"`
	FailedCount  int      `json:"failed_count"`
	Message      string   `json:"message"`
	Errors       []string `json:"errors,omitempty"`
}

// BatchUpdateRepositoryStatus handles POST /api/v1/repositories/batch-update
func (h *Handler) BatchUpdateRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateRepositoryStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateBatchAction(req.Action); err != nil {
		h.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	repos, err := h.fetchRepositoriesForBatchUpdate(ctx, req)
	if err != nil {
		h.handleBatchUpdateError(w, err)
		return
	}

	if err := h.checkBatchUpdatePermissions(ctx, repos); err != nil {
		h.logger.Warn("Batch update access denied", "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	response := h.executeBatchUpdate(ctx, repos, req.Action, req.Reason)
	statusCode := determineBatchUpdateStatusCode(response)
	h.sendJSON(w, statusCode, response)
}

// validateBatchAction validates the requested action
func validateBatchAction(action string) error {
	validActions := map[string]bool{
		"mark_migrated":       true,
		"mark_wont_migrate":   true,
		"unmark_wont_migrate": true,
		"rollback":            true,
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action. Must be 'mark_migrated', 'mark_wont_migrate', 'unmark_wont_migrate', or 'rollback'")
	}
	return nil
}

// fetchRepositoriesForBatchUpdate fetches repositories by IDs or names
func (h *Handler) fetchRepositoriesForBatchUpdate(ctx context.Context, req BatchUpdateRepositoryStatusRequest) ([]*models.Repository, error) {
	var repos []*models.Repository
	var err error

	if len(req.RepositoryIDs) > 0 {
		repos, err = h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
	} else if len(req.FullNames) > 0 {
		repos, err = h.db.GetRepositoriesByNames(ctx, req.FullNames)
	} else {
		return nil, fmt.Errorf("must provide repository_ids or full_names")
	}

	if err != nil {
		h.logger.Error("Failed to fetch repositories", "error", err)
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}

	return repos, nil
}

// handleBatchUpdateError handles errors from fetching repositories
func (h *Handler) handleBatchUpdateError(w http.ResponseWriter, err error) {
	if err.Error() == "no repositories found" {
		h.sendError(w, http.StatusNotFound, err.Error())
	} else if err.Error() == "must provide repository_ids or full_names" {
		h.sendError(w, http.StatusBadRequest, err.Error())
	} else {
		h.sendError(w, http.StatusInternalServerError, err.Error())
	}
}

// checkBatchUpdatePermissions checks user permissions for all repositories
func (h *Handler) checkBatchUpdatePermissions(ctx context.Context, repos []*models.Repository) error {
	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	return h.checkRepositoriesAccess(ctx, repoFullNames)
}

// executeBatchUpdate processes the batch update for all repositories
func (h *Handler) executeBatchUpdate(ctx context.Context, repos []*models.Repository, action string, reason string) BatchUpdateRepositoryStatusResponse {
	updatedIDs := make([]int64, 0, len(repos))
	failedIDs := make([]int64, 0)
	errors := make([]string, 0)
	initiatingUser := getInitiatingUser(ctx)

	for _, repo := range repos {
		if err := h.processBatchUpdate(ctx, repo, action, initiatingUser, reason); err != nil {
			failedIDs = append(failedIDs, repo.ID)
			errors = append(errors, fmt.Sprintf("%s: %s", repo.FullName, err.Error()))
			h.logger.Warn("Failed to update repository",
				"repo", repo.FullName,
				"action", action,
				"error", err)
			continue
		}
		updatedIDs = append(updatedIDs, repo.ID)
	}

	return BatchUpdateRepositoryStatusResponse{
		UpdatedIDs:   updatedIDs,
		FailedIDs:    failedIDs,
		UpdatedCount: len(updatedIDs),
		FailedCount:  len(failedIDs),
		Message:      buildBatchUpdateMessage(len(updatedIDs), len(failedIDs), len(repos)),
		Errors:       errors,
	}
}

// buildBatchUpdateMessage builds a human-readable message for the batch update result
func buildBatchUpdateMessage(updatedCount, failedCount, totalCount int) string {
	if failedCount == 0 {
		return fmt.Sprintf("Successfully updated %d repositories", updatedCount)
	} else if updatedCount == 0 {
		return fmt.Sprintf("Failed to update all %d repositories", failedCount)
	}
	return fmt.Sprintf("Updated %d of %d repositories (%d failed)", updatedCount, totalCount, failedCount)
}

// determineBatchUpdateStatusCode determines the HTTP status code based on results
func determineBatchUpdateStatusCode(response BatchUpdateRepositoryStatusResponse) int {
	if response.FailedCount > 0 && response.UpdatedCount == 0 {
		return http.StatusBadRequest
	} else if response.FailedCount > 0 {
		return http.StatusMultiStatus
	}
	return http.StatusOK
}

// processBatchUpdate processes a single repository update for batch operations
func (h *Handler) processBatchUpdate(ctx context.Context, repo *models.Repository, action string, initiatingUser *string, reason string) error {
	switch action {
	case "mark_migrated":
		return h.markRepositoryMigrated(ctx, repo, initiatingUser)
	case "mark_wont_migrate":
		return h.markRepositoryWontMigrateBatch(ctx, repo, false, initiatingUser)
	case "unmark_wont_migrate":
		return h.markRepositoryWontMigrateBatch(ctx, repo, true, initiatingUser)
	case "rollback":
		return h.rollbackRepositoryBatch(ctx, repo, reason, initiatingUser)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
}

// markRepositoryMigrated marks a repository as migrated (for external migrations)
func (h *Handler) markRepositoryMigrated(ctx context.Context, repo *models.Repository, initiatingUser *string) error {
	// Only allow marking from certain statuses
	allowedStatuses := map[string]bool{
		string(models.StatusPending):         true,
		string(models.StatusDryRunComplete):  true,
		string(models.StatusDryRunFailed):    true,
		string(models.StatusMigrationFailed): true,
		string(models.StatusRolledBack):      true,
	}

	if !allowedStatuses[repo.Status] {
		return fmt.Errorf("cannot mark repository with status '%s' as migrated", repo.Status)
	}

	// Update status to complete
	repo.Status = string(models.StatusComplete)
	now := time.Now()
	repo.MigratedAt = &now
	repo.UpdatedAt = now

	// Clear batch assignment (migrated outside the system)
	repo.BatchID = nil

	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	// Create migration history entry (shows in Migration History tab)
	message := "Repository marked as migrated (external migration)"
	if initiatingUser != nil {
		message = fmt.Sprintf("Repository marked as migrated by %s (external migration)", *initiatingUser)
	}

	history := &models.MigrationHistory{
		RepositoryID: repo.ID,
		Status:       "completed",
		Phase:        "migration",
		Message:      &message,
		StartedAt:    now,
		CompletedAt:  &now,
	}

	if _, err := h.db.CreateMigrationHistory(ctx, history); err != nil {
		h.logger.Warn("Failed to create migration history", "error", err)
	}

	// Create migration log entry (shows in Detailed Logs tab)
	logEntry := &models.MigrationLog{
		RepositoryID: repo.ID,
		Level:        "INFO",
		Phase:        "migration",
		Operation:    "mark_migrated",
		Message:      "Repository marked as migrated (external migration)",
		InitiatedBy:  initiatingUser,
	}
	if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
		h.logger.Warn("Failed to create migration log", "error", err)
	}

	return nil
}

// markRepositoryWontMigrateBatch marks/unmarks a repository as won't migrate (batch version)
func (h *Handler) markRepositoryWontMigrateBatch(ctx context.Context, repo *models.Repository, unmark bool, initiatingUser *string) error {
	var newStatus string
	var message string
	var operation string

	if unmark {
		// Unmark: change from wont_migrate back to pending
		if repo.Status != string(models.StatusWontMigrate) {
			return fmt.Errorf("repository is not marked as won't migrate")
		}
		newStatus = string(models.StatusPending)
		message = "Repository unmarked - changed to pending status"
		operation = "unmark_wont_migrate"
	} else {
		// Mark: only allow marking from certain statuses
		allowedStatuses := map[string]bool{
			string(models.StatusPending):         true,
			string(models.StatusDryRunComplete):  true,
			string(models.StatusDryRunFailed):    true,
			string(models.StatusMigrationFailed): true,
			string(models.StatusRolledBack):      true,
		}

		if !allowedStatuses[repo.Status] {
			return fmt.Errorf("cannot mark repository with status '%s' as won't migrate", repo.Status)
		}

		newStatus = string(models.StatusWontMigrate)
		message = "Repository marked as won't migrate"
		operation = "mark_wont_migrate"
	}

	// Remove from batch if assigned and marking (not unmarking)
	if repo.BatchID != nil && !unmark {
		repo.BatchID = nil
	}

	// Update status
	repo.Status = newStatus
	repo.UpdatedAt = time.Now()
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	// Create migration log entry
	logEntry := &models.MigrationLog{
		RepositoryID: repo.ID,
		Level:        "INFO",
		Phase:        "status_update",
		Operation:    operation,
		Message:      message,
		InitiatedBy:  initiatingUser,
	}
	if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
		h.logger.Warn("Failed to create migration log", "error", err)
	}

	return nil
}

// rollbackRepositoryBatch rolls back a repository (batch version)
func (h *Handler) rollbackRepositoryBatch(ctx context.Context, repo *models.Repository, reason string, initiatingUser *string) error {
	// Validate repository status is complete
	if repo.Status != string(models.StatusComplete) {
		return fmt.Errorf("only completed migrations can be rolled back (current status: %s)", repo.Status)
	}

	// Build reason message with user attribution
	reasonMessage := reason
	if reasonMessage == "" {
		reasonMessage = "Repository rolled back via batch operation"
	}
	if initiatingUser != nil {
		reasonMessage = fmt.Sprintf("%s (by %s)", reasonMessage, *initiatingUser)
	}

	// Perform rollback using existing database method (this creates migration history)
	if err := h.db.RollbackRepository(ctx, repo.FullName, reasonMessage); err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	// Create additional detailed log entry (always log for audit trail)
	logEntry := &models.MigrationLog{
		RepositoryID: repo.ID,
		Level:        "INFO",
		Phase:        "rollback",
		Operation:    "batch_rollback",
		Message:      fmt.Sprintf("Repository rolled back via batch operation. Reason: %s", reasonMessage),
		InitiatedBy:  initiatingUser,
	}
	if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
		h.logger.Warn("Failed to create migration log", "error", err)
	}

	return nil
}

// GetMigrationStatus handles GET /api/v1/migrations/{id}
func (h *Handler) GetMigrationStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid repository ID")
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepositoryByID(ctx, id)
	if err != nil {
		if h.handleContextError(ctx, err, "get repository by ID", r) {
			return
		}
		h.logger.Error("Failed to get repository", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch migration status")
		return
	}

	if repo == nil {
		h.sendError(w, http.StatusNotFound, "Migration not found")
		return
	}

	// Get latest history entry
	history, err := h.db.GetMigrationHistory(ctx, repo.ID)
	if err != nil {
		h.logger.Error("Failed to get migration history", "error", err)
		history = []*models.MigrationHistory{}
	}

	var latestEvent *models.MigrationHistory
	if len(history) > 0 {
		latestEvent = history[0]
	}

	response := map[string]interface{}{
		"repository_id":   repo.ID,
		"full_name":       repo.FullName,
		"status":          repo.Status,
		"destination_url": repo.DestinationURL,
		"migrated_at":     repo.MigratedAt,
		"latest_event":    latestEvent,
		"can_retry":       repo.Status == string(models.StatusMigrationFailed),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetMigrationHistory handles GET /api/v1/migrations/{id}/history
func (h *Handler) GetMigrationHistory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid repository ID")
		return
	}

	ctx := r.Context()
	history, err := h.db.GetMigrationHistory(ctx, id)
	if err != nil {
		if h.handleContextError(ctx, err, "get migration history", r) {
			return
		}
		h.logger.Error("Failed to get migration history", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch history")
		return
	}

	h.sendJSON(w, http.StatusOK, history)
}

// GetMigrationLogs handles GET /api/v1/migrations/{id}/logs
func (h *Handler) GetMigrationLogs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid repository ID")
		return
	}

	// Parse query parameters for filtering
	query := r.URL.Query()
	level := query.Get("level")
	phase := query.Get("phase")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	limit := 500 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := r.Context()
	logs, err := h.db.GetMigrationLogs(ctx, id, level, phase, limit, offset)
	if err != nil {
		if h.handleContextError(ctx, err, "get migration logs", r) {
			return
		}
		h.logger.Error("Failed to get migration logs", "error", err, "repo_id", id)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch logs")
		return
	}

	response := map[string]interface{}{
		"logs":   logs,
		"count":  len(logs),
		"limit":  limit,
		"offset": offset,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetAnalyticsSummary handles GET /api/v1/analytics/summary
//
//nolint:gocyclo // Complexity is inherent to analytics aggregation
func (h *Handler) GetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get filter parameters
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	// Get repository stats with filters
	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get repository stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch analytics")
		return
	}

	// Calculate totals (exclude wont_migrate from total count)
	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	// Explicitly categorize statuses to match organization table calculation
	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]

	failed := stats[string(models.StatusMigrationFailed)] +
		stats[string(models.StatusDryRunFailed)] +
		stats[string(models.StatusRolledBack)]

	// Pending includes initial state and dry run phase
	pending := stats[string(models.StatusPending)] +
		stats[string(models.StatusDryRunQueued)] +
		stats[string(models.StatusDryRunInProgress)] +
		stats[string(models.StatusDryRunComplete)]

	// In progress includes actual migration phases
	inProgress := stats[string(models.StatusPreMigration)] +
		stats[string(models.StatusArchiveGenerating)] +
		stats[string(models.StatusQueuedForMigration)] +
		stats[string(models.StatusMigratingContent)] +
		stats[string(models.StatusPostMigration)]

	// Calculate success rate
	successRate := 0.0
	if migrated+failed > 0 {
		successRate = float64(migrated) / float64(migrated+failed) * 100
	}

	// Get complexity distribution
	complexityDistribution, err := h.db.GetComplexityDistribution(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get complexity distribution", "error", err)
		complexityDistribution = []*storage.ComplexityDistribution{}
	}

	// Get migration velocity (last 30 days)
	migrationVelocity, err := h.db.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 30)
	if err != nil {
		h.logger.Error("Failed to get migration velocity", "error", err)
		migrationVelocity = &storage.MigrationVelocity{}
	}

	// Get migration time series
	migrationTimeSeries, err := h.db.GetMigrationTimeSeries(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get migration time series", "error", err)
		migrationTimeSeries = []*storage.MigrationTimeSeriesPoint{}
	}

	// Get average migration time
	avgMigrationTime, err := h.db.GetAverageMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get average migration time", "error", err)
		avgMigrationTime = 0
	}

	// Get median migration time
	medianMigrationTime, err := h.db.GetMedianMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get median migration time", "error", err)
		medianMigrationTime = 0
	}

	// Calculate estimated completion date
	var estimatedCompletionDate *string
	remaining := total - migrated
	if remaining > 0 && migrationVelocity.ReposPerDay > 0 {
		daysRemaining := float64(remaining) / migrationVelocity.ReposPerDay
		completionDate := time.Now().Add(time.Duration(daysRemaining*24) * time.Hour)
		dateStr := completionDate.Format("2006-01-02")
		estimatedCompletionDate = &dateStr
	}

	// Get discovery statistics with filters
	orgStats, err := h.db.GetOrganizationStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get organization stats", "error", err)
		orgStats = []*storage.OrganizationStats{}
	}

	// For Azure DevOps sources, also get project-level stats
	var projectStats []*storage.OrganizationStats
	if h.sourceType == sourceTypeAzureDevOps {
		projectStats, err = h.db.GetProjectStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
		if err != nil {
			h.logger.Error("Failed to get project stats", "error", err)
			projectStats = []*storage.OrganizationStats{}
		}
	}

	sizeDistribution, err := h.db.GetSizeDistributionFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get size distribution", "error", err)
		sizeDistribution = []*storage.SizeDistribution{}
	}

	featureStats, err := h.db.GetFeatureStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get feature stats", "error", err)
		featureStats = &storage.FeatureStats{}
	}

	// Get migration completion stats - use project-level for ADO, org-level for GitHub
	var migrationCompletionStats []*storage.MigrationCompletionStats
	if h.sourceType == sourceTypeAzureDevOps {
		migrationCompletionStats, err = h.db.GetMigrationCompletionStatsByProjectFiltered(ctx, orgFilter, projectFilter, batchFilter)
	} else {
		migrationCompletionStats, err = h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, projectFilter, batchFilter)
	}
	if err != nil {
		h.logger.Error("Failed to get migration completion stats", "error", err)
		migrationCompletionStats = []*storage.MigrationCompletionStats{}
	}

	summary := map[string]interface{}{
		"total_repositories":         total,
		"migrated_count":             migrated,
		"failed_count":               failed,
		"in_progress_count":          inProgress,
		"pending_count":              pending,
		"success_rate":               successRate,
		"status_breakdown":           stats,
		"complexity_distribution":    complexityDistribution,
		"migration_velocity":         migrationVelocity,
		"migration_time_series":      migrationTimeSeries,
		"average_migration_time":     avgMigrationTime,
		"median_migration_time":      medianMigrationTime,
		"estimated_completion_date":  estimatedCompletionDate,
		"organization_stats":         orgStats,
		"project_stats":              projectStats,
		"size_distribution":          sizeDistribution,
		"feature_stats":              featureStats,
		"migration_completion_stats": migrationCompletionStats,
	}

	h.sendJSON(w, http.StatusOK, summary)
}

// GetMigrationProgress handles GET /api/v1/analytics/progress
func (h *Handler) GetMigrationProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.db.GetRepositoryStatsByStatus(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get repository stats", r) {
			return
		}
		h.logger.Error("Failed to get repository stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch progress")
		return
	}

	// Calculate total (exclude wont_migrate)
	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"total":            total,
		"status_breakdown": stats,
	})
}

// GetExecutiveReport handles GET /api/v1/analytics/executive-report
func (h *Handler) GetExecutiveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get filter parameters
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	// Get basic analytics
	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get repository stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch analytics")
		return
	}

	// Calculate totals (exclude wont_migrate from total count)
	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	// Explicitly categorize statuses to match organization table calculation
	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]

	failed := stats[string(models.StatusMigrationFailed)] +
		stats[string(models.StatusDryRunFailed)] +
		stats[string(models.StatusRolledBack)]

	// Pending includes initial state and dry run phase
	pending := stats[string(models.StatusPending)] +
		stats[string(models.StatusDryRunQueued)] +
		stats[string(models.StatusDryRunInProgress)] +
		stats[string(models.StatusDryRunComplete)]

	// In progress includes actual migration phases
	inProgress := stats[string(models.StatusPreMigration)] +
		stats[string(models.StatusArchiveGenerating)] +
		stats[string(models.StatusQueuedForMigration)] +
		stats[string(models.StatusMigratingContent)] +
		stats[string(models.StatusPostMigration)]

	// Calculate success rate
	successRate := 0.0
	if migrated+failed > 0 {
		successRate = float64(migrated) / float64(migrated+failed) * 100
	}

	// Get migration velocity (last 30 days)
	migrationVelocity, err := h.db.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 30)
	if err != nil {
		h.logger.Error("Failed to get migration velocity", "error", err)
		migrationVelocity = &storage.MigrationVelocity{}
	}

	// Get migration time series for trend analysis
	migrationTimeSeries, err := h.db.GetMigrationTimeSeries(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get migration time series", "error", err)
		migrationTimeSeries = []*storage.MigrationTimeSeriesPoint{}
	}

	// Get average migration time
	avgMigrationTime, err := h.db.GetAverageMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get average migration time", "error", err)
		avgMigrationTime = 0
	}

	// Get median migration time
	medianMigrationTime, err := h.db.GetMedianMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get median migration time", "error", err)
		medianMigrationTime = 0
	}

	// Calculate estimated completion date
	var estimatedCompletionDate *string
	var daysRemaining int
	remaining := total - migrated
	if remaining > 0 && migrationVelocity.ReposPerDay > 0 {
		daysRemainingFloat := float64(remaining) / migrationVelocity.ReposPerDay
		daysRemaining = int(daysRemainingFloat)
		completionDate := time.Now().Add(time.Duration(daysRemainingFloat*24) * time.Hour)
		dateStr := completionDate.Format("2006-01-02")
		estimatedCompletionDate = &dateStr
	}

	// Get organization/project breakdowns
	var migrationCompletionStats []*storage.MigrationCompletionStats
	if h.sourceType == sourceTypeAzureDevOps {
		migrationCompletionStats, err = h.db.GetMigrationCompletionStatsByProjectFiltered(ctx, orgFilter, projectFilter, batchFilter)
	} else {
		migrationCompletionStats, err = h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, projectFilter, batchFilter)
	}
	if err != nil {
		h.logger.Error("Failed to get migration completion stats", "error", err)
		migrationCompletionStats = []*storage.MigrationCompletionStats{}
	}

	// Get complexity distribution
	complexityDistribution, err := h.db.GetComplexityDistribution(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get complexity distribution", "error", err)
		complexityDistribution = []*storage.ComplexityDistribution{}
	}

	// Get size distribution
	sizeDistribution, err := h.db.GetSizeDistributionFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get size distribution", "error", err)
		sizeDistribution = []*storage.SizeDistribution{}
	}

	// Get feature stats
	featureStats, err := h.db.GetFeatureStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get feature stats", "error", err)
		featureStats = &storage.FeatureStats{}
	}

	// Calculate risk metrics
	highComplexityPending := 0
	veryComplexCount := 0
	veryLargePending := 0
	for _, dist := range complexityDistribution {
		if dist.Category == categoryComplex || dist.Category == categoryVeryComplex {
			highComplexityPending += dist.Count
		}
		if dist.Category == categoryVeryComplex {
			veryComplexCount += dist.Count
		}
	}
	for _, dist := range sizeDistribution {
		if dist.Category == categorySizeVeryLarge {
			veryLargePending += dist.Count
		}
	}

	// Get batch statistics
	batches, err := h.db.ListBatches(ctx)
	if err != nil {
		h.logger.Error("Failed to get batches", "error", err)
		batches = []*models.Batch{}
	}

	completedBatches := 0
	inProgressBatches := 0
	pendingBatches := 0
	for _, batch := range batches {
		switch batch.Status {
		case "completed", "completed_with_errors":
			completedBatches++
		case statusInProgress:
			inProgressBatches++
		case statusPending, statusReady:
			pendingBatches++
		}
	}

	// Get first migration date for timeline
	var firstMigrationDate *string
	if len(migrationTimeSeries) > 0 {
		firstMigrationDate = &migrationTimeSeries[0].Date
	}

	// Calculate completion percentage
	completionRate := 0.0
	if total > 0 {
		completionRate = float64(migrated) / float64(total) * 100
	}

	// Build executive report with two main sections
	report := map[string]interface{}{
		// Source type identifier
		"source_type": h.sourceType,

		// Report metadata
		"report_metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"filters": map[string]interface{}{
				"organization": orgFilter,
				"project":      projectFilter,
				"batch_id":     batchFilter,
			},
		},

		// SECTION 1: DISCOVERY DATA
		// Repository characteristics discovered from source systems
		"discovery_data": map[string]interface{}{
			"overview": map[string]interface{}{
				"total_repositories": total,
				"source_type":        h.sourceType,
			},

			// Feature statistics from discovery
			"features": featureStats,

			// Repository complexity analysis
			"complexity": map[string]interface{}{
				"distribution":          complexityDistribution,
				"high_complexity_count": highComplexityPending,
				"very_complex_count":    veryComplexCount,
			},

			// Repository size analysis
			"size": map[string]interface{}{
				"distribution":     sizeDistribution,
				"very_large_count": veryLargePending,
			},

			// Organization/Project breakdown
			"organizational_breakdown": migrationCompletionStats,
		},

		// SECTION 2: MIGRATION PROGRESS & ANALYTICS
		// Migration execution status, velocity, and performance
		"migration_analytics": map[string]interface{}{
			// Overall migration status summary
			"summary": map[string]interface{}{
				"total_repositories":        total,
				"migrated_count":            migrated,
				"in_progress_count":         inProgress,
				"pending_count":             pending,
				"failed_count":              failed,
				"completion_percentage":     completionRate,
				"success_rate":              successRate,
				"estimated_completion_date": estimatedCompletionDate,
				"days_remaining":            daysRemaining,
				"first_migration_date":      firstMigrationDate,
			},

			// Detailed status breakdown by state
			"status_breakdown": stats,

			// Migration velocity and performance
			"velocity": map[string]interface{}{
				"repos_per_day":        migrationVelocity.ReposPerDay,
				"repos_per_week":       migrationVelocity.ReposPerWeek,
				"average_duration_sec": avgMigrationTime,
				"median_duration_sec":  medianMigrationTime,
				"trend":                migrationTimeSeries,
			},

			// Batch execution performance
			"batches": map[string]interface{}{
				"total":       len(batches),
				"completed":   completedBatches,
				"in_progress": inProgressBatches,
				"pending":     pendingBatches,
			},

			// Risk factors affecting migration
			"risk_factors": map[string]interface{}{
				"high_complexity_pending": highComplexityPending,
				"very_large_pending":      veryLargePending,
				"failed_migrations":       failed,
			},
		},
	}

	// Add ADO-specific discovery data if source is Azure DevOps
	if h.sourceType == sourceTypeAzureDevOps {
		// Enhance discovery data with ADO-specific risk factors
		if discoveryData, ok := report["discovery_data"].(map[string]interface{}); ok {
			discoveryData["ado_specific_risks"] = map[string]interface{}{
				"tfvc_repos":                   featureStats.ADOTFVCCount,
				"classic_pipelines":            featureStats.ADOHasClassicPipelines,
				"repos_with_active_work_items": featureStats.ADOHasWorkItems,
				"repos_with_wikis":             featureStats.ADOHasWiki,
				"repos_with_test_plans":        featureStats.ADOHasTestPlans,
				"repos_with_package_feeds":     featureStats.ADOHasPackageFeeds,
			}
		}
	}

	h.sendJSON(w, http.StatusOK, report)
}

// ExportExecutiveReport handles GET /api/v1/analytics/executive-report/export
func (h *Handler) ExportExecutiveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	format := r.URL.Query().Get("format")
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	if format != formatCSV && format != formatJSON {
		h.sendError(w, http.StatusBadRequest, "Invalid format. Must be 'csv' or 'json'")
		return
	}

	// Get all the same data as GetExecutiveReport
	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get repository stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch analytics")
		return
	}

	// Calculate totals (exclude wont_migrate from total count)
	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	// Explicitly categorize statuses to match organization table calculation
	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]

	failed := stats[string(models.StatusMigrationFailed)] +
		stats[string(models.StatusDryRunFailed)] +
		stats[string(models.StatusRolledBack)]

	// Pending includes initial state and dry run phase
	pending := stats[string(models.StatusPending)] +
		stats[string(models.StatusDryRunQueued)] +
		stats[string(models.StatusDryRunInProgress)] +
		stats[string(models.StatusDryRunComplete)]

	// In progress includes actual migration phases
	inProgress := stats[string(models.StatusPreMigration)] +
		stats[string(models.StatusArchiveGenerating)] +
		stats[string(models.StatusQueuedForMigration)] +
		stats[string(models.StatusMigratingContent)] +
		stats[string(models.StatusPostMigration)]

	successRate := 0.0
	if migrated+failed > 0 {
		successRate = float64(migrated) / float64(migrated+failed) * 100
	}

	migrationVelocity, err := h.db.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 30)
	if err != nil {
		migrationVelocity = &storage.MigrationVelocity{}
	}

	avgMigrationTime, err := h.db.GetAverageMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		avgMigrationTime = 0
	}

	medianMigrationTime, err := h.db.GetMedianMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		medianMigrationTime = 0
	}

	var estimatedCompletionDate string
	var daysRemaining int
	remaining := total - migrated
	if remaining > 0 && migrationVelocity.ReposPerDay > 0 {
		daysRemainingFloat := float64(remaining) / migrationVelocity.ReposPerDay
		daysRemaining = int(daysRemainingFloat)
		completionDate := time.Now().Add(time.Duration(daysRemainingFloat*24) * time.Hour)
		estimatedCompletionDate = completionDate.Format("2006-01-02")
	}

	// Get organization/project breakdowns
	var migrationCompletionStats []*storage.MigrationCompletionStats
	if h.sourceType == sourceTypeAzureDevOps {
		migrationCompletionStats, err = h.db.GetMigrationCompletionStatsByProjectFiltered(ctx, orgFilter, projectFilter, batchFilter)
	} else {
		migrationCompletionStats, err = h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, projectFilter, batchFilter)
	}
	if err != nil {
		migrationCompletionStats = []*storage.MigrationCompletionStats{}
	}

	complexityDistribution, err := h.db.GetComplexityDistribution(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		complexityDistribution = []*storage.ComplexityDistribution{}
	}

	sizeDistribution, err := h.db.GetSizeDistributionFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		sizeDistribution = []*storage.SizeDistribution{}
	}

	featureStats, err := h.db.GetFeatureStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		featureStats = &storage.FeatureStats{}
	}

	batches, err := h.db.ListBatches(ctx)
	if err != nil {
		batches = []*models.Batch{}
	}

	completedBatches := 0
	inProgressBatches := 0
	pendingBatches := 0
	for _, batch := range batches {
		switch batch.Status {
		case "completed", "completed_with_errors":
			completedBatches++
		case "in_progress":
			inProgressBatches++
		case "pending", "ready":
			pendingBatches++
		}
	}

	completionRate := 0.0
	if total > 0 {
		completionRate = float64(migrated) / float64(total) * 100
	}

	if format == formatCSV {
		h.exportExecutiveReportCSV(w, h.sourceType, total, migrated, inProgress, pending, failed, completionRate, successRate,
			estimatedCompletionDate, daysRemaining, migrationVelocity, int(avgMigrationTime), int(medianMigrationTime),
			migrationCompletionStats, complexityDistribution, sizeDistribution, featureStats,
			stats, completedBatches, inProgressBatches, pendingBatches)
	} else {
		h.exportExecutiveReportJSON(w, h.sourceType, total, migrated, inProgress, pending, failed, completionRate, successRate,
			estimatedCompletionDate, daysRemaining, migrationVelocity, int(avgMigrationTime), int(medianMigrationTime),
			migrationCompletionStats, complexityDistribution, sizeDistribution, featureStats,
			stats, completedBatches, inProgressBatches, pendingBatches)
	}
}

func (h *Handler) exportExecutiveReportCSV(w http.ResponseWriter, sourceType string, total, migrated, inProgress, pending, failed int,
	completionRate, successRate float64, estimatedCompletionDate string, daysRemaining int,
	velocity *storage.MigrationVelocity, avgMigrationTime, medianMigrationTime int,
	orgStats []*storage.MigrationCompletionStats, complexityDist []*storage.ComplexityDistribution,
	sizeDist []*storage.SizeDistribution, featureStats *storage.FeatureStats,
	statusBreakdown map[string]int, completedBatches, inProgressBatches, pendingBatches int) {

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=executive_migration_report.csv")

	var output strings.Builder

	// Report Header
	output.WriteString("EXECUTIVE MIGRATION REPORT\n")
	output.WriteString(fmt.Sprintf("Source Platform: %s\n", strings.ToUpper(sourceType)))
	output.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	output.WriteString("\n")

	// ========================================
	// SECTION 1: DISCOVERY DATA
	// ========================================
	output.WriteString("================================================================================\n")
	output.WriteString("SECTION 1: DISCOVERY DATA\n")
	output.WriteString("Repository characteristics discovered from source platform\n")
	output.WriteString("================================================================================\n")
	output.WriteString("\n")

	// 1.1 Discovery Overview
	output.WriteString("--- DISCOVERY OVERVIEW ---\n")
	output.WriteString("Metric,Value\n")
	output.WriteString(fmt.Sprintf("Total Repositories Discovered,%d\n", total))
	output.WriteString(fmt.Sprintf("Source Platform,%s\n", strings.ToUpper(sourceType)))
	output.WriteString("\n")

	// 1.2 Repository Complexity Analysis
	output.WriteString("--- REPOSITORY COMPLEXITY ---\n")
	output.WriteString("Complexity Category,Repository Count,Percentage\n")
	for _, dist := range complexityDist {
		pct := 0.0
		if total > 0 {
			pct = float64(dist.Count) / float64(total) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%.1f%%\n", escapesCSV(dist.Category), dist.Count, pct))
	}
	output.WriteString("\n")

	// 1.3 Repository Size Distribution
	output.WriteString("--- REPOSITORY SIZE DISTRIBUTION ---\n")
	output.WriteString("Size Category,Repository Count,Percentage\n")
	for _, dist := range sizeDist {
		pct := 0.0
		if total > 0 {
			pct = float64(dist.Count) / float64(total) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%.1f%%\n", escapesCSV(dist.Category), dist.Count, pct))
	}
	output.WriteString("\n")

	// 1.4 Feature Discovery
	output.WriteString("--- FEATURE DISCOVERY ---\n")
	output.WriteString("Feature,Repository Count,Percentage\n")
	totalRepos := featureStats.TotalRepositories
	if totalRepos > 0 {
		if sourceType == sourceTypeAzureDevOps {
			// ADO-specific features
			output.WriteString(fmt.Sprintf("TFVC Repositories,%d,%.1f%%\n", featureStats.ADOTFVCCount, float64(featureStats.ADOTFVCCount)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Azure Boards,%d,%.1f%%\n", featureStats.ADOHasBoards, float64(featureStats.ADOHasBoards)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Azure Pipelines,%d,%.1f%%\n", featureStats.ADOHasPipelines, float64(featureStats.ADOHasPipelines)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("YAML Pipelines,%d,%.1f%%\n", featureStats.ADOHasYAMLPipelines, float64(featureStats.ADOHasYAMLPipelines)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Classic Pipelines,%d,%.1f%%\n", featureStats.ADOHasClassicPipelines, float64(featureStats.ADOHasClassicPipelines)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Pull Requests,%d,%.1f%%\n", featureStats.ADOHasPullRequests, float64(featureStats.ADOHasPullRequests)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Work Items,%d,%.1f%%\n", featureStats.ADOHasWorkItems, float64(featureStats.ADOHasWorkItems)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Branch Policies,%d,%.1f%%\n", featureStats.ADOHasBranchPolicies, float64(featureStats.ADOHasBranchPolicies)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Wikis,%d,%.1f%%\n", featureStats.ADOHasWiki, float64(featureStats.ADOHasWiki)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Test Plans,%d,%.1f%%\n", featureStats.ADOHasTestPlans, float64(featureStats.ADOHasTestPlans)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Package Feeds,%d,%.1f%%\n", featureStats.ADOHasPackageFeeds, float64(featureStats.ADOHasPackageFeeds)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Service Hooks,%d,%.1f%%\n", featureStats.ADOHasServiceHooks, float64(featureStats.ADOHasServiceHooks)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("GHAS (Azure DevOps),%d,%.1f%%\n", featureStats.ADOHasGHAS, float64(featureStats.ADOHasGHAS)/float64(totalRepos)*100))
		} else {
			// GitHub-specific features
			output.WriteString(fmt.Sprintf("Archived,%d,%.1f%%\n", featureStats.IsArchived, float64(featureStats.IsArchived)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Forked Repositories,%d,%.1f%%\n", featureStats.IsFork, float64(featureStats.IsFork)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("GitHub Actions,%d,%.1f%%\n", featureStats.HasActions, float64(featureStats.HasActions)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Wikis,%d,%.1f%%\n", featureStats.HasWiki, float64(featureStats.HasWiki)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Pages,%d,%.1f%%\n", featureStats.HasPages, float64(featureStats.HasPages)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Discussions,%d,%.1f%%\n", featureStats.HasDiscussions, float64(featureStats.HasDiscussions)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Projects,%d,%.1f%%\n", featureStats.HasProjects, float64(featureStats.HasProjects)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Packages,%d,%.1f%%\n", featureStats.HasPackages, float64(featureStats.HasPackages)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Branch Protections,%d,%.1f%%\n", featureStats.HasBranchProtections, float64(featureStats.HasBranchProtections)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Rulesets,%d,%.1f%%\n", featureStats.HasRulesets, float64(featureStats.HasRulesets)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Code Scanning,%d,%.1f%%\n", featureStats.HasCodeScanning, float64(featureStats.HasCodeScanning)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Dependabot,%d,%.1f%%\n", featureStats.HasDependabot, float64(featureStats.HasDependabot)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Secret Scanning,%d,%.1f%%\n", featureStats.HasSecretScanning, float64(featureStats.HasSecretScanning)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("CODEOWNERS,%d,%.1f%%\n", featureStats.HasCodeowners, float64(featureStats.HasCodeowners)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Self-Hosted Runners,%d,%.1f%%\n", featureStats.HasSelfHostedRunners, float64(featureStats.HasSelfHostedRunners)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Release Assets,%d,%.1f%%\n", featureStats.HasReleaseAssets, float64(featureStats.HasReleaseAssets)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Webhooks,%d,%.1f%%\n", featureStats.HasWebhooks, float64(featureStats.HasWebhooks)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Environments,%d,%.1f%%\n", featureStats.HasEnvironments, float64(featureStats.HasEnvironments)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Secrets,%d,%.1f%%\n", featureStats.HasSecrets, float64(featureStats.HasSecrets)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Variables,%d,%.1f%%\n", featureStats.HasVariables, float64(featureStats.HasVariables)/float64(totalRepos)*100))
		}
		// Common features (applicable to both GitHub and Azure DevOps)
		output.WriteString(fmt.Sprintf("LFS,%d,%.1f%%\n", featureStats.HasLFS, float64(featureStats.HasLFS)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Submodules,%d,%.1f%%\n", featureStats.HasSubmodules, float64(featureStats.HasSubmodules)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Large Files,%d,%.1f%%\n", featureStats.HasLargeFiles, float64(featureStats.HasLargeFiles)/float64(totalRepos)*100))
	}
	output.WriteString("\n")

	// 1.5 Organizational Breakdown
	if sourceType == sourceTypeAzureDevOps {
		output.WriteString("--- PROJECT BREAKDOWN ---\n")
		output.WriteString("Project,Total Repositories\n")
	} else {
		output.WriteString("--- ORGANIZATION BREAKDOWN ---\n")
		output.WriteString("Organization,Total Repositories\n")
	}
	for _, org := range orgStats {
		output.WriteString(fmt.Sprintf("%s,%d\n", escapesCSV(org.Organization), org.TotalRepos))
	}
	output.WriteString("\n")

	// ========================================
	// SECTION 2: MIGRATION PROGRESS & ANALYTICS
	// ========================================
	output.WriteString("================================================================================\n")
	output.WriteString("SECTION 2: MIGRATION PROGRESS & ANALYTICS\n")
	output.WriteString("Migration execution status, velocity, and performance\n")
	output.WriteString("================================================================================\n")
	output.WriteString("\n")

	// 2.1 Migration Summary
	output.WriteString("--- MIGRATION SUMMARY ---\n")
	output.WriteString("Metric,Value\n")
	output.WriteString(fmt.Sprintf("Total Repositories,%d\n", total))
	output.WriteString(fmt.Sprintf("Completion Percentage,%.1f%%\n", completionRate))
	output.WriteString(fmt.Sprintf("Successfully Migrated,%d\n", migrated))
	output.WriteString(fmt.Sprintf("In Progress,%d\n", inProgress))
	output.WriteString(fmt.Sprintf("Pending,%d\n", pending))
	output.WriteString(fmt.Sprintf("Failed,%d\n", failed))
	output.WriteString(fmt.Sprintf("Success Rate,%.1f%%\n", successRate))
	if estimatedCompletionDate != "" {
		output.WriteString(fmt.Sprintf("Estimated Completion,%s\n", estimatedCompletionDate))
		output.WriteString(fmt.Sprintf("Days Remaining,%d\n", daysRemaining))
	}
	output.WriteString("\n")

	// 2.2 Migration Velocity
	output.WriteString("--- MIGRATION VELOCITY ---\n")
	output.WriteString("Metric,Value\n")
	output.WriteString(fmt.Sprintf("Repos Per Day,%.1f\n", velocity.ReposPerDay))
	output.WriteString(fmt.Sprintf("Repos Per Week,%.1f\n", velocity.ReposPerWeek))
	if avgMigrationTime > 0 {
		avgMinutes := avgMigrationTime / 60
		output.WriteString(fmt.Sprintf("Average Migration Time,%d minutes\n", avgMinutes))
	}
	if medianMigrationTime > 0 {
		medianMinutes := medianMigrationTime / 60
		output.WriteString(fmt.Sprintf("Median Migration Time,%d minutes\n", medianMinutes))
	}
	output.WriteString("\n")

	// 2.3 Organization/Project Migration Progress
	if sourceType == sourceTypeAzureDevOps {
		output.WriteString("--- PROJECT MIGRATION PROGRESS ---\n")
		output.WriteString("Project,Total,Completed,In Progress,Pending,Failed,Completion %\n")
	} else {
		output.WriteString("--- ORGANIZATION MIGRATION PROGRESS ---\n")
		output.WriteString("Organization,Total,Completed,In Progress,Pending,Failed,Completion %\n")
	}
	for _, org := range orgStats {
		completionPct := 0.0
		if org.TotalRepos > 0 {
			completionPct = float64(org.CompletedCount) / float64(org.TotalRepos) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%d,%d,%d,%d,%.1f%%\n",
			escapesCSV(org.Organization),
			org.TotalRepos,
			org.CompletedCount,
			org.InProgressCount,
			org.PendingCount,
			org.FailedCount,
			completionPct))
	}
	output.WriteString("\n")

	// ADO Risk Analysis
	if sourceType == sourceTypeAzureDevOps {
		output.WriteString("=== AZURE DEVOPS MIGRATION RISKS ===\n")
		output.WriteString("Risk Factor,Repository Count,Percentage\n")
		if totalRepos > 0 {
			output.WriteString(fmt.Sprintf("TFVC Repositories (Requires Conversion),%d,%.1f%%\n", featureStats.ADOTFVCCount, float64(featureStats.ADOTFVCCount)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Classic Pipelines (Manual Recreation),%d,%.1f%%\n", featureStats.ADOHasClassicPipelines, float64(featureStats.ADOHasClassicPipelines)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Active Work Items (Won't Migrate),%d,%.1f%%\n", featureStats.ADOHasWorkItems, float64(featureStats.ADOHasWorkItems)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Wikis (Manual Migration),%d,%.1f%%\n", featureStats.ADOHasWiki, float64(featureStats.ADOHasWiki)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Test Plans (No GitHub Equivalent),%d,%.1f%%\n", featureStats.ADOHasTestPlans, float64(featureStats.ADOHasTestPlans)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Package Feeds (Separate Migration),%d,%.1f%%\n", featureStats.ADOHasPackageFeeds, float64(featureStats.ADOHasPackageFeeds)/float64(totalRepos)*100))
		}
		output.WriteString("\n")
	}

	// 2.4 Batch Execution Performance
	output.WriteString("--- BATCH EXECUTION PERFORMANCE ---\n")
	output.WriteString("Status,Count\n")
	output.WriteString(fmt.Sprintf("Completed,%d\n", completedBatches))
	output.WriteString(fmt.Sprintf("In Progress,%d\n", inProgressBatches))
	output.WriteString(fmt.Sprintf("Pending,%d\n", pendingBatches))
	output.WriteString(fmt.Sprintf("Total Batches,%d\n", completedBatches+inProgressBatches+pendingBatches))
	output.WriteString("\n")

	// 2.5 Risk Factors
	output.WriteString("--- MIGRATION RISK FACTORS ---\n")
	output.WriteString("Risk Factor,Count\n")
	highComplexity := 0
	for _, dist := range complexityDist {
		if dist.Category == categoryComplex || dist.Category == categoryVeryComplex {
			highComplexity += dist.Count
		}
	}
	veryLarge := 0
	for _, dist := range sizeDist {
		if dist.Category == categorySizeVeryLarge {
			veryLarge += dist.Count
		}
	}
	output.WriteString(fmt.Sprintf("High Complexity Repositories Pending,%d\n", highComplexity))
	output.WriteString(fmt.Sprintf("Very Large Repositories Pending,%d\n", veryLarge))
	output.WriteString(fmt.Sprintf("Failed Migrations,%d\n", failed))
	output.WriteString("\n")

	// 2.6 Detailed Status Breakdown
	output.WriteString("--- DETAILED STATUS BREAKDOWN ---\n")
	output.WriteString("Status,Repository Count,Percentage\n")
	for status, count := range statusBreakdown {
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%.1f%%\n", escapesCSV(status), count, pct))
	}

	if _, err := w.Write([]byte(output.String())); err != nil {
		h.logger.Error("Failed to write CSV response", "error", err)
	}
}

func (h *Handler) exportExecutiveReportJSON(w http.ResponseWriter, sourceType string, total, migrated, inProgress, pending, failed int,
	completionRate, successRate float64, estimatedCompletionDate string, daysRemaining int,
	velocity *storage.MigrationVelocity, avgMigrationTime, medianMigrationTime int,
	orgStats []*storage.MigrationCompletionStats, complexityDist []*storage.ComplexityDistribution,
	sizeDist []*storage.SizeDistribution, featureStats *storage.FeatureStats,
	statusBreakdown map[string]int, completedBatches, inProgressBatches, pendingBatches int) {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=executive_migration_report.json")

	// Calculate risk metrics
	highComplexity := 0
	for _, dist := range complexityDist {
		if dist.Category == categoryComplex || dist.Category == categoryVeryComplex {
			highComplexity += dist.Count
		}
	}
	veryLarge := 0
	for _, dist := range sizeDist {
		if dist.Category == categorySizeVeryLarge {
			veryLarge += dist.Count
		}
	}

	report := map[string]interface{}{
		"source_type": sourceType,
		"report_metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"report_type":  "Executive Migration Report",
			"version":      "2.0",
		},

		// SECTION 1: DISCOVERY DATA
		"discovery_data": map[string]interface{}{
			"overview": map[string]interface{}{
				"total_repositories": total,
				"source_type":        sourceType,
			},
			"features":                 featureStats,
			"complexity_distribution":  complexityDist,
			"size_distribution":        sizeDist,
			"organizational_breakdown": orgStats,
		},

		// SECTION 2: MIGRATION PROGRESS & ANALYTICS
		"migration_analytics": map[string]interface{}{
			"summary": map[string]interface{}{
				"total_repositories":        total,
				"migrated_count":            migrated,
				"in_progress_count":         inProgress,
				"pending_count":             pending,
				"failed_count":              failed,
				"completion_percentage":     completionRate,
				"success_rate":              successRate,
				"estimated_completion_date": estimatedCompletionDate,
				"days_remaining":            daysRemaining,
			},
			"status_breakdown": statusBreakdown,
			"velocity": map[string]interface{}{
				"repos_per_day":        velocity.ReposPerDay,
				"repos_per_week":       velocity.ReposPerWeek,
				"average_duration_sec": avgMigrationTime,
				"median_duration_sec":  medianMigrationTime,
			},
			"batches": map[string]interface{}{
				"total":       completedBatches + inProgressBatches + pendingBatches,
				"completed":   completedBatches,
				"in_progress": inProgressBatches,
				"pending":     pendingBatches,
			},
			"risk_factors": map[string]interface{}{
				"high_complexity_pending": highComplexity,
				"very_large_pending":      veryLarge,
				"failed_migrations":       failed,
			},
			"organization_progress": orgStats,
		},
	}

	// Add ADO-specific discovery data if source is Azure DevOps
	if sourceType == sourceTypeAzureDevOps {
		if discoveryData, ok := report["discovery_data"].(map[string]interface{}); ok {
			discoveryData["ado_specific_risks"] = map[string]interface{}{
				"tfvc_repos":                   featureStats.ADOTFVCCount,
				"classic_pipelines":            featureStats.ADOHasClassicPipelines,
				"repos_with_active_work_items": featureStats.ADOHasWorkItems,
				"repos_with_wikis":             featureStats.ADOHasWiki,
				"repos_with_test_plans":        featureStats.ADOHasTestPlans,
				"repos_with_package_feeds":     featureStats.ADOHasPackageFeeds,
			}
		}
	}

	if err := json.NewEncoder(w).Encode(report); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// ExportDetailedDiscoveryReport handles GET /api/v1/analytics/detailed-discovery-report/export
func (h *Handler) ExportDetailedDiscoveryReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	format := r.URL.Query().Get("format")
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	if format != formatCSV && format != formatJSON {
		h.sendError(w, http.StatusBadRequest, "Invalid format. Must be 'csv' or 'json'")
		return
	}

	// Build filters and get repositories
	filters := buildDiscoveryReportFilters(orgFilter, projectFilter, batchFilter)
	repos, err := h.db.ListRepositories(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to list repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

	// Check permissions
	if err := h.checkDiscoveryReportAccess(ctx, repos); err != nil {
		h.logger.Warn("Detailed discovery report access denied", "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	// Get local dependencies count for each repository
	localDepsCount := h.getLocalDependenciesCount(ctx, repos)

	// Get batch names for repositories
	batchNames := h.getBatchNames(ctx, repos)

	if format == formatCSV {
		h.exportDetailedDiscoveryReportCSV(w, repos, localDepsCount, batchNames)
	} else {
		h.exportDetailedDiscoveryReportJSON(w, repos, localDepsCount, batchNames, orgFilter, projectFilter, batchFilter)
	}
}

// buildDiscoveryReportFilters constructs filters from query parameters
func buildDiscoveryReportFilters(orgFilter, projectFilter, batchFilter string) map[string]interface{} {
	filters := make(map[string]interface{})
	if orgFilter != "" {
		filters["organization"] = orgFilter
	}
	if projectFilter != "" {
		filters["project"] = projectFilter
	}
	if batchFilter != "" {
		batchID, err := strconv.ParseInt(batchFilter, 10, 64)
		if err == nil {
			filters["batch_id"] = batchID
		}
	}
	return filters
}

// checkDiscoveryReportAccess validates user access to repositories
func (h *Handler) checkDiscoveryReportAccess(ctx context.Context, repos []*models.Repository) error {
	if !h.authConfig.Enabled {
		return nil
	}
	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	return h.checkRepositoriesAccess(ctx, repoFullNames)
}

// getLocalDependenciesCount calculates local dependencies for each repository
func (h *Handler) getLocalDependenciesCount(ctx context.Context, repos []*models.Repository) map[int64]int {
	localDepsCount := make(map[int64]int)
	for _, repo := range repos {
		deps, err := h.db.GetRepositoryDependencies(ctx, repo.ID)
		if err == nil {
			count := 0
			for _, dep := range deps {
				if dep.IsLocal {
					count++
				}
			}
			localDepsCount[repo.ID] = count
		}
	}
	return localDepsCount
}

// getBatchNames retrieves batch names for repositories
func (h *Handler) getBatchNames(ctx context.Context, repos []*models.Repository) map[int64]string {
	batchNames := make(map[int64]string)
	uniqueBatchIDs := make(map[int64]bool)

	// Collect unique batch IDs
	for _, repo := range repos {
		if repo.BatchID != nil {
			uniqueBatchIDs[*repo.BatchID] = true
		}
	}

	// Fetch batch details for each unique ID
	for batchID := range uniqueBatchIDs {
		batch, err := h.db.GetBatch(ctx, batchID)
		if err == nil && batch != nil {
			batchNames[batchID] = batch.Name
		}
	}

	return batchNames
}

// titleCase capitalizes the first letter of each word
func titleCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// formatStatusForDisplay converts internal status to human-readable format
func formatStatusForDisplay(status string) string {
	// Replace underscores with spaces and title-case each word
	status = strings.ReplaceAll(status, "_", " ")
	return titleCase(status)
}

// formatSourceForDisplay converts internal source type to human-readable format
func formatSourceForDisplay(source string) string {
	switch source {
	case "github":
		return "GitHub"
	case "azuredevops":
		return "Azure DevOps"
	case "gitlab":
		return "GitLab"
	case "ghes":
		return "GitHub Enterprise Server"
	default:
		return titleCase(source)
	}
}

// formatVisibilityForDisplay capitalizes visibility
func formatVisibilityForDisplay(visibility string) string {
	if visibility == "" {
		return ""
	}
	return titleCase(visibility)
}

func (h *Handler) exportDetailedDiscoveryReportJSON(w http.ResponseWriter, repos []*models.Repository, localDepsCount map[int64]int, batchNames map[int64]string, orgFilter, projectFilter, batchFilter string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=detailed_discovery_report.json")

	// Build filter information
	filtersApplied := make(map[string]string)
	if orgFilter != "" {
		filtersApplied["organization"] = orgFilter
	}
	if projectFilter != "" {
		filtersApplied["project"] = projectFilter
	}
	if batchFilter != "" {
		filtersApplied["batch_id"] = batchFilter
	}

	// Create repository data with local dependencies count added
	repoData := make([]map[string]interface{}, 0, len(repos))
	for _, repo := range repos {
		// Start with the repository as JSON
		repoJSON, err := json.Marshal(repo)
		if err != nil {
			h.logger.Warn("Failed to marshal repository", "repo", repo.FullName, "error", err)
			continue
		}

		var repoMap map[string]interface{}
		if err := json.Unmarshal(repoJSON, &repoMap); err != nil {
			h.logger.Warn("Failed to unmarshal repository JSON", "repo", repo.FullName, "error", err)
			continue
		}

		// Add local dependencies count
		if count, exists := localDepsCount[repo.ID]; exists {
			repoMap["local_dependencies_count"] = count
		} else {
			repoMap["local_dependencies_count"] = 0
		}

		// Add computed organization field for easier filtering
		repoMap["organization"] = repo.Organization()

		repoData = append(repoData, repoMap)
	}

	report := map[string]interface{}{
		"report_metadata": map[string]interface{}{
			"generated_at":       time.Now().Format(time.RFC3339),
			"report_type":        "Detailed Repository Discovery Report",
			"source_type":        h.sourceType,
			"version":            "1.0",
			"filters_applied":    filtersApplied,
			"total_repositories": len(repos),
		},
		"repositories": repoData,
	}

	if err := json.NewEncoder(w).Encode(report); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *Handler) exportDetailedDiscoveryReportCSV(w http.ResponseWriter, repos []*models.Repository, localDepsCount map[int64]int, batchNames map[int64]string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=detailed_discovery_report.csv")

	var output strings.Builder

	// Write report header
	h.writeCSVReportHeader(&output, len(repos))

	// Write CSV column headers
	h.writeCSVColumnHeaders(&output)

	// Write data rows
	for _, repo := range repos {
		h.writeCSVRepoRow(&output, repo, localDepsCount, batchNames)
	}

	if _, err := w.Write([]byte(output.String())); err != nil {
		h.logger.Error("Failed to write CSV response", "error", err)
	}
}

func (h *Handler) writeCSVReportHeader(output *strings.Builder, repoCount int) {
	sourceDisplay := formatSourceForDisplay(h.sourceType)
	output.WriteString("DETAILED REPOSITORY DISCOVERY REPORT\n")
	output.WriteString(fmt.Sprintf("Source: %s\n", sourceDisplay))
	output.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("Total Repositories: %d\n", repoCount))
	output.WriteString("\n")
}

func (h *Handler) writeCSVColumnHeaders(output *strings.Builder) {
	if h.sourceType == sourceTypeAzureDevOps {
		output.WriteString("Repository,Organization,Project,Source,Status,Batch,")
	} else {
		output.WriteString("Repository,Organization,Source,Status,Batch,")
	}
	output.WriteString("Size (Bytes),Size (Human),Commit Count,Commits (Last 12 Weeks),")
	output.WriteString("Has LFS,Has Submodules,Has Large Files,Large File Count,Largest File Size (Bytes),")
	output.WriteString("Has Blocking Files,Local Dependencies,Complexity Score,")
	output.WriteString("Default Branch,Branch Count,Last Commit Date,Visibility,Is Archived,Is Fork,")

	if h.sourceType == sourceTypeAzureDevOps {
		output.WriteString("Is Git,Pipeline Count,YAML Pipelines,Classic Pipelines,Has Boards,Has Wiki,")
		output.WriteString("Pull Requests,Work Items,Branch Policies,Test Plans,Package Feeds,Service Hooks")
	} else {
		output.WriteString("Workflow Count,Environment Count,Secret Count,Has Actions,Has Environments,Has Packages,")
		output.WriteString("Has Projects,Branch Protections,Has Rulesets,Contributor Count,")
		output.WriteString("Issue Count,Pull Request Count,Has Self-Hosted Runners")
	}
	output.WriteString("\n")
}

func (h *Handler) writeCSVRepoRow(output *strings.Builder, repo *models.Repository, localDepsCount map[int64]int, batchNames map[int64]string) {
	// Common fields
	output.WriteString(escapesCSV(repo.FullName))
	output.WriteString(",")
	output.WriteString(escapesCSV(repo.Organization()))
	output.WriteString(",")

	// ADO project field
	if h.sourceType == sourceTypeAzureDevOps {
		if repo.ADOProject != nil {
			output.WriteString(escapesCSV(*repo.ADOProject))
		}
		output.WriteString(",")
	}

	// Format source and status for human readability
	output.WriteString(escapesCSV(formatSourceForDisplay(repo.Source)))
	output.WriteString(",")
	output.WriteString(escapesCSV(formatStatusForDisplay(repo.Status)))
	output.WriteString(",")

	// Batch name (instead of just ID)
	if repo.BatchID != nil {
		if batchName, exists := batchNames[*repo.BatchID]; exists {
			output.WriteString(escapesCSV(batchName))
		} else {
			output.WriteString(fmt.Sprintf("Batch %d", *repo.BatchID))
		}
	}
	output.WriteString(",")

	// Size fields
	if repo.TotalSize != nil {
		output.WriteString(fmt.Sprintf("%d,%s,", *repo.TotalSize, escapesCSV(formatBytes(*repo.TotalSize))))
	} else {
		output.WriteString("0,0 B,")
	}

	// Commit, file, and complexity data
	output.WriteString(fmt.Sprintf("%d,%d,", repo.CommitCount, repo.CommitsLast12Weeks))
	output.WriteString(fmt.Sprintf("%s,%s,%s,%d,", formatBool(repo.HasLFS), formatBool(repo.HasSubmodules), formatBool(repo.HasLargeFiles), repo.LargeFileCount))

	if repo.LargestFileSize != nil {
		output.WriteString(fmt.Sprintf("%d,", *repo.LargestFileSize))
	} else {
		output.WriteString("0,")
	}

	output.WriteString(formatBool(repo.HasBlockingFiles))
	output.WriteString(",")

	// Local dependencies
	if count, exists := localDepsCount[repo.ID]; exists {
		output.WriteString(fmt.Sprintf("%d,", count))
	} else {
		output.WriteString("0,")
	}

	// Complexity score
	if repo.ComplexityScore != nil {
		output.WriteString(fmt.Sprintf("%d,", *repo.ComplexityScore))
	} else {
		output.WriteString(",")
	}

	// Git metadata
	if repo.DefaultBranch != nil {
		output.WriteString(escapesCSV(*repo.DefaultBranch))
	}
	output.WriteString(",")
	output.WriteString(fmt.Sprintf("%d,", repo.BranchCount))

	if repo.LastCommitDate != nil {
		output.WriteString(repo.LastCommitDate.Format("2006-01-02"))
	}
	output.WriteString(",")

	// Repository properties (formatted for readability)
	output.WriteString(fmt.Sprintf("%s,%s,%s,", escapesCSV(formatVisibilityForDisplay(repo.Visibility)), formatBool(repo.IsArchived), formatBool(repo.IsFork)))

	// Source-specific fields
	h.writeCSVSourceSpecificFields(output, repo)
	output.WriteString("\n")
}

func (h *Handler) writeCSVSourceSpecificFields(output *strings.Builder, repo *models.Repository) {
	if h.sourceType == sourceTypeAzureDevOps {
		output.WriteString(fmt.Sprintf("%s,%d,%d,%d,%s,%s,",
			formatBool(repo.ADOIsGit),
			repo.ADOPipelineCount,
			repo.ADOYAMLPipelineCount,
			repo.ADOClassicPipelineCount,
			formatBool(repo.ADOHasBoards),
			formatBool(repo.ADOHasWiki)))
		output.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d",
			repo.ADOPullRequestCount,
			repo.ADOWorkItemCount,
			repo.ADOBranchPolicyCount,
			repo.ADOTestPlanCount,
			repo.ADOPackageFeedCount,
			repo.ADOServiceHookCount))
	} else {
		output.WriteString(fmt.Sprintf("%d,%d,%d,%s,%s,%s,%s,%d,%s,",
			repo.WorkflowCount,
			repo.EnvironmentCount,
			repo.SecretCount,
			formatBool(repo.HasActions),
			formatBool(repo.EnvironmentCount > 0),
			formatBool(repo.HasPackages),
			formatBool(repo.HasProjects),
			repo.BranchProtections,
			formatBool(repo.HasRulesets)))
		output.WriteString(fmt.Sprintf("%d,%d,%d,%s",
			repo.ContributorCount,
			repo.IssueCount,
			repo.PullRequestCount,
			formatBool(repo.HasSelfHostedRunners)))
	}
}

// Helper methods

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, map[string]string{"error": message})
}

// handleContextError checks if an error is due to request cancellation and logs appropriately
// Returns true if the error is a context cancellation (caller should return early)
func (h *Handler) handleContextError(ctx context.Context, err error, operation string, r *http.Request) bool {
	if ctx.Err() == context.Canceled {
		h.logger.Debug("Request canceled by client",
			"operation", operation,
			"path", r.URL.Path,
			"method", r.Method)
		return true
	}
	if ctx.Err() == context.DeadlineExceeded {
		h.logger.Warn("Request timeout",
			"operation", operation,
			"path", r.URL.Path,
			"method", r.Method,
			"error", err)
		return true
	}
	return false
}

func canMigrate(status string) bool {
	// Cannot migrate repositories marked as wont_migrate
	if status == string(models.StatusWontMigrate) {
		return false
	}

	allowedStatuses := []string{
		string(models.StatusPending),         // Can queue pending repositories for migration
		string(models.StatusDryRunQueued),    // Allow re-queuing dry runs
		string(models.StatusDryRunFailed),    // Allow retrying failed dry runs
		string(models.StatusDryRunComplete),  // Can queue after successful dry run
		string(models.StatusMigrationFailed), // Allow retrying failed migrations
		string(models.StatusRolledBack),      // Allow re-migrating rolled back repositories
	}

	for _, allowed := range allowedStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

func isEligibleForBatch(status string) bool {
	eligibleStatuses := []string{
		string(models.StatusPending),
		string(models.StatusDryRunComplete),
		string(models.StatusDryRunFailed),
		string(models.StatusMigrationFailed),
		string(models.StatusRolledBack),
	}

	for _, eligible := range eligibleStatuses {
		if status == eligible {
			return true
		}
	}
	return false
}

// getInitiatingUser extracts the authenticated username from the context
// Returns nil if auth is disabled or user not found
func getInitiatingUser(ctx context.Context) *string {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil
	}
	username := user.Login
	return &username
}

func isRepositoryEligibleForBatch(repo *models.Repository) (bool, string) {
	// Check if already in a batch
	if repo.BatchID != nil {
		return false, "repository is already assigned to a batch"
	}

	// Check if repository exceeds GitHub's 40 GiB size limit
	if repo.HasOversizedRepository {
		return false, "repository exceeds GitHub's 40 GiB size limit and requires remediation"
	}

	// Check if status is eligible
	if !isEligibleForBatch(repo.Status) {
		return false, fmt.Sprintf("repository status '%s' is not eligible for batch assignment", repo.Status)
	}

	return true, ""
}

// checkRepositoryAccess validates that the user has access to a specific repository
// Returns an error if auth is enabled and user doesn't have access
func (h *Handler) checkRepositoryAccess(ctx context.Context, repoFullName string) error {
	// If auth is not enabled, allow access
	if !h.authConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if h.sourceDualClient == nil {
		h.logger.Warn("Cannot check repository access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := h.sourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, h.authConfig, h.logger, h.sourceBaseURL)

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

// checkRepositoriesAccess validates that the user has access to all specified repositories
// Returns an error if auth is enabled and user doesn't have access to any repository
func (h *Handler) checkRepositoriesAccess(ctx context.Context, repoFullNames []string) error {
	// If auth is not enabled, allow access
	if !h.authConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if h.sourceDualClient == nil {
		h.logger.Warn("Cannot check repositories access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := h.sourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, h.authConfig, h.logger, h.sourceBaseURL)

	// Validate access to all repositories
	return checker.ValidateRepositoryAccess(ctx, user, token, repoFullNames)
}

// adoProjectStats holds the computed statistics for an ADO project
type adoProjectStats struct {
	statusCounts                map[string]int
	migratedCount               int
	inProgressCount             int
	failedCount                 int
	pendingCount                int
	migrationProgressPercentage int
}

// getADOProjectStats queries and calculates status distribution and progress metrics for an ADO project
//
//nolint:dupl // Intentionally extracted to avoid duplication in ListOrganizations and ListProjects
func (h *Handler) getADOProjectStats(ctx context.Context, projectName, organization string, repoCount int) adoProjectStats {
	stats := adoProjectStats{
		statusCounts: make(map[string]int),
	}

	if repoCount == 0 {
		return stats
	}

	// Query actual status distribution using SQL for efficiency
	// Filter by both ado_project AND organization (via full_name prefix) to handle
	// duplicate project names across different ADO organizations
	var results []struct {
		Status string
		Count  int
	}
	err := h.db.DB().WithContext(ctx).
		Raw(`
			SELECT status, COUNT(*) as count
			FROM repositories
			WHERE ado_project = ?
			AND full_name LIKE ?
			AND status != 'wont_migrate'
			GROUP BY status
		`, projectName, organization+"/%").
		Scan(&results).Error

	if err != nil {
		h.logger.Warn("Failed to get status counts for project", "project", projectName, "org", organization, "error", err)
		// Fallback: assume all pending
		stats.statusCounts["pending"] = repoCount
		stats.pendingCount = repoCount
	} else {
		for _, result := range results {
			stats.statusCounts[result.Status] = result.Count

			// Calculate progress metrics
			switch result.Status {
			case "complete", "migration_complete":
				stats.migratedCount += result.Count
			case "migration_failed", "dry_run_failed", "rolled_back":
				stats.failedCount += result.Count
			case "queued_for_migration", "migrating_content", "dry_run_in_progress",
				"dry_run_queued", "pre_migration", "archive_generating", "post_migration":
				stats.inProgressCount += result.Count
			default:
				// pending, dry_run_complete, remediation_required
				stats.pendingCount += result.Count
			}
		}
	}

	// Calculate migration progress percentage
	if repoCount > 0 {
		stats.migrationProgressPercentage = (stats.migratedCount * 100) / repoCount
	}

	return stats
}

// ListTeams handles GET /api/v1/teams
// Returns GitHub teams with optional organization filter
// Teams are only available for GitHub sources, not Azure DevOps
func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Teams are only available for GitHub sources
	if h.sourceType == sourceTypeAzureDevOps {
		h.sendJSON(w, http.StatusOK, []interface{}{})
		return
	}

	// Get optional organization filter
	orgFilter := r.URL.Query().Get("organization")

	teams, err := h.db.ListTeams(ctx, orgFilter)
	if err != nil {
		if h.handleContextError(ctx, err, "list teams", r) {
			return
		}
		h.logger.Error("Failed to list teams", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch teams")
		return
	}

	// Convert to response format with full_slug for unique identification
	type TeamResponse struct {
		ID           int64   `json:"id"`
		Organization string  `json:"organization"`
		Slug         string  `json:"slug"`
		Name         string  `json:"name"`
		Description  *string `json:"description,omitempty"`
		Privacy      string  `json:"privacy"`
		FullSlug     string  `json:"full_slug"` // "org/team-slug" format
	}

	response := make([]TeamResponse, len(teams))
	for i, team := range teams {
		response[i] = TeamResponse{
			ID:           team.ID,
			Organization: team.Organization,
			Slug:         team.Slug,
			Name:         team.Name,
			Description:  team.Description,
			Privacy:      team.Privacy,
			FullSlug:     team.FullSlug(),
		}
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ListOrganizations handles GET /api/v1/organizations
// Returns GitHub organizations or ADO projects depending on source type
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// For Azure DevOps sources, return project-level data (not aggregated orgs)
	// The frontend will group projects by their ado_organization field
	if h.sourceType == sourceTypeAzureDevOps {
		// Get all ADO projects
		projects, err := h.db.GetADOProjects(ctx, "")
		if err != nil {
			if h.handleContextError(ctx, err, "get ADO projects", r) {
				return
			}
			h.logger.Error("Failed to get ADO projects", "error", err)
			h.sendError(w, http.StatusInternalServerError, "Failed to fetch projects")
			return
		}

		// Build project stats with repository counts and status distribution
		projectStats := make([]interface{}, 0, len(projects))
		for _, project := range projects {
			// Count repositories for this project
			repoCount, err := h.db.CountRepositoriesByADOProject(ctx, project.Organization, project.Name)
			if err != nil {
				h.logger.Warn("Failed to count repositories for project", "project", project.Name, "error", err)
				repoCount = 0
			}

			// Get status distribution and progress metrics for this project
			stats := h.getADOProjectStats(ctx, project.Name, project.Organization, repoCount)

			projectStats = append(projectStats, map[string]interface{}{
				"organization":                  project.Name,         // Project name (frontend expects this field)
				"ado_organization":              project.Organization, // Parent ADO organization name
				"total_repos":                   repoCount,
				"status_counts":                 stats.statusCounts,
				"migrated_count":                stats.migratedCount,
				"in_progress_count":             stats.inProgressCount,
				"failed_count":                  stats.failedCount,
				"pending_count":                 stats.pendingCount,
				"migration_progress_percentage": stats.migrationProgressPercentage,
			})
		}

		h.sendJSON(w, http.StatusOK, projectStats)
		return
	}

	// For GitHub sources, use standard organization stats
	orgStats, err := h.db.GetOrganizationStats(ctx)
	if err != nil {
		// Check if request was canceled by client (e.g., navigating away)
		if h.handleContextError(ctx, err, "get organization stats", r) {
			return
		}
		h.logger.Error("Failed to get organization stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch organizations")
		return
	}

	h.sendJSON(w, http.StatusOK, orgStats)
}

// ListProjects handles GET /api/v1/projects
// Returns ADO projects with repository counts and status breakdown
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Only applicable for Azure DevOps sources
	if h.sourceType != sourceTypeAzureDevOps {
		h.sendJSON(w, http.StatusOK, []interface{}{})
		return
	}

	// Get all ADO projects
	projects, err := h.db.GetADOProjects(ctx, "")
	if err != nil {
		if h.handleContextError(ctx, err, "get ADO projects", r) {
			return
		}
		h.logger.Error("Failed to get ADO projects", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch projects")
		return
	}

	// Build project stats with repository counts and status distribution
	projectStats := make([]interface{}, 0, len(projects))
	for _, project := range projects {
		// Count repositories for this project
		repoCount, err := h.db.CountRepositoriesByADOProject(ctx, project.Organization, project.Name)
		if err != nil {
			h.logger.Warn("Failed to count repositories for project", "project", project.Name, "error", err)
			repoCount = 0
		}

		// Get status distribution and progress metrics for this project
		stats := h.getADOProjectStats(ctx, project.Name, project.Organization, repoCount)

		projectStats = append(projectStats, map[string]interface{}{
			"organization":                  project.Name,         // This is actually the project name but kept for compatibility with frontend
			"ado_organization":              project.Organization, // The actual ADO organization
			"project":                       project.Name,
			"total_repos":                   repoCount,
			"status_counts":                 stats.statusCounts,
			"migrated_count":                stats.migratedCount,
			"in_progress_count":             stats.inProgressCount,
			"failed_count":                  stats.failedCount,
			"pending_count":                 stats.pendingCount,
			"migration_progress_percentage": stats.migrationProgressPercentage,
		})
	}

	h.sendJSON(w, http.StatusOK, projectStats)
}

// GetOrganizationList handles GET /api/v1/organizations/list
// Returns a simple list of organization names (for filters/dropdowns)
func (h *Handler) GetOrganizationList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgs, err := h.db.GetDistinctOrganizations(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get organization list", r) {
			return
		}
		h.logger.Error("Failed to get organization list", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch organization list")
		return
	}

	h.sendJSON(w, http.StatusOK, orgs)
}

// GetDashboardActionItems handles GET /api/v1/dashboard/action-items
// Returns all items requiring admin attention: failed migrations, failed dry runs, ready batches, blocked repositories
func (h *Handler) GetDashboardActionItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionItems, err := h.db.GetDashboardActionItems(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get dashboard action items", r) {
			return
		}
		h.logger.Error("Failed to get dashboard action items", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch dashboard action items")
		return
	}

	h.sendJSON(w, http.StatusOK, actionItems)
}

// GetMigrationHistoryList handles GET /api/v1/migrations/history
func (h *Handler) GetMigrationHistoryList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	migrations, err := h.db.GetCompletedMigrations(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get migration history", r) {
			return
		}
		h.logger.Error("Failed to get migration history", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch migration history")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"migrations": migrations,
		"total":      len(migrations),
	})
}

// ExportMigrationHistory handles GET /api/v1/migrations/history/export
func (h *Handler) ExportMigrationHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	format := r.URL.Query().Get("format")

	if format != "csv" && format != "json" {
		h.sendError(w, http.StatusBadRequest, "Invalid format. Must be 'csv' or 'json'")
		return
	}

	migrations, err := h.db.GetCompletedMigrations(ctx)
	if err != nil {
		h.logger.Error("Failed to get migration history for export", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch migration history")
		return
	}

	if format == formatCSV {
		h.exportMigrationHistoryCSV(w, migrations)
	} else {
		h.exportMigrationHistoryJSON(w, migrations)
	}
}

func (h *Handler) exportMigrationHistoryCSV(w http.ResponseWriter, migrations []*storage.CompletedMigration) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=migration_history.csv")

	// Write CSV header
	_, _ = w.Write([]byte("Repository,Source URL,Destination URL,Status,Started At,Completed At,Duration (seconds)\n"))

	// Write data rows
	for _, m := range migrations {
		row := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%d\n",
			escapesCSV(m.FullName),
			escapesCSV(m.SourceURL),
			escapesCSV(stringPtrOrEmpty(m.DestinationURL)),
			escapesCSV(m.Status),
			formatTimePtr(m.StartedAt),
			formatTimePtr(m.CompletedAt),
			intPtrOrZero(m.DurationSeconds),
		)
		_, _ = w.Write([]byte(row))
	}
}

func (h *Handler) exportMigrationHistoryJSON(w http.ResponseWriter, migrations []*storage.CompletedMigration) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=migration_history.json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"migrations":  migrations,
		"total":       len(migrations),
		"exported_at": time.Now().Format(time.RFC3339),
	})
}

func escapesCSV(s string) string {
	// Escape quotes and wrap in quotes if contains comma, quote, or newline
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func intPtrOrZero(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// SelfServiceMigrationRequest represents a request for self-service migration
type SelfServiceMigrationRequest struct {
	Repositories []string          `json:"repositories"`
	Mappings     map[string]string `json:"mappings,omitempty"`
	DryRun       bool              `json:"dry_run"`
}

// SelfServiceMigrationResponse represents the response from self-service migration
type SelfServiceMigrationResponse struct {
	BatchID           int64    `json:"batch_id"`
	BatchName         string   `json:"batch_name"`
	Message           string   `json:"message"`
	TotalRepositories int      `json:"total_repositories"`
	NewlyDiscovered   int      `json:"newly_discovered"`
	AlreadyExisted    int      `json:"already_existed"`
	DiscoveryErrors   []string `json:"discovery_errors,omitempty"`
	ExecutionStarted  bool     `json:"execution_started"`
}

// HandleSelfServiceMigration handles POST /api/v1/self-service/migrate
// This endpoint orchestrates discovery, batch creation, and execution for self-service migrations
//
//nolint:gocyclo // Complex orchestration logic with multiple validation and processing steps
func (h *Handler) HandleSelfServiceMigration(w http.ResponseWriter, r *http.Request) {
	var req SelfServiceMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if len(req.Repositories) == 0 {
		h.sendError(w, http.StatusBadRequest, "No repositories provided")
		return
	}

	// Validate repository format
	for _, repoFullName := range req.Repositories {
		if !strings.Contains(repoFullName, "/") {
			h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid repository format: %s (must be 'org/repo')", repoFullName))
			return
		}
	}

	ctx := r.Context()

	// Check if user has permission to access all requested repositories
	if err := h.checkRepositoriesAccess(ctx, req.Repositories); err != nil {
		h.logger.Warn("Self-service migration access denied", "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	h.logger.Info("Processing self-service migration request",
		"repo_count", len(req.Repositories),
		"dry_run", req.DryRun,
		"has_mappings", len(req.Mappings) > 0)

	// Step 1: Check which repositories exist in database and which need discovery
	var existingRepos []*models.Repository
	var reposToDiscover []string
	discoveryErrors := []string{}

	for _, repoFullName := range req.Repositories {
		repo, err := h.db.GetRepository(ctx, repoFullName)
		if err != nil {
			h.logger.Error("Failed to check repository existence", "repo", repoFullName, "error", err)
			h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check repository: %s", repoFullName))
			return
		}

		if repo != nil {
			existingRepos = append(existingRepos, repo)
			h.logger.Debug("Repository already exists in database", "repo", repoFullName)
		} else {
			reposToDiscover = append(reposToDiscover, repoFullName)
			h.logger.Debug("Repository needs discovery", "repo", repoFullName)
		}
	}

	// Step 2: Discover new repositories
	if len(reposToDiscover) > 0 {
		h.logger.Info("Starting discovery for new repositories", "count", len(reposToDiscover))

		if h.collector == nil {
			h.sendError(w, http.StatusServiceUnavailable, "Discovery service not configured")
			return
		}

		for _, repoFullName := range reposToDiscover {
			// Parse org and repo name
			parts := strings.SplitN(repoFullName, "/", 2)
			if len(parts) != 2 {
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: invalid format", repoFullName))
				continue
			}
			org, repoName := parts[0], parts[1]

			// Get the appropriate client for this organization
			// In JWT-only mode, this creates an org-specific client with installation token
			client, err := h.getClientForOrg(ctx, org)
			if err != nil {
				h.logger.Error("Failed to get client for organization", "repo", repoFullName, "org", org, "error", err)
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: failed to initialize client", repoFullName))
				continue
			}

			// Fetch repository from GitHub API
			ghRepo, _, err := client.REST().Repositories.Get(ctx, org, repoName)
			if err != nil {
				h.logger.Error("Failed to fetch repository from GitHub", "repo", repoFullName, "error", err)
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: not found on source", repoFullName))
				continue
			}

			// Profile repository (includes cloning and git-sizer analysis)
			if err := h.collector.ProfileRepository(ctx, ghRepo); err != nil {
				h.logger.Error("Failed to profile repository", "repo", repoFullName, "error", err)
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: discovery failed - %v", repoFullName, err))
				continue
			}

			// Fetch the newly created repository from database
			repo, err := h.db.GetRepository(ctx, repoFullName)
			if err != nil || repo == nil {
				h.logger.Error("Failed to retrieve discovered repository", "repo", repoFullName, "error", err)
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: failed to save discovery data", repoFullName))
				continue
			}

			existingRepos = append(existingRepos, repo)
			h.logger.Info("Repository discovered and profiled", "repo", repoFullName)
		}
	}

	// Check if we have any repositories to migrate
	if len(existingRepos) == 0 {
		h.sendError(w, http.StatusBadRequest, "No valid repositories to migrate. All repositories failed discovery or validation.")
		return
	}

	// Step 3: Apply destination mappings if provided
	if len(req.Mappings) > 0 {
		h.logger.Info("Applying destination mappings", "count", len(req.Mappings))
		for _, repo := range existingRepos {
			if destFullName, ok := req.Mappings[repo.FullName]; ok {
				repo.DestinationFullName = &destFullName
				if err := h.db.UpdateRepository(ctx, repo); err != nil {
					h.logger.Error("Failed to update repository destination", "repo", repo.FullName, "error", err)
				} else {
					h.logger.Debug("Updated destination mapping", "repo", repo.FullName, "destination", destFullName)
				}
			}
		}
	}

	// Step 4: Create batch with timestamp-based name
	batchName := fmt.Sprintf("Self-Service - %s", time.Now().Format(time.RFC3339))
	batch := &models.Batch{
		Name:            batchName,
		Type:            "self-service",
		Status:          statusPending,
		RepositoryCount: len(existingRepos),
		CreatedAt:       time.Now(),
	}

	if err := h.db.CreateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to create batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to create batch")
		return
	}

	h.logger.Info("Batch created", "batch_id", batch.ID, "batch_name", batch.Name)

	// Step 5: Add repositories to batch
	repoIDs := make([]int64, len(existingRepos))
	for i, repo := range existingRepos {
		repoIDs[i] = repo.ID
	}

	if err := h.db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
		h.logger.Error("Failed to add repositories to batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to add repositories to batch")
		return
	}

	h.logger.Info("Repositories added to batch", "batch_id", batch.ID, "count", len(repoIDs))

	// Update the in-memory repo objects to include the batch_id
	// This is critical because UpdateRepository will use these objects
	for _, repo := range existingRepos {
		repo.BatchID = &batch.ID
	}

	// Step 6: Execute batch (dry run or production)
	executionStarted := false
	var executionError error

	if req.DryRun {
		// Start dry run
		h.logger.Info("Starting dry run for batch", "batch_id", batch.ID)

		// Update batch status - use UpdateBatchProgress to preserve scheduled_at
		now := time.Now()
		if err := h.db.UpdateBatchProgress(ctx, batch.ID, statusInProgress, &now, &now, nil); err != nil {
			h.logger.Error("Failed to update batch status", "error", err)
		}

		// Queue repositories for dry run
		priority := 0
		initiatingUser := getInitiatingUser(ctx)
		for _, repo := range existingRepos {
			repo.Status = string(models.StatusDryRunQueued)
			repo.Priority = priority
			if err := h.db.UpdateRepository(ctx, repo); err != nil {
				h.logger.Error("Failed to queue repository for dry run", "repo", repo.FullName, "error", err)
			}

			// Log the dry run initiation with user info
			logEntry := &models.MigrationLog{
				RepositoryID: repo.ID,
				Level:        "INFO",
				Phase:        "dry_run",
				Operation:    "batch_start",
				Message:      fmt.Sprintf("Dry run started via batch %s", batch.Name),
				InitiatedBy:  initiatingUser,
			}
			if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
				h.logger.Warn("Failed to create migration log", "error", err)
			}
		}
		executionStarted = true
	} else {
		// Start production migration
		h.logger.Info("Starting production migration for batch", "batch_id", batch.ID)

		// Update batch status - use UpdateBatchProgress to preserve scheduled_at
		now := time.Now()
		if err := h.db.UpdateBatchProgress(ctx, batch.ID, statusInProgress, &now, nil, &now); err != nil {
			h.logger.Error("Failed to update batch status", "error", err)
		}

		// Queue repositories for migration
		priority := 0
		initiatingUser := getInitiatingUser(ctx)
		for _, repo := range existingRepos {
			repo.Status = string(models.StatusQueuedForMigration)
			repo.Priority = priority
			if err := h.db.UpdateRepository(ctx, repo); err != nil {
				h.logger.Error("Failed to queue repository for migration", "repo", repo.FullName, "error", err)
				executionError = err
			}

			// Log the migration initiation with user info
			logEntry := &models.MigrationLog{
				RepositoryID: repo.ID,
				Level:        "INFO",
				Phase:        "migration",
				Operation:    "batch_start",
				Message:      fmt.Sprintf("Migration started via batch %s", batch.Name),
				InitiatedBy:  initiatingUser,
			}
			if err := h.db.CreateMigrationLog(ctx, logEntry); err != nil {
				h.logger.Warn("Failed to create migration log", "error", err)
			}
		}
		executionStarted = executionError == nil
	}

	// Step 7: Build response
	response := SelfServiceMigrationResponse{
		BatchID:           batch.ID,
		BatchName:         batch.Name,
		TotalRepositories: len(existingRepos),
		NewlyDiscovered:   len(reposToDiscover) - len(discoveryErrors),
		AlreadyExisted:    len(existingRepos) - (len(reposToDiscover) - len(discoveryErrors)),
		DiscoveryErrors:   discoveryErrors,
		ExecutionStarted:  executionStarted,
	}

	if req.DryRun {
		response.Message = fmt.Sprintf("Self-service dry run started for %d repositories in batch '%s'", len(existingRepos), batch.Name)
	} else {
		response.Message = fmt.Sprintf("Self-service migration started for %d repositories in batch '%s'", len(existingRepos), batch.Name)
	}

	if len(discoveryErrors) > 0 {
		response.Message += fmt.Sprintf(" (Note: %d repositories failed discovery and were skipped)", len(discoveryErrors))
	}

	h.logger.Info("Self-service migration request processed",
		"batch_id", batch.ID,
		"total_repos", response.TotalRepositories,
		"newly_discovered", response.NewlyDiscovered,
		"already_existed", response.AlreadyExisted,
		"discovery_errors", len(discoveryErrors),
		"dry_run", req.DryRun)

	h.sendJSON(w, http.StatusAccepted, response)
}

// ========================================
// Dependency Graph Handlers
// ========================================

// GetRepositoryDependents returns repositories that depend on the specified repository
// GET /api/v1/repositories/{fullName}/dependents
func (h *Handler) GetRepositoryDependents(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// URL decode the fullName and strip /dependents suffix
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	// Strip /dependents suffix if present
	decodedFullName = strings.TrimSuffix(decodedFullName, "/dependents")

	// Get repositories that depend on this one
	dependents, err := h.db.GetDependentRepositories(r.Context(), decodedFullName)
	if err != nil {
		h.logger.Error("Failed to get dependent repositories",
			"repo", decodedFullName,
			"error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to retrieve dependent repositories")
		return
	}

	// Get the dependency types for each dependent
	type DependentRepo struct {
		ID              int64    `json:"id"`
		FullName        string   `json:"full_name"`
		SourceURL       string   `json:"source_url"`
		Status          string   `json:"status"`
		DependencyTypes []string `json:"dependency_types"`
	}

	result := make([]DependentRepo, 0, len(dependents))
	for _, repo := range dependents {
		// Get the dependency details for this repo -> target relationship
		deps, err := h.db.GetRepositoryDependencies(r.Context(), repo.ID)
		if err != nil {
			h.logger.Warn("Failed to get dependencies for repo", "repo", repo.FullName, "error", err)
			continue
		}

		// Find dependency types for the target repo
		depTypes := make([]string, 0)
		seen := make(map[string]bool)
		for _, dep := range deps {
			if dep.DependencyFullName == decodedFullName && !seen[dep.DependencyType] {
				depTypes = append(depTypes, dep.DependencyType)
				seen[dep.DependencyType] = true
			}
		}

		result = append(result, DependentRepo{
			ID:              repo.ID,
			FullName:        repo.FullName,
			SourceURL:       repo.SourceURL,
			Status:          repo.Status,
			DependencyTypes: depTypes,
		})
	}

	response := map[string]interface{}{
		"dependents": result,
		"total":      len(result),
		"target":     decodedFullName,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetDependencyGraph returns enterprise-wide local dependency graph data
// GET /api/v1/dependencies/graph
func (h *Handler) GetDependencyGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	dependencyTypeFilter := r.URL.Query().Get("dependency_type")
	var dependencyTypes []string
	if dependencyTypeFilter != "" {
		dependencyTypes = strings.Split(dependencyTypeFilter, ",")
	}

	// Get all local dependency pairs from the database
	edges, err := h.db.GetAllLocalDependencyPairs(ctx, dependencyTypes)
	if err != nil {
		h.logger.Error("Failed to get dependency graph", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to retrieve dependency graph")
		return
	}

	// Build node set from edges
	nodeMap := make(map[string]bool)
	for _, edge := range edges {
		nodeMap[edge.SourceRepo] = true
		nodeMap[edge.TargetRepo] = true
	}

	// Get repository details for all nodes
	type GraphNode struct {
		ID              string `json:"id"`
		FullName        string `json:"full_name"`
		Organization    string `json:"organization"`
		Status          string `json:"status"`
		DependsOnCount  int    `json:"depends_on_count"`
		DependedByCount int    `json:"depended_by_count"`
	}

	type GraphEdge struct {
		Source         string `json:"source"`
		Target         string `json:"target"`
		DependencyType string `json:"dependency_type"`
	}

	// Count dependencies for each node
	dependsOnCount := make(map[string]int)
	dependedByCount := make(map[string]int)
	for _, edge := range edges {
		dependsOnCount[edge.SourceRepo]++
		dependedByCount[edge.TargetRepo]++
	}

	// Build nodes with repository info
	nodes := make([]GraphNode, 0, len(nodeMap))
	for fullName := range nodeMap {
		repo, err := h.db.GetRepository(ctx, fullName)
		status := "unknown"
		org := ""
		if err == nil && repo != nil {
			status = repo.Status
			parts := strings.Split(repo.FullName, "/")
			if len(parts) > 0 {
				org = parts[0]
			}
		} else {
			parts := strings.Split(fullName, "/")
			if len(parts) > 0 {
				org = parts[0]
			}
		}

		nodes = append(nodes, GraphNode{
			ID:              fullName,
			FullName:        fullName,
			Organization:    org,
			Status:          status,
			DependsOnCount:  dependsOnCount[fullName],
			DependedByCount: dependedByCount[fullName],
		})
	}

	// Convert edges to response format
	graphEdges := make([]GraphEdge, 0, len(edges))
	for _, edge := range edges {
		graphEdges = append(graphEdges, GraphEdge{
			Source:         edge.SourceRepo,
			Target:         edge.TargetRepo,
			DependencyType: edge.DependencyType,
		})
	}

	// Calculate stats
	stats := map[string]interface{}{
		"total_repos_with_dependencies": len(nodeMap),
		"total_local_dependencies":      len(edges),
	}

	// Detect circular dependencies (simple detection: A->B and B->A)
	circularCount := 0
	edgeSet := make(map[string]bool)
	for _, edge := range edges {
		key := edge.SourceRepo + "->" + edge.TargetRepo
		reverseKey := edge.TargetRepo + "->" + edge.SourceRepo
		if edgeSet[reverseKey] {
			circularCount++
		}
		edgeSet[key] = true
	}
	stats["circular_dependency_count"] = circularCount

	response := map[string]interface{}{
		"nodes": nodes,
		"edges": graphEdges,
		"stats": stats,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ExportDependencies exports local dependency data in CSV or JSON format
// GET /api/v1/dependencies/export
func (h *Handler) ExportDependencies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	format := r.URL.Query().Get("format")
	if format == "" {
		format = formatCSV
	}

	// Parse dependency type filter
	dependencyTypeFilter := r.URL.Query().Get("dependency_type")
	var dependencyTypes []string
	if dependencyTypeFilter != "" {
		dependencyTypes = strings.Split(dependencyTypeFilter, ",")
	}

	// Get all local dependency pairs
	edges, err := h.db.GetAllLocalDependencyPairs(ctx, dependencyTypes)
	if err != nil {
		h.logger.Error("Failed to get dependencies for export", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to retrieve dependencies")
		return
	}

	// Build export data with both directions
	type ExportRow struct {
		Repository     string `json:"repository"`
		DependencyName string `json:"dependency_full_name"`
		Direction      string `json:"direction"`
		DependencyType string `json:"dependency_type"`
		DependencyURL  string `json:"dependency_url"`
	}

	exportData := make([]ExportRow, 0, len(edges)*2)

	// Add "depends_on" rows
	for _, edge := range edges {
		exportData = append(exportData, ExportRow{
			Repository:     edge.SourceRepo,
			DependencyName: edge.TargetRepo,
			Direction:      "depends_on",
			DependencyType: edge.DependencyType,
			DependencyURL:  edge.DependencyURL,
		})
	}

	// Add "depended_by" rows (reverse direction)
	for _, edge := range edges {
		exportData = append(exportData, ExportRow{
			Repository:     edge.TargetRepo,
			DependencyName: edge.SourceRepo,
			Direction:      "depended_by",
			DependencyType: edge.DependencyType,
			DependencyURL:  edge.DependencyURL,
		})
	}

	if format == formatJSON {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=dependencies.json")
		if err := json.NewEncoder(w).Encode(exportData); err != nil {
			h.logger.Error("Failed to encode JSON", "error", err)
		}
		return
	}

	// CSV format
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=dependencies.csv")

	// Write CSV header
	fmt.Fprintln(w, "repository,dependency_full_name,direction,dependency_type,dependency_url")

	// Write data rows
	for _, row := range exportData {
		fmt.Fprintf(w, "%s,%s,%s,%s,%s\n",
			escapeCSV(row.Repository),
			escapeCSV(row.DependencyName),
			escapeCSV(row.Direction),
			escapeCSV(row.DependencyType),
			escapeCSV(row.DependencyURL),
		)
	}
}

// repoDependencyExportRow represents a row in the repository dependency export
type repoDependencyExportRow struct {
	Repository     string `json:"repository"`
	DependencyName string `json:"dependency_full_name"`
	Direction      string `json:"direction"`
	DependencyType string `json:"dependency_type"`
	DependencyURL  string `json:"dependency_url"`
}

// collectRepoDependsOn collects dependencies that this repo depends on (local only)
func (h *Handler) collectRepoDependsOn(ctx context.Context, repoID int64, repoFullName string) []repoDependencyExportRow {
	rows := make([]repoDependencyExportRow, 0)
	deps, err := h.db.GetRepositoryDependencies(ctx, repoID)
	if err != nil {
		h.logger.Error("Failed to get dependencies", "repo", repoFullName, "error", err)
		return rows
	}

	for _, dep := range deps {
		if dep.IsLocal {
			rows = append(rows, repoDependencyExportRow{
				Repository:     repoFullName,
				DependencyName: dep.DependencyFullName,
				Direction:      "depends_on",
				DependencyType: dep.DependencyType,
				DependencyURL:  dep.DependencyURL,
			})
		}
	}
	return rows
}

// collectRepoDependedBy collects repositories that depend on this repo (local only)
func (h *Handler) collectRepoDependedBy(ctx context.Context, repoFullName string) []repoDependencyExportRow {
	rows := make([]repoDependencyExportRow, 0)
	dependents, err := h.db.GetDependentRepositories(ctx, repoFullName)
	if err != nil {
		h.logger.Error("Failed to get dependents", "repo", repoFullName, "error", err)
		return rows
	}

	for _, dependent := range dependents {
		depDeps, err := h.db.GetRepositoryDependencies(ctx, dependent.ID)
		if err != nil {
			continue
		}
		for _, dep := range depDeps {
			if dep.DependencyFullName == repoFullName && dep.IsLocal {
				rows = append(rows, repoDependencyExportRow{
					Repository:     repoFullName,
					DependencyName: dependent.FullName,
					Direction:      "depended_by",
					DependencyType: dep.DependencyType,
					DependencyURL:  dependent.SourceURL,
				})
			}
		}
	}
	return rows
}

// writeRepoDependencyExport writes the export data in the specified format
func (h *Handler) writeRepoDependencyExport(w http.ResponseWriter, format, repoFullName string, data []repoDependencyExportRow) {
	filename := strings.ReplaceAll(repoFullName, "/", "-")

	if format == formatJSON {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-dependencies.json", filename))
		if err := json.NewEncoder(w).Encode(data); err != nil {
			h.logger.Error("Failed to encode JSON", "error", err)
		}
		return
	}

	// CSV format
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-dependencies.csv", filename))

	fmt.Fprintln(w, "repository,dependency_full_name,direction,dependency_type,dependency_url")
	for _, row := range data {
		fmt.Fprintf(w, "%s,%s,%s,%s,%s\n",
			escapeCSV(row.Repository),
			escapeCSV(row.DependencyName),
			escapeCSV(row.Direction),
			escapeCSV(row.DependencyType),
			escapeCSV(row.DependencyURL),
		)
	}
}

// ExportRepositoryDependencies exports dependencies for a single repository
// GET /api/v1/repositories/{fullName}/dependencies/export
func (h *Handler) ExportRepositoryDependencies(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// URL decode and clean up the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		decodedFullName = fullName
	}
	decodedFullName = strings.TrimSuffix(decodedFullName, "/dependencies/export")

	ctx := r.Context()
	format := r.URL.Query().Get("format")
	if format == "" {
		format = formatCSV
	}

	// Get the repository first
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Collect export data
	exportData := h.collectRepoDependsOn(ctx, repo.ID, decodedFullName)
	exportData = append(exportData, h.collectRepoDependedBy(ctx, decodedFullName)...)

	// Write output
	h.writeRepoDependencyExport(w, format, decodedFullName, exportData)
}

// escapeCSV escapes a string for CSV output
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
