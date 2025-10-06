package discovery

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/models"
)

func TestNewAnalyzer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	if analyzer == nil {
		t.Fatal("NewAnalyzer returned nil")
	}
	if analyzer.logger == nil {
		t.Error("Analyzer logger is nil")
	}
}

func TestDetectLFS(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)
	ctx := context.Background()

	// Create a temporary directory with .gitattributes
	tempDir := t.TempDir()

	t.Run("No LFS - empty directory", func(t *testing.T) {
		hasLFS := analyzer.detectLFS(ctx, tempDir)
		if hasLFS {
			t.Error("Expected no LFS, but detected LFS")
		}
	})

	t.Run("LFS detected via .gitattributes", func(t *testing.T) {
		gitattributes := filepath.Join(tempDir, ".gitattributes")
		content := "*.psd filter=lfs diff=lfs merge=lfs -text\n"
		if err := os.WriteFile(gitattributes, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .gitattributes: %v", err)
		}

		hasLFS := analyzer.detectLFS(ctx, tempDir)
		if !hasLFS {
			t.Error("Expected LFS to be detected via .gitattributes")
		}
	})
}

func TestDetectSubmodules(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)
	ctx := context.Background()

	tempDir := t.TempDir()

	t.Run("No submodules - empty directory", func(t *testing.T) {
		hasSubmodules := analyzer.detectSubmodules(ctx, tempDir)
		if hasSubmodules {
			t.Error("Expected no submodules, but detected submodules")
		}
	})

	t.Run("Submodules detected via .gitmodules", func(t *testing.T) {
		gitmodules := filepath.Join(tempDir, ".gitmodules")
		content := "[submodule \"vendor/lib\"]\n\tpath = vendor/lib\n\turl = https://github.com/example/lib.git\n"
		if err := os.WriteFile(gitmodules, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .gitmodules: %v", err)
		}

		hasSubmodules := analyzer.detectSubmodules(ctx, tempDir)
		if !hasSubmodules {
			t.Error("Expected submodules to be detected via .gitmodules")
		}
	})
}

func TestGetBranchCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)
	ctx := context.Background()

	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Configure git user
	exec.Command("git", "config", "user.name", "Test User").Dir = tempDir
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tempDir

	// Create an initial commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Skip("Failed to create initial commit, skipping")
	}

	count := analyzer.getBranchCount(ctx, tempDir)
	// Should have at least 0 (no remote branches in a fresh repo)
	if count < 0 {
		t.Errorf("Expected non-negative branch count, got %d", count)
	}
}

func TestCheckRepositoryProblems(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	tests := []struct {
		name     string
		output   *GitSizerOutput
		expected int // expected number of problems
	}{
		{
			name: "No problems",
			output: &GitSizerOutput{
				MaxBlobSize:      1024 * 1024,       // 1MB
				UniqueBlobSize:   100 * 1024 * 1024, // 100MB
				UniqueTreeSize:   10 * 1024 * 1024,  // 10MB
				UniqueCommitSize: 1 * 1024 * 1024,   // 1MB
				MaxHistoryDepth:  1000,
				MaxTreeEntries:   100,
				MaxFileCount:     1000,
			},
			expected: 0,
		},
		{
			name: "Large blob problem",
			output: &GitSizerOutput{
				MaxBlobSize:      100 * 1024 * 1024, // 100MB
				UniqueBlobSize:   100 * 1024 * 1024,
				UniqueTreeSize:   10 * 1024 * 1024,
				UniqueCommitSize: 1 * 1024 * 1024,
				MaxHistoryDepth:  1000,
				MaxTreeEntries:   100,
				MaxFileCount:     1000,
			},
			expected: 1,
		},
		{
			name: "Large repository problem",
			output: &GitSizerOutput{
				MaxBlobSize:      1024 * 1024,
				UniqueBlobSize:   6 * 1024 * 1024 * 1024, // 6GB
				UniqueTreeSize:   10 * 1024 * 1024,
				UniqueCommitSize: 1 * 1024 * 1024,
				MaxHistoryDepth:  1000,
				MaxTreeEntries:   100,
				MaxFileCount:     1000,
			},
			expected: 1,
		},
		{
			name: "Multiple problems",
			output: &GitSizerOutput{
				MaxBlobSize:      100 * 1024 * 1024,      // 100MB
				UniqueBlobSize:   6 * 1024 * 1024 * 1024, // 6GB
				UniqueTreeSize:   10 * 1024 * 1024,
				UniqueCommitSize: 1 * 1024 * 1024,
				MaxHistoryDepth:  150000,
				MaxTreeEntries:   15000,
				MaxFileCount:     150000,
			},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := analyzer.CheckRepositoryProblems(tt.output)
			if len(problems) != tt.expected {
				t.Errorf("Expected %d problems, got %d: %v", tt.expected, len(problems), problems)
			}
		})
	}
}

func TestAnalyzeGitProperties_Integration(t *testing.T) {
	// Skip if GITHUB_TEST_INTEGRATION is not set
	if os.Getenv("GITHUB_TEST_INTEGRATION") == "" {
		t.Skip("Skipping integration test (set GITHUB_TEST_INTEGRATION=1 to run)")
	}

	// Check if git-sizer is available
	if _, err := exec.LookPath("git-sizer"); err != nil {
		t.Skip("git-sizer not available, skipping integration test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)
	ctx := context.Background()

	// Create a test repository
	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Dir = tempDir
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tempDir

	// Create a test file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test analysis
	repo := &models.Repository{
		FullName: "test/repo",
	}

	if err := analyzer.AnalyzeGitProperties(ctx, repo, tempDir); err != nil {
		t.Fatalf("AnalyzeGitProperties failed: %v", err)
	}

	// Verify results
	if repo.TotalSize == nil || *repo.TotalSize == 0 {
		t.Error("Expected non-zero total size")
	}
	if repo.CommitCount == 0 {
		t.Error("Expected non-zero commit count")
	}
	if repo.BranchCount < 0 {
		t.Error("Expected non-negative branch count")
	}
}
