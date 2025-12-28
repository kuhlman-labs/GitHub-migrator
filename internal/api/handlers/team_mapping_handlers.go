package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/migration"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// teamExecutorMu protects the team executor singleton
var teamExecutorMu sync.Mutex

// teamExecutor is the singleton team executor instance
var teamExecutor *migration.TeamExecutor

// Team mapping status constants
const (
	teamMappingStatusMapped   = "mapped"
	teamMappingStatusUnmapped = "unmapped"
	teamMappingStatusSkipped  = "skipped"
)

// ListTeamMappings handles GET /api/v1/team-mappings
// Returns team mappings with optional filtering
func (h *Handler) ListTeamMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters - now queries teams with their mapping status
	// Accept both "organization" and "source_org" for compatibility
	org := r.URL.Query().Get("organization")
	if org == "" {
		org = r.URL.Query().Get("source_org")
	}
	filters := storage.TeamWithMappingFilters{
		Organization: org,
		Status:       r.URL.Query().Get("status"),
		Search:       r.URL.Query().Get("search"),
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filters.Limit = l
		}
	} else {
		filters.Limit = 100
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filters.Offset = o
		}
	}

	teams, total, err := h.db.ListTeamsWithMappings(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "list teams with mappings", r) {
			return
		}
		h.logger.Error("Failed to list teams with mappings", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("teams"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"mappings": teams,
		"total":    total,
	})
}

// GetTeamMappingStats handles GET /api/v1/team-mappings/stats
// Returns summary statistics for teams with mapping status
// Supports optional ?organization= query parameter to filter by org
func (h *Handler) GetTeamMappingStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgFilter := r.URL.Query().Get("organization")

	stats, err := h.db.GetTeamsWithMappingsStats(ctx, orgFilter)
	if err != nil {
		if h.handleContextError(ctx, err, "get team mapping stats", r) {
			return
		}
		h.logger.Error("Failed to get team mapping stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("team mapping stats"))
		return
	}

	h.sendJSON(w, http.StatusOK, stats)
}

// DiscoverTeams handles POST /api/v1/teams/discover
// Discovers teams and their members for a single organization (standalone, teams-only discovery)
func (h *Handler) DiscoverTeams(w http.ResponseWriter, r *http.Request) {
	var req DiscoverTeamsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.Organization == "" {
		WriteError(w, ErrMissingField.WithField("organization"))
		return
	}

	if h.collector == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("GitHub client"))
		return
	}

	// Run discovery synchronously since it provides immediate feedback
	ctx := r.Context()
	teamsDiscovered, membersDiscovered, err := h.collector.DiscoverTeamsOnly(ctx, req.Organization)
	if err != nil {
		if h.handleContextError(ctx, err, "discover teams", r) {
			return
		}
		h.logger.Error("Team discovery failed", "error", err, "org", req.Organization)
		WriteError(w, ErrInternal.WithDetails(fmt.Sprintf("Discovery failed: %v", err)))
		return
	}

	// Auto-sync discovered teams to team_mappings
	synced, err := h.db.SyncTeamMappingsFromTeams(ctx)
	if err != nil {
		h.logger.Warn("Failed to sync team mappings after discovery", "error", err)
	}

	// Also sync users to user_mappings since team members are discovered
	usersSynced, err := h.db.SyncUserMappingsFromUsers(ctx)
	if err != nil {
		h.logger.Warn("Failed to sync user mappings after team discovery", "error", err)
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"message":              fmt.Sprintf("Discovered %d teams with %d members from '%s'", teamsDiscovered, membersDiscovered, req.Organization),
		"organization":         req.Organization,
		"teams_discovered":     teamsDiscovered,
		"members_discovered":   membersDiscovered,
		"team_mappings_synced": synced,
		"user_mappings_synced": usersSynced,
	})
}

// GetTeamSourceOrgs handles GET /api/v1/team-mappings/source-orgs
// Returns all distinct source organizations that have teams
func (h *Handler) GetTeamSourceOrgs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgs, err := h.db.GetTeamSourceOrgs(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get team source orgs", r) {
			return
		}
		h.logger.Error("Failed to get team source orgs", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("team source organizations"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"organizations": orgs,
	})
}

