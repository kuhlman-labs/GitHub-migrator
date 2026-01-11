package models

import "time"

// This file provides backward compatibility methods for code that needs to
// directly set properties on Repository that are now in related tables.
// These methods ensure the related table struct is initialized before setting values.

// EnsureGitProperties ensures GitProperties is initialized
func (r *Repository) EnsureGitProperties() *RepositoryGitProperties {
	if r.GitProperties == nil {
		r.GitProperties = &RepositoryGitProperties{}
	}
	return r.GitProperties
}

// EnsureFeatures ensures Features is initialized
func (r *Repository) EnsureFeatures() *RepositoryFeatures {
	if r.Features == nil {
		r.Features = &RepositoryFeatures{}
	}
	return r.Features
}

// EnsureADOProperties ensures ADOProperties is initialized
func (r *Repository) EnsureADOProperties() *RepositoryADOProperties {
	if r.ADOProperties == nil {
		r.ADOProperties = &RepositoryADOProperties{}
	}
	return r.ADOProperties
}

// EnsureValidation ensures Validation is initialized
func (r *Repository) EnsureValidation() *RepositoryValidation {
	if r.Validation == nil {
		r.Validation = &RepositoryValidation{}
	}
	return r.Validation
}

// Setter methods for git properties

// SetTotalSize sets the total size in git properties
func (r *Repository) SetTotalSize(size *int64) {
	r.EnsureGitProperties().TotalSize = size
}

// SetDefaultBranch sets the default branch in git properties
func (r *Repository) SetDefaultBranch(branch *string) {
	r.EnsureGitProperties().DefaultBranch = branch
}

// SetBranchCount sets the branch count in git properties
func (r *Repository) SetBranchCount(count int) {
	r.EnsureGitProperties().BranchCount = count
}

// SetCommitCount sets the commit count in git properties
func (r *Repository) SetCommitCount(count int) {
	r.EnsureGitProperties().CommitCount = count
}

// SetCommitsLast12Weeks sets the commits last 12 weeks in git properties
func (r *Repository) SetCommitsLast12Weeks(count int) {
	r.EnsureGitProperties().CommitsLast12Weeks = count
}

// SetHasLFS sets the has_lfs flag in git properties
func (r *Repository) SetHasLFS(value bool) {
	r.EnsureGitProperties().HasLFS = value
}

// SetHasSubmodules sets the has_submodules flag in git properties
func (r *Repository) SetHasSubmodules(value bool) {
	r.EnsureGitProperties().HasSubmodules = value
}

// SetHasLargeFiles sets the has_large_files flag in git properties
func (r *Repository) SetHasLargeFiles(value bool) {
	r.EnsureGitProperties().HasLargeFiles = value
}

// SetLargeFileCount sets the large file count in git properties
func (r *Repository) SetLargeFileCount(count int) {
	r.EnsureGitProperties().LargeFileCount = count
}

// SetLargestFile sets the largest file in git properties
func (r *Repository) SetLargestFile(file *string) {
	r.EnsureGitProperties().LargestFile = file
}

// SetLargestFileSize sets the largest file size in git properties
func (r *Repository) SetLargestFileSize(size *int64) {
	r.EnsureGitProperties().LargestFileSize = size
}

// SetLargestCommit sets the largest commit in git properties
func (r *Repository) SetLargestCommit(commit *string) {
	r.EnsureGitProperties().LargestCommit = commit
}

// SetLargestCommitSize sets the largest commit size in git properties
func (r *Repository) SetLargestCommitSize(size *int64) {
	r.EnsureGitProperties().LargestCommitSize = size
}

// SetLastCommitSHA sets the last commit SHA in git properties
func (r *Repository) SetLastCommitSHA(sha *string) {
	r.EnsureGitProperties().LastCommitSHA = sha
}

// SetLastCommitDate sets the last commit date in git properties
func (r *Repository) SetLastCommitDate(date *time.Time) {
	r.EnsureGitProperties().LastCommitDate = date
}

