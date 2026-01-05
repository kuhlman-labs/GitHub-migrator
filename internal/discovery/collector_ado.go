package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

// ADOCollector wraps an ADO client for discovery operations
type ADOCollector struct {
	client          *azuredevops.Client
	storage         any // Will be *storage.Database
	logger          *slog.Logger
	provider        any // Will be source.Provider
	profiler        *ADOProfiler
	workers         int             // Number of parallel workers
	sourceID        *int64          // Multi-source ID to associate with discovered entities
	progressTracker ProgressTracker // Progress tracker for discovery updates
}

// NewADOCollector creates a new Azure DevOps collector
func NewADOCollector(client *azuredevops.Client, storage any, logger *slog.Logger, provider any) *ADOCollector {
	return &ADOCollector{
		client:   client,
		storage:  storage,
		logger:   logger,
		provider: provider,
		profiler: NewADOProfiler(client, logger, provider, storage),
		workers:  5, // Default to 5 parallel workers (matches GitHub collector)
	}
}

// SetWorkers sets the number of parallel workers for processing
func (c *ADOCollector) SetWorkers(workers int) {
	if workers > 0 {
		c.workers = workers
	}
}

// SetSourceID sets the source ID to associate with discovered repositories
func (c *ADOCollector) SetSourceID(sourceID *int64) {
	c.sourceID = sourceID
}

// SetProgressTracker sets the progress tracker for discovery updates
func (c *ADOCollector) SetProgressTracker(tracker ProgressTracker) {
	c.progressTracker = tracker
}

// getTracker returns the progress tracker, or a no-op tracker if none is set
func (c *ADOCollector) getTracker() ProgressTracker {
	if c.progressTracker != nil {
		return c.progressTracker
	}
	return NoOpProgressTracker{}
}

// DiscoverADOOrganization discovers all projects and repositories in an Azure DevOps organization
func (c *ADOCollector) DiscoverADOOrganization(ctx context.Context, organization string) error {
	c.logger.Info("Starting Azure DevOps organization discovery", "organization", organization)
	tracker := c.getTracker()

	// Get all projects in the organization
	tracker.SetPhase(models.PhaseListingRepos)
	projects, err := c.client.GetProjects(ctx)
	if err != nil {
		tracker.RecordError(err)
		return fmt.Errorf("failed to get projects: %w", err)
	}

	// Count valid projects for progress tracking (ADO projects = GitHub orgs in terminology)
	validProjects := 0
	for _, p := range projects {
		if p.Name != nil {
			validProjects++
		}
	}
	tracker.SetTotalOrgs(validProjects)

	c.logger.Info("Found projects in organization",
		"organization", organization,
		"count", len(projects))

	// Process each project
	projectIndex := 0
	for _, project := range projects {
		if project.Name == nil {
			c.logger.Warn("Skipping project with nil name")
			continue
		}

		projectName := *project.Name
		tracker.StartOrg(projectName, projectIndex) // Treat ADO projects like GitHub orgs
		c.logger.Info("Discovering project", "project", projectName)

		// Save project to database
		adoProject := &models.ADOProject{
			Organization: organization,
			Name:         projectName,
			DiscoveredAt: time.Now(),
			UpdatedAt:    time.Now(),
		}

		if project.Description != nil {
			adoProject.Description = project.Description
		}
		if project.State != nil {
			state := string(*project.State)
			adoProject.State = state
		}
		if project.Visibility != nil {
			visibility := string(*project.Visibility)
			adoProject.Visibility = visibility
		}

		// Save project to database
		if db, ok := c.storage.(*storage.Database); ok {
			if err := db.SaveADOProject(ctx, adoProject); err != nil {
				c.logger.Error("Failed to save ADO project",
					"project", projectName,
					"error", err)
				tracker.RecordError(err)
				// Continue with next project
				projectIndex++
				continue
			}
		}
		c.logger.Debug("Project saved", "project", projectName)

		// Discover repositories in this project, passing the project visibility
		projectVisibility := "private" // Default to private
		if project.Visibility != nil {
			projectVisibility = string(*project.Visibility)
		}

		repoCount, err := c.discoverADOProjectWithVisibilityTracked(ctx, organization, projectName, projectVisibility, tracker)
		if err != nil {
			c.logger.Error("Failed to discover project",
				"project", projectName,
				"error", err)
			tracker.RecordError(err)
			// Continue with next project
		}
		tracker.CompleteOrg(projectName, repoCount)
		projectIndex++
	}

	c.logger.Info("Azure DevOps organization discovery complete",
		"organization", organization,
		"projects", len(projects))

	// After discovery completes, update local dependency flags
	c.logger.Info("Updating local dependency flags", "organization", organization)
	if db, ok := c.storage.(*storage.Database); ok {
		if err := db.UpdateLocalDependencyFlags(ctx); err != nil {
			c.logger.Warn("Failed to update local dependency flags", "error", err)
			// Don't fail the whole discovery if this fails
		}
	}

	return nil
}

