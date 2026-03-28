package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
)

// Migration status constants for checking batch progress
const (
	StatusDryRunQueued       = string(models.StatusDryRunQueued)
	StatusQueuedForMigration = string(models.StatusQueuedForMigration)
)

// Helper function to convert repository to summary
func (s *Server) repoToSummary(repo *models.Repository) RepositorySummary {
	summary := RepositorySummary{
		FullName:   repo.FullName,
		Status:     repo.Status,
		IsArchived: repo.IsArchived,
		IsFork:     repo.IsFork,
		BatchID:    repo.BatchID,
		UpdatedAt:  repo.UpdatedAt,
	}

	// Extract organization from full name
	if parts := strings.Split(repo.FullName, "/"); len(parts) > 0 {
		summary.Organization = parts[0]
	}

	// Get size from git properties
	if repo.GitProperties != nil && repo.GitProperties.TotalSize != nil {
		summary.Size = *repo.GitProperties.TotalSize / 1024 // Convert to KB
	}

	// Get complexity from validation
	if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
		summary.ComplexityScore = *repo.Validation.ComplexityScore
		summary.ComplexityRating = getComplexityRating(*repo.Validation.ComplexityScore)
	}

	// Format migrated_at if available
	if repo.MigratedAt != nil {
		t := repo.MigratedAt.Format(time.RFC3339)
		summary.MigratedAt = &t
	}

	return summary
}

// Helper to get complexity rating from score
func getComplexityRating(score int) string {
	switch {
	case score <= 5:
		return "simple"
	case score <= 10:
		return "medium"
	case score <= 17:
		return "complex"
	default:
		return "very_complex"
	}
}

// buildComplexityBreakdown extracts complexity breakdown from a repository
func (s *Server) buildComplexityBreakdown(repo *models.Repository) ComplexityBreakdown {
	breakdown := ComplexityBreakdown{
		TotalScore: 0,
		Rating:     "unknown",
		Components: make(map[string]int),
	}

	if repo.Validation != nil {
		if repo.Validation.ComplexityScore != nil {
			breakdown.TotalScore = *repo.Validation.ComplexityScore
			breakdown.Rating = getComplexityRating(*repo.Validation.ComplexityScore)
		}

		if repo.Validation.ComplexityBreakdown != nil {
			var components map[string]int
			if err := json.Unmarshal([]byte(*repo.Validation.ComplexityBreakdown), &components); err == nil {
				breakdown.Components = components
			}
		}

		if repo.Validation.HasBlockingFiles {
			breakdown.Blockers = append(breakdown.Blockers, "Has blocking files")
		}
		if repo.Validation.HasOversizedCommits {
			breakdown.Blockers = append(breakdown.Blockers, "Has oversized commits")
		}
		if repo.Validation.HasOversizedRepository {
			breakdown.Blockers = append(breakdown.Blockers, "Repository is oversized")
		}
		if repo.Validation.HasLongRefs {
			breakdown.Warnings = append(breakdown.Warnings, "Has long references")
		}
		if repo.Validation.HasLargeFileWarnings {
			breakdown.Warnings = append(breakdown.Warnings, "Has large file warnings")
		}
	}

	if breakdown.TotalScore > 17 {
		breakdown.Recommendations = append(breakdown.Recommendations,
			"Consider breaking into multiple migrations",
			"Run a dry-run first to identify issues",
		)
	} else if breakdown.TotalScore > 10 {
		breakdown.Recommendations = append(breakdown.Recommendations,
			"Run a dry-run before full migration",
		)
	}

	return breakdown
}

// buildDependencyInfos converts raw dependencies to DependencyInfo slice
func (s *Server) buildDependencyInfos(ctx context.Context, deps []*models.RepositoryDependency) []DependencyInfo {
	infos := make([]DependencyInfo, 0, len(deps))
	for _, dep := range deps {
		info := DependencyInfo{
			DependencyFullName: dep.DependencyFullName,
			DependencyType:     dep.DependencyType,
			IsLocal:            dep.IsLocal,
		}
		if dep.IsLocal {
			depRepo, err := s.db.GetRepository(ctx, dep.DependencyFullName)
			if err == nil && depRepo != nil {
				info.MigrationStatus = depRepo.Status
				info.IsMigrated = depRepo.Status == StatusCompleted || depRepo.Status == StatusMigrationComplete
			}
		}
		infos = append(infos, info)
	}
	return infos
}

// buildFeaturesSummary extracts features summary from a repository
func buildFeaturesSummary(repo *models.Repository) FeaturesSummary {
	fs := FeaturesSummary{}
	if repo.GitProperties != nil {
		fs.HasLFS = repo.GitProperties.HasLFS
		fs.HasSubmodules = repo.GitProperties.HasSubmodules
		fs.HasLargeFiles = repo.GitProperties.HasLargeFiles
		fs.LargeFileCount = repo.GitProperties.LargeFileCount
		fs.BranchCount = repo.GitProperties.BranchCount
		fs.CommitCount = repo.GitProperties.CommitCount
		if repo.GitProperties.TotalSize != nil {
			fs.TotalSizeKB = *repo.GitProperties.TotalSize / 1024
		}
	}
	if repo.Features != nil {
		fs.HasWiki = repo.Features.HasWiki
		fs.HasPages = repo.Features.HasPages
		fs.HasActions = repo.Features.HasActions
		fs.HasPackages = repo.Features.HasPackages
		fs.HasDiscussions = repo.Features.HasDiscussions
		fs.HasProjects = repo.Features.HasProjects
		fs.HasRulesets = repo.Features.HasRulesets
		fs.HasCodeScanning = repo.Features.HasCodeScanning
		fs.HasDependabot = repo.Features.HasDependabot
		fs.HasSecretScanning = repo.Features.HasSecretScanning
		fs.HasCodeowners = repo.Features.HasCodeowners
		fs.HasSelfHostedRunners = repo.Features.HasSelfHostedRunners
		fs.HasReleaseAssets = repo.Features.HasReleaseAssets
		fs.BranchProtections = repo.Features.BranchProtections
		fs.WebhookCount = repo.Features.WebhookCount
		fs.EnvironmentCount = repo.Features.EnvironmentCount
		fs.SecretCount = repo.Features.SecretCount
		fs.VariableCount = repo.Features.VariableCount
		fs.WorkflowCount = repo.Features.WorkflowCount
	}
	return fs
}

