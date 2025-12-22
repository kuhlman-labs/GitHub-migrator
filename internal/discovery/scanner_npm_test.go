package discovery

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestPackageScanner_ParsePackageJSONBasic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name             string
		content          string
		sourceHost       string
		additionalHosts  []string
		expectDeps       int
		expectFirstOwner string
		expectFirstRepo  string
		expectLocal      bool
	}{
		{
			name: "github shorthand",
			content: `{
	"dependencies": {
		"my-lib": "github:owner/repo"
	}
}`,
			additionalHosts:  []string{"github.com"},
			expectDeps:       1,
			expectFirstOwner: "owner",
			expectFirstRepo:  "repo",
			expectLocal:      false,
		},
		{
			name: "github shorthand with tag",
			content: `{
	"dependencies": {
		"my-lib": "github:owner/repo#v1.0.0"
	}
}`,
			additionalHosts:  []string{"github.com"},
			expectDeps:       1,
			expectFirstOwner: "owner",
			expectFirstRepo:  "repo",
			expectLocal:      false,
		},
		{
			name: "git+https url",
			content: `{
	"dependencies": {
		"my-lib": "git+https://github.com/owner/repo.git"
	}
}`,
			additionalHosts:  []string{"github.com"},
			expectDeps:       1,
			expectFirstOwner: "owner",
			expectFirstRepo:  "repo",
			expectLocal:      false,
		},
		{
			name: "npm registry packages - no github deps",
			content: `{
	"dependencies": {
		"lodash": "^4.17.21",
		"express": "~4.18.0"
	}
}`,
			additionalHosts: []string{"github.com"},
			expectDeps:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pkgPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(pkgPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			ps := &PackageScanner{
				logger:          logger,
				sourceHost:      tt.sourceHost,
				additionalHosts: tt.additionalHosts,
			}

			deps := ps.parsePackageJSON(pkgPath, "package.json")

			if len(deps) != tt.expectDeps {
				t.Errorf("Expected %d dependencies, got %d", tt.expectDeps, len(deps))
			}

			if tt.expectDeps > 0 && len(deps) > 0 {
				var found bool
				for _, dep := range deps {
					if dep.GitHubOwner == tt.expectFirstOwner && dep.GitHubRepo == tt.expectFirstRepo {
						found = true
						if dep.IsLocal != tt.expectLocal {
							t.Errorf("Expected IsLocal %v, got %v", tt.expectLocal, dep.IsLocal)
						}
						if dep.Ecosystem != EcosystemNodeJS {
							t.Errorf("Expected ecosystem %v, got %v", EcosystemNodeJS, dep.Ecosystem)
						}
						break
					}
				}
				if !found && tt.expectDeps == 1 {
					t.Errorf("Expected dependency %s/%s not found", tt.expectFirstOwner, tt.expectFirstRepo)
				}
			}
		})
	}
}

func TestPackageScanner_ExtractGitHubFromNpmVersionUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name            string
		version         string
		sourceHost      string
		additionalHosts []string
		expectOwner     string
		expectRepo      string
		expectHost      string
		expectLocal     bool
	}{
		{
			name:            "github shorthand",
			version:         "github:owner/repo",
			additionalHosts: []string{"github.com"},
			expectOwner:     "owner",
			expectRepo:      "repo",
			expectHost:      "github.com",
			expectLocal:     false,
		},
		{
			name:            "github shorthand with branch",
			version:         "github:owner/repo#main",
			additionalHosts: []string{"github.com"},
			expectOwner:     "owner",
			expectRepo:      "repo",
			expectHost:      "github.com",
			expectLocal:     false,
		},
		{
			name:            "git+https url",
			version:         "git+https://github.com/owner/repo.git",
			additionalHosts: []string{"github.com"},
			expectOwner:     "owner",
			expectRepo:      "repo",
			expectHost:      "github.com",
			expectLocal:     false,
		},
		{
			name:            "semver version - not a github dep",
			version:         "^1.0.0",
			additionalHosts: []string{"github.com"},
			expectOwner:     "",
			expectRepo:      "",
			expectHost:      "",
			expectLocal:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PackageScanner{
				logger:          logger,
				sourceHost:      tt.sourceHost,
				additionalHosts: tt.additionalHosts,
			}

			owner, repo, host, isLocal := ps.extractGitHubFromNpmVersion(tt.version)

			if owner != tt.expectOwner {
				t.Errorf("Expected owner %q, got %q", tt.expectOwner, owner)
			}
			if repo != tt.expectRepo {
				t.Errorf("Expected repo %q, got %q", tt.expectRepo, repo)
			}
			if host != tt.expectHost {
				t.Errorf("Expected host %q, got %q", tt.expectHost, host)
			}
			if isLocal != tt.expectLocal {
				t.Errorf("Expected isLocal %v, got %v", tt.expectLocal, isLocal)
			}
		})
	}
}

