import { useState } from 'react';
import { Button, Label } from '@primer/react';
import { AlertIcon, XCircleIcon, ClockIcon, ChevronDownIcon, ChevronRightIcon } from '@primer/octicons-react';
import { DashboardActionItems } from '../../types';
import { Link } from 'react-router-dom';
import { formatDate } from '../../utils/format';

interface ActionItemsPanelProps {
  actionItems: DashboardActionItems | undefined;
  isLoading: boolean;
}

export function ActionItemsPanel({ actionItems, isLoading }: ActionItemsPanelProps) {
  if (isLoading || !actionItems) {
    return (
      <div className="mb-8">
        <div 
          className="rounded-lg border p-6 animate-pulse"
          style={{
            backgroundColor: 'var(--bgColor-default)',
            borderColor: 'var(--borderColor-default)',
          }}
        >
          <div className="h-6 bg-gray-300 rounded w-1/3 mb-4"></div>
          <div className="space-y-3">
            <div className="h-4 bg-gray-200 rounded w-full"></div>
            <div className="h-4 bg-gray-200 rounded w-5/6"></div>
            <div className="h-4 bg-gray-200 rounded w-4/6"></div>
          </div>
        </div>
      </div>
    );
  }

  const failedMigrationsCount = actionItems.failed_migrations.length;
  const failedDryRunsCount = actionItems.failed_dry_runs.length;
  const readyBatchesCount = actionItems.ready_batches.length;
  const blockedReposCount = actionItems.blocked_repositories.length;

  const totalActionItems = failedMigrationsCount + failedDryRunsCount + readyBatchesCount + blockedReposCount;

  // Hide the panel completely when there are no action items
  if (totalActionItems === 0) {
    return null;
  }

  return (
    <div className="mb-8">
      <div 
        className="rounded-lg border"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderColor: 'var(--borderColor-default)',
          boxShadow: 'var(--shadow-resting-small)'
        }}
      >
        <div className="p-4 border-b flex items-center justify-between" style={{ borderColor: 'var(--borderColor-default)' }}>
          <div className="flex items-center gap-3">
            <span style={{ color: 'var(--fgColor-attention)' }}>
              <AlertIcon size={20} />
            </span>
            <h2 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
              Action Items
            </h2>
            <Label variant="danger" size="large">
              {totalActionItems}
            </Label>
          </div>
        </div>

        <div className="divide-y" style={{ borderColor: 'var(--borderColor-default)' }}>
          {failedMigrationsCount > 0 && (
            <CollapsibleActionSection
              title="Failed Migrations"
              count={failedMigrationsCount}
              icon={<span style={{ color: 'var(--fgColor-danger)' }}><XCircleIcon size={16} /></span>}
              variant="danger"
              defaultExpanded={true}
            >
              <div className="space-y-2">
                {actionItems.failed_migrations.map((repo) => (
                  <div
                    key={repo.id}
                    className="flex items-center justify-between p-3 rounded border"
                    style={{
                      backgroundColor: 'var(--bgColor-muted)',
                      borderColor: 'var(--borderColor-default)',
                    }}
                  >
                    <div className="flex-1 min-w-0">
                      <Link
                        to={`/repository/${encodeURIComponent(repo.full_name)}`}
                        className="font-medium hover:underline"
                        style={{ color: 'var(--fgColor-accent)' }}
                      >
                        {repo.full_name}
                      </Link>
                      {repo.batch_name && (
                        <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                          Batch: {repo.batch_name}
                        </div>
                      )}
                      {repo.failed_at && (
                        <div className="text-xs mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                          Failed: {formatDate(repo.failed_at)}
                        </div>
                      )}
                    </div>
                    <Link to={`/repository/${encodeURIComponent(repo.full_name)}`}>
                      <Button variant="danger" size="small">
                        View Details
                      </Button>
                    </Link>
                  </div>
                ))}
              </div>
            </CollapsibleActionSection>
          )}

          {failedDryRunsCount > 0 && (
            <CollapsibleActionSection
              title="Failed Dry Runs"
              count={failedDryRunsCount}
              icon={<span style={{ color: 'var(--fgColor-attention)' }}><AlertIcon size={16} /></span>}
              variant="attention"
              defaultExpanded={true}
            >
              <div className="space-y-2">
                {actionItems.failed_dry_runs.map((repo) => (
                  <div
                    key={repo.id}
                    className="flex items-center justify-between p-3 rounded border"
                    style={{
                      backgroundColor: 'var(--bgColor-muted)',
                      borderColor: 'var(--borderColor-default)',
                    }}
                  >
                    <div className="flex-1 min-w-0">
                      <Link
                        to={`/repository/${encodeURIComponent(repo.full_name)}`}
                        className="font-medium hover:underline"
                        style={{ color: 'var(--fgColor-accent)' }}
                      >
                        {repo.full_name}
                      </Link>
                      {repo.batch_name && (
                        <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                          Batch: {repo.batch_name}
                        </div>
                      )}
                      {repo.failed_at && (
                        <div className="text-xs mt-0.5" style={{ color: 'var(--fgColor-muted)' }}>
                          Failed: {formatDate(repo.failed_at)}
                        </div>
                      )}
                    </div>
                    <Link to={`/repository/${encodeURIComponent(repo.full_name)}`}>
                      <Button variant="default" size="small">
                        View Details
                      </Button>
                    </Link>
                  </div>
                ))}
              </div>
            </CollapsibleActionSection>
          )}

          {readyBatchesCount > 0 && (
            <CollapsibleActionSection
              title="Batches Ready to Start"
              count={readyBatchesCount}
              icon={<span style={{ color: 'var(--fgColor-accent)' }}><ClockIcon size={16} /></span>}
              variant="accent"
              defaultExpanded={true}
            >
              <div className="space-y-2">
                {actionItems.ready_batches.map((batch) => (
                  <div
                    key={batch.id}
                    className="flex items-center justify-between p-3 rounded border"
                    style={{
                      backgroundColor: 'var(--bgColor-muted)',
                      borderColor: 'var(--borderColor-default)',
                    }}
                  >
                    <div className="flex-1 min-w-0">
                      <Link
                        to={`/batches`}
                        state={{ selectedBatchId: batch.id }}
                        className="font-medium hover:underline"
                        style={{ color: 'var(--fgColor-accent)' }}
                      >
                        {batch.name}
                      </Link>
                      <div className="text-xs mt-1" style={{ color: 'var(--fgColor-muted)' }}>
                        {batch.repository_count} repositories
                        {batch.scheduled_at && ` • Scheduled: ${formatDate(batch.scheduled_at)}`}
                      </div>
                    </div>
                    <Link to={`/batches`} state={{ selectedBatchId: batch.id }}>
                      <Button variant="primary" size="small">
                        View Batch
                      </Button>
                    </Link>
                  </div>
                ))}
              </div>
            </CollapsibleActionSection>
          )}

          {blockedReposCount > 0 && (
            <CollapsibleActionSection
              title="Blocked Repositories"
              count={blockedReposCount}
              icon={<span style={{ color: 'var(--fgColor-attention)' }}><AlertIcon size={16} /></span>}
              variant="attention"
              defaultExpanded={false}
            >
              <div className="space-y-2">
                {actionItems.blocked_repositories.slice(0, 10).map((repo) => (
                  <div
                    key={repo.id}
                    className="flex items-center justify-between p-3 rounded border"
                    style={{
                      backgroundColor: 'var(--bgColor-muted)',
                      borderColor: 'var(--borderColor-default)',
                    }}
                  >
                    <div className="flex-1 min-w-0">
                      <Link
                        to={`/repository/${encodeURIComponent(repo.full_name)}`}
                        className="font-medium hover:underline"
                        style={{ color: 'var(--fgColor-accent)' }}
                      >
                        {repo.full_name}
                      </Link>
                      <div className="text-xs mt-1 flex gap-2">
                        {repo.has_oversized_repository && (
                          <Label variant="danger" size="small">Oversized (&gt;40GB)</Label>
                        )}
                        {repo.has_blocking_files && (
                          <Label variant="attention" size="small">Blocking Files</Label>
                        )}
                        {repo.status === 'remediation_required' && (
                          <Label variant="attention" size="small">Remediation Required</Label>
                        )}
                      </div>
                    </div>
                    <Link to={`/repository/${encodeURIComponent(repo.full_name)}`}>
                      <Button variant="default" size="small">
                        View Details
                      </Button>
                    </Link>
                  </div>
                ))}
                {blockedReposCount > 10 && (
                  <div className="text-center pt-2">
                    <Link to="/repositories?status=remediation_required">
                      <Button variant="invisible">
                        View all {blockedReposCount} blocked repositories →
                      </Button>
                    </Link>
                  </div>
                )}
              </div>
            </CollapsibleActionSection>
          )}
        </div>
      </div>
    </div>
  );
}

