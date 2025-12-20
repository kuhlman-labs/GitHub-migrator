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

	// Check if a discovery is already in progress
	activeProgress, err := h.db.GetActiveDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to check for active discovery", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to check discovery status")
		return
	}
	if activeProgress != nil {
		h.sendError(w, http.StatusConflict, fmt.Sprintf("Discovery already in progress (target: %s, started: %s)", activeProgress.Target, activeProgress.StartedAt.Format(time.RFC3339)))
		return
	}

	// Set workers if specified
	if req.Workers > 0 {
		h.collector.SetWorkers(req.Workers)
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
			h.sendError(w, http.StatusConflict, err.Error())
			return
		}
		h.logger.Error("Failed to create discovery progress", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to initialize discovery progress")
		return
	}

	// Create progress tracker
	progressTracker := discovery.NewDBProgressTracker(h.db, h.logger, progress)
	h.collector.SetProgressTracker(progressTracker)

	// Start discovery asynchronously based on type
	if req.EnterpriseSlug != "" {
		go h.runDiscoveryAsync(progress.ID, progressTracker, func(ctx context.Context) error {
			return h.collector.DiscoverEnterpriseRepositories(ctx, req.EnterpriseSlug)
		}, "enterprise", req.EnterpriseSlug)

		h.sendJSON(w, http.StatusAccepted, map[string]interface{}{
			"message":     "Enterprise discovery started",
			"enterprise":  req.EnterpriseSlug,
			"status":      statusInProgress,
			"type":        "enterprise",
			"progress_id": progress.ID,
		})
	} else {
		go h.runDiscoveryAsync(progress.ID, progressTracker, func(ctx context.Context) error {
			return h.collector.DiscoverRepositories(ctx, req.Organization)
		}, "organization", req.Organization)

		h.sendJSON(w, http.StatusAccepted, map[string]interface{}{
			"message":      "Discovery started",
			"organization": req.Organization,
			"status":       statusInProgress,
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

	// Count total repositories discovered
	count, err := h.db.CountRepositories(ctx, nil)
	if err != nil {
		if h.handleContextError(ctx, err, "count repositories", r) {
			return
		}
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

// GetDiscoveryProgress handles GET /api/v1/discovery/progress
// Returns the current or most recent discovery progress
func (h *Handler) GetDiscoveryProgress(w http.ResponseWriter, r *http.Request) {
	progress, err := h.db.GetLatestDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to get discovery progress", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get discovery progress")
		return
	}

	if progress == nil {
		// No discovery has been run yet
		h.sendJSON(w, http.StatusOK, map[string]interface{}{
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
	var req struct {
		Organization string `json:"organization"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Organization == "" {
		h.sendError(w, http.StatusBadRequest, "organization is required")
		return
	}

	if h.collector == nil {
		h.sendError(w, http.StatusServiceUnavailable, "GitHub client not configured")
		return
	}

	// Start discovery asynchronously
	go func() {
		ctx := context.Background()
		if err := h.collector.DiscoverRepositories(ctx, req.Organization); err != nil {
			h.logger.Error("Repository discovery failed", "error", err, "org", req.Organization)
		}
	}()

	h.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":      "Repository discovery started",
		"organization": req.Organization,
		"status":       statusInProgress,
	})
}
