package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// Team mapping status constants
const (
	teamMappingStatusUnmapped = "unmapped"
	teamMappingStatusMapped   = "mapped"
	teamMappingStatusSkipped  = "skipped"
)

// SaveTeamMapping inserts or updates a team mapping in the database
func (d *Database) SaveTeamMapping(ctx context.Context, mapping *models.TeamMapping) error {
	// Check if mapping already exists
	var existing models.TeamMapping
	err := d.db.WithContext(ctx).
		Where("source_org = ? AND source_team_slug = ?", mapping.SourceOrg, mapping.SourceTeamSlug).
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
			mapping.MappingStatus = teamMappingStatusUnmapped
		}

		result := d.db.WithContext(ctx).Create(mapping)
		if result.Error != nil {
			return fmt.Errorf("failed to create team mapping: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing team mapping: %w", err)
	}

	// Mapping exists - update it
	mapping.ID = existing.ID
	mapping.CreatedAt = existing.CreatedAt
	mapping.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Save(mapping)
	if result.Error != nil {
		return fmt.Errorf("failed to update team mapping: %w", result.Error)
	}

	return nil
}

// GetTeamMapping retrieves a team mapping by source org and team slug
func (d *Database) GetTeamMapping(ctx context.Context, sourceOrg, sourceTeamSlug string) (*models.TeamMapping, error) {
	var mapping models.TeamMapping
	err := d.db.WithContext(ctx).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		First(&mapping).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team mapping: %w", err)
	}

	return &mapping, nil
}

// GetTeamMappingByID retrieves a team mapping by ID
func (d *Database) GetTeamMappingByID(ctx context.Context, id int64) (*models.TeamMapping, error) {
	var mapping models.TeamMapping
	err := d.db.WithContext(ctx).
		Where("id = ?", id).
		First(&mapping).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team mapping: %w", err)
	}

	return &mapping, nil
}

// TeamMappingFilters defines filters for listing team mappings
type TeamMappingFilters struct {
	SourceOrg      string // Filter by source organization
	DestinationOrg string // Filter by destination organization
	Status         string // Filter by mapping_status
	HasDestination *bool  // Filter by whether destination is set
	Search         string // Search in team names/slugs
	Limit          int
	Offset         int
}

// ListTeamMappings returns team mappings with optional filters
func (d *Database) ListTeamMappings(ctx context.Context, filters TeamMappingFilters) ([]*models.TeamMapping, int64, error) {
	var mappings []*models.TeamMapping
	var total int64

	query := d.db.WithContext(ctx).Model(&models.TeamMapping{})

	// Apply filters
	if filters.SourceOrg != "" {
		query = query.Where("source_org = ?", filters.SourceOrg)
	}

	if filters.DestinationOrg != "" {
		query = query.Where("destination_org = ?", filters.DestinationOrg)
	}

	if filters.Status != "" {
		query = query.Where("mapping_status = ?", filters.Status)
	}

	if filters.HasDestination != nil {
		if *filters.HasDestination {
			query = query.Where("destination_org IS NOT NULL AND destination_team_slug IS NOT NULL")
		} else {
			query = query.Where("destination_org IS NULL OR destination_team_slug IS NULL")
		}
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where(
			"source_team_slug LIKE ? OR source_team_name LIKE ? OR destination_team_slug LIKE ? OR destination_team_name LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count team mappings: %w", err)
	}

	// Apply pagination and ordering
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("source_org ASC, source_team_slug ASC").Find(&mappings).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list team mappings: %w", err)
	}

	return mappings, total, nil
}

// GetTeamMappingStats returns summary statistics for team mappings
func (d *Database) GetTeamMappingStats(ctx context.Context) (map[string]interface{}, error) {
	var total int64
	var mapped int64
	var unmapped int64
	var skipped int64

	db := d.db.WithContext(ctx).Model(&models.TeamMapping{})

	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count mappings: %w", err)
	}

	if err := db.Where("mapping_status = ?", teamMappingStatusMapped).Count(&mapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count mapped: %w", err)
	}

	if err := db.Where("mapping_status = ?", teamMappingStatusUnmapped).Count(&unmapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count unmapped: %w", err)
	}

	if err := db.Where("mapping_status = ?", teamMappingStatusSkipped).Count(&skipped).Error; err != nil {
		return nil, fmt.Errorf("failed to count skipped: %w", err)
	}

	return map[string]interface{}{
		"total":    total,
		"mapped":   mapped,
		"unmapped": unmapped,
		"skipped":  skipped,
	}, nil
}

