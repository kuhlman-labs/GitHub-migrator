package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
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
	handlerUtils := NewHandlerUtils(authConfig, sourceDualClient, sourceBaseConfig, sourceBaseURL, logger)
	handlerUtils.SetDatabase(db)

	return &Handler{
		HandlerUtils:   handlerUtils,
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

	response := map[string]any{
		"source_type":  sourceType,
		"auth_enabled": h.authConfig != nil && h.authConfig.Enabled,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// HandleAuthorizationStatus handles GET /api/v1/auth/authorization-status
// Returns the current user's authorization tier and permissions
func (h *Handler) HandleAuthorizationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := h.GetUserAuthorizationStatus(r.Context())
	if err != nil {
		h.logger.Error("Failed to get authorization status", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.sendJSON(w, http.StatusOK, status)
}

// sendJSON sends a JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, status int, data any) {
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

// getCollectorForSource returns a collector for the given source ID.
// If h.collector is already configured, it uses that.
// Otherwise, it creates a collector dynamically from the source's credentials in the database.
func (h *Handler) getCollectorForSource(sourceID *int64) (*discovery.Collector, error) {
	// If we have a pre-configured collector, use it
	if h.collector != nil {
		return h.collector, nil
	}

	// No pre-configured collector - we need a source ID to create one dynamically
	if sourceID == nil {
		return nil, fmt.Errorf("no GitHub client configured and no source_id provided")
	}

	// Get the database - need to type assert to *storage.Database
	db, ok := h.db.(*storage.Database)
	if !ok {
		return nil, fmt.Errorf("database type assertion failed - cannot create dynamic collector")
	}

	// Fetch the source from the database
	ctx := context.Background()
	src, err := db.GetSource(ctx, *sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source %d: %w", *sourceID, err)
	}
	if src == nil {
		return nil, fmt.Errorf("source %d not found", *sourceID)
	}

	// Only GitHub sources are supported for discovery
	if src.Type != models.SourceConfigTypeGitHub {
		return nil, fmt.Errorf("source %d is not a GitHub source (type: %s)", *sourceID, src.Type)
	}

	// Create GitHub client configuration with proper timeout and retry settings
	clientConfig := github.ClientConfig{
		BaseURL:     src.BaseURL,
		Token:       src.Token,
		Timeout:     120 * time.Second, // Match server initialization
		RetryConfig: github.DefaultRetryConfig(),
		Logger:      h.logger,
	}

	// Add App credentials if configured
	// Note: If AppInstallationID is nil/0, this creates a JWT-only client
	// which can discover app installations across an enterprise
	if src.HasAppAuth() {
		clientConfig.AppID = *src.AppID
		clientConfig.AppPrivateKey = *src.AppPrivateKey
		if src.AppInstallationID != nil && *src.AppInstallationID > 0 {
			clientConfig.AppInstallationID = *src.AppInstallationID
			h.logger.Info("Creating client with GitHub App installation auth",
				"source_id", *sourceID,
				"app_id", *src.AppID,
				"installation_id", *src.AppInstallationID)
		} else {
			// JWT-only mode for enterprise-wide discovery
			h.logger.Info("Creating client with GitHub App JWT-only auth (enterprise mode)",
				"source_id", *sourceID,
				"app_id", *src.AppID)
		}
	}

	// Create GitHub client
	client, err := github.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client for source %d: %w", *sourceID, err)
	}

	// Create source provider
	sourceProvider, err := source.NewGitHubProvider(src.BaseURL, src.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create source provider for source %d: %w", *sourceID, err)
	}

	// Create collector
	collector := discovery.NewCollector(client, db, h.logger, sourceProvider)

	// Set source ID to associate discovered entities with this source
	collector.SetSourceID(sourceID)

	// Set base config for App auth if available
	if src.HasAppAuth() {
		collector.WithBaseConfig(clientConfig)
	}

	h.logger.Info("Created dynamic collector for source", "source_id", *sourceID, "source_name", src.Name)

	return collector, nil
}

// getADOCollectorForSource returns an ADO collector for the given source ID.
// It creates a collector dynamically from the source's credentials in the database.
func (h *Handler) getADOCollectorForSource(sourceID *int64) (*discovery.ADOCollector, *azuredevops.Client, error) {
	// No source ID means we can't create a dynamic collector
	if sourceID == nil {
		return nil, nil, fmt.Errorf("no source_id provided for ADO discovery")
	}

	// Get the database - need to type assert to *storage.Database
	db, ok := h.db.(*storage.Database)
	if !ok {
		return nil, nil, fmt.Errorf("database type assertion failed - cannot create dynamic ADO collector")
	}

	// Fetch the source from the database
	ctx := context.Background()
	src, err := db.GetSource(ctx, *sourceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get source %d: %w", *sourceID, err)
	}
	if src == nil {
		return nil, nil, fmt.Errorf("source %d not found", *sourceID)
	}

	// Only Azure DevOps sources are supported for ADO discovery
	if src.Type != models.SourceConfigTypeAzureDevOps {
		return nil, nil, fmt.Errorf("source %d is not an Azure DevOps source (type: %s)", *sourceID, src.Type)
	}

	// Extract organization name from base URL or use stored organization field
	var orgName string
	if src.Organization != nil && *src.Organization != "" {
		orgName = *src.Organization
	} else {
		// Try to extract from base URL
		orgName = extractADOOrganization(src.BaseURL)
	}

	if orgName == "" {
		return nil, nil, fmt.Errorf("source %d: could not determine Azure DevOps organization. Please ensure the organization field is set or the base URL includes the organization (e.g., https://dev.azure.com/your-org)", *sourceID)
	}

	// Build the full organization URL by combining base URL and organization
	// If the organization is already in the URL, use it as-is; otherwise append it
	orgURL := src.BaseURL
	if !strings.Contains(src.BaseURL, orgName) {
		orgURL = strings.TrimSuffix(src.BaseURL, "/") + "/" + orgName
	}

	h.logger.Info("Creating ADO client for source",
		"source_id", *sourceID,
		"source_name", src.Name,
		"base_url", src.BaseURL,
		"organization", orgName,
		"org_url", orgURL)

	// Create Azure DevOps client
	adoClient, err := azuredevops.NewClient(azuredevops.ClientConfig{
		OrganizationURL:     orgURL,
		PersonalAccessToken: src.Token,
		Logger:              h.logger,
	})
	if err != nil {
		h.logger.Error("Failed to create ADO client",
			"source_id", *sourceID,
			"base_url", src.BaseURL,
			"organization", orgName,
			"org_url", orgURL,
			"error", err)
		return nil, nil, fmt.Errorf("failed to create ADO client for source %d: %w", *sourceID, err)
	}

	// Create source provider
	adoProvider, provErr := source.NewAzureDevOpsProvider(orgName, src.Token, "")
	if provErr != nil {
		return nil, nil, fmt.Errorf("failed to create ADO provider for source %d: %w", *sourceID, provErr)
	}

	h.logger.Info("Created dynamic ADO collector",
		"source_id", *sourceID,
		"source_name", src.Name,
		"organization", orgName)

	// Create ADO collector
	adoCollector := discovery.NewADOCollector(adoClient, db, h.logger, adoProvider)

	// Set source ID to associate discovered entities with this source
	adoCollector.SetSourceID(sourceID)

	return adoCollector, adoClient, nil
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

// Helper functions

// statusIn checks if a status is in the given list of allowed statuses.
func statusIn(status string, allowed []string) bool {
	return slices.Contains(allowed, status)
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

// extractADOOrganization extracts the organization name from an Azure DevOps URL.
// Supports: https://dev.azure.com/myorg or https://myorg.visualstudio.com
func extractADOOrganization(baseURL string) string {
	// Handle https://dev.azure.com/myorg format
	if strings.Contains(baseURL, "dev.azure.com") {
		parts := strings.Split(strings.TrimSuffix(baseURL, "/"), "/")
		if len(parts) >= 4 {
			return parts[3] // https://dev.azure.com/myorg -> myorg
		}
	}

	// Handle https://myorg.visualstudio.com format
	if strings.Contains(baseURL, ".visualstudio.com") {
		// Extract subdomain
		trimmed := strings.TrimPrefix(baseURL, "https://")
		trimmed = strings.TrimPrefix(trimmed, "http://")
		if idx := strings.Index(trimmed, "."); idx > 0 {
			return trimmed[:idx]
		}
	}

	// Fallback: return the URL as-is (might be a custom URL)
	return baseURL
}
