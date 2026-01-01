package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// Database driver name constants
const (
	driverSQLite    = "sqlite3"
	driverPostgres  = "postgres"
	driverSQLServer = "sqlserver"
)

// SetupHandler handles setup wizard API requests
type SetupHandler struct {
	db           *storage.Database
	logger       *slog.Logger
	cfg          *config.Config
	shutdownChan chan struct{}
}

// NewSetupHandler creates a new SetupHandler
func NewSetupHandler(db *storage.Database, logger *slog.Logger, cfg *config.Config, shutdownChan chan struct{}) *SetupHandler {
	return &SetupHandler{
		db:           db,
		logger:       logger,
		cfg:          cfg,
		shutdownChan: shutdownChan,
	}
}

// SetupStatusResponse represents the response for setup status
type SetupStatusResponse struct {
	SetupCompleted bool              `json:"setup_completed"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	CurrentConfig  *MaskedConfigData `json:"current_config,omitempty"`
}

// MaskedConfigData contains current configuration with sensitive data masked
type MaskedConfigData struct {
	SourceType    string `json:"source_type"`
	SourceBaseURL string `json:"source_base_url"`
	SourceToken   string `json:"source_token"` // masked
	DestBaseURL   string `json:"dest_base_url"`
	DestToken     string `json:"dest_token"` // masked
	DatabaseType  string `json:"database_type"`
	DatabaseDSN   string `json:"database_dsn"` // masked
	ServerPort    int    `json:"server_port"`
}

// ValidationRequest represents a validation request for source/destination
type ValidationRequest struct {
	Type         string `json:"type"` // github or azuredevops
	BaseURL      string `json:"base_url"`
	Token        string `json:"token"`
	Organization string `json:"organization"` // for Azure DevOps
}

// DatabaseValidationRequest represents a database validation request
type DatabaseValidationRequest struct {
	Type string `json:"type"` // sqlite, postgres, or sqlserver
	DSN  string `json:"dsn"`
}

// ValidationResponse represents the validation result
type ValidationResponse struct {
	Valid    bool           `json:"valid"`
	Error    string         `json:"error,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

// SetupConfig represents the full setup configuration
type SetupConfig struct {
	Source      SourceConfigData      `json:"source"`
	Destination DestinationConfigData `json:"destination"`
	Database    DatabaseConfigData    `json:"database"`
	Server      ServerConfigData      `json:"server"`
	Migration   MigrationConfigData   `json:"migration"`
	Logging     LoggingConfigData     `json:"logging"`
	Auth        *AuthConfigData       `json:"auth,omitempty"`
}

type SourceConfigData struct {
	Type         string `json:"type"`
	BaseURL      string `json:"base_url"`
	Token        string `json:"token"`
	Organization string `json:"organization,omitempty"`
	// GitHub App for source discovery (optional, only when source type is github)
	AppID             int64  `json:"app_id,omitempty"`
	AppPrivateKey     string `json:"app_private_key,omitempty"`
	AppInstallationID int64  `json:"app_installation_id,omitempty"`
}

type DestinationConfigData struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
	// GitHub App for enhanced discovery (optional, destination is always GitHub)
	AppID             int64  `json:"app_id,omitempty"`
	AppPrivateKey     string `json:"app_private_key,omitempty"`
	AppInstallationID int64  `json:"app_installation_id,omitempty"`
}

type DatabaseConfigData struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}

type ServerConfigData struct {
	Port int `json:"port"`
}

type MigrationConfigData struct {
	Workers              int                          `json:"workers"`
	PollIntervalSeconds  int                          `json:"poll_interval_seconds"`
	DestRepoExistsAction string                       `json:"dest_repo_exists_action"`
	VisibilityHandling   VisibilityHandlingConfigData `json:"visibility_handling"`
}

type VisibilityHandlingConfigData struct {
	PublicRepos   string `json:"public_repos"`
	InternalRepos string `json:"internal_repos"`
}

