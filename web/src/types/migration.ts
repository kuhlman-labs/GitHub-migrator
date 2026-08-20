/**
 * Migration-related types for tracking migration history and status.
 */

import type { Repository } from './repository';

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
  initiated_by?: string;
  timestamp: string;
}

export interface MigrationHistoryEntry {
  id: number;
  full_name: string;
  source_url: string;
  destination_url?: string;
  source_id?: number; // Added for multi-source support
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

/**
 * The migration route recorded on a repository. A repository with no route
 * recorded is GEI-routed, which is why the backend reports 'gei' for a null
 * column rather than an empty value.
 */
export type MigrationRoute = 'gei' | 'elm';

/**
 * ELM (Enterprise Live Migrations) status for a single repository, as returned
 * by GET /api/v1/repositories/{fullName}/elm.
 *
 * cutover_ready is the backend's answer to "may this repository be cut over" and
 * is the ONLY thing that enables the dashboard's cutover control.
 */
export interface ELMStatus {
  full_name: string;
  migration_route: MigrationRoute | string;
  status: string;
  has_elm_migration: boolean;
  elm_migration_id?: string;
  elm_status?: string;
  elm_phase?: string;
  progress_percent?: number;
  cutover_ready: boolean;
  last_polled_at?: string;
  last_error?: string;
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
  // ELM (Enterprise Live Migrations) lifecycle
  | 'syncing'
  | 'awaiting_cutover'
  | 'cutting_over'
  | 'migration_complete'
  | 'migration_failed'
  | 'post_migration'
  | 'complete'
  | 'rolled_back'
  | 'wont_migrate';

