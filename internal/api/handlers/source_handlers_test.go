package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func createTestSourceModel(name, sourceType string) *models.Source {
	org := "test-org"
	source := &models.Source{
		Name:     name,
		Type:     sourceType,
		BaseURL:  "https://api.github.com",
		Token:    "ghp_test_token_12345678901234567890",
		IsActive: true,
	}
	if sourceType == models.SourceConfigTypeAzureDevOps {
		source.BaseURL = "https://dev.azure.com/test-org"
		source.Organization = &org
	}
	return source
}

func TestListSources(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	// Create source handler
	sourceHandler := NewSourceHandler(db, h.logger)

	// Create test sources
	source1 := createTestSourceModel("Source Alpha", models.SourceConfigTypeGitHub)
	source2 := createTestSourceModel("Source Beta", models.SourceConfigTypeGitHub)
	_ = db.CreateSource(ctx, source1)
	_ = db.CreateSource(ctx, source2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	w := httptest.NewRecorder()

	sourceHandler.ListSources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var sources []*models.SourceResponse
	if err := json.NewDecoder(w.Body).Decode(&sources); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(sources))
	}
}

func TestListSourcesActiveOnly(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()

	sourceHandler := NewSourceHandler(db, h.logger)

	// Create active and inactive sources
	active := createTestSourceModel("Active Source", models.SourceConfigTypeGitHub)
	inactive := createTestSourceModel("Inactive Source", models.SourceConfigTypeGitHub)

	if err := db.CreateSource(ctx, active); err != nil {
		t.Fatalf("Failed to create active source: %v", err)
	}
	if err := db.CreateSource(ctx, inactive); err != nil {
		t.Fatalf("Failed to create inactive source: %v", err)
	}

	// Set inactive source to inactive using SetSourceActive (to bypass GORM default)
	if err := db.SetSourceActive(ctx, inactive.ID, false); err != nil {
		t.Fatalf("Failed to set source inactive: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources?active=true", nil)
	w := httptest.NewRecorder()

	sourceHandler.ListSources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var sources []*models.SourceResponse
	if err := json.NewDecoder(w.Body).Decode(&sources); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(sources) != 1 {
		t.Errorf("Expected 1 active source, got %d", len(sources))
	}
}

func TestCreateSource(t *testing.T) {
	h, db := setupTestHandler(t)
	sourceHandler := NewSourceHandler(db, h.logger)

	t.Run("valid github source", func(t *testing.T) {
		createReq := CreateSourceRequest{
			Name:    "Test GitHub Source",
			Type:    "github",
			BaseURL: "https://api.github.com",
			Token:   "ghp_test_token_value_12345678",
		}
		body, _ := json.Marshal(createReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
		w := httptest.NewRecorder()

		sourceHandler.CreateSource(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var created models.SourceResponse
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.Name != "Test GitHub Source" {
			t.Errorf("Expected name 'Test GitHub Source', got '%s'", created.Name)
		}
		if created.Type != "github" {
			t.Errorf("Expected type 'github', got '%s'", created.Type)
		}
	})

	t.Run("valid azuredevops source", func(t *testing.T) {
		createReq := CreateSourceRequest{
			Name:         "Test ADO Source",
			Type:         "azuredevops",
			BaseURL:      "https://dev.azure.com/my-org",
			Token:        "ado_pat_token_value_12345678",
			Organization: "my-org",
		}
		body, _ := json.Marshal(createReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
		w := httptest.NewRecorder()

		sourceHandler.CreateSource(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader([]byte("invalid")))
		w := httptest.NewRecorder()

		sourceHandler.CreateSource(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		createReq := CreateSourceRequest{
			Name: "Incomplete Source",
			// Missing type, base_url, token
		}
		body, _ := json.Marshal(createReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
		w := httptest.NewRecorder()

		sourceHandler.CreateSource(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		// Create first source
		createReq := CreateSourceRequest{
			Name:    "Duplicate Name Source",
			Type:    "github",
			BaseURL: "https://api.github.com",
			Token:   "ghp_test_token_value_12345678",
		}
		body, _ := json.Marshal(createReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
		w := httptest.NewRecorder()
		sourceHandler.CreateSource(w, req)

		// Try to create with same name
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
		w2 := httptest.NewRecorder()
		sourceHandler.CreateSource(w2, req2)

		if w2.Code != http.StatusConflict {
			t.Errorf("Expected status %d for duplicate, got %d", http.StatusConflict, w2.Code)
		}
	})
}

func TestGetSource(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	source := createTestSourceModel("Get Test Source", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	t.Run("existing source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.GetSource(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response models.SourceResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != "Get Test Source" {
			t.Errorf("Expected name 'Get Test Source', got '%s'", response.Name)
		}
	})

	t.Run("non-existent source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/999", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		sourceHandler.GetSource(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/invalid", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		sourceHandler.GetSource(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestUpdateSource(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	source := createTestSourceModel("Update Test Source", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	t.Run("update name", func(t *testing.T) {
		newName := "Updated Source Name"
		updateReq := UpdateSourceRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/sources/1", bytes.NewReader(body))
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.UpdateSource(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response models.SourceResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != "Updated Source Name" {
			t.Errorf("Expected name 'Updated Source Name', got '%s'", response.Name)
		}
	})

	t.Run("non-existent source", func(t *testing.T) {
		newName := "Updated"
		updateReq := UpdateSourceRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/sources/999", bytes.NewReader(body))
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		sourceHandler.UpdateSource(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestDeleteSource(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	t.Run("delete existing source", func(t *testing.T) {
		source := createTestSourceModel("Delete Test Source", models.SourceConfigTypeGitHub)
		if err := db.CreateSource(ctx, source); err != nil {
			t.Fatalf("Failed to create source: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sources/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.DeleteSource(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
		}
	})

	t.Run("delete non-existent source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sources/999", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		sourceHandler.DeleteSource(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestSetSourceActive(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	source := createTestSourceModel("Active Toggle Source", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	t.Run("deactivate source", func(t *testing.T) {
		reqBody := `{"is_active": false}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/1/set-active", bytes.NewReader([]byte(reqBody)))
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.SetSourceActive(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Verify source is inactive
		updated, _ := db.GetSource(ctx, 1)
		if updated.IsActive {
			t.Error("Expected source to be inactive")
		}
	})

	t.Run("reactivate source", func(t *testing.T) {
		reqBody := `{"is_active": true}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/1/set-active", bytes.NewReader([]byte(reqBody)))
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.SetSourceActive(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify source is active
		updated, _ := db.GetSource(ctx, 1)
		if !updated.IsActive {
			t.Error("Expected source to be active")
		}
	})
}

func TestSourceResponseMasksToken(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	source := createTestSourceModel("Token Mask Test", models.SourceConfigTypeGitHub)
	source.Token = "ghp_supersecrettoken12345678901234567890"
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	sourceHandler.GetSource(w, req)

	var response models.SourceResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Token should be masked
	if response.MaskedToken == "ghp_supersecrettoken12345678901234567890" {
		t.Error("Token should be masked in response")
	}
	if response.MaskedToken != "ghp_...7890" {
		t.Errorf("Expected masked token 'ghp_...7890', got '%s'", response.MaskedToken)
	}
}

func TestGetSourceDeletionPreviewHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	source := createTestSourceModel("Preview Handler Test", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	t.Run("existing source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/1/deletion-preview", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.GetSourceDeletionPreview(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var preview map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&preview); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response structure
		if _, ok := preview["source_id"]; !ok {
			t.Error("Expected source_id in response")
		}
		if _, ok := preview["source_name"]; !ok {
			t.Error("Expected source_name in response")
		}
		if _, ok := preview["repository_count"]; !ok {
			t.Error("Expected repository_count in response")
		}
		if _, ok := preview["total_affected_records"]; !ok {
			t.Error("Expected total_affected_records in response")
		}
	})

	t.Run("non-existent source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/999/deletion-preview", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		sourceHandler.GetSourceDeletionPreview(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sources/invalid/deletion-preview", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		sourceHandler.GetSourceDeletionPreview(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestDeleteSourceWithForce(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	t.Run("force delete with correct confirmation", func(t *testing.T) {
		source := createTestSourceModel("Force Delete Test", models.SourceConfigTypeGitHub)
		if err := db.CreateSource(ctx, source); err != nil {
			t.Fatalf("Failed to create source: %v", err)
		}

		// Create a repository for this source to test cascade
		repo := &models.Repository{
			FullName:  "org/force-delete-repo",
			Source:    "ghes",
			SourceURL: "https://github.com/org/force-delete-repo",
			SourceID:  &source.ID,
			Status:    "pending",
		}
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		// Standard delete should fail (has repos)
		reqFail := httptest.NewRequest(http.MethodDelete, "/api/v1/sources/1", nil)
		reqFail.SetPathValue("id", "1")
		wFail := httptest.NewRecorder()
		sourceHandler.DeleteSource(wFail, reqFail)
		if wFail.Code != http.StatusConflict {
			t.Errorf("Expected conflict for standard delete, got %d", wFail.Code)
		}

		// Force delete with correct confirmation should succeed
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sources/1?force=true&confirm=Force+Delete+Test", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.DeleteSource(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
		}

		// Verify source is deleted
		deleted, _ := db.GetSource(ctx, source.ID)
		if deleted != nil {
			t.Error("Expected source to be deleted")
		}

		// Verify repository is deleted
		deletedRepo, _ := db.GetRepository(ctx, "org/force-delete-repo")
		if deletedRepo != nil {
			t.Error("Expected repository to be cascade deleted")
		}
	})
}

func TestDeleteSourceForceRequiresConfirmation(t *testing.T) {
	h, db := setupTestHandler(t)
	ctx := context.Background()
	sourceHandler := NewSourceHandler(db, h.logger)

	source := createTestSourceModel("Confirm Required Test", models.SourceConfigTypeGitHub)
	if err := db.CreateSource(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	t.Run("force delete without confirmation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sources/1?force=true", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.DeleteSource(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status %d, got %d: %s", http.StatusUnprocessableEntity, w.Code, w.Body.String())
		}

		// Verify source still exists
		existing, _ := db.GetSource(ctx, source.ID)
		if existing == nil {
			t.Error("Source should not be deleted without confirmation")
		}
	})

	t.Run("force delete with wrong confirmation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/sources/1?force=true&confirm=Wrong+Name", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		sourceHandler.DeleteSource(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status %d, got %d: %s", http.StatusUnprocessableEntity, w.Code, w.Body.String())
		}

		// Verify source still exists
		existing, _ := db.GetSource(ctx, source.ID)
		if existing == nil {
			t.Error("Source should not be deleted with wrong confirmation")
		}
	})
}
