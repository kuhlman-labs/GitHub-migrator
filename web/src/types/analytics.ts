/**
 * Analytics and reporting types for migration dashboards and executive reports.
 */

import type { Organization } from './common';
import type { FeatureStats } from './common';

export interface SizeDistribution {
  category: string;
  count: number;
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
  project_progress?: MigrationCompletionStats[];
  risk_analysis: RiskAnalysis;
  ado_risk_analysis?: ADORiskAnalysis;
  batch_performance: BatchPerformance;
  feature_migration_status: FeatureStats;
  status_breakdown: Record<string, number>;
}

