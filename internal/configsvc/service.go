// Package configsvc provides dynamic configuration access with caching.
// It reads settings from the database and caches them for performance.
package configsvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Service provides dynamic configuration access with caching.
// It reads settings from the database and caches them for performance.
// Call Reload() to refresh the cache after settings are updated.
type Service struct {
	db     *storage.Database
	logger *slog.Logger

	// Static config (from .env, requires restart)
	staticConfig *config.Config

	// Cached dynamic settings
	mu       sync.RWMutex
	settings *models.Settings
	lastLoad time.Time

	// Reload callbacks - components register to be notified of config changes
	reloadCallbacks []func()
}

// New creates a new ConfigService
func New(db *storage.Database, staticConfig *config.Config, logger *slog.Logger) (*Service, error) {
	cs := &Service{
		db:           db,
		logger:       logger,
		staticConfig: staticConfig,
	}

	// Load initial settings
	if err := cs.Reload(); err != nil {
		return nil, fmt.Errorf("failed to load initial settings: %w", err)
	}

	return cs, nil
}

// Reload refreshes settings from the database and notifies registered callbacks
func (cs *Service) Reload() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	settings, err := cs.db.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to reload settings: %w", err)
	}

	cs.mu.Lock()
	cs.settings = settings
	cs.lastLoad = time.Now()
	cs.mu.Unlock()

	cs.logger.Info("Configuration reloaded from database",
		"destination_configured", settings.HasDestination(),
		"auth_enabled", settings.AuthEnabled,
		"migration_workers", settings.MigrationWorkers)

	// Notify all registered callbacks
	for _, callback := range cs.reloadCallbacks {
		go callback()
	}

	return nil
}

// OnReload registers a callback to be called when configuration is reloaded
func (cs *Service) OnReload(callback func()) {
	cs.reloadCallbacks = append(cs.reloadCallbacks, callback)
}

// GetSettings returns the cached settings (read-only)
func (cs *Service) GetSettings() *models.Settings {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings
}

// GetStaticConfig returns the static configuration (from .env)
func (cs *Service) GetStaticConfig() *config.Config {
	return cs.staticConfig
}

// DestinationConfig contains destination GitHub configuration
type DestinationConfig struct {
	BaseURL           string
	Token             string
	AppID             int64
	AppPrivateKey     string
	AppInstallationID int64
	EnterpriseSlug    string
	Configured        bool
}

// GetDestinationConfig returns the destination GitHub configuration
func (cs *Service) GetDestinationConfig() DestinationConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	cfg := DestinationConfig{
		BaseURL:    cs.settings.DestinationBaseURL,
		Configured: cs.settings.HasDestination(),
	}

	if cs.settings.DestinationToken != nil {
		cfg.Token = *cs.settings.DestinationToken
	}
	if cs.settings.DestinationAppID != nil {
		cfg.AppID = *cs.settings.DestinationAppID
	}
	if cs.settings.DestinationAppPrivateKey != nil {
		cfg.AppPrivateKey = *cs.settings.DestinationAppPrivateKey
	}
	if cs.settings.DestinationAppInstallationID != nil {
		cfg.AppInstallationID = *cs.settings.DestinationAppInstallationID
	}
	if cs.settings.DestinationEnterpriseSlug != nil {
		cfg.EnterpriseSlug = *cs.settings.DestinationEnterpriseSlug
	}

	return cfg
}

// MigrationConfig contains migration worker configuration
type MigrationConfig struct {
	Workers              int
	PollIntervalSeconds  int
	DestRepoExistsAction string
	VisibilityPublic     string
	VisibilityInternal   string
}

// GetMigrationConfig returns the migration worker configuration
func (cs *Service) GetMigrationConfig() MigrationConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return MigrationConfig{
		Workers:              cs.settings.MigrationWorkers,
		PollIntervalSeconds:  cs.settings.MigrationPollIntervalSeconds,
		DestRepoExistsAction: cs.settings.MigrationDestRepoExistsAction,
		VisibilityPublic:     cs.settings.MigrationVisibilityPublic,
		VisibilityInternal:   cs.settings.MigrationVisibilityInternal,
	}
}

