package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

func (h *Handler) exportExecutiveReportCSV(w http.ResponseWriter, sourceType string, total, migrated, inProgress, pending, failed int,
	completionRate, successRate float64, estimatedCompletionDate string, daysRemaining int,
	velocity *storage.MigrationVelocity, avgMigrationTime, medianMigrationTime int,
	orgStats []*storage.MigrationCompletionStats, complexityDist []*storage.ComplexityDistribution,
	sizeDist []*storage.SizeDistribution, featureStats *storage.FeatureStats,
	statusBreakdown map[string]int, completedBatches, inProgressBatches, pendingBatches int) {

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=executive_migration_report.csv")

	// Ensure featureStats is not nil to prevent panics
	if featureStats == nil {
		featureStats = &storage.FeatureStats{}
	}

	var output strings.Builder

	output.WriteString("EXECUTIVE MIGRATION REPORT\n")
	output.WriteString(fmt.Sprintf("Source Platform: %s\n", strings.ToUpper(sourceType)))
	output.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	output.WriteString("\n")

	output.WriteString("================================================================================\n")
	output.WriteString("SECTION 1: DISCOVERY DATA\n")
	output.WriteString("================================================================================\n\n")

	output.WriteString("--- DISCOVERY OVERVIEW ---\n")
	output.WriteString("Metric,Value\n")
	output.WriteString(fmt.Sprintf("Total Repositories Discovered,%d\n", total))
	output.WriteString(fmt.Sprintf("Source Platform,%s\n", strings.ToUpper(sourceType)))
	output.WriteString("\n")

	output.WriteString("--- REPOSITORY COMPLEXITY ---\n")
	output.WriteString("Complexity Category,Repository Count,Percentage\n")
	for _, dist := range complexityDist {
		pct := 0.0
		if total > 0 {
			pct = float64(dist.Count) / float64(total) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%.1f%%\n", escapeCSV(dist.Category), dist.Count, pct))
	}
	output.WriteString("\n")

	output.WriteString("--- REPOSITORY SIZE DISTRIBUTION ---\n")
	output.WriteString("Size Category,Repository Count,Percentage\n")
	for _, dist := range sizeDist {
		pct := 0.0
		if total > 0 {
			pct = float64(dist.Count) / float64(total) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%.1f%%\n", escapeCSV(dist.Category), dist.Count, pct))
	}
	output.WriteString("\n")

	output.WriteString("--- FEATURE DISCOVERY ---\n")
	output.WriteString("Feature,Repository Count,Percentage\n")
	totalRepos := featureStats.TotalRepositories
	if totalRepos > 0 {
		calcPct := func(count int) float64 {
			return float64(count) / float64(totalRepos) * 100
		}

		if sourceType == models.SourceTypeAzureDevOps {
			// Azure DevOps specific features
			output.WriteString(fmt.Sprintf("TFVC Repositories,%d,%.1f%%\n", featureStats.ADOTFVCCount, calcPct(featureStats.ADOTFVCCount)))
			output.WriteString(fmt.Sprintf("Azure Pipelines,%d,%.1f%%\n", featureStats.ADOHasPipelines, calcPct(featureStats.ADOHasPipelines)))
			output.WriteString(fmt.Sprintf("YAML Pipelines,%d,%.1f%%\n", featureStats.ADOHasYAMLPipelines, calcPct(featureStats.ADOHasYAMLPipelines)))
			output.WriteString(fmt.Sprintf("Classic Pipelines,%d,%.1f%%\n", featureStats.ADOHasClassicPipelines, calcPct(featureStats.ADOHasClassicPipelines)))
			output.WriteString(fmt.Sprintf("Azure Boards,%d,%.1f%%\n", featureStats.ADOHasBoards, calcPct(featureStats.ADOHasBoards)))
			output.WriteString(fmt.Sprintf("Work Items,%d,%.1f%%\n", featureStats.ADOHasWorkItems, calcPct(featureStats.ADOHasWorkItems)))
			output.WriteString(fmt.Sprintf("Wiki,%d,%.1f%%\n", featureStats.ADOHasWiki, calcPct(featureStats.ADOHasWiki)))
			output.WriteString(fmt.Sprintf("Branch Policies,%d,%.1f%%\n", featureStats.ADOHasBranchPolicies, calcPct(featureStats.ADOHasBranchPolicies)))
			output.WriteString(fmt.Sprintf("Pull Requests,%d,%.1f%%\n", featureStats.ADOHasPullRequests, calcPct(featureStats.ADOHasPullRequests)))
			output.WriteString(fmt.Sprintf("Test Plans,%d,%.1f%%\n", featureStats.ADOHasTestPlans, calcPct(featureStats.ADOHasTestPlans)))
			output.WriteString(fmt.Sprintf("Package Feeds,%d,%.1f%%\n", featureStats.ADOHasPackageFeeds, calcPct(featureStats.ADOHasPackageFeeds)))
			output.WriteString(fmt.Sprintf("Service Hooks,%d,%.1f%%\n", featureStats.ADOHasServiceHooks, calcPct(featureStats.ADOHasServiceHooks)))
			output.WriteString(fmt.Sprintf("GitHub Advanced Security,%d,%.1f%%\n", featureStats.ADOHasGHAS, calcPct(featureStats.ADOHasGHAS)))
		} else {
			// GitHub specific features
			output.WriteString(fmt.Sprintf("GitHub Actions,%d,%.1f%%\n", featureStats.HasActions, calcPct(featureStats.HasActions)))
			output.WriteString(fmt.Sprintf("Wikis,%d,%.1f%%\n", featureStats.HasWiki, calcPct(featureStats.HasWiki)))
			output.WriteString(fmt.Sprintf("Pages,%d,%.1f%%\n", featureStats.HasPages, calcPct(featureStats.HasPages)))
			output.WriteString(fmt.Sprintf("Discussions,%d,%.1f%%\n", featureStats.HasDiscussions, calcPct(featureStats.HasDiscussions)))
			output.WriteString(fmt.Sprintf("Projects,%d,%.1f%%\n", featureStats.HasProjects, calcPct(featureStats.HasProjects)))
			output.WriteString(fmt.Sprintf("Packages,%d,%.1f%%\n", featureStats.HasPackages, calcPct(featureStats.HasPackages)))
			output.WriteString(fmt.Sprintf("Environments,%d,%.1f%%\n", featureStats.HasEnvironments, calcPct(featureStats.HasEnvironments)))
			output.WriteString(fmt.Sprintf("Secrets,%d,%.1f%%\n", featureStats.HasSecrets, calcPct(featureStats.HasSecrets)))
			output.WriteString(fmt.Sprintf("Variables,%d,%.1f%%\n", featureStats.HasVariables, calcPct(featureStats.HasVariables)))
			output.WriteString(fmt.Sprintf("Branch Protections,%d,%.1f%%\n", featureStats.HasBranchProtections, calcPct(featureStats.HasBranchProtections)))
			output.WriteString(fmt.Sprintf("Rulesets,%d,%.1f%%\n", featureStats.HasRulesets, calcPct(featureStats.HasRulesets)))
			output.WriteString(fmt.Sprintf("Code Scanning,%d,%.1f%%\n", featureStats.HasCodeScanning, calcPct(featureStats.HasCodeScanning)))
			output.WriteString(fmt.Sprintf("Dependabot,%d,%.1f%%\n", featureStats.HasDependabot, calcPct(featureStats.HasDependabot)))
			output.WriteString(fmt.Sprintf("Secret Scanning,%d,%.1f%%\n", featureStats.HasSecretScanning, calcPct(featureStats.HasSecretScanning)))
			output.WriteString(fmt.Sprintf("CODEOWNERS,%d,%.1f%%\n", featureStats.HasCodeowners, calcPct(featureStats.HasCodeowners)))
			output.WriteString(fmt.Sprintf("Self-Hosted Runners,%d,%.1f%%\n", featureStats.HasSelfHostedRunners, calcPct(featureStats.HasSelfHostedRunners)))
			output.WriteString(fmt.Sprintf("Release Assets,%d,%.1f%%\n", featureStats.HasReleaseAssets, calcPct(featureStats.HasReleaseAssets)))
			output.WriteString(fmt.Sprintf("Webhooks,%d,%.1f%%\n", featureStats.HasWebhooks, calcPct(featureStats.HasWebhooks)))
		}
		// Common features for both platforms
		output.WriteString(fmt.Sprintf("LFS,%d,%.1f%%\n", featureStats.HasLFS, calcPct(featureStats.HasLFS)))
		output.WriteString(fmt.Sprintf("Submodules,%d,%.1f%%\n", featureStats.HasSubmodules, calcPct(featureStats.HasSubmodules)))
		output.WriteString(fmt.Sprintf("Large Files,%d,%.1f%%\n", featureStats.HasLargeFiles, calcPct(featureStats.HasLargeFiles)))
		output.WriteString(fmt.Sprintf("Archived,%d,%.1f%%\n", featureStats.IsArchived, calcPct(featureStats.IsArchived)))
		output.WriteString(fmt.Sprintf("Forks,%d,%.1f%%\n", featureStats.IsFork, calcPct(featureStats.IsFork)))
	}
	output.WriteString("\n")

	output.WriteString("================================================================================\n")
	output.WriteString("SECTION 2: MIGRATION PROGRESS & ANALYTICS\n")
	output.WriteString("================================================================================\n\n")

	output.WriteString("--- MIGRATION SUMMARY ---\n")
	output.WriteString("Metric,Value\n")
	output.WriteString(fmt.Sprintf("Total Repositories,%d\n", total))
	output.WriteString(fmt.Sprintf("Completion Percentage,%.1f%%\n", completionRate))
	output.WriteString(fmt.Sprintf("Successfully Migrated,%d\n", migrated))
	output.WriteString(fmt.Sprintf("In Progress,%d\n", inProgress))
	output.WriteString(fmt.Sprintf("Pending,%d\n", pending))
	output.WriteString(fmt.Sprintf("Failed,%d\n", failed))
	output.WriteString(fmt.Sprintf("Success Rate,%.1f%%\n", successRate))
	if estimatedCompletionDate != "" {
		output.WriteString(fmt.Sprintf("Estimated Completion,%s\n", estimatedCompletionDate))
		output.WriteString(fmt.Sprintf("Days Remaining,%d\n", daysRemaining))
	}
	output.WriteString("\n")

	output.WriteString("--- MIGRATION VELOCITY ---\n")
	output.WriteString("Metric,Value\n")
	output.WriteString(fmt.Sprintf("Repos Per Day,%.1f\n", velocity.ReposPerDay))
	output.WriteString(fmt.Sprintf("Repos Per Week,%.1f\n", velocity.ReposPerWeek))
	if avgMigrationTime > 0 {
		output.WriteString(fmt.Sprintf("Average Migration Time,%d minutes\n", avgMigrationTime/60))
	}
	if medianMigrationTime > 0 {
		output.WriteString(fmt.Sprintf("Median Migration Time,%d minutes\n", medianMigrationTime/60))
	}
	output.WriteString("\n")

	output.WriteString("--- BATCH EXECUTION PERFORMANCE ---\n")
	output.WriteString("Status,Count\n")
	output.WriteString(fmt.Sprintf("Completed,%d\n", completedBatches))
	output.WriteString(fmt.Sprintf("In Progress,%d\n", inProgressBatches))
	output.WriteString(fmt.Sprintf("Pending,%d\n", pendingBatches))
	output.WriteString(fmt.Sprintf("Total Batches,%d\n", completedBatches+inProgressBatches+pendingBatches))
	output.WriteString("\n")

	output.WriteString("--- DETAILED STATUS BREAKDOWN ---\n")
	output.WriteString("Status,Repository Count,Percentage\n")
	for status, count := range statusBreakdown {
		// Skip wont_migrate since total excludes it (would produce incorrect percentage)
		if status == string(models.StatusWontMigrate) {
			continue
		}
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%.1f%%\n", escapeCSV(status), count, pct))
	}
	// Add wont_migrate separately without percentage since it's excluded from migration scope
	if wontMigrateCount, ok := statusBreakdown[string(models.StatusWontMigrate)]; ok && wontMigrateCount > 0 {
		output.WriteString(fmt.Sprintf("%s,%d,N/A (excluded from migration)\n", escapeCSV(string(models.StatusWontMigrate)), wontMigrateCount))
	}

	if _, err := w.Write([]byte(output.String())); err != nil {
		h.logger.Error("Failed to write CSV response", "error", err)
	}
}

func (h *Handler) exportExecutiveReportJSON(w http.ResponseWriter, sourceType string, total, migrated, inProgress, pending, failed int,
	completionRate, successRate float64, estimatedCompletionDate string, daysRemaining int,
	velocity *storage.MigrationVelocity, avgMigrationTime, medianMigrationTime int,
	orgStats []*storage.MigrationCompletionStats, complexityDist []*storage.ComplexityDistribution,
	sizeDist []*storage.SizeDistribution, featureStats *storage.FeatureStats,
	statusBreakdown map[string]int, completedBatches, inProgressBatches, pendingBatches int) {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=executive_migration_report.json")

	// Ensure featureStats is not nil to prevent panics
	if featureStats == nil {
		featureStats = &storage.FeatureStats{}
	}

	highComplexity := 0
	for _, dist := range complexityDist {
		if dist.Category == models.ComplexityComplex || dist.Category == models.ComplexityVeryComplex {
			highComplexity += dist.Count
		}
	}
	veryLarge := 0
	for _, dist := range sizeDist {
		if dist.Category == models.SizeCategoryVeryLarge {
			veryLarge += dist.Count
		}
	}

	report := map[string]any{
		"source_type": sourceType,
		"report_metadata": map[string]any{
			"generated_at": time.Now().Format(time.RFC3339),
			"report_type":  "Executive Migration Report",
			"version":      "2.0",
		},
		"discovery_data": map[string]any{
			"overview":                 map[string]any{"total_repositories": total, "source_type": sourceType},
			"features":                 featureStats,
			"complexity_distribution":  complexityDist,
			"size_distribution":        sizeDist,
			"organizational_breakdown": orgStats,
		},
		"migration_analytics": map[string]any{
			"summary": map[string]any{
				"total_repositories":        total,
				"migrated_count":            migrated,
				"in_progress_count":         inProgress,
				"pending_count":             pending,
				"failed_count":              failed,
				"completion_percentage":     completionRate,
				"success_rate":              successRate,
				"estimated_completion_date": estimatedCompletionDate,
				"days_remaining":            daysRemaining,
			},
			"status_breakdown": statusBreakdown,
			"velocity": map[string]any{
				"repos_per_day":        velocity.ReposPerDay,
				"repos_per_week":       velocity.ReposPerWeek,
				"average_duration_sec": avgMigrationTime,
				"median_duration_sec":  medianMigrationTime,
			},
			"batches": map[string]any{
				"total":       completedBatches + inProgressBatches + pendingBatches,
				"completed":   completedBatches,
				"in_progress": inProgressBatches,
				"pending":     pendingBatches,
			},
			"risk_factors": map[string]any{
				"high_complexity_pending": highComplexity,
				"very_large_pending":      veryLarge,
				"failed_migrations":       failed,
			},
			"organization_progress": orgStats,
		},
	}

	if sourceType == models.SourceTypeAzureDevOps {
		if discoveryData, ok := report["discovery_data"].(map[string]any); ok {
			discoveryData["ado_specific_risks"] = map[string]any{
				"tfvc_repos":                   featureStats.ADOTFVCCount,
				"classic_pipelines":            featureStats.ADOHasClassicPipelines,
				"repos_with_active_work_items": featureStats.ADOHasWorkItems,
				"repos_with_wikis":             featureStats.ADOHasWiki,
				"repos_with_test_plans":        featureStats.ADOHasTestPlans,
				"repos_with_package_feeds":     featureStats.ADOHasPackageFeeds,
			}
		}
	}

	if err := json.NewEncoder(w).Encode(report); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *Handler) exportDetailedDiscoveryReportJSON(w http.ResponseWriter, repos []*models.Repository, localDepsCount map[int64]int, batchNames map[int64]string, orgFilter, projectFilter, batchFilter string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=detailed_discovery_report.json")

	filtersApplied := make(map[string]string)
	if orgFilter != "" {
		filtersApplied["organization"] = orgFilter
	}
	if projectFilter != "" {
		filtersApplied["project"] = projectFilter
	}
	if batchFilter != "" {
		filtersApplied["batch_id"] = batchFilter
	}

	repoData := make([]map[string]any, 0, len(repos))
	for _, repo := range repos {
		repoJSON, err := json.Marshal(repo)
		if err != nil {
			continue
		}

		var repoMap map[string]any
		if err := json.Unmarshal(repoJSON, &repoMap); err != nil {
			continue
		}

		if count, exists := localDepsCount[repo.ID]; exists {
			repoMap["local_dependencies_count"] = count
		} else {
			repoMap["local_dependencies_count"] = 0
		}
		repoMap["organization"] = repo.Organization()
		repoData = append(repoData, repoMap)
	}

	report := map[string]any{
		"report_metadata": map[string]any{
			"generated_at":       time.Now().Format(time.RFC3339),
			"report_type":        "Detailed Repository Discovery Report",
			"source_type":        h.sourceType,
			"version":            "1.0",
			"filters_applied":    filtersApplied,
			"total_repositories": len(repos),
		},
		"repositories": repoData,
	}

	if err := json.NewEncoder(w).Encode(report); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *Handler) exportDetailedDiscoveryReportCSV(w http.ResponseWriter, repos []*models.Repository, localDepsCount map[int64]int, batchNames map[int64]string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=detailed_discovery_report.csv")

	var output strings.Builder

	h.writeCSVReportHeader(&output, len(repos))
	h.writeCSVColumnHeaders(&output)

	for _, repo := range repos {
		h.writeCSVRepoRow(&output, repo, localDepsCount, batchNames)
	}

	if _, err := w.Write([]byte(output.String())); err != nil {
		h.logger.Error("Failed to write CSV response", "error", err)
	}
}

func (h *Handler) writeCSVReportHeader(output *strings.Builder, repoCount int) {
	sourceDisplay := formatSourceForDisplay(h.sourceType)
	output.WriteString("DETAILED REPOSITORY DISCOVERY REPORT\n")
	output.WriteString(fmt.Sprintf("Source: %s\n", sourceDisplay))
	output.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("Total Repositories: %d\n", repoCount))
	output.WriteString("\n")
}

func (h *Handler) writeCSVColumnHeaders(output *strings.Builder) {
	if h.sourceType == models.SourceTypeAzureDevOps {
		output.WriteString("Repository,Organization,Project,Source,Status,Batch,")
	} else {
		output.WriteString("Repository,Organization,Source,Status,Batch,")
	}
	output.WriteString("Size (Bytes),Size (Human),Commit Count,Commits (Last 12 Weeks),")
	output.WriteString("Has LFS,Has Submodules,Has Large Files,Large File Count,Largest File Size (Bytes),")
	output.WriteString("Has Blocking Files,Local Dependencies,Complexity Score,")
	output.WriteString("Default Branch,Branch Count,Last Commit Date,Visibility,Is Archived,Is Fork,")

	if h.sourceType == models.SourceTypeAzureDevOps {
		output.WriteString("Is Git,Pipeline Count,YAML Pipelines,Classic Pipelines,Has Boards,Has Wiki,")
		output.WriteString("Pull Requests,Work Items,Branch Policies,Test Plans,Package Feeds,Service Hooks")
	} else {
		output.WriteString("Workflow Count,Environment Count,Secret Count,Has Actions,Has Environments,Has Packages,")
		output.WriteString("Has Projects,Branch Protections,Has Rulesets,Contributor Count,")
		output.WriteString("Issue Count,Pull Request Count,Has Self-Hosted Runners")
	}
	output.WriteString("\n")
}

func (h *Handler) writeCSVRepoRow(output *strings.Builder, repo *models.Repository, localDepsCount map[int64]int, batchNames map[int64]string) {
	output.WriteString(escapeCSV(repo.FullName))
	output.WriteString(",")
	output.WriteString(escapeCSV(repo.Organization()))
	output.WriteString(",")

	if h.sourceType == models.SourceTypeAzureDevOps {
		adoProject := repo.GetADOProject()
		if adoProject != nil {
			output.WriteString(escapeCSV(*adoProject))
		}
		output.WriteString(",")
	}

	output.WriteString(escapeCSV(formatSourceForDisplay(repo.Source)))
	output.WriteString(",")
	output.WriteString(escapeCSV(formatStatusForDisplay(repo.Status)))
	output.WriteString(",")

	if repo.BatchID != nil {
		if batchName, exists := batchNames[*repo.BatchID]; exists {
			output.WriteString(escapeCSV(batchName))
		} else {
			output.WriteString(fmt.Sprintf("Batch %d", *repo.BatchID))
		}
	}
	output.WriteString(",")

	totalSize := repo.GetTotalSize()
	if totalSize != nil {
		output.WriteString(fmt.Sprintf("%d,%s,", *totalSize, escapeCSV(formatBytes(*totalSize))))
	} else {
		output.WriteString("0,0 B,")
	}

	output.WriteString(fmt.Sprintf("%d,%d,", repo.GetCommitCount(), repo.GetCommitsLast12Weeks()))
	output.WriteString(fmt.Sprintf("%s,%s,%s,%d,", formatBool(repo.HasLFS()), formatBool(repo.HasSubmodules()), formatBool(repo.HasLargeFiles()), repo.GetLargeFileCount()))

	largestFileSize := repo.GetLargestFileSize()
	if largestFileSize != nil {
		output.WriteString(fmt.Sprintf("%d,", *largestFileSize))
	} else {
		output.WriteString("0,")
	}

	output.WriteString(formatBool(repo.HasBlockingFiles()))
	output.WriteString(",")

	if count, exists := localDepsCount[repo.ID]; exists {
		output.WriteString(fmt.Sprintf("%d,", count))
	} else {
		output.WriteString("0,")
	}

	complexityScore := repo.GetComplexityScore()
	if complexityScore != nil {
		output.WriteString(fmt.Sprintf("%d,", *complexityScore))
	} else {
		output.WriteString(",")
	}

	defaultBranch := repo.GetDefaultBranch()
	if defaultBranch != nil {
		output.WriteString(escapeCSV(*defaultBranch))
	}
	output.WriteString(",")
	output.WriteString(fmt.Sprintf("%d,", repo.GetBranchCount()))

	lastCommitDate := repo.GetLastCommitDate()
	if lastCommitDate != nil {
		output.WriteString(lastCommitDate.Format("2006-01-02"))
	}
	output.WriteString(",")

	output.WriteString(fmt.Sprintf("%s,%s,%s,", escapeCSV(formatVisibilityForDisplay(repo.Visibility)), formatBool(repo.IsArchived), formatBool(repo.IsFork)))

	h.writeCSVSourceSpecificFields(output, repo)
	output.WriteString("\n")
}

func (h *Handler) writeCSVSourceSpecificFields(output *strings.Builder, repo *models.Repository) {
	if h.sourceType == models.SourceTypeAzureDevOps {
		output.WriteString(fmt.Sprintf("%s,%d,%d,%d,%s,%s,",
			formatBool(repo.GetADOIsGit()),
			repo.GetADOPipelineCount(),
			repo.GetADOYAMLPipelineCount(),
			repo.GetADOClassicPipelineCount(),
			formatBool(repo.GetADOHasBoards()),
			formatBool(repo.GetADOHasWiki())))
		output.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d",
			repo.GetADOPullRequestCount(),
			repo.GetADOWorkItemCount(),
			repo.GetADOBranchPolicyCount(),
			repo.GetADOTestPlanCount(),
			repo.GetADOPackageFeedCount(),
			repo.GetADOServiceHookCount()))
	} else {
		output.WriteString(fmt.Sprintf("%d,%d,%d,%s,%s,%s,%s,%d,%s,",
			repo.GetWorkflowCount(),
			repo.GetEnvironmentCount(),
			repo.GetSecretCount(),
			formatBool(repo.HasActions()),
			formatBool(repo.GetEnvironmentCount() > 0),
			formatBool(repo.HasPackages()),
			formatBool(repo.HasProjects()),
			repo.GetBranchProtections(),
			formatBool(repo.HasRulesets())))
		output.WriteString(fmt.Sprintf("%d,%d,%d,%s",
			repo.GetContributorCount(),
			repo.GetIssueCount(),
			repo.GetPullRequestCount(),
			formatBool(repo.HasSelfHostedRunners())))
	}
}