// buildAggregateStats computes aggregate statistics for a set of repositories
func buildAggregateStats(repos []*models.Repository) *AnalyzeRepositoriesStats {
	stats := &AnalyzeRepositoriesStats{
		ComplexityDistribution: map[string]int{},
		FeatureCounts:          map[string]int{},
		StatusDistribution:     map[string]int{},
	}

	for _, repo := range repos {
		if repo.GitProperties != nil && repo.GitProperties.TotalSize != nil {
			stats.TotalSizeKB += *repo.GitProperties.TotalSize / 1024
		}

		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			stats.ComplexityDistribution[getComplexityRating(*repo.Validation.ComplexityScore)]++
		}

		if repo.HasLFS() {
			stats.FeatureCounts["lfs"]++
		}
		if repo.HasActions() {
			stats.FeatureCounts["actions"]++
		}
		if repo.HasSubmodules() {
			stats.FeatureCounts["submodules"]++
		}
		if repo.HasPackages() {
			stats.FeatureCounts["packages"]++
		}
		if repo.HasPages() {
			stats.FeatureCounts["pages"]++
		}
		if repo.HasWiki() {
			stats.FeatureCounts["wiki"]++
		}
		if repo.HasDiscussions() {
			stats.FeatureCounts["discussions"]++
		}
		if repo.HasProjects() {
			stats.FeatureCounts["projects"]++
		}
		if repo.HasRulesets() {
			stats.FeatureCounts["rulesets"]++
		}
		if repo.HasCodeScanning() {
			stats.FeatureCounts["code_scanning"]++
		}
		if repo.HasDependabot() {
			stats.FeatureCounts["dependabot"]++
		}
		if repo.HasSecretScanning() {
			stats.FeatureCounts["secret_scanning"]++
		}
		if repo.HasSelfHostedRunners() {
			stats.FeatureCounts["self_hosted_runners"]++
		}
		if repo.HasLargeFiles() {
			stats.FeatureCounts["large_files"]++
		}

		if repo.HasMigrationBlockers() {
			stats.BlockerCount++
		}
		if repo.IsArchived {
			stats.ArchivedCount++
		}
		if repo.IsFork {
			stats.ForkCount++
		}

		stats.StatusDistribution[repo.Status]++
	}

	return stats
}

// parseRepoNamesArray extracts a string array from MCP request arguments
func parseRepoNamesArray(args map[string]any, key string) []string {
	reposArg, ok := args[key]
	if !ok {
		return nil
	}
	var names []string
	switch v := reposArg.(type) {
	case []interface{}:
		for _, r := range v {
			if s, ok := r.(string); ok {
				names = append(names, s)
			}
		}
	case []string:
		names = v
	}
	return names
}

// handleAnalyzeRepositories implements the analyze_repositories tool
func (s *Server) handleAnalyzeRepositories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	org := req.GetString("organization", "")
	status := req.GetString("status", "")
	maxComplexity := req.GetInt("max_complexity", 0)
	minComplexity := req.GetInt("min_complexity", 0)
	limit := req.GetInt("limit", 0) // 0 = no limit (return all)

	// Check for explicit repository list
	repoNames := parseRepoNamesArray(req.GetArguments(), "repositories")

	var repos []*models.Repository
	var err error

	if len(repoNames) > 0 {
		// Fetch specific repos by name
		repos, err = s.db.GetRepositoriesByNames(ctx, repoNames)
		if err != nil {
			s.logger.Error("Failed to get repositories by names", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get repositories: %v", err)), nil
		}

		// Apply optional in-memory filters
		filtered := make([]*models.Repository, 0, len(repos))
		for _, repo := range repos {
			if org != "" {
				parts := strings.Split(repo.FullName, "/")
				if len(parts) > 0 && !strings.EqualFold(parts[0], org) {
					continue
				}
			}
			if status != "" && repo.Status != status {
				continue
			}
			if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
				score := *repo.Validation.ComplexityScore
				if minComplexity > 0 && score < minComplexity {
					continue
				}
				if maxComplexity > 0 && score > maxComplexity {
					continue
				}
			}
			filtered = append(filtered, repo)
		}
		repos = filtered
		if limit > 0 && len(repos) > limit {
			repos = repos[:limit]
		}
	} else {
		// Build filters for DB query
		filters := map[string]any{
			"include_details": true,
		}
		if limit > 0 {
			filters["limit"] = limit
		}
		if org != "" {
			filters["organization"] = org
		}
		if status != "" {
			filters["status"] = status
		}
		if minComplexity > 0 || maxComplexity > 0 {
			filters["min_complexity_score"] = minComplexity
			filters["max_complexity_score"] = maxComplexity
		}

		repos, err = s.db.ListRepositories(ctx, filters)
		if err != nil {
			s.logger.Error("Failed to list repositories", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to query repositories: %v", err)), nil
		}
	}

	// Convert to summaries
	summaries := make([]RepositorySummary, 0, len(repos))
	for _, repo := range repos {
		summaries = append(summaries, s.repoToSummary(repo))
	}

	output := AnalyzeRepositoriesOutput{
		Repositories: summaries,
		TotalCount:   len(summaries),
		Stats:        buildAggregateStats(repos),
		Message:      fmt.Sprintf("Found %d repositories matching criteria", len(summaries)),
	}

	return s.jsonResult(output)
}

// handleGetComplexityBreakdown implements the get_complexity_breakdown tool
func (s *Server) handleGetComplexityBreakdown(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName, err := req.RequireString("repository")
	if err != nil {
		return mcp.NewToolResultError("repository parameter is required"), nil
	}

	repo, err := s.db.GetRepository(ctx, repoName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %s", repoName)), nil
	}

	breakdown := s.buildComplexityBreakdown(repo)

	output := GetComplexityBreakdownOutput{
		Repository: repoName,
		Breakdown:  breakdown,
		Message:    fmt.Sprintf("Complexity breakdown for %s: %s (%d points)", repoName, breakdown.Rating, breakdown.TotalScore),
	}

	return s.jsonResult(output)
}

