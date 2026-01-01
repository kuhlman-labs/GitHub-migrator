// Package handlers contains HTTP request and response types for the API.
package handlers

// ===== Batch Handlers =====

// RunDryRunRequest is the request body for running a dry run on a batch.
type RunDryRunRequest struct {
	OnlyPending bool `json:"only_pending,omitempty"`
}

// StartBatchRequest is the request body for starting a batch migration.
type StartBatchRequest struct {
	SkipDryRun bool `json:"skip_dry_run,omitempty"`
}

// BatchRepositoryIDsRequest is the request body for adding/removing repositories from a batch.
type BatchRepositoryIDsRequest struct {
	RepositoryIDs []int64 `json:"repository_ids"`
}

// RetryBatchRequest is the request body for retrying failed repositories in a batch.
type RetryBatchRequest struct {
	RepositoryIDs []int64 `json:"repository_ids,omitempty"`
}

// ===== Discovery Handlers =====

// StartDiscoveryRequest is the request body for starting repository discovery.
type StartDiscoveryRequest struct {
	Organization   string `json:"organization,omitempty"`
	EnterpriseSlug string `json:"enterprise_slug,omitempty"`
	Workers        int    `json:"workers,omitempty"`
	SourceID       *int64 `json:"source_id,omitempty"` // Optional: associate discovered repos with a source
}

// StartProfilingRequest is the request body for starting repository profiling.
type StartProfilingRequest struct {
	Organization string `json:"organization"`
	SourceID     *int64 `json:"source_id,omitempty"` // Optional: associate discovered repos with a source
}

// ===== User Mapping Handlers =====

// DiscoverUsersRequest is the request body for discovering users from an organization.
type DiscoverUsersRequest struct {
	Organization string `json:"organization"`
	SourceID     *int64 `json:"source_id,omitempty"` // Optional: use specific source for discovery
}

// CreateUserMappingRequest is the request body for creating a user mapping.
type CreateUserMappingRequest struct {
	SourceLogin      string  `json:"source_login"`
	SourceEmail      *string `json:"source_email,omitempty"`
	SourceName       *string `json:"source_name,omitempty"`
	DestinationLogin *string `json:"destination_login,omitempty"`
	DestinationEmail *string `json:"destination_email,omitempty"`
	MappingStatus    string  `json:"mapping_status,omitempty"`
}

// UpdateUserMappingRequest is the request body for updating a user mapping.
type UpdateUserMappingRequest struct {
	DestinationLogin *string `json:"destination_login,omitempty"`
	DestinationEmail *string `json:"destination_email,omitempty"`
	MappingStatus    *string `json:"mapping_status,omitempty"`
}

// ReconcileUsersRequest is the request body for reconciling users with destination.
type ReconcileUsersRequest struct {
	DestinationOrg string `json:"destination_org"`
	DryRun         bool   `json:"dry_run"`
}

// AutoMapUsersRequest is the request body for auto-mapping users.
type AutoMapUsersRequest struct {
	DestinationOrg string `json:"destination_org"`
	EMUShortcode   string `json:"emu_shortcode,omitempty"`
}

// ValidateMappingsRequest is the request body for validating user mappings.
type ValidateMappingsRequest struct {
	DestinationOrg string `json:"destination_org"`
}

// MigrateUsersRequest is the request body for migrating users.
type MigrateUsersRequest struct {
	DestinationOrg string   `json:"destination_org"`
	SourceLogins   []string `json:"source_logins,omitempty"`
}

// ===== Team Mapping Handlers =====

// DiscoverTeamsRequest is the request body for discovering teams from an organization.
type DiscoverTeamsRequest struct {
	Organization string `json:"organization"`
	SourceID     *int64 `json:"source_id,omitempty"` // Optional: use specific source for discovery
}

// CreateTeamMappingRequest is the request body for creating a team mapping.
type CreateTeamMappingRequest struct {
	SourceOrg           string  `json:"source_org"`
	SourceTeamSlug      string  `json:"source_team_slug"`
	SourceTeamName      *string `json:"source_team_name,omitempty"`
	DestinationOrg      *string `json:"destination_org,omitempty"`
	DestinationTeamSlug *string `json:"destination_team_slug,omitempty"`
	DestinationTeamName *string `json:"destination_team_name,omitempty"`
	MappingStatus       string  `json:"mapping_status,omitempty"`
}

// UpdateTeamMappingRequest is the request body for updating a team mapping.
type UpdateTeamMappingRequest struct {
	DestinationOrg      *string `json:"destination_org,omitempty"`
	DestinationTeamSlug *string `json:"destination_team_slug,omitempty"`
	DestinationTeamName *string `json:"destination_team_name,omitempty"`
	MappingStatus       *string `json:"mapping_status,omitempty"`
}

// SuggestTeamMappingsRequest is the request body for suggesting team mappings.
type SuggestTeamMappingsRequest struct {
	DestinationOrg string   `json:"destination_org"`
	DestTeamSlugs  []string `json:"dest_team_slugs"`
}

// MigrateTeamsRequest is the request body for migrating teams.
type MigrateTeamsRequest struct {
	SourceOrg      string `json:"source_org,omitempty"`
	SourceTeamSlug string `json:"source_team_slug,omitempty"`
	DryRun         bool   `json:"dry_run,omitempty"`
}

// ===== ADO Handlers =====

// StartADODiscoveryRequest is the request body for starting ADO discovery.
type StartADODiscoveryRequest struct {
	Organization string   `json:"organization"`
	Projects     []string `json:"projects,omitempty"`
	Workers      int      `json:"workers,omitempty"`
	SourceID     *int64   `json:"source_id,omitempty"` // Optional: associate discovered repos with a source
}

// ===== Repository Handlers =====

// RollbackRepositoryRequest is the request body for rolling back a repository.
type RollbackRepositoryRequest struct {
	Reason string `json:"reason,omitempty"`
}

// MarkWontMigrateRequest is the request body for marking a repository as won't migrate.
type MarkWontMigrateRequest struct {
	Unmark bool `json:"unmark,omitempty"`
}
