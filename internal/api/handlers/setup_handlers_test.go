package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// ============================================================================
// escapeEnvValue Tests
// ============================================================================

func TestEscapeEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "value with newline",
			input:    "line1\nline2",
			expected: `line1\nline2`,
		},
		{
			name:     "value with carriage return",
			input:    "line1\rline2",
			expected: `line1\rline2`,
		},
		{
			name:     "value with CRLF",
			input:    "line1\r\nline2",
			expected: `line1\r\nline2`,
		},
		{
			name:     "value with double quotes",
			input:    `say "hello"`,
			expected: `say \"hello\"`,
		},
		{
			name:     "value with backslash",
			input:    `path\to\file`,
			expected: `path\\to\\file`,
		},
		{
			name:     "value with backslash before n",
			input:    `literal\n`,
			expected: `literal\\n`,
		},
		{
			name:     "value with all special characters",
			input:    "a\\b\"c\nd\re",
			expected: `a\\b\"c\nd\re`,
		},
		{
			name:     "empty value",
			input:    "",
			expected: "",
		},
		{
			name:     "GitHub token",
			input:    "ghp_1234567890abcdefghij",
			expected: "ghp_1234567890abcdefghij",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeEnvValue(tt.input)
			if result != tt.expected {
				t.Errorf("escapeEnvValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// writeQuotedEnv Tests
// ============================================================================

func TestWriteQuotedEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "simple value",
			key:      "MY_KEY",
			value:    "my_value",
			expected: "MY_KEY=\"my_value\"\n",
		},
		{
			name:     "value with newline is escaped and quoted",
			key:      "TOKEN",
			value:    "real_token\nINJECTED_KEY=evil",
			expected: "TOKEN=\"real_token\\nINJECTED_KEY=evil\"\n",
		},
		{
			name:     "value with double quotes",
			key:      "DSN",
			value:    `host=db user="admin"`,
			expected: "DSN=\"host=db user=\\\"admin\\\"\"\n",
		},
		{
			name:     "empty value",
			key:      "EMPTY",
			value:    "",
			expected: "EMPTY=\"\"\n",
		},
		{
			name:     "value with backslash",
			key:      "PATH",
			value:    `C:\Users\test`,
			expected: "PATH=\"C:\\\\Users\\\\test\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			writeQuotedEnv(&sb, tt.key, tt.value)
			result := sb.String()
			if result != tt.expected {
				t.Errorf("writeQuotedEnv(%q, %q) = %q, want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// generateEnvFile Tests
// ============================================================================

func TestGenerateEnvFile_ValuesAreQuoted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := &SetupHandler{
		db:     nil, // generateEnvFile does not use the DB
		logger: logger,
	}

	cfg := SetupConfig{
		Source: SourceConfigData{
			Type:    models.SourceTypeGitHub,
			BaseURL: "https://github.example.com",
			Token:   "ghp_sourcetoken123",
		},
		Destination: DestinationConfigData{
			BaseURL: "https://github.com",
			Token:   "ghp_desttoken456",
		},
		Database: DatabaseConfigData{
			Type: "sqlite",
			DSN:  "data/migrator.db",
		},
		Server: ServerConfigData{
			Port: 8080,
		},
		Migration: MigrationConfigData{
			Workers:              2,
			PollIntervalSeconds:  30,
			DestRepoExistsAction: "skip",
			VisibilityHandling: VisibilityHandlingConfigData{
				PublicRepos:   "private",
				InternalRepos: "private",
			},
		},
		Logging: LoggingConfigData{
			Level:      "info",
			Format:     "json",
			OutputFile: "logs/migrator.log",
		},
	}

	result := handler.generateEnvFile(cfg)

	// All string values should be double-quoted
	quotedValues := []string{
		`GHMIG_SOURCE_TYPE="github"`,
		`GHMIG_SOURCE_BASE_URL="https://github.example.com"`,
		`GHMIG_SOURCE_TOKEN="ghp_sourcetoken123"`,
		`GHMIG_DESTINATION_BASE_URL="https://github.com"`,
		`GHMIG_DESTINATION_TOKEN="ghp_desttoken456"`,
		`GHMIG_DATABASE_TYPE="sqlite"`,
		`GHMIG_DATABASE_DSN="data/migrator.db"`,
		`GHMIG_MIGRATION_DEST_REPO_EXISTS_ACTION="skip"`,
		`GHMIG_MIGRATION_VISIBILITY_HANDLING_PUBLIC_REPOS="private"`,
		`GHMIG_MIGRATION_VISIBILITY_HANDLING_INTERNAL_REPOS="private"`,
		`GHMIG_LOGGING_LEVEL="info"`,
		`GHMIG_LOGGING_FORMAT="json"`,
		`GHMIG_LOGGING_OUTPUT_FILE="logs/migrator.log"`,
	}

	for _, expected := range quotedValues {
		if !strings.Contains(result, expected) {
			t.Errorf("generateEnvFile output missing %q.\nGot:\n%s", expected, result)
		}
	}

	// Integer values should NOT be quoted
	intValues := []string{
		"GHMIG_SERVER_PORT=8080",
		"GHMIG_MIGRATION_WORKERS=2",
		"GHMIG_MIGRATION_POLL_INTERVAL_SECONDS=30",
	}
	for _, expected := range intValues {
		if !strings.Contains(result, expected) {
			t.Errorf("generateEnvFile output missing %q.\nGot:\n%s", expected, result)
		}
	}
}

func TestGenerateEnvFile_NewlineInjectionPrevented(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := &SetupHandler{
		db:     nil,
		logger: logger,
	}

	// Attempt to inject a new env var via a token with embedded newline
	maliciousToken := "ghp_real\nGHMIG_AUTH_ENABLED=false"

	cfg := SetupConfig{
		Source: SourceConfigData{
			Type:    models.SourceTypeGitHub,
			BaseURL: "https://github.com",
			Token:   maliciousToken,
		},
		Destination: DestinationConfigData{
			BaseURL: "https://github.com",
			Token:   "ghp_dest",
		},
		Database: DatabaseConfigData{
			Type: "sqlite",
			DSN:  "data/test.db",
		},
		Server:    ServerConfigData{Port: 8080},
		Migration: MigrationConfigData{Workers: 1, PollIntervalSeconds: 30},
		Logging:   LoggingConfigData{Level: "info", Format: "json"},
	}

	result := handler.generateEnvFile(cfg)

	// The injected key should NOT appear as a separate line
	if strings.Contains(result, "\nGHMIG_AUTH_ENABLED=false") {
		t.Errorf("newline injection succeeded: malicious env var was injected.\nGenerated:\n%s", result)
	}

	// The escaped value should be contained within quotes on a single line
	if !strings.Contains(result, `GHMIG_SOURCE_TOKEN="ghp_real\nGHMIG_AUTH_ENABLED=false"`) {
		t.Errorf("token was not properly escaped and quoted.\nGenerated:\n%s", result)
	}
}

func TestGenerateEnvFile_AuthConfigQuoted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := &SetupHandler{
		db:     nil,
		logger: logger,
	}

	cfg := SetupConfig{
		Source: SourceConfigData{
			Type:    models.SourceTypeGitHub,
			BaseURL: "https://github.com",
			Token:   "ghp_src",
		},
		Destination: DestinationConfigData{
			BaseURL: "https://github.com",
			Token:   "ghp_dest",
		},
		Database: DatabaseConfigData{
			Type: "sqlite",
			DSN:  "data/test.db",
		},
		Server:    ServerConfigData{Port: 8080},
		Migration: MigrationConfigData{Workers: 1, PollIntervalSeconds: 30},
		Logging:   LoggingConfigData{Level: "info", Format: "json"},
		Auth: &AuthConfigData{
			Enabled:                 true,
			GitHubOAuthClientID:     "client_id_123",
			GitHubOAuthClientSecret: "client_secret_456",
			GitHubOAuthBaseURL:      "https://github.example.com",
			SessionSecret:           "super-secret-session-key-32chars!",
			CallbackURL:             "https://app.example.com/callback",
			FrontendURL:             "https://app.example.com",
			SessionDurationHours:    24,
		},
	}

	result := handler.generateEnvFile(cfg)

	quotedAuthValues := []string{
		`GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID="client_id_123"`,
		`GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET="client_secret_456"`,
		`GHMIG_AUTH_GITHUB_OAUTH_BASE_URL="https://github.example.com"`,
		`GHMIG_AUTH_SESSION_SECRET="super-secret-session-key-32chars!"`,
		`GHMIG_AUTH_CALLBACK_URL="https://app.example.com/callback"`,
		`GHMIG_AUTH_FRONTEND_URL="https://app.example.com"`,
	}

	for _, expected := range quotedAuthValues {
		if !strings.Contains(result, expected) {
			t.Errorf("generateEnvFile output missing %q.\nGot:\n%s", expected, result)
		}
	}
}

func TestGenerateEnvFile_SkipsSourceWhenPlaceholder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := &SetupHandler{
		db:     nil,
		logger: logger,
	}

	cfg := SetupConfig{
		Source: SourceConfigData{
			Type:    models.SourceTypeGitHub,
			BaseURL: "https://github.com",
			Token:   "placeholder",
		},
		Destination: DestinationConfigData{
			BaseURL: "https://github.com",
			Token:   "ghp_dest",
		},
		Database:  DatabaseConfigData{Type: "sqlite", DSN: "data/test.db"},
		Server:    ServerConfigData{Port: 8080},
		Migration: MigrationConfigData{Workers: 1, PollIntervalSeconds: 30},
		Logging:   LoggingConfigData{Level: "info", Format: "json"},
	}

	result := handler.generateEnvFile(cfg)

	if strings.Contains(result, "GHMIG_SOURCE_TOKEN") {
		t.Errorf("source token should be skipped for placeholder, but was written.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "configure via Sources page") {
		t.Errorf("expected placeholder comment for source config.\nGot:\n%s", result)
	}
}

// ============================================================================
// ApplySetup Guard Tests
// ============================================================================

func TestApplySetup_RejectsWhenSetupAlreadyComplete(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	shutdownChan := make(chan struct{})

	handler := NewSetupHandler(db, logger, nil, shutdownChan)

	// Mark setup as already complete
	if err := db.MarkSetupComplete(); err != nil {
		t.Fatalf("Failed to mark setup complete: %v", err)
	}

	// Build a valid setup request
	setupCfg := SetupConfig{
		Source: SourceConfigData{
			Type:    "github",
			BaseURL: "https://github.com",
			Token:   "ghp_test123",
		},
		Destination: DestinationConfigData{
			BaseURL: "https://github.com",
			Token:   "ghp_dest456",
		},
		Database: DatabaseConfigData{
			Type: "sqlite",
			DSN:  "data/test.db",
		},
	}

	body, err := json.Marshal(setupCfg)
	if err != nil {
		t.Fatalf("Failed to marshal setup config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", testContentTypeJSON)
	w := httptest.NewRecorder()

	handler.ApplySetup(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Setup has already been completed") {
		t.Errorf("expected rejection message, got: %s", w.Body.String())
	}
}

func TestApplySetup_AllowedWhenSetupNotComplete(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	shutdownChan := make(chan struct{})

	handler := NewSetupHandler(db, logger, nil, shutdownChan)

	// Do NOT mark setup as complete -- default state is not complete

	// Build a valid setup request (will fail at writeEnvFile since we're in a test,
	// but the guard should NOT block it)
	setupCfg := SetupConfig{
		Source: SourceConfigData{
			Type:    "github",
			BaseURL: "https://github.com",
			Token:   "ghp_test123",
		},
		Destination: DestinationConfigData{
			BaseURL: "https://github.com",
			Token:   "ghp_dest456",
		},
		Database: DatabaseConfigData{
			Type: "sqlite",
			DSN:  "data/test.db",
		},
	}

	body, err := json.Marshal(setupCfg)
	if err != nil {
		t.Fatalf("Failed to marshal setup config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", testContentTypeJSON)
	w := httptest.NewRecorder()

	handler.ApplySetup(w, req)

	// It should NOT return 403 (it may fail later due to file I/O in tests, but
	// the guard itself should not block the request)
	if w.Code == http.StatusForbidden {
		t.Errorf("setup guard should not block when setup is not yet complete, got 403")
	}
}
