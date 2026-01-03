package models

import (
	"errors"
	"strings"
	"time"
)

// Source type constants for migration sources.
const (
	SourceConfigTypeGitHub      = "github"
	SourceConfigTypeAzureDevOps = "azuredevops"
)

// Source represents a configured migration source (e.g., GitHub Enterprise Server, Azure DevOps).
// Sources are stored in the database and can be managed via the UI.
// All repositories discovered from a source are linked via the source_id foreign key.
type Source struct {
	ID   int64  `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" db:"name" gorm:"column:name;uniqueIndex;not null"` // User-friendly name (e.g., "GHES Production", "ADO Main")

	// Connection configuration
	Type           string  `json:"type" db:"type" gorm:"column:type;not null;index"`                             // "github" or "azuredevops"
	BaseURL        string  `json:"base_url" db:"base_url" gorm:"column:base_url;not null"`                       // API base URL
	Token          string  `json:"-" db:"token" gorm:"column:token;not null"`                                    // PAT token (excluded from JSON serialization)
	Organization   *string `json:"organization,omitempty" db:"organization" gorm:"column:organization"`          // Required for Azure DevOps (top-level container)
	EnterpriseSlug *string `json:"enterprise_slug,omitempty" db:"enterprise_slug" gorm:"column:enterprise_slug"` // Optional for GitHub (top-level container for enterprise discovery)

	// GitHub App authentication (optional, for enhanced discovery)
	AppID             *int64  `json:"app_id,omitempty" db:"app_id" gorm:"column:app_id"`
	AppPrivateKey     *string `json:"-" db:"app_private_key" gorm:"column:app_private_key;type:text"` // Excluded from JSON
	AppInstallationID *int64  `json:"app_installation_id,omitempty" db:"app_installation_id" gorm:"column:app_installation_id"`

	// OAuth configuration (optional, enables user self-service authentication)
	// For GitHub/GHES sources
	OAuthClientID     *string `json:"-" db:"oauth_client_id" gorm:"column:oauth_client_id"`
	OAuthClientSecret *string `json:"-" db:"oauth_client_secret" gorm:"column:oauth_client_secret"`

	// For Azure DevOps sources (Entra ID OAuth)
	EntraTenantID     *string `json:"-" db:"entra_tenant_id" gorm:"column:entra_tenant_id"`
	EntraClientID     *string `json:"-" db:"entra_client_id" gorm:"column:entra_client_id"`
	EntraClientSecret *string `json:"-" db:"entra_client_secret" gorm:"column:entra_client_secret"`

	// Status
	IsActive bool `json:"is_active" db:"is_active" gorm:"column:is_active;default:true;index"`

	// Metadata
	RepositoryCount int        `json:"repository_count" db:"repository_count" gorm:"column:repository_count;default:0"` // Cached count of repos from this source
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at" gorm:"column:last_sync_at"`             // Last successful discovery
	CreatedAt       time.Time  `json:"created_at" db:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName specifies the table name for Source
func (Source) TableName() string {
	return "sources"
}

// Validation errors
var (
	ErrSourceNameRequired    = errors.New("source name is required")
	ErrSourceNameTooLong     = errors.New("source name must be 100 characters or less")
	ErrSourceTypeRequired    = errors.New("source type is required")
	ErrSourceTypeInvalid     = errors.New("source type must be 'github' or 'azuredevops'")
	ErrSourceBaseURLRequired = errors.New("source base URL is required")
	ErrSourceTokenRequired   = errors.New("source token is required")
	ErrSourceOrgRequired     = errors.New("organization is required for Azure DevOps sources")
)

// Validate checks if the source configuration is valid
func (s *Source) Validate() error {
	// Name validation
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return ErrSourceNameRequired
	}
	if len(s.Name) > 100 {
		return ErrSourceNameTooLong
	}

	// Type validation
	s.Type = strings.ToLower(strings.TrimSpace(s.Type))
	if s.Type == "" {
		return ErrSourceTypeRequired
	}
	if s.Type != SourceConfigTypeGitHub && s.Type != SourceConfigTypeAzureDevOps {
		return ErrSourceTypeInvalid
	}

	// URL validation
	s.BaseURL = strings.TrimSpace(s.BaseURL)
	if s.BaseURL == "" {
		return ErrSourceBaseURLRequired
	}
	// Remove trailing slash for consistency
	s.BaseURL = strings.TrimSuffix(s.BaseURL, "/")

	// Token validation
	if strings.TrimSpace(s.Token) == "" {
		return ErrSourceTokenRequired
	}

	// Organization required for Azure DevOps
	if s.Type == SourceConfigTypeAzureDevOps {
		if s.Organization == nil || strings.TrimSpace(*s.Organization) == "" {
			return ErrSourceOrgRequired
		}
		org := strings.TrimSpace(*s.Organization)
		s.Organization = &org
	}

	// Trim enterprise slug if provided for GitHub sources
	if s.Type == SourceConfigTypeGitHub && s.EnterpriseSlug != nil {
		slug := strings.TrimSpace(*s.EnterpriseSlug)
		if slug == "" {
			s.EnterpriseSlug = nil // Clear if empty after trimming
		} else {
			s.EnterpriseSlug = &slug
		}
	}

	// Trim organization if provided for GitHub sources (optional for GitHub)
	if s.Type == SourceConfigTypeGitHub && s.Organization != nil {
		org := strings.TrimSpace(*s.Organization)
		if org == "" {
			s.Organization = nil // Clear if empty after trimming
		} else {
			s.Organization = &org
		}
	}

	return nil
}

// IsGitHub returns true if this is a GitHub source
func (s *Source) IsGitHub() bool {
	return s.Type == SourceConfigTypeGitHub
}

// IsAzureDevOps returns true if this is an Azure DevOps source
func (s *Source) IsAzureDevOps() bool {
	return s.Type == SourceConfigTypeAzureDevOps
}

// HasAppAuth returns true if GitHub App authentication is configured
func (s *Source) HasAppAuth() bool {
	return s.AppID != nil && *s.AppID > 0 && s.AppPrivateKey != nil && *s.AppPrivateKey != ""
}

// HasOAuth returns true if OAuth is configured for this source (enables user self-service)
func (s *Source) HasOAuth() bool {
	if s.IsGitHub() {
		return s.OAuthClientID != nil && *s.OAuthClientID != "" &&
			s.OAuthClientSecret != nil && *s.OAuthClientSecret != ""
	}
	if s.IsAzureDevOps() {
		return s.EntraTenantID != nil && *s.EntraTenantID != "" &&
			s.EntraClientID != nil && *s.EntraClientID != "" &&
			s.EntraClientSecret != nil && *s.EntraClientSecret != ""
	}
	return false
}

// MaskedToken returns a masked version of the token for display
func (s *Source) MaskedToken() string {
	if len(s.Token) <= 8 {
		return "****"
	}
	return s.Token[:4] + "..." + s.Token[len(s.Token)-4:]
}

// SourceResponse is a JSON-safe representation of Source for API responses.
// It excludes sensitive fields and adds computed fields.
type SourceResponse struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	BaseURL         string     `json:"base_url"`
	Organization    *string    `json:"organization,omitempty"`
	EnterpriseSlug  *string    `json:"enterprise_slug,omitempty"`
	HasAppAuth      bool       `json:"has_app_auth"`
	HasOAuth        bool       `json:"has_oauth"` // True if OAuth is configured for user self-service
	AppID           *int64     `json:"app_id,omitempty"`
	IsActive        bool       `json:"is_active"`
	RepositoryCount int        `json:"repository_count"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	MaskedToken     string     `json:"masked_token"`
}

// ToResponse converts a Source to a SourceResponse for API output
func (s *Source) ToResponse() *SourceResponse {
	return &SourceResponse{
		ID:              s.ID,
		Name:            s.Name,
		Type:            s.Type,
		BaseURL:         s.BaseURL,
		Organization:    s.Organization,
		EnterpriseSlug:  s.EnterpriseSlug,
		HasAppAuth:      s.HasAppAuth(),
		HasOAuth:        s.HasOAuth(),
		AppID:           s.AppID,
		IsActive:        s.IsActive,
		RepositoryCount: s.RepositoryCount,
		LastSyncAt:      s.LastSyncAt,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
		MaskedToken:     s.MaskedToken(),
	}
}

// SourcesToResponses converts a slice of Sources to SourceResponses
func SourcesToResponses(sources []*Source) []*SourceResponse {
	responses := make([]*SourceResponse, len(sources))
	for i, s := range sources {
		responses[i] = s.ToResponse()
	}
	return responses
}
