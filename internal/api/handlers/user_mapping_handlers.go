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
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// ListUsers handles GET /api/v1/users
// Returns discovered GitHub users with optional filtering
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	sourceInstance := r.URL.Query().Get("source_instance")
	pagination := ParsePagination(r)

	users, total, err := h.db.ListUsers(ctx, sourceInstance, pagination.Limit, pagination.Offset)
	if err != nil {
		if h.handleContextError(ctx, err, "list users", r) {
			return
		}
		h.logger.Error("Failed to list users", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("users"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"users": users,
		"total": total,
	})
}

// GetUserStats handles GET /api/v1/users/stats
// Returns summary statistics for discovered users
func (h *Handler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.db.GetUserStats(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get user stats", r) {
			return
		}
		h.logger.Error("Failed to get user stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("user stats"))
		return
	}

	h.sendJSON(w, http.StatusOK, stats)
}

// DiscoverOrgMembers handles POST /api/v1/users/discover
// Discovers organization members for a single organization (standalone, users-only discovery)
func (h *Handler) DiscoverOrgMembers(w http.ResponseWriter, r *http.Request) {
	var req DiscoverUsersRequest

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

	// Run discovery synchronously since it's typically fast for org members
	ctx := r.Context()
	discovered, err := collector.DiscoverOrgMembersOnly(ctx, req.Organization)
	if err != nil {
		if h.handleContextError(ctx, err, "discover org members", r) {
			return
		}
		h.logger.Error("Org member discovery failed", "error", err, "org", req.Organization)
		WriteError(w, ErrInternal.WithDetails(fmt.Sprintf("Discovery failed: %v", err)))
		return
	}

	// Auto-sync discovered users to user_mappings
	synced, err := h.db.SyncUserMappingsFromUsers(ctx)
	if err != nil {
		h.logger.Warn("Failed to sync user mappings after discovery", "error", err)
	}

	// Update source_org for user mappings from memberships
	orgsUpdated, err := h.db.UpdateUserMappingSourceOrgsFromMemberships(ctx)
	if err != nil {
		h.logger.Warn("Failed to update source orgs", "error", err)
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"message":         fmt.Sprintf("Discovered %d organization members from '%s'", discovered, req.Organization),
		"organization":    req.Organization,
		"discovered":      discovered,
		"mappings_synced": synced,
		"source_orgs_set": orgsUpdated,
	})
}

// ListUserMappings handles GET /api/v1/user-mappings
// Returns discovered users with their mapping status (unified view)
func (h *Handler) ListUserMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filters := storage.UserWithMappingFilters{
		Status:    r.URL.Query().Get("status"),
		Search:    r.URL.Query().Get("search"),
		SourceOrg: r.URL.Query().Get("source_org"),
	}

	// Parse source_id for multi-source filtering
	if sourceIDStr := r.URL.Query().Get("source_id"); sourceIDStr != "" {
		if sid, err := strconv.Atoi(sourceIDStr); err == nil {
			filters.SourceID = &sid
		}
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

	users, total, err := h.db.ListUsersWithMappings(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "list users with mappings", r) {
			return
		}
		h.logger.Error("Failed to list users with mappings", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("users"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"mappings": users,
		"total":    total,
	})
}

// GetUserMappingStats handles GET /api/v1/user-mappings/stats
// Returns summary statistics for users with mapping status
// Supports optional ?source_org= and ?source_id= query parameters to filter
func (h *Handler) GetUserMappingStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgFilter := r.URL.Query().Get("source_org")

	// Parse source_id for multi-source filtering
	var sourceID *int
	if sourceIDStr := r.URL.Query().Get("source_id"); sourceIDStr != "" {
		if sid, err := strconv.Atoi(sourceIDStr); err == nil {
			sourceID = &sid
		}
	}

	stats, err := h.db.GetUsersWithMappingsStats(ctx, orgFilter, sourceID)
	if err != nil {
		if h.handleContextError(ctx, err, "get user mapping stats", r) {
			return
		}
		h.logger.Error("Failed to get user mapping stats", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("user mapping stats"))
		return
	}

	h.sendJSON(w, http.StatusOK, stats)
}

// GetUserDetail handles GET /api/v1/user-mappings/{login}
// Returns detailed information about a user including org memberships and stats
func (h *Handler) GetUserDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	login := r.PathValue("login")
	if login == "" {
		WriteError(w, ErrMissingField.WithField("login"))
		return
	}

	// Get user from github_users
	user, err := h.db.GetUserByLogin(ctx, login)
	if err != nil {
		if h.handleContextError(ctx, err, "get user", r) {
			return
		}
		h.logger.Error("Failed to get user", "login", login, "error", err)
		WriteError(w, ErrNotFound.WithDetails("User not found"))
		return
	}

	// Get user mapping if exists (mapping is optional - user details can be returned without it)
	mapping, err := h.db.GetUserMappingBySourceLogin(ctx, login)
	if err != nil {
		// Log the error but don't fail - mapping lookup is optional for user details
		h.logger.Warn("Failed to get user mapping", "login", login, "error", err)
	}

	// Get org memberships
	orgMemberships, err := h.db.GetUserOrgMemberships(ctx, login)
	if err != nil {
		h.logger.Warn("Failed to get org memberships", "login", login, "error", err)
		orgMemberships = []*models.UserOrgMembership{}
	}

	// Build response
	response := map[string]any{
		"login":           user.Login,
		"name":            user.Name,
		"email":           user.Email,
		"avatar_url":      user.AvatarURL,
		"source_instance": user.SourceInstance,
		"discovered_at":   user.DiscoveredAt,
		"updated_at":      user.UpdatedAt,
		// Contribution stats
		"stats": map[string]any{
			"commit_count":     user.CommitCount,
			"issue_count":      user.IssueCount,
			"pr_count":         user.PRCount,
			"comment_count":    user.CommentCount,
			"repository_count": user.RepositoryCount,
		},
		// Organizations
		"organizations": orgMemberships,
	}

	// Add mapping info if exists
	if mapping != nil {
		response["mapping"] = map[string]any{
			"source_org":        mapping.SourceOrg,
			"destination_login": mapping.DestinationLogin,
			"destination_email": mapping.DestinationEmail,
			"mapping_status":    mapping.MappingStatus,
			"mannequin_id":      mapping.MannequinID,
			"mannequin_login":   mapping.MannequinLogin,
			"reclaim_status":    mapping.ReclaimStatus,
			"reclaim_error":     mapping.ReclaimError,
			"match_confidence":  mapping.MatchConfidence,
			"match_reason":      mapping.MatchReason,
		}
	}

	h.sendJSON(w, http.StatusOK, response)
}

