package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/discovery"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// StartDiscovery handles POST /api/v1/discovery/start
func (h *Handler) StartDiscovery(w http.ResponseWriter, r *http.Request) {
	var req StartDiscoveryRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	// Validate that either organization or enterprise is provided, but not both
	if req.Organization == "" && req.EnterpriseSlug == "" {
		WriteError(w, ErrMissingField.WithDetails("Either organization or enterprise_slug is required"))
		return
	}

	if req.Organization != "" && req.EnterpriseSlug != "" {
		WriteError(w, ErrBadRequest.WithDetails("Cannot specify both organization and enterprise_slug"))
		return
	}

	// Get or create collector for this source
	collector, err := h.getCollectorForSource(r.Context(), req.SourceID)
	if err != nil {
		h.logger.Error("Failed to get collector for source", "error", err, "source_id", req.SourceID)
		WriteError(w, ErrClientNotConfigured.WithDetails(err.Error()))
		return
	}

	// Check if a discovery is already in progress
	activeProgress, err := h.db.GetActiveDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to check for active discovery", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("discovery status"))
		return
	}
	if activeProgress != nil {
		WriteError(w, ErrConflict.WithDetails(fmt.Sprintf("Discovery already in progress (target: %s, started: %s)", activeProgress.Target, activeProgress.StartedAt.Format(time.RFC3339))))
		return
	}

	// Set workers: use request value if provided, otherwise use settings value
	workers := req.Workers
	if workers <= 0 {
		// Get workers from settings
		settings, err := h.db.GetSettings(r.Context())
		if err != nil {
			h.logger.Warn("Failed to get settings for workers count, using default", "error", err)
		} else if settings.MigrationWorkers > 0 {
			workers = settings.MigrationWorkers
		}
	}
	if workers > 0 {
		collector.SetWorkers(workers)
		h.logger.Debug("Discovery workers configured", "workers", workers)
	}

	// Set source ID if provided
	if req.SourceID != nil {
		collector.SetSourceID(req.SourceID)
		h.logger.Info("Discovery will associate repos with source", "source_id", *req.SourceID)
	} else {
		collector.SetSourceID(nil)
	}

	// Determine discovery type and target
	var discoveryType, target string
	if req.EnterpriseSlug != "" {
		discoveryType = models.DiscoveryTypeEnterprise
		target = req.EnterpriseSlug
	} else {
		discoveryType = models.DiscoveryTypeOrganization
		target = req.Organization
	}

	// Create progress record
	progress := &models.DiscoveryProgress{
		DiscoveryType: discoveryType,
		Target:        target,
		TotalOrgs:     1, // Default to 1, will be updated for enterprise discovery
	}

	if err := h.db.CreateDiscoveryProgress(progress); err != nil {
		// Check if this is a race condition where another discovery started between our check and create
		if errors.Is(err, storage.ErrDiscoveryInProgress) {
			WriteError(w, ErrConflict.WithDetails(err.Error()))
			return
		}
		h.logger.Error("Failed to create discovery progress", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("discovery progress initialization"))
		return
	}

	// Create progress tracker using the database (which implements storage.DiscoveryStore)
	progressTracker := discovery.NewDBProgressTracker(h.db, h.logger, progress)
	collector.SetProgressTracker(progressTracker)

	// Start discovery asynchronously based on type
	if req.EnterpriseSlug != "" {
		go h.runDiscoveryAsync(progress.ID, progressTracker, func(ctx context.Context) error {
			return collector.DiscoverEnterpriseRepositories(ctx, req.EnterpriseSlug)
		}, "enterprise", req.EnterpriseSlug, req.SourceID)

		h.sendJSON(w, http.StatusAccepted, map[string]any{
			"message":     "Enterprise discovery started",
			"enterprise":  req.EnterpriseSlug,
			"status":      models.DiscoveryStatusInProgress,
			"type":        "enterprise",
			"progress_id": progress.ID,
		})
	} else {
		go h.runDiscoveryAsync(progress.ID, progressTracker, func(ctx context.Context) error {
			return collector.DiscoverRepositories(ctx, req.Organization)
		}, "organization", req.Organization, req.SourceID)

		h.sendJSON(w, http.StatusAccepted, map[string]any{
			"message":      "Discovery started",
			"organization": req.Organization,
			"status":       models.DiscoveryStatusInProgress,
			"type":         "organization",
			"progress_id":  progress.ID,
		})
	}
}

