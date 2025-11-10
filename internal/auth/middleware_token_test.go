package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMiddleware_TokenInjection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	// Create test user and original GitHub token
	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
		Name:  "Test User",
		Email: "test@example.com",
	}
	originalGitHubToken := "gh_test_token_12345"

	// Generate JWT with encrypted GitHub token
	jwtToken, err := jwtManager.GenerateToken(user, originalGitHubToken)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Create middleware (authorizer is nil for this test)
	m := NewMiddleware(jwtManager, nil, logger, true)

	// Create test handler that checks context
	var capturedUser *GitHubUser
	var capturedToken string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser, _ = GetUserFromContext(r.Context())
		capturedToken, _ = GetTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Wrap handler with middleware
	handler := m.RequireAuth(testHandler)

	// Create request with JWT cookie
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: jwtToken,
	})
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify status
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify user was injected
	if capturedUser == nil {
		t.Fatal("expected user to be injected into context")
	}
	if capturedUser.Login != user.Login {
		t.Errorf("expected user login %s, got %s", user.Login, capturedUser.Login)
	}

	// Verify GitHub token was injected and decrypted
	if capturedToken == "" {
		t.Fatal("expected GitHub token to be injected into context")
	}
	if capturedToken != originalGitHubToken {
		t.Errorf("expected GitHub token %s, got %s", originalGitHubToken, capturedToken)
	}
}

func TestMiddleware_TokenInjection_InvalidToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	m := NewMiddleware(jwtManager, nil, logger, true)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
		w.WriteHeader(http.StatusOK)
	})

	handler := m.RequireAuth(testHandler)

	// Create request with invalid JWT cookie
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "invalid-jwt-token",
	})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should be unauthorized
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestGetTokenFromContext_Extended(t *testing.T) {
	tests := []struct {
		name          string
		setupContext  func() context.Context
		expectedToken string
		expectFound   bool
	}{
		{
			name: "token from authenticated request context",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, ContextKeyGitHubToken, "test-token-123")
			},
			expectedToken: "test-token-123",
			expectFound:   true,
		},
		{
			name: "token from OAuth callback context",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, contextKeyGitHubToken, "oauth-token-456")
			},
			expectedToken: "oauth-token-456",
			expectFound:   true,
		},
		{
			name: "no token in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedToken: "",
			expectFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			token, found := GetTokenFromContext(ctx)

			if found != tt.expectFound {
				t.Errorf("expected found=%v, got=%v", tt.expectFound, found)
			}
			if token != tt.expectedToken {
				t.Errorf("expected token=%s, got=%s", tt.expectedToken, token)
			}
		})
	}
}