// GetDestRepoExistsAction returns the current action for existing destination repos.
// Implements migration.MigrationConfigProvider interface.
func (cs *Service) GetDestRepoExistsAction() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings.MigrationDestRepoExistsAction
}

// GetVisibilityPublic returns how to handle public repos.
// Implements migration.MigrationConfigProvider interface.
func (cs *Service) GetVisibilityPublic() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings.MigrationVisibilityPublic
}

// GetVisibilityInternal returns how to handle internal repos.
// Implements migration.MigrationConfigProvider interface.
func (cs *Service) GetVisibilityInternal() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings.MigrationVisibilityInternal
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled              bool
	SessionSecret        string
	SessionDurationHours int
	CallbackURL          string
	FrontendURL          string
}

// GetAuthConfig returns the authentication configuration
func (cs *Service) GetAuthConfig() AuthConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	cfg := AuthConfig{
		Enabled:              cs.settings.AuthEnabled,
		SessionDurationHours: cs.settings.AuthSessionDurationHours,
		FrontendURL:          cs.settings.AuthFrontendURL,
	}

	if cs.settings.AuthSessionSecret != nil {
		cfg.SessionSecret = *cs.settings.AuthSessionSecret
	}
	if cs.settings.AuthCallbackURL != nil {
		cfg.CallbackURL = *cs.settings.AuthCallbackURL
	}

	return cfg
}

// IsDestinationConfigured returns true if destination is properly configured
func (cs *Service) IsDestinationConfigured() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings.HasDestination()
}

// IsAuthEnabled returns true if authentication is enabled
func (cs *Service) IsAuthEnabled() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.settings.AuthEnabled
}

// GetEnterpriseSlug returns the destination enterprise slug from database settings.
// This is used for enterprise admin authorization checks.
// Returns empty string if not configured.
func (cs *Service) GetEnterpriseSlug() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if cs.settings.DestinationEnterpriseSlug != nil {
		return *cs.settings.DestinationEnterpriseSlug
	}
	return ""
}

// GetEffectiveAuthConfig returns an AuthConfig that merges static config with database settings.
// The enterprise slug is taken from database if set, otherwise from static config.
func (cs *Service) GetEffectiveAuthConfig() config.AuthConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Start with static config
	effectiveCfg := cs.staticConfig.Auth

	// Override enterprise slug from database if set
	if cs.settings.DestinationEnterpriseSlug != nil && *cs.settings.DestinationEnterpriseSlug != "" {
		effectiveCfg.AuthorizationRules.RequireEnterpriseSlug = *cs.settings.DestinationEnterpriseSlug
	}

	// Override authorization rules from database settings
	effectiveCfg.AuthorizationRules.AllowOrgAdminMigrations = cs.settings.AuthAllowOrgAdminMigrations
	effectiveCfg.AuthorizationRules.AllowEnterpriseAdminMigrations = cs.settings.AuthAllowEnterpriseAdminMigrations
	effectiveCfg.AuthorizationRules.EnableSelfService = cs.settings.AuthEnableSelfService

	// Parse migration admin teams from database
	if cs.settings.AuthMigrationAdminTeams != nil && *cs.settings.AuthMigrationAdminTeams != "" {
		teams := []string{}
		for _, team := range splitAndTrim(*cs.settings.AuthMigrationAdminTeams, ",") {
			if team != "" {
				teams = append(teams, team)
			}
		}
		effectiveCfg.AuthorizationRules.MigrationAdminTeams = teams
	}

	return effectiveCfg
}

// splitAndTrim splits a string by separator and trims whitespace from each part
func splitAndTrim(s string, sep string) []string {
	parts := []string{}
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString is a simple string split (avoiding strings import for minimal dependencies)
func splitString(s string, sep string) []string {
	if s == "" {
		return nil
	}
	result := []string{}
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace trims whitespace from a string
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// GetDatabaseConfig returns the static database configuration (requires restart to change)
func (cs *Service) GetDatabaseConfig() config.DatabaseConfig {
	return cs.staticConfig.Database
}

// GetServerPort returns the static server port (requires restart to change)
func (cs *Service) GetServerPort() int {
	return cs.staticConfig.Server.Port
}
