import { ProgressBar, Label } from '@primer/react';
import { SyncIcon, CheckCircleIcon, AlertIcon, XCircleIcon, XIcon } from '@primer/octicons-react';
import type { DiscoveryProgress, DiscoveryPhase } from '../../types';

const phaseLabels: Record<DiscoveryPhase, string> = {
  listing_repos: 'Listing repositories...',
  profiling_repos: 'Profiling repositories...',
  discovering_teams: 'Discovering teams...',
  discovering_members: 'Discovering members...',
  completed: 'Completed',
};

interface DiscoveryProgressCardProps {
  progress: DiscoveryProgress;
  onDismiss?: () => void;
}

export function DiscoveryProgressCard({ progress, onDismiss }: DiscoveryProgressCardProps) {
  const percentage = progress.total_repos > 0 
    ? Math.round((progress.processed_repos / progress.total_repos) * 100) 
    : 0;

  const isComplete = progress.status === 'completed';
  const isFailed = progress.status === 'failed';
  const isInProgress = progress.status === 'in_progress';

  // Determine card styling based on status using CSS variables for dark mode compatibility
  const cardStyle = {
    borderColor: isFailed 
      ? 'var(--borderColor-danger-muted, var(--color-danger-muted))' 
      : isComplete 
        ? 'var(--borderColor-success-muted, var(--color-success-muted))' 
        : 'var(--borderColor-accent-muted, var(--color-accent-muted))',
    backgroundColor: isFailed 
      ? 'var(--bgColor-danger-muted, var(--color-danger-subtle))' 
      : isComplete 
        ? 'var(--bgColor-success-muted, var(--color-success-subtle))' 
        : 'var(--bgColor-accent-muted, var(--color-accent-subtle))',
  };

  // Get the icon based on status
  const statusIcon = isFailed 
    ? <XCircleIcon style={{ color: 'var(--fgColor-danger)' }} size={16} />
    : isComplete 
      ? <CheckCircleIcon style={{ color: 'var(--fgColor-success)' }} size={16} />
      : <SyncIcon className="animate-spin" style={{ color: 'var(--fgColor-accent)' }} size={16} />;

  // Get status text
  const statusText = isFailed 
    ? 'Discovery Failed' 
    : isComplete 
      ? 'Discovery Complete' 
      : 'Discovery in Progress';

  // Format the discovery type for display
  const typeLabel = progress.discovery_type.charAt(0).toUpperCase() + progress.discovery_type.slice(1);

  return (
    <div className="p-4 rounded-lg border" style={cardStyle}>
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          {statusIcon}
          <span className="font-semibold" style={{ color: 'var(--fgColor-default)' }}>{statusText}</span>
          <Label variant="accent">{typeLabel}</Label>
        </div>
        {isComplete && onDismiss && (
          <button
            onClick={onDismiss}
            className="p-1 rounded hover:bg-black/10 transition-colors"
            aria-label="Dismiss"
            style={{ color: 'var(--fgColor-muted)' }}
          >
            <XIcon size={16} />
          </button>
        )}
      </div>
      
      {isInProgress && (
        <>
          <div className="mb-2">
            <ProgressBar 
              progress={percentage} 
              aria-label={`${progress.processed_repos} of ${progress.total_repos} repositories processed`}
              sx={{
                backgroundColor: 'var(--bgColor-neutral-muted, var(--color-neutral-muted))',
              }}
            />
          </div>
          
          <div className="flex justify-between text-sm mb-1">
            <span style={{ color: 'var(--fgColor-muted)' }}>
              {progress.total_orgs > 1 
                ? `Org ${progress.processed_orgs + 1} of ${progress.total_orgs}: ${progress.current_org}`
                : `Processing: ${progress.current_org || progress.target}`
              }
            </span>
            <span style={{ color: 'var(--fgColor-muted)' }}>
              {progress.processed_repos} / {progress.total_repos} repos
            </span>
          </div>
          
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            {phaseLabels[progress.phase] || progress.phase}
          </p>
        </>
      )}

      {isComplete && (
        <div className="text-sm">
          <p style={{ color: 'var(--fgColor-muted)' }}>
            Discovered {progress.processed_repos} repositories across {progress.processed_orgs} organization{progress.processed_orgs !== 1 ? 's' : ''}
          </p>
          {progress.completed_at && (
            <p className="mt-1" style={{ color: 'var(--fgColor-muted)' }}>
              Completed {new Date(progress.completed_at).toLocaleString()}
            </p>
          )}
        </div>
      )}

      {isFailed && (
        <div className="text-sm">
          <p style={{ color: 'var(--fgColor-muted)' }}>
            Processed {progress.processed_repos} of {progress.total_repos} repositories before failure
          </p>
          {progress.last_error && (
            <div className="flex items-start gap-1 mt-2">
              <AlertIcon className="flex-shrink-0 mt-0.5" style={{ color: 'var(--fgColor-danger)' }} size={14} />
              <p style={{ color: 'var(--fgColor-danger)' }}>
                {progress.last_error}
              </p>
            </div>
          )}
        </div>
      )}
      
      {progress.error_count > 0 && isInProgress && (
        <div className="flex items-center gap-1 mt-2">
          <AlertIcon style={{ color: 'var(--fgColor-attention)' }} size={14} />
          <span className="text-sm" style={{ color: 'var(--fgColor-attention)' }}>
            {progress.error_count} error{progress.error_count !== 1 ? 's' : ''} encountered
          </span>
        </div>
      )}
    </div>
  );
}

interface LastDiscoveryIndicatorProps {
  progress: DiscoveryProgress;
  onExpand?: () => void;
}

export function LastDiscoveryIndicator({ progress, onExpand }: LastDiscoveryIndicatorProps) {
  const completedDate = progress.completed_at 
    ? new Date(progress.completed_at) 
    : null;
  
  const formatRelativeTime = (date: Date): string => {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);
    
    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays === 1) return 'yesterday';
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <div 
      className="flex items-center gap-2 text-sm cursor-pointer hover:opacity-80 transition-opacity"
      onClick={onExpand}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onExpand?.()}
      title="Click to show details"
    >
      <CheckCircleIcon size={14} style={{ color: 'var(--fgColor-success)' }} />
      <span style={{ color: 'var(--fgColor-muted)' }}>
        Last discovery: {progress.processed_repos} repos
        {completedDate && ` · ${formatRelativeTime(completedDate)}`}
      </span>
    </div>
  );
}
