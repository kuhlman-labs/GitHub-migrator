package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
	boolTrue         = "true"
)

// Handler contains all HTTP handlers
type Handler struct {
	db           *storage.Database
	logger       *slog.Logger
	sourceClient *github.Client
	destClient   *github.Client
	collector    *discovery.Collector
}

// NewHandler creates a new Handler instance
// sourceProvider can be nil if discovery is not needed
func NewHandler(db *storage.Database, logger *slog.Logger, sourceClient *github.Client, destClient *github.Client, sourceProvider source.Provider) *Handler {
	var collector *discovery.Collector
	if sourceClient != nil && sourceProvider != nil {
		collector = discovery.NewCollector(sourceClient, db, logger, sourceProvider)
	}
	return &Handler{
		db:           db,
		logger:       logger,
		sourceClient: sourceClient,
		destClient:   destClient,
		collector:    collector,
	}
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
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

	// Search filter
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
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

	h.sendJSON(w, http.StatusOK, repos)
}

// GetRepository handles GET /api/v1/repositories/{fullName}
func (h *Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, fullName)
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

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, fullName)
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

	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to update repository", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update repository")
		return
	}

	h.sendJSON(w, http.StatusOK, repo)
}

// RediscoverRepository handles POST /api/v1/repositories/{fullName}/rediscover
func (h *Handler) RediscoverRepository(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	if h.collector == nil {
		h.sendError(w, http.StatusServiceUnavailable, "Discovery service not configured")
		return
	}

	ctx := r.Context()

	// Check if repository exists
	repo, err := h.db.GetRepository(ctx, fullName)
	if err != nil || repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Extract org and repo name from fullName
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		h.sendError(w, http.StatusBadRequest, "Invalid repository name format")
		return
	}
	org, repoName := parts[0], parts[1]

	// Fetch repository from GitHub API
	ghRepo, _, err := h.sourceClient.REST().Repositories.Get(ctx, org, repoName)
	if err != nil {
		h.logger.Error("Failed to fetch repository from GitHub", "error", err, "repo", fullName)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repository from GitHub")
		return
	}

	// Run discovery asynchronously
	go func() {
		bgCtx := context.Background()
		if err := h.collector.ProfileRepository(bgCtx, ghRepo); err != nil {
			h.logger.Error("Re-discovery failed", "error", err, "repo", fullName)
		} else {
			h.logger.Info("Re-discovery completed", "repo", fullName)
		}
	}()

	h.sendJSON(w, http.StatusAccepted, map[string]string{
		"message":   "Re-discovery started",
		"full_name": fullName,
		"status":    "in_progress",
	})
}

// UnlockRepository handles POST /api/v1/repositories/{fullName}/unlock
func (h *Handler) UnlockRepository(w http.ResponseWriter, r *http.Request) {
	fullName := r.PathValue("fullName")
	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	if h.sourceClient == nil {
		h.sendError(w, http.StatusServiceUnavailable, "Source client not configured")
		return
	}

	ctx := r.Context()

	// Get repository
	repo, err := h.db.GetRepository(ctx, fullName)
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

	// Unlock the repository
	err = h.sourceClient.UnlockRepository(ctx, org, repoName, *repo.SourceMigrationID)
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

	ctx := r.Context()
	batch.CreatedAt = time.Now()
	batch.Status = statusReady

	if err := h.db.CreateBatch(ctx, &batch); err != nil {
		h.logger.Error("Failed to create batch", "error", err)
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

// StartBatch handles POST /api/v1/batches/{id}/start
func (h *Handler) StartBatch(w http.ResponseWriter, r *http.Request) {
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
	batch.Status = statusInProgress
	now := time.Now()
	batch.StartedAt = &now
	if err := h.db.UpdateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to update batch", "error", err)
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

	// Only allow updates for "ready" batches
	if batch.Status != statusReady {
		h.sendError(w, http.StatusBadRequest, "Can only edit batches with 'ready' status")
		return
	}

	// Parse update request
	var updates struct {
		Name        *string    `json:"name,omitempty"`
		Description *string    `json:"description,omitempty"`
		Type        *string    `json:"type,omitempty"`
		ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
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

	if err := h.db.UpdateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to update batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update batch")
		return
	}

	h.sendJSON(w, http.StatusOK, batch)
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

	// Only allow adding repos to "ready" batches
	if batch.Status != statusReady {
		h.sendError(w, http.StatusBadRequest, "Can only add repositories to batches with 'ready' status")
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
	for _, repo := range repos {
		if !isEligibleForBatch(repo.Status) {
			ineligibleRepos = append(ineligibleRepos, repo.FullName)
		}
	}

	if len(ineligibleRepos) > 0 {
		h.sendError(w, http.StatusBadRequest,
			fmt.Sprintf("Some repositories are not eligible for batch assignment: %s",
				strings.Join(ineligibleRepos, ", ")))
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

	// Only allow removing repos from "ready" batches
	if batch.Status != statusReady {
		h.sendError(w, http.StatusBadRequest, "Can only remove repositories from batches with 'ready' status")
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

	// Get repository stats
	stats, err := h.db.GetRepositoryStatsByStatus(ctx)
	if err != nil {
		h.logger.Error("Failed to get repository stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch analytics")
		return
	}

	// Calculate totals
	total := 0
	migrated := stats[string(models.StatusComplete)]
	failed := stats[string(models.StatusMigrationFailed)] + stats[string(models.StatusDryRunFailed)]
	pending := stats[string(models.StatusPending)]

	for _, count := range stats {
		total += count
	}

	inProgress := total - migrated - failed - pending

	// Get discovery statistics
	orgStats, err := h.db.GetOrganizationStats(ctx)
	if err != nil {
		h.logger.Error("Failed to get organization stats", "error", err)
		orgStats = []*storage.OrganizationStats{}
	}

	sizeDistribution, err := h.db.GetSizeDistribution(ctx)
	if err != nil {
		h.logger.Error("Failed to get size distribution", "error", err)
		sizeDistribution = []*storage.SizeDistribution{}
	}

	featureStats, err := h.db.GetFeatureStats(ctx)
	if err != nil {
		h.logger.Error("Failed to get feature stats", "error", err)
		featureStats = &storage.FeatureStats{}
	}

	migrationCompletionStats, err := h.db.GetMigrationCompletionStatsByOrg(ctx)
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
		"status_breakdown":           stats,
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

	total := 0
	for _, count := range stats {
		total += count
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"total":            total,
		"status_breakdown": stats,
	})
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
	allowedStatuses := []string{
		string(models.StatusPending),         // Can queue pending repositories for migration
		string(models.StatusDryRunQueued),    // Allow re-queuing dry runs
		string(models.StatusDryRunFailed),    // Allow retrying failed dry runs
		string(models.StatusDryRunComplete),  // Can queue after successful dry run
		string(models.StatusMigrationFailed), // Allow retrying failed migrations
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
	}

	for _, eligible := range eligibleStatuses {
		if status == eligible {
			return true
		}
	}
	return false
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

	if format == "csv" {
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