// DiscoverADOProject discovers all repositories in a specific Azure DevOps project
func (c *ADOCollector) DiscoverADOProject(ctx context.Context, organization, projectName string) error {
	tracker := c.getTracker()

	// For single project discovery, set up progress tracking
	tracker.SetTotalOrgs(1)
	tracker.StartOrg(projectName, 0)

	// Get project details to fetch visibility
	project, err := c.client.GetProject(ctx, projectName)
	if err != nil {
		c.logger.Warn("Failed to get project details, using default visibility",
			"project", projectName,
			"error", err)
		repoCount, discErr := c.discoverADOProjectWithVisibilityTracked(ctx, organization, projectName, "private", tracker)
		tracker.CompleteOrg(projectName, repoCount)
		return discErr
	}

	projectVisibility := "private" // Default to private
	if project.Visibility != nil {
		projectVisibility = string(*project.Visibility)
	}

	repoCount, err := c.discoverADOProjectWithVisibilityTracked(ctx, organization, projectName, projectVisibility, tracker)
	tracker.CompleteOrg(projectName, repoCount)
	return err
}

// DiscoverADOProjectWithVisibility discovers all repositories in a specific Azure DevOps project with known visibility
// This is the public API that doesn't require a tracker (uses internal tracker if set)
func (c *ADOCollector) DiscoverADOProjectWithVisibility(ctx context.Context, organization, projectName, projectVisibility string) error {
	_, err := c.discoverADOProjectWithVisibilityTracked(ctx, organization, projectName, projectVisibility, c.getTracker())
	return err
}

// discoverADOProjectWithVisibilityTracked is the internal implementation with progress tracking
func (c *ADOCollector) discoverADOProjectWithVisibilityTracked(ctx context.Context, organization, projectName, projectVisibility string, tracker ProgressTracker) (int, error) {
	c.logger.Info("Starting Azure DevOps project discovery",
		"organization", organization,
		"project", projectName,
		"visibility", projectVisibility)

	// Get all repositories in the project
	tracker.SetPhase(models.PhaseListingRepos)
	repos, err := c.client.GetRepositories(ctx, projectName)
	if err != nil {
		tracker.RecordError(err)
		return 0, fmt.Errorf("failed to get repositories: %w", err)
	}

	// Update repo count for progress tracking
	tracker.AddRepos(len(repos))
	tracker.SetPhase(models.PhaseProfilingRepos)

	c.logger.Info("Found repositories in project",
		"project", projectName,
		"count", len(repos),
		"workers", c.workers)

	// Process repositories in parallel using worker pool
	if err := c.processADORepositoriesInParallelTracked(ctx, organization, projectName, projectVisibility, repos, tracker); err != nil {
		return len(repos), fmt.Errorf("failed to process repositories: %w", err)
	}

	c.logger.Info("Azure DevOps project discovery complete",
		"project", projectName,
		"repositories", len(repos))

	// After discovery completes, update local dependency flags
	c.logger.Info("Updating local dependency flags", "project", projectName)
	if db, ok := c.storage.(*storage.Database); ok {
		if err := db.UpdateLocalDependencyFlags(ctx); err != nil {
			c.logger.Warn("Failed to update local dependency flags", "error", err)
			// Don't fail the whole discovery if this fails
		}
	}

	return len(repos), nil
}

