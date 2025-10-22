package discovery

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	ghapi "github.com/google/go-github/v75/github"
)

// Profiler profiles GitHub-specific features via API
type Profiler struct {
	client *github.Client
	logger *slog.Logger
	token  string // GitHub token for authenticated git operations
}

// NewProfiler creates a new GitHub features profiler
func NewProfiler(client *github.Client, logger *slog.Logger) *Profiler {
	return &Profiler{
		client: client,
		logger: logger,
		token:  client.Token(),
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
	p.profileWorkflows(ctx, org, name, repo)
	p.profileBranchProtections(ctx, org, name, repo)
	p.profileEnvironments(ctx, org, name, repo)
	p.profileWebhooks(ctx, org, name, repo)
	p.profileContributors(ctx, org, name, repo)
	p.profileTags(ctx, org, name, repo)
	p.profilePackages(ctx, org, name, repo)

	// Check if wiki actually has content (not just enabled)
	p.profileWikiContent(ctx, repo)

	// Get issue counts for verification
	if err := p.countIssuesAndPRs(ctx, org, name, repo); err != nil {
		p.logger.Debug("Failed to get issue/PR counts", "error", err)
	}

	p.logger.Info("GitHub features profiled",
		"repo", repo.FullName,
		"has_actions", repo.HasActions,
		"has_wiki", repo.HasWiki,
		"has_pages", repo.HasPages,
		"has_discussions", repo.HasDiscussions,
		"has_packages", repo.HasPackages,
		"contributors", repo.ContributorCount,
		"issues", repo.IssueCount,
		"prs", repo.PullRequestCount,
		"tags", repo.TagCount)

	return nil
}

// profileWorkflows checks for GitHub Actions workflows
func (p *Profiler) profileWorkflows(ctx context.Context, org, name string, repo *models.Repository) {
	workflows, _, err := p.client.REST().Actions.ListWorkflows(ctx, org, name, nil)
	if err == nil && workflows != nil {
		repo.HasActions = workflows.GetTotalCount() > 0
	} else {
		p.logger.Debug("Failed to get workflows", "error", err)
	}
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

// profileContributors gets contributor information
func (p *Profiler) profileContributors(ctx context.Context, org, name string, repo *models.Repository) {
	contributors, _, err := p.client.REST().Repositories.ListContributors(ctx, org, name, nil)
	if err == nil {
		repo.ContributorCount = len(contributors)

		// Store top contributors (up to 5)
		topContributors := make([]string, 0, 5)
		for i, contributor := range contributors {
			if i >= 5 {
				break
			}
			topContributors = append(topContributors, contributor.GetLogin())
		}
		topContribStr := strings.Join(topContributors, ",")
		repo.TopContributors = &topContribStr
	} else {
		p.logger.Debug("Failed to get contributors", "error", err)
	}
}

// profileTags counts repository tags
func (p *Profiler) profileTags(ctx context.Context, org, name string, repo *models.Repository) {
	tags, _, err := p.client.REST().Repositories.ListTags(ctx, org, name, nil)
	if err == nil {
		repo.TagCount = len(tags)
	} else {
		p.logger.Debug("Failed to get tags", "error", err)
	}
}

// profilePackages checks for GitHub Packages
func (p *Profiler) profilePackages(ctx context.Context, org, name string, repo *models.Repository) {
	// List packages for the repository
	// Note: The Packages API requires specific permissions and may not work for all repos
	packages, _, err := p.client.REST().Organizations.ListPackages(ctx, org, nil)
	if err == nil && packages != nil {
		// Check if any packages belong to this repository
		for _, pkg := range packages {
			if pkg.Repository != nil && pkg.Repository.GetName() == name {
				repo.HasPackages = true
				p.logger.Debug("Found packages for repository", "repo", repo.FullName)
				return
			}
		}
	} else {
		p.logger.Debug("Failed to list packages (may require additional permissions)",
			"repo", repo.FullName,
			"error", err)
	}

	// If we couldn't detect packages, default to false
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

		// Limit pagination to avoid rate limit issues (max 10 pages = 1000 issues)
		if allIssuesOpts.ListOptions.Page > 10 {
			p.logger.Debug("Reached pagination limit for issues, counts may be underestimated",
				"repo", repo.FullName)
			break
		}
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

		// Limit pagination to avoid rate limit issues (max 10 pages = 1000 PRs)
		if allPRsOpts.ListOptions.Page > 10 {
			p.logger.Debug("Reached pagination limit for PRs, counts may be underestimated",
				"repo", repo.FullName)
			break
		}
	}

	repo.PullRequestCount = totalPRs
	repo.OpenPRCount = openPRs

	return nil
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
