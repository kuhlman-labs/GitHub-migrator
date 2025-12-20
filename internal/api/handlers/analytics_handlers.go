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
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// GetAnalyticsSummary handles GET /api/v1/analytics/summary
//
//nolint:gocyclo // Complexity is inherent to analytics aggregation
func (h *Handler) GetAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get repository stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("analytics"))
		return
	}

	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]
	failed := stats[string(models.StatusMigrationFailed)] + stats[string(models.StatusDryRunFailed)] + stats[string(models.StatusRolledBack)]
	pending := stats[string(models.StatusPending)] + stats[string(models.StatusDryRunQueued)] + stats[string(models.StatusDryRunInProgress)] + stats[string(models.StatusDryRunComplete)]
	inProgress := stats[string(models.StatusPreMigration)] + stats[string(models.StatusArchiveGenerating)] + stats[string(models.StatusQueuedForMigration)] + stats[string(models.StatusMigratingContent)] + stats[string(models.StatusPostMigration)]

	successRate := 0.0
	if migrated+failed > 0 {
		successRate = float64(migrated) / float64(migrated+failed) * 100
	}

	complexityDistribution, err := h.db.GetComplexityDistribution(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get complexity distribution", "error", err)
		complexityDistribution = []*storage.ComplexityDistribution{}
	}

	migrationVelocity, err := h.db.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 30)
	if err != nil {
		h.logger.Error("Failed to get migration velocity", "error", err)
		migrationVelocity = &storage.MigrationVelocity{}
	}

	migrationTimeSeries, err := h.db.GetMigrationTimeSeries(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get migration time series", "error", err)
		migrationTimeSeries = []*storage.MigrationTimeSeriesPoint{}
	}

	avgMigrationTime, _ := h.db.GetAverageMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	medianMigrationTime, _ := h.db.GetMedianMigrationTime(ctx, orgFilter, projectFilter, batchFilter)

	var estimatedCompletionDate *string
	remaining := total - migrated
	if remaining > 0 && migrationVelocity.ReposPerDay > 0 {
		daysRemaining := float64(remaining) / migrationVelocity.ReposPerDay
		completionDate := time.Now().Add(time.Duration(daysRemaining*24) * time.Hour)
		dateStr := completionDate.Format("2006-01-02")
		estimatedCompletionDate = &dateStr
	}

	orgStats, _ := h.db.GetOrganizationStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	var projectStats []*storage.OrganizationStats
	if h.sourceType == models.SourceTypeAzureDevOps {
		projectStats, _ = h.db.GetProjectStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	}

	sizeDistribution, _ := h.db.GetSizeDistributionFiltered(ctx, orgFilter, projectFilter, batchFilter)
	featureStats, _ := h.db.GetFeatureStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)

	var migrationCompletionStats []*storage.MigrationCompletionStats
	if h.sourceType == models.SourceTypeAzureDevOps {
		migrationCompletionStats, _ = h.db.GetMigrationCompletionStatsByProjectFiltered(ctx, orgFilter, projectFilter, batchFilter)
	} else {
		migrationCompletionStats, _ = h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, projectFilter, batchFilter)
	}

	summary := map[string]interface{}{
		"total_repositories":         total,
		"migrated_count":             migrated,
		"failed_count":               failed,
		"in_progress_count":          inProgress,
		"pending_count":              pending,
		"success_rate":               successRate,
		"status_breakdown":           stats,
		"complexity_distribution":    complexityDistribution,
		"migration_velocity":         migrationVelocity,
		"migration_time_series":      migrationTimeSeries,
		"average_migration_time":     avgMigrationTime,
		"median_migration_time":      medianMigrationTime,
		"estimated_completion_date":  estimatedCompletionDate,
		"organization_stats":         orgStats,
		"project_stats":              projectStats,
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
		if h.handleContextError(ctx, err, "get repository stats", r) {
			return
		}
		h.logger.Error("Failed to get repository stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("progress"))
		return
	}

	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"total":            total,
		"status_breakdown": stats,
	})
}

