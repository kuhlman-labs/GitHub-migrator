package mcp

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleGetExecutiveSummary provides a high-level overview of migration progress
func (s *Server) handleGetExecutiveSummary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	orgFilter := req.GetString("organization", "")
	sourceID := int64(req.GetInt("source_id", 0))

	var sourceIDPtr *int64
	if sourceID > 0 {
		sourceIDPtr = &sourceID
	}

	// Get repository stats by status
	statusStats, err := s.db.GetRepositoryStatsByStatusFiltered(ctx, orgFilter, "", "", sourceIDPtr)
	if err != nil {
		s.logger.Error("Failed to get repository stats", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get repository stats: %v", err)), nil
	}

	// Get migration velocity
	velocity, err := s.db.GetMigrationVelocity(ctx, orgFilter, "", "", sourceIDPtr, 30)
	if err != nil {
		s.logger.Error("Failed to get migration velocity", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get migration velocity: %v", err)), nil
	}

	// Get average migration time
	avgMigrationTime, err := s.db.GetAverageMigrationTime(ctx, orgFilter, "", "", sourceIDPtr)
	if err != nil {
		s.logger.Error("Failed to get average migration time", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get average migration time: %v", err)), nil
	}

	// Get organization stats
	orgStats, err := s.db.GetOrganizationStatsFiltered(ctx, orgFilter, "", "", sourceIDPtr)
	if err != nil {
		s.logger.Error("Failed to get organization stats", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get organization stats: %v", err)), nil
	}

	// Define status groupings
	migratedStatuses := map[string]bool{
		string(models.StatusComplete):          true,
		string(models.StatusMigrationComplete): true,
	}
	failedStatuses := map[string]bool{
		string(models.StatusMigrationFailed): true,
		string(models.StatusDryRunFailed):    true,
		string(models.StatusRolledBack):      true,
	}
	pendingStatuses := map[string]bool{
		string(models.StatusPending):             true,
		string(models.StatusDryRunQueued):        true,
		string(models.StatusDryRunInProgress):    true,
		string(models.StatusDryRunComplete):      true,
		string(models.StatusRemediationRequired): true,
	}
	inProgressStatuses := map[string]bool{
		string(models.StatusPreMigration):       true,
		string(models.StatusArchiveGenerating):  true,
		string(models.StatusQueuedForMigration): true,
		string(models.StatusMigratingContent):   true,
		string(models.StatusPostMigration):      true,
	}

	// Compute counts
	var migratedCount, failedCount, pendingCount, inProgressCount, totalRepos int
	for status, count := range statusStats {
		if status == string(models.StatusWontMigrate) {
			continue // Excluded from total
		}
		totalRepos += count
		if migratedStatuses[status] {
			migratedCount += count
		} else if failedStatuses[status] {
			failedCount += count
		} else if pendingStatuses[status] {
			pendingCount += count
		} else if inProgressStatuses[status] {
			inProgressCount += count
		}
	}

	// Compute success rate
	var successRate float64
	attempted := migratedCount + failedCount
	if attempted > 0 {
		successRate = math.Round(float64(migratedCount)/float64(attempted)*10000) / 100
	}

	// Estimate completion date
	var estimatedCompletion *string
	remaining := pendingCount + inProgressCount + failedCount
	if remaining > 0 && velocity != nil && velocity.ReposPerDay > 0 {
		daysRemaining := math.Ceil(float64(remaining) / velocity.ReposPerDay)
		est := time.Now().AddDate(0, 0, int(daysRemaining)).Format("2006-01-02")
		estimatedCompletion = &est
	}

	// Build organization breakdown
	orgBreakdown := make([]ExecutiveSummaryOrgStats, 0, len(orgStats))
	for _, os := range orgStats {
		orgBreakdown = append(orgBreakdown, ExecutiveSummaryOrgStats{
			Organization:    os.Organization,
			TotalRepos:      os.TotalRepos,
			MigratedCount:   os.MigratedCount,
			FailedCount:     os.FailedCount,
			PendingCount:    os.PendingCount,
			InProgressCount: os.InProgressCount,
			ProgressPercent: os.MigrationProgressPercent,
		})
	}

	output := GetExecutiveSummaryOutput{
		TotalRepositories:   totalRepos,
		MigratedCount:       migratedCount,
		FailedCount:         failedCount,
		PendingCount:        pendingCount,
		InProgressCount:     inProgressCount,
		SuccessRate:         successRate,
		EstimatedCompletion: estimatedCompletion,
		Velocity: ExecutiveSummaryVelocity{
			ReposPerDay:  velocity.ReposPerDay,
			ReposPerWeek: velocity.ReposPerWeek,
		},
		AvgMigrationTimeSeconds: avgMigrationTime,
		StatusBreakdown:         statusStats,
		OrganizationBreakdown:   orgBreakdown,
		Message:                 fmt.Sprintf("Migration summary: %d/%d repositories migrated (%.1f%% success rate), %d pending, %d in progress, %d failed", migratedCount, totalRepos, successRate, pendingCount, inProgressCount, failedCount),
	}

	return s.jsonResult(output)
}

// handleGetMigrationHistory returns migration history records
func (s *Server) handleGetMigrationHistory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName := req.GetString("repository", "")
	limit := req.GetInt("limit", 20)
	statusFilter := req.GetString("status_filter", "")

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if repoName != "" {
		// Get history for a specific repository
		repo, err := s.db.GetRepository(ctx, repoName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %v", err)), nil
		}

		history, err := s.db.GetMigrationHistory(ctx, repo.ID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get migration history: %v", err)), nil
		}

		records := make([]MigrationHistoryRecord, 0, len(history))
		for _, h := range history {
			record := MigrationHistoryRecord{
				ID:              h.ID,
				RepositoryID:    h.RepositoryID,
				RepositoryName:  repoName,
				Status:          h.Status,
				Phase:           h.Phase,
				StartedAt:       h.StartedAt.Format(time.RFC3339),
				DurationSeconds: h.DurationSeconds,
			}
			if h.Message != nil {
				record.Message = *h.Message
			}
			if h.ErrorMessage != nil {
				record.ErrorMessage = *h.ErrorMessage
			}
			if h.CompletedAt != nil {
				t := h.CompletedAt.Format(time.RFC3339)
				record.CompletedAt = &t
			}
			records = append(records, record)
		}

		// Apply limit
		if len(records) > limit {
			records = records[:limit]
		}

		output := GetMigrationHistoryOutput{
			Repository: repoName,
			Records:    records,
			Count:      len(records),
			Message:    fmt.Sprintf("Found %d migration history records for %s", len(records), repoName),
		}
		return s.jsonResult(output)
	}

	// Get all completed migrations
	completedMigrations, err := s.db.GetCompletedMigrations(ctx, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get completed migrations: %v", err)), nil
	}

	records := make([]MigrationHistoryRecord, 0, len(completedMigrations))
	for _, cm := range completedMigrations {
		// Apply status filter
		if statusFilter == "completed" && cm.Status != string(models.StatusComplete) && cm.Status != string(models.StatusMigrationComplete) {
			continue
		}
		if statusFilter == "failed" && cm.Status != string(models.StatusMigrationFailed) && cm.Status != string(models.StatusDryRunFailed) && cm.Status != string(models.StatusRolledBack) {
			continue
		}

		record := MigrationHistoryRecord{
			ID:             cm.ID,
			RepositoryName: cm.FullName,
			Status:         cm.Status,
		}
		if cm.MigratedAt != nil {
			t := cm.MigratedAt.Format(time.RFC3339)
			record.CompletedAt = &t
		}
		if cm.StartedAt != nil {
			record.StartedAt = cm.StartedAt.Format(time.RFC3339)
		}
		record.DurationSeconds = cm.DurationSeconds

		records = append(records, record)
	}

	// Apply limit
	if len(records) > limit {
		records = records[:limit]
	}

	filterMsg := ""
	if statusFilter != "" {
		filterMsg = fmt.Sprintf(" (filtered by %s)", statusFilter)
	}

	output := GetMigrationHistoryOutput{
		Records: records,
		Count:   len(records),
		Message: fmt.Sprintf("Found %d migration records%s", len(records), filterMsg),
	}
	return s.jsonResult(output)
}

// handleGetMigrationLogs returns migration log entries for a repository
func (s *Server) handleGetMigrationLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName, err := req.RequireString("repository")
	if err != nil {
		return mcp.NewToolResultError("repository parameter is required"), nil
	}

	level := req.GetString("level", "")
	phase := req.GetString("phase", "")
	limit := req.GetInt("limit", 50)

	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	repo, err := s.db.GetRepository(ctx, repoName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %v", err)), nil
	}

	logs, err := s.db.GetMigrationLogs(ctx, repo.ID, level, phase, limit, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get migration logs: %v", err)), nil
	}

	entries := make([]MigrationLogEntry, 0, len(logs))
	for _, l := range logs {
		entry := MigrationLogEntry{
			ID:           l.ID,
			RepositoryID: l.RepositoryID,
			Level:        l.Level,
			Phase:        l.Phase,
			Operation:    l.Operation,
			Message:      l.Message,
			Timestamp:    l.Timestamp.Format(time.RFC3339),
		}
		if l.Details != nil {
			entry.Details = *l.Details
		}
		entries = append(entries, entry)
	}

	output := GetMigrationLogsOutput{
		Repository: repoName,
		Entries:    entries,
		Count:      len(entries),
		Message:    fmt.Sprintf("Found %d log entries for %s", len(entries), repoName),
	}
	return s.jsonResult(output)
}

// handleGetActionItems aggregates pending work items across domains
func (s *Server) handleGetActionItems(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var items []ActionItem

	// Check for failed repositories needing retry
	failedRepos, err := s.db.ListRepositories(ctx, map[string]any{"status": "migration_failed"})
	if err != nil {
		s.logger.Error("Failed to list failed repositories", "error", err)
	} else if len(failedRepos) > 0 {
		repoNames := make([]string, 0, len(failedRepos))
		for _, r := range failedRepos {
			repoNames = append(repoNames, r.FullName)
		}
		items = append(items, ActionItem{
			Severity:    "critical",
			Category:    "failed_migrations",
			Title:       fmt.Sprintf("%d repositories failed migration", len(failedRepos)),
			Description: "These repositories encountered errors during migration and need investigation or retry.",
			Count:       len(failedRepos),
			Details:     repoNames,
			Action:      "Review error logs with get_migration_logs, fix issues, then retry with start_migration",
		})
	}

	// Check for repos needing remediation
	remediationRepos, err := s.db.ListRepositories(ctx, map[string]any{"status": "remediation_required"})
	if err != nil {
		s.logger.Error("Failed to list remediation repositories", "error", err)
	} else if len(remediationRepos) > 0 {
		repoNames := make([]string, 0, len(remediationRepos))
		for _, r := range remediationRepos {
			repoNames = append(repoNames, r.FullName)
		}
		items = append(items, ActionItem{
			Severity:    "warning",
			Category:    "remediation_required",
			Title:       fmt.Sprintf("%d repositories need remediation", len(remediationRepos)),
			Description: "These repositories have issues that must be resolved before migration can proceed.",
			Count:       len(remediationRepos),
			Details:     repoNames,
			Action:      "Review repository details with get_complexity_breakdown and resolve blockers",
		})
	}

	// Check user mapping stats
	userStats, err := s.db.GetUserMappingStats(ctx, "")
	if err != nil {
		s.logger.Error("Failed to get user mapping stats", "error", err)
	} else if item, ok := mappingActionItem(userStats, "warning", "user_mappings", "users",
		"Unmapped users will not have their contributions properly attributed after migration.",
		"Review and complete user mappings before migrating affected repositories"); ok {
		items = append(items, item)
	}

	// Check team mapping stats
	teamStats, err := s.db.GetTeamMappingStats(ctx, "")
	if err != nil {
		s.logger.Error("Failed to get team mapping stats", "error", err)
	} else if item, ok := mappingActionItem(teamStats, "info", "team_mappings", "teams",
		"Unmapped teams will not have their permissions preserved after migration.",
		"Review and complete team mappings"); ok {
		items = append(items, item)
	}

	// Check for pending batches ready to execute
	batches, err := s.db.ListBatches(ctx)
	if err != nil {
		s.logger.Error("Failed to list batches", "error", err)
	} else {
		pendingBatches := make([]string, 0)
		for _, b := range batches {
			if b.Status == models.BatchStatusPending && b.RepositoryCount > 0 {
				pendingBatches = append(pendingBatches, b.Name)
			}
		}
		if len(pendingBatches) > 0 {
			items = append(items, ActionItem{
				Severity:    "info",
				Category:    "pending_batches",
				Title:       fmt.Sprintf("%d batches ready for execution", len(pendingBatches)),
				Description: "These batches have been created and have repositories assigned but have not been started.",
				Count:       len(pendingBatches),
				Details:     pendingBatches,
				Action:      "Schedule with schedule_batch or start with start_migration",
			})
		}
	}

	output := GetActionItemsOutput{
		Items:   items,
		Count:   len(items),
		Message: fmt.Sprintf("Found %d action items requiring attention", len(items)),
	}
	return s.jsonResult(output)
}

// handleRollbackMigration rolls back a completed migration
func (s *Server) handleRollbackMigration(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName, err := req.RequireString("repository")
	if err != nil {
		return mcp.NewToolResultError("repository parameter is required"), nil
	}

	reason := req.GetString("reason", "")

	repo, err := s.db.GetRepository(ctx, repoName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %v", err)), nil
	}

	// Validate that the repository is in a completed state
	if repo.Status != string(models.StatusComplete) && repo.Status != string(models.StatusMigrationComplete) {
		return mcp.NewToolResultError(fmt.Sprintf("Cannot rollback repository %s: current status is '%s', must be '%s' or '%s'",
			repoName, repo.Status, models.StatusComplete, models.StatusMigrationComplete)), nil
	}

	previousStatus := repo.Status

	if err := s.db.RollbackRepository(ctx, repoName, reason); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to rollback repository: %v", err)), nil
	}

	output := RollbackMigrationOutput{
		Repository:     repoName,
		PreviousStatus: previousStatus,
		NewStatus:      string(models.StatusRolledBack),
		Reason:         reason,
		Success:        true,
		Message:        fmt.Sprintf("Successfully rolled back migration for %s (was: %s, now: %s)", repoName, previousStatus, models.StatusRolledBack),
	}
	return s.jsonResult(output)
}

// mappingActionItem extracts unmapped/total counts from a stats map and builds an ActionItem if there are unmapped entries.
func mappingActionItem(stats map[string]any, severity, category, entityName, description, action string) (ActionItem, bool) {
	unmapped := intFromStatsMap(stats, "unmapped")
	if unmapped <= 0 {
		return ActionItem{}, false
	}
	total := intFromStatsMap(stats, "total")
	return ActionItem{
		Severity:    severity,
		Category:    category,
		Title:       fmt.Sprintf("%d unmapped %s out of %d total", unmapped, entityName, total),
		Description: description,
		Count:       unmapped,
		Action:      action,
	}, true
}

// intFromStatsMap safely extracts an int from a map[string]any, handling both int and float64 values.
func intFromStatsMap(m map[string]any, key string) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}
