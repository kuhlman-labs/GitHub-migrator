package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/copilot"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// CopilotHandler handles Copilot API requests
type CopilotHandler struct {
	db            *storage.Database
	logger        *slog.Logger
	gitHubBaseURL string
	authorizer    *auth.Authorizer
	authConfig    *config.AuthConfig

	// Persistent service instance to maintain session state
	service   *copilot.Service
	serviceMu sync.RWMutex
}

// NewCopilotHandler creates a new CopilotHandler
func NewCopilotHandler(db *storage.Database, logger *slog.Logger, gitHubBaseURL string) *CopilotHandler {
	return &CopilotHandler{
		db:            db,
		logger:        logger,
		gitHubBaseURL: gitHubBaseURL,
	}
}

// SetAuthorizer sets the authorizer for authorization checks
func (h *CopilotHandler) SetAuthorizer(authorizer *auth.Authorizer, authConfig *config.AuthConfig) {
	h.authorizer = authorizer
	h.authConfig = authConfig
}

// sessionError represents an error during session handling
type sessionError struct {
	StatusCode int
	Message    string
}

// getOrCreateSession creates a new session or verifies ownership of existing one.
// Returns the session ID to use, or a sessionError if something went wrong.
func (h *CopilotHandler) getOrCreateSession(
	ctx context.Context,
	service *copilot.Service,
	sessionID, userIDStr, userLogin string,
	timeoutMin int,
	authCtx *copilot.AuthContext,
) (string, *sessionError) {
	if sessionID == "" {
		session, err := service.CreateSession(ctx, userIDStr, userLogin, timeoutMin, authCtx)
		if err != nil {
			h.logger.Error("Failed to create session", "error", err, "user", userLogin)
			return "", &sessionError{http.StatusInternalServerError, "Failed to create session"}
		}
		return session.ID, nil
	}

	// Verify the session belongs to the authenticated user
	session, err := service.GetSession(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to get session", "error", err, "session_id", sessionID)
		return "", &sessionError{http.StatusNotFound, "Session not found"}
	}
	if session.UserID != userIDStr {
		return "", &sessionError{http.StatusForbidden, "Access denied"}
	}
	// Update the session's auth context in case permissions changed
	// This persists the auth to the cached session for subsequent operations
	service.UpdateSessionAuth(sessionID, authCtx)
	return sessionID, nil
}

// getUserAuthContext gets the authorization context for the current user
func (h *CopilotHandler) getUserAuthContext(ctx context.Context, user *auth.GitHubUser, token string) *copilot.AuthContext {
	authCtx := &copilot.AuthContext{
		UserID:    strconv.FormatInt(user.ID, 10),
		UserLogin: user.Login,
		Tier:      "read_only", // Default
		Permissions: copilot.ToolPermissions{
			CanRead:           true,
			CanMigrateOwn:     false,
			CanMigrateAll:     false,
			CanManageSettings: false,
		},
	}

	// If no authorizer configured, default to admin (backward compatibility)
	if h.authorizer == nil {
		authCtx.Tier = copilot.AuthTierAdmin
		authCtx.Permissions.CanMigrateOwn = true
		authCtx.Permissions.CanMigrateAll = true
		authCtx.Permissions.CanManageSettings = true
		return authCtx
	}

	// Get user's authorization tier
	tierInfo, err := h.authorizer.GetUserAuthorizationTier(ctx, user, token)
	if err != nil {
		h.logger.Warn("Failed to get user authorization tier, defaulting to read-only", "user", user.Login, "error", err)
		return authCtx
	}

	// Convert auth tier to copilot auth context
	switch tierInfo.Tier {
	case auth.TierAdmin:
		authCtx.Tier = copilot.AuthTierAdmin
		authCtx.Permissions.CanMigrateOwn = true
		authCtx.Permissions.CanMigrateAll = true
		authCtx.Permissions.CanManageSettings = true
	case auth.TierSelfService:
		authCtx.Tier = copilot.AuthTierSelfService
		authCtx.Permissions.CanMigrateOwn = true
	case auth.TierReadOnly:
		authCtx.Tier = copilot.AuthTierReadOnly
	}

	return authCtx
}

