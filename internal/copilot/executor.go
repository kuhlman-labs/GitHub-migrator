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
	ToolFindPilotCandidates  = "find_pilot_candidates"
	ToolAnalyzeRepositories  = "analyze_repositories"
	ToolCreateBatch          = "create_batch"
	ToolConfigureBatch       = "configure_batch"
	ToolCheckDependencies    = "check_dependencies"
	ToolPlanWaves            = "plan_waves"
	ToolGetComplexityBreak   = "get_complexity_breakdown"
	ToolGetTeamRepositories  = "get_team_repositories"
	ToolGetMigrationStatus   = "get_migration_status"
	ToolScheduleBatch        = "schedule_batch"
	ToolStartMigration       = "start_migration"
	ToolCancelMigration      = "cancel_migration"
	ToolGetMigrationProgress = "get_migration_progress"
)

// Status constants
const (
	StatusPending           = "pending"
	StatusScheduled         = "scheduled"
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
		return e.executeScheduleBatch(ctx, intent.Args, previousResult)
	case ToolConfigureBatch:
		return e.executeConfigureBatch(ctx, intent.Args, previousResult)
	case ToolStartMigration:
		return e.executeStartMigration(ctx, intent.Args, previousResult)
	case ToolCancelMigration:
		return e.executeCancelMigration(ctx, intent.Args, previousResult)
	case ToolGetMigrationProgress:
		return e.executeGetMigrationProgress(ctx, intent.Args, previousResult)
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

	// Get destination organization if specified
	destinationOrg := ""
	if v, ok := args["destination_org"].(string); ok {
		destinationOrg = v
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
		Status:          StatusPending,
		RepositoryCount: len(repos),
	}

	// Set destination organization if specified
	if destinationOrg != "" {
		batch.DestinationOrg = &destinationOrg
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

	result := map[string]any{
		"batch_id":         batch.ID,
		"batch_name":       batch.Name,
		"repository_count": batch.RepositoryCount,
		"status":           batch.Status,
	}
	summary := fmt.Sprintf("Created batch '%s' with %d repositories", name, len(repos))
	suggestions := []string{
		fmt.Sprintf("Batch ID: %d", batch.ID),
		"You can schedule this batch for migration or view it on the Batches page",
	}

	if destinationOrg != "" {
		result["destination_org"] = destinationOrg
		summary = fmt.Sprintf("Created batch '%s' with %d repositories, destination: %s", name, len(repos), destinationOrg)
		suggestions = append(suggestions, fmt.Sprintf("Destination organization: %s", destinationOrg))
	}

	return &ToolExecutionResult{
		Tool:        ToolCreateBatch,
		Success:     true,
		Result:      result,
		Summary:     summary,
		Suggestions: suggestions,
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
func (e *ToolExecutor) executeScheduleBatch(ctx context.Context, args map[string]any, previousResult *ToolExecutionResult) (*ToolExecutionResult, error) {
	batchName := ""
	if v, ok := args["batch_name"].(string); ok {
		batchName = v
	}
	// Try to get batch name from previous result (after create_batch or configure_batch)
	if batchName == "" && previousResult != nil && previousResult.FollowUp != nil {
		batchName = previousResult.FollowUp.DefaultName
	}

	scheduledAtStr := ""
	if v, ok := args["scheduled_at"].(string); ok {
		scheduledAtStr = v
	}

	// Also check for destination_org to set during scheduling
	destinationOrg := ""
	if v, ok := args["destination_org"].(string); ok {
		destinationOrg = v
	}

	if batchName == "" {
		return &ToolExecutionResult{
			Tool:       ToolScheduleBatch,
			Success:    false,
			Error:      "batch_name is required",
			ExecutedAt: time.Now(),
		}, nil
	}

	// Parse scheduled time if provided
	var scheduledAt *time.Time
	if scheduledAtStr != "" {
		parsed, err := time.Parse(time.RFC3339, scheduledAtStr)
		if err != nil {
			// Try other common formats
			parsed, err = time.Parse("2006-01-02", scheduledAtStr)
			if err != nil {
				return &ToolExecutionResult{
					Tool:       ToolScheduleBatch,
					Success:    false,
					Error:      "Invalid datetime format. Use ISO 8601 (e.g., 2024-01-15T09:00:00Z)",
					ExecutedAt: time.Now(),
				}, nil
			}
		}
		scheduledAt = &parsed
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

	// Set destination org if provided
	if destinationOrg != "" {
		batch.DestinationOrg = &destinationOrg
	}

	// Set scheduled time
	if scheduledAt != nil {
		batch.ScheduledAt = scheduledAt
		batch.Status = StatusScheduled
	} else {
		// Schedule for now if no time specified
		now := time.Now()
		batch.ScheduledAt = &now
		batch.Status = StatusScheduled
		scheduledAt = &now
	}

	if err := e.db.UpdateBatch(ctx, batch); err != nil {
		return &ToolExecutionResult{
			Tool:       ToolScheduleBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to schedule batch: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	result := map[string]any{
		"batch_id":     batch.ID,
		"batch_name":   batch.Name,
		"status":       batch.Status,
		"scheduled_at": scheduledAt.Format(time.RFC3339),
	}
	summary := fmt.Sprintf("Batch '%s' scheduled for %s", batchName, scheduledAt.Format("2006-01-02 15:04:05"))

	if destinationOrg != "" {
		result["destination_org"] = destinationOrg
		summary = fmt.Sprintf("Batch '%s' configured for destination '%s' and scheduled for %s", batchName, destinationOrg, scheduledAt.Format("2006-01-02 15:04:05"))
	}

	return &ToolExecutionResult{
		Tool:       ToolScheduleBatch,
		Success:    true,
		Result:     result,
		Summary:    summary,
		ExecutedAt: time.Now(),
	}, nil
}

// executeConfigureBatch configures batch settings including destination organization
func (e *ToolExecutor) executeConfigureBatch(ctx context.Context, args map[string]any, previousResult *ToolExecutionResult) (*ToolExecutionResult, error) {
	batchName := ""
	if v, ok := args["batch_name"].(string); ok {
		batchName = v
	}
	// Try batch ID
	var batchID int64
	if v, ok := args["batch_id"].(string); ok {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			batchID = parsed
		}
	} else if v, ok := args["batch_id"].(float64); ok {
		batchID = int64(v)
	} else if v, ok := args["batch_id"].(int64); ok {
		batchID = v
	}

	// Try to get batch name from previous result
	if batchName == "" && batchID == 0 && previousResult != nil && previousResult.FollowUp != nil {
		batchName = previousResult.FollowUp.DefaultName
	}

	destinationOrg := ""
	if v, ok := args["destination_org"].(string); ok {
		destinationOrg = v
	}

	migrationAPI := ""
	if v, ok := args["migration_api"].(string); ok {
		migrationAPI = strings.ToUpper(v)
	}

	if batchName == "" && batchID == 0 {
		return &ToolExecutionResult{
			Tool:       ToolConfigureBatch,
			Success:    false,
			Error:      "batch_name or batch_id is required",
			ExecutedAt: time.Now(),
		}, nil
	}

	if destinationOrg == "" && migrationAPI == "" {
		return &ToolExecutionResult{
			Tool:       ToolConfigureBatch,
			Success:    false,
			Error:      "At least one setting must be specified (destination_org or migration_api)",
			ExecutedAt: time.Now(),
		}, nil
	}

	// Find the batch
	batches, err := e.db.ListBatches(ctx)
	if err != nil {
		return &ToolExecutionResult{
			Tool:       ToolConfigureBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to list batches: %v", err),
			ExecutedAt: time.Now(),
		}, nil
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
		return &ToolExecutionResult{
			Tool:       ToolConfigureBatch,
			Success:    false,
			Error:      fmt.Sprintf("Batch not found: %s", searchTerm),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Update batch settings
	changes := make([]string, 0)
	if destinationOrg != "" {
		batch.DestinationOrg = &destinationOrg
		changes = append(changes, fmt.Sprintf("destination organization set to '%s'", destinationOrg))
	}
	if migrationAPI != "" {
		if migrationAPI != models.MigrationAPIGEI && migrationAPI != models.MigrationAPIELM {
			return &ToolExecutionResult{
				Tool:       ToolConfigureBatch,
				Success:    false,
				Error:      fmt.Sprintf("Invalid migration_api '%s'. Must be 'GEI' or 'ELM'", migrationAPI),
				ExecutedAt: time.Now(),
			}, nil
		}
		batch.MigrationAPI = migrationAPI
		changes = append(changes, fmt.Sprintf("migration API set to '%s'", migrationAPI))
	}

	if err := e.db.UpdateBatch(ctx, batch); err != nil {
		return &ToolExecutionResult{
			Tool:       ToolConfigureBatch,
			Success:    false,
			Error:      fmt.Sprintf("Failed to update batch: %v", err),
			ExecutedAt: time.Now(),
		}, nil
	}

	result := map[string]any{
		"batch_id":   batch.ID,
		"batch_name": batch.Name,
		"status":     batch.Status,
	}
	if batch.DestinationOrg != nil {
		result["destination_org"] = *batch.DestinationOrg
	}
	result["migration_api"] = batch.MigrationAPI

	return &ToolExecutionResult{
		Tool:    ToolConfigureBatch,
		Success: true,
		Result:  result,
		Summary: fmt.Sprintf("Batch '%s' updated: %s", batch.Name, strings.Join(changes, ", ")),
		Suggestions: []string{
			"You can now schedule this batch for migration",
		},
		FollowUp: &FollowUpAction{
			Action:      ToolScheduleBatch,
			Description: fmt.Sprintf("Would you like to schedule batch '%s' for migration now?", batch.Name),
			DefaultName: batch.Name,
		},
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

// executeStartMigration starts a migration (dry-run or production) for a batch or repositories
// nolint:gocyclo // Migration starting requires handling multiple scenarios
func (e *ToolExecutor) executeStartMigration(ctx context.Context, args map[string]any, previousResult *ToolExecutionResult) (*ToolExecutionResult, error) {
	// Extract parameters
	batchName := ""
	if v, ok := args["batch_name"].(string); ok {
		batchName = v
	}

	var batchID int64
	if v, ok := args["batch_id"].(string); ok {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			batchID = parsed
		}
	} else if v, ok := args["batch_id"].(float64); ok {
		batchID = int64(v)
	} else if v, ok := args["batch_id"].(int64); ok {
		batchID = v
	}

	repository := ""
	if v, ok := args["repository"].(string); ok {
		repository = v
	}

	// Default to dry-run for safety
	dryRun := true
	if v, ok := args["dry_run"].(bool); ok {
		dryRun = v
	} else if v, ok := args["dry_run"].(string); ok {
		dryRun = v != "false" && v != "no" && v != "0"
	}

	// Try to infer batch from previous result
	if batchName == "" && batchID == 0 && repository == "" && previousResult != nil && previousResult.FollowUp != nil {
		batchName = previousResult.FollowUp.DefaultName
	}

	if batchName == "" && batchID == 0 && repository == "" {
		return &ToolExecutionResult{
			Tool:       ToolStartMigration,
			Success:    false,
			Error:      "At least one of batch_name, batch_id, or repository must be specified",
			ExecutedAt: time.Now(),
		}, nil
	}

	targetStatus := models.StatusQueuedForMigration
	if dryRun {
		targetStatus = models.StatusDryRunQueued
	}

	var queuedRepos []map[string]any
	var batch *models.Batch
	skippedCount := 0

	// Handle batch migration
	if batchName != "" || batchID != 0 {
		batches, err := e.db.ListBatches(ctx)
		if err != nil {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Failed to list batches: %v", err),
				ExecutedAt: time.Now(),
			}, nil
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
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Batch not found: %s", searchTerm),
				ExecutedAt: time.Now(),
			}, nil
		}

		if batch.Status == models.BatchStatusInProgress {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Batch '%s' is already running", batch.Name),
				ExecutedAt: time.Now(),
			}, nil
		}

		// Get batch repositories
		repos, err := e.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Failed to get batch repositories: %v", err),
				ExecutedAt: time.Now(),
			}, nil
		}

		if len(repos) == 0 {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Batch '%s' has no repositories", batch.Name),
				ExecutedAt: time.Now(),
			}, nil
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
		if err := e.db.UpdateBatch(ctx, batch); err != nil {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Failed to update batch status: %v", err),
				ExecutedAt: time.Now(),
			}, nil
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
				if err := e.db.UpdateRepository(ctx, repo); err != nil {
					if e.logger != nil {
						e.logger.Error("Failed to queue repository", "repo", repo.FullName, "error", err)
					}
					continue
				}
				queuedRepos = append(queuedRepos, map[string]any{
					"full_name": repo.FullName,
					"status":    repo.Status,
				})
			} else {
				skippedCount++
			}
		}
	}

	// Handle single repository
	if repository != "" {
		repo, err := e.db.GetRepository(ctx, repository)
		if err != nil || repo == nil {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Repository not found: %s", repository),
				ExecutedAt: time.Now(),
			}, nil
		}

		if !canQueueForMigration(repo.Status, dryRun) {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Repository '%s' cannot be queued for migration (status: %s)", repository, repo.Status),
				ExecutedAt: time.Now(),
			}, nil
		}

		repo.Status = string(targetStatus)
		if err := e.db.UpdateRepository(ctx, repo); err != nil {
			return &ToolExecutionResult{
				Tool:       ToolStartMigration,
				Success:    false,
				Error:      fmt.Sprintf("Failed to queue repository: %v", err),
				ExecutedAt: time.Now(),
			}, nil
		}
		queuedRepos = append(queuedRepos, map[string]any{
			"full_name": repo.FullName,
			"status":    repo.Status,
		})
	}

	if len(queuedRepos) == 0 {
		return &ToolExecutionResult{
			Tool:       ToolStartMigration,
			Success:    false,
			Error:      "No repositories could be queued for migration",
			ExecutedAt: time.Now(),
		}, nil
	}

	migrationType := "production migration"
	if dryRun {
		migrationType = "dry-run"
	}

	result := map[string]any{
		"queued_count":  len(queuedRepos),
		"skipped_count": skippedCount,
		"dry_run":       dryRun,
		"repositories":  queuedRepos,
	}
	if batch != nil {
		result["batch_id"] = batch.ID
		result["batch_name"] = batch.Name
	}

	suggestions := []string{
		"Monitor progress with get_migration_progress",
	}
	if dryRun {
		suggestions = append(suggestions, "After dry-run completes, start production migration with start_migration(dry_run=false)")
	}

	summary := fmt.Sprintf("Started %s for %d repositories", migrationType, len(queuedRepos))
	if batch != nil {
		summary = fmt.Sprintf("Started %s for batch '%s' (%d repositories)", migrationType, batch.Name, len(queuedRepos))
	}

	return &ToolExecutionResult{
		Tool:        ToolStartMigration,
		Success:     true,
		Result:      result,
		Summary:     summary,
		Suggestions: suggestions,
		FollowUp: &FollowUpAction{
			Action:      ToolGetMigrationProgress,
			Description: "Check migration progress",
			DefaultName: func() string {
				if batch != nil {
					return batch.Name
				}
				return repository
			}(),
		},
		ExecutedAt: time.Now(),
	}, nil
}

