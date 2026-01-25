package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/copilot"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// CopilotHandler handles Copilot API requests
type CopilotHandler struct {
	db            *storage.Database
	logger        *slog.Logger
	gitHubBaseURL string
}

// NewCopilotHandler creates a new CopilotHandler
func NewCopilotHandler(db *storage.Database, logger *slog.Logger, gitHubBaseURL string) *CopilotHandler {
	// Service will be initialized when settings are loaded
	return &CopilotHandler{
		db:            db,
		logger:        logger,
		gitHubBaseURL: gitHubBaseURL,
	}
}

// initService initializes the Copilot service with current settings
func (h *CopilotHandler) initService(settings *models.Settings) *copilot.Service {
	// Use the configured base URL, falling back to settings if available
	baseURL := h.gitHubBaseURL
	if baseURL == "" && settings.DestinationBaseURL != "" {
		baseURL = settings.DestinationBaseURL
	}

	config := copilot.ServiceConfig{
		RequireLicense:    settings.CopilotRequireLicense,
		SessionTimeoutMin: settings.CopilotSessionTimeoutMin,
		GitHubBaseURL:     baseURL,
	}

	if settings.CopilotCLIPath != nil {
		config.CLIPath = *settings.CopilotCLIPath
	}
	if settings.CopilotModel != nil {
		config.Model = *settings.CopilotModel
	}
	if settings.CopilotMaxTokens != nil {
		config.MaxTokens = *settings.CopilotMaxTokens
	}

	return copilot.NewService(h.db, h.logger, config)
}

// GetStatus handles GET /api/v1/copilot/status
// Returns the current Copilot availability status for the authenticated user
func (h *CopilotHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		h.sendJSON(w, http.StatusOK, &models.CopilotStatus{
			Enabled:           false,
			Available:         false,
			UnavailableReason: "Authentication required",
		})
		return
	}

	// Get user's OAuth token for license check
	token, _ := auth.GetTokenFromContext(ctx)

	// Get current settings
	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	// Initialize service with current settings
	service := h.initService(settings)

	// Get status
	status, err := service.GetStatus(ctx, user.Login, token, settings)
	if err != nil {
		h.logger.Error("Failed to get Copilot status", "error", err, "user", user.Login)
		h.sendError(w, http.StatusInternalServerError, "Failed to get Copilot status")
		return
	}

	h.sendJSON(w, http.StatusOK, status)
}

// SendMessage handles POST /api/v1/copilot/chat
// Sends a message to Copilot and returns the response
func (h *CopilotHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Parse request
	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		h.sendError(w, http.StatusBadRequest, "Message is required")
		return
	}

	// Get current settings
	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	// Check if Copilot is enabled
	if !settings.CopilotEnabled {
		h.sendError(w, http.StatusForbidden, "Copilot is not enabled")
		return
	}

	// Initialize service with current settings
	service := h.initService(settings)

	userIDStr := strconv.FormatInt(user.ID, 10)

	// Create new session if no session ID provided, or verify ownership of existing session
	sessionID := req.SessionID
	if sessionID == "" {
		session, err := service.CreateSession(ctx, userIDStr, user.Login, settings.CopilotSessionTimeoutMin)
		if err != nil {
			h.logger.Error("Failed to create session", "error", err, "user", user.Login)
			h.sendError(w, http.StatusInternalServerError, "Failed to create session")
			return
		}
		sessionID = session.ID
	} else {
		// Verify the session belongs to the authenticated user
		session, err := service.GetSession(ctx, sessionID)
		if err != nil {
			h.logger.Error("Failed to get session", "error", err, "session_id", sessionID)
			h.sendError(w, http.StatusNotFound, "Session not found")
			return
		}
		if session.UserID != userIDStr {
			h.sendError(w, http.StatusForbidden, "Access denied")
			return
		}
	}

	// Send message
	response, err := service.SendMessage(ctx, sessionID, req.Message, settings)
	if err != nil {
		h.logger.Error("Failed to send message", "error", err, "session_id", sessionID)
		h.sendError(w, http.StatusInternalServerError, "Failed to process message")
		return
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ListSessions handles GET /api/v1/copilot/sessions
// Returns all chat sessions for the authenticated user
func (h *CopilotHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get current settings
	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	// Initialize service with current settings
	service := h.initService(settings)

	// List sessions
	userIDStr := strconv.FormatInt(user.ID, 10)
	sessions, err := service.ListSessions(ctx, userIDStr)
	if err != nil {
		h.logger.Error("Failed to list sessions", "error", err, "user", user.Login)
		h.sendError(w, http.StatusInternalServerError, "Failed to list sessions")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// GetSessionHistory handles GET /api/v1/copilot/sessions/{id}/history
// Returns the message history for a specific session
func (h *CopilotHandler) GetSessionHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get session ID from URL path
	sessionID := r.PathValue("id")
	if sessionID == "" {
		h.sendError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Get authenticated user
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get current settings
	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	// Initialize service with current settings
	service := h.initService(settings)

	// Get session to verify ownership
	session, err := service.GetSession(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to get session", "error", err, "session_id", sessionID)
		h.sendError(w, http.StatusNotFound, "Session not found")
		return
	}

	// Verify user owns this session
	userIDStr := strconv.FormatInt(user.ID, 10)
	if session.UserID != userIDStr {
		h.sendError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Get message history
	messages, err := service.GetSessionHistory(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to get session history", "error", err, "session_id", sessionID)
		h.sendError(w, http.StatusInternalServerError, "Failed to get session history")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]any{
		"session_id": sessionID,
		"messages":   messages,
		"count":      len(messages),
	})
}

// DeleteSession handles DELETE /api/v1/copilot/sessions/{id}
// Deletes a chat session
func (h *CopilotHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get session ID from URL path
	sessionID := r.PathValue("id")
	if sessionID == "" {
		h.sendError(w, http.StatusBadRequest, "Session ID is required")
		return
	}

	// Get authenticated user
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get current settings
	settings, err := h.db.GetSettings(ctx)
	if err != nil {
		h.logger.Error("Failed to get settings", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}

	// Initialize service with current settings
	service := h.initService(settings)

	// Get session to verify ownership
	session, err := service.GetSession(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to get session", "error", err, "session_id", sessionID)
		h.sendError(w, http.StatusNotFound, "Session not found")
		return
	}

	// Verify user owns this session
	userIDStr := strconv.FormatInt(user.ID, 10)
	if session.UserID != userIDStr {
		h.sendError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Delete session
	if err := service.DeleteSession(ctx, sessionID); err != nil {
		h.logger.Error("Failed to delete session", "error", err, "session_id", sessionID)
		h.sendError(w, http.StatusInternalServerError, "Failed to delete session")
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]string{
		"message": "Session deleted",
	})
}

// ValidateCLI handles POST /api/v1/copilot/validate-cli
// Tests if the Copilot CLI is accessible at the configured path
func (h *CopilotHandler) ValidateCLI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CLIPath string `json:"cli_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	available, version, err := copilot.CheckCLIAvailable(req.CLIPath)

	response := map[string]any{
		"available": available,
		"version":   version,
	}
	if err != nil {
		response["error"] = err.Error()
	}

	h.sendJSON(w, http.StatusOK, response)
}

// sendJSON sends a JSON response
func (h *CopilotHandler) sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// sendError sends an error response
func (h *CopilotHandler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, map[string]string{"error": message})
}
