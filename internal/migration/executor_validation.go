package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ghapi "github.com/google/go-github/v75/github"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// ValidationMismatch represents a mismatch between source and destination repository characteristics
type ValidationMismatch struct {
	Field       string      `json:"field"`
	SourceValue interface{} `json:"source_value"`
	DestValue   interface{} `json:"dest_value"`
	Critical    bool        `json:"critical"` // Whether this mismatch is critical (affects migration success)
}

// validatePreMigration performs pre-migration validation
// nolint:gocyclo // Complex validation logic - refactoring would reduce readability
func (e *Executor) validatePreMigration(ctx context.Context, repo *models.Repository, batch *models.Batch) error {
	// Check for GitHub Enterprise Importer blocking issues
	if repo.HasOversizedRepository {
		return fmt.Errorf("repository exceeds GitHub's 40 GiB size limit and requires remediation before migration (reduce repository size using Git LFS or history rewriting)")
	}

	// Check for blockers
	var issues []string

	// Check for very large files
	if repo.LargestFileSize != nil && *repo.LargestFileSize > 100*1024*1024 { // >100MB
		issues = append(issues, fmt.Sprintf("Very large file detected: %s (%d MB)",
			*repo.LargestFile, *repo.LargestFileSize/(1024*1024)))
	}

	// Check for very large repository
	if repo.TotalSize != nil && *repo.TotalSize > 50*1024*1024*1024 { // >50GB
		issues = append(issues, fmt.Sprintf("Very large repository: %d GB",
			*repo.TotalSize/(1024*1024*1024)))
	}

	// 1. Verify source repository exists and is accessible (GitHub sources only)
	// For ADO sources, sourceClient is nil and we skip this check
	// GEI will validate ADO source accessibility during migration
	var err error
	if e.sourceClient != nil {
		e.logger.Info("Checking source repository", "repo", repo.FullName)
		var sourceRepoData *ghapi.Repository

		_, err = e.sourceClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
			var resp *ghapi.Response
			sourceRepoData, resp, err = e.sourceClient.REST().Repositories.Get(ctx, repo.Organization(), repo.Name())
			return resp, err
		})

		if err != nil {
			return fmt.Errorf("source repository not found or inaccessible: %w", err)
		}

		e.logger.Info("Source repository verified", "repo", repo.FullName)

		// Verify repository is not archived
		if sourceRepoData.GetArchived() {
			issues = append(issues, "Repository is archived")
		}
	} else {
		e.logger.Info("Skipping source repository check (non-GitHub source)", "repo", repo.FullName)
	}

	// 2. Check if destination repository already exists
	destOrg := e.getDestinationOrg(repo, batch)
	destRepoName := e.getDestinationRepoName(repo)
	e.logger.Info("Checking destination repository",
		"source_repo", repo.FullName,
		"dest_org", destOrg,
		"dest_repo_name", destRepoName,
		"action", e.destRepoExistsAction)
	var destRepoData *ghapi.Repository
	destExists := false

	_, err = e.destClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		destRepoData, resp, err = e.destClient.REST().Repositories.Get(ctx, destOrg, destRepoName)
		return resp, err
	})

	if err == nil {
		// Destination repository exists
		destExists = true
		e.logger.Warn("Destination repository already exists",
			"repo", repo.FullName,
			"dest_repo", destRepoData.GetFullName(),
			"action", e.destRepoExistsAction)
	} else if github.IsNotFoundError(err) {
		// Destination repository does not exist - this is expected
		e.logger.Info("Destination repository does not exist - ready for migration", "repo", repo.FullName)
		destExists = false
	} else {
		// Some other error occurred
		e.logger.Warn("Unable to check destination repository", "repo", repo.FullName, "error", err)
		// Continue - we'll find out during migration if there's an issue
	}

	// Handle destination repository exists scenarios
	if destExists {
		switch e.destRepoExistsAction {
		case DestinationRepoExistsFail:
			return fmt.Errorf("destination repository already exists: %s (action: fail)", destRepoData.GetFullName())

		case DestinationRepoExistsSkip:
			e.logger.Info("Skipping migration - destination repository exists",
				"repo", repo.FullName,
				"action", e.destRepoExistsAction)
			return fmt.Errorf("destination repository already exists: %s (action: skip)", destRepoData.GetFullName())

		case DestinationRepoExistsDelete:
			e.logger.Warn("Deleting existing destination repository",
				"source_repo", repo.FullName,
				"dest_repo", destRepoData.GetFullName())

			// Delete the existing repository
			_, err = e.destClient.DoWithRetry(ctx, "DeleteRepository", func(ctx context.Context) (*ghapi.Response, error) {
				resp, err := e.destClient.REST().Repositories.Delete(ctx, destOrg, destRepoName)
				return resp, err
			})

			if err != nil {
				return fmt.Errorf("failed to delete existing destination repository: %w", err)
			}

			e.logger.Info("Successfully deleted existing destination repository",
				"repo", repo.FullName)
		}
	}

	// Log warnings but don't fail
	if len(issues) > 0 {
		e.logger.Warn("Pre-migration validation warnings",
			"repo", repo.FullName,
			"issues", issues)
	}

	return nil
}

