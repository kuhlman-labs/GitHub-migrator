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
		if sourceType == models.SourceTypeAzureDevOps {
			output.WriteString(fmt.Sprintf("TFVC Repositories,%d,%.1f%%\n", featureStats.ADOTFVCCount, float64(featureStats.ADOTFVCCount)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Azure Boards,%d,%.1f%%\n", featureStats.ADOHasBoards, float64(featureStats.ADOHasBoards)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Azure Pipelines,%d,%.1f%%\n", featureStats.ADOHasPipelines, float64(featureStats.ADOHasPipelines)/float64(totalRepos)*100))
		} else {
			output.WriteString(fmt.Sprintf("GitHub Actions,%d,%.1f%%\n", featureStats.HasActions, float64(featureStats.HasActions)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Wikis,%d,%.1f%%\n", featureStats.HasWiki, float64(featureStats.HasWiki)/float64(totalRepos)*100))
			output.WriteString(fmt.Sprintf("Pages,%d,%.1f%%\n", featureStats.HasPages, float64(featureStats.HasPages)/float64(totalRepos)*100))
		}
		output.WriteString(fmt.Sprintf("LFS,%d,%.1f%%\n", featureStats.HasLFS, float64(featureStats.HasLFS)/float64(totalRepos)*100))
		output.WriteString(fmt.Sprintf("Submodules,%d,%.1f%%\n", featureStats.HasSubmodules, float64(featureStats.HasSubmodules)/float64(totalRepos)*100))
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
		pct := 0.0
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		output.WriteString(fmt.Sprintf("%s,%d,%.1f%%\n", escapeCSV(status), count, pct))
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

	report := map[string]interface{}{
		"source_type": sourceType,
		"report_metadata": map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"report_type":  "Executive Migration Report",
			"version":      "2.0",
		},
		"discovery_data": map[string]interface{}{
			"overview":                 map[string]interface{}{"total_repositories": total, "source_type": sourceType},
			"features":                 featureStats,
			"complexity_distribution":  complexityDist,
			"size_distribution":        sizeDist,
			"organizational_breakdown": orgStats,
		},
		"migration_analytics": map[string]interface{}{
			"summary": map[string]interface{}{
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
			"velocity": map[string]interface{}{
				"repos_per_day":        velocity.ReposPerDay,
				"repos_per_week":       velocity.ReposPerWeek,
				"average_duration_sec": avgMigrationTime,
				"median_duration_sec":  medianMigrationTime,
			},
			"batches": map[string]interface{}{
				"total":       completedBatches + inProgressBatches + pendingBatches,
				"completed":   completedBatches,
				"in_progress": inProgressBatches,
				"pending":     pendingBatches,
			},
			"risk_factors": map[string]interface{}{
				"high_complexity_pending": highComplexity,
				"very_large_pending":      veryLarge,
				"failed_migrations":       failed,
			},
			"organization_progress": orgStats,
		},
	}

	if sourceType == models.SourceTypeAzureDevOps {
		if discoveryData, ok := report["discovery_data"].(map[string]interface{}); ok {
			discoveryData["ado_specific_risks"] = map[string]interface{}{
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

	repoData := make([]map[string]interface{}, 0, len(repos))
	for _, repo := range repos {
		repoJSON, err := json.Marshal(repo)
		if err != nil {
			continue
		}

		var repoMap map[string]interface{}
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

	report := map[string]interface{}{
		"report_metadata": map[string]interface{}{
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
		if repo.ADOProject != nil {
			output.WriteString(escapeCSV(*repo.ADOProject))
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

	if repo.TotalSize != nil {
		output.WriteString(fmt.Sprintf("%d,%s,", *repo.TotalSize, escapeCSV(formatBytes(*repo.TotalSize))))
	} else {
		output.WriteString("0,0 B,")
	}

	output.WriteString(fmt.Sprintf("%d,%d,", repo.CommitCount, repo.CommitsLast12Weeks))
	output.WriteString(fmt.Sprintf("%s,%s,%s,%d,", formatBool(repo.HasLFS), formatBool(repo.HasSubmodules), formatBool(repo.HasLargeFiles), repo.LargeFileCount))

	if repo.LargestFileSize != nil {
		output.WriteString(fmt.Sprintf("%d,", *repo.LargestFileSize))
	} else {
		output.WriteString("0,")
	}

	output.WriteString(formatBool(repo.HasBlockingFiles))
	output.WriteString(",")

	if count, exists := localDepsCount[repo.ID]; exists {
		output.WriteString(fmt.Sprintf("%d,", count))
	} else {
		output.WriteString("0,")
	}

	if repo.ComplexityScore != nil {
		output.WriteString(fmt.Sprintf("%d,", *repo.ComplexityScore))
	} else {
		output.WriteString(",")
	}

	if repo.DefaultBranch != nil {
		output.WriteString(escapeCSV(*repo.DefaultBranch))
	}
	output.WriteString(",")
	output.WriteString(fmt.Sprintf("%d,", repo.BranchCount))

	if repo.LastCommitDate != nil {
		output.WriteString(repo.LastCommitDate.Format("2006-01-02"))
	}
	output.WriteString(",")

	output.WriteString(fmt.Sprintf("%s,%s,%s,", escapeCSV(formatVisibilityForDisplay(repo.Visibility)), formatBool(repo.IsArchived), formatBool(repo.IsFork)))

	h.writeCSVSourceSpecificFields(output, repo)
	output.WriteString("\n")
}

func (h *Handler) writeCSVSourceSpecificFields(output *strings.Builder, repo *models.Repository) {
	if h.sourceType == models.SourceTypeAzureDevOps {
		output.WriteString(fmt.Sprintf("%s,%d,%d,%d,%s,%s,",
			formatBool(repo.ADOIsGit),
			repo.ADOPipelineCount,
			repo.ADOYAMLPipelineCount,
			repo.ADOClassicPipelineCount,
			formatBool(repo.ADOHasBoards),
			formatBool(repo.ADOHasWiki)))
		output.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d",
			repo.ADOPullRequestCount,
			repo.ADOWorkItemCount,
			repo.ADOBranchPolicyCount,
			repo.ADOTestPlanCount,
			repo.ADOPackageFeedCount,
			repo.ADOServiceHookCount))
	} else {
		output.WriteString(fmt.Sprintf("%d,%d,%d,%s,%s,%s,%s,%d,%s,",
			repo.WorkflowCount,
			repo.EnvironmentCount,
			repo.SecretCount,
			formatBool(repo.HasActions),
			formatBool(repo.EnvironmentCount > 0),
			formatBool(repo.HasPackages),
			formatBool(repo.HasProjects),
			repo.BranchProtections,
			formatBool(repo.HasRulesets)))
		output.WriteString(fmt.Sprintf("%d,%d,%d,%s",
			repo.ContributorCount,
			repo.IssueCount,
			repo.PullRequestCount,
			formatBool(repo.HasSelfHostedRunners)))
	}
}
