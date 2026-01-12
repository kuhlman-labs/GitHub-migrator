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
	sourceProvider any // Will be source.Provider
	storage        any // Will be *storage.Database
}

// NewADOProfiler creates a new ADO profiler
func NewADOProfiler(client *azuredevops.Client, logger *slog.Logger, sourceProvider any, storage any) *ADOProfiler {
	return &ADOProfiler{
		client:         client,
		logger:         logger,
		sourceProvider: sourceProvider,
		storage:        storage,
	}
}

// ProfileRepository profiles an Azure DevOps repository
// This includes both Git analysis (if it's a Git repo) and ADO-specific features
func (p *ADOProfiler) ProfileRepository(ctx context.Context, repo *models.Repository, adoRepo any) error {
	adoProject := repo.GetADOProject()
	if adoProject == nil {
		return fmt.Errorf("repository missing ADO project name")
	}

	projectName := *adoProject

	// Extract repo ID from adoRepo - type assert to *git.GitRepository
	repoID := ""
	if gitRepo, ok := adoRepo.(*git.GitRepository); ok {
		if gitRepo.Id != nil {
			repoID = gitRepo.Id.String()
		}
	} else {
		p.logger.Warn("Failed to type assert adoRepo to *git.GitRepository",
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
		"is_git", repo.GetADOIsGit())

	// 1. Check repository type (Git vs TFVC)
	if !repo.GetADOIsGit() {
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

	// 6. Calculate complexity score based on all profiled features
	complexity, breakdown := p.EstimateComplexityWithBreakdown(repo)
	repo.SetComplexityScore(&complexity)

	// Serialize complexity breakdown to JSON for storage
	if err := repo.SetComplexityBreakdown(breakdown); err != nil {
		p.logger.Warn("Failed to serialize complexity breakdown",
			"repo", repo.FullName,
			"error", err)
	}

	p.logger.Info("ADO repository profiled",
		"repo", repo.FullName,
		"prs", repo.GetADOPullRequestCount(),
		"has_boards", repo.GetADOHasBoards(),
		"has_pipelines", repo.GetADOHasPipelines(),
		"branch_policies", repo.GetADOBranchPolicyCount(),
		"complexity", complexity)

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
		"has_lfs", repo.HasLFS(),
		"has_submodules", repo.HasSubmodules(),
		"has_large_files", repo.HasLargeFiles())

	// Analyze and save dependencies (submodules, etc.)
	if err := p.analyzeDependencies(ctx, repo, tempDir); err != nil {
		p.logger.Warn("Failed to analyze dependencies",
			"repo", repo.FullName,
			"error", err)
		// Don't fail the whole profiling if dependency analysis fails
	}

	return nil
}

// analyzeDependencies analyzes repository dependencies and saves them to the database
func (p *ADOProfiler) analyzeDependencies(ctx context.Context, repo *models.Repository, repoPath string) error {
	p.logger.Debug("Analyzing dependencies", "repo", repo.FullName)

	// Create dependency analyzer
	depAnalyzer := NewDependencyAnalyzer(p.logger)

	// Analyze dependencies from cloned repo (submodules, workflows, etc.)
	// Note: Workflow dependencies are GitHub Actions specific, but submodules work for any Git repo
	dependencies, err := depAnalyzer.AnalyzeDependencies(ctx, repoPath, repo.FullName, repo.ID, repo.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies from repo: %w", err)
	}

	// Save dependencies to database
	if len(dependencies) > 0 {
		// Type assert storage to *storage.Database
		if db, ok := p.storage.(interface {
			SaveRepositoryDependencies(ctx context.Context, repoID int64, dependencies []*models.RepositoryDependency) error
		}); ok {
			if err := db.SaveRepositoryDependencies(ctx, repo.ID, dependencies); err != nil {
				return fmt.Errorf("failed to save dependencies: %w", err)
			}

			p.logger.Info("Dependencies saved",
				"repo", repo.FullName,
				"count", len(dependencies))
		} else {
			p.logger.Warn("Storage not available for saving dependencies", "repo", repo.FullName)
		}
	}

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
	// Also sanitize other potentially dangerous characters
	safeName = strings.ReplaceAll(safeName, "..", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	tempDir := filepath.Join(tempBase, safeName)

	// Validate the constructed path
	if err := source.ValidateRepoPath(tempDir); err != nil {
		return "", fmt.Errorf("invalid temp directory path: %w", err)
	}

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
		repo.SetBranchCount(len(branches))
	}

	// Get commit count (approximate)
	commitCount, err := p.client.GetCommitCount(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get commit count", "error", err)
	} else {
		repo.SetCommitCount(commitCount)
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
		repo.SetADOHasBoards(hasBoards)
	}
}

// profileAzurePipelines profiles Azure Pipelines and related features
func (p *ADOProfiler) profileAzurePipelines(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	// Check if pipelines exist
	hasPipelines, err := p.client.HasAzurePipelines(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to check Azure Pipelines", "error", err)
	} else {
		repo.SetADOHasPipelines(hasPipelines)
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
		repo.SetADOPipelineRunCount(pipelineRunCount)
	}

	// Check for service connections and variable groups
	p.profilePipelineResources(ctx, repo, projectName)

	// Check GitHub Advanced Security
	hasGHAS, err := p.client.HasGHAS(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to check GHAS", "error", err)
	} else {
		repo.SetADOHasGHAS(hasGHAS)
	}
}

// categorizePipelines categorizes pipelines into YAML and Classic
func (p *ADOProfiler) categorizePipelines(repo *models.Repository, pipelineDefs []build.BuildDefinition) {
	repo.SetADOPipelineCount(len(pipelineDefs))
	yamlCount := 0
	classicCount := 0
	for _, def := range pipelineDefs {
		// Check the Process field to determine pipeline type
		// Process.Type: 1 = Designer (Classic), 2 = YAML
		if def.Process != nil {
			// Type assert to check the process type
			// The Process field is an interface{} that can be different types
			if processMap, ok := def.Process.(map[string]any); ok {
				if processType, ok := processMap["type"].(float64); ok {
					if processType == 2 {
						// YAML pipeline
						yamlCount++
					} else {
						// Classic/Designer pipeline
						classicCount++
					}
					continue
				}
			}
		}
		// Fallback: if we can't determine, assume classic
		classicCount++
	}
	repo.SetADOYAMLPipelineCount(yamlCount)
	repo.SetADOClassicPipelineCount(classicCount)
}

// profilePipelineResources profiles pipeline-related resources
func (p *ADOProfiler) profilePipelineResources(ctx context.Context, repo *models.Repository, projectName string) {
	serviceConnCount, err := p.client.GetServiceConnections(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get service connections", "error", err)
	} else {
		repo.SetADOHasServiceConnections(serviceConnCount > 0)
	}

	varGroupCount, err := p.client.GetVariableGroups(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get variable groups", "error", err)
	} else {
		repo.SetADOHasVariableGroups(varGroupCount > 0)
	}
}

// profilePullRequests profiles pull request details
func (p *ADOProfiler) profilePullRequests(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	openCount, withWorkItems, withAttachments, err := p.client.GetPRDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get PR details", "error", err)
	} else {
		repo.SetADOOpenPRCount(openCount)
		repo.SetADOPRWithLinkedWorkItems(withWorkItems)
		repo.SetADOPRWithAttachments(withAttachments)
	}

	prs, err := p.client.GetPullRequests(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get pull requests", "error", err)
	} else {
		repo.SetADOPullRequestCount(len(prs))
		repo.SetPullRequestCount(len(prs))
	}
}

