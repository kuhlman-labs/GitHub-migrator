package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestStartDiscovery(t *testing.T) {
	testStartDiscoveryWithoutClient(t)
	testStartDiscoveryValidation(t)
	testStartDiscoveryOrganization(t)
	testStartDiscoveryEnterprise(t)
}

func testStartDiscoveryWithoutClient(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")

	reqBody := map[string]any{
		"organization": "test-org",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.StartDiscovery(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func testStartDiscoveryValidation(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	sourceDualClient := createTestDualClient(t, logger)
	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, logger, sourceDualClient, nil, nil, nil, authConfig, "https://api.github.com", "github")

	tests := []struct {
		name     string
		reqBody  map[string]any
		rawBody  string
		wantCode int
	}{
		{
			name:     "missing both",
			reqBody:  map[string]any{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "both provided",
			reqBody: map[string]any{
				"organization":    "test-org",
				"enterprise_slug": "test-enterprise",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid json",
			rawBody:  "invalid json",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.rawBody != "" {
				body = []byte(tt.rawBody)
			} else {
				body, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.StartDiscovery(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

func testStartDiscoveryOrganization(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	sourceDualClient := createTestDualClient(t, logger)
	mockProvider := &mockSourceProvider{}
	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, logger, sourceDualClient, nil, mockProvider, nil, authConfig, "https://api.github.com", "github")

	reqBody := map[string]any{
		"organization": "test-org",
		"workers":      10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.StartDiscovery(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response map[string]any
	_ = json.NewDecoder(w.Body).Decode(&response)

	if response["type"] != "organization" {
		t.Errorf("Expected type 'organization', got %v", response["type"])
	}
	if response["organization"] != "test-org" {
		t.Errorf("Expected organization 'test-org', got %v", response["organization"])
	}
}

func testStartDiscoveryEnterprise(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	sourceDualClient := createTestDualClient(t, logger)
	mockProvider := &mockSourceProvider{}
	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, logger, sourceDualClient, nil, mockProvider, nil, authConfig, "https://api.github.com", "github")

	reqBody := map[string]any{
		"enterprise_slug": "test-enterprise",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/start", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.StartDiscovery(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response map[string]any
	_ = json.NewDecoder(w.Body).Decode(&response)

	if response["type"] != "enterprise" {
		t.Errorf("Expected type 'enterprise', got %v", response["type"])
	}
	if response["enterprise"] != "test-enterprise" {
		t.Errorf("Expected enterprise 'test-enterprise', got %v", response["enterprise"])
	}
}

func TestDiscoveryStatus(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Add some repositories
	repo1 := &models.Repository{FullName: "org/repo1", Status: string(models.StatusPending)}
	repo2 := &models.Repository{FullName: "org/repo2", Status: string(models.StatusPending)}
	_ = db.SaveRepository(ctx, repo1)
	_ = db.SaveRepository(ctx, repo2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/status", nil)
	w := httptest.NewRecorder()

	h.DiscoveryStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["repositories_found"] != float64(2) {
		t.Errorf("Expected 2 repositories, got %v", response["repositories_found"])
	}
}

func TestCancelDiscovery(t *testing.T) {
	t.Run("NoActiveDiscovery", testCancelDiscoveryNoActiveDiscovery)
	t.Run("Success", testCancelDiscoverySuccess)
	t.Run("CancelFunctionNotFound", testCancelDiscoveryCancelFunctionNotFound)
}

func testCancelDiscoveryNoActiveDiscovery(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/cancel", nil)
	w := httptest.NewRecorder()

	h.CancelDiscovery(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] == nil {
		t.Error("Expected error in response")
	}
}

func testCancelDiscoverySuccess(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")

	// Create an active discovery progress record
	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		Status:        models.DiscoveryStatusInProgress,
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery progress: %v", err)
	}

	// Register a cancel function for this discovery
	cancelled := false
	h.discoveryMu.Lock()
	h.discoveryCancel[progress.ID] = func() { cancelled = true }
	h.discoveryMu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/cancel", nil)
	w := httptest.NewRecorder()

	h.CancelDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "Discovery cancellation initiated" {
		t.Errorf("Expected message 'Discovery cancellation initiated', got %v", response["message"])
	}
	if response["progress_id"] != float64(progress.ID) {
		t.Errorf("Expected progress_id %d, got %v", progress.ID, response["progress_id"])
	}
	if response["status"] != "cancelling" {
		t.Errorf("Expected status 'cancelling', got %v", response["status"])
	}
	if !cancelled {
		t.Error("Expected cancel function to be called")
	}

	// Verify phase was updated to cancelling
	updatedProgress, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get updated progress: %v", err)
	}
	if updatedProgress.Phase != models.PhaseCancelling {
		t.Errorf("Expected phase '%s', got '%s'", models.PhaseCancelling, updatedProgress.Phase)
	}
}

func testCancelDiscoveryCancelFunctionNotFound(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")

	// Create an active discovery progress record but don't register a cancel function
	// This simulates a discovery that was started by a different server instance
	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		Status:        models.DiscoveryStatusInProgress,
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery progress: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/cancel", nil)
	w := httptest.NewRecorder()

	h.CancelDiscovery(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should indicate cancel function not found
	if response["error"] == nil {
		t.Error("Expected error in response")
	}
}

func TestForceResetDiscovery(t *testing.T) {
	t.Run("NoStuckDiscovery", testForceResetDiscoveryNoStuck)
	t.Run("ResetsStuckDiscovery", testForceResetDiscoverySuccess)
	t.Run("CleansUpCancelFunction", testForceResetDiscoveryCleansUpCancel)
	t.Run("RaceConditionHandled", testForceResetDiscoveryRaceCondition)
}

func testForceResetDiscoveryNoStuck(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/force-reset", nil)
	w := httptest.NewRecorder()

	h.ForceResetDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "No stuck discovery found" {
		t.Errorf("Expected message 'No stuck discovery found', got %v", response["message"])
	}
	if response["records_reset"] != float64(0) {
		t.Errorf("Expected records_reset 0, got %v", response["records_reset"])
	}
	if response["action_taken"] != false {
		t.Errorf("Expected action_taken false, got %v", response["action_taken"])
	}
}

func testForceResetDiscoverySuccess(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")

	// Create an active discovery progress record (simulating stuck state)
	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "stuck-org",
		Status:        models.DiscoveryStatusInProgress,
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery progress: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/force-reset", nil)
	w := httptest.NewRecorder()

	h.ForceResetDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["message"] != "Discovery force reset completed" {
		t.Errorf("Expected message 'Discovery force reset completed', got %v", response["message"])
	}
	if response["records_reset"] != float64(1) {
		t.Errorf("Expected records_reset 1, got %v", response["records_reset"])
	}
	if response["action_taken"] != true {
		t.Errorf("Expected action_taken true, got %v", response["action_taken"])
	}

	// Verify reset_discovery contains the expected fields
	resetDiscovery, ok := response["reset_discovery"].(map[string]any)
	if !ok {
		t.Fatal("Expected reset_discovery to be a map")
	}
	if resetDiscovery["id"] != float64(progress.ID) {
		t.Errorf("Expected reset_discovery.id %d, got %v", progress.ID, resetDiscovery["id"])
	}
	if resetDiscovery["target"] != "stuck-org" {
		t.Errorf("Expected reset_discovery.target 'stuck-org', got %v", resetDiscovery["target"])
	}

	// Verify no active discovery anymore
	active, err := db.GetActiveDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get active discovery: %v", err)
	}
	if active != nil {
		t.Error("Expected no active discovery after force reset")
	}

	// Verify the discovery was marked as cancelled
	latest, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get latest discovery: %v", err)
	}
	if latest.Status != models.DiscoveryStatusCancelled {
		t.Errorf("Expected status '%s', got '%s'", models.DiscoveryStatusCancelled, latest.Status)
	}
}

func testForceResetDiscoveryCleansUpCancel(t *testing.T) {
	db := setupTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	authConfig := &config.AuthConfig{Enabled: false}

	h := NewHandler(db, logger, nil, nil, nil, nil, authConfig, "https://api.github.com", "github")

	// Create an active discovery progress record
	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "stuck-org",
		Status:        models.DiscoveryStatusInProgress,
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery progress: %v", err)
	}

	// Register a cancel function (simulating a discovery that was running)
	h.discoveryMu.Lock()
	h.discoveryCancel[progress.ID] = func() {}
	h.discoveryMu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/force-reset", nil)
	w := httptest.NewRecorder()

	h.ForceResetDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify the cancel function was removed
	h.discoveryMu.RLock()
	_, exists := h.discoveryCancel[progress.ID]
	h.discoveryMu.RUnlock()

	if exists {
		t.Error("Expected cancel function to be removed after force reset")
	}
}

// testForceResetDiscoveryRaceCondition tests the race condition where
// a discovery completes between GetActiveDiscoveryProgress and ForceResetDiscovery.
func testForceResetDiscoveryRaceCondition(t *testing.T) {
	// Create a mock that simulates the race condition:
	// - GetActiveDiscoveryProgress returns a progress (discovery was in progress when checked)
	// - ForceResetDiscovery returns 0 (discovery completed before the reset was applied)
	mock := NewMockDataStore()
	mock.ActiveDiscoveryProgress = &models.DiscoveryProgress{
		ID:            42,
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "race-condition-org",
		Status:        models.DiscoveryStatusInProgress,
		Phase:         models.PhaseProfilingRepos,
	}
	mock.ForceResetDiscoveryResult = 0 // Simulates the race: discovery already completed

	h := setupTestHandlerWithMock(t, mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/force-reset", nil)
	w := httptest.NewRecorder()

	h.ForceResetDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify the response correctly indicates the race condition was handled
	if response["message"] != "Discovery already completed before reset was applied" {
		t.Errorf("Expected message 'Discovery already completed before reset was applied', got %v", response["message"])
	}
	if response["records_reset"] != float64(0) {
		t.Errorf("Expected records_reset 0, got %v", response["records_reset"])
	}
	if response["action_taken"] != false {
		t.Errorf("Expected action_taken false (race condition handled), got %v", response["action_taken"])
	}

	// Verify discovery info is included for context
	discovery, ok := response["discovery"].(map[string]any)
	if !ok {
		t.Fatal("Expected discovery to be present in response")
	}
	if discovery["id"] != float64(42) {
		t.Errorf("Expected discovery.id 42, got %v", discovery["id"])
	}
	if discovery["target"] != "race-condition-org" {
		t.Errorf("Expected discovery.target 'race-condition-org', got %v", discovery["target"])
	}
}
