package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/brettkuhlman/github-migrator/internal/models"
)

// SaveRepository inserts or updates a repository in the database
// nolint:dupl // SaveRepository and UpdateRepository have different SQL operations
func (d *Database) SaveRepository(ctx context.Context, repo *models.Repository) error {
	query := `
		INSERT INTO repositories (
			full_name, source, source_url, total_size, largest_file, 
			largest_file_size, largest_commit, largest_commit_size,
			has_lfs, has_submodules, default_branch, branch_count, 
			commit_count, has_wiki, has_pages, has_discussions, 
			has_actions, has_projects, branch_protections, 
			environment_count, secret_count, variable_count, 
			webhook_count, contributor_count, top_contributors,
			status, batch_id, priority, discovered_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(full_name) DO UPDATE SET
			source = excluded.source,
			source_url = excluded.source_url,
			total_size = excluded.total_size,
			largest_file = excluded.largest_file,
			largest_file_size = excluded.largest_file_size,
			largest_commit = excluded.largest_commit,
			largest_commit_size = excluded.largest_commit_size,
			has_lfs = excluded.has_lfs,
			has_submodules = excluded.has_submodules,
			default_branch = excluded.default_branch,
			branch_count = excluded.branch_count,
			commit_count = excluded.commit_count,
			has_wiki = excluded.has_wiki,
			has_pages = excluded.has_pages,
			has_discussions = excluded.has_discussions,
			has_actions = excluded.has_actions,
			has_projects = excluded.has_projects,
			branch_protections = excluded.branch_protections,
			environment_count = excluded.environment_count,
			secret_count = excluded.secret_count,
			variable_count = excluded.variable_count,
			webhook_count = excluded.webhook_count,
			contributor_count = excluded.contributor_count,
			top_contributors = excluded.top_contributors,
			updated_at = excluded.updated_at
	`

	_, err := d.db.ExecContext(ctx, query,
		repo.FullName, repo.Source, repo.SourceURL, repo.TotalSize,
		repo.LargestFile, repo.LargestFileSize, repo.LargestCommit,
		repo.LargestCommitSize, repo.HasLFS, repo.HasSubmodules,
		repo.DefaultBranch, repo.BranchCount, repo.CommitCount,
		repo.HasWiki, repo.HasPages, repo.HasDiscussions,
		repo.HasActions, repo.HasProjects, repo.BranchProtections,
		repo.EnvironmentCount, repo.SecretCount, repo.VariableCount,
		repo.WebhookCount, repo.ContributorCount, repo.TopContributors,
		repo.Status, repo.BatchID, repo.Priority,
		repo.DiscoveredAt, repo.UpdatedAt,
	)

	return err
}

// GetRepository retrieves a repository by full name
func (d *Database) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	query := `
		SELECT id, full_name, source, source_url, total_size, largest_file, 
			   largest_file_size, largest_commit, largest_commit_size,
			   has_lfs, has_submodules, default_branch, branch_count, 
			   commit_count, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   status, batch_id, priority, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE full_name = ?
	`

	var repo models.Repository
	err := d.db.QueryRowContext(ctx, query, fullName).Scan(
		&repo.ID, &repo.FullName, &repo.Source, &repo.SourceURL,
		&repo.TotalSize, &repo.LargestFile, &repo.LargestFileSize,
		&repo.LargestCommit, &repo.LargestCommitSize, &repo.HasLFS,
		&repo.HasSubmodules, &repo.DefaultBranch, &repo.BranchCount,
		&repo.CommitCount, &repo.HasWiki, &repo.HasPages,
		&repo.HasDiscussions, &repo.HasActions, &repo.HasProjects,
		&repo.BranchProtections, &repo.EnvironmentCount, &repo.SecretCount,
		&repo.VariableCount, &repo.WebhookCount, &repo.ContributorCount,
		&repo.TopContributors, &repo.Status, &repo.BatchID, &repo.Priority,
		&repo.DiscoveredAt, &repo.UpdatedAt, &repo.MigratedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return &repo, nil
}