// CreateTeamMapping handles POST /api/v1/team-mappings
// Creates or updates a single team mapping
func (h *Handler) CreateTeamMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateTeamMappingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.SourceOrg == "" || req.SourceTeamSlug == "" {
		WriteError(w, ErrMissingField.WithDetails("source_org and source_team_slug are required"))
		return
	}

	// Determine mapping status
	status := req.MappingStatus
	if status == "" {
		if req.DestinationOrg != nil && req.DestinationTeamSlug != nil {
			status = teamMappingStatusMapped
		} else {
			status = teamMappingStatusUnmapped
		}
	}

	mapping := &models.TeamMapping{
		SourceOrg:           req.SourceOrg,
		SourceTeamSlug:      req.SourceTeamSlug,
		SourceTeamName:      req.SourceTeamName,
		DestinationOrg:      req.DestinationOrg,
		DestinationTeamSlug: req.DestinationTeamSlug,
		DestinationTeamName: req.DestinationTeamName,
		MappingStatus:       status,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := h.db.SaveTeamMapping(ctx, mapping); err != nil {
		if h.handleContextError(ctx, err, "save team mapping", r) {
			return
		}
		h.logger.Error("Failed to save team mapping", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("team mapping"))
		return
	}

	h.sendJSON(w, http.StatusCreated, mapping)
}

// UpdateTeamMapping handles PATCH /api/v1/team-mappings/{sourceOrg}/{sourceTeamSlug}
// Updates an existing team mapping
//
//nolint:gocyclo // Handler with multiple field validations and update logic
func (h *Handler) UpdateTeamMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract source org and team slug from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/team-mappings/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		WriteError(w, ErrMissingField.WithDetails("source_org and source_team_slug are required in path"))
		return
	}
	sourceOrg, _ := decodePathComponent(parts[0])
	sourceTeamSlug, _ := decodePathComponent(parts[1])

	var req UpdateTeamMappingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	// Get existing mapping
	existing, err := h.db.GetTeamMapping(ctx, sourceOrg, sourceTeamSlug)
	if err != nil {
		if h.handleContextError(ctx, err, "get team mapping", r) {
			return
		}
		h.logger.Error("Failed to get team mapping", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("team mapping"))
		return
	}

	// If mapping doesn't exist, create a new one (upsert behavior for unmapped teams)
	if existing == nil {
		existing = &models.TeamMapping{
			SourceOrg:      sourceOrg,
			SourceTeamSlug: sourceTeamSlug,
			MappingStatus:  teamMappingStatusUnmapped,
			CreatedAt:      time.Now(),
		}
	}

	// Track if user is actively updating destination mapping fields
	updatingDestination := req.DestinationOrg != nil || req.DestinationTeamSlug != nil

	// Update fields if provided
	if req.DestinationOrg != nil {
		existing.DestinationOrg = req.DestinationOrg
	}
	if req.DestinationTeamSlug != nil {
		existing.DestinationTeamSlug = req.DestinationTeamSlug
	}
	if req.DestinationTeamName != nil {
		existing.DestinationTeamName = req.DestinationTeamName
	}
	if req.MappingStatus != nil {
		existing.MappingStatus = *req.MappingStatus
	} else if updatingDestination &&
		existing.DestinationOrg != nil && *existing.DestinationOrg != "" &&
		existing.DestinationTeamSlug != nil && *existing.DestinationTeamSlug != "" {
		// Only auto-set "mapped" when user is actively completing the destination mapping
		existing.MappingStatus = teamMappingStatusMapped
	}

	// Validate data integrity: "mapped" status requires destination_org and destination_team_slug
	if existing.MappingStatus == teamMappingStatusMapped {
		if existing.DestinationOrg == nil || *existing.DestinationOrg == "" ||
			existing.DestinationTeamSlug == nil || *existing.DestinationTeamSlug == "" {
			WriteError(w, ErrBadRequest.WithDetails("Cannot set status to 'mapped' without destination_org and destination_team_slug"))
			return
		}
	}

	existing.UpdatedAt = time.Now()

	if err := h.db.SaveTeamMapping(ctx, existing); err != nil {
		if h.handleContextError(ctx, err, "update team mapping", r) {
			return
		}
		h.logger.Error("Failed to update team mapping", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("team mapping"))
		return
	}

	h.sendJSON(w, http.StatusOK, existing)
}