// CreateUserMapping handles POST /api/v1/user-mappings
// Creates or updates a single user mapping
func (h *Handler) CreateUserMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateUserMappingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.SourceLogin == "" {
		WriteError(w, ErrMissingField.WithField("source_login"))
		return
	}

	// Determine mapping status
	status := req.MappingStatus
	if status == "" {
		if req.DestinationLogin != nil && *req.DestinationLogin != "" {
			status = string(models.UserMappingStatusMapped)
		} else {
			status = string(models.UserMappingStatusUnmapped)
		}
	}

	mapping := &models.UserMapping{
		SourceLogin:      req.SourceLogin,
		SourceEmail:      req.SourceEmail,
		SourceName:       req.SourceName,
		DestinationLogin: req.DestinationLogin,
		DestinationEmail: req.DestinationEmail,
		MappingStatus:    status,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.db.SaveUserMapping(ctx, mapping); err != nil {
		if h.handleContextError(ctx, err, "save user mapping", r) {
			return
		}
		h.logger.Error("Failed to save user mapping", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("user mapping"))
		return
	}

	h.sendJSON(w, http.StatusCreated, mapping)
}

// UpdateUserMapping handles PATCH /api/v1/user-mappings/{sourceLogin}
// Updates an existing user mapping
func (h *Handler) UpdateUserMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract source login from URL path
	sourceLogin := strings.TrimPrefix(r.URL.Path, "/api/v1/user-mappings/")
	sourceLogin, _ = decodePathComponent(sourceLogin)

	if sourceLogin == "" {
		WriteError(w, ErrMissingField.WithField("source_login"))
		return
	}

	var req UpdateUserMappingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	// Get existing mapping
	existing, err := h.db.GetUserMappingBySourceLogin(ctx, sourceLogin)
	if err != nil {
		if h.handleContextError(ctx, err, "get user mapping", r) {
			return
		}
		h.logger.Error("Failed to get user mapping", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("user mapping"))
		return
	}

	// If mapping doesn't exist, create a new one (upsert behavior for unmapped users)
	if existing == nil {
		existing = &models.UserMapping{
			SourceLogin:   sourceLogin,
			MappingStatus: string(models.UserMappingStatusUnmapped),
			CreatedAt:     time.Now(),
		}
	}

	// Update fields if provided
	if req.DestinationLogin != nil {
		existing.DestinationLogin = req.DestinationLogin
		if *req.DestinationLogin != "" {
			status := string(models.UserMappingStatusMapped)
			existing.MappingStatus = status
		}
	}
	if req.DestinationEmail != nil {
		existing.DestinationEmail = req.DestinationEmail
	}
	if req.MappingStatus != nil {
		existing.MappingStatus = *req.MappingStatus
	}

	// Validate data integrity: "mapped" status requires a destination_login
	if existing.MappingStatus == string(models.UserMappingStatusMapped) {
		if existing.DestinationLogin == nil || *existing.DestinationLogin == "" {
			WriteError(w, ErrBadRequest.WithDetails("Cannot set status to 'mapped' without a destination_login"))
			return
		}
	}

	existing.UpdatedAt = time.Now()

	if err := h.db.SaveUserMapping(ctx, existing); err != nil {
		if h.handleContextError(ctx, err, "update user mapping", r) {
			return
		}
		h.logger.Error("Failed to update user mapping", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("user mapping"))
		return
	}

	h.sendJSON(w, http.StatusOK, existing)
}

