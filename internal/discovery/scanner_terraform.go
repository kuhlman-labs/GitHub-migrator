package discovery

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parseTerraformFiles parses a list of .tf files and extracts dependencies
func (ps *PackageScanner) parseTerraformFiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
	for _, tfPath := range files {
		relPath, _ := filepath.Rel(repoPath, tfPath)
		extracted := ps.parseTerraformFile(tfPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseTerraformFile parses a .tf file and extracts GitHub module sources
// Supports:
//   - module "name" { source = "github.com/owner/repo" }
//   - module "name" { source = "github.com/owner/repo//subdir" }
//   - module "name" { source = "git::https://github.com/owner/repo.git" }
//   - module "name" { source = "git::https://github.example.com/org/repo.git" }
//   - module "name" { source = "git@github.com:owner/repo.git" }
func (ps *PackageScanner) parseTerraformFile(tfPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- tfPath is validated via findFilesWithPattern
	content, err := os.ReadFile(tfPath)
	if err != nil {
		return deps
	}

	// Pattern to match module blocks and extract source
	// module "name" {
	//   source = "..."
	// }
	modulePattern := regexp.MustCompile(`module\s+"[^"]+"\s*\{[^}]*source\s*=\s*"([^"]+)"`)
	matches := modulePattern.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		source := match[1]

		// Skip local paths
		if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") || strings.HasPrefix(source, "/") {
			continue
		}

		// Skip Terraform Registry modules (format: namespace/name/provider)
		if ps.isTerraformRegistryModule(source) {
			continue
		}

		// Check for Azure DevOps URLs first
		if isADOURL(source) {
			org, project, repo, host, isLocal := ps.extractADOReference(source)
			if org != "" && repo != "" {
				deps = append(deps, ExtractedDependency{
					Name:         source,
					Version:      source,
					Ecosystem:    EcosystemTerraform,
					Manifest:     manifestPath,
					IsGitHubRepo: true,
					GitHubOwner:  org + "/" + project,
					GitHubRepo:   repo,
					IsLocal:      isLocal,
					SourceHost:   host,
				})
				continue
			}
		}

		// Try to extract GitHub reference
		owner, repo, host, isLocal := ps.extractGitHubFromTerraformSource(source)
		if owner != "" && repo != "" {
			deps = append(deps, ExtractedDependency{
				Name:         source,
				Version:      source,
				Ecosystem:    EcosystemTerraform,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  owner,
				GitHubRepo:   repo,
				IsLocal:      isLocal,
				SourceHost:   host,
			})
		}
	}

	return deps
}

// isTerraformRegistryModule checks if a source string is a Terraform Registry module
// Registry modules have format: namespace/name/provider (e.g., hashicorp/consul/aws)
func (ps *PackageScanner) isTerraformRegistryModule(source string) bool {
	// Skip if it contains git:: prefix or URL schemes
	if strings.HasPrefix(source, "git::") || strings.Contains(source, "://") {
		return false
	}

	// Registry modules have exactly 2 slashes: namespace/name/provider
	parts := strings.Split(source, "/")

	// If it has exactly 3 parts and no dots in the first part, it's likely a registry module
	if len(parts) == 3 && !strings.Contains(parts[0], ".") {
		return true
	}

	return false
}

// extractGitHubFromTerraformSource extracts GitHub owner/repo from Terraform module source strings
// Supports:
//   - github.com/owner/repo
//   - github.com/owner/repo//subdir
//   - git::https://github.com/owner/repo.git
//   - git::https://github.com/owner/repo.git//subdir
//   - git::https://github.com/owner/repo.git?ref=v1.0.0
//   - git@github.com:owner/repo.git
func (ps *PackageScanner) extractGitHubFromTerraformSource(source string) (owner, repo, host string, isLocal bool) {
	// Remove git:: prefix if present
	cleanSource := strings.TrimPrefix(source, "git::")

	// Check for each tracked host
	for _, h := range ps.additionalHosts {
		escapedHost := regexp.QuoteMeta(h)

		// Pattern: host/owner/repo or host/owner/repo//subdir or host/owner/repo?ref=xxx
		directPattern := regexp.MustCompile(`^` + escapedHost + `/([^/]+)/([^/?#]+)`)
		if matches := directPattern.FindStringSubmatch(cleanSource); len(matches) == 3 {
			isLocalDep := ps.sourceHost != "" && h == ps.sourceHost
			return matches[1], strings.TrimSuffix(matches[2], ".git"), h, isLocalDep
		}

		// Pattern: https://host/owner/repo.git or https://host/owner/repo
		httpsPattern := regexp.MustCompile(`https://` + escapedHost + `/([^/]+)/([^/?#]+)`)
		if matches := httpsPattern.FindStringSubmatch(cleanSource); len(matches) == 3 {
			isLocalDep := ps.sourceHost != "" && h == ps.sourceHost
			return matches[1], strings.TrimSuffix(matches[2], ".git"), h, isLocalDep
		}

		// Pattern: git@host:owner/repo.git
		sshPattern := regexp.MustCompile(`git@` + escapedHost + `:([^/]+)/([^/?#]+)`)
		if matches := sshPattern.FindStringSubmatch(cleanSource); len(matches) == 3 {
			isLocalDep := ps.sourceHost != "" && h == ps.sourceHost
			return matches[1], strings.TrimSuffix(matches[2], ".git"), h, isLocalDep
		}
	}

	return "", "", "", false
}
