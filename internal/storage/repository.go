package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/models"
)

// SaveRepository inserts or updates a repository in the database
// nolint:dupl // SaveRepository and UpdateRepository have different SQL operations
func (d *Database) SaveRepository(ctx context.Context, repo *models.Repository) error {
	query := `
		INSERT INTO repositories (
			full_name, source, source_url, total_size, largest_file, 
			largest_file_size, largest_commit, largest_commit_size,
			has_lfs, has_submodules, has_large_files, large_file_count,
			default_branch, branch_count, commit_count, last_commit_sha,
			last_commit_date, has_wiki, has_pages, has_discussions, 
			has_actions, has_projects, branch_protections, 
			environment_count, secret_count, variable_count, 
			webhook_count, contributor_count, top_contributors,
			issue_count, pull_request_count, tag_count, 
			open_issue_count, open_pr_count,
			status, batch_id, priority, destination_url, 
			destination_full_name, discovered_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			has_large_files = excluded.has_large_files,
			large_file_count = excluded.large_file_count,
			default_branch = excluded.default_branch,
			branch_count = excluded.branch_count,
			commit_count = excluded.commit_count,
			last_commit_sha = excluded.last_commit_sha,
			last_commit_date = excluded.last_commit_date,
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
			issue_count = excluded.issue_count,
			pull_request_count = excluded.pull_request_count,
			tag_count = excluded.tag_count,
			open_issue_count = excluded.open_issue_count,
			open_pr_count = excluded.open_pr_count,
			destination_url = excluded.destination_url,
			destination_full_name = excluded.destination_full_name,
			updated_at = excluded.updated_at
	`

	_, err := d.db.ExecContext(ctx, query,
		repo.FullName, repo.Source, repo.SourceURL, repo.TotalSize,
		repo.LargestFile, repo.LargestFileSize, repo.LargestCommit,
		repo.LargestCommitSize, repo.HasLFS, repo.HasSubmodules,
		repo.HasLargeFiles, repo.LargeFileCount,
		repo.DefaultBranch, repo.BranchCount, repo.CommitCount,
		repo.LastCommitSHA, repo.LastCommitDate,
		repo.HasWiki, repo.HasPages, repo.HasDiscussions,
		repo.HasActions, repo.HasProjects, repo.BranchProtections,
		repo.EnvironmentCount, repo.SecretCount, repo.VariableCount,
		repo.WebhookCount, repo.ContributorCount, repo.TopContributors,
		repo.IssueCount, repo.PullRequestCount, repo.TagCount,
		repo.OpenIssueCount, repo.OpenPRCount,
		repo.Status, repo.BatchID, repo.Priority,
		repo.DestinationURL, repo.DestinationFullName,
		repo.DiscoveredAt, repo.UpdatedAt,
	)

	return err
}

// GetRepository retrieves a repository by full name
// nolint:dupl // Similar to GetRepositoryByID but queries by full_name
func (d *Database) GetRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	query := `
		SELECT id, full_name, source, source_url, total_size, largest_file, 
			   largest_file_size, largest_commit, largest_commit_size,
			   has_lfs, has_submodules, has_large_files, large_file_count,
			   default_branch, branch_count, commit_count, last_commit_sha,
			   last_commit_date, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   issue_count, pull_request_count, tag_count,
			   open_issue_count, open_pr_count,
			   status, batch_id, priority, destination_url, 
			   destination_full_name, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE full_name = ?
	`

	var repo models.Repository
	err := d.db.QueryRowContext(ctx, query, fullName).Scan(
		&repo.ID, &repo.FullName, &repo.Source, &repo.SourceURL,
		&repo.TotalSize, &repo.LargestFile, &repo.LargestFileSize,
		&repo.LargestCommit, &repo.LargestCommitSize, &repo.HasLFS,
		&repo.HasSubmodules, &repo.HasLargeFiles, &repo.LargeFileCount,
		&repo.DefaultBranch, &repo.BranchCount, &repo.CommitCount,
		&repo.LastCommitSHA, &repo.LastCommitDate,
		&repo.HasWiki, &repo.HasPages, &repo.HasDiscussions,
		&repo.HasActions, &repo.HasProjects, &repo.BranchProtections,
		&repo.EnvironmentCount, &repo.SecretCount, &repo.VariableCount,
		&repo.WebhookCount, &repo.ContributorCount, &repo.TopContributors,
		&repo.IssueCount, &repo.PullRequestCount, &repo.TagCount,
		&repo.OpenIssueCount, &repo.OpenPRCount,
		&repo.Status, &repo.BatchID, &repo.Priority,
		&repo.DestinationURL, &repo.DestinationFullName,
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

const (
	orderByFullNameASC = " ORDER BY full_name ASC"
)

// applyRepositoryFilters applies filters to the SQL query and returns the updated query and args
func applyRepositoryFilters(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	// Apply status filter
	query, args = applyStatusFilter(query, args, filters)

	// Apply simple filters
	query, args = applySimpleFilters(query, args, filters)

	// Apply organization filter
	query, args = applyOrganizationFilter(query, args, filters)

	// Apply feature filters
	query, args = applyFeatureFilters(query, args, filters)

	// Apply available for batch filter
	query, args = applyAvailableForBatchFilter(query, args, filters)

	// Apply sort order
	query, args = applySortOrder(query, args, filters)

	return query, args
}

// applySimpleFilters applies basic filters like batch_id, source, size, and search
func applySimpleFilters(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	// Apply batch_id filter
	if batchID, ok := filters["batch_id"].(int64); ok && batchID > 0 {
		query += " AND batch_id = ?"
		args = append(args, batchID)
	}

	// Apply source filter
	if source, ok := filters["source"].(string); ok && source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}

	// Apply size range filters
	if minSize, ok := filters["min_size"].(int64); ok && minSize > 0 {
		query += " AND total_size >= ?"
		args = append(args, minSize)
	}
	if maxSize, ok := filters["max_size"].(int64); ok && maxSize > 0 {
		query += " AND total_size <= ?"
		args = append(args, maxSize)
	}

	// Apply search filter (case-insensitive)
	if search, ok := filters["search"].(string); ok && search != "" {
		query += " AND LOWER(full_name) LIKE LOWER(?)"
		args = append(args, "%"+search+"%")
	}

	return query, args
}

