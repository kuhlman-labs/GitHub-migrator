package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// RepoDiscoverer handles repository discovery and listing operations.
// This component extracts repository discovery logic from the Collector,
// providing a focused, testable interface for repository operations.
type RepoDiscoverer struct {
	client     *github.Client
	storage    *storage.Database
	logger     *slog.Logger
	baseConfig *github.ClientConfig
}

// NewRepoDiscoverer creates a new RepoDiscoverer instance.
func NewRepoDiscoverer(
	client *github.Client,
	storage *storage.Database,
	logger *slog.Logger,
) *RepoDiscoverer {
	return &RepoDiscoverer{
		client:  client,
		storage: storage,
		logger:  logger,
	}
}

// WithBaseConfig sets the base configuration for creating per-org clients.
func (d *RepoDiscoverer) WithBaseConfig(cfg github.ClientConfig) *RepoDiscoverer {
	d.baseConfig = &cfg
	return d
}

// DiscoveryResult contains the results of a repository discovery operation.
type DiscoveryResult struct {
	Repositories   []*ghapi.Repository
	TotalCount     int
	ProcessedCount int
	FailedCount    int
	Duration       time.Duration
}

// ListOrganizationRepositories lists all repositories for a single organization.
func (d *RepoDiscoverer) ListOrganizationRepositories(ctx context.Context, org string) ([]*ghapi.Repository, error) {
	d.logger.Info("Listing repositories for organization", "org", org)

	client, err := d.getClientForOrg(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for org %s: %w", org, err)
	}

	return d.listAllRepositoriesWithClient(ctx, org, client)
}

// ListEnterpriseRepositories lists all repositories across an enterprise.
// Returns a map of organization name to repositories.
func (d *RepoDiscoverer) ListEnterpriseRepositories(ctx context.Context, enterpriseSlug string) (map[string][]*ghapi.Repository, error) {
	d.logger.Info("Listing repositories for enterprise", "enterprise", enterpriseSlug)

	result := make(map[string][]*ghapi.Repository)

	// Check if we're using GitHub App installations
	useAppInstallations := d.baseConfig != nil && d.baseConfig.AppID > 0 && d.baseConfig.AppInstallationID == 0

	var orgs []string

	if useAppInstallations {
		// Use GitHub App Installations API
		installations, err := d.client.ListAppInstallations(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list app installations: %w", err)
		}

		orgs = make([]string, 0, len(installations))
		for org := range installations {
			orgs = append(orgs, org)
		}

		d.logger.Info("Discovered organizations via app installations", "count", len(orgs))
	} else {
		// Use enterprise GraphQL API
		orgInfos, err := d.client.ListEnterpriseOrganizationsWithCounts(ctx, enterpriseSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to list enterprise organizations: %w", err)
		}

		orgs = make([]string, len(orgInfos))
		for i, info := range orgInfos {
			orgs[i] = info.Login
		}

		d.logger.Info("Discovered organizations via enterprise API",
			"enterprise", enterpriseSlug,
			"count", len(orgs))
	}

	// Collect repositories from each organization
	for _, org := range orgs {
		repos, err := d.ListOrganizationRepositories(ctx, org)
		if err != nil {
			d.logger.Error("Failed to list repositories for organization",
				"org", org,
				"error", err)
			continue
		}
		result[org] = repos
	}

	return result, nil
}

// GetOrganizationRepoCount returns the total number of repositories in an organization.
func (d *RepoDiscoverer) GetOrganizationRepoCount(ctx context.Context, org string) (int, error) {
	client, err := d.getClientForOrg(ctx, org)
	if err != nil {
		return 0, fmt.Errorf("failed to get client for org %s: %w", org, err)
	}

	return client.GetOrganizationRepoCount(ctx, org)
}

// FilterByStatus filters repositories by their current migration status.
func (d *RepoDiscoverer) FilterByStatus(repos []*models.Repository, statuses ...models.MigrationStatus) []*models.Repository {
	statusSet := make(map[string]bool)
	for _, s := range statuses {
		statusSet[string(s)] = true
	}

	filtered := make([]*models.Repository, 0, len(repos))
	for _, repo := range repos {
		if statusSet[repo.Status] {
			filtered = append(filtered, repo)
		}
	}
	return filtered
}

