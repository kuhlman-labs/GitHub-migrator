package discovery

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDependencyAnalyzer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewDependencyAnalyzer(logger)

	if analyzer == nil {
		t.Fatal("NewDependencyAnalyzer returned nil")
		return // Explicitly unreachable, but satisfies static analysis
	}
	if analyzer.logger == nil {
		t.Error("analyzer.logger is nil")
	}
}

func TestExtractGitHubRepoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/owner/repo",
			expected: "owner/repo",
			wantErr:  false,
		},
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/owner/repo.git",
			expected: "owner/repo",
			wantErr:  false,
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:owner/repo.git",
			expected: "owner/repo",
			wantErr:  false,
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:owner/repo",
			expected: "owner/repo",
			wantErr:  false,
		},
		{
			name:     "Git protocol URL",
			url:      "git://github.com/owner/repo.git",
			expected: "owner/repo",
			wantErr:  false,
		},
		{
			name:     "Enterprise HTTPS URL",
			url:      "https://github.enterprise.com/owner/repo",
			expected: "owner/repo",
			wantErr:  false,
		},
		{
			name:     "Enterprise SSH URL",
			url:      "git@github.enterprise.com:owner/repo.git",
			expected: "owner/repo",
			wantErr:  false,
		},
		{
			name:    "Non-GitHub URL",
			url:     "https://gitlab.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "Invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractGitHubRepoFromURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractSubmodules(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	analyzer := NewDependencyAnalyzer(logger)
	ctx := context.Background()

	t.Run("no .gitmodules file", func(t *testing.T) {
		tempDir := t.TempDir()
		submodules, err := analyzer.ExtractSubmodules(ctx, tempDir)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(submodules) > 0 {
			t.Errorf("Expected no submodules, got %d", len(submodules))
		}
	})

	t.Run("with .gitmodules file", func(t *testing.T) {
		tempDir := t.TempDir()
		gitmodulesContent := `[submodule "lib/submodule1"]
	path = lib/submodule1
	url = https://github.com/owner/submodule1.git
	branch = main
[submodule "lib/submodule2"]
	path = lib/submodule2
	url = git@github.com:owner/submodule2.git
`
		gitmodulesPath := filepath.Join(tempDir, ".gitmodules")
		if err := os.WriteFile(gitmodulesPath, []byte(gitmodulesContent), 0600); err != nil {
			t.Fatalf("Failed to create .gitmodules: %v", err)
		}

		submodules, err := analyzer.ExtractSubmodules(ctx, tempDir)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(submodules) != 2 {
			t.Errorf("Expected 2 submodules, got %d", len(submodules))
		}

		if len(submodules) > 0 {
			if submodules[0].Path != "lib/submodule1" {
				t.Errorf("Expected path 'lib/submodule1', got '%s'", submodules[0].Path)
			}
			if submodules[0].URL != "https://github.com/owner/submodule1.git" {
				t.Errorf("Expected URL 'https://github.com/owner/submodule1.git', got '%s'", submodules[0].URL)
			}
			if submodules[0].Branch != "main" {
				t.Errorf("Expected branch 'main', got '%s'", submodules[0].Branch)
			}
		}
	})
}

//nolint:gocyclo // Table-driven test with multiple subtests and assertions
func TestExtractWorkflowDependencies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	analyzer := NewDependencyAnalyzer(logger)
	ctx := context.Background()

	t.Run("no workflows directory", func(t *testing.T) {
		tempDir := t.TempDir()
		deps, err := analyzer.ExtractWorkflowDependencies(ctx, tempDir)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(deps) > 0 {
			t.Errorf("Expected no dependencies, got %d", len(deps))
		}
	})

	t.Run("with workflow files", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowsDir := filepath.Join(tempDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("Failed to create workflows dir: %v", err)
		}

		workflowContent := `name: CI
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go build
  call-workflow:
    uses: owner/repo/.github/workflows/reusable.yml@main
`
		workflowPath := filepath.Join(workflowsDir, "ci.yml")
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0600); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		deps, err := analyzer.ExtractWorkflowDependencies(ctx, tempDir)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(deps) != 3 {
			t.Errorf("Expected 3 dependencies, got %d", len(deps))
		}

		// Check for specific dependencies
		foundCheckout := false
		foundSetupGo := false
		foundReusable := false
		for _, dep := range deps {
			if dep.Repository == "actions/checkout" {
				foundCheckout = true
				if dep.Ref != "v4" {
					t.Errorf("Expected checkout ref 'v4', got '%s'", dep.Ref)
				}
			}
			if dep.Repository == "actions/setup-go" {
				foundSetupGo = true
				if dep.Ref != "v5" {
					t.Errorf("Expected setup-go ref 'v5', got '%s'", dep.Ref)
				}
			}
			if dep.Repository == "owner/repo" {
				foundReusable = true
				if dep.WorkflowPath != ".github/workflows/reusable.yml" {
					t.Errorf("Expected workflow path '.github/workflows/reusable.yml', got '%s'", dep.WorkflowPath)
				}
			}
		}

		if !foundCheckout {
			t.Error("Expected to find actions/checkout dependency")
		}
		if !foundSetupGo {
			t.Error("Expected to find actions/setup-go dependency")
		}
		if !foundReusable {
			t.Error("Expected to find reusable workflow dependency")
		}
	})
}