// Setter methods for features

// SetHasWiki sets the has_wiki flag in features
func (r *Repository) SetHasWiki(value bool) {
	r.EnsureFeatures().HasWiki = value
}

// SetHasPages sets the has_pages flag in features
func (r *Repository) SetHasPages(value bool) {
	r.EnsureFeatures().HasPages = value
}

// SetHasDiscussions sets the has_discussions flag in features
func (r *Repository) SetHasDiscussions(value bool) {
	r.EnsureFeatures().HasDiscussions = value
}

// SetHasActions sets the has_actions flag in features
func (r *Repository) SetHasActions(value bool) {
	r.EnsureFeatures().HasActions = value
}

// SetHasProjects sets the has_projects flag in features
func (r *Repository) SetHasProjects(value bool) {
	r.EnsureFeatures().HasProjects = value
}

// SetHasPackages sets the has_packages flag in features
func (r *Repository) SetHasPackages(value bool) {
	r.EnsureFeatures().HasPackages = value
}

// SetHasRulesets sets the has_rulesets flag in features
func (r *Repository) SetHasRulesets(value bool) {
	r.EnsureFeatures().HasRulesets = value
}

// SetBranchProtections sets the branch protections count in features
func (r *Repository) SetBranchProtections(count int) {
	r.EnsureFeatures().BranchProtections = count
}

// SetTagProtectionCount sets the tag protection count in features
func (r *Repository) SetTagProtectionCount(count int) {
	r.EnsureFeatures().TagProtectionCount = count
}

// SetEnvironmentCount sets the environment count in features
func (r *Repository) SetEnvironmentCount(count int) {
	r.EnsureFeatures().EnvironmentCount = count
}

// SetSecretCount sets the secret count in features
func (r *Repository) SetSecretCount(count int) {
	r.EnsureFeatures().SecretCount = count
}

// SetVariableCount sets the variable count in features
func (r *Repository) SetVariableCount(count int) {
	r.EnsureFeatures().VariableCount = count
}

// SetWebhookCount sets the webhook count in features
func (r *Repository) SetWebhookCount(count int) {
	r.EnsureFeatures().WebhookCount = count
}

// SetWorkflowCount sets the workflow count in features
func (r *Repository) SetWorkflowCount(count int) {
	r.EnsureFeatures().WorkflowCount = count
}

// SetHasCodeScanning sets the has_code_scanning flag in features
func (r *Repository) SetHasCodeScanning(value bool) {
	r.EnsureFeatures().HasCodeScanning = value
}

// SetHasDependabot sets the has_dependabot flag in features
func (r *Repository) SetHasDependabot(value bool) {
	r.EnsureFeatures().HasDependabot = value
}

// SetHasSecretScanning sets the has_secret_scanning flag in features
func (r *Repository) SetHasSecretScanning(value bool) {
	r.EnsureFeatures().HasSecretScanning = value
}

// SetHasCodeowners sets the has_codeowners flag in features
func (r *Repository) SetHasCodeowners(value bool) {
	r.EnsureFeatures().HasCodeowners = value
}

// SetCodeownersContent sets the codeowners content in features
func (r *Repository) SetCodeownersContent(content *string) {
	r.EnsureFeatures().CodeownersContent = content
}

// SetCodeownersTeams sets the codeowners teams in features
func (r *Repository) SetCodeownersTeams(teams *string) {
	r.EnsureFeatures().CodeownersTeams = teams
}

// SetCodeownersUsers sets the codeowners users in features
func (r *Repository) SetCodeownersUsers(users *string) {
	r.EnsureFeatures().CodeownersUsers = users
}

// SetHasSelfHostedRunners sets the has_self_hosted_runners flag in features
func (r *Repository) SetHasSelfHostedRunners(value bool) {
	r.EnsureFeatures().HasSelfHostedRunners = value
}

// SetCollaboratorCount sets the collaborator count in features
func (r *Repository) SetCollaboratorCount(count int) {
	r.EnsureFeatures().CollaboratorCount = count
}

