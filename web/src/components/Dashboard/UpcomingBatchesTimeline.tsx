import { Link } from 'react-router-dom';
import { Button, ProgressBar, Label } from '@primer/react';
import { ClockIcon, CalendarIcon, PlayIcon, PackageIcon } from '@primer/octicons-react';
import { Batch } from '../../types';
// import { formatDate } from '../../utils/format';

interface UpcomingBatchesTimelineProps {
  batches: Batch[];
  isLoading: boolean;
}

export function UpcomingBatchesTimeline({ batches, isLoading }: UpcomingBatchesTimelineProps) {
  if (isLoading) {
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
          </div>
        </div>
      </div>
    );
  }

  // Filter to only show ready, in_progress, and pending batches with scheduled dates
  const upcomingBatches = batches
    .filter(b => 
      (b.status === 'ready' || b.status === 'in_progress' || b.status === 'pending') &&
      (b.status === 'in_progress' || b.scheduled_at)
    )
    .sort((a, b) => {
      // In-progress batches first
      if (a.status === 'in_progress' && b.status !== 'in_progress') return -1;
      if (b.status === 'in_progress' && a.status !== 'in_progress') return 1;
      
      // Then sort by scheduled date
      if (!a.scheduled_at && !b.scheduled_at) return 0;
      if (!a.scheduled_at) return 1;
      if (!b.scheduled_at) return -1;
      return new Date(a.scheduled_at).getTime() - new Date(b.scheduled_at).getTime();
    })
    .slice(0, 5);

  if (upcomingBatches.length === 0) {
    return null;
  }

  return (
    <div className="mb-8">
      <div
        className="rounded-lg border"
        style={{
          backgroundColor: 'var(--bgColor-default)',
          borderColor: 'var(--borderColor-default)',
          boxShadow: 'var(--shadow-resting-small)',
        }}
      >
        <div className="p-4 border-b flex items-center justify-between" style={{ borderColor: 'var(--borderColor-default)' }}>
          <div className="flex items-center gap-3">
            <span style={{ color: 'var(--fgColor-accent)' }}>
              <CalendarIcon size={20} />
            </span>
            <h2 className="text-lg font-semibold" style={{ color: 'var(--fgColor-default)' }}>
              Upcoming Batches
            </h2>
          </div>
          <Link to="/batches">
            <Button variant="invisible" size="small">
              View All Batches →
            </Button>
          </Link>
        </div>

        <div className="divide-y" style={{ borderColor: 'var(--borderColor-default)' }}>
          {upcomingBatches.map((batch) => (
            <BatchTimelineItem key={batch.id} batch={batch} />
          ))}
        </div>
      </div>
    </div>
  );
}

interface BatchTimelineItemProps {
  batch: Batch;
}

function BatchTimelineItem({ batch }: BatchTimelineItemProps) {
  const isInProgress = batch.status === 'in_progress';
  const isReady = batch.status === 'ready';
  const isPending = batch.status === 'pending';

  // Calculate countdown for scheduled batches
  const getCountdown = () => {
    if (!batch.scheduled_at) return null;
    
    const scheduledDate = new Date(batch.scheduled_at);
    const now = new Date();
    const diffMs = scheduledDate.getTime() - now.getTime();
    
    if (diffMs < 0) return 'Now';
    
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    const diffHours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    
    if (diffDays > 7) {
      return `in ${diffDays} days`;
    } else if (diffDays > 0) {
      return `in ${diffDays}d ${diffHours}h`;
    } else if (diffHours > 0) {
      return `in ${diffHours}h`;
    } else {
      return 'within 1 hour';
    }
  };

  const countdown = getCountdown();

  // Get progress from batch data (calculated by backend)
  const progress = batch.percent_complete ?? 0;

  return (
    <div className="p-4 hover:bg-[var(--bgColor-muted)] transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <span style={{ color: 'var(--fgColor-accent)' }}>
              <PackageIcon size={16} />
            </span>
            <Link
              to="/batches"
              state={{ selectedBatchId: batch.id }}
              className="font-medium hover:underline truncate"
              style={{ color: 'var(--fgColor-accent)' }}
            >
              {batch.name}
            </Link>
            {isInProgress && <Label variant="accent">In Progress</Label>}
            {isReady && <Label variant="success">Ready</Label>}
            {isPending && <Label variant="default">Pending</Label>}
          </div>

          {batch.description && (
            <p className="text-sm mb-2 line-clamp-2" style={{ color: 'var(--fgColor-muted)' }}>
              {batch.description}
            </p>
          )}

          <div className="flex items-center gap-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
            <span>{batch.repository_count} repositories</span>
            {batch.scheduled_at && !isInProgress && (
              <div className="flex items-center gap-1">
                <ClockIcon size={12} />
                <span>{countdown}</span>
              </div>
            )}
          </div>

          {isInProgress && (
            <div className="mt-3">
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs" style={{ color: 'var(--fgColor-muted)' }}>
                  Progress
                </span>
                <span className="text-xs font-medium" style={{ color: 'var(--fgColor-accent)' }}>
                  {Math.round(progress)}%
                </span>
              </div>
              <ProgressBar 
                progress={progress} 
                aria-label={`${batch.name} ${Math.round(progress)}% complete`}
                bg="accent.emphasis"
                barSize="small"
              />
            </div>
          )}
        </div>

        <div className="flex-shrink-0">
          <Link to="/batches" state={{ selectedBatchId: batch.id }}>
            <Button variant={isReady ? 'primary' : 'default'} size="small" leadingVisual={isReady ? PlayIcon : undefined}>
              {isReady ? 'Start' : 'View'}
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
