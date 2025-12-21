package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// GetMigrationHistory retrieves migration history for a repository using GORM
func (d *Database) GetMigrationHistory(ctx context.Context, repoID int64) ([]*models.MigrationHistory, error) {
	var history []*models.MigrationHistory
	err := d.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("started_at DESC").
		Find(&history).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}

	return history, nil
}

// GetMigrationLogs retrieves detailed logs for a repository's migration operations
func (d *Database) GetMigrationLogs(ctx context.Context, repoID int64, level, phase string, limit, offset int) ([]*models.MigrationLog, error) {
	var logs []*models.MigrationLog
	query := d.db.WithContext(ctx).Where("repository_id = ?", repoID)

	// Add optional filters
	if level != "" {
		query = query.Where("level = ?", level)
	}
	if phase != "" {
		query = query.Where("phase = ?", phase)
	}

	err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration logs: %w", err)
	}

	return logs, nil
}

// CreateMigrationHistory creates a new migration history record using GORM
func (d *Database) CreateMigrationHistory(ctx context.Context, history *models.MigrationHistory) (int64, error) {
	result := d.db.WithContext(ctx).Create(history)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to create migration history: %w", result.Error)
	}

	return history.ID, nil
}

// UpdateMigrationHistory updates a migration history record using GORM
func (d *Database) UpdateMigrationHistory(ctx context.Context, id int64, status string, errorMsg *string) error {
	completedAt := time.Now()

	// Get the started_at time to calculate duration
	var history models.MigrationHistory
	err := d.db.WithContext(ctx).Select("started_at").Where("id = ?", id).First(&history).Error
	if err != nil {
		return fmt.Errorf("failed to get started_at time: %w", err)
	}

	durationSeconds := int(completedAt.Sub(history.StartedAt).Seconds())

	// Update the record
	result := d.db.WithContext(ctx).Model(&models.MigrationHistory{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":           status,
		"error_message":    errorMsg,
		"completed_at":     completedAt,
		"duration_seconds": durationSeconds,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update migration history: %w", result.Error)
	}

	return nil
}

// CreateMigrationLog creates a new migration log entry using GORM
func (d *Database) CreateMigrationLog(ctx context.Context, log *models.MigrationLog) error {
	result := d.db.WithContext(ctx).Create(log)
	if result.Error != nil {
		return fmt.Errorf("failed to create migration log: %w", result.Error)
	}

	return nil
}

// CompletedMigration represents a completed migration for the history page
type CompletedMigration struct {
	ID              int64      `json:"id"`
	FullName        string     `json:"full_name"`
	SourceURL       string     `json:"source_url"`
	DestinationURL  *string    `json:"destination_url"`
	Status          string     `json:"status"`
	MigratedAt      *time.Time `json:"migrated_at"`
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
			h.started_at as started_at_str,
			h.completed_at as completed_at_str,
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

	// Use a temporary struct to handle SQLite string datetime values
	type tempMigration struct {
		ID              int64      `gorm:"column:id"`
		FullName        string     `gorm:"column:full_name"`
		SourceURL       string     `gorm:"column:source_url"`
		DestinationURL  *string    `gorm:"column:destination_url"`
		Status          string     `gorm:"column:status"`
		MigratedAt      *time.Time `gorm:"column:migrated_at"`
		StartedAtStr    *string    `gorm:"column:started_at_str"`
		CompletedAtStr  *string    `gorm:"column:completed_at_str"`
		DurationSeconds *int       `gorm:"column:duration_seconds"`
	}

	var tempMigrations []tempMigration
	err := d.db.WithContext(ctx).Raw(query).Scan(&tempMigrations).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get completed migrations: %w", err)
	}

	// Convert to CompletedMigration with proper time parsing
	migrations := make([]*CompletedMigration, len(tempMigrations))
	for i, temp := range tempMigrations {
		migrations[i] = &CompletedMigration{
			ID:              temp.ID,
			FullName:        temp.FullName,
			SourceURL:       temp.SourceURL,
			DestinationURL:  temp.DestinationURL,
			Status:          temp.Status,
			MigratedAt:      temp.MigratedAt,
			DurationSeconds: temp.DurationSeconds,
		}

		// Parse started_at string to time.Time
		if temp.StartedAtStr != nil && *temp.StartedAtStr != "" {
			migrations[i].StartedAt = parseDateTime(*temp.StartedAtStr)
		}

		// Parse completed_at string to time.Time
		if temp.CompletedAtStr != nil && *temp.CompletedAtStr != "" {
			migrations[i].CompletedAt = parseDateTime(*temp.CompletedAtStr)
		}
	}

	return migrations, nil
}