// applyOrganizationFilter applies organization filter (single or multiple)
func applyOrganizationFilter(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	orgValue, ok := filters["organization"]
	if !ok {
		return query, args
	}

	switch org := orgValue.(type) {
	case string:
		if org != "" {
			query += " AND LOWER(full_name) LIKE LOWER(?)"
			args = append(args, org+"/%")
		}
	case []string:
		if len(org) > 0 {
			placeholders := make([]string, len(org))
			for i, o := range org {
				placeholders[i] = "LOWER(full_name) LIKE LOWER(?)"
				args = append(args, o+"/%")
			}
			query += fmt.Sprintf(" AND (%s)", strings.Join(placeholders, " OR "))
		}
	}

	return query, args
}

// applyFeatureFilters applies feature-based filters
func applyFeatureFilters(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	featureFilters := []struct {
		key    string
		column string
	}{
		{"has_lfs", "has_lfs"},
		{"has_submodules", "has_submodules"},
		{"has_actions", "has_actions"},
		{"has_wiki", "has_wiki"},
		{"has_pages", "has_pages"},
		{"is_archived", "is_archived"},
	}

	for _, f := range featureFilters {
		if value, ok := filters[f.key].(bool); ok {
			query += fmt.Sprintf(" AND %s = ?", f.column)
			args = append(args, value)
		}
	}

	return query, args
}

// applyAvailableForBatchFilter excludes repos that are not eligible for batch assignment
func applyAvailableForBatchFilter(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	availableForBatch, ok := filters["available_for_batch"].(bool)
	if !ok || !availableForBatch {
		return query, args
	}

	// Exclude repos that are completed or in active migration
	excludedStatuses := []string{
		"complete",
		"queued_for_migration",
		"dry_run_in_progress",
		"dry_run_queued",
		"migrating_content",
		"archive_generating",
		"post_migration",
		"migration_complete",
	}
	placeholders := make([]string, len(excludedStatuses))
	for i, status := range excludedStatuses {
		placeholders[i] = "?"
		args = append(args, status)
	}
	query += fmt.Sprintf(" AND status NOT IN (%s)", strings.Join(placeholders, ","))

	return query, args
}

// applySortOrder applies the sort order to the query
func applySortOrder(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	sortBy, ok := filters["sort_by"].(string)
	if !ok || sortBy == "" {
		return query, args
	}

	switch sortBy {
	case "name":
		query += orderByFullNameASC
	case "size":
		query += " ORDER BY total_size DESC"
	case "org":
		query += orderByFullNameASC // Already sorts by org/repo
	case "updated":
		query += " ORDER BY updated_at DESC"
	default:
		query += orderByFullNameASC
	}

	// Mark as sorted to prevent double ORDER BY
	filters["_sorted"] = true

	return query, args
}

// applyStatusFilter handles the status filter which can be a string or slice of strings
func applyStatusFilter(query string, args []interface{}, filters map[string]interface{}) (string, []interface{}) {
	statusValue, ok := filters["status"]
	if !ok {
		return query, args
	}

	switch status := statusValue.(type) {
	case string:
		if status != "" {
			query += " AND status = ?"
			args = append(args, status)
		}
	case []string:
		if len(status) > 0 {
			placeholders := make([]string, len(status))
			for i, s := range status {
				placeholders[i] = "?"
				args = append(args, s)
			}
			query += fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ","))
		}
	}

	return query, args
}

