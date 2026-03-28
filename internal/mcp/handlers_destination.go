package mcp

import (
	"context"
	"fmt"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleGetDestination implements the get_destination tool.
// Returns the current destination configuration with sensitive fields masked.
func (s *Server) handleGetDestination(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	settings, err := s.db.GetSettings(ctx)
	if err != nil {
		s.logger.Error("Failed to get settings", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get destination settings: %v", err)), nil
	}

	configured := settings.HasDestination()

	msg := "Destination is configured"
	if !configured {
		msg = "Destination is not configured. Use configure_destination to set it up."
	}

	output := GetDestinationOutput{
		BaseURL:          settings.DestinationBaseURL,
		TokenConfigured:  settings.DestinationToken != nil && *settings.DestinationToken != "",
		EnterpriseSlug:   settings.DestinationEnterpriseSlug,
		AppID:            settings.DestinationAppID,
		AppKeyConfigured: settings.DestinationAppPrivateKey != nil && *settings.DestinationAppPrivateKey != "",
		InstallationID:   settings.DestinationAppInstallationID,
		Configured:       configured,
		Message:          msg,
	}

	return s.jsonResult(output)
}

// handleConfigureDestination implements the configure_destination tool.
// Updates the migration destination settings.
func (s *Server) handleConfigureDestination(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	baseURL := req.GetString("base_url", "")
	token := req.GetString("token", "")
	enterpriseSlug := req.GetString("enterprise_slug", "")

	// At least one field must be provided
	if baseURL == "" && token == "" && enterpriseSlug == "" {
		return mcp.NewToolResultError("At least one of base_url, token, or enterprise_slug is required"), nil
	}

	// Build update request with only the provided fields
	updateReq := &models.UpdateSettingsRequest{}
	if baseURL != "" {
		updateReq.DestinationBaseURL = &baseURL
	}
	if token != "" {
		updateReq.DestinationToken = &token
	}
	if enterpriseSlug != "" {
		updateReq.DestinationEnterpriseSlug = &enterpriseSlug
	}

	settings, err := s.db.UpdateSettings(ctx, updateReq)
	if err != nil {
		s.logger.Error("Failed to update destination settings", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update destination: %v", err)), nil
	}

	// Trigger config reload if configSvc is available
	if s.configSvc != nil {
		if reloadErr := s.configSvc.Reload(); reloadErr != nil {
			s.logger.Warn("Failed to reload config after destination update", "error", reloadErr)
		}
	}

	s.logger.Info("Destination configured via MCP",
		"base_url", settings.DestinationBaseURL,
		"configured", settings.HasDestination())

	output := ConfigureDestinationOutput{
		BaseURL:        settings.DestinationBaseURL,
		EnterpriseSlug: settings.DestinationEnterpriseSlug,
		Configured:     settings.HasDestination(),
		Message:        fmt.Sprintf("Destination updated (base_url: %s, configured: %t)", settings.DestinationBaseURL, settings.HasDestination()),
	}

	return s.jsonResult(output)
}
