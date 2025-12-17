package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Collector discovers and profiles repositories
type Collector struct {
	client         *github.Client
	storage        *storage.Database
	logger         *slog.Logger
	workers        int // Number of parallel workers
	sourceProvider source.Provider
	baseConfig     *github.ClientConfig // Base config for creating per-org clients (optional)
}

// NewCollector creates a new repository collector
func NewCollector(client *github.Client, storage *storage.Database, logger *slog.Logger, sourceProvider source.Provider) *Collector {
	return &Collector{
		client:         client,
		storage:        storage,
		logger:         logger,
		workers:        5, // Default to 5 parallel workers
		sourceProvider: sourceProvider,
	}
}

// WithBaseConfig sets the base configuration for creating per-org clients
// This enables automatic per-org client creation for GitHub Enterprise Apps
func (c *Collector) WithBaseConfig(config github.ClientConfig) *Collector {
	c.baseConfig = &config
	return c
}

// SetWorkers sets the number of parallel workers for processing
func (c *Collector) SetWorkers(workers int) {
	if workers > 0 {
		c.workers = workers
	}
}

// DiscoverRepositories discovers all repositories from the source organization
func (c *Collector) DiscoverRepositories(ctx context.Context, org string) error {
	c.logger.Info("Starting repository discovery", "organization", org)

	// Check if we need to create an org-specific client (JWT-only mode)
	var orgClient *github.Client
	var profiler *Profiler

	if c.baseConfig != nil && c.baseConfig.AppID > 0 && c.baseConfig.AppInstallationID == 0 {
		// JWT-only mode: create org-specific client
		c.logger.Info("Creating org-specific client for single-org discovery",
			"org", org,
			"app_id", c.baseConfig.AppID)

		// Get installation ID for this org
		installationID, err := c.client.GetOrganizationInstallationID(ctx, org)
		if err != nil {
			return fmt.Errorf("failed to get installation ID for org %s: %w", org, err)
		}

		// Create org-specific client
		orgConfig := *c.baseConfig
		orgConfig.AppInstallationID = installationID

		orgClient, err = github.NewClient(orgConfig)
		if err != nil {
			return fmt.Errorf("failed to create org-specific client for %s: %w", org, err)
		}

		c.logger.Debug("Created org-specific client",
			"org", org,
			"installation_id", installationID)
	} else {
		// Use the default client (PAT or App with installation ID)
		orgClient = c.client
	}

	// List all repositories using the appropriate client
	repos, err := c.listAllRepositoriesWithClient(ctx, org, orgClient)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	c.logger.Info("Found repositories", "count", len(repos))

	// Create profiler with the appropriate client and load package cache
	profiler = NewProfiler(orgClient, c.logger)
	if err := profiler.LoadPackageCache(ctx, org); err != nil {
		c.logger.Warn("Failed to load package cache, package detection may be slower",
			"org", org,
			"error", err)
	}

	// Load ProjectsV2 map for this organization
	if err := profiler.LoadProjectsMap(ctx, org); err != nil {
		c.logger.Warn("Failed to load ProjectsV2 map, will use fallback detection",
			"org", org,
			"error", err)
	}

	// Load GitHub App installations for this organization
	if err := profiler.LoadOrgInstallations(ctx, org); err != nil {
		c.logger.Warn("Failed to load org installations, app detection may be limited",
			"org", org,
			"error", err)
	}

	// Process repositories in parallel
	if err := c.processRepositoriesWithProfiler(ctx, repos, profiler); err != nil {
		return err
	}

	// After discovery completes, update local dependency flags
	c.logger.Info("Updating local dependency flags", "organization", org)
	if err := c.storage.UpdateLocalDependencyFlags(ctx); err != nil {
		c.logger.Warn("Failed to update local dependency flags", "error", err)
		// Don't fail the whole discovery if this fails
	}

	// Discover teams and their repository associations
	c.logger.Info("Discovering teams", "organization", org)
	if err := c.discoverTeams(ctx, org, orgClient); err != nil {
		c.logger.Warn("Failed to discover teams", "organization", org, "error", err)
		// Don't fail the whole discovery if team discovery fails
	}

	// Discover organization members
	c.logger.Info("Discovering organization members", "organization", org)
	if err := c.discoverOrgMembers(ctx, org, orgClient); err != nil {
		c.logger.Warn("Failed to discover org members", "organization", org, "error", err)
		// Don't fail the whole discovery if member discovery fails
	}

	return nil
}