// executeCancelMigration cancels a running migration
// nolint:gocyclo // Cancellation requires handling multiple scenarios (batch, single repo, validation)
func (e *ToolExecutor) executeCancelMigration(ctx context.Context, args map[string]any, previousResult *ToolExecutionResult) (*ToolExecutionResult, error) {
	batchName := ""
	if v, ok := args["batch_name"].(string); ok {
		batchName = v
	}

	var batchID int64
	if v, ok := args["batch_id"].(string); ok {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			batchID = parsed
		}
	} else if v, ok := args["batch_id"].(float64); ok {
		batchID = int64(v)
	}

	repository := ""
	if v, ok := args["repository"].(string); ok {
		repository = v
	}

	// Try to infer from previous result
	if batchName == "" && batchID == 0 && repository == "" && previousResult != nil && previousResult.FollowUp != nil {
		batchName = previousResult.FollowUp.DefaultName
	}

	if batchName == "" && batchID == 0 && repository == "" {
		return &ToolExecutionResult{
			Tool:       ToolCancelMigration,
			Success:    false,
			Error:      "At least one of batch_name, batch_id, or repository must be specified",
			ExecutedAt: time.Now(),
		}, nil
	}

	cancelledCount := 0

	// Handle batch cancellation
	if batchName != "" || batchID != 0 {
		batches, err := e.db.ListBatches(ctx)
		if err != nil {
			return &ToolExecutionResult{
				Tool:       ToolCancelMigration,
				Success:    false,
				Error:      fmt.Sprintf("Failed to list batches: %v", err),
				ExecutedAt: time.Now(),
			}, nil
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
			return &ToolExecutionResult{
				Tool:       ToolCancelMigration,
				Success:    false,
				Error:      fmt.Sprintf("Batch not found: %s", searchTerm),
				ExecutedAt: time.Now(),
			}, nil
		}

		// Get batch repositories
		repos, err := e.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return &ToolExecutionResult{
				Tool:       ToolCancelMigration,
				Success:    false,
				Error:      fmt.Sprintf("Failed to get batch repositories: %v", err),
				ExecutedAt: time.Now(),
			}, nil
		}

		// Cancel queued repositories
		for _, repo := range repos {
			if isInQueuedOrInProgressState(repo.Status) {
				repo.Status = StatusPending
				if err := e.db.UpdateRepository(ctx, repo); err != nil {
					if e.logger != nil {
						e.logger.Error("Failed to cancel repository", "repo", repo.FullName, "error", err)
					}
					continue
				}
				cancelledCount++
			}
		}

		// Update batch status
		batch.Status = models.BatchStatusCancelled
		if err := e.db.UpdateBatch(ctx, batch); err != nil {
			if e.logger != nil {
				e.logger.Error("Failed to update batch status", "batch", batch.Name, "error", err)
			}
		}

		return &ToolExecutionResult{
			Tool:    ToolCancelMigration,
			Success: true,
			Result: map[string]any{
				"batch_id":        batch.ID,
				"batch_name":      batch.Name,
				"cancelled_count": cancelledCount,
			},
			Summary:    fmt.Sprintf("Cancelled batch '%s' (%d repositories)", batch.Name, cancelledCount),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Handle single repository cancellation
	if repository != "" {
		repo, err := e.db.GetRepository(ctx, repository)
		if err != nil || repo == nil {
			return &ToolExecutionResult{
				Tool:       ToolCancelMigration,
				Success:    false,
				Error:      fmt.Sprintf("Repository not found: %s", repository),
				ExecutedAt: time.Now(),
			}, nil
		}

		if !isInQueuedOrInProgressState(repo.Status) {
			return &ToolExecutionResult{
				Tool:       ToolCancelMigration,
				Success:    false,
				Error:      fmt.Sprintf("Repository '%s' is not in a cancellable state (status: %s)", repository, repo.Status),
				ExecutedAt: time.Now(),
			}, nil
		}

		repo.Status = StatusPending
		if err := e.db.UpdateRepository(ctx, repo); err != nil {
			return &ToolExecutionResult{
				Tool:       ToolCancelMigration,
				Success:    false,
				Error:      fmt.Sprintf("Failed to cancel repository: %v", err),
				ExecutedAt: time.Now(),
			}, nil
		}

		return &ToolExecutionResult{
			Tool:    ToolCancelMigration,
			Success: true,
			Result: map[string]any{
				"repository":      repository,
				"cancelled_count": 1,
			},
			Summary:    fmt.Sprintf("Cancelled migration for repository '%s'", repository),
			ExecutedAt: time.Now(),
		}, nil
	}

	return &ToolExecutionResult{
		Tool:       ToolCancelMigration,
		Success:    false,
		Error:      "No target specified for cancellation",
		ExecutedAt: time.Now(),
	}, nil
}

