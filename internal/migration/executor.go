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

// Completion message constants
const (
	msgMigrationComplete = "Migration completed successfully"
	msgDryRunComplete    = "Dry run completed successfully - repository can be migrated safely"
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
		visibilityHandling.PublicRepos = models.VisibilityPrivate
	}
	if visibilityHandling.InternalRepos == "" {
		visibilityHandling.InternalRepos = models.VisibilityPrivate
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

// ExecuteMigration performs a full repository migration.
// batch parameter is optional - if provided, batch-level settings will be applied when repo settings are not specified.
//
// The migration proceeds through these phases:
//  1. Pre-migration validation and discovery
//  2. Archive generation on source (GHES)
//  3. Polling for archive completion
//  4. Migration start on destination (GHEC)
//  5. Polling for migration completion
//  6. Post-migration validation
//  7. Completion and cleanup
func (e *Executor) ExecuteMigration(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	// GitHub-to-GitHub migrations require source client
	if e.sourceClient == nil {
		return fmt.Errorf("source client is required for GitHub-to-GitHub migrations")
	}

	// Create migration context with computed values
	mc := e.NewMigrationContext(repo, batch, dryRun)

	e.logger.Info("Starting migration",
		"repo", repo.FullName,
		"dry_run", dryRun,
		"has_batch", batch != nil)

	// Log all migration flags for observability and audit
	e.logger.Info("Migration flags",
		"repo", repo.FullName,
		"dry_run", dryRun,
		"lock_repositories", mc.LockRepositories,
		"exclude_releases", mc.ExcludeReleases,
		"exclude_attachments", mc.ExcludeAttachments,
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
	mc.HistoryID = historyID

	// Log operation start
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "start",
		fmt.Sprintf("Starting %s for repository", map[bool]string{true: "dry run", false: "migration"}[dryRun]), nil)

	// Log migration flags to history for audit trail
	flagsDetails := fmt.Sprintf("lock_repositories=%v, exclude_releases=%v, exclude_attachments=%v, archive_mode=separate",
		mc.LockRepositories, mc.ExcludeReleases, mc.ExcludeAttachments)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "flags", "Migration flags configured", &flagsDetails)

	// Phase 1: Pre-migration validation
	if err := e.phasePreMigration(ctx, mc); err != nil {
		e.handlePhaseError(ctx, mc, err)
		return err
	}

	// Phase 2: Archive generation
	if err := e.phaseArchiveGeneration(ctx, mc); err != nil {
		e.handlePhaseError(ctx, mc, err)
		return err
	}

	// Phase 3: Archive polling
	if err := e.phaseArchivePolling(ctx, mc); err != nil {
		e.handlePhaseError(ctx, mc, err)
		return err
	}

	// Phase 4: Migration start
	if err := e.phaseMigrationStart(ctx, mc); err != nil {
		e.handlePhaseError(ctx, mc, err)
		return err
	}

	// Phase 5: Migration polling
	if err := e.phaseMigrationPolling(ctx, mc); err != nil {
		e.handlePhaseError(ctx, mc, err)
		return err
	}

	// Phase 6: Post-migration validation (errors are logged but don't fail migration)
	if err := e.phasePostMigration(ctx, mc); err != nil {
		e.logger.Warn("Post-migration phase returned error", "error", err)
	}

	// Phase 7: Completion
	return e.phaseCompletion(ctx, mc)
}
