package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// ListRepositories handles GET /api/v1/repositories
func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filters from query parameters using the dedicated struct
	repoFilters := ParseRepositoryFilters(r)

	// Log status filter for debugging
	if len(repoFilters.Status) > 0 {
		h.logger.Info("Status filter", "status", repoFilters.Status)
	}

	// Convert to map for storage layer compatibility
	filters := repoFilters.ToMap()

	repos, err := h.db.ListRepositories(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "list repositories", r) {
			return
		}
		h.logger.Error("Failed to list repositories", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
		return
	}

	// Get total count if pagination is used
	response := map[string]interface{}{
		"repositories": repos,
	}

	if repoFilters.HasPagination() {
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
		WriteError(w, ErrMissingField.WithField("fullName"))
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
		WriteError(w, ErrNotFound.WithDetails("Unknown repository action"))
		return
	}

	// Check if user has permission to access this repository
	if err := h.checkRepositoryAccess(r.Context(), fullName); err != nil {
		h.logger.Warn("Repository access denied", "repo", fullName, "action", action, "error", err)
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
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
		WriteError(w, ErrNotFound.WithDetails("Unknown repository action"))
	}
}

// GetRepository handles GET /api/v1/repositories/{fullName}
func (h *Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		WriteError(w, ErrMissingField.WithField("fullName"))
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
		WriteError(w, ErrDatabaseFetch.WithDetails("repository"))
		return
	}

	if repo == nil {
		WriteError(w, ErrRepositoryNotFound)
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
		WriteError(w, ErrMissingField.WithField("fullName"))
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
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}

	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		WriteError(w, ErrRepositoryNotFound)
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
		WriteError(w, ErrDatabaseUpdate.WithDetails("repository"))
		return
	}

	h.sendJSON(w, http.StatusOK, repo)
}

