package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultGitHubAPIURL = "https://api.github.com"
	licenseCacheTTL     = 5 * time.Minute
)

// validateCLIPath validates that a CLI path is safe to execute.
// This prevents command injection by ensuring the path:
// - Does not contain shell metacharacters
// - Is a valid executable path (exists or is in PATH)
// Returns the sanitized path or an error if validation fails.
func validateCLIPath(cliPath string) (string, error) {
	if cliPath == "" {
		return "", fmt.Errorf("CLI path cannot be empty")
	}

	// Check for dangerous shell metacharacters that could enable command injection
	// These characters have special meaning in shells and could be exploited
	dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "<", ">", "\n", "\r", "\\", "'", "\"", "*", "?", "[", "]", "!", "~"}
	for _, char := range dangerousChars {
		if strings.Contains(cliPath, char) {
			return "", fmt.Errorf("CLI path contains invalid character: %q", char)
		}
	}

	// Clean the path to normalize it (removes .., extra slashes, etc.)
	cleanPath := filepath.Clean(cliPath)

	// If it's an absolute path, verify the file exists
	if filepath.IsAbs(cleanPath) {
		info, err := os.Stat(cleanPath)
		if err != nil {
			return "", fmt.Errorf("CLI path does not exist: %s", cleanPath)
		}
		if info.IsDir() {
			return "", fmt.Errorf("CLI path is a directory, not an executable: %s", cleanPath)
		}
		// Check if the file is executable (on Unix systems)
		if info.Mode()&0111 == 0 {
			return "", fmt.Errorf("CLI path is not executable: %s", cleanPath)
		}
		return cleanPath, nil
	}

	// For relative paths or bare command names, verify it can be found in PATH
	resolvedPath, err := exec.LookPath(cleanPath)
	if err != nil {
		return "", fmt.Errorf("CLI not found in PATH: %s", cleanPath)
	}

	return resolvedPath, nil
}

// LicenseValidator validates Copilot license/subscription status
type LicenseValidator struct {
	baseURL string
	logger  *slog.Logger
	cache   *licenseCache
}

// LicenseStatus represents the Copilot license status for a user
type LicenseStatus struct {
	Valid      bool      `json:"valid"`
	HasSeat    bool      `json:"has_seat"`
	SeatType   string    `json:"seat_type,omitempty"`   // "assigned", "pending"
	AssignedAt time.Time `json:"assigned_at,omitempty"` // When the seat was assigned
	Message    string    `json:"message,omitempty"`     // Human-readable status message
	CheckedAt  time.Time `json:"checked_at"`            // When this status was checked
	ExpiresAt  time.Time `json:"expires_at"`            // When this cache entry expires
	Error      string    `json:"error,omitempty"`       // Error message if check failed
}

// licenseCache stores license status to avoid repeated API calls
type licenseCache struct {
	mu      sync.RWMutex
	entries map[string]*LicenseStatus // keyed by userLogin
}

func newLicenseCache() *licenseCache {
	return &licenseCache{
		entries: make(map[string]*LicenseStatus),
	}
}

func (c *licenseCache) get(userLogin string) (*LicenseStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status, ok := c.entries[userLogin]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Now().After(status.ExpiresAt) {
		return nil, false
	}

	return status, true
}

func (c *licenseCache) set(userLogin string, status *LicenseStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userLogin] = status
}

func (c *licenseCache) invalidate(userLogin string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userLogin)
}

// NewLicenseValidator creates a new license validator
func NewLicenseValidator(baseURL string, logger *slog.Logger) *LicenseValidator {
	if baseURL == "" {
		baseURL = defaultGitHubAPIURL
	}
	// Normalize URL
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &LicenseValidator{
		baseURL: baseURL,
		logger:  logger,
		cache:   newLicenseCache(),
	}
}

// CheckLicense checks if a user has a valid Copilot license
// It first checks the cache, then queries the GitHub API if needed
func (v *LicenseValidator) CheckLicense(ctx context.Context, userLogin string, token string) (*LicenseStatus, error) {
	// Check cache first
	if cached, ok := v.cache.get(userLogin); ok {
		v.logger.Debug("Using cached Copilot license status", "user", userLogin, "valid", cached.Valid)
		return cached, nil
	}

	// Query the GitHub API
	status, err := v.queryLicenseStatus(ctx, userLogin, token)
	if err != nil {
		// Return a status with error info, but don't fail completely
		errorStatus := &LicenseStatus{
			Valid:     false,
			HasSeat:   false,
			Message:   "Failed to verify Copilot license",
			CheckedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Minute), // Short TTL for errors
			Error:     err.Error(),
		}
		v.cache.set(userLogin, errorStatus)
		return errorStatus, nil
	}

	// Cache the result
	v.cache.set(userLogin, status)
	return status, nil
}

// InvalidateCache removes a user's cached license status
func (v *LicenseValidator) InvalidateCache(userLogin string) {
	v.cache.invalidate(userLogin)
}

