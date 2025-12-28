package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// analyzeDependencies analyzes repository dependencies and saves them to the database
//
// Detection priority:
// 1. PRIMARY: File-based scanning from cloned repository (source-agnostic)
//   - Package manager files (npm, Go, Python, Maven, Gradle, .NET, Ruby, Rust, PHP, Terraform, Helm, Docker)
//   - Git submodules (.gitmodules)
//   - GitHub Actions workflows (.github/workflows/)
//
// 2. FALLBACK: GitHub dependency graph API (GitHub sources only, supplements file scanning)
func (c *Collector) analyzeDependencies(ctx context.Context, repo *models.Repository, repoPath string, profiler *Profiler) error {
	c.logger.Debug("Analyzing dependencies", "repo", repo.FullName)

	// Create dependency analyzer
	depAnalyzer := NewDependencyAnalyzer(c.logger)

	// 1. PRIMARY: Analyze dependencies from cloned repo (source-agnostic file scanning)
	// This includes: package manager files, submodules, and workflow dependencies
	dependencies, err := depAnalyzer.AnalyzeDependencies(ctx, repoPath, repo.FullName, repo.ID, repo.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies from repo: %w", err)
	}

	// Count file-based package dependencies for logging
	fileScanCount := len(dependencies)

	// 2. FALLBACK: Fetch dependency graph from GitHub API (GitHub sources only)
	// This supplements file scanning with additional dependency data that GitHub may have
	// detected through its dependency graph analysis (e.g., transitive dependencies)
	if profiler != nil && c.shouldFetchDependencyGraph(repo, dependencies) {
		parts := strings.Split(repo.FullName, "/")
		if len(parts) == 2 {
			owner, repoName := parts[0], parts[1]
			manifests, err := profiler.client.GetDependencyGraph(ctx, owner, repoName)
			if err != nil {
				c.logger.Debug("Failed to fetch dependency graph (fallback)",
					"repo", repo.FullName,
					"error", err)
				// Not a fatal error - file scanning is primary, continue with what we have
			} else {
				// Process and merge dependency graph data, avoiding duplicates
				depGraphDeps, stats := c.processDependencyGraph(manifests, repo.ID)
				dependencies = c.mergeDependencies(dependencies, depGraphDeps)

				// Log at appropriate level based on results
				if stats.TotalDependencies > 0 && stats.GitHubRepoDeps == 0 {
					// All dependencies were filtered - this is expected for registry packages
					// Log at INFO level since this is useful context, not an error
					c.logger.Info("Dependency graph returned only registry packages (no GitHub repo linkage)",
						"repo", repo.FullName,
						"manifests_fetched", len(manifests),
						"total_packages", stats.TotalDependencies,
						"note", "Registry packages (npm, PyPI, etc.) don't include GitHub repository info in the API response")
				} else {
					c.logger.Info("Dependency graph processed",
						"repo", repo.FullName,
						"manifests_fetched", len(manifests),
						"total_in_graph", stats.TotalDependencies,
						"external_packages_filtered", stats.ExternalPackages,
						"github_repo_deps", stats.GitHubRepoDeps,
						"duplicates_filtered", stats.DuplicatesFiltered,
						"graph_additions", len(dependencies)-fileScanCount)
				}
			}
		}
	}

	// Save dependencies to database
	if len(dependencies) > 0 {
		if err := c.storage.SaveRepositoryDependencies(ctx, repo.ID, dependencies); err != nil {
			return fmt.Errorf("failed to save dependencies: %w", err)
		}

		c.logger.Info("Dependencies saved",
			"repo", repo.FullName,
			"count", len(dependencies),
			"from_file_scan", fileScanCount)
	}

	return nil
}

// shouldFetchDependencyGraph determines if we should call the GitHub dependency graph API
// This is a fallback mechanism - we always prefer file scanning but can supplement with API data
func (c *Collector) shouldFetchDependencyGraph(repo *models.Repository, existingDeps []*models.RepositoryDependency) bool {
	// Only fetch for GitHub sources
	if repo.Source != "ghes" && repo.Source != "ghec" && repo.Source != "github" {
		return false
	}

	// Always fetch as a supplement - GitHub's dependency graph may have additional
	// information about transitive dependencies or dependencies we couldn't detect
	// from file scanning alone
	return true
}

