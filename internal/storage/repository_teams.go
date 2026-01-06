package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// SaveTeam inserts or updates a GitHub team in the database
func (d *Database) SaveTeam(ctx context.Context, team *models.GitHubTeam) error {
	// Check if team already exists
	var existing models.GitHubTeam
	err := d.db.WithContext(ctx).
		Where("organization = ? AND slug = ?", team.Organization, team.Slug).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Insert new team
		if team.DiscoveredAt.IsZero() {
			team.DiscoveredAt = time.Now()
		}
		if team.UpdatedAt.IsZero() {
			team.UpdatedAt = time.Now()
		}

		result := d.db.WithContext(ctx).Create(team)
		if result.Error != nil {
			return fmt.Errorf("failed to create team: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing team: %w", err)
	}

	// Team exists - update it
	team.ID = existing.ID
	team.DiscoveredAt = existing.DiscoveredAt
	team.UpdatedAt = time.Now()

	result := d.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"name":        team.Name,
		"description": team.Description,
		"privacy":     team.Privacy,
		"updated_at":  team.UpdatedAt,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update team: %w", result.Error)
	}

	return nil
}

// SaveTeamRepository creates or updates a team-repository association
// repoFullName should be in "org/repo" format
func (d *Database) SaveTeamRepository(ctx context.Context, teamID int64, repoFullName, permission string) error {
	// First, find the repository by full name
	var repo models.Repository
	err := d.db.WithContext(ctx).
		Where("full_name = ?", repoFullName).
		First(&repo).Error

	if err == gorm.ErrRecordNotFound {
		// Repository not found in our database - skip it
		// This can happen if the team has access to repos we haven't discovered yet
		// The calling code in collector.go logs the repo count found for the team
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to find repository: %w", err)
	}

	// Check if association already exists
	var existing models.GitHubTeamRepository
	err = d.db.WithContext(ctx).
		Where("team_id = ? AND repository_id = ?", teamID, repo.ID).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new association
		association := &models.GitHubTeamRepository{
			TeamID:       teamID,
			RepositoryID: repo.ID,
			Permission:   permission,
			DiscoveredAt: time.Now(),
		}

		result := d.db.WithContext(ctx).Create(association)
		if result.Error != nil {
			return fmt.Errorf("failed to create team-repository association: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing association: %w", err)
	}

	// Association exists - update permission if changed
	if existing.Permission != permission {
		result := d.db.WithContext(ctx).Model(&existing).Update("permission", permission)
		if result.Error != nil {
			return fmt.Errorf("failed to update team-repository association: %w", result.Error)
		}
	}

	return nil
}

// ListTeams returns all teams with optional organization filter
func (d *Database) ListTeams(ctx context.Context, orgFilter string) ([]*models.GitHubTeam, error) {
	var teams []*models.GitHubTeam
	query := d.db.WithContext(ctx)

	if orgFilter != "" {
		// Support comma-separated organization list
		if strings.Contains(orgFilter, ",") {
			orgs := strings.Split(orgFilter, ",")
			trimmedOrgs := make([]string, len(orgs))
			for i, org := range orgs {
				trimmedOrgs[i] = strings.TrimSpace(org)
			}
			query = query.Where("organization IN ?", trimmedOrgs)
		} else {
			query = query.Where("organization = ?", orgFilter)
		}
	}

	err := query.Order("organization ASC, name ASC").Find(&teams).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	return teams, nil
}

// GetTeamByOrgAndSlug retrieves a team by organization and slug
func (d *Database) GetTeamByOrgAndSlug(ctx context.Context, org, slug string) (*models.GitHubTeam, error) {
	var team models.GitHubTeam
	err := d.db.WithContext(ctx).
		Where("organization = ? AND slug = ?", org, slug).
		First(&team).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return &team, nil
}

// GetTeamsForRepository returns all teams that have access to a repository
func (d *Database) GetTeamsForRepository(ctx context.Context, repoID int64) ([]*models.GitHubTeam, error) {
	var teams []*models.GitHubTeam
	err := d.db.WithContext(ctx).
		Joins("JOIN github_team_repositories gtr ON gtr.team_id = github_teams.id").
		Where("gtr.repository_id = ?", repoID).
		Order("organization ASC, name ASC").
		Find(&teams).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get teams for repository: %w", err)
	}

	return teams, nil
}

