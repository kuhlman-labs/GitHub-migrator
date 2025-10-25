package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/embedded"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

const (
	// LargeFileThreshold is the size threshold for detecting large files (100MB)
	// Files larger than this may cause issues during migration
	LargeFileThreshold = 100 * 1024 * 1024
)

// Repository Analysis Strategy:
// 1. Use `git count-objects -vH` for accurate repository size calculation
//    - Provides size of loose objects and packfiles
//    - More accurate for Git-specific sizing than disk usage
// 2. Use `git-sizer` for detailed Git statistics and analysis
//    - Commit counts, largest files, tree entries, history depth
//    - Identifies potential migration problems
//    - Provides detailed blob and tree analysis

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

// GitSizerMetric represents a single metric in git-sizer JSON output (version 2)
type GitSizerMetric struct {
	Value             int64  `json:"value"`
	ObjectName        string `json:"objectName,omitempty"`
	ObjectDescription string `json:"objectDescription,omitempty"`
}

// GitSizerOutput represents the JSON output from git-sizer (json-version=2)
// Based on actual git-sizer output format: https://github.com/github/git-sizer
// Each field is a nested object with value, unit, and other metadata
type GitSizerOutput struct {
	UniqueCommitCount         GitSizerMetric `json:"uniqueCommitCount"`
	UniqueCommitSize          GitSizerMetric `json:"uniqueCommitSize"`
	UniqueTreeCount           GitSizerMetric `json:"uniqueTreeCount"`
	UniqueTreeSize            GitSizerMetric `json:"uniqueTreeSize"`
	UniqueBlobCount           GitSizerMetric `json:"uniqueBlobCount"`
	UniqueBlobSize            GitSizerMetric `json:"uniqueBlobSize"`
	UniqueTagCount            GitSizerMetric `json:"uniqueTagCount"`
	MaxCommitSize             GitSizerMetric `json:"maxCommitSize"`
	MaxTreeEntries            GitSizerMetric `json:"maxTreeEntries"`
	MaxBlobSize               GitSizerMetric `json:"maxBlobSize"`
	MaxHistoryDepth           GitSizerMetric `json:"maxHistoryDepth"`
	MaxTagDepth               GitSizerMetric `json:"maxTagDepth"`
	MaxCheckoutPathDepth      GitSizerMetric `json:"maxCheckoutPathDepth"`
	MaxCheckoutPathLength     GitSizerMetric `json:"maxCheckoutPathLength"`
	MaxCheckoutTreeCount      GitSizerMetric `json:"maxCheckoutTreeCount"`
	MaxCheckoutBlobCount      GitSizerMetric `json:"maxCheckoutBlobCount"`
	MaxCheckoutBlobSize       GitSizerMetric `json:"maxCheckoutBlobSize"`
	MaxCheckoutLinkCount      GitSizerMetric `json:"maxCheckoutLinkCount"`
	MaxCheckoutSubmoduleCount GitSizerMetric `json:"maxCheckoutSubmoduleCount"`
}