// SetInstalledAppsCount sets the installed apps count in features
func (r *Repository) SetInstalledAppsCount(count int) {
	r.EnsureFeatures().InstalledAppsCount = count
}

// SetInstalledApps sets the installed apps in features
func (r *Repository) SetInstalledApps(apps *string) {
	r.EnsureFeatures().InstalledApps = apps
}

// SetReleaseCount sets the release count in features
func (r *Repository) SetReleaseCount(count int) {
	r.EnsureFeatures().ReleaseCount = count
}

// SetHasReleaseAssets sets the has_release_assets flag in features
func (r *Repository) SetHasReleaseAssets(value bool) {
	r.EnsureFeatures().HasReleaseAssets = value
}

// SetContributorCount sets the contributor count in features
func (r *Repository) SetContributorCount(count int) {
	r.EnsureFeatures().ContributorCount = count
}

// SetTopContributors sets the top contributors in features
func (r *Repository) SetTopContributors(contributors *string) {
	r.EnsureFeatures().TopContributors = contributors
}

// SetIssueCount sets the issue count in features
func (r *Repository) SetIssueCount(count int) {
	r.EnsureFeatures().IssueCount = count
}

// SetPullRequestCount sets the pull request count in features
func (r *Repository) SetPullRequestCount(count int) {
	r.EnsureFeatures().PullRequestCount = count
}

// SetTagCount sets the tag count in features
func (r *Repository) SetTagCount(count int) {
	r.EnsureFeatures().TagCount = count
}

// SetOpenIssueCount sets the open issue count in features
func (r *Repository) SetOpenIssueCount(count int) {
	r.EnsureFeatures().OpenIssueCount = count
}

// SetOpenPRCount sets the open PR count in features
func (r *Repository) SetOpenPRCount(count int) {
	r.EnsureFeatures().OpenPRCount = count
}

// Setter methods for ADO properties

// SetADOProject sets the project in ADO properties
func (r *Repository) SetADOProject(project *string) {
	r.EnsureADOProperties().Project = project
}

// SetADOIsGit sets the is_git flag in ADO properties
func (r *Repository) SetADOIsGit(value bool) {
	r.EnsureADOProperties().IsGit = value
}

// SetADOHasBoards sets the has_boards flag in ADO properties
func (r *Repository) SetADOHasBoards(value bool) {
	r.EnsureADOProperties().HasBoards = value
}

// SetADOHasPipelines sets the has_pipelines flag in ADO properties
func (r *Repository) SetADOHasPipelines(value bool) {
	r.EnsureADOProperties().HasPipelines = value
}

// SetADOHasGHAS sets the has_ghas flag in ADO properties
func (r *Repository) SetADOHasGHAS(value bool) {
	r.EnsureADOProperties().HasGHAS = value
}

// SetADOPipelineCount sets the pipeline count in ADO properties
func (r *Repository) SetADOPipelineCount(count int) {
	r.EnsureADOProperties().PipelineCount = count
}

// SetADOYAMLPipelineCount sets the YAML pipeline count in ADO properties
func (r *Repository) SetADOYAMLPipelineCount(count int) {
	r.EnsureADOProperties().YAMLPipelineCount = count
}

// SetADOClassicPipelineCount sets the classic pipeline count in ADO properties
func (r *Repository) SetADOClassicPipelineCount(count int) {
	r.EnsureADOProperties().ClassicPipelineCount = count
}

// SetADOPipelineRunCount sets the pipeline run count in ADO properties
func (r *Repository) SetADOPipelineRunCount(count int) {
	r.EnsureADOProperties().PipelineRunCount = count
}

// SetADOHasServiceConnections sets the has_service_connections flag in ADO properties
func (r *Repository) SetADOHasServiceConnections(value bool) {
	r.EnsureADOProperties().HasServiceConnections = value
}

