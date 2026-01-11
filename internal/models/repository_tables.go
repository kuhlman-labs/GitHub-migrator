package models

import (
	"encoding/json"
	"time"
)

// RepositoryGitProperties stores Git-related repository properties (1:1 with Repository)
type RepositoryGitProperties struct {
	RepositoryID       int64      `json:"repository_id" gorm:"primaryKey"`
	TotalSize          *int64     `json:"total_size,omitempty"`
	DefaultBranch      *string    `json:"default_branch,omitempty"`
	BranchCount        int        `json:"branch_count" gorm:"default:0"`
	CommitCount        int        `json:"commit_count" gorm:"default:0"`
	CommitsLast12Weeks int        `json:"commits_last_12_weeks" gorm:"column:commits_last_12_weeks;default:0"`
	HasLFS             bool       `json:"has_lfs" gorm:"default:false"`
	HasSubmodules      bool       `json:"has_submodules" gorm:"default:false"`
	HasLargeFiles      bool       `json:"has_large_files" gorm:"default:false"`
	LargeFileCount     int        `json:"large_file_count" gorm:"default:0"`
	LargestFile        *string    `json:"largest_file,omitempty"`
	LargestFileSize    *int64     `json:"largest_file_size,omitempty"`
	LargestCommit      *string    `json:"largest_commit,omitempty"`
	LargestCommitSize  *int64     `json:"largest_commit_size,omitempty"`
	LastCommitSHA      *string    `json:"last_commit_sha,omitempty"`
	LastCommitDate     *time.Time `json:"last_commit_date,omitempty"`
}

// TableName specifies the table name for RepositoryGitProperties
func (RepositoryGitProperties) TableName() string { return "repository_git_properties" }

// RepositoryFeatures stores GitHub feature flags (1:1 with Repository)
type RepositoryFeatures struct {
	RepositoryID         int64   `json:"repository_id" gorm:"primaryKey"`
	HasWiki              bool    `json:"has_wiki" gorm:"default:false"`
	HasPages             bool    `json:"has_pages" gorm:"default:false"`
	HasDiscussions       bool    `json:"has_discussions" gorm:"default:false"`
	HasActions           bool    `json:"has_actions" gorm:"default:false"`
	HasProjects          bool    `json:"has_projects" gorm:"default:false"`
	HasPackages          bool    `json:"has_packages" gorm:"default:false"`
	HasRulesets          bool    `json:"has_rulesets" gorm:"default:false"`
	BranchProtections    int     `json:"branch_protections" gorm:"default:0"`
	TagProtectionCount   int     `json:"tag_protection_count" gorm:"default:0"`
	EnvironmentCount     int     `json:"environment_count" gorm:"default:0"`
	SecretCount          int     `json:"secret_count" gorm:"default:0"`
	VariableCount        int     `json:"variable_count" gorm:"default:0"`
	WebhookCount         int     `json:"webhook_count" gorm:"default:0"`
	WorkflowCount        int     `json:"workflow_count" gorm:"default:0"`
	HasCodeScanning      bool    `json:"has_code_scanning" gorm:"default:false"`
	HasDependabot        bool    `json:"has_dependabot" gorm:"default:false"`
	HasSecretScanning    bool    `json:"has_secret_scanning" gorm:"default:false"`
	HasCodeowners        bool    `json:"has_codeowners" gorm:"default:false"`
	CodeownersContent    *string `json:"codeowners_content,omitempty" gorm:"type:text"`
	CodeownersTeams      *string `json:"codeowners_teams,omitempty" gorm:"type:text"`
	CodeownersUsers      *string `json:"codeowners_users,omitempty" gorm:"type:text"`
	HasSelfHostedRunners bool    `json:"has_self_hosted_runners" gorm:"default:false"`
	CollaboratorCount    int     `json:"collaborator_count" gorm:"default:0"`
	InstalledAppsCount   int     `json:"installed_apps_count" gorm:"default:0"`
	InstalledApps        *string `json:"installed_apps,omitempty" gorm:"type:text"`
	ReleaseCount         int     `json:"release_count" gorm:"default:0"`
	HasReleaseAssets     bool    `json:"has_release_assets" gorm:"default:false"`
	ContributorCount     int     `json:"contributor_count" gorm:"default:0"`
	TopContributors      *string `json:"top_contributors,omitempty" gorm:"type:text"`
	IssueCount           int     `json:"issue_count" gorm:"default:0"`
	PullRequestCount     int     `json:"pull_request_count" gorm:"default:0"`
	TagCount             int     `json:"tag_count" gorm:"default:0"`
	OpenIssueCount       int     `json:"open_issue_count" gorm:"default:0"`
	OpenPRCount          int     `json:"open_pr_count" gorm:"default:0"`
}

// TableName specifies the table name for RepositoryFeatures
func (RepositoryFeatures) TableName() string { return "repository_features" }

