package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"gopkg.in/yaml.v3"
)

// DependencyAnalyzer analyzes repository dependencies
type DependencyAnalyzer struct {
	logger *slog.Logger
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(logger *slog.Logger) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		logger: logger,
	}
}

// SubmoduleInfo contains details about a Git submodule
type SubmoduleInfo struct {
	Path   string
	URL    string
	Branch string
}

// WorkflowDependency contains details about a GitHub Actions workflow dependency
type WorkflowDependency struct {
	WorkflowFile string
	Uses         string
	Repository   string // owner/repo format
	Ref          string
	WorkflowPath string
}

// ExtractSubmodules parses .gitmodules file to extract submodule information
func (da *DependencyAnalyzer) ExtractSubmodules(ctx context.Context, repoPath string) ([]SubmoduleInfo, error) {
	// Validate repository path to prevent path traversal
	if err := source.ValidateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repository path: %w", err)
	}

	gitmodulesPath := filepath.Join(repoPath, ".gitmodules")

	// Check if .gitmodules exists
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return nil, nil
	}

	// #nosec G304 -- repoPath is validated via ValidateRepoPath above
	content, err := os.ReadFile(gitmodulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .gitmodules: %w", err)
	}

	var submodules []SubmoduleInfo
	lines := strings.Split(string(content), "\n")
	var currentSubmodule *SubmoduleInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// New submodule section
		if strings.HasPrefix(line, "[submodule ") {
			if currentSubmodule != nil {
				submodules = append(submodules, *currentSubmodule)
			}
			currentSubmodule = &SubmoduleInfo{}
			continue
		}

		// Parse path
		if strings.HasPrefix(line, "path = ") && currentSubmodule != nil {
			currentSubmodule.Path = strings.TrimPrefix(line, "path = ")
		}

		// Parse URL
		if strings.HasPrefix(line, "url = ") && currentSubmodule != nil {
			currentSubmodule.URL = strings.TrimPrefix(line, "url = ")
		}

		// Parse branch
		if strings.HasPrefix(line, "branch = ") && currentSubmodule != nil {
			currentSubmodule.Branch = strings.TrimPrefix(line, "branch = ")
		}
	}

	// Add the last submodule
	if currentSubmodule != nil && currentSubmodule.URL != "" {
		submodules = append(submodules, *currentSubmodule)
	}

	da.logger.Debug("Extracted submodules", "repo_path", repoPath, "count", len(submodules))
	return submodules, nil
}

// ExtractGitHubRepoFromURL extracts owner/repo from a GitHub URL
func ExtractGitHubRepoFromURL(url string) (string, error) {
	// Handle various GitHub URL formats:
	// https://github.com/owner/repo
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// git://github.com/owner/repo.git

	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Pattern for HTTPS URLs
	httpsPattern := regexp.MustCompile(`https?://[^/]*github[^/]*/([^/]+)/([^/]+)`)
	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2]), nil
	}

	// Pattern for SSH URLs (git@github.com:owner/repo)
	sshPattern := regexp.MustCompile(`git@[^:]*github[^:]*:([^/]+)/([^/]+)`)
	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2]), nil
	}

	// Pattern for git:// URLs
	gitPattern := regexp.MustCompile(`git://[^/]*github[^/]*/([^/]+)/([^/]+)`)
	if matches := gitPattern.FindStringSubmatch(url); len(matches) == 3 {
		return fmt.Sprintf("%s/%s", matches[1], matches[2]), nil
	}

	return "", fmt.Errorf("unable to extract GitHub repo from URL: %s", url)
}

// ExtractWorkflowDependencies scans GitHub Actions workflow files for reusable workflow dependencies
func (da *DependencyAnalyzer) ExtractWorkflowDependencies(ctx context.Context, repoPath string) ([]WorkflowDependency, error) {
	// Validate repository path to prevent path traversal
	if err := source.ValidateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repository path: %w", err)
	}

	workflowsDir := filepath.Join(repoPath, ".github", "workflows")

	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var dependencies []WorkflowDependency

	// Read all workflow files - path is validated and constructed safely
	files, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only process .yml and .yaml files
		ext := filepath.Ext(file.Name())
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		filePath := filepath.Join(workflowsDir, file.Name())
		deps, err := da.parseWorkflowFile(filePath, file.Name())
		if err != nil {
			da.logger.Warn("Failed to parse workflow file", "file", file.Name(), "error", err)
			continue
		}

		dependencies = append(dependencies, deps...)
	}

	da.logger.Debug("Extracted workflow dependencies", "repo_path", repoPath, "count", len(dependencies))
	return dependencies, nil
}

