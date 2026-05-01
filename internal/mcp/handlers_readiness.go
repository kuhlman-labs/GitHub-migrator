package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleGetMigrationReadiness implements the get_migration_readiness tool
func (s *Server) handleGetMigrationReadiness(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org := req.GetString("organization", "")
	batchID := int64(req.GetInt("batch_id", 0))
	sourceIDVal := int64(req.GetInt("source_id", 0))

	var sourceID *int64
	if sourceIDVal > 0 {
		sourceID = &sourceIDVal
	}

	checklist := []ReadinessCheckItem{}
	totalScore := 0
	maxScore := 0
	blockingIssues := []string{}

	// 1. Check user mapping completeness — sync from discovered users first
	maxScore += 25
	if _, syncErr := s.db.SyncUserMappingsFromUsers(ctx); syncErr != nil {
		s.logger.Warn("Failed to sync user mappings before readiness check", "error", syncErr)
	}
	userStats, err := s.db.GetUserMappingStats(ctx, org)
	if err != nil {
		checklist = append(checklist, ReadinessCheckItem{
			Name:   "User Mappings",
			Status: "warning",
			Detail: "Could not retrieve user mapping stats",
		})
	} else {
		totalUsers := getIntFromMap(userStats, "total")
		mappedUsers := getIntFromMap(userStats, "mapped")
		if totalUsers == 0 {
			checklist = append(checklist, ReadinessCheckItem{
				Name:   "User Mappings",
				Status: "warning",
				Detail: "No users discovered yet. Run user discovery first.",
			})
		} else {
			pct := float64(mappedUsers) / float64(totalUsers) * 100
			if pct >= 90 {
				totalScore += 25
				checklist = append(checklist, ReadinessCheckItem{
					Name:   "User Mappings",
					Status: "pass",
					Detail: fmt.Sprintf("%d/%d users mapped (%.0f%%)", mappedUsers, totalUsers, pct),
				})
			} else if pct >= 50 {
				totalScore += 15
				checklist = append(checklist, ReadinessCheckItem{
					Name:   "User Mappings",
					Status: "warning",
					Detail: fmt.Sprintf("%d/%d users mapped (%.0f%%) — unmapped users will appear as mannequins", mappedUsers, totalUsers, pct),
				})
			} else {
				checklist = append(checklist, ReadinessCheckItem{
					Name:   "User Mappings",
					Status: "fail",
					Detail: fmt.Sprintf("Only %d/%d users mapped (%.0f%%). Most users will appear as mannequins.", mappedUsers, totalUsers, pct),
				})
				blockingIssues = append(blockingIssues, fmt.Sprintf("User mapping is only %.0f%% complete", pct))
			}
		}
	}

	// 2. Check team mapping completeness — sync from discovered teams first
	maxScore += 25
	if _, syncErr := s.db.SyncTeamMappingsFromTeams(ctx); syncErr != nil {
		s.logger.Warn("Failed to sync team mappings before readiness check", "error", syncErr)
	}
	teamStats, err := s.db.GetTeamMappingStats(ctx, org)
	if err != nil {
		checklist = append(checklist, ReadinessCheckItem{
			Name:   "Team Mappings",
			Status: "warning",
			Detail: "Could not retrieve team mapping stats",
		})
	} else {
		totalTeams := getIntFromMap(teamStats, "total")
		mappedTeams := getIntFromMap(teamStats, "mapped")
		if totalTeams == 0 {
			totalScore += 25 // No teams to map is fine
			checklist = append(checklist, ReadinessCheckItem{
				Name:   "Team Mappings",
				Status: "pass",
				Detail: "No teams discovered (or teams not applicable)",
			})
		} else {
			pct := float64(mappedTeams) / float64(totalTeams) * 100
			if pct >= 80 {
				totalScore += 25
				checklist = append(checklist, ReadinessCheckItem{
					Name:   "Team Mappings",
					Status: "pass",
					Detail: fmt.Sprintf("%d/%d teams mapped (%.0f%%)", mappedTeams, totalTeams, pct),
				})
			} else if pct >= 50 {
				totalScore += 15
				checklist = append(checklist, ReadinessCheckItem{
					Name:   "Team Mappings",
					Status: "warning",
					Detail: fmt.Sprintf("%d/%d teams mapped (%.0f%%)", mappedTeams, totalTeams, pct),
				})
			} else {
				checklist = append(checklist, ReadinessCheckItem{
					Name:   "Team Mappings",
					Status: "fail",
					Detail: fmt.Sprintf("Only %d/%d teams mapped (%.0f%%)", mappedTeams, totalTeams, pct),
				})
			}
		}
	}

	// 3. Check for repos with blockers (load validation details and inspect)
	maxScore += 25
	blockerFilters := map[string]any{
		"include_details": true,
	}
	if org != "" {
		blockerFilters["organization"] = org
	}
	if batchID > 0 {
		blockerFilters["batch_id"] = batchID
	}
	allRepos, err := s.db.ListRepositories(ctx, blockerFilters)
	if err != nil {
		checklist = append(checklist, ReadinessCheckItem{
			Name:   "Migration Blockers",
			Status: "warning",
			Detail: "Could not check for blockers",
		})
	} else {
		blockerCount := countReposWithBlockers(allRepos)
		if blockerCount == 0 {
			totalScore += 25
			checklist = append(checklist, ReadinessCheckItem{
				Name:   "Migration Blockers",
				Status: "pass",
				Detail: "No repositories with migration blockers",
			})
		} else {
			checklist = append(checklist, ReadinessCheckItem{
				Name:   "Migration Blockers",
				Status: "fail",
				Detail: fmt.Sprintf("%d repositories have migration blockers requiring remediation", blockerCount),
			})
			blockingIssues = append(blockingIssues, fmt.Sprintf("%d repos have migration blockers", blockerCount))
		}
	}

	// 4. Check dry-run status
	maxScore += 25
	dryRunFilters := map[string]any{
		"status": string(models.StatusDryRunComplete),
	}
	if org != "" {
		dryRunFilters["organization"] = org
	}
	if batchID > 0 {
		dryRunFilters["batch_id"] = batchID
	}
	dryRunRepos, _ := s.db.ListRepositories(ctx, dryRunFilters)

	pendingFilters := map[string]any{
		"status": string(models.StatusPending),
	}
	if org != "" {
		pendingFilters["organization"] = org
	}
	if batchID > 0 {
		pendingFilters["batch_id"] = batchID
	}
	pendingRepos, _ := s.db.ListRepositories(ctx, pendingFilters)

	totalEligible := len(dryRunRepos) + len(pendingRepos)
	if totalEligible == 0 {
		totalScore += 25
		checklist = append(checklist, ReadinessCheckItem{
			Name:   "Dry Run Status",
			Status: "pass",
			Detail: "No pending repositories to validate",
		})
	} else if len(dryRunRepos) > 0 && len(pendingRepos) == 0 {
		totalScore += 25
		checklist = append(checklist, ReadinessCheckItem{
			Name:   "Dry Run Status",
			Status: "pass",
			Detail: fmt.Sprintf("All %d eligible repositories have completed dry-run", len(dryRunRepos)),
		})
	} else if len(dryRunRepos) > 0 {
		totalScore += 15
		checklist = append(checklist, ReadinessCheckItem{
			Name:   "Dry Run Status",
			Status: "warning",
			Detail: fmt.Sprintf("%d repos completed dry-run, %d still pending", len(dryRunRepos), len(pendingRepos)),
		})
	} else {
		checklist = append(checklist, ReadinessCheckItem{
			Name:   "Dry Run Status",
			Status: "warning",
			Detail: fmt.Sprintf("%d repos have not had a dry-run yet", len(pendingRepos)),
		})
	}

	// Calculate overall readiness
	readinessScore := 0
	if maxScore > 0 {
		readinessScore = totalScore * 100 / maxScore
	}

	// Build next steps
	nextSteps := []string{}
	if readinessScore < 50 {
		nextSteps = append(nextSteps, "Address blocking issues before starting migration")
	}
	for _, item := range checklist {
		if item.Status == "fail" {
			switch {
			case strings.Contains(item.Name, "User"):
				nextSteps = append(nextSteps, "Run suggest_user_mappings to auto-map users")
			case strings.Contains(item.Name, "Team"):
				nextSteps = append(nextSteps, "Run suggest_team_mappings to auto-map teams")
			case strings.Contains(item.Name, "Blocker"):
				nextSteps = append(nextSteps, "Use get_repository_blockers to see what needs remediation")
			}
		}
		if item.Status == "warning" && strings.Contains(item.Name, "Dry Run") {
			nextSteps = append(nextSteps, "Run dry-run migrations on pending repositories first")
		}
	}

	_ = sourceID // reserved for future source-scoped readiness

	output := MigrationReadinessOutput{
		ReadinessScore: readinessScore,
		Checklist:      checklist,
		BlockingIssues: blockingIssues,
		NextSteps:      nextSteps,
		Message:        fmt.Sprintf("Migration readiness: %d%% (%d/%d checks passing)", readinessScore, totalScore/25, maxScore/25),
	}

	return s.jsonResult(output)
}