// mergeDependencies merges dependency graph dependencies with existing file-scanned dependencies
// It avoids duplicates by checking both dependency full name AND dependency type.
// This allows the same repository to appear multiple times with different dependency types
// (e.g., as both a workflow dependency and a submodule).
func (c *Collector) mergeDependencies(existing, graphDeps []*models.RepositoryDependency) []*models.RepositoryDependency {
	// Build a set of existing dependency identifiers using composite key (name + type)
	seen := make(map[string]bool)
	for _, dep := range existing {
		// Use composite key: full name + dependency type
		// This preserves entries where the same repo is referenced through different mechanisms
		key := dep.DependencyFullName + ":" + dep.DependencyType
		seen[key] = true
	}

	// Add graph dependencies that don't already exist (same name AND same type)
	for _, dep := range graphDeps {
		key := dep.DependencyFullName + ":" + dep.DependencyType
		if !seen[key] {
			existing = append(existing, dep)
			seen[key] = true
		}
	}

	return existing
}

// DependencyGraphStats tracks statistics from processing the dependency graph
type DependencyGraphStats struct {
	TotalDependencies  int // Total dependencies in the graph
	ExternalPackages   int // External packages (npm, PyPI, etc.) - filtered out
	GitHubRepoDeps     int // Dependencies that are GitHub repositories - kept
	DuplicatesFiltered int // Duplicates that were filtered out
}

// processDependencyGraph processes dependency graph manifests and extracts repository dependencies
// This is used as a FALLBACK to supplement file-based scanning
// Returns both the dependencies and statistics about what was processed/filtered
//
// NOTE: The GitHub Dependency Graph API only populates RepositoryOwner/RepositoryName for
// dependencies that GitHub can definitively link to a GitHub repository. For most packages
// from registries (npm, PyPI, Maven, etc.), this linkage doesn't exist in the API response,
// even if the package source code is hosted on GitHub. As a result, most dependencies from
// the API will be filtered as "external packages" and only direct GitHub repo references
// (like Go modules with github.com/... paths) will be captured.
//
// For migration planning, this is acceptable since we primarily care about repositories
// that directly reference other repositories via GitHub URLs in their manifest files.
// The file-based PackageScanner is more effective at detecting these references.
func (c *Collector) processDependencyGraph(manifests []*github.DependencyGraphManifest, repoID int64) ([]*models.RepositoryDependency, DependencyGraphStats) {
	var dependencies []*models.RepositoryDependency
	seen := make(map[string]bool)
	now := time.Now()
	stats := DependencyGraphStats{}

	for _, manifest := range manifests {
		for _, dep := range manifest.Dependencies {
			stats.TotalDependencies++

			// Only include GitHub repository dependencies, not external packages
			if dep.RepositoryOwner == nil || dep.RepositoryName == nil {
				stats.ExternalPackages++
				continue
			}

			depFullName := fmt.Sprintf("%s/%s", *dep.RepositoryOwner, *dep.RepositoryName)

			// Deduplicate
			if seen[depFullName] {
				stats.DuplicatesFiltered++
				continue
			}
			seen[depFullName] = true
			stats.GitHubRepoDeps++

			// Create metadata JSON with source marker to indicate this came from API fallback
			metadataMap := map[string]any{
				"source":          "github_api",
				"manifest":        manifest.Filename,
				"package_name":    dep.PackageName,
				"package_manager": dep.PackageManager,
				"requirements":    dep.Requirements,
			}
			metadataBytes, _ := json.Marshal(metadataMap)
			metadataStr := string(metadataBytes)

			dependency := &models.RepositoryDependency{
				RepositoryID:       repoID,
				DependencyFullName: depFullName,
				DependencyType:     models.DependencyTypeDependencyGraph,
				DependencyURL:      fmt.Sprintf("https://github.com/%s", depFullName),
				IsLocal:            false, // Will be updated later
				DiscoveredAt:       now,
				Metadata:           &metadataStr,
			}
			dependencies = append(dependencies, dependency)
		}
	}

	return dependencies, stats
}
