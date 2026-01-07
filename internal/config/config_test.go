package config

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestSetDefaults(t *testing.T) {
	viper.Reset()
	setDefaults()

	tests := []struct {
		key      string
		expected any
	}{
		{"server.port", 8080},
		{"database.type", "sqlite"},
		{"database.dsn", "./data/migrator.db"},
		{"logging.level", "info"},
		{"logging.format", "json"},
		{"logging.output_file", "./logs/migrator.log"},
		{"logging.max_size", 100},
		{"logging.max_backups", 3},
		{"logging.max_age", 28},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := viper.Get(tt.key)
			if got != tt.expected {
				t.Errorf("setDefaults() for %s = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestLoadConfig_WithDefaults(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	configContent := `
server:
  port: 9090

database:
  type: sqlite
  dsn: ./test.db

github:
  source:
    base_url: "https://source.example.com"
    token: "source-token"
  destination:
    base_url: "https://dest.example.com"
    token: "dest-token"

logging:
  level: debug
  format: text
  output_file: ./test.log
  max_size: 50
  max_backups: 5
  max_age: 14
`

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Save current directory and change to temp file location
	viper.Reset()
	viper.SetConfigFile(tmpfile.Name())

	var cfg Config
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify values
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}

	if cfg.Database.Type != "sqlite" {
		t.Errorf("Database.Type = %s, want sqlite", cfg.Database.Type)
	}

	if cfg.Database.DSN != "./test.db" {
		t.Errorf("Database.DSN = %s, want ./test.db", cfg.Database.DSN)
	}

	if cfg.GitHub.Source.BaseURL != "https://source.example.com" {
		t.Errorf("GitHub.Source.BaseURL = %s, want https://source.example.com", cfg.GitHub.Source.BaseURL)
	}

	if cfg.GitHub.Source.Token != "source-token" {
		t.Errorf("GitHub.Source.Token = %s, want source-token", cfg.GitHub.Source.Token)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %s, want debug", cfg.Logging.Level)
	}

	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %s, want text", cfg.Logging.Format)
	}

	if cfg.Logging.MaxSize != 50 {
		t.Errorf("Logging.MaxSize = %d, want 50", cfg.Logging.MaxSize)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	// Test that missing config file is OK - we use defaults and env vars
	// Save current dir and change to a temp dir with no config file
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(currentDir)

	tmpDir, err := os.MkdirTemp("", "config-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	viper.Reset()

	// Should succeed with defaults even without config file
	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() should succeed without config file, got error: %v", err)
	}

	// Verify defaults are loaded
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, expected 8080 (default)", cfg.Server.Port)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Test that invalid YAML in an existing config file returns an error
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(currentDir)

	tmpDir, err := os.MkdirTemp("", "config-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create configs directory in temp dir
	configsDir := tmpDir + "/configs"
	if err := os.Mkdir(configsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write invalid YAML to config.yaml
	invalidYAML := `
server:
  port: not-a-number
  invalid yaml content [[[
`
	configFile := configsDir + "/config.yaml"
	if err := os.WriteFile(configFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	viper.Reset()

	_, err = Load()
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}

// TestLoadConfig_EnvironmentVariables tests that environment variables with GHMIG_ prefix override config file values
func TestParseStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single value",
			input:    []string{"org1"},
			expected: []string{"org1"},
		},
		{
			name:     "comma-separated values",
			input:    []string{"org1,org2,org3"},
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "JSON array string",
			input:    []string{`["org1","org2","org3"]`},
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "JSON array with single quotes",
			input:    []string{`['org1','org2']`},
			expected: []string{"org1", "org2"},
		},
		{
			name:     "comma-separated with spaces",
			input:    []string{"org1 , org2 , org3"},
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "already parsed array",
			input:    []string{"org1", "org2", "org3"},
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "empty string",
			input:    []string{""},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStringSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i, val := range result {
				if val != tt.expected[i] {
					t.Errorf("Index %d: expected %s, got %s", i, tt.expected[i], val)
				}
			}
		})
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Create a temporary config file with base values
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	configContent := `
server:
  port: 8080

database:
  type: sqlite
  dsn: ./data/migrator.db

source:
  type: github
  base_url: https://api.github.com
  token: file-token

destination:
  type: github
  base_url: https://api.github.com
  token: file-token

migration:
  workers: 5
  poll_interval_seconds: 30
  post_migration_mode: production_only
  dest_repo_exists_action: fail
  visibility_handling:
    public_repos: private
    internal_repos: private

logging:
  level: info
  format: json

auth:
  enabled: false
  session_duration_hours: 24
  authorization_rules:
    require_enterprise_admin: false
`

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Set environment variables
	envVars := map[string]string{
		"GHMIG_SERVER_PORT":                                       "9090",
		"GHMIG_DATABASE_TYPE":                                     "postgres",
		"GHMIG_DATABASE_DSN":                                      "postgres://user:pass@host:5432/db",
		"GHMIG_SOURCE_TYPE":                                       "github",
		"GHMIG_SOURCE_BASE_URL":                                   "https://source.example.com",
		"GHMIG_SOURCE_TOKEN":                                      "env-source-token",
		"GHMIG_DESTINATION_TYPE":                                  "github",
		"GHMIG_DESTINATION_BASE_URL":                              "https://dest.example.com",
		"GHMIG_DESTINATION_TOKEN":                                 "env-dest-token",
		"GHMIG_MIGRATION_WORKERS":                                 "10",
		"GHMIG_MIGRATION_POLL_INTERVAL_SECONDS":                   "60",
		"GHMIG_MIGRATION_POST_MIGRATION_MODE":                     "always",
		"GHMIG_MIGRATION_DEST_REPO_EXISTS_ACTION":                 "skip",
		"GHMIG_MIGRATION_VISIBILITY_HANDLING_PUBLIC_REPOS":        "public",
		"GHMIG_MIGRATION_VISIBILITY_HANDLING_INTERNAL_REPOS":      "internal",
		"GHMIG_LOGGING_LEVEL":                                     "debug",
		"GHMIG_LOGGING_FORMAT":                                    "text",
		"GHMIG_AUTH_ENABLED":                                      "true",
		"GHMIG_AUTH_SESSION_DURATION_HOURS":                       "48",
		"GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_ADMIN": "true",
	}

	// Set environment variables
	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("Failed to set env var %s: %v", key, err)
		}
		defer os.Unsetenv(key)
	}

	// Reset viper and configure it to use the temp file
	viper.Reset()
	viper.SetConfigFile(tmpfile.Name())
	viper.SetConfigType("yaml")

	// Setup environment variable handling
	viper.SetEnvPrefix("GHMIG")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Unmarshal into config struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Test that environment variables override file values
	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"server.port", cfg.Server.Port, 9090},
		{"database.type", cfg.Database.Type, "postgres"},
		{"database.dsn", cfg.Database.DSN, "postgres://user:pass@host:5432/db"},
		{"source.type", cfg.Source.Type, "github"},
		{"source.base_url", cfg.Source.BaseURL, "https://source.example.com"},
		{"source.token", cfg.Source.Token, "env-source-token"},
		{"destination.type", cfg.Destination.Type, "github"},
		{"destination.base_url", cfg.Destination.BaseURL, "https://dest.example.com"},
		{"destination.token", cfg.Destination.Token, "env-dest-token"},
		{"migration.workers", cfg.Migration.Workers, 10},
		{"migration.poll_interval_seconds", cfg.Migration.PollIntervalSeconds, 60},
		{"migration.post_migration_mode", cfg.Migration.PostMigrationMode, "always"},
		{"migration.dest_repo_exists_action", cfg.Migration.DestRepoExistsAction, "skip"},
		{"migration.visibility_handling.public_repos", cfg.Migration.VisibilityHandling.PublicRepos, "public"},
		{"migration.visibility_handling.internal_repos", cfg.Migration.VisibilityHandling.InternalRepos, "internal"},
		{"logging.level", cfg.Logging.Level, "debug"},
		{"logging.format", cfg.Logging.Format, "text"},
		{"auth.enabled", cfg.Auth.Enabled, true},
		{"auth.session_duration_hours", cfg.Auth.SessionDurationHours, 48},
		{"auth.authorization_rules.require_enterprise_admin", cfg.Auth.AuthorizationRules.RequireEnterpriseAdmin, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Config %s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestLoadConfig_ArrayEnvironmentVariables(t *testing.T) {
	// Create a temporary config file with base values
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	configContent := `
auth:
  enabled: true
  authorization_rules:
    require_org_membership: []
    require_team_membership: []
`

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "single org",
			envValue: "kuhlman-labs-org",
			expected: []string{"kuhlman-labs-org"},
		},
		{
			name:     "comma-separated orgs",
			envValue: "org1,org2,org3",
			expected: []string{"org1", "org2", "org3"},
		},
		{
			name:     "JSON array format",
			envValue: `["org1","org2"]`,
			expected: []string{"org1", "org2"},
		},
		{
			name:     "comma-separated with spaces",
			envValue: "org1 , org2 , org3",
			expected: []string{"org1", "org2", "org3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			envVar := "GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ORG_MEMBERSHIP"
			if err := os.Setenv(envVar, tt.envValue); err != nil {
				t.Fatalf("Failed to set env var: %v", err)
			}
			defer os.Unsetenv(envVar)

			// Reset viper and configure it
			viper.Reset()
			viper.SetConfigFile(tmpfile.Name())
			viper.SetConfigType("yaml")
			viper.SetEnvPrefix("GHMIG")
			viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
			viper.AutomaticEnv()

			// Read config
			if err := viper.ReadInConfig(); err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			// Unmarshal into config struct
			var cfg Config
			if err := viper.Unmarshal(&cfg); err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}

			// Parse array environment variables
			cfg.ParseArrayEnvVars()

			// Verify the result
			result := cfg.Auth.AuthorizationRules.RequireOrgMembership
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d orgs, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Index %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// Tests for destination-centric authorization config

func TestAuthorizationRulesDefaults(t *testing.T) {
	viper.Reset()
	setDefaults()

	tests := []struct {
		key      string
		expected any
	}{
		{"auth.authorization_rules.allow_org_admin_migrations", true},
		{"auth.authorization_rules.allow_enterprise_admin_migrations", true},
		{"auth.authorization_rules.enable_self_service", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := viper.Get(tt.key)
			if got != tt.expected {
				t.Errorf("Default for %s = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestParseAuthorizationRules_MigrationAdminTeams(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "single team",
			envValue: "myorg/migration-admins",
			expected: []string{"myorg/migration-admins"},
		},
		{
			name:     "multiple teams comma separated",
			envValue: "org1/team1,org2/team2,org3/team3",
			expected: []string{"org1/team1", "org2/team2", "org3/team3"},
		},
		{
			name:     "teams with spaces",
			envValue: "org1/team1, org2/team2 , org3/team3",
			expected: []string{"org1/team1", "org2/team2", "org3/team3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			setDefaults()

			// Set environment variable
			t.Setenv("GHMIG_AUTH_AUTHORIZATION_RULES_MIGRATION_ADMIN_TEAMS", tt.envValue)

			// Create minimal config
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			configContent := `server:
  port: 8080
`
			if _, err := tmpfile.Write([]byte(configContent)); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			viper.SetConfigFile(tmpfile.Name())
			viper.SetEnvPrefix("GHMIG")
			viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
			viper.AutomaticEnv()
			bindEnvVars()

			if err := viper.ReadInConfig(); err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			var cfg Config
			if err := viper.Unmarshal(&cfg); err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}

			cfg.ParseArrayEnvVars()

			result := cfg.Auth.AuthorizationRules.MigrationAdminTeams
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d teams, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Index %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestParseAuthorizationRules_AllowOrgAdminMigrations(t *testing.T) {
	viper.Reset()
	setDefaults()

	// Create minimal config with the new field
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	configContent := `
auth:
  enabled: true
  authorization_rules:
    allow_org_admin_migrations: false
    allow_enterprise_admin_migrations: true
    enable_self_service: false
`
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	viper.SetConfigFile(tmpfile.Name())
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Auth.AuthorizationRules.AllowOrgAdminMigrations != false {
		t.Error("Expected AllowOrgAdminMigrations to be false")
	}
	if cfg.Auth.AuthorizationRules.AllowEnterpriseAdminMigrations != true {
		t.Error("Expected AllowEnterpriseAdminMigrations to be true")
	}
	if cfg.Auth.AuthorizationRules.EnableSelfService != false {
		t.Error("Expected EnableSelfService to be false")
	}
}

func TestParseAuthorizationRules_PrivilegedTeamsMigration(t *testing.T) {
	// Test backward compatibility: privileged_teams should be merged into migration_admin_teams
	viper.Reset()
	setDefaults()

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	configContent := `
auth:
  enabled: true
  authorization_rules:
    privileged_teams:
      - org1/old-team
      - org2/legacy-team
`
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	viper.SetConfigFile(tmpfile.Name())
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	cfg.ParseArrayEnvVars()

	// Check that privileged_teams are migrated to migration_admin_teams
	if len(cfg.Auth.AuthorizationRules.MigrationAdminTeams) != 2 {
		t.Errorf("Expected 2 migration admin teams, got %d", len(cfg.Auth.AuthorizationRules.MigrationAdminTeams))
	}
	if cfg.Auth.AuthorizationRules.MigrationAdminTeams[0] != "org1/old-team" {
		t.Errorf("Expected first team to be 'org1/old-team', got %s", cfg.Auth.AuthorizationRules.MigrationAdminTeams[0])
	}
}
