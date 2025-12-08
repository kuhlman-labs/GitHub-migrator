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
	updates := map[string]interface{}{
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
func (d *Database) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
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

	return map[string]interface{}{
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
	DestinationLogin *string `json:"destination_login,omitempty"`
	MappingStatus    string  `json:"mapping_status"` // "unmapped", "mapped", "reclaimed", "skipped"
	MannequinID      *string `json:"mannequin_id,omitempty"`
	MannequinLogin   *string `json:"mannequin_login,omitempty"`
	ReclaimStatus    *string `json:"reclaim_status,omitempty"`
}

// UserWithMappingFilters defines filters for listing users with mappings
type UserWithMappingFilters struct {
	Status string // Filter by mapping status (unmapped, mapped, reclaimed, skipped)
	Search string // Search in login, email, name
	Limit  int
	Offset int
}

// ListUsersWithMappings returns discovered users with their mapping info
// This provides a unified view without needing to sync
func (d *Database) ListUsersWithMappings(ctx context.Context, filters UserWithMappingFilters) ([]UserWithMapping, int64, error) {
	// Build the query with LEFT JOIN to get mapping info
	query := d.db.WithContext(ctx).
		Table("github_users u").
		Select(`
			u.id,
			u.login,
			u.name,
			u.email,
			u.avatar_url,
			u.source_instance,
			m.destination_login,
			COALESCE(m.mapping_status, 'unmapped') as mapping_status,
			m.mannequin_id,
			m.mannequin_login,
			m.reclaim_status
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
func (d *Database) GetUsersWithMappingsStats(ctx context.Context) (map[string]interface{}, error) {
	var total int64
	if err := d.db.WithContext(ctx).Model(&models.GitHubUser{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}

	var mapped int64
	if err := d.db.WithContext(ctx).
		Table("user_mappings").
		Where("mapping_status = 'mapped'").
		Count(&mapped).Error; err != nil {
		return nil, fmt.Errorf("failed to count mapped: %w", err)
	}

	var reclaimed int64
	if err := d.db.WithContext(ctx).
		Table("user_mappings").
		Where("mapping_status = 'reclaimed'").
		Count(&reclaimed).Error; err != nil {
		return nil, fmt.Errorf("failed to count reclaimed: %w", err)
	}

	var skipped int64
	if err := d.db.WithContext(ctx).
		Table("user_mappings").
		Where("mapping_status = 'skipped'").
		Count(&skipped).Error; err != nil {
		return nil, fmt.Errorf("failed to count skipped: %w", err)
	}

	unmapped := total - mapped - reclaimed - skipped

	return map[string]interface{}{
		"total":     total,
		"unmapped":  unmapped,
		"mapped":    mapped,
		"reclaimed": reclaimed,
		"skipped":   skipped,
	}, nil
}
