package batch

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// Organizer handles batch organization and pilot selection
type Organizer struct {
	storage *storage.Database
	logger  *slog.Logger
}

// OrganizerConfig holds configuration for the batch organizer
type OrganizerConfig struct {
	Storage *storage.Database
	Logger  *slog.Logger
}

// NewOrganizer creates a new batch organizer
func NewOrganizer(cfg OrganizerConfig) (*Organizer, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &Organizer{
		storage: cfg.Storage,
		logger:  cfg.Logger,
	}, nil
}

// PilotCriteria defines criteria for selecting pilot repositories
type PilotCriteria struct {
	// MinSize minimum repository size in KB
	MinSize int64
	// MaxSize maximum repository size in KB
	MaxSize int64
	// RequireLFS whether to require LFS
	RequireLFS bool
	// RequireSubmodules whether to require submodules
	RequireSubmodules bool
	// RequireActions whether to require GitHub Actions
	RequireActions bool
	// RequireWiki whether to require wiki
	RequireWiki bool
	// RequirePages whether to require GitHub Pages
	RequirePages bool
	// MaxCount maximum number of pilot repos to select
	MaxCount int
	// Organizations specific organizations to include (empty = all)
	Organizations []string
}

// DefaultPilotCriteria returns sensible defaults for pilot selection
func DefaultPilotCriteria() PilotCriteria {
	return PilotCriteria{
		MinSize:           100,      // 100 KB minimum
		MaxSize:           10485760, // 10 GB maximum
		RequireLFS:        false,
		RequireSubmodules: false,
		RequireActions:    false,
		RequireWiki:       false,
		RequirePages:      false,
		MaxCount:          10,
		Organizations:     []string{},
	}
}

