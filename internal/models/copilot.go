package models

import (
	"encoding/json"
	"time"
)

// CopilotSession represents a chat session with the Copilot assistant
type CopilotSession struct {
	ID        string    `json:"id" gorm:"primaryKey;column:id"`
	UserID    string    `json:"user_id" gorm:"column:user_id;not null"`
	UserLogin string    `json:"user_login" gorm:"column:user_login;not null"`
	Title     *string   `json:"title,omitempty" gorm:"column:title"`
	Model     *string   `json:"model,omitempty" gorm:"column:model"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
	ExpiresAt time.Time `json:"expires_at" gorm:"column:expires_at"`

	// Messages in this session (for eager loading)
	Messages []CopilotMessage `json:"messages,omitempty" gorm:"foreignKey:SessionID;references:ID"`
}

// TableName returns the table name for GORM
func (CopilotSession) TableName() string {
	return "copilot_sessions"
}

// IsExpired returns true if the session has expired
func (s *CopilotSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// CopilotMessage represents a single message in a Copilot chat session
type CopilotMessage struct {
	ID          int64           `json:"id" gorm:"primaryKey;column:id"`
	SessionID   string          `json:"session_id" gorm:"column:session_id;not null"`
	Role        string          `json:"role" gorm:"column:role;not null"` // "user", "assistant", "system"
	Content     string          `json:"content" gorm:"column:content;not null"`
	ToolCalls   json.RawMessage `json:"tool_calls,omitempty" gorm:"column:tool_calls;type:jsonb"`
	ToolResults json.RawMessage `json:"tool_results,omitempty" gorm:"column:tool_results;type:jsonb"`
	CreatedAt   time.Time       `json:"created_at" gorm:"column:created_at"`
}

// TableName returns the table name for GORM
func (CopilotMessage) TableName() string {
	return "copilot_messages"
}

// MessageRole constants
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// ToolCall represents a tool invocation by the assistant
type ToolCall struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Args     map[string]any `json:"args"`
	Status   string         `json:"status"` // "pending", "completed", "failed"
	Duration int64          `json:"duration_ms,omitempty"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Success    bool   `json:"success"`
	Result     any    `json:"result,omitempty"`
	Error      string `json:"error,omitempty"`
}

// CopilotSessionResponse is the API response for a session
type CopilotSessionResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// ToResponse converts a CopilotSession to an API response
func (s *CopilotSession) ToResponse() *CopilotSessionResponse {
	title := ""
	if s.Title != nil {
		title = *s.Title
	}
	return &CopilotSessionResponse{
		ID:           s.ID,
		Title:        title,
		MessageCount: len(s.Messages),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
		ExpiresAt:    s.ExpiresAt,
	}
}

// CopilotStatus represents the current status of Copilot availability
type CopilotStatus struct {
	Enabled           bool   `json:"enabled"`
	Available         bool   `json:"available"`
	CLIInstalled      bool   `json:"cli_installed"`
	CLIVersion        string `json:"cli_version,omitempty"`
	LicenseRequired   bool   `json:"license_required"`
	LicenseValid      bool   `json:"license_valid"`
	LicenseMessage    string `json:"license_message,omitempty"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
}

// ChatRequest represents a request to send a message to Copilot
type ChatRequest struct {
	SessionID string  `json:"session_id,omitempty"` // Empty to create new session
	Message   string  `json:"message"`
	Model     *string `json:"model,omitempty"` // Optional model override for this request
}

// ModelInfo represents an available AI model
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// ModelsResponse is the API response for listing models
type ModelsResponse struct {
	Models       []ModelInfo `json:"models"`
	DefaultModel string      `json:"default_model"`
}

// ChatResponse represents a response from Copilot
type ChatResponse struct {
	SessionID   string       `json:"session_id"`
	MessageID   int64        `json:"message_id"`
	Content     string       `json:"content"`
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`
	Done        bool         `json:"done"`
}
