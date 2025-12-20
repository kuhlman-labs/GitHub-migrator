package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// RepositoryHandler handles repository-related HTTP requests.
// This is a focused handler that separates repository operations from other concerns.
type RepositoryHandler struct {
	*HandlerUtils // Embed shared utilities for checkRepositoryAccess, getClientForOrg
	repoStore     storage.RepositoryStore
	historyStore  storage.MigrationHistoryStore
	logger        *slog.Logger
}

// NewRepositoryHandler creates a new RepositoryHandler with focused dependencies.
func NewRepositoryHandler(
	repoStore storage.RepositoryStore,
	historyStore storage.MigrationHistoryStore,
	logger *slog.Logger,
	sourceDualClient *github.DualClient,
	authConfig *config.AuthConfig,
	sourceBaseURL string,
	sourceBaseConfig *github.ClientConfig,
) *RepositoryHandler {
	return &RepositoryHandler{
		HandlerUtils: NewHandlerUtils(authConfig, sourceDualClient, sourceBaseConfig, sourceBaseURL, logger),
		repoStore:    repoStore,
		historyStore: historyStore,
		logger:       logger,
	}
}

// ListRepositories handles GET /api/v1/repositories
func (h *RepositoryHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pagination := ParsePagination(r)

	// Parse filter parameters
	filters := make(map[string]interface{})
	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}
	if org := r.URL.Query().Get("org"); org != "" {
		filters["organization"] = org
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}
	if batchID := r.URL.Query().Get("batch_id"); batchID != "" {
		filters["batch_id"] = batchID
	}

	// Sort parameters
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy
	}
	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		filters["sort_order"] = sortOrder
	}

	// Add pagination to filters
	filters["limit"] = pagination.Limit
	filters["offset"] = pagination.Offset

	// Get repositories using focused interface
	repos, err := h.repoStore.ListRepositories(ctx, filters)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to list repositories")
		return
	}

	// Get count for pagination
	total, err := h.repoStore.CountRepositoriesWithFilters(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to count repositories", "error", err)
		total = len(repos)
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"repositories": repos,
		"total":        total,
		"limit":        pagination.Limit,
		"offset":       pagination.Offset,
	})
}

// GetRepository handles GET /api/v1/repositories/{fullName}
func (h *RepositoryHandler) GetRepository(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fullName := r.PathValue("fullName")

	// Clean the full name (remove leading/trailing slashes)
	fullName = strings.Trim(fullName, "/")

	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// Check repository access if auth is enabled
	if err := h.CheckRepositoryAccess(ctx, fullName); err != nil {
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	repo, err := h.repoStore.GetRepository(ctx, fullName)
	if err != nil {
		h.logger.Error("Failed to get repository", "full_name", fullName, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get repository")
		return
	}

	if repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Get migration history for the repository
	history, err := h.historyStore.GetMigrationHistory(ctx, repo.ID)
	if err != nil {
		h.logger.Error("Failed to get migration history", "repo_id", repo.ID, "error", err)
		// Continue without history rather than failing
		history = nil
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"repository": repo,
		"history":    history,
	})
}

// UpdateRepository handles PATCH /api/v1/repositories/{fullName}
func (h *RepositoryHandler) UpdateRepository(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fullName := r.PathValue("fullName")
	fullName = strings.Trim(fullName, "/")

	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// Check repository access if auth is enabled
	if err := h.CheckRepositoryAccess(ctx, fullName); err != nil {
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	// Parse request body - support updating status and batch assignment
	var updateReq struct {
		Status  *string `json:"status,omitempty"`
		BatchID *int64  `json:"batch_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get current repository
	repo, err := h.repoStore.GetRepository(ctx, fullName)
	if err != nil {
		h.logger.Error("Failed to get repository", "full_name", fullName, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get repository")
		return
	}

	if repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Update fields if provided
	if updateReq.Status != nil {
		repo.Status = *updateReq.Status
	}
	if updateReq.BatchID != nil {
		repo.BatchID = updateReq.BatchID
	}

	// Save updates
	if err := h.repoStore.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to update repository", "repo_id", repo.ID, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update repository")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"repository": repo,
		"message":    "Repository updated successfully",
	})
}

// Helper methods

// sendJSON sends a JSON response
func (h *RepositoryHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// sendError sends an error response
func (h *RepositoryHandler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, map[string]string{"error": message})
}

// MarkWontMigrate handles marking a repository as wont_migrate
func (h *RepositoryHandler) MarkWontMigrate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fullName := r.PathValue("fullName")
	fullName = strings.Trim(fullName, "/")

	// Remove trailing action if present
	fullName = strings.TrimSuffix(fullName, "/wont-migrate")

	if fullName == "" {
		h.sendError(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	// Check repository access
	if err := h.CheckRepositoryAccess(ctx, fullName); err != nil {
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	repo, err := h.repoStore.GetRepository(ctx, fullName)
	if err != nil {
		h.logger.Error("Failed to get repository", "full_name", fullName, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get repository")
		return
	}

	if repo == nil {
		h.sendError(w, http.StatusNotFound, "Repository not found")
		return
	}

	// Update status
	repo.Status = string(models.StatusWontMigrate)
	if err := h.repoStore.UpdateRepository(ctx, repo); err != nil {
		h.logger.Error("Failed to update repository status", "repo_id", repo.ID, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update repository status")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"repository": repo,
		"message":    "Repository marked as won't migrate",
	})
}
