package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
)

// OrgInstallationInfo holds cached app installation info for an org
type OrgInstallationInfo struct {
	AppSlug             string
	RepositorySelection string   // "all" or "selected"
	SelectedRepos       []string // populated only for "selected" installations
}

// Profiler profiles GitHub-specific features via API
type Profiler struct {
	client            *github.Client
	logger            *slog.Logger
	token             string                            // GitHub token for authenticated git operations
	packageCache      map[string]bool                   // Cache of repo names that have packages
	packageCacheMu    sync.RWMutex                      // Mutex for thread-safe cache access
	projectsMap       map[string]bool                   // Cache of repo names that have ProjectsV2 (org-level)
	projectsMapMu     sync.RWMutex                      // Mutex for thread-safe projects map access
	orgInstallations  map[string][]*OrgInstallationInfo // Cache of org installations by org name
	orgInstallationMu sync.RWMutex                      // Mutex for thread-safe installations access
}

// NewProfiler creates a new GitHub features profiler
func NewProfiler(client *github.Client, logger *slog.Logger) *Profiler {
	return &Profiler{
		client:           client,
		logger:           logger,
		token:            client.Token(),
		packageCache:     make(map[string]bool),
		projectsMap:      make(map[string]bool),
		orgInstallations: make(map[string][]*OrgInstallationInfo),
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
	p.profileSecrets(ctx, org, name, repo)
	p.profileVariables(ctx, org, name, repo)
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

	// Check if projects (classic) actually exist (not just enabled)
	p.profileProjectContent(ctx, org, name, repo)

	// Get issue counts for verification
	if err := p.countIssuesAndPRs(ctx, org, name, repo); err != nil {
		p.logger.Debug("Failed to get issue/PR counts", "error", err)
	}

	// Estimate metadata size for GitHub Enterprise Importer 40 GiB limit
	if err := p.estimateMetadataSize(ctx, org, name, repo); err != nil {
		p.logger.Debug("Failed to estimate metadata size", "error", err)
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
		"environment_count", repo.EnvironmentCount,
		"secret_count", repo.SecretCount,
		"variable_count", repo.VariableCount,
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

// profileSecrets counts repository secrets (Actions secrets)
func (p *Profiler) profileSecrets(ctx context.Context, org, name string, repo *models.Repository) {
	secrets, _, err := p.client.REST().Actions.ListRepoSecrets(ctx, org, name, nil)
	if err == nil && secrets != nil {
		// Secrets type has TotalCount field
		repo.SecretCount = secrets.TotalCount
	} else {
		p.logger.Debug("Failed to get secrets", "error", err)
	}
}

// profileVariables counts repository variables (Actions variables)
func (p *Profiler) profileVariables(ctx context.Context, org, name string, repo *models.Repository) {
	variables, _, err := p.client.REST().Actions.ListRepoVariables(ctx, org, name, nil)
	if err == nil && variables != nil {
		// ActionsVariables type has TotalCount field
		repo.VariableCount = variables.TotalCount
	} else {
		p.logger.Debug("Failed to get variables", "error", err)
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
			PackageType: ghapi.Ptr(pkgType),
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

// LoadProjectsMap loads all ProjectsV2 for an organization into the cache
// This should be called once per organization before profiling repositories
func (p *Profiler) LoadProjectsMap(ctx context.Context, org string) error {
	p.logger.Info("Loading ProjectsV2 map for organization", "org", org)

	// Fetch organization projects using GraphQL
	projectsMap, err := p.client.ListOrganizationProjects(ctx, org)
	if err != nil {
		p.logger.Warn("Failed to load ProjectsV2 map", "org", org, "error", err)
		return err
	}

	p.projectsMapMu.Lock()
	defer p.projectsMapMu.Unlock()

	// Store the projects map
	p.projectsMap = projectsMap

	p.logger.Info("ProjectsV2 map loaded",
		"org", org,
		"repos_with_projects", len(projectsMap))

	return nil
}

// LoadOrgInstallations loads and caches GitHub App installations for an organization.
// This is called once per org during discovery to avoid repeated API calls.
func (p *Profiler) LoadOrgInstallations(ctx context.Context, org string) error {
	p.logger.Info("Loading GitHub App installations for organization", "org", org)

	// Fetch org installations
	installations, err := p.client.ListOrgInstallations(ctx, org)
	if err != nil {
		p.logger.Warn("Failed to load org installations", "org", org, "error", err)
		return err
	}

	// Convert to our cached format
	cachedInstallations := make([]*OrgInstallationInfo, 0, len(installations))
	for _, install := range installations {
		info := &OrgInstallationInfo{
			AppSlug:             install.AppSlug,
			RepositorySelection: install.RepositorySelection,
		}

		// For "selected" installations, try to get the list of repos
		// This can fail if we don't have permission, which is fine
		if install.RepositorySelection == "selected" {
			repos, err := p.client.ListInstallationRepos(ctx, install.ID)
			if err != nil {
				p.logger.Debug("Could not fetch repos for installation",
					"app", install.AppSlug,
					"error", err)
			} else {
				info.SelectedRepos = repos
			}
		}

		cachedInstallations = append(cachedInstallations, info)
	}

	p.orgInstallationMu.Lock()
	defer p.orgInstallationMu.Unlock()

	p.orgInstallations[org] = cachedInstallations

	p.logger.Info("Org installations loaded",
		"org", org,
		"count", len(cachedInstallations))

	return nil
}

// profilePackages checks for GitHub Packages using the REST API cache
func (p *Profiler) profilePackages(ctx context.Context, org, name string, repo *models.Repository) {
	// Check cache (thread-safe read)
	// The cache is populated by LoadPackageCache which queries all package types via REST API
	p.packageCacheMu.RLock()
	hasPackages, inCache := p.packageCache[name]
	p.packageCacheMu.RUnlock()

	if inCache && hasPackages {
		repo.HasPackages = true
		p.logger.Debug("Found packages for repository (from REST API cache)", "repo", repo.FullName)
		return
	}

	// If not in cache or explicitly false, the repository has no packages
	// The cache is built from org-level package queries across all package types
	repo.HasPackages = false
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

// profileCodeowners checks for CODEOWNERS file in common locations and parses its content
func (p *Profiler) profileCodeowners(ctx context.Context, org, name string, repo *models.Repository) {
	// Check common locations: .github/CODEOWNERS, docs/CODEOWNERS, CODEOWNERS
	locations := []string{".github/CODEOWNERS", "docs/CODEOWNERS", "CODEOWNERS"}

	p.logger.Debug("Checking for CODEOWNERS file", "repo", repo.FullName)

	for _, path := range locations {
		fileContent, _, resp, err := p.client.REST().Repositories.GetContents(ctx, org, name, path, nil)

		if err != nil {
			p.logger.Debug("CODEOWNERS check failed at location",
				"repo", repo.FullName,
				"path", path,
				"error", err.Error(),
				"is_404", resp != nil && resp.StatusCode == 404)
			continue
		}

		if resp != nil && resp.StatusCode == 200 {
			// Verify it's a file, not a directory
			if fileContent != nil && fileContent.GetType() == "file" {
				repo.HasCodeowners = true
				p.logger.Debug("Found CODEOWNERS file",
					"repo", repo.FullName,
					"path", path,
					"size", fileContent.GetSize())

				// Parse CODEOWNERS content to extract team and user references
				if content, err := fileContent.GetContent(); err == nil && content != "" {
					p.parseCodeownersContent(repo, content)
				}
				return
			} else {
				p.logger.Debug("Path exists but is not a file",
					"repo", repo.FullName,
					"path", path,
					"type", fileContent.GetType())
			}
		}
	}

	repo.HasCodeowners = false
	p.logger.Debug("No CODEOWNERS file found", "repo", repo.FullName)
}

// extractCodeownersReferences extracts team and user references from CODEOWNERS content
func extractCodeownersReferences(content string) (teams, users map[string]bool) {
	teams = make(map[string]bool)
	users = make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// Process each owner (skip the first part which is the pattern)
		for _, owner := range parts[1:] {
			if strings.HasPrefix(owner, "#") {
				break
			}
			classifyCodeowner(owner, teams, users)
		}
	}
	return teams, users
}

// classifyCodeowner determines if an owner is a team, user, or email reference
func classifyCodeowner(owner string, teams, users map[string]bool) {
	if strings.HasPrefix(owner, "@") && strings.Contains(owner, "/") {
		teams[owner] = true
	} else if strings.HasPrefix(owner, "@") {
		users[owner] = true
	} else if strings.Contains(owner, "@") {
		users[owner] = true
	}
}

// stringPtr returns a pointer to a heap-allocated copy of the string.
func stringPtr(s string) *string {
	ptr := new(string)
	*ptr = s
	return ptr
}

// storeCodeownersJSON stores extracted references as JSON in the repository
func storeCodeownersJSON(repo *models.Repository, teams, users map[string]bool) {
	if len(teams) > 0 {
		teamList := make([]string, 0, len(teams))
		for team := range teams {
			teamList = append(teamList, team)
		}
		if teamsJSON, err := json.Marshal(teamList); err == nil {
			repo.CodeownersTeams = stringPtr(string(teamsJSON))
		}
	}

	if len(users) > 0 {
		userList := make([]string, 0, len(users))
		for user := range users {
			userList = append(userList, user)
		}
		if usersJSON, err := json.Marshal(userList); err == nil {
			repo.CodeownersUsers = stringPtr(string(usersJSON))
		}
	}
}

// parseCodeownersContent parses CODEOWNERS file content and extracts team/user references
func (p *Profiler) parseCodeownersContent(repo *models.Repository, content string) {
	// Store the raw content using stringPtr to ensure heap allocation
	repo.CodeownersContent = stringPtr(content)

	// Parse and extract team and user references
	teams, users := extractCodeownersReferences(content)

	// Store as JSON
	storeCodeownersJSON(repo, teams, users)

	p.logger.Debug("Parsed CODEOWNERS content",
		"repo", repo.FullName,
		"teams_found", len(teams),
		"users_found", len(users))
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

// profileApps identifies GitHub Apps installed for the repository
// Uses cached org-level installations to determine which apps have access to this repo
func (p *Profiler) profileApps(ctx context.Context, org, name string, repo *models.Repository) {
	p.orgInstallationMu.RLock()
	installations, ok := p.orgInstallations[org]
	p.orgInstallationMu.RUnlock()

	if !ok || len(installations) == 0 {
		// No cached installations for this org
		p.logger.Debug("No cached installations for org", "org", org)
		repo.InstalledAppsCount = 0
		return
	}

	// Collect app names that have access to this specific repository
	var appNames []string
	repoFullName := repo.FullName

	for _, install := range installations {
		hasAccess := false

		if install.RepositorySelection == "all" {
			// App has access to all repos in the org
			hasAccess = true
		} else if install.RepositorySelection == "selected" {
			// Check if this repo is in the selected repos list
			for _, selectedRepo := range install.SelectedRepos {
				if selectedRepo == repoFullName {
					hasAccess = true
					break
				}
			}
		}

		if hasAccess && install.AppSlug != "" {
			appNames = append(appNames, install.AppSlug)
		}
	}

	repo.InstalledAppsCount = len(appNames)

	// Store app names as JSON array
	if len(appNames) > 0 {
		appNamesJSON, err := json.Marshal(appNames)
		if err != nil {
			p.logger.Debug("Failed to marshal app names", "error", err)
		} else {
			appNamesStr := string(appNamesJSON)
			repo.InstalledApps = &appNamesStr
		}
		p.logger.Debug("Found installed apps",
			"repo", repo.FullName,
			"count", len(appNames),
			"apps", appNames)
	}
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

// profileProjectContent checks if the repository has ProjectsV2 associated with it
// Uses the organization-level ProjectsV2 map loaded during discovery
// Note: Classic projects detection removed as those APIs are deprecated
func (p *Profiler) profileProjectContent(ctx context.Context, org, name string, repo *models.Repository) {
	// Check the projects map (loaded at org level via ProjectsV2 API)
	p.projectsMapMu.RLock()
	hasProjectsInMap, inMap := p.projectsMap[name]
	p.projectsMapMu.RUnlock()

	if inMap {
		// We have data from the org-level ProjectsV2 query
		repo.HasProjects = hasProjectsInMap
		if hasProjectsInMap {
			p.logger.Debug("Found ProjectsV2 for repository (from org map)", "repo", repo.FullName)
		}
		return
	}

	// If not in map, default to false (no projects detected)
	// The map should contain all repos with ProjectsV2, so absence means no projects
	repo.HasProjects = false
	p.logger.Debug("No ProjectsV2 found for repository", "repo", repo.FullName)
}

// checkWikiHasContent checks if a wiki repository exists and has content
// Uses git ls-remote which is fast and doesn't require cloning
func (p *Profiler) checkWikiHasContent(ctx context.Context, wikiURL string) (bool, error) {
	// Validate the wiki URL to prevent command injection
	if err := source.ValidateCloneURL(wikiURL); err != nil {
		return false, fmt.Errorf("invalid wiki URL: %w", err)
	}

	// Add authentication to the URL if we have a token
	// Format: https://TOKEN@github.com/org/repo.wiki.git
	authenticatedURL := wikiURL
	if p.token != "" {
		// Parse and inject the token into the URL
		parsedURL, err := url.Parse(wikiURL)
		if err != nil {
			return false, fmt.Errorf("failed to parse wiki URL: %w", err)
		}

		// Use url.User to properly encode the token
		parsedURL.User = url.User(p.token)
		authenticatedURL = parsedURL.String()
	}

	// Use git ls-remote to check if the wiki repo exists and has refs
	// If the wiki has been initialized, it will have at least a master/main branch
	// #nosec G204 -- URL is validated via ValidateCloneURL and properly constructed using url.Parse
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

// estimateMetadataSize estimates the size of repository metadata for GitHub Enterprise Importer's 40 GiB limit
// This provides a rough estimate to help users determine if they need to use exclusion flags
func (p *Profiler) estimateMetadataSize(ctx context.Context, org, name string, repo *models.Repository) error {
	// Estimation constants (based on typical GitHub metadata sizes)
	const (
		avgIssueSize      = 5 * 1024         // 5 KB per issue (includes comments)
		avgPRSize         = 10 * 1024        // 10 KB per PR (includes reviews, comments)
		attachmentPercent = 0.1              // Estimate 10% of issue/PR data is attachments
		metadataOverhead  = 50 * 1024 * 1024 // 50 MB for other metadata (collaborators, labels, etc.)
	)

	var totalEstimate int64 = metadataOverhead

	// Estimate issue and PR data
	issueEstimate := int64(repo.IssueCount) * avgIssueSize
	prEstimate := int64(repo.PullRequestCount) * avgPRSize
	totalEstimate += issueEstimate + prEstimate

	// Estimate attachments (10% of issue/PR data)
	attachmentEstimate := int64(float64(issueEstimate+prEstimate) * attachmentPercent)
	totalEstimate += attachmentEstimate

	// Get actual release sizes (most accurate component)
	releaseSize, releaseDetails, err := p.getReleasesWithAssets(ctx, org, name, repo)
	if err != nil {
		p.logger.Debug("Failed to get release sizes, using estimate",
			"repo", repo.FullName,
			"error", err)
		// Fallback: estimate ~1 MB per release
		releaseSize = int64(repo.ReleaseCount) * 1024 * 1024
	}
	totalEstimate += releaseSize

	// Store the estimate
	repo.EstimatedMetadataSize = &totalEstimate

	// Create detailed breakdown in JSON
	details := fmt.Sprintf(`{"issues_estimate_bytes":%d,"prs_estimate_bytes":%d,"attachments_estimate_bytes":%d,"releases_bytes":%d,"overhead_bytes":%d,"total_bytes":%d,"releases":%s}`,
		issueEstimate,
		prEstimate,
		attachmentEstimate,
		releaseSize,
		metadataOverhead,
		totalEstimate,
		releaseDetails)
	repo.MetadataSizeDetails = &details

	// Log if estimate is large (approaching 40 GiB limit)
	estimateGB := float64(totalEstimate) / (1024 * 1024 * 1024)
	if estimateGB > 35 {
		p.logger.Warn("Repository metadata estimate approaching GitHub's 40 GiB limit",
			"repo", repo.FullName,
			"estimated_gb", estimateGB,
			"consider_excluding_releases", releaseSize > 10*1024*1024*1024) // >10 GB in releases
	} else if estimateGB > 1 {
		p.logger.Info("Repository metadata size estimated",
			"repo", repo.FullName,
			"estimated_gb", estimateGB)
	}

	// Calculate and store complexity score
	complexity, breakdown := p.CalculateComplexity(repo)
	repo.ComplexityScore = &complexity

	// Serialize complexity breakdown to JSON for storage
	if err := repo.SetComplexityBreakdown(breakdown); err != nil {
		p.logger.Warn("Failed to serialize complexity breakdown",
			"repo", repo.FullName,
			"error", err)
	}

	p.logger.Debug("GitHub repository complexity calculated",
		"repo", repo.FullName,
		"complexity", complexity)

	return nil
}

// getReleasesWithAssets fetches releases and calculates total size of release assets
// Returns total size in bytes and JSON array of release details
func (p *Profiler) getReleasesWithAssets(ctx context.Context, org, name string, repo *models.Repository) (int64, string, error) {
	// List all releases
	opts := &ghapi.ListOptions{PerPage: 100}
	var allReleases []*ghapi.RepositoryRelease

	for {
		releases, resp, err := p.client.REST().Repositories.ListReleases(ctx, org, name, opts)
		if err != nil {
			return 0, "[]", fmt.Errorf("failed to list releases: %w", err)
		}

		allReleases = append(allReleases, releases...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Calculate total size and build details
	var totalSize int64
	type releaseInfo struct {
		TagName    string `json:"tag_name"`
		AssetCount int    `json:"asset_count"`
		AssetSize  int64  `json:"asset_size_bytes"`
	}

	releaseInfos := make([]releaseInfo, 0, len(allReleases))

	for _, release := range allReleases {
		var releaseAssetSize int64
		for _, asset := range release.Assets {
			if asset.Size != nil {
				releaseAssetSize += int64(*asset.Size)
			}
		}

		if releaseAssetSize > 0 || len(release.Assets) > 0 {
			releaseInfos = append(releaseInfos, releaseInfo{
				TagName:    release.GetTagName(),
				AssetCount: len(release.Assets),
				AssetSize:  releaseAssetSize,
			})
		}

		totalSize += releaseAssetSize

		// Also count the release metadata itself (~10 KB per release for description, notes, etc.)
		totalSize += 10 * 1024
	}

	// Convert to JSON
	var detailsJSON string
	if len(releaseInfos) > 0 {
		// Only include releases with assets in the details
		releaseBytes, err := json.Marshal(releaseInfos)
		if err != nil {
			p.logger.Warn("Failed to marshal release details", "error", err)
			detailsJSON = "[]"
		} else {
			detailsJSON = string(releaseBytes)
		}
	} else {
		detailsJSON = "[]"
	}

	p.logger.Debug("Release assets calculated",
		"repo", repo.FullName,
		"release_count", len(allReleases),
		"releases_with_assets", len(releaseInfos),
		"total_size_mb", totalSize/(1024*1024))

	return totalSize, detailsJSON, nil
}