// DeleteUserMapping handles DELETE /api/v1/user-mappings/{sourceLogin}
// Deletes a user mapping
func (h *Handler) DeleteUserMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract source login from URL path
	sourceLogin := strings.TrimPrefix(r.URL.Path, "/api/v1/user-mappings/")
	sourceLogin, _ = decodePathComponent(sourceLogin)

	if sourceLogin == "" {
		WriteError(w, ErrMissingField.WithField("source_login"))
		return
	}

	if err := h.db.DeleteUserMapping(ctx, sourceLogin); err != nil {
		if h.handleContextError(ctx, err, "delete user mapping", r) {
			return
		}
		h.logger.Error("Failed to delete user mapping", "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails("user mapping deletion"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{"message": "User mapping deleted"})
}

// ImportUserMappings handles POST /api/v1/user-mappings/import
// Imports user mappings from CSV
// nolint:gocyclo // CSV parsing requires multiple validation branches
func (h *Handler) ImportUserMappings(w http.ResponseWriter, r *http.Request) {
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
	sourceLoginIdx := -1
	sourceEmailIdx := -1
	sourceNameIdx := -1
	destLoginIdx := -1
	destEmailIdx := -1
	statusIdx := -1

	for i, col := range header {
		col = strings.TrimSpace(strings.ToLower(col))
		switch col {
		case "source_login", "sourcelogin", "source":
			sourceLoginIdx = i
		case "source_email", "sourceemail":
			sourceEmailIdx = i
		case "source_name", "sourcename":
			sourceNameIdx = i
		case "destination_login", "destinationlogin", "destination", "target_login", "targetlogin", "target":
			destLoginIdx = i
		case "destination_email", "destinationemail", "target_email", "targetemail":
			destEmailIdx = i
		case "status", "mapping_status":
			statusIdx = i
		}
	}

	if sourceLoginIdx == -1 {
		WriteError(w, ErrBadRequest.WithDetails("CSV must have a 'source_login' column"))
		return
	}

	// Process rows
	var mappings []*models.UserMapping
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

		if sourceLoginIdx >= len(record) {
			errors++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: missing source_login column", lineNum))
			continue
		}

		sourceLogin := strings.TrimSpace(record[sourceLoginIdx])
		if sourceLogin == "" {
			errors++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: empty source_login", lineNum))
			continue
		}

		mapping := &models.UserMapping{
			SourceLogin: sourceLogin,
		}

		// Extract optional fields using stringPtr to ensure heap-allocated copies
		// This avoids any potential issues with loop variable scoping
		if sourceEmailIdx >= 0 && sourceEmailIdx < len(record) {
			if v := strings.TrimSpace(record[sourceEmailIdx]); v != "" {
				mapping.SourceEmail = stringPtr(v)
			}
		}
		if sourceNameIdx >= 0 && sourceNameIdx < len(record) {
			if v := strings.TrimSpace(record[sourceNameIdx]); v != "" {
				mapping.SourceName = stringPtr(v)
			}
		}
		if destLoginIdx >= 0 && destLoginIdx < len(record) {
			if v := strings.TrimSpace(record[destLoginIdx]); v != "" {
				mapping.DestinationLogin = stringPtr(v)
			}
		}
		if destEmailIdx >= 0 && destEmailIdx < len(record) {
			if v := strings.TrimSpace(record[destEmailIdx]); v != "" {
				mapping.DestinationEmail = stringPtr(v)
			}
		}

		// Determine status
		if statusIdx >= 0 && statusIdx < len(record) {
			if v := strings.TrimSpace(record[statusIdx]); v != "" {
				mapping.MappingStatus = v
			}
		}
		if mapping.MappingStatus == "" {
			if mapping.DestinationLogin != nil && *mapping.DestinationLogin != "" {
				mapping.MappingStatus = string(models.UserMappingStatusMapped)
			} else {
				mapping.MappingStatus = string(models.UserMappingStatusUnmapped)
			}
		}

		mappings = append(mappings, mapping)
	}

	// Save mappings
	for _, mapping := range mappings {
		existing, _ := h.db.GetUserMappingBySourceLogin(ctx, mapping.SourceLogin)
		isUpdate := existing != nil

		if err := h.db.SaveUserMapping(ctx, mapping); err != nil {
			errors++
			errorMessages = append(errorMessages, fmt.Sprintf("Failed to save %s: %s", mapping.SourceLogin, err.Error()))
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

// ExportUserMappings handles GET /api/v1/user-mappings/export
// Exports user mappings to CSV
func (h *Handler) ExportUserMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all mappings
	filters := storage.UserMappingFilters{
		Limit: 0, // No limit - get all
	}

	// Apply filters if provided
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = status
	}

	// Parse source_id filter
	if sourceIDStr := r.URL.Query().Get("source_id"); sourceIDStr != "" {
		if id, err := strconv.Atoi(sourceIDStr); err == nil {
			filters.SourceID = &id
		}
	}

	mappings, _, err := h.db.ListUserMappings(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "export user mappings", r) {
			return
		}
		h.logger.Error("Failed to export user mappings", "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to export user mappings"))
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=user-mappings.csv")

	writer := csv.NewWriter(w)

	// Write header
	_ = writer.Write([]string{
		"source_login",
		"source_email",
		"source_name",
		"destination_login",
		"destination_email",
		"mapping_status",
		"mannequin_id",
		"mannequin_login",
		"reclaim_status",
	})

	// Write rows
	for _, m := range mappings {
		row := []string{
			m.SourceLogin,
			ptrToString(m.SourceEmail),
			ptrToString(m.SourceName),
			ptrToString(m.DestinationLogin),
			ptrToString(m.DestinationEmail),
			m.MappingStatus,
			ptrToString(m.MannequinID),
			ptrToString(m.MannequinLogin),
			ptrToString(m.ReclaimStatus),
		}
		_ = writer.Write(row)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		// Headers already sent, can only log the error
		h.logger.Error("Failed to flush CSV writer for user mappings export",
			"error", err)
	}
}

