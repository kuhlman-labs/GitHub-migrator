package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestValidation tests request validation for various handlers
func TestRequestValidation(t *testing.T) {
	h, _ := setupTestHandler(t)

	t.Run("CreateBatch - empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", nil)
		w := httptest.NewRecorder()

		h.CreateBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateBatch - invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader([]byte("not json")))
		w := httptest.NewRecorder()

		h.CreateBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateBatch - missing required fields", func(t *testing.T) {
		batch := map[string]any{
			"description": "No name provided",
		}
		body, _ := json.Marshal(batch)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.CreateBatch(w, req)

		// Should fail because name is required
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateRepository - invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/repositories/org%2Frepo", bytes.NewReader([]byte("not json")))
		req.SetPathValue("name", "org%2Frepo")
		w := httptest.NewRecorder()

		h.UpdateRepository(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestPathParameterValidation tests path parameter validation
func TestPathParameterValidation(t *testing.T) {
	h, _ := setupTestHandler(t)

	t.Run("GetBatch - invalid ID format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/not-a-number", nil)
		req.SetPathValue("id", "not-a-number")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetBatch - empty ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/", nil)
		req.SetPathValue("id", "")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetBatch - negative ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/-1", nil)
		req.SetPathValue("id", "-1")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		// Should be not found (negative IDs don't exist)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteBatch - invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/batches/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		h.DeleteBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("StartBatch - invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/xyz/start", nil)
		req.SetPathValue("id", "xyz")
		w := httptest.NewRecorder()

		h.StartBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestQueryParameterValidation tests query parameter validation
func TestQueryParameterValidation(t *testing.T) {
	h, _ := setupTestHandler(t)

	t.Run("ListRepositories - valid pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories?limit=10&offset=0", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListRepositories - invalid limit (negative)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories?limit=-5", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		// Should handle gracefully (use default or return error)
		// Most handlers normalize negative values to defaults
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", w.Code)
		}
	})

	t.Run("ListRepositories - non-numeric limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories?limit=abc", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		// Should handle gracefully (use default or return error)
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", w.Code)
		}
	})

	t.Run("ListRepositories - filter by status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories?status=pending", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListRepositories - filter by organization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories?organization=my-org", nil)
		w := httptest.NewRecorder()

		h.ListRepositories(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// TestContentTypeValidation tests content type handling
func TestContentTypeValidation(t *testing.T) {
	h, _ := setupTestHandler(t)

	t.Run("CreateBatch - with content-type header", func(t *testing.T) {
		batch := map[string]any{
			"name": "Test Batch",
			"type": "pilot",
		}
		body, _ := json.Marshal(batch)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		h.CreateBatch(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		// Verify response content type
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
		}
	})
}

// TestErrorResponseFormat tests that error responses are properly formatted
func TestErrorResponseFormat(t *testing.T) {
	h, _ := setupTestHandler(t)

	t.Run("error response has error field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/batches/invalid", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		h.GetBatch(w, req)

		var response map[string]any
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if _, hasError := response["error"]; !hasError {
			t.Error("Error response should have 'error' field")
		}
	})
}

// TestAddRepositoriesToBatchValidation tests validation for adding repos to batch
func TestAddRepositoriesToBatchValidation(t *testing.T) {
	h, _ := setupTestHandler(t)

	t.Run("empty repository_ids", func(t *testing.T) {
		reqBody := map[string]any{
			"repository_ids": []int64{},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/1/repositories", bytes.NewReader(body))
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		h.AddRepositoriesToBatch(w, req)

		// Empty list or non-existent batch should return appropriate error
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 200, 400 or 404, got %d", w.Code)
		}
	})

	t.Run("invalid batch ID", func(t *testing.T) {
		reqBody := map[string]any{
			"repository_ids": []int64{1, 2, 3},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/abc/repositories", bytes.NewReader(body))
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		h.AddRepositoriesToBatch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("non-existent batch", func(t *testing.T) {
		reqBody := map[string]any{
			"repository_ids": []int64{1, 2, 3},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/batches/99999/repositories", bytes.NewReader(body))
		req.SetPathValue("id", "99999")
		w := httptest.NewRecorder()

		h.AddRepositoriesToBatch(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}