// parseMappingFilterParams extracts the common source_org and source_id parameters used by mapping stats handlers.
func parseMappingFilterParams(req mcp.CallToolRequest) (string, *int) {
	sourceOrg := req.GetString("source_org", "")
	sourceIDVal := int64(req.GetInt("source_id", 0))

	var sourceID *int
	if sourceIDVal > 0 {
		id := int(sourceIDVal)
		sourceID = &id
	}
	return sourceOrg, sourceID
}

// handleGetUserMappingStats implements the get_user_mapping_stats tool.
// It syncs discovered users into the mapping table before querying stats,
// so results reflect all users found during discovery.
//
//nolint:dupl // Structurally similar to handleGetTeamMappingStats but calls different sync/stats methods
func (s *Server) handleGetUserMappingStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceOrg, sourceID := parseMappingFilterParams(req)

	synced, syncErr := s.db.SyncUserMappingsFromUsers(ctx)
	if syncErr != nil {
		s.logger.Warn("Failed to sync user mappings before stats", "error", syncErr)
	}

	stats, err := s.db.GetUsersWithMappingsStats(ctx, sourceOrg, sourceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get user mapping stats: %v", err)), nil
	}

	msg := "User mapping statistics retrieved"
	if synced > 0 {
		msg = fmt.Sprintf("Synced %d new user mappings. %s", synced, msg)
	}
	return s.jsonResult(UserMappingStatsOutput{Stats: stats, Message: msg})
}

