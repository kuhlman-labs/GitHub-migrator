package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
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
		} else if before, ok := strings.CutSuffix(host, suffixVisualStudio); ok {
			// {org}.visualstudio.com - org is in the hostname
			ps.sourceOrg = before
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
		wg.Go(func() {
			deps := ps.parseGoModFiles(manifests.GoMod, repoPath)
			depsChan <- deps
		})
	}

	// npm/package.json
	if len(manifests.PackageJSON) > 0 {
		wg.Go(func() {
			deps := ps.parsePackageJSONFiles(manifests.PackageJSON, repoPath)
			depsChan <- deps
		})
	}

	// Python requirements.txt
	if len(manifests.Requirements) > 0 || len(manifests.RequirementsVariants) > 0 {
		wg.Go(func() {
			allReqs := append(manifests.Requirements, manifests.RequirementsVariants...)
			deps := ps.parseRequirementsFiles(allReqs, repoPath)
			depsChan <- deps
		})
	}

	// Ruby Gemfile
	if len(manifests.Gemfile) > 0 {
		wg.Go(func() {
			deps := ps.parseGemfiles(manifests.Gemfile, repoPath)
			depsChan <- deps
		})
	}

	// Terraform *.tf
	if len(manifests.Terraform) > 0 {
		wg.Go(func() {
			deps := ps.parseTerraformFiles(manifests.Terraform, repoPath)
			depsChan <- deps
		})
	}

	// Rust Cargo.toml
	if len(manifests.CargoToml) > 0 {
		wg.Go(func() {
			deps := ps.parseCargoFiles(manifests.CargoToml, repoPath)
			depsChan <- deps
		})
	}

	// Helm Chart.yaml
	if len(manifests.ChartYaml) > 0 {
		wg.Go(func() {
			deps := ps.parseChartYamlFiles(manifests.ChartYaml, repoPath)
			depsChan <- deps
		})
	}

	// Swift Package.swift
	if len(manifests.PackageSwift) > 0 {
		wg.Go(func() {
			deps := ps.parsePackageSwiftFiles(manifests.PackageSwift, repoPath)
			depsChan <- deps
		})
	}

	// Elixir mix.exs
	if len(manifests.MixExs) > 0 {
		wg.Go(func() {
			deps := ps.parseMixExsFiles(manifests.MixExs, repoPath)
			depsChan <- deps
		})
	}

	// Gradle build.gradle and build.gradle.kts
	if len(manifests.BuildGradle) > 0 || len(manifests.BuildGradleKts) > 0 {
		wg.Go(func() {
			allGradle := append(manifests.BuildGradle, manifests.BuildGradleKts...)
			deps := ps.parseBuildGradleFiles(allGradle, repoPath)
			depsChan <- deps
		})
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
			packageManager = "BUNDLER"
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

		// Build metadata JSON with package manager and manifest info
		metadata := fmt.Sprintf(`{"package_manager":"%s","manifest":"%s","version":"%s"}`, packageManager, dep.Manifest, dep.Version)

		// Build URL with https:// prefix
		depURL := "https://" + dep.SourceHost + "/" + repoFullName

		result = append(result, &models.RepositoryDependency{
			RepositoryID:       repoID,
			DependencyFullName: repoFullName,
			DependencyType:     "package",
			DependencyURL:      depURL,
			IsLocal:            dep.IsLocal,
			DiscoveredAt:       now,
			Metadata:           &metadata,
		})
	}

	return result
}

// deduplicateDependencies removes duplicate dependencies keeping the first occurrence
func (ps *PackageScanner) deduplicateDependencies(deps []*models.RepositoryDependency) []*models.RepositoryDependency {
	seen := make(map[string]bool)
	result := make([]*models.RepositoryDependency, 0, len(deps))

	for _, dep := range deps {
		key := fmt.Sprintf("%d:%s:%s", dep.RepositoryID, dep.DependencyFullName, dep.DependencyType)
		if !seen[key] {
			seen[key] = true
			result = append(result, dep)
		}
	}

	return result
}
