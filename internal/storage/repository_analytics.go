package storage

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// OrganizationStats represents statistics for a single organization
type OrganizationStats struct {
	Organization             string         `json:"organization"`
	TotalRepos               int            `json:"total_repos"`
	StatusCounts             map[string]int `json:"status_counts"`
	MigratedCount            int            `json:"migrated_count"`
	InProgressCount          int            `json:"in_progress_count"`
	FailedCount              int            `json:"failed_count"`
	PendingCount             int            `json:"pending_count"`
	MigrationProgressPercent int            `json:"migration_progress_percentage"`
	Enterprise               *string        `json:"enterprise,omitempty"`       // GitHub Enterprise name
	ADOOrganization          *string        `json:"ado_organization,omitempty"` // Azure DevOps organization
	SourceID                 *int64         `json:"source_id,omitempty"`        // Source ID for multi-source support
	SourceName               *string        `json:"source_name,omitempty"`      // Display name of the source
	SourceType               *string        `json:"source_type,omitempty"`      // Type of source (github or azuredevops)
}

// GetOrganizationStats returns repository counts grouped by organization
func (d *Database) GetOrganizationStats(ctx context.Context) ([]*OrganizationStats, error) {
	// Use dialect-specific string functions via DialectDialer interface
	extractOrg := d.dialect.ExtractOrgFromFullName("r.full_name")
	findSlash := d.dialect.FindCharPosition("r.full_name", "/")

	// For ADO repos, we need to extract both org and project
	// full_name format: "org/project/repo" for ADO, "org/repo" for GitHub
	query := fmt.Sprintf(`
		SELECT 
			%s as org,
			r.source,
			a.project as ado_project,
			COUNT(*) as total,
			r.status,
			COUNT(*) as status_count
		FROM repositories r
		LEFT JOIN repository_ado_properties a ON r.id = a.repository_id
		WHERE %s > 0
		AND r.status != 'wont_migrate'
		GROUP BY org, r.source, a.project, r.status
		ORDER BY total DESC, org ASC
	`, extractOrg, findSlash)

	// Use GORM Raw() for analytics query
	type OrgStatusResult struct {
		Org         string
		Source      string
		ADOProject  *string `gorm:"column:ado_project"`
		Total       int
		Status      string
		StatusCount int
	}

	var results []OrgStatusResult
	err := d.db.WithContext(ctx).Raw(query).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get organization stats: %w", err)
	}

	// Build organization stats map
	// For GitHub: key is org name
	// For ADO: key is project name (to group by project)
	orgMap := make(map[string]*OrganizationStats)
	for _, result := range results {
		// Determine the key and populate fields based on source type
		var mapKey string
		var orgStat *OrganizationStats

		if result.Source == models.SourceTypeAzureDevOps && result.ADOProject != nil {
			// For ADO: key is project name, org goes in ADOOrganization field
			mapKey = *result.ADOProject
			if _, exists := orgMap[mapKey]; !exists {
				orgMap[mapKey] = &OrganizationStats{
					Organization:    mapKey,      // Project name
					ADOOrganization: &result.Org, // Org name (extracted from full_name)
					TotalRepos:      0,
					StatusCounts:    make(map[string]int),
				}
			}
			orgStat = orgMap[mapKey]
		} else {
			// For GitHub: key is org name
			mapKey = result.Org
			if _, exists := orgMap[mapKey]; !exists {
				orgMap[mapKey] = &OrganizationStats{
					Organization: mapKey,
					TotalRepos:   0,
					StatusCounts: make(map[string]int),
				}
			}
			orgStat = orgMap[mapKey]
		}

		orgStat.StatusCounts[result.Status] = result.StatusCount
		orgStat.TotalRepos += result.StatusCount

		// Calculate progress metrics
		switch result.Status {
		case string(models.StatusComplete), string(models.StatusMigrationComplete):
			orgStat.MigratedCount += result.StatusCount
		case "migration_failed", "dry_run_failed", "rolled_back":
			orgStat.FailedCount += result.StatusCount
		case "queued_for_migration", "migrating_content", "dry_run_in_progress",
			"dry_run_queued", "pre_migration", "archive_generating", "post_migration":
			orgStat.InProgressCount += result.StatusCount
		default:
			// pending, dry_run_complete, remediation_required
			orgStat.PendingCount += result.StatusCount
		}
	}

	// Convert map to slice and calculate percentages
	stats := make([]*OrganizationStats, 0, len(orgMap))
	for _, stat := range orgMap {
		// Calculate migration progress percentage
		if stat.TotalRepos > 0 {
			stat.MigrationProgressPercent = (stat.MigratedCount * 100) / stat.TotalRepos
		}
		stats = append(stats, stat)
	}

	// Sort by total repos (descending), then by organization name (ascending) for consistent ordering
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalRepos == stats[j].TotalRepos {
			return stats[i].Organization < stats[j].Organization
		}
		return stats[i].TotalRepos > stats[j].TotalRepos
	})

	return stats, nil
}

