package storage

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// GORM Scopes for common repository filters
// Scopes provide a clean way to compose queries

// WithStatus filters repositories by status
func WithStatus(status interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := status.(type) {
		case string:
			if v != "" {
				return db.Where("status = ?", v)
			}
		case []string:
			if len(v) > 0 {
				return db.Where("status IN ?", v)
			}
		}
		return db
	}
}

// WithBatchID filters repositories by batch ID
func WithBatchID(batchID int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if batchID > 0 {
			return db.Where("batch_id = ?", batchID)
		}
		return db
	}
}

// WithSource filters repositories by source
func WithSource(source string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if source != "" {
			return db.Where("source = ?", source)
		}
		return db
	}
}

// WithSizeRange filters repositories by size range
func WithSizeRange(minSize, maxSize int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if minSize > 0 {
			db = db.Where("total_size >= ?", minSize)
		}
		if maxSize > 0 {
			db = db.Where("total_size <= ?", maxSize)
		}
		return db
	}
}

// WithSearch performs case-insensitive search on full_name
func WithSearch(search string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if search != "" {
			return db.Where("LOWER(full_name) LIKE LOWER(?)", "%"+search+"%")
		}
		return db
	}
}

// WithOrganization filters by organization (single or multiple)
func WithOrganization(org interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := org.(type) {
		case string:
			if v != "" {
				return db.Where("LOWER(full_name) LIKE LOWER(?)", v+"/%")
			}
		case []string:
			if len(v) > 0 {
				conditions := make([]string, len(v))
				args := make([]interface{}, len(v))
				for i, o := range v {
					conditions[i] = "LOWER(full_name) LIKE LOWER(?)"
					args[i] = o + "/%"
				}
				return db.Where(strings.Join(conditions, " OR "), args...)
			}
		}
		return db
	}
}

// WithADOProject filters by Azure DevOps project (single or multiple)
func WithADOProject(project interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := project.(type) {
		case string:
			if v != "" {
				return db.Where("ado_project = ?", v)
			}
		case []string:
			if len(v) > 0 {
				return db.Where("ado_project IN ?", v)
			}
		}
		return db
	}
}

// WithVisibility filters by visibility
func WithVisibility(visibility string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if visibility != "" {
			return db.Where("visibility = ?", visibility)
		}
		return db
	}
}

// WithFeatureFlags filters by various feature flags
func WithFeatureFlags(filters map[string]bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		featureColumns := map[string]string{
			// GitHub features
			"has_lfs":                 "has_lfs",
			"has_submodules":          "has_submodules",
			"has_large_files":         "has_large_files",
			"has_actions":             "has_actions",
			"has_wiki":                "has_wiki",
			"has_pages":               "has_pages",
			"has_discussions":         "has_discussions",
			"has_projects":            "has_projects",
			"has_packages":            "has_packages",
			"has_rulesets":            "has_rulesets",
			"is_archived":             "is_archived",
			"is_fork":                 "is_fork",
			"has_code_scanning":       "has_code_scanning",
			"has_dependabot":          "has_dependabot",
			"has_secret_scanning":     "has_secret_scanning",
			"has_codeowners":          "has_codeowners",
			"has_self_hosted_runners": "has_self_hosted_runners",
			"has_release_assets":      "has_release_assets",
			// Azure DevOps features
			"ado_is_git":       "ado_is_git",
			"ado_has_boards":   "ado_has_boards",
			"ado_has_pipelines": "ado_has_pipelines",
			"ado_has_ghas":     "ado_has_ghas",
			"ado_has_wiki":     "ado_has_wiki",
		}

		for key, column := range featureColumns {
			if value, ok := filters[key]; ok {
				db = db.Where(column+" = ?", value)
			}
		}

		// Special handling for branch_protections (checking if count > 0)
		if value, ok := filters["has_branch_protections"]; ok {
			if value {
				db = db.Where("branch_protections > 0")
			} else {
				db = db.Where("(branch_protections = 0 OR branch_protections IS NULL)")
			}
		}

		// Special handling for webhooks (checking if count > 0)
		if value, ok := filters["has_webhooks"]; ok {
			if value {
				db = db.Where("webhook_count > 0")
			} else {
				db = db.Where("(webhook_count = 0 OR webhook_count IS NULL)")
			}
		}

		return db
	}
}

