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

	// Get current settings to detect auth changes
	currentSettings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get current settings", "error", err)
		http.Error(w, "Failed to get settings", http.StatusInternalServerError)
		return
	}
	currentAuthEnabled := currentSettings.AuthEnabled

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

	// Detect if auth settings changed - these require a server restart to take effect
	// because the HTTP router and auth middleware are built at startup
	authSettingsChanged := false
	if req.AuthEnabled != nil && *req.AuthEnabled != currentAuthEnabled {
		authSettingsChanged = true
	}
	if req.AuthGitHubOAuthClientID != nil || req.AuthGitHubOAuthClientSecret != nil ||
		req.AuthSessionSecret != nil || req.AuthCallbackURL != nil {
		authSettingsChanged = true
	}

	h.logger.Info("Settings updated successfully")

	// Return response with restart_required flag if auth settings changed
	response := settings.ToResponse()
	if authSettingsChanged {
		h.logger.Info("Auth settings changed - server restart required for changes to take effect")
		h.sendJSONWithRestart(w, http.StatusOK, response, true, "Authentication settings require a server restart to take effect")
	} else {
		h.sendJSON(w, http.StatusOK, response)
	}
}

// sendJSONWithRestart sends a JSON response with a restart_required flag
func (h *SettingsHandler) sendJSONWithRestart(w http.ResponseWriter, statusCode int, data any, restartRequired bool, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]any{
		"settings":         data,
		"restart_required": restartRequired,
		"message":          message,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
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

// ValidateOAuthRequest represents a request to validate OAuth configuration
type ValidateOAuthRequest struct {
	OAuthBaseURL  string `json:"oauth_base_url"`  // GitHub instance URL (e.g., https://github.com or https://ghes.example.com)
	OAuthClientID string `json:"oauth_client_id"` // GitHub OAuth App Client ID
	CallbackURL   string `json:"callback_url"`    // OAuth callback URL
	SessionSecret string `json:"session_secret"`  // Session encryption secret
	FrontendURL   string `json:"frontend_url"`    // Frontend URL for redirects
}

// ValidateOAuthResponse represents the OAuth validation result
type ValidateOAuthResponse struct {
	Valid    bool           `json:"valid"`
	Error    string         `json:"error,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

// ValidateOAuth handles POST /api/v1/settings/oauth/validate
// Validates OAuth configuration before saving to prevent lockout scenarios
func (h *SettingsHandler) ValidateOAuth(w http.ResponseWriter, r *http.Request) {
	var req ValidateOAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := h.validateOAuthConfig(req)
	h.sendJSON(w, http.StatusOK, response)
}

// validateOAuthConfig validates OAuth configuration
func (h *SettingsHandler) validateOAuthConfig(req ValidateOAuthRequest) ValidateOAuthResponse {
	response := ValidateOAuthResponse{
		Valid:   true,
		Details: make(map[string]any),
	}
	var warnings []string

	// 1. Validate OAuth Client ID
	if req.OAuthClientID == "" {
		response.Valid = false
		response.Error = "OAuth Client ID is required"
		return response
	}

	// GitHub OAuth client IDs have a specific format: Iv1.xxxx or Ov23xxxxx (for OAuth apps)
	// GitHub App client IDs start with Iv1. or similar
	if len(req.OAuthClientID) < 10 {
		response.Valid = false
		response.Error = "OAuth Client ID appears to be invalid (too short)"
		return response
	}
	response.Details["client_id_format"] = "valid"

	// 2. Validate Session Secret (minimum length for security)
	if req.SessionSecret == "" {
		response.Valid = false
		response.Error = "Session secret is required for secure token encryption"
		return response
	}
	if len(req.SessionSecret) < 32 {
		warnings = append(warnings, "Session secret is short (< 32 chars). Consider using a longer, random secret for better security.")
	}
	response.Details["session_secret_length"] = len(req.SessionSecret)

	// 3. Validate Callback URL format
	if req.CallbackURL == "" {
		warnings = append(warnings, "Callback URL not specified. Will be auto-generated based on server configuration.")
	} else {
		if !isValidURL(req.CallbackURL) {
			response.Valid = false
			response.Error = "Callback URL is not a valid URL"
			return response
		}
		// Check that callback URL contains the expected path
		if !strings.Contains(req.CallbackURL, "/api/v1/auth/callback") {
			warnings = append(warnings, "Callback URL should end with /api/v1/auth/callback")
		}
		response.Details["callback_url"] = req.CallbackURL
	}

	// 4. Validate Frontend URL format
	if req.FrontendURL == "" {
		warnings = append(warnings, "Frontend URL not specified. Users will be redirected to root path after login.")
	} else {
		if !isValidURL(req.FrontendURL) && req.FrontendURL != "/" {
			response.Valid = false
			response.Error = "Frontend URL is not a valid URL"
			return response
		}
		response.Details["frontend_url"] = req.FrontendURL
	}

	// 5. Validate OAuth Base URL and attempt to verify the OAuth app exists
	if req.OAuthBaseURL != "" && req.OAuthClientID != "" {
		// Try to construct the OAuth authorization URL and verify it's reachable
		oauthURL := buildOAuthBaseURL(req.OAuthBaseURL)
		response.Details["oauth_base_url"] = oauthURL

		// Note: We can't fully validate OAuth credentials without doing an actual OAuth flow
		// The client secret is only validated during token exchange
		h.logger.Info("OAuth configuration validated",
			"client_id", maskClientID(req.OAuthClientID),
			"oauth_base_url", oauthURL)
	}

	response.Warnings = warnings

	if response.Valid {
		response.Details["status"] = "Configuration appears valid. Test by logging in after enabling auth."
	}

	return response
}

// isValidURL checks if a string is a valid URL
func isValidURL(s string) bool {
	if s == "" {
		return false
	}
	// Simple validation: must start with http:// or https://
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// buildOAuthBaseURL normalizes the OAuth base URL
func buildOAuthBaseURL(baseURL string) string {
	// For api.github.com, use github.com for OAuth
	if strings.Contains(baseURL, "api.github.com") {
		return "https://github.com"
	}
	// For GHES API URLs (/api/v3), strip the API path
	if strings.Contains(baseURL, "/api/v3") {
		return strings.Replace(baseURL, "/api/v3", "", 1)
	}
	return baseURL
}

// maskClientID masks a client ID for logging, showing only first and last few chars
func maskClientID(clientID string) string {
	if len(clientID) <= 8 {
		return "****"
	}
	return clientID[:4] + "..." + clientID[len(clientID)-4:]
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
