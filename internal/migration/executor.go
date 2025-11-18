package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/shurcooL/githubv4"
)

// PostMigrationMode defines when to run post-migration validation/tasks
type PostMigrationMode string

const (
	// PostMigrationNever - Never run post-migration validation/tasks
	PostMigrationNever PostMigrationMode = "never"

	// PostMigrationProductionOnly - Only run on production migrations (default)
	PostMigrationProductionOnly PostMigrationMode = "production_only"

	// PostMigrationDryRunOnly - Only run on dry runs (for testing validation)
	PostMigrationDryRunOnly PostMigrationMode = "dry_run_only"

	// PostMigrationAlways - Always run (both dry run and production)
	PostMigrationAlways PostMigrationMode = "always"
)

// Visibility constants
const (
	visibilityPrivate  = "private"
	visibilityPublic   = "public"
	visibilityInternal = "internal"
)

// DestinationRepoExistsAction defines what to do if destination repo already exists
type DestinationRepoExistsAction string

const (
	// DestinationRepoExistsFail - Fail migration if destination repo exists (default/safest)
	DestinationRepoExistsFail DestinationRepoExistsAction = "fail"

	// DestinationRepoExistsSkip - Skip migration if destination repo exists
	DestinationRepoExistsSkip DestinationRepoExistsAction = "skip"

	// DestinationRepoExistsDelete - Delete existing destination repo before migration
	DestinationRepoExistsDelete DestinationRepoExistsAction = "delete"
)

// VisibilityHandling defines how to map source visibility to destination
type VisibilityHandling struct {
	PublicRepos   string // public, internal, or private (default: private)
	InternalRepos string // internal or private (default: private)
}

// Executor handles repository migrations from GHES to GHEC
type Executor struct {
	sourceClient         *github.Client // GHES client (nil for ADO sources)
	sourceToken          string         // Source PAT (for ADO sources where sourceClient is nil)
	sourceURL            string         // Source system URL (GitHub base URL or primary ADO org URL for config validation)
	destClient           *github.Client // GHEC client
	storage              *storage.Database
	orgIDCache           map[string]string // Cache of org name -> org ID
	migSourceID          string            // Cached migration source ID for GitHub (created on first use)
	adoMigSourceCache    map[string]string // Cache of ADO org URL -> migration source ID (supports multiple ADO orgs)
	logger               *slog.Logger
	postMigrationMode    PostMigrationMode           // When to run post-migration tasks
	destRepoExistsAction DestinationRepoExistsAction // What to do if destination repo exists
	visibilityHandling   VisibilityHandling          // How to handle visibility transformations
}

// ExecutorConfig configures the migration executor
type ExecutorConfig struct {
	SourceClient         *github.Client
	SourceToken          string // Source PAT (required for ADO sources, optional for GitHub if SourceClient provided)
	SourceURL            string // Source system URL (GitHub base URL or ADO org URL, e.g., https://dev.azure.com/org)
	DestClient           *github.Client
	Storage              *storage.Database
	Logger               *slog.Logger
	PostMigrationMode    PostMigrationMode           // When to run post-migration tasks (default: production_only)
	DestRepoExistsAction DestinationRepoExistsAction // What to do if destination repo exists (default: fail)
	VisibilityHandling   VisibilityHandling          // How to handle visibility transformations (default: all private)
}

// ArchiveURLs contains the URLs for migration archives
type ArchiveURLs struct {
	GitSource string
	Metadata  string
}

// NewExecutor creates a new migration executor
func NewExecutor(cfg ExecutorConfig) (*Executor, error) {
	// Note: SourceClient can be nil for Azure DevOps sources (ADO migrations don't need it)
	// GitHub Enterprise Importer pulls directly from ADO using ADO PAT
	if cfg.DestClient == nil {
		return nil, fmt.Errorf("destination client is required")
	}
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Default to production_only if not specified
	postMigMode := cfg.PostMigrationMode
	if postMigMode == "" {
		postMigMode = PostMigrationProductionOnly
	}

	// Default to fail if not specified (safest option)
	destRepoAction := cfg.DestRepoExistsAction
	if destRepoAction == "" {
		destRepoAction = DestinationRepoExistsFail
	}

	// Default visibility handling to private if not specified (safest option)
	visibilityHandling := cfg.VisibilityHandling
	if visibilityHandling.PublicRepos == "" {
		visibilityHandling.PublicRepos = visibilityPrivate
	}
	if visibilityHandling.InternalRepos == "" {
		visibilityHandling.InternalRepos = visibilityPrivate
	}

	return &Executor{
		sourceClient:         cfg.SourceClient,
		sourceToken:          cfg.SourceToken,
		sourceURL:            cfg.SourceURL,
		destClient:           cfg.DestClient,
		storage:              cfg.Storage,
		orgIDCache:           make(map[string]string),
		adoMigSourceCache:    make(map[string]string), // Initialize cache for multiple ADO orgs
		logger:               cfg.Logger,
		postMigrationMode:    postMigMode,
		destRepoExistsAction: destRepoAction,
		visibilityHandling:   visibilityHandling,
	}, nil
}

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
	if e.migSourceID != "" {
		return e.migSourceID, nil
	}

	e.logger.Info("Creating migration source")

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

	e.migSourceID = string(mutation.CreateMigrationSource.MigrationSource.ID)
	e.logger.Info("Created migration source",
		"source_id", e.migSourceID,
		"source_url", sourceURL,
		"name", string(mutation.CreateMigrationSource.MigrationSource.Name),
		"type", string(mutation.CreateMigrationSource.MigrationSource.Type))

	return e.migSourceID, nil
}

