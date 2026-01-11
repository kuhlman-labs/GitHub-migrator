package storage

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// GORM Scopes for common repository filters
// Scopes provide a clean way to compose queries
// These scopes work with the new related table structure:
//   - repository_git_properties: size, LFS, submodules, etc.
//   - repository_features: wiki, pages, actions, security features, etc.
//   - repository_ado_properties: Azure DevOps specific fields
//   - repository_validation: complexity scores, limit violations

// WithStatus filters repositories by status
func WithStatus(status any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := status.(type) {
		case string:
			if v != "" {
				return db.Where("repositories.status = ?", v)
			}
		case []string:
			if len(v) > 0 {
				return db.Where("repositories.status IN ?", v)
			}
		}
		return db
	}
}

// WithBatchID filters repositories by batch ID
func WithBatchID(batchID int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if batchID > 0 {
			return db.Where("repositories.batch_id = ?", batchID)
		}
		return db
	}
}

// WithSourceID filters repositories by multi-source ID (sources table foreign key)
func WithSourceID(sourceID int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if sourceID > 0 {
			return db.Where("repositories.source_id = ?", sourceID)
		}
		return db
	}
}

// WithSource filters repositories by source
func WithSource(source string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if source != "" {
			return db.Where("repositories.source = ?", source)
		}
		return db
	}
}

// WithSizeRange filters repositories by size range
// This now joins repository_git_properties table
func WithSizeRange(minSize, maxSize int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if minSize > 0 || maxSize > 0 {
			db = db.Joins("LEFT JOIN repository_git_properties rgp ON rgp.repository_id = repositories.id")
		}
		if minSize > 0 {
			db = db.Where("rgp.total_size >= ?", minSize)
		}
		if maxSize > 0 {
			db = db.Where("rgp.total_size <= ?", maxSize)
		}
		return db
	}
}

// WithSearch performs case-insensitive search on full_name
func WithSearch(search string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if search != "" {
			return db.Where("LOWER(repositories.full_name) LIKE LOWER(?)", "%"+search+"%")
		}
		return db
	}
}

// WithOrganization filters by organization (single or multiple)
func WithOrganization(org any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := org.(type) {
		case string:
			if v != "" {
				return db.Where("LOWER(repositories.full_name) LIKE LOWER(?)", v+"/%")
			}
		case []string:
			if len(v) > 0 {
				conditions := make([]string, len(v))
				args := make([]any, len(v))
				for i, o := range v {
					conditions[i] = "LOWER(repositories.full_name) LIKE LOWER(?)"
					args[i] = o + "/%"
				}
				return db.Where(strings.Join(conditions, " OR "), args...)
			}
		}
		return db
	}
}

// WithADOProject filters by Azure DevOps project (single or multiple)
// This now joins repository_ado_properties table
func WithADOProject(project any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := project.(type) {
		case string:
			if v != "" {
				return db.Joins("LEFT JOIN repository_ado_properties rap_proj ON rap_proj.repository_id = repositories.id").
					Where("rap_proj.project = ?", v)
			}
		case []string:
			if len(v) > 0 {
				return db.Joins("LEFT JOIN repository_ado_properties rap_proj ON rap_proj.repository_id = repositories.id").
					Where("rap_proj.project IN ?", v)
			}
		}
		return db
	}
}

// WithADOOrganization filters by Azure DevOps organization (single or multiple)
// This filters repositories where ado_project belongs to the specified organization(s)
func WithADOOrganization(org any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch v := org.(type) {
		case string:
			if v != "" {
				return db.Joins("LEFT JOIN repository_ado_properties rap_org ON rap_org.repository_id = repositories.id").
					Where("rap_org.project IN (SELECT name FROM ado_projects WHERE organization = ?)", v)
			}
		case []string:
			if len(v) > 0 {
				return db.Joins("LEFT JOIN repository_ado_properties rap_org ON rap_org.repository_id = repositories.id").
					Where("rap_org.project IN (SELECT name FROM ado_projects WHERE organization IN ?)", v)
			}
		}
		return db
	}
}

// WithVisibility filters by visibility
func WithVisibility(visibility string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if visibility != "" {
			return db.Where("repositories.visibility = ?", visibility)
		}
		return db
	}
}

// featureFlagColumnMappings contains the column mappings for feature flag filtering
type featureFlagColumnMappings struct {
	gitColumns           map[string]string
	featuresColumns      map[string]string
	featuresCountColumns map[string]string
	adoColumns           map[string]string
	coreColumns          map[string]string
}

