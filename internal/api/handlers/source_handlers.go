package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// SourceHandler handles source management API requests
type SourceHandler struct {
	db     *storage.Database
	logger *slog.Logger
}

// NewSourceHandler creates a new SourceHandler
func NewSourceHandler(db *storage.Database, logger *slog.Logger) *SourceHandler {
	return &SourceHandler{
		db:     db,
		logger: logger,
	}
}

// CreateSourceRequest represents the request body for creating a source
type CreateSourceRequest struct {
	Name              string `json:"name"`
	Type              string `json:"type"` // github or azuredevops
	BaseURL           string `json:"base_url"`
	Token             string `json:"token"`
	Organization      string `json:"organization,omitempty"`    // Required for Azure DevOps
	EnterpriseSlug    string `json:"enterprise_slug,omitempty"` // Optional for GitHub Enterprise
	AppID             *int64 `json:"app_id,omitempty"`
	AppPrivateKey     string `json:"app_private_key,omitempty"`
	AppInstallationID *int64 `json:"app_installation_id,omitempty"`
}

// UpdateSourceRequest represents the request body for updating a source
type UpdateSourceRequest struct {
	Name              *string `json:"name,omitempty"`
	BaseURL           *string `json:"base_url,omitempty"`
	Token             *string `json:"token,omitempty"`
	Organization      *string `json:"organization,omitempty"`
	EnterpriseSlug    *string `json:"enterprise_slug,omitempty"`
	AppID             *int64  `json:"app_id,omitempty"`
	AppPrivateKey     *string `json:"app_private_key,omitempty"`
	AppInstallationID *int64  `json:"app_installation_id,omitempty"`
	IsActive          *bool   `json:"is_active,omitempty"`
}

// SourceValidationRequest represents a request to validate a source connection
type SourceValidationRequest struct {
	// If source_id is provided, use stored credentials; otherwise use inline credentials
	SourceID     *int64 `json:"source_id,omitempty"`
	Type         string `json:"type,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
	Token        string `json:"token,omitempty"`
	Organization string `json:"organization,omitempty"`
}

// SourceValidationResponse represents the validation result
type SourceValidationResponse struct {
	Valid    bool           `json:"valid"`
	Error    string         `json:"error,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

// ListSources handles GET /api/v1/sources
func (h *SourceHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check for active-only filter
	activeOnly := r.URL.Query().Get("active") == boolTrue

	var sources []*models.Source
	var err error

	if activeOnly {
		sources, err = h.db.ListActiveSources(ctx)
	} else {
		sources, err = h.db.ListSources(ctx)
	}

	if err != nil {
		h.logger.Error("Failed to list sources", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("sources"))
		return
	}

	// Convert to response format (masks sensitive data)
	responses := models.SourcesToResponses(sources)
	WriteJSON(w, http.StatusOK, responses)
}

// CreateSource handles POST /api/v1/sources
func (h *SourceHandler) CreateSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	// Build source model
	source := &models.Source{
		Name:     req.Name,
		Type:     req.Type,
		BaseURL:  req.BaseURL,
		Token:    req.Token,
		IsActive: true,
	}

	if req.Organization != "" {
		source.Organization = &req.Organization
	}
	if req.EnterpriseSlug != "" {
		source.EnterpriseSlug = &req.EnterpriseSlug
	}
	if req.AppID != nil {
		source.AppID = req.AppID
	}
	if req.AppPrivateKey != "" {
		source.AppPrivateKey = &req.AppPrivateKey
	}
	if req.AppInstallationID != nil {
		source.AppInstallationID = req.AppInstallationID
	}

	// Create source (validation happens inside CreateSource)
	if err := h.db.CreateSource(ctx, source); err != nil {
		h.logger.Error("Failed to create source", "error", err)

		// Check for specific validation errors
		if strings.Contains(err.Error(), "validation failed") {
			WriteError(w, ErrValidationFailed.WithDetails(err.Error()))
			return
		}
		// Check for duplicate name
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "duplicate") {
			WriteError(w, ErrConflict.WithDetails("A source with this name already exists"))
			return
		}

		WriteError(w, ErrDatabaseSave.WithDetails("source"))
		return
	}

	h.logger.Info("Source created", "source_id", source.ID, "name", source.Name, "type", source.Type)
	WriteJSON(w, http.StatusCreated, source.ToResponse())
}

// GetSource handles GET /api/v1/sources/{id}
func (h *SourceHandler) GetSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := h.parseSourceID(r)
	if err != nil {
		WriteError(w, ErrInvalidID.WithDetails("source ID"))
		return
	}

	source, err := h.db.GetSource(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get source", "error", err, "source_id", id)
		WriteError(w, ErrDatabaseFetch.WithDetails("source"))
		return
	}
	if source == nil {
		WriteError(w, ErrNotFound.WithDetails("source"))
		return
	}

	WriteJSON(w, http.StatusOK, source.ToResponse())
}