type AuthConfigData struct {
	Enabled bool `json:"enabled"`
	// GitHub OAuth (when source/destination is GitHub)
	GitHubOAuthClientID     string `json:"github_oauth_client_id,omitempty"`
	GitHubOAuthClientSecret string `json:"github_oauth_client_secret,omitempty"`
	GitHubOAuthBaseURL      string `json:"github_oauth_base_url,omitempty"`
	// Azure AD (when source is Azure DevOps)
	AzureADTenantID     string `json:"azure_ad_tenant_id,omitempty"`
	AzureADClientID     string `json:"azure_ad_client_id,omitempty"`
	AzureADClientSecret string `json:"azure_ad_client_secret,omitempty"`
	// Common settings
	CallbackURL          string `json:"callback_url,omitempty"`
	FrontendURL          string `json:"frontend_url,omitempty"`
	SessionSecret        string `json:"session_secret,omitempty"`
	SessionDurationHours int    `json:"session_duration_hours,omitempty"`
	// Authorization rules (optional)
	AuthorizationRules *AuthorizationRulesData `json:"authorization_rules,omitempty"`
}

type AuthorizationRulesData struct {
	RequireOrgMembership        []string `json:"require_org_membership,omitempty"`        // List of org names
	RequireTeamMembership       []string `json:"require_team_membership,omitempty"`       // List of "org/team-slug"
	RequireEnterpriseAdmin      bool     `json:"require_enterprise_admin,omitempty"`      // Require GitHub Enterprise admin role
	RequireEnterpriseMembership bool     `json:"require_enterprise_membership,omitempty"` // Require enterprise membership
	EnterpriseSlug              string   `json:"enterprise_slug,omitempty"`               // Enterprise slug
	PrivilegedTeams             []string `json:"privileged_teams,omitempty"`              // Privileged teams with full access
}

type LoggingConfigData struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputFile string `json:"output_file"`
}

