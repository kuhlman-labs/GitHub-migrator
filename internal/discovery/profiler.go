package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
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

	// Check for GitHub Actions workflows
	workflows, _, err := p.client.REST().Actions.ListWorkflows(ctx, org, name, nil)
	if err == nil && workflows != nil {
		repo.HasActions = workflows.GetTotalCount() > 0
	} else {
		p.logger.Debug("Failed to get workflows", "error", err)
	}

	// Count branch protections
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

	// Count environments
	environments, _, err := p.client.REST().Repositories.ListEnvironments(ctx, org, name, nil)
	if err == nil && environments != nil {
		repo.EnvironmentCount = environments.GetTotalCount()
	} else {
		p.logger.Debug("Failed to get environments", "error", err)
	}

	// Count webhooks
	hooks, _, err := p.client.REST().Repositories.ListHooks(ctx, org, name, nil)
	if err == nil {
		repo.WebhookCount = len(hooks)
	} else {
		p.logger.Debug("Failed to get webhooks", "error", err)
	}

	// Get contributors
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

	p.logger.Info("GitHub features profiled",
		"repo", repo.FullName,
		"has_actions", repo.HasActions,
		"has_wiki", repo.HasWiki,
		"has_pages", repo.HasPages,
		"has_discussions", repo.HasDiscussions,
		"contributors", repo.ContributorCount)

	return nil
}
