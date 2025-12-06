export interface Repository {
  id: number;
  full_name: string;
  source: string;
  source_url: string;
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

export interface MigrationHistory {
  id: number;
  repository_id: number;
  status: string;
  phase: string;
  message: string;
  error_message?: string;
  started_at: string;
  completed_at?: string;
  duration_seconds?: number;
}

export interface MigrationLog {
  id: number;
  repository_id: number;
  history_id?: number;
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  phase: string;
  operation: string;
  message: string;
  details?: string;
  initiated_by?: string; // GitHub username of user who initiated the action
  timestamp: string;
}

export interface Batch {
  id: number;
  name: string;
  description: string;
  type: string;
  repository_count: number;
  status: string;
  scheduled_at?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  last_dry_run_at?: string;
  last_migration_attempt_at?: string;
  // Migration settings (batch-level defaults, repository settings take precedence)
  destination_org?: string;
  migration_api?: 'GEI' | 'ELM';
  exclude_releases?: boolean;
}

// Helper function to calculate batch duration in seconds
export function getBatchDuration(batch: Batch): number | null {
  if (!batch.started_at || !batch.completed_at) {
    return null;
  }
  const startTime = new Date(batch.started_at).getTime();
  const endTime = new Date(batch.completed_at).getTime();
  return (endTime - startTime) / 1000; // Duration in seconds
}

// Helper function to format batch duration as human-readable string
export function formatBatchDuration(batch: Batch): string | null {
  const durationSeconds = getBatchDuration(batch);
  if (durationSeconds === null) {
    return null;
  }

  const hours = Math.floor(durationSeconds / 3600);
  const minutes = Math.floor((durationSeconds % 3600) / 60);
  const seconds = Math.floor(durationSeconds % 60);

  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  } else {
    return `${seconds}s`;
  }
}

export interface Organization {
  organization: string;
  total_repos: number;
  total_projects?: number; // For ADO orgs, total number of projects
  status_counts: Record<string, number>;
  ado_organization?: string; // For ADO projects, the parent organization name
  enterprise?: string; // For GitHub orgs, the parent enterprise name (future enhancement)
  migrated_count: number;
  in_progress_count: number;
  failed_count: number;
  pending_count: number;
  migration_progress_percentage: number;
}

export interface Project {
  project: string;
  total_repos: number;
  status_counts: Record<string, number>;
}

// GitHub Team type - teams are org-scoped
export interface GitHubTeam {
  id: number;
  organization: string;
  slug: string;
  name: string;
  description?: string;
  privacy: string;
  full_slug: string; // "org/team-slug" format for unique identification
}

// Azure DevOps Project type
export interface ADOProject {
  id: string;
  name: string;
  organization: string;
  description?: string;
  url: string;
  status_counts?: Record<string, number>;
  state: string;
  visibility: string;
  last_update: string;
  discovered_at: string;
  updated_at: string;
  repository_count?: number; // Populated when listing projects
}

export interface ADODiscoveryRequest {
  organization: string;
  projects?: string[];
  workers?: number;
}

export interface ADODiscoveryStatus {
  total_repositories: number;
  total_projects: number;
  tfvc_repositories: number;
  git_repositories: number;
  status_breakdown: Record<string, number>;
  organization?: string;
}

export interface SizeDistribution {
  category: string;
  count: number;
}

export interface FeatureStats {
  // GitHub features
  is_archived: number;
  is_fork: number;
  has_lfs: number;
  has_submodules: number;
  has_large_files: number;
  has_wiki: number;
  has_pages: number;
  has_discussions: number;
  has_actions: number;
  has_projects: number;
  has_packages: number;
  has_branch_protections: number;
  has_rulesets: number;
  has_code_scanning: number;
  has_dependabot: number;
  has_secret_scanning: number;
  has_codeowners: number;
  has_self_hosted_runners: number;
  has_release_assets: number;
  has_webhooks: number;
  has_environments: number;
  has_secrets: number;
  has_variables: number;
  
