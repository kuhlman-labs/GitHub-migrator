package handlers

import (
	"encoding/json"
	"fmt"
	"io"
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
		WriteError(w, ErrDatabaseFetch.WithDetails("batches"))
		return
	}
	h.sendJSON(w, http.StatusOK, batches)
}

// CreateBatch handles POST /api/v1/batches
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var batch models.Batch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if strings.TrimSpace(batch.Name) == "" {
		WriteError(w, ErrMissingField.WithField("name"))
		return
	}

	if batch.MigrationAPI != "" && batch.MigrationAPI != models.MigrationAPIGEI && batch.MigrationAPI != models.MigrationAPIELM {
		WriteError(w, ErrInvalidField.WithDetails("Invalid migration_api. Must be 'GEI' or 'ELM'"))
		return
	}

	if batch.MigrationAPI == "" {
		batch.MigrationAPI = models.MigrationAPIGEI
	}

	ctx := r.Context()
	batch.CreatedAt = time.Now()
	batch.Status = models.BatchStatusPending

	if err := h.db.CreateBatch(ctx, &batch); err != nil {
		h.logger.Error("Failed to create batch", "error", err, "name", batch.Name)

		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "unique constraint") {
			WriteError(w, ErrConflict.WithDetails(fmt.Sprintf("A batch with the name '%s' already exists. Please choose a different name.", batch.Name)))
			return
		}

		WriteError(w, ErrDatabaseUpdate.WithDetails("batch creation"))
		return
	}

	h.sendJSON(w, http.StatusCreated, batch)
}

// GetBatch handles GET /api/v1/batches/{id}
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()
	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()

	var req RunDryRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		// Reject malformed JSON, but allow empty body (optional request)
		WriteError(w, ErrInvalidJSON)
		return
	}

	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
		return
	}

	if batch.Status != models.BatchStatusPending && batch.Status != models.BatchStatusReady {
		WriteError(w, ErrBadRequest.WithDetails("Can only run dry run on batches with 'pending' or 'ready' status"))
		return
	}

	repos, err := h.db.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		h.logger.Error("Failed to get batch repositories", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
		return
	}

	if len(repos) == 0 {
		WriteError(w, ErrBadRequest.WithDetails("Batch has no repositories"))
		return
	}

	priority := 0
	if batch.Type == models.BatchTypePilot {
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
		WriteError(w, ErrBadRequest.WithDetails(fmt.Sprintf("No repositories to run dry run. %d repositories were skipped.", skippedCount)))
		return
	}

	now := time.Now()
	if err := h.db.UpdateBatchProgress(ctx, batch.ID, models.BatchStatusInProgress, &now, &now, nil); err != nil {
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()

	var req StartBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		// Reject malformed JSON, but allow empty body (optional request)
		WriteError(w, ErrInvalidJSON)
		return
	}

	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
		return
	}

	if batch.Status == models.BatchStatusPending && !req.SkipDryRun {
		WriteError(w, ErrBadRequest.WithDetails("Batch is in 'pending' state. Run dry run first or set skip_dry_run=true"))
		return
	}

	if batch.Status != models.BatchStatusReady && batch.Status != models.BatchStatusPending {
		WriteError(w, ErrBadRequest.WithDetails("Can only start batches with 'ready' or 'pending' status"))
		return
	}

	repos, err := h.db.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		h.logger.Error("Failed to get batch repositories", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
		return
	}

	if len(repos) == 0 {
		WriteError(w, ErrBadRequest.WithDetails("Batch has no repositories"))
		return
	}

	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.CheckRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Start batch access denied", "batch_id", batchID, "error", err)
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
		return
	}

	priority := 0
	if batch.Type == models.BatchTypePilot {
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
	if err := h.db.UpdateBatchProgress(ctx, batch.ID, models.BatchStatusInProgress, &now, nil, &now); err != nil {
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()

	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
		return
	}

	if batch.Status != models.BatchStatusReady && batch.Status != models.BatchStatusPending {
		WriteError(w, ErrBadRequest.WithDetails("Can only edit batches with 'pending' or 'ready' status"))
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
		WriteError(w, ErrInvalidJSON)
		return
	}

	if updates.MigrationAPI != nil && *updates.MigrationAPI != models.MigrationAPIGEI && *updates.MigrationAPI != models.MigrationAPIELM {
		WriteError(w, ErrInvalidField.WithDetails("Invalid migration_api. Must be 'GEI' or 'ELM'"))
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
		// A change occurred if:
		// 1. Old was nil and new value is explicitly provided (even if empty string)
		// 2. Old and new have different dereferenced values
		destinationOrgChanged = (batch.DestinationOrg == nil) || (oldDestinationOrg != newDestinationOrg)
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
		WriteError(w, ErrDatabaseUpdate.WithDetails("batch"))
		return
	}

	if destinationOrgChanged {
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
				parts := strings.Split(repo.FullName, "/")
				if len(parts) != 2 {
					continue
				}
				repoName := parts[1]

				// Case 1: Setting destination_org for the first time - initialize repos without destinations
				if oldDestinationOrg == "" && newDestinationOrg != "" {
					if repo.DestinationFullName == nil || *repo.DestinationFullName == "" {
						newDestination := fmt.Sprintf("%s/%s", newDestinationOrg, repoName)
						repo.DestinationFullName = &newDestination
						if err := h.db.UpdateRepository(ctx, repo); err != nil {
							h.logger.Error("Failed to set repository destination",
								"repo_id", repo.ID,
								"repo_name", repo.FullName,
								"error", err)
						} else {
							updatedCount++
						}
					}
					continue
				}

				// Case 2: Updating from old org to new org - only update repos matching old pattern
				if repo.DestinationFullName == nil || *repo.DestinationFullName == "" {
					continue
				}

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
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()

	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
		return
	}

	if batch.Status == models.BatchStatusInProgress {
		WriteError(w, ErrBadRequest.WithDetails("Cannot delete batch in 'in_progress' status"))
		return
	}

	if err := h.db.DeleteBatch(ctx, batchID); err != nil {
		h.logger.Error("Failed to delete batch", "error", err, "batch_id", batchID)
		WriteError(w, ErrDatabaseUpdate.WithDetails("batch deletion"))
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()

	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
		return
	}

	if batch.Status != models.BatchStatusReady && batch.Status != models.BatchStatusPending {
		WriteError(w, ErrBadRequest.WithDetails("Can only add repositories to batches with 'pending' or 'ready' status"))
		return
	}

	var req BatchRepositoryIDsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if len(req.RepositoryIDs) == 0 {
		WriteError(w, ErrMissingField.WithDetails("repository_ids"))
		return
	}

	repos, err := h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
	if err != nil {
		h.logger.Error("Failed to get repositories", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("repository validation"))
		return
	}

	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.CheckRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Add repositories to batch access denied", "batch_id", batchID, "error", err)
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
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
		WriteError(w, ErrBadRequest.WithDetails(strings.TrimSpace(errorMsg)))
		return
	}

	if err := h.db.AddRepositoriesToBatch(ctx, batchID, eligibleRepoIDs); err != nil {
		h.logger.Error("Failed to add repositories to batch", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("adding repositories to batch"))
		return
	}

	// Core operation succeeded - now try to apply batch defaults
	// If this fails, we continue with partial success since repos are already in the batch
	repos, err = h.db.GetRepositoriesByIDs(ctx, eligibleRepoIDs)
	if err != nil {
		h.logger.Warn("Failed to re-fetch repositories for batch defaults - repos were added but defaults not applied",
			"batch_id", batchID,
			"error", err)
		// Continue with partial success - repos are in the batch, just can't apply defaults
		repos = nil
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

	batch, err = h.db.GetBatch(ctx, batchID)
	if err != nil {
		h.logger.Warn("Failed to refresh batch after adding repositories", "batch_id", batchID, "error", err)
		// Continue without batch in response - the operation succeeded, just can't return updated batch
	}

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
		"repositories_added":     len(eligibleRepoIDs),
		"repositories_requested": len(req.RepositoryIDs),
		"defaults_applied_count": updatedCount,
		"message":                message,
	}

	// Only include batch in response if successfully retrieved
	if batch != nil {
		response["batch"] = batch
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()

	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
		return
	}

	if batch.Status != models.BatchStatusReady && batch.Status != models.BatchStatusPending {
		WriteError(w, ErrBadRequest.WithDetails("Can only remove repositories from batches with 'pending' or 'ready' status"))
		return
	}

	var req BatchRepositoryIDsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if len(req.RepositoryIDs) == 0 {
		WriteError(w, ErrMissingField.WithDetails("repository_ids"))
		return
	}

	if err := h.db.RemoveRepositoriesFromBatch(ctx, batchID, req.RepositoryIDs); err != nil {
		h.logger.Error("Failed to remove repositories from batch", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("removing repositories from batch"))
		return
	}

	batch, err = h.db.GetBatch(ctx, batchID)
	if err != nil {
		h.logger.Warn("Failed to refresh batch after removing repositories", "batch_id", batchID, "error", err)
		// Continue without batch in response - the operation succeeded, just can't return updated batch
	}

	response := map[string]interface{}{
		"repositories_removed": len(req.RepositoryIDs),
		"message":              fmt.Sprintf("Removed %d repositories from batch", len(req.RepositoryIDs)),
	}

	// Only include batch in response if successfully retrieved
	if batch != nil {
		response["batch"] = batch
	}

	h.sendJSON(w, http.StatusOK, response)
}