// DeleteTeamMapping handles DELETE /api/v1/team-mappings/{sourceOrg}/{sourceTeamSlug}
// Deletes a team mapping
func (h *Handler) DeleteTeamMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract source org and team slug from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/team-mappings/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		WriteError(w, ErrMissingField.WithDetails("source_org and source_team_slug are required in path"))
		return
	}
	sourceOrg, _ := decodePathComponent(parts[0])
	sourceTeamSlug, _ := decodePathComponent(parts[1])

	if err := h.db.DeleteTeamMapping(ctx, sourceOrg, sourceTeamSlug); err != nil {
		if h.handleContextError(ctx, err, "delete team mapping", r) {
			return
		}
		h.logger.Error("Failed to delete team mapping", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("team mapping deletion"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{"message": "Team mapping deleted"})
}

// ImportTeamMappings handles POST /api/v1/team-mappings/import
// Imports team mappings from CSV
// nolint:gocyclo // CSV parsing requires multiple validation branches
func (h *Handler) ImportTeamMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		WriteError(w, ErrBadRequest.WithDetails("Failed to parse form data"))
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		WriteError(w, ErrMissingField.WithField("file"))
		return
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		WriteError(w, ErrBadRequest.WithDetails("Failed to read CSV header"))
		return
	}

	// Find column indices
	sourceOrgIdx := -1
	sourceTeamSlugIdx := -1
	sourceTeamNameIdx := -1
	destOrgIdx := -1
	destTeamSlugIdx := -1
	destTeamNameIdx := -1
	statusIdx := -1

	for i, col := range header {
		col = strings.TrimSpace(strings.ToLower(col))
		switch col {
		case "source_org", "sourceorg":
			sourceOrgIdx = i
		case "source_team_slug", "sourceteamslug", "source_team":
			sourceTeamSlugIdx = i
		case "source_team_name", "sourceteamname":
			sourceTeamNameIdx = i
		case "destination_org", "destinationorg", "dest_org", "destorg", "target_org":
			destOrgIdx = i
		case "destination_team_slug", "destinationteamslug", "dest_team_slug", "destteamslug", "target_team":
			destTeamSlugIdx = i
		case "destination_team_name", "destinationteamname", "dest_team_name":
			destTeamNameIdx = i
		case "status", "mapping_status":
			statusIdx = i
		}
	}

	if sourceOrgIdx == -1 || sourceTeamSlugIdx == -1 {
		WriteError(w, ErrBadRequest.WithDetails("CSV must have 'source_org' and 'source_team_slug' columns"))
		return
	}

	// Process rows
	var created, updated, errors int
	var errorMessages []string

	lineNum := 1
	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: failed to read row", lineNum))
			continue
		}

		if sourceOrgIdx >= len(record) || sourceTeamSlugIdx >= len(record) {
			errors++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: missing required columns", lineNum))
			continue
		}

		sourceOrg := strings.TrimSpace(record[sourceOrgIdx])
		sourceTeamSlug := strings.TrimSpace(record[sourceTeamSlugIdx])

		if sourceOrg == "" || sourceTeamSlug == "" {
			errors++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: empty source_org or source_team_slug", lineNum))
			continue
		}

		mapping := &models.TeamMapping{
			SourceOrg:      sourceOrg,
			SourceTeamSlug: sourceTeamSlug,
		}

		// Extract optional fields using stringPtr to ensure heap-allocated copies
		if sourceTeamNameIdx >= 0 && sourceTeamNameIdx < len(record) {
			if v := strings.TrimSpace(record[sourceTeamNameIdx]); v != "" {
				mapping.SourceTeamName = stringPtr(v)
			}
		}
		if destOrgIdx >= 0 && destOrgIdx < len(record) {
			if v := strings.TrimSpace(record[destOrgIdx]); v != "" {
				mapping.DestinationOrg = stringPtr(v)
			}
		}
		if destTeamSlugIdx >= 0 && destTeamSlugIdx < len(record) {
			if v := strings.TrimSpace(record[destTeamSlugIdx]); v != "" {
				mapping.DestinationTeamSlug = stringPtr(v)
			}
		}
		if destTeamNameIdx >= 0 && destTeamNameIdx < len(record) {
			if v := strings.TrimSpace(record[destTeamNameIdx]); v != "" {
				mapping.DestinationTeamName = stringPtr(v)
			}
		}

		// Determine status
		if statusIdx >= 0 && statusIdx < len(record) {
			if v := strings.TrimSpace(record[statusIdx]); v != "" {
				mapping.MappingStatus = v
			}
		}
		if mapping.MappingStatus == "" {
			if mapping.DestinationOrg != nil && mapping.DestinationTeamSlug != nil {
				mapping.MappingStatus = teamMappingStatusMapped
			} else {
				mapping.MappingStatus = teamMappingStatusUnmapped
			}
		}

		// Check if exists
		existing, _ := h.db.GetTeamMapping(ctx, sourceOrg, sourceTeamSlug)
		isUpdate := existing != nil

		if err := h.db.SaveTeamMapping(ctx, mapping); err != nil {
			errors++
			errorMessages = append(errorMessages, fmt.Sprintf("Failed to save %s/%s: %s", sourceOrg, sourceTeamSlug, err.Error()))
		} else {
			// Only increment counters after successful save
			if isUpdate {
				updated++
			} else {
				created++
			}
		}
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"created":  created,
		"updated":  updated,
		"errors":   errors,
		"messages": errorMessages,
	})
}

