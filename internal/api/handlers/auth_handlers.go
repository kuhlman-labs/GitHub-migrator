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
// Accepts optional query param: ?source_id=123
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	sourceID, err := auth.GetSourceIDFromRequest(r)
	if err != nil {
		h.logger.Error("Invalid source_id", "error", err)
		http.Error(w, "Invalid source_id parameter", http.StatusBadRequest)
		return
	}

	// If source_id is specified, use source-scoped OAuth
	if sourceID > 0 && h.sourceStore != nil {
		h.handleSourceLogin(w, r, sourceID)
		return
	}

	// Fall back to destination OAuth (existing behavior)
	h.oauthHandler.HandleLogin(w, r)
}

// handleSourceLogin initiates OAuth flow for a specific source
func (h *AuthHandler) handleSourceLogin(w http.ResponseWriter, r *http.Request, sourceID int64) {
	ctx := r.Context()

	// Get source from database
	source, err := h.sourceStore.GetSource(ctx, sourceID)
	if err != nil {
		h.logger.Error("Failed to get source", "error", err, "source_id", sourceID)
		http.Error(w, "Failed to get source", http.StatusInternalServerError)
		return
	}
	if source == nil {
		http.Error(w, "Source not found", http.StatusNotFound)
		return
	}

	// Verify source has OAuth configured
	if !source.HasOAuth() {
		h.logger.Warn("Source does not have OAuth configured", "source_id", sourceID, "source_name", source.Name)
		http.Error(w, "Source does not have OAuth configured", http.StatusBadRequest)
		return
	}

	// Create source OAuth handler
	sourceOAuth, err := auth.NewSourceOAuthHandler(source, h.callbackURL, h.logger)
	if err != nil {
		h.logger.Error("Failed to create source OAuth handler", "error", err, "source_id", sourceID)
		http.Error(w, "Failed to initialize OAuth", http.StatusInternalServerError)
		return
	}

	// Generate state with source ID encoded
	state, err := auth.EncodeSourceState(sourceID)
	if err != nil {
		h.logger.Error("Failed to generate state", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state in cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to OAuth provider
	authURL := sourceOAuth.GetAuthURL(state)
	h.logger.Debug("Redirecting to source OAuth", "source_id", sourceID, "source_name", source.Name)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles OAuth callback
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Get state from cookie
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		h.logger.Error("Missing state cookie", "error", err)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Verify state matches
	state := r.URL.Query().Get("state")
	if state == "" || state != stateCookie.Value {
		h.logger.Error("State mismatch", "expected", stateCookie.Value, "got", state)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Try to decode source state (new format)
	sourceState, err := auth.DecodeSourceState(state)
	if err == nil && sourceState.SourceID > 0 {
		// Source-scoped callback
		h.handleSourceCallback(w, r, sourceState.SourceID)
		return
	}

	// Fall back to destination OAuth callback (legacy format)
	h.handleDestinationCallback(w, r)
}

// handleSourceCallback handles OAuth callback for source-scoped auth
func (h *AuthHandler) handleSourceCallback(w http.ResponseWriter, r *http.Request, sourceID int64) {
	ctx := r.Context()

	// Get source from database
	source, err := h.sourceStore.GetSource(ctx, sourceID)
	if err != nil {
		h.logger.Error("Failed to get source for callback", "error", err, "source_id", sourceID)
		http.Error(w, "Failed to get source", http.StatusInternalServerError)
		return
	}
	if source == nil {
		h.logger.Error("Source not found for callback", "source_id", sourceID)
		http.Error(w, "Source not found", http.StatusBadRequest)
		return
	}

	// Create source OAuth handler
	sourceOAuth, err := auth.NewSourceOAuthHandler(source, h.callbackURL, h.logger)
	if err != nil {
		h.logger.Error("Failed to create source OAuth handler", "error", err)
		http.Error(w, "Failed to process callback", http.StatusInternalServerError)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		h.logger.Error("Missing authorization code")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := sourceOAuth.ExchangeCode(ctx, code)
	if err != nil {
		h.logger.Error("Failed to exchange code", "error", err)
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Get user info
	user, err := sourceOAuth.GetUser(ctx, token.AccessToken)
	if err != nil {
		h.logger.Error("Failed to get user info", "error", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	h.logger.Info("User authenticated via source", "login", user.Login, "source_id", sourceID, "source_name", source.Name)

	// For source-scoped auth, we don't use the global authorization rules
	// The user just needs to have authenticated with the source
	// Permission checks happen at the repository level

	// Generate JWT with source info
	sourceInfo := &auth.SourceInfo{
		SourceID:   sourceID,
		SourceType: source.Type,
	}
	jwtToken, err := h.jwtManager.GenerateTokenWithSource(user, token.AccessToken, sourceInfo)
	if err != nil {
		h.logger.Error("Failed to generate token", "error", err, "user", user.Login)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set cookie with JWT
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		MaxAge:   h.config.SessionDurationHours * 3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	h.logger.Info("User logged in via source", "user", user.Login, "source_id", sourceID)

	// Redirect to frontend
	redirectURL := h.config.FrontendURL
	if redirectURL == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// handleDestinationCallback handles OAuth callback for destination-based auth (legacy)
func (h *AuthHandler) handleDestinationCallback(w http.ResponseWriter, r *http.Request) {
	// Process OAuth callback using the legacy handler
	h.oauthHandler.HandleCallback(w, r)

	// Get user and token from context
	user, ok := auth.GetGitHubUserFromContext(r.Context())
	if !ok {
		h.logger.Error("Failed to get user from context")
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	githubToken, ok := auth.GetTokenFromContext(r.Context())
	if !ok {
		h.logger.Error("Failed to get token from context")
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Check authorization using global rules
	authResult, err := h.authorizer.Authorize(r.Context(), user, githubToken)
	if err != nil {
		h.logger.Error("Authorization check failed", "error", err, "user", user.Login)
		http.Error(w, "Authorization check failed", http.StatusInternalServerError)
		return
	}

	if !authResult.Authorized {
		h.logger.Warn("User not authorized", "user", user.Login, "reason", authResult.Reason)
		h.renderAccessDenied(w, authResult.Reason)
		return
	}

	// Generate JWT token (no source info for destination auth)
	token, err := h.jwtManager.GenerateToken(user, githubToken)
	if err != nil {
		h.logger.Error("Failed to generate token", "error", err, "user", user.Login)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set cookie with JWT
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   h.config.SessionDurationHours * 3600,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	h.logger.Info("User logged in successfully", "user", user.Login)

	// Redirect to frontend URL
	redirectURL := h.config.FrontendURL
	if redirectURL == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// renderAccessDenied renders an access denied HTML page
func (h *AuthHandler) renderAccessDenied(w http.ResponseWriter, reason string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusForbidden)
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <title>Access Denied</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background-color: #0d1117;
            color: #c9d1d9;
        }
        .container {
            text-align: center;
            max-width: 500px;
            padding: 2rem;
        }
        h1 {
            color: #f85149;
            margin-bottom: 1rem;
        }
        p {
            margin-bottom: 1.5rem;
            line-height: 1.6;
        }
        a {
            color: #58a6ff;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Access Denied</h1>
        <p>` + reason + `</p>
        <p>Please contact your administrator if you believe you should have access.</p>
        <a href="/">Return to Home</a>
    </div>
</body>
</html>
	`
	if _, err := w.Write([]byte(htmlContent)); err != nil {
		h.logger.Error("Failed to write access denied response", "error", err)
	}
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
func (h *AuthHandler) HandleAuthSources(w http.ResponseWriter, r *http.Request) {
	if h.sourceStore == nil {
		// No source store configured, return empty list
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]any{}); err != nil {
			h.logger.Error("Failed to encode empty sources response", "error", err)
		}
		return
	}

	ctx := r.Context()
	sources, err := h.sourceStore.ListSources(ctx)
	if err != nil {
		h.logger.Error("Failed to list sources", "error", err)
		http.Error(w, "Failed to list sources", http.StatusInternalServerError)
		return
	}

	// Filter to only sources with OAuth configured
	var authSources []map[string]any
	for _, source := range sources {
		if source.HasOAuth() && source.IsActive {
			authSources = append(authSources, map[string]any{
				"id":   source.ID,
				"name": source.Name,
				"type": source.Type,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(authSources); err != nil {
		h.logger.Error("Failed to encode auth sources response", "error", err)
	}
}
