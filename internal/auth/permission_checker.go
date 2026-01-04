package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// PermissionChecker handles repository-level permission checks
type PermissionChecker struct {
	client     *github.Client
	authorizer *Authorizer
	config     *config.AuthConfig
	logger     *slog.Logger
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(client *github.Client, cfg *config.AuthConfig, logger *slog.Logger, githubBaseURL string) *PermissionChecker {
	return &PermissionChecker{
		client:     client,
		authorizer: NewAuthorizer(cfg, logger, githubBaseURL),
		config:     cfg,
		logger:     logger,
	}
}

// HasFullAccess checks if a user has full access to all repositories
// Returns true if user is:
// - Enterprise admin (when an enterprise slug is configured)
// - Member of a privileged team
func (p *PermissionChecker) HasFullAccess(ctx context.Context, user *GitHubUser, token string) (bool, error) {
	// Check if user is enterprise admin (whenever an enterprise is configured)
	// Note: This is separate from RequireEnterpriseAdmin which controls application access
	// Enterprise admins always get full migration privileges when an enterprise is configured
	if p.config.AuthorizationRules.RequireEnterpriseSlug != "" {
		isAdmin, err := p.authorizer.CheckEnterpriseAdmin(ctx, user.Login, p.config.AuthorizationRules.RequireEnterpriseSlug, token)
		if err != nil {
			p.logger.Warn("Failed to check enterprise admin status", "user", user.Login, "error", err)
		} else if isAdmin {
			p.logger.Info("User has full migration access as enterprise admin", "user", user.Login)
			return true, nil
		}
	}

	// Check if user is member of a migration admin team
	if len(p.config.AuthorizationRules.MigrationAdminTeams) > 0 {
		isMember, err := p.authorizer.CheckTeamMembership(ctx, user.Login, p.config.AuthorizationRules.MigrationAdminTeams, token)
		if err != nil {
			p.logger.Warn("Failed to check migration admin team membership", "user", user.Login, "error", err)
		} else if isMember {
			p.logger.Debug("User has full access as migration admin team member", "user", user.Login)
			return true, nil
		}
	}

	return false, nil
}

// HasRepoAccess checks if a user has admin permission on a specific repository
// Returns true if user:
// - Has full access (enterprise admin or privileged team member)
// - Is an org admin in the repository's organization
// - Has admin permission on the specific repository
func (p *PermissionChecker) HasRepoAccess(ctx context.Context, user *GitHubUser, token string, fullName string) (bool, error) {
	// Check for full access first
	hasFullAccess, err := p.HasFullAccess(ctx, user, token)
	if err != nil {
		return false, fmt.Errorf("failed to check full access: %w", err)
	}
	if hasFullAccess {
		return true, nil
	}

	// Parse org and repo name
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid repository full name: %s", fullName)
	}
	org, repo := parts[0], parts[1]

	// Check if user is an org admin
	isOrgAdmin, err := p.authorizer.IsOrgAdmin(ctx, user.Login, org, token)
	if err != nil {
		p.logger.Warn("Failed to check org admin status", "user", user.Login, "org", org, "error", err)
	} else if isOrgAdmin {
		p.logger.Debug("User has access as org admin", "user", user.Login, "org", org)
		return true, nil
	}

	// Check if user has admin permission on the specific repository
	hasRepoAdmin, err := p.authorizer.HasRepoAdminPermission(ctx, user.Login, org, repo, token)
	if err != nil {
		return false, fmt.Errorf("failed to check repository admin permission: %w", err)
	}

	if hasRepoAdmin {
		p.logger.Debug("User has access as repository admin", "user", user.Login, "repo", fullName)
	} else {
		p.logger.Debug("User does not have admin access to repository", "user", user.Login, "repo", fullName)
	}

	return hasRepoAdmin, nil
}

