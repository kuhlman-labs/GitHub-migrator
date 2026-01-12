package discovery

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parsePackageJSONFiles parses a list of package.json files and extracts dependencies
func (ps *PackageScanner) parsePackageJSONFiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*2)
	for _, pkgPath := range files {
		relPath, _ := filepath.Rel(repoPath, pkgPath)
		extracted := ps.parsePackageJSON(pkgPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parsePackageJSON parses package.json and extracts GitHub dependencies
func (ps *PackageScanner) parsePackageJSON(pkgPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- pkgPath is validated via findFiles
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return deps
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(content, &pkg); err != nil {
		return deps
	}

	// Check all dependencies for GitHub references
	allDeps := make(map[string]string)
	maps.Copy(allDeps, pkg.Dependencies)
	maps.Copy(allDeps, pkg.DevDependencies)

	for name, version := range allDeps {
		owner, repo, host, isLocal := ps.extractGitHubFromNpmVersion(version)
		if owner != "" && repo != "" {
			deps = append(deps, ExtractedDependency{
				Name:         name,
				Version:      version,
				Ecosystem:    EcosystemNodeJS,
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

// extractGitHubFromNpmVersion extracts GitHub owner/repo and host from npm version strings
// Supports: github:owner/repo, git+https://github.com/owner/repo, git+https://host/owner/repo, owner/repo
func (ps *PackageScanner) extractGitHubFromNpmVersion(version string) (owner, repo, host string, isLocal bool) {
	// Pattern: github:owner/repo or github:owner/repo#tag
	// Shorthand always implies github.com - check if that's the source instance
	if owner, repo := ps.extractNpmGitHubShorthand(version); owner != "" {
		isLocal := ps.sourceHost != "" && ps.sourceHost == hostGitHubCom
		return owner, repo, hostGitHubCom, isLocal
	}

	// Check for git+https:// or https:// patterns with any tracked host
	if owner, repo, host, isLocal := ps.extractNpmGitURL(version); owner != "" {
		return owner, repo, host, isLocal
	}

	// Pattern: owner/repo (shorthand) - assume github.com for shorthand
	// Shorthand always implies github.com - check if that's the source instance
	if owner, repo := ps.extractNpmOwnerRepoShorthand(version); owner != "" {
		isLocal := ps.sourceHost != "" && ps.sourceHost == hostGitHubCom
		return owner, repo, hostGitHubCom, isLocal
	}

	return "", "", "", false
}

// extractNpmGitHubShorthand extracts owner/repo from github: shorthand
func (ps *PackageScanner) extractNpmGitHubShorthand(version string) (owner, repo string) {
	if !strings.HasPrefix(version, "github:") {
		return "", ""
	}
	ref := strings.TrimPrefix(version, "github:")
	ref = strings.Split(ref, "#")[0] // Remove tag/branch
	parts := strings.Split(ref, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// extractNpmGitURL extracts owner/repo from git+https:// or https:// URLs
func (ps *PackageScanner) extractNpmGitURL(version string) (owner, repo, host string, isLocal bool) {
	for _, h := range ps.additionalHosts {
		escapedHost := regexp.QuoteMeta(h)

		// Pattern: git+https://host/owner/repo.git
		// Note: Allow dots in repo names (e.g., my-lib.backup), trim .git suffix separately
		gitPattern := regexp.MustCompile(`git\+https://` + escapedHost + `/([^/]+)/([^/?#]+)`)
		if matches := gitPattern.FindStringSubmatch(version); len(matches) == 3 {
			isLocalDep := ps.sourceHost != "" && h == ps.sourceHost
			return matches[1], strings.TrimSuffix(matches[2], ".git"), h, isLocalDep
		}

		// Pattern: https://host/owner/repo
		// Note: Allow dots in repo names (e.g., my-lib.backup)
		httpsPattern := regexp.MustCompile(`https://` + escapedHost + `/([^/]+)/([^/?#]+)`)
		if matches := httpsPattern.FindStringSubmatch(version); len(matches) == 3 {
			isLocalDep := ps.sourceHost != "" && h == ps.sourceHost
			return matches[1], strings.TrimSuffix(matches[2], ".git"), h, isLocalDep
		}
	}
	return "", "", "", false
}

// extractNpmOwnerRepoShorthand extracts owner/repo from shorthand format
func (ps *PackageScanner) extractNpmOwnerRepoShorthand(version string) (owner, repo string) {
	// Must have exactly one slash, no colons, and not start with version prefixes
	if strings.Count(version, "/") != 1 || strings.Contains(version, ":") {
		return "", ""
	}
	if strings.HasPrefix(version, "^") || strings.HasPrefix(version, "~") {
		return "", ""
	}

	parts := strings.Split(version, "/")
	// Validate it looks like owner/repo (not a scoped package like @scope/pkg)
	if len(parts) != 2 || strings.HasPrefix(parts[0], "@") || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return "", ""
	}

	// Additional check: version shouldn't start with numbers (semver)
	if regexp.MustCompile(`^\d`).MatchString(version) {
		return "", ""
	}

	return parts[0], strings.Split(parts[1], "#")[0]
}