// SizeDistribution represents repository size distribution
type SizeDistribution struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetSizeDistribution categorizes repositories by size
func (d *Database) GetSizeDistribution(ctx context.Context) ([]*SizeDistribution, error) {
	// Size categories: small (<100MB), medium (100MB-1GB), large (1GB-5GB), very_large (>5GB)
	// Note: PostgreSQL doesn't allow GROUP BY on column aliases, so we use a subquery
	query := `
		SELECT 
			category,
			COUNT(*) as count
		FROM (
			SELECT 
				CASE 
					WHEN gp.total_size IS NULL THEN 'unknown'
					WHEN gp.total_size < 104857600 THEN 'small'
					WHEN gp.total_size < 1073741824 THEN 'medium'
					WHEN gp.total_size < 5368709120 THEN 'large'
					ELSE 'very_large'
				END as category
			FROM repositories r
			LEFT JOIN repository_git_properties gp ON r.id = gp.repository_id
		) categorized
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'small' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'large' THEN 3
				WHEN 'very_large' THEN 4
				WHEN 'unknown' THEN 5
			END
	`

	// Use GORM Raw() for analytics query
	var distribution []*SizeDistribution
	err := d.db.WithContext(ctx).Raw(query).Scan(&distribution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get size distribution: %w", err)
	}

	return distribution, nil
}

// FeatureStats represents aggregated feature usage statistics
type FeatureStats struct {
	// GitHub features
	IsArchived           int `json:"is_archived" gorm:"column:archived_count"`
	IsFork               int `json:"is_fork" gorm:"column:fork_count"`
	HasLFS               int `json:"has_lfs" gorm:"column:lfs_count"`
	HasSubmodules        int `json:"has_submodules" gorm:"column:submodules_count"`
	HasLargeFiles        int `json:"has_large_files" gorm:"column:large_files_count"`
	HasWiki              int `json:"has_wiki" gorm:"column:wiki_count"`
	HasPages             int `json:"has_pages" gorm:"column:pages_count"`
	HasDiscussions       int `json:"has_discussions" gorm:"column:discussions_count"`
	HasActions           int `json:"has_actions" gorm:"column:actions_count"`
	HasProjects          int `json:"has_projects" gorm:"column:projects_count"`
	HasPackages          int `json:"has_packages" gorm:"column:packages_count"`
	HasBranchProtections int `json:"has_branch_protections" gorm:"column:branch_protections_count"`
	HasRulesets          int `json:"has_rulesets" gorm:"column:rulesets_count"`
	HasCodeScanning      int `json:"has_code_scanning" gorm:"column:code_scanning_count"`
	HasDependabot        int `json:"has_dependabot" gorm:"column:dependabot_count"`
	HasSecretScanning    int `json:"has_secret_scanning" gorm:"column:secret_scanning_count"`
	HasCodeowners        int `json:"has_codeowners" gorm:"column:codeowners_count"`
	HasSelfHostedRunners int `json:"has_self_hosted_runners" gorm:"column:self_hosted_runners_count"`
	HasReleaseAssets     int `json:"has_release_assets" gorm:"column:release_assets_count"`
	HasWebhooks          int `json:"has_webhooks" gorm:"column:webhooks_count"`
	HasEnvironments      int `json:"has_environments" gorm:"column:environments_count"`
	HasSecrets           int `json:"has_secrets" gorm:"column:secrets_count"`
	HasVariables         int `json:"has_variables" gorm:"column:variables_count"`

	// Azure DevOps features
	ADOTFVCCount           int `json:"ado_tfvc_count" gorm:"column:ado_tfvc_count"`
	ADOHasBoards           int `json:"ado_has_boards" gorm:"column:ado_has_boards_count"`
	ADOHasPipelines        int `json:"ado_has_pipelines" gorm:"column:ado_has_pipelines_count"`
	ADOHasGHAS             int `json:"ado_has_ghas" gorm:"column:ado_has_ghas_count"`
	ADOHasPullRequests     int `json:"ado_has_pull_requests" gorm:"column:ado_has_pull_requests_count"`
	ADOHasWorkItems        int `json:"ado_has_work_items" gorm:"column:ado_has_work_items_count"`
	ADOHasBranchPolicies   int `json:"ado_has_branch_policies" gorm:"column:ado_has_branch_policies_count"`
	ADOHasYAMLPipelines    int `json:"ado_has_yaml_pipelines" gorm:"column:ado_has_yaml_pipelines_count"`
	ADOHasClassicPipelines int `json:"ado_has_classic_pipelines" gorm:"column:ado_has_classic_pipelines_count"`
	ADOHasWiki             int `json:"ado_has_wiki" gorm:"column:ado_has_wiki_count"`
	ADOHasTestPlans        int `json:"ado_has_test_plans" gorm:"column:ado_has_test_plans_count"`
	ADOHasPackageFeeds     int `json:"ado_has_package_feeds" gorm:"column:ado_has_package_feeds_count"`
	ADOHasServiceHooks     int `json:"ado_has_service_hooks" gorm:"column:ado_has_service_hooks_count"`

	TotalRepositories int `json:"total_repositories" gorm:"column:total"`
}

