package github

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v75/github"
	"github.com/jferrl/go-githubauth"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Client wraps GitHub REST and GraphQL clients with rate limiting and retry logic
type Client struct {
	rest           *github.Client
	graphql        *githubv4.Client
	baseURL        string
	token          string
	rateLimiter    *RateLimiter
	retryer        *Retryer
	circuitBreaker *CircuitBreaker
	logger         *slog.Logger
}

// ClientConfig configures the GitHub client
type ClientConfig struct {
	BaseURL     string
	Token       string
	Timeout     time.Duration
	RetryConfig RetryConfig
	Logger      *slog.Logger

	// GitHub App authentication (optional, takes precedence over Token if provided)
	AppID             int64  // GitHub App ID
	AppPrivateKey     string // Private key (file path or inline PEM)
	AppInstallationID int64  // Installation ID (optional: if 0, creates JWT-only client for enterprise-wide discovery)
}

// InstanceType represents the type of GitHub instance
type InstanceType int

const (
	// InstanceTypeGitHub is standard GitHub.com
	InstanceTypeGitHub InstanceType = iota
	// InstanceTypeGHEC is GitHub Enterprise Cloud with data residency
	InstanceTypeGHEC
	// InstanceTypeGHES is GitHub Enterprise Server (self-hosted)
	InstanceTypeGHES
)

const (
	// GitHubAPIURL is the standard GitHub.com API URL
	GitHubAPIURL = "https://api.github.com"
)

// detectInstanceType determines the type of GitHub instance from the base URL
func detectInstanceType(baseURL string) InstanceType {
	if baseURL == "" || baseURL == GitHubAPIURL {
		return InstanceTypeGitHub
	}

	// GitHub Enterprise Cloud with data residency uses .ghe.com domains
	// e.g., https://octocorp.ghe.com or https://api.octocorp.ghe.com
	if strings.Contains(baseURL, ".ghe.com") {
		return InstanceTypeGHEC
	}

	// Everything else is assumed to be GitHub Enterprise Server
	return InstanceTypeGHES
}

// buildGraphQLURL builds the correct GraphQL endpoint URL based on instance type
func buildGraphQLURL(baseURL string) string {
	instanceType := detectInstanceType(baseURL)

	switch instanceType {
	case InstanceTypeGitHub:
		return GitHubAPIURL + "/graphql"

	case InstanceTypeGHEC:
		// For GHE Cloud with data residency, convert domain to API endpoint
		// e.g., octocorp.ghe.com -> https://api.octocorp.ghe.com/graphql
		domain := strings.TrimPrefix(baseURL, "https://")
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "api.")
		domain = strings.TrimSuffix(domain, "/")
		return fmt.Sprintf("https://api.%s/graphql", domain)

	case InstanceTypeGHES:
		// For GitHub Enterprise Server, use /api/graphql path
		// Strip any existing /api/v3 or /api paths first to avoid duplication
		url := strings.TrimSuffix(baseURL, "/")
		url = strings.TrimSuffix(url, "/api/v3")
		url = strings.TrimSuffix(url, "/api")
		return url + "/api/graphql"

	default:
		// Default to GHES-style endpoint, strip any existing /api/v3 or /api paths
		url := strings.TrimSuffix(baseURL, "/")
		url = strings.TrimSuffix(url, "/api/v3")
		url = strings.TrimSuffix(url, "/api")
		return url + "/api/graphql"
	}
}

// CreateOrgSpecificClient creates a GitHub client with an org-specific installation token
// This is used for GitHub Enterprise Apps where each org has its own installation
// Parameters:
//   - jwtClient: A JWT-authenticated client (created with AppID + AppPrivateKey, no InstallationID)
//   - org: The organization name
//   - baseConfig: The base configuration (with AppID and AppPrivateKey)
//
// Returns a new client configured with the org's installation token
func CreateOrgSpecificClient(ctx context.Context, jwtClient *Client, org string, baseConfig ClientConfig) (*Client, error) {
	// Get the installation ID for this org
	installationID, err := jwtClient.GetOrganizationInstallationID(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation ID for org %s: %w", org, err)
	}

	jwtClient.logger.Info("Creating org-specific client",
		"org", org,
		"installation_id", installationID)

	// Create a new client config with the org's installation ID
	orgConfig := baseConfig
	orgConfig.AppInstallationID = installationID

	// Create the org-specific client
	orgClient, err := NewClient(orgConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for org %s: %w", org, err)
	}

	return orgClient, nil
}

