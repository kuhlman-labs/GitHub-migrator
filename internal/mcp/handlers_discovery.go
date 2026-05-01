package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/discovery"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleCreateSource implements the create_source tool.
// Creates a new migration source for discovering repositories.
func (s *Server) handleCreateSource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	sourceType := req.GetString("type", "github")
	baseURL := req.GetString("base_url", "")
	token := req.GetString("token", "")
	if token == "" {
		return mcp.NewToolResultError("token is required"), nil
	}

	// Default base_url for GitHub
	if baseURL == "" && sourceType == models.SourceConfigTypeGitHub {
		baseURL = "https://api.github.com"
	}
	if baseURL == "" {
		return mcp.NewToolResultError("base_url is required for non-GitHub sources"), nil
	}

	src := &models.Source{
		Name:     name,
		Type:     sourceType,
		BaseURL:  baseURL,
		Token:    token,
		IsActive: true,
	}

	if org := req.GetString("organization", ""); org != "" {
		src.Organization = &org
	}
	if slug := req.GetString("enterprise_slug", ""); slug != "" {
		src.EnterpriseSlug = &slug
	}

	if err := s.db.CreateSource(ctx, src); err != nil {
		s.logger.Error("Failed to create source", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create source: %v", err)), nil
	}

	s.logger.Info("Source created via MCP", "source_id", src.ID, "name", src.Name)

	output := CreateSourceOutput{
		Source:  src.ToResponse(),
		Message: fmt.Sprintf("Source '%s' created (ID: %d). You can now run start_discovery with source_id=%d.", src.Name, src.ID, src.ID),
	}

	return s.jsonResult(output)
}

// handleListSources implements the list_sources tool.
// Returns all configured migration sources, optionally filtered to active-only.
func (s *Server) handleListSources(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	activeOnly := req.GetBool("active_only", false)

	var sources []*models.Source
	var err error

	if activeOnly {
		sources, err = s.db.ListActiveSources(ctx)
	} else {
		sources, err = s.db.ListSources(ctx)
	}
	if err != nil {
		s.logger.Error("Failed to list sources", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list sources: %v", err)), nil
	}

	// Convert to safe response format (masks tokens)
	responses := models.SourcesToResponses(sources)

	output := ListSourcesOutput{
		Sources: responses,
		Count:   len(responses),
		Message: fmt.Sprintf("Found %d source(s)", len(responses)),
	}

	return s.jsonResult(output)
}

// handleGetSource implements the get_source tool.
// Returns details for a single migration source by ID.
func (s *Server) handleGetSource(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceID := req.GetInt("source_id", 0)
	if sourceID == 0 {
		return mcp.NewToolResultError("source_id is required"), nil
	}

	src, err := s.db.GetSource(ctx, int64(sourceID))
	if err != nil {
		s.logger.Error("Failed to get source", "error", err, "source_id", sourceID)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get source %d: %v", sourceID, err)), nil
	}
	if src == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Source %d not found", sourceID)), nil
	}

	output := GetSourceOutput{
		Source:  src.ToResponse(),
		Message: fmt.Sprintf("Source '%s' (ID: %d)", src.Name, src.ID),
	}

	return s.jsonResult(output)
}

// resolveSource resolves a source from the provided parameters.
// It tries, in order: explicit source_id, enterprise_slug lookup, single active source fallback.
func (s *Server) resolveSource(ctx context.Context, sourceID int64, enterpriseSlug string) (*models.Source, error) {
	if sourceID > 0 {
		src, err := s.db.GetSource(ctx, sourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get source %d: %w", sourceID, err)
		}
		if src == nil {
			return nil, fmt.Errorf("source %d not found", sourceID)
		}
		return src, nil
	}

	if enterpriseSlug != "" {
		src, err := s.db.GetSourceByEnterpriseSlug(ctx, enterpriseSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to find source for enterprise '%s': %w", enterpriseSlug, err)
		}
		if src == nil {
			return nil, fmt.Errorf("no source found for enterprise '%s'. Create one first with create_source", enterpriseSlug)
		}
		return src, nil
	}

	// Fallback: use the only active source
	activeSources, err := s.db.ListActiveSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	switch len(activeSources) {
	case 0:
		return nil, fmt.Errorf("no active sources configured. Create one first with create_source")
	case 1:
		return activeSources[0], nil
	default:
		return nil, fmt.Errorf("multiple active sources found (%d). Specify source_id or enterprise_slug to disambiguate", len(activeSources))
	}
}

