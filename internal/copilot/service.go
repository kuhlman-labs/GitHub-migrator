package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Service manages Copilot interactions and sessions
type Service struct {
	db               *storage.Database
	logger           *slog.Logger
	licenseValidator *LicenseValidator
	toolRegistry     *ToolRegistry
	sessions         map[string]*Session
	sessionsMu       sync.RWMutex
}

// Session represents an active Copilot chat session
type Session struct {
	ID        string
	UserID    string
	UserLogin string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

// Message represents a chat message
type Message struct {
	Role        string       `json:"role"` // "user", "assistant", "system"
	Content     string       `json:"content"`
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Args   map[string]any `json:"args"`
	Status string         `json:"status"` // "pending", "completed", "failed"
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Success    bool   `json:"success"`
	Result     any    `json:"result,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ServiceConfig configures the Copilot service
type ServiceConfig struct {
	CLIPath           string
	Model             string
	MaxTokens         int
	SessionTimeoutMin int
	RequireLicense    bool
	GitHubBaseURL     string
}

// NewService creates a new Copilot service
func NewService(db *storage.Database, logger *slog.Logger, config ServiceConfig) *Service {
	licenseValidator := NewLicenseValidator(config.GitHubBaseURL, logger)
	toolRegistry := NewToolRegistry(db, logger)

	return &Service{
		db:               db,
		logger:           logger,
		licenseValidator: licenseValidator,
		toolRegistry:     toolRegistry,
		sessions:         make(map[string]*Session),
	}
}

// GetStatus returns the current Copilot status for a user
func (s *Service) GetStatus(ctx context.Context, userLogin string, token string, settings *models.Settings) (*models.CopilotStatus, error) {
	status := &models.CopilotStatus{
		Enabled:         settings.CopilotEnabled,
		LicenseRequired: settings.CopilotRequireLicense,
	}

	// Check if Copilot is enabled in settings
	if !settings.CopilotEnabled {
		status.Available = false
		status.UnavailableReason = "Copilot is not enabled in settings"
		return status, nil
	}

	// Check CLI availability
	cliPath := ""
	if settings.CopilotCLIPath != nil {
		cliPath = *settings.CopilotCLIPath
	}
	cliInstalled, cliVersion, cliErr := CheckCLIAvailable(cliPath)
	status.CLIInstalled = cliInstalled
	status.CLIVersion = cliVersion

	if !cliInstalled {
		status.Available = false
		if cliErr != nil {
			status.UnavailableReason = cliErr.Error()
		} else {
			status.UnavailableReason = "Copilot CLI is not installed or not accessible"
		}
		return status, nil
	}

	// Check license if required
	if settings.CopilotRequireLicense {
		licenseStatus, err := s.licenseValidator.CheckLicense(ctx, userLogin, token)
		if err != nil {
			s.logger.Error("Failed to check Copilot license", "error", err, "user", userLogin)
			status.LicenseValid = false
			status.LicenseMessage = "Failed to verify license"
		} else {
			status.LicenseValid = licenseStatus.Valid
			status.LicenseMessage = licenseStatus.Message
		}

		if !status.LicenseValid {
			status.Available = false
			status.UnavailableReason = status.LicenseMessage
			return status, nil
		}
	} else {
		// License not required, mark as valid
		status.LicenseValid = true
		status.LicenseMessage = "License validation disabled"
	}

	// All checks passed
	status.Available = true
	return status, nil
}

// CreateSession creates a new chat session
func (s *Service) CreateSession(ctx context.Context, userID, userLogin string, timeoutMin int) (*Session, error) {
	sessionID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(time.Duration(timeoutMin) * time.Minute)

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		UserLogin: userLogin,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}

	// Store in memory
	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	// Also persist to database
	dbSession := &models.CopilotSession{
		ID:        sessionID,
		UserID:    userID,
		UserLogin: userLogin,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}
	if err := s.db.CreateCopilotSession(ctx, dbSession); err != nil {
		s.logger.Error("Failed to persist Copilot session", "error", err, "session_id", sessionID)
		// Continue anyway - in-memory session is still valid
	}

	s.logger.Info("Created Copilot session", "session_id", sessionID, "user", userLogin)
	return session, nil
}

// GetSession retrieves a session by ID
func (s *Service) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	s.sessionsMu.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionsMu.RUnlock()

	if ok {
		if time.Now().After(session.ExpiresAt) {
			return nil, fmt.Errorf("session expired")
		}
		return session, nil
	}

	// Try to load from database
	dbSession, err := s.db.GetCopilotSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if dbSession == nil {
		return nil, fmt.Errorf("session not found")
	}

	if dbSession.IsExpired() {
		return nil, fmt.Errorf("session expired")
	}

	// Reconstruct session
	session = &Session{
		ID:        dbSession.ID,
		UserID:    dbSession.UserID,
		UserLogin: dbSession.UserLogin,
		Messages:  make([]Message, 0, len(dbSession.Messages)),
		CreatedAt: dbSession.CreatedAt,
		UpdatedAt: dbSession.UpdatedAt,
		ExpiresAt: dbSession.ExpiresAt,
	}

	// Convert messages
	for _, msg := range dbSession.Messages {
		message := Message{
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		}
		if msg.ToolCalls != nil {
			_ = json.Unmarshal(msg.ToolCalls, &message.ToolCalls)
		}
		if msg.ToolResults != nil {
			_ = json.Unmarshal(msg.ToolResults, &message.ToolResults)
		}
		session.Messages = append(session.Messages, message)
	}

	// Cache in memory
	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	return session, nil
}

// ListSessions returns all sessions for a user
func (s *Service) ListSessions(ctx context.Context, userID string) ([]*models.CopilotSessionResponse, error) {
	sessions, err := s.db.ListCopilotSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	responses := make([]*models.CopilotSessionResponse, 0, len(sessions))
	for _, session := range sessions {
		responses = append(responses, session.ToResponse())
	}
	return responses, nil
}

// DeleteSession deletes a session
func (s *Service) DeleteSession(ctx context.Context, sessionID string) error {
	// Remove from memory
	s.sessionsMu.Lock()
	delete(s.sessions, sessionID)
	s.sessionsMu.Unlock()

	// Remove from database
	if err := s.db.DeleteCopilotSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	s.logger.Info("Deleted Copilot session", "session_id", sessionID)
	return nil
}

// SendMessage sends a message to Copilot and returns the response
func (s *Service) SendMessage(ctx context.Context, sessionID, userMessage string, settings *models.Settings) (*models.ChatResponse, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Add user message to session
	userMsg := Message{
		Role:      models.RoleUser,
		Content:   userMessage,
		CreatedAt: time.Now(),
	}
	session.Messages = append(session.Messages, userMsg)
	session.UpdatedAt = time.Now()

	// Persist user message
	userMsgModel := &models.CopilotMessage{
		SessionID: sessionID,
		Role:      models.RoleUser,
		Content:   userMessage,
		CreatedAt: userMsg.CreatedAt,
	}
	msgID, err := s.db.CreateCopilotMessage(ctx, userMsgModel)
	if err != nil {
		s.logger.Error("Failed to persist user message", "error", err)
	}

	// Process with Copilot
	// For now, we'll simulate a response since the SDK integration requires the CLI
	response, err := s.processMessage(ctx, session, userMessage, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to process message: %w", err)
	}

	// Add assistant response to session
	assistantMsg := Message{
		Role:        models.RoleAssistant,
		Content:     response.Content,
		ToolCalls:   response.ToolCalls,
		ToolResults: response.ToolResults,
		CreatedAt:   time.Now(),
	}
	session.Messages = append(session.Messages, assistantMsg)
	session.UpdatedAt = time.Now()

	// Persist assistant message
	toolCallsJSON, _ := json.Marshal(response.ToolCalls)
	toolResultsJSON, _ := json.Marshal(response.ToolResults)
	assistantMsgModel := &models.CopilotMessage{
		SessionID:   sessionID,
		Role:        models.RoleAssistant,
		Content:     response.Content,
		ToolCalls:   toolCallsJSON,
		ToolResults: toolResultsJSON,
		CreatedAt:   assistantMsg.CreatedAt,
	}
	assistantMsgID, err := s.db.CreateCopilotMessage(ctx, assistantMsgModel)
	if err != nil {
		s.logger.Error("Failed to persist assistant message", "error", err)
	}

	_ = msgID // silence unused variable warning

	return &models.ChatResponse{
		SessionID:   sessionID,
		MessageID:   assistantMsgID,
		Content:     response.Content,
		ToolCalls:   convertToolCalls(response.ToolCalls),
		ToolResults: convertToolResults(response.ToolResults),
		Done:        true,
	}, nil
}

// processMessage processes a user message and generates a response
func (s *Service) processMessage(ctx context.Context, session *Session, userMessage string, settings *models.Settings) (*Message, error) {
	// Build the system prompt with available tools
	tools := s.toolRegistry.GetTools()

	// For now, we'll use a simple pattern matching approach
	// In a full implementation, this would call the Copilot SDK
	response, toolCalls, toolResults := s.generateResponse(ctx, userMessage, tools)

	return &Message{
		Role:        models.RoleAssistant,
		Content:     response,
		ToolCalls:   toolCalls,
		ToolResults: toolResults,
		CreatedAt:   time.Now(),
	}, nil
}

// generateResponse generates a response based on the user message
// This is a placeholder that will be replaced with actual Copilot SDK integration
func (s *Service) generateResponse(ctx context.Context, userMessage string, tools []Tool) (string, []ToolCall, []ToolResult) {
	// This is a simplified implementation that demonstrates the tool-calling pattern
	// In production, this would be replaced with actual Copilot SDK calls

	response := fmt.Sprintf("I received your message: %q\n\nI'm the GitHub Migrator Copilot assistant. ", userMessage)
	response += "I can help you with:\n"
	response += "- Analyzing repositories for migration readiness\n"
	response += "- Finding dependencies between repositories\n"
	response += "- Creating and managing migration batches\n"
	response += "- Planning migration waves\n"
	response += "- Identifying pilot candidates\n\n"
	response += "**Note**: Full Copilot SDK integration is pending. Please ensure the Copilot CLI is installed and configured in settings."

	return response, nil, nil
}

// GetSessionHistory returns the message history for a session
func (s *Service) GetSessionHistory(ctx context.Context, sessionID string) ([]models.CopilotMessage, error) {
	messages, err := s.db.GetCopilotMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	return messages, nil
}

// Helper functions to convert between types
func convertToolCalls(calls []ToolCall) []models.ToolCall {
	result := make([]models.ToolCall, len(calls))
	for i, c := range calls {
		result[i] = models.ToolCall{
			ID:     c.ID,
			Name:   c.Name,
			Args:   c.Args,
			Status: c.Status,
		}
	}
	return result
}

func convertToolResults(results []ToolResult) []models.ToolResult {
	result := make([]models.ToolResult, len(results))
	for i, r := range results {
		result[i] = models.ToolResult{
			ToolCallID: r.ToolCallID,
			Success:    r.Success,
			Result:     r.Result,
			Error:      r.Error,
		}
	}
	return result
}
