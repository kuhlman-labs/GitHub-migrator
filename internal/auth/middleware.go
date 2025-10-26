package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// Middleware handles authentication middleware
type Middleware struct {
	jwtManager *JWTManager
	authorizer *Authorizer
	logger     *slog.Logger
	enabled    bool
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

// Context keys for storing user information
type authContextKey string

const (
	ContextKeyUser   authContextKey = "auth_user"
	ContextKeyClaims authContextKey = "auth_claims"
)

// RequireAuth is middleware that requires authentication
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, allow request through
		if !m.enabled {
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

		// Store user and claims in context
		ctx := context.WithValue(r.Context(), ContextKeyUser, user)
		ctx = context.WithValue(ctx, ContextKeyClaims, claims)

		m.logger.Debug("User authenticated", "login", user.Login, "path", r.URL.Path)

		// Pass request with context to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole is middleware that requires a specific role (for future RBAC)
func (m *Middleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If auth is disabled, allow request through
			if !m.enabled {
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
			hasRole := false
			for _, userRole := range claims.Roles {
				if userRole == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				m.respondForbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
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