// resolveDiscoveryTarget determines the organization or enterprise_slug for discovery.
// Returns (organization, enterpriseSlug, error).
func resolveDiscoveryTarget(src *models.Source, organization, enterpriseSlug string) (string, string, error) {
	if organization != "" || enterpriseSlug != "" {
		return organization, enterpriseSlug, nil
	}
	if src.EnterpriseSlug != nil && *src.EnterpriseSlug != "" {
		return "", *src.EnterpriseSlug, nil
	}
	if src.Organization != nil && *src.Organization != "" {
		return *src.Organization, "", nil
	}
	return "", "", fmt.Errorf("either organization or enterprise_slug is required (none configured on source either)")
}

// handleStartDiscovery implements the start_discovery tool.
// Starts repository discovery for a source, targeting either an organization or enterprise.
// If source_id is not provided, it will auto-resolve from enterprise_slug or use the only active source.
func (s *Server) handleStartDiscovery(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceID := int64(req.GetInt("source_id", 0))
	organization := req.GetString("organization", "")
	enterpriseSlug := req.GetString("enterprise_slug", "")

	// Validate mutually exclusive params
	if organization != "" && enterpriseSlug != "" {
		return mcp.NewToolResultError("Cannot specify both organization and enterprise_slug"), nil
	}

	// Resolve source
	src, err := s.resolveSource(ctx, sourceID, enterpriseSlug)
	if err != nil {
		s.logger.Error("Failed to resolve source", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	sourceID = src.ID

	// Resolve discovery target
	organization, enterpriseSlug, err = resolveDiscoveryTarget(src, organization, enterpriseSlug)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate source type
	if src.Type != models.SourceConfigTypeGitHub {
		return mcp.NewToolResultError(fmt.Sprintf("Source %d is not a GitHub source (type: %s)", sourceID, src.Type)), nil
	}

	// Check for active discovery
	activeProgress, err := s.db.GetActiveDiscoveryProgress()
	if err != nil {
		s.logger.Error("Failed to check for active discovery", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to check discovery status: %v", err)), nil
	}
	if activeProgress != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Discovery already in progress (target: %s, started: %s)",
			activeProgress.Target, activeProgress.StartedAt.Format(time.RFC3339))), nil
	}

	// Build GitHub client config
	clientConfig := github.ClientConfig{
		BaseURL:     src.BaseURL,
		Token:       src.Token,
		Timeout:     120 * time.Second,
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      s.logger,
	}

	// Add App credentials if configured
	if src.HasAppAuth() {
		clientConfig.AppID = *src.AppID
		clientConfig.AppPrivateKey = *src.AppPrivateKey
		if src.AppInstallationID != nil && *src.AppInstallationID > 0 {
			clientConfig.AppInstallationID = *src.AppInstallationID
		}
	}

	// Create GitHub client
	client, err := github.NewClient(clientConfig)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create GitHub client: %v", err)), nil
	}

	// Create source provider
	sourceProvider, err := source.NewGitHubProvider(src.BaseURL, src.Token)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create source provider: %v", err)), nil
	}

	// Create collector
	collector := discovery.NewCollector(client, s.db, s.logger, sourceProvider)
	collector.SetSourceID(&sourceID)

	// Set base config for App auth if available
	if src.HasAppAuth() {
		collector.WithBaseConfig(clientConfig)
	}

	// Determine discovery type and target
	var discoveryType, target string
	if enterpriseSlug != "" {
		discoveryType = models.DiscoveryTypeEnterprise
		target = enterpriseSlug
	} else {
		discoveryType = models.DiscoveryTypeOrganization
		target = organization
	}

	// Create progress record
	progress := &models.DiscoveryProgress{
		DiscoveryType: discoveryType,
		Target:        target,
		TotalOrgs:     1,
	}

	if err := s.db.CreateDiscoveryProgress(progress); err != nil {
		if errors.Is(err, storage.ErrDiscoveryInProgress) {
			return mcp.NewToolResultError(fmt.Sprintf("Discovery already in progress: %v", err)), nil
		}
		s.logger.Error("Failed to create discovery progress", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize discovery: %v", err)), nil
	}

	// Create progress tracker
	progressTracker := discovery.NewDBProgressTracker(s.db, s.logger, progress)
	collector.SetProgressTracker(progressTracker)

	// Launch discovery goroutine with 72-hour timeout
	// #nosec G118 -- cancel is called in the goroutine's deferred cleanup
	discoveryCtx, cancel := context.WithTimeout(context.Background(), 72*time.Hour)

	// Register cancel function
	s.discoveryMu.Lock()
	s.discoveryCancel[progress.ID] = cancel
	s.discoveryMu.Unlock()

	go func() {
		defer func() {
			s.discoveryMu.Lock()
			delete(s.discoveryCancel, progress.ID)
			s.discoveryMu.Unlock()
			cancel()
			progressTracker.Flush()
		}()

		var discoverErr error
		if enterpriseSlug != "" {
			discoverErr = collector.DiscoverEnterpriseRepositories(discoveryCtx, enterpriseSlug)
		} else {
			discoverErr = collector.DiscoverRepositories(discoveryCtx, organization)
		}

		if discoverErr != nil {
			if discoveryCtx.Err() == context.Canceled {
				s.logger.Info("Discovery cancelled", "type", discoveryType, "target", target)
				if dbErr := s.db.MarkDiscoveryCancelled(progress.ID); dbErr != nil {
					s.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
				}
				return
			}

			s.logger.Error("Discovery failed", "error", discoverErr, "type", discoveryType, "target", target)
			if dbErr := s.db.MarkDiscoveryFailed(progress.ID, discoverErr.Error()); dbErr != nil {
				s.logger.Error("Failed to mark discovery as failed", "error", dbErr)
			}
		} else {
			if discoveryCtx.Err() == context.Canceled {
				s.logger.Info("Discovery cancelled", "type", discoveryType, "target", target)
				if dbErr := s.db.MarkDiscoveryCancelled(progress.ID); dbErr != nil {
					s.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
				}
				return
			}

			if dbErr := s.db.MarkDiscoveryComplete(progress.ID); dbErr != nil {
				s.logger.Error("Failed to mark discovery as complete", "error", dbErr)
			}

			// Update source repository count
			if err := s.db.UpdateSourceRepositoryCount(discoveryCtx, sourceID); err != nil {
				s.logger.Error("Failed to update source repository count",
					"error", err, "source_id", sourceID)
			} else {
				s.logger.Info("Updated source repository count after discovery",
					"source_id", sourceID)
			}
		}
	}()

	output := StartDiscoveryOutput{
		ProgressID:    progress.ID,
		SourceID:      sourceID,
		DiscoveryType: discoveryType,
		Target:        target,
		Status:        models.DiscoveryStatusInProgress,
		Message:       fmt.Sprintf("Discovery started for %s '%s' using source '%s'", discoveryType, target, src.Name),
		Hint:          "Use get_discovery_progress to track status.",
	}

	return s.jsonResult(output)
}