// handleCheckDependencies implements the check_dependencies tool
func (s *Server) handleCheckDependencies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName, err := req.RequireString("repository")
	if err != nil {
		return mcp.NewToolResultError("repository parameter is required"), nil
	}
	includeReverse := req.GetBool("include_reverse", false)

	deps, err := s.db.GetRepositoryDependenciesByFullName(ctx, repoName)
	if err != nil {
		s.logger.Error("Failed to get dependencies", "repository", repoName, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get dependencies: %v", err)), nil
	}

	dependencies := s.buildDependencyInfos(ctx, deps)

	output := CheckDependenciesOutput{
		Repository:      repoName,
		Dependencies:    dependencies,
		DependencyCount: len(dependencies),
		Message:         fmt.Sprintf("Found %d dependencies for %s", len(dependencies), repoName),
	}

	if includeReverse {
		reverseDeps, err := s.db.GetDependentRepositories(ctx, repoName)
		if err != nil {
			s.logger.Warn("Failed to get reverse dependencies", "repository", repoName, "error", err)
			output.Message = fmt.Sprintf("Found %d dependencies for %s (reverse dependency lookup failed)",
				len(dependencies), repoName)
		} else {
			for _, repo := range reverseDeps {
				output.ReverseDependencies = append(output.ReverseDependencies, DependencyInfo{
					DependencyFullName: repo.FullName,
					DependencyType:     "depends_on_this",
					IsLocal:            true,
					MigrationStatus:    repo.Status,
					IsMigrated:         repo.Status == StatusCompleted || repo.Status == StatusMigrationComplete,
				})
			}
			output.Message = fmt.Sprintf("Found %d dependencies and %d reverse dependencies for %s",
				len(dependencies), len(output.ReverseDependencies), repoName)
		}
	}

	return s.jsonResult(output)
}

// handleGetDependencyGraph implements the get_dependency_graph tool.
// Returns the enterprise-wide local dependency graph showing all repos with
// internal dependencies and their relationships.
func (s *Server) handleGetDependencyGraph(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	depTypeFilter := req.GetString("dependency_type", "")
	var depTypes []string
	if depTypeFilter != "" {
		depTypes = strings.Split(depTypeFilter, ",")
	}

	edges, err := s.db.GetAllLocalDependencyPairs(ctx, depTypes, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get dependency graph: %v", err)), nil
	}

	// Build node set and count edges per node
	nodeSet := make(map[string]bool)
	dependsOnCount := make(map[string]int)
	dependedByCount := make(map[string]int)
	for _, edge := range edges {
		nodeSet[edge.SourceRepo] = true
		nodeSet[edge.TargetRepo] = true
		dependsOnCount[edge.SourceRepo]++
		dependedByCount[edge.TargetRepo]++
	}

	// Build nodes with repo metadata
	nodes := make([]DependencyGraphNode, 0, len(nodeSet))
	for fullName := range nodeSet {
		org := ""
		status := "unknown"
		if parts := strings.Split(fullName, "/"); len(parts) > 0 {
			org = parts[0]
		}
		if repo, lookupErr := s.db.GetRepository(ctx, fullName); lookupErr == nil && repo != nil {
			status = repo.Status
		}
		nodes = append(nodes, DependencyGraphNode{
			FullName:        fullName,
			Organization:    org,
			Status:          status,
			DependsOnCount:  dependsOnCount[fullName],
			DependedByCount: dependedByCount[fullName],
		})
	}

	// Sort nodes by total connections descending for readability
	sort.Slice(nodes, func(i, j int) bool {
		totalI := nodes[i].DependsOnCount + nodes[i].DependedByCount
		totalJ := nodes[j].DependsOnCount + nodes[j].DependedByCount
		return totalI > totalJ
	})

	// Build edges
	graphEdges := make([]DependencyGraphEdge, 0, len(edges))
	for _, edge := range edges {
		graphEdges = append(graphEdges, DependencyGraphEdge{
			Source:         edge.SourceRepo,
			Target:         edge.TargetRepo,
			DependencyType: edge.DependencyType,
		})
	}

	// Detect circular dependencies
	edgeSet := make(map[string]bool)
	circularPairs := make(map[string]bool)
	for _, edge := range edges {
		key := edge.SourceRepo + "->" + edge.TargetRepo
		reverseKey := edge.TargetRepo + "->" + edge.SourceRepo
		if edgeSet[reverseKey] {
			pairKey := edge.SourceRepo + "|" + edge.TargetRepo
			if edge.SourceRepo > edge.TargetRepo {
				pairKey = edge.TargetRepo + "|" + edge.SourceRepo
			}
			circularPairs[pairKey] = true
		}
		edgeSet[key] = true
	}

	output := GetDependencyGraphOutput{
		Nodes: nodes,
		Edges: graphEdges,
		Stats: DependencyGraphStats{
			TotalReposWithDeps:   len(nodeSet),
			TotalLocalDeps:       len(edges),
			CircularDependencies: len(circularPairs),
		},
		Message: fmt.Sprintf("Found %d repositories with %d local dependency relationships", len(nodeSet), len(edges)),
	}

	return s.jsonResult(output)
}

// handleUpdateRepositoryStatus implements the update_repository_status tool.
// Supports batch updates with actions: mark_wont_migrate, unmark_wont_migrate,
// mark_migrated, reset_to_pending.
func (s *Server) handleUpdateRepositoryStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action, err := req.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError("action parameter is required"), nil
	}

	// Validate action
	validActions := map[string]models.MigrationStatus{
		"mark_wont_migrate":   models.StatusWontMigrate,
		"unmark_wont_migrate": models.StatusPending,
		"mark_migrated":       models.StatusComplete,
		"reset_to_pending":    models.StatusPending,
	}
	targetStatus, ok := validActions[action]
	if !ok {
		return mcp.NewToolResultError("action must be one of: mark_wont_migrate, unmark_wont_migrate, mark_migrated, reset_to_pending"), nil
	}

	// Get repositories list
	repoNames := parseRepoNamesArray(req.GetArguments(), "repositories")
	if len(repoNames) == 0 {
		return mcp.NewToolResultError("repositories parameter is required (array of full names)"), nil
	}

	updatedCount := 0
	var errors []string
	for _, repoName := range repoNames {
		if updateErr := s.db.UpdateRepositoryStatus(ctx, repoName, targetStatus); updateErr != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", repoName, updateErr))
		} else {
			updatedCount++
		}
	}

	output := UpdateRepositoryStatusOutput{
		UpdatedCount: updatedCount,
		FailedCount:  len(errors),
		Errors:       errors,
		Message:      fmt.Sprintf("Updated %d/%d repositories to '%s'", updatedCount, len(repoNames), targetStatus),
	}

	return s.jsonResult(output)
}