// SetADOHasVariableGroups sets the has_variable_groups flag in ADO properties
func (r *Repository) SetADOHasVariableGroups(value bool) {
	r.EnsureADOProperties().HasVariableGroups = value
}

// SetADOHasSelfHostedAgents sets the has_self_hosted_agents flag in ADO properties
func (r *Repository) SetADOHasSelfHostedAgents(value bool) {
	r.EnsureADOProperties().HasSelfHostedAgents = value
}

// SetADOPullRequestCount sets the pull request count in ADO properties
func (r *Repository) SetADOPullRequestCount(count int) {
	r.EnsureADOProperties().PullRequestCount = count
}

// SetADOOpenPRCount sets the open PR count in ADO properties
func (r *Repository) SetADOOpenPRCount(count int) {
	r.EnsureADOProperties().OpenPRCount = count
}

// SetADOPRWithLinkedWorkItems sets the PR with linked work items count in ADO properties
func (r *Repository) SetADOPRWithLinkedWorkItems(count int) {
	r.EnsureADOProperties().PRWithLinkedWorkItems = count
}

// SetADOPRWithAttachments sets the PR with attachments count in ADO properties
func (r *Repository) SetADOPRWithAttachments(count int) {
	r.EnsureADOProperties().PRWithAttachments = count
}

// SetADOWorkItemCount sets the work item count in ADO properties
func (r *Repository) SetADOWorkItemCount(count int) {
	r.EnsureADOProperties().WorkItemCount = count
}

// SetADOWorkItemLinkedCount sets the work item linked count in ADO properties
func (r *Repository) SetADOWorkItemLinkedCount(count int) {
	r.EnsureADOProperties().WorkItemLinkedCount = count
}

// SetADOActiveWorkItemCount sets the active work item count in ADO properties
func (r *Repository) SetADOActiveWorkItemCount(count int) {
	r.EnsureADOProperties().ActiveWorkItemCount = count
}

// SetADOWorkItemTypes sets the work item types in ADO properties
func (r *Repository) SetADOWorkItemTypes(types *string) {
	r.EnsureADOProperties().WorkItemTypes = types
}

// SetADOBranchPolicyCount sets the branch policy count in ADO properties
func (r *Repository) SetADOBranchPolicyCount(count int) {
	r.EnsureADOProperties().BranchPolicyCount = count
}

// SetADOBranchPolicyTypes sets the branch policy types in ADO properties
func (r *Repository) SetADOBranchPolicyTypes(types *string) {
	r.EnsureADOProperties().BranchPolicyTypes = types
}

// SetADORequiredReviewerCount sets the required reviewer count in ADO properties
func (r *Repository) SetADORequiredReviewerCount(count int) {
	r.EnsureADOProperties().RequiredReviewerCount = count
}

// SetADOBuildValidationPolicies sets the build validation policies in ADO properties
func (r *Repository) SetADOBuildValidationPolicies(count int) {
	r.EnsureADOProperties().BuildValidationPolicies = count
}

// SetADOHasWiki sets the has_wiki flag in ADO properties
func (r *Repository) SetADOHasWiki(value bool) {
	r.EnsureADOProperties().HasWiki = value
}

// SetADOWikiPageCount sets the wiki page count in ADO properties
func (r *Repository) SetADOWikiPageCount(count int) {
	r.EnsureADOProperties().WikiPageCount = count
}

// SetADOTestPlanCount sets the test plan count in ADO properties
func (r *Repository) SetADOTestPlanCount(count int) {
	r.EnsureADOProperties().TestPlanCount = count
}

// SetADOTestCaseCount sets the test case count in ADO properties
func (r *Repository) SetADOTestCaseCount(count int) {
	r.EnsureADOProperties().TestCaseCount = count
}

// SetADOPackageFeedCount sets the package feed count in ADO properties
func (r *Repository) SetADOPackageFeedCount(count int) {
	r.EnsureADOProperties().PackageFeedCount = count
}

// SetADOHasArtifacts sets the has_artifacts flag in ADO properties
func (r *Repository) SetADOHasArtifacts(value bool) {
	r.EnsureADOProperties().HasArtifacts = value
}