// ListRepositories retrieves repositories with optional filters
func (d *Database) ListRepositories(ctx context.Context, filters map[string]interface{}) ([]*models.Repository, error) {
	query := `
		SELECT id, full_name, source, source_url, total_size, largest_file, 
			   largest_file_size, largest_commit, largest_commit_size,
			   has_lfs, has_submodules, default_branch, branch_count, 
			   commit_count, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   status, batch_id, priority, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE 1=1
	`
	args := []interface{}{}

	// Apply filters dynamically
	if status, ok := filters["status"].(string); ok && status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if batchID, ok := filters["batch_id"].(int64); ok && batchID > 0 {
		query += " AND batch_id = ?"
		args = append(args, batchID)
	}

	if source, ok := filters["source"].(string); ok && source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}

	if hasLFS, ok := filters["has_lfs"].(bool); ok {
		query += " AND has_lfs = ?"
		args = append(args, hasLFS)
	}

	if hasSubmodules, ok := filters["has_submodules"].(bool); ok {
		query += " AND has_submodules = ?"
		args = append(args, hasSubmodules)
	}

	// Add ordering
	query += " ORDER BY full_name ASC"

	// Add limit if specified
	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	var repos []*models.Repository
	for rows.Next() {
		var repo models.Repository
		err := rows.Scan(
			&repo.ID, &repo.FullName, &repo.Source, &repo.SourceURL,
			&repo.TotalSize, &repo.LargestFile, &repo.LargestFileSize,
			&repo.LargestCommit, &repo.LargestCommitSize, &repo.HasLFS,
			&repo.HasSubmodules, &repo.DefaultBranch, &repo.BranchCount,
			&repo.CommitCount, &repo.HasWiki, &repo.HasPages,
			&repo.HasDiscussions, &repo.HasActions, &repo.HasProjects,
			&repo.BranchProtections, &repo.EnvironmentCount, &repo.SecretCount,
			&repo.VariableCount, &repo.WebhookCount, &repo.ContributorCount,
			&repo.TopContributors, &repo.Status, &repo.BatchID, &repo.Priority,
			&repo.DiscoveredAt, &repo.UpdatedAt, &repo.MigratedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repos = append(repos, &repo)
	}

	return repos, rows.Err()
}

// UpdateRepository updates a repository's fields
// nolint:dupl // SaveRepository and UpdateRepository have different SQL operations
func (d *Database) UpdateRepository(ctx context.Context, repo *models.Repository) error {
	query := `
		UPDATE repositories SET
			source = ?,
			source_url = ?,
			total_size = ?,
			largest_file = ?,
			largest_file_size = ?,
			largest_commit = ?,
			largest_commit_size = ?,
			has_lfs = ?,
			has_submodules = ?,
			default_branch = ?,
			branch_count = ?,
			commit_count = ?,
			has_wiki = ?,
			has_pages = ?,
			has_discussions = ?,
			has_actions = ?,
			has_projects = ?,
			branch_protections = ?,
			environment_count = ?,
			secret_count = ?,
			variable_count = ?,
			webhook_count = ?,
			contributor_count = ?,
			top_contributors = ?,
			status = ?,
			batch_id = ?,
			priority = ?,
			updated_at = ?,
			migrated_at = ?
		WHERE full_name = ?
	`

	_, err := d.db.ExecContext(ctx, query,
		repo.Source, repo.SourceURL, repo.TotalSize,
		repo.LargestFile, repo.LargestFileSize, repo.LargestCommit,
		repo.LargestCommitSize, repo.HasLFS, repo.HasSubmodules,
		repo.DefaultBranch, repo.BranchCount, repo.CommitCount,
		repo.HasWiki, repo.HasPages, repo.HasDiscussions,
		repo.HasActions, repo.HasProjects, repo.BranchProtections,
		repo.EnvironmentCount, repo.SecretCount, repo.VariableCount,
		repo.WebhookCount, repo.ContributorCount, repo.TopContributors,
		repo.Status, repo.BatchID, repo.Priority,
		repo.UpdatedAt, repo.MigratedAt,
		repo.FullName,
	)

	return err
}

// UpdateRepositoryStatus updates only the status of a repository
func (d *Database) UpdateRepositoryStatus(ctx context.Context, fullName string, status models.MigrationStatus) error {
	query := `UPDATE repositories SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE full_name = ?`
	_, err := d.db.ExecContext(ctx, query, string(status), fullName)
	return err
}

// DeleteRepository deletes a repository by full name
func (d *Database) DeleteRepository(ctx context.Context, fullName string) error {
	query := `DELETE FROM repositories WHERE full_name = ?`
	_, err := d.db.ExecContext(ctx, query, fullName)
	return err
}

// CountRepositories returns the total count of repositories with optional filters
func (d *Database) CountRepositories(ctx context.Context, filters map[string]interface{}) (int, error) {
	query := "SELECT COUNT(*) FROM repositories WHERE 1=1"
	args := []interface{}{}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if batchID, ok := filters["batch_id"].(int64); ok && batchID > 0 {
		query += " AND batch_id = ?"
		args = append(args, batchID)
	}

	if source, ok := filters["source"].(string); ok && source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}

	var count int
	err := d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// GetRepositoryStatsByStatus returns counts grouped by status
func (d *Database) GetRepositoryStatsByStatus(ctx context.Context) (map[string]int, error) {
	query := `SELECT status, COUNT(*) as count FROM repositories GROUP BY status`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, rows.Err()
}
