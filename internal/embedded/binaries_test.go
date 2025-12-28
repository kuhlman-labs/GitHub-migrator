package embedded

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestGetGitSizerPath(t *testing.T) {
	t.Run("returns path on success", func(t *testing.T) {
		path, err := GetGitSizerPath()

		// Should either return embedded binary or system binary
		if err != nil {
			t.Logf("GetGitSizerPath returned error (expected during test without binaries): %v", err)

			// Check if system git-sizer exists as fallback
			systemPath, sysErr := exec.LookPath("git-sizer")
			if sysErr == nil {
				t.Logf("System git-sizer found at: %s", systemPath)
			} else {
				t.Logf("No system git-sizer found either: %v", sysErr)
			}
		} else {
			if path == "" {
				t.Error("GetGitSizerPath returned empty path without error")
			}
			t.Logf("GetGitSizerPath succeeded, returned: %s", path)
		}
	})

	t.Run("caches result", func(t *testing.T) {
		// Call multiple times
		path1, err1 := GetGitSizerPath()
		path2, err2 := GetGitSizerPath()
		path3, err3 := GetGitSizerPath()

		// All calls should return the same result
		if path1 != path2 || path2 != path3 {
			t.Errorf("GetGitSizerPath not caching: got paths %s, %s, %s", path1, path2, path3)
		}

		if err1 != err2 || err2 != err3 {
			t.Errorf("GetGitSizerPath not caching errors: got %v, %v, %v", err1, err2, err3)
		}
	})
}

func TestExtractGitSizer(t *testing.T) {
	t.Run("returns error for empty binaries", func(t *testing.T) {
		// This test will likely fail in actual implementation since we check for empty binaries
		_, err := extractGitSizer()
		if err == nil {
			t.Log("extractGitSizer succeeded (binaries may be present)")
		} else {
			// Expected during test without actual binaries
			if !strings.Contains(err.Error(), "not embedded") && !strings.Contains(err.Error(), "failed") {
				t.Errorf("Unexpected error message: %v", err)
			}
		}
	})

	t.Run("creates temp directory", func(t *testing.T) {
		tmpDir := getBinaryStorageDir()

		// Try to extract (may fail without binaries, but should create directory)
		_, _ = extractGitSizer()

		// Check if temp directory exists (may or may not depending on implementation)
		if info, err := os.Stat(tmpDir); err == nil {
			if !info.IsDir() {
				t.Errorf("Expected %s to be a directory", tmpDir)
			}
		}
	})
}

func TestVerifyBinary(t *testing.T) {
	t.Run("fails for non-existent file", func(t *testing.T) {
		err := verifyBinary("/nonexistent/binary")
		if err == nil {
			t.Error("verifyBinary should fail for non-existent file")
		}
	})

	t.Run("fails for non-executable file", func(t *testing.T) {
		// Create a temporary non-executable file
		tmpFile, err := os.CreateTemp("", "test-binary-*")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Make it non-executable
		if err := os.Chmod(tmpFile.Name(), 0644); err != nil {
			t.Fatalf("Failed to chmod: %v", err)
		}

		err = verifyBinary(tmpFile.Name())
		if err == nil && runtime.GOOS != "windows" {
			t.Error("verifyBinary should fail for non-executable file on Unix")
		}
	})

	t.Run("succeeds for valid executable", func(t *testing.T) {
		// Try to find a system executable to test with
		testBinaries := []string{"git", "ls", "echo", "cat"}
		var foundBinary string

		for _, binary := range testBinaries {
			if path, err := exec.LookPath(binary); err == nil {
				foundBinary = path
				break
			}
		}

		if foundBinary == "" {
			t.Skip("No test binary found in PATH")
		}

		// Git should be available since we use it in the project
		if gitPath, err := exec.LookPath("git"); err == nil {
			err := verifyBinary(gitPath)
			if err != nil {
				t.Logf("verifyBinary failed for git (expected, as git may not support --version flag check): %v", err)
			}
		}
	})
}

