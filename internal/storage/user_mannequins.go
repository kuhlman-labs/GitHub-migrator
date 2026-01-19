package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// SaveUserMannequin inserts or updates a user mannequin record
// The unique key is (source_login, mannequin_org)
func (d *Database) SaveUserMannequin(ctx context.Context, mannequin *models.UserMannequin) error {
	// Check if mannequin already exists for this source_login + org combination
	var existing models.UserMannequin
	err := d.db.WithContext(ctx).
		Where("source_login = ? AND mannequin_org = ?", mannequin.SourceLogin, mannequin.MannequinOrg).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Insert new mannequin
		if mannequin.CreatedAt.IsZero() {
			mannequin.CreatedAt = time.Now()
		}
		if mannequin.UpdatedAt.IsZero() {
			mannequin.UpdatedAt = time.Now()
		}

		result := d.db.WithContext(ctx).Create(mannequin)
		if result.Error != nil {
			return fmt.Errorf("failed to create user mannequin: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing user mannequin: %w", err)
	}

	// Mannequin exists - update it
	mannequin.ID = existing.ID
	mannequin.CreatedAt = existing.CreatedAt
	mannequin.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Save(mannequin)
	if result.Error != nil {
		return fmt.Errorf("failed to update user mannequin: %w", result.Error)
	}

	return nil
}

// GetUserMannequin retrieves a mannequin record by source login and org
func (d *Database) GetUserMannequin(ctx context.Context, sourceLogin, mannequinOrg string) (*models.UserMannequin, error) {
	var mannequin models.UserMannequin
	err := d.db.WithContext(ctx).
		Where("source_login = ? AND mannequin_org = ?", sourceLogin, mannequinOrg).
		First(&mannequin).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user mannequin: %w", err)
	}

	return &mannequin, nil
}

// GetUserMannequinsBySourceLogin retrieves all mannequins for a source user across all orgs
func (d *Database) GetUserMannequinsBySourceLogin(ctx context.Context, sourceLogin string) ([]*models.UserMannequin, error) {
	var mannequins []*models.UserMannequin
	err := d.db.WithContext(ctx).
		Where("source_login = ?", sourceLogin).
		Order("mannequin_org ASC").
		Find(&mannequins).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user mannequins: %w", err)
	}

	return mannequins, nil
}

// UserMannequinFilters defines filters for listing user mannequins
type UserMannequinFilters struct {
	MannequinOrg  string // Filter by mannequin_org
	ReclaimStatus string // Filter by reclaim_status
	Search        string // Search in source_login
	Limit         int
	Offset        int
}

// ListUserMannequins returns user mannequins with optional filters
func (d *Database) ListUserMannequins(ctx context.Context, filters UserMannequinFilters) ([]*models.UserMannequin, int64, error) {
	var mannequins []*models.UserMannequin
	var total int64

	query := d.db.WithContext(ctx).Model(&models.UserMannequin{})

	// Apply filters
	if filters.MannequinOrg != "" {
		query = query.Where("mannequin_org = ?", filters.MannequinOrg)
	}

	if filters.ReclaimStatus != "" {
		query = query.Where("reclaim_status = ?", filters.ReclaimStatus)
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where("source_login LIKE ?", searchPattern)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count user mannequins: %w", err)
	}

	// Apply pagination and ordering
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("source_login ASC").Find(&mannequins).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list user mannequins: %w", err)
	}

	return mannequins, total, nil
}

// UpdateMannequinReclaimStatus updates the reclaim status for a specific mannequin
func (d *Database) UpdateMannequinReclaimStatus(ctx context.Context, sourceLogin, mannequinOrg, reclaimStatus string, reclaimError *string) error {
	updates := map[string]any{
		"reclaim_status": reclaimStatus,
		"updated_at":     time.Now(),
	}

	if reclaimError != nil {
		updates["reclaim_error"] = *reclaimError
	}

	result := d.db.WithContext(ctx).Model(&models.UserMannequin{}).
		Where("source_login = ? AND mannequin_org = ?", sourceLogin, mannequinOrg).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update mannequin reclaim status: %w", result.Error)
	}

	return nil
}

// GetMannequinOrgs returns a list of unique mannequin organizations
func (d *Database) GetMannequinOrgs(ctx context.Context) ([]string, error) {
	var orgs []string

	err := d.db.WithContext(ctx).
		Model(&models.UserMannequin{}).
		Distinct("mannequin_org").
		Order("mannequin_org ASC").
		Pluck("mannequin_org", &orgs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get mannequin orgs: %w", err)
	}

	return orgs, nil
}

// DeleteUserMannequin deletes a mannequin record by source login and org
func (d *Database) DeleteUserMannequin(ctx context.Context, sourceLogin, mannequinOrg string) error {
	result := d.db.WithContext(ctx).
		Where("source_login = ? AND mannequin_org = ?", sourceLogin, mannequinOrg).
		Delete(&models.UserMannequin{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete user mannequin: %w", result.Error)
	}

	return nil
}

// ListMappingsWithMannequins returns user mappings joined with their mannequin data for a specific org
// This is used for generating GEI CSV files
func (d *Database) ListMappingsWithMannequins(ctx context.Context, mannequinOrg string, status string) ([]*MappingWithMannequin, error) {
	var results []*MappingWithMannequin

	query := d.db.WithContext(ctx).
		Table("user_mappings um").
		Select(`
			um.source_login,
			um.source_email,
			um.source_name,
			um.destination_login,
			um.destination_email,
			um.mapping_status,
			um.match_confidence,
			um.match_reason,
			umq.mannequin_id,
			umq.mannequin_login,
			umq.mannequin_org,
			umq.reclaim_status,
			umq.reclaim_error
		`).
		Joins("INNER JOIN user_mannequins umq ON um.source_login = umq.source_login AND umq.mannequin_org = ?", mannequinOrg)

	if status != "" {
		query = query.Where("um.mapping_status = ?", status)
	}

	err := query.Order("um.source_login ASC").Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list mappings with mannequins: %w", err)
	}

	return results, nil
}

// MappingWithMannequin represents a user mapping joined with mannequin data
type MappingWithMannequin struct {
	SourceLogin      string  `gorm:"column:source_login"`
	SourceEmail      *string `gorm:"column:source_email"`
	SourceName       *string `gorm:"column:source_name"`
	DestinationLogin *string `gorm:"column:destination_login"`
	DestinationEmail *string `gorm:"column:destination_email"`
	MappingStatus    string  `gorm:"column:mapping_status"`
	MatchConfidence  *int    `gorm:"column:match_confidence"`
	MatchReason      *string `gorm:"column:match_reason"`
	MannequinID      string  `gorm:"column:mannequin_id"`
	MannequinLogin   *string `gorm:"column:mannequin_login"`
	MannequinOrg     string  `gorm:"column:mannequin_org"`
	ReclaimStatus    *string `gorm:"column:reclaim_status"`
	ReclaimError     *string `gorm:"column:reclaim_error"`
}
