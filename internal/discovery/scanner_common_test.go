package discovery

import (
	"testing"
)

func TestIsSkippedDir(t *testing.T) {
	tests := []struct {
		name    string
		dirName string
		want    bool
	}{
		{"node_modules", dirNodeModules, true},
		{"vendor", dirVendor, true},
		{".git", dirGit, true},
		{"__pycache__", dirPycache, true},
		{".terraform", dirTerraform, true},
		{"target", dirTarget, true},
		{"bin", dirBin, true},
		{"obj", dirObj, true},
		{"packages", dirPackages, true},
		{"src", "src", false},
		{"lib", "lib", false},
		{"app", "app", false},
		{"tests", "tests", false},
		{"internal", "internal", false},
		{"pkg", "pkg", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSkippedDir(tt.dirName)
			if got != tt.want {
				t.Errorf("isSkippedDir(%q) = %v, want %v", tt.dirName, got, tt.want)
			}
		})
	}
}

func TestManifestFileConstants(t *testing.T) {
	// Verify manifest file constants have expected values
	constants := map[string]string{
		"fileGoMod":          fileGoMod,
		"filePackageJSON":    filePackageJSON,
		"fileRequirements":   fileRequirements,
		"fileGemfile":        fileGemfile,
		"fileCargoToml":      fileCargoToml,
		"fileChartYaml":      fileChartYaml,
		"filePackageSwift":   filePackageSwift,
		"fileMixExs":         fileMixExs,
		"fileBuildGradle":    fileBuildGradle,
		"fileBuildGradleKts": fileBuildGradleKts,
	}

	expected := map[string]string{
		"fileGoMod":          "go.mod",
		"filePackageJSON":    "package.json",
		"fileRequirements":   "requirements.txt",
		"fileGemfile":        "Gemfile",
		"fileCargoToml":      "Cargo.toml",
		"fileChartYaml":      "Chart.yaml",
		"filePackageSwift":   "Package.swift",
		"fileMixExs":         "mix.exs",
		"fileBuildGradle":    "build.gradle",
		"fileBuildGradleKts": "build.gradle.kts",
	}

	for name, got := range constants {
		want := expected[name]
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestIsADOURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://dev.azure.com/org/project/_git/repo", true},
		{"https://myorg@dev.azure.com/myorg/project/_git/repo", true},
		{"https://org.visualstudio.com/project/_git/repo", true},
		{"git@ssh.dev.azure.com:v3/org/project/repo", true},
		{"https://github.com/owner/repo", false},
		{"https://gitlab.com/owner/repo", false},
		{"git@github.com:owner/repo.git", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isADOURL(tt.url)
			if got != tt.want {
				t.Errorf("isADOURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsADOHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"dev.azure.com", true},
		{"org.visualstudio.com", true},
		{"mycompany.visualstudio.com", true},
		{"ssh.dev.azure.com", true},
		{"github.com", false},
		{"gitlab.com", false},
		{"bitbucket.org", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isADOHost(tt.host)
			if got != tt.want {
				t.Errorf("isADOHost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestManifestFiles_Structure(t *testing.T) {
	// Test that ManifestFiles can hold all manifest types
	manifests := &ManifestFiles{
		GoMod:                []string{"/path/to/go.mod"},
		PackageJSON:          []string{"/path/to/package.json"},
		Requirements:         []string{"/path/to/requirements.txt"},
		RequirementsVariants: []string{"/path/to/requirements-dev.txt"},
		Gemfile:              []string{"/path/to/Gemfile"},
		CargoToml:            []string{"/path/to/Cargo.toml"},
		ChartYaml:            []string{"/path/to/Chart.yaml"},
		PackageSwift:         []string{"/path/to/Package.swift"},
		MixExs:               []string{"/path/to/mix.exs"},
		BuildGradle:          []string{"/path/to/build.gradle"},
		BuildGradleKts:       []string{"/path/to/build.gradle.kts"},
		Terraform:            []string{"/path/to/main.tf"},
	}

	if len(manifests.GoMod) != 1 {
		t.Errorf("GoMod count = %d, want 1", len(manifests.GoMod))
	}
	if len(manifests.PackageJSON) != 1 {
		t.Errorf("PackageJSON count = %d, want 1", len(manifests.PackageJSON))
	}
	if len(manifests.Terraform) != 1 {
		t.Errorf("Terraform count = %d, want 1", len(manifests.Terraform))
	}
}

func TestExtractedDependency_Structure(t *testing.T) {
	dep := ExtractedDependency{
		Name:         "owner/repo",
		Ecosystem:    EcosystemGo,
		Manifest:     "/path/to/go.mod",
		IsGitHubRepo: true,
		GitHubOwner:  "owner",
		GitHubRepo:   "repo",
		IsLocal:      true,
		SourceHost:   "github.com",
	}

	if dep.Name != "owner/repo" {
		t.Errorf("Name = %q, want %q", dep.Name, "owner/repo")
	}
	if dep.Ecosystem != EcosystemGo {
		t.Errorf("Ecosystem = %q, want %q", dep.Ecosystem, EcosystemGo)
	}
	if !dep.IsGitHubRepo {
		t.Error("IsGitHubRepo should be true")
	}
	if !dep.IsLocal {
		t.Error("IsLocal should be true")
	}
}

func TestPackageEcosystemConstants(t *testing.T) {
	ecosystems := []PackageEcosystem{
		EcosystemGo,
		EcosystemNodeJS,
		EcosystemPython,
		EcosystemRuby,
		EcosystemRust,
		EcosystemHelm,
		EcosystemSwift,
		EcosystemElixir,
		EcosystemGradle,
		EcosystemTerraform,
	}

	for _, eco := range ecosystems {
		if eco == "" {
			t.Error("Package ecosystem should not be empty")
		}
	}
}

func TestSkippedDirConstants(t *testing.T) {
	// Verify the constants exist and have expected values
	if dirNodeModules != "node_modules" {
		t.Errorf("dirNodeModules = %q, want %q", dirNodeModules, "node_modules")
	}
	if dirVendor != "vendor" {
		t.Errorf("dirVendor = %q, want %q", dirVendor, "vendor")
	}
	if dirGit != ".git" {
		t.Errorf("dirGit = %q, want %q", dirGit, ".git")
	}
	if dirPycache != "__pycache__" {
		t.Errorf("dirPycache = %q, want %q", dirPycache, "__pycache__")
	}
	if dirTerraform != ".terraform" {
		t.Errorf("dirTerraform = %q, want %q", dirTerraform, ".terraform")
	}
}
