package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
)

// SaveUser inserts or updates a GitHub user in the database
func (d *Database) SaveUser(ctx context.Context, user *models.GitHubUser) error {
	// Check if user already exists
	var existing models.GitHubUser
	err := d.db.WithContext(ctx).
		Where("login = ?", user.Login).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Insert new user
		if user.DiscoveredAt.IsZero() {
			user.DiscoveredAt = time.Now()
		}
		if user.UpdatedAt.IsZero() {
			user.UpdatedAt = time.Now()
		}

		result := d.db.WithContext(ctx).Create(user)
		if result.Error != nil {
			return fmt.Errorf("failed to create user: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	// User exists - update it (merge contribution stats)
	user.ID = existing.ID
	user.DiscoveredAt = existing.DiscoveredAt
	user.UpdatedAt = time.Now()

	// Update contribution counts (take the max to avoid decreasing)
	if user.CommitCount < existing.CommitCount {
		user.CommitCount = existing.CommitCount
	}
	if user.IssueCount < existing.IssueCount {
		user.IssueCount = existing.IssueCount
	}
	if user.PRCount < existing.PRCount {
		user.PRCount = existing.PRCount
	}
	if user.CommentCount < existing.CommentCount {
		user.CommentCount = existing.CommentCount
	}
	if user.RepositoryCount < existing.RepositoryCount {
		user.RepositoryCount = existing.RepositoryCount
	}

	// Update name and email if provided
	updates := map[string]any{
		"updated_at":       user.UpdatedAt,
		"commit_count":     user.CommitCount,
		"issue_count":      user.IssueCount,
		"pr_count":         user.PRCount,
		"comment_count":    user.CommentCount,
		"repository_count": user.RepositoryCount,
	}

	if user.Name != nil {
		updates["name"] = user.Name
	}
	if user.Email != nil {
		updates["email"] = user.Email
	}
	if user.AvatarURL != nil {
		updates["avatar_url"] = user.AvatarURL
	}

	result := d.db.WithContext(ctx).Model(&existing).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}

	return nil
}

// GetUserByLogin retrieves a user by their login
func (d *Database) GetUserByLogin(ctx context.Context, login string) (*models.GitHubUser, error) {
	var user models.GitHubUser
	err := d.db.WithContext(ctx).
		Where("login = ?", login).
		First(&user).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by their email
func (d *Database) GetUserByEmail(ctx context.Context, email string) (*models.GitHubUser, error) {
	var user models.GitHubUser
	err := d.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// ListUsers returns all discovered users with optional filters
func (d *Database) ListUsers(ctx context.Context, sourceInstance string, limit, offset int) ([]*models.GitHubUser, int64, error) {
	var users []*models.GitHubUser
	var total int64

	query := d.db.WithContext(ctx).Model(&models.GitHubUser{})

	// Filter by source instance if provided
	if sourceInstance != "" {
		query = query.Where("source_instance = ?", sourceInstance)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply pagination and ordering
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Order("commit_count DESC, login ASC").Find(&users).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// GetUserStats returns summary statistics for discovered users
func (d *Database) GetUserStats(ctx context.Context) (map[string]any, error) {
	var totalUsers int64
	var usersWithEmail int64
	var totalCommits int64
	var totalPRs int64
	var totalIssues int64

	db := d.db.WithContext(ctx).Model(&models.GitHubUser{})

	if err := db.Count(&totalUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	if err := db.Where("email IS NOT NULL AND email != ''").Count(&usersWithEmail).Error; err != nil {
		return nil, fmt.Errorf("failed to count users with email: %w", err)
	}

	// Sum contribution stats
	var stats struct {
		TotalCommits int64
		TotalPRs     int64
		TotalIssues  int64
	}
	err := d.db.WithContext(ctx).Model(&models.GitHubUser{}).
		Select("SUM(commit_count) as total_commits, SUM(pr_count) as total_prs, SUM(issue_count) as total_issues").
		Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to sum user stats: %w", err)
	}

	totalCommits = stats.TotalCommits
	totalPRs = stats.TotalPRs
	totalIssues = stats.TotalIssues

	return map[string]any{
		"total_users":      totalUsers,
		"users_with_email": usersWithEmail,
		"total_commits":    totalCommits,
		"total_prs":        totalPRs,
		"total_issues":     totalIssues,
	}, nil
}

// IncrementUserRepositoryCount increments the repository count for a user
func (d *Database) IncrementUserRepositoryCount(ctx context.Context, login string) error {
	result := d.db.WithContext(ctx).Model(&models.GitHubUser{}).
		Where("login = ?", login).
		UpdateColumn("repository_count", gorm.Expr("repository_count + 1"))

	if result.Error != nil {
		return fmt.Errorf("failed to increment repository count: %w", result.Error)
	}

	return nil
}

// DeleteAllUsers removes all users (useful for re-discovery)
func (d *Database) DeleteAllUsers(ctx context.Context) error {
	result := d.db.WithContext(ctx).Where("1=1").Delete(&models.GitHubUser{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete users: %w", result.Error)
	}
	return nil
}

// SaveUserOrgMembership saves or updates a user's organization membership
func (d *Database) SaveUserOrgMembership(ctx context.Context, membership *models.UserOrgMembership) error {
	// Check if membership already exists
	var existing models.UserOrgMembership
	err := d.db.WithContext(ctx).
		Where("user_login = ? AND organization = ?", membership.UserLogin, membership.Organization).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Insert new membership
		if membership.DiscoveredAt.IsZero() {
			membership.DiscoveredAt = time.Now()
		}
		result := d.db.WithContext(ctx).Create(membership)
		if result.Error != nil {
			return fmt.Errorf("failed to create user org membership: %w", result.Error)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check existing membership: %w", err)
	}

	// Membership exists - update role if changed
	if existing.Role != membership.Role {
		result := d.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
			"role": membership.Role,
		})
		if result.Error != nil {
			return fmt.Errorf("failed to update user org membership: %w", result.Error)
		}
	}

	return nil
}

// GetUserOrgMemberships returns all organizations a user belongs to
func (d *Database) GetUserOrgMemberships(ctx context.Context, userLogin string) ([]*models.UserOrgMembership, error) {
	var memberships []*models.UserOrgMembership
	err := d.db.WithContext(ctx).
		Where("user_login = ?", userLogin).
		Order("organization ASC").
		Find(&memberships).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user org memberships: %w", err)
	}

	return memberships, nil
}

// GetOrgMembers returns all users in an organization
func (d *Database) GetOrgMembers(ctx context.Context, organization string) ([]*models.UserOrgMembership, error) {
	var memberships []*models.UserOrgMembership
	err := d.db.WithContext(ctx).
		Where("organization = ?", organization).
		Order("user_login ASC").
		Find(&memberships).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get org members: %w", err)
	}

	return memberships, nil
}

// GetDistinctUserOrgs returns a list of all unique organizations from user memberships
func (d *Database) GetDistinctUserOrgs(ctx context.Context) ([]string, error) {
	var orgs []string
	err := d.db.WithContext(ctx).
		Model(&models.UserOrgMembership{}).
		Distinct("organization").
		Pluck("organization", &orgs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get distinct user orgs: %w", err)
	}

	return orgs, nil
}

// GetPrimaryOrgForUser returns the "primary" org for a user (first org alphabetically, for consistency)
func (d *Database) GetPrimaryOrgForUser(ctx context.Context, userLogin string) (string, error) {
	var membership models.UserOrgMembership
	err := d.db.WithContext(ctx).
		Where("user_login = ?", userLogin).
		Order("organization ASC").
		First(&membership).Error

	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get primary org: %w", err)
	}

	return membership.Organization, nil
}

// UserWithMapping represents a user with their mapping status
type UserWithMapping struct {
	// User fields
	ID             int64   `json:"id"`
	Login          string  `json:"login"`
	Name           *string `json:"name,omitempty"`
	Email          *string `json:"email,omitempty"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	SourceInstance string  `json:"source_instance"`

	// Mapping fields (nil if no mapping exists)
	SourceOrg        *string `json:"source_org,omitempty"`
	DestinationLogin *string `json:"destination_login,omitempty"`
	MappingStatus    string  `json:"mapping_status"` // "unmapped", "mapped", "reclaimed", "skipped"
	MannequinID      *string `json:"mannequin_id,omitempty"`
	MannequinLogin   *string `json:"mannequin_login,omitempty"`
	ReclaimStatus    *string `json:"reclaim_status,omitempty"`
	MatchConfidence  *int    `json:"match_confidence,omitempty"`
	MatchReason      *string `json:"match_reason,omitempty"`
}

// UserWithMappingFilters defines filters for listing users with mappings
type UserWithMappingFilters struct {
	Status    string // Filter by mapping status (unmapped, mapped, reclaimed, skipped)
	Search    string // Search in login, email, name
	SourceOrg string // Filter by source organization
	SourceID  *int   // Filter by source ID (multi-source support)
	Limit     int
	Offset    int
}

// ListUsersWithMappings returns discovered users with their mapping info
// This provides a unified view without needing to sync
func (d *Database) ListUsersWithMappings(ctx context.Context, filters UserWithMappingFilters) ([]UserWithMapping, int64, error) {
	// Build the query with LEFT JOIN to get mapping info
	// source_org comes from user_mappings first, or falls back to user_org_memberships
	query := d.db.WithContext(ctx).
		Table("github_users u").
		Select(`
			u.id,
			u.login,
			u.name,
			u.email,
			u.avatar_url,
			u.source_instance,
			COALESCE(m.source_org, (SELECT organization FROM user_org_memberships WHERE user_login = u.login ORDER BY organization ASC LIMIT 1)) as source_org,
			m.destination_login,
			COALESCE(m.mapping_status, 'unmapped') as mapping_status,
			m.mannequin_id,
			m.mannequin_login,
			m.reclaim_status,
			m.match_confidence,
			m.match_reason
		`).
		Joins("LEFT JOIN user_mappings m ON u.login = m.source_login")

	// Apply status filter
	if filters.Status != "" {
		if filters.Status == "unmapped" {
			query = query.Where("m.id IS NULL OR m.mapping_status = 'unmapped'")
		} else {
			query = query.Where("m.mapping_status = ?", filters.Status)
		}
	}

	// Apply source org filter - check both user_mappings and user_org_memberships
	if filters.SourceOrg != "" {
		query = query.Where(`(m.source_org = ? OR (m.source_org IS NULL AND EXISTS (
			SELECT 1 FROM user_org_memberships WHERE user_login = u.login AND organization = ?
		)))`, filters.SourceOrg, filters.SourceOrg)
	}

	// Apply source ID filter (multi-source support)
	if filters.SourceID != nil {
		query = query.Where("u.source_id = ?", *filters.SourceID)
	}

	// Apply search filter
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where("u.login LIKE ? OR u.email LIKE ? OR u.name LIKE ? OR m.destination_login LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	var total int64
	countQuery := d.db.WithContext(ctx).
		Table("github_users u").
		Joins("LEFT JOIN user_mappings m ON u.login = m.source_login")

	if filters.Status != "" {
		if filters.Status == "unmapped" {
			countQuery = countQuery.Where("m.id IS NULL OR m.mapping_status = 'unmapped'")
		} else {
			countQuery = countQuery.Where("m.mapping_status = ?", filters.Status)
		}
	}
	if filters.SourceOrg != "" {
		countQuery = countQuery.Where(`(m.source_org = ? OR (m.source_org IS NULL AND EXISTS (
			SELECT 1 FROM user_org_memberships WHERE user_login = u.login AND organization = ?
		)))`, filters.SourceOrg, filters.SourceOrg)
	}
	if filters.SourceID != nil {
		countQuery = countQuery.Where("u.source_id = ?", *filters.SourceID)
	}
	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		countQuery = countQuery.Where("u.login LIKE ? OR u.email LIKE ? OR u.name LIKE ? OR m.destination_login LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply pagination and ordering
	query = query.Order("u.login ASC")
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	var results []UserWithMapping
	if err := query.Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users with mappings: %w", err)
	}

	return results, total, nil
}

// GetUsersWithMappingsStats returns stats for users with their mapping status
// If orgFilter is provided, stats are filtered to that organization only
// If sourceID is provided, stats are filtered to that source only (multi-source support)
func (d *Database) GetUsersWithMappingsStats(ctx context.Context, orgFilter string, sourceID *int) (map[string]any, error) {
	// Helper to apply common filters including source_id via join
	applyFilters := func(query *gorm.DB) *gorm.DB {
		if sourceID != nil {
			query = query.
				Joins("JOIN github_users u ON user_mappings.source_login = u.login").
				Where("u.source_id = ?", *sourceID)
		}
		if orgFilter != "" {
			query = query.Where("user_mappings.source_org = ?", orgFilter)
		}
		return query
	}

	var total int64
	baseQuery := applyFilters(d.db.WithContext(ctx).Table("user_mappings"))
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}

	var mapped int64
	mappedQuery := applyFilters(d.db.WithContext(ctx).
		Table("user_mappings").
		Where("user_mappings.mapping_status = 'mapped'"))
	if err := mappedQuery.Count(&mapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count mapped: %w", err)
	}

	var reclaimed int64
	reclaimedQuery := applyFilters(d.db.WithContext(ctx).
		Table("user_mappings").
		Where("user_mappings.mapping_status = 'reclaimed'"))
	if err := reclaimedQuery.Count(&reclaimed).Error; err != nil {
		return nil, fmt.Errorf("failed to count reclaimed: %w", err)
	}

	var skipped int64
	skippedQuery := applyFilters(d.db.WithContext(ctx).
		Table("user_mappings").
		Where("user_mappings.mapping_status = 'skipped'"))
	if err := skippedQuery.Count(&skipped).Error; err != nil {
		return nil, fmt.Errorf("failed to count skipped: %w", err)
	}

	unmapped := total - mapped - reclaimed - skipped

	// Count users who can be invited (mapped + has mannequin + has destination + not already invited/completed)
	var invitable int64
	invitableQuery := applyFilters(d.db.WithContext(ctx).
		Table("user_mappings").
		Where("user_mappings.mapping_status = 'mapped'").
		Where("user_mappings.mannequin_id IS NOT NULL AND user_mappings.mannequin_id != ''").
		Where("user_mappings.destination_login IS NOT NULL AND user_mappings.destination_login != ''").
		Where("(user_mappings.reclaim_status IS NULL OR user_mappings.reclaim_status IN ('pending', 'failed'))"))
	if err := invitableQuery.Count(&invitable).Error; err != nil {
		return nil, fmt.Errorf("failed to count invitable: %w", err)
	}

	return map[string]any{
		"total":     total,
		"unmapped":  unmapped,
		"mapped":    mapped,
		"reclaimed": reclaimed,
		"skipped":   skipped,
		"invitable": invitable,
	}, nil
}
