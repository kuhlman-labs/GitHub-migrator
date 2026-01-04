package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// SourceStore interface for source lookups (to avoid tight coupling)
type SourceStore interface {
	GetSource(ctx context.Context, id int64) (*models.Source, error)
	ListSources(ctx context.Context) ([]*models.Source, error)
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	oauthHandler *auth.OAuthHandler // Fallback for destination-based auth
	jwtManager   *auth.JWTManager
	authorizer   *auth.Authorizer
	logger       *slog.Logger
	config       *config.AuthConfig
	sourceStore  SourceStore // For source-scoped auth
	callbackURL  string      // OAuth callback URL
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(cfg *config.AuthConfig, logger *slog.Logger, githubBaseURL string) (*AuthHandler, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	jwtManager, err := auth.NewJWTManager(cfg.SessionSecret, cfg.SessionDurationHours)
	if err != nil {
		return nil, err
	}

	return &AuthHandler{
		oauthHandler: auth.NewOAuthHandler(cfg, logger, githubBaseURL),
		jwtManager:   jwtManager,
		authorizer:   auth.NewAuthorizer(cfg, logger, githubBaseURL),
		logger:       logger,
		config:       cfg,
		callbackURL:  cfg.CallbackURL,
	}, nil
}

// SetSourceStore sets the source store for source-scoped authentication
func (h *AuthHandler) SetSourceStore(store SourceStore) {
	h.sourceStore = store
}

// HandleLogin initiates OAuth login flow
// Uses destination-centric authentication (GitHub OAuth only)
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Use destination OAuth only - source-specific OAuth has been removed
	h.oauthHandler.HandleLogin(w, r)
}

// HandleCallback handles OAuth callback
// Uses destination-centric authentication (GitHub OAuth only)
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Use destination OAuth callback - validates state, exchanges code, gets user info
	h.oauthHandler.HandleCallback(w, r)

	// Get user and token from context (set by oauthHandler)
	user, ok := auth.GetGitHubUserFromContext(r.Context())
	if !ok {
		// oauthHandler already sent error response
		return
	}

	token, ok := auth.GetTokenFromContext(r.Context())
	if !ok {
		h.logger.Error("Missing token in context after OAuth callback")
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	h.logger.Info("OAuth callback successful", "user", user.Login)

	// Perform authorization check using destination-centric auth
	authResult, err := h.authorizer.Authorize(r.Context(), user, token)
	if err != nil {
		h.logger.Error("Authorization check failed", "user", user.Login, "error", err)
		// Don't fail login, but log the error - user gets minimal permissions
	}

	if authResult != nil && !authResult.Authorized {
		h.logger.Warn("User not authorized", "user", user.Login, "reason", authResult.Reason)
		// Redirect to frontend with error
		frontendURL := h.config.FrontendURL
		if frontendURL == "" {
			frontendURL = "http://localhost:3000"
		}
		http.Redirect(w, r, frontendURL+"/login?error=unauthorized", http.StatusFound)
		return
	}

	// Generate JWT token (includes encrypted OAuth token for API calls)
	jwtToken, err := h.jwtManager.GenerateToken(user, token)
	if err != nil {
		h.logger.Error("Failed to generate JWT token", "user", user.Login, "error", err)
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	// Set auth_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		MaxAge:   h.config.SessionDurationHours * 3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	h.logger.Info("User authenticated successfully", "user", user.Login)

	// Redirect to frontend
	frontendURL := h.config.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	http.Redirect(w, r, frontendURL, http.StatusFound)
}

// HandleLogout logs out the user
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get user from context for logging
	user, _ := auth.GetUserFromContext(r.Context())
	if user != nil {
		h.logger.Info("User logged out", "user", user.Login)
	}

	// Clear auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	}); err != nil {
		h.logger.Error("Failed to encode logout response", "error", err)
	}
}

// HandleCurrentUser returns current authenticated user info
func (h *AuthHandler) HandleCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	claims, _ := auth.GetClaimsFromContext(r.Context())

	response := map[string]any{
		"id":         user.ID,
		"login":      user.Login,
		"name":       user.Name,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
	}

	if claims != nil {
		if len(claims.Roles) > 0 {
			response["roles"] = claims.Roles
		}
		// Include source info if present
		if claims.SourceID != nil {
			response["source_id"] = *claims.SourceID
			response["source_type"] = claims.SourceType
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode current user response", "error", err)
	}
}

// HandleRefreshToken refreshes the JWT token
func (h *AuthHandler) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Generate new token with extended expiration
	newToken, err := h.jwtManager.RefreshToken(claims)
	if err != nil {
		h.logger.Error("Failed to refresh token", "error", err)
		http.Error(w, "Failed to refresh token", http.StatusInternalServerError)
		return
	}

	// Set new cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    newToken,
		Path:     "/",
		MaxAge:   h.config.SessionDurationHours * 3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	h.logger.Info("Token refreshed", "user", claims.Login)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Token refreshed successfully",
	}); err != nil {
		h.logger.Error("Failed to encode refresh token response", "error", err)
	}
}

// HandleAuthConfig returns auth configuration (for frontend)
func (h *AuthHandler) HandleAuthConfig(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"enabled": h.config.Enabled,
	}

	if h.config.Enabled {
		response["login_url"] = "/api/v1/auth/login"

		// Include authorization requirements (sanitized)
		rules := make(map[string]any)
		if len(h.config.AuthorizationRules.RequireOrgMembership) > 0 {
			rules["requires_org_membership"] = true
			rules["required_orgs"] = h.config.AuthorizationRules.RequireOrgMembership
		}
		if len(h.config.AuthorizationRules.RequireTeamMembership) > 0 {
			rules["requires_team_membership"] = true
			rules["required_teams"] = h.config.AuthorizationRules.RequireTeamMembership
		}
		if h.config.AuthorizationRules.RequireEnterpriseAdmin {
			rules["requires_enterprise_admin"] = true
			rules["enterprise"] = h.config.AuthorizationRules.RequireEnterpriseSlug
		}
		if h.config.AuthorizationRules.RequireEnterpriseMembership && h.config.AuthorizationRules.RequireEnterpriseSlug != "" {
			rules["requires_enterprise_membership"] = true
			if !h.config.AuthorizationRules.RequireEnterpriseAdmin {
				rules["enterprise"] = h.config.AuthorizationRules.RequireEnterpriseSlug
			}
		}
		response["authorization_rules"] = rules
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode auth config response", "error", err)
	}
}

// HandleAuthSources returns list of sources with OAuth configured (for login page)
// Note: Source-specific OAuth has been removed. This endpoint now always returns an empty array.
// It is kept for backward compatibility with the frontend.
func (h *AuthHandler) HandleAuthSources(w http.ResponseWriter, r *http.Request) {
	// Source-specific OAuth has been removed in favor of destination-centric auth
	// Always return an empty array for backward compatibility
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode([]any{}); err != nil {
		h.logger.Error("Failed to encode empty sources response", "error", err)
	}
}

// HandleAuthorizationStatus returns the current user's authorization tier and details
// This endpoint allows users to check their authorization level and what actions they can perform
func (h *AuthHandler) HandleAuthorizationStatus(w http.ResponseWriter, r *http.Request, handlerUtils *HandlerUtils) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := handlerUtils.GetUserAuthorizationStatus(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authorization status", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("Failed to encode authorization status response", "error", err)
	}
}