// GetFeatureStats returns aggregated statistics on feature usage
func (d *Database) GetFeatureStats(ctx context.Context) (*FeatureStats, error) {
	query := `
		SELECT 
			-- GitHub features (from repositories table)
			SUM(CASE WHEN r.is_archived = TRUE THEN 1 ELSE 0 END) as archived_count,
			SUM(CASE WHEN r.is_fork = TRUE THEN 1 ELSE 0 END) as fork_count,
			-- Git properties (from repository_git_properties table)
			SUM(CASE WHEN gp.has_lfs = TRUE THEN 1 ELSE 0 END) as lfs_count,
			SUM(CASE WHEN gp.has_submodules = TRUE THEN 1 ELSE 0 END) as submodules_count,
			SUM(CASE WHEN gp.has_large_files = TRUE THEN 1 ELSE 0 END) as large_files_count,
			-- Features (from repository_features table)
			SUM(CASE WHEN f.has_wiki = TRUE THEN 1 ELSE 0 END) as wiki_count,
			SUM(CASE WHEN f.has_pages = TRUE THEN 1 ELSE 0 END) as pages_count,
			SUM(CASE WHEN f.has_discussions = TRUE THEN 1 ELSE 0 END) as discussions_count,
			SUM(CASE WHEN f.has_actions = TRUE THEN 1 ELSE 0 END) as actions_count,
			SUM(CASE WHEN f.has_projects = TRUE THEN 1 ELSE 0 END) as projects_count,
			SUM(CASE WHEN f.has_packages = TRUE THEN 1 ELSE 0 END) as packages_count,
			SUM(CASE WHEN f.branch_protections > 0 THEN 1 ELSE 0 END) as branch_protections_count,
			SUM(CASE WHEN f.has_rulesets = TRUE THEN 1 ELSE 0 END) as rulesets_count,
			SUM(CASE WHEN f.has_code_scanning = TRUE THEN 1 ELSE 0 END) as code_scanning_count,
			SUM(CASE WHEN f.has_dependabot = TRUE THEN 1 ELSE 0 END) as dependabot_count,
			SUM(CASE WHEN f.has_secret_scanning = TRUE THEN 1 ELSE 0 END) as secret_scanning_count,
			SUM(CASE WHEN f.has_codeowners = TRUE THEN 1 ELSE 0 END) as codeowners_count,
			SUM(CASE WHEN f.has_self_hosted_runners = TRUE THEN 1 ELSE 0 END) as self_hosted_runners_count,
			SUM(CASE WHEN f.has_release_assets = TRUE THEN 1 ELSE 0 END) as release_assets_count,
			SUM(CASE WHEN f.webhook_count > 0 THEN 1 ELSE 0 END) as webhooks_count,
			SUM(CASE WHEN f.environment_count > 0 THEN 1 ELSE 0 END) as environments_count,
			SUM(CASE WHEN f.secret_count > 0 THEN 1 ELSE 0 END) as secrets_count,
			SUM(CASE WHEN f.variable_count > 0 THEN 1 ELSE 0 END) as variables_count,
			
			-- Azure DevOps features (from repository_ado_properties table, only count for ADO sources)
			SUM(CASE WHEN r.source = 'azuredevops' AND a.is_git = FALSE THEN 1 ELSE 0 END) as ado_tfvc_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.has_boards = TRUE THEN 1 ELSE 0 END) as ado_has_boards_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.has_pipelines = TRUE THEN 1 ELSE 0 END) as ado_has_pipelines_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.has_ghas = TRUE THEN 1 ELSE 0 END) as ado_has_ghas_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.pull_request_count > 0 THEN 1 ELSE 0 END) as ado_has_pull_requests_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.work_item_count > 0 THEN 1 ELSE 0 END) as ado_has_work_items_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.branch_policy_count > 0 THEN 1 ELSE 0 END) as ado_has_branch_policies_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.yaml_pipeline_count > 0 THEN 1 ELSE 0 END) as ado_has_yaml_pipelines_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.classic_pipeline_count > 0 THEN 1 ELSE 0 END) as ado_has_classic_pipelines_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.has_wiki = TRUE THEN 1 ELSE 0 END) as ado_has_wiki_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.test_plan_count > 0 THEN 1 ELSE 0 END) as ado_has_test_plans_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.package_feed_count > 0 THEN 1 ELSE 0 END) as ado_has_package_feeds_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND a.service_hook_count > 0 THEN 1 ELSE 0 END) as ado_has_service_hooks_count,
			
			COUNT(*) as total
		FROM repositories r
		LEFT JOIN repository_git_properties gp ON r.id = gp.repository_id
		LEFT JOIN repository_features f ON r.id = f.repository_id
		LEFT JOIN repository_ado_properties a ON r.id = a.repository_id
`

	// Use GORM Raw() for analytics query
	var stats FeatureStats
	err := d.db.WithContext(ctx).Raw(query).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get feature stats: %w", err)
	}

	return &stats, nil
}