// RepositoryADOProperties stores Azure DevOps properties (1:1, only for ADO repos)
type RepositoryADOProperties struct {
	RepositoryID            int64   `json:"repository_id" gorm:"primaryKey"`
	Project                 *string `json:"project,omitempty"`
	IsGit                   bool    `json:"is_git" gorm:"default:true"`
	HasBoards               bool    `json:"has_boards" gorm:"default:false"`
	HasPipelines            bool    `json:"has_pipelines" gorm:"default:false"`
	HasGHAS                 bool    `json:"has_ghas" gorm:"default:false"`
	PipelineCount           int     `json:"pipeline_count" gorm:"default:0"`
	YAMLPipelineCount       int     `json:"yaml_pipeline_count" gorm:"default:0"`
	ClassicPipelineCount    int     `json:"classic_pipeline_count" gorm:"default:0"`
	PipelineRunCount        int     `json:"pipeline_run_count" gorm:"default:0"`
	HasServiceConnections   bool    `json:"has_service_connections" gorm:"default:false"`
	HasVariableGroups       bool    `json:"has_variable_groups" gorm:"default:false"`
	HasSelfHostedAgents     bool    `json:"has_self_hosted_agents" gorm:"default:false"`
	PullRequestCount        int     `json:"pull_request_count" gorm:"default:0"`
	OpenPRCount             int     `json:"open_pr_count" gorm:"default:0"`
	PRWithLinkedWorkItems   int     `json:"pr_with_linked_work_items" gorm:"default:0"`
	PRWithAttachments       int     `json:"pr_with_attachments" gorm:"default:0"`
	WorkItemCount           int     `json:"work_item_count" gorm:"default:0"`
	WorkItemLinkedCount     int     `json:"work_item_linked_count" gorm:"default:0"`
	ActiveWorkItemCount     int     `json:"active_work_item_count" gorm:"default:0"`
	WorkItemTypes           *string `json:"work_item_types,omitempty" gorm:"type:text"`
	BranchPolicyCount       int     `json:"branch_policy_count" gorm:"default:0"`
	BranchPolicyTypes       *string `json:"branch_policy_types,omitempty" gorm:"type:text"`
	RequiredReviewerCount   int     `json:"required_reviewer_count" gorm:"default:0"`
	BuildValidationPolicies int     `json:"build_validation_policies" gorm:"default:0"`
	HasWiki                 bool    `json:"has_wiki" gorm:"default:false"`
	WikiPageCount           int     `json:"wiki_page_count" gorm:"default:0"`
	TestPlanCount           int     `json:"test_plan_count" gorm:"default:0"`
	TestCaseCount           int     `json:"test_case_count" gorm:"default:0"`
	PackageFeedCount        int     `json:"package_feed_count" gorm:"default:0"`
	HasArtifacts            bool    `json:"has_artifacts" gorm:"default:false"`
	ServiceHookCount        int     `json:"service_hook_count" gorm:"default:0"`
	InstalledExtensions     *string `json:"installed_extensions,omitempty" gorm:"type:text"`
}

// TableName specifies the table name for RepositoryADOProperties
func (RepositoryADOProperties) TableName() string { return "repository_ado_properties" }

// RepositoryValidation stores migration validation and limit violations (1:1)
type RepositoryValidation struct {
	RepositoryID               int64   `json:"repository_id" gorm:"primaryKey"`
	ValidationStatus           *string `json:"validation_status,omitempty"`
	ValidationDetails          *string `json:"validation_details,omitempty" gorm:"type:text"`
	DestinationData            *string `json:"destination_data,omitempty" gorm:"type:text"`
	HasOversizedCommits        bool    `json:"has_oversized_commits" gorm:"default:false"`
	OversizedCommitDetails     *string `json:"oversized_commit_details,omitempty" gorm:"type:text"`
	HasLongRefs                bool    `json:"has_long_refs" gorm:"default:false"`
	LongRefDetails             *string `json:"long_ref_details,omitempty" gorm:"type:text"`
	HasBlockingFiles           bool    `json:"has_blocking_files" gorm:"default:false"`
	BlockingFileDetails        *string `json:"blocking_file_details,omitempty" gorm:"type:text"`
	HasLargeFileWarnings       bool    `json:"has_large_file_warnings" gorm:"default:false"`
	LargeFileWarningDetails    *string `json:"large_file_warning_details,omitempty" gorm:"type:text"`
	HasOversizedRepository     bool    `json:"has_oversized_repository" gorm:"default:false"`
	OversizedRepositoryDetails *string `json:"oversized_repository_details,omitempty" gorm:"type:text"`
	EstimatedMetadataSize      *int64  `json:"estimated_metadata_size,omitempty"`
	MetadataSizeDetails        *string `json:"metadata_size_details,omitempty" gorm:"type:text"`
	ComplexityScore            *int    `json:"complexity_score,omitempty"`
	ComplexityBreakdown        *string `json:"complexity_breakdown,omitempty" gorm:"type:text"`
}

// TableName specifies the table name for RepositoryValidation
func (RepositoryValidation) TableName() string { return "repository_validation" }

// MarshalJSON implements custom JSON marshaling to parse complexity_breakdown as object
func (v RepositoryValidation) MarshalJSON() ([]byte, error) {
	type Alias RepositoryValidation
	result := struct {
		Alias
		ComplexityBreakdown any `json:"complexity_breakdown,omitempty"`
	}{
		Alias: Alias(v),
	}
	// Parse complexity breakdown JSON string into object
	if v.ComplexityBreakdown != nil && *v.ComplexityBreakdown != "" {
		var breakdown map[string]any
		if err := json.Unmarshal([]byte(*v.ComplexityBreakdown), &breakdown); err == nil {
			result.ComplexityBreakdown = breakdown
		} else {
			result.ComplexityBreakdown = *v.ComplexityBreakdown
		}
	}
	return json.Marshal(result)
}