// getOrCreateService returns the persistent service instance, creating it if needed
func (h *CopilotHandler) getOrCreateService(settings *models.Settings) *copilot.Service {
	h.serviceMu.RLock()
	if h.service != nil {
		h.serviceMu.RUnlock()
		return h.service
	}
	h.serviceMu.RUnlock()

	// Need to create the service
	h.serviceMu.Lock()
	defer h.serviceMu.Unlock()

	// Double-check after acquiring write lock
	if h.service != nil {
		return h.service
	}

	// Use the configured base URL, falling back to settings if available
	baseURL := h.gitHubBaseURL
	if baseURL == "" && settings.DestinationBaseURL != "" {
		baseURL = settings.DestinationBaseURL
	}

	config := copilot.ServiceConfig{
		RequireLicense:    settings.CopilotRequireLicense,
		SessionTimeoutMin: settings.CopilotSessionTimeoutMin,
		GitHubBaseURL:     baseURL,
		Streaming:         settings.CopilotStreaming,
		LogLevel:          settings.CopilotLogLevel,
	}

	if settings.CopilotCLIPath != nil {
		config.CLIPath = *settings.CopilotCLIPath
	}
	if settings.CopilotModel != nil {
		config.Model = *settings.CopilotModel
	}

	h.service = copilot.NewService(h.db, h.logger, config)
	h.logger.Info("Created persistent Copilot service with SDK")
	return h.service
}

// GetStatus handles GET /api/v1/copilot/status
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
	service := h.getOrCreateService(settings)

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
	service := h.getOrCreateService(settings)

	userIDStr := strconv.FormatInt(user.ID, 10)

	// Get user's OAuth token for authorization checks
	token, _ := auth.GetTokenFromContext(ctx)

	// Get authorization context for this user
	authCtx := h.getUserAuthContext(ctx, user, token)

	// Create new session if no session ID provided, or verify ownership of existing session
	sessionID, sessionErr := h.getOrCreateSession(ctx, service, req.SessionID, userIDStr, user.Login, settings.CopilotSessionTimeoutMin, authCtx)
	if sessionErr != nil {
		h.sendError(w, sessionErr.StatusCode, sessionErr.Message)
		return
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

// StreamChat handles GET /api/v1/copilot/chat/stream
// This endpoint uses Server-Sent Events (SSE) for streaming responses.
func (h *CopilotHandler) StreamChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authenticated user
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get parameters from query string
	sessionID := r.URL.Query().Get("session_id")
	message := r.URL.Query().Get("message")

	if strings.TrimSpace(message) == "" {
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
	service := h.getOrCreateService(settings)

	userIDStr := strconv.FormatInt(user.ID, 10)

	// Get user's OAuth token for authorization checks
	token, _ := auth.GetTokenFromContext(ctx)

	// Get authorization context for this user
	authCtx := h.getUserAuthContext(ctx, user, token)

	// Create new session if no session ID provided, or verify ownership of existing one
	sessionID, sessionErr := h.getOrCreateSession(ctx, service, sessionID, userIDStr, user.Login, settings.CopilotSessionTimeoutMin, authCtx)
	if sessionErr != nil {
		h.sendError(w, sessionErr.StatusCode, sessionErr.Message)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get the flusher interface
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.sendError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Send initial session_id event
	h.writeSSEEvent(w, flusher, "session", map[string]string{"session_id": sessionID})

	h.logger.Debug("Starting stream for message", "session_id", sessionID, "message_length", len(message))

	// Start keepalive goroutine to prevent connection timeout
	// SSE comments (lines starting with :) are ignored by clients but keep the connection alive
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Send SSE comment as keepalive
				_, _ = fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	}()

	// Stream the response
	err = service.StreamMessage(ctx, sessionID, message, func(event copilot.StreamEvent) {
		h.logger.Debug("Received stream event", "type", event.Type, "session_id", sessionID)
		eventData := map[string]any{
			"type": string(event.Type),
		}

		switch event.Type {
		case copilot.StreamEventDelta:
			eventData["content"] = event.Data.Content
		case copilot.StreamEventToolCall:
			if event.Data.ToolCall != nil {
				eventData["tool_call"] = event.Data.ToolCall
			}
		case copilot.StreamEventToolResult:
			if event.Data.ToolResult != nil {
				eventData["tool_result"] = event.Data.ToolResult
			}
		case copilot.StreamEventDone:
			eventData["content"] = event.Data.Content
		case copilot.StreamEventError:
			eventData["error"] = event.Data.Error
		}

		h.writeSSEEvent(w, flusher, string(event.Type), eventData)
	})

	// Stop keepalive goroutine
	close(done)

	if err != nil {
		h.logger.Error("Stream error", "error", err, "session_id", sessionID)
		h.writeSSEEvent(w, flusher, "error", map[string]string{"error": err.Error()})
	}
}

// writeSSEEvent writes an SSE event to the response.
func (h *CopilotHandler) writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("Failed to marshal SSE event data", "error", err)
		return
	}

	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	flusher.Flush()
}

// ListSessions handles GET /api/v1/copilot/sessions
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
	service := h.getOrCreateService(settings)

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
	service := h.getOrCreateService(settings)

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
	service := h.getOrCreateService(settings)

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