// ComplexityDistribution represents repository complexity distribution
type ComplexityDistribution struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetComplexityDistribution categorizes repositories by complexity score
// Uses the stored complexity_score field which is calculated during profiling
// and supports both GitHub and Azure DevOps repositories
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetComplexityDistribution(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) ([]*ComplexityDistribution, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	// Use stored complexity_score field (supports both GitHub and ADO repositories)
	query := `
		SELECT 
			category,
			COUNT(*) as count
		FROM (
			SELECT 
				CASE 
					WHEN COALESCE(v.complexity_score, 0) <= 5 THEN 'simple'
					WHEN v.complexity_score <= 10 THEN 'medium'
					WHEN v.complexity_score <= 17 THEN 'complex'
					ELSE 'very_complex'
				END as category
			FROM repositories r
			LEFT JOIN repository_validation v ON r.id = v.repository_id
			LEFT JOIN repository_ado_properties a ON r.id = a.repository_id
			WHERE 1=1
				AND r.status != 'wont_migrate'
				` + orgFilterSQL + `
				` + projectFilterSQL + `
				` + batchFilterSQL + `
				` + sourceFilterSQL + `
		) as categorized
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'simple' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'complex' THEN 3
				WHEN 'very_complex' THEN 4
			END
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	var distribution []*ComplexityDistribution
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&distribution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get complexity distribution: %w", err)
	}

	return distribution, nil
}

// MigrationVelocity represents migration velocity metrics
type MigrationVelocity struct {
	ReposPerDay  float64 `json:"repos_per_day"`
	ReposPerWeek float64 `json:"repos_per_week"`
}

// GetMigrationVelocity calculates migration velocity over the specified period
func (d *Database) GetMigrationVelocity(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64, days int) (*MigrationVelocity, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	// Use dialect-specific date arithmetic via DialectDialer interface
	var args []any
	dateCondition := "AND mh.completed_at >= " + d.dialect.DateIntervalAgo(days)
	args = append(args, orgArgs...)
	args = append(args, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	query := `
		SELECT COUNT(DISTINCT r.id) as total_completed
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed' 
			AND mh.phase = 'migration'
			` + dateCondition + `
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + projectFilterSQL + `
			` + batchFilterSQL + `
			` + sourceFilterSQL + `
	`

	// Use GORM Raw() for analytics query
	var result struct {
		TotalCompleted int
	}
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration velocity: %w", err)
	}

	velocity := &MigrationVelocity{
		ReposPerDay:  float64(result.TotalCompleted) / float64(days),
		ReposPerWeek: (float64(result.TotalCompleted) / float64(days)) * 7,
	}

	return velocity, nil
}

// MigrationTimeSeriesPoint represents a point in the migration time series
type MigrationTimeSeriesPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetMigrationTimeSeries returns daily migration completions for the last 30 days
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetMigrationTimeSeries(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) ([]*MigrationTimeSeriesPoint, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	// Use dialect-specific date arithmetic via DialectDialer interface
	dateCondition := "AND mh.completed_at >= " + d.dialect.DateIntervalAgo(30)

	query := `
		SELECT 
			DATE(mh.completed_at) as date,
			COUNT(DISTINCT r.id) as count
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed'
			AND mh.phase = 'migration'
			` + dateCondition + `
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + projectFilterSQL + `
			` + batchFilterSQL + `
			` + sourceFilterSQL + `
		GROUP BY DATE(mh.completed_at)
		ORDER BY date ASC
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	var series []*MigrationTimeSeriesPoint
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&series).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get migration time series: %w", err)
	}

	return series, nil
}

// GetAverageMigrationTime calculates the average migration duration
func (d *Database) GetAverageMigrationTime(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) (float64, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	query := `
		SELECT AVG(mh.duration_seconds) as avg_duration
		FROM repositories r
		INNER JOIN migration_history mh ON r.id = mh.repository_id
		WHERE mh.status = 'completed'
			AND mh.phase = 'migration'
			AND mh.duration_seconds IS NOT NULL
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + projectFilterSQL + `
			` + batchFilterSQL + `
			` + sourceFilterSQL + `
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	var result struct {
		AvgDuration *float64
	}
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&result).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get average migration time: %w", err)
	}

	if result.AvgDuration == nil {
		return 0, nil
	}

	return *result.AvgDuration, nil
}

// GetMedianMigrationTime calculates the median migration duration
func (d *Database) GetMedianMigrationTime(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) (float64, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	// Use dialect-specific median calculation via DialectDialer interface
	var query string
	if d.dialect.SupportsPercentileCont() {
		// PostgreSQL and SQL Server support PERCENTILE_CONT
		medianExpr := d.dialect.PercentileMedian("mh.duration_seconds")
		query = `
			SELECT ` + medianExpr + ` as median_duration
			FROM repositories r
			INNER JOIN migration_history mh ON r.id = mh.repository_id
			WHERE mh.status = 'completed'
				AND mh.phase = 'migration'
				AND mh.duration_seconds IS NOT NULL
				AND r.status != 'wont_migrate'
				` + orgFilterSQL + `
				` + projectFilterSQL + `
				` + batchFilterSQL + `
				` + sourceFilterSQL + `
		`
	} else {
		// SQLite - use subquery approach for median
		query = `
			WITH ordered AS (
				SELECT mh.duration_seconds,
					ROW_NUMBER() OVER (ORDER BY mh.duration_seconds) as row_num,
					COUNT(*) OVER () as total_count
				FROM repositories r
				INNER JOIN migration_history mh ON r.id = mh.repository_id
				WHERE mh.status = 'completed'
					AND mh.phase = 'migration'
					AND mh.duration_seconds IS NOT NULL
					AND r.status != 'wont_migrate'
					` + orgFilterSQL + `
					` + projectFilterSQL + `
					` + batchFilterSQL + `
					` + sourceFilterSQL + `
			)
			SELECT AVG(duration_seconds) as median_duration
			FROM ordered
			WHERE row_num IN ((total_count + 1) / 2, (total_count + 2) / 2)
		`
	}

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	var result struct {
		MedianDuration *float64
	}
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&result).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get median migration time: %w", err)
	}

	if result.MedianDuration == nil {
		return 0, nil
	}

	return *result.MedianDuration, nil
}