// ListRepositories retrieves repositories with optional filters
func (d *Database) ListRepositories(ctx context.Context, filters map[string]interface{}) ([]*models.Repository, error) {
	query := `
		SELECT id, full_name, source, source_url, total_size, largest_file, 
			   largest_file_size, largest_commit, largest_commit_size,
			   has_lfs, has_submodules, has_large_files, large_file_count,
			   default_branch, branch_count, commit_count, last_commit_sha,
			   last_commit_date, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   issue_count, pull_request_count, tag_count,
			   open_issue_count, open_pr_count,
			   status, batch_id, priority, destination_url, 
			   destination_full_name, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE 1=1
	`
	args := []interface{}{}

	// Apply filters dynamically
	query, args = applyRepositoryFilters(query, args, filters)

	// Add ordering if not already added by filters
	if _, sorted := filters["_sorted"]; !sorted {
		query += orderByFullNameASC
	}

	// Add limit and offset if specified
	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)

		if offset, ok := filters["offset"].(int); ok && offset > 0 {
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	repos := []*models.Repository{}
	for rows.Next() {
		var repo models.Repository
		// nolint:dupl // Standard repository scanning, duplication expected
		err := rows.Scan(
			&repo.ID, &repo.FullName, &repo.Source, &repo.SourceURL,
			&repo.TotalSize, &repo.LargestFile, &repo.LargestFileSize,
			&repo.LargestCommit, &repo.LargestCommitSize, &repo.HasLFS,
			&repo.HasSubmodules, &repo.HasLargeFiles, &repo.LargeFileCount,
			&repo.DefaultBranch, &repo.BranchCount, &repo.CommitCount,
			&repo.LastCommitSHA, &repo.LastCommitDate,
			&repo.HasWiki, &repo.HasPages, &repo.HasDiscussions,
			&repo.HasActions, &repo.HasProjects, &repo.BranchProtections,
			&repo.EnvironmentCount, &repo.SecretCount, &repo.VariableCount,
			&repo.WebhookCount, &repo.ContributorCount, &repo.TopContributors,
			&repo.IssueCount, &repo.PullRequestCount, &repo.TagCount,
			&repo.OpenIssueCount, &repo.OpenPRCount,
			&repo.Status, &repo.BatchID, &repo.Priority,
			&repo.DestinationURL, &repo.DestinationFullName,
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
			has_large_files = ?,
			large_file_count = ?,
			default_branch = ?,
			branch_count = ?,
			commit_count = ?,
			last_commit_sha = ?,
			last_commit_date = ?,
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
			issue_count = ?,
			pull_request_count = ?,
			tag_count = ?,
			open_issue_count = ?,
			open_pr_count = ?,
			status = ?,
			batch_id = ?,
			priority = ?,
			destination_url = ?,
			destination_full_name = ?,
			updated_at = ?,
			migrated_at = ?
		WHERE full_name = ?
	`

	_, err := d.db.ExecContext(ctx, query,
		repo.Source, repo.SourceURL, repo.TotalSize,
		repo.LargestFile, repo.LargestFileSize, repo.LargestCommit,
		repo.LargestCommitSize, repo.HasLFS, repo.HasSubmodules,
		repo.HasLargeFiles, repo.LargeFileCount,
		repo.DefaultBranch, repo.BranchCount, repo.CommitCount,
		repo.LastCommitSHA, repo.LastCommitDate,
		repo.HasWiki, repo.HasPages, repo.HasDiscussions,
		repo.HasActions, repo.HasProjects, repo.BranchProtections,
		repo.EnvironmentCount, repo.SecretCount, repo.VariableCount,
		repo.WebhookCount, repo.ContributorCount, repo.TopContributors,
		repo.IssueCount, repo.PullRequestCount, repo.TagCount,
		repo.OpenIssueCount, repo.OpenPRCount,
		repo.Status, repo.BatchID, repo.Priority,
		repo.DestinationURL, repo.DestinationFullName,
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

	// Handle status as either string or slice of strings
	if statusValue, ok := filters["status"]; ok {
		switch status := statusValue.(type) {
		case string:
			if status != "" {
				query += " AND status = ?"
				args = append(args, status)
			}
		case []string:
			if len(status) > 0 {
				placeholders := make([]string, len(status))
				for i, s := range status {
					placeholders[i] = "?"
					args = append(args, s)
				}
				query += fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ","))
			}
		}
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

// GetRepositoriesByIDs retrieves multiple repositories by their IDs
// nolint:dupl // Similar to GetRepositoriesByNames but operates on IDs
func (d *Database) GetRepositoriesByIDs(ctx context.Context, ids []int64) ([]*models.Repository, error) {
	if len(ids) == 0 {
		return []*models.Repository{}, nil
	}

	// Build IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	//nolint:gosec // G201: Safe use of fmt.Sprintf with placeholders for IN clause
	query := fmt.Sprintf(`
		SELECT id, full_name, source, source_url, total_size, largest_file, 
			   largest_file_size, largest_commit, largest_commit_size,
			   has_lfs, has_submodules, has_large_files, large_file_count,
			   default_branch, branch_count, commit_count, last_commit_sha,
			   last_commit_date, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   issue_count, pull_request_count, tag_count,
			   open_issue_count, open_pr_count,
			   status, batch_id, priority, destination_url, 
			   destination_full_name, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by IDs: %w", err)
	}
	defer rows.Close()

	return d.scanRepositories(rows)
}

// GetRepositoriesByNames retrieves multiple repositories by their full names
// nolint:dupl // Similar to GetRepositoriesByIDs but operates on names
func (d *Database) GetRepositoriesByNames(ctx context.Context, names []string) ([]*models.Repository, error) {
	if len(names) == 0 {
		return []*models.Repository{}, nil
	}

	placeholders := make([]string, len(names))
	args := make([]interface{}, len(names))
	for i, name := range names {
		placeholders[i] = "?"
		args[i] = name
	}

	//nolint:gosec // G201: Safe use of fmt.Sprintf with placeholders for IN clause
	query := fmt.Sprintf(`
		SELECT id, full_name, source, source_url, total_size, largest_file, 
			   largest_file_size, largest_commit, largest_commit_size,
			   has_lfs, has_submodules, has_large_files, large_file_count,
			   default_branch, branch_count, commit_count, last_commit_sha,
			   last_commit_date, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   issue_count, pull_request_count, tag_count,
			   open_issue_count, open_pr_count,
			   status, batch_id, priority, destination_url, 
			   destination_full_name, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE full_name IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by names: %w", err)
	}
	defer rows.Close()

	return d.scanRepositories(rows)
}

// GetRepositoryByID retrieves a repository by ID
// nolint:dupl // Similar to GetRepository but operates on ID instead of fullName
func (d *Database) GetRepositoryByID(ctx context.Context, id int64) (*models.Repository, error) {
	query := `
		SELECT id, full_name, source, source_url, total_size, largest_file, 
			   largest_file_size, largest_commit, largest_commit_size,
			   has_lfs, has_submodules, has_large_files, large_file_count,
			   default_branch, branch_count, commit_count, last_commit_sha,
			   last_commit_date, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   issue_count, pull_request_count, tag_count,
			   open_issue_count, open_pr_count,
			   status, batch_id, priority, destination_url, 
			   destination_full_name, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE id = ?
	`

	var repo models.Repository
	err := d.db.QueryRowContext(ctx, query, id).Scan(
		&repo.ID, &repo.FullName, &repo.Source, &repo.SourceURL,
		&repo.TotalSize, &repo.LargestFile, &repo.LargestFileSize,
		&repo.LargestCommit, &repo.LargestCommitSize, &repo.HasLFS,
		&repo.HasSubmodules, &repo.HasLargeFiles, &repo.LargeFileCount,
		&repo.DefaultBranch, &repo.BranchCount, &repo.CommitCount,
		&repo.LastCommitSHA, &repo.LastCommitDate,
		&repo.HasWiki, &repo.HasPages, &repo.HasDiscussions,
		&repo.HasActions, &repo.HasProjects, &repo.BranchProtections,
		&repo.EnvironmentCount, &repo.SecretCount, &repo.VariableCount,
		&repo.WebhookCount, &repo.ContributorCount, &repo.TopContributors,
		&repo.IssueCount, &repo.PullRequestCount, &repo.TagCount,
		&repo.OpenIssueCount, &repo.OpenPRCount,
		&repo.Status, &repo.BatchID, &repo.Priority,
		&repo.DestinationURL, &repo.DestinationFullName,
		&repo.DiscoveredAt, &repo.UpdatedAt, &repo.MigratedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get repository by ID: %w", err)
	}

	return &repo, nil
}

// GetMigrationHistory retrieves migration history for a repository
func (d *Database) GetMigrationHistory(ctx context.Context, repoID int64) ([]*models.MigrationHistory, error) {
	query := `
		SELECT id, repository_id, status, phase, message, error_message, 
			   started_at, completed_at, duration_seconds
		FROM migration_history 
		WHERE repository_id = ? 
		ORDER BY started_at DESC
	`

	rows, err := d.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}
	defer rows.Close()

	var history []*models.MigrationHistory
	for rows.Next() {
		var h models.MigrationHistory
		if err := rows.Scan(
			&h.ID, &h.RepositoryID, &h.Status, &h.Phase,
			&h.Message, &h.ErrorMessage, &h.StartedAt,
			&h.CompletedAt, &h.DurationSeconds,
		); err != nil {
			return nil, fmt.Errorf("failed to scan migration history: %w", err)
		}
		history = append(history, &h)
	}

	return history, rows.Err()
}

// GetMigrationLogs retrieves detailed logs for a repository's migration operations
func (d *Database) GetMigrationLogs(ctx context.Context, repoID int64, level, phase string, limit, offset int) ([]*models.MigrationLog, error) {
	query := `
		SELECT id, repository_id, history_id, level, phase, operation, message, details, timestamp
		FROM migration_logs 
		WHERE repository_id = ?
	`
	args := []interface{}{repoID}

	// Add optional filters
	if level != "" {
		query += " AND level = ?"
		args = append(args, level)
	}
	if phase != "" {
		query += " AND phase = ?"
		args = append(args, phase)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.MigrationLog
	for rows.Next() {
		var log models.MigrationLog
		if err := rows.Scan(
			&log.ID, &log.RepositoryID, &log.HistoryID, &log.Level,
			&log.Phase, &log.Operation, &log.Message, &log.Details,
			&log.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan migration log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, rows.Err()
}

// GetBatch retrieves a batch by ID
func (d *Database) GetBatch(ctx context.Context, id int64) (*models.Batch, error) {
	query := `
		SELECT id, name, description, type, repository_count, status, 
			   scheduled_at, started_at, completed_at, created_at
		FROM batches 
		WHERE id = ?
	`

	var batch models.Batch
	err := d.db.QueryRowContext(ctx, query, id).Scan(
		&batch.ID, &batch.Name, &batch.Description, &batch.Type,
		&batch.RepositoryCount, &batch.Status, &batch.ScheduledAt,
		&batch.StartedAt, &batch.CompletedAt, &batch.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	return &batch, nil
}

// UpdateBatch updates a batch
func (d *Database) UpdateBatch(ctx context.Context, batch *models.Batch) error {
	query := `
		UPDATE batches SET
			name = ?, description = ?, type = ?, repository_count = ?,
			status = ?, scheduled_at = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`

	_, err := d.db.ExecContext(ctx, query,
		batch.Name, batch.Description, batch.Type, batch.RepositoryCount,
		batch.Status, batch.ScheduledAt, batch.StartedAt, batch.CompletedAt,
		batch.ID,
	)

	return err
}

// ListBatches retrieves all batches
func (d *Database) ListBatches(ctx context.Context) ([]*models.Batch, error) {
	query := `
		SELECT id, name, description, type, repository_count, status, 
			   scheduled_at, started_at, completed_at, created_at
		FROM batches 
		ORDER BY created_at DESC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}
	defer rows.Close()

	batches := []*models.Batch{}
	for rows.Next() {
		var batch models.Batch
		if err := rows.Scan(
			&batch.ID, &batch.Name, &batch.Description, &batch.Type,
			&batch.RepositoryCount, &batch.Status, &batch.ScheduledAt,
			&batch.StartedAt, &batch.CompletedAt, &batch.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch: %w", err)
		}
		batches = append(batches, &batch)
	}

	return batches, rows.Err()
}

// CreateBatch creates a new batch
func (d *Database) CreateBatch(ctx context.Context, batch *models.Batch) error {
	query := `
		INSERT INTO batches (name, description, type, repository_count, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.ExecContext(ctx, query,
		batch.Name, batch.Description, batch.Type,
		batch.RepositoryCount, batch.Status, batch.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get batch ID: %w", err)
	}

	batch.ID = id
	return nil
}

// scanRepositories is a helper to scan multiple repositories from rows
// nolint:dupl // Expected duplication in scanning logic for consistency
func (d *Database) scanRepositories(rows *sql.Rows) ([]*models.Repository, error) {
	var repos []*models.Repository
	for rows.Next() {
		var repo models.Repository
		if err := rows.Scan(
			&repo.ID, &repo.FullName, &repo.Source, &repo.SourceURL,
			&repo.TotalSize, &repo.LargestFile, &repo.LargestFileSize,
			&repo.LargestCommit, &repo.LargestCommitSize, &repo.HasLFS,
			&repo.HasSubmodules, &repo.HasLargeFiles, &repo.LargeFileCount,
			&repo.DefaultBranch, &repo.BranchCount, &repo.CommitCount,
			&repo.LastCommitSHA, &repo.LastCommitDate,
			&repo.HasWiki, &repo.HasPages, &repo.HasDiscussions,
			&repo.HasActions, &repo.HasProjects, &repo.BranchProtections,
			&repo.EnvironmentCount, &repo.SecretCount, &repo.VariableCount,
			&repo.WebhookCount, &repo.ContributorCount, &repo.TopContributors,
			&repo.IssueCount, &repo.PullRequestCount, &repo.TagCount,
			&repo.OpenIssueCount, &repo.OpenPRCount,
			&repo.Status, &repo.BatchID, &repo.Priority,
			&repo.DestinationURL, &repo.DestinationFullName,
			&repo.DiscoveredAt, &repo.UpdatedAt, &repo.MigratedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repos = append(repos, &repo)
	}
	return repos, rows.Err()
}

// CreateMigrationHistory creates a new migration history record
func (d *Database) CreateMigrationHistory(ctx context.Context, history *models.MigrationHistory) (int64, error) {
	query := `
		INSERT INTO migration_history (repository_id, status, phase, message, error_message, started_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.ExecContext(ctx, query,
		history.RepositoryID,
		history.Status,
		history.Phase,
		history.Message,
		history.ErrorMessage,
		history.StartedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create migration history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get migration history ID: %w", err)
	}

	return id, nil
}

// UpdateMigrationHistory updates a migration history record
func (d *Database) UpdateMigrationHistory(ctx context.Context, id int64, status string, errorMsg *string) error {
	completedAt := time.Now()

	// Calculate duration
	var startedAt time.Time
	err := d.db.QueryRowContext(ctx, "SELECT started_at FROM migration_history WHERE id = ?", id).Scan(&startedAt)
	if err != nil {
		return fmt.Errorf("failed to get started_at time: %w", err)
	}

	durationSeconds := int(completedAt.Sub(startedAt).Seconds())

	query := `
		UPDATE migration_history 
		SET status = ?, error_message = ?, completed_at = ?, duration_seconds = ?
		WHERE id = ?
	`

	_, err = d.db.ExecContext(ctx, query, status, errorMsg, completedAt, durationSeconds, id)
	if err != nil {
		return fmt.Errorf("failed to update migration history: %w", err)
	}

	return nil
}

// CreateMigrationLog creates a new migration log entry
func (d *Database) CreateMigrationLog(ctx context.Context, log *models.MigrationLog) error {
	query := `
		INSERT INTO migration_logs (repository_id, history_id, level, phase, operation, message, details, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.ExecContext(ctx, query,
		log.RepositoryID,
		log.HistoryID,
		log.Level,
		log.Phase,
		log.Operation,
		log.Message,
		log.Details,
		log.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration log: %w", err)
	}

	return nil
}

// AddRepositoriesToBatch assigns multiple repositories to a batch
//
//nolint:dupl // Similar to RemoveRepositoriesFromBatch but performs different operations
func (d *Database) AddRepositoriesToBatch(ctx context.Context, batchID int64, repoIDs []int64) error {
	if len(repoIDs) == 0 {
		return nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(repoIDs))
	args := make([]interface{}, len(repoIDs)+1)
	args[0] = batchID
	for i, id := range repoIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}

	//nolint:gosec // G201: Safe use of fmt.Sprintf with placeholders for IN clause
	query := fmt.Sprintf(`
		UPDATE repositories 
		SET batch_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to add repositories to batch: %w", err)
	}

	// Update batch repository count
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		if err := d.updateBatchRepositoryCount(ctx, batchID); err != nil {
			return err
		}
	}

	return nil
}

