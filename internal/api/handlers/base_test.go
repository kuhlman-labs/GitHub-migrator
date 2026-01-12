package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestNewHandler(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("without GitHub clients", func(t *testing.T) {
		authConfig := &config.AuthConfig{Enabled: false}
		h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")
		if h == nil {
			t.Fatal("Expected handler to be created")
			return // Prevent staticcheck SA5011
		}
		if h.collector != nil {
			t.Error("Expected collector to be nil when GitHub clients are nil")
		}
	})

	t.Run("with GitHub clients but no source provider", func(t *testing.T) {
		sourceDualClient := createTestDualClient(t, logger)
		destDualClient := createTestDualClient(t, logger)
		authConfig := &config.AuthConfig{Enabled: false}
		h := NewHandler(db, logger, sourceDualClient, destDualClient, nil, nil, authConfig, "https://api.github.com", "github")
		if h == nil {
			t.Fatal("Expected handler to be created")
			return // Prevent staticcheck SA5011
		}
		if h.collector != nil {
			t.Error("Expected collector to be nil when source provider is nil")
		}
	})
}

func TestHealth(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}

	if response["time"] == "" {
		t.Error("Expected time to be set")
	}
}

func TestSendJSON(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}

	h.sendJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != testContentTypeJSON {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["test"] != "value" {
		t.Errorf("Expected test=value, got test=%s", response["test"])
	}
}

func TestSendError(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	h.sendError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] != "test error" {
		t.Errorf("Expected error='test error', got error='%s'", response["error"])
	}
}

func TestCanMigrate(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{string(models.StatusPending), true},             // Can queue pending repos
		{string(models.StatusDryRunQueued), true},        // Can re-queue dry runs
		{string(models.StatusDryRunFailed), true},        // Can retry failed dry runs
		{string(models.StatusDryRunComplete), true},      // Can queue after dry run
		{string(models.StatusPreMigration), false},       // Already in migration process
		{string(models.StatusMigrationFailed), true},     // Can retry failed migrations
		{string(models.StatusComplete), false},           // Already complete
		{string(models.StatusMigratingContent), false},   // Already migrating
		{string(models.StatusQueuedForMigration), false}, // Already queued
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := canMigrate(tt.status)
			if result != tt.expected {
				t.Errorf("canMigrate(%s) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}
