package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

// ADOProfiler profiles Azure DevOps repositories
type ADOProfiler struct {
	client         *azuredevops.Client
	logger         *slog.Logger
	sourceProvider interface{} // Will be source.Provider
}

// NewADOProfiler creates a new ADO profiler
func NewADOProfiler(client *azuredevops.Client, logger *slog.Logger, sourceProvider interface{}) *ADOProfiler {
	return &ADOProfiler{
		client:         client,
		logger:         logger,
		sourceProvider: sourceProvider,
	}
}

// ProfileRepository profiles an Azure DevOps repository
// This includes both Git analysis (if it's a Git repo) and ADO-specific features
func (p *ADOProfiler) ProfileRepository(ctx context.Context, repo *models.Repository, adoRepo interface{}) error {
	if repo.ADOProject == nil {
		return fmt.Errorf("repository missing ADO project name")
	}

	projectName := *repo.ADOProject

	// Extract repo ID from adoRepo - type assert to git.GitRepository
	repoID := ""
	if gitRepo, ok := adoRepo.(git.GitRepository); ok {
		if gitRepo.Id != nil {
			repoID = gitRepo.Id.String()
		}
	} else {
		p.logger.Warn("Failed to type assert adoRepo to git.GitRepository",
			"repo", repo.FullName)
	}

	if repoID == "" {
		p.logger.Warn("Repository ID not available, profiling may be incomplete",
			"repo", repo.FullName)
	}

	p.logger.Debug("Profiling ADO repository",
		"repo", repo.FullName,
		"repo_id", repoID,
		"project", projectName,
		"is_git", repo.ADOIsGit)

	// 1. Check repository type (Git vs TFVC)
	if !repo.ADOIsGit {
		// TFVC repository - mark for remediation and skip further analysis
		repo.Status = string(models.StatusRemediationRequired)
		p.logger.Warn("TFVC repository detected - requires git conversion before migration",
			"repo", repo.FullName)
		return nil
	}

	// 2. Profile Git properties (for Git repos)
	if err := p.profileGitProperties(ctx, repo, projectName, repoID); err != nil {
		p.logger.Warn("Failed to profile git properties",
			"repo", repo.FullName,
			"error", err)
	}

	// 3. Profile ADO-specific features
	if err := p.profileADOFeatures(ctx, repo, projectName, repoID); err != nil {
		p.logger.Warn("Failed to profile ADO features",
			"repo", repo.FullName,
			"error", err)
	}

	// 4. Profile what migrates with GEI
	// Per GitHub docs, GEI supports:
	// - Git source (commit history)
	// - Pull requests
	// - Work item links on PRs
	// - Attachments on PRs
	// - Branch policies (repo-level only)
	p.profileMigratableFeatures(ctx, repo, projectName, repoID)

	// 5. Clone and analyze Git properties (LFS, submodules, large files)
	// This performs deep Git analysis similar to GitHub repos
	if err := p.cloneAndAnalyzeGit(ctx, repo); err != nil {
		p.logger.Warn("Failed to clone and analyze Git properties",
			"repo", repo.FullName,
			"error", err)
		// Continue even if Git analysis fails - we have API-based data
	}

	p.logger.Info("ADO repository profiled",
		"repo", repo.FullName,
		"prs", repo.ADOPullRequestCount,
		"has_boards", repo.ADOHasBoards,
		"has_pipelines", repo.ADOHasPipelines,
		"branch_policies", repo.ADOBranchPolicyCount)

	return nil
}

// cloneAndAnalyzeGit clones the repository and analyzes Git properties
func (p *ADOProfiler) cloneAndAnalyzeGit(ctx context.Context, repo *models.Repository) error {
	// Type assert the provider to source.Provider
	provider, ok := p.sourceProvider.(source.Provider)
	if !ok || provider == nil {
		return fmt.Errorf("source provider not available")
	}

	// Setup temp directory for cloning
	tempDir, err := p.setupTempDir(repo.FullName)
	if err != nil {
		return fmt.Errorf("failed to setup temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			p.logger.Warn("Failed to clean up temp directory",
				"path", tempDir,
				"error", err)
		}
	}()

	p.logger.Info("Cloning repository for Git analysis",
		"repo", repo.FullName,
		"path", tempDir)

	// Clone the repository
	repoInfo := source.RepositoryInfo{
		FullName: repo.FullName,
		CloneURL: repo.SourceURL,
	}

	cloneOpts := source.CloneOptions{
		Shallow:           false, // Full clone required for git-sizer analysis
		Bare:              false,
		IncludeLFS:        true, // Fetch LFS to detect LFS usage
		IncludeSubmodules: false,
	}

	if err := provider.CloneRepository(ctx, repoInfo, tempDir, cloneOpts); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	p.logger.Info("Repository cloned, analyzing Git properties",
		"repo", repo.FullName)

	// Analyze Git properties using git-sizer
	analyzer := NewAnalyzer(p.logger)
	if err := analyzer.AnalyzeGitProperties(ctx, repo, tempDir); err != nil {
		return fmt.Errorf("failed to analyze Git properties: %w", err)
	}

	p.logger.Info("Git analysis complete",
		"repo", repo.FullName,
		"has_lfs", repo.HasLFS,
		"has_submodules", repo.HasSubmodules,
		"has_large_files", repo.HasLargeFiles)

	return nil
}