// DeleteTeamsForOrganization removes all teams and their repository associations for an organization
// This is useful when re-running discovery for an organization
func (d *Database) DeleteTeamsForOrganization(ctx context.Context, org string) error {
	// Use transaction to ensure atomicity
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, delete all team members for teams in this org
		result := tx.Exec(`
			DELETE FROM github_team_members 
			WHERE team_id IN (SELECT id FROM github_teams WHERE organization = ?)
		`, org)
		if result.Error != nil {
			return fmt.Errorf("failed to delete team members: %w", result.Error)
		}

		// Then delete all team-repository associations for teams in this org
		result = tx.Exec(`
			DELETE FROM github_team_repositories 
			WHERE team_id IN (SELECT id FROM github_teams WHERE organization = ?)
		`, org)
		if result.Error != nil {
			return fmt.Errorf("failed to delete team-repository associations: %w", result.Error)
		}

		// Then delete the teams themselves
		result = tx.Where("organization = ?", org).Delete(&models.GitHubTeam{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete teams: %w", result.Error)
		}

		return nil
	})
}

// SaveTeamMember inserts or updates a team member
func (d *Database) SaveTeamMember(ctx context.Context, member *models.GitHubTeamMember) error {
	// Check if member already exists
	var existing models.GitHubTeamMember
	err := d.db.WithContext(ctx).
		Where("team_id = ? AND login = ?", member.TeamID, member.Login).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Insert new member
		if member.DiscoveredAt.IsZero() {
			member.DiscoveredAt = time.Now()
		}

		result := d.db.WithContext(ctx).Create(member)
		if result.Error != nil {
			return fmt.Errorf("failed to create team member: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing team member: %w", err)
	}

	// Member exists - update role if changed
	if existing.Role != member.Role {
		result := d.db.WithContext(ctx).Model(&existing).Update("role", member.Role)
		if result.Error != nil {
			return fmt.Errorf("failed to update team member role: %w", result.Error)
		}
	}

	member.ID = existing.ID
	return nil
}

// GetTeamMembers returns all members of a team
func (d *Database) GetTeamMembers(ctx context.Context, teamID int64) ([]*models.GitHubTeamMember, error) {
	var members []*models.GitHubTeamMember
	err := d.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("login ASC").
		Find(&members).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	return members, nil
}

// GetTeamMembersByOrgAndSlug returns all members of a team by organization and slug
func (d *Database) GetTeamMembersByOrgAndSlug(ctx context.Context, org, slug string) ([]*models.GitHubTeamMember, error) {
	team, err := d.GetTeamByOrgAndSlug(ctx, org, slug)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return []*models.GitHubTeamMember{}, nil
	}

	return d.GetTeamMembers(ctx, team.ID)
}

// GetTeamsForUser returns all teams that a user is a member of
func (d *Database) GetTeamsForUser(ctx context.Context, login string) ([]*models.GitHubTeam, error) {
	var teams []*models.GitHubTeam
	err := d.db.WithContext(ctx).
		Joins("JOIN github_team_members gtm ON gtm.team_id = github_teams.id").
		Where("gtm.login = ?", login).
		Order("organization ASC, name ASC").
		Find(&teams).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get teams for user: %w", err)
	}

	return teams, nil
}

// GetTeamMemberCount returns the number of members in a team
func (d *Database) GetTeamMemberCount(ctx context.Context, teamID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&models.GitHubTeamMember{}).
		Where("team_id = ?", teamID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count team members: %w", err)
	}

	return count, nil
}

