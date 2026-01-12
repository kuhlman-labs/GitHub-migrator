package storage

import (
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func setupTestDBForDiscovery(t *testing.T) *Database {
	t.Helper()

	dbCfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := NewDatabase(dbCfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func TestCreateDiscoveryProgress(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}

	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery progress: %v", err)
	}

	if progress.ID == 0 {
		t.Error("Expected progress ID to be set")
	}
	if progress.Status != models.DiscoveryStatusInProgress {
		t.Errorf("Expected status '%s', got '%s'", models.DiscoveryStatusInProgress, progress.Status)
	}
	if progress.Phase != models.PhaseListingRepos {
		t.Errorf("Expected phase '%s', got '%s'", models.PhaseListingRepos, progress.Phase)
	}
}

func TestCreateDiscoveryProgress_AlreadyInProgress(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	// Create first discovery
	progress1 := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress1); err != nil {
		t.Fatalf("Failed to create first discovery progress: %v", err)
	}

	// Try to create another - should fail
	progress2 := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "another-org",
		TotalOrgs:     1,
	}
	err := db.CreateDiscoveryProgress(progress2)
	if err == nil {
		t.Error("Expected error when creating second discovery while one is in progress")
	}
}

func TestGetActiveDiscoveryProgress(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	// No active discovery initially
	progress, err := db.GetActiveDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get active discovery: %v", err)
	}
	if progress != nil {
		t.Error("Expected no active discovery initially")
	}

	// Create a discovery
	newProgress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(newProgress); err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}

	// Should now have an active discovery
	active, err := db.GetActiveDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get active discovery: %v", err)
	}
	if active == nil {
		t.Fatal("Expected active discovery")
		return // Explicitly unreachable, but satisfies static analysis
	}
	if active.ID != newProgress.ID {
		t.Errorf("Expected ID %d, got %d", newProgress.ID, active.ID)
	}
}

//nolint:dupl // Test cases have similar structure but test different states
func TestMarkDiscoveryComplete(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}

	if err := db.MarkDiscoveryComplete(progress.ID); err != nil {
		t.Fatalf("Failed to mark discovery complete: %v", err)
	}

	completed, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get completed discovery: %v", err)
	}
	if completed.Status != models.DiscoveryStatusCompleted {
		t.Errorf("Expected status '%s', got '%s'", models.DiscoveryStatusCompleted, completed.Status)
	}
	if completed.Phase != models.PhaseCompleted {
		t.Errorf("Expected phase '%s', got '%s'", models.PhaseCompleted, completed.Phase)
	}
	if completed.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestMarkDiscoveryFailed(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}

	errorMsg := "something went wrong"
	if err := db.MarkDiscoveryFailed(progress.ID, errorMsg); err != nil {
		t.Fatalf("Failed to mark discovery failed: %v", err)
	}

	failed, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get failed discovery: %v", err)
	}
	if failed.Status != models.DiscoveryStatusFailed {
		t.Errorf("Expected status '%s', got '%s'", models.DiscoveryStatusFailed, failed.Status)
	}
	if failed.LastError == nil || *failed.LastError != errorMsg {
		t.Errorf("Expected LastError '%s', got %v", errorMsg, failed.LastError)
	}
	if failed.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

//nolint:dupl // Test cases have similar structure but test different states
func TestMarkDiscoveryCancelled(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}

	if err := db.MarkDiscoveryCancelled(progress.ID); err != nil {
		t.Fatalf("Failed to mark discovery cancelled: %v", err)
	}

	cancelled, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get cancelled discovery: %v", err)
	}
	if cancelled.Status != models.DiscoveryStatusCancelled {
		t.Errorf("Expected status '%s', got '%s'", models.DiscoveryStatusCancelled, cancelled.Status)
	}
	if cancelled.Phase != models.PhaseCancelling {
		t.Errorf("Expected phase '%s', got '%s'", models.PhaseCancelling, cancelled.Phase)
	}
	if cancelled.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestMarkDiscoveryCancelled_NotFound(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	// Try to mark non-existent discovery as cancelled
	err := db.MarkDiscoveryCancelled(9999)
	if err == nil {
		t.Error("Expected error when marking non-existent discovery as cancelled")
	}
}

func TestDeleteCompletedDiscoveryProgress(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	// Create and complete a discovery
	progress1 := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org-1",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress1); err != nil {
		t.Fatalf("Failed to create discovery 1: %v", err)
	}
	if err := db.MarkDiscoveryComplete(progress1.ID); err != nil {
		t.Fatalf("Failed to complete discovery 1: %v", err)
	}

	// Delete completed discoveries
	if err := db.DeleteCompletedDiscoveryProgress(); err != nil {
		t.Fatalf("Failed to delete completed discoveries: %v", err)
	}

	// Should be no discoveries left
	latest, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get latest: %v", err)
	}
	if latest != nil {
		t.Error("Expected no discoveries after deletion")
	}
}

func TestDeleteCompletedDiscoveryProgress_IncludesCancelled(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	// Create and cancel a discovery
	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	if err := db.MarkDiscoveryCancelled(progress.ID); err != nil {
		t.Fatalf("Failed to cancel discovery: %v", err)
	}

	// Delete completed (which includes cancelled) discoveries
	if err := db.DeleteCompletedDiscoveryProgress(); err != nil {
		t.Fatalf("Failed to delete completed discoveries: %v", err)
	}

	// Should be no discoveries left
	latest, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get latest: %v", err)
	}
	if latest != nil {
		t.Error("Expected cancelled discovery to be deleted")
	}
}

func TestUpdateDiscoveryPhase(t *testing.T) {
	db := setupTestDBForDiscovery(t)
	defer func() { _ = db.Close() }()

	progress := &models.DiscoveryProgress{
		DiscoveryType: models.DiscoveryTypeOrganization,
		Target:        "test-org",
		TotalOrgs:     1,
	}
	if err := db.CreateDiscoveryProgress(progress); err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}

	// Update to profiling phase
	if err := db.UpdateDiscoveryPhase(progress.ID, models.PhaseProfilingRepos); err != nil {
		t.Fatalf("Failed to update phase: %v", err)
	}

	updated, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get updated discovery: %v", err)
	}
	if updated.Phase != models.PhaseProfilingRepos {
		t.Errorf("Expected phase '%s', got '%s'", models.PhaseProfilingRepos, updated.Phase)
	}

	// Update to cancelling phase
	if err := db.UpdateDiscoveryPhase(progress.ID, models.PhaseCancelling); err != nil {
		t.Fatalf("Failed to update phase to cancelling: %v", err)
	}

	updated2, err := db.GetLatestDiscoveryProgress()
	if err != nil {
		t.Fatalf("Failed to get updated discovery: %v", err)
	}
	if updated2.Phase != models.PhaseCancelling {
		t.Errorf("Expected phase '%s', got '%s'", models.PhaseCancelling, updated2.Phase)
	}
}
