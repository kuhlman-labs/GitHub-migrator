package discovery

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestPackageScanner_ParseGoModBasic(t *testing.T) {
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
			name: "simple require block with github.com",
			content: `module example.com/myproject

go 1.21

require (
	github.com/owner/repo v1.0.0
	github.com/another/pkg v2.0.0
)
`,
			sourceHost:       "",
			additionalHosts:  []string{"github.com"},
			expectDeps:       2,
			expectFirstOwner: "owner",
			expectFirstRepo:  "repo",
			expectLocal:      false,
		},
		{
			name: "single require statement",
			content: `module example.com/myproject

go 1.21

require github.com/owner/repo v1.0.0
`,
			sourceHost:       "",
			additionalHosts:  []string{"github.com"},
			expectDeps:       1,
			expectFirstOwner: "owner",
			expectFirstRepo:  "repo",
			expectLocal:      false,
		},
		{
			name: "azure devops go module",
			content: `module example.com/myproject

go 1.21

require (
	dev.azure.com/myorg/myproject/_git/myrepo.git v0.0.0-20230101000000-abcdef123456
)
`,
			sourceHost:       "dev.azure.com",
			additionalHosts:  []string{"github.com"},
			expectDeps:       1,
			expectFirstOwner: "myorg/myproject",
			expectFirstRepo:  "myrepo",
			expectLocal:      false, // sourceOrg not set
		},
		{
			name: "empty go.mod",
			content: `module example.com/myproject

go 1.21
`,
			sourceHost:      "",
			additionalHosts: []string{"github.com"},
			expectDeps:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and go.mod file
			tmpDir := t.TempDir()
			modPath := filepath.Join(tmpDir, "go.mod")
			if err := os.WriteFile(modPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			ps := &PackageScanner{
				logger:          logger,
				sourceHost:      tt.sourceHost,
				additionalHosts: tt.additionalHosts,
			}

			deps := ps.parseGoMod(modPath, "go.mod")

			if len(deps) != tt.expectDeps {
				t.Errorf("Expected %d dependencies, got %d", tt.expectDeps, len(deps))
				for _, d := range deps {
					t.Logf("  Dep: %s/%s", d.GitHubOwner, d.GitHubRepo)
				}
			}

			if tt.expectDeps > 0 && len(deps) > 0 {
				if deps[0].GitHubOwner != tt.expectFirstOwner {
					t.Errorf("Expected first owner %q, got %q", tt.expectFirstOwner, deps[0].GitHubOwner)
				}
				if deps[0].GitHubRepo != tt.expectFirstRepo {
					t.Errorf("Expected first repo %q, got %q", tt.expectFirstRepo, deps[0].GitHubRepo)
				}
				if deps[0].IsLocal != tt.expectLocal {
					t.Errorf("Expected IsLocal %v, got %v", tt.expectLocal, deps[0].IsLocal)
				}
				if deps[0].Ecosystem != EcosystemGo {
					t.Errorf("Expected ecosystem %v, got %v", EcosystemGo, deps[0].Ecosystem)
				}
			}
		})
	}
}

func TestPackageScanner_ExtractGoModulePathUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ps := &PackageScanner{
		logger:          logger,
		additionalHosts: []string{"github.com"},
	}

	tests := []struct {
		name           string
		line           string
		inRequireBlock bool
		expectModule   string
		expectVersion  string
	}{
		{
			name:           "in require block",
			line:           "github.com/owner/repo v1.0.0",
			inRequireBlock: true,
			expectModule:   "github.com/owner/repo",
			expectVersion:  "v1.0.0",
		},
		{
			name:           "single require statement",
			line:           "require github.com/owner/repo v1.0.0",
			inRequireBlock: false,
			expectModule:   "github.com/owner/repo",
			expectVersion:  "v1.0.0",
		},
		{
			name:           "comment in require block",
			line:           "// this is a comment",
			inRequireBlock: true,
			expectModule:   "",
			expectVersion:  "",
		},
		{
			name:           "empty line in require block",
			line:           "",
			inRequireBlock: true,
			expectModule:   "",
			expectVersion:  "",
		},
		{
			name:           "require with parens - should not match",
			line:           "require (",
			inRequireBlock: false,
			expectModule:   "",
			expectVersion:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, version := ps.extractGoModulePath(tt.line, tt.inRequireBlock)
			if module != tt.expectModule {
				t.Errorf("Expected module %q, got %q", tt.expectModule, module)
			}
			if version != tt.expectVersion {
				t.Errorf("Expected version %q, got %q", tt.expectVersion, version)
			}
		})
	}
}

