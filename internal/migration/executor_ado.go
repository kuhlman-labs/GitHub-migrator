package migration

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/shurcooL/githubv4"
)

// ExecuteADOMigration executes a migration from Azure DevOps to GitHub
// ADO migrations use a different flow than GitHub migrations:
// - No archive generation (GEI pulls directly from ADO)
// - No source repository locking (ADO doesn't support it)
// - Uses ADO-specific GraphQL mutation fields
//
//nolint:gocyclo // Complex migration flow requires multiple steps and error handling
func (e *Executor) ExecuteADOMigration(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	e.logger.Info("Starting ADO migration",
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
		fmt.Sprintf("Starting %s for ADO repository", map[bool]string{true: "dry run", false: "migration"}[dryRun]), nil)

	// Phase 1: Pre-migration validation
	e.logger.Info("Running pre-migration validation", "repo", repo.FullName)
	e.logOperation(ctx, repo, historyID, "INFO", "pre_migration", "validate", "Running pre-migration validation", nil)

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
	status := models.StatusMigratingContent
	if dryRun {
		status = models.StatusDryRunInProgress
	}
	repo.Status = string(status)
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	// Phase 2: Start ADO migration
	// Note: ADO migrations don't require archive generation
	// GEI pulls directly from ADO using the provided PAT
	e.logger.Info("Starting ADO repository migration on GitHub", "repo", repo.FullName)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "initiate",
		"Starting ADO-to-GitHub migration with GitHub Enterprise Importer", nil)

	migrationID, err := e.startADORepositoryMigration(ctx, repo, batch)
	if err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "migration", "initiate", "Failed to start migration", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		status := models.StatusMigrationFailed
		if dryRun {
			status = models.StatusDryRunFailed
		}
		repo.Status = string(status)
		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("failed to start migration: %w", err)
	}

	e.logger.Info("Migration started successfully",
		"repo", repo.FullName,
		"migration_id", migrationID)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "initiated",
		fmt.Sprintf("Migration started with ID: %s", migrationID), nil)

	// Phase 3: Poll migration status
	repo.Status = string(models.StatusMigratingContent)
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	e.logger.Info("Polling migration status", "repo", repo.FullName, "migration_id", migrationID)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "poll", "Polling migration status", nil)

	if err := e.pollMigrationStatus(ctx, repo, batch, historyID, migrationID); err != nil {
		// Error already logged and status updated in pollMigrationStatus
		return fmt.Errorf("migration failed: %w", err)
	}

	// Phase 4: Post-migration validation (if enabled)
	if e.shouldRunPostMigration(dryRun) {
		e.logger.Info("Running post-migration validation", "repo", repo.FullName)
		e.logOperation(ctx, repo, historyID, "INFO", "post_migration", "validate", "Running post-migration validation", nil)

		if err := e.validatePostMigration(ctx, repo); err != nil {
			errMsg := err.Error()
			e.logOperation(ctx, repo, historyID, "ERROR", "post_migration", "validate", "Post-migration validation failed", &errMsg)
			// Don't fail the entire migration, just log validation failure
			e.logger.Warn("Post-migration validation failed", "repo", repo.FullName, "error", err)
		} else {
			e.logOperation(ctx, repo, historyID, "INFO", "post_migration", "validate", "Post-migration validation passed", nil)
		}
	}

	// Update final status
	status = models.StatusMigrationComplete
	if dryRun {
		status = models.StatusDryRunComplete
	}
	repo.Status = string(status)
	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	// Update history as completed
	e.updateHistoryStatus(ctx, historyID, "completed", nil)

	e.logger.Info("ADO migration completed successfully",
		"repo", repo.FullName,
		"dry_run", dryRun)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "complete", "Migration completed successfully", nil)

	return nil
}

