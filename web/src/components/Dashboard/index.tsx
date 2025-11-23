import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Heading, Button, TextInput, Flash, Label, FormControl } from '@primer/react';
import { SearchIcon, XIcon } from '@primer/octicons-react';
import type { Organization } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { Pagination } from '../common/Pagination';
import { useOrganizations } from '../../hooks/useQueries';
import { useStartDiscovery, useStartADODiscovery } from '../../hooks/useMutations';
import { api } from '../../services/api';

export function Dashboard() {
  const { data: organizations = [], isLoading, isFetching } = useOrganizations();
  const startDiscoveryMutation = useStartDiscovery();
  const startADODiscoveryMutation = useStartADODiscovery();
  
  const [searchTerm, setSearchTerm] = useState('');
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

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      <div className="flex justify-between items-center mb-8">
        <Heading as="h1">Organizations</Heading>
        <div className="flex gap-3">
          <Button
            variant="primary"
            onClick={() => setShowDiscoveryModal(true)}
          >
            Start Discovery
          </Button>
          <TextInput
            leadingVisual={SearchIcon}
            placeholder="Search organizations..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            style={{ width: 250 }}
          />
        </div>
      </div>

      {discoverySuccess && (
        <Flash variant="success" className="mb-3">
          {discoverySuccess}
        </Flash>
      )}

      <div className="mb-4 text-sm text-gh-text-secondary">
        {totalItems > 0 ? (
          <>
            Showing {startIndex + 1}-{Math.min(endIndex, totalItems)} of {totalItems} {totalItems === 1 ? 'organization' : 'organizations'} with {totalRepos} total repositories
          </>
        ) : (
          'No organizations found'
        )}
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : filteredOrgs.length === 0 ? (
        <div className="text-center py-12 text-gh-text-secondary">
          No organizations found
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
            {paginatedOrgs.map((org) => (
              <OrganizationCard key={org.organization} organization={org} />
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

function OrganizationCard({ organization }: { organization: Organization }) {
  const getStatusVariant = (status: string): 'default' | 'primary' | 'secondary' | 'accent' | 'success' | 'attention' | 'severe' | 'danger' | 'done' | 'sponsors' => {
    const statusMap: Record<string, 'default' | 'primary' | 'secondary' | 'accent' | 'success' | 'attention' | 'severe' | 'danger' | 'done' | 'sponsors'> = {
      // Pending / Ready (neutral gray)
      pending: 'default',
      ready: 'default',
      
      // In Progress (blue)
      dry_run_queued: 'accent',
      dry_run_in_progress: 'accent',
      pre_migration: 'accent',
      archive_generating: 'accent',
      queued_for_migration: 'accent',
      migrating_content: 'accent',
      post_migration: 'accent',
      in_progress: 'accent',
      
      // Complete/Success (green)
      dry_run_complete: 'success',
      migration_complete: 'success',
      complete: 'success',
      completed: 'success',
      
      // Failures (red)
      dry_run_failed: 'danger',
      migration_failed: 'danger',
      failed: 'danger',
      
      // Warnings (yellow/orange)
      completed_with_errors: 'attention',
      rolled_back: 'attention',
      remediation_required: 'attention',
      
      // Cancelled (secondary/muted)
      cancelled: 'secondary',
      wont_migrate: 'secondary',
    };
    return statusMap[status] || 'default';
  };

  const totalRepos = organization.total_repos;
  const totalProjects = organization.total_projects;
  const statusCounts = organization.status_counts;

  return (
    <Link
      to={`/org/${encodeURIComponent(organization.organization)}`}
      className="block bg-white rounded-lg border border-gh-border-default hover:border-gh-border-hover transition-colors p-6 shadow-gh-card"
    >
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-gh-text-primary mb-2">
          {organization.organization}
        </h3>
        {organization.ado_organization && (
          <Label variant="accent" size="small">
            ADO Org: {organization.ado_organization}
          </Label>
        )}
        {organization.enterprise && (
          <Label variant="sponsors" size="small" className="ml-1">
            GitHub Enterprise: {organization.enterprise}
          </Label>
        )}
      </div>
      
      <div className="mb-4 space-y-3">
        <div>
          <div className="text-3xl font-semibold text-blue-600 mb-1">{totalRepos}</div>
          <div className="text-sm text-gh-text-secondary">Total Repositories</div>
        </div>
        
        {totalProjects !== undefined && (
          <div>
            <div className="text-2xl font-semibold text-gh-text-primary mb-1">{totalProjects}</div>
            <div className="text-sm text-gh-text-secondary">Total Projects</div>
          </div>
        )}
      </div>

      <div className="mb-3">
        <div className="text-xs font-semibold text-gh-text-secondary mb-2 uppercase tracking-wide">
          Status Breakdown
        </div>
        <div className="flex flex-wrap gap-1">
          {Object.entries(statusCounts).map(([status, count]) => (
            <Label key={status} variant={getStatusVariant(status)} size="small">
              {status.replace(/_/g, ' ')}: {count}
            </Label>
          ))}
        </div>
      </div>

      <div className="text-sm text-blue-600 hover:underline font-medium">
        View repositories →
      </div>
    </Link>
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
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-auto">
          <div className="flex items-center justify-between p-4 border-b border-gh-border-default">
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
          <FormControl className="mb-3">
            <FormControl.Label>Organization Name</FormControl.Label>
            <TextInput
              value={organization}
              onChange={(e) => setOrganization(e.target.value)}
              placeholder="e.g., your-github-org"
              disabled={loading}
              autoFocus
            />
            <FormControl.Caption>
              Enter the GitHub organization name to discover all repositories.
            </FormControl.Caption>
          </FormControl>
        )}

        {discoveryType === 'enterprise' && (
          <FormControl className="mb-3">
            <FormControl.Label>Enterprise Slug</FormControl.Label>
            <TextInput
              value={enterpriseSlug}
              onChange={(e) => setEnterpriseSlug(e.target.value)}
              placeholder="e.g., your-enterprise-slug"
              disabled={loading}
              autoFocus
            />
            <FormControl.Caption>
              Enter the GitHub Enterprise slug to discover repositories across all organizations.
            </FormControl.Caption>
          </FormControl>
        )}

        {discoveryType === 'ado-org' && (
          <FormControl className="mb-3">
            <FormControl.Label>Azure DevOps Organization</FormControl.Label>
            <TextInput
              value={adoOrganization}
              onChange={(e) => setAdoOrganization(e.target.value)}
              placeholder="e.g., your-ado-org"
              disabled={loading}
              autoFocus
            />
            <FormControl.Caption>
              Discover all projects and repositories in this Azure DevOps organization.
            </FormControl.Caption>
          </FormControl>
        )}

        {discoveryType === 'ado-project' && (
          <div className="space-y-3 mb-3">
            <FormControl>
              <FormControl.Label>Azure DevOps Organization</FormControl.Label>
              <TextInput
                value={adoOrganization}
                onChange={(e) => setAdoOrganization(e.target.value)}
                placeholder="e.g., your-ado-org"
                disabled={loading}
              />
            </FormControl>
            <FormControl>
              <FormControl.Label>Project Name</FormControl.Label>
              <TextInput
                value={adoProject}
                onChange={(e) => setAdoProject(e.target.value)}
                placeholder="e.g., your-project"
                disabled={loading}
                autoFocus
              />
              <FormControl.Caption>
                Discover repositories in a specific Azure DevOps project.
              </FormControl.Caption>
            </FormControl>
          </div>
        )}

        {error && (
          <Flash variant="danger" className="mb-3">
            {error}
          </Flash>
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