func TestPackageScanner_MatchGoModuleToHostUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name            string
		modulePath      string
		version         string
		sourceHost      string
		additionalHosts []string
		expectNil       bool
		expectOwner     string
		expectRepo      string
		expectLocal     bool
	}{
		{
			name:            "github.com module",
			modulePath:      "github.com/owner/repo",
			version:         "v1.0.0",
			sourceHost:      "",
			additionalHosts: []string{"github.com"},
			expectNil:       false,
			expectOwner:     "owner",
			expectRepo:      "repo",
			expectLocal:     false,
		},
		{
			name:            "github.com module with subpath",
			modulePath:      "github.com/owner/repo/subpkg",
			version:         "v1.0.0",
			sourceHost:      "",
			additionalHosts: []string{"github.com"},
			expectNil:       false,
			expectOwner:     "owner",
			expectRepo:      "repo",
			expectLocal:     false,
		},
		{
			name:            "enterprise host - local",
			modulePath:      "ghes.example.com/org/lib",
			version:         "v2.0.0",
			sourceHost:      "ghes.example.com",
			additionalHosts: []string{"github.com", "ghes.example.com"},
			expectNil:       false,
			expectOwner:     "org",
			expectRepo:      "lib",
			expectLocal:     true,
		},
		{
			name:            "non-tracked host",
			modulePath:      "gitlab.com/owner/repo",
			version:         "v1.0.0",
			sourceHost:      "",
			additionalHosts: []string{"github.com"},
			expectNil:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PackageScanner{
				logger:          logger,
				sourceHost:      tt.sourceHost,
				additionalHosts: tt.additionalHosts,
			}

			result := ps.matchGoModuleToHost(tt.modulePath, tt.version, "go.mod")

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil result, got %+v", result)
				}
			} else {
				if result == nil {
					t.Fatal("Expected non-nil result, got nil")
					return
				}
				if result.GitHubOwner != tt.expectOwner {
					t.Errorf("Expected owner %q, got %q", tt.expectOwner, result.GitHubOwner)
				}
				if result.GitHubRepo != tt.expectRepo {
					t.Errorf("Expected repo %q, got %q", tt.expectRepo, result.GitHubRepo)
				}
				if result.IsLocal != tt.expectLocal {
					t.Errorf("Expected IsLocal %v, got %v", tt.expectLocal, result.IsLocal)
				}
			}
		})
	}
}

func TestPackageScanner_MatchGoModuleToADOUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name        string
		modulePath  string
		version     string
		sourceOrg   string
		isADOSource bool
		expectNil   bool
		expectOwner string
		expectRepo  string
		expectLocal bool
	}{
		{
			name:        "valid ADO module path",
			modulePath:  "dev.azure.com/myorg/myproject/_git/myrepo.git",
			version:     "v0.0.0-20230101000000-abc123",
			isADOSource: false,
			expectNil:   false,
			expectOwner: "myorg/myproject",
			expectRepo:  "myrepo",
			expectLocal: false,
		},
		{
			name:        "ADO module - local",
			modulePath:  "dev.azure.com/localorg/project/_git/lib.git",
			version:     "v1.0.0",
			sourceOrg:   "localorg",
			isADOSource: true,
			expectNil:   false,
			expectOwner: "localorg/project",
			expectRepo:  "lib",
			expectLocal: true,
		},
		{
			name:        "non-ADO module path",
			modulePath:  "github.com/owner/repo",
			version:     "v1.0.0",
			isADOSource: false,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PackageScanner{
				logger:          logger,
				sourceOrg:       tt.sourceOrg,
				isADOSource:     tt.isADOSource,
				additionalHosts: []string{"github.com"},
			}

			result := ps.matchGoModuleToADO(tt.modulePath, tt.version, "go.mod")

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil result, got %+v", result)
				}
			} else {
				if result == nil {
					t.Fatal("Expected non-nil result, got nil")
					return
				}
				if result.GitHubOwner != tt.expectOwner {
					t.Errorf("Expected owner %q, got %q", tt.expectOwner, result.GitHubOwner)
				}
				if result.GitHubRepo != tt.expectRepo {
					t.Errorf("Expected repo %q, got %q", tt.expectRepo, result.GitHubRepo)
				}
				if result.IsLocal != tt.expectLocal {
					t.Errorf("Expected IsLocal %v, got %v", tt.expectLocal, result.IsLocal)
				}
			}
		})
	}
}

func TestPackageScanner_ParseGoModFilesUnit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ps := &PackageScanner{
		logger:          logger,
		additionalHosts: []string{"github.com"},
	}

	tmpDir := t.TempDir()

	// Create first go.mod
	mod1 := filepath.Join(tmpDir, "mod1", "go.mod")
	if err := os.MkdirAll(filepath.Dir(mod1), 0750); err != nil {
		t.Fatal(err)
	}
	mod1Content := `module example.com/mod1
go 1.21
require github.com/owner1/repo1 v1.0.0
`
	if err := os.WriteFile(mod1, []byte(mod1Content), 0600); err != nil {
		t.Fatal(err)
	}

	// Create second go.mod
	mod2 := filepath.Join(tmpDir, "mod2", "go.mod")
	if err := os.MkdirAll(filepath.Dir(mod2), 0750); err != nil {
		t.Fatal(err)
	}
	mod2Content := `module example.com/mod2
go 1.21
require (
	github.com/owner2/repo2 v2.0.0
	github.com/owner2/repo3 v2.1.0
)
`
	if err := os.WriteFile(mod2, []byte(mod2Content), 0600); err != nil {
		t.Fatal(err)
	}

	files := []string{mod1, mod2}
	deps := ps.parseGoModFiles(files, tmpDir)

	if len(deps) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(deps))
	}

	// Verify dependencies have correct manifest paths (relative)
	for _, dep := range deps {
		if dep.Manifest != "mod1/go.mod" && dep.Manifest != "mod2/go.mod" {
			t.Errorf("Unexpected manifest path: %s", dep.Manifest)
		}
	}
}