// SelectPilotRepositories intelligently selects repositories for pilot migration
// It aims to select a diverse set of repositories that represent different
// characteristics (sizes, features) for thorough testing
func (o *Organizer) SelectPilotRepositories(ctx context.Context, criteria PilotCriteria) ([]*models.Repository, error) {
	o.logger.Info("Starting pilot repository selection", "criteria", criteria)

	// Get all pending repositories
	repos, err := o.storage.ListRepositories(ctx, map[string]interface{}{
		"status": models.StatusPending,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		o.logger.Warn("No pending repositories found for pilot selection")
		return []*models.Repository{}, nil
	}

	o.logger.Info("Found pending repositories", "count", len(repos))

	// Filter repositories based on criteria
	candidates := make([]*models.Repository, 0, len(repos))
	for _, repo := range repos {
		if !o.matchesCriteria(repo, criteria) {
			continue
		}
		candidates = append(candidates, repo)
	}

	o.logger.Info("Filtered candidates", "count", len(candidates))

	if len(candidates) == 0 {
		return []*models.Repository{}, nil
	}

	// Score and rank candidates for diversity
	scored := o.scoreRepositories(candidates)

	// Sort by score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Select top N repositories ensuring diversity
	selected := o.selectDiverse(scored, criteria.MaxCount)

	o.logger.Info("Selected pilot repositories", "count", len(selected))

	return selected, nil
}

// scoredRepo holds a repository and its diversity score
type scoredRepo struct {
	Repo  *models.Repository
	Score float64
}

// matchesCriteria checks if a repository matches the pilot criteria
func (o *Organizer) matchesCriteria(repo *models.Repository, criteria PilotCriteria) bool {
	return o.matchesSize(repo, criteria) &&
		o.matchesOrganization(repo, criteria) &&
		o.matchesFeatures(repo, criteria)
}

// matchesSize checks if repository size is within criteria range
func (o *Organizer) matchesSize(repo *models.Repository, criteria PilotCriteria) bool {
	size := int64(0)
	if repo.TotalSize != nil {
		size = *repo.TotalSize / 1024 // Convert bytes to KB
	}
	return size >= criteria.MinSize && size <= criteria.MaxSize
}

// matchesOrganization checks if repository org matches criteria
func (o *Organizer) matchesOrganization(repo *models.Repository, criteria PilotCriteria) bool {
	if len(criteria.Organizations) == 0 {
		return true
	}

	repoOrg := repo.Organization()
	for _, org := range criteria.Organizations {
		if org == repoOrg {
			return true
		}
	}
	return false
}

// matchesFeatures checks if repository has required features
func (o *Organizer) matchesFeatures(repo *models.Repository, criteria PilotCriteria) bool {
	if criteria.RequireLFS && !repo.HasLFS {
		return false
	}
	if criteria.RequireSubmodules && !repo.HasSubmodules {
		return false
	}
	if criteria.RequireActions && !repo.HasActions {
		return false
	}
	if criteria.RequireWiki && !repo.HasWiki {
		return false
	}
	if criteria.RequirePages && !repo.HasPages {
		return false
	}
	return true
}

// scoreRepositories assigns diversity scores to repositories
// Higher scores indicate repositories that are more valuable for pilot testing
func (o *Organizer) scoreRepositories(repos []*models.Repository) []scoredRepo {
	scored := make([]scoredRepo, 0, len(repos))

	for _, repo := range repos {
		score := 0.0

		// Size diversity (prefer medium-sized repos)
		// Score based on logarithmic scale
		sizeScore := 0.0
		if repo.TotalSize != nil && *repo.TotalSize > 0 {
			sizeKB := *repo.TotalSize / 1024
			// Prefer repos around 100MB (102400 KB)
			targetSize := 102400.0
			sizeRatio := float64(sizeKB) / targetSize
			if sizeRatio > 1 {
				sizeScore = 1.0 / sizeRatio
			} else {
				sizeScore = sizeRatio
			}
			score += sizeScore * 10
		}

		// Feature diversity (more features = higher score)
		if repo.HasLFS {
			score += 5
		}
		if repo.HasSubmodules {
			score += 5
		}
		if repo.HasActions {
			score += 8 // Actions are important to test
		}
		if repo.HasWiki {
			score += 3
		}
		if repo.HasPages {
			score += 3
		}
		if repo.HasProjects {
			score += 2
		}

		// Commit count (prefer repos with reasonable activity)
		if repo.CommitCount > 10 && repo.CommitCount < 10000 {
			score += 5
		}

		// Branch count (prefer repos with multiple branches)
		if repo.BranchCount > 1 {
			score += float64(repo.BranchCount) * 0.5
		}

		// Protection rules (important to test)
		if repo.BranchProtections > 0 {
			score += 7
		}

		scored = append(scored, scoredRepo{
			Repo:  repo,
			Score: score,
		})
	}

	return scored
}

// selectDiverse selects repositories ensuring feature diversity
func (o *Organizer) selectDiverse(scored []scoredRepo, maxCount int) []*models.Repository {
	if len(scored) <= maxCount {
		result := make([]*models.Repository, len(scored))
		for i, s := range scored {
			result[i] = s.Repo
		}
		return result
	}

	selected := make([]*models.Repository, 0, maxCount)
	features := make(map[string]bool)

	// First pass: select repos with unique feature combinations
	for _, s := range scored {
		if len(selected) >= maxCount {
			break
		}

		featureKey := o.getFeatureKey(s.Repo)
		if !features[featureKey] {
			selected = append(selected, s.Repo)
			features[featureKey] = true
		}
	}

	// Second pass: fill remaining slots with highest scoring repos
	for _, s := range scored {
		if len(selected) >= maxCount {
			break
		}

		alreadySelected := false
		for _, sel := range selected {
			if sel.FullName == s.Repo.FullName {
				alreadySelected = true
				break
			}
		}

		if !alreadySelected {
			selected = append(selected, s.Repo)
		}
	}

	return selected
}

// getFeatureKey creates a key representing the feature combination of a repo
func (o *Organizer) getFeatureKey(repo *models.Repository) string {
	return fmt.Sprintf("%t_%t_%t_%t_%t",
		repo.HasLFS,
		repo.HasSubmodules,
		repo.HasActions,
		repo.HasWiki,
		repo.HasPages,
	)
}

// CreatePilotBatch creates a new pilot batch with selected repositories
func (o *Organizer) CreatePilotBatch(ctx context.Context, name string, criteria PilotCriteria) (*models.Batch, []*models.Repository, error) {
	o.logger.Info("Creating pilot batch", "name", name)

	// Select pilot repositories
	repos, err := o.SelectPilotRepositories(ctx, criteria)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to select pilot repositories: %w", err)
	}

	if len(repos) == 0 {
		return nil, nil, fmt.Errorf("no repositories match pilot criteria")
	}

	// Create batch
	batch := &models.Batch{
		Name:            name,
		Description:     strPtr(fmt.Sprintf("Pilot batch with %d repositories for initial migration testing", len(repos))),
		Type:            "pilot",
		RepositoryCount: len(repos),
		Status:          "ready",
		CreatedAt:       time.Now(),
	}

	if err := o.storage.CreateBatch(ctx, batch); err != nil {
		return nil, nil, fmt.Errorf("failed to create batch: %w", err)
	}

	o.logger.Info("Created pilot batch", "batch_id", batch.ID, "repo_count", len(repos))

	// Assign repositories to batch
	for _, repo := range repos {
		repo.BatchID = &batch.ID
		repo.Priority = 1 // Pilot repos get high priority
		if err := o.storage.UpdateRepository(ctx, repo); err != nil {
			o.logger.Error("Failed to assign repository to batch",
				"repo", repo.FullName,
				"batch_id", batch.ID,
				"error", err)
		}
	}

	return batch, repos, nil
}

