package discovery

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	ghapi "github.com/google/go-github/v75/github"
	"github.com/shurcooL/githubv4"
)

// Profiler profiles GitHub-specific features via API
type Profiler struct {
	client         *github.Client
	logger         *slog.Logger
	token          string          // GitHub token for authenticated git operations
	packageCache   map[string]bool // Cache of repo names that have packages
	packageCacheMu sync.RWMutex    // Mutex for thread-safe cache access
}

// NewProfiler creates a new GitHub features profiler
func NewProfiler(client *github.Client, logger *slog.Logger) *Profiler {
	return &Profiler{
		client:       client,
		logger:       logger,
		token:        client.Token(),
		packageCache: make(map[string]bool),
	}
}

// ProfileFeatures profiles GitHub-specific features via API
func (p *Profiler) ProfileFeatures(ctx context.Context, repo *models.Repository) error {
	parts := strings.SplitN(repo.FullName, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid full_name format: %s (expected: org/repo)", repo.FullName)
	}
	org := parts[0]
	name := parts[1]

	p.logger.Debug("Profiling GitHub features",
		"repo", repo.FullName,
		"org", org,
		"name", name)

	// Get repository details
	ghRepo, _, err := p.client.REST().Repositories.Get(ctx, org, name)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Set GitHub-specific features from repository object
	repo.HasDiscussions = ghRepo.GetHasDiscussions()
	repo.HasProjects = ghRepo.GetHasProjects()

	// Profile various GitHub features
	p.profileWorkflowCount(ctx, org, name, repo)
	p.profileBranchProtections(ctx, org, name, repo)
	p.profileRulesets(ctx, org, name, repo)
	p.profileEnvironments(ctx, org, name, repo)
	p.profileWebhooks(ctx, org, name, repo)
	p.profileContributors(ctx, org, name, repo)
	p.profileTags(ctx, org, name, repo)
	p.profilePackages(ctx, org, name, repo)

	// Profile advanced features
	p.profileSecurity(ctx, org, name, repo)
	p.profileCodeowners(ctx, org, name, repo)
	p.profileRunners(ctx, org, name, repo)
	p.profileCollaborators(ctx, org, name, repo)
	p.profileApps(ctx, org, name, repo)
	p.profileReleases(ctx, org, name, repo)

	// Check if wiki actually has content (not just enabled)
	p.profileWikiContent(ctx, repo)

	// Get issue counts for verification
	if err := p.countIssuesAndPRs(ctx, org, name, repo); err != nil {
		p.logger.Debug("Failed to get issue/PR counts", "error", err)
	}

	p.logger.Info("GitHub features profiled",
		"repo", repo.FullName,
		"has_actions", repo.HasActions,
		"workflow_count", repo.WorkflowCount,
		"has_wiki", repo.HasWiki,
		"has_pages", repo.HasPages,
		"has_discussions", repo.HasDiscussions,
		"has_packages", repo.HasPackages,
		"has_rulesets", repo.HasRulesets,
		"has_code_scanning", repo.HasCodeScanning,
		"has_dependabot", repo.HasDependabot,
		"has_secret_scanning", repo.HasSecretScanning,
		"has_codeowners", repo.HasCodeowners,
		"has_self_hosted_runners", repo.HasSelfHostedRunners,
		"collaborator_count", repo.CollaboratorCount,
		"installed_apps_count", repo.InstalledAppsCount,
		"release_count", repo.ReleaseCount,
		"has_release_assets", repo.HasReleaseAssets,
		"contributors", repo.ContributorCount,
		"issues", repo.IssueCount,
		"prs", repo.PullRequestCount,
		"tags", repo.TagCount)

	return nil
}

// profileBranchProtections counts protected branches
func (p *Profiler) profileBranchProtections(ctx context.Context, org, name string, repo *models.Repository) {
	branches, _, err := p.client.REST().Repositories.ListBranches(ctx, org, name, nil)
	if err == nil {
		protectedCount := 0
		for _, branch := range branches {
			if branch.GetProtected() {
				protectedCount++
			}
		}
		repo.BranchProtections = protectedCount
	} else {
		p.logger.Debug("Failed to get branches", "error", err)
	}
}

