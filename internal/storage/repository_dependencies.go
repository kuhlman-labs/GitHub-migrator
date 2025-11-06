package storage

import (
	"context"
	"fmt"

	"github.com/brettkuhlman/github-migrator/internal/models"
)

// SaveRepositoryDependencies saves or updates dependencies for a repository
// This replaces existing dependencies for the given repository
func (d *Database) SaveRepositoryDependencies(ctx context.Context, repoID int64, dependencies []*models.RepositoryDependency) error {
	// Start transaction
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	// Clear existing dependencies for this repository
	if _, err := tx.ExecContext(ctx, d.rebindQuery("DELETE FROM repository_dependencies WHERE repository_id = ?"), repoID); err != nil {
		return fmt.Errorf("failed to clear existing dependencies: %w", err)
	}

	// Insert new dependencies
	if len(dependencies) > 0 {
		query := `
			INSERT INTO repository_dependencies (
				repository_id, dependency_full_name, dependency_type, 
				dependency_url, is_local, discovered_at, metadata
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`

		stmt, err := tx.PrepareContext(ctx, d.rebindQuery(query))
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, dep := range dependencies {
			if _, err := stmt.ExecContext(ctx,
				repoID,
				dep.DependencyFullName,
				dep.DependencyType,
				dep.DependencyURL,
				dep.IsLocal,
				dep.DiscoveredAt,
				dep.Metadata,
			); err != nil {
				return fmt.Errorf("failed to insert dependency %s: %w", dep.DependencyFullName, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetRepositoryDependencies retrieves all dependencies for a repository
func (d *Database) GetRepositoryDependencies(ctx context.Context, repoID int64) ([]*models.RepositoryDependency, error) {
	query := `
		SELECT 
			id, repository_id, dependency_full_name, dependency_type,
			dependency_url, is_local, discovered_at, metadata
		FROM repository_dependencies
		WHERE repository_id = ?
		ORDER BY dependency_type, dependency_full_name
	`

	rows, err := d.db.QueryContext(ctx, d.rebindQuery(query), repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies: %w", err)
	}
	defer rows.Close()

	// Initialize as empty slice instead of nil so JSON serialization returns [] not null
	dependencies := make([]*models.RepositoryDependency, 0)
	for rows.Next() {
		var dep models.RepositoryDependency
		if err := rows.Scan(
			&dep.ID,
			&dep.RepositoryID,
			&dep.DependencyFullName,
			&dep.DependencyType,
			&dep.DependencyURL,
			&dep.IsLocal,
			&dep.DiscoveredAt,
			&dep.Metadata,
		); err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		dependencies = append(dependencies, &dep)
	}

	return dependencies, rows.Err()
}

// GetRepositoryDependenciesByFullName retrieves all dependencies for a repository by its full name
func (d *Database) GetRepositoryDependenciesByFullName(ctx context.Context, fullName string) ([]*models.RepositoryDependency, error) {
	// First get the repository ID
	repo, err := d.GetRepository(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found: %s", fullName)
	}

	return d.GetRepositoryDependencies(ctx, repo.ID)
}

// GetDependentRepositories retrieves all repositories that depend on a specific dependency
// This is useful for batch planning - "what repos depend on X?"
func (d *Database) GetDependentRepositories(ctx context.Context, dependencyFullName string) ([]*models.Repository, error) {
	query := `
		SELECT DISTINCT r.id, r.full_name, r.source, r.source_url, r.total_size, r.largest_file, 
			   r.largest_file_size, r.largest_commit, r.largest_commit_size,
			   r.has_lfs, r.has_submodules, r.has_large_files, r.large_file_count,
			   r.default_branch, r.branch_count, r.commit_count, r.last_commit_sha,
			   r.last_commit_date, r.is_archived, r.is_fork, r.has_wiki, r.has_pages, 
			   r.has_discussions, r.has_actions, r.has_projects, r.has_packages, 
			   r.branch_protections, r.has_rulesets, r.tag_protection_count, r.environment_count, r.secret_count, r.variable_count, 
			   r.webhook_count, r.contributor_count, r.top_contributors,
			   r.issue_count, r.pull_request_count, r.tag_count,
			   r.open_issue_count, r.open_pr_count,
			   r.has_code_scanning, r.has_dependabot, r.has_secret_scanning, r.has_codeowners,
			   r.visibility, r.workflow_count, r.has_self_hosted_runners, r.collaborator_count,
			   r.installed_apps_count, r.release_count, r.has_release_assets,
			   r.has_oversized_commits, r.oversized_commit_details,
			   r.has_long_refs, r.long_ref_details,
			   r.has_blocking_files, r.blocking_file_details,
			   r.has_large_file_warnings, r.large_file_warning_details,
			   r.has_oversized_repository, r.oversized_repository_details,
			   r.estimated_metadata_size, r.metadata_size_details,
			   r.exclude_releases, r.exclude_attachments, r.exclude_metadata, r.exclude_git_data, r.exclude_owner_projects,
			   r.status, r.batch_id, r.priority, r.destination_url, 
			   r.destination_full_name, r.source_migration_id, r.is_source_locked,
			   r.validation_status, r.validation_details, 
			   r.destination_data, r.discovered_at, r.updated_at, r.migrated_at,
			   r.last_discovery_at, r.last_dry_run_at
		FROM repositories r
		INNER JOIN repository_dependencies rd ON r.id = rd.repository_id
		WHERE rd.dependency_full_name = ?
		ORDER BY r.full_name
	`

	rows, err := d.db.QueryContext(ctx, d.rebindQuery(query), dependencyFullName)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependent repositories: %w", err)
	}
	defer rows.Close()

	// Use the existing scanRepositories helper to avoid code duplication
	return d.scanRepositories(rows)
}

// ClearRepositoryDependencies removes all dependencies for a repository
// Useful before re-discovery
func (d *Database) ClearRepositoryDependencies(ctx context.Context, repoID int64) error {
	_, err := d.db.ExecContext(ctx, d.rebindQuery("DELETE FROM repository_dependencies WHERE repository_id = ?"), repoID)
	if err != nil {
		return fmt.Errorf("failed to clear dependencies: %w", err)
	}
	return nil
}

// UpdateLocalDependencyFlags updates the is_local flag for all dependencies
// based on whether the dependency exists in our database
// This should be run after discovery to properly mark local dependencies
func (d *Database) UpdateLocalDependencyFlags(ctx context.Context) error {
	query := `
		UPDATE repository_dependencies
		SET is_local = CASE
			WHEN dependency_full_name IN (SELECT full_name FROM repositories)
			THEN 1
			ELSE 0
		END
	`

	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to update local dependency flags: %w", err)
	}

	return nil
}