// ExecuteMigration performs a full repository migration
// batch parameter is optional - if provided, batch-level settings will be applied when repo settings are not specified
//
//nolint:gocyclo // Sequential state machine with multiple phases requires complexity
func (e *Executor) ExecuteMigration(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	// GitHub-to-GitHub migrations require source client
	if e.sourceClient == nil {
		return fmt.Errorf("source client is required for GitHub-to-GitHub migrations")
	}

	e.logger.Info("Starting migration",
		"repo", repo.FullName,
		"dry_run", dryRun,
		"has_batch", batch != nil)

	// Create migration history record
	historyID, err := e.createMigrationHistory(ctx, repo, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
	}

	// Log operation
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "start",
		fmt.Sprintf("Starting %s for repository", map[bool]string{true: "dry run", false: "migration"}[dryRun]), nil)

	// Phase 1: Pre-migration validation
	e.logger.Info("Running pre-migration validation", "repo", repo.FullName)
	e.logOperation(ctx, repo, historyID, "INFO", "pre_migration", "validate", "Running pre-migration validation", nil)

	// Run discovery on source repository for production migrations to get latest stats
	if !dryRun {
		e.logger.Info("Running pre-migration discovery to refresh repository data", "repo", repo.FullName)
		e.logOperation(ctx, repo, historyID, "INFO", "pre_migration", "discovery", "Refreshing repository characteristics", nil)

		if err := e.runPreMigrationDiscovery(ctx, repo); err != nil {
			// Log warning but don't fail migration
			errMsg := err.Error()
			e.logger.Warn("Pre-migration discovery failed, continuing with existing data",
				"repo", repo.FullName,
				"error", err)
			e.logOperation(ctx, repo, historyID, "WARN", "pre_migration", "discovery", "Pre-migration discovery failed", &errMsg)
		} else {
			e.logOperation(ctx, repo, historyID, "INFO", "pre_migration", "discovery", "Repository data refreshed successfully", nil)
		}
	}

	if err := e.validatePreMigration(ctx, repo, batch); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "pre_migration", "validate", "Pre-migration validation failed", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		status := models.StatusMigrationFailed
		if dryRun {
			status = models.StatusDryRunFailed
		}
		repo.Status = string(status)
		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("pre-migration validation failed: %w", err)
	}
	e.logOperation(ctx, repo, historyID, "INFO", "pre_migration", "validate", "Pre-migration validation passed", nil)

	// Update status
	status := models.StatusPreMigration
	if dryRun {
		status = models.StatusDryRunInProgress
	}
	repo.Status = string(status)
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	// Phase 2: Generate migration archives on GHES
	// For dry runs: lock_repositories = false (test migration without locking)
	// For production: lock_repositories = true (lock during migration)
	lockRepos := !dryRun
	migrationMode := "production migration"
	if dryRun {
		migrationMode = "dry run migration (lock_repositories: false)"
	}

	e.logger.Info("Generating archives on source repository", "repo", repo.FullName, "mode", migrationMode)
	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "initiate",
		fmt.Sprintf("Initiating archive generation on %s (%s)", e.sourceClient.BaseURL(), migrationMode), nil)

	archiveID, err := e.generateArchivesOnGHES(ctx, repo, lockRepos)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "archive_generation", "initiate", "Failed to generate archives", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		status := models.StatusMigrationFailed
		if dryRun {
			status = models.StatusDryRunFailed
		}
		repo.Status = string(status)

		// Unlock repository if it was locked
		if lockRepos && repo.SourceMigrationID != nil {
			repo.IsSourceLocked = false
			e.unlockSourceRepository(ctx, repo)
		}

		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("failed to generate archives: %w", err)
	}

	details := fmt.Sprintf("Archive ID: %d", archiveID)
	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "initiate", "Archive generation initiated successfully", &details)

	repo.Status = string(models.StatusArchiveGenerating)
	// Track migration ID and lock status for production migrations
	migID := archiveID
	repo.SourceMigrationID = &migID
	repo.IsSourceLocked = lockRepos
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	// Phase 3: Poll for archive generation completion
	e.logger.Info("Polling archive generation status", "repo", repo.FullName, "archive_id", archiveID)
	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "poll", "Polling for archive generation completion", nil)

	archiveURLs, err := e.pollArchiveGeneration(ctx, repo, historyID, archiveID)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "archive_generation", "poll", "Archive generation failed", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		repo.Status = string(models.StatusMigrationFailed)

		// Unlock repository if it was locked
		if lockRepos && repo.SourceMigrationID != nil {
			repo.IsSourceLocked = false
			e.unlockSourceRepository(ctx, repo)
		}

		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("archive generation failed: %w", err)
	}

	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "complete", "Archives generated successfully", nil)

	// Phase 4: Start migration on GHEC using GraphQL
	e.logger.Info("Starting migration on GHEC", "repo", repo.FullName)
	e.logOperation(ctx, repo, historyID, "INFO", "migration_start", "initiate", "Starting migration on destination", nil)

	migrationID, err := e.startRepositoryMigration(ctx, repo, batch, archiveURLs)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "migration_start", "initiate", "Failed to start migration", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		repo.Status = string(models.StatusMigrationFailed)

		// Unlock repository if it was locked
		if lockRepos && repo.SourceMigrationID != nil {
			repo.IsSourceLocked = false
			e.unlockSourceRepository(ctx, repo)
		}

		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("failed to start migration: %w", err)
	}

	details = fmt.Sprintf("Migration ID: %s", migrationID)
	e.logOperation(ctx, repo, historyID, "INFO", "migration_start", "initiate", "Migration started successfully", &details)

	repo.Status = string(models.StatusMigratingContent)
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	// Phase 5: Poll for migration completion
	e.logger.Info("Polling migration status", "repo", repo.FullName, "migration_id", migrationID)
	e.logOperation(ctx, repo, historyID, "INFO", "migration_progress", "poll", "Polling for migration completion", nil)

	if err := e.pollMigrationStatus(ctx, repo, batch, historyID, migrationID); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "migration_progress", "poll", "Migration failed", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		repo.Status = string(models.StatusMigrationFailed)

		// Unlock repository if it was locked
		if lockRepos && repo.SourceMigrationID != nil {
			repo.IsSourceLocked = false
			e.unlockSourceRepository(ctx, repo)
		}

		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	e.logOperation(ctx, repo, historyID, "INFO", "migration_progress", "complete", "Migration completed successfully", nil)

	// Phase 6: Post-migration validation (configurable)
	if e.shouldRunPostMigration(dryRun) {
		e.logger.Info("Running post-migration validation", "repo", repo.FullName, "mode", e.postMigrationMode)
		e.logOperation(ctx, repo, historyID, "INFO", "post_migration", "validate", "Running post-migration validation", nil)

		if err := e.validatePostMigration(ctx, repo); err != nil {
			errMsg := err.Error()
			e.logOperation(ctx, repo, historyID, "WARN", "post_migration", "validate", "Post-migration validation failed", &errMsg)
			// Don't fail the migration on validation warnings
		} else {
			e.logOperation(ctx, repo, historyID, "INFO", "post_migration", "validate", "Post-migration validation passed", nil)
		}
	} else {
		reason := fmt.Sprintf("Skipping post-migration validation (mode: %s, dry_run: %v)", e.postMigrationMode, dryRun)
		e.logger.Info(reason, "repo", repo.FullName)
		e.logOperation(ctx, repo, historyID, "INFO", "post_migration", "skip", reason, nil)
	}

	// Phase 7: Mark complete
	completionStatus := models.StatusComplete
	completionMsg := "Migration completed successfully"
	// Clear lock status on successful completion
	repo.IsSourceLocked = false

	// Unlock source repository for production migrations
	if !dryRun && repo.SourceMigrationID != nil {
		e.unlockSourceRepository(ctx, repo)
	}

	if dryRun {
		completionStatus = models.StatusDryRunComplete
		completionMsg = "Dry run completed successfully - repository can be migrated safely"
	}

	e.logger.Info("Migration complete", "repo", repo.FullName, "dry_run", dryRun)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "complete", completionMsg, nil)
	e.updateHistoryStatus(ctx, historyID, "completed", nil)

	repo.Status = string(completionStatus)
	now := time.Now()

	// Set appropriate timestamps based on migration type
	if dryRun {
		// Set last dry run timestamp
		repo.LastDryRunAt = &now
	} else {
		// Set migrated timestamp for production migrations
		repo.MigratedAt = &now
	}

	return e.storage.UpdateRepository(ctx, repo)
}

