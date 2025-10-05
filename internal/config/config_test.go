package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestSetDefaults(t *testing.T) {
	viper.Reset()
	setDefaults()

	tests := []struct {
		key      string
		expected interface{}
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
	viper.Reset()
	viper.SetConfigFile("/nonexistent/config.yaml")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for missing config file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Write invalid YAML
	invalidYAML := `
server:
  port: not-a-number
  invalid yaml content [[[
`
	if _, err := tmpfile.Write([]byte(invalidYAML)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	viper.Reset()
	viper.SetConfigFile(tmpfile.Name())

	_, err = Load()
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}
