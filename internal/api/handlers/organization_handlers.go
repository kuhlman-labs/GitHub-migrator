package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// adoProjectStats holds the computed statistics for an ADO project
type adoProjectStats struct {
	statusCounts                map[string]int
	migratedCount               int
	inProgressCount             int
	failedCount                 int
	pendingCount                int
	migrationProgressPercentage int
}

// getADOProjectStats queries and calculates status distribution and progress metrics for an ADO project
//
//nolint:dupl // Intentionally extracted to avoid duplication in ListOrganizations and ListProjects
func (h *Handler) getADOProjectStats(ctx context.Context, projectName, organization string, repoCount int) adoProjectStats {
	stats := adoProjectStats{
		statusCounts: make(map[string]int),
	}

	if repoCount == 0 {
		return stats
	}

	var results []struct {
		Status string
		Count  int
	}
	err := h.db.DB().WithContext(ctx).
		Raw(`
			SELECT status, COUNT(*) as count
			FROM repositories
			WHERE ado_project = ?
			AND full_name LIKE ?
			AND status != 'wont_migrate'
			GROUP BY status
		`, projectName, organization+"/%").
		Scan(&results).Error

	if err != nil {
		h.logger.Warn("Failed to get status counts for project", "project", projectName, "org", organization, "error", err)
		stats.statusCounts["pending"] = repoCount
		stats.pendingCount = repoCount
	} else {
		for _, result := range results {
			stats.statusCounts[result.Status] = result.Count

			switch result.Status {
			case "complete", "migration_complete":
				stats.migratedCount += result.Count
			case "migration_failed", "dry_run_failed", "rolled_back":
				stats.failedCount += result.Count
			case "queued_for_migration", "migrating_content", "dry_run_in_progress",
				"dry_run_queued", "pre_migration", "archive_generating", "post_migration":
				stats.inProgressCount += result.Count
			default:
				stats.pendingCount += result.Count
			}
		}
	}

	if repoCount > 0 {
		stats.migrationProgressPercentage = (stats.migratedCount * 100) / repoCount
	}

	return stats
}

