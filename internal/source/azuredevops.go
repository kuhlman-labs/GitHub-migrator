package source

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
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
//
//nolint:dupl // Similar to GitHub clone but with ADO-specific authentication
func (p *AzureDevOpsProvider) CloneRepository(ctx context.Context, info RepositoryInfo, destPath string, opts CloneOptions) error {
	// Validate and sanitize destination path to prevent command injection
	if err := ValidateDestPath(destPath); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	// Get authenticated URL with validation
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
	// Arguments are validated and sanitized before use
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
func (p *AzureDevOpsProvider) GetAuthenticatedCloneURL(cloneURL string) (string, error) {
	// Validate and parse the clone URL
	if err := ValidateCloneURL(cloneURL); err != nil {
		return "", fmt.Errorf("invalid clone URL: %w", err)
	}

	parsedURL, err := url.Parse(cloneURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse clone URL: %w", err)
	}

	// Azure DevOps PAT authentication format: https://PAT@dev.azure.com/org/project/_git/repo
	// The PAT goes in the username field with no password (most reliable method)
	// Alternative format is username:PAT@ but PAT alone works universally
	parsedURL.User = url.User(p.token)

	return parsedURL.String(), nil
}

// ValidateCredentials validates that the provider's credentials are valid
func (p *AzureDevOpsProvider) ValidateCredentials(ctx context.Context) error {
	// Build base API URL to list projects
	// https://dev.azure.com/{org}/_apis/projects?api-version=6.0
	apiURL := fmt.Sprintf("https://dev.azure.com/%s/_apis/projects?api-version=6.0&$top=1", p.organization)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header (PAT as Basic auth with empty username)
	req.SetBasicAuth("", p.token)
	req.Header.Set("Accept", "application/json")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrAuthenticationFailed, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: invalid credentials", ErrAuthenticationFailed)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to validate credentials: status %d", resp.StatusCode)
	}

	return nil
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

// fetchLFSObjects fetches Git LFS objects for a cloned repository
func (p *AzureDevOpsProvider) fetchLFSObjects(ctx context.Context, repoPath string) error {
	// Validate path before using as working directory
	if err := ValidateDestPath(repoPath); err != nil {
		return fmt.Errorf("invalid repository path: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "lfs", "fetch", "--all")
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git lfs fetch failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// Note: validateCloneURL, validateDestPath, and sanitizeGitError are defined in github.go and reused here
