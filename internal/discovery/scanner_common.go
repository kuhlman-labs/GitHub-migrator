package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/ado"
)

// isSkippedDir checks if a directory should be skipped during scanning
func isSkippedDir(name string) bool {
	switch name {
	case dirNodeModules, dirVendor, dirGit, dirPycache, dirTerraform, dirTarget, dirBin, dirObj, dirPackages:
		return true
	default:
		return false
	}
}

// Manifest file names as constants
const (
	fileGoMod          = "go.mod"
	filePackageJSON    = "package.json"
	fileRequirements   = "requirements.txt"
	fileGemfile        = "Gemfile"
	fileCargoToml      = "Cargo.toml"
	fileChartYaml      = "Chart.yaml"
	filePackageSwift   = "Package.swift"
	fileMixExs         = "mix.exs"
	fileBuildGradle    = "build.gradle"
	fileBuildGradleKts = "build.gradle.kts"
)

// collectAllManifests performs a single directory walk to discover all manifest files
// This is much more efficient than calling findFiles/findFilesWithPattern for each type
func (ps *PackageScanner) collectAllManifests(repoPath string) *ManifestFiles {
	manifests := &ManifestFiles{}

	// #nosec G304 -- repoPath is validated before this function is called
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip common non-source directories
		if info.IsDir() {
			if isSkippedDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Categorize the file by name
		ps.categorizeManifestFile(info.Name(), path, manifests)
		return nil
	})

	if err != nil {
		ps.logger.Debug("Error walking directory for manifests", "path", repoPath, "error", err)
	}

	return manifests
}

// categorizeManifestFile adds a file to the appropriate manifest category based on its name
func (ps *PackageScanner) categorizeManifestFile(name, path string, manifests *ManifestFiles) {
	// Match exact filenames using a map-like approach
	switch name {
	case fileGoMod:
		manifests.GoMod = append(manifests.GoMod, path)
	case filePackageJSON:
		manifests.PackageJSON = append(manifests.PackageJSON, path)
	case fileRequirements:
		manifests.Requirements = append(manifests.Requirements, path)
	case fileGemfile:
		manifests.Gemfile = append(manifests.Gemfile, path)
	case fileCargoToml:
		manifests.CargoToml = append(manifests.CargoToml, path)
	case fileChartYaml:
		manifests.ChartYaml = append(manifests.ChartYaml, path)
	case filePackageSwift:
		manifests.PackageSwift = append(manifests.PackageSwift, path)
	case fileMixExs:
		manifests.MixExs = append(manifests.MixExs, path)
	case fileBuildGradle:
		manifests.BuildGradle = append(manifests.BuildGradle, path)
	case fileBuildGradleKts:
		manifests.BuildGradleKts = append(manifests.BuildGradleKts, path)
	}

	// Match pattern-based filenames
	ps.matchPatternManifests(name, path, manifests)
}

// matchPatternManifests matches files that require pattern matching (not exact name match)
func (ps *PackageScanner) matchPatternManifests(name, path string, manifests *ManifestFiles) {
	// requirements*.txt variants (excluding requirements.txt which is already matched above)
	if strings.HasPrefix(name, "requirements") && strings.HasSuffix(name, ".txt") && name != fileRequirements {
		manifests.RequirementsVariants = append(manifests.RequirementsVariants, path)
	}

	// *.tf files for Terraform
	if strings.HasSuffix(name, ".tf") {
		manifests.Terraform = append(manifests.Terraform, path)
	}
}

// parseFileWithURLPattern parses a file looking for URLs matching a specific pattern
// This is a generic parser used by multiple ecosystems (Helm, Swift, etc.)
func (ps *PackageScanner) parseFileWithURLPattern(filePath, manifestPath string, ecosystem PackageEcosystem, patternPrefix, patternSuffix string) []ExtractedDependency {
	var deps []ExtractedDependency

	// Build combined pattern for each host we're scanning
	for _, host := range ps.additionalHosts {
		// Skip ADO hosts - they use a different URL format (org/project/_git/repo)
		// and should be handled by parseFileForADOURLs instead
		if isADOHost(host) {
			continue
		}

		// Build full pattern: prefix + host + suffix
		fullPattern := patternPrefix + regexp.QuoteMeta(host) + patternSuffix
		re := regexp.MustCompile(fullPattern)

		// Process file in closure to properly handle defer
		func() {
			// #nosec G304 -- filePath is validated by caller
			file, err := os.Open(filePath)
			if err != nil {
				return
			}
			defer func() { _ = file.Close() }()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				matches := re.FindStringSubmatch(line)
				if len(matches) >= 3 {
					owner := matches[1]
					repo := strings.TrimSuffix(matches[2], ".git")

					// Check if this is a local dependency
					isLocal := host == ps.sourceHost

					deps = append(deps, ExtractedDependency{
						Name:         owner + "/" + repo,
						Ecosystem:    ecosystem,
						Manifest:     manifestPath,
						IsGitHubRepo: true,
						GitHubOwner:  owner,
						GitHubRepo:   repo,
						IsLocal:      isLocal,
						SourceHost:   host,
					})
				}
			}
		}()
	}

	return deps
}

