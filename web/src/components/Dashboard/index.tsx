import { useState, useMemo, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { PrimaryButton } from '../common/buttons';
import { useToast } from '../../contexts/ToastContext';
import { Blankslate } from '@primer/react/experimental';
import { Flash } from '@primer/react';
import { RepoIcon, TelescopeIcon } from '@primer/octicons-react';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Pagination } from '../common/Pagination';
import { useOrganizations, useAnalytics, useBatches, useDashboardActionItems, useDiscoveryProgress, useConfig, useSetupProgress } from '../../hooks/useQueries';
import { useStartDiscovery, useStartADODiscovery } from '../../hooks/useMutations';
import { useSourceContext } from '../../contexts/SourceContext';
import { KPISection } from './KPISection';
import { ActionItemsPanel } from './ActionItemsPanel';
import { GitHubOrganizationCard } from './GitHubOrganizationCard';
import { ADOOrganizationCard } from './ADOOrganizationCard';
import { UpcomingBatchesTimeline } from './UpcomingBatchesTimeline';
import { DiscoveryProgressCard, LastDiscoveryIndicator } from './DiscoveryProgressCard';
import { DiscoveryModal, type DiscoveryType } from './DiscoveryModal';
import { SetupProgress } from './SetupProgress';

// Polling intervals based on activity level
const POLLING_INTERVALS = {
  actionItems: 15000, // 15s - critical for admin attention
  orgsIdle: 60000, // 1min when idle
  orgsDiscovery: 5000, // 5s during discovery for real-time updates
  analyticsActive: 30000, // 30s when migrations active
  analyticsIdle: 120000, // 2min when idle
  analyticsDiscovery: 5000, // 5s during discovery for real-time updates
  batchesActive: 60000, // 1min when migrations active
  batchesIdle: 300000, // 5min when idle
} as const;