// parseWorkflowFile parses a single workflow file and extracts reusable workflow dependencies
// nolint:gocyclo // Workflow parsing inherently requires multiple nested checks
func (da *DependencyAnalyzer) parseWorkflowFile(filePath, fileName string) ([]WorkflowDependency, error) {
	content, err := da.readWorkflowFile(filePath)
	if err != nil {
		return nil, err
	}

	workflow, err := da.parseWorkflowYAML(content)
	if err != nil {
		return nil, err
	}

	jobs, ok := workflow["jobs"].(map[string]any)
	if !ok {
		return nil, nil
	}

	return da.extractDependenciesFromJobs(jobs, fileName), nil
}

// readWorkflowFile reads a workflow file from disk
func (da *DependencyAnalyzer) readWorkflowFile(filePath string) ([]byte, error) {
	// Validate file path to prevent path traversal
	if err := source.ValidateRepoPath(filePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// #nosec G304 -- filePath is validated via ValidateRepoPath above
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}
	return content, nil
}

// parseWorkflowYAML parses workflow YAML content
func (da *DependencyAnalyzer) parseWorkflowYAML(content []byte) (map[string]any, error) {
	var workflow map[string]any
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}
	return workflow, nil
}

// extractDependenciesFromJobs extracts dependencies from workflow jobs
func (da *DependencyAnalyzer) extractDependenciesFromJobs(jobs map[string]any, fileName string) []WorkflowDependency {
	var dependencies []WorkflowDependency

	for _, job := range jobs {
		jobMap, ok := job.(map[string]any)
		if !ok {
			continue
		}

		// Check for "uses" key (reusable workflow)
		if uses, ok := jobMap["uses"].(string); ok {
			if dep := da.parseUsesString(uses, fileName); dep != nil {
				dependencies = append(dependencies, *dep)
			}
		}

		// Check steps for action dependencies
		dependencies = append(dependencies, da.extractDependenciesFromSteps(jobMap, fileName)...)
	}

	return dependencies
}

// extractDependenciesFromSteps extracts dependencies from job steps
func (da *DependencyAnalyzer) extractDependenciesFromSteps(jobMap map[string]any, fileName string) []WorkflowDependency {
	var dependencies []WorkflowDependency

	steps, ok := jobMap["steps"].([]any)
	if !ok {
		return dependencies
	}

	for _, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}

		uses, ok := stepMap["uses"].(string)
		if !ok {
			continue
		}

		// Only include repository references, not Docker or local actions
		if da.isRepositoryReference(uses) {
			if dep := da.parseUsesString(uses, fileName); dep != nil {
				dependencies = append(dependencies, *dep)
			}
		}
	}

	return dependencies
}

// isRepositoryReference checks if a uses string references a repository
func (da *DependencyAnalyzer) isRepositoryReference(uses string) bool {
	return strings.Contains(uses, "/") &&
		!strings.HasPrefix(uses, "docker://") &&
		!strings.HasPrefix(uses, "./")
}

// parseUsesString parses a GitHub Actions "uses" string
// Format: owner/repo/path/to/workflow.yml@ref or owner/repo@ref
func (da *DependencyAnalyzer) parseUsesString(uses, workflowFile string) *WorkflowDependency {
	// Split by @ to get ref
	parts := strings.SplitN(uses, "@", 2)
	if len(parts) != 2 {
		return nil
	}

	repoPath := parts[0]
	ref := parts[1]

	// Extract owner/repo
	// Format can be: owner/repo or owner/repo/.github/workflows/workflow.yml
	pathParts := strings.SplitN(repoPath, "/", 3)
	if len(pathParts) < 2 {
		return nil
	}

	owner := pathParts[0]
	repo := pathParts[1]
	fullRepo := fmt.Sprintf("%s/%s", owner, repo)

	workflowPath := ""
	if len(pathParts) == 3 {
		workflowPath = pathParts[2]
	}

	return &WorkflowDependency{
		WorkflowFile: workflowFile,
		Uses:         uses,
		Repository:   fullRepo,
		Ref:          ref,
		WorkflowPath: workflowPath,
	}
}

