// Package copilot provides the Copilot chat service integration using the official SDK.
package copilot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/google/uuid"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Default configuration values
const (
	DefaultLogLevel = "info"
	DefaultModel    = "gpt-4.1"
)

// resolveCLIPath finds the Copilot CLI path using environment variable or well-known locations.
func resolveCLIPath() string {
	// Check environment variable first
	if envPath := os.Getenv("COPILOT_CLI_PATH"); envPath != "" {
		return envPath
	}

	// Try well-known paths in order of preference
	knownPaths := []string{
		"/usr/local/bin/copilot", // Docker/Linux standard
		"/usr/bin/copilot",       // System-wide install
		"copilot",                // In PATH (fallback)
	}

	for _, path := range knownPaths {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	// Return empty to let SDK use its default
	return ""
}

// Client wraps the Copilot SDK client and manages sessions.
type Client struct {
	sdkClient *copilot.Client
	db        *storage.Database
	logger    *slog.Logger
	tools     []copilot.Tool
	config    ClientConfig

	// Session management
	sessions   map[string]*SDKSession
	sessionsMu sync.RWMutex

	// Current auth context for tool execution
	// Set before sending messages, used by tools during execution
	currentAuth   *AuthContext
	currentAuthMu sync.RWMutex

	// Lifecycle
	started bool
	mu      sync.Mutex
}

// Authorization tier constants for AuthContext.Tier
const (
	AuthTierAdmin       = "admin"
	AuthTierSelfService = "self_service"
	AuthTierReadOnly    = "read_only"
)

// AuthContext carries authorization information for tool execution.
type AuthContext struct {
	UserID      string
	UserLogin   string
	Tier        string // AuthTierAdmin, AuthTierSelfService, or AuthTierReadOnly
	Permissions ToolPermissions
}

// ToolPermissions defines what tool categories the user can execute.
type ToolPermissions struct {
	CanRead           bool // Can use read-only tools (analytics, listings)
	CanMigrateOwn     bool // Can migrate repos they have admin access to
	CanMigrateAll     bool // Can migrate any repository
	CanManageSettings bool // Can modify system settings
}

// SDKSession wraps an SDK session with additional metadata.
type SDKSession struct {
	ID        string
	UserID    string
	UserLogin string
	Session   *copilot.Session
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	Auth      *AuthContext // Authorization context for this session
}

// ClientConfig holds configuration for the Copilot SDK client.
type ClientConfig struct {
	CLIPath           string
	CLIUrl            string // URL of existing CLI server (optional)
	Model             string
	LogLevel          string
	SessionTimeoutMin int
	Streaming         bool
}

// NewClient creates a new Copilot SDK client wrapper.
func NewClient(db *storage.Database, logger *slog.Logger, cfg ClientConfig) *Client {
	// Set defaults
	if cfg.LogLevel == "" {
		cfg.LogLevel = DefaultLogLevel
	}
	if cfg.SessionTimeoutMin == 0 {
		cfg.SessionTimeoutMin = 30
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}

	c := &Client{
		db:       db,
		logger:   logger,
		config:   cfg,
		sessions: make(map[string]*SDKSession),
	}

	return c
}

// Start initializes the SDK client and starts the CLI server.
func (c *Client) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	opts := &copilot.ClientOptions{
		LogLevel: c.config.LogLevel,
	}

	// Resolve CLI path using the same logic as CheckCLIAvailable
	cliPath := c.config.CLIPath
	if cliPath == "" {
		cliPath = resolveCLIPath()
	}
	if cliPath != "" {
		opts.CLIPath = cliPath
	}

	// Set CLI URL if connecting to external server
	if c.config.CLIUrl != "" {
		opts.CLIUrl = c.config.CLIUrl
	}

	c.sdkClient = copilot.NewClient(opts)

	if err := c.sdkClient.Start(); err != nil {
		return fmt.Errorf("failed to start Copilot SDK client: %w", err)
	}

	// Register tools after client is started
	c.registerTools()

	c.started = true
	c.logger.Info("Copilot SDK client started",
		"cli_path", c.config.CLIPath,
		"model", c.config.Model,
		"streaming", c.config.Streaming,
	)

	return nil
}

// Stop gracefully shuts down the SDK client.
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	// Destroy all active sessions
	c.sessionsMu.Lock()
	for _, sess := range c.sessions {
		if sess.Session != nil {
			_ = sess.Session.Destroy()
		}
	}
	c.sessions = make(map[string]*SDKSession)
	c.sessionsMu.Unlock()

	// Stop the SDK client
	errs := c.sdkClient.Stop()
	if len(errs) > 0 {
		c.logger.Error("Errors stopping Copilot SDK client", "errors", errs)
		return errs[0]
	}

	c.started = false
	c.logger.Info("Copilot SDK client stopped")

	return nil
}