// queryLicenseStatus queries the GitHub API for Copilot license status
func (v *LicenseValidator) queryLicenseStatus(ctx context.Context, userLogin string, token string) (*LicenseStatus, error) {
	// The /user endpoint returns information about the authenticated user
	// including their Copilot access
	url := fmt.Sprintf("%s/user", v.baseURL)

	v.logger.Debug("Checking Copilot license via GitHub API", "url", url, "user", userLogin)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Now check the Copilot-specific endpoint
	// GET /user/copilot_seat returns the user's Copilot seat information
	copilotURL := fmt.Sprintf("%s/user/copilot_seat", v.baseURL)
	copilotReq, err := http.NewRequestWithContext(ctx, "GET", copilotURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Copilot request: %w", err)
	}

	copilotReq.Header.Set("Authorization", "Bearer "+token)
	copilotReq.Header.Set("Accept", "application/vnd.github+json")
	copilotReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	copilotResp, err := client.Do(copilotReq)
	if err != nil {
		return nil, fmt.Errorf("failed to query Copilot API: %w", err)
	}
	defer func() { _ = copilotResp.Body.Close() }()

	now := time.Now()

	// 404 means no Copilot seat assigned
	if copilotResp.StatusCode == http.StatusNotFound {
		return &LicenseStatus{
			Valid:     false,
			HasSeat:   false,
			Message:   "No Copilot seat assigned to this user",
			CheckedAt: now,
			ExpiresAt: now.Add(licenseCacheTTL),
		}, nil
	}

	// 401/403 means authentication issue or insufficient permissions
	if copilotResp.StatusCode == http.StatusUnauthorized || copilotResp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(copilotResp.Body)
		return &LicenseStatus{
			Valid:     false,
			HasSeat:   false,
			Message:   "Unable to verify Copilot license - insufficient permissions",
			CheckedAt: now,
			ExpiresAt: now.Add(licenseCacheTTL),
			Error:     string(body),
		}, nil
	}

	if copilotResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(copilotResp.Body)
		return nil, fmt.Errorf("copilot API returned status %d: %s", copilotResp.StatusCode, string(body))
	}

	// Parse the response
	var seatInfo struct {
		SeatType                string    `json:"seat_type"`               // e.g., "assigned"
		SeatManagementSetting   string    `json:"seat_management_setting"` // e.g., "assign_all", "assign_selected"
		CreatedAt               time.Time `json:"created_at"`
		UpdatedAt               time.Time `json:"updated_at"`
		PendingCancellationDate *string   `json:"pending_cancellation_date"`
	}

	if err := json.NewDecoder(copilotResp.Body).Decode(&seatInfo); err != nil {
		return nil, fmt.Errorf("failed to parse Copilot seat info: %w", err)
	}

	// User has a Copilot seat
	status := &LicenseStatus{
		Valid:      true,
		HasSeat:    true,
		SeatType:   seatInfo.SeatType,
		AssignedAt: seatInfo.CreatedAt,
		Message:    "Copilot seat assigned",
		CheckedAt:  now,
		ExpiresAt:  now.Add(licenseCacheTTL),
	}

	// Check if there's a pending cancellation
	if seatInfo.PendingCancellationDate != nil && *seatInfo.PendingCancellationDate != "" {
		status.Message = fmt.Sprintf("Copilot seat assigned (pending cancellation on %s)", *seatInfo.PendingCancellationDate)
	}

	v.logger.Info("Copilot license check complete",
		"user", userLogin,
		"valid", status.Valid,
		"seat_type", status.SeatType)

	return status, nil
}

// CheckCLIAvailable checks if the Copilot CLI is installed and accessible
func CheckCLIAvailable(cliPath string) (bool, string, error) {
	if cliPath == "" {
		// Check environment variable first
		if envPath := os.Getenv("COPILOT_CLI_PATH"); envPath != "" {
			cliPath = envPath
		} else {
			// Try well-known paths in order of preference
			knownPaths := []string{
				"/usr/local/bin/copilot", // Docker/Linux standard
				"/usr/bin/copilot",       // System-wide install
				"copilot",                // In PATH (fallback)
			}
			for _, path := range knownPaths {
				if _, err := exec.LookPath(path); err == nil {
					cliPath = path
					break
				}
			}
			// If none found, use "copilot" and let the error handling below report it
			if cliPath == "" {
				cliPath = "copilot"
			}
		}
	}

	// Validate the CLI path to prevent command injection (G204)
	// This ensures the path is safe before passing to exec.CommandContext
	validatedPath, err := validateCLIPath(cliPath)
	if err != nil {
		return false, "", fmt.Errorf("invalid CLI path: %w", err)
	}

	// Try to execute the CLI to verify it works and get version info
	// Use --version or version command to check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try --version first (common for most CLIs)
	// #nosec G204 -- validatedPath has been sanitized by validateCLIPath
	cmd := exec.CommandContext(ctx, validatedPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without arguments - some CLIs print version info
		// #nosec G204 -- validatedPath has been sanitized by validateCLIPath
		cmd = exec.CommandContext(ctx, validatedPath, "version")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("failed to execute Copilot CLI: %v", err)
		}
	}

	// Parse version from output
	version := strings.TrimSpace(string(output))
	if version == "" {
		version = "unknown"
	}
	// Truncate long output (only keep first line)
	if idx := strings.Index(version, "\n"); idx > 0 {
		version = version[:idx]
	}
	// Limit length
	if len(version) > 100 {
		version = version[:100] + "..."
	}

	return true, version, nil
}