// getFeatureFlagMappings returns all column mappings for feature flags
func getFeatureFlagMappings() featureFlagColumnMappings {
	return featureFlagColumnMappings{
		gitColumns: map[string]string{
			"has_lfs":         "has_lfs",
			"has_submodules":  "has_submodules",
			"has_large_files": "has_large_files",
		},
		featuresColumns: map[string]string{
			"has_actions":             "has_actions",
			"has_wiki":                "has_wiki",
			"has_pages":               "has_pages",
			"has_discussions":         "has_discussions",
			"has_projects":            "has_projects",
			"has_packages":            "has_packages",
			"has_rulesets":            "has_rulesets",
			"has_code_scanning":       "has_code_scanning",
			"has_dependabot":          "has_dependabot",
			"has_secret_scanning":     "has_secret_scanning",
			"has_codeowners":          "has_codeowners",
			"has_self_hosted_runners": "has_self_hosted_runners",
			"has_release_assets":      "has_release_assets",
		},
		featuresCountColumns: map[string]string{
			"has_branch_protections": "branch_protections",
			"has_webhooks":           "webhook_count",
			"has_environments":       "environment_count",
			"has_secrets":            "secret_count",
			"has_variables":          "variable_count",
		},
		adoColumns: map[string]string{
			"ado_is_git":        "is_git",
			"ado_has_boards":    "has_boards",
			"ado_has_pipelines": "has_pipelines",
			"ado_has_ghas":      "has_ghas",
			"ado_has_wiki":      "has_wiki",
		},
		coreColumns: map[string]string{
			"is_archived": "is_archived",
			"is_fork":     "is_fork",
		},
	}
}

// applyBoolFiltersWithPrefix applies boolean filters for a column mapping with table prefix
func applyBoolFiltersWithPrefix(db *gorm.DB, tablePrefix string, columns map[string]string, filters map[string]bool) *gorm.DB {
	for key, column := range columns {
		if value, ok := filters[key]; ok {
			db = db.Where(tablePrefix+column+" = ?", value)
		}
	}
	return db
}

// WithFeatureFlags filters by various feature flags
// This now joins the appropriate related tables based on which flags are requested
func WithFeatureFlags(filters map[string]bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		m := getFeatureFlagMappings()

		// Check which joins are needed
		needsGitJoin, needsFeaturesJoin, needsADOJoin := false, false, false
		for key := range filters {
			if _, ok := m.gitColumns[key]; ok {
				needsGitJoin = true
			}
			if _, ok := m.featuresColumns[key]; ok {
				needsFeaturesJoin = true
			}
			if _, ok := m.featuresCountColumns[key]; ok {
				needsFeaturesJoin = true
			}
			if _, ok := m.adoColumns[key]; ok {
				needsADOJoin = true
			}
		}

		// Apply necessary joins
		if needsGitJoin {
			db = db.Joins("LEFT JOIN repository_git_properties rgp_feat ON rgp_feat.repository_id = repositories.id")
		}
		if needsFeaturesJoin {
			db = db.Joins("LEFT JOIN repository_features rf ON rf.repository_id = repositories.id")
		}
		if needsADOJoin {
			db = db.Joins("LEFT JOIN repository_ado_properties rap ON rap.repository_id = repositories.id")
		}

		// Apply filters using helper
		db = applyBoolFiltersWithPrefix(db, "repositories.", m.coreColumns, filters)
		db = applyBoolFiltersWithPrefix(db, "rgp_feat.", m.gitColumns, filters)
		db = applyBoolFiltersWithPrefix(db, "rf.", m.featuresColumns, filters)
		db = applyBoolFiltersWithPrefix(db, "rap.", m.adoColumns, filters)

		// Special handling for count-based feature flags in features table
		for key, column := range m.featuresCountColumns {
			if value, ok := filters[key]; ok {
				if value {
					db = db.Where("rf." + column + " > 0")
				} else {
					db = db.Where("(rf." + column + " = 0 OR rf." + column + " IS NULL)")
				}
			}
		}

		return db
	}
}

