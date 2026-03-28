package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerResources registers MCP resources that provide contextual data to agents
func (s *Server) registerResources() {
	// migration://dashboard - Current migration dashboard state
	s.mcpServer.AddResource(
		mcp.NewResource(
			"migration://dashboard",
			"Migration Dashboard",
			mcp.WithResourceDescription("Current migration dashboard state including active discoveries, batch progress, recent completions, and alerts"),
			mcp.WithMIMEType("application/json"),
		),
		s.handleDashboardResource,
	)

	// migration://sources - Overview of all configured sources
	s.mcpServer.AddResource(
		mcp.NewResource(
			"migration://sources",
			"Migration Sources",
			mcp.WithResourceDescription("Overview of all configured migration sources with repository counts and sync status"),
			mcp.WithMIMEType("application/json"),
		),
		s.handleSourcesResource,
	)

	// migration://readiness - Overall migration readiness summary
	s.mcpServer.AddResource(
		mcp.NewResource(
			"migration://readiness",
			"Migration Readiness",
			mcp.WithResourceDescription("Overall migration readiness including user/team mapping completeness and blocker counts"),
			mcp.WithMIMEType("application/json"),
		),
		s.handleReadinessResource,
	)

	s.logger.Info("Registered MCP resources", "count", 3)
}

// handleDashboardResource provides the current migration dashboard state
func (s *Server) handleDashboardResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	dashboard := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Discovery status
	progress, err := s.db.GetLatestDiscoveryProgress()
	if err == nil && progress != nil {
		dashboard["discovery"] = map[string]any{
			"status":          progress.Status,
			"target":          progress.Target,
			"type":            progress.DiscoveryType,
			"processed_repos": progress.ProcessedRepos,
			"total_repos":     progress.TotalRepos,
		}
	} else {
		dashboard["discovery"] = map[string]any{"status": "none"}
	}

	// Repository stats
	stats, err := s.db.GetRepositoryStatsByStatus(ctx)
	if err == nil {
		total := 0
		for status, count := range stats {
			if status != string(models.StatusWontMigrate) {
				total += count
			}
		}
		migrated := stats[string(models.StatusComplete)] + stats[string(models.StatusMigrationComplete)]
		dashboard["repositories"] = map[string]any{
			"total":        total,
			"migrated":     migrated,
			"status":       stats,
			"progress_pct": safePercent(migrated, total),
		}
	}

	// Active batches
	batches, err := s.db.ListBatches(ctx)
	if err == nil {
		activeBatches := []map[string]any{}
		for _, b := range batches {
			if b.Status == models.BatchStatusInProgress || b.Status == "scheduled" {
				activeBatches = append(activeBatches, map[string]any{
					"id":     b.ID,
					"name":   b.Name,
					"status": b.Status,
					"repos":  b.RepositoryCount,
				})
			}
		}
		dashboard["active_batches"] = activeBatches
	}

	data, _ := json.MarshalIndent(dashboard, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: req.Params.URI, MIMEType: "application/json", Text: string(data)},
	}, nil
}

// handleSourcesResource provides an overview of configured sources
func (s *Server) handleSourcesResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	sources, err := s.db.ListSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}

	sourceList := []map[string]any{}
	for _, src := range sources {
		sourceList = append(sourceList, map[string]any{
			"id":               src.ID,
			"name":             src.Name,
			"type":             src.Type,
			"base_url":         src.BaseURL,
			"is_active":        src.IsActive,
			"repository_count": src.RepositoryCount,
			"last_sync_at":     formatTimeForResource(src.LastSyncAt),
			"has_app_auth":     src.HasAppAuth(),
		})
	}

	data, _ := json.MarshalIndent(map[string]any{
		"sources": sourceList,
		"count":   len(sourceList),
	}, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: req.Params.URI, MIMEType: "application/json", Text: string(data)},
	}, nil
}

// handleReadinessResource provides overall migration readiness
func (s *Server) handleReadinessResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	readiness := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// User mapping stats
	userStats, err := s.db.GetUserMappingStats(ctx, "")
	if err == nil {
		readiness["user_mappings"] = userStats
	}

	// Team mapping stats
	teamStats, err := s.db.GetTeamMappingStats(ctx, "")
	if err == nil {
		readiness["team_mappings"] = teamStats
	}

	// Blocker count — load validation details and count actual blockers
	blockerFilters := map[string]any{"include_details": true}
	blockerRepos, err := s.db.ListRepositories(ctx, blockerFilters)
	if err == nil {
		readiness["blocker_count"] = countReposWithBlockers(blockerRepos)
	}

	// Repos by key status
	pendingCount, _ := s.db.CountRepositories(ctx, map[string]any{"status": string(models.StatusPending)})
	dryRunCount, _ := s.db.CountRepositories(ctx, map[string]any{"status": string(models.StatusDryRunComplete)})
	readiness["pending_repos"] = pendingCount
	readiness["dry_run_complete_repos"] = dryRunCount

	data, _ := json.MarshalIndent(readiness, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{URI: req.Params.URI, MIMEType: "application/json", Text: string(data)},
	}, nil
}

func safePercent(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom) * 100
}

func formatTimeForResource(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