func TestPackageScanner_ExtractNpmGitHubShorthandUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ps := &PackageScanner{
		logger:          logger,
		additionalHosts: []string{"github.com"},
	}

	tests := []struct {
		name        string
		version     string
		expectOwner string
		expectRepo  string
	}{
		{
			name:        "basic shorthand",
			version:     "github:owner/repo",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "shorthand with tag",
			version:     "github:owner/repo#v1.0.0",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "not a github shorthand",
			version:     "^1.0.0",
			expectOwner: "",
			expectRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := ps.extractNpmGitHubShorthand(tt.version)
			if owner != tt.expectOwner {
				t.Errorf("Expected owner %q, got %q", tt.expectOwner, owner)
			}
			if repo != tt.expectRepo {
				t.Errorf("Expected repo %q, got %q", tt.expectRepo, repo)
			}
		})
	}
}

func TestPackageScanner_ExtractNpmGitURLUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name            string
		version         string
		sourceHost      string
		additionalHosts []string
		expectOwner     string
		expectRepo      string
		expectHost      string
		expectLocal     bool
	}{
		{
			name:            "git+https github",
			version:         "git+https://github.com/owner/repo.git",
			additionalHosts: []string{"github.com"},
			expectOwner:     "owner",
			expectRepo:      "repo",
			expectHost:      "github.com",
			expectLocal:     false,
		},
		{
			name:            "https github",
			version:         "https://github.com/owner/repo",
			additionalHosts: []string{"github.com"},
			expectOwner:     "owner",
			expectRepo:      "repo",
			expectHost:      "github.com",
			expectLocal:     false,
		},
		{
			name:            "untracked host",
			version:         "git+https://gitlab.com/owner/repo.git",
			additionalHosts: []string{"github.com"},
			expectOwner:     "",
			expectRepo:      "",
			expectHost:      "",
			expectLocal:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PackageScanner{
				logger:          logger,
				sourceHost:      tt.sourceHost,
				additionalHosts: tt.additionalHosts,
			}

			owner, repo, host, isLocal := ps.extractNpmGitURL(tt.version)

			if owner != tt.expectOwner {
				t.Errorf("Expected owner %q, got %q", tt.expectOwner, owner)
			}
			if repo != tt.expectRepo {
				t.Errorf("Expected repo %q, got %q", tt.expectRepo, repo)
			}
			if host != tt.expectHost {
				t.Errorf("Expected host %q, got %q", tt.expectHost, host)
			}
			if isLocal != tt.expectLocal {
				t.Errorf("Expected isLocal %v, got %v", tt.expectLocal, isLocal)
			}
		})
	}
}

func TestPackageScanner_ExtractNpmOwnerRepoShorthandUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ps := &PackageScanner{
		logger:          logger,
		additionalHosts: []string{"github.com"},
	}

	tests := []struct {
		name        string
		version     string
		expectOwner string
		expectRepo  string
	}{
		{
			name:        "basic shorthand",
			version:     "owner/repo",
			expectOwner: "owner",
			expectRepo:  "repo",
		},
		{
			name:        "semver with caret - not shorthand",
			version:     "^1.0.0",
			expectOwner: "",
			expectRepo:  "",
		},
		{
			name:        "scoped package - not shorthand",
			version:     "@scope/pkg",
			expectOwner: "",
			expectRepo:  "",
		},
		{
			name:        "multiple slashes - not shorthand",
			version:     "owner/repo/extra",
			expectOwner: "",
			expectRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := ps.extractNpmOwnerRepoShorthand(tt.version)
			if owner != tt.expectOwner {
				t.Errorf("Expected owner %q, got %q", tt.expectOwner, owner)
			}
			if repo != tt.expectRepo {
				t.Errorf("Expected repo %q, got %q", tt.expectRepo, repo)
			}
		})
	}
}

func TestPackageScanner_ParsePackageJSONFilesUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ps := &PackageScanner{
		logger:          logger,
		additionalHosts: []string{"github.com"},
	}

	tmpDir := t.TempDir()

	// Create first package.json
	pkg1 := filepath.Join(tmpDir, "pkg1", "package.json")
	if err := os.MkdirAll(filepath.Dir(pkg1), 0755); err != nil {
		t.Fatal(err)
	}
	pkg1Content := `{"dependencies": {"lib1": "github:owner1/repo1"}}`
	if err := os.WriteFile(pkg1, []byte(pkg1Content), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{pkg1}
	deps := ps.parsePackageJSONFiles(files, tmpDir)

	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	}
}

func TestPackageScanner_ParsePackageJSON_InvalidJSONUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ps := &PackageScanner{
		logger:          logger,
		additionalHosts: []string{"github.com"},
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgPath, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	deps := ps.parsePackageJSON(pkgPath, "package.json")

	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for invalid JSON, got %d", len(deps))
	}
}
