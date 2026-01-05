/**
 * Shared/common types used across multiple domains.
 */

import type { Repository } from './repository';
import type { Batch } from './batch';

// Organization types
export interface Organization {
  organization: string;
  total_repos: number;
  total_projects?: number;
  status_counts: Record<string, number>;
  ado_organization?: string;
  enterprise?: string;
  migrated_count: number;
  in_progress_count: number;
  failed_count: number;
  pending_count: number;
  migration_progress_percentage: number;
  /** Source ID for multi-source support */
  source_id?: number;
  /** Display name of the source */
  source_name?: string;
  /** Type of source (github or azuredevops) */
  source_type?: 'github' | 'azuredevops';
}

export interface Project {
  project: string;
  total_repos: number;
  status_counts: Record<string, number>;
}

// Azure DevOps types
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
  repository_count?: number;
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

// Feature statistics
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

// Dashboard types
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

// Setup types
export interface SetupStatus {
  setup_completed: boolean;
  completed_at?: string;
  current_config?: MaskedConfigData;
}

export interface MaskedConfigData {
  source_type: string;
  source_base_url: string;
  source_token: string;
  dest_base_url: string;
  dest_token: string;
  database_type: string;
  database_dsn: string;
  server_port: number;
}

export interface SetupConfig {
  source: {
    type: 'github' | 'azuredevops';
    base_url: string;
    token: string;
    organization?: string;
    app_id?: number;
    app_private_key?: string;
    app_installation_id?: number;
  };
  destination: {
    base_url: string;
    token: string;
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
    github_oauth_client_id?: string;
    github_oauth_client_secret?: string;
    github_oauth_base_url?: string;
    azure_ad_tenant_id?: string;
    azure_ad_client_id?: string;
    azure_ad_client_secret?: string;
    callback_url?: string;
    frontend_url?: string;
    session_secret?: string;
    session_duration_hours?: number;
    authorization_rules?: {
      require_org_membership?: string[];
      require_team_membership?: string[];
      require_enterprise_admin?: boolean;
      require_enterprise_membership?: boolean;
      enterprise_slug?: string;
      privileged_teams?: string[];
    };
  };
}

export interface ValidationResult {
  valid: boolean;
  error?: string;
  warnings?: string[];
  details?: Record<string, unknown>;
}

export interface ImportResult {
  created: number;
  updated: number;
  errors: number;
  messages: string[];
}

// Discovery types
export type DiscoveryPhase =
  | 'listing_repos'
  | 'profiling_repos'
  | 'discovering_teams'
  | 'discovering_members'
  | 'completed';

export type DiscoveryStatus = 'in_progress' | 'completed' | 'failed' | 'none';
export type DiscoveryType = 'enterprise' | 'organization' | 'repository' | 'ado_organization' | 'ado_project';

export interface DiscoveryProgress {
  id: number;
  discovery_type: DiscoveryType;
  target: string;
  status: DiscoveryStatus;
  started_at: string;
  completed_at?: string;
  total_orgs: number;
  processed_orgs: number;
  current_org: string;
  total_repos: number;
  processed_repos: number;
  phase: DiscoveryPhase;
  error_count: number;
  last_error?: string;
}

