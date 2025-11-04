package embedded

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	archAmd64      = "amd64"
	osWindows      = "windows"
	binaryGitSizer = "git-sizer"
)

// Embed git-sizer binaries for different platforms
// These files should be placed in internal/embedded/bin/ directory

//go:embed bin/git-sizer-linux-amd64
var gitSizerLinuxAmd64 []byte

//go:embed bin/git-sizer-linux-arm64
var gitSizerLinuxArm64 []byte

//go:embed bin/git-sizer-darwin-amd64
var gitSizerDarwinAmd64 []byte

//go:embed bin/git-sizer-darwin-arm64
var gitSizerDarwinArm64 []byte

//go:embed bin/git-sizer-windows-amd64.exe
var gitSizerWindowsAmd64 []byte

var (
	extractedBinaryPath string
	extractOnce         sync.Once
	extractErr          error
)

// GetGitSizerPath returns the path to the git-sizer binary
// It extracts the embedded binary on first call and returns the cached path
// Falls back to system git-sizer if embedded binary is not available
func GetGitSizerPath() (string, error) {
	extractOnce.Do(func() {
		extractedBinaryPath, extractErr = extractGitSizer()

		// Fallback to system git-sizer if extraction fails (e.g., during development)
		if extractErr != nil {
			systemPath, err := exec.LookPath("git-sizer")
			if err == nil {
				extractedBinaryPath = systemPath
				extractErr = nil
			}
		}
	})
	return extractedBinaryPath, extractErr
}

// extractGitSizer extracts the appropriate git-sizer binary for the current platform
func extractGitSizer() (string, error) {
	// Select the appropriate binary for the current platform
	binaryData, binaryName, err := selectPlatformBinary()
	if err != nil {
		return "", err
	}

	if len(binaryData) == 0 {
		return "", fmt.Errorf("git-sizer binary not embedded for %s/%s (run './scripts/download-git-sizer.sh' before building)", runtime.GOOS, runtime.GOARCH)
	}

	// Get the target path for the binary
	binaryPath, err := prepareBinaryPath(binaryName)
	if err != nil {
		return "", err
	}

	// Check if binary already exists and is valid
	if _, statErr := os.Stat(binaryPath); statErr == nil {
		if verifyBinary(binaryPath) == nil {
			return binaryPath, nil
		}
		_ = os.Remove(binaryPath)
	}

	// Write and verify the binary
	return writeBinary(binaryPath, binaryData)
}

// selectPlatformBinary selects the appropriate binary data based on OS and architecture
func selectPlatformBinary() ([]byte, string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "linux":
		return selectLinuxBinary(goarch)
	case "darwin":
		return selectDarwinBinary(goarch)
	case osWindows:
		return selectWindowsBinary(goarch)
	default:
		return nil, "", fmt.Errorf("unsupported operating system: %s", goos)
	}
}

// selectLinuxBinary selects the Linux binary based on architecture
func selectLinuxBinary(goarch string) ([]byte, string, error) {
	switch goarch {
	case archAmd64:
		return gitSizerLinuxAmd64, binaryGitSizer, nil
	case "arm64":
		return gitSizerLinuxArm64, binaryGitSizer, nil
	default:
		return nil, "", fmt.Errorf("unsupported Linux architecture: %s", goarch)
	}
}

// selectDarwinBinary selects the macOS binary based on architecture
func selectDarwinBinary(goarch string) ([]byte, string, error) {
	switch goarch {
	case archAmd64:
		return gitSizerDarwinAmd64, binaryGitSizer, nil
	case "arm64":
		return gitSizerDarwinArm64, binaryGitSizer, nil
	default:
		return nil, "", fmt.Errorf("unsupported macOS architecture: %s", goarch)
	}
}

// selectWindowsBinary selects the Windows binary based on architecture
func selectWindowsBinary(goarch string) ([]byte, string, error) {
	if goarch == archAmd64 {
		return gitSizerWindowsAmd64, "git-sizer.exe", nil
	}
	return nil, "", fmt.Errorf("unsupported Windows architecture: %s", goarch)
}

// prepareBinaryPath creates the directory and returns the target path
func prepareBinaryPath(binaryName string) (string, error) {
	tmpDir := getBinaryStorageDir()
	// #nosec G301 -- 0755 is appropriate for temporary directory
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory %s: %w", tmpDir, err)
	}
	return filepath.Join(tmpDir, binaryName), nil
}

// writeBinary writes the binary to disk and verifies it
func writeBinary(binaryPath string, binaryData []byte) (string, error) {
	// #nosec G306 -- 0755 is required for binary to be executable
	if err := os.WriteFile(binaryPath, binaryData, 0755); err != nil {
		return "", fmt.Errorf("failed to write git-sizer binary to %s: %w", binaryPath, err)
	}

	// Verify the extracted binary
	if err := verifyBinary(binaryPath); err != nil {
		if shouldSkipVerification() {
			fmt.Fprintf(os.Stderr, "WARNING: Binary verification failed (common in restricted environments): %v\n", err)
			fmt.Fprintf(os.Stderr, "WARNING: Proceeding with unverified binary - this may cause git analysis to fail\n")
			return binaryPath, nil
		}
		return "", fmt.Errorf("extracted binary verification failed for %s: %w", binaryPath, err)
	}

	return binaryPath, nil
}

// verifyBinary checks if the binary is executable and responds to --version
func verifyBinary(path string) error {
	// Check if file is executable
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Check file permissions on Unix-like systems
	if runtime.GOOS != osWindows {
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("binary is not executable")
		}
	}

	// Try to run --version to verify it works
	cmd := exec.Command(path, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("binary execution test failed: %w", err)
	}

	return nil
}

// getBinaryStorageDir returns the appropriate directory for storing extracted binaries
// In Azure App Service, /tmp may have restrictions, so we use /home/site/tmp
// In other environments, we use the system temp directory
func getBinaryStorageDir() string {
	// Check if we're running in Azure App Service
	// Azure sets WEBSITE_SITE_NAME environment variable
	if os.Getenv("WEBSITE_SITE_NAME") != "" {
		// Use /home/site/tmp in Azure App Service
		// This directory has proper permissions and is local to the instance
		return filepath.Join("/home", "site", "tmp", "github-migrator-binaries")
	}

	// Check if custom temp directory is set via environment variable
	if customTmp := os.Getenv("GHMIG_TEMP_DIR"); customTmp != "" {
		return filepath.Join(customTmp, "github-migrator-binaries")
	}

	// Default to system temp directory
	return filepath.Join(os.TempDir(), "github-migrator-binaries")
}

// shouldSkipVerification checks if we should skip binary verification
// Returns true in restricted environments where verification may fail but binary may still work
func shouldSkipVerification() bool {
	// Check if we're in Azure App Service
	if os.Getenv("WEBSITE_SITE_NAME") != "" {
		return true
	}

	// Check if verification skip is explicitly requested
	if os.Getenv("GHMIG_SKIP_BINARY_VERIFICATION") == "true" {
		return true
	}

	// Check if we're in a container (common restricted environment)
	// Docker creates /.dockerenv file, and many containers have this
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

// CleanupExtractedBinaries removes the temporary directory containing extracted binaries
// This should be called during application shutdown if cleanup is desired
func CleanupExtractedBinaries() error {
	tmpDir := getBinaryStorageDir()
	return os.RemoveAll(tmpDir)
}