// createGitHubAppTokenSource creates an oauth2.TokenSource for GitHub App authentication
// using the go-githubauth library which provides intelligent token caching and
// avoids mutex contention for concurrent requests
func createGitHubAppTokenSource(appID int64, privateKey string, installationID int64, baseURL string) (oauth2.TokenSource, error) {
	var privateKeyBytes []byte
	var err error

	// Check if privateKey is a file path or inline PEM
	if strings.HasPrefix(privateKey, "-----BEGIN") {
		// Inline PEM string
		privateKeyBytes = []byte(privateKey)
	} else {
		// File path
		// #nosec G304 -- private key file path is provided by configuration, not user input
		privateKeyBytes, err = os.ReadFile(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}
	}

	// Create GitHub App JWT token source (Application-level authentication)
	// This generates JWTs for authenticating as the GitHub App itself
	appTokenSource, err := githubauth.NewApplicationTokenSource(appID, privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create application token source: %w", err)
	}

	// Create Installation token source (Installation-level authentication)
	// This exchanges the App JWT for an installation access token
	// The go-githubauth library handles token caching and refresh automatically,
	// avoiding mutex contention that caused performance issues with concurrent requests
	installationTokenSource := githubauth.NewInstallationTokenSource(installationID, appTokenSource)

	return installationTokenSource, nil
}

// NewClient creates a new GitHub client with rate limiting and retry logic
// Supports both PAT and GitHub App authentication. If App credentials are provided,
// they take precedence over PAT.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	var httpClient *http.Client
	var authMethod string
	var token string

	// Determine authentication method
	if cfg.AppID > 0 && cfg.AppPrivateKey != "" {
		// GitHub App authentication (with or without installation token)
		var tokenSource oauth2.TokenSource
		var err error

		if cfg.AppInstallationID > 0 {
			// Use installation token (for org-specific operations)
			authMethod = "GitHub App (Installation)"
			cfg.Logger.Debug("Using GitHub App installation authentication",
				"app_id", cfg.AppID,
				"installation_id", cfg.AppInstallationID)

			tokenSource, err = createGitHubAppTokenSource(cfg.AppID, cfg.AppPrivateKey, cfg.AppInstallationID, cfg.BaseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to create GitHub App token source: %w", err)
			}
		} else {
			// Use JWT-only authentication (for App-level APIs like finding installations)
			authMethod = "GitHub App (JWT)"
			cfg.Logger.Debug("Using GitHub App JWT authentication (no installation)",
				"app_id", cfg.AppID)

			var privateKeyBytes []byte
			if strings.HasPrefix(cfg.AppPrivateKey, "-----BEGIN") {
				privateKeyBytes = []byte(cfg.AppPrivateKey)
			} else {
				// #nosec G304 -- private key file path is provided by configuration, not user input
				privateKeyBytes, err = os.ReadFile(cfg.AppPrivateKey)
				if err != nil {
					return nil, fmt.Errorf("failed to read private key file: %w", err)
				}
			}

			// Create JWT-only token source for App-level authentication
			tokenSource, err = githubauth.NewApplicationTokenSource(cfg.AppID, privateKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to create application token source: %w", err)
			}
		}

		// Create a custom transport with higher concurrency limits
		// The default http.DefaultTransport has MaxIdleConnsPerHost: 2, which severely
		// limits concurrent requests. With 5 discovery workers, increase to 100 to allow
		// all workers to make concurrent requests without blocking.
		customTransport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100, // Increased from default 2 to support concurrent workers
			MaxConnsPerHost:       0,   // 0 means unlimited
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		// Create HTTP client with OAuth2 transport and optimized settings for concurrent access
		httpClient = &http.Client{
			Transport: &oauth2.Transport{
				Source: oauth2.ReuseTokenSource(nil, tokenSource),
				Base:   customTransport,
			},
			Timeout: cfg.Timeout,
		}

		// Get the initial token for logging/debugging
		initialToken, err := tokenSource.Token()
		if err != nil {
			cfg.Logger.Warn("Failed to get initial token", "error", err)
		} else {
			token = initialToken.AccessToken
		}
	} else {
		// Use PAT authentication
		authMethod = "PAT"
		if cfg.Token == "" {
			return nil, fmt.Errorf("either Token or GitHub App credentials (AppID, AppPrivateKey, AppInstallationID) must be provided")
		}

		cfg.Logger.Debug("Using PAT authentication")

		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: cfg.Token},
		)
		httpClient = oauth2.NewClient(ctx, ts)
		httpClient.Timeout = cfg.Timeout
		token = cfg.Token
	}

	// Create REST client
	var restClient *github.Client
	if cfg.BaseURL == "" || cfg.BaseURL == GitHubAPIURL {
		restClient = github.NewClient(httpClient)
	} else {
		var err error
		restClient, err = github.NewClient(httpClient).WithEnterpriseURLs(cfg.BaseURL, cfg.BaseURL)
		if err != nil {
			return nil, WrapError(err, "NewClient", cfg.BaseURL)
		}
	}

	// Create GraphQL client with the correct endpoint based on instance type
	graphqlURL := buildGraphQLURL(cfg.BaseURL)
	var graphqlClient *githubv4.Client
	if cfg.BaseURL == "" || cfg.BaseURL == GitHubAPIURL {
		graphqlClient = githubv4.NewClient(httpClient)
	} else {
		graphqlClient = githubv4.NewEnterpriseClient(graphqlURL, httpClient)
	}

	cfg.Logger.Debug("GraphQL client configured",
		"base_url", cfg.BaseURL,
		"graphql_url", graphqlURL,
		"instance_type", detectInstanceType(cfg.BaseURL),
		"auth_method", authMethod)

	// Initialize rate limiter and retry logic
	rateLimiter := NewRateLimiter(cfg.Logger)
	retryer := NewRetryer(cfg.RetryConfig, rateLimiter, cfg.Logger)
	circuitBreaker := NewCircuitBreaker(5, 1*time.Minute, cfg.Logger)

	client := &Client{
		rest:           restClient,
		graphql:        graphqlClient,
		baseURL:        cfg.BaseURL,
		token:          token,
		rateLimiter:    rateLimiter,
		retryer:        retryer,
		circuitBreaker: circuitBreaker,
		logger:         cfg.Logger,
	}

	// Initialize rate limits (skip for JWT-only clients as they can't access rate limit API)
	if cfg.AppInstallationID != 0 || cfg.Token != "" {
		if err := client.updateRateLimits(context.Background()); err != nil {
			cfg.Logger.Warn("Failed to initialize rate limits", "error", err)
		}
	} else {
		cfg.Logger.Debug("Skipping rate limit initialization for JWT-only client")
	}

	return client, nil
}

