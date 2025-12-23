import { ActionMenu, ActionList } from '@primer/react';
import { GearIcon, ClockIcon, PencilIcon, TrashIcon, TriangleDownIcon, PlayIcon, SyncIcon, IterationsIcon, BeakerIcon } from '@primer/octicons-react';
import { Button, SuccessButton, BorderedButton } from '../common/buttons';
import type { Batch, Repository } from '../../types';
import { formatBatchDuration, formatDryRunDuration } from '../../types';
import { StatusBadge } from '../common/StatusBadge';
import { formatDate } from '../../utils/format';

interface BatchDetailHeaderProps {
  batch: Batch;
  batchRepositories: Repository[];
  onEdit: (batch: Batch) => void;
  onDelete: (batch: Batch) => void;
  onDryRun: (batchId: number, onlyPending?: boolean) => void;
  onStart: (batchId: number, skipDryRun?: boolean) => void;
  onRetryFailed: () => void;
  onRollback?: (batch: Batch) => void;
  dryRunButtonRef?: React.RefObject<HTMLButtonElement | null>;
}

export function BatchDetailHeader({
  batch,
  batchRepositories,
  onEdit,
  onDelete,
  onDryRun,
  onStart,
  onRetryFailed,
  onRollback,
  dryRunButtonRef,
}: BatchDetailHeaderProps) {
  // Calculate counts for different states
  const pendingCount = batchRepositories.filter((r) => r.status === 'pending' || r.status === 'discovered').length;
  const dryRunCompleteCount = batchRepositories.filter((r) => r.status === 'dry_run_complete').length;
  const failedCount = batchRepositories.filter(
    (r) => r.status === 'migration_failed' || r.status === 'dry_run_failed'
  ).length;
  const inProgressCount = batchRepositories.filter(
    (r) => r.status === 'in_progress' || r.status === 'dry_run_in_progress'
  ).length;

  const hasPendingRepos = pendingCount > 0;
  const hasDryRunComplete = dryRunCompleteCount > 0;
  const hasFailedRepos = failedCount > 0;
  const isInProgress = batch.status === 'in_progress' || inProgressCount > 0;

  // batchRepositories is used for calculating counts above

  return (
    <>
      <div className="flex justify-between items-start mb-6">
        <div className="flex-1">
          <h2 className="text-xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>{batch.name}</h2>
          {batch.description && (
            <p className="mt-1" style={{ color: 'var(--fgColor-muted)' }}>{batch.description}</p>
          )}
          <div className="flex items-center gap-3 mt-3">
            <StatusBadge status={batch.status} />
            <span className="text-sm" style={{ color: 'var(--fgColor-muted)' }}>
              {batch.repository_count} repositories
            </span>
            {batch.created_at && (
              <>
                <span style={{ color: 'var(--fgColor-muted)' }}>•</span>
                <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                  Created {formatDate(batch.created_at)}
                </span>
              </>
            )}
          </div>
          
          {/* Two-column layout for settings and timestamps */}
          <div className="mt-4 grid grid-cols-1 lg:grid-cols-2 gap-6 border-t border-gh-border-default pt-4">
            {/* Left Column: Migration Settings */}
            {(batch.destination_org || batch.exclude_releases || batch.migration_api !== 'GEI') && (
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <span style={{ color: 'var(--fgColor-muted)' }}>
                    <GearIcon size={16} />
                  </span>
                  <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>Migration Settings</span>
                </div>
                <div className="space-y-2 pl-6">
                  {batch.destination_org && (
                    <div className="text-sm">
                      <span style={{ color: 'var(--fgColor-muted)' }}>Default Destination:</span>
                      <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-accent)' }}>{batch.destination_org}</div>
                      <div className="text-xs italic mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>For repos without specific destination</div>
                    </div>
                  )}
                  {batch.migration_api && batch.migration_api !== 'GEI' && (
                    <div className="text-sm">
                      <span style={{ color: 'var(--fgColor-muted)' }}>Migration API:</span>
                      <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-default)' }}>
                        {batch.migration_api === 'ELM' ? 'ELM (Enterprise Live Migrator)' : batch.migration_api}
                      </div>
                    </div>
                  )}
                  {batch.exclude_releases && (
                    <div className="text-sm">
                      <span style={{ color: 'var(--fgColor-muted)' }}>Exclude Releases:</span>
                      <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-attention)' }}>Yes</div>
                      <div className="text-xs italic mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>Repo settings can override</div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Right Column: Timeline */}
            <div>
              <div className="flex items-center gap-2 mb-2">
                <span style={{ color: 'var(--fgColor-muted)' }}>
                  <ClockIcon size={16} />
                </span>
                <span className="text-sm font-semibold" style={{ color: 'var(--fgColor-default)' }}>Timeline</span>
              </div>
              <div className="space-y-3 pl-6">
                {/* Step 1: Scheduled (if applicable) */}
                {batch.scheduled_at && (
                  <div className="text-sm">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium px-1.5 py-0.5 rounded" style={{ 
                        backgroundColor: new Date(batch.scheduled_at) > new Date() ? 'var(--bgColor-accent-muted)' : 'var(--bgColor-muted)',
                        color: new Date(batch.scheduled_at) > new Date() ? 'var(--fgColor-accent)' : 'var(--fgColor-muted)'
                      }}>
                        {new Date(batch.scheduled_at) > new Date() ? 'SCHEDULED' : 'WAS SCHEDULED'}
                      </span>
                    </div>
                    <div className="font-medium mt-1" style={{ color: 'var(--fgColor-default)' }}>
                      {formatDate(batch.scheduled_at)}
                    </div>
                  </div>
                )}

                {/* Step 2: Dry Run */}
                {batch.last_dry_run_at && (
                  <div className="text-sm">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium px-1.5 py-0.5 rounded" style={{ 
                        backgroundColor: 'var(--bgColor-accent-muted)',
                        color: 'var(--fgColor-accent)'
                      }}>
                        DRY RUN
                      </span>
                    </div>
                    <div className="font-medium mt-1" style={{ color: 'var(--fgColor-default)' }}>
                      {formatDate(batch.last_dry_run_at)}
                    </div>
                    {formatDryRunDuration(batch) ? (
                      <div className="font-semibold mt-1" style={{ color: 'var(--fgColor-accent)' }}>
                        Duration: {formatDryRunDuration(batch)}
                      </div>
                    ) : batch.dry_run_started_at && !batch.dry_run_completed_at ? (
                      <div className="text-xs mt-0.5 italic" style={{ color: 'var(--fgColor-attention)' }}>
                        Dry run in progress...
                      </div>
                    ) : null}
                  </div>
                )}

                {/* Step 3: Production Migration */}
                {batch.started_at && (
                  <div className="text-sm">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium px-1.5 py-0.5 rounded" style={{ 
                        backgroundColor: batch.completed_at ? 'var(--bgColor-success-muted)' : 'var(--bgColor-attention-muted)',
                        color: batch.completed_at ? 'var(--fgColor-success)' : 'var(--fgColor-attention)'
                      }}>
                        {batch.completed_at ? 'MIGRATED' : 'IN PROGRESS'}
                      </span>
                    </div>
                    <div className="font-medium mt-1" style={{ color: 'var(--fgColor-default)' }}>
                      Started: {formatDate(batch.started_at)}
                    </div>
                    {batch.completed_at && (
                      <>
                        <div className="font-medium mt-0.5" style={{ color: 'var(--fgColor-default)' }}>
                          Completed: {formatDate(batch.completed_at)}
                        </div>
                        <div className="font-semibold mt-1" style={{ color: 'var(--fgColor-success)' }}>
                          Duration: {formatBatchDuration(batch)}
                        </div>
                      </>
                    )}
                  </div>
                )}

                {/* Show message if no timeline events yet */}
                {!batch.scheduled_at && !batch.last_dry_run_at && !batch.started_at && (
                  <div className="text-sm italic" style={{ color: 'var(--fgColor-muted)' }}>
                    No activity yet
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        
        {/* Action buttons */}
        <div className="flex items-center gap-2 ml-4 flex-shrink-0">
          {batch.status === 'completed' ? (
            /* Completed batches can only be rolled back, not edited or deleted */
            onRollback && (
              <Button
                size="small"
                variant="danger"
                leadingVisual={IterationsIcon}
                onClick={() => onRollback(batch)}
              >
                Rollback
              </Button>
            )
          ) : (
            /* Non-completed batches can be edited and deleted */
            <>
              <BorderedButton
                size="small"
                leadingVisual={PencilIcon}
                onClick={() => onEdit(batch)}
              >
                Edit
              </BorderedButton>
              <Button
                size="small"
                variant="danger"
                leadingVisual={TrashIcon}
                onClick={() => onDelete(batch)}
              >
                Delete
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Action Bar */}
      {!isInProgress && batch.status !== 'completed' && (
        <div className="flex items-center gap-2 mb-6 pb-4 border-b" style={{ borderColor: 'var(--borderColor-muted)' }}>
          {/* Dry Run Action */}
          {(hasPendingRepos || hasDryRunComplete) && (
            <ActionMenu>
              <ActionMenu.Anchor>
                <Button 
                  ref={dryRunButtonRef}
                  size="small" 
                  variant="primary"
                  leadingVisual={BeakerIcon}
                  trailingVisual={TriangleDownIcon}
                >
                  Dry Run
                </Button>
              </ActionMenu.Anchor>
              <ActionMenu.Overlay>
                <ActionList>
                  <ActionList.Item onSelect={() => onDryRun(batch.id, false)}>
                    Run All ({batchRepositories.length} repos)
                  </ActionList.Item>
                  {hasPendingRepos && (
                    <ActionList.Item onSelect={() => onDryRun(batch.id, true)}>
                      Run Pending Only ({pendingCount} repos)
                    </ActionList.Item>
                  )}
                </ActionList>
              </ActionMenu.Overlay>
            </ActionMenu>
          )}

          {/* Start Migration Action */}
          {batch.status === 'ready' && (
            <SuccessButton
              size="small"
              leadingVisual={PlayIcon}
              onClick={() => onStart(batch.id)}
            >
              Start Migration
            </SuccessButton>
          )}

          {/* Start Migration for Pending Batches (with warning) */}
          {batch.status === 'pending' && batchRepositories.length > 0 && (
            <ActionMenu>
              <ActionMenu.Anchor>
                <SuccessButton
                  size="small"
                  leadingVisual={PlayIcon}
                  trailingVisual={TriangleDownIcon}
                >
                  Start Migration
                </SuccessButton>
              </ActionMenu.Anchor>
              <ActionMenu.Overlay>
                <ActionList>
                  <ActionList.Item onSelect={() => onStart(batch.id, true)}>
                    Start Now (Skip Dry Run)
                  </ActionList.Item>
                </ActionList>
              </ActionMenu.Overlay>
            </ActionMenu>
          )}

          {/* Retry Failed */}
          {hasFailedRepos && (
            <Button
              size="small"
              leadingVisual={SyncIcon}
              onClick={onRetryFailed}
            >
              Retry Failed ({failedCount})
            </Button>
          )}
        </div>
      )}
    </>
  );
}

