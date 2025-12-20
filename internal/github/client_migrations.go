package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v75/github"
)

// StartMigrationOptions contains all options for starting a migration archive generation.
// This struct supports additional fields not available in go-github's MigrationOptions.
// See: https://docs.github.com/en/rest/migrations/orgs#start-an-organization-migration
type StartMigrationOptions struct {
	Repositories         []string
	LockRepositories     bool
	ExcludeMetadata      bool
	ExcludeGitData       bool
	ExcludeAttachments   bool
	ExcludeReleases      bool
	ExcludeOwnerProjects bool // Exclude organization projects (recommended: true for metadata archive)
}

// startMigrationRequest is the JSON body for the start migration API.
type startMigrationRequest struct {
	Repositories         []string `json:"repositories"`
	LockRepositories     *bool    `json:"lock_repositories,omitempty"`
	ExcludeMetadata      *bool    `json:"exclude_metadata,omitempty"`
	ExcludeGitData       *bool    `json:"exclude_git_data,omitempty"`
	ExcludeAttachments   *bool    `json:"exclude_attachments,omitempty"`
	ExcludeReleases      *bool    `json:"exclude_releases,omitempty"`
	ExcludeOwnerProjects *bool    `json:"exclude_owner_projects,omitempty"`
}

// StartMigrationWithOptions starts a migration archive generation with extended options.
// This method uses raw HTTP requests to access exclude_metadata and exclude_git_data parameters
// that are not exposed by the go-github library.
// See: https://docs.github.com/en/rest/migrations/orgs#start-an-organization-migration
func (c *Client) StartMigrationWithOptions(ctx context.Context, org string, opts StartMigrationOptions) (*github.Migration, error) {
	body := &startMigrationRequest{
		Repositories:         opts.Repositories,
		LockRepositories:     github.Ptr(opts.LockRepositories),
		ExcludeMetadata:      github.Ptr(opts.ExcludeMetadata),
		ExcludeGitData:       github.Ptr(opts.ExcludeGitData),
		ExcludeAttachments:   github.Ptr(opts.ExcludeAttachments),
		ExcludeReleases:      github.Ptr(opts.ExcludeReleases),
		ExcludeOwnerProjects: github.Ptr(opts.ExcludeOwnerProjects),
	}

	var migration *github.Migration
	_, err := c.DoWithRetry(ctx, "StartMigrationWithOptions", func(ctx context.Context) (*github.Response, error) {
		req, err := c.rest.NewRequest("POST", fmt.Sprintf("orgs/%s/migrations", org), body)
		if err != nil {
			return nil, err
		}

		// Set the migrations preview header required by the API
		req.Header.Set("Accept", "application/vnd.github.wyandotte-preview+json")

		migration = &github.Migration{}
		resp, err := c.rest.Do(ctx, req, migration)
		return resp, err
	})

	if err != nil {
		return nil, WrapError(err, "StartMigrationWithOptions", c.baseURL)
	}

	c.logger.Info("Migration started successfully",
		"org", org,
		"repositories", opts.Repositories,
		"exclude_metadata", opts.ExcludeMetadata,
		"exclude_git_data", opts.ExcludeGitData,
		"exclude_attachments", opts.ExcludeAttachments,
		"exclude_releases", opts.ExcludeReleases,
		"lock_repositories", opts.LockRepositories)

	return migration, nil
}

// UnlockRepository unlocks a repository that was locked during a migration.
// This is used when a migration fails and the source repository remains locked.
// See: https://docs.github.com/en/rest/migrations/orgs#unlock-an-organization-repository
func (c *Client) UnlockRepository(ctx context.Context, org, repo string, migrationID int64) error {
	_, err := c.DoWithRetry(ctx, "UnlockRepository", func(ctx context.Context) (*github.Response, error) {
		req, err := c.rest.NewRequest("DELETE",
			fmt.Sprintf("orgs/%s/migrations/%d/repos/%s/lock", org, migrationID, repo),
			nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.rest.Do(ctx, req, nil)
		return resp, err
	})

	if err != nil {
		return WrapError(err, "UnlockRepository", c.baseURL)
	}

	c.logger.Info("Repository unlocked successfully",
		"org", org,
		"repo", repo,
		"migration_id", migrationID)

	return nil
}

// OrgAppInstallation represents a GitHub App installation with repo access info.
type OrgAppInstallation struct {
	ID                  int64
	AppSlug             string
	RepositorySelection string // "all" or "selected"
}

// ListOrgInstallations lists all GitHub App installations for an organization.
// Returns app installations with their repository access type.
func (c *Client) ListOrgInstallations(ctx context.Context, org string) ([]*OrgAppInstallation, error) {
	var allInstallations []*OrgAppInstallation
	opts := &github.ListOptions{PerPage: 100}

	for {
		result, resp, err := c.rest.Organizations.ListInstallations(ctx, org, opts)
		if err != nil {
			return nil, WrapError(err, "ListOrgInstallations", c.baseURL)
		}

		for _, install := range result.Installations {
			allInstallations = append(allInstallations, &OrgAppInstallation{
				ID:                  install.GetID(),
				AppSlug:             install.GetAppSlug(),
				RepositorySelection: install.GetRepositorySelection(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Debug("Listed org installations",
		"org", org,
		"count", len(allInstallations))

	return allInstallations, nil
}

// ListInstallationRepos lists repositories accessible to a specific app installation.
// This is used to check if a "selected" installation has access to a specific repo.
func (c *Client) ListInstallationRepos(ctx context.Context, installationID int64) ([]string, error) {
	var repoNames []string

	// This requires authentication as the app installation
	// For now, we use a raw request since we might be using PAT auth
	_, err := c.DoWithRetry(ctx, "ListInstallationRepos", func(ctx context.Context) (*github.Response, error) {
		req, err := c.rest.NewRequest("GET", fmt.Sprintf("user/installations/%d/repositories", installationID), nil)
		if err != nil {
			return nil, err
		}

		var result struct {
			Repositories []*github.Repository `json:"repositories"`
		}
		resp, err := c.rest.Do(ctx, req, &result)
		if err != nil {
			return resp, err
		}

		for _, repo := range result.Repositories {
			repoNames = append(repoNames, repo.GetFullName())
		}
		return resp, nil
	})

	if err != nil {
		return nil, WrapError(err, "ListInstallationRepos", c.baseURL)
	}

	return repoNames, nil
}
