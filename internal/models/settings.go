package models

import "time"

// Settings represents the application settings stored in the database.
// These settings can be updated at runtime without requiring a server restart.
// Only database DSN and server port are stored in .env and require restart.
type Settings struct {
	ID int64 `json:"id" db:"id" gorm:"primaryKey"`

	// Destination GitHub configuration
	DestinationBaseURL          string  `json:"destination_base_url" db:"destination_base_url" gorm:"column:destination_base_url;not null;default:'https://api.github.com'"`
	DestinationToken            *string `json:"-" db:"destination_token" gorm:"column:destination_token"`
	DestinationAppID            *int64  `json:"destination_app_id,omitempty" db:"destination_app_id" gorm:"column:destination_app_id"`
	DestinationAppPrivateKey    *string `json:"-" db:"destination_app_private_key" gorm:"column:destination_app_private_key"`
	DestinationAppInstallationID *int64  `json:"destination_app_installation_id,omitempty" db:"destination_app_installation_id" gorm:"column:destination_app_installation_id"`

	// Migration settings
	MigrationWorkers             int    `json:"migration_workers" db:"migration_workers" gorm:"column:migration_workers;not null;default:5"`
	MigrationPollIntervalSeconds int    `json:"migration_poll_interval_seconds" db:"migration_poll_interval_seconds" gorm:"column:migration_poll_interval_seconds;not null;default:30"`
	MigrationDestRepoExistsAction string `json:"migration_dest_repo_exists_action" db:"migration_dest_repo_exists_action" gorm:"column:migration_dest_repo_exists_action;not null;default:'fail'"`
	MigrationVisibilityPublic    string `json:"migration_visibility_public" db:"migration_visibility_public" gorm:"column:migration_visibility_public;not null;default:'private'"`
	MigrationVisibilityInternal  string `json:"migration_visibility_internal" db:"migration_visibility_internal" gorm:"column:migration_visibility_internal;not null;default:'private'"`

	// Auth settings
	AuthEnabled              bool    `json:"auth_enabled" db:"auth_enabled" gorm:"column:auth_enabled;not null;default:false"`
	AuthSessionSecret        *string `json:"-" db:"auth_session_secret" gorm:"column:auth_session_secret"`
	AuthSessionDurationHours int     `json:"auth_session_duration_hours" db:"auth_session_duration_hours" gorm:"column:auth_session_duration_hours;not null;default:24"`
	AuthCallbackURL          *string `json:"auth_callback_url,omitempty" db:"auth_callback_url" gorm:"column:auth_callback_url"`
	AuthFrontendURL          string  `json:"auth_frontend_url" db:"auth_frontend_url" gorm:"column:auth_frontend_url;not null;default:'http://localhost:3000'"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" gorm:"column:updated_at"`
}

// TableName returns the table name for GORM
func (Settings) TableName() string {
	return "settings"
}

// HasDestination returns true if a destination token or app credentials are configured
func (s *Settings) HasDestination() bool {
	hasToken := s.DestinationToken != nil && *s.DestinationToken != ""
	hasApp := s.DestinationAppID != nil && *s.DestinationAppID > 0 &&
		s.DestinationAppPrivateKey != nil && *s.DestinationAppPrivateKey != ""
	return hasToken || hasApp
}