// profileBranchPolicies profiles branch protection policies
func (p *ADOProfiler) profileBranchPolicies(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	policyTypes, requiredReviewers, buildValidations, err := p.client.GetBranchPolicyDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get branch policy details", "error", err)
		return
	}

	repo.SetADOBranchPolicyCount(len(policyTypes))
	repo.SetBranchProtections(len(policyTypes))
	repo.SetADORequiredReviewerCount(requiredReviewers)
	repo.SetADOBuildValidationPolicies(buildValidations)

	if len(policyTypes) > 0 {
		policyTypesJSON := fmt.Sprintf(`["%s"]`, joinStrings(policyTypes, `","`))
		repo.SetADOBranchPolicyTypes(&policyTypesJSON)
	}
}

// profileWorkItems profiles Azure Boards work items
func (p *ADOProfiler) profileWorkItems(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	linkedCount, activeCount, workItemTypes, err := p.client.GetWorkItemDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get work item details", "error", err)
		return
	}

	repo.SetADOWorkItemLinkedCount(linkedCount)
	repo.SetADOActiveWorkItemCount(activeCount)
	repo.SetADOWorkItemCount(linkedCount)

	if len(workItemTypes) > 0 {
		workItemTypesJSON := fmt.Sprintf(`["%s"]`, joinStrings(workItemTypes, `","`))
		repo.SetADOWorkItemTypes(&workItemTypesJSON)
	}
}

