package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

const (
	defaultGitHubAPIURL     = "https://api.github.com"
	defaultGitHubGraphQLURL = "https://api.github.com/graphql"
	membershipStateActive   = "active"
	membershipRoleAdmin     = "admin"
)

// Authorizer handles user authorization checks
type Authorizer struct {
	config  *config.AuthConfig
	logger  *slog.Logger
	baseURL string // GitHub API base URL
}

// NewAuthorizer creates a new authorizer
func NewAuthorizer(cfg *config.AuthConfig, logger *slog.Logger, githubBaseURL string) *Authorizer {
	apiURL := githubBaseURL
	if apiURL == "" {
		apiURL = defaultGitHubAPIURL
	}
	if apiURL[len(apiURL)-1] == '/' {
		apiURL = apiURL[:len(apiURL)-1]
	}

	return &Authorizer{
		config:  cfg,
		logger:  logger,
		baseURL: apiURL,
	}
}

// AuthorizationResult contains the result of authorization checks
type AuthorizationResult struct {
	Authorized bool
	Reason     string
	Details    map[string]any
}

// Authorize checks if a user is authorized based on configured rules
func (a *Authorizer) Authorize(ctx context.Context, user *GitHubUser, githubToken string) (*AuthorizationResult, error) { //nolint:gocyclo // TODO: refactor to reduce complexity
	rules := a.config.AuthorizationRules

	// If no rules are configured, allow access
	if len(rules.RequireOrgMembership) == 0 &&
		len(rules.RequireTeamMembership) == 0 &&
		!rules.RequireEnterpriseAdmin {
		return &AuthorizationResult{
			Authorized: true,
			Reason:     "No authorization rules configured",
		}, nil
	}

	// Check organization membership
	if len(rules.RequireOrgMembership) > 0 {
		authorized, err := a.CheckOrganizationMembership(ctx, user.Login, rules.RequireOrgMembership, githubToken)
		if err != nil {
			return nil, fmt.Errorf("failed to check org membership: %w", err)
		}
		if !authorized {
			return &AuthorizationResult{
				Authorized: false,
				Reason:     fmt.Sprintf("User is not a member of required organizations: %s", strings.Join(rules.RequireOrgMembership, ", ")),
				Details: map[string]any{
					"required_orgs": rules.RequireOrgMembership,
				},
			}, nil
		}
	}

	// Check team membership
	if len(rules.RequireTeamMembership) > 0 {
		authorized, err := a.CheckTeamMembership(ctx, user.Login, rules.RequireTeamMembership, githubToken)
		if err != nil {
			return nil, fmt.Errorf("failed to check team membership: %w", err)
		}
		if !authorized {
			return &AuthorizationResult{
				Authorized: false,
				Reason:     fmt.Sprintf("User is not a member of required teams: %s", strings.Join(rules.RequireTeamMembership, ", ")),
				Details: map[string]any{
					"required_teams": rules.RequireTeamMembership,
				},
			}, nil
		}
	}

	// Check enterprise admin
	if rules.RequireEnterpriseAdmin {
		if rules.RequireEnterpriseSlug == "" {
			return nil, fmt.Errorf("require_enterprise_slug must be set when require_enterprise_admin is true")
		}
		authorized, err := a.CheckEnterpriseAdmin(ctx, user.Login, rules.RequireEnterpriseSlug, githubToken)
		if err != nil {
			return nil, fmt.Errorf("failed to check enterprise admin: %w", err)
		}
		if !authorized {
			return &AuthorizationResult{
				Authorized: false,
				Reason:     fmt.Sprintf("User is not an enterprise admin of %s", rules.RequireEnterpriseSlug),
				Details: map[string]any{
					"required_enterprise": rules.RequireEnterpriseSlug,
				},
			}, nil
		}
	}

	// Check enterprise membership (any role, not just admin)
	if rules.RequireEnterpriseMembership {
		if rules.RequireEnterpriseSlug == "" {
			return nil, fmt.Errorf("require_enterprise_slug must be set when require_enterprise_membership is true")
		}
		authorized, err := a.CheckEnterpriseMembership(ctx, user.Login, rules.RequireEnterpriseSlug, githubToken)
		if err != nil {
			return nil, fmt.Errorf("failed to check enterprise membership: %w", err)
		}
		if !authorized {
			return &AuthorizationResult{
				Authorized: false,
				Reason:     fmt.Sprintf("User is not a member of enterprise %s", rules.RequireEnterpriseSlug),
				Details: map[string]any{
					"required_enterprise": rules.RequireEnterpriseSlug,
				},
			}, nil
		}
	}

	// All checks passed
	return &AuthorizationResult{
		Authorized: true,
		Reason:     "User meets all authorization requirements",
	}, nil
}

