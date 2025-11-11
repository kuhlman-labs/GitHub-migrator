package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/azuredevops"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/source"
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
	// 1. Check Azure Boards integration
	hasBoards, err := p.client.HasAzureBoards(ctx, projectName)
	if err != nil {
		p.logger.Debug("Failed to check Azure Boards", "error", err)
	} else {
		repo.ADOHasBoards = hasBoards
	}

	// 2. Check Azure Pipelines
	hasPipelines, err := p.client.HasAzurePipelines(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to check Azure Pipelines", "error", err)
	} else {
		repo.ADOHasPipelines = hasPipelines
	}

	// 3. Check GitHub Advanced Security for Azure DevOps
	hasGHAS, err := p.client.HasGHAS(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to check GHAS", "error", err)
	} else {
		repo.ADOHasGHAS = hasGHAS
	}

	// 4. Get Pull Request count
	prs, err := p.client.GetPullRequests(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get pull requests", "error", err)
	} else {
		repo.ADOPullRequestCount = len(prs)
		// Also set the standard PullRequestCount for consistency
		repo.PullRequestCount = len(prs)
	}

	// 5. Get Branch Policies
	policies, err := p.client.GetBranchPolicies(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get branch policies", "error", err)
	} else {
		repo.ADOBranchPolicyCount = len(policies)
		// Also set the standard BranchProtections for consistency
		repo.BranchProtections = len(policies)
	}

	// 6. Get Work Items linked to repository
	// Note: This is complex and may require querying work items
	// For now, use the placeholder method
	workItemCount, err := p.client.GetWorkItemsLinkedToRepo(ctx, projectName, repoID)
	if err != nil {
		p.logger.Debug("Failed to get work items", "error", err)
	} else {
		repo.ADOWorkItemCount = workItemCount
	}

	return nil
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
	// ❌ Azure Pipelines history/runs (YAML files migrate, but history doesn't)
	// ❌ Azure Repos wikis (separate from GitHub wikis)
	// ❌ User-scoped branch policies
	// ❌ Cross-repo branch policies

	// Log warnings for features that won't migrate
	if repo.ADOHasBoards {
		p.logger.Info("Repository uses Azure Boards - work items won't migrate, only PR links",
			"repo", repo.FullName)
	}

	if repo.ADOHasPipelines {
		p.logger.Info("Repository uses Azure Pipelines - YAML files will migrate, but pipeline history won't",
			"repo", repo.FullName)
	}

	if repo.ADOHasGHAS {
		p.logger.Info("Repository uses GitHub Advanced Security for Azure DevOps",
			"repo", repo.FullName)
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
		complexity += 50 // Requires conversion
	}

	// Azure Boards (work items don't migrate, only PR links)
	if repo.ADOHasBoards {
		complexity += 3 // Manual migration needed for work items
	}

	// Azure Pipelines (history doesn't migrate)
	if repo.ADOHasPipelines {
		complexity += 3 // YAML files migrate, but not history
	}

	// Pull requests (these migrate with GEI)
	if repo.ADOPullRequestCount > 50 {
		complexity += 2
	} else if repo.ADOPullRequestCount > 10 {
		complexity += 1
	}

	// Branch policies (these migrate with GEI)
	if repo.ADOBranchPolicyCount > 0 {
		complexity += 1
	}

	// Work items (PR links migrate, but not the work items themselves)
	if repo.ADOWorkItemCount > 0 {
		complexity += 1
	}

	// Standard git complexity factors (size, LFS, submodules, etc.)
	// These would be added by the standard git analyzer

	return complexity
}

// splitADOFullName splits an ADO full name into parts
// Format: "org/project/repo" -> ["org", "project", "repo"]
func splitADOFullName(fullName string) []string {
	// For ADO repos, we expect "org/project/repo" format
	// We need to handle cases where project or repo names might contain slashes
	// For now, we'll use a simple split and take first 3 parts
	parts := make([]string, 0, 3)
	remainder := fullName
	
	// Split org (first part before /)
	if idx := findNthSlash(remainder, 0); idx >= 0 {
		parts = append(parts, remainder[:idx])
		remainder = remainder[idx+1:]
	} else {
		return []string{fullName}
	}
	
	// Split project (second part before /)
	if idx := findNthSlash(remainder, 0); idx >= 0 {
		parts = append(parts, remainder[:idx])
		remainder = remainder[idx+1:]
	} else {
		parts = append(parts, remainder)
		return parts
	}
	
	// Repo is everything else
	parts = append(parts, remainder)
	return parts
}

// findNthSlash finds the nth occurrence of '/' in a string
func findNthSlash(s string, n int) int {
	count := 0
	for i, c := range s {
		if c == '/' {
			if count == n {
				return i
			}
			count++
		}
	}
	return -1
}