// RemoveRepositoriesFromBatch removes repositories from a batch
//
//nolint:dupl // Similar to AddRepositoriesToBatch but performs different operations
func (d *Database) RemoveRepositoriesFromBatch(ctx context.Context, batchID int64, repoIDs []int64) error {
	if len(repoIDs) == 0 {
		return nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(repoIDs))
	args := make([]interface{}, len(repoIDs)+1)
	args[0] = batchID
	for i, id := range repoIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}

	//nolint:gosec // G201: Safe use of fmt.Sprintf with placeholders for IN clause
	query := fmt.Sprintf(`
		UPDATE repositories 
		SET batch_id = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE batch_id = ? AND id IN (%s)
	`, strings.Join(placeholders, ","))

	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to remove repositories from batch: %w", err)
	}

	// Update batch repository count
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		if err := d.updateBatchRepositoryCount(ctx, batchID); err != nil {
			return err
		}
	}

	return nil
}

// updateBatchRepositoryCount updates the repository count for a batch
func (d *Database) updateBatchRepositoryCount(ctx context.Context, batchID int64) error {
	query := `
		UPDATE batches 
		SET repository_count = (
			SELECT COUNT(*) FROM repositories WHERE batch_id = ?
		)
		WHERE id = ?
	`

	_, err := d.db.ExecContext(ctx, query, batchID, batchID)
	if err != nil {
		return fmt.Errorf("failed to update batch repository count: %w", err)
	}

	return nil
}

