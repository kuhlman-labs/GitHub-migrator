package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
)

// PackageEcosystem represents a package manager ecosystem
type PackageEcosystem string

const (
	EcosystemNodeJS    PackageEcosystem = "npm"
	EcosystemGo        PackageEcosystem = "go"
	EcosystemPython    PackageEcosystem = "python"
	EcosystemJava      PackageEcosystem = "maven"
	EcosystemGradle    PackageEcosystem = "gradle"
	EcosystemDotNet    PackageEcosystem = "nuget"
	EcosystemRuby      PackageEcosystem = "rubygems"
	EcosystemRust      PackageEcosystem = "cargo"
	EcosystemPHP       PackageEcosystem = "composer"
	EcosystemTerraform PackageEcosystem = "terraform"
	EcosystemHelm      PackageEcosystem = "helm"
	EcosystemDocker    PackageEcosystem = "docker"
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

// PackageManifest represents a detected package manifest file
type PackageManifest struct {
	Ecosystem       PackageEcosystem
	ManifestPath    string
	DependencyCount int
	HasLockFile     bool
	Metadata        map[string]interface{}
}

// PackageScanner scans repositories for package manager files
type PackageScanner struct {
	logger *slog.Logger
}

// NewPackageScanner creates a new package scanner
func NewPackageScanner(logger *slog.Logger) *PackageScanner {
	return &PackageScanner{
		logger: logger,
	}
}

// ScanPackageManagers scans a repository for package manager files and returns dependencies
func (ps *PackageScanner) ScanPackageManagers(ctx context.Context, repoPath string, repoID int64) ([]*models.RepositoryDependency, error) {
	// Validate repository path
	if err := source.ValidateRepoPath(repoPath); err != nil {
		return nil, fmt.Errorf("invalid repository path: %w", err)
	}

	now := time.Now()

	// Scan for each ecosystem
	manifests := ps.scanAllEcosystems(ctx, repoPath)

	// Pre-allocate dependencies slice
	dependencies := make([]*models.RepositoryDependency, 0, len(manifests))

	ps.logger.Debug("Package scan complete",
		"repo_path", repoPath,
		"manifests_found", len(manifests))

	// Convert manifests to dependencies
	for _, manifest := range manifests {
		metadata := map[string]interface{}{
			"source":           "file_scan",
			"ecosystem":        string(manifest.Ecosystem),
			"manifest":         manifest.ManifestPath,
			"dependency_count": manifest.DependencyCount,
			"has_lock_file":    manifest.HasLockFile,
		}

		// Merge any additional metadata
		for k, v := range manifest.Metadata {
			metadata[k] = v
		}

		metadataJSON, _ := json.Marshal(metadata)
		metadataStr := string(metadataJSON)

		dep := &models.RepositoryDependency{
			RepositoryID:       repoID,
			DependencyFullName: fmt.Sprintf("%s:%s", manifest.Ecosystem, manifest.ManifestPath),
			DependencyType:     models.DependencyTypePackage,
			DependencyURL:      "", // Package manifests don't have URLs
			IsLocal:            true,
			DiscoveredAt:       now,
			Metadata:           &metadataStr,
		}
		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// scanAllEcosystems scans for all supported package ecosystems
func (ps *PackageScanner) scanAllEcosystems(ctx context.Context, repoPath string) []PackageManifest {
	var manifests []PackageManifest

	// Node.js (npm, yarn, pnpm)
	manifests = append(manifests, ps.scanNodeJS(ctx, repoPath)...)

	// Go
	manifests = append(manifests, ps.scanGo(ctx, repoPath)...)

	// Python
	manifests = append(manifests, ps.scanPython(ctx, repoPath)...)

	// Java (Maven)
	manifests = append(manifests, ps.scanMaven(ctx, repoPath)...)

	// Java/Kotlin (Gradle)
	manifests = append(manifests, ps.scanGradle(ctx, repoPath)...)

	// .NET
	manifests = append(manifests, ps.scanDotNet(ctx, repoPath)...)

	// Ruby
	manifests = append(manifests, ps.scanRuby(ctx, repoPath)...)

	// Rust
	manifests = append(manifests, ps.scanRust(ctx, repoPath)...)

	// PHP (Composer)
	manifests = append(manifests, ps.scanPHP(ctx, repoPath)...)

	// Terraform
	manifests = append(manifests, ps.scanTerraform(ctx, repoPath)...)

	// Helm
	manifests = append(manifests, ps.scanHelm(ctx, repoPath)...)

	// Docker
	manifests = append(manifests, ps.scanDocker(ctx, repoPath)...)

	return manifests
}

// scanNodeJS scans for Node.js package files
func (ps *PackageScanner) scanNodeJS(ctx context.Context, repoPath string) []PackageManifest {
	// Find all package.json files
	packageJSONFiles := ps.findFiles(repoPath, "package.json")

	// Pre-allocate manifests slice
	manifests := make([]PackageManifest, 0, len(packageJSONFiles))

	for _, pkgPath := range packageJSONFiles {
		relPath, _ := filepath.Rel(repoPath, pkgPath)

		manifest := PackageManifest{
			Ecosystem:    EcosystemNodeJS,
			ManifestPath: relPath,
			Metadata:     make(map[string]interface{}),
		}

		// Count dependencies from package.json
		depCount, devDepCount := ps.countNodeJSDependencies(pkgPath)
		manifest.DependencyCount = depCount + devDepCount
		manifest.Metadata["dev_dependencies"] = devDepCount

		// Check for lock files in the same directory
		dir := filepath.Dir(pkgPath)
		if ps.fileExists(filepath.Join(dir, "package-lock.json")) {
			manifest.HasLockFile = true
			manifest.Metadata["lock_file"] = "package-lock.json"
		} else if ps.fileExists(filepath.Join(dir, "yarn.lock")) {
			manifest.HasLockFile = true
			manifest.Metadata["lock_file"] = "yarn.lock"
		} else if ps.fileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
			manifest.HasLockFile = true
			manifest.Metadata["lock_file"] = "pnpm-lock.yaml"
		}

		manifests = append(manifests, manifest)
	}

	return manifests
}

// countNodeJSDependencies counts dependencies in a package.json file
func (ps *PackageScanner) countNodeJSDependencies(pkgPath string) (deps int, devDeps int) {
	// #nosec G304 -- pkgPath is validated via findFiles which uses validated repoPath
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return 0, 0
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(content, &pkg); err != nil {
		return 0, 0
	}

	return len(pkg.Dependencies), len(pkg.DevDependencies)
}

// scanGo scans for Go module files
func (ps *PackageScanner) scanGo(ctx context.Context, repoPath string) []PackageManifest {
	goModFiles := ps.findFiles(repoPath, "go.mod")

	// Pre-allocate manifests slice
	manifests := make([]PackageManifest, 0, len(goModFiles))

	for _, modPath := range goModFiles {
		relPath, _ := filepath.Rel(repoPath, modPath)

		manifest := PackageManifest{
			Ecosystem:    EcosystemGo,
			ManifestPath: relPath,
			Metadata:     make(map[string]interface{}),
		}

		// Count require statements
		manifest.DependencyCount = ps.countGoModDependencies(modPath)

		// Check for go.sum
		dir := filepath.Dir(modPath)
		if ps.fileExists(filepath.Join(dir, "go.sum")) {
			manifest.HasLockFile = true
			manifest.Metadata["lock_file"] = "go.sum"
		}

		// Extract module name
		if moduleName := ps.extractGoModuleName(modPath); moduleName != "" {
			manifest.Metadata["module_name"] = moduleName
		}

		manifests = append(manifests, manifest)
	}

	return manifests
}

// countGoModDependencies counts require statements in go.mod
func (ps *PackageScanner) countGoModDependencies(modPath string) int {
	// #nosec G304 -- modPath is validated via findFiles
	file, err := os.Open(modPath)
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0
	inRequireBlock := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}
		if inRequireBlock && line != "" && !strings.HasPrefix(line, "//") {
			count++
		}
		if strings.HasPrefix(line, "require ") && !strings.HasPrefix(line, "require (") {
			count++
		}
	}

	return count
}

