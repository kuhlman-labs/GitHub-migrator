import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../services/api';
import type { Organization } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';

export function Dashboard() {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showDiscoveryModal, setShowDiscoveryModal] = useState(false);
  const [discoveryType, setDiscoveryType] = useState<'organization' | 'enterprise'>('organization');
  const [organization, setOrganization] = useState('');
  const [enterpriseSlug, setEnterpriseSlug] = useState('');
  const [discoveryLoading, setDiscoveryLoading] = useState(false);
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);
  const [discoverySuccess, setDiscoverySuccess] = useState<string | null>(null);

  useEffect(() => {
    loadOrganizations();
    // Poll for updates every 10 seconds
    const interval = setInterval(loadOrganizations, 10000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadOrganizations = async () => {
    setLoading(true);
    try {
      const data = await api.listOrganizations();
      setOrganizations(data);
    } catch (error) {
      console.error('Failed to load organizations:', error);
    } finally {
      setLoading(false);
    }
  };

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

    setDiscoveryLoading(true);
    setDiscoveryError(null);
    setDiscoverySuccess(null);

    try {
      if (discoveryType === 'enterprise') {
        await api.startDiscovery({ enterprise_slug: enterpriseSlug.trim() });
        setDiscoverySuccess(`Enterprise discovery started for ${enterpriseSlug}`);
        setEnterpriseSlug('');
      } else {
        await api.startDiscovery({ organization: organization.trim() });
        setDiscoverySuccess(`Discovery started for ${organization}`);
        setOrganization('');
      }
      
      setShowDiscoveryModal(false);
      
      // Reload organizations after a short delay
      setTimeout(() => {
        loadOrganizations();
        setDiscoverySuccess(null);
      }, 2000);
    } catch (error) {
      setDiscoveryError(error instanceof Error ? error.message : 'Failed to start discovery');
    } finally {
      setDiscoveryLoading(false);
    }
  };

  const filteredOrgs = organizations.filter(org =>
    org.organization.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const totalRepos = organizations.reduce((sum, org) => sum + org.total_repos, 0);

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-light text-gray-900">Organizations</h1>
        <div className="flex gap-4">
          <button
            onClick={() => setShowDiscoveryModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
          >
            Start Discovery
          </button>
          <input
            type="text"
            placeholder="Search organizations..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
      </div>

      {discoverySuccess && (
        <div className="mb-4 bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg">
          {discoverySuccess}
        </div>
      )}

      <div className="mb-4 text-sm text-gray-600">
        Showing {filteredOrgs.length} organizations with {totalRepos} total repositories
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : filteredOrgs.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
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
          loading={discoveryLoading}
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
    const colors: Record<string, string> = {
      complete: 'bg-green-100 text-green-800',
      migration_complete: 'bg-green-100 text-green-800',
      pending: 'bg-gray-100 text-gray-800',
      in_progress: 'bg-blue-100 text-blue-800',
      failed: 'bg-red-100 text-red-800',
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
  };

  const totalRepos = organization.total_repos;
  const statusCounts = organization.status_counts;

  return (
    <Link
      to={`/org/${encodeURIComponent(organization.organization)}`}
      className="bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow p-6 block"
    >
      <h3 className="text-xl font-medium text-gray-900 mb-4">
        {organization.organization}
      </h3>
      
      <div className="mb-4">
        <div className="text-3xl font-light text-blue-600 mb-1">{totalRepos}</div>
        <div className="text-sm text-gray-600">Total Repositories</div>
      </div>

      <div className="space-y-2">
        <div className="text-sm font-medium text-gray-700 mb-2">Status Breakdown:</div>
        <div className="flex flex-wrap gap-2">
          {Object.entries(statusCounts).map(([status, count]) => (
            <span
              key={status}
              className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(status)}`}
            >
              {status.replace(/_/g, ' ')}: {count}
            </span>
          ))}
        </div>
      </div>

      <div className="mt-4 text-sm text-blue-600 hover:underline">
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
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="flex justify-between items-center p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">Start Repository Discovery</h2>
          <button
            onClick={onClose}
            disabled={loading}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        
        <form onSubmit={handleSubmit} className="p-6">
          {/* Discovery Type Selector */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Discovery Type
            </label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setDiscoveryType('organization')}
                disabled={loading}
                className={`flex-1 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  discoveryType === 'organization'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                } disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                Organization
              </button>
              <button
                type="button"
                onClick={() => setDiscoveryType('enterprise')}
                disabled={loading}
                className={`flex-1 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  discoveryType === 'enterprise'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                } disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                Enterprise
              </button>
            </div>
          </div>

          {/* Organization Input */}
          {discoveryType === 'organization' && (
            <div className="mb-4">
              <label htmlFor="organization" className="block text-sm font-medium text-gray-700 mb-2">
                Organization Name
              </label>
              <input
                id="organization"
                type="text"
                value={organization}
                onChange={(e) => setOrganization(e.target.value)}
                placeholder="e.g., your-github-org"
                disabled={loading}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed"
                autoFocus
              />
              <p className="mt-2 text-sm text-gray-500">
                Enter the GitHub organization name to discover all repositories.
              </p>
            </div>
          )}

          {/* Enterprise Input */}
          {discoveryType === 'enterprise' && (
            <div className="mb-4">
              <label htmlFor="enterprise" className="block text-sm font-medium text-gray-700 mb-2">
                Enterprise Slug
              </label>
              <input
                id="enterprise"
                type="text"
                value={enterpriseSlug}
                onChange={(e) => setEnterpriseSlug(e.target.value)}
                placeholder="e.g., your-enterprise-slug"
                disabled={loading}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed"
                autoFocus
              />
              <p className="mt-2 text-sm text-gray-500">
                Enter the GitHub Enterprise slug to discover repositories across all organizations.
              </p>
            </div>
          )}

          {error && (
            <div className="mb-4 bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:bg-gray-100 disabled:cursor-not-allowed transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !isFormValid}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
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

