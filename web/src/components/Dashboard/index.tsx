import { useState } from 'react';
import { Link } from 'react-router-dom';
import type { Organization } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { useOrganizations } from '../../hooks/useQueries';
import { useStartDiscovery } from '../../hooks/useMutations';

export function Dashboard() {
  const { data: organizations = [], isLoading, isFetching } = useOrganizations();
  const startDiscoveryMutation = useStartDiscovery();
  
  const [searchTerm, setSearchTerm] = useState('');
  const [showDiscoveryModal, setShowDiscoveryModal] = useState(false);
  const [discoveryType, setDiscoveryType] = useState<'organization' | 'enterprise'>('organization');
  const [organization, setOrganization] = useState('');
  const [enterpriseSlug, setEnterpriseSlug] = useState('');
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);
  const [discoverySuccess, setDiscoverySuccess] = useState<string | null>(null);

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

    setDiscoveryError(null);
    setDiscoverySuccess(null);

    try {
      if (discoveryType === 'enterprise') {
        await startDiscoveryMutation.mutateAsync({ enterprise_slug: enterpriseSlug.trim() });
        setDiscoverySuccess(`Enterprise discovery started for ${enterpriseSlug}`);
        setEnterpriseSlug('');
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

  return (
    <div className="relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold text-gh-text-primary">Organizations</h1>
        <div className="flex gap-3">
          <button
            onClick={() => setShowDiscoveryModal(true)}
            className="px-4 py-1.5 bg-gh-success text-white text-sm font-medium rounded-md hover:bg-gh-success-hover transition-colors"
          >
            Start Discovery
          </button>
          <input
            type="text"
            placeholder="Search organizations..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-3 py-1.5 text-sm border border-gh-border-default rounded-md"
          />
        </div>
      </div>

      {discoverySuccess && (
        <div className="mb-4 bg-gh-success-bg border border-gh-success text-gh-success px-4 py-3 rounded-md text-sm">
          {discoverySuccess}
        </div>
      )}

      <div className="mb-4 text-sm text-gh-text-secondary">
        Showing {filteredOrgs.length} organizations with {totalRepos} total repositories
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : filteredOrgs.length === 0 ? (
        <div className="text-center py-12 text-gh-text-secondary">
          No organizations found
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredOrgs.map((org) => (
            <OrganizationCard key={org.organization} organization={org} />
          ))}
        </div>
      )}

      {showDiscoveryModal && (
        <DiscoveryModal
          discoveryType={discoveryType}
          setDiscoveryType={setDiscoveryType}
          organization={organization}
          setOrganization={setOrganization}
          enterpriseSlug={enterpriseSlug}
          setEnterpriseSlug={setEnterpriseSlug}
          loading={startDiscoveryMutation.isPending}
          error={discoveryError}
          onStart={handleStartDiscovery}
          onClose={() => {
            setShowDiscoveryModal(false);
            setDiscoveryError(null);
            setOrganization('');
            setEnterpriseSlug('');
          }}
        />
      )}
    </div>
  );
}

function OrganizationCard({ organization }: { organization: Organization }) {
  const getStatusColor = (status: string) => {
    // Map all backend statuses to GitHub color scheme
    const colors: Record<string, string> = {
      // Pending
      pending: 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default',
      
      // In Progress (blue)
      dry_run_queued: 'bg-gh-blue text-white',
      dry_run_in_progress: 'bg-gh-blue text-white',
      pre_migration: 'bg-gh-blue text-white',
      archive_generating: 'bg-gh-blue text-white',
      queued_for_migration: 'bg-gh-blue text-white',
      migrating_content: 'bg-gh-blue text-white',
      post_migration: 'bg-gh-blue text-white',
      
      // Complete (green)
      dry_run_complete: 'bg-gh-success text-white',
      migration_complete: 'bg-gh-success text-white',
      complete: 'bg-gh-success text-white',
      
      // Failed (red)
      dry_run_failed: 'bg-gh-danger text-white',
      migration_failed: 'bg-gh-danger text-white',
      
      // Rolled Back (yellow/orange)
      rolled_back: 'bg-gh-warning text-white',
    };
    return colors[status] || 'bg-gh-neutral-bg text-gh-text-secondary border border-gh-border-default';
  };

  const totalRepos = organization.total_repos;
  const statusCounts = organization.status_counts;

  return (
    <Link
      to={`/org/${encodeURIComponent(organization.organization)}`}
      className="bg-white rounded-lg border border-gh-border-default hover:border-gh-border-hover transition-colors p-6 block shadow-gh-card"
    >
      <h3 className="text-lg font-semibold text-gh-text-primary mb-4">
        {organization.organization}
      </h3>
      
      <div className="mb-4">
        <div className="text-3xl font-semibold text-gh-blue mb-1">{totalRepos}</div>
        <div className="text-sm text-gh-text-secondary">Total Repositories</div>
      </div>

      <div className="space-y-2">
        <div className="text-xs font-semibold text-gh-text-secondary mb-2 uppercase tracking-wide">Status Breakdown</div>
        <div className="flex flex-wrap gap-2">
          {Object.entries(statusCounts).map(([status, count]) => (
            <span
              key={status}
              className={`px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(status)}`}
            >
              {status.replace(/_/g, ' ')}: {count}
            </span>
          ))}
        </div>
      </div>

      <div className="mt-4 text-sm text-gh-blue hover:underline font-medium">
        View repositories →
      </div>
    </Link>
  );
}

