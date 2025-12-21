package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

// RepositoryFilters encapsulates all filter parameters for listing repositories.
// This struct provides a type-safe way to handle the numerous filter options
// supported by the ListRepositories endpoint.
type RepositoryFilters struct {
	// Core filters
	Status       []string `json:"status,omitempty"`
	BatchID      *int64   `json:"batch_id,omitempty"`
	Source       string   `json:"source,omitempty"`
	Organization []string `json:"organization,omitempty"`
	Team         []string `json:"team,omitempty"`
	Search       string   `json:"search,omitempty"`
	Visibility   string   `json:"visibility,omitempty"`

	// Size filters
	MinSize      *int64   `json:"min_size,omitempty"`
	MaxSize      *int64   `json:"max_size,omitempty"`
	SizeCategory []string `json:"size_category,omitempty"`
	Complexity   []string `json:"complexity,omitempty"`

	// GitHub feature filters
	HasLFS               *bool `json:"has_lfs,omitempty"`
	HasSubmodules        *bool `json:"has_submodules,omitempty"`
	HasActions           *bool `json:"has_actions,omitempty"`
	HasWiki              *bool `json:"has_wiki,omitempty"`
	HasPages             *bool `json:"has_pages,omitempty"`
	HasDiscussions       *bool `json:"has_discussions,omitempty"`
	HasProjects          *bool `json:"has_projects,omitempty"`
	HasLargeFiles        *bool `json:"has_large_files,omitempty"`
	HasBranchProtections *bool `json:"has_branch_protections,omitempty"`
	IsArchived           *bool `json:"is_archived,omitempty"`
	IsFork               *bool `json:"is_fork,omitempty"`
	HasPackages          *bool `json:"has_packages,omitempty"`
	HasRulesets          *bool `json:"has_rulesets,omitempty"`
	HasCodeScanning      *bool `json:"has_code_scanning,omitempty"`
	HasDependabot        *bool `json:"has_dependabot,omitempty"`
	HasSecretScanning    *bool `json:"has_secret_scanning,omitempty"`
	HasCodeowners        *bool `json:"has_codeowners,omitempty"`
	HasSelfHostedRunners *bool `json:"has_self_hosted_runners,omitempty"`
	HasReleaseAssets     *bool `json:"has_release_assets,omitempty"`
	HasWebhooks          *bool `json:"has_webhooks,omitempty"`
	HasEnvironments      *bool `json:"has_environments,omitempty"`
	HasSecrets           *bool `json:"has_secrets,omitempty"`
	HasVariables         *bool `json:"has_variables,omitempty"`

	// Azure DevOps filters
	ADOOrganization         []string `json:"ado_organization,omitempty"`
	ADOProject              []string `json:"ado_project,omitempty"`
	ADOIsGit                *bool    `json:"ado_is_git,omitempty"`
	ADOHasBoards            *bool    `json:"ado_has_boards,omitempty"`
	ADOHasPipelines         *bool    `json:"ado_has_pipelines,omitempty"`
	ADOHasGHAS              *bool    `json:"ado_has_ghas,omitempty"`
	ADOHasWiki              *bool    `json:"ado_has_wiki,omitempty"`
	ADOPullRequestCount     string   `json:"ado_pull_request_count,omitempty"`
	ADOWorkItemCount        string   `json:"ado_work_item_count,omitempty"`
	ADOBranchPolicyCount    string   `json:"ado_branch_policy_count,omitempty"`
	ADOYAMLPipelineCount    string   `json:"ado_yaml_pipeline_count,omitempty"`
	ADOClassicPipelineCount string   `json:"ado_classic_pipeline_count,omitempty"`
	ADOTestPlanCount        string   `json:"ado_test_plan_count,omitempty"`
	ADOPackageFeedCount     string   `json:"ado_package_feed_count,omitempty"`
	ADOServiceHookCount     string   `json:"ado_service_hook_count,omitempty"`

	// Sorting and pagination
	SortBy            string `json:"sort_by,omitempty"`
	AvailableForBatch bool   `json:"available_for_batch,omitempty"`
	Limit             *int   `json:"limit,omitempty"`
	Offset            *int   `json:"offset,omitempty"`
}

