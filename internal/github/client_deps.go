package github

import (
	"context"

	"github.com/shurcooL/githubv4"
)

// DependencyGraphManifest represents a dependency manifest from the dependency graph
type DependencyGraphManifest struct {
	Filename     string
	Dependencies []DependencyGraphDependency
}

// DependencyGraphDependency represents a single dependency
type DependencyGraphDependency struct {
	PackageName     string
	PackageManager  string
	Requirements    string
	RepositoryName  *string // For GitHub repository dependencies
	RepositoryOwner *string // For GitHub repository dependencies
}

// GetDependencyGraph fetches the dependency graph for a repository using GraphQL API
// This includes both manifest dependencies and dependent repositories
// The function paginates through both manifests and dependencies within each manifest.
func (c *Client) GetDependencyGraph(ctx context.Context, owner, repo string) ([]*DependencyGraphManifest, error) {
	c.logger.Info("Fetching dependency graph", "owner", owner, "repo", repo)

	var manifests []*DependencyGraphManifest
	var manifestCursor *githubv4.String
	totalDependencies := 0

	// GraphQL query for dependency graph - fetches manifests with first page of dependencies
	var query struct {
		Repository struct {
			DependencyGraphManifests struct {
				Nodes []struct {
					Filename     githubv4.String
					Dependencies struct {
						TotalCount githubv4.Int
						Nodes      []struct {
							PackageName    githubv4.String
							PackageManager githubv4.String
							Requirements   githubv4.String
							Repository     *struct {
								Name  githubv4.String
								Owner struct {
									Login githubv4.String
								}
							}
						}
						PageInfo struct {
							HasNextPage githubv4.Boolean
							EndCursor   githubv4.String
						}
					} `graphql:"dependencies(first: 100)"`
				}
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"dependencyGraphManifests(first: 10, after: $cursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for {
		variables := map[string]any{
			"owner":  githubv4.String(owner),
			"name":   githubv4.String(repo),
			"cursor": manifestCursor,
		}

		err := c.QueryWithRetry(ctx, "GetDependencyGraph", &query, variables)
		if err != nil {
			// If dependency graph is not enabled or permission denied, return empty result
			// This is not an error - just means the feature isn't available
			c.logger.Debug("Dependency graph not available", "owner", owner, "repo", repo, "error", err)
			return manifests, nil
		}

		// Collect manifests and their dependencies
		for _, node := range query.Repository.DependencyGraphManifests.Nodes {
			manifest := &DependencyGraphManifest{
				Filename:     string(node.Filename),
				Dependencies: []DependencyGraphDependency{},
			}

			// Add first page of dependencies
			for _, dep := range node.Dependencies.Nodes {
				dependency := DependencyGraphDependency{
					PackageName:    string(dep.PackageName),
					PackageManager: string(dep.PackageManager),
					Requirements:   string(dep.Requirements),
				}

				// If this is a GitHub repository dependency, extract repo info
				if dep.Repository != nil {
					// Heap-allocate strings to ensure they survive loop iterations
					dependency.RepositoryName = newStr(string(dep.Repository.Name))
					dependency.RepositoryOwner = newStr(string(dep.Repository.Owner.Login))
				}

				manifest.Dependencies = append(manifest.Dependencies, dependency)
			}

			// If there are more dependencies, paginate through them
			if node.Dependencies.PageInfo.HasNextPage {
				// Explicitly heap-allocate the cursor to ensure it survives function calls
				depCursor := newString(node.Dependencies.PageInfo.EndCursor)
				additionalDeps, err := c.paginateManifestDependencies(ctx, owner, repo, string(node.Filename), depCursor)
				if err != nil {
					c.logger.Debug("Failed to paginate manifest dependencies",
						"manifest", string(node.Filename),
						"error", err)
					// Continue with what we have
				} else {
					manifest.Dependencies = append(manifest.Dependencies, additionalDeps...)
				}
			}

			totalDependencies += len(manifest.Dependencies)
			manifests = append(manifests, manifest)
		}

		if !query.Repository.DependencyGraphManifests.PageInfo.HasNextPage {
			break
		}
		// Explicitly heap-allocate the cursor to ensure it survives loop iterations
		manifestCursor = newString(query.Repository.DependencyGraphManifests.PageInfo.EndCursor)
	}

	c.logger.Info("Dependency graph fetched",
		"owner", owner,
		"repo", repo,
		"manifests", len(manifests),
		"total_dependencies", totalDependencies)

	return manifests, nil
}

// paginateManifestDependencies fetches additional dependencies for a specific manifest
func (c *Client) paginateManifestDependencies(ctx context.Context, owner, repo, filename string, startCursor *githubv4.String) ([]DependencyGraphDependency, error) {
	var dependencies []DependencyGraphDependency
	depCursor := startCursor

	// Unfortunately GitHub's GraphQL API doesn't support filtering manifests by filename directly
	// We need to paginate through all manifests to find the one we need each time
	// Using same page size as GetDependencyGraph (first: 10) for consistency
	var manifestQuery struct {
		Repository struct {
			DependencyGraphManifests struct {
				Nodes []struct {
					Filename     githubv4.String
					Dependencies struct {
						Nodes []struct {
							PackageName    githubv4.String
							PackageManager githubv4.String
							Requirements   githubv4.String
							Repository     *struct {
								Name  githubv4.String
								Owner struct {
									Login githubv4.String
								}
							}
						}
						PageInfo struct {
							HasNextPage githubv4.Boolean
							EndCursor   githubv4.String
						}
					} `graphql:"dependencies(first: 100, after: $depCursor)"`
				}
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"dependencyGraphManifests(first: 10, after: $manifestCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	// Outer loop: paginate through dependency pages for the target manifest
	for depCursor != nil {
		// Inner loop: paginate through manifests to find the target one
		// Always start from the beginning for reliability - cursor caching across
		// different queries with different parameters is unreliable
		var manifestCursor *githubv4.String
		found := false

		for {
			variables := map[string]any{
				"owner":          githubv4.String(owner),
				"name":           githubv4.String(repo),
				"depCursor":      depCursor,
				"manifestCursor": manifestCursor,
			}

			err := c.QueryWithRetry(ctx, "GetManifestDependencies", &manifestQuery, variables)
			if err != nil {
				return dependencies, err
			}

			// Search for the manifest in this page
			for _, node := range manifestQuery.Repository.DependencyGraphManifests.Nodes {
				if string(node.Filename) == filename {
					found = true

					for _, dep := range node.Dependencies.Nodes {
						dependency := DependencyGraphDependency{
							PackageName:    string(dep.PackageName),
							PackageManager: string(dep.PackageManager),
							Requirements:   string(dep.Requirements),
						}

						if dep.Repository != nil {
							// Heap-allocate strings to ensure they survive loop iterations
							dependency.RepositoryName = newStr(string(dep.Repository.Name))
							dependency.RepositoryOwner = newStr(string(dep.Repository.Owner.Login))
						}

						dependencies = append(dependencies, dependency)
					}

					// Update depCursor for next iteration (or nil if done)
					if node.Dependencies.PageInfo.HasNextPage {
						// Explicitly heap-allocate the cursor to ensure it survives loop iterations
						depCursor = newString(node.Dependencies.PageInfo.EndCursor)
					} else {
						depCursor = nil
					}
					break
				}
			}

			if found {
				break // Found the manifest, exit inner loop
			}

			// Check if there are more manifest pages to search
			if !manifestQuery.Repository.DependencyGraphManifests.PageInfo.HasNextPage {
				// Exhausted all manifests without finding the target
				c.logger.Debug("Manifest not found for dependency pagination", "filename", filename)
				return dependencies, nil
			}

			// Move to next page of manifests
			// Explicitly heap-allocate the cursor to ensure it survives loop iterations
			manifestCursor = newString(manifestQuery.Repository.DependencyGraphManifests.PageInfo.EndCursor)
		}
	}

	return dependencies, nil
}

// newString returns a pointer to a heap-allocated copy of the given githubv4.String.
// This ensures the string value survives beyond the current scope and prevents
// dangling pointer issues when reusing query structs across GraphQL calls.
func newString(s githubv4.String) *githubv4.String {
	ptr := new(githubv4.String)
	*ptr = s
	return ptr
}

// newStr returns a pointer to a heap-allocated copy of the given string.
// This ensures the string value survives beyond the current scope and prevents
// dangling pointer issues when storing strings from loop iterations.
func newStr(s string) *string {
	ptr := new(string)
	*ptr = s
	return ptr
}