// setupTempDir creates a temporary directory for cloning
func (p *ADOProfiler) setupTempDir(fullName string) (string, error) {
	tempBase := os.TempDir()
	tempBase = filepath.Join(tempBase, "github-migrator-ado")

	// #nosec G301 -- 0755 is appropriate for temporary directory
	if err := os.MkdirAll(tempBase, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp base directory: %w", err)
	}

	// Use full name with slashes replaced to avoid collisions
	// For example: "org/project/repo" becomes "org_project_repo"
	safeName := strings.ReplaceAll(fullName, "/", "_")
	tempDir := filepath.Join(tempBase, safeName)

	// Remove if it already exists
	if err := os.RemoveAll(tempDir); err != nil {
		return "", fmt.Errorf("failed to remove existing temp directory: %w", err)
	}

	return tempDir, nil
}

// profileGitProperties profiles standard Git properties
func (p *ADOProfiler) profileGitProperties(ctx context.Context, repo *models.Repository, projectName, repoID string) error {
	// Get branch count
	branches, err := p.client.GetBranches(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get branches", "error", err)
	} else {
		repo.BranchCount = len(branches)
	}

	// Get commit count (approximate)
	commitCount, err := p.client.GetCommitCount(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get commit count", "error", err)
	} else {
		repo.CommitCount = commitCount
	}

	// Note: For accurate git-sizer analysis (LFS, submodules, large files, etc.),
	// we would need to clone the repository, which is handled by the standard
	// git analyzer used for GitHub repos. This can be reused for ADO Git repos.

	return nil
}

// profileADOFeatures profiles Azure DevOps-specific features
func (p *ADOProfiler) profileADOFeatures(ctx context.Context, repo *models.Repository, projectName, repoID string) error {
	p.profileAzureBoards(ctx, repo, projectName)
	p.profileAzurePipelines(ctx, repo, projectName, repoID)
	p.profilePullRequests(ctx, repo, projectName, repoID)
	p.profileBranchPolicies(ctx, repo, projectName, repoID)
	p.profileWorkItems(ctx, repo, projectName, repoID)
	p.profileAdditionalFeatures(ctx, repo, projectName, repoID)
	return nil
}

// profileAzureBoards profiles Azure Boards integration
func (p *ADOProfiler) profileAzureBoards(ctx context.Context, repo *models.Repository, projectName string) {
	hasBoards, err := p.client.HasAzureBoards(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to check Azure Boards", "error", err)
	} else {
		repo.ADOHasBoards = hasBoards
	}
}

// profileAzurePipelines profiles Azure Pipelines and related features
func (p *ADOProfiler) profileAzurePipelines(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	// Check if pipelines exist
	hasPipelines, err := p.client.HasAzurePipelines(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to check Azure Pipelines", "error", err)
	} else {
		repo.ADOHasPipelines = hasPipelines
	}

	// Get pipeline definitions and categorize them
	pipelineDefs, err := p.client.GetPipelineDefinitions(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get pipeline definitions", "error", err)
	} else {
		p.categorizePipelines(repo, pipelineDefs)
	}

	// Get recent pipeline runs
	pipelineRunCount, err := p.client.GetPipelineRuns(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get pipeline runs", "error", err)
	} else {
		repo.ADOPipelineRunCount = pipelineRunCount
	}

	// Check for service connections and variable groups
	p.profilePipelineResources(ctx, repo, projectName)

	// Check GitHub Advanced Security
	hasGHAS, err := p.client.HasGHAS(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to check GHAS", "error", err)
	} else {
		repo.ADOHasGHAS = hasGHAS
	}
}