// buildOrgFilter builds the organization filter clause using parameterized queries
// Returns the SQL fragment and any additional arguments to append
func (d *Database) buildOrgFilter(orgFilter string) (string, []any) {
	if orgFilter == "" {
		return "", nil
	}

	// Use dialect-specific string functions via DialectDialer interface
	extractOrg := d.dialect.ExtractOrgFromFullName("r.full_name")
	filterSQL := fmt.Sprintf(" AND %s = ?", extractOrg)

	return filterSQL, []any{orgFilter}
}

// buildBatchFilter builds the batch filter clause using parameterized queries
// Returns the SQL fragment and any additional arguments to append
func (d *Database) buildBatchFilter(batchFilter string) (string, []any) {
	if batchFilter == "" {
		return "", nil
	}
	// Validate that batchFilter contains only digits
	batchID, err := strconv.ParseInt(batchFilter, 10, 64)
	if err != nil {
		return "", nil
	}
	return " AND r.batch_id = ?", []any{batchID}
}

// buildSourceFilter builds the source_id filter clause using parameterized queries
// Returns the SQL fragment and any additional arguments to append
func (d *Database) buildSourceFilter(sourceID *int64) (string, []any) {
	if sourceID == nil {
		return "", nil
	}
	return " AND r.source_id = ?", []any{*sourceID}
}

// buildProjectFilter builds SQL filter for ADO project (project field in repository_ado_properties)
func (d *Database) buildProjectFilter(projectFilter string) (string, []any) {
	if projectFilter == "" {
		return "", nil
	}
	return " AND a.project = ?", []any{projectFilter}
}

// GetRepositoryStatsByStatusFiltered returns repository counts by status with filters
func (d *Database) GetRepositoryStatsByStatusFiltered(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) (map[string]int, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	query := `
		SELECT r.status, COUNT(*) as count
		FROM repositories r
		LEFT JOIN repository_ado_properties a ON r.id = a.repository_id
		WHERE 1=1
			` + orgFilterSQL + `
			` + projectFilterSQL + `
			` + batchFilterSQL + `
			` + sourceFilterSQL + `
		GROUP BY r.status
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	type StatusCount struct {
		Status string
		Count  int
	}

	var results []StatusCount
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get repository stats: %w", err)
	}

	stats := make(map[string]int)
	for _, result := range results {
		stats[result.Status] = result.Count
	}

	return stats, nil
}

// GetSizeDistributionFiltered returns size distribution with filters
//
//nolint:dupl // Similar query pattern but different business logic
func (d *Database) GetSizeDistributionFiltered(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) ([]*SizeDistribution, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	// Note: PostgreSQL doesn't allow GROUP BY on column aliases, so we use a subquery
	query := `
		SELECT 
			category,
			COUNT(*) as count
		FROM (
			SELECT 
				CASE 
					WHEN gp.total_size IS NULL THEN 'unknown'
					WHEN gp.total_size < 104857600 THEN 'small'
					WHEN gp.total_size < 1073741824 THEN 'medium'
					WHEN gp.total_size < 5368709120 THEN 'large'
					ELSE 'very_large'
				END as category
			FROM repositories r
			LEFT JOIN repository_git_properties gp ON r.id = gp.repository_id
			WHERE 1=1
				AND r.status != 'wont_migrate'
				` + orgFilterSQL + `
				` + projectFilterSQL + `
				` + batchFilterSQL + `
				` + sourceFilterSQL + `
		) categorized
		GROUP BY category
		ORDER BY 
			CASE category
				WHEN 'small' THEN 1
				WHEN 'medium' THEN 2
				WHEN 'large' THEN 3
				WHEN 'very_large' THEN 4
				WHEN 'unknown' THEN 5
			END
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	var distribution []*SizeDistribution
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&distribution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get size distribution: %w", err)
	}

	return distribution, nil
}

