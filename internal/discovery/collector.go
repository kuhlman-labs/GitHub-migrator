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

// SetWorkers sets the number of parallel workers for processing
func (c *Collector) SetWorkers(workers int) {
	if workers > 0 {
		c.workers = workers
	}
}

// DiscoverRepositories discovers all repositories from the source organization
func (c *Collector) DiscoverRepositories(ctx context.Context, org string) error {
	c.logger.Info("Starting repository discovery", "organization", org)

	// List all repositories
	repos, err := c.listAllRepositories(ctx, org)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	c.logger.Info("Found repositories", "count", len(repos))

	// Create profiler and load package cache for the organization
	profiler := NewProfiler(c.client, c.logger)
	if err := profiler.LoadPackageCache(ctx, org); err != nil {
		c.logger.Warn("Failed to load package cache, package detection may be slower",
			"org", org,
			"error", err)
	}

	// Process repositories in parallel
	return c.processRepositoriesWithProfiler(ctx, repos, profiler)
}

// DiscoverEnterpriseRepositories discovers all repositories across all organizations in an enterprise
func (c *Collector) DiscoverEnterpriseRepositories(ctx context.Context, enterpriseSlug string) error {
	c.logger.Info("Starting enterprise-wide repository discovery", "enterprise", enterpriseSlug)

	// Get all organizations in the enterprise
	orgs, err := c.client.ListEnterpriseOrganizations(ctx, enterpriseSlug)
	if err != nil {
		return fmt.Errorf("failed to list enterprise organizations: %w", err)
	}

	c.logger.Info("Found organizations in enterprise",
		"enterprise", enterpriseSlug,
		"org_count", len(orgs))

	// Create a shared profiler for all organizations
	profiler := NewProfiler(c.client, c.logger)

	// Collect repositories from all organizations and load package caches
	var allRepos []*ghapi.Repository
	for _, org := range orgs {
		c.logger.Info("Discovering repositories for organization",
			"enterprise", enterpriseSlug,
			"organization", org)

		repos, err := c.listAllRepositories(ctx, org)
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

		// Load package cache for this organization
		if err := profiler.LoadPackageCache(ctx, org); err != nil {
			c.logger.Warn("Failed to load package cache for organization",
				"enterprise", enterpriseSlug,
				"org", org,
				"error", err)
		}

		allRepos = append(allRepos, repos...)
	}

	c.logger.Info("Enterprise discovery complete",
		"enterprise", enterpriseSlug,
		"total_orgs", len(orgs),
		"total_repos", len(allRepos))

	// Process all repositories in parallel using the shared profiler
	return c.processRepositoriesWithProfiler(ctx, allRepos, profiler)
}

// listAllRepositories lists all repositories for an organization with pagination
func (c *Collector) listAllRepositories(ctx context.Context, org string) ([]*ghapi.Repository, error) {
	var allRepos []*ghapi.Repository
	opts := &ghapi.RepositoryListByOrgOptions{
		ListOptions: ghapi.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := c.client.REST().Repositories.ListByOrg(ctx, org, opts)
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

	c.logger.Info("Destination repository profiled",
		"repo", repo.FullName,
		"size", repo.TotalSize,
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
		profiler = NewProfiler(c.client, c.logger)
	}

	if err := profiler.ProfileFeatures(ctx, repo); err != nil {
		c.logger.Warn("Failed to profile features",
			"repo", repo.FullName,
			"error", err)
	}

	// Save to database
	if err := c.storage.SaveRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to save repository: %w", err)
	}

	c.logger.Info("Repository profiled and saved",
		"repo", repo.FullName,
		"size", repo.TotalSize,
		"commits", repo.CommitCount)

	return nil
}

// setupTempDir creates and prepares a temporary directory for cloning
func (c *Collector) setupTempDir(fullName string) (string, error) {
	tempBase := filepath.Join(os.TempDir(), "gh-migrator")
	// #nosec G301 -- 0755 is appropriate for temporary directory
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp base directory: %w", err)
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