// WaveCriteria defines criteria for organizing repositories into waves
type WaveCriteria struct {
	// WaveSize target number of repositories per wave
	WaveSize int
	// GroupByOrganization whether to keep organizations together
	GroupByOrganization bool
	// SortBy how to sort repos ("size", "name", "org")
	SortBy string
}

// DefaultWaveCriteria returns sensible defaults for wave organization
func DefaultWaveCriteria() WaveCriteria {
	return WaveCriteria{
		WaveSize:            50,
		GroupByOrganization: true,
		SortBy:              "org",
	}
}

// OrganizeIntoWaves organizes pending repositories into migration waves
func (o *Organizer) OrganizeIntoWaves(ctx context.Context, criteria WaveCriteria) ([]*models.Batch, error) {
	o.logger.Info("Organizing repositories into waves", "criteria", criteria)

	// Get all pending repositories (not in any batch)
	repos, err := o.storage.ListRepositories(ctx, map[string]interface{}{
		"status": models.StatusPending,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Filter out repos that already have a batch
	var unbatched []*models.Repository
	for _, repo := range repos {
		if repo.BatchID == nil {
			unbatched = append(unbatched, repo)
		}
	}

	o.logger.Info("Found unbatched repositories", "count", len(unbatched))

	if len(unbatched) == 0 {
		return []*models.Batch{}, nil
	}

	// Sort repositories based on criteria
	o.sortRepositories(unbatched, criteria)

	// Group into waves
	var waves []*models.Batch
	waveNum := 1

	for i := 0; i < len(unbatched); i += criteria.WaveSize {
		end := i + criteria.WaveSize
		if end > len(unbatched) {
			end = len(unbatched)
		}

		waveRepos := unbatched[i:end]

		// Create wave batch
		batch := &models.Batch{
			Name:            fmt.Sprintf("Wave %d", waveNum),
			Description:     strPtr(fmt.Sprintf("Migration wave %d with %d repositories", waveNum, len(waveRepos))),
			Type:            fmt.Sprintf("wave_%d", waveNum),
			RepositoryCount: len(waveRepos),
			Status:          "ready",
			CreatedAt:       time.Now(),
		}

		if err := o.storage.CreateBatch(ctx, batch); err != nil {
			o.logger.Error("Failed to create wave batch", "wave", waveNum, "error", err)
			continue
		}

		o.logger.Info("Created wave batch", "wave", waveNum, "batch_id", batch.ID, "repo_count", len(waveRepos))

		// Assign repositories to wave
		for _, repo := range waveRepos {
			repo.BatchID = &batch.ID
			repo.Priority = 0 // Normal priority
			if err := o.storage.UpdateRepository(ctx, repo); err != nil {
				o.logger.Error("Failed to assign repository to wave",
					"repo", repo.FullName,
					"wave", waveNum,
					"error", err)
			}
		}

		waves = append(waves, batch)
		waveNum++
	}

	o.logger.Info("Successfully organized waves", "wave_count", len(waves), "total_repos", len(unbatched))

	return waves, nil
}

// sortRepositories sorts repos based on criteria
func (o *Organizer) sortRepositories(repos []*models.Repository, criteria WaveCriteria) {
	switch criteria.SortBy {
	case "size":
		sort.Slice(repos, func(i, j int) bool {
			sizeI := int64(0)
			if repos[i].TotalSize != nil {
				sizeI = *repos[i].TotalSize
			}
			sizeJ := int64(0)
			if repos[j].TotalSize != nil {
				sizeJ = *repos[j].TotalSize
			}
			return sizeI < sizeJ
		})
	case "name":
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].FullName < repos[j].FullName
		})
	case "org":
		// Sort by organization first, then by name within org
		sort.Slice(repos, func(i, j int) bool {
			orgI := repos[i].Organization()
			orgJ := repos[j].Organization()
			if orgI == orgJ {
				return repos[i].Name() < repos[j].Name()
			}
			return orgI < orgJ
		})
	default:
		// Default to org sorting
		sort.Slice(repos, func(i, j int) bool {
			orgI := repos[i].Organization()
			orgJ := repos[j].Organization()
			if orgI == orgJ {
				return repos[i].Name() < repos[j].Name()
			}
			return orgI < orgJ
		})
	}
}