// parseFileForADOURLs scans a file for Azure DevOps repository URLs
func (ps *PackageScanner) parseFileForADOURLs(filePath, manifestPath string, ecosystem PackageEcosystem) []ExtractedDependency {
	var deps []ExtractedDependency

	// Patterns for Azure DevOps URLs
	// Format: https://dev.azure.com/{org}/{project}/_git/{repo}
	// Format: https://{org}@dev.azure.com/{org}/{project}/_git/{repo}
	// Format: git@ssh.dev.azure.com:v3/{org}/{project}/{repo}

	// #nosec G304 -- filePath is validated by caller
	file, err := os.Open(filePath)
	if err != nil {
		return deps
	}
	defer func() { _ = file.Close() }()

	adoPattern := regexp.MustCompile(`(?:https://)?(?:[^@]+@)?dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/"'\s]+)`)
	vstsPattern := regexp.MustCompile(`https://([^.]+)\.visualstudio\.com/([^/]+)/_git/([^/"'\s]+)`)
	sshPattern := regexp.MustCompile(`git@ssh\.dev\.azure\.com:v3/([^/]+)/([^/]+)/([^/"'\s]+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check dev.azure.com format
		if matches := adoPattern.FindStringSubmatch(line); len(matches) >= 4 {
			org, project, repo := matches[1], matches[2], matches[3]
			repo = strings.TrimSuffix(repo, ".git")

			isLocal := ps.isADOSource && ps.sourceOrg == org

			deps = append(deps, ExtractedDependency{
				Name:         org + "/" + project + "/" + repo,
				Ecosystem:    ecosystem,
				Manifest:     manifestPath,
				IsGitHubRepo: true, // We treat ADO repos similarly for dependency tracking
				GitHubOwner:  org + "/" + project,
				GitHubRepo:   repo,
				IsLocal:      isLocal,
				SourceHost:   hostAzureDevOps,
			})
		}

		// Check visualstudio.com format
		if matches := vstsPattern.FindStringSubmatch(line); len(matches) >= 4 {
			org, project, repo := matches[1], matches[2], matches[3]
			repo = strings.TrimSuffix(repo, ".git")

			isLocal := ps.isADOSource && ps.sourceOrg == org

			deps = append(deps, ExtractedDependency{
				Name:         org + "/" + project + "/" + repo,
				Ecosystem:    ecosystem,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  org + "/" + project,
				GitHubRepo:   repo,
				IsLocal:      isLocal,
				SourceHost:   org + suffixVisualStudio,
			})
		}

		// Check SSH format
		if matches := sshPattern.FindStringSubmatch(line); len(matches) >= 4 {
			org, project, repo := matches[1], matches[2], matches[3]
			repo = strings.TrimSuffix(repo, ".git")

			isLocal := ps.isADOSource && ps.sourceOrg == org

			deps = append(deps, ExtractedDependency{
				Name:         org + "/" + project + "/" + repo,
				Ecosystem:    ecosystem,
				Manifest:     manifestPath,
				IsGitHubRepo: true,
				GitHubOwner:  org + "/" + project,
				GitHubRepo:   repo,
				IsLocal:      isLocal,
				SourceHost:   hostAzureDevSSH,
			})
		}
	}

	return deps
}

// isADOURL checks if a URL is an Azure DevOps URL.
// This is a wrapper around ado.IsADOURL for package-internal use.
func isADOURL(gitURL string) bool {
	return ado.IsADOURL(gitURL)
}

// isADOHost checks if a host is an Azure DevOps host.
// This is a wrapper around ado.IsADOHost for package-internal use.
func isADOHost(host string) bool {
	return ado.IsADOHost(host)
}

// extractADOReference extracts org, project, and repo from an Azure DevOps Git URL.
// Uses the centralized ado.Parse function internally.
func (ps *PackageScanner) extractADOReference(gitURL string) (org, project, repo, host string, isLocal bool) {
	parsed := ado.Parse(gitURL)
	if parsed == nil {
		return "", "", "", "", false
	}

	org = parsed.Organization
	project = parsed.Project
	repo = parsed.Repository
	host = parsed.Host
	isLocal = ps.isADOSource && ps.sourceOrg == org
	return
}
