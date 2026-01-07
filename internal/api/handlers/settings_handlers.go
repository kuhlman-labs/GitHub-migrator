package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/configsvc"
	ghClient "github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/logging"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// SettingsHandler handles settings API requests
type SettingsHandler struct {
	db        *storage.Database
	logger    *slog.Logger
	configSvc *configsvc.Service
}

// NewSettingsHandler creates a new SettingsHandler
func NewSettingsHandler(db *storage.Database, logger *slog.Logger, configSvc *configsvc.Service) *SettingsHandler {
	return &SettingsHandler{
		db:        db,
		logger:    logger,
		configSvc: configSvc,
	}
}

// GetSettings handles GET /api/v1/settings
// Returns current settings with sensitive data masked
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, http.StatusOK, settings.ToResponse())
}

// UpdateSettings handles PUT /api/v1/settings
// Updates settings and triggers hot reload
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update settings in database
	settings, err := h.db.UpdateSettings(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to update settings", "error", err)
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	// Trigger hot reload
	if h.configSvc != nil {
		if err := h.configSvc.Reload(); err != nil {
			h.logger.Warn("Failed to reload config after settings update", "error", err)
			// Don't fail the request - settings were saved
		}
	}

	h.logger.Info("Settings updated successfully")
	h.sendJSON(w, http.StatusOK, settings.ToResponse())
}

// ValidateDestinationRequest represents a request to validate destination connection
type ValidateDestinationRequest struct {
	BaseURL           string `json:"base_url"`
	Token             string `json:"token"`
	AppID             int64  `json:"app_id,omitempty"`
	AppPrivateKey     string `json:"app_private_key,omitempty"`
	AppInstallationID int64  `json:"app_installation_id,omitempty"`
}

// ValidateDestination handles POST /api/v1/settings/destination/validate
// Tests connection to the destination GitHub instance
func (h *SettingsHandler) ValidateDestination(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ValidateDestinationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.BaseURL == "" {
		http.Error(w, "base_url is required", http.StatusBadRequest)
		return
	}
	if req.Token == "" && req.AppID == 0 {
		http.Error(w, "Either token or app credentials are required", http.StatusBadRequest)
		return
	}

	response := ValidateGitHubConnection(ctx, req.BaseURL, req.Token, h.logger)
	h.sendJSON(w, http.StatusOK, response)
}

// GetSetupProgress handles GET /api/v1/settings/setup-progress
// Returns the setup progress for guided empty states
func (h *SettingsHandler) GetSetupProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get settings
	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}

	// Get source count
	sources, err := h.db.ListSources(ctx)
	if err != nil {
		h.logger.Error("Failed to list sources", "error", err)
		http.Error(w, "Failed to list sources", http.StatusInternalServerError)
		return
	}

	// Get batch count
	batches, err := h.db.ListBatches(ctx)
	if err != nil {
		h.logger.Error("Failed to list batches", "error", err)
		http.Error(w, "Failed to list batches", http.StatusInternalServerError)
		return
	}

	progress := SetupProgressResponse{
		DestinationConfigured: settings.HasDestination(),
		SourcesConfigured:     len(sources) > 0,
		SourceCount:           len(sources),
		BatchesCreated:        len(batches) > 0,
		BatchCount:            len(batches),
		SetupComplete:         settings.HasDestination() && len(sources) > 0,
	}

	h.sendJSON(w, http.StatusOK, progress)
}

// SetupProgressResponse represents the setup progress for guided empty states
type SetupProgressResponse struct {
	DestinationConfigured bool `json:"destination_configured"`
	SourcesConfigured     bool `json:"sources_configured"`
	SourceCount           int  `json:"source_count"`
	BatchesCreated        bool `json:"batches_created"`
	BatchCount            int  `json:"batch_count"`
	SetupComplete         bool `json:"setup_complete"`
}

// ValidateTeamsRequest represents a request to validate teams exist in destination
type ValidateTeamsRequest struct {
	Teams []string `json:"teams"` // Teams in "org/team-slug" format
}