// executeGetMigrationProgress gets the progress of a running migration
func (e *ToolExecutor) executeGetMigrationProgress(ctx context.Context, args map[string]any, previousResult *ToolExecutionResult) (*ToolExecutionResult, error) {
	batchName := ""
	if v, ok := args["batch_name"].(string); ok {
		batchName = v
	}

	var batchID int64
	if v, ok := args["batch_id"].(string); ok {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			batchID = parsed
		}
	} else if v, ok := args["batch_id"].(float64); ok {
		batchID = int64(v)
	}

	repository := ""
	if v, ok := args["repository"].(string); ok {
		repository = v
	}

	// Try to infer from previous result
	if batchName == "" && batchID == 0 && repository == "" && previousResult != nil && previousResult.FollowUp != nil {
		batchName = previousResult.FollowUp.DefaultName
	}

	// Handle single repository progress
	if repository != "" {
		repo, err := e.db.GetRepository(ctx, repository)
		if err != nil || repo == nil {
			return &ToolExecutionResult{
				Tool:       ToolGetMigrationProgress,
				Success:    false,
				Error:      fmt.Sprintf("Repository not found: %s", repository),
				ExecutedAt: time.Now(),
			}, nil
		}

		progress := calculateProgress([]string{repo.Status})

		return &ToolExecutionResult{
			Tool:    ToolGetMigrationProgress,
			Success: true,
			Result: map[string]any{
				"repository": repository,
				"status":     repo.Status,
				"progress":   progress,
			},
			Summary:    fmt.Sprintf("Repository '%s' status: %s", repository, repo.Status),
			ExecutedAt: time.Now(),
		}, nil
	}

	// Handle batch progress
	if batchName != "" || batchID != 0 {
		batches, err := e.db.ListBatches(ctx)
		if err != nil {
			return &ToolExecutionResult{
				Tool:       ToolGetMigrationProgress,
				Success:    false,
				Error:      fmt.Sprintf("Failed to list batches: %v", err),
				ExecutedAt: time.Now(),
			}, nil
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
			return &ToolExecutionResult{
				Tool:       ToolGetMigrationProgress,
				Success:    false,
				Error:      fmt.Sprintf("Batch not found: %s", searchTerm),
				ExecutedAt: time.Now(),
			}, nil
		}

		// Get batch repositories
		repos, err := e.db.ListRepositories(ctx, map[string]any{
			"batch_id": batch.ID,
		})
		if err != nil {
			return &ToolExecutionResult{
				Tool:       ToolGetMigrationProgress,
				Success:    false,
				Error:      fmt.Sprintf("Failed to get batch repositories: %v", err),
				ExecutedAt: time.Now(),
			}, nil
		}

		statuses := make([]string, len(repos))
		repoDetails := make([]map[string]any, len(repos))
		for i, repo := range repos {
			statuses[i] = repo.Status
			repoDetails[i] = map[string]any{
				"full_name": repo.FullName,
				"status":    repo.Status,
			}
		}

		progress := calculateProgress(statuses)

		return &ToolExecutionResult{
			Tool:    ToolGetMigrationProgress,
			Success: true,
			Result: map[string]any{
				"batch_id":     batch.ID,
				"batch_name":   batch.Name,
				"batch_status": batch.Status,
				"progress":     progress,
				"repositories": repoDetails,
			},
			Summary: fmt.Sprintf("Batch '%s': %d/%d complete (%.1f%%)",
				batch.Name, progress["completed_count"], progress["total_count"], progress["percent_complete"]),
			ExecutedAt: time.Now(),
		}, nil
	}

	return &ToolExecutionResult{
		Tool:       ToolGetMigrationProgress,
		Success:    false,
		Error:      "At least one of batch_name, batch_id, or repository must be specified",
		ExecutedAt: time.Now(),
	}, nil
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