// handleGetTeamMappingStats implements the get_team_mapping_stats tool.
// It syncs discovered teams into the mapping table before querying stats,
// so results reflect all teams found during discovery.
//
//nolint:dupl // Structurally similar to handleGetUserMappingStats but calls different sync/stats methods
func (s *Server) handleGetTeamMappingStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceOrg, sourceID := parseMappingFilterParams(req)

	synced, syncErr := s.db.SyncTeamMappingsFromTeams(ctx)
	if syncErr != nil {
		s.logger.Warn("Failed to sync team mappings before stats", "error", syncErr)
	}

	stats, err := s.db.GetTeamsWithMappingsStats(ctx, sourceOrg, sourceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get team mapping stats: %v", err)), nil
	}

	msg := "Team mapping statistics retrieved"
	if synced > 0 {
		msg = fmt.Sprintf("Synced %d new team mappings. %s", synced, msg)
	}
	return s.jsonResult(TeamMappingStatsOutput{Stats: stats, Message: msg})
}

// handleGetRepositoryBlockers implements the get_repository_blockers tool
func (s *Server) handleGetRepositoryBlockers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	org := req.GetString("organization", "")
	limit := req.GetInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	// Find repos with validation blockers
	filters := map[string]any{
		"include_details": true,
		"limit":           limit,
	}
	if org != "" {
		filters["organization"] = org
	}

	repos, err := s.db.ListRepositories(ctx, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list repositories: %v", err)), nil
	}

	// Filter to repos with blockers and build detailed output
	blockerRepos := []RepositoryBlockerInfo{}
	for _, repo := range repos {
		blockers := []string{}
		remediationSteps := []string{}

		if repo.Validation != nil {
			if repo.Validation.HasBlockingFiles {
				blockers = append(blockers, "Has blocking files (files >100MB)")
				remediationSteps = append(remediationSteps, "Remove or migrate large files to Git LFS")
			}
			if repo.Validation.HasOversizedCommits {
				blockers = append(blockers, "Has oversized commits")
				remediationSteps = append(remediationSteps, "Use BFG Repo-Cleaner or git filter-branch to clean history")
			}
			if repo.Validation.HasOversizedRepository {
				blockers = append(blockers, "Repository exceeds 40 GiB size limit")
				remediationSteps = append(remediationSteps, "Reduce repository size before migration — consider splitting or archiving old branches")
			}
			if repo.Validation.HasLongRefs {
				blockers = append(blockers, "Has references with paths > 255 characters")
				remediationSteps = append(remediationSteps, "Rename or delete long reference names")
			}
			if repo.Validation.HasLargeFileWarnings {
				blockers = append(blockers, "Has large file warnings (files 50-100MB)")
				remediationSteps = append(remediationSteps, "Consider migrating large files to Git LFS before migration")
			}
		}

		if repo.Status == string(models.StatusRemediationRequired) {
			if len(blockers) == 0 {
				blockers = append(blockers, "Marked as requiring remediation")
			}
		}

		if len(blockers) > 0 {
			complexity := 0
			if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
				complexity = *repo.Validation.ComplexityScore
			}
			blockerRepos = append(blockerRepos, RepositoryBlockerInfo{
				FullName:         repo.FullName,
				Organization:     repo.GetOrganization(),
				Status:           repo.Status,
				ComplexityScore:  complexity,
				Blockers:         blockers,
				RemediationSteps: remediationSteps,
			})
		}
	}

	output := RepositoryBlockersOutput{
		Repositories: blockerRepos,
		TotalCount:   len(blockerRepos),
		Message:      fmt.Sprintf("Found %d repositories with migration blockers", len(blockerRepos)),
	}

	return s.jsonResult(output)
}

