import { ActionMenu, ActionList, Button, Spinner, Text, Flash } from '@primer/react';
import { 
  TriangleDownIcon, 
  MarkGithubIcon, 
  GlobeIcon,
  ShieldLockIcon,
  AlertIcon,
} from '@primer/octicons-react';
import { useSourceContext } from '../../contexts/SourceContext';
import { useAuth } from '../../contexts/AuthContext';
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
 * 
 * It also shows authentication status for sources with OAuth configured.
 */
export function SourceSelector() {
  const { 
    sources, 
    activeSourceFilter, 
    setActiveSourceFilter, 
    activeSource,
    isLoading,
  } = useSourceContext();

  const { authenticatedSourceId, login, authEnabled } = useAuth();

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

  // Check if user is authenticated for the active source
  const needsAuth = activeSource && 
    activeSource.has_oauth && 
    authEnabled &&
    authenticatedSourceId !== activeSource.id;

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

    // If this source has OAuth and user isn't authenticated for it, prompt login
    if (source.has_oauth && authEnabled && authenticatedSourceId !== sourceId) {
      // Could show a confirmation dialog here, for now just set the filter
      // The UI will show a warning banner
    }
  };

  const handleReauth = () => {
    if (activeSource) {
      login(activeSource.id);
    }
  };

  return (
    <div className="flex items-center gap-2">
      <ActionMenu>
        <ActionMenu.Button
          variant="invisible"
          size="small"
          className="flex items-center gap-1 font-medium"
          style={{ color: 'var(--fgColor-default)' }}
        >
          {activeSource ? (
            <SourceIcon type={activeSource.type} />
          ) : (
            <GlobeIcon />
          )}
          <span className="hidden sm:inline">{buttonLabel}</span>
          {needsAuth && (
            <AlertIcon size={12} className="text-attention-fg" />
          )}
          <TriangleDownIcon size={12} />
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

            {sources.map((source) => {
              const isAuthenticatedForSource = authenticatedSourceId === source.id;
              const showAuthBadge = source.has_oauth && authEnabled;

              return (
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
                    <span className="flex items-center gap-2">
                      {showAuthBadge && (
                        <ShieldLockIcon 
                          size={12} 
                          className={isAuthenticatedForSource ? 'text-success-fg' : 'text-muted'}
                        />
                      )}
                      <span className="text-xs text-muted">
                        {source.repository_count} repos
                      </span>
                    </span>
                  </ActionList.TrailingVisual>
                </ActionList.Item>
              );
            })}
          </ActionList>
        </ActionMenu.Overlay>
      </ActionMenu>

      {/* Auth warning indicator */}
      {needsAuth && (
        <Button
          variant="invisible"
          size="small"
          onClick={handleReauth}
          className="flex items-center gap-1"
          style={{ color: 'var(--fgColor-attention)' }}
        >
          <AlertIcon size={12} />
          <span className="hidden md:inline text-xs">Authenticate</span>
        </Button>
      )}
    </div>
  );
}

/**
 * Banner component to show when user needs to re-authenticate for the active source.
 * Use this in pages where source-scoped actions are performed.
 */
export function SourceAuthBanner() {
  const { activeSource } = useSourceContext();
  const { authenticatedSourceId, login, authEnabled } = useAuth();

  if (!activeSource || !activeSource.has_oauth || !authEnabled) {
    return null;
  }

  if (authenticatedSourceId === activeSource.id) {
    return null;
  }

  return (
    <Flash variant="warning" className="mb-4">
      <div className="flex items-center justify-between">
        <div>
          <Text className="font-semibold">Authentication Required</Text>
          <Text as="p" className="text-sm mt-1">
            To perform actions on repositories from "{activeSource.name}", 
            you need to authenticate with this source.
          </Text>
        </div>
        <Button onClick={() => login(activeSource.id)}>
          Authenticate with {activeSource.name}
        </Button>
      </div>
    </Flash>
  );
}