// GetSetupStatus returns the current setup status
func (h *SetupHandler) GetSetupStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.db.GetSetupStatus()
	if err != nil {
		h.logger.Error("Failed to get setup status", "error", err)
		http.Error(w, "Failed to get setup status", http.StatusInternalServerError)
		return
	}

	// If setup is not marked complete in DB, check if config exists via environment variables
	// This handles container deployments where config is provided via env vars
	if !status.SetupCompleted && h.cfg != nil && h.hasRequiredConfig() {
		h.logger.Info("Configuration detected via environment variables, marking setup as complete")
		if err := h.db.MarkSetupComplete(); err != nil {
			h.logger.Error("Failed to mark setup complete", "error", err)
			// Don't fail the request, just log the error
		} else {
			// Re-fetch status to get the updated values
			updatedStatus, err := h.db.GetSetupStatus()
			if err != nil {
				h.logger.Error("Failed to get updated setup status", "error", err)
				// Continue with original status if re-fetch fails
			} else {
				status = updatedStatus
			}
		}
	}

	response := SetupStatusResponse{
		SetupCompleted: status.SetupCompleted,
		CompletedAt:    status.CompletedAt,
	}

	// If setup is complete, include masked current config
	if status.SetupCompleted && h.cfg != nil {
		response.CurrentConfig = &MaskedConfigData{
			SourceType:    h.cfg.Source.Type,
			SourceBaseURL: h.cfg.Source.BaseURL,
			SourceToken:   maskToken(h.cfg.Source.Token),
			DestBaseURL:   h.cfg.Destination.BaseURL,
			DestToken:     maskToken(h.cfg.Destination.Token),
			DatabaseType:  h.cfg.Database.Type,
			DatabaseDSN:   maskDSN(h.cfg.Database.DSN),
			ServerPort:    h.cfg.Server.Port,
		}
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ValidateSource validates the source connection (GitHub or Azure DevOps)
func (h *SetupHandler) ValidateSource(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	response := ValidationResponse{Details: make(map[string]any)}

	switch strings.ToLower(req.Type) {
	case models.SourceTypeGitHub:
		response = ValidateGitHubConnection(ctx, req.BaseURL, req.Token, h.logger)
	case models.SourceTypeAzureDevOps:
		response = h.validateAzureDevOps(ctx, req.BaseURL, req.Token, req.Organization)
	default:
		response.Valid = false
		response.Error = fmt.Sprintf("Unsupported source type: %s", req.Type)
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ValidateDestination validates the destination GitHub connection
func (h *SetupHandler) ValidateDestination(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	response := ValidateGitHubConnection(ctx, req.BaseURL, req.Token, h.logger)

	h.sendJSON(w, http.StatusOK, response)
}

// ValidateDatabase validates the database connection
func (h *SetupHandler) ValidateDatabase(w http.ResponseWriter, r *http.Request) {
	var req DatabaseValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := h.validateDatabaseConnection(req.Type, req.DSN)
	h.sendJSON(w, http.StatusOK, response)
}

// ApplySetup applies the configuration and triggers server restart
func (h *SetupHandler) ApplySetup(w http.ResponseWriter, r *http.Request) {
	var setupCfg SetupConfig
	if err := json.NewDecoder(r.Body).Decode(&setupCfg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if setupCfg.Source.Type == "" || setupCfg.Source.BaseURL == "" || setupCfg.Source.Token == "" {
		http.Error(w, "Source configuration is incomplete", http.StatusBadRequest)
		return
	}
	if setupCfg.Destination.BaseURL == "" || setupCfg.Destination.Token == "" {
		http.Error(w, "Destination configuration is incomplete", http.StatusBadRequest)
		return
	}
	if setupCfg.Database.Type == "" || setupCfg.Database.DSN == "" {
		http.Error(w, "Database configuration is incomplete", http.StatusBadRequest)
		return
	}

	// Generate .env file content
	envContent := h.generateEnvFile(setupCfg)

	// Write .env file atomically
	if err := h.writeEnvFile(envContent); err != nil {
		h.logger.Error("Failed to write .env file", "error", err)
		http.Error(w, "Failed to write configuration file", http.StatusInternalServerError)
		return
	}

	// Mark setup as complete in database
	if err := h.db.MarkSetupComplete(); err != nil {
		h.logger.Error("Failed to mark setup complete", "error", err)
		http.Error(w, "Failed to save setup status", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Setup completed successfully")
	h.logger.Info("Configuration has been saved. Server will shutdown to apply changes.")

	// Send success response
	h.sendJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"message":        "Configuration applied successfully. Server will restart.",
		"restart_needed": true,
	})

	// Trigger graceful shutdown after response is sent
	// Use a goroutine with a small delay to ensure the response reaches the client
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.logger.Info("Triggering server shutdown for configuration reload...")
		close(h.shutdownChan)
	}()
}

// validateAzureDevOps validates an Azure DevOps connection
func (h *SetupHandler) validateAzureDevOps(ctx context.Context, orgURL, token, organization string) ValidationResponse {
	response := ValidationResponse{Details: make(map[string]any)}

	// Create temporary ADO client
	client, err := azuredevops.NewClient(azuredevops.ClientConfig{
		OrganizationURL:     orgURL,
		PersonalAccessToken: token,
		Logger:              h.logger,
	})
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to create Azure DevOps client: %v", err)
		return response
	}

	// Test connection by listing projects
	projects, err := client.GetProjects(ctx)
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to connect to Azure DevOps: %v", err)
		return response
	}

	response.Valid = true
	response.Details["organization_url"] = orgURL
	response.Details["project_count"] = len(projects)

	if len(projects) == 0 {
		response.Warnings = append(response.Warnings, "No projects found in organization")
	}

	return response
}

// validateSQLitePath validates that a SQLite database path is secure
func (h *SetupHandler) validateSQLitePath(dsn string) ValidationResponse {
	response := ValidationResponse{Details: make(map[string]any)}
	const safeBaseDir = "./data"

	// Extract directory from DSN
	var dir string
	if strings.Contains(dsn, "/") {
		lastSlash := strings.LastIndex(dsn, "/")
		dir = dsn[:lastSlash]
	} else {
		dir = "."
	}

	// Validate directory against safe base dir to prevent path traversal
	absSafeBaseDir, err := filepath.Abs(safeBaseDir)
	if err != nil {
		response.Valid = false
		response.Error = "Server misconfiguration: unable to resolve base directory"
		return response
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Invalid directory: %v", err)
		return response
	}
	// Ensure absDir is within or equal to absSafeBaseDir using filepath.Rel
	// This is more robust than HasPrefix and handles symlinks better
	rel, err := filepath.Rel(absSafeBaseDir, absDir)
	if err != nil || strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)+"..") {
		response.Valid = false
		response.Error = fmt.Sprintf("Database directory must be inside %s", safeBaseDir)
		return response
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		response.Valid = false
		response.Error = fmt.Sprintf("Directory does not exist: %s", dir)
		return response
	}

	response.Valid = true
	return response
}