// generateArchivesOnGHES creates migration archives on GHES using REST API
func (e *Executor) generateArchivesOnGHES(ctx context.Context, repo *models.Repository, lockRepositories bool) (int64, error) {
	opt := &ghapi.MigrationOptions{
		LockRepositories:   lockRepositories,
		ExcludeAttachments: false,
		ExcludeReleases:    false,
	}

	// Check if we need to exclude releases due to size
	if repo.TotalSize != nil && *repo.TotalSize > 10*1024*1024*1024 { // >10GB
		opt.ExcludeReleases = true
		e.logger.Info("Excluding releases due to repository size", "repo", repo.FullName, "size", *repo.TotalSize)
	}

	var migration *ghapi.Migration
	var err error

	_, err = e.sourceClient.DoWithRetry(ctx, "StartMigration", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		migration, resp, err = e.sourceClient.REST().Migrations.StartMigration(
			ctx,
			repo.Organization(),
			[]string{repo.Name()},
			opt,
		)
		return resp, err
	})

	if err != nil {
		return 0, fmt.Errorf("failed to start migration for repository %s: %w", repo.FullName, err)
	}

	if migration == nil || migration.ID == nil {
		return 0, fmt.Errorf("invalid migration response from source: %w", err)
	}

	return *migration.ID, nil
}

