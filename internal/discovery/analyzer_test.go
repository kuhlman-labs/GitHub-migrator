package discovery

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestNewAnalyzer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	if analyzer == nil {
		t.Fatal("NewAnalyzer returned nil")
		return
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

	t.Run("LFS detected via .git/config", func(t *testing.T) {
		// Create a new temp directory for this test
		tempDir2 := t.TempDir()

		// Create .git directory
		gitDir := filepath.Join(tempDir2, ".git")
		if err := os.Mkdir(gitDir, 0755); err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		// Create .git/config with LFS filter configuration
		gitConfig := filepath.Join(gitDir, "config")
		configContent := `[core]
	repositoryformatversion = 0
	filemode = true
[filter "lfs"]
	clean = git-lfs clean -- %f
	smudge = git-lfs smudge -- %f
	process = git-lfs filter-process
	required = true
`
		if err := os.WriteFile(gitConfig, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create .git/config: %v", err)
		}

		hasLFS := analyzer.detectLFS(ctx, tempDir2)
		if !hasLFS {
			t.Error("Expected LFS to be detected via .git/config")
		}
	})
}

func TestDetectLFSPointerFiles(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)
	ctx := context.Background()

	t.Run("No LFS pointer files", func(t *testing.T) {
		// Create a temporary git repository
		tempDir := t.TempDir()

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Skip("git not available, skipping test")
		}

		// Configure git user
		cmd = exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = tempDir
		cmd.Run()

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tempDir
		cmd.Run()

		// Create a regular file and commit
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("regular file content"), 0644); err != nil {
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

		hasLFS := analyzer.detectLFSPointerFiles(ctx, tempDir)
		if hasLFS {
			t.Error("Expected no LFS pointer files to be detected")
		}
	})

	t.Run("LFS pointer file detected", func(t *testing.T) {
		// Create a temporary git repository
		tempDir := t.TempDir()

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Skip("git not available, skipping test")
		}

		// Configure git user
		cmd = exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = tempDir
		cmd.Run()

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tempDir
		cmd.Run()

		// Create a file with LFS pointer content
		lfsPointerFile := filepath.Join(tempDir, "large-file.bin")
		lfsPointerContent := `version https://git-lfs.github.com/spec/v1
oid sha256:4d70b55500e395744f43477146522c019d3f11d132a9a834927b5f63901b0f5b
size 123456
`
		if err := os.WriteFile(lfsPointerFile, []byte(lfsPointerContent), 0644); err != nil {
			t.Fatalf("Failed to create LFS pointer file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		cmd.Run()

		cmd = exec.Command("git", "commit", "-m", "Add LFS pointer")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Skip("Failed to create commit with LFS pointer, skipping")
		}

		hasLFS := analyzer.detectLFSPointerFiles(ctx, tempDir)
		if !hasLFS {
			t.Error("Expected LFS pointer files to be detected")
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

	t.Run("Submodules detected via .git/config", func(t *testing.T) {
		// Create a new temp directory for this test
		tempDir2 := t.TempDir()

		// Create .git directory
		gitDir := filepath.Join(tempDir2, ".git")
		if err := os.Mkdir(gitDir, 0755); err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		// Create .git/config with submodule configuration
		gitConfig := filepath.Join(gitDir, "config")
		configContent := `[core]
	repositoryformatversion = 0
	filemode = true
[submodule "vendor/lib"]
	url = https://github.com/example/lib.git
	active = true
`
		if err := os.WriteFile(gitConfig, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create .git/config: %v", err)
		}

		hasSubmodules := analyzer.detectSubmodules(ctx, tempDir2)
		if !hasSubmodules {
			t.Error("Expected submodules to be detected via .git/config")
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
				MaxBlobSize:          GitSizerMetric{Value: 1024 * 1024},       // 1MB
				UniqueBlobSize:       GitSizerMetric{Value: 100 * 1024 * 1024}, // 100MB
				UniqueTreeSize:       GitSizerMetric{Value: 10 * 1024 * 1024},  // 10MB
				UniqueCommitSize:     GitSizerMetric{Value: 1 * 1024 * 1024},   // 1MB
				MaxHistoryDepth:      GitSizerMetric{Value: 1000},
				MaxTreeEntries:       GitSizerMetric{Value: 100},
				MaxCheckoutBlobCount: GitSizerMetric{Value: 1000},
			},
			expected: 0,
		},
		{
			name: "Large blob problem",
			output: &GitSizerOutput{
				MaxBlobSize:          GitSizerMetric{Value: 100 * 1024 * 1024}, // 100MB
				UniqueBlobSize:       GitSizerMetric{Value: 100 * 1024 * 1024},
				UniqueTreeSize:       GitSizerMetric{Value: 10 * 1024 * 1024},
				UniqueCommitSize:     GitSizerMetric{Value: 1 * 1024 * 1024},
				MaxHistoryDepth:      GitSizerMetric{Value: 1000},
				MaxTreeEntries:       GitSizerMetric{Value: 100},
				MaxCheckoutBlobCount: GitSizerMetric{Value: 1000},
			},
			expected: 1,
		},
		{
			name: "Large repository problem",
			output: &GitSizerOutput{
				MaxBlobSize:          GitSizerMetric{Value: 1024 * 1024},
				UniqueBlobSize:       GitSizerMetric{Value: 6 * 1024 * 1024 * 1024}, // 6GB
				UniqueTreeSize:       GitSizerMetric{Value: 10 * 1024 * 1024},
				UniqueCommitSize:     GitSizerMetric{Value: 1 * 1024 * 1024},
				MaxHistoryDepth:      GitSizerMetric{Value: 1000},
				MaxTreeEntries:       GitSizerMetric{Value: 100},
				MaxCheckoutBlobCount: GitSizerMetric{Value: 1000},
			},
			expected: 1,
		},
		{
			name: "Large commit problem",
			output: &GitSizerOutput{
				MaxCommitSize:        GitSizerMetric{Value: 150 * 1024 * 1024}, // 150MB - exceeds GitHub limit
				MaxBlobSize:          GitSizerMetric{Value: 1024 * 1024},
				UniqueBlobSize:       GitSizerMetric{Value: 100 * 1024 * 1024},
				UniqueTreeSize:       GitSizerMetric{Value: 10 * 1024 * 1024},
				UniqueCommitSize:     GitSizerMetric{Value: 1 * 1024 * 1024},
				MaxHistoryDepth:      GitSizerMetric{Value: 1000},
				MaxTreeEntries:       GitSizerMetric{Value: 100},
				MaxCheckoutBlobCount: GitSizerMetric{Value: 1000},
			},
			expected: 1,
		},
		{
			name: "Multiple problems",
			output: &GitSizerOutput{
				MaxCommitSize:        GitSizerMetric{Value: 150 * 1024 * 1024},      // 150MB - exceeds limit
				MaxBlobSize:          GitSizerMetric{Value: 100 * 1024 * 1024},      // 100MB
				UniqueBlobSize:       GitSizerMetric{Value: 6 * 1024 * 1024 * 1024}, // 6GB
				UniqueTreeSize:       GitSizerMetric{Value: 10 * 1024 * 1024},
				UniqueCommitSize:     GitSizerMetric{Value: 1 * 1024 * 1024},
				MaxHistoryDepth:      GitSizerMetric{Value: 150000},
				MaxTreeEntries:       GitSizerMetric{Value: 15000},
				MaxCheckoutBlobCount: GitSizerMetric{Value: 150000},
			},
			expected: 6,
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

func TestExtractCommitSHA(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Full SHA only",
			input:    "f7a25d8bede5b581accd6abe89cad8cc1c4b6d8d",
			expected: "f7a25d8bede5b581accd6abe89cad8cc1c4b6d8d",
		},
		{
			name:     "SHA with branch info",
			input:    "f7a25d8bede5b581accd6abe89cad8cc1c4b6d8d (refs/heads/main)",
			expected: "f7a25d8bede5b581accd6abe89cad8cc1c4b6d8d",
		},
		{
			name:     "Short SHA",
			input:    "f7a25d8",
			expected: "f7a25d8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractCommitSHA(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractFilenameFromBlobInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard format",
			input:    "319b802f686f9d80d4d2e7e62d1ccea8eea87766 (9ae8b638196a3ff9ec70b1b556db104c42e3365c:IMPLEMENTATION_GUIDE.md)",
			expected: "IMPLEMENTATION_GUIDE.md",
		},
		{
			name:     "With path",
			input:    "abc123 (def456:docs/README.md)",
			expected: "docs/README.md",
		},
		{
			name:     "No filename",
			input:    "abc123",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractFilenameFromBlobInfo(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetLastCommitSHA(t *testing.T) {
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
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	cmd.Run()

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

	sha := analyzer.getLastCommitSHA(ctx, tempDir)
	if sha == "" {
		t.Error("Expected non-empty commit SHA")
	}
	if len(sha) != 40 {
		t.Errorf("Expected 40-character SHA, got %d characters: %s", len(sha), sha)
	}
}

func TestGetTagCount(t *testing.T) {
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
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	cmd.Run()

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

	// Initially no tags
	count := analyzer.getTagCount(ctx, tempDir)
	if count != 0 {
		t.Errorf("Expected 0 tags, got %d", count)
	}

	// Create a tag
	cmd = exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Skip("Failed to create tag, skipping")
	}

	count = analyzer.getTagCount(ctx, tempDir)
	if count != 1 {
		t.Errorf("Expected 1 tag, got %d", count)
	}
}

func TestLargeFileDetection(t *testing.T) {
	tests := []struct {
		name             string
		maxBlobSize      int64
		expectLargeFiles bool
		expectCount      int
	}{
		{
			name:             "No large files - 50MB",
			maxBlobSize:      50 * 1024 * 1024,
			expectLargeFiles: false,
			expectCount:      0,
		},
		{
			name:             "Large file - exactly 100MB",
			maxBlobSize:      100 * 1024 * 1024,
			expectLargeFiles: false,
			expectCount:      0,
		},
		{
			name:             "Large file - 101MB",
			maxBlobSize:      101 * 1024 * 1024,
			expectLargeFiles: true,
			expectCount:      1,
		},
		{
			name:             "Large file - 500MB",
			maxBlobSize:      500 * 1024 * 1024,
			expectLargeFiles: true,
			expectCount:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &models.Repository{
				FullName: "test/repo",
			}

			// Simulate the large file detection logic
			if tt.maxBlobSize > LargeFileThreshold {
				repo.HasLargeFiles = true
				repo.LargeFileCount = 1
			}

			if repo.HasLargeFiles != tt.expectLargeFiles {
				t.Errorf("Expected HasLargeFiles=%v, got %v", tt.expectLargeFiles, repo.HasLargeFiles)
			}
			if repo.LargeFileCount != tt.expectCount {
				t.Errorf("Expected LargeFileCount=%d, got %d", tt.expectCount, repo.LargeFileCount)
			}
		})
	}
}

func TestParseGitCountObjects(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	tests := []struct {
		name     string
		input    string
		expected *GitCountObjectsOutput
	}{
		{
			name: "Standard output with human-readable sizes",
			input: `count: 391
size: 1.84 MiB
in-pack: 4
packs: 1
size-pack: 2.44 KiB
prune-packable: 0
garbage: 0
size-garbage: 0 bytes`,
			expected: &GitCountObjectsOutput{
				Count:         391,
				Size:          1929379, // 1.84 MiB in bytes (1.84 * 1024 * 1024)
				InPack:        4,
				Packs:         1,
				SizePack:      2498, // 2.44 KiB in bytes (approx)
				PrunePackable: 0,
				Garbage:       0,
				SizeGarbage:   0,
			},
		},
		{
			name: "Output with larger values",
			input: `count: 1000
size: 500 MiB
in-pack: 50000
packs: 5
size-pack: 2.5 GiB
prune-packable: 10
garbage: 2
size-garbage: 1.5 KiB`,
			expected: &GitCountObjectsOutput{
				Count:         1000,
				Size:          524288000, // 500 MiB
				InPack:        50000,
				Packs:         5,
				SizePack:      2684354560, // 2.5 GiB
				PrunePackable: 10,
				Garbage:       2,
				SizeGarbage:   1536, // 1.5 KiB
			},
		},
		{
			name: "Output with zero bytes",
			input: `count: 0
size: 0 bytes
in-pack: 0
packs: 0
size-pack: 0 bytes
prune-packable: 0
garbage: 0
size-garbage: 0 bytes`,
			expected: &GitCountObjectsOutput{
				Count:         0,
				Size:          0,
				InPack:        0,
				Packs:         0,
				SizePack:      0,
				PrunePackable: 0,
				Garbage:       0,
				SizeGarbage:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.parseGitCountObjects(tt.input)
			if err != nil {
				t.Fatalf("parseGitCountObjects failed: %v", err)
			}

			if result.Count != tt.expected.Count {
				t.Errorf("Count: expected %d, got %d", tt.expected.Count, result.Count)
			}
			if result.InPack != tt.expected.InPack {
				t.Errorf("InPack: expected %d, got %d", tt.expected.InPack, result.InPack)
			}
			if result.Packs != tt.expected.Packs {
				t.Errorf("Packs: expected %d, got %d", tt.expected.Packs, result.Packs)
			}
			if result.PrunePackable != tt.expected.PrunePackable {
				t.Errorf("PrunePackable: expected %d, got %d", tt.expected.PrunePackable, result.PrunePackable)
			}
			if result.Garbage != tt.expected.Garbage {
				t.Errorf("Garbage: expected %d, got %d", tt.expected.Garbage, result.Garbage)
			}
			// Allow some tolerance for size conversions due to floating point
			if result.Size < tt.expected.Size-1000 || result.Size > tt.expected.Size+1000 {
				t.Errorf("Size: expected ~%d, got %d", tt.expected.Size, result.Size)
			}
			if result.SizePack < tt.expected.SizePack-1000 || result.SizePack > tt.expected.SizePack+1000 {
				t.Errorf("SizePack: expected ~%d, got %d", tt.expected.SizePack, result.SizePack)
			}
			if result.SizeGarbage < tt.expected.SizeGarbage-10 || result.SizeGarbage > tt.expected.SizeGarbage+10 {
				t.Errorf("SizeGarbage: expected ~%d, got %d", tt.expected.SizeGarbage, result.SizeGarbage)
			}
		})
	}
}

func TestParseHumanSize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	tests := []struct {
		name     string
		input    string
		expected int64
		hasError bool
	}{
		{
			name:     "Bytes",
			input:    "0 bytes",
			expected: 0,
		},
		{
			name:     "Single byte",
			input:    "1 byte",
			expected: 1,
		},
		{
			name:     "KiB",
			input:    "2.44 KiB",
			expected: 2498, // approximately
		},
		{
			name:     "MiB",
			input:    "1.84 MiB",
			expected: 1929379, // approximately
		},
		{
			name:     "GiB",
			input:    "2.5 GiB",
			expected: 2684354560,
		},
		{
			name:     "TiB",
			input:    "1.0 TiB",
			expected: 1099511627776,
		},
		{
			name:     "Integer KiB",
			input:    "100 KiB",
			expected: 102400,
		},
		{
			name:     "Integer MiB",
			input:    "50 MiB",
			expected: 52428800,
		},
		{
			name:     "Just number",
			input:    "1024",
			expected: 1024,
		},
		{
			name:     "Unknown unit",
			input:    "100 foobar",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.parseHumanSize(tt.input)
			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			// Allow 1% tolerance for floating point conversions
			tolerance := tt.expected / 100
			if tolerance < 10 {
				tolerance = 10
			}
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("Expected ~%d (±%d), got %d", tt.expected, tolerance, result)
			}
		})
	}
}

func TestParseInteger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)

	tests := []struct {
		name     string
		input    string
		expected int64
		hasError bool
	}{
		{
			name:     "Zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "Small number",
			input:    "42",
			expected: 42,
		},
		{
			name:     "Large number",
			input:    "1234567890",
			expected: 1234567890,
		},
		{
			name:     "Invalid number",
			input:    "abc",
			hasError: true,
		},
		{
			name:     "Number with text",
			input:    "123abc",
			expected: 123, // Sscanf stops at first non-digit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.parseInteger(tt.input)
			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetGitObjectSize_Integration(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	analyzer := NewAnalyzer(logger)
	ctx := context.Background()

	// Create a temporary git repository
	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	cmd.Run()

	// Create a test file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content for git object size calculation"), 0644); err != nil {
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

	// Test getGitObjectSize
	size, err := analyzer.getGitObjectSize(ctx, tempDir)
	if err != nil {
		t.Fatalf("getGitObjectSize failed: %v", err)
	}

	if size <= 0 {
		t.Error("Expected positive size, got", size)
	}

	t.Logf("Git object size: %d bytes", size)
}