// validateDatabaseConnection validates a database connection
func (h *SetupHandler) validateDatabaseConnection(dbType, dsn string) ValidationResponse {
	response := ValidationResponse{Details: make(map[string]any)}

	var driverName string
	switch strings.ToLower(dbType) {
	case "sqlite":
		driverName = driverSQLite
	case "postgres", "postgresql":
		driverName = driverPostgres
	case "sqlserver":
		driverName = driverSQLServer
	default:
		response.Valid = false
		response.Error = fmt.Sprintf("Unsupported database type: %s", dbType)
		return response
	}

	// For SQLite, validate path security first
	if driverName == driverSQLite {
		pathValidation := h.validateSQLitePath(dsn)
		if !pathValidation.Valid {
			return pathValidation
		}
	}

	// Test database connection for all types
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to open database: %v", err)
		return response
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		response.Valid = false
		response.Error = fmt.Sprintf("Failed to connect to database: %v", err)
		return response
	}

	response.Valid = true
	response.Details["type"] = dbType
	response.Details["connected"] = true
	if driverName == driverSQLite {
		response.Details["path"] = dsn
	}

	return response
}

// generateEnvFile generates the .env file content from setup configuration
func (h *SetupHandler) generateEnvFile(cfg SetupConfig) string {
	var sb strings.Builder

	sb.WriteString("# GitHub Migrator Configuration\n")
	sb.WriteString("# Generated by setup wizard on " + time.Now().Format(time.RFC3339) + "\n\n")

	h.writeSourceConfig(&sb, cfg.Source)
	h.writeDestinationConfig(&sb, cfg.Destination)
	h.writeDatabaseConfig(&sb, cfg.Database)
	h.writeServerConfig(&sb, cfg.Server)
	h.writeMigrationConfig(&sb, cfg.Migration)
	h.writeLoggingConfig(&sb, cfg.Logging)
	h.writeAuthConfig(&sb, cfg.Auth)

	return sb.String()
}

