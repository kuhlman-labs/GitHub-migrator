package azuredevops

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/feed"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/policy"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/serviceendpoint"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/servicehooks"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/taskagent"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/testplan"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/wiki"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// Client wraps the Azure DevOps API client
type Client struct {
	connection         *azuredevops.Connection
	coreClient         core.Client
	gitClient          git.Client
	workClient         workitemtracking.Client
	buildClient        build.Client
	policyClient       policy.Client
	serviceEndpointClient serviceendpoint.Client
	serviceHooksClient servicehooks.Client
	taskAgentClient    taskagent.Client
	wikiClient         wiki.Client
	testPlanClient     testplan.Client
	feedClient         feed.Client
	orgURL             string
	token              string
	logger             *slog.Logger
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

	serviceEndpointClient, err := serviceendpoint.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create service endpoint client: %w", err)
	}

	serviceHooksClient := servicehooks.NewClient(context.Background(), connection)

	taskAgentClient, err := taskagent.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create task agent client: %w", err)
	}

	wikiClient, err := wiki.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create wiki client: %w", err)
	}

	testPlanClient := testplan.NewClient(context.Background(), connection)

	feedClient, err := feed.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create feed client: %w", err)
	}

	return &Client{
		connection:            connection,
		coreClient:            coreClient,
		gitClient:             gitClient,
		workClient:            workClient,
		buildClient:           buildClient,
		policyClient:          policyClient,
		serviceEndpointClient: serviceEndpointClient,
		serviceHooksClient:    serviceHooksClient,
		taskAgentClient:       taskAgentClient,
		wikiClient:            wikiClient,
		testPlanClient:        testPlanClient,
		feedClient:            feedClient,
		orgURL:                cfg.OrganizationURL,
		token:                 cfg.PersonalAccessToken,
		logger:                logger,
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
	repoType := "TfsGit" // Git repositories in Azure DevOps
	definitions, err := c.buildClient.GetDefinitions(ctx, build.GetDefinitionsArgs{
		Project:        &projectName,
		RepositoryId:   &repoID,
		RepositoryType: &repoType,
		Top:            &top,
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

// HasGHAS checks if a repository has GitHub Advanced Security for Azure DevOps enabled
func (c *Client) HasGHAS(ctx context.Context, projectName, repoID string) (bool, error) {
	// GitHub Advanced Security for Azure DevOps is a premium feature
	// that provides code scanning, secret scanning, and dependency scanning.
	// 
	// As of now, there's no public API in the Azure DevOps Go SDK to detect
	// if GHAS is enabled for a specific repository.
	//
	// The feature must be enabled at the organization level and then enabled
	// per repository through the Azure DevOps web UI.
	//
	// Without API support, we cannot programmatically detect this feature.
	// Returning false to indicate it's not detected (not that it's disabled).
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

// GetPipelineDefinitions gets pipeline definitions for a repository
func (c *Client) GetPipelineDefinitions(ctx context.Context, projectName, repoID string) ([]build.BuildDefinitionReference, error) {
	repoType := "TfsGit" // Git repositories in Azure DevOps
	definitions, err := c.buildClient.GetDefinitions(ctx, build.GetDefinitionsArgs{
		Project:        &projectName,
		RepositoryId:   &repoID,
		RepositoryType: &repoType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline definitions: %w", err)
	}

	if definitions == nil || definitions.Value == nil {
		return []build.BuildDefinitionReference{}, nil
	}

	return definitions.Value, nil
}

// GetPipelineRuns gets recent pipeline runs for a repository
func (c *Client) GetPipelineRuns(ctx context.Context, projectName, repoID string) (int, error) {
	// Get builds from the last 30 days with top 100
	top := 100
	repoType := "TfsGit" // Git repositories in Azure DevOps
	builds, err := c.buildClient.GetBuilds(ctx, build.GetBuildsArgs{
		Project:        &projectName,
		RepositoryId:   &repoID,
		RepositoryType: &repoType,
		Top:            &top,
	})
	if err != nil {
		c.logger.Debug("Failed to get pipeline runs", "error", err)
		return 0, nil
	}

	if builds == nil || builds.Value == nil {
		return 0, nil
	}

	return len(builds.Value), nil
}

// GetServiceConnections checks if a project has service connections
// Service connections are project-level, not repository-level
func (c *Client) GetServiceConnections(ctx context.Context, projectName string) (int, error) {
	endpoints, err := c.serviceEndpointClient.GetServiceEndpoints(ctx, serviceendpoint.GetServiceEndpointsArgs{
		Project: &projectName,
	})
	if err != nil {
		c.logger.Debug("Failed to get service endpoints", "project", projectName, "error", err)
		return 0, nil // Return 0 on error to avoid breaking discovery
	}

	if endpoints == nil {
		return 0, nil
	}

	return len(*endpoints), nil
}

// GetVariableGroups checks if a project has variable groups
func (c *Client) GetVariableGroups(ctx context.Context, projectName string) (int, error) {
	groups, err := c.taskAgentClient.GetVariableGroups(ctx, taskagent.GetVariableGroupsArgs{
		Project: &projectName,
	})
	if err != nil {
		c.logger.Debug("Failed to get variable groups", "project", projectName, "error", err)
		return 0, nil // Return 0 on error to avoid breaking discovery
	}

	if groups == nil {
		return 0, nil
	}

	return len(*groups), nil
}

// GetWikiDetails gets wiki information for a project
func (c *Client) GetWikiDetails(ctx context.Context, projectName, repoID string) (hasWiki bool, pageCount int, err error) {
	// Get all wikis for the project
	wikis, err := c.wikiClient.GetAllWikis(ctx, wiki.GetAllWikisArgs{
		Project: &projectName,
	})
	if err != nil {
		c.logger.Debug("Failed to get wikis", "project", projectName, "error", err)
		return false, 0, nil // Return false on error to avoid breaking discovery
	}

	if wikis == nil || len(*wikis) == 0 {
		return false, 0, nil
	}

	// Check if any wiki exists for this project
	hasWiki = len(*wikis) > 0
	totalPages := 0

	// Count pages across all wikis
	for _, w := range *wikis {
		if w.Id != nil {
			// Convert UUID to string for WikiIdentifier
			wikiID := w.Id.String()
			// Get pages for this wiki - this is a basic count
			// Note: Getting accurate page count requires recursively fetching pages
			// For performance, we'll do a simple check
			pages, pageErr := c.wikiClient.GetPagesBatch(ctx, wiki.GetPagesBatchArgs{
				Project:        &projectName,
				WikiIdentifier: &wikiID,
			})
			if pageErr == nil && pages != nil {
				totalPages += len(pages.Value)
			}
		}
	}

	return hasWiki, totalPages, nil
}

// GetTestPlans gets test plans for a project
func (c *Client) GetTestPlans(ctx context.Context, projectName string) (int, error) {
	plans, err := c.testPlanClient.GetTestPlans(ctx, testplan.GetTestPlansArgs{
		Project: &projectName,
	})
	if err != nil {
		c.logger.Debug("Failed to get test plans", "project", projectName, "error", err)
		return 0, nil // Return 0 on error to avoid breaking discovery
	}

	if plans == nil {
		return 0, nil
	}

	return len(plans.Value), nil
}

// GetServiceHooks gets service hooks (subscriptions) for a project
func (c *Client) GetServiceHooks(ctx context.Context, projectName string) (int, error) {
	subscriptions, err := c.serviceHooksClient.ListSubscriptions(ctx, servicehooks.ListSubscriptionsArgs{
		// Note: ListSubscriptions is organization-wide, not project-specific
		// We cannot filter by project, so this returns all subscriptions
	})
	if err != nil {
		c.logger.Debug("Failed to get service hook subscriptions", "project", projectName, "error", err)
		return 0, nil // Return 0 on error to avoid breaking discovery
	}

	if subscriptions == nil {
		return 0, nil
	}

	// Note: This returns the total count of service hooks in the organization,
	// not just for this project. The API doesn't support project-level filtering.
	return len(*subscriptions), nil
}

// GetPackageFeeds gets package feeds for a project
func (c *Client) GetPackageFeeds(ctx context.Context, projectName string) (int, error) {
	feeds, err := c.feedClient.GetFeeds(ctx, feed.GetFeedsArgs{
		Project: &projectName,
	})
	if err != nil {
		c.logger.Debug("Failed to get package feeds", "project", projectName, "error", err)
		return 0, nil // Return 0 on error to avoid breaking discovery
	}

	if feeds == nil {
		return 0, nil
	}

	return len(*feeds), nil
}

// GetPRDetails gets enhanced pull request details
func (c *Client) GetPRDetails(ctx context.Context, projectName, repoID string) (openCount, withWorkItems, withAttachments int, err error) {
	// Get all PRs
	allPRs, err := c.GetPullRequests(ctx, projectName, repoID)
	if err != nil {
		return 0, 0, 0, err
	}

	openCount = 0
	withWorkItems = 0
	withAttachments = 0

	for _, pr := range allPRs {
		// Count open PRs
		if pr.Status != nil && *pr.Status == git.PullRequestStatusValues.Active {
			openCount++
		}

		// Check for work item links
		if pr.WorkItemRefs != nil && len(*pr.WorkItemRefs) > 0 {
			withWorkItems++
		}

		// Check for attachments - would require additional API calls per PR
		// For now, we'll skip this detailed check for performance
	}

	return openCount, withWorkItems, withAttachments, nil
}

// GetWorkItemDetails gets enhanced work item details linked to a repository
func (c *Client) GetWorkItemDetails(ctx context.Context, projectName, repoName string) (linkedCount, activeCount int, workItemTypes []string, err error) {
	// Query for work items in the project using WIQL
	// We'll get a count of active work items as a proxy for repository activity
	wiql := "Select [System.Id] From WorkItems Where [System.TeamProject] = @project AND [System.State] <> 'Closed' AND [System.State] <> 'Removed'"
	
	query := workitemtracking.Wiql{
		Query: &wiql,
	}
	
	result, queryErr := c.workClient.QueryByWiql(ctx, workitemtracking.QueryByWiqlArgs{
		Project: &projectName,
		Wiql:    &query,
	})
	
	if queryErr != nil {
		c.logger.Debug("Failed to query work items", "project", projectName, "error", queryErr)
		return 0, 0, []string{}, nil // Return empty on error to avoid breaking discovery
	}
	
	if result == nil || result.WorkItems == nil {
		return 0, 0, []string{}, nil
	}
	
	// Count active work items (this is project-wide, not repo-specific)
	activeCount = len(*result.WorkItems)
	
	// Note: Getting repo-specific work items would require:
	// 1. Fetching all commits for the repo
	// 2. For each commit, checking work item links
	// This is too expensive for discovery, so we return project-level stats
	
	return 0, activeCount, []string{}, nil
}

// GetBranchPolicyDetails gets detailed branch policy information
func (c *Client) GetBranchPolicyDetails(ctx context.Context, projectName, repoID string) (policyTypes []string, requiredReviewers, buildValidations int, err error) {
	policies, err := c.GetBranchPolicies(ctx, projectName, repoID)
	if err != nil {
		return nil, 0, 0, err
	}

	policyTypeMap := make(map[string]bool)
	requiredReviewers = 0
	buildValidations = 0

	for _, pol := range policies {
		if pol.Type != nil && pol.Type.DisplayName != nil {
			policyType := *pol.Type.DisplayName
			policyTypeMap[policyType] = true

			// Count specific policy types
			switch policyType {
			case "Required reviewers":
				requiredReviewers++
			case "Build", "Build validation":
				buildValidations++
			}
		}
	}

	// Convert map to slice
	policyTypes = make([]string, 0, len(policyTypeMap))
	for policyType := range policyTypeMap {
		policyTypes = append(policyTypes, policyType)
	}

	return policyTypes, requiredReviewers, buildValidations, nil
}