// startADORepositoryMigration starts a migration from Azure DevOps to GitHub using GraphQL
// This uses the GitHub Enterprise Importer API with ADO-specific parameters
func (e *Executor) startADORepositoryMigration(ctx context.Context, repo *models.Repository, batch *models.Batch) (string, error) {
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

	// Create ADO migration source if not already cached
	migSourceID, err := e.getOrCreateADOMigrationSource(ctx, destOrgID)
	if err != nil {
		return "", fmt.Errorf("failed to get ADO migration source ID: %w", err)
	}

	// Build ADO repository URL
	// Format: https://dev.azure.com/{org}/{project}/_git/{repo}
	// The repo.SourceURL should already be in this format from discovery
	if repo.SourceURL == "" {
		return "", fmt.Errorf("repository missing source URL")
	}

	// Apply visibility transformation
	targetVisibility := e.determineTargetVisibility(repo.Visibility)
	targetRepoVisibility := githubv4.String(targetVisibility)

	e.logger.Info("Applying visibility transformation",
		"repo", repo.FullName,
		"source_visibility", repo.Visibility,
		"target_visibility", targetVisibility)

	// Get the destination repository name
	destRepoName := e.getDestinationRepoName(repo)

	// Prepare ADO mutation
	// Per GitHub docs: https://docs.github.com/en/migrations/using-github-enterprise-importer/
	// migrating-from-azure-devops-to-github-enterprise-cloud
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

	// ADO PAT for accessing the source repository
	// This should be the server-level ADO PAT configured in SourceConfig
	adoPAT := githubv4.String(e.sourceClient.Token())

	// GitHub PAT for the destination
	githubPAT := githubv4.String(e.destClient.Token())

	continueOnError := githubv4.Boolean(true)

	// Build ADO repository URL with embedded PAT for authentication
	// Format: https://{PAT}@dev.azure.com/{org}/{project}/_git/{repo}
	sourceRepoURL, err := url.Parse(repo.SourceURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse source URL: %w", err)
	}

	// Embed the PAT in the URL for ADO authentication
	// GEI uses this to authenticate with ADO
	sourceRepoURL.User = url.User(e.sourceClient.Token())
	sourceRepoURI := githubv4.URI{URL: sourceRepoURL}

	// Build input for ADO migration
	input := githubv4.StartRepositoryMigrationInput{
		SourceID:             githubv4.ID(migSourceID),
		OwnerID:              githubv4.ID(destOrgID),
		RepositoryName:       githubv4.String(destRepoName),
		ContinueOnError:      &continueOnError,
		TargetRepoVisibility: &targetRepoVisibility,
		SourceRepositoryURL:  sourceRepoURI, // ADO repo URL with embedded PAT
		AccessToken:          &adoPAT,       // ADO PAT for source access
		GitHubPat:            &githubPAT,    // GitHub PAT for destination
	}

	// Note: For ADO migrations, we don't provide GitArchiveURL or MetadataArchiveURL
	// GEI pulls directly from ADO using the SourceRepositoryURL with embedded PAT

	e.logger.Debug("Starting ADO migration with GEI",
		"repo", repo.FullName,
		"dest_org", destOrgName,
		"dest_repo", destRepoName,
		"source_url", repo.SourceURL)

	// Execute mutation
	err = e.destClient.GraphQL().Mutate(ctx, &mutation, input, nil)
	if err != nil {
		return "", fmt.Errorf("GraphQL mutation failed: %w", err)
	}

	migrationID := string(mutation.StartRepositoryMigration.RepositoryMigration.ID)
	migrationState := string(mutation.StartRepositoryMigration.RepositoryMigration.State)

	e.logger.Info("ADO migration started",
		"repo", repo.FullName,
		"migration_id", migrationID,
		"state", migrationState)

	return migrationID, nil
}

// getOrCreateADOMigrationSource gets or creates an Azure DevOps migration source
func (e *Executor) getOrCreateADOMigrationSource(ctx context.Context, ownerID string) (string, error) {
	// Check if we already have a cached ADO migration source ID
	if e.migSourceID != "" {
		return e.migSourceID, nil
	}

	// Create a new ADO migration source
	var mutation struct {
		CreateMigrationSource struct {
			MigrationSource struct {
				ID   githubv4.String
				Name githubv4.String
				Type githubv4.String
			}
		} `graphql:"createMigrationSource(input: $input)"`
	}

	sourceName := githubv4.String("Azure DevOps")
	sourceType := githubv4.MigrationSourceTypeAzureDevOps

	input := githubv4.CreateMigrationSourceInput{
		Name:    sourceName,
		Type:    sourceType,
		OwnerID: githubv4.ID(ownerID),
	}

	e.logger.Debug("Creating ADO migration source",
		"owner_id", ownerID,
		"source_name", sourceName)

	err := e.destClient.GraphQL().Mutate(ctx, &mutation, input, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create ADO migration source: %w", err)
	}

	sourceID := string(mutation.CreateMigrationSource.MigrationSource.ID)
	e.migSourceID = sourceID // Cache it

	e.logger.Info("ADO migration source created",
		"source_id", sourceID,
		"source_type", mutation.CreateMigrationSource.MigrationSource.Type)

	return sourceID, nil
}
