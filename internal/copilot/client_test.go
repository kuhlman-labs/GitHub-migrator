package copilot

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// testDatabase creates an in-memory test database
func testDatabase(t *testing.T) *storage.Database {
	t.Helper()

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// =============================================================================
// Auth Context Management Tests
// =============================================================================

func TestAuthContext_SetAndGet(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := &Client{
		db:       db,
		logger:   logger,
		sessions: make(map[string]*SDKSession),
	}

	// Initially should be nil
	if auth := client.getCurrentAuth(); auth != nil {
		t.Error("expected nil auth context initially")
	}

	// Set auth context
	authCtx := &AuthContext{
		UserID:    "123",
		UserLogin: "testuser",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	}

	client.setCurrentAuth(authCtx)

	// Should retrieve the same context
	retrieved := client.getCurrentAuth()
	if retrieved == nil {
		t.Fatal("expected auth context, got nil")
	}

	if retrieved.UserID != "123" {
		t.Errorf("expected UserID '123', got '%s'", retrieved.UserID)
	}
	if retrieved.UserLogin != "testuser" {
		t.Errorf("expected UserLogin 'testuser', got '%s'", retrieved.UserLogin)
	}
	if retrieved.Tier != "admin" {
		t.Errorf("expected Tier 'admin', got '%s'", retrieved.Tier)
	}

	// Clear auth context
	client.clearCurrentAuth()

	if auth := client.getCurrentAuth(); auth != nil {
		t.Error("expected nil auth context after clear")
	}
}

func TestAuthContext_ConcurrentAccess(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := &Client{
		db:       db,
		logger:   logger,
		sessions: make(map[string]*SDKSession),
	}

	// Test concurrent read/write access doesn't panic
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			client.setCurrentAuth(&AuthContext{
				UserID:    "user",
				UserLogin: "test",
				Tier:      "admin",
			})
			client.clearCurrentAuth()
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = client.getCurrentAuth()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// =============================================================================
// Client Configuration Tests
// =============================================================================

func TestNewClient_DefaultConfig(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := ClientConfig{}

	client := NewClient(db, logger, cfg)

	if client == nil {
		t.Fatal("expected client, got nil")
	}

	// Check defaults were applied
	if client.config.LogLevel != DefaultLogLevel {
		t.Errorf("expected default LogLevel 'info', got '%s'", client.config.LogLevel)
	}
	if client.config.SessionTimeoutMin != 30 {
		t.Errorf("expected default SessionTimeoutMin 30, got %d", client.config.SessionTimeoutMin)
	}
	if client.config.Model != DefaultModel {
		t.Errorf("expected default Model 'gpt-4.1', got '%s'", client.config.Model)
	}
}

func TestNewClient_CustomConfig(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := ClientConfig{
		CLIPath:           "/custom/path/copilot",
		Model:             "gpt-3.5-turbo",
		LogLevel:          "debug",
		SessionTimeoutMin: 60,
		Streaming:         true,
	}

	client := NewClient(db, logger, cfg)

	if client.config.CLIPath != "/custom/path/copilot" {
		t.Errorf("expected CLIPath '/custom/path/copilot', got '%s'", client.config.CLIPath)
	}
	if client.config.Model != "gpt-3.5-turbo" {
		t.Errorf("expected Model 'gpt-3.5-turbo', got '%s'", client.config.Model)
	}
	if client.config.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", client.config.LogLevel)
	}
	if client.config.SessionTimeoutMin != 60 {
		t.Errorf("expected SessionTimeoutMin 60, got %d", client.config.SessionTimeoutMin)
	}
	if !client.config.Streaming {
		t.Error("expected Streaming true, got false")
	}
}

func TestClient_IsStarted(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := NewClient(db, logger, ClientConfig{})

	// Should not be started initially
	if client.IsStarted() {
		t.Error("expected client to not be started initially")
	}
}

// =============================================================================
// Session Structure Tests
// =============================================================================

func TestSDKSession_Structure(t *testing.T) {
	session := &SDKSession{
		ID:        "test-session-id",
		UserID:    "user-123",
		UserLogin: "testuser",
		Auth: &AuthContext{
			UserID:    "user-123",
			UserLogin: "testuser",
			Tier:      "self_service",
			Permissions: ToolPermissions{
				CanRead:       true,
				CanMigrateOwn: true,
				CanMigrateAll: false,
			},
		},
	}

	if session.ID != "test-session-id" {
		t.Errorf("expected ID 'test-session-id', got '%s'", session.ID)
	}
	if session.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got '%s'", session.UserID)
	}
	if session.Auth == nil {
		t.Fatal("expected Auth context, got nil")
	}
	if session.Auth.Tier != "self_service" {
		t.Errorf("expected Auth.Tier 'self_service', got '%s'", session.Auth.Tier)
	}
}

