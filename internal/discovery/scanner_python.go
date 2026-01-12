package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parseRequirementsFiles parses a list of requirements.txt files and extracts dependencies
func (ps *PackageScanner) parseRequirementsFiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*5)
	for _, reqPath := range files {
		relPath, _ := filepath.Rel(repoPath, reqPath)
		extracted := ps.parseRequirementsTxt(reqPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseRequirementsTxt parses requirements.txt and extracts GitHub dependencies
func (ps *PackageScanner) parseRequirementsTxt(reqPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- reqPath is validated via findFiles
	file, err := os.Open(reqPath)
	if err != nil {
		return deps
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if dep := ps.parsePythonRequirementLine(line, manifestPath); dep != nil {
			deps = append(deps, *dep)
		}
	}

	return deps
}

// parsePythonRequirementLine parses a single line from requirements.txt
func (ps *PackageScanner) parsePythonRequirementLine(line, manifestPath string) *ExtractedDependency {
	// Pattern for GitHub references: git+https://github.com/owner/repo.git@tag
	// Note: Allow dots in repo names (e.g., my-lib.backup), handle .git suffix via TrimSuffix
	gitPattern := regexp.MustCompile(`git\+(?:https|ssh)://(?:git@)?github\.com/([^/]+)/([^/@#]+)`)
	if matches := gitPattern.FindStringSubmatch(line); len(matches) == 3 {
		repoName := strings.TrimSuffix(matches[2], ".git")
		return &ExtractedDependency{
			Name:         extractPythonPkgName(line, repoName),
			Version:      line,
			Ecosystem:    EcosystemPython,
			Manifest:     manifestPath,
			IsGitHubRepo: true,
			GitHubOwner:  matches[1],
			GitHubRepo:   repoName,
			IsLocal:      false,
			SourceHost:   hostGitHubCom,
		}
	}

	// Check for Azure DevOps URLs
	if isADOURL(line) {
		org, project, repo, host, isLocal := ps.extractADOReference(line)
		if org != "" && repo != "" {
			return &ExtractedDependency{
				Name:         extractPythonPkgName(line, repo),
				Version:      line,
				Ecosystem:    EcosystemPython,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  org + "/" + project,
				GitHubRepo:   repo,
				IsLocal:      isLocal,
				SourceHost:   host,
			}
		}
	}

	// Check for tracked hosts (including source host for local dependencies)
	return ps.parsePythonEnterpriseHost(line, manifestPath)
}

// parsePythonEnterpriseHost checks for GitHub Enterprise hosts in requirements
func (ps *PackageScanner) parsePythonEnterpriseHost(line, manifestPath string) *ExtractedDependency {
	for _, host := range ps.additionalHosts {
		if host == hostGitHubCom {
			continue // Already checked
		}
		escapedHost := regexp.QuoteMeta(host)
		// Note: Allow dots in repo names (e.g., my-lib.backup)
		hostPattern := regexp.MustCompile(`git\+(?:https|ssh)://(?:git@)?` + escapedHost + `/([^/]+)/([^/@#]+)`)
		if matches := hostPattern.FindStringSubmatch(line); len(matches) == 3 {
			isLocal := ps.sourceHost != "" && host == ps.sourceHost
			repoName := strings.TrimSuffix(matches[2], ".git")
			return &ExtractedDependency{
				Name:         extractPythonPkgName(line, repoName),
				Version:      line,
				Ecosystem:    EcosystemPython,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  matches[1],
				GitHubRepo:   repoName,
				IsLocal:      isLocal,
				SourceHost:   host,
			}
		}
	}
	return nil
}

// extractPythonPkgName extracts package name from #egg= or uses fallback
func extractPythonPkgName(line, fallback string) string {
	if _, after, ok := strings.Cut(line, "#egg="); ok {
		pkgName := after
		return strings.Split(pkgName, "&")[0]
	}
	return fallback
}
