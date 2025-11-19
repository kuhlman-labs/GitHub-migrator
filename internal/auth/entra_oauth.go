package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// EntraIDOAuthHandler handles Entra ID OAuth flow for Azure DevOps
type EntraIDOAuthHandler struct {
	config      *config.AuthConfig
	oauthConfig *oauth2.Config
	adoOrgURL   string
	logger      *slog.Logger
}

// NewEntraIDOAuthHandler creates a new Entra ID OAuth handler
func NewEntraIDOAuthHandler(cfg *config.AuthConfig) *EntraIDOAuthHandler {
	// Build Entra ID OAuth configuration
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.EntraIDClientID,
		ClientSecret: cfg.EntraIDClientSecret,
		RedirectURL:  cfg.EntraIDCallbackURL,
		Scopes: []string{
			"499b84ac-1321-427f-aa17-267ca6975798/.default", // Azure DevOps default scope
		},
		Endpoint: microsoft.AzureADEndpoint(cfg.EntraIDTenantID),
	}

	return &EntraIDOAuthHandler{
		config:      cfg,
		oauthConfig: oauthConfig,
		adoOrgURL:   cfg.ADOOrganizationURL,
		logger:      slog.Default(),
	}
}

// GetAuthorizationURL returns the OAuth authorization URL
func (h *EntraIDOAuthHandler) GetAuthorizationURL(state string) string {
	return h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for an access token
func (h *EntraIDOAuthHandler) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := h.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// ADOUser represents an Azure DevOps user profile
type ADOUser struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	PublicAlias  string `json:"publicAlias"`
}

// GetUserProfile fetches the user profile from Azure DevOps
func (h *EntraIDOAuthHandler) GetUserProfile(ctx context.Context, accessToken string) (*ADOUser, error) {
	// Build profile API URL
	// Azure DevOps profile API: https://app.vssps.visualstudio.com/_apis/profile/profiles/me?api-version=6.0
	profileURL := "https://app.vssps.visualstudio.com/_apis/profile/profiles/me?api-version=6.0"

	req, err := http.NewRequestWithContext(ctx, "GET", profileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user profile: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var user ADOUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &user, nil
}

// CheckOrganizationMembership verifies if a user has access to the ADO organization
func (h *EntraIDOAuthHandler) CheckOrganizationMembership(ctx context.Context, userID string, accessToken string) (bool, error) {
	// Build API URL to check organization membership
	// https://dev.azure.com/{org}/_apis/userentitlements?api-version=6.0-preview.3
	apiURL := fmt.Sprintf("%s/_apis/userentitlements?api-version=6.0-preview.3&$filter=id eq '%s'", h.adoOrgURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check organization membership: %w", err)
	}
	defer resp.Body.Close()

	// If we get a 200, user has access to the organization
	// If we get a 401/403, user doesn't have access
	if resp.StatusCode == http.StatusOK {
		// Parse response to verify user is in the results
		var result struct {
			Count int `json:"count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return false, fmt.Errorf("failed to decode response: %w", err)
		}
		return result.Count > 0, nil
	}

	return false, nil
}

// CheckProjectAccess verifies if a user has access to a specific ADO project
func (h *EntraIDOAuthHandler) CheckProjectAccess(ctx context.Context, projectName string, accessToken string) (bool, error) {
	// Build API URL to get project details
	// https://dev.azure.com/{org}/_apis/projects/{project}?api-version=6.0
	apiURL := fmt.Sprintf("%s/_apis/projects/%s?api-version=6.0", h.adoOrgURL, url.PathEscape(projectName))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check project access: %w", err)
	}
	defer resp.Body.Close()

	// If we get a 200, user has access
	// If we get a 401/403/404, user doesn't have access
	return resp.StatusCode == http.StatusOK, nil
}

// Login initiates the Entra ID OAuth flow
func (h *EntraIDOAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Generate state parameter for CSRF protection
	state := generateRandomState()

	// Store state in session/cookie (simplified - in production use proper session management)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Get authorization URL
	authURL := h.GetAuthorizationURL(state)

	// Redirect to Entra ID login
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth callback from Entra ID
func (h *EntraIDOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify state parameter
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
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

	// Exchange authorization code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := h.ExchangeCode(ctx, code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user profile
	user, err := h.GetUserProfile(ctx, token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user profile: %v", err), http.StatusInternalServerError)
		return
	}

	// Check organization membership
	hasAccess, err := h.CheckOrganizationMembership(ctx, user.ID, token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check organization membership: %v", err), http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		http.Error(w, "User does not have access to the ADO organization", http.StatusForbidden)
		return
	}

	// Create JWT with user info and ADO token
	jwtManager, _ := NewJWTManager(h.config.SessionSecret, h.config.SessionDurationHours)

	// Create a GitHubUser struct for compatibility with existing JWT generation
	ghUser := &GitHubUser{
		Login: user.PublicAlias,
		Email: user.EmailAddress,
		Name:  user.DisplayName,
	}

	jwtToken, err := jwtManager.GenerateToken(ghUser, token.AccessToken)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	// Return JWT in response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"token": jwtToken,
		"user":  user.DisplayName,
		"email": user.EmailAddress,
	}); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

// GetUser returns the current authenticated user's information
func (h *EntraIDOAuthHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT (should be set by auth middleware)
	// For now, return a placeholder
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "User info endpoint - implement JWT extraction",
	}); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

// generateRandomState generates a random state parameter for OAuth
func generateRandomState() string {
	// Simple implementation - in production use crypto/rand
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
