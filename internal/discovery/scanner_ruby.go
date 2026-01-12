package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parseGemfiles parses a list of Gemfile files and extracts dependencies
func (ps *PackageScanner) parseGemfiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*5)
	for _, gemPath := range files {
		relPath, _ := filepath.Rel(repoPath, gemPath)
		extracted := ps.parseGemfile(gemPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseGemfile parses a Gemfile and extracts GitHub dependencies
// Supports:
//   - gem 'name', github: 'owner/repo'
//   - gem 'name', git: 'https://github.com/owner/repo.git'
//   - gem 'name', git: 'git@github.com:owner/repo.git'
//   - gem 'name', git: 'https://github.example.com/owner/repo.git' (local)
func (ps *PackageScanner) parseGemfile(gemPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- gemPath is validated via findFiles
	file, err := os.Open(gemPath)
	if err != nil {
		return deps
	}
	defer func() { _ = file.Close() }()

	// Pattern for gem name extraction
	gemNamePattern := regexp.MustCompile(`gem\s+['"]([^'"]+)['"]`)

	// Pattern for github: shorthand - gem 'name', github: 'owner/repo'
	githubShortPattern := regexp.MustCompile(`github:\s*['"]([^/'"]+)/([^'"]+)['"]`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract gem name
		gemNameMatch := gemNamePattern.FindStringSubmatch(line)
		if len(gemNameMatch) < 2 {
			continue
		}
		gemName := gemNameMatch[1]

		if dep := ps.parseGemfileLine(line, gemName, manifestPath, githubShortPattern); dep != nil {
			deps = append(deps, *dep)
		}
	}

	return deps
}

// parseGemfileLine parses a single Gemfile line and returns an ExtractedDependency if a git dependency is found
func (ps *PackageScanner) parseGemfileLine(line, gemName, manifestPath string, githubShortPattern *regexp.Regexp) *ExtractedDependency {
	// Check for github: shorthand (always github.com)
	// Shorthand always implies github.com - check if that's the source instance
	if matches := githubShortPattern.FindStringSubmatch(line); len(matches) == 3 {
		isLocal := ps.sourceHost != "" && ps.sourceHost == hostGitHubCom
		return &ExtractedDependency{
			Name:         gemName,
			Version:      line,
			Ecosystem:    EcosystemRuby,
			Manifest:     manifestPath,
			IsGitHubRepo: true,
			GitHubOwner:  matches[1],
			GitHubRepo:   matches[2],
			IsLocal:      isLocal,
			SourceHost:   hostGitHubCom,
		}
	}

	// Check for Azure DevOps URLs
	if isADOURL(line) {
		org, project, repo, host, isLocal := ps.extractADOReference(line)
		if org != "" && repo != "" {
			return &ExtractedDependency{
				Name:         gemName,
				Version:      line,
				Ecosystem:    EcosystemRuby,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  org + "/" + project,
				GitHubRepo:   repo,
				IsLocal:      isLocal,
				SourceHost:   host,
			}
		}
	}

	// Check for git: with tracked hosts
	return ps.parseGemfileGitURL(line, gemName, manifestPath)
}

// parseGemfileGitURL extracts git dependencies from Gemfile lines with explicit git URLs
func (ps *PackageScanner) parseGemfileGitURL(line, gemName, manifestPath string) *ExtractedDependency {
	for _, host := range ps.additionalHosts {
		escapedHost := regexp.QuoteMeta(host)

		// Pattern: git: 'https://host/owner/repo.git'
		// Note: Allow dots in repo names (e.g., my-lib.backup), trim .git suffix separately
		httpsPattern := regexp.MustCompile(`git:\s*['"]https://` + escapedHost + `/([^/]+)/([^/'"]+)`)
		if matches := httpsPattern.FindStringSubmatch(line); len(matches) == 3 {
			isLocal := ps.sourceHost != "" && host == ps.sourceHost
			return &ExtractedDependency{
				Name:         gemName,
				Version:      line,
				Ecosystem:    EcosystemRuby,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  matches[1],
				GitHubRepo:   strings.TrimSuffix(matches[2], ".git"),
				IsLocal:      isLocal,
				SourceHost:   host,
			}
		}

		// Pattern: git: 'git@host:owner/repo.git'
		// Note: Allow dots in repo names (e.g., my-lib.backup), trim .git suffix separately
		sshPattern := regexp.MustCompile(`git:\s*['"]git@` + escapedHost + `:([^/]+)/([^/'"]+)`)
		if matches := sshPattern.FindStringSubmatch(line); len(matches) == 3 {
			isLocal := ps.sourceHost != "" && host == ps.sourceHost
			return &ExtractedDependency{
				Name:         gemName,
				Version:      line,
				Ecosystem:    EcosystemRuby,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  matches[1],
				GitHubRepo:   strings.TrimSuffix(matches[2], ".git"),
				IsLocal:      isLocal,
				SourceHost:   host,
			}
		}
	}
	return nil
}