// pollArchiveGeneration polls for archive generation completion
func (e *Executor) pollArchiveGeneration(ctx context.Context, repo *models.Repository, historyID *int64, archiveID int64) (*ArchiveURLs, error) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.After(24 * time.Hour)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("archive generation timeout exceeded (24 hours)")
		case <-ticker.C:
			var migration *ghapi.Migration
			var err error

			_, err = e.sourceClient.DoWithRetry(ctx, "MigrationStatus", func(ctx context.Context) (*ghapi.Response, error) {
				var resp *ghapi.Response
				migration, resp, err = e.sourceClient.REST().Migrations.MigrationStatus(
					ctx,
					repo.Organization(),
					archiveID,
				)
				return resp, err
			})

			if err != nil {
				return nil, fmt.Errorf("failed to check migration status: %w", err)
			}

			state := migration.GetState()
			e.logger.Debug("Archive generation status", "repo", repo.FullName, "state", state)

			switch state {
			case "exported":
				// Archives are ready, get download URL
				var archiveURL string
				var urlErr error
				_, err = e.sourceClient.DoWithRetry(ctx, "MigrationArchiveURL", func(ctx context.Context) (*ghapi.Response, error) {
					archiveURL, urlErr = e.sourceClient.REST().Migrations.MigrationArchiveURL(
						ctx,
						repo.Organization(),
						archiveID,
					)
					return nil, urlErr
				})

				if err != nil {
					return nil, fmt.Errorf("failed to get archive URL: %w", err)
				}

				return &ArchiveURLs{
					GitSource: archiveURL,
					Metadata:  archiveURL, // In practice, these may be separate
				}, nil

			case "failed":
				return nil, fmt.Errorf("archive generation failed for repository %s: %w", repo.FullName, err)

			case "pending", "exporting":
				// Continue polling
				if historyID != nil {
					msg := fmt.Sprintf("Archive generation in progress (state: %s)", state)
					e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "poll", msg, nil)
				}
				continue

			default:
				e.logger.Warn("Unknown archive state", "state", state)
				continue
			}
		}
	}
}

// startRepositoryMigration starts migration on GHEC using GraphQL
// nolint:gocyclo // Migration startup involves multiple steps and validations
func (e *Executor) startRepositoryMigration(ctx context.Context, repo *models.Repository, batch *models.Batch, urls *ArchiveURLs) (string, error) {
	// Get destination org name for this repository
	destOrgName := e.getDestinationOrg(repo, batch)
	if destOrgName == "" {
		return "", fmt.Errorf("unable to determine destination organization for repository %s", repo.FullName)
	}

	// Fetch destination org ID
	destOrgID, err := e.getOrFetchDestOrgID(ctx, destOrgName)
	if err != nil {
		return "", fmt.Errorf("failed to get destination org ID: %w", err)
	}

	// Create migration source if not already cached (pass ownerID)
	migSourceID, err := e.getOrCreateMigrationSource(ctx, destOrgID)
	if err != nil {
		return "", fmt.Errorf("failed to get migration source ID: %w", err)
	}

	var mutation struct {
		StartRepositoryMigration struct {
			RepositoryMigration struct {
				ID              githubv4.String
				State           githubv4.String
				SourceURL       githubv4.String
				MigrationSource struct {
					ID   githubv4.String
					Name githubv4.String
					Type githubv4.String
				}
			}
		} `graphql:"startRepositoryMigration(input: $input)"`
	}

	// Parse the source repository URL for URI type
	parsedURL, err := url.Parse(repo.SourceURL)
	if err != nil {
		e.logger.Error("Failed to parse source repository URL",
			"error", err,
			"url", repo.SourceURL)
		return "", fmt.Errorf("invalid source repository URL: %w", err)
	}

	// Create URI from parsed URL
	sourceRepoURI := githubv4.URI{URL: parsedURL}

	// Create pointers for optional fields
	continueOnError := githubv4.Boolean(true)

	// Apply visibility transformation based on source visibility and config
	targetVisibility := e.determineTargetVisibility(repo.Visibility)
	targetRepoVisibility := githubv4.String(targetVisibility)

	e.logger.Info("Applying visibility transformation",
		"repo", repo.FullName,
		"source_visibility", repo.Visibility,
		"target_visibility", targetVisibility)

	gitArchiveURL := githubv4.String(urls.GitSource)
	metadataArchiveURL := githubv4.String(urls.Metadata)

	// Get tokens - IMPORTANT: Per GitHub documentation:
	// - AccessToken: Personal access token for the SOURCE (GHES)
	// - GitHubPat: Personal access token for the DESTINATION (GHEC)
	sourceToken := githubv4.String(e.sourceClient.Token()) // GHES token
	destToken := githubv4.String(e.destClient.Token())     // GHEC token

	// Get the destination repository name (respects DestinationFullName if set)
	destRepoName := e.getDestinationRepoName(repo)

	// Use typed input struct
	input := githubv4.StartRepositoryMigrationInput{
		SourceID:             githubv4.ID(migSourceID),
		OwnerID:              githubv4.ID(destOrgID),
		RepositoryName:       githubv4.String(destRepoName),
		ContinueOnError:      &continueOnError,
		TargetRepoVisibility: &targetRepoVisibility,
		SourceRepositoryURL:  sourceRepoURI,
		GitArchiveURL:        &gitArchiveURL,
		MetadataArchiveURL:   &metadataArchiveURL,
		AccessToken:          &sourceToken, // Source GHES token
		GitHubPat:            &destToken,   // Destination GHEC token
	}

	// Add skipReleases flag if enabled (check both repo and batch settings)
	if e.shouldExcludeReleases(repo, batch) {
		skipReleases := githubv4.Boolean(true)
		input.SkipReleases = &skipReleases
		e.logger.Info("Excluding releases from migration (skipReleases=true)", "repo", repo.FullName)
	}

	err = e.destClient.MutateWithRetry(ctx, "StartRepositoryMigration", &mutation, input, nil)
	if err != nil {
		return "", fmt.Errorf("failed to start migration via GraphQL: %w", err)
	}

	migrationID := string(mutation.StartRepositoryMigration.RepositoryMigration.ID)
	e.logger.Info("Repository migration started via GraphQL",
		"migration_id", migrationID,
		"repository", repo.Name(),
		"source_id", string(mutation.StartRepositoryMigration.RepositoryMigration.MigrationSource.ID),
		"source_url", string(mutation.StartRepositoryMigration.RepositoryMigration.SourceURL))

	return migrationID, nil
}

