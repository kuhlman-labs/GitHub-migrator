import { Link } from 'react-router-dom';
import { Button } from '@primer/react';
import type { Repository, Batch } from '../../types';
import { StatusBadge } from '../common/StatusBadge';
import { SourceTypeIcon } from '../common/SourceBadge';
import { formatBytes } from '../../utils/format';

export interface BatchRepositoryItemProps {
  repository: Repository;
  onRetry?: () => void;
  batchId?: number;
  batchName?: string;
  batch?: Batch;
}

export function BatchRepositoryItem({
  repository,
  onRetry,
  batchId,
  batchName,
  batch,
}: BatchRepositoryItemProps) {
  const isFailed = repository.status === 'migration_failed' || repository.status === 'dry_run_failed';
  const isDryRunFailed = repository.status === 'dry_run_failed';

  // Determine destination with batch-level fallback
  let destination = repository.destination_full_name || repository.full_name;
  let isCustomDestination = false;
  let isBatchDestination = false;
  let isDefaultDestination = false;

  // Calculate what the batch default destination would be for this repository
  const sourceRepoName = repository.full_name.split('/')[1];
  const batchDefaultDestination = batch?.destination_org
    ? `${batch.destination_org}/${sourceRepoName}`
    : null;

  // Check if repo has a custom destination
  if (repository.destination_full_name && repository.destination_full_name !== repository.full_name) {
    // Check if the destination matches the batch default
    if (batchDefaultDestination && repository.destination_full_name === batchDefaultDestination) {
      // Destination matches batch default - show as batch destination
      isBatchDestination = true;
    } else {
      // Destination is truly custom (different from both source and batch default)
      isCustomDestination = true;
    }
  } else if (!repository.destination_full_name && batchDefaultDestination) {
    // If repo doesn't have custom destination but batch has destination_org, show that
    destination = batchDefaultDestination;
    isBatchDestination = true;
  } else if (!repository.destination_full_name) {
    // No custom destination and no batch destination - using default (same as source)
    isDefaultDestination = true;
  }

  return (
    <div
      className="flex justify-between items-center p-3 border rounded-lg hover:bg-[var(--bgColor-muted)] group"
      style={{ borderColor: 'var(--borderColor-default)' }}
    >
      <Link
        to={`/repository/${encodeURIComponent(repository.full_name)}`}
        state={{ fromBatch: true, batchId, batchName }}
        className="flex-1 min-w-0"
      >
        <div
          className="font-semibold transition-colors flex items-center gap-2"
          style={{ color: 'var(--fgColor-default)' }}
        >
          {repository.source_id && (
            <SourceTypeIcon sourceId={repository.source_id} size={14} />
          )}
          {repository.full_name}
        </div>
        <div className="text-sm mt-1 space-y-0.5" style={{ color: 'var(--fgColor-muted)' }}>
          <div>
            {formatBytes(repository.total_size || 0)} • {repository.branch_count} branches
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-xs">→ Destination:</span>
            <span
              className="text-xs font-medium"
              style={{
                color: isCustomDestination
                  ? 'var(--fgColor-accent)'
                  : isBatchDestination
                    ? 'var(--fgColor-attention)'
                    : 'var(--fgColor-muted)',
              }}
            >
              {destination}
            </span>
            {isCustomDestination && (
              <span
                className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold uppercase tracking-wide"
                style={{
                  backgroundColor: 'var(--bgColor-accent-emphasis)',
                  color: 'var(--fgColor-onEmphasis)',
                }}
              >
                Custom
              </span>
            )}
            {isBatchDestination && (
              <span
                className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold uppercase tracking-wide"
                style={{
                  backgroundColor: 'var(--bgColor-attention-emphasis)',
                  color: 'var(--fgColor-onEmphasis)',
                }}
              >
                Batch Default
              </span>
            )}
            {isDefaultDestination && (
              <span
                className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold uppercase tracking-wide"
                style={{
                  backgroundColor: 'var(--bgColor-neutral-emphasis)',
                  color: 'var(--fgColor-onEmphasis)',
                }}
              >
                Default
              </span>
            )}
          </div>
        </div>
      </Link>
      <div className="flex items-center gap-3">
        <StatusBadge status={repository.status} size="small" />
        {isFailed && onRetry && (
          <Button
            onClick={(e) => {
              e.preventDefault();
              onRetry();
            }}
            variant="danger"
            size="small"
            title={
              isDryRunFailed
                ? 'Re-run the dry run for this repository'
                : 'Retry the production migration'
            }
          >
            {isDryRunFailed ? 'Re-run Dry Run' : 'Retry Migration'}
          </Button>
        )}
      </div>
    </div>
  );
}