// OrganizationStats represents statistics for a single organization
type OrganizationStats struct {
	Organization string         `json:"organization"`
	TotalRepos   int            `json:"total_repos"`
	StatusCounts map[string]int `json:"status_counts"`
}

// GetOrganizationStats returns repository counts grouped by organization
func (d *Database) GetOrganizationStats(ctx context.Context) ([]*OrganizationStats, error) {
	// First, get unique organizations with their total counts
	query := `
		SELECT 
			SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as org,
			COUNT(*) as total,
			status,
			COUNT(*) as status_count
		FROM repositories
		WHERE INSTR(full_name, '/') > 0
		GROUP BY org, status
		ORDER BY total DESC, org ASC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization stats: %w", err)
	}
	defer rows.Close()

	// Build organization stats map
	orgMap := make(map[string]*OrganizationStats)
	for rows.Next() {
		var org, status string
		var total, statusCount int
		if err := rows.Scan(&org, &total, &status, &statusCount); err != nil {
			return nil, fmt.Errorf("failed to scan organization stats: %w", err)
		}

		if _, exists := orgMap[org]; !exists {
			orgMap[org] = &OrganizationStats{
				Organization: org,
				TotalRepos:   0,
				StatusCounts: make(map[string]int),
			}
		}

		orgMap[org].StatusCounts[status] = statusCount
		orgMap[org].TotalRepos += statusCount
	}

	// Convert map to slice
	stats := make([]*OrganizationStats, 0, len(orgMap))
	for _, stat := range orgMap {
		stats = append(stats, stat)
	}

	// Sort by total repos (descending), then by organization name (ascending) for consistent ordering
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalRepos == stats[j].TotalRepos {
			return stats[i].Organization < stats[j].Organization
		}
		return stats[i].TotalRepos > stats[j].TotalRepos
	})

	return stats, rows.Err()
}

// SizeDistribution represents repository size distribution
type SizeDistribution struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetSizeDistribution categorizes repositories by size
func (d *Database) GetSizeDistribution(ctx context.Context) ([]*SizeDistribution, error) {
	// Size categories: small (<100MB), medium (100MB-1GB), large (1GB-5GB), very_large (>5GB)
	query := `
		SELECT 
			CASE 
				WHEN total_size IS NULL THEN 'unknown'
				WHEN total_size < 104857600 THEN 'small'
				WHEN total_size < 1073741824 THEN 'medium'
				WHEN total_size < 5368709120 THEN 'large'
				ELSE 'very_large'
			END as category,
			COUNT(*) as count
		FROM repositories
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'small' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'large' THEN 3
				WHEN 'very_large' THEN 4
				WHEN 'unknown' THEN 5
			END
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get size distribution: %w", err)
	}
	defer rows.Close()

	var distribution []*SizeDistribution
	for rows.Next() {
		var dist SizeDistribution
		if err := rows.Scan(&dist.Category, &dist.Count); err != nil {
			return nil, fmt.Errorf("failed to scan size distribution: %w", err)
		}
		distribution = append(distribution, &dist)
	}

	return distribution, rows.Err()
}

// FeatureStats represents aggregated feature usage statistics
type FeatureStats struct {
	IsArchived           int `json:"is_archived"`
	HasLFS               int `json:"has_lfs"`
	HasSubmodules        int `json:"has_submodules"`
	HasLargeFiles        int `json:"has_large_files"`
	HasWiki              int `json:"has_wiki"`
	HasPages             int `json:"has_pages"`
	HasDiscussions       int `json:"has_discussions"`
	HasActions           int `json:"has_actions"`
	HasProjects          int `json:"has_projects"`
	HasBranchProtections int `json:"has_branch_protections"`
	TotalRepositories    int `json:"total_repositories"`
}