// SetADOServiceHookCount sets the service hook count in ADO properties
func (r *Repository) SetADOServiceHookCount(count int) {
	r.EnsureADOProperties().ServiceHookCount = count
}

// SetADOInstalledExtensions sets the installed extensions in ADO properties
func (r *Repository) SetADOInstalledExtensions(extensions *string) {
	r.EnsureADOProperties().InstalledExtensions = extensions
}

// Setter methods for validation

// SetValidationStatus sets the validation status
func (r *Repository) SetValidationStatus(status *string) {
	r.EnsureValidation().ValidationStatus = status
}

// SetValidationDetails sets the validation details
func (r *Repository) SetValidationDetails(details *string) {
	r.EnsureValidation().ValidationDetails = details
}

// SetDestinationData sets the destination data
func (r *Repository) SetDestinationData(data *string) {
	r.EnsureValidation().DestinationData = data
}

// SetHasOversizedCommits sets the has_oversized_commits flag in validation
func (r *Repository) SetHasOversizedCommits(value bool) {
	r.EnsureValidation().HasOversizedCommits = value
}

// SetOversizedCommitDetails sets the oversized commit details in validation
func (r *Repository) SetOversizedCommitDetails(details *string) {
	r.EnsureValidation().OversizedCommitDetails = details
}

// SetHasLongRefs sets the has_long_refs flag in validation
func (r *Repository) SetHasLongRefs(value bool) {
	r.EnsureValidation().HasLongRefs = value
}

// SetLongRefDetails sets the long ref details in validation
func (r *Repository) SetLongRefDetails(details *string) {
	r.EnsureValidation().LongRefDetails = details
}

// SetHasBlockingFiles sets the has_blocking_files flag in validation
func (r *Repository) SetHasBlockingFiles(value bool) {
	r.EnsureValidation().HasBlockingFiles = value
}

// SetBlockingFileDetails sets the blocking file details in validation
func (r *Repository) SetBlockingFileDetails(details *string) {
	r.EnsureValidation().BlockingFileDetails = details
}

// SetHasLargeFileWarnings sets the has_large_file_warnings flag in validation
func (r *Repository) SetHasLargeFileWarnings(value bool) {
	r.EnsureValidation().HasLargeFileWarnings = value
}

// SetLargeFileWarningDetails sets the large file warning details in validation
func (r *Repository) SetLargeFileWarningDetails(details *string) {
	r.EnsureValidation().LargeFileWarningDetails = details
}

// SetHasOversizedRepository sets the has_oversized_repository flag in validation
func (r *Repository) SetHasOversizedRepository(value bool) {
	r.EnsureValidation().HasOversizedRepository = value
}

// SetOversizedRepositoryDetails sets the oversized repository details in validation
func (r *Repository) SetOversizedRepositoryDetails(details *string) {
	r.EnsureValidation().OversizedRepositoryDetails = details
}

// SetEstimatedMetadataSize sets the estimated metadata size in validation
func (r *Repository) SetEstimatedMetadataSize(size *int64) {
	r.EnsureValidation().EstimatedMetadataSize = size
}

// SetMetadataSizeDetails sets the metadata size details in validation
func (r *Repository) SetMetadataSizeDetails(details *string) {
	r.EnsureValidation().MetadataSizeDetails = details
}

// SetComplexityScore sets the complexity score in validation
func (r *Repository) SetComplexityScore(score *int) {
	r.EnsureValidation().ComplexityScore = score
}

// Additional getter methods for less common fields

// GetLargestFile returns the largest file from git properties
func (r *Repository) GetLargestFile() *string {
	if r.GitProperties != nil {
		return r.GitProperties.LargestFile
	}
	return nil
}

// GetLargestFileSize returns the largest file size from git properties
func (r *Repository) GetLargestFileSize() *int64 {
	if r.GitProperties != nil {
		return r.GitProperties.LargestFileSize
	}
	return nil
}

