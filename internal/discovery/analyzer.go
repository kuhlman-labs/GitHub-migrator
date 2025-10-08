package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/models"
)

// Analyzer analyzes Git repository properties
type Analyzer struct {
	logger *slog.Logger
}

// NewAnalyzer creates a new Git analyzer
func NewAnalyzer(logger *slog.Logger) *Analyzer {
	return &Analyzer{
		logger: logger,
	}
}

// GitSizerOutput represents the JSON output from git-sizer
// Based on actual git-sizer output format: https://github.com/github/git-sizer
// Values are returned as integers directly in the JSON, not wrapped in objects
type GitSizerOutput struct {
	UniqueCommitCount         int64  `json:"unique_commit_count"`
	UniqueCommitSize          int64  `json:"unique_commit_size"`
	UniqueTreeCount           int64  `json:"unique_tree_count"`
	UniqueTreeSize            int64  `json:"unique_tree_size"`
	UniqueBlobCount           int64  `json:"unique_blob_count"`
	UniqueBlobSize            int64  `json:"unique_blob_size"`
	UniqueTagCount            int64  `json:"unique_tag_count"`
	MaxCommitSize             int64  `json:"max_commit_size"`
	MaxCommit                 string `json:"max_commit"` // SHA of largest commit
	MaxTreeEntries            int64  `json:"max_tree_entries"`
	MaxBlobSize               int64  `json:"max_blob_size"`
	MaxBlobSizeBlob           string `json:"max_blob_size_blob"` // Blob info for largest file
	MaxHistoryDepth           int64  `json:"max_history_depth"`
	MaxTagDepth               int64  `json:"max_tag_depth"`
	MaxPathDepth              int64  `json:"max_path_depth"`
	MaxPathLength             int64  `json:"max_path_length"`
	MaxExpandedTreeCount      int64  `json:"max_expanded_tree_count"`
	MaxExpandedBlobCount      int64  `json:"max_expanded_blob_count"`
	MaxExpandedBlobSize       int64  `json:"max_expanded_blob_size"`
	MaxExpandedLinkCount      int64  `json:"max_expanded_link_count"`
	MaxExpandedSubmoduleCount int64  `json:"max_expanded_submodule_count"`
}

// AnalyzeGitProperties analyzes Git repository using git-sizer and additional detection methods
func (a *Analyzer) AnalyzeGitProperties(ctx context.Context, repo *models.Repository, repoPath string) error {
	a.logger.Debug("Analyzing Git properties",
		"repo", repo.FullName,
		"path", repoPath)

	// Get actual disk size using du command
	diskSize, err := a.getDiskSize(ctx, repoPath)
	if err != nil {
		a.logger.Warn("Failed to get disk size, will use git-sizer estimate",
			"repo", repo.FullName,
			"error", err)
		// Fall back to git-sizer estimate if du fails
		diskSize = 0
	}

	// Run git-sizer with JSON output
	output, err := a.runGitSizer(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("git-sizer failed: %w", err)
	}

	// Use disk size if available, otherwise use git-sizer calculation
	if diskSize > 0 {
		repo.TotalSize = &diskSize
	} else {
		totalSize := output.UniqueBlobSize + output.UniqueTreeSize + output.UniqueCommitSize
		repo.TotalSize = &totalSize
	}

	// Store largest file info
	largestFileSize := output.MaxBlobSize
	repo.LargestFileSize = &largestFileSize
	if output.MaxBlobSizeBlob != "" {
		// Extract filename from blob info (format: "hash (commit:path)")
		filename := a.extractFilenameFromBlobInfo(output.MaxBlobSizeBlob)
		if filename != "" {
			repo.LargestFile = &filename
		}
	}

	// Store largest commit info
	largestCommitSize := output.MaxCommitSize
	repo.LargestCommitSize = &largestCommitSize
	if output.MaxCommit != "" {
		// Store just the commit SHA (may have branch info appended)
		commitSHA := a.extractCommitSHA(output.MaxCommit)
		repo.LargestCommit = &commitSHA
	}

	repo.CommitCount = int(output.UniqueCommitCount)

	// Detect LFS using .gitattributes and git lfs ls-files
	repo.HasLFS = a.detectLFS(ctx, repoPath)

	// Detect submodules using .gitmodules file and git submodule command
	repo.HasSubmodules = a.detectSubmodules(ctx, repoPath)

	// Get branch count
	repo.BranchCount = a.getBranchCount(ctx, repoPath)

	a.logger.Info("Git analysis complete",
		"repo", repo.FullName,
		"total_size", repo.TotalSize,
		"largest_file", repo.LargestFile,
		"largest_file_size", repo.LargestFileSize,
		"largest_commit", repo.LargestCommit,
		"largest_commit_size", repo.LargestCommitSize,
		"commits", repo.CommitCount,
		"has_lfs", repo.HasLFS,
		"has_submodules", repo.HasSubmodules,
		"branches", repo.BranchCount)

	return nil
}

