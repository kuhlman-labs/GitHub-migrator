package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v75/github"
	"github.com/shurcooL/githubv4"
)

// ListEnterpriseOrganizations lists all organizations in an enterprise using GraphQL
func (c *Client) ListEnterpriseOrganizations(ctx context.Context, enterpriseSlug string) ([]string, error) {
	c.logger.Info("Listing organizations for enterprise", "enterprise", enterpriseSlug)

	var allOrgs []string
	var endCursor *githubv4.String

	// GraphQL query for enterprise organizations
	var query struct {
		Enterprise struct {
			Organizations struct {
				Nodes []struct {
					Login githubv4.String
				}
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"organizations(first: 100, after: $cursor)"`
		} `graphql:"enterprise(slug: $slug)"`
	}

	for {
		variables := map[string]interface{}{
			"slug":   githubv4.String(enterpriseSlug),
			"cursor": endCursor,
		}

		err := c.QueryWithRetry(ctx, "ListEnterpriseOrganizations", &query, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to list enterprise organizations: %w", err)
		}

		// Collect organization logins
		for _, org := range query.Enterprise.Organizations.Nodes {
			allOrgs = append(allOrgs, string(org.Login))
		}

		if !query.Enterprise.Organizations.PageInfo.HasNextPage {
			break
		}
		endCursor = &query.Enterprise.Organizations.PageInfo.EndCursor
	}

	c.logger.Info("Enterprise organizations listed",
		"enterprise", enterpriseSlug,
		"total_orgs", len(allOrgs))

	return allOrgs, nil
}

// EnterpriseOrgInfo contains organization info including repo count
type EnterpriseOrgInfo struct {
	Login     string
	RepoCount int
}

// ListEnterpriseOrganizationsWithCounts lists all organizations in an enterprise with their repo counts
// This allows getting accurate total repo counts upfront for progress tracking
func (c *Client) ListEnterpriseOrganizationsWithCounts(ctx context.Context, enterpriseSlug string) ([]EnterpriseOrgInfo, error) {
	c.logger.Info("Listing organizations with repo counts for enterprise", "enterprise", enterpriseSlug)

	var allOrgs []EnterpriseOrgInfo
	var endCursor *githubv4.String

	// GraphQL query for enterprise organizations with repo counts
	var query struct {
		Enterprise struct {
			Organizations struct {
				Nodes []struct {
					Login        githubv4.String
					Repositories struct {
						TotalCount githubv4.Int
					} `graphql:"repositories(first: 1)"`
				}
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"organizations(first: 100, after: $cursor)"`
		} `graphql:"enterprise(slug: $slug)"`
	}

	for {
		variables := map[string]interface{}{
			"slug":   githubv4.String(enterpriseSlug),
			"cursor": endCursor,
		}

		err := c.QueryWithRetry(ctx, "ListEnterpriseOrganizationsWithCounts", &query, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to list enterprise organizations with counts: %w", err)
		}

		// Collect organization info
		for _, org := range query.Enterprise.Organizations.Nodes {
			allOrgs = append(allOrgs, EnterpriseOrgInfo{
				Login:     string(org.Login),
				RepoCount: int(org.Repositories.TotalCount),
			})
		}

		if !query.Enterprise.Organizations.PageInfo.HasNextPage {
			break
		}
		endCursor = &query.Enterprise.Organizations.PageInfo.EndCursor
	}

	totalRepos := 0
	for _, org := range allOrgs {
		totalRepos += org.RepoCount
	}

	c.logger.Info("Enterprise organizations with counts listed",
		"enterprise", enterpriseSlug,
		"total_orgs", len(allOrgs),
		"total_repos", totalRepos)

	return allOrgs, nil
}

// GetOrganizationRepoCount returns the total number of repositories in an organization
// This is a lightweight query that only fetches the count, not the actual repos
func (c *Client) GetOrganizationRepoCount(ctx context.Context, org string) (int, error) {
	c.logger.Debug("Getting repository count for organization", "org", org)

	var query struct {
		Organization struct {
			Repositories struct {
				TotalCount githubv4.Int
			} `graphql:"repositories(first: 1)"`
		} `graphql:"organization(login: $owner)"`
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(org),
	}

	err := c.QueryWithRetry(ctx, "GetOrganizationRepoCount", &query, variables)
	if err != nil {
		return 0, fmt.Errorf("failed to get organization repo count: %w", err)
	}

	count := int(query.Organization.Repositories.TotalCount)
	c.logger.Debug("Organization repository count retrieved", "org", org, "count", count)

	return count, nil
}

// OrganizationProject represents a ProjectsV2 at the organization level
type OrganizationProject struct {
	Title        string
	Repositories []string // Repository names (not full names, just the repo name)
}

// ListOrganizationProjects fetches all ProjectsV2 for an organization using GraphQL
// Returns a map of repository names to a boolean indicating if they have projects
func (c *Client) ListOrganizationProjects(ctx context.Context, org string) (map[string]bool, error) {
	c.logger.Info("Listing organization projects (ProjectsV2)", "org", org)

	repoProjectMap := make(map[string]bool)
	var endCursor *githubv4.String

	// GraphQL query for organization ProjectsV2
	var query struct {
		Organization struct {
			Login      githubv4.String
			ProjectsV2 struct {
				TotalCount githubv4.Int
				Nodes      []struct {
					ID           githubv4.String
					Title        githubv4.String
					Repositories struct {
						TotalCount githubv4.Int
						Nodes      []struct {
							Name githubv4.String
						}
						PageInfo struct {
							HasNextPage githubv4.Boolean
							EndCursor   githubv4.String
						}
					} `graphql:"repositories(first: 100)"`
				}
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"projectsV2(first: 100, after: $cursor)"`
		} `graphql:"organization(login: $owner)"`
	}

	// Paginate through all projects
	for {
		variables := map[string]interface{}{
			"owner":  githubv4.String(org),
			"cursor": endCursor,
		}

		err := c.QueryWithRetry(ctx, "ListOrganizationProjects", &query, variables)
		if err != nil {
			// If ProjectsV2 is not available or permission denied, return empty map
			c.logger.Debug("ProjectsV2 not available for organization", "org", org, "error", err)
			return repoProjectMap, nil
		}

		// Build map of repositories that have projects
		for _, project := range query.Organization.ProjectsV2.Nodes {
			// Add repositories from first page
			for _, repo := range project.Repositories.Nodes {
				repoName := string(repo.Name)
				repoProjectMap[repoName] = true
			}

			// If this project has more than 100 repositories, paginate through them
			if project.Repositories.PageInfo.HasNextPage {
				c.logger.Debug("Project has more than 100 repositories, paginating",
					"project", project.Title,
					"total_repos", project.Repositories.TotalCount)

				if err := c.paginateProjectRepositories(ctx, string(project.ID), &project.Repositories.PageInfo.EndCursor, repoProjectMap); err != nil {
					c.logger.Warn("Failed to paginate project repositories",
						"project", project.Title,
						"error", err)
					// Continue with other projects even if one fails
				}
			}
		}

		if !query.Organization.ProjectsV2.PageInfo.HasNextPage {
			break
		}
		endCursor = &query.Organization.ProjectsV2.PageInfo.EndCursor
	}

	c.logger.Info("Organization projects (ProjectsV2) fetched",
		"org", org,
		"total_projects", query.Organization.ProjectsV2.TotalCount,
		"repos_with_projects", len(repoProjectMap))

	return repoProjectMap, nil
}

// paginateProjectRepositories fetches additional repositories for a project beyond the first 100
func (c *Client) paginateProjectRepositories(ctx context.Context, projectID string, startCursor *githubv4.String, repoMap map[string]bool) error {
	var query struct {
		Node struct {
			ProjectV2 struct {
				Repositories struct {
					Nodes []struct {
						Name githubv4.String
					}
					PageInfo struct {
						HasNextPage githubv4.Boolean
						EndCursor   githubv4.String
					}
				} `graphql:"repositories(first: 100, after: $cursor)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	cursor := startCursor
	for cursor != nil {
		variables := map[string]interface{}{
			"id":     githubv4.ID(projectID),
			"cursor": cursor,
		}

		err := c.QueryWithRetry(ctx, "PaginateProjectRepositories", &query, variables)
		if err != nil {
			return err
		}

		// Add repositories from this page
		for _, repo := range query.Node.ProjectV2.Repositories.Nodes {
			repoName := string(repo.Name)
			repoMap[repoName] = true
		}

		// Check if there are more pages
		if !query.Node.ProjectV2.Repositories.PageInfo.HasNextPage {
			break
		}
		cursor = &query.Node.ProjectV2.Repositories.PageInfo.EndCursor
	}

	return nil
}

// OrgMember represents a member of an organization with full details
type OrgMember struct {
	Login     string  `json:"login"`
	ID        int64   `json:"id"`
	Name      *string `json:"name,omitempty"`
	Email     *string `json:"email,omitempty"`
	AvatarURL string  `json:"avatar_url"`
	Role      string  `json:"role"` // "admin" or "member"
}

// ListOrgMembers lists all members of an organization using GraphQL to get full details
func (c *Client) ListOrgMembers(ctx context.Context, org string) ([]*OrgMember, error) {
	c.logger.Debug("Listing organization members", "org", org)

	var allMembers []*OrgMember
	var cursor *githubv4.String

	// GraphQL query for organization members with full details
	var query struct {
		Organization struct {
			MembersWithRole struct {
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
				Edges []struct {
					Role githubv4.String
					Node struct {
						Login      githubv4.String
						Name       githubv4.String
						Email      githubv4.String
						AvatarUrl  githubv4.String
						DatabaseId githubv4.Int
					}
				}
			} `graphql:"membersWithRole(first: 100, after: $cursor)"`
		} `graphql:"organization(login: $org)"`
	}

	variables := map[string]interface{}{
		"org":    githubv4.String(org),
		"cursor": cursor,
	}

	for {
		variables["cursor"] = cursor

		err := c.retryer.Do(ctx, "ListOrgMembers", func(ctx context.Context) error {
			return c.graphql.Query(ctx, &query, variables)
		})

		if err != nil {
			// Fall back to REST API if GraphQL fails (e.g., on GHES without GraphQL)
			c.logger.Debug("GraphQL query failed, falling back to REST", "error", err)
			return c.listOrgMembersREST(ctx, org)
		}

		for _, edge := range query.Organization.MembersWithRole.Edges {
			member := &OrgMember{
				Login:     string(edge.Node.Login),
				ID:        int64(edge.Node.DatabaseId),
				AvatarURL: string(edge.Node.AvatarUrl),
				Role:      strings.ToLower(string(edge.Role)), // GraphQL returns "ADMIN" or "MEMBER"
			}
			// Use newStr for explicit heap allocation
			if edge.Node.Name != "" {
				member.Name = newStr(string(edge.Node.Name))
			}
			if edge.Node.Email != "" {
				member.Email = newStr(string(edge.Node.Email))
			}
			allMembers = append(allMembers, member)
		}

		if !bool(query.Organization.MembersWithRole.PageInfo.HasNextPage) {
			break
		}
		cursor = newString(query.Organization.MembersWithRole.PageInfo.EndCursor)
	}

	c.logger.Debug("Organization member listing complete",
		"org", org,
		"total_members", len(allMembers))

	return allMembers, nil
}

// listOrgMembersREST is a fallback that uses REST API when GraphQL is unavailable
func (c *Client) listOrgMembersREST(ctx context.Context, org string) ([]*OrgMember, error) {
	var allMembers []*OrgMember
	opts := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		var members []*github.User
		var resp *github.Response

		err := c.retryer.Do(ctx, "ListOrgMembersREST", func(ctx context.Context) error {
			var err error
			members, resp, err = c.rest.Organizations.ListMembers(ctx, org, opts)
			if err != nil {
				return WrapError(err, "ListOrgMembersREST", c.baseURL)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		for _, member := range members {
			m := &OrgMember{
				Login:     member.GetLogin(),
				ID:        member.GetID(),
				AvatarURL: member.GetAvatarURL(),
				Role:      "member",
			}
			// Get user details for name/email, using newStr for explicit heap allocation
			if user, _, err := c.rest.Users.Get(ctx, member.GetLogin()); err == nil && user != nil {
				if user.GetName() != "" {
					m.Name = newStr(user.GetName())
				}
				if user.GetEmail() != "" {
					m.Email = newStr(user.GetEmail())
				}
			}
			allMembers = append(allMembers, m)
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allMembers, nil
}

// Mannequin represents a mannequin user created by GEI during migration
type Mannequin struct {
	ID        string  `json:"id"`         // GraphQL node ID
	Login     string  `json:"login"`      // Mannequin login (e.g., "mona-user-12345")
	Email     string  `json:"email"`      // Original commit email
	Name      string  `json:"name"`       // Display name (may contain original username)
	CreatedAt string  `json:"created_at"` // When the mannequin was created
	Claimant  *string `json:"claimant"`   // User who claimed the mannequin (if any)
	OrgID     string  `json:"org_id"`     // Organization node ID (needed for reclaim mutation)
}

// ListMannequins lists all mannequins in an organization
// This uses the GraphQL API as mannequins are only available via GraphQL
func (c *Client) ListMannequins(ctx context.Context, org string) ([]*Mannequin, error) {
	c.logger.Debug("Listing mannequins for organization", "org", org)

	var allMannequins []*Mannequin
	var cursor *githubv4.String
	var orgID string

	// GraphQL query for listing mannequins
	var query struct {
		Organization struct {
			ID         githubv4.String
			Mannequins struct {
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
				Nodes []struct {
					ID        githubv4.String
					Login     githubv4.String
					Email     githubv4.String
					Name      githubv4.String
					CreatedAt githubv4.String
					Claimant  *struct {
						Login githubv4.String
					}
				}
			} `graphql:"mannequins(first: 100, after: $cursor)"`
		} `graphql:"organization(login: $org)"`
	}

	variables := map[string]interface{}{
		"org":    githubv4.String(org),
		"cursor": cursor,
	}

	for {
		variables["cursor"] = cursor

		err := c.retryer.Do(ctx, "ListMannequins", func(ctx context.Context) error {
			return c.graphql.Query(ctx, &query, variables)
		})

		if err != nil {
			// If organization doesn't support mannequins, return empty list
			if strings.Contains(err.Error(), "Could not resolve to an Organization") {
				c.logger.Debug("Organization not found or doesn't have mannequins", "org", org)
				return allMannequins, nil
			}
			return nil, WrapError(err, "ListMannequins", c.baseURL)
		}

		// Capture org ID from first query
		if orgID == "" {
			orgID = string(query.Organization.ID)
		}

		for _, m := range query.Organization.Mannequins.Nodes {
			mannequin := &Mannequin{
				ID:        string(m.ID),
				Login:     string(m.Login),
				Email:     string(m.Email),
				Name:      string(m.Name),
				CreatedAt: string(m.CreatedAt),
				OrgID:     orgID,
			}
			// Use newStr for explicit heap allocation
			if m.Claimant != nil {
				mannequin.Claimant = newStr(string(m.Claimant.Login))
			}
			allMannequins = append(allMannequins, mannequin)
		}

		if !bool(query.Organization.Mannequins.PageInfo.HasNextPage) {
			break
		}
		cursor = newString(query.Organization.Mannequins.PageInfo.EndCursor)
	}

	c.logger.Info("Mannequin listing complete",
		"org", org,
		"total_mannequins", len(allMannequins))

	return allMannequins, nil
}

// UserInfo represents basic user information from GraphQL
type UserInfo struct {
	ID        string `json:"id"`    // GraphQL node ID
	Login     string `json:"login"` // GitHub username
	Name      string `json:"name"`  // Display name
	Email     string `json:"email"` // Public email
	AvatarURL string `json:"avatar_url"`
}

// GetUserByLogin retrieves a user by login using GraphQL
// Returns the user's node ID which is needed for the createAttributionInvitation mutation
func (c *Client) GetUserByLogin(ctx context.Context, login string) (*UserInfo, error) {
	c.logger.Debug("Getting user by login", "login", login)

	var query struct {
		User struct {
			ID        githubv4.String
			Login     githubv4.String
			Name      githubv4.String
			Email     githubv4.String
			AvatarUrl githubv4.String
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(login),
	}

	err := c.retryer.Do(ctx, "GetUserByLogin", func(ctx context.Context) error {
		return c.graphql.Query(ctx, &query, variables)
	})

	if err != nil {
		// If user not found, return nil without error
		if strings.Contains(err.Error(), "Could not resolve to a User") {
			c.logger.Debug("User not found", "login", login)
			return nil, nil
		}
		return nil, WrapError(err, "GetUserByLogin", c.baseURL)
	}

	user := &UserInfo{
		ID:        string(query.User.ID),
		Login:     string(query.User.Login),
		Name:      string(query.User.Name),
		Email:     string(query.User.Email),
		AvatarURL: string(query.User.AvatarUrl),
	}

	c.logger.Debug("User found", "login", login, "id", user.ID)
	return user, nil
}

// AttributionInvitationResult represents the result of a createAttributionInvitation mutation
type AttributionInvitationResult struct {
	Success         bool   `json:"success"`
	MannequinID     string `json:"mannequin_id"`
	MannequinLogin  string `json:"mannequin_login"`
	TargetUserID    string `json:"target_user_id"`
	TargetUserLogin string `json:"target_user_login"`
}

// CreateAttributionInvitation sends an invitation to reclaim a mannequin
// This uses the createAttributionInvitation GraphQL mutation
// See: https://docs.github.com/en/enterprise-cloud@latest/graphql/reference/mutations#createattributioninvitation
//
// Parameters:
//   - ownerID: The organization node ID (from ListMannequins)
//   - mannequinID: The mannequin node ID to reclaim
//   - targetUserID: The target user node ID to attribute to (from GetUserByLogin)
func (c *Client) CreateAttributionInvitation(ctx context.Context, ownerID, mannequinID, targetUserID string) (*AttributionInvitationResult, error) {
	c.logger.Info("Creating attribution invitation",
		"owner_id", ownerID,
		"mannequin_id", mannequinID,
		"target_user_id", targetUserID)

	var mutation struct {
		CreateAttributionInvitation struct {
			ClientMutationId githubv4.String
			Owner            struct {
				Login githubv4.String
				ID    githubv4.String
			}
			Source struct {
				Mannequin struct {
					ID    githubv4.String
					Login githubv4.String
					Email githubv4.String
				} `graphql:"... on Mannequin"`
			}
			Target struct {
				User struct {
					ID    githubv4.String
					Login githubv4.String
					Email githubv4.String
					Name  githubv4.String
				} `graphql:"... on User"`
			}
		} `graphql:"createAttributionInvitation(input: $input)"`
	}

	input := githubv4.CreateAttributionInvitationInput{
		OwnerID:  githubv4.ID(ownerID),
		SourceID: githubv4.ID(mannequinID),
		TargetID: githubv4.ID(targetUserID),
	}

	err := c.retryer.Do(ctx, "CreateAttributionInvitation", func(ctx context.Context) error {
		return c.graphql.Mutate(ctx, &mutation, input, nil)
	})

	if err != nil {
		c.logger.Error("Failed to create attribution invitation",
			"owner_id", ownerID,
			"mannequin_id", mannequinID,
			"target_user_id", targetUserID,
			"error", err)
		return nil, WrapError(err, "CreateAttributionInvitation", c.baseURL)
	}

	result := &AttributionInvitationResult{
		Success:         true,
		MannequinID:     string(mutation.CreateAttributionInvitation.Source.Mannequin.ID),
		MannequinLogin:  string(mutation.CreateAttributionInvitation.Source.Mannequin.Login),
		TargetUserID:    string(mutation.CreateAttributionInvitation.Target.User.ID),
		TargetUserLogin: string(mutation.CreateAttributionInvitation.Target.User.Login),
	}

	c.logger.Info("Attribution invitation created successfully",
		"mannequin_login", result.MannequinLogin,
		"target_user_login", result.TargetUserLogin)

	return result, nil
}