// SettingsResponse is the API response for settings (with sensitive data masked)
type SettingsResponse struct {
	ID int64 `json:"id"`

	// Destination GitHub configuration (token masked)
	DestinationBaseURL           string `json:"destination_base_url"`
	DestinationTokenConfigured   bool   `json:"destination_token_configured"`
	DestinationAppID             *int64 `json:"destination_app_id,omitempty"`
	DestinationAppKeyConfigured  bool   `json:"destination_app_key_configured"`
	DestinationAppInstallationID *int64 `json:"destination_app_installation_id,omitempty"`

	// Migration settings
	MigrationWorkers             int    `json:"migration_workers"`
	MigrationPollIntervalSeconds int    `json:"migration_poll_interval_seconds"`
	MigrationDestRepoExistsAction string `json:"migration_dest_repo_exists_action"`
	MigrationVisibilityPublic    string `json:"migration_visibility_public"`
	MigrationVisibilityInternal  string `json:"migration_visibility_internal"`

	// Auth settings (secrets masked)
	AuthEnabled              bool   `json:"auth_enabled"`
	AuthSessionSecretSet     bool   `json:"auth_session_secret_set"`
	AuthSessionDurationHours int    `json:"auth_session_duration_hours"`
	AuthCallbackURL          string `json:"auth_callback_url,omitempty"`
	AuthFrontendURL          string `json:"auth_frontend_url"`

	// Status
	DestinationConfigured bool      `json:"destination_configured"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// ToResponse converts Settings to a safe API response with masked secrets
func (s *Settings) ToResponse() *SettingsResponse {
	callbackURL := ""
	if s.AuthCallbackURL != nil {
		callbackURL = *s.AuthCallbackURL
	}

	return &SettingsResponse{
		ID: s.ID,

		// Destination
		DestinationBaseURL:           s.DestinationBaseURL,
		DestinationTokenConfigured:   s.DestinationToken != nil && *s.DestinationToken != "",
		DestinationAppID:             s.DestinationAppID,
		DestinationAppKeyConfigured:  s.DestinationAppPrivateKey != nil && *s.DestinationAppPrivateKey != "",
		DestinationAppInstallationID: s.DestinationAppInstallationID,

		// Migration
		MigrationWorkers:             s.MigrationWorkers,
		MigrationPollIntervalSeconds: s.MigrationPollIntervalSeconds,
		MigrationDestRepoExistsAction: s.MigrationDestRepoExistsAction,
		MigrationVisibilityPublic:    s.MigrationVisibilityPublic,
		MigrationVisibilityInternal:  s.MigrationVisibilityInternal,

		// Auth
		AuthEnabled:              s.AuthEnabled,
		AuthSessionSecretSet:     s.AuthSessionSecret != nil && *s.AuthSessionSecret != "",
		AuthSessionDurationHours: s.AuthSessionDurationHours,
		AuthCallbackURL:          callbackURL,
		AuthFrontendURL:          s.AuthFrontendURL,

		// Status
		DestinationConfigured: s.HasDestination(),
		UpdatedAt:             s.UpdatedAt,
	}
}

// UpdateSettingsRequest is the request to update settings
type UpdateSettingsRequest struct {
	// Destination GitHub configuration
	DestinationBaseURL           *string `json:"destination_base_url,omitempty"`
	DestinationToken             *string `json:"destination_token,omitempty"`
	DestinationAppID             *int64  `json:"destination_app_id,omitempty"`
	DestinationAppPrivateKey     *string `json:"destination_app_private_key,omitempty"`
	DestinationAppInstallationID *int64  `json:"destination_app_installation_id,omitempty"`

	// Migration settings
	MigrationWorkers             *int    `json:"migration_workers,omitempty"`
	MigrationPollIntervalSeconds *int    `json:"migration_poll_interval_seconds,omitempty"`
	MigrationDestRepoExistsAction *string `json:"migration_dest_repo_exists_action,omitempty"`
	MigrationVisibilityPublic    *string `json:"migration_visibility_public,omitempty"`
	MigrationVisibilityInternal  *string `json:"migration_visibility_internal,omitempty"`

	// Auth settings
	AuthEnabled              *bool   `json:"auth_enabled,omitempty"`
	AuthSessionSecret        *string `json:"auth_session_secret,omitempty"`
	AuthSessionDurationHours *int    `json:"auth_session_duration_hours,omitempty"`
	AuthCallbackURL          *string `json:"auth_callback_url,omitempty"`
	AuthFrontendURL          *string `json:"auth_frontend_url,omitempty"`
}

