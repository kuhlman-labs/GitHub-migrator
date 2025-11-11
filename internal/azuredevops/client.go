package azuredevops

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/policy"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// Client wraps the Azure DevOps API client
type Client struct {
	connection   *azuredevops.Connection
	coreClient   core.Client
	gitClient    git.Client
	workClient   workitemtracking.Client
	buildClient  build.Client
	policyClient policy.Client
	orgURL       string
	token        string
	logger       *slog.Logger
}

// ClientConfig contains configuration for creating an ADO client
type ClientConfig struct {
	OrganizationURL     string
	PersonalAccessToken string
	Logger              *slog.Logger
}

// Validate checks if the configuration is valid
func (c ClientConfig) Validate() error {
	if c.OrganizationURL == "" {
		return fmt.Errorf("organization URL is required")
	}
	if c.PersonalAccessToken == "" {
		return fmt.Errorf("personal access token is required")
	}
	return nil
}

// NewClient creates a new Azure DevOps client
func NewClient(cfg ClientConfig) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Create connection
	connection := azuredevops.NewPatConnection(cfg.OrganizationURL, cfg.PersonalAccessToken)

	// Create service clients
	coreClient, err := core.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	gitClient, err := git.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create git client: %w", err)
	}

	workClient, err := workitemtracking.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work item tracking client: %w", err)
	}

	buildClient, err := build.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create build client: %w", err)
	}

	policyClient, err := policy.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy client: %w", err)
	}

	return &Client{
		connection:   connection,
		coreClient:   coreClient,
		gitClient:    gitClient,
		workClient:   workClient,
		buildClient:  buildClient,
		policyClient: policyClient,
		orgURL:       cfg.OrganizationURL,
		token:        cfg.PersonalAccessToken,
		logger:       logger,
	}, nil
}

// GetProjects returns all projects in the organization
func (c *Client) GetProjects(ctx context.Context) ([]core.TeamProjectReference, error) {
	// Get all projects with all states
	projects, err := c.coreClient.GetProjects(ctx, core.GetProjectsArgs{})
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	if projects == nil || projects.Value == nil {
		return []core.TeamProjectReference{}, nil
	}

	return projects.Value, nil
}

