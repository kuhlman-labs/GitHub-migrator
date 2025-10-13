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
  last_commit_sha?: string;
  last_commit_date?: string;
  has_wiki: boolean;
  has_pages: boolean;
  has_discussions: boolean;
  has_actions: boolean;
  has_projects: boolean;
  branch_protections: number;
  environment_count: number;
  secret_count: number;
  variable_count: number;
  webhook_count: number;
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
  discovered_at: string;
  updated_at: string;
  migrated_at?: string;
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
}

export interface Organization {
  organization: string;
  total_repos: number;
  status_counts: Record<string, number>;
}

export interface SizeDistribution {
  category: string;
  count: number;
}

export interface FeatureStats {
  has_lfs: number;
  has_submodules: number;
  has_large_files: number;
  has_wiki: number;
  has_pages: number;
  has_discussions: number;
  has_actions: number;
  has_projects: number;
  has_branch_protections: number;
  total_repositories: number;
}

export interface Analytics {
  total_repositories: number;
  migrated_count: number;
  failed_count: number;
  in_progress_count: number;
  pending_count: number;
  average_migration_time?: number;
  status_breakdown: Record<string, number>;
  organization_stats?: Organization[];
  size_distribution?: SizeDistribution[];
  feature_stats?: FeatureStats;
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