// GetExecutiveReport handles GET /api/v1/analytics/executive-report
func (h *Handler) GetExecutiveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	stats, err := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if err != nil {
		h.logger.Error("Failed to get repository stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("analytics"))
		return
	}

	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]
	failed := stats[string(models.StatusMigrationFailed)] + stats[string(models.StatusDryRunFailed)] + stats[string(models.StatusRolledBack)]
	pending := stats[string(models.StatusPending)] + stats[string(models.StatusDryRunQueued)] + stats[string(models.StatusDryRunInProgress)] + stats[string(models.StatusDryRunComplete)]
	inProgress := stats[string(models.StatusPreMigration)] + stats[string(models.StatusArchiveGenerating)] + stats[string(models.StatusQueuedForMigration)] + stats[string(models.StatusMigratingContent)] + stats[string(models.StatusPostMigration)]

	successRate := 0.0
	if migrated+failed > 0 {
		successRate = float64(migrated) / float64(migrated+failed) * 100
	}

	migrationVelocity, _ := h.db.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 30)
	if migrationVelocity == nil {
		migrationVelocity = &storage.MigrationVelocity{}
	}
	migrationTimeSeries, _ := h.db.GetMigrationTimeSeries(ctx, orgFilter, projectFilter, batchFilter)
	avgMigrationTime, _ := h.db.GetAverageMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	medianMigrationTime, _ := h.db.GetMedianMigrationTime(ctx, orgFilter, projectFilter, batchFilter)

	var estimatedCompletionDate *string
	var daysRemaining int
	remaining := total - migrated
	if remaining > 0 && migrationVelocity.ReposPerDay > 0 {
		daysRemainingFloat := float64(remaining) / migrationVelocity.ReposPerDay
		daysRemaining = int(daysRemainingFloat)
		completionDate := time.Now().Add(time.Duration(daysRemainingFloat*24) * time.Hour)
		dateStr := completionDate.Format("2006-01-02")
		estimatedCompletionDate = &dateStr
	}

	var migrationCompletionStats []*storage.MigrationCompletionStats
	if h.sourceType == models.SourceTypeAzureDevOps {
		migrationCompletionStats, _ = h.db.GetMigrationCompletionStatsByProjectFiltered(ctx, orgFilter, projectFilter, batchFilter)
	} else {
		migrationCompletionStats, _ = h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, projectFilter, batchFilter)
	}

	complexityDistribution, _ := h.db.GetComplexityDistribution(ctx, orgFilter, projectFilter, batchFilter)
	sizeDistribution, _ := h.db.GetSizeDistributionFiltered(ctx, orgFilter, projectFilter, batchFilter)
	featureStats, _ := h.db.GetFeatureStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if featureStats == nil {
		featureStats = &storage.FeatureStats{}
	}

	highComplexityPending := 0
	veryComplexCount := 0
	veryLargePending := 0
	for _, dist := range complexityDistribution {
		if dist.Category == models.ComplexityComplex || dist.Category == models.ComplexityVeryComplex {
			highComplexityPending += dist.Count
		}
		if dist.Category == models.ComplexityVeryComplex {
			veryComplexCount += dist.Count
		}
	}
	for _, dist := range sizeDistribution {
		if dist.Category == models.SizeCategoryVeryLarge {
			veryLargePending += dist.Count
		}
	}

	batches, _ := h.db.ListBatches(ctx)
	completedBatches, inProgressBatches, pendingBatches := 0, 0, 0
	for _, batch := range batches {
		switch batch.Status {
		case models.BatchStatusCompleted, models.BatchStatusCompletedWithErrors:
			completedBatches++
		case models.BatchStatusInProgress:
			inProgressBatches++
		case models.BatchStatusPending, models.BatchStatusReady:
			pendingBatches++
		}
	}

	var firstMigrationDate *string
	if len(migrationTimeSeries) > 0 {
		firstMigrationDate = &migrationTimeSeries[0].Date
	}

	completionRate := 0.0
	if total > 0 {
		completionRate = float64(migrated) / float64(total) * 100
	}

	report := map[string]interface{}{
		"source_type": h.sourceType,
		"report_metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"filters":      map[string]interface{}{"organization": orgFilter, "project": projectFilter, "batch_id": batchFilter},
		},
		"discovery_data": map[string]interface{}{
			"overview":                 map[string]interface{}{"total_repositories": total, "source_type": h.sourceType},
			"features":                 featureStats,
			"complexity":               map[string]interface{}{"distribution": complexityDistribution, "high_complexity_count": highComplexityPending, "very_complex_count": veryComplexCount},
			"size":                     map[string]interface{}{"distribution": sizeDistribution, "very_large_count": veryLargePending},
			"organizational_breakdown": migrationCompletionStats,
		},
		"migration_analytics": map[string]interface{}{
			"summary": map[string]interface{}{
				"total_repositories":        total,
				"migrated_count":            migrated,
				"in_progress_count":         inProgress,
				"pending_count":             pending,
				"failed_count":              failed,
				"completion_percentage":     completionRate,
				"success_rate":              successRate,
				"estimated_completion_date": estimatedCompletionDate,
				"days_remaining":            daysRemaining,
				"first_migration_date":      firstMigrationDate,
			},
			"status_breakdown": stats,
			"velocity":         map[string]interface{}{"repos_per_day": migrationVelocity.ReposPerDay, "repos_per_week": migrationVelocity.ReposPerWeek, "average_duration_sec": avgMigrationTime, "median_duration_sec": medianMigrationTime, "trend": migrationTimeSeries},
			"batches":          map[string]interface{}{"total": len(batches), "completed": completedBatches, "in_progress": inProgressBatches, "pending": pendingBatches},
			"risk_factors":     map[string]interface{}{"high_complexity_pending": highComplexityPending, "very_large_pending": veryLargePending, "failed_migrations": failed},
		},
	}

	if h.sourceType == models.SourceTypeAzureDevOps {
		if discoveryData, ok := report["discovery_data"].(map[string]interface{}); ok {
			discoveryData["ado_specific_risks"] = map[string]interface{}{
				"tfvc_repos":                   featureStats.ADOTFVCCount,
				"classic_pipelines":            featureStats.ADOHasClassicPipelines,
				"repos_with_active_work_items": featureStats.ADOHasWorkItems,
				"repos_with_wikis":             featureStats.ADOHasWiki,
				"repos_with_test_plans":        featureStats.ADOHasTestPlans,
				"repos_with_package_feeds":     featureStats.ADOHasPackageFeeds,
			}
		}
	}

	h.sendJSON(w, http.StatusOK, report)
}