// runGitSizer executes git-sizer and parses its JSON output
func (a *Analyzer) runGitSizer(ctx context.Context, repoPath string) (*GitSizerOutput, error) {
	cmd := exec.CommandContext(ctx, "git-sizer", "--json", "--json-version=2")
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git-sizer execution failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse JSON output directly
	var output GitSizerOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("failed to parse git-sizer output: %w", err)
	}

	return &output, nil
}

// detectLFS checks for Git LFS usage using two methods:
// 1. Check .gitattributes for "filter=lfs"
// 2. Run "git lfs ls-files" to list LFS-tracked files
func (a *Analyzer) detectLFS(ctx context.Context, repoPath string) bool {
	// Method 1: Check .gitattributes file
	gitattributesPath := repoPath + "/.gitattributes"
	// #nosec G304 -- repoPath is a controlled temporary directory path
	if data, err := os.ReadFile(gitattributesPath); err == nil {
		if strings.Contains(string(data), "filter=lfs") {
			a.logger.Debug("LFS detected via .gitattributes", "repo_path", repoPath)
			return true
		}
	}

	// Method 2: Run git lfs ls-files
	cmd := exec.CommandContext(ctx, "git", "lfs", "ls-files")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// git lfs might not be installed or no LFS files exist
		return false
	}

	// If any LFS files exist, the output will not be empty
	hasLFS := len(bytes.TrimSpace(output)) > 0
	if hasLFS {
		a.logger.Debug("LFS detected via git lfs ls-files", "repo_path", repoPath)
	}
	return hasLFS
}

// detectSubmodules checks for Git submodules using two methods:
// 1. Check for .gitmodules file
// 2. Run "git submodule" command
func (a *Analyzer) detectSubmodules(ctx context.Context, repoPath string) bool {
	// Method 1: Check for .gitmodules file
	gitmodulesPath := repoPath + "/.gitmodules"
	if _, err := os.Stat(gitmodulesPath); err == nil {
		a.logger.Debug("Submodules detected via .gitmodules", "repo_path", repoPath)
		return true
	}

	// Method 2: Run git submodule command
	cmd := exec.CommandContext(ctx, "git", "submodule", "status")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// No submodules or error
		return false
	}

	// If any submodules exist, the output will not be empty
	hasSubmodules := len(bytes.TrimSpace(output)) > 0
	if hasSubmodules {
		a.logger.Debug("Submodules detected via git submodule", "repo_path", repoPath)
	}
	return hasSubmodules
}

// getBranchCount returns the number of branches in the repository
func (a *Analyzer) getBranchCount(ctx context.Context, repoPath string) int {
	cmd := exec.CommandContext(ctx, "git", "branch", "-r")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		a.logger.Warn("Failed to get branch count", "error", err)
		return 0
	}

	// Count non-empty lines (each line is a branch)
	lines := bytes.Split(output, []byte("\n"))
	count := 0
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) > 0 {
			count++
		}
	}

	return count
}