// GetFeatureStats returns aggregated statistics on feature usage
func (d *Database) GetFeatureStats(ctx context.Context) (*FeatureStats, error) {
	query := `
		SELECT 
			SUM(CASE WHEN is_archived = 1 THEN 1 ELSE 0 END) as archived_count,
			SUM(CASE WHEN has_lfs = 1 THEN 1 ELSE 0 END) as lfs_count,
			SUM(CASE WHEN has_submodules = 1 THEN 1 ELSE 0 END) as submodules_count,
			SUM(CASE WHEN has_large_files = 1 THEN 1 ELSE 0 END) as large_files_count,
			SUM(CASE WHEN has_wiki = 1 THEN 1 ELSE 0 END) as wiki_count,
			SUM(CASE WHEN has_pages = 1 THEN 1 ELSE 0 END) as pages_count,
			SUM(CASE WHEN has_discussions = 1 THEN 1 ELSE 0 END) as discussions_count,
			SUM(CASE WHEN has_actions = 1 THEN 1 ELSE 0 END) as actions_count,
			SUM(CASE WHEN has_projects = 1 THEN 1 ELSE 0 END) as projects_count,
			SUM(CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END) as branch_protections_count,
			COUNT(*) as total
		FROM repositories
	`

	var stats FeatureStats
	err := d.db.QueryRowContext(ctx, query).Scan(
		&stats.IsArchived,
		&stats.HasLFS,
		&stats.HasSubmodules,
		&stats.HasLargeFiles,
		&stats.HasWiki,
		&stats.HasPages,
		&stats.HasDiscussions,
		&stats.HasActions,
		&stats.HasProjects,
		&stats.HasBranchProtections,
		&stats.TotalRepositories,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature stats: %w", err)
	}

	return &stats, nil
}

// MigrationCompletionStats represents migration completion stats by organization
type MigrationCompletionStats struct {
	Organization    string `json:"organization"`
	TotalRepos      int    `json:"total_repos"`
	CompletedCount  int    `json:"completed_count"`
	InProgressCount int    `json:"in_progress_count"`
	PendingCount    int    `json:"pending_count"`
	FailedCount     int    `json:"failed_count"`
}

// CompletedMigration represents a completed migration for the history page
type CompletedMigration struct {
	ID              int64      `json:"id"`
	FullName        string     `json:"full_name"`
	SourceURL       string     `json:"source_url"`
	DestinationURL  *string    `json:"destination_url"`
	Status          string     `json:"status"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	DurationSeconds *int       `json:"duration_seconds"`
}

// GetCompletedMigrations returns all completed, failed, and rolled back migrations
func (d *Database) GetCompletedMigrations(ctx context.Context) ([]*CompletedMigration, error) {
	query := `
		SELECT 
			r.id,
			r.full_name,
			r.source_url,
			r.destination_url,
			r.status,
			r.migrated_at,
			h.started_at,
			h.completed_at,
			h.duration_seconds
		FROM repositories r
		LEFT JOIN (
			SELECT 
				repository_id,
				MIN(started_at) as started_at,
				MAX(completed_at) as completed_at,
				SUM(duration_seconds) as duration_seconds
			FROM migration_history
			WHERE phase IN ('migration', 'rollback') 
			GROUP BY repository_id
		) h ON r.id = h.repository_id
		WHERE r.status IN ('complete', 'migration_failed', 'rolled_back')
		ORDER BY r.migrated_at DESC, r.updated_at DESC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed migrations: %w", err)
	}
	defer rows.Close()

	var migrations []*CompletedMigration
	for rows.Next() {
		var m CompletedMigration
		var migratedAt *time.Time
		var startedAtStr, completedAtStr sql.NullString

		if err := rows.Scan(
			&m.ID,
			&m.FullName,
			&m.SourceURL,
			&m.DestinationURL,
			&m.Status,
			&migratedAt,
			&startedAtStr,
			&completedAtStr,
			&m.DurationSeconds,
		); err != nil {
			return nil, fmt.Errorf("failed to scan completed migration: %w", err)
		}

		// Parse timestamp strings to time.Time
		if startedAtStr.Valid {
			t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", startedAtStr.String)
			if err != nil {
				// Try alternative format
				t, err = time.Parse("2006-01-02 15:04:05", startedAtStr.String)
			}
			if err == nil {
				m.StartedAt = &t
			}
		}

		if completedAtStr.Valid {
			t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", completedAtStr.String)
			if err != nil {
				// Try alternative format
				t, err = time.Parse("2006-01-02 15:04:05", completedAtStr.String)
			}
			if err == nil {
				m.CompletedAt = &t
			}
		}

		migrations = append(migrations, &m)
	}

	return migrations, rows.Err()
}