// handleGetRepositoryDetails implements the get_repository_details tool
func (s *Server) handleGetRepositoryDetails(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoName, err := req.RequireString("repository")
	if err != nil {
		return mcp.NewToolResultError("repository parameter is required"), nil
	}
	includeReverse := req.GetBool("include_reverse", false)
	historyLimit := req.GetInt("history_limit", 10)
	if historyLimit <= 0 {
		historyLimit = 10
	}

	// Get repository with all details
	repo, err := s.db.GetRepository(ctx, repoName)
	if err != nil || repo == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %s", repoName)), nil
	}

	// Build all sections
	summary := s.repoToSummary(repo)
	complexity := s.buildComplexityBreakdown(repo)
	features := buildFeaturesSummary(repo)

	// Dependencies
	deps, err := s.db.GetRepositoryDependenciesByFullName(ctx, repoName)
	if err != nil {
		deps = nil
	}
	dependencies := s.buildDependencyInfos(ctx, deps)

	// Reverse dependencies
	var reverseDeps []DependencyInfo
	if includeReverse {
		revRepos, err := s.db.GetDependentRepositories(ctx, repoName)
		if err == nil {
			for _, r := range revRepos {
				reverseDeps = append(reverseDeps, DependencyInfo{
					DependencyFullName: r.FullName,
					DependencyType:     "depends_on_this",
					IsLocal:            true,
					MigrationStatus:    r.Status,
					IsMigrated:         r.Status == StatusCompleted || r.Status == StatusMigrationComplete,
				})
			}
		}
	}

	// Migration history
	var historyRecords []MigrationHistoryRecord
	history, err := s.db.GetMigrationHistory(ctx, repo.ID)
	if err == nil {
		for i, h := range history {
			if i >= historyLimit {
				break
			}
			record := MigrationHistoryRecord{
				ID:           h.ID,
				RepositoryID: h.RepositoryID,
				Status:       h.Status,
				Phase:        h.Phase,
				StartedAt:    h.StartedAt.Format(time.RFC3339),
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
			if h.DurationSeconds != nil {
				record.DurationSeconds = h.DurationSeconds
			}
			historyRecords = append(historyRecords, record)
		}
	}

	output := GetRepositoryDetailsOutput{
		Repository:          summary,
		Complexity:          complexity,
		Features:            features,
		Dependencies:        dependencies,
		DependencyCount:     len(dependencies),
		ReverseDependencies: reverseDeps,
		MigrationHistory:    historyRecords,
		HistoryCount:        len(historyRecords),
		Message:             fmt.Sprintf("Details for %s: %s complexity (%d points), %d dependencies, %d history records", repoName, complexity.Rating, complexity.TotalScore, len(dependencies), len(historyRecords)),
	}

	return s.jsonResult(output)
}

// handleFindPilotCandidates implements the find_pilot_candidates tool
func (s *Server) handleFindPilotCandidates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	maxCount := req.GetInt("max_count", 10)
	org := req.GetString("organization", "")
	sourceIDVal := int64(req.GetInt("source_id", 0))
	maxComplexity := req.GetInt("max_complexity", 5)

	// Find pending repositories within complexity threshold
	// Fetch extra to allow scoring and filtering by dependencies
	fetchLimit := maxCount * 3
	if fetchLimit < 100 {
		fetchLimit = 100
	}
	filters := map[string]any{
		"status":          StatusPending,
		"max_complexity":  maxComplexity,
		"limit":           fetchLimit,
		"include_details": true,
	}
	if org != "" {
		filters["organization"] = org
	}
	if sourceIDVal > 0 {
		filters["source_id"] = sourceIDVal
	}

	repos, err := s.db.ListRepositories(ctx, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find candidates: %v", err)), nil
	}

	// Score candidates by how good they are for pilots
	type scoredRepo struct {
		repo  *models.Repository
		score int // Lower is better
	}

	scored := make([]scoredRepo, 0, len(repos))
	for _, repo := range repos {
		score := 0

		// Check dependency count
		deps, _ := s.db.GetRepositoryDependenciesByFullName(ctx, repo.FullName)
		localDeps := 0
		for _, dep := range deps {
			if dep.IsLocal {
				localDeps++
			}
		}
		score += localDeps * 10 // Penalize local dependencies heavily

		// Prefer non-archived, non-fork repos
		if repo.IsArchived {
			score += 5
		}
		if repo.IsFork {
			score += 5
		}

		// Add complexity to score
		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			score += *repo.Validation.ComplexityScore
		}

		scored = append(scored, scoredRepo{repo: repo, score: score})
	}

	// Sort by score (lower is better)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score < scored[j].score
	})

	// Take top candidates
	candidates := make([]RepositorySummary, 0, maxCount)
	for i := 0; i < len(scored) && len(candidates) < maxCount; i++ {
		candidates = append(candidates, s.repoToSummary(scored[i].repo))
	}

	output := FindPilotCandidatesOutput{
		Candidates: candidates,
		Count:      len(candidates),
		Criteria:   fmt.Sprintf("Complexity ≤%d, few local dependencies, not archived, not a fork", maxComplexity),
		Message:    fmt.Sprintf("Found %d good pilot migration candidates", len(candidates)),
	}

	return s.jsonResult(output)
}

// handleCreateBatch implements the create_batch tool
func (s *Server) handleCreateBatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name parameter is required"), nil
	}
	description := req.GetString("description", "")

	// Get repositories array
	args := req.GetArguments()
	reposArg, ok := args["repositories"]
	if !ok {
		return mcp.NewToolResultError("repositories parameter is required"), nil
	}

	// Convert repositories to string slice
	var repoNames []string
	switch v := reposArg.(type) {
	case []interface{}:
		for _, r := range v {
			if s, ok := r.(string); ok {
				repoNames = append(repoNames, s)
			}
		}
	case []string:
		repoNames = v
	default:
		return mcp.NewToolResultError("repositories must be an array of strings"), nil
	}

	if len(repoNames) == 0 {
		return mcp.NewToolResultError("at least one repository is required"), nil
	}

	// Verify repositories exist
	repos, err := s.db.GetRepositoriesByNames(ctx, repoNames)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to verify repositories: %v", err)), nil
	}

	if len(repos) != len(repoNames) {
		return mcp.NewToolResultError(fmt.Sprintf("Only %d of %d repositories found", len(repos), len(repoNames))), nil
	}

	// Create batch
	batch := &models.Batch{
		Name:            name,
		Description:     &description,
		Type:            "custom",
		Status:          StatusPending,
		RepositoryCount: len(repos),
	}

	if err := s.db.CreateBatch(ctx, batch); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create batch: %v", err)), nil
	}

	// Add repositories to batch
	repoIDs := make([]int64, len(repos))
	for i, repo := range repos {
		repoIDs[i] = repo.ID
	}

	if err := s.db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add repositories to batch: %v", err)), nil
	}

	output := CreateBatchOutput{
		Batch: BatchInfo{
			ID:              batch.ID,
			Name:            batch.Name,
			Description:     description,
			Status:          batch.Status,
			RepositoryCount: batch.RepositoryCount,
			CreatedAt:       batch.CreatedAt,
		},
		Success: true,
		Message: fmt.Sprintf("Created batch '%s' with %d repositories", name, len(repos)),
	}

	return s.jsonResult(output)
}