// GenerateGEICSV handles GET /api/v1/user-mappings/generate-gei-csv
// Generates a CSV file compatible with gh gei reclaim-mannequin
func (h *Handler) GenerateGEICSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get only mapped users that have mannequin info
	filters := storage.UserMappingFilters{
		Status: string(models.UserMappingStatusMapped),
		Limit:  0, // Get all
	}

	// Optionally filter to only those with mannequin IDs
	if r.URL.Query().Get("mannequins_only") == boolTrue {
		hasMannequin := true
		filters.HasMannequin = &hasMannequin
	}

	mappings, _, err := h.db.ListUserMappings(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "generate GEI CSV", r) {
			return
		}
		h.logger.Error("Failed to generate GEI CSV", "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to generate GEI CSV"))
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=mannequin-mappings.csv")

	writer := csv.NewWriter(w)

	// GEI reclaim-mannequin expects: mannequin-user,mannequin-id,target-user
	// Or for EMU: source-login,target-login
	_ = writer.Write([]string{"mannequin-user", "mannequin-id", "target-user"})

	for _, m := range mappings {
		if m.DestinationLogin == nil || *m.DestinationLogin == "" {
			continue
		}

		mannequinUser := ptrToString(m.MannequinLogin)
		if mannequinUser == "" {
			mannequinUser = m.SourceLogin // Use source login as fallback
		}

		mannequinID := ptrToString(m.MannequinID)

		row := []string{
			mannequinUser,
			mannequinID,
			*m.DestinationLogin,
		}
		_ = writer.Write(row)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		// Headers already sent, can only log the error
		h.logger.Error("Failed to flush CSV writer for GEI CSV export",
			"error", err)
	}
}

// SuggestUserMappings handles POST /api/v1/user-mappings/suggest
// Suggests destination users for unmapped source users based on email/login matching
func (h *Handler) SuggestUserMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get unmapped users
	filters := storage.UserMappingFilters{
		Status: string(models.UserMappingStatusUnmapped),
		Limit:  0, // Get all
	}

	mappings, _, err := h.db.ListUserMappings(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "suggest user mappings", r) {
			return
		}
		h.logger.Error("Failed to get unmapped users", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("unmapped users"))
		return
	}

	type Suggestion struct {
		SourceLogin       string  `json:"source_login"`
		SuggestedLogin    string  `json:"suggested_login"`
		SuggestedEmail    *string `json:"suggested_email,omitempty"`
		MatchReason       string  `json:"match_reason"`
		ConfidencePercent int     `json:"confidence_percent"`
	}

	suggestions := make([]Suggestion, 0, len(mappings)*2)

	// For each unmapped user, try to find matches
	// In a real implementation, you would query the destination GitHub instance
	// For now, we'll suggest based on login pattern matching (same login)
	for _, m := range mappings {
		// Simple suggestion: same login
		suggestions = append(suggestions, Suggestion{
			SourceLogin:       m.SourceLogin,
			SuggestedLogin:    m.SourceLogin, // Same login
			MatchReason:       "same_login",
			ConfidencePercent: 70,
		})

		// If email is available, suggest email-based match
		if m.SourceEmail != nil && *m.SourceEmail != "" {
			suggestions = append(suggestions, Suggestion{
				SourceLogin:       m.SourceLogin,
				SuggestedEmail:    m.SourceEmail,
				MatchReason:       "same_email",
				ConfidencePercent: 90,
			})
		}
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}

// SyncUserMappingsFromDiscovery handles POST /api/v1/user-mappings/sync
// Creates user mappings for all discovered users that don't have one
// Also updates source_org for existing mappings from user_org_memberships
func (h *Handler) SyncUserMappingsFromDiscovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Create new mappings for users without mappings
	created, err := h.db.SyncUserMappingsFromUsers(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "sync user mappings", r) {
			return
		}
		h.logger.Error("Failed to sync user mappings", "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to sync user mappings"))
		return
	}

	// Update source_org for existing mappings that don't have it set
	updated, err := h.db.UpdateUserMappingSourceOrgsFromMemberships(ctx)
	if err != nil {
		h.logger.Warn("Failed to update source orgs from memberships", "error", err)
		// Don't fail - the sync itself succeeded
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"created":      created,
		"orgs_updated": updated,
		"message":      fmt.Sprintf("Created %d new user mappings, updated source org for %d existing mappings", created, updated),
	})
}

// ReclaimMannequins handles POST /api/v1/user-mappings/reclaim-mannequins
// Initiates the mannequin reclaim process for mapped users
func (h *Handler) ReclaimMannequins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ReconcileUsersRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.DestinationOrg == "" {
		WriteError(w, ErrMissingField.WithField("destination_org"))
		return
	}

	// Get all mapped users with mannequin info that haven't been reclaimed
	hasMannequin := true
	filters := storage.UserMappingFilters{
		Status:       string(models.UserMappingStatusMapped),
		HasMannequin: &hasMannequin,
		Limit:        0, // Get all
	}

	mappings, _, err := h.db.ListUserMappings(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to list user mappings", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("user mappings"))
		return
	}

	// Filter to only those not yet reclaimed
	var pendingReclaims []*models.UserMapping
	for _, m := range mappings {
		if m.ReclaimStatus == nil || *m.ReclaimStatus != string(models.ReclaimStatusCompleted) {
			pendingReclaims = append(pendingReclaims, m)
		}
	}

	if len(pendingReclaims) == 0 {
		h.sendJSON(w, http.StatusOK, map[string]any{
			"message":       "No pending reclaims found",
			"pending_count": 0,
			"instructions":  nil,
		})
		return
	}

	// Mark mappings as pending reclaim
	for _, m := range pendingReclaims {
		reclaimStatus := string(models.ReclaimStatusPending)
		_ = h.db.UpdateReclaimStatus(ctx, m.SourceLogin, reclaimStatus, nil)
	}

	// Generate instructions for manual reclaim (gh gei reclaim-mannequin requires CLI)
	instructions := []string{
		"1. Download the GEI CSV file from the 'Generate GEI Reclaim CSV' button",
		fmt.Sprintf("2. Run: gh gei reclaim-mannequin --github-target-org %s --csv <path-to-csv>", req.DestinationOrg),
		"3. After reclaim completes, use 'Fetch Mannequins' to update reclaim status",
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"message":       fmt.Sprintf("Found %d mannequins pending reclaim", len(pendingReclaims)),
		"pending_count": len(pendingReclaims),
		"mappings":      pendingReclaims,
		"instructions":  instructions,
		"dry_run":       req.DryRun,
	})
}