// categorizePipelines categorizes pipelines into YAML and Classic
func (p *ADOProfiler) categorizePipelines(repo *models.Repository, pipelineDefs []build.BuildDefinitionReference) {
	repo.ADOPipelineCount = len(pipelineDefs)
	yamlCount := 0
	classicCount := 0
	for _, def := range pipelineDefs {
		if def.Path != nil && (strings.HasSuffix(*def.Path, ".yml") || strings.HasSuffix(*def.Path, ".yaml")) {
			yamlCount++
		} else {
			classicCount++
		}
	}
	repo.ADOYAMLPipelineCount = yamlCount
	repo.ADOClassicPipelineCount = classicCount
}

// profilePipelineResources profiles pipeline-related resources
func (p *ADOProfiler) profilePipelineResources(ctx context.Context, repo *models.Repository, projectName string) {
	serviceConnCount, err := p.client.GetServiceConnections(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get service connections", "error", err)
	} else {
		repo.ADOHasServiceConnections = serviceConnCount > 0
	}

	varGroupCount, err := p.client.GetVariableGroups(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get variable groups", "error", err)
	} else {
		repo.ADOHasVariableGroups = varGroupCount > 0
	}
}

// profilePullRequests profiles pull request details
func (p *ADOProfiler) profilePullRequests(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	openCount, withWorkItems, withAttachments, err := p.client.GetPRDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get PR details", "error", err)
	} else {
		repo.ADOOpenPRCount = openCount
		repo.ADOPRWithLinkedWorkItems = withWorkItems
		repo.ADOPRWithAttachments = withAttachments
	}

	prs, err := p.client.GetPullRequests(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get pull requests", "error", err)
	} else {
		repo.ADOPullRequestCount = len(prs)
		repo.PullRequestCount = len(prs)
	}
}

// profileBranchPolicies profiles branch protection policies
func (p *ADOProfiler) profileBranchPolicies(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	policyTypes, requiredReviewers, buildValidations, err := p.client.GetBranchPolicyDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get branch policy details", "error", err)
		return
	}

	repo.ADOBranchPolicyCount = len(policyTypes)
	repo.BranchProtections = len(policyTypes)
	repo.ADORequiredReviewerCount = requiredReviewers
	repo.ADOBuildValidationPolicies = buildValidations

	if len(policyTypes) > 0 {
		policyTypesJSON := fmt.Sprintf(`["%s"]`, joinStrings(policyTypes, `","`))
		repo.ADOBranchPolicyTypes = &policyTypesJSON
	}
}

// profileWorkItems profiles Azure Boards work items
func (p *ADOProfiler) profileWorkItems(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	linkedCount, activeCount, workItemTypes, err := p.client.GetWorkItemDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get work item details", "error", err)
		return
	}

	repo.ADOWorkItemLinkedCount = linkedCount
	repo.ADOActiveWorkItemCount = activeCount
	repo.ADOWorkItemCount = linkedCount

	if len(workItemTypes) > 0 {
		workItemTypesJSON := fmt.Sprintf(`["%s"]`, joinStrings(workItemTypes, `","`))
		repo.ADOWorkItemTypes = &workItemTypesJSON
	}
}

// profileAdditionalFeatures profiles wiki, test plans, package feeds, and service hooks
func (p *ADOProfiler) profileAdditionalFeatures(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	// Wiki
	hasWiki, wikiPageCount, err := p.client.GetWikiDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get wiki details", "error", err)
	} else {
		repo.ADOHasWiki = hasWiki
		repo.ADOWikiPageCount = wikiPageCount
	}

	// Test Plans
	testPlanCount, err := p.client.GetTestPlans(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get test plans", "error", err)
	} else {
		repo.ADOTestPlanCount = testPlanCount
	}

	// Package Feeds
	packageFeedCount, err := p.client.GetPackageFeeds(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get package feeds", "error", err)
	} else {
		repo.ADOPackageFeedCount = packageFeedCount
	}

	// Service Hooks
	serviceHookCount, err := p.client.GetServiceHooks(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get service hooks", "error", err)
	} else {
		repo.ADOServiceHookCount = serviceHookCount
	}
}

// joinStrings joins strings with a separator
func joinStrings(strings []string, sep string) string {
	if len(strings) == 0 {
		return ""
	}
	result := strings[0]
	for i := 1; i < len(strings); i++ {
		result += sep + strings[i]
	}
	return result
}