// GetFeatureStatsFiltered returns feature stats with filters
func (d *Database) GetFeatureStatsFiltered(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) (*FeatureStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	query := `
		SELECT 
			-- GitHub features (from features and git_properties tables)
			SUM(CASE WHEN r.is_archived = TRUE THEN 1 ELSE 0 END) as archived_count,
			SUM(CASE WHEN r.is_fork = TRUE THEN 1 ELSE 0 END) as fork_count,
			SUM(CASE WHEN gp.has_lfs = TRUE THEN 1 ELSE 0 END) as lfs_count,
			SUM(CASE WHEN gp.has_submodules = TRUE THEN 1 ELSE 0 END) as submodules_count,
			SUM(CASE WHEN gp.has_large_files = TRUE THEN 1 ELSE 0 END) as large_files_count,
			SUM(CASE WHEN f.has_wiki = TRUE THEN 1 ELSE 0 END) as wiki_count,
			SUM(CASE WHEN f.has_pages = TRUE THEN 1 ELSE 0 END) as pages_count,
			SUM(CASE WHEN f.has_discussions = TRUE THEN 1 ELSE 0 END) as discussions_count,
			SUM(CASE WHEN f.has_actions = TRUE THEN 1 ELSE 0 END) as actions_count,
			SUM(CASE WHEN f.has_projects = TRUE THEN 1 ELSE 0 END) as projects_count,
			SUM(CASE WHEN f.has_packages = TRUE THEN 1 ELSE 0 END) as packages_count,
			SUM(CASE WHEN f.branch_protections > 0 THEN 1 ELSE 0 END) as branch_protections_count,
			SUM(CASE WHEN f.has_rulesets = TRUE THEN 1 ELSE 0 END) as rulesets_count,
			SUM(CASE WHEN f.has_code_scanning = TRUE THEN 1 ELSE 0 END) as code_scanning_count,
			SUM(CASE WHEN f.has_dependabot = TRUE THEN 1 ELSE 0 END) as dependabot_count,
			SUM(CASE WHEN f.has_secret_scanning = TRUE THEN 1 ELSE 0 END) as secret_scanning_count,
			SUM(CASE WHEN f.has_codeowners = TRUE THEN 1 ELSE 0 END) as codeowners_count,
			SUM(CASE WHEN f.has_self_hosted_runners = TRUE THEN 1 ELSE 0 END) as self_hosted_runners_count,
			SUM(CASE WHEN f.has_release_assets = TRUE THEN 1 ELSE 0 END) as release_assets_count,
			SUM(CASE WHEN f.webhook_count > 0 THEN 1 ELSE 0 END) as webhooks_count,
			SUM(CASE WHEN f.environment_count > 0 THEN 1 ELSE 0 END) as environments_count,
			SUM(CASE WHEN f.secret_count > 0 THEN 1 ELSE 0 END) as secrets_count,
			SUM(CASE WHEN f.variable_count > 0 THEN 1 ELSE 0 END) as variables_count,
			
			-- Azure DevOps features (only count for ADO sources)
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.is_git = FALSE THEN 1 ELSE 0 END) as ado_tfvc_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.has_boards = TRUE THEN 1 ELSE 0 END) as ado_has_boards_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.has_pipelines = TRUE THEN 1 ELSE 0 END) as ado_has_pipelines_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.has_ghas = TRUE THEN 1 ELSE 0 END) as ado_has_ghas_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.pull_request_count > 0 THEN 1 ELSE 0 END) as ado_has_pull_requests_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.work_item_count > 0 THEN 1 ELSE 0 END) as ado_has_work_items_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.branch_policy_count > 0 THEN 1 ELSE 0 END) as ado_has_branch_policies_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.yaml_pipeline_count > 0 THEN 1 ELSE 0 END) as ado_has_yaml_pipelines_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.classic_pipeline_count > 0 THEN 1 ELSE 0 END) as ado_has_classic_pipelines_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.has_wiki = TRUE THEN 1 ELSE 0 END) as ado_has_wiki_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.test_plan_count > 0 THEN 1 ELSE 0 END) as ado_has_test_plans_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.package_feed_count > 0 THEN 1 ELSE 0 END) as ado_has_package_feeds_count,
			SUM(CASE WHEN r.source = 'azuredevops' AND ap.service_hook_count > 0 THEN 1 ELSE 0 END) as ado_has_service_hooks_count,
			
			COUNT(r.id) as total
		FROM repositories r
		LEFT JOIN repository_git_properties gp ON r.id = gp.repository_id
		LEFT JOIN repository_features f ON r.id = f.repository_id
		LEFT JOIN repository_ado_properties ap ON r.id = ap.repository_id
		WHERE 1=1
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + projectFilterSQL + `
			` + batchFilterSQL + `
			` + sourceFilterSQL + `
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	var stats FeatureStats
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get feature stats: %w", err)
	}

	return &stats, nil
}