// handlePlanWaves implements the plan_waves tool
// nolint:gocyclo // Wave planning requires handling multiple scenarios (dependencies, sorting, circular deps)
func (s *Server) handlePlanWaves(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	waveSize := req.GetInt("wave_size", 10)
	if waveSize > 100 {
		waveSize = 100
	}
	org := req.GetString("organization", "")

	// Get all pending repositories
	filters := map[string]any{
		"status":          StatusPending,
		"include_details": true,
	}
	if org != "" {
		filters["organization"] = org
	}

	repos, err := s.db.ListRepositories(ctx, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get repositories: %v", err)), nil
	}

	if len(repos) == 0 {
		return s.jsonResult(PlanWavesOutput{
			Waves:             []WavePlan{},
			TotalWaves:        0,
			TotalRepositories: 0,
			Message:           "No pending repositories found",
		})
	}

	// Build dependency graph
	depGraph := make(map[string][]string) // repo -> dependencies
	for _, repo := range repos {
		deps, _ := s.db.GetRepositoryDependenciesByFullName(ctx, repo.FullName)
		for _, dep := range deps {
			if dep.IsLocal {
				depGraph[repo.FullName] = append(depGraph[repo.FullName], dep.DependencyFullName)
			}
		}
	}

	// Create waves using topological sort approach
	waves := []WavePlan{}
	migrated := make(map[string]bool)
	repoMap := make(map[string]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.FullName] = repo
	}

	waveNum := 1
	remaining := len(repos)
	for remaining > 0 {
		wave := WavePlan{
			WaveNumber:   waveNum,
			Repositories: []RepositorySummary{},
		}

		// Find repos whose dependencies are all migrated
		candidates := []*models.Repository{}
		for _, repo := range repos {
			if migrated[repo.FullName] {
				continue
			}

			// Check if all dependencies are migrated
			allDepsMigrated := true
			for _, dep := range depGraph[repo.FullName] {
				if !migrated[dep] {
					// Check if dependency is in pending repos (if not, assume it's already migrated)
					if _, inPending := repoMap[dep]; inPending {
						allDepsMigrated = false
						break
					}
				}
			}

			if allDepsMigrated {
				candidates = append(candidates, repo)
			}
		}

		// Sort by complexity (simple first)
		sort.Slice(candidates, func(i, j int) bool {
			ci, cj := 0, 0
			if candidates[i].Validation != nil && candidates[i].Validation.ComplexityScore != nil {
				ci = *candidates[i].Validation.ComplexityScore
			}
			if candidates[j].Validation != nil && candidates[j].Validation.ComplexityScore != nil {
				cj = *candidates[j].Validation.ComplexityScore
			}
			return ci < cj
		})

		// Take up to waveSize candidates
		for i := 0; i < len(candidates) && len(wave.Repositories) < waveSize; i++ {
			repo := candidates[i]
			wave.Repositories = append(wave.Repositories, s.repoToSummary(repo))
			migrated[repo.FullName] = true
			remaining--

			// Update wave stats
			if repo.GitProperties != nil && repo.GitProperties.TotalSize != nil {
				wave.TotalSize += *repo.GitProperties.TotalSize / 1024
			}
			if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
				wave.AvgComplexity += float64(*repo.Validation.ComplexityScore)
			}
			wave.Dependencies += len(depGraph[repo.FullName])
		}

		// Calculate average complexity
		if len(wave.Repositories) > 0 {
			wave.AvgComplexity /= float64(len(wave.Repositories))
		}

		// Handle circular dependencies - if no candidates but remaining repos
		if len(wave.Repositories) == 0 && remaining > 0 {
			// Force add remaining repos (circular dependency case)
			for _, repo := range repos {
				if !migrated[repo.FullName] && len(wave.Repositories) < waveSize {
					wave.Repositories = append(wave.Repositories, s.repoToSummary(repo))
					migrated[repo.FullName] = true
					remaining--
				}
			}
		}

		if len(wave.Repositories) > 0 {
			waves = append(waves, wave)
			waveNum++
		}

		// Safety check to prevent infinite loop
		if waveNum > 100 {
			break
		}
	}

	output := PlanWavesOutput{
		Waves:             waves,
		TotalWaves:        len(waves),
		TotalRepositories: len(repos),
		Message:           fmt.Sprintf("Planned %d waves for %d repositories", len(waves), len(repos)),
	}

	return s.jsonResult(output)
}

// handleGetTeamRepositories implements the get_team_repositories tool
func (s *Server) handleGetTeamRepositories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	teamArg, err := req.RequireString("team")
	if err != nil {
		return mcp.NewToolResultError("team parameter is required"), nil
	}
	includeMigrated := req.GetBool("include_migrated", false)

	// Parse team format: org/team-slug
	parts := strings.SplitN(teamArg, "/", 2)
	if len(parts) != 2 {
		return mcp.NewToolResultError("team must be in format org/team-slug"), nil
	}
	org, slug := parts[0], parts[1]

	// Get team detail which includes repositories
	teamDetail, err := s.db.GetTeamDetail(ctx, org, slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Team not found: %s", teamArg)), nil
	}

	// Convert to summaries
	repos := make([]RepositorySummary, 0)
	for _, tr := range teamDetail.Repositories {
		status := StatusPending
		if tr.MigrationStatus != nil {
			status = *tr.MigrationStatus
		}

		// Filter out migrated if not requested
		if !includeMigrated && (status == StatusCompleted || status == StatusComplete || status == StatusMigrationComplete) {
			continue
		}

		repos = append(repos, RepositorySummary{
			FullName:     tr.FullName,
			Organization: org,
			Status:       status,
		})
	}

	output := GetTeamRepositoriesOutput{
		Team:         teamArg,
		Repositories: repos,
		Count:        len(repos),
		Message:      fmt.Sprintf("Found %d repositories for team %s", len(repos), teamArg),
	}

	return s.jsonResult(output)
}

