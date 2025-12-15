package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
)

// ManifestType represents the type of package manifest file
type ManifestType string

const (
	ManifestGoMod            ManifestType = "go.mod"
	ManifestPackageJSON      ManifestType = "package.json"
	ManifestRequirements     ManifestType = "requirements.txt"
	ManifestRequirementsStar ManifestType = "requirements*.txt"
	ManifestGemfile          ManifestType = "Gemfile"
	ManifestTerraform        ManifestType = "*.tf"
	ManifestCargoToml        ManifestType = "Cargo.toml"
	ManifestChartYaml        ManifestType = "Chart.yaml"
	ManifestPackageSwift     ManifestType = "Package.swift"
	ManifestMixExs           ManifestType = "mix.exs"
	ManifestBuildGradle      ManifestType = "build.gradle"
	ManifestBuildGradleKts   ManifestType = "build.gradle.kts"
)

// ManifestFiles holds all discovered manifest files organized by type
type ManifestFiles struct {
	GoMod                []string // go.mod files
	PackageJSON          []string // package.json files
	Requirements         []string // requirements.txt files
	RequirementsVariants []string // requirements*.txt variants (excluding requirements.txt)
	Gemfile              []string // Gemfile files
	Terraform            []string // *.tf files
	CargoToml            []string // Cargo.toml files
	ChartYaml            []string // Chart.yaml files
	PackageSwift         []string // Package.swift files
	MixExs               []string // mix.exs files
	BuildGradle          []string // build.gradle files
	BuildGradleKts       []string // build.gradle.kts files
}

// PackageEcosystem represents a package manager ecosystem
type PackageEcosystem string

const (
	EcosystemNodeJS    PackageEcosystem = "npm"
	EcosystemGo        PackageEcosystem = "go"
	EcosystemPython    PackageEcosystem = "python"
	EcosystemRuby      PackageEcosystem = "rubygems"
	EcosystemTerraform PackageEcosystem = "terraform"
	EcosystemRust      PackageEcosystem = "cargo"
	EcosystemHelm      PackageEcosystem = "helm"
	EcosystemSwift     PackageEcosystem = "swift"
	EcosystemElixir    PackageEcosystem = "mix"
	EcosystemGradle    PackageEcosystem = "gradle"
)

// Common host names
const (
	hostGitHubCom      = "github.com"
	hostAzureDevOps    = "dev.azure.com"
	hostAzureDevSSH    = "ssh.dev.azure.com"
	suffixVisualStudio = ".visualstudio.com"
)

// Directories to skip during file scanning
const (
	dirNodeModules = "node_modules"
	dirVendor      = "vendor"
	dirGit         = ".git"
	dirPycache     = "__pycache__"
	dirTerraform   = ".terraform"
	dirTarget      = "target"
	dirBin         = "bin"
	dirObj         = "obj"
	dirPackages    = "packages"
)

// ExtractedDependency represents a dependency extracted from a manifest file
type ExtractedDependency struct {
	Name         string           // Package name or GitHub repo (owner/repo)
	Version      string           // Version constraint
	Ecosystem    PackageEcosystem // Which ecosystem this is from
	Manifest     string           // Path to manifest file
	IsGitHubRepo bool             // Whether this is a GitHub/Git host repository reference
	GitHubOwner  string           // GitHub owner (if IsGitHubRepo)
	GitHubRepo   string           // GitHub repo name (if IsGitHubRepo)
	IsLocal      bool             // Whether this dependency is local to the source instance
	SourceHost   string           // The host this dependency is from (e.g., "github.com" or "github.example.com")
}

// PackageScanner scans repositories for package manager files and extracts GitHub/Git dependencies
type PackageScanner struct {
	logger          *slog.Logger
	sourceHost      string   // The hostname of the source instance (e.g., "github.example.com" or "dev.azure.com")
	sourceOrg       string   // For ADO sources, the organization name
	isADOSource     bool     // Whether the source is Azure DevOps
	additionalHosts []string // Additional hosts to scan for (always includes github.com)
}

