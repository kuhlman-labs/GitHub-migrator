package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	ghapi "github.com/google/go-github/v75/github"
)

// Profiler profiles GitHub-specific features via API
type Profiler struct {
	client *github.Client
	logger *slog.Logger
}

// NewProfiler creates a new GitHub features profiler
func NewProfiler(client *github.Client, logger *slog.Logger) *Profiler {
	return &Profiler{
		client: client,
		logger: logger,
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