// ExportExecutiveReport handles GET /api/v1/analytics/executive-report/export
func (h *Handler) ExportExecutiveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	format := r.URL.Query().Get("format")
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	if format != formatCSV && format != formatJSON {
		WriteError(w, ErrInvalidField.WithDetails("Invalid format. Must be 'csv' or 'json'"))
		return
	}

	stats, _ := h.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, projectFilter, batchFilter)

	total := 0
	for status, count := range stats {
		if status != string(models.StatusWontMigrate) {
			total += count
		}
	}

	migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]
	failed := stats[string(models.StatusMigrationFailed)] + stats[string(models.StatusDryRunFailed)] + stats[string(models.StatusRolledBack)]
	pending := stats[string(models.StatusPending)] + stats[string(models.StatusDryRunQueued)] + stats[string(models.StatusDryRunInProgress)] + stats[string(models.StatusDryRunComplete)]
	inProgress := stats[string(models.StatusPreMigration)] + stats[string(models.StatusArchiveGenerating)] + stats[string(models.StatusQueuedForMigration)] + stats[string(models.StatusMigratingContent)] + stats[string(models.StatusPostMigration)]

	successRate := 0.0
	if migrated+failed > 0 {
		successRate = float64(migrated) / float64(migrated+failed) * 100
	}

	migrationVelocity, _ := h.db.GetMigrationVelocity(ctx, orgFilter, projectFilter, batchFilter, 30)
	if migrationVelocity == nil {
		migrationVelocity = &storage.MigrationVelocity{}
	}
	avgMigrationTime, _ := h.db.GetAverageMigrationTime(ctx, orgFilter, projectFilter, batchFilter)
	medianMigrationTime, _ := h.db.GetMedianMigrationTime(ctx, orgFilter, projectFilter, batchFilter)

	var estimatedCompletionDate string
	var daysRemaining int
	remaining := total - migrated
	if remaining > 0 && migrationVelocity.ReposPerDay > 0 {
		daysRemainingFloat := float64(remaining) / migrationVelocity.ReposPerDay
		daysRemaining = int(daysRemainingFloat)
		completionDate := time.Now().Add(time.Duration(daysRemainingFloat*24) * time.Hour)
		estimatedCompletionDate = completionDate.Format("2006-01-02")
	}

	var migrationCompletionStats []*storage.MigrationCompletionStats
	if h.sourceType == models.SourceTypeAzureDevOps {
		migrationCompletionStats, _ = h.db.GetMigrationCompletionStatsByProjectFiltered(ctx, orgFilter, projectFilter, batchFilter)
	} else {
		migrationCompletionStats, _ = h.db.GetMigrationCompletionStatsByOrgFiltered(ctx, orgFilter, projectFilter, batchFilter)
	}

	complexityDistribution, _ := h.db.GetComplexityDistribution(ctx, orgFilter, projectFilter, batchFilter)
	sizeDistribution, _ := h.db.GetSizeDistributionFiltered(ctx, orgFilter, projectFilter, batchFilter)
	featureStats, _ := h.db.GetFeatureStatsFiltered(ctx, orgFilter, projectFilter, batchFilter)
	if featureStats == nil {
		featureStats = &storage.FeatureStats{}
	}

	batches, _ := h.db.ListBatches(ctx)
	completedBatches, inProgressBatches, pendingBatches := 0, 0, 0
	for _, batch := range batches {
		switch batch.Status {
		case models.BatchStatusCompleted, models.BatchStatusCompletedWithErrors:
			completedBatches++
		case models.BatchStatusInProgress:
			inProgressBatches++
		case models.BatchStatusPending, models.BatchStatusReady:
			pendingBatches++
		}
	}

	completionRate := 0.0
	if total > 0 {
		completionRate = float64(migrated) / float64(total) * 100
	}

	if format == formatCSV {
		h.exportExecutiveReportCSV(w, h.sourceType, total, migrated, inProgress, pending, failed, completionRate, successRate,
			estimatedCompletionDate, daysRemaining, migrationVelocity, int(avgMigrationTime), int(medianMigrationTime),
			migrationCompletionStats, complexityDistribution, sizeDistribution, featureStats,
			stats, completedBatches, inProgressBatches, pendingBatches)
	} else {
		h.exportExecutiveReportJSON(w, h.sourceType, total, migrated, inProgress, pending, failed, completionRate, successRate,
			estimatedCompletionDate, daysRemaining, migrationVelocity, int(avgMigrationTime), int(medianMigrationTime),
			migrationCompletionStats, complexityDistribution, sizeDistribution, featureStats,
			stats, completedBatches, inProgressBatches, pendingBatches)
	}
}

