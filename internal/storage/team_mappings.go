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

// Team migration status constants
const (
	TeamMigrationStatusPending    = "pending"
	TeamMigrationStatusInProgress = "in_progress"
	TeamMigrationStatusCompleted  = "completed"
	TeamMigrationStatusFailed     = "failed"
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
// If orgFilter is provided, stats are filtered to that organization only
func (d *Database) GetTeamMappingStats(ctx context.Context, orgFilter string) (map[string]any, error) {
	var total int64
	var mapped int64
	var unmapped int64
	var skipped int64

	db := d.db.WithContext(ctx).Model(&models.TeamMapping{})
	if orgFilter != "" {
		db = db.Where("source_org = ?", orgFilter)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count mappings: %w", err)
	}

	mappedQuery := d.db.WithContext(ctx).Model(&models.TeamMapping{}).Where("mapping_status = ?", teamMappingStatusMapped)
	if orgFilter != "" {
		mappedQuery = mappedQuery.Where("source_org = ?", orgFilter)
	}
	if err := mappedQuery.Count(&mapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count mapped: %w", err)
	}

	unmappedQuery := d.db.WithContext(ctx).Model(&models.TeamMapping{}).Where("mapping_status = ?", teamMappingStatusUnmapped)
	if orgFilter != "" {
		unmappedQuery = unmappedQuery.Where("source_org = ?", orgFilter)
	}
	if err := unmappedQuery.Count(&unmapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count unmapped: %w", err)
	}

	skippedQuery := d.db.WithContext(ctx).Model(&models.TeamMapping{}).Where("mapping_status = ?", teamMappingStatusSkipped)
	if orgFilter != "" {
		skippedQuery = skippedQuery.Where("source_org = ?", orgFilter)
	}
	if err := skippedQuery.Count(&skipped).Error; err != nil {
		return nil, fmt.Errorf("failed to count skipped: %w", err)
	}

	return map[string]any{
		"total":    total,
		"mapped":   mapped,
		"unmapped": unmapped,
		"skipped":  skipped,
	}, nil
}

// UpdateTeamMappingDestination updates the destination for a team mapping
func (d *Database) UpdateTeamMappingDestination(ctx context.Context, sourceOrg, sourceTeamSlug, destOrg, destTeamSlug, destTeamName string) error {
	updates := map[string]any{
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
		Updates(map[string]any{
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

	// Migration execution fields
	MigrationStatus   string `json:"migration_status"`   // "pending", "in_progress", "completed", "failed"
	ReposSynced       int    `json:"repos_synced"`       // Number of repos with permissions synced
	ReposEligible     int    `json:"repos_eligible"`     // Number of migrated repos eligible for sync
	TotalSourceRepos  int    `json:"total_source_repos"` // Total repos in source org this team has access to
	TeamCreatedInDest bool   `json:"team_created_in_dest"`
	SyncStatus        string `json:"sync_status"` // Derived: "pending", "team_only", "partial", "complete", "needs_sync"
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
// Uses subqueries to calculate repos_eligible and total_source_repos dynamically
func (d *Database) ListTeamsWithMappings(ctx context.Context, filters TeamWithMappingFilters) ([]TeamWithMapping, int64, error) {
	// Subquery to count total repos for each team
	totalReposSubquery := `(SELECT COUNT(*) FROM github_team_repositories gtr WHERE gtr.team_id = t.id)`

	// Subquery to count migrated repos (status = 'complete') for each team
	eligibleReposSubquery := `(SELECT COUNT(*) FROM github_team_repositories gtr 
		JOIN repositories r ON r.id = gtr.repository_id 
		WHERE gtr.team_id = t.id AND r.status = 'complete')`

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
			COALESCE(m.mapping_status, 'unmapped') as mapping_status,
			COALESCE(m.migration_status, 'pending') as migration_status,
			COALESCE(m.repos_synced, 0) as repos_synced,
			` + eligibleReposSubquery + ` as repos_eligible,
			` + totalReposSubquery + ` as total_source_repos,
			COALESCE(m.team_created_in_dest, 0) as team_created_in_dest,
			CASE
				WHEN m.migration_status IS NULL OR m.migration_status = 'pending' THEN 'pending'
				WHEN m.migration_status = 'failed' THEN 'failed'
				WHEN COALESCE(m.team_created_in_dest, 0) = 1 AND ` + eligibleReposSubquery + ` = 0 THEN 'team_only'
				WHEN COALESCE(m.repos_synced, 0) = 0 AND ` + eligibleReposSubquery + ` > 0 THEN 'needs_sync'
				WHEN COALESCE(m.repos_synced, 0) < ` + eligibleReposSubquery + ` THEN 'partial'
				WHEN COALESCE(m.repos_synced, 0) >= ` + eligibleReposSubquery + ` AND ` + eligibleReposSubquery + ` > 0 THEN 'complete'
				ELSE 'pending'
			END as sync_status
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
// If orgFilter is provided, stats are filtered to that organization only
func (d *Database) GetTeamsWithMappingsStats(ctx context.Context, orgFilter string) (map[string]any, error) {
	var total int64
	baseQuery := d.db.WithContext(ctx).Model(&models.GitHubTeam{})
	if orgFilter != "" {
		baseQuery = baseQuery.Where("organization = ?", orgFilter)
	}
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total teams: %w", err)
	}

	// Count mapped teams (teams with mapping status 'mapped')
	// Uses LEFT JOIN to match teams with their mappings
	var mapped int64
	mappedQuery := d.db.WithContext(ctx).
		Table("github_teams").
		Joins("LEFT JOIN team_mappings ON github_teams.organization = team_mappings.source_org AND github_teams.slug = team_mappings.source_team_slug").
		Where("team_mappings.mapping_status = ?", teamMappingStatusMapped)
	if orgFilter != "" {
		mappedQuery = mappedQuery.Where("github_teams.organization = ?", orgFilter)
	}
	if err := mappedQuery.Count(&mapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count mapped: %w", err)
	}

	// Count skipped teams (teams with mapping status 'skipped')
	var skipped int64
	skippedQuery := d.db.WithContext(ctx).
		Table("github_teams").
		Joins("LEFT JOIN team_mappings ON github_teams.organization = team_mappings.source_org AND github_teams.slug = team_mappings.source_team_slug").
		Where("team_mappings.mapping_status = ?", teamMappingStatusSkipped)
	if orgFilter != "" {
		skippedQuery = skippedQuery.Where("github_teams.organization = ?", orgFilter)
	}
	if err := skippedQuery.Count(&skipped).Error; err != nil {
		return nil, fmt.Errorf("failed to count skipped: %w", err)
	}

	// Count unmapped teams (teams with no mapping OR mapping status 'unmapped')
	var unmapped int64
	unmappedQuery := d.db.WithContext(ctx).
		Table("github_teams").
		Joins("LEFT JOIN team_mappings ON github_teams.organization = team_mappings.source_org AND github_teams.slug = team_mappings.source_team_slug").
		Where("team_mappings.id IS NULL OR team_mappings.mapping_status = ?", teamMappingStatusUnmapped)
	if orgFilter != "" {
		unmappedQuery = unmappedQuery.Where("github_teams.organization = ?", orgFilter)
	}
	if err := unmappedQuery.Count(&unmapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count unmapped: %w", err)
	}

	return map[string]any{
		"total":    total,
		"unmapped": unmapped,
		"mapped":   mapped,
		"skipped":  skipped,
	}, nil
}

// UpdateTeamMigrationStatus updates the migration execution status for a team mapping
func (d *Database) UpdateTeamMigrationStatus(ctx context.Context, sourceOrg, sourceTeamSlug, status string, errMsg *string) error {
	updates := map[string]any{
		"migration_status": status,
		"updated_at":       time.Now(),
	}

	if status == TeamMigrationStatusCompleted {
		now := time.Now()
		updates["migrated_at"] = &now
		updates["error_message"] = nil
	}

	if errMsg != nil {
		updates["error_message"] = *errMsg
	}

	result := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update team migration status: %w", result.Error)
	}

	return nil
}

// UpdateTeamReposSynced updates the count of repos with permissions applied
func (d *Database) UpdateTeamReposSynced(ctx context.Context, sourceOrg, sourceTeamSlug string, count int) error {
	now := time.Now()
	result := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Updates(map[string]any{
			"repos_synced":   count,
			"last_synced_at": &now,
			"updated_at":     now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update repos synced count: %w", result.Error)
	}

	return nil
}

// UpdateTeamMigrationTracking updates the migration tracking fields for a team
// This is used during team migration to track progress and state
type TeamMigrationTrackingUpdate struct {
	TeamCreatedInDest *bool // Set to true when team is created in destination
	TotalSourceRepos  *int  // Total repos this team has access to in source
	ReposEligible     *int  // How many repos have been migrated and are available for sync
	ReposSynced       *int  // How many repos had permissions applied
}

func (d *Database) UpdateTeamMigrationTracking(ctx context.Context, sourceOrg, sourceTeamSlug string, update TeamMigrationTrackingUpdate) error {
	updates := map[string]any{
		"updated_at": time.Now(),
	}

	if update.TeamCreatedInDest != nil {
		updates["team_created_in_dest"] = *update.TeamCreatedInDest
	}
	if update.TotalSourceRepos != nil {
		updates["total_source_repos"] = *update.TotalSourceRepos
	}
	if update.ReposEligible != nil {
		updates["repos_eligible"] = *update.ReposEligible
	}
	if update.ReposSynced != nil {
		updates["repos_synced"] = *update.ReposSynced
		now := time.Now()
		updates["last_synced_at"] = &now
	}

	result := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update team migration tracking: %w", result.Error)
	}

	return nil
}

// UpdateTeamReposEligible calculates and updates the repos_eligible count for a team
// This counts migrated repos that the team has access to
func (d *Database) UpdateTeamReposEligible(ctx context.Context, teamID int64, sourceOrg, sourceTeamSlug string) (int, error) {
	// Count repos that are migrated (status = 'complete') and the team has access to
	var count int64
	err := d.db.WithContext(ctx).
		Table("github_team_repositories gtr").
		Joins("JOIN repositories r ON r.id = gtr.repository_id").
		Where("gtr.team_id = ?", teamID).
		Where("r.status = ?", "complete").
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count eligible repos: %w", err)
	}

	// Update the team mapping
	result := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Updates(map[string]any{
			"repos_eligible": int(count),
			"updated_at":     time.Now(),
		})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to update repos eligible: %w", result.Error)
	}

	return int(count), nil
}

// UpdateTeamTotalSourceRepos updates the total_source_repos count for a team
func (d *Database) UpdateTeamTotalSourceRepos(ctx context.Context, teamID int64, sourceOrg, sourceTeamSlug string) (int, error) {
	// Count all repos the team has access to (regardless of migration status)
	var count int64
	err := d.db.WithContext(ctx).
		Table("github_team_repositories").
		Where("team_id = ?", teamID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count total source repos: %w", err)
	}

	// Update the team mapping
	result := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("source_org = ? AND source_team_slug = ?", sourceOrg, sourceTeamSlug).
		Updates(map[string]any{
			"total_source_repos": int(count),
			"updated_at":         time.Now(),
		})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to update total source repos: %w", result.Error)
	}

	return int(count), nil
}

// GetMappedTeamsForMigration returns all team mappings that are ready for migration
// Includes:
//   - Teams with migration_status = 'pending' or 'failed' (not yet migrated or failed)
//   - Teams with migration_status = 'completed' but repos_synced < repos_eligible (need re-sync for new repos)
//
// Uses dynamic calculation of repos_eligible to ensure we pick up newly migrated repos
func (d *Database) GetMappedTeamsForMigration(ctx context.Context, sourceOrgFilter string) ([]*models.TeamMapping, error) {
	var mappings []*models.TeamMapping

	// Subquery to dynamically calculate repos_eligible (migrated repos with status 'complete')
	// This ensures we pick up repos that were migrated after the team was initially migrated
	eligibleReposSubquery := `(
		SELECT COUNT(*) FROM github_team_repositories gtr 
		JOIN github_teams gt ON gt.id = gtr.team_id
		JOIN repositories r ON r.id = gtr.repository_id 
		WHERE gt.organization = team_mappings.source_org 
		AND gt.slug = team_mappings.source_team_slug 
		AND r.status = 'complete'
	)`

	// Build the query with dynamic repos_eligible calculation
	// Either pending/failed OR completed but needs re-sync (repos_synced < dynamically calculated repos_eligible)
	whereClause := fmt.Sprintf(`
		mapping_status = ? AND (
			migration_status IN (?, ?) 
			OR (migration_status = ? AND COALESCE(repos_synced, 0) < %s)
		)
	`, eligibleReposSubquery)

	query := d.db.WithContext(ctx).
		Where(whereClause, teamMappingStatusMapped, TeamMigrationStatusPending, TeamMigrationStatusFailed, TeamMigrationStatusCompleted)

	if sourceOrgFilter != "" {
		query = query.Where("source_org = ?", sourceOrgFilter)
	}

	if err := query.Order("source_org ASC, source_team_slug ASC").Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get mapped teams for migration: %w", err)
	}

	return mappings, nil
}