  // Azure DevOps features
  ado_tfvc_count: number;
  ado_has_boards: number;
  ado_has_pipelines: number;
  ado_has_ghas: number;
  ado_has_pull_requests: number;
  ado_has_work_items: number;
  ado_has_branch_policies: number;
  ado_has_yaml_pipelines: number;
  ado_has_classic_pipelines: number;
  ado_has_wiki: number;
  ado_has_test_plans: number;
  ado_has_package_feeds: number;
  ado_has_service_hooks: number;
  
  total_repositories: number;
}

export interface MigrationCompletionStats {
  organization: string;
  total_repos: number;
  completed_count: number;
  in_progress_count: number;
  pending_count: number;
  failed_count: number;
}

export interface ComplexityDistribution {
  category: string;
  count: number;
}

export interface MigrationVelocity {
  repos_per_day: number;
  repos_per_week: number;
}

export interface MigrationTimeSeriesPoint {
  date: string;
  count: number;
}

export interface Analytics {
  total_repositories: number;
  migrated_count: number;
  failed_count: number;
  in_progress_count: number;
  pending_count: number;
  success_rate?: number;
  average_migration_time?: number;
  median_migration_time?: number;
  status_breakdown: Record<string, number>;
  complexity_distribution?: ComplexityDistribution[];
  migration_velocity?: MigrationVelocity;
  migration_time_series?: MigrationTimeSeriesPoint[];
  estimated_completion_date?: string;
  organization_stats?: Organization[];
  project_stats?: Organization[];
  size_distribution?: SizeDistribution[];
  feature_stats?: FeatureStats;
  migration_completion_stats?: MigrationCompletionStats[];
}

export interface ExecutiveSummary {
  total_repositories: number;
  completion_percentage: number;
  migrated_count: number;
  in_progress_count: number;
  pending_count: number;
  failed_count: number;
  success_rate: number;
  estimated_completion_date?: string;
  days_remaining: number;
  first_migration_date?: string;
  report_generated_at: string;
}

export interface VelocityMetrics {
  repos_per_day: number;
  repos_per_week: number;
  average_duration_sec: number;
  migration_trend?: MigrationTimeSeriesPoint[];
}

export interface RiskAnalysis {
  high_complexity_pending: number;
  very_large_pending: number;
  failed_migrations: number;
  complexity_distribution: ComplexityDistribution[];
  size_distribution: SizeDistribution[];
}

export interface ADORiskAnalysis {
  tfvc_repos: number;
  classic_pipelines: number;
  repos_with_active_work_items: number;
  repos_with_wikis: number;
  repos_with_test_plans: number;
  repos_with_package_feeds: number;
}

export interface BatchPerformance {
  total_batches: number;
  completed_batches: number;
  in_progress_batches: number;
  pending_batches: number;
}

export interface ExecutiveReport {
  source_type: 'github' | 'azuredevops';
  executive_summary: ExecutiveSummary;
  velocity_metrics: VelocityMetrics;
  organization_progress: MigrationCompletionStats[];
  project_progress?: MigrationCompletionStats[]; // For Azure DevOps
  risk_analysis: RiskAnalysis;
  ado_risk_analysis?: ADORiskAnalysis; // For Azure DevOps specific risks
  batch_performance: BatchPerformance;
  feature_migration_status: FeatureStats;
  status_breakdown: Record<string, number>;
}

export interface MigrationHistoryEntry {
  id: number;
  full_name: string;
  source_url: string;
  destination_url?: string;
  status: string;
  started_at?: string;
  completed_at?: string;
  duration_seconds?: number;
}

export interface RepositoryDetailResponse {
  repository: Repository;
  history: MigrationHistory[];
}

export interface MigrationLogsResponse {
  logs: MigrationLog[];
  total: number;
}

export interface RollbackRequest {
  reason?: string;
}

export type MigrationStatus = 
  | 'pending'
  | 'remediation_required'
  | 'dry_run_queued'
  | 'dry_run_in_progress'
  | 'dry_run_complete'
  | 'dry_run_failed'
  | 'pre_migration'
  | 'archive_generating'
  | 'queued_for_migration'
  | 'migrating_content'
  | 'migration_complete'
  | 'migration_failed'
  | 'post_migration'
  | 'complete'
  | 'rolled_back'
  | 'wont_migrate';

export type BatchStatus =
  | 'pending'
  | 'ready'
  | 'in_progress'
  | 'completed'
  | 'completed_with_errors'
  | 'failed'
  | 'cancelled';