// DeleteTeamMembers removes all members from a team
func (d *Database) DeleteTeamMembers(ctx context.Context, teamID int64) error {
	result := d.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Delete(&models.GitHubTeamMember{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete team members: %w", result.Error)
	}

	return nil
}

// BulkSaveTeamMembers saves multiple team members efficiently
func (d *Database) BulkSaveTeamMembers(ctx context.Context, members []*models.GitHubTeamMember) error {
	if len(members) == 0 {
		return nil
	}

	now := time.Now()
	for _, m := range members {
		if m.DiscoveredAt.IsZero() {
			m.DiscoveredAt = now
		}
	}

	// Use upsert-style insert with conflict handling
	for _, member := range members {
		if err := d.SaveTeamMember(ctx, member); err != nil {
			return err
		}
	}

	return nil
}

// GetAllUniqueTeamMemberLogins returns all unique user logins from team members
func (d *Database) GetAllUniqueTeamMemberLogins(ctx context.Context) ([]string, error) {
	var logins []string
	err := d.db.WithContext(ctx).Model(&models.GitHubTeamMember{}).
		Distinct("login").
		Pluck("login", &logins).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get unique team member logins: %w", err)
	}

	return logins, nil
}

// TeamDetailMember represents a team member in the team detail response
type TeamDetailMember struct {
	Login string `json:"login"`
	Role  string `json:"role"`
}

// TeamDetailRepository represents a repository in the team detail response
type TeamDetailRepository struct {
	FullName        string  `json:"full_name"`
	Permission      string  `json:"permission"`
	MigrationStatus *string `json:"migration_status,omitempty"`
}

// TeamDetailMapping represents the mapping info in the team detail response
type TeamDetailMapping struct {
	DestinationOrg      *string    `json:"destination_org,omitempty"`
	DestinationTeamSlug *string    `json:"destination_team_slug,omitempty"`
	MappingStatus       string     `json:"mapping_status"`
	MigrationStatus     string     `json:"migration_status,omitempty"`
	MigratedAt          *time.Time `json:"migrated_at,omitempty"`
	ReposSynced         int        `json:"repos_synced"`
	ErrorMessage        *string    `json:"error_message,omitempty"`
	// New fields for tracking partial vs. full migration
	TotalSourceRepos      int        `json:"total_source_repos"`
	ReposEligible         int        `json:"repos_eligible"`
	TeamCreatedInDest     bool       `json:"team_created_in_dest"`
	LastSyncedAt          *time.Time `json:"last_synced_at,omitempty"`
	MigrationCompleteness string     `json:"migration_completeness"` // pending, team_only, partial, complete, needs_sync
	SyncStatus            string     `json:"sync_status"`            // Derived status for UI
}

// TeamDetail represents comprehensive team information
type TeamDetail struct {
	ID           int64                  `json:"id"`
	Organization string                 `json:"organization"`
	Slug         string                 `json:"slug"`
	Name         string                 `json:"name"`
	Description  *string                `json:"description,omitempty"`
	Privacy      string                 `json:"privacy"`
	DiscoveredAt time.Time              `json:"discovered_at"`
	Members      []TeamDetailMember     `json:"members"`
	Repositories []TeamDetailRepository `json:"repositories"`
	Mapping      *TeamDetailMapping     `json:"mapping,omitempty"`
}

// calculateSyncStatus determines the sync status based on current state
// This is used to calculate the status dynamically at query time
func calculateSyncStatus(mapping models.TeamMapping, dynamicReposEligible int) string {
	if mapping.MigrationStatus == "" || mapping.MigrationStatus == TeamMigrationStatusPending {
		return TeamMigrationStatusPending
	}
	if mapping.MigrationStatus == TeamMigrationStatusFailed {
		return "failed"
	}
	if mapping.TeamCreatedInDest && dynamicReposEligible == 0 {
		return "team_only"
	}
	if mapping.ReposSynced == 0 && dynamicReposEligible > 0 {
		return "needs_sync"
	}
	if mapping.ReposSynced < dynamicReposEligible {
		return "partial"
	}
	if mapping.ReposSynced >= dynamicReposEligible && dynamicReposEligible > 0 {
		return "complete"
	}
	return TeamMigrationStatusPending
}