// DiscoverOrgMembersOnly discovers only organization members without repository discovery
// This is used for standalone user discovery from the Users page
func (c *Collector) DiscoverOrgMembersOnly(ctx context.Context, org string) (int, error) {
	c.logger.Info("Starting org members-only discovery", "organization", org)

	// Check if we need to create an org-specific client (JWT-only mode)
	var orgClient *github.Client
	if c.baseConfig != nil && c.baseConfig.AppID > 0 && c.baseConfig.AppInstallationID == 0 {
		// Get installation ID for this org
		installationID, err := c.client.GetOrganizationInstallationID(ctx, org)
		if err != nil {
			return 0, fmt.Errorf("failed to get installation for org %s: %w", org, err)
		}
		// Create org-specific client
		orgConfig := *c.baseConfig
		orgConfig.AppInstallationID = installationID
		orgClient, err = github.NewClient(orgConfig)
		if err != nil {
			return 0, fmt.Errorf("failed to create org client for %s: %w", org, err)
		}
	} else {
		orgClient = c.client
	}

	members, err := orgClient.ListOrgMembers(ctx, org)
	if err != nil {
		return 0, fmt.Errorf("failed to list org members: %w", err)
	}

	c.logger.Info("Found organization members", "organization", org, "count", len(members))

	sourceInstance := c.getSourceInstance()
	savedCount := 0

	for _, member := range members {
		user := &models.GitHubUser{
			Login:          member.Login,
			Name:           member.Name,
			Email:          member.Email,
			SourceInstance: sourceInstance,
		}
		if member.AvatarURL != "" {
			avatarURL := member.AvatarURL
			user.AvatarURL = &avatarURL
		}

		if err := c.storage.SaveUser(ctx, user); err != nil {
			c.logger.Warn("Failed to save organization member",
				"organization", org,
				"login", member.Login,
				"error", err)
			continue
		}
		savedCount++

		// Save org membership
		membership := &models.UserOrgMembership{
			UserLogin:    member.Login,
			Organization: org,
			Role:         member.Role,
		}
		if err := c.storage.SaveUserOrgMembership(ctx, membership); err != nil {
			c.logger.Warn("Failed to save org membership",
				"organization", org,
				"login", member.Login,
				"error", err)
		}
	}

	c.logger.Info("Org members-only discovery complete",
		"organization", org,
		"total_members", len(members),
		"users_saved", savedCount)

	return savedCount, nil
}

// teamDiscoveryResult holds the result of processing a single team
type teamDiscoveryResult struct {
	teamSaved   bool
	memberCount int
}

// DiscoverTeamsOnly discovers only teams and their members without repository discovery
// This is used for standalone team discovery from the Teams page
// Uses parallel processing with worker pool for improved performance
func (c *Collector) DiscoverTeamsOnly(ctx context.Context, org string) (int, int, error) {
	c.logger.Info("Starting teams-only discovery", "organization", org)

	// Check if we need to create an org-specific client (JWT-only mode)
	var orgClient *github.Client
	if c.baseConfig != nil && c.baseConfig.AppID > 0 && c.baseConfig.AppInstallationID == 0 {
		// Get installation ID for this org
		installationID, err := c.client.GetOrganizationInstallationID(ctx, org)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get installation for org %s: %w", org, err)
		}
		// Create org-specific client
		orgConfig := *c.baseConfig
		orgConfig.AppInstallationID = installationID
		orgClient, err = github.NewClient(orgConfig)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to create org client for %s: %w", org, err)
		}
	} else {
		orgClient = c.client
	}

	teams, err := orgClient.ListOrganizationTeams(ctx, org)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list teams: %w", err)
	}

	c.logger.Info("Found teams", "organization", org, "count", len(teams), "workers", c.workers)

	if len(teams) == 0 {
		return 0, 0, nil
	}

	// Process teams in parallel using worker pool
	jobs := make(chan *github.TeamInfo, len(teams))
	results := make(chan teamDiscoveryResult, len(teams))
	var wg sync.WaitGroup

	sourceInstance := c.getSourceInstance()

	// Start workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.teamsOnlyWorker(ctx, &wg, i, org, orgClient, sourceInstance, jobs, results)
	}

	// Send jobs
	for _, team := range teams {
		jobs <- team
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(results)

	// Collect results
	teamCount := 0
	memberCount := 0
	for result := range results {
		if result.teamSaved {
			teamCount++
		}
		memberCount += result.memberCount
	}

	c.logger.Info("Teams-only discovery complete",
		"organization", org,
		"teams_saved", teamCount,
		"members_saved", memberCount)

	return teamCount, memberCount, nil
}

