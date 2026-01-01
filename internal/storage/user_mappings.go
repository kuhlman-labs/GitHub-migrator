package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// SaveUserMapping inserts or updates a user mapping in the database
func (d *Database) SaveUserMapping(ctx context.Context, mapping *models.UserMapping) error {
	// Check if mapping already exists
	var existing models.UserMapping
	err := d.db.WithContext(ctx).
		Where("source_login = ?", mapping.SourceLogin).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Insert new mapping
		if mapping.CreatedAt.IsZero() {
			mapping.CreatedAt = time.Now()
		}
		if mapping.UpdatedAt.IsZero() {
			mapping.UpdatedAt = time.Now()
		}
		if mapping.MappingStatus == "" {
			mapping.MappingStatus = string(models.UserMappingStatusUnmapped)
		}

		result := d.db.WithContext(ctx).Create(mapping)
		if result.Error != nil {
			return fmt.Errorf("failed to create user mapping: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing user mapping: %w", err)
	}

	// Mapping exists - update it
	mapping.ID = existing.ID
	mapping.CreatedAt = existing.CreatedAt
	mapping.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Save(mapping)
	if result.Error != nil {
		return fmt.Errorf("failed to update user mapping: %w", result.Error)
	}

	return nil
}

// GetUserMappingBySourceLogin retrieves a user mapping by source login
func (d *Database) GetUserMappingBySourceLogin(ctx context.Context, sourceLogin string) (*models.UserMapping, error) {
	var mapping models.UserMapping
	err := d.db.WithContext(ctx).
		Where("source_login = ?", sourceLogin).
		First(&mapping).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user mapping: %w", err)
	}

	return &mapping, nil
}

// GetUserMappingByID retrieves a user mapping by ID
func (d *Database) GetUserMappingByID(ctx context.Context, id int64) (*models.UserMapping, error) {
	var mapping models.UserMapping
	err := d.db.WithContext(ctx).
		Where("id = ?", id).
		First(&mapping).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user mapping: %w", err)
	}

	return &mapping, nil
}

// UserMappingFilters defines filters for listing user mappings
type UserMappingFilters struct {
	Status         string // Filter by mapping_status
	HasDestination *bool  // Filter by whether destination_login is set
	HasMannequin   *bool  // Filter by whether mannequin_id is set
	ReclaimStatus  string // Filter by reclaim_status
	SourceOrg      string // Filter by source_org
	Search         string // Search in source_login, source_email, destination_login
	Limit          int
	Offset         int
}

// ListUserMappings returns user mappings with optional filters
func (d *Database) ListUserMappings(ctx context.Context, filters UserMappingFilters) ([]*models.UserMapping, int64, error) {
	var mappings []*models.UserMapping
	var total int64

	query := d.db.WithContext(ctx).Model(&models.UserMapping{})

	// Apply filters
	if filters.Status != "" {
		query = query.Where("mapping_status = ?", filters.Status)
	}

	if filters.HasDestination != nil {
		if *filters.HasDestination {
			query = query.Where("destination_login IS NOT NULL AND destination_login != ''")
		} else {
			query = query.Where("destination_login IS NULL OR destination_login = ''")
		}
	}

	if filters.HasMannequin != nil {
		if *filters.HasMannequin {
			query = query.Where("mannequin_id IS NOT NULL AND mannequin_id != ''")
		} else {
			query = query.Where("mannequin_id IS NULL OR mannequin_id = ''")
		}
	}

	if filters.ReclaimStatus != "" {
		query = query.Where("reclaim_status = ?", filters.ReclaimStatus)
	}

	if filters.SourceOrg != "" {
		query = query.Where("source_org = ?", filters.SourceOrg)
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where(
			"source_login LIKE ? OR source_email LIKE ? OR destination_login LIKE ? OR source_name LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count user mappings: %w", err)
	}

	// Apply pagination and ordering
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("source_login ASC").Find(&mappings).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list user mappings: %w", err)
	}

	return mappings, total, nil
}

