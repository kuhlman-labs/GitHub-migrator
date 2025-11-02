package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/source"
	"github.com/brettkuhlman/github-migrator/internal/storage"
	ghapi "github.com/google/go-github/v75/github"
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

	// Process repositories in parallel
	return c.processRepositoriesWithProfiler(ctx, repos, profiler)
}

// DiscoverEnterpriseRepositories discovers all repositories across all organizations in an enterprise
// For GitHub Enterprise Apps without an installation ID, this will:
//  1. Use JWT auth to list all app installations (discovers orgs automatically)
//  2. Create per-org clients with org-specific installation tokens
//  3. Use org-specific clients for all repo operations (higher rate limits, proper isolation)
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

			allRepos = append(allRepos, repos...)
		}
	}

	c.logger.Info("Enterprise discovery complete",
		"enterprise", enterpriseSlug,
		"total_orgs", len(orgs),
		"total_repos", len(allRepos))

	// If not using per-org clients, process all repositories in parallel using the shared profiler
	if !useAppInstallations {
		return c.processRepositoriesWithProfiler(ctx, allRepos, profiler)
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
	repo := &models.Repository{
		FullName:      ghRepo.GetFullName(),
		Source:        "ghes",
		SourceURL:     ghRepo.GetHTMLURL(),
		TotalSize:     &totalSize,
		DefaultBranch: &defaultBranch,
		IsArchived:    ghRepo.GetArchived(),
		IsFork:        ghRepo.GetFork(),
		HasWiki:       ghRepo.GetHasWiki(),
		HasPages:      ghRepo.GetHasPages(),
		HasPackages:   false, // Will be detected by profiler
		Visibility:    ghRepo.GetVisibility(),
		Status:        string(models.StatusPending),
		DiscoveredAt:  time.Now(),
		UpdatedAt:     time.Now(),
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
