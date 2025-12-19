import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Button, TextInput, Flash, FormControl } from '@primer/react';
import { Blankslate } from '@primer/react/experimental';
import { XIcon, RepoIcon } from '@primer/octicons-react';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Pagination } from '../common/Pagination';
import { useOrganizations, useAnalytics, useBatches, useDashboardActionItems, useDiscoveryProgress } from '../../hooks/useQueries';
import { useStartDiscovery, useStartADODiscovery } from '../../hooks/useMutations';
import { api } from '../../services/api';
import { KPISection } from './KPISection';
import { ActionItemsPanel } from './ActionItemsPanel';
// import { OrganizationProgressCard } from './OrganizationProgressCard';
import { GitHubOrganizationCard } from './GitHubOrganizationCard';
import { ADOOrganizationCard } from './ADOOrganizationCard';
import { UpcomingBatchesTimeline } from './UpcomingBatchesTimeline';
import { DiscoveryProgressCard, LastDiscoveryIndicator } from './DiscoveryProgressCard';

export function Dashboard() {
  // Fetch all dashboard data with polling
  const { data: organizations = [], isLoading: orgsLoading, isFetching: orgsFetching, refetch: refetchOrgs } = useOrganizations();
  const { data: analytics, isLoading: analyticsLoading, isFetching: analyticsFetching, refetch: refetchAnalytics } = useAnalytics();
  const { data: batches = [], isLoading: batchesLoading, isFetching: batchesFetching, refetch: refetchBatches } = useBatches();
  const { data: actionItems, isLoading: actionItemsLoading, isFetching: actionItemsFetching, refetch: refetchActionItems } = useDashboardActionItems();
  const { data: discoveryProgress } = useDiscoveryProgress();
  
  const startDiscoveryMutation = useStartDiscovery();
  const startADODiscoveryMutation = useStartADODiscovery();
  const [searchParams] = useSearchParams();
  
  const searchTerm = searchParams.get('search') || '';
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 12;
  const [showDiscoveryModal, setShowDiscoveryModal] = useState(false);
  const [sourceType, setSourceType] = useState<'github' | 'azuredevops'>('github');
  const [discoveryType, setDiscoveryType] = useState<'organization' | 'enterprise' | 'ado-org' | 'ado-project'>('organization');
  const [organization, setOrganization] = useState('');
  const [enterpriseSlug, setEnterpriseSlug] = useState('');
  const [adoOrganization, setAdoOrganization] = useState('');
  const [adoProject, setAdoProject] = useState('');
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);
  const [discoverySuccess, setDiscoverySuccess] = useState<string | null>(null);
  const [discoveryBannerDismissed, setDiscoveryBannerDismissed] = useState(false);

  // Persist dismissed state in localStorage, keyed by discovery ID
  const dismissedDiscoveryKey = 'dismissedDiscoveryId';
  const currentDiscoveryId = discoveryProgress?.id;

  // Sync dismissed state with localStorage when discovery data loads or changes
  useEffect(() => {
    if (!currentDiscoveryId) return;
    
    if (discoveryProgress?.status === 'in_progress') {
      // New discovery in progress - clear any previous dismissal
      localStorage.removeItem(dismissedDiscoveryKey);
      setDiscoveryBannerDismissed(false);
    } else {
      // Check if this completed discovery was previously dismissed
      const dismissedId = localStorage.getItem(dismissedDiscoveryKey);
      setDiscoveryBannerDismissed(dismissedId === String(currentDiscoveryId));
    }
  }, [discoveryProgress?.status, currentDiscoveryId]);

  const handleDismissDiscoveryBanner = () => {
    if (currentDiscoveryId) {
      localStorage.setItem(dismissedDiscoveryKey, String(currentDiscoveryId));
    }
    setDiscoveryBannerDismissed(true);
  };

  const handleExpandDiscoveryBanner = () => {
    localStorage.removeItem(dismissedDiscoveryKey);
    setDiscoveryBannerDismissed(false);
  };

  // Fetch source type on mount
  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const config = await api.getConfig();
        setSourceType(config.source_type);
        // Set default discovery type based on source
        if (config.source_type === 'azuredevops') {
          setDiscoveryType('ado-org');
        }
      } catch (error) {
        console.error('Failed to fetch config:', error);
      }
    };
    fetchConfig();
  }, []);

  // Polling strategy based on activity
  useEffect(() => {
    const hasActiveMigrations = analytics && analytics.in_progress_count > 0;

    let analyticsInterval: NodeJS.Timeout | null = null;
    let actionItemsInterval: NodeJS.Timeout | null = null;
    let orgsInterval: NodeJS.Timeout | null = null;
    let batchesInterval: NodeJS.Timeout | null = null;

    // KPIs: Poll every 30s if migrations active, else 2min
    const analyticsDelay = hasActiveMigrations ? 30000 : 120000;
    analyticsInterval = setInterval(() => {
      refetchAnalytics();
    }, analyticsDelay);

    // Action Items: Poll every 15s (critical for admin attention)
    actionItemsInterval = setInterval(() => {
      refetchActionItems();
    }, 15000);

    // Org Progress: Poll every 1min
    orgsInterval = setInterval(() => {
      refetchOrgs();
    }, 60000);

    // Upcoming Batches: Poll every 1min if any in-progress, else 5min
    const batchesDelay = hasActiveMigrations ? 60000 : 300000;
    batchesInterval = setInterval(() => {
      refetchBatches();
    }, batchesDelay);

    return () => {
      if (analyticsInterval) clearInterval(analyticsInterval);
      if (actionItemsInterval) clearInterval(actionItemsInterval);
      if (orgsInterval) clearInterval(orgsInterval);
      if (batchesInterval) clearInterval(batchesInterval);
    };
    // Only depend on in_progress_count to avoid recreating intervals on data changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [analytics?.in_progress_count]);

  const handleStartDiscovery = async () => {
    // Validate input based on discovery type
    if (discoveryType === 'organization' && !organization.trim()) {
      setDiscoveryError('Organization name is required');
      return;
    }
    
    if (discoveryType === 'enterprise' && !enterpriseSlug.trim()) {
      setDiscoveryError('Enterprise slug is required');
      return;
    }

    if (discoveryType === 'ado-org' && !adoOrganization.trim()) {
      setDiscoveryError('Azure DevOps organization name is required');
      return;
    }

    if (discoveryType === 'ado-project' && (!adoOrganization.trim() || !adoProject.trim())) {
      setDiscoveryError('Both Azure DevOps organization and project names are required');
      return;
    }

    setDiscoveryError(null);
    setDiscoverySuccess(null);

    try {
      if (discoveryType === 'enterprise') {
        await startDiscoveryMutation.mutateAsync({ enterprise_slug: enterpriseSlug.trim() });
        setDiscoverySuccess(`Enterprise discovery started for ${enterpriseSlug}`);
        setEnterpriseSlug('');
      } else if (discoveryType === 'ado-org') {
        await startADODiscoveryMutation.mutateAsync({ organization: adoOrganization.trim() });
        setDiscoverySuccess(`ADO organization discovery started for ${adoOrganization}`);
        setAdoOrganization('');
      } else if (discoveryType === 'ado-project') {
        await startADODiscoveryMutation.mutateAsync({ 
          organization: adoOrganization.trim(), 
          project: adoProject.trim() 
        });
        setDiscoverySuccess(`ADO project discovery started for ${adoOrganization}/${adoProject}`);
        setAdoOrganization('');
        setAdoProject('');
      } else {
        await startDiscoveryMutation.mutateAsync({ organization: organization.trim() });
        setDiscoverySuccess(`Discovery started for ${organization}`);
        setOrganization('');
      }
      
      setShowDiscoveryModal(false);
      
      // Clear success message after 2 seconds
      setTimeout(() => {
        setDiscoverySuccess(null);
      }, 2000);
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

  // Reset page when search changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchTerm]);

  const isLoading = orgsLoading || analyticsLoading || batchesLoading || actionItemsLoading;
  const isFetching = orgsFetching || analyticsFetching || batchesFetching || actionItemsFetching;

  // Group ADO projects by organization
  const groupedADOOrgs = sourceType === 'azuredevops' 
    ? filteredOrgs.reduce((acc, org) => {
        const adoOrgName = org.ado_organization || 'Unknown';
        if (!acc[adoOrgName]) {
          acc[adoOrgName] = [];
        }
        acc[adoOrgName].push(org);
        return acc;
      }, {} as Record<string, typeof filteredOrgs>)
    : {};

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
          <Button
            variant="primary"
            onClick={() => setShowDiscoveryModal(true)}
          >
            Start Discovery
          </Button>
        </div>
      </div>

      {discoverySuccess && (
        <Flash variant="success" className="mb-3">
          {discoverySuccess}
        </Flash>
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

      {/* Organizations Section */}
      <div className="mb-6">
        <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--fgColor-default)' }}>
          {sourceType === 'azuredevops' ? 'Azure DevOps Organizations' : 'GitHub Organizations'}
        </h2>
        <div className="mb-4 text-sm" style={{ color: 'var(--fgColor-muted)' }}>
          {sourceType === 'azuredevops' ? (
            totalItems > 0 ? (
              <>
                Showing {Object.keys(groupedADOOrgs).length} {Object.keys(groupedADOOrgs).length === 1 ? 'organization' : 'organizations'} with {totalItems} {totalItems === 1 ? 'project' : 'projects'} and {totalRepos} total repositories
              </>
            ) : (
              'No organizations found'
            )
          ) : (
            totalItems > 0 ? (
              <>
                Showing {startIndex + 1}-{Math.min(endIndex, totalItems)} of {totalItems} {totalItems === 1 ? 'organization' : 'organizations'} with {totalRepos} total repositories
              </>
            ) : (
              'No organizations found'
            )
          )}
        </div>

        {orgsLoading ? (
          <LoadingSpinner />
        ) : filteredOrgs.length === 0 ? (
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
              <Blankslate.PrimaryAction onClick={() => setShowDiscoveryModal(true)}>
                Start Discovery
              </Blankslate.PrimaryAction>
            )}
          </Blankslate>
        ) : sourceType === 'azuredevops' ? (
          // Azure DevOps: Show organizations containing projects
          <div className="space-y-6">
            {Object.entries(groupedADOOrgs).map(([adoOrgName, projects]) => (
              <ADOOrganizationCard 
                key={adoOrgName} 
                adoOrgName={adoOrgName} 
                projects={projects} 
              />
            ))}
          </div>
        ) : (
          // GitHub: Show organizations with pagination
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
              {paginatedOrgs.map((org) => (
                <GitHubOrganizationCard key={org.organization} organization={org} />
              ))}
            </div>
            {totalItems > pageSize && (
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

      {/* Discovery Modal - reuse existing modal from original Dashboard */}
      <DiscoveryModal
        isOpen={showDiscoveryModal}
        sourceType={sourceType}
        discoveryType={discoveryType}
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

interface DiscoveryModalProps {
  isOpen: boolean;
  sourceType: 'github' | 'azuredevops';
  discoveryType: 'organization' | 'enterprise' | 'ado-org' | 'ado-project';
  setDiscoveryType: (type: 'organization' | 'enterprise' | 'ado-org' | 'ado-project') => void;
  organization: string;
  setOrganization: (org: string) => void;
  enterpriseSlug: string;
  setEnterpriseSlug: (slug: string) => void;
  adoOrganization: string;
  setAdoOrganization: (org: string) => void;
  adoProject: string;
  setAdoProject: (project: string) => void;
  loading: boolean;
  error: string | null;
  onStart: () => void;
  onClose: () => void;
}

function DiscoveryModal({ 
  isOpen,
  sourceType,
  discoveryType,
  setDiscoveryType,
  organization, 
  setOrganization,
  enterpriseSlug,
  setEnterpriseSlug,
  adoOrganization,
  setAdoOrganization,
  adoProject,
  setAdoProject,
  loading,
  error,
  onStart, 
  onClose 
}: DiscoveryModalProps) {
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onStart();
  };

  const isFormValid = 
    (discoveryType === 'organization' && organization.trim()) ||
    (discoveryType === 'enterprise' && enterpriseSlug.trim()) ||
    (discoveryType === 'ado-org' && adoOrganization.trim()) ||
    (discoveryType === 'ado-project' && adoOrganization.trim() && adoProject.trim());

  if (!isOpen) return null;
  
  return (
    <>
      {/* Backdrop overlay */}
      <div 
        className="fixed inset-0 bg-black/50 z-40"
        onClick={onClose}
        aria-hidden="true"
      />
      
      {/* Modal */}
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>
        <div className="rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-auto" style={{ backgroundColor: 'var(--bgColor-default)' }}>
          <div className="flex items-center justify-between p-4 border-b" style={{ borderColor: 'var(--borderColor-default)' }}>
            <h2 id="discovery-modal-title" className="text-xl font-semibold">
              Start Repository Discovery
            </h2>
            <button
              onClick={onClose}
              className="text-gh-text-secondary hover:text-gh-text-primary"
              aria-label="Close"
            >
              <XIcon size={20} />
            </button>
          </div>
      <form onSubmit={handleSubmit} className="p-4">
        {error && (
          <Flash variant="danger" className="mb-3">
            {error}
          </Flash>
        )}
        <FormControl className="mb-3">
          <FormControl.Label>Discovery Type</FormControl.Label>
          <div className="flex gap-2">
            {sourceType === 'github' ? (
              <>
                <Button
                  type="button"
                  variant={discoveryType === 'organization' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('organization')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Organization
                </Button>
                <Button
                  type="button"
                  variant={discoveryType === 'enterprise' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('enterprise')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Enterprise
                </Button>
              </>
            ) : (
              <>
                <Button
                  type="button"
                  variant={discoveryType === 'ado-org' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('ado-org')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Organization
                </Button>
                <Button
                  type="button"
                  variant={discoveryType === 'ado-project' ? 'primary' : 'default'}
                  onClick={() => setDiscoveryType('ado-project')}
                  disabled={loading}
                  style={{ flex: 1 }}
                >
                  Project
                </Button>
              </>
            )}
          </div>
        </FormControl>

        {discoveryType === 'organization' && (
          <FormControl className="mb-3" required>
            <FormControl.Label>Organization Name</FormControl.Label>
            <TextInput
              value={organization}
              onChange={(e) => setOrganization(e.target.value)}
              placeholder="e.g., your-github-org"
              disabled={loading}
              autoFocus
              required
            />
            <FormControl.Caption>
              Enter the GitHub organization name to discover all repositories.
            </FormControl.Caption>
          </FormControl>
        )}

        {discoveryType === 'enterprise' && (
          <FormControl className="mb-3" required>
            <FormControl.Label>Enterprise Slug</FormControl.Label>
            <TextInput
              value={enterpriseSlug}
              onChange={(e) => setEnterpriseSlug(e.target.value)}
              placeholder="e.g., your-enterprise-slug"
              disabled={loading}
              autoFocus
              required
            />
            <FormControl.Caption>
              Enter the GitHub Enterprise slug to discover repositories across all organizations.
            </FormControl.Caption>
          </FormControl>
        )}

        {discoveryType === 'ado-org' && (
          <FormControl className="mb-3" required>
            <FormControl.Label>Azure DevOps Organization</FormControl.Label>
            <TextInput
              value={adoOrganization}
              onChange={(e) => setAdoOrganization(e.target.value)}
              placeholder="e.g., your-ado-org"
              disabled={loading}
              autoFocus
              required
            />
            <FormControl.Caption>
              Discover all projects and repositories in this Azure DevOps organization.
            </FormControl.Caption>
          </FormControl>
        )}

        {discoveryType === 'ado-project' && (
          <div className="space-y-3 mb-3">
            <FormControl required>
              <FormControl.Label>Azure DevOps Organization</FormControl.Label>
              <TextInput
                value={adoOrganization}
                onChange={(e) => setAdoOrganization(e.target.value)}
                placeholder="e.g., your-ado-org"
                disabled={loading}
                required
              />
            </FormControl>
            <FormControl required>
              <FormControl.Label>Project Name</FormControl.Label>
              <TextInput
                value={adoProject}
                onChange={(e) => setAdoProject(e.target.value)}
                placeholder="e.g., your-project"
                disabled={loading}
                autoFocus
                required
              />
              <FormControl.Caption>
                Discover repositories in a specific Azure DevOps project.
              </FormControl.Caption>
            </FormControl>
          </div>
        )}

        <div className="flex justify-end gap-2 pt-4 border-t border-gh-border-default">
          <Button
            type="button"
            onClick={onClose}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            disabled={loading || !isFormValid}
          >
            {loading ? 'Starting...' : 'Start Discovery'}
          </Button>
        </div>
      </form>
        </div>
      </div>
    </>
  );
}