// GetMigrationCompletionStatsByOrg returns migration completion stats grouped by organization
func (d *Database) GetMigrationCompletionStatsByOrg(ctx context.Context) ([]*MigrationCompletionStats, error) {
	query := `
		SELECT 
			SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as organization,
			COUNT(*) as total_repos,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as completed_count,
			SUM(CASE WHEN status IN ('in_progress', 'pre_migration', 'queued_for_migration', 'migrating_content') THEN 1 ELSE 0 END) as in_progress_count,
			SUM(CASE WHEN status IN ('pending', 'discovered') THEN 1 ELSE 0 END) as pending_count,
			SUM(CASE WHEN status LIKE '%failed%' THEN 1 ELSE 0 END) as failed_count
		FROM repositories
		WHERE full_name LIKE '%/%'
		GROUP BY organization
		ORDER BY total_repos DESC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration completion stats: %w", err)
	}
	defer rows.Close()

	var stats []*MigrationCompletionStats
	for rows.Next() {
		var s MigrationCompletionStats
		if err := rows.Scan(
			&s.Organization,
			&s.TotalRepos,
			&s.CompletedCount,
			&s.InProgressCount,
			&s.PendingCount,
			&s.FailedCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan migration completion stats: %w", err)
		}
		stats = append(stats, &s)
	}

	return stats, rows.Err()
}

// RollbackRepository marks a repository as rolled back and creates a migration history entry
func (d *Database) RollbackRepository(ctx context.Context, fullName string, reason string) error {
	// Get the repository
	repo, err := d.GetRepository(ctx, fullName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}
	if repo == nil {
		return fmt.Errorf("repository not found")
	}

	// Update repository status to rolled_back
	query := `UPDATE repositories SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE full_name = ?`
	_, err = d.db.ExecContext(ctx, query, string(models.StatusRolledBack), fullName)
	if err != nil {
		return fmt.Errorf("failed to update repository status: %w", err)
	}

	// Create migration history entry for rollback
	message := "Repository rolled back"
	if reason != "" {
		message = reason
	}

	historyQuery := `
		INSERT INTO migration_history (repository_id, status, phase, message, started_at, completed_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err = d.db.ExecContext(ctx, historyQuery, repo.ID, "rolled_back", "rollback", message)
	if err != nil {
		return fmt.Errorf("failed to create rollback history: %w", err)
	}

	return nil
}

// ComplexityDistribution represents repository complexity distribution
type ComplexityDistribution struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetComplexityDistribution categorizes repositories by complexity score
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetComplexityDistribution(ctx context.Context, orgFilter, batchFilter string) ([]*ComplexityDistribution, error) {
	// Calculate complexity score based on:
	// Size (weight: 3), LFS (weight: 2), Submodules (weight: 2), Large files (weight: 2), Branch protections (weight: 1)
	//nolint:gosec // G202: Filter values are sanitized by buildOrgFilter and buildBatchFilter
	query := `
		SELECT 
			CASE 
				WHEN complexity_score <= 3 THEN 'low'
				WHEN complexity_score <= 6 THEN 'medium'
				WHEN complexity_score <= 9 THEN 'high'
				ELSE 'very_high'
			END as category,
			COUNT(*) as count
		FROM (
			SELECT 
				(CASE 
					WHEN total_size IS NULL THEN 0
					WHEN total_size < 104857600 THEN 0
					WHEN total_size < 1073741824 THEN 1
					WHEN total_size < 5368709120 THEN 2
					ELSE 3
				END) * 3 +
				(CASE WHEN has_lfs = 1 THEN 2 ELSE 0 END) +
				(CASE WHEN has_submodules = 1 THEN 2 ELSE 0 END) +
				(CASE WHEN has_large_files = 1 THEN 2 ELSE 0 END) +
				(CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END) as complexity_score
			FROM repositories r
			WHERE 1=1
				` + d.buildOrgFilter(orgFilter) + `
				` + d.buildBatchFilter(batchFilter) + `
		) as scored_repos
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'low' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'high' THEN 3
				WHEN 'very_high' THEN 4
			END
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get complexity distribution: %w", err)
	}
	defer rows.Close()

	var distribution []*ComplexityDistribution
	for rows.Next() {
		var dist ComplexityDistribution
		if err := rows.Scan(&dist.Category, &dist.Count); err != nil {
			return nil, fmt.Errorf("failed to scan complexity distribution: %w", err)
		}
		distribution = append(distribution, &dist)
	}

	return distribution, rows.Err()
}

// MigrationVelocity represents migration velocity metrics
type MigrationVelocity struct {
	ReposPerDay  float64 `json:"repos_per_day"`
	ReposPerWeek float64 `json:"repos_per_week"`
}

// GetMigrationVelocity calculates migration velocity over the specified period
func (d *Database) GetMigrationVelocity(ctx context.Context, orgFilter, batchFilter string, days int) (*MigrationVelocity, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildOrgFilter and buildBatchFilter
	query := `
		SELECT COUNT(DISTINCT r.id) as total_completed
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed' 
			AND mh.phase = 'migration'
			AND mh.completed_at >= datetime('now', '-' || ? || ' days')
			` + d.buildOrgFilter(orgFilter) + `
			` + d.buildBatchFilter(batchFilter) + `
	`

	var totalCompleted int
	err := d.db.QueryRowContext(ctx, query, days).Scan(&totalCompleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration velocity: %w", err)
	}

	velocity := &MigrationVelocity{
		ReposPerDay:  float64(totalCompleted) / float64(days),
		ReposPerWeek: (float64(totalCompleted) / float64(days)) * 7,
	}

	return velocity, nil
}

// MigrationTimeSeriesPoint represents a point in the migration time series
type MigrationTimeSeriesPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetMigrationTimeSeries returns daily migration completions for the last 30 days
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetMigrationTimeSeries(ctx context.Context, orgFilter, batchFilter string) ([]*MigrationTimeSeriesPoint, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildOrgFilter and buildBatchFilter
	query := `
		SELECT 
			DATE(mh.completed_at) as date,
			COUNT(DISTINCT r.id) as count
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed'
			AND mh.phase = 'migration'
			AND mh.completed_at >= datetime('now', '-30 days')
			` + d.buildOrgFilter(orgFilter) + `
			` + d.buildBatchFilter(batchFilter) + `
		GROUP BY DATE(mh.completed_at)
		ORDER BY date ASC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration time series: %w", err)
	}
	defer rows.Close()

	var series []*MigrationTimeSeriesPoint
	for rows.Next() {
		var point MigrationTimeSeriesPoint
		if err := rows.Scan(&point.Date, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan time series point: %w", err)
		}
		series = append(series, &point)
	}

	return series, rows.Err()
}

// GetAverageMigrationTime calculates the average migration duration
func (d *Database) GetAverageMigrationTime(ctx context.Context, orgFilter, batchFilter string) (float64, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildOrgFilter and buildBatchFilter
	query := `
		SELECT AVG(mh.duration_seconds) as avg_duration
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed'
			AND mh.phase = 'migration'
			AND mh.duration_seconds IS NOT NULL
			` + d.buildOrgFilter(orgFilter) + `
			` + d.buildBatchFilter(batchFilter) + `
	`

	var avgDuration sql.NullFloat64
	err := d.db.QueryRowContext(ctx, query).Scan(&avgDuration)
	if err != nil {
		return 0, fmt.Errorf("failed to get average migration time: %w", err)
	}

	if !avgDuration.Valid {
		return 0, nil
	}

	return avgDuration.Float64, nil
}

// buildOrgFilter builds the organization filter clause
// Note: orgFilter is from query parameters and is safe for use in SQL
func (d *Database) buildOrgFilter(orgFilter string) string {
	if orgFilter == "" {
		return ""
	}
	// Sanitize input by replacing single quotes
	sanitized := strings.ReplaceAll(orgFilter, "'", "''")
	return fmt.Sprintf(" AND SUBSTR(r.full_name, 1, INSTR(r.full_name, '/') - 1) = '%s'", sanitized)
}

// buildBatchFilter builds the batch filter clause
// Note: batchFilter is from query parameters and validated as integer-like
func (d *Database) buildBatchFilter(batchFilter string) string {
	if batchFilter == "" {
		return ""
	}
	// Validate that batchFilter contains only digits
	if _, err := strconv.Atoi(batchFilter); err != nil {
		return ""
	}
	return fmt.Sprintf(" AND r.batch_id = %s", batchFilter)
}

