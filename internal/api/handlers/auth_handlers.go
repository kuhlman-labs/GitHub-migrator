package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/brettkuhlman/github-migrator/internal/auth"
	"github.com/brettkuhlman/github-migrator/internal/config"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	oauthHandler *auth.OAuthHandler
	jwtManager   *auth.JWTManager
	authorizer   *auth.Authorizer
	logger       *slog.Logger
	config       *config.AuthConfig
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
	}, nil
}

// HandleLogin initiates OAuth login flow
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	h.oauthHandler.HandleLogin(w, r)
}

// HandleCallback handles OAuth callback
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Process OAuth callback
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

	// Check authorization
	authResult, err := h.authorizer.Authorize(r.Context(), user, githubToken)
	if err != nil {
		h.logger.Error("Authorization check failed", "error", err, "user", user.Login)
		http.Error(w, "Authorization check failed", http.StatusInternalServerError)
		return
	}

	if !authResult.Authorized {
		h.logger.Warn("User not authorized", "user", user.Login, "reason", authResult.Reason)

		// Return HTML page with error message
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
        <p>` + authResult.Reason + `</p>
        <p>Please contact your administrator if you believe you should have access.</p>
        <a href="/">Return to Home</a>
    </div>
</body>
</html>
		`
		if _, err := w.Write([]byte(htmlContent)); err != nil {
			h.logger.Error("Failed to write access denied response", "error", err)
		}
		return
	}

	// Generate JWT token
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
		redirectURL = "/" // Fallback to root if not configured
	}
	h.logger.Debug("Redirecting after login", "url", redirectURL, "user", user.Login)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// HandleLogout logs out the user
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get user from context for logging
	user, _ := auth.GetUserFromContext(r.Context())
	if user != nil {
		h.logger.Info("User logged out", "user", user.Login)
	}

	// Clear auth cookie - must match all attributes from when cookie was set
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

	response := map[string]interface{}{
		"id":         user.ID,
		"login":      user.Login,
		"name":       user.Name,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
	}

	if claims != nil && len(claims.Roles) > 0 {
		response["roles"] = claims.Roles
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
	response := map[string]interface{}{
		"enabled": h.config.Enabled,
	}

	if h.config.Enabled {
		response["login_url"] = "/api/v1/auth/login"

		// Include authorization requirements (sanitized)
		rules := make(map[string]interface{})
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
		response["authorization_rules"] = rules
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode auth config response", "error", err)
	}
}