// validatePostMigration performs comprehensive post-migration validation
func (e *Executor) validatePostMigration(ctx context.Context, repo *models.Repository) error {
	if repo.DestinationFullName == nil {
		return fmt.Errorf("destination repository not set")
	}

	e.logger.Info("Running post-migration validation with characteristic comparison",
		"repo", repo.FullName,
		"destination", *repo.DestinationFullName)

	// Profile the destination repository (API-only, no cloning)
	destRepo, err := e.profileDestinationRepository(ctx, *repo.DestinationFullName)
	if err != nil {
		return fmt.Errorf("failed to profile destination repository: %w", err)
	}

	// Compare source and destination characteristics
	mismatches, hasCriticalMismatches := e.compareRepositoryCharacteristics(repo, destRepo)

	// Generate validation report
	validationStatus := "passed"
	var validationDetails *string
	var destinationData *string

	if len(mismatches) > 0 {
		validationStatus = statusFailed

		// Log all mismatches
		e.logger.Warn("Post-migration validation found mismatches",
			"repo", repo.FullName,
			"mismatch_count", len(mismatches),
			"critical", hasCriticalMismatches)

		for _, mismatch := range mismatches {
			e.logger.Warn("Validation mismatch",
				"repo", repo.FullName,
				"field", mismatch.Field,
				"source", mismatch.SourceValue,
				"destination", mismatch.DestValue,
				"critical", mismatch.Critical)
		}

		// Generate JSON validation details
		validationReport := e.generateValidationReport(mismatches)
		validationDetails = &validationReport

		// Store destination data for further analysis
		destDataJSON := e.serializeDestinationData(destRepo)
		destinationData = &destDataJSON
	} else {
		e.logger.Info("Post-migration validation passed - all characteristics match",
			"repo", repo.FullName)
	}

	// Update validation fields in database
	if err := e.storage.UpdateRepositoryValidation(ctx, repo.FullName, validationStatus, validationDetails, destinationData); err != nil {
		e.logger.Error("Failed to update validation status", "error", err)
		// Don't fail the migration due to database update error
	}

	// Update repository validation fields
	repo.ValidationStatus = &validationStatus
	repo.ValidationDetails = validationDetails
	repo.DestinationData = destinationData

	// Don't fail migration on validation warnings - just log them
	return nil
}