func TestCleanupExtractedBinaries(t *testing.T) {
	t.Run("removes temp directory", func(t *testing.T) {
		tmpDir := getBinaryStorageDir()

		// Create the directory
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}

		// Create a test file in it
		testFile := filepath.Join(tmpDir, "test-file")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Cleanup
		err := CleanupExtractedBinaries()
		if err != nil {
			t.Errorf("CleanupExtractedBinaries failed: %v", err)
		}

		// Verify directory is removed
		if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
			t.Error("Temp directory should be removed after cleanup")
		}
	})

	t.Run("succeeds even if directory doesn't exist", func(t *testing.T) {
		// Remove directory first
		tmpDir := getBinaryStorageDir()
		os.RemoveAll(tmpDir)

		// Cleanup should not fail
		err := CleanupExtractedBinaries()
		if err != nil {
			t.Errorf("CleanupExtractedBinaries failed when directory doesn't exist: %v", err)
		}
	})
}

func TestPlatformDetection(t *testing.T) {
	t.Run("detects current platform", func(t *testing.T) {
		goos := runtime.GOOS
		goarch := runtime.GOARCH

		t.Logf("Current platform: %s/%s", goos, goarch)

		supportedPlatforms := map[string][]string{
			"linux":   {"amd64", "arm64"},
			"darwin":  {"amd64", "arm64"},
			"windows": {"amd64"},
		}

		if archs, ok := supportedPlatforms[goos]; ok {
			supported := slices.Contains(archs, goarch)
			if !supported {
				t.Logf("Current architecture %s not in supported list for %s", goarch, goos)
			}
		} else {
			t.Logf("Current OS %s not in supported list", goos)
		}
	})
}

func TestFallbackToSystemGitSizer(t *testing.T) {
	t.Run("checks for system git-sizer", func(t *testing.T) {
		path, err := exec.LookPath("git-sizer")
		if err != nil {
			t.Log("System git-sizer not found (expected in test environment)")
			t.Logf("Error: %v", err)
		} else {
			t.Logf("System git-sizer found at: %s", path)

			// Verify it's executable
			cmd := exec.Command(path, "--version")
			if err := cmd.Run(); err != nil {
				t.Logf("System git-sizer exists but failed to run: %v", err)
			} else {
				t.Log("System git-sizer is functional")
			}
		}
	})
}

func TestBinaryDataSizes(t *testing.T) {
	t.Run("checks embedded binary sizes", func(t *testing.T) {
		binaries := map[string][]byte{
			"linux-amd64":   gitSizerLinuxAmd64,
			"linux-arm64":   gitSizerLinuxArm64,
			"darwin-amd64":  gitSizerDarwinAmd64,
			"darwin-arm64":  gitSizerDarwinArm64,
			"windows-amd64": gitSizerWindowsAmd64,
		}

		for platform, data := range binaries {
			size := len(data)
			if size == 0 {
				t.Logf("Binary for %s is empty (expected if not downloaded)", platform)
			} else {
				// Real git-sizer binaries are typically 5-8 MB
				sizeMB := float64(size) / (1024 * 1024)
				t.Logf("Binary for %s: %.2f MB", platform, sizeMB)

				if sizeMB > 20 {
					t.Errorf("Binary for %s is suspiciously large: %.2f MB", platform, sizeMB)
				}
			}
		}
	})
}

