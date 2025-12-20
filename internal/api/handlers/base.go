package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/discovery"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

const (
	boolTrue = "true"

	formatCSV  = "csv"
	formatJSON = "json"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	cleanFullNameKey contextKey = "cleanFullName"
)

// Handler contains all HTTP handlers
type Handler struct {
	*HandlerUtils            // Embed shared utilities for getClientForOrg
	db             DataStore // Uses interface for testability (storage.Database implements this)
	logger         *slog.Logger
	destDualClient *github.DualClient
	collector      *discovery.Collector
	sourceType     string      // Source type: models.SourceTypeGitHub or models.SourceTypeAzureDevOps
	adoHandler     *ADOHandler // ADO-specific handler (set by server if ADO is configured)
}

// SetADOHandler sets the ADO handler reference for delegating ADO operations
func (h *Handler) SetADOHandler(adoHandler *ADOHandler) {
	h.adoHandler = adoHandler
}

// NewHandler creates a new Handler instance
// sourceProvider can be nil if discovery is not needed
// sourceBaseConfig is used for per-org client creation in enterprise discovery (can be nil for PAT-only mode)
// authConfig is used for permission checks (can be nil if auth is disabled)
// sourceBaseURL is the source GitHub base URL for permission checks
func NewHandler(db *storage.Database, logger *slog.Logger, sourceDualClient *github.DualClient, destDualClient *github.DualClient, sourceProvider source.Provider, sourceBaseConfig *github.ClientConfig, authConfig *config.AuthConfig, sourceBaseURL string, sourceType string) *Handler {
	var collector *discovery.Collector
	// Use API client for discovery operations (will use App client if available, otherwise PAT)
	if sourceDualClient != nil && sourceProvider != nil {
		apiClient := sourceDualClient.APIClient()
		collector = discovery.NewCollector(apiClient, db, logger, sourceProvider)

		// If we have a base config with GitHub App credentials, set it on the collector
		// This enables per-org client creation for enterprise-wide discovery
		if sourceBaseConfig != nil {
			collector.WithBaseConfig(*sourceBaseConfig)
		}
	}
	return &Handler{
		HandlerUtils:   NewHandlerUtils(authConfig, sourceDualClient, sourceBaseConfig, sourceBaseURL, logger),
		db:             db,
		logger:         logger,
		destDualClient: destDualClient,
		collector:      collector,
		sourceType:     sourceType,
	}
}

// NewHandlerWithDataStore creates a new Handler instance with a DataStore interface.
// This is primarily used for testing with MockDataStore.
// Note: When using MockDataStore, the collector will be nil since it requires a real database.
func NewHandlerWithDataStore(db DataStore, logger *slog.Logger, sourceDualClient *github.DualClient, destDualClient *github.DualClient, sourceProvider source.Provider, sourceBaseConfig *github.ClientConfig, authConfig *config.AuthConfig, sourceBaseURL string, sourceType string) *Handler {
	// Note: collector requires *storage.Database, so it's nil when using MockDataStore
	// Tests that need the collector should use NewHandler with a real database
	return &Handler{
		HandlerUtils:   NewHandlerUtils(authConfig, sourceDualClient, sourceBaseConfig, sourceBaseURL, logger),
		db:             db,
		logger:         logger,
		destDualClient: destDualClient,
		collector:      nil,
		sourceType:     sourceType,
	}
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// GetConfig handles GET /api/v1/config
// Returns application-level configuration for the frontend
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	// Default to github if not set
	sourceType := h.sourceType
	if sourceType == "" {
		sourceType = models.SourceTypeGitHub
	}

	response := map[string]interface{}{
		"source_type":  sourceType,
		"auth_enabled": h.authConfig != nil && h.authConfig.Enabled,
	}

	// Add Entra ID enabled flag if auth is enabled
	if h.authConfig != nil && h.authConfig.Enabled {
		response["entraid_enabled"] = h.authConfig.EntraIDEnabled
	}

	h.sendJSON(w, http.StatusOK, response)
}

// sendJSON sends a JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, map[string]string{"error": message})
}

// handleContextError checks if an error is due to request cancellation and logs appropriately
// Returns true if the error is a context cancellation (caller should return early)
func (h *Handler) handleContextError(ctx context.Context, err error, operation string, r *http.Request) bool {
	if ctx.Err() == context.Canceled {
		h.logger.Debug("Request canceled by client",
			"operation", operation,
			"path", r.URL.Path,
			"method", r.Method)
		return true
	}
	if ctx.Err() == context.DeadlineExceeded {
		h.logger.Warn("Request timeout",
			"operation", operation,
			"path", r.URL.Path,
			"method", r.Method,
			"error", err)
		return true
	}
	return false
}

// PaginationParams holds parsed pagination parameters
type PaginationParams struct {
	Limit  int
	Offset int
}

// DefaultPaginationLimit is the default number of items per page
const DefaultPaginationLimit = 100

// ParsePagination extracts and validates pagination parameters from a request.
// Returns default values (limit=100, offset=0) if parameters are missing or invalid.
func ParsePagination(r *http.Request) PaginationParams {
	return ParsePaginationWithDefaults(r, DefaultPaginationLimit, 0)
}