// pollMigrationStatus polls for migration completion on GHEC
func (e *Executor) pollMigrationStatus(ctx context.Context, repo *models.Repository, batch *models.Batch, historyID *int64, migrationID string) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.After(48 * time.Hour) // Migrations can take longer

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("migration timeout exceeded (48 hours)")
		case <-ticker.C:
			var query struct {
				Node struct {
					Migration struct {
						ID              githubv4.String
						State           githubv4.String
						FailureReason   githubv4.String
						RepositoryName  githubv4.String
						MigrationSource struct {
							Name githubv4.String
						}
					} `graphql:"... on Migration"`
				} `graphql:"node(id: $id)"`
			}

			variables := map[string]interface{}{
				"id": githubv4.ID(migrationID),
			}

			err := e.destClient.QueryWithRetry(ctx, "GetMigrationStatus", &query, variables)
			if err != nil {
				return fmt.Errorf("failed to query migration status: %w", err)
			}

			state := string(query.Node.Migration.State)
			e.logger.Debug("Migration status", "repo", repo.FullName, "state", state)

			switch state {
			case "SUCCEEDED":
				repo.Status = string(models.StatusMigrationComplete)
				// Set destination details using the correct destination org and repo name
				destOrg := e.getDestinationOrg(repo, batch)
				destRepoName := e.getDestinationRepoName(repo)
				destFullName := fmt.Sprintf("%s/%s", destOrg, destRepoName)
				repo.DestinationFullName = &destFullName
				// Convert full name to URL using the destination client
				destURL := e.destClient.RepositoryURL(destFullName)
				repo.DestinationURL = &destURL
				if err := e.storage.UpdateRepository(ctx, repo); err != nil {
					e.logger.Error("Failed to update repository status", "error", err)
				}
				return nil

			case "FAILED":
				failureReason := string(query.Node.Migration.FailureReason)
				repo.Status = string(models.StatusMigrationFailed)
				if err := e.storage.UpdateRepository(ctx, repo); err != nil {
					e.logger.Error("Failed to update repository status", "error", err)
				}
				return fmt.Errorf("migration failed: %s", failureReason)

			case "IN_PROGRESS", "QUEUED", "PENDING_VALIDATION":
				repo.Status = string(models.StatusMigratingContent)
				if err := e.storage.UpdateRepository(ctx, repo); err != nil {
					e.logger.Error("Failed to update repository status", "error", err)
				}

				if historyID != nil {
					msg := fmt.Sprintf("Migration in progress (state: %s)", state)
					e.logOperation(ctx, repo, historyID, "INFO", "migration_progress", "poll", msg, nil)
				}
				continue

			default:
				e.logger.Warn("Unknown migration state", "state", state)
				continue
			}
		}
	}
}