// WithADOCountFilters filters by Azure DevOps count-based fields
func WithADOCountFilters(filters map[string]string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		for key, value := range filters {
			// Support "> 0" syntax for "has any"
			if value == "> 0" || value == ">0" {
				db = db.Where(key+" > 0")
			} else if value == "= 0" || value == "=0" || value == "0" {
				db = db.Where("("+key+" = 0 OR "+key+" IS NULL)")
			} else {
				// Support other operators like ">= 5", etc.
				db = db.Where(key + " " + value)
			}
		}
		return db
	}
}

// WithSizeCategory filters by size category
func WithSizeCategory(categories interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		const (
			MB100 = 100 * 1024 * 1024      // 100MB
			GB1   = 1024 * 1024 * 1024     // 1GB
			GB5   = 5 * 1024 * 1024 * 1024 // 5GB
		)

		var categoryList []string
		switch v := categories.(type) {
		case string:
			categoryList = []string{v}
		case []string:
			categoryList = v
		default:
			return db
		}

		if len(categoryList) == 0 {
			return db
		}

		var conditions []string
		var args []interface{}

		for _, category := range categoryList {
			switch category {
			case "small":
				conditions = append(conditions, "(total_size > 0 AND total_size < ?)")
				args = append(args, MB100)
			case "medium":
				conditions = append(conditions, "(total_size >= ? AND total_size < ?)")
				args = append(args, MB100, GB1)
			case "large":
				conditions = append(conditions, "(total_size >= ? AND total_size < ?)")
				args = append(args, GB1, GB5)
			case "very_large":
				conditions = append(conditions, "(total_size >= ?)")
				args = append(args, GB5)
			case "unknown":
				conditions = append(conditions, "(total_size IS NULL OR total_size = 0)")
			}
		}

		if len(conditions) > 0 {
			db = db.Where(strings.Join(conditions, " OR "), args...)
		}

		return db
	}
}

// WithComplexity filters by complexity category
func WithComplexity(categories interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		var categoryList []string
		switch v := categories.(type) {
		case string:
			categoryList = []string{v}
		case []string:
			categoryList = v
		default:
			return db
		}

		if len(categoryList) == 0 {
			return db
		}

		// Use the stored complexity_score field (supports both GitHub and ADO)
		var conditions []string
		for _, category := range categoryList {
			switch category {
			case "simple":
				conditions = append(conditions, "(COALESCE(complexity_score, 0) <= 5)")
			case "medium":
				conditions = append(conditions, "(complexity_score BETWEEN 6 AND 10)")
			case "complex":
				conditions = append(conditions, "(complexity_score BETWEEN 11 AND 17)")
			case "very_complex":
				conditions = append(conditions, "(complexity_score >= 18)")
			}
		}

		if len(conditions) > 0 {
			db = db.Where(strings.Join(conditions, " OR "))
		}

		return db
	}
}

// WithAvailableForBatch filters repositories that are available for batch assignment
func WithAvailableForBatch() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Exclude repos already in a batch
		db = db.Where("batch_id IS NULL")

		// Exclude repos in certain statuses
		excludedStatuses := []string{
			"complete",
			"queued_for_migration",
			"dry_run_in_progress",
			"dry_run_queued",
			"migrating_content",
			"archive_generating",
			"post_migration",
			"migration_complete",
		}
		db = db.Where("status NOT IN ?", excludedStatuses)

		return db
	}
}

// WithOrdering applies ordering to the query
func WithOrdering(sortBy string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch sortBy {
		case "name":
			return db.Order("full_name ASC")
		case "size":
			return db.Order("total_size DESC")
		case "org":
			return db.Order("full_name ASC") // Already sorts by org/repo
		case "updated":
			return db.Order("updated_at DESC")
		default:
			return db.Order("full_name ASC")
		}
	}
}

