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
	// Determine which binary to use based on OS and architecture
	var binaryData []byte
	var binaryName string

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "linux":
		switch goarch {
		case archAmd64:
			binaryData = gitSizerLinuxAmd64
			binaryName = binaryGitSizer
		case "arm64":
			binaryData = gitSizerLinuxArm64
			binaryName = binaryGitSizer
		default:
			return "", fmt.Errorf("unsupported Linux architecture: %s", goarch)
		}
	case "darwin":
		switch goarch {
		case archAmd64:
			binaryData = gitSizerDarwinAmd64
			binaryName = binaryGitSizer
		case "arm64":
			binaryData = gitSizerDarwinArm64
			binaryName = binaryGitSizer
		default:
			return "", fmt.Errorf("unsupported macOS architecture: %s", goarch)
		}
	case osWindows:
		if goarch == archAmd64 {
			binaryData = gitSizerWindowsAmd64
			binaryName = "git-sizer.exe"
		} else {
			return "", fmt.Errorf("unsupported Windows architecture: %s", goarch)
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", goos)
	}

	if len(binaryData) == 0 {
		return "", fmt.Errorf("git-sizer binary not embedded for %s/%s (run './scripts/download-git-sizer.sh' before building)", goos, goarch)
	}

	// Create a temporary directory for the binary
	// We use a subdirectory of os.TempDir() to avoid conflicts
	tmpDir := filepath.Join(os.TempDir(), "github-migrator-binaries")
	// #nosec G301 -- 0755 is appropriate for temporary directory
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write the binary to a temporary file
	binaryPath := filepath.Join(tmpDir, binaryName)

	// Check if binary already exists and is valid
	if _, err := os.Stat(binaryPath); err == nil {
		// Binary exists, verify it's executable
		if err := verifyBinary(binaryPath); err == nil {
			return binaryPath, nil
		}
		// If verification fails, remove and re-extract
		_ = os.Remove(binaryPath)
	}

	// #nosec G306 -- 0755 is required for binary to be executable
	if err := os.WriteFile(binaryPath, binaryData, 0755); err != nil {
		return "", fmt.Errorf("failed to write git-sizer binary: %w", err)
	}

	// Verify the extracted binary
	if err := verifyBinary(binaryPath); err != nil {
		return "", fmt.Errorf("extracted binary verification failed: %w", err)
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

// CleanupExtractedBinaries removes the temporary directory containing extracted binaries
// This should be called during application shutdown if cleanup is desired
func CleanupExtractedBinaries() error {
	tmpDir := filepath.Join(os.TempDir(), "github-migrator-binaries")
	return os.RemoveAll(tmpDir)
}
