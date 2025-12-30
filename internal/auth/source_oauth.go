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
	"strconv"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// SourceOAuthHandler handles OAuth authentication for a specific source
type SourceOAuthHandler struct {
	source  *models.Source
	logger  *slog.Logger
	oauth   *oauth2.Config
	baseURL string
}

// NewSourceOAuthHandler creates an OAuth handler for a specific source
func NewSourceOAuthHandler(source *models.Source, callbackURL string, logger *slog.Logger) (*SourceOAuthHandler, error) {
	if source == nil {
		return nil, fmt.Errorf("source is required")
	}

	if !source.HasOAuth() {
		return nil, fmt.Errorf("source %s does not have OAuth configured", source.Name)
	}

	var oauthConfig *oauth2.Config
	var baseURL string

	if source.IsGitHub() {
		oauthConfig, baseURL = buildGitHubOAuthConfig(source, callbackURL)
	} else if source.IsAzureDevOps() {
		oauthConfig, baseURL = buildEntraIDOAuthConfig(source, callbackURL)
	} else {
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}

	return &SourceOAuthHandler{
		source:  source,
		logger:  logger,
		oauth:   oauthConfig,
		baseURL: baseURL,
	}, nil
}

// buildGitHubOAuthConfig creates OAuth config for GitHub/GHES sources
func buildGitHubOAuthConfig(source *models.Source, callbackURL string) (*oauth2.Config, string) {
	oauthBaseURL := BuildOAuthURL(source.BaseURL)

	var endpoint oauth2.Endpoint
	if oauthBaseURL == githubDotComURL {
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

	return &oauth2.Config{
		ClientID:     *source.OAuthClientID,
		ClientSecret: *source.OAuthClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  callbackURL,
		Scopes:       []string{"read:org", "read:user", "user:email", "repo"},
	}, source.BaseURL
}

// buildEntraIDOAuthConfig creates OAuth config for Azure DevOps sources via Entra ID
func buildEntraIDOAuthConfig(source *models.Source, callbackURL string) (*oauth2.Config, string) {
	tenantID := *source.EntraTenantID

	return &oauth2.Config{
		ClientID:     *source.EntraClientID,
		ClientSecret: *source.EntraClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID),
			TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		},
		RedirectURL: callbackURL,
		Scopes:      []string{"499b84ac-1321-427f-aa17-267ca6975798/.default", "openid", "profile", "email"},
	}, source.BaseURL
}

// GetSource returns the source this handler is for
func (h *SourceOAuthHandler) GetSource() *models.Source {
	return h.source
}

// GetAuthURL returns the OAuth authorization URL with state
func (h *SourceOAuthHandler) GetAuthURL(state string) string {
	return h.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode exchanges an authorization code for an access token
func (h *SourceOAuthHandler) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return h.oauth.Exchange(ctx, code)
}

// GetGitHubUser fetches user information from GitHub API
func (h *SourceOAuthHandler) GetGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	apiURL := h.baseURL
	if apiURL == "" {
		apiURL = defaultGitHubAPIURL
	}
	if apiURL[len(apiURL)-1] == '/' {
		apiURL = apiURL[:len(apiURL)-1]
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"/user", nil)
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
		user.Email = h.fetchUserEmail(ctx, accessToken)
	}

	return &user, nil
}

// fetchUserEmail attempts to fetch a user's primary email address
func (h *SourceOAuthHandler) fetchUserEmail(ctx context.Context, accessToken string) string {
	apiURL := h.baseURL
	if apiURL == "" {
		apiURL = defaultGitHubAPIURL
	}
	if apiURL[len(apiURL)-1] == '/' {
		apiURL = apiURL[:len(apiURL)-1]
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"/user/emails", nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
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

// GetEntraIDUser fetches user information from Microsoft Graph API
func (h *SourceOAuthHandler) GetEntraIDUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	var profile struct {
		ID                string `json:"id"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	// Convert Entra ID profile to GitHubUser format for consistency
	email := profile.Mail
	if email == "" {
		email = profile.UserPrincipalName
	}

	// Parse ID to int64 (Entra ID uses GUIDs, so we'll hash it)
	var userID int64
	for _, c := range profile.ID {
		userID = userID*31 + int64(c)
	}
	if userID < 0 {
		userID = -userID
	}

	return &GitHubUser{
		ID:    userID,
		Login: profile.UserPrincipalName,
		Name:  profile.DisplayName,
		Email: email,
	}, nil
}

// GetUser fetches user information based on source type
func (h *SourceOAuthHandler) GetUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	if h.source.IsGitHub() {
		return h.GetGitHubUser(ctx, accessToken)
	}
	return h.GetEntraIDUser(ctx, accessToken)
}

// SourceOAuthState contains state information for source-aware OAuth
type SourceOAuthState struct {
	RandomState string `json:"r"` // Random CSRF token
	SourceID    int64  `json:"s"` // Source ID
}

// EncodeSourceState encodes source OAuth state for the OAuth flow
func EncodeSourceState(sourceID int64) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	state := SourceOAuthState{
		RandomState: base64.URLEncoding.EncodeToString(randomBytes),
		SourceID:    sourceID,
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(stateJSON), nil
}

// DecodeSourceState decodes source OAuth state from the OAuth callback
func DecodeSourceState(encoded string) (*SourceOAuthState, error) {
	stateJSON, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		// Try to handle legacy state format (just random string without source ID)
		return nil, fmt.Errorf("invalid state format: %w", err)
	}

	var state SourceOAuthState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("invalid state JSON: %w", err)
	}

	return &state, nil
}

// GetSourceIDFromRequest extracts source_id from request query params
func GetSourceIDFromRequest(r *http.Request) (int64, error) {
	sourceIDStr := r.URL.Query().Get("source_id")
	if sourceIDStr == "" {
		return 0, nil // No source specified, will use fallback
	}

	sourceID, err := strconv.ParseInt(sourceIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid source_id: %w", err)
	}

	return sourceID, nil
}
