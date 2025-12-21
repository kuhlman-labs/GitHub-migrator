package storage

import (
	"fmt"
	"strconv"
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

// WithADOOrganization filters by Azure DevOps organization (single or multiple)
// This filters repositories where ado_project belongs to the specified organization(s)
func WithADOOrganization(org interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := org.(type) {
		case string:
			if v != "" {
				return db.Where("ado_project IN (SELECT name FROM ado_projects WHERE organization = ?)", v)
			}
		case []string:
			if len(v) > 0 {
				return db.Where("ado_project IN (SELECT name FROM ado_projects WHERE organization IN ?)", v)
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
			"ado_is_git":        "ado_is_git",
			"ado_has_boards":    "ado_has_boards",
			"ado_has_pipelines": "ado_has_pipelines",
			"ado_has_ghas":      "ado_has_ghas",
			"ado_has_wiki":      "ado_has_wiki",
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

		// Special handling for environments (checking if count > 0)
		if value, ok := filters["has_environments"]; ok {
			if value {
				db = db.Where("environment_count > 0")
			} else {
				db = db.Where("(environment_count = 0 OR environment_count IS NULL)")
			}
		}

		// Special handling for secrets (checking if count > 0)
		if value, ok := filters["has_secrets"]; ok {
			if value {
				db = db.Where("secret_count > 0")
			} else {
				db = db.Where("(secret_count = 0 OR secret_count IS NULL)")
			}
		}

		// Special handling for variables (checking if count > 0)
		if value, ok := filters["has_variables"]; ok {
			if value {
				db = db.Where("variable_count > 0")
			} else {
				db = db.Where("(variable_count = 0 OR variable_count IS NULL)")
			}
		}

		return db
	}
}

// WithADOCountFilters filters by Azure DevOps count-based fields
func WithADOCountFilters(filters map[string]string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Whitelist of allowed column names to prevent SQL injection
		allowedColumns := map[string]bool{
			"ado_pull_request_count":     true,
			"ado_work_item_count":        true,
			"ado_branch_policy_count":    true,
			"ado_yaml_pipeline_count":    true,
			"ado_classic_pipeline_count": true,
			"ado_test_plan_count":        true,
			"ado_package_feed_count":     true,
			"ado_service_hook_count":     true,
		}

		for key, value := range filters {
			// Validate that key is in the whitelist
			if !allowedColumns[key] {
				// Skip invalid column names silently to prevent information leakage
				continue
			}

			// Parse and validate the value to extract operator and number safely
			operator, numValue, err := parseFilterValue(value)
			if err != nil {
				// Skip invalid values silently
				continue
			}

			// Use parameterized queries to prevent SQL injection
			switch operator {
			case ">":
				db = db.Where(key+" > ?", numValue)
			case ">=":
				db = db.Where(key+" >= ?", numValue)
			case "<":
				db = db.Where(key+" < ?", numValue)
			case "<=":
				db = db.Where(key+" <= ?", numValue)
			case "=":
				if numValue == 0 {
					// Handle zero specially to include NULL values
					db = db.Where("("+key+" = 0 OR "+key+" IS NULL)", numValue)
				} else {
					db = db.Where(key+" = ?", numValue)
				}
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
			"wont_migrate",
			"remediation_required",
			"pre_migration",
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

// WithTeam filters repositories by team membership
// teamFilter accepts values in "org/team-slug" format to uniquely identify teams across organizations
// Supports single value or slice of values
func WithTeam(teamFilter interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		var teamSpecs []string
		switch v := teamFilter.(type) {
		case string:
			if v == "" {
				return db
			}
			teamSpecs = []string{v}
		case []string:
			if len(v) == 0 {
				return db
			}
			teamSpecs = v
		default:
			return db
		}

		// Parse team specs into org/slug pairs and build conditions
		var conditions []string
		var args []interface{}

		for _, spec := range teamSpecs {
			parts := strings.SplitN(spec, "/", 2)
			if len(parts) != 2 {
				// Invalid format, skip
				continue
			}
			org := parts[0]
			slug := parts[1]
			conditions = append(conditions, "(gt.organization = ? AND gt.slug = ?)")
			args = append(args, org, slug)
		}

		if len(conditions) == 0 {
			return db
		}

		// Join with team tables and filter
		// Use EXISTS subquery for better performance with multiple team filters
		subquery := fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM github_team_repositories gtr
				JOIN github_teams gt ON gtr.team_id = gt.id
				WHERE gtr.repository_id = repositories.id
				AND (%s)
			)
		`, strings.Join(conditions, " OR "))

		return db.Where(subquery, args...)
	}
}

// parseFilterValue safely parses a filter value like "> 0", ">=5", "= 0" into operator and numeric value
// Returns the operator, numeric value, and any error
func parseFilterValue(value string) (string, int, error) {
	value = strings.TrimSpace(value)

	// Handle common patterns
	if value == "> 0" || value == ">0" {
		return ">", 0, nil
	}
	if value == "= 0" || value == "=0" || value == "0" {
		return "=", 0, nil
	}

	// Parse operator from the beginning
	var operator string
	var numStr string

	if strings.HasPrefix(value, ">=") {
		operator = ">="
		numStr = strings.TrimSpace(value[2:])
	} else if strings.HasPrefix(value, "<=") {
		operator = "<="
		numStr = strings.TrimSpace(value[2:])
	} else if strings.HasPrefix(value, ">") {
		operator = ">"
		numStr = strings.TrimSpace(value[1:])
	} else if strings.HasPrefix(value, "<") {
		operator = "<"
		numStr = strings.TrimSpace(value[1:])
	} else if strings.HasPrefix(value, "=") {
		operator = "="
		numStr = strings.TrimSpace(value[1:])
	} else {
		// No operator, assume equality
		operator = "="
		numStr = value
	}

	// Parse the numeric value
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid numeric value: %s", numStr)
	}

	// Validate the numeric value is non-negative (count fields can't be negative)
	if num < 0 {
		return "", 0, fmt.Errorf("negative values not allowed")
	}

	return operator, num, nil
}