// ExportTeamMappings handles GET /api/v1/team-mappings/export
// Exports discovered teams with their mapping info to CSV
func (h *Handler) ExportTeamMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all discovered teams with their mapping info
	filters := storage.TeamWithMappingFilters{
		Limit: 0, // No limit - get all
	}

	// Apply filters if provided
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = status
	}
	if sourceOrg := r.URL.Query().Get("source_org"); sourceOrg != "" {
		filters.Organization = sourceOrg
	}

	// Use ListTeamsWithMappings to get discovered teams (from github_teams)
	// joined with their mapping info (from team_mappings)
	teams, _, err := h.db.ListTeamsWithMappings(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "export team mappings", r) {
			return
		}
		h.logger.Error("Failed to export team mappings", "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to export team mappings"))
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=team-mappings.csv")

	writer := csv.NewWriter(w)

	// Write header
	_ = writer.Write([]string{
		"source_org",
		"source_team_slug",
		"source_team_name",
		"destination_org",
		"destination_team_slug",
		"destination_team_name",
		"mapping_status",
	})

	// Write rows
	for _, t := range teams {
		row := []string{
			t.Organization,
			t.Slug,
			t.Name,
			ptrToString(t.DestinationOrg),
			ptrToString(t.DestinationTeamSlug),
			ptrToString(t.DestinationTeamName),
			t.MappingStatus,
		}
		_ = writer.Write(row)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		// Headers already sent, can only log the error
		h.logger.Error("Failed to flush CSV writer for team mappings export",
			"error", err)
	}
}

// SuggestTeamMappings handles POST /api/v1/team-mappings/suggest
// Suggests destination teams for unmapped source teams based on slug matching
func (h *Handler) SuggestTeamMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SuggestTeamMappingsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body, just return suggestions based on same-slug matching
		req.DestinationOrg = r.URL.Query().Get("destination_org")
	}

	if req.DestinationOrg == "" {
		WriteError(w, ErrMissingField.WithField("destination_org"))
		return
	}

	suggestions, err := h.db.SuggestTeamMappings(ctx, req.DestinationOrg, req.DestTeamSlugs)
	if err != nil {
		if h.handleContextError(ctx, err, "suggest team mappings", r) {
			return
		}
		h.logger.Error("Failed to suggest team mappings", "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to suggest team mappings"))
		return
	}

	// Convert to a more detailed response format
	type Suggestion struct {
		SourceFullSlug      string `json:"source_full_slug"`
		DestinationFullSlug string `json:"destination_full_slug"`
		MatchReason         string `json:"match_reason"`
		ConfidencePercent   int    `json:"confidence_percent"`
	}

	response := make([]Suggestion, 0, len(suggestions))
	for source, dest := range suggestions {
		response = append(response, Suggestion{
			SourceFullSlug:      source,
			DestinationFullSlug: dest,
			MatchReason:         "same_slug",
			ConfidencePercent:   80,
		})
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"suggestions": response,
		"total":       len(response),
	})
}