// GetLargestCommit returns the largest commit from git properties
func (r *Repository) GetLargestCommit() *string {
	if r.GitProperties != nil {
		return r.GitProperties.LargestCommit
	}
	return nil
}

// GetLargestCommitSize returns the largest commit size from git properties
func (r *Repository) GetLargestCommitSize() *int64 {
	if r.GitProperties != nil {
		return r.GitProperties.LargestCommitSize
	}
	return nil
}

// GetLargeFileCount returns the large file count from git properties
func (r *Repository) GetLargeFileCount() int {
	if r.GitProperties != nil {
		return r.GitProperties.LargeFileCount
	}
	return 0
}

// GetCommitsLast12Weeks returns the commits in last 12 weeks from git properties
func (r *Repository) GetCommitsLast12Weeks() int {
	if r.GitProperties != nil {
		return r.GitProperties.CommitsLast12Weeks
	}
	return 0
}

// GetLastCommitSHA returns the last commit SHA from git properties
func (r *Repository) GetLastCommitSHA() *string {
	if r.GitProperties != nil {
		return r.GitProperties.LastCommitSHA
	}
	return nil
}

// GetLastCommitDate returns the last commit date from git properties
func (r *Repository) GetLastCommitDate() *time.Time {
	if r.GitProperties != nil {
		return r.GitProperties.LastCommitDate
	}
	return nil
}

// GetTagProtectionCount returns the tag protection count from features
func (r *Repository) GetTagProtectionCount() int {
	if r.Features != nil {
		return r.Features.TagProtectionCount
	}
	return 0
}

// GetEnvironmentCount returns the environment count from features
func (r *Repository) GetEnvironmentCount() int {
	if r.Features != nil {
		return r.Features.EnvironmentCount
	}
	return 0
}

// GetSecretCount returns the secret count from features
func (r *Repository) GetSecretCount() int {
	if r.Features != nil {
		return r.Features.SecretCount
	}
	return 0
}

// GetVariableCount returns the variable count from features
func (r *Repository) GetVariableCount() int {
	if r.Features != nil {
		return r.Features.VariableCount
	}
	return 0
}

// GetWebhookCount returns the webhook count from features
func (r *Repository) GetWebhookCount() int {
	if r.Features != nil {
		return r.Features.WebhookCount
	}
	return 0
}

// GetWorkflowCount returns the workflow count from features
func (r *Repository) GetWorkflowCount() int {
	if r.Features != nil {
		return r.Features.WorkflowCount
	}
	return 0
}

// GetCodeownersContent returns the codeowners content from features
func (r *Repository) GetCodeownersContent() *string {
	if r.Features != nil {
		return r.Features.CodeownersContent
	}
	return nil
}

// GetCodeownersTeams returns the codeowners teams from features
func (r *Repository) GetCodeownersTeams() *string {
	if r.Features != nil {
		return r.Features.CodeownersTeams
	}
	return nil
}

// GetCodeownersUsers returns the codeowners users from features
func (r *Repository) GetCodeownersUsers() *string {
	if r.Features != nil {
		return r.Features.CodeownersUsers
	}
	return nil
}

// GetCollaboratorCount returns the collaborator count from features
func (r *Repository) GetCollaboratorCount() int {
	if r.Features != nil {
		return r.Features.CollaboratorCount
	}
	return 0
}

// GetInstalledAppsCount returns the installed apps count from features
func (r *Repository) GetInstalledAppsCount() int {
	if r.Features != nil {
		return r.Features.InstalledAppsCount
	}
	return 0
}

// GetInstalledApps returns the installed apps from features
func (r *Repository) GetInstalledApps() *string {
	if r.Features != nil {
		return r.Features.InstalledApps
	}
	return nil
}

// GetReleaseCount returns the release count from features
func (r *Repository) GetReleaseCount() int {
	if r.Features != nil {
		return r.Features.ReleaseCount
	}
	return 0
}