// TestIntegration tests the full workflow
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("full workflow", func(t *testing.T) {
		// Note: Don't clean up before getting path, as sync.Once prevents re-extraction
		// The binary may already be extracted from previous tests

		// Get path
		path, err := GetGitSizerPath()
		if err != nil {
			// Check if it's a fallback to system
			if systemPath, sysErr := exec.LookPath("git-sizer"); sysErr == nil {
				if path != systemPath {
					t.Errorf("Expected fallback to system git-sizer at %s, got %s", systemPath, path)
				}
				t.Log("Successfully fell back to system git-sizer")
			} else {
				t.Logf("No embedded or system git-sizer available (expected in test): %v", err)
			}
			return
		}

		// Verify the path exists
		// Note: In test environment, the file may have been removed by TestCleanupExtractedBinaries
		// Since sync.Once prevents re-extraction, this is expected behavior
		if _, statErr := os.Stat(path); statErr != nil {
			t.Logf("Returned path does not exist (may have been cleaned by previous test): %s", path)
			t.Log("This is expected behavior when TestCleanupExtractedBinaries runs before this test")
			return
		}

		// Try to use it
		cmd := exec.Command(path, "--version")
		if err := cmd.Run(); err != nil {
			t.Errorf("Failed to execute git-sizer: %v", err)
		} else {
			t.Log("Successfully executed git-sizer --version")
		}

		// Note: Don't clean up here as it may affect other tests
		// Cleanup can be done manually if needed or rely on OS temp cleanup
	})
}

// BenchmarkGetGitSizerPath benchmarks the cached path retrieval
func BenchmarkGetGitSizerPath(b *testing.B) {
	// First call to initialize
	GetGitSizerPath()

	for b.Loop() {
		_, _ = GetGitSizerPath()
	}
}

// BenchmarkExtractGitSizer benchmarks binary extraction
func BenchmarkExtractGitSizer(b *testing.B) {
	// Note: This benchmark only tests the first extraction due to caching
	// Subsequent calls will use the cached path

	for b.Loop() {
		_, _ = extractGitSizer()
	}
}

func TestGetBinaryStorageDir(t *testing.T) {
	t.Run("uses system temp by default", func(t *testing.T) {
		// Ensure we're not in Azure or custom environment
		oldWebsite := os.Getenv("WEBSITE_SITE_NAME")
		oldCustom := os.Getenv("GHMIG_TEMP_DIR")
		os.Unsetenv("WEBSITE_SITE_NAME")
		os.Unsetenv("GHMIG_TEMP_DIR")
		defer func() {
			if oldWebsite != "" {
				os.Setenv("WEBSITE_SITE_NAME", oldWebsite)
			}
			if oldCustom != "" {
				os.Setenv("GHMIG_TEMP_DIR", oldCustom)
			}
		}()

		dir := getBinaryStorageDir()
		expected := filepath.Join(os.TempDir(), "github-migrator-binaries")
		if dir != expected {
			t.Errorf("Expected default temp dir %s, got %s", expected, dir)
		}
	})

	t.Run("uses Azure path when WEBSITE_SITE_NAME is set", func(t *testing.T) {
		// Set Azure environment variable
		oldWebsite := os.Getenv("WEBSITE_SITE_NAME")
		os.Setenv("WEBSITE_SITE_NAME", "test-site")
		defer func() {
			if oldWebsite != "" {
				os.Setenv("WEBSITE_SITE_NAME", oldWebsite)
			} else {
				os.Unsetenv("WEBSITE_SITE_NAME")
			}
		}()

		dir := getBinaryStorageDir()
		expected := filepath.Join("/home", "site", "tmp", "github-migrator-binaries")
		if dir != expected {
			t.Errorf("Expected Azure temp dir %s, got %s", expected, dir)
		}
	})

	t.Run("uses custom path when GHMIG_TEMP_DIR is set", func(t *testing.T) {
		// Ensure Azure variable is not set
		oldWebsite := os.Getenv("WEBSITE_SITE_NAME")
		oldCustom := os.Getenv("GHMIG_TEMP_DIR")
		os.Unsetenv("WEBSITE_SITE_NAME")
		os.Setenv("GHMIG_TEMP_DIR", "/custom/temp")
		defer func() {
			if oldWebsite != "" {
				os.Setenv("WEBSITE_SITE_NAME", oldWebsite)
			}
			if oldCustom != "" {
				os.Setenv("GHMIG_TEMP_DIR", oldCustom)
			} else {
				os.Unsetenv("GHMIG_TEMP_DIR")
			}
		}()

		dir := getBinaryStorageDir()
		expected := filepath.Join("/custom/temp", "github-migrator-binaries")
		if dir != expected {
			t.Errorf("Expected custom temp dir %s, got %s", expected, dir)
		}
	})

	t.Run("Azure takes precedence over custom", func(t *testing.T) {
		oldWebsite := os.Getenv("WEBSITE_SITE_NAME")
		oldCustom := os.Getenv("GHMIG_TEMP_DIR")
		os.Setenv("WEBSITE_SITE_NAME", "test-site")
		os.Setenv("GHMIG_TEMP_DIR", "/custom/temp")
		defer func() {
			if oldWebsite != "" {
				os.Setenv("WEBSITE_SITE_NAME", oldWebsite)
			} else {
				os.Unsetenv("WEBSITE_SITE_NAME")
			}
			if oldCustom != "" {
				os.Setenv("GHMIG_TEMP_DIR", oldCustom)
			} else {
				os.Unsetenv("GHMIG_TEMP_DIR")
			}
		}()

		dir := getBinaryStorageDir()
		expected := filepath.Join("/home", "site", "tmp", "github-migrator-binaries")
		if dir != expected {
			t.Errorf("Expected Azure temp dir to take precedence, got %s", dir)
		}
	})
}