// handleGetMigrationStatus implements the get_migration_status tool
func (s *Server) handleGetMigrationStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get repositories array
	args := req.GetArguments()
	reposArg, ok := args["repositories"]
	if !ok {
		return mcp.NewToolResultError("repositories parameter is required"), nil
	}

	// Convert to string slice
	var repoNames []string
	switch v := reposArg.(type) {
	case []interface{}:
		for _, r := range v {
			if s, ok := r.(string); ok {
				repoNames = append(repoNames, s)
			}
		}
	case []string:
		repoNames = v
	default:
		return mcp.NewToolResultError("repositories must be an array of strings"), nil
	}

	if len(repoNames) == 0 {
		return mcp.NewToolResultError("at least one repository is required"), nil
	}

	// Get repositories
	repos, err := s.db.GetRepositoriesByNames(ctx, repoNames)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get repositories: %v", err)), nil
	}

	// Convert to summaries
	statuses := make([]RepositorySummary, 0, len(repos))
	for _, repo := range repos {
		statuses = append(statuses, s.repoToSummary(repo))
	}

	output := GetMigrationStatusOutput{
		Statuses: statuses,
		Count:    len(statuses),
		Message:  fmt.Sprintf("Found status for %d of %d requested repositories", len(statuses), len(repoNames)),
	}

	return s.jsonResult(output)
}

// handleScheduleBatch implements the schedule_batch tool
func (s *Server) handleScheduleBatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	batchName, err := req.RequireString("batch_name")
	if err != nil {
		return mcp.NewToolResultError("batch_name parameter is required"), nil
	}
	scheduledAtStr, err := req.RequireString("scheduled_at")
	if err != nil {
		return mcp.NewToolResultError("scheduled_at parameter is required"), nil
	}

	// Parse scheduled time
	scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid datetime format. Use ISO 8601 (e.g., 2024-01-15T09:00:00Z): %v", err)), nil
	}

	// Find batch by name
	batches, err := s.db.ListBatches(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list batches: %v", err)), nil
	}

	var batch *models.Batch
	for _, b := range batches {
		if b.Name == batchName {
			batch = b
			break
		}
	}

	if batch == nil {
		return mcp.NewToolResultError(fmt.Sprintf("Batch not found: %s", batchName)), nil
	}

	// Update batch with scheduled time
	batch.ScheduledAt = &scheduledAt
	batch.Status = "scheduled"

	if err := s.db.UpdateBatch(ctx, batch); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to schedule batch: %v", err)), nil
	}

	scheduledAtFormatted := scheduledAt.Format(time.RFC3339)
	output := ScheduleBatchOutput{
		Batch: BatchInfo{
			ID:              batch.ID,
			Name:            batch.Name,
			Status:          batch.Status,
			RepositoryCount: batch.RepositoryCount,
			ScheduledAt:     &scheduledAtFormatted,
			CreatedAt:       batch.CreatedAt,
		},
		Success: true,
		Message: fmt.Sprintf("Batch '%s' scheduled for %s", batchName, scheduledAt.Format("2006-01-02 15:04:05 MST")),
	}

	return s.jsonResult(output)
}

// handleConfigureBatch implements the configure_batch tool
func (s *Server) handleConfigureBatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract batch name or ID
	batchName := req.GetString("batch_name", "")
	batchID := int64(req.GetInt("batch_id", 0))

	// Extract settings to configure
	destinationOrg := req.GetString("destination_org", "")
	migrationAPI := req.GetString("migration_api", "")

	if batchName == "" && batchID == 0 {
		return mcp.NewToolResultError("batch_name or batch_id is required"), nil
	}

	if destinationOrg == "" && migrationAPI == "" {
		return mcp.NewToolResultError("At least one setting must be specified (destination_org or migration_api)"), nil
	}

	// Find batch
	batches, err := s.db.ListBatches(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list batches: %v", err)), nil
	}

	var batch *models.Batch
	for _, b := range batches {
		if (batchName != "" && b.Name == batchName) || (batchID != 0 && b.ID == batchID) {
			batch = b
			break
		}
	}

	if batch == nil {
		searchTerm := batchName
		if batchID != 0 {
			searchTerm = fmt.Sprintf("ID %d", batchID)
		}
		return mcp.NewToolResultError(fmt.Sprintf("Batch not found: %s", searchTerm)), nil
	}

	// Update batch settings
	changes := []string{}
	if destinationOrg != "" {
		batch.DestinationOrg = &destinationOrg
		changes = append(changes, fmt.Sprintf("destination organization set to '%s'", destinationOrg))
	}
	if migrationAPI != "" {
		migrationAPI = strings.ToUpper(migrationAPI)
		if migrationAPI != models.MigrationAPIGEI && migrationAPI != models.MigrationAPIELM {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid migration_api '%s'. Must be 'GEI' or 'ELM'", migrationAPI)), nil
		}
		batch.MigrationAPI = migrationAPI
		changes = append(changes, fmt.Sprintf("migration API set to '%s'", migrationAPI))
	}

	if err := s.db.UpdateBatch(ctx, batch); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update batch: %v", err)), nil
	}

	output := ConfigureBatchOutput{
		Batch: BatchInfo{
			ID:              batch.ID,
			Name:            batch.Name,
			Status:          batch.Status,
			RepositoryCount: batch.RepositoryCount,
			DestinationOrg:  batch.DestinationOrg,
			MigrationAPI:    batch.MigrationAPI,
			CreatedAt:       batch.CreatedAt,
		},
		Success: true,
		Message: fmt.Sprintf("Batch '%s' updated: %s", batch.Name, strings.Join(changes, ", ")),
	}

	return s.jsonResult(output)
}