// ExportDetailedDiscoveryReport handles GET /api/v1/analytics/detailed-discovery-report/export
func (h *Handler) ExportDetailedDiscoveryReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	format := r.URL.Query().Get("format")
	orgFilter := r.URL.Query().Get("organization")
	projectFilter := r.URL.Query().Get("project")
	batchFilter := r.URL.Query().Get("batch_id")

	if format != formatCSV && format != formatJSON {
		WriteError(w, ErrInvalidField.WithDetails("Invalid format. Must be 'csv' or 'json'"))
		return
	}

	filters := buildDiscoveryReportFilters(orgFilter, projectFilter, batchFilter)
	repos, err := h.db.ListRepositories(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to list repositories", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
		return
	}

	if err := h.checkDiscoveryReportAccess(ctx, repos); err != nil {
		h.logger.Warn("Detailed discovery report access denied", "error", err)
		WriteError(w, ErrForbidden.WithDetails(err.Error()))
		return
	}

	localDepsCount := h.getLocalDependenciesCount(ctx, repos)
	batchNames := h.getBatchNames(ctx, repos)

	if format == formatCSV {
		h.exportDetailedDiscoveryReportCSV(w, repos, localDepsCount, batchNames)
	} else {
		h.exportDetailedDiscoveryReportJSON(w, repos, localDepsCount, batchNames, orgFilter, projectFilter, batchFilter)
	}
}