// FetchMannequins handles POST /api/v1/user-mappings/fetch-mannequins
// Fetches mannequins from the destination organization and matches them to destination org members.
// The mannequin's login IS the source user's login (from the source org).
// We match mannequins to destination org members using login, email, and name matching.
// For EMU migrations, an optional shortcode can be provided to match "jsmith" -> "jsmith_coinbase".
func (h *Handler) FetchMannequins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req AutoMapUsersRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.DestinationOrg == "" {
		WriteError(w, ErrMissingField.WithField("destination_org"))
		return
	}

	destClient := h.getDestinationClient()
	if destClient == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}

	// Step 1: Fetch mannequins from destination org
	// The mannequin's login IS the source user's login
	mannequins, err := destClient.ListMannequins(ctx, req.DestinationOrg)
	if err != nil {
		h.logger.Error("Failed to fetch mannequins", "org", req.DestinationOrg, "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to fetch mannequins from destination"))
		return
	}

	h.logger.Info("Fetched mannequins from destination", "org", req.DestinationOrg, "count", len(mannequins))

	// Step 2: Fetch destination org members to match against
	destMembers, err := destClient.ListOrgMembers(ctx, req.DestinationOrg)
	if err != nil {
		h.logger.Error("Failed to fetch destination org members", "org", req.DestinationOrg, "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to fetch destination organization members"))
		return
	}

	h.logger.Info("Fetched destination org members", "org", req.DestinationOrg, "count", len(destMembers))

	// Step 3: Match mannequins to destination members
	matched, unmatched, err := h.matchMannequinsToDestMembers(ctx, mannequins, destMembers, req.EMUShortcode)
	if err != nil {
		h.logger.Error("Failed to match mannequins to destination members", "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to match mannequins to destination members"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"total_mannequins":      len(mannequins),
		"total_dest_members":    len(destMembers),
		"matched":               matched,
		"unmatched":             unmatched,
		"destination_org":       req.DestinationOrg,
		"emu_shortcode_applied": req.EMUShortcode != "",
		"message":               fmt.Sprintf("Processed %d mannequins from '%s': %d matched to destination members, %d unmatched", len(mannequins), req.DestinationOrg, matched, unmatched),
	})
}

// getDestinationClient returns the destination GitHub API client if available
func (h *Handler) getDestinationClient() *github.Client {
	if h.destDualClient == nil {
		return nil
	}
	return h.destDualClient.APIClient()
}

// matchMannequinsToDestMembers matches mannequins to destination org members
// The mannequin's login IS the source user's login.
// We try to find the matching destination org member using:
// 1. Exact login match (100% confidence)
// 2. Login + EMU shortcode match (95% confidence)
// 3. Email exact match (90% confidence)
// 4. Name fuzzy match (70% confidence)
// Returns matched count, unmatched count, and any error
func (h *Handler) matchMannequinsToDestMembers(ctx context.Context, mannequins []*github.Mannequin, destMembers []*github.OrgMember, emuShortcode string) (matched, unmatched int, err error) {
	h.logger.Info("Starting mannequin to destination member matching",
		"total_mannequins", len(mannequins),
		"total_dest_members", len(destMembers),
		"emu_shortcode", emuShortcode)

	// Log sample of mannequin data for debugging
	if len(mannequins) > 0 {
		sample := mannequins[0]
		h.logger.Debug("Sample mannequin data",
			"login", sample.Login,
			"email", sample.Email,
			"name", sample.Name,
			"id", sample.ID)
	}

	// Log sample of destination member data for debugging
	if len(destMembers) > 0 {
		sample := destMembers[0]
		email := ""
		if sample.Email != nil {
			email = *sample.Email
		}
		name := ""
		if sample.Name != nil {
			name = *sample.Name
		}
		h.logger.Debug("Sample destination member data",
			"login", sample.Login,
			"email", email,
			"name", name)
	}

	for _, mannequin := range mannequins {
		// The mannequin login IS the source user login
		sourceLogin := mannequin.Login

		// Try to find matching destination member
		destMember, confidence, reason := h.matchMannequinToDestMember(mannequin, destMembers, emuShortcode)

		if destMember != nil {
			// Create or update user mapping with both source and destination info
			destEmail := ""
			if destMember.Email != nil {
				destEmail = *destMember.Email
			}

			// Upsert user mapping: source_login = mannequin.Login, destination_login = destMember.Login
			mapping := &models.UserMapping{
				SourceLogin:      sourceLogin,
				SourceEmail:      stringPtr(mannequin.Email),
				SourceName:       stringPtr(mannequin.Name),
				DestinationLogin: stringPtr(destMember.Login),
				DestinationEmail: stringPtr(destEmail),
				MannequinID:      stringPtr(mannequin.ID),
				MannequinLogin:   stringPtr(mannequin.Login),
				MappingStatus:    string(models.UserMappingStatusMapped),
				MatchConfidence:  &confidence,
				MatchReason:      stringPtr(reason),
			}

			if err := h.db.SaveUserMapping(ctx, mapping); err != nil {
				h.logger.Warn("Failed to save user mapping", "source_login", sourceLogin, "error", err)
				unmatched++
				continue
			}

			// Check if mannequin was already claimed
			if mannequin.Claimant != nil {
				_ = h.db.UpdateReclaimStatus(ctx, sourceLogin, string(models.ReclaimStatusCompleted), nil)
			}

			matched++
			h.logger.Debug("Matched mannequin to destination member",
				"source_login", sourceLogin,
				"destination_login", destMember.Login,
				"confidence", confidence,
				"reason", reason)
		} else {
			// No match found - create user mapping with source info only
			mapping := &models.UserMapping{
				SourceLogin:    sourceLogin,
				SourceEmail:    stringPtr(mannequin.Email),
				SourceName:     stringPtr(mannequin.Name),
				MannequinID:    stringPtr(mannequin.ID),
				MannequinLogin: stringPtr(mannequin.Login),
				MappingStatus:  string(models.UserMappingStatusUnmapped),
			}

			if err := h.db.SaveUserMapping(ctx, mapping); err != nil {
				h.logger.Warn("Failed to save unmatched user mapping", "source_login", sourceLogin, "error", err)
			}

			// Check if mannequin was already claimed even if we didn't match
			if mannequin.Claimant != nil {
				_ = h.db.UpdateReclaimStatus(ctx, sourceLogin, string(models.ReclaimStatusCompleted), nil)
			}

			unmatched++
		}
	}

	h.logger.Info("Mannequin to destination member matching complete",
		"matched", matched,
		"unmatched", unmatched)

	return matched, unmatched, nil
}

