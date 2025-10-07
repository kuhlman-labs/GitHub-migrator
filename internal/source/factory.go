package source

import (
	"fmt"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/config"
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
// This is similar to NewProviderFromConfig but uses DestinationConfig
func NewDestinationProviderFromConfig(cfg config.DestinationConfig) (Provider, error) {
	providerType := strings.ToLower(cfg.Type)

	switch providerType {
	case "github":
		return NewGitHubProvider(cfg.BaseURL, cfg.Token)

	case "gitlab":
		return NewGitLabProvider(cfg.BaseURL, cfg.Token)

	case "azuredevops", "ado":
		// For destination, we might not have organization in the config
		// This needs to be handled differently for ADO
		return nil, fmt.Errorf("Azure DevOps as destination not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported provider type: %s (supported: github, gitlab)", cfg.Type)
	}
}
