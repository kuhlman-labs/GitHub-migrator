package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/configsvc"
	"github.com/kuhlman-labs/github-migrator/internal/github"
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

	response := h.validateGitHub(ctx, req.BaseURL, req.Token)
	h.sendJSON(w, http.StatusOK, response)
}

// validateGitHub validates a GitHub connection
func (h *SettingsHandler) validateGitHub(ctx context.Context, baseURL, token string) ValidationResponse {
	response := ValidationResponse{Details: make(map[string]any)}

	// Create temporary GitHub client
	client, err := github.NewClient(github.ClientConfig{
		BaseURL: baseURL,
		Token:   token,
		Timeout: 10 * time.Second,
		Logger:  h.logger,
	})
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to create GitHub client: %v", err)
		return response
	}

	// Test connection by getting authenticated user
	user, _, err := client.REST().Users.Get(ctx, "")
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to connect to GitHub: %v", err)
		return response
	}

	response.Valid = true
	response.Details["username"] = user.GetLogin()
	response.Details["user_id"] = user.GetID()
	response.Details["base_url"] = baseURL

	// Check rate limit
	rateLimits, _, err := client.REST().RateLimit.Get(ctx)
	if err == nil && rateLimits != nil && rateLimits.Core != nil {
		response.Details["rate_limit_remaining"] = rateLimits.Core.Remaining
		response.Details["rate_limit_total"] = rateLimits.Core.Limit

		// Warn if rate limit is low
		if rateLimits.Core.Remaining < 100 {
			response.Warnings = append(response.Warnings,
				fmt.Sprintf("Low rate limit remaining: %d/%d", rateLimits.Core.Remaining, rateLimits.Core.Limit))
		}
	}

	return response
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

func (h *SettingsHandler) sendJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

