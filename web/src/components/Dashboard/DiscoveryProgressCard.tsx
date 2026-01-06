import { ProgressBar, Label } from '@primer/react';
import { SyncIcon, CheckCircleIcon, AlertIcon, XCircleIcon, XIcon } from '@primer/octicons-react';
import type { DiscoveryProgress, DiscoveryPhase } from '../../types';

const phaseLabels: Record<DiscoveryPhase, string> = {
  listing_repos: 'Listing repositories...',
  profiling_repos: 'Profiling repositories...',
  discovering_teams: 'Discovering teams...',
  discovering_members: 'Discovering members...',
  waiting_for_rate_limit: 'Waiting for rate limit reset...',
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
  const isWaitingForRateLimit = progress.phase === 'waiting_for_rate_limit';

  // Determine card styling based on status using CSS variables for dark mode compatibility
  const cardStyle = {
    borderColor: isFailed 
      ? 'var(--borderColor-danger-muted, var(--color-danger-muted))' 
      : isComplete 
        ? 'var(--borderColor-success-muted, var(--color-success-muted))' 
        : isWaitingForRateLimit
          ? 'var(--borderColor-attention-muted, var(--color-attention-muted))'
          : 'var(--borderColor-accent-muted, var(--color-accent-muted))',
    backgroundColor: isFailed 
      ? 'var(--bgColor-danger-muted, var(--color-danger-subtle))' 
      : isComplete 
        ? 'var(--bgColor-success-muted, var(--color-success-subtle))' 
        : isWaitingForRateLimit
          ? 'var(--bgColor-attention-muted, var(--color-attention-subtle))'
          : 'var(--bgColor-accent-muted, var(--color-accent-subtle))',
  };

  // Get the icon based on status (wrapped in span for color styling)
  const statusIcon = isFailed 
    ? <span style={{ color: 'var(--fgColor-danger)' }}><XCircleIcon size={16} /></span>
    : isComplete 
      ? <span style={{ color: 'var(--fgColor-success)' }}><CheckCircleIcon size={16} /></span>
      : isWaitingForRateLimit
        ? <span style={{ color: 'var(--fgColor-attention)' }}><AlertIcon size={16} /></span>
        : <span className="animate-spin" style={{ color: 'var(--fgColor-accent)', display: 'inline-flex' }}><SyncIcon size={16} /></span>;

  // Get status text
  const statusText = isFailed 
    ? 'Discovery Failed' 
    : isComplete 
      ? 'Discovery Complete' 
      : isWaitingForRateLimit
        ? 'Rate Limited - Waiting'
        : 'Discovery in Progress';

  // Format the discovery type for display
  const formatDiscoveryType = (type: string): string => {
    switch (type) {
      case 'enterprise':
        return 'Enterprise';
      case 'organization':
        return 'Organization';
      case 'repository':
        return 'Repository';
      case 'ado_organization':
        return 'ADO Organization';
      case 'ado_project':
        return 'ADO Project';
      default:
        return type.charAt(0).toUpperCase() + type.slice(1);
    }
  };
  const typeLabel = formatDiscoveryType(progress.discovery_type);

  // Determine if this is an ADO discovery (use "projects" terminology)
  const isADODiscovery = progress.discovery_type.startsWith('ado_');
  const orgLabel = isADODiscovery ? 'project' : 'org';
  const orgsLabel = isADODiscovery ? 'projects' : 'organizations';

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
              bg="accent.emphasis"
            />
          </div>
          
          <div className="flex justify-between text-sm mb-1">
            <span style={{ color: 'var(--fgColor-muted)' }}>
              {/* Show org/project progress only when current_org is set */}
              {progress.current_org 
                ? (progress.total_orgs > 1 
                    ? `${isADODiscovery ? 'Project' : 'Org'} ${progress.processed_orgs + 1} of ${progress.total_orgs}: ${progress.current_org}`
                    : `Processing: ${progress.current_org}`)
                : (progress.total_orgs > 1 
                    ? `${progress.total_orgs} ${orgsLabel}`
                    : `Processing: ${progress.target}`)
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
            Discovered {progress.processed_repos} repositories across {progress.processed_orgs} {progress.processed_orgs !== 1 ? orgsLabel : orgLabel}
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
              <span className="flex-shrink-0 mt-0.5" style={{ color: 'var(--fgColor-danger)' }}>
                <AlertIcon size={14} />
              </span>
              <p style={{ color: 'var(--fgColor-danger)' }}>
                {progress.last_error}
              </p>
            </div>
          )}
        </div>
      )}
      
      {progress.error_count > 0 && isInProgress && (
        <div className="flex items-center gap-1 mt-2">
          <span style={{ color: 'var(--fgColor-attention)' }}>
            <AlertIcon size={14} />
          </span>
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
      <span style={{ color: 'var(--fgColor-success)' }}>
        <CheckCircleIcon size={14} />
      </span>
      <span style={{ color: 'var(--fgColor-muted)' }}>
        Last discovery: {progress.processed_repos} repos
        {completedDate && ` · ${formatRelativeTime(completedDate)}`}
      </span>
    </div>
  );
}
