package discovery

import "github.com/kuhlman-labs/github-migrator/internal/models"

// CalculateComplexity calculates the complexity score for a GitHub repository
// This matches the logic from buildGitHubComplexityScoreSQL() but works in-memory
//
//nolint:gocyclo // Complexity is inherent to the calculation logic
func (p *Profiler) CalculateComplexity(repo *models.Repository) (int, *models.ComplexityBreakdown) {
	breakdown := &models.ComplexityBreakdown{}
	complexity := 0

	// Size tier scoring (0-9 points)
	sizePoints := calculateSizePoints(repo.GetTotalSize())
	breakdown.SizePoints = sizePoints
	complexity += sizePoints

	// Large files (blocking for GitHub migrations) - 4 points
	if repo.HasLargeFiles() {
		breakdown.LargeFilesPoints = 4
		complexity += 4
	}

	// High impact features (3 points each)
	if repo.GetEnvironmentCount() > 0 {
		breakdown.EnvironmentsPoints = 3
		complexity += 3
	}
	if repo.GetSecretCount() > 0 {
		breakdown.SecretsPoints = 3
		complexity += 3
	}
	if repo.HasPackages() {
		breakdown.PackagesPoints = 3
		complexity += 3
	}
	if repo.HasSelfHostedRunners() {
		breakdown.RunnersPoints = 3
		complexity += 3
	}

	// Moderate impact features (2 points each)
	if repo.GetVariableCount() > 0 {
		breakdown.VariablesPoints = 2
		complexity += 2
	}
	if repo.HasDiscussions() {
		breakdown.DiscussionsPoints = 2
		complexity += 2
	}
	if repo.GetReleaseCount() > 0 {
		breakdown.ReleasesPoints = 2
		complexity += 2
	}
	if repo.HasLFS() {
		breakdown.LFSPoints = 2
		complexity += 2
	}
	if repo.HasSubmodules() {
		breakdown.SubmodulesPoints = 2
		complexity += 2
	}
	if repo.GetInstalledAppsCount() > 0 {
		breakdown.AppsPoints = 2
		complexity += 2
	}
	if repo.HasProjects() {
		breakdown.ProjectsPoints = 2
		complexity += 2
	}

	// Low impact features (1 point each)
	if repo.HasCodeScanning() || repo.HasDependabot() || repo.HasSecretScanning() {
		breakdown.SecurityPoints = 1
		complexity += 1
	}
	if repo.GetWebhookCount() > 0 {
		breakdown.WebhooksPoints = 1
		complexity += 1
	}
	if repo.GetBranchProtections() > 0 {
		breakdown.BranchProtectionsPoints = 1
		complexity += 1
	}
	if repo.HasRulesets() {
		breakdown.RulesetsPoints = 1
		complexity += 1
	}
	if repo.Visibility == "public" {
		breakdown.PublicVisibilityPoints = 1
		complexity += 1
	}
	if repo.Visibility == "internal" {
		breakdown.InternalVisibilityPoints = 1
		complexity += 1
	}
	if repo.HasCodeowners() {
		breakdown.CodeownersPoints = 1
		complexity += 1
	}

	// Activity-based scoring (0, 2, or 4 points)
	// Note: This is simplified since we don't have access to quantiles here
	// The SQL version uses database-wide quantiles for more accurate scoring
	activityPoints := calculateActivityPoints(repo)
	breakdown.ActivityPoints = activityPoints
	complexity += activityPoints

	return complexity, breakdown
}

// calculateSizePoints returns points based on repository size
func calculateSizePoints(totalSize *int64) int {
	const (
		MB100 = 104857600  // 100MB
		GB1   = 1073741824 // 1GB
		GB5   = 5368709120 // 5GB
	)

	if totalSize == nil || *totalSize < MB100 {
		return 0
	}
	if *totalSize < GB1 {
		return 3 // 1 * 3
	}
	if *totalSize < GB5 {
		return 6 // 2 * 3
	}
	return 9 // 3 * 3
}

// calculateActivityPoints returns points based on repository activity
// Uses simplified thresholds since we don't have access to database quantiles
func calculateActivityPoints(repo *models.Repository) int {
	// Estimate activity from commit count and open issue count
	activity := repo.GetCommitCount() + (repo.GetOpenIssueCount() * 2) // Issues/PRs count double

	// Use static thresholds (less accurate than SQL quantiles, but reasonable)
	if activity > 1000 {
		return 4 // Top quantile
	}
	if activity > 100 {
		return 2 // Middle quantile
	}
	return 0 // Low activity
}