export function Dashboard() {
  // Use React Query for config
  const { data: config } = useConfig();
  const { showSuccess } = useToast();
  const { activeSource, sources } = useSourceContext();
  
  // Derive source types from configured sources
  // If a specific source is selected, use its type; otherwise look at all sources
  const hasGitHubSources = sources.some(s => s.type === 'github');
  const hasADOSources = sources.some(s => s.type === 'azuredevops');
  
  // Determine the primary source type for display
  // If a specific source is active, use its type
  // Otherwise, if we have mixed sources, default to github; if only ADO, use azuredevops
  const sourceType = activeSource?.type 
    || (hasADOSources && !hasGitHubSources ? 'azuredevops' : 'github')
    || config?.source_type 
    || 'github';
  
  // Fetch setup progress for guided empty states
  const { data: setupProgress } = useSetupProgress();
  
  // Track if there are active migrations to adjust polling intervals
  const [hasActiveMigrations, setHasActiveMigrations] = useState(false);
  
  // Fetch discovery progress first to determine polling intervals
  const { data: discoveryProgress } = useDiscoveryProgress();
  const isDiscoveryInProgress = discoveryProgress?.status === 'in_progress';
  
  // Fetch all dashboard data with React Query polling
  // Use faster polling when discovery is in progress to show real-time updates
  // Filter by active source when one is selected
  const { data: organizations = [], isLoading: orgsLoading, isFetching: orgsFetching } = useOrganizations({
    sourceId: activeSource?.id,
    refetchInterval: isDiscoveryInProgress 
      ? POLLING_INTERVALS.orgsDiscovery 
      : POLLING_INTERVALS.orgsIdle,
  });
  const { data: analytics, isLoading: analyticsLoading, isFetching: analyticsFetching } = useAnalytics(
    { source_id: activeSource?.id }, 
    {
      refetchInterval: isDiscoveryInProgress
        ? POLLING_INTERVALS.analyticsDiscovery
        : hasActiveMigrations 
          ? POLLING_INTERVALS.analyticsActive 
          : POLLING_INTERVALS.analyticsIdle,
    }
  );
  
  // Update active migrations state when analytics changes
  // This is the standard React pattern for syncing state with derived values
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setHasActiveMigrations(analytics?.in_progress_count ? analytics.in_progress_count > 0 : false);
  }, [analytics?.in_progress_count]);
  
  const { data: batches = [], isLoading: batchesLoading, isFetching: batchesFetching } = useBatches({
    refetchInterval: hasActiveMigrations ? POLLING_INTERVALS.batchesActive : POLLING_INTERVALS.batchesIdle,
  });
  const { data: actionItems, isLoading: actionItemsLoading, isFetching: actionItemsFetching } = useDashboardActionItems({
    refetchInterval: POLLING_INTERVALS.actionItems,
  });
  
  const startDiscoveryMutation = useStartDiscovery();
  const startADODiscoveryMutation = useStartADODiscovery();
  const [searchParams] = useSearchParams();
  
  const searchTerm = searchParams.get('search') || '';
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 12;
  
  // Reset page when search changes - standard React pattern for prop-dependent state
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setCurrentPage(1);
  }, [searchTerm]);
  
  const [showDiscoveryModal, setShowDiscoveryModal] = useState(false);
  // Initialize discoveryType lazily - will be set to defaultDiscoveryType when modal opens
  const [discoveryType, setDiscoveryType] = useState<DiscoveryType | null>(null);
  const [organization, setOrganization] = useState('');
  const [enterpriseSlug, setEnterpriseSlug] = useState('');
  const [adoOrganization, setAdoOrganization] = useState('');
  const [adoProject, setAdoProject] = useState('');
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);

  // Persist dismissed state in localStorage, keyed by discovery ID
  const dismissedDiscoveryKey = 'dismissedDiscoveryId';
  const currentDiscoveryId = discoveryProgress?.id;
  
  // Version counter to trigger localStorage re-reads
  const [localStorageVersion, setLocalStorageVersion] = useState(0);
  
  // Derive dismissed state from localStorage (no setState in effect)
  const discoveryBannerDismissed = useMemo(() => {
    // Include localStorageVersion in closure to trigger re-computation
    void localStorageVersion;
    if (!currentDiscoveryId) return false;
    if (discoveryProgress?.status === 'in_progress') return false;
    const dismissedId = localStorage.getItem(dismissedDiscoveryKey);
    return dismissedId === String(currentDiscoveryId);
  }, [currentDiscoveryId, discoveryProgress?.status, localStorageVersion]);
  
  // Clear localStorage when discovery starts (side effect only, no setState)
  useEffect(() => {
    if (discoveryProgress?.status === 'in_progress') {
      localStorage.removeItem(dismissedDiscoveryKey);
    }
  }, [discoveryProgress?.status]);

  const handleDismissDiscoveryBanner = () => {
    if (currentDiscoveryId) {
      localStorage.setItem(dismissedDiscoveryKey, String(currentDiscoveryId));
      setLocalStorageVersion(v => v + 1);
    }
  };

  const handleExpandDiscoveryBanner = () => {
    localStorage.removeItem(dismissedDiscoveryKey);
    setLocalStorageVersion(v => v + 1);
  };

  // Compute default discovery type based on configured sources
  const defaultDiscoveryType = useMemo<DiscoveryType>(() => 
    sourceType === 'azuredevops' ? 'ado-org' : 'organization',
    [sourceType]
  );
  


  const handleStartDiscovery = async () => {
    // Use the effective discovery type (fallback to default if null)
    const effectiveDiscoveryType = discoveryType ?? defaultDiscoveryType;
    
    // Validate input based on discovery type
    if (effectiveDiscoveryType === 'organization' && !organization.trim()) {
      setDiscoveryError('Organization name is required');
      return;
    }
    
    if (effectiveDiscoveryType === 'enterprise' && !enterpriseSlug.trim()) {
      setDiscoveryError('Enterprise slug is required');
      return;
    }

    if (effectiveDiscoveryType === 'ado-org' && !adoOrganization.trim()) {
      setDiscoveryError('Azure DevOps organization name is required');
      return;
    }

    if (effectiveDiscoveryType === 'ado-project' && (!adoOrganization.trim() || !adoProject.trim())) {
      setDiscoveryError('Both Azure DevOps organization and project names are required');
      return;
    }

    setDiscoveryError(null);

    try {
      // Determine which source to use for discovery
      // Priority: active source > single configured source > undefined (will fail if no legacy config)
      const gitHubSources = sources.filter(s => s.type === 'github');
      const adoSources = sources.filter(s => s.type === 'azuredevops');
      
      let sourceId: number | undefined;
      if (activeSource?.id) {
        sourceId = activeSource.id;
      } else if (effectiveDiscoveryType === 'organization' || effectiveDiscoveryType === 'enterprise') {
        // For GitHub discovery, use the single GitHub source if only one exists
        if (gitHubSources.length === 1) {
          sourceId = gitHubSources[0].id;
        } else if (gitHubSources.length > 1) {
          setDiscoveryError('Multiple GitHub sources configured. Please select a source from the dropdown.');
          return;
        }
      } else if (effectiveDiscoveryType === 'ado-org' || effectiveDiscoveryType === 'ado-project') {
        // For ADO discovery, use the single ADO source if only one exists
        if (adoSources.length === 1) {
          sourceId = adoSources[0].id;
        } else if (adoSources.length > 1) {
          setDiscoveryError('Multiple Azure DevOps sources configured. Please select a source from the dropdown.');
          return;
        }
      }
      
      if (effectiveDiscoveryType === 'enterprise') {
        await startDiscoveryMutation.mutateAsync({ 
          enterprise_slug: enterpriseSlug.trim(),
          source_id: sourceId
        });
        showSuccess(`Enterprise discovery started for ${enterpriseSlug}`);
        setEnterpriseSlug('');
      } else if (effectiveDiscoveryType === 'ado-org') {
        await startADODiscoveryMutation.mutateAsync({ 
          organization: adoOrganization.trim(),
          source_id: sourceId,
        });
        showSuccess(`ADO organization discovery started for ${adoOrganization}`);
        setAdoOrganization('');
      } else if (effectiveDiscoveryType === 'ado-project') {
        await startADODiscoveryMutation.mutateAsync({ 
          organization: adoOrganization.trim(), 
          project: adoProject.trim(),
          source_id: sourceId,
        });
        showSuccess(`ADO project discovery started for ${adoOrganization}/${adoProject}`);
        setAdoOrganization('');
        setAdoProject('');
      } else {
        await startDiscoveryMutation.mutateAsync({ 
          organization: organization.trim(),
          source_id: sourceId
        });
        showSuccess(`Discovery started for ${organization}`);
        setOrganization('');
      }
      
      setShowDiscoveryModal(false);
    } catch (error) {
      setDiscoveryError(error instanceof Error ? error.message : 'Failed to start discovery');
    }
  };

  const filteredOrgs = organizations.filter(org =>
    org.organization.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const totalRepos = organizations.reduce((sum, org) => sum + org.total_repos, 0);

  // Paginate
  const totalItems = filteredOrgs.length;
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const paginatedOrgs = filteredOrgs.slice(startIndex, endIndex);

  const isLoading = orgsLoading || analyticsLoading || batchesLoading || actionItemsLoading;
  const isFetching = orgsFetching || analyticsFetching || batchesFetching || actionItemsFetching;

  // Separate organizations by source type for "All Sources" view
  const gitHubOrgs = useMemo(() => 
    filteredOrgs.filter(org => !org.ado_organization), 
    [filteredOrgs]
  );
  
  const adoOrgs = useMemo(() => 
    filteredOrgs.filter(org => !!org.ado_organization),
    [filteredOrgs]
  );
  
  // Group ADO projects by organization
  const groupedADOOrgs = useMemo(() => 
    adoOrgs.reduce((acc, org) => {
        const adoOrgName = org.ado_organization || 'Unknown';
        if (!acc[adoOrgName]) {
          acc[adoOrgName] = [];
        }
        acc[adoOrgName].push(org);
        return acc;
    }, {} as Record<string, typeof adoOrgs>),
    [adoOrgs]
  );

  // Aggregate ADO orgs for "All Sources" view (simple org cards like GitHub)
  const aggregatedADOOrgs = useMemo(() => 
    Object.entries(groupedADOOrgs).map(([adoOrgName, projects]) => {
      // Merge status_counts from all projects
      const mergedStatusCounts: Record<string, number> = {};
      projects.forEach(p => {
        if (p.status_counts) {
          Object.entries(p.status_counts).forEach(([status, count]) => {
            mergedStatusCounts[status] = (mergedStatusCounts[status] || 0) + count;
          });
        }
      });
      
      const totalRepos = projects.reduce((sum, p) => sum + p.total_repos, 0);
      const migrated = projects.reduce((sum, p) => sum + p.migrated_count, 0);
      
      // Get source info from first project (all projects in same ADO org should be from same source)
      const firstProject = projects[0];
      
      return {
        organization: adoOrgName,
        ado_organization: adoOrgName,
        total_repos: totalRepos,
        status_counts: mergedStatusCounts,
        migrated_count: migrated,
        in_progress_count: projects.reduce((sum, p) => sum + p.in_progress_count, 0),
        failed_count: projects.reduce((sum, p) => sum + p.failed_count, 0),
        pending_count: projects.reduce((sum, p) => sum + p.pending_count, 0),
        migration_progress_percentage: totalRepos > 0 ? Math.floor((migrated * 100) / totalRepos) : 0,
        // Include source info from the projects
        source_id: firstProject?.source_id,
        source_name: firstProject?.source_name,
        source_type: firstProject?.source_type || 'azuredevops' as const,
      };
    }),
    [groupedADOOrgs]
  );
  
  // Determine what to show based on source filter and available data
  const showAllSources = activeSource === null;
  const showGitHubSection = showAllSources ? gitHubOrgs.length > 0 : sourceType === 'github';
  const showADOSection = showAllSources ? adoOrgs.length > 0 : sourceType === 'azuredevops';

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      <div className="flex justify-between items-start mb-8">
        <div>
          <h1 className="text-2xl font-semibold" style={{ color: 'var(--fgColor-default)' }}>
            Dashboard
          </h1>
          <p className="text-sm mt-1" style={{ color: 'var(--fgColor-muted)' }}>
            Overview of migration progress across all organizations
          </p>
        </div>
        <div className="flex items-center gap-4">
          <PrimaryButton
            leadingVisual={TelescopeIcon}
            onClick={() => {
              setDiscoveryType(defaultDiscoveryType);
              setShowDiscoveryModal(true);
            }}
          >
            Start Discovery
          </PrimaryButton>
        </div>
      </div>


      {/* Setup Progress - guided empty states for initial configuration */}
      {setupProgress && !setupProgress.setup_complete && (
        <SetupProgress
          destinationConfigured={setupProgress.destination_configured}
          sourcesConfigured={setupProgress.sources_configured}
          sourceCount={setupProgress.source_count}
          batchesCreated={setupProgress.batches_created}
          batchCount={setupProgress.batch_count}
        />
      )}
      
      {/* Active Source Filter Indicator */}
      {activeSource && (
        <div 
          className="mb-4 px-4 py-3 rounded-md border text-sm"
          style={{
            backgroundColor: 'transparent',
            borderColor: 'var(--borderColor-default)',
            color: 'var(--fgColor-muted)',
          }}
        >
          Showing data from: <strong style={{ color: 'var(--fgColor-default)' }}>{activeSource.name}</strong> ({activeSource.type === 'github' ? 'GitHub' : 'Azure DevOps'})
        </div>
      )}

      {/* Discovery Progress Card - shown when discovery is active or recently completed */}
      {discoveryProgress && (
        <div className="mb-4">
          {discoveryProgress.status === 'completed' && discoveryBannerDismissed ? (
            <LastDiscoveryIndicator 
              progress={discoveryProgress} 
              onExpand={handleExpandDiscoveryBanner}
            />
          ) : (
            <DiscoveryProgressCard 
              progress={discoveryProgress} 
              onDismiss={handleDismissDiscoveryBanner}
            />
          )}
        </div>
      )}

      {/* KPI Section */}
      <KPISection analytics={analytics} isLoading={analyticsLoading} />

      {/* Action Items Panel */}
      <ActionItemsPanel actionItems={actionItems} isLoading={actionItemsLoading} />

      {/* Upcoming Batches Timeline */}
      <UpcomingBatchesTimeline batches={batches} isLoading={batchesLoading} />

      {/* Organizations Section - only show when sources are configured */}
      {setupProgress?.sources_configured && (
        <>
        {orgsLoading ? (
            <div className="mb-6">
          <LoadingSpinner />
            </div>
        ) : filteredOrgs.length === 0 ? (
            <div className="mb-6">
          <Blankslate>
            <Blankslate.Visual>
              <RepoIcon size={48} />
            </Blankslate.Visual>
            <Blankslate.Heading>
              {sourceType === 'azuredevops' ? 'No Azure DevOps organizations discovered yet' : 'No organizations discovered yet'}
            </Blankslate.Heading>
            <Blankslate.Description>
              {searchTerm 
                ? 'No organizations match your search. Try a different search term.'
                : sourceType === 'azuredevops'
                  ? 'Get started by discovering repositories from your Azure DevOps organizations and projects.'
                  : 'Get started by discovering repositories from your GitHub organizations.'}
            </Blankslate.Description>
            {!searchTerm && (
              <Blankslate.PrimaryAction onClick={() => {
                setDiscoveryType(defaultDiscoveryType);
                setShowDiscoveryModal(true);
              }}>
                Start Discovery
              </Blankslate.PrimaryAction>
            )}
          </Blankslate>
          </div>
        ) : (
            <>
              {/* GitHub Organizations Section */}
              {showGitHubSection && (
                <div className="mb-6">
                  <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--fgColor-default)' }}>
                    GitHub Organizations
                  </h2>
                  <div className="mb-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                    {gitHubOrgs.length > 0 ? (
                      <>
                        {showAllSources ? (
                          <>Showing {gitHubOrgs.length} {gitHubOrgs.length === 1 ? 'organization' : 'organizations'} with {gitHubOrgs.reduce((sum, org) => sum + org.total_repos, 0)} total repositories</>
                        ) : (
                          <>Showing {startIndex + 1}-{Math.min(endIndex, totalItems)} of {totalItems} {totalItems === 1 ? 'organization' : 'organizations'} with {totalRepos} total repositories</>
                        )}
                      </>
                    ) : (
                      'No GitHub organizations found'
                    )}
                  </div>
                  {gitHubOrgs.length === 0 ? (
                    <Flash>No GitHub organizations discovered yet. Start discovery to find repositories.</Flash>
                  ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
                        {(showAllSources ? gitHubOrgs : paginatedOrgs).map((org) => (
                <GitHubOrganizationCard key={org.organization} organization={org} />
              ))}
            </div>
                      {!showAllSources && totalItems > pageSize && (
              <Pagination
                currentPage={currentPage}
                totalItems={totalItems}
                pageSize={pageSize}
                onPageChange={setCurrentPage}
              />
            )}
          </>
        )}
      </div>
              )}

              {/* Azure DevOps Organizations Section */}
              {showADOSection && (
                <div className="mb-6">
                  <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--fgColor-default)' }}>
                    Azure DevOps Organizations
                  </h2>
                  <div className="mb-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
                    {adoOrgs.length > 0 ? (
                      showAllSources ? (
                        // All Sources view: show org count only
                        <>
                          Showing {aggregatedADOOrgs.length} {aggregatedADOOrgs.length === 1 ? 'organization' : 'organizations'} with {adoOrgs.reduce((sum, org) => sum + org.total_repos, 0)} total repositories
                        </>
                      ) : (
                        // Specific source view: show org + project count
                        <>
                          Showing {Object.keys(groupedADOOrgs).length} {Object.keys(groupedADOOrgs).length === 1 ? 'organization' : 'organizations'} with {adoOrgs.length} {adoOrgs.length === 1 ? 'project' : 'projects'} and {adoOrgs.reduce((sum, org) => sum + org.total_repos, 0)} total repositories
                        </>
                      )
                    ) : (
                      'No Azure DevOps organizations found'
                    )}
                  </div>
                  {adoOrgs.length === 0 ? (
                    <Flash>No Azure DevOps organizations discovered yet. Start discovery to find repositories.</Flash>
                  ) : showAllSources ? (
                    // All Sources view: show simple org cards (like GitHub orgs)
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                      {aggregatedADOOrgs.map((org) => (
                        <GitHubOrganizationCard 
                          key={org.organization} 
                          organization={org}
                          linkParam="ado_organization"
                        />
                      ))}
                    </div>
                  ) : (
                    // Specific source view: show detailed project breakdown
                    <div className="space-y-6">
                      {Object.entries(groupedADOOrgs).map(([adoOrgName, projects]) => (
                        <ADOOrganizationCard 
                          key={adoOrgName} 
                          adoOrgName={adoOrgName} 
                          projects={projects} 
                        />
                      ))}
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </>
      )}

      {/* Discovery Modal - reuse existing modal from original Dashboard */}
      <DiscoveryModal
        isOpen={showDiscoveryModal}
        sourceType={sourceType}
        discoveryType={discoveryType ?? defaultDiscoveryType}
        setDiscoveryType={setDiscoveryType}
        organization={organization}
        setOrganization={setOrganization}
        enterpriseSlug={enterpriseSlug}
        setEnterpriseSlug={setEnterpriseSlug}
        adoOrganization={adoOrganization}
        setAdoOrganization={setAdoOrganization}
        adoProject={adoProject}
        setAdoProject={setAdoProject}
        loading={startDiscoveryMutation.isPending || startADODiscoveryMutation.isPending}
        error={discoveryError}
        onStart={handleStartDiscovery}
        onClose={() => {
          setShowDiscoveryModal(false);
          setDiscoveryType(null);
          setDiscoveryError(null);
          setOrganization('');
          setEnterpriseSlug('');
          setAdoOrganization('');
          setAdoProject('');
        }}
      />
    </div>
  );
}
