package discovery

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// parseChartYamlFiles parses a list of Chart.yaml files and extracts dependencies
func (ps *PackageScanner) parseChartYamlFiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*2)
	for _, chartPath := range files {
		relPath, _ := filepath.Rel(repoPath, chartPath)
		// Pattern: repository: "https://host/owner/repo" or repository: "git+https://..."
		extracted := ps.parseFileWithURLPattern(chartPath, relPath, EcosystemHelm,
			`repository:\s*["']?(?:git\+)?https://`, `/([^/]+)/([^/"'\s]+)`)
		deps = append(deps, extracted...)

		// Also check for ADO URLs
		adoDeps := ps.parseFileForADOURLs(chartPath, relPath, EcosystemHelm)
		deps = append(deps, adoDeps...)
	}
	return deps
}

// parsePackageSwiftFiles parses a list of Package.swift files and extracts dependencies
func (ps *PackageScanner) parsePackageSwiftFiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*2)
	for _, swiftPath := range files {
		relPath, _ := filepath.Rel(repoPath, swiftPath)
		// Pattern: .package(url: "https://host/owner/repo"
		extracted := ps.parseFileWithURLPattern(swiftPath, relPath, EcosystemSwift,
			`\.package\s*\(\s*url:\s*"https://`, `/([^/]+)/([^/"]+)"`)
		deps = append(deps, extracted...)

		// Also check for ADO URLs
		adoDeps := ps.parseFileForADOURLs(swiftPath, relPath, EcosystemSwift)
		deps = append(deps, adoDeps...)
	}
	return deps
}

// parseMixExsFiles parses a list of mix.exs files and extracts dependencies
func (ps *PackageScanner) parseMixExsFiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*2)
	for _, mixPath := range files {
		relPath, _ := filepath.Rel(repoPath, mixPath)
		extracted := ps.parseMixExs(mixPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseMixExs parses mix.exs and extracts GitHub dependencies
// Supports:
//   - {:dep, github: "owner/repo"}
//   - {:dep, github: "owner/repo", branch: "main"}
//   - {:dep, git: "https://github.com/owner/repo.git"}
func (ps *PackageScanner) parseMixExs(mixPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- mixPath is validated via findFiles
	content, err := os.ReadFile(mixPath)
	if err != nil {
		return deps
	}

	// Pattern for github: shorthand (always github.com)
	// Shorthand always implies github.com - check if that's the source instance
	githubPattern := regexp.MustCompile(`github:\s*"([^/]+)/([^"]+)"`)
	githubMatches := githubPattern.FindAllStringSubmatch(string(content), -1)
	isLocalGitHub := ps.sourceHost != "" && ps.sourceHost == hostGitHubCom
	for _, match := range githubMatches {
		if len(match) == 3 {
			deps = append(deps, ExtractedDependency{
				Name:         match[1] + "/" + match[2],
				Version:      match[0],
				Ecosystem:    EcosystemElixir,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  match[1],
				GitHubRepo:   match[2],
				IsLocal:      isLocalGitHub,
				SourceHost:   hostGitHubCom,
			})
		}
	}

	// Check for Azure DevOps URLs in git: patterns
	adoDeps := ps.parseFileForADOURLs(mixPath, manifestPath, EcosystemElixir)
	deps = append(deps, adoDeps...)

	// Pattern for git: with tracked hosts (GitHub-style: host/owner/repo)
	// Note: github.com is NOT skipped here because the github: shorthand pattern above
	// only matches `github: "owner/repo"` syntax, not `git: "https://github.com/owner/repo.git"` URLs.
	// Both formats are valid and need to be handled.
	// Note: ADO hosts are skipped because they use a different URL format (org/project/_git/repo)
	// and are already handled by parseFileForADOURLs above.
	for _, host := range ps.additionalHosts {
		// Skip ADO hosts - they don't follow the host/owner/repo pattern
		// and are already handled by parseFileForADOURLs above
		if isADOHost(host) {
			continue
		}

		escapedHost := regexp.QuoteMeta(host)

		gitPattern := regexp.MustCompile(`git:\s*"https://` + escapedHost + `/([^/]+)/([^"]+)"`)
		matches := gitPattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) == 3 {
				isLocal := ps.sourceHost != "" && host == ps.sourceHost
				deps = append(deps, ExtractedDependency{
					Name:         match[1] + "/" + strings.TrimSuffix(match[2], ".git"),
					Version:      match[0],
					Ecosystem:    EcosystemElixir,
					Manifest:     manifestPath,
					IsGitHubRepo: true,
					GitHubOwner:  match[1],
					GitHubRepo:   strings.TrimSuffix(match[2], ".git"),
					IsLocal:      isLocal,
					SourceHost:   host,
				})
			}
		}
	}

	return deps
}

// parseBuildGradleFiles parses a list of build.gradle/build.gradle.kts files and extracts dependencies
func (ps *PackageScanner) parseBuildGradleFiles(files []string, repoPath string) []ExtractedDependency {
	deps := make([]ExtractedDependency, 0, len(files)*2)
	for _, gradlePath := range files {
		relPath, _ := filepath.Rel(repoPath, gradlePath)
		extracted := ps.parseBuildGradle(gradlePath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseBuildGradle parses build.gradle and extracts JitPack GitHub dependencies
// JitPack format: com.github.Owner:Repo:Tag
func (ps *PackageScanner) parseBuildGradle(gradlePath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- gradlePath is validated via findFiles
	content, err := os.ReadFile(gradlePath)
	if err != nil {
		return deps
	}

	// Pattern for JitPack dependencies: com.github.Owner:Repo:Tag
	// Matches: implementation 'com.github.Owner:Repo:v1.0'
	// Matches: implementation "com.github.Owner:Repo:v1.0"
	// JitPack always uses github.com - check if that's the source instance
	jitpackPattern := regexp.MustCompile(`['"]com\.github\.([^:]+):([^:'"]+)(?::[^'"]+)?['"]`)
	matches := jitpackPattern.FindAllStringSubmatch(string(content), -1)
	isLocalJitPack := ps.sourceHost != "" && ps.sourceHost == hostGitHubCom
	for _, match := range matches {
		if len(match) == 3 {
			deps = append(deps, ExtractedDependency{
				Name:         "com.github." + match[1] + ":" + match[2],
				Version:      match[0],
				Ecosystem:    EcosystemGradle,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  match[1],
				GitHubRepo:   match[2],
				IsLocal:      isLocalJitPack,
				SourceHost:   hostGitHubCom,
			})
		}
	}

	return deps
}