// getDiskSize returns the actual disk size of the repository using du command
func (a *Analyzer) getDiskSize(ctx context.Context, repoPath string) (int64, error) {
	// Use du -sb for size in bytes
	cmd := exec.CommandContext(ctx, "du", "-sb", repoPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("du command failed: %w", err)
	}

	// Parse output: "size<tab>path"
	parts := bytes.Fields(output)
	if len(parts) < 1 {
		return 0, fmt.Errorf("unexpected du output format")
	}

	// Parse size
	sizeStr := string(parts[0])
	size := int64(0)
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	}

	return size, nil
}

// extractCommitSHA extracts the commit SHA from git-sizer output
// Format can be: "f7a25d8bede5b581accd6abe89cad8cc1c4b6d8d" or "f7a25d8... (refs/heads/main)"
func (a *Analyzer) extractCommitSHA(commitInfo string) string {
	// Take first 40 characters (full SHA) or until first space
	if idx := strings.Index(commitInfo, " "); idx > 0 {
		return commitInfo[:idx]
	}
	// Limit to 40 chars for full SHA
	if len(commitInfo) > 40 {
		return commitInfo[:40]
	}
	return commitInfo
}

// extractFilenameFromBlobInfo extracts the filename from git-sizer blob info
// Format: "319b802f686f9d80d4d2e7e62d1ccea8eea87766 (9ae8b638196a3ff9ec70b1b556db104c42e3365c:IMPLEMENTATION_GUIDE.md)"
func (a *Analyzer) extractFilenameFromBlobInfo(blobInfo string) string {
	// Find the filename after the colon
	if idx := strings.LastIndex(blobInfo, ":"); idx > 0 {
		// Extract between : and )
		filename := blobInfo[idx+1:]
		if endIdx := strings.Index(filename, ")"); endIdx > 0 {
			return filename[:endIdx]
		}
		return filename
	}
	return ""
}

// CheckRepositoryProblems identifies potential migration issues using git-sizer output
func (a *Analyzer) CheckRepositoryProblems(output *GitSizerOutput) []string {
	var problems []string

	// Based on git-sizer's "level of concern" thresholds
	// Reference: https://github.com/github/git-sizer

	// Very large commits (>100MB) - GitHub has a 100MB limit
	if output.MaxCommitSize > 100*1024*1024 {
		problems = append(problems,
			fmt.Sprintf("Commit exceeds GitHub limit: %d MB (limit: 100 MB)", output.MaxCommitSize/(1024*1024)))
	}

	// Very large blobs (>50MB)
	if output.MaxBlobSize > 50*1024*1024 {
		problems = append(problems,
			fmt.Sprintf("Very large file detected: %d MB", output.MaxBlobSize/(1024*1024)))
	}

	// Extremely large repositories (>5GB)
	totalSize := output.UniqueBlobSize + output.UniqueTreeSize + output.UniqueCommitSize
	if totalSize > 5*1024*1024*1024 {
		problems = append(problems,
			fmt.Sprintf("Very large repository: %d GB", totalSize/(1024*1024*1024)))
	}

	// Very deep history (>100k commits)
	if output.MaxHistoryDepth > 100000 {
		problems = append(problems,
			fmt.Sprintf("Very deep history: %d commits", output.MaxHistoryDepth))
	}

	// Extremely large trees (>10k entries)
	if output.MaxTreeEntries > 10000 {
		problems = append(problems,
			fmt.Sprintf("Very large directory: %d entries", output.MaxTreeEntries))
	}

	// Extremely large checkouts (>100k files)
	if output.MaxExpandedBlobCount > 100000 {
		problems = append(problems,
			fmt.Sprintf("Very large checkout: %d files", output.MaxExpandedBlobCount))
	}

	return problems
}