// validatePreMigration performs pre-migration validation
// nolint:gocyclo // Complex validation logic - refactoring would reduce readability
func (e *Executor) validatePreMigration(ctx context.Context, repo *models.Repository, batch *models.Batch) error {
	// Check for GitHub Enterprise Importer blocking issues
	if repo.HasOversizedRepository {
		return fmt.Errorf("repository exceeds GitHub's 40 GiB size limit and requires remediation before migration (reduce repository size using Git LFS or history rewriting)")
	}

	// Check for blockers
	var issues []string

	// Check for very large files
	if repo.LargestFileSize != nil && *repo.LargestFileSize > 100*1024*1024 { // >100MB
		issues = append(issues, fmt.Sprintf("Very large file detected: %s (%d MB)",
			*repo.LargestFile, *repo.LargestFileSize/(1024*1024)))
	}

	// Check for very large repository
	if repo.TotalSize != nil && *repo.TotalSize > 50*1024*1024*1024 { // >50GB
		issues = append(issues, fmt.Sprintf("Very large repository: %d GB",
			*repo.TotalSize/(1024*1024*1024)))
	}

	// 1. Verify source repository exists and is accessible (GitHub sources only)
	// For ADO sources, sourceClient is nil and we skip this check
	// GEI will validate ADO source accessibility during migration
	var err error
	if e.sourceClient != nil {
		e.logger.Info("Checking source repository", "repo", repo.FullName)
		var sourceRepoData *ghapi.Repository

		_, err = e.sourceClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
			var resp *ghapi.Response
			sourceRepoData, resp, err = e.sourceClient.REST().Repositories.Get(ctx, repo.Organization(), repo.Name())
			return resp, err
		})

		if err != nil {
			return fmt.Errorf("source repository not found or inaccessible: %w", err)
		}

		e.logger.Info("Source repository verified", "repo", repo.FullName)

		// Verify repository is not archived
		if sourceRepoData.GetArchived() {
			issues = append(issues, "Repository is archived")
		}
	} else {
		e.logger.Info("Skipping source repository check (non-GitHub source)", "repo", repo.FullName)
	}

	// 2. Check if destination repository already exists
	destOrg := e.getDestinationOrg(repo, batch)
	destRepoName := e.getDestinationRepoName(repo)
	e.logger.Info("Checking destination repository",
		"source_repo", repo.FullName,
		"dest_org", destOrg,
		"dest_repo_name", destRepoName,
		"action", e.destRepoExistsAction)
	var destRepoData *ghapi.Repository
	destExists := false

	_, err = e.destClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		destRepoData, resp, err = e.destClient.REST().Repositories.Get(ctx, destOrg, destRepoName)
		return resp, err
	})

	if err == nil {
		// Destination repository exists
		destExists = true
		e.logger.Warn("Destination repository already exists",
			"repo", repo.FullName,
			"dest_repo", destRepoData.GetFullName(),
			"action", e.destRepoExistsAction)
	} else if github.IsNotFoundError(err) {
		// Destination repository does not exist - this is expected
		e.logger.Info("Destination repository does not exist - ready for migration", "repo", repo.FullName)
		destExists = false
	} else {
		// Some other error occurred
		e.logger.Warn("Unable to check destination repository", "repo", repo.FullName, "error", err)
		// Continue - we'll find out during migration if there's an issue
	}

	// Handle destination repository exists scenarios
	if destExists {
		switch e.destRepoExistsAction {
		case DestinationRepoExistsFail:
			return fmt.Errorf("destination repository already exists: %s (action: fail)", destRepoData.GetFullName())

		case DestinationRepoExistsSkip:
			e.logger.Info("Skipping migration - destination repository exists",
				"repo", repo.FullName,
				"action", e.destRepoExistsAction)
			return fmt.Errorf("destination repository already exists: %s (action: skip)", destRepoData.GetFullName())

		case DestinationRepoExistsDelete:
			e.logger.Warn("Deleting existing destination repository",
				"source_repo", repo.FullName,
				"dest_repo", destRepoData.GetFullName())

			// Delete the existing repository
			_, err = e.destClient.DoWithRetry(ctx, "DeleteRepository", func(ctx context.Context) (*ghapi.Response, error) {
				resp, err := e.destClient.REST().Repositories.Delete(ctx, destOrg, destRepoName)
				return resp, err
			})

			if err != nil {
				return fmt.Errorf("failed to delete existing destination repository: %w", err)
			}

			e.logger.Info("Successfully deleted existing destination repository",
				"repo", repo.FullName)
		}
	}

	// Log warnings but don't fail
	if len(issues) > 0 {
		e.logger.Warn("Pre-migration validation warnings",
			"repo", repo.FullName,
			"issues", issues)
	}

	return nil
}