// runDiscoveryAsync executes a discovery operation asynchronously and updates progress
func (h *Handler) runDiscoveryAsync(progressID int64, tracker *discovery.DBProgressTracker, discoverFn func(context.Context) error, discoveryType, target string, sourceID *int64) {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Register cancel function for this discovery
	h.discoveryMu.Lock()
	h.discoveryCancel[progressID] = cancel
	h.discoveryMu.Unlock()

	// Clean up when done
	defer func() {
		h.discoveryMu.Lock()
		delete(h.discoveryCancel, progressID)
		h.discoveryMu.Unlock()
		cancel()
		tracker.Flush()
	}()

	if err := discoverFn(ctx); err != nil {
		// Check if this was a cancellation
		if ctx.Err() == context.Canceled {
			h.logger.Info("Discovery cancelled", "type", discoveryType, "target", target)
			if dbErr := h.db.MarkDiscoveryCancelled(progressID); dbErr != nil {
				h.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
			}
			return
		}

		h.logger.Error("Discovery failed", "error", err, "type", discoveryType, "target", target)
		if dbErr := h.db.MarkDiscoveryFailed(progressID, err.Error()); dbErr != nil {
			h.logger.Error("Failed to mark discovery as failed", "error", dbErr)
		}
	} else {
		// Check if cancelled even on success (rare but possible)
		if ctx.Err() == context.Canceled {
			h.logger.Info("Discovery cancelled", "type", discoveryType, "target", target)
			if dbErr := h.db.MarkDiscoveryCancelled(progressID); dbErr != nil {
				h.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
			}
			return
		}

		if dbErr := h.db.MarkDiscoveryComplete(progressID); dbErr != nil {
			h.logger.Error("Failed to mark discovery as complete", "error", dbErr)
		}

		// Update source repository count and last sync time if source ID is provided
		if sourceID != nil {
			if err := h.db.UpdateSourceRepositoryCount(ctx, *sourceID); err != nil {
				h.logger.Error("Failed to update source repository count",
					"error", err,
					"source_id", *sourceID)
			} else {
				h.logger.Info("Updated source repository count after discovery",
					"source_id", *sourceID)
			}
		}
	}
}

// DiscoveryStatus handles GET /api/v1/discovery/status
func (h *Handler) DiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the latest discovery progress to determine actual status
	progress, err := h.db.GetLatestDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to get discovery progress", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("discovery status"))
		return
	}

	// Count total repositories discovered
	count, err := h.db.CountRepositories(ctx, nil)
	if err != nil {
		if h.handleContextError(ctx, err, "count repositories", r) {
			return
		}
		h.logger.Error("Failed to count repositories", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("discovery status"))
		return
	}

	// Build response based on actual discovery state
	response := map[string]any{
		"repositories_found": count,
	}

	if progress == nil {
		// No discovery has been run yet
		response["status"] = "none"
		response["message"] = "No discovery has been run yet"
	} else {
		response["status"] = progress.Status
		response["target"] = progress.Target
		response["discovery_type"] = progress.DiscoveryType
		response["started_at"] = progress.StartedAt.Format(time.RFC3339)

		if progress.Status == models.DiscoveryStatusCompleted && progress.CompletedAt != nil {
			response["completed_at"] = progress.CompletedAt.Format(time.RFC3339)
		}
		if progress.Status == models.DiscoveryStatusFailed && progress.LastError != nil && *progress.LastError != "" {
			response["error"] = *progress.LastError
		}
		if progress.Status == models.DiscoveryStatusInProgress {
			response["processed_repos"] = progress.ProcessedRepos
			response["total_repos"] = progress.TotalRepos
			response["phase"] = progress.Phase
		}
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetDiscoveryProgress handles GET /api/v1/discovery/progress
// Returns the current or most recent discovery progress
func (h *Handler) GetDiscoveryProgress(w http.ResponseWriter, r *http.Request) {
	progress, err := h.db.GetLatestDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to get discovery progress", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("discovery progress"))
		return
	}

	if progress == nil {
		// No discovery has been run yet
		h.sendJSON(w, http.StatusOK, map[string]any{
			"status":  "none",
			"message": "No discovery has been run yet",
		})
		return
	}

	h.sendJSON(w, http.StatusOK, progress)
}

