package source

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
)

// GitHubProvider implements the Provider interface for GitHub and GitHub Enterprise Server
type GitHubProvider struct {
	baseURL string
	token   string
	client  *github.Client
	name    string
}

// NewGitHubProvider creates a new GitHub provider
// baseURL should be the API base URL (e.g., "https://api.github.com" or "https://github.example.com/api/v3")
// For github.com, use "https://api.github.com"
// For GHES, use "https://your-ghes-instance.com/api/v3"
func NewGitHubProvider(baseURL, token string) (*GitHubProvider, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	var client *github.Client
	var err error

	// Determine if this is GitHub.com or GHES
	if baseURL == "" || baseURL == "https://api.github.com" {
		client = github.NewClient(tc)
	} else {
		client, err = github.NewClient(tc).WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client: %w", err)
		}
	}

	// Determine friendly name based on URL
	name := "GitHub.com"
	if baseURL != "" && baseURL != "https://api.github.com" {
		parsedURL, err := url.Parse(baseURL)
		if err == nil {
			name = fmt.Sprintf("GHES (%s)", parsedURL.Host)
		}
	}

	return &GitHubProvider{
		baseURL: baseURL,
		token:   token,
		client:  client,
		name:    name,
	}, nil
}

// Type returns the provider type
func (p *GitHubProvider) Type() ProviderType {
	return ProviderGitHub
}

// Name returns a human-readable name for this provider instance
func (p *GitHubProvider) Name() string {
	return p.name
}

// CloneRepository clones a repository to the specified directory
//
//nolint:dupl // Similar to ADO clone but with GitHub-specific authentication
func (p *GitHubProvider) CloneRepository(ctx context.Context, info RepositoryInfo, destPath string, opts CloneOptions) error {
	// Get authenticated URL
	authURL, err := p.GetAuthenticatedCloneURL(info.CloneURL)
	if err != nil {
		return fmt.Errorf("failed to get authenticated URL: %w", err)
	}

	// Build git clone command with options
	args := []string{"clone"}

	if opts.Shallow {
		args = append(args, "--depth=1")
	}

	if opts.Bare {
		args = append(args, "--bare")
	}

	if !opts.IncludeSubmodules {
		args = append(args, "--no-recurse-submodules")
	}

	args = append(args, authURL, destPath)

	// Execute git clone
	// #nosec G204 -- arguments are constructed from controlled inputs
	cmd := exec.CommandContext(ctx, "git", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Set GIT_TERMINAL_PROMPT=0 to prevent interactive prompts
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")

	if err := cmd.Run(); err != nil {
		// Sanitize error message to avoid leaking token
		sanitizedErr := sanitizeGitError(stderr.String(), p.token)
		return fmt.Errorf("%w: %s", ErrCloneFailed, sanitizedErr)
	}

	// If LFS is requested and supported, fetch LFS objects
	if opts.IncludeLFS {
		if err := p.fetchLFSObjects(ctx, destPath); err != nil {
			// Log warning but don't fail - LFS might not be configured
			return fmt.Errorf("warning: failed to fetch LFS objects: %w", err)
		}
	}

	return nil
}

// GetAuthenticatedCloneURL returns a clone URL with embedded credentials
func (p *GitHubProvider) GetAuthenticatedCloneURL(cloneURL string) (string, error) {
	// Parse the clone URL
	parsedURL, err := url.Parse(cloneURL)
	if err != nil {
		return "", fmt.Errorf("invalid clone URL: %w", err)
	}

	// GitHub uses token as username with empty password
	// Format: https://TOKEN@github.com/org/repo.git
	parsedURL.User = url.User(p.token)

	return parsedURL.String(), nil
}

// ValidateCredentials validates that the provider's credentials are valid
func (p *GitHubProvider) ValidateCredentials(ctx context.Context) error {
	// Try to get the authenticated user
	_, resp, err := p.client.Users.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return fmt.Errorf("%w: invalid token", ErrAuthenticationFailed)
		}
		return fmt.Errorf("failed to validate credentials: %w", err)
	}

	return nil
}

// SupportsFeature indicates whether a specific feature is supported
func (p *GitHubProvider) SupportsFeature(feature Feature) bool {
	// GitHub supports all common features
	switch feature {
	case FeatureLFS, FeatureSubmodules, FeatureWiki, FeaturePages, FeatureActions, FeatureDiscussions:
		return true
	default:
		return false
	}
}

// fetchLFSObjects fetches Git LFS objects for a cloned repository
func (p *GitHubProvider) fetchLFSObjects(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "lfs", "fetch", "--all")
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git lfs fetch failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// sanitizeGitError removes token from error messages
func sanitizeGitError(errMsg, token string) string {
	if token == "" {
		return errMsg
	}
	// Replace token with [REDACTED]
	return strings.ReplaceAll(errMsg, token, "[REDACTED]")
}