// =============================================================================
// Tool Permissions Tests
// =============================================================================

func TestToolPermissions_AdminHasAll(t *testing.T) {
	perms := ToolPermissions{
		CanRead:           true,
		CanMigrateOwn:     true,
		CanMigrateAll:     true,
		CanManageSettings: true,
	}

	if !perms.CanRead {
		t.Error("admin should have CanRead")
	}
	if !perms.CanMigrateOwn {
		t.Error("admin should have CanMigrateOwn")
	}
	if !perms.CanMigrateAll {
		t.Error("admin should have CanMigrateAll")
	}
	if !perms.CanManageSettings {
		t.Error("admin should have CanManageSettings")
	}
}

func TestToolPermissions_SelfServiceLimits(t *testing.T) {
	perms := ToolPermissions{
		CanRead:           true,
		CanMigrateOwn:     true,
		CanMigrateAll:     false,
		CanManageSettings: false,
	}

	if !perms.CanRead {
		t.Error("self-service should have CanRead")
	}
	if !perms.CanMigrateOwn {
		t.Error("self-service should have CanMigrateOwn")
	}
	if perms.CanMigrateAll {
		t.Error("self-service should NOT have CanMigrateAll")
	}
	if perms.CanManageSettings {
		t.Error("self-service should NOT have CanManageSettings")
	}
}

func TestToolPermissions_ReadOnlyLimits(t *testing.T) {
	perms := ToolPermissions{
		CanRead:           true,
		CanMigrateOwn:     false,
		CanMigrateAll:     false,
		CanManageSettings: false,
	}

	if !perms.CanRead {
		t.Error("read-only should have CanRead")
	}
	if perms.CanMigrateOwn {
		t.Error("read-only should NOT have CanMigrateOwn")
	}
	if perms.CanMigrateAll {
		t.Error("read-only should NOT have CanMigrateAll")
	}
	if perms.CanManageSettings {
		t.Error("read-only should NOT have CanManageSettings")
	}
}

// =============================================================================
// Build System Message Tests
// =============================================================================

func TestBuildSystemMessage_WithAdminAuth(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := NewClient(db, logger, ClientConfig{})

	authCtx := &AuthContext{
		UserID:    "1",
		UserLogin: "admin",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	}

	msg := client.buildSystemMessage(context.TODO(), authCtx)

	// Should contain admin-specific guidance
	if msg == "" {
		t.Error("expected non-empty system message")
	}
	if !containsString(msg, "administrator") {
		t.Error("expected admin message to mention 'administrator'")
	}
}

func TestBuildSystemMessage_WithSelfServiceAuth(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := NewClient(db, logger, ClientConfig{})

	authCtx := &AuthContext{
		UserID:    "2",
		UserLogin: "selfservice",
		Tier:      "self_service",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: false,
		},
	}

	msg := client.buildSystemMessage(context.TODO(), authCtx)

	if !containsString(msg, "self-service") {
		t.Error("expected self-service message to mention 'self-service'")
	}
}

func TestBuildSystemMessage_WithReadOnlyAuth(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := NewClient(db, logger, ClientConfig{})

	authCtx := &AuthContext{
		UserID:    "3",
		UserLogin: "readonly",
		Tier:      "read_only",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: false,
			CanMigrateAll: false,
		},
	}

	msg := client.buildSystemMessage(context.TODO(), authCtx)

	if !containsString(msg, "read-only") {
		t.Error("expected read-only message to mention 'read-only'")
	}
}

func TestBuildSystemMessage_NilAuth(t *testing.T) {
	db := testDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := NewClient(db, logger, ClientConfig{})

	msg := client.buildSystemMessage(context.TODO(), nil)

	// Should still return base message
	if msg == "" {
		t.Error("expected non-empty system message even with nil auth")
	}
	if !containsString(msg, "GitHub Migrator Copilot") {
		t.Error("expected base message to contain 'GitHub Migrator Copilot'")
	}
}

// =============================================================================
// Stream Event Tests
// =============================================================================

func TestStreamEventType_Constants(t *testing.T) {
	// Verify stream event type constants exist and have expected values
	tests := []struct {
		eventType StreamEventType
		expected  string
	}{
		{StreamEventDelta, "delta"},
		{StreamEventToolCall, "tool_call"},
		{StreamEventToolResult, "tool_result"},
		{StreamEventDone, "done"},
		{StreamEventError, "error"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.eventType)
		}
	}
}

func TestStreamEvent_Structure(t *testing.T) {
	event := StreamEvent{
		Type: StreamEventDelta,
		Data: StreamEventData{
			Content: "Hello, world!",
		},
	}

	if event.Type != StreamEventDelta {
		t.Errorf("expected type 'delta', got '%s'", event.Type)
	}
	if event.Data.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got '%s'", event.Data.Content)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