// profileAdditionalFeatures profiles wiki, test plans, package feeds, and service hooks
func (p *ADOProfiler) profileAdditionalFeatures(ctx context.Context, repo *models.Repository, projectName, repoID string) {
	// Wiki
	hasWiki, wikiPageCount, err := p.client.GetWikiDetails(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get wiki details", "error", err)
	} else {
		repo.SetADOHasWiki(hasWiki)
		repo.SetADOWikiPageCount(wikiPageCount)
	}

	// Test Plans
	testPlanCount, err := p.client.GetTestPlans(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get test plans", "error", err)
	} else {
		repo.SetADOTestPlanCount(testPlanCount)
	}

	// Package Feeds
	packageFeedCount, err := p.client.GetPackageFeeds(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get package feeds", "error", err)
	} else {
		repo.SetADOPackageFeedCount(packageFeedCount)
	}

	// Service Hooks
	serviceHookCount, err := p.client.GetServiceHooks(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to get service hooks", "error", err)
	} else {
		repo.SetADOServiceHookCount(serviceHookCount)
	}
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString(strs[0])
	for i := 1; i < len(strs); i++ {
		result.WriteString(sep + strs[i])
	}
	return result.String()
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
	if repo.GetADOHasBoards() && repo.GetADOActiveWorkItemCount() > 0 {
		p.logger.Warn("Repository has active Azure Boards work items - these won't migrate",
			"repo", repo.FullName,
			"active_work_items", repo.GetADOActiveWorkItemCount(),
			"note", "Only work item links on PRs will migrate")
	}

	if repo.GetADOClassicPipelineCount() > 0 {
		p.logger.Warn("Repository has Classic pipelines - these require manual recreation",
			"repo", repo.FullName,
			"classic_pipelines", repo.GetADOClassicPipelineCount(),
			"note", "Classic pipelines cannot be automatically converted to GitHub Actions")
	}

	if repo.GetADOHasPipelines() {
		p.logger.Info("Repository uses Azure Pipelines - pipeline history won't migrate",
			"repo", repo.FullName,
			"yaml_pipelines", repo.GetADOYAMLPipelineCount(),
			"classic_pipelines", repo.GetADOClassicPipelineCount(),
			"note", "YAML files migrate as source code, but execution history doesn't")
	}

	if repo.GetADOHasWiki() && repo.GetADOWikiPageCount() > 0 {
		p.logger.Warn("Repository has wiki pages - these require manual migration",
			"repo", repo.FullName,
			"wiki_pages", repo.GetADOWikiPageCount(),
			"note", "Azure Repos wikis are separate from GitHub wikis")
	}

	if repo.GetADOTestPlanCount() > 0 {
		p.logger.Warn("Repository has test plans - no GitHub equivalent exists",
			"repo", repo.FullName,
			"test_plans", repo.GetADOTestPlanCount(),
			"note", "Consider using third-party test management tools")
	}

	if repo.GetADOPackageFeedCount() > 0 {
		p.logger.Warn("Repository uses package feeds - require separate migration",
			"repo", repo.FullName,
			"package_feeds", repo.GetADOPackageFeedCount(),
			"note", "Migrate to GitHub Packages separately")
	}

	if repo.GetADOHasServiceConnections() {
		p.logger.Info("Project uses service connections - must be recreated in GitHub",
			"repo", repo.FullName,
			"note", "Recreate as GitHub Actions secrets and variables")
	}

	if repo.GetADOHasVariableGroups() {
		p.logger.Info("Project uses variable groups - must be recreated in GitHub",
			"repo", repo.FullName,
			"note", "Convert to GitHub repository or organization secrets")
	}

	if repo.GetADOServiceHookCount() > 0 {
		p.logger.Info("Repository has service hooks - must be recreated as webhooks",
			"repo", repo.FullName,
			"service_hooks", repo.GetADOServiceHookCount())
	}

	if repo.GetADOHasGHAS() {
		p.logger.Info("Repository uses GitHub Advanced Security for Azure DevOps",
			"repo", repo.FullName,
			"note", "Enable GitHub Advanced Security in GitHub after migration")
	}

	// Log what WILL migrate successfully
	if repo.GetADOPRWithLinkedWorkItems() > 0 {
		p.logger.Info("Pull requests with work item links will migrate",
			"repo", repo.FullName,
			"prs_with_links", repo.GetADOPRWithLinkedWorkItems())
	}

	if repo.GetADOBranchPolicyCount() > 0 {
		p.logger.Info("Branch policies will migrate (repository-level only)",
			"repo", repo.FullName,
			"policies", repo.GetADOBranchPolicyCount(),
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
	if !repo.GetADOIsGit() {
		complexity += 50 // Requires Git conversion - BLOCKING
	}

	// Classic Pipelines (require manual recreation)
	complexity += repo.GetADOClassicPipelineCount() * 5 // 5 points per classic pipeline

	// Package Feeds (require separate migration)
	if repo.GetADOPackageFeedCount() > 0 {
		complexity += 3
	}

	// Service Connections (must recreate in GitHub)
	if repo.GetADOHasServiceConnections() {
		complexity += 3
	}

	// Active Pipelines with runs (CI/CD reconfiguration needed)
	if repo.GetADOPipelineRunCount() > 0 {
		complexity += 3
	}

	// Azure Boards with active work items (don't migrate)
	if repo.GetADOActiveWorkItemCount() > 0 {
		complexity += 3
	}

	// Wiki Pages (manual migration needed)
	if repo.GetADOWikiPageCount() > 0 {
		// 2 points per 10 pages
		complexity += ((repo.GetADOWikiPageCount() + 9) / 10) * 2
	}

	// Test Plans (no GitHub equivalent)
	if repo.GetADOTestPlanCount() > 0 {
		complexity += 2
	}

	// Variable Groups (convert to GitHub secrets)
	if repo.GetADOHasVariableGroups() {
		complexity += 1
	}

	// Service Hooks (recreate webhooks)
	if repo.GetADOServiceHookCount() > 0 {
		complexity += 1
	}

	// Many PRs (metadata migration time)
	if repo.GetADOPullRequestCount() > 50 {
		complexity += 2
	}

	// Branch Policies (need validation/recreation)
	if repo.GetADOBranchPolicyCount() > 0 {
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
	if !repo.GetADOIsGit() {
		breakdown.ADOTFVCPoints = 50
	}

	// Classic Pipelines
	breakdown.ADOClassicPipelinePoints = repo.GetADOClassicPipelineCount() * 5

	// Package Feeds
	if repo.GetADOPackageFeedCount() > 0 {
		breakdown.ADOPackageFeedPoints = 3
	}

	// Service Connections
	if repo.GetADOHasServiceConnections() {
		breakdown.ADOServiceConnectionPoints = 3
	}

	// Active Pipelines
	if repo.GetADOPipelineRunCount() > 0 {
		breakdown.ADOActivePipelinePoints = 3
	}

	// Active Boards
	if repo.GetADOActiveWorkItemCount() > 0 {
		breakdown.ADOActiveBoardsPoints = 3
	}

	// Wiki Pages
	if repo.GetADOWikiPageCount() > 0 {
		breakdown.ADOWikiPoints = ((repo.GetADOWikiPageCount() + 9) / 10) * 2
	}

	// Test Plans
	if repo.GetADOTestPlanCount() > 0 {
		breakdown.ADOTestPlanPoints = 2
	}

	// Variable Groups
	if repo.GetADOHasVariableGroups() {
		breakdown.ADOVariableGroupPoints = 1
	}

	// Service Hooks
	if repo.GetADOServiceHookCount() > 0 {
		breakdown.ADOServiceHookPoints = 1
	}

	// Many PRs
	if repo.GetADOPullRequestCount() > 50 {
		breakdown.ADOManyPRsPoints = 2
	}

	// Branch Policies
	if repo.GetADOBranchPolicyCount() > 0 {
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