// AnalyzeGitProperties analyzes Git repository using git-sizer and additional detection methods
func (a *Analyzer) AnalyzeGitProperties(ctx context.Context, repo *models.Repository, repoPath string) error {
	a.logger.Debug("Analyzing Git properties",
		"repo", repo.FullName,
		"path", repoPath)

	// Get repository size using git count-objects
	gitSize, err := a.getGitObjectSize(ctx, repoPath)
	if err != nil {
		a.logger.Warn("Failed to get git object size, will use git-sizer estimate",
			"repo", repo.FullName,
			"error", err)
		// Fall back to git-sizer estimate if git count-objects fails
		gitSize = 0
	}

	// Run git-sizer with JSON output
	output, err := a.runGitSizer(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("git-sizer failed: %w", err)
	}

	// Use git count-objects size if available, otherwise use git-sizer calculation
	if gitSize > 0 {
		repo.TotalSize = &gitSize
	} else {
		totalSize := output.UniqueBlobSize.Value + output.UniqueTreeSize.Value + output.UniqueCommitSize.Value
		repo.TotalSize = &totalSize
	}

	// Store largest file info
	largestFileSize := output.MaxBlobSize.Value
	repo.LargestFileSize = &largestFileSize
	if output.MaxBlobSize.ObjectDescription != "" {
		// Extract filename from blob info (format: "hash (commit:path)")
		filename := a.extractFilenameFromBlobInfo(output.MaxBlobSize.ObjectDescription)
		if filename != "" {
			repo.LargestFile = &filename
		}
	}

	// Store largest commit info
	largestCommitSize := output.MaxCommitSize.Value
	repo.LargestCommitSize = &largestCommitSize
	if output.MaxCommitSize.ObjectName != "" {
		// Store just the commit SHA
		commitSHA := a.extractCommitSHA(output.MaxCommitSize.ObjectName)
		repo.LargestCommit = &commitSHA
	}

	repo.CommitCount = int(output.UniqueCommitCount.Value)

	// Detect large files (>100MB) that may cause migration issues
	if output.MaxBlobSize.Value > LargeFileThreshold {
		repo.HasLargeFiles = true
		// git-sizer only gives us the max blob size, not a count of large files
		// We set count to 1 to indicate at least one large file exists
		repo.LargeFileCount = 1
		a.logger.Warn("Large file detected in repository",
			"repo", repo.FullName,
			"size_mb", output.MaxBlobSize.Value/(1024*1024),
			"file", repo.LargestFile)
	}

	// Detect LFS using .gitattributes and git lfs ls-files
	repo.HasLFS = a.detectLFS(ctx, repoPath)

	// Detect submodules using .gitmodules file and git submodule command
	repo.HasSubmodules = a.detectSubmodules(ctx, repoPath)

	// Get branch count
	repo.BranchCount = a.getBranchCount(ctx, repoPath)

	// Get last commit SHA from default branch
	if lastCommitSHA := a.getLastCommitSHA(ctx, repoPath); lastCommitSHA != "" {
		repo.LastCommitSHA = &lastCommitSHA
	}

	// Get tag count
	repo.TagCount = a.getTagCount(ctx, repoPath)

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
// Uses the embedded git-sizer binary for portability
func (a *Analyzer) runGitSizer(ctx context.Context, repoPath string) (*GitSizerOutput, error) {
	// Get the path to the embedded git-sizer binary
	gitSizerPath, err := embedded.GetGitSizerPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get git-sizer binary: %w", err)
	}

	cmd := exec.CommandContext(ctx, gitSizerPath, "--json", "--json-version=2")
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

// detectLFS checks for Git LFS usage using four methods:
// 1. Check .gitattributes for "filter=lfs"
// 2. Run "git lfs ls-files" to list LFS-tracked files
// 3. Check .git/config for [filter "lfs"] configuration
// 4. Check for LFS pointer files in the repository
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
	if err == nil {
		// If any LFS files exist, the output will not be empty
		hasLFS := len(bytes.TrimSpace(output)) > 0
		if hasLFS {
			a.logger.Debug("LFS detected via git lfs ls-files", "repo_path", repoPath)
			return true
		}
	}

	// Method 3: Check .git/config for LFS filter configuration
	gitConfigPath := filepath.Join(repoPath, ".git", "config")
	// #nosec G304 -- repoPath is a controlled temporary directory path
	if data, err := os.ReadFile(gitConfigPath); err == nil {
		if strings.Contains(string(data), "[filter \"lfs\"]") {
			a.logger.Debug("LFS detected via .git/config", "repo_path", repoPath)
			return true
		}
	}

	// Method 4: Check for LFS pointer files
	// LFS pointer files contain "version https://git-lfs.github.com/spec/"
	if a.detectLFSPointerFiles(ctx, repoPath) {
		a.logger.Debug("LFS detected via pointer files", "repo_path", repoPath)
		return true
	}

	return false
}

