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

	"github.com/brettkuhlman/github-migrator/internal/discovery"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/source"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

const (
	statusInProgress = "in_progress"
	statusReady      = "ready"
	statusPending    = "pending"
	boolTrue         = "true"

	formatCSV  = "csv"
	formatJSON = "json"
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
}

// NewHandler creates a new Handler instance
// sourceProvider can be nil if discovery is not needed
// sourceBaseConfig is used for per-org client creation in enterprise discovery (can be nil for PAT-only mode)
func NewHandler(db *storage.Database, logger *slog.Logger, sourceDualClient *github.DualClient, destDualClient *github.DualClient, sourceProvider source.Provider, sourceBaseConfig *github.ClientConfig) *Handler {
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
	}
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
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
		filters["status"] = status
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
		h.logger.Error("Failed to list repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
		return
	}

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
	// Possible actions: rediscover, mark-remediated, unlock, rollback, mark-wont-migrate
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
		action = "rollback"
		fullName = strings.TrimSuffix(fullPath, "/rollback")
	} else if strings.HasSuffix(fullPath, "/mark-wont-migrate") {
		action = "mark-wont-migrate"
		fullName = strings.TrimSuffix(fullPath, "/mark-wont-migrate")
	} else {
		h.sendError(w, http.StatusNotFound, "Unknown repository action")
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

// RediscoverRepository handles POST /api/v1/repositories/{fullName}/rediscover
func (h *Handler) RediscoverRepository(w http.ResponseWriter, r *http.Request) {
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

	// Check if repository exists
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
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
	// In JWT-only mode, this creates an org-specific client with installation token
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

	// Run discovery asynchronously
	go func() {
		bgCtx := context.Background()
		if err := h.collector.ProfileRepository(bgCtx, ghRepo); err != nil {
			h.logger.Error("Re-discovery failed", "error", err, "repo", decodedFullName)
		} else {
			h.logger.Info("Re-discovery completed", "repo", decodedFullName)
		}
	}()

	h.sendJSON(w, http.StatusAccepted, map[string]string{
		"message":   "Re-discovery started",
		"full_name": decodedFullName,
		"status":    "in_progress",
	})
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
func (h *Handler) AddRepositoriesToBatch(w http.ResponseWriter, r *http.Request) {
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

	// Check each repo is eligible
	ineligibleRepos := []string{}
	ineligibleReasons := make(map[string]string)
	for _, repo := range repos {
		if eligible, reason := isRepositoryEligibleForBatch(repo); !eligible {
			ineligibleRepos = append(ineligibleRepos, repo.FullName)
			ineligibleReasons[repo.FullName] = reason
		}
	}

	if len(ineligibleRepos) > 0 {
		// Build detailed error message
		errorMsg := "Some repositories are not eligible for batch assignment:\n"
		for _, repoName := range ineligibleRepos {
			errorMsg += fmt.Sprintf("  - %s: %s\n", repoName, ineligibleReasons[repoName])
		}
		h.sendError(w, http.StatusBadRequest, strings.TrimSpace(errorMsg))
		return
	}

	// Add repositories to batch
	if err := h.db.AddRepositoriesToBatch(ctx, batchID, req.RepositoryIDs); err != nil {
		h.logger.Error("Failed to add repositories to batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to add repositories to batch")
		return
	}

	// Get updated batch
	batch, _ = h.db.GetBatch(ctx, batchID)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"batch":              batch,
		"repositories_added": len(req.RepositoryIDs),
		"message":            fmt.Sprintf("Added %d repositories to batch", len(req.RepositoryIDs)),
	})
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

	// Queue repositories for retry
	retriedIDs := make([]int64, 0, len(reposToRetry))
	for _, repo := range reposToRetry {
		repo.Status = string(models.StatusQueuedForMigration)
		if err := h.db.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to update repository", "error", err, "repo", repo.FullName)
			continue
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
func (h *Handler) GetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get filter parameters
	orgFilter := r.URL.Query().Get("organization")
	batchFilter := r.URL.Query().Get("batch_id")

	// Get repository stats with filters
	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, batchFilter)
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
	complexityDistribution, err := h.db.GetComplexityDistribution(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get complexity distribution", "error", err)
		complexityDistribution = []*storage.ComplexityDistribution{}
	}

	// Get migration velocity (last 30 days)
	migrationVelocity, err := h.db.GetMigrationVelocity(ctx, orgFilter, batchFilter, 30)
	if err != nil {
		h.logger.Error("Failed to get migration velocity", "error", err)
		migrationVelocity = &storage.MigrationVelocity{}
	}

	// Get migration time series
	migrationTimeSeries, err := h.db.GetMigrationTimeSeries(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get migration time series", "error", err)
		migrationTimeSeries = []*storage.MigrationTimeSeriesPoint{}
	}

	// Get average migration time
	avgMigrationTime, err := h.db.GetAverageMigrationTime(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get average migration time", "error", err)
		avgMigrationTime = 0
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
	orgStats, err := h.db.GetOrganizationStatsFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get organization stats", "error", err)
		orgStats = []*storage.OrganizationStats{}
	}

	sizeDistribution, err := h.db.GetSizeDistributionFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get size distribution", "error", err)
		sizeDistribution = []*storage.SizeDistribution{}
	}

	featureStats, err := h.db.GetFeatureStatsFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get feature stats", "error", err)
		featureStats = &storage.FeatureStats{}
	}

	migrationCompletionStats, err := h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, batchFilter)
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
		"estimated_completion_date":  estimatedCompletionDate,
		"organization_stats":         orgStats,
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
	batchFilter := r.URL.Query().Get("batch_id")

	// Get basic analytics
	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, batchFilter)
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
	migrationVelocity, err := h.db.GetMigrationVelocity(ctx, orgFilter, batchFilter, 30)
	if err != nil {
		h.logger.Error("Failed to get migration velocity", "error", err)
		migrationVelocity = &storage.MigrationVelocity{}
	}

	// Get migration time series for trend analysis
	migrationTimeSeries, err := h.db.GetMigrationTimeSeries(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get migration time series", "error", err)
		migrationTimeSeries = []*storage.MigrationTimeSeriesPoint{}
	}

	// Get average migration time
	avgMigrationTime, err := h.db.GetAverageMigrationTime(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get average migration time", "error", err)
		avgMigrationTime = 0
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

	// Get organization breakdowns
	migrationCompletionStats, err := h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get migration completion stats", "error", err)
		migrationCompletionStats = []*storage.MigrationCompletionStats{}
	}

	// Get complexity distribution
	complexityDistribution, err := h.db.GetComplexityDistribution(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get complexity distribution", "error", err)
		complexityDistribution = []*storage.ComplexityDistribution{}
	}

	// Get size distribution
	sizeDistribution, err := h.db.GetSizeDistributionFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get size distribution", "error", err)
		sizeDistribution = []*storage.SizeDistribution{}
	}

	// Get feature stats
	featureStats, err := h.db.GetFeatureStatsFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get feature stats", "error", err)
		featureStats = &storage.FeatureStats{}
	}

	// Calculate risk metrics
	highComplexityPending := 0
	veryLargePending := 0
	for _, dist := range complexityDistribution {
		if dist.Category == "complex" || dist.Category == "very_complex" {
			highComplexityPending += dist.Count
		}
	}
	for _, dist := range sizeDistribution {
		if dist.Category == "very_large" {
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

	// Build executive report
	report := map[string]interface{}{
		// Executive Summary
		"executive_summary": map[string]interface{}{
			"total_repositories":        total,
			"completion_percentage":     completionRate,
			"migrated_count":            migrated,
			"in_progress_count":         inProgress,
			"pending_count":             pending,
			"failed_count":              failed,
			"success_rate":              successRate,
			"estimated_completion_date": estimatedCompletionDate,
			"days_remaining":            daysRemaining,
			"first_migration_date":      firstMigrationDate,
			"report_generated_at":       time.Now().Format(time.RFC3339),
		},

		// Migration Velocity & Timeline
		"velocity_metrics": map[string]interface{}{
			"repos_per_day":        migrationVelocity.ReposPerDay,
			"repos_per_week":       migrationVelocity.ReposPerWeek,
			"average_duration_sec": avgMigrationTime,
			"migration_trend":      migrationTimeSeries,
		},

		// Organization Progress
		"organization_progress": migrationCompletionStats,

		// Risk & Complexity Analysis
		"risk_analysis": map[string]interface{}{
			"high_complexity_pending": highComplexityPending,
			"very_large_pending":      veryLargePending,
			"failed_migrations":       failed,
			"complexity_distribution": complexityDistribution,
			"size_distribution":       sizeDistribution,
		},

		// Batch Performance
		"batch_performance": map[string]interface{}{
			"total_batches":       len(batches),
			"completed_batches":   completedBatches,
			"in_progress_batches": inProgressBatches,
			"pending_batches":     pendingBatches,
		},

		// Feature Migration Status
		"feature_migration_status": featureStats,

		// Detailed Status Breakdown
		"status_breakdown": stats,
	}

	h.sendJSON(w, http.StatusOK, report)
}

// ExportExecutiveReport handles GET /api/v1/analytics/executive-report/export
func (h *Handler) ExportExecutiveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	format := r.URL.Query().Get("format")
	orgFilter := r.URL.Query().Get("organization")
	batchFilter := r.URL.Query().Get("batch_id")

	if format != formatCSV && format != formatJSON {
		h.sendError(w, http.StatusBadRequest, "Invalid format. Must be 'csv' or 'json'")
		return
	}

	// Get all the same data as GetExecutiveReport
	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, batchFilter)
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

	migrationVelocity, err := h.db.GetMigrationVelocity(ctx, orgFilter, batchFilter, 30)
	if err != nil {
		migrationVelocity = &storage.MigrationVelocity{}
	}

	avgMigrationTime, err := h.db.GetAverageMigrationTime(ctx, orgFilter, batchFilter)
	if err != nil {
		avgMigrationTime = 0
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

	migrationCompletionStats, err := h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		migrationCompletionStats = []*storage.MigrationCompletionStats{}
	}

	complexityDistribution, err := h.db.GetComplexityDistribution(ctx, orgFilter, batchFilter)
	if err != nil {
		complexityDistribution = []*storage.ComplexityDistribution{}
	}

	sizeDistribution, err := h.db.GetSizeDistributionFiltered(ctx, orgFilter, batchFilter)
	if err != nil {
		sizeDistribution = []*storage.SizeDistribution{}
	}

	featureStats, err := h.db.GetFeatureStatsFiltered(ctx, orgFilter, batchFilter)
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
		h.exportExecutiveReportCSV(w, total, migrated, inProgress, pending, failed, completionRate, successRate,
			estimatedCompletionDate, daysRemaining, migrationVelocity, int(avgMigrationTime),
			migrationCompletionStats, complexityDistribution, sizeDistribution, featureStats,
			stats, completedBatches, inProgressBatches, pendingBatches)
	} else {
		h.exportExecutiveReportJSON(w, total, migrated, inProgress, pending, failed, completionRate, successRate,
			estimatedCompletionDate, daysRemaining, migrationVelocity, int(avgMigrationTime),
			migrationCompletionStats, complexityDistribution, sizeDistribution, featureStats,
			stats, completedBatches, inProgressBatches, pendingBatches)
	}
}