// profileDestinationRepository profiles a destination repository using API-only metrics
func (e *Executor) profileDestinationRepository(ctx context.Context, fullName string) (*models.Repository, error) {
	// Parse full name
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository full name: %s", fullName)
	}
	org := parts[0]
	name := parts[1]

	// Get repository details from destination
	var ghRepo *ghapi.Repository
	var err error

	_, err = e.destClient.DoWithRetry(ctx, "GetRepository", func(ctx context.Context) (*ghapi.Response, error) {
		var resp *ghapi.Response
		ghRepo, resp, err = e.destClient.REST().Repositories.Get(ctx, org, name)
		return resp, err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get destination repository: %w", err)
	}

	// Create basic repository profile from GitHub API data
	totalSize := int64(ghRepo.GetSize()) * 1024 // Convert KB to bytes
	defaultBranch := ghRepo.GetDefaultBranch()
	repo := &models.Repository{
		FullName:      ghRepo.GetFullName(),
		DefaultBranch: &defaultBranch,
		TotalSize:     &totalSize,
		HasWiki:       ghRepo.GetHasWiki(),
		HasPages:      ghRepo.GetHasPages(),
		IsArchived:    ghRepo.GetArchived(),
	}

	// Get branch count
	branches, _, err := e.destClient.REST().Repositories.ListBranches(ctx, org, name, nil)
	if err == nil {
		repo.BranchCount = len(branches)
	}

	// Get last commit SHA from default branch
	if defaultBranch != "" {
		branch, _, err := e.destClient.REST().Repositories.GetBranch(ctx, org, name, defaultBranch, 0)
		if err == nil && branch != nil && branch.Commit != nil {
			sha := branch.Commit.GetSHA()
			repo.LastCommitSHA = &sha
		}
	}

	// Get commit count (approximation from contributors API)
	contributors, _, err := e.destClient.REST().Repositories.ListContributors(ctx, org, name, nil)
	if err == nil {
		totalCommits := 0
		for _, contributor := range contributors {
			totalCommits += contributor.GetContributions()
		}
		repo.CommitCount = totalCommits
	}

	// Get tag count
	tags, _, err := e.destClient.REST().Repositories.ListTags(ctx, org, name, nil)
	if err == nil {
		repo.TagCount = len(tags)
	}

	// Get issue and PR counts
	// Note: This is a simplified approach
	issues, _, err := e.destClient.REST().Issues.ListByRepo(ctx, org, name, &ghapi.IssueListByRepoOptions{
		State:       "all",
		ListOptions: ghapi.ListOptions{PerPage: 1},
	})
	if err == nil {
		// Count issues (excluding PRs)
		for _, issue := range issues {
			if issue.PullRequestLinks == nil {
				repo.IssueCount++
			} else {
				repo.PullRequestCount++
			}
		}
	}

	return repo, nil
}

// compareRepositoryCharacteristics compares source and destination repository characteristics
func (e *Executor) compareRepositoryCharacteristics(source, dest *models.Repository) ([]ValidationMismatch, bool) {
	var mismatches []ValidationMismatch
	hasCritical := false

	// Compare critical Git properties
	if source.DefaultBranch != nil && dest.DefaultBranch != nil && *source.DefaultBranch != *dest.DefaultBranch {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "default_branch",
			SourceValue: *source.DefaultBranch,
			DestValue:   *dest.DefaultBranch,
			Critical:    true,
		})
		hasCritical = true
	}

	if source.CommitCount != dest.CommitCount {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "commit_count",
			SourceValue: source.CommitCount,
			DestValue:   dest.CommitCount,
			Critical:    true,
		})
		hasCritical = true
	}

	if source.BranchCount != dest.BranchCount {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "branch_count",
			SourceValue: source.BranchCount,
			DestValue:   dest.BranchCount,
			Critical:    true,
		})
		hasCritical = true
	}

	if source.TagCount != dest.TagCount {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "tag_count",
			SourceValue: source.TagCount,
			DestValue:   dest.TagCount,
			Critical:    false,
		})
	}

	// Compare last commit SHA if available
	if source.LastCommitSHA != nil && dest.LastCommitSHA != nil && *source.LastCommitSHA != *dest.LastCommitSHA {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "last_commit_sha",
			SourceValue: *source.LastCommitSHA,
			DestValue:   *dest.LastCommitSHA,
			Critical:    true,
		})
		hasCritical = true
	}

	// Compare GitHub features (non-critical)
	if source.HasWiki != dest.HasWiki {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_wiki",
			SourceValue: source.HasWiki,
			DestValue:   dest.HasWiki,
			Critical:    false,
		})
	}

	if source.HasPages != dest.HasPages {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_pages",
			SourceValue: source.HasPages,
			DestValue:   dest.HasPages,
			Critical:    false,
		})
	}

	if source.HasDiscussions != dest.HasDiscussions {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_discussions",
			SourceValue: source.HasDiscussions,
			DestValue:   dest.HasDiscussions,
			Critical:    false,
		})
	}

	if source.HasActions != dest.HasActions {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "has_actions",
			SourceValue: source.HasActions,
			DestValue:   dest.HasActions,
			Critical:    false,
		})
	}

	if source.BranchProtections != dest.BranchProtections {
		mismatches = append(mismatches, ValidationMismatch{
			Field:       "branch_protections",
			SourceValue: source.BranchProtections,
			DestValue:   dest.BranchProtections,
			Critical:    false,
		})
	}

	return mismatches, hasCritical
}

