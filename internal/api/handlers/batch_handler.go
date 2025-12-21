package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// BatchHandler handles batch-related HTTP requests.
// This is a focused handler that separates batch operations from other concerns.
type BatchHandler struct {
	*HandlerUtils  // Embed shared utilities for checkRepositoryAccess
	batchStore     storage.BatchStore
	repoStore      storage.RepositoryStore
	logger         *slog.Logger
	destDualClient *github.DualClient
}

// NewBatchHandler creates a new BatchHandler with focused dependencies.
func NewBatchHandler(
	batchStore storage.BatchStore,
	repoStore storage.RepositoryStore,
	logger *slog.Logger,
	sourceDualClient *github.DualClient,
	destDualClient *github.DualClient,
	authConfig *config.AuthConfig,
	sourceBaseURL string,
) *BatchHandler {
	return &BatchHandler{
		HandlerUtils:   NewHandlerUtils(authConfig, sourceDualClient, nil, sourceBaseURL, logger),
		batchStore:     batchStore,
		repoStore:      repoStore,
		logger:         logger,
		destDualClient: destDualClient,
	}
}

// ListBatches handles GET /api/v1/batches
func (h *BatchHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	batches, err := h.batchStore.ListBatches(ctx)
	if err != nil {
		h.logger.Error("Failed to list batches", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to list batches")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"batches": batches,
		"total":   len(batches),
	})
}

// GetBatch handles GET /api/v1/batches/{id}
func (h *BatchHandler) GetBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	batch, err := h.batchStore.GetBatch(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get batch", "id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Get repositories in this batch
	repos, err := h.repoStore.ListRepositories(ctx, map[string]interface{}{
		"batch_id": id,
	})
	if err != nil {
		h.logger.Error("Failed to get batch repositories", "batch_id", id, "error", err)
		// Continue without repos rather than failing
		repos = nil
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"batch":        batch,
		"repositories": repos,
	})
}

// CreateBatch handles POST /api/v1/batches
func (h *BatchHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var createReq struct {
		Name        string  `json:"name"`
		Description *string `json:"description,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if createReq.Name == "" {
		h.sendError(w, http.StatusBadRequest, "Batch name is required")
		return
	}

	batch := &models.Batch{
		Name:         createReq.Name,
		Description:  createReq.Description,
		Status:       models.BatchStatusPending,
		Type:         "wave", // Default batch type
		MigrationAPI: "GEI",  // Default migration API
	}

	if err := h.batchStore.CreateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to create batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to create batch")
		return
	}

	h.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"batch":   batch,
		"message": "Batch created successfully",
	})
}

// UpdateBatch handles PATCH /api/v1/batches/{id}
func (h *BatchHandler) UpdateBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	batch, err := h.batchStore.GetBatch(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get batch", "id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	var updateReq struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
		Status      *string `json:"status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields if provided
	if updateReq.Name != nil {
		batch.Name = *updateReq.Name
	}
	if updateReq.Description != nil {
		batch.Description = updateReq.Description
	}
	if updateReq.Status != nil {
		batch.Status = *updateReq.Status
	}

	if err := h.batchStore.UpdateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to update batch", "id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update batch")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"batch":   batch,
		"message": "Batch updated successfully",
	})
}

// DeleteBatch handles DELETE /api/v1/batches/{id}
func (h *BatchHandler) DeleteBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	batch, err := h.batchStore.GetBatch(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get batch", "id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	// Only allow deleting pending batches
	if batch.Status != models.BatchStatusPending {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Cannot delete batch with status '%s'. Only pending batches can be deleted.", batch.Status))
		return
	}

	// Remove batch assignment from repositories
	repos, err := h.repoStore.ListRepositories(ctx, map[string]interface{}{
		"batch_id": id,
	})
	if err != nil {
		h.logger.Error("Failed to get batch repositories", "batch_id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get batch repositories")
		return
	}

	for _, repo := range repos {
		repo.BatchID = nil
		if err := h.repoStore.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to remove repository from batch", "repo_id", repo.ID, "error", err)
			// Continue with other repos
		}
	}

	if err := h.batchStore.DeleteBatch(ctx, id); err != nil {
		h.logger.Error("Failed to delete batch", "id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to delete batch")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Batch deleted successfully",
	})
}

// addRepoToBatchResult holds the result of adding a single repository to a batch
type addRepoToBatchResult struct {
	added bool
	err   string
}

