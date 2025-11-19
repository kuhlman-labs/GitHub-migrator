package source

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ErrAuthenticationFailed indicates authentication to the source system failed
var ErrAuthenticationFailed = errors.New("authentication failed")

// ErrRepositoryNotFound indicates the repository does not exist
var ErrRepositoryNotFound = errors.New("repository not found")

// ErrCloneFailed indicates the clone operation failed
var ErrCloneFailed = errors.New("clone operation failed")

// ProviderType represents the type of source control system
type ProviderType string

const (
	// ProviderGitHub represents GitHub.com or GitHub Enterprise Server
	ProviderGitHub ProviderType = "github"
	// ProviderGitLab represents GitLab.com or self-hosted GitLab
	ProviderGitLab ProviderType = "gitlab"
	// ProviderAzureDevOps represents Azure DevOps (formerly VSTS)
	ProviderAzureDevOps ProviderType = "azuredevops"
)

// CloneOptions contains options for cloning a repository
type CloneOptions struct {
	// Shallow indicates whether to perform a shallow clone (--depth=1)
	Shallow bool
	// Bare indicates whether to perform a bare clone (--bare)
	Bare bool
	// IncludeLFS indicates whether to fetch LFS objects
	IncludeLFS bool
	// IncludeSubmodules indicates whether to clone submodules
	IncludeSubmodules bool
}

// DefaultCloneOptions returns default clone options for discovery
// Uses full clone for accurate git-sizer analysis
func DefaultCloneOptions() CloneOptions {
	return CloneOptions{
		Shallow:           false, // Full clone needed for accurate git-sizer metrics
		Bare:              false, // Need working tree for file analysis
		IncludeLFS:        false, // Don't fetch LFS content during discovery
		IncludeSubmodules: false, // Don't clone submodules during discovery
	}
}

// ShallowCloneOptions returns options for fast shallow clones
// Use this when full git history is not needed (e.g., quick file checks)
func ShallowCloneOptions() CloneOptions {
	return CloneOptions{
		Shallow:           true,  // Shallow clone for speed
		Bare:              false, // Need working tree for file analysis
		IncludeLFS:        false, // Don't fetch LFS content
		IncludeSubmodules: false, // Don't clone submodules
	}
}

// RepositoryInfo contains basic information about a repository
type RepositoryInfo struct {
	// FullName is the full name of the repository (org/repo)
	FullName string
	// CloneURL is the HTTPS clone URL
	CloneURL string
	// DefaultBranch is the default branch name
	DefaultBranch string
	// IsPrivate indicates if the repository is private
	IsPrivate bool
	// Size is the repository size in bytes
	Size int64
}

// Provider defines the interface for source control system providers
// This abstraction allows supporting multiple platforms (GitHub, GitLab, ADO, etc.)
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// Name returns a human-readable name for this provider instance
	Name() string

	// CloneRepository clones a repository to the specified directory
	// Returns the path to the cloned repository
	CloneRepository(ctx context.Context, info RepositoryInfo, destPath string, opts CloneOptions) error

	// GetAuthenticatedCloneURL returns a clone URL with embedded credentials
	// This is used by the git command for authentication
	GetAuthenticatedCloneURL(cloneURL string) (string, error)

	// ValidateCredentials validates that the provider's credentials are valid
	ValidateCredentials(ctx context.Context) error

	// SupportsFeature indicates whether a specific feature is supported
	SupportsFeature(feature Feature) bool
}

// Feature represents a platform-specific feature
type Feature string

const (
	// FeatureLFS indicates Git LFS support
	FeatureLFS Feature = "lfs"
	// FeatureSubmodules indicates Git submodules support
	FeatureSubmodules Feature = "submodules"
	// FeatureWiki indicates wiki support
	FeatureWiki Feature = "wiki"
	// FeaturePages indicates pages/sites support
	FeaturePages Feature = "pages"
	// FeatureActions indicates CI/CD workflow support
	FeatureActions Feature = "actions"
	// FeatureDiscussions indicates discussions/forums support
	FeatureDiscussions Feature = "discussions"
)

// ValidateCloneURL validates that a clone URL is safe to use in git commands
// This prevents command injection attacks by ensuring the URL is well-formed
// and doesn't contain dangerous characters.
func ValidateCloneURL(cloneURL string) error {
	if cloneURL == "" {
		return fmt.Errorf("clone URL cannot be empty")
	}

	// Parse URL to ensure it's valid
	parsedURL, err := url.Parse(cloneURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	// Only allow https and ssh protocols for git operations
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "https" && scheme != "http" && scheme != "ssh" && scheme != "git" {
		return fmt.Errorf("unsupported URL scheme: %s (only https, http, ssh, git allowed)", scheme)
	}

	// Ensure the URL has a host
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	// Check for potentially dangerous characters that could be used for injection
	// Even though we use exec.CommandContext with separate args, validate the URL
	dangerousChars := []string{"\n", "\r", "\x00", "`", "$", ";", "|", "&", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(cloneURL, char) {
			return fmt.Errorf("URL contains potentially dangerous character")
		}
	}

	return nil
}

// ValidateDestPath validates that a destination path is safe to use in git commands
// This prevents command injection and directory traversal attacks.
func ValidateDestPath(destPath string) error {
	if destPath == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(destPath)

	// Check for null bytes and other dangerous characters
	dangerousChars := []string{"\n", "\r", "\x00", "`", "$", ";", "|", "&", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(destPath, char) {
			return fmt.Errorf("path contains potentially dangerous character")
		}
	}

	// Ensure the path doesn't try to escape using absolute path tricks
	// This is a safety check to prevent directory traversal
	if strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path cannot start with '..'")
	}

	return nil
}

// sanitizeGitError removes token from error messages to prevent credential leakage
func sanitizeGitError(errMsg, token string) string {
	if token == "" {
		return errMsg
	}
	// Replace token with [REDACTED]
	return strings.ReplaceAll(errMsg, token, "[REDACTED]")
}

// ValidateRepoPath validates that a repository path is safe to use for file operations
// This prevents path traversal attacks and ensures paths are within expected boundaries
func ValidateRepoPath(repoPath string) error {
	if repoPath == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(repoPath)

	// Check for dangerous characters
	dangerousChars := []string{"\n", "\r", "\x00"}
	for _, char := range dangerousChars {
		if strings.Contains(repoPath, char) {
			return fmt.Errorf("path contains dangerous character")
		}
	}

	// Ensure the cleaned path doesn't escape using path traversal
	// Check if the path tries to go up beyond the root
	if strings.Contains(cleanPath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path contains directory traversal")
	}

	// For absolute paths, ensure they are truly absolute and normalized
	if filepath.IsAbs(repoPath) {
		if repoPath != cleanPath {
			return fmt.Errorf("path is not normalized")
		}
	}

	return nil
}
