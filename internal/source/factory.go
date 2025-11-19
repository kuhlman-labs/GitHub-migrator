package source

import (
	"fmt"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

// NewProviderFromConfig creates a source provider based on configuration
func NewProviderFromConfig(cfg config.SourceConfig) (Provider, error) {
	providerType := strings.ToLower(cfg.Type)

	switch providerType {
	case "github":
		return NewGitHubProvider(cfg.BaseURL, cfg.Token)

	case "gitlab":
		return NewGitLabProvider(cfg.BaseURL, cfg.Token)

	case "azuredevops", "ado":
		if cfg.Organization == "" {
			return nil, fmt.Errorf("organization is required for Azure DevOps provider")
		}
		return NewAzureDevOpsProvider(cfg.Organization, cfg.Token, cfg.Username)

	default:
		return nil, fmt.Errorf("unsupported provider type: %s (supported: github, gitlab, azuredevops)", cfg.Type)
	}
}

// NewDestinationProviderFromConfig creates a destination provider based on configuration
// Currently only GitHub is supported as a destination provider
func NewDestinationProviderFromConfig(cfg config.DestinationConfig) (Provider, error) {
	providerType := strings.ToLower(cfg.Type)

	if providerType != "github" {
		return nil, fmt.Errorf("unsupported destination provider type: %s (only github is supported as destination)", cfg.Type)
	}

	return NewGitHubProvider(cfg.BaseURL, cfg.Token)
}