// matchMannequinToDestMember finds the best matching destination org member for a mannequin
// The mannequin's login IS the source user's login.
// Returns the matched member, confidence score (0-100), and match reason
func (h *Handler) matchMannequinToDestMember(mannequin *github.Mannequin, destMembers []*github.OrgMember, emuShortcode string) (*github.OrgMember, int, string) {
	sourceLogin := mannequin.Login

	for _, strategy := range getDestMemberMatchStrategies(emuShortcode) {
		for _, member := range destMembers {
			if strategy.match(sourceLogin, mannequin, member) {
				return member, strategy.confidence, strategy.name
			}
		}
	}

	return nil, 0, ""
}

// destMemberMatchStrategy represents a single matching strategy for mannequin-to-destination matching
type destMemberMatchStrategy struct {
	name       string
	confidence int
	match      func(sourceLogin string, mannequin *github.Mannequin, member *github.OrgMember) bool
}

// getDestMemberMatchStrategies returns all matching strategies in priority order
func getDestMemberMatchStrategies(emuShortcode string) []destMemberMatchStrategy {
	strategies := []destMemberMatchStrategy{
		// Priority 1: Exact login match (100% confidence)
		{name: "login_exact", confidence: 100, match: matchLoginExactDest},
	}

	// Priority 2: Login + EMU shortcode match (95% confidence)
	// Only add if shortcode is provided
	if emuShortcode != "" {
		strategies = append(strategies, destMemberMatchStrategy{
			name:       "login_emu",
			confidence: 95,
			match:      makeEMULoginMatcher(emuShortcode),
		})
	}

	// Priority 3: Email exact match (90% confidence)
	strategies = append(strategies, destMemberMatchStrategy{
		name: "email_exact", confidence: 90, match: matchEmailExactDest,
	})

	// Priority 4: Name contains login (fuzzy, 70% confidence)
	strategies = append(strategies, destMemberMatchStrategy{
		name: "name_fuzzy", confidence: 70, match: matchNameFuzzyDest,
	})

	return strategies
}

// matchLoginExactDest matches when source login exactly equals destination member login
func matchLoginExactDest(sourceLogin string, _ *github.Mannequin, member *github.OrgMember) bool {
	return sourceLogin != "" && member.Login != "" && strings.EqualFold(sourceLogin, member.Login)
}

// makeEMULoginMatcher creates a matcher for EMU login patterns (e.g., "jsmith" -> "jsmith_coinbase")
func makeEMULoginMatcher(emuShortcode string) func(string, *github.Mannequin, *github.OrgMember) bool {
	return func(sourceLogin string, _ *github.Mannequin, member *github.OrgMember) bool {
		if sourceLogin == "" || member.Login == "" {
			return false
		}
		expectedLogin := sourceLogin + "_" + emuShortcode
		return strings.EqualFold(expectedLogin, member.Login)
	}
}

// matchEmailExactDest matches when mannequin email exactly equals destination member email
func matchEmailExactDest(_ string, mannequin *github.Mannequin, member *github.OrgMember) bool {
	return mannequin.Email != "" && member.Email != nil && *member.Email != "" &&
		strings.EqualFold(mannequin.Email, *member.Email)
}