// ParseRepositoryFilters parses repository filter parameters from an HTTP request.
// It returns a RepositoryFilters struct with all parsed values.
func ParseRepositoryFilters(r *http.Request) *RepositoryFilters {
	f := &RepositoryFilters{}
	q := r.URL.Query()

	// Parse status filter (supports comma-separated values)
	f.Status = parseCommaSeparatedList(q.Get("status"))

	// Parse batch_id
	f.BatchID = parseInt64Ptr(q.Get("batch_id"))

	// Parse source
	f.Source = q.Get("source")

	// Parse organization (supports comma-separated values)
	f.Organization = parseCommaSeparatedList(q.Get("organization"))

	// Parse team (supports comma-separated values)
	f.Team = parseCommaSeparatedList(q.Get("team"))

	// Parse search
	f.Search = q.Get("search")

	// Parse visibility
	f.Visibility = q.Get("visibility")

	// Parse size filters
	f.MinSize = parseInt64Ptr(q.Get("min_size"))
	f.MaxSize = parseInt64Ptr(q.Get("max_size"))
	f.SizeCategory = parseCommaSeparatedList(q.Get("size_category"))
	f.Complexity = parseCommaSeparatedList(q.Get("complexity"))

	// Parse GitHub feature filters
	f.HasLFS = parseBoolPtr(q.Get("has_lfs"))
	f.HasSubmodules = parseBoolPtr(q.Get("has_submodules"))
	f.HasActions = parseBoolPtr(q.Get("has_actions"))
	f.HasWiki = parseBoolPtr(q.Get("has_wiki"))
	f.HasPages = parseBoolPtr(q.Get("has_pages"))
	f.HasDiscussions = parseBoolPtr(q.Get("has_discussions"))
	f.HasProjects = parseBoolPtr(q.Get("has_projects"))
	f.HasLargeFiles = parseBoolPtr(q.Get("has_large_files"))
	f.HasBranchProtections = parseBoolPtr(q.Get("has_branch_protections"))
	f.IsArchived = parseBoolPtr(q.Get("is_archived"))
	f.IsFork = parseBoolPtr(q.Get("is_fork"))
	f.HasPackages = parseBoolPtr(q.Get("has_packages"))
	f.HasRulesets = parseBoolPtr(q.Get("has_rulesets"))
	f.HasCodeScanning = parseBoolPtr(q.Get("has_code_scanning"))
	f.HasDependabot = parseBoolPtr(q.Get("has_dependabot"))
	f.HasSecretScanning = parseBoolPtr(q.Get("has_secret_scanning"))
	f.HasCodeowners = parseBoolPtr(q.Get("has_codeowners"))
	f.HasSelfHostedRunners = parseBoolPtr(q.Get("has_self_hosted_runners"))
	f.HasReleaseAssets = parseBoolPtr(q.Get("has_release_assets"))
	f.HasWebhooks = parseBoolPtr(q.Get("has_webhooks"))
	f.HasEnvironments = parseBoolPtr(q.Get("has_environments"))
	f.HasSecrets = parseBoolPtr(q.Get("has_secrets"))
	f.HasVariables = parseBoolPtr(q.Get("has_variables"))

	// Parse Azure DevOps filters
	f.ADOOrganization = parseCommaSeparatedList(q.Get("ado_organization"))
	f.ADOProject = parseCommaSeparatedList(q.Get("project"))
	f.ADOIsGit = parseBoolPtr(q.Get("ado_is_git"))
	f.ADOHasBoards = parseBoolPtr(q.Get("ado_has_boards"))
	f.ADOHasPipelines = parseBoolPtr(q.Get("ado_has_pipelines"))
	f.ADOHasGHAS = parseBoolPtr(q.Get("ado_has_ghas"))
	f.ADOHasWiki = parseBoolPtr(q.Get("ado_has_wiki"))
	f.ADOPullRequestCount = q.Get("ado_pull_request_count")
	f.ADOWorkItemCount = q.Get("ado_work_item_count")
	f.ADOBranchPolicyCount = q.Get("ado_branch_policy_count")
	f.ADOYAMLPipelineCount = q.Get("ado_yaml_pipeline_count")
	f.ADOClassicPipelineCount = q.Get("ado_classic_pipeline_count")
	f.ADOTestPlanCount = q.Get("ado_test_plan_count")
	f.ADOPackageFeedCount = q.Get("ado_package_feed_count")
	f.ADOServiceHookCount = q.Get("ado_service_hook_count")

	// Parse sorting and pagination
	f.SortBy = q.Get("sort_by")
	f.AvailableForBatch = q.Get("available_for_batch") == boolTrue
	f.Limit = parseIntPtr(q.Get("limit"))
	f.Offset = parseIntPtr(q.Get("offset"))

	return f
}

