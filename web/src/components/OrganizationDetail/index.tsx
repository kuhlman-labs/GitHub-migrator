import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import type { Repository } from '../../types';
import { LoadingSpinner } from '../common/LoadingSpinner';
import { RefreshIndicator } from '../common/RefreshIndicator';
import { StatusBadge } from '../common/StatusBadge';
import { Badge } from '../common/Badge';
import { formatBytes } from '../../utils/format';
import { useRepositories } from '../../hooks/useQueries';

type FeatureFilter = {
  key: keyof Repository;
  label: string;
  color: string;
};

const FEATURE_FILTERS: FeatureFilter[] = [
  { key: 'is_archived', label: 'Archived', color: 'gray' },
  { key: 'has_lfs', label: 'LFS', color: 'blue' },
  { key: 'has_submodules', label: 'Submodules', color: 'purple' },
  { key: 'has_large_files', label: 'Large Files (>100MB)', color: 'orange' },
  { key: 'has_actions', label: 'GitHub Actions', color: 'green' },
  { key: 'has_wiki', label: 'Wiki', color: 'yellow' },
  { key: 'has_pages', label: 'Pages', color: 'pink' },
  { key: 'has_discussions', label: 'Discussions', color: 'indigo' },
  { key: 'has_projects', label: 'Projects', color: 'teal' },
  { key: 'branch_protections', label: 'Branch Protections', color: 'red' },
];

// Map simplified filter values to actual backend statuses
const STATUS_MAP: Record<string, string[]> = {
  all: [],
  pending: ['pending'],
  in_progress: [
    'dry_run_queued',
    'dry_run_in_progress',
    'pre_migration',
    'archive_generating',
    'queued_for_migration',
    'migrating_content',
    'post_migration',
  ],
  complete: ['dry_run_complete', 'migration_complete', 'complete'],
  failed: ['dry_run_failed', 'migration_failed'],
  rolled_back: ['rolled_back'],
};