// REST returns the underlying GitHub REST client
func (c *Client) REST() *github.Client {
	return c.rest
}

// BaseURL returns the base URL of the GitHub instance
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Token returns the authentication token
func (c *Client) Token() string {
	return c.token
}

// GraphQL returns the underlying GitHub GraphQL client
func (c *Client) GraphQL() *githubv4.Client {
	return c.graphql
}

// GetOrganizationInstallationID gets the installation ID for a specific organization
// This is needed when using Enterprise-level GitHub Apps to get org-specific tokens
// Requires JWT authentication (AppID + AppPrivateKey without InstallationID)
func (c *Client) GetOrganizationInstallationID(ctx context.Context, org string) (int64, error) {
	c.logger.Debug("Getting installation ID for organization", "org", org)

	installation, _, err := c.rest.Apps.FindOrganizationInstallation(ctx, org)
	if err != nil {
		return 0, fmt.Errorf("failed to get installation for org %s: %w", org, err)
	}

	if installation == nil || installation.ID == nil {
		return 0, fmt.Errorf("no installation found for org %s", org)
	}

	c.logger.Debug("Found installation for organization", "org", org, "installation_id", *installation.ID)
	return *installation.ID, nil
}

// ListAppInstallations lists all installations for the GitHub App
// This is the proper way for GitHub Apps to discover all organizations they're installed in
// Requires JWT authentication (AppID + AppPrivateKey without InstallationID)
// Returns a map of org login -> installation ID
func (c *Client) ListAppInstallations(ctx context.Context) (map[string]int64, error) {
	c.logger.Debug("Listing all app installations")

	installations := make(map[string]int64)
	opts := &github.ListOptions{PerPage: 100}

	for {
		installs, resp, err := c.rest.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list app installations: %w", err)
		}

		for _, install := range installs {
			if install.Account != nil && install.Account.Login != nil && install.ID != nil {
				orgLogin := *install.Account.Login
				installations[orgLogin] = *install.ID
				c.logger.Debug("Found app installation",
					"org", orgLogin,
					"installation_id", *install.ID,
					"type", install.Account.GetType())
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Info("Listed all app installations", "count", len(installations))
	return installations, nil
}

// GetRateLimiter returns the rate limiter
func (c *Client) GetRateLimiter() *RateLimiter {
	return c.rateLimiter
}

// GetRetryer returns the retryer
func (c *Client) GetRetryer() *Retryer {
	return c.retryer
}

// DoWithRetry executes a REST API operation with retry logic
func (c *Client) DoWithRetry(ctx context.Context, operation string, fn func(ctx context.Context) (*github.Response, error)) (*github.Response, error) {
	var resp *github.Response
	var lastErr error

	err := c.retryer.Do(ctx, operation, func(ctx context.Context) error {
		start := time.Now()
		c.logger.Debug("GitHub API call started",
			"operation", operation,
			"base_url", c.baseURL)

		var err error
		resp, err = fn(ctx)
		duration := time.Since(start)

		if err != nil {
			lastErr = WrapError(err, operation, c.baseURL)
			c.logger.Error("GitHub API call failed",
				"operation", operation,
				"base_url", c.baseURL,
				"duration_ms", duration.Milliseconds(),
				"error", lastErr)
			return lastErr
		}

		// Update rate limits from response
		if resp != nil && resp.Rate.Limit > 0 {
			c.rateLimiter.UpdateLimits(
				resp.Rate.Remaining,
				resp.Rate.Limit,
				resp.Rate.Reset.Time,
			)

			c.logger.Debug("GitHub API call completed",
				"operation", operation,
				"base_url", c.baseURL,
				"status_code", resp.StatusCode,
				"duration_ms", duration.Milliseconds(),
				"rate_limit_remaining", resp.Rate.Remaining,
				"rate_limit_limit", resp.Rate.Limit,
				"rate_limit_reset", resp.Rate.Reset.Time)
		} else {
			c.logger.Debug("GitHub API call completed",
				"operation", operation,
				"base_url", c.baseURL,
				"duration_ms", duration.Milliseconds())
		}

		return nil
	})

	if err != nil {
		return resp, lastErr
	}
	return resp, nil
}

// QueryWithRetry executes a GraphQL query with retry logic
func (c *Client) QueryWithRetry(ctx context.Context, operation string, query interface{}, variables map[string]interface{}) error {
	return c.retryer.Do(ctx, operation, func(ctx context.Context) error {
		start := time.Now()
		c.logger.Debug("GitHub GraphQL query started",
			"operation", operation,
			"base_url", c.baseURL,
			"variables", variables)

		err := c.graphql.Query(ctx, query, variables)
		duration := time.Since(start)

		if err != nil {
			wrappedErr := WrapError(err, operation, c.baseURL)
			c.logger.Error("GitHub GraphQL query failed",
				"operation", operation,
				"base_url", c.baseURL,
				"duration_ms", duration.Milliseconds(),
				"error", wrappedErr)
			return wrappedErr
		}

		c.logger.Debug("GitHub GraphQL query completed",
			"operation", operation,
			"base_url", c.baseURL,
			"duration_ms", duration.Milliseconds())

		return nil
	})
}

// MutateWithRetry executes a GraphQL mutation with retry logic
// The input parameter should be a typed githubv4 input struct (e.g., githubv4.CreateMigrationSourceInput)
// or a map[string]interface{} for dynamic inputs
func (c *Client) MutateWithRetry(ctx context.Context, operation string, mutation interface{}, input interface{}, variables map[string]interface{}) error {
	return c.retryer.Do(ctx, operation, func(ctx context.Context) error {
		start := time.Now()
		c.logger.Debug("GitHub GraphQL mutation started",
			"operation", operation,
			"base_url", c.baseURL,
			"variables", variables)

		err := c.graphql.Mutate(ctx, mutation, input, variables)
		duration := time.Since(start)

		if err != nil {
			wrappedErr := WrapError(err, operation, c.baseURL)
			c.logger.Error("GitHub GraphQL mutation failed",
				"operation", operation,
				"base_url", c.baseURL,
				"duration_ms", duration.Milliseconds(),
				"error", wrappedErr)
			return wrappedErr
		}

		c.logger.Debug("GitHub GraphQL mutation completed",
			"operation", operation,
			"base_url", c.baseURL,
			"duration_ms", duration.Milliseconds())

		return nil
	})
}

// updateRateLimits fetches and updates rate limit information
func (c *Client) updateRateLimits(ctx context.Context) error {
	limits, resp, err := c.rest.RateLimit.Get(ctx)
	if err != nil {
		return WrapError(err, "GetRateLimits", c.baseURL)
	}

	if limits != nil && limits.Core != nil {
		c.rateLimiter.UpdateLimits(
			limits.Core.Remaining,
			limits.Core.Limit,
			limits.Core.Reset.Time,
		)
	}

	if resp != nil && limits != nil && limits.Core != nil {
		c.logger.Debug("Rate limits fetched",
			"remaining", limits.Core.Remaining,
			"limit", limits.Core.Limit,
			"reset", limits.Core.Reset.Time)
	}

	return nil
}

// GetRateLimitStatus returns the current rate limit status
func (c *Client) GetRateLimitStatus(ctx context.Context) (*github.RateLimits, error) {
	limits, _, err := c.rest.RateLimit.Get(ctx)
	if err != nil {
		return nil, WrapError(err, "GetRateLimits", c.baseURL)
	}
	return limits, nil
}

// CheckRateLimit logs rate limit information
func (c *Client) CheckRateLimit(ctx context.Context) error {
	limits, err := c.GetRateLimitStatus(ctx)
	if err != nil {
		return err
	}

	if limits.Core != nil {
		c.logger.Info("Rate limit status",
			"remaining", limits.Core.Remaining,
			"limit", limits.Core.Limit,
			"reset", limits.Core.Reset.Time)

		c.rateLimiter.UpdateLimits(
			limits.Core.Remaining,
			limits.Core.Limit,
			limits.Core.Reset.Time,
		)
	}

	return nil
}

// ListRepositories lists all repositories for an organization with pagination
func (c *Client) ListRepositories(ctx context.Context, org string) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	c.logger.Info("Listing repositories", "org", org)

	for {
		var repos []*github.Repository
		var resp *github.Response

		err := c.retryer.Do(ctx, "ListRepositories", func(ctx context.Context) error {
			var err error
			repos, resp, err = c.rest.Repositories.ListByOrg(ctx, org, opt)
			if err != nil {
				return WrapError(err, "ListByOrg", c.baseURL)
			}

			// Update rate limits
			if resp != nil && resp.Rate.Limit > 0 {
				c.rateLimiter.UpdateLimits(
					resp.Rate.Remaining,
					resp.Rate.Limit,
					resp.Rate.Reset.Time,
				)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	c.logger.Info("Repository listing complete",
		"org", org,
		"total_repos", len(allRepos))

	return allRepos, nil
}

// GetRepository gets a single repository by owner and name
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	var repository *github.Repository

	err := c.retryer.Do(ctx, "GetRepository", func(ctx context.Context) error {
		var resp *github.Response
		var err error
		repository, resp, err = c.rest.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return WrapError(err, "Get", c.baseURL)
		}

		// Update rate limits
		if resp != nil && resp.Rate.Limit > 0 {
			c.rateLimiter.UpdateLimits(
				resp.Rate.Remaining,
				resp.Rate.Limit,
				resp.Rate.Reset.Time,
			)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return repository, nil
}

// TestAuthentication verifies that the client is authenticated properly
func (c *Client) TestAuthentication(ctx context.Context) error {
	c.logger.Info("Testing GitHub authentication")

	var user *github.User
	err := c.retryer.Do(ctx, "TestAuthentication", func(ctx context.Context) error {
		var resp *github.Response
		var err error
		user, resp, err = c.rest.Users.Get(ctx, "")
		if err != nil {
			return WrapError(err, "GetAuthenticatedUser", c.baseURL)
		}

		// Update rate limits
		if resp != nil && resp.Rate.Limit > 0 {
			c.rateLimiter.UpdateLimits(
				resp.Rate.Remaining,
				resp.Rate.Limit,
				resp.Rate.Reset.Time,
			)
		}
		return nil
	})

	if err != nil {
		c.logger.Error("Authentication test failed", "error", err)
		return err
	}

	c.logger.Info("Authentication successful",
		"user", user.GetLogin(),
		"type", user.GetType())

	return nil
}

// RepositoryURL converts a repository full name (org/repo) to a full repository URL
// based on the client's base URL. This is useful for converting destination repository
// names to URLs for display and linking purposes.
func (c *Client) RepositoryURL(fullName string) string {
	// Parse the base URL to get the domain
	instanceType := detectInstanceType(c.baseURL)

	switch instanceType {
	case InstanceTypeGitHub:
		// Standard GitHub.com
		return fmt.Sprintf("https://github.com/%s", fullName)

	case InstanceTypeGHEC:
		// GitHub Enterprise Cloud with data residency (e.g., octocorp.ghe.com)
		domain := strings.TrimPrefix(c.baseURL, "https://")
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "api.")
		domain = strings.TrimSuffix(domain, "/")
		domain = strings.TrimSuffix(domain, "/api/v3")
		domain = strings.TrimSuffix(domain, "/api")
		return fmt.Sprintf("https://%s/%s", domain, fullName)

	case InstanceTypeGHES:
		// GitHub Enterprise Server
		domain := strings.TrimSuffix(c.baseURL, "/")
		domain = strings.TrimSuffix(domain, "/api/v3")
		domain = strings.TrimSuffix(domain, "/api")
		return fmt.Sprintf("%s/%s", domain, fullName)

	default:
		// Fallback to base URL with full name
		domain := strings.TrimSuffix(c.baseURL, "/")
		domain = strings.TrimSuffix(domain, "/api/v3")
		domain = strings.TrimSuffix(domain, "/api")
		return fmt.Sprintf("%s/%s", domain, fullName)
	}
}

// UnlockRepository unlocks a repository that was locked during a migration.
// This is used when a migration fails and the source repository remains locked.
// See: https://docs.github.com/en/rest/migrations/orgs#unlock-an-organization-repository
func (c *Client) UnlockRepository(ctx context.Context, org, repo string, migrationID int64) error {
	_, err := c.DoWithRetry(ctx, "UnlockRepository", func(ctx context.Context) (*github.Response, error) {
		req, err := c.rest.NewRequest("DELETE",
			fmt.Sprintf("orgs/%s/migrations/%d/repos/%s/lock", org, migrationID, repo),
			nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.rest.Do(ctx, req, nil)
		return resp, err
	})

	if err != nil {
		return WrapError(err, "UnlockRepository", c.baseURL)
	}

	c.logger.Info("Repository unlocked successfully",
		"org", org,
		"repo", repo,
		"migration_id", migrationID)

	return nil
}

// ListEnterpriseOrganizations lists all organizations in an enterprise using GraphQL
func (c *Client) ListEnterpriseOrganizations(ctx context.Context, enterpriseSlug string) ([]string, error) {
	c.logger.Info("Listing organizations for enterprise", "enterprise", enterpriseSlug)

	var allOrgs []string
	var endCursor *githubv4.String

	// GraphQL query for enterprise organizations
	var query struct {
		Enterprise struct {
			Organizations struct {
				Nodes []struct {
					Login githubv4.String
				}
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"organizations(first: 100, after: $cursor)"`
		} `graphql:"enterprise(slug: $slug)"`
	}

	for {
		variables := map[string]interface{}{
			"slug":   githubv4.String(enterpriseSlug),
			"cursor": endCursor,
		}

		err := c.QueryWithRetry(ctx, "ListEnterpriseOrganizations", &query, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to list enterprise organizations: %w", err)
		}

		// Collect organization logins
		for _, org := range query.Enterprise.Organizations.Nodes {
			allOrgs = append(allOrgs, string(org.Login))
		}

		if !query.Enterprise.Organizations.PageInfo.HasNextPage {
			break
		}
		endCursor = &query.Enterprise.Organizations.PageInfo.EndCursor
	}

	c.logger.Info("Enterprise organizations listed",
		"enterprise", enterpriseSlug,
		"total_orgs", len(allOrgs))

	return allOrgs, nil
}
