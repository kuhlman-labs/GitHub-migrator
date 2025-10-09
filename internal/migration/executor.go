package migration

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
	ghapi "github.com/google/go-github/v75/github"
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

// Executor handles repository migrations from GHES to GHEC
type Executor struct {
	sourceClient         *github.Client // GHES client
	destClient           *github.Client // GHEC client
	storage              *storage.Database
	orgIDCache           map[string]string // Cache of org name -> org ID
	migSourceID          string            // Cached migration source ID (created on first use)
	logger               *slog.Logger
	postMigrationMode    PostMigrationMode           // When to run post-migration tasks
	destRepoExistsAction DestinationRepoExistsAction // What to do if destination repo exists
}

// ExecutorConfig configures the migration executor
type ExecutorConfig struct {
	SourceClient         *github.Client
	DestClient           *github.Client
	Storage              *storage.Database
	Logger               *slog.Logger
	PostMigrationMode    PostMigrationMode           // When to run post-migration tasks (default: production_only)
	DestRepoExistsAction DestinationRepoExistsAction // What to do if destination repo exists (default: fail)
}

// ArchiveURLs contains the URLs for migration archives
type ArchiveURLs struct {
	GitSource string
	Metadata  string
}

// NewExecutor creates a new migration executor
func NewExecutor(cfg ExecutorConfig) (*Executor, error) {
	if cfg.SourceClient == nil {
		return nil, fmt.Errorf("source client is required")
	}
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

	return &Executor{
		sourceClient:         cfg.SourceClient,
		destClient:           cfg.DestClient,
		storage:              cfg.Storage,
		orgIDCache:           make(map[string]string),
		logger:               cfg.Logger,
		postMigrationMode:    postMigMode,
		destRepoExistsAction: destRepoAction,
	}, nil
}