// GetTeamDetail retrieves comprehensive team information including members, repos, and mapping
func (d *Database) GetTeamDetail(ctx context.Context, org, slug string) (*TeamDetail, error) {
	// Get the team
	team, err := d.GetTeamByOrgAndSlug(ctx, org, slug)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, nil
	}

	detail := &TeamDetail{
		ID:           team.ID,
		Organization: team.Organization,
		Slug:         team.Slug,
		Name:         team.Name,
		Description:  team.Description,
		Privacy:      team.Privacy,
		DiscoveredAt: team.DiscoveredAt,
		Members:      []TeamDetailMember{},
		Repositories: []TeamDetailRepository{},
	}

	// Get team members
	members, err := d.GetTeamMembers(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	for _, m := range members {
		detail.Members = append(detail.Members, TeamDetailMember{
			Login: m.Login,
			Role:  m.Role,
		})
	}

	// Get team repositories with their migration status
	var repoDetails []struct {
		FullName        string  `gorm:"column:full_name"`
		Permission      string  `gorm:"column:permission"`
		MigrationStatus *string `gorm:"column:status"`
	}

	err = d.db.WithContext(ctx).
		Table("github_team_repositories gtr").
		Select("r.full_name, gtr.permission, r.status").
		Joins("JOIN repositories r ON r.id = gtr.repository_id").
		Where("gtr.team_id = ?", team.ID).
		Order("r.full_name ASC").
		Scan(&repoDetails).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get team repositories: %w", err)
	}

	// Count migrated repos dynamically (repos with status "complete")
	migratedRepoCount := 0
	for _, r := range repoDetails {
		detail.Repositories = append(detail.Repositories, TeamDetailRepository{
			FullName:        r.FullName,
			Permission:      r.Permission,
			MigrationStatus: r.MigrationStatus,
		})
		if r.MigrationStatus != nil && *r.MigrationStatus == "complete" {
			migratedRepoCount++
		}
	}

	// Get mapping info
	var mapping models.TeamMapping
	err = d.db.WithContext(ctx).
		Where("source_org = ? AND source_team_slug = ?", org, slug).
		First(&mapping).Error

	if err == nil {
		// Calculate repos_eligible dynamically from actual migrated repos
		// This ensures it's always up-to-date even if repos were migrated after team migration
		dynamicReposEligible := migratedRepoCount

		// Build the mapping detail
		mappingDetail := &TeamDetailMapping{
			DestinationOrg:      mapping.DestinationOrg,
			DestinationTeamSlug: mapping.DestinationTeamSlug,
			MappingStatus:       mapping.MappingStatus,
			MigrationStatus:     mapping.MigrationStatus,
			MigratedAt:          mapping.MigratedAt,
			ReposSynced:         mapping.ReposSynced,
			ErrorMessage:        mapping.ErrorMessage,
			TotalSourceRepos:    len(repoDetails), // Use actual count
			ReposEligible:       dynamicReposEligible,
			TeamCreatedInDest:   mapping.TeamCreatedInDest,
			LastSyncedAt:        mapping.LastSyncedAt,
		}

		// Calculate sync status dynamically based on current state
		mappingDetail.MigrationCompleteness = calculateSyncStatus(mapping, dynamicReposEligible)
		mappingDetail.SyncStatus = mappingDetail.MigrationCompleteness

		detail.Mapping = mappingDetail
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get team mapping: %w", err)
	}

	return detail, nil
}

// TeamRepoForMigration represents a repository to apply permissions for during team migration
type TeamRepoForMigration struct {
	SourceFullName string
	DestFullName   string
	Permission     string
}

// GetTeamRepositoriesForMigration returns repositories for a team that have been migrated
// This is used during team migration execution to know which repos to apply permissions to
func (d *Database) GetTeamRepositoriesForMigration(ctx context.Context, teamID int64) ([]TeamRepoForMigration, error) {
	var results []TeamRepoForMigration

	// Join team repositories with repositories table to get destination mapping
	// Only include repos that have been successfully migrated (status = 'complete')
	err := d.db.WithContext(ctx).
		Table("github_team_repositories gtr").
		Select("r.full_name as source_full_name, r.destination_full_name as dest_full_name, gtr.permission").
		Joins("JOIN repositories r ON r.id = gtr.repository_id").
		Where("gtr.team_id = ?", teamID).
		Where("r.status = ?", "complete").
		Where("r.destination_full_name IS NOT NULL AND r.destination_full_name != ''").
		Order("r.full_name ASC").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get team repositories for migration: %w", err)
	}

	return results, nil
}