func (h *Handler) exportExecutiveReportCSV(w http.ResponseWriter, total, migrated, inProgress, pending, failed int,
	completionRate, successRate float64, estimatedCompletionDate string, daysRemaining int,
	velocity *storage.MigrationVelocity, avgMigrationTime int,
	orgStats []*storage.MigrationCompletionStats, complexityDist []*storage.ComplexityDistribution,
	sizeDist []*storage.SizeDistribution, featureStats *storage.FeatureStats,
	statusBreakdown map[string]int, completedBatches, inProgressBatches, pendingBatches int) {

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=executive_migration_report.csv")

	var output strings.Builder

	// Section 1: Executive Summary
	output.WriteString("EXECUTIVE MIGRATION PROGRESS REPORT\n")
	output.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	output.WriteString("\n")

	output.WriteString("=== EXECUTIVE SUMMARY ===\n")
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

	// Section 2: Velocity Metrics
	output.WriteString("=== MIGRATION VELOCITY ===\n")
	output.WriteString("Metric,Value\n")
	output.WriteString(fmt.Sprintf("Repos Per Day,%.1f\n", velocity.ReposPerDay))
	output.WriteString(fmt.Sprintf("Repos Per Week,%.1f\n", velocity.ReposPerWeek))
	if avgMigrationTime > 0 {
		avgMinutes := avgMigrationTime / 60
		output.WriteString(fmt.Sprintf("Average Migration Time,%d minutes\n", avgMinutes))
	}
	output.WriteString("\n")

	// Section 3: Organization Progress
	output.WriteString("=== ORGANIZATION PROGRESS ===\n")
	output.WriteString("Organization,Total,Completed,In Progress,Pending,Failed,Completion %\n")
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

	// Section 4: Risk Analysis - Complexity
	output.WriteString("=== RISK ANALYSIS - COMPLEXITY ===\n")
	output.WriteString("Complexity Category,Repository Count\n")
	for _, dist := range complexityDist {
		output.WriteString(fmt.Sprintf("%s,%d\n", escapesCSV(dist.Category), dist.Count))
	}
	output.WriteString("\n")

	// Section 5: Risk Analysis - Size
	output.WriteString("=== RISK ANALYSIS - SIZE ===\n")
	output.WriteString("Size Category,Repository Count\n")
	for _, dist := range sizeDist {
		output.WriteString(fmt.Sprintf("%s,%d\n", escapesCSV(dist.Category), dist.Count))
	}
	output.WriteString("\n")

	// Section 6: Feature Migration Status
	output.WriteString("=== FEATURE MIGRATION STATUS ===\n")
	output.WriteString("Feature,Repository Count,Percentage\n")
	totalRepos := featureStats.TotalRepositories
	if totalRepos > 0 {
		output.WriteString(fmt.Sprintf("Archived,%d,%.1f%%\n", featureStats.IsArchived, float64(featureStats.IsArchived)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("LFS,%d,%.1f%%\n", featureStats.HasLFS, float64(featureStats.HasLFS)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Submodules,%d,%.1f%%\n", featureStats.HasSubmodules, float64(featureStats.HasSubmodules)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Large Files,%d,%.1f%%\n", featureStats.HasLargeFiles, float64(featureStats.HasLargeFiles)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("GitHub Actions,%d,%.1f%%\n", featureStats.HasActions, float64(featureStats.HasActions)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Wikis,%d,%.1f%%\n", featureStats.HasWiki, float64(featureStats.HasWiki)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Pages,%d,%.1f%%\n", featureStats.HasPages, float64(featureStats.HasPages)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Discussions,%d,%.1f%%\n", featureStats.HasDiscussions, float64(featureStats.HasDiscussions)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Projects,%d,%.1f%%\n", featureStats.HasProjects, float64(featureStats.HasProjects)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Branch Protections,%d,%.1f%%\n", featureStats.HasBranchProtections, float64(featureStats.HasBranchProtections)/float64(totalRepos)*100))
	}
	output.WriteString("\n")

	// Section 7: Batch Performance
	output.WriteString("=== BATCH PERFORMANCE ===\n")
	output.WriteString("Status,Count\n")
	output.WriteString(fmt.Sprintf("Completed,%d\n", completedBatches))
	output.WriteString(fmt.Sprintf("In Progress,%d\n", inProgressBatches))
	output.WriteString(fmt.Sprintf("Pending,%d\n", pendingBatches))
	output.WriteString("\n")

	// Section 8: Detailed Status Breakdown
	output.WriteString("=== DETAILED STATUS BREAKDOWN ===\n")
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

func (h *Handler) exportExecutiveReportJSON(w http.ResponseWriter, total, migrated, inProgress, pending, failed int,
	completionRate, successRate float64, estimatedCompletionDate string, daysRemaining int,
	velocity *storage.MigrationVelocity, avgMigrationTime int,
	orgStats []*storage.MigrationCompletionStats, complexityDist []*storage.ComplexityDistribution,
	sizeDist []*storage.SizeDistribution, featureStats *storage.FeatureStats,
	statusBreakdown map[string]int, completedBatches, inProgressBatches, pendingBatches int) {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=executive_migration_report.json")

	report := map[string]interface{}{
		"report_metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"report_type":  "Executive Migration Progress Report",
			"version":      "1.0",
		},
		"executive_summary": map[string]interface{}{
			"total_repositories":        total,
			"completion_percentage":     completionRate,
			"migrated_count":            migrated,
			"in_progress_count":         inProgress,
			"pending_count":             pending,
			"failed_count":              failed,
			"success_rate":              successRate,
			"estimated_completion_date": estimatedCompletionDate,
			"days_remaining":            daysRemaining,
		},
		"velocity_metrics": map[string]interface{}{
			"repos_per_day":        velocity.ReposPerDay,
			"repos_per_week":       velocity.ReposPerWeek,
			"average_duration_sec": avgMigrationTime,
		},
		"organization_progress":    orgStats,
		"complexity_distribution":  complexityDist,
		"size_distribution":        sizeDist,
		"feature_migration_status": featureStats,
		"batch_performance": map[string]interface{}{
			"completed_batches":   completedBatches,
			"in_progress_batches": inProgressBatches,
			"pending_batches":     pendingBatches,
		},
		"status_breakdown": statusBreakdown,
	}

	if err := json.NewEncoder(w).Encode(report); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
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

// ListOrganizations handles GET /api/v1/organizations
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgStats, err := h.db.GetOrganizationStats(ctx)
	if err != nil {
		h.logger.Error("Failed to get organization stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch organizations")
		return
	}

	h.sendJSON(w, http.StatusOK, orgStats)
}

// GetOrganizationList handles GET /api/v1/organizations/list
// Returns a simple list of organization names (for filters/dropdowns)
func (h *Handler) GetOrganizationList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgs, err := h.db.GetDistinctOrganizations(ctx)
	if err != nil {
		h.logger.Error("Failed to get organization list", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch organization list")
		return
	}

	h.sendJSON(w, http.StatusOK, orgs)
}

// GetMigrationHistoryList handles GET /api/v1/migrations/history
func (h *Handler) GetMigrationHistoryList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	migrations, err := h.db.GetCompletedMigrations(ctx)
	if err != nil {
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
		for _, repo := range existingRepos {
			repo.Status = string(models.StatusDryRunQueued)
			repo.Priority = priority
			if err := h.db.UpdateRepository(ctx, repo); err != nil {
				h.logger.Error("Failed to queue repository for dry run", "repo", repo.FullName, "error", err)
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
		for _, repo := range existingRepos {
			repo.Status = string(models.StatusQueuedForMigration)
			repo.Priority = priority
			if err := h.db.UpdateRepository(ctx, repo); err != nil {
				h.logger.Error("Failed to queue repository for migration", "repo", repo.FullName, "error", err)
				executionError = err
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
