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

// handleAnalyzeRepositories implements the analyze_repositories tool
func (s *Server) handleAnalyzeRepositories(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	org := req.GetString("organization", "")
	status := req.GetString("status", "")
	maxComplexity := req.GetInt("max_complexity", 0)
	minComplexity := req.GetInt("min_complexity", 0)
	limit := req.GetInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	// Build filters
	filters := map[string]any{
		"limit":           limit,
		"include_details": true, // Load git properties and validation
	}

	if org != "" {
		filters["organization"] = org
	}
	if status != "" {
		filters["status"] = status
	}
	if maxComplexity > 0 {
		filters["max_complexity"] = maxComplexity
	}
	if minComplexity > 0 {
		filters["min_complexity"] = minComplexity
	}

	// Query repositories
	repos, err := s.db.ListRepositories(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to list repositories", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query repositories: %v", err)), nil
	}

	// Convert to summaries
	summaries := make([]RepositorySummary, 0, len(repos))
	for _, repo := range repos {
		summaries = append(summaries, s.repoToSummary(repo))
	}

	output := AnalyzeRepositoriesOutput{
		Repositories: summaries,
		TotalCount:   len(summaries),
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

	// Get repository with details
	repo, err := s.db.GetRepository(ctx, repoName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository not found: %s", repoName)), nil
	}

	breakdown := ComplexityBreakdown{
		TotalScore: 0,
		Rating:     "unknown",
		Components: make(map[string]int),
	}

	// Parse complexity breakdown from validation data
	if repo.Validation != nil {
		if repo.Validation.ComplexityScore != nil {
			breakdown.TotalScore = *repo.Validation.ComplexityScore
			breakdown.Rating = getComplexityRating(*repo.Validation.ComplexityScore)
		}

		// Parse breakdown JSON if available
		if repo.Validation.ComplexityBreakdown != nil {
			var components map[string]int
			if err := json.Unmarshal([]byte(*repo.Validation.ComplexityBreakdown), &components); err == nil {
				breakdown.Components = components
			}
		}

		// Add blockers based on validation flags
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

	// Add recommendations based on complexity
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

	// Get dependencies
	deps, err := s.db.GetRepositoryDependenciesByFullName(ctx, repoName)
	if err != nil {
		s.logger.Error("Failed to get dependencies", "repository", repoName, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get dependencies: %v", err)), nil
	}

	// Convert to dependency info
	dependencies := make([]DependencyInfo, 0, len(deps))
	for _, dep := range deps {
		info := DependencyInfo{
			DependencyFullName: dep.DependencyFullName,
			DependencyType:     dep.DependencyType,
			IsLocal:            dep.IsLocal,
		}

		// Check if dependency is migrated
		if dep.IsLocal {
			depRepo, err := s.db.GetRepository(ctx, dep.DependencyFullName)
			if err == nil && depRepo != nil {
				info.MigrationStatus = depRepo.Status
				info.IsMigrated = depRepo.Status == StatusCompleted || depRepo.Status == StatusMigrationComplete
			}
		}

		dependencies = append(dependencies, info)
	}

	output := CheckDependenciesOutput{
		Repository:      repoName,
		Dependencies:    dependencies,
		DependencyCount: len(dependencies),
		Message:         fmt.Sprintf("Found %d dependencies for %s", len(dependencies), repoName),
	}

	// Get reverse dependencies if requested
	if includeReverse {
		reverseDeps, err := s.db.GetDependentRepositories(ctx, repoName)
		if err == nil {
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

// handleFindPilotCandidates implements the find_pilot_candidates tool
func (s *Server) handleFindPilotCandidates(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	maxCount := req.GetInt("max_count", 10)
	if maxCount > 50 {
		maxCount = 50
	}
	org := req.GetString("organization", "")

	// Find simple, pending repositories with few dependencies
	filters := map[string]any{
		"status":          StatusPending,
		"max_complexity":  5,            // Simple complexity
		"limit":           maxCount * 2, // Get extra to filter by dependencies
		"include_details": true,
	}
	if org != "" {
		filters["organization"] = org
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
		Criteria:   "Simple complexity (≤5), few local dependencies, not archived, not a fork",
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

// jsonResult creates a JSON tool result
func (s *Server) jsonResult(data any) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