// ResetRepositoryStatus handles POST /api/v1/repositories/{fullName}/reset
func (h *Handler) ResetRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	decodedFullName, err := h.getDecodedRepoName(r)
	if err != nil {
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		WriteError(w, ErrRepositoryNotFound)
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
		WriteError(w, ErrBadRequest.WithDetails(fmt.Sprintf("Cannot reset repository in status '%s'. Only in-progress states can be reset.", repo.Status)))
		return
	}

	h.logger.Info("Resetting repository status",
		"repo", repo.FullName,
		"old_status", repo.Status)

	repo.Status = string(models.StatusPending)
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to reset repository status", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("repository status"))
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
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		WriteError(w, ErrRepositoryNotFound)
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
		WriteError(w, ErrServiceUnavailable.WithDetails("ADO discovery service not configured"))
		return
	}

	h.logger.Info("Delegating rediscovery to ADO handler", "repo", decodedFullName)

	if err := h.adoHandler.RediscoverADORepository(ctx, repo); err != nil {
		h.logger.Error("Failed to rediscover ADO repository", "error", err, "repo", decodedFullName)
		WriteError(w, ErrInternal.WithDetails(fmt.Sprintf("Failed to rediscover repository: %v", err)))
		return
	}

	updatedRepo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil {
		h.logger.Error("Failed to fetch updated repository", "error", err, "repo", decodedFullName)
		WriteError(w, ErrDatabaseFetch.WithDetails("updated repository"))
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
		WriteError(w, ErrClientNotConfigured.WithDetails("GitHub discovery service"))
		return
	}

	parts := strings.SplitN(decodedFullName, "/", 2)
	if len(parts) != 2 {
		WriteError(w, ErrInvalidField.WithDetails("Invalid repository name format - expected 'org/repo'"))
		return
	}
	org, repoName := parts[0], parts[1]

	client, err := h.getClientForOrg(ctx, org)
	if err != nil {
		h.logger.Error("Failed to get client for organization", "error", err, "org", org)
		WriteError(w, ErrInternal.WithDetails("Failed to initialize client for repository"))
		return
	}

	ghRepo, _, err := client.REST().Repositories.Get(ctx, org, repoName)
	if err != nil {
		h.logger.Error("Failed to fetch repository from GitHub", "error", err, "repo", decodedFullName)
		WriteError(w, ErrInternal.WithDetails("Failed to fetch repository from GitHub"))
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
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}

	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	if h.collector == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Discovery service"))
		return
	}

	ctx := r.Context()

	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		WriteError(w, ErrRepositoryNotFound)
		return
	}

	if repo.Status != string(models.StatusRemediationRequired) {
		WriteError(w, ErrBadRequest.WithDetails(
			fmt.Sprintf("Repository status must be 'remediation_required', current status: '%s'", repo.Status)))
		return
	}

	parts := strings.SplitN(decodedFullName, "/", 2)
	if len(parts) != 2 {
		WriteError(w, ErrInvalidField.WithDetails("Invalid repository name format - expected 'org/repo'"))
		return
	}
	org, repoName := parts[0], parts[1]

	client, err := h.getClientForOrg(ctx, org)
	if err != nil {
		h.logger.Error("Failed to get client for organization", "error", err, "org", org)
		WriteError(w, ErrInternal.WithDetails("Failed to initialize client for repository"))
		return
	}

	ghRepo, _, err := client.REST().Repositories.Get(ctx, org, repoName)
	if err != nil {
		h.logger.Error("Failed to fetch repository from GitHub", "error", err, "repo", decodedFullName)
		WriteError(w, ErrInternal.WithDetails("Failed to fetch repository from GitHub"))
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
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}

	decodedFullName, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decodedFullName = fullName
	}

	if h.sourceDualClient == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Source client"))
		return
	}

	ctx := r.Context()

	repo, err := h.db.GetRepository(ctx, decodedFullName)
	if err != nil || repo == nil {
		WriteError(w, ErrRepositoryNotFound)
		return
	}

	if repo.SourceMigrationID == nil {
		WriteError(w, ErrBadRequest.WithDetails("No migration ID found for this repository"))
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid repository name format - expected 'org/repo'"))
		return
	}
	org, repoName := parts[0], parts[1]

	migrationClient := h.sourceDualClient.MigrationClient()
	err = migrationClient.UnlockRepository(ctx, org, repoName, *repo.SourceMigrationID)
	if err != nil {
		h.logger.Error("Failed to unlock repository", "error", err, "repo", fullName)
		WriteError(w, ErrInternal.WithDetails(fmt.Sprintf("Failed to unlock repository: %v", err)))
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
		WriteError(w, ErrMissingField.WithField("fullName"))
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
		WriteError(w, ErrRepositoryNotFound)
		return
	}

	if repo.Status != string(models.StatusComplete) {
		WriteError(w, ErrBadRequest.WithDetails("Only completed migrations can be rolled back"))
		return
	}

	var req RollbackRepositoryRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = ""
	}

	if err := h.db.RollbackRepository(ctx, decodedFullName, req.Reason); err != nil {
		h.logger.Error("Failed to rollback repository", "error", err, "repo", decodedFullName)
		WriteError(w, ErrDatabaseUpdate.WithDetails("repository rollback"))
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
		WriteError(w, ErrMissingField.WithField("fullName"))
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
		WriteError(w, ErrRepositoryNotFound)
		return
	}

	var req MarkWontMigrateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Unmark = false
	}

	var newStatus string
	var message string

	if req.Unmark {
		if repo.Status != string(models.StatusWontMigrate) {
			WriteError(w, ErrBadRequest.WithDetails("Repository is not marked as won't migrate"))
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
			WriteError(w, ErrBadRequest.WithDetails(fmt.Sprintf("Cannot mark repository with status '%s' as won't migrate", repo.Status)))
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
		WriteError(w, ErrDatabaseUpdate.WithDetails("repository status"))
		return
	}

	h.logger.Info("Repository wont_migrate status changed", "repo", fullName, "status", newStatus)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":    message,
		"repository": repo,
	})
}