export interface RepositoryFilters {
  status?: string | string[];
  batch_id?: number;
  source?: string;
  organization?: string | string[];
  ado_organization?: string | string[]; // For Azure DevOps organizations (filters by ado_projects table)
  project?: string | string[]; // For Azure DevOps projects
  team?: string | string[]; // For GitHub teams (format: "org/team-slug")
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
  ado_pull_request_count?: string; // Supports '> 0' for filtering
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

// Dependent repository (repos that depend on a target repo)
export interface DependentRepository {
  id: number;
  full_name: string;
  source_url: string;
  status: string;
  dependency_types: string[]; // How this repo depends on target
}

export interface DependentsResponse {
  dependents: DependentRepository[];
  total: number;
  target: string;
}

// Dependency graph types for enterprise-wide visualization
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

// Dependency export row
export interface DependencyExportRow {
  repository: string;
  dependency_full_name: string;
  direction: 'depends_on' | 'depended_by';
  dependency_type: string;
  dependency_url: string;
}

// Setup wizard types
export interface SetupStatus {
  setup_completed: boolean;
  completed_at?: string;
  current_config?: MaskedConfigData;
}

export interface MaskedConfigData {
  source_type: string;
  source_base_url: string;
  source_token: string; // masked
  dest_base_url: string;
  dest_token: string; // masked
  database_type: string;
  database_dsn: string; // masked
  server_port: number;
}

export interface SetupConfig {
  source: {
    type: 'github' | 'azuredevops';
    base_url: string;
    token: string;
    organization?: string; // For Azure DevOps
    // GitHub App for enhanced discovery (optional, only when source is GitHub)
    app_id?: number;
    app_private_key?: string;
    app_installation_id?: number;
  };
  destination: {
    base_url: string;
    token: string;
    // GitHub App for enhanced discovery (optional, always available since destination is GitHub)
    app_id?: number;
    app_private_key?: string;
    app_installation_id?: number;
  };
  database: {
    type: 'sqlite' | 'postgres' | 'sqlserver';
    dsn: string;
  };
  server: {
    port: number;
  };
  migration: {
    workers: number;
    poll_interval_seconds: number;
    dest_repo_exists_action: 'fail' | 'skip' | 'delete';
    visibility_handling: {
      public_repos: 'public' | 'internal' | 'private';
      internal_repos: 'internal' | 'private';
    };
  };
  logging: {
    level: 'debug' | 'info' | 'warn' | 'error';
    format: 'json' | 'text';
    output_file: string;
  };
  auth?: {
    enabled: boolean;
    // GitHub OAuth (only when source or destination is GitHub)
    github_oauth_client_id?: string;
    github_oauth_client_secret?: string;
    github_oauth_base_url?: string;
    // Azure AD (only when source is Azure DevOps)
    azure_ad_tenant_id?: string;
    azure_ad_client_id?: string;
    azure_ad_client_secret?: string;
    // Common auth settings
    callback_url?: string;
    frontend_url?: string;
    session_secret?: string;
    session_duration_hours?: number;
    // Authorization rules (optional)
    authorization_rules?: {
      require_org_membership?: string[]; // List of org names
      require_team_membership?: string[]; // List of "org/team-slug"
      require_enterprise_admin?: boolean;
      require_enterprise_membership?: boolean;
      enterprise_slug?: string;
      privileged_teams?: string[]; // List of "org/team-slug"
    };
  };
}

export interface ValidationResult {
  valid: boolean;
  error?: string;
  warnings?: string[];
  details?: Record<string, unknown>;
}

// Migration settings that can be imported from files
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

// Repository with imported migration settings
export interface ImportedRepository extends Repository {
  importedSettings?: ImportedMigrationSettings;
}

// Dashboard action items
export interface FailedRepository {
  id: number;
  full_name: string;
  organization: string;
  status: string;
  error_summary?: string;
  failed_at?: string;
  batch_id?: number;
  batch_name?: string;
}

export interface DashboardActionItems {
  failed_migrations: FailedRepository[];
  failed_dry_runs: FailedRepository[];
  ready_batches: Batch[];
  blocked_repositories: Repository[];
}