// NewPackageScanner creates a new package scanner
func NewPackageScanner(logger *slog.Logger) *PackageScanner {
	return &PackageScanner{
		logger:          logger,
		sourceHost:      "",
		additionalHosts: []string{hostGitHubCom},
	}
}

// WithSourceURL configures the scanner with the source instance URL
// This allows detection of local dependencies (dependencies hosted on the source instance)
// Supports both GitHub (github.com, GitHub Enterprise) and Azure DevOps sources
func (ps *PackageScanner) WithSourceURL(sourceURL string) *PackageScanner {
	if sourceURL == "" {
		return ps
	}

	// Parse the URL to extract the hostname
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		ps.logger.Debug("Failed to parse source URL", "url", sourceURL, "error", err)
		return ps
	}

	host := parsed.Host
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	ps.sourceHost = host

	// Check if this is an Azure DevOps source
	if host == hostAzureDevOps || strings.HasSuffix(host, suffixVisualStudio) {
		ps.isADOSource = true
		// Extract ADO organization from URL path
		// Format: https://dev.azure.com/{org}/... or https://{org}.visualstudio.com/...
		if host == hostAzureDevOps {
			// dev.azure.com/{org}/project/_git/repo
			pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(pathParts) > 0 {
				ps.sourceOrg = pathParts[0]
			}
		} else if strings.HasSuffix(host, suffixVisualStudio) {
			// {org}.visualstudio.com - org is in the hostname
			ps.sourceOrg = strings.TrimSuffix(host, suffixVisualStudio)
		}

		// Add ADO hosts for scanning
		ps.additionalHosts = append(ps.additionalHosts, hostAzureDevOps)

		ps.logger.Debug("Package scanner configured for Azure DevOps source",
			"source_host", ps.sourceHost,
			"source_org", ps.sourceOrg,
			"scan_hosts", ps.additionalHosts)
	} else {
		// GitHub source (github.com or GitHub Enterprise)
		// Add source host to additional hosts if it's not already github.com
		if host != "" && host != hostGitHubCom {
			ps.additionalHosts = append(ps.additionalHosts, host)
		}

		ps.logger.Debug("Package scanner configured with source host",
			"source_host", ps.sourceHost,
			"scan_hosts", ps.additionalHosts)
	}

	return ps
}

// ScanPackageManagers scans a repository for package manager files and extracts actual dependencies
// It focuses on extracting dependencies that are GitHub repositories, which are relevant for migration planning
// Uses a single-pass directory walk for efficiency, then parses manifests in parallel
func (ps *PackageScanner) ScanPackageManagers(ctx context.Context, repoPath string, repoID int64) ([]*models.RepositoryDependency, error) {
	// Validate repository path
	if err := source.ValidateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repository path: %w", err)
	}

	now := time.Now()

	// Single-pass directory walk to collect all manifest files
	manifests := ps.collectAllManifests(repoPath)

	// Parse manifests in parallel using goroutines
	var wg sync.WaitGroup
	depsChan := make(chan []ExtractedDependency, 12) // Buffer for all manifest types

	// Go modules
	if len(manifests.GoMod) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parseGoModFiles(manifests.GoMod, repoPath)
			depsChan <- deps
		}()
	}

	// npm/package.json
	if len(manifests.PackageJSON) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parsePackageJSONFiles(manifests.PackageJSON, repoPath)
			depsChan <- deps
		}()
	}

	// Python requirements.txt
	if len(manifests.Requirements) > 0 || len(manifests.RequirementsVariants) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allReqs := append(manifests.Requirements, manifests.RequirementsVariants...)
			deps := ps.parseRequirementsFiles(allReqs, repoPath)
			depsChan <- deps
		}()
	}

	// Ruby Gemfile
	if len(manifests.Gemfile) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parseGemfiles(manifests.Gemfile, repoPath)
			depsChan <- deps
		}()
	}

	// Terraform *.tf
	if len(manifests.Terraform) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parseTerraformFiles(manifests.Terraform, repoPath)
			depsChan <- deps
		}()
	}

	// Rust Cargo.toml
	if len(manifests.CargoToml) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parseCargoFiles(manifests.CargoToml, repoPath)
			depsChan <- deps
		}()
	}

	// Helm Chart.yaml
	if len(manifests.ChartYaml) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parseChartYamlFiles(manifests.ChartYaml, repoPath)
			depsChan <- deps
		}()
	}

	// Swift Package.swift
	if len(manifests.PackageSwift) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parsePackageSwiftFiles(manifests.PackageSwift, repoPath)
			depsChan <- deps
		}()
	}

	// Elixir mix.exs
	if len(manifests.MixExs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps := ps.parseMixExsFiles(manifests.MixExs, repoPath)
			depsChan <- deps
		}()
	}

	// Gradle build.gradle and build.gradle.kts
	if len(manifests.BuildGradle) > 0 || len(manifests.BuildGradleKts) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allGradle := append(manifests.BuildGradle, manifests.BuildGradleKts...)
			deps := ps.parseBuildGradleFiles(allGradle, repoPath)
			depsChan <- deps
		}()
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(depsChan)
	}()

	// Collect all dependencies from channel
	var allDeps []ExtractedDependency
	for deps := range depsChan {
		allDeps = append(allDeps, deps...)
	}

	// Convert extracted dependencies to RepositoryDependency objects
	dependencies := ps.convertToRepositoryDependencies(allDeps, repoID, now)

	// Deduplicate dependencies (same repo might be referenced from multiple manifests)
	dependencies = ps.deduplicateDependencies(dependencies)

	ps.logger.Debug("Package dependency extraction complete",
		"repo_path", repoPath,
		"github_dependencies", len(dependencies))

	return dependencies, nil
}