// validatePostMigration performs comprehensive post-migration validation
func (e *Executor) validatePostMigration(ctx context.Context, repo *models.Repository) error {
	if repo.DestinationFullName == nil {
		return fmt.Errorf("destination repository not set")
	}

	e.logger.Info("Running post-migration validation with characteristic comparison",
		"repo", repo.FullName,
		"destination", *repo.DestinationFullName)

	// Profile the destination repository (API-only, no cloning)
	destRepo, err := e.profileDestinationRepository(ctx, *repo.DestinationFullName)
	if err != nil {
		return fmt.Errorf("failed to profile destination repository: %w", err)
	}

	// Compare source and destination characteristics
	mismatches, hasCriticalMismatches := e.compareRepositoryCharacteristics(repo, destRepo)

	// Generate validation report
	validationStatus := "passed"
	var validationDetails *string
	var destinationData *string

	if len(mismatches) > 0 {
		validationStatus = "failed"

		// Log all mismatches
		e.logger.Warn("Post-migration validation found mismatches",
			"repo", repo.FullName,
			"mismatch_count", len(mismatches),
			"critical", hasCriticalMismatches)

		for _, mismatch := range mismatches {
			e.logger.Warn("Validation mismatch",
				"repo", repo.FullName,
				"field", mismatch.Field,
				"source", mismatch.SourceValue,
				"destination", mismatch.DestValue,
				"critical", mismatch.Critical)
		}

		// Generate JSON validation details
		validationReport := e.generateValidationReport(mismatches)
		validationDetails = &validationReport

		// Store destination data for further analysis
		destDataJSON := e.serializeDestinationData(destRepo)
		destinationData = &destDataJSON
	} else {
		e.logger.Info("Post-migration validation passed - all characteristics match",
			"repo", repo.FullName)
	}

	// Update validation fields in database
	if err := e.storage.UpdateRepositoryValidation(ctx, repo.FullName, validationStatus, validationDetails, destinationData); err != nil {
		e.logger.Error("Failed to update validation status", "error", err)
		// Don't fail the migration due to database update error
	}

	// Update repository validation fields
	repo.ValidationStatus = &validationStatus
	repo.ValidationDetails = validationDetails
	repo.DestinationData = destinationData

	// Don't fail migration on validation warnings - just log them
	return nil
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

// ValidationMismatch represents a mismatch between source and destination repository characteristics
type ValidationMismatch struct {
	Field       string
	SourceValue interface{}
	DestValue   interface{}
	Critical    bool // Whether this mismatch is critical (affects migration success)
}

// profileDestinationRepository profiles a destination repository using API-only metrics
func (e *Executor) profileDestinationRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	// Parse full name
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository full name: %s", fullName)
	}
	org := parts[0]
	name := parts[1]

	// Get repository details from destination
	var ghRepo *ghapi.Repository
	var err error

	_, err = e.destClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		ghRepo, resp, err = e.destClient.REST().Repositories.Get(ctx, org, name)
		return resp, err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get destination repository: %w", err)
	}

	// Create basic repository profile from GitHub API data
	totalSize := int64(ghRepo.GetSize()) * 1024 // Convert KB to bytes
	defaultBranch := ghRepo.GetDefaultBranch()
	repo := &models.Repository{
		FullName:      ghRepo.GetFullName(),
		DefaultBranch: &defaultBranch,
		TotalSize:     &totalSize,
		HasWiki:       ghRepo.GetHasWiki(),
		HasPages:      ghRepo.GetHasPages(),
		IsArchived:    ghRepo.GetArchived(),
	}

	// Get branch count
	branches, _, err := e.destClient.REST().Repositories.ListBranches(ctx, org, name, nil)
	if err == nil {
		repo.BranchCount = len(branches)
	}

	// Get last commit SHA from default branch
	if defaultBranch != "" {
		branch, _, err := e.destClient.REST().Repositories.GetBranch(ctx, org, name, defaultBranch, 0)
		if err == nil && branch != nil && branch.Commit != nil {
			sha := branch.Commit.GetSHA()
			repo.LastCommitSHA = &sha
		}
	}

	// Get commit count (approximation from contributors API)
	contributors, _, err := e.destClient.REST().Repositories.ListContributors(ctx, org, name, nil)
	if err == nil {
		totalCommits := 0
		for _, contributor := range contributors {
			totalCommits += contributor.GetContributions()
		}
		repo.CommitCount = totalCommits
	}

	// Get tag count
	tags, _, err := e.destClient.REST().Repositories.ListTags(ctx, org, name, nil)
	if err == nil {
		repo.TagCount = len(tags)
	}

	// Get issue and PR counts
	// Note: This is a simplified approach
	issues, _, err := e.destClient.REST().Issues.ListByRepo(ctx, org, name, &ghapi.IssueListByRepoOptions{
		State:       "all",
		ListOptions: ghapi.ListOptions{PerPage: 1},
	})
	if err == nil {
		// Count issues (excluding PRs)
		for _, issue := range issues {
			if issue.PullRequestLinks == nil {
				repo.IssueCount++
			} else {
				repo.PullRequestCount++
			}
		}
	}

	return repo, nil
}