// extractGoModuleName extracts the module name from go.mod
func (ps *PackageScanner) extractGoModuleName(modPath string) string {
	// #nosec G304 -- modPath is validated via findFiles
	file, err := os.Open(modPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	return ""
}

// scanPython scans for Python dependency files
func (ps *PackageScanner) scanPython(ctx context.Context, repoPath string) []PackageManifest {
	// Pre-allocate with estimated capacity (will grow if needed)
	manifests := make([]PackageManifest, 0, 4)

	// requirements.txt
	reqFiles := ps.findFiles(repoPath, "requirements.txt")
	for _, reqPath := range reqFiles {
		relPath, _ := filepath.Rel(repoPath, reqPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemPython,
			ManifestPath:    relPath,
			DependencyCount: ps.countRequirementsTxtDeps(reqPath),
			Metadata:        map[string]interface{}{"format": "requirements.txt"},
		}
		manifests = append(manifests, manifest)
	}

	// Also check for requirements*.txt variants
	reqVariants := ps.findFilesWithPattern(repoPath, "requirements*.txt")
	for _, reqPath := range reqVariants {
		relPath, _ := filepath.Rel(repoPath, reqPath)
		// Skip if already processed as requirements.txt
		if filepath.Base(reqPath) == "requirements.txt" {
			continue
		}
		manifest := PackageManifest{
			Ecosystem:       EcosystemPython,
			ManifestPath:    relPath,
			DependencyCount: ps.countRequirementsTxtDeps(reqPath),
			Metadata:        map[string]interface{}{"format": "requirements.txt"},
		}
		manifests = append(manifests, manifest)
	}

	// Pipfile
	pipfiles := ps.findFiles(repoPath, "Pipfile")
	for _, pipPath := range pipfiles {
		relPath, _ := filepath.Rel(repoPath, pipPath)
		dir := filepath.Dir(pipPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemPython,
			ManifestPath:    relPath,
			DependencyCount: ps.countPipfileDeps(pipPath),
			HasLockFile:     ps.fileExists(filepath.Join(dir, "Pipfile.lock")),
			Metadata:        map[string]interface{}{"format": "pipenv"},
		}
		manifests = append(manifests, manifest)
	}

	// pyproject.toml (Poetry, PEP 517/518)
	pyprojects := ps.findFiles(repoPath, "pyproject.toml")
	for _, ppPath := range pyprojects {
		relPath, _ := filepath.Rel(repoPath, ppPath)
		dir := filepath.Dir(ppPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemPython,
			ManifestPath:    relPath,
			DependencyCount: ps.countPyProjectDeps(ppPath),
			HasLockFile:     ps.fileExists(filepath.Join(dir, "poetry.lock")),
			Metadata:        map[string]interface{}{"format": "pyproject.toml"},
		}
		manifests = append(manifests, manifest)
	}

	// setup.py
	setupPyFiles := ps.findFiles(repoPath, "setup.py")
	for _, setupPath := range setupPyFiles {
		relPath, _ := filepath.Rel(repoPath, setupPath)
		manifest := PackageManifest{
			Ecosystem:    EcosystemPython,
			ManifestPath: relPath,
			Metadata:     map[string]interface{}{"format": "setup.py"},
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// countRequirementsTxtDeps counts dependencies in requirements.txt
func (ps *PackageScanner) countRequirementsTxtDeps(reqPath string) int {
	// #nosec G304 -- reqPath is validated via findFiles
	file, err := os.Open(reqPath)
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines, comments, and -r includes
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "-") {
			count++
		}
	}
	return count
}

// countPipfileDeps counts dependencies in a Pipfile (simplified)
func (ps *PackageScanner) countPipfileDeps(pipPath string) int {
	// #nosec G304 -- pipPath is validated via findFiles
	content, err := os.ReadFile(pipPath)
	if err != nil {
		return 0
	}

	// Simple heuristic: count lines that look like package definitions
	// This is a simplified parser - actual TOML parsing would be more accurate
	count := 0
	inPackages := false
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "[packages]" || line == "[dev-packages]" {
			inPackages = true
			continue
		}
		if strings.HasPrefix(line, "[") && inPackages {
			inPackages = false
			continue
		}
		if inPackages && line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
			count++
		}
	}
	return count
}

