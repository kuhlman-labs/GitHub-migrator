package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v75/github"
	"github.com/shurcooL/githubv4"
)

// Permission constants for team repository access levels
const (
	PermissionAdmin    = "admin"
	PermissionMaintain = "maintain"
	PermissionPush     = "push"
	PermissionTriage   = "triage"
	PermissionPull     = "pull"
)

// TeamInfo represents basic team information returned from discovery
type TeamInfo struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	Privacy     string
}

// TeamRepository represents a repository associated with a team
type TeamRepository struct {
	FullName   string // org/repo format
	Permission string // pull, push, admin, maintain, triage
}

// TeamMember represents a member of a GitHub team
type TeamMember struct {
	Login string // GitHub username
	Role  string // member or maintainer
}

// ListOrganizationTeams lists all teams for an organization using REST API with pagination
func (c *Client) ListOrganizationTeams(ctx context.Context, org string) ([]*TeamInfo, error) {
	c.logger.Info("Listing teams for organization", "org", org)

	var allTeams []*TeamInfo
	opts := &github.ListOptions{PerPage: 100}

	for {
		var teams []*github.Team
		var resp *github.Response

		err := c.retryer.Do(ctx, "ListOrganizationTeams", func(ctx context.Context) error {
			var err error
			teams, resp, err = c.rest.Teams.ListTeams(ctx, org, opts)
			if err != nil {
				return WrapError(err, "ListTeams", c.baseURL)
			}

			// Update rate limits
			if resp != nil && resp.Rate.Limit > 0 {
				c.rateLimiter.UpdateLimits(
					resp.Rate.Remaining,
					resp.Rate.Limit,
					resp.Rate.Reset.Time,
				)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		for _, team := range teams {
			info := &TeamInfo{
				ID:      team.GetID(),
				Slug:    team.GetSlug(),
				Name:    team.GetName(),
				Privacy: team.GetPrivacy(),
			}
			if team.Description != nil {
				info.Description = *team.Description
			}
			allTeams = append(allTeams, info)
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Info("Team listing complete",
		"org", org,
		"total_teams", len(allTeams))

	return allTeams, nil
}

// ListTeamRepositories lists all repositories that a team has access to
func (c *Client) ListTeamRepositories(ctx context.Context, org, teamSlug string) ([]*TeamRepository, error) {
	c.logger.Debug("Listing repositories for team", "org", org, "team", teamSlug)

	var allRepos []*TeamRepository
	opts := &github.ListOptions{PerPage: 100}

	for {
		var repos []*github.Repository
		var resp *github.Response

		err := c.retryer.Do(ctx, "ListTeamRepositories", func(ctx context.Context) error {
			var err error
			repos, resp, err = c.rest.Teams.ListTeamReposBySlug(ctx, org, teamSlug, opts)
			if err != nil {
				return WrapError(err, "ListTeamReposBySlug", c.baseURL)
			}

			// Update rate limits
			if resp != nil && resp.Rate.Limit > 0 {
				c.rateLimiter.UpdateLimits(
					resp.Rate.Remaining,
					resp.Rate.Limit,
					resp.Rate.Reset.Time,
				)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		for _, repo := range repos {
			teamRepo := &TeamRepository{
				FullName: repo.GetFullName(),
			}
			// Extract permission from the repo's permissions map
			if repo.Permissions != nil {
				if repo.Permissions[PermissionAdmin] {
					teamRepo.Permission = PermissionAdmin
				} else if repo.Permissions[PermissionMaintain] {
					teamRepo.Permission = PermissionMaintain
				} else if repo.Permissions[PermissionPush] {
					teamRepo.Permission = PermissionPush
				} else if repo.Permissions[PermissionTriage] {
					teamRepo.Permission = PermissionTriage
				} else {
					teamRepo.Permission = PermissionPull
				}
			} else {
				teamRepo.Permission = PermissionPull
			}
			allRepos = append(allRepos, teamRepo)
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Debug("Team repository listing complete",
		"org", org,
		"team", teamSlug,
		"total_repos", len(allRepos))

	return allRepos, nil
}

// ListTeamMembers lists all members of a team
func (c *Client) ListTeamMembers(ctx context.Context, org, teamSlug string) ([]*TeamMember, error) {
	c.logger.Debug("Listing members for team", "org", org, "team", teamSlug)

	var allMembers []*TeamMember
	opts := &github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		var members []*github.User
		var resp *github.Response

		err := c.retryer.Do(ctx, "ListTeamMembers", func(ctx context.Context) error {
			var err error
			members, resp, err = c.rest.Teams.ListTeamMembersBySlug(ctx, org, teamSlug, opts)
			if err != nil {
				return WrapError(err, "ListTeamMembersBySlug", c.baseURL)
			}

			// Update rate limits
			if resp != nil && resp.Rate.Limit > 0 {
				c.rateLimiter.UpdateLimits(
					resp.Rate.Remaining,
					resp.Rate.Limit,
					resp.Rate.Reset.Time,
				)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		for _, member := range members {
			// Get the member's role in the team
			role := "member"
			membership, _, err := c.rest.Teams.GetTeamMembershipBySlug(ctx, org, teamSlug, member.GetLogin())
			if err == nil && membership != nil {
				role = membership.GetRole() // "member" or "maintainer"
			}

			allMembers = append(allMembers, &TeamMember{
				Login: member.GetLogin(),
				Role:  role,
			})
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Debug("Team member listing complete",
		"org", org,
		"team", teamSlug,
		"total_members", len(allMembers))

	return allMembers, nil
}

// ListTeamMembersGraphQL lists all members of a team using GraphQL
// This is more efficient than the REST API as it fetches member roles in a single query
// instead of making N+1 API calls (one per member to get their role)
func (c *Client) ListTeamMembersGraphQL(ctx context.Context, org, teamSlug string) ([]*TeamMember, error) {
	c.logger.Debug("Listing members for team via GraphQL", "org", org, "team", teamSlug)

	var allMembers []*TeamMember
	var cursor *githubv4.String

	// GraphQL query for team members with roles
	var query struct {
		Organization struct {
			Team struct {
				Members struct {
					PageInfo struct {
						HasNextPage githubv4.Boolean
						EndCursor   githubv4.String
					}
					Edges []struct {
						Role githubv4.String
						Node struct {
							Login githubv4.String
						}
					}
				} `graphql:"members(first: 100, after: $cursor)"`
			} `graphql:"team(slug: $slug)"`
		} `graphql:"organization(login: $org)"`
	}

	variables := map[string]any{
		"org":    githubv4.String(org),
		"slug":   githubv4.String(teamSlug),
		"cursor": cursor,
	}

	for {
		variables["cursor"] = cursor

		err := c.retryer.Do(ctx, "ListTeamMembersGraphQL", func(ctx context.Context) error {
			return c.graphql.Query(ctx, &query, variables)
		})

		if err != nil {
			// Fall back to REST API if GraphQL fails (e.g., on GHES without GraphQL team support)
			c.logger.Debug("GraphQL query failed for team members, falling back to REST",
				"org", org,
				"team", teamSlug,
				"error", err)
			return c.ListTeamMembers(ctx, org, teamSlug)
		}

		for _, edge := range query.Organization.Team.Members.Edges {
			allMembers = append(allMembers, &TeamMember{
				Login: string(edge.Node.Login),
				Role:  strings.ToLower(string(edge.Role)), // GraphQL returns "MAINTAINER" or "MEMBER"
			})
		}

		if !bool(query.Organization.Team.Members.PageInfo.HasNextPage) {
			break
		}
		cursor = newString(query.Organization.Team.Members.PageInfo.EndCursor)
	}

	c.logger.Debug("Team member listing via GraphQL complete",
		"org", org,
		"team", teamSlug,
		"total_members", len(allMembers))

	return allMembers, nil
}

// GetTeamBySlug retrieves a team by organization and slug
// Returns nil, nil if the team doesn't exist (404)
func (c *Client) GetTeamBySlug(ctx context.Context, org, slug string) (*TeamInfo, error) {
	c.logger.Debug("Getting team by slug", "org", org, "slug", slug)

	var team *github.Team
	err := c.retryer.Do(ctx, "GetTeamBySlug", func(ctx context.Context) error {
		var err error
		team, _, err = c.rest.Teams.GetTeamBySlug(ctx, org, slug)
		if err != nil {
			// Check if it's a 404 (team not found)
			if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == 404 {
				return nil // Return nil error for 404, we'll check team == nil
			}
			return WrapError(err, "GetTeamBySlug", c.baseURL)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if team == nil {
		c.logger.Debug("Team not found", "org", org, "slug", slug)
		return nil, nil
	}

	info := &TeamInfo{
		ID:      team.GetID(),
		Slug:    team.GetSlug(),
		Name:    team.GetName(),
		Privacy: team.GetPrivacy(),
	}
	if team.Description != nil {
		info.Description = *team.Description
	}

	c.logger.Debug("Team found", "org", org, "slug", slug, "team_id", info.ID)
	return info, nil
}

// CreateTeamInput contains the parameters for creating a team
type CreateTeamInput struct {
	Name        string  // Required: team name
	Description *string // Optional: team description
	Privacy     string  // "secret" or "closed" (default: "secret")
	ParentTeam  *int64  // Optional: parent team ID for nested teams
}

// CreateTeam creates a new team in the organization
// Teams are created WITHOUT members by default to support EMU/IdP-managed environments
func (c *Client) CreateTeam(ctx context.Context, org string, input CreateTeamInput) (*TeamInfo, error) {
	c.logger.Info("Creating team", "org", org, "name", input.Name)

	privacy := input.Privacy
	if privacy == "" {
		privacy = "secret"
	}

	newTeam := &github.NewTeam{
		Name:        input.Name,
		Description: input.Description,
		Privacy:     &privacy,
	}

	if input.ParentTeam != nil {
		newTeam.ParentTeamID = input.ParentTeam
	}

	var team *github.Team
	err := c.retryer.Do(ctx, "CreateTeam", func(ctx context.Context) error {
		var err error
		team, _, err = c.rest.Teams.CreateTeam(ctx, org, *newTeam)
		if err != nil {
			return WrapError(err, "CreateTeam", c.baseURL)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	info := &TeamInfo{
		ID:      team.GetID(),
		Slug:    team.GetSlug(),
		Name:    team.GetName(),
		Privacy: team.GetPrivacy(),
	}
	if team.Description != nil {
		info.Description = *team.Description
	}

	c.logger.Info("Team created successfully",
		"org", org,
		"team_slug", info.Slug,
		"team_id", info.ID)

	return info, nil
}

// AddTeamRepoPermission adds or updates a repository's permission for a team
// Permission must be one of: pull, triage, push, maintain, admin
func (c *Client) AddTeamRepoPermission(ctx context.Context, org, teamSlug, repoOwner, repoName, permission string) error {
	c.logger.Debug("Adding team repo permission",
		"org", org,
		"team", teamSlug,
		"repo", repoOwner+"/"+repoName,
		"permission", permission)

	// Validate permission
	validPermissions := map[string]bool{
		PermissionPull:     true,
		PermissionTriage:   true,
		PermissionPush:     true,
		PermissionMaintain: true,
		PermissionAdmin:    true,
	}
	if !validPermissions[permission] {
		return fmt.Errorf("invalid permission %q, must be one of: %s, %s, %s, %s, %s",
			permission, PermissionPull, PermissionTriage, PermissionPush, PermissionMaintain, PermissionAdmin)
	}

	opts := &github.TeamAddTeamRepoOptions{
		Permission: permission,
	}

	err := c.retryer.Do(ctx, "AddTeamRepoPermission", func(ctx context.Context) error {
		_, err := c.rest.Teams.AddTeamRepoBySlug(ctx, org, teamSlug, repoOwner, repoName, opts)
		if err != nil {
			return WrapError(err, "AddTeamRepoBySlug", c.baseURL)
		}
		return nil
	})

	if err != nil {
		return err
	}

	c.logger.Debug("Team repo permission added successfully",
		"org", org,
		"team", teamSlug,
		"repo", repoOwner+"/"+repoName,
		"permission", permission)

	return nil
}

// RemoveTeamMembership removes a user from a team
// This is used to remove the PAT owner from a team after creation when using PAT authentication,
// since GitHub automatically adds the PAT owner as a maintainer when creating a team with a PAT.
func (c *Client) RemoveTeamMembership(ctx context.Context, org, teamSlug, username string) error {
	c.logger.Debug("Removing team membership",
		"org", org,
		"team", teamSlug,
		"username", username)

	err := c.retryer.Do(ctx, "RemoveTeamMembership", func(ctx context.Context) error {
		_, err := c.rest.Teams.RemoveTeamMembershipBySlug(ctx, org, teamSlug, username)
		if err != nil {
			return WrapError(err, "RemoveTeamMembershipBySlug", c.baseURL)
		}
		return nil
	})

	if err != nil {
		return err
	}

	c.logger.Debug("Team membership removed successfully",
		"org", org,
		"team", teamSlug,
		"username", username)

	return nil
}
