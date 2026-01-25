package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
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
	intentDetector   *IntentDetector
	toolExecutor     *ToolExecutor
	sessions         map[string]*Session
	sessionsMu       sync.RWMutex
}

// Session represents an active Copilot chat session
type Session struct {
	ID             string
	UserID         string
	UserLogin      string
	Messages       []Message
	LastToolResult *ToolExecutionResult // Last tool execution result for follow-up actions
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      time.Time
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
	MCPEnabled        bool
	MCPPort           int
}

// NewService creates a new Copilot service
func NewService(db *storage.Database, logger *slog.Logger, config ServiceConfig) *Service {
	licenseValidator := NewLicenseValidator(config.GitHubBaseURL, logger)
	toolRegistry := NewToolRegistry(logger)
	intentDetector := NewIntentDetector()
	toolExecutor := NewToolExecutor(db, logger)

	return &Service{
		db:               db,
		logger:           logger,
		licenseValidator: licenseValidator,
		toolRegistry:     toolRegistry,
		intentDetector:   intentDetector,
		toolExecutor:     toolExecutor,
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

	// Add user message to session (with lock to prevent race conditions)
	userMsg := Message{
		Role:      models.RoleUser,
		Content:   userMessage,
		CreatedAt: time.Now(),
	}
	s.sessionsMu.Lock()
	session.Messages = append(session.Messages, userMsg)
	session.UpdatedAt = time.Now()
	// Make a copy of messages for processing (to release lock during CLI call)
	messagesCopy := make([]Message, len(session.Messages))
	copy(messagesCopy, session.Messages)
	s.sessionsMu.Unlock()

	// Persist user message
	userMsgModel := &models.CopilotMessage{
		SessionID: sessionID,
		Role:      models.RoleUser,
		Content:   userMessage,
		CreatedAt: userMsg.CreatedAt,
	}
	userMsgID, err := s.db.CreateCopilotMessage(ctx, userMsgModel)
	if err != nil {
		s.logger.Error("Failed to persist user message", "error", err)
		return nil, fmt.Errorf("failed to persist user message: %w", err)
	}

	// Process with Copilot using the copied messages (avoids holding lock during CLI call)
	// Create a temporary session view for processing - include LastToolResult for follow-ups
	sessionView := &Session{
		ID:             session.ID,
		UserID:         session.UserID,
		UserLogin:      session.UserLogin,
		Messages:       messagesCopy,
		LastToolResult: session.LastToolResult, // Carry over for follow-up detection
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
		ExpiresAt:      session.ExpiresAt,
	}
	response, err := s.processMessage(ctx, sessionView, userMessage, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to process message: %w", err)
	}

	// Add assistant response to session (with lock)
	assistantMsg := Message{
		Role:        models.RoleAssistant,
		Content:     response.Content,
		ToolCalls:   response.ToolCalls,
		ToolResults: response.ToolResults,
		CreatedAt:   time.Now(),
	}
	s.sessionsMu.Lock()
	session.Messages = append(session.Messages, assistantMsg)
	session.UpdatedAt = time.Now()
	// Preserve the LastToolResult from processing for future follow-ups
	if sessionView.LastToolResult != nil {
		session.LastToolResult = sessionView.LastToolResult
	}
	s.sessionsMu.Unlock()

	// Persist assistant message
	toolCallsJSON, err := json.Marshal(response.ToolCalls)
	if err != nil {
		s.logger.Error("Failed to marshal tool calls", "error", err)
		toolCallsJSON = nil
	}
	toolResultsJSON, err := json.Marshal(response.ToolResults)
	if err != nil {
		s.logger.Error("Failed to marshal tool results", "error", err)
		toolResultsJSON = nil
	}
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
		return nil, fmt.Errorf("failed to persist assistant message: %w", err)
	}

	s.logger.Debug("Messages persisted successfully",
		"session_id", sessionID,
		"user_msg_id", userMsgID,
		"assistant_msg_id", assistantMsgID)

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
	// Get CLI path from settings
	cliPath := "copilot"
	if settings.CopilotCLIPath != nil && *settings.CopilotCLIPath != "" {
		cliPath = *settings.CopilotCLIPath
	}

	// Detect intent from user message
	var toolResult *ToolExecutionResult
	intent := s.intentDetector.DetectIntent(userMessage)

	// Check for follow-up batch creation
	if s.intentDetector.IsFollowUpBatchCreate(userMessage) && session.LastToolResult != nil && session.LastToolResult.FollowUp != nil {
		// Extract batch name from message or use default
		batchName := s.intentDetector.ExtractBatchNameFromFollowUp(userMessage)
		if batchName == "" {
			batchName = session.LastToolResult.FollowUp.DefaultName
		}

		// Create batch from previous results
		batchIntent := &DetectedIntent{
			Tool: "create_batch",
			Args: map[string]any{
				"name": batchName,
			},
			Confidence: 1.0,
		}
		result, err := s.toolExecutor.ExecuteTool(ctx, batchIntent, session.LastToolResult)
		if err != nil {
			s.logger.Error("Failed to execute batch creation", "error", err)
		} else {
			toolResult = result
		}
	} else if intent != nil && intent.IsConfident() {
		// Execute the detected tool
		s.logger.Info("Detected intent", "tool", intent.Tool, "confidence", intent.Confidence)
		result, err := s.toolExecutor.ExecuteTool(ctx, intent, session.LastToolResult)
		if err != nil {
			s.logger.Error("Failed to execute tool", "tool", intent.Tool, "error", err)
		} else {
			toolResult = result
		}
	}

	// Store the tool result for potential follow-ups
	if toolResult != nil {
		session.LastToolResult = toolResult
	}

	// Fetch migration context from database
	migrationContext := s.getMigrationContext(ctx, settings)

	// Build the system prompt with context about available tools and environment
	tools := s.toolRegistry.GetTools()
	systemPrompt := s.buildSystemPrompt(tools, migrationContext)

	// If we have tool results, enhance the prompt with them
	if toolResult != nil {
		systemPrompt += s.formatToolResultsForPrompt(toolResult)
	}

	// Get MCP configuration from settings
	mcpEnabled := settings.CopilotMCPEnabled
	mcpPort := settings.CopilotMCPPort
	if mcpPort == 0 {
		mcpPort = 8081 // Default port
	}

	// Try to call the Copilot CLI with MCP configuration
	response, err := s.callCopilotCLI(ctx, cliPath, systemPrompt, session.Messages, userMessage, mcpEnabled, mcpPort)
	if err != nil {
		s.logger.Error("Failed to call Copilot CLI, using fallback response", "error", err)
		// If we have tool results, generate a response from them instead
		if toolResult != nil {
			response = s.generateResponseFromToolResult(toolResult)
		} else {
			response = s.generateFallbackResponse(userMessage, err)
		}
	}

	// Build tool calls and results for the message
	var toolCalls []ToolCall
	var toolResults []ToolResult
	if toolResult != nil {
		toolCalls = []ToolCall{{
			ID:     fmt.Sprintf("call_%d", time.Now().UnixNano()),
			Name:   toolResult.Tool,
			Args:   intent.Args,
			Status: "completed",
		}}
		toolResults = []ToolResult{{
			ToolCallID: toolCalls[0].ID,
			Success:    toolResult.Success,
			Result:     toolResult.Result,
			Error:      toolResult.Error,
		}}
	}

	return &Message{
		Role:        models.RoleAssistant,
		Content:     response,
		ToolCalls:   toolCalls,
		ToolResults: toolResults,
		CreatedAt:   time.Now(),
	}, nil
}