// teamsOnlyWorker processes teams from the jobs channel for DiscoverTeamsOnly
func (c *Collector) teamsOnlyWorker(ctx context.Context, wg *sync.WaitGroup, workerID int, org string, client *github.Client, sourceInstance string, jobs <-chan *github.TeamInfo, results chan<- teamDiscoveryResult) {
	defer wg.Done()

	for teamInfo := range jobs {
		result := teamDiscoveryResult{}

		c.logger.Debug("Worker processing team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug)

		team := &models.GitHubTeam{
			Organization: org,
			Slug:         teamInfo.Slug,
			Name:         teamInfo.Name,
			Privacy:      teamInfo.Privacy,
		}
		if teamInfo.Description != "" {
			team.Description = stringPtr(teamInfo.Description)
		}

		if err := c.storage.SaveTeam(ctx, team); err != nil {
			c.logger.Warn("Failed to save team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"error", err)
			results <- result
			continue
		}
		result.teamSaved = true

		// List and save team members using GraphQL (more efficient, no N+1 queries)
		teamMembers, err := client.ListTeamMembersGraphQL(ctx, org, teamInfo.Slug)
		if err != nil {
			c.logger.Warn("Failed to list members for team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"error", err)
			results <- result
			continue
		}

		c.logger.Debug("Found members for team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug,
			"count", len(teamMembers))

		for _, member := range teamMembers {
			// Save team member relationship
			teamMember := &models.GitHubTeamMember{
				TeamID: team.ID,
				Login:  member.Login,
				Role:   member.Role,
			}
			if err := c.storage.SaveTeamMember(ctx, teamMember); err != nil {
				c.logger.Warn("Failed to save team member",
					"worker_id", workerID,
					"organization", org,
					"team", teamInfo.Slug,
					"member", member.Login,
					"error", err)
				continue
			}
			result.memberCount++

			// Also save the user to github_users table
			user := &models.GitHubUser{
				Login:          member.Login,
				SourceInstance: sourceInstance,
			}
			if err := c.storage.SaveUser(ctx, user); err != nil {
				c.logger.Debug("User may already exist", "login", member.Login, "error", err)
			}
		}

		c.logger.Debug("Worker completed team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug,
			"members_saved", result.memberCount)

		results <- result
	}
}

// DiscoverEnterpriseRepositories discovers all repositories across all organizations in an enterprise
// For GitHub Enterprise Apps without an installation ID, this will:
//  1. Use JWT auth to list all app installations (discovers orgs automatically)
//  2. Create per-org clients with org-specific installation tokens
//  3. Use org-specific clients for all repo operations (higher rate limits, proper isolation)
//
//nolint:gocyclo // Complexity justified for handling multiple discovery modes and error cases
func (c *Collector) DiscoverEnterpriseRepositories(ctx context.Context, enterpriseSlug string) error {
	c.logger.Info("Starting enterprise-wide repository discovery", "enterprise", enterpriseSlug)

	// Check if we need to use per-org clients (GitHub App without installation ID)
	useAppInstallations := c.baseConfig != nil && c.baseConfig.AppID > 0 && c.baseConfig.AppInstallationID == 0

	var orgs []string
	var orgInstallations map[string]int64

	if useAppInstallations {
		// Use GitHub App Installations API to discover all organizations
		// This is the proper way for GitHub Apps to find their installations
		c.logger.Info("Using GitHub App Installations API to discover organizations",
			"app_id", c.baseConfig.AppID)

		installations, err := c.client.ListAppInstallations(ctx)
		if err != nil {
			return fmt.Errorf("failed to list app installations: %w", err)
		}

		orgInstallations = installations
		orgs = make([]string, 0, len(installations))
		for org := range installations {
			orgs = append(orgs, org)
		}

		c.logger.Info("Discovered organizations via app installations",
			"org_count", len(orgs))
	} else {
		// Use enterprise GraphQL API (requires installation token or PAT with enterprise access)
		var err error
		orgs, err = c.client.ListEnterpriseOrganizations(ctx, enterpriseSlug)
		if err != nil {
			return fmt.Errorf("failed to list enterprise organizations: %w", err)
		}

		c.logger.Info("Found organizations in enterprise",
			"enterprise", enterpriseSlug,
			"org_count", len(orgs))
	}

	// Collect repositories from all organizations
	var allRepos []*ghapi.Repository
	var profiler *Profiler

	// If not using per-org clients, create a single shared profiler
	if !useAppInstallations {
		profiler = NewProfiler(c.client, c.logger)
	}

	if useAppInstallations {
		// Process organizations in parallel when using per-org clients
		// Each org has its own isolated client and installation token
		c.logger.Info("Processing organizations in parallel",
			"org_count", len(orgs),
			"workers", c.workers)

		allRepos = c.processOrganizationsInParallel(ctx, enterpriseSlug, orgs, orgInstallations)
	} else {
		// Sequential processing for PAT/shared client mode
		for _, org := range orgs {
			c.logger.Info("Discovering repositories for organization",
				"enterprise", enterpriseSlug,
				"organization", org)

			// Use shared client and profiler
			orgClient := c.client
			orgProfiler := profiler

			// Use org-specific client for listing repos
			repos, err := c.listAllRepositoriesWithClient(ctx, org, orgClient)
			if err != nil {
				c.logger.Error("Failed to list repositories for organization",
					"enterprise", enterpriseSlug,
					"organization", org,
					"error", err)
				// Continue with other organizations even if one fails
				continue
			}

			c.logger.Info("Found repositories in organization",
				"enterprise", enterpriseSlug,
				"organization", org,
				"count", len(repos))

			// Load package cache for this organization using org-specific profiler
			if err := orgProfiler.LoadPackageCache(ctx, org); err != nil {
				c.logger.Warn("Failed to load package cache for organization",
					"enterprise", enterpriseSlug,
					"org", org,
					"error", err)
			}

			// Load ProjectsV2 map for this organization
			if err := orgProfiler.LoadProjectsMap(ctx, org); err != nil {
				c.logger.Warn("Failed to load ProjectsV2 map for organization",
					"enterprise", enterpriseSlug,
					"org", org,
					"error", err)
			}

			// Load GitHub App installations for this organization
			if err := orgProfiler.LoadOrgInstallations(ctx, org); err != nil {
				c.logger.Warn("Failed to load org installations for organization",
					"enterprise", enterpriseSlug,
					"org", org,
					"error", err)
			}

			allRepos = append(allRepos, repos...)

			// Discover teams and their repository associations for this org
			c.logger.Info("Discovering teams for organization",
				"enterprise", enterpriseSlug,
				"organization", org)
			if err := c.discoverTeams(ctx, org, orgClient); err != nil {
				c.logger.Warn("Failed to discover teams for organization",
					"enterprise", enterpriseSlug,
					"organization", org,
					"error", err)
				// Don't fail if team discovery fails
			}

			// Discover organization members
			c.logger.Info("Discovering organization members",
				"enterprise", enterpriseSlug,
				"organization", org)
			if err := c.discoverOrgMembers(ctx, org, orgClient); err != nil {
				c.logger.Warn("Failed to discover org members for organization",
					"enterprise", enterpriseSlug,
					"organization", org,
					"error", err)
				// Don't fail if member discovery fails
			}
		}
	}

	c.logger.Info("Enterprise discovery complete",
		"enterprise", enterpriseSlug,
		"total_orgs", len(orgs),
		"total_repos", len(allRepos))

	// If not using per-org clients, process all repositories in parallel using the shared profiler
	if !useAppInstallations {
		if err := c.processRepositoriesWithProfiler(ctx, allRepos, profiler); err != nil {
			return err
		}
	}

	// After discovery completes, update local dependency flags
	c.logger.Info("Updating local dependency flags")
	if err := c.storage.UpdateLocalDependencyFlags(ctx); err != nil {
		c.logger.Warn("Failed to update local dependency flags", "error", err)
		// Don't fail the whole discovery if this fails
	}

	return nil
}

// processOrganizationsInParallel processes multiple organizations concurrently
// Each org gets its own client with its own installation token for complete isolation
func (c *Collector) processOrganizationsInParallel(ctx context.Context, enterpriseSlug string, orgs []string, orgInstallations map[string]int64) []*ghapi.Repository {
	// Create channels for work distribution
	orgJobs := make(chan string, len(orgs))
	allRepos := make([]*ghapi.Repository, 0)
	var reposMu sync.Mutex // Protect allRepos slice
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for org := range orgJobs {
				c.logger.Info("Worker processing organization",
					"worker_id", workerID,
					"enterprise", enterpriseSlug,
					"organization", org)

				// Create org-specific client with installation token
				installationID := orgInstallations[org]
				orgConfig := *c.baseConfig
				orgConfig.AppInstallationID = installationID

				orgClient, err := github.NewClient(orgConfig)
				if err != nil {
					c.logger.Error("Failed to create org-specific client",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"organization", org,
						"installation_id", installationID,
						"error", err)
					continue
				}

				c.logger.Debug("Created org-specific client",
					"worker_id", workerID,
					"enterprise", enterpriseSlug,
					"organization", org,
					"installation_id", installationID)

				// Create org-specific profiler
				orgProfiler := NewProfiler(orgClient, c.logger)

				// List repositories for this org
				repos, err := c.listAllRepositoriesWithClient(ctx, org, orgClient)
				if err != nil {
					c.logger.Error("Failed to list repositories for organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"organization", org,
						"error", err)
					continue
				}

				c.logger.Info("Found repositories in organization",
					"worker_id", workerID,
					"enterprise", enterpriseSlug,
					"organization", org,
					"count", len(repos))

				// Load package cache for this organization
				if err := orgProfiler.LoadPackageCache(ctx, org); err != nil {
					c.logger.Warn("Failed to load package cache for organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"org", org,
						"error", err)
				}

				// Load ProjectsV2 map for this organization
				if err := orgProfiler.LoadProjectsMap(ctx, org); err != nil {
					c.logger.Warn("Failed to load ProjectsV2 map for organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"org", org,
						"error", err)
				}

				// Load GitHub App installations for this organization
				if err := orgProfiler.LoadOrgInstallations(ctx, org); err != nil {
					c.logger.Warn("Failed to load org installations for organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"org", org,
						"error", err)
				}

				// Add repos to global list (thread-safe)
				reposMu.Lock()
				allRepos = append(allRepos, repos...)
				reposMu.Unlock()

				// Process this org's repos with its profiler
				if err := c.processRepositoriesWithProfiler(ctx, repos, orgProfiler); err != nil {
					c.logger.Error("Failed to process repositories for organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"organization", org,
						"error", err)
				} else {
					c.logger.Info("Completed processing organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"organization", org,
						"repo_count", len(repos))
				}

				// Discover teams and their repository associations for this org
				if err := c.discoverTeams(ctx, org, orgClient); err != nil {
					c.logger.Warn("Failed to discover teams for organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"organization", org,
						"error", err)
					// Don't fail if team discovery fails
				}

				// Discover organization members
				if err := c.discoverOrgMembers(ctx, org, orgClient); err != nil {
					c.logger.Warn("Failed to discover org members for organization",
						"worker_id", workerID,
						"enterprise", enterpriseSlug,
						"organization", org,
						"error", err)
					// Don't fail if member discovery fails
				}
			}
		}(i)
	}

	// Queue all organizations
	for _, org := range orgs {
		orgJobs <- org
	}
	close(orgJobs)

	// Wait for all workers to complete
	wg.Wait()

	return allRepos
}

// listAllRepositoriesWithClient lists all repositories for an organization using a specific client
func (c *Collector) listAllRepositoriesWithClient(ctx context.Context, org string, client *github.Client) ([]*ghapi.Repository, error) {
	var allRepos []*ghapi.Repository
	opts := &ghapi.RepositoryListByOrgOptions{
		ListOptions: ghapi.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.REST().Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

// listAllRepositories lists all repositories for an organization with pagination
// Uses the collector's default client
// nolint:unused // Convenience method for testing and future use
func (c *Collector) listAllRepositories(ctx context.Context, org string) ([]*ghapi.Repository, error) {
	return c.listAllRepositoriesWithClient(ctx, org, c.client)
}

// processRepositoriesWithProfiler processes repositories in parallel using worker pool with a shared profiler
func (c *Collector) processRepositoriesWithProfiler(ctx context.Context, repos []*ghapi.Repository, profiler *Profiler) error {
	jobs := make(chan *ghapi.Repository, len(repos))
	errors := make(chan error, len(repos))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.workerWithProfiler(ctx, &wg, jobs, errors, profiler)
	}

	// Send jobs
	for _, repo := range repos {
		jobs <- repo
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
		c.logger.Warn("Discovery completed with errors",
			"total_repos", len(repos),
			"error_count", len(errs))
		return fmt.Errorf("encountered %d errors during discovery (see logs for details)", len(errs))
	}

	c.logger.Info("Discovery completed successfully", "total_repos", len(repos))
	return nil
}

// workerWithProfiler processes repositories from the jobs channel with an optional shared profiler
func (c *Collector) workerWithProfiler(ctx context.Context, wg *sync.WaitGroup, jobs <-chan *ghapi.Repository, errors chan<- error, profiler *Profiler) {
	defer wg.Done()

	for repo := range jobs {
		if err := c.ProfileRepositoryWithProfiler(ctx, repo, profiler); err != nil {
			c.logger.Error("Failed to profile repository",
				"repo", repo.GetFullName(),
				"error", err)
			errors <- err
		}
	}
}

// ProfileDestinationRepository profiles a destination repository using API-only metrics (no cloning)
// This is used for post-migration validation to compare with source repository
func (c *Collector) ProfileDestinationRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	c.logger.Debug("Profiling destination repository (API-only)", "repo", fullName)

	// Parse full name
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository full name: %s", fullName)
	}
	org := parts[0]
	name := parts[1]

	// Get repository details from destination
	ghRepo, _, err := c.client.REST().Repositories.Get(ctx, org, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination repository: %w", err)
	}

	// Create basic repository profile from GitHub API data
	totalSize := int64(ghRepo.GetSize()) * 1024 // Convert KB to bytes
	defaultBranch := ghRepo.GetDefaultBranch()
	repo := &models.Repository{
		FullName:      ghRepo.GetFullName(),
		Source:        "ghec", // Destination is GHEC
		SourceURL:     ghRepo.GetHTMLURL(),
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		IsArchived:    ghRepo.GetArchived(),
		IsFork:        ghRepo.GetFork(),
		HasWiki:       ghRepo.GetHasWiki(),
		HasPages:      ghRepo.GetHasPages(),
		HasPackages:   false, // Will be detected by profiler
		Visibility:    ghRepo.GetVisibility(),
		Status:        string(models.StatusComplete),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Extract last push date
	if ghRepo.PushedAt != nil {
		pushTime := ghRepo.PushedAt.Time
		repo.LastCommitDate = &pushTime
	}

	// Get branch count using git API
	branches, _, err := c.client.REST().Repositories.ListBranches(ctx, org, name, nil)
	if err == nil {
		repo.BranchCount = len(branches)
	}

	// Get last commit SHA from default branch
	if defaultBranch != "" {
		branch, _, err := c.client.REST().Repositories.GetBranch(ctx, org, name, defaultBranch, 0)
		if err == nil && branch != nil && branch.Commit != nil {
			sha := branch.Commit.GetSHA()
			repo.LastCommitSHA = &sha
		}
	}

	// Get commit count (approximation from contributors API)
	contributors, _, err := c.client.REST().Repositories.ListContributors(ctx, org, name, nil)
	if err == nil {
		totalCommits := 0
		for _, contributor := range contributors {
			totalCommits += contributor.GetContributions()
		}
		repo.CommitCount = totalCommits
	}

	// Profile GitHub features via API (no clone needed)
	// Note: Don't save users for destination profiling, only source discovery
	profiler := NewProfiler(c.client, c.logger)
	if err := profiler.ProfileFeatures(ctx, repo); err != nil {
		c.logger.Warn("Failed to profile destination features",
			"repo", repo.FullName,
			"error", err)
	}

	// Log the profiled repository with dereferenced values
	destSizeBytes := int64(0)
	if repo.TotalSize != nil {
		destSizeBytes = *repo.TotalSize
	}
	c.logger.Info("Destination repository profiled",
		"repo", repo.FullName,
		"size_bytes", destSizeBytes,
		"commits", repo.CommitCount,
		"branches", repo.BranchCount,
		"tags", repo.TagCount)

	return repo, nil
}

// ProfileRepository profiles a single repository with both Git and GitHub features
func (c *Collector) ProfileRepository(ctx context.Context, ghRepo *ghapi.Repository) error {
	return c.ProfileRepositoryWithProfiler(ctx, ghRepo, nil)
}

// ProfileRepositoryWithProfiler profiles a single repository with both Git and GitHub features
// using an optional shared profiler (with package cache)
// nolint:gocyclo // Repository profiling involves many sequential checks
func (c *Collector) ProfileRepositoryWithProfiler(ctx context.Context, ghRepo *ghapi.Repository, profiler *Profiler) error {
	c.logger.Debug("Profiling repository", "repo", ghRepo.GetFullName())

	// Create basic repository profile from GitHub API data
	totalSize := int64(ghRepo.GetSize()) * 1024 // Convert KB to bytes (GitHub API returns KB)
	defaultBranch := ghRepo.GetDefaultBranch()
	now := time.Now()
	repo := &models.Repository{
		FullName:        ghRepo.GetFullName(),
		Source:          "ghes",
		SourceURL:       ghRepo.GetHTMLURL(),
		TotalSize:       &totalSize,
		DefaultBranch:   &defaultBranch,
		IsArchived:      ghRepo.GetArchived(),
		IsFork:          ghRepo.GetFork(),
		HasWiki:         ghRepo.GetHasWiki(),
		HasPages:        ghRepo.GetHasPages(),
		HasPackages:     false, // Will be detected by profiler
		Visibility:      ghRepo.GetVisibility(),
		Status:          string(models.StatusPending),
		DiscoveredAt:    now,
		UpdatedAt:       now,
		LastDiscoveryAt: &now, // Track when repository data was last refreshed
	}

	// Extract last push date from repository object
	if ghRepo.PushedAt != nil {
		pushTime := ghRepo.PushedAt.Time
		repo.LastCommitDate = &pushTime
	}

	// Clone repository temporarily for git-sizer analysis
	cloneUrl := ghRepo.GetCloneURL()
	tempDir, err := c.cloneRepositoryWithProvider(ctx, cloneUrl, repo.FullName)

	if err != nil {
		c.logger.Warn("Failed to clone repository for analysis, using API-only metrics",
			"repo", repo.FullName,
			"error", err)
		// Continue with basic profiling even if clone fails
	} else {
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				c.logger.Warn("Failed to clean up temp directory",
					"path", tempDir,
					"error", err)
			}
		}()

		// Analyze Git properties with git-sizer
		analyzer := NewAnalyzer(c.logger)
		if err := analyzer.AnalyzeGitProperties(ctx, repo, tempDir); err != nil {
			c.logger.Warn("Failed to analyze git properties",
				"repo", repo.FullName,
				"error", err)
		}
	}

	// Profile GitHub features via API (no clone needed)
	// Use shared profiler if provided, otherwise create a new one
	if profiler == nil {
		// Check if we need to create an org-specific client (JWT-only mode)
		var profilerClient *github.Client
		if c.baseConfig != nil && c.baseConfig.AppID > 0 && c.baseConfig.AppInstallationID == 0 {
			// JWT-only mode: create org-specific client for this repo
			parts := strings.Split(repo.FullName, "/")
			if len(parts) != 2 {
				c.logger.Error("Invalid repository full name",
					"repo", repo.FullName)
			} else {
				org := parts[0]
				c.logger.Debug("Creating org-specific client for single-repo profiling",
					"org", org,
					"repo", repo.FullName)

				// Get installation ID for this org
				installationID, err := c.client.GetOrganizationInstallationID(ctx, org)
				if err != nil {
					c.logger.Error("Failed to get installation ID for org",
						"org", org,
						"error", err)
					// Fall back to default client (will likely fail but better than nothing)
					profilerClient = c.client
				} else {
					// Create org-specific client
					orgConfig := *c.baseConfig
					orgConfig.AppInstallationID = installationID

					orgClient, err := github.NewClient(orgConfig)
					if err != nil {
						c.logger.Error("Failed to create org-specific client",
							"org", org,
							"error", err)
						// Fall back to default client
						profilerClient = c.client
					} else {
						c.logger.Debug("Created org-specific client for profiling",
							"org", org,
							"installation_id", installationID)
						profilerClient = orgClient
					}
				}
			}
		} else {
			// PAT or App with installation ID: use default client
			profilerClient = c.client
		}

		profiler = NewProfiler(profilerClient, c.logger)

		// Load org-level package cache and projects map for single repository profiling
		parts := strings.Split(repo.FullName, "/")
		if len(parts) == 2 {
			org := parts[0]

			// Load package cache
			if err := profiler.LoadPackageCache(ctx, org); err != nil {
				c.logger.Warn("Failed to load package cache for single repository",
					"repo", repo.FullName,
					"org", org,
					"error", err)
			}

			// Load ProjectsV2 map
			if err := profiler.LoadProjectsMap(ctx, org); err != nil {
				c.logger.Warn("Failed to load ProjectsV2 map for single repository",
					"repo", repo.FullName,
					"org", org,
					"error", err)
			}

			// Load GitHub App installations
			if err := profiler.LoadOrgInstallations(ctx, org); err != nil {
				c.logger.Warn("Failed to load org installations for single repository",
					"repo", repo.FullName,
					"org", org,
					"error", err)
			}
		} else {
			c.logger.Warn("Invalid repository full name format, cannot load org-level caches",
				"repo", repo.FullName)
		}
	}

	if err := profiler.ProfileFeatures(ctx, repo); err != nil {
		c.logger.Warn("Failed to profile features",
			"repo", repo.FullName,
			"error", err)
	}

	// Check for GitHub migration limit violations and set status accordingly
	if repo.HasOversizedCommits || repo.HasLongRefs || repo.HasBlockingFiles {
		repo.Status = string(models.StatusRemediationRequired)
		c.logger.Warn("Repository requires remediation before migration",
			"repo", repo.FullName,
			"has_oversized_commits", repo.HasOversizedCommits,
			"has_long_refs", repo.HasLongRefs,
			"has_blocking_files", repo.HasBlockingFiles)
	}

	// Save to database
	if err := c.storage.SaveRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to save repository: %w", err)
	}

	// Analyze and save dependencies (only if we cloned the repo)
	if tempDir != "" {
		if err := c.analyzeDependencies(ctx, repo, tempDir, profiler); err != nil {
			c.logger.Warn("Failed to analyze dependencies",
				"repo", repo.FullName,
				"error", err)
			// Don't fail the whole profiling if dependency analysis fails
		}
	}

	// Log the profiled repository with dereferenced values
	sizeBytes := int64(0)
	if repo.TotalSize != nil {
		sizeBytes = *repo.TotalSize
	}
	c.logger.Info("Repository profiled and saved",
		"repo", repo.FullName,
		"size_bytes", sizeBytes,
		"commits", repo.CommitCount)

	return nil
}

