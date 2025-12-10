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
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	users, total, err := h.db.ListUsers(ctx, sourceInstance, limit, offset)
	if err != nil {
		if h.handleContextError(ctx, err, "list users", r) {
			return
		}
		h.logger.Error("Failed to list users", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
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
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch user stats")
		return
	}

	h.sendJSON(w, http.StatusOK, stats)
}

// ListUserMappings handles GET /api/v1/user-mappings
// Returns discovered users with their mapping status (unified view)
func (h *Handler) ListUserMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filters := storage.UserWithMappingFilters{
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("search"),
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
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"mappings": users,
		"total":    total,
	})
}

// GetUserMappingStats handles GET /api/v1/user-mappings/stats
// Returns summary statistics for users with mapping status
func (h *Handler) GetUserMappingStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.db.GetUsersWithMappingsStats(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "get user mapping stats", r) {
			return
		}
		h.logger.Error("Failed to get user mapping stats", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch user mapping stats")
		return
	}

	h.sendJSON(w, http.StatusOK, stats)
}

// CreateUserMapping handles POST /api/v1/user-mappings
// Creates or updates a single user mapping
func (h *Handler) CreateUserMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		SourceLogin      string  `json:"source_login"`
		SourceEmail      *string `json:"source_email,omitempty"`
		SourceName       *string `json:"source_name,omitempty"`
		DestinationLogin *string `json:"destination_login,omitempty"`
		DestinationEmail *string `json:"destination_email,omitempty"`
		MappingStatus    string  `json:"mapping_status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SourceLogin == "" {
		h.sendError(w, http.StatusBadRequest, "source_login is required")
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
		h.sendError(w, http.StatusInternalServerError, "Failed to save user mapping")
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
		h.sendError(w, http.StatusBadRequest, "source_login is required")
		return
	}

	var req struct {
		DestinationLogin *string `json:"destination_login,omitempty"`
		DestinationEmail *string `json:"destination_email,omitempty"`
		MappingStatus    *string `json:"mapping_status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing mapping
	existing, err := h.db.GetUserMappingBySourceLogin(ctx, sourceLogin)
	if err != nil {
		if h.handleContextError(ctx, err, "get user mapping", r) {
			return
		}
		h.logger.Error("Failed to get user mapping", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get user mapping")
		return
	}

	if existing == nil {
		h.sendError(w, http.StatusNotFound, "User mapping not found")
		return
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
			h.sendError(w, http.StatusBadRequest, "Cannot set status to 'mapped' without a destination_login")
			return
		}
	}

	existing.UpdatedAt = time.Now()

	if err := h.db.SaveUserMapping(ctx, existing); err != nil {
		if h.handleContextError(ctx, err, "update user mapping", r) {
			return
		}
		h.logger.Error("Failed to update user mapping", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to update user mapping")
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
		h.sendError(w, http.StatusBadRequest, "source_login is required")
		return
	}

	if err := h.db.DeleteUserMapping(ctx, sourceLogin); err != nil {
		if h.handleContextError(ctx, err, "delete user mapping", r) {
			return
		}
		h.logger.Error("Failed to delete user mapping", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to delete user mapping")
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
		h.sendError(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Failed to read CSV header")
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
		h.sendError(w, http.StatusBadRequest, "CSV must have a 'source_login' column")
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

		// Declare variables at loop level to ensure they survive beyond the if blocks
		// This prevents dangling pointers when mappings are processed later
		var sourceEmail, sourceName, destLogin, destEmail string

		// Extract optional fields
		if sourceEmailIdx >= 0 && sourceEmailIdx < len(record) {
			if v := strings.TrimSpace(record[sourceEmailIdx]); v != "" {
				sourceEmail = v
				mapping.SourceEmail = &sourceEmail
			}
		}
		if sourceNameIdx >= 0 && sourceNameIdx < len(record) {
			if v := strings.TrimSpace(record[sourceNameIdx]); v != "" {
				sourceName = v
				mapping.SourceName = &sourceName
			}
		}
		if destLoginIdx >= 0 && destLoginIdx < len(record) {
			if v := strings.TrimSpace(record[destLoginIdx]); v != "" {
				destLogin = v
				mapping.DestinationLogin = &destLogin
			}
		}
		if destEmailIdx >= 0 && destEmailIdx < len(record) {
			if v := strings.TrimSpace(record[destEmailIdx]); v != "" {
				destEmail = v
				mapping.DestinationEmail = &destEmail
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

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
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

	mappings, _, err := h.db.ListUserMappings(ctx, filters)
	if err != nil {
		if h.handleContextError(ctx, err, "export user mappings", r) {
			return
		}
		h.logger.Error("Failed to export user mappings", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to export user mappings")
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
		h.sendError(w, http.StatusInternalServerError, "Failed to generate GEI CSV")
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
		h.sendError(w, http.StatusInternalServerError, "Failed to get unmapped users")
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

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}

// SyncUserMappingsFromDiscovery handles POST /api/v1/user-mappings/sync
// Creates user mappings for all discovered users that don't have one
func (h *Handler) SyncUserMappingsFromDiscovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	created, err := h.db.SyncUserMappingsFromUsers(ctx)
	if err != nil {
		if h.handleContextError(ctx, err, "sync user mappings", r) {
			return
		}
		h.logger.Error("Failed to sync user mappings", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to sync user mappings")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"created": created,
		"message": fmt.Sprintf("Created %d new user mappings from discovered users", created),
	})
}

// ReclaimMannequins handles POST /api/v1/user-mappings/reclaim-mannequins
// Initiates the mannequin reclaim process for mapped users
func (h *Handler) ReclaimMannequins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		DestinationOrg string `json:"destination_org"`
		DryRun         bool   `json:"dry_run"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DestinationOrg == "" {
		h.sendError(w, http.StatusBadRequest, "destination_org is required")
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
		h.sendError(w, http.StatusInternalServerError, "Failed to get user mappings")
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
		h.sendJSON(w, http.StatusOK, map[string]interface{}{
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

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":       fmt.Sprintf("Found %d mannequins pending reclaim", len(pendingReclaims)),
		"pending_count": len(pendingReclaims),
		"mappings":      pendingReclaims,
		"instructions":  instructions,
		"dry_run":       req.DryRun,
	})
}

// FetchMannequins handles POST /api/v1/user-mappings/fetch-mannequins
// Fetches mannequins from the destination organization and matches them to source users
func (h *Handler) FetchMannequins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		DestinationOrg string `json:"destination_org"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DestinationOrg == "" {
		h.sendError(w, http.StatusBadRequest, "destination_org is required")
		return
	}

	destClient := h.getDestinationClient()
	if destClient == nil {
		h.sendError(w, http.StatusBadRequest, "Destination GitHub client not configured")
		return
	}

	mannequins, err := destClient.ListMannequins(ctx, req.DestinationOrg)
	if err != nil {
		h.logger.Error("Failed to fetch mannequins", "org", req.DestinationOrg, "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch mannequins from destination")
		return
	}

	h.logger.Info("Fetched mannequins from destination", "org", req.DestinationOrg, "count", len(mannequins))

	matched, unmatched := h.matchMannequinsToUsers(ctx, mannequins)

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"total_mannequins": len(mannequins),
		"matched":          matched,
		"unmatched":        unmatched,
		"message":          fmt.Sprintf("Processed %d mannequins: %d matched to source users, %d unmatched", len(mannequins), matched, unmatched),
	})
}

// getDestinationClient returns the destination GitHub API client if available
func (h *Handler) getDestinationClient() *github.Client {
	if h.destDualClient == nil {
		return nil
	}
	return h.destDualClient.APIClient()
}

// matchMannequinsToUsers matches mannequins to existing user mappings by email
func (h *Handler) matchMannequinsToUsers(ctx context.Context, mannequins []*github.Mannequin) (matched, unmatched int) {
	allMappings, _, _ := h.db.ListUserMappings(ctx, storage.UserMappingFilters{Limit: 0})

	for _, mannequin := range mannequins {
		if mannequin.Email == "" {
			unmatched++
			continue
		}

		foundMapping := findMappingByEmail(allMappings, mannequin.Email)
		if foundMapping != nil {
			if err := h.db.UpdateMannequinInfo(ctx, foundMapping.SourceLogin, mannequin.ID, mannequin.Login); err != nil {
				h.logger.Warn("Failed to update mannequin info", "source_login", foundMapping.SourceLogin, "error", err)
				// Count as unmatched since the update failed
				unmatched++
			} else {
				matched++
				if mannequin.Claimant != nil {
					_ = h.db.UpdateReclaimStatus(ctx, foundMapping.SourceLogin, string(models.ReclaimStatusCompleted), nil)
				}
			}
		} else {
			unmatched++
		}
	}
	return matched, unmatched
}

// findMappingByEmail finds a user mapping by source email
func findMappingByEmail(mappings []*models.UserMapping, email string) *models.UserMapping {
	for _, m := range mappings {
		if m.SourceEmail != nil && *m.SourceEmail == email {
			return m
		}
	}
	return nil
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