// CancelDiscovery handles POST /api/v1/discovery/cancel
// Cancels the currently running discovery, allowing workers to finish their current repo
func (h *Handler) CancelDiscovery(w http.ResponseWriter, r *http.Request) {
	// Get active discovery progress
	progress, err := h.db.GetActiveDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to get active discovery progress", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("discovery progress"))
		return
	}

	if progress == nil {
		WriteError(w, ErrNotFound.WithDetails("No active discovery to cancel"))
		return
	}

	// Look up the cancel function
	h.discoveryMu.RLock()
	cancel, exists := h.discoveryCancel[progress.ID]
	h.discoveryMu.RUnlock()

	if !exists {
		// Discovery might have just finished or wasn't started by this handler instance
		WriteError(w, ErrNotFound.WithDetails("Discovery cancel function not found - discovery may have already completed"))
		return
	}

	// Update phase to show cancelling in progress
	if err := h.db.UpdateDiscoveryPhase(progress.ID, models.PhaseCancelling); err != nil {
		h.logger.Warn("Failed to update discovery phase to cancelling", "error", err)
	}

	// Trigger cancellation - this will cause workers to stop picking up new repos
	h.logger.Info("Cancelling discovery", "progress_id", progress.ID, "target", progress.Target)
	cancel()

	h.sendJSON(w, http.StatusOK, map[string]any{
		"message":     "Discovery cancellation initiated",
		"progress_id": progress.ID,
		"target":      progress.Target,
		"status":      "cancelling",
	})
}

// DiscoverRepositories handles POST /api/v1/repositories/discover
// Discovers repositories for a single organization (standalone, repos-only discovery)
func (h *Handler) DiscoverRepositories(w http.ResponseWriter, r *http.Request) {
	var req StartProfilingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.Organization == "" {
		WriteError(w, ErrMissingField.WithField("organization"))
		return
	}

	// Get or create collector for this source
	collector, err := h.getCollectorForSource(r.Context(), req.SourceID)
	if err != nil {
		h.logger.Error("Failed to get collector for source", "error", err, "source_id", req.SourceID)
		WriteError(w, ErrClientNotConfigured.WithDetails(err.Error()))
		return
	}

	// Set source ID if provided
	if req.SourceID != nil {
		collector.SetSourceID(req.SourceID)
		h.logger.Info("Discovery will associate repos with source", "source_id", *req.SourceID)
	} else {
		collector.SetSourceID(nil)
	}

	// Start discovery asynchronously
	go func() {
		ctx := context.Background()
		if err := collector.DiscoverRepositories(ctx, req.Organization); err != nil {
			h.logger.Error("Repository discovery failed", "error", err, "org", req.Organization)
		}
	}()

	h.sendJSON(w, http.StatusAccepted, map[string]any{
		"message":      "Repository discovery started",
		"organization": req.Organization,
		"status":       models.DiscoveryStatusInProgress,
		"source_id":    req.SourceID,
	})
}

// StartADODiscoveryDynamic handles POST /api/v1/ado/discover when no static ADO handler is configured.
// It creates a dynamic ADO collector from the database source configuration.
func (h *Handler) StartADODiscoveryDynamic(w http.ResponseWriter, r *http.Request) {
	// If we have a pre-configured ADO handler, delegate to it
	if h.adoHandler != nil {
		h.adoHandler.StartADODiscovery(w, r)
		return
	}

	req, err := h.parseAndValidateADORequest(r)
	if err != nil {
		h.handleADORequestError(w, err)
		return
	}

	// Check for existing in-progress discovery
	if h.hasActiveDiscovery(w) {
		return
	}

	// Get ADO collector for this source
	adoCollector, err := h.getADOCollector(r.Context(), w, req)
	if err != nil {
		return
	}

	// Create and start discovery
	progress, progressTracker, err := h.setupADODiscoveryProgress(req)
	if err != nil {
		h.handleProgressError(w, err)
		return
	}
	adoCollector.SetProgressTracker(progressTracker)

	// Start discovery asynchronously based on scope
	if len(req.Projects) == 0 {
		h.startDynamicADOOrgDiscovery(w, req, adoCollector, progress, progressTracker)
	} else {
		h.startDynamicADOProjectDiscovery(w, req, adoCollector, progress, progressTracker)
	}
}