// setupTempDir creates and prepares a temporary directory for cloning
func (c *Collector) setupTempDir(fullName string) (string, error) {
	tempBase := c.getTempBaseDir()
	// #nosec G301 -- 0755 is appropriate for temporary directory
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp base directory %s: %w", tempBase, err)
	}

	// Use full name with slashes replaced to avoid collisions between org1/repo and org2/repo
	// For example: "kuhlman-labs-org/node" becomes "kuhlman-labs-org_node"
	safeName := strings.ReplaceAll(fullName, "/", "_")
	tempDir := filepath.Join(tempBase, safeName)

	// Remove if it already exists
	if err := os.RemoveAll(tempDir); err != nil {
		return "", fmt.Errorf("failed to clean existing temp directory: %w", err)
	}

	return tempDir, nil
}

// getTempBaseDir returns the appropriate base directory for temporary repository clones
// In Azure App Service, /tmp may have restrictions, so we use /home/site/tmp
func (c *Collector) getTempBaseDir() string {
	// Check if we're running in Azure App Service
	if os.Getenv("WEBSITE_SITE_NAME") != "" {
		return filepath.Join("/home", "site", "tmp", "gh-migrator")
	}

	// Check if custom temp directory is set
	if customTmp := os.Getenv("GHMIG_TEMP_DIR"); customTmp != "" {
		return filepath.Join(customTmp, "gh-migrator")
	}

	// Default to system temp directory
	return filepath.Join(os.TempDir(), "gh-migrator")
}

