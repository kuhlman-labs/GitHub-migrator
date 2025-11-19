package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

// EntraIDAuthorizer handles ADO-specific authorization checks
type EntraIDAuthorizer struct {
	config    *config.AuthConfig
	adoOrgURL string
	logger    *slog.Logger
}

// NewEntraIDAuthorizer creates a new Entra ID authorizer
func NewEntraIDAuthorizer(cfg *config.AuthConfig, logger *slog.Logger) *EntraIDAuthorizer {
	return &EntraIDAuthorizer{
		config:    cfg,
		adoOrgURL: cfg.ADOOrganizationURL,
		logger:    logger,
	}
}

// CheckADOOrganizationMembership verifies if a user is a member of the ADO organization
func (a *EntraIDAuthorizer) CheckADOOrganizationMembership(ctx context.Context, userID string, accessToken string) (bool, error) {
	// Build API URL to check user entitlements in the organization
	apiURL := fmt.Sprintf("%s/_apis/userentitlements?api-version=6.0-preview.3&$filter=id eq '%s'", a.adoOrgURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check organization membership: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logger.Debug("User not found in organization", "user_id", userID, "status", resp.StatusCode)
		return false, nil
	}

	// Parse response to verify user exists
	var result struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Count > 0, nil
}

// CheckADOProjectAccess verifies if a user has access to a specific ADO project
// Note: ADO doesn't expose fine-grained repository permissions via API
// If a user has access to the project, we assume they can access repositories within it
func (a *EntraIDAuthorizer) CheckADOProjectAccess(ctx context.Context, projectName string, accessToken string) (bool, error) {
	// Build API URL to get project details
	apiURL := fmt.Sprintf("%s/_apis/projects/%s?api-version=6.0", a.adoOrgURL, url.PathEscape(projectName))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check project access: %w", err)
	}
	defer resp.Body.Close()

	// 200 = user has access
	// 401/403/404 = user doesn't have access
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	a.logger.Debug("User doesn't have access to project",
		"project", projectName,
		"status", resp.StatusCode)
	return false, nil
}

// GetUserProjects returns a list of projects the user has access to
func (a *EntraIDAuthorizer) GetUserProjects(ctx context.Context, accessToken string) ([]string, error) {
	// Build API URL to list all projects
	apiURL := fmt.Sprintf("%s/_apis/projects?api-version=6.0", a.adoOrgURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list projects: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Value []struct {
			Name string `json:"name"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	projects := make([]string, len(result.Value))
	for i, p := range result.Value {
		projects[i] = p.Name
	}

	return projects, nil
}

// ValidateAccessForRepository validates that a user can migrate a specific repository
// For ADO sources, we check project-level access (ADO doesn't expose repo-level permissions)
func (a *EntraIDAuthorizer) ValidateAccessForRepository(ctx context.Context, projectName string, accessToken string) error {
	hasAccess, err := a.CheckADOProjectAccess(ctx, projectName, accessToken)
	if err != nil {
		return fmt.Errorf("failed to check project access: %w", err)
	}

	if !hasAccess {
		return fmt.Errorf("you don't have access to project: %s", projectName)
	}

	return nil
}

// ValidateAccessForRepositories validates access for multiple repositories
// For ADO, we check unique projects and verify access to each
func (a *EntraIDAuthorizer) ValidateAccessForRepositories(ctx context.Context, repoProjects []string, accessToken string) error {
	// Get unique projects
	projectMap := make(map[string]bool)
	for _, project := range repoProjects {
		if project != "" {
			projectMap[project] = true
		}
	}

	// Check access to each unique project
	var inaccessibleProjects []string
	for project := range projectMap {
		hasAccess, err := a.CheckADOProjectAccess(ctx, project, accessToken)
		if err != nil {
			a.logger.Warn("Failed to check project access",
				"project", project,
				"error", err)
			inaccessibleProjects = append(inaccessibleProjects, project)
			continue
		}

		if !hasAccess {
			inaccessibleProjects = append(inaccessibleProjects, project)
		}
	}

	if len(inaccessibleProjects) > 0 {
		return fmt.Errorf("you don't have access to the following projects: %v", inaccessibleProjects)
	}

	return nil
}
