package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

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