interface CollapsibleActionSectionProps {
  title: string;
  count: number;
  icon: React.ReactNode;
  variant: 'danger' | 'attention' | 'accent';
  defaultExpanded: boolean;
  children: React.ReactNode;
}

function CollapsibleActionSection({ 
  title, 
  count, 
  icon, 
  variant, 
  defaultExpanded, 
  children 
}: CollapsibleActionSectionProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <div>
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between p-4 transition-colors"
        style={{
          backgroundColor: expanded ? 'var(--bgColor-muted)' : 'transparent',
          border: 'none',
          cursor: 'pointer',
        }}
        onMouseEnter={(e) => {
          if (!expanded) {
            e.currentTarget.style.backgroundColor = 'var(--bgColor-muted)';
          }
        }}
        onMouseLeave={(e) => {
          if (!expanded) {
            e.currentTarget.style.backgroundColor = 'transparent';
          }
        }}
      >
        <div className="flex items-center gap-3">
          {expanded ? <ChevronDownIcon size={16} /> : <ChevronRightIcon size={16} />}
          {icon}
          <span className="font-medium" style={{ color: 'var(--fgColor-default)' }}>
            {title}
          </span>
          <Label variant={variant} size="small">
            {count}
          </Label>
        </div>
      </button>

      {expanded && (
        <div className="px-4 pb-4">
          {children}
        </div>
      )}
    </div>
  );
}