// convertToRepositoryDependencies converts extracted dependencies to RepositoryDependency objects
func (ps *PackageScanner) convertToRepositoryDependencies(deps []ExtractedDependency, repoID int64, now time.Time) []*models.RepositoryDependency {
	result := make([]*models.RepositoryDependency, 0, len(deps))

	for _, dep := range deps {
		if !dep.IsGitHubRepo {
			continue
		}

		repoFullName := fmt.Sprintf("%s/%s", dep.GitHubOwner, dep.GitHubRepo)

		// Determine package manager name for metadata
		packageManager := string(dep.Ecosystem)
		switch dep.Ecosystem {
		case EcosystemGo:
			packageManager = "GO"
		case EcosystemNodeJS:
			packageManager = "NPM"
		case EcosystemPython:
			packageManager = "PIP"
		case EcosystemRuby:
			packageManager = "RUBYGEMS"
		case EcosystemTerraform:
			packageManager = "TERRAFORM"
		case EcosystemRust:
			packageManager = "CARGO"
		case EcosystemHelm:
			packageManager = "HELM"
		case EcosystemSwift:
			packageManager = "SWIFT"
		case EcosystemElixir:
			packageManager = "MIX"
		case EcosystemGradle:
			packageManager = "GRADLE"
		}

		// Determine the dependency URL based on the source host
		depURL := fmt.Sprintf("https://%s/%s", dep.SourceHost, repoFullName)

		metadata := map[string]interface{}{
			"source":          "file_scan",
			"ecosystem":       string(dep.Ecosystem),
			"manifest":        dep.Manifest,
			"package_name":    dep.Name,
			"version":         dep.Version,
			"package_manager": packageManager,
			"source_host":     dep.SourceHost,
			"is_local":        dep.IsLocal,
		}
		metadataJSON, _ := json.Marshal(metadata)
		metadataStr := string(metadataJSON)

		result = append(result, &models.RepositoryDependency{
			RepositoryID:       repoID,
			DependencyFullName: repoFullName,
			DependencyType:     models.DependencyTypePackage,
			DependencyURL:      depURL,
			IsLocal:            dep.IsLocal,
			DiscoveredAt:       now,
			Metadata:           &metadataStr,
		})
	}

	return result
}