// writeSourceConfig writes source configuration to the env file
// Skips writing if token is empty or a placeholder (sources are now configured via Sources page)
func (h *SetupHandler) writeSourceConfig(sb *strings.Builder, src SourceConfigData) {
	// Skip source config if no real token is provided
	// Sources are now managed via the Sources page, not the initial setup
	if src.Token == "" || src.Token == "placeholder" {
		sb.WriteString("# Source configuration - configure via Sources page after setup\n\n")
		return
	}

	sb.WriteString("# Source Repository System Configuration\n")
	sb.WriteString(fmt.Sprintf("GHMIG_SOURCE_TYPE=%s\n", src.Type))
	sb.WriteString(fmt.Sprintf("GHMIG_SOURCE_BASE_URL=%s\n", src.BaseURL))
	sb.WriteString(fmt.Sprintf("GHMIG_SOURCE_TOKEN=%s\n", src.Token))
	if src.Organization != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_SOURCE_ORGANIZATION=%s\n", src.Organization))
	}

	// GitHub App for source (only when source is GitHub)
	if src.Type == models.SourceTypeGitHub && src.AppID > 0 {
		sb.WriteString("\n# GitHub App Configuration for Source (Optional)\n")
		sb.WriteString(fmt.Sprintf("GHMIG_SOURCE_APP_ID=%d\n", src.AppID))
		if src.AppPrivateKey != "" {
			sb.WriteString(fmt.Sprintf("GHMIG_SOURCE_APP_PRIVATE_KEY=\"%s\"\n", escapeEnvValue(src.AppPrivateKey)))
		}
		if src.AppInstallationID > 0 {
			sb.WriteString(fmt.Sprintf("GHMIG_SOURCE_APP_INSTALLATION_ID=%d\n", src.AppInstallationID))
		}
	}
	sb.WriteString("\n")
}

// writeDestinationConfig writes destination configuration to the env file
// Skips writing if token is empty or a placeholder (destination is now configured via Settings page)
func (h *SetupHandler) writeDestinationConfig(sb *strings.Builder, dest DestinationConfigData) {
	// Skip destination config if no real token is provided
	// Destination is now managed via the Settings page, not the initial setup
	if dest.Token == "" || dest.Token == "placeholder" {
		sb.WriteString("# Destination configuration - configure via Settings page after setup\n\n")
		return
	}

	sb.WriteString("# Destination Repository System Configuration\n")
	sb.WriteString("GHMIG_DESTINATION_TYPE=github\n")
	sb.WriteString(fmt.Sprintf("GHMIG_DESTINATION_BASE_URL=%s\n", dest.BaseURL))
	sb.WriteString(fmt.Sprintf("GHMIG_DESTINATION_TOKEN=%s\n", dest.Token))

	// GitHub App for destination (always available since destination is GitHub)
	if dest.AppID > 0 {
		sb.WriteString("\n# GitHub App Configuration for Destination (Optional)\n")
		sb.WriteString(fmt.Sprintf("GHMIG_DESTINATION_APP_ID=%d\n", dest.AppID))
		if dest.AppPrivateKey != "" {
			sb.WriteString(fmt.Sprintf("GHMIG_DESTINATION_APP_PRIVATE_KEY=\"%s\"\n", escapeEnvValue(dest.AppPrivateKey)))
		}
		if dest.AppInstallationID > 0 {
			sb.WriteString(fmt.Sprintf("GHMIG_DESTINATION_APP_INSTALLATION_ID=%d\n", dest.AppInstallationID))
		}
	}
	sb.WriteString("\n")
}

// writeDatabaseConfig writes database configuration to the env file
func (h *SetupHandler) writeDatabaseConfig(sb *strings.Builder, db DatabaseConfigData) {
	sb.WriteString("# Database Configuration\n")
	sb.WriteString(fmt.Sprintf("GHMIG_DATABASE_TYPE=%s\n", db.Type))
	// Quote DSN as it often contains special characters
	sb.WriteString(fmt.Sprintf("GHMIG_DATABASE_DSN=\"%s\"\n", escapeEnvValue(db.DSN)))
	sb.WriteString("\n")
}

// writeServerConfig writes server configuration to the env file
func (h *SetupHandler) writeServerConfig(sb *strings.Builder, srv ServerConfigData) {
	sb.WriteString("# Server Configuration\n")
	sb.WriteString(fmt.Sprintf("GHMIG_SERVER_PORT=%d\n", srv.Port))
	sb.WriteString("\n")
}