// countPyProjectDeps counts dependencies in pyproject.toml (simplified)
func (ps *PackageScanner) countPyProjectDeps(ppPath string) int {
	// #nosec G304 -- ppPath is validated via findFiles
	content, err := os.ReadFile(ppPath)
	if err != nil {
		return 0
	}

	// Simple heuristic: count lines in dependencies sections
	count := 0
	inDeps := false
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "dependencies") && strings.Contains(line, "[") {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") && inDeps {
			inDeps = false
			continue
		}
		if inDeps && line != "" && !strings.HasPrefix(line, "#") {
			// Check if it looks like a dependency line
			if strings.Contains(line, "=") || strings.Contains(line, "\"") {
				count++
			}
		}
	}
	return count
}

// scanMaven scans for Maven pom.xml files
func (ps *PackageScanner) scanMaven(ctx context.Context, repoPath string) []PackageManifest {
	pomFiles := ps.findFiles(repoPath, "pom.xml")
	manifests := make([]PackageManifest, 0, len(pomFiles))

	for _, pomPath := range pomFiles {
		relPath, _ := filepath.Rel(repoPath, pomPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemJava,
			ManifestPath:    relPath,
			DependencyCount: ps.countMavenDependencies(pomPath),
			Metadata:        map[string]interface{}{"build_tool": "maven"},
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// countMavenDependencies counts dependencies in pom.xml (simplified)
func (ps *PackageScanner) countMavenDependencies(pomPath string) int {
	// #nosec G304 -- pomPath is validated via findFiles
	content, err := os.ReadFile(pomPath)
	if err != nil {
		return 0
	}

	// Simple regex to count <dependency> blocks
	re := regexp.MustCompile(`<dependency>`)
	matches := re.FindAllString(string(content), -1)
	return len(matches)
}

// scanGradle scans for Gradle build files
func (ps *PackageScanner) scanGradle(ctx context.Context, repoPath string) []PackageManifest {
	// build.gradle (Groovy DSL)
	gradleFiles := ps.findFiles(repoPath, "build.gradle")
	// build.gradle.kts (Kotlin DSL)
	ktsFiles := ps.findFiles(repoPath, "build.gradle.kts")

	// Pre-allocate with combined capacity
	manifests := make([]PackageManifest, 0, len(gradleFiles)+len(ktsFiles))

	for _, gradlePath := range gradleFiles {
		relPath, _ := filepath.Rel(repoPath, gradlePath)
		dir := filepath.Dir(gradlePath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemGradle,
			ManifestPath:    relPath,
			DependencyCount: ps.countGradleDependencies(gradlePath),
			HasLockFile:     ps.fileExists(filepath.Join(dir, "gradle.lockfile")),
			Metadata:        map[string]interface{}{"dsl": "groovy"},
		}
		manifests = append(manifests, manifest)
	}

	for _, ktsPath := range ktsFiles {
		relPath, _ := filepath.Rel(repoPath, ktsPath)
		dir := filepath.Dir(ktsPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemGradle,
			ManifestPath:    relPath,
			DependencyCount: ps.countGradleDependencies(ktsPath),
			HasLockFile:     ps.fileExists(filepath.Join(dir, "gradle.lockfile")),
			Metadata:        map[string]interface{}{"dsl": "kotlin"},
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// countGradleDependencies counts dependencies in Gradle files (simplified)
func (ps *PackageScanner) countGradleDependencies(gradlePath string) int {
	// #nosec G304 -- gradlePath is validated via findFiles
	content, err := os.ReadFile(gradlePath)
	if err != nil {
		return 0
	}

	// Count implementation, api, compileOnly, testImplementation, etc.
	re := regexp.MustCompile(`(?m)^\s*(implementation|api|compileOnly|runtimeOnly|testImplementation|testRuntimeOnly)\s*[\("]`)
	matches := re.FindAllString(string(content), -1)
	return len(matches)
}

// scanDotNet scans for .NET project files
func (ps *PackageScanner) scanDotNet(ctx context.Context, repoPath string) []PackageManifest {
	// .csproj files
	csprojFiles := ps.findFilesWithPattern(repoPath, "*.csproj")
	// .fsproj files
	fsprojFiles := ps.findFilesWithPattern(repoPath, "*.fsproj")
	// packages.config (legacy NuGet)
	pkgConfigFiles := ps.findFiles(repoPath, "packages.config")

	// Pre-allocate with combined capacity
	manifests := make([]PackageManifest, 0, len(csprojFiles)+len(fsprojFiles)+len(pkgConfigFiles))

	for _, csprojPath := range csprojFiles {
		relPath, _ := filepath.Rel(repoPath, csprojPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemDotNet,
			ManifestPath:    relPath,
			DependencyCount: ps.countDotNetPackageRefs(csprojPath),
			Metadata:        map[string]interface{}{"project_type": "csharp"},
		}
		manifests = append(manifests, manifest)
	}

	for _, fsprojPath := range fsprojFiles {
		relPath, _ := filepath.Rel(repoPath, fsprojPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemDotNet,
			ManifestPath:    relPath,
			DependencyCount: ps.countDotNetPackageRefs(fsprojPath),
			Metadata:        map[string]interface{}{"project_type": "fsharp"},
		}
		manifests = append(manifests, manifest)
	}
	for _, pkgPath := range pkgConfigFiles {
		relPath, _ := filepath.Rel(repoPath, pkgPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemDotNet,
			ManifestPath:    relPath,
			DependencyCount: ps.countPackagesConfig(pkgPath),
			Metadata:        map[string]interface{}{"format": "packages.config"},
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// countDotNetPackageRefs counts PackageReference elements in project files
func (ps *PackageScanner) countDotNetPackageRefs(projPath string) int {
	// #nosec G304 -- projPath is validated via findFiles
	content, err := os.ReadFile(projPath)
	if err != nil {
		return 0
	}

	re := regexp.MustCompile(`<PackageReference`)
	matches := re.FindAllString(string(content), -1)
	return len(matches)
}

// countPackagesConfig counts package elements in packages.config
func (ps *PackageScanner) countPackagesConfig(pkgPath string) int {
	// #nosec G304 -- pkgPath is validated via findFiles
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return 0
	}

	re := regexp.MustCompile(`<package\s+`)
	matches := re.FindAllString(string(content), -1)
	return len(matches)
}

// scanSimpleEcosystem is a helper for ecosystems with a simple manifest + lock file pattern
func (ps *PackageScanner) scanSimpleEcosystem(
	repoPath string,
	ecosystem PackageEcosystem,
	manifestFile string,
	lockFile string,
	countFunc func(string) int,
) []PackageManifest {
	files := ps.findFiles(repoPath, manifestFile)
	manifests := make([]PackageManifest, 0, len(files))

	for _, filePath := range files {
		relPath, _ := filepath.Rel(repoPath, filePath)
		dir := filepath.Dir(filePath)
		manifest := PackageManifest{
			Ecosystem:       ecosystem,
			ManifestPath:    relPath,
			DependencyCount: countFunc(filePath),
			HasLockFile:     ps.fileExists(filepath.Join(dir, lockFile)),
			Metadata:        make(map[string]interface{}),
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// scanRuby scans for Ruby Gemfile
func (ps *PackageScanner) scanRuby(ctx context.Context, repoPath string) []PackageManifest {
	return ps.scanSimpleEcosystem(repoPath, EcosystemRuby, "Gemfile", "Gemfile.lock", ps.countGemfileDeps)
}

// countGemfileDeps counts gem dependencies in Gemfile
func (ps *PackageScanner) countGemfileDeps(gemPath string) int {
	// #nosec G304 -- gemPath is validated via findFiles
	content, err := os.ReadFile(gemPath)
	if err != nil {
		return 0
	}

	re := regexp.MustCompile(`(?m)^\s*gem\s+['"]`)
	matches := re.FindAllString(string(content), -1)
	return len(matches)
}

// scanRust scans for Rust Cargo files
func (ps *PackageScanner) scanRust(ctx context.Context, repoPath string) []PackageManifest {
	return ps.scanSimpleEcosystem(repoPath, EcosystemRust, "Cargo.toml", "Cargo.lock", ps.countCargoDeps)
}

// countCargoDeps counts dependencies in Cargo.toml
func (ps *PackageScanner) countCargoDeps(cargoPath string) int {
	// #nosec G304 -- cargoPath is validated via findFiles
	content, err := os.ReadFile(cargoPath)
	if err != nil {
		return 0
	}

	// Count lines in [dependencies], [dev-dependencies], [build-dependencies]
	count := 0
	inDeps := false
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "[dependencies]" || line == "[dev-dependencies]" || line == "[build-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") && inDeps {
			inDeps = false
			continue
		}
		if inDeps && line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
			count++
		}
	}
	return count
}

// scanPHP scans for PHP Composer files
func (ps *PackageScanner) scanPHP(ctx context.Context, repoPath string) []PackageManifest {
	composerFiles := ps.findFiles(repoPath, "composer.json")
	manifests := make([]PackageManifest, 0, len(composerFiles))

	for _, compPath := range composerFiles {
		relPath, _ := filepath.Rel(repoPath, compPath)
		dir := filepath.Dir(compPath)

		deps, devDeps := ps.countComposerDeps(compPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemPHP,
			ManifestPath:    relPath,
			DependencyCount: deps + devDeps,
			HasLockFile:     ps.fileExists(filepath.Join(dir, "composer.lock")),
			Metadata: map[string]interface{}{
				"dev_dependencies": devDeps,
			},
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// countComposerDeps counts dependencies in composer.json
func (ps *PackageScanner) countComposerDeps(compPath string) (deps int, devDeps int) {
	// #nosec G304 -- compPath is validated via findFiles
	content, err := os.ReadFile(compPath)
	if err != nil {
		return 0, 0
	}

	var composer struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}

	if err := json.Unmarshal(content, &composer); err != nil {
		return 0, 0
	}

	return len(composer.Require), len(composer.RequireDev)
}

// scanTerraform scans for Terraform files
func (ps *PackageScanner) scanTerraform(ctx context.Context, repoPath string) []PackageManifest {
	// Find directories with .tf files
	tfDirs := make(map[string]bool)
	tfFiles := ps.findFilesWithPattern(repoPath, "*.tf")

	// Pre-allocate with estimated capacity
	manifests := make([]PackageManifest, 0, len(tfFiles))

	for _, tfPath := range tfFiles {
		dir := filepath.Dir(tfPath)
		if !tfDirs[dir] {
			tfDirs[dir] = true

			relDir, _ := filepath.Rel(repoPath, dir)
			if relDir == "." {
				relDir = ""
			}

			providerCount := ps.countTerraformProviders(dir)
			moduleCount := ps.countTerraformModules(dir)

			manifest := PackageManifest{
				Ecosystem:       EcosystemTerraform,
				ManifestPath:    filepath.Join(relDir, "*.tf"),
				HasLockFile:     ps.fileExists(filepath.Join(dir, ".terraform.lock.hcl")),
				DependencyCount: providerCount + moduleCount,
				Metadata: map[string]interface{}{
					"providers": providerCount,
					"modules":   moduleCount,
				},
			}

			manifests = append(manifests, manifest)
		}
	}

	return manifests
}

// countTerraformProviders counts provider blocks in .tf files
func (ps *PackageScanner) countTerraformProviders(dir string) int {
	count := 0
	// #nosec G304 -- dir is validated via findFiles
	files, _ := filepath.Glob(filepath.Join(dir, "*.tf"))

	for _, file := range files {
		// #nosec G304 -- file is from validated directory
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`(?m)^\s*provider\s+"`)
		matches := re.FindAllString(string(content), -1)
		count += len(matches)
	}
	return count
}

// countTerraformModules counts module blocks in .tf files
func (ps *PackageScanner) countTerraformModules(dir string) int {
	count := 0
	// #nosec G304 -- dir is validated via findFiles
	files, _ := filepath.Glob(filepath.Join(dir, "*.tf"))

	for _, file := range files {
		// #nosec G304 -- file is from validated directory
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`(?m)^\s*module\s+"`)
		matches := re.FindAllString(string(content), -1)
		count += len(matches)
	}
	return count
}

// scanHelm scans for Helm charts
func (ps *PackageScanner) scanHelm(ctx context.Context, repoPath string) []PackageManifest {
	chartFiles := ps.findFiles(repoPath, "Chart.yaml")
	manifests := make([]PackageManifest, 0, len(chartFiles))

	for _, chartPath := range chartFiles {
		relPath, _ := filepath.Rel(repoPath, chartPath)
		dir := filepath.Dir(chartPath)

		manifest := PackageManifest{
			Ecosystem:       EcosystemHelm,
			ManifestPath:    relPath,
			DependencyCount: ps.countHelmDependencies(chartPath),
			HasLockFile:     ps.fileExists(filepath.Join(dir, "Chart.lock")),
			Metadata:        make(map[string]interface{}),
		}

		// Extract chart name if possible
		if chartName := ps.extractHelmChartName(chartPath); chartName != "" {
			manifest.Metadata["chart_name"] = chartName
		}

		manifests = append(manifests, manifest)
	}

	return manifests
}

// countHelmDependencies counts dependencies in Chart.yaml
func (ps *PackageScanner) countHelmDependencies(chartPath string) int {
	// #nosec G304 -- chartPath is validated via findFiles
	content, err := os.ReadFile(chartPath)
	if err != nil {
		return 0
	}

	// Count dependencies entries (simplified YAML parsing)
	count := 0
	inDeps := false
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "dependencies:" {
			inDeps = true
			continue
		}
		if inDeps && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
			inDeps = false
			continue
		}
		if inDeps && strings.HasPrefix(trimmed, "- name:") {
			count++
		}
	}
	return count
}

// extractHelmChartName extracts the chart name from Chart.yaml
func (ps *PackageScanner) extractHelmChartName(chartPath string) string {
	// #nosec G304 -- chartPath is validated via findFiles
	content, err := os.ReadFile(chartPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "name:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
	}
	return ""
}

// scanDocker scans for Docker-related files
func (ps *PackageScanner) scanDocker(ctx context.Context, repoPath string) []PackageManifest {
	// Dockerfile
	dockerfiles := ps.findFilesWithPattern(repoPath, "Dockerfile*")

	// docker-compose.yml / docker-compose.yaml
	composeFiles := ps.findFiles(repoPath, "docker-compose.yml")
	composeFiles = append(composeFiles, ps.findFiles(repoPath, "docker-compose.yaml")...)
	composeFiles = append(composeFiles, ps.findFiles(repoPath, "compose.yml")...)
	composeFiles = append(composeFiles, ps.findFiles(repoPath, "compose.yaml")...)

	// Pre-allocate with combined capacity
	manifests := make([]PackageManifest, 0, len(dockerfiles)+len(composeFiles))

	for _, dockerPath := range dockerfiles {
		relPath, _ := filepath.Rel(repoPath, dockerPath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemDocker,
			ManifestPath:    relPath,
			DependencyCount: ps.countDockerBaseImages(dockerPath),
			Metadata: map[string]interface{}{
				"type": "dockerfile",
			},
		}
		manifests = append(manifests, manifest)
	}

	for _, composePath := range composeFiles {
		relPath, _ := filepath.Rel(repoPath, composePath)
		manifest := PackageManifest{
			Ecosystem:       EcosystemDocker,
			ManifestPath:    relPath,
			DependencyCount: ps.countDockerComposeServices(composePath),
			Metadata: map[string]interface{}{
				"type": "docker-compose",
			},
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// countDockerBaseImages counts FROM instructions in Dockerfile
func (ps *PackageScanner) countDockerBaseImages(dockerPath string) int {
	// #nosec G304 -- dockerPath is validated via findFiles
	content, err := os.ReadFile(dockerPath)
	if err != nil {
		return 0
	}

	re := regexp.MustCompile(`(?mi)^\s*FROM\s+`)
	matches := re.FindAllString(string(content), -1)
	return len(matches)
}

// countDockerComposeServices counts services in docker-compose files (simplified)
func (ps *PackageScanner) countDockerComposeServices(composePath string) int {
	// #nosec G304 -- composePath is validated via findFiles
	content, err := os.ReadFile(composePath)
	if err != nil {
		return 0
	}

	// Count image: lines as a proxy for external service dependencies
	re := regexp.MustCompile(`(?m)^\s+image:\s*`)
	matches := re.FindAllString(string(content), -1)
	return len(matches)
}

// Helper functions

// isSkippedDir checks if a directory should be skipped during scanning
func isSkippedDir(name string) bool {
	switch name {
	case dirNodeModules, dirVendor, dirGit, dirPycache, dirTerraform, dirTarget, dirBin, dirObj, dirPackages:
		return true
	default:
		return false
	}
}

// findFiles finds all files with the exact name in the repository
func (ps *PackageScanner) findFiles(repoPath, filename string) []string {
	var files []string

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

		if info.Name() == filename {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		ps.logger.Debug("Error walking directory", "path", repoPath, "error", err)
	}

	return files
}

// findFilesWithPattern finds files matching a glob pattern
func (ps *PackageScanner) findFilesWithPattern(repoPath, pattern string) []string {
	var files []string

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

		matched, _ := filepath.Match(pattern, info.Name())
		if matched {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		ps.logger.Debug("Error walking directory", "path", repoPath, "error", err)
	}

	return files
}

// fileExists checks if a file exists
func (ps *PackageScanner) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
