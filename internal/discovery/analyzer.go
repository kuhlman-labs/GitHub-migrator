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
// Based on: https://github.com/github/git-sizer
type GitSizerOutput struct {
	UniqueCommitCount   int64 `json:"unique_commit_count"`
	UniqueCommitSize    int64 `json:"unique_commit_size"`
	UniqueTreeCount     int64 `json:"unique_tree_count"`
	UniqueTreeSize      int64 `json:"unique_tree_size"`
	UniqueBlobCount     int64 `json:"unique_blob_count"`
	UniqueBlobSize      int64 `json:"unique_blob_size"`
	UniqueTagCount      int64 `json:"unique_tag_count"`
	MaxCommitSize       int64 `json:"max_commit_size"`
	MaxTreeEntries      int64 `json:"max_tree_entries"`
	MaxBlobSize         int64 `json:"max_blob_size"`
	MaxHistoryDepth     int64 `json:"max_history_depth"`
	MaxTagDepth         int64 `json:"max_tag_depth"`
	MaxPathDepth        int64 `json:"max_path_depth"`
	MaxPathLength       int64 `json:"max_path_length"`
	MaxDirectoryCount   int64 `json:"max_directory_count"`
	MaxFileCount        int64 `json:"max_file_count"`
	MaxExpandedTreeSize int64 `json:"max_expanded_tree_size"`
	MaxSymlinkCount     int64 `json:"max_symlink_count"`
	MaxSubmoduleCount   int64 `json:"max_submodule_count"`
}

// AnalyzeGitProperties analyzes Git repository using git-sizer and additional detection methods
func (a *Analyzer) AnalyzeGitProperties(ctx context.Context, repo *models.Repository, repoPath string) error {
	a.logger.Debug("Analyzing Git properties",
		"repo", repo.FullName,
		"path", repoPath)

	// Run git-sizer with JSON output
	output, err := a.runGitSizer(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("git-sizer failed: %w", err)
	}

	// Map git-sizer output to repository model
	totalSize := output.UniqueBlobSize + output.UniqueTreeSize + output.UniqueCommitSize
	repo.TotalSize = &totalSize
	largestFileSize := output.MaxBlobSize
	repo.LargestFileSize = &largestFileSize
	largestCommitSize := output.MaxCommitSize
	repo.LargestCommitSize = &largestCommitSize
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
		"largest_file", repo.LargestFileSize,
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

	// Parse JSON output
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

// CheckRepositoryProblems identifies potential migration issues using git-sizer output
func (a *Analyzer) CheckRepositoryProblems(output *GitSizerOutput) []string {
	var problems []string

	// Based on git-sizer's "level of concern" thresholds
	// Reference: https://github.com/github/git-sizer

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
	if output.MaxFileCount > 100000 {
		problems = append(problems,
			fmt.Sprintf("Very large checkout: %d files", output.MaxFileCount))
	}

	return problems
}