// ValidateTeamsResponse represents the response from team validation
type ValidateTeamsResponse struct {
	Valid        bool                   `json:"valid"`
	Teams        []TeamValidationResult `json:"teams"`
	InvalidTeams []string               `json:"invalid_teams,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// TeamValidationResult represents the validation result for a single team
type TeamValidationResult struct {
	Team  string `json:"team"`            // Original team string (org/team-slug)
	Valid bool   `json:"valid"`           // Whether the team exists
	Error string `json:"error,omitempty"` // Error message if not found
}

// ValidateTeams handles POST /api/v1/settings/teams/validate
// Validates that specified teams exist in the destination GitHub instance
func (h *SettingsHandler) ValidateTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ValidateTeamsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Filter out empty strings
	var teams []string
	for _, t := range req.Teams {
		t = strings.TrimSpace(t)
		if t != "" {
			teams = append(teams, t)
		}
	}

	// If no teams to validate, return success
	if len(teams) == 0 {
		h.sendJSON(w, http.StatusOK, ValidateTeamsResponse{
			Valid: true,
			Teams: []TeamValidationResult{},
		})
		return
	}

	// Get destination credentials from settings
	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		h.sendJSON(w, http.StatusOK, ValidateTeamsResponse{
			Valid:        false,
			ErrorMessage: "Failed to retrieve destination settings",
		})
		return
	}

	if settings.DestinationBaseURL == "" {
		h.sendJSON(w, http.StatusOK, ValidateTeamsResponse{
			Valid:        false,
			ErrorMessage: "Destination is not configured",
		})
		return
	}

	// Get destination token
	destToken := ""
	if settings.DestinationToken != nil && *settings.DestinationToken != "" {
		destToken = *settings.DestinationToken
	}
	if destToken == "" {
		h.sendJSON(w, http.StatusOK, ValidateTeamsResponse{
			Valid:        false,
			ErrorMessage: "Destination token not configured",
		})
		return
	}

	// Validate each team
	results, invalidTeams := h.validateTeamsExistence(ctx, settings.DestinationBaseURL, destToken, teams)

	response := ValidateTeamsResponse{
		Valid:        len(invalidTeams) == 0,
		Teams:        results,
		InvalidTeams: invalidTeams,
	}

	if len(invalidTeams) > 0 {
		response.ErrorMessage = fmt.Sprintf("The following teams were not found: %s", strings.Join(invalidTeams, ", "))
	}

	h.sendJSON(w, http.StatusOK, response)
}

// validateTeamsExistence checks if teams exist in the destination GitHub instance
func (h *SettingsHandler) validateTeamsExistence(ctx context.Context, baseURL, token string, teams []string) ([]TeamValidationResult, []string) {
	results := make([]TeamValidationResult, 0, len(teams))
	invalidTeams := []string{}

	// Create GitHub client for destination
	cfg := ghClient.ClientConfig{
		BaseURL: baseURL,
		Token:   token,
		Logger:  h.logger,
	}
	client, err := ghClient.NewClient(cfg)
	if err != nil {
		h.logger.Error("Failed to create GitHub client", "error", err)
		// Return all teams as invalid
		for _, team := range teams {
			results = append(results, TeamValidationResult{
				Team:  team,
				Valid: false,
				Error: "Failed to connect to destination GitHub",
			})
			invalidTeams = append(invalidTeams, team)
		}
		return results, invalidTeams
	}

	for _, team := range teams {
		result := TeamValidationResult{Team: team}

		// Parse org/team-slug format
		parts := strings.SplitN(team, "/", 2)
		if len(parts) != 2 {
			result.Valid = false
			result.Error = "Invalid format (expected org/team-slug)"
			results = append(results, result)
			invalidTeams = append(invalidTeams, team)
			continue
		}

		org := parts[0]
		teamSlug := parts[1]

		// Check if team exists using the GitHub API
		// GetTeamBySlug returns (nil, nil) for 404 (team not found)
		teamInfo, err := client.GetTeamBySlug(ctx, org, teamSlug)
		if err != nil {
			result.Valid = false
			result.Error = fmt.Sprintf("API error: %v", err)
			h.logger.Debug("Error checking team", "org", org, "team", teamSlug, "error", err)
			results = append(results, result)
			invalidTeams = append(invalidTeams, team)
			continue
		}

		if teamInfo == nil {
			result.Valid = false
			result.Error = "Team not found"
			h.logger.Debug("Team not found in destination", "org", org, "team", teamSlug)
			results = append(results, result)
			invalidTeams = append(invalidTeams, team)
			continue
		}

		result.Valid = true
		results = append(results, result)
	}

	return results, invalidTeams
}

// LoggingSettingsResponse represents the current logging configuration
type LoggingSettingsResponse struct {
	DebugEnabled bool   `json:"debug_enabled"`
	CurrentLevel string `json:"current_level"`
	DefaultLevel string `json:"default_level"`
}

// UpdateLoggingRequest represents a request to update logging settings
type UpdateLoggingRequest struct {
	DebugEnabled *bool `json:"debug_enabled,omitempty"`
}

// GetLoggingSettings handles GET /api/v1/settings/logging
// Returns current logging configuration
func (h *SettingsHandler) GetLoggingSettings(w http.ResponseWriter, _ *http.Request) {
	manager := logging.GetLogLevelManager()
	if manager == nil {
		h.sendJSON(w, http.StatusOK, LoggingSettingsResponse{
			DebugEnabled: false,
			CurrentLevel: "info",
			DefaultLevel: "info",
		})
		return
	}

	h.sendJSON(w, http.StatusOK, LoggingSettingsResponse{
		DebugEnabled: manager.IsDebugEnabled(),
		CurrentLevel: manager.GetLevel(),
		DefaultLevel: manager.GetDefaultLevel(),
	})
}

// UpdateLoggingSettings handles PUT /api/v1/settings/logging
// Updates logging configuration at runtime
func (h *SettingsHandler) UpdateLoggingSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateLoggingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	manager := logging.GetLogLevelManager()
	if manager == nil {
		http.Error(w, "Logging manager not initialized", http.StatusInternalServerError)
		return
	}

	if req.DebugEnabled != nil {
		manager.SetDebugEnabled(*req.DebugEnabled)
		if *req.DebugEnabled {
			h.logger.Info("Debug logging enabled via settings")
		} else {
			h.logger.Info("Debug logging disabled via settings, reset to default level",
				"default_level", manager.GetDefaultLevel())
		}
	}

	h.sendJSON(w, http.StatusOK, LoggingSettingsResponse{
		DebugEnabled: manager.IsDebugEnabled(),
		CurrentLevel: manager.GetLevel(),
		DefaultLevel: manager.GetDefaultLevel(),
	})
}

func (h *SettingsHandler) sendJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}