// ToMap converts the RepositoryFilters struct to a map[string]interface{}
// for compatibility with the existing database layer.
func (f *RepositoryFilters) ToMap() map[string]interface{} {
	m := make(map[string]interface{})

	f.addCoreFiltersToMap(m)
	f.addSizeFiltersToMap(m)
	f.addGitHubFeatureFiltersToMap(m)
	f.addADOFiltersToMap(m)
	f.addPaginationToMap(m)

	return m
}

// addCoreFiltersToMap adds core filters (status, source, organization, etc.) to the map.
func (f *RepositoryFilters) addCoreFiltersToMap(m map[string]interface{}) {
	addSliceOrSingleFilter(m, "status", f.Status)

	if f.BatchID != nil {
		m["batch_id"] = *f.BatchID
	}

	if f.Source != "" {
		m["source"] = f.Source
	}

	addSliceOrSingleFilter(m, "organization", f.Organization)
	addSliceOrSingleFilter(m, "team", f.Team)

	if f.Search != "" {
		m["search"] = f.Search
	}

	if f.Visibility != "" {
		m["visibility"] = f.Visibility
	}
}

// addSizeFiltersToMap adds size-related filters to the map.
func (f *RepositoryFilters) addSizeFiltersToMap(m map[string]interface{}) {
	if f.MinSize != nil {
		m["min_size"] = *f.MinSize
	}
	if f.MaxSize != nil {
		m["max_size"] = *f.MaxSize
	}
	addSliceOrSingleFilter(m, "size_category", f.SizeCategory)
	addSliceOrSingleFilter(m, "complexity", f.Complexity)
}

// addGitHubFeatureFiltersToMap adds GitHub feature filters to the map.
func (f *RepositoryFilters) addGitHubFeatureFiltersToMap(m map[string]interface{}) {
	addBoolFilter(m, "has_lfs", f.HasLFS)
	addBoolFilter(m, "has_submodules", f.HasSubmodules)
	addBoolFilter(m, "has_actions", f.HasActions)
	addBoolFilter(m, "has_wiki", f.HasWiki)
	addBoolFilter(m, "has_pages", f.HasPages)
	addBoolFilter(m, "has_discussions", f.HasDiscussions)
	addBoolFilter(m, "has_projects", f.HasProjects)
	addBoolFilter(m, "has_large_files", f.HasLargeFiles)
	addBoolFilter(m, "has_branch_protections", f.HasBranchProtections)
	addBoolFilter(m, "is_archived", f.IsArchived)
	addBoolFilter(m, "is_fork", f.IsFork)
	addBoolFilter(m, "has_packages", f.HasPackages)
	addBoolFilter(m, "has_rulesets", f.HasRulesets)
	addBoolFilter(m, "has_code_scanning", f.HasCodeScanning)
	addBoolFilter(m, "has_dependabot", f.HasDependabot)
	addBoolFilter(m, "has_secret_scanning", f.HasSecretScanning)
	addBoolFilter(m, "has_codeowners", f.HasCodeowners)
	addBoolFilter(m, "has_self_hosted_runners", f.HasSelfHostedRunners)
	addBoolFilter(m, "has_release_assets", f.HasReleaseAssets)
	addBoolFilter(m, "has_webhooks", f.HasWebhooks)
	addBoolFilter(m, "has_environments", f.HasEnvironments)
	addBoolFilter(m, "has_secrets", f.HasSecrets)
	addBoolFilter(m, "has_variables", f.HasVariables)
}