// addSingleRepoToBatch attempts to add a single repository to a batch
func (h *BatchHandler) addSingleRepoToBatch(ctx context.Context, repoID, batchID int64) addRepoToBatchResult {
	repo, err := h.repoStore.GetRepositoryByID(ctx, repoID)
	if err != nil {
		return addRepoToBatchResult{err: fmt.Sprintf("Repository %d: not found", repoID)}
	}
	if repo == nil {
		return addRepoToBatchResult{err: fmt.Sprintf("Repository %d: not found", repoID)}
	}

	if repo.BatchID != nil && *repo.BatchID != batchID {
		return addRepoToBatchResult{err: fmt.Sprintf("Repository %s: already in another batch", repo.FullName)}
	}

	eligible, reason := isRepositoryEligibleForBatch(repo)
	if !eligible {
		return addRepoToBatchResult{err: fmt.Sprintf("Repository %s: %s", repo.FullName, reason)}
	}

	repo.BatchID = &batchID
	if err := h.repoStore.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to add repository to batch", "repo_id", repoID, "error", err)
		return addRepoToBatchResult{err: fmt.Sprintf("Repository %d: failed to add", repoID)}
	}

	return addRepoToBatchResult{added: true}
}

// AddRepositoriesToBatch handles POST /api/v1/batches/{id}/repositories
func (h *BatchHandler) AddRepositoriesToBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	batch, err := h.batchStore.GetBatch(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get batch", "id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	var addReq struct {
		RepositoryIDs []int64 `json:"repository_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&addReq); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(addReq.RepositoryIDs) == 0 {
		h.sendError(w, http.StatusBadRequest, "No repository IDs provided")
		return
	}

	// Check repository access for all repos if auth is enabled
	if err := h.checkBatchReposAccess(ctx, addReq.RepositoryIDs); err != nil {
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	added := 0
	errors := make([]string, 0)

	for _, repoID := range addReq.RepositoryIDs {
		result := h.addSingleRepoToBatch(ctx, repoID, id)
		if result.added {
			added++
		} else if result.err != "" {
			errors = append(errors, result.err)
		}
	}

	response := map[string]interface{}{
		"added":   added,
		"total":   len(addReq.RepositoryIDs),
		"message": fmt.Sprintf("Added %d of %d repositories to batch", added, len(addReq.RepositoryIDs)),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	h.sendJSON(w, http.StatusOK, response)
}

// checkBatchReposAccess checks if the user has access to all repositories in the batch
func (h *BatchHandler) checkBatchReposAccess(ctx context.Context, repoIDs []int64) error {
	if h.authConfig == nil || !h.authConfig.Enabled {
		return nil
	}

	for _, repoID := range repoIDs {
		repo, err := h.repoStore.GetRepositoryByID(ctx, repoID)
		if err != nil || repo == nil {
			continue
		}
		if err := h.CheckRepositoryAccess(ctx, repo.FullName); err != nil {
			return fmt.Errorf("access denied to repository %s: %w", repo.FullName, err)
		}
	}
	return nil
}

// RemoveRepositoriesFromBatch handles DELETE /api/v1/batches/{id}/repositories
func (h *BatchHandler) RemoveRepositoriesFromBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	batch, err := h.batchStore.GetBatch(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get batch", "id", id, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get batch")
		return
	}

	if batch == nil {
		h.sendError(w, http.StatusNotFound, "Batch not found")
		return
	}

	var removeReq struct {
		RepositoryIDs []int64 `json:"repository_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&removeReq); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(removeReq.RepositoryIDs) == 0 {
		h.sendError(w, http.StatusBadRequest, "No repository IDs provided")
		return
	}

	removed := 0

	for _, repoID := range removeReq.RepositoryIDs {
		repo, err := h.repoStore.GetRepositoryByID(ctx, repoID)
		if err != nil || repo == nil {
			continue
		}

		if repo.BatchID == nil || *repo.BatchID != id {
			continue
		}

		repo.BatchID = nil
		if err := h.repoStore.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to remove repository from batch", "repo_id", repoID, "error", err)
			continue
		}

		removed++
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"removed": removed,
		"message": fmt.Sprintf("Removed %d repositories from batch", removed),
	})
}

// Helper methods

// sendJSON sends a JSON response
func (h *BatchHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// sendError sends an error response
func (h *BatchHandler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, map[string]string{"error": message})
}
