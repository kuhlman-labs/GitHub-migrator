import { Box, Text, ProgressBar, Label } from '@primer/react';
import { SyncIcon, CheckCircleIcon, AlertIcon, XCircleIcon } from '@primer/octicons-react';
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
}

export function DiscoveryProgressCard({ progress }: DiscoveryProgressCardProps) {
  const percentage = progress.total_repos > 0 
    ? Math.round((progress.processed_repos / progress.total_repos) * 100) 
    : 0;

  const isComplete = progress.status === 'completed';
  const isFailed = progress.status === 'failed';
  const isInProgress = progress.status === 'in_progress';

  // Determine card styling based on status
  const borderColor = isFailed 
    ? 'border-red-300' 
    : isComplete 
      ? 'border-green-300' 
      : 'border-blue-300';
  const bgColor = isFailed 
    ? 'bg-red-50' 
    : isComplete 
      ? 'bg-green-50' 
      : 'bg-blue-50';

  // Get the icon based on status
  const StatusIcon = () => {
    if (isFailed) {
      return <XCircleIcon className="text-red-600" size={16} />;
    }
    if (isComplete) {
      return <CheckCircleIcon className="text-green-600" size={16} />;
    }
    return <SyncIcon className="animate-spin text-blue-600" size={16} />;
  };

  // Get status text
  const statusText = isFailed 
    ? 'Discovery Failed' 
    : isComplete 
      ? 'Discovery Complete' 
      : 'Discovery in Progress';

  // Format the discovery type for display
  const typeLabel = progress.discovery_type.charAt(0).toUpperCase() + progress.discovery_type.slice(1);

  return (
    <Box className={`p-4 rounded-lg border ${borderColor} ${bgColor}`}>
      <div className="flex items-center gap-2 mb-3">
        <StatusIcon />
        <Text as="span" sx={{ fontWeight: 'semibold' }}>{statusText}</Text>
        <Label variant="accent">{typeLabel}</Label>
      </div>
      
      {isInProgress && (
        <>
          <ProgressBar progress={percentage} sx={{ mb: 2 }} />
          
          <div className="flex justify-between text-sm mb-1">
            <Text as="span" sx={{ color: 'fg.muted' }}>
              {progress.total_orgs > 1 
                ? `Org ${progress.processed_orgs + 1} of ${progress.total_orgs}: ${progress.current_org}`
                : `Processing: ${progress.current_org || progress.target}`
              }
            </Text>
            <Text as="span" sx={{ color: 'fg.muted' }}>
              {progress.processed_repos} / {progress.total_repos} repos
            </Text>
          </div>
          
          <Text as="p" sx={{ fontSize: 1, color: 'fg.muted', mt: 1 }}>
            {phaseLabels[progress.phase] || progress.phase}
          </Text>
        </>
      )}

      {isComplete && (
        <div className="text-sm">
          <Text as="p" sx={{ color: 'fg.muted' }}>
            Discovered {progress.processed_repos} repositories across {progress.processed_orgs} organization{progress.processed_orgs !== 1 ? 's' : ''}
          </Text>
          {progress.completed_at && (
            <Text as="p" sx={{ color: 'fg.muted', mt: 1 }}>
              Completed {new Date(progress.completed_at).toLocaleString()}
            </Text>
          )}
        </div>
      )}

      {isFailed && (
        <div className="text-sm">
          <Text as="p" sx={{ color: 'fg.muted' }}>
            Processed {progress.processed_repos} of {progress.total_repos} repositories before failure
          </Text>
          {progress.last_error && (
            <div className="flex items-start gap-1 mt-2">
              <AlertIcon className="text-red-500 flex-shrink-0 mt-0.5" size={14} />
              <Text as="p" sx={{ color: 'danger.fg', fontSize: 1 }}>
                {progress.last_error}
              </Text>
            </div>
          )}
        </div>
      )}
      
      {progress.error_count > 0 && isInProgress && (
        <div className="flex items-center gap-1 mt-2">
          <AlertIcon className="text-orange-500" size={14} />
          <Text as="span" sx={{ fontSize: 1, color: 'attention.fg' }}>
            {progress.error_count} error{progress.error_count !== 1 ? 's' : ''} encountered
          </Text>
        </div>
      )}
    </Box>
  );
}