export function OrganizationDetail() {
  const { orgName } = useParams<{ orgName: string }>();
  const [filter, setFilter] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedFeatures, setSelectedFeatures] = useState<Set<keyof Repository>>(new Set());
  const [showFilters, setShowFilters] = useState(false);

  const { data, isLoading, isFetching } = useRepositories({});

  // Filter repositories for this organization (client-side)
  const repositories = (data?.repositories || []).filter((repo: Repository) => {
    const org = repo.full_name.split('/')[0];
    return org === orgName;
  });

  const toggleFeature = (feature: keyof Repository) => {
    const newSelected = new Set(selectedFeatures);
    if (newSelected.has(feature)) {
      newSelected.delete(feature);
    } else {
      newSelected.add(feature);
    }
    setSelectedFeatures(newSelected);
  };

  const clearAllFilters = () => {
    setSelectedFeatures(new Set());
    setFilter('all');
    setSearchTerm('');
  };

  const filteredRepos = repositories.filter((repo: Repository) => {
    // Status filter - check if repo status matches any of the mapped backend statuses
    if (filter !== 'all') {
      const allowedStatuses = STATUS_MAP[filter] || [];
      if (!allowedStatuses.includes(repo.status)) {
        return false;
      }
    }

    // Search filter
    if (searchTerm && !repo.full_name.toLowerCase().includes(searchTerm.toLowerCase())) {
      return false;
    }

    // Feature filters (AND logic - repo must have ALL selected features)
    if (selectedFeatures.size > 0) {
      for (const feature of selectedFeatures) {
        const value = repo[feature];
        const hasFeature = typeof value === 'boolean' ? value : (typeof value === 'number' && value > 0);
        if (!hasFeature) {
          return false;
        }
      }
    }

    return true;
  });

  const statuses = ['all', 'pending', 'in_progress', 'complete', 'failed', 'rolled_back'];
  const hasActiveFilters = selectedFeatures.size > 0 || filter !== 'all' || searchTerm !== '';

  return (
    <div className="max-w-7xl mx-auto relative">
      <RefreshIndicator isRefreshing={isFetching && !isLoading} />
      <div className="mb-6">
        <Link to="/" className="text-blue-600 hover:underline text-sm">
          ← Back to Organizations
        </Link>
      </div>

      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-light text-gray-900">{orgName}</h1>
        <div className="flex gap-4">
          <input
            type="text"
            placeholder="Search repositories..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            {statuses.map((status) => (
              <option key={status} value={status}>
                {status === 'all' ? 'All Status' : status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, ' ')}
              </option>
            ))}
          </select>
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`px-4 py-2 rounded-lg transition-colors ${
              selectedFeatures.size > 0
                ? 'bg-blue-600 text-white hover:bg-blue-700'
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
            }`}
          >
            <span className="flex items-center gap-2">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
              </svg>
              Features
              {selectedFeatures.size > 0 && (
                <span className="bg-white text-blue-600 rounded-full px-2 py-0.5 text-xs font-medium">
                  {selectedFeatures.size}
                </span>
              )}
            </span>
          </button>
          {hasActiveFilters && (
            <button
              onClick={clearAllFilters}
              className="px-4 py-2 text-gray-600 hover:text-gray-900 transition-colors"
            >
              Clear All
            </button>
          )}
        </div>
      </div>

      {/* Feature Filters Panel */}
      {showFilters && (
        <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Filter by Features</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
            {FEATURE_FILTERS.map((featureFilter) => {
              const count = repositories.filter(r => {
                const value = r[featureFilter.key];
                return typeof value === 'boolean' ? value : (typeof value === 'number' && value > 0);
              }).length;
              return (
                <label
                  key={featureFilter.key}
                  className={`flex items-center gap-3 p-3 rounded-lg border-2 cursor-pointer transition-all ${
                    selectedFeatures.has(featureFilter.key)
                      ? 'border-blue-500 bg-blue-50'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selectedFeatures.has(featureFilter.key)}
                    onChange={() => toggleFeature(featureFilter.key)}
                    className="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                  />
                  <div className="flex-1">
                    <div className="text-sm font-medium text-gray-900">{featureFilter.label}</div>
                    <div className="text-xs text-gray-500">{count} repos</div>
                  </div>
                </label>
              );
            })}
          </div>
        </div>
      )}

      <div className="mb-4 flex items-center justify-between">
        <div className="text-sm text-gray-600">
          Showing {filteredRepos.length} of {repositories.length} repositories
        </div>
        {selectedFeatures.size > 0 && (
          <div className="flex gap-2 flex-wrap">
            {Array.from(selectedFeatures).map((feature) => {
              const featureConfig = FEATURE_FILTERS.find(f => f.key === feature);
              return (
                <span
                  key={feature}
                  className="inline-flex items-center gap-1 px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm"
                >
                  {featureConfig?.label}
                  <button
                    onClick={() => toggleFeature(feature)}
                    className="hover:bg-blue-200 rounded-full p-0.5"
                  >
                    <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  </button>
                </span>
              );
            })}
          </div>
        )}
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : filteredRepos.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          No repositories found
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredRepos.map((repo) => (
            <RepositoryCard key={repo.id} repository={repo} />
          ))}
        </div>
      )}
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
          {repository.is_archived && <Badge color="gray">Archived</Badge>}
          {repository.has_lfs && <Badge color="blue">LFS</Badge>}
          {repository.has_submodules && <Badge color="purple">Submodules</Badge>}
          {repository.has_large_files && <Badge color="orange">Large Files</Badge>}
          {repository.has_actions && <Badge color="green">Actions</Badge>}
          {repository.has_wiki && <Badge color="yellow">Wiki</Badge>}
          {repository.has_pages && <Badge color="pink">Pages</Badge>}
          {repository.has_discussions && <Badge color="indigo">Discussions</Badge>}
          {repository.has_projects && <Badge color="teal">Projects</Badge>}
          {repository.branch_protections > 0 && <Badge color="red">Protected</Badge>}
        </div>
      </div>
    </Link>
  );
}

