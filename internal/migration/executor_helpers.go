package migration

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/shurcooL/githubv4"
)

// getDestinationOrg returns the destination org for a repository
// Precedence: repo.DestinationFullName > batch.DestinationOrg > source org
func (e *Executor) getDestinationOrg(repo *models.Repository, batch *models.Batch) string {
	// Priority 1: If DestinationFullName is set, extract org from it
	if repo.DestinationFullName != nil && *repo.DestinationFullName != "" {
		parts := strings.Split(*repo.DestinationFullName, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	// Priority 2: If batch has a destination org, use it
	if batch != nil && batch.DestinationOrg != nil && *batch.DestinationOrg != "" {
		return *batch.DestinationOrg
	}

	// Priority 3: Default to source org
	parts := strings.Split(repo.FullName, "/")
	if len(parts) >= 1 {
		return parts[0]
	}

	return ""
}

// getDestinationRepoName returns the destination repository name for a repository
// Defaults to the source repo name if not explicitly set
func (e *Executor) getDestinationRepoName(repo *models.Repository) string {
	// If DestinationFullName is set, extract repo name from it
	if repo.DestinationFullName != nil && *repo.DestinationFullName != "" {
		parts := strings.Split(*repo.DestinationFullName, "/")
		if len(parts) >= 2 {
			return sanitizeRepoName(parts[1])
		}
		// If only one part, return it as the repo name
		if len(parts) == 1 {
			return sanitizeRepoName(parts[0])
		}
	}

	// For ADO repos, extract ONLY the repository name (last part)
	// ADO full_name format: org/project/repo -> we want just "repo"
	if repo.ADOProject != nil && *repo.ADOProject != "" {
		parts := strings.Split(repo.FullName, "/")
		if len(parts) >= 3 {
			// Return sanitized repo name (last part)
			return sanitizeRepoName(parts[len(parts)-1])
		}
	}

	// Default to source repo name (works for GitHub org/repo format)
	return sanitizeRepoName(repo.Name())
}

// sanitizeRepoName replaces spaces with hyphens for GitHub compatibility
func sanitizeRepoName(name string) string {
	return strings.ReplaceAll(name, " ", "-")
}

// shouldExcludeReleases determines whether to exclude releases during migration
// Precedence: repo.ExcludeReleases OR batch.ExcludeReleases (either can enable it)
func (e *Executor) shouldExcludeReleases(repo *models.Repository, batch *models.Batch) bool {
	// If repo explicitly excludes releases, honor it
	if repo.ExcludeReleases {
		return true
	}

	// If batch excludes releases, apply it
	if batch != nil && batch.ExcludeReleases {
		return true
	}

	return false
}

// shouldExcludeAttachments determines whether to exclude attachments during migration
// Precedence: repo.ExcludeAttachments OR batch.ExcludeAttachments (either can enable it)
func (e *Executor) shouldExcludeAttachments(repo *models.Repository, batch *models.Batch) bool {
	// If repo explicitly excludes attachments, honor it
	if repo.ExcludeAttachments {
		return true
	}

	// Check batch-level setting if available
	if batch != nil && batch.ExcludeAttachments {
		return true
	}

	return false
}

// determineTargetVisibility determines the target visibility based on source visibility and config
func (e *Executor) determineTargetVisibility(sourceVisibility string) string {
	switch strings.ToLower(sourceVisibility) {
	case visibilityPublic:
		// Apply configured mapping for public repos
		targetVis := strings.ToLower(e.visibilityHandling.PublicRepos)
		// Validate target visibility
		if targetVis == visibilityPublic || targetVis == visibilityInternal || targetVis == visibilityPrivate {
			return targetVis
		}
		// Default to private if invalid
		e.logger.Warn("Invalid target visibility for public repos, defaulting to private",
			"configured", e.visibilityHandling.PublicRepos)
		return visibilityPrivate

	case visibilityInternal:
		// Apply configured mapping for internal repos
		targetVis := strings.ToLower(e.visibilityHandling.InternalRepos)
		// Validate target visibility (internal repos can only become internal or private)
		if targetVis == visibilityInternal || targetVis == visibilityPrivate {
			return targetVis
		}
		// Default to private if invalid
		e.logger.Warn("Invalid target visibility for internal repos, defaulting to private",
			"configured", e.visibilityHandling.InternalRepos)
		return visibilityPrivate

	case visibilityPrivate:
		// Private repos always stay private
		return visibilityPrivate

	default:
		// Unknown visibility, default to private (safest)
		e.logger.Warn("Unknown source visibility, defaulting to private",
			"source_visibility", sourceVisibility)
		return visibilityPrivate
	}
}

// shouldRunPostMigration determines if post-migration tasks should run
func (e *Executor) shouldRunPostMigration(dryRun bool) bool {
	switch e.postMigrationMode {
	case PostMigrationNever:
		return false
	case PostMigrationProductionOnly:
		return !dryRun
	case PostMigrationDryRunOnly:
		return dryRun
	case PostMigrationAlways:
		return true
	default:
		// Default to production only
		return !dryRun
	}
}

// getOrFetchDestOrgID returns the destination org ID for a given org name, fetching it if not cached
func (e *Executor) getOrFetchDestOrgID(ctx context.Context, orgName string) (string, error) {
	if orgName == "" {
		return "", fmt.Errorf("organization name is required")
	}

	// Check cache first
	if orgID, exists := e.orgIDCache[orgName]; exists {
		return orgID, nil
	}

	e.logger.Info("Fetching destination organization ID", "org", orgName)

	// GraphQL query to get organization ID
	var query struct {
		Organization struct {
			ID string
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(orgName),
	}

	if err := e.destClient.QueryWithRetry(ctx, "GetOrganizationID", &query, variables); err != nil {
		return "", fmt.Errorf("failed to fetch organization ID for %s: %w", orgName, err)
	}

	orgID := query.Organization.ID
	e.orgIDCache[orgName] = orgID // Cache it

	e.logger.Info("Fetched destination organization ID",
		"org", orgName,
		"org_id", orgID)

	return orgID, nil
}

// getOrCreateMigrationSource returns the migration source ID, creating it if not cached
func (e *Executor) getOrCreateMigrationSource(ctx context.Context, ownerID string) (string, error) {
	// Check if we already have a migration source for this owner
	if migSourceID, exists := e.migSourceCache[ownerID]; exists {
		e.logger.Debug("Using cached migration source", "owner_id", ownerID, "source_id", migSourceID)
		return migSourceID, nil
	}

	e.logger.Info("Creating migration source for destination organization", "owner_id", ownerID)

	// Get the source URL from the source client
	sourceURL := e.sourceClient.BaseURL()

	// GraphQL mutation to create migration source
	var mutation struct {
		CreateMigrationSource struct {
			MigrationSource struct {
				ID   githubv4.String
				Name githubv4.String
				URL  githubv4.String
				Type githubv4.String
			}
		} `graphql:"createMigrationSource(input: $input)"`
	}

	// Create string pointer for URL
	urlPtr := githubv4.String(sourceURL)

	// Use typed input struct
	// Note: GitHubPat is set to nil because archive URLs are pre-signed S3/blob storage URLs
	// that don't require authentication
	input := githubv4.CreateMigrationSourceInput{
		Name:      githubv4.String(fmt.Sprintf("Migration from %s", sourceURL)),
		URL:       &urlPtr,
		OwnerID:   githubv4.ID(ownerID),
		Type:      githubv4.MigrationSourceTypeGitHubArchive,
		GitHubPat: nil, // Not needed for archive-based migrations with pre-signed URLs
	}

	if err := e.destClient.MutateWithRetry(ctx, "CreateMigrationSource", &mutation, input, nil); err != nil {
		return "", fmt.Errorf("failed to create migration source: %w", err)
	}

	migSourceID := string(mutation.CreateMigrationSource.MigrationSource.ID)

	// Cache it for this owner
	e.migSourceCache[ownerID] = migSourceID

	e.logger.Info("Created migration source",
		"owner_id", ownerID,
		"source_id", migSourceID,
		"source_url", sourceURL,
		"name", string(mutation.CreateMigrationSource.MigrationSource.Name),
		"type", string(mutation.CreateMigrationSource.MigrationSource.Type))

	return migSourceID, nil
}

// createMigrationHistory creates a migration history record
func (e *Executor) createMigrationHistory(ctx context.Context, repo *models.Repository, dryRun bool) (*int64, error) {
	phase := "migration"
	if dryRun {
		phase = "dry_run"
	}

	history := &models.MigrationHistory{
		RepositoryID: repo.ID,
		Status:       "in_progress",
		Phase:        phase,
		StartedAt:    time.Now(),
	}

	id, err := e.storage.CreateMigrationHistory(ctx, history)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

// updateHistoryStatus updates migration history status
func (e *Executor) updateHistoryStatus(ctx context.Context, historyID *int64, status string, errorMsg *string) {
	if historyID == nil {
		return
	}

	if err := e.storage.UpdateMigrationHistory(ctx, *historyID, status, errorMsg); err != nil {
		e.logger.Error("Failed to update migration history", "error", err)
	}
}

// logOperation logs a migration operation
func (e *Executor) logOperation(ctx context.Context, repo *models.Repository, historyID *int64, level, phase, operation, message string, details *string) {
	log := &models.MigrationLog{
		RepositoryID: repo.ID,
		HistoryID:    historyID,
		Level:        level,
		Phase:        phase,
		Operation:    operation,
		Message:      message,
		Details:      details,
		Timestamp:    time.Now(),
	}

	if err := e.storage.CreateMigrationLog(ctx, log); err != nil {
		e.logger.Error("Failed to create migration log", "error", err)
	}
}

// unlockSourceRepository unlocks the source repository if it was locked during migration
func (e *Executor) unlockSourceRepository(ctx context.Context, repo *models.Repository) {
	if repo.SourceMigrationID == nil {
		e.logger.Debug("No source migration ID, skipping unlock", "repo", repo.FullName)
		return
	}

	e.logger.Info("Unlocking source repository",
		"repo", repo.FullName,
		"migration_id", *repo.SourceMigrationID)

	err := e.sourceClient.UnlockRepository(ctx, repo.Organization(), repo.Name(), *repo.SourceMigrationID)
	if err != nil {
		// Log error but don't fail the migration - the unlock can be done manually if needed
		e.logger.Error("Failed to unlock source repository (can be unlocked manually)",
			"error", err,
			"repo", repo.FullName,
			"migration_id", *repo.SourceMigrationID)
	} else {
		e.logger.Info("Successfully unlocked source repository", "repo", repo.FullName)
	}
}

// runPreMigrationDiscovery refreshes repository characteristics before migration
// This uses API-only calls to update basic repository information
func (e *Executor) runPreMigrationDiscovery(ctx context.Context, repo *models.Repository) error {
	e.logger.Info("Refreshing repository characteristics before migration", "repo", repo.FullName)

	// Get repository from source API
	var sourceRepo *ghapi.Repository
	var err error

	_, err = e.sourceClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		sourceRepo, resp, err = e.sourceClient.REST().Repositories.Get(ctx, repo.Organization(), repo.Name())
		return resp, err
	})

	if err != nil {
		return fmt.Errorf("failed to get repository from source: %w", err)
	}

	// Update basic repository information from API
	totalSize := int64(sourceRepo.GetSize()) * 1024 // Convert KB to bytes
	repo.TotalSize = &totalSize

	defaultBranch := sourceRepo.GetDefaultBranch()
	repo.DefaultBranch = &defaultBranch

	repo.HasWiki = sourceRepo.GetHasWiki()
	repo.HasPages = sourceRepo.GetHasPages()
	repo.IsArchived = sourceRepo.GetArchived()

	// Update last push date
	if sourceRepo.PushedAt != nil {
		pushTime := sourceRepo.PushedAt.Time
		repo.LastCommitDate = &pushTime
	}

	// Get branch count
	branches, _, err := e.sourceClient.REST().Repositories.ListBranches(ctx, repo.Organization(), repo.Name(), nil)
	if err == nil {
		repo.BranchCount = len(branches)
	}

	// Get last commit SHA from default branch
	if defaultBranch != "" {
		branch, _, err := e.sourceClient.REST().Repositories.GetBranch(ctx, repo.Organization(), repo.Name(), defaultBranch, 0)
		if err == nil && branch != nil && branch.Commit != nil {
			sha := branch.Commit.GetSHA()
			repo.LastCommitSHA = &sha
		}
	}

	// Get tag count
	tags, _, err := e.sourceClient.REST().Repositories.ListTags(ctx, repo.Organization(), repo.Name(), nil)
	if err == nil {
		repo.TagCount = len(tags)
	}

	// Update repository in database
	repo.UpdatedAt = time.Now()
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Warn("Failed to update repository after discovery", "error", err)
		// Don't fail - just log the warning
	}

	e.logger.Info("Pre-migration discovery complete",
		"repo", repo.FullName,
		"total_size", repo.TotalSize,
		"branches", repo.BranchCount,
		"tags", repo.TagCount)

	return nil
}

// calculateAdaptivePollInterval returns the appropriate polling interval based on elapsed time.
// During the fast phase, it returns the initial interval to catch quick completions.
// After the fast phase, it applies exponential backoff up to the maximum interval.
func calculateAdaptivePollInterval(elapsed, initial, max, fastPhaseDuration time.Duration) time.Duration {
	// During fast phase, use initial interval for quick polling
	if elapsed < fastPhaseDuration {
		return initial
	}

	// Calculate how many intervals have passed since fast phase ended
	timeSinceFastPhase := elapsed - fastPhaseDuration
	iterationsSinceFastPhase := float64(timeSinceFastPhase) / float64(initial)

	// Apply exponential backoff: initial * (multiplier ^ iterations)
	interval := float64(initial) * math.Pow(pollingBackoffMultiplier, iterationsSinceFastPhase)

	// Cap at maximum interval
	if time.Duration(interval) > max {
		return max
	}
	return time.Duration(interval)
}
