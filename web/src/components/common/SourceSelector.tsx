import { ActionMenu, ActionList, Button, Spinner } from '@primer/react';
import { 
  MarkGithubIcon, 
  GlobeIcon,
} from '@primer/octicons-react';
import { useSourceContext } from '../../contexts/SourceContext';
import type { Source } from '../../types';

/**
 * Azure DevOps icon component using the official SVG
 */
function AzureDevOpsIcon({ size = 16 }: { size?: number }) {
  return (
    <svg 
      width={size} 
      height={size} 
      viewBox="0 0 18 18" 
      xmlns="http://www.w3.org/2000/svg"
      style={{ display: 'inline-block', verticalAlign: 'text-bottom' }}
    >
      <defs>
        <linearGradient id="ado-gradient-selector" x1="9" y1="16.97" x2="9" y2="1.03" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#0078d4"/>
          <stop offset="0.16" stopColor="#1380da"/>
          <stop offset="0.53" stopColor="#3c91e5"/>
          <stop offset="0.82" stopColor="#559cec"/>
          <stop offset="1" stopColor="#5ea0ef"/>
        </linearGradient>
      </defs>
      <path 
        d="M17,4v9.74l-4,3.28-6.2-2.26V17L3.29,12.41l10.23.8V4.44Zm-3.41.49L7.85,1V3.29L2.58,4.84,1,6.87v4.61l2.26,1V6.57Z" 
        fill="url(#ado-gradient-selector)"
      />
    </svg>
  );
}

/**
 * Get the appropriate icon for a source type.
 */
function SourceIcon({ type }: { type: Source['type'] }) {
  return type === 'github' ? <MarkGithubIcon /> : <AzureDevOpsIcon />;
}

/**
 * SourceSelector is a dropdown in the navigation header that allows users
 * to filter the view by a specific source or see all sources.
 */
export function SourceSelector() {
  const { 
    sources, 
    activeSourceFilter, 
    setActiveSourceFilter, 
    activeSource,
    isLoading,
  } = useSourceContext();

  if (isLoading) {
    return (
      <Button disabled variant="invisible" size="small">
        <Spinner size="small" />
      </Button>
    );
  }

  // Don't show selector if there are no sources or only one source
  if (sources.length <= 1) {
    return null;
  }

  const buttonLabel = activeSource 
    ? activeSource.name 
    : 'All Sources';

  const handleSourceSelect = (sourceId: number | 'all') => {
    if (sourceId === 'all') {
      setActiveSourceFilter('all');
      return;
    }

    const source = sources.find(s => s.id === sourceId);
    if (!source) return;

    setActiveSourceFilter(sourceId);
  };

  return (
    <div className="flex items-center gap-2">
      <ActionMenu>
        <ActionMenu.Button
          variant="invisible"
          size="small"
          className="font-medium"
          style={{ color: 'var(--fgColor-default)' }}
        >
          <span className="flex items-center gap-1.5">
            {activeSource ? (
              <SourceIcon type={activeSource.type} />
            ) : (
              <GlobeIcon size={16} />
            )}
            <span className="hidden sm:inline">{buttonLabel}</span>
          </span>
        </ActionMenu.Button>

        <ActionMenu.Overlay width="medium">
          <ActionList selectionVariant="single">
            <ActionList.Item
              selected={activeSourceFilter === 'all'}
              onSelect={() => handleSourceSelect('all')}
            >
              <ActionList.LeadingVisual>
                <GlobeIcon />
              </ActionList.LeadingVisual>
              All Sources
              <ActionList.TrailingVisual>
                <span className="text-xs text-muted">
                  {sources.reduce((sum, s) => sum + s.repository_count, 0)} repos
                </span>
              </ActionList.TrailingVisual>
            </ActionList.Item>

            <ActionList.Divider />

            {sources.map((source) => (
              <ActionList.Item
                key={source.id}
                selected={activeSourceFilter === source.id}
                onSelect={() => handleSourceSelect(source.id)}
                disabled={!source.is_active}
              >
                <ActionList.LeadingVisual>
                  <SourceIcon type={source.type} />
                </ActionList.LeadingVisual>
                {source.name}
                {!source.is_active && (
                  <ActionList.Description variant="inline">
                    (inactive)
                  </ActionList.Description>
                )}
                <ActionList.TrailingVisual>
                  <span className="text-xs text-muted">
                    {source.repository_count} repos
                  </span>
                </ActionList.TrailingVisual>
              </ActionList.Item>
            ))}
          </ActionList>
        </ActionMenu.Overlay>
      </ActionMenu>
    </div>
  );
}