// processADORepositoriesInParallelTracked processes ADO repositories in parallel with progress tracking
func (c *ADOCollector) processADORepositoriesInParallelTracked(ctx context.Context, organization, projectName, projectVisibility string, repos []git.GitRepository, tracker ProgressTracker) error {
	jobs := make(chan *git.GitRepository, len(repos))
	errors := make(chan error, len(repos))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.adoWorkerTracked(ctx, &wg, i, organization, projectName, projectVisibility, jobs, errors, tracker)
	}

	// Send jobs (send pointers to avoid copies)
	for i := range repos {
		jobs <- &repos[i]
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(errors)

	// Collect errors
	var errs []error
	for err := range errors {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		c.logger.Warn("Repository processing completed with errors",
			"project", projectName,
			"total_repos", len(repos),
			"error_count", len(errs))
		return fmt.Errorf("encountered %d errors during repository processing (see logs for details)", len(errs))
	}

	return nil
}

// adoWorkerTracked processes ADO repositories from the jobs channel with progress tracking
func (c *ADOCollector) adoWorkerTracked(ctx context.Context, wg *sync.WaitGroup, workerID int, organization, projectName, projectVisibility string, jobs <-chan *git.GitRepository, errors chan<- error, tracker ProgressTracker) {
	defer wg.Done()

	for adoRepo := range jobs {
		if adoRepo.Name == nil {
			c.logger.Warn("Skipping repository with nil name",
				"worker_id", workerID,
				"project", projectName)
			tracker.IncrementProcessedRepos(1) // Still count as processed
			continue
		}

		repoName := *adoRepo.Name
		c.logger.Debug("Worker processing repository",
			"worker_id", workerID,
			"project", projectName,
			"repo", repoName)

		// Create repository model
		fullName := fmt.Sprintf("%s/%s/%s", organization, projectName, repoName)
		now := time.Now()

		repo := &models.Repository{
			FullName:        fullName,
			Source:          "azuredevops",
			SourceID:        c.sourceID, // Associate with multi-source
			SourceURL:       getRemoteURL(adoRepo),
			Status:          string(models.StatusPending),
			Visibility:      projectVisibility, // Use project-level visibility from ADO
			DiscoveredAt:    now,
			UpdatedAt:       now,
			LastDiscoveryAt: &now,
		}

		// Set ADO-specific fields
		repo.ADOProject = &projectName

		// Check if this is a Git repository (vs TFVC)
		if adoRepo.Id != nil {
			// Git repos have an ID, TFVC repos don't
			repo.ADOIsGit = true
		} else {
			repo.ADOIsGit = false
			// Mark TFVC repos for remediation
			repo.Status = string(models.StatusRemediationRequired)
			c.logger.Warn("TFVC repository detected - requires conversion",
				"worker_id", workerID,
				"repo", fullName)
		}

		// Set default branch if available
		if adoRepo.DefaultBranch != nil {
			defaultBranch := *adoRepo.DefaultBranch
			repo.DefaultBranch = &defaultBranch
		}

		// Set repo size if available
		if adoRepo.Size != nil {
			// Safe conversion: ADO repository sizes are reasonable and won't overflow int64
			//#nosec G115 -- ADO Size is in bytes, repository sizes won't exceed int64 max
			size := int64(*adoRepo.Size)
			repo.TotalSize = &size
		}

		// Profile repository with ADO profiler
		c.logger.Debug("Profiling repository",
			"worker_id", workerID,
			"project", projectName,
			"repo", repoName)

		if err := c.profiler.ProfileRepository(ctx, repo, adoRepo); err != nil {
			c.logger.Warn("Failed to profile repository",
				"worker_id", workerID,
				"project", projectName,
				"repo", repoName,
				"error", err)
			// Continue even if profiling fails - we have basic info
		}

		// Save repository to database
		if db, ok := c.storage.(*storage.Database); ok {
			if err := db.SaveRepository(ctx, repo); err != nil {
				c.logger.Error("Failed to save repository",
					"worker_id", workerID,
					"project", projectName,
					"repo", repoName,
					"error", err)
				errors <- err
				tracker.RecordError(err)
				tracker.IncrementProcessedRepos(1) // Still count as processed
				continue
			}
		}

		c.logger.Debug("Repository saved",
			"worker_id", workerID,
			"project", projectName,
			"repo", repoName,
			"is_git", repo.ADOIsGit)

		// Update progress
		tracker.IncrementProcessedRepos(1)
	}
}