// parseAndValidateADORequest parses and validates an ADO discovery request
func (h *Handler) parseAndValidateADORequest(r *http.Request) (*StartADODiscoveryRequest, error) {
	var req StartADODiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	if req.Organization == "" {
		return nil, fmt.Errorf("missing: organization")
	}
	if req.SourceID == nil {
		return nil, fmt.Errorf("missing: source_id")
	}
	if req.Workers <= 0 {
		req.Workers = 5
	}
	return &req, nil
}

// handleADORequestError handles request parsing/validation errors
func (h *Handler) handleADORequestError(w http.ResponseWriter, err error) {
	errStr := err.Error()
	if strings.HasPrefix(errStr, "json:") {
		WriteError(w, ErrInvalidJSON)
	} else if strings.HasPrefix(errStr, "missing: organization") {
		WriteError(w, ErrMissingField.WithField("organization"))
	} else if strings.HasPrefix(errStr, "missing: source_id") {
		WriteError(w, ErrMissingField.WithField("source_id").WithDetails("source_id is required when no static ADO configuration is present"))
	}
}

// hasActiveDiscovery checks if there's an active discovery and writes error if so
func (h *Handler) hasActiveDiscovery(w http.ResponseWriter) bool {
	existingProgress, err := h.db.GetActiveDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to check for active discovery", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("discovery status"))
		return true
	}
	if existingProgress != nil {
		WriteError(w, ErrConflict.WithDetails("Another discovery is already in progress"))
		return true
	}
	return false
}

// getADOCollector gets the ADO collector for the source
func (h *Handler) getADOCollector(ctx context.Context, w http.ResponseWriter, req *StartADODiscoveryRequest) (*discovery.ADOCollector, error) {
	adoCollector, _, err := h.getADOCollectorForSource(ctx, req.SourceID)
	if err != nil {
		h.logger.Error("Failed to get ADO collector for source", "error", err, "source_id", req.SourceID)
		WriteError(w, ErrClientNotConfigured.WithDetails(err.Error()))
		return nil, err
	}
	// Set workers: use request value if provided, otherwise use settings value
	workers := req.Workers
	if workers <= 0 {
		// Get workers from settings
		settings, err := h.db.GetSettings(ctx)
		if err != nil {
			h.logger.Warn("Failed to get settings for workers count, using default", "error", err)
		} else if settings.MigrationWorkers > 0 {
			workers = settings.MigrationWorkers
		}
	}
	if workers > 0 {
		adoCollector.SetWorkers(workers)
		h.logger.Debug("ADO discovery workers configured", "workers", workers)
	}
	return adoCollector, nil
}

// setupADODiscoveryProgress creates progress record and tracker
func (h *Handler) setupADODiscoveryProgress(req *StartADODiscoveryRequest) (*models.DiscoveryProgress, *discovery.DBProgressTracker, error) {
	var discoveryType, target string
	var totalOrgs int
	if len(req.Projects) == 0 {
		discoveryType = models.DiscoveryTypeADOOrganization
		target = req.Organization
		totalOrgs = 1
	} else {
		discoveryType = models.DiscoveryTypeADOProject
		target = req.Organization + "/" + strings.Join(req.Projects, ",")
		totalOrgs = len(req.Projects)
	}

	progress := &models.DiscoveryProgress{
		DiscoveryType: discoveryType,
		Target:        target,
		TotalOrgs:     totalOrgs,
	}

	if err := h.db.CreateDiscoveryProgress(progress); err != nil {
		return nil, nil, err
	}

	tracker := discovery.NewDBProgressTracker(h.db, h.logger, progress)
	return progress, tracker, nil
}