// profileMigratableFeatures determines what will and won't migrate with GEI
func (p *ADOProfiler) profileMigratableFeatures(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	// What migrates automatically with GEI (per GitHub docs):
	// ✅ Git source (commit history)
	// ✅ Pull requests
	// ✅ Work item links on PRs (but not the work items themselves)
	// ✅ Attachments on PRs
	// ✅ Branch policies (repo-level only, not user-scoped or cross-repo)

	// What doesn't migrate (requires manual work):
	// ❌ Azure Boards work items (only PR links migrate)
	// ❌ Azure Pipelines history/runs (YAML files migrate as source code)
	// ❌ Azure Repos wikis (separate from GitHub wikis)
	// ❌ Test Plans (no GitHub equivalent)
	// ❌ User-scoped branch policies
	// ❌ Cross-repo branch policies
	// ❌ Package feeds (require separate migration)
	// ❌ Service connections (must be recreated in GitHub)
	// ❌ Variable groups (must be recreated as GitHub secrets)

	// Log warnings for features that won't migrate
	if repo.ADOHasBoards && repo.ADOActiveWorkItemCount > 0 {
		p.logger.Warn("Repository has active Azure Boards work items - these won't migrate",
			"repo", repo.FullName,
			"active_work_items", repo.ADOActiveWorkItemCount,
			"note", "Only work item links on PRs will migrate")
	}

	if repo.ADOClassicPipelineCount > 0 {
		p.logger.Warn("Repository has Classic pipelines - these require manual recreation",
			"repo", repo.FullName,
			"classic_pipelines", repo.ADOClassicPipelineCount,
			"note", "Classic pipelines cannot be automatically converted to GitHub Actions")
	}

	if repo.ADOHasPipelines {
		p.logger.Info("Repository uses Azure Pipelines - pipeline history won't migrate",
			"repo", repo.FullName,
			"yaml_pipelines", repo.ADOYAMLPipelineCount,
			"classic_pipelines", repo.ADOClassicPipelineCount,
			"note", "YAML files migrate as source code, but execution history doesn't")
	}

	if repo.ADOHasWiki && repo.ADOWikiPageCount > 0 {
		p.logger.Warn("Repository has wiki pages - these require manual migration",
			"repo", repo.FullName,
			"wiki_pages", repo.ADOWikiPageCount,
			"note", "Azure Repos wikis are separate from GitHub wikis")
	}

	if repo.ADOTestPlanCount > 0 {
		p.logger.Warn("Repository has test plans - no GitHub equivalent exists",
			"repo", repo.FullName,
			"test_plans", repo.ADOTestPlanCount,
			"note", "Consider using third-party test management tools")
	}

	if repo.ADOPackageFeedCount > 0 {
		p.logger.Warn("Repository uses package feeds - require separate migration",
			"repo", repo.FullName,
			"package_feeds", repo.ADOPackageFeedCount,
			"note", "Migrate to GitHub Packages separately")
	}

	if repo.ADOHasServiceConnections {
		p.logger.Info("Project uses service connections - must be recreated in GitHub",
			"repo", repo.FullName,
			"note", "Recreate as GitHub Actions secrets and variables")
	}

	if repo.ADOHasVariableGroups {
		p.logger.Info("Project uses variable groups - must be recreated in GitHub",
			"repo", repo.FullName,
			"note", "Convert to GitHub repository or organization secrets")
	}

	if repo.ADOServiceHookCount > 0 {
		p.logger.Info("Repository has service hooks - must be recreated as webhooks",
			"repo", repo.FullName,
			"service_hooks", repo.ADOServiceHookCount)
	}

	if repo.ADOHasGHAS {
		p.logger.Info("Repository uses GitHub Advanced Security for Azure DevOps",
			"repo", repo.FullName,
			"note", "Enable GitHub Advanced Security in GitHub after migration")
	}

	// Log what WILL migrate successfully
	if repo.ADOPRWithLinkedWorkItems > 0 {
		p.logger.Info("Pull requests with work item links will migrate",
			"repo", repo.FullName,
			"prs_with_links", repo.ADOPRWithLinkedWorkItems)
	}

	if repo.ADOBranchPolicyCount > 0 {
		p.logger.Info("Branch policies will migrate (repository-level only)",
			"repo", repo.FullName,
			"policies", repo.ADOBranchPolicyCount,
			"note", "Verify and adjust policies after migration")
	}
}

// DetectTFVC checks if a repository is TFVC (vs Git)
func (p *ADOProfiler) DetectTFVC(ctx context.Context, projectName, repoName string) (bool, error) {
	isGit, err := p.client.IsGitRepo(ctx, projectName, repoName)
	if err != nil {
		return false, fmt.Errorf("failed to check repository type: %w", err)
	}

	// If not Git, it's TFVC
	return !isGit, nil
}