// detectLFSPointerFiles checks for Git LFS pointer files by searching for
// files that contain the LFS pointer format
func (a *Analyzer) detectLFSPointerFiles(ctx context.Context, repoPath string) bool {
	// Use git grep to search for LFS pointer file pattern
	// LFS pointer files start with "version https://git-lfs.github.com/spec/"
	cmd := exec.CommandContext(ctx, "git", "grep", "-q", "version https://git-lfs.github.com/spec/")
	cmd.Dir = repoPath

	// git grep -q returns exit code 0 if pattern is found, 1 if not found
	err := cmd.Run()
	return err == nil
}

// detectSubmodules checks for Git submodules using three methods:
// 1. Check for .gitmodules file
// 2. Run "git submodule status" command
// 3. Check .git/config for [submodule sections
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
	if err == nil {
		// If any submodules exist, the output will not be empty
		hasSubmodules := len(bytes.TrimSpace(output)) > 0
		if hasSubmodules {
			a.logger.Debug("Submodules detected via git submodule", "repo_path", repoPath)
			return true
		}
	}

	// Method 3: Check .git/config for submodule configuration
	gitConfigPath := filepath.Join(repoPath, ".git", "config")
	// #nosec G304 -- repoPath is a controlled temporary directory path
	if data, err := os.ReadFile(gitConfigPath); err == nil {
		if strings.Contains(string(data), "[submodule") {
			a.logger.Debug("Submodules detected via .git/config", "repo_path", repoPath)
			return true
		}
	}

	return false
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

// getLastCommitSHA returns the SHA of the last commit on the default branch
func (a *Analyzer) getLastCommitSHA(ctx context.Context, repoPath string) string {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		a.logger.Debug("Failed to get last commit SHA", "error", err)
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getTagCount returns the number of tags in the repository
func (a *Analyzer) getTagCount(ctx context.Context, repoPath string) int {
	cmd := exec.CommandContext(ctx, "git", "tag", "--list")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		a.logger.Debug("Failed to get tag count", "error", err)
		return 0
	}

	// Count non-empty lines (each line is a tag)
	lines := bytes.Split(output, []byte("\n"))
	count := 0
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) > 0 {
			count++
		}
	}

	return count
}

// GitCountObjectsOutput represents parsed output from git count-objects -vH
type GitCountObjectsOutput struct {
	Count         int64 // Number of loose objects
	Size          int64 // Size of loose objects in bytes
	InPack        int64 // Number of objects in packs
	Packs         int64 // Number of packs
	SizePack      int64 // Size of packs in bytes
	PrunePackable int64 // Number of objects that could be pruned
	Garbage       int64 // Number of garbage files
	SizeGarbage   int64 // Size of garbage files in bytes
}

// getGitObjectSize returns the repository size using git count-objects -vH
// This provides Git-specific size information about loose objects and packfiles
func (a *Analyzer) getGitObjectSize(ctx context.Context, repoPath string) (int64, error) {
	cmd := exec.CommandContext(ctx, "git", "count-objects", "-vH")
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("git count-objects failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse the output
	output, err := a.parseGitCountObjects(stdout.String())
	if err != nil {
		return 0, fmt.Errorf("failed to parse git count-objects output: %w", err)
	}

	// Total size is loose objects + packed objects
	totalSize := output.Size + output.SizePack

	a.logger.Debug("Git object statistics",
		"repo_path", repoPath,
		"loose_objects", output.Count,
		"loose_size_bytes", output.Size,
		"packed_objects", output.InPack,
		"pack_count", output.Packs,
		"pack_size_bytes", output.SizePack,
		"total_size_bytes", totalSize)

	return totalSize, nil
}

// parseGitCountObjects parses the output of git count-objects -vH
// Example output:
// count: 391
// size: 1.84 MiB
// in-pack: 4
// packs: 1
// size-pack: 2.44 KiB
func (a *Analyzer) parseGitCountObjects(output string) (*GitCountObjectsOutput, error) {
	result := &GitCountObjectsOutput{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		a.parseCountObjectsLine(line, result)
	}

	return result, nil
}

// parseCountObjectsLine parses a single line from git count-objects output
func (a *Analyzer) parseCountObjectsLine(line string, result *GitCountObjectsOutput) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "count":
		result.Count, _ = a.parseInteger(value)
	case "size":
		result.Size, _ = a.parseHumanSize(value)
	case "in-pack":
		result.InPack, _ = a.parseInteger(value)
	case "packs":
		result.Packs, _ = a.parseInteger(value)
	case "size-pack":
		result.SizePack, _ = a.parseHumanSize(value)
	case "prune-packable":
		result.PrunePackable, _ = a.parseInteger(value)
	case "garbage":
		result.Garbage, _ = a.parseInteger(value)
	case "size-garbage":
		result.SizeGarbage, _ = a.parseHumanSize(value)
	}
}