// handleGetDiscoveryProgress implements the get_discovery_progress tool.
// Returns the latest discovery progress record with rate limit and completion details.
func (s *Server) handleGetDiscoveryProgress(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	progress, err := s.db.GetLatestDiscoveryProgress()
	if err != nil {
		s.logger.Error("Failed to get discovery progress", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get discovery progress: %v", err)), nil
	}

	if progress == nil {
		output := GetDiscoveryProgressOutput{
			Status:  "none",
			Message: "No discovery run yet",
		}
		return s.jsonResult(output)
	}

	output := GetDiscoveryProgressOutput{
		ProgressID:      progress.ID,
		DiscoveryType:   progress.DiscoveryType,
		Target:          progress.Target,
		Status:          progress.Status,
		Phase:           progress.Phase,
		StartedAt:       progress.StartedAt.Format(time.RFC3339),
		TotalOrgs:       progress.TotalOrgs,
		ProcessedOrgs:   progress.ProcessedOrgs,
		CurrentOrg:      progress.CurrentOrg,
		TotalRepos:      progress.TotalRepos,
		ProcessedRepos:  progress.ProcessedRepos,
		ErrorCount:      progress.ErrorCount,
		PercentComplete: progress.PercentComplete(),
		Message:         fmt.Sprintf("Discovery %s: %s '%s'", progress.Status, progress.DiscoveryType, progress.Target),
	}

	if progress.CompletedAt != nil {
		completedAt := progress.CompletedAt.Format(time.RFC3339)
		output.CompletedAt = &completedAt
	}
	if progress.LastError != nil && *progress.LastError != "" {
		output.LastError = progress.LastError
	}

	// Surface rate limit info when waiting
	if progress.Phase == models.PhaseWaitingForRateLimit && progress.RateLimitResetAt != nil {
		resetStr := progress.RateLimitResetAt.Format(time.RFC3339)
		output.RateLimitResetAt = &resetStr
		waitSecs := int(time.Until(*progress.RateLimitResetAt).Seconds())
		if waitSecs < 0 {
			waitSecs = 0
		}
		output.RateLimitWaitSecs = &waitSecs
	}

	// Add post-discovery summary when completed
	if progress.Status == models.DiscoveryStatusCompleted {
		summary, nextSteps := s.buildDiscoverySummary(ctx, progress)
		output.Summary = &summary
		output.NextSteps = &nextSteps
	}

	return s.jsonResult(output)
}

// buildDiscoverySummary generates a summary and next-step hints for completed discovery.
func (s *Server) buildDiscoverySummary(ctx context.Context, progress *models.DiscoveryProgress) (string, string) {
	summary := fmt.Sprintf("%d repositories discovered across %d organizations.",
		progress.TotalRepos, progress.TotalOrgs)

	if progress.CompletedAt != nil {
		duration := progress.CompletedAt.Sub(progress.StartedAt).Round(time.Second)
		summary += fmt.Sprintf(" Completed in %s.", duration)
	}
	if progress.ErrorCount > 0 {
		summary += fmt.Sprintf(" %d errors encountered.", progress.ErrorCount)
	}

	// Try to get complexity breakdown using the same query as the UI
	dist, err := s.db.GetComplexityDistribution(ctx, "", "", "", nil)
	if err == nil && len(dist) > 0 {
		counts := map[string]int{}
		for _, d := range dist {
			counts[d.Category] = d.Count
		}
		summary += fmt.Sprintf(" Complexity: %d simple, %d medium, %d complex, %d very complex.",
			counts[models.ComplexitySimple], counts[models.ComplexityMedium],
			counts[models.ComplexityComplex], counts["very_complex"])
	}

	nextSteps := "Next steps: Run analyze_repositories to explore discovered repos, " +
		"find_pilot_candidates to identify easy wins, or get_migration_readiness to check readiness."

	return summary, nextSteps
}

// handleCancelDiscovery implements the cancel_discovery tool.
// Cancels the currently active discovery operation.
func (s *Server) handleCancelDiscovery(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Find active discovery
	activeProgress, err := s.db.GetActiveDiscoveryProgress()
	if err != nil {
		s.logger.Error("Failed to check for active discovery", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to check discovery status: %v", err)), nil
	}
	if activeProgress == nil {
		return mcp.NewToolResultError("No active discovery to cancel"), nil
	}

	// Look up cancel function
	s.discoveryMu.RLock()
	cancel, exists := s.discoveryCancel[activeProgress.ID]
	s.discoveryMu.RUnlock()

	if !exists {
		return mcp.NewToolResultError(fmt.Sprintf("No cancel function found for discovery %d — it may have been started by another process", activeProgress.ID)), nil
	}

	// Cancel the discovery
	cancel()

	// Update phase to cancelling
	if err := s.db.UpdateDiscoveryPhase(activeProgress.ID, models.PhaseCancelling); err != nil {
		s.logger.Error("Failed to update discovery phase", "error", err)
	}

	output := CancelDiscoveryOutput{
		ProgressID: activeProgress.ID,
		Target:     activeProgress.Target,
		Success:    true,
		Message:    fmt.Sprintf("Cancellation requested for discovery of '%s'", activeProgress.Target),
	}

	return s.jsonResult(output)
}