// UpdateSource handles PUT /api/v1/sources/{id}
func (h *SourceHandler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := h.parseSourceID(r)
	if err != nil {
		WriteError(w, ErrInvalidID.WithDetails("source ID"))
		return
	}

	// Get existing source
	source, err := h.db.GetSource(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get source", "error", err, "source_id", id)
		WriteError(w, ErrDatabaseFetch.WithDetails("source"))
		return
	}
	if source == nil {
		WriteError(w, ErrNotFound.WithDetails("source"))
		return
	}

	// Parse update request
	var req UpdateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	// Apply updates
	if req.Name != nil {
		source.Name = *req.Name
	}
	if req.BaseURL != nil {
		source.BaseURL = *req.BaseURL
	}
	if req.Token != nil {
		source.Token = *req.Token
	}
	if req.Organization != nil {
		source.Organization = req.Organization
	}
	if req.EnterpriseSlug != nil {
		source.EnterpriseSlug = req.EnterpriseSlug
	}
	if req.AppID != nil {
		source.AppID = req.AppID
	}
	if req.AppPrivateKey != nil {
		source.AppPrivateKey = req.AppPrivateKey
	}
	if req.AppInstallationID != nil {
		source.AppInstallationID = req.AppInstallationID
	}
	if req.IsActive != nil {
		source.IsActive = *req.IsActive
	}

	// Update source
	if err := h.db.UpdateSource(ctx, source); err != nil {
		h.logger.Error("Failed to update source", "error", err, "source_id", id)

		if strings.Contains(err.Error(), "validation failed") {
			WriteError(w, ErrValidationFailed.WithDetails(err.Error()))
			return
		}

		WriteError(w, ErrDatabaseUpdate.WithDetails("source"))
		return
	}

	h.logger.Info("Source updated", "source_id", id, "name", source.Name)
	WriteJSON(w, http.StatusOK, source.ToResponse())
}

// DeleteSource handles DELETE /api/v1/sources/{id}
// Supports optional query parameters:
//   - force=true: Enable cascade deletion of all related data
//   - confirm={sourceName}: Required when force=true, must match the source name exactly
func (h *SourceHandler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := h.parseSourceID(r)
	if err != nil {
		WriteError(w, ErrInvalidID.WithDetails("source ID"))
		return
	}

	// Check if source exists first
	source, err := h.db.GetSource(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get source", "error", err, "source_id", id)
		WriteError(w, ErrDatabaseFetch.WithDetails("source"))
		return
	}
	if source == nil {
		WriteError(w, ErrNotFound.WithDetails("source"))
		return
	}

	// Check for force delete mode
	forceDelete := r.URL.Query().Get("force") == boolTrue
	confirmName := r.URL.Query().Get("confirm")

	if forceDelete {
		// Validate confirmation parameter
		if confirmName != source.Name {
			WriteError(w, ErrValidationFailed.WithDetails("confirmation name must match the source name exactly"))
			return
		}

		// Perform cascade delete
		if err := h.db.DeleteSourceCascade(ctx, id); err != nil {
			h.logger.Error("Failed to cascade delete source", "error", err, "source_id", id)
			WriteError(w, ErrDatabaseDelete.WithDetails("source cascade deletion failed"))
			return
		}

		h.logger.Info("Source cascade deleted", "source_id", id, "name", source.Name)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Standard delete (fails if repositories exist)
	if err := h.db.DeleteSource(ctx, id); err != nil {
		h.logger.Error("Failed to delete source", "error", err, "source_id", id)

		// Check for constraint violation (has repositories)
		if strings.Contains(err.Error(), "repositories are associated") {
			WriteError(w, ErrConflict.WithDetails(err.Error()))
			return
		}

		WriteError(w, ErrDatabaseDelete.WithDetails("source"))
		return
	}

	h.logger.Info("Source deleted", "source_id", id, "name", source.Name)
	w.WriteHeader(http.StatusNoContent)
}

// ValidateSource handles POST /api/v1/sources/validate or POST /api/v1/sources/{id}/validate
func (h *SourceHandler) ValidateSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SourceValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	var sourceType, baseURL, token, organization string

	// If source_id is provided, fetch credentials from database
	if req.SourceID != nil {
		source, err := h.db.GetSource(ctx, *req.SourceID)
		if err != nil {
			h.logger.Error("Failed to get source", "error", err, "source_id", *req.SourceID)
			WriteError(w, ErrDatabaseFetch.WithDetails("source"))
			return
		}
		if source == nil {
			WriteError(w, ErrNotFound.WithDetails("source"))
			return
		}
		sourceType = source.Type
		baseURL = source.BaseURL
		token = source.Token
		if source.Organization != nil {
			organization = *source.Organization
		}
	} else {
		// Use inline credentials
		sourceType = req.Type
		baseURL = req.BaseURL
		token = req.Token
		organization = req.Organization
	}

	// Validate based on type
	var response SourceValidationResponse
	switch strings.ToLower(sourceType) {
	case models.SourceConfigTypeGitHub:
		response = h.validateGitHubConnection(ctx, baseURL, token)
	case models.SourceConfigTypeAzureDevOps:
		response = h.validateAzureDevOpsConnection(ctx, baseURL, token, organization)
	default:
		response = SourceValidationResponse{
			Valid: false,
			Error: "Unsupported source type: " + sourceType,
		}
	}

	WriteJSON(w, http.StatusOK, response)
}