// ListTeams handles GET /api/v1/teams
// Returns GitHub teams with optional organization filter
func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.sourceType == models.SourceTypeAzureDevOps {
		h.sendJSON(w, http.StatusOK, []any{})
		return
	}

	orgFilter := r.URL.Query().Get("organization")

	teams, err := h.db.ListTeams(ctx, orgFilter)
	if err != nil {
		if h.handleContextError(ctx, err, "list teams", r) {
			return
		}
		h.logger.Error("Failed to list teams", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("teams"))
		return
	}

	type TeamResponse struct {
		ID           int64   `json:"id"`
		Organization string  `json:"organization"`
		Slug         string  `json:"slug"`
		Name         string  `json:"name"`
		Description  *string `json:"description,omitempty"`
		Privacy      string  `json:"privacy"`
		FullSlug     string  `json:"full_slug"`
	}

	response := make([]TeamResponse, len(teams))
	for i, team := range teams {
		response[i] = TeamResponse{
			ID:           team.ID,
			Organization: team.Organization,
			Slug:         team.Slug,
			Name:         team.Name,
			Description:  team.Description,
			Privacy:      team.Privacy,
			FullSlug:     team.FullSlug(),
		}
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ListOrganizations handles GET /api/v1/organizations
// Returns GitHub organizations or ADO projects depending on source type
// Supports multi-source environments via source_id parameter
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse source_id filter for multi-source support
	var sourceID *int64
	if sourceIDStr := r.URL.Query().Get("source_id"); sourceIDStr != "" {
		if id, err := strconv.ParseInt(sourceIDStr, 10, 64); err == nil {
			// Allocate on heap to avoid dangling pointer when if block exits
			sourceID = new(int64)
			*sourceID = id
		}
	}

	// Determine source type - either from specific source or global config
	sourceType := h.sourceType
	if sourceID != nil {
		// Look up the source to determine its type
		source, err := h.db.GetSource(ctx, *sourceID)
		if err != nil {
			h.logger.Warn("Failed to get source for ID, using global source type", "source_id", *sourceID, "error", err)
		} else if source != nil {
			sourceType = source.Type
		}
	}

	// For ADO sources, return projects grouped by ADO organization
	if sourceType == models.SourceTypeAzureDevOps {
		projects, err := h.db.GetADOProjectsFiltered(ctx, "", sourceID)
		if err != nil {
			if h.handleContextError(ctx, err, "get ADO projects", r) {
				return
			}
			h.logger.Error("Failed to get ADO projects", "error", err)
			WriteError(w, ErrDatabaseFetch.WithDetails("projects"))
			return
		}

		// Get source info if we have a source ID
		// Declare string values in outer scope to avoid dangling pointers
		var sourceName, sourceTypeStr *string
		var sourceNameVal, sourceTypeVal string
		if sourceID != nil {
			source, err := h.db.GetSource(ctx, *sourceID)
			if err == nil && source != nil {
				// Copy values to outer scope variables, then take their addresses
				sourceNameVal = source.Name
				sourceTypeVal = source.Type
				sourceName = &sourceNameVal
				sourceTypeStr = &sourceTypeVal
			}
		}

		projectStats := make([]any, 0, len(projects))
		for _, project := range projects {
			repoCount, err := h.db.CountRepositoriesByADOProjectFiltered(ctx, project.Organization, project.Name, sourceID)
			if err != nil {
				h.logger.Warn("Failed to count repositories for project", "project", project.Name, "error", err)
				repoCount = 0
			}

			stats := h.getADOProjectStats(ctx, project.Name, project.Organization, repoCount)

			projectStats = append(projectStats, map[string]any{
				"organization":                  project.Name,
				"ado_organization":              project.Organization,
				"total_repos":                   repoCount,
				"status_counts":                 stats.statusCounts,
				"migrated_count":                stats.migratedCount,
				"in_progress_count":             stats.inProgressCount,
				"failed_count":                  stats.failedCount,
				"pending_count":                 stats.pendingCount,
				"migration_progress_percentage": stats.migrationProgressPercentage,
				"source_id":                     sourceID,
				"source_name":                   sourceName,
				"source_type":                   sourceTypeStr,
			})
		}

		h.sendJSON(w, http.StatusOK, projectStats)
		return
	}

	// For GitHub sources, use filtered query with source_id support
	orgStats, err := h.db.GetOrganizationStatsFiltered(ctx, "", "", "", sourceID)
	if err != nil {
		if h.handleContextError(ctx, err, "get organization stats", r) {
			return
		}
		h.logger.Error("Failed to get organization stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("organizations"))
		return
	}

	h.sendJSON(w, http.StatusOK, orgStats)
}

// ListProjects handles GET /api/v1/projects
// Returns ADO projects with repository counts and status breakdown
// Supports optional source_id filter for multi-source environments
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse source_id filter for multi-source support
	var sourceID *int64
	if sourceIDStr := r.URL.Query().Get("source_id"); sourceIDStr != "" {
		if id, err := strconv.ParseInt(sourceIDStr, 10, 64); err == nil {
			// Allocate on heap to avoid dangling pointer when if block exits
			sourceID = new(int64)
			*sourceID = id
		}
	}

	// Get ADO projects - in multi-source environments, always try to get projects
	// The GetADOProjects query will only return results if there are ADO repos
	projects, err := h.db.GetADOProjectsFiltered(ctx, "", sourceID)
	if err != nil {
		if h.handleContextError(ctx, err, "get ADO projects", r) {
			return
		}
		h.logger.Error("Failed to get ADO projects", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("projects"))
		return
	}

	projectStats := make([]any, 0, len(projects))
	for _, project := range projects {
		repoCount, err := h.db.CountRepositoriesByADOProjectFiltered(ctx, project.Organization, project.Name, sourceID)
		if err != nil {
			h.logger.Warn("Failed to count repositories for project", "project", project.Name, "error", err)
			repoCount = 0
		}

		stats := h.getADOProjectStats(ctx, project.Name, project.Organization, repoCount)

		projectStats = append(projectStats, map[string]any{
			"organization":                  project.Name,
			"ado_organization":              project.Organization,
			"project":                       project.Name,
			"total_repos":                   repoCount,
			"status_counts":                 stats.statusCounts,
			"migrated_count":                stats.migratedCount,
			"in_progress_count":             stats.inProgressCount,
			"failed_count":                  stats.failedCount,
			"pending_count":                 stats.pendingCount,
			"migration_progress_percentage": stats.migrationProgressPercentage,
		})
	}

	h.sendJSON(w, http.StatusOK, projectStats)
}

// GetOrganizationList handles GET /api/v1/organizations/list
// Returns a simple list of organization names (for filters/dropdowns)
func (h *Handler) GetOrganizationList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgs, err := h.db.GetDistinctOrganizations(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get organization list", r) {
			return
		}
		h.logger.Error("Failed to get organization list", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("organization list"))
		return
	}

	h.sendJSON(w, http.StatusOK, orgs)
}

// GetDashboardActionItems handles GET /api/v1/dashboard/action-items
// Returns all items requiring admin attention
func (h *Handler) GetDashboardActionItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionItems, err := h.db.GetDashboardActionItems(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get dashboard action items", r) {
			return
		}
		h.logger.Error("Failed to get dashboard action items", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("dashboard action items"))
		return
	}

	h.sendJSON(w, http.StatusOK, actionItems)
}
