package discovery

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parseCargoFiles parses a list of Cargo.toml files and extracts dependencies
func (ps *PackageScanner) parseCargoFiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*10)
	for _, cargoPath := range files {
		relPath, _ := filepath.Rel(repoPath, cargoPath)
		extracted := ps.parseCargoToml(cargoPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseCargoToml parses Cargo.toml and extracts GitHub/ADO dependencies
// Supports:
//   - dep = { git = "https://github.com/owner/repo" }
//   - dep = { git = "https://github.com/owner/repo", branch = "main" }
//   - dep = { git = "git@github.com:owner/repo.git" }
//   - dep = { git = "https://dev.azure.com/org/project/_git/repo" }
func (ps *PackageScanner) parseCargoToml(cargoPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- cargoPath is validated via findFiles
	content, err := os.ReadFile(cargoPath)
	if err != nil {
		return deps
	}

	// Extract all git URLs from Cargo.toml
	gitURLPattern := regexp.MustCompile(`git\s*=\s*"([^"]+)"`)
	matches := gitURLPattern.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		gitURL := match[1]

		// Check for ADO URL first
		if isADOURL(gitURL) {
			org, project, repo, host, isLocal := ps.extractADOReference(gitURL)
			if org != "" && repo != "" {
				deps = append(deps, ExtractedDependency{
					Name:         org + "/" + project + "/" + repo,
					Version:      match[0],
					Ecosystem:    EcosystemRust,
					Manifest:     manifestPath,
					IsGitHubRepo: true,
					GitHubOwner:  org + "/" + project, // ADO uses org/project as "owner"
					GitHubRepo:   repo,
					IsLocal:      isLocal,
					SourceHost:   host,
				})
			}
			continue
		}

		// Check for GitHub-style URLs
		for _, host := range ps.additionalHosts {
			escapedHost := regexp.QuoteMeta(host)

			// Pattern: https://host/owner/repo or https://host/owner/repo.git
			httpsPattern := regexp.MustCompile(`https://` + escapedHost + `/([^/]+)/([^/?#]+)`)
			if httpsMatches := httpsPattern.FindStringSubmatch(gitURL); len(httpsMatches) == 3 {
				isLocal := ps.sourceHost != "" && host == ps.sourceHost
				deps = append(deps, ExtractedDependency{
					Name:         httpsMatches[1] + "/" + strings.TrimSuffix(httpsMatches[2], ".git"),
					Version:      match[0],
					Ecosystem:    EcosystemRust,
					Manifest:     manifestPath,
					IsGitHubRepo: true,
					GitHubOwner:  httpsMatches[1],
					GitHubRepo:   strings.TrimSuffix(httpsMatches[2], ".git"),
					IsLocal:      isLocal,
					SourceHost:   host,
				})
				break
			}

			// Pattern: git@host:owner/repo.git
			sshPattern := regexp.MustCompile(`git@` + escapedHost + `:([^/]+)/([^/?#]+)`)
			if sshMatches := sshPattern.FindStringSubmatch(gitURL); len(sshMatches) == 3 {
				isLocal := ps.sourceHost != "" && host == ps.sourceHost
				deps = append(deps, ExtractedDependency{
					Name:         sshMatches[1] + "/" + strings.TrimSuffix(sshMatches[2], ".git"),
					Version:      match[0],
					Ecosystem:    EcosystemRust,
					Manifest:     manifestPath,
					IsGitHubRepo: true,
					GitHubOwner:  sshMatches[1],
					GitHubRepo:   strings.TrimSuffix(sshMatches[2], ".git"),
					IsLocal:      isLocal,
					SourceHost:   host,
				})
				break
			}
		}
	}

	return deps
}
