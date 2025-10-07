package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Source      SourceConfig      `mapstructure:"source"`
	Destination DestinationConfig `mapstructure:"destination"`
	Logging     LoggingConfig     `mapstructure:"logging"`
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
	Token        string `mapstructure:"token"`        // Authentication token
	Organization string `mapstructure:"organization"` // Organization name (required for Azure DevOps)
	Username     string `mapstructure:"username"`     // Username (optional, for Azure DevOps)
}

// DestinationConfig defines the destination repository system configuration
type DestinationConfig struct {
	Type    string `mapstructure:"type"`     // "github", "gitlab", or "azuredevops"
	BaseURL string `mapstructure:"base_url"` // API base URL
	Token   string `mapstructure:"token"`    // Authentication token
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

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Environment variable support
	viper.SetEnvPrefix("GHMIG")
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
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output_file", "./logs/migrator.log")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
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
