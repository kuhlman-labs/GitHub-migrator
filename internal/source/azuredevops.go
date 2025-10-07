package source

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// AzureDevOpsProvider implements the Provider interface for Azure DevOps (formerly VSTS)
type AzureDevOpsProvider struct {
	organization string
	token        string
	username     string // Optional: can be empty, defaults to token
	name         string
}

// NewAzureDevOpsProvider creates a new Azure DevOps provider
// organization is the Azure DevOps organization name
// username is optional; if empty, token will be used as username
func NewAzureDevOpsProvider(organization, token, username string) (*AzureDevOpsProvider, error) {
	if organization == "" {
		return nil, fmt.Errorf("organization is required")
	}
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	// If username is empty, use the token as username (common pattern for PATs)
	if username == "" {
		username = token
	}

	name := fmt.Sprintf("Azure DevOps (%s)", organization)

	return &AzureDevOpsProvider{
		organization: organization,
		token:        token,
		username:     username,
		name:         name,
	}, nil
}

// Type returns the provider type
func (p *AzureDevOpsProvider) Type() ProviderType {
	return ProviderAzureDevOps
}

// Name returns a human-readable name for this provider instance
func (p *AzureDevOpsProvider) Name() string {
	return p.name
}

// CloneRepository clones a repository to the specified directory
func (p *AzureDevOpsProvider) CloneRepository(ctx context.Context, info RepositoryInfo, destPath string, opts CloneOptions) error {
	// TODO: Implement Azure DevOps clone logic
	// Similar to GitHub but with different URL structure and auth
	return fmt.Errorf("Azure DevOps provider not yet implemented")
}

// GetAuthenticatedCloneURL returns a clone URL with embedded credentials
func (p *AzureDevOpsProvider) GetAuthenticatedCloneURL(cloneURL string) (string, error) {
	// Parse the clone URL
	parsedURL, err := url.Parse(cloneURL)
	if err != nil {
		return "", fmt.Errorf("invalid clone URL: %w", err)
	}

	// Azure DevOps format: https://USERNAME:TOKEN@dev.azure.com/org/project/_git/repo
	// For PATs, username can be anything or the token itself
	parsedURL.User = url.UserPassword(p.username, p.token)

	return parsedURL.String(), nil
}

// ValidateCredentials validates that the provider's credentials are valid
func (p *AzureDevOpsProvider) ValidateCredentials(ctx context.Context) error {
	// TODO: Implement Azure DevOps credential validation
	// Use Azure DevOps REST API to verify token
	return fmt.Errorf("Azure DevOps provider not yet implemented")
}

// SupportsFeature indicates whether a specific feature is supported
func (p *AzureDevOpsProvider) SupportsFeature(feature Feature) bool {
	// Azure DevOps supports some features with different names/implementations
	switch feature {
	case FeatureLFS, FeatureSubmodules:
		return true
	case FeatureWiki:
		// Azure DevOps has wikis but they're different
		return true
	case FeaturePages, FeatureActions, FeatureDiscussions:
		// These are GitHub-specific
		return false
	default:
		return false
	}
}

// NormalizeRepoURL normalizes Azure DevOps repository URLs which can have various formats
func (p *AzureDevOpsProvider) NormalizeRepoURL(rawURL string) (string, error) {
	// Azure DevOps URLs can be:
	// https://dev.azure.com/org/project/_git/repo
	// https://org.visualstudio.com/project/_git/repo (legacy)

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Convert legacy visualstudio.com URLs to dev.azure.com format
	if strings.Contains(parsedURL.Host, "visualstudio.com") {
		// Extract org from subdomain
		parts := strings.Split(parsedURL.Host, ".")
		if len(parts) > 0 {
			org := parts[0]
			parsedURL.Host = "dev.azure.com"
			parsedURL.Path = "/" + org + parsedURL.Path
		}
	}

	return parsedURL.String(), nil
}