// handleStartMigration implements the start_migration tool
// nolint:gocyclo // Migration starting requires handling multiple scenarios (batch, single repo, multiple repos)
func (s *Server) handleStartMigration(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	batchName := req.GetString("batch_name", "")
	batchID := int64(req.GetInt("batch_id", 0))
	repository := req.GetString("repository", "")
	dryRun := req.GetBool("dry_run", true) // Default to dry-run for safety

	// Get repositories array if provided
	var repoNames []string
	args := req.GetArguments()
	if reposArg, ok := args["repositories"]; ok {
		switch v := reposArg.(type) {
		case []interface{}:
			for _, r := range v {
				if str, ok := r.(string); ok {
					repoNames = append(repoNames, str)
				}
			}
		case []string:
			repoNames = v
		}
	}

	// Validate that at least one target is specified
	if batchName == "" && batchID == 0 && repository == "" && len(repoNames) == 0 {
		return mcp.NewToolResultError("At least one of batch_name, batch_id, repository, or repositories must be specified"), nil
	}

	// Determine target status based on dry_run flag
	targetStatus := models.StatusQueuedForMigration
	if dryRun {
		targetStatus = models.StatusDryRunQueued
	}

	var queuedRepos []RepositorySummary
	var batch *models.Batch
	skippedCount := 0

	// Handle batch migration
	if batchName != "" || batchID != 0 {
		batches, err := s.db.ListBatches(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list batches: %v", err)), nil
		}

		for _, b := range batches {
			if (batchName != "" && b.Name == batchName) || (batchID != 0 && b.ID == batchID) {
				batch = b
				break
			}
		}

		if batch == nil {
			searchTerm := batchName
			if batchID != 0 {
				searchTerm = fmt.Sprintf("ID %d", batchID)
			}
			return mcp.NewToolResultError(fmt.Sprintf("Batch not found: %s", searchTerm)), nil
		}

		// Validate batch status
		if batch.Status == models.BatchStatusInProgress {
			return mcp.NewToolResultError(fmt.Sprintf("Batch '%s' is already running", batch.Name)), nil
		}

		// Get batch repositories
		repos, err := s.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get batch repositories: %v", err)), nil
		}

		if len(repos) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Batch '%s' has no repositories", batch.Name)), nil
		}

		// Update batch status
		batch.Status = models.BatchStatusInProgress
		now := time.Now()
		if dryRun {
			batch.DryRunStartedAt = &now
			batch.LastDryRunAt = &now
		} else {
			batch.StartedAt = &now
			batch.LastMigrationAttemptAt = &now
		}
		if err := s.db.UpdateBatch(ctx, batch); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update batch status: %v", err)), nil
		}

		// Queue repositories
		priority := 0
		if batch.Type == models.BatchTypePilot {
			priority = 1
		}

		for _, repo := range repos {
			if canQueueForMigration(repo.Status, dryRun) {
				repo.Status = string(targetStatus)
				repo.Priority = priority
				if err := s.db.UpdateRepository(ctx, repo); err != nil {
					s.logger.Error("Failed to queue repository", "repo", repo.FullName, "error", err)
					continue
				}
				queuedRepos = append(queuedRepos, s.repoToSummary(repo))
			} else {
				skippedCount++
			}
		}
	}

	// Handle single repository
	if repository != "" {
		repo, err := s.db.GetRepository(ctx, repository)
		if err != nil || repo == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %s", repository)), nil
		}

		if !canQueueForMigration(repo.Status, dryRun) {
			return mcp.NewToolResultError(fmt.Sprintf("Repository '%s' cannot be queued for migration (status: %s)", repository, repo.Status)), nil
		}

		repo.Status = string(targetStatus)
		if err := s.db.UpdateRepository(ctx, repo); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to queue repository: %v", err)), nil
		}
		queuedRepos = append(queuedRepos, s.repoToSummary(repo))
	}

	// Handle multiple repositories
	if len(repoNames) > 0 {
		repos, err := s.db.GetRepositoriesByNames(ctx, repoNames)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get repositories: %v", err)), nil
		}

		for _, repo := range repos {
			if canQueueForMigration(repo.Status, dryRun) {
				repo.Status = string(targetStatus)
				if err := s.db.UpdateRepository(ctx, repo); err != nil {
					s.logger.Error("Failed to queue repository", "repo", repo.FullName, "error", err)
					continue
				}
				queuedRepos = append(queuedRepos, s.repoToSummary(repo))
			} else {
				skippedCount++
			}
		}
	}

	if len(queuedRepos) == 0 {
		return mcp.NewToolResultError("No repositories could be queued for migration"), nil
	}

	// Build output message
	migrationType := "production migration"
	if dryRun {
		migrationType = "dry-run"
	}

	output := StartMigrationOutput{
		Repositories: queuedRepos,
		QueuedCount:  len(queuedRepos),
		SkippedCount: skippedCount,
		DryRun:       dryRun,
		Success:      true,
		Message:      fmt.Sprintf("Started %s for %d repositories", migrationType, len(queuedRepos)),
	}

	if batch != nil {
		output.BatchID = batch.ID
		output.BatchName = batch.Name
		output.Message = fmt.Sprintf("Started %s for batch '%s' (%d repositories)", migrationType, batch.Name, len(queuedRepos))
	}

	// Add next steps
	if dryRun {
		output.NextSteps = []string{
			"Monitor progress with get_migration_progress",
			"After dry-run completes, start production migration with start_migration(dry_run=false)",
		}
	} else {
		output.NextSteps = []string{
			"Monitor progress with get_migration_progress",
			"Migrations will be processed by the worker pool",
		}
	}

	return s.jsonResult(output)
}

