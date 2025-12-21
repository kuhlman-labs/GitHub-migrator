package migration

import (
	"context"
	"fmt"
	"net/url"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/shurcooL/githubv4"
)

// generateArchivesOnGHES creates separate git and metadata migration archives on GHES using REST API.
// This generates two archives: one for git data only (exclude_metadata=true) and one for metadata only (exclude_git_data=true).
// This approach helps with large repositories by keeping each archive smaller and more manageable.
func (e *Executor) generateArchivesOnGHES(ctx context.Context, repo *models.Repository, batch *models.Batch, lockRepositories bool) (*ArchiveIDs, error) {
	// Determine exclusion flags from repo and batch settings
	excludeReleases := e.shouldExcludeReleases(repo, batch)
	excludeAttachments := e.shouldExcludeAttachments(repo, batch)

	// Check if we need to exclude releases due to size (override if not already set)
	if !excludeReleases && repo.TotalSize != nil && *repo.TotalSize > 10*1024*1024*1024 { // >10GB
		excludeReleases = true
		e.logger.Info("Excluding releases due to repository size", "repo", repo.FullName, "size", *repo.TotalSize)
	}

	e.logger.Info("Generating separate git and metadata archives",
		"repo", repo.FullName,
		"lock_repositories", lockRepositories,
		"exclude_releases", excludeReleases,
		"exclude_attachments", excludeAttachments)

	// Generate git-only archive (exclude_metadata=true, exclude_git_data=false)
	// Per GitHub docs: { "repositories": [...], "exclude_metadata": true }
	gitOpts := github.StartMigrationOptions{
		Repositories:         []string{repo.Name()},
		LockRepositories:     lockRepositories,
		ExcludeMetadata:      true,  // Git-only archive
		ExcludeGitData:       false, // Include git data
		ExcludeAttachments:   false, // Attachments are metadata, not relevant for git archive
		ExcludeReleases:      false, // Release assets are metadata, git archive only has tags
		ExcludeOwnerProjects: true,  // Owner projects are not relevant for git archive
	}

	gitMigration, err := e.sourceClient.StartMigrationWithOptions(ctx, repo.Organization(), gitOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to start git archive generation for repository %s: %w", repo.FullName, err)
	}

	if gitMigration == nil || gitMigration.ID == nil {
		return nil, fmt.Errorf("invalid git migration response from source: received nil migration or nil migration ID")
	}

	e.logger.Info("Git archive generation started",
		"repo", repo.FullName,
		"git_archive_id", *gitMigration.ID)

	// CRITICAL: Set SourceMigrationID immediately after git archive is created.
	// This ensures the repository can be unlocked if metadata archive generation fails.
	// The git archive locks the repository, so we must track the migration ID before proceeding.
	if lockRepositories {
		gitMigrationID := *gitMigration.ID
		repo.SourceMigrationID = &gitMigrationID
		repo.IsSourceLocked = true
		if err := e.storage.UpdateRepository(ctx, repo); err != nil {
			e.logger.Error("Failed to persist source migration ID after git archive creation",
				"error", err,
				"repo", repo.FullName,
				"git_archive_id", gitMigrationID)
			// Continue anyway - the migration ID is in memory and can be used for unlock
		}
	}

	// Generate metadata-only archive (exclude_metadata=false, exclude_git_data=true)
	// Per GitHub docs: { "repositories": [...], "exclude_git_data": true, "exclude_releases": false, "exclude_owner_projects": true }
	metadataOpts := github.StartMigrationOptions{
		Repositories:         []string{repo.Name()},
		LockRepositories:     false, // Don't lock again for metadata archive
		ExcludeMetadata:      false, // Include metadata
		ExcludeGitData:       true,  // Metadata-only archive
		ExcludeAttachments:   excludeAttachments,
		ExcludeReleases:      excludeReleases, // User preference (docs show false as default)
		ExcludeOwnerProjects: true,            // Per docs: exclude organization projects
	}

	metadataMigration, err := e.sourceClient.StartMigrationWithOptions(ctx, repo.Organization(), metadataOpts)
	if err != nil {
		// Git archive succeeded but metadata failed - repo is already locked and SourceMigrationID is set
		// The caller's error handler will be able to unlock the repository
		return nil, fmt.Errorf("failed to start metadata archive generation for repository %s: %w", repo.FullName, err)
	}

	if metadataMigration == nil || metadataMigration.ID == nil {
		return nil, fmt.Errorf("invalid metadata migration response from source: received nil migration or nil migration ID")
	}

	e.logger.Info("Metadata archive generation started",
		"repo", repo.FullName,
		"metadata_archive_id", *metadataMigration.ID)

	return &ArchiveIDs{
		GitArchiveID:      *gitMigration.ID,
		MetadataArchiveID: *metadataMigration.ID,
	}, nil
}