// SetSourceActive handles POST /api/v1/sources/{id}/set-active
func (h *SourceHandler) SetSourceActive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := h.parseSourceID(r)
	if err != nil {
		WriteError(w, ErrInvalidID.WithDetails("source ID"))
		return
	}

	// Parse request body for active state
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if err := h.db.SetSourceActive(ctx, id, req.IsActive); err != nil {
		h.logger.Error("Failed to set source active state", "error", err, "source_id", id)
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, ErrNotFound.WithDetails("source"))
			return
		}
		WriteError(w, ErrDatabaseUpdate.WithDetails("source"))
		return
	}

	h.logger.Info("Source active state changed", "source_id", id, "is_active", req.IsActive)
	WriteJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"source_id": id,
		"is_active": req.IsActive,
	})
}

// GetSourceRepositories handles GET /api/v1/sources/{id}/repositories
func (h *SourceHandler) GetSourceRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := h.parseSourceID(r)
	if err != nil {
		WriteError(w, ErrInvalidID.WithDetails("source ID"))
		return
	}

	// Check source exists
	source, err := h.db.GetSource(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get source", "error", err, "source_id", id)
		WriteError(w, ErrDatabaseFetch.WithDetails("source"))
		return
	}
	if source == nil {
		WriteError(w, ErrNotFound.WithDetails("source"))
		return
	}

	repos, err := h.db.GetRepositoriesBySourceID(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get repositories", "error", err, "source_id", id)
		WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
		return
	}

	WriteJSON(w, http.StatusOK, repos)
}

// GetSourceDeletionPreview handles GET /api/v1/sources/{id}/deletion-preview
// Returns counts of all data that would be deleted if the source is cascade deleted
func (h *SourceHandler) GetSourceDeletionPreview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := h.parseSourceID(r)
	if err != nil {
		WriteError(w, ErrInvalidID.WithDetails("source ID"))
		return
	}

	preview, err := h.db.GetSourceDeletionPreview(ctx, id)
	if err != nil {
		h.logger.Error("Failed to get deletion preview", "error", err, "source_id", id)
		if strings.Contains(err.Error(), "not found") {
			WriteError(w, ErrNotFound.WithDetails("source"))
			return
		}
		WriteError(w, ErrDatabaseFetch.WithDetails("deletion preview"))
		return
	}

	WriteJSON(w, http.StatusOK, preview)
}

// parseSourceID extracts the source ID from the URL path
func (h *SourceHandler) parseSourceID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		return 0, ErrInvalidID
	}
	return strconv.ParseInt(idStr, 10, 64)
}

// validateGitHubConnection validates a GitHub connection
func (h *SourceHandler) validateGitHubConnection(ctx context.Context, baseURL, token string) SourceValidationResponse {
	response := SourceValidationResponse{
		Details: make(map[string]any),
	}

	clientCfg := github.ClientConfig{
		BaseURL: baseURL,
		Token:   token,
	}

	client, err := github.NewClient(clientCfg)
	if err != nil {
		response.Valid = false
		response.Error = "Failed to create GitHub client: " + err.Error()
		return response
	}

	// Test connection by getting authenticated user
	user, _, err := client.REST().Users.Get(ctx, "")
	if err != nil {
		response.Valid = false
		response.Error = "Authentication failed: " + err.Error()
		return response
	}

	response.Valid = true
	response.Details["authenticated_user"] = user.GetLogin()
	response.Details["user_type"] = user.GetType()

	// Check scopes if available
	if user.GetLogin() != "" {
		response.Details["connection_status"] = "connected"
	}

	return response
}

// validateAzureDevOpsConnection validates an Azure DevOps connection
func (h *SourceHandler) validateAzureDevOpsConnection(ctx context.Context, baseURL, token, organization string) SourceValidationResponse {
	response := SourceValidationResponse{
		Details: make(map[string]any),
	}

	if organization == "" {
		response.Valid = false
		response.Error = "Organization is required for Azure DevOps"
		return response
	}

	// Build the organization URL (e.g., https://dev.azure.com/myorg)
	orgURL := baseURL
	if organization != "" && !strings.Contains(baseURL, organization) {
		orgURL = strings.TrimSuffix(baseURL, "/") + "/" + organization
	}

	cfg := azuredevops.ClientConfig{
		OrganizationURL:     orgURL,
		PersonalAccessToken: token,
		Logger:              h.logger,
	}

	client, err := azuredevops.NewClient(cfg)
	if err != nil {
		response.Valid = false
		response.Error = "Failed to create Azure DevOps client: " + err.Error()
		return response
	}

	// Test connection by listing projects
	projects, err := client.GetProjects(ctx)
	if err != nil {
		response.Valid = false
		response.Error = "Connection failed: " + err.Error()
		return response
	}

	response.Valid = true
	response.Details["organization"] = organization
	response.Details["project_count"] = len(projects)
	response.Details["connection_status"] = "connected"

	return response
}
