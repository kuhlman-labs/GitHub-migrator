/**
 * Repository-related types including filters, dependencies, and complexity.
 */

export interface Repository {
  id: number;
  full_name: string;
  source: string;
  source_url: string;
  source_id?: number; // Foreign key to sources table
  total_size: number;
  largest_file?: string;
  largest_file_size?: number;
  largest_commit?: string;
  largest_commit_size?: number;
  has_lfs: boolean;
  has_submodules: boolean;
  has_large_files: boolean;
  large_file_count: number;
  default_branch: string;
  branch_count: number;
  commit_count: number;
  commits_last_12_weeks: number;
  last_commit_sha?: string;
  last_commit_date?: string;
  is_archived: boolean;
  is_fork: boolean;
  has_wiki: boolean;
  has_pages: boolean;
  has_discussions: boolean;
  has_actions: boolean;
  has_projects: boolean;
  has_packages: boolean;
  branch_protections: number;
  has_rulesets: boolean;
  tag_protection_count: number;
  environment_count: number;
  secret_count: number;
  variable_count: number;
  webhook_count: number;
  // Security & Compliance
  has_code_scanning: boolean;
  has_dependabot: boolean;
  has_secret_scanning: boolean;
  has_codeowners: boolean;
  // Repository Settings
  visibility: 'public' | 'private' | 'internal';
  workflow_count: number;
  // Infrastructure & Access
  has_self_hosted_runners: boolean;
  collaborator_count: number;
  installed_apps_count: number;
  installed_apps?: string; // JSON array of app names
  // Releases
  release_count: number;
  has_release_assets: boolean;
  contributor_count: number;
  top_contributors?: string;
  issue_count: number;
  pull_request_count: number;
  tag_count: number;
  open_issue_count: number;
  open_pr_count: number;
  status: string;
  batch_id?: number;
  priority: number;
  destination_url?: string;
  destination_full_name?: string;
  source_migration_id?: number;
  is_source_locked: boolean;
  discovered_at: string;
  updated_at: string;
  migrated_at?: string;
  last_discovery_at?: string;
  last_dry_run_at?: string;
  // GitHub Migration Limit Validations
  has_oversized_commits: boolean;
  oversized_commit_details?: string;
  has_long_refs: boolean;
  long_ref_details?: string;
  has_blocking_files: boolean;
  blocking_file_details?: string;
  has_large_file_warnings: boolean;
  large_file_warning_details?: string;
  // Repository Size Validation (40 GiB limit)
  has_oversized_repository: boolean;
  oversized_repository_details?: string;
  // Metadata Size Estimation (40 GiB metadata limit)
  estimated_metadata_size?: number;
  metadata_size_details?: string;
  // Migration Exclusion Flags (per-repository settings)
  exclude_releases: boolean;
  exclude_attachments: boolean;
  exclude_metadata: boolean;
  exclude_git_data: boolean;
  exclude_owner_projects: boolean;
  // Azure DevOps specific fields
  ado_project?: string;
  ado_is_git: boolean;
  ado_has_boards: boolean;
  ado_has_pipelines: boolean;
  ado_has_ghas: boolean;
  ado_pull_request_count: number;
  ado_work_item_count: number;
  ado_branch_policy_count: number;
  // Enhanced Pipeline Data
  ado_pipeline_count: number;
  ado_yaml_pipeline_count: number;
  ado_classic_pipeline_count: number;
  ado_pipeline_run_count: number;
  ado_has_service_connections: boolean;
  ado_has_variable_groups: boolean;
  ado_has_self_hosted_agents: boolean;
  // Enhanced Work Item Data
  ado_work_item_linked_count: number;
  ado_active_work_item_count: number;
  ado_work_item_types?: string;
  // Pull Request Details
  ado_open_pr_count: number;
  ado_pr_with_linked_work_items: number;
  ado_pr_with_attachments: number;
  // Enhanced Branch Policy Data
  ado_branch_policy_types?: string;
  ado_required_reviewer_count: number;
  ado_build_validation_policies: number;
  // Wiki & Documentation
  ado_has_wiki: boolean;
  ado_wiki_page_count: number;
  // Test Plans
  ado_test_plan_count: number;
  ado_test_case_count: number;
  // Artifacts & Packages
  ado_package_feed_count: number;
  ado_has_artifacts: boolean;
  // Service Hooks & Extensions
  ado_service_hook_count: number;
  ado_installed_extensions?: string;
  // Computed fields
  complexity_score?: number;
  complexity_breakdown?: ComplexityBreakdown;
}