// FilterEligibleForMigration returns repositories that can be migrated.
func (d *RepoDiscoverer) FilterEligibleForMigration(repos []*models.Repository) []*models.Repository {
	eligible := make([]*models.Repository, 0, len(repos))
	for _, repo := range repos {
		if repo.CanBeMigrated() {
			eligible = append(eligible, repo)
		}
	}
	return eligible
}

// FilterEligibleForBatch returns repositories that can be assigned to a batch.
func (d *RepoDiscoverer) FilterEligibleForBatch(repos []*models.Repository) []*models.Repository {
	eligible := make([]*models.Repository, 0, len(repos))
	for _, repo := range repos {
		if ok, _ := repo.CanBeAssignedToBatch(); ok {
			eligible = append(eligible, repo)
		}
	}
	return eligible
}

// Helper methods

// getClientForOrg returns the appropriate client for an organization.
func (d *RepoDiscoverer) getClientForOrg(ctx context.Context, org string) (*github.Client, error) {
	// Check if we need per-org clients (GitHub App without installation ID)
	if d.baseConfig != nil && d.baseConfig.AppID > 0 && d.baseConfig.AppInstallationID == 0 {
		d.logger.Debug("Creating org-specific client",
			"org", org,
			"app_id", d.baseConfig.AppID)

		// Get installation ID for this org
		installationID, err := d.client.GetOrganizationInstallationID(ctx, org)
		if err != nil {
			return nil, fmt.Errorf("failed to get installation ID for org %s: %w", org, err)
		}

		// Create org-specific client
		orgConfig := *d.baseConfig
		orgConfig.AppInstallationID = installationID

		orgClient, err := github.NewClient(orgConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create org-specific client: %w", err)
		}

		return orgClient, nil
	}

	return d.client, nil
}

// listAllRepositoriesWithClient lists all repositories for an organization using a specific client.
func (d *RepoDiscoverer) listAllRepositoriesWithClient(ctx context.Context, org string, client *github.Client) ([]*ghapi.Repository, error) {
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

	d.logger.Debug("Listed repositories for organization",
		"org", org,
		"count", len(allRepos))

	return allRepos, nil
}

// GroupByOrganization groups repositories by their organization.
func (d *RepoDiscoverer) GroupByOrganization(repos []*models.Repository) map[string][]*models.Repository {
	grouped := make(map[string][]*models.Repository)
	for _, repo := range repos {
		org := repo.GetOrganization()
		grouped[org] = append(grouped[org], repo)
	}
	return grouped
}

// GroupByStatus groups repositories by their migration status.
func (d *RepoDiscoverer) GroupByStatus(repos []*models.Repository) map[string][]*models.Repository {
	grouped := make(map[string][]*models.Repository)
	for _, repo := range repos {
		grouped[repo.Status] = append(grouped[repo.Status], repo)
	}
	return grouped
}

// GetRepositoryStats returns statistics about a collection of repositories.
func (d *RepoDiscoverer) GetRepositoryStats(repos []*models.Repository) RepositoryStats {
	stats := RepositoryStats{
		Total:            len(repos),
		StatusCounts:     make(map[string]int),
		ComplexityCounts: make(map[string]int),
	}

	var totalSize int64
	for _, repo := range repos {
		// Status counts
		stats.StatusCounts[repo.Status]++

		// Complexity counts
		complexity := repo.GetComplexityCategoryFromFeatures()
		stats.ComplexityCounts[complexity]++

		// Size totals
		if repo.GetTotalSize() != nil {
			totalSize += *repo.GetTotalSize()
		}

		// Feature counts
		if repo.HasLFS() {
			stats.WithLFS++
		}
		if repo.HasSubmodules() {
			stats.WithSubmodules++
		}
		if repo.HasMigrationBlockers() {
			stats.WithBlockers++
		}
	}

	stats.TotalSizeBytes = totalSize
	return stats
}

// RepositoryStats contains aggregate statistics about repositories.
type RepositoryStats struct {
	Total            int
	TotalSizeBytes   int64
	StatusCounts     map[string]int
	ComplexityCounts map[string]int
	WithLFS          int
	WithSubmodules   int
	WithBlockers     int
}