// GetUserMappingStats returns summary statistics for user mappings
// If orgFilter is provided, stats are filtered to that organization only
func (d *Database) GetUserMappingStats(ctx context.Context, orgFilter string) (map[string]any, error) {
	var total int64
	var mapped int64
	var unmapped int64
	var skipped int64
	var reclaimed int64
	var pendingReclaim int64

	db := d.db.WithContext(ctx).Model(&models.UserMapping{})
	if orgFilter != "" {
		db = db.Where("source_org = ?", orgFilter)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count mappings: %w", err)
	}

	mappedQuery := d.db.WithContext(ctx).Model(&models.UserMapping{}).Where("mapping_status = ?", models.UserMappingStatusMapped)
	if orgFilter != "" {
		mappedQuery = mappedQuery.Where("source_org = ?", orgFilter)
	}
	if err := mappedQuery.Count(&mapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count mapped: %w", err)
	}

	unmappedQuery := d.db.WithContext(ctx).Model(&models.UserMapping{}).Where("mapping_status = ?", models.UserMappingStatusUnmapped)
	if orgFilter != "" {
		unmappedQuery = unmappedQuery.Where("source_org = ?", orgFilter)
	}
	if err := unmappedQuery.Count(&unmapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count unmapped: %w", err)
	}

	skippedQuery := d.db.WithContext(ctx).Model(&models.UserMapping{}).Where("mapping_status = ?", models.UserMappingStatusSkipped)
	if orgFilter != "" {
		skippedQuery = skippedQuery.Where("source_org = ?", orgFilter)
	}
	if err := skippedQuery.Count(&skipped).Error; err != nil {
		return nil, fmt.Errorf("failed to count skipped: %w", err)
	}

	reclaimedQuery := d.db.WithContext(ctx).Model(&models.UserMapping{}).Where("mapping_status = ?", models.UserMappingStatusReclaimed)
	if orgFilter != "" {
		reclaimedQuery = reclaimedQuery.Where("source_org = ?", orgFilter)
	}
	if err := reclaimedQuery.Count(&reclaimed).Error; err != nil {
		return nil, fmt.Errorf("failed to count reclaimed: %w", err)
	}

	pendingQuery := d.db.WithContext(ctx).Model(&models.UserMapping{}).Where("reclaim_status = ?", models.ReclaimStatusPending)
	if orgFilter != "" {
		pendingQuery = pendingQuery.Where("source_org = ?", orgFilter)
	}
	if err := pendingQuery.Count(&pendingReclaim).Error; err != nil {
		return nil, fmt.Errorf("failed to count pending reclaim: %w", err)
	}

	return map[string]any{
		"total":           total,
		"mapped":          mapped,
		"unmapped":        unmapped,
		"skipped":         skipped,
		"reclaimed":       reclaimed,
		"pending_reclaim": pendingReclaim,
	}, nil
}

// UpdateUserMappingStatus updates the mapping status for a user
func (d *Database) UpdateUserMappingStatus(ctx context.Context, sourceLogin, status string) error {
	result := d.db.WithContext(ctx).Model(&models.UserMapping{}).
		Where("source_login = ?", sourceLogin).
		Updates(map[string]any{
			"mapping_status": status,
			"updated_at":     time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update mapping status: %w", result.Error)
	}

	return nil
}

// UpdateUserMappingDestination updates the destination user for a mapping
func (d *Database) UpdateUserMappingDestination(ctx context.Context, sourceLogin, destLogin, destEmail string) error {
	updates := map[string]any{
		"destination_login": destLogin,
		"mapping_status":    string(models.UserMappingStatusMapped),
		"updated_at":        time.Now(),
	}

	if destEmail != "" {
		updates["destination_email"] = destEmail
	}

	result := d.db.WithContext(ctx).Model(&models.UserMapping{}).
		Where("source_login = ?", sourceLogin).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update mapping destination: %w", result.Error)
	}

	return nil
}

// UpdateMannequinInfo updates mannequin information for a user mapping
func (d *Database) UpdateMannequinInfo(ctx context.Context, sourceLogin, mannequinID, mannequinLogin string) error {
	result := d.db.WithContext(ctx).Model(&models.UserMapping{}).
		Where("source_login = ?", sourceLogin).
		Updates(map[string]any{
			"mannequin_id":    mannequinID,
			"mannequin_login": mannequinLogin,
			"updated_at":      time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update mannequin info: %w", result.Error)
	}

	return nil
}

// UpdateReclaimStatus updates the reclaim status for a user mapping
func (d *Database) UpdateReclaimStatus(ctx context.Context, sourceLogin, reclaimStatus string, reclaimError *string) error {
	updates := map[string]any{
		"reclaim_status": reclaimStatus,
		"updated_at":     time.Now(),
	}

	if reclaimError != nil {
		updates["reclaim_error"] = *reclaimError
	}

	// If reclaimed, update mapping status too
	if reclaimStatus == string(models.ReclaimStatusCompleted) {
		updates["mapping_status"] = string(models.UserMappingStatusReclaimed)
	}

	result := d.db.WithContext(ctx).Model(&models.UserMapping{}).
		Where("source_login = ?", sourceLogin).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update reclaim status: %w", result.Error)
	}

	return nil
}