// deduplicateDependencies removes duplicate dependencies, keeping the first occurrence
func (ps *PackageScanner) deduplicateDependencies(deps []*models.RepositoryDependency) []*models.RepositoryDependency {
	seen := make(map[string]bool)
	result := make([]*models.RepositoryDependency, 0, len(deps))

	for _, dep := range deps {
		if !seen[dep.DependencyFullName] {
			seen[dep.DependencyFullName] = true
			result = append(result, dep)
		}
	}

	return result
}

// parseGoMod parses a go.mod file and extracts GitHub dependencies
func (ps *PackageScanner) parseGoMod(modPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- modPath is validated via findFiles
	file, err := os.Open(modPath)
	if err != nil {
		return deps
	}
	defer file.Close()

	inRequireBlock := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Track require block
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		// Parse require lines
		dep := ps.parseGoModRequireLine(line, inRequireBlock, manifestPath)
		if dep != nil {
			deps = append(deps, *dep)
		}
	}

	return deps
}

// parseGoModRequireLine parses a single line from go.mod and returns a dependency if it's from a tracked host
func (ps *PackageScanner) parseGoModRequireLine(line string, inRequireBlock bool, manifestPath string) *ExtractedDependency {
	modulePath, version := ps.extractGoModulePath(line, inRequireBlock)
	if modulePath == "" {
		return nil
	}

	// Check if this module is from any of the tracked hosts (GitHub/GitHub Enterprise)
	if dep := ps.matchGoModuleToHost(modulePath, version, manifestPath); dep != nil {
		return dep
	}

	// Check for Azure DevOps module paths
	return ps.matchGoModuleToADO(modulePath, version, manifestPath)
}

// extractGoModulePath extracts module path and version from a go.mod line
func (ps *PackageScanner) extractGoModulePath(line string, inRequireBlock bool) (modulePath, version string) {
	if inRequireBlock && line != "" && !strings.HasPrefix(line, "//") {
		// Format: module/path v1.2.3
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	} else if strings.HasPrefix(line, "require ") && !strings.HasPrefix(line, "require (") {
		// Single require: require module/path v1.2.3
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			return parts[1], parts[2]
		}
	}
	return "", ""
}

// matchGoModuleToHost checks if a module path matches any tracked GitHub host
func (ps *PackageScanner) matchGoModuleToHost(modulePath, version, manifestPath string) *ExtractedDependency {
	for _, host := range ps.additionalHosts {
		prefix := host + "/"
		if strings.HasPrefix(modulePath, prefix) {
			parts := strings.Split(modulePath, "/")
			if len(parts) >= 3 {
				isLocal := ps.sourceHost != "" && host == ps.sourceHost
				return &ExtractedDependency{
					Name:         modulePath,
					Version:      version,
					Ecosystem:    EcosystemGo,
					Manifest:     manifestPath,
					IsGitHubRepo: true,
					GitHubOwner:  parts[1],
					GitHubRepo:   parts[2],
					IsLocal:      isLocal,
					SourceHost:   host,
				}
			}
		}
	}
	return nil
}

