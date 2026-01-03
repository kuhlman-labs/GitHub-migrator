package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

const (
	// ProviderTypeGitHub is the GitHub provider type
	ProviderTypeGitHub = "github"
)

type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Source      SourceConfig      `mapstructure:"source"`
	Destination DestinationConfig `mapstructure:"destination"`
	Migration   MigrationConfig   `mapstructure:"migration"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	Auth        AuthConfig        `mapstructure:"auth"`
	// Deprecated: Use Source and Destination instead
	GitHub GitHubConfig `mapstructure:"github"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Type                   string `mapstructure:"type"` // "sqlite", "postgres", or "sqlserver"
	DSN                    string `mapstructure:"dsn"`
	MaxOpenConns           int    `mapstructure:"max_open_conns"`            // Maximum number of open connections
	MaxIdleConns           int    `mapstructure:"max_idle_conns"`            // Maximum number of idle connections
	ConnMaxLifetimeSeconds int    `mapstructure:"conn_max_lifetime_seconds"` // Maximum lifetime of a connection in seconds
	AutoMigrate            bool   `mapstructure:"auto_migrate"`              // Use GORM AutoMigrate vs SQL migrations
	// SQL Server specific
	TrustServerCertificate bool `mapstructure:"trust_server_certificate"` // For SQL Server TLS
	// PostgreSQL specific
	SSLMode string `mapstructure:"ssl_mode"` // PostgreSQL SSL mode: disable, require, verify-ca, verify-full
}

// SourceConfig defines the source repository system configuration
type SourceConfig struct {
	Type         string `mapstructure:"type"`         // "github", "gitlab", or "azuredevops"
	BaseURL      string `mapstructure:"base_url"`     // API base URL
	Token        string `mapstructure:"token"`        // Authentication token (PAT)
	Organization string `mapstructure:"organization"` // Organization name (required for Azure DevOps)
	Username     string `mapstructure:"username"`     // Username (optional, for Azure DevOps)

	// GitHub App authentication (optional, for non-migration operations)
	AppID             int64  `mapstructure:"app_id"`              // GitHub App ID
	AppPrivateKey     string `mapstructure:"app_private_key"`     // Private key (file path or inline PEM)
	AppInstallationID int64  `mapstructure:"app_installation_id"` // Installation ID
}

// DestinationConfig defines the destination repository system configuration
type DestinationConfig struct {
	Type    string `mapstructure:"type"`     // "github", "gitlab", or "azuredevops"
	BaseURL string `mapstructure:"base_url"` // API base URL
	Token   string `mapstructure:"token"`    // Authentication token (PAT)

	// GitHub App authentication (optional, for non-migration operations)
	AppID             int64  `mapstructure:"app_id"`              // GitHub App ID
	AppPrivateKey     string `mapstructure:"app_private_key"`     // Private key (file path or inline PEM)
	AppInstallationID int64  `mapstructure:"app_installation_id"` // Installation ID
}

// MigrationConfig defines migration worker configuration
type MigrationConfig struct {
	Workers              int                      `mapstructure:"workers"`                 // Number of parallel workers
	PollIntervalSeconds  int                      `mapstructure:"poll_interval_seconds"`   // Polling interval in seconds
	PostMigrationMode    string                   `mapstructure:"post_migration_mode"`     // never, production_only, dry_run_only, always
	DestRepoExistsAction string                   `mapstructure:"dest_repo_exists_action"` // fail, skip, delete
	VisibilityHandling   VisibilityHandlingConfig `mapstructure:"visibility_handling"`     // Visibility transformation rules
}

// VisibilityHandlingConfig defines how to handle repository visibility during migration
type VisibilityHandlingConfig struct {
	PublicRepos   string `mapstructure:"public_repos"`   // public, internal, or private (default: private)
	InternalRepos string `mapstructure:"internal_repos"` // internal or private (default: private)
}

// GitHubConfig is deprecated but kept for backward compatibility
type GitHubConfig struct {
	Source      GitHubInstanceConfig `mapstructure:"source"`
	Destination GitHubInstanceConfig `mapstructure:"destination"`
}