// handleCancelMigration implements the cancel_migration tool
func (s *Server) handleCancelMigration(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	batchName := req.GetString("batch_name", "")
	batchID := int64(req.GetInt("batch_id", 0))
	repository := req.GetString("repository", "")

	if batchName == "" && batchID == 0 && repository == "" {
		return mcp.NewToolResultError("At least one of batch_name, batch_id, or repository must be specified"), nil
	}

	cancelledCount := 0

	// Handle batch cancellation
	if batchName != "" || batchID != 0 {
		batches, err := s.db.ListBatches(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list batches: %v", err)), nil
		}

		var batch *models.Batch
		for _, b := range batches {
			if (batchName != "" && b.Name == batchName) || (batchID != 0 && b.ID == batchID) {
				batch = b
				break
			}
		}

		if batch == nil {
			searchTerm := batchName
			if batchID != 0 {
				searchTerm = fmt.Sprintf("ID %d", batchID)
			}
			return mcp.NewToolResultError(fmt.Sprintf("Batch not found: %s", searchTerm)), nil
		}

		// Get batch repositories
		repos, err := s.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get batch repositories: %v", err)), nil
		}

		// Cancel queued repositories
		for _, repo := range repos {
			if isInQueuedOrInProgressState(repo.Status) {
				repo.Status = string(models.StatusPending)
				if err := s.db.UpdateRepository(ctx, repo); err != nil {
					s.logger.Error("Failed to cancel repository", "repo", repo.FullName, "error", err)
					continue
				}
				cancelledCount++
			}
		}

		// Update batch status
		batch.Status = models.BatchStatusCancelled
		if err := s.db.UpdateBatch(ctx, batch); err != nil {
			s.logger.Error("Failed to update batch status", "batch", batch.Name, "error", err)
		}

		return s.jsonResult(CancelMigrationOutput{
			BatchID:        batch.ID,
			BatchName:      batch.Name,
			CancelledCount: cancelledCount,
			Success:        true,
			Message:        fmt.Sprintf("Cancelled batch '%s' (%d repositories)", batch.Name, cancelledCount),
		})
	}

	// Handle single repository cancellation
	if repository != "" {
		repo, err := s.db.GetRepository(ctx, repository)
		if err != nil || repo == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %s", repository)), nil
		}

		if !isInQueuedOrInProgressState(repo.Status) {
			return mcp.NewToolResultError(fmt.Sprintf("Repository '%s' is not in a cancellable state (status: %s)", repository, repo.Status)), nil
		}

		repo.Status = string(models.StatusPending)
		if err := s.db.UpdateRepository(ctx, repo); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to cancel repository: %v", err)), nil
		}

		return s.jsonResult(CancelMigrationOutput{
			Repository:     repository,
			CancelledCount: 1,
			Success:        true,
			Message:        fmt.Sprintf("Cancelled migration for repository '%s'", repository),
		})
	}

	return mcp.NewToolResultError("No target specified for cancellation"), nil
}

// handleGetMigrationProgress implements the get_migration_progress tool
func (s *Server) handleGetMigrationProgress(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	batchName := req.GetString("batch_name", "")
	batchID := int64(req.GetInt("batch_id", 0))
	repository := req.GetString("repository", "")

	// Handle single repository progress
	if repository != "" {
		repo, err := s.db.GetRepository(ctx, repository)
		if err != nil || repo == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %s", repository)), nil
		}

		progress := MigrationProgress{TotalCount: 1}
		updateProgressFromStatus(&progress, repo.Status)

		return s.jsonResult(GetMigrationProgressOutput{
			Repository:   repository,
			Progress:     progress,
			Repositories: []RepositorySummary{s.repoToSummary(repo)},
			Message:      fmt.Sprintf("Repository '%s' status: %s", repository, repo.Status),
		})
	}

	// Handle batch progress
	if batchName != "" || batchID != 0 {
		batches, err := s.db.ListBatches(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list batches: %v", err)), nil
		}

		var batch *models.Batch
		for _, b := range batches {
			if (batchName != "" && b.Name == batchName) || (batchID != 0 && b.ID == batchID) {
				batch = b
				break
			}
		}

		if batch == nil {
			searchTerm := batchName
			if batchID != 0 {
				searchTerm = fmt.Sprintf("ID %d", batchID)
			}
			return mcp.NewToolResultError(fmt.Sprintf("Batch not found: %s", searchTerm)), nil
		}

		// Get batch repositories
		repos, err := s.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get batch repositories: %v", err)), nil
		}

		progress := MigrationProgress{TotalCount: len(repos)}
		summaries := make([]RepositorySummary, 0, len(repos))

		for _, repo := range repos {
			updateProgressFromStatus(&progress, repo.Status)
			summaries = append(summaries, s.repoToSummary(repo))
		}

		// Calculate percent complete
		if progress.TotalCount > 0 {
			progress.PercentComplete = float64(progress.CompletedCount) / float64(progress.TotalCount) * 100
		}

		return s.jsonResult(GetMigrationProgressOutput{
			BatchID:      batch.ID,
			BatchName:    batch.Name,
			BatchStatus:  batch.Status,
			Progress:     progress,
			Repositories: summaries,
			Message: fmt.Sprintf("Batch '%s': %d/%d complete (%.1f%%)",
				batch.Name, progress.CompletedCount, progress.TotalCount, progress.PercentComplete),
		})
	}

	return mcp.NewToolResultError("At least one of batch_name, batch_id, or repository must be specified"), nil
}

// canQueueForMigration checks if a repository can be queued for migration
func canQueueForMigration(status string, dryRun bool) bool {
	switch models.MigrationStatus(status) {
	case models.StatusPending,
		models.StatusDryRunFailed,
		models.StatusMigrationFailed,
		models.StatusRolledBack:
		return true
	case models.StatusDryRunComplete:
		// After dry-run, can do production migration
		return !dryRun
	default:
		return false
	}
}

// isInQueuedOrInProgressState checks if a repository is in a cancellable state
func isInQueuedOrInProgressState(status string) bool {
	switch models.MigrationStatus(status) {
	case models.StatusDryRunQueued,
		models.StatusDryRunInProgress,
		models.StatusQueuedForMigration,
		models.StatusMigratingContent,
		models.StatusArchiveGenerating,
		models.StatusPreMigration:
		return true
	default:
		return false
	}
}

// updateProgressFromStatus updates progress counters based on repository status
func updateProgressFromStatus(progress *MigrationProgress, status string) {
	switch models.MigrationStatus(status) {
	case models.StatusPending:
		progress.PendingCount++
	case models.StatusDryRunQueued, models.StatusQueuedForMigration:
		progress.QueuedCount++
	case models.StatusDryRunInProgress, models.StatusMigratingContent,
		models.StatusArchiveGenerating, models.StatusPreMigration, models.StatusPostMigration:
		progress.InProgressCount++
	case models.StatusDryRunComplete, models.StatusMigrationComplete, models.StatusComplete:
		progress.CompletedCount++
	case models.StatusDryRunFailed, models.StatusMigrationFailed:
		progress.FailedCount++
	case models.StatusWontMigrate:
		progress.SkippedCount++
	}
}

// jsonResult creates a JSON tool result
func (s *Server) jsonResult(data any) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
