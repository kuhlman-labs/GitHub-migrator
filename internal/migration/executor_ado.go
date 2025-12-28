package migration

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/ado"
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

	// Phase 1.5: Validate ADO PAT access to source repository
	// This catches permission issues before GitHub's preflight checks
	e.logger.Info("Validating ADO PAT access to source repository", "repo", repo.FullName)
	e.logOperation(ctx, repo, historyID, "INFO", "pre_migration", "validate_ado_access", "Validating ADO PAT can access source repository", nil)

	if err := e.validateADORepositoryAccess(ctx, repo); err != nil {
		errMsg := err.Error()
		e.logOperation(ctx, repo, historyID, "ERROR", "pre_migration", "validate_ado_access", "ADO PAT validation failed", &errMsg)
		e.updateHistoryStatus(ctx, historyID, "failed", &errMsg)

		status := models.StatusMigrationFailed
		if dryRun {
			status = models.StatusDryRunFailed
		}
		repo.Status = string(status)
		if updateErr := e.storage.UpdateRepository(ctx, repo); updateErr != nil {
			e.logger.Error("Failed to update repository status", "error", updateErr)
		}
		return fmt.Errorf("ADO PAT validation failed: %w", err)
	}
	e.logOperation(ctx, repo, historyID, "INFO", "pre_migration", "validate_ado_access", "ADO PAT has access to source repository", nil)

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
	} else {
		reason := fmt.Sprintf("Skipping post-migration validation (mode: %s, dry_run: %v)", e.postMigrationMode, dryRun)
		e.logger.Info(reason, "repo", repo.FullName)
		e.logOperation(ctx, repo, historyID, "INFO", "post_migration", "skip", reason, nil)
	}

	// Phase 5: Mark complete (matches GitHub migration flow)
	completionStatus := models.StatusComplete
	completionMsg := msgMigrationComplete

	if dryRun {
		completionStatus = models.StatusDryRunComplete
		completionMsg = msgDryRunComplete
	}

	e.logger.Info("ADO migration complete", "repo", repo.FullName, "dry_run", dryRun)
	e.logOperation(ctx, repo, historyID, "INFO", "migration", "complete", completionMsg, nil)
	e.updateHistoryStatus(ctx, historyID, "completed", nil)

	repo.Status = string(completionStatus)
	now := time.Now()

	// Set appropriate timestamps based on migration type
	if dryRun {
		// Set last dry run timestamp
		repo.LastDryRunAt = &now
	} else {
		// Set migration completion timestamp
		repo.MigratedAt = &now
	}

	if err := e.storage.UpdateRepository(ctx, repo); err != nil {
		e.logger.Error("Failed to update repository status", "error", err)
	}

	return nil
}

