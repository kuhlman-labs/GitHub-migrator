package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
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
	Type string `mapstructure:"type"` // "sqlite" or "postgres"
	DSN  string `mapstructure:"dsn"`
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
	CallbackURL             string             `mapstructure:"callback_url"`
	FrontendURL             string             `mapstructure:"frontend_url"`
	SessionSecret           string             `mapstructure:"session_secret"`
	SessionDurationHours    int                `mapstructure:"session_duration_hours"`
	AuthorizationRules      AuthorizationRules `mapstructure:"authorization_rules"`
}

// AuthorizationRules defines rules for authorizing users
type AuthorizationRules struct {
	RequireOrgMembership   []string `mapstructure:"require_org_membership"`
	RequireTeamMembership  []string `mapstructure:"require_team_membership"`
	RequireEnterpriseAdmin bool     `mapstructure:"require_enterprise_admin"`
	RequireEnterpriseSlug  string   `mapstructure:"require_enterprise_slug"`
}

func Load() (*Config, error) {
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

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

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

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dsn", "./data/migrator.db")
	viper.SetDefault("source.type", "github")
	viper.SetDefault("source.base_url", "https://api.github.com")
	viper.SetDefault("destination.type", "github")
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
}

// MigrateDeprecatedConfig migrates old GitHub config format to new Source/Destination format
func (c *Config) MigrateDeprecatedConfig() {
	// If new config is not set but old GitHub config exists, migrate it
	if c.Source.Type == "" && c.GitHub.Source.Token != "" {
		c.Source.Type = "github"
		c.Source.BaseURL = c.GitHub.Source.BaseURL
		c.Source.Token = c.GitHub.Source.Token
	}
	if c.Destination.Type == "" && c.GitHub.Destination.Token != "" {
		c.Destination.Type = "github"
		c.Destination.BaseURL = c.GitHub.Destination.BaseURL
		c.Destination.Token = c.GitHub.Destination.Token
	}
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