// GetRepositoryStatsByStatusFiltered returns repository counts by status with filters
func (d *Database) GetRepositoryStatsByStatusFiltered(ctx context.Context, orgFilter, batchFilter string) (map[string]int, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildOrgFilter and buildBatchFilter
	query := `
		SELECT status, COUNT(*) as count
		FROM repositories r
		WHERE 1=1
			` + d.buildOrgFilter(orgFilter) + `
			` + d.buildBatchFilter(batchFilter) + `
		GROUP BY status
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats[status] = count
	}

	return stats, rows.Err()
}

// GetSizeDistributionFiltered returns size distribution with filters
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetSizeDistributionFiltered(ctx context.Context, orgFilter, batchFilter string) ([]*SizeDistribution, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildOrgFilter and buildBatchFilter
	query := `
		SELECT 
			CASE 
				WHEN total_size IS NULL THEN 'unknown'
				WHEN total_size < 104857600 THEN 'small'
				WHEN total_size < 1073741824 THEN 'medium'
				WHEN total_size < 5368709120 THEN 'large'
				ELSE 'very_large'
			END as category,
			COUNT(*) as count
		FROM repositories r
		WHERE 1=1
			` + d.buildOrgFilter(orgFilter) + `
			` + d.buildBatchFilter(batchFilter) + `
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'small' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'large' THEN 3
				WHEN 'very_large' THEN 4
				WHEN 'unknown' THEN 5
			END
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get size distribution: %w", err)
	}
	defer rows.Close()

	var distribution []*SizeDistribution
	for rows.Next() {
		var dist SizeDistribution
		if err := rows.Scan(&dist.Category, &dist.Count); err != nil {
			return nil, fmt.Errorf("failed to scan size distribution: %w", err)
		}
		distribution = append(distribution, &dist)
	}

	return distribution, rows.Err()
}

// GetFeatureStatsFiltered returns feature stats with filters
func (d *Database) GetFeatureStatsFiltered(ctx context.Context, orgFilter, batchFilter string) (*FeatureStats, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildOrgFilter and buildBatchFilter
	query := `
		SELECT 
			SUM(CASE WHEN is_archived = 1 THEN 1 ELSE 0 END) as archived_count,
			SUM(CASE WHEN has_lfs = 1 THEN 1 ELSE 0 END) as lfs_count,
			SUM(CASE WHEN has_submodules = 1 THEN 1 ELSE 0 END) as submodules_count,
			SUM(CASE WHEN has_large_files = 1 THEN 1 ELSE 0 END) as large_files_count,
			SUM(CASE WHEN has_wiki = 1 THEN 1 ELSE 0 END) as wiki_count,
			SUM(CASE WHEN has_pages = 1 THEN 1 ELSE 0 END) as pages_count,
			SUM(CASE WHEN has_discussions = 1 THEN 1 ELSE 0 END) as discussions_count,
			SUM(CASE WHEN has_actions = 1 THEN 1 ELSE 0 END) as actions_count,
			SUM(CASE WHEN has_projects = 1 THEN 1 ELSE 0 END) as projects_count,
			SUM(CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END) as branch_protections_count,
			COUNT(*) as total
		FROM repositories r
		WHERE 1=1
			` + d.buildOrgFilter(orgFilter) + `
			` + d.buildBatchFilter(batchFilter) + `
	`

	var stats FeatureStats
	err := d.db.QueryRowContext(ctx, query).Scan(
		&stats.IsArchived,
		&stats.HasLFS,
		&stats.HasSubmodules,
		&stats.HasLargeFiles,
		&stats.HasWiki,
		&stats.HasPages,
		&stats.HasDiscussions,
		&stats.HasActions,
		&stats.HasProjects,
		&stats.HasBranchProtections,
		&stats.TotalRepositories,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature stats: %w", err)
	}

	return &stats, nil
}

// GetOrganizationStatsFiltered returns organization stats with batch filter
func (d *Database) GetOrganizationStatsFiltered(ctx context.Context, batchFilter string) ([]*OrganizationStats, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildBatchFilter
	query := `
		SELECT 
			SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as org,
			COUNT(*) as total,
			status,
			COUNT(*) as status_count
		FROM repositories r
		WHERE INSTR(full_name, '/') > 0
			` + d.buildBatchFilter(batchFilter) + `
		GROUP BY org, status
		ORDER BY total DESC, org ASC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization stats: %w", err)
	}
	defer rows.Close()

	orgMap := make(map[string]*OrganizationStats)
	for rows.Next() {
		var org, status string
		var total, statusCount int
		if err := rows.Scan(&org, &total, &status, &statusCount); err != nil {
			return nil, fmt.Errorf("failed to scan organization stats: %w", err)
		}

		if _, exists := orgMap[org]; !exists {
			orgMap[org] = &OrganizationStats{
				Organization: org,
				TotalRepos:   0,
				StatusCounts: make(map[string]int),
			}
		}

		orgMap[org].StatusCounts[status] = statusCount
		orgMap[org].TotalRepos += statusCount
	}

	stats := make([]*OrganizationStats, 0, len(orgMap))
	for _, stat := range orgMap {
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// GetMigrationCompletionStatsByOrgFiltered returns migration completion stats with batch filter
func (d *Database) GetMigrationCompletionStatsByOrgFiltered(ctx context.Context, batchFilter string) ([]*MigrationCompletionStats, error) {
	//nolint:gosec // G202: Filter values are sanitized by buildBatchFilter
	query := `
		SELECT 
			SUBSTR(full_name, 1, INSTR(full_name, '/') - 1) as organization,
			COUNT(*) as total_repos,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as completed_count,
			SUM(CASE WHEN status IN ('in_progress', 'pre_migration', 'queued_for_migration', 'migrating_content') THEN 1 ELSE 0 END) as in_progress_count,
			SUM(CASE WHEN status IN ('pending', 'discovered') THEN 1 ELSE 0 END) as pending_count,
			SUM(CASE WHEN status LIKE '%failed%' THEN 1 ELSE 0 END) as failed_count
		FROM repositories r
		WHERE full_name LIKE '%/%'
			` + d.buildBatchFilter(batchFilter) + `
		GROUP BY organization
		ORDER BY total_repos DESC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration completion stats: %w", err)
	}
	defer rows.Close()

	var stats []*MigrationCompletionStats
	for rows.Next() {
		var s MigrationCompletionStats
		if err := rows.Scan(
			&s.Organization,
			&s.TotalRepos,
			&s.CompletedCount,
			&s.InProgressCount,
			&s.PendingCount,
			&s.FailedCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan migration completion stats: %w", err)
		}
		stats = append(stats, &s)
	}

	return stats, rows.Err()
}
