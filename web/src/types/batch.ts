/**
 * Batch-related types for migration batch management.
 */

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
  exclude_attachments?: boolean;
  // Progress information (populated by backend for in-progress/completed batches)
  percent_complete?: number;
  completed_repos?: number;
}

// Helper function to calculate batch duration in seconds
export function getBatchDuration(batch: Batch): number | null {
  if (!batch.started_at || !batch.completed_at) {
    return null;
  }
  const startTime = new Date(batch.started_at).getTime();
  const endTime = new Date(batch.completed_at).getTime();
  return (endTime - startTime) / 1000;
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

export type BatchStatus =
  | 'pending'
  | 'ready'
  | 'in_progress'
  | 'completed'
  | 'completed_with_errors'
  | 'failed'
  | 'cancelled';

