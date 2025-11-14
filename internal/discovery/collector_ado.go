package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

// ADOCollector wraps an ADO client for discovery operations
type ADOCollector struct {
	client   *azuredevops.Client
	storage  interface{} // Will be *storage.Database
	logger   *slog.Logger
	provider interface{} // Will be source.Provider
	profiler *ADOProfiler
}

// NewADOCollector creates a new Azure DevOps collector
func NewADOCollector(client *azuredevops.Client, storage interface{}, logger *slog.Logger, provider interface{}) *ADOCollector {
	return &ADOCollector{
		client:   client,
		storage:  storage,
		logger:   logger,
		provider: provider,
		profiler: NewADOProfiler(client, logger, provider),
	}
}

// DiscoverADOOrganization discovers all projects and repositories in an Azure DevOps organization
func (c *ADOCollector) DiscoverADOOrganization(ctx context.Context, organization string) error {
	c.logger.Info("Starting Azure DevOps organization discovery", "organization", organization)

	// Get all projects in the organization
	projects, err := c.client.GetProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}

	c.logger.Info("Found projects in organization",
		"organization", organization,
		"count", len(projects))

	// Process each project
	for _, project := range projects {
		if project.Name == nil {
			c.logger.Warn("Skipping project with nil name")
			continue
		}

		projectName := *project.Name
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
				// Continue with next project
				continue
			}
		}
		c.logger.Debug("Project saved", "project", projectName)

		// Discover repositories in this project, passing the project visibility
		projectVisibility := "private" // Default to private
		if project.Visibility != nil {
			projectVisibility = string(*project.Visibility)
		}

		if err := c.DiscoverADOProjectWithVisibility(ctx, organization, projectName, projectVisibility); err != nil {
			c.logger.Error("Failed to discover project",
				"project", projectName,
				"error", err)
			// Continue with next project
			continue
		}
	}

	c.logger.Info("Azure DevOps organization discovery complete",
		"organization", organization,
		"projects", len(projects))

	return nil
}

// DiscoverADOProject discovers all repositories in a specific Azure DevOps project
func (c *ADOCollector) DiscoverADOProject(ctx context.Context, organization, projectName string) error {
	// Get project details to fetch visibility
	project, err := c.client.GetProject(ctx, projectName)
	if err != nil {
		c.logger.Warn("Failed to get project details, using default visibility",
			"project", projectName,
			"error", err)
		return c.DiscoverADOProjectWithVisibility(ctx, organization, projectName, "private")
	}

	projectVisibility := "private" // Default to private
	if project.Visibility != nil {
		projectVisibility = string(*project.Visibility)
	}

	return c.DiscoverADOProjectWithVisibility(ctx, organization, projectName, projectVisibility)
}

// DiscoverADOProjectWithVisibility discovers all repositories in a specific Azure DevOps project with known visibility
func (c *ADOCollector) DiscoverADOProjectWithVisibility(ctx context.Context, organization, projectName, projectVisibility string) error {
	c.logger.Info("Starting Azure DevOps project discovery",
		"organization", organization,
		"project", projectName,
		"visibility", projectVisibility)

	// Get all repositories in the project
	repos, err := c.client.GetRepositories(ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	c.logger.Info("Found repositories in project",
		"project", projectName,
		"count", len(repos))

	// Process each repository
	for _, adoRepo := range repos {
		if adoRepo.Name == nil {
			c.logger.Warn("Skipping repository with nil name")
			continue
		}

		repoName := *adoRepo.Name
		c.logger.Debug("Processing repository",
			"project", projectName,
			"repo", repoName)

		// Create repository model
		fullName := fmt.Sprintf("%s/%s/%s", organization, projectName, repoName)
		now := time.Now()

		repo := &models.Repository{
			FullName:        fullName,
			Source:          "azuredevops",
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
		// This will fill in additional details like:
		// - Branch count, commit count
		// - Pull request count, work item count
		// - Azure Boards/Pipelines detection
		// - Branch policies, GHAS detection
		c.logger.Debug("Profiling repository",
			"project", projectName,
			"repo", repoName)

		if err := c.profiler.ProfileRepository(ctx, repo, adoRepo); err != nil {
			c.logger.Warn("Failed to profile repository",
				"project", projectName,
				"repo", repoName,
				"error", err)
			// Continue even if profiling fails - we have basic info
		}

		// Save repository to database
		if db, ok := c.storage.(*storage.Database); ok {
			if err := db.SaveRepository(ctx, repo); err != nil {
				c.logger.Error("Failed to save repository",
					"project", projectName,
					"repo", repoName,
					"error", err)
				// Continue with next repository
				continue
			}
		}
		c.logger.Debug("Repository saved",
			"project", projectName,
			"repo", repoName,
			"is_git", repo.ADOIsGit)
	}

	c.logger.Info("Azure DevOps project discovery complete",
		"project", projectName,
		"repositories", len(repos))

	return nil
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

	return nil
}

// getRemoteURL extracts the web URL from an ADO repository (for viewing in browser)
func getRemoteURL(repo interface{}) string {
	// Type assert to git.GitRepository from ADO SDK
	if gitRepo, ok := repo.(git.GitRepository); ok {
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