// HasReleaseAssets returns true if the repository has release assets
func (r *Repository) HasReleaseAssets() bool {
	return r.Features != nil && r.Features.HasReleaseAssets
}

// GetContributorCount returns the contributor count from features
func (r *Repository) GetContributorCount() int {
	if r.Features != nil {
		return r.Features.ContributorCount
	}
	return 0
}

// GetTopContributors returns the top contributors from features
func (r *Repository) GetTopContributors() *string {
	if r.Features != nil {
		return r.Features.TopContributors
	}
	return nil
}

// GetIssueCount returns the issue count from features
func (r *Repository) GetIssueCount() int {
	if r.Features != nil {
		return r.Features.IssueCount
	}
	return 0
}

// GetPullRequestCount returns the pull request count from features
func (r *Repository) GetPullRequestCount() int {
	if r.Features != nil {
		return r.Features.PullRequestCount
	}
	return 0
}

// GetTagCount returns the tag count from features
func (r *Repository) GetTagCount() int {
	if r.Features != nil {
		return r.Features.TagCount
	}
	return 0
}

// GetOpenIssueCount returns the open issue count from features
func (r *Repository) GetOpenIssueCount() int {
	if r.Features != nil {
		return r.Features.OpenIssueCount
	}
	return 0
}

// GetOpenPRCount returns the open PR count from features
func (r *Repository) GetOpenPRCount() int {
	if r.Features != nil {
		return r.Features.OpenPRCount
	}
	return 0
}

// ADO property getters

// GetADOPullRequestCount returns the pull request count from ADO properties
func (r *Repository) GetADOPullRequestCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.PullRequestCount
	}
	return 0
}

// GetADOWorkItemCount returns the work item count from ADO properties
func (r *Repository) GetADOWorkItemCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.WorkItemCount
	}
	return 0
}

// GetADOBranchPolicyCount returns the branch policy count from ADO properties
func (r *Repository) GetADOBranchPolicyCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.BranchPolicyCount
	}
	return 0
}

// GetADOPipelineCount returns the pipeline count from ADO properties
func (r *Repository) GetADOPipelineCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.PipelineCount
	}
	return 0
}

// GetADOYAMLPipelineCount returns the YAML pipeline count from ADO properties
func (r *Repository) GetADOYAMLPipelineCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.YAMLPipelineCount
	}
	return 0
}

// GetADOClassicPipelineCount returns the classic pipeline count from ADO properties
func (r *Repository) GetADOClassicPipelineCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.ClassicPipelineCount
	}
	return 0
}

// GetADOWikiPageCount returns the wiki page count from ADO properties
func (r *Repository) GetADOWikiPageCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.WikiPageCount
	}
	return 0
}

// GetADOPackageFeedCount returns the package feed count from ADO properties
func (r *Repository) GetADOPackageFeedCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.PackageFeedCount
	}
	return 0
}

// GetADOServiceHookCount returns the service hook count from ADO properties
func (r *Repository) GetADOServiceHookCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.ServiceHookCount
	}
	return 0
}

// ADO boolean getters

// GetADOHasBoards returns true if ADO has boards
func (r *Repository) GetADOHasBoards() bool {
	return r.ADOProperties != nil && r.ADOProperties.HasBoards
}

// GetADOHasPipelines returns true if ADO has pipelines
func (r *Repository) GetADOHasPipelines() bool {
	return r.ADOProperties != nil && r.ADOProperties.HasPipelines
}

// GetADOHasServiceConnections returns true if ADO has service connections
func (r *Repository) GetADOHasServiceConnections() bool {
	return r.ADOProperties != nil && r.ADOProperties.HasServiceConnections
}

// GetADOHasVariableGroups returns true if ADO has variable groups
func (r *Repository) GetADOHasVariableGroups() bool {
	return r.ADOProperties != nil && r.ADOProperties.HasVariableGroups
}

// GetADOActiveWorkItemCount returns the active work item count from ADO properties
func (r *Repository) GetADOActiveWorkItemCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.ActiveWorkItemCount
	}
	return 0
}