// UpdateTeamMappingDestination updates the destination for a team mapping
func (d *Database) UpdateTeamMappingDestination(ctx context.Context, sourceOrg, sourceTeamSlug, destOrg, destTeamSlug, destTeamName string) error {
	updates := map[string]interface{}{
		"destination_org":       destOrg,
		"destination_team_slug": destTeamSlug,
		"mapping_status":        "mapped",
		"updated_at":            time.Now(),
	}

	if destTeamName != "" {
		updates["destination_team_name"] = destTeamName
	}

	result := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update team mapping destination: %w", result.Error)
	}

	return nil
}

// UpdateTeamMappingStatus updates the mapping status for a team
func (d *Database) UpdateTeamMappingStatus(ctx context.Context, sourceOrg, sourceTeamSlug, status string) error {
	result := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Updates(map[string]interface{}{
			"mapping_status": status,
			"updated_at":     time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update team mapping status: %w", result.Error)
	}

	return nil
}

// BulkCreateTeamMappings creates multiple team mappings efficiently
func (d *Database) BulkCreateTeamMappings(ctx context.Context, mappings []*models.TeamMapping) error {
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
			m.MappingStatus = teamMappingStatusUnmapped
		}
	}

	result := d.db.WithContext(ctx).CreateInBatches(mappings, 100)
	if result.Error != nil {
		return fmt.Errorf("failed to bulk create team mappings: %w", result.Error)
	}

	return nil
}

