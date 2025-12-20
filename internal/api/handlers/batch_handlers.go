package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

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

	if strings.TrimSpace(batch.Name) == "" {
		h.sendError(w, http.StatusBadRequest, "Batch name is required")
		return
	}

	if batch.MigrationAPI != "" && batch.MigrationAPI != models.MigrationAPIGEI && batch.MigrationAPI != models.MigrationAPIELM {
		h.sendError(w, http.StatusBadRequest, "Invalid migration_api. Must be 'GEI' or 'ELM'")
		return
	}

	if batch.MigrationAPI == "" {
		batch.MigrationAPI = models.MigrationAPIGEI
	}

	ctx := r.Context()
	batch.CreatedAt = time.Now()
	batch.Status = statusPending

	if err := h.db.CreateBatch(ctx, &batch); err != nil {
		h.logger.Error("Failed to create batch", "error", err, "name", batch.Name)

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

	var req struct {
		OnlyPending bool `json:"only_pending,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

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

	if batch.Status != statusPending && batch.Status != statusReady {
		h.sendError(w, http.StatusBadRequest, "Can only run dry run on batches with 'pending' or 'ready' status")
		return
	}

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

	priority := 0
	if batch.Type == "pilot" {
		priority = 1
	}

	dryRunIDs := make([]int64, 0, len(repos))
	skippedCount := 0

	for _, repo := range repos {
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

	var req struct {
		SkipDryRun bool `json:"skip_dry_run,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

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

	if batch.Status == statusPending && !req.SkipDryRun {
		h.sendError(w, http.StatusBadRequest, "Batch is in 'pending' state. Run dry run first or set skip_dry_run=true")
		return
	}

	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only start batches with 'ready' or 'pending' status")
		return
	}

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

	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Start batch access denied", "batch_id", batchID, "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

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

	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only edit batches with 'pending' or 'ready' status")
		return
	}

	var updates struct {
		Name               *string    `json:"name,omitempty"`
		Description        *string    `json:"description,omitempty"`
		Type               *string    `json:"type,omitempty"`
		ScheduledAt        *time.Time `json:"scheduled_at,omitempty"`
		DestinationOrg     *string    `json:"destination_org,omitempty"`
		MigrationAPI       *string    `json:"migration_api,omitempty"`
		ExcludeReleases    *bool      `json:"exclude_releases,omitempty"`
		ExcludeAttachments *bool      `json:"exclude_attachments,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if updates.MigrationAPI != nil && *updates.MigrationAPI != models.MigrationAPIGEI && *updates.MigrationAPI != models.MigrationAPIELM {
		h.sendError(w, http.StatusBadRequest, "Invalid migration_api. Must be 'GEI' or 'ELM'")
		return
	}

	oldDestinationOrg := ""
	newDestinationOrg := ""
	destinationOrgChanged := false

	if updates.DestinationOrg != nil {
		newDestinationOrg = *updates.DestinationOrg
		if batch.DestinationOrg != nil {
			oldDestinationOrg = *batch.DestinationOrg
		}
		destinationOrgChanged = oldDestinationOrg != newDestinationOrg
	}

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
	if updates.ExcludeAttachments != nil {
		batch.ExcludeAttachments = *updates.ExcludeAttachments
	}

	if err := h.db.UpdateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to update batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update batch")
		return
	}

	if destinationOrgChanged && oldDestinationOrg != "" {
		h.logger.Info("Batch destination_org changed, updating repository destinations",
			"batch_id", batchID,
			"old_destination_org", oldDestinationOrg,
			"new_destination_org", newDestinationOrg)

		repos, err := h.db.ListRepositories(ctx, map[string]interface{}{"batch_id": batchID})
		if err != nil {
			h.logger.Error("Failed to list batch repositories for destination update", "error", err)
		} else {
			updatedCount := 0
			for _, repo := range repos {
				if repo.DestinationFullName == nil || *repo.DestinationFullName == "" {
					continue
				}

				parts := strings.Split(repo.FullName, "/")
				if len(parts) != 2 {
					continue
				}
				repoName := parts[1]

				expectedOldDestination := fmt.Sprintf("%s/%s", oldDestinationOrg, repoName)
				if *repo.DestinationFullName == expectedOldDestination {
					if newDestinationOrg != "" {
						newDestination := fmt.Sprintf("%s/%s", newDestinationOrg, repoName)
						repo.DestinationFullName = &newDestination
					} else {
						repo.DestinationFullName = nil
					}

					if err := h.db.UpdateRepository(ctx, repo); err != nil {
						h.logger.Error("Failed to update repository destination",
							"repo_id", repo.ID,
							"repo_name", repo.FullName,
							"error", err)
					} else {
						updatedCount++
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

	if batch.Status == "in_progress" {
		h.sendError(w, http.StatusBadRequest, "Cannot delete batch in 'in_progress' status")
		return
	}

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
//
//nolint:gocyclo // TODO: refactor to reduce complexity
func (h *Handler) AddRepositoriesToBatch(w http.ResponseWriter, r *http.Request) {
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

	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only add repositories to batches with 'pending' or 'ready' status")
		return
	}

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

	repos, err := h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
	if err != nil {
		h.logger.Error("Failed to get repositories", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to validate repositories")
		return
	}

	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Add repositories to batch access denied", "batch_id", batchID, "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

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

	if len(eligibleRepoIDs) == 0 {
		errorMsg := "No repositories are eligible for batch assignment:\n"
		for _, repoName := range ineligibleRepos {
			errorMsg += fmt.Sprintf("  - %s: %s\n", repoName, ineligibleReasons[repoName])
		}
		h.sendError(w, http.StatusBadRequest, strings.TrimSpace(errorMsg))
		return
	}

	if err := h.db.AddRepositoriesToBatch(ctx, batchID, eligibleRepoIDs); err != nil {
		h.logger.Error("Failed to add repositories to batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to add repositories to batch")
		return
	}

	repos, err = h.db.GetRepositoriesByIDs(ctx, eligibleRepoIDs)
	if err != nil {
		h.logger.Error("Failed to re-fetch repositories after adding to batch", "error", err)
	}

	updatedCount := 0
	failedUpdates := []string{}

	for _, repo := range repos {
		needsUpdate := false

		if batch.DestinationOrg != nil && *batch.DestinationOrg != "" && repo.DestinationFullName == nil {
			destinationFullName := fmt.Sprintf("%s/%s", *batch.DestinationOrg, repo.Name())
			repo.DestinationFullName = &destinationFullName
			needsUpdate = true
		}

		if batch.ExcludeReleases && !repo.ExcludeReleases {
			repo.ExcludeReleases = batch.ExcludeReleases
			needsUpdate = true
		}

		if batch.ExcludeAttachments && !repo.ExcludeAttachments {
			repo.ExcludeAttachments = batch.ExcludeAttachments
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

	batch, _ = h.db.GetBatch(ctx, batchID)

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

	if len(ineligibleRepos) > 0 {
		response["ineligible_count"] = len(ineligibleRepos)
		response["ineligible_repos"] = ineligibleReasons
	}

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

	if batch.Status != statusReady && batch.Status != statusPending {
		h.sendError(w, http.StatusBadRequest, "Can only remove repositories from batches with 'pending' or 'ready' status")
		return
	}

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

	if err := h.db.RemoveRepositoriesFromBatch(ctx, batchID, req.RepositoryIDs); err != nil {
		h.logger.Error("Failed to remove repositories from batch", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to remove repositories from batch")
		return
	}

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

	var req struct {
		RepositoryIDs []int64 `json:"repository_ids,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.RepositoryIDs = nil
	}

	var reposToRetry []*models.Repository

	if len(req.RepositoryIDs) > 0 {
		reposToRetry, err = h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
		if err != nil {
			h.logger.Error("Failed to get repositories", "error", err)
			h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
			return
		}

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

	repoFullNames := make([]string, len(reposToRetry))
	for i, repo := range reposToRetry {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Retry batch access denied", "batch_id", batchID, "error", err)
		h.sendError(w, http.StatusForbidden, err.Error())
		return
	}

	retriedIDs := make([]int64, 0, len(reposToRetry))
	initiatingUser := getInitiatingUser(ctx)
	for _, repo := range reposToRetry {
		repo.Status = string(models.StatusQueuedForMigration)
		if err := h.db.UpdateRepository(ctx, repo); err != nil {
			h.logger.Error("Failed to update repository", "error", err, "repo", repo.FullName)
			continue
		}

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
