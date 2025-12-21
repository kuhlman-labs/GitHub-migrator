package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

func TestListOrganizationsHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test repositories in different orgs
	repos := []struct {
		fullName string
		status   string
	}{
		{"org1/repo1", string(models.StatusPending)},
		{"org1/repo2", string(models.StatusComplete)},
		{"org2/repo1", string(models.StatusPending)},
	}

	for _, r := range repos {
		repo := &models.Repository{
			FullName:     r.fullName,
			Source:       "ghes",
			SourceURL:    fmt.Sprintf("https://github.com/%s", r.fullName),
			Status:       r.status,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.SaveRepository(context.Background(), repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{}, nil, authConfig, "https://api.github.com", "github")

	req := httptest.NewRequest("GET", "/api/v1/organizations", nil)
	w := httptest.NewRecorder()

	h.ListOrganizations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []storage.OrganizationStats
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 organizations, got %d", len(response))
	}
}

func TestListOrganizationsWithMultipleSources(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test repositories in different orgs with different sources
	repos := []struct {
		fullName string
		source   string
		status   string
	}{
		{"org1/repo1", "github", string(models.StatusPending)},
		{"org1/repo2", "github", string(models.StatusComplete)},
		{"org2/repo1", "azure-devops", string(models.StatusPending)},
	}

	for _, r := range repos {
		repo := &models.Repository{
			FullName:     r.fullName,
			Source:       r.source,
			SourceURL:    fmt.Sprintf("https://github.com/%s", r.fullName),
			Status:       r.status,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.SaveRepository(context.Background(), repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{}, nil, authConfig, "https://api.github.com", "github")

	t.Run("returns all organizations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/organizations", nil)
		w := httptest.NewRecorder()

		h.ListOrganizations(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []storage.OrganizationStats
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should return all 2 organizations (org1 and org2)
		if len(response) != 2 {
			t.Errorf("Expected 2 organizations, got %d", len(response))
		}
	})
}

func TestGetOrganizationListHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test repositories
	repos := []struct {
		fullName string
		status   string
	}{
		{"org1/repo1", string(models.StatusPending)},
		{"org2/repo1", string(models.StatusComplete)},
		{"org3/repo1", string(models.StatusMigrationFailed)},
	}

	for _, r := range repos {
		repo := &models.Repository{
			FullName:     r.fullName,
			Source:       "ghes",
			SourceURL:    fmt.Sprintf("https://github.com/%s", r.fullName),
			Status:       r.status,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := db.SaveRepository(context.Background(), repo); err != nil {
			t.Fatalf("Failed to save repository: %v", err)
		}
	}

	authConfig := &config.AuthConfig{Enabled: false}
	h := NewHandler(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, &mockSourceProvider{}, nil, authConfig, "https://api.github.com", "github")

	req := httptest.NewRequest("GET", "/api/v1/organizations/list", nil)
	w := httptest.NewRecorder()

	h.GetOrganizationList(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 3 {
		t.Errorf("Expected 3 organizations, got %d", len(response))
	}
}