// DiscoverADORepository discovers a single Azure DevOps repository
func (c *ADOCollector) DiscoverADORepository(ctx context.Context, organization, projectName, repoName string) error {
	c.logger.Info("Starting Azure DevOps single repository discovery",
		"organization", organization,
		"project", projectName,
		"repo", repoName)

	// Get the specific repository
	repoPtr, err := c.client.GetRepository(ctx, projectName, repoName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	if repoPtr == nil {
		return fmt.Errorf("repository not found")
	}

	// Dereference the pointer for consistency with GetRepositories
	repo := *repoPtr

	if repo.Name == nil {
		return fmt.Errorf("repository has no name")
	}

	c.logger.Debug("Processing repository",
		"project", projectName,
		"repo", repoName)

	// Create repository model
	fullName := fmt.Sprintf("%s/%s/%s", organization, projectName, repoName)
	now := time.Now()

	repoModel := &models.Repository{
		FullName:        fullName,
		Source:          "azuredevops",
		SourceID:        c.sourceID, // Associate with multi-source
		SourceURL:       getRemoteURL(repo),
		Status:          string(models.StatusPending),
		DiscoveredAt:    now,
		UpdatedAt:       now,
		LastDiscoveryAt: &now,
	}

	// Set ADO-specific fields
	repoModel.ADOProject = &projectName

	// Check if this is a Git repository (vs TFVC)
	if repo.Id != nil {
		// Git repos have an ID, TFVC repos don't
		repoModel.ADOIsGit = true
	} else {
		repoModel.ADOIsGit = false
		// Mark TFVC repos for remediation
		repoModel.Status = string(models.StatusRemediationRequired)
		c.logger.Warn("TFVC repository detected - requires conversion",
			"repo", fullName)
	}

	// Set default branch if available
	if repo.DefaultBranch != nil {
		defaultBranch := *repo.DefaultBranch
		repoModel.DefaultBranch = &defaultBranch
	}

	// Set repo size if available
	if repo.Size != nil {
		// Safe conversion: ADO repository sizes are reasonable and won't overflow int64
		//#nosec G115 -- ADO Size is in bytes, repository sizes won't exceed int64 max
		size := int64(*repo.Size)
		repoModel.TotalSize = &size
	}

	// Profile repository with ADO profiler
	c.logger.Debug("Profiling repository",
		"project", projectName,
		"repo", repoName)

	if err := c.profiler.ProfileRepository(ctx, repoModel, repo); err != nil {
		c.logger.Warn("Failed to profile repository",
			"project", projectName,
			"repo", repoName,
			"error", err)
		// Continue even if profiling fails - we have basic info
	}

	// Save repository to database
	if db, ok := c.storage.(*storage.Database); ok {
		if err := db.SaveRepository(ctx, repoModel); err != nil {
			return fmt.Errorf("failed to save repository: %w", err)
		}
	}

	c.logger.Info("Azure DevOps repository discovery complete",
		"project", projectName,
		"repo", repoName)

	// After discovery completes, update local dependency flags
	c.logger.Info("Updating local dependency flags", "repo", repoName)
	if db, ok := c.storage.(*storage.Database); ok {
		if err := db.UpdateLocalDependencyFlags(ctx); err != nil {
			c.logger.Warn("Failed to update local dependency flags", "error", err)
			// Don't fail the whole discovery if this fails
		}
	}

	return nil
}

// getRemoteURL extracts the web URL from an ADO repository (for viewing in browser)
func getRemoteURL(repo any) string {
	// Type assert to *git.GitRepository from ADO SDK
	if gitRepo, ok := repo.(*git.GitRepository); ok {
		// Use WebUrl first - this is the HTTPS URL for viewing the repo in a browser
		if gitRepo.WebUrl != nil && *gitRepo.WebUrl != "" {
			return *gitRepo.WebUrl
		}
		// Fallback to RemoteUrl (HTTPS clone URL) if WebUrl is not available
		if gitRepo.RemoteUrl != nil && *gitRepo.RemoteUrl != "" {
			return *gitRepo.RemoteUrl
		}
		// Last resort: use API URL
		if gitRepo.Url != nil && *gitRepo.Url != "" {
			return *gitRepo.Url
		}
	}

	// If we can't extract the URL, return empty string
	// The caller should handle this case
	return ""
}