// handleProgressError handles progress creation errors
func (h *Handler) handleProgressError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrDiscoveryInProgress) {
		WriteError(w, ErrConflict.WithDetails(err.Error()))
		return
	}
	h.logger.Error("Failed to create discovery progress", "error", err)
	WriteError(w, ErrDatabaseUpdate.WithDetails("discovery progress initialization"))
}

// startDynamicADOOrgDiscovery starts organization discovery with dynamic collector
func (h *Handler) startDynamicADOOrgDiscovery(w http.ResponseWriter, req *StartADODiscoveryRequest, collector *discovery.ADOCollector, progress *models.DiscoveryProgress, tracker *discovery.DBProgressTracker) {
	h.logger.Info("Starting ADO organization discovery (dynamic)",
		"organization", req.Organization,
		"workers", req.Workers,
		"source_id", *req.SourceID,
		"progress_id", progress.ID)

	go h.runDynamicADOOrgDiscovery(req.Organization, *req.SourceID, progress.ID, collector, tracker)

	h.sendJSON(w, http.StatusAccepted, map[string]any{
		"message":      "ADO organization discovery started",
		"organization": req.Organization,
		"type":         "organization",
		"source_id":    *req.SourceID,
		"progress_id":  progress.ID,
	})
}

// runDynamicADOOrgDiscovery executes org discovery in background
func (h *Handler) runDynamicADOOrgDiscovery(organization string, sourceID, progressID int64, collector *discovery.ADOCollector, tracker *discovery.DBProgressTracker) {
	ctx := context.Background()
	defer tracker.Flush() // Ensure pending progress updates are written to the database

	if err := collector.DiscoverADOOrganization(ctx, organization); err != nil {
		h.logger.Error("ADO organization discovery failed", "organization", organization, "error", err)
		if markErr := h.db.MarkDiscoveryFailed(progressID, err.Error()); markErr != nil {
			h.logger.Error("Failed to mark discovery as failed", "error", markErr)
		}
		return
	}

	h.logger.Info("ADO organization discovery completed", "organization", organization)
	if markErr := h.db.MarkDiscoveryComplete(progressID); markErr != nil {
		h.logger.Error("Failed to mark discovery as complete", "error", markErr)
	}
	h.updateSourceRepoCount(ctx, sourceID)
}

// startDynamicADOProjectDiscovery starts project discovery with dynamic collector
func (h *Handler) startDynamicADOProjectDiscovery(w http.ResponseWriter, req *StartADODiscoveryRequest, collector *discovery.ADOCollector, progress *models.DiscoveryProgress, tracker *discovery.DBProgressTracker) {
	h.logger.Info("Starting ADO project discovery (dynamic)",
		"organization", req.Organization,
		"projects", req.Projects,
		"workers", req.Workers,
		"source_id", *req.SourceID,
		"progress_id", progress.ID)

	go h.runDynamicADOProjectDiscovery(req.Organization, req.Projects, *req.SourceID, progress.ID, collector, tracker)

	h.sendJSON(w, http.StatusAccepted, map[string]any{
		"message":      "ADO project discovery started",
		"organization": req.Organization,
		"projects":     req.Projects,
		"type":         "project",
		"source_id":    *req.SourceID,
		"progress_id":  progress.ID,
	})
}

// runDynamicADOProjectDiscovery executes project discovery in background
func (h *Handler) runDynamicADOProjectDiscovery(organization string, projects []string, sourceID, progressID int64, collector *discovery.ADOCollector, tracker *discovery.DBProgressTracker) {
	ctx := context.Background()
	defer tracker.Flush() // Ensure pending progress updates are written to the database

	var lastErr error
	for i, project := range projects {
		tracker.StartOrg(project, i)
		if err := collector.DiscoverADOProject(ctx, organization, project); err != nil {
			h.logger.Error("Failed to discover ADO project", "organization", organization, "project", project, "error", err)
			tracker.RecordError(err)
			lastErr = err
		} else {
			h.logger.Info("ADO project discovery completed", "organization", organization, "project", project)
		}
		tracker.CompleteOrg(project, 0)
	}

	if lastErr != nil {
		if markErr := h.db.MarkDiscoveryFailed(progressID, lastErr.Error()); markErr != nil {
			h.logger.Error("Failed to mark discovery as failed", "error", markErr)
		}
	} else {
		if markErr := h.db.MarkDiscoveryComplete(progressID); markErr != nil {
			h.logger.Error("Failed to mark discovery as complete", "error", markErr)
		}
	}
	h.updateSourceRepoCount(ctx, sourceID)
}

