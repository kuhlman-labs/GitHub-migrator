package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

// Middleware handles authentication middleware
type Middleware struct {
	jwtManager      *JWTManager
	authorizer      *Authorizer
	logger          *slog.Logger
	enabled         bool        // Static enabled flag (fallback)
	authEnabledFunc func() bool // Dynamic check for auth enabled state (optional)
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(jwtManager *JWTManager, authorizer *Authorizer, logger *slog.Logger, enabled bool) *Middleware {
	return &Middleware{
		jwtManager: jwtManager,
		authorizer: authorizer,
		logger:     logger,
		enabled:    enabled,
	}
}

// SetAuthEnabledFunc sets a callback to dynamically check if auth is enabled.
// This allows the middleware to respect runtime config changes from the database.
func (m *Middleware) SetAuthEnabledFunc(fn func() bool) {
	m.authEnabledFunc = fn
}

// isAuthEnabled returns whether authentication is currently enabled.
// Uses the dynamic callback if set, otherwise falls back to the static flag.
func (m *Middleware) isAuthEnabled() bool {
	if m.authEnabledFunc != nil {
		return m.authEnabledFunc()
	}
	return m.enabled
}

// Context keys for storing user information
type authContextKey string

const (
	ContextKeyUser        authContextKey = "auth_user"
	ContextKeyClaims      authContextKey = "auth_claims"
	ContextKeyGitHubToken authContextKey = "auth_github_token" // #nosec G101 -- This is a context key, not credentials
)

// RequireAuth is middleware that requires authentication
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled (check dynamically if callback is set), allow request through
		if !m.isAuthEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			m.logger.Debug("No auth token cookie found", "path", r.URL.Path)
			m.respondUnauthorized(w, "Authentication required")
			return
		}

		// Validate token
		claims, err := m.jwtManager.ValidateToken(cookie.Value)
		if err != nil {
			m.logger.Warn("Invalid token", "error", err, "path", r.URL.Path)
			m.respondUnauthorized(w, "Invalid or expired token")
			return
		}

		// Create user object from claims
		user := &GitHubUser{
			ID:        claims.UserID,
			Login:     claims.Login,
			Name:      claims.Name,
			Email:     claims.Email,
			AvatarURL: claims.AvatarURL,
		}

		// Decrypt GitHub token from claims
		githubToken, err := m.jwtManager.DecryptToken(claims.GitHubToken)
		if err != nil {
			m.logger.Warn("Failed to decrypt GitHub token", "error", err, "path", r.URL.Path)
			m.respondUnauthorized(w, "Invalid token")
			return
		}

		// Store user, claims, and GitHub token in context
		ctx := context.WithValue(r.Context(), ContextKeyUser, user)
		ctx = context.WithValue(ctx, ContextKeyClaims, claims)
		ctx = context.WithValue(ctx, ContextKeyGitHubToken, githubToken)

		m.logger.Debug("User authenticated", "login", user.Login, "path", r.URL.Path)

		// Pass request with context to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole is middleware that requires a specific role (for future RBAC)
func (m *Middleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If auth is disabled (check dynamically if callback is set), allow request through
			if !m.isAuthEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			// Get claims from context
			claims, ok := GetClaimsFromContext(r.Context())
			if !ok {
				m.respondUnauthorized(w, "Authentication required")
				return
			}

			// Check if user has required role
			hasRole := slices.Contains(claims.Roles, role)

			if !hasRole {
				m.respondForbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin is middleware that requires Tier 1 (Admin) authorization
// This is used to protect sensitive endpoints like settings and source management
func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled (check dynamically if callback is set), allow request through
		if !m.isAuthEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Get user from context (set by RequireAuth)
		user, hasUser := GetUserFromContext(r.Context())
		if !hasUser {
			m.respondUnauthorized(w, "Authentication required")
			return
		}

		// Get token from context
		token, hasToken := GetTokenFromContext(r.Context())
		if !hasToken {
			m.respondUnauthorized(w, "Authentication required")
			return
		}

		// Check if user has admin tier
		tierInfo, err := m.authorizer.GetUserAuthorizationTier(r.Context(), user, token)
		if err != nil {
			m.logger.Error("Failed to check user authorization tier", "user", user.Login, "error", err)
			m.respondForbidden(w, "Failed to verify authorization")
			return
		}

		if tierInfo.Tier != TierAdmin {
			m.logger.Warn("Non-admin user attempted to access admin endpoint",
				"user", user.Login,
				"tier", tierInfo.Tier,
				"path", r.URL.Path)
			m.respondForbidden(w, "Administrator access required. Only Tier 1 administrators can access this resource.")
			return
		}

		m.logger.Debug("Admin access granted", "user", user.Login, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// respondUnauthorized sends a 401 Unauthorized response
func (m *Middleware) respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		m.logger.Error("Failed to encode unauthorized response", "error", err)
	}
}

// respondForbidden sends a 403 Forbidden response
func (m *Middleware) respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		m.logger.Error("Failed to encode forbidden response", "error", err)
	}
}

// GetUserFromContext retrieves the authenticated user from request context
func GetUserFromContext(ctx context.Context) (*GitHubUser, bool) {
	user, ok := ctx.Value(ContextKeyUser).(*GitHubUser)
	return user, ok
}

// GetClaimsFromContext retrieves the JWT claims from request context
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ContextKeyClaims).(*Claims)
	return claims, ok
}

// GetSourceIDFromContext retrieves the authenticated source ID from context
// Returns nil if user authenticated via destination OAuth (not source-scoped)
func GetSourceIDFromContext(ctx context.Context) *int64 {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return nil
	}
	return claims.SourceID
}

// GetSourceTypeFromContext retrieves the authenticated source type from context
func GetSourceTypeFromContext(ctx context.Context) string {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return ""
	}
	return claims.SourceType
}

// ExtractBearerToken extracts a bearer token from the Authorization header
func ExtractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}