// addADOFiltersToMap adds Azure DevOps filters to the map.
func (f *RepositoryFilters) addADOFiltersToMap(m map[string]interface{}) {
	addSliceOrSingleFilter(m, "ado_organization", f.ADOOrganization)
	addSliceOrSingleFilter(m, "ado_project", f.ADOProject)

	addBoolFilter(m, "ado_is_git", f.ADOIsGit)
	addBoolFilter(m, "ado_has_boards", f.ADOHasBoards)
	addBoolFilter(m, "ado_has_pipelines", f.ADOHasPipelines)
	addBoolFilter(m, "ado_has_ghas", f.ADOHasGHAS)
	addBoolFilter(m, "ado_has_wiki", f.ADOHasWiki)

	addStringFilter(m, "ado_pull_request_count", f.ADOPullRequestCount)
	addStringFilter(m, "ado_work_item_count", f.ADOWorkItemCount)
	addStringFilter(m, "ado_branch_policy_count", f.ADOBranchPolicyCount)
	addStringFilter(m, "ado_yaml_pipeline_count", f.ADOYAMLPipelineCount)
	addStringFilter(m, "ado_classic_pipeline_count", f.ADOClassicPipelineCount)
	addStringFilter(m, "ado_test_plan_count", f.ADOTestPlanCount)
	addStringFilter(m, "ado_package_feed_count", f.ADOPackageFeedCount)
	addStringFilter(m, "ado_service_hook_count", f.ADOServiceHookCount)
}

// addPaginationToMap adds sorting and pagination filters to the map.
func (f *RepositoryFilters) addPaginationToMap(m map[string]interface{}) {
	if f.SortBy != "" {
		m["sort_by"] = f.SortBy
	}
	if f.AvailableForBatch {
		m["available_for_batch"] = true
	}
	if f.Limit != nil && *f.Limit > 0 {
		m["limit"] = *f.Limit
	}
	if f.Offset != nil && *f.Offset >= 0 {
		m["offset"] = *f.Offset
	}
}

// addSliceOrSingleFilter adds a slice value as single value or slice depending on length.
func addSliceOrSingleFilter(m map[string]interface{}, key string, values []string) {
	if len(values) == 1 {
		m[key] = values[0]
	} else if len(values) > 1 {
		m[key] = values
	}
}

// addStringFilter adds a non-empty string to the map.
func addStringFilter(m map[string]interface{}, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// HasPagination returns true if pagination parameters are set
func (f *RepositoryFilters) HasPagination() bool {
	return f.Limit != nil && *f.Limit > 0
}

// Helper functions for parsing

// parseCommaSeparatedList parses a comma-separated string into a slice.
// Returns nil if the input is empty.
func parseCommaSeparatedList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// parseBoolPtr parses a string to a *bool.
// Returns nil if the input is empty.
func parseBoolPtr(s string) *bool {
	if s == "" {
		return nil
	}
	val := s == boolTrue
	return &val
}

// parseInt64Ptr parses a string to a *int64.
// Returns nil if the input is empty or invalid.
func parseInt64Ptr(s string) *int64 {
	if s == "" {
		return nil
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &val
}

// parseIntPtr parses a string to a *int.
// Returns nil if the input is empty or invalid.
func parseIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &val
}

// addBoolFilter adds a boolean filter to the map if the value is not nil.
func addBoolFilter(m map[string]interface{}, key string, val *bool) {
	if val != nil {
		m[key] = *val
	}
}