// profileRulesets checks if the repository has any rulesets configured
// Rulesets are the newer version of branch protections and don't migrate with GEI
func (p *Profiler) profileRulesets(ctx context.Context, org, name string, repo *models.Repository) {
	// List repository rulesets
	includeParents := false
	rulesets, _, err := p.client.REST().Repositories.GetAllRulesets(ctx, org, name, &ghapi.RepositoryListRulesetsOptions{
		IncludesParents: &includeParents,
	})
	if err == nil && len(rulesets) > 0 {
		repo.HasRulesets = true
		p.logger.Debug("Repository has rulesets", "repo", repo.FullName, "count", len(rulesets))
	} else if err != nil {
		p.logger.Debug("Failed to get rulesets", "error", err)
	}
}

// profileEnvironments counts deployment environments
func (p *Profiler) profileEnvironments(ctx context.Context, org, name string, repo *models.Repository) {
	environments, _, err := p.client.REST().Repositories.ListEnvironments(ctx, org, name, nil)
	if err == nil && environments != nil {
		repo.EnvironmentCount = environments.GetTotalCount()
	} else {
		p.logger.Debug("Failed to get environments", "error", err)
	}
}

// profileWebhooks counts webhooks
func (p *Profiler) profileWebhooks(ctx context.Context, org, name string, repo *models.Repository) {
	hooks, _, err := p.client.REST().Repositories.ListHooks(ctx, org, name, nil)
	if err == nil {
		repo.WebhookCount = len(hooks)
	} else {
		p.logger.Debug("Failed to get webhooks", "error", err)
	}
}

