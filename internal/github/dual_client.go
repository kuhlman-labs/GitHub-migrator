package github

import (
	"fmt"
	"log/slog"
)

// DualClient manages two GitHub clients: one for migrations (PAT) and one for API operations (App or PAT)
//
// Architecture:
//
// Per GitHub's migration API documentation, migration operations require Personal Access Tokens (PAT).
// However, GitHub Apps provide higher rate limits and better security for non-migration operations
// like repository discovery, profiling, and general API access.
//
// This dual-client architecture solves this by:
//   - PAT Client (required): Always used for migration operations (StartMigration, MigrationStatus, etc.)
//   - App Client (optional): Used for all other operations when configured, otherwise falls back to PAT
//
// Usage:
//
//	// Create with PAT only (App auth is optional)
//	dc, err := github.NewDualClient(github.DualClientConfig{
//	    PATConfig: patConfig,
//	    Logger: logger,
//	})
//
//	// Use migration client for migration operations
//	migClient := dc.MigrationClient() // Always returns PAT client
//
//	// Use API client for discovery and other operations
//	apiClient := dc.APIClient() // Returns App client if configured, otherwise PAT client
type DualClient struct {
	patClient *Client // Required: Used for migration operations
	appClient *Client // Optional: Used for non-migration operations if configured
	logger    *slog.Logger
}

// DualClientConfig configures a dual client
type DualClientConfig struct {
	// PAT configuration (required)
	PATConfig ClientConfig

	// App configuration (optional)
	// If provided, creates a second client for non-migration operations
	AppConfig *ClientConfig

	Logger *slog.Logger
}

// NewDualClient creates a new dual client with PAT and optionally App authentication
func NewDualClient(cfg DualClientConfig) (*DualClient, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// PAT client is always required
	if cfg.PATConfig.Token == "" {
		return nil, fmt.Errorf("PAT token is required")
	}

	cfg.Logger.Info("Initializing GitHub PAT client")
	patClient, err := NewClient(cfg.PATConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PAT client: %w", err)
	}

	dc := &DualClient{
		patClient: patClient,
		logger:    cfg.Logger,
	}

	// Optionally create App client if credentials are provided
	if cfg.AppConfig != nil &&
		cfg.AppConfig.AppID > 0 &&
		cfg.AppConfig.AppPrivateKey != "" &&
		cfg.AppConfig.AppInstallationID > 0 {

		cfg.Logger.Info("Initializing GitHub App client",
			"app_id", cfg.AppConfig.AppID,
			"installation_id", cfg.AppConfig.AppInstallationID)

		appClient, err := NewClient(*cfg.AppConfig)
		if err != nil {
			// Log warning but don't fail - fall back to PAT for all operations
			cfg.Logger.Warn("Failed to create GitHub App client, falling back to PAT for all operations",
				"error", err)
		} else {
			dc.appClient = appClient
			cfg.Logger.Info("GitHub App client initialized successfully")
		}
	} else {
		cfg.Logger.Info("No GitHub App credentials provided, using PAT for all operations")
	}

	return dc, nil
}

// MigrationClient returns the client to use for migration operations
// Always returns the PAT client per GitHub's migration API requirements
func (dc *DualClient) MigrationClient() *Client {
	dc.logger.Debug("Using PAT client for migration operation")
	return dc.patClient
}

// APIClient returns the client to use for general API operations
// Returns App client if configured, otherwise returns PAT client
func (dc *DualClient) APIClient() *Client {
	if dc.appClient != nil {
		dc.logger.Debug("Using App client for API operation")
		return dc.appClient
	}
	dc.logger.Debug("Using PAT client for API operation (no App client configured)")
	return dc.patClient
}

// HasAppClient returns true if an App client is configured
func (dc *DualClient) HasAppClient() bool {
	return dc.appClient != nil
}

// BaseURL returns the base URL from the PAT client
func (dc *DualClient) BaseURL() string {
	return dc.patClient.BaseURL()
}

// Token returns the PAT token
func (dc *DualClient) Token() string {
	return dc.patClient.Token()
}
