import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../services/api';
import type { Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { formatBytes } from '../../utils/format';

export function Dashboard() {
  const [repositories, setRepositories] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [showDiscoveryModal, setShowDiscoveryModal] = useState(false);
  const [discoveryType, setDiscoveryType] = useState<'organization' | 'enterprise'>('organization');
  const [organization, setOrganization] = useState('');
  const [enterpriseSlug, setEnterpriseSlug] = useState('');
  const [discoveryLoading, setDiscoveryLoading] = useState(false);
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);
  const [discoverySuccess, setDiscoverySuccess] = useState<string | null>(null);

  useEffect(() => {
    loadRepositories();
    // Poll for updates every 10 seconds
    const interval = setInterval(loadRepositories, 10000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter]);

  const loadRepositories = async () => {
    setLoading(true);
    try {
      const data = await api.listRepositories({ 
        status: filter === 'all' ? undefined : filter 
      });
      setRepositories(data);
    } catch (error) {
      console.error('Failed to load repositories:', error);
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
      
      // Reload repositories after a short delay
      setTimeout(() => {
        loadRepositories();
        setDiscoverySuccess(null);
      }, 2000);
    } catch (error) {
      setDiscoveryError(error instanceof Error ? error.message : 'Failed to start discovery');
    } finally {
      setDiscoveryLoading(false);
    }
  };

  const filteredRepos = repositories.filter(repo =>
    repo.full_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-light text-gray-900">Repository Dashboard</h1>
        <div className="flex gap-4">
          <button
            onClick={() => setShowDiscoveryModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
          >
            Start Discovery
          </button>
          <input
            type="text"
            placeholder="Search repositories..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <StatusFilter value={filter} onChange={setFilter} />
        </div>
      </div>

      {discoverySuccess && (
        <div className="mb-4 bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg">
          {discoverySuccess}
        </div>
      )}

      <div className="mb-4 text-sm text-gray-600">
        Showing {filteredRepos.length} of {repositories.length} repositories
      </div>

      {loading ? (
        <LoadingSpinner />
      ) : (
        <RepositoryGrid repositories={filteredRepos} />
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

function StatusFilter({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const statuses = ['all', 'pending', 'in_progress', 'migration_complete', 'complete', 'failed'];
  
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
    >
      {statuses.map((status) => (
        <option key={status} value={status}>
          {status === 'all' ? 'All Status' : status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ')}
        </option>
      ))}
    </select>
  );
}

function RepositoryGrid({ repositories }: { repositories: Repository[] }) {
  if (repositories.length === 0) {
    return (
      <div className="text-center py-12 text-gray-500">
        No repositories found
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {repositories.map((repo) => (
        <RepositoryCard key={repo.id} repository={repo} />
      ))}
    </div>
  );
}

function RepositoryCard({ repository }: { repository: Repository }) {
  return (
    <Link
      to={`/repository/${encodeURIComponent(repository.full_name)}`}
      className="bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow p-6 block"
    >
      <h3 className="text-lg font-medium text-gray-900 mb-2 truncate">
        {repository.full_name}
      </h3>
      <div className="mb-4">
        <StatusBadge status={repository.status} />
      </div>
      <div className="space-y-2 text-sm text-gray-600">
        <div>Size: {formatBytes(repository.total_size)}</div>
        <div>Branches: {repository.branch_count}</div>
        <div className="flex gap-2 flex-wrap mt-2">
          {repository.has_lfs && <Badge color="blue">LFS</Badge>}
          {repository.has_submodules && <Badge color="purple">Submodules</Badge>}
          {repository.has_actions && <Badge color="green">Actions</Badge>}
          {repository.has_wiki && <Badge color="yellow">Wiki</Badge>}
        </div>
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