// FilterRepositoriesByAccess filters a list of repositories to only those the user can migrate
// This is more efficient than checking each repository individually when processing a large list
func (p *PermissionChecker) FilterRepositoriesByAccess(ctx context.Context, user *GitHubUser, token string, repos []*models.Repository) ([]*models.Repository, error) {
	// Check for full access first
	hasFullAccess, err := p.HasFullAccess(ctx, user, token)
	if err != nil {
		return nil, fmt.Errorf("failed to check full access: %w", err)
	}
	if hasFullAccess {
		p.logger.Debug("User has full access, returning all repositories", "user", user.Login, "count", len(repos))
		return repos, nil
	}

	// Get organizations where user is an admin
	adminOrgs, err := p.GetUserOrganizationsWithAdminRole(ctx, user, token)
	if err != nil {
		p.logger.Warn("Failed to get user's admin organizations", "user", user.Login, "error", err)
		adminOrgs = make(map[string]bool) // Continue with empty map
	}

	var filteredRepos []*models.Repository
	for _, repo := range repos {
		org := repo.Organization()

		// If user is org admin, include the repo
		if adminOrgs[org] {
			filteredRepos = append(filteredRepos, repo)
			continue
		}

		// Otherwise, check individual repository permission
		hasAccess, err := p.HasRepoAccess(ctx, user, token, repo.FullName)
		if err != nil {
			p.logger.Warn("Failed to check repository access", "user", user.Login, "repo", repo.FullName, "error", err)
			continue
		}
		if hasAccess {
			filteredRepos = append(filteredRepos, repo)
		}
	}

	p.logger.Info("Filtered repositories by user access",
		"user", user.Login,
		"total_repos", len(repos),
		"accessible_repos", len(filteredRepos))

	return filteredRepos, nil
}

// GetUserOrganizationsWithAdminRole returns a map of organizations where the user has admin role
func (p *PermissionChecker) GetUserOrganizationsWithAdminRole(ctx context.Context, user *GitHubUser, token string) (map[string]bool, error) {
	adminOrgs := make(map[string]bool)

	// Get all organizations the user is a member of
	orgs, err := p.authorizer.GetUserOrganizations(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	// Check admin role for each organization
	for _, org := range orgs {
		isAdmin, err := p.authorizer.IsOrgAdmin(ctx, user.Login, org, token)
		if err != nil {
			p.logger.Warn("Failed to check org admin status", "user", user.Login, "org", org, "error", err)
			continue
		}
		if isAdmin {
			adminOrgs[org] = true
		}
	}

	p.logger.Debug("Found user's admin organizations",
		"user", user.Login,
		"admin_org_count", len(adminOrgs))

	return adminOrgs, nil
}

// ValidateRepositoryAccess validates that a user has access to all specified repositories
// Returns an error with details about inaccessible repositories if validation fails
func (p *PermissionChecker) ValidateRepositoryAccess(ctx context.Context, user *GitHubUser, token string, repoFullNames []string) error {
	// Check for full access first
	hasFullAccess, err := p.HasFullAccess(ctx, user, token)
	if err != nil {
		return fmt.Errorf("failed to check full access: %w", err)
	}
	if hasFullAccess {
		return nil // User has full access
	}

	// Extract unique organizations from the repositories
	// Only check admin status for these specific orgs (optimization)
	uniqueOrgs := make(map[string]bool)
	for _, fullName := range repoFullNames {
		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) == 2 {
			uniqueOrgs[parts[0]] = true
		}
	}

	// Check admin status only for the specific organizations of these repositories
	adminOrgs := make(map[string]bool)
	for org := range uniqueOrgs {
		isAdmin, err := p.authorizer.IsOrgAdmin(ctx, user.Login, org, token)
		if err != nil {
			p.logger.Debug("Failed to check org admin status (continuing with repo check)",
				"user", user.Login,
				"org", org,
				"error", err)
			continue
		}
		if isAdmin {
			adminOrgs[org] = true
		}
	}

	p.logger.Debug("Checked admin status for relevant organizations",
		"user", user.Login,
		"total_orgs_checked", len(uniqueOrgs),
		"admin_org_count", len(adminOrgs))

	var inaccessibleRepos []string

	for _, fullName := range repoFullNames {
		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) != 2 {
			inaccessibleRepos = append(inaccessibleRepos, fullName)
			continue
		}
		org, repo := parts[0], parts[1]

		// If user is org admin, they have access
		if adminOrgs[org] {
			continue
		}

		// We already checked org admin above, so skip straight to repo-level check
		// (avoids redundant IsOrgAdmin API call in HasRepoAccess)
		hasRepoAdmin, err := p.authorizer.HasRepoAdminPermission(ctx, user.Login, org, repo, token)
		if err != nil {
			p.logger.Warn("Failed to check repository admin permission",
				"user", user.Login,
				"repo", fullName,
				"error", err)
			inaccessibleRepos = append(inaccessibleRepos, fullName)
			continue
		}
		if !hasRepoAdmin {
			inaccessibleRepos = append(inaccessibleRepos, fullName)
		}
	}

	if len(inaccessibleRepos) > 0 {
		return fmt.Errorf("you don't have admin access to the following repositories: %s", strings.Join(inaccessibleRepos, ", "))
	}

	return nil
}