// GetADOHasWiki returns true if ADO has wiki
func (r *Repository) GetADOHasWiki() bool {
	return r.ADOProperties != nil && r.ADOProperties.HasWiki
}

// GetADOHasGHAS returns true if ADO has GHAS
func (r *Repository) GetADOHasGHAS() bool {
	return r.ADOProperties != nil && r.ADOProperties.HasGHAS
}

// GetADOTestPlanCount returns the test plan count from ADO properties
func (r *Repository) GetADOTestPlanCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.TestPlanCount
	}
	return 0
}

// GetADOPRWithLinkedWorkItems returns the PR with linked work items count from ADO properties
func (r *Repository) GetADOPRWithLinkedWorkItems() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.PRWithLinkedWorkItems
	}
	return 0
}

// GetADOPipelineRunCount returns the pipeline run count from ADO properties
func (r *Repository) GetADOPipelineRunCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.PipelineRunCount
	}
	return 0
}

// GetADOOpenPRCount returns the open PR count from ADO properties
func (r *Repository) GetADOOpenPRCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.OpenPRCount
	}
	return 0
}

// GetADOPRWithAttachments returns the PR with attachments count from ADO properties
func (r *Repository) GetADOPRWithAttachments() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.PRWithAttachments
	}
	return 0
}

// GetADOWorkItemLinkedCount returns the work item linked count from ADO properties
func (r *Repository) GetADOWorkItemLinkedCount() int {
	if r.ADOProperties != nil {
		return r.ADOProperties.WorkItemLinkedCount
	}
	return 0
}

// Validation getters

// GetValidationStatus returns the validation status
func (r *Repository) GetValidationStatus() *string {
	if r.Validation != nil {
		return r.Validation.ValidationStatus
	}
	return nil
}

// GetValidationDetails returns the validation details
func (r *Repository) GetValidationDetails() *string {
	if r.Validation != nil {
		return r.Validation.ValidationDetails
	}
	return nil
}

// GetDestinationData returns the destination data
func (r *Repository) GetDestinationData() *string {
	if r.Validation != nil {
		return r.Validation.DestinationData
	}
	return nil
}

// GetOversizedCommitDetails returns the oversized commit details
func (r *Repository) GetOversizedCommitDetails() *string {
	if r.Validation != nil {
		return r.Validation.OversizedCommitDetails
	}
	return nil
}

// GetLongRefDetails returns the long ref details
func (r *Repository) GetLongRefDetails() *string {
	if r.Validation != nil {
		return r.Validation.LongRefDetails
	}
	return nil
}

// GetBlockingFileDetails returns the blocking file details
func (r *Repository) GetBlockingFileDetails() *string {
	if r.Validation != nil {
		return r.Validation.BlockingFileDetails
	}
	return nil
}

// GetLargeFileWarningDetails returns the large file warning details
func (r *Repository) GetLargeFileWarningDetails() *string {
	if r.Validation != nil {
		return r.Validation.LargeFileWarningDetails
	}
	return nil
}

// GetOversizedRepositoryDetails returns the oversized repository details
func (r *Repository) GetOversizedRepositoryDetails() *string {
	if r.Validation != nil {
		return r.Validation.OversizedRepositoryDetails
	}
	return nil
}

// GetEstimatedMetadataSize returns the estimated metadata size
func (r *Repository) GetEstimatedMetadataSize() *int64 {
	if r.Validation != nil {
		return r.Validation.EstimatedMetadataSize
	}
	return nil
}

// GetMetadataSizeDetails returns the metadata size details
func (r *Repository) GetMetadataSizeDetails() *string {
	if r.Validation != nil {
		return r.Validation.MetadataSizeDetails
	}
	return nil
}

// GetComplexityBreakdownString returns the complexity breakdown as a string
func (r *Repository) GetComplexityBreakdownString() *string {
	if r.Validation != nil {
		return r.Validation.ComplexityBreakdown
	}
	return nil
}
