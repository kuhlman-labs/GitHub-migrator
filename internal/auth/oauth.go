package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const githubDotComURL = "https://github.com"

// OAuthHandler handles GitHub OAuth authentication flow
type OAuthHandler struct {
	config  *config.AuthConfig
	logger  *slog.Logger
	oauth   *oauth2.Config
	baseURL string // GitHub base URL (for GHES support)
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(cfg *config.AuthConfig, logger *slog.Logger, githubBaseURL string) *OAuthHandler {
	// Convert API base URL to OAuth base URL
	oauthBaseURL := BuildOAuthURL(githubBaseURL)

	// Determine OAuth endpoints based on GitHub base URL
	var endpoint oauth2.Endpoint
	if oauthBaseURL == githubDotComURL {
		// GitHub.com
		endpoint = github.Endpoint
	} else {
		// GitHub Enterprise Server
		baseURL := oauthBaseURL
		if baseURL[len(baseURL)-1] == '/' {
			baseURL = baseURL[:len(baseURL)-1]
		}
		endpoint = oauth2.Endpoint{
			AuthURL:  baseURL + "/login/oauth/authorize",
			TokenURL: baseURL + "/login/oauth/access_token",
		}
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GitHubOAuthClientID,
		ClientSecret: cfg.GitHubOAuthClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  cfg.CallbackURL,
		Scopes:       []string{"read:org", "read:user", "user:email", "admin:enterprise"},
	}

	return &OAuthHandler{
		config:  cfg,
		logger:  logger,
		oauth:   oauthConfig,
		baseURL: githubBaseURL,
	}
}

// HandleLogin initiates the OAuth flow
func (h *OAuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Generate random state token
	state, err := generateStateToken()
	if err != nil {
		h.logger.Error("Failed to generate state token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state in secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to GitHub OAuth
	authURL := h.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback processes the OAuth callback
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state token
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		h.logger.Error("Missing state cookie", "error", err)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

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

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		h.logger.Error("Missing authorization code")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := h.oauth.Exchange(context.Background(), code)
	if err != nil {
		h.logger.Error("Failed to exchange code for token", "error", err)
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Get user info
	user, err := h.getGitHubUser(token.AccessToken)
	if err != nil {
		h.logger.Error("Failed to get user info", "error", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	h.logger.Info("User authenticated", "login", user.Login, "id", user.ID)

	// Return user and token for further processing by handler
	w.Header().Set("X-User-Login", user.Login)
	w.Header().Set("X-User-ID", fmt.Sprintf("%d", user.ID))

	// Store in context for handler to access
	ctx := context.WithValue(r.Context(), contextKeyGitHubUser, user)
	ctx = context.WithValue(ctx, contextKeyGitHubToken, token.AccessToken)
	*r = *r.WithContext(ctx)
}

// GetGitHubUser fetches user information from GitHub API
func (h *OAuthHandler) getGitHubUser(accessToken string) (*GitHubUser, error) {
	apiURL := h.baseURL
	if apiURL == "" {
		apiURL = defaultGitHubAPIURL
	}
	if apiURL[len(apiURL)-1] == '/' {
		apiURL = apiURL[:len(apiURL)-1]
	}

	req, err := http.NewRequest("GET", apiURL+"/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	// Fetch user's email if not public
	if user.Email == "" {
		user.Email = h.fetchUserEmail(accessToken)
	}

	return &user, nil
}

// fetchUserEmail attempts to fetch a user's email address
func (h *OAuthHandler) fetchUserEmail(accessToken string) string {
	emails, err := h.getUserEmails(accessToken)
	if err != nil || len(emails) == 0 {
		return ""
	}

	// Find primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email
		}
	}

	// Fall back to first verified email
	for _, email := range emails {
		if email.Verified {
			return email.Email
		}
	}

	return ""
}

// getUserEmails fetches user's email addresses from GitHub API
func (h *OAuthHandler) getUserEmails(accessToken string) ([]GitHubEmail, error) {
	apiURL := h.baseURL
	if apiURL == "" {
		apiURL = defaultGitHubAPIURL
	}
	if apiURL[len(apiURL)-1] == '/' {
		apiURL = apiURL[:len(apiURL)-1]
	}

	req, err := http.NewRequest("GET", apiURL+"/user/emails", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// GitHubEmail represents a user's email address
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// generateStateToken generates a random state token for CSRF protection
func generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Context keys
type contextKey string

const (
	contextKeyGitHubUser  contextKey = "github_user"
	contextKeyGitHubToken contextKey = "github_token"
)

// GetGitHubUserFromContext retrieves the GitHub user from OAuth callback context
func GetGitHubUserFromContext(ctx context.Context) (*GitHubUser, bool) {
	user, ok := ctx.Value(contextKeyGitHubUser).(*GitHubUser)
	return user, ok
}

// GetTokenFromContext retrieves the GitHub token from OAuth callback context
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(contextKeyGitHubToken).(string)
	return token, ok
}

// BuildOAuthURL builds the GitHub OAuth URL for a given base URL
func BuildOAuthURL(baseURL string) string {
	if baseURL == "" {
		return githubDotComURL
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return githubDotComURL
	}

	// For api.github.com, return github.com
	if parsedURL.Host == "api.github.com" {
		return githubDotComURL
	}

	// For GitHub with data residency (e.g., api.company.ghe.com), strip the "api." prefix
	// OAuth endpoints are at company.ghe.com, not api.company.ghe.com
	parsedURL.Host = strings.TrimPrefix(parsedURL.Host, "api.")

	// For GHES, remove "/api/v3" from path if present
	parsedURL.Path = ""
	return parsedURL.String()
}