// writeMigrationConfig writes migration configuration to the env file
func (h *SetupHandler) writeMigrationConfig(sb *strings.Builder, mig MigrationConfigData) {
	sb.WriteString("# Migration Configuration\n")
	sb.WriteString(fmt.Sprintf("GHMIG_MIGRATION_WORKERS=%d\n", mig.Workers))
	sb.WriteString(fmt.Sprintf("GHMIG_MIGRATION_POLL_INTERVAL_SECONDS=%d\n", mig.PollIntervalSeconds))
	sb.WriteString(fmt.Sprintf("GHMIG_MIGRATION_DEST_REPO_EXISTS_ACTION=%s\n", mig.DestRepoExistsAction))
	sb.WriteString(fmt.Sprintf("GHMIG_MIGRATION_VISIBILITY_HANDLING_PUBLIC_REPOS=%s\n", mig.VisibilityHandling.PublicRepos))
	sb.WriteString(fmt.Sprintf("GHMIG_MIGRATION_VISIBILITY_HANDLING_INTERNAL_REPOS=%s\n", mig.VisibilityHandling.InternalRepos))
	sb.WriteString("\n")
}

// writeLoggingConfig writes logging configuration to the env file
func (h *SetupHandler) writeLoggingConfig(sb *strings.Builder, log LoggingConfigData) {
	sb.WriteString("# Logging Configuration\n")
	sb.WriteString(fmt.Sprintf("GHMIG_LOGGING_LEVEL=%s\n", log.Level))
	sb.WriteString(fmt.Sprintf("GHMIG_LOGGING_FORMAT=%s\n", log.Format))
	sb.WriteString(fmt.Sprintf("GHMIG_LOGGING_OUTPUT_FILE=%s\n", log.OutputFile))
	sb.WriteString("\n")
}

// writeAuthConfig writes authentication configuration to the env file
func (h *SetupHandler) writeAuthConfig(sb *strings.Builder, auth *AuthConfigData) {
	if auth == nil || !auth.Enabled {
		return
	}

	sb.WriteString("# Authentication Configuration (Optional)\n")
	sb.WriteString("GHMIG_AUTH_ENABLED=true\n")

	h.writeGitHubOAuthConfig(sb, auth)
	h.writeAzureADConfig(sb, auth)
	h.writeCommonAuthConfig(sb, auth)
	sb.WriteString("\n")
}

// writeGitHubOAuthConfig writes GitHub OAuth settings
func (h *SetupHandler) writeGitHubOAuthConfig(sb *strings.Builder, auth *AuthConfigData) {
	if auth.GitHubOAuthClientID != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID=%s\n", auth.GitHubOAuthClientID))
	}
	if auth.GitHubOAuthClientSecret != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET=%s\n", auth.GitHubOAuthClientSecret))
	}
	if auth.GitHubOAuthBaseURL != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_GITHUB_OAUTH_BASE_URL=%s\n", auth.GitHubOAuthBaseURL))
	}
}

// writeAzureADConfig writes Azure AD settings
func (h *SetupHandler) writeAzureADConfig(sb *strings.Builder, auth *AuthConfigData) {
	if auth.AzureADTenantID != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AZURE_AD_TENANT_ID=%s\n", auth.AzureADTenantID))
	}
	if auth.AzureADClientID != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AZURE_AD_CLIENT_ID=%s\n", auth.AzureADClientID))
	}
	if auth.AzureADClientSecret != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AZURE_AD_CLIENT_SECRET=%s\n", auth.AzureADClientSecret))
	}
}

// writeCommonAuthConfig writes common authentication settings
func (h *SetupHandler) writeCommonAuthConfig(sb *strings.Builder, auth *AuthConfigData) {
	if auth.CallbackURL != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_CALLBACK_URL=%s\n", auth.CallbackURL))
	}
	if auth.FrontendURL != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_FRONTEND_URL=%s\n", auth.FrontendURL))
	}
	if auth.SessionSecret != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_SESSION_SECRET=%s\n", auth.SessionSecret))
	}
	if auth.SessionDurationHours > 0 {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_SESSION_DURATION_HOURS=%d\n", auth.SessionDurationHours))
	}

	// Write authorization rules if present
	h.writeAuthorizationRules(sb, auth.AuthorizationRules)
}