// generateValidationReport generates a JSON validation report from mismatches
func (e *Executor) generateValidationReport(mismatches []ValidationMismatch) string {
	type Report struct {
		TotalMismatches    int                  `json:"total_mismatches"`
		CriticalMismatches int                  `json:"critical_mismatches"`
		Mismatches         []ValidationMismatch `json:"mismatches"`
	}

	criticalCount := 0
	for _, m := range mismatches {
		if m.Critical {
			criticalCount++
		}
	}

	report := Report{
		TotalMismatches:    len(mismatches),
		CriticalMismatches: criticalCount,
		Mismatches:         mismatches,
	}

	// Marshal to JSON
	data, err := json.Marshal(report)
	if err != nil {
		e.logger.Error("Failed to marshal validation report", "error", err)
		return fmt.Sprintf(`{"error": "failed to generate report: %s"}`, err.Error())
	}

	return string(data)
}

// serializeDestinationData serializes destination repository data to JSON
func (e *Executor) serializeDestinationData(dest *models.Repository) string {
	// Create a simplified struct with key fields
	type DestData struct {
		DefaultBranch     *string `json:"default_branch,omitempty"`
		BranchCount       int     `json:"branch_count"`
		CommitCount       int     `json:"commit_count"`
		TagCount          int     `json:"tag_count"`
		LastCommitSHA     *string `json:"last_commit_sha,omitempty"`
		TotalSize         *int64  `json:"total_size,omitempty"`
		HasWiki           bool    `json:"has_wiki"`
		HasPages          bool    `json:"has_pages"`
		HasDiscussions    bool    `json:"has_discussions"`
		HasActions        bool    `json:"has_actions"`
		BranchProtections int     `json:"branch_protections"`
		IssueCount        int     `json:"issue_count"`
		PullRequestCount  int     `json:"pull_request_count"`
	}

	data := DestData{
		DefaultBranch:     dest.DefaultBranch,
		BranchCount:       dest.BranchCount,
		CommitCount:       dest.CommitCount,
		TagCount:          dest.TagCount,
		LastCommitSHA:     dest.LastCommitSHA,
		TotalSize:         dest.TotalSize,
		HasWiki:           dest.HasWiki,
		HasPages:          dest.HasPages,
		HasDiscussions:    dest.HasDiscussions,
		HasActions:        dest.HasActions,
		BranchProtections: dest.BranchProtections,
		IssueCount:        dest.IssueCount,
		PullRequestCount:  dest.PullRequestCount,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		e.logger.Error("Failed to serialize destination data", "error", err)
		return fmt.Sprintf(`{"error": "failed to serialize: %s"}`, err.Error())
	}

	return string(jsonData)
}