// getDestinationOrg returns the destination org for a repository
// Defaults to the source org if not explicitly set
func (e *Executor) getDestinationOrg(repo *models.Repository) string {
	// If DestinationFullName is set, extract org from it
	if repo.DestinationFullName != nil && *repo.DestinationFullName != "" {
		parts := strings.Split(*repo.DestinationFullName, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	// Default to source org
	parts := strings.Split(repo.FullName, "/")
	if len(parts) >= 1 {
		return parts[0]
	}

	return ""
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
//
//nolint:gocyclo // Sequential state machine with multiple phases requires complexity
func (e *Executor) ExecuteMigration(ctx context.Context, repo *models.Repository, dryRun bool) error {
	e.logger.Info("Starting migration",
		"repo", repo.FullName,
		"dry_run", dryRun)

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

	if err := e.validatePreMigration(ctx, repo); err != nil {
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

	e.logger.Info("Generating archives on GHES", "repo", repo.FullName, "mode", migrationMode)
	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "initiate",
		fmt.Sprintf("Initiating archive generation on GHES (%s)", migrationMode), nil)

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
		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("failed to generate archives: %w", err)
	}

	details := fmt.Sprintf("Archive ID: %d", archiveID)
	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "initiate", "Archive generation initiated successfully", &details)

	repo.Status = string(models.StatusArchiveGenerating)
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
		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("archive generation failed: %w", err)
	}

	e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "complete", "Archives generated successfully", nil)

	// Phase 4: Start migration on GHEC using GraphQL
	e.logger.Info("Starting migration on GHEC", "repo", repo.FullName)
	e.logOperation(ctx, repo, historyID, "INFO", "migration_start", "initiate", "Starting migration on destination", nil)

	migrationID, err := e.startRepositoryMigration(ctx, repo, archiveURLs)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "migration_start", "initiate", "Failed to start migration", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		repo.Status = string(models.StatusMigrationFailed)
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

	if err := e.pollMigrationStatus(ctx, repo, historyID, migrationID); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "migration_progress", "poll", "Migration failed", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		repo.Status = string(models.StatusMigrationFailed)
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
	if dryRun {
		completionStatus = models.StatusDryRunComplete
		completionMsg = "Dry run completed successfully - repository can be migrated safely"
	}

	e.logger.Info("Migration complete", "repo", repo.FullName, "dry_run", dryRun)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "complete", completionMsg, nil)
	e.updateHistoryStatus(ctx, historyID, "completed", nil)

	repo.Status = string(completionStatus)
	now := time.Now()

	// Only set MigratedAt for production migrations, not dry runs
	if !dryRun {
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
		return 0, fmt.Errorf("failed to start migration on GHES: %w", err)
	}

	if migration == nil || migration.ID == nil {
		return 0, fmt.Errorf("invalid migration response from GHES")
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
				return nil, fmt.Errorf("archive generation failed on GHES")

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
func (e *Executor) startRepositoryMigration(ctx context.Context, repo *models.Repository, urls *ArchiveURLs) (string, error) {
	// Get destination org name for this repository
	destOrgName := e.getDestinationOrg(repo)
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
	targetRepoVisibility := githubv4.String("private")
	gitArchiveURL := githubv4.String(urls.GitSource)
	metadataArchiveURL := githubv4.String(urls.Metadata)

	// Get tokens - IMPORTANT: Per GitHub documentation:
	// - AccessToken: Personal access token for the SOURCE (GHES)
	// - GitHubPat: Personal access token for the DESTINATION (GHEC)
	sourceToken := githubv4.String(e.sourceClient.Token()) // GHES token
	destToken := githubv4.String(e.destClient.Token())     // GHEC token

	// Use typed input struct
	input := githubv4.StartRepositoryMigrationInput{
		SourceID:             githubv4.ID(migSourceID),
		OwnerID:              githubv4.ID(destOrgID),
		RepositoryName:       githubv4.String(repo.Name()),
		ContinueOnError:      &continueOnError,
		TargetRepoVisibility: &targetRepoVisibility,
		SourceRepositoryURL:  sourceRepoURI,
		GitArchiveURL:        &gitArchiveURL,
		MetadataArchiveURL:   &metadataArchiveURL,
		AccessToken:          &sourceToken, // Source GHES token
		GitHubPat:            &destToken,   // Destination GHEC token
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
func (e *Executor) pollMigrationStatus(ctx context.Context, repo *models.Repository, historyID *int64, migrationID string) error {
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
				// Set destination details
				destFullName := fmt.Sprintf("%s/%s", repo.Organization(), repo.Name())
				repo.DestinationFullName = &destFullName
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
func (e *Executor) validatePreMigration(ctx context.Context, repo *models.Repository) error {
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

	// 1. Verify source repository exists and is accessible
	e.logger.Info("Checking source repository", "repo", repo.FullName)
	var sourceRepoData *ghapi.Repository
	var err error

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

	// 2. Check if destination repository already exists
	e.logger.Info("Checking destination repository", "repo", repo.FullName, "action", e.destRepoExistsAction)
	var destRepoData *ghapi.Repository
	destExists := false

	_, err = e.destClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		destRepoData, resp, err = e.destClient.REST().Repositories.Get(ctx, repo.Organization(), repo.Name())
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
				"repo", repo.FullName,
				"dest_repo", destRepoData.GetFullName())

			// Delete the existing repository
			_, err = e.destClient.DoWithRetry(ctx, "DeleteRepository", func(ctx context.Context) (*ghapi.Response, error) {
				resp, err := e.destClient.REST().Repositories.Delete(ctx, repo.Organization(), repo.Name())
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

// validatePostMigration performs post-migration validation
func (e *Executor) validatePostMigration(ctx context.Context, repo *models.Repository) error {
	if repo.DestinationFullName == nil {
		return fmt.Errorf("destination repository not set")
	}

	// Get destination repository
	var destRepo *ghapi.Repository
	var err error

	_, err = e.destClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		destRepo, resp, err = e.destClient.REST().Repositories.Get(ctx, repo.Organization(), repo.Name())
		return resp, err
	})

	if err != nil {
		return fmt.Errorf("destination repository not found: %w", err)
	}

	// Verify basic properties
	var warnings []string

	// Check default branch
	if repo.DefaultBranch != nil && destRepo.GetDefaultBranch() != *repo.DefaultBranch {
		warnings = append(warnings, fmt.Sprintf("Default branch mismatch: expected %s, got %s",
			*repo.DefaultBranch, destRepo.GetDefaultBranch()))
	}

	// Note: Branch and commit counts require additional API calls
	// This is a simplified validation

	if len(warnings) > 0 {
		e.logger.Warn("Post-migration validation warnings",
			"repo", repo.FullName,
			"warnings", warnings)
	}

	e.logger.Info("Post-migration validation passed", "repo", repo.FullName)
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