interface DiscoveryModalProps {
  discoveryType: 'organization' | 'enterprise';
  setDiscoveryType: (type: 'organization' | 'enterprise') => void;
  organization: string;
  setOrganization: (org: string) => void;
  enterpriseSlug: string;
  setEnterpriseSlug: (slug: string) => void;
  loading: boolean;
  error: string | null;
  onStart: () => void;
  onClose: () => void;
}

function DiscoveryModal({ 
  discoveryType,
  setDiscoveryType,
  organization, 
  setOrganization,
  enterpriseSlug,
  setEnterpriseSlug,
  loading, 
  error, 
  onStart, 
  onClose 
}: DiscoveryModalProps) {
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onStart();
  };

  const isFormValid = discoveryType === 'organization' ? organization.trim() : enterpriseSlug.trim();

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 border border-gh-border-default">
        <div className="flex justify-between items-center p-4 border-b border-gh-border-default">
          <h2 className="text-base font-semibold text-gh-text-primary">Start Repository Discovery</h2>
          <button
            onClick={onClose}
            disabled={loading}
            className="text-gh-text-secondary hover:text-gh-text-primary transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        
        <form onSubmit={handleSubmit} className="p-4">
          {/* Discovery Type Selector */}
          <div className="mb-4">
            <label className="block text-sm font-semibold text-gh-text-primary mb-2">
              Discovery Type
            </label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setDiscoveryType('organization')}
                disabled={loading}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                  discoveryType === 'organization'
                    ? 'bg-gh-blue text-white'
                    : 'bg-gh-neutral-bg text-gh-text-primary hover:bg-gh-canvas-inset border border-gh-border-default'
                } disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                Organization
              </button>
              <button
                type="button"
                onClick={() => setDiscoveryType('enterprise')}
                disabled={loading}
                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                  discoveryType === 'enterprise'
                    ? 'bg-gh-blue text-white'
                    : 'bg-gh-neutral-bg text-gh-text-primary hover:bg-gh-canvas-inset border border-gh-border-default'
                } disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                Enterprise
              </button>
            </div>
          </div>

          {/* Organization Input */}
          {discoveryType === 'organization' && (
            <div className="mb-4">
              <label htmlFor="organization" className="block text-sm font-semibold text-gh-text-primary mb-2">
                Organization Name
              </label>
              <input
                id="organization"
                type="text"
                value={organization}
                onChange={(e) => setOrganization(e.target.value)}
                placeholder="e.g., your-github-org"
                disabled={loading}
                className="w-full px-3 py-1.5 text-sm border border-gh-border-default rounded-md disabled:opacity-60 disabled:cursor-not-allowed"
                autoFocus
              />
              <p className="mt-2 text-xs text-gh-text-secondary">
                Enter the GitHub organization name to discover all repositories.
              </p>
            </div>
          )}

          {/* Enterprise Input */}
          {discoveryType === 'enterprise' && (
            <div className="mb-4">
              <label htmlFor="enterprise" className="block text-sm font-semibold text-gh-text-primary mb-2">
                Enterprise Slug
              </label>
              <input
                id="enterprise"
                type="text"
                value={enterpriseSlug}
                onChange={(e) => setEnterpriseSlug(e.target.value)}
                placeholder="e.g., your-enterprise-slug"
                disabled={loading}
                className="w-full px-3 py-1.5 text-sm border border-gh-border-default rounded-md disabled:opacity-60 disabled:cursor-not-allowed"
                autoFocus
              />
              <p className="mt-2 text-xs text-gh-text-secondary">
                Enter the GitHub Enterprise slug to discover repositories across all organizations.
              </p>
            </div>
          )}

          {error && (
            <div className="mb-4 bg-gh-danger-bg border border-gh-danger text-gh-danger px-3 py-2 rounded-md text-xs">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="px-3 py-1.5 text-sm border border-gh-border-default text-gh-text-primary rounded-md hover:bg-gh-neutral-bg disabled:bg-gh-neutral-bg disabled:cursor-not-allowed transition-colors font-medium"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !isFormValid}
              className="px-3 py-1.5 text-sm bg-gh-success text-white rounded-md hover:bg-gh-success-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2 font-medium"
            >
              {loading ? (
                <>
                  <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Starting...
                </>
              ) : (
                'Start Discovery'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

