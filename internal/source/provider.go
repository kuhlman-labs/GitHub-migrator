package source

import (
	"context"
	"errors"
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
func DefaultCloneOptions() CloneOptions {
	return CloneOptions{
		Shallow:           true,  // Faster for analysis
		Bare:              false, // Need working tree for file analysis
		IncludeLFS:        false, // Don't fetch LFS content during discovery
		IncludeSubmodules: false, // Don't clone submodules during discovery
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