// parseInteger parses an integer value from git count-objects output
func (a *Analyzer) parseInteger(value string) (int64, error) {
	var result int64
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return 0, err
	}
	return result, nil
}

// parseHumanSize parses human-readable size (e.g., "1.84 MiB", "2.44 KiB") to bytes
func (a *Analyzer) parseHumanSize(value string) (int64, error) {
	// Handle formats like "1.84 MiB", "2.44 KiB", "0 bytes"
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty size value")
	}

	// If it's just a number (bytes), parse directly
	if len(parts) == 1 || parts[1] == "bytes" {
		return a.parseInteger(parts[0])
	}

	// Parse the numeric value
	var numValue float64
	if _, err := fmt.Sscanf(parts[0], "%f", &numValue); err != nil {
		return 0, fmt.Errorf("failed to parse size number: %w", err)
	}

	// Parse the unit
	if len(parts) < 2 {
		return int64(numValue), nil
	}

	unit := strings.ToLower(parts[1])
	var multiplier int64

	switch unit {
	case "kib":
		multiplier = 1024
	case "mib":
		multiplier = 1024 * 1024
	case "gib":
		multiplier = 1024 * 1024 * 1024
	case "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "bytes", "byte":
		multiplier = 1
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}

	return int64(numValue * float64(multiplier)), nil
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
	if output.MaxCommitSize.Value > 100*1024*1024 {
		problems = append(problems,
			fmt.Sprintf("Commit exceeds GitHub limit: %d MB (limit: 100 MB)", output.MaxCommitSize.Value/(1024*1024)))
	}

	// Very large blobs (>50MB)
	if output.MaxBlobSize.Value > 50*1024*1024 {
		problems = append(problems,
			fmt.Sprintf("Very large file detected: %d MB", output.MaxBlobSize.Value/(1024*1024)))
	}

	// Extremely large repositories (>5GB)
	totalSize := output.UniqueBlobSize.Value + output.UniqueTreeSize.Value + output.UniqueCommitSize.Value
	if totalSize > 5*1024*1024*1024 {
		problems = append(problems,
			fmt.Sprintf("Very large repository: %d GB", totalSize/(1024*1024*1024)))
	}

	// Very deep history (>100k commits)
	if output.MaxHistoryDepth.Value > 100000 {
		problems = append(problems,
			fmt.Sprintf("Very deep history: %d commits", output.MaxHistoryDepth.Value))
	}

	// Extremely large trees (>10k entries)
	if output.MaxTreeEntries.Value > 10000 {
		problems = append(problems,
			fmt.Sprintf("Very large directory: %d entries", output.MaxTreeEntries.Value))
	}

	// Extremely large checkouts (>100k files)
	if output.MaxCheckoutBlobCount.Value > 100000 {
		problems = append(problems,
			fmt.Sprintf("Very large checkout: %d files", output.MaxCheckoutBlobCount.Value))
	}

	return problems
}