// matchNameFuzzyDest matches when source login is contained in destination member name
func matchNameFuzzyDest(sourceLogin string, mannequin *github.Mannequin, member *github.OrgMember) bool {
	if member.Name == nil || *member.Name == "" || sourceLogin == "" || len(sourceLogin) < 3 {
		return false
	}
	normalizedSourceLogin := strings.ToLower(sourceLogin)
	normalizedMemberName := strings.ToLower(*member.Name)

	// Check if member name contains source login
	if strings.Contains(normalizedMemberName, normalizedSourceLogin) {
		return true
	}
	// Check if mannequin name matches member name
	if mannequin.Name != "" {
		normalizedMannequinName := normalizeName(mannequin.Name)
		normalizedMemberNameClean := normalizeName(*member.Name)
		if normalizedMannequinName != "" && normalizedMemberNameClean != "" && len(normalizedMemberNameClean) >= 3 {
			return strings.Contains(normalizedMannequinName, normalizedMemberNameClean) ||
				strings.Contains(normalizedMemberNameClean, normalizedMannequinName)
		}
	}
	return false
}

// normalizeName normalizes a name for comparison
func normalizeName(name string) string {
	// Convert to lowercase
	normalized := strings.ToLower(name)
	// Handle archive-style names like "GitHubArchive\Username"
	if idx := strings.LastIndex(normalized, "\\"); idx >= 0 {
		normalized = normalized[idx+1:]
	}
	// Remove common prefixes/suffixes
	normalized = strings.TrimSpace(normalized)
	return normalized
}

// Helper functions

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func decodePathComponent(s string) (string, error) {
	// URL-decode the path component
	decoded := strings.ReplaceAll(s, "%2F", "/")
	decoded = strings.ReplaceAll(decoded, "%40", "@")
	return decoded, nil
}

// validateMappingForInvitation validates that a mapping has all required fields for sending an invitation
func validateMappingForInvitation(mapping *models.UserMapping) error {
	if mapping.MannequinID == nil || *mapping.MannequinID == "" {
		return fmt.Errorf("user has no mannequin associated - run 'Fetch Mannequins' first")
	}
	if mapping.DestinationLogin == nil || *mapping.DestinationLogin == "" {
		return fmt.Errorf("user has no destination login mapped")
	}
	return nil
}

// getOrgIDFromMannequins fetches mannequins and returns the org ID
func (h *Handler) getOrgIDFromMannequins(ctx context.Context, destClient *github.Client, org string) (string, error) {
	mannequins, err := destClient.ListMannequins(ctx, org)
	if err != nil {
		return "", fmt.Errorf("failed to get organization info: %w", err)
	}
	if len(mannequins) == 0 {
		return "", fmt.Errorf("no mannequins found in destination organization")
	}
	return mannequins[0].OrgID, nil
}

// SendAttributionInvitation handles POST /api/v1/user-mappings/{sourceLogin}/send-invitation
// Sends an attribution invitation for a single user mapping to reclaim a mannequin
func (h *Handler) SendAttributionInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract source login from URL path
	path := r.URL.Path
	sourceLogin := strings.TrimPrefix(path, "/api/v1/user-mappings/")
	sourceLogin = strings.TrimSuffix(sourceLogin, "/send-invitation")
	sourceLogin, _ = decodePathComponent(sourceLogin)

	if sourceLogin == "" {
		WriteError(w, ErrMissingField.WithField("source_login"))
		return
	}

	var req ValidateMappingsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.DestinationOrg == "" {
		WriteError(w, ErrMissingField.WithField("destination_org"))
		return
	}

	// Get and validate the mapping
	mapping, err := h.db.GetUserMappingBySourceLogin(ctx, sourceLogin)
	if err != nil {
		h.logger.Error("Failed to get user mapping", "source_login", sourceLogin, "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("user mapping"))
		return
	}
	if mapping == nil {
		WriteError(w, ErrNotFound.WithDetails("User mapping not found"))
		return
	}
	if err := validateMappingForInvitation(mapping); err != nil {
		WriteError(w, ErrBadRequest.WithDetails(err.Error()))
		return
	}

	destClient := h.getDestinationClient()
	if destClient == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}

	// Get target user and org ID
	targetUser, err := destClient.GetUserByLogin(ctx, *mapping.DestinationLogin)
	if err != nil {
		h.logger.Error("Failed to get destination user", "login", *mapping.DestinationLogin, "error", err)
		WriteError(w, ErrInternal.WithDetails("Failed to look up destination user"))
		return
	}
	if targetUser == nil {
		WriteError(w, ErrBadRequest.WithDetails(fmt.Sprintf("Destination user '%s' not found on GitHub", *mapping.DestinationLogin)))
		return
	}

	orgID, err := h.getOrgIDFromMannequins(ctx, destClient, req.DestinationOrg)
	if err != nil {
		h.logger.Error("Failed to fetch mannequins for org ID", "org", req.DestinationOrg, "error", err)
		WriteError(w, ErrInternal.WithDetails(err.Error()))
		return
	}

	// Send the attribution invitation
	result, err := destClient.CreateAttributionInvitation(ctx, orgID, *mapping.MannequinID, targetUser.ID)
	if err != nil {
		h.logger.Error("Failed to create attribution invitation",
			"mannequin_id", *mapping.MannequinID,
			"target_user", targetUser.Login,
			"error", err)
		errMsg := err.Error()
		_ = h.db.UpdateReclaimStatus(ctx, sourceLogin, string(models.ReclaimStatusFailed), &errMsg)
		WriteError(w, ErrInternal.WithDetails(fmt.Sprintf("Failed to send invitation: %s", err.Error())))
		return
	}

	_ = h.db.UpdateReclaimStatus(ctx, sourceLogin, string(models.ReclaimStatusInvited), nil)

	h.logger.Info("Attribution invitation sent",
		"source_login", sourceLogin,
		"mannequin_login", result.MannequinLogin,
		"target_user", result.TargetUserLogin)

	h.sendJSON(w, http.StatusOK, map[string]any{
		"success":         true,
		"source_login":    sourceLogin,
		"mannequin_login": result.MannequinLogin,
		"target_user":     result.TargetUserLogin,
		"message":         fmt.Sprintf("Invitation sent to %s to reclaim mannequin %s", result.TargetUserLogin, result.MannequinLogin),
	})
}

