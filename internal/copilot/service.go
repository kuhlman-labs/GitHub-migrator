// Package copilot provides the Copilot chat service integration using the official SDK.
package copilot

import (
	"context"
	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Service manages Copilot interactions using the SDK client.
// This is a facade that delegates to the SDK Client.
type Service struct {
	client           *Client
	db               *storage.Database
	logger           *slog.Logger
	licenseValidator *LicenseValidator
}

// ServiceConfig configures the Copilot service using the SDK.
type ServiceConfig struct {
	CLIPath           string // Path to Copilot CLI executable
	CLIUrl            string // URL of existing CLI server (optional)
	Model             string // AI model to use (e.g., DefaultModel)
	SessionTimeoutMin int    // Session timeout in minutes
	RequireLicense    bool   // Require valid Copilot license
	GitHubBaseURL     string // GitHub base URL for license validation
	Streaming         bool   // Enable streaming responses
	LogLevel          string // SDK log level (debug, info, warn, error)
	GHToken           string // GitHub token for Copilot CLI authentication (optional)
}

// NewService creates a new Copilot service that uses the SDK.
func NewService(db *storage.Database, logger *slog.Logger, config ServiceConfig) *Service {
	licenseValidator := NewLicenseValidator(config.GitHubBaseURL, logger)

	// Create SDK client configuration
	clientConfig := ClientConfig{
		CLIPath:           config.CLIPath,
		CLIUrl:            config.CLIUrl,
		Model:             config.Model,
		LogLevel:          config.LogLevel,
		SessionTimeoutMin: config.SessionTimeoutMin,
		Streaming:         config.Streaming,
		GHToken:           config.GHToken,
	}

	// Set defaults
	if clientConfig.Model == "" {
		clientConfig.Model = DefaultModel
	}
	if clientConfig.SessionTimeoutMin == 0 {
		clientConfig.SessionTimeoutMin = 30
	}
	if clientConfig.LogLevel == "" {
		clientConfig.LogLevel = DefaultLogLevel
	}

	client := NewClient(db, logger, clientConfig)

	return &Service{
		client:           client,
		db:               db,
		logger:           logger,
		licenseValidator: licenseValidator,
	}
}

// Start initializes the SDK client.
func (s *Service) Start() error {
	return s.client.Start()
}

// Stop shuts down the SDK client.
func (s *Service) Stop() error {
	return s.client.Stop()
}

// GetClient returns the underlying SDK client.
func (s *Service) GetClient() *Client {
	return s.client
}

// GetStatus returns the current Copilot status for a user.
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

// CreateSession creates a new chat session with authorization context.
// If model is empty, the configured default model is used.
func (s *Service) CreateSession(ctx context.Context, userID, userLogin string, timeoutMin int, authCtx *AuthContext, model string) (*SDKSession, error) {
	return s.client.CreateSession(ctx, userID, userLogin, timeoutMin, authCtx, model)
}

// GetSession retrieves a session by ID.
func (s *Service) GetSession(ctx context.Context, sessionID string) (*SDKSession, error) {
	return s.client.GetSession(ctx, sessionID)
}

// UpdateSessionAuth updates the authorization context for an existing session.
func (s *Service) UpdateSessionAuth(sessionID string, authCtx *AuthContext) {
	s.client.UpdateSessionAuth(sessionID, authCtx)
}

// ListSessions returns all sessions for a user.
func (s *Service) ListSessions(ctx context.Context, userID string) ([]*models.CopilotSessionResponse, error) {
	return s.client.ListSessions(ctx, userID)
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, sessionID string) error {
	return s.client.DeleteSession(ctx, sessionID)
}

// SendMessage sends a message to Copilot and returns the response.
func (s *Service) SendMessage(ctx context.Context, sessionID, userMessage string, settings *models.Settings) (*models.ChatResponse, error) {
	return s.client.SendMessage(ctx, sessionID, userMessage)
}

// StreamMessage sends a message and streams the response via a callback.
func (s *Service) StreamMessage(ctx context.Context, sessionID, message string, onEvent func(event StreamEvent)) error {
	return s.client.StreamMessage(ctx, sessionID, message, onEvent)
}

// GetSessionHistory returns the message history for a session.
func (s *Service) GetSessionHistory(ctx context.Context, sessionID string) ([]models.CopilotMessage, error) {
	return s.client.GetSessionHistory(ctx, sessionID)
}

// ListModels returns the available AI models.
func (s *Service) ListModels(ctx context.Context) ([]models.ModelInfo, error) {
	return s.client.ListModels(ctx)
}

// GetDefaultModel returns the configured default model.
func (s *Service) GetDefaultModel() string {
	return s.client.GetDefaultModel()
}

// Note: CheckCLIAvailable is defined in license.go

// Status constants for tool implementations.
const (
	StatusPending           = "pending"
	StatusScheduled         = "scheduled"
	StatusCompleted         = "completed"
	StatusMigrationComplete = "migration_complete"
	RatingUnknown           = "unknown"
)

// canQueueForMigration checks if a repository can be queued for migration.
func canQueueForMigration(status string, dryRun bool) bool {
	switch models.MigrationStatus(status) {
	case models.StatusPending,
		models.StatusDryRunFailed,
		models.StatusMigrationFailed,
		models.StatusRolledBack:
		return true
	case models.StatusDryRunComplete:
		return !dryRun
	default:
		return false
	}
}

// isInQueuedOrInProgressState checks if a repository is in a cancellable state.
func isInQueuedOrInProgressState(status string) bool {
	switch models.MigrationStatus(status) {
	case models.StatusDryRunQueued,
		models.StatusDryRunInProgress,
		models.StatusQueuedForMigration,
		models.StatusMigratingContent,
		models.StatusArchiveGenerating,
		models.StatusPreMigration:
		return true
	default:
		return false
	}
}

// calculateProgress calculates progress metrics from a list of statuses.
func calculateProgress(statuses []string) map[string]any {
	var pendingCount, queuedCount, inProgressCount, completedCount, failedCount, skippedCount int
	totalCount := len(statuses)

	for _, status := range statuses {
		switch models.MigrationStatus(status) {
		case models.StatusPending:
			pendingCount++
		case models.StatusDryRunQueued, models.StatusQueuedForMigration:
			queuedCount++
		case models.StatusDryRunInProgress, models.StatusMigratingContent,
			models.StatusArchiveGenerating, models.StatusPreMigration, models.StatusPostMigration:
			inProgressCount++
		case models.StatusDryRunComplete, models.StatusMigrationComplete, models.StatusComplete:
			completedCount++
		case models.StatusDryRunFailed, models.StatusMigrationFailed:
			failedCount++
		case models.StatusWontMigrate:
			skippedCount++
		}
	}

	var percentComplete float64
	if totalCount > 0 {
		percentComplete = float64(completedCount) / float64(totalCount) * 100
	}

	return map[string]any{
		"total_count":       totalCount,
		"pending_count":     pendingCount,
		"queued_count":      queuedCount,
		"in_progress_count": inProgressCount,
		"completed_count":   completedCount,
		"failed_count":      failedCount,
		"skipped_count":     skippedCount,
		"percent_complete":  percentComplete,
	}
}