// Migration history export helpers

func (h *Handler) exportMigrationHistoryCSV(w http.ResponseWriter, migrations []*storage.CompletedMigration) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=migration_history.csv")

	_, _ = w.Write([]byte("Repository,Source URL,Destination URL,Status,Started At,Completed At,Duration (seconds)\n"))

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

// Helper functions for report export

func buildDiscoveryReportFilters(orgFilter, projectFilter, batchFilter string) map[string]interface{} {
	filters := make(map[string]interface{})
	if orgFilter != "" {
		filters["organization"] = orgFilter
	}
	if projectFilter != "" {
		filters["project"] = projectFilter
	}
	if batchFilter != "" {
		batchID, err := strconv.ParseInt(batchFilter, 10, 64)
		if err == nil {
			filters["batch_id"] = batchID
		}
	}
	return filters
}

func (h *Handler) checkDiscoveryReportAccess(ctx context.Context, repos []*models.Repository) error {
	if !h.authConfig.Enabled {
		return nil
	}
	repoFullNames := make([]string, len(repos))
	for i, repo := range repos {
		repoFullNames[i] = repo.FullName
	}
	return h.checkRepositoriesAccess(ctx, repoFullNames)
}

func (h *Handler) getLocalDependenciesCount(ctx context.Context, repos []*models.Repository) map[int64]int {
	localDepsCount := make(map[int64]int)
	for _, repo := range repos {
		deps, err := h.db.GetRepositoryDependencies(ctx, repo.ID)
		if err == nil {
			count := 0
			for _, dep := range deps {
				if dep.IsLocal {
					count++
				}
			}
			localDepsCount[repo.ID] = count
		}
	}
	return localDepsCount
}

func (h *Handler) getBatchNames(ctx context.Context, repos []*models.Repository) map[int64]string {
	batchNames := make(map[int64]string)
	uniqueBatchIDs := make(map[int64]bool)

	for _, repo := range repos {
		if repo.BatchID != nil {
			uniqueBatchIDs[*repo.BatchID] = true
		}
	}

	for batchID := range uniqueBatchIDs {
		batch, err := h.db.GetBatch(ctx, batchID)
		if err == nil && batch != nil {
			batchNames[batchID] = batch.Name
		}
	}

	return batchNames
}

// Title case helper functions
func titleCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func formatStatusForDisplay(status string) string {
	status = strings.ReplaceAll(status, "_", " ")
	return titleCase(status)
}

func formatSourceForDisplay(source string) string {
	switch source {
	case "github":
		return "GitHub"
	case "azuredevops":
		return "Azure DevOps"
	case "gitlab":
		return "GitLab"
	case "ghes":
		return "GitHub Enterprise Server"
	default:
		return titleCase(source)
	}
}