// updateSourceRepoCount updates the source repository count after discovery
func (h *Handler) updateSourceRepoCount(ctx context.Context, sourceID int64) {
	if err := h.db.UpdateSourceRepositoryCount(ctx, sourceID); err != nil {
		h.logger.Error("Failed to update source repository count", "error", err, "source_id", sourceID)
	} else {
		h.logger.Info("Updated source repository count after ADO discovery", "source_id", sourceID)
	}
}

// ADODiscoveryStatusDynamic handles GET /api/v1/ado/discovery/status when no static ADO handler is configured.
func (h *Handler) ADODiscoveryStatusDynamic(w http.ResponseWriter, r *http.Request) {
	// If we have a pre-configured ADO handler, delegate to it
	if h.adoHandler != nil {
		h.adoHandler.ADODiscoveryStatus(w, r)
		return
	}

	ctx := r.Context()
	organization := r.URL.Query().Get("organization")

	// Count ADO repositories from the database
	filters := map[string]any{
		"source": "azuredevops",
	}
	if organization != "" {
		filters["ado_organization"] = organization
	}

	repos, err := h.db.ListRepositories(ctx, filters)
	if err != nil {
		WriteError(w, ErrDatabaseFetch.WithDetails("ADO repositories"))
		return
	}

	// Aggregate stats
	totalRepos := len(repos)
	pendingCount := 0
	completedCount := 0
	failedCount := 0

	for _, repo := range repos {
		switch repo.Status {
		case string(models.StatusPending):
			pendingCount++
		case string(models.StatusComplete), string(models.StatusMigrationComplete):
			completedCount++
		case string(models.StatusMigrationFailed):
			failedCount++
		}
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"organization":    organization,
		"total_repos":     totalRepos,
		"pending_count":   pendingCount,
		"completed_count": completedCount,
		"failed_count":    failedCount,
		"status":          "completed", // Discovery is synchronous, so it's always completed when we query
	})
}

// ListADOProjectsDynamic handles GET /api/v1/ado/projects when no static ADO handler is configured.
func (h *Handler) ListADOProjectsDynamic(w http.ResponseWriter, r *http.Request) {
	// If we have a pre-configured ADO handler, delegate to it
	if h.adoHandler != nil {
		h.adoHandler.ListADOProjects(w, r)
		return
	}

	ctx := r.Context()
	organization := r.URL.Query().Get("organization")

	// Query projects from database
	projects, err := h.db.GetADOProjects(ctx, organization)
	if err != nil {
		WriteError(w, ErrDatabaseFetch.WithDetails("ADO projects"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
		"total":    len(projects),
	})
}

// GetADOProjectDynamic handles GET /api/v1/ado/projects/{organization}/{project} when no static ADO handler is configured.
func (h *Handler) GetADOProjectDynamic(w http.ResponseWriter, r *http.Request) {
	// If we have a pre-configured ADO handler, delegate to it
	if h.adoHandler != nil {
		h.adoHandler.GetADOProject(w, r)
		return
	}

	ctx := r.Context()

	// Extract organization and project from URL path
	org := r.PathValue("organization")
	project := r.PathValue("project")

	if org == "" || project == "" {
		WriteError(w, ErrMissingField.WithDetails("organization and project are required"))
		return
	}

	// Get project from database
	projectData, err := h.db.GetADOProject(ctx, org, project)
	if err != nil {
		WriteError(w, ErrDatabaseFetch.WithDetails("ADO project"))
		return
	}

	if projectData == nil {
		WriteError(w, ErrNotFound.WithDetails(fmt.Sprintf("Project %s/%s not found", org, project)))
		return
	}

	h.sendJSON(w, http.StatusOK, projectData)
}