// GetTeamMigrationExecutionStats returns statistics about team migration execution
func (d *Database) GetTeamMigrationExecutionStats(ctx context.Context) (map[string]any, error) {
	var pending, inProgress, completed, failed, needsSync, teamOnly, partial int64

	db := d.db.WithContext(ctx).Model(&models.TeamMapping{}).Where("mapping_status = ?", teamMappingStatusMapped)

	if err := db.Where("migration_status = ? OR migration_status IS NULL", TeamMigrationStatusPending).Count(&pending).Error; err != nil {
		return nil, fmt.Errorf("failed to count pending: %w", err)
	}

	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("mapping_status = ?", teamMappingStatusMapped).
		Where("migration_status = ?", TeamMigrationStatusInProgress).
		Count(&inProgress).Error; err != nil {
		return nil, fmt.Errorf("failed to count in_progress: %w", err)
	}

	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("mapping_status = ?", teamMappingStatusMapped).
		Where("migration_status = ?", TeamMigrationStatusCompleted).
		Count(&completed).Error; err != nil {
		return nil, fmt.Errorf("failed to count completed: %w", err)
	}

	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("mapping_status = ?", teamMappingStatusMapped).
		Where("migration_status = ?", TeamMigrationStatusFailed).
		Count(&failed).Error; err != nil {
		return nil, fmt.Errorf("failed to count failed: %w", err)
	}

	// Count teams that need re-sync (completed but repos_synced < repos_eligible)
	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("mapping_status = ?", teamMappingStatusMapped).
		Where("migration_status = ?", TeamMigrationStatusCompleted).
		Where("team_created_in_dest = ?", true).
		Where("repos_synced < repos_eligible").
		Count(&needsSync).Error; err != nil {
		return nil, fmt.Errorf("failed to count needs_sync: %w", err)
	}

	// Count teams that are "team only" (team created but no repos eligible yet)
	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("mapping_status = ?", teamMappingStatusMapped).
		Where("team_created_in_dest = ?", true).
		Where("repos_eligible = ?", 0).
		Count(&teamOnly).Error; err != nil {
		return nil, fmt.Errorf("failed to count team_only: %w", err)
	}

	// Count partial migrations (some repos synced but not all)
	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("mapping_status = ?", teamMappingStatusMapped).
		Where("team_created_in_dest = ?", true).
		Where("repos_synced > 0").
		Where("repos_synced < repos_eligible").
		Count(&partial).Error; err != nil {
		return nil, fmt.Errorf("failed to count partial: %w", err)
	}

	// Also get total repos synced and eligible
	var totalReposSynced, totalReposEligible int64
	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Select("COALESCE(SUM(repos_synced), 0)").
		Where("mapping_status = ?", teamMappingStatusMapped).
		Scan(&totalReposSynced).Error; err != nil {
		return nil, fmt.Errorf("failed to sum repos synced: %w", err)
	}

	if err := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Select("COALESCE(SUM(repos_eligible), 0)").
		Where("mapping_status = ?", teamMappingStatusMapped).
		Scan(&totalReposEligible).Error; err != nil {
		return nil, fmt.Errorf("failed to sum repos eligible: %w", err)
	}

	return map[string]any{
		"pending":              pending,
		"in_progress":          inProgress,
		"completed":            completed,
		"failed":               failed,
		"needs_sync":           needsSync,
		"team_only":            teamOnly,
		"partial":              partial,
		"total_repos_synced":   totalReposSynced,
		"total_repos_eligible": totalReposEligible,
	}, nil
}