// startADORepositoryMigration starts a migration from Azure DevOps to GitHub using GraphQL
// This uses the GitHub Enterprise Importer API with ADO-specific parameters
//
//nolint:gocyclo // Complexity is inherent to ADO migration orchestration
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

	// Create or get ADO migration source
	// Per GitHub docs, the migration source URL should be just "https://dev.azure.com"
	// The specific org/project/repo is specified in sourceRepositoryUrl parameter
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
	if e.sourceToken == "" {
		return "", fmt.Errorf("source token (ADO PAT) is required for ADO migrations")
	}
	adoPAT := githubv4.String(e.sourceToken)

	// GitHub PAT for the destination
	// Must have: repo, admin:org, workflow scopes
	// MUST be a classic PAT (not fine-grained) - GEI doesn't support fine-grained PATs for migrations
	githubToken := e.destClient.Token()
	if githubToken == "" {
		return "", fmt.Errorf("destination GitHub token is required for ADO migrations")
	}

	// Validate GitHub PAT format
	// Classic PATs start with ghp_ or gho_
	// Fine-grained PATs start with github_pat_ (NOT supported by GEI)
	if strings.HasPrefix(githubToken, "github_pat_") {
		return "", fmt.Errorf("fine-grained GitHub PATs are not supported for migrations - please use a classic PAT (ghp_ or gho_) with repo, admin:org, workflow scopes")
	}

	// Get PAT prefix for logging (safe slice)
	patPrefixLen := min(len(githubToken), 10)
	patPrefix := githubToken[:patPrefixLen]

	if !strings.HasPrefix(githubToken, "ghp_") && !strings.HasPrefix(githubToken, "gho_") {
		e.logger.Warn("GitHub PAT format may be invalid - expected classic PAT starting with ghp_ or gho_", "prefix", patPrefix)
	}

	githubPAT := githubv4.String(githubToken)

	// Validate ADO PAT format (should be 52 characters, base64-encoded)
	if len(e.sourceToken) < 40 {
		return "", fmt.Errorf("ADO PAT appears to be too short (length: %d) - ensure you're using a valid Azure DevOps Personal Access Token", len(e.sourceToken))
	}

	e.logger.Info("Validating migration PATs",
		"repo", repo.FullName,
		"has_ado_pat", len(e.sourceToken) > 0,
		"ado_pat_length", len(e.sourceToken),
		"has_github_pat", len(githubToken) > 0,
		"github_pat_prefix", patPrefix)

	continueOnError := githubv4.Boolean(true)

	// Build ADO repository URL WITHOUT embedded credentials
	// Format: https://dev.azure.com/{org}/{project}/_git/{repo}
	// The PAT is passed separately via AccessToken field
	sourceRepoURL, err := url.Parse(repo.SourceURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse source URL: %w", err)
	}

	// Ensure no credentials are embedded in the URL
	// GEI expects the PAT to be passed via AccessToken, not embedded in URL
	sourceRepoURL.User = nil

	// Validate ADO URL format
	if !strings.Contains(sourceRepoURL.Host, "dev.azure.com") && !strings.Contains(sourceRepoURL.Host, "visualstudio.com") {
		e.logger.Warn("Source URL may not be a valid Azure DevOps URL", "host", sourceRepoURL.Host, "full_url", sourceRepoURL.String())
	}
	if !strings.Contains(sourceRepoURL.Path, "/_git/") {
		return "", fmt.Errorf("invalid ADO repository URL format - expected path to contain '/_git/', got: %s", sourceRepoURL.Path)
	}

	sourceRepoURI := githubv4.URI{URL: sourceRepoURL}

	e.logger.Info("ADO source repository URL validated",
		"repo", repo.FullName,
		"source_url", sourceRepoURL.String(),
		"host", sourceRepoURL.Host,
		"path", sourceRepoURL.Path)

	// Build input for ADO migration
	// Both AccessToken (ADO PAT) and GitHubPat (GitHub PAT) are required
	input := githubv4.StartRepositoryMigrationInput{
		SourceID:             githubv4.ID(migSourceID),
		OwnerID:              githubv4.ID(destOrgID),
		RepositoryName:       githubv4.String(destRepoName),
		ContinueOnError:      &continueOnError,
		TargetRepoVisibility: &targetRepoVisibility,
		SourceRepositoryURL:  sourceRepoURI, // ADO repo URL (clean, no credentials)
		AccessToken:          &adoPAT,       // ADO PAT for source access
		GitHubPat:            &githubPAT,    // GitHub PAT for destination (required by GEI)
	}

	// Note: For ADO migrations, we don't provide GitArchiveURL or MetadataArchiveURL
	// GEI pulls directly from ADO using the SourceRepositoryURL + AccessToken

	e.logger.Info("Starting ADO migration with GEI",
		"repo", repo.FullName,
		"dest_org", destOrgName,
		"dest_repo", destRepoName,
		"migration_source_id", migSourceID,
		"source_url", sourceRepoURL.String(),
		"has_ado_pat", e.sourceToken != "",
		"visibility", targetVisibility)

	// Execute mutation
	err = e.destClient.GraphQL().Mutate(ctx, &mutation, input, nil)
	if err != nil {
		errMsg := err.Error()

		// Detailed logging for troubleshooting
		patPrefix := "unknown"
		if len(githubToken) >= 10 {
			patPrefix = githubToken[:10]
		} else if len(githubToken) > 0 {
			patPrefix = githubToken[:]
		}

		e.logger.Error("Failed to start ADO repository migration",
			"repo", repo.FullName,
			"error", errMsg,
			"migration_source_id", migSourceID,
			"source_url", sourceRepoURL.String(),
			"dest_org", destOrgName,
			"dest_repo", destRepoName,
			"github_pat_prefix", patPrefix,
			"ado_pat_length", len(e.sourceToken))

		// Check for common issues and provide helpful error messages
		if strings.Contains(errMsg, "githubPat") || strings.Contains(errMsg, "github_pat") {
			return "", fmt.Errorf("GitHub PAT validation failed - ensure your destination PAT: 1) is a classic PAT (starts with ghp_ or gho_), 2) has scopes: repo, admin:org, workflow, 3) has SSO authorization if your org requires it: %w", err)
		}
		if strings.Contains(errMsg, "accessToken") || strings.Contains(errMsg, "access_token") {
			return "", fmt.Errorf("ADO PAT validation failed - ensure your source PAT: 1) is valid, 2) has 'Code: Read' scope, 3) has access to the source organization: %w", err)
		}
		if strings.Contains(errMsg, "sourceRepositoryUrl") || strings.Contains(errMsg, "repository") {
			return "", fmt.Errorf("source repository URL validation failed - ensure: 1) URL format is correct (https://dev.azure.com/{org}/{project}/_git/{repo}), 2) repository exists and is accessible: %w", err)
		}
		if strings.Contains(errMsg, "preflight") {
			return "", fmt.Errorf("preflight checks failed - this usually means: 1) ADO PAT cannot access the source repo, 2) GitHub PAT lacks required scopes, or 3) source repository URL is incorrect. Check logs above for details: %w", err)
		}

		return "", fmt.Errorf("GraphQL mutation failed: %w", err)
	}

	migrationID := string(mutation.StartRepositoryMigration.RepositoryMigration.ID)
	migrationState := string(mutation.StartRepositoryMigration.RepositoryMigration.State)

	e.logger.Info("ADO migration started successfully",
		"repo", repo.FullName,
		"migration_id", migrationID,
		"state", migrationState,
		"source_url", sourceRepoURL.String())

	return migrationID, nil
}

