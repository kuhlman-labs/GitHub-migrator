package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// AnalyticsHandler handles analytics-related HTTP requests.
// This is a focused handler that separates analytics operations from other concerns.
type AnalyticsHandler struct {
	analyticsStore storage.AnalyticsStore
	repoStore      storage.RepositoryStore
	batchStore     storage.BatchStore
	logger         *slog.Logger
}

// NewAnalyticsHandler creates a new AnalyticsHandler with focused dependencies.
func NewAnalyticsHandler(
	analyticsStore storage.AnalyticsStore,
	repoStore storage.RepositoryStore,
	batchStore storage.BatchStore,
	logger *slog.Logger,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsStore: analyticsStore,
		repoStore:      repoStore,
		batchStore:     batchStore,
		logger:         logger,
	}
}

// GetAnalyticsSummary handles GET /api/v1/analytics/summary
func (h *AnalyticsHandler) GetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filter parameters
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	// Get status distribution
	statusStats, err := h.analyticsStore.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get status stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get analytics summary")
		return
	}

	// Get total count (excluding wont_migrate to match main analytics handler)
	total := 0
	for status, count := range statusStats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	// Get organization breakdown
	orgStats, err := h.analyticsStore.GetOrganizationStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get organization stats", "error", err)
		// Continue without org stats
		orgStats = nil
	}

	// Get complexity distribution
	complexityDist, err := h.analyticsStore.GetComplexityDistribution(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get complexity distribution", "error", err)
		// Continue without complexity stats
		complexityDist = nil
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"total_repositories":      total,
		"status_distribution":     statusStats,
		"organization_breakdown":  orgStats,
		"complexity_distribution": complexityDist,
	})
}

// GetMigrationProgress handles GET /api/v1/analytics/progress
func (h *AnalyticsHandler) GetMigrationProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filter parameters
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	// Get status stats
	statusStats, err := h.analyticsStore.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get status stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get migration progress")
		return
	}

	// Calculate totals using helper function
	counts := categorizeStatuses(statusStats)

	// Calculate percentage
	var progressPercent float64
	if counts.Total > 0 {
		progressPercent = float64(counts.Completed) / float64(counts.Total) * 100
	}

	// Get time series data
	timeSeries, err := h.analyticsStore.GetMigrationTimeSeries(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get time series data", "error", err)
		// Continue without time series
		timeSeries = nil
	}

	// Get velocity
	velocity, err := h.analyticsStore.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 7)
	if err != nil {
		h.logger.Error("Failed to get velocity", "error", err)
		// Continue without velocity
		velocity = nil
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"total":            counts.Total,
		"completed":        counts.Completed,
		"in_progress":      counts.InProgress,
		"pending":          counts.Pending,
		"failed":           counts.Failed,
		"progress_percent": progressPercent,
		"time_series":      timeSeries,
		"velocity":         velocity,
	})
}

// GetExecutiveReport handles GET /api/v1/analytics/executive-report
func (h *AnalyticsHandler) GetExecutiveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filter parameters
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	// Get status stats
	statusStats, err := h.analyticsStore.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get status stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get executive report")
		return
	}

	// Calculate totals using helper function
	counts := categorizeStatuses(statusStats)

	// Calculate percentages
	var progressPercent, successRate float64
	if counts.Total > 0 {
		progressPercent = float64(counts.Completed) / float64(counts.Total) * 100
	}
	if counts.Completed+counts.Failed > 0 {
		successRate = float64(counts.Completed) / float64(counts.Completed+counts.Failed) * 100
	}

	// Get organization stats
	orgStats, err := h.analyticsStore.GetOrganizationStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get organization stats", "error", err)
		orgStats = nil
	}

	// Get velocity
	velocity, err := h.analyticsStore.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 7)
	if err != nil {
		h.logger.Error("Failed to get velocity", "error", err)
		velocity = nil
	}

	// Get average migration time
	avgTime, err := h.analyticsStore.GetAverageMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get average migration time", "error", err)
		avgTime = 0
	}

	// Get median migration time
	medianTime, err := h.analyticsStore.GetMedianMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get median migration time", "error", err)
		medianTime = 0
	}

	// Estimate completion date based on velocity
	var estimatedCompletionDays int
	if velocity != nil && velocity.ReposPerDay > 0 {
		remaining := counts.Pending + counts.InProgress
		estimatedCompletionDays = int(float64(remaining) / velocity.ReposPerDay)
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"summary": map[string]interface{}{
			"total":            counts.Total,
			"completed":        counts.Completed,
			"in_progress":      counts.InProgress,
			"pending":          counts.Pending,
			"failed":           counts.Failed,
			"progress_percent": progressPercent,
			"success_rate":     successRate,
		},
		"organization_breakdown": orgStats,
		"performance": map[string]interface{}{
			"velocity":                 velocity,
			"average_migration_time":   avgTime,
			"median_migration_time":    medianTime,
			"estimated_days_remaining": estimatedCompletionDays,
		},
	})
}

// Helper methods

// statusCounts holds categorized status counts for migration progress tracking
type statusCounts struct {
	Total      int
	Completed  int
	InProgress int
	Failed     int
	Pending    int
}

// categorizeStatuses categorizes repository statuses into summary counts
// Note: StatusWontMigrate is excluded from the total to match the behavior
// of the main Handler implementation - these repos are out of migration scope.
func categorizeStatuses(statusStats map[string]int) statusCounts {
	var counts statusCounts
	for status, count := range statusStats {
		// Exclude wont_migrate from total - these repos are out of migration scope
		if models.MigrationStatus(status) == models.StatusWontMigrate {
			continue
		}
		counts.Total += count
		switch models.MigrationStatus(status) {
		case models.StatusComplete, models.StatusMigrationComplete:
			counts.Completed += count
		case models.StatusPreMigration, models.StatusArchiveGenerating, models.StatusQueuedForMigration, models.StatusMigratingContent, models.StatusPostMigration:
			counts.InProgress += count
		case models.StatusPending, models.StatusDryRunQueued, models.StatusDryRunInProgress, models.StatusDryRunComplete:
			counts.Pending += count
		case models.StatusMigrationFailed, models.StatusDryRunFailed, models.StatusRolledBack:
			counts.Failed += count
		}
	}
	return counts
}

// sendJSON sends a JSON response
func (h *AnalyticsHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// sendError sends an error response
func (h *AnalyticsHandler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, map[string]string{"error": message})
}