// GitHubInstanceConfig is deprecated but kept for backward compatibility
type GitHubInstanceConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Token   string `mapstructure:"token"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`  // "debug", "info", "warn", "error"
	Format     string `mapstructure:"format"` // "json" or "text"
	OutputFile string `mapstructure:"output_file"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
}

// AuthConfig defines authentication and authorization settings
type AuthConfig struct {
	Enabled                 bool               `mapstructure:"enabled"`
	GitHubOAuthClientID     string             `mapstructure:"github_oauth_client_id"`
	GitHubOAuthClientSecret string             `mapstructure:"github_oauth_client_secret"`
	GitHubOAuthBaseURL      string             `mapstructure:"github_oauth_base_url"` // Optional: defaults to source URL if source is GitHub, otherwise destination
	CallbackURL             string             `mapstructure:"callback_url"`
	FrontendURL             string             `mapstructure:"frontend_url"`
	SessionSecret           string             `mapstructure:"session_secret"`
	SessionDurationHours    int                `mapstructure:"session_duration_hours"`
	AuthorizationRules      AuthorizationRules `mapstructure:"authorization_rules"`

	// Entra ID OAuth for Azure DevOps
	EntraIDEnabled      bool   `mapstructure:"entraid_enabled"`
	EntraIDTenantID     string `mapstructure:"entraid_tenant_id"`
	EntraIDClientID     string `mapstructure:"entraid_client_id"`
	EntraIDClientSecret string `mapstructure:"entraid_client_secret"`
	EntraIDCallbackURL  string `mapstructure:"entraid_callback_url"`
	ADOOrganizationURL  string `mapstructure:"ado_organization_url"` // e.g., https://dev.azure.com/your-org
}

// AuthorizationRules defines rules for authorizing users
type AuthorizationRules struct {
	// Application access requirements (who can access the app at all)
	RequireOrgMembership        []string `mapstructure:"require_org_membership"`
	RequireTeamMembership       []string `mapstructure:"require_team_membership"`
	RequireEnterpriseAdmin      bool     `mapstructure:"require_enterprise_admin"`
	RequireEnterpriseMembership bool     `mapstructure:"require_enterprise_membership"` // Require user to be member of enterprise (any role)
	RequireEnterpriseSlug       string   `mapstructure:"require_enterprise_slug"`

	// Destination-centric authorization (who can migrate repositories)
	MigrationAdminTeams                  []string `mapstructure:"migration_admin_teams"`                     // Teams with full migration rights (format: "org/team-slug")
	AllowOrgAdminMigrations              bool     `mapstructure:"allow_org_admin_migrations"`                // Allow destination org admins to migrate any repo
	AllowEnterpriseAdminMigrations       bool     `mapstructure:"allow_enterprise_admin_migrations"`         // Allow destination enterprise admins to migrate any repo
	RequireIdentityMappingForSelfService bool     `mapstructure:"require_identity_mapping_for_self_service"` // Require identity mapping for self-service (default: true)

	// Deprecated: Use MigrationAdminTeams instead
	PrivilegedTeams []string `mapstructure:"privileged_teams"`
}

func Load() (*Config, error) {
	// Load .env file if it exists (for local development)
	// This loads environment variables from .env file into the environment
	// Silently ignore if .env doesn't exist - not an error
	if _, err := os.Stat(".env"); err == nil {
		if err := gotenv.Load(".env"); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Environment variable support
	viper.SetEnvPrefix("GHMIG")
	// Replace dots with underscores in config keys when looking for env vars
	// This allows migration.visibility_handling.public_repos -> GHMIG_MIGRATION_VISIBILITY_HANDLING_PUBLIC_REPOS
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Try to read config file, but don't fail if it doesn't exist
	// This allows pure environment variable configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred (e.g., parse error)
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found - this is OK, we'll use env vars and defaults
	}

	// WORKAROUND: Viper's Unmarshal doesn't pick up environment variables properly
	// We need to explicitly bind env vars to ensure they override config file values
	// See: https://github.com/spf13/viper/issues/188
	// This ensures proper precedence: env vars > config file > defaults
	bindEnvVars()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Migrate deprecated GitHub config if needed
	cfg.MigrateDeprecatedConfig()

	// Parse array environment variables (Viper doesn't automatically handle comma-separated values)
	cfg.ParseArrayEnvVars()

	return &cfg, nil
}

// bindEnvVars explicitly binds environment variables for Viper Unmarshal
// This is required when no config file exists, as Viper's AutomaticEnv() only works with Get()
func bindEnvVars() {
	// Bind all configuration keys to environment variables
	// Viper automatically converts dots to underscores and adds the GHMIG prefix
	envKeys := []string{
		"server.port",
		"database.type",
		"database.dsn",
		"database.max_open_conns",
		"database.max_idle_conns",
		"database.conn_max_lifetime_seconds",
		"database.auto_migrate",
		"database.trust_server_certificate",
		"database.ssl_mode",
		"source.type",
		"source.base_url",
		"source.token",
		"source.organization",
		"source.username",
		"source.app_id",
		"source.app_private_key",
		"source.app_installation_id",
		"destination.type",
		"destination.base_url",
		"destination.token",
		"destination.app_id",
		"destination.app_private_key",
		"destination.app_installation_id",
		"migration.workers",
		"migration.poll_interval_seconds",
		"migration.post_migration_mode",
		"migration.dest_repo_exists_action",
		"migration.visibility_handling.public_repos",
		"migration.visibility_handling.internal_repos",
		"logging.level",
		"logging.format",
		"logging.output_file",
		"logging.max_size",
		"logging.max_backups",
		"logging.max_age",
		"auth.enabled",
		"auth.github_oauth_client_id",
		"auth.github_oauth_client_secret",
		"auth.github_oauth_base_url",
		"auth.callback_url",
		"auth.frontend_url",
		"auth.session_secret",
		"auth.session_duration_hours",
		"auth.authorization_rules.require_org_membership",
		"auth.authorization_rules.require_team_membership",
		"auth.authorization_rules.require_enterprise_admin",
		"auth.authorization_rules.require_enterprise_membership",
		"auth.authorization_rules.require_enterprise_slug",
		"auth.authorization_rules.privileged_teams",
		"auth.authorization_rules.migration_admin_teams",
		"auth.authorization_rules.allow_org_admin_migrations",
		"auth.authorization_rules.allow_enterprise_admin_migrations",
		"auth.authorization_rules.require_identity_mapping_for_self_service",
		"auth.entraid_enabled",
		"auth.entraid_tenant_id",
		"auth.entraid_client_id",
		"auth.entraid_client_secret",
		"auth.entraid_callback_url",
		"auth.ado_organization_url",
	}

	for _, key := range envKeys {
		_ = viper.BindEnv(key) // Explicitly ignore error - BindEnv only fails if key is empty
	}
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dsn", "./data/migrator.db")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime_seconds", 300)
	viper.SetDefault("database.auto_migrate", false)
	viper.SetDefault("database.trust_server_certificate", false)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("source.type", ProviderTypeGitHub)
	viper.SetDefault("source.base_url", "https://api.github.com")
	viper.SetDefault("destination.type", ProviderTypeGitHub)
	viper.SetDefault("destination.base_url", "https://api.github.com")
	viper.SetDefault("migration.workers", 5)
	viper.SetDefault("migration.poll_interval_seconds", 30)
	viper.SetDefault("migration.post_migration_mode", "production_only")
	viper.SetDefault("migration.dest_repo_exists_action", "fail")
	viper.SetDefault("migration.visibility_handling.public_repos", "private")
	viper.SetDefault("migration.visibility_handling.internal_repos", "private")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output_file", "./logs/migrator.log")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
	viper.SetDefault("auth.enabled", false)
	viper.SetDefault("auth.frontend_url", "http://localhost:3000")
	viper.SetDefault("auth.session_duration_hours", 24)
	viper.SetDefault("auth.authorization_rules.require_enterprise_admin", false)
	viper.SetDefault("auth.authorization_rules.allow_org_admin_migrations", true)
	viper.SetDefault("auth.authorization_rules.allow_enterprise_admin_migrations", true)
	viper.SetDefault("auth.authorization_rules.require_identity_mapping_for_self_service", true)
	viper.SetDefault("auth.entraid_enabled", false)
}

// MigrateDeprecatedConfig migrates old GitHub config format to new Source/Destination format
func (c *Config) MigrateDeprecatedConfig() {
	// If new config is not set but old GitHub config exists, migrate it
	if c.Source.Type == "" && c.GitHub.Source.Token != "" {
		c.Source.Type = ProviderTypeGitHub
		c.Source.BaseURL = c.GitHub.Source.BaseURL
		c.Source.Token = c.GitHub.Source.Token
	}
	if c.Destination.Type == "" && c.GitHub.Destination.Token != "" {
		c.Destination.Type = ProviderTypeGitHub
		c.Destination.BaseURL = c.GitHub.Destination.BaseURL
		c.Destination.Token = c.GitHub.Destination.Token
	}
}

// GetOAuthBaseURL returns the GitHub OAuth base URL, with intelligent defaulting
// Priority: explicit GitHubOAuthBaseURL > source URL (if GitHub) > destination URL
func (c *AuthConfig) GetOAuthBaseURL(cfg *Config) string {
	// If explicitly configured, use that
	if c.GitHubOAuthBaseURL != "" {
		return c.GitHubOAuthBaseURL
	}

	// Default to source URL if source is GitHub
	if cfg.Source.Type == ProviderTypeGitHub && cfg.Source.BaseURL != "" {
		return cfg.Source.BaseURL
	}

	// Fall back to destination URL if destination is GitHub
	if cfg.Destination.Type == ProviderTypeGitHub && cfg.Destination.BaseURL != "" {
		return cfg.Destination.BaseURL
	}

	// Final fallback to github.com
	return "https://api.github.com"
}

// ParseArrayEnvVars handles parsing of array fields from environment variables
// Viper doesn't automatically parse comma-separated values or handle array syntax
func (c *Config) ParseArrayEnvVars() {
	// Parse require_org_membership
	c.Auth.AuthorizationRules.RequireOrgMembership = parseStringSlice(
		c.Auth.AuthorizationRules.RequireOrgMembership,
	)

	// Parse require_team_membership
	c.Auth.AuthorizationRules.RequireTeamMembership = parseStringSlice(
		c.Auth.AuthorizationRules.RequireTeamMembership,
	)

	// Parse privileged_teams (deprecated, but still supported)
	c.Auth.AuthorizationRules.PrivilegedTeams = parseStringSlice(
		c.Auth.AuthorizationRules.PrivilegedTeams,
	)

	// Parse migration_admin_teams
	c.Auth.AuthorizationRules.MigrationAdminTeams = parseStringSlice(
		c.Auth.AuthorizationRules.MigrationAdminTeams,
	)

	// Merge privileged_teams into migration_admin_teams for backward compatibility
	if len(c.Auth.AuthorizationRules.PrivilegedTeams) > 0 && len(c.Auth.AuthorizationRules.MigrationAdminTeams) == 0 {
		c.Auth.AuthorizationRules.MigrationAdminTeams = c.Auth.AuthorizationRules.PrivilegedTeams
	}
}

// parseStringSlice handles parsing of string slice from various formats:
// - Comma-separated: "org1,org2,org3"
// - Single value: "org1"
// - Already parsed array: ["org1", "org2"]
// - JSON array string: '["org1","org2"]' (incorrectly set)
func parseStringSlice(input []string) []string {
	if len(input) == 0 {
		return input
	}

	// If we have a single element, check if it needs parsing
	if len(input) == 1 {
		value := strings.TrimSpace(input[0])

		// Empty value
		if value == "" {
			return []string{}
		}

		// Check if it's a JSON array string (common mistake)
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			// Strip outer brackets first
			value = value[1 : len(value)-1]
			value = strings.TrimSpace(value)

			// Remove all quotes
			value = strings.ReplaceAll(value, "\"", "")
			value = strings.ReplaceAll(value, "'", "")

			// If now empty, return empty array
			if value == "" {
				return []string{}
			}
		}

		// Split by comma and trim spaces
		if strings.Contains(value, ",") {
			parts := strings.Split(value, ",")
			result := make([]string, 0, len(parts))
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					result = append(result, trimmed)
				}
			}
			return result
		}

		// Single value (not comma-separated)
		return []string{value}
	}

	// Multiple elements - could be properly parsed or incorrectly split
	// Check if first element looks like it starts with JSON bracket
	if strings.HasPrefix(strings.TrimSpace(input[0]), "[") {
		// This might be a JSON array that got split by comma
		// Reconstruct and reparse
		reconstructed := strings.Join(input, ",")
		return parseStringSlice([]string{reconstructed})
	}

	// Already a properly parsed array, just trim spaces
	result := make([]string, 0, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		// Clean up any remaining quotes or brackets
		trimmed = strings.Trim(trimmed, "[]\"'")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