// writeAuthorizationRules writes authorization rules configuration
func (h *SetupHandler) writeAuthorizationRules(sb *strings.Builder, rules *AuthorizationRulesData) {
	if rules == nil {
		return
	}

	sb.WriteString("\n# Authorization Rules (Optional)\n")

	if len(rules.RequireOrgMembership) > 0 {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ORG_MEMBERSHIP=%s\n",
			strings.Join(rules.RequireOrgMembership, ",")))
	}

	if len(rules.RequireTeamMembership) > 0 {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_TEAM_MEMBERSHIP=%s\n",
			strings.Join(rules.RequireTeamMembership, ",")))
	}

	if rules.RequireEnterpriseAdmin {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_ADMIN=%t\n", rules.RequireEnterpriseAdmin))
	}

	if rules.RequireEnterpriseMembership {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_MEMBERSHIP=%t\n", rules.RequireEnterpriseMembership))
	}

	if rules.EnterpriseSlug != "" {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_SLUG=%s\n", rules.EnterpriseSlug))
	}

	if len(rules.PrivilegedTeams) > 0 {
		sb.WriteString(fmt.Sprintf("GHMIG_AUTH_AUTHORIZATION_RULES_PRIVILEGED_TEAMS=%s\n",
			strings.Join(rules.PrivilegedTeams, ",")))
	}
}

// writeEnvFile writes the .env file atomically
// escapeEnvValue escapes a value for use in a .env file
// It replaces newlines with the literal string \n and escapes quotes and backslashes
func escapeEnvValue(value string) string {
	// Replace actual newlines with the literal string \n
	value = strings.ReplaceAll(value, "\n", "\\n")
	// Escape double quotes
	value = strings.ReplaceAll(value, "\"", "\\\"")
	// Note: backslashes are already handled by the \n replacement above
	return value
}

func (h *SetupHandler) writeEnvFile(content string) error {
	envPath := ".env"
	tempPath := ".env.tmp"

	// Write to temporary file
	if err := os.WriteFile(tempPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename to final location (atomic operation)
	if err := os.Rename(tempPath, envPath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tempPath) // Ignore error on cleanup
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	h.logger.Info("Configuration file written successfully", "path", envPath)
	return nil
}

// Helper functions

// hasRequiredConfig checks if all critical configuration exists
// This is used to detect container deployments with env vars
func (h *SetupHandler) hasRequiredConfig() bool {
	if h.cfg == nil {
		return false
	}

	// Check critical source configuration
	// Source.Type is required per ApplySetup validation (line 279)
	hasSource := h.cfg.Source.Type != "" && h.cfg.Source.Token != "" && h.cfg.Source.BaseURL != ""

	// Check critical destination configuration
	hasDestination := h.cfg.Destination.Token != "" && h.cfg.Destination.BaseURL != ""

	// Check critical database configuration
	hasDatabase := h.cfg.Database.DSN != ""

	return hasSource && hasDestination && hasDatabase
}

func (h *SetupHandler) sendJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// maskToken masks a token, showing only the last 4 characters
func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return "****" + token[len(token)-4:]
}

// maskDSN masks sensitive parts of a database DSN
func maskDSN(dsn string) string {
	// For SQLite, just return as-is (it's a file path)
	if !strings.Contains(dsn, "://") && !strings.Contains(dsn, "@") {
		return dsn
	}

	// For connection strings with passwords, mask them
	if strings.Contains(dsn, "@") {
		parts := strings.Split(dsn, "@")
		if len(parts) >= 2 {
			// Mask the user:password part
			beforeAt := parts[0]
			if strings.Contains(beforeAt, ":") {
				userPass := strings.Split(beforeAt, ":")
				if len(userPass) >= 2 {
					return userPass[0] + ":****@" + strings.Join(parts[1:], "@")
				}
			}
			return "****@" + strings.Join(parts[1:], "@")
		}
	}

	return dsn
}