// WithPagination applies limit and offset
func WithPagination(limit, offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if limit > 0 {
			db = db.Limit(limit)
		}
		if offset > 0 {
			db = db.Offset(offset)
		}
		return db
	}
}

// buildComplexityScoreSQL builds the complexity score calculation SQL
// This should match the calculation in repository.go
func buildComplexityScoreSQL() string {
	const (
		MB100 = 104857600  // 100MB
		GB1   = 1073741824 // 1GB
		GB5   = 5368709120 // 5GB
	)

	// Use TRUE/FALSE for boolean comparisons (works across all databases: SQLite, PostgreSQL, SQL Server)
	const trueVal = "TRUE"

	return fmt.Sprintf(`(
		-- Size tier scoring (0-9 points)
		(CASE 
			WHEN total_size IS NULL THEN 0
			WHEN total_size < %d THEN 0
			WHEN total_size < %d THEN 1
			WHEN total_size < %d THEN 2
			ELSE 3
		END) * 3 +
		
		-- High impact features (3-4 points each)
		CASE WHEN has_large_files = %s THEN 4 ELSE 0 END +
		CASE WHEN environment_count > 0 THEN 3 ELSE 0 END +
		CASE WHEN secret_count > 0 THEN 3 ELSE 0 END +
		CASE WHEN has_packages = %s THEN 3 ELSE 0 END +
		CASE WHEN has_self_hosted_runners = %s THEN 3 ELSE 0 END +
		
		-- Moderate impact features (2 points each)
		CASE WHEN variable_count > 0 THEN 2 ELSE 0 END +
		CASE WHEN has_discussions = %s THEN 2 ELSE 0 END +
		CASE WHEN release_count > 0 THEN 2 ELSE 0 END +
		CASE WHEN has_lfs = %s THEN 2 ELSE 0 END +
		CASE WHEN has_submodules = %s THEN 2 ELSE 0 END +
		CASE WHEN installed_apps_count > 0 THEN 2 ELSE 0 END +
		
		-- Low impact features (1 point each)
		CASE WHEN has_code_scanning = %s OR has_dependabot = %s OR has_secret_scanning = %s THEN 1 ELSE 0 END +
		CASE WHEN webhook_count > 0 THEN 1 ELSE 0 END +
		CASE WHEN branch_protections > 0 THEN 1 ELSE 0 END +
		CASE WHEN has_rulesets = %s THEN 1 ELSE 0 END +
		CASE WHEN visibility = 'public' THEN 1 ELSE 0 END +
		CASE WHEN visibility = 'internal' THEN 1 ELSE 0 END +
		CASE WHEN has_codeowners = %s THEN 1 ELSE 0 END +
		
		-- Activity-based scoring (0-4 points) using quantiles
		(CASE 
			WHEN (
				(CAST(branch_count AS REAL) / NULLIF((SELECT MAX(branch_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(commit_count AS REAL) / NULLIF((SELECT MAX(commit_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(issue_count AS REAL) / NULLIF((SELECT MAX(issue_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(pull_request_count AS REAL) / NULLIF((SELECT MAX(pull_request_count) FROM repositories WHERE source = 'ghes'), 0)) / 4.0
			) >= 0.75 THEN 4
			WHEN (
				(CAST(branch_count AS REAL) / NULLIF((SELECT MAX(branch_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(commit_count AS REAL) / NULLIF((SELECT MAX(commit_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(issue_count AS REAL) / NULLIF((SELECT MAX(issue_count) FROM repositories WHERE source = 'ghes'), 0) +
				 CAST(pull_request_count AS REAL) / NULLIF((SELECT MAX(pull_request_count) FROM repositories WHERE source = 'ghes'), 0)) / 4.0
			) >= 0.25 THEN 2
			ELSE 0
		END)
	)`, MB100, GB1, GB5,
		trueVal, trueVal, trueVal, // has_large_files, has_packages, has_self_hosted_runners
		trueVal, trueVal, trueVal, // has_discussions, has_lfs, has_submodules
		trueVal, trueVal, trueVal, // has_code_scanning, has_dependabot, has_secret_scanning
		trueVal, trueVal) // has_rulesets, has_codeowners
}