export interface ComplexityBreakdown {
  size_points: number;
  large_files_points: number;
  environments_points: number;
  secrets_points: number;
  packages_points: number;
  runners_points: number;
  variables_points: number;
  discussions_points: number;
  releases_points: number;
  lfs_points: number;
  submodules_points: number;
  apps_points: number;
  projects_points: number;
  security_points: number;
  webhooks_points: number;
  branch_protections_points: number;
  rulesets_points: number;
  public_visibility_points: number;
  internal_visibility_points: number;
  codeowners_points: number;
  activity_points: number;
  // Azure DevOps specific breakdown
  ado_tfvc_points?: number;
  ado_classic_pipeline_points?: number;
  ado_package_feed_points?: number;
  ado_service_connection_points?: number;
  ado_active_pipeline_points?: number;
  ado_active_boards_points?: number;
  ado_wiki_points?: number;
  ado_test_plan_points?: number;
  ado_variable_group_points?: number;
  ado_service_hook_points?: number;
  ado_many_prs_points?: number;
  ado_branch_policy_points?: number;
}

export interface RepositoryFilters {
  status?: string | string[];
  batch_id?: number;
  source?: string;
  organization?: string | string[];
  ado_organization?: string | string[];
  project?: string | string[];
  team?: string | string[];
  min_size?: number;
  max_size?: number;

  // GitHub features
  has_lfs?: boolean;
  has_submodules?: boolean;
  has_large_files?: boolean;
  has_actions?: boolean;
  has_wiki?: boolean;
  has_pages?: boolean;
  has_discussions?: boolean;
  has_projects?: boolean;
  has_packages?: boolean;
  has_branch_protections?: boolean;
  has_rulesets?: boolean;
  is_archived?: boolean;
  is_fork?: boolean;
  has_code_scanning?: boolean;
  has_dependabot?: boolean;
  has_secret_scanning?: boolean;
  has_codeowners?: boolean;
  visibility?: 'public' | 'private' | 'internal' | string;
  has_self_hosted_runners?: boolean;
  has_release_assets?: boolean;
  has_webhooks?: boolean;
  has_environments?: boolean;
  has_secrets?: boolean;
  has_variables?: boolean;

  // Azure DevOps features
  ado_is_git?: boolean;
  ado_has_boards?: boolean;
  ado_has_pipelines?: boolean;
  ado_has_ghas?: boolean;
  ado_pull_request_count?: string;
  ado_work_item_count?: string;
  ado_branch_policy_count?: string;
  ado_yaml_pipeline_count?: string;
  ado_classic_pipeline_count?: string;
  ado_has_wiki?: boolean;
  ado_test_plan_count?: string;
  ado_package_feed_count?: string;
  ado_service_hook_count?: string;

  complexity?: string | string[];
  size_category?: string | string[];
  search?: string;
  sort_by?: 'name' | 'size' | 'org' | 'updated';
  available_for_batch?: boolean;
  limit?: number;
  offset?: number;
}

export interface RepositoryListResponse {
  repositories: Repository[];
  total?: number;
}

// Dependency types
export type DependencyType = 'submodule' | 'workflow' | 'dependency_graph' | 'package';

export interface RepositoryDependency {
  id: number;
  repository_id: number;
  dependency_full_name: string;
  dependency_type: DependencyType;
  dependency_url: string;
  is_local: boolean;
  discovered_at: string;
  metadata?: string;
}

export interface DependencySummary {
  total: number;
  local: number;
  external: number;
  by_type: Record<string, number>;
}

export interface DependenciesResponse {
  dependencies: RepositoryDependency[];
  summary: DependencySummary;
}

export interface DependentRepository {
  id: number;
  full_name: string;
  source_url: string;
  status: string;
  dependency_types: string[];
}

export interface DependentsResponse {
  dependents: DependentRepository[];
  total: number;
  target: string;
}

export interface DependencyGraphNode {
  id: string;
  full_name: string;
  organization: string;
  status: string;
  depends_on_count: number;
  depended_by_count: number;
}

export interface DependencyGraphEdge {
  source: string;
  target: string;
  dependency_type: string;
}

export interface DependencyGraphStats {
  total_repos_with_dependencies: number;
  total_local_dependencies: number;
  circular_dependency_count: number;
}

export interface DependencyGraphResponse {
  nodes: DependencyGraphNode[];
  edges: DependencyGraphEdge[];
  stats: DependencyGraphStats;
}

export interface DependencyExportRow {
  repository: string;
  dependency_full_name: string;
  direction: 'depends_on' | 'depended_by';
  dependency_type: string;
  dependency_url: string;
}

// Migration settings
export interface ImportedMigrationSettings {
  destination_org?: string;
  destination_repo_name?: string;
  migration_api?: 'GEI' | 'ELM';
  exclude_releases?: boolean;
  exclude_attachments?: boolean;
  exclude_metadata?: boolean;
  exclude_git_data?: boolean;
  exclude_owner_projects?: boolean;
}

export interface ImportedRepository extends Repository {
  importedSettings?: ImportedMigrationSettings;
}