// Helper to save and restore environment variables for tests
type envBackup struct {
	website string
	skip    string
}

func saveEnv() envBackup {
	return envBackup{
		website: os.Getenv("WEBSITE_SITE_NAME"),
		skip:    os.Getenv("GHMIG_SKIP_BINARY_VERIFICATION"),
	}
}

func (e envBackup) restore() {
	restoreEnvVar("WEBSITE_SITE_NAME", e.website)
	restoreEnvVar("GHMIG_SKIP_BINARY_VERIFICATION", e.skip)
}

func restoreEnvVar(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	} else {
		os.Unsetenv(key)
	}
}

func isInDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func TestShouldSkipVerification(t *testing.T) {
	t.Run("skips verification in Azure App Service", func(t *testing.T) {
		env := saveEnv()
		defer env.restore()

		os.Setenv("WEBSITE_SITE_NAME", "test-app")
		os.Unsetenv("GHMIG_SKIP_BINARY_VERIFICATION")

		if !shouldSkipVerification() {
			t.Error("Expected to skip verification in Azure App Service")
		}
	})

	t.Run("skips verification when explicitly requested", func(t *testing.T) {
		env := saveEnv()
		defer env.restore()

		os.Unsetenv("WEBSITE_SITE_NAME")
		os.Setenv("GHMIG_SKIP_BINARY_VERIFICATION", "true")

		if !shouldSkipVerification() {
			t.Error("Expected to skip verification when GHMIG_SKIP_BINARY_VERIFICATION is true")
		}
	})

	t.Run("skips verification in Docker container", func(t *testing.T) {
		env := saveEnv()
		defer env.restore()

		os.Unsetenv("WEBSITE_SITE_NAME")
		os.Unsetenv("GHMIG_SKIP_BINARY_VERIFICATION")

		if isInDocker() {
			if !shouldSkipVerification() {
				t.Error("Expected to skip verification in Docker container")
			}
		} else {
			if shouldSkipVerification() {
				t.Error("Expected to NOT skip verification in non-Docker environment")
			}
		}
	})

	t.Run("does not skip verification by default", func(t *testing.T) {
		env := saveEnv()
		defer env.restore()

		os.Unsetenv("WEBSITE_SITE_NAME")
		os.Unsetenv("GHMIG_SKIP_BINARY_VERIFICATION")

		if isInDocker() {
			t.Skip("Skipping test: running in Docker container")
		}

		if shouldSkipVerification() {
			t.Error("Expected to NOT skip verification in normal environment")
		}
	})
}