// calculateProgress calculates progress metrics from a list of statuses
func calculateProgress(statuses []string) map[string]any {
	progress := map[string]any{
		"total_count":       len(statuses),
		"pending_count":     0,
		"queued_count":      0,
		"in_progress_count": 0,
		"completed_count":   0,
		"failed_count":      0,
		"skipped_count":     0,
		"percent_complete":  0.0,
	}

	for _, status := range statuses {
		switch models.MigrationStatus(status) {
		case models.StatusPending:
			progress["pending_count"] = progress["pending_count"].(int) + 1
		case models.StatusDryRunQueued, models.StatusQueuedForMigration:
			progress["queued_count"] = progress["queued_count"].(int) + 1
		case models.StatusDryRunInProgress, models.StatusMigratingContent,
			models.StatusArchiveGenerating, models.StatusPreMigration, models.StatusPostMigration:
			progress["in_progress_count"] = progress["in_progress_count"].(int) + 1
		case models.StatusDryRunComplete, models.StatusMigrationComplete, models.StatusComplete:
			progress["completed_count"] = progress["completed_count"].(int) + 1
		case models.StatusDryRunFailed, models.StatusMigrationFailed:
			progress["failed_count"] = progress["failed_count"].(int) + 1
		case models.StatusWontMigrate:
			progress["skipped_count"] = progress["skipped_count"].(int) + 1
		}
	}

	total := progress["total_count"].(int)
	if total > 0 {
		completed := progress["completed_count"].(int)
		progress["percent_complete"] = float64(completed) / float64(total) * 100
	}

	return progress
}
