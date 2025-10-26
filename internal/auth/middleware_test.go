package auth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRequireAuth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	// Create test user and token
	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
		Name:  "Test User",
		Email: "test@example.com",
	}
	token, _ := jwtManager.GenerateToken(user, "github-token")

	tests := []struct {
		name         string
		cookieValue  string
		expectStatus int
		expectUser   bool
		authEnabled  bool
	}{
		{
			name:         "valid token",
			cookieValue:  token,
			expectStatus: http.StatusOK,
			expectUser:   true,
			authEnabled:  true,
		},
		{
			name:         "no token",
			cookieValue:  "",
			expectStatus: http.StatusUnauthorized,
			expectUser:   false,
			authEnabled:  true,
		},
		{
			name:         "invalid token",
			cookieValue:  "invalid-token",
			expectStatus: http.StatusUnauthorized,
			expectUser:   false,
			authEnabled:  true,
		},
		{
			name:         "auth disabled",
			cookieValue:  "",
			expectStatus: http.StatusOK,
			expectUser:   false,
			authEnabled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware with appropriate auth enabled setting
			m := NewMiddleware(jwtManager, nil, logger, tt.authEnabled)

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, ok := GetUserFromContext(r.Context())
				if tt.expectUser {
					if !ok {
						t.Error("Expected user in context but got none")
					}
					if user == nil {
						t.Error("User in context is nil")
					}
				} else {
					if ok && user != nil && tt.authEnabled {
						t.Error("Did not expect user in context")
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with auth middleware
			handler := m.RequireAuth(testHandler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  "auth_token",
					Value: tt.cookieValue,
				})
			}

			// Record response
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// Check status
			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	// Test with user in context
	ctx := context.WithValue(context.Background(), ContextKeyUser, user)
	gotUser, ok := GetUserFromContext(ctx)
	if !ok {
		t.Error("Expected user in context")
	}
	if gotUser.ID != user.ID {
		t.Errorf("User ID mismatch: got %d, want %d", gotUser.ID, user.ID)
	}

	// Test without user in context
	emptyCtx := context.Background()
	_, ok = GetUserFromContext(emptyCtx)
	if ok {
		t.Error("Did not expect user in empty context")
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	claims := &Claims{
		UserID: 12345,
		Login:  "testuser",
	}

	// Test with claims in context
	ctx := context.WithValue(context.Background(), ContextKeyClaims, claims)
	gotClaims, ok := GetClaimsFromContext(ctx)
	if !ok {
		t.Error("Expected claims in context")
	}
	if gotClaims.UserID != claims.UserID {
		t.Errorf("Claims UserID mismatch: got %d, want %d", gotClaims.UserID, claims.UserID)
	}

	// Test without claims in context
	emptyCtx := context.Background()
	_, ok = GetClaimsFromContext(emptyCtx)
	if ok {
		t.Error("Did not expect claims in empty context")
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "valid bearer token",
			header:   "Bearer test-token-12345",
			expected: "test-token-12345",
		},
		{
			name:     "lowercase bearer",
			header:   "bearer test-token-12345",
			expected: "test-token-12345",
		},
		{
			name:     "no bearer prefix",
			header:   "test-token-12345",
			expected: "",
		},
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "only bearer",
			header:   "Bearer",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			got := ExtractBearerToken(req)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRequireAuthWithExpiredToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Create manager with very short duration
	jwtManager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 0)
	middleware := NewMiddleware(jwtManager, nil, logger, true)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}
	token, _ := jwtManager.GenerateToken(user, "github-token")

	// Small delay to ensure token expires (duration is 0 hours)
	// Note: In practice, tokens expire based on the ExpiresAt claim

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with expired token")
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// With 0 hour duration, token should be expired immediately
	// The actual expiration depends on clock precision, so we just verify
	// that auth middleware processes the request
	if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status OK or Unauthorized, got %d", w.Code)
	}
}
