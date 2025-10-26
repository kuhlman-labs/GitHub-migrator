package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/config"
)

const defaultGitHubAPIURL = "https://api.github.com"

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
	Details    map[string]interface{}
}

// Authorize checks if a user is authorized based on configured rules
func (a *Authorizer) Authorize(ctx context.Context, user *GitHubUser, githubToken string) (*AuthorizationResult, error) {
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
				Details: map[string]interface{}{
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
				Details: map[string]interface{}{
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
				Details: map[string]interface{}{
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

// CheckEnterpriseAdmin checks if user is an enterprise admin
func (a *Authorizer) CheckEnterpriseAdmin(ctx context.Context, username string, enterpriseSlug string, token string) (bool, error) {
	url := fmt.Sprintf("%s/enterprises/%s/users/%s", a.baseURL, enterpriseSlug, username)

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

	var result struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	// Check if user has admin role
	return result.Role == "admin" || result.Role == "owner", nil
}

// isOrgMember checks if a user is a member of an organization
func (a *Authorizer) isOrgMember(ctx context.Context, username string, org string, token string) (bool, error) {
	url := fmt.Sprintf("%s/orgs/%s/members/%s", a.baseURL, org, username)

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

	// 204 means member, 404 means not a member, 302 means requester doesn't have permission
	if resp.StatusCode == http.StatusNoContent {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
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
	return membership.State == "active", nil
}
