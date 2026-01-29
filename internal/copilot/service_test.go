package copilot

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// testServiceDatabase creates an in-memory test database for service tests
func testServiceDatabase(t *testing.T) *storage.Database {
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
// Service Creation Tests
// =============================================================================

func TestNewService_DefaultConfig(t *testing.T) {
	db := testServiceDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := ServiceConfig{}

	service := NewService(db, logger, cfg)

	if service == nil {
		t.Fatal("expected service, got nil")
	}
	if service.client == nil {
		t.Error("expected service to have client")
	}
	if service.db == nil {
		t.Error("expected service to have database")
	}
}

func TestNewService_CustomConfig(t *testing.T) {
	db := testServiceDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := ServiceConfig{
		CLIPath:           "/custom/copilot",
		Model:             "gpt-4",
		SessionTimeoutMin: 45,
		Streaming:         true,
		LogLevel:          "debug",
	}

	service := NewService(db, logger, cfg)

	if service == nil {
		t.Fatal("expected service, got nil")
	}

	// Verify config was passed to client
	client := service.GetClient()
	if client.config.CLIPath != "/custom/copilot" {
		t.Errorf("expected CLIPath '/custom/copilot', got '%s'", client.config.CLIPath)
	}
	if client.config.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got '%s'", client.config.Model)
	}
	if client.config.SessionTimeoutMin != 45 {
		t.Errorf("expected SessionTimeoutMin 45, got %d", client.config.SessionTimeoutMin)
	}
	if !client.config.Streaming {
		t.Error("expected Streaming true")
	}
	if client.config.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", client.config.LogLevel)
	}
}

// =============================================================================
// Status Tests
// =============================================================================

func TestService_GetStatus_CopilotDisabled(t *testing.T) {
	db := testServiceDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	service := NewService(db, logger, ServiceConfig{})

	settings := &models.Settings{
		CopilotEnabled: false,
	}

	status, err := service.GetStatus(ctx, "testuser", "token", settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Available {
		t.Error("expected Available false when Copilot disabled")
	}
	if status.UnavailableReason != "Copilot is not enabled in settings" {
		t.Errorf("unexpected reason: %s", status.UnavailableReason)
	}
}

func TestService_GetStatus_CLINotInstalled(t *testing.T) {
	db := testServiceDatabase(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	service := NewService(db, logger, ServiceConfig{})

	// Copilot enabled but CLI not available
	settings := &models.Settings{
		CopilotEnabled: true,
	}

	status, err := service.GetStatus(ctx, "testuser", "token", settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In test environment, CLI is not installed
	if status.CLIInstalled {
		// If CLI happens to be installed, just verify the field is set
		t.Log("CLI is installed in test environment")
	} else {
		if status.Available {
			t.Error("expected Available false when CLI not installed")
		}
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestCanQueueForMigration(t *testing.T) {
	tests := []struct {
		status   string
		dryRun   bool
		expected bool
	}{
		{string(models.StatusPending), false, true},
		{string(models.StatusPending), true, true},
		{string(models.StatusDryRunFailed), false, true},
		{string(models.StatusMigrationFailed), false, true},
		{string(models.StatusRolledBack), false, true},
		{string(models.StatusDryRunComplete), false, true},    // Can migrate after dry run
		{string(models.StatusDryRunComplete), true, false},    // Can't do another dry run
		{string(models.StatusComplete), false, false},         // Already complete
		{string(models.StatusMigratingContent), false, false}, // In progress
	}

	for _, tt := range tests {
		t.Run(tt.status+"_dryRun_"+boolToStr(tt.dryRun), func(t *testing.T) {
			result := canQueueForMigration(tt.status, tt.dryRun)
			if result != tt.expected {
				t.Errorf("canQueueForMigration(%s, %v) = %v, want %v",
					tt.status, tt.dryRun, result, tt.expected)
			}
		})
	}
}

func TestIsInQueuedOrInProgressState(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{string(models.StatusDryRunQueued), true},
		{string(models.StatusDryRunInProgress), true},
		{string(models.StatusQueuedForMigration), true},
		{string(models.StatusMigratingContent), true},
		{string(models.StatusArchiveGenerating), true},
		{string(models.StatusPreMigration), true},
		{string(models.StatusPending), false},
		{string(models.StatusComplete), false},
		{string(models.StatusMigrationFailed), false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := isInQueuedOrInProgressState(tt.status)
			if result != tt.expected {
				t.Errorf("isInQueuedOrInProgressState(%s) = %v, want %v",
					tt.status, result, tt.expected)
			}
		})
	}
}

func TestCalculateProgress(t *testing.T) {
	statuses := []string{
		string(models.StatusPending),
		string(models.StatusPending),
		string(models.StatusDryRunQueued),
		string(models.StatusDryRunInProgress),
		string(models.StatusComplete),
		string(models.StatusComplete),
		string(models.StatusComplete),
		string(models.StatusMigrationFailed),
		string(models.StatusWontMigrate),
	}

	progress := calculateProgress(statuses)

	if progress["total_count"] != 9 {
		t.Errorf("expected total_count 9, got %v", progress["total_count"])
	}
	if progress["pending_count"] != 2 {
		t.Errorf("expected pending_count 2, got %v", progress["pending_count"])
	}
	if progress["queued_count"] != 1 {
		t.Errorf("expected queued_count 1, got %v", progress["queued_count"])
	}
	if progress["in_progress_count"] != 1 {
		t.Errorf("expected in_progress_count 1, got %v", progress["in_progress_count"])
	}
	if progress["completed_count"] != 3 {
		t.Errorf("expected completed_count 3, got %v", progress["completed_count"])
	}
	if progress["failed_count"] != 1 {
		t.Errorf("expected failed_count 1, got %v", progress["failed_count"])
	}
	if progress["skipped_count"] != 1 {
		t.Errorf("expected skipped_count 1, got %v", progress["skipped_count"])
	}

	// Verify percentage
	percentComplete := progress["percent_complete"].(float64)
	expectedPercent := float64(3) / float64(9) * 100
	if percentComplete != expectedPercent {
		t.Errorf("expected percent_complete %.2f, got %.2f", expectedPercent, percentComplete)
	}
}

func TestCalculateProgress_Empty(t *testing.T) {
	progress := calculateProgress([]string{})

	if progress["total_count"] != 0 {
		t.Errorf("expected total_count 0, got %v", progress["total_count"])
	}
	if progress["percent_complete"] != 0.0 {
		t.Errorf("expected percent_complete 0, got %v", progress["percent_complete"])
	}
}

func TestCalculateProgress_AllComplete(t *testing.T) {
	statuses := []string{
		string(models.StatusComplete),
		string(models.StatusComplete),
		string(models.StatusMigrationComplete),
	}

	progress := calculateProgress(statuses)

	if progress["completed_count"] != 3 {
		t.Errorf("expected completed_count 3, got %v", progress["completed_count"])
	}
	if progress["percent_complete"] != 100.0 {
		t.Errorf("expected percent_complete 100, got %v", progress["percent_complete"])
	}
}

// =============================================================================
// Helper
// =============================================================================

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