// ResetTeamMigrationStatus resets all team migration statuses to pending
// Useful for re-running a migration from scratch
func (d *Database) ResetTeamMigrationStatus(ctx context.Context, sourceOrgFilter string) error {
	query := d.db.WithContext(ctx).Model(&models.TeamMapping{}).
		Where("mapping_status = ?", teamMappingStatusMapped)

	if sourceOrgFilter != "" {
		query = query.Where("source_org = ?", sourceOrgFilter)
	}

	result := query.Updates(map[string]any{
		"migration_status":     TeamMigrationStatusPending,
		"migrated_at":          nil,
		"error_message":        nil,
		"repos_synced":         0,
		"repos_eligible":       0,
		"team_created_in_dest": false,
		"last_synced_at":       nil,
		"updated_at":           time.Now(),
	})

	if result.Error != nil {
		return fmt.Errorf("failed to reset team migration status: %w", result.Error)
	}

	return nil
}

// GetTeamSourceOrgs returns all distinct source organizations that have teams
func (d *Database) GetTeamSourceOrgs(ctx context.Context) ([]string, error) {
	var orgs []string
	err := d.db.WithContext(ctx).Model(&models.GitHubTeam{}).
		Distinct("organization").
		Order("organization ASC").
		Pluck("organization", &orgs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get distinct team organizations: %w", err)
	}

	return orgs, nil
}
