package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// CopilotStore defines operations for Copilot sessions and messages
type CopilotStore interface {
	CreateCopilotSession(ctx context.Context, session *models.CopilotSession) error
	GetCopilotSession(ctx context.Context, id string) (*models.CopilotSession, error)
	ListCopilotSessions(ctx context.Context, userID string) ([]*models.CopilotSession, error)
	UpdateCopilotSession(ctx context.Context, session *models.CopilotSession) error
	DeleteCopilotSession(ctx context.Context, id string) error
	CreateCopilotMessage(ctx context.Context, message *models.CopilotMessage) (int64, error)
	GetCopilotMessages(ctx context.Context, sessionID string) ([]models.CopilotMessage, error)
	CleanupExpiredSessions(ctx context.Context) (int64, error)
}

// CreateCopilotSession creates a new Copilot chat session
func (d *Database) CreateCopilotSession(ctx context.Context, session *models.CopilotSession) error {
	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}
	if session.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.UpdatedAt.IsZero() {
		session.UpdatedAt = now
	}

	result := d.db.WithContext(ctx).Create(session)
	if result.Error != nil {
		return fmt.Errorf("failed to create Copilot session: %w", result.Error)
	}
	return nil
}

// GetCopilotSession retrieves a Copilot session by ID
func (d *Database) GetCopilotSession(ctx context.Context, id string) (*models.CopilotSession, error) {
	var session models.CopilotSession
	result := d.db.WithContext(ctx).
		Preload("Messages").
		Where("id = ?", id).
		First(&session)

	if result.Error != nil {
		if isNotFoundError(result.Error) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get Copilot session: %w", result.Error)
	}
	return &session, nil
}

// ListCopilotSessions returns all sessions for a user
func (d *Database) ListCopilotSessions(ctx context.Context, userID string) ([]*models.CopilotSession, error) {
	var sessions []*models.CopilotSession
	result := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("expires_at > ?", time.Now()).
		Order("updated_at DESC").
		Find(&sessions)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list Copilot sessions: %w", result.Error)
	}

	// Get message counts for each session
	for _, session := range sessions {
		var count int64
		d.db.WithContext(ctx).
			Model(&models.CopilotMessage{}).
			Where("session_id = ?", session.ID).
			Count(&count)
		// The count will be reflected in ToResponse() through len(Messages)
		// We could optimize by adding a message_count column
	}

	return sessions, nil
}

// UpdateCopilotSession updates a Copilot session
func (d *Database) UpdateCopilotSession(ctx context.Context, session *models.CopilotSession) error {
	session.UpdatedAt = time.Now()
	result := d.db.WithContext(ctx).Save(session)
	if result.Error != nil {
		return fmt.Errorf("failed to update Copilot session: %w", result.Error)
	}
	return nil
}

// DeleteCopilotSession deletes a Copilot session and its messages
func (d *Database) DeleteCopilotSession(ctx context.Context, id string) error {
	// Messages will be deleted by cascade
	result := d.db.WithContext(ctx).Where("id = ?", id).Delete(&models.CopilotSession{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete Copilot session: %w", result.Error)
	}
	return nil
}

// CreateCopilotMessage creates a new message in a Copilot session
func (d *Database) CreateCopilotMessage(ctx context.Context, message *models.CopilotMessage) (int64, error) {
	if message.SessionID == "" {
		return 0, fmt.Errorf("session ID is required")
	}
	if message.Role == "" {
		return 0, fmt.Errorf("role is required")
	}

	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	result := d.db.WithContext(ctx).Create(message)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to create Copilot message: %w", result.Error)
	}

	// Update session's updated_at timestamp
	d.db.WithContext(ctx).
		Model(&models.CopilotSession{}).
		Where("id = ?", message.SessionID).
		Update("updated_at", time.Now())

	return message.ID, nil
}

// GetCopilotMessages retrieves all messages for a session
func (d *Database) GetCopilotMessages(ctx context.Context, sessionID string) ([]models.CopilotMessage, error) {
	var messages []models.CopilotMessage
	result := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get Copilot messages: %w", result.Error)
	}
	return messages, nil
}

// CleanupExpiredSessions removes expired Copilot sessions
func (d *Database) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	result := d.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&models.CopilotSession{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// isNotFoundError checks if an error is a "not found" error
func isNotFoundError(err error) bool {
	return err.Error() == "record not found"
}

// Compile-time interface check
var _ CopilotStore = (*Database)(nil)