// GetOrganizationStatsFiltered returns organization stats with batch filter
func (d *Database) GetOrganizationStatsFiltered(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) ([]*OrganizationStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	// Use dialect-specific string functions via DialectDialer interface
	extractOrg := d.dialect.ExtractOrgFromFullName("r.full_name")
	findSlash := d.dialect.FindCharPosition("r.full_name", "/")

	// Group by organization (extracted from full_name), source, and status
	// Join with sources table to get source name and type
	query := fmt.Sprintf(`
		SELECT 
			%s as org,
			r.source,
			r.source_id,
			s.name as source_name,
			s.type as source_type,
			COUNT(*) as total,
			r.status,
			COUNT(*) as status_count
		FROM repositories r
		LEFT JOIN sources s ON r.source_id = s.id
		WHERE %s > 0
			AND r.status != 'wont_migrate'
			%s
			%s
			%s
			%s
		GROUP BY org, r.source, r.source_id, s.name, s.type, r.status
		ORDER BY total DESC, org ASC
	`, extractOrg, findSlash, orgFilterSQL, projectFilterSQL, batchFilterSQL, sourceFilterSQL)

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	type OrgStatusResult struct {
		Org         string
		Source      string
		SourceID    *int64
		SourceName  *string
		SourceType  *string
		Total       int
		Status      string
		StatusCount int
	}

	var results []OrgStatusResult
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get organization stats: %w", err)
	}

	// Build organization stats map
	// Key by org + source_id to handle same org name across different sources
	// For GitHub: key is org name + source_id
	// For ADO: key is ADO organization name + source_id
	type orgSourceKey struct {
		Org      string
		SourceID int64
	}
	orgMap := make(map[orgSourceKey]*OrganizationStats)
	for _, result := range results {
		// Use org + source_id as key to distinguish same org across different sources
		sourceIDForKey := int64(0)
		if result.SourceID != nil {
			sourceIDForKey = *result.SourceID
		}
		mapKey := orgSourceKey{Org: result.Org, SourceID: sourceIDForKey}

		if _, exists := orgMap[mapKey]; !exists {
			orgMap[mapKey] = &OrganizationStats{
				Organization: result.Org,
				TotalRepos:   0,
				StatusCounts: make(map[string]int),
				SourceID:     result.SourceID,
				SourceName:   result.SourceName,
				SourceType:   result.SourceType,
			}
			// For ADO, store the organization name in ADOOrganization as well
			if result.Source == models.SourceTypeAzureDevOps {
				orgMap[mapKey].ADOOrganization = &result.Org
			}
		}
		orgStat := orgMap[mapKey]

		orgStat.StatusCounts[result.Status] += result.StatusCount
		orgStat.TotalRepos += result.StatusCount

		// Calculate progress metrics
		switch result.Status {
		case string(models.StatusComplete), string(models.StatusMigrationComplete):
			orgStat.MigratedCount += result.StatusCount
		case "migration_failed", "dry_run_failed", "rolled_back":
			orgStat.FailedCount += result.StatusCount
		case "queued_for_migration", "migrating_content", "dry_run_in_progress",
			"dry_run_queued", "pre_migration", "archive_generating", "post_migration":
			orgStat.InProgressCount += result.StatusCount
		default:
			// pending, dry_run_complete, remediation_required
			orgStat.PendingCount += result.StatusCount
		}
	}

	stats := make([]*OrganizationStats, 0, len(orgMap))
	for _, stat := range orgMap {
		// Calculate migration progress percentage
		if stat.TotalRepos > 0 {
			stat.MigrationProgressPercent = (stat.MigratedCount * 100) / stat.TotalRepos
		}
		stats = append(stats, stat)
	}

	// Sort by total repos (descending), then by organization name (ascending) for consistent ordering
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalRepos == stats[j].TotalRepos {
			return stats[i].Organization < stats[j].Organization
		}
		return stats[i].TotalRepos > stats[j].TotalRepos
	})

	return stats, nil
}

// GetProjectStatsFiltered returns repository counts grouped by ADO project (for Azure DevOps sources)
func (d *Database) GetProjectStatsFiltered(ctx context.Context, orgFilter, projectFilter, batchFilter string, sourceID *int64) ([]*OrganizationStats, error) {
	// Build filter clauses and collect arguments
	orgFilterSQL, orgArgs := d.buildOrgFilter(orgFilter)
	projectFilterSQL, projectArgs := d.buildProjectFilter(projectFilter)
	batchFilterSQL, batchArgs := d.buildBatchFilter(batchFilter)
	sourceFilterSQL, sourceArgs := d.buildSourceFilter(sourceID)

	query := `
		SELECT 
			ap.project as org,
			COUNT(*) as total,
			r.status as status,
			COUNT(*) as status_count
		FROM repositories r
		LEFT JOIN repository_ado_properties ap ON r.id = ap.repository_id
		WHERE ap.project IS NOT NULL
			AND ap.project != ''
			AND r.status != 'wont_migrate'
			` + orgFilterSQL + `
			` + projectFilterSQL + `
			` + batchFilterSQL + `
			` + sourceFilterSQL + `
		GROUP BY ap.project, r.status
		ORDER BY total DESC, ap.project ASC
	`

	// Combine all arguments
	args := append(orgArgs, projectArgs...)
	args = append(args, batchArgs...)
	args = append(args, sourceArgs...)

	// Use GORM Raw() for analytics query
	type ProjectCount struct {
		Org         string
		Total       int
		Status      string
		StatusCount int
	}

	var results []ProjectCount
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	// Group by project and aggregate status counts
	projectMap := make(map[string]*OrganizationStats)
	for _, result := range results {
		if _, exists := projectMap[result.Org]; !exists {
			projectMap[result.Org] = &OrganizationStats{
				Organization: result.Org,
				TotalRepos:   0,
				StatusCounts: make(map[string]int),
			}
		}
		projectMap[result.Org].StatusCounts[result.Status] = result.StatusCount
	}

	// Calculate total repos for each project
	for projectName, stats := range projectMap {
		total := 0
		for _, count := range stats.StatusCounts {
			total += count
		}
		stats.TotalRepos = total
		projectMap[projectName] = stats
	}

	// Convert map to array
	projectStatsArray := make([]*OrganizationStats, 0, len(projectMap))
	for _, stats := range projectMap {
		projectStatsArray = append(projectStatsArray, stats)
	}

	// Sort by total repos descending
	sort.Slice(projectStatsArray, func(i, j int) bool {
		return projectStatsArray[i].TotalRepos > projectStatsArray[j].TotalRepos
	})

	return projectStatsArray, nil
}

