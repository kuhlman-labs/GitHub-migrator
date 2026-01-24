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

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// Helper to create authenticated context
func withAuthUser(ctx context.Context, user *auth.GitHubUser, token string) context.Context {
	ctx = context.WithValue(ctx, auth.ContextKeyUser, user)
	ctx = context.WithValue(ctx, auth.ContextKeyGitHubToken, token)
	return ctx
}

func TestCopilotHandler_GetStatus_Unauthenticated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/copilot/status", nil)
	rec := httptest.NewRecorder()

	handler.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var status models.CopilotStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if status.Enabled {
		t.Error("expected Enabled to be false for unauthenticated user")
	}
	if status.Available {
		t.Error("expected Available to be false for unauthenticated user")
	}
	if status.UnavailableReason != "Authentication required" {
		t.Errorf("unexpected reason: %s", status.UnavailableReason)
	}
}

func TestCopilotHandler_SendMessage_Unauthenticated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	body := bytes.NewBufferString(`{"message": "Hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SendMessage(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCopilotHandler_SendMessage_EmptyMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	// Create authenticated context
	user := &auth.GitHubUser{
		ID:    123,
		Login: "testuser",
	}
	ctx := withAuthUser(context.Background(), user, "test-token")

	body := bytes.NewBufferString(`{"message": "   "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", body).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SendMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCopilotHandler_SendMessage_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	// Create authenticated context
	user := &auth.GitHubUser{
		ID:    123,
		Login: "testuser",
	}
	ctx := withAuthUser(context.Background(), user, "test-token")

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", body).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SendMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCopilotHandler_ListSessions_Unauthenticated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/copilot/sessions", nil)
	rec := httptest.NewRecorder()

	handler.ListSessions(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCopilotHandler_DeleteSession_NoSessionID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	// Create authenticated context
	user := &auth.GitHubUser{
		ID:    123,
		Login: "testuser",
	}
	ctx := withAuthUser(context.Background(), user, "test-token")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/copilot/sessions/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.DeleteSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCopilotHandler_ValidateCLI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	body := bytes.NewBufferString(`{"cli_path": "/usr/local/bin/copilot"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/validate-cli", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ValidateCLI(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// CLI won't be available in test environment, so check for error
	if _, ok := response["available"]; !ok {
		t.Error("expected 'available' field in response")
	}
}

func TestCopilotHandler_ValidateCLI_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewCopilotHandler(nil, logger, "https://api.github.com")

	body := bytes.NewBufferString(`{invalid}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/validate-cli", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ValidateCLI(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