// RetryBatchFailures handles POST /api/v1/batches/{id}/retry
//
//nolint:gocyclo // Complexity is due to multiple validation and error handling paths
func (h *Handler) RetryBatchFailures(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	batchID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, ErrInvalidField.WithDetails("Invalid batch ID"))
		return
	}

	ctx := r.Context()

	batch, err := h.db.GetBatch(ctx, batchID)
	if err != nil {
		if h.handleContextError(ctx, err, "get batch", r) {
			return
		}
		h.logger.Error("Failed to get batch", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("batch"))
		return
	}

	if batch == nil {
		WriteError(w, ErrBatchNotFound)
		return
	}

	var req RetryBatchRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.RepositoryIDs = nil
	}

	var reposToRetry []*models.Repository

	if len(req.RepositoryIDs) > 0 {
		reposToRetry, err = h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
		if err != nil {
			h.logger.Error("Failed to get repositories", "error", err)
			WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
			return
		}

		for _, repo := range reposToRetry {
			if repo.BatchID == nil || *repo.BatchID != batchID {
				WriteError(w, ErrBadRequest.WithDetails(
					fmt.Sprintf("Repository %s is not in this batch", repo.FullName)))
				return
			}
			if repo.Status != string(models.StatusMigrationFailed) && repo.Status != string(models.StatusDryRunFailed) {
				WriteError(w, ErrBadRequest.WithDetails(
					fmt.Sprintf("Repository %s is not in a failed state", repo.FullName)))
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
			WriteError(w, ErrDatabaseFetch.WithDetails("failed repositories"))
			return
		}
	}

	if len(reposToRetry) == 0 {
		WriteError(w, ErrBadRequest.WithDetails("No failed repositories to retry"))
		return
	}

	repoFullNames := make([]string, len(reposToRetry))
	for i, repo := range reposToRetry {
		repoFullNames[i] = repo.FullName
	}
	if err := h.CheckRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Retry batch access denied", "batch_id", batchID, "error", err)
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
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
