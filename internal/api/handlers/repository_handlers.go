package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

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
	if availableForBatch := r.URL.Query().Get("available_for_batch"); availableForBatch == boolTrue {
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
func (h *Handler) HandleRepositoryAction(w http.ResponseWriter, r *http.Request) {
	fullPath := r.PathValue("fullName")
	if fullPath == "" {
		h.sendError(w, http.StatusBadRequest, "Repository path is required")
		return
	}

	// Parse the action from the path
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
	// URL decode the fullName
	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
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
		history = []*models.MigrationHistory{}
	}

	response := map[string]interface{}{
		"repository": repo,
		"history":    history,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetRepositoryOrDependencies routes GET requests to either repository details or dependencies
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
		fullName := strings.TrimSuffix(fullPath, "/dependencies")
		h.getRepositoryDependencies(w, r, fullName)
		return
	}

	// Regular repository details request
	h.getRepository(w, r, fullPath)
}

// UpdateRepository handles PATCH /api/v1/repositories/{fullName}
func (h *Handler) UpdateRepository(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

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
func (h *Handler) MarkRepositoryRemediated(w http.ResponseWriter, r *http.Request) {
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

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

	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	if repo.Status != string(models.StatusRemediationRequired) {
		h.sendError(w, http.StatusBadRequest,
			fmt.Sprintf("Repository status must be 'remediation_required', current status: '%s'", repo.Status))
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

	h.logger.Info("Starting re-validation after remediation",
		"repo", decodedFullName,
		"had_oversized_commits", repo.HasOversizedCommits,
		"had_long_refs", repo.HasLongRefs,
		"had_blocking_files", repo.HasBlockingFiles)

	go func() {
		bgCtx := context.Background()
		if err := h.collector.ProfileRepository(bgCtx, ghRepo); err != nil {
			h.logger.Error("Re-validation after remediation failed", "error", err, "repo", decodedFullName)
		} else {
			h.logger.Info("Re-validation completed", "repo", decodedFullName)
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
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

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

	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

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

	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		h.sendError(w, http.StatusBadRequest, "Invalid repository name format")
		return
	}
	org, repoName := parts[0], parts[1]

	migrationClient := h.sourceDualClient.MigrationClient()
	err = migrationClient.UnlockRepository(ctx, org, repoName, *repo.SourceMigrationID)
	if err != nil {
		h.logger.Error("Failed to unlock repository", "error", err, "repo", fullName)
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to unlock repository: %v", err))
		return
	}

	repo.IsSourceLocked = false
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to update repository lock status", "error", err)
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
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	ctx := r.Context()

	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	if repo.Status != string(models.StatusComplete) {
		h.sendError(w, http.StatusBadRequest, "Only completed migrations can be rolled back")
		return
	}

	var req struct {
		Reason string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = ""
	}

	if err := h.db.RollbackRepository(ctx, decodedFullName, req.Reason); err != nil {
		h.logger.Error("Failed to rollback repository", "error", err, "repo", decodedFullName)
		h.sendError(w, http.StatusInternalServerError, "Failed to rollback repository")
		return
	}

	h.logger.Info("Repository rolled back successfully", "repo", fullName, "reason", req.Reason)

	repo, _ = h.db.GetRepository(ctx, fullName)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Repository rolled back successfully",
		"repository": repo,
	})
}

// MarkRepositoryWontMigrate handles POST /api/v1/repositories/{fullName}/mark-wont-migrate
func (h *Handler) MarkRepositoryWontMigrate(w http.ResponseWriter, r *http.Request) {
	fullName, ok := r.Context().Value(cleanFullNameKey).(string)
	if !ok || fullName == "" {
		fullName = r.PathValue("fullName")
	}
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	ctx := r.Context()

	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	var req struct {
		Unmark bool `json:"unmark,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Unmark = false
	}

	var newStatus string
	var message string

	if req.Unmark {
		if repo.Status != string(models.StatusWontMigrate) {
			h.sendError(w, http.StatusBadRequest, "Repository is not marked as won't migrate")
			return
		}
		newStatus = string(models.StatusPending)
		message = "Repository unmarked - changed to pending status"
	} else {
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

	if repo.BatchID != nil && !req.Unmark {
		repo.BatchID = nil
	}

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