// matchGoModuleToADO checks if a module path matches Azure DevOps format
func (ps *PackageScanner) matchGoModuleToADO(modulePath, version, manifestPath string) *ExtractedDependency {
	// Format: dev.azure.com/{org}/{project}/_git/{repo}.git/...
	if !strings.HasPrefix(modulePath, hostAzureDevOps+"/") {
		return nil
	}

	parts := strings.Split(modulePath, "/")
	// dev.azure.com / org / project / _git / repo.git / subpath...
	if len(parts) >= 5 && parts[3] == "_git" {
		org := parts[1]
		project := parts[2]
		repo := strings.TrimSuffix(parts[4], ".git")
		isLocal := ps.isADOSource && ps.sourceOrg == org

		return &ExtractedDependency{
			Name:         modulePath,
			Version:      version,
			Ecosystem:    EcosystemGo,
			Manifest:     manifestPath,
			IsGitHubRepo: true,
			GitHubOwner:  org + "/" + project,
			GitHubRepo:   repo,
			IsLocal:      isLocal,
			SourceHost:   hostAzureDevOps,
		}
	}
	return nil
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
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

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

// parseRequirementsTxt parses requirements.txt and extracts GitHub dependencies
func (ps *PackageScanner) parseRequirementsTxt(reqPath, manifestPath string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- reqPath is validated via findFiles
	file, err := os.Open(reqPath)
	if err != nil {
		return deps
	}
	defer file.Close()

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
	if idx := strings.Index(line, "#egg="); idx != -1 {
		pkgName := line[idx+5:]
		return strings.Split(pkgName, "&")[0]
	}
	return fallback
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
	defer file.Close()

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

// extractADOReference extracts Azure DevOps repository reference from a git URL
// Returns org, project, repo, host, isLocal
// Supports:
//   - https://dev.azure.com/{org}/{project}/_git/{repo}
//   - https://{org}.visualstudio.com/{project}/_git/{repo}
//   - git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
func (ps *PackageScanner) extractADOReference(gitURL string) (org, project, repo, host string, isLocal bool) {
	// Pattern for dev.azure.com URLs
	// https://dev.azure.com/{org}/{project}/_git/{repo}
	adoHTTPSPattern := regexp.MustCompile(`https://dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/?#]+)`)
	if matches := adoHTTPSPattern.FindStringSubmatch(gitURL); len(matches) == 4 {
		org, project, repo = matches[1], matches[2], strings.TrimSuffix(matches[3], ".git")
		isLocal = ps.isADOSource && ps.sourceOrg == org
		return org, project, repo, hostAzureDevOps, isLocal
	}

	// Pattern for visualstudio.com URLs
	// https://{org}.visualstudio.com/{project}/_git/{repo}
	vsPattern := regexp.MustCompile(`https://([^.]+)\.visualstudio\.com/([^/]+)/_git/([^/?#]+)`)
	if matches := vsPattern.FindStringSubmatch(gitURL); len(matches) == 4 {
		org, project, repo = matches[1], matches[2], strings.TrimSuffix(matches[3], ".git")
		isLocal = ps.isADOSource && ps.sourceOrg == org
		return org, project, repo, org + suffixVisualStudio, isLocal
	}

	// Pattern for SSH URLs
	// git@ssh.dev.azure.com:v3/{org}/{project}/{repo}
	adoSSHPattern := regexp.MustCompile(`git@ssh\.dev\.azure\.com:v3/([^/]+)/([^/]+)/([^/?#]+)`)
	if matches := adoSSHPattern.FindStringSubmatch(gitURL); len(matches) == 4 {
		org, project, repo = matches[1], matches[2], strings.TrimSuffix(matches[3], ".git")
		isLocal = ps.isADOSource && ps.sourceOrg == org
		return org, project, repo, hostAzureDevSSH, isLocal
	}

	return "", "", "", "", false
}

// isADOURL checks if a URL is an Azure DevOps URL
func isADOURL(gitURL string) bool {
	return strings.Contains(gitURL, "dev.azure.com") ||
		strings.Contains(gitURL, ".visualstudio.com") ||
		strings.Contains(gitURL, "ssh.dev.azure.com")
}

// isADOHost checks if a host is an Azure DevOps host
// This is used to skip ADO hosts when iterating over additionalHosts for GitHub-style pattern matching,
// since ADO URLs have a different format (org/project/_git/repo) that doesn't match the host/owner/repo pattern
func isADOHost(host string) bool {
	return host == hostAzureDevOps ||
		host == hostAzureDevSSH ||
		strings.HasSuffix(host, suffixVisualStudio)
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

// parseFileWithURLPattern is a generic parser for files containing GitHub URL references
// It takes a pattern prefix (before the host) and suffix (after the host) to match URLs
// Note: This function uses GitHub-style URL patterns (host/owner/repo) and should NOT be used
// for ADO URLs which have a different format (org/project/_git/repo). ADO URLs are handled
// separately by parseFileForADOURLs.
func (ps *PackageScanner) parseFileWithURLPattern(filePath, manifestPath string, ecosystem PackageEcosystem, patternPrefix, patternSuffix string) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- filePath is validated via findFiles
	content, err := os.ReadFile(filePath)
	if err != nil {
		return deps
	}

	for _, host := range ps.additionalHosts {
		// Skip ADO hosts - they don't follow the host/owner/repo pattern
		// and are handled separately by parseFileForADOURLs
		if isADOHost(host) {
			continue
		}

		escapedHost := regexp.QuoteMeta(host)
		pattern := regexp.MustCompile(patternPrefix + escapedHost + patternSuffix)
		matches := pattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) == 3 {
				isLocal := ps.sourceHost != "" && host == ps.sourceHost
				deps = append(deps, ExtractedDependency{
					Name:         match[1] + "/" + strings.TrimSuffix(match[2], ".git"),
					Version:      match[0],
					Ecosystem:    ecosystem,
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

// parseFileForADOURLs scans a file for Azure DevOps git URLs
func (ps *PackageScanner) parseFileForADOURLs(filePath, manifestPath string, ecosystem PackageEcosystem) []ExtractedDependency {
	var deps []ExtractedDependency

	// #nosec G304 -- filePath is validated via findFiles
	content, err := os.ReadFile(filePath)
	if err != nil {
		return deps
	}

	// Find all URLs that look like ADO
	urlPattern := regexp.MustCompile(`https?://[^\s"'<>]+`)
	matches := urlPattern.FindAllString(string(content), -1)

	for _, match := range matches {
		if isADOURL(match) {
			org, project, repo, host, isLocal := ps.extractADOReference(match)
			if org != "" && repo != "" {
				deps = append(deps, ExtractedDependency{
					Name:         org + "/" + project + "/" + repo,
					Version:      match,
					Ecosystem:    ecosystem,
					Manifest:     manifestPath,
					IsGitHubRepo: true,
					GitHubOwner:  org + "/" + project,
					GitHubRepo:   repo,
					IsLocal:      isLocal,
					SourceHost:   host,
				})
			}
		}
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

// parseGoModFiles parses a list of go.mod files and extracts dependencies
func (ps *PackageScanner) parseGoModFiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
	for _, modPath := range files {
		relPath, _ := filepath.Rel(repoPath, modPath)
		extracted := ps.parseGoMod(modPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parsePackageJSONFiles parses a list of package.json files and extracts dependencies
func (ps *PackageScanner) parsePackageJSONFiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
	for _, pkgPath := range files {
		relPath, _ := filepath.Rel(repoPath, pkgPath)
		extracted := ps.parsePackageJSON(pkgPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseRequirementsFiles parses a list of requirements.txt files and extracts dependencies
func (ps *PackageScanner) parseRequirementsFiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
	for _, reqPath := range files {
		relPath, _ := filepath.Rel(repoPath, reqPath)
		extracted := ps.parseRequirementsTxt(reqPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseGemfiles parses a list of Gemfile files and extracts dependencies
func (ps *PackageScanner) parseGemfiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
	for _, gemPath := range files {
		relPath, _ := filepath.Rel(repoPath, gemPath)
		extracted := ps.parseGemfile(gemPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

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

// parseCargoFiles parses a list of Cargo.toml files and extracts dependencies
func (ps *PackageScanner) parseCargoFiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
	for _, cargoPath := range files {
		relPath, _ := filepath.Rel(repoPath, cargoPath)
		extracted := ps.parseCargoToml(cargoPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseChartYamlFiles parses a list of Chart.yaml files and extracts dependencies
func (ps *PackageScanner) parseChartYamlFiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
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
	var deps []ExtractedDependency
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
	var deps []ExtractedDependency
	for _, mixPath := range files {
		relPath, _ := filepath.Rel(repoPath, mixPath)
		extracted := ps.parseMixExs(mixPath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}

// parseBuildGradleFiles parses a list of build.gradle/build.gradle.kts files and extracts dependencies
func (ps *PackageScanner) parseBuildGradleFiles(files []string, repoPath string) []ExtractedDependency {
	var deps []ExtractedDependency
	for _, gradlePath := range files {
		relPath, _ := filepath.Rel(repoPath, gradlePath)
		extracted := ps.parseBuildGradle(gradlePath, relPath)
		deps = append(deps, extracted...)
	}
	return deps
}