// DeleteTeamMapping deletes a team mapping by source org and team slug
func (d *Database) DeleteTeamMapping(ctx context.Context, sourceOrg, sourceTeamSlug string) error {
	result := d.db.WithContext(ctx).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Delete(&models.TeamMapping{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete team mapping: %w", result.Error)
	}

	return nil
}

// DeleteAllTeamMappings removes all team mappings
func (d *Database) DeleteAllTeamMappings(ctx context.Context) error {
	result := d.db.WithContext(ctx).Where("1=1").Delete(&models.TeamMapping{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete team mappings: %w", result.Error)
	}
	return nil
}

// CreateTeamMappingFromTeam creates a team mapping from a discovered team
func (d *Database) CreateTeamMappingFromTeam(ctx context.Context, team *models.GitHubTeam) error {
	mapping := &models.TeamMapping{
		SourceOrg:      team.Organization,
		SourceTeamSlug: team.Slug,
		SourceTeamName: &team.Name,
		MappingStatus:  teamMappingStatusUnmapped,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return d.SaveTeamMapping(ctx, mapping)
}

// SyncTeamMappingsFromTeams creates team mappings for all discovered teams that don't have one
func (d *Database) SyncTeamMappingsFromTeams(ctx context.Context) (int64, error) {
	// Find teams without mappings and create mappings for them
	var teams []*models.GitHubTeam
	err := d.db.WithContext(ctx).
		Raw(`
			SELECT t.* FROM github_teams t
			LEFT JOIN team_mappings m ON t.organization = m.source_org AND t.slug = m.source_team_slug
			WHERE m.id IS NULL
		`).
		Scan(&teams).Error

	if err != nil {
		return 0, fmt.Errorf("failed to find teams without mappings: %w", err)
	}

	if len(teams) == 0 {
		return 0, nil
	}

	mappings := make([]*models.TeamMapping, 0, len(teams))
	now := time.Now()
	for _, team := range teams {
		mappings = append(mappings, &models.TeamMapping{
			SourceOrg:      team.Organization,
			SourceTeamSlug: team.Slug,
			SourceTeamName: &team.Name,
			MappingStatus:  teamMappingStatusUnmapped,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	if err := d.BulkCreateTeamMappings(ctx, mappings); err != nil {
		return 0, err
	}

	return int64(len(mappings)), nil
}

// SuggestTeamMappings suggests mappings based on matching team slugs in destination
// Returns a map of source team full slug to suggested destination team full slug
func (d *Database) SuggestTeamMappings(ctx context.Context, destinationOrg string, existingDestTeams []string) (map[string]string, error) {
	// Get all unmapped team mappings
	var unmapped []*models.TeamMapping
	err := d.db.WithContext(ctx).
		Where("mapping_status = ?", teamMappingStatusUnmapped).
		Find(&unmapped).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get unmapped teams: %w", err)
	}

	// Create a map of existing destination team slugs for quick lookup
	destTeamSet := make(map[string]bool)
	for _, t := range existingDestTeams {
		destTeamSet[t] = true
	}

	suggestions := make(map[string]string)
	for _, mapping := range unmapped {
		// Check if a team with the same slug exists in destination
		destSlug := destinationOrg + "/" + mapping.SourceTeamSlug
		if destTeamSet[mapping.SourceTeamSlug] {
			suggestions[mapping.SourceFullSlug()] = destSlug
		}
	}

	return suggestions, nil
}

// TeamWithMapping represents a team with its mapping status
type TeamWithMapping struct {
	// Team fields
	ID           int64   `json:"id"`
	Organization string  `json:"organization"`
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	Privacy      string  `json:"privacy"`

	// Mapping fields
	DestinationOrg      *string `json:"destination_org,omitempty"`
	DestinationTeamSlug *string `json:"destination_team_slug,omitempty"`
	DestinationTeamName *string `json:"destination_team_name,omitempty"`
	MappingStatus       string  `json:"mapping_status"` // "unmapped", "mapped", "skipped"
}

// TeamWithMappingFilters defines filters for listing teams with mappings
type TeamWithMappingFilters struct {
	Organization string // Filter by source organization
	Status       string // Filter by mapping status
	Search       string // Search in slug, name
	Limit        int
	Offset       int
}

// ListTeamsWithMappings returns discovered teams with their mapping info
func (d *Database) ListTeamsWithMappings(ctx context.Context, filters TeamWithMappingFilters) ([]TeamWithMapping, int64, error) {
	query := d.db.WithContext(ctx).
		Table("github_teams t").
		Select(`
			t.id,
			t.organization,
			t.slug,
			t.name,
			t.description,
			t.privacy,
			m.destination_org,
			m.destination_team_slug,
			m.destination_team_name,
			COALESCE(m.mapping_status, 'unmapped') as mapping_status
		`).
		Joins("LEFT JOIN team_mappings m ON t.organization = m.source_org AND t.slug = m.source_team_slug")

	// Apply organization filter
	if filters.Organization != "" {
		query = query.Where("t.organization = ?", filters.Organization)
	}

	// Apply status filter
	if filters.Status != "" {
		if filters.Status == teamMappingStatusUnmapped {
			query = query.Where("m.id IS NULL OR m.mapping_status = ?", teamMappingStatusUnmapped)
		} else {
			query = query.Where("m.mapping_status = ?", filters.Status)
		}
	}

	// Apply search filter
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where("t.slug LIKE ? OR t.name LIKE ? OR m.destination_team_slug LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	var total int64
	countQuery := d.db.WithContext(ctx).
		Table("github_teams t").
		Joins("LEFT JOIN team_mappings m ON t.organization = m.source_org AND t.slug = m.source_team_slug")

	if filters.Organization != "" {
		countQuery = countQuery.Where("t.organization = ?", filters.Organization)
	}
	if filters.Status != "" {
		if filters.Status == teamMappingStatusUnmapped {
			countQuery = countQuery.Where("m.id IS NULL OR m.mapping_status = ?", teamMappingStatusUnmapped)
		} else {
			countQuery = countQuery.Where("m.mapping_status = ?", filters.Status)
		}
	}
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		countQuery = countQuery.Where("t.slug LIKE ? OR t.name LIKE ? OR m.destination_team_slug LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count teams: %w", err)
	}

	// Apply pagination and ordering
	query = query.Order("t.organization ASC, t.slug ASC")
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	var results []TeamWithMapping
	if err := query.Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list teams with mappings: %w", err)
	}

	return results, total, nil
}

// GetTeamsWithMappingsStats returns stats for teams with their mapping status
func (d *Database) GetTeamsWithMappingsStats(ctx context.Context) (map[string]interface{}, error) {
	var total int64
	if err := d.db.WithContext(ctx).Model(&models.GitHubTeam{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total teams: %w", err)
	}

	var mapped int64
	if err := d.db.WithContext(ctx).
		Table("team_mappings").
		Where("mapping_status = ?", teamMappingStatusMapped).
		Count(&mapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count mapped: %w", err)
	}

	var skipped int64
	if err := d.db.WithContext(ctx).
		Table("team_mappings").
		Where("mapping_status = ?", teamMappingStatusSkipped).
		Count(&skipped).Error; err != nil {
		return nil, fmt.Errorf("failed to count skipped: %w", err)
	}

	unmapped := total - mapped - skipped

	return map[string]interface{}{
		"total":    total,
		"unmapped": unmapped,
		"mapped":   mapped,
		"skipped":  skipped,
	}, nil
}