func TestParseUsesString(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	analyzer := NewDependencyAnalyzer(logger)

	tests := []struct {
		name     string
		uses     string
		wantRepo string
		wantRef  string
		wantPath string
		wantNil  bool
	}{
		{
			name:     "action with tag",
			uses:     "actions/checkout@v4",
			wantRepo: "actions/checkout",
			wantRef:  "v4",
			wantPath: "",
			wantNil:  false,
		},
		{
			name:     "action with SHA",
			uses:     "actions/checkout@abc123def456",
			wantRepo: "actions/checkout",
			wantRef:  "abc123def456",
			wantPath: "",
			wantNil:  false,
		},
		{
			name:     "reusable workflow",
			uses:     "owner/repo/.github/workflows/build.yml@main",
			wantRepo: "owner/repo",
			wantRef:  "main",
			wantPath: ".github/workflows/build.yml",
			wantNil:  false,
		},
		{
			name:    "no @ separator",
			uses:    "actions/checkout",
			wantNil: true,
		},
		{
			name:    "only owner",
			uses:    "actions@v1",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.parseUsesString(tt.uses, "test.yml")

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result, got nil")
				return // Explicitly unreachable, but satisfies static analysis
			}

			if result.Repository != tt.wantRepo {
				t.Errorf("Expected repo '%s', got '%s'", tt.wantRepo, result.Repository)
			}
			if result.Ref != tt.wantRef {
				t.Errorf("Expected ref '%s', got '%s'", tt.wantRef, result.Ref)
			}
			if result.WorkflowPath != tt.wantPath {
				t.Errorf("Expected path '%s', got '%s'", tt.wantPath, result.WorkflowPath)
			}
		})
	}
}

func TestIsRepositoryReference(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	analyzer := NewDependencyAnalyzer(logger)

	tests := []struct {
		uses     string
		expected bool
	}{
		{"actions/checkout@v4", true},
		{"owner/repo@main", true},
		{"docker://my-image:latest", false},
		{"./local/action", false},
		{"single-word", false},
	}

	for _, tt := range tests {
		t.Run(tt.uses, func(t *testing.T) {
			result := analyzer.isRepositoryReference(tt.uses)
			if result != tt.expected {
				t.Errorf("Expected %v for '%s', got %v", tt.expected, tt.uses, result)
			}
		})
	}
}

func TestSubmoduleInfo(t *testing.T) {
	info := SubmoduleInfo{
		Path:   "lib/submodule",
		URL:    "https://github.com/owner/submodule.git",
		Branch: "main",
	}

	if info.Path != "lib/submodule" {
		t.Errorf("Expected path 'lib/submodule', got '%s'", info.Path)
	}
	if info.URL != "https://github.com/owner/submodule.git" {
		t.Errorf("Expected URL 'https://github.com/owner/submodule.git', got '%s'", info.URL)
	}
	if info.Branch != "main" {
		t.Errorf("Expected branch 'main', got '%s'", info.Branch)
	}
}

func TestWorkflowDependency(t *testing.T) {
	dep := WorkflowDependency{
		WorkflowFile: "ci.yml",
		Uses:         "actions/checkout@v4",
		Repository:   "actions/checkout",
		Ref:          "v4",
		WorkflowPath: "",
	}

	if dep.WorkflowFile != "ci.yml" {
		t.Errorf("Expected workflow file 'ci.yml', got '%s'", dep.WorkflowFile)
	}
	if dep.Repository != "actions/checkout" {
		t.Errorf("Expected repository 'actions/checkout', got '%s'", dep.Repository)
	}
	if dep.Ref != "v4" {
		t.Errorf("Expected ref 'v4', got '%s'", dep.Ref)
	}
}