// parseDateTime tries multiple datetime formats for cross-database compatibility
func parseDateTime(s string) *time.Time {
	formats := []string{
		"2006-01-02 15:04:05.999999-07:00",        // SQLite with microseconds and full timezone offset
		"2006-01-02 15:04:05.999999-07",           // PostgreSQL with short timezone offset
		"2006-01-02 15:04:05.9999999",             // SQL Server with 7 fractional digits (no timezone)
		"2006-01-02 15:04:05.999999",              // Most databases with microseconds (no timezone)
		"2006-01-02 15:04:05.999999999-07:00",     // Nanoseconds with full timezone
		"2006-01-02 15:04:05.999999999-07",        // Nanoseconds with short timezone
		"2006-01-02 15:04:05.999999999 -0700 MST", // Go's default format
		"2006-01-02 15:04:05",                     // Basic format without fractional seconds or timezone
		time.RFC3339,                              // ISO 8601 (2006-01-02T15:04:05Z07:00)
		"2006-01-02T15:04:05.999999-07:00",        // ISO 8601 with microseconds
		"2006-01-02T15:04:05",                     // ISO 8601 without timezone
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}
	return nil
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

// GetMigrationCompletionStatsByOrg returns migration completion stats grouped by organization
func (d *Database) GetMigrationCompletionStatsByOrg(ctx context.Context) ([]*MigrationCompletionStats, error) {
	// Use dialect-specific string functions via DialectDialer interface
	extractOrg := d.dialect.ExtractOrgFromFullName("full_name")

	query := fmt.Sprintf(`
		SELECT 
			%s as organization,
			COUNT(*) as total_repos,
			SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
			SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
			SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
			SUM(CASE WHEN status LIKE '%%failed%%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
		FROM repositories
		WHERE full_name LIKE '%%/%%'
		AND status != 'wont_migrate'
		GROUP BY organization
		ORDER BY total_repos DESC
	`, extractOrg)

	// Use GORM Raw() for analytics query
	var stats []*MigrationCompletionStats
	err := d.db.WithContext(ctx).Raw(query).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration completion stats: %w", err)
	}

	return stats, nil
}

// GetMigrationCompletionStatsByOrgFiltered returns migration completion stats with org and batch filters
//
//nolint:dupl // Similar to GetMigrationCompletionStatsByOrg but with filters
func (d *Database) GetMigrationCompletionStatsByOrgFiltered(ctx context.Context, orgFilter, projectFilter, batchFilter string) ([]*MigrationCompletionStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Use dialect-specific string functions via DialectDialer interface
	extractOrg := d.dialect.ExtractOrgFromFullName("full_name")

	query := fmt.Sprintf(`
		SELECT 
			%s as organization,
			COUNT(*) as total_repos,
			SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
			SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
			SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
			SUM(CASE WHEN status LIKE '%%failed%%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
		FROM repositories r
		WHERE full_name LIKE '%%/%%'
			AND status != 'wont_migrate'
			%s
			%s
			%s
		GROUP BY organization
		ORDER BY total_repos DESC
	`, extractOrg, orgFilterSQL, projectFilterSQL, batchFilterSQL)

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)

	// Use GORM Raw() for analytics query
	var stats []*MigrationCompletionStats
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration completion stats: %w", err)
	}

	return stats, nil
}

// GetMigrationCompletionStatsByProjectFiltered returns migration completion stats grouped by ADO project
// Similar to GetMigrationCompletionStatsByOrgFiltered but groups by ado_project field
//
//nolint:dupl // Similar to GetMigrationCompletionStatsByOrgFiltered but groups by project
func (d *Database) GetMigrationCompletionStatsByProjectFiltered(ctx context.Context, orgFilter, projectFilter, batchFilter string) ([]*MigrationCompletionStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)

	// Query groups by ado_project field for Azure DevOps repositories
	query := `
		SELECT 
			ado_project as organization,
			COUNT(*) as total_repos,
			SUM(CASE WHEN status IN ('complete', 'migration_complete') THEN 1 ELSE 0 END) as completed_count,
			SUM(CASE WHEN status IN ('pre_migration', 'archive_generating', 'queued_for_migration', 'migrating_content', 'post_migration') THEN 1 ELSE 0 END) as in_progress_count,
			SUM(CASE WHEN status IN ('pending', 'dry_run_queued', 'dry_run_in_progress', 'dry_run_complete') THEN 1 ELSE 0 END) as pending_count,
			SUM(CASE WHEN status LIKE '%failed%' OR status = 'rolled_back' THEN 1 ELSE 0 END) as failed_count
		FROM repositories r
		WHERE ado_project IS NOT NULL AND ado_project != ''
			AND status != 'wont_migrate'
			` + orgFilterSQL + `
			` + projectFilterSQL + `
			` + batchFilterSQL + `
		GROUP BY ado_project
		ORDER BY total_repos DESC
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)

	// Use GORM Raw() for analytics query
	var stats []*MigrationCompletionStats
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration completion stats by project: %w", err)
	}

	return stats, nil
}