// EstimateComplexity estimates the complexity of migrating an ADO repository
// This is similar to GitHub complexity scoring but adjusted for ADO features
func (p *ADOProfiler) EstimateComplexity(repo *models.Repository) int {
	complexity := 0

	// TFVC repos are blocking - very high complexity
	if !repo.ADOIsGit {
		complexity += 50 // Requires Git conversion - BLOCKING
	}

	// Classic Pipelines (require manual recreation)
	complexity += repo.ADOClassicPipelineCount * 5 // 5 points per classic pipeline

	// Package Feeds (require separate migration)
	if repo.ADOPackageFeedCount > 0 {
		complexity += 3
	}

	// Service Connections (must recreate in GitHub)
	if repo.ADOHasServiceConnections {
		complexity += 3
	}

	// Active Pipelines with runs (CI/CD reconfiguration needed)
	if repo.ADOPipelineRunCount > 0 {
		complexity += 3
	}

	// Azure Boards with active work items (don't migrate)
	if repo.ADOActiveWorkItemCount > 0 {
		complexity += 3
	}

	// Wiki Pages (manual migration needed)
	if repo.ADOWikiPageCount > 0 {
		// 2 points per 10 pages
		complexity += ((repo.ADOWikiPageCount + 9) / 10) * 2
	}

	// Test Plans (no GitHub equivalent)
	if repo.ADOTestPlanCount > 0 {
		complexity += 2
	}

	// Variable Groups (convert to GitHub secrets)
	if repo.ADOHasVariableGroups {
		complexity += 1
	}

	// Service Hooks (recreate webhooks)
	if repo.ADOServiceHookCount > 0 {
		complexity += 1
	}

	// Many PRs (metadata migration time)
	if repo.ADOPullRequestCount > 50 {
		complexity += 2
	}

	// Branch Policies (need validation/recreation)
	if repo.ADOBranchPolicyCount > 0 {
		complexity += 1
	}

	// Standard git complexity factors (size, LFS, submodules, etc.)
	// These are added by the standard git analyzer via AnalyzeGitProperties

	return complexity
}

// EstimateComplexityWithBreakdown estimates complexity and provides a breakdown
func (p *ADOProfiler) EstimateComplexityWithBreakdown(repo *models.Repository) (int, *models.ComplexityBreakdown) {
	breakdown := &models.ComplexityBreakdown{}

	// TFVC - blocking
	if !repo.ADOIsGit {
		breakdown.ADOTFVCPoints = 50
	}

	// Classic Pipelines
	breakdown.ADOClassicPipelinePoints = repo.ADOClassicPipelineCount * 5

	// Package Feeds
	if repo.ADOPackageFeedCount > 0 {
		breakdown.ADOPackageFeedPoints = 3
	}

	// Service Connections
	if repo.ADOHasServiceConnections {
		breakdown.ADOServiceConnectionPoints = 3
	}

	// Active Pipelines
	if repo.ADOPipelineRunCount > 0 {
		breakdown.ADOActivePipelinePoints = 3
	}

	// Active Boards
	if repo.ADOActiveWorkItemCount > 0 {
		breakdown.ADOActiveBoardsPoints = 3
	}

	// Wiki Pages
	if repo.ADOWikiPageCount > 0 {
		breakdown.ADOWikiPoints = ((repo.ADOWikiPageCount + 9) / 10) * 2
	}

	// Test Plans
	if repo.ADOTestPlanCount > 0 {
		breakdown.ADOTestPlanPoints = 2
	}

	// Variable Groups
	if repo.ADOHasVariableGroups {
		breakdown.ADOVariableGroupPoints = 1
	}

	// Service Hooks
	if repo.ADOServiceHookCount > 0 {
		breakdown.ADOServiceHookPoints = 1
	}

	// Many PRs
	if repo.ADOPullRequestCount > 50 {
		breakdown.ADOManyPRsPoints = 2
	}

	// Branch Policies
	if repo.ADOBranchPolicyCount > 0 {
		breakdown.ADOBranchPolicyPoints = 1
	}

	// Calculate total
	total := breakdown.ADOTFVCPoints +
		breakdown.ADOClassicPipelinePoints +
		breakdown.ADOPackageFeedPoints +
		breakdown.ADOServiceConnectionPoints +
		breakdown.ADOActivePipelinePoints +
		breakdown.ADOActiveBoardsPoints +
		breakdown.ADOWikiPoints +
		breakdown.ADOTestPlanPoints +
		breakdown.ADOVariableGroupPoints +
		breakdown.ADOServiceHookPoints +
		breakdown.ADOManyPRsPoints +
		breakdown.ADOBranchPolicyPoints

	return total, breakdown
}
