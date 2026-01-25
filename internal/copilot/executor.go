// Package copilot provides the Copilot chat service integration.
package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Tool name constants
const (
	ToolFindPilotCandidates = "find_pilot_candidates"
	ToolAnalyzeRepositories = "analyze_repositories"
	ToolCreateBatch         = "create_batch"
	ToolCheckDependencies   = "check_dependencies"
	ToolPlanWaves           = "plan_waves"
	ToolGetComplexityBreak  = "get_complexity_breakdown"
	ToolGetTeamRepositories = "get_team_repositories"
	ToolGetMigrationStatus  = "get_migration_status"
	ToolScheduleBatch       = "schedule_batch"
)

// Status constants
const (
	StatusPending           = "pending"
	StatusCompleted         = "completed"
	StatusMigrationComplete = "migration_complete"
	RatingUnknown           = "unknown"
)

// ToolExecutionResult represents the result of executing a tool
type ToolExecutionResult struct {
	Tool        string          `json:"tool"`
	Success     bool            `json:"success"`
	Result      any             `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	Summary     string          `json:"summary"`
	Suggestions []string        `json:"suggestions,omitempty"`
	FollowUp    *FollowUpAction `json:"follow_up,omitempty"`
	ExecutedAt  time.Time       `json:"executed_at"`
}

// FollowUpAction represents a suggested follow-up action
type FollowUpAction struct {
	Action       string   `json:"action"` // e.g., "create_batch"
	Description  string   `json:"description"`
	Repositories []string `json:"repositories,omitempty"`
	DefaultName  string   `json:"default_name,omitempty"`
}

// ToolExecutor executes migration tools directly
type ToolExecutor struct {
	db     *storage.Database
	logger *slog.Logger
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(db *storage.Database, logger *slog.Logger) *ToolExecutor {
	return &ToolExecutor{
		db:     db,
		logger: logger,
	}
}

// ExecuteTool executes the specified tool with the given arguments
func (e *ToolExecutor) ExecuteTool(ctx context.Context, intent *DetectedIntent, previousResult *ToolExecutionResult) (*ToolExecutionResult, error) {
	if intent == nil {
		return nil, fmt.Errorf("no intent provided")
	}

	if e.logger != nil {
		e.logger.Info("Executing tool", "tool", intent.Tool, "confidence", intent.Confidence, "args", intent.Args)
	}

	switch intent.Tool {
	case ToolFindPilotCandidates:
		return e.executeFindPilotCandidates(ctx, intent.Args)
	case ToolAnalyzeRepositories:
		return e.executeAnalyzeRepositories(ctx, intent.Args)
	case ToolCreateBatch:
		return e.executeCreateBatch(ctx, intent.Args, previousResult)
	case ToolCheckDependencies:
		return e.executeCheckDependencies(ctx, intent.Args)
	case ToolPlanWaves:
		return e.executePlanWaves(ctx, intent.Args)
	case ToolGetComplexityBreak:
		return e.executeGetComplexityBreakdown(ctx, intent.Args)
	case ToolGetTeamRepositories:
		return e.executeGetTeamRepositories(ctx, intent.Args)
	case ToolGetMigrationStatus:
		return e.executeGetMigrationStatus(ctx, intent.Args)
	case ToolScheduleBatch:
		return e.executeScheduleBatch(ctx, intent.Args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", intent.Tool)
	}
}

// executeFindPilotCandidates finds repositories suitable for pilot migration
func (e *ToolExecutor) executeFindPilotCandidates(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	maxCount := 10
	if v, ok := args["max_count"].(string); ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			maxCount = parsed
		}
	} else if v, ok := args["max_count"].(int); ok {
		maxCount = v
	}
	if maxCount > 50 {
		maxCount = 50
	}

	org := ""
	if v, ok := args["organization"].(string); ok {
		org = v
	}

	// Find simple, pending repositories with few dependencies
	filters := map[string]any{
		"status":          StatusPending,
		"max_complexity":  5,
		"limit":           maxCount * 2,
		"include_details": true,
	}
	if org != "" {
		filters["organization"] = org
	}

	repos, err := e.db.ListRepositories(ctx, filters)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolFindPilotCandidates,
			Success:    false,
			Error:      fmt.Sprintf("Failed to query repositories: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Score candidates
	type scoredRepo struct {
		repo  *models.Repository
		score int
	}

	scored := make([]scoredRepo, 0, len(repos))
	for _, repo := range repos {
		score := 0

		// Check dependency count
		deps, _ := e.db.GetRepositoryDependenciesByFullName(ctx, repo.FullName)
		localDeps := 0
		for _, dep := range deps {
			if dep.IsLocal {
				localDeps++
			}
		}
		score += localDeps * 10

		if repo.IsArchived {
			score += 5
		}
		if repo.IsFork {
			score += 5
		}

		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			score += *repo.Validation.ComplexityScore
		}

		scored = append(scored, scoredRepo{repo: repo, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score < scored[j].score
	})

	// Build result with summaries
	candidates := make([]map[string]any, 0, maxCount)
	repoNames := make([]string, 0, maxCount)
	for i := 0; i < len(scored) && len(candidates) < maxCount; i++ {
		repo := scored[i].repo
		complexity := 0
		rating := RatingUnknown
		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			complexity = *repo.Validation.ComplexityScore
			rating = getComplexityRating(complexity)
		}

		size := int64(0)
		if repo.GitProperties != nil && repo.GitProperties.TotalSize != nil {
			size = *repo.GitProperties.TotalSize / 1024
		}

		candidates = append(candidates, map[string]any{
			"full_name":         repo.FullName,
			"complexity_score":  complexity,
			"complexity_rating": rating,
			"size_kb":           size,
			"is_archived":       repo.IsArchived,
			"is_fork":           repo.IsFork,
		})
		repoNames = append(repoNames, repo.FullName)
	}

	// Generate default batch name
	defaultBatchName := "pilot-wave-1"
	if org != "" {
		defaultBatchName = fmt.Sprintf("%s-pilot", org)
	}

	return &ToolExecutionResult{
		Tool:    ToolFindPilotCandidates,
		Success: true,
		Result:  candidates,
		Summary: fmt.Sprintf("Found %d repositories suitable for pilot migration", len(candidates)),
		Suggestions: []string{
			"These repositories have low complexity (≤5) and few local dependencies",
			"They're ideal for testing your migration process",
		},
		FollowUp: &FollowUpAction{
			Action:       ToolCreateBatch,
			Description:  fmt.Sprintf("Create a batch with these %d pilot repositories?", len(candidates)),
			Repositories: repoNames,
			DefaultName:  defaultBatchName,
		},
		ExecutedAt: time.Now(),
	}, nil
}

// executeAnalyzeRepositories analyzes repositories based on filters
func (e *ToolExecutor) executeAnalyzeRepositories(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	filters := map[string]any{
		"limit":           20,
		"include_details": true,
	}

	if v, ok := args["organization"].(string); ok && v != "" {
		filters["organization"] = v
	}
	if v, ok := args["status"].(string); ok && v != "" {
		filters["status"] = v
	}
	if v, ok := args["max_complexity"].(int); ok && v > 0 {
		filters["max_complexity"] = v
	}

	repos, err := e.db.ListRepositories(ctx, filters)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       "analyze_repositories",
			Success:    false,
			Error:      fmt.Sprintf("Failed to query repositories: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Build summaries
	results := make([]map[string]any, 0, len(repos))
	for _, repo := range repos {
		complexity := 0
		rating := RatingUnknown
		if repo.Validation != nil && repo.Validation.ComplexityScore != nil {
			complexity = *repo.Validation.ComplexityScore
			rating = getComplexityRating(complexity)
		}

		results = append(results, map[string]any{
			"full_name":         repo.FullName,
			"status":            repo.Status,
			"complexity_score":  complexity,
			"complexity_rating": rating,
			"is_archived":       repo.IsArchived,
			"is_fork":           repo.IsFork,
		})
	}

	// Generate summary message
	status := "all"
	if v, ok := args["status"].(string); ok && v != "" {
		status = v
	}

	return &ToolExecutionResult{
		Tool:       "analyze_repositories",
		Success:    true,
		Result:     results,
		Summary:    fmt.Sprintf("Found %d repositories (filter: %s)", len(results), status),
		ExecutedAt: time.Now(),
	}, nil
}

// executeCreateBatch creates a migration batch
func (e *ToolExecutor) executeCreateBatch(ctx context.Context, args map[string]any, previousResult *ToolExecutionResult) (*ToolExecutionResult, error) {
	name := ""
	if v, ok := args["name"].(string); ok {
		name = v
	}

	// If no name provided, use default from previous result or generate one
	if name == "" && previousResult != nil && previousResult.FollowUp != nil {
		name = previousResult.FollowUp.DefaultName
	}
	if name == "" {
		name = fmt.Sprintf("batch-%s", time.Now().Format("20060102-150405"))
	}

	// Get repositories - from args or previous result
	var repoNames []string
	if v, ok := args["repositories"].([]string); ok {
		repoNames = v
	} else if previousResult != nil && previousResult.FollowUp != nil {
		repoNames = previousResult.FollowUp.Repositories
	}

	if len(repoNames) == 0 {
		return &ToolExecutionResult{
			Tool:       ToolCreateBatch,
			Success:    false,
			Error:      "No repositories specified for batch",
			ExecutedAt: time.Now(),
		}, nil
	}

	// Verify repositories exist
	repos, err := e.db.GetRepositoriesByNames(ctx, repoNames)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolCreateBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to verify repositories: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	if len(repos) != len(repoNames) {
		return &ToolExecutionResult{
			Tool:       ToolCreateBatch,
			Success:    false,
			Error:      fmt.Sprintf("Only %d of %d repositories found", len(repos), len(repoNames)),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Create batch
	description := fmt.Sprintf("Created via Copilot with %d repositories", len(repos))
	batch := &models.Batch{
		Name:            name,
		Description:     &description,
		Type:            "custom",
		Status:          "pending",
		RepositoryCount: len(repos),
	}

	if err := e.db.CreateBatch(ctx, batch); err != nil {
		return &ToolExecutionResult{
			Tool:       ToolCreateBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to create batch: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Add repositories to batch
	repoIDs := make([]int64, len(repos))
	for i, repo := range repos {
		repoIDs[i] = repo.ID
	}

	if err := e.db.AddRepositoriesToBatch(ctx, batch.ID, repoIDs); err != nil {
		return &ToolExecutionResult{
			Tool:       ToolCreateBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to add repositories to batch: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	return &ToolExecutionResult{
		Tool:    ToolCreateBatch,
		Success: true,
		Result: map[string]any{
			"batch_id":         batch.ID,
			"batch_name":       batch.Name,
			"repository_count": batch.RepositoryCount,
			"status":           batch.Status,
		},
		Summary: fmt.Sprintf("Created batch '%s' with %d repositories", name, len(repos)),
		Suggestions: []string{
			fmt.Sprintf("Batch ID: %d", batch.ID),
			"You can schedule this batch for migration or view it on the Batches page",
		},
		FollowUp: &FollowUpAction{
			Action:      ToolScheduleBatch,
			Description: fmt.Sprintf("Would you like to schedule batch '%s' for migration?", name),
			DefaultName: name,
		},
		ExecutedAt: time.Now(),
	}, nil
}

// executeCheckDependencies checks repository dependencies
func (e *ToolExecutor) executeCheckDependencies(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	repoName := ""
	if v, ok := args["repository"].(string); ok {
		repoName = v
	}

	if repoName == "" {
		return &ToolExecutionResult{
			Tool:       ToolCheckDependencies,
			Success:    false,
			Error:      "Repository name is required",
			ExecutedAt: time.Now(),
		}, nil
	}

	includeReverse := false
	if v, ok := args["include_reverse"].(bool); ok {
		includeReverse = v
	}

	deps, err := e.db.GetRepositoryDependenciesByFullName(ctx, repoName)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolCheckDependencies,
			Success:    false,
			Error:      fmt.Sprintf("Failed to get dependencies: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Build dependency info
	dependencies := make([]map[string]any, 0, len(deps))
	for _, dep := range deps {
		info := map[string]any{
			"dependency":  dep.DependencyFullName,
			"type":        dep.DependencyType,
			"is_local":    dep.IsLocal,
			"is_migrated": false,
		}

		if dep.IsLocal {
			depRepo, err := e.db.GetRepository(ctx, dep.DependencyFullName)
			if err == nil && depRepo != nil {
				info["status"] = depRepo.Status
				info["is_migrated"] = depRepo.Status == StatusCompleted || depRepo.Status == StatusMigrationComplete
			}
		}

		dependencies = append(dependencies, info)
	}

	result := map[string]any{
		"repository":   repoName,
		"dependencies": dependencies,
		"count":        len(dependencies),
	}

	// Get reverse dependencies if requested
	if includeReverse {
		reverseDeps, err := e.db.GetDependentRepositories(ctx, repoName)
		if err == nil {
			reverse := make([]map[string]any, 0, len(reverseDeps))
			for _, repo := range reverseDeps {
				reverse = append(reverse, map[string]any{
					"repository":  repo.FullName,
					"status":      repo.Status,
					"is_migrated": repo.Status == StatusCompleted || repo.Status == StatusMigrationComplete,
				})
			}
			result["reverse_dependencies"] = reverse
		}
	}

	return &ToolExecutionResult{
		Tool:       ToolCheckDependencies,
		Success:    true,
		Result:     result,
		Summary:    fmt.Sprintf("Found %d dependencies for %s", len(dependencies), repoName),
		ExecutedAt: time.Now(),
	}, nil
}

// executePlanWaves plans migration waves
func (e *ToolExecutor) executePlanWaves(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	waveSize := 10
	if v, ok := args["wave_size"].(string); ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			waveSize = parsed
		}
	} else if v, ok := args["wave_size"].(int); ok {
		waveSize = v
	}
	if waveSize > 100 {
		waveSize = 100
	}

	org := ""
	if v, ok := args["organization"].(string); ok {
		org = v
	}

	filters := map[string]any{
		"status":          StatusPending,
		"include_details": true,
	}
	if org != "" {
		filters["organization"] = org
	}

	repos, err := e.db.ListRepositories(ctx, filters)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolPlanWaves,
			Success:    false,
			Error:      fmt.Sprintf("Failed to get repositories: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	if len(repos) == 0 {
		return &ToolExecutionResult{
			Tool:       ToolPlanWaves,
			Success:    true,
			Result:     []any{},
			Summary:    "No pending repositories found",
			ExecutedAt: time.Now(),
		}, nil
	}

	// Build dependency graph
	depGraph := make(map[string][]string)
	for _, repo := range repos {
		deps, _ := e.db.GetRepositoryDependenciesByFullName(ctx, repo.FullName)
		for _, dep := range deps {
			if dep.IsLocal {
				depGraph[repo.FullName] = append(depGraph[repo.FullName], dep.DependencyFullName)
			}
		}
	}

	// Create waves using topological sort
	waves := make([]map[string]any, 0)
	migrated := make(map[string]bool)
	repoMap := make(map[string]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.FullName] = repo
	}

	waveNum := 1
	remaining := len(repos)
	for remaining > 0 && waveNum <= 100 {
		waveRepos := make([]string, 0)

		// Find repos whose dependencies are all migrated
		for _, repo := range repos {
			if migrated[repo.FullName] {
				continue
			}

			allDepsMigrated := true
			for _, dep := range depGraph[repo.FullName] {
				if !migrated[dep] {
					if _, inPending := repoMap[dep]; inPending {
						allDepsMigrated = false
						break
					}
				}
			}

			if allDepsMigrated && len(waveRepos) < waveSize {
				waveRepos = append(waveRepos, repo.FullName)
				migrated[repo.FullName] = true
				remaining--
			}
		}

		// Handle circular dependencies
		if len(waveRepos) == 0 && remaining > 0 {
			for _, repo := range repos {
				if !migrated[repo.FullName] && len(waveRepos) < waveSize {
					waveRepos = append(waveRepos, repo.FullName)
					migrated[repo.FullName] = true
					remaining--
				}
			}
		}

		if len(waveRepos) > 0 {
			waves = append(waves, map[string]any{
				"wave_number":  waveNum,
				"repositories": waveRepos,
				"count":        len(waveRepos),
			})
			waveNum++
		}
	}

	return &ToolExecutionResult{
		Tool:    ToolPlanWaves,
		Success: true,
		Result:  waves,
		Summary: fmt.Sprintf("Planned %d waves for %d repositories", len(waves), len(repos)),
		Suggestions: []string{
			"Waves are ordered to respect dependencies",
			"Simple repositories are migrated first within each wave",
		},
		ExecutedAt: time.Now(),
	}, nil
}

// executeGetComplexityBreakdown gets complexity details for a repository
func (e *ToolExecutor) executeGetComplexityBreakdown(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	repoName := ""
	if v, ok := args["repository"].(string); ok {
		repoName = v
	}

	if repoName == "" {
		return &ToolExecutionResult{
			Tool:       ToolGetComplexityBreak,
			Success:    false,
			Error:      "Repository name is required",
			ExecutedAt: time.Now(),
		}, nil
	}

	repo, err := e.db.GetRepository(ctx, repoName)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolGetComplexityBreak,
			Success:    false,
			Error:      fmt.Sprintf("Repository not found: %s", repoName),
			ExecutedAt: time.Now(),
		}, nil
	}

	breakdown := map[string]any{
		"repository":  repoName,
		"total_score": 0,
		"rating":      RatingUnknown,
		"components":  map[string]int{},
		"blockers":    []string{},
		"warnings":    []string{},
	}

	if repo.Validation != nil {
		if repo.Validation.ComplexityScore != nil {
			breakdown["total_score"] = *repo.Validation.ComplexityScore
			breakdown["rating"] = getComplexityRating(*repo.Validation.ComplexityScore)
		}

		if repo.Validation.ComplexityBreakdown != nil {
			var components map[string]int
			if err := json.Unmarshal([]byte(*repo.Validation.ComplexityBreakdown), &components); err == nil {
				breakdown["components"] = components
			}
		}

		blockers := []string{}
		warnings := []string{}

		if repo.Validation.HasBlockingFiles {
			blockers = append(blockers, "Has blocking files")
		}
		if repo.Validation.HasOversizedCommits {
			blockers = append(blockers, "Has oversized commits")
		}
		if repo.Validation.HasOversizedRepository {
			blockers = append(blockers, "Repository is oversized")
		}
		if repo.Validation.HasLongRefs {
			warnings = append(warnings, "Has long references")
		}
		if repo.Validation.HasLargeFileWarnings {
			warnings = append(warnings, "Has large file warnings")
		}

		breakdown["blockers"] = blockers
		breakdown["warnings"] = warnings
	}

	return &ToolExecutionResult{
		Tool:       ToolGetComplexityBreak,
		Success:    true,
		Result:     breakdown,
		Summary:    fmt.Sprintf("Complexity breakdown for %s: %s", repoName, breakdown["rating"]),
		ExecutedAt: time.Now(),
	}, nil
}

// executeGetTeamRepositories gets repositories for a team
func (e *ToolExecutor) executeGetTeamRepositories(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	team := ""
	if v, ok := args["team"].(string); ok {
		team = v
	}

	if team == "" {
		return &ToolExecutionResult{
			Tool:       ToolGetTeamRepositories,
			Success:    false,
			Error:      "Team name is required (format: org/team-slug)",
			ExecutedAt: time.Now(),
		}, nil
	}

	parts := strings.SplitN(team, "/", 2)
	if len(parts) != 2 {
		return &ToolExecutionResult{
			Tool:       ToolGetTeamRepositories,
			Success:    false,
			Error:      "Team must be in format org/team-slug",
			ExecutedAt: time.Now(),
		}, nil
	}

	teamDetail, err := e.db.GetTeamDetail(ctx, parts[0], parts[1])
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolGetTeamRepositories,
			Success:    false,
			Error:      fmt.Sprintf("Team not found: %s", team),
			ExecutedAt: time.Now(),
		}, nil
	}

	repos := make([]map[string]any, 0)
	for _, tr := range teamDetail.Repositories {
		status := StatusPending
		if tr.MigrationStatus != nil {
			status = *tr.MigrationStatus
		}
		repos = append(repos, map[string]any{
			"full_name": tr.FullName,
			"status":    status,
		})
	}

	return &ToolExecutionResult{
		Tool:    ToolGetTeamRepositories,
		Success: true,
		Result: map[string]any{
			"team":         team,
			"repositories": repos,
			"count":        len(repos),
		},
		Summary:    fmt.Sprintf("Found %d repositories for team %s", len(repos), team),
		ExecutedAt: time.Now(),
	}, nil
}

// executeGetMigrationStatus gets migration status for repositories
func (e *ToolExecutor) executeGetMigrationStatus(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	var repoNames []string
	if v, ok := args["repositories"].([]string); ok {
		repoNames = v
	}

	if len(repoNames) == 0 {
		return &ToolExecutionResult{
			Tool:       ToolGetMigrationStatus,
			Success:    false,
			Error:      "At least one repository is required",
			ExecutedAt: time.Now(),
		}, nil
	}

	repos, err := e.db.GetRepositoriesByNames(ctx, repoNames)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolGetMigrationStatus,
			Success:    false,
			Error:      fmt.Sprintf("Failed to get repositories: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	statuses := make([]map[string]any, 0, len(repos))
	for _, repo := range repos {
		statuses = append(statuses, map[string]any{
			"full_name": repo.FullName,
			"status":    repo.Status,
		})
	}

	return &ToolExecutionResult{
		Tool:    ToolGetMigrationStatus,
		Success: true,
		Result: map[string]any{
			"statuses": statuses,
			"count":    len(statuses),
		},
		Summary:    fmt.Sprintf("Found status for %d of %d repositories", len(statuses), len(repoNames)),
		ExecutedAt: time.Now(),
	}, nil
}

// executeScheduleBatch schedules a batch for migration
func (e *ToolExecutor) executeScheduleBatch(ctx context.Context, args map[string]any) (*ToolExecutionResult, error) {
	batchName := ""
	if v, ok := args["batch_name"].(string); ok {
		batchName = v
	}

	scheduledAtStr := ""
	if v, ok := args["scheduled_at"].(string); ok {
		scheduledAtStr = v
	}

	if batchName == "" || scheduledAtStr == "" {
		return &ToolExecutionResult{
			Tool:       ToolScheduleBatch,
			Success:    false,
			Error:      "batch_name and scheduled_at are required",
			ExecutedAt: time.Now(),
		}, nil
	}

	scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
	if err != nil {
		// Try other common formats
		scheduledAt, err = time.Parse("2006-01-02", scheduledAtStr)
		if err != nil {
			return &ToolExecutionResult{
				Tool:       ToolScheduleBatch,
				Success:    false,
				Error:      "Invalid datetime format. Use ISO 8601 (e.g., 2024-01-15T09:00:00Z)",
				ExecutedAt: time.Now(),
			}, nil
		}
	}

	batches, err := e.db.ListBatches(ctx)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolScheduleBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to list batches: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	var batch *models.Batch
	for _, b := range batches {
		if b.Name == batchName {
			batch = b
			break
		}
	}

	if batch == nil {
		return &ToolExecutionResult{
			Tool:       ToolScheduleBatch,
			Success:    false,
			Error:      fmt.Sprintf("Batch not found: %s", batchName),
			ExecutedAt: time.Now(),
		}, nil
	}

	batch.ScheduledAt = &scheduledAt
	batch.Status = "scheduled"

	if err := e.db.UpdateBatch(ctx, batch); err != nil {
		return &ToolExecutionResult{
			Tool:       ToolScheduleBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to schedule batch: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	return &ToolExecutionResult{
		Tool:    ToolScheduleBatch,
		Success: true,
		Result: map[string]any{
			"batch_id":     batch.ID,
			"batch_name":   batch.Name,
			"status":       batch.Status,
			"scheduled_at": scheduledAt.Format(time.RFC3339),
		},
		Summary:    fmt.Sprintf("Batch '%s' scheduled for %s", batchName, scheduledAt.Format("2006-01-02 15:04:05")),
		ExecutedAt: time.Now(),
	}, nil
}

// getComplexityRating returns a rating based on complexity score
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