// getOrCreateADOMigrationSource gets or creates an Azure DevOps migration source
// Per GitHub docs, the migration source URL should be just "https://dev.azure.com"
// The specific org/project/repo is specified in the sourceRepositoryUrl parameter
func (e *Executor) getOrCreateADOMigrationSource(ctx context.Context, ownerID string) (string, error) {
	// ADO base URL per GitHub documentation
	const adoBaseURL = "https://dev.azure.com"

	// Check if we already have a cached ADO migration source ID
	if cachedID, exists := e.adoMigSourceCache[adoBaseURL]; exists {
		e.logger.Debug("Using cached ADO migration source",
			"base_url", adoBaseURL,
			"source_id", cachedID)
		return cachedID, nil
	}

	// Create a new ADO migration source
	var mutation struct {
		CreateMigrationSource struct {
			MigrationSource struct {
				ID   githubv4.String
				Name githubv4.String
				Type githubv4.String
				URL  githubv4.String
			}
		} `graphql:"createMigrationSource(input: $input)"`
	}

	sourceName := githubv4.String("Azure DevOps")
	sourceType := githubv4.MigrationSourceTypeAzureDevOps
	urlPtr := githubv4.String(adoBaseURL)

	input := githubv4.CreateMigrationSourceInput{
		Name:    sourceName,
		URL:     &urlPtr,
		Type:    sourceType,
		OwnerID: githubv4.ID(ownerID),
	}

	e.logger.Info("Creating ADO migration source",
		"owner_id", ownerID,
		"source_name", sourceName,
		"base_url", adoBaseURL)

	err := e.destClient.GraphQL().Mutate(ctx, &mutation, input, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create ADO migration source: %w", err)
	}

	sourceID := string(mutation.CreateMigrationSource.MigrationSource.ID)

	// Cache it for reuse
	e.adoMigSourceCache[adoBaseURL] = sourceID

	e.logger.Info("ADO migration source created",
		"base_url", adoBaseURL,
		"source_id", sourceID,
		"source_type", mutation.CreateMigrationSource.MigrationSource.Type,
		"source_url", mutation.CreateMigrationSource.MigrationSource.URL)

	return sourceID, nil
}

// validateADORepositoryAccess validates that the ADO PAT can access the source repository
// This helps catch permission issues before GitHub's preflight checks
func (e *Executor) validateADORepositoryAccess(ctx context.Context, repo *models.Repository) error {
	if e.sourceToken == "" {
		return fmt.Errorf("ADO PAT is not configured")
	}

	if repo.SourceURL == "" {
		return fmt.Errorf("repository source URL is not set")
	}

	// Parse the ADO repository URL using the centralized ado package
	parsed, err := ado.ParseFromSourceURL(repo.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to parse ADO source URL: %w", err)
	}

	org := parsed.Organization
	project := parsed.Project
	repoName := parsed.Repository

	// Build the ADO API URL to get repository information
	// API: https://dev.azure.com/{org}/{project}/_apis/git/repositories/{repo}?api-version=7.0
	apiURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s?api-version=7.0",
		url.PathEscape(org),
		url.PathEscape(project),
		url.PathEscape(repoName))

	e.logger.Debug("Validating ADO PAT access",
		"repo", repo.FullName,
		"ado_org", org,
		"ado_project", project,
		"ado_repo", repoName,
		"api_url", apiURL)

	// Create HTTP request with PAT authentication
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// ADO uses Basic Auth with empty username and PAT as password
	// Encode as base64: ":{PAT}"
	auth := base64.StdEncoding.EncodeToString([]byte(":" + e.sourceToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	// Make the API call
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call ADO API: %w - ensure the ADO organization is accessible and your network allows outbound connections to dev.azure.com", err)
	}
	defer resp.Body.Close()

	// Read response body for better error messages
	body, _ := io.ReadAll(resp.Body)

	e.logger.Debug("ADO API response",
		"repo", repo.FullName,
		"status_code", resp.StatusCode,
		"status", resp.Status)

	// Check response status
	switch resp.StatusCode {
	case 200:
		e.logger.Info("ADO PAT successfully validated - repository is accessible",
			"repo", repo.FullName,
			"ado_org", org,
			"ado_project", project,
			"ado_repo", repoName)
		return nil

	case 401:
		return fmt.Errorf("ADO PAT authentication failed (401 Unauthorized) - ensure your PAT is valid and not expired. Required scope: 'Code: Read'. Response: %s", string(body))

	case 403:
		return fmt.Errorf("ADO PAT access denied (403 Forbidden) - your PAT may lack 'Code: Read' scope or you don't have permission to access this repository/project. Response: %s", string(body))

	case 404:
		return fmt.Errorf("ADO repository not found (404 Not Found) - verify the repository exists and your PAT has access to the organization '%s', project '%s', and repository '%s'. Response: %s", org, project, repoName, string(body))

	default:
		return fmt.Errorf("ADO API returned unexpected status %d: %s. Response: %s", resp.StatusCode, resp.Status, string(body))
	}
}