// cloneRepositoryWithProvider uses the configured source provider to clone a repository
// Uses full clone (not shallow) for accurate git-sizer analysis of repository history
func (c *Collector) cloneRepositoryWithProvider(ctx context.Context, cloneURL, fullName string) (string, error) {
	tempDir, err := c.setupTempDir(fullName)
	if err != nil {
		return "", err
	}

	// Create repository info for the provider
	repoInfo := source.RepositoryInfo{
		FullName: fullName,
		CloneURL: cloneURL,
	}

	// Use full clone for accurate git-sizer metrics
	// Note: This is slower but necessary for proper analysis of:
	// - Total commit count and history depth
	// - Historical blob sizes and tree entries
	// - Accurate repository size calculations
	opts := source.DefaultCloneOptions()

	c.logger.Debug("Cloning repository for analysis",
		"repo", fullName,
		"shallow", opts.Shallow,
		"bare", opts.Bare)

	// Clone using the provider
	if err := c.sourceProvider.CloneRepository(ctx, repoInfo, tempDir, opts); err != nil {
		// Clean up temp directory on failure
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return tempDir, nil
}

// cloneRepositoryBare creates a bare clone for specialized analysis
// Bare clones are faster and smaller (no working directory) but can't be used for file inspection
// This is useful for pure git-sizer analysis when file content inspection is not needed
// nolint:unused
func (c *Collector) cloneRepositoryBare(ctx context.Context, cloneURL, fullName string) (string, error) {
	tempDir, err := c.setupTempDir(fullName)
	if err != nil {
		return "", err
	}

	// Create repository info for the provider
	repoInfo := source.RepositoryInfo{
		FullName: fullName,
		CloneURL: cloneURL,
	}

	// Use bare clone for maximum git-sizer accuracy with minimal disk space
	opts := source.CloneOptions{
		Shallow:           false, // Full clone for accurate metrics
		Bare:              true,  // Bare clone - no working directory
		IncludeLFS:        false, // Don't fetch LFS content during discovery
		IncludeSubmodules: false, // Don't clone submodules during discovery
	}

	c.logger.Debug("Creating bare clone for git-sizer analysis",
		"repo", fullName,
		"bare", true)

	// Clone using the provider
	if err := c.sourceProvider.CloneRepository(ctx, repoInfo, tempDir, opts); err != nil {
		// Clean up temp directory on failure
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return tempDir, nil
}

// discoverTeamsInParallel processes teams in parallel using worker pool
// This significantly improves performance for organizations with many teams
func (c *Collector) discoverTeamsInParallel(ctx context.Context, org string, client *github.Client, teams []*github.TeamInfo) error {
	if len(teams) == 0 {
		return nil
	}

	jobs := make(chan *github.TeamInfo, len(teams))
	errors := make(chan error, len(teams))
	var wg sync.WaitGroup

	c.logger.Info("Processing teams in parallel",
		"organization", org,
		"team_count", len(teams),
		"workers", c.workers)

	// Start workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.teamDiscoveryWorker(ctx, &wg, i, org, client, jobs, errors)
	}

	// Send jobs
	for _, team := range teams {
		jobs <- team
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
		c.logger.Warn("Team discovery completed with errors",
			"organization", org,
			"total_teams", len(teams),
			"error_count", len(errs))
		return fmt.Errorf("encountered %d errors during team discovery (see logs for details)", len(errs))
	}

	return nil
}

// teamDiscoveryWorker processes teams from the jobs channel
func (c *Collector) teamDiscoveryWorker(ctx context.Context, wg *sync.WaitGroup, workerID int, org string, client *github.Client, jobs <-chan *github.TeamInfo, errors chan<- error) {
	defer wg.Done()

	for teamInfo := range jobs {
		c.logger.Debug("Worker processing team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug)

		// Save the team to the database
		team := &models.GitHubTeam{
			Organization: org,
			Slug:         teamInfo.Slug,
			Name:         teamInfo.Name,
			Privacy:      teamInfo.Privacy,
		}
		if teamInfo.Description != "" {
			team.Description = stringPtr(teamInfo.Description)
		}

		if err := c.storage.SaveTeam(ctx, team); err != nil {
			c.logger.Warn("Failed to save team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"error", err)
			errors <- err
			continue
		}

		// List repositories for this team
		teamRepos, err := client.ListTeamRepositories(ctx, org, teamInfo.Slug)
		if err != nil {
			c.logger.Warn("Failed to list repositories for team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"error", err)
			// Don't send error - continue with members
		} else {
			c.logger.Debug("Found repositories for team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"count", len(teamRepos))

			// Save team-repository associations
			for _, teamRepo := range teamRepos {
				if err := c.storage.SaveTeamRepository(ctx, team.ID, teamRepo.FullName, teamRepo.Permission); err != nil {
					c.logger.Warn("Failed to save team-repository association",
						"worker_id", workerID,
						"organization", org,
						"team", teamInfo.Slug,
						"repo", teamRepo.FullName,
						"error", err)
					// Continue with other repos even if one fails
				}
			}
		}

		// List members for this team using GraphQL (more efficient, no N+1 queries)
		teamMembers, err := client.ListTeamMembersGraphQL(ctx, org, teamInfo.Slug)
		if err != nil {
			c.logger.Warn("Failed to list members for team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"error", err)
			// Continue to next team - don't fail completely
		} else {
			c.logger.Debug("Found members for team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"count", len(teamMembers))

			// Save team members
			for _, member := range teamMembers {
				teamMember := &models.GitHubTeamMember{
					TeamID: team.ID,
					Login:  member.Login,
					Role:   member.Role,
				}
				if err := c.storage.SaveTeamMember(ctx, teamMember); err != nil {
					c.logger.Warn("Failed to save team member",
						"worker_id", workerID,
						"organization", org,
						"team", teamInfo.Slug,
						"member", member.Login,
						"error", err)
					// Continue with other members even if one fails
				}
			}
		}

		c.logger.Debug("Worker completed team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug)
	}
}

// discoverTeams discovers all teams for an organization and their repository associations
// This enables filtering repositories by team membership in the UI
// Uses parallel processing with worker pool for improved performance
func (c *Collector) discoverTeams(ctx context.Context, org string, client *github.Client) error {
	c.logger.Info("Discovering teams for organization", "organization", org)

	// List all teams in the organization
	teams, err := client.ListOrganizationTeams(ctx, org)
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	c.logger.Info("Found teams", "organization", org, "count", len(teams), "workers", c.workers)

	// Process teams in parallel using worker pool
	if err := c.discoverTeamsInParallel(ctx, org, client, teams); err != nil {
		c.logger.Warn("Team discovery completed with errors",
			"organization", org,
			"error", err)
		// Don't return error - some teams may have been processed successfully
	}

	c.logger.Info("Team discovery complete", "organization", org, "teams_found", len(teams))
	return nil
}

// discoverOrgMembers discovers all members of an organization and saves them as users
// Also saves org membership to track which orgs each user belongs to
func (c *Collector) discoverOrgMembers(ctx context.Context, org string, client *github.Client) error {
	c.logger.Info("Discovering organization members", "organization", org)

	members, err := client.ListOrgMembers(ctx, org)
	if err != nil {
		return fmt.Errorf("failed to list org members: %w", err)
	}

	c.logger.Info("Found organization members", "organization", org, "count", len(members))

	sourceInstance := c.getSourceInstance()
	savedCount := 0
	membershipCount := 0

	for _, member := range members {
		user := &models.GitHubUser{
			Login:          member.Login,
			Name:           member.Name,
			Email:          member.Email,
			SourceInstance: sourceInstance,
		}
		// Copy value before taking address to avoid loop variable aliasing
		if member.AvatarURL != "" {
			avatarURL := member.AvatarURL
			user.AvatarURL = &avatarURL
		}

		if err := c.storage.SaveUser(ctx, user); err != nil {
			c.logger.Warn("Failed to save organization member",
				"organization", org,
				"login", member.Login,
				"error", err)
			continue
		}
		savedCount++

		// Save org membership to track which orgs this user belongs to
		membership := &models.UserOrgMembership{
			UserLogin:    member.Login,
			Organization: org,
			Role:         member.Role,
		}
		if err := c.storage.SaveUserOrgMembership(ctx, membership); err != nil {
			c.logger.Warn("Failed to save org membership",
				"organization", org,
				"login", member.Login,
				"error", err)
			// Continue - user was saved, just membership tracking failed
		} else {
			membershipCount++
		}
	}

	c.logger.Info("Organization member discovery complete",
		"organization", org,
		"total_members", len(members),
		"users_saved", savedCount,
		"memberships_saved", membershipCount)
	return nil
}

// getSourceInstance returns the source GitHub instance hostname
func (c *Collector) getSourceInstance() string {
	if c.client == nil {
		return hostGitHubCom
	}

	baseURL := c.client.BaseURL()
	if baseURL == "" {
		return hostGitHubCom
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return hostGitHubCom
	}

	host := parsed.Host
	if host == "" {
		return hostGitHubCom
	}

	return host
}