// GetBatchProgress returns progress statistics for a batch
func (o *Organizer) GetBatchProgress(ctx context.Context, batchID int64) (*BatchProgress, error) {
	// Get batch
	batch, err := o.storage.GetBatch(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return nil, fmt.Errorf("batch not found")
	}

	// Get all repositories in batch
	repos, err := o.storage.ListRepositories(ctx, map[string]interface{}{
		"batch_id": batchID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list batch repositories: %w", err)
	}

	// Calculate statistics
	progress := &BatchProgress{
		BatchID:         batchID,
		BatchName:       batch.Name,
		BatchType:       batch.Type,
		BatchStatus:     batch.Status,
		TotalRepos:      len(repos),
		StatusCounts:    make(map[string]int),
		StartedAt:       batch.StartedAt,
		CompletedAt:     batch.CompletedAt,
		EstimatedTimeMS: 0,
	}

	completed := 0
	failed := 0

	for _, repo := range repos {
		progress.StatusCounts[repo.Status]++

		if repo.Status == string(models.StatusComplete) {
			completed++
		}
		if repo.Status == string(models.StatusMigrationFailed) || repo.Status == string(models.StatusDryRunFailed) {
			failed++
		}
	}

	progress.CompletedRepos = completed
	progress.FailedRepos = failed
	progress.InProgressRepos = len(repos) - completed - failed

	// Calculate percentage
	if len(repos) > 0 {
		progress.PercentComplete = float64(completed) / float64(len(repos)) * 100
	}

	// Calculate duration if started
	if batch.StartedAt != nil {
		endTime := time.Now()
		if batch.CompletedAt != nil {
			endTime = *batch.CompletedAt
		}
		duration := endTime.Sub(*batch.StartedAt)
		progress.DurationMS = duration.Milliseconds()

		// Estimate remaining time based on current rate
		if completed > 0 && progress.InProgressRepos > 0 {
			avgTimePerRepo := duration.Milliseconds() / int64(completed)
			progress.EstimatedTimeMS = avgTimePerRepo * int64(progress.InProgressRepos)
		}
	}

	return progress, nil
}

// BatchProgress represents the current progress of a batch
type BatchProgress struct {
	BatchID         int64          `json:"batch_id"`
	BatchName       string         `json:"batch_name"`
	BatchType       string         `json:"batch_type"`
	BatchStatus     string         `json:"batch_status"`
	TotalRepos      int            `json:"total_repos"`
	CompletedRepos  int            `json:"completed_repos"`
	InProgressRepos int            `json:"in_progress_repos"`
	FailedRepos     int            `json:"failed_repos"`
	PercentComplete float64        `json:"percent_complete"`
	StatusCounts    map[string]int `json:"status_counts"`
	DurationMS      int64          `json:"duration_ms"`
	EstimatedTimeMS int64          `json:"estimated_time_ms"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}