// filterMappingsByLogins filters mappings to only include those with matching source logins
func filterMappingsByLogins(mappings []*models.UserMapping, sourceLogins []string) []*models.UserMapping {
	if len(sourceLogins) == 0 {
		return mappings
	}
	loginSet := make(map[string]bool)
	for _, login := range sourceLogins {
		loginSet[login] = true
	}
	var filtered []*models.UserMapping
	for _, m := range mappings {
		if loginSet[m.SourceLogin] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// filterPendingMappings returns only mappings not yet invited or reclaimed
func filterPendingMappings(mappings []*models.UserMapping) []*models.UserMapping {
	var pending []*models.UserMapping
	for _, m := range mappings {
		if m.ReclaimStatus == nil || (*m.ReclaimStatus != string(models.ReclaimStatusInvited) && *m.ReclaimStatus != string(models.ReclaimStatusCompleted)) {
			pending = append(pending, m)
		}
	}
	return pending
}

// bulkInvitationResult holds the results of bulk invitation processing
type bulkInvitationResult struct {
	invited int
	failed  int
	skipped int
	errors  []string
}

// processBulkInvitations sends invitations for a list of mappings
func (h *Handler) processBulkInvitations(ctx context.Context, destClient *github.Client, orgID string, mappings []*models.UserMapping) bulkInvitationResult {
	result := bulkInvitationResult{}

	for _, mapping := range mappings {
		if err := validateMappingForInvitation(mapping); err != nil {
			result.skipped++
			continue
		}

		targetUser, err := destClient.GetUserByLogin(ctx, *mapping.DestinationLogin)
		if err != nil || targetUser == nil {
			result.failed++
			result.errors = append(result.errors, fmt.Sprintf("%s: destination user not found", mapping.SourceLogin))
			continue
		}

		_, err = destClient.CreateAttributionInvitation(ctx, orgID, *mapping.MannequinID, targetUser.ID)
		if err != nil {
			result.failed++
			errMsg := err.Error()
			result.errors = append(result.errors, fmt.Sprintf("%s: %s", mapping.SourceLogin, errMsg))
			_ = h.db.UpdateReclaimStatus(ctx, mapping.SourceLogin, string(models.ReclaimStatusFailed), &errMsg)
		} else {
			result.invited++
			_ = h.db.UpdateReclaimStatus(ctx, mapping.SourceLogin, string(models.ReclaimStatusInvited), nil)
		}
	}

	return result
}

// BulkSendAttributionInvitations handles POST /api/v1/user-mappings/send-invitations
// Sends attribution invitations for multiple user mappings
func (h *Handler) BulkSendAttributionInvitations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req MigrateUsersRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.DestinationOrg == "" {
		WriteError(w, ErrMissingField.WithField("destination_org"))
		return
	}

	destClient := h.getDestinationClient()
	if destClient == nil {
		WriteError(w, ErrClientNotConfigured.WithDetails("Destination GitHub client"))
		return
	}

	orgID, err := h.getOrgIDFromMannequins(ctx, destClient, req.DestinationOrg)
	if err != nil {
		h.logger.Error("Failed to fetch mannequins", "org", req.DestinationOrg, "error", err)
		WriteError(w, ErrInternal.WithDetails(err.Error()))
		return
	}

	// Get mappings to process
	hasMannequin := true
	filters := storage.UserMappingFilters{
		Status:       string(models.UserMappingStatusMapped),
		HasMannequin: &hasMannequin,
		Limit:        0,
	}

	mappings, _, err := h.db.ListUserMappings(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to list user mappings", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("user mappings"))
		return
	}

	// Apply filters
	mappings = filterMappingsByLogins(mappings, req.SourceLogins)
	pendingMappings := filterPendingMappings(mappings)

	if len(pendingMappings) == 0 {
		h.sendJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"invited": 0,
			"failed":  0,
			"skipped": len(mappings),
			"message": "No pending mappings to invite",
		})
		return
	}

	result := h.processBulkInvitations(ctx, destClient, orgID, pendingMappings)

	h.logger.Info("Bulk invitation complete",
		"invited", result.invited,
		"failed", result.failed,
		"skipped", result.skipped)

	h.sendJSON(w, http.StatusOK, map[string]any{
		"success": result.failed == 0,
		"invited": result.invited,
		"failed":  result.failed,
		"skipped": result.skipped,
		"errors":  result.errors,
		"message": fmt.Sprintf("Sent %d invitations, %d failed, %d skipped", result.invited, result.failed, result.skipped),
	})
}

// GetSourceOrgs handles GET /api/v1/user-mappings/source-orgs
// Returns a list of unique source organizations from user mappings
func (h *Handler) GetSourceOrgs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgs, err := h.db.GetUserMappingSourceOrgs(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get source orgs", r) {
			return
		}
		h.logger.Error("Failed to get source orgs", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("source organizations"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"organizations": orgs,
	})
}
