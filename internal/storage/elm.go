package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ELMUnknownSourceBucket is the bucket key used in ELMInFlightCounts.BySourceID for
// repositories whose source_id is NULL. Such repositories are never dropped from the
// count; they are all attributed to this single, well-defined bucket so the per-source
// admission ceiling still applies to them collectively.
const ELMUnknownSourceBucket int64 = 0

// columnLastError is the last_error column name shared by the polling writers.
const columnLastError = "last_error"

// elmInFlightStatuses are the repository statuses that mean a live migration is
// currently occupying capacity on the appliance: the backfill is running, is caught
// up and waiting on an operator, or is cutting over.
var elmInFlightStatuses = []string{
	string(models.StatusSyncing),
	string(models.StatusAwaitingCutover),
	string(models.StatusCuttingOver),
}

// ELMInFlightCounts reports the in-flight ELM migration counts used for admission control.
//
// Both counts are derived from the same predicate (an elm_migrations row whose
// repository is in one of the ELM lifecycle statuses), so they are always coherent
// with each other and with ListELMMigrationsInFlight.
type ELMInFlightCounts struct {
	// BySourceID counts in-flight migrations grouped by the existing indexed FK
	// repositories.source_id -- the source-instance ceiling (10 concurrent per
	// source) is enforced against this. Repositories with a NULL source_id are all
	// counted under ELMUnknownSourceBucket.
	BySourceID map[int64]int
	// Global is the total in-flight count across every source. The destination
	// ceiling (20 concurrent) is a GLOBAL count rather than a grouped one: the
	// deployment targets exactly one configured destination (config.Destination is
	// a single DestinationConfig with one base_url), so per-deployment and
	// per-destination-enterprise are the same number. If multi-destination support
	// ever lands, this count must become grouped.
	Global int
}

// CountForSource returns the in-flight count for a repository's source, mapping a
// NULL source_id onto ELMUnknownSourceBucket.
func (c *ELMInFlightCounts) CountForSource(sourceID *int64) int {
	key := ELMUnknownSourceBucket
	if sourceID != nil {
		key = *sourceID
	}
	return c.BySourceID[key]
}

// GetELMMigration retrieves the ELM migration record for a repository.
// Returns (nil, nil) when no record exists.
func (d *Database) GetELMMigration(ctx context.Context, repoID int64) (*models.ELMMigration, error) {
	var rec models.ELMMigration
	err := d.db.WithContext(ctx).Where("repository_id = ?", repoID).First(&rec).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ELM migration: %w", err)
	}
	return &rec, nil
}

// GetELMMigrationByELMID retrieves the ELM migration record by the appliance-side id.
// Returns (nil, nil) when no record exists.
func (d *Database) GetELMMigrationByELMID(ctx context.Context, elmMigrationID string) (*models.ELMMigration, error) {
	var rec models.ELMMigration
	err := d.db.WithContext(ctx).Where("elm_migration_id = ?", elmMigrationID).First(&rec).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ELM migration by ELM id: %w", err)
	}
	return &rec, nil
}

// UpsertELMMigration creates or updates the ELM migration record for a repository.
// The record is keyed by repository_id, so repeated polls update in place rather
// than accumulating rows.
func (d *Database) UpsertELMMigration(ctx context.Context, rec *models.ELMMigration) error {
	if rec == nil {
		return fmt.Errorf("elm migration record is nil")
	}
	if rec.RepositoryID == 0 {
		return fmt.Errorf("elm migration record requires a repository id")
	}
	if rec.ELMMigrationID == "" {
		return fmt.Errorf("elm migration record requires an ELM migration id")
	}

	now := time.Now()
	rec.UpdatedAt = now
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}

	err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "repository_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"elm_migration_id",
			"elm_status",
			"elm_phase",
			"cutover_ready",
			"progress_percent",
			"last_polled_at",
			columnLastError,
			"updated_at",
		}),
	}).Create(rec).Error
	if err != nil {
		return fmt.Errorf("failed to upsert ELM migration: %w", err)
	}
	return nil
}

// ListELMMigrationsInFlight returns the ELM migration records whose repository is
// currently in an ELM lifecycle status (syncing, awaiting_cutover, cutting_over).
// This is the set the poll loop advances and the set the admission counts measure.
func (d *Database) ListELMMigrationsInFlight(ctx context.Context) ([]*models.ELMMigration, error) {
	var records []*models.ELMMigration
	err := d.db.WithContext(ctx).
		Model(&models.ELMMigration{}).
		Joins("JOIN repositories ON repositories.id = elm_migrations.repository_id").
		Where("repositories.status IN ?", elmInFlightStatuses).
		Order("elm_migrations.id ASC").
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list in-flight ELM migrations: %w", err)
	}
	return records, nil
}

// CountELMMigrationsInFlight returns the in-flight counts used by the ELM admission
// ceilings: grouped by repositories.source_id for the per-source ceiling, and a
// global total for the per-deployment (single destination) ceiling.
func (d *Database) CountELMMigrationsInFlight(ctx context.Context) (*ELMInFlightCounts, error) {
	type row struct {
		SourceID *int64
		Total    int
	}
	var rows []row

	err := d.db.WithContext(ctx).
		Model(&models.ELMMigration{}).
		Select("repositories.source_id AS source_id, COUNT(*) AS total").
		Joins("JOIN repositories ON repositories.id = elm_migrations.repository_id").
		Where("repositories.status IN ?", elmInFlightStatuses).
		Group("repositories.source_id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count in-flight ELM migrations: %w", err)
	}

	counts := &ELMInFlightCounts{BySourceID: make(map[int64]int, len(rows))}
	for _, r := range rows {
		// A NULL source_id is bucketed under ELMUnknownSourceBucket rather than
		// dropped, so unattributed repositories still count against a ceiling.
		key := ELMUnknownSourceBucket
		if r.SourceID != nil {
			key = *r.SourceID
		}
		counts.BySourceID[key] += r.Total
		counts.Global += r.Total
	}
	return counts, nil
}

// DeleteELMMigration removes the ELM migration record for a repository.
// Deleting a record that does not exist is not an error.
func (d *Database) DeleteELMMigration(ctx context.Context, repoID int64) error {
	err := d.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Delete(&models.ELMMigration{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete ELM migration: %w", err)
	}
	return nil
}

// UpdateRepositoryMigrationRoute sets or clears the migration route for a repository.
//
// A nil route clears the column, returning the repository to the GEI default. A
// non-nil route must be one of the two legal values (models.IsValidMigrationRoute);
// anything else is rejected and the stored route is left unchanged.
func (d *Database) UpdateRepositoryMigrationRoute(ctx context.Context, fullName string, route *string) error {
	var value any
	if route != nil {
		if !models.IsValidMigrationRoute(*route) {
			return fmt.Errorf("invalid migration route %q: must be %q or %q",
				*route, models.MigrationRouteGEI, models.MigrationRouteELM)
		}
		value = *route
	}

	result := d.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("full_name = ?", fullName).
		Updates(map[string]any{
			"migration_route": value,
			"updated_at":      time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update migration route: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("repository not found: %s", fullName)
	}
	return nil
}