// WithADOCountFilters filters by Azure DevOps count-based fields
// This now joins repository_ado_properties table
func WithADOCountFilters(filters map[string]string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(filters) == 0 {
			return db
		}

		// Join the ADO properties table
		db = db.Joins("LEFT JOIN repository_ado_properties rap_count ON rap_count.repository_id = repositories.id")

		// Whitelist of allowed column names to prevent SQL injection
		// Map from filter key to actual column name in repository_ado_properties
		allowedColumns := map[string]string{
			"ado_pull_request_count":     "pull_request_count",
			"ado_work_item_count":        "work_item_count",
			"ado_branch_policy_count":    "branch_policy_count",
			"ado_yaml_pipeline_count":    "yaml_pipeline_count",
			"ado_classic_pipeline_count": "classic_pipeline_count",
			"ado_test_plan_count":        "test_plan_count",
			"ado_package_feed_count":     "package_feed_count",
			"ado_service_hook_count":     "service_hook_count",
		}

		for key, value := range filters {
			// Validate that key is in the whitelist
			column, ok := allowedColumns[key]
			if !ok {
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
				db = db.Where("rap_count."+column+" > ?", numValue)
			case ">=":
				db = db.Where("rap_count."+column+" >= ?", numValue)
			case "<":
				db = db.Where("rap_count."+column+" < ?", numValue)
			case "<=":
				db = db.Where("rap_count."+column+" <= ?", numValue)
			case "=":
				if numValue == 0 {
					// Handle zero specially to include NULL values
					db = db.Where("(rap_count." + column + " = 0 OR rap_count." + column + " IS NULL)")
				} else {
					db = db.Where("rap_count."+column+" = ?", numValue)
				}
			}
		}
		return db
	}
}

// WithSizeCategory filters by size category
// This now joins repository_git_properties table
func WithSizeCategory(categories any) func(db *gorm.DB) *gorm.DB {
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

		// Join the git properties table
		db = db.Joins("LEFT JOIN repository_git_properties rgp_size ON rgp_size.repository_id = repositories.id")

		var conditions []string
		var args []any

		for _, category := range categoryList {
			switch category {
			case "small":
				conditions = append(conditions, "(rgp_size.total_size > 0 AND rgp_size.total_size < ?)")
				args = append(args, MB100)
			case "medium":
				conditions = append(conditions, "(rgp_size.total_size >= ? AND rgp_size.total_size < ?)")
				args = append(args, MB100, GB1)
			case "large":
				conditions = append(conditions, "(rgp_size.total_size >= ? AND rgp_size.total_size < ?)")
				args = append(args, GB1, GB5)
			case "very_large":
				conditions = append(conditions, "(rgp_size.total_size >= ?)")
				args = append(args, GB5)
			case "unknown":
				conditions = append(conditions, "(rgp_size.total_size IS NULL OR rgp_size.total_size = 0)")
			}
		}

		if len(conditions) > 0 {
			db = db.Where(strings.Join(conditions, " OR "), args...)
		}

		return db
	}
}

// WithComplexity filters by complexity category
// This now joins repository_validation table
func WithComplexity(categories any) func(db *gorm.DB) *gorm.DB {
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

		// Join the validation table
		db = db.Joins("LEFT JOIN repository_validation rv ON rv.repository_id = repositories.id")

		// Use the stored complexity_score field (supports both GitHub and ADO)
		var conditions []string
		for _, category := range categoryList {
			switch category {
			case "simple":
				conditions = append(conditions, "(COALESCE(rv.complexity_score, 0) <= 5)")
			case "medium":
				conditions = append(conditions, "(rv.complexity_score BETWEEN 6 AND 10)")
			case "complex":
				conditions = append(conditions, "(rv.complexity_score BETWEEN 11 AND 17)")
			case "very_complex":
				conditions = append(conditions, "(rv.complexity_score >= 18)")
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
		db = db.Where("repositories.batch_id IS NULL")

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
		db = db.Where("repositories.status NOT IN ?", excludedStatuses)

		return db
	}
}

// WithOrdering applies ordering to the query
// For size ordering, this now joins repository_git_properties table
func WithOrdering(sortBy string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch sortBy {
		case "name":
			return db.Order("repositories.full_name ASC")
		case "size":
			// Join git properties for size ordering
			return db.Joins("LEFT JOIN repository_git_properties rgp_order ON rgp_order.repository_id = repositories.id").
				Order("rgp_order.total_size DESC")
		case "org":
			return db.Order("repositories.full_name ASC") // Already sorts by org/repo
		case "updated":
			return db.Order("repositories.updated_at DESC")
		default:
			return db.Order("repositories.full_name ASC")
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
func WithTeam(teamFilter any) func(db *gorm.DB) *gorm.DB {
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
		var args []any

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