// DashboardActionItems contains all action items requiring admin attention
type DashboardActionItems struct {
	FailedMigrations    []*FailedRepository  `json:"failed_migrations"`
	FailedDryRuns       []*FailedRepository  `json:"failed_dry_runs"`
	ReadyBatches        []*models.Batch      `json:"ready_batches"`
	BlockedRepositories []*models.Repository `json:"blocked_repositories"`
}

// FailedRepository represents a repository that needs attention
type FailedRepository struct {
	ID           int64      `json:"id"`
	FullName     string     `json:"full_name"`
	Organization string     `json:"organization"`
	Status       string     `json:"status"`
	ErrorSummary *string    `json:"error_summary,omitempty"`
	FailedAt     *time.Time `json:"failed_at,omitempty"`
	BatchID      *int64     `json:"batch_id,omitempty"`
	BatchName    *string    `json:"batch_name,omitempty"`
}

// GetDashboardActionItems retrieves all action items requiring admin attention
func (d *Database) GetDashboardActionItems(ctx context.Context) (*DashboardActionItems, error) {
	actionItems := &DashboardActionItems{
		FailedMigrations:    make([]*FailedRepository, 0),
		FailedDryRuns:       make([]*FailedRepository, 0),
		ReadyBatches:        make([]*models.Batch, 0),
		BlockedRepositories: make([]*models.Repository, 0),
	}

	// Get failed migrations
	failedMigrationQuery := `
		SELECT 
			r.id,
			r.full_name,
			r.status,
			r.batch_id,
			b.name as batch_name,
			r.migrated_at as failed_at
		FROM repositories r
		LEFT JOIN batches b ON r.batch_id = b.id
		WHERE r.status = 'migration_failed'
		ORDER BY r.migrated_at DESC
		LIMIT 50
	`

	err := d.db.WithContext(ctx).Raw(failedMigrationQuery).Scan(&actionItems.FailedMigrations).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get failed migrations: %w", err)
	}

	// Extract organization from full_name for each failed migration
	for _, repo := range actionItems.FailedMigrations {
		slashIdx := -1
		for idx := 0; idx < len(repo.FullName); idx++ {
			if repo.FullName[idx] == '/' {
				slashIdx = idx
				break
			}
		}
		if slashIdx > 0 {
			repo.Organization = repo.FullName[:slashIdx]
		} else {
			// Defensive: use entire name if no slash found
			repo.Organization = repo.FullName
		}
	}

	// Get failed dry runs
	failedDryRunQuery := `
		SELECT 
			r.id,
			r.full_name,
			r.status,
			r.batch_id,
			b.name as batch_name,
			r.last_dry_run_at as failed_at
		FROM repositories r
		LEFT JOIN batches b ON r.batch_id = b.id
		WHERE r.status = 'dry_run_failed'
		ORDER BY r.last_dry_run_at DESC
		LIMIT 50
	`

	err = d.db.WithContext(ctx).Raw(failedDryRunQuery).Scan(&actionItems.FailedDryRuns).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get failed dry runs: %w", err)
	}

	// Extract organization from full_name for each failed dry run
	for _, repo := range actionItems.FailedDryRuns {
		slashIdx := -1
		for idx := 0; idx < len(repo.FullName); idx++ {
			if repo.FullName[idx] == '/' {
				slashIdx = idx
				break
			}
		}
		if slashIdx > 0 {
			repo.Organization = repo.FullName[:slashIdx]
		} else {
			// Defensive: use entire name if no slash found
			repo.Organization = repo.FullName
		}
	}

	// Get ready batches (status = ready OR status = pending/ready with scheduled time in the past)
	// Exclude completed, failed, or cancelled batches
	now := time.Now()
	err = d.db.WithContext(ctx).
		Where("status = ? OR (status IN (?, ?) AND scheduled_at IS NOT NULL AND scheduled_at <= ?)",
			"ready", "pending", "ready", now).
		Order("scheduled_at ASC NULLS LAST, created_at ASC").
		Limit(10).
		Find(&actionItems.ReadyBatches).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get ready batches: %w", err)
	}

	// Get blocked repositories (remediation_required or oversized)
	err = d.db.WithContext(ctx).
		Joins("LEFT JOIN repository_validation rv ON repositories.id = rv.repository_id").
		Where("repositories.status = ? OR rv.has_oversized_repository = ? OR rv.has_blocking_files = ?",
			"remediation_required", true, true).
		Order("repositories.discovered_at DESC").
		Limit(50).
		Preload("Validation").
		Find(&actionItems.BlockedRepositories).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked repositories: %w", err)
	}

	return actionItems, nil
}