// MigrationContext holds context about the current migration environment
type MigrationContext struct {
	// Source info
	SourceType string
	SourceURL  string
	SourceOrgs []string

	// Destination info
	DestinationURL            string
	DestinationEnterpriseSlug string

	// Repository stats
	TotalRepositories  int
	PendingRepos       int
	InProgressRepos    int
	CompletedRepos     int
	FailedRepos        int
	AvgComplexityScore float64

	// Batch info
	TotalBatches     int
	PendingBatches   int
	ScheduledBatches int
}

// getMigrationContext fetches the current migration context from the database
func (s *Service) getMigrationContext(ctx context.Context, settings *models.Settings) *MigrationContext {
	mc := &MigrationContext{}

	// Get destination configuration from settings
	mc.DestinationURL = settings.DestinationBaseURL
	if settings.DestinationEnterpriseSlug != nil {
		mc.DestinationEnterpriseSlug = *settings.DestinationEnterpriseSlug
	}

	// Get source organizations from repositories
	orgs, err := s.db.GetDistinctOrganizations(ctx)
	if err == nil {
		mc.SourceOrgs = orgs
	}

	// Get repository statistics by status
	stats, err := s.db.GetRepositoryStatsByStatus(ctx)
	if err == nil {
		for status, count := range stats {
			mc.TotalRepositories += count
			switch status {
			case "pending", "not_started":
				mc.PendingRepos += count
			case "in_progress", "queued", "exporting", "importing":
				mc.InProgressRepos += count
			case "completed", "complete", "migration_complete":
				mc.CompletedRepos += count
			case "failed", "error":
				mc.FailedRepos += count
			}
		}
	}

	// Get batch counts
	batches, err := s.db.ListBatches(ctx)
	if err == nil {
		mc.TotalBatches = len(batches)
		for _, batch := range batches {
			switch batch.Status {
			case "pending":
				mc.PendingBatches++
			case "scheduled":
				mc.ScheduledBatches++
			}
		}
	}

	return mc
}

