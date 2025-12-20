package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// Action constants for batch repository status updates
const (
	actionRollback = "rollback"
)

// StartMigrationRequest defines the request for starting migrations
type StartMigrationRequest struct {
	RepositoryIDs []int64  `json:"repository_ids,omitempty"`
	FullNames     []string `json:"full_names,omitempty"`
	DryRun        bool     `json:"dry_run"`
	Priority      int      `json:"priority"`
}

// StartMigrationResponse defines the response for starting migrations
type StartMigrationResponse struct {
	MigrationIDs []int64 `json:"migration_ids"`
	Message      string  `json:"message"`
	Count        int     `json:"count"`
}

// StartMigration handles POST /api/v1/migrations/start
func (h *Handler) StartMigration(w http.ResponseWriter, r *http.Request) {
	var req StartMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	ctx := r.Context()
	var repos []*models.Repository
	var err error

	if len(req.RepositoryIDs) > 0 {
		repos, err = h.db.GetRepositoriesByIDs(ctx, req.RepositoryIDs)
	} else if len(req.FullNames) > 0 {
		repos, err = h.db.GetRepositoriesByNames(ctx, req.FullNames)
	} else {
		WriteError(w, ErrMissingField.WithDetails("Must provide repository_ids or full_names"))
		return
	}

	if err != nil {
		h.logger.Error("Failed to fetch repositories", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
		return
	}

	if len(repos) == 0 {
		WriteError(w, ErrRepositoryNotFound)
		return
	}

	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	if err := h.checkRepositoriesAccess(ctx, repoFullNames); err != nil {
		h.logger.Warn("Start migration access denied", "error", err)
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
		return
	}

	migrationIDs := make([]int64, 0, len(repos))
	for _, repo := range repos {
		if !canMigrate(repo.Status) {
			h.logger.Warn("Repository cannot be migrated",
				"repo", repo.FullName,
				"status", repo.Status)
			continue
		}

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
		WriteError(w, ErrInvalidJSON)
		return
	}

	if err := validateBatchAction(req.Action); err != nil {
		WriteError(w, ErrBadRequest.WithDetails(err.Error()))
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
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
		return
	}

	response := h.executeBatchUpdate(ctx, repos, req.Action, req.Reason)
	statusCode := determineBatchUpdateStatusCode(response)
	h.sendJSON(w, statusCode, response)
}

func validateBatchAction(action string) error {
	validActions := map[string]bool{
		"mark_migrated":       true,
		"mark_wont_migrate":   true,
		"unmark_wont_migrate": true,
		actionRollback:        true,
	}
	if !validActions[action] {
		return fmt.Errorf("invalid action. Must be 'mark_migrated', 'mark_wont_migrate', 'unmark_wont_migrate', or 'rollback'")
	}
	return nil
}

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

func (h *Handler) handleBatchUpdateError(w http.ResponseWriter, err error) {
	if err.Error() == "no repositories found" {
		WriteError(w, ErrRepositoryNotFound)
	} else if err.Error() == "must provide repository_ids or full_names" {
		WriteError(w, ErrMissingField.WithDetails("repository_ids or full_names"))
	} else {
		WriteError(w, ErrInternal.WithDetails(err.Error()))
	}
}

func (h *Handler) checkBatchUpdatePermissions(ctx context.Context, repos []*models.Repository) error {
	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	return h.checkRepositoriesAccess(ctx, repoFullNames)
}

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

func buildBatchUpdateMessage(updatedCount, failedCount, totalCount int) string {
	if failedCount == 0 {
		return fmt.Sprintf("Successfully updated %d repositories", updatedCount)
	} else if updatedCount == 0 {
		return fmt.Sprintf("Failed to update all %d repositories", failedCount)
	}
	return fmt.Sprintf("Updated %d of %d repositories (%d failed)", updatedCount, totalCount, failedCount)
}

func determineBatchUpdateStatusCode(response BatchUpdateRepositoryStatusResponse) int {
	if response.FailedCount > 0 && response.UpdatedCount == 0 {
		return http.StatusBadRequest
	} else if response.FailedCount > 0 {
		return http.StatusMultiStatus
	}
	return http.StatusOK
}

func (h *Handler) processBatchUpdate(ctx context.Context, repo *models.Repository, action string, initiatingUser *string, reason string) error {
	switch action {
	case "mark_migrated":
		return h.markRepositoryMigrated(ctx, repo, initiatingUser)
	case "mark_wont_migrate":
		return h.markRepositoryWontMigrateBatch(ctx, repo, false, initiatingUser)
	case "unmark_wont_migrate":
		return h.markRepositoryWontMigrateBatch(ctx, repo, true, initiatingUser)
	case actionRollback:
		return h.rollbackRepositoryBatch(ctx, repo, reason, initiatingUser)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
}

func (h *Handler) markRepositoryMigrated(ctx context.Context, repo *models.Repository, initiatingUser *string) error {
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

	repo.Status = string(models.StatusComplete)
	now := time.Now()
	repo.MigratedAt = &now
	repo.UpdatedAt = now
	repo.BatchID = nil

	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	message := "Repository marked as migrated (external migration)"
	if initiatingUser != nil {
		message = fmt.Sprintf("Repository marked as migrated by %s (external migration)", *initiatingUser)
	}

	history := &models.MigrationHistory{
		RepositoryID: repo.ID,
		Status:       models.BatchStatusCompleted,
		Phase:        "migration",
		Message:      &message,
		StartedAt:    now,
		CompletedAt:  &now,
	}

	if _, err := h.db.CreateMigrationHistory(ctx, history); err != nil {
		h.logger.Warn("Failed to create migration history", "error", err)
	}

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

func (h *Handler) markRepositoryWontMigrateBatch(ctx context.Context, repo *models.Repository, unmark bool, initiatingUser *string) error {
	var newStatus string
	var message string
	var operation string

	if unmark {
		if repo.Status != string(models.StatusWontMigrate) {
			return fmt.Errorf("repository is not marked as won't migrate")
		}
		newStatus = string(models.StatusPending)
		message = "Repository unmarked - changed to pending status"
		operation = "unmark_wont_migrate"
	} else {
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

	if repo.BatchID != nil && !unmark {
		repo.BatchID = nil
	}

	repo.Status = newStatus
	repo.UpdatedAt = time.Now()
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

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

func (h *Handler) rollbackRepositoryBatch(ctx context.Context, repo *models.Repository, reason string, initiatingUser *string) error {
	if repo.Status != string(models.StatusComplete) {
		return fmt.Errorf("only completed migrations can be rolled back (current status: %s)", repo.Status)
	}

	reasonMessage := reason
	if reasonMessage == "" {
		reasonMessage = "Repository rolled back via batch operation"
	}
	if initiatingUser != nil {
		reasonMessage = fmt.Sprintf("%s (by %s)", reasonMessage, *initiatingUser)
	}

	if err := h.db.RollbackRepository(ctx, repo.FullName, reasonMessage); err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	logEntry := &models.MigrationLog{
		RepositoryID: repo.ID,
		Level:        "INFO",
		Phase:        actionRollback,
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid repository ID"))
		return
	}

	ctx := r.Context()
	repo, err := h.db.GetRepositoryByID(ctx, id)
	if err != nil {
		if h.handleContextError(ctx, err, "get repository by ID", r) {
			return
		}
		h.logger.Error("Failed to get repository", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("migration status"))
		return
	}

	if repo == nil {
		WriteError(w, ErrNotFound.WithDetails("Migration not found"))
		return
	}

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
		WriteError(w, ErrInvalidField.WithDetails("Invalid repository ID"))
		return
	}

	ctx := r.Context()
	history, err := h.db.GetMigrationHistory(ctx, id)
	if err != nil {
		if h.handleContextError(ctx, err, "get migration history", r) {
			return
		}
		h.logger.Error("Failed to get migration history", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("migration history"))
		return
	}

	h.sendJSON(w, http.StatusOK, history)
}

// GetMigrationLogs handles GET /api/v1/migrations/{id}/logs
func (h *Handler) GetMigrationLogs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, ErrInvalidField.WithDetails("Invalid repository ID"))
		return
	}

	query := r.URL.Query()
	level := query.Get("level")
	phase := query.Get("phase")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	limit := 500
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
		WriteError(w, ErrDatabaseFetch.WithDetails("migration logs"))
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
//
//nolint:gocyclo // Complex orchestration logic with multiple validation and processing steps
func (h *Handler) HandleSelfServiceMigration(w http.ResponseWriter, r *http.Request) {
	var req SelfServiceMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if len(req.Repositories) == 0 {
		WriteError(w, ErrMissingField.WithDetails("No repositories provided"))
		return
	}

	for _, repoFullName := range req.Repositories {
		if !strings.Contains(repoFullName, "/") {
			WriteError(w, ErrInvalidField.WithDetails(fmt.Sprintf("Invalid repository format: %s (must be 'org/repo')", repoFullName)))
			return
		}
	}

	ctx := r.Context()

	if err := h.checkRepositoriesAccess(ctx, req.Repositories); err != nil {
		h.logger.Warn("Self-service migration access denied", "error", err)
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
		return
	}

	h.logger.Info("Processing self-service migration request",
		"repo_count", len(req.Repositories),
		"dry_run", req.DryRun,
		"has_mappings", len(req.Mappings) > 0)

	var existingRepos []*models.Repository
	var reposToDiscover []string
	discoveryErrors := []string{}

	for _, repoFullName := range req.Repositories {
		repo, err := h.db.GetRepository(ctx, repoFullName)
		if err != nil {
			h.logger.Error("Failed to check repository existence", "repo", repoFullName, "error", err)
			WriteError(w, ErrDatabaseFetch.WithDetails(fmt.Sprintf("repository %s", repoFullName)))
			return
		}

		if repo != nil {
			existingRepos = append(existingRepos, repo)
		} else {
			reposToDiscover = append(reposToDiscover, repoFullName)
		}
	}

	if len(reposToDiscover) > 0 {
		h.logger.Info("Starting discovery for new repositories", "count", len(reposToDiscover))

		if h.collector == nil {
			WriteError(w, ErrClientNotConfigured.WithDetails("Discovery service"))
			return
		}

		for _, repoFullName := range reposToDiscover {
			parts := strings.SplitN(repoFullName, "/", 2)
			if len(parts) != 2 {
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: invalid format", repoFullName))
				continue
			}
			org, repoName := parts[0], parts[1]

			client, err := h.GetClientForOrg(ctx, org)
			if err != nil {
				h.logger.Error("Failed to get client for organization", "repo", repoFullName, "org", org, "error", err)
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: failed to initialize client", repoFullName))
				continue
			}

			ghRepo, _, err := client.REST().Repositories.Get(ctx, org, repoName)
			if err != nil {
				h.logger.Error("Failed to fetch repository from GitHub", "repo", repoFullName, "error", err)
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: not found on source", repoFullName))
				continue
			}

			if err := h.collector.ProfileRepository(ctx, ghRepo); err != nil {
				h.logger.Error("Failed to profile repository", "repo", repoFullName, "error", err)
				discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: discovery failed - %v", repoFullName, err))
				continue
			}

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

	if len(existingRepos) == 0 {
		WriteError(w, ErrBadRequest.WithDetails("No valid repositories to migrate. All repositories failed discovery or validation."))
		return
	}

	if len(req.Mappings) > 0 {
		h.logger.Info("Applying destination mappings", "count", len(req.Mappings))
		for _, repo := range existingRepos {
			if destFullName, ok := req.Mappings[repo.FullName]; ok {
				repo.DestinationFullName = &destFullName
				if err := h.db.UpdateRepository(ctx, repo); err != nil {
					h.logger.Error("Failed to update repository destination", "repo", repo.FullName, "error", err)
				}
			}
		}
	}

	batchName := fmt.Sprintf("Self-Service - %s", time.Now().Format(time.RFC3339))
	batch := &models.Batch{
		Name:            batchName,
		Type:            "self-service",
		Status:          models.BatchStatusPending,
		RepositoryCount: len(existingRepos),
		CreatedAt:       time.Now(),
	}

	if err := h.db.CreateBatch(ctx, batch); err != nil {
		h.logger.Error("Failed to create batch", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("batch creation"))
		return
	}

	h.logger.Info("Batch created", "batch_id", batch.ID, "batch_name", batch.Name)

	repoIDs := make([]int64, len(existingRepos))
	for i, repo := range existingRepos {
		repoIDs[i] = repo.ID
	}

	if err := h.db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
		h.logger.Error("Failed to add repositories to batch", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("adding repositories to batch"))
		return
	}

	h.logger.Info("Repositories added to batch", "batch_id", batch.ID, "count", len(repoIDs))

	for _, repo := range existingRepos {
		repo.BatchID = &batch.ID
	}

	executionStarted := false
	var executionError error

	if req.DryRun {
		h.logger.Info("Starting dry run for batch", "batch_id", batch.ID)

		now := time.Now()
		if err := h.db.UpdateBatchProgress(ctx, batch.ID, models.BatchStatusInProgress, &now, &now, nil); err != nil {
			h.logger.Error("Failed to update batch status", "error", err)
		}

		priority := 0
		initiatingUser := getInitiatingUser(ctx)
		for _, repo := range existingRepos {
			repo.Status = string(models.StatusDryRunQueued)
			repo.Priority = priority
			if err := h.db.UpdateRepository(ctx, repo); err != nil {
				h.logger.Error("Failed to queue repository for dry run", "repo", repo.FullName, "error", err)
			}

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
		h.logger.Info("Starting production migration for batch", "batch_id", batch.ID)

		now := time.Now()
		if err := h.db.UpdateBatchProgress(ctx, batch.ID, models.BatchStatusInProgress, &now, nil, &now); err != nil {
			h.logger.Error("Failed to update batch status", "error", err)
		}

		priority := 0
		initiatingUser := getInitiatingUser(ctx)
		for _, repo := range existingRepos {
			repo.Status = string(models.StatusQueuedForMigration)
			repo.Priority = priority
			if err := h.db.UpdateRepository(ctx, repo); err != nil {
				h.logger.Error("Failed to queue repository for migration", "repo", repo.FullName, "error", err)
				executionError = err
			}

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

// GetMigrationHistoryList handles GET /api/v1/migrations/history
func (h *Handler) GetMigrationHistoryList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	migrations, err := h.db.GetCompletedMigrations(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get migration history", r) {
			return
		}
		h.logger.Error("Failed to get migration history", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("migration history"))
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
		WriteError(w, ErrInvalidField.WithDetails("Invalid format. Must be 'csv' or 'json'"))
		return
	}

	migrations, err := h.db.GetCompletedMigrations(ctx)
	if err != nil {
		h.logger.Error("Failed to get migration history for export", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("migration history"))
		return
	}

	if format == formatCSV {
		h.exportMigrationHistoryCSV(w, migrations)
	} else {
		h.exportMigrationHistoryJSON(w, migrations)
	}
}