// compareRepositoryCharacteristics compares source and destination repository characteristics
func (e *Executor) compareRepositoryCharacteristics(source, dest *models.Repository) ([]ValidationMismatch, bool) {
	var mismatches []ValidationMismatch
	hasCritical := false

	// Compare critical Git properties
	if source.DefaultBranch != nil && dest.DefaultBranch != nil && *source.DefaultBranch != *dest.DefaultBranch {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "default_branch",
			SourceValue: *source.DefaultBranch,
			DestValue:   *dest.DefaultBranch,
			Critical:    true,
		})
		hasCritical = true
	}

	if source.CommitCount != dest.CommitCount {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "commit_count",
			SourceValue: source.CommitCount,
			DestValue:   dest.CommitCount,
			Critical:    true,
		})
		hasCritical = true
	}

	if source.BranchCount != dest.BranchCount {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "branch_count",
			SourceValue: source.BranchCount,
			DestValue:   dest.BranchCount,
			Critical:    true,
		})
		hasCritical = true
	}

	if source.TagCount != dest.TagCount {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "tag_count",
			SourceValue: source.TagCount,
			DestValue:   dest.TagCount,
			Critical:    false,
		})
	}

	// Compare last commit SHA if available
	if source.LastCommitSHA != nil && dest.LastCommitSHA != nil && *source.LastCommitSHA != *dest.LastCommitSHA {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "last_commit_sha",
			SourceValue: *source.LastCommitSHA,
			DestValue:   *dest.LastCommitSHA,
			Critical:    true,
		})
		hasCritical = true
	}

	// Compare GitHub features (non-critical)
	if source.HasWiki != dest.HasWiki {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_wiki",
			SourceValue: source.HasWiki,
			DestValue:   dest.HasWiki,
			Critical:    false,
		})
	}

	if source.HasPages != dest.HasPages {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_pages",
			SourceValue: source.HasPages,
			DestValue:   dest.HasPages,
			Critical:    false,
		})
	}

	if source.HasDiscussions != dest.HasDiscussions {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_discussions",
			SourceValue: source.HasDiscussions,
			DestValue:   dest.HasDiscussions,
			Critical:    false,
		})
	}

	if source.HasActions != dest.HasActions {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_actions",
			SourceValue: source.HasActions,
			DestValue:   dest.HasActions,
			Critical:    false,
		})
	}

	if source.BranchProtections != dest.BranchProtections {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "branch_protections",
			SourceValue: source.BranchProtections,
			DestValue:   dest.BranchProtections,
			Critical:    false,
		})
	}

	return mismatches, hasCritical
}

// generateValidationReport generates a JSON validation report from mismatches
func (e *Executor) generateValidationReport(mismatches []ValidationMismatch) string {
	type Report struct {
		TotalMismatches    int                  `json:"total_mismatches"`
		CriticalMismatches int                  `json:"critical_mismatches"`
		Mismatches         []ValidationMismatch `json:"mismatches"`
	}

	criticalCount := 0
	for _, m := range mismatches {
		if m.Critical {
			criticalCount++
		}
	}

	report := Report{
		TotalMismatches:    len(mismatches),
		CriticalMismatches: criticalCount,
		Mismatches:         mismatches,
	}

	// Marshal to JSON
	data, err := json.Marshal(report)
	if err != nil {
		e.logger.Error("Failed to marshal validation report", "error", err)
		return fmt.Sprintf(`{"error": "failed to generate report: %s"}`, err.Error())
	}

	return string(data)
}

// serializeDestinationData serializes destination repository data to JSON
func (e *Executor) serializeDestinationData(dest *models.Repository) string {
	// Create a simplified struct with key fields
	type DestData struct {
		DefaultBranch     *string `json:"default_branch,omitempty"`
		BranchCount       int     `json:"branch_count"`
		CommitCount       int     `json:"commit_count"`
		TagCount          int     `json:"tag_count"`
		LastCommitSHA     *string `json:"last_commit_sha,omitempty"`
		TotalSize         *int64  `json:"total_size,omitempty"`
		HasWiki           bool    `json:"has_wiki"`
		HasPages          bool    `json:"has_pages"`
		HasDiscussions    bool    `json:"has_discussions"`
		HasActions        bool    `json:"has_actions"`
		BranchProtections int     `json:"branch_protections"`
		IssueCount        int     `json:"issue_count"`
		PullRequestCount  int     `json:"pull_request_count"`
	}

	data := DestData{
		DefaultBranch:     dest.DefaultBranch,
		BranchCount:       dest.BranchCount,
		CommitCount:       dest.CommitCount,
		TagCount:          dest.TagCount,
		LastCommitSHA:     dest.LastCommitSHA,
		TotalSize:         dest.TotalSize,
		HasWiki:           dest.HasWiki,
		HasPages:          dest.HasPages,
		HasDiscussions:    dest.HasDiscussions,
		HasActions:        dest.HasActions,
		BranchProtections: dest.BranchProtections,
		IssueCount:        dest.IssueCount,
		PullRequestCount:  dest.PullRequestCount,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		e.logger.Error("Failed to serialize destination data", "error", err)
		return fmt.Sprintf(`{"error": "failed to serialize: %s"}`, err.Error())
	}

	return string(jsonData)
}