// CheckOrganizationMembership checks if user is a member of at least one required org
func (a *Authorizer) CheckOrganizationMembership(ctx context.Context, username string, requiredOrgs []string, token string) (bool, error) {
	for _, org := range requiredOrgs {
		isMember, err := a.isOrgMember(ctx, username, org, token)
		if err != nil {
			a.logger.Warn("Failed to check org membership", "org", org, "user", username, "error", err)
			continue
		}
		if isMember {
			a.logger.Info("User is member of required org", "org", org, "user", username)
			return true, nil
		}
	}
	return false, nil
}

// CheckTeamMembership checks if user is a member of at least one required team
func (a *Authorizer) CheckTeamMembership(ctx context.Context, username string, requiredTeams []string, token string) (bool, error) {
	for _, teamSlug := range requiredTeams {
		// Parse "org/team-slug" format
		parts := strings.Split(teamSlug, "/")
		if len(parts) != 2 {
			a.logger.Warn("Invalid team slug format", "team", teamSlug)
			continue
		}
		org, team := parts[0], parts[1]

		isMember, err := a.isTeamMember(ctx, username, org, team, token)
		if err != nil {
			a.logger.Warn("Failed to check team membership", "org", org, "team", team, "user", username, "error", err)
			continue
		}
		if isMember {
			a.logger.Info("User is member of required team", "org", org, "team", team, "user", username)
			return true, nil
		}
	}
	return false, nil
}

// CheckEnterpriseAdmin checks if user is an enterprise admin using GraphQL API
func (a *Authorizer) CheckEnterpriseAdmin(ctx context.Context, username string, enterpriseSlug string, token string) (bool, error) {
	// Use GraphQL API to check enterprise admin status
	// This works with OAuth tokens, unlike the REST API endpoint
	graphqlURL := defaultGitHubGraphQLURL
	if a.baseURL != defaultGitHubAPIURL && a.baseURL != "" {
		// For GHES, GraphQL endpoint is at /api/graphql
		graphqlURL = strings.TrimSuffix(a.baseURL, "/api") + "/graphql"
	}

	query := `query($enterpriseSlug: String!) {
		enterprise(slug: $enterpriseSlug) {
			slug
			viewerIsAdmin
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]string{
			"enterpriseSlug": enterpriseSlug,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	a.logger.Debug("Checking enterprise admin status via GraphQL",
		"url", graphqlURL,
		"username", username,
		"enterprise", enterpriseSlug)

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		a.logger.Error("Enterprise admin check failed with error", "error", err, "url", graphqlURL)
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	a.logger.Debug("Enterprise admin GraphQL response",
		"status", resp.StatusCode,
		"body", string(body))

	if resp.StatusCode != http.StatusOK {
		a.logger.Error("GitHub GraphQL API returned non-OK status",
			"status", resp.StatusCode,
			"body", string(body))
		return false, fmt.Errorf("github GraphQL API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Enterprise struct {
				Slug          string `json:"slug"`
				ViewerIsAdmin bool   `json:"viewerIsAdmin"`
			} `json:"enterprise"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		a.logger.Error("Failed to parse GraphQL response", "error", err, "body", string(body))
		return false, err
	}

	// Check for GraphQL errors
	if len(result.Errors) > 0 {
		errMsg := result.Errors[0].Message
		a.logger.Warn("GraphQL query returned errors",
			"error", errMsg,
			"enterprise", enterpriseSlug,
			"user", username)
		// If enterprise not found or user doesn't have access, treat as not admin
		return false, nil
	}

	isAdmin := result.Data.Enterprise.ViewerIsAdmin
	a.logger.Info("Enterprise admin check result",
		"username", username,
		"enterprise", enterpriseSlug,
		"is_admin", isAdmin)

	return isAdmin, nil
}

