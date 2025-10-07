package source

import (
	"context"
	"fmt"
	"net/url"
)

// GitLabProvider implements the Provider interface for GitLab.com and self-hosted GitLab
type GitLabProvider struct {
	baseURL string
	token   string
	name    string
}

// NewGitLabProvider creates a new GitLab provider
// baseURL should be the GitLab instance URL (e.g., "https://gitlab.com" or "https://gitlab.example.com")
// For gitlab.com, use "https://gitlab.com"
func NewGitLabProvider(baseURL, token string) (*GitLabProvider, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	// Determine friendly name based on URL
	name := "GitLab.com"
	if baseURL != "https://gitlab.com" {
		parsedURL, err := url.Parse(baseURL)
		if err == nil {
			name = fmt.Sprintf("GitLab (%s)", parsedURL.Host)
		}
	}

	return &GitLabProvider{
		baseURL: baseURL,
		token:   token,
		name:    name,
	}, nil
}

// Type returns the provider type
func (p *GitLabProvider) Type() ProviderType {
	return ProviderGitLab
}

// Name returns a human-readable name for this provider instance
func (p *GitLabProvider) Name() string {
	return p.name
}

// CloneRepository clones a repository to the specified directory
func (p *GitLabProvider) CloneRepository(ctx context.Context, info RepositoryInfo, destPath string, opts CloneOptions) error {
	// TODO: Implement GitLab clone logic
	// Similar to GitHub but uses different auth format: https://oauth2:TOKEN@gitlab.com/org/repo.git
	return fmt.Errorf("GitLab provider not yet implemented")
}

// GetAuthenticatedCloneURL returns a clone URL with embedded credentials
func (p *GitLabProvider) GetAuthenticatedCloneURL(cloneURL string) (string, error) {
	// Parse the clone URL
	parsedURL, err := url.Parse(cloneURL)
	if err != nil {
		return "", fmt.Errorf("invalid clone URL: %w", err)
	}

	// GitLab uses oauth2 as username with token as password
	// Format: https://oauth2:TOKEN@gitlab.com/org/repo.git
	parsedURL.User = url.UserPassword("oauth2", p.token)

	return parsedURL.String(), nil
}

// ValidateCredentials validates that the provider's credentials are valid
func (p *GitLabProvider) ValidateCredentials(ctx context.Context) error {
	// TODO: Implement GitLab credential validation
	// Use GitLab API to verify token: GET /api/v4/user
	return fmt.Errorf("GitLab provider not yet implemented")
}

// SupportsFeature indicates whether a specific feature is supported
func (p *GitLabProvider) SupportsFeature(feature Feature) bool {
	// GitLab supports most features, but not all GitHub-specific ones
	switch feature {
	case FeatureLFS, FeatureSubmodules, FeatureWiki, FeaturePages:
		return true
	case FeatureActions:
		// GitLab has CI/CD but it's different from GitHub Actions
		return false
	case FeatureDiscussions:
		// GitLab has issues/merge requests but not GitHub-style discussions
		return false
	default:
		return false
	}
}