// IsStarted returns whether the client is running.
func (c *Client) IsStarted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.started
}

// setCurrentAuth sets the auth context for the current message processing.
func (c *Client) setCurrentAuth(auth *AuthContext) {
	c.currentAuthMu.Lock()
	defer c.currentAuthMu.Unlock()
	c.currentAuth = auth
}

// clearCurrentAuth clears the auth context after message processing.
func (c *Client) clearCurrentAuth() {
	c.currentAuthMu.Lock()
	defer c.currentAuthMu.Unlock()
	c.currentAuth = nil
}

// getCurrentAuth returns the current auth context for tool authorization.
func (c *Client) getCurrentAuth() *AuthContext {
	c.currentAuthMu.RLock()
	defer c.currentAuthMu.RUnlock()
	return c.currentAuth
}

// CreateSession creates a new SDK session for a user with authorization context.
func (c *Client) CreateSession(ctx context.Context, userID, userLogin string, authCtx *AuthContext) (*SDKSession, error) {
	if !c.IsStarted() {
		if err := c.Start(); err != nil {
			return nil, err
		}
	}

	sessionID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(time.Duration(c.config.SessionTimeoutMin) * time.Minute)

	// Build system message with migration context and permissions
	systemMessage := c.buildSystemMessage(ctx, authCtx)

	// Create SDK session with tools
	sdkSession, err := c.sdkClient.CreateSession(&copilot.SessionConfig{
		Model:     c.config.Model,
		Streaming: c.config.Streaming,
		Tools:     c.tools,
		SystemMessage: &copilot.SystemMessageConfig{
			Content: systemMessage,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK session: %w", err)
	}

	sess := &SDKSession{
		ID:        sessionID,
		UserID:    userID,
		UserLogin: userLogin,
		Session:   sdkSession,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
		Auth:      authCtx,
	}

	// Store in memory
	c.sessionsMu.Lock()
	c.sessions[sessionID] = sess
	c.sessionsMu.Unlock()

	// Persist to database
	dbSession := &models.CopilotSession{
		ID:        sessionID,
		UserID:    userID,
		UserLogin: userLogin,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}
	if err := c.db.CreateCopilotSession(ctx, dbSession); err != nil {
		c.logger.Error("Failed to persist Copilot session", "error", err, "session_id", sessionID)
	}

	c.logger.Info("Created Copilot SDK session", "session_id", sessionID, "user", userLogin)

	return sess, nil
}

// GetSession retrieves a session by ID.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*SDKSession, error) {
	c.sessionsMu.RLock()
	sess, ok := c.sessions[sessionID]
	c.sessionsMu.RUnlock()

	if ok {
		if time.Now().After(sess.ExpiresAt) {
			return nil, fmt.Errorf("session expired")
		}
		return sess, nil
	}

	// Try to load from database and recreate SDK session
	dbSession, err := c.db.GetCopilotSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if dbSession == nil {
		return nil, fmt.Errorf("session not found")
	}

	if dbSession.IsExpired() {
		return nil, fmt.Errorf("session expired")
	}

	// Recreate SDK session (note: message history won't be preserved in SDK)
	if !c.IsStarted() {
		if err := c.Start(); err != nil {
			return nil, err
		}
	}

	// Auth context will be set by the handler when the session is used
	systemMessage := c.buildSystemMessage(ctx, nil)
	sdkSession, err := c.sdkClient.CreateSession(&copilot.SessionConfig{
		Model:     c.config.Model,
		Streaming: c.config.Streaming,
		Tools:     c.tools,
		SystemMessage: &copilot.SystemMessageConfig{
			Content: systemMessage,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to recreate SDK session: %w", err)
	}

	sess = &SDKSession{
		ID:        dbSession.ID,
		UserID:    dbSession.UserID,
		UserLogin: dbSession.UserLogin,
		Session:   sdkSession,
		CreatedAt: dbSession.CreatedAt,
		UpdatedAt: dbSession.UpdatedAt,
		ExpiresAt: dbSession.ExpiresAt,
		Auth:      nil, // Auth context will be set by handler when session is used
	}

	// Cache in memory
	c.sessionsMu.Lock()
	c.sessions[sessionID] = sess
	c.sessionsMu.Unlock()

	return sess, nil
}

// UpdateSessionAuth updates the authorization context for an existing session.
// This should be called by handlers to ensure the session uses the current user's permissions.
func (c *Client) UpdateSessionAuth(sessionID string, authCtx *AuthContext) {
	c.sessionsMu.Lock()
	defer c.sessionsMu.Unlock()

	if sess, ok := c.sessions[sessionID]; ok {
		sess.Auth = authCtx
	}
}

// DeleteSession removes a session.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	c.sessionsMu.Lock()
	if sess, ok := c.sessions[sessionID]; ok {
		if sess.Session != nil {
			_ = sess.Session.Destroy()
		}
		delete(c.sessions, sessionID)
	}
	c.sessionsMu.Unlock()

	if err := c.db.DeleteCopilotSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	c.logger.Info("Deleted Copilot session", "session_id", sessionID)
	return nil
}

// ListSessions returns all sessions for a user.
func (c *Client) ListSessions(ctx context.Context, userID string) ([]*models.CopilotSessionResponse, error) {
	sessions, err := c.db.ListCopilotSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	responses := make([]*models.CopilotSessionResponse, 0, len(sessions))
	for _, session := range sessions {
		responses = append(responses, session.ToResponse())
	}
	return responses, nil
}

// SendMessage sends a message and waits for the complete response.
func (c *Client) SendMessage(ctx context.Context, sessionID, message string) (*models.ChatResponse, error) {
	sess, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Set auth context for tool authorization during this message
	c.setCurrentAuth(sess.Auth)
	defer c.clearCurrentAuth()

	// Persist user message
	userMsg := &models.CopilotMessage{
		SessionID: sessionID,
		Role:      models.RoleUser,
		Content:   message,
		CreatedAt: time.Now(),
	}
	userMsgID, err := c.db.CreateCopilotMessage(ctx, userMsg)
	if err != nil {
		c.logger.Error("Failed to persist user message", "error", err)
	}

	// Send message and wait for response
	response, err := sess.Session.SendAndWait(copilot.MessageOptions{
		Prompt: message,
	}, 120000) // 2 minute timeout
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Extract content from response
	content := ""
	if response != nil && response.Data.Content != nil {
		content = *response.Data.Content
	}

	// Persist assistant message
	assistantMsg := &models.CopilotMessage{
		SessionID: sessionID,
		Role:      models.RoleAssistant,
		Content:   content,
		CreatedAt: time.Now(),
	}
	assistantMsgID, err := c.db.CreateCopilotMessage(ctx, assistantMsg)
	if err != nil {
		c.logger.Error("Failed to persist assistant message", "error", err)
	}

	// Update session timestamp
	sess.UpdatedAt = time.Now()

	c.logger.Debug("Message exchange completed",
		"session_id", sessionID,
		"user_msg_id", userMsgID,
		"assistant_msg_id", assistantMsgID,
	)

	return &models.ChatResponse{
		SessionID: sessionID,
		MessageID: assistantMsgID,
		Content:   content,
		Done:      true,
	}, nil
}

// StreamMessage sends a message and streams the response via a callback.
func (c *Client) StreamMessage(ctx context.Context, sessionID, message string, onEvent func(event StreamEvent)) error {
	sess, err := c.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Set auth context for tool authorization during this message
	c.setCurrentAuth(sess.Auth)
	defer c.clearCurrentAuth()

	// Persist user message
	userMsg := &models.CopilotMessage{
		SessionID: sessionID,
		Role:      models.RoleUser,
		Content:   message,
		CreatedAt: time.Now(),
	}
	if _, err := c.db.CreateCopilotMessage(ctx, userMsg); err != nil {
		c.logger.Error("Failed to persist user message", "error", err)
	}

	// Track accumulated content for persistence
	var fullContent string
	done := make(chan struct{})
	var closeOnce sync.Once

	// Set up event handler for streaming
	unsubscribe := sess.Session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				fullContent += *event.Data.DeltaContent
				onEvent(StreamEvent{
					Type: StreamEventDelta,
					Data: StreamEventData{
						Content: *event.Data.DeltaContent,
					},
				})
			}

		case "tool.execution_start":
			if event.Data.ToolName != nil {
				onEvent(StreamEvent{
					Type: StreamEventToolCall,
					Data: StreamEventData{
						ToolCall: &models.ToolCall{
							ID:     getStringOrDefault(event.Data.ToolCallID, ""),
							Name:   *event.Data.ToolName,
							Status: "pending",
						},
					},
				})
			}

		case "tool.execution_complete":
			onEvent(StreamEvent{
				Type: StreamEventToolResult,
				Data: StreamEventData{
					ToolResult: &models.ToolResult{
						ToolCallID: getStringOrDefault(event.Data.ToolCallID, ""),
						Success:    true,
					},
				},
			})

		case "assistant.message":
			// Final message received
			if event.Data.Content != nil {
				fullContent = *event.Data.Content
			}

		case "session.idle":
			onEvent(StreamEvent{
				Type: StreamEventDone,
				Data: StreamEventData{
					Content: fullContent,
				},
			})
			closeOnce.Do(func() { close(done) })

		case "error":
			errMsg := "Unknown error"
			if event.Data.Content != nil {
				errMsg = *event.Data.Content
			}
			onEvent(StreamEvent{
				Type: StreamEventError,
				Data: StreamEventData{
					Error: errMsg,
				},
			})
			closeOnce.Do(func() { close(done) })
		}
	})
	defer unsubscribe()

	// Send the message
	_, err = sess.Session.Send(copilot.MessageOptions{
		Prompt: message,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Wait for completion or context cancellation
	select {
	case <-done:
	case <-ctx.Done():
		_ = sess.Session.Abort()
		return ctx.Err()
	}

	// Persist assistant message
	assistantMsg := &models.CopilotMessage{
		SessionID: sessionID,
		Role:      models.RoleAssistant,
		Content:   fullContent,
		CreatedAt: time.Now(),
	}
	if _, err := c.db.CreateCopilotMessage(ctx, assistantMsg); err != nil {
		c.logger.Error("Failed to persist assistant message", "error", err)
	}

	sess.UpdatedAt = time.Now()

	return nil
}

// GetSessionHistory returns the message history for a session.
func (c *Client) GetSessionHistory(ctx context.Context, sessionID string) ([]models.CopilotMessage, error) {
	messages, err := c.db.GetCopilotMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	return messages, nil
}

// buildSystemMessage creates the system prompt with migration context.
func (c *Client) buildSystemMessage(ctx context.Context, authCtx *AuthContext) string {
	baseMessage := `You are the GitHub Migrator Copilot assistant. You help users plan and execute GitHub migrations.

You have access to tools that can:
- Analyze repositories for migration complexity and readiness
- Find and check dependencies between repositories
- Create and manage migration batches
- Plan migration waves to minimize downtime
- Identify repositories suitable for pilot migrations
- Get migration status and history
- Start, cancel, and monitor migrations

When users ask about migrations, use the appropriate tools to gather information and perform actions.
Be concise but thorough in your explanations. Present data in tables or lists when showing multiple items.
After performing actions, offer specific follow-up actions the user can take.`

	// Add authorization-specific guidance
	if authCtx != nil {
		switch authCtx.Tier {
		case "admin":
			baseMessage += `

You are interacting with an administrator who has full migration rights. They can:
- Start and manage migrations for any repository
- Create and modify batches
- Execute team migrations
- Modify system settings`
		case "self_service":
			baseMessage += `

You are interacting with a self-service user. They can:
- View analytics and migration status
- Migrate repositories where they have admin access on the source
- Create batches for their own repositories
NOTE: If they request operations on repositories they don't own, politely explain they need admin rights.`
		case "read_only":
			baseMessage += `

You are interacting with a read-only user. They can:
- View analytics, repository information, and migration status
- Get reports and audit information
NOTE: If they request migration operations, explain they need elevated permissions.`
		}
	}

	return baseMessage
}

// StreamEvent represents a streaming event sent to the client.
type StreamEvent struct {
	Type StreamEventType `json:"type"`
	Data StreamEventData `json:"data"`
}

// StreamEventType defines the type of streaming event.
type StreamEventType string

const (
	StreamEventDelta      StreamEventType = "delta"
	StreamEventToolCall   StreamEventType = "tool_call"
	StreamEventToolResult StreamEventType = "tool_result"
	StreamEventDone       StreamEventType = "done"
	StreamEventError      StreamEventType = "error"
)

// StreamEventData contains the data for a streaming event.
type StreamEventData struct {
	Content    string             `json:"content,omitempty"`
	ToolCall   *models.ToolCall   `json:"tool_call,omitempty"`
	ToolResult *models.ToolResult `json:"tool_result,omitempty"`
	Error      string             `json:"error,omitempty"`
}

// Helper function to safely get string value or default.
func getStringOrDefault(s *string, defaultVal string) string {
	if s != nil {
		return *s
	}
	return defaultVal
}
