package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/config"
)

func TestNewOAuthHandler(t *testing.T) {
	cfg := &config.AuthConfig{
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
		CallbackURL:             "http://localhost:8080/auth/callback",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name      string
		baseURL   string
		expectNil bool
	}{
		{
			name:      "github.com",
			baseURL:   "https://api.github.com",
			expectNil: false,
		},
		{
			name:      "github enterprise",
			baseURL:   "https://github.example.com/api/v3",
			expectNil: false,
		},
		{
			name:      "empty base URL",
			baseURL:   "",
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewOAuthHandler(cfg, logger, tt.baseURL)
			if tt.expectNil && handler != nil {
				t.Error("Expected nil handler")
			}
			if !tt.expectNil && handler == nil {
				t.Error("Expected non-nil handler")
			}
		})
	}
}

func TestHandleLogin(t *testing.T) {
	cfg := &config.AuthConfig{
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
		CallbackURL:             "http://localhost:8080/auth/callback",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewOAuthHandler(cfg, logger, "https://api.github.com")

	req := httptest.NewRequest("GET", "/auth/login", nil)
	w := httptest.NewRecorder()

	handler.HandleLogin(w, req)

	// Should redirect to GitHub
	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}

	// Should have Location header
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header")
	}

	// Location should contain GitHub OAuth URL
	if !strings.Contains(location, "github.com") && !strings.Contains(location, "client_id") {
		t.Errorf("Location doesn't look like GitHub OAuth URL: %s", location)
	}

	// Should set state cookie
	cookies := w.Result().Cookies()
	hasStateCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "oauth_state" {
			hasStateCookie = true
			if cookie.Value == "" {
				t.Error("State cookie value is empty")
			}
			if !cookie.HttpOnly {
				t.Error("State cookie should be HttpOnly")
			}
		}
	}
	if !hasStateCookie {
		t.Error("Expected oauth_state cookie")
	}
}

func TestHandleCallbackMissingState(t *testing.T) {
	cfg := &config.AuthConfig{
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
		CallbackURL:             "http://localhost:8080/auth/callback",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewOAuthHandler(cfg, logger, "https://api.github.com")

	// Request without state cookie
	req := httptest.NewRequest("GET", "/auth/callback?state=test-state&code=test-code", nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	// Should return bad request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleCallbackStateMismatch(t *testing.T) {
	cfg := &config.AuthConfig{
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
		CallbackURL:             "http://localhost:8080/auth/callback",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewOAuthHandler(cfg, logger, "https://api.github.com")

	// Request with mismatched state
	req := httptest.NewRequest("GET", "/auth/callback?state=different-state&code=test-code", nil)
	req.AddCookie(&http.Cookie{
		Name:  "oauth_state",
		Value: "original-state",
	})
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	// Should return bad request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleCallbackMissingCode(t *testing.T) {
	cfg := &config.AuthConfig{
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
		CallbackURL:             "http://localhost:8080/auth/callback",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewOAuthHandler(cfg, logger, "https://api.github.com")

	// Request without code parameter
	req := httptest.NewRequest("GET", "/auth/callback?state=test-state", nil)
	req.AddCookie(&http.Cookie{
		Name:  "oauth_state",
		Value: "test-state",
	})
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	// Should return bad request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetGitHubUserFromContext(t *testing.T) {
	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
		Name:  "Test User",
	}

	// Test with user in context
	ctx := context.WithValue(context.Background(), contextKeyGitHubUser, user)
	gotUser, ok := GetGitHubUserFromContext(ctx)
	if !ok {
		t.Error("Expected user in context")
	}
	if gotUser.ID != user.ID {
		t.Errorf("User ID mismatch: got %d, want %d", gotUser.ID, user.ID)
	}

	// Test without user in context
	emptyCtx := context.Background()
	_, ok = GetGitHubUserFromContext(emptyCtx)
	if ok {
		t.Error("Did not expect user in empty context")
	}
}

func TestGetTokenFromContext(t *testing.T) {
	token := "test-token-12345"

	// Test with token in context
	ctx := context.WithValue(context.Background(), contextKeyGitHubToken, token)
	gotToken, ok := GetTokenFromContext(ctx)
	if !ok {
		t.Error("Expected token in context")
	}
	if gotToken != token {
		t.Errorf("Token mismatch: got %s, want %s", gotToken, token)
	}

	// Test without token in context
	emptyCtx := context.Background()
	_, ok = GetTokenFromContext(emptyCtx)
	if ok {
		t.Error("Did not expect token in empty context")
	}
}

func TestBuildOAuthURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "github.com API URL",
			baseURL:  "https://api.github.com",
			expected: "https://github.com",
		},
		{
			name:     "github with data residency",
			baseURL:  "https://api.company.ghe.com",
			expected: "https://company.ghe.com",
		},
		{
			name:     "github enterprise",
			baseURL:  "https://github.example.com/api/v3",
			expected: "https://github.example.com",
		},
		{
			name:     "github enterprise with api prefix",
			baseURL:  "https://api.github.example.com",
			expected: "https://github.example.com",
		},
		{
			name:     "invalid URL",
			baseURL:  "://invalid",
			expected: "https://github.com",
		},
		{
			name:     "empty URL",
			baseURL:  "",
			expected: "https://github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOAuthURL(tt.baseURL)
			if got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestGenerateStateToken(t *testing.T) {
	// Generate multiple tokens
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := generateStateToken()
		if err != nil {
			t.Fatalf("Failed to generate state token: %v", err)
		}
		if token == "" {
			t.Error("Generated empty state token")
		}
		if tokens[token] {
			t.Error("Generated duplicate state token")
		}
		tokens[token] = true
	}
}

func TestOAuthHandlerWithGHES(t *testing.T) {
	cfg := &config.AuthConfig{
		GitHubOAuthClientID:     "test-client-id",
		GitHubOAuthClientSecret: "test-client-secret",
		CallbackURL:             "http://localhost:8080/auth/callback",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with GHES URL
	ghesURL := "https://github.example.com/api/v3"
	handler := NewOAuthHandler(cfg, logger, ghesURL)

	req := httptest.NewRequest("GET", "/auth/login", nil)
	w := httptest.NewRecorder()

	handler.HandleLogin(w, req)

	// Should redirect
	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}

	// Location should contain GHES domain
	location := w.Header().Get("Location")
	parsedURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Failed to parse location URL: %v", err)
	}

	// For GHES, the OAuth URL should use the base domain
	if !strings.Contains(parsedURL.Host, "github.example.com") {
		t.Errorf("Expected GHES domain in OAuth URL, got: %s", location)
	}
}
