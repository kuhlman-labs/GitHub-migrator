package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
)

// analyzeDependencies analyzes repository dependencies and saves them to the database
func (c *Collector) analyzeDependencies(ctx context.Context, repo *models.Repository, repoPath string, profiler *Profiler) error {
	c.logger.Debug("Analyzing dependencies", "repo", repo.FullName)

	// Create dependency analyzer
	depAnalyzer := NewDependencyAnalyzer(c.logger)

	// Analyze dependencies from cloned repo (submodules, workflows)
	dependencies, err := depAnalyzer.AnalyzeDependencies(ctx, repoPath, repo.FullName, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies from repo: %w", err)
	}

	// Fetch dependency graph from GitHub API if profiler is available
	if profiler != nil {
		parts := strings.Split(repo.FullName, "/")
		if len(parts) == 2 {
			owner, repoName := parts[0], parts[1]
			manifests, err := profiler.client.GetDependencyGraph(ctx, owner, repoName)
			if err != nil {
				c.logger.Debug("Failed to fetch dependency graph", "repo", repo.FullName, "error", err)
				// Not a fatal error - continue with what we have
			} else {
				// Process dependency graph manifests
				depGraphDeps := c.processDependencyGraph(manifests, repo.ID)
				dependencies = append(dependencies, depGraphDeps...)
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
			"count", len(dependencies))
	}

	return nil
}

// processDependencyGraph processes dependency graph manifests and extracts repository dependencies
func (c *Collector) processDependencyGraph(manifests []*github.DependencyGraphManifest, repoID int64) []*models.RepositoryDependency {
	var dependencies []*models.RepositoryDependency
	seen := make(map[string]bool)
	now := time.Now()

	for _, manifest := range manifests {
		for _, dep := range manifest.Dependencies {
			// Only include GitHub repository dependencies, not external packages
			if dep.RepositoryOwner == nil || dep.RepositoryName == nil {
				continue
			}

			depFullName := fmt.Sprintf("%s/%s", *dep.RepositoryOwner, *dep.RepositoryName)

			// Deduplicate
			if seen[depFullName] {
				continue
			}
			seen[depFullName] = true

			// Create metadata JSON
			metadataMap := map[string]interface{}{
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

	return dependencies
}
