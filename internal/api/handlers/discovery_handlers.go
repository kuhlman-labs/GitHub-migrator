package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	collector, err := h.getCollectorForSource(req.SourceID)
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

	// Set workers if specified
	if req.Workers > 0 {
		collector.SetWorkers(req.Workers)
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
		}, "enterprise", req.EnterpriseSlug)

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
		}, "organization", req.Organization)

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
func (h *Handler) runDiscoveryAsync(progressID int64, tracker *discovery.DBProgressTracker, discoverFn func(context.Context) error, discoveryType, target string) {
	ctx := context.Background()
	if err := discoverFn(ctx); err != nil {
		h.logger.Error("Discovery failed", "error", err, "type", discoveryType, "target", target)
		if dbErr := h.db.MarkDiscoveryFailed(progressID, err.Error()); dbErr != nil {
			h.logger.Error("Failed to mark discovery as failed", "error", dbErr)
		}
	} else {
		if dbErr := h.db.MarkDiscoveryComplete(progressID); dbErr != nil {
			h.logger.Error("Failed to mark discovery as complete", "error", dbErr)
		}
	}
	tracker.Flush()
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
	collector, err := h.getCollectorForSource(req.SourceID)
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