// CheckEnterpriseMembership checks if user is a member of an enterprise (any role) using GraphQL API
func (a *Authorizer) CheckEnterpriseMembership(ctx context.Context, username string, enterpriseSlug string, token string) (bool, error) {
	// Use GraphQL API to check enterprise membership
	// This checks if the user has any access to the enterprise (member or admin)
	graphqlURL := defaultGitHubGraphQLURL
	if a.baseURL != defaultGitHubAPIURL && a.baseURL != "" {
		// For GHES, GraphQL endpoint is at /api/graphql
		graphqlURL = strings.TrimSuffix(a.baseURL, "/api") + "/graphql"
	}

	query := `query($enterpriseSlug: String!) {
		enterprise(slug: $enterpriseSlug) {
			slug
			viewerIsAdmin
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]string{
			"enterpriseSlug": enterpriseSlug,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	a.logger.Debug("Checking enterprise membership via GraphQL",
		"url", graphqlURL,
		"username", username,
		"enterprise", enterpriseSlug)

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		a.logger.Error("Enterprise membership check failed with error", "error", err, "url", graphqlURL)
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	a.logger.Debug("Enterprise membership GraphQL response",
		"status", resp.StatusCode,
		"body", string(body))

	if resp.StatusCode != http.StatusOK {
		a.logger.Error("GitHub GraphQL API returned non-OK status",
			"status", resp.StatusCode,
			"body", string(body))
		return false, fmt.Errorf("github GraphQL API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Enterprise struct {
				Slug          string `json:"slug"`
				ViewerIsAdmin bool   `json:"viewerIsAdmin"`
			} `json:"enterprise"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		a.logger.Error("Failed to parse GraphQL response", "error", err, "body", string(body))
		return false, err
	}

	// Check for GraphQL errors
	if len(result.Errors) > 0 {
		errMsg := result.Errors[0].Message
		a.logger.Warn("GraphQL query returned errors",
			"error", errMsg,
			"enterprise", enterpriseSlug,
			"user", username)
		// If enterprise not found or user doesn't have access, treat as not a member
		return false, nil
	}

	// If we successfully queried the enterprise, the user is a member
	// (non-members would get an error)
	isMember := result.Data.Enterprise.Slug != ""

	a.logger.Info("Enterprise membership check result",
		"username", username,
		"enterprise", enterpriseSlug,
		"is_member", isMember,
		"is_admin", result.Data.Enterprise.ViewerIsAdmin)

	return isMember, nil
}

// isOrgMember checks if a user is a member of an organization
// For OAuth flows, use the authenticated user's membership endpoint which is more reliable
func (a *Authorizer) isOrgMember(ctx context.Context, username string, org string, token string) (bool, error) {
	// Use the /user/memberships/orgs/{org} endpoint which checks the authenticated user's membership
	// This is more reliable for OAuth tokens and works regardless of membership visibility
	url := fmt.Sprintf("%s/user/memberships/orgs/%s", a.baseURL, org)

	a.logger.Debug("Checking org membership for authenticated user",
		"url", url,
		"username", username,
		"org", org)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		a.logger.Error("Failed to make org membership API request",
			"url", url,
			"error", err)
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	a.logger.Debug("Org membership API response",
		"status", resp.StatusCode,
		"org", org,
		"username", username,
		"response_body", string(body))

	// 200 means member, 404 means not a member
	if resp.StatusCode == http.StatusOK {
		// Parse response to check state
		var membership struct {
			State string `json:"state"`
			Role  string `json:"role"`
		}
		if err := json.Unmarshal(body, &membership); err != nil {
			a.logger.Error("Failed to parse membership response",
				"error", err,
				"body", string(body))
			return false, err
		}

		// State can be "active" or "pending"
		// Only consider "active" as valid membership
		if membership.State == membershipStateActive {
			a.logger.Info("User IS an active member of organization",
				"org", org,
				"username", username,
				"role", membership.Role)
			return true, nil
		}

		a.logger.Info("User membership is not active",
			"org", org,
			"username", username,
			"state", membership.State)
		return false, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		a.logger.Info("User is NOT a member of organization",
			"org", org,
			"username", username)
		return false, nil
	}

	a.logger.Error("Unexpected status code from org membership API",
		"status", resp.StatusCode,
		"org", org,
		"username", username,
		"body", string(body))
	return false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
}

