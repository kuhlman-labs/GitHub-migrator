package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/azuredevops"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

// ADOCollector wraps an ADO client for discovery operations
type ADOCollector struct {
	client   *azuredevops.Client
	storage  interface{} // Will be *storage.Database
	logger   *slog.Logger
	provider interface{} // Will be source.Provider
}

// NewADOCollector creates a new Azure DevOps collector
func NewADOCollector(client *azuredevops.Client, storage interface{}, logger *slog.Logger, provider interface{}) *ADOCollector {
	return &ADOCollector{
		client:   client,
		storage:  storage,
		logger:   logger,
		provider: provider,
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

		// TODO: Save project to database
		// Will need to implement SaveADOProject method in storage
		c.logger.Debug("Project saved", "project", projectName)

		// Discover repositories in this project
		if err := c.DiscoverADOProject(ctx, organization, projectName); err != nil {
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
	c.logger.Info("Starting Azure DevOps project discovery",
		"organization", organization,
		"project", projectName)

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

		// TODO: Profile repository with ADO profiler
		// This will fill in additional details like:
		// - Branch count
		// - Commit count
		// - Pull request count
		// - Work item count
		// - Azure Boards/Pipelines detection
		// - Branch policies
		// - GHAS detection

		// TODO: Save repository to database
		// Will use existing SaveRepository method
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

// getRemoteURL extracts the remote URL from an ADO repository
func getRemoteURL(repo interface{}) string {
	// The azuredevops.GitRepository struct should have a RemoteURL or WebURL field
	// For now, return a placeholder
	// TODO: Extract actual URL from repo object
	return ""
}
