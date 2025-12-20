package migration

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
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

// Status constants
const (
	statusFailed   = "failed"
	statusExported = "exported"
)

// Adaptive polling configuration - preserves rate limits for long-running migrations
const (
	// Archive polling intervals
	archiveInitialInterval   = 30 * time.Second // Initial polling interval
	archiveMaxInterval       = 5 * time.Minute  // Maximum polling interval (don't poll slower than this)
	archiveFastPhaseDuration = 10 * time.Minute // Duration of fast polling phase
	archiveTimeout           = 24 * time.Hour   // Maximum time to wait for archive generation

	// Migration status polling intervals
	migrationInitialInterval   = 30 * time.Second // Initial polling interval
	migrationMaxInterval       = 10 * time.Minute // Maximum polling interval
	migrationFastPhaseDuration = 15 * time.Minute // Duration of fast polling phase
	migrationTimeout           = 48 * time.Hour   // Maximum time to wait for migration

	// Backoff multiplier (interval grows by this factor each iteration after fast phase)
	pollingBackoffMultiplier = 1.5
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
	migSourceCache       map[string]string // Cache of owner ID -> migration source ID for GitHub (supports multiple dest orgs)
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

// ArchiveIDs holds the migration IDs for both git and metadata archives
type ArchiveIDs struct {
	GitArchiveID      int64
	MetadataArchiveID int64
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
		migSourceCache:       make(map[string]string), // Initialize cache for multiple dest orgs
		adoMigSourceCache:    make(map[string]string), // Initialize cache for multiple ADO orgs
		logger:               cfg.Logger,
		postMigrationMode:    postMigMode,
		destRepoExistsAction: destRepoAction,
		visibilityHandling:   visibilityHandling,
	}, nil
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

	// Determine effective migration flags (repo settings take precedence over batch settings)
	excludeReleases := e.shouldExcludeReleases(repo, batch)
	excludeAttachments := e.shouldExcludeAttachments(repo, batch)
	lockRepositories := !dryRun

	e.logger.Info("Starting migration",
		"repo", repo.FullName,
		"dry_run", dryRun,
		"has_batch", batch != nil)

	// Log all migration flags for observability and audit
	e.logger.Info("Migration flags",
		"repo", repo.FullName,
		"dry_run", dryRun,
		"lock_repositories", lockRepositories,
		"exclude_releases", excludeReleases,
		"exclude_attachments", excludeAttachments,
		"archive_mode", "separate",
		"repo_exclude_releases", repo.ExcludeReleases,
		"repo_exclude_attachments", repo.ExcludeAttachments,
		"batch_exclude_releases", batch != nil && batch.ExcludeReleases,
		"batch_exclude_attachments", batch != nil && batch.ExcludeAttachments)

	// Create migration history record
	historyID, err := e.createMigrationHistory(ctx, repo, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
	}

	// Log operation
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "start",
		fmt.Sprintf("Starting %s for repository", map[bool]string{true: "dry run", false: "migration"}[dryRun]), nil)

	// Log migration flags to history for audit trail
	flagsDetails := fmt.Sprintf("lock_repositories=%v, exclude_releases=%v, exclude_attachments=%v, archive_mode=separate",
		lockRepositories, excludeReleases, excludeAttachments)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "flags", "Migration flags configured", &flagsDetails)

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
		e.updateHistoryStatus(ctx, historyID, statusFailed, &errMsg)

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
		fmt.Sprintf("Initiating archive generation on %s with options: exclude_releases=%v, exclude_attachments=%v (%s)", e.sourceClient.BaseURL(), excludeReleases, excludeAttachments, migrationMode), nil)

	archiveIDs, err := e.generateArchivesOnGHES(ctx, repo, batch, lockRepos)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "archive_generation", "initiate", "Failed to generate archives", &errMsg)
		e.updateHistoryStatus(ctx, historyID, statusFailed, &errMsg)

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

	details := fmt.Sprintf("Git Archive ID: %d, Metadata Archive ID: %d", archiveIDs.GitArchiveID, archiveIDs.MetadataArchiveID)
	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "initiate", "Archive generation initiated successfully", &details)

	repo.Status = string(models.StatusArchiveGenerating)
	// Track migration ID and lock status for production migrations (use git archive ID as primary)
	migID := archiveIDs.GitArchiveID
	repo.SourceMigrationID = &migID
	repo.IsSourceLocked = lockRepos
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	// Phase 3: Poll for archive generation completion (both git and metadata archives)
	e.logger.Info("Polling archive generation status", "repo", repo.FullName,
		"git_archive_id", archiveIDs.GitArchiveID,
		"metadata_archive_id", archiveIDs.MetadataArchiveID)
	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "poll", "Polling for archive generation completion", nil)

	archiveURLs, err := e.pollArchiveGeneration(ctx, repo, historyID, archiveIDs)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "archive_generation", "poll", "Archive generation failed", &errMsg)
		e.updateHistoryStatus(ctx, historyID, statusFailed, &errMsg)

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
		e.updateHistoryStatus(ctx, historyID, statusFailed, &errMsg)

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
		e.updateHistoryStatus(ctx, historyID, statusFailed, &errMsg)

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