// buildSystemPrompt creates the system prompt with tool descriptions and environment context
func (s *Service) buildSystemPrompt(tools []ToolDescription, mc *MigrationContext) string {
	var prompt strings.Builder

	prompt.WriteString(`You are the GitHub Migrator Copilot assistant. You help users plan and execute GitHub migrations.

You have access to the following capabilities:
- Analyze repositories for migration complexity and readiness
- Find and check dependencies between repositories
- Create and manage migration batches
- Plan migration waves to minimize downtime
- Identify repositories suitable for pilot migrations
- Get migration status and history

`)

	// Add environment context
	prompt.WriteString("## Current Migration Environment\n\n")

	// Source info
	if len(mc.SourceOrgs) > 0 {
		prompt.WriteString("**Source Organizations:** ")
		if len(mc.SourceOrgs) <= 5 {
			prompt.WriteString(strings.Join(mc.SourceOrgs, ", "))
		} else {
			prompt.WriteString(fmt.Sprintf("%s, and %d more", strings.Join(mc.SourceOrgs[:5], ", "), len(mc.SourceOrgs)-5))
		}
		prompt.WriteString("\n")
	}

	// Destination info
	if mc.DestinationURL != "" {
		prompt.WriteString(fmt.Sprintf("**Destination:** %s", mc.DestinationURL))
		if mc.DestinationEnterpriseSlug != "" {
			prompt.WriteString(fmt.Sprintf(" (Enterprise: %s)", mc.DestinationEnterpriseSlug))
		}
		prompt.WriteString("\n")
	}

	// Repository stats
	if mc.TotalRepositories > 0 {
		prompt.WriteString(fmt.Sprintf("\n**Repository Summary:** %d total repositories\n", mc.TotalRepositories))
		prompt.WriteString(fmt.Sprintf("- Pending: %d\n", mc.PendingRepos))
		if mc.InProgressRepos > 0 {
			prompt.WriteString(fmt.Sprintf("- In Progress: %d\n", mc.InProgressRepos))
		}
		if mc.CompletedRepos > 0 {
			prompt.WriteString(fmt.Sprintf("- Completed: %d\n", mc.CompletedRepos))
		}
		if mc.FailedRepos > 0 {
			prompt.WriteString(fmt.Sprintf("- Failed: %d\n", mc.FailedRepos))
		}
	}

	// Batch info
	if mc.TotalBatches > 0 {
		prompt.WriteString(fmt.Sprintf("\n**Batches:** %d total", mc.TotalBatches))
		if mc.PendingBatches > 0 {
			prompt.WriteString(fmt.Sprintf(", %d pending", mc.PendingBatches))
		}
		if mc.ScheduledBatches > 0 {
			prompt.WriteString(fmt.Sprintf(", %d scheduled", mc.ScheduledBatches))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("\n")

	// Instructions for using context
	prompt.WriteString(`## Response Guidelines

When answering questions:
- Use the migration context above to provide specific, actionable responses
- Present data in tables or lists when showing multiple items
- Don't ask clarifying questions if you can answer from context
- Be concise but thorough in your explanations

When presenting tool results:
- Format data in a clear, readable way (use markdown tables for lists)
- Offer specific follow-up actions the user can take
- For pilot candidates: Offer to create a batch with the selected repositories
- For batches: Offer to schedule them for migration
- For dependencies: Explain the migration implications
- Always suggest a natural next step the user can take

Example follow-up offers:
- "Would you like me to create a batch with these repositories?"
- "Should I schedule this batch for migration?"
- "Would you like to see more details about any of these repositories?"

`)

	// Add tool descriptions
	if len(tools) > 0 {
		prompt.WriteString("## Available Tools\n\n")
		for _, tool := range tools {
			prompt.WriteString(fmt.Sprintf("- **%s**: %s\n", tool.Name, tool.Description))
		}
	}

	return prompt.String()
}

// callCopilotCLI calls the Copilot CLI with the given message
func (s *Service) callCopilotCLI(ctx context.Context, cliPath, systemPrompt string, history []Message, userMessage string, mcpEnabled bool, mcpPort int) (string, error) {
	// Create a timeout context for the CLI call
	cmdCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Build the full prompt with system context and history
	var promptBuilder strings.Builder
	promptBuilder.WriteString("You are a GitHub migration assistant. ")
	promptBuilder.WriteString(systemPrompt)
	promptBuilder.WriteString("\n\n")

	// Include conversation history for context
	if len(history) > 0 {
		promptBuilder.WriteString("Previous conversation:\n")
		// Include last few messages for context (limit to avoid token limits)
		startIdx := 0
		if len(history) > 6 {
			startIdx = len(history) - 6
		}
		for _, msg := range history[startIdx:] {
			role := "User"
			if msg.Role == "assistant" {
				role = "Assistant"
			}
			promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
		}
		promptBuilder.WriteString("\n")
	}

	promptBuilder.WriteString("Current request: ")
	promptBuilder.WriteString(userMessage)
	promptBuilder.WriteString("\n\nProvide a helpful response for this GitHub migration question. Be concise and actionable.")

	fullPrompt := promptBuilder.String()

	s.logger.Debug("Calling Copilot CLI", "cli_path", cliPath, "prompt_length", len(fullPrompt), "mcp_enabled", mcpEnabled, "mcp_port", mcpPort)

	// Build command arguments
	// Note: The Copilot CLI doesn't support --mcp-config flag directly.
	// The MCP server runs separately and can be used by MCP-compatible clients.
	// For the CLI, we pass the prompt with context that describes the available tools.
	args := []string{"-p", fullPrompt}

	// #nosec G204 - CLI path is configured by admin in settings, not user-controlled input
	cmd := exec.CommandContext(cmdCtx, cliPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		s.logger.Error("Copilot CLI execution failed",
			"error", err,
			"output", string(output),
			"cli_path", cliPath)
		return "", fmt.Errorf("copilot CLI execution failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	response := strings.TrimSpace(string(output))
	if response == "" {
		return "", fmt.Errorf("copilot CLI returned empty response")
	}

	s.logger.Debug("Copilot CLI response received", "response_length", len(response))
	return response, nil
}

// formatToolResultsForPrompt formats tool execution results for inclusion in the system prompt
func (s *Service) formatToolResultsForPrompt(result *ToolExecutionResult) string {
	var prompt strings.Builder

	prompt.WriteString("\n\n## Tool Execution Results\n\n")
	prompt.WriteString(fmt.Sprintf("I executed the **%s** tool and got the following results:\n\n", result.Tool))

	if result.Success {
		prompt.WriteString(fmt.Sprintf("**Summary:** %s\n\n", result.Summary))

		// Format the result based on the tool type
		if data, err := json.MarshalIndent(result.Result, "", "  "); err == nil {
			prompt.WriteString("**Data:**\n```json\n")
			prompt.WriteString(string(data))
			prompt.WriteString("\n```\n\n")
		}

		// Add suggestions if present
		if len(result.Suggestions) > 0 {
			prompt.WriteString("**Notes:**\n")
			for _, s := range result.Suggestions {
				prompt.WriteString(fmt.Sprintf("- %s\n", s))
			}
			prompt.WriteString("\n")
		}

		// Add follow-up action if available
		if result.FollowUp != nil {
			prompt.WriteString(fmt.Sprintf("**Suggested Follow-up:** %s\n\n", result.FollowUp.Description))
		}
	} else {
		prompt.WriteString(fmt.Sprintf("**Error:** %s\n\n", result.Error))
	}

	prompt.WriteString(`**Instructions for Response:**
- Present these results clearly to the user in a readable format
- If there are candidates/repositories, show them in a table or list
- Offer the follow-up action if one is suggested
- Be specific about what the user can do next
`)

	return prompt.String()
}

// generateResponseFromToolResult creates a response directly from tool results when CLI fails
func (s *Service) generateResponseFromToolResult(result *ToolExecutionResult) string {
	var response strings.Builder

	if !result.Success {
		response.WriteString(fmt.Sprintf("I encountered an issue: %s\n\n", result.Error))
		return response.String()
	}

	response.WriteString(fmt.Sprintf("%s\n\n", result.Summary))

	// Format results based on tool type
	switch result.Tool {
	case "find_pilot_candidates":
		if candidates, ok := result.Result.([]map[string]any); ok && len(candidates) > 0 {
			response.WriteString("| Repository | Complexity | Size |\n")
			response.WriteString("|------------|------------|------|\n")
			for _, c := range candidates {
				response.WriteString(fmt.Sprintf("| %s | %s (%v) | %v KB |\n",
					c["full_name"], c["complexity_rating"], c["complexity_score"], c["size_kb"]))
			}
			response.WriteString("\n")
		}

	case "analyze_repositories":
		if repos, ok := result.Result.([]map[string]any); ok && len(repos) > 0 {
			response.WriteString("| Repository | Status | Complexity |\n")
			response.WriteString("|------------|--------|------------|\n")
			for _, r := range repos {
				response.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
					r["full_name"], r["status"], r["complexity_rating"]))
			}
			response.WriteString("\n")
		}

	case "create_batch":
		if batch, ok := result.Result.(map[string]any); ok {
			response.WriteString("**Batch Details:**\n")
			response.WriteString(fmt.Sprintf("- Name: %s\n", batch["batch_name"]))
			response.WriteString(fmt.Sprintf("- ID: %v\n", batch["batch_id"]))
			response.WriteString(fmt.Sprintf("- Repositories: %v\n", batch["repository_count"]))
			response.WriteString(fmt.Sprintf("- Status: %s\n\n", batch["status"]))
		}

	case "plan_waves":
		if waves, ok := result.Result.([]map[string]any); ok {
			for _, w := range waves {
				response.WriteString(fmt.Sprintf("**Wave %v** (%v repos):\n", w["wave_number"], w["count"]))
				if repos, ok := w["repositories"].([]string); ok {
					for _, r := range repos {
						response.WriteString(fmt.Sprintf("- %s\n", r))
					}
				}
				response.WriteString("\n")
			}
		}

	default:
		// Generic JSON output
		if data, err := json.MarshalIndent(result.Result, "", "  "); err == nil {
			response.WriteString("```json\n")
			response.WriteString(string(data))
			response.WriteString("\n```\n\n")
		}
	}

	// Add suggestions
	if len(result.Suggestions) > 0 {
		response.WriteString("**Notes:**\n")
		for _, s := range result.Suggestions {
			response.WriteString(fmt.Sprintf("- %s\n", s))
		}
		response.WriteString("\n")
	}

	// Add follow-up action
	if result.FollowUp != nil {
		response.WriteString(fmt.Sprintf("\n%s\n", result.FollowUp.Description))
	}

	return response.String()
}

// generateFallbackResponse creates a helpful response when CLI fails
func (s *Service) generateFallbackResponse(userMessage string, cliErr error) string {
	response := fmt.Sprintf("I received your message: %q\n\n", userMessage)
	response += "I'm the GitHub Migrator Copilot assistant. I can help you with:\n"
	response += "- Analyzing repositories for migration readiness\n"
	response += "- Finding dependencies between repositories\n"
	response += "- Creating and managing migration batches\n"
	response += "- Planning migration waves\n"
	response += "- Identifying pilot candidates\n\n"

	response += "**Note**: I encountered an issue communicating with the Copilot CLI.\n"
	response += fmt.Sprintf("Error details: %v\n\n", cliErr)
	response += "Please verify:\n"
	response += "1. The Copilot CLI is correctly installed at the configured path\n"
	response += "2. You are authenticated with the Copilot CLI (run `copilot auth login`)\n"
	response += "3. Your GitHub Copilot license is active\n\n"
	response += "In the meantime, you can use the tool-specific features in the application:\n"
	response += "- View repository complexity on the Repositories page\n"
	response += "- Check dependencies on the Dependencies page\n"
	response += "- Create and manage batches on the Batches page"

	return response
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
