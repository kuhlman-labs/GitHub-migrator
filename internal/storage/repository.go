package storage

import (
	"context"
	"database/sql"
	"fmt"
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
// nolint:dupl // Similar to GetRepositoryByID but queries by full_name
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

	repos := []*models.Repository{}
	for rows.Next() {
		var repo models.Repository
		// nolint:dupl // Standard repository scanning, duplication expected
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
			   has_lfs, has_submodules, default_branch, branch_count, 
			   commit_count, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   status, batch_id, priority, discovered_at, updated_at, migrated_at
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
			   has_lfs, has_submodules, default_branch, branch_count, 
			   commit_count, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   status, batch_id, priority, discovered_at, updated_at, migrated_at
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
			   has_lfs, has_submodules, default_branch, branch_count, 
			   commit_count, has_wiki, has_pages, has_discussions, 
			   has_actions, has_projects, branch_protections, 
			   environment_count, secret_count, variable_count, 
			   webhook_count, contributor_count, top_contributors,
			   status, batch_id, priority, discovered_at, updated_at, migrated_at
		FROM repositories 
		WHERE id = ?
	`

	var repo models.Repository
	err := d.db.QueryRowContext(ctx, query, id).Scan(
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
			&repo.HasSubmodules, &repo.DefaultBranch, &repo.BranchCount,
			&repo.CommitCount, &repo.HasWiki, &repo.HasPages,
			&repo.HasDiscussions, &repo.HasActions, &repo.HasProjects,
			&repo.BranchProtections, &repo.EnvironmentCount, &repo.SecretCount,
			&repo.VariableCount, &repo.WebhookCount, &repo.ContributorCount,
			&repo.TopContributors, &repo.Status, &repo.BatchID, &repo.Priority,
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