// AnalyzeDependencies performs complete dependency analysis on a repository
// Returns a list of RepositoryDependency objects ready to be saved
//
// Detection priority (all file-based, source-agnostic):
// 1. Package manager files (PRIMARY) - npm, Go, Python, Maven, Gradle, .NET, Ruby, Rust, PHP, Terraform, Helm, Docker
// 2. Git submodules (.gitmodules)
// 3. GitHub Actions workflows (.github/workflows/)
//
// The sourceURL parameter is used to identify local dependencies (dependencies hosted on the source instance)
func (da *DependencyAnalyzer) AnalyzeDependencies(ctx context.Context, repoPath, repoFullName string, repoID int64, sourceURL string) ([]*models.RepositoryDependency, error) {
	var dependencies []*models.RepositoryDependency
	now := time.Now()

	// 1. PRIMARY: Scan package manager files (source-agnostic)
	// This is the main dependency detection mechanism that works consistently
	// across all source systems (GitHub, Azure DevOps, GitLab, etc.)
	packageScanner := NewPackageScanner(da.logger).WithSourceURL(sourceURL)
	packageDeps, err := packageScanner.ScanPackageManagers(ctx, repoPath, repoID)
	if err != nil {
		da.logger.Warn("Failed to scan package managers", "repo", repoFullName, "error", err)
	} else {
		dependencies = append(dependencies, packageDeps...)
		da.logger.Debug("Package scan complete",
			"repo", repoFullName,
			"package_manifests", len(packageDeps))
	}

	// 2. Extract submodules
	submodules, err := da.ExtractSubmodules(ctx, repoPath)
	if err != nil {
		da.logger.Warn("Failed to extract submodules", "repo", repoFullName, "error", err)
	} else {
		for _, sub := range submodules {
			// Try to extract GitHub repo from URL
			depFullName, err := ExtractGitHubRepoFromURL(sub.URL)
			if err != nil {
				// Not a GitHub repo, skip it (external dependency)
				da.logger.Debug("Skipping non-GitHub submodule", "url", sub.URL)
				continue
			}

			// Create metadata JSON
			metadata := map[string]any{
				"path":   sub.Path,
				"url":    sub.URL,
				"branch": sub.Branch,
			}
			metadataJSON, _ := json.Marshal(metadata)
			metadataStr := string(metadataJSON)

			dep := &models.RepositoryDependency{
				RepositoryID:       repoID,
				DependencyFullName: depFullName,
				DependencyType:     models.DependencyTypeSubmodule,
				DependencyURL:      sub.URL,
				IsLocal:            false, // Will be updated later
				DiscoveredAt:       now,
				Metadata:           &metadataStr,
			}
			dependencies = append(dependencies, dep)
		}
	}

	// 3. Extract workflow dependencies (GitHub Actions specific, but works from cloned files)
	workflowDeps, err := da.ExtractWorkflowDependencies(ctx, repoPath)
	if err != nil {
		da.logger.Warn("Failed to extract workflow dependencies", "repo", repoFullName, "error", err)
	} else {
		// Deduplicate workflow dependencies
		seenDeps := make(map[string]bool)
		for _, wf := range workflowDeps {
			key := fmt.Sprintf("%s@%s", wf.Repository, wf.Ref)
			if seenDeps[key] {
				continue
			}
			seenDeps[key] = true

			// Create metadata JSON
			metadata := map[string]any{
				"workflow_file": wf.WorkflowFile,
				"uses":          wf.Uses,
				"ref":           wf.Ref,
				"workflow_path": wf.WorkflowPath,
			}
			metadataJSON, _ := json.Marshal(metadata)
			metadataStr := string(metadataJSON)

			dep := &models.RepositoryDependency{
				RepositoryID:       repoID,
				DependencyFullName: wf.Repository,
				DependencyType:     models.DependencyTypeWorkflow,
				DependencyURL:      fmt.Sprintf("https://github.com/%s", wf.Repository),
				IsLocal:            false, // Will be updated later
				DiscoveredAt:       now,
				Metadata:           &metadataStr,
			}
			dependencies = append(dependencies, dep)
		}
	}

	da.logger.Info("Dependency analysis complete",
		"repo", repoFullName,
		"total_dependencies", len(dependencies))

	return dependencies, nil
}