// SyncTeamMappingsFromDiscovery handles POST /api/v1/team-mappings/sync
// Creates team mappings for all discovered teams that don't have one
func (h *Handler) SyncTeamMappingsFromDiscovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	created, err := h.db.SyncTeamMappingsFromTeams(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "sync team mappings", r) {
			return
		}
		h.logger.Error("Failed to sync team mappings", "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to sync team mappings"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"created": created,
		"message": fmt.Sprintf("Created %d new team mappings from discovered teams", created),
	})
}

// GetTeamMembers handles GET /api/v1/teams/{org}/{teamSlug}/members
// Returns members of a specific team
func (h *Handler) GetTeamMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract org and team slug from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/teams/")
	path = strings.TrimSuffix(path, "/members")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		WriteError(w, ErrMissingField.WithDetails("org and team_slug are required"))
		return
	}

	org, _ := decodePathComponent(parts[0])
	teamSlug, _ := decodePathComponent(parts[1])

	members, err := h.db.GetTeamMembersByOrgAndSlug(ctx, org, teamSlug)
	if err != nil {
		if h.handleContextError(ctx, err, "get team members", r) {
			return
		}
		h.logger.Error("Failed to get team members", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("team members"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"members": members,
		"total":   len(members),
	})
}

// GetTeamDetail handles GET /api/v1/teams/{org}/{teamSlug}
// Returns comprehensive team information including members, repos, and mapping status
func (h *Handler) GetTeamDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract org and team slug from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/teams/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		WriteError(w, ErrMissingField.WithDetails("org and team_slug are required"))
		return
	}

	org, _ := decodePathComponent(parts[0])
	teamSlug, _ := decodePathComponent(parts[1])

	detail, err := h.db.GetTeamDetail(ctx, org, teamSlug)
	if err != nil {
		if h.handleContextError(ctx, err, "get team detail", r) {
			return
		}
		h.logger.Error("Failed to get team detail", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("team detail"))
		return
	}

	if detail == nil {
		WriteError(w, ErrNotFound.WithDetails("Team not found"))
		return
	}

	h.sendJSON(w, http.StatusOK, detail)
}