// UpdateMatchInfo updates the auto-match confidence and reason for a user mapping
func (d *Database) UpdateMatchInfo(ctx context.Context, sourceLogin string, confidence int, reason string) error {
	result := d.db.WithContext(ctx).Model(&models.UserMapping{}).
		Where("source_login = ?", sourceLogin).
		Updates(map[string]any{
			"match_confidence": confidence,
			"match_reason":     reason,
			"updated_at":       time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update match info: %w", result.Error)
	}

	return nil
}

// BulkCreateUserMappings creates multiple user mappings efficiently
func (d *Database) BulkCreateUserMappings(ctx context.Context, mappings []*models.UserMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	now := time.Now()
	for _, m := range mappings {
		if m.CreatedAt.IsZero() {
			m.CreatedAt = now
		}
		if m.UpdatedAt.IsZero() {
			m.UpdatedAt = now
		}
		if m.MappingStatus == "" {
			m.MappingStatus = string(models.UserMappingStatusUnmapped)
		}
	}

	result := d.db.WithContext(ctx).CreateInBatches(mappings, 100)
	if result.Error != nil {
		return fmt.Errorf("failed to bulk create user mappings: %w", result.Error)
	}

	return nil
}

// DeleteUserMapping deletes a user mapping by source login
func (d *Database) DeleteUserMapping(ctx context.Context, sourceLogin string) error {
	result := d.db.WithContext(ctx).
		Where("source_login = ?", sourceLogin).
		Delete(&models.UserMapping{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete user mapping: %w", result.Error)
	}

	return nil
}

// DeleteAllUserMappings removes all user mappings
func (d *Database) DeleteAllUserMappings(ctx context.Context) error {
	result := d.db.WithContext(ctx).Where("1=1").Delete(&models.UserMapping{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete user mappings: %w", result.Error)
	}
	return nil
}

// GetUserMappingSourceOrgs returns a list of unique source organizations
// First checks user_mappings.source_org, falls back to user_org_memberships if empty
func (d *Database) GetUserMappingSourceOrgs(ctx context.Context) ([]string, error) {
	var orgs []string

	// First try to get from user_mappings
	err := d.db.WithContext(ctx).
		Model(&models.UserMapping{}).
		Where("source_org IS NOT NULL AND source_org != ''").
		Distinct("source_org").
		Pluck("source_org", &orgs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get source orgs from mappings: %w", err)
	}

	// If no source orgs in mappings, fall back to user_org_memberships
	if len(orgs) == 0 {
		err = d.db.WithContext(ctx).
			Model(&models.UserOrgMembership{}).
			Distinct("organization").
			Order("organization ASC").
			Pluck("organization", &orgs).Error

		if err != nil {
			return nil, fmt.Errorf("failed to get source orgs from memberships: %w", err)
		}
	}

	return orgs, nil
}

// CreateUserMappingFromUser creates a user mapping from a discovered user
func (d *Database) CreateUserMappingFromUser(ctx context.Context, user *models.GitHubUser) error {
	mapping := &models.UserMapping{
		SourceID:      user.SourceID, // Copy source_id for multi-source support
		SourceLogin:   user.Login,
		SourceEmail:   user.Email,
		SourceName:    user.Name,
		MappingStatus: string(models.UserMappingStatusUnmapped),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return d.SaveUserMapping(ctx, mapping)
}

// SyncUserMappingsFromUsers creates user mappings for all discovered users that don't have one
// Also populates source_org from user_org_memberships and source_id from the user
func (d *Database) SyncUserMappingsFromUsers(ctx context.Context) (int64, error) {
	// Find users without mappings and create mappings for them
	// Also get the primary org (first alphabetically) from user_org_memberships
	type userWithOrg struct {
		SourceID   *int64 `gorm:"column:source_id"`
		Login      string
		Email      *string
		Name       *string
		PrimaryOrg *string `gorm:"column:primary_org"`
	}

	var usersWithOrgs []userWithOrg
	err := d.db.WithContext(ctx).
		Raw(`
			SELECT 
				u.source_id,
				u.login,
				u.email,
				u.name,
				(SELECT organization FROM user_org_memberships 
				 WHERE user_login = u.login 
				 ORDER BY organization ASC 
				 LIMIT 1) as primary_org
			FROM github_users u
			LEFT JOIN user_mappings m ON u.login = m.source_login
			WHERE m.id IS NULL
		`).
		Scan(&usersWithOrgs).Error

	if err != nil {
		return 0, fmt.Errorf("failed to find users without mappings: %w", err)
	}

	if len(usersWithOrgs) == 0 {
		return 0, nil
	}

	mappings := make([]*models.UserMapping, 0, len(usersWithOrgs))
	now := time.Now()
	for _, user := range usersWithOrgs {
		mappings = append(mappings, &models.UserMapping{
			SourceID:      user.SourceID, // Copy source_id for multi-source support
			SourceLogin:   user.Login,
			SourceEmail:   user.Email,
			SourceName:    user.Name,
			SourceOrg:     user.PrimaryOrg, // Set primary org from memberships
			MappingStatus: string(models.UserMappingStatusUnmapped),
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}

	if err := d.BulkCreateUserMappings(ctx, mappings); err != nil {
		return 0, err
	}

	return int64(len(mappings)), nil
}

// UpdateUserMappingSourceOrgsFromMemberships updates source_org for existing mappings
// based on data from user_org_memberships table
func (d *Database) UpdateUserMappingSourceOrgsFromMemberships(ctx context.Context) (int64, error) {
	// Update mappings that have NULL source_org but have org memberships
	result := d.db.WithContext(ctx).Exec(`
		UPDATE user_mappings 
		SET source_org = (
			SELECT organization 
			FROM user_org_memberships 
			WHERE user_login = source_login 
			ORDER BY organization ASC 
			LIMIT 1
		),
		updated_at = ?
		WHERE source_org IS NULL 
		AND EXISTS (
			SELECT 1 FROM user_org_memberships WHERE user_login = source_login
		)
	`, time.Now())

	if result.Error != nil {
		return 0, fmt.Errorf("failed to update source orgs: %w", result.Error)
	}

	return result.RowsAffected, nil
}