// profileContributors gets contributor information with pagination
func (p *Profiler) profileContributors(ctx context.Context, org, name string, repo *models.Repository) {
	opts := &ghapi.ListContributorsOptions{
		ListOptions: ghapi.ListOptions{PerPage: 100},
	}

	var allContributors []*ghapi.Contributor
	topContributors := make([]string, 0, 5)

	// Paginate through all contributors
	for {
		contributors, resp, err := p.client.REST().Repositories.ListContributors(ctx, org, name, opts)
		if err != nil {
			p.logger.Debug("Failed to get contributors", "error", err)
			return
		}

		allContributors = append(allContributors, contributors...)

		// Store top 5 contributors from the first page (already sorted by contributions)
		if opts.Page == 0 && len(topContributors) < 5 {
			for i, contributor := range contributors {
				if i >= 5 {
					break
				}
				topContributors = append(topContributors, contributor.GetLogin())
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	repo.ContributorCount = len(allContributors)
	if len(topContributors) > 0 {
		topContribStr := strings.Join(topContributors, ",")
		repo.TopContributors = &topContribStr
	}
}

// profileTags counts repository tags with pagination
func (p *Profiler) profileTags(ctx context.Context, org, name string, repo *models.Repository) {
	opts := &ghapi.ListOptions{PerPage: 100}
	totalTags := 0

	// Paginate through all tags
	for {
		tags, resp, err := p.client.REST().Repositories.ListTags(ctx, org, name, opts)
		if err != nil {
			p.logger.Debug("Failed to get tags", "error", err)
			return
		}

		totalTags += len(tags)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	repo.TagCount = totalTags
}

// LoadPackageCache loads all packages for an organization into the cache
// This should be called once per organization before profiling repositories
func (p *Profiler) LoadPackageCache(ctx context.Context, org string) error {
	p.logger.Info("Loading package cache for organization", "org", org)

	// GitHub package types to check
	packageTypes := []string{"npm", "maven", "rubygems", "docker", "nuget", "container"}

	p.packageCacheMu.Lock()
	defer p.packageCacheMu.Unlock()

	// Clear existing cache for this org
	p.packageCache = make(map[string]bool)

	packagesFound := 0
	for _, pkgType := range packageTypes {
		opts := &ghapi.PackageListOptions{
			PackageType: ghapi.String(pkgType),
			ListOptions: ghapi.ListOptions{PerPage: 100},
		}

		for {
			packages, resp, err := p.client.REST().Organizations.ListPackages(ctx, org, opts)
			if err != nil {
				// Some package types may not be available or may require different permissions
				p.logger.Debug("Failed to list packages for type (continuing with other types)",
					"org", org,
					"type", pkgType,
					"error", err)
				break
			}

			// Mark repositories that have packages
			for _, pkg := range packages {
				if pkg.Repository != nil {
					repoName := pkg.Repository.GetName()
					if !p.packageCache[repoName] {
						p.packageCache[repoName] = true
						packagesFound++
						p.logger.Debug("Found package for repository",
							"org", org,
							"repo", repoName,
							"package_type", pkgType,
							"package_name", pkg.GetName())
					}
				}
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	p.logger.Info("Package cache loaded",
		"org", org,
		"repos_with_packages", len(p.packageCache),
		"total_packages", packagesFound)

	return nil
}

// profilePackages checks for GitHub Packages using the cache
func (p *Profiler) profilePackages(ctx context.Context, org, name string, repo *models.Repository) {
	// Check cache first (thread-safe read)
	p.packageCacheMu.RLock()
	hasPackages, inCache := p.packageCache[name]
	p.packageCacheMu.RUnlock()

	if inCache {
		repo.HasPackages = hasPackages
		if hasPackages {
			p.logger.Debug("Found packages for repository (from cache)", "repo", repo.FullName)
		}
		return
	}

	// If not in cache, try GraphQL as fallback
	hasPackages, err := p.detectPackagesViaGraphQL(ctx, org, name)
	if err == nil {
		repo.HasPackages = hasPackages
		if hasPackages {
			p.logger.Debug("Found packages for repository via GraphQL", "repo", repo.FullName)
		}
		return
	}

	p.logger.Debug("GraphQL package detection failed",
		"repo", repo.FullName,
		"error", err)

	// If we couldn't detect packages, default to false
	repo.HasPackages = false
}

// detectPackagesViaGraphQL uses GraphQL to detect if a repository has packages
func (p *Profiler) detectPackagesViaGraphQL(ctx context.Context, org, name string) (bool, error) {
	var query struct {
		Repository struct {
			Packages struct {
				TotalCount int
			} `graphql:"packages(first: 1)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(org),
		"name":  githubv4.String(name),
	}

	err := p.client.QueryWithRetry(ctx, "DetectPackages", &query, variables)
	if err != nil {
		return false, err
	}

	return query.Repository.Packages.TotalCount > 0, nil
}

// countIssuesAndPRs counts issues and PRs separately for accurate verification data
// GitHub's open_issues_count includes both issues and PRs, so we need separate API calls
func (p *Profiler) countIssuesAndPRs(ctx context.Context, org, name string, repo *models.Repository) error {
	// Count all issues (GitHub API treats PRs as issues, so we need to filter)
	// We'll use a single page request with state=all and count manually
	allIssuesOpts := &ghapi.IssueListByRepoOptions{
		State:       "all",
		ListOptions: ghapi.ListOptions{PerPage: 100},
	}

	openIssues := 0
	totalIssues := 0

	// Paginate through all issues
	for {
		issues, resp, err := p.client.REST().Issues.ListByRepo(ctx, org, name, allIssuesOpts)
		if err != nil {
			return fmt.Errorf("failed to list issues: %w", err)
		}

		for _, issue := range issues {
			// Skip pull requests (they have a PullRequestLinks field)
			if issue.PullRequestLinks == nil {
				totalIssues++
				if issue.GetState() == "open" {
					openIssues++
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		allIssuesOpts.ListOptions.Page = resp.NextPage
	}

	repo.IssueCount = totalIssues
	repo.OpenIssueCount = openIssues

	// Count pull requests
	allPRsOpts := &ghapi.PullRequestListOptions{
		State:       "all",
		ListOptions: ghapi.ListOptions{PerPage: 100},
	}

	openPRs := 0
	totalPRs := 0

	// Paginate through all PRs
	for {
		prs, resp, err := p.client.REST().PullRequests.List(ctx, org, name, allPRsOpts)
		if err != nil {
			return fmt.Errorf("failed to list pull requests: %w", err)
		}

		totalPRs += len(prs)
		for _, pr := range prs {
			if pr.GetState() == "open" {
				openPRs++
			}
		}

		if resp.NextPage == 0 {
			break
		}
		allPRsOpts.ListOptions.Page = resp.NextPage
	}

	repo.PullRequestCount = totalPRs
	repo.OpenPRCount = openPRs

	return nil
}

// profileSecurity checks for GitHub Advanced Security features
func (p *Profiler) profileSecurity(ctx context.Context, org, name string, repo *models.Repository) {
	// Code Scanning - check if the API endpoint is accessible
	_, resp, err := p.client.REST().CodeScanning.ListAlertsForRepo(ctx, org, name, &ghapi.AlertListOptions{
		ListOptions: ghapi.ListOptions{PerPage: 1},
	})
	if err == nil && resp.StatusCode != 404 {
		// If we can access the endpoint, code scanning is enabled
		repo.HasCodeScanning = true
	} else {
		repo.HasCodeScanning = false
		p.logger.Debug("Code scanning not available", "repo", repo.FullName)
	}

	// Dependabot - check if alerts endpoint is accessible
	_, resp, err = p.client.REST().Dependabot.ListRepoAlerts(ctx, org, name, &ghapi.ListAlertsOptions{
		ListOptions: ghapi.ListOptions{PerPage: 1},
	})
	if err == nil && resp.StatusCode != 404 {
		repo.HasDependabot = true
	} else {
		repo.HasDependabot = false
		p.logger.Debug("Dependabot not available", "repo", repo.FullName)
	}

	// Secret Scanning - check if alerts endpoint is accessible
	_, resp, err = p.client.REST().SecretScanning.ListAlertsForRepo(ctx, org, name, &ghapi.SecretScanningAlertListOptions{
		ListOptions: ghapi.ListOptions{PerPage: 1},
	})
	if err == nil && resp.StatusCode != 404 {
		repo.HasSecretScanning = true
	} else {
		repo.HasSecretScanning = false
		p.logger.Debug("Secret scanning not available", "repo", repo.FullName)
	}
}

// profileCodeowners checks for CODEOWNERS file in common locations
func (p *Profiler) profileCodeowners(ctx context.Context, org, name string, repo *models.Repository) {
	// Check common locations: .github/CODEOWNERS, docs/CODEOWNERS, CODEOWNERS
	locations := []string{".github/CODEOWNERS", "docs/CODEOWNERS", "CODEOWNERS"}

	for _, path := range locations {
		_, _, resp, err := p.client.REST().Repositories.GetContents(ctx, org, name, path, nil)
		if err == nil && resp.StatusCode == 200 {
			repo.HasCodeowners = true
			p.logger.Debug("Found CODEOWNERS file", "repo", repo.FullName, "path", path)
			return
		}
	}

	repo.HasCodeowners = false
}

// profileWorkflowCount counts GitHub Actions workflows (enhances existing workflow detection)
func (p *Profiler) profileWorkflowCount(ctx context.Context, org, name string, repo *models.Repository) {
	workflows, _, err := p.client.REST().Actions.ListWorkflows(ctx, org, name, nil)
	if err == nil && workflows != nil {
		repo.WorkflowCount = workflows.GetTotalCount()
		repo.HasActions = workflows.GetTotalCount() > 0
	} else {
		repo.WorkflowCount = 0
		repo.HasActions = false
		p.logger.Debug("Failed to get workflows", "error", err)
	}
}

// profileRunners checks for self-hosted runners
func (p *Profiler) profileRunners(ctx context.Context, org, name string, repo *models.Repository) {
	runners, _, err := p.client.REST().Actions.ListRunners(ctx, org, name, nil)
	if err == nil && runners != nil {
		// Check if any runners are self-hosted (not GitHub-hosted)
		for _, runner := range runners.Runners {
			if !isGitHubHosted(runner.GetName()) {
				repo.HasSelfHostedRunners = true
				p.logger.Debug("Found self-hosted runner", "repo", repo.FullName, "runner", runner.GetName())
				return
			}
		}
	} else {
		p.logger.Debug("Failed to get runners", "error", err)
	}
	repo.HasSelfHostedRunners = false
}

// isGitHubHosted checks if a runner name indicates it's GitHub-hosted
func isGitHubHosted(name string) bool {
	// GitHub-hosted runners typically have names containing "GitHub Actions" or "Hosted Agent"
	nameLower := strings.ToLower(name)
	return strings.Contains(nameLower, "github actions") ||
		strings.Contains(nameLower, "hosted agent") ||
		strings.Contains(nameLower, "azure pipelines")
}

// profileCollaborators counts outside collaborators
func (p *Profiler) profileCollaborators(ctx context.Context, org, name string, repo *models.Repository) {
	// List collaborators with affiliation filter for outside collaborators
	opts := &ghapi.ListCollaboratorsOptions{
		Affiliation: "outside",
		ListOptions: ghapi.ListOptions{PerPage: 100},
	}

	outsideCount := 0
	for {
		collaborators, resp, err := p.client.REST().Repositories.ListCollaborators(ctx, org, name, opts)
		if err != nil {
			p.logger.Debug("Failed to get collaborators", "error", err)
			break
		}

		outsideCount += len(collaborators)

		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	repo.CollaboratorCount = outsideCount
}

// profileApps counts installed GitHub Apps for the repository
func (p *Profiler) profileApps(ctx context.Context, org, name string, repo *models.Repository) {
	// Get installations for this repository
	// Note: This requires the app to have the appropriate permissions
	installations, _, err := p.client.REST().Apps.ListRepos(ctx, &ghapi.ListOptions{PerPage: 100})
	if err != nil {
		p.logger.Debug("Failed to list app installations (requires app token)", "error", err)
		repo.InstalledAppsCount = 0
		return
	}

	// Count apps that have access to this specific repository
	appCount := 0
	for _, installation := range installations.Repositories {
		if installation.GetFullName() == repo.FullName {
			appCount++
		}
	}

	repo.InstalledAppsCount = appCount
}

// profileReleases counts releases and checks for assets
func (p *Profiler) profileReleases(ctx context.Context, org, name string, repo *models.Repository) {
	releases, _, err := p.client.REST().Repositories.ListReleases(ctx, org, name, &ghapi.ListOptions{PerPage: 100})
	if err != nil {
		p.logger.Debug("Failed to list releases", "error", err)
		repo.ReleaseCount = 0
		repo.HasReleaseAssets = false
		return
	}

	repo.ReleaseCount = len(releases)

	// Check if any releases have assets
	for _, release := range releases {
		if len(release.Assets) > 0 {
			repo.HasReleaseAssets = true
			p.logger.Debug("Found release assets", "repo", repo.FullName, "release", release.GetTagName())
			return
		}
	}

	repo.HasReleaseAssets = false
}

// profileWikiContent checks if the wiki actually has content, not just if it's enabled
// GitHub wikis are separate git repositories, so we check if the wiki repo exists and has commits
func (p *Profiler) profileWikiContent(ctx context.Context, repo *models.Repository) {
	// If wiki is not enabled, skip the check
	if !repo.HasWiki {
		return
	}

	// Get the wiki URL from the repository source URL
	// Wiki repos follow the pattern: https://github.com/org/repo.wiki.git
	wikiURL := repo.SourceURL
	if !strings.HasSuffix(wikiURL, ".git") {
		wikiURL += ".git"
	}
	wikiURL = strings.TrimSuffix(wikiURL, ".git") + ".wiki.git"

	p.logger.Debug("Checking wiki content", "repo", repo.FullName, "wiki_url", wikiURL)

	// Use git ls-remote to check if wiki repo exists and has content
	// This is fast and doesn't require cloning the entire wiki
	hasContent, err := p.checkWikiHasContent(ctx, wikiURL)
	if err != nil {
		p.logger.Debug("Failed to check wiki content, assuming no content",
			"repo", repo.FullName,
			"error", err)
		repo.HasWiki = false
		return
	}

	// Update HasWiki to reflect actual content presence
	repo.HasWiki = hasContent

	if !hasContent {
		p.logger.Debug("Wiki feature enabled but no content found",
			"repo", repo.FullName)
	}
}

// checkWikiHasContent checks if a wiki repository exists and has content
// Uses git ls-remote which is fast and doesn't require cloning
func (p *Profiler) checkWikiHasContent(ctx context.Context, wikiURL string) (bool, error) {
	// Add authentication to the URL if we have a token
	// Format: https://TOKEN@github.com/org/repo.wiki.git
	authenticatedURL := wikiURL
	if p.token != "" {
		// Parse and inject the token into the URL
		if strings.HasPrefix(wikiURL, "https://") {
			authenticatedURL = strings.Replace(wikiURL, "https://", fmt.Sprintf("https://%s@", p.token), 1)
		} else if strings.HasPrefix(wikiURL, "http://") {
			authenticatedURL = strings.Replace(wikiURL, "http://", fmt.Sprintf("http://%s@", p.token), 1)
		}
	}

	// Use git ls-remote to check if the wiki repo exists and has refs
	// If the wiki has been initialized, it will have at least a master/main branch
	// #nosec G204 -- wikiURL is constructed from controlled repository data
	cmd := exec.CommandContext(ctx, "git", "ls-remote", authenticatedURL)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set GIT_TERMINAL_PROMPT=0 to prevent interactive prompts
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")

	err := cmd.Run()
	if err != nil {
		// If ls-remote fails, the wiki likely doesn't exist or is empty
		// Don't return the error since an empty wiki is not an error condition
		return false, nil
	}

	// Check if there's any output (refs)
	// An empty or non-existent wiki will have no refs
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return false, nil
	}

	// If we have refs, the wiki has content
	return true, nil
}