// UnmappedTeam represents a team without a destination mapping
type UnmappedTeam struct {
	Organization string `json:"organization"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	FullSlug     string `json:"full_slug"`
}

// CodeownersIssue represents a repository with unmapped team references in CODEOWNERS
type CodeownersIssue struct {
	RepositoryFullName string   `json:"repository_full_name"`
	UnmappedTeams      []string `json:"unmapped_teams"`
}

// GetPermissionAudit handles GET /api/v1/analytics/permission-audit
// Returns a permission audit report for migration planning
func (h *Handler) GetPermissionAudit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teams, err := h.db.ListTeams(ctx, "")
	if err != nil {
		h.logger.Error("Failed to list teams", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("teams"))
		return
	}

	teamMappings, _, err := h.db.ListTeamMappings(ctx, storage.TeamMappingFilters{Limit: 0})
	if err != nil {
		h.logger.Error("Failed to list team mappings", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("team mappings"))
		return
	}

	mappingMap := buildTeamMappingLookup(teamMappings)
	unmappedTeams := findUnmappedTeams(teams, mappingMap)
	codeownersIssues := h.findCodeownersIssues(ctx, mappingMap)

	userStats, _ := h.db.GetUserMappingStats(ctx, "")
	if userStats == nil {
		userStats = map[string]any{}
	}
	teamStats, _ := h.db.GetTeamMappingStats(ctx, "")
	if teamStats == nil {
		teamStats = map[string]any{}
	}

	report := map[string]any{
		"summary": map[string]any{
			"total_teams":        len(teams),
			"unmapped_teams":     len(unmappedTeams),
			"mapped_teams":       len(teams) - len(unmappedTeams),
			"codeowners_issues":  len(codeownersIssues),
			"user_mapping_stats": userStats,
			"team_mapping_stats": teamStats,
		},
		"unmapped_teams":    unmappedTeams,
		"codeowners_issues": codeownersIssues,
		"recommendations": []string{
			"Map all teams before migration to preserve repository access",
			"Review CODEOWNERS files with unmapped team references",
			"Ensure user mappings are complete for accurate commit attribution",
			"Use the 'Sync from Discovery' feature to create mappings for newly discovered teams/users",
		},
	}

	h.sendJSON(w, http.StatusOK, report)
}

// buildTeamMappingLookup creates a map for quick team mapping lookup
func buildTeamMappingLookup(mappings []*models.TeamMapping) map[string]*models.TeamMapping {
	result := make(map[string]*models.TeamMapping)
	for _, m := range mappings {
		key := m.SourceOrg + "/" + m.SourceTeamSlug
		result[key] = m
	}
	return result
}

// findUnmappedTeams identifies teams without valid destination mappings
func findUnmappedTeams(teams []*models.GitHubTeam, mappingMap map[string]*models.TeamMapping) []UnmappedTeam {
	var unmapped []UnmappedTeam
	for _, team := range teams {
		key := team.Organization + "/" + team.Slug
		if mapping, ok := mappingMap[key]; !ok || mapping.MappingStatus == teamMappingStatusUnmapped {
			unmapped = append(unmapped, UnmappedTeam{
				Organization: team.Organization,
				Slug:         team.Slug,
				Name:         team.Name,
				FullSlug:     team.FullSlug(),
			})
		}
	}
	return unmapped
}

// findCodeownersIssues finds repositories with CODEOWNERS that reference unmapped teams
func (h *Handler) findCodeownersIssues(ctx context.Context, mappingMap map[string]*models.TeamMapping) []CodeownersIssue {
	var issues []CodeownersIssue

	repos, err := h.db.ListRepositories(ctx, map[string]any{"has_codeowners": true})
	if err != nil {
		h.logger.Warn("Failed to query repositories with CODEOWNERS", "error", err)
		return issues
	}

	for _, repo := range repos {
		if repo.CodeownersTeams == nil || *repo.CodeownersTeams == "" {
			continue
		}

		var teamRefs []string
		if err := json.Unmarshal([]byte(*repo.CodeownersTeams), &teamRefs); err != nil {
			continue
		}

		unmapped := findUnmappedTeamRefs(teamRefs, mappingMap)
		if len(unmapped) > 0 {
			issues = append(issues, CodeownersIssue{
				RepositoryFullName: repo.FullName,
				UnmappedTeams:      unmapped,
			})
		}
	}
	return issues
}

// findUnmappedTeamRefs finds team references that are not mapped
func findUnmappedTeamRefs(teamRefs []string, mappingMap map[string]*models.TeamMapping) []string {
	var unmapped []string
	for _, ref := range teamRefs {
		teamRef := strings.TrimPrefix(ref, "@")
		if m, ok := mappingMap[teamRef]; !ok || m.MappingStatus == teamMappingStatusUnmapped {
			unmapped = append(unmapped, ref)
		}
	}
	return unmapped
}

// getOrCreateTeamExecutor returns the singleton team executor, creating it if necessary.
// Returns nil if the destination client is not configured.
func (h *Handler) getOrCreateTeamExecutor() *migration.TeamExecutor {
	teamExecutorMu.Lock()
	defer teamExecutorMu.Unlock()

	if teamExecutor == nil {
		// Check if destination client is available
		if h.destDualClient == nil {
			return nil
		}
		// Get the clients from the dual clients
		var sourceClient *github.Client
		if h.sourceDualClient != nil {
			sourceClient = h.sourceDualClient.APIClient()
		}
		var destClient = h.destDualClient.APIClient()
		// Type assert to get concrete *storage.Database for NewTeamExecutor
		db, ok := h.db.(*storage.Database)
		if !ok {
			h.logger.Error("Database type assertion failed in team executor creation")
			return nil
		}
		teamExecutor = migration.NewTeamExecutor(db, sourceClient, destClient, h.logger)
	}

	return teamExecutor
}

// ExecuteTeamMigration handles POST /api/v1/team-mappings/execute
// Triggers team migration for all mapped teams, or a single team if both source_org AND source_team_slug are provided
func (h *Handler) ExecuteTeamMigration(w http.ResponseWriter, r *http.Request) {
	// Check if destination client is available
	if h.destDualClient == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}

	// Parse request body
	var req MigrateTeamsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		WriteError(w, ErrInvalidJSON)
		return
	}

	// Also check query params for dry_run
	if r.URL.Query().Get("dry_run") == "true" {
		req.DryRun = true
	}

	executor := h.getOrCreateTeamExecutor()
	if executor == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}

	// Check if already running
	if executor.IsRunning() {
		h.sendJSON(w, http.StatusConflict, map[string]any{
			"error":    "Team migration is already running",
			"progress": executor.GetProgress(),
		})
		return
	}

	// Start execution in background
	go func() {
		if err := executor.ExecuteTeamMigration(context.Background(), req.SourceOrg, req.SourceTeamSlug, req.DryRun); err != nil {
			h.logger.Error("Team migration execution failed", "error", err)
		}
	}()

	// Return immediately with status
	h.sendJSON(w, http.StatusAccepted, map[string]any{
		"message":          "Team migration started",
		"dry_run":          req.DryRun,
		"source_org":       req.SourceOrg,
		"source_team_slug": req.SourceTeamSlug,
	})
}

// GetTeamMigrationStatus handles GET /api/v1/team-mappings/execution-status
// Returns the current status and progress of team migration execution
func (h *Handler) GetTeamMigrationStatus(w http.ResponseWriter, r *http.Request) {
	executor := h.getOrCreateTeamExecutor()
	if executor == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}
	ctx := r.Context()

	// Get execution progress
	progress := executor.GetProgress()

	// Get database stats
	executionStats, err := h.db.GetTeamMigrationExecutionStats(ctx)
	if err != nil {
		h.logger.Warn("Failed to get team migration execution stats", "error", err)
		executionStats = map[string]any{}
	}

	// Get mapping stats (all orgs for migration status)
	mappingStats, err := h.db.GetTeamsWithMappingsStats(ctx, "")
	if err != nil {
		h.logger.Warn("Failed to get team mapping stats", "error", err)
		mappingStats = map[string]any{}
	}

	response := map[string]any{
		"is_running":      executor.IsRunning(),
		"progress":        progress,
		"execution_stats": executionStats,
		"mapping_stats":   mappingStats,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// CancelTeamMigration handles POST /api/v1/team-mappings/cancel
// Cancels the currently running team migration
func (h *Handler) CancelTeamMigration(w http.ResponseWriter, r *http.Request) {
	executor := h.getOrCreateTeamExecutor()
	if executor == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}

	if err := executor.Cancel(); err != nil {
		WriteError(w, ErrBadRequest.WithDetails(err.Error()))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{
		"message": "Team migration cancellation requested",
	})
}

// ResetTeamMigrationStatus handles POST /api/v1/team-mappings/reset
// Resets all team migration statuses to pending
func (h *Handler) ResetTeamMigrationStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse optional source org filter
	sourceOrg := r.URL.Query().Get("source_org")

	// Check if migration is running
	executor := h.getOrCreateTeamExecutor()
	if executor == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}
	if executor.IsRunning() {
		WriteError(w, ErrConflict.WithDetails("Cannot reset while migration is running"))
		return
	}

	if err := h.db.ResetTeamMigrationStatus(ctx, sourceOrg); err != nil {
		if h.handleContextError(ctx, err, "reset team migration status", r) {
			return
		}
		h.logger.Error("Failed to reset team migration status", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("team migration status reset"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{
		"message": "Team migration status reset to pending",
	})
}