// isTeamMember checks if a user is a member of a team
func (a *Authorizer) isTeamMember(ctx context.Context, username string, org string, teamSlug string, token string) (bool, error) {
	url := fmt.Sprintf("%s/orgs/%s/teams/%s/memberships/%s", a.baseURL, org, teamSlug, username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	var membership struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&membership); err != nil {
		return false, err
	}

	// State can be "active" or "pending"
	return membership.State == membershipStateActive, nil
}

// IsOrgAdmin checks if a user has admin role in an organization
func (a *Authorizer) IsOrgAdmin(ctx context.Context, username string, org string, token string) (bool, error) {
	// Use the /user/memberships/orgs/{org} endpoint which returns role information
	url := fmt.Sprintf("%s/user/memberships/orgs/%s", a.baseURL, org)

	a.logger.Debug("Checking org admin role for user",
		"url", url,
		"username", username,
		"org", org)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// User is not a member of the organization
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	var membership struct {
		State string `json:"state"`
		Role  string `json:"role"` // "member" or "admin"
	}
	if err := json.NewDecoder(resp.Body).Decode(&membership); err != nil {
		return false, err
	}

	// Only consider active admin memberships
	isAdmin := membership.State == membershipStateActive && membership.Role == membershipRoleAdmin

	a.logger.Debug("Org admin check result",
		"username", username,
		"org", org,
		"is_admin", isAdmin,
		"role", membership.Role,
		"state", membership.State)

	return isAdmin, nil
}

// HasRepoAdminPermission checks if a user has admin permission on a specific repository
// Uses GraphQL to check the viewer's (authenticated user's) permission directly
func (a *Authorizer) HasRepoAdminPermission(ctx context.Context, username string, org string, repo string, token string) (bool, error) {
	// Use GraphQL API to check viewer's permission (more reliable than REST API)
	graphqlURL := defaultGitHubGraphQLURL
	if a.baseURL != defaultGitHubAPIURL && a.baseURL != "" {
		// For GHES, GraphQL endpoint is at /api/graphql
		graphqlURL = strings.TrimSuffix(a.baseURL, "/api") + "/graphql"
	}

	query := `query($owner: String!, $name: String!) {
		repository(owner: $owner, name: $name) {
			viewerPermission
		}
	}`

	payload := map[string]any{
		"query": query,
		"variables": map[string]string{
			"owner": org,
			"name":  repo,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	a.logger.Debug("Checking repository admin permission via GraphQL",
		"url", graphqlURL,
		"username", username,
		"org", org,
		"repo", repo)

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("github GraphQL API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Repository struct {
				ViewerPermission string `json:"viewerPermission"` // "ADMIN", "WRITE", "READ", "NONE"
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	// Check for GraphQL errors
	if len(result.Errors) > 0 {
		errorMsg := result.Errors[0].Message
		a.logger.Debug("GraphQL query returned errors",
			"error", errorMsg,
			"repo", fmt.Sprintf("%s/%s", org, repo))

		// Provide more context for common errors
		if strings.Contains(errorMsg, "Could not resolve to a Repository") {
			a.logger.Info("Repository not found or no access",
				"repo", fmt.Sprintf("%s/%s", org, repo),
				"username", username,
				"hint", "Repository may not exist, user may lack access, or name may have incorrect case (GitHub is case-sensitive)")
		}

		// If repo not found or user doesn't have access, treat as no permission
		return false, nil
	}

	// Check if user has ADMIN permission
	hasAdmin := result.Data.Repository.ViewerPermission == "ADMIN"

	a.logger.Debug("Repository permission check result",
		"username", username,
		"repo", fmt.Sprintf("%s/%s", org, repo),
		"permission", result.Data.Repository.ViewerPermission,
		"has_admin", hasAdmin)

	return hasAdmin, nil
}

// GetUserOrganizations returns a list of organizations the authenticated user is a member of
func (a *Authorizer) GetUserOrganizations(ctx context.Context, token string) ([]string, error) {
	url := fmt.Sprintf("%s/user/memberships/orgs", a.baseURL)

	a.logger.Debug("Fetching user organizations", "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	var memberships []struct {
		Organization struct {
			Login string `json:"login"`
		} `json:"organization"`
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&memberships); err != nil {
		return nil, err
	}

	var orgs []string
	for _, membership := range memberships {
		// Only include active memberships
		if membership.State == membershipStateActive {
			orgs = append(orgs, membership.Organization.Login)
		}
	}

	a.logger.Debug("Found user organizations", "count", len(orgs))

	return orgs, nil
}
