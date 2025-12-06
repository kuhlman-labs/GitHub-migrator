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

	result := d.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
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
		// This can happen if the team has access to repos we haven't discovered
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
		// First, delete all team-repository associations for teams in this org
		// (using subquery to find team IDs)
		result := tx.Exec(`
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