// pollArchiveGeneration polls for both git and metadata archive generation completion
// Uses adaptive polling: fast polling initially, then backs off to preserve rate limits
func (e *Executor) pollArchiveGeneration(ctx context.Context, repo *models.Repository, historyID *int64, archiveIDs *ArchiveIDs) (*ArchiveURLs, error) {
	startTime := time.Now()
	timeoutDeadline := startTime.Add(archiveTimeout)

	var gitArchiveURL, metadataArchiveURL string
	gitDone, metadataDone := false, false
	lastInterval := archiveInitialInterval

	// Initial poll immediately
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			// Check timeout
			if time.Now().After(timeoutDeadline) {
				return nil, fmt.Errorf("archive generation timeout exceeded (24 hours)")
			}

			// Poll git archive if not done
			if !gitDone {
				done, url, err := e.checkArchiveStatus(ctx, repo, archiveIDs.GitArchiveID, "git")
				if err != nil {
					return nil, err
				}
				if done {
					gitArchiveURL = url
					gitDone = true
				}
			}

			// Poll metadata archive if not done
			if !metadataDone {
				done, url, err := e.checkArchiveStatus(ctx, repo, archiveIDs.MetadataArchiveID, "metadata")
				if err != nil {
					return nil, err
				}
				if done {
					metadataArchiveURL = url
					metadataDone = true
				}
			}

			// Both archives ready
			if gitDone && metadataDone {
				elapsed := time.Since(startTime)
				e.logger.Info("Both archives ready",
					"repo", repo.FullName,
					"elapsed", elapsed.Round(time.Second))
				return &ArchiveURLs{
					GitSource: gitArchiveURL,
					Metadata:  metadataArchiveURL,
				}, nil
			}

			// Calculate next polling interval using adaptive backoff
			elapsed := time.Since(startTime)
			nextInterval := calculateAdaptivePollInterval(elapsed, archiveInitialInterval, archiveMaxInterval, archiveFastPhaseDuration)

			// Log progress with interval info (only when interval changes significantly)
			if historyID != nil {
				msg := fmt.Sprintf("Archive generation in progress (git: %v, metadata: %v, next_poll: %v)", gitDone, metadataDone, nextInterval.Round(time.Second))
				e.logOperation(ctx, repo, historyID, "INFO", "archive_generation", "poll", msg, nil)
			}

			// Log when transitioning to slower polling
			if nextInterval > lastInterval {
				e.logger.Info("Adjusting archive poll interval",
					"repo", repo.FullName,
					"previous_interval", lastInterval.Round(time.Second),
					"new_interval", nextInterval.Round(time.Second),
					"elapsed", elapsed.Round(time.Second))
				lastInterval = nextInterval
			}

			// Schedule next poll
			timer.Reset(nextInterval)
		}
	}
}

// checkArchiveStatus checks a single archive's status and returns (done, url, error)
func (e *Executor) checkArchiveStatus(ctx context.Context, repo *models.Repository, archiveID int64, archiveType string) (bool, string, error) {
	state, url, err := e.pollSingleArchive(ctx, repo, archiveID)
	if err != nil {
		return false, "", fmt.Errorf("%s archive generation failed: %w", archiveType, err)
	}
	if state == statusExported {
		e.logger.Info(fmt.Sprintf("%s archive ready", archiveType), "repo", repo.FullName, "archive_id", archiveID)
		return true, url, nil
	}
	if state == statusFailed {
		return false, "", fmt.Errorf("%s archive generation failed for repository %s (migration ID: %d)", archiveType, repo.FullName, archiveID)
	}
	return false, "", nil
}

// pollSingleArchive polls a single archive and returns its state and URL (if ready)
func (e *Executor) pollSingleArchive(ctx context.Context, repo *models.Repository, archiveID int64) (string, string, error) {
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
		return "", "", fmt.Errorf("failed to check migration status: %w", err)
	}

	state := migration.GetState()
	e.logger.Debug("Archive generation status", "repo", repo.FullName, "archive_id", archiveID, "state", state)

	if state == statusExported {
		// Archive is ready, get download URL
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
			return "", "", fmt.Errorf("failed to get archive URL: %w", err)
		}

		return state, archiveURL, nil
	}

	return state, "", nil
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
// Uses adaptive polling: fast polling initially, then backs off to preserve rate limits
func (e *Executor) pollMigrationStatus(ctx context.Context, repo *models.Repository, batch *models.Batch, historyID *int64, migrationID string) error {
	startTime := time.Now()
	timeoutDeadline := startTime.Add(migrationTimeout)
	lastInterval := migrationInitialInterval

	// Initial poll immediately
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			// Check timeout
			if time.Now().After(timeoutDeadline) {
				return fmt.Errorf("migration timeout exceeded (48 hours)")
			}

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
				elapsed := time.Since(startTime)
				e.logger.Info("Migration completed successfully",
					"repo", repo.FullName,
					"elapsed", elapsed.Round(time.Second))
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

				// Calculate next polling interval using adaptive backoff
				elapsed := time.Since(startTime)
				nextInterval := calculateAdaptivePollInterval(elapsed, migrationInitialInterval, migrationMaxInterval, migrationFastPhaseDuration)

				if historyID != nil {
					msg := fmt.Sprintf("Migration in progress (state: %s, next_poll: %v)", state, nextInterval.Round(time.Second))
					e.logOperation(ctx, repo, historyID, "INFO", "migration_progress", "poll", msg, nil)
				}

				// Log when transitioning to slower polling
				if nextInterval > lastInterval {
					e.logger.Info("Adjusting migration poll interval",
						"repo", repo.FullName,
						"previous_interval", lastInterval.Round(time.Second),
						"new_interval", nextInterval.Round(time.Second),
						"elapsed", elapsed.Round(time.Second))
					lastInterval = nextInterval
				}

				// Schedule next poll
				timer.Reset(nextInterval)
				continue

			default:
				e.logger.Warn("Unknown migration state", "state", state)
				// Calculate next polling interval even for unknown states
				elapsed := time.Since(startTime)
				nextInterval := calculateAdaptivePollInterval(elapsed, migrationInitialInterval, migrationMaxInterval, migrationFastPhaseDuration)
				timer.Reset(nextInterval)
				continue
			}
		}
	}
}