// ParsePaginationWithDefaults extracts pagination parameters with custom defaults.
func ParsePaginationWithDefaults(r *http.Request, defaultLimit, defaultOffset int) PaginationParams {
	limit := defaultLimit
	offset := defaultOffset

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
}

// ParsePageParams extracts page-based pagination and converts to limit/offset.
// Useful for endpoints that use "page" and "per_page" instead of "offset" and "limit".
func ParsePageParams(r *http.Request, defaultPerPage int) PaginationParams {
	perPage := defaultPerPage
	page := 1

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			perPage = pp
		}
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	return PaginationParams{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}
}

// checkRepositoryAccess validates that the user has access to a specific repository
// Returns an error if auth is enabled and user doesn't have access
func (h *Handler) checkRepositoryAccess(ctx context.Context, repoFullName string) error {
	// If auth is not enabled, allow access
	if !h.authConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if h.sourceDualClient == nil {
		h.logger.Warn("Cannot check repository access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := h.sourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, h.authConfig, h.logger, h.sourceBaseURL)

	// Check if user has access to this repository
	hasAccess, err := checker.HasRepoAccess(ctx, user, token, repoFullName)
	if err != nil {
		return fmt.Errorf("failed to check repository access: %w", err)
	}

	if !hasAccess {
		return fmt.Errorf("you don't have admin access to repository: %s", repoFullName)
	}

	return nil
}

// checkRepositoriesAccess validates that the user has access to all specified repositories
// Returns an error if auth is enabled and user doesn't have access to any repository
func (h *Handler) checkRepositoriesAccess(ctx context.Context, repoFullNames []string) error {
	// If auth is not enabled, allow access
	if !h.authConfig.Enabled {
		return nil
	}

	user, hasUser := auth.GetUserFromContext(ctx)
	token, hasToken := auth.GetTokenFromContext(ctx)

	if !hasUser || !hasToken {
		return fmt.Errorf("authentication required")
	}

	if h.sourceDualClient == nil {
		h.logger.Warn("Cannot check repositories access: source client not available")
		return nil // Allow access if we can't check
	}

	// Create permission checker
	apiClient := h.sourceDualClient.APIClient()
	checker := auth.NewPermissionChecker(apiClient, h.authConfig, h.logger, h.sourceBaseURL)

	// Validate access to all repositories
	return checker.ValidateRepositoryAccess(ctx, user, token, repoFullNames)
}

// Helper functions

// statusIn checks if a status is in the given list of allowed statuses.
func statusIn(status string, allowed []string) bool {
	for _, s := range allowed {
		if status == s {
			return true
		}
	}
	return false
}

// batchEligibleStatuses defines the statuses that make a repository eligible for batch assignment.
// This is the base set used by both batch assignment and migration eligibility checks.
var batchEligibleStatuses = []string{
	string(models.StatusPending),
	string(models.StatusDryRunComplete),
	string(models.StatusDryRunFailed),
	string(models.StatusMigrationFailed),
	string(models.StatusRolledBack),
}

// migrationAllowedStatuses extends batchEligibleStatuses with additional statuses
// that allow re-queuing for migration (like DryRunQueued for re-runs).
var migrationAllowedStatuses = append(
	batchEligibleStatuses,
	string(models.StatusDryRunQueued), // Allow re-queuing dry runs
)

func canMigrate(status string) bool {
	// Cannot migrate repositories marked as wont_migrate
	if status == string(models.StatusWontMigrate) {
		return false
	}
	return statusIn(status, migrationAllowedStatuses)
}

func isEligibleForBatch(status string) bool {
	return statusIn(status, batchEligibleStatuses)
}

// getInitiatingUser extracts the authenticated username from the context
// Returns nil if auth is disabled or user not found
func getInitiatingUser(ctx context.Context) *string {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil
	}
	username := user.Login
	return &username
}

func isRepositoryEligibleForBatch(repo *models.Repository) (bool, string) {
	// Check if already in a batch
	if repo.BatchID != nil {
		return false, "repository is already assigned to a batch"
	}

	// Check if repository exceeds GitHub's 40 GiB size limit
	if repo.HasOversizedRepository {
		return false, "repository exceeds GitHub's 40 GiB size limit and requires remediation"
	}

	// Check if status is eligible
	if !isEligibleForBatch(repo.Status) {
		return false, fmt.Sprintf("repository status '%s' is not eligible for batch assignment", repo.Status)
	}

	return true, ""
}

// CSV helper functions

// escapeCSV escapes a string for safe inclusion in a CSV field.
// It wraps the string in quotes and escapes internal quotes if the string
// contains commas, quotes, newlines, or carriage returns.
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stringPtr returns a pointer to a heap-allocated copy of the string.
func stringPtr(s string) *string {
	ptr := new(string)
	*ptr = s
	return ptr
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func intPtrOrZero(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func formatVisibilityForDisplay(visibility string) string {
	switch visibility {
	case "public":
		return "Public"
	case "private":
		return "Private"
	case "internal":
		return "Internal"
	default:
		return visibility
	}
}
