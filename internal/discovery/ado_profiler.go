package discovery

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/brettkuhlman/github-migrator/internal/azuredevops"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

// ADOProfiler profiles Azure DevOps repositories
type ADOProfiler struct {
	client *azuredevops.Client
	logger *slog.Logger
}

// NewADOProfiler creates a new ADO profiler
func NewADOProfiler(client *azuredevops.Client, logger *slog.Logger) *ADOProfiler {
	return &ADOProfiler{
		client: client,
		logger: logger,
	}
}

// ProfileRepository profiles an Azure DevOps repository
// This includes both Git analysis (if it's a Git repo) and ADO-specific features
func (p *ADOProfiler) ProfileRepository(ctx context.Context, repo *models.Repository, adoRepo interface{}) error {
	if repo.ADOProject == nil {
		return fmt.Errorf("repository missing ADO project name")
	}

	projectName := *repo.ADOProject
	// Extract repo ID from adoRepo if available
	// Type assertion would be needed based on the actual ADO SDK type
	repoID := ""
	// TODO: Extract repoID from adoRepo when integrated with actual ADO SDK types

	p.logger.Debug("Profiling ADO repository",
		"repo", repo.FullName,
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

	p.logger.Info("ADO repository profiled",
		"repo", repo.FullName,
		"prs", repo.ADOPullRequestCount,
		"has_boards", repo.ADOHasBoards,
		"has_pipelines", repo.ADOHasPipelines,
		"branch_policies", repo.ADOBranchPolicyCount)

	return nil
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