// GetProject returns a specific project by name
func (c *Client) GetProject(ctx context.Context, projectName string) (*core.TeamProject, error) {
	includeCapabilities := true
	project, err := c.coreClient.GetProject(ctx, core.GetProjectArgs{
		ProjectId:           &projectName,
		IncludeCapabilities: &includeCapabilities,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// GetRepositories returns all Git repositories in a project
func (c *Client) GetRepositories(ctx context.Context, projectName string) ([]git.GitRepository, error) {
	repos, err := c.gitClient.GetRepositories(ctx, git.GetRepositoriesArgs{
		Project: &projectName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	if repos == nil {
		return []git.GitRepository{}, nil
	}

	return *repos, nil
}

// GetRepository returns a specific repository
func (c *Client) GetRepository(ctx context.Context, projectName, repoName string) (*git.GitRepository, error) {
	repo, err := c.gitClient.GetRepository(ctx, git.GetRepositoryArgs{
		RepositoryId: &repoName,
		Project:      &projectName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// GetBranches returns all branches in a repository
func (c *Client) GetBranches(ctx context.Context, projectName, repoName string) ([]git.GitBranchStats, error) {
	branches, err := c.gitClient.GetBranches(ctx, git.GetBranchesArgs{
		RepositoryId: &repoName,
		Project:      &projectName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	if branches == nil {
		return []git.GitBranchStats{}, nil
	}

	return *branches, nil
}

// GetPullRequests returns all pull requests in a repository
func (c *Client) GetPullRequests(ctx context.Context, projectName, repoID string) ([]git.GitPullRequest, error) {
	searchCriteria := git.GitPullRequestSearchCriteria{
		Status: &git.PullRequestStatusValues.All,
	}

	prs, err := c.gitClient.GetPullRequests(ctx, git.GetPullRequestsArgs{
		RepositoryId:   &repoID,
		Project:        &projectName,
		SearchCriteria: &searchCriteria,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}

	if prs == nil {
		return []git.GitPullRequest{}, nil
	}

	return *prs, nil
}

// GetCommitCount returns the number of commits in a repository
func (c *Client) GetCommitCount(ctx context.Context, projectName, repoID string) (int, error) {
	// Use search criteria to get all commits with top=1 to check if there are commits
	top := 1
	searchCriteria := git.GitQueryCommitsCriteria{
		Top: &top,
	}

	commits, err := c.gitClient.GetCommits(ctx, git.GetCommitsArgs{
		RepositoryId:   &repoID,
		Project:        &projectName,
		SearchCriteria: &searchCriteria,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get commits: %w", err)
	}

	if commits == nil {
		return 0, nil
	}

	// Note: ADO API doesn't provide total count directly for commits
	// We'll need to use pagination or estimate based on default branch history
	// For now, return a conservative estimate
	return len(*commits), nil
}

// IsGitRepo checks if a repository is a Git repository (vs TFVC)
func (c *Client) IsGitRepo(ctx context.Context, projectName, repoName string) (bool, error) {
	repo, err := c.GetRepository(ctx, projectName, repoName)
	if err != nil {
		return false, err
	}

	// In Azure DevOps, Git repositories have an ID
	// TFVC repositories would be accessed differently
	return repo.Id != nil, nil
}

// HasAzureBoards checks if a project has Azure Boards enabled
func (c *Client) HasAzureBoards(ctx context.Context, projectName string) (bool, error) {
	project, err := c.GetProject(ctx, projectName)
	if err != nil {
		return false, err
	}

	// Check if work item tracking is enabled in project capabilities
	if project.Capabilities != nil {
		if processTemplate, ok := (*project.Capabilities)["processTemplate"]; ok {
			// If process template exists, Azure Boards is likely enabled
			return processTemplate != nil, nil
		}
	}

	return false, nil
}

// HasAzurePipelines checks if a repository has Azure Pipelines configured
func (c *Client) HasAzurePipelines(ctx context.Context, projectName, repoID string) (bool, error) {
	// Get build definitions for this repository
	top := 1
	definitions, err := c.buildClient.GetDefinitions(ctx, build.GetDefinitionsArgs{
		Project:      &projectName,
		RepositoryId: &repoID,
		Top:          &top,
	})
	if err != nil {
		c.logger.Debug("Failed to get build definitions", "error", err)
		return false, nil
	}

	return definitions != nil && len(definitions.Value) > 0, nil
}

// GetBranchPolicies returns branch policies for a repository
func (c *Client) GetBranchPolicies(ctx context.Context, projectName, repoID string) ([]policy.PolicyConfiguration, error) {
	policies, err := c.policyClient.GetPolicyConfigurations(ctx, policy.GetPolicyConfigurationsArgs{
		Project: &projectName,
		Scope:   &repoID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get branch policies: %w", err)
	}

	if policies == nil || policies.Value == nil {
		return []policy.PolicyConfiguration{}, nil
	}

	return policies.Value, nil
}

// GetWorkItemsLinkedToRepo gets work items linked to a repository
// This is a simplified version - full implementation would require more complex queries
func (c *Client) GetWorkItemsLinkedToRepo(ctx context.Context, projectName, repoName string) (int, error) {
	// Note: Getting work items linked to a specific repo requires
	// querying work items and checking their commit links
	// This is complex and may require multiple API calls
	// For now, we'll return 0 as a placeholder
	c.logger.Debug("GetWorkItemsLinkedToRepo not fully implemented",
		"project", projectName,
		"repo", repoName)
	return 0, nil
}

// HasGHAS checks if a repository has GitHub Advanced Security enabled
func (c *Client) HasGHAS(ctx context.Context, projectName, repoID string) (bool, error) {
	// GitHub Advanced Security for Azure DevOps is a separate feature
	// It requires additional API calls to check if enabled
	// For now, we'll return false as a placeholder
	c.logger.Debug("GHAS detection not fully implemented",
		"project", projectName,
		"repo", repoID)
	return false, nil
}

// ValidateCredentials validates the PAT by attempting to list projects
func (c *Client) ValidateCredentials(ctx context.Context) error {
	_, err := c.GetProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}
	return nil
}