// handleSuggestUserMappings implements the suggest_user_mappings tool
func (s *Server) handleSuggestUserMappings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Sync user mappings from discovered users first
	synced, err := s.db.SyncUserMappingsFromUsers(ctx)
	if err != nil {
		s.logger.Warn("Failed to sync user mappings from users", "error", err)
	}

	// Update source org info from memberships
	orgsUpdated, err := s.db.UpdateUserMappingSourceOrgsFromMemberships(ctx)
	if err != nil {
		s.logger.Warn("Failed to update user mapping source orgs", "error", err)
	}

	// Get current stats
	stats, err := s.db.GetUserMappingStats(ctx, "")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get user mapping stats: %v", err)), nil
	}

	output := SuggestUserMappingsOutput{
		SyncedCount:     synced,
		OrgsUpdated:     orgsUpdated,
		CurrentStats:    stats,
		Message:         fmt.Sprintf("Synced %d user mappings, updated orgs for %d mappings", synced, orgsUpdated),
		Recommendations: []string{},
	}

	unmapped := getIntFromMap(stats, "unmapped")
	if unmapped > 0 {
		output.Recommendations = append(output.Recommendations,
			fmt.Sprintf("%d users still unmapped — use the web UI to fetch mannequins from the destination org for auto-matching", unmapped),
			"Export mappings via CSV, fill in destination logins, then re-import",
		)
	} else {
		output.Recommendations = append(output.Recommendations, "All users are mapped!")
	}

	return s.jsonResult(output)
}

// handleSuggestTeamMappings implements the suggest_team_mappings tool
func (s *Server) handleSuggestTeamMappings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	destOrg, err := req.RequireString("destination_org")
	if err != nil {
		return mcp.NewToolResultError("destination_org is required"), nil
	}

	// Sync team mappings from discovered teams
	synced, syncErr := s.db.SyncTeamMappingsFromTeams(ctx)
	if syncErr != nil {
		s.logger.Warn("Failed to sync team mappings from teams", "error", syncErr)
	}

	// Get suggestions based on name matching
	suggestions, err := s.db.SuggestTeamMappings(ctx, destOrg, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to suggest team mappings: %v", err)), nil
	}

	// Get current stats
	stats, _ := s.db.GetTeamMappingStats(ctx, "")

	// Convert suggestions map to slice
	suggestionList := []TeamMappingSuggestion{}
	for sourceTeam, destTeam := range suggestions {
		suggestionList = append(suggestionList, TeamMappingSuggestion{
			SourceTeam:      sourceTeam,
			DestinationTeam: destTeam,
			MatchReason:     "Name similarity",
		})
	}

	output := SuggestTeamMappingsOutput{
		SyncedCount:    synced,
		Suggestions:    suggestionList,
		CurrentStats:   stats,
		DestinationOrg: destOrg,
		Message:        fmt.Sprintf("Synced %d team mappings, found %d suggestions for org '%s'", synced, len(suggestionList), destOrg),
	}

	return s.jsonResult(output)
}

// countReposWithBlockers counts repositories that have actual migration blockers
// by inspecting validation data rather than relying on a filter.
func countReposWithBlockers(repos []*models.Repository) int {
	count := 0
	for _, repo := range repos {
		if repo.Validation != nil &&
			(repo.Validation.HasBlockingFiles || repo.Validation.HasOversizedCommits ||
				repo.Validation.HasOversizedRepository || repo.Validation.HasLongRefs) {
			count++
		} else if repo.Status == string(models.StatusRemediationRequired) {
			count++
		}
	}
	return count
}

// Helper to safely extract int from map[string]any
func getIntFromMap(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}
